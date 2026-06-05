package machine

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

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
	if !isLoad(in[3], 0) || in[4].Kind != ir.IRConstI32 || in[4].Imm != 0 || in[5].Kind != ir.IRCmpGtI32 || in[6].Kind != ir.IRJmpIfZero {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}
	endLabel := in[6].Label
	if endLabel < 0 {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}
	if !isLoad(in[7], totalLocal) || !isLoad(in[8], 0) || in[9].Kind != ir.IRAddI32 || !isStore(in[10], totalLocal) {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}
	if !isLoad(in[11], 0) || in[12].Kind != ir.IRConstI32 || in[12].Imm != 1 || in[13].Kind != ir.IRSubI32 || !isStore(in[14], 0) {
		return ScalarIntCountdownLoopPlan{}, false, nil
	}
	if in[15].Kind != ir.IRJmp || in[15].Label != startLabel || in[16].Kind != ir.IRLabel || in[16].Label != endLabel || !isLoad(in[17], totalLocal) || in[18].Kind != ir.IRReturn {
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
					{Op: OpAdd, Defs: []VReg{local(totalLocal)}, Uses: []VReg{local(totalLocal), local(0)}, Note: "total += n"},
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
