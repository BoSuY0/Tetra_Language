package machine

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
)

const (
	recursionFibName          = "p25.recursion.fib"
	recursionMainName         = "p25.recursion.main"
	recursionFibBaseCaseLimit = int32(2)
	recursionMainLoopBound    = int32(40)
	recursionMainCallArg      = int32(10)
	recursionMainSuccessTotal = int32(2200)
)

type RecursionFibPlan struct {
	Function   Function
	ParamLocal int
	BaseLabel  int
	CallName   string
}

type RecursionMainPlan struct {
	Function       Function
	IndexLocal     int
	TotalLocal     int
	StartLabel     int
	EndLabel       int
	FailureLabel   int
	CallName       string
	LoopBound      int32
	CallArg        int32
	SuccessTotal   int32
	TrueReturnImm  int32
	FalseReturnImm int32
}

func RecursionFibFunctionFromStackIRWithCallABI(fn ir.IRFunc, callABI CallABIInfo) (Function, bool, error) {
	plan, ok, err := RecursionFibPlanFromStackIRWithCallABI(fn, callABI)
	return plan.Function, ok, err
}

func RecursionFibPlanFromStackIRWithCallABI(fn ir.IRFunc, callABI CallABIInfo) (RecursionFibPlan, bool, error) {
	if fn.Name != recursionFibName || fn.ParamSlots != 1 || fn.LocalSlots != 1 || fn.ReturnSlots != 1 || len(fn.Instrs) != 17 {
		return RecursionFibPlan{}, false, nil
	}
	if err := validateCallABIInfo(callABI); err != nil {
		return RecursionFibPlan{}, true, err
	}
	if callABI.MaxArgSlots < 1 || callABI.MaxRetSlots < 1 {
		return RecursionFibPlan{}, false, nil
	}
	in := fn.Instrs
	if !isLoad(in[0], 0) ||
		in[1].Kind != ir.IRConstI32 || in[1].Imm != recursionFibBaseCaseLimit ||
		in[2].Kind != ir.IRCmpLtI32 ||
		in[3].Kind != ir.IRJmpIfZero {
		return RecursionFibPlan{}, false, nil
	}
	baseLabel := in[3].Label
	if baseLabel < 0 {
		return RecursionFibPlan{}, false, nil
	}
	if !isLoad(in[4], 0) || in[5].Kind != ir.IRReturn ||
		in[6].Kind != ir.IRLabel || in[6].Label != baseLabel {
		return RecursionFibPlan{}, false, nil
	}
	if !isLoad(in[7], 0) ||
		in[8].Kind != ir.IRConstI32 || in[8].Imm != 1 ||
		in[9].Kind != ir.IRSubI32 ||
		in[10].Kind != ir.IRCall || in[10].Name != fn.Name || in[10].ArgSlots != 1 || in[10].RetSlots != 1 {
		return RecursionFibPlan{}, false, nil
	}
	if !isLoad(in[11], 0) ||
		in[12].Kind != ir.IRConstI32 || in[12].Imm != recursionFibBaseCaseLimit ||
		in[13].Kind != ir.IRSubI32 ||
		in[14].Kind != ir.IRCall || in[14].Name != fn.Name || in[14].ArgSlots != 1 || in[14].RetSlots != 1 ||
		in[15].Kind != ir.IRAddI32 || in[16].Kind != ir.IRReturn {
		return RecursionFibPlan{}, false, nil
	}

	plan := RecursionFibPlan{
		ParamLocal: 0,
		BaseLabel:  baseLabel,
		CallName:   fn.Name,
	}
	out, err := buildRecursionFibMachineFunction(fn.Name, callABI, plan)
	if err != nil {
		return RecursionFibPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func RecursionMainFunctionFromStackIRWithCallABI(fn ir.IRFunc, callABI CallABIInfo) (Function, bool, error) {
	plan, ok, err := RecursionMainPlanFromStackIRWithCallABI(fn, callABI)
	return plan.Function, ok, err
}

func RecursionMainPlanFromStackIRWithCallABI(fn ir.IRFunc, callABI CallABIInfo) (RecursionMainPlan, bool, error) {
	if fn.Name != recursionMainName || fn.ParamSlots != 0 || fn.LocalSlots != 2 || fn.ReturnSlots != 1 || len(fn.Instrs) != 29 {
		return RecursionMainPlan{}, false, nil
	}
	if err := validateCallABIInfo(callABI); err != nil {
		return RecursionMainPlan{}, true, err
	}
	if callABI.MaxArgSlots < 1 || callABI.MaxRetSlots < 1 {
		return RecursionMainPlan{}, false, nil
	}
	in := fn.Instrs
	if !isConstStore(in[0], in[1], 0) || !isConstStore(in[2], in[3], 0) {
		return RecursionMainPlan{}, false, nil
	}
	indexLocal := in[1].Local
	totalLocal := in[3].Local
	startLabel := in[4].Label
	if in[4].Kind != ir.IRLabel || startLabel < 0 {
		return RecursionMainPlan{}, false, nil
	}
	if !isLoad(in[5], indexLocal) ||
		in[6].Kind != ir.IRConstI32 || in[6].Imm != recursionMainLoopBound ||
		in[7].Kind != ir.IRCmpLtI32 ||
		in[8].Kind != ir.IRJmpIfZero {
		return RecursionMainPlan{}, false, nil
	}
	endLabel := in[8].Label
	if endLabel < 0 || endLabel == startLabel {
		return RecursionMainPlan{}, false, nil
	}
	if !isLoad(in[9], totalLocal) ||
		in[10].Kind != ir.IRConstI32 || in[10].Imm != recursionMainCallArg ||
		in[11].Kind != ir.IRCall || in[11].Name != recursionFibName || in[11].ArgSlots != 1 || in[11].RetSlots != 1 ||
		in[12].Kind != ir.IRAddI32 || !isStore(in[13], totalLocal) {
		return RecursionMainPlan{}, false, nil
	}
	if !isLoad(in[14], indexLocal) ||
		in[15].Kind != ir.IRConstI32 || in[15].Imm != 1 ||
		in[16].Kind != ir.IRAddI32 || !isStore(in[17], indexLocal) ||
		in[18].Kind != ir.IRJmp || in[18].Label != startLabel ||
		in[19].Kind != ir.IRLabel || in[19].Label != endLabel {
		return RecursionMainPlan{}, false, nil
	}
	if !isLoad(in[20], totalLocal) ||
		in[21].Kind != ir.IRConstI32 || in[21].Imm != recursionMainSuccessTotal ||
		in[22].Kind != ir.IRCmpEqI32 ||
		in[23].Kind != ir.IRJmpIfZero {
		return RecursionMainPlan{}, false, nil
	}
	failLabel := in[23].Label
	if failLabel < 0 || failLabel == startLabel || failLabel == endLabel {
		return RecursionMainPlan{}, false, nil
	}
	if in[24].Kind != ir.IRConstI32 || in[24].Imm != 0 ||
		in[25].Kind != ir.IRReturn ||
		in[26].Kind != ir.IRLabel || in[26].Label != failLabel ||
		in[27].Kind != ir.IRConstI32 || in[27].Imm != 1 ||
		in[28].Kind != ir.IRReturn {
		return RecursionMainPlan{}, false, nil
	}
	if err := validateScalarLoopLocal(fn, indexLocal, "index"); err != nil {
		return RecursionMainPlan{}, true, err
	}
	if err := validateScalarLoopLocal(fn, totalLocal, "total"); err != nil {
		return RecursionMainPlan{}, true, err
	}
	if indexLocal == totalLocal {
		return RecursionMainPlan{}, false, nil
	}

	plan := RecursionMainPlan{
		IndexLocal:     indexLocal,
		TotalLocal:     totalLocal,
		StartLabel:     startLabel,
		EndLabel:       endLabel,
		FailureLabel:   failLabel,
		CallName:       in[11].Name,
		LoopBound:      recursionMainLoopBound,
		CallArg:        recursionMainCallArg,
		SuccessTotal:   recursionMainSuccessTotal,
		TrueReturnImm:  0,
		FalseReturnImm: 1,
	}
	out, err := buildRecursionMainMachineFunction(fn.Name, callABI, plan)
	if err != nil {
		return RecursionMainPlan{}, true, err
	}
	plan.Function = out
	return plan, true, nil
}

func buildRecursionFibMachineFunction(name string, callABI CallABIInfo, plan RecursionFibPlan) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	baseConst := VReg("base")
	one := VReg("one")
	two := VReg("two")
	cmp := VReg("cmp")
	nMinusOne := VReg("n_minus_one")
	nMinusTwo := VReg("n_minus_two")
	first := VReg("fib_n_minus_one")
	second := VReg("fib_n_minus_two")
	sum := VReg("sum")
	baseName := scalarLoopLabelName(plan.BaseLabel)
	recurseName := baseName + "_recurse"
	out := Function{
		Name:   name,
		Target: "recursion-fib",
		Params: []VReg{local(plan.ParamLocal)},
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{baseConst}, Imm: int64(recursionFibBaseCaseLimit), Note: "fib base threshold"},
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(plan.ParamLocal), baseConst}, Note: "n < 2"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: baseName, Note: "base case"},
					{Op: OpBranch, Target: recurseName},
				},
				Successors: []string{baseName, recurseName},
			},
			{
				Name: baseName,
				Instrs: []Instr{
					{Op: OpReturn, Uses: []VReg{local(plan.ParamLocal)}, Note: "return n"},
				},
			},
			{
				Name: recurseName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{one}, Imm: 1},
					{Op: OpSub, Defs: []VReg{nMinusOne}, Uses: []VReg{local(plan.ParamLocal), one}, Note: "n - 1"},
					{
						Op:       OpCall,
						Defs:     []VReg{first},
						Uses:     []VReg{nMinusOne},
						Call:     plan.CallName,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "recursive fib(n - 1); caller-saved state is frame-spilled around call",
					},
					{Op: OpMov, Defs: []VReg{two}, Imm: int64(recursionFibBaseCaseLimit)},
					{Op: OpSub, Defs: []VReg{nMinusTwo}, Uses: []VReg{local(plan.ParamLocal), two}, Note: "n - 2"},
					{
						Op:       OpCall,
						Defs:     []VReg{second},
						Uses:     []VReg{nMinusTwo},
						Call:     plan.CallName,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "recursive fib(n - 2); first result is frame-spilled around call",
					},
					{Op: OpAdd, Defs: []VReg{sum}, Uses: []VReg{first, second}, Note: "fib(n - 1) + fib(n - 2)"},
					{Op: OpReturn, Uses: []VReg{sum}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}

func buildRecursionMainMachineFunction(name string, callABI CallABIInfo, plan RecursionMainPlan) (Function, error) {
	local := func(slot int) VReg { return VReg(fmt.Sprintf("local%d", slot)) }
	bound := VReg("bound")
	callArg := VReg("call_arg")
	callRet := VReg("fib_ret")
	expected := VReg("expected")
	cmp := VReg("cmp")
	ret0 := VReg("ret0")
	ret1 := VReg("ret1")
	loopName := scalarLoopLabelName(plan.StartLabel)
	exitName := scalarLoopLabelName(plan.EndLabel)
	failName := scalarLoopLabelName(plan.FailureLabel)
	okName := exitName + "_success"
	out := Function{
		Name:   name,
		Target: "recursion-main-loop",
		Blocks: []Block{
			{
				Name: "entry",
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{local(plan.IndexLocal)}, Imm: 0, Note: "loop index = 0"},
					{Op: OpMov, Defs: []VReg{local(plan.TotalLocal)}, Imm: 0, Note: "loop total = 0"},
					{Op: OpMov, Defs: []VReg{bound}, Imm: int64(plan.LoopBound), Note: "loop bound 40"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{loopName},
			},
			{
				Name: loopName,
				Instrs: []Instr{
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(plan.IndexLocal), bound}, Note: "i < 40"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: exitName, Note: "if_zero"},
					{Op: OpMov, Defs: []VReg{callArg}, Imm: int64(plan.CallArg), Note: "fib(10)"},
					{
						Op:       OpCall,
						Defs:     []VReg{callRet},
						Uses:     []VReg{callArg},
						Call:     plan.CallName,
						ABI:      callABI.Name,
						Clobbers: append([]PhysReg(nil), callABI.Clobbers...),
						Note:     "loop calls fib(10); total/index are frame-spilled around call",
					},
					{Op: OpAdd, Defs: []VReg{local(plan.TotalLocal)}, Uses: []VReg{local(plan.TotalLocal), callRet}, Note: "total += fib(10)"},
					{Op: OpInc, Defs: []VReg{local(plan.IndexLocal)}, Uses: []VReg{local(plan.IndexLocal)}, Note: "i++"},
					{Op: OpBranch, Target: loopName},
				},
				Successors: []string{exitName, loopName},
			},
			{
				Name: exitName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{expected}, Imm: int64(plan.SuccessTotal), Note: "success total 2200"},
					{Op: OpCmp, Defs: []VReg{cmp}, Uses: []VReg{local(plan.TotalLocal), expected}, Note: "total == 2200"},
					{Op: OpBranchIf, Uses: []VReg{cmp}, Target: failName, Note: "if_not_equal"},
					{Op: OpBranch, Target: okName},
				},
				Successors: []string{failName, okName},
			},
			{
				Name: okName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{ret0}, Imm: int64(plan.TrueReturnImm), Note: "return 0"},
					{Op: OpReturn, Uses: []VReg{ret0}},
				},
			},
			{
				Name: failName,
				Instrs: []Instr{
					{Op: OpMov, Defs: []VReg{ret1}, Imm: int64(plan.FalseReturnImm), Note: "return 1"},
					{Op: OpReturn, Uses: []VReg{ret1}},
				},
			},
		},
	}
	if err := VerifyFunction(out); err != nil {
		return Function{}, err
	}
	return out, nil
}
