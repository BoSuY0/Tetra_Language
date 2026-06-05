package machine

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

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
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 0) || in[7].Kind != ir.IRCmpLtI32 || in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntProductLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntProductLoopPlan{}, false, nil
	}
	if !isLoad(in[9], productLocal) || !isLoad(in[10], indexLocal) || in[11].Kind != ir.IRConstI32 || in[11].Imm != 1 || in[12].Kind != ir.IRAddI32 || in[13].Kind != ir.IRMulI32 || !isStore(in[14], productLocal) {
		return ScalarIntProductLoopPlan{}, false, nil
	}
	if !isLoad(in[15], indexLocal) || in[16].Kind != ir.IRConstI32 || in[16].Imm != 1 || in[17].Kind != ir.IRAddI32 || !isStore(in[18], indexLocal) {
		return ScalarIntProductLoopPlan{}, false, nil
	}
	if in[19].Kind != ir.IRJmp || in[19].Label != startLabel || in[20].Kind != ir.IRLabel || in[20].Label != endLabel || !isLoad(in[21], productLocal) || in[22].Kind != ir.IRReturn {
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
					{Op: OpMov, Defs: []VReg{local(productLocal)}, Imm: 1, Note: "loop product = 1"},
					{Op: OpMov, Defs: []VReg{one}, Imm: 1, Note: "one"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(indexLocal), local(0)}, Note: "index < n"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{Op: OpAdd, Defs: []VReg{factor}, Uses: []VReg{local(indexLocal), one}, Note: "index + 1"},
					{Op: OpMul, Defs: []VReg{local(productLocal)}, Uses: []VReg{local(productLocal), factor}, Note: "product *= index + 1"},
					{Op: OpInc, Defs: []VReg{local(indexLocal)}, Uses: []VReg{local(indexLocal)}, Note: "index++"},
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
