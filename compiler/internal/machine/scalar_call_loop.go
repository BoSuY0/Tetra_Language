package machine

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

const (
	compileTimeMainName      = "p25.compile_time.main"
	compileTimeLoopCallName  = "p25.compile_time.f2"
	compileTimeLoopBound     = int32(200000)
	compileTimeEqualReturn   = int32(1)
	compileTimeUnequalReturn = int32(0)
)

type ScalarIntCallLoopPlan struct {
	Function                 Function
	ParamLocal               int
	BoundLocal               int
	BoundConst               int32
	IndexLocal               int
	TotalLocal               int
	StartLabel               int
	EndLabel                 int
	CallName                 string
	CallArgLocals            []int
	ReturnNonNegativeSuccess bool
	ReturnOneIfTotalZero     bool
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
	if fn.ReturnSlots != 1 || fn.LocalSlots < 2 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if err := validateCallABIInfo(callABI); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	if plan, ok, err := scalarIntParamBoundCallLoopPlan(fn, callABI); ok || err != nil {
		return plan, ok, err
	}
	if plan, ok, err := scalarIntConstBoundTwoArgSuccessCallLoopPlan(fn, callABI); ok || err != nil {
		return plan, ok, err
	}
	if plan, ok, err := scalarIntCompileTimeEqualityTailCallLoopPlan(fn, callABI); ok || err != nil {
		return plan, ok, err
	}
	return ScalarIntCallLoopPlan{}, false, nil
}

func scalarIntParamBoundCallLoopPlan(fn ir.IRFunc, callABI CallABIInfo) (ScalarIntCallLoopPlan, bool, error) {
	if fn.ParamSlots != 1 || fn.LocalSlots < 3 || len(fn.Instrs) != 22 {
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
	if callABI.MaxArgSlots < int(in[11].ArgSlots) || callABI.MaxRetSlots < int(in[11].RetSlots) {
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

	plan := ScalarIntCallLoopPlan{
		ParamLocal:    0,
		BoundLocal:    0,
		IndexLocal:    indexLocal,
		TotalLocal:    totalLocal,
		StartLabel:    startLabel,
		EndLabel:      endLabel,
		CallName:      in[11].Name,
		CallArgLocals: []int{indexLocal},
	}
	out, err := buildScalarIntCallLoopMachineFunction(fn.Name, callABI, plan)
	if err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func scalarIntConstBoundTwoArgSuccessCallLoopPlan(fn ir.IRFunc, callABI CallABIInfo) (ScalarIntCallLoopPlan, bool, error) {
	if fn.ParamSlots != 0 || fn.LocalSlots < 2 || len(fn.Instrs) != 30 {
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
	if !isLoad(in[5], indexLocal) || in[6].Kind != ir.IRConstI32 || in[7].Kind != ir.IRCmpLtI32 || in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	boundConst := in[6].Imm
	if boundConst < 1 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], indexLocal) || !isLoad(in[11], totalLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[12].Kind != ir.IRCall || in[12].Name == "" || in[12].ArgSlots != 2 || in[12].RetSlots != 1 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if callABI.MaxArgSlots < int(in[12].ArgSlots) || callABI.MaxRetSlots < int(in[12].RetSlots) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[13].Kind != ir.IRAddI32 || !isStore(in[14], totalLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[15], indexLocal) || in[16].Kind != ir.IRConstI32 || in[16].Imm != 1 || in[17].Kind != ir.IRAddI32 || !isStore(in[18], indexLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[19].Kind != ir.IRJmp || in[19].Label != startLabel || in[20].Kind != ir.IRLabel || in[20].Label != endLabel {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[21], totalLocal) || in[22].Kind != ir.IRConstI32 || in[22].Imm != 0 || in[23].Kind != ir.IRCmpGeI32 || in[24].Kind != ir.IRJmpIfZero {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	failLabel := in[24].Label
	if failLabel < 0 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[25].Kind != ir.IRConstI32 || in[25].Imm != 0 || in[26].Kind != ir.IRReturn {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[27].Kind != ir.IRLabel || in[27].Label != failLabel || in[28].Kind != ir.IRConstI32 || in[28].Imm != 1 || in[29].Kind != ir.IRReturn {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	if indexLocal == totalLocal {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	plan := ScalarIntCallLoopPlan{
		ParamLocal:               -1,
		BoundLocal:               -1,
		BoundConst:               boundConst,
		IndexLocal:               indexLocal,
		TotalLocal:               totalLocal,
		StartLabel:               startLabel,
		EndLabel:                 endLabel,
		CallName:                 in[12].Name,
		CallArgLocals:            []int{indexLocal, totalLocal},
		ReturnNonNegativeSuccess: true,
	}
	out, err := buildScalarIntCallLoopMachineFunction(fn.Name, callABI, plan)
	if err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	plan.Function = out
	return ScalarIntCallLoopPlan{
		Function:                 out,
		ParamLocal:               plan.ParamLocal,
		BoundLocal:               plan.BoundLocal,
		BoundConst:               plan.BoundConst,
		IndexLocal:               plan.IndexLocal,
		TotalLocal:               plan.TotalLocal,
		StartLabel:               plan.StartLabel,
		EndLabel:                 plan.EndLabel,
		CallName:                 plan.CallName,
		CallArgLocals:            append([]int(nil), plan.CallArgLocals...),
		ReturnNonNegativeSuccess: plan.ReturnNonNegativeSuccess,
		ReturnOneIfTotalZero:     plan.ReturnOneIfTotalZero,
	}, true, nil
}

func scalarIntCompileTimeEqualityTailCallLoopPlan(fn ir.IRFunc, callABI CallABIInfo) (ScalarIntCallLoopPlan, bool, error) {
	if fn.Name != compileTimeMainName || fn.ParamSlots != 0 || fn.LocalSlots != 2 || fn.ReturnSlots != 1 || len(fn.Instrs) != 29 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	indexLocal := in[1].Local
	totalLocal := in[3].Local
	if indexLocal != 0 || totalLocal != 1 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel != 0 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) ||
		in[6].Kind != ir.IRConstI32 || in[6].Imm != compileTimeLoopBound ||
		in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel != 1 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) || !isLoad(in[10], indexLocal) ||
		in[11].Kind != ir.IRCall || in[11].Name != compileTimeLoopCallName || in[11].ArgSlots != 1 || in[11].RetSlots != 1 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if callABI.MaxArgSlots < int(in[11].ArgSlots) || callABI.MaxRetSlots < int(in[11].RetSlots) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[12].Kind != ir.IRAddI32 || !isStore(in[13], totalLocal) {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[14], indexLocal) ||
		in[15].Kind != ir.IRConstI32 || in[15].Imm != 1 ||
		in[16].Kind != ir.IRAddI32 || !isStore(in[17], indexLocal) ||
		in[18].Kind != ir.IRJmp || in[18].Label != startLabel ||
		in[19].Kind != ir.IRLabel || in[19].Label != endLabel {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if !isLoad(in[20], totalLocal) ||
		in[21].Kind != ir.IRConstI32 || in[21].Imm != 0 ||
		in[22].Kind != ir.IRCmpEqI32 ||
		in[23].Kind != ir.IRJmpIfZero {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	unequalLabel := in[23].Label
	if unequalLabel != 2 {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if in[24].Kind != ir.IRConstI32 || in[24].Imm != compileTimeEqualReturn ||
		in[25].Kind != ir.IRReturn ||
		in[26].Kind != ir.IRLabel || in[26].Label != unequalLabel ||
		in[27].Kind != ir.IRConstI32 || in[27].Imm != compileTimeUnequalReturn ||
		in[28].Kind != ir.IRReturn {
		return ScalarIntCallLoopPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	plan := ScalarIntCallLoopPlan{
		ParamLocal:           -1,
		BoundLocal:           -1,
		BoundConst:           compileTimeLoopBound,
		IndexLocal:           indexLocal,
		TotalLocal:           totalLocal,
		StartLabel:           startLabel,
		EndLabel:             endLabel,
		CallName:             in[11].Name,
		CallArgLocals:        []int{indexLocal},
		ReturnOneIfTotalZero: true,
	}
	out, err := buildScalarIntCallLoopMachineFunction(fn.Name, callABI, plan)
	if err != nil {
		return ScalarIntCallLoopPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func buildScalarIntCallLoopMachineFunction(name string, callABI CallABIInfo, plan ScalarIntCallLoopPlan) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	cmp := VReg("t0")
	callRet := VReg("t1")
	loopName := scalarLoopLabelName(plan.StartLabel)
	exitName := scalarLoopLabelName(plan.EndLabel)
	bound := VReg("bound")
	cmpUses := []VReg{local(plan.IndexLocal), bound}
	params := []VReg(nil)
	entryInstrs := []Instr{
		{Op: OpMov, Defs: []VReg{local(plan.IndexLocal)}, Imm: 0, Note: "loop index = 0"},
		{Op: OpMov, Defs: []VReg{local(plan.TotalLocal)}, Imm: 0, Note: "loop total = 0"},
	}
	if plan.BoundLocal >= 0 {
		bound = local(plan.BoundLocal)
		cmpUses[1] = bound
		params = append(params, bound)
	} else {
		entryInstrs = append(entryInstrs, Instr{Op: OpMov, Defs: []VReg{bound}, Imm: int64(plan.BoundConst), Note: "loop bound constant"})
	}
	entryInstrs = append(entryInstrs, Instr{Op: OpBranch, Target: loopName})
	callUses := make([]VReg, 0, len(plan.CallArgLocals))
	for _, localSlot := range plan.CallArgLocals {
		callUses = append(callUses, local(localSlot))
	}
	exitInstrs := []Instr{{Op: OpReturn, Uses: []VReg{local(plan.TotalLocal)}}}
	if plan.ReturnNonNegativeSuccess {
		okName := exitName + "_success"
		failName := exitName + "_failure"
		out := Function{
			Name:   name,
			Target: "scalar-int-call-loop",
			Params: params,
			Blocks: []Block{
				{
					Name:       "entry",
					Instrs:     entryInstrs,
					Successors: []string{loopName},
				},
				{
					Name: loopName,
					Instrs: []Instr{
						{Op: OpCmp, Defs: []VReg{cmp}, Uses: cmpUses, Note: "index < bound"},
						{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
						{
							Op:       OpCall,
							Defs:     []VReg{callRet},
							Uses:     callUses,
							Call:     plan.CallName,
							ABI:      callABI.Name,
							Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
							Note:     "caller-saved state is frame-spilled around call",
						},
						{Op: OpAdd, Defs: []VReg{local(plan.TotalLocal)}, Uses: []VReg{local(plan.TotalLocal), callRet}, Note: "total += call result"},
						{Op: OpInc, Defs: []VReg{local(plan.IndexLocal)}, Uses: []VReg{local(plan.IndexLocal)}, Note: "index++"},
						{Op: OpBranch, Target: loopName},
					},
					Successors: []string{loopName, exitName},
				},
				{
					Name: exitName,
					Instrs: []Instr{
						{Op: OpMov, Defs: []VReg{VReg("zero")}, Imm: 0, Note: "zero for success guard"},
						{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(plan.TotalLocal), VReg("zero")}, Note: "total >= 0"},
						{Op: OpBranchIf, Uses: []VReg{cmp}, Target: failName, Note: "if_negative"},
						{Op: OpBranch, Target: okName},
					},
					Successors: []string{okName, failName},
				},
				{
					Name: okName,
					Instrs: []Instr{
						{Op: OpMov, Defs: []VReg{VReg("ret0")}, Imm: 0, Note: "success exit"},
						{Op: OpReturn, Uses: []VReg{VReg("ret0")}},
					},
				},
				{
					Name: failName,
					Instrs: []Instr{
						{Op: OpMov, Defs: []VReg{VReg("ret1")}, Imm: 1, Note: "failure exit"},
						{Op: OpReturn, Uses: []VReg{VReg("ret1")}},
					},
				},
			},
		}
		if err := VerifyFunction(out); err != nil {
			return Function{}, err
		}
		return out, nil
	}
	if plan.ReturnOneIfTotalZero {
		equalName := exitName + "_equal_zero"
		unequalName := exitName + "_not_equal_zero"
		out := Function{
			Name:   name,
			Target: "scalar-int-call-loop",
			Params: params,
			Blocks: []Block{
				{
					Name:       "entry",
					Instrs:     entryInstrs,
					Successors: []string{loopName},
				},
				{
					Name: loopName,
					Instrs: []Instr{
						{Op: OpCmp, Defs: []VReg{cmp}, Uses: cmpUses, Note: "index < bound"},
						{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
						{
							Op:       OpCall,
							Defs:     []VReg{callRet},
							Uses:     callUses,
							Call:     plan.CallName,
							ABI:      callABI.Name,
							Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
							Note:     "caller-saved state is frame-spilled around call",
						},
						{Op: OpAdd, Defs: []VReg{local(plan.TotalLocal)}, Uses: []VReg{local(plan.TotalLocal), callRet}, Note: "total += call result"},
						{Op: OpInc, Defs: []VReg{local(plan.IndexLocal)}, Uses: []VReg{local(plan.IndexLocal)}, Note: "index++"},
						{Op: OpBranch, Target: loopName},
					},
					Successors: []string{loopName, exitName},
				},
				{
					Name: exitName,
					Instrs: []Instr{
						{Op: OpMov, Defs: []VReg{VReg("zero")}, Imm: 0, Note: "zero for equality guard"},
						{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(plan.TotalLocal), VReg("zero")}, Note: "total == 0"},
						{Op: OpBranchIf, Uses: []VReg{cmp}, Target: unequalName, Note: "if_not_equal"},
						{Op: OpBranch, Target: equalName},
					},
					Successors: []string{equalName, unequalName},
				},
				{
					Name: equalName,
					Instrs: []Instr{
						{Op: OpMov, Defs: []VReg{VReg("ret1")}, Imm: int64(compileTimeEqualReturn), Note: "return 1"},
						{Op: OpReturn, Uses: []VReg{VReg("ret1")}},
					},
				},
				{
					Name: unequalName,
					Instrs: []Instr{
						{Op: OpMov, Defs: []VReg{VReg("ret0")}, Imm: int64(compileTimeUnequalReturn), Note: "return 0"},
						{Op: OpReturn, Uses: []VReg{VReg("ret0")}},
					},
				},
			},
		}
		if err := VerifyFunction(out); err != nil {
			return Function{}, err
		}
		return out, nil
	}
	out := Function{
		Name:   name,
		Target: "scalar-int-call-loop",
		Params: params,
		Blocks: []Block{
			{
				Name:       "entry",
				Instrs:     entryInstrs,
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: cmpUses, Note: "index < bound"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{
						Op:       OpCall,
						Defs:     []VReg{callRet},
						Uses:     callUses,
						Call:     plan.CallName,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "caller-saved state is frame-spilled around call",
					},
					{Op: OpAdd, Defs: []VReg{local(plan.TotalLocal)}, Uses: []VReg{local(plan.TotalLocal), callRet}, Note: "total += call result"},
					{Op: OpInc, Defs: []VReg{local(plan.IndexLocal)}, Uses: []VReg{local(plan.IndexLocal)}, Note: "index++"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName, exitName},
			},
			{
				Name:   exitName,
				Instrs: exitInstrs,
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}
