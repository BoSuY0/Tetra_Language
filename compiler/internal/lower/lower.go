package lower

import (
	"fmt"
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	lowermodel "tetra_language/compiler/internal/lower/model"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/semantics"
)

type Options = lowermodel.Options

type runtimePolicy struct {
	hasBudget    bool
	budget       int32
	consentParam string
}

const consentTokenRuntimeSentinel int32 = -0x43544f4b

func runtimePolicyFromClauses(clauses []frontend.SemanticClause) runtimePolicy {
	policy := runtimePolicy{}
	for _, clause := range clauses {
		switch clause.Name {
		case "budget":
			if v, ok := clauseConstI32(clause.Value); ok {
				policy.hasBudget = true
				policy.budget = v
			}
		case "consent":
			if ident, ok := clause.Value.(*frontend.IdentExpr); ok {
				policy.consentParam = ident.Name
			}
		}
	}
	return policy
}

func clauseConstI32(expr frontend.Expr) (int32, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return e.Value, true
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenMinus {
			return 0, false
		}
		v, ok := e.X.(*frontend.NumberExpr)
		if !ok {
			return 0, false
		}
		return -v.Value, true
	default:
		return 0, false
	}
}

type budgetCharge struct {
	kind ir.IRInstrKind
	cost int32
}

const (
	policyFailureDefaultSlot int32 = 0
	policyFailureStatusTrap  int32 = 1
)

var budgetChargeTable = []budgetCharge{
	{kind: ir.IRWrite, cost: 1},
	{kind: ir.IRCall, cost: 1},
	{kind: ir.IRAllocBytes, cost: 1},
	{kind: ir.IRMakeSliceU8, cost: 1},
	{kind: ir.IRMakeSliceU16, cost: 1},
	{kind: ir.IRMakeSliceI32, cost: 1},
	{kind: ir.IRStackSliceU8, cost: 1},
	{kind: ir.IRStackSliceU16, cost: 1},
	{kind: ir.IRStackSliceI32, cost: 1},
	{kind: ir.IRRegionEnter, cost: 1},
	{kind: ir.IRRegionMakeSliceU8, cost: 1},
	{kind: ir.IRRegionMakeSliceU16, cost: 1},
	{kind: ir.IRRegionMakeSliceI32, cost: 1},
	{kind: ir.IRRegionReset, cost: 1},
	{kind: ir.IRRawSliceFromParts, cost: 1},
	{kind: ir.IRSliceWindow, cost: 1},
	{kind: ir.IRSlicePrefix, cost: 1},
	{kind: ir.IRSliceSuffix, cost: 1},
	{kind: ir.IRIndexLoadI32, cost: 1},
	{kind: ir.IRIndexLoadI32Unchecked, cost: 1},
	{kind: ir.IRIndexStoreI32, cost: 1},
	{kind: ir.IRIndexLoadU8, cost: 1},
	{kind: ir.IRIndexLoadU8Unchecked, cost: 1},
	{kind: ir.IRIndexStoreU8, cost: 1},
	{kind: ir.IRIndexLoadU16, cost: 1},
	{kind: ir.IRIndexLoadU16Unchecked, cost: 1},
	{kind: ir.IRIndexStoreU16, cost: 1},
	{kind: ir.IRIslandNew, cost: 1},
	{kind: ir.IRIslandMakeSliceU8, cost: 1},
	{kind: ir.IRIslandMakeSliceU16, cost: 1},
	{kind: ir.IRIslandMakeSliceI32, cost: 1},
	{kind: ir.IRIslandFree, cost: 1},
	{kind: ir.IRIslandReset, cost: 1},
	{kind: ir.IRCapIO, cost: 1},
	{kind: ir.IRCapMem, cost: 1},
	{kind: ir.IRMemReadI32, cost: 1},
	{kind: ir.IRMemWriteI32, cost: 1},
	{kind: ir.IRMemReadU8, cost: 1},
	{kind: ir.IRMemWriteU8, cost: 1},
	{kind: ir.IRMemReadPtr, cost: 1},
	{kind: ir.IRMemWritePtr, cost: 1},
	{kind: ir.IRMemWriteArchPtr, cost: 1},
	{kind: ir.IRMemReadI32Offset, cost: 1},
	{kind: ir.IRMemWriteI32Offset, cost: 1},
	{kind: ir.IRMemReadU8Offset, cost: 1},
	{kind: ir.IRMemWriteU8Offset, cost: 1},
	{kind: ir.IRMemReadPtrOffset, cost: 1},
	{kind: ir.IRMemWritePtrOffset, cost: 1},
	{kind: ir.IRMemWriteArchPtrOffset, cost: 1},
	{kind: ir.IRPtrAdd, cost: 1},
	{kind: ir.IRMmioReadI32, cost: 1},
	{kind: ir.IRMmioWriteI32, cost: 1},
	{kind: ir.IRSymAddr, cost: 1},
	{kind: ir.IRCtxSwitch, cost: 1},
	{kind: ir.IRAtomicLoadPtr, cost: 1},
	{kind: ir.IRAtomicStorePtr, cost: 1},
	{kind: ir.IRAtomicExchangePtr, cost: 1},
	{kind: ir.IRAtomicFetchAddPtr, cost: 1},
	{kind: ir.IRAtomicFetchSubPtr, cost: 1},
	{kind: ir.IRAtomicFetchAndPtr, cost: 1},
	{kind: ir.IRAtomicFetchOrPtr, cost: 1},
	{kind: ir.IRAtomicFetchXorPtr, cost: 1},
	{kind: ir.IRAtomicCompareExchangePtr, cost: 1},
	{kind: ir.IRAtomicFenceSeqCst, cost: 1},
	{kind: ir.IRAtomicFenceRelaxed, cost: 1},
	{kind: ir.IRAtomicFenceAcquire, cost: 1},
	{kind: ir.IRAtomicFenceRelease, cost: 1},
	{kind: ir.IRAtomicFenceAcqRel, cost: 1},
	{kind: ir.IRAtomicLoadI32, cost: 1},
	{kind: ir.IRAtomicStoreI32, cost: 1},
	{kind: ir.IRAtomicExchangeI32, cost: 1},
	{kind: ir.IRAtomicCompareExchangeI32, cost: 1},
	{kind: ir.IRAtomicFetchAddI32, cost: 1},
	{kind: ir.IRAtomicFetchSubI32, cost: 1},
	{kind: ir.IRAtomicFetchAndI32, cost: 1},
	{kind: ir.IRAtomicFetchOrI32, cost: 1},
	{kind: ir.IRAtomicFetchXorI32, cost: 1},
	{kind: ir.IRAtomicLoadI64, cost: 1},
	{kind: ir.IRAtomicStoreI64, cost: 1},
	{kind: ir.IRAtomicExchangeI64, cost: 1},
	{kind: ir.IRAtomicCompareExchangeI64, cost: 1},
	{kind: ir.IRAtomicFetchAddI64, cost: 1},
	{kind: ir.IRAtomicFetchSubI64, cost: 1},
	{kind: ir.IRAtomicFetchAndI64, cost: 1},
	{kind: ir.IRAtomicFetchOrI64, cost: 1},
	{kind: ir.IRAtomicFetchXorI64, cost: 1},
	{kind: ir.IRAtomicLoadI8, cost: 1},
	{kind: ir.IRAtomicStoreI8, cost: 1},
	{kind: ir.IRAtomicExchangeI8, cost: 1},
	{kind: ir.IRAtomicCompareExchangeI8, cost: 1},
	{kind: ir.IRAtomicFetchAddI8, cost: 1},
	{kind: ir.IRAtomicFetchSubI8, cost: 1},
	{kind: ir.IRAtomicFetchAndI8, cost: 1},
	{kind: ir.IRAtomicFetchOrI8, cost: 1},
	{kind: ir.IRAtomicFetchXorI8, cost: 1},
	{kind: ir.IRAtomicLoadI16, cost: 1},
	{kind: ir.IRAtomicStoreI16, cost: 1},
	{kind: ir.IRAtomicExchangeI16, cost: 1},
	{kind: ir.IRAtomicCompareExchangeI16, cost: 1},
	{kind: ir.IRAtomicFetchAddI16, cost: 1},
	{kind: ir.IRAtomicFetchSubI16, cost: 1},
	{kind: ir.IRAtomicFetchAndI16, cost: 1},
	{kind: ir.IRAtomicFetchOrI16, cost: 1},
	{kind: ir.IRAtomicFetchXorI16, cost: 1},
}

func budgetChargeForInstr(kind ir.IRInstrKind) (int32, bool) {
	for _, charge := range budgetChargeTable {
		if charge.kind == kind {
			return charge.cost, true
		}
	}
	return 0, false
}

func budgetChargedInstr(kind ir.IRInstrKind) bool {
	_, ok := budgetChargeForInstr(kind)
	return ok
}

func Lower(checked *semantics.CheckedProgram) (*ir.IRProgram, error) {
	return LowerWithOptions(checked, Options{})
}

func LowerWithOptions(checked *semantics.CheckedProgram, opt Options) (*ir.IRProgram, error) {
	if checked == nil {
		return nil, fmt.Errorf("missing checked program")
	}
	if len(checked.Funcs) == 0 {
		return nil, fmt.Errorf("expected at least one function")
	}
	plirProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		return nil, err
	}
	if err := plir.VerifyProgram(plirProg); err != nil {
		return nil, err
	}
	allocationPlan, err := allocplan.FromPLIRWithOptions(plirProg, allocationPlannerOptions(opt))
	if err != nil {
		return nil, err
	}
	allocationsByFunction := allocationPlanByFunction(allocationPlan)

	prog := ir.IRProgram{MainIndex: checked.MainIndex, MainName: checked.MainName}
	wrappers := collectTypedTaskWrappers(checked, "")
	stagedTargets := collectStagedTypedTaskTargets(wrappers)
	callableTargets := collectFunctionTypedParamTargets(checked, "")
	for _, fn := range checked.Funcs {
		irFunc, err := lowerCheckedFuncWithOptions(fn, checked.Types, checked.FuncSigs, checked.GlobalsByModule[fn.Module], stagedTargets[fn.Name], callableTargets[fn.Name], opt, allocationsByFunction[fn.Name])
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		prog.Funcs = append(prog.Funcs, irFunc)
	}
	for _, wrapper := range wrappers {
		irFunc, err := lowerTypedTaskWrapper(wrapper)
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		prog.Funcs = append(prog.Funcs, irFunc)
	}
	if err := VerifyProgram(&prog); err != nil {
		return nil, err
	}
	return &prog, nil
}

func LowerModule(checked *semantics.CheckedProgram, module string) ([]ir.IRFunc, error) {
	return LowerModuleWithOptions(checked, module, Options{})
}

func LowerModuleWithOptions(checked *semantics.CheckedProgram, module string, opt Options) ([]ir.IRFunc, error) {
	if checked == nil {
		return nil, fmt.Errorf("missing checked program")
	}
	plirProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		return nil, err
	}
	if err := plir.VerifyProgram(plirProg); err != nil {
		return nil, err
	}
	allocationPlan, err := allocplan.FromPLIRWithOptions(plirProg, allocationPlannerOptions(opt))
	if err != nil {
		return nil, err
	}
	allocationsByFunction := allocationPlanByFunction(allocationPlan)
	var out []ir.IRFunc
	wrappers := collectTypedTaskWrappers(checked, module)
	stagedTargets := collectStagedTypedTaskTargets(wrappers)
	callableTargets := collectFunctionTypedParamTargets(checked, "")
	for _, fn := range checked.Funcs {
		if fn.Module != module {
			continue
		}
		irFunc, err := lowerCheckedFuncWithOptions(fn, checked.Types, checked.FuncSigs, checked.GlobalsByModule[fn.Module], stagedTargets[fn.Name], callableTargets[fn.Name], opt, allocationsByFunction[fn.Name])
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		out = append(out, irFunc)
	}
	for _, wrapper := range wrappers {
		irFunc, err := lowerTypedTaskWrapper(wrapper)
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		out = append(out, irFunc)
	}
	return out, nil
}

func allocationPlanByFunction(plan *allocplan.Plan) map[string]map[string]allocplan.Allocation {
	out := map[string]map[string]allocplan.Allocation{}
	if plan == nil {
		return out
	}
	for _, fn := range plan.Functions {
		row := map[string]allocplan.Allocation{}
		for _, alloc := range fn.Allocations {
			row[alloc.ID] = alloc
		}
		out[fn.Name] = row
	}
	return out
}

func allocationPlannerOptions(opt Options) allocplan.Options {
	return allocplan.Options{
		EnableStackLowering:  opt.StackAllocationLowering,
		EnableRegionPlanning: opt.FunctionTempRegionLowering,
		EnableRegionLowering: opt.FunctionTempRegionLowering,
	}
}

func LowerModules(checked *semantics.CheckedProgram) (map[string][]ir.IRFunc, error) {
	if checked == nil {
		return nil, fmt.Errorf("missing checked program")
	}
	modules := make(map[string][]ir.IRFunc)
	wrappers := collectTypedTaskWrappers(checked, "")
	stagedTargets := collectStagedTypedTaskTargets(wrappers)
	callableTargets := collectFunctionTypedParamTargets(checked, "")
	for _, fn := range checked.Funcs {
		irFunc, err := lowerCheckedFunc(fn, checked.Types, checked.FuncSigs, checked.GlobalsByModule[fn.Module], stagedTargets[fn.Name], callableTargets[fn.Name])
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		modules[fn.Module] = append(modules[fn.Module], irFunc)
	}
	for _, wrapper := range wrappers {
		irFunc, err := lowerTypedTaskWrapper(wrapper)
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		modules[wrapper.Module] = append(modules[wrapper.Module], irFunc)
	}
	return modules, nil
}

func lowerCheckedFunc(fn semantics.CheckedFunc, types map[string]*semantics.TypeInfo, funcs map[string]semantics.FuncSig, globals map[string]semantics.GlobalInfo, stagedTarget typedTaskStagedTarget, callableParamTargets map[string][]string) (ir.IRFunc, error) {
	return lowerCheckedFuncWithOptions(fn, types, funcs, globals, stagedTarget, callableParamTargets, Options{}, nil)
}

func lowerCheckedFuncWithOptions(fn semantics.CheckedFunc, types map[string]*semantics.TypeInfo, funcs map[string]semantics.FuncSig, globals map[string]semantics.GlobalInfo, stagedTarget typedTaskStagedTarget, callableParamTargets map[string][]string, opt Options, allocationPlan map[string]allocplan.Allocation) (ir.IRFunc, error) {
	throwSuccessSlots := 0
	throwErrorSlots := 0
	throwCompact := false
	throwScratchBase := 0
	if fn.ThrowsType != "" {
		var err error
		throwSuccessSlots, throwErrorSlots, throwCompact, err = throwingLayout(fn.ReturnType, fn.ThrowsType, types)
		if err != nil {
			return ir.IRFunc{}, err
		}
		throwScratchBase = fn.LocalSlots - throwErrorSlots
		if throwScratchBase < 0 {
			return ir.IRFunc{}, fmt.Errorf("internal error: invalid throwing scratch layout for '%s'", fn.Name)
		}
	}
	policy := runtimePolicyFromClauses(fn.Decl.SemanticClauses)
	localSlots := fn.LocalSlots
	budgetLocal := -1
	if policy.hasBudget {
		budgetLocal = localSlots
		localSlots++
	}
	effectiveReturnSlots := fn.ReturnSlots
	if stagedTarget.SlotCount > 4 {
		effectiveReturnSlots = 1
	}
	inoutReturnLocals := []inoutReturnLocal(nil)
	if fn.ThrowsType == "" && stagedTarget.SlotCount <= 4 {
		var err error
		inoutReturnLocals, err = collectInoutReturnLocals(fn)
		if err != nil {
			return ir.IRFunc{}, err
		}
	}
	abiReturnSlots := effectiveReturnSlots + inoutReturnSlotCount(inoutReturnLocals)
	l := &lowerer{
		locals:                     fn.Locals,
		actorState:                 fn.ActorState,
		globals:                    globals,
		types:                      types,
		funcs:                      funcs,
		imports:                    fn.Imports,
		module:                     fn.Module,
		localSlots:                 localSlots,
		returnType:                 fn.ReturnType,
		throwsType:                 fn.ThrowsType,
		returnSlots:                effectiveReturnSlots,
		abiReturnSlots:             abiReturnSlots,
		inoutReturnLocals:          inoutReturnLocals,
		throwSuccessSlots:          throwSuccessSlots,
		throwErrorSlots:            throwErrorSlots,
		throwCompact:               throwCompact,
		throwScratchBase:           throwScratchBase,
		policyFailLabel:            -1,
		budgetEnabled:              policy.hasBudget,
		budgetLocal:                budgetLocal,
		discardLocal:               -1,
		budgetScratchBase:          -1,
		stagedTaskTarget:           stagedTarget,
		callableParamTargets:       callableParamTargets,
		allocationPlan:             allocationPlan,
		stackAllocationLowering:    opt.StackAllocationLowering,
		functionTempRegionLowering: opt.FunctionTempRegionLowering,
		scalarSlices:               map[string]scalarSliceLocal{},
		rawPtrOffsetLocals:         map[int]rawPtrOffsetLocal{},
		zeroLocals:                 map[string]bool{},
		constIntLocals:             map[string]int64{},
		lenBoundLocals:             map[string]string{},
		externalSliceLocals:        map[string]bool{},
		invalidSliceLocals:         map[string]bool{},
	}
	if policy.hasBudget || policy.consentParam != "" {
		l.policyFailLabel = l.newLabel()
	}
	if policy.hasBudget {
		l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: policy.budget, Pos: fn.Decl.Pos})
		l.emitRaw(ir.IRInstr{Kind: ir.IRStoreLocal, Local: budgetLocal, Pos: fn.Decl.Pos})
	}
	if policy.consentParam != "" {
		info, ok := l.locals[policy.consentParam]
		if !ok {
			return ir.IRFunc{}, fmt.Errorf("%s: semantic clause 'consent' references unknown local '%s' during lowering", frontend.FormatPos(fn.Decl.Pos), policy.consentParam)
		}
		if info.SlotCount != 1 {
			return ir.IRFunc{}, fmt.Errorf("%s: semantic clause 'consent' expects 1-slot token parameter '%s'", frontend.FormatPos(fn.Decl.Pos), policy.consentParam)
		}
		l.emitRaw(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: fn.Decl.Pos})
		l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: consentTokenRuntimeSentinel, Pos: fn.Decl.Pos})
		l.emitRaw(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: fn.Decl.Pos})
		l.emitRaw(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: l.policyFailLabel, Pos: fn.Decl.Pos})
	}
	if err := l.lowerBlock(fn.Decl.Body, fn.Decl.Pos); err != nil {
		return ir.IRFunc{}, err
	}
	if l.policyFailLabel >= 0 {
		l.emitPolicyFailureHandler(fn.Decl.Pos)
	}
	irPolicy := ir.IRPolicy{
		HasBudget:    policy.hasBudget,
		Budget:       policy.budget,
		BudgetLocal:  budgetLocal,
		HasConsent:   policy.consentParam != "",
		ConsentLocal: -1,
		FailLabel:    l.policyFailLabel,
	}
	if policy.consentParam != "" {
		irPolicy.ConsentLocal = l.locals[policy.consentParam].Base
	}
	return ir.IRFunc{
		Name:        fn.Name,
		ExportName:  fn.Decl.ExportName,
		ParamSlots:  fn.ParamSlots,
		LocalSlots:  l.localSlots,
		ReturnSlots: l.abiReturnSlots,
		Policy:      irPolicy,
		Instrs:      l.instrs,
	}, nil
}

// lowerer owns function-local IR emission state. stackHeight is emission-side
// bookkeeping used to preserve local invariants while lowering; VerifyFunc is
// still the final target-neutral check before codegen sees the function.
