package machine

import (
	"strings"

	"tetra_language/compiler/internal/ir"
)

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
	if !isLoad(in[3], indexLocal) || !isLoad(in[4], 1) || in[5].Kind != ir.IRCmpLtI32 || in[6].Kind != ir.IRJmpIfZero {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	endLabel := in[6].Label
	if endLabel < 0 {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	if !isLoad(in[7], 0) || !isLoad(in[8], 1) || !isLoad(in[9], indexLocal) || in[10].Kind != ir.IRConstI32 || in[10].Imm != 0 {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	if in[11].Kind != ir.IRIndexStoreU8 || !strings.HasPrefix(in[11].ProofID, "proof:memset-loop:") {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	if !isLoad(in[12], indexLocal) || in[13].Kind != ir.IRConstI32 || in[13].Imm != 1 || in[14].Kind != ir.IRAddI32 || !isStore(in[15], indexLocal) {
		return ScalarU8MemsetZeroPlan{}, false, nil
	}
	if in[16].Kind != ir.IRJmp || in[16].Label != startLabel || in[17].Kind != ir.IRLabel || in[17].Label != endLabel || in[18].Kind != ir.IRConstI32 || in[18].Imm != 0 || in[19].Kind != ir.IRReturn {
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
					{Op: OpMov, Defs: []VReg{"zero"}, Imm: 0, Note: "zero fill and return code = 0"},
					{Op: OpMov, Defs: []VReg{local(indexLocal)}, Imm: 0, Note: "index = 0"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(indexLocal), local(1)}, Note: "index < len"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{Op: OpIndexStore, Uses: []VReg{local(0), local(1), local(indexLocal), "zero"}, Note: in[11].ProofID + "; single mutable slice zero-fill helper"},
					{Op: OpInc, Defs: []VReg{local(indexLocal)}, Uses: []VReg{local(indexLocal)}, Note: "index++"},
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
	proofNote := scalar.ProofID + "; safe unaligned u8x16 zero-fill store; single mutable slice zero-fill helper"

	out := Function{
		Name:   fn.Name,
		Target: "vector-u8x16-memset-zero-plan",
		Params: []VReg{local(0), local(1)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{"zero"}, Imm: 0, Note: "zero fill and return code = 0"},
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
					{Op: OpVectorCanMemsetU8x16, Defs: []VReg{cmp}, Uses: []VReg{local(1), local(scalar.IndexLocal)}, Note: "index + 16 <= len"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: tailName, Note: "if fewer than sixteen bytes remain"},
					{Op: OpVectorStoreU8x16Unaligned, Uses: []VReg{local(0), local(1), local(scalar.IndexLocal), vzero}, Note: proofNote},
					{Op: OpAdd, Defs: []VReg{local(scalar.IndexLocal)}, Uses: []VReg{local(scalar.IndexLocal), lane}, Note: "index += 16"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, tailName},
			},
			{
				Name: tailName,
				Instrs: []Instr{
					{Op: OpTailScalarU8Memset, Uses: []VReg{local(0), local(1), local(scalar.IndexLocal), "zero"}, Note: scalar.ProofID + "; scalar tail handles len % 16"},
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
