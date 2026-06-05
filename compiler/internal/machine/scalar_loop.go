package machine

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

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
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 0) || in[7].Kind != ir.IRCmpLtI32 || in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], indexLocal) || in[11].Kind != ir.IRAddI32 || !isStore(in[12], totalLocal) {
		return ScalarIntLoopPlan{}, false, nil
	}
	if !isLoad(in[13], indexLocal) || in[14].Kind != ir.IRConstI32 || in[15].Kind != ir.IRAddI32 || !isStore(in[16], indexLocal) {
		return ScalarIntLoopPlan{}, false, nil
	}
	step := in[14].Imm
	if !validScalarLoopStep(step) {
		return ScalarIntLoopPlan{}, false, nil
	}
	if in[17].Kind != ir.IRJmp || in[17].Label != startLabel || in[18].Kind != ir.IRLabel || in[18].Label != endLabel || !isLoad(in[19], totalLocal) || in[20].Kind != ir.IRReturn {
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
		entryInstrs = append(entryInstrs, Instr{Op: OpMov, Defs: []VReg{stepReg}, Imm: int64(step), Note: "loop step"})
	}
	entryInstrs = append(entryInstrs, Instr{Op: OpBranch, Target: loopName})
	advanceInstr := Instr{Op: OpInc, Defs: []VReg{local(indexLocal)}, Uses: []VReg{local(indexLocal)}, Note: "index++"}
	if step != 1 {
		advanceInstr = Instr{Op: OpAdd, Defs: []VReg{local(indexLocal)}, Uses: []VReg{local(indexLocal), stepReg}, Note: "index += step"}
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
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(indexLocal), local(0)}, Note: "index < n"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{Op: OpAdd, Defs: []VReg{local(totalLocal)}, Uses: []VReg{local(totalLocal), local(indexLocal)}, Note: "total += index"},
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
		return fmt.Errorf("machine scalar loop lowering: %s %s local %d out of bounds", fn.Name, name, local)
	}
	return nil
}
