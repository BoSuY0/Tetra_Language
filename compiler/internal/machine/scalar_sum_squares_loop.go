package machine

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

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

func ScalarIntSumSquaresLoopPlanFromStackIR(fn ir.IRFunc) (ScalarIntSumSquaresLoopPlan, bool, error) {
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
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 0) || in[7].Kind != ir.IRCmpLtI32 || in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], indexLocal) || !isLoad(in[11], indexLocal) || in[12].Kind != ir.IRMulI32 || in[13].Kind != ir.IRAddI32 || !isStore(in[14], totalLocal) {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}
	if !isLoad(in[15], indexLocal) || in[16].Kind != ir.IRConstI32 || in[16].Imm != 1 || in[17].Kind != ir.IRAddI32 || !isStore(in[18], indexLocal) {
		return ScalarIntSumSquaresLoopPlan{}, false, nil
	}
	if in[19].Kind != ir.IRJmp || in[19].Label != startLabel || in[20].Kind != ir.IRLabel || in[20].Label != endLabel || !isLoad(in[21], totalLocal) || in[22].Kind != ir.IRReturn {
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
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(indexLocal), local(0)}, Note: "index < n"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{Op: OpMul, Defs: []VReg{square}, Uses: []VReg{local(indexLocal), local(indexLocal)}, Note: "index * index"},
					{Op: OpAdd, Defs: []VReg{local(totalLocal)}, Uses: []VReg{local(totalLocal), square}, Note: "total += index * index"},
					{Op: OpInc, Defs: []VReg{local(indexLocal)}, Uses: []VReg{local(indexLocal)}, Note: "index++"},
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
