package machine

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

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
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 0) || in[7].Kind != ir.IRCmpLtI32 || in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], indexLocal) || in[11].Kind != ir.IRConstI32 || in[12].Kind != ir.IRMulI32 {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	if in[13].Kind != ir.IRConstI32 || in[14].Kind != ir.IRAddI32 || in[15].Kind != ir.IRAddI32 || !isStore(in[16], totalLocal) {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	scale := in[11].Imm
	bias := in[13].Imm
	if !validScalarAffineConstant(scale) || !validScalarAffineConstant(bias) {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	if !isLoad(in[17], indexLocal) || in[18].Kind != ir.IRConstI32 || in[18].Imm != 1 || in[19].Kind != ir.IRAddI32 || !isStore(in[20], indexLocal) {
		return ScalarIntAffineLoopPlan{}, false, nil
	}
	if in[21].Kind != ir.IRJmp || in[21].Label != startLabel || in[22].Kind != ir.IRLabel || in[22].Label != endLabel || !isLoad(in[23], totalLocal) || in[24].Kind != ir.IRReturn {
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
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(indexLocal), local(0)}, Note: "index < n"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{Op: OpMul, Defs: []VReg{scaled}, Uses: []VReg{local(indexLocal), scaleReg}, Note: "index * scale"},
					{Op: OpAdd, Defs: []VReg{affine}, Uses: []VReg{scaled, biasReg}, Note: "index * scale + bias"},
					{Op: OpAdd, Defs: []VReg{local(totalLocal)}, Uses: []VReg{local(totalLocal), affine}, Note: "total += index * scale + bias"},
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
