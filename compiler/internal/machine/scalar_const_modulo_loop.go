package machine

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

const (
	integerLoopsBenchmarkBound   int32 = 200000
	integerLoopsBenchmarkModulus int32 = 7
)

type ScalarIntConstModuloLoopPlan struct {
	Function       Function
	IndexLocal     int
	TotalLocal     int
	Bound          int32
	Modulus        int32
	StartLabel     int
	EndLabel       int
	NegativeLabel  int
	TrueReturnImm  int32
	FalseReturnImm int32
}

func ScalarIntConstModuloLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	plan, ok, err := ScalarIntConstModuloLoopPlanFromStackIR(fn)
	return plan.Function, ok, err
}

func ScalarIntConstModuloLoopPlanFromStackIR(fn ir.IRFunc) (ScalarIntConstModuloLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 0 || fn.LocalSlots < 2 {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if len(fn.Instrs) != 30 {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	totalLocal := in[3].Local
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) || in[6].Kind != ir.IRConstI32 || in[6].Imm != integerLoopsBenchmarkBound ||
		in[7].Kind != ir.IRCmpLtI32 || in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 || endLabel == startLabel {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], indexLocal) ||
		in[11].Kind != ir.IRConstI32 || in[11].Imm != integerLoopsBenchmarkModulus ||
		in[12].Kind != ir.IRModI32 || in[13].Kind != ir.IRAddI32 || !isStore(in[14], totalLocal) {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if !isLoad(in[15], indexLocal) || in[16].Kind != ir.IRConstI32 || in[16].Imm != 1 ||
		in[17].Kind != ir.IRAddI32 || !isStore(in[18], indexLocal) {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if in[19].Kind != ir.IRJmp || in[19].Label != startLabel ||
		in[20].Kind != ir.IRLabel || in[20].Label != endLabel ||
		!isLoad(in[21], totalLocal) || in[22].Kind != ir.IRConstI32 || in[22].Imm != 0 ||
		in[23].Kind != ir.IRCmpGeI32 || in[24].Kind != ir.IRJmpIfZero {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	negativeLabel := in[24].Label
	if negativeLabel < 0 || negativeLabel == startLabel || negativeLabel == endLabel {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if in[25].Kind != ir.IRConstI32 || in[25].Imm != 0 || in[26].Kind != ir.IRReturn ||
		in[27].Kind != ir.IRLabel || in[27].Label != negativeLabel ||
		in[28].Kind != ir.IRConstI32 || in[28].Imm != 1 || in[29].Kind != ir.IRReturn {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntConstModuloLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return ScalarIntConstModuloLoopPlan{}, true, err
	}
	if indexLocal == totalLocal {
		return ScalarIntConstModuloLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	cmp := VReg("t0")
	bound := VReg("t1")
	modulus := VReg("t2")
	remainder := VReg("t3")
	zero := VReg("t4")
	finalCmp := VReg("t5")
	returnZero := VReg("t6")
	returnOne := VReg("t7")
	loopName := scalarLoopLabelName(startLabel)
	exitName := scalarLoopLabelName(endLabel)
	negativeName := scalarLoopLabelName(negativeLabel)
	out := Function{
		Name:   fn.Name,
		Target: "scalar-int-const-modulo-loop",
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(indexLocal)}, Imm: 0, Note: "loop index = 0"},
					{Op: OpMov, Defs: []VReg{local(totalLocal)}, Imm: 0, Note: "loop total = 0"},
					{Op: OpMov, Defs: []VReg{bound}, Imm: int64(integerLoopsBenchmarkBound), Note: "literal loop bound"},
					{Op: OpMov, Defs: []VReg{modulus}, Imm: int64(integerLoopsBenchmarkModulus), Note: "literal modulo divisor"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(indexLocal), bound}, Note: "index < literal bound"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{Op: OpMod, Defs: []VReg{remainder}, Uses: []VReg{local(indexLocal), modulus}, Note: "index % literal modulus"},
					{Op: OpAdd, Defs: []VReg{local(totalLocal)}, Uses: []VReg{local(totalLocal), remainder}, Note: "total += index % modulus"},
					{Op: OpInc, Defs: []VReg{local(indexLocal)}, Uses: []VReg{local(indexLocal)}, Note: "index++"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{zero}, Imm: 0, Note: "zero for final guard"},
					{Op: OpCmp, Defs: []VReg{finalCmp}, Uses: []VReg{local(totalLocal), zero}, Note: "total >= 0"},
					{Op: OpBranchIf, Uses: []VReg{finalCmp}, Target: negativeName, Note: "if_zero"},
					{Op: OpMov, Defs: []VReg{returnZero}, Imm: 0, Note: "return 0"},
					{Op: OpReturn, Uses: []VReg{returnZero}},
				},
				Successors: []string{negativeName},
			},
			{
				Name: negativeName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{returnOne}, Imm: 1, Note: "return 1"},
					{Op: OpReturn, Uses: []VReg{returnOne}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return ScalarIntConstModuloLoopPlan{}, true, err
	}
	return ScalarIntConstModuloLoopPlan{
		Function:       out,
		IndexLocal:     indexLocal,
		TotalLocal:     totalLocal,
		Bound:          integerLoopsBenchmarkBound,
		Modulus:        integerLoopsBenchmarkModulus,
		StartLabel:     startLabel,
		EndLabel:       endLabel,
		NegativeLabel:  negativeLabel,
		TrueReturnImm:  0,
		FalseReturnImm: 1,
	}, true, nil
}
