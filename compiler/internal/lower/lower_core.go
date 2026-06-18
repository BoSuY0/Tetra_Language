package lower

import (
	"fmt"
	"sort"
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	lowermodel "tetra_language/compiler/internal/lower/model"
	lowertasks "tetra_language/compiler/internal/lower/tasks"
	"tetra_language/compiler/internal/plir"
	corerangeproof "tetra_language/compiler/internal/rangeproof"
	"tetra_language/compiler/internal/runtimeabi"
	"tetra_language/compiler/internal/semantics"
)

// ---- diagnostics.go ----

const (
	DiagnosticCodeIRVerifier       = "TETRA3001"
	DiagnosticCodeLowerUnsupported = "TETRA3002"
)

func irVerifierError(format string, args ...interface{}) error {
	return lowerDiagnostic(
		frontend.Position{},
		DiagnosticCodeIRVerifier,
		fmt.Sprintf(format, args...),
		"Fix the IR producer before backend codegen.",
	)
}

func irVerifierErrorAt(pos frontend.Position, format string, args ...interface{}) error {
	return lowerDiagnostic(
		pos,
		DiagnosticCodeIRVerifier,
		fmt.Sprintf(format, args...),
		"Fix the IR producer before backend codegen.",
	)
}

func lowerUnsupportedError(pos frontend.Position, format string, args ...interface{}) error {
	return lowerDiagnostic(
		pos,
		DiagnosticCodeLowerUnsupported,
		fmt.Sprintf(format, args...),
		"This syntax reached lowering without a supported IR translation.",
	)
}

func lowerDiagnostic(pos frontend.Position, code string, message string, hint string) error {
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code:     code,
		Message:  message,
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint:     hint,
	}}
}

// ---- lower.go ----

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
	callBoundaryProofs := corerangeproof.CollectHashLookupCallBoundaryLenProofs(checked)
	helperSummaryProofs := corerangeproof.CollectHelperSummaryProofs(checked)
	helperOffsetProofs := corerangeproof.CollectHelperOffsetProofs(checked)

	prog := ir.IRProgram{MainIndex: checked.MainIndex, MainName: checked.MainName}
	wrappers := collectTypedTaskWrappers(checked, "")
	stagedTargets := collectStagedTypedTaskTargets(wrappers)
	callableTargets := collectFunctionTypedParamTargets(checked, "")
	for _, fn := range checked.Funcs {
		irFunc, err := lowerCheckedFuncWithOptions(
			fn,
			checked.Types,
			checked.FuncSigs,
			checked.GlobalsByModule[fn.Module],
			stagedTargets[fn.Name],
			callableTargets[fn.Name],
			opt,
			allocationsByFunction[fn.Name],
			callBoundaryProofs[fn.Name],
			helperSummaryProofs[fn.Name],
			helperOffsetProofs[fn.Name],
		)
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

func LowerModuleWithOptions(
	checked *semantics.CheckedProgram,
	module string,
	opt Options,
) ([]ir.IRFunc, error) {
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
	callBoundaryProofs := corerangeproof.CollectHashLookupCallBoundaryLenProofs(checked)
	helperSummaryProofs := corerangeproof.CollectHelperSummaryProofs(checked)
	helperOffsetProofs := corerangeproof.CollectHelperOffsetProofs(checked)
	var out []ir.IRFunc
	wrappers := collectTypedTaskWrappers(checked, module)
	stagedTargets := collectStagedTypedTaskTargets(wrappers)
	callableTargets := collectFunctionTypedParamTargets(checked, "")
	for _, fn := range checked.Funcs {
		if fn.Module != module {
			continue
		}
		irFunc, err := lowerCheckedFuncWithOptions(
			fn,
			checked.Types,
			checked.FuncSigs,
			checked.GlobalsByModule[fn.Module],
			stagedTargets[fn.Name],
			callableTargets[fn.Name],
			opt,
			allocationsByFunction[fn.Name],
			callBoundaryProofs[fn.Name],
			helperSummaryProofs[fn.Name],
			helperOffsetProofs[fn.Name],
		)
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
	callBoundaryProofs := corerangeproof.CollectHashLookupCallBoundaryLenProofs(checked)
	helperSummaryProofs := corerangeproof.CollectHelperSummaryProofs(checked)
	helperOffsetProofs := corerangeproof.CollectHelperOffsetProofs(checked)
	for _, fn := range checked.Funcs {
		irFunc, err := lowerCheckedFuncWithOptions(
			fn,
			checked.Types,
			checked.FuncSigs,
			checked.GlobalsByModule[fn.Module],
			stagedTargets[fn.Name],
			callableTargets[fn.Name],
			Options{},
			nil,
			callBoundaryProofs[fn.Name],
			helperSummaryProofs[fn.Name],
			helperOffsetProofs[fn.Name],
		)
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

func lowerCheckedFunc(
	fn semantics.CheckedFunc,
	types map[string]*semantics.TypeInfo,
	funcs map[string]semantics.FuncSig,
	globals map[string]semantics.GlobalInfo,
	stagedTarget typedTaskStagedTarget,
	callableParamTargets map[string][]string,
) (ir.IRFunc, error) {
	return lowerCheckedFuncWithOptions(
		fn,
		types,
		funcs,
		globals,
		stagedTarget,
		callableParamTargets,
		Options{},
		nil,
		corerangeproof.CallBoundaryLenProof{},
		corerangeproof.HelperSummaryProof{},
		corerangeproof.HelperOffsetProof{},
	)
}

func lowerCheckedFuncWithOptions(
	fn semantics.CheckedFunc,
	types map[string]*semantics.TypeInfo,
	funcs map[string]semantics.FuncSig,
	globals map[string]semantics.GlobalInfo,
	stagedTarget typedTaskStagedTarget,
	callableParamTargets map[string][]string,
	opt Options,
	allocationPlan map[string]allocplan.Allocation,
	callBoundaryProof corerangeproof.CallBoundaryLenProof,
	helperSummaryProof corerangeproof.HelperSummaryProof,
	helperOffsetProof corerangeproof.HelperOffsetProof,
) (ir.IRFunc, error) {
	throwSuccessSlots := 0
	throwErrorSlots := 0
	throwCompact := false
	throwScratchBase := 0
	if fn.ThrowsType != "" {
		var err error
		throwSuccessSlots, throwErrorSlots, throwCompact, err = throwingLayout(
			fn.ReturnType,
			fn.ThrowsType,
			types,
		)
		if err != nil {
			return ir.IRFunc{}, err
		}
		throwScratchBase = fn.LocalSlots - throwErrorSlots
		if throwScratchBase < 0 {
			return ir.IRFunc{}, fmt.Errorf(
				"internal error: invalid throwing scratch layout for '%s'",
				fn.Name,
			)
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
		callBoundaryLenProof:       callBoundaryProof,
		helperSummaryProof:         helperSummaryProof,
		helperOffsetProof:          helperOffsetProof,
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
			return ir.IRFunc{}, fmt.Errorf(
				"%s: semantic clause 'consent' references unknown local '%s' during lowering",
				frontend.FormatPos(fn.Decl.Pos),
				policy.consentParam,
			)
		}
		if info.SlotCount != 1 {
			return ir.IRFunc{}, fmt.Errorf(
				"%s: semantic clause 'consent' expects 1-slot token parameter '%s'",
				frontend.FormatPos(fn.Decl.Pos),
				policy.consentParam,
			)
		}
		l.emitRaw(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: fn.Decl.Pos})
		l.emitRaw(
			ir.IRInstr{Kind: ir.IRConstI32, Imm: consentTokenRuntimeSentinel, Pos: fn.Decl.Pos},
		)
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

// ---- lower_stmts.go ----

func (l *lowerer) lowerBlock(stmts []frontend.Stmt, pos frontend.Position) error {
	frameIndex := len(l.deferFrames)
	l.deferFrames = append(l.deferFrames, deferFrame{})
	for _, stmt := range stmts {
		if err := l.lowerStmt(stmt); err != nil {
			l.deferFrames = l.deferFrames[:frameIndex]
			return err
		}
	}
	if err := l.emitDeferredFrame(frameIndex, pos); err != nil {
		l.deferFrames = l.deferFrames[:frameIndex]
		return err
	}
	l.deferFrames = l.deferFrames[:frameIndex]
	return nil
}

func (l *lowerer) emitDeferredFrame(frameIndex int, pos frontend.Position) error {
	if frameIndex < 0 || frameIndex >= len(l.deferFrames) {
		return nil
	}
	bodies := l.deferFrames[frameIndex].bodies
	for i := len(bodies) - 1; i >= 0; i-- {
		if err := l.lowerBlock(bodies[i], pos); err != nil {
			return err
		}
	}
	return nil
}

func (l *lowerer) emitDeferredFramesSince(start int, pos frontend.Position) error {
	end := len(l.deferFrames) - 1
	for i := end; i >= start; i-- {
		if err := l.emitDeferredFrame(i, pos); err != nil {
			return err
		}
	}
	return nil
}

func (l *lowerer) prepareGlobalStringFieldAccessesForStmt(
	stmt frontend.Stmt,
) map[string]frontend.Position {
	prepared := map[string]frontend.Position{}
	var collectExpr func(frontend.Expr)
	collectExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.FieldAccessExpr:
			baseName, fields, _, ok := splitFieldPathLower(e)
			if ok && len(fields) > 0 {
				if g, exists := l.globals[baseName]; exists && g.TypeName == "str" && g.HasStringLiteralInit {
					prepared[baseName] = e.At
				}
			}
			collectExpr(e.Base)
		case *frontend.IndexExpr:
			collectExpr(e.Base)
			collectExpr(e.Index)
		case *frontend.BinaryExpr:
			collectExpr(e.Left)
			collectExpr(e.Right)
		case *frontend.UnaryExpr:
			collectExpr(e.X)
		case *frontend.CallExpr:
			for _, arg := range e.Args {
				collectExpr(arg)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				collectExpr(field.Value)
			}
		case *frontend.MatchExpr:
			collectExpr(e.Value)
			for _, c := range e.Cases {
				if c.Pattern != nil {
					collectExpr(c.Pattern)
				}
				if c.Guard != nil {
					collectExpr(c.Guard)
				}
				collectExpr(c.Value)
			}
		case *frontend.CatchExpr:
			collectExpr(e.Call)
			for _, c := range e.Cases {
				if c.Pattern != nil {
					collectExpr(c.Pattern)
				}
				if c.Guard != nil {
					collectExpr(c.Guard)
				}
				collectExpr(c.Value)
			}
		case *frontend.TryExpr:
			collectExpr(e.X)
		case *frontend.AwaitExpr:
			collectExpr(e.X)
		}
	}

	switch s := stmt.(type) {
	case *frontend.PrintStmt:
		collectExpr(s.Value)
	case *frontend.FreeStmt:
		collectExpr(s.Value)
	case *frontend.ReturnStmt:
		collectExpr(s.Value)
	case *frontend.ThrowStmt:
		collectExpr(s.Value)
	case *frontend.IslandStmt:
		collectExpr(s.Size)
	case *frontend.LetStmt:
		collectExpr(s.Value)
	case *frontend.AssignStmt:
		collectExpr(s.Target)
		collectExpr(s.Value)
	case *frontend.IfStmt:
		collectExpr(s.Cond)
	case *frontend.IfLetStmt:
		collectExpr(s.Value)
	case *frontend.WhileStmt:
		collectExpr(s.Cond)
	case *frontend.ForRangeStmt:
		if s.Iterable != nil {
			collectExpr(s.Iterable)
		} else {
			collectExpr(s.Start)
			collectExpr(s.End)
		}
	case *frontend.MatchStmt:
		collectExpr(s.Value)
		for _, c := range s.Cases {
			if c.Pattern != nil {
				collectExpr(c.Pattern)
			}
			if c.Guard != nil {
				collectExpr(c.Guard)
			}
		}
	case *frontend.ExprStmt:
		collectExpr(s.Expr)
	}

	if len(prepared) == 0 {
		return nil
	}
	names := make([]string, 0, len(prepared))
	for name := range prepared {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		l.emitGlobalStringLiteralInitIfNeeded(l.globals[name], prepared[name])
	}
	return prepared
}

func (l *lowerer) lowerStmt(stmt frontend.Stmt) error {
	prepared := l.prepareGlobalStringFieldAccessesForStmt(stmt)
	if len(prepared) == 0 {
		return l.lowerStmtPrepared(stmt)
	}
	old := l.preparedStringFields
	merged := make(map[string]bool, len(old)+len(prepared))
	for name := range old {
		merged[name] = true
	}
	for name := range prepared {
		merged[name] = true
	}
	l.preparedStringFields = merged
	err := l.lowerStmtPrepared(stmt)
	l.preparedStringFields = old
	return err
}

func (l *lowerer) lowerStmtPrepared(stmt frontend.Stmt) error {
	switch s := stmt.(type) {
	case *frontend.PrintStmt:
		slots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		if slots != 2 {
			return fmt.Errorf("%s: print expects str or []u8", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRWrite, Pos: s.At})
	case *frontend.FreeStmt:
		slots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		if slots != 1 {
			return fmt.Errorf("%s: free expects island (1 slot)", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandFree, Pos: s.At})
	case *frontend.ReturnStmt:
		if l.stagedTaskTarget.SlotCount > 4 {
			valueSlots, err := l.lowerExprAs(s.Value, l.returnType)
			if err != nil {
				return err
			}
			if valueSlots != 1 {
				return fmt.Errorf("%s: staged typed task return expects 1-slot value", frontend.FormatPos(s.At))
			}
			valueLocal := l.allocScratchSlots(1)
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: valueLocal, Pos: s.At})
			if err := l.emitStageTypedTaskFromLocals(
				valueLocal,
				-1,
				l.stagedTaskTarget.SlotCount,
				0,
				s.At,
			); err != nil {
				return err
			}
			if err := l.emitDeferredFramesSince(0, s.At); err != nil {
				return err
			}
			l.emitCleanup(s.At)
			l.emitFunctionTempRegionReset(s.At)
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: s.At})
			return nil
		}
		slots := 0
		if closure, ok := s.Value.(*frontend.ClosureExpr); ok && l.returnType == "fnptr" {
			if l.returnSlots == semantics.CallableHandleSlotCount {
				slots = l.emitCallableHandleValue(l.closureSymbolName(closure), closure.Captures, closure.At)
			} else {
				slots = l.emitFunctionSymbolValue(
					l.closureSymbolName(closure),
					l.closureEnvLocals(closure.Captures),
					closure.At,
				)
			}
		} else if id, ok := s.Value.(*frontend.IdentExpr); ok && l.returnType == "fnptr" {
			if info, exists := l.locals[id.Name]; exists && info.FunctionValue != "" && len(
				info.FunctionCaptures,
			) > 0 {
				if l.returnSlots == semantics.CallableHandleSlotCount || info.FunctionHandleValue || len(
					l.closureEnvLocalsUnbounded(info.FunctionCaptures),
				) > semantics.FnPtrEnvSlotCount {
					slots = l.emitCallableHandleValue(info.FunctionValue, info.FunctionCaptures, s.At)
				} else {
					slots = l.emitFunctionSymbolValue(info.FunctionValue, l.capturedClosureEnvLocals(info), s.At)
				}
			}
		} else if target, ok := importedFunctionTargetFromExpr(s.Value, l.imports, l.funcs); ok {
			slots = l.emitFunctionSymbolValue(target, nil, s.At)
		}
		if slots == 0 {
			var err error
			slots, err = l.lowerExprAs(s.Value, l.returnType)
			if err != nil {
				return err
			}
		}
		expectedSlots := l.returnSlots
		if l.throwsType != "" {
			expectedSlots = l.throwSuccessSlots
		}
		if slots != expectedSlots {
			return fmt.Errorf("%s: return slot mismatch", frontend.FormatPos(s.At))
		}
		if l.throwsType != "" {
			if !l.throwCompact {
				l.emitZeroSlots(l.throwErrorSlots, s.At)
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: s.At})
		}
		if err := l.emitDeferredFramesSince(0, s.At); err != nil {
			return err
		}
		l.emitCleanup(s.At)
		if l.throwsType == "" {
			l.emitInoutReturnSlots(s.At)
		}
		l.emitFunctionTempRegionReset(s.At)
		l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: s.At})
	case *frontend.ThrowStmt:
		if l.stagedTaskTarget.SlotCount > 4 {
			if l.throwsType == "" {
				return fmt.Errorf("%s: throw is only allowed in throwing functions", frontend.FormatPos(s.At))
			}
			slots, err := l.lowerExprAs(s.Value, l.throwsType)
			if err != nil {
				return err
			}
			if slots != l.throwErrorSlots {
				return fmt.Errorf("%s: throw slot mismatch", frontend.FormatPos(s.At))
			}
			errBase := l.allocScratchSlots(l.throwErrorSlots)
			for slot := l.throwErrorSlots - 1; slot >= 0; slot-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: errBase + slot, Pos: s.At})
			}
			if err := l.emitStageTypedTaskFromLocals(
				-1,
				errBase,
				l.stagedTaskTarget.SlotCount,
				1,
				s.At,
			); err != nil {
				return err
			}
			if err := l.emitDeferredFramesSince(0, s.At); err != nil {
				return err
			}
			l.emitCleanup(s.At)
			l.emitFunctionTempRegionReset(s.At)
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: s.At})
			return nil
		}
		if l.throwsType == "" {
			return fmt.Errorf("%s: throw is only allowed in throwing functions", frontend.FormatPos(s.At))
		}
		if !l.throwCompact {
			l.emitZeroSlots(l.throwSuccessSlots, s.At)
		}
		slots, err := l.lowerExprAs(s.Value, l.throwsType)
		if err != nil {
			return err
		}
		if slots != l.throwErrorSlots {
			return fmt.Errorf("%s: throw slot mismatch", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: s.At})
		if err := l.emitDeferredFramesSince(0, s.At); err != nil {
			return err
		}
		l.emitCleanup(s.At)
		l.emitFunctionTempRegionReset(s.At)
		l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: s.At})
	case *frontend.DeferStmt:
		if len(l.deferFrames) == 0 {
			return fmt.Errorf("%s: defer outside block", frontend.FormatPos(s.At))
		}
		frameIndex := len(l.deferFrames) - 1
		l.deferFrames[frameIndex].bodies = append(l.deferFrames[frameIndex].bodies, s.Body)
	case *frontend.BreakStmt:
		loop, ok := l.currentLoop()
		if !ok {
			return fmt.Errorf("%s: break outside loop", frontend.FormatPos(s.At))
		}
		if err := l.emitDeferredFramesSince(loop.deferDepth, s.At); err != nil {
			return err
		}
		l.emitCleanupSince(loop.cleanupDepth, s.At)
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: loop.breakLabel, Pos: s.At})
	case *frontend.ContinueStmt:
		loop, ok := l.currentLoop()
		if !ok {
			return fmt.Errorf("%s: continue outside loop", frontend.FormatPos(s.At))
		}
		if err := l.emitDeferredFramesSince(loop.deferDepth, s.At); err != nil {
			return err
		}
		l.emitCleanupSince(loop.cleanupDepth, s.At)
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: loop.continueLabel, Pos: s.At})
	case *frontend.IslandStmt:
		slots, err := l.lowerExpr(s.Size)
		if err != nil {
			return err
		}
		if slots != 1 {
			return fmt.Errorf("%s: island size must be i32", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRIslandNew, Pos: s.At})
		info, ok := l.locals[s.Name]
		if !ok {
			return fmt.Errorf("unknown local '%s'", s.Name)
		}
		if info.SlotCount != 1 {
			return fmt.Errorf("%s: island slot mismatch", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base, Pos: s.At})
		l.cleanupIslands = append(l.cleanupIslands, info.Base)
		if err := l.lowerBlock(s.Body, s.At); err != nil {
			return err
		}
		l.cleanupIslands = l.cleanupIslands[:len(l.cleanupIslands)-1]
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRIslandFree, Pos: s.At})
	case *frontend.LetStmt:
		info, ok := l.locals[s.Name]
		if !ok {
			return fmt.Errorf("unknown local '%s'", s.Name)
		}
		slots := 0
		if info.FunctionTypeValue {
			if _, ok := s.Value.(*frontend.ClosureExpr); ok && info.FunctionValue != "" {
				if info.FunctionHandleValue {
					closure := s.Value.(*frontend.ClosureExpr)
					slots = l.emitCallableHandleValue(info.FunctionValue, closure.Captures, s.At)
				} else {
					slots = l.emitFunctionSymbolValue(info.FunctionValue, l.capturedClosureEnvLocals(info), s.At)
				}
			} else if id, ok := s.Value.(*frontend.IdentExpr); ok && info.FunctionValue != "" {
				if source, ok := l.locals[id.Name]; ok && source.FunctionTypeValue {
					for slot := 0; slot < source.SlotCount; slot++ {
						l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: source.Base + slot, Pos: s.At})
					}
					slots = source.SlotCount
				} else if source, ok := l.locals[id.Name]; ok && !source.FunctionTypeValue && source.FunctionValue != "" && (source.FunctionHandleValue || len(
					l.closureEnvLocalsUnbounded(source.FunctionCaptures),
				) > semantics.FnPtrEnvSlotCount) {
					slots = l.emitCallableHandleValue(source.FunctionValue, source.FunctionCaptures, s.At)
				} else if len(info.FunctionCaptures) > 0 {
					slots = l.emitFunctionSymbolValue(info.FunctionValue, l.capturedClosureEnvLocals(info), s.At)
				} else {
					slots = l.emitFunctionSymbolValue(info.FunctionValue, nil, s.At)
				}
			} else if _, ok := functionTypedGlobalFieldTargetFromExpr(
				s.Value,
				l.globals,
			); ok && info.FunctionValue != "" {
				slots = l.emitFunctionSymbolValue(info.FunctionValue, nil, s.At)
			}
		} else if len(info.FunctionFields) > 0 {
			if call, ok := s.Value.(*frontend.CallExpr); ok {
				var handled bool
				var err error
				slots, handled, err = l.lowerStructConstructorCall(call, info.FunctionFields)
				if err != nil {
					return err
				}
				if !handled {
					slots = 0
				}
			} else if lit, ok := s.Value.(*frontend.StructLitExpr); ok {
				var err error
				slots, err = l.lowerStructLiteralExpr(lit, info.FunctionFields)
				if err != nil {
					return err
				}
			}
		} else if len(info.EnumPayloadFunctions) > 0 {
			if call, ok := s.Value.(*frontend.CallExpr); ok {
				var handled bool
				var err error
				slots, handled, err = l.lowerEnumCaseConstructorCall(call, info.EnumPayloadFunctions)
				if err != nil {
					return err
				}
				if !handled {
					slots = 0
				}
			}
		}
		if slots == 0 {
			var lowered bool
			var err error
			lowered, slots, err = l.lowerUnusedCopyLet(s.Name, info, s.Value, s.At)
			if err != nil {
				return err
			}
			if !lowered {
				lowered, slots, err = l.lowerScalarReplacementLet(s.Name, info, s.Value, s.At)
				if err != nil {
					return err
				}
				if !lowered {
					lowered, slots, err = l.lowerFunctionTempRegionCopyLet(s.Name, info, s.Value, s.At)
					if err != nil {
						return err
					}
					if !lowered {
						lowered, slots, err = l.lowerExplicitIslandAllocationLet(s.Name, info, s.Value, s.At)
						if err != nil {
							return err
						}
						if !lowered {
							lowered, slots, err = l.lowerStackCopyLet(s.Name, info, s.Value, s.At)
							if err != nil {
								return err
							}
							if !lowered {
								lowered, slots, err = l.lowerStackAllocationLet(s.Name, info, s.Value, s.At)
								if err != nil {
									return err
								}
								if !lowered {
									slots, err = l.lowerExprAs(s.Value, info.TypeName)
									if err != nil {
										return err
									}
								}
							}
						}
					}
				}
			}
		}
		if slots != info.SlotCount {
			return fmt.Errorf("%s: slot mismatch for '%s'", frontend.FormatPos(s.At), s.Name)
		}
		for i := info.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base + i, Pos: s.At})
		}
		l.rememberRangeMetadataForLocal(s.Name, s.Value)
		if info.SlotCount == 1 {
			l.rememberRawPtrOffsetAlias(info.Base, s.Value)
		}
	case *frontend.AssignStmt:
		if id, ok := s.Target.(*frontend.IdentExpr); ok {
			if info, ok := l.locals[id.Name]; ok && info.ActorField {
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(info.ActorFieldSlot), Pos: s.At})
				slots, err := l.lowerExprAs(s.Value, info.TypeName)
				if err != nil {
					return err
				}
				if slots != 1 {
					return fmt.Errorf(
						"%s: actor state assignment expects single-slot value",
						frontend.FormatPos(s.At),
					)
				}
				l.emit(
					ir.IRInstr{
						Kind:     ir.IRCall,
						Name:     "__tetra_actor_state_store",
						ArgSlots: 2,
						RetSlots: 1,
						Pos:      s.At,
					},
				)
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.ensureDiscardLocal(), Pos: s.At})
				return nil
			}
		}
		if idx, ok := s.Target.(*frontend.IndexExpr); ok {
			if lowered, err := l.lowerScalarIndexStore(idx, s.Value, s.At); lowered || err != nil {
				return err
			}
			elemType, err := l.indexElemType(idx.Base)
			if err != nil {
				return err
			}
			baseSlots, err := l.lowerExpr(idx.Base)
			if err != nil {
				return err
			}
			if baseSlots != 2 {
				return fmt.Errorf("%s: index base slot mismatch", frontend.FormatPos(idx.At))
			}
			idxSlots, err := l.lowerExpr(idx.Index)
			if err != nil {
				return err
			}
			if idxSlots != 1 {
				return fmt.Errorf("%s: index must be i32", frontend.FormatPos(idx.At))
			}
			valSlots, err := l.lowerExpr(s.Value)
			if err != nil {
				return err
			}
			if valSlots != 1 {
				return fmt.Errorf("%s: index assignment expects single-slot value", frontend.FormatPos(s.At))
			}
			targetKind, ok := lowerIndexStoreKind(elemType, l.types)
			if !ok {
				return lowerUnsupportedError(s.At, "unsupported index element type '%s'", elemType)
			}
			store := ir.IRInstr{Kind: targetKind, Pos: s.At}
			if proofID, ok := l.activeWhileProofForIndex(idx); ok {
				store.ProofID = proofID
			}
			l.emit(store)
			return nil
		}
		if id, ok := s.Target.(*frontend.IdentExpr); ok {
			if g, ok := l.globals[id.Name]; ok {
				var slots int
				var err error
				if g.FunctionTypeValue {
					slots, err = l.lowerFunctionTypedLocalAssignmentValue(s.Value, semantics.LocalInfo{
						SlotCount:         gSlotCount(g.TypeName, l.types),
						TypeName:          g.TypeName,
						FunctionTypeValue: true,
					}, s.At)
				} else {
					slots, err = l.lowerExprAs(s.Value, g.TypeName)
				}
				if err != nil {
					return err
				}
				slotCount := gSlotCount(g.TypeName, l.types)
				if slots != slotCount {
					return fmt.Errorf("%s: slot mismatch for assignment", frontend.FormatPos(s.At))
				}
				for i := slotCount - 1; i >= 0; i-- {
					l.emit(ir.IRInstr{Kind: ir.IRStoreGlobal, Local: g.DataIndex + i, Pos: s.At})
				}
				return nil
			}
			if info, ok := l.locals[id.Name]; ok && info.FunctionTypeValue {
				slots, err := l.lowerFunctionTypedLocalAssignmentValue(s.Value, info, s.At)
				if err != nil {
					return err
				}
				if slots != info.SlotCount {
					return fmt.Errorf("%s: slot mismatch for assignment", frontend.FormatPos(s.At))
				}
				for i := info.SlotCount - 1; i >= 0; i-- {
					l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base + i, Pos: s.At})
				}
				return nil
			}
		} else if targetName := functionTypedFieldNameFromExpr(s.Target); targetName != "" {
			if _, ok, _ := resolveFunctionFieldName(targetName, l.locals); ok {
				target, err := l.resolveLValue(s.Target)
				if err != nil {
					return err
				}
				slots, err := l.lowerFunctionTypedLocalAssignmentValue(
					s.Value,
					semantics.LocalInfo{
						SlotCount:         target.SlotCount,
						TypeName:          target.TypeName,
						FunctionTypeValue: true,
					},
					s.At,
				)
				if err != nil {
					return err
				}
				if slots != target.SlotCount {
					return fmt.Errorf("%s: slot mismatch for assignment", frontend.FormatPos(s.At))
				}
				storeKind := ir.IRStoreLocal
				if target.Global {
					storeKind = ir.IRStoreGlobal
				}
				for i := target.SlotCount - 1; i >= 0; i-- {
					l.emit(ir.IRInstr{Kind: storeKind, Local: target.Base + i, Pos: s.At})
				}
				return nil
			}
		}
		target, err := l.resolveLValue(s.Target)
		if err != nil {
			return err
		}
		slots, err := l.lowerExprAs(s.Value, target.TypeName)
		if err != nil {
			return err
		}
		if slots != target.SlotCount {
			return fmt.Errorf("%s: slot mismatch for assignment", frontend.FormatPos(s.At))
		}
		storeKind := ir.IRStoreLocal
		if target.Global {
			storeKind = ir.IRStoreGlobal
		}
		for i := target.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: storeKind, Local: target.Base + i, Pos: s.At})
		}
		if !target.Global && target.SlotCount == 1 {
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if info, ok := l.locals[id.Name]; ok && info.Base == target.Base {
					l.rememberRawPtrOffsetAlias(target.Base, s.Value)
				}
			}
		}
		if !target.Global {
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				delete(l.scalarSlices, id.Name)
				l.rememberRangeMetadataForLocal(id.Name, s.Value)
				l.invalidateWhileRangeProofForLocal(id.Name)
			}
		}
	case *frontend.IfStmt:
		elseLabel := l.newLabel()
		endLabel := -1
		if len(s.Else) > 0 {
			endLabel = l.newLabel()
		}
		proof, hasProof := l.ifRangeProof(s)
		slots, err := l.lowerExpr(s.Cond)
		if err != nil {
			return err
		}
		if slots != 1 {
			return fmt.Errorf("%s: condition must be i32", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: s.At})
		branchState := l.snapshotRangeMetadata()
		if hasProof {
			l.pushWhileRangeProof(proof)
		}
		if err := l.lowerBlock(s.Then, s.At); err != nil {
			if hasProof {
				l.popWhileRangeProof()
			}
			return err
		}
		if hasProof {
			l.popWhileRangeProof()
		}
		thenState := l.snapshotRangeMetadata()
		elseState := branchState
		if len(s.Else) > 0 {
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: s.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: elseLabel, Pos: s.At})
		if len(s.Else) > 0 {
			l.restoreRangeMetadata(branchState)
			if err := l.lowerBlock(s.Else, s.At); err != nil {
				return err
			}
			elseState = l.snapshotRangeMetadata()
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		}
		l.mergeRangeMetadata(thenState, elseState)
	case *frontend.IfLetStmt:
		valueInfo, ok := l.locals[s.ValueLocal]
		if !ok {
			return fmt.Errorf("%s: unknown if-let value local", frontend.FormatPos(s.At))
		}
		slots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		if slots != valueInfo.SlotCount {
			return fmt.Errorf("%s: if-let value slot mismatch", frontend.FormatPos(s.At))
		}
		for i := valueInfo.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: valueInfo.Base + i, Pos: s.At})
		}
		elseLabel := l.newLabel()
		endLabel := -1
		if len(s.Else) > 0 {
			endLabel = l.newLabel()
		}
		if s.Pattern == nil {
			bindInfo, ok := l.locals[s.Name]
			if !ok {
				return fmt.Errorf("%s: unknown if-let local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + bindInfo.SlotCount, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: elseLabel, Pos: s.At})
			for i := 0; i < bindInfo.SlotCount; i++ {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueInfo.Base + i, Pos: s.At})
			}
			for i := bindInfo.SlotCount - 1; i >= 0; i-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + i, Pos: s.At})
			}
		} else {
			if err := l.emitIfLetPatternCheck(s.Pattern, valueInfo, elseLabel, s.At); err != nil {
				return err
			}
			if err := l.emitIfLetPatternBindings(s.Pattern, valueInfo); err != nil {
				return err
			}
		}
		if err := l.lowerBlock(s.Then, s.At); err != nil {
			return err
		}
		if len(s.Else) > 0 {
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: s.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: elseLabel, Pos: s.At})
		if len(s.Else) > 0 {
			if err := l.lowerBlock(s.Else, s.At); err != nil {
				return err
			}
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		}
	case *frontend.WhileStmt:
		startLabel := l.newLabel()
		endLabel := l.newLabel()
		proof, hasProof := l.whileRangeProof(s)
		l.pushLoop(startLabel, endLabel)
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: startLabel, Pos: s.At})
		slots, err := l.lowerExpr(s.Cond)
		if err != nil {
			l.popLoop()
			return err
		}
		if slots != 1 {
			l.popLoop()
			return fmt.Errorf("%s: condition must be i32", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: s.At})
		if hasProof {
			l.pushWhileRangeProof(proof)
		}
		if err := l.lowerBlock(s.Body, s.At); err != nil {
			if hasProof {
				l.popWhileRangeProof()
			}
			l.popLoop()
			return err
		}
		if hasProof {
			l.popWhileRangeProof()
			l.zeroLocals[proof.indexName] = false
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: startLabel, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		l.popLoop()
	case *frontend.ForRangeStmt:
		loopInfo, ok := l.locals[s.Name]
		if !ok {
			return fmt.Errorf("%s: unknown for local '%s'", frontend.FormatPos(s.At), s.Name)
		}
		endInfo, ok := l.locals[s.EndLocal]
		if !ok {
			return fmt.Errorf("%s: unknown for end local", frontend.FormatPos(s.At))
		}
		if s.Iterable != nil {
			iterInfo, ok := l.locals[s.IterableLocal]
			if !ok {
				return fmt.Errorf("%s: unknown for iterable local", frontend.FormatPos(s.At))
			}
			indexInfo, ok := l.locals[s.IndexLocal]
			if !ok {
				return fmt.Errorf("%s: unknown for index local", frontend.FormatPos(s.At))
			}
			iterSlots, err := l.lowerExpr(s.Iterable)
			if err != nil {
				return err
			}
			if iterSlots != iterInfo.SlotCount || iterInfo.SlotCount != 2 {
				return fmt.Errorf("%s: for collection iterable slot mismatch", frontend.FormatPos(s.At))
			}
			for i := iterInfo.SlotCount - 1; i >= 0; i-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: iterInfo.Base + i, Pos: s.At})
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: indexInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: iterInfo.Base + 1, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: endInfo.Base, Pos: s.At})
			startLabel := l.newLabel()
			continueLabel := l.newLabel()
			endLabel := l.newLabel()
			l.pushLoop(continueLabel, endLabel)
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: startLabel, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: indexInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: endInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: iterInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: iterInfo.Base + 1, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: indexInfo.Base, Pos: s.At})
			loadKind, ok := lowerIndexLoadKind(loopInfo.TypeName, l.types)
			if !ok {
				return lowerUnsupportedError(
					s.At,
					"unsupported for collection element type '%s'",
					loopInfo.TypeName,
				)
			}
			if l.collectionIterableProofAllowed(s.Iterable) {
				l.emit(
					ir.IRInstr{
						Kind:    uncheckedIndexLoadKind(loadKind),
						ProofID: forCollectionBoundsProofID(s),
						Pos:     s.At,
					},
				)
			} else {
				l.emit(ir.IRInstr{Kind: loadKind, Pos: s.At})
			}
			if loopInfo.SlotCount != 1 {
				return fmt.Errorf("%s: for collection element slot mismatch", frontend.FormatPos(s.At))
			}
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: loopInfo.Base, Pos: s.At})
			if err := l.lowerBlock(s.Body, s.At); err != nil {
				l.popLoop()
				return err
			}
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: continueLabel, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: indexInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: indexInfo.Base, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: startLabel, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
			l.popLoop()
			return nil
		}
		startSlots, err := l.lowerExpr(s.Start)
		if err != nil {
			return err
		}
		if startSlots != 1 || loopInfo.SlotCount != 1 {
			return fmt.Errorf("%s: for range start slot mismatch", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: loopInfo.Base, Pos: s.At})
		endSlots, err := l.lowerExpr(s.End)
		if err != nil {
			return err
		}
		if endSlots != 1 || endInfo.SlotCount != 1 {
			return fmt.Errorf("%s: for range end slot mismatch", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: endInfo.Base, Pos: s.At})
		startLabel := l.newLabel()
		continueLabel := l.newLabel()
		endLabel := l.newLabel()
		l.pushLoop(continueLabel, endLabel)
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: startLabel, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: loopInfo.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: endInfo.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRCmpLtI32, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: s.At})
		if err := l.lowerBlock(s.Body, s.At); err != nil {
			l.popLoop()
			return err
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: continueLabel, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: loopInfo.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRAddI32, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: loopInfo.Base, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: startLabel, Pos: s.At})
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		l.popLoop()
	case *frontend.MatchStmt:
		info, ok := l.locals[s.ScrutineeLocal]
		if !ok {
			return fmt.Errorf("%s: unknown match scrutinee local", frontend.FormatPos(s.At))
		}
		valueSlots, err := l.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		if valueSlots != info.SlotCount {
			return fmt.Errorf("%s: match value slot mismatch", frontend.FormatPos(s.At))
		}
		for i := info.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: info.Base + i, Pos: s.At})
		}
		endLabel := l.newLabel()
		defaultLabel := -1
		caseLabels := make([]int, len(s.Cases))
		guardFailLabels := make([]int, len(s.Cases))
		scrutTypeInfo, scrutTypeOK := l.types[info.TypeName]
		for i, c := range s.Cases {
			guardFailLabels[i] = endLabel
			caseLabels[i] = l.newLabel()
			if c.Default {
				defaultLabel = caseLabels[i]
				continue
			}
			nextLabel := l.newLabel()
			guardFailLabels[i] = nextLabel
			if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeOptional {
				if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
					l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + info.SlotCount - 1, Pos: c.At})
					l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
					l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
					l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
					continue
				}
				if !isNoneExpr(c.Pattern) {
					return fmt.Errorf(
						"%s: optional match supports only 'none', 'some(name)', and '_' patterns",
						frontend.FormatPos(c.At),
					)
				}
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + info.SlotCount - 1, Pos: c.At})
				l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: c.At})
			} else if scrutTypeOK && scrutTypeInfo.Kind == semantics.TypeEnum {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: c.At})
				switch pat := c.Pattern.(type) {
				case *frontend.FieldAccessExpr:
					if pat.EnumType == "" {
						return fmt.Errorf("%s: enum match pattern was not resolved", frontend.FormatPos(c.At))
					}
					l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
				case *frontend.EnumCasePatternExpr:
					if pat.EnumType == "" {
						return fmt.Errorf("%s: enum match pattern was not resolved", frontend.FormatPos(c.At))
					}
					if err := l.validateEnumPatternLayout(pat, info); err != nil {
						return err
					}
					l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: pat.EnumOrdinal, Pos: c.At})
				default:
					return fmt.Errorf(
						"%s: enum match supports enum case patterns and '_'",
						frontend.FormatPos(c.At),
					)
				}
			} else {
				if info.SlotCount != 1 {
					return fmt.Errorf("%s: match value slot mismatch", frontend.FormatPos(s.At))
				}
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base, Pos: c.At})
				patSlots, err := l.lowerExpr(c.Pattern)
				if err != nil {
					return err
				}
				if patSlots != 1 {
					return fmt.Errorf("%s: match pattern slot mismatch", frontend.FormatPos(c.At))
				}
			}
			l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: c.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: nextLabel, Pos: c.At})
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: caseLabels[i], Pos: c.At})
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: nextLabel, Pos: c.At})
		}
		if defaultLabel >= 0 {
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: defaultLabel, Pos: s.At})
		} else {
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: s.At})
		}
		for i, c := range s.Cases {
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: caseLabels[i], Pos: c.At})
			if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				bindInfo, ok := l.locals[some.Name]
				if !ok {
					return fmt.Errorf("%s: unknown some binding '%s'", frontend.FormatPos(some.At), some.Name)
				}
				if bindInfo.SlotCount != info.SlotCount-1 {
					return fmt.Errorf("%s: optional some binding slot mismatch", frontend.FormatPos(some.At))
				}
				for slot := 0; slot < bindInfo.SlotCount; slot++ {
					l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: info.Base + slot, Pos: some.At})
				}
				for slot := bindInfo.SlotCount - 1; slot >= 0; slot-- {
					l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: bindInfo.Base + slot, Pos: some.At})
				}
			}
			if enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr); ok {
				if err := l.emitIfLetPatternBindings(enumPat, info); err != nil {
					return err
				}
			}
			if c.Guard != nil {
				slots, err := l.lowerExpr(c.Guard)
				if err != nil {
					return err
				}
				if slots != 1 {
					return fmt.Errorf("%s: match guard must be single-slot", frontend.FormatPos(c.Guard.Pos()))
				}
				l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: guardFailLabels[i], Pos: c.Guard.Pos()})
			}
			if err := l.lowerBlock(c.Body, c.At); err != nil {
				return err
			}
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: c.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
	case *frontend.ExprStmt:
		slots, err := l.lowerExpr(s.Expr)
		if err != nil {
			return err
		}
		discardLocal := l.ensureDiscardLocal()
		for i := 0; i < slots; i++ {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discardLocal, Pos: s.At})
		}
	case *frontend.UnsafeStmt:
		return l.lowerBlock(s.Body, s.At)
	default:
		return lowerUnsupportedError(s.Pos(), "unsupported statement kind %T", s)
	}
	return nil
}

// ---- lower_types_tasks.go ----

type lowerer struct {
	instrs                     []ir.IRInstr
	locals                     map[string]semantics.LocalInfo
	actorState                 map[string]semantics.ActorStateField
	globals                    map[string]semantics.GlobalInfo
	types                      map[string]*semantics.TypeInfo
	funcs                      map[string]semantics.FuncSig
	imports                    map[string]string
	module                     string
	localSlots                 int
	returnType                 string
	throwsType                 string
	returnSlots                int
	abiReturnSlots             int
	inoutReturnLocals          []inoutReturnLocal
	throwSuccessSlots          int
	throwErrorSlots            int
	throwCompact               bool
	throwScratchBase           int
	policyFailLabel            int
	budgetEnabled              bool
	budgetLocal                int
	discardLocal               int
	budgetScratchBase          int
	budgetScratchSlots         int
	stagedTaskTarget           typedTaskStagedTarget
	callableParamTargets       map[string][]string
	allocationPlan             map[string]allocplan.Allocation
	stackAllocationLowering    bool
	functionTempRegionLowering bool
	functionTempRegionEntered  bool
	scalarSlices               map[string]scalarSliceLocal
	rawPtrOffsetLocals         map[int]rawPtrOffsetLocal
	preparedStringFields       map[string]bool
	zeroLocals                 map[string]bool
	constIntLocals             map[string]int64
	lenBoundLocals             map[string]string
	callBoundaryLenProof       corerangeproof.CallBoundaryLenProof
	helperSummaryProof         corerangeproof.HelperSummaryProof
	helperOffsetProof          corerangeproof.HelperOffsetProof
	externalSliceLocals        map[string]bool
	invalidSliceLocals         map[string]bool
	whileRangeProofs           []whileRangeProof
	stackHeight                int
	nextLabel                  int
	cleanupIslands             []int
	deferFrames                []deferFrame
	loopStack                  []loopLabels
}

type scalarSliceLocal struct {
	elemType    string
	length      int64
	elementBase int
}

type whileRangeProof struct {
	indexName         string
	baseName          string
	proofID           string
	callBoundaryBases map[string]bool
	active            bool
}

type rawPtrOffsetLocal struct {
	BaseLocal    int
	OffsetLocal  int
	OffsetImm    int32
	HasOffsetImm bool
}

type typedTaskWrapper = lowertasks.Wrapper

type typedTaskStagedTarget = lowertasks.StagedTarget

type inoutReturnLocal struct {
	Base      int
	SlotCount int
}

type inoutWriteback struct {
	Base      int
	SlotCount int
	Global    bool
}

func collectInoutReturnLocals(fn semantics.CheckedFunc) ([]inoutReturnLocal, error) {
	locals := make([]inoutReturnLocal, 0)
	for _, param := range fn.Decl.Params {
		if param.Ownership != "inout" {
			continue
		}
		info, ok := fn.Locals[param.Name]
		if !ok {
			return nil, fmt.Errorf(
				"%s: inout parameter '%s' is missing lowering local metadata",
				frontend.FormatPos(param.At),
				param.Name,
			)
		}
		locals = append(locals, inoutReturnLocal{Base: info.Base, SlotCount: info.SlotCount})
	}
	return locals, nil
}

func inoutReturnSlotCount(locals []inoutReturnLocal) int {
	total := 0
	for _, local := range locals {
		total += local.SlotCount
	}
	return total
}

func inoutWritebackSlotCount(writebacks []inoutWriteback) int {
	total := 0
	for _, writeback := range writebacks {
		total += writeback.SlotCount
	}
	return total
}

func typedTaskWrapperName(target, errorType string) string {
	return lowertasks.WrapperName(target, errorType)
}

func typedActorMessageTagBase(typeName string) int32 {
	return lowertasks.ActorMessageTagBase(typeName)
}

func collectTypedTaskWrappers(checked *semantics.CheckedProgram, module string) []typedTaskWrapper {
	return lowertasks.CollectWrappers(checked, module)
}

func collectStagedTypedTaskTargets(wrappers []typedTaskWrapper) map[string]typedTaskStagedTarget {
	return lowertasks.CollectStagedTargets(wrappers)
}

func lowerTypedTaskWrapper(wrapper typedTaskWrapper) (ir.IRFunc, error) {
	return lowertasks.LowerWrapper(wrapper, lowerUnsupportedError)
}

func lowerCallExprWithName(call *frontend.CallExpr, name string) *frontend.CallExpr {
	return lowertasks.CallExprWithName(call, name)
}

func lowerCallExprWithBuiltinAlias(call *frontend.CallExpr) *frontend.CallExpr {
	return lowertasks.CallExprWithBuiltinAlias(call)
}

// throwingLayout computes the slot layout for typed-error returns. The compact
// path is only valid when both success and error payloads fit in one slot.
func throwingLayout(
	returnType, throwsType string,
	types map[string]*semantics.TypeInfo,
) (int, int, bool, error) {
	return lowertasks.ThrowingLayout(returnType, throwsType, types)
}

func throwingReturnSlotCount(successSlots, errorSlots int) int {
	return lowertasks.ThrowingReturnSlotCount(successSlots, errorSlots)
}

type loopLabels struct {
	continueLabel int
	breakLabel    int
	cleanupDepth  int
	deferDepth    int
}

type deferFrame struct {
	bodies [][]frontend.Stmt
}

func (l *lowerer) newLabel() int {
	id := l.nextLabel
	l.nextLabel++
	return id
}

func (l *lowerer) emit(instr ir.IRInstr) {
	if cost, ok := budgetChargeForInstr(instr.Kind); l.budgetEnabled && l.policyFailLabel >= 0 &&
		ok {
		l.emitBudgetGuardPreservingStack(instr.Pos, cost)
	}
	l.emitRaw(instr)
}

func (l *lowerer) emitRaw(instr ir.IRInstr) {
	l.instrs = append(l.instrs, instr)
	pop, push, _ := stackEffect(instr)
	if l.stackHeight < pop {
		l.stackHeight = 0
	} else {
		l.stackHeight = l.stackHeight - pop + push
	}
	if instr.Kind == ir.IRReturn {
		l.stackHeight = 0
	}
}

func (l *lowerer) emitBudgetGuardPreservingStack(pos frontend.Position, cost int32) {
	depth := l.stackHeight
	if depth == 0 {
		l.emitBudgetGuard(pos, cost)
		return
	}
	base := l.ensureBudgetScratchSlots(depth)
	for slot := depth - 1; slot >= 0; slot-- {
		l.emitRaw(ir.IRInstr{Kind: ir.IRStoreLocal, Local: base + slot, Pos: pos})
	}
	l.emitBudgetGuard(pos, cost)
	for slot := 0; slot < depth; slot++ {
		l.emitRaw(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base + slot, Pos: pos})
	}
}

func (l *lowerer) ensureBudgetScratchSlots(slots int) int {
	if l.budgetScratchBase >= 0 && l.budgetScratchSlots >= slots {
		return l.budgetScratchBase
	}
	if l.budgetScratchBase >= 0 {
		l.localSlots += slots - l.budgetScratchSlots
		l.budgetScratchSlots = slots
		return l.budgetScratchBase
	}
	l.budgetScratchBase = l.localSlots
	l.budgetScratchSlots = slots
	l.localSlots += slots
	return l.budgetScratchBase
}

func (l *lowerer) emitBudgetGuard(pos frontend.Position, cost int32) {
	if l.budgetLocal < 0 {
		return
	}
	l.emitRaw(ir.IRInstr{Kind: ir.IRLoadLocal, Local: l.budgetLocal, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: cost, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRSubI32, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.budgetLocal, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRLoadLocal, Local: l.budgetLocal, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRCmpGeI32, Pos: pos})
	l.emitRaw(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: l.policyFailLabel, Pos: pos})
}

func (l *lowerer) emitCleanup(pos frontend.Position) {
	l.emitCleanupSince(0, pos)
}

func (l *lowerer) emitCleanupSince(start int, pos frontend.Position) {
	for i := len(l.cleanupIslands) - 1; i >= 0; i-- {
		if i < start {
			break
		}
		base := l.cleanupIslands[i]
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base, Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRIslandFree, Pos: pos})
	}
}

func (l *lowerer) emitCleanupRaw(pos frontend.Position) {
	l.emitCleanupRawSince(0, pos)
}

func (l *lowerer) emitCleanupRawSince(start int, pos frontend.Position) {
	for i := len(l.cleanupIslands) - 1; i >= 0; i-- {
		if i < start {
			break
		}
		base := l.cleanupIslands[i]
		l.emitRaw(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base, Pos: pos})
		l.emitRaw(ir.IRInstr{Kind: ir.IRIslandFree, Pos: pos})
	}
}

func (l *lowerer) emitZeroSlots(count int, pos frontend.Position) {
	for i := 0; i < count; i++ {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	}
}

func (l *lowerer) emitZeroSlotsRaw(count int, pos frontend.Position) {
	for i := 0; i < count; i++ {
		l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: policyFailureDefaultSlot, Pos: pos})
	}
}

func (l *lowerer) emitInoutReturnSlots(pos frontend.Position) {
	for _, local := range l.inoutReturnLocals {
		for slot := 0; slot < local.SlotCount; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: local.Base + slot, Pos: pos})
		}
	}
}

func (l *lowerer) collectInoutWritebacks(
	args []frontend.Expr,
	ownership []string,
) ([]inoutWriteback, error) {
	if len(ownership) == 0 {
		return nil, nil
	}
	writebacks := make([]inoutWriteback, 0)
	for i, owner := range ownership {
		if owner != "inout" {
			continue
		}
		if i >= len(args) {
			break
		}
		target, err := l.resolveLValue(args[i])
		if err != nil {
			return nil, err
		}
		if !target.Global && target.Base < 0 {
			return nil, fmt.Errorf(
				"%s: inout writeback target cannot be lowered",
				frontend.FormatPos(args[i].Pos()),
			)
		}
		writebacks = append(writebacks, inoutWriteback{
			Base:      target.Base,
			SlotCount: target.SlotCount,
			Global:    target.Global,
		})
	}
	return writebacks, nil
}

func (l *lowerer) emitInoutWritebacks(writebacks []inoutWriteback, pos frontend.Position) {
	for i := len(writebacks) - 1; i >= 0; i-- {
		writeback := writebacks[i]
		storeKind := ir.IRStoreLocal
		if writeback.Global {
			storeKind = ir.IRStoreGlobal
		}
		for slot := writeback.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: storeKind, Local: writeback.Base + slot, Pos: pos})
		}
	}
}

// emitPolicyFailureHandler is the public lowering ABI for budget exhaustion and
// the current local policy-failure path. Non-throwing functions return their
// normal result shape filled with zero/default slots. Throwing functions return
// the normal throwing result shape with a zero/default error payload and status
// 1, so catch/try observe a deterministic trap-shaped error result.
func (l *lowerer) emitPolicyFailureHandler(pos frontend.Position) {
	l.emitRaw(ir.IRInstr{Kind: ir.IRLabel, Label: l.policyFailLabel, Pos: pos})
	if l.stagedTaskTarget.SlotCount > 4 {
		if err := l.emitStageTypedTaskStatus(
			policyFailureDefaultSlot,
			policyFailureStatusTrap,
			l.stagedTaskTarget.SlotCount,
			pos,
		); err == nil {
			l.emitCleanupRaw(pos)
			l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: policyFailureStatusTrap, Pos: pos})
			l.emitRaw(ir.IRInstr{Kind: ir.IRReturn, Pos: pos})
			return
		}
	}
	if l.throwsType != "" {
		if l.throwCompact {
			l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: policyFailureDefaultSlot, Pos: pos})
		} else {
			l.emitZeroSlotsRaw(l.throwSuccessSlots, pos)
			l.emitZeroSlotsRaw(l.throwErrorSlots, pos)
		}
		l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: policyFailureStatusTrap, Pos: pos})
	} else {
		l.emitZeroSlotsRaw(l.abiReturnSlots, pos)
	}
	l.emitCleanupRaw(pos)
	l.emitRaw(ir.IRInstr{Kind: ir.IRReturn, Pos: pos})
}

func (l *lowerer) emitConvertedThrowFromScratch(
	srcType, dstType string,
	pos frontend.Position,
) (int, error) {
	return l.emitConvertedValueFromScratch(srcType, dstType, l.throwScratchBase, pos)
}

func (l *lowerer) lowerTypedTaskJoin(call *frontend.CallExpr, pos frontend.Position) (int, error) {
	if l.throwsType == "" {
		return 0, fmt.Errorf(
			"%s: try is only allowed in throwing functions",
			frontend.FormatPos(pos),
		)
	}
	if len(call.TypeArgs) != 1 {
		return 0, fmt.Errorf(
			"%s: task_join_i32_typed expects one explicit error type argument",
			frontend.FormatPos(call.At),
		)
	}
	errorType := call.TypeArgs[0].Name
	if errorType == "" {
		return 0, fmt.Errorf(
			"%s: task_join_i32_typed missing resolved error type",
			frontend.FormatPos(call.At),
		)
	}
	if errorType != l.throwsType {
		return 0, fmt.Errorf(
			"%s: thrown error type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(call.At),
			l.throwsType,
			errorType,
		)
	}
	errorInfo, ok := l.types[errorType]
	if !ok || errorInfo.Kind != semantics.TypeEnum {
		return 0, fmt.Errorf(
			"%s: typed task error argument must be an enum",
			frontend.FormatPos(call.TypeArgs[0].At),
		)
	}
	handleType, handleInfo, err := semantics.EnsureTypedTaskHandleType(errorType, l.types)
	if err != nil {
		return 0, fmt.Errorf("%s: %v", frontend.FormatPos(call.TypeArgs[0].At), err)
	}
	if len(call.Args) != 1 {
		return 0, fmt.Errorf(
			"%s: task_join_i32_typed expects 1 argument",
			frontend.FormatPos(call.At),
		)
	}
	slots, err := l.lowerTypedTaskJoinHandleArg(call.Args[0], handleType, handleInfo)
	if err != nil {
		return 0, err
	}
	if slots != handleInfo.SlotCount {
		return 0, fmt.Errorf(
			"%s: task_join_i32_typed handle slot mismatch",
			frontend.FormatPos(call.Args[0].Pos()),
		)
	}
	if handleInfo.SlotCount > 4 {
		statusLocal := l.allocScratchSlots(1)
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     typedTaskJoinRuntimeSymbol(handleInfo.SlotCount),
				ArgSlots: handleInfo.SlotCount,
				RetSlots: 1,
				Pos:      pos,
			},
		)
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: statusLocal, Pos: pos})
		if err := l.emitLoadTypedTaskResultSlots(handleInfo.SlotCount-1, pos); err != nil {
			return 0, err
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: statusLocal, Pos: pos})

		okLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: okLabel, Pos: pos})

		if errorInfo.SlotCount == 1 && l.throwCompact {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: pos})
		} else {
			for slot := errorInfo.SlotCount - 1; slot >= 0; slot-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.throwScratchBase + slot, Pos: pos})
			}
			if errorInfo.SlotCount > 1 {
				discard := l.ensureDiscardLocal()
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
			}
			l.emitZeroSlots(l.throwSuccessSlots, pos)
			propagated, err := l.emitConvertedThrowFromScratch(errorType, l.throwsType, pos)
			if err != nil {
				return 0, err
			}
			if propagated != l.throwErrorSlots {
				return 0, fmt.Errorf("%s: task_join_i32_typed error slot mismatch", frontend.FormatPos(pos))
			}
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: pos})
		}
		l.emitCleanup(pos)
		l.emitFunctionTempRegionReset(pos)
		l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: pos})
		l.emitZeroSlots(handleInfo.SlotCount-1, pos)
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: okLabel, Pos: pos})
		if errorInfo.SlotCount > 1 {
			discard := l.ensureDiscardLocal()
			for slot := 0; slot < errorInfo.SlotCount; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
			}
		}
		return 1, nil
	}
	l.emit(
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     typedTaskJoinRuntimeSymbol(handleInfo.SlotCount),
			ArgSlots: handleInfo.SlotCount,
			RetSlots: handleInfo.SlotCount,
			Pos:      pos,
		},
	)

	okLabel := l.newLabel()
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: okLabel, Pos: pos})

	if errorInfo.SlotCount == 1 && l.throwCompact {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: pos})
	} else {
		for slot := errorInfo.SlotCount - 1; slot >= 0; slot-- {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: l.throwScratchBase + slot, Pos: pos})
		}
		if errorInfo.SlotCount > 1 {
			discard := l.ensureDiscardLocal()
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
		}
		l.emitZeroSlots(l.throwSuccessSlots, pos)
		propagated, err := l.emitConvertedThrowFromScratch(errorType, l.throwsType, pos)
		if err != nil {
			return 0, err
		}
		if propagated != l.throwErrorSlots {
			return 0, fmt.Errorf("%s: task_join_i32_typed error slot mismatch", frontend.FormatPos(pos))
		}
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: pos})
	}
	l.emitCleanup(pos)
	l.emitFunctionTempRegionReset(pos)
	l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: pos})
	l.emitZeroSlots(handleInfo.SlotCount-1, pos)
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: okLabel, Pos: pos})
	if errorInfo.SlotCount > 1 {
		discard := l.ensureDiscardLocal()
		for slot := 0; slot < errorInfo.SlotCount; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
		}
	}
	return 1, nil
}

func (l *lowerer) lowerTypedTaskJoinForCatch(
	call *frontend.CallExpr,
	pos frontend.Position,
) (int, error) {
	if len(call.TypeArgs) != 1 {
		return 0, fmt.Errorf(
			"%s: task_join_i32_typed expects one explicit error type argument",
			frontend.FormatPos(call.At),
		)
	}
	errorType := call.TypeArgs[0].Name
	if errorType == "" {
		return 0, fmt.Errorf(
			"%s: task_join_i32_typed missing resolved error type",
			frontend.FormatPos(call.At),
		)
	}
	if info, ok := l.types[errorType]; !ok || info.Kind != semantics.TypeEnum {
		return 0, fmt.Errorf(
			"%s: typed task error argument must be an enum",
			frontend.FormatPos(call.TypeArgs[0].At),
		)
	}
	handleType, handleInfo, err := semantics.EnsureTypedTaskHandleType(errorType, l.types)
	if err != nil {
		return 0, fmt.Errorf("%s: %v", frontend.FormatPos(call.TypeArgs[0].At), err)
	}
	if len(call.Args) != 1 {
		return 0, fmt.Errorf(
			"%s: task_join_i32_typed expects 1 argument",
			frontend.FormatPos(call.At),
		)
	}
	slots, err := l.lowerTypedTaskJoinHandleArg(call.Args[0], handleType, handleInfo)
	if err != nil {
		return 0, err
	}
	if slots != handleInfo.SlotCount {
		return 0, fmt.Errorf(
			"%s: task_join_i32_typed handle slot mismatch",
			frontend.FormatPos(call.Args[0].Pos()),
		)
	}
	if handleInfo.SlotCount > 4 {
		statusLocal := l.allocScratchSlots(1)
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     typedTaskJoinRuntimeSymbol(handleInfo.SlotCount),
				ArgSlots: handleInfo.SlotCount,
				RetSlots: 1,
				Pos:      pos,
			},
		)
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: statusLocal, Pos: pos})
		if err := l.emitLoadTypedTaskResultSlots(handleInfo.SlotCount-1, pos); err != nil {
			return 0, err
		}
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: statusLocal, Pos: pos})
		return handleInfo.SlotCount, nil
	}
	l.emit(
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     typedTaskJoinRuntimeSymbol(handleInfo.SlotCount),
			ArgSlots: handleInfo.SlotCount,
			RetSlots: handleInfo.SlotCount,
			Pos:      pos,
		},
	)
	return handleInfo.SlotCount, nil
}

func isTypedTaskJoinCall(name string) bool {
	return lowertasks.IsTypedTaskJoinCall(name)
}

func typedTaskJoinRuntimeSymbol(slotCount int) string {
	return lowertasks.TypedTaskJoinRuntimeSymbol(slotCount)
}

func (l *lowerer) lowerTypedTaskJoinHandleArg(
	expr frontend.Expr,
	handleType string,
	handleInfo *semantics.TypeInfo,
) (int, error) {
	argType, err := l.inferExprType(expr)
	if err != nil {
		return 0, err
	}
	if argType != handleType && !semantics.TypedTaskHandleTypesCompatible(handleType, argType) {
		return 0, fmt.Errorf(
			"%s: task_join_i32_typed expects a %s handle",
			frontend.FormatPos(expr.Pos()),
			handleType,
		)
	}
	slots, err := l.lowerExpr(expr)
	if err != nil {
		return 0, err
	}
	if slots == handleInfo.SlotCount {
		return slots, nil
	}
	if argType == "task.i32" && semantics.IsTypedTaskHandleTypeName(handleType) && slots == 2 {
		return l.expandPublicTypedTaskHandleSlots(expr.Pos(), handleInfo.SlotCount), nil
	}
	return slots, nil
}

func (l *lowerer) expandPublicTypedTaskHandleSlots(pos frontend.Position, targetSlots int) int {
	if targetSlots <= 2 {
		return 2
	}
	statusLocal := l.allocScratchSlots(1)
	handleLocal := l.allocScratchSlots(1)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: statusLocal, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: handleLocal, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: handleLocal, Pos: pos})
	l.emitZeroSlots(targetSlots-2, pos)
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: statusLocal, Pos: pos})
	return targetSlots
}

func (l *lowerer) emitLoadTypedTaskResultSlots(count int, pos frontend.Position) error {
	if count < 0 || count > 8 {
		return fmt.Errorf(
			"%s: staged typed task slot count %d is out of range",
			frontend.FormatPos(pos),
			count,
		)
	}
	for slot := 0; slot < count; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot), Pos: pos})
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_result_get",
				ArgSlots: 1,
				RetSlots: 1,
				Pos:      pos,
			},
		)
	}
	return nil
}

func (l *lowerer) emitStageTypedTaskStatus(
	value int32,
	status int32,
	slots int,
	pos frontend.Position,
) error {
	if slots < 5 || slots > 8 {
		return fmt.Errorf(
			"%s: staged typed task slots out of range: %d",
			frontend.FormatPos(pos),
			slots,
		)
	}
	discard := l.ensureDiscardLocal()
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slots), Pos: pos})
	l.emit(
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     "__tetra_task_result_begin",
			ArgSlots: 1,
			RetSlots: 1,
			Pos:      pos,
		},
	)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: value, Pos: pos})
	l.emit(
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     "__tetra_task_result_slot",
			ArgSlots: 2,
			RetSlots: 1,
			Pos:      pos,
		},
	)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
	for slot := 1; slot < slots-1; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot), Pos: pos})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_result_slot",
				ArgSlots: 2,
				RetSlots: 1,
				Pos:      pos,
			},
		)
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
	}
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slots - 1), Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: status, Pos: pos})
	l.emit(
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     "__tetra_task_result_slot",
			ArgSlots: 2,
			RetSlots: 1,
			Pos:      pos,
		},
	)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
	return nil
}

func (l *lowerer) emitStageTypedTaskFromLocals(
	valueLocal int,
	errBase int,
	slots int,
	status int32,
	pos frontend.Position,
) error {
	if slots < 5 || slots > 8 {
		return fmt.Errorf(
			"%s: staged typed task slots out of range: %d",
			frontend.FormatPos(pos),
			slots,
		)
	}
	discard := l.ensureDiscardLocal()
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slots), Pos: pos})
	l.emit(
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     "__tetra_task_result_begin",
			ArgSlots: 1,
			RetSlots: 1,
			Pos:      pos,
		},
	)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})

	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	if valueLocal >= 0 {
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: valueLocal, Pos: pos})
	} else {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
	}
	l.emit(
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     "__tetra_task_result_slot",
			ArgSlots: 2,
			RetSlots: 1,
			Pos:      pos,
		},
	)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})

	errorSlots := slots - 2
	for slot := 0; slot < errorSlots; slot++ {
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slot + 1), Pos: pos})
		if errBase >= 0 {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: errBase + slot, Pos: pos})
		} else {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: pos})
		}
		l.emit(
			ir.IRInstr{
				Kind:     ir.IRCall,
				Name:     "__tetra_task_result_slot",
				ArgSlots: 2,
				RetSlots: 1,
				Pos:      pos,
			},
		)
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
	}
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(slots - 1), Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: status, Pos: pos})
	l.emit(
		ir.IRInstr{
			Kind:     ir.IRCall,
			Name:     "__tetra_task_result_slot",
			ArgSlots: 2,
			RetSlots: 1,
			Pos:      pos,
		},
	)
	l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: pos})
	return nil
}

func (l *lowerer) emitConvertedValueFromScratch(
	srcType, dstType string,
	base int,
	pos frontend.Position,
) (int, error) {
	srcInfo, ok := l.types[srcType]
	if !ok {
		return 0, fmt.Errorf("%s: unknown type '%s'", frontend.FormatPos(pos), srcType)
	}
	if srcType == dstType || (isThrowIntLike(srcType) && isThrowIntLike(dstType)) {
		for slot := 0; slot < srcInfo.SlotCount; slot++ {
			l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: base + slot, Pos: pos})
		}
		return srcInfo.SlotCount, nil
	}
	dstInfo, ok := l.types[dstType]
	if ok && dstInfo.Kind == semantics.TypeOptional {
		slots, err := l.emitConvertedValueFromScratch(srcType, dstInfo.ElemType, base, pos)
		if err == nil {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: pos})
			return slots + 1, nil
		}
	}
	return 0, fmt.Errorf("%s: try error slot mismatch", frontend.FormatPos(pos))
}

func isThrowIntLike(typeName string) bool {
	return lowertasks.IsThrowIntLike(typeName)
}

func (l *lowerer) pushLoop(continueLabel, breakLabel int) {
	l.loopStack = append(l.loopStack, loopLabels{
		continueLabel: continueLabel,
		breakLabel:    breakLabel,
		cleanupDepth:  len(l.cleanupIslands),
		deferDepth:    len(l.deferFrames),
	})
}

func (l *lowerer) popLoop() {
	l.loopStack = l.loopStack[:len(l.loopStack)-1]
}

func (l *lowerer) currentLoop() (loopLabels, bool) {
	if len(l.loopStack) == 0 {
		return loopLabels{}, false
	}
	return l.loopStack[len(l.loopStack)-1], true
}

// ---- verify.go ----

// IR verifier invariants:
//   - program main metadata names an existing function and function names are unique;
//   - function slot metadata is non-negative and parameters fit inside locals;
//   - branch labels are non-negative and every branch target names a label in
//     the same function;
//   - all control-flow paths entering an instruction agree on stack height;
//   - each instruction has enough input stack slots and leaves a non-negative stack;
//   - local loads/stores reference slots inside IRFunc.LocalSlots;
//   - global loads/stores reference non-negative lowered data slots;
//   - returns see exactly IRFunc.ReturnSlots values on the stack;
//   - calls declare non-negative argument and return slot counts and match
//     known in-program function signatures and known runtime ABI signatures;
//   - policy-protected functions carry the expected budget/consent guard
//     shape before backend codegen.
//
// The verifier is intentionally target-neutral: semantic type safety has already
// been checked, and new IRInstrKind values must update stackEffect here before
// they can reach backend codegen.
func VerifyProgram(prog *ir.IRProgram) error {
	if prog == nil {
		return irVerifierError("ir verifier: missing program")
	}
	if len(prog.Funcs) > 0 {
		if prog.MainIndex < 0 || prog.MainIndex >= len(prog.Funcs) {
			return irVerifierError(
				"ir verifier: main index %d out of bounds (funcs=%d)",
				prog.MainIndex,
				len(prog.Funcs),
			)
		}
		if prog.MainName == "" {
			return irVerifierError("ir verifier: missing main name")
		}
		if got := prog.Funcs[prog.MainIndex].Name; got != prog.MainName {
			return irVerifierError(
				"ir verifier: main metadata mismatch: index %d names %q, want %q",
				prog.MainIndex,
				got,
				prog.MainName,
			)
		}
	}
	funcSigs := make(map[string]ir.IRFunc, len(prog.Funcs))
	for _, fn := range prog.Funcs {
		if fn.Name == "" {
			return irVerifierError("ir verifier: function with empty name")
		}
		if _, exists := funcSigs[fn.Name]; exists {
			return irVerifierError("ir verifier: duplicate function name %q", fn.Name)
		}
		funcSigs[fn.Name] = fn
	}
	for _, fn := range prog.Funcs {
		if err := VerifyFunc(fn); err != nil {
			return err
		}
		if err := verifyKnownCallSignatures(fn, funcSigs); err != nil {
			return err
		}
	}
	return nil
}

func VerifyFunc(fn ir.IRFunc) error {
	if fn.ParamSlots < 0 || fn.LocalSlots < 0 || fn.ReturnSlots < 0 {
		return irVerifierError(
			"ir verifier: %s has negative slot metadata params=%d locals=%d returns=%d",
			fn.Name,
			fn.ParamSlots,
			fn.LocalSlots,
			fn.ReturnSlots,
		)
	}
	if fn.ParamSlots > fn.LocalSlots {
		return irVerifierError(
			"ir verifier: %s param slots %d exceed locals %d",
			fn.Name,
			fn.ParamSlots,
			fn.LocalSlots,
		)
	}
	labels := make(map[int]int)
	for i, instr := range fn.Instrs {
		if instr.Kind != ir.IRLabel {
			continue
		}
		if instr.Label < 0 {
			return verifyError(fn, i, "negative label %d", instr.Label)
		}
		if _, exists := labels[instr.Label]; exists {
			return verifyError(fn, i, "duplicate label %d", instr.Label)
		}
		labels[instr.Label] = i
	}
	for i, instr := range fn.Instrs {
		if _, _, known := stackEffect(instr); !known {
			return verifyError(fn, i, "unknown instruction kind %d", instr.Kind)
		}
		switch instr.Kind {
		case ir.IRJmp, ir.IRJmpIfZero:
			if instr.Label < 0 {
				return verifyError(fn, i, "negative label %d", instr.Label)
			}
			if _, ok := labels[instr.Label]; !ok {
				return verifyError(fn, i, "unknown label %d", instr.Label)
			}
		case ir.IRLoadLocal, ir.IRStoreLocal:
			if instr.Local < 0 || instr.Local >= fn.LocalSlots {
				return verifyError(
					fn,
					i,
					"local slot %d out of bounds (locals=%d)",
					instr.Local,
					fn.LocalSlots,
				)
			}
		case ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32:
			if instr.Name == "" {
				return verifyError(fn, i, "stack slice is missing allocation name")
			}
			if instr.ArgSlots < 0 {
				return verifyError(
					fn,
					i,
					"stack slice %q has negative backing slots %d",
					instr.Name,
					instr.ArgSlots,
				)
			}
			if instr.Imm < 0 {
				return verifyError(
					fn,
					i,
					"stack slice %q has negative logical length %d",
					instr.Name,
					instr.Imm,
				)
			}
			if instr.ArgSlots == 0 {
				if instr.Local != -1 {
					return verifyError(fn, i, "empty stack slice %q must use local -1", instr.Name)
				}
				break
			}
			if instr.Local < 0 || instr.Local+instr.ArgSlots > fn.LocalSlots {
				return verifyError(
					fn,
					i,
					"stack slice %q backing slots [%d,%d) out of bounds (locals=%d)",
					instr.Name,
					instr.Local,
					instr.Local+instr.ArgSlots,
					fn.LocalSlots,
				)
			}
		case ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32:
			if instr.Name == "" {
				return verifyError(fn, i, "region slice is missing allocation name")
			}
		case ir.IRRawSliceFromParts:
			if instr.Imm < 0 || instr.Imm > 2 {
				return verifyError(
					fn,
					i,
					"raw slice element-size shift %d out of supported range [0,2]",
					instr.Imm,
				)
			}
		case ir.IRLoadGlobal, ir.IRStoreGlobal:
			if instr.Local < 0 {
				return verifyError(fn, i, "global slot %d out of bounds", instr.Local)
			}
		case ir.IRCall:
			if instr.Name == "" {
				return verifyError(fn, i, "call is missing target name")
			}
			if instr.ArgSlots < 0 || instr.RetSlots < 0 {
				return verifyError(
					fn,
					i,
					"call %q has negative ABI slots args=%d rets=%d",
					instr.Name,
					instr.ArgSlots,
					instr.RetSlots,
				)
			}
			if sig, ok := runtimeabi.SignatureForSymbol(instr.Name); ok &&
				(instr.ArgSlots != sig.ParamSlots || instr.RetSlots != sig.ReturnSlots) {
				return verifyError(
					fn,
					i,
					"runtime call %q ABI mismatch args=%d rets=%d want args=%d rets=%d",
					instr.Name,
					instr.ArgSlots,
					instr.RetSlots,
					sig.ParamSlots,
					sig.ReturnSlots,
				)
			}
		case ir.IRSymAddr:
			if instr.Name == "" {
				return verifyError(fn, i, "symbol address is missing name")
			}
		case ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
			if instr.ProofID == "" {
				return verifyError(fn, i, "unchecked index load is missing proof id")
			}
		}
	}

	if err := verifyPolicyGuardMetadata(fn, labels); err != nil {
		return err
	}

	if len(fn.Instrs) == 0 {
		if fn.ReturnSlots != 0 {
			return irVerifierError(
				"ir verifier: %s empty body cannot produce %d return slots",
				fn.Name,
				fn.ReturnSlots,
			)
		}
		return nil
	}

	heights := make([]int, len(fn.Instrs))
	seen := make([]bool, len(fn.Instrs))
	work := []stackState{{idx: 0, height: 0}}
	for len(work) > 0 {
		cur := work[len(work)-1]
		work = work[:len(work)-1]
		if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
			if cur.height != 0 {
				return irVerifierError(
					"ir verifier: %s falls off end with stack height %d",
					fn.Name,
					cur.height,
				)
			}
			continue
		}
		if seen[cur.idx] {
			if heights[cur.idx] != cur.height {
				return verifyError(
					fn,
					cur.idx,
					"inconsistent stack height: got %d, previously %d",
					cur.height,
					heights[cur.idx],
				)
			}
			continue
		}
		seen[cur.idx] = true
		heights[cur.idx] = cur.height

		instr := fn.Instrs[cur.idx]
		pop, push, known := stackEffect(instr)
		if !known {
			return verifyError(fn, cur.idx, "unknown instruction kind %d", instr.Kind)
		}
		if cur.height < pop {
			return verifyError(
				fn,
				cur.idx,
				"stack underflow: need %d slots, have %d",
				pop,
				cur.height,
			)
		}
		nextHeight := cur.height - pop + push
		if nextHeight < 0 {
			return verifyError(fn, cur.idx, "negative stack height %d", nextHeight)
		}

		switch instr.Kind {
		case ir.IRReturn:
			if cur.height != fn.ReturnSlots {
				return verifyError(
					fn,
					cur.idx,
					"return expects %d stack slots, have %d",
					fn.ReturnSlots,
					cur.height,
				)
			}
		case ir.IRJmp:
			work = append(work, stackState{idx: labels[instr.Label], height: nextHeight})
		case ir.IRJmpIfZero:
			work = append(work, stackState{idx: labels[instr.Label], height: nextHeight})
			work = append(work, stackState{idx: cur.idx + 1, height: nextHeight})
		default:
			work = append(work, stackState{idx: cur.idx + 1, height: nextHeight})
		}
	}

	if err := verifyPolicyGuardShape(fn, labels, heights, seen); err != nil {
		return err
	}

	if err := verifyLinearEmitterStack(fn); err != nil {
		return err
	}

	return nil
}

type stackState struct {
	idx    int
	height int
}

func verifyError(fn ir.IRFunc, idx int, format string, args ...interface{}) error {
	pos := fn.Instrs[idx].Pos
	fullArgs := append([]interface{}{fn.Name, idx}, args...)
	return irVerifierErrorAt(pos, "ir verifier: %s instr %d: "+format, fullArgs...)
}

func verifyKnownCallSignatures(fn ir.IRFunc, funcSigs map[string]ir.IRFunc) error {
	for i, instr := range fn.Instrs {
		if instr.Kind != ir.IRCall {
			continue
		}
		target, ok := funcSigs[instr.Name]
		if !ok {
			continue
		}
		if instr.ArgSlots != target.ParamSlots || instr.RetSlots != target.ReturnSlots {
			return verifyError(
				fn,
				i,
				"call %q ABI mismatch args=%d rets=%d want args=%d rets=%d",
				instr.Name,
				instr.ArgSlots,
				instr.RetSlots,
				target.ParamSlots,
				target.ReturnSlots,
			)
		}
	}
	return nil
}

func verifyLinearEmitterStack(fn ir.IRFunc) error {
	height := 0
	for i, instr := range fn.Instrs {
		pop, push, known := stackEffect(instr)
		if !known {
			return verifyError(fn, i, "unknown instruction kind %d", instr.Kind)
		}
		if instr.Kind == ir.IRReturn {
			if height < fn.ReturnSlots {
				return verifyError(
					fn,
					i,
					"linear return underflow: need %d stack slots, have %d",
					fn.ReturnSlots,
					height,
				)
			}
			height = 0
			continue
		}
		if height < pop {
			return verifyError(fn, i, "linear stack underflow: need %d slots, have %d", pop, height)
		}
		height = height - pop + push
		if height < 0 {
			return verifyError(fn, i, "linear negative stack height %d", height)
		}
	}
	return nil
}

func verifyPolicyGuardMetadata(fn ir.IRFunc, labels map[int]int) error {
	policy := fn.Policy
	if !policy.HasBudget && !policy.HasConsent {
		return nil
	}
	if policy.FailLabel < 0 {
		return irVerifierError("ir verifier: %s policy guard missing failure label", fn.Name)
	}
	if _, ok := labels[policy.FailLabel]; !ok {
		return irVerifierError(
			"ir verifier: %s policy guard failure label %d is not defined",
			fn.Name,
			policy.FailLabel,
		)
	}
	if policy.HasBudget {
		if policy.BudgetLocal < 0 || policy.BudgetLocal >= fn.LocalSlots {
			return irVerifierError(
				"ir verifier: %s policy budget local %d out of bounds (locals=%d)",
				fn.Name,
				policy.BudgetLocal,
				fn.LocalSlots,
			)
		}
	}
	if policy.HasConsent {
		if policy.ConsentLocal < 0 || policy.ConsentLocal >= fn.LocalSlots {
			return irVerifierError(
				"ir verifier: %s policy consent local %d out of bounds (locals=%d)",
				fn.Name,
				policy.ConsentLocal,
				fn.LocalSlots,
			)
		}
		if policy.ConsentLocal >= fn.ParamSlots {
			return irVerifierError(
				"ir verifier: %s policy consent local %d is not a parameter slot (params=%d)",
				fn.Name,
				policy.ConsentLocal,
				fn.ParamSlots,
			)
		}
	}
	return nil
}

func verifyPolicyGuardShape(fn ir.IRFunc, labels map[int]int, heights []int, seen []bool) error {
	policy := fn.Policy
	if !policy.HasBudget && !policy.HasConsent {
		return nil
	}

	next := 0
	if policy.HasBudget {
		if !matchesBudgetInitializerAt(fn, next, policy) {
			return policyShapeError(fn, next, "malformed budget initializer")
		}
		next += 2
	}
	if policy.HasConsent {
		if !matchesConsentGuardAt(fn, next, policy) {
			return policyShapeError(fn, next, "malformed consent guard")
		}
		next += 4
	}

	if !policy.HasBudget {
		return nil
	}
	failIdx := labels[policy.FailLabel]
	for i, instr := range fn.Instrs {
		if i >= failIdx {
			break
		}
		cost, ok := budgetChargeForInstr(instr.Kind)
		if !ok {
			continue
		}
		if matchesBudgetGuardBefore(fn, i, 0, policy, cost) {
			continue
		}
		if seen[i] && heights[i] > 0 && matchesBudgetGuardBefore(fn, i, heights[i], policy, cost) {
			continue
		}
		return verifyError(fn, i, "missing budget guard before charged instruction")
	}
	return nil
}

func policyShapeError(fn ir.IRFunc, idx int, message string) error {
	if idx >= 0 && idx < len(fn.Instrs) {
		return verifyError(fn, idx, message)
	}
	return irVerifierError("ir verifier: %s %s", fn.Name, message)
}

func matchesBudgetInitializerAt(fn ir.IRFunc, idx int, policy ir.IRPolicy) bool {
	return idx+1 < len(fn.Instrs) &&
		fn.Instrs[idx].Kind == ir.IRConstI32 &&
		fn.Instrs[idx].Imm == policy.Budget &&
		fn.Instrs[idx+1].Kind == ir.IRStoreLocal &&
		fn.Instrs[idx+1].Local == policy.BudgetLocal
}

func matchesConsentGuardAt(fn ir.IRFunc, idx int, policy ir.IRPolicy) bool {
	return idx+3 < len(fn.Instrs) &&
		fn.Instrs[idx].Kind == ir.IRLoadLocal &&
		fn.Instrs[idx].Local == policy.ConsentLocal &&
		fn.Instrs[idx+1].Kind == ir.IRConstI32 &&
		fn.Instrs[idx+1].Imm == consentTokenRuntimeSentinel &&
		fn.Instrs[idx+2].Kind == ir.IRCmpEqI32 &&
		fn.Instrs[idx+3].Kind == ir.IRJmpIfZero &&
		fn.Instrs[idx+3].Label == policy.FailLabel
}

func matchesBudgetGuardBefore(
	fn ir.IRFunc,
	chargedIdx int,
	preservedDepth int,
	policy ir.IRPolicy,
	cost int32,
) bool {
	loadStart := chargedIdx
	if preservedDepth > 0 {
		loadStart = chargedIdx - preservedDepth
		if loadStart < 0 {
			return false
		}
		base := fn.Instrs[loadStart].Local
		for i := 0; i < preservedDepth; i++ {
			instr := fn.Instrs[loadStart+i]
			if instr.Kind != ir.IRLoadLocal || instr.Local != base+i {
				return false
			}
		}
		storeStart := loadStart - budgetGuardInstrs - preservedDepth
		if storeStart < 0 {
			return false
		}
		for i := 0; i < preservedDepth; i++ {
			instr := fn.Instrs[storeStart+i]
			if instr.Kind != ir.IRStoreLocal || instr.Local != base+preservedDepth-1-i {
				return false
			}
		}
	}
	guardStart := loadStart - budgetGuardInstrs
	return matchesBudgetGuardAt(fn, guardStart, policy, cost)
}

const budgetGuardInstrs = 8

func matchesBudgetGuardAt(fn ir.IRFunc, idx int, policy ir.IRPolicy, cost int32) bool {
	return idx >= 0 &&
		idx+budgetGuardInstrs-1 < len(fn.Instrs) &&
		fn.Instrs[idx].Kind == ir.IRLoadLocal &&
		fn.Instrs[idx].Local == policy.BudgetLocal &&
		fn.Instrs[idx+1].Kind == ir.IRConstI32 &&
		fn.Instrs[idx+1].Imm == cost &&
		fn.Instrs[idx+2].Kind == ir.IRSubI32 &&
		fn.Instrs[idx+3].Kind == ir.IRStoreLocal &&
		fn.Instrs[idx+3].Local == policy.BudgetLocal &&
		fn.Instrs[idx+4].Kind == ir.IRLoadLocal &&
		fn.Instrs[idx+4].Local == policy.BudgetLocal &&
		fn.Instrs[idx+5].Kind == ir.IRConstI32 &&
		fn.Instrs[idx+5].Imm == 0 &&
		fn.Instrs[idx+6].Kind == ir.IRCmpGeI32 &&
		fn.Instrs[idx+7].Kind == ir.IRJmpIfZero &&
		fn.Instrs[idx+7].Label == policy.FailLabel
}

func stackEffect(instr ir.IRInstr) (pop int, push int, known bool) {
	switch instr.Kind {
	case ir.IRWrite:
		return 2, 0, true
	case ir.IRStrLit:
		return 0, 2, true
	case ir.IRConstI32, ir.IRLoadLocal, ir.IRLoadGlobal:
		return 0, 1, true
	case ir.IRStoreLocal, ir.IRStoreGlobal:
		return 1, 0, true
	case ir.IRAddI32, ir.IRSubI32, ir.IRCmpEqI32, ir.IRCmpLtI32,
		ir.IRMulI32, ir.IRDivI32, ir.IRModI32, ir.IRCmpGtI32,
		ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
		return 2, 1, true
	case ir.IRNegI32:
		return 1, 1, true
	case ir.IRCall:
		return instr.ArgSlots, instr.RetSlots, true
	case ir.IRLabel, ir.IRJmp:
		return 0, 0, true
	case ir.IRJmpIfZero:
		return 1, 0, true
	case ir.IRReturn:
		return 0, 0, true
	case ir.IRAllocBytes, ir.IRIslandNew:
		return 1, 1, true
	case ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32,
		ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32,
		ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32:
		return 1, 2, true
	case ir.IRRegionEnter, ir.IRRegionReset:
		return 0, 0, true
	case ir.IRRawSliceFromParts:
		return 3, 2, true
	case ir.IRSliceWindow:
		return 4, 2, true
	case ir.IRSlicePrefix, ir.IRSliceSuffix:
		return 3, 2, true
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
		ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked:
		return 3, 1, true
	case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		return 4, 0, true
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		return 2, 2, true
	case ir.IRIslandReset:
		return 1, 1, true
	case ir.IRIslandFree:
		return 1, 0, true
	case ir.IRCapIO, ir.IRCapMem, ir.IRSymAddr:
		return 0, 1, true
	case ir.IRMemReadI32, ir.IRMemReadU8, ir.IRMemReadPtr, ir.IRMmioReadI32,
		ir.IRAtomicLoadPtr, ir.IRAtomicLoadI32, ir.IRAtomicLoadI64,
		ir.IRAtomicLoadI8, ir.IRAtomicLoadI16:
		return 2, 1, true
	case ir.IRMemWriteI32, ir.IRMemWriteU8, ir.IRMemWritePtr, ir.IRMemWriteArchPtr, ir.IRPtrAdd,
		ir.IRMmioWriteI32, ir.IRCtxSwitch, ir.IRAtomicStorePtr,
		ir.IRAtomicExchangePtr, ir.IRAtomicFetchAddPtr, ir.IRAtomicFetchSubPtr,
		ir.IRAtomicFetchAndPtr, ir.IRAtomicFetchOrPtr, ir.IRAtomicFetchXorPtr,
		ir.IRAtomicStoreI32, ir.IRAtomicExchangeI32, ir.IRAtomicFetchAddI32,
		ir.IRAtomicFetchSubI32, ir.IRAtomicFetchAndI32, ir.IRAtomicFetchOrI32,
		ir.IRAtomicFetchXorI32, ir.IRAtomicStoreI64, ir.IRAtomicExchangeI64,
		ir.IRAtomicFetchAddI64, ir.IRAtomicFetchSubI64, ir.IRAtomicFetchAndI64,
		ir.IRAtomicFetchOrI64, ir.IRAtomicFetchXorI64, ir.IRAtomicStoreI8,
		ir.IRAtomicExchangeI8, ir.IRAtomicFetchAddI8, ir.IRAtomicFetchSubI8,
		ir.IRAtomicFetchAndI8, ir.IRAtomicFetchOrI8, ir.IRAtomicFetchXorI8,
		ir.IRAtomicStoreI16, ir.IRAtomicExchangeI16, ir.IRAtomicFetchAddI16,
		ir.IRAtomicFetchSubI16, ir.IRAtomicFetchAndI16, ir.IRAtomicFetchOrI16,
		ir.IRAtomicFetchXorI16:
		return 3, 1, true
	case ir.IRAtomicCompareExchangePtr,
		ir.IRAtomicCompareExchangeI32,
		ir.IRAtomicCompareExchangeI64,
		ir.IRAtomicCompareExchangeI8,
		ir.IRAtomicCompareExchangeI16:
		return 4, 1, true
	case ir.IRMemReadI32Offset, ir.IRMemReadU8Offset, ir.IRMemReadPtrOffset:
		return 3, 1, true
	case ir.IRMemWriteI32Offset,
		ir.IRMemWriteU8Offset,
		ir.IRMemWritePtrOffset,
		ir.IRMemWriteArchPtrOffset:
		return 4, 1, true
	case ir.IRAtomicFenceSeqCst, ir.IRAtomicFenceRelaxed, ir.IRAtomicFenceAcquire,
		ir.IRAtomicFenceRelease, ir.IRAtomicFenceAcqRel:
		return 0, 0, true
	default:
		return 0, 0, false
	}
}
