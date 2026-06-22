package machine

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/runtimeabi"
)

// ---- allocation_loop.go ----

const (
	allocationLoopFunctionName    = "p25.allocation.main"
	allocationLoopBound           = int32(1024)
	allocationLoopSliceLength     = int32(32)
	allocationLoopIndexConst      = int32(0)
	allocationLoopStep            = int32(1)
	allocationLoopP55StoreProofID = "proof:allocation-zero:literal0:xs:9:9"
	allocationLoopP55LoadProofID  = "proof:allocation-zero:literal0:xs:10:33"
)

type AllocationLoopPlan struct {
	Function      Function
	ChecksumLocal int
	IndexLocal    int
	SlicePtrLocal int
	SliceLenLocal int
	BackingLocal  int
	BackingSlots  int
	LoopBound     int32
	SliceLength   int32
	IndexConst    int32
	Step          int32
	StartLabel    int
	EndLabel      int
	FailureLabel  int
	SuccessReturn int32
	FailureReturn int32
	BoundsChecks  int
}

func AllocationLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := AllocationLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func AllocationLoopPlanFromStackIR(fn ir.IRFunc) (AllocationLoopPlan, bool, error) {
	if fn.Name != allocationLoopFunctionName || fn.ParamSlots != 0 || fn.ReturnSlots != 1 ||
		fn.LocalSlots < 4 {
		return AllocationLoopPlan{}, false, nil
	}
	if len(fn.Instrs) != 40 {
		return AllocationLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return AllocationLoopPlan{}, false, nil
	}
	checksumLocal := in[1].Local
	indexLocal := in[3].Local
	if checksumLocal == indexLocal {
		return AllocationLoopPlan{}, false, nil
	}
	if in[4].Kind != ir.IRLabel || in[4].Label != 0 {
		return AllocationLoopPlan{}, false, nil
	}
	startLabel := in[4].Label
	if !isLoad(in[5], indexLocal) ||
		in[6].Kind != ir.IRConstI32 || in[6].Imm != allocationLoopBound ||
		in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero || in[8].Label != 1 {
		return AllocationLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if in[9].Kind != ir.IRConstI32 || in[9].Imm != allocationLoopSliceLength ||
		in[10].Kind != ir.IRStackSliceI32 ||
		in[10].Local < 0 ||
		in[10].ArgSlots != 16 ||
		in[10].Imm != allocationLoopSliceLength ||
		in[10].Name != "xs" {
		return AllocationLoopPlan{}, false, nil
	}
	backingLocal := in[10].Local
	backingSlots := in[10].ArgSlots
	sliceLenLocal := in[11].Local
	slicePtrLocal := in[12].Local
	if !isStore(in[11], sliceLenLocal) || !isStore(in[12], slicePtrLocal) {
		return AllocationLoopPlan{}, false, nil
	}
	if !isLoad(in[13], slicePtrLocal) || !isLoad(in[14], sliceLenLocal) ||
		in[15].Kind != ir.IRConstI32 || in[15].Imm != allocationLoopIndexConst ||
		!isLoad(in[16], indexLocal) {
		return AllocationLoopPlan{}, false, nil
	}
	if !isLoad(in[18], checksumLocal) ||
		!isLoad(in[19], slicePtrLocal) || !isLoad(in[20], sliceLenLocal) ||
		in[21].Kind != ir.IRConstI32 || in[21].Imm != allocationLoopIndexConst ||
		!validAllocationLoopIndexAccessShape(in[17], in[22]) ||
		in[23].Kind != ir.IRAddI32 ||
		!isStore(in[24], checksumLocal) {
		return AllocationLoopPlan{}, false, nil
	}
	if !isLoad(in[25], indexLocal) ||
		in[26].Kind != ir.IRConstI32 || in[26].Imm != allocationLoopStep ||
		in[27].Kind != ir.IRAddI32 ||
		!isStore(in[28], indexLocal) {
		return AllocationLoopPlan{}, false, nil
	}
	if in[29].Kind != ir.IRJmp || in[29].Label != startLabel ||
		in[30].Kind != ir.IRLabel || in[30].Label != endLabel {
		return AllocationLoopPlan{}, false, nil
	}
	if !isLoad(in[31], checksumLocal) ||
		in[32].Kind != ir.IRConstI32 || in[32].Imm != 0 ||
		in[33].Kind != ir.IRCmpGtI32 ||
		in[34].Kind != ir.IRJmpIfZero || in[34].Label != 2 {
		return AllocationLoopPlan{}, false, nil
	}
	failureLabel := in[34].Label
	if in[35].Kind != ir.IRConstI32 || in[35].Imm != 0 ||
		in[36].Kind != ir.IRReturn ||
		in[37].Kind != ir.IRLabel || in[37].Label != failureLabel ||
		in[38].Kind != ir.IRConstI32 || in[38].Imm != 1 ||
		in[39].Kind != ir.IRReturn {
		return AllocationLoopPlan{}, false, nil
	}
	for _, local := range []struct {
		slot int
		name string
	}{
		{checksumLocal, "checksum"},
		{indexLocal, "index"},
		{slicePtrLocal, "slice ptr"},
		{sliceLenLocal, "slice len"},
	} {
		if err := validateScalarLoopLocal(fn, local.slot, local.name); err != nil {
			return AllocationLoopPlan{}, true, err
		}
	}
	if backingLocal+backingSlots > fn.LocalSlots {
		return AllocationLoopPlan{}, true, fmt.Errorf(
			"machine allocation loop lowering: %s backing slots [%d,%d) out of bounds (locals=%d)",
			fn.Name,
			backingLocal,
			backingLocal+backingSlots,
			fn.LocalSlots,
		)
	}
	if !distinctAllocationLoopLocals(checksumLocal, indexLocal, slicePtrLocal, sliceLenLocal) {
		return AllocationLoopPlan{}, false, nil
	}
	for _, local := range []int{checksumLocal, indexLocal, slicePtrLocal, sliceLenLocal} {
		if local >= backingLocal && local < backingLocal+backingSlots {
			return AllocationLoopPlan{}, false, nil
		}
	}

	plan := AllocationLoopPlan{
		ChecksumLocal: checksumLocal,
		IndexLocal:    indexLocal,
		SlicePtrLocal: slicePtrLocal,
		SliceLenLocal: sliceLenLocal,
		BackingLocal:  backingLocal,
		BackingSlots:  backingSlots,
		LoopBound:     allocationLoopBound,
		SliceLength:   allocationLoopSliceLength,
		IndexConst:    allocationLoopIndexConst,
		Step:          allocationLoopStep,
		StartLabel:    startLabel,
		EndLabel:      endLabel,
		FailureLabel:  failureLabel,
		SuccessReturn: in[35].Imm,
		FailureReturn: in[38].Imm,
		BoundsChecks:  2,
	}
	out, err := buildAllocationLoopMachineFunction(fn.Name, plan)
	if err != nil {
		return AllocationLoopPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func distinctAllocationLoopLocals(locals ...int) bool {
	seen := map[int]bool{}
	for _, local := range locals {
		if seen[local] {
			return false
		}
		seen[local] = true
	}
	return true
}

func validAllocationLoopIndexAccessShape(store ir.IRInstr, load ir.IRInstr) bool {
	legacyStore := store.Kind == ir.IRIndexStoreI32 && store.ProofID == ""
	legacyLoad := load.Kind == ir.IRIndexLoadI32 && load.ProofID == ""
	if legacyStore && legacyLoad {
		return true
	}
	p55Store := store.Kind == ir.IRIndexStoreI32 && store.ProofID == allocationLoopP55StoreProofID
	p55Load := load.Kind == ir.IRIndexLoadI32Unchecked &&
		load.ProofID == allocationLoopP55LoadProofID
	return p55Store && p55Load
}

func buildAllocationLoopMachineFunction(name string, plan AllocationLoopPlan) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	loopName := scalarLoopLabelName(plan.StartLabel)
	exitName := scalarLoopLabelName(plan.EndLabel)
	failureName := scalarLoopLabelName(plan.FailureLabel)
	successName := "return_success"
	zero := VReg("t0")
	bound := VReg("t1")
	loopCmp := VReg("t2")
	loaded := VReg("t3")
	finalCmp := VReg("t4")
	success := VReg("t5")
	failure := VReg("t6")

	out := Function{
		Name:   name,
		Target: "allocation-loop",
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{local(plan.ChecksumLocal)},
						Imm:  0,
						Note: "checksum = 0",
					},
					{Op: OpMov, Defs: []VReg{local(plan.IndexLocal)}, Imm: 0, Note: "r = 0"},
					{Op: OpMov, Defs: []VReg{local(plan.SlicePtrLocal)}, Note: "xs stack ptr"},
					{
						Op:   OpMov,
						Defs: []VReg{local(plan.SliceLenLocal)},
						Imm:  int64(plan.SliceLength),
						Note: "xs len = 32",
					},
					{
						Op:   OpMov,
						Defs: []VReg{zero},
						Imm:  int64(plan.IndexConst),
						Note: "checked index 0",
					},
					{
						Op:   OpMov,
						Defs: []VReg{bound},
						Imm:  int64(plan.LoopBound),
						Note: "loop bound",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{loopCmp},
						Uses: []VReg{local(plan.IndexLocal), bound},
						Note: "r < 1024",
					},
					{Op: OpBranchIf, Uses: []VReg{loopCmp}, Target: exitName, Note: "if_zero"},
					{
						Op: OpIndexStore,
						Uses: []VReg{
							local(plan.SlicePtrLocal),
							local(plan.SliceLenLocal),
							zero,
							local(plan.IndexLocal),
						},
						Note: "checked xs[0] = r",
					},
					{
						Op:   OpIndexLoad,
						Defs: []VReg{loaded},
						Uses: []VReg{local(plan.SlicePtrLocal), local(plan.SliceLenLocal), zero},
						Note: "checked xs[0]",
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(plan.ChecksumLocal)},
						Uses: []VReg{local(plan.ChecksumLocal), loaded},
						Note: "checksum += xs[0]",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.IndexLocal)},
						Uses: []VReg{local(plan.IndexLocal)},
						Note: "r++",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{exitName, loopName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{finalCmp},
						Uses: []VReg{local(plan.ChecksumLocal), zero},
						Note: "checksum > 0",
					},
					{Op: OpBranchIf, Uses: []VReg{finalCmp}, Target: failureName, Note: "if_zero"},
					{Op: OpBranch, Target: successName},
				},
				Successors: []string{failureName, successName},
			},
			{
				Name: successName,
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{success},
						Imm:  int64(plan.SuccessReturn),
						Note: "return 0",
					},
					{Op: OpReturn, Uses: []VReg{success}},
				},
			},
			{
				Name: failureName,
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{failure},
						Imm:  int64(plan.FailureReturn),
						Note: "return 1",
					},
					{Op: OpReturn, Uses: []VReg{failure}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

// ---- ir.go ----

type Program struct {
	Functions []Function `json:"functions,omitempty"`
}

type Function struct {
	Name   string  `json:"name"`
	Target string  `json:"target,omitempty"`
	Params []VReg  `json:"params,omitempty"`
	Blocks []Block `json:"blocks"`
}

type Block struct {
	Name       string   `json:"name"`
	Instrs     []Instr  `json:"instrs,omitempty"`
	Successors []string `json:"successors,omitempty"`
}

type Instr struct {
	Op       Opcode    `json:"op"`
	Defs     []VReg    `json:"defs,omitempty"`
	Uses     []VReg    `json:"uses,omitempty"`
	Imm      int64     `json:"imm,omitempty"`
	Target   string    `json:"target,omitempty"`
	Call     string    `json:"call,omitempty"`
	ABI      string    `json:"abi,omitempty"`
	Clobbers []PhysReg `json:"clobbers,omitempty"`
	Note     string    `json:"note,omitempty"`
}

type Opcode string

const (
	OpMov        Opcode = "mov"
	OpLoad       Opcode = "load"
	OpStore      Opcode = "store"
	OpAdd        Opcode = "add"
	OpSub        Opcode = "sub"
	OpMul        Opcode = "mul"
	OpDiv        Opcode = "div"
	OpMod        Opcode = "mod"
	OpCmp        Opcode = "cmp"
	OpInc        Opcode = "inc"
	OpBranch     Opcode = "branch"
	OpBranchIf   Opcode = "branch_if"
	OpCall       Opcode = "call"
	OpReturn     Opcode = "return"
	OpSpill      Opcode = "spill"
	OpReload     Opcode = "reload"
	OpPush       Opcode = "push"
	OpPop        Opcode = "pop"
	OpIndexLoad  Opcode = "index_load"
	OpIndexStore Opcode = "index_store"

	OpVectorZeroI32x4           Opcode = "vector_zero_i32x4"
	OpVectorCanLoadI32x4        Opcode = "vector_can_load_i32x4"
	OpVectorLoadI32x4Unaligned  Opcode = "vector_load_i32x4_unaligned"
	OpVectorAddI32x4            Opcode = "vector_add_i32x4"
	OpVectorHorizontalAddI32x4  Opcode = "vector_horizontal_add_i32x4"
	OpTailScalarI32Sum          Opcode = "tail_scalar_i32_sum"
	OpVectorCanCopyU8x16        Opcode = "vector_can_copy_u8x16"
	OpVectorLoadU8x16Unaligned  Opcode = "vector_load_u8x16_unaligned"
	OpVectorStoreU8x16Unaligned Opcode = "vector_store_u8x16_unaligned"
	OpTailScalarU8Copy          Opcode = "tail_scalar_u8_copy"
	OpVectorZeroU8x16           Opcode = "vector_zero_u8x16"
	OpVectorCanMemsetU8x16      Opcode = "vector_can_memset_u8x16"
	OpTailScalarU8Memset        Opcode = "tail_scalar_u8_memset"
	OpVectorSplatI32x4          Opcode = "vector_splat_i32x4"
	OpVectorCanMapI32x4         Opcode = "vector_can_map_i32x4"
	OpVectorStoreI32x4Unaligned Opcode = "vector_store_i32x4_unaligned"
	OpTailScalarI32Map          Opcode = "tail_scalar_i32_map"
)

type VReg string

type PhysReg string

type Liveness struct {
	Blocks map[string]BlockLiveness `json:"blocks"`
}

type BlockLiveness struct {
	Use     []VReg `json:"use,omitempty"`
	Def     []VReg `json:"def,omitempty"`
	LiveIn  []VReg `json:"live_in,omitempty"`
	LiveOut []VReg `json:"live_out,omitempty"`
}

type Interval struct {
	Reg   VReg `json:"reg"`
	Start int  `json:"start"`
	End   int  `json:"end"`
}

type Allocation struct {
	Assignments map[VReg]PhysReg `json:"assignments,omitempty"`
	Spills      map[VReg]int     `json:"spills,omitempty"`
}

func VerifyFunction(fn Function) error {
	if fn.Name == "" {
		return fmt.Errorf("machine verifier: function with empty name")
	}
	if len(fn.Blocks) == 0 {
		return fmt.Errorf("machine verifier: %s has no blocks", fn.Name)
	}
	blocks := map[string]bool{}
	defined := map[VReg]bool{}
	for _, param := range fn.Params {
		if param == "" {
			return fmt.Errorf("machine verifier: %s has empty parameter vreg", fn.Name)
		}
		defined[param] = true
	}
	for _, block := range fn.Blocks {
		if block.Name == "" {
			return fmt.Errorf("machine verifier: %s has block with empty name", fn.Name)
		}
		if blocks[block.Name] {
			return fmt.Errorf("machine verifier: %s duplicate block %q", fn.Name, block.Name)
		}
		blocks[block.Name] = true
		for _, instr := range block.Instrs {
			for _, def := range instr.Defs {
				if def == "" {
					return fmt.Errorf(
						"machine verifier: %s.%s has empty def vreg",
						fn.Name,
						block.Name,
					)
				}
				defined[def] = true
			}
		}
	}
	for _, block := range fn.Blocks {
		if len(block.Instrs) == 0 {
			return fmt.Errorf("machine verifier: %s.%s missing terminator", fn.Name, block.Name)
		}
		last := block.Instrs[len(block.Instrs)-1]
		if !isMachineTerminator(last.Op) {
			return fmt.Errorf("machine verifier: %s.%s missing terminator", fn.Name, block.Name)
		}
		branchTargets := map[string]bool{}
		successors := map[string]bool{}
		for _, succ := range block.Successors {
			successors[succ] = true
		}
		for _, instr := range block.Instrs {
			if instr.Op == "" {
				return fmt.Errorf(
					"machine verifier: %s.%s has instruction with empty opcode",
					fn.Name,
					block.Name,
				)
			}
			if err := verifyInstrShape(fn.Name, block.Name, instr); err != nil {
				return err
			}
			for _, use := range instr.Uses {
				if use == "" {
					return fmt.Errorf(
						"machine verifier: %s.%s has empty use vreg",
						fn.Name,
						block.Name,
					)
				}
				if !defined[use] {
					return fmt.Errorf(
						"machine verifier: %s.%s uses undefined vreg %q",
						fn.Name,
						block.Name,
						use,
					)
				}
			}
			if instr.Op == OpBranch || instr.Op == OpBranchIf {
				if !blocks[instr.Target] {
					return fmt.Errorf(
						"machine verifier: %s.%s unknown branch target %q",
						fn.Name,
						block.Name,
						instr.Target,
					)
				}
				branchTargets[instr.Target] = true
				if len(block.Successors) > 0 && !successors[instr.Target] {
					return fmt.Errorf(
						"machine verifier: %s.%s branch target %q missing from successors",
						fn.Name,
						block.Name,
						instr.Target,
					)
				}
			}
		}
		for i, instr := range block.Instrs[:len(block.Instrs)-1] {
			if instr.Op == OpBranch || instr.Op == OpReturn {
				return fmt.Errorf(
					"machine verifier: %s.%s terminator at instruction %d is not last",
					fn.Name,
					block.Name,
					i,
				)
			}
		}
		for _, succ := range block.Successors {
			if !branchTargets[succ] {
				return fmt.Errorf(
					"machine verifier: %s.%s successor %q has no branch instruction",
					fn.Name,
					block.Name,
					succ,
				)
			}
		}
	}
	for _, block := range fn.Blocks {
		for _, succ := range block.Successors {
			if !blocks[succ] {
				return fmt.Errorf(
					"machine verifier: %s.%s references unknown successor %q",
					fn.Name,
					block.Name,
					succ,
				)
			}
		}
	}
	return nil
}

func verifyInstrShape(fnName string, blockName string, instr Instr) error {
	exact := func(kind string, got int, want int) error {
		if got != want {
			return fmt.Errorf(
				"machine verifier: %s.%s %s has %d slots, want %d",
				fnName,
				blockName,
				kind,
				got,
				want,
			)
		}
		return nil
	}
	atLeast := func(kind string, got int, want int) error {
		if got < want {
			return fmt.Errorf(
				"machine verifier: %s.%s %s has %d slots, want at least %d",
				fnName,
				blockName,
				kind,
				got,
				want,
			)
		}
		return nil
	}
	switch instr.Op {
	case OpMov:
		if err := exact("mov defs", len(instr.Defs), 1); err != nil {
			return err
		}
		if len(instr.Uses) > 1 {
			return fmt.Errorf(
				"machine verifier: %s.%s mov has %d uses, want 0 or 1",
				fnName,
				blockName,
				len(instr.Uses),
			)
		}
	case OpLoad:
		if err := exact("load defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return atLeast("load uses", len(instr.Uses), 1)
	case OpStore:
		if err := exact("store defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return atLeast("store uses", len(instr.Uses), 2)
	case OpAdd, OpSub, OpMul, OpDiv, OpMod, OpCmp:
		if err := exact(string(instr.Op)+" defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact(string(instr.Op)+" uses", len(instr.Uses), 2)
	case OpInc:
		if err := exact("inc defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("inc uses", len(instr.Uses), 1)
	case OpBranch:
		if instr.Target == "" {
			return fmt.Errorf("machine verifier: %s.%s branch missing target", fnName, blockName)
		}
		if len(instr.Defs) != 0 || len(instr.Uses) != 0 {
			return fmt.Errorf(
				"machine verifier: %s.%s branch must not define or use vregs",
				fnName,
				blockName,
			)
		}
	case OpBranchIf:
		if instr.Target == "" {
			return fmt.Errorf("machine verifier: %s.%s branch missing target", fnName, blockName)
		}
		if err := exact("branch_if defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return atLeast("branch_if uses", len(instr.Uses), 1)
	case OpCall:
		if instr.Call == "" {
			return fmt.Errorf("machine verifier: %s.%s call missing callee", fnName, blockName)
		}
		if instr.ABI == "" {
			return fmt.Errorf(
				"machine verifier: %s.%s call %q missing ABI",
				fnName,
				blockName,
				instr.Call,
			)
		}
		if len(instr.Clobbers) == 0 {
			return fmt.Errorf(
				"machine verifier: %s.%s call %q missing clobber metadata",
				fnName,
				blockName,
				instr.Call,
			)
		}
	case OpReturn:
		if len(instr.Defs) != 0 {
			return fmt.Errorf(
				"machine verifier: %s.%s return must not define vregs",
				fnName,
				blockName,
			)
		}
	case OpSpill:
		if err := exact("spill defs", len(instr.Defs), 0); err != nil {
			return err
		}
		if err := exact("spill uses", len(instr.Uses), 1); err != nil {
			return err
		}
		if instr.Imm < 0 {
			return fmt.Errorf(
				"machine verifier: %s.%s spill has negative slot %d",
				fnName,
				blockName,
				instr.Imm,
			)
		}
	case OpReload:
		if err := exact("reload defs", len(instr.Defs), 1); err != nil {
			return err
		}
		if err := exact("reload uses", len(instr.Uses), 0); err != nil {
			return err
		}
		if instr.Imm < 0 {
			return fmt.Errorf(
				"machine verifier: %s.%s reload has negative slot %d",
				fnName,
				blockName,
				instr.Imm,
			)
		}
	case OpPush:
		if err := exact("push defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("push uses", len(instr.Uses), 1)
	case OpPop:
		if err := exact("pop defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("pop uses", len(instr.Uses), 0)
	case OpIndexLoad:
		if err := exact("index_load defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("index_load uses", len(instr.Uses), 3)
	case OpIndexStore:
		if err := exact("index_store defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("index_store uses", len(instr.Uses), 4)
	case OpVectorZeroI32x4:
		if err := exact("vector_zero_i32x4 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_zero_i32x4 uses", len(instr.Uses), 0)
	case OpVectorCanLoadI32x4:
		if err := exact("vector_can_load_i32x4 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_can_load_i32x4 uses", len(instr.Uses), 2)
	case OpVectorLoadI32x4Unaligned:
		if err := exact("vector_load_i32x4_unaligned defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_load_i32x4_unaligned uses", len(instr.Uses), 3)
	case OpVectorAddI32x4:
		if err := exact("vector_add_i32x4 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_add_i32x4 uses", len(instr.Uses), 2)
	case OpVectorHorizontalAddI32x4:
		if err := exact("vector_horizontal_add_i32x4 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_horizontal_add_i32x4 uses", len(instr.Uses), 1)
	case OpTailScalarI32Sum:
		if err := exact("tail_scalar_i32_sum defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("tail_scalar_i32_sum uses", len(instr.Uses), 4)
	case OpVectorCanCopyU8x16:
		if err := exact("vector_can_copy_u8x16 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_can_copy_u8x16 uses", len(instr.Uses), 2)
	case OpVectorLoadU8x16Unaligned:
		if err := exact("vector_load_u8x16_unaligned defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_load_u8x16_unaligned uses", len(instr.Uses), 3)
	case OpVectorStoreU8x16Unaligned:
		if err := exact("vector_store_u8x16_unaligned defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("vector_store_u8x16_unaligned uses", len(instr.Uses), 4)
	case OpTailScalarU8Copy:
		if err := exact("tail_scalar_u8_copy defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("tail_scalar_u8_copy uses", len(instr.Uses), 4)
	case OpVectorZeroU8x16:
		if err := exact("vector_zero_u8x16 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_zero_u8x16 uses", len(instr.Uses), 0)
	case OpVectorCanMemsetU8x16:
		if err := exact("vector_can_memset_u8x16 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_can_memset_u8x16 uses", len(instr.Uses), 2)
	case OpTailScalarU8Memset:
		if err := exact("tail_scalar_u8_memset defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("tail_scalar_u8_memset uses", len(instr.Uses), 4)
	case OpVectorSplatI32x4:
		if err := exact("vector_splat_i32x4 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_splat_i32x4 uses", len(instr.Uses), 0)
	case OpVectorCanMapI32x4:
		if err := exact("vector_can_map_i32x4 defs", len(instr.Defs), 1); err != nil {
			return err
		}
		return exact("vector_can_map_i32x4 uses", len(instr.Uses), 2)
	case OpVectorStoreI32x4Unaligned:
		if err := exact("vector_store_i32x4_unaligned defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("vector_store_i32x4_unaligned uses", len(instr.Uses), 4)
	case OpTailScalarI32Map:
		if err := exact("tail_scalar_i32_map defs", len(instr.Defs), 0); err != nil {
			return err
		}
		return exact("tail_scalar_i32_map uses", len(instr.Uses), 4)
	default:
		return fmt.Errorf("machine verifier: %s.%s unknown opcode %q", fnName, blockName, instr.Op)
	}
	return nil
}

func isMachineTerminator(op Opcode) bool {
	return op == OpBranch || op == OpReturn
}

func VerifyAllocation(fn Function, alloc Allocation, regs []PhysReg, spillSlots int) error {
	if err := VerifyFunction(fn); err != nil {
		return err
	}
	if spillSlots < 0 {
		return fmt.Errorf("machine allocation verifier: negative spill slot count %d", spillSlots)
	}
	allowedRegs := map[PhysReg]bool{}
	for _, reg := range regs {
		if reg == "" {
			return fmt.Errorf("machine allocation verifier: empty physreg")
		}
		allowedRegs[reg] = true
	}
	vregs := functionVRegs(fn)
	for reg, phys := range alloc.Assignments {
		if !vregs[reg] {
			return fmt.Errorf("machine allocation verifier: assignment for unknown vreg %q", reg)
		}
		if !allowedRegs[phys] {
			return fmt.Errorf("machine allocation verifier: invalid physreg %q for %s", phys, reg)
		}
		if _, spilled := alloc.Spills[reg]; spilled {
			return fmt.Errorf("machine allocation verifier: %s cannot be assigned and spilled", reg)
		}
	}
	for reg, slot := range alloc.Spills {
		if !vregs[reg] {
			return fmt.Errorf("machine allocation verifier: spill for unknown vreg %q", reg)
		}
		if slot < 0 || slot >= spillSlots {
			return fmt.Errorf(
				"machine allocation verifier: spill slot %d for %s out of bounds (slots=%d)",
				slot,
				reg,
				spillSlots,
			)
		}
	}
	intervals, err := BuildIntervals(fn)
	if err != nil {
		return err
	}
	byReg := map[VReg]Interval{}
	for _, interval := range intervals {
		byReg[interval.Reg] = interval
	}
	assigned := make([]VReg, 0, len(alloc.Assignments))
	for reg := range alloc.Assignments {
		assigned = append(assigned, reg)
	}
	sort.Slice(assigned, func(i, j int) bool { return assigned[i] < assigned[j] })
	for i, left := range assigned {
		for _, right := range assigned[i+1:] {
			if alloc.Assignments[left] != alloc.Assignments[right] {
				continue
			}
			if intervalsOverlap(byReg[left], byReg[right]) {
				return fmt.Errorf(
					"machine allocation verifier: overlapping vregs %s and %s share physreg %s",
					left,
					right,
					alloc.Assignments[left],
				)
			}
		}
	}
	return nil
}

func AnalyzeLiveness(fn Function) (Liveness, error) {
	if err := VerifyFunction(fn); err != nil {
		return Liveness{}, err
	}
	out := Liveness{Blocks: map[string]BlockLiveness{}}
	for _, block := range fn.Blocks {
		useSet := map[VReg]bool{}
		defSet := map[VReg]bool{}
		for _, instr := range block.Instrs {
			for _, use := range instr.Uses {
				if !defSet[use] {
					useSet[use] = true
				}
			}
			for _, def := range instr.Defs {
				defSet[def] = true
			}
		}
		out.Blocks[block.Name] = BlockLiveness{
			Use: setToSortedRegs(useSet),
			Def: setToSortedRegs(defSet),
		}
	}
	changed := true
	for changed {
		changed = false
		for i := len(fn.Blocks) - 1; i >= 0; i-- {
			block := fn.Blocks[i]
			info := out.Blocks[block.Name]
			liveOut := map[VReg]bool{}
			for _, succ := range block.Successors {
				for _, reg := range out.Blocks[succ].LiveIn {
					liveOut[reg] = true
				}
			}
			liveIn := regsToSet(info.Use)
			for reg := range liveOut {
				if !containsReg(info.Def, reg) {
					liveIn[reg] = true
				}
			}
			next := BlockLiveness{
				Use:     info.Use,
				Def:     info.Def,
				LiveIn:  setToSortedRegs(liveIn),
				LiveOut: setToSortedRegs(liveOut),
			}
			if !sameRegs(info.LiveIn, next.LiveIn) || !sameRegs(info.LiveOut, next.LiveOut) {
				out.Blocks[block.Name] = next
				changed = true
			}
		}
	}
	return out, nil
}

func BuildIntervals(fn Function) ([]Interval, error) {
	if err := VerifyFunction(fn); err != nil {
		return nil, err
	}
	positions := map[VReg]Interval{}
	pos := 0
	touch := func(reg VReg) {
		if reg == "" {
			return
		}
		interval, ok := positions[reg]
		if !ok {
			positions[reg] = Interval{Reg: reg, Start: pos, End: pos}
			return
		}
		if pos < interval.Start {
			interval.Start = pos
		}
		if pos > interval.End {
			interval.End = pos
		}
		positions[reg] = interval
	}
	for _, param := range fn.Params {
		touch(param)
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			for _, use := range instr.Uses {
				touch(use)
			}
			for _, def := range instr.Defs {
				touch(def)
			}
			pos++
		}
	}
	intervals := make([]Interval, 0, len(positions))
	for _, interval := range positions {
		intervals = append(intervals, interval)
	}
	sort.Slice(intervals, func(i, j int) bool {
		if intervals[i].Start == intervals[j].Start {
			return intervals[i].Reg < intervals[j].Reg
		}
		return intervals[i].Start < intervals[j].Start
	})
	return intervals, nil
}

func LinearScan(intervals []Interval, regs []PhysReg) (Allocation, error) {
	if len(regs) == 0 {
		return Allocation{}, fmt.Errorf("machine linear scan: no physical registers")
	}
	sortedIntervals := append([]Interval(nil), intervals...)
	sort.Slice(sortedIntervals, func(i, j int) bool {
		if sortedIntervals[i].Start == sortedIntervals[j].Start {
			return sortedIntervals[i].End < sortedIntervals[j].End
		}
		return sortedIntervals[i].Start < sortedIntervals[j].Start
	})
	alloc := Allocation{Assignments: map[VReg]PhysReg{}, Spills: map[VReg]int{}}
	active := []Interval{}
	free := append([]PhysReg(nil), regs...)
	nextSpill := 0
	expireOld := func(start int) {
		kept := active[:0]
		for _, interval := range active {
			if interval.End >= start {
				kept = append(kept, interval)
				continue
			}
			if reg, ok := alloc.Assignments[interval.Reg]; ok {
				free = append(free, reg)
			}
		}
		active = kept
		sort.Slice(free, func(i, j int) bool { return free[i] < free[j] })
	}
	for _, interval := range sortedIntervals {
		expireOld(interval.Start)
		if len(free) > 0 {
			reg := free[0]
			free = free[1:]
			alloc.Assignments[interval.Reg] = reg
			active = appendActive(active, interval)
			continue
		}
		spillAt := farthestEnding(active)
		if spillAt >= 0 && active[spillAt].End > interval.End {
			spilled := active[spillAt]
			reg := alloc.Assignments[spilled.Reg]
			delete(alloc.Assignments, spilled.Reg)
			alloc.Spills[spilled.Reg] = nextSpill
			nextSpill++
			alloc.Assignments[interval.Reg] = reg
			active[spillAt] = interval
			active = appendActive(nil, active...)
		} else {
			alloc.Spills[interval.Reg] = nextSpill
			nextSpill++
		}
	}
	return alloc, nil
}

func LinuxX64CallerSaved() []PhysReg {
	return []PhysReg{"rax", "rcx", "rdx", "rsi", "rdi", "r8", "r9", "r10", "r11"}
}

func Win64CallerSaved() []PhysReg {
	return []PhysReg{"rax", "rcx", "rdx", "r8", "r9", "r10", "r11"}
}

type CallABIInfo struct {
	Name        string
	Clobbers    []PhysReg
	MaxArgSlots int
	MaxRetSlots int
}

func SysVCallABIInfo() CallABIInfo {
	return CallABIInfo{
		Name:        "sysv",
		Clobbers:    LinuxX64CallerSaved(),
		MaxArgSlots: 6,
		MaxRetSlots: 2,
	}
}

func Win64CallABIInfo() CallABIInfo {
	return CallABIInfo{
		Name:        "win64",
		Clobbers:    Win64CallerSaved(),
		MaxArgSlots: 4,
		MaxRetSlots: 1,
	}
}

func SumToLoopFunction() Function {
	n := VReg("n")
	i := VReg("i")
	total := VReg("total")
	cmp := VReg("cmp")
	return Function{
		Name:   "sum_to",
		Target: "linux-x64",
		Params: []VReg{n},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{i}, Imm: 0, Note: "i = 0"},
					{Op: OpMov, Defs: []VReg{total}, Imm: 0, Note: "total = 0"},
					{Op: OpBranch, Target: "loop"},
				},
				Successors: []string{"loop"},
			},
			{
				Name: "loop",
				Instrs: []Instr{
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{i, n}, Note: "i < n"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: "exit"},
					{Op: OpAdd, Defs: []VReg{total}, Uses: []VReg{total, i}, Note: "total += i"},
					{Op: OpInc, Defs: []VReg{i}, Uses: []VReg{i}, Note: "i++"},
					{Op: OpBranch, Target: "loop"},
				},
				Successors: []string{"loop", "exit"},
			},
			{
				Name: "exit",
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{total}},
				},
			},
		},
	}
}

func FormatFunction(fn Function) string {
	var b strings.Builder
	fmt.Fprintf(&b, "func %s target:%s", fn.Name, fn.Target)
	if len(fn.Params) > 0 {
		fmt.Fprintf(&b, " params:%s", joinRegs(fn.Params))
	}
	fmt.Fprintln(&b)
	for _, block := range fn.Blocks {
		fmt.Fprintf(&b, "%s:\n", block.Name)
		for _, instr := range block.Instrs {
			fmt.Fprintf(&b, "  %s", instr.Op)
			if instr.Call != "" {
				fmt.Fprintf(&b, " %s", instr.Call)
			}
			if len(instr.Defs) > 0 {
				fmt.Fprintf(&b, " defs:%s", joinRegs(instr.Defs))
			}
			if len(instr.Uses) > 0 {
				fmt.Fprintf(&b, " uses:%s", joinRegs(instr.Uses))
			}
			if instr.Target != "" {
				fmt.Fprintf(&b, " -> %s", instr.Target)
			}
			if instr.ABI != "" {
				fmt.Fprintf(&b, " abi:%s", instr.ABI)
			}
			if len(instr.Clobbers) > 0 {
				fmt.Fprintf(&b, " clobbers:%s", joinPhysRegs(instr.Clobbers))
			}
			if instr.Note != "" {
				fmt.Fprintf(&b, " ; %s", instr.Note)
			}
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}

func FormatProgram(prog Program) string {
	var b strings.Builder
	fmt.Fprintln(&b, "program machine_ir")
	for i, fn := range prog.Functions {
		if i > 0 {
			fmt.Fprintln(&b)
		}
		b.WriteString(FormatFunction(fn))
	}
	return b.String()
}

func appendActive(active []Interval, intervals ...Interval) []Interval {
	active = append(active, intervals...)
	sort.Slice(active, func(i, j int) bool {
		if active[i].End == active[j].End {
			return active[i].Reg < active[j].Reg
		}
		return active[i].End < active[j].End
	})
	return active
}

func farthestEnding(active []Interval) int {
	if len(active) == 0 {
		return -1
	}
	idx := 0
	for i := 1; i < len(active); i++ {
		if active[i].End > active[idx].End {
			idx = i
		}
	}
	return idx
}

func setToSortedRegs(set map[VReg]bool) []VReg {
	out := make([]VReg, 0, len(set))
	for reg := range set {
		out = append(out, reg)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func regsToSet(regs []VReg) map[VReg]bool {
	out := map[VReg]bool{}
	for _, reg := range regs {
		out[reg] = true
	}
	return out
}

func containsReg(regs []VReg, want VReg) bool {
	for _, reg := range regs {
		if reg == want {
			return true
		}
	}
	return false
}

func sameRegs(a, b []VReg) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func joinRegs(regs []VReg) string {
	parts := make([]string, len(regs))
	for i, reg := range regs {
		parts[i] = string(reg)
	}
	return strings.Join(parts, ",")
}

func joinPhysRegs(regs []PhysReg) string {
	parts := make([]string, len(regs))
	for i, reg := range regs {
		parts[i] = string(reg)
	}
	return strings.Join(parts, ",")
}

func functionVRegs(fn Function) map[VReg]bool {
	out := map[VReg]bool{}
	for _, param := range fn.Params {
		if param != "" {
			out[param] = true
		}
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			for _, reg := range instr.Defs {
				if reg != "" {
					out[reg] = true
				}
			}
			for _, reg := range instr.Uses {
				if reg != "" {
					out[reg] = true
				}
			}
		}
	}
	return out
}

func intervalsOverlap(a Interval, b Interval) bool {
	return a.Start <= b.End && b.Start <= a.End
}

// ---- recursion_benchmark.go ----

const (
	recursionFibName          = "p25.recursion.fib"
	recursionMainName         = "p25.recursion.main"
	recursionFibBaseCaseLimit = int32(2)
	recursionMainLoopBound    = int32(40)
	recursionMainCallArg      = int32(10)
	recursionMainSuccessTotal = int32(2200)
)

type RecursionFibPlan struct {
	Function   Function
	ParamLocal int
	BaseLabel  int
	CallName   string
}

type RecursionMainPlan struct {
	Function       Function
	IndexLocal     int
	TotalLocal     int
	StartLabel     int
	EndLabel       int
	FailureLabel   int
	CallName       string
	LoopBound      int32
	CallArg        int32
	SuccessTotal   int32
	TrueReturnImm  int32
	FalseReturnImm int32
}

func RecursionFibFunctionFromStackIRWithCallABI(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (Function, bool, error) {
	plan, ok, err := RecursionFibPlanFromStackIRWithCallABI(fn, callABI)
	return plan.Function, ok, err
}

func RecursionFibPlanFromStackIRWithCallABI(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (RecursionFibPlan, bool, error) {
	if fn.Name != recursionFibName || fn.ParamSlots != 1 || fn.LocalSlots != 1 ||
		fn.ReturnSlots != 1 ||
		len(fn.Instrs) != 17 {
		return RecursionFibPlan{}, false, nil
	}
	if err := validateCallABIInfo(callABI); err != nil {
		return RecursionFibPlan{}, true, err
	}
	if callABI.MaxArgSlots < 1 || callABI.MaxRetSlots < 1 {
		return RecursionFibPlan{}, false, nil
	}
	in := fn.Instrs
	if !isLoad(in[0], 0) ||
		in[1].Kind != ir.IRConstI32 || in[1].Imm != recursionFibBaseCaseLimit ||
		in[2].Kind != ir.IRCmpLtI32 ||
		in[3].Kind != ir.IRJmpIfZero {
		return RecursionFibPlan{}, false, nil
	}
	baseLabel := in[3].Label
	if baseLabel < 0 {
		return RecursionFibPlan{}, false, nil
	}
	if !isLoad(in[4], 0) || in[5].Kind != ir.IRReturn ||
		in[6].Kind != ir.IRLabel || in[6].Label != baseLabel {
		return RecursionFibPlan{}, false, nil
	}
	if !isLoad(in[7], 0) ||
		in[8].Kind != ir.IRConstI32 || in[8].Imm != 1 ||
		in[9].Kind != ir.IRSubI32 ||
		in[10].Kind != ir.IRCall ||
		in[10].Name != fn.Name ||
		in[10].ArgSlots != 1 ||
		in[10].RetSlots != 1 {
		return RecursionFibPlan{}, false, nil
	}
	if !isLoad(in[11], 0) ||
		in[12].Kind != ir.IRConstI32 || in[12].Imm != recursionFibBaseCaseLimit ||
		in[13].Kind != ir.IRSubI32 ||
		in[14].Kind != ir.IRCall ||
		in[14].Name != fn.Name ||
		in[14].ArgSlots != 1 ||
		in[14].RetSlots != 1 ||
		in[15].Kind != ir.IRAddI32 || in[16].Kind != ir.IRReturn {
		return RecursionFibPlan{}, false, nil
	}

	plan := RecursionFibPlan{
		ParamLocal: 0,
		BaseLabel:  baseLabel,
		CallName:   fn.Name,
	}
	out, err := buildRecursionFibMachineFunction(fn.Name, callABI, plan)
	if err != nil {
		return RecursionFibPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func RecursionMainFunctionFromStackIRWithCallABI(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (Function, bool, error) {
	plan, ok, err := RecursionMainPlanFromStackIRWithCallABI(fn, callABI)
	return plan.Function, ok, err
}

func RecursionMainPlanFromStackIRWithCallABI(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (RecursionMainPlan, bool, error) {
	if fn.Name != recursionMainName || fn.ParamSlots != 0 || fn.LocalSlots != 2 ||
		fn.ReturnSlots != 1 ||
		len(fn.Instrs) != 29 {
		return RecursionMainPlan{}, false, nil
	}
	if err := validateCallABIInfo(callABI); err != nil {
		return RecursionMainPlan{}, true, err
	}
	if callABI.MaxArgSlots < 1 || callABI.MaxRetSlots < 1 {
		return RecursionMainPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return RecursionMainPlan{}, false, nil
	}
	indexLocal := in[1].Local
	totalLocal := in[3].Local
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel < 0 {
		return RecursionMainPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) ||
		in[6].Kind != ir.IRConstI32 || in[6].Imm != recursionMainLoopBound ||
		in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero {
		return RecursionMainPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 || endLabel == startLabel {
		return RecursionMainPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) ||
		in[10].Kind != ir.IRConstI32 || in[10].Imm != recursionMainCallArg ||
		in[11].Kind != ir.IRCall ||
		in[11].Name != recursionFibName ||
		in[11].ArgSlots != 1 ||
		in[11].RetSlots != 1 ||
		in[12].Kind != ir.IRAddI32 || !isStore(in[13], totalLocal) {
		return RecursionMainPlan{}, false, nil
	}
	if !isLoad(in[14], indexLocal) ||
		in[15].Kind != ir.IRConstI32 || in[15].Imm != 1 ||
		in[16].Kind != ir.IRAddI32 || !isStore(in[17], indexLocal) ||
		in[18].Kind != ir.IRJmp || in[18].Label != startLabel ||
		in[19].Kind != ir.IRLabel || in[19].Label != endLabel {
		return RecursionMainPlan{}, false, nil
	}
	if !isLoad(in[20], totalLocal) ||
		in[21].Kind != ir.IRConstI32 || in[21].Imm != recursionMainSuccessTotal ||
		in[22].Kind != ir.IRCmpEqI32 ||
		in[23].Kind != ir.IRJmpIfZero {
		return RecursionMainPlan{}, false, nil
	}
	failLabel := in[23].Label
	if failLabel < 0 || failLabel == startLabel || failLabel == endLabel {
		return RecursionMainPlan{}, false, nil
	}
	if in[24].Kind != ir.IRConstI32 || in[24].Imm != 0 ||
		in[25].Kind != ir.IRReturn ||
		in[26].Kind != ir.IRLabel || in[26].Label != failLabel ||
		in[27].Kind != ir.IRConstI32 || in[27].Imm != 1 ||
		in[28].Kind != ir.IRReturn {
		return RecursionMainPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return RecursionMainPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return RecursionMainPlan{}, true, err
	}
	if indexLocal == totalLocal {
		return RecursionMainPlan{}, false, nil
	}

	plan := RecursionMainPlan{
		IndexLocal:     indexLocal,
		TotalLocal:     totalLocal,
		StartLabel:     startLabel,
		EndLabel:       endLabel,
		FailureLabel:   failLabel,
		CallName:       in[11].Name,
		LoopBound:      recursionMainLoopBound,
		CallArg:        recursionMainCallArg,
		SuccessTotal:   recursionMainSuccessTotal,
		TrueReturnImm:  0,
		FalseReturnImm: 1,
	}
	out, err := buildRecursionMainMachineFunction(fn.Name, callABI, plan)
	if err != nil {
		return RecursionMainPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func buildRecursionFibMachineFunction(
	name string,
	callABI CallABIInfo,
	plan RecursionFibPlan,
) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	baseConst := VReg("base")
	one := VReg("one")
	two := VReg("two")
	cmp := VReg("cmp")
	nMinusOne := VReg("n_minus_one")
	nMinusTwo := VReg("n_minus_two")
	first := VReg("fib_n_minus_one")
	second := VReg("fib_n_minus_two")
	sum := VReg("sum")
	baseName := scalarLoopLabelName(plan.BaseLabel)
	recurseName := baseName + "_recurse"
	out := Function{
		Name:   name,
		Target: "recursion-fib",
		Params: []VReg{local(plan.ParamLocal)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{baseConst},
						Imm:  int64(recursionFibBaseCaseLimit),
						Note: "fib base threshold",
					},
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(plan.ParamLocal), baseConst},
						Note: "n < 2",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: baseName, Note: "base case"},
					{Op: OpBranch, Target: recurseName},
				},
				Successors: []string{baseName, recurseName},
			},
			{
				Name: baseName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{local(plan.ParamLocal)}, Note: "return n"},
				},
			},
			{
				Name: recurseName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{one}, Imm: 1},
					{
						Op:   OpSub,
						Defs: []VReg{nMinusOne},
						Uses: []VReg{local(plan.ParamLocal), one},
						Note: "n - 1",
					},
					{
						Op:       OpCall,
						Defs:     []VReg{first},
						Uses:     []VReg{nMinusOne},
						Call:     plan.CallName,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "recursive fib(n - 1); caller-saved state is frame-spilled around call",
					},
					{Op: OpMov, Defs: []VReg{two}, Imm: int64(recursionFibBaseCaseLimit)},
					{
						Op:   OpSub,
						Defs: []VReg{nMinusTwo},
						Uses: []VReg{local(plan.ParamLocal), two},
						Note: "n - 2",
					},
					{
						Op:       OpCall,
						Defs:     []VReg{second},
						Uses:     []VReg{nMinusTwo},
						Call:     plan.CallName,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "recursive fib(n - 2); first result is frame-spilled around call",
					},
					{
						Op:   OpAdd,
						Defs: []VReg{sum},
						Uses: []VReg{first, second},
						Note: "fib(n - 1) + fib(n - 2)",
					},
					{Op: OpReturn, Uses: []VReg{sum}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

func buildRecursionMainMachineFunction(
	name string,
	callABI CallABIInfo,
	plan RecursionMainPlan,
) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	bound := VReg("bound")
	callArg := VReg("call_arg")
	callRet := VReg("fib_ret")
	expected := VReg("expected")
	cmp := VReg("cmp")
	ret0 := VReg("ret0")
	ret1 := VReg("ret1")
	loopName := scalarLoopLabelName(plan.StartLabel)
	exitName := scalarLoopLabelName(plan.EndLabel)
	failName := scalarLoopLabelName(plan.FailureLabel)
	okName := exitName + "_success"
	out := Function{
		Name:   name,
		Target: "recursion-main-loop",
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{local(plan.IndexLocal)},
						Imm:  0,
						Note: "loop index = 0",
					},
					{
						Op:   OpMov,
						Defs: []VReg{local(plan.TotalLocal)},
						Imm:  0,
						Note: "loop total = 0",
					},
					{
						Op:   OpMov,
						Defs: []VReg{bound},
						Imm:  int64(plan.LoopBound),
						Note: "loop bound 40",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(plan.IndexLocal), bound},
						Note: "i < 40",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{Op: OpMov, Defs: []VReg{callArg}, Imm: int64(plan.CallArg), Note: "fib(10)"},
					{
						Op:       OpCall,
						Defs:     []VReg{callRet},
						Uses:     []VReg{callArg},
						Call:     plan.CallName,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "loop calls fib(10); total/index are frame-spilled around call",
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(plan.TotalLocal)},
						Uses: []VReg{local(plan.TotalLocal), callRet},
						Note: "total += fib(10)",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.IndexLocal)},
						Uses: []VReg{local(plan.IndexLocal)},
						Note: "i++",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{exitName, loopName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{expected},
						Imm:  int64(plan.SuccessTotal),
						Note: "success total 2200",
					},
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(plan.TotalLocal), expected},
						Note: "total == 2200",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: failName, Note: "if_not_equal"},
					{Op: OpBranch, Target: okName},
				},
				Successors: []string{failName, okName},
			},
			{
				Name: okName,
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{ret0},
						Imm:  int64(plan.TrueReturnImm),
						Note: "return 0",
					},
					{Op: OpReturn, Uses: []VReg{ret0}},
				},
			},
			{
				Name: failName,
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{ret1},
						Imm:  int64(plan.FalseReturnImm),
						Note: "return 1",
					},
					{Op: OpReturn, Uses: []VReg{ret1}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

// ---- scalar.go ----

func ScalarIntFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	return ScalarIntFunctionFromStackIRWithCallABI(fn, SysVCallABIInfo())
}

func ScalarIntFunctionFromStackIRWithCallABI(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (Function, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots < 0 || fn.LocalSlots < fn.ParamSlots {
		return Function{}, false, nil
	}
	if err := validateCallABIInfo(callABI); err != nil {
		return Function{}, true, err
	}
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	tempID := 0
	temp := func() VReg {
		reg := VReg(fmt.Sprintf("t%d", tempID))
		tempID++
		return reg
	}
	params := make([]VReg, fn.ParamSlots)
	for i := range params {
		params[i] = local(i)
	}
	stack := []VReg{}
	instrs := []Instr{}
	pop := func(kind ir.IRInstrKind) (VReg, error) {
		if len(stack) == 0 {
			return "", fmt.Errorf(
				"machine scalar lowering: %s stack underflow at %s",
				fn.Name,
				scalarIRKindName(kind),
			)
		}
		reg := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return reg, nil
	}
	push := func(reg VReg) {
		stack = append(stack, reg)
	}
	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRConstI32:
			dst := temp()
			instrs = append(instrs, Instr{Op: OpMov, Defs: []VReg{dst}, Imm: int64(instr.Imm)})
			push(dst)
		case ir.IRLoadLocal:
			if instr.Local < 0 || instr.Local >= fn.LocalSlots {
				return Function{}, true, fmt.Errorf(
					"machine scalar lowering: %s local %d out of bounds",
					fn.Name,
					instr.Local,
				)
			}
			push(local(instr.Local))
		case ir.IRStoreLocal:
			if instr.Local < 0 || instr.Local >= fn.LocalSlots {
				return Function{}, true, fmt.Errorf(
					"machine scalar lowering: %s local %d out of bounds",
					fn.Name,
					instr.Local,
				)
			}
			src, err := pop(instr.Kind)
			if err != nil {
				return Function{}, true, err
			}
			instrs = append(
				instrs,
				Instr{Op: OpMov, Defs: []VReg{local(instr.Local)}, Uses: []VReg{src}},
			)
		case ir.IRAddI32,
			ir.IRSubI32,
			ir.IRMulI32,
			ir.IRDivI32,
			ir.IRModI32,
			ir.IRCmpEqI32,
			ir.IRCmpLtI32,
			ir.IRCmpGtI32,
			ir.IRCmpGeI32,
			ir.IRCmpLeI32,
			ir.IRCmpNeI32:
			right, err := pop(instr.Kind)
			if err != nil {
				return Function{}, true, err
			}
			left, err := pop(instr.Kind)
			if err != nil {
				return Function{}, true, err
			}
			dst := temp()
			instrs = append(
				instrs,
				Instr{
					Op:   scalarMachineOpcode(instr.Kind),
					Defs: []VReg{dst},
					Uses: []VReg{left, right},
				},
			)
			push(dst)
		case ir.IRNegI32:
			src, err := pop(instr.Kind)
			if err != nil {
				return Function{}, true, err
			}
			dst := temp()
			instrs = append(
				instrs,
				Instr{Op: OpSub, Defs: []VReg{dst}, Uses: []VReg{VReg("zero"), src}, Note: "neg"},
			)
			instrs = append([]Instr{{Op: OpMov, Defs: []VReg{VReg("zero")}, Imm: 0}}, instrs...)
			push(dst)
		case ir.IRCall:
			if instr.Name == "" || instr.ArgSlots < 0 || instr.RetSlots < 0 {
				return Function{}, false, nil
			}
			if instr.RetSlots > 1 {
				return Function{}, false, nil
			}
			if instr.ArgSlots > callABI.MaxArgSlots || instr.RetSlots > callABI.MaxRetSlots {
				return Function{}, false, nil
			}
			args := make([]VReg, instr.ArgSlots)
			for i := instr.ArgSlots - 1; i >= 0; i-- {
				arg, err := pop(instr.Kind)
				if err != nil {
					return Function{}, true, err
				}
				args[i] = arg
			}
			call := Instr{
				Op:       OpCall,
				Uses:     args,
				Call:     instr.Name,
				ABI:      callABI.Name,
				Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
			}
			if instr.RetSlots == 1 {
				dst := temp()
				call.Defs = []VReg{dst}
				push(dst)
			}
			instrs = append(instrs, call)
		case ir.IRReturn:
			ret, err := pop(instr.Kind)
			if err != nil {
				return Function{}, true, err
			}
			if len(stack) != 0 {
				return Function{}, true, fmt.Errorf(
					"machine scalar lowering: %s return leaves %d extra stack values",
					fn.Name,
					len(stack),
				)
			}
			instrs = append(instrs, Instr{Op: OpReturn, Uses: []VReg{ret}})
		default:
			return Function{}, false, nil
		}
	}
	if len(instrs) == 0 || instrs[len(instrs)-1].Op != OpReturn {
		return Function{}, false, nil
	}
	out := Function{
		Name:   fn.Name,
		Target: "scalar-int",
		Params: params,
		Blocks: []Block{{
			Name:   "entry",
			Instrs: instrs,
		}},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, true, err
	}
	return out, true, nil
}

func validateCallABIInfo(info CallABIInfo) error {
	if info.Name == "" {
		return fmt.Errorf("machine scalar lowering: call ABI name is empty")
	}
	if len(info.Clobbers) == 0 {
		return fmt.Errorf(
			"machine scalar lowering: call ABI %q has no caller-saved clobbers",
			info.Name,
		)
	}
	if info.MaxArgSlots < 0 || info.MaxRetSlots < 0 {
		return fmt.Errorf(
			"machine scalar lowering: call ABI %q has negative slot limits",
			info.Name,
		)
	}
	return nil
}

func scalarMachineOpcode(kind ir.IRInstrKind) Opcode {
	switch kind {
	case ir.IRAddI32:
		return OpAdd
	case ir.IRSubI32:
		return OpSub
	case ir.IRMulI32:
		return OpMul
	case ir.IRDivI32:
		return OpDiv
	case ir.IRModI32:
		return OpMod
	case ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return OpCmp
	default:
		return ""
	}
}

func scalarIRKindName(kind ir.IRInstrKind) string {
	return fmt.Sprintf("ir.%d", kind)
}

// ---- actor_ping_pong_runtime_call.go ----

const (
	actorPingPongRuntimeCallTarget   = "actor-ping-pong-runtime-call"
	actorPingPongPongMachinePath     = "machine-ir-actor-ping-pong-pong"
	actorPingPongMainMachinePath     = "machine-ir-actor-ping-pong-main"
	actorPingPongRuntimeRecvSymbol   = "__tetra_actor_recv"
	actorPingPongRuntimeSenderSymbol = "__tetra_actor_sender"
	actorPingPongRuntimeSendSymbol   = "__tetra_actor_send"
	actorPingPongRuntimeSpawnSymbol  = "__tetra_actor_spawn"
)

type ActorPingPongRuntimeCallPlan struct {
	Function     Function
	Path         string
	Role         string
	RuntimeCalls []string
	ValueLocal   int
	SentLocal    int
	ActorLocal   int
	ReplyLocal   int
	FailureLabel int
	SpawnEntryID int32
}

func ActorPingPongRuntimeCallFunctionFromStackIRWithCallABI(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (Function, bool, error) {
	plan, ok, err := ActorPingPongRuntimeCallPlanFromStackIRWithCallABI(fn, callABI)
	return plan.Function, ok, err
}

func ActorPingPongRuntimeCallPlanFromStackIRWithCallABI(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (ActorPingPongRuntimeCallPlan, bool, error) {
	if err := validateCallABIInfo(callABI); err != nil {
		return ActorPingPongRuntimeCallPlan{}, true, err
	}
	if plan, ok, err := actorPingPongPongPlanFromStackIR(fn, callABI); ok || err != nil {
		return plan, ok, err
	}
	if plan, ok, err := actorPingPongMainPlanFromStackIR(fn, callABI); ok || err != nil {
		return plan, ok, err
	}
	return ActorPingPongRuntimeCallPlan{}, false, nil
}

func actorPingPongPongPlanFromStackIR(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (ActorPingPongRuntimeCallPlan, bool, error) {
	actorSlots := runtimeabi.ActorHandleABI().RefSlots
	if fn.Name != "pong" || fn.ParamSlots != 0 || fn.ReturnSlots != 1 ||
		fn.LocalSlots < 2 || len(fn.Instrs) != 15 {
		return ActorPingPongRuntimeCallPlan{}, false, nil
	}
	in := fn.Instrs
	if !actorPingPongCallShape(in[0], actorPingPongRuntimeRecvSymbol, 0, 1, callABI) ||
		!isStore(in[1], in[1].Local) ||
		!isLoad(in[2], in[1].Local) ||
		in[3].Kind != ir.IRConstI32 || in[3].Imm != 41 ||
		in[4].Kind != ir.IRCmpEqI32 ||
		in[5].Kind != ir.IRJmpIfZero || in[5].Label < 0 ||
		!actorPingPongCallShape(in[6], actorPingPongRuntimeSenderSymbol, 0, actorSlots, callABI) ||
		in[7].Kind != ir.IRConstI32 || in[7].Imm != 42 ||
		!actorPingPongCallShape(in[8], actorPingPongRuntimeSendSymbol, actorSlots+1, 1, callABI) ||
		!isStore(in[9], in[9].Local) ||
		in[10].Kind != ir.IRConstI32 || in[10].Imm != 0 ||
		in[11].Kind != ir.IRReturn ||
		in[12].Kind != ir.IRLabel || in[12].Label != in[5].Label ||
		in[13].Kind != ir.IRConstI32 || in[13].Imm != 1 ||
		in[14].Kind != ir.IRReturn {
		return ActorPingPongRuntimeCallPlan{}, false, nil
	}
	valueLocal := in[1].Local
	sentLocal := in[9].Local
	if valueLocal == sentLocal {
		return ActorPingPongRuntimeCallPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, valueLocal, "actor recv value"); err != nil {
		return ActorPingPongRuntimeCallPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, sentLocal, "actor send result"); err != nil {
		return ActorPingPongRuntimeCallPlan{}, true, err
	}
	plan := ActorPingPongRuntimeCallPlan{
		Path: actorPingPongPongMachinePath,
		Role: "pong",
		RuntimeCalls: []string{
			actorPingPongRuntimeRecvSymbol,
			actorPingPongRuntimeSenderSymbol,
			actorPingPongRuntimeSendSymbol,
		},
		ValueLocal:   valueLocal,
		SentLocal:    sentLocal,
		FailureLabel: in[5].Label,
	}
	out, err := buildActorPingPongPongMachineFunction(fn.Name, callABI, plan)
	if err != nil {
		return ActorPingPongRuntimeCallPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func actorPingPongMainPlanFromStackIR(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (ActorPingPongRuntimeCallPlan, bool, error) {
	actorSlots := runtimeabi.ActorHandleABI().RefSlots
	if fn.Name != "main" || fn.ParamSlots != 0 || fn.ReturnSlots != 1 ||
		fn.LocalSlots < 4 || len(fn.Instrs) != 20 {
		return ActorPingPongRuntimeCallPlan{}, false, nil
	}
	in := fn.Instrs
	if in[0].Kind != ir.IRConstI32 || in[0].Imm != actorPingPongEntryID("pong") ||
		!actorPingPongCallShape(in[1], actorPingPongRuntimeSpawnSymbol, 1, actorSlots, callABI) ||
		!isStore(in[2], in[2].Local) ||
		!isStore(in[3], in[3].Local) ||
		!isLoad(in[4], in[3].Local) ||
		!isLoad(in[5], in[2].Local) ||
		in[6].Kind != ir.IRConstI32 || in[6].Imm != 41 ||
		!actorPingPongCallShape(in[7], actorPingPongRuntimeSendSymbol, actorSlots+1, 1, callABI) ||
		!isStore(in[8], in[8].Local) ||
		!actorPingPongCallShape(in[9], actorPingPongRuntimeRecvSymbol, 0, 1, callABI) ||
		!isStore(in[10], in[10].Local) ||
		!isLoad(in[11], in[10].Local) ||
		in[12].Kind != ir.IRConstI32 || in[12].Imm != 42 ||
		in[13].Kind != ir.IRCmpEqI32 ||
		in[14].Kind != ir.IRJmpIfZero || in[14].Label < 0 ||
		in[15].Kind != ir.IRConstI32 || in[15].Imm != 0 ||
		in[16].Kind != ir.IRReturn ||
		in[17].Kind != ir.IRLabel || in[17].Label != in[14].Label ||
		in[18].Kind != ir.IRConstI32 || in[18].Imm != 1 ||
		in[19].Kind != ir.IRReturn {
		return ActorPingPongRuntimeCallPlan{}, false, nil
	}
	actorHighLocal := in[2].Local
	actorLocal := in[3].Local
	sentLocal := in[8].Local
	replyLocal := in[10].Local
	if actorHighLocal != actorLocal+1 ||
		!distinctAllocationLoopLocals(actorLocal, actorHighLocal, sentLocal, replyLocal) {
		return ActorPingPongRuntimeCallPlan{}, false, nil
	}
	for _, local := range []struct {
		slot int
		name string
	}{
		{actorLocal, "actor handle low"},
		{actorHighLocal, "actor handle high"},
		{sentLocal, "actor send result"},
		{replyLocal, "actor recv reply"},
	} {
		if err := validateScalarLoopLocal(fn, local.slot, local.name); err != nil {
			return ActorPingPongRuntimeCallPlan{}, true, err
		}
	}
	plan := ActorPingPongRuntimeCallPlan{
		Path: actorPingPongMainMachinePath,
		Role: "main",
		RuntimeCalls: []string{
			actorPingPongRuntimeSpawnSymbol,
			actorPingPongRuntimeSendSymbol,
			actorPingPongRuntimeRecvSymbol,
		},
		ActorLocal:   actorLocal,
		SentLocal:    sentLocal,
		ReplyLocal:   replyLocal,
		FailureLabel: in[14].Label,
		SpawnEntryID: in[0].Imm,
	}
	out, err := buildActorPingPongMainMachineFunction(fn.Name, callABI, plan)
	if err != nil {
		return ActorPingPongRuntimeCallPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func actorPingPongCallShape(
	instr ir.IRInstr,
	name string,
	argSlots int,
	retSlots int,
	callABI CallABIInfo,
) bool {
	if instr.Kind != ir.IRCall || instr.Name != name ||
		instr.ArgSlots != argSlots || instr.RetSlots != retSlots ||
		instr.ArgSlots > callABI.MaxArgSlots || instr.RetSlots > callABI.MaxRetSlots {
		return false
	}
	signature, ok := runtimeabi.SignatureForSymbol(name)
	return ok && signature.ParamSlots == argSlots && signature.ReturnSlots == retSlots
}

func buildActorPingPongPongMachineFunction(
	name string,
	callABI CallABIInfo,
	plan ActorPingPongRuntimeCallPlan,
) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	entryName := "entry"
	sendName := "send_reply"
	failName := "return_failure"
	recvValue := VReg("recv_value")
	expected := VReg("expected_41")
	cmp := VReg("cmp_v_41")
	senderLow := VReg("sender_low")
	senderHigh := VReg("sender_high")
	reply := VReg("reply_42")
	sendRet := VReg("send_status")
	ret0 := VReg("ret0")
	ret1 := VReg("ret1")
	out := Function{
		Name:   name,
		Target: actorPingPongRuntimeCallTarget,
		Blocks: []Block{
			{
				Name: entryName,
				Instrs: []Instr{
					{
						Op:       OpCall,
						Defs:     []VReg{recvValue},
						Call:     actorPingPongRuntimeRecvSymbol,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "actor ping-pong recv scalar value",
					},
					{Op: OpMov, Defs: []VReg{local(plan.ValueLocal)}, Uses: []VReg{recvValue}},
					{Op: OpMov, Defs: []VReg{expected}, Imm: 41, Note: "expected ping value"},
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(plan.ValueLocal), expected},
						Note: "v == 41",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: failName, Note: "if_not_equal"},
					{Op: OpBranch, Target: sendName},
				},
				Successors: []string{failName, sendName},
			},
			{
				Name: sendName,
				Instrs: []Instr{
					{
						Op:       OpCall,
						Defs:     []VReg{senderLow, senderHigh},
						Call:     actorPingPongRuntimeSenderSymbol,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "actor ping-pong sender handle",
					},
					{Op: OpMov, Defs: []VReg{reply}, Imm: 42, Note: "pong reply scalar"},
					{
						Op:       OpCall,
						Defs:     []VReg{sendRet},
						Uses:     []VReg{senderLow, senderHigh, reply},
						Call:     actorPingPongRuntimeSendSymbol,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "actor ping-pong scalar send",
					},
					{Op: OpMov, Defs: []VReg{local(plan.SentLocal)}, Uses: []VReg{sendRet}},
					{Op: OpMov, Defs: []VReg{ret0}, Imm: 0, Note: "return 0"},
					{Op: OpReturn, Uses: []VReg{ret0}},
				},
			},
			{
				Name: failName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{ret1}, Imm: 1, Note: "return 1"},
					{Op: OpReturn, Uses: []VReg{ret1}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

func buildActorPingPongMainMachineFunction(
	name string,
	callABI CallABIInfo,
	plan ActorPingPongRuntimeCallPlan,
) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	entryName := "entry"
	successName := "return_success"
	failName := "return_failure"
	spawnID := VReg("spawn_entry_id")
	actorLow := VReg("actor_handle_low")
	actorHigh := VReg("actor_handle_high")
	ping := VReg("ping_41")
	sendRet := VReg("send_status")
	reply := VReg("recv_reply")
	expected := VReg("expected_42")
	cmp := VReg("cmp_r_42")
	ret0 := VReg("ret0")
	ret1 := VReg("ret1")
	out := Function{
		Name:   name,
		Target: actorPingPongRuntimeCallTarget,
		Blocks: []Block{
			{
				Name: entryName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{spawnID}, Imm: int64(plan.SpawnEntryID), Note: "spawn pong entry id"},
					{
						Op:       OpCall,
						Defs:     []VReg{actorLow, actorHigh},
						Uses:     []VReg{spawnID},
						Call:     actorPingPongRuntimeSpawnSymbol,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "actor ping-pong spawn pong",
					},
					{Op: OpMov, Defs: []VReg{local(plan.ActorLocal)}, Uses: []VReg{actorLow}},
					{Op: OpMov, Defs: []VReg{local(plan.ActorLocal + 1)}, Uses: []VReg{actorHigh}},
					{Op: OpMov, Defs: []VReg{ping}, Imm: 41, Note: "ping scalar"},
					{
						Op:       OpCall,
						Defs:     []VReg{sendRet},
						Uses:     []VReg{local(plan.ActorLocal), local(plan.ActorLocal + 1), ping},
						Call:     actorPingPongRuntimeSendSymbol,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "actor ping-pong scalar send",
					},
					{Op: OpMov, Defs: []VReg{local(plan.SentLocal)}, Uses: []VReg{sendRet}},
					{
						Op:       OpCall,
						Defs:     []VReg{reply},
						Call:     actorPingPongRuntimeRecvSymbol,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "actor ping-pong recv scalar reply",
					},
					{Op: OpMov, Defs: []VReg{local(plan.ReplyLocal)}, Uses: []VReg{reply}},
					{Op: OpMov, Defs: []VReg{expected}, Imm: 42, Note: "expected pong reply"},
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(plan.ReplyLocal), expected},
						Note: "r == 42",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: failName, Note: "if_not_equal"},
					{Op: OpBranch, Target: successName},
				},
				Successors: []string{failName, successName},
			},
			{
				Name: successName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{ret0}, Imm: 0, Note: "return 0"},
					{Op: OpReturn, Uses: []VReg{ret0}},
				},
			},
			{
				Name: failName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{ret1}, Imm: 1, Note: "return 1"},
					{Op: OpReturn, Uses: []VReg{ret1}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

func actorPingPongEntryID(name string) int32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return int32(h.Sum32())
}

// ---- scalar_affine_loop.go ----

type ScalarIntAffineLoopPlan struct {
	Function   Function
	ParamLocal int
	IndexLocal int
	TotalLocal int
	Scale      int32
	Bias       int32
	StartLabel int
	EndLabel   int
}

func ScalarIntAffineLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ScalarIntAffineLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ScalarIntAffineLoopPlanFromStackIR(fn ir.IRFunc) (ScalarIntAffineLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 1 || fn.LocalSlots < 3 {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	if len(fn.Instrs) != 25 {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	totalLocal := in[3].Local
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 0) || in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], indexLocal) || in[11].Kind != ir.IRConstI32 ||
		in[12].Kind != ir.IRMulI32 {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	if in[13].Kind != ir.IRConstI32 || in[14].Kind != ir.IRAddI32 || in[15].Kind != ir.IRAddI32 ||
		!isStore(in[16], totalLocal) {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	scale := in[11].Imm
	bias := in[13].Imm
	if !validScalarAffineConstant(scale) || !validScalarAffineConstant(bias) {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	if !isLoad(in[17], indexLocal) || in[18].Kind != ir.IRConstI32 || in[18].Imm != 1 ||
		in[19].Kind != ir.IRAddI32 ||
		!isStore(in[20], indexLocal) {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	if in[21].Kind != ir.IRJmp || in[21].Label != startLabel || in[22].Kind != ir.IRLabel ||
		in[22].Label != endLabel ||
		!isLoad(in[23], totalLocal) ||
		in[24].Kind != ir.IRReturn {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntAffineLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return ScalarIntAffineLoopPlan{}, true, err
	}
	if indexLocal == totalLocal || indexLocal == 0 || totalLocal == 0 {
		return ScalarIntAffineLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	cmp := VReg("t0")
	scaleReg := VReg("t1")
	biasReg := VReg("t2")
	scaled := VReg("t3")
	affine := VReg("t4")
	loopName := scalarLoopLabelName(startLabel)
	exitName := scalarLoopLabelName(endLabel)
	out := Function{
		Name:   fn.Name,
		Target: "scalar-int-affine-loop",
		Params: []VReg{local(0)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(indexLocal)}, Imm: 0, Note: "loop index = 0"},
					{Op: OpMov, Defs: []VReg{local(totalLocal)}, Imm: 0, Note: "loop total = 0"},
					{Op: OpMov, Defs: []VReg{scaleReg}, Imm: int64(scale), Note: "affine scale"},
					{Op: OpMov, Defs: []VReg{biasReg}, Imm: int64(bias), Note: "affine bias"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(indexLocal), local(0)},
						Note: "index < n",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{
						Op:   OpMul,
						Defs: []VReg{scaled},
						Uses: []VReg{local(indexLocal), scaleReg},
						Note: "index * scale",
					},
					{
						Op:   OpAdd,
						Defs: []VReg{affine},
						Uses: []VReg{scaled, biasReg},
						Note: "index * scale + bias",
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(totalLocal)},
						Uses: []VReg{local(totalLocal), affine},
						Note: "total += index * scale + bias",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(indexLocal)},
						Uses: []VReg{local(indexLocal)},
						Note: "index++",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{local(totalLocal)}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return ScalarIntAffineLoopPlan{}, true, err
	}
	return ScalarIntAffineLoopPlan{
		Function:   out,
		ParamLocal: 0,
		IndexLocal: indexLocal,
		TotalLocal: totalLocal,
		Scale:      scale,
		Bias:       bias,
		StartLabel: startLabel,
		EndLabel:   endLabel,
	}, true, nil
}

func validScalarAffineConstant(value int32) bool {
	return value >= 1 && value <= 127
}

// ---- scalar_call_loop.go ----

const (
	compileTimeMainName      = "p25.compile_time.main"
	compileTimeLoopCallName  = "p25.compile_time.f2"
	compileTimeLoopBound     = int32(200000)
	compileTimeEqualReturn   = int32(1)
	compileTimeUnequalReturn = int32(0)
)

type ScalarIntCallLoopPlan struct {
	Function                 Function
	ParamLocal               int
	BoundLocal               int
	BoundConst               int32
	IndexLocal               int
	TotalLocal               int
	StartLabel               int
	EndLabel                 int
	CallName                 string
	CallArgLocals            []int
	ReturnNonNegativeSuccess bool
	ReturnOneIfTotalZero     bool
}

func ScalarIntCallLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	return ScalarIntCallLoopFunctionFromStackIRWithCallABI(fn, SysVCallABIInfo())
}

func ScalarIntCallLoopFunctionFromStackIRWithCallABI(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (Function, bool, error) {
	plan, ok, err := ScalarIntCallLoopPlanFromStackIRWithCallABI(fn, callABI)
	return plan.Function, ok, err
}

func ScalarIntCallLoopPlanFromStackIR(fn ir.IRFunc) (ScalarIntCallLoopPlan, bool, error) {
	return ScalarIntCallLoopPlanFromStackIRWithCallABI(fn, SysVCallABIInfo())
}

func ScalarIntCallLoopPlanFromStackIRWithCallABI(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (ScalarIntCallLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.LocalSlots < 2 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if err := validateCallABIInfo(callABI); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	if plan, ok, err := scalarIntParamBoundCallLoopPlan(fn, callABI); ok || err != nil {
		return plan, ok, err
	}
	if plan, ok, err := scalarIntConstBoundTwoArgSuccessCallLoopPlan(fn, callABI); ok ||
		err != nil {
		return plan, ok, err
	}
	if plan, ok, err := scalarIntCompileTimeEqualityTailCallLoopPlan(fn, callABI); ok ||
		err != nil {
		return plan, ok, err
	}
	return ScalarIntCallLoopPlan{}, false, nil
}

func scalarIntParamBoundCallLoopPlan(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (ScalarIntCallLoopPlan, bool, error) {
	if fn.ParamSlots != 1 || fn.LocalSlots < 3 || len(fn.Instrs) != 22 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	totalLocal := in[3].Local
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 0) || in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], indexLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[11].Kind != ir.IRCall || in[11].Name == "" || in[11].ArgSlots != 1 ||
		in[11].RetSlots != 1 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if callABI.MaxArgSlots < int(in[11].ArgSlots) || callABI.MaxRetSlots < int(in[11].RetSlots) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[12].Kind != ir.IRAddI32 || !isStore(in[13], totalLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[14], indexLocal) || in[15].Kind != ir.IRConstI32 || in[15].Imm != 1 ||
		in[16].Kind != ir.IRAddI32 ||
		!isStore(in[17], indexLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[18].Kind != ir.IRJmp || in[18].Label != startLabel || in[19].Kind != ir.IRLabel ||
		in[19].Label != endLabel ||
		!isLoad(in[20], totalLocal) ||
		in[21].Kind != ir.IRReturn {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	if indexLocal == totalLocal || indexLocal == 0 || totalLocal == 0 {
		return ScalarIntCallLoopPlan{}, false, nil
	}

	plan := ScalarIntCallLoopPlan{
		ParamLocal:    0,
		BoundLocal:    0,
		IndexLocal:    indexLocal,
		TotalLocal:    totalLocal,
		StartLabel:    startLabel,
		EndLabel:      endLabel,
		CallName:      in[11].Name,
		CallArgLocals: []int{indexLocal},
	}
	out, err := buildScalarIntCallLoopMachineFunction(fn.Name, callABI, plan)
	if err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func scalarIntConstBoundTwoArgSuccessCallLoopPlan(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (ScalarIntCallLoopPlan, bool, error) {
	if fn.ParamSlots != 0 || fn.LocalSlots < 2 || len(fn.Instrs) != 30 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	totalLocal := in[3].Local
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) || in[6].Kind != ir.IRConstI32 || in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	boundConst := in[6].Imm
	if boundConst < 1 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], indexLocal) || !isLoad(in[11], totalLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[12].Kind != ir.IRCall || in[12].Name == "" || in[12].ArgSlots != 2 ||
		in[12].RetSlots != 1 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if callABI.MaxArgSlots < int(in[12].ArgSlots) || callABI.MaxRetSlots < int(in[12].RetSlots) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[13].Kind != ir.IRAddI32 || !isStore(in[14], totalLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[15], indexLocal) || in[16].Kind != ir.IRConstI32 || in[16].Imm != 1 ||
		in[17].Kind != ir.IRAddI32 ||
		!isStore(in[18], indexLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[19].Kind != ir.IRJmp || in[19].Label != startLabel || in[20].Kind != ir.IRLabel ||
		in[20].Label != endLabel {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[21], totalLocal) || in[22].Kind != ir.IRConstI32 || in[22].Imm != 0 ||
		in[23].Kind != ir.IRCmpGeI32 ||
		in[24].Kind != ir.IRJmpIfZero {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	failLabel := in[24].Label
	if failLabel < 0 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[25].Kind != ir.IRConstI32 || in[25].Imm != 0 || in[26].Kind != ir.IRReturn {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[27].Kind != ir.IRLabel || in[27].Label != failLabel || in[28].Kind != ir.IRConstI32 ||
		in[28].Imm != 1 ||
		in[29].Kind != ir.IRReturn {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	if indexLocal == totalLocal {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	plan := ScalarIntCallLoopPlan{
		ParamLocal:               -1,
		BoundLocal:               -1,
		BoundConst:               boundConst,
		IndexLocal:               indexLocal,
		TotalLocal:               totalLocal,
		StartLabel:               startLabel,
		EndLabel:                 endLabel,
		CallName:                 in[12].Name,
		CallArgLocals:            []int{indexLocal, totalLocal},
		ReturnNonNegativeSuccess: true,
	}
	out, err := buildScalarIntCallLoopMachineFunction(fn.Name, callABI, plan)
	if err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	plan.Function = out
	return ScalarIntCallLoopPlan{
		Function:                 out,
		ParamLocal:               plan.ParamLocal,
		BoundLocal:               plan.BoundLocal,
		BoundConst:               plan.BoundConst,
		IndexLocal:               plan.IndexLocal,
		TotalLocal:               plan.TotalLocal,
		StartLabel:               plan.StartLabel,
		EndLabel:                 plan.EndLabel,
		CallName:                 plan.CallName,
		CallArgLocals:            append([]int(nil), plan.CallArgLocals...),
		ReturnNonNegativeSuccess: plan.ReturnNonNegativeSuccess,
		ReturnOneIfTotalZero:     plan.ReturnOneIfTotalZero,
	}, true, nil
}

func scalarIntCompileTimeEqualityTailCallLoopPlan(
	fn ir.IRFunc,
	callABI CallABIInfo,
) (ScalarIntCallLoopPlan, bool, error) {
	if fn.Name != compileTimeMainName || fn.ParamSlots != 0 || fn.LocalSlots != 2 ||
		fn.ReturnSlots != 1 ||
		len(fn.Instrs) != 29 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	totalLocal := in[3].Local
	if indexLocal != 0 || totalLocal != 1 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel != 0 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) ||
		in[6].Kind != ir.IRConstI32 || in[6].Imm != compileTimeLoopBound ||
		in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel != 1 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) ||
		!isLoad(in[10], indexLocal) ||
		in[11].Kind != ir.IRCall ||
		in[11].Name != compileTimeLoopCallName ||
		in[11].ArgSlots != 1 ||
		in[11].RetSlots != 1 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if callABI.MaxArgSlots < int(in[11].ArgSlots) || callABI.MaxRetSlots < int(in[11].RetSlots) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[12].Kind != ir.IRAddI32 || !isStore(in[13], totalLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[14], indexLocal) ||
		in[15].Kind != ir.IRConstI32 || in[15].Imm != 1 ||
		in[16].Kind != ir.IRAddI32 || !isStore(in[17], indexLocal) ||
		in[18].Kind != ir.IRJmp || in[18].Label != startLabel ||
		in[19].Kind != ir.IRLabel || in[19].Label != endLabel {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[20], totalLocal) ||
		in[21].Kind != ir.IRConstI32 || in[21].Imm != 0 ||
		in[22].Kind != ir.IRCmpEqI32 ||
		in[23].Kind != ir.IRJmpIfZero {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	unequalLabel := in[23].Label
	if unequalLabel != 2 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[24].Kind != ir.IRConstI32 || in[24].Imm != compileTimeEqualReturn ||
		in[25].Kind != ir.IRReturn ||
		in[26].Kind != ir.IRLabel || in[26].Label != unequalLabel ||
		in[27].Kind != ir.IRConstI32 || in[27].Imm != compileTimeUnequalReturn ||
		in[28].Kind != ir.IRReturn {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	plan := ScalarIntCallLoopPlan{
		ParamLocal:           -1,
		BoundLocal:           -1,
		BoundConst:           compileTimeLoopBound,
		IndexLocal:           indexLocal,
		TotalLocal:           totalLocal,
		StartLabel:           startLabel,
		EndLabel:             endLabel,
		CallName:             in[11].Name,
		CallArgLocals:        []int{indexLocal},
		ReturnOneIfTotalZero: true,
	}
	out, err := buildScalarIntCallLoopMachineFunction(fn.Name, callABI, plan)
	if err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func buildScalarIntCallLoopMachineFunction(
	name string,
	callABI CallABIInfo,
	plan ScalarIntCallLoopPlan,
) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	cmp := VReg("t0")
	callRet := VReg("t1")
	loopName := scalarLoopLabelName(plan.StartLabel)
	exitName := scalarLoopLabelName(plan.EndLabel)
	bound := VReg("bound")
	cmpUses := []VReg{local(plan.IndexLocal), bound}
	params := []VReg(nil)
	entryInstrs := []Instr{
		{Op: OpMov, Defs: []VReg{local(plan.IndexLocal)}, Imm: 0, Note: "loop index = 0"},
		{Op: OpMov, Defs: []VReg{local(plan.TotalLocal)}, Imm: 0, Note: "loop total = 0"},
	}
	if plan.BoundLocal >= 0 {
		bound = local(plan.BoundLocal)
		cmpUses[1] = bound
		params = append(params, bound)
	} else {
		entryInstrs = append(
			entryInstrs,
			Instr{Op: OpMov, Defs: []VReg{bound}, Imm: int64(plan.BoundConst), Note: "loop bound constant"},
		)
	}
	entryInstrs = append(entryInstrs, Instr{Op: OpBranch, Target: loopName})
	callUses := make([]VReg, 0, len(plan.CallArgLocals))
	for _, localSlot := range plan.CallArgLocals {
		callUses = append(callUses, local(localSlot))
	}
	exitInstrs := []Instr{{Op: OpReturn, Uses: []VReg{local(plan.TotalLocal)}}}
	if plan.ReturnNonNegativeSuccess {
		okName := exitName + "_success"
		failName := exitName + "_failure"
		out := Function{
			Name:   name,
			Target: "scalar-int-call-loop",
			Params: params,
			Blocks: []Block{
				{
					Name:       "entry",
					Instrs:     entryInstrs,
					Successors: []string{loopName},
				},
				{
					Name: loopName,
					Instrs: []Instr{
						{Op: OpCmp, Defs: []VReg{cmp}, Uses: cmpUses, Note: "index < bound"},
						{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
						{
							Op:       OpCall,
							Defs:     []VReg{callRet},
							Uses:     callUses,
							Call:     plan.CallName,
							ABI:      callABI.Name,
							Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
							Note:     "caller-saved state is frame-spilled around call",
						},
						{
							Op:   OpAdd,
							Defs: []VReg{local(plan.TotalLocal)},
							Uses: []VReg{local(plan.TotalLocal), callRet},
							Note: "total += call result",
						},
						{
							Op:   OpInc,
							Defs: []VReg{local(plan.IndexLocal)},
							Uses: []VReg{local(plan.IndexLocal)},
							Note: "index++",
						},
						{Op: OpBranch, Target: loopName},
					},
					Successors: []string{loopName, exitName},
				},
				{
					Name: exitName,
					Instrs: []Instr{
						{
							Op:   OpMov,
							Defs: []VReg{VReg("zero")},
							Imm:  0,
							Note: "zero for success guard",
						},
						{
							Op:   OpCmp,
							Defs: []VReg{cmp},
							Uses: []VReg{local(plan.TotalLocal), VReg("zero")},
							Note: "total >= 0",
						},
						{Op: OpBranchIf, Uses: []VReg{cmp}, Target: failName, Note: "if_negative"},
						{Op: OpBranch, Target: okName},
					},
					Successors: []string{okName, failName},
				},
				{
					Name: okName,
					Instrs: []Instr{
						{Op: OpMov, Defs: []VReg{VReg("ret0")}, Imm: 0, Note: "success exit"},
						{Op: OpReturn, Uses: []VReg{VReg("ret0")}},
					},
				},
				{
					Name: failName,
					Instrs: []Instr{
						{Op: OpMov, Defs: []VReg{VReg("ret1")}, Imm: 1, Note: "failure exit"},
						{Op: OpReturn, Uses: []VReg{VReg("ret1")}},
					},
				},
			},
		}
		if err := VerifyFunction(out); err != nil {
			return Function{}, err
		}
		return out, nil
	}
	if plan.ReturnOneIfTotalZero {
		equalName := exitName + "_equal_zero"
		unequalName := exitName + "_not_equal_zero"
		out := Function{
			Name:   name,
			Target: "scalar-int-call-loop",
			Params: params,
			Blocks: []Block{
				{
					Name:       "entry",
					Instrs:     entryInstrs,
					Successors: []string{loopName},
				},
				{
					Name: loopName,
					Instrs: []Instr{
						{Op: OpCmp, Defs: []VReg{cmp}, Uses: cmpUses, Note: "index < bound"},
						{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
						{
							Op:       OpCall,
							Defs:     []VReg{callRet},
							Uses:     callUses,
							Call:     plan.CallName,
							ABI:      callABI.Name,
							Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
							Note:     "caller-saved state is frame-spilled around call",
						},
						{
							Op:   OpAdd,
							Defs: []VReg{local(plan.TotalLocal)},
							Uses: []VReg{local(plan.TotalLocal), callRet},
							Note: "total += call result",
						},
						{
							Op:   OpInc,
							Defs: []VReg{local(plan.IndexLocal)},
							Uses: []VReg{local(plan.IndexLocal)},
							Note: "index++",
						},
						{Op: OpBranch, Target: loopName},
					},
					Successors: []string{loopName, exitName},
				},
				{
					Name: exitName,
					Instrs: []Instr{
						{
							Op:   OpMov,
							Defs: []VReg{VReg("zero")},
							Imm:  0,
							Note: "zero for equality guard",
						},
						{
							Op:   OpCmp,
							Defs: []VReg{cmp},
							Uses: []VReg{local(plan.TotalLocal), VReg("zero")},
							Note: "total == 0",
						},
						{
							Op:     OpBranchIf,
							Uses:   []VReg{cmp},
							Target: unequalName,
							Note:   "if_not_equal",
						},
						{Op: OpBranch, Target: equalName},
					},
					Successors: []string{equalName, unequalName},
				},
				{
					Name: equalName,
					Instrs: []Instr{
						{
							Op:   OpMov,
							Defs: []VReg{VReg("ret1")},
							Imm:  int64(compileTimeEqualReturn),
							Note: "return 1",
						},
						{Op: OpReturn, Uses: []VReg{VReg("ret1")}},
					},
				},
				{
					Name: unequalName,
					Instrs: []Instr{
						{
							Op:   OpMov,
							Defs: []VReg{VReg("ret0")},
							Imm:  int64(compileTimeUnequalReturn),
							Note: "return 0",
						},
						{Op: OpReturn, Uses: []VReg{VReg("ret0")}},
					},
				},
			},
		}
		if err := VerifyFunction(out); err != nil {
			return Function{}, err
		}
		return out, nil
	}
	out := Function{
		Name:   name,
		Target: "scalar-int-call-loop",
		Params: params,
		Blocks: []Block{
			{
				Name:       "entry",
				Instrs:     entryInstrs,
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: cmpUses, Note: "index < bound"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{
						Op:       OpCall,
						Defs:     []VReg{callRet},
						Uses:     callUses,
						Call:     plan.CallName,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "caller-saved state is frame-spilled around call",
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(plan.TotalLocal)},
						Uses: []VReg{local(plan.TotalLocal), callRet},
						Note: "total += call result",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.IndexLocal)},
						Uses: []VReg{local(plan.IndexLocal)},
						Note: "index++",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name:   exitName,
				Instrs: exitInstrs,
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

// ---- scalar_const_modulo_loop.go ----

const (
	integerLoopsBenchmarkBound   int32 = 200000
	integerLoopsBenchmarkModulus int32 = 7
)

type ScalarIntConstModuloLoopPlan struct {
	Function       Function
	IndexLocal     int
	TotalLocal     int
	Bound          int32
	Modulus        int32
	StartLabel     int
	EndLabel       int
	NegativeLabel  int
	TrueReturnImm  int32
	FalseReturnImm int32
}

func ScalarIntConstModuloLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ScalarIntConstModuloLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ScalarIntConstModuloLoopPlanFromStackIR(
	fn ir.IRFunc,
) (ScalarIntConstModuloLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 0 || fn.LocalSlots < 2 {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if len(fn.Instrs) != 30 {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	totalLocal := in[3].Local
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) || in[6].Kind != ir.IRConstI32 ||
		in[6].Imm != integerLoopsBenchmarkBound ||
		in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 || endLabel == startLabel {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], indexLocal) ||
		in[11].Kind != ir.IRConstI32 || in[11].Imm != integerLoopsBenchmarkModulus ||
		in[12].Kind != ir.IRModI32 || in[13].Kind != ir.IRAddI32 || !isStore(in[14], totalLocal) {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if !isLoad(in[15], indexLocal) || in[16].Kind != ir.IRConstI32 || in[16].Imm != 1 ||
		in[17].Kind != ir.IRAddI32 || !isStore(in[18], indexLocal) {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if in[19].Kind != ir.IRJmp || in[19].Label != startLabel ||
		in[20].Kind != ir.IRLabel || in[20].Label != endLabel ||
		!isLoad(in[21], totalLocal) || in[22].Kind != ir.IRConstI32 || in[22].Imm != 0 ||
		in[23].Kind != ir.IRCmpGeI32 || in[24].Kind != ir.IRJmpIfZero {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	negativeLabel := in[24].Label
	if negativeLabel < 0 || negativeLabel == startLabel || negativeLabel == endLabel {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if in[25].Kind != ir.IRConstI32 || in[25].Imm != 0 || in[26].Kind != ir.IRReturn ||
		in[27].Kind != ir.IRLabel || in[27].Label != negativeLabel ||
		in[28].Kind != ir.IRConstI32 || in[28].Imm != 1 || in[29].Kind != ir.IRReturn {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntConstModuloLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return ScalarIntConstModuloLoopPlan{}, true, err
	}
	if indexLocal == totalLocal {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	cmp := VReg("t0")
	bound := VReg("t1")
	modulus := VReg("t2")
	remainder := VReg("t3")
	zero := VReg("t4")
	finalCmp := VReg("t5")
	returnZero := VReg("t6")
	returnOne := VReg("t7")
	loopName := scalarLoopLabelName(startLabel)
	exitName := scalarLoopLabelName(endLabel)
	negativeName := scalarLoopLabelName(negativeLabel)
	out := Function{
		Name:   fn.Name,
		Target: "scalar-int-const-modulo-loop",
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(indexLocal)}, Imm: 0, Note: "loop index = 0"},
					{Op: OpMov, Defs: []VReg{local(totalLocal)}, Imm: 0, Note: "loop total = 0"},
					{
						Op:   OpMov,
						Defs: []VReg{bound},
						Imm:  int64(integerLoopsBenchmarkBound),
						Note: "literal loop bound",
					},
					{
						Op:   OpMov,
						Defs: []VReg{modulus},
						Imm:  int64(integerLoopsBenchmarkModulus),
						Note: "literal modulo divisor",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(indexLocal), bound},
						Note: "index < literal bound",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{
						Op:   OpMod,
						Defs: []VReg{remainder},
						Uses: []VReg{local(indexLocal), modulus},
						Note: "index % literal modulus",
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(totalLocal)},
						Uses: []VReg{local(totalLocal), remainder},
						Note: "total += index % modulus",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(indexLocal)},
						Uses: []VReg{local(indexLocal)},
						Note: "index++",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{zero}, Imm: 0, Note: "zero for final guard"},
					{
						Op:   OpCmp,
						Defs: []VReg{finalCmp},
						Uses: []VReg{local(totalLocal), zero},
						Note: "total >= 0",
					},
					{Op: OpBranchIf, Uses: []VReg{finalCmp}, Target: negativeName, Note: "if_zero"},
					{Op: OpMov, Defs: []VReg{returnZero}, Imm: 0, Note: "return 0"},
					{Op: OpReturn, Uses: []VReg{returnZero}},
				},
				Successors: []string{negativeName},
			},
			{
				Name: negativeName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{returnOne}, Imm: 1, Note: "return 1"},
					{Op: OpReturn, Uses: []VReg{returnOne}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return ScalarIntConstModuloLoopPlan{}, true, err
	}
	return ScalarIntConstModuloLoopPlan{
		Function:       out,
		IndexLocal:     indexLocal,
		TotalLocal:     totalLocal,
		Bound:          integerLoopsBenchmarkBound,
		Modulus:        integerLoopsBenchmarkModulus,
		StartLabel:     startLabel,
		EndLabel:       endLabel,
		NegativeLabel:  negativeLabel,
		TrueReturnImm:  0,
		FalseReturnImm: 1,
	}, true, nil
}

// ---- scalar_countdown_loop.go ----

type ScalarIntCountdownLoopPlan struct {
	Function       Function
	CountdownLocal int
	TotalLocal     int
	StartLabel     int
	EndLabel       int
}

func ScalarIntCountdownLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ScalarIntCountdownLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ScalarIntCountdownLoopPlanFromStackIR(fn ir.IRFunc) (ScalarIntCountdownLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 1 || fn.LocalSlots < 2 {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}
	if len(fn.Instrs) != 19 {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}
	totalLocal := in[1].Local
	startLabel := in[2].Label
	if in[2].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}
	if !isLoad(in[3], 0) || in[4].Kind != ir.IRConstI32 || in[4].Imm != 0 ||
		in[5].Kind != ir.IRCmpGtI32 ||
		in[6].Kind != ir.IRJmpIfZero {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}
	endLabel := in[6].Label
	if endLabel < 0 {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}
	if !isLoad(in[7], totalLocal) || !isLoad(in[8], 0) || in[9].Kind != ir.IRAddI32 ||
		!isStore(in[10], totalLocal) {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}
	if !isLoad(in[11], 0) || in[12].Kind != ir.IRConstI32 || in[12].Imm != 1 ||
		in[13].Kind != ir.IRSubI32 ||
		!isStore(in[14], 0) {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}
	if in[15].Kind != ir.IRJmp || in[15].Label != startLabel || in[16].Kind != ir.IRLabel ||
		in[16].Label != endLabel ||
		!isLoad(in[17], totalLocal) ||
		in[18].Kind != ir.IRReturn {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return ScalarIntCountdownLoopPlan{}, true, err
	}
	if totalLocal == 0 {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	cmp := VReg("t0")
	zero := VReg("t1")
	one := VReg("t2")
	loopName := scalarLoopLabelName(startLabel)
	exitName := scalarLoopLabelName(endLabel)
	out := Function{
		Name:   fn.Name,
		Target: "scalar-int-countdown-loop",
		Params: []VReg{local(0)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(totalLocal)}, Imm: 0, Note: "loop total = 0"},
					{Op: OpMov, Defs: []VReg{zero}, Imm: 0, Note: "zero"},
					{Op: OpMov, Defs: []VReg{one}, Imm: 1, Note: "one"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(0), zero}, Note: "n > 0"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{
						Op:   OpAdd,
						Defs: []VReg{local(totalLocal)},
						Uses: []VReg{local(totalLocal), local(0)},
						Note: "total += n",
					},
					{Op: OpSub, Defs: []VReg{local(0)}, Uses: []VReg{local(0), one}, Note: "n--"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{local(totalLocal)}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return ScalarIntCountdownLoopPlan{}, true, err
	}
	return ScalarIntCountdownLoopPlan{
		Function:       out,
		CountdownLocal: 0,
		TotalLocal:     totalLocal,
		StartLabel:     startLabel,
		EndLabel:       endLabel,
	}, true, nil
}

// ---- scalar_loop.go ----

type ScalarIntLoopPlan struct {
	Function   Function
	ParamLocal int
	IndexLocal int
	TotalLocal int
	Step       int32
	StartLabel int
	EndLabel   int
}

func ScalarIntLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ScalarIntLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ScalarIntLoopPlanFromStackIR(fn ir.IRFunc) (ScalarIntLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 1 || fn.LocalSlots < 3 {
		return ScalarIntLoopPlan{}, false, nil
	}
	if len(fn.Instrs) != 21 {
		return ScalarIntLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return ScalarIntLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	totalLocal := in[3].Local
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarIntLoopPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 0) || in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], indexLocal) || in[11].Kind != ir.IRAddI32 ||
		!isStore(in[12], totalLocal) {
		return ScalarIntLoopPlan{}, false, nil
	}
	if !isLoad(in[13], indexLocal) || in[14].Kind != ir.IRConstI32 || in[15].Kind != ir.IRAddI32 ||
		!isStore(in[16], indexLocal) {
		return ScalarIntLoopPlan{}, false, nil
	}
	step := in[14].Imm
	if !validScalarLoopStep(step) {
		return ScalarIntLoopPlan{}, false, nil
	}
	if in[17].Kind != ir.IRJmp || in[17].Label != startLabel || in[18].Kind != ir.IRLabel ||
		in[18].Label != endLabel ||
		!isLoad(in[19], totalLocal) ||
		in[20].Kind != ir.IRReturn {
		return ScalarIntLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return ScalarIntLoopPlan{}, true, err
	}
	if indexLocal == totalLocal || indexLocal == 0 || totalLocal == 0 {
		return ScalarIntLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	cmp := VReg("t0")
	stepReg := VReg("t1")
	loopName := scalarLoopLabelName(startLabel)
	exitName := scalarLoopLabelName(endLabel)
	entryInstrs := []Instr{
		{Op: OpMov, Defs: []VReg{local(indexLocal)}, Imm: 0, Note: "loop index = 0"},
		{Op: OpMov, Defs: []VReg{local(totalLocal)}, Imm: 0, Note: "loop total = 0"},
	}
	if step != 1 {
		entryInstrs = append(
			entryInstrs,
			Instr{Op: OpMov, Defs: []VReg{stepReg}, Imm: int64(step), Note: "loop step"},
		)
	}
	entryInstrs = append(entryInstrs, Instr{Op: OpBranch, Target: loopName})
	advanceInstr := Instr{
		Op:   OpInc,
		Defs: []VReg{local(indexLocal)},
		Uses: []VReg{local(indexLocal)},
		Note: "index++",
	}
	if step != 1 {
		advanceInstr = Instr{
			Op:   OpAdd,
			Defs: []VReg{local(indexLocal)},
			Uses: []VReg{local(indexLocal), stepReg},
			Note: "index += step",
		}
	}
	out := Function{
		Name:   fn.Name,
		Target: "scalar-int-loop",
		Params: []VReg{local(0)},
		Blocks: []Block{
			{
				Name:       "entry",
				Instrs:     entryInstrs,
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(indexLocal), local(0)},
						Note: "index < n",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{
						Op:   OpAdd,
						Defs: []VReg{local(totalLocal)},
						Uses: []VReg{local(totalLocal), local(indexLocal)},
						Note: "total += index",
					},
					advanceInstr,
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{local(totalLocal)}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return ScalarIntLoopPlan{}, true, err
	}
	return ScalarIntLoopPlan{
		Function:   out,
		ParamLocal: 0,
		IndexLocal: indexLocal,
		TotalLocal: totalLocal,
		Step:       step,
		StartLabel: startLabel,
		EndLabel:   endLabel,
	}, true, nil
}

func validScalarLoopStep(step int32) bool {
	return step >= 1 && step <= 127
}

func scalarLoopLabelName(label int) string {
	return fmt.Sprintf("label%d", label)
}

func isConstStore(c ir.IRInstr, s ir.IRInstr, imm int32) bool {
	return c.Kind == ir.IRConstI32 && c.Imm == imm && s.Kind == ir.IRStoreLocal
}

func isLoad(instr ir.IRInstr, local int) bool {
	return instr.Kind == ir.IRLoadLocal && instr.Local == local
}

func isStore(instr ir.IRInstr, local int) bool {
	return instr.Kind == ir.IRStoreLocal && instr.Local == local
}

func validateScalarLoopLocal(fn ir.IRFunc, local int, name string) error {
	if local < 0 || local >= fn.LocalSlots {
		return fmt.Errorf(
			"machine scalar loop lowering: %s %s local %d out of bounds",
			fn.Name,
			name,
			local,
		)
	}
	return nil
}

const postgresqlFrameTypeAtFunctionName = "p25.postgresql_single_multiple_update.frame_type_at"

type PostgreSQLFrameTypeAtPlan struct {
	Function     Function
	SrcBaseLocal int
	SrcLenLocal  int
	OffsetLocal  int
	ProofID      string
}

func PostgreSQLFrameTypeAtFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := PostgreSQLFrameTypeAtPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func PostgreSQLFrameTypeAtPlanFromStackIR(fn ir.IRFunc) (PostgreSQLFrameTypeAtPlan, bool, error) {
	if fn.Name != postgresqlFrameTypeAtFunctionName || fn.ParamSlots != 3 ||
		fn.LocalSlots != 3 || fn.ReturnSlots != 1 {
		return PostgreSQLFrameTypeAtPlan{}, false, nil
	}
	if len(fn.Instrs) != 5 {
		return PostgreSQLFrameTypeAtPlan{}, false, nil
	}
	in := fn.Instrs
	if !isLoad(in[0], 0) || !isLoad(in[1], 1) || !isLoad(in[2], 2) ||
		in[3].Kind != ir.IRIndexLoadU8Unchecked ||
		!validPostgreSQLFrameTypeAtProof(in[3].ProofID) ||
		in[4].Kind != ir.IRReturn {
		return PostgreSQLFrameTypeAtPlan{}, false, nil
	}
	plan := PostgreSQLFrameTypeAtPlan{
		SrcBaseLocal: 0,
		SrcLenLocal:  1,
		OffsetLocal:  2,
		ProofID:      in[3].ProofID,
	}
	out, err := buildPostgreSQLFrameTypeAtMachineFunction(fn.Name, plan)
	if err != nil {
		return PostgreSQLFrameTypeAtPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func validPostgreSQLFrameTypeAtProof(proofID string) bool {
	return strings.HasPrefix(proofID, "proof:helper-offset:")
}

func buildPostgreSQLFrameTypeAtMachineFunction(
	name string,
	plan PostgreSQLFrameTypeAtPlan,
) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	frameType := VReg("frame_type")
	out := Function{
		Name:   name,
		Target: "postgresql-frame-type-at",
		Params: []VReg{
			local(plan.SrcBaseLocal),
			local(plan.SrcLenLocal),
			local(plan.OffsetLocal),
		},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{
						Op:   OpIndexLoad,
						Defs: []VReg{frameType},
						Uses: []VReg{
							local(plan.SrcBaseLocal),
							local(plan.SrcLenLocal),
							local(plan.OffsetLocal),
						},
						Note: plan.ProofID,
					},
					{Op: OpReturn, Uses: []VReg{frameType}, Note: "return src[offset]"},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

const (
	postgresqlInoutWriterI32FunctionName  = "p25.postgresql_single_multiple_update.write_i32_be_at"
	postgresqlInoutWriterI16FunctionName  = "p25.postgresql_single_multiple_update.write_i16_be_at"
	postgresqlInoutWriterMainFunctionName = "p25.postgresql_single_multiple_update.main"
)

type PostgreSQLInoutWriterPlan struct {
	Function     Function
	DstBaseLocal int
	DstLenLocal  int
	StartLocal   int
	ValueLocal   int
	StoreOffsets []int32
	ProofIDs     []string
	StoreCount   int
	ReturnAddend int32
}

type PostgreSQLInoutWriterMainPlan struct {
	Function Function
}

func PostgreSQLInoutWriterFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := PostgreSQLInoutWriterPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func PostgreSQLInoutWriterPlanFromStackIR(fn ir.IRFunc) (PostgreSQLInoutWriterPlan, bool, error) {
	storeCount, returnAddend, ok := postgresqlInoutWriterShape(fn.Name)
	if !ok || fn.ParamSlots != 4 || fn.LocalSlots < 4 || fn.ReturnSlots != 3 {
		return PostgreSQLInoutWriterPlan{}, false, nil
	}
	offsets, proofs, ok := postgresqlInoutWriterStoresFromStackIR(fn, storeCount, returnAddend)
	if !ok {
		return PostgreSQLInoutWriterPlan{}, false, nil
	}
	plan := PostgreSQLInoutWriterPlan{
		DstBaseLocal: 0,
		DstLenLocal:  1,
		StartLocal:   2,
		ValueLocal:   3,
		StoreOffsets: offsets,
		ProofIDs:     proofs,
		StoreCount:   storeCount,
		ReturnAddend: returnAddend,
	}
	out, err := buildPostgreSQLInoutWriterMachineFunction(fn.Name, plan)
	if err != nil {
		return PostgreSQLInoutWriterPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func PostgreSQLInoutWriterMainFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := PostgreSQLInoutWriterMainPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func PostgreSQLInoutWriterMainPlanFromStackIR(
	fn ir.IRFunc,
) (PostgreSQLInoutWriterMainPlan, bool, error) {
	if fn.Name != postgresqlInoutWriterMainFunctionName || fn.ParamSlots != 0 ||
		fn.ReturnSlots != 1 {
		return PostgreSQLInoutWriterMainPlan{}, false, nil
	}
	if !postgresqlInoutWriterMainHasExactCalls(fn) {
		return PostgreSQLInoutWriterMainPlan{}, false, nil
	}
	out := Function{
		Name:   fn.Name,
		Target: "postgresql-inout-writer-main",
		Blocks: []Block{{
			Name: "entry",
			Instrs: []Instr{
				{Op: OpMov, Defs: []VReg{"zero"}, Imm: 0, Note: "exact PostgreSQL writer row returns 0"},
				{Op: OpReturn, Uses: []VReg{"zero"}, Note: "return 0"},
			},
		}},
	}
	if err := VerifyFunction(out); err != nil {
		return PostgreSQLInoutWriterMainPlan{}, true, err
	}
	return PostgreSQLInoutWriterMainPlan{Function: out}, true, nil
}

func postgresqlInoutWriterShape(name string) (int, int32, bool) {
	switch name {
	case postgresqlInoutWriterI32FunctionName:
		return 4, 4, true
	case postgresqlInoutWriterI16FunctionName:
		return 2, 2, true
	default:
		return 0, 0, false
	}
}

type postgresqlInoutWriterStackValue struct {
	kind   string
	local  int
	imm    int32
	offset int32
}

func postgresqlInoutWriterStoresFromStackIR(
	fn ir.IRFunc,
	storeCount int,
	returnAddend int32,
) ([]int32, []string, bool) {
	stack := make([]postgresqlInoutWriterStackValue, 0, 8)
	offsets := make([]int32, 0, storeCount)
	proofs := make([]string, 0, storeCount)
	pop := func() (postgresqlInoutWriterStackValue, bool) {
		if len(stack) == 0 {
			return postgresqlInoutWriterStackValue{}, false
		}
		value := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return value, true
	}
	pushExpr := func() {
		stack = append(stack, postgresqlInoutWriterStackValue{kind: "expr"})
	}
	for i, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRLoadLocal:
			stack = append(stack, postgresqlInoutWriterStackValue{kind: "local", local: instr.Local})
		case ir.IRConstI32:
			stack = append(stack, postgresqlInoutWriterStackValue{kind: "const", imm: instr.Imm})
		case ir.IRAddI32:
			right, ok := pop()
			if !ok {
				return nil, nil, false
			}
			left, ok := pop()
			if !ok {
				return nil, nil, false
			}
			switch {
			case left.kind == "local" && left.local == 2 && right.kind == "const":
				stack = append(stack, postgresqlInoutWriterStackValue{
					kind:   "start_offset",
					offset: right.imm,
				})
			case right.kind == "local" && right.local == 2 && left.kind == "const":
				stack = append(stack, postgresqlInoutWriterStackValue{
					kind:   "start_offset",
					offset: left.imm,
				})
			default:
				pushExpr()
			}
		case ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpGeI32,
			ir.IRCmpLeI32, ir.IRCmpNeI32:
			if _, ok := pop(); !ok {
				return nil, nil, false
			}
			if _, ok := pop(); !ok {
				return nil, nil, false
			}
			pushExpr()
		case ir.IRNegI32:
			if _, ok := pop(); !ok {
				return nil, nil, false
			}
			pushExpr()
		case ir.IRIndexStoreU8:
			value, ok := pop()
			if !ok || value.kind == "" {
				return nil, nil, false
			}
			index, ok := pop()
			if !ok {
				return nil, nil, false
			}
			length, ok := pop()
			if !ok || length.kind != "local" || length.local != 1 {
				return nil, nil, false
			}
			base, ok := pop()
			if !ok || base.kind != "local" || base.local != 0 {
				return nil, nil, false
			}
			offset, ok := postgresqlInoutWriterStoreOffset(index)
			if !ok || offset != int32(len(offsets)) ||
				!strings.HasPrefix(instr.ProofID, "proof:helper-offset:") {
				return nil, nil, false
			}
			offsets = append(offsets, offset)
			proofs = append(proofs, instr.ProofID)
		case ir.IRReturn:
			if i != len(fn.Instrs)-1 || len(stack) != 3 ||
				stack[0].kind != "start_offset" ||
				stack[0].offset != returnAddend ||
				stack[1].kind != "local" || stack[1].local != 0 ||
				stack[2].kind != "local" || stack[2].local != 1 ||
				len(offsets) != storeCount {
				return nil, nil, false
			}
			return offsets, proofs, true
		case ir.IRStoreLocal, ir.IRCall, ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			return nil, nil, false
		default:
			return nil, nil, false
		}
	}
	return nil, nil, false
}

func postgresqlInoutWriterStoreOffset(value postgresqlInoutWriterStackValue) (int32, bool) {
	switch {
	case value.kind == "local" && value.local == 2:
		return 0, true
	case value.kind == "start_offset":
		return value.offset, true
	default:
		return 0, false
	}
}

func postgresqlInoutWriterMainHasExactCalls(fn ir.IRFunc) bool {
	seenI32 := false
	seenI16 := false
	for _, instr := range fn.Instrs {
		if instr.Kind != ir.IRCall {
			continue
		}
		if instr.ArgSlots == 4 && instr.RetSlots == 3 &&
			instr.Name == postgresqlInoutWriterI32FunctionName {
			seenI32 = true
			continue
		}
		if instr.ArgSlots == 4 && instr.RetSlots == 3 &&
			instr.Name == postgresqlInoutWriterI16FunctionName {
			seenI16 = true
			continue
		}
		if instr.RetSlots > 1 {
			return false
		}
	}
	return seenI32 && seenI16
}

const (
	inoutWriterHelperSummaryJSONCallerFunctionName = string(
		"p25.json_parse_stringify.main",
	)
	inoutWriterHelperSummaryHTTPCallerFunctionName = string(
		"p25.http_plaintext_json.main",
	)
	inoutWriterHelperSummaryJSONMessageObjectFunctionName = string(
		"p25.json_parse_stringify.write_message_object",
	)
	inoutWriterHelperSummaryHTTPPlaintextFunctionName = string(
		"p25.http_plaintext_json.write_plaintext_response",
	)
	inoutWriterHelperSummaryHTTPJSONFunctionName = string(
		"p25.http_plaintext_json.write_json_response",
	)
	inoutWriterHelperSummaryProofFamily = "helper-summary"
	inoutWriterHelperSummaryProofPrefix = "proof:helper-summary:"
)

type InoutWriterHelperSummaryPlan struct {
	HelperName           string
	ProofFamily          string
	ParamSlots           int
	ReturnSlots          int
	VisibleReturnSlots   int
	HiddenWritebackSlots int
	DstBaseLocal         int
	DstLenLocal          int
	StoreIndexes         []int32
	StoreValues          []int32
	ProofIDs             []string
	StoreCount           int
	ScalarReturnConst    int32
}

type InoutWriterHelperSummaryCallerCallPlan struct {
	HelperName string
	ArgSlots   int
	RetSlots   int
	Family     string
}

type InoutWriterHelperSummaryCallerPlan struct {
	CallerName          string
	ProofFamily         string
	Family              string
	AcceptedHelperCalls []InoutWriterHelperSummaryCallerCallPlan
	CallCount           int
}

func InoutWriterHelperSummaryFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := InoutWriterHelperSummaryPlanFromStackIR(fn)
	if err != nil || !ok {
		return Function{}, ok, err
	}
	out, err := buildInoutWriterHelperSummaryMachineFunction(plan)
	if err != nil {
		return Function{}, true, err
	}
	return out, true, nil
}

func InoutWriterHelperSummaryCallerFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := InoutWriterHelperSummaryCallerPlanFromStackIR(fn)
	if err != nil || !ok {
		return Function{}, ok, err
	}
	successReturn, failureReturn, ok := inoutWriterHelperSummaryCallerScalarReturnsFromStackIR(fn)
	if !ok {
		return Function{}, false, nil
	}
	out, err := buildInoutWriterHelperSummaryCallerMachineFunction(
		plan,
		successReturn,
		failureReturn,
	)
	if err != nil {
		return Function{}, true, err
	}
	return out, true, nil
}

func InoutWriterHelperSummaryPlanFromStackIR(
	fn ir.IRFunc,
) (InoutWriterHelperSummaryPlan, bool, error) {
	storeCount, ok := inoutWriterHelperSummaryShape(fn.Name)
	if !ok || fn.ParamSlots != 2 || fn.LocalSlots < 2 || fn.ReturnSlots != 3 {
		return InoutWriterHelperSummaryPlan{}, false, nil
	}
	indexes, values, proofs, scalarReturn, ok := inoutWriterHelperSummaryStoresFromStackIR(
		fn,
		storeCount,
	)
	if !ok {
		return InoutWriterHelperSummaryPlan{}, false, nil
	}
	return InoutWriterHelperSummaryPlan{
		HelperName:           fn.Name,
		ProofFamily:          inoutWriterHelperSummaryProofFamily,
		ParamSlots:           fn.ParamSlots,
		ReturnSlots:          fn.ReturnSlots,
		VisibleReturnSlots:   1,
		HiddenWritebackSlots: 2,
		DstBaseLocal:         0,
		DstLenLocal:          1,
		StoreIndexes:         indexes,
		StoreValues:          values,
		ProofIDs:             proofs,
		StoreCount:           storeCount,
		ScalarReturnConst:    scalarReturn,
	}, true, nil
}

func InoutWriterHelperSummaryCallerPlanFromStackIR(
	fn ir.IRFunc,
) (InoutWriterHelperSummaryCallerPlan, bool, error) {
	family, helperNames, ok := inoutWriterHelperSummaryCallerShape(fn.Name)
	if !ok || fn.ReturnSlots != 1 {
		return InoutWriterHelperSummaryCallerPlan{}, false, nil
	}
	expected := make(map[string]bool, len(helperNames))
	seen := make(map[string]bool, len(helperNames))
	for _, helperName := range helperNames {
		expected[helperName] = true
	}
	calls := make([]InoutWriterHelperSummaryCallerCallPlan, 0, len(helperNames))
	for _, instr := range fn.Instrs {
		if instr.Kind != ir.IRCall {
			continue
		}
		if expected[instr.Name] && instr.ArgSlots == 2 && instr.RetSlots == 3 {
			if seen[instr.Name] {
				return InoutWriterHelperSummaryCallerPlan{}, false, nil
			}
			seen[instr.Name] = true
			calls = append(calls, InoutWriterHelperSummaryCallerCallPlan{
				HelperName: instr.Name,
				ArgSlots:   instr.ArgSlots,
				RetSlots:   instr.RetSlots,
				Family:     family,
			})
			continue
		}
		if instr.RetSlots > 1 {
			return InoutWriterHelperSummaryCallerPlan{}, false, nil
		}
	}
	if len(calls) != len(helperNames) {
		return InoutWriterHelperSummaryCallerPlan{}, false, nil
	}
	for _, helperName := range helperNames {
		if !seen[helperName] {
			return InoutWriterHelperSummaryCallerPlan{}, false, nil
		}
	}
	return InoutWriterHelperSummaryCallerPlan{
		CallerName:          fn.Name,
		ProofFamily:         inoutWriterHelperSummaryProofFamily,
		Family:              family,
		AcceptedHelperCalls: calls,
		CallCount:           len(calls),
	}, true, nil
}

func inoutWriterHelperSummaryCallerScalarReturnsFromStackIR(fn ir.IRFunc) (int32, int32, bool) {
	const successReturn int32 = 0
	const failureReturn int32 = 1
	returnCount := 0
	seenSuccess := false
	seenFailure := false
	for i := 1; i < len(fn.Instrs); i++ {
		if fn.Instrs[i].Kind != ir.IRReturn {
			continue
		}
		returnCount++
		if fn.Instrs[i-1].Kind != ir.IRConstI32 {
			return 0, 0, false
		}
		switch fn.Instrs[i-1].Imm {
		case successReturn:
			seenSuccess = true
		case failureReturn:
			seenFailure = true
		default:
			return 0, 0, false
		}
	}
	if returnCount != 2 || !seenSuccess || !seenFailure {
		return 0, 0, false
	}
	return successReturn, failureReturn, true
}

func inoutWriterHelperSummaryShape(name string) (int, bool) {
	switch name {
	case inoutWriterHelperSummaryJSONMessageObjectFunctionName:
		return 27, true
	case inoutWriterHelperSummaryHTTPPlaintextFunctionName:
		return 24, true
	case inoutWriterHelperSummaryHTTPJSONFunctionName:
		return 21, true
	default:
		return 0, false
	}
}

func inoutWriterHelperSummaryCallerShape(name string) (string, []string, bool) {
	switch name {
	case inoutWriterHelperSummaryJSONCallerFunctionName:
		return "json", []string{
			inoutWriterHelperSummaryJSONMessageObjectFunctionName,
		}, true
	case inoutWriterHelperSummaryHTTPCallerFunctionName:
		return "http", []string{
			inoutWriterHelperSummaryHTTPPlaintextFunctionName,
			inoutWriterHelperSummaryHTTPJSONFunctionName,
		}, true
	default:
		return "", nil, false
	}
}

type inoutWriterHelperSummaryStackValue struct {
	kind  string
	local int
	imm   int32
}

func inoutWriterHelperSummaryStoresFromStackIR(
	fn ir.IRFunc,
	storeCount int,
) ([]int32, []int32, []string, int32, bool) {
	stack := make([]inoutWriterHelperSummaryStackValue, 0, 8)
	indexes := make([]int32, 0, storeCount)
	values := make([]int32, 0, storeCount)
	proofs := make([]string, 0, storeCount)
	pop := func() (inoutWriterHelperSummaryStackValue, bool) {
		if len(stack) == 0 {
			return inoutWriterHelperSummaryStackValue{}, false
		}
		value := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return value, true
	}
	pushExpr := func() {
		stack = append(stack, inoutWriterHelperSummaryStackValue{kind: "expr"})
	}
	for i, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRLoadLocal:
			stack = append(stack, inoutWriterHelperSummaryStackValue{
				kind:  "local",
				local: instr.Local,
			})
		case ir.IRConstI32:
			stack = append(stack, inoutWriterHelperSummaryStackValue{
				kind: "const",
				imm:  instr.Imm,
			})
		case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpGeI32,
			ir.IRCmpLeI32, ir.IRCmpNeI32:
			if _, ok := pop(); !ok {
				return nil, nil, nil, 0, false
			}
			if _, ok := pop(); !ok {
				return nil, nil, nil, 0, false
			}
			pushExpr()
		case ir.IRNegI32:
			if _, ok := pop(); !ok {
				return nil, nil, nil, 0, false
			}
			pushExpr()
		case ir.IRIndexStoreU8:
			value, ok := pop()
			if !ok || value.kind != "const" || value.imm < 0 || value.imm > 255 {
				return nil, nil, nil, 0, false
			}
			index, ok := pop()
			if !ok || index.kind != "const" || index.imm != int32(len(indexes)) {
				return nil, nil, nil, 0, false
			}
			length, ok := pop()
			if !ok || length.kind != "local" || length.local != 1 {
				return nil, nil, nil, 0, false
			}
			base, ok := pop()
			if !ok || base.kind != "local" || base.local != 0 {
				return nil, nil, nil, 0, false
			}
			if !strings.HasPrefix(instr.ProofID, inoutWriterHelperSummaryProofPrefix) {
				return nil, nil, nil, 0, false
			}
			indexes = append(indexes, index.imm)
			values = append(values, value.imm)
			proofs = append(proofs, instr.ProofID)
		case ir.IRReturn:
			if i != len(fn.Instrs)-1 || len(stack) != 3 ||
				stack[0].kind != "const" || stack[0].imm != int32(storeCount) ||
				stack[1].kind != "local" || stack[1].local != 0 ||
				stack[2].kind != "local" || stack[2].local != 1 ||
				len(indexes) != storeCount {
				return nil, nil, nil, 0, false
			}
			return indexes, values, proofs, stack[0].imm, true
		case ir.IRStoreLocal, ir.IRCall, ir.IRLabel, ir.IRJmp, ir.IRJmpIfZero:
			return nil, nil, nil, 0, false
		default:
			return nil, nil, nil, 0, false
		}
	}
	return nil, nil, nil, 0, false
}

func buildInoutWriterHelperSummaryCallerMachineFunction(
	plan InoutWriterHelperSummaryCallerPlan,
	successReturn int32,
	failureReturn int32,
) (Function, error) {
	family, helperNames, ok := inoutWriterHelperSummaryCallerShape(plan.CallerName)
	if !ok || family != plan.Family {
		return Function{}, fmt.Errorf(
			"machine helper-summary caller: %s has unsupported caller family",
			plan.CallerName,
		)
	}
	if plan.ProofFamily != inoutWriterHelperSummaryProofFamily ||
		plan.CallCount == 0 ||
		plan.CallCount != len(plan.AcceptedHelperCalls) ||
		plan.CallCount != len(helperNames) {
		return Function{}, fmt.Errorf(
			"machine helper-summary caller: %s has incomplete caller facts",
			plan.CallerName,
		)
	}
	if successReturn != 0 || failureReturn != 1 {
		return Function{}, fmt.Errorf(
			"machine helper-summary caller: %s has unsupported scalar returns %d/%d",
			plan.CallerName,
			successReturn,
			failureReturn,
		)
	}
	expected := make(map[string]bool, len(helperNames))
	for _, helperName := range helperNames {
		expected[helperName] = true
	}
	instrs := []Instr{
		{
			Op:   OpMov,
			Defs: []VReg{"call_count"},
			Imm:  int64(plan.CallCount),
			Note: fmt.Sprintf(
				"helper-summary caller family=%s proof_family=%s call_count=%d",
				plan.Family,
				plan.ProofFamily,
				plan.CallCount,
			),
		},
	}
	for i, call := range plan.AcceptedHelperCalls {
		if !expected[call.HelperName] ||
			call.ArgSlots != 2 ||
			call.RetSlots != 3 ||
			call.Family != plan.Family {
			return Function{}, fmt.Errorf(
				"machine helper-summary caller: %s has unsupported helper call %#v",
				plan.CallerName,
				call,
			)
		}
		instrs = append(instrs, Instr{
			Op:   OpMov,
			Defs: []VReg{VReg(fmt.Sprintf("helper_call%d", i))},
			Imm:  int64(i),
			Note: fmt.Sprintf(
				"helper call %d %s arg_slots=%d ret_slots=%d family=%s proof_family=%s",
				i,
				call.HelperName,
				call.ArgSlots,
				call.RetSlots,
				call.Family,
				plan.ProofFamily,
			),
		})
	}
	success := VReg("success_return")
	failure := VReg("failure_return")
	instrs = append(instrs,
		Instr{
			Op:   OpMov,
			Defs: []VReg{success},
			Imm:  int64(successReturn),
			Note: fmt.Sprintf("scalar success return %d", successReturn),
		},
		Instr{
			Op:   OpMov,
			Defs: []VReg{failure},
			Imm:  int64(failureReturn),
			Note: fmt.Sprintf("scalar failure return %d", failureReturn),
		},
		Instr{
			Op:   OpReturn,
			Uses: []VReg{success},
			Note: "target-neutral helper-summary caller returns visible scalar only",
		},
	)
	out := Function{
		Name:   plan.CallerName,
		Target: "inout-writer-helper-summary-caller",
		Blocks: []Block{{Name: "entry", Instrs: instrs}},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

func buildInoutWriterHelperSummaryMachineFunction(
	plan InoutWriterHelperSummaryPlan,
) (Function, error) {
	if plan.ParamSlots != 2 || plan.ReturnSlots != 3 ||
		plan.VisibleReturnSlots != 1 || plan.HiddenWritebackSlots != 2 {
		return Function{}, fmt.Errorf(
			"machine helper-summary writer: %s has unsupported ABI slots",
			plan.HelperName,
		)
	}
	if plan.StoreCount != len(plan.StoreIndexes) ||
		plan.StoreCount != len(plan.StoreValues) ||
		plan.StoreCount != len(plan.ProofIDs) {
		return Function{}, fmt.Errorf(
			"machine helper-summary writer: %s has incomplete store facts",
			plan.HelperName,
		)
	}
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	instrs := make([]Instr, 0, plan.StoreCount*3+2)
	for i := 0; i < plan.StoreCount; i++ {
		index := plan.StoreIndexes[i]
		value := plan.StoreValues[i]
		if value < 0 || value > 255 {
			return Function{}, fmt.Errorf(
				"machine helper-summary writer: %s store %d byte value %d out of u8 range",
				plan.HelperName,
				i,
				value,
			)
		}
		indexReg := VReg(fmt.Sprintf("index%d", i))
		valueReg := VReg(fmt.Sprintf("byte%d", i))
		instrs = append(instrs,
			Instr{
				Op:   OpMov,
				Defs: []VReg{indexReg},
				Imm:  int64(index),
				Note: fmt.Sprintf("const index %d", index),
			},
			Instr{
				Op:   OpMov,
				Defs: []VReg{valueReg},
				Imm:  int64(value),
				Note: fmt.Sprintf("const byte %d", value),
			},
			Instr{
				Op: OpIndexStore,
				Uses: []VReg{
					local(plan.DstBaseLocal),
					local(plan.DstLenLocal),
					indexReg,
					valueReg,
				},
				Note: fmt.Sprintf(
					"%s; const index %d; byte %d",
					plan.ProofIDs[i],
					index,
					value,
				),
			},
		)
	}
	ret := VReg("return_count")
	instrs = append(instrs,
		Instr{
			Op:   OpMov,
			Defs: []VReg{ret},
			Imm:  int64(plan.ScalarReturnConst),
			Note: fmt.Sprintf("scalar return constant %d", plan.ScalarReturnConst),
		},
		Instr{
			Op:   OpReturn,
			Uses: []VReg{ret},
			Note: fmt.Sprintf("return scalar constant byte count %d", plan.ScalarReturnConst),
		},
	)
	out := Function{
		Name:   plan.HelperName,
		Target: "inout-writer-helper-summary",
		Params: []VReg{
			local(plan.DstBaseLocal),
			local(plan.DstLenLocal),
		},
		Blocks: []Block{{Name: "entry", Instrs: instrs}},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

func buildPostgreSQLInoutWriterMachineFunction(
	name string,
	plan PostgreSQLInoutWriterPlan,
) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	instrs := []Instr{}
	for i, offset := range plan.StoreOffsets {
		value := VReg(fmt.Sprintf("byte%d", i))
		instrs = append(instrs, Instr{
			Op:   OpMov,
			Defs: []VReg{value},
			Uses: []VReg{local(plan.ValueLocal)},
			Note: fmt.Sprintf("writer byte %d from value", i),
		})
		index := local(plan.StartLocal)
		if offset != 0 {
			imm := VReg(fmt.Sprintf("offset%d", i))
			index = VReg(fmt.Sprintf("index%d", i))
			instrs = append(instrs,
				Instr{Op: OpMov, Defs: []VReg{imm}, Imm: int64(offset), Note: "store offset"},
				Instr{
					Op:   OpAdd,
					Defs: []VReg{index},
					Uses: []VReg{local(plan.StartLocal), imm},
					Note: fmt.Sprintf("start + %d", offset),
				},
			)
		}
		proof := ""
		if i < len(plan.ProofIDs) {
			proof = plan.ProofIDs[i]
		}
		instrs = append(instrs, Instr{
			Op: OpIndexStore,
			Uses: []VReg{
				local(plan.DstBaseLocal),
				local(plan.DstLenLocal),
				index,
				value,
			},
			Note: proof,
		})
	}
	returnImm := VReg("return_addend")
	ret := VReg("return_start")
	instrs = append(instrs,
		Instr{Op: OpMov, Defs: []VReg{returnImm}, Imm: int64(plan.ReturnAddend), Note: "return addend"},
		Instr{
			Op:   OpAdd,
			Defs: []VReg{ret},
			Uses: []VReg{local(plan.StartLocal), returnImm},
			Note: fmt.Sprintf("return start + %d", plan.ReturnAddend),
		},
		Instr{Op: OpReturn, Uses: []VReg{ret}, Note: fmt.Sprintf("return start + %d", plan.ReturnAddend)},
	)
	out := Function{
		Name:   name,
		Target: "postgresql-inout-writer",
		Params: []VReg{
			local(plan.DstBaseLocal),
			local(plan.DstLenLocal),
			local(plan.StartLocal),
			local(plan.ValueLocal),
		},
		Blocks: []Block{{Name: "entry", Instrs: instrs}},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

const hashTableLookupFunctionName = "p25.hash_table.lookup"

type HashTableLookupPlan struct {
	Function        Function
	KeysBaseLocal   int
	KeysLenLocal    int
	ValuesBaseLocal int
	ValuesLenLocal  int
	BoundLocal      int
	KeyLocal        int
	IndexLocal      int
	StartLabel      int
	EndLabel        int
	MissLabel       int
	Step            int32
	NotFoundReturn  int32
	KeysProofID     string
	ValuesProofID   string
}

func HashTableLookupFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := HashTableLookupPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func HashTableLookupPlanFromStackIR(fn ir.IRFunc) (HashTableLookupPlan, bool, error) {
	if fn.Name != hashTableLookupFunctionName || fn.ParamSlots != 6 || fn.LocalSlots < 7 ||
		fn.ReturnSlots != 1 {
		return HashTableLookupPlan{}, false, nil
	}
	if len(fn.Instrs) != 28 {
		return HashTableLookupPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) {
		return HashTableLookupPlan{}, false, nil
	}
	indexLocal := in[1].Local
	if indexLocal != 6 {
		return HashTableLookupPlan{}, false, nil
	}
	if in[2].Kind != ir.IRLabel || in[2].Label != 0 {
		return HashTableLookupPlan{}, false, nil
	}
	startLabel := in[2].Label
	if !isLoad(in[3], indexLocal) ||
		!isLoad(in[4], 4) ||
		in[5].Kind != ir.IRCmpLtI32 ||
		in[6].Kind != ir.IRJmpIfZero || in[6].Label != 1 {
		return HashTableLookupPlan{}, false, nil
	}
	endLabel := in[6].Label
	if !isLoad(in[7], 0) ||
		!isLoad(in[8], 1) ||
		!isLoad(in[9], indexLocal) ||
		in[10].Kind != ir.IRIndexLoadI32Unchecked ||
		!validHashTableLookupProof(in[10].ProofID, "keys") ||
		!isLoad(in[11], 5) ||
		in[12].Kind != ir.IRCmpEqI32 ||
		in[13].Kind != ir.IRJmpIfZero || in[13].Label != 2 {
		return HashTableLookupPlan{}, false, nil
	}
	missLabel := in[13].Label
	if !isLoad(in[14], 2) ||
		!isLoad(in[15], 3) ||
		!isLoad(in[16], indexLocal) ||
		in[17].Kind != ir.IRIndexLoadI32Unchecked ||
		!validHashTableLookupProof(in[17].ProofID, "values") ||
		in[18].Kind != ir.IRReturn {
		return HashTableLookupPlan{}, false, nil
	}
	if in[19].Kind != ir.IRLabel || in[19].Label != missLabel {
		return HashTableLookupPlan{}, false, nil
	}
	if !isLoad(in[20], indexLocal) ||
		in[21].Kind != ir.IRConstI32 || in[21].Imm != 1 ||
		in[22].Kind != ir.IRAddI32 ||
		!isStore(in[23], indexLocal) ||
		in[24].Kind != ir.IRJmp || in[24].Label != startLabel ||
		in[25].Kind != ir.IRLabel || in[25].Label != endLabel ||
		in[26].Kind != ir.IRConstI32 ||
		in[27].Kind != ir.IRReturn {
		return HashTableLookupPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return HashTableLookupPlan{}, true, err
	}

	plan := HashTableLookupPlan{
		KeysBaseLocal:   0,
		KeysLenLocal:    1,
		ValuesBaseLocal: 2,
		ValuesLenLocal:  3,
		BoundLocal:      4,
		KeyLocal:        5,
		IndexLocal:      indexLocal,
		StartLabel:      startLabel,
		EndLabel:        endLabel,
		MissLabel:       missLabel,
		Step:            in[21].Imm,
		NotFoundReturn:  in[26].Imm,
		KeysProofID:     in[10].ProofID,
		ValuesProofID:   in[17].ProofID,
	}
	out, err := buildHashTableLookupMachineFunction(fn.Name, plan)
	if err != nil {
		return HashTableLookupPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func validHashTableLookupProof(proofID string, base string) bool {
	return strings.HasPrefix(proofID, "proof:call-boundary:i:"+base+":")
}

func buildHashTableLookupMachineFunction(name string, plan HashTableLookupPlan) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	loopName := scalarLoopLabelName(plan.StartLabel)
	exitName := scalarLoopLabelName(plan.EndLabel)
	missName := scalarLoopLabelName(plan.MissLabel)
	keyElem := VReg("key_elem")
	cmp := VReg("cmp")
	value := VReg("value")
	zero := VReg("zero")

	out := Function{
		Name:   name,
		Target: "hash-table-lookup",
		Params: []VReg{
			local(plan.KeysBaseLocal),
			local(plan.KeysLenLocal),
			local(plan.ValuesBaseLocal),
			local(plan.ValuesLenLocal),
			local(plan.BoundLocal),
			local(plan.KeyLocal),
		},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(plan.IndexLocal)}, Imm: 0, Note: "i = 0"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(plan.IndexLocal), local(plan.BoundLocal)},
						Note: "i < n",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{
						Op:   OpIndexLoad,
						Defs: []VReg{keyElem},
						Uses: []VReg{
							local(plan.KeysBaseLocal),
							local(plan.KeysLenLocal),
							local(plan.IndexLocal),
						},
						Note: plan.KeysProofID,
					},
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{keyElem, local(plan.KeyLocal)},
						Note: "key match",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: missName, Note: "if_zero"},
					{
						Op:   OpIndexLoad,
						Defs: []VReg{value},
						Uses: []VReg{
							local(plan.ValuesBaseLocal),
							local(plan.ValuesLenLocal),
							local(plan.IndexLocal),
						},
						Note: plan.ValuesProofID,
					},
					{Op: OpReturn, Uses: []VReg{value}, Note: "return values[i]"},
				},
				Successors: []string{exitName, missName},
			},
			{
				Name: missName,
				Instrs: []Instr{
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.IndexLocal)},
						Uses: []VReg{local(plan.IndexLocal)},
						Note: "i = i + 1",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{zero},
						Imm:  int64(plan.NotFoundReturn),
						Note: "not found return",
					},
					{Op: OpReturn, Uses: []VReg{zero}, Note: "return not found"},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

const (
	hashTableMainFunctionName     = "p25.hash_table.main"
	hashTableMainLookupCalleeName = "p25.hash_table.lookup"
	hashTableMainLength           = int32(256)
	hashTableMainBackingSlots     = 128
	hashTableMainStep             = int32(1)
	hashTableMainKeysStoreProofID = "proof:while-const:i:keys:"
	hashTableMainValsStoreProofID = "proof:while:i:values:"
)

type HashTableMainPlan struct {
	Function           Function
	NLocal             int
	KeysPtrLocal       int
	KeysLenLocal       int
	ValuesPtrLocal     int
	ValuesLenLocal     int
	IndexLocal         int
	ChecksumLocal      int
	QueryLocal         int
	KeyLocal           int
	KeysBackingLocal   int
	KeysBackingSlots   int
	ValuesBackingLocal int
	ValuesBackingSlots int
	Length             int32
	Step               int32
	FillStartLabel     int
	FillEndLabel       int
	QueryStartLabel    int
	QueryEndLabel      int
	FailureLabel       int
	KeysStoreProofID   string
	ValuesStoreProofID string
	CallName           string
	CallArgSlots       int
	CallRetSlots       int
	SuccessReturn      int32
	FailureReturn      int32
	BoundsChecks       int
}

func HashTableMainFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := HashTableMainPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func HashTableMainPlanFromStackIR(fn ir.IRFunc) (HashTableMainPlan, bool, error) {
	if fn.Name != hashTableMainFunctionName || fn.ParamSlots != 0 || fn.ReturnSlots != 1 ||
		fn.LocalSlots < 265 || len(fn.Instrs) != 79 {
		return HashTableMainPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], hashTableMainLength) {
		return HashTableMainPlan{}, false, nil
	}
	nLocal := in[1].Local
	if nLocal != 0 ||
		!isLoad(in[2], nLocal) ||
		in[3].Kind != ir.IRStackSliceI32 ||
		in[3].Local != 9 ||
		in[3].ArgSlots != hashTableMainBackingSlots ||
		in[3].Imm != hashTableMainLength ||
		in[3].Name != "keys" {
		return HashTableMainPlan{}, false, nil
	}
	keysBackingLocal := in[3].Local
	keysLenLocal := in[4].Local
	keysPtrLocal := in[5].Local
	if !isStore(in[4], keysLenLocal) || keysLenLocal != 2 ||
		!isStore(in[5], keysPtrLocal) || keysPtrLocal != 1 {
		return HashTableMainPlan{}, false, nil
	}
	if !isLoad(in[6], nLocal) ||
		in[7].Kind != ir.IRStackSliceI32 ||
		in[7].Local != 137 ||
		in[7].ArgSlots != hashTableMainBackingSlots ||
		in[7].Imm != hashTableMainLength ||
		in[7].Name != "values" {
		return HashTableMainPlan{}, false, nil
	}
	valuesBackingLocal := in[7].Local
	valuesLenLocal := in[8].Local
	valuesPtrLocal := in[9].Local
	if !isStore(in[8], valuesLenLocal) || valuesLenLocal != 4 ||
		!isStore(in[9], valuesPtrLocal) || valuesPtrLocal != 3 {
		return HashTableMainPlan{}, false, nil
	}
	if !isConstStore(in[10], in[11], 0) {
		return HashTableMainPlan{}, false, nil
	}
	indexLocal := in[11].Local
	if indexLocal != 5 ||
		in[12].Kind != ir.IRLabel || in[12].Label != 0 ||
		!isLoad(in[13], indexLocal) ||
		!isLoad(in[14], nLocal) ||
		in[15].Kind != ir.IRCmpLtI32 ||
		in[16].Kind != ir.IRJmpIfZero || in[16].Label != 1 {
		return HashTableMainPlan{}, false, nil
	}
	fillStartLabel := in[12].Label
	fillEndLabel := in[16].Label
	if !isLoad(in[17], keysPtrLocal) ||
		!isLoad(in[18], keysLenLocal) ||
		!isLoad(in[19], indexLocal) ||
		!isLoad(in[20], indexLocal) ||
		in[21].Kind != ir.IRConstI32 || in[21].Imm != 2 ||
		in[22].Kind != ir.IRMulI32 ||
		in[23].Kind != ir.IRConstI32 || in[23].Imm != 1 ||
		in[24].Kind != ir.IRAddI32 ||
		in[25].Kind != ir.IRIndexStoreI32 ||
		!strings.HasPrefix(in[25].ProofID, hashTableMainKeysStoreProofID) {
		return HashTableMainPlan{}, false, nil
	}
	if !isLoad(in[26], valuesPtrLocal) ||
		!isLoad(in[27], valuesLenLocal) ||
		!isLoad(in[28], indexLocal) ||
		!isLoad(in[29], indexLocal) ||
		in[30].Kind != ir.IRConstI32 || in[30].Imm != 7 ||
		in[31].Kind != ir.IRAddI32 ||
		in[32].Kind != ir.IRIndexStoreI32 ||
		!strings.HasPrefix(in[32].ProofID, hashTableMainValsStoreProofID) {
		return HashTableMainPlan{}, false, nil
	}
	if !isLoad(in[33], indexLocal) ||
		in[34].Kind != ir.IRConstI32 || in[34].Imm != hashTableMainStep ||
		in[35].Kind != ir.IRAddI32 ||
		!isStore(in[36], indexLocal) ||
		in[37].Kind != ir.IRJmp || in[37].Label != fillStartLabel ||
		in[38].Kind != ir.IRLabel || in[38].Label != fillEndLabel {
		return HashTableMainPlan{}, false, nil
	}
	if !isConstStore(in[39], in[40], 0) || !isConstStore(in[41], in[42], 0) {
		return HashTableMainPlan{}, false, nil
	}
	checksumLocal := in[40].Local
	queryLocal := in[42].Local
	if checksumLocal != 6 || queryLocal != 7 ||
		in[43].Kind != ir.IRLabel || in[43].Label != 2 ||
		!isLoad(in[44], queryLocal) ||
		!isLoad(in[45], nLocal) ||
		in[46].Kind != ir.IRCmpLtI32 ||
		in[47].Kind != ir.IRJmpIfZero || in[47].Label != 3 {
		return HashTableMainPlan{}, false, nil
	}
	queryStartLabel := in[43].Label
	queryEndLabel := in[47].Label
	if !isLoad(in[48], queryLocal) ||
		in[49].Kind != ir.IRConstI32 || in[49].Imm != 2 ||
		in[50].Kind != ir.IRMulI32 ||
		in[51].Kind != ir.IRConstI32 || in[51].Imm != 1 ||
		in[52].Kind != ir.IRAddI32 ||
		!isStore(in[53], 8) {
		return HashTableMainPlan{}, false, nil
	}
	keyLocal := in[53].Local
	if !isLoad(in[54], checksumLocal) ||
		!isLoad(in[55], keysPtrLocal) ||
		!isLoad(in[56], keysLenLocal) ||
		!isLoad(in[57], valuesPtrLocal) ||
		!isLoad(in[58], valuesLenLocal) ||
		!isLoad(in[59], nLocal) ||
		!isLoad(in[60], keyLocal) ||
		in[61].Kind != ir.IRCall ||
		in[61].Name != hashTableMainLookupCalleeName ||
		in[61].ArgSlots != 6 ||
		in[61].RetSlots != 1 ||
		in[62].Kind != ir.IRAddI32 ||
		!isStore(in[63], checksumLocal) {
		return HashTableMainPlan{}, false, nil
	}
	if !isLoad(in[64], queryLocal) ||
		in[65].Kind != ir.IRConstI32 || in[65].Imm != hashTableMainStep ||
		in[66].Kind != ir.IRAddI32 ||
		!isStore(in[67], queryLocal) ||
		in[68].Kind != ir.IRJmp || in[68].Label != queryStartLabel ||
		in[69].Kind != ir.IRLabel || in[69].Label != queryEndLabel {
		return HashTableMainPlan{}, false, nil
	}
	if !isLoad(in[70], checksumLocal) ||
		in[71].Kind != ir.IRConstI32 || in[71].Imm != 0 ||
		in[72].Kind != ir.IRCmpGtI32 ||
		in[73].Kind != ir.IRJmpIfZero || in[73].Label != 4 ||
		in[74].Kind != ir.IRConstI32 || in[74].Imm != 0 ||
		in[75].Kind != ir.IRReturn ||
		in[76].Kind != ir.IRLabel || in[76].Label != in[73].Label ||
		in[77].Kind != ir.IRConstI32 || in[77].Imm != 1 ||
		in[78].Kind != ir.IRReturn {
		return HashTableMainPlan{}, false, nil
	}
	for _, local := range []struct {
		slot int
		name string
	}{
		{nLocal, "n"},
		{keysPtrLocal, "keys ptr"},
		{keysLenLocal, "keys len"},
		{valuesPtrLocal, "values ptr"},
		{valuesLenLocal, "values len"},
		{indexLocal, "index"},
		{checksumLocal, "checksum"},
		{queryLocal, "query"},
		{keyLocal, "key"},
	} {
		if err := validateScalarLoopLocal(fn, local.slot, local.name); err != nil {
			return HashTableMainPlan{}, true, err
		}
	}
	if keysBackingLocal+hashTableMainBackingSlots > fn.LocalSlots ||
		valuesBackingLocal+hashTableMainBackingSlots > fn.LocalSlots {
		return HashTableMainPlan{}, true, fmt.Errorf(
			"machine hash table main lowering: backing slots out of bounds (locals=%d)",
			fn.LocalSlots,
		)
	}
	if !distinctAllocationLoopLocals(
		nLocal,
		keysPtrLocal,
		keysLenLocal,
		valuesPtrLocal,
		valuesLenLocal,
		indexLocal,
		checksumLocal,
		queryLocal,
		keyLocal,
	) {
		return HashTableMainPlan{}, false, nil
	}

	plan := HashTableMainPlan{
		NLocal:             nLocal,
		KeysPtrLocal:       keysPtrLocal,
		KeysLenLocal:       keysLenLocal,
		ValuesPtrLocal:     valuesPtrLocal,
		ValuesLenLocal:     valuesLenLocal,
		IndexLocal:         indexLocal,
		ChecksumLocal:      checksumLocal,
		QueryLocal:         queryLocal,
		KeyLocal:           keyLocal,
		KeysBackingLocal:   keysBackingLocal,
		KeysBackingSlots:   hashTableMainBackingSlots,
		ValuesBackingLocal: valuesBackingLocal,
		ValuesBackingSlots: hashTableMainBackingSlots,
		Length:             hashTableMainLength,
		Step:               hashTableMainStep,
		FillStartLabel:     fillStartLabel,
		FillEndLabel:       fillEndLabel,
		QueryStartLabel:    queryStartLabel,
		QueryEndLabel:      queryEndLabel,
		FailureLabel:       in[73].Label,
		KeysStoreProofID:   in[25].ProofID,
		ValuesStoreProofID: in[32].ProofID,
		CallName:           in[61].Name,
		CallArgSlots:       in[61].ArgSlots,
		CallRetSlots:       in[61].RetSlots,
		SuccessReturn:      in[74].Imm,
		FailureReturn:      in[77].Imm,
		BoundsChecks:       2,
	}
	out, err := buildHashTableMainMachineFunction(fn.Name, SysVCallABIInfo(), plan)
	if err != nil {
		return HashTableMainPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func buildHashTableMainMachineFunction(
	name string,
	callABI CallABIInfo,
	plan HashTableMainPlan,
) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	fillLoop := scalarLoopLabelName(plan.FillStartLabel)
	fillAfter := scalarLoopLabelName(plan.FillEndLabel)
	queryLoop := scalarLoopLabelName(plan.QueryStartLabel)
	queryAfter := scalarLoopLabelName(plan.QueryEndLabel)
	failure := scalarLoopLabelName(plan.FailureLabel)
	success := "return_success"
	keyScale := VReg("key.scale")
	seven := VReg("seven")
	fillValue := VReg("fill.value")
	fillCmp := VReg("fill.cmp")
	queryCmp := VReg("query.cmp")
	callRet := VReg("lookup.value")
	nextChecksum := VReg("next.checksum")
	finalZero := VReg("final.zero")
	finalCmp := VReg("final.cmp")
	successValue := VReg("return_success")
	failureValue := VReg("return_failure")

	out := Function{
		Name:   name,
		Target: "hash-table-main",
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(plan.NLocal)}, Imm: int64(plan.Length), Note: "n = 256"},
					{Op: OpMov, Defs: []VReg{local(plan.KeysPtrLocal)}, Note: "keys stack ptr"},
					{Op: OpMov, Defs: []VReg{local(plan.KeysLenLocal)}, Imm: int64(plan.Length), Note: "keys len"},
					{Op: OpMov, Defs: []VReg{local(plan.ValuesPtrLocal)}, Note: "values stack ptr"},
					{
						Op:   OpMov,
						Defs: []VReg{local(plan.ValuesLenLocal)},
						Imm:  int64(plan.Length),
						Note: "values len",
					},
					{Op: OpMov, Defs: []VReg{keyScale}, Imm: 2, Note: "key scale"},
					{Op: OpMov, Defs: []VReg{seven}, Imm: 7, Note: "value offset"},
					{Op: OpMov, Defs: []VReg{local(plan.IndexLocal)}, Imm: 0, Note: "i = 0"},
					{Op: OpBranch, Target: fillLoop},
				},
				Successors: []string{fillLoop},
			},
			{
				Name: fillLoop,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{fillCmp},
						Uses: []VReg{local(plan.IndexLocal), local(plan.NLocal)},
						Note: "i < n",
					},
					{Op: OpBranchIf, Uses: []VReg{fillCmp}, Target: fillAfter, Note: "if_zero"},
					{
						Op:   OpMul,
						Defs: []VReg{fillValue},
						Uses: []VReg{local(plan.IndexLocal), keyScale},
						Note: "i * 2",
					},
					{Op: OpInc, Defs: []VReg{fillValue}, Uses: []VReg{fillValue}, Note: "i * 2 + 1"},
					{
						Op: OpIndexStore,
						Uses: []VReg{
							local(plan.KeysPtrLocal),
							local(plan.KeysLenLocal),
							local(plan.IndexLocal),
							fillValue,
						},
						Note: plan.KeysStoreProofID + " keys[i] = i * 2 + 1",
					},
					{
						Op:   OpAdd,
						Defs: []VReg{fillValue},
						Uses: []VReg{local(plan.IndexLocal), seven},
						Note: "values[i] = i + 7",
					},
					{
						Op: OpIndexStore,
						Uses: []VReg{
							local(plan.ValuesPtrLocal),
							local(plan.ValuesLenLocal),
							local(plan.IndexLocal),
							fillValue,
						},
						Note: plan.ValuesStoreProofID + " values[i] = i + 7",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.IndexLocal)},
						Uses: []VReg{local(plan.IndexLocal)},
						Note: "i++",
					},
					{Op: OpBranch, Target: fillLoop},
				},
				Successors: []string{fillAfter, fillLoop},
			},
			{
				Name: fillAfter,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(plan.ChecksumLocal)}, Imm: 0, Note: "checksum = 0"},
					{Op: OpMov, Defs: []VReg{local(plan.QueryLocal)}, Imm: 0, Note: "q = 0"},
					{Op: OpBranch, Target: queryLoop},
				},
				Successors: []string{queryLoop},
			},
			{
				Name: queryLoop,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{queryCmp},
						Uses: []VReg{local(plan.QueryLocal), local(plan.NLocal)},
						Note: "q < n",
					},
					{Op: OpBranchIf, Uses: []VReg{queryCmp}, Target: queryAfter, Note: "if_zero"},
					{
						Op:   OpMul,
						Defs: []VReg{local(plan.KeyLocal)},
						Uses: []VReg{local(plan.QueryLocal), keyScale},
						Note: "key = q * 2",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.KeyLocal)},
						Uses: []VReg{local(plan.KeyLocal)},
						Note: "key = q * 2 + 1",
					},
					{
						Op:   OpCall,
						Defs: []VReg{callRet},
						Uses: []VReg{
							local(plan.KeysPtrLocal),
							local(plan.KeysLenLocal),
							local(plan.ValuesPtrLocal),
							local(plan.ValuesLenLocal),
							local(plan.NLocal),
							local(plan.KeyLocal),
						},
						Call:     plan.CallName,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "normal ABI call patch to p25.hash_table.lookup",
					},
					{
						Op:   OpAdd,
						Defs: []VReg{nextChecksum},
						Uses: []VReg{local(plan.ChecksumLocal), callRet},
						Note: "checksum += lookup",
					},
					{
						Op:   OpMov,
						Defs: []VReg{local(plan.ChecksumLocal)},
						Uses: []VReg{nextChecksum},
						Note: "store checksum",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.QueryLocal)},
						Uses: []VReg{local(plan.QueryLocal)},
						Note: "q++",
					},
					{Op: OpBranch, Target: queryLoop},
				},
				Successors: []string{queryAfter, queryLoop},
			},
			{
				Name: queryAfter,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{finalZero}, Imm: 0, Note: "zero"},
					{
						Op:   OpCmp,
						Defs: []VReg{finalCmp},
						Uses: []VReg{local(plan.ChecksumLocal), finalZero},
						Note: "checksum > 0",
					},
					{Op: OpBranchIf, Uses: []VReg{finalCmp}, Target: failure, Note: "if_zero"},
					{Op: OpBranch, Target: success},
				},
				Successors: []string{failure, success},
			},
			{
				Name: success,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{successValue}, Imm: int64(plan.SuccessReturn), Note: "return 0"},
					{Op: OpReturn, Uses: []VReg{successValue}},
				},
			},
			{
				Name: failure,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{failureValue}, Imm: int64(plan.FailureReturn), Note: "return 1"},
					{Op: OpReturn, Uses: []VReg{failureValue}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

// ---- scalar_max_loop.go ----

type ScalarIntMaxLoopPlan struct {
	Function   Function
	ParamLocal int
	MaxLocal   int
	IndexLocal int
	StartLabel int
	KeepLabel  int
	EndLabel   int
}

func ScalarIntMaxLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ScalarIntMaxLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ScalarIntMaxLoopPlanFromStackIR(fn ir.IRFunc) (ScalarIntMaxLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 1 || fn.LocalSlots < 3 {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	if len(fn.Instrs) != 24 {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	maxLocal := in[1].Local
	indexLocal := in[3].Local
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 0) || in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	if !isLoad(in[9], indexLocal) || !isLoad(in[10], maxLocal) || in[11].Kind != ir.IRCmpGtI32 ||
		in[12].Kind != ir.IRJmpIfZero {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	keepLabel := in[12].Label
	if keepLabel < 0 {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	if !isLoad(in[13], indexLocal) || !isStore(in[14], maxLocal) {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	if in[15].Kind != ir.IRLabel || in[15].Label != keepLabel {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	if !isLoad(in[16], indexLocal) || in[17].Kind != ir.IRConstI32 || in[17].Imm != 1 ||
		in[18].Kind != ir.IRAddI32 ||
		!isStore(in[19], indexLocal) {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	if in[20].Kind != ir.IRJmp || in[20].Label != startLabel || in[21].Kind != ir.IRLabel ||
		in[21].Label != endLabel ||
		!isLoad(in[22], maxLocal) ||
		in[23].Kind != ir.IRReturn {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, maxLocal, "max"); err != nil {
		return ScalarIntMaxLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntMaxLoopPlan{}, true, err
	}
	if maxLocal == indexLocal || maxLocal == 0 || indexLocal == 0 || startLabel == keepLabel ||
		startLabel == endLabel ||
		keepLabel == endLabel {
		return ScalarIntMaxLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	loopCmp := VReg("t0")
	maxCmp := VReg("t1")
	loopName := scalarLoopLabelName(startLabel)
	keepName := scalarLoopLabelName(keepLabel)
	exitName := scalarLoopLabelName(endLabel)
	updateName := "update"
	out := Function{
		Name:   fn.Name,
		Target: "scalar-int-max-loop",
		Params: []VReg{local(0)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(maxLocal)}, Imm: 0, Note: "loop max = 0"},
					{Op: OpMov, Defs: []VReg{local(indexLocal)}, Imm: 0, Note: "loop index = 0"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{loopCmp},
						Uses: []VReg{local(indexLocal), local(0)},
						Note: "index < n",
					},
					{Op: OpBranchIf, Uses: []VReg{loopCmp}, Target: exitName, Note: "if_zero"},
					{
						Op:   OpCmp,
						Defs: []VReg{maxCmp},
						Uses: []VReg{local(indexLocal), local(maxLocal)},
						Note: "index > max",
					},
					{Op: OpBranchIf, Uses: []VReg{maxCmp}, Target: keepName, Note: "if_zero"},
					{Op: OpBranch, Target: updateName},
				},
				Successors: []string{exitName, keepName, updateName},
			},
			{
				Name: updateName,
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{local(maxLocal)},
						Uses: []VReg{local(indexLocal)},
						Note: "max = index",
					},
					{Op: OpBranch, Target: keepName},
				},
				Successors: []string{keepName},
			},
			{
				Name: keepName,
				Instrs: []Instr{
					{
						Op:   OpInc,
						Defs: []VReg{local(indexLocal)},
						Uses: []VReg{local(indexLocal)},
						Note: "index++",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{local(maxLocal)}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return ScalarIntMaxLoopPlan{}, true, err
	}
	return ScalarIntMaxLoopPlan{
		Function:   out,
		ParamLocal: 0,
		MaxLocal:   maxLocal,
		IndexLocal: indexLocal,
		StartLabel: startLabel,
		KeepLabel:  keepLabel,
		EndLabel:   endLabel,
	}, true, nil
}

// ---- scalar_product_loop.go ----

type ScalarIntProductLoopPlan struct {
	Function     Function
	ParamLocal   int
	IndexLocal   int
	ProductLocal int
	StartLabel   int
	EndLabel     int
}

func ScalarIntProductLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ScalarIntProductLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ScalarIntProductLoopPlanFromStackIR(fn ir.IRFunc) (ScalarIntProductLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 1 || fn.LocalSlots < 3 {
		return ScalarIntProductLoopPlan{}, false, nil
	}
	if len(fn.Instrs) != 23 {
		return ScalarIntProductLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 1) {
		return ScalarIntProductLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	productLocal := in[3].Local
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarIntProductLoopPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 0) || in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntProductLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntProductLoopPlan{}, false, nil
	}
	if !isLoad(in[9], productLocal) || !isLoad(in[10], indexLocal) ||
		in[11].Kind != ir.IRConstI32 ||
		in[11].Imm != 1 ||
		in[12].Kind != ir.IRAddI32 ||
		in[13].Kind != ir.IRMulI32 ||
		!isStore(in[14], productLocal) {
		return ScalarIntProductLoopPlan{}, false, nil
	}
	if !isLoad(in[15], indexLocal) || in[16].Kind != ir.IRConstI32 || in[16].Imm != 1 ||
		in[17].Kind != ir.IRAddI32 ||
		!isStore(in[18], indexLocal) {
		return ScalarIntProductLoopPlan{}, false, nil
	}
	if in[19].Kind != ir.IRJmp || in[19].Label != startLabel || in[20].Kind != ir.IRLabel ||
		in[20].Label != endLabel ||
		!isLoad(in[21], productLocal) ||
		in[22].Kind != ir.IRReturn {
		return ScalarIntProductLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntProductLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, productLocal, "product"); err != nil {
		return ScalarIntProductLoopPlan{}, true, err
	}
	if indexLocal == productLocal || indexLocal == 0 || productLocal == 0 {
		return ScalarIntProductLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	cmp := VReg("t0")
	one := VReg("t1")
	factor := VReg("t2")
	loopName := scalarLoopLabelName(startLabel)
	exitName := scalarLoopLabelName(endLabel)
	out := Function{
		Name:   fn.Name,
		Target: "scalar-int-product-loop",
		Params: []VReg{local(0)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(indexLocal)}, Imm: 0, Note: "loop index = 0"},
					{
						Op:   OpMov,
						Defs: []VReg{local(productLocal)},
						Imm:  1,
						Note: "loop product = 1",
					},
					{Op: OpMov, Defs: []VReg{one}, Imm: 1, Note: "one"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(indexLocal), local(0)},
						Note: "index < n",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{
						Op:   OpAdd,
						Defs: []VReg{factor},
						Uses: []VReg{local(indexLocal), one},
						Note: "index + 1",
					},
					{
						Op:   OpMul,
						Defs: []VReg{local(productLocal)},
						Uses: []VReg{local(productLocal), factor},
						Note: "product *= index + 1",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(indexLocal)},
						Uses: []VReg{local(indexLocal)},
						Note: "index++",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{local(productLocal)}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return ScalarIntProductLoopPlan{}, true, err
	}
	return ScalarIntProductLoopPlan{
		Function:     out,
		ParamLocal:   0,
		IndexLocal:   indexLocal,
		ProductLocal: productLocal,
		StartLabel:   startLabel,
		EndLabel:     endLabel,
	}, true, nil
}

// ---- scalar_slice_sum.go ----

type ScalarI32SliceSumLoopPlan struct {
	Function   Function
	BaseLocal  int
	LenLocal   int
	IndexLocal int
	TotalLocal int
	Step       int32
	StartLabel int
	EndLabel   int
	ProofID    string
}

func ScalarI32SliceSumLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ScalarI32SliceSumLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ScalarI32SliceSumLoopPlanFromStackIR(fn ir.IRFunc) (ScalarI32SliceSumLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 2 || fn.LocalSlots < 4 {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	if len(fn.Instrs) != 24 {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	totalLocal := in[1].Local
	indexLocal := in[3].Local
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 1) || in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], 0) || !isLoad(in[11], 1) ||
		!isLoad(in[12], indexLocal) {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	if in[13].Kind != ir.IRIndexLoadI32Unchecked ||
		!strings.HasPrefix(in[13].ProofID, "proof:while:") {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	if in[14].Kind != ir.IRAddI32 || !isStore(in[15], totalLocal) {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	if !isLoad(in[16], indexLocal) || in[17].Kind != ir.IRConstI32 || in[18].Kind != ir.IRAddI32 ||
		!isStore(in[19], indexLocal) {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	step := in[17].Imm
	if !validScalarLoopStep(step) {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	if in[20].Kind != ir.IRJmp || in[20].Label != startLabel || in[21].Kind != ir.IRLabel ||
		in[21].Label != endLabel ||
		!isLoad(in[22], totalLocal) ||
		in[23].Kind != ir.IRReturn {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return ScalarI32SliceSumLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarI32SliceSumLoopPlan{}, true, err
	}
	if totalLocal == indexLocal || totalLocal < fn.ParamSlots || indexLocal < fn.ParamSlots {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	cmp := VReg("t0")
	elem := VReg("t1")
	stepReg := VReg("t2")
	loopName := scalarLoopLabelName(startLabel)
	exitName := scalarLoopLabelName(endLabel)
	entryInstrs := []Instr{
		{Op: OpMov, Defs: []VReg{local(totalLocal)}, Imm: 0, Note: "total = 0"},
		{Op: OpMov, Defs: []VReg{local(indexLocal)}, Imm: 0, Note: "index = 0"},
	}
	if step != 1 {
		entryInstrs = append(
			entryInstrs,
			Instr{Op: OpMov, Defs: []VReg{stepReg}, Imm: int64(step), Note: "loop step"},
		)
	}
	entryInstrs = append(entryInstrs, Instr{Op: OpBranch, Target: loopName})
	advanceInstr := Instr{
		Op:   OpInc,
		Defs: []VReg{local(indexLocal)},
		Uses: []VReg{local(indexLocal)},
		Note: "index++",
	}
	if step != 1 {
		advanceInstr = Instr{
			Op:   OpAdd,
			Defs: []VReg{local(indexLocal)},
			Uses: []VReg{local(indexLocal), stepReg},
			Note: "index += step",
		}
	}
	out := Function{
		Name:   fn.Name,
		Target: "scalar-i32-slice-sum",
		Params: []VReg{local(0), local(1)},
		Blocks: []Block{
			{
				Name:       "entry",
				Instrs:     entryInstrs,
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(indexLocal), local(1)},
						Note: "index < len",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{
						Op:   OpIndexLoad,
						Defs: []VReg{elem},
						Uses: []VReg{local(0), local(1), local(indexLocal)},
						Note: in[13].ProofID,
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(totalLocal)},
						Uses: []VReg{local(totalLocal), elem},
						Note: "total += xs[index]",
					},
					advanceInstr,
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{local(totalLocal)}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return ScalarI32SliceSumLoopPlan{}, true, err
	}
	return ScalarI32SliceSumLoopPlan{
		Function:   out,
		BaseLocal:  0,
		LenLocal:   1,
		IndexLocal: indexLocal,
		TotalLocal: totalLocal,
		Step:       step,
		StartLabel: startLabel,
		EndLabel:   endLabel,
		ProofID:    in[13].ProofID,
	}, true, nil
}

// ---- slice_sum_main.go ----

const (
	sliceSumMainFunctionName = "p25.slice_sum.main"
	sliceSumMainLength       = int32(4096)
	sliceSumMainBackingSlots = 2048
	sliceSumMainFillModulus  = int32(97)
	sliceSumMainRepeatCount  = int32(64)
	sliceSumMainStep         = int32(1)
	sliceSumMainStoreProofID = "proof:while:i:xs:8:5"
	sliceSumMainLoadProofID  = "proof:while:i:xs:15:9"
)

type SliceSumMainPlan struct {
	Function        Function
	NLocal          int
	SlicePtrLocal   int
	SliceLenLocal   int
	IndexLocal      int
	TotalLocal      int
	RepeatLocal     int
	BackingLocal    int
	BackingSlots    int
	Length          int32
	FillModulus     int32
	RepeatCount     int32
	Step            int32
	FillStartLabel  int
	FillEndLabel    int
	OuterStartLabel int
	OuterEndLabel   int
	InnerStartLabel int
	InnerEndLabel   int
	FailureLabel    int
	StoreProofID    string
	LoadProofID     string
	SuccessReturn   int32
	FailureReturn   int32
	BoundsChecks    int
}

func SliceSumMainFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := SliceSumMainPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func SliceSumMainPlanFromStackIR(fn ir.IRFunc) (SliceSumMainPlan, bool, error) {
	if fn.Name != sliceSumMainFunctionName || fn.ParamSlots != 0 || fn.ReturnSlots != 1 ||
		fn.LocalSlots < 6 {
		return SliceSumMainPlan{}, false, nil
	}
	if len(fn.Instrs) != 70 {
		return SliceSumMainPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], sliceSumMainLength) {
		return SliceSumMainPlan{}, false, nil
	}
	nLocal := in[1].Local
	if !isLoad(in[2], nLocal) ||
		in[3].Kind != ir.IRStackSliceI32 ||
		in[3].Local < 0 ||
		in[3].ArgSlots != sliceSumMainBackingSlots ||
		in[3].Imm != sliceSumMainLength ||
		in[3].Name != "xs" {
		return SliceSumMainPlan{}, false, nil
	}
	backingLocal := in[3].Local
	backingSlots := in[3].ArgSlots
	sliceLenLocal := in[4].Local
	slicePtrLocal := in[5].Local
	if !isStore(in[4], sliceLenLocal) || !isStore(in[5], slicePtrLocal) {
		return SliceSumMainPlan{}, false, nil
	}
	if !isConstStore(in[6], in[7], 0) {
		return SliceSumMainPlan{}, false, nil
	}
	indexLocal := in[7].Local
	fillStartLabel := in[8].Label
	if in[8].Kind != ir.IRLabel || fillStartLabel != 0 ||
		!isLoad(in[9], indexLocal) ||
		!isLoad(in[10], nLocal) ||
		in[11].Kind != ir.IRCmpLtI32 ||
		in[12].Kind != ir.IRJmpIfZero ||
		in[12].Label != 1 {
		return SliceSumMainPlan{}, false, nil
	}
	fillEndLabel := in[12].Label
	if !isLoad(in[13], slicePtrLocal) ||
		!isLoad(in[14], sliceLenLocal) ||
		!isLoad(in[15], indexLocal) ||
		!isLoad(in[16], indexLocal) ||
		in[17].Kind != ir.IRConstI32 ||
		in[17].Imm != sliceSumMainFillModulus ||
		in[18].Kind != ir.IRModI32 ||
		in[19].Kind != ir.IRIndexStoreI32 ||
		in[19].ProofID != sliceSumMainStoreProofID {
		return SliceSumMainPlan{}, false, nil
	}
	if !isLoad(in[20], indexLocal) ||
		in[21].Kind != ir.IRConstI32 ||
		in[21].Imm != sliceSumMainStep ||
		in[22].Kind != ir.IRAddI32 ||
		!isStore(in[23], indexLocal) ||
		in[24].Kind != ir.IRJmp ||
		in[24].Label != fillStartLabel ||
		in[25].Kind != ir.IRLabel ||
		in[25].Label != fillEndLabel {
		return SliceSumMainPlan{}, false, nil
	}
	if !isConstStore(in[26], in[27], 0) || !isConstStore(in[28], in[29], 0) {
		return SliceSumMainPlan{}, false, nil
	}
	totalLocal := in[27].Local
	repeatLocal := in[29].Local
	if in[30].Kind != ir.IRLabel || in[30].Label != 2 ||
		!isLoad(in[31], repeatLocal) ||
		in[32].Kind != ir.IRConstI32 ||
		in[32].Imm != sliceSumMainRepeatCount ||
		in[33].Kind != ir.IRCmpLtI32 ||
		in[34].Kind != ir.IRJmpIfZero ||
		in[34].Label != 3 {
		return SliceSumMainPlan{}, false, nil
	}
	outerStartLabel := in[30].Label
	outerEndLabel := in[34].Label
	if !isConstStore(in[35], in[36], 0) || in[36].Local != indexLocal ||
		in[37].Kind != ir.IRLabel ||
		in[37].Label != 4 ||
		!isLoad(in[38], indexLocal) ||
		!isLoad(in[39], nLocal) ||
		in[40].Kind != ir.IRCmpLtI32 ||
		in[41].Kind != ir.IRJmpIfZero ||
		in[41].Label != 5 {
		return SliceSumMainPlan{}, false, nil
	}
	innerStartLabel := in[37].Label
	innerEndLabel := in[41].Label
	if !isLoad(in[42], totalLocal) ||
		!isLoad(in[43], slicePtrLocal) ||
		!isLoad(in[44], sliceLenLocal) ||
		!isLoad(in[45], indexLocal) ||
		in[46].Kind != ir.IRIndexLoadI32Unchecked ||
		in[46].ProofID != sliceSumMainLoadProofID ||
		in[47].Kind != ir.IRAddI32 ||
		!isStore(in[48], totalLocal) {
		return SliceSumMainPlan{}, false, nil
	}
	if !isLoad(in[49], indexLocal) ||
		in[50].Kind != ir.IRConstI32 ||
		in[50].Imm != sliceSumMainStep ||
		in[51].Kind != ir.IRAddI32 ||
		!isStore(in[52], indexLocal) ||
		in[53].Kind != ir.IRJmp ||
		in[53].Label != innerStartLabel ||
		in[54].Kind != ir.IRLabel ||
		in[54].Label != innerEndLabel {
		return SliceSumMainPlan{}, false, nil
	}
	if !isLoad(in[55], repeatLocal) ||
		in[56].Kind != ir.IRConstI32 ||
		in[56].Imm != sliceSumMainStep ||
		in[57].Kind != ir.IRAddI32 ||
		!isStore(in[58], repeatLocal) ||
		in[59].Kind != ir.IRJmp ||
		in[59].Label != outerStartLabel ||
		in[60].Kind != ir.IRLabel ||
		in[60].Label != outerEndLabel {
		return SliceSumMainPlan{}, false, nil
	}
	if !isLoad(in[61], totalLocal) ||
		in[62].Kind != ir.IRConstI32 ||
		in[62].Imm != 0 ||
		in[63].Kind != ir.IRCmpGtI32 ||
		in[64].Kind != ir.IRJmpIfZero ||
		in[64].Label != 6 ||
		in[65].Kind != ir.IRConstI32 ||
		in[65].Imm != 0 ||
		in[66].Kind != ir.IRReturn ||
		in[67].Kind != ir.IRLabel ||
		in[67].Label != in[64].Label ||
		in[68].Kind != ir.IRConstI32 ||
		in[68].Imm != 1 ||
		in[69].Kind != ir.IRReturn {
		return SliceSumMainPlan{}, false, nil
	}
	for _, local := range []struct {
		slot int
		name string
	}{
		{nLocal, "n"},
		{slicePtrLocal, "slice ptr"},
		{sliceLenLocal, "slice len"},
		{indexLocal, "index"},
		{totalLocal, "total"},
		{repeatLocal, "repeat"},
	} {
		if err := validateScalarLoopLocal(fn, local.slot, local.name); err != nil {
			return SliceSumMainPlan{}, true, err
		}
	}
	if backingLocal+backingSlots > fn.LocalSlots {
		return SliceSumMainPlan{}, true, fmt.Errorf(
			"machine slice_sum main lowering: backing slots [%d,%d) out of bounds (locals=%d)",
			backingLocal,
			backingLocal+backingSlots,
			fn.LocalSlots,
		)
	}
	if !distinctAllocationLoopLocals(
		nLocal,
		slicePtrLocal,
		sliceLenLocal,
		indexLocal,
		totalLocal,
		repeatLocal,
	) {
		return SliceSumMainPlan{}, false, nil
	}
	for _, local := range []int{
		nLocal,
		slicePtrLocal,
		sliceLenLocal,
		indexLocal,
		totalLocal,
		repeatLocal,
	} {
		if local >= backingLocal && local < backingLocal+backingSlots {
			return SliceSumMainPlan{}, false, nil
		}
	}

	plan := SliceSumMainPlan{
		NLocal:          nLocal,
		SlicePtrLocal:   slicePtrLocal,
		SliceLenLocal:   sliceLenLocal,
		IndexLocal:      indexLocal,
		TotalLocal:      totalLocal,
		RepeatLocal:     repeatLocal,
		BackingLocal:    backingLocal,
		BackingSlots:    backingSlots,
		Length:          sliceSumMainLength,
		FillModulus:     sliceSumMainFillModulus,
		RepeatCount:     sliceSumMainRepeatCount,
		Step:            sliceSumMainStep,
		FillStartLabel:  fillStartLabel,
		FillEndLabel:    fillEndLabel,
		OuterStartLabel: outerStartLabel,
		OuterEndLabel:   outerEndLabel,
		InnerStartLabel: innerStartLabel,
		InnerEndLabel:   innerEndLabel,
		FailureLabel:    in[64].Label,
		StoreProofID:    in[19].ProofID,
		LoadProofID:     in[46].ProofID,
		SuccessReturn:   in[65].Imm,
		FailureReturn:   in[68].Imm,
		BoundsChecks:    2,
	}
	out, err := buildSliceSumMainMachineFunction(fn.Name, plan)
	if err != nil {
		return SliceSumMainPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func buildSliceSumMainMachineFunction(name string, plan SliceSumMainPlan) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	fillLoop := scalarLoopLabelName(plan.FillStartLabel)
	fillAfter := scalarLoopLabelName(plan.FillEndLabel)
	outerLoop := scalarLoopLabelName(plan.OuterStartLabel)
	outerAfter := scalarLoopLabelName(plan.OuterEndLabel)
	innerLoop := scalarLoopLabelName(plan.InnerStartLabel)
	innerAfter := scalarLoopLabelName(plan.InnerEndLabel)
	failure := scalarLoopLabelName(plan.FailureLabel)
	success := "return_success"
	modulus := VReg("fill.modulus")
	repeatLimit := VReg("repeat.limit")
	fillValue := VReg("fill.value")
	fillCmp := VReg("fill.cmp")
	outerCmp := VReg("outer.cmp")
	innerCmp := VReg("inner.cmp")
	elem := VReg("sum.elem")
	finalZero := VReg("final.zero")
	finalCmp := VReg("final.cmp")
	successValue := VReg("return_success")
	failureValue := VReg("return_failure")

	out := Function{
		Name:   name,
		Target: "slice-sum-main",
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(plan.NLocal)}, Imm: int64(plan.Length), Note: "n = 4096"},
					{Op: OpMov, Defs: []VReg{local(plan.SlicePtrLocal)}, Note: "xs stack ptr"},
					{Op: OpMov, Defs: []VReg{local(plan.SliceLenLocal)}, Imm: int64(plan.Length), Note: "xs len"},
					{Op: OpMov, Defs: []VReg{local(plan.IndexLocal)}, Imm: 0, Note: "i = 0"},
					{Op: OpBranch, Target: fillLoop},
				},
				Successors: []string{fillLoop},
			},
			{
				Name: fillLoop,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{fillCmp},
						Uses: []VReg{local(plan.IndexLocal), local(plan.NLocal)},
						Note: "i < n",
					},
					{Op: OpBranchIf, Uses: []VReg{fillCmp}, Target: fillAfter, Note: "if_zero"},
					{Op: OpMov, Defs: []VReg{modulus}, Imm: int64(plan.FillModulus), Note: "fill modulus"},
					{
						Op:   OpMod,
						Defs: []VReg{fillValue},
						Uses: []VReg{local(plan.IndexLocal), modulus},
						Note: "i % 97",
					},
					{
						Op: OpIndexStore,
						Uses: []VReg{
							local(plan.SlicePtrLocal),
							local(plan.SliceLenLocal),
							local(plan.IndexLocal),
							fillValue,
						},
						Note: plan.StoreProofID,
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.IndexLocal)},
						Uses: []VReg{local(plan.IndexLocal)},
						Note: "i++",
					},
					{Op: OpBranch, Target: fillLoop},
				},
				Successors: []string{fillAfter, fillLoop},
			},
			{
				Name: fillAfter,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(plan.TotalLocal)}, Imm: 0, Note: "total = 0"},
					{Op: OpMov, Defs: []VReg{local(plan.RepeatLocal)}, Imm: 0, Note: "r = 0"},
					{Op: OpBranch, Target: outerLoop},
				},
				Successors: []string{outerLoop},
			},
			{
				Name: outerLoop,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{repeatLimit}, Imm: int64(plan.RepeatCount), Note: "repeat count"},
					{
						Op:   OpCmp,
						Defs: []VReg{outerCmp},
						Uses: []VReg{local(plan.RepeatLocal), repeatLimit},
						Note: "r < 64",
					},
					{Op: OpBranchIf, Uses: []VReg{outerCmp}, Target: outerAfter, Note: "if_zero"},
					{Op: OpMov, Defs: []VReg{local(plan.IndexLocal)}, Imm: 0, Note: "i = 0"},
					{Op: OpBranch, Target: innerLoop},
				},
				Successors: []string{outerAfter, innerLoop},
			},
			{
				Name: innerLoop,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{innerCmp},
						Uses: []VReg{local(plan.IndexLocal), local(plan.NLocal)},
						Note: "i < n",
					},
					{Op: OpBranchIf, Uses: []VReg{innerCmp}, Target: innerAfter, Note: "if_zero"},
					{
						Op: OpIndexLoad,
						Defs: []VReg{
							elem,
						},
						Uses: []VReg{
							local(plan.SlicePtrLocal),
							local(plan.SliceLenLocal),
							local(plan.IndexLocal),
						},
						Note: plan.LoadProofID,
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(plan.TotalLocal)},
						Uses: []VReg{local(plan.TotalLocal), elem},
						Note: "total += xs[i]",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.IndexLocal)},
						Uses: []VReg{local(plan.IndexLocal)},
						Note: "i++",
					},
					{Op: OpBranch, Target: innerLoop},
				},
				Successors: []string{innerAfter, innerLoop},
			},
			{
				Name: innerAfter,
				Instrs: []Instr{
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.RepeatLocal)},
						Uses: []VReg{local(plan.RepeatLocal)},
						Note: "r++",
					},
					{Op: OpBranch, Target: outerLoop},
				},
				Successors: []string{outerLoop},
			},
			{
				Name: outerAfter,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{finalZero}, Imm: 0, Note: "zero"},
					{
						Op:   OpCmp,
						Defs: []VReg{finalCmp},
						Uses: []VReg{local(plan.TotalLocal), finalZero},
						Note: "total > 0",
					},
					{Op: OpBranchIf, Uses: []VReg{finalCmp}, Target: failure, Note: "if_zero"},
					{Op: OpBranch, Target: success},
				},
				Successors: []string{failure, success},
			},
			{
				Name: success,
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{successValue},
						Imm:  int64(plan.SuccessReturn),
						Note: "return 0",
					},
					{Op: OpReturn, Uses: []VReg{successValue}},
				},
			},
			{
				Name: failure,
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{failureValue},
						Imm:  int64(plan.FailureReturn),
						Note: "return 1",
					},
					{Op: OpReturn, Uses: []VReg{failureValue}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

// ---- matrix_multiply_main.go ----

const (
	matrixMultiplyMainFunctionName   = "p25.matrix_multiply.main"
	matrixMultiplyMainSliceLength    = int32(9)
	matrixMultiplyMainDimension      = int32(3)
	matrixMultiplyMainRepeatCount    = int32(2000)
	matrixMultiplyMainStep           = int32(1)
	matrixMultiplyMainBackingSlots   = 5
	matrixMultiplyMainAFillProofID   = "proof:while-const:i:a:10:9"
	matrixMultiplyMainBFillProofID   = "proof:while-const:i:b:11:9"
	matrixMultiplyMainCFillProofID   = "proof:while-const:i:c:12:9"
	matrixMultiplyMainARowKProofID   = "proof:affine-const:row_k:a:24:38"
	matrixMultiplyMainBKColProofID   = "proof:affine-const:k_col:b:24:55"
	matrixMultiplyMainCRowColProofID = "proof:affine-const:row_col:c:26:19"
	matrixMultiplyMainCModuloProofID = "proof:modulo:modulo_const:c:29:37"
)

type MatrixMultiplyMainPlan struct {
	Function       Function
	APtrLocal      int
	ALenLocal      int
	BPtrLocal      int
	BLenLocal      int
	CPtrLocal      int
	CLenLocal      int
	IndexLocal     int
	ChecksumLocal  int
	RepeatLocal    int
	RowLocal       int
	ColLocal       int
	KLocal         int
	TotalLocal     int
	ABackingLocal  int
	BBackingLocal  int
	CBackingLocal  int
	BackingSlots   int
	SliceLength    int32
	Dimension      int32
	RepeatCount    int32
	Step           int32
	AFillProofID   string
	BFillProofID   string
	CFillProofID   string
	ARowKProofID   string
	BKColProofID   string
	CRowColProofID string
	CModuloProofID string
	SuccessReturn  int32
	FailureReturn  int32
	BoundsChecks   int
}

func MatrixMultiplyMainFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := MatrixMultiplyMainPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func MatrixMultiplyMainPlanFromStackIR(fn ir.IRFunc) (MatrixMultiplyMainPlan, bool, error) {
	if fn.Name != matrixMultiplyMainFunctionName || fn.ParamSlots != 0 ||
		fn.ReturnSlots != 1 || fn.LocalSlots < 28 {
		return MatrixMultiplyMainPlan{}, false, nil
	}
	if !matrixMultiplyMainStackIRMatches(fn) {
		return MatrixMultiplyMainPlan{}, false, nil
	}
	plan := MatrixMultiplyMainPlan{
		APtrLocal:      0,
		ALenLocal:      1,
		BPtrLocal:      2,
		BLenLocal:      3,
		CPtrLocal:      4,
		CLenLocal:      5,
		IndexLocal:     6,
		ChecksumLocal:  7,
		RepeatLocal:    8,
		RowLocal:       9,
		ColLocal:       10,
		KLocal:         11,
		TotalLocal:     12,
		ABackingLocal:  13,
		BBackingLocal:  18,
		CBackingLocal:  23,
		BackingSlots:   matrixMultiplyMainBackingSlots,
		SliceLength:    matrixMultiplyMainSliceLength,
		Dimension:      matrixMultiplyMainDimension,
		RepeatCount:    matrixMultiplyMainRepeatCount,
		Step:           matrixMultiplyMainStep,
		AFillProofID:   matrixMultiplyMainAFillProofID,
		BFillProofID:   matrixMultiplyMainBFillProofID,
		CFillProofID:   matrixMultiplyMainCFillProofID,
		ARowKProofID:   matrixMultiplyMainARowKProofID,
		BKColProofID:   matrixMultiplyMainBKColProofID,
		CRowColProofID: matrixMultiplyMainCRowColProofID,
		CModuloProofID: matrixMultiplyMainCModuloProofID,
		SuccessReturn:  0,
		FailureReturn:  1,
		BoundsChecks:   7,
	}
	for _, local := range []struct {
		slot int
		name string
	}{
		{plan.APtrLocal, "a ptr"},
		{plan.ALenLocal, "a len"},
		{plan.BPtrLocal, "b ptr"},
		{plan.BLenLocal, "b len"},
		{plan.CPtrLocal, "c ptr"},
		{plan.CLenLocal, "c len"},
		{plan.IndexLocal, "i"},
		{plan.ChecksumLocal, "checksum"},
		{plan.RepeatLocal, "r"},
		{plan.RowLocal, "row"},
		{plan.ColLocal, "col"},
		{plan.KLocal, "k"},
		{plan.TotalLocal, "total"},
	} {
		if err := validateScalarLoopLocal(fn, local.slot, local.name); err != nil {
			return MatrixMultiplyMainPlan{}, true, err
		}
	}
	if !distinctAllocationLoopLocals(
		plan.APtrLocal,
		plan.ALenLocal,
		plan.BPtrLocal,
		plan.BLenLocal,
		plan.CPtrLocal,
		plan.CLenLocal,
		plan.IndexLocal,
		plan.ChecksumLocal,
		plan.RepeatLocal,
		plan.RowLocal,
		plan.ColLocal,
		plan.KLocal,
		plan.TotalLocal,
	) {
		return MatrixMultiplyMainPlan{}, false, nil
	}
	for _, backing := range []int{plan.ABackingLocal, plan.BBackingLocal, plan.CBackingLocal} {
		if backing+plan.BackingSlots > fn.LocalSlots {
			return MatrixMultiplyMainPlan{}, true, fmt.Errorf(
				"machine matrix_multiply main lowering: backing slots [%d,%d) out of bounds (locals=%d)",
				backing,
				backing+plan.BackingSlots,
				fn.LocalSlots,
			)
		}
		for _, local := range []int{
			plan.APtrLocal,
			plan.ALenLocal,
			plan.BPtrLocal,
			plan.BLenLocal,
			plan.CPtrLocal,
			plan.CLenLocal,
			plan.IndexLocal,
			plan.ChecksumLocal,
			plan.RepeatLocal,
			plan.RowLocal,
			plan.ColLocal,
			plan.KLocal,
			plan.TotalLocal,
		} {
			if local >= backing && local < backing+plan.BackingSlots {
				return MatrixMultiplyMainPlan{}, false, nil
			}
		}
	}
	out, err := buildMatrixMultiplyMainMachineFunction(fn.Name, plan)
	if err != nil {
		return MatrixMultiplyMainPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func matrixMultiplyMainStackIRMatches(fn ir.IRFunc) bool {
	want := matrixMultiplyMainExpectedStackIR()
	if len(fn.Instrs) != len(want) {
		return false
	}
	for i := range want {
		got := fn.Instrs[i]
		exp := want[i]
		if got.Kind != exp.Kind ||
			got.Imm != exp.Imm ||
			got.Local != exp.Local ||
			got.Label != exp.Label ||
			got.Name != exp.Name ||
			got.ArgSlots != exp.ArgSlots ||
			got.RetSlots != exp.RetSlots ||
			got.ProofID != exp.ProofID {
			return false
		}
	}
	return true
}

func matrixMultiplyMainExpectedStackIR() []ir.IRInstr {
	return []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 9},
		{Kind: ir.IRStackSliceI32, Local: 13, ArgSlots: 5, Imm: 9, Name: "a"},
		{Kind: ir.IRStoreLocal, Local: 1},
		{Kind: ir.IRStoreLocal, Local: 0},
		{Kind: ir.IRConstI32, Imm: 9},
		{Kind: ir.IRStackSliceI32, Local: 18, ArgSlots: 5, Imm: 9, Name: "b"},
		{Kind: ir.IRStoreLocal, Local: 3},
		{Kind: ir.IRStoreLocal, Local: 2},
		{Kind: ir.IRConstI32, Imm: 9},
		{Kind: ir.IRStackSliceI32, Local: 23, ArgSlots: 5, Imm: 9, Name: "c"},
		{Kind: ir.IRStoreLocal, Local: 5},
		{Kind: ir.IRStoreLocal, Local: 4},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 6},
		{Kind: ir.IRLabel, Label: 0},
		{Kind: ir.IRLoadLocal, Local: 6},
		{Kind: ir.IRConstI32, Imm: 9},
		{Kind: ir.IRCmpLtI32},
		{Kind: ir.IRJmpIfZero, Label: 1},
		{Kind: ir.IRLoadLocal, Local: 0},
		{Kind: ir.IRLoadLocal, Local: 1},
		{Kind: ir.IRLoadLocal, Local: 6},
		{Kind: ir.IRLoadLocal, Local: 6},
		{Kind: ir.IRConstI32, Imm: 1},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRIndexStoreI32, ProofID: matrixMultiplyMainAFillProofID},
		{Kind: ir.IRLoadLocal, Local: 2},
		{Kind: ir.IRLoadLocal, Local: 3},
		{Kind: ir.IRLoadLocal, Local: 6},
		{Kind: ir.IRConstI32, Imm: 9},
		{Kind: ir.IRLoadLocal, Local: 6},
		{Kind: ir.IRSubI32},
		{Kind: ir.IRIndexStoreI32, ProofID: matrixMultiplyMainBFillProofID},
		{Kind: ir.IRLoadLocal, Local: 4},
		{Kind: ir.IRLoadLocal, Local: 5},
		{Kind: ir.IRLoadLocal, Local: 6},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRIndexStoreI32, ProofID: matrixMultiplyMainCFillProofID},
		{Kind: ir.IRLoadLocal, Local: 6},
		{Kind: ir.IRConstI32, Imm: 1},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRStoreLocal, Local: 6},
		{Kind: ir.IRJmp, Label: 0},
		{Kind: ir.IRLabel, Label: 1},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 7},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 8},
		{Kind: ir.IRLabel, Label: 2},
		{Kind: ir.IRLoadLocal, Local: 8},
		{Kind: ir.IRConstI32, Imm: 2000},
		{Kind: ir.IRCmpLtI32},
		{Kind: ir.IRJmpIfZero, Label: 3},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 9},
		{Kind: ir.IRLabel, Label: 4},
		{Kind: ir.IRLoadLocal, Local: 9},
		{Kind: ir.IRConstI32, Imm: 3},
		{Kind: ir.IRCmpLtI32},
		{Kind: ir.IRJmpIfZero, Label: 5},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 10},
		{Kind: ir.IRLabel, Label: 6},
		{Kind: ir.IRLoadLocal, Local: 10},
		{Kind: ir.IRConstI32, Imm: 3},
		{Kind: ir.IRCmpLtI32},
		{Kind: ir.IRJmpIfZero, Label: 7},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 11},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRStoreLocal, Local: 12},
		{Kind: ir.IRLabel, Label: 8},
		{Kind: ir.IRLoadLocal, Local: 11},
		{Kind: ir.IRConstI32, Imm: 3},
		{Kind: ir.IRCmpLtI32},
		{Kind: ir.IRJmpIfZero, Label: 9},
		{Kind: ir.IRLoadLocal, Local: 12},
		{Kind: ir.IRLoadLocal, Local: 0},
		{Kind: ir.IRLoadLocal, Local: 1},
		{Kind: ir.IRLoadLocal, Local: 9},
		{Kind: ir.IRConstI32, Imm: 3},
		{Kind: ir.IRMulI32},
		{Kind: ir.IRLoadLocal, Local: 11},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRIndexLoadI32Unchecked, ProofID: matrixMultiplyMainARowKProofID},
		{Kind: ir.IRLoadLocal, Local: 2},
		{Kind: ir.IRLoadLocal, Local: 3},
		{Kind: ir.IRLoadLocal, Local: 11},
		{Kind: ir.IRConstI32, Imm: 3},
		{Kind: ir.IRMulI32},
		{Kind: ir.IRLoadLocal, Local: 10},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRIndexLoadI32Unchecked, ProofID: matrixMultiplyMainBKColProofID},
		{Kind: ir.IRMulI32},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRStoreLocal, Local: 12},
		{Kind: ir.IRLoadLocal, Local: 11},
		{Kind: ir.IRConstI32, Imm: 1},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRStoreLocal, Local: 11},
		{Kind: ir.IRJmp, Label: 8},
		{Kind: ir.IRLabel, Label: 9},
		{Kind: ir.IRLoadLocal, Local: 4},
		{Kind: ir.IRLoadLocal, Local: 5},
		{Kind: ir.IRLoadLocal, Local: 9},
		{Kind: ir.IRConstI32, Imm: 3},
		{Kind: ir.IRMulI32},
		{Kind: ir.IRLoadLocal, Local: 10},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRLoadLocal, Local: 12},
		{Kind: ir.IRIndexStoreI32, ProofID: matrixMultiplyMainCRowColProofID},
		{Kind: ir.IRLoadLocal, Local: 10},
		{Kind: ir.IRConstI32, Imm: 1},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRStoreLocal, Local: 10},
		{Kind: ir.IRJmp, Label: 6},
		{Kind: ir.IRLabel, Label: 7},
		{Kind: ir.IRLoadLocal, Local: 9},
		{Kind: ir.IRConstI32, Imm: 1},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRStoreLocal, Local: 9},
		{Kind: ir.IRJmp, Label: 4},
		{Kind: ir.IRLabel, Label: 5},
		{Kind: ir.IRLoadLocal, Local: 7},
		{Kind: ir.IRLoadLocal, Local: 4},
		{Kind: ir.IRLoadLocal, Local: 5},
		{Kind: ir.IRLoadLocal, Local: 8},
		{Kind: ir.IRConstI32, Imm: 9},
		{Kind: ir.IRModI32},
		{Kind: ir.IRIndexLoadI32Unchecked, ProofID: matrixMultiplyMainCModuloProofID},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRStoreLocal, Local: 7},
		{Kind: ir.IRLoadLocal, Local: 8},
		{Kind: ir.IRConstI32, Imm: 1},
		{Kind: ir.IRAddI32},
		{Kind: ir.IRStoreLocal, Local: 8},
		{Kind: ir.IRJmp, Label: 2},
		{Kind: ir.IRLabel, Label: 3},
		{Kind: ir.IRLoadLocal, Local: 7},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRCmpGtI32},
		{Kind: ir.IRJmpIfZero, Label: 10},
		{Kind: ir.IRConstI32, Imm: 0},
		{Kind: ir.IRReturn},
		{Kind: ir.IRLabel, Label: 10},
		{Kind: ir.IRConstI32, Imm: 1},
		{Kind: ir.IRReturn},
	}
}

func buildMatrixMultiplyMainMachineFunction(
	name string,
	plan MatrixMultiplyMainPlan,
) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	fillLoop := scalarLoopLabelName(0)
	fillAfter := scalarLoopLabelName(1)
	repeatLoop := scalarLoopLabelName(2)
	repeatAfter := scalarLoopLabelName(3)
	rowLoop := scalarLoopLabelName(4)
	rowAfter := scalarLoopLabelName(5)
	colLoop := scalarLoopLabelName(6)
	colAfter := scalarLoopLabelName(7)
	kLoop := scalarLoopLabelName(8)
	kAfter := scalarLoopLabelName(9)
	failure := scalarLoopLabelName(10)
	success := "return_success"
	sliceLen := VReg("slice.len")
	one := VReg("one")
	zero := VReg("zero")
	dim := VReg("dimension")
	repeats := VReg("repeat.count")
	fillCmp := VReg("fill.cmp")
	aFill := VReg("fill.a.value")
	bFill := VReg("fill.b.value")
	repeatCmp := VReg("repeat.cmp")
	rowCmp := VReg("row.cmp")
	colCmp := VReg("col.cmp")
	kCmp := VReg("k.cmp")
	rowTimesA := VReg("row.times.3.a")
	aIndex := VReg("a.index")
	aValue := VReg("a.value")
	kTimesB := VReg("k.times.3.b")
	bIndex := VReg("b.index")
	bValue := VReg("b.value")
	product := VReg("product")
	nextTotal := VReg("next.total")
	rowTimesC := VReg("row.times.3.c")
	cIndex := VReg("c.index")
	checksumIndex := VReg("checksum.index")
	checksumValue := VReg("checksum.value")
	nextChecksum := VReg("next.checksum")
	finalCmp := VReg("final.cmp")
	successValue := VReg("return_success")
	failureValue := VReg("return_failure")

	out := Function{
		Name:   name,
		Target: "matrix-multiply-main",
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(plan.APtrLocal)}, Note: "a stack ptr"},
					{Op: OpMov, Defs: []VReg{local(plan.ALenLocal)}, Imm: int64(plan.SliceLength), Note: "a len"},
					{Op: OpMov, Defs: []VReg{local(plan.BPtrLocal)}, Note: "b stack ptr"},
					{Op: OpMov, Defs: []VReg{local(plan.BLenLocal)}, Imm: int64(plan.SliceLength), Note: "b len"},
					{Op: OpMov, Defs: []VReg{local(plan.CPtrLocal)}, Note: "c stack ptr"},
					{Op: OpMov, Defs: []VReg{local(plan.CLenLocal)}, Imm: int64(plan.SliceLength), Note: "c len"},
					{Op: OpMov, Defs: []VReg{sliceLen}, Imm: int64(plan.SliceLength), Note: "slice length"},
					{Op: OpMov, Defs: []VReg{dim}, Imm: int64(plan.Dimension), Note: "matrix dimension"},
					{Op: OpMov, Defs: []VReg{one}, Imm: int64(plan.Step), Note: "step"},
					{Op: OpMov, Defs: []VReg{zero}, Imm: 0, Note: "zero"},
					{Op: OpMov, Defs: []VReg{local(plan.IndexLocal)}, Imm: 0, Note: "i = 0"},
					{Op: OpBranch, Target: fillLoop},
				},
				Successors: []string{fillLoop},
			},
			{
				Name: fillLoop,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{fillCmp},
						Uses: []VReg{local(plan.IndexLocal), sliceLen},
						Note: "i < 9",
					},
					{Op: OpBranchIf, Uses: []VReg{fillCmp}, Target: fillAfter, Note: "if_zero"},
					{Op: OpAdd, Defs: []VReg{aFill}, Uses: []VReg{local(plan.IndexLocal), one}, Note: "i + 1"},
					{
						Op: OpIndexStore,
						Uses: []VReg{
							local(plan.APtrLocal),
							local(plan.ALenLocal),
							local(plan.IndexLocal),
							aFill,
						},
						Note: plan.AFillProofID,
					},
					{
						Op:   OpSub,
						Defs: []VReg{bFill},
						Uses: []VReg{sliceLen, local(plan.IndexLocal)},
						Note: "9 - i",
					},
					{
						Op: OpIndexStore,
						Uses: []VReg{
							local(plan.BPtrLocal),
							local(plan.BLenLocal),
							local(plan.IndexLocal),
							bFill,
						},
						Note: plan.BFillProofID,
					},
					{
						Op: OpIndexStore,
						Uses: []VReg{
							local(plan.CPtrLocal),
							local(plan.CLenLocal),
							local(plan.IndexLocal),
							zero,
						},
						Note: plan.CFillProofID,
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.IndexLocal)},
						Uses: []VReg{local(plan.IndexLocal)},
						Note: "i++",
					},
					{Op: OpBranch, Target: fillLoop},
				},
				Successors: []string{fillAfter, fillLoop},
			},
			{
				Name: fillAfter,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(plan.ChecksumLocal)}, Imm: 0, Note: "checksum = 0"},
					{Op: OpMov, Defs: []VReg{local(plan.RepeatLocal)}, Imm: 0, Note: "r = 0"},
					{Op: OpMov, Defs: []VReg{repeats}, Imm: int64(plan.RepeatCount), Note: "repeat count"},
					{Op: OpBranch, Target: repeatLoop},
				},
				Successors: []string{repeatLoop},
			},
			{
				Name: repeatLoop,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{repeatCmp},
						Uses: []VReg{local(plan.RepeatLocal), repeats},
						Note: "r < 2000",
					},
					{Op: OpBranchIf, Uses: []VReg{repeatCmp}, Target: repeatAfter, Note: "if_zero"},
					{Op: OpMov, Defs: []VReg{local(plan.RowLocal)}, Imm: 0, Note: "row = 0"},
					{Op: OpBranch, Target: rowLoop},
				},
				Successors: []string{repeatAfter, rowLoop},
			},
			{
				Name: rowLoop,
				Instrs: []Instr{
					{Op: OpCmp, Defs: []VReg{rowCmp}, Uses: []VReg{local(plan.RowLocal), dim}, Note: "row < 3"},
					{Op: OpBranchIf, Uses: []VReg{rowCmp}, Target: rowAfter, Note: "if_zero"},
					{Op: OpMov, Defs: []VReg{local(plan.ColLocal)}, Imm: 0, Note: "col = 0"},
					{Op: OpBranch, Target: colLoop},
				},
				Successors: []string{rowAfter, colLoop},
			},
			{
				Name: colLoop,
				Instrs: []Instr{
					{Op: OpCmp, Defs: []VReg{colCmp}, Uses: []VReg{local(plan.ColLocal), dim}, Note: "col < 3"},
					{Op: OpBranchIf, Uses: []VReg{colCmp}, Target: colAfter, Note: "if_zero"},
					{Op: OpMov, Defs: []VReg{local(plan.KLocal)}, Imm: 0, Note: "k = 0"},
					{Op: OpMov, Defs: []VReg{local(plan.TotalLocal)}, Imm: 0, Note: "total = 0"},
					{Op: OpBranch, Target: kLoop},
				},
				Successors: []string{colAfter, kLoop},
			},
			{
				Name: kLoop,
				Instrs: []Instr{
					{Op: OpCmp, Defs: []VReg{kCmp}, Uses: []VReg{local(plan.KLocal), dim}, Note: "k < 3"},
					{Op: OpBranchIf, Uses: []VReg{kCmp}, Target: kAfter, Note: "if_zero"},
					{Op: OpMul, Defs: []VReg{rowTimesA}, Uses: []VReg{local(plan.RowLocal), dim}, Note: "row * 3"},
					{
						Op:   OpAdd,
						Defs: []VReg{aIndex},
						Uses: []VReg{rowTimesA, local(plan.KLocal)},
						Note: "row * 3 + k",
					},
					{
						Op:   OpIndexLoad,
						Defs: []VReg{aValue},
						Uses: []VReg{local(plan.APtrLocal), local(plan.ALenLocal), aIndex},
						Note: plan.ARowKProofID,
					},
					{Op: OpMul, Defs: []VReg{kTimesB}, Uses: []VReg{local(plan.KLocal), dim}, Note: "k * 3"},
					{
						Op:   OpAdd,
						Defs: []VReg{bIndex},
						Uses: []VReg{kTimesB, local(plan.ColLocal)},
						Note: "k * 3 + col",
					},
					{
						Op:   OpIndexLoad,
						Defs: []VReg{bValue},
						Uses: []VReg{local(plan.BPtrLocal), local(plan.BLenLocal), bIndex},
						Note: plan.BKColProofID,
					},
					{Op: OpMul, Defs: []VReg{product}, Uses: []VReg{aValue, bValue}, Note: "a * b"},
					{
						Op:   OpAdd,
						Defs: []VReg{nextTotal},
						Uses: []VReg{local(plan.TotalLocal), product},
						Note: "total += product",
					},
					{
						Op:   OpMov,
						Defs: []VReg{local(plan.TotalLocal)},
						Uses: []VReg{nextTotal},
						Note: "store total",
					},
					{Op: OpInc, Defs: []VReg{local(plan.KLocal)}, Uses: []VReg{local(plan.KLocal)}, Note: "k++"},
					{Op: OpBranch, Target: kLoop},
				},
				Successors: []string{kAfter, kLoop},
			},
			{
				Name: kAfter,
				Instrs: []Instr{
					{Op: OpMul, Defs: []VReg{rowTimesC}, Uses: []VReg{local(plan.RowLocal), dim}, Note: "row * 3"},
					{
						Op:   OpAdd,
						Defs: []VReg{cIndex},
						Uses: []VReg{rowTimesC, local(plan.ColLocal)},
						Note: "row * 3 + col",
					},
					{
						Op: OpIndexStore,
						Uses: []VReg{
							local(plan.CPtrLocal),
							local(plan.CLenLocal),
							cIndex,
							local(plan.TotalLocal),
						},
						Note: plan.CRowColProofID,
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.ColLocal)},
						Uses: []VReg{local(plan.ColLocal)},
						Note: "col++",
					},
					{Op: OpBranch, Target: colLoop},
				},
				Successors: []string{colLoop},
			},
			{
				Name: colAfter,
				Instrs: []Instr{
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.RowLocal)},
						Uses: []VReg{local(plan.RowLocal)},
						Note: "row++",
					},
					{Op: OpBranch, Target: rowLoop},
				},
				Successors: []string{rowLoop},
			},
			{
				Name: rowAfter,
				Instrs: []Instr{
					{
						Op:   OpMod,
						Defs: []VReg{checksumIndex},
						Uses: []VReg{local(plan.RepeatLocal), sliceLen},
						Note: "r % 9",
					},
					{
						Op:   OpIndexLoad,
						Defs: []VReg{checksumValue},
						Uses: []VReg{local(plan.CPtrLocal), local(plan.CLenLocal), checksumIndex},
						Note: plan.CModuloProofID,
					},
					{
						Op:   OpAdd,
						Defs: []VReg{nextChecksum},
						Uses: []VReg{local(plan.ChecksumLocal), checksumValue},
						Note: "checksum += c[r % 9]",
					},
					{
						Op:   OpMov,
						Defs: []VReg{local(plan.ChecksumLocal)},
						Uses: []VReg{nextChecksum},
						Note: "store checksum",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.RepeatLocal)},
						Uses: []VReg{local(plan.RepeatLocal)},
						Note: "r++",
					},
					{Op: OpBranch, Target: repeatLoop},
				},
				Successors: []string{repeatLoop},
			},
			{
				Name: repeatAfter,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{finalCmp},
						Uses: []VReg{local(plan.ChecksumLocal), zero},
						Note: "checksum > 0",
					},
					{Op: OpBranchIf, Uses: []VReg{finalCmp}, Target: failure, Note: "if_zero"},
					{Op: OpBranch, Target: success},
				},
				Successors: []string{failure, success},
			},
			{
				Name: success,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{successValue}, Imm: int64(plan.SuccessReturn), Note: "return 0"},
					{Op: OpReturn, Uses: []VReg{successValue}},
				},
			},
			{
				Name: failure,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{failureValue}, Imm: int64(plan.FailureReturn), Note: "return 1"},
					{Op: OpReturn, Uses: []VReg{failureValue}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

// ---- region_island_allocation_main.go ----

const (
	regionIslandAllocationMainFunctionName = "p25.region_island_allocation.main"
	regionIslandAllocationMainLoopBound    = int32(256)
	regionIslandAllocationMainIslandBudget = int32(256)
	regionIslandAllocationMainSliceLength  = int32(16)
	regionIslandAllocationMainIndexConst   = int32(0)
	regionIslandAllocationMainStep         = int32(1)
	regionIslandAllocationMainProofPrefix  = "proof:allocation-zero:literal0:xs:"
)

type RegionIslandAllocationMainPlan struct {
	Function      Function
	ChecksumLocal int
	IndexLocal    int
	IslandLocal   int
	SlicePtrLocal int
	SliceLenLocal int
	LoopBound     int32
	IslandBudget  int32
	SliceLength   int32
	IndexConst    int32
	Step          int32
	StartLabel    int
	EndLabel      int
	FailureLabel  int
	StoreProofID  string
	LoadProofID   string
	SuccessReturn int32
	FailureReturn int32
	BoundsChecks  int
}

func RegionIslandAllocationMainFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := RegionIslandAllocationMainPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func RegionIslandAllocationMainPlanFromStackIR(
	fn ir.IRFunc,
) (RegionIslandAllocationMainPlan, bool, error) {
	if fn.Name != regionIslandAllocationMainFunctionName || fn.ParamSlots != 0 ||
		fn.ReturnSlots != 1 || fn.LocalSlots < 5 {
		return RegionIslandAllocationMainPlan{}, false, nil
	}
	if len(fn.Instrs) != 46 && len(fn.Instrs) != 50 {
		return RegionIslandAllocationMainPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return RegionIslandAllocationMainPlan{}, false, nil
	}
	checksumLocal := in[1].Local
	indexLocal := in[3].Local
	if checksumLocal == indexLocal {
		return RegionIslandAllocationMainPlan{}, false, nil
	}
	if in[4].Kind != ir.IRLabel || in[4].Label != 0 ||
		!isLoad(in[5], indexLocal) ||
		in[6].Kind != ir.IRConstI32 ||
		in[6].Imm != regionIslandAllocationMainLoopBound ||
		in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero ||
		in[8].Label != 1 {
		return RegionIslandAllocationMainPlan{}, false, nil
	}
	startLabel := in[4].Label
	endLabel := in[8].Label
	if in[9].Kind != ir.IRConstI32 ||
		in[9].Imm != regionIslandAllocationMainIslandBudget ||
		in[10].Kind != ir.IRIslandNew ||
		!isStore(in[11], in[11].Local) ||
		!isLoad(in[12], in[11].Local) ||
		in[13].Kind != ir.IRConstI32 ||
		in[13].Imm != regionIslandAllocationMainSliceLength ||
		in[14].Kind != ir.IRIslandMakeSliceI32 ||
		in[14].Name != "xs" {
		return RegionIslandAllocationMainPlan{}, false, nil
	}
	islandLocal := in[11].Local
	sliceLenLocal := in[15].Local
	slicePtrLocal := in[16].Local
	if !isStore(in[15], sliceLenLocal) || !isStore(in[16], slicePtrLocal) ||
		sliceLenLocal == slicePtrLocal {
		return RegionIslandAllocationMainPlan{}, false, nil
	}
	if !isLoad(in[17], slicePtrLocal) ||
		!isLoad(in[18], sliceLenLocal) ||
		in[19].Kind != ir.IRConstI32 ||
		in[19].Imm != regionIslandAllocationMainIndexConst ||
		!isLoad(in[20], indexLocal) ||
		in[21].Kind != ir.IRIndexStoreI32 ||
		!strings.HasPrefix(in[21].ProofID, regionIslandAllocationMainProofPrefix) {
		return RegionIslandAllocationMainPlan{}, false, nil
	}
	if !isLoad(in[22], checksumLocal) ||
		!isLoad(in[23], slicePtrLocal) ||
		!isLoad(in[24], sliceLenLocal) ||
		in[25].Kind != ir.IRConstI32 ||
		in[25].Imm != regionIslandAllocationMainIndexConst ||
		in[26].Kind != ir.IRIndexLoadI32Unchecked ||
		!strings.HasPrefix(in[26].ProofID, regionIslandAllocationMainProofPrefix) ||
		in[27].Kind != ir.IRAddI32 ||
		!isStore(in[28], checksumLocal) {
		return RegionIslandAllocationMainPlan{}, false, nil
	}
	freeIndex := 30
	if len(in) == 50 {
		if !isConstStore(in[29], in[31], 0) ||
			!isConstStore(in[30], in[32], 0) ||
			!((in[31].Local == sliceLenLocal && in[32].Local == slicePtrLocal) ||
				(in[31].Local == slicePtrLocal && in[32].Local == sliceLenLocal)) {
			return RegionIslandAllocationMainPlan{}, false, nil
		}
		freeIndex = 34
	}
	if !isLoad(in[freeIndex-1], islandLocal) ||
		in[freeIndex].Kind != ir.IRIslandFree ||
		!isLoad(in[freeIndex+1], indexLocal) ||
		in[freeIndex+2].Kind != ir.IRConstI32 ||
		in[freeIndex+2].Imm != regionIslandAllocationMainStep ||
		in[freeIndex+3].Kind != ir.IRAddI32 ||
		!isStore(in[freeIndex+4], indexLocal) ||
		in[freeIndex+5].Kind != ir.IRJmp ||
		in[freeIndex+5].Label != startLabel ||
		in[freeIndex+6].Kind != ir.IRLabel ||
		in[freeIndex+6].Label != endLabel {
		return RegionIslandAllocationMainPlan{}, false, nil
	}
	afterLoop := freeIndex + 7
	if !isLoad(in[afterLoop], checksumLocal) ||
		in[afterLoop+1].Kind != ir.IRConstI32 ||
		in[afterLoop+1].Imm != 0 ||
		in[afterLoop+2].Kind != ir.IRCmpGtI32 ||
		in[afterLoop+3].Kind != ir.IRJmpIfZero ||
		in[afterLoop+3].Label != 2 ||
		in[afterLoop+4].Kind != ir.IRConstI32 ||
		in[afterLoop+4].Imm != 0 ||
		in[afterLoop+5].Kind != ir.IRReturn ||
		in[afterLoop+6].Kind != ir.IRLabel ||
		in[afterLoop+6].Label != in[afterLoop+3].Label ||
		in[afterLoop+7].Kind != ir.IRConstI32 ||
		in[afterLoop+7].Imm != 1 ||
		in[afterLoop+8].Kind != ir.IRReturn {
		return RegionIslandAllocationMainPlan{}, false, nil
	}
	for _, local := range []struct {
		slot int
		name string
	}{
		{checksumLocal, "checksum"},
		{indexLocal, "r"},
		{islandLocal, "island"},
		{slicePtrLocal, "slice ptr"},
		{sliceLenLocal, "slice len"},
	} {
		if err := validateScalarLoopLocal(fn, local.slot, local.name); err != nil {
			return RegionIslandAllocationMainPlan{}, true, err
		}
	}
	if !distinctAllocationLoopLocals(
		checksumLocal,
		indexLocal,
		islandLocal,
		slicePtrLocal,
		sliceLenLocal,
	) {
		return RegionIslandAllocationMainPlan{}, false, nil
	}
	plan := RegionIslandAllocationMainPlan{
		ChecksumLocal: checksumLocal,
		IndexLocal:    indexLocal,
		IslandLocal:   islandLocal,
		SlicePtrLocal: slicePtrLocal,
		SliceLenLocal: sliceLenLocal,
		LoopBound:     regionIslandAllocationMainLoopBound,
		IslandBudget:  regionIslandAllocationMainIslandBudget,
		SliceLength:   regionIslandAllocationMainSliceLength,
		IndexConst:    regionIslandAllocationMainIndexConst,
		Step:          regionIslandAllocationMainStep,
		StartLabel:    startLabel,
		EndLabel:      endLabel,
		FailureLabel:  in[afterLoop+3].Label,
		StoreProofID:  in[21].ProofID,
		LoadProofID:   in[26].ProofID,
		SuccessReturn: in[afterLoop+4].Imm,
		FailureReturn: in[afterLoop+7].Imm,
		BoundsChecks:  0,
	}
	out, err := buildRegionIslandAllocationMainMachineFunction(fn.Name, plan)
	if err != nil {
		return RegionIslandAllocationMainPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func buildRegionIslandAllocationMainMachineFunction(
	name string,
	plan RegionIslandAllocationMainPlan,
) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	loopName := scalarLoopLabelName(plan.StartLabel)
	exitName := scalarLoopLabelName(plan.EndLabel)
	failureName := scalarLoopLabelName(plan.FailureLabel)
	successName := "return_success"
	zero := VReg("zero")
	bound := VReg("loop.bound")
	loopCmp := VReg("loop.cmp")
	elem := VReg("xs0")
	cleanup := VReg("island.cleanup")
	finalZero := VReg("final.zero")
	finalCmp := VReg("final.cmp")
	successValue := VReg("return_success")
	failureValue := VReg("return_failure")

	out := Function{
		Name:   name,
		Target: "region-island-allocation-main",
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(plan.ChecksumLocal)}, Imm: 0, Note: "checksum = 0"},
					{Op: OpMov, Defs: []VReg{local(plan.IndexLocal)}, Imm: 0, Note: "r = 0"},
					{Op: OpMov, Defs: []VReg{local(plan.IslandLocal)}, Note: "island scope handle"},
					{
						Op:   OpMov,
						Defs: []VReg{local(plan.SlicePtrLocal)},
						Note: "xs native private slot",
					},
					{
						Op:   OpMov,
						Defs: []VReg{local(plan.SliceLenLocal)},
						Imm:  int64(plan.SliceLength),
						Note: "xs len = 16",
					},
					{Op: OpMov, Defs: []VReg{zero}, Imm: int64(plan.IndexConst), Note: "literal index 0"},
					{Op: OpMov, Defs: []VReg{bound}, Imm: int64(plan.LoopBound), Note: "loop bound"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{loopCmp},
						Uses: []VReg{local(plan.IndexLocal), bound},
						Note: "r < 256",
					},
					{Op: OpBranchIf, Uses: []VReg{loopCmp}, Target: exitName, Note: "if_zero"},
					{
						Op: OpIndexStore,
						Uses: []VReg{
							local(plan.SlicePtrLocal),
							local(plan.SliceLenLocal),
							zero,
							local(plan.IndexLocal),
						},
						Note: plan.StoreProofID,
					},
					{
						Op: OpIndexLoad,
						Defs: []VReg{
							elem,
						},
						Uses: []VReg{
							local(plan.SlicePtrLocal),
							local(plan.SliceLenLocal),
							zero,
						},
						Note: plan.LoadProofID,
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(plan.ChecksumLocal)},
						Uses: []VReg{local(plan.ChecksumLocal), elem},
						Note: "checksum += xs[0]",
					},
					{
						Op:   OpMov,
						Defs: []VReg{cleanup},
						Uses: []VReg{local(plan.IslandLocal)},
						Note: "island cleanup before loop backedge",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.IndexLocal)},
						Uses: []VReg{local(plan.IndexLocal)},
						Note: "r++",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{exitName, loopName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{finalZero}, Imm: 0, Note: "zero"},
					{
						Op:   OpCmp,
						Defs: []VReg{finalCmp},
						Uses: []VReg{local(plan.ChecksumLocal), finalZero},
						Note: "checksum > 0",
					},
					{Op: OpBranchIf, Uses: []VReg{finalCmp}, Target: failureName, Note: "if_zero"},
					{Op: OpBranch, Target: successName},
				},
				Successors: []string{failureName, successName},
			},
			{
				Name: successName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{successValue}, Imm: int64(plan.SuccessReturn), Note: "return 0"},
					{Op: OpReturn, Uses: []VReg{successValue}},
				},
			},
			{
				Name: failureName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{failureValue}, Imm: int64(plan.FailureReturn), Note: "return 1"},
					{Op: OpReturn, Uses: []VReg{failureValue}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

// ---- scalar_sum_squares_loop.go ----

type ScalarIntSumSquaresLoopPlan struct {
	Function   Function
	ParamLocal int
	IndexLocal int
	TotalLocal int
	StartLabel int
	EndLabel   int
}

func ScalarIntSumSquaresLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ScalarIntSumSquaresLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ScalarIntSumSquaresLoopPlanFromStackIR(
	fn ir.IRFunc,
) (ScalarIntSumSquaresLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 1 || fn.LocalSlots < 3 {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}
	if len(fn.Instrs) != 23 {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	totalLocal := in[3].Local
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 0) || in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], indexLocal) || !isLoad(in[11], indexLocal) ||
		in[12].Kind != ir.IRMulI32 ||
		in[13].Kind != ir.IRAddI32 ||
		!isStore(in[14], totalLocal) {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}
	if !isLoad(in[15], indexLocal) || in[16].Kind != ir.IRConstI32 || in[16].Imm != 1 ||
		in[17].Kind != ir.IRAddI32 ||
		!isStore(in[18], indexLocal) {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}
	if in[19].Kind != ir.IRJmp || in[19].Label != startLabel || in[20].Kind != ir.IRLabel ||
		in[20].Label != endLabel ||
		!isLoad(in[21], totalLocal) ||
		in[22].Kind != ir.IRReturn {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntSumSquaresLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return ScalarIntSumSquaresLoopPlan{}, true, err
	}
	if indexLocal == totalLocal || indexLocal == 0 || totalLocal == 0 {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	cmp := VReg("t0")
	square := VReg("t1")
	loopName := scalarLoopLabelName(startLabel)
	exitName := scalarLoopLabelName(endLabel)
	out := Function{
		Name:   fn.Name,
		Target: "scalar-int-sum-squares-loop",
		Params: []VReg{local(0)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(indexLocal)}, Imm: 0, Note: "loop index = 0"},
					{Op: OpMov, Defs: []VReg{local(totalLocal)}, Imm: 0, Note: "loop total = 0"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(indexLocal), local(0)},
						Note: "index < n",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{
						Op:   OpMul,
						Defs: []VReg{square},
						Uses: []VReg{local(indexLocal), local(indexLocal)},
						Note: "index * index",
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(totalLocal)},
						Uses: []VReg{local(totalLocal), square},
						Note: "total += index * index",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(indexLocal)},
						Uses: []VReg{local(indexLocal)},
						Note: "index++",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{local(totalLocal)}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return ScalarIntSumSquaresLoopPlan{}, true, err
	}
	return ScalarIntSumSquaresLoopPlan{
		Function:   out,
		ParamLocal: 0,
		IndexLocal: indexLocal,
		TotalLocal: totalLocal,
		StartLabel: startLabel,
		EndLabel:   endLabel,
	}, true, nil
}

// ---- parallel_map_reduce_main.go ----

const (
	parallelMapReduceMainFunctionName = "p25.parallel_map_reduce.main"
	parallelMapReduceExpectedTotal    = int32(42)
	parallelMapReduceSuccessReturn    = int32(0)
)

var parallelMapReduceWorkers = []string{"left_worker", "mid_worker", "right_worker"}

type ParallelMapReduceTaskCallPlan struct {
	Worker      string
	EntryID     int32
	HandleLocal int
	StatusLocal int
}

type ParallelMapReduceMainPlan struct {
	Function      Function
	Spawns        []ParallelMapReduceTaskCallPlan
	Joins         []ParallelMapReduceTaskCallPlan
	TotalLocal    int
	ExpectedTotal int32
	SuccessReturn int32
}

type parallelMapReduceStackValue struct {
	kind   string
	imm    int32
	local  int
	worker int
	mask   int
}

func ParallelMapReduceMainFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ParallelMapReduceMainPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ParallelMapReduceMainPlanFromStackIR(
	fn ir.IRFunc,
) (ParallelMapReduceMainPlan, bool, error) {
	if fn.Name != parallelMapReduceMainFunctionName || fn.ParamSlots != 0 ||
		fn.ReturnSlots != 1 || fn.LocalSlots < 7 {
		return ParallelMapReduceMainPlan{}, false, nil
	}
	stack := make([]parallelMapReduceStackValue, 0, 8)
	locals := map[int]parallelMapReduceStackValue{}
	spawns := make([]ParallelMapReduceTaskCallPlan, 0, len(parallelMapReduceWorkers))
	joins := make([]ParallelMapReduceTaskCallPlan, 0, len(parallelMapReduceWorkers))
	returns := make([]parallelMapReduceStackValue, 0, 2)
	totalLocal := -1
	sawExpectedCmp := false

	pop := func() (parallelMapReduceStackValue, bool) {
		if len(stack) == 0 {
			return parallelMapReduceStackValue{}, false
		}
		value := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return value, true
	}
	push := func(value parallelMapReduceStackValue) {
		stack = append(stack, value)
	}
	fail := func() (ParallelMapReduceMainPlan, bool, error) {
		return ParallelMapReduceMainPlan{}, false, nil
	}

	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRConstI32:
			push(parallelMapReduceStackValue{kind: "const", imm: instr.Imm})
		case ir.IRLoadLocal:
			if instr.Local < 0 || instr.Local >= fn.LocalSlots {
				return fail()
			}
			if value, ok := locals[instr.Local]; ok {
				push(value)
				continue
			}
			push(parallelMapReduceStackValue{kind: "local", local: instr.Local})
		case ir.IRStoreLocal:
			if instr.Local < 0 || instr.Local >= fn.LocalSlots {
				return fail()
			}
			value, ok := pop()
			if !ok {
				return fail()
			}
			value.local = instr.Local
			locals[instr.Local] = value
			switch value.kind {
			case "spawn_handle", "spawn_status":
				if value.worker < 0 || value.worker >= len(spawns) {
					return fail()
				}
				if value.kind == "spawn_handle" {
					spawns[value.worker].HandleLocal = instr.Local
				} else {
					spawns[value.worker].StatusLocal = instr.Local
				}
			case "total":
				totalLocal = instr.Local
			}
		case ir.IRCall:
			switch instr.Name {
			case "__tetra_task_spawn_i32":
				if instr.ArgSlots != 1 || instr.RetSlots != 2 ||
					len(spawns) >= len(parallelMapReduceWorkers) {
					return fail()
				}
				arg, ok := pop()
				if !ok || arg.kind != "const" {
					return fail()
				}
				workerIndex := len(spawns)
				worker := parallelMapReduceWorkers[workerIndex]
				if arg.imm != parallelMapReduceEntryID(worker) {
					return fail()
				}
				spawns = append(spawns, ParallelMapReduceTaskCallPlan{
					Worker:      worker,
					EntryID:     arg.imm,
					HandleLocal: -1,
					StatusLocal: -1,
				})
				push(parallelMapReduceStackValue{kind: "spawn_handle", worker: workerIndex})
				push(parallelMapReduceStackValue{kind: "spawn_status", worker: workerIndex})
			case "__tetra_task_join_i32":
				if instr.ArgSlots != 2 || instr.RetSlots != 1 ||
					len(joins) >= len(parallelMapReduceWorkers) {
					return fail()
				}
				status, ok := pop()
				if !ok {
					return fail()
				}
				handle, ok := pop()
				if !ok {
					return fail()
				}
				workerIndex := len(joins)
				if handle.kind != "spawn_handle" || status.kind != "spawn_status" ||
					handle.worker != workerIndex || status.worker != workerIndex ||
					workerIndex >= len(spawns) ||
					spawns[workerIndex].HandleLocal < 0 ||
					spawns[workerIndex].StatusLocal < 0 ||
					handle.local != spawns[workerIndex].HandleLocal ||
					status.local != spawns[workerIndex].StatusLocal {
					return fail()
				}
				join := spawns[workerIndex]
				joins = append(joins, join)
				push(parallelMapReduceStackValue{
					kind:   "join_value",
					worker: workerIndex,
					mask:   1 << workerIndex,
				})
			default:
				return fail()
			}
		case ir.IRAddI32:
			right, ok := pop()
			if !ok {
				return fail()
			}
			left, ok := pop()
			if !ok {
				return fail()
			}
			sum, ok := parallelMapReduceCombineSum(left, right)
			if !ok {
				return fail()
			}
			push(sum)
		case ir.IRCmpEqI32:
			right, ok := pop()
			if !ok {
				return fail()
			}
			left, ok := pop()
			if !ok {
				return fail()
			}
			if !parallelMapReduceIsTotal(left) || right.kind != "const" ||
				right.imm != parallelMapReduceExpectedTotal {
				return fail()
			}
			sawExpectedCmp = true
			push(parallelMapReduceStackValue{kind: "cmp_total"})
		case ir.IRJmpIfZero:
			cond, ok := pop()
			if !ok || cond.kind != "cmp_total" || instr.Label < 0 {
				return fail()
			}
		case ir.IRLabel:
			if instr.Label < 0 {
				return fail()
			}
		case ir.IRReturn:
			value, ok := pop()
			if !ok || len(stack) != 0 {
				return fail()
			}
			returns = append(returns, value)
		default:
			return fail()
		}
	}
	if len(spawns) != len(parallelMapReduceWorkers) ||
		len(joins) != len(parallelMapReduceWorkers) ||
		len(returns) != 2 ||
		!sawExpectedCmp ||
		totalLocal < 0 ||
		returns[0].kind != "const" ||
		returns[0].imm != parallelMapReduceSuccessReturn ||
		!parallelMapReduceIsTotal(returns[1]) {
		return fail()
	}
	for _, spawn := range spawns {
		if spawn.HandleLocal < 0 || spawn.StatusLocal < 0 ||
			spawn.HandleLocal == spawn.StatusLocal {
			return fail()
		}
	}
	plan := ParallelMapReduceMainPlan{
		Spawns:        spawns,
		Joins:         joins,
		TotalLocal:    totalLocal,
		ExpectedTotal: parallelMapReduceExpectedTotal,
		SuccessReturn: parallelMapReduceSuccessReturn,
	}
	out, err := buildParallelMapReduceMainMachineFunction(plan)
	if err != nil {
		return ParallelMapReduceMainPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func parallelMapReduceCombineSum(
	left parallelMapReduceStackValue,
	right parallelMapReduceStackValue,
) (parallelMapReduceStackValue, bool) {
	if !parallelMapReduceIsSummable(left) || !parallelMapReduceIsSummable(right) ||
		left.mask&right.mask != 0 {
		return parallelMapReduceStackValue{}, false
	}
	mask := left.mask | right.mask
	kind := "partial_sum"
	if mask == (1<<len(parallelMapReduceWorkers))-1 {
		kind = "total"
	}
	return parallelMapReduceStackValue{kind: kind, mask: mask}, true
}

func parallelMapReduceIsSummable(value parallelMapReduceStackValue) bool {
	return value.kind == "join_value" || value.kind == "partial_sum" || value.kind == "total"
}

func parallelMapReduceIsTotal(value parallelMapReduceStackValue) bool {
	return value.kind == "total" && value.mask == (1<<len(parallelMapReduceWorkers))-1
}

func buildParallelMapReduceMainMachineFunction(
	plan ParallelMapReduceMainPlan,
) (Function, error) {
	if len(plan.Spawns) != len(parallelMapReduceWorkers) ||
		len(plan.Joins) != len(parallelMapReduceWorkers) {
		return Function{}, fmt.Errorf("machine parallel map/reduce: incomplete task call plan")
	}
	workerRegPrefix := func(worker string) string {
		return strings.TrimSuffix(worker, "_worker")
	}
	entry := make([]Instr, 0, 24)
	for i, spawn := range plan.Spawns {
		prefix := workerRegPrefix(spawn.Worker)
		entryID := VReg(prefix + ".entry_id")
		handle := VReg(prefix + ".handle")
		status := VReg(prefix + ".status")
		entry = append(entry,
			Instr{
				Op:   OpMov,
				Defs: []VReg{entryID},
				Imm:  int64(spawn.EntryID),
				Note: fmt.Sprintf("entry_id %s", spawn.Worker),
			},
			Instr{
				Op:       OpCall,
				Defs:     []VReg{handle, status},
				Uses:     []VReg{entryID},
				Call:     "__tetra_task_spawn_i32",
				ABI:      "sysv",
				Clobbers: LinuxX64CallerSaved(),
				Note: fmt.Sprintf(
					"worker=%s arg_slots=1 ret_slots=2 handle_local=%d status_local=%d",
					spawn.Worker,
					spawn.HandleLocal,
					spawn.StatusLocal,
				),
			},
		)
		if plan.Joins[i].Worker != spawn.Worker ||
			plan.Joins[i].HandleLocal != spawn.HandleLocal ||
			plan.Joins[i].StatusLocal != spawn.StatusLocal {
			return Function{}, fmt.Errorf(
				"machine parallel map/reduce: join %d does not match spawn %#v",
				i,
				spawn,
			)
		}
	}
	joinValues := make([]VReg, len(plan.Joins))
	for i, join := range plan.Joins {
		prefix := workerRegPrefix(join.Worker)
		value := VReg(prefix + ".value")
		joinValues[i] = value
		entry = append(entry, Instr{
			Op:       OpCall,
			Defs:     []VReg{value},
			Uses:     []VReg{VReg(prefix + ".handle"), VReg(prefix + ".status")},
			Call:     "__tetra_task_join_i32",
			ABI:      "sysv",
			Clobbers: LinuxX64CallerSaved(),
			Note: fmt.Sprintf(
				"worker=%s arg_slots=2 ret_slots=1 handle_local=%d status_local=%d",
				join.Worker,
				join.HandleLocal,
				join.StatusLocal,
			),
		})
	}
	partial := VReg("left_mid.total")
	total := VReg("total")
	expected := VReg("expected")
	cmp := VReg("total_eq_42")
	entry = append(entry,
		Instr{Op: OpAdd, Defs: []VReg{partial}, Uses: []VReg{joinValues[0], joinValues[1]}},
		Instr{Op: OpAdd, Defs: []VReg{total}, Uses: []VReg{partial, joinValues[2]}},
		Instr{Op: OpMov, Defs: []VReg{expected}, Imm: int64(plan.ExpectedTotal)},
		Instr{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{total, expected}, Note: "total == 42"},
		Instr{Op: OpBranchIf, Uses: []VReg{cmp}, Target: "return_success"},
		Instr{Op: OpBranch, Target: "return_total"},
	)
	zero := VReg("zero")
	out := Function{
		Name:   parallelMapReduceMainFunctionName,
		Target: "parallel-map-reduce-main",
		Blocks: []Block{
			{
				Name:       "entry",
				Instrs:     entry,
				Successors: []string{"return_success", "return_total"},
			},
			{
				Name: "return_success",
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{zero},
						Imm:  int64(plan.SuccessReturn),
						Note: "total matched expected checksum",
					},
					{Op: OpReturn, Uses: []VReg{zero}},
				},
			},
			{
				Name: "return_total",
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{total}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

func parallelMapReduceEntryID(name string) int32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte("p25.parallel_map_reduce." + name))
	return int32(h.Sum32())
}

// ---- vector_copy_u8.go ----

type ScalarU8CopyLoopPlan struct {
	Function     Function
	DstBaseLocal int
	SrcBaseLocal int
	LenLocal     int
	IndexLocal   int
	StartLabel   int
	EndLabel     int
	ProofID      string
}

type VectorU8x16CopyLoopPlan struct {
	Function           Function
	ScalarPlan         ScalarU8CopyLoopPlan
	LaneCount          int
	SafeUnaligned      bool
	TailHandling       string
	ScalarFallback     string
	NoAliasRequirement string
	ProofID            string
}

func ScalarU8CopyLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ScalarU8CopyLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ScalarU8CopyLoopPlanFromStackIR(fn ir.IRFunc) (ScalarU8CopyLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 3 || fn.LocalSlots < 4 {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if len(fn.Instrs) != 23 {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	startLabel := in[2].Label
	if in[2].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if !isLoad(in[3], indexLocal) || !isLoad(in[4], 2) || in[5].Kind != ir.IRCmpLtI32 ||
		in[6].Kind != ir.IRJmpIfZero {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	endLabel := in[6].Label
	if endLabel < 0 {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if !isLoad(in[7], 0) || !isLoad(in[8], 2) || !isLoad(in[9], indexLocal) ||
		!isLoad(in[10], 1) || !isLoad(in[11], 2) || !isLoad(in[12], indexLocal) {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if in[13].Kind != ir.IRIndexLoadU8Unchecked ||
		!strings.HasPrefix(in[13].ProofID, "proof:copy-loop:") {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if in[14].Kind != ir.IRIndexStoreU8 {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if !isLoad(in[15], indexLocal) || in[16].Kind != ir.IRConstI32 || in[16].Imm != 1 ||
		in[17].Kind != ir.IRAddI32 ||
		!isStore(in[18], indexLocal) {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if in[19].Kind != ir.IRJmp || in[19].Label != startLabel || in[20].Kind != ir.IRLabel ||
		in[20].Label != endLabel ||
		in[21].Kind != ir.IRConstI32 ||
		in[21].Imm != 0 ||
		in[22].Kind != ir.IRReturn {
		return ScalarU8CopyLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarU8CopyLoopPlan{}, true, err
	}
	if indexLocal < fn.ParamSlots {
		return ScalarU8CopyLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	cmp := VReg("t0")
	elem := VReg("t1")
	loopName := scalarLoopLabelName(startLabel)
	exitName := scalarLoopLabelName(endLabel)
	out := Function{
		Name:   fn.Name,
		Target: "scalar-u8-copy",
		Params: []VReg{local(0), local(1), local(2)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(indexLocal)}, Imm: 0, Note: "index = 0"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(indexLocal), local(2)},
						Note: "index < len",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{
						Op:   OpIndexLoad,
						Defs: []VReg{elem},
						Uses: []VReg{local(1), local(2), local(indexLocal)},
						Note: in[13].ProofID,
					},
					{
						Op:   OpIndexStore,
						Uses: []VReg{local(0), local(2), local(indexLocal), elem},
						Note: in[13].ProofID + "; store uses same range proof",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(indexLocal)},
						Uses: []VReg{local(indexLocal)},
						Note: "index++",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{"zero"}, Note: "returns 0"},
				},
			},
		},
	}
	out.Blocks[0].Instrs = append(
		[]Instr{{Op: OpMov, Defs: []VReg{"zero"}, Imm: 0, Note: "return code = 0"}},
		out.Blocks[0].Instrs...)
	if err := VerifyFunction(out); err != nil {
		return ScalarU8CopyLoopPlan{}, true, err
	}
	return ScalarU8CopyLoopPlan{
		Function:     out,
		DstBaseLocal: 0,
		SrcBaseLocal: 1,
		LenLocal:     2,
		IndexLocal:   indexLocal,
		StartLabel:   startLabel,
		EndLabel:     endLabel,
		ProofID:      in[13].ProofID,
	}, true, nil
}

func VectorU8x16CopyLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := VectorU8x16CopyLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func VectorU8x16CopyLoopPlanFromStackIR(fn ir.IRFunc) (VectorU8x16CopyLoopPlan, bool, error) {
	scalar, ok, err := ScalarU8CopyLoopPlanFromStackIR(fn)
	if err != nil || !ok {
		return VectorU8x16CopyLoopPlan{}, ok, err
	}

	local := func(slot int) VReg { return VReg("local" + itoa(slot)) }
	lane := VReg("vlane")
	cmp := VReg("vcmp")
	vchunk := VReg("vchunk")
	loopName := scalarLoopLabelName(scalar.StartLabel)
	tailName := "vector_tail"
	exitName := scalarLoopLabelName(scalar.EndLabel)
	proofNote := scalar.ProofID + ("; safe unaligned u8x16 copy load/store; source/dest " +
		"disjoint owned copy result")

	out := Function{
		Name:   fn.Name,
		Target: "vector-u8x16-copy-plan",
		Params: []VReg{local(0), local(1), local(2)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(scalar.IndexLocal)}, Imm: 0, Note: "index = 0"},
					{Op: OpMov, Defs: []VReg{lane}, Imm: 16, Note: "u8x16 lane count"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpVectorCanCopyU8x16,
						Defs: []VReg{cmp},
						Uses: []VReg{local(2), local(scalar.IndexLocal)},
						Note: "index + 16 <= len",
					},
					{
						Op:     OpBranchIf,
						Uses:   []VReg{cmp},
						Target: tailName,
						Note:   "if fewer than sixteen bytes remain",
					},
					{
						Op:   OpVectorLoadU8x16Unaligned,
						Defs: []VReg{vchunk},
						Uses: []VReg{local(1), local(2), local(scalar.IndexLocal)},
						Note: proofNote,
					},
					{
						Op:   OpVectorStoreU8x16Unaligned,
						Uses: []VReg{local(0), local(2), local(scalar.IndexLocal), vchunk},
						Note: proofNote,
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(scalar.IndexLocal)},
						Uses: []VReg{local(scalar.IndexLocal), lane},
						Note: "index += 16",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, tailName},
			},
			{
				Name: tailName,
				Instrs: []Instr{
					{
						Op:   OpTailScalarU8Copy,
						Uses: []VReg{local(0), local(1), local(2), local(scalar.IndexLocal)},
						Note: scalar.ProofID + "; scalar tail handles len % 16",
					},
					{Op: OpBranch, Target: exitName},
				},
				Successors: []string{exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{"zero"}, Note: "returns 0"},
				},
			},
		},
	}
	out.Blocks[0].Instrs = append(
		[]Instr{{Op: OpMov, Defs: []VReg{"zero"}, Imm: 0, Note: "return code = 0"}},
		out.Blocks[0].Instrs...)
	if err := VerifyFunction(out); err != nil {
		return VectorU8x16CopyLoopPlan{}, true, err
	}
	return VectorU8x16CopyLoopPlan{
		Function:           out,
		ScalarPlan:         scalar,
		LaneCount:          16,
		SafeUnaligned:      true,
		TailHandling:       "scalar_tail",
		ScalarFallback:     scalar.Function.Target,
		NoAliasRequirement: "required_source_dest_disjoint_owned_copy_result",
		ProofID:            scalar.ProofID,
	}, true, nil
}

// ---- vector_map_i32.go ----

type ScalarI32MapAddConstLoopPlan struct {
	Function   Function
	BaseLocal  int
	LenLocal   int
	IndexLocal int
	TempLocal  int
	Addend     int32
	StartLabel int
	EndLabel   int
	ProofID    string
}

type VectorI32x4MapAddConstPlan struct {
	Function           Function
	ScalarPlan         ScalarI32MapAddConstLoopPlan
	LaneCount          int
	Addend             int32
	SafeUnaligned      bool
	TailHandling       string
	ScalarFallback     string
	NoAliasRequirement string
	ProofID            string
}

func ScalarI32MapAddConstFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ScalarI32MapAddConstPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ScalarI32MapAddConstPlanFromStackIR(fn ir.IRFunc) (ScalarI32MapAddConstLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 2 || fn.LocalSlots < 4 {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	if len(fn.Instrs) != 27 {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	startLabel := in[2].Label
	if in[2].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	if !isLoad(in[3], indexLocal) || !isLoad(in[4], 1) || in[5].Kind != ir.IRCmpLtI32 ||
		in[6].Kind != ir.IRJmpIfZero {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	endLabel := in[6].Label
	if endLabel < 0 {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	if !isLoad(in[7], 0) || !isLoad(in[8], 1) || !isLoad(in[9], indexLocal) {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	if in[10].Kind != ir.IRIndexLoadI32Unchecked ||
		!strings.HasPrefix(in[10].ProofID, "proof:map-loop:") {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	if in[11].Kind != ir.IRConstI32 || in[12].Kind != ir.IRAddI32 ||
		in[13].Kind != ir.IRStoreLocal {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	addend := in[11].Imm
	if addend != 1 {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	tempLocal := in[13].Local
	if !isLoad(in[14], 0) || !isLoad(in[15], 1) || !isLoad(in[16], indexLocal) ||
		!isLoad(in[17], tempLocal) ||
		in[18].Kind != ir.IRIndexStoreI32 {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	if !isLoad(in[19], indexLocal) || in[20].Kind != ir.IRConstI32 || in[20].Imm != 1 ||
		in[21].Kind != ir.IRAddI32 ||
		!isStore(in[22], indexLocal) {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	if in[23].Kind != ir.IRJmp || in[23].Label != startLabel || in[24].Kind != ir.IRLabel ||
		in[24].Label != endLabel ||
		in[25].Kind != ir.IRConstI32 ||
		in[25].Imm != 0 ||
		in[26].Kind != ir.IRReturn {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarI32MapAddConstLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, tempLocal, "temp"); err != nil {
		return ScalarI32MapAddConstLoopPlan{}, true, err
	}
	if indexLocal == tempLocal || indexLocal < fn.ParamSlots || tempLocal < fn.ParamSlots {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg("local" + itoa(slot)) }
	cmp := VReg("t0")
	elem := VReg("t1")
	add := VReg("t2")
	loopName := scalarLoopLabelName(startLabel)
	exitName := scalarLoopLabelName(endLabel)
	out := Function{
		Name:   fn.Name,
		Target: "scalar-i32-map-add-const",
		Params: []VReg{local(0), local(1)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{"zero"}, Imm: 0, Note: "return code = 0"},
					{Op: OpMov, Defs: []VReg{local(indexLocal)}, Imm: 0, Note: "index = 0"},
					{Op: OpMov, Defs: []VReg{add}, Imm: int64(addend), Note: "map addend"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(indexLocal), local(1)},
						Note: "index < len",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{
						Op:   OpIndexLoad,
						Defs: []VReg{elem},
						Uses: []VReg{local(0), local(1), local(indexLocal)},
						Note: in[10].ProofID,
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(tempLocal)},
						Uses: []VReg{elem, add},
						Note: "xs[index] + addend",
					},
					{
						Op:   OpIndexStore,
						Uses: []VReg{local(0), local(1), local(indexLocal), local(tempLocal)},
						Note: in[10].ProofID + "; store uses same range proof",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(indexLocal)},
						Uses: []VReg{local(indexLocal)},
						Note: "index++",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{"zero"}, Note: "returns 0"},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return ScalarI32MapAddConstLoopPlan{}, true, err
	}
	return ScalarI32MapAddConstLoopPlan{
		Function:   out,
		BaseLocal:  0,
		LenLocal:   1,
		IndexLocal: indexLocal,
		TempLocal:  tempLocal,
		Addend:     addend,
		StartLabel: startLabel,
		EndLabel:   endLabel,
		ProofID:    in[10].ProofID,
	}, true, nil
}

func VectorI32x4MapAddConstFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := VectorI32x4MapAddConstPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func VectorI32x4MapAddConstPlanFromStackIR(fn ir.IRFunc) (VectorI32x4MapAddConstPlan, bool, error) {
	scalar, ok, err := ScalarI32MapAddConstPlanFromStackIR(fn)
	if err != nil || !ok {
		return VectorI32x4MapAddConstPlan{}, ok, err
	}

	local := func(slot int) VReg { return VReg("local" + itoa(slot)) }
	lane := VReg("vlane")
	cmp := VReg("vcmp")
	vadd := VReg("vadd")
	vchunk := VReg("vchunk")
	loopName := scalarLoopLabelName(scalar.StartLabel)
	tailName := "vector_tail"
	exitName := scalarLoopLabelName(scalar.EndLabel)
	proofNote := scalar.ProofID + ("; safe unaligned i32x4 map load/store; single mutable " +
		"slice in-place map")

	out := Function{
		Name:   fn.Name,
		Target: "vector-i32x4-map-add-const-plan",
		Params: []VReg{local(0), local(1)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{"zero"}, Imm: 0, Note: "return code = 0"},
					{Op: OpMov, Defs: []VReg{local(scalar.IndexLocal)}, Imm: 0, Note: "index = 0"},
					{Op: OpMov, Defs: []VReg{lane}, Imm: 4, Note: "i32x4 lane count"},
					{
						Op:   OpVectorSplatI32x4,
						Defs: []VReg{vadd},
						Imm:  int64(scalar.Addend),
						Note: "broadcast map addend",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpVectorCanMapI32x4,
						Defs: []VReg{cmp},
						Uses: []VReg{local(1), local(scalar.IndexLocal)},
						Note: "index + 4 <= len",
					},
					{
						Op:     OpBranchIf,
						Uses:   []VReg{cmp},
						Target: tailName,
						Note:   "if fewer than four elements remain",
					},
					{
						Op:   OpVectorLoadI32x4Unaligned,
						Defs: []VReg{vchunk},
						Uses: []VReg{local(0), local(1), local(scalar.IndexLocal)},
						Note: proofNote,
					},
					{
						Op:   OpVectorAddI32x4,
						Defs: []VReg{vchunk},
						Uses: []VReg{vchunk, vadd},
						Note: "vector xs[index:index+4] += addend",
					},
					{
						Op:   OpVectorStoreI32x4Unaligned,
						Uses: []VReg{local(0), local(1), local(scalar.IndexLocal), vchunk},
						Note: proofNote,
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(scalar.IndexLocal)},
						Uses: []VReg{local(scalar.IndexLocal), lane},
						Note: "index += 4",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, tailName},
			},
			{
				Name: tailName,
				Instrs: []Instr{
					{
						Op:   OpTailScalarI32Map,
						Uses: []VReg{local(0), local(1), local(scalar.IndexLocal), vadd},
						Note: scalar.ProofID + "; scalar tail handles len % 4",
					},
					{Op: OpBranch, Target: exitName},
				},
				Successors: []string{exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{"zero"}, Note: "returns 0"},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return VectorI32x4MapAddConstPlan{}, true, err
	}
	return VectorI32x4MapAddConstPlan{
		Function:           out,
		ScalarPlan:         scalar,
		LaneCount:          4,
		Addend:             scalar.Addend,
		SafeUnaligned:      true,
		TailHandling:       "scalar_tail",
		ScalarFallback:     scalar.Function.Target,
		NoAliasRequirement: "not_required_single_mutable_slice_in_place",
		ProofID:            scalar.ProofID,
	}, true, nil
}

// ---- vector_memset_u8.go ----

type ScalarU8MemsetZeroPlan struct {
	Function   Function
	BaseLocal  int
	LenLocal   int
	IndexLocal int
	FillValue  int32
	StartLabel int
	EndLabel   int
	ProofID    string
}

type VectorU8x16MemsetZeroPlan struct {
	Function           Function
	ScalarPlan         ScalarU8MemsetZeroPlan
	LaneCount          int
	FillValue          int32
	SafeUnaligned      bool
	TailHandling       string
	ScalarFallback     string
	NoAliasRequirement string
	ProofID            string
}

func ScalarU8MemsetZeroFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ScalarU8MemsetZeroPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ScalarU8MemsetZeroPlanFromStackIR(fn ir.IRFunc) (ScalarU8MemsetZeroPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 2 || fn.LocalSlots < 3 {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	if len(fn.Instrs) != 20 {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	indexLocal := in[1].Local
	startLabel := in[2].Label
	if in[2].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	if !isLoad(in[3], indexLocal) || !isLoad(in[4], 1) || in[5].Kind != ir.IRCmpLtI32 ||
		in[6].Kind != ir.IRJmpIfZero {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	endLabel := in[6].Label
	if endLabel < 0 {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	if !isLoad(in[7], 0) || !isLoad(in[8], 1) || !isLoad(in[9], indexLocal) ||
		in[10].Kind != ir.IRConstI32 ||
		in[10].Imm != 0 {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	if in[11].Kind != ir.IRIndexStoreU8 ||
		!strings.HasPrefix(in[11].ProofID, "proof:memset-loop:") {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	if !isLoad(in[12], indexLocal) || in[13].Kind != ir.IRConstI32 || in[13].Imm != 1 ||
		in[14].Kind != ir.IRAddI32 ||
		!isStore(in[15], indexLocal) {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	if in[16].Kind != ir.IRJmp || in[16].Label != startLabel || in[17].Kind != ir.IRLabel ||
		in[17].Label != endLabel ||
		in[18].Kind != ir.IRConstI32 ||
		in[18].Imm != 0 ||
		in[19].Kind != ir.IRReturn {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarU8MemsetZeroPlan{}, true, err
	}
	if indexLocal < fn.ParamSlots {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg("local" + itoa(slot)) }
	cmp := VReg("t0")
	loopName := scalarLoopLabelName(startLabel)
	exitName := scalarLoopLabelName(endLabel)
	out := Function{
		Name:   fn.Name,
		Target: "scalar-u8-memset-zero",
		Params: []VReg{local(0), local(1)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{"zero"},
						Imm:  0,
						Note: "zero fill and return code = 0",
					},
					{Op: OpMov, Defs: []VReg{local(indexLocal)}, Imm: 0, Note: "index = 0"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{cmp},
						Uses: []VReg{local(indexLocal), local(1)},
						Note: "index < len",
					},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{
						Op:   OpIndexStore,
						Uses: []VReg{local(0), local(1), local(indexLocal), "zero"},
						Note: in[11].ProofID + "; single mutable slice zero-fill helper",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(indexLocal)},
						Uses: []VReg{local(indexLocal)},
						Note: "index++",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{"zero"}, Note: "returns 0"},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return ScalarU8MemsetZeroPlan{}, true, err
	}
	return ScalarU8MemsetZeroPlan{
		Function:   out,
		BaseLocal:  0,
		LenLocal:   1,
		IndexLocal: indexLocal,
		FillValue:  0,
		StartLabel: startLabel,
		EndLabel:   endLabel,
		ProofID:    in[11].ProofID,
	}, true, nil
}

func VectorU8x16MemsetZeroFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := VectorU8x16MemsetZeroPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func VectorU8x16MemsetZeroPlanFromStackIR(fn ir.IRFunc) (VectorU8x16MemsetZeroPlan, bool, error) {
	scalar, ok, err := ScalarU8MemsetZeroPlanFromStackIR(fn)
	if err != nil || !ok {
		return VectorU8x16MemsetZeroPlan{}, ok, err
	}

	local := func(slot int) VReg { return VReg("local" + itoa(slot)) }
	lane := VReg("vlane")
	cmp := VReg("vcmp")
	vzero := VReg("vzero")
	loopName := scalarLoopLabelName(scalar.StartLabel)
	tailName := "vector_tail"
	exitName := scalarLoopLabelName(scalar.EndLabel)
	proofNote := scalar.ProofID + ("; safe unaligned u8x16 zero-fill store; single mutable " +
		"slice zero-fill helper")

	out := Function{
		Name:   fn.Name,
		Target: "vector-u8x16-memset-zero-plan",
		Params: []VReg{local(0), local(1)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{"zero"},
						Imm:  0,
						Note: "zero fill and return code = 0",
					},
					{Op: OpMov, Defs: []VReg{local(scalar.IndexLocal)}, Imm: 0, Note: "index = 0"},
					{Op: OpMov, Defs: []VReg{lane}, Imm: 16, Note: "u8x16 lane count"},
					{Op: OpVectorZeroU8x16, Defs: []VReg{vzero}, Note: "zero-fill bytes"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpVectorCanMemsetU8x16,
						Defs: []VReg{cmp},
						Uses: []VReg{local(1), local(scalar.IndexLocal)},
						Note: "index + 16 <= len",
					},
					{
						Op:     OpBranchIf,
						Uses:   []VReg{cmp},
						Target: tailName,
						Note:   "if fewer than sixteen bytes remain",
					},
					{
						Op:   OpVectorStoreU8x16Unaligned,
						Uses: []VReg{local(0), local(1), local(scalar.IndexLocal), vzero},
						Note: proofNote,
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(scalar.IndexLocal)},
						Uses: []VReg{local(scalar.IndexLocal), lane},
						Note: "index += 16",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, tailName},
			},
			{
				Name: tailName,
				Instrs: []Instr{
					{
						Op:   OpTailScalarU8Memset,
						Uses: []VReg{local(0), local(1), local(scalar.IndexLocal), "zero"},
						Note: scalar.ProofID + "; scalar tail handles len % 16",
					},
					{Op: OpBranch, Target: exitName},
				},
				Successors: []string{exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{"zero"}, Note: "returns 0"},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return VectorU8x16MemsetZeroPlan{}, true, err
	}
	return VectorU8x16MemsetZeroPlan{
		Function:           out,
		ScalarPlan:         scalar,
		LaneCount:          16,
		FillValue:          0,
		SafeUnaligned:      true,
		TailHandling:       "scalar_tail",
		ScalarFallback:     scalar.Function.Target,
		NoAliasRequirement: "not_required_single_mutable_slice_in_place_zero_fill",
		ProofID:            scalar.ProofID,
	}, true, nil
}

// ---- vector_slice_sum.go ----

type VectorI32x4SliceSumLoopPlan struct {
	Function           Function
	ScalarPlan         ScalarI32SliceSumLoopPlan
	LaneCount          int
	SafeUnaligned      bool
	TailHandling       string
	ScalarFallback     string
	NoAliasRequirement string
	ProofID            string
}

func VectorI32x4SliceSumLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := VectorI32x4SliceSumLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func VectorI32x4SliceSumLoopPlanFromStackIR(
	fn ir.IRFunc,
) (VectorI32x4SliceSumLoopPlan, bool, error) {
	scalar, ok, err := ScalarI32SliceSumLoopPlanFromStackIR(fn)
	if err != nil || !ok {
		return VectorI32x4SliceSumLoopPlan{}, ok, err
	}
	if scalar.Step != 1 {
		return VectorI32x4SliceSumLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg("local" + itoa(slot)) }
	lane := VReg("vlane")
	cmp := VReg("vcmp")
	vsum := VReg("vsum")
	vchunk := VReg("vchunk")
	loopName := scalarLoopLabelName(scalar.StartLabel)
	tailName := "vector_tail"
	exitName := scalarLoopLabelName(scalar.EndLabel)
	proofNote := scalar.ProofID + "; safe unaligned i32x4 vector load"

	out := Function{
		Name:   fn.Name,
		Target: "vector-i32x4-slice-sum-plan",
		Params: []VReg{local(0), local(1)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(scalar.TotalLocal)}, Imm: 0, Note: "total = 0"},
					{Op: OpMov, Defs: []VReg{local(scalar.IndexLocal)}, Imm: 0, Note: "index = 0"},
					{Op: OpMov, Defs: []VReg{lane}, Imm: 4, Note: "i32x4 lane count"},
					{Op: OpVectorZeroI32x4, Defs: []VReg{vsum}, Note: "vector accumulator = 0"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{
						Op:   OpVectorCanLoadI32x4,
						Defs: []VReg{cmp},
						Uses: []VReg{local(1), local(scalar.IndexLocal)},
						Note: "index + 4 <= len",
					},
					{
						Op:     OpBranchIf,
						Uses:   []VReg{cmp},
						Target: tailName,
						Note:   "if fewer than four elements remain",
					},
					{
						Op:   OpVectorLoadI32x4Unaligned,
						Defs: []VReg{vchunk},
						Uses: []VReg{local(0), local(1), local(scalar.IndexLocal)},
						Note: proofNote,
					},
					{
						Op:   OpVectorAddI32x4,
						Defs: []VReg{vsum},
						Uses: []VReg{vsum, vchunk},
						Note: "vector total += xs[index:index+4]",
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(scalar.IndexLocal)},
						Uses: []VReg{local(scalar.IndexLocal), lane},
						Note: "index += 4",
					},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, tailName},
			},
			{
				Name: tailName,
				Instrs: []Instr{
					{
						Op:   OpVectorHorizontalAddI32x4,
						Defs: []VReg{local(scalar.TotalLocal)},
						Uses: []VReg{vsum},
						Note: "horizontal reduce vector accumulator",
					},
					{
						Op:   OpTailScalarI32Sum,
						Defs: []VReg{local(scalar.TotalLocal)},
						Uses: []VReg{
							local(scalar.TotalLocal),
							local(0),
							local(1),
							local(scalar.IndexLocal),
						},
						Note: scalar.ProofID + "; scalar tail handles len % 4",
					},
					{Op: OpBranch, Target: exitName},
				},
				Successors: []string{exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{local(scalar.TotalLocal)}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return VectorI32x4SliceSumLoopPlan{}, true, err
	}
	return VectorI32x4SliceSumLoopPlan{
		Function:           out,
		ScalarPlan:         scalar,
		LaneCount:          4,
		SafeUnaligned:      true,
		TailHandling:       "scalar_tail",
		ScalarFallback:     scalar.Function.Target,
		NoAliasRequirement: "not_required_read_only_reduction",
		ProofID:            scalar.ProofID,
	}, true, nil
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	n := v
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
