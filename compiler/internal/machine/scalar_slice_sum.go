package machine

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/ir"
)

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
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 1) || in[7].Kind != ir.IRCmpLtI32 || in[8].Kind != ir.IRJmpIfZero {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], 0) || !isLoad(in[11], 1) || !isLoad(in[12], indexLocal) {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	if in[13].Kind != ir.IRIndexLoadI32Unchecked || !strings.HasPrefix(in[13].ProofID, "proof:while:") {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	if in[14].Kind != ir.IRAddI32 || !isStore(in[15], totalLocal) {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	if !isLoad(in[16], indexLocal) || in[17].Kind != ir.IRConstI32 || in[18].Kind != ir.IRAddI32 || !isStore(in[19], indexLocal) {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	step := in[17].Imm
	if !validScalarLoopStep(step) {
		return ScalarI32SliceSumLoopPlan{}, false, nil
	}
	if in[20].Kind != ir.IRJmp || in[20].Label != startLabel || in[21].Kind != ir.IRLabel || in[21].Label != endLabel || !isLoad(in[22], totalLocal) || in[23].Kind != ir.IRReturn {
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
		entryInstrs = append(entryInstrs, Instr{Op: OpMov, Defs: []VReg{stepReg}, Imm: int64(step), Note: "loop step"})
	}
	entryInstrs = append(entryInstrs, Instr{Op: OpBranch, Target: loopName})
	advanceInstr := Instr{Op: OpInc, Defs: []VReg{local(indexLocal)}, Uses: []VReg{local(indexLocal)}, Note: "index++"}
	if step != 1 {
		advanceInstr = Instr{Op: OpAdd, Defs: []VReg{local(indexLocal)}, Uses: []VReg{local(indexLocal), stepReg}, Note: "index += step"}
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
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(indexLocal), local(1)}, Note: "index < len"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{Op: OpIndexLoad, Defs: []VReg{elem}, Uses: []VReg{local(0), local(1), local(indexLocal)}, Note: in[13].ProofID},
					{Op: OpAdd, Defs: []VReg{local(totalLocal)}, Uses: []VReg{local(totalLocal), elem}, Note: "total += xs[index]"},
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
