package bounds

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

type (
	Function = machine.Function
	VReg     = machine.VReg
	Block    = machine.Block
	Instr    = machine.Instr
)

const (
	OpAdd        = machine.OpAdd
	OpBranch     = machine.OpBranch
	OpBranchIf   = machine.OpBranchIf
	OpCmp        = machine.OpCmp
	OpInc        = machine.OpInc
	OpIndexLoad  = machine.OpIndexLoad
	OpIndexStore = machine.OpIndexStore
	OpMod        = machine.OpMod
	OpMov        = machine.OpMov
	OpMul        = machine.OpMul
	OpReturn     = machine.OpReturn
)

const (
	boundsCheckLoopsFunctionName    = "p25.bounds_check_loops.main"
	boundsCheckLoopsSliceLength     = int32(4096)
	boundsCheckLoopsBackingSlots    = 2048
	boundsCheckLoopsSliceBytes      = int32(16384)
	boundsCheckLoopsFillModulus     = int32(97)
	boundsCheckLoopsHotLoopBound    = int32(200000)
	boundsCheckLoopsIndexMultiplier = int32(17)
	boundsCheckLoopsStep            = int32(1)
)

type BoundsCheckLoopsPlan struct {
	Function        Function
	NLocal          int
	SlicePtrLocal   int
	SliceLenLocal   int
	IndexLocal      int
	TotalLocal      int
	IdxLocal        int
	BackingLocal    int
	BackingSlots    int
	FillStartLabel  int
	FillEndLabel    int
	HotStartLabel   int
	HotEndLabel     int
	FailureLabel    int
	SliceLength     int32
	SliceBytes      int32
	FillModulus     int32
	HotLoopBound    int32
	IndexMultiplier int32
	Step            int32
	SuccessReturn   int32
	FailureReturn   int32
	StoreProofID    string
	LoadProofID     string
}

func BoundsCheckLoopsFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := BoundsCheckLoopsPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func BoundsCheckLoopsPlanFromStackIR(fn ir.IRFunc) (BoundsCheckLoopsPlan, bool, error) {
	if fn.Name != boundsCheckLoopsFunctionName || fn.ParamSlots != 0 || fn.ReturnSlots != 1 ||
		fn.LocalSlots < 6 {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	if len(fn.Instrs) != 63 {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	in := fn.Instrs
	if in[0].Kind != ir.IRConstI32 || in[0].Imm != boundsCheckLoopsSliceLength ||
		in[1].Kind != ir.IRStoreLocal ||
		!isLoad(in[2], in[1].Local) ||
		in[3].Kind != ir.IRStackSliceI32 ||
		in[3].Local < 0 ||
		in[3].ArgSlots != boundsCheckLoopsBackingSlots ||
		in[3].Imm != boundsCheckLoopsSliceLength ||
		in[3].Name != "xs" {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	nLocal := in[1].Local
	backingLocal := in[3].Local
	backingSlots := in[3].ArgSlots
	if in[4].Kind != ir.IRStoreLocal || in[5].Kind != ir.IRStoreLocal {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	sliceLenLocal := in[4].Local
	slicePtrLocal := in[5].Local
	if !isConstStore(in[6], in[7], 0) {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	indexLocal := in[7].Local
	if in[8].Kind != ir.IRLabel || in[8].Label < 0 {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	fillStartLabel := in[8].Label
	if !isLoad(in[9], indexLocal) || !isLoad(in[10], nLocal) ||
		in[11].Kind != ir.IRCmpLtI32 ||
		in[12].Kind != ir.IRJmpIfZero || in[12].Label < 0 || in[12].Label == fillStartLabel {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	fillEndLabel := in[12].Label
	if !isLoad(in[13], slicePtrLocal) || !isLoad(in[14], sliceLenLocal) ||
		!isLoad(in[15], indexLocal) || !isLoad(in[16], indexLocal) ||
		in[17].Kind != ir.IRConstI32 || in[17].Imm != boundsCheckLoopsFillModulus ||
		in[18].Kind != ir.IRModI32 ||
		in[19].Kind != ir.IRIndexStoreI32 ||
		!strings.HasPrefix(in[19].ProofID, "proof:while:") {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	if !isLoad(in[20], indexLocal) ||
		in[21].Kind != ir.IRConstI32 || in[21].Imm != boundsCheckLoopsStep ||
		in[22].Kind != ir.IRAddI32 ||
		!isStore(in[23], indexLocal) ||
		in[24].Kind != ir.IRJmp || in[24].Label != fillStartLabel ||
		in[25].Kind != ir.IRLabel || in[25].Label != fillEndLabel {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	if !isConstStore(in[26], in[27], 0) || !isConstStore(in[28], in[29], 0) {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	totalLocal := in[27].Local
	hotIndexLocal := in[29].Local
	if hotIndexLocal != indexLocal {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	if in[30].Kind != ir.IRLabel || in[30].Label < 0 || in[30].Label == fillStartLabel ||
		in[30].Label == fillEndLabel {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	hotStartLabel := in[30].Label
	if !isLoad(in[31], indexLocal) ||
		in[32].Kind != ir.IRConstI32 || in[32].Imm != boundsCheckLoopsHotLoopBound ||
		in[33].Kind != ir.IRCmpLtI32 ||
		in[34].Kind != ir.IRJmpIfZero || in[34].Label < 0 ||
		in[34].Label == fillStartLabel || in[34].Label == fillEndLabel || in[34].Label == hotStartLabel {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	hotEndLabel := in[34].Label
	if !isLoad(in[35], indexLocal) ||
		in[36].Kind != ir.IRConstI32 || in[36].Imm != boundsCheckLoopsIndexMultiplier ||
		in[37].Kind != ir.IRMulI32 ||
		!isLoad(in[38], nLocal) ||
		in[39].Kind != ir.IRModI32 ||
		in[40].Kind != ir.IRStoreLocal {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	idxLocal := in[40].Local
	if !isLoad(in[41], totalLocal) ||
		!isLoad(
			in[42],
			slicePtrLocal,
		) || !isLoad(in[43], sliceLenLocal) || !isLoad(in[44], idxLocal) ||
		in[45].Kind != ir.IRIndexLoadI32Unchecked ||
		!strings.HasPrefix(in[45].ProofID, "proof:modulo:") ||
		in[46].Kind != ir.IRAddI32 ||
		!isStore(in[47], totalLocal) {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	if !isLoad(in[48], indexLocal) ||
		in[49].Kind != ir.IRConstI32 || in[49].Imm != boundsCheckLoopsStep ||
		in[50].Kind != ir.IRAddI32 ||
		!isStore(in[51], indexLocal) ||
		in[52].Kind != ir.IRJmp || in[52].Label != hotStartLabel ||
		in[53].Kind != ir.IRLabel || in[53].Label != hotEndLabel {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	if !isLoad(in[54], totalLocal) ||
		in[55].Kind != ir.IRConstI32 || in[55].Imm != 0 ||
		in[56].Kind != ir.IRCmpGeI32 ||
		in[57].Kind != ir.IRJmpIfZero || in[57].Label < 0 ||
		in[58].Kind != ir.IRConstI32 || in[58].Imm != 0 ||
		in[59].Kind != ir.IRReturn ||
		in[60].Kind != ir.IRLabel || in[60].Label != in[57].Label ||
		in[61].Kind != ir.IRConstI32 || in[61].Imm != 1 ||
		in[62].Kind != ir.IRReturn {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	failureLabel := in[57].Label
	for _, local := range []struct {
		slot int
		name string
	}{
		{nLocal, "n"},
		{slicePtrLocal, "slice ptr"},
		{sliceLenLocal, "slice len"},
		{indexLocal, "index"},
		{totalLocal, "total"},
		{idxLocal, "idx"},
	} {
		if err := validateScalarLoopLocal(fn, local.slot, local.name); err != nil {
			return BoundsCheckLoopsPlan{}, true, err
		}
	}
	if backingLocal+backingSlots > fn.LocalSlots {
		return BoundsCheckLoopsPlan{}, true, fmt.Errorf(
			"machine bounds-check-loops lowering: %s backing slots [%d,%d) out of bounds (locals=%d)",
			fn.Name,
			backingLocal,
			backingLocal+backingSlots,
			fn.LocalSlots,
		)
	}
	if !distinctBoundsCheckLoopLocals(
		nLocal,
		slicePtrLocal,
		sliceLenLocal,
		indexLocal,
		totalLocal,
		idxLocal,
	) {
		return BoundsCheckLoopsPlan{}, false, nil
	}
	for _, local := range []int{
		nLocal,
		slicePtrLocal,
		sliceLenLocal,
		indexLocal,
		totalLocal,
		idxLocal,
	} {
		if local >= backingLocal && local < backingLocal+backingSlots {
			return BoundsCheckLoopsPlan{}, false, nil
		}
	}

	plan := BoundsCheckLoopsPlan{
		NLocal:          nLocal,
		SlicePtrLocal:   slicePtrLocal,
		SliceLenLocal:   sliceLenLocal,
		IndexLocal:      indexLocal,
		TotalLocal:      totalLocal,
		IdxLocal:        idxLocal,
		BackingLocal:    backingLocal,
		BackingSlots:    backingSlots,
		FillStartLabel:  fillStartLabel,
		FillEndLabel:    fillEndLabel,
		HotStartLabel:   hotStartLabel,
		HotEndLabel:     hotEndLabel,
		FailureLabel:    failureLabel,
		SliceLength:     boundsCheckLoopsSliceLength,
		SliceBytes:      boundsCheckLoopsSliceBytes,
		FillModulus:     boundsCheckLoopsFillModulus,
		HotLoopBound:    boundsCheckLoopsHotLoopBound,
		IndexMultiplier: boundsCheckLoopsIndexMultiplier,
		Step:            boundsCheckLoopsStep,
		SuccessReturn:   in[58].Imm,
		FailureReturn:   in[61].Imm,
		StoreProofID:    in[19].ProofID,
		LoadProofID:     in[45].ProofID,
	}
	out, err := buildBoundsCheckLoopsMachineFunction(fn.Name, plan)
	if err != nil {
		return BoundsCheckLoopsPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func distinctBoundsCheckLoopLocals(locals ...int) bool {
	seen := map[int]bool{}
	for _, local := range locals {
		if seen[local] {
			return false
		}
		seen[local] = true
	}
	return true
}

func buildBoundsCheckLoopsMachineFunction(
	name string,
	plan BoundsCheckLoopsPlan,
) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	fillName := scalarLoopLabelName(plan.FillStartLabel)
	afterFillName := scalarLoopLabelName(plan.FillEndLabel)
	hotName := scalarLoopLabelName(plan.HotStartLabel)
	afterHotName := scalarLoopLabelName(plan.HotEndLabel)
	failureName := scalarLoopLabelName(plan.FailureLabel)
	successName := "return_success"
	fillCmp := VReg("t0")
	fillModulus := VReg("t1")
	fillValue := VReg("t2")
	hotBound := VReg("t3")
	hotCmp := VReg("t4")
	multiplier := VReg("t5")
	idxProduct := VReg("t6")
	loaded := VReg("t7")
	finalZero := VReg("t8")
	finalCmp := VReg("t9")
	ret := VReg("t10")

	out := Function{
		Name:   name,
		Target: "bounds-check-loops",
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{local(plan.NLocal)},
						Imm:  int64(plan.SliceLength),
						Note: "n = 4096",
					},
					{Op: OpMov, Defs: []VReg{local(plan.SlicePtrLocal)}, Note: "xs stack ptr"},
					{
						Op:   OpMov,
						Defs: []VReg{local(plan.SliceLenLocal)},
						Imm:  int64(plan.SliceLength),
						Note: "xs len = 4096",
					},
					{Op: OpMov, Defs: []VReg{local(plan.IndexLocal)}, Imm: 0, Note: "i = 0"},
					{Op: OpBranch, Target: fillName},
				},
				Successors: []string{fillName},
			},
			{
				Name: fillName,
				Instrs: []Instr{
					{
						Op:   OpCmp,
						Defs: []VReg{fillCmp},
						Uses: []VReg{local(plan.IndexLocal), local(plan.NLocal)},
						Note: "i < n",
					},
					{Op: OpBranchIf, Uses: []VReg{fillCmp}, Target: afterFillName, Note: "if_zero"},
					{
						Op:   OpMov,
						Defs: []VReg{fillModulus},
						Imm:  int64(plan.FillModulus),
						Note: "literal fill modulus",
					},
					{
						Op:   OpMod,
						Defs: []VReg{fillValue},
						Uses: []VReg{local(plan.IndexLocal), fillModulus},
						Note: "fill xs[i] = i % 97",
					},
					{
						Op: OpIndexStore,
						Uses: []VReg{
							local(plan.SlicePtrLocal),
							local(plan.SliceLenLocal),
							local(plan.IndexLocal),
							fillValue,
						},
						Note: "fill xs[i] = i % 97 " + plan.StoreProofID,
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.IndexLocal)},
						Uses: []VReg{local(plan.IndexLocal)},
						Note: "i++",
					},
					{Op: OpBranch, Target: fillName},
				},
				Successors: []string{afterFillName, fillName},
			},
			{
				Name: afterFillName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(plan.TotalLocal)}, Imm: 0, Note: "total = 0"},
					{Op: OpMov, Defs: []VReg{local(plan.IndexLocal)}, Imm: 0, Note: "i = 0"},
					{Op: OpBranch, Target: hotName},
				},
				Successors: []string{hotName},
			},
			{
				Name: hotName,
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{hotBound},
						Imm:  int64(plan.HotLoopBound),
						Note: "hot loop bound = 200000",
					},
					{
						Op:   OpCmp,
						Defs: []VReg{hotCmp},
						Uses: []VReg{local(plan.IndexLocal), hotBound},
						Note: "i < 200000",
					},
					{Op: OpBranchIf, Uses: []VReg{hotCmp}, Target: afterHotName, Note: "if_zero"},
					{
						Op:   OpMov,
						Defs: []VReg{multiplier},
						Imm:  int64(plan.IndexMultiplier),
						Note: "literal index multiplier",
					},
					{
						Op:   OpMul,
						Defs: []VReg{idxProduct},
						Uses: []VReg{local(plan.IndexLocal), multiplier},
						Note: "i * 17",
					},
					{
						Op:   OpMod,
						Defs: []VReg{local(plan.IdxLocal)},
						Uses: []VReg{idxProduct, local(plan.NLocal)},
						Note: "idx = (i * 17) % 4096",
					},
					{
						Op:   OpIndexLoad,
						Defs: []VReg{loaded},
						Uses: []VReg{
							local(plan.SlicePtrLocal),
							local(plan.SliceLenLocal),
							local(plan.IdxLocal),
						},
						Note: plan.LoadProofID,
					},
					{
						Op:   OpAdd,
						Defs: []VReg{local(plan.TotalLocal)},
						Uses: []VReg{local(plan.TotalLocal), loaded},
						Note: "total += xs[idx]",
					},
					{
						Op:   OpInc,
						Defs: []VReg{local(plan.IndexLocal)},
						Uses: []VReg{local(plan.IndexLocal)},
						Note: "i++",
					},
					{Op: OpBranch, Target: hotName},
				},
				Successors: []string{afterHotName, hotName},
			},
			{
				Name: afterHotName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{finalZero}, Imm: 0, Note: "zero for final guard"},
					{
						Op:   OpCmp,
						Defs: []VReg{finalCmp},
						Uses: []VReg{local(plan.TotalLocal), finalZero},
						Note: "total >= 0",
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
						Defs: []VReg{ret},
						Imm:  int64(plan.SuccessReturn),
						Note: "return 0",
					},
					{Op: OpReturn, Uses: []VReg{ret}},
				},
			},
			{
				Name: failureName,
				Instrs: []Instr{
					{
						Op:   OpMov,
						Defs: []VReg{ret},
						Imm:  int64(plan.FailureReturn),
						Note: "return 1",
					},
					{Op: OpReturn, Uses: []VReg{ret}},
				},
			},
		},
	}
	if err := machine.VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
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
			"machine bounds-check-loops lowering: %s %s local %d out of bounds",
			fn.Name,
			name,
			local,
		)
	}
	return nil
}
