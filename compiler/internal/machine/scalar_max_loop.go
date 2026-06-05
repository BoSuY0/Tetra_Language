package machine

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

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
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 0) || in[7].Kind != ir.IRCmpLtI32 || in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	if !isLoad(in[9], indexLocal) || !isLoad(in[10], maxLocal) || in[11].Kind != ir.IRCmpGtI32 || in[12].Kind != ir.IRJmpIfZero {
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
	if !isLoad(in[16], indexLocal) || in[17].Kind != ir.IRConstI32 || in[17].Imm != 1 || in[18].Kind != ir.IRAddI32 || !isStore(in[19], indexLocal) {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	if in[20].Kind != ir.IRJmp || in[20].Label != startLabel || in[21].Kind != ir.IRLabel || in[21].Label != endLabel || !isLoad(in[22], maxLocal) || in[23].Kind != ir.IRReturn {
		return ScalarIntMaxLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, maxLocal, "max"); err != nil {
		return ScalarIntMaxLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntMaxLoopPlan{}, true, err
	}
	if maxLocal == indexLocal || maxLocal == 0 || indexLocal == 0 || startLabel == keepLabel || startLabel == endLabel || keepLabel == endLabel {
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
					{Op: OpCmp, Defs: []VReg{loopCmp}, Uses: []VReg{local(indexLocal), local(0)}, Note: "index < n"},
					{Op: OpBranchIf, Uses: []VReg{loopCmp}, Target: exitName, Note: "if_zero"},
					{Op: OpCmp, Defs: []VReg{maxCmp}, Uses: []VReg{local(indexLocal), local(maxLocal)}, Note: "index > max"},
					{Op: OpBranchIf, Uses: []VReg{maxCmp}, Target: keepName, Note: "if_zero"},
					{Op: OpBranch, Target: updateName},
				},
				Successors: []string{exitName, keepName, updateName},
			},
			{
				Name: updateName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(maxLocal)}, Uses: []VReg{local(indexLocal)}, Note: "max = index"},
					{Op: OpBranch, Target: keepName},
				},
				Successors: []string{keepName},
			},
			{
				Name: keepName,
				Instrs: []Instr{
					{Op: OpInc, Defs: []VReg{local(indexLocal)}, Uses: []VReg{local(indexLocal)}, Note: "index++"},
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
