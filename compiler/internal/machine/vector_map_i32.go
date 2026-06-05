package machine

import (
	"strings"

	"tetra_language/compiler/internal/ir"
)

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
	if !isLoad(in[3], indexLocal) || !isLoad(in[4], 1) || in[5].Kind != ir.IRCmpLtI32 || in[6].Kind != ir.IRJmpIfZero {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	endLabel := in[6].Label
	if endLabel < 0 {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	if !isLoad(in[7], 0) || !isLoad(in[8], 1) || !isLoad(in[9], indexLocal) {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	if in[10].Kind != ir.IRIndexLoadI32Unchecked || !strings.HasPrefix(in[10].ProofID, "proof:map-loop:") {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	if in[11].Kind != ir.IRConstI32 || in[12].Kind != ir.IRAddI32 || in[13].Kind != ir.IRStoreLocal {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	addend := in[11].Imm
	if addend != 1 {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	tempLocal := in[13].Local
	if !isLoad(in[14], 0) || !isLoad(in[15], 1) || !isLoad(in[16], indexLocal) || !isLoad(in[17], tempLocal) || in[18].Kind != ir.IRIndexStoreI32 {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	if !isLoad(in[19], indexLocal) || in[20].Kind != ir.IRConstI32 || in[20].Imm != 1 || in[21].Kind != ir.IRAddI32 || !isStore(in[22], indexLocal) {
		return ScalarI32MapAddConstLoopPlan{}, false, nil
	}
	if in[23].Kind != ir.IRJmp || in[23].Label != startLabel || in[24].Kind != ir.IRLabel || in[24].Label != endLabel || in[25].Kind != ir.IRConstI32 || in[25].Imm != 0 || in[26].Kind != ir.IRReturn {
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
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(indexLocal), local(1)}, Note: "index < len"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{Op: OpIndexLoad, Defs: []VReg{elem}, Uses: []VReg{local(0), local(1), local(indexLocal)}, Note: in[10].ProofID},
					{Op: OpAdd, Defs: []VReg{local(tempLocal)}, Uses: []VReg{elem, add}, Note: "xs[index] + addend"},
					{Op: OpIndexStore, Uses: []VReg{local(0), local(1), local(indexLocal), local(tempLocal)}, Note: in[10].ProofID + "; store uses same range proof"},
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
	proofNote := scalar.ProofID + "; safe unaligned i32x4 map load/store; single mutable slice in-place map"

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
					{Op: OpVectorSplatI32x4, Defs: []VReg{vadd}, Imm: int64(scalar.Addend), Note: "broadcast map addend"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{Op: OpVectorCanMapI32x4, Defs: []VReg{cmp}, Uses: []VReg{local(1), local(scalar.IndexLocal)}, Note: "index + 4 <= len"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: tailName, Note: "if fewer than four elements remain"},
					{Op: OpVectorLoadI32x4Unaligned, Defs: []VReg{vchunk}, Uses: []VReg{local(0), local(1), local(scalar.IndexLocal)}, Note: proofNote},
					{Op: OpVectorAddI32x4, Defs: []VReg{vchunk}, Uses: []VReg{vchunk, vadd}, Note: "vector xs[index:index+4] += addend"},
					{Op: OpVectorStoreI32x4Unaligned, Uses: []VReg{local(0), local(1), local(scalar.IndexLocal), vchunk}, Note: proofNote},
					{Op: OpAdd, Defs: []VReg{local(scalar.IndexLocal)}, Uses: []VReg{local(scalar.IndexLocal), lane}, Note: "index += 4"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, tailName},
			},
			{
				Name: tailName,
				Instrs: []Instr{
					{Op: OpTailScalarI32Map, Uses: []VReg{local(0), local(1), local(scalar.IndexLocal), vadd}, Note: scalar.ProofID + "; scalar tail handles len % 4"},
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
