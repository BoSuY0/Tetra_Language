package machine

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

type ScalarIntCallLoopPlan struct {
	Function   Function
	ParamLocal int
	IndexLocal int
	TotalLocal int
	StartLabel int
	EndLabel   int
	CallName   string
}

func ScalarIntCallLoopFunctionFromStackIR(fn ir.IRFunc) (Function, bool, error) {
	return ScalarIntCallLoopFunctionFromStackIRWithCallABI(fn, SysVCallABIInfo())
}

func ScalarIntCallLoopFunctionFromStackIRWithCallABI(fn ir.IRFunc, callABI CallABIInfo) (Function, bool, error) {
	plan, ok, err := ScalarIntCallLoopPlanFromStackIRWithCallABI(fn, callABI)
	return plan.Function, ok, err
}

func ScalarIntCallLoopPlanFromStackIR(fn ir.IRFunc) (ScalarIntCallLoopPlan, bool, error) {
	return ScalarIntCallLoopPlanFromStackIRWithCallABI(fn, SysVCallABIInfo())
}

func ScalarIntCallLoopPlanFromStackIRWithCallABI(fn ir.IRFunc, callABI CallABIInfo) (ScalarIntCallLoopPlan, bool, error) {
	if fn.ReturnSlots != 1 || fn.ParamSlots != 1 || fn.LocalSlots < 3 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if err := validateCallABIInfo(callABI); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	if len(fn.Instrs) != 22 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	totalLocal := in[3].Local
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel < 0 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) || !isLoad(in[6], 0) || in[7].Kind != ir.IRCmpLtI32 || in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], indexLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[11].Kind != ir.IRCall || in[11].Name == "" || in[11].ArgSlots != 1 || in[11].RetSlots != 1 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if callABI.MaxArgSlots < 1 || callABI.MaxRetSlots < 1 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[12].Kind != ir.IRAddI32 || !isStore(in[13], totalLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[14], indexLocal) || in[15].Kind != ir.IRConstI32 || in[15].Imm != 1 || in[16].Kind != ir.IRAddI32 || !isStore(in[17], indexLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[18].Kind != ir.IRJmp || in[18].Label != startLabel || in[19].Kind != ir.IRLabel || in[19].Label != endLabel || !isLoad(in[20], totalLocal) || in[21].Kind != ir.IRReturn {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	if indexLocal == totalLocal || indexLocal == 0 || totalLocal == 0 {
		return ScalarIntCallLoopPlan{}, false, nil
	}

	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	cmp := VReg("t0")
	callRet := VReg("t1")
	loopName := scalarLoopLabelName(startLabel)
	exitName := scalarLoopLabelName(endLabel)
	out := Function{
		Name:   fn.Name,
		Target: "scalar-int-call-loop",
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
					{
						Op:       OpCall,
						Defs:     []VReg{callRet},
						Uses:     []VReg{local(indexLocal)},
						Call:     in[11].Name,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "caller-saved state is frame-spilled around call",
					},
					{Op: OpAdd, Defs: []VReg{local(totalLocal)}, Uses: []VReg{local(totalLocal), callRet}, Note: "total += call(index)"},
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
		return ScalarIntCallLoopPlan{}, true, err
	}
	return ScalarIntCallLoopPlan{
		Function:   out,
		ParamLocal: 0,
		IndexLocal: indexLocal,
		TotalLocal: totalLocal,
		StartLabel: startLabel,
		EndLabel:   endLabel,
		CallName:   in[11].Name,
	}, true, nil
}
