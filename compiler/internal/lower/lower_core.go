package lower

import (
	"fmt"
	"sort"
	"strings"
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
	{kind: ir.IRDropOwned, cost: 1},
	{kind: ir.IRReleaseAllocation, cost: 1},
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
	ownedReturnSummaries := collectOwnedReturnSummaries(checked, opt)
	ownedThrowSummaries := collectOwnedThrowSummaries(checked, opt, ownedReturnSummaries)

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
			ownedReturnSummaries,
			ownedThrowSummaries,
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
	ownedReturnSummaries := collectOwnedReturnSummaries(checked, opt)
	ownedThrowSummaries := collectOwnedThrowSummaries(checked, opt, ownedReturnSummaries)
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
			ownedReturnSummaries,
			ownedThrowSummaries,
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

func collectOwnedReturnSummaries(
	checked *semantics.CheckedProgram,
	opt Options,
) map[string]ownedReturnSummary {
	if checked == nil || !opt.OwnedAllocDropLowering {
		return nil
	}
	out := map[string]ownedReturnSummary{}
	changed := true
	for changed {
		changed = false
		knownThrows := collectOwnedThrowSummaries(checked, opt, out)
		for _, fn := range checked.Funcs {
			if _, exists := out[fn.Name]; exists {
				continue
			}
			summary, ok := ownedReturnSummaryForCheckedFunc(fn, checked.FuncSigs, checked.Types, out, knownThrows)
			if !ok {
				continue
			}
			out[fn.Name] = summary
			changed = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ownedReturnSummaryForCheckedFunc(
	fn semantics.CheckedFunc,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	known map[string]ownedReturnSummary,
	knownThrows map[string]ownedThrowSummary,
) (ownedReturnSummary, bool) {
	returnSlot, ok := ownedReturnStorageSlot(fn.ReturnType, types)
	if fn.Decl == nil ||
		!ok ||
		fn.ReturnOwnership == "borrow" ||
		fn.ReturnSlots <= returnSlot ||
		fn.Async {
		return ownedReturnSummary{}, false
	}
	allocLocals := map[string]int{}
	conditionalAllocLocals := map[string]ownedReturnCondition{}
	nonOwnedEnumTagLocals := map[string]int32{}
	returnedSource := ""
	returnedSlot := -1
	returnCount := 0
	conditionalReturn := ownedReturnCondition{}
	valid := true
	collectDirectOwnedReturnInfo(
		fn.Decl.Body,
		fn,
		funcs,
		types,
		known,
		knownThrows,
		allocLocals,
		conditionalAllocLocals,
		nonOwnedEnumTagLocals,
		&returnedSource,
		&returnedSlot,
		&returnCount,
		&conditionalReturn,
		&valid,
	)
	if !valid || returnCount == 0 || returnedSource == "" || returnedSlot != returnSlot {
		return ownedReturnSummary{}, false
	}
	conditionalTagSlot := -1
	if conditionalReturn.conditional {
		var guardOK bool
		conditionalTagSlot, guardOK = conditionalOwnedReturnGuardSlot(fn.ReturnType, returnedSlot, types)
		if !guardOK {
			return ownedReturnSummary{}, false
		}
	}
	return ownedReturnSummary{
		returnSlot:                  returnedSlot,
		conditional:                 conditionalReturn.conditional,
		conditionalTagSlot:          conditionalTagSlot,
		hasConditionalTagExactValue: conditionalReturn.hasExactTagValue,
		conditionalTagExactValue:    conditionalReturn.exactTagValue,
		layoutID:                    fmt.Sprintf("layout:return:%s:%s", fn.Name, returnedSource),
		domain:                      ir.IROwnershipDomainHeap,
		releaseKind:                 ir.IRReleaseKindLinuxMmap,
	}, true
}

func collectOwnedThrowSummaries(
	checked *semantics.CheckedProgram,
	opt Options,
	ownedReturnSummaries map[string]ownedReturnSummary,
) map[string]ownedThrowSummary {
	if checked == nil || !opt.OwnedAllocDropLowering {
		return nil
	}
	out := map[string]ownedThrowSummary{}
	for _, fn := range checked.Funcs {
		summary, ok := ownedThrowSummaryForCheckedFunc(
			fn,
			checked.FuncSigs,
			checked.Types,
			ownedReturnSummaries,
		)
		if !ok {
			continue
		}
		out[fn.Name] = summary
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ownedThrowSummaryForCheckedFunc(
	fn semantics.CheckedFunc,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	knownReturns map[string]ownedReturnSummary,
) (ownedThrowSummary, bool) {
	throwSlot, ok := ownedThrowStorageSlot(fn.ThrowsType, types)
	if fn.Decl == nil ||
		fn.ThrowsType == "" ||
		!ok ||
		fn.Async {
		return ownedThrowSummary{}, false
	}
	allocLocals := map[string]int{}
	thrownSource := ""
	thrownSlot := -1
	thrownMultiSource := false
	throwCount := 0
	valid := true
	collectDirectOwnedThrowInfo(
		fn.Decl.Body,
		fn,
		funcs,
		types,
		knownReturns,
		allocLocals,
		&thrownSource,
		&thrownSlot,
		&thrownMultiSource,
		&throwCount,
		&valid,
	)
	if !valid || throwCount == 0 || thrownSource == "" || thrownSlot != throwSlot {
		return ownedThrowSummary{}, false
	}
	layoutSource := thrownSource
	if thrownMultiSource {
		layoutSource = fmt.Sprintf("slot%d", thrownSlot)
	}
	return ownedThrowSummary{
		errorSlot:    thrownSlot,
		alwaysThrows: stmtListAlwaysThrows(fn.Decl.Body),
		layoutID:     fmt.Sprintf("layout:throw:%s:%s", fn.Name, layoutSource),
		domain:       ir.IROwnershipDomainHeap,
		releaseKind:  ir.IRReleaseKindLinuxMmap,
	}, true
}

func collectDirectOwnedReturnInfo(
	stmts []frontend.Stmt,
	fn semantics.CheckedFunc,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	known map[string]ownedReturnSummary,
	knownThrows map[string]ownedThrowSummary,
	allocLocals map[string]int,
	conditionalAllocLocals map[string]ownedReturnCondition,
	nonOwnedEnumTagLocals map[string]int32,
	returnedSource *string,
	returnedSlot *int,
	returnCount *int,
	conditionalReturn *ownedReturnCondition,
	valid *bool,
) {
	if !*valid {
		return
	}
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			info, ok := fn.Locals[s.Name]
			if _, storageOK := ownedReturnStorageSlot(info.TypeName, types); ok && storageOK {
				if nonOwnedTag, tagOK := enumConstructorNonOwnedTag(s.Value, info.TypeName, types); tagOK {
					deleteOwnedReturnSourcesForBase(allocLocals, s.Name)
					deleteOwnedReturnConditionalSourcesForBase(conditionalAllocLocals, s.Name)
					nonOwnedEnumTagLocals[s.Name] = nonOwnedTag
					continue
				}
				if info.TypeName == "ptr" && isDirectAllocBytesCall(s.Value) {
					allocLocals[s.Name] = 0
					deleteOwnedReturnConditionalSourcesForBase(conditionalAllocLocals, s.Name)
					delete(nonOwnedEnumTagLocals, s.Name)
					continue
				}
				if source, sourceSlot, sourceCondition, ok := conditionalOwnedReturnCallSource(
					s.Value,
					funcs,
					known,
				); ok {
					allocLocals[s.Name] = sourceSlot
					setOwnedReturnSourceCondition(conditionalAllocLocals, s.Name, sourceCondition)
					delete(nonOwnedEnumTagLocals, s.Name)
					forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, s.Name)
					continue
				}
				if source, sourceSlot, ok := directOwnedReturnSource(s.Value, funcs, known, allocLocals); ok {
					sourceCondition := ownedReturnSourceCondition(conditionalAllocLocals, source)
					allocLocals[s.Name] = sourceSlot
					setOwnedReturnSourceCondition(conditionalAllocLocals, s.Name, sourceCondition)
					delete(nonOwnedEnumTagLocals, s.Name)
					forgetMovedOwnedReturnSource(allocLocals, source, s.Name)
					forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, s.Name)
					continue
				}
				if source, destSlot, ok := enumConstructorOwnedReturnSource(
					s.Value,
					info.TypeName,
					funcs,
					types,
					known,
					allocLocals,
				); ok {
					sourceCondition := ownedReturnSourceCondition(conditionalAllocLocals, source)
					allocLocals[s.Name] = destSlot
					setOwnedReturnSourceCondition(conditionalAllocLocals, s.Name, sourceCondition)
					delete(nonOwnedEnumTagLocals, s.Name)
					forgetMovedOwnedReturnSource(allocLocals, source, s.Name)
					forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, s.Name)
					continue
				}
				if source, destSlot, sourceCondition, ok := matchExprOwnedReturnSourceInfo(
					s.Value,
					info.TypeName,
					funcs,
					types,
					known,
					allocLocals,
				); ok {
					sourceCondition = mergeOwnedReturnConditions(
						sourceCondition,
						ownedReturnSourceCondition(conditionalAllocLocals, source),
					)
					allocLocals[s.Name] = destSlot
					setOwnedReturnSourceCondition(conditionalAllocLocals, s.Name, sourceCondition)
					delete(nonOwnedEnumTagLocals, s.Name)
					forgetMovedOwnedReturnSource(allocLocals, source, s.Name)
					forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, s.Name)
					continue
				}
				if source, destSlot, ok := catchExprOwnedReturnSource(
					s.Value,
					info.TypeName,
					funcs,
					types,
					known,
					knownThrows,
					allocLocals,
				); ok {
					allocLocals[s.Name] = destSlot
					deleteOwnedReturnConditionalSourcesForBase(conditionalAllocLocals, s.Name)
					delete(nonOwnedEnumTagLocals, s.Name)
					forgetMovedOwnedReturnSource(allocLocals, source, s.Name)
					forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, s.Name)
					continue
				}
				if source, destSlot, sourceCondition, ok := catchExprNonOwnedErrorOwnedResultSourceInfo(
					s.Value,
					info.TypeName,
					funcs,
					types,
					known,
				); ok {
					allocLocals[s.Name] = destSlot
					setOwnedReturnSourceCondition(conditionalAllocLocals, s.Name, sourceCondition)
					delete(nonOwnedEnumTagLocals, s.Name)
					forgetMovedOwnedReturnSource(allocLocals, source, s.Name)
					forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, s.Name)
					continue
				}
				if source, destSlot, sourceCondition, ok := catchExprOwnedErrorMixedResultSourceInfo(
					s.Value,
					info.TypeName,
					funcs,
					types,
					known,
					knownThrows,
				); ok {
					allocLocals[s.Name] = destSlot
					setOwnedReturnSourceCondition(conditionalAllocLocals, s.Name, sourceCondition)
					delete(nonOwnedEnumTagLocals, s.Name)
					forgetMovedOwnedReturnSource(allocLocals, source, s.Name)
					forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, s.Name)
					continue
				}
				if source, destSlot, fieldName, fieldSlot, ok := structLiteralOwnedReturnSource(
					s.Value,
					info.TypeName,
					funcs,
					types,
					known,
					allocLocals,
				); ok {
					allocLocals[s.Name] = destSlot
					deleteOwnedReturnConditionalSourcesForBase(conditionalAllocLocals, s.Name)
					delete(nonOwnedEnumTagLocals, s.Name)
					if fieldName != "" {
						allocLocals[fieldOwnedReturnSourceName(s.Name, []string{fieldName})] = fieldSlot
					}
					forgetMovedOwnedReturnSource(allocLocals, source, s.Name)
					forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, s.Name)
				}
			}
		case *frontend.AssignStmt:
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				info, ok := fn.Locals[id.Name]
				if !ok || !ownedReturnStorageType(info.TypeName, types) {
					continue
				}
				if s.Op == 0 {
					if nonOwnedTag, tagOK := enumConstructorNonOwnedTag(s.Value, info.TypeName, types); tagOK {
						deleteOwnedReturnSourcesForBase(allocLocals, id.Name)
						deleteOwnedReturnConditionalSourcesForBase(conditionalAllocLocals, id.Name)
						nonOwnedEnumTagLocals[id.Name] = nonOwnedTag
						continue
					}
					if info.TypeName == "ptr" && isDirectAllocBytesCall(s.Value) {
						allocLocals[id.Name] = 0
						deleteOwnedReturnConditionalSourcesForBase(conditionalAllocLocals, id.Name)
						delete(nonOwnedEnumTagLocals, id.Name)
						continue
					}
					if source, sourceSlot, sourceCondition, ok := conditionalOwnedReturnCallSource(
						s.Value,
						funcs,
						known,
					); ok {
						allocLocals[id.Name] = sourceSlot
						setOwnedReturnSourceCondition(conditionalAllocLocals, id.Name, sourceCondition)
						delete(nonOwnedEnumTagLocals, id.Name)
						forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, id.Name)
						continue
					}
					if source, sourceSlot, ok := directOwnedReturnSource(s.Value, funcs, known, allocLocals); ok {
						sourceCondition := ownedReturnSourceCondition(conditionalAllocLocals, source)
						allocLocals[id.Name] = sourceSlot
						setOwnedReturnSourceCondition(conditionalAllocLocals, id.Name, sourceCondition)
						delete(nonOwnedEnumTagLocals, id.Name)
						forgetMovedOwnedReturnSource(allocLocals, source, id.Name)
						forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, id.Name)
						continue
					}
					if source, destSlot, ok := enumConstructorOwnedReturnSource(
						s.Value,
						info.TypeName,
						funcs,
						types,
						known,
						allocLocals,
					); ok {
						sourceCondition := ownedReturnSourceCondition(conditionalAllocLocals, source)
						allocLocals[id.Name] = destSlot
						setOwnedReturnSourceCondition(conditionalAllocLocals, id.Name, sourceCondition)
						delete(nonOwnedEnumTagLocals, id.Name)
						forgetMovedOwnedReturnSource(allocLocals, source, id.Name)
						forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, id.Name)
						continue
					}
					if source, destSlot, sourceCondition, ok := matchExprOwnedReturnSourceInfo(
						s.Value,
						info.TypeName,
						funcs,
						types,
						known,
						allocLocals,
					); ok {
						sourceCondition = mergeOwnedReturnConditions(
							sourceCondition,
							ownedReturnSourceCondition(conditionalAllocLocals, source),
						)
						allocLocals[id.Name] = destSlot
						setOwnedReturnSourceCondition(conditionalAllocLocals, id.Name, sourceCondition)
						delete(nonOwnedEnumTagLocals, id.Name)
						forgetMovedOwnedReturnSource(allocLocals, source, id.Name)
						forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, id.Name)
						continue
					}
					if source, destSlot, ok := catchExprOwnedReturnSource(
						s.Value,
						info.TypeName,
						funcs,
						types,
						known,
						knownThrows,
						allocLocals,
					); ok {
						allocLocals[id.Name] = destSlot
						deleteOwnedReturnConditionalSourcesForBase(conditionalAllocLocals, id.Name)
						delete(nonOwnedEnumTagLocals, id.Name)
						forgetMovedOwnedReturnSource(allocLocals, source, id.Name)
						forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, id.Name)
						continue
					}
					if source, destSlot, sourceCondition, ok := catchExprNonOwnedErrorOwnedResultSourceInfo(
						s.Value,
						info.TypeName,
						funcs,
						types,
						known,
					); ok {
						allocLocals[id.Name] = destSlot
						setOwnedReturnSourceCondition(conditionalAllocLocals, id.Name, sourceCondition)
						delete(nonOwnedEnumTagLocals, id.Name)
						forgetMovedOwnedReturnSource(allocLocals, source, id.Name)
						forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, id.Name)
						continue
					}
					if source, destSlot, sourceCondition, ok := catchExprOwnedErrorMixedResultSourceInfo(
						s.Value,
						info.TypeName,
						funcs,
						types,
						known,
						knownThrows,
					); ok {
						allocLocals[id.Name] = destSlot
						setOwnedReturnSourceCondition(conditionalAllocLocals, id.Name, sourceCondition)
						delete(nonOwnedEnumTagLocals, id.Name)
						forgetMovedOwnedReturnSource(allocLocals, source, id.Name)
						forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, id.Name)
						continue
					}
					if source, destSlot, fieldName, fieldSlot, ok := structLiteralOwnedReturnSource(
						s.Value,
						info.TypeName,
						funcs,
						types,
						known,
						allocLocals,
					); ok {
						allocLocals[id.Name] = destSlot
						deleteOwnedReturnConditionalSourcesForBase(conditionalAllocLocals, id.Name)
						delete(nonOwnedEnumTagLocals, id.Name)
						if fieldName != "" {
							allocLocals[fieldOwnedReturnSourceName(id.Name, []string{fieldName})] = fieldSlot
						}
						forgetMovedOwnedReturnSource(allocLocals, source, id.Name)
						forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, id.Name)
						continue
					}
				}
				deleteOwnedReturnSourcesForBase(allocLocals, id.Name)
				deleteOwnedReturnConditionalSourcesForBase(conditionalAllocLocals, id.Name)
				delete(nonOwnedEnumTagLocals, id.Name)
				continue
			}

			baseName, fields, pos, ok := splitFieldPathLower(s.Target)
			if !ok || len(fields) == 0 {
				continue
			}
			info, ok := fn.Locals[baseName]
			ownedSlot, storageOK := ownedReturnStorageSlot(info.TypeName, types)
			if !ok || !storageOK {
				continue
			}
			fieldType, fieldSlots, fieldOffset, err := resolveFieldChainLower(
				info.TypeName,
				info.Base,
				fields,
				types,
				pos,
			)
			if err != nil {
				deleteOwnedReturnSourcesForBase(allocLocals, baseName)
				delete(nonOwnedEnumTagLocals, baseName)
				continue
			}
			fieldSlot := fieldOffset - info.Base
			fieldOwnedSlot, fieldStorageOK := ownedReturnStorageSlot(fieldType, types)
			if !fieldStorageOK ||
				fieldOwnedSlot < 0 ||
				fieldOwnedSlot >= fieldSlots ||
				fieldSlot+fieldOwnedSlot != ownedSlot {
				deleteOwnedReturnSourcesForBase(allocLocals, baseName)
				continue
			}
			fieldSource := fieldOwnedReturnSourceName(baseName, fields)
			if s.Op == 0 {
				if fieldType == "ptr" && isDirectAllocBytesCall(s.Value) {
					allocLocals[baseName] = ownedSlot
					allocLocals[fieldSource] = fieldOwnedSlot
					deleteOwnedReturnConditionalSourcesForBase(conditionalAllocLocals, baseName)
					delete(nonOwnedEnumTagLocals, baseName)
					continue
				}
				if source, sourceSlot, ok := directOwnedReturnSource(s.Value, funcs, known, allocLocals); ok && sourceSlot == fieldOwnedSlot {
					sourceCondition := ownedReturnSourceCondition(conditionalAllocLocals, source)
					allocLocals[baseName] = ownedSlot
					allocLocals[fieldSource] = fieldOwnedSlot
					setOwnedReturnSourceCondition(conditionalAllocLocals, baseName, sourceCondition)
					setOwnedReturnSourceCondition(conditionalAllocLocals, fieldSource, sourceCondition)
					delete(nonOwnedEnumTagLocals, baseName)
					forgetMovedOwnedReturnSource(allocLocals, source, baseName)
					forgetMovedOwnedReturnConditionalSource(conditionalAllocLocals, source, baseName)
					continue
				}
			}
			deleteOwnedReturnSourcesForBase(allocLocals, baseName)
			deleteOwnedReturnConditionalSourcesForBase(conditionalAllocLocals, baseName)
			delete(nonOwnedEnumTagLocals, baseName)
		case *frontend.ReturnStmt:
			*returnCount++
			source, sourceSlot, ok := directOwnedReturnSource(s.Value, funcs, known, allocLocals)
			if ok {
				*conditionalReturn = mergeOwnedReturnConditions(
					*conditionalReturn,
					ownedReturnSourceCondition(conditionalAllocLocals, source),
				)
			}
			if !ok {
				if isNoneExpr(s.Value) && optionalOwnedReturnTagSlot(fn.ReturnType, types) >= 0 {
					*conditionalReturn = mergeOwnedReturnConditions(
						*conditionalReturn,
						ownedReturnCondition{conditional: true},
					)
					continue
				}
				if callSource, callSlot, callCondition, callOK := conditionalOwnedReturnCallSource(
					s.Value,
					funcs,
					known,
				); callOK {
					source = callSource
					sourceSlot = callSlot
					*conditionalReturn = mergeOwnedReturnConditions(*conditionalReturn, callCondition)
					ok = true
				}
			}
			if !ok {
				if enumSource, destSlot, enumOK := enumConstructorOwnedReturnSource(
					s.Value,
					fn.ReturnType,
					funcs,
					types,
					known,
					allocLocals,
				); enumOK {
					source = enumSource
					sourceSlot = destSlot
					ok = true
				}
			}
			if !ok {
				if matchSource, destSlot, matchCondition, matchOK := matchExprOwnedReturnSourceInfo(
					s.Value,
					fn.ReturnType,
					funcs,
					types,
					known,
					allocLocals,
				); matchOK {
					source = matchSource
					sourceSlot = destSlot
					*conditionalReturn = mergeOwnedReturnConditions(
						*conditionalReturn,
						mergeOwnedReturnConditions(
							matchCondition,
							ownedReturnSourceCondition(conditionalAllocLocals, matchSource),
						),
					)
					ok = true
				}
			}
			if !ok {
				if structSource, destSlot, fieldName, _, structOK := structLiteralOwnedReturnSource(
					s.Value,
					fn.ReturnType,
					funcs,
					types,
					known,
					allocLocals,
				); structOK {
					if structSource == "" && fieldName != "" {
						structSource = fmt.Sprintf("inline:core.alloc_bytes:%s.%s", fn.ReturnType, fieldName)
					}
					if structSource == "" {
						*valid = false
						return
					}
					source = structSource
					sourceSlot = destSlot
					ok = true
				}
			}
			if !ok {
				if catchSource, catchSlot, catchOK := catchExprOwnedReturnSource(
					s.Value,
					fn.ReturnType,
					funcs,
					types,
					known,
					knownThrows,
					allocLocals,
				); catchOK {
					source = catchSource
					sourceSlot = catchSlot
					ok = true
				}
			}
			if !ok {
				if catchSource, catchSlot, catchCondition, catchOK := catchExprNonOwnedErrorOwnedResultSourceInfo(
					s.Value,
					fn.ReturnType,
					funcs,
					types,
					known,
				); catchOK {
					source = catchSource
					sourceSlot = catchSlot
					*conditionalReturn = mergeOwnedReturnConditions(*conditionalReturn, catchCondition)
					ok = true
				}
			}
			if !ok {
				if catchSource, catchSlot, catchCondition, catchOK := catchExprOwnedErrorMixedResultSourceInfo(
					s.Value,
					fn.ReturnType,
					funcs,
					types,
					known,
					knownThrows,
				); catchOK {
					source = catchSource
					sourceSlot = catchSlot
					*conditionalReturn = mergeOwnedReturnConditions(*conditionalReturn, catchCondition)
					ok = true
				}
			}
			if !ok {
				*valid = false
				return
			}
			if *returnedSource == "" {
				*returnedSource = source
				*returnedSlot = sourceSlot
			} else if *returnedSource != source {
				*valid = false
				return
			} else if *returnedSlot != sourceSlot {
				*valid = false
				return
			}
		case *frontend.IfStmt:
			thenLocals := cloneOwnedReturnSourceLocals(allocLocals)
			thenConditionalLocals := cloneOwnedReturnSourceConditionals(conditionalAllocLocals)
			thenNonOwnedTags := cloneOwnedReturnSourceNonOwnedEnumTags(nonOwnedEnumTagLocals)
			fallthroughLocals := cloneOwnedReturnSourceLocals(allocLocals)
			fallthroughConditionalLocals := cloneOwnedReturnSourceConditionals(conditionalAllocLocals)
			fallthroughNonOwnedTags := cloneOwnedReturnSourceNonOwnedEnumTags(nonOwnedEnumTagLocals)
			collectDirectOwnedReturnInfo(
				s.Then,
				fn,
				funcs,
				types,
				known,
				knownThrows,
				thenLocals,
				thenConditionalLocals,
				thenNonOwnedTags,
				returnedSource,
				returnedSlot,
				returnCount,
				conditionalReturn,
				valid,
			)
			if !*valid {
				return
			}
			if len(s.Else) > 0 {
				elseLocals := cloneOwnedReturnSourceLocals(allocLocals)
				elseConditionalLocals := cloneOwnedReturnSourceConditionals(conditionalAllocLocals)
				elseNonOwnedTags := cloneOwnedReturnSourceNonOwnedEnumTags(nonOwnedEnumTagLocals)
				collectDirectOwnedReturnInfo(
					s.Else,
					fn,
					funcs,
					types,
					known,
					knownThrows,
					elseLocals,
					elseConditionalLocals,
					elseNonOwnedTags,
					returnedSource,
					returnedSlot,
					returnCount,
					conditionalReturn,
					valid,
				)
				if !*valid {
					return
				}
				mergeOwnedReturnSourceIfBranchFacts(
					allocLocals,
					conditionalAllocLocals,
					nonOwnedEnumTagLocals,
					thenLocals,
					thenConditionalLocals,
					thenNonOwnedTags,
					elseLocals,
					elseConditionalLocals,
					elseNonOwnedTags,
					fn,
					types,
				)
			} else {
				mergeOwnedReturnSourceIfBranchFacts(
					allocLocals,
					conditionalAllocLocals,
					nonOwnedEnumTagLocals,
					thenLocals,
					thenConditionalLocals,
					thenNonOwnedTags,
					fallthroughLocals,
					fallthroughConditionalLocals,
					fallthroughNonOwnedTags,
					fn,
					types,
				)
			}
		case *frontend.IfLetStmt:
			thenLocals := cloneOwnedReturnSourceLocals(allocLocals)
			thenConditionalLocals := cloneOwnedReturnSourceConditionals(conditionalAllocLocals)
			collectDirectOwnedReturnInfo(
				s.Then,
				fn,
				funcs,
				types,
				known,
				knownThrows,
				thenLocals,
				thenConditionalLocals,
				cloneOwnedReturnSourceNonOwnedEnumTags(nonOwnedEnumTagLocals),
				returnedSource,
				returnedSlot,
				returnCount,
				conditionalReturn,
				valid,
			)
			if !*valid {
				return
			}
			if len(s.Else) > 0 {
				elseLocals := cloneOwnedReturnSourceLocals(allocLocals)
				elseConditionalLocals := cloneOwnedReturnSourceConditionals(conditionalAllocLocals)
				collectDirectOwnedReturnInfo(
					s.Else,
					fn,
					funcs,
					types,
					known,
					knownThrows,
					elseLocals,
					elseConditionalLocals,
					cloneOwnedReturnSourceNonOwnedEnumTags(nonOwnedEnumTagLocals),
					returnedSource,
					returnedSlot,
					returnCount,
					conditionalReturn,
					valid,
				)
			}
		case *frontend.WhileStmt:
			bodyLocals := cloneOwnedReturnSourceLocals(allocLocals)
			bodyConditionalLocals := cloneOwnedReturnSourceConditionals(conditionalAllocLocals)
			collectDirectOwnedReturnInfo(
				s.Body,
				fn,
				funcs,
				types,
				known,
				knownThrows,
				bodyLocals,
				bodyConditionalLocals,
				cloneOwnedReturnSourceNonOwnedEnumTags(nonOwnedEnumTagLocals),
				returnedSource,
				returnedSlot,
				returnCount,
				conditionalReturn,
				valid,
			)
		case *frontend.ForRangeStmt:
			bodyLocals := cloneOwnedReturnSourceLocals(allocLocals)
			bodyConditionalLocals := cloneOwnedReturnSourceConditionals(conditionalAllocLocals)
			collectDirectOwnedReturnInfo(
				s.Body,
				fn,
				funcs,
				types,
				known,
				knownThrows,
				bodyLocals,
				bodyConditionalLocals,
				cloneOwnedReturnSourceNonOwnedEnumTags(nonOwnedEnumTagLocals),
				returnedSource,
				returnedSlot,
				returnCount,
				conditionalReturn,
				valid,
			)
		case *frontend.MatchStmt:
			mergeCaseFacts := matchStmtOwnedReturnBranchMergeOK(s)
			mergedLocals := map[string]int(nil)
			mergedConditionals := map[string]ownedReturnCondition(nil)
			mergedNonOwnedTags := map[string]int32(nil)
			for _, c := range s.Cases {
				caseLocals := cloneOwnedReturnSourceLocals(allocLocals)
				caseConditionalLocals := cloneOwnedReturnSourceConditionals(conditionalAllocLocals)
				caseNonOwnedTags := cloneOwnedReturnSourceNonOwnedEnumTags(nonOwnedEnumTagLocals)
				collectDirectOwnedReturnInfo(
					c.Body,
					fn,
					funcs,
					types,
					known,
					knownThrows,
					caseLocals,
					caseConditionalLocals,
					caseNonOwnedTags,
					returnedSource,
					returnedSlot,
					returnCount,
					conditionalReturn,
					valid,
				)
				if !*valid {
					return
				}
				if mergeCaseFacts {
					if mergedLocals == nil {
						mergedLocals = caseLocals
						mergedConditionals = caseConditionalLocals
						mergedNonOwnedTags = caseNonOwnedTags
					} else {
						nextLocals := map[string]int{}
						nextConditionals := map[string]ownedReturnCondition{}
						nextNonOwnedTags := map[string]int32{}
						mergeOwnedReturnSourceIfBranchFacts(
							nextLocals,
							nextConditionals,
							nextNonOwnedTags,
							mergedLocals,
							mergedConditionals,
							mergedNonOwnedTags,
							caseLocals,
							caseConditionalLocals,
							caseNonOwnedTags,
							fn,
							types,
						)
						mergedLocals = nextLocals
						mergedConditionals = nextConditionals
						mergedNonOwnedTags = nextNonOwnedTags
					}
				}
			}
			if mergeCaseFacts && mergedLocals != nil {
				for name := range allocLocals {
					delete(allocLocals, name)
				}
				for name := range conditionalAllocLocals {
					delete(conditionalAllocLocals, name)
				}
				for name := range nonOwnedEnumTagLocals {
					delete(nonOwnedEnumTagLocals, name)
				}
				for name, slot := range mergedLocals {
					allocLocals[name] = slot
				}
				for name, condition := range mergedConditionals {
					setOwnedReturnSourceCondition(conditionalAllocLocals, name, condition)
				}
				for name, tag := range mergedNonOwnedTags {
					nonOwnedEnumTagLocals[name] = tag
				}
			}
		case *frontend.UnsafeStmt:
			collectDirectOwnedReturnInfo(
				s.Body,
				fn,
				funcs,
				types,
				known,
				knownThrows,
				allocLocals,
				conditionalAllocLocals,
				nonOwnedEnumTagLocals,
				returnedSource,
				returnedSlot,
				returnCount,
				conditionalReturn,
				valid,
			)
		}
	}
}

func collectDirectOwnedThrowInfo(
	stmts []frontend.Stmt,
	fn semantics.CheckedFunc,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	knownReturns map[string]ownedReturnSummary,
	allocLocals map[string]int,
	thrownSource *string,
	thrownSlot *int,
	thrownMultiSource *bool,
	throwCount *int,
	valid *bool,
) {
	if !*valid {
		return
	}
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			info, ok := fn.Locals[s.Name]
			if _, storageOK := ownedReturnStorageSlot(info.TypeName, types); ok && storageOK {
				if info.TypeName == "ptr" && isDirectAllocBytesCall(s.Value) {
					allocLocals[s.Name] = 0
					continue
				}
				if source, sourceSlot, ok := directOwnedReturnSource(s.Value, funcs, knownReturns, allocLocals); ok {
					allocLocals[s.Name] = sourceSlot
					forgetMovedOwnedReturnSource(allocLocals, source, s.Name)
					continue
				}
				if source, destSlot, ok := enumConstructorOwnedReturnSource(
					s.Value,
					info.TypeName,
					funcs,
					types,
					knownReturns,
					allocLocals,
				); ok {
					allocLocals[s.Name] = destSlot
					forgetMovedOwnedReturnSource(allocLocals, source, s.Name)
					continue
				}
				if source, destSlot, fieldName, fieldSlot, ok := structLiteralOwnedReturnSource(
					s.Value,
					info.TypeName,
					funcs,
					types,
					knownReturns,
					allocLocals,
				); ok {
					allocLocals[s.Name] = destSlot
					if fieldName != "" {
						allocLocals[fieldOwnedReturnSourceName(s.Name, []string{fieldName})] = fieldSlot
					}
					forgetMovedOwnedReturnSource(allocLocals, source, s.Name)
				}
			}
		case *frontend.AssignStmt:
			id, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			info, ok := fn.Locals[id.Name]
			if !ok || !ownedReturnStorageType(info.TypeName, types) {
				continue
			}
			if s.Op == 0 {
				if info.TypeName == "ptr" && isDirectAllocBytesCall(s.Value) {
					allocLocals[id.Name] = 0
					continue
				}
				if source, sourceSlot, ok := directOwnedReturnSource(s.Value, funcs, knownReturns, allocLocals); ok {
					allocLocals[id.Name] = sourceSlot
					forgetMovedOwnedReturnSource(allocLocals, source, id.Name)
					continue
				}
				if source, destSlot, ok := enumConstructorOwnedReturnSource(
					s.Value,
					info.TypeName,
					funcs,
					types,
					knownReturns,
					allocLocals,
				); ok {
					allocLocals[id.Name] = destSlot
					forgetMovedOwnedReturnSource(allocLocals, source, id.Name)
					continue
				}
				if source, destSlot, fieldName, fieldSlot, ok := structLiteralOwnedReturnSource(
					s.Value,
					info.TypeName,
					funcs,
					types,
					knownReturns,
					allocLocals,
				); ok {
					allocLocals[id.Name] = destSlot
					if fieldName != "" {
						allocLocals[fieldOwnedReturnSourceName(id.Name, []string{fieldName})] = fieldSlot
					}
					forgetMovedOwnedReturnSource(allocLocals, source, id.Name)
					continue
				}
			}
			deleteOwnedReturnSourcesForBase(allocLocals, id.Name)
		case *frontend.ThrowStmt:
			if _, ok := enumConstructorNonOwnedTag(s.Value, fn.ThrowsType, types); ok {
				continue
			}
			*throwCount++
			source, sourceSlot, ok := directOwnedReturnSource(s.Value, funcs, knownReturns, allocLocals)
			if !ok {
				if enumSource, destSlot, enumOK := enumConstructorOwnedReturnSource(
					s.Value,
					fn.ThrowsType,
					funcs,
					types,
					knownReturns,
					allocLocals,
				); enumOK {
					source = enumSource
					sourceSlot = destSlot
					ok = true
				}
			}
			if !ok {
				if structSource, destSlot, fieldName, _, structOK := structLiteralOwnedReturnSource(
					s.Value,
					fn.ThrowsType,
					funcs,
					types,
					knownReturns,
					allocLocals,
				); structOK {
					if structSource == "" && fieldName != "" {
						structSource = fmt.Sprintf("inline:core.alloc_bytes:%s.%s", fn.ThrowsType, fieldName)
					}
					if structSource == "" {
						*valid = false
						return
					}
					source = structSource
					sourceSlot = destSlot
					ok = true
				}
			}
			if !ok {
				*valid = false
				return
			}
			if *thrownSource == "" {
				*thrownSource = source
				*thrownSlot = sourceSlot
			} else if *thrownSlot != sourceSlot {
				*valid = false
				return
			} else if *thrownSource != source {
				*thrownMultiSource = true
			}
		case *frontend.IfStmt:
			thenLocals := cloneOwnedReturnSourceLocals(allocLocals)
			collectDirectOwnedThrowInfo(
				s.Then,
				fn,
				funcs,
				types,
				knownReturns,
				thenLocals,
				thrownSource,
				thrownSlot,
				thrownMultiSource,
				throwCount,
				valid,
			)
			if !*valid {
				return
			}
			if len(s.Else) > 0 {
				elseLocals := cloneOwnedReturnSourceLocals(allocLocals)
				collectDirectOwnedThrowInfo(
					s.Else,
					fn,
					funcs,
					types,
					knownReturns,
					elseLocals,
					thrownSource,
					thrownSlot,
					thrownMultiSource,
					throwCount,
					valid,
				)
			}
		case *frontend.UnsafeStmt:
			collectDirectOwnedThrowInfo(
				s.Body,
				fn,
				funcs,
				types,
				knownReturns,
				allocLocals,
				thrownSource,
				thrownSlot,
				thrownMultiSource,
				throwCount,
				valid,
			)
		}
	}
}

func stmtListAlwaysThrows(stmts []frontend.Stmt) bool {
	for _, stmt := range stmts {
		if stmtAlwaysThrows(stmt) {
			return true
		}
		if _, ok := stmt.(*frontend.ReturnStmt); ok {
			return false
		}
	}
	return false
}

func stmtAlwaysThrows(stmt frontend.Stmt) bool {
	switch s := stmt.(type) {
	case *frontend.ThrowStmt:
		return true
	case *frontend.UnsafeStmt:
		return stmtListAlwaysThrows(s.Body)
	case *frontend.IfStmt:
		return len(s.Else) > 0 && stmtListAlwaysThrows(s.Then) && stmtListAlwaysThrows(s.Else)
	case *frontend.MatchStmt:
		if len(s.Cases) == 0 {
			return false
		}
		for _, c := range s.Cases {
			if !stmtListAlwaysThrows(c.Body) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func cloneOwnedReturnSourceLocals(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for name, owned := range in {
		out[name] = owned
	}
	return out
}

type ownedReturnCondition struct {
	conditional      bool
	hasExactTagValue bool
	exactTagValue    int32
}

func cloneOwnedReturnSourceConditionals(in map[string]ownedReturnCondition) map[string]ownedReturnCondition {
	out := make(map[string]ownedReturnCondition, len(in))
	for name, condition := range in {
		out[name] = condition
	}
	return out
}

func cloneOwnedReturnSourceNonOwnedEnumTags(in map[string]int32) map[string]int32 {
	out := make(map[string]int32, len(in))
	for name, tag := range in {
		out[name] = tag
	}
	return out
}

func mergeOwnedReturnSourceBranchFacts(
	out map[string]int,
	conditionals map[string]ownedReturnCondition,
	thenLocals map[string]int,
	thenConditionals map[string]ownedReturnCondition,
	elseLocals map[string]int,
	elseConditionals map[string]ownedReturnCondition,
) {
	for name := range out {
		delete(out, name)
	}
	for name := range conditionals {
		delete(conditionals, name)
	}
	for name, thenSlot := range thenLocals {
		elseSlot, ok := elseLocals[name]
		if !ok || elseSlot != thenSlot {
			continue
		}
		thenCondition := thenConditionals[name]
		elseCondition := elseConditionals[name]
		if thenCondition != elseCondition {
			continue
		}
		out[name] = thenSlot
		setOwnedReturnSourceCondition(conditionals, name, thenCondition)
	}
}

func mergeOwnedReturnSourceIfBranchFacts(
	out map[string]int,
	conditionals map[string]ownedReturnCondition,
	nonOwnedTags map[string]int32,
	thenLocals map[string]int,
	thenConditionals map[string]ownedReturnCondition,
	thenNonOwnedTags map[string]int32,
	elseLocals map[string]int,
	elseConditionals map[string]ownedReturnCondition,
	elseNonOwnedTags map[string]int32,
	fn semantics.CheckedFunc,
	types map[string]*semantics.TypeInfo,
) {
	for name := range out {
		delete(out, name)
	}
	for name := range conditionals {
		delete(conditionals, name)
	}
	for name := range nonOwnedTags {
		delete(nonOwnedTags, name)
	}
	mergedNames := map[string]bool{}
	for name, thenSlot := range thenLocals {
		thenCondition := thenConditionals[name]
		if elseSlot, ok := elseLocals[name]; ok && elseSlot == thenSlot {
			elseCondition := elseConditionals[name]
			if thenCondition == elseCondition {
				out[name] = thenSlot
				setOwnedReturnSourceCondition(conditionals, name, thenCondition)
				mergedNames[name] = true
			}
			continue
		}
		if elseTag, ok := elseNonOwnedTags[name]; ok {
			if condition, ok := conditionalOwnedReturnConditionForBranchLocal(
				name,
				thenSlot,
				thenCondition,
				elseTag,
				fn,
				types,
			); ok {
				out[name] = thenSlot
				setOwnedReturnSourceCondition(conditionals, name, condition)
				mergedNames[name] = true
			}
		}
	}
	for name, elseSlot := range elseLocals {
		if mergedNames[name] {
			continue
		}
		elseCondition := elseConditionals[name]
		if thenTag, ok := thenNonOwnedTags[name]; ok {
			if condition, ok := conditionalOwnedReturnConditionForBranchLocal(
				name,
				elseSlot,
				elseCondition,
				thenTag,
				fn,
				types,
			); ok {
				out[name] = elseSlot
				setOwnedReturnSourceCondition(conditionals, name, condition)
				mergedNames[name] = true
			}
		}
	}
	for name, thenTag := range thenNonOwnedTags {
		if elseTag, ok := elseNonOwnedTags[name]; ok && elseTag == thenTag && !mergedNames[name] {
			nonOwnedTags[name] = thenTag
		}
	}
}

func conditionalOwnedReturnConditionForBranchLocal(
	name string,
	ownedSlot int,
	ownedCondition ownedReturnCondition,
	nonOwnedTag int32,
	fn semantics.CheckedFunc,
	types map[string]*semantics.TypeInfo,
) (ownedReturnCondition, bool) {
	info, ok := fn.Locals[name]
	if !ok {
		return ownedReturnCondition{}, false
	}
	ownedTag, ok := uniqueOwnedEnumTagValue(info.TypeName, ownedSlot, types)
	if !ok || ownedTag == nonOwnedTag {
		return ownedReturnCondition{}, false
	}
	if _, ok := conditionalOwnedReturnGuardSlot(info.TypeName, ownedSlot, types); !ok {
		return ownedReturnCondition{}, false
	}
	if ownedCondition.conditional {
		if !ownedCondition.hasExactTagValue || ownedCondition.exactTagValue != ownedTag {
			return ownedReturnCondition{}, false
		}
		return ownedCondition, true
	}
	return ownedReturnCondition{
		conditional:      true,
		hasExactTagValue: true,
		exactTagValue:    ownedTag,
	}, true
}

func matchStmtOwnedReturnBranchMergeOK(s *frontend.MatchStmt) bool {
	if s == nil || !matchStmtHasUnguardedDefault(s) {
		return false
	}
	for _, c := range s.Cases {
		if c.Guard != nil || stmtListContainsReturnOrThrow(c.Body) {
			return false
		}
	}
	return true
}

func stmtListContainsReturnOrThrow(stmts []frontend.Stmt) bool {
	for _, stmt := range stmts {
		if stmtContainsReturnOrThrow(stmt) {
			return true
		}
	}
	return false
}

func stmtContainsReturnOrThrow(stmt frontend.Stmt) bool {
	switch s := stmt.(type) {
	case *frontend.ReturnStmt, *frontend.ThrowStmt:
		return true
	case *frontend.UnsafeStmt:
		return stmtListContainsReturnOrThrow(s.Body)
	case *frontend.IfStmt:
		return stmtListContainsReturnOrThrow(s.Then) || stmtListContainsReturnOrThrow(s.Else)
	case *frontend.IfLetStmt:
		return stmtListContainsReturnOrThrow(s.Then) || stmtListContainsReturnOrThrow(s.Else)
	case *frontend.WhileStmt:
		return stmtListContainsReturnOrThrow(s.Body)
	case *frontend.ForRangeStmt:
		return stmtListContainsReturnOrThrow(s.Body)
	case *frontend.MatchStmt:
		for _, c := range s.Cases {
			if stmtListContainsReturnOrThrow(c.Body) {
				return true
			}
		}
	}
	return false
}

func ownedReturnSourceCondition(conditionals map[string]ownedReturnCondition, source string) ownedReturnCondition {
	return conditionals[source]
}

func setOwnedReturnSourceCondition(
	conditionals map[string]ownedReturnCondition,
	source string,
	condition ownedReturnCondition,
) {
	if condition.conditional {
		conditionals[source] = condition
		return
	}
	delete(conditionals, source)
}

func mergeOwnedReturnConditions(a, b ownedReturnCondition) ownedReturnCondition {
	if !a.conditional {
		return b
	}
	if !b.conditional {
		return a
	}
	merged := ownedReturnCondition{conditional: true}
	if a.hasExactTagValue && b.hasExactTagValue && a.exactTagValue == b.exactTagValue {
		merged.hasExactTagValue = true
		merged.exactTagValue = a.exactTagValue
	}
	return merged
}

func forgetMovedOwnedReturnSource(allocLocals map[string]int, source string, destinationBase string) {
	if source == "" || len(source) >= 5 && source[:5] == "call:" {
		return
	}
	sourceBase := source
	for i := 0; i < len(source); i++ {
		if source[i] == '.' {
			sourceBase = source[:i]
			break
		}
	}
	if sourceBase == "" || sourceBase == destinationBase {
		return
	}
	deleteOwnedReturnSourcesForBase(allocLocals, sourceBase)
}

func forgetMovedOwnedReturnConditionalSource(
	conditionals map[string]ownedReturnCondition,
	source string,
	destinationBase string,
) {
	if source == "" || len(source) >= 5 && source[:5] == "call:" {
		return
	}
	sourceBase := source
	for i := 0; i < len(source); i++ {
		if source[i] == '.' {
			sourceBase = source[:i]
			break
		}
	}
	if sourceBase == "" || sourceBase == destinationBase {
		return
	}
	deleteOwnedReturnConditionalSourcesForBase(conditionals, sourceBase)
}

func deleteOwnedReturnSourcesForBase(allocLocals map[string]int, base string) {
	delete(allocLocals, base)
	prefix := base + "."
	for name := range allocLocals {
		if strings.HasPrefix(name, prefix) {
			delete(allocLocals, name)
		}
	}
}

func deleteOwnedReturnConditionalSourcesForBase(conditionals map[string]ownedReturnCondition, base string) {
	delete(conditionals, base)
	prefix := base + "."
	for name := range conditionals {
		if strings.HasPrefix(name, prefix) {
			delete(conditionals, name)
		}
	}
}

func fieldOwnedReturnSourceName(base string, fields []string) string {
	source := base
	for _, field := range fields {
		source += "." + field
	}
	return source
}

func directOwnedReturnSource(
	expr frontend.Expr,
	funcs map[string]semantics.FuncSig,
	known map[string]ownedReturnSummary,
	allocLocals map[string]int,
) (string, int, bool) {
	if isDirectAllocBytesCall(expr) {
		return "inline:core.alloc_bytes", 0, true
	}
	if id, ok := expr.(*frontend.IdentExpr); ok {
		if slot, ok := allocLocals[id.Name]; ok {
			return id.Name, slot, true
		}
		return "", 0, false
	}
	if baseName, fields, _, ok := splitFieldPathLower(expr); ok && len(fields) > 0 {
		source := fieldOwnedReturnSourceName(baseName, fields)
		if slot, ok := allocLocals[source]; ok {
			return source, slot, true
		}
		return "", 0, false
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return "", 0, false
	}
	call = lowerCallExprWithBuiltinAlias(call)
	summary, ok := known[call.Name]
	if !ok {
		return "", 0, false
	}
	if summary.conditional {
		return "", 0, false
	}
	sig, ok := funcs[call.Name]
	if !ok || sig.ReturnSlots <= summary.returnSlot || sig.ThrowsType != "" {
		return "", 0, false
	}
	for _, ownership := range sig.ParamOwnership {
		if ownership == "inout" {
			return "", 0, false
		}
	}
	return "call:" + call.Name, summary.returnSlot, true
}

func conditionalOwnedReturnCallSource(
	expr frontend.Expr,
	funcs map[string]semantics.FuncSig,
	known map[string]ownedReturnSummary,
) (string, int, ownedReturnCondition, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return "", 0, ownedReturnCondition{}, false
	}
	call = lowerCallExprWithBuiltinAlias(call)
	summary, ok := known[call.Name]
	if !ok || !summary.conditional {
		return "", 0, ownedReturnCondition{}, false
	}
	sig, ok := funcs[call.Name]
	if !ok || sig.ReturnSlots <= summary.returnSlot || sig.ThrowsType != "" {
		return "", 0, ownedReturnCondition{}, false
	}
	for _, ownership := range sig.ParamOwnership {
		if ownership == "inout" {
			return "", 0, ownedReturnCondition{}, false
		}
	}
	condition := ownedReturnCondition{
		conditional:      true,
		hasExactTagValue: summary.hasConditionalTagExactValue,
		exactTagValue:    summary.conditionalTagExactValue,
	}
	return "call:" + call.Name, summary.returnSlot, condition, true
}

func structLiteralOwnedReturnSource(
	expr frontend.Expr,
	destType string,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	known map[string]ownedReturnSummary,
	allocLocals map[string]int,
) (string, int, string, int, bool) {
	source, destSlot, fieldPath, fieldSlot, ok := structLiteralOwnedReturnSourceInType(
		expr,
		destType,
		funcs,
		types,
		known,
		allocLocals,
		map[string]bool{},
	)
	if !ok {
		return "", 0, "", 0, false
	}
	return source, destSlot, strings.Join(fieldPath, "."), fieldSlot, true
}

func enumConstructorOwnedReturnSource(
	expr frontend.Expr,
	destType string,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	known map[string]ownedReturnSummary,
	allocLocals map[string]int,
) (string, int, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return "", 0, false
	}
	typeName, caseInfo, ok := enumCaseConstructorInfo(call, destType, types)
	if !ok || typeName != destType {
		return "", 0, false
	}
	destOwnedSlot, ok := ownedReturnStorageSlot(destType, types)
	if !ok {
		return "", 0, false
	}
	payloadOffset := 1
	for i, arg := range call.Args {
		if i >= len(caseInfo.PayloadTypes) || i >= len(caseInfo.PayloadSlots) {
			return "", 0, false
		}
		payloadType := caseInfo.PayloadTypes[i]
		payloadSlots := caseInfo.PayloadSlots[i]
		payloadOwnedSlot, ok := ownedReturnStorageSlot(payloadType, types)
		if !ok || payloadOwnedSlot < 0 || payloadOwnedSlot >= payloadSlots {
			payloadOffset += payloadSlots
			continue
		}
		destSlot := payloadOffset + payloadOwnedSlot
		if destSlot != destOwnedSlot {
			return "", 0, false
		}
		if payloadType == "ptr" && isDirectAllocBytesCall(arg) {
			return fmt.Sprintf("inline:core.alloc_bytes:%s.%s", destType, caseInfo.Name), destSlot, true
		}
		source, sourceSlot, ok := directOwnedReturnSource(arg, funcs, known, allocLocals)
		if ok && sourceSlot == payloadOwnedSlot {
			return source, destSlot, true
		}
		if nestedSource, nestedSlot, _, _, nestedOK := structLiteralOwnedReturnSource(
			arg,
			payloadType,
			funcs,
			types,
			known,
			allocLocals,
		); nestedOK && nestedSlot == payloadOwnedSlot {
			if nestedSource == "" {
				nestedSource = fmt.Sprintf("inline:core.alloc_bytes:%s.%s", destType, caseInfo.Name)
			}
			return nestedSource, destSlot, true
		}
		return "", 0, false
	}
	return "", 0, false
}

func matchExprOwnedReturnSource(
	expr frontend.Expr,
	destType string,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	known map[string]ownedReturnSummary,
	allocLocals map[string]int,
) (string, int, bool) {
	source, slot, _, ok := matchExprOwnedReturnSourceInfo(expr, destType, funcs, types, known, allocLocals)
	return source, slot, ok
}

func matchExprOwnedReturnSourceInfo(
	expr frontend.Expr,
	destType string,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	known map[string]ownedReturnSummary,
	allocLocals map[string]int,
) (string, int, ownedReturnCondition, bool) {
	matchExpr, ok := expr.(*frontend.MatchExpr)
	if !ok || matchExpr == nil || len(matchExpr.Cases) == 0 {
		return "", 0, ownedReturnCondition{}, false
	}
	destOwnedSlot, ok := ownedReturnStorageSlot(destType, types)
	if !ok {
		return "", 0, ownedReturnCondition{}, false
	}
	if !matchExprOwnedResultGuardCoverageOK(matchExpr.Cases) {
		return "", 0, ownedReturnCondition{}, false
	}
	source := ""
	hasOwnedArm := false
	hasNonOwnedArm := false
	allNonOwnedTagsZero := true
	ownedTagValue := int32(-1)
	for _, c := range matchExpr.Cases {
		armSource, armSlot, ok := enumConstructorOwnedReturnSource(
			c.Value,
			destType,
			funcs,
			types,
			known,
			allocLocals,
		)
		if !ok {
			nonOwnedTag, tagOK := enumConstructorNonOwnedTag(c.Value, destType, types)
			if !tagOK {
				return "", 0, ownedReturnCondition{}, false
			}
			if nonOwnedTag != 0 {
				allNonOwnedTagsZero = false
			}
			hasNonOwnedArm = true
			continue
		}
		if armSource == "" || armSlot != destOwnedSlot {
			return "", 0, ownedReturnCondition{}, false
		}
		if _, caseInfo, caseOK := enumConstructorInfoForValue(c.Value, destType, types); !caseOK {
			return "", 0, ownedReturnCondition{}, false
		} else if ownedTagValue < 0 {
			ownedTagValue = caseInfo.Ordinal
		} else if ownedTagValue != caseInfo.Ordinal {
			return "", 0, ownedReturnCondition{}, false
		}
		if source == "" {
			source = armSource
			hasOwnedArm = true
			continue
		}
		if source != armSource {
			return "", 0, ownedReturnCondition{}, false
		}
		hasOwnedArm = true
	}
	if !hasOwnedArm || source == "" {
		return "", 0, ownedReturnCondition{}, false
	}
	condition, ok := ownedReturnConditionForEnumMixedResult(hasNonOwnedArm, allNonOwnedTagsZero, ownedTagValue)
	if !ok {
		return "", 0, ownedReturnCondition{}, false
	}
	return source, destOwnedSlot, condition, true
}

func matchExprOwnedResultGuardCoverageOK(cases []frontend.MatchExprCase) bool {
	hasGuard := false
	defaultCount := 0
	hasUnguardedDefault := false
	for _, c := range cases {
		if c.Guard != nil {
			hasGuard = true
		}
		if !c.Default {
			continue
		}
		defaultCount++
		if c.Guard == nil {
			hasUnguardedDefault = true
		}
	}
	if defaultCount > 1 {
		return false
	}
	return !hasGuard || hasUnguardedDefault
}

func enumConstructorInfoForValue(
	expr frontend.Expr,
	destType string,
	types map[string]*semantics.TypeInfo,
) (string, semantics.EnumCaseInfo, bool) {
	switch value := expr.(type) {
	case *frontend.CallExpr:
		if value == nil {
			return "", semantics.EnumCaseInfo{}, false
		}
		return enumCaseConstructorInfo(value, destType, types)
	case *frontend.FieldAccessExpr:
		if value == nil || value.Field == "" {
			return "", semantics.EnumCaseInfo{}, false
		}
		candidateTypes := []string(nil)
		if value.EnumType != "" {
			candidateTypes = append(candidateTypes, value.EnumType)
		}
		if destType != "" && destType != value.EnumType {
			candidateTypes = append(candidateTypes, destType)
		}
		for _, typeName := range candidateTypes {
			info, ok := types[typeName]
			if !ok || info == nil || info.Kind != semantics.TypeEnum {
				continue
			}
			caseInfo, ok := info.CaseMap[value.Field]
			if !ok {
				continue
			}
			return typeName, caseInfo, true
		}
		return "", semantics.EnumCaseInfo{}, false
	default:
		return "", semantics.EnumCaseInfo{}, false
	}
}

func enumConstructorNonOwnedZeroTag(
	expr frontend.Expr,
	destType string,
	types map[string]*semantics.TypeInfo,
) bool {
	tag, ok := enumConstructorNonOwnedTag(expr, destType, types)
	return ok && tag == 0
}

func enumConstructorNonOwnedTag(
	expr frontend.Expr,
	destType string,
	types map[string]*semantics.TypeInfo,
) (int32, bool) {
	typeName, caseInfo, ok := enumConstructorInfoForValue(expr, destType, types)
	if !ok || typeName != destType || enumCaseHasOwnedPayload(caseInfo, types) {
		return 0, false
	}
	return caseInfo.Ordinal, true
}

func uniqueOwnedEnumTagValue(
	typeName string,
	returnSlot int,
	types map[string]*semantics.TypeInfo,
) (int32, bool) {
	info, ok := types[typeName]
	if !ok || info == nil || info.Kind != semantics.TypeEnum {
		return 0, false
	}
	var tag int32
	found := false
	for _, caseInfo := range info.EnumCases {
		payloadOffset := 1
		caseHasSlot := false
		for i, payloadType := range caseInfo.PayloadTypes {
			payloadSlots := 1
			if i < len(caseInfo.PayloadSlots) {
				payloadSlots = caseInfo.PayloadSlots[i]
			}
			payloadOwnedSlot, ok := ownedReturnStorageSlot(payloadType, types)
			if ok && payloadOwnedSlot >= 0 && payloadOwnedSlot < payloadSlots &&
				payloadOffset+payloadOwnedSlot == returnSlot {
				caseHasSlot = true
				break
			}
			payloadOffset += payloadSlots
		}
		if !caseHasSlot {
			continue
		}
		if found {
			return 0, false
		}
		found = true
		tag = caseInfo.Ordinal
	}
	return tag, found
}

func ownedReturnConditionForEnumMixedResult(
	hasNonOwnedArm bool,
	allNonOwnedTagsZero bool,
	ownedTagValue int32,
) (ownedReturnCondition, bool) {
	if !hasNonOwnedArm {
		return ownedReturnCondition{}, true
	}
	if ownedTagValue < 0 {
		return ownedReturnCondition{}, false
	}
	condition := ownedReturnCondition{conditional: true}
	if !allNonOwnedTagsZero || ownedTagValue == 0 {
		condition.hasExactTagValue = true
		condition.exactTagValue = ownedTagValue
	}
	return condition, true
}

func catchExprOwnedReturnSource(
	expr frontend.Expr,
	destType string,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	knownReturns map[string]ownedReturnSummary,
	knownThrows map[string]ownedThrowSummary,
	allocLocals map[string]int,
) (string, int, bool) {
	catchExpr, ok := expr.(*frontend.CatchExpr)
	if !ok || catchExpr == nil {
		return "", 0, false
	}
	call, ok := catchExpr.Call.(*frontend.CallExpr)
	if !ok || call == nil {
		return "", 0, false
	}
	call = lowerCallExprWithBuiltinAlias(call)
	sig, ok := funcs[call.Name]
	if !ok || sig.ThrowsType == "" || sig.ReturnType != destType {
		return "", 0, false
	}
	for _, ownership := range sig.ParamOwnership {
		if ownership == "inout" {
			return "", 0, false
		}
	}
	summary, ok := knownThrows[call.Name]
	if !ok {
		return "", 0, false
	}
	destSlot, ok := ownedReturnStorageSlot(destType, types)
	if !ok {
		return "", 0, false
	}
	successSummary, hasOwnedSuccess := knownReturns[call.Name]
	if hasOwnedSuccess &&
		(successSummary.returnSlot < 0 || successSummary.returnSlot != destSlot) {
		return "", 0, false
	}
	if !summary.alwaysThrows && !hasOwnedSuccess {
		return "", 0, false
	}
	if !catchExprRelaysOwnedError(
		catchExpr,
		sig.ThrowsType,
		destType,
		summary.errorSlot,
		types,
		func(value frontend.Expr, resultSlot int) bool {
			return catchCaseValueProducesOwnedResult(
				value,
				destType,
				resultSlot,
				funcs,
				types,
				knownReturns,
				allocLocals,
			)
		},
	) {
		return "", 0, false
	}
	return "catch:" + call.Name, destSlot, true
}

func catchExprNonOwnedErrorOwnedResultSource(
	expr frontend.Expr,
	destType string,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	knownReturns map[string]ownedReturnSummary,
) (string, int, bool) {
	source, slot, _, ok := catchExprNonOwnedErrorOwnedResultSourceInfo(
		expr,
		destType,
		funcs,
		types,
		knownReturns,
	)
	return source, slot, ok
}

func catchExprNonOwnedErrorOwnedResultSourceInfo(
	expr frontend.Expr,
	destType string,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	knownReturns map[string]ownedReturnSummary,
) (string, int, ownedReturnCondition, bool) {
	catchExpr, ok := expr.(*frontend.CatchExpr)
	if !ok || catchExpr == nil || len(catchExpr.Cases) == 0 {
		return "", 0, ownedReturnCondition{}, false
	}
	call, ok := catchExpr.Call.(*frontend.CallExpr)
	if !ok || call == nil {
		return "", 0, ownedReturnCondition{}, false
	}
	call = lowerCallExprWithBuiltinAlias(call)
	sig, ok := funcs[call.Name]
	if !ok || sig.ThrowsType == "" || sig.ReturnType != destType {
		return "", 0, ownedReturnCondition{}, false
	}
	for _, ownership := range sig.ParamOwnership {
		if ownership == "inout" {
			return "", 0, ownedReturnCondition{}, false
		}
	}
	if ownedStorageSlotCount(sig.ThrowsType, types) != 0 {
		return "", 0, ownedReturnCondition{}, false
	}
	destSlot, ok := ownedReturnStorageSlot(destType, types)
	if !ok {
		return "", 0, ownedReturnCondition{}, false
	}
	successSummary, ok := knownReturns[call.Name]
	if !ok ||
		successSummary.conditional ||
		successSummary.returnSlot < 0 ||
		successSummary.returnSlot != destSlot {
		return "", 0, ownedReturnCondition{}, false
	}
	hasDefault := false
	hasNonOwnedResult := false
	allNonOwnedTagsZero := true
	ownedTagValue := int32(-1)
	ownedTagKnown := false
	ownedTagUnknown := false
	for _, c := range catchExpr.Cases {
		if c.Default {
			if c.Guard != nil {
				return "", 0, ownedReturnCondition{}, false
			}
			if hasDefault {
				return "", 0, ownedReturnCondition{}, false
			}
			hasDefault = true
		}
		if !catchCaseValueProducesOwnedResult(
			c.Value,
			destType,
			destSlot,
			funcs,
			types,
			knownReturns,
			nil,
		) {
			if condition, ok := catchCaseValueProducesConditionalOwnedResult(
				c.Value,
				destType,
				destSlot,
				funcs,
				types,
				knownReturns,
			); ok {
				if !condition.hasExactTagValue {
					return "", 0, ownedReturnCondition{}, false
				}
				if ownedTagKnown {
					if ownedTagValue != condition.exactTagValue {
						return "", 0, ownedReturnCondition{}, false
					}
				} else {
					ownedTagKnown = true
					ownedTagValue = condition.exactTagValue
				}
				hasNonOwnedResult = true
				allNonOwnedTagsZero = false
				continue
			}
			nonOwnedTag, tagOK := enumConstructorNonOwnedTag(c.Value, destType, types)
			if !tagOK {
				return "", 0, ownedReturnCondition{}, false
			}
			if nonOwnedTag != 0 {
				allNonOwnedTagsZero = false
			}
			hasNonOwnedResult = true
			continue
		}
		if _, caseInfo, caseOK := enumConstructorInfoForValue(c.Value, destType, types); !caseOK {
			ownedTagUnknown = true
		} else if !ownedTagKnown {
			ownedTagKnown = true
			ownedTagValue = caseInfo.Ordinal
		} else if ownedTagValue != caseInfo.Ordinal {
			return "", 0, ownedReturnCondition{}, false
		}
	}
	if !hasDefault {
		return "", 0, ownedReturnCondition{}, false
	}
	if hasNonOwnedResult && !ownedTagKnown && !ownedTagUnknown {
		if tagValue, tagOK := uniqueOwnedEnumTagValue(destType, destSlot, types); tagOK {
			ownedTagKnown = true
			ownedTagValue = tagValue
		}
	}
	if hasNonOwnedResult && (ownedTagUnknown || !ownedTagKnown) {
		return "", 0, ownedReturnCondition{}, false
	}
	condition, conditionOK := ownedReturnConditionForEnumMixedResult(
		hasNonOwnedResult,
		allNonOwnedTagsZero,
		ownedTagValue,
	)
	if !conditionOK {
		return "", 0, ownedReturnCondition{}, false
	}
	if condition.conditional {
		if _, guardOK := conditionalOwnedReturnGuardSlot(destType, destSlot, types); !guardOK {
			return "", 0, ownedReturnCondition{}, false
		}
	}
	return "catch-result:" + call.Name, destSlot, condition, true
}

func catchExprOwnedErrorMixedResultSourceInfo(
	expr frontend.Expr,
	destType string,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	knownReturns map[string]ownedReturnSummary,
	knownThrows map[string]ownedThrowSummary,
) (string, int, ownedReturnCondition, bool) {
	catchExpr, ok := expr.(*frontend.CatchExpr)
	if !ok || catchExpr == nil || len(catchExpr.Cases) == 0 {
		return "", 0, ownedReturnCondition{}, false
	}
	call, ok := catchExpr.Call.(*frontend.CallExpr)
	if !ok || call == nil {
		return "", 0, ownedReturnCondition{}, false
	}
	call = lowerCallExprWithBuiltinAlias(call)
	sig, ok := funcs[call.Name]
	if !ok || sig.ThrowsType == "" || sig.ReturnType != destType {
		return "", 0, ownedReturnCondition{}, false
	}
	for _, ownership := range sig.ParamOwnership {
		if ownership == "inout" {
			return "", 0, ownedReturnCondition{}, false
		}
	}
	throwSummary, ok := knownThrows[call.Name]
	if !ok || throwSummary.errorSlot < 0 {
		return "", 0, ownedReturnCondition{}, false
	}
	if ownedStorageSlotCount(sig.ThrowsType, types) != 1 {
		return "", 0, ownedReturnCondition{}, false
	}
	errorInfo, ok := types[sig.ThrowsType]
	if !ok || errorInfo == nil || errorInfo.Kind != semantics.TypeEnum || len(errorInfo.EnumCases) == 0 {
		return "", 0, ownedReturnCondition{}, false
	}
	ownedErrorCaseName := ""
	ownedErrorCaseCount := 0
	for _, errorCase := range errorInfo.EnumCases {
		errorSlot, hasOwnedPayload := enumCaseOwnedPayloadSlot(errorCase, types)
		if !hasOwnedPayload {
			continue
		}
		if errorSlot != throwSummary.errorSlot {
			return "", 0, ownedReturnCondition{}, false
		}
		ownedErrorCaseCount++
		if ownedErrorCaseCount > 1 {
			return "", 0, ownedReturnCondition{}, false
		}
		ownedErrorCaseName = errorCase.Name
	}
	if ownedErrorCaseCount != 1 || ownedErrorCaseName == "" {
		return "", 0, ownedReturnCondition{}, false
	}
	destSlot, ok := ownedReturnStorageSlot(destType, types)
	if !ok {
		return "", 0, ownedReturnCondition{}, false
	}
	successSummary, ok := knownReturns[call.Name]
	if !ok ||
		successSummary.conditional ||
		successSummary.returnSlot < 0 ||
		successSummary.returnSlot != destSlot {
		return "", 0, ownedReturnCondition{}, false
	}
	ownedTagValue, ok := uniqueOwnedEnumTagValue(destType, destSlot, types)
	if !ok {
		return "", 0, ownedReturnCondition{}, false
	}
	hasDefault := false
	coveredCases := map[string]bool{}
	hasNonOwnedResult := false
	allNonOwnedTagsZero := true
	for _, c := range catchExpr.Cases {
		caseName := ""
		caseInfo := semantics.EnumCaseInfo{}
		hasCaseInfo := false
		if c.Default {
			if c.Guard != nil || hasDefault {
				return "", 0, ownedReturnCondition{}, false
			}
			hasDefault = true
		} else {
			var ok bool
			caseName, ok = catchEnumCaseName(c.Pattern)
			if !ok {
				return "", 0, ownedReturnCondition{}, false
			}
			caseInfo, hasCaseInfo = errorInfo.CaseMap[caseName]
			if !hasCaseInfo {
				return "", 0, ownedReturnCondition{}, false
			}
			if hasDefault {
				return "", 0, ownedReturnCondition{}, false
			}
			if c.Guard == nil {
				if coveredCases[caseName] {
					return "", 0, ownedReturnCondition{}, false
				}
				coveredCases[caseName] = true
			} else if coveredCases[caseName] || caseName != ownedErrorCaseName {
				return "", 0, ownedReturnCondition{}, false
			}
		}
		producesOwnedResult := catchCaseValueProducesOwnedResult(c.Value, destType, destSlot, funcs, types, knownReturns, nil)
		if !producesOwnedResult && hasCaseInfo && caseName == ownedErrorCaseName {
			producesOwnedResult = catchCaseWrapsOwnedEnumErrorInResultEnum(
				c,
				caseInfo,
				throwSummary.errorSlot,
				destType,
				destSlot,
				types,
			)
		}
		if producesOwnedResult {
			_, caseInfo, caseOK := enumConstructorInfoForValue(c.Value, destType, types)
			if caseOK && (!enumCaseHasOwnedPayload(caseInfo, types) || caseInfo.Ordinal != ownedTagValue) {
				return "", 0, ownedReturnCondition{}, false
			}
			continue
		}
		if condition, ok := catchCaseValueProducesConditionalOwnedResult(
			c.Value,
			destType,
			destSlot,
			funcs,
			types,
			knownReturns,
		); ok {
			if !condition.hasExactTagValue || condition.exactTagValue != ownedTagValue {
				return "", 0, ownedReturnCondition{}, false
			}
			hasNonOwnedResult = true
			allNonOwnedTagsZero = false
			continue
		}
		nonOwnedTag, tagOK := enumConstructorNonOwnedTag(c.Value, destType, types)
		if !tagOK {
			return "", 0, ownedReturnCondition{}, false
		}
		if nonOwnedTag != 0 {
			allNonOwnedTagsZero = false
		}
		hasNonOwnedResult = true
	}
	if !hasNonOwnedResult {
		return "", 0, ownedReturnCondition{}, false
	}
	if hasDefault {
		if !coveredCases[ownedErrorCaseName] &&
			!defaultCoversOnlySingleOwnedEnumPayloadCase(catchExpr.Cases, sig.ThrowsType, throwSummary.errorSlot, ownedErrorCaseName, types) {
			return "", 0, ownedReturnCondition{}, false
		}
	} else if len(coveredCases) != len(errorInfo.EnumCases) {
		return "", 0, ownedReturnCondition{}, false
	}
	condition, conditionOK := ownedReturnConditionForEnumMixedResult(
		hasNonOwnedResult,
		allNonOwnedTagsZero,
		ownedTagValue,
	)
	if !conditionOK {
		return "", 0, ownedReturnCondition{}, false
	}
	if condition.conditional {
		if _, guardOK := conditionalOwnedReturnGuardSlot(destType, destSlot, types); !guardOK {
			return "", 0, ownedReturnCondition{}, false
		}
	}
	return "catch-owned-error-result:" + call.Name, destSlot, condition, true
}

func catchCaseValueProducesOwnedResult(
	value frontend.Expr,
	resultType string,
	resultSlot int,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	knownReturns map[string]ownedReturnSummary,
	allocLocals map[string]int,
) bool {
	if source, sourceSlot, ok := directOwnedReturnSource(
		value,
		funcs,
		knownReturns,
		allocLocals,
	); ok && source != "" && sourceSlot == resultSlot {
		return true
	}
	if enumSource, enumSlot, ok := enumConstructorOwnedReturnSource(
		value,
		resultType,
		funcs,
		types,
		knownReturns,
		allocLocals,
	); ok && enumSource != "" && enumSlot == resultSlot {
		return true
	}
	if structSource, structSlot, fieldName, _, ok := structLiteralOwnedReturnSource(
		value,
		resultType,
		funcs,
		types,
		knownReturns,
		allocLocals,
	); ok && (structSource != "" || fieldName != "") && structSlot == resultSlot {
		return true
	}
	return false
}

func catchExprRelaysEnumOwnedError(
	catchExpr *frontend.CatchExpr,
	errorType string,
	resultType string,
	errorSlot int,
	types map[string]*semantics.TypeInfo,
	nonRelayValueOwned func(frontend.Expr, int) bool,
) bool {
	if catchExpr == nil || len(catchExpr.Cases) == 0 {
		return false
	}
	info, ok := types[errorType]
	if !ok || info.Kind != semantics.TypeEnum {
		return false
	}
	resultOwnedSlot, ok := ownedReturnStorageSlot(resultType, types)
	if !ok {
		return false
	}
	covered := map[string]bool{}
	guardedRelay := map[string]bool{}
	relayCount := 0
	defaultValue := frontend.Expr(nil)
	for _, c := range catchExpr.Cases {
		if c.Default {
			if c.Guard != nil {
				return false
			}
			if defaultValue != nil {
				return false
			}
			defaultValue = c.Value
			continue
		}
		caseName, ok := catchEnumCaseName(c.Pattern)
		if !ok {
			return false
		}
		caseInfo, ok := info.CaseMap[caseName]
		if !ok {
			return false
		}
		caseHasOwnedPayload := enumCaseHasOwnedPayload(caseInfo, types)
		if c.Guard != nil {
			if covered[caseName] {
				return false
			}
			if caseHasOwnedPayload {
				if !catchCaseRelaysOwnedEnumError(c, caseInfo, errorSlot, resultOwnedSlot, types) &&
					!catchCaseWrapsOwnedEnumErrorInResultEnum(c, caseInfo, errorSlot, resultType, resultOwnedSlot, types) &&
					(nonRelayValueOwned == nil || !nonRelayValueOwned(c.Value, resultOwnedSlot)) {
					return false
				}
				guardedRelay[caseName] = true
				continue
			}
			if nonRelayValueOwned == nil || !nonRelayValueOwned(c.Value, resultOwnedSlot) {
				return false
			}
			continue
		}
		if covered[caseName] {
			return false
		}
		covered[caseName] = true
		if caseHasOwnedPayload {
			if catchCaseRelaysOwnedEnumError(c, caseInfo, errorSlot, resultOwnedSlot, types) ||
				catchCaseWrapsOwnedEnumErrorInResultEnum(c, caseInfo, errorSlot, resultType, resultOwnedSlot, types) {
				relayCount++
				continue
			}
			if c.Guard != nil ||
				nonRelayValueOwned == nil ||
				!nonRelayValueOwned(c.Value, resultOwnedSlot) {
				return false
			}
			relayCount++
			continue
		}
		if nonRelayValueOwned == nil || !nonRelayValueOwned(c.Value, resultOwnedSlot) {
			return false
		}
	}
	for caseName := range guardedRelay {
		if !covered[caseName] {
			if _, ok := defaultCoversSingleOwnedEnumPayloadCaseAllowingNonOwned(
				catchExpr.Cases,
				errorType,
				errorSlot,
				caseName,
				types,
			); !ok {
				return false
			}
		}
	}
	if len(covered) == len(info.EnumCases) {
		return relayCount > 0
	}
	if defaultValue == nil ||
		nonRelayValueOwned == nil ||
		!nonRelayValueOwned(defaultValue, resultOwnedSlot) {
		return false
	}
	missingOwnedPayloadCases := 0
	for _, enumCase := range info.EnumCases {
		if covered[enumCase.Name] {
			continue
		}
		ownedSlot, hasOwnedPayload := enumCaseOwnedPayloadSlot(enumCase, types)
		if !hasOwnedPayload {
			continue
		}
		if ownedSlot != errorSlot {
			return false
		}
		missingOwnedPayloadCases++
	}
	if missingOwnedPayloadCases == 0 {
		return relayCount > 0
	}
	if missingOwnedPayloadCases != 1 {
		return false
	}
	_, ok = defaultCoversSingleOwnedEnumPayloadCaseAllowingNonOwned(
		catchExpr.Cases,
		errorType,
		errorSlot,
		"",
		types,
	)
	return ok
}

func catchExprRelaysOwnedError(
	catchExpr *frontend.CatchExpr,
	errorType string,
	resultType string,
	errorSlot int,
	types map[string]*semantics.TypeInfo,
	nonRelayValueOwned func(frontend.Expr, int) bool,
) bool {
	return catchExprRelaysEnumOwnedError(
		catchExpr,
		errorType,
		resultType,
		errorSlot,
		types,
		nonRelayValueOwned,
	) || catchExprRelaysOptionalOwnedError(
		catchExpr,
		errorType,
		resultType,
		errorSlot,
		types,
		nonRelayValueOwned,
	)
}

func catchExprRelaysOptionalOwnedError(
	catchExpr *frontend.CatchExpr,
	errorType string,
	resultType string,
	errorSlot int,
	types map[string]*semantics.TypeInfo,
	nonRelayValueOwned func(frontend.Expr, int) bool,
) bool {
	if catchExpr == nil || len(catchExpr.Cases) == 0 {
		return false
	}
	info, ok := types[errorType]
	if !ok || info.Kind != semantics.TypeOptional {
		return false
	}
	resultOwnedSlot, ok := ownedReturnStorageSlot(resultType, types)
	if !ok {
		return false
	}
	elemOwnedSlot, ok := ownedReturnStorageSlot(info.ElemType, types)
	if !ok || errorSlot != elemOwnedSlot || resultOwnedSlot != elemOwnedSlot {
		return false
	}
	seenSome := false
	seenNone := false
	defaultValue := frontend.Expr(nil)
	for _, c := range catchExpr.Cases {
		if c.Default {
			if c.Guard != nil || defaultValue != nil {
				return false
			}
			defaultValue = c.Value
			continue
		}
		switch c.Pattern.(type) {
		case *frontend.SomePatternExpr:
			if seenSome {
				return false
			}
			relaysPayload := catchCaseRelaysOwnedOptionalError(c, elemOwnedSlot, errorSlot, resultOwnedSlot)
			if c.Guard != nil {
				if !relaysPayload &&
					(nonRelayValueOwned == nil || !nonRelayValueOwned(c.Value, resultOwnedSlot)) {
					return false
				}
				continue
			}
			if !relaysPayload &&
				(nonRelayValueOwned == nil || !nonRelayValueOwned(c.Value, resultOwnedSlot)) {
				return false
			}
			seenSome = true
		case *frontend.NoneLitExpr:
			if seenNone {
				return false
			}
			if nonRelayValueOwned == nil || !nonRelayValueOwned(c.Value, resultOwnedSlot) {
				return false
			}
			if c.Guard != nil {
				continue
			}
			seenNone = true
		default:
			return false
		}
	}
	if seenSome {
		if seenNone {
			return true
		}
		return defaultValue != nil &&
			nonRelayValueOwned != nil &&
			nonRelayValueOwned(defaultValue, resultOwnedSlot)
	}
	return defaultValue != nil &&
		nonRelayValueOwned != nil &&
		nonRelayValueOwned(defaultValue, resultOwnedSlot) &&
		defaultCoversOnlyOptionalSomeOwnedPayload(catchExpr.Cases, errorType, errorSlot, types)
}

func catchEnumCaseName(pattern frontend.Expr) (string, bool) {
	switch pat := pattern.(type) {
	case *frontend.EnumCasePatternExpr:
		if pat == nil || pat.CaseName == "" {
			return "", false
		}
		return pat.CaseName, true
	case *frontend.FieldAccessExpr:
		if pat == nil || pat.Field == "" {
			return "", false
		}
		return pat.Field, true
	default:
		return "", false
	}
}

func enumCaseHasOwnedPayload(
	caseInfo semantics.EnumCaseInfo,
	types map[string]*semantics.TypeInfo,
) bool {
	_, ok := enumCaseOwnedPayloadSlot(caseInfo, types)
	return ok
}

func enumCaseOwnedPayloadSlot(
	caseInfo semantics.EnumCaseInfo,
	types map[string]*semantics.TypeInfo,
) (int, bool) {
	payloadOffset := 1
	ownedSlot := -1
	for i, payloadType := range caseInfo.PayloadTypes {
		payloadSlots := 1
		if i < len(caseInfo.PayloadSlots) {
			payloadSlots = caseInfo.PayloadSlots[i]
		}
		payloadOwnedSlot, ok := ownedReturnStorageSlot(payloadType, types)
		if ok && payloadOwnedSlot >= 0 && payloadOwnedSlot < payloadSlots {
			if ownedSlot >= 0 {
				return -1, false
			}
			ownedSlot = payloadOffset + payloadOwnedSlot
		}
		payloadOffset += payloadSlots
	}
	if ownedSlot < 0 {
		return -1, false
	}
	return ownedSlot, true
}

func defaultCoversOnlySingleOwnedEnumPayload(
	cases []frontend.CatchExprCase,
	errorType string,
	errorSlot int,
	types map[string]*semantics.TypeInfo,
) bool {
	return defaultCoversOnlySingleOwnedEnumPayloadCase(cases, errorType, errorSlot, "", types)
}

func defaultCoversOnlySingleOwnedEnumPayloadCase(
	cases []frontend.CatchExprCase,
	errorType string,
	errorSlot int,
	wantCaseName string,
	types map[string]*semantics.TypeInfo,
) bool {
	info, ok := types[errorType]
	if !ok || info == nil || info.Kind != semantics.TypeEnum {
		return false
	}
	hasDefault := false
	covered := map[string]bool{}
	for _, c := range cases {
		if c.Default {
			hasDefault = true
			continue
		}
		if c.Guard != nil {
			continue
		}
		caseName, ok := catchEnumCaseName(c.Pattern)
		if !ok {
			return false
		}
		covered[caseName] = true
	}
	if !hasDefault {
		return false
	}
	missingOwnedPayloadCases := 0
	for _, enumCase := range info.EnumCases {
		if covered[enumCase.Name] {
			continue
		}
		ownedSlot, hasOwnedPayload := enumCaseOwnedPayloadSlot(enumCase, types)
		if !hasOwnedPayload || ownedSlot != errorSlot {
			return false
		}
		if wantCaseName != "" && enumCase.Name != wantCaseName {
			return false
		}
		missingOwnedPayloadCases++
	}
	return missingOwnedPayloadCases == 1
}

func defaultCoversSingleOwnedEnumPayloadCaseAllowingNonOwned(
	cases []frontend.CatchExprCase,
	errorType string,
	errorSlot int,
	wantCaseName string,
	types map[string]*semantics.TypeInfo,
) (semantics.EnumCaseInfo, bool) {
	info, ok := types[errorType]
	if !ok || info == nil || info.Kind != semantics.TypeEnum {
		return semantics.EnumCaseInfo{}, false
	}
	hasDefault := false
	covered := map[string]bool{}
	for _, c := range cases {
		if c.Default {
			if c.Guard != nil {
				return semantics.EnumCaseInfo{}, false
			}
			hasDefault = true
			continue
		}
		if c.Guard != nil {
			continue
		}
		caseName, ok := catchEnumCaseName(c.Pattern)
		if !ok {
			return semantics.EnumCaseInfo{}, false
		}
		covered[caseName] = true
	}
	if !hasDefault {
		return semantics.EnumCaseInfo{}, false
	}
	missingOwnedPayloadCases := 0
	missingOwnedCase := semantics.EnumCaseInfo{}
	for _, enumCase := range info.EnumCases {
		if covered[enumCase.Name] {
			continue
		}
		ownedSlot, hasOwnedPayload := enumCaseOwnedPayloadSlot(enumCase, types)
		if !hasOwnedPayload {
			continue
		}
		if ownedSlot != errorSlot {
			return semantics.EnumCaseInfo{}, false
		}
		if wantCaseName != "" && enumCase.Name != wantCaseName {
			return semantics.EnumCaseInfo{}, false
		}
		missingOwnedPayloadCases++
		missingOwnedCase = enumCase
	}
	if missingOwnedPayloadCases != 1 {
		return semantics.EnumCaseInfo{}, false
	}
	return missingOwnedCase, true
}

func defaultCoversOnlyOptionalSomeOwnedPayload(
	cases []frontend.CatchExprCase,
	errorType string,
	errorSlot int,
	types map[string]*semantics.TypeInfo,
) bool {
	info, ok := types[errorType]
	if !ok || info == nil || info.Kind != semantics.TypeOptional {
		return false
	}
	elemOwnedSlot, ok := ownedReturnStorageSlot(info.ElemType, types)
	if !ok || elemOwnedSlot != errorSlot {
		return false
	}
	hasDefault := false
	seenUnguardedSome := false
	seenGuardedSome := false
	seenUnguardedNone := false
	for _, c := range cases {
		if c.Default {
			if c.Guard != nil {
				return false
			}
			hasDefault = true
			continue
		}
		switch c.Pattern.(type) {
		case *frontend.SomePatternExpr:
			if c.Guard == nil {
				seenUnguardedSome = true
			} else {
				seenGuardedSome = true
			}
		case *frontend.NoneLitExpr:
			if c.Guard != nil {
				return false
			}
			seenUnguardedNone = true
		default:
			return false
		}
	}
	return hasDefault && !seenUnguardedSome && (seenUnguardedNone || seenGuardedSome)
}

func catchCaseRelaysOwnedOptionalError(
	c frontend.CatchExprCase,
	elemOwnedSlot int,
	errorSlot int,
	resultOwnedSlot int,
) bool {
	some, ok := c.Pattern.(*frontend.SomePatternExpr)
	if !ok || some == nil {
		return false
	}
	id, ok := c.Value.(*frontend.IdentExpr)
	if !ok {
		return false
	}
	return some.Name == id.Name &&
		errorSlot == elemOwnedSlot &&
		resultOwnedSlot == elemOwnedSlot
}

func catchCaseNeedsOwnedErrorCleanup(
	c frontend.CatchExprCase,
	errorType string,
	resultType string,
	errorSlot int,
	types map[string]*semantics.TypeInfo,
) bool {
	if c.Default {
		return false
	}
	resultOwnedSlot, ok := ownedReturnStorageSlot(resultType, types)
	if !ok {
		return false
	}
	info, ok := types[errorType]
	if !ok || info == nil {
		return false
	}
	switch info.Kind {
	case semantics.TypeEnum:
		caseName, ok := catchEnumCaseName(c.Pattern)
		if !ok {
			return false
		}
		caseInfo, ok := info.CaseMap[caseName]
		if !ok || !enumCaseHasOwnedPayload(caseInfo, types) {
			return false
		}
		return !catchCaseRelaysOwnedEnumError(c, caseInfo, errorSlot, resultOwnedSlot, types) &&
			!catchCaseWrapsOwnedEnumErrorInResultEnum(c, caseInfo, errorSlot, resultType, resultOwnedSlot, types)
	case semantics.TypeOptional:
		if _, ok := c.Pattern.(*frontend.SomePatternExpr); !ok {
			return false
		}
		elemOwnedSlot, ok := ownedReturnStorageSlot(info.ElemType, types)
		if !ok || errorSlot != elemOwnedSlot || resultOwnedSlot != elemOwnedSlot {
			return false
		}
		return !catchCaseRelaysOwnedOptionalError(c, elemOwnedSlot, errorSlot, resultOwnedSlot)
	default:
		return false
	}
}

func catchCaseRelaysOwnedEnumError(
	c frontend.CatchExprCase,
	caseInfo semantics.EnumCaseInfo,
	errorSlot int,
	resultOwnedSlot int,
	types map[string]*semantics.TypeInfo,
) bool {
	enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr)
	if !ok || enumPat == nil {
		return false
	}
	id, ok := c.Value.(*frontend.IdentExpr)
	if !ok {
		return false
	}
	if len(enumPat.Bindings) != len(caseInfo.PayloadTypes) {
		return false
	}
	payloadOffset := 1
	for i, binding := range enumPat.Bindings {
		if i >= len(caseInfo.PayloadSlots) || i >= len(caseInfo.PayloadTypes) {
			return false
		}
		payloadSlots := caseInfo.PayloadSlots[i]
		payloadOwnedSlot, ok := ownedReturnStorageSlot(caseInfo.PayloadTypes[i], types)
		if ok &&
			binding == id.Name &&
			errorSlot == payloadOffset+payloadOwnedSlot &&
			resultOwnedSlot == payloadOwnedSlot {
			return true
		}
		payloadOffset += payloadSlots
	}
	return false
}

func catchCaseWrapsOwnedEnumErrorInResultEnum(
	c frontend.CatchExprCase,
	errorCaseInfo semantics.EnumCaseInfo,
	errorSlot int,
	resultType string,
	resultOwnedSlot int,
	types map[string]*semantics.TypeInfo,
) bool {
	enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr)
	if !ok || enumPat == nil {
		return false
	}
	wrapped, ok := c.Value.(*frontend.CallExpr)
	if !ok || wrapped == nil {
		return false
	}
	typeName, resultCaseInfo, ok := enumCaseConstructorInfo(wrapped, resultType, types)
	if !ok || typeName != resultType {
		return false
	}
	if len(enumPat.Bindings) != len(errorCaseInfo.PayloadTypes) {
		return false
	}
	bindingName := ""
	payloadOffset := 1
	for i, binding := range enumPat.Bindings {
		if i >= len(errorCaseInfo.PayloadSlots) || i >= len(errorCaseInfo.PayloadTypes) {
			return false
		}
		payloadSlots := errorCaseInfo.PayloadSlots[i]
		payloadOwnedSlot, ok := ownedReturnStorageSlot(errorCaseInfo.PayloadTypes[i], types)
		if ok && errorSlot == payloadOffset+payloadOwnedSlot {
			bindingName = binding
			break
		}
		payloadOffset += payloadSlots
	}
	if bindingName == "" {
		return false
	}
	payloadOffset = 1
	for i, arg := range wrapped.Args {
		if i >= len(resultCaseInfo.PayloadSlots) || i >= len(resultCaseInfo.PayloadTypes) {
			return false
		}
		payloadSlots := resultCaseInfo.PayloadSlots[i]
		payloadOwnedSlot, ok := ownedReturnStorageSlot(resultCaseInfo.PayloadTypes[i], types)
		if ok && resultOwnedSlot == payloadOffset+payloadOwnedSlot {
			id, ok := arg.(*frontend.IdentExpr)
			return ok && id.Name == bindingName
		}
		payloadOffset += payloadSlots
	}
	return false
}

func catchCaseValueProducesConditionalOwnedResult(
	value frontend.Expr,
	resultType string,
	resultSlot int,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	knownReturns map[string]ownedReturnSummary,
) (ownedReturnCondition, bool) {
	_, sourceSlot, condition, ok := conditionalOwnedReturnCallSource(value, funcs, knownReturns)
	if !ok || sourceSlot != resultSlot {
		return ownedReturnCondition{}, false
	}
	if _, guardOK := conditionalOwnedReturnGuardSlot(resultType, resultSlot, types); !guardOK {
		return ownedReturnCondition{}, false
	}
	return condition, true
}

func enumCaseConstructorInfo(
	call *frontend.CallExpr,
	destType string,
	types map[string]*semantics.TypeInfo,
) (string, semantics.EnumCaseInfo, bool) {
	if call == nil {
		return "", semantics.EnumCaseInfo{}, false
	}
	candidateTypes := []string(nil)
	if call.ResolvedType != "" {
		candidateTypes = append(candidateTypes, call.ResolvedType)
	}
	if destType != "" && destType != call.ResolvedType {
		candidateTypes = append(candidateTypes, destType)
	}
	parts := strings.Split(call.Name, ".")
	caseName := ""
	if len(parts) >= 2 {
		caseName = parts[len(parts)-1]
		typeName := strings.Join(parts[:len(parts)-1], ".")
		if typeName != "" && typeName != call.ResolvedType && typeName != destType {
			candidateTypes = append(candidateTypes, typeName)
		}
	}
	for _, typeName := range candidateTypes {
		info, ok := types[typeName]
		if !ok || info == nil || info.Kind != semantics.TypeEnum {
			continue
		}
		if caseName == "" {
			continue
		}
		caseInfo, ok := info.CaseMap[caseName]
		if !ok {
			continue
		}
		return typeName, caseInfo, true
	}
	return "", semantics.EnumCaseInfo{}, false
}

func structLiteralOwnedReturnSourceInType(
	expr frontend.Expr,
	destType string,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
	known map[string]ownedReturnSummary,
	allocLocals map[string]int,
	visiting map[string]bool,
) (string, int, []string, int, bool) {
	lit, ok := expr.(*frontend.StructLitExpr)
	if !ok || lit == nil {
		return "", 0, nil, 0, false
	}
	destOwnedSlot, ok := ownedReturnStorageSlot(destType, types)
	if !ok {
		return "", 0, nil, 0, false
	}
	info, ok := types[destType]
	if !ok || info == nil || info.Kind != semantics.TypeStruct {
		return "", 0, nil, 0, false
	}
	if visiting[destType] {
		return "", 0, nil, 0, false
	}
	visiting[destType] = true
	defer delete(visiting, destType)

	fieldValues := make(map[string]frontend.Expr, len(lit.Fields))
	for _, field := range lit.Fields {
		fieldValues[field.Name] = field.Value
	}
	for _, field := range info.Fields {
		fieldOwnedSlot, ok := ownedReturnStorageSlot(field.TypeName, types)
		if !ok || fieldOwnedSlot < 0 || fieldOwnedSlot >= field.SlotCount {
			continue
		}
		destSlot := field.Offset + fieldOwnedSlot
		if destSlot != destOwnedSlot {
			continue
		}
		value, ok := fieldValues[field.Name]
		if !ok {
			return "", 0, nil, 0, false
		}
		if field.TypeName == "ptr" && isDirectAllocBytesCall(value) {
			return "", destSlot, []string{field.Name}, fieldOwnedSlot, true
		}
		source, sourceSlot, ok := directOwnedReturnSource(value, funcs, known, allocLocals)
		if ok && sourceSlot == fieldOwnedSlot {
			return source, destSlot, []string{field.Name}, fieldOwnedSlot, true
		}
		nestedSource, nestedSlot, nestedPath, nestedFieldSlot, nestedOK := structLiteralOwnedReturnSourceInType(
			value,
			field.TypeName,
			funcs,
			types,
			known,
			allocLocals,
			visiting,
		)
		if !nestedOK || nestedSlot != fieldOwnedSlot {
			return "", 0, nil, 0, false
		}
		return nestedSource,
			destSlot,
			append([]string{field.Name}, nestedPath...),
			nestedFieldSlot,
			true
	}
	return "", 0, nil, 0, false
}

func ownedReturnStorageType(typeName string, types map[string]*semantics.TypeInfo) bool {
	_, ok := ownedReturnStorageSlot(typeName, types)
	return ok
}

func ownedReturnStorageSlot(typeName string, types map[string]*semantics.TypeInfo) (int, bool) {
	return ownedReturnStorageSlotInType(typeName, types, map[string]bool{})
}

func ownedStorageSlotCount(typeName string, types map[string]*semantics.TypeInfo) int {
	return ownedStorageSlotCountInType(typeName, types, map[string]bool{})
}

func ownedStorageSlotCountInType(
	typeName string,
	types map[string]*semantics.TypeInfo,
	visiting map[string]bool,
) int {
	if typeName == "ptr" {
		return 1
	}
	info, ok := types[typeName]
	if !ok || info == nil {
		return 0
	}
	if visiting[typeName] {
		return 0
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)

	switch info.Kind {
	case semantics.TypeStruct:
		count := 0
		for _, field := range info.Fields {
			count += ownedStorageSlotCountInType(field.TypeName, types, visiting)
			if count > 1 {
				return 2
			}
		}
		return count
	case semantics.TypeOptional:
		return ownedStorageSlotCountInType(info.ElemType, types, visiting)
	case semantics.TypeEnum:
		seenSlots := map[int]struct{}{}
		for _, enumCase := range info.EnumCases {
			payloadOffset := 1
			for i, payloadType := range enumCase.PayloadTypes {
				payloadSlots := 1
				if i < len(enumCase.PayloadSlots) {
					payloadSlots = enumCase.PayloadSlots[i]
				}
				payloadCount := ownedStorageSlotCountInType(payloadType, types, visiting)
				if payloadCount > 1 {
					return 2
				}
				if payloadCount == 1 {
					payloadOwnedSlot, ok := ownedReturnStorageSlotInType(payloadType, types, visiting)
					if ok && payloadOwnedSlot >= 0 && payloadOwnedSlot < payloadSlots {
						seenSlots[payloadOffset+payloadOwnedSlot] = struct{}{}
						if len(seenSlots) > 1 {
							return 2
						}
					}
				}
				payloadOffset += payloadSlots
			}
		}
		return len(seenSlots)
	default:
		return 0
	}
}

func optionalOwnedReturnTagSlot(typeName string, types map[string]*semantics.TypeInfo) int {
	info, ok := types[typeName]
	if !ok || info == nil || info.Kind != semantics.TypeOptional {
		return -1
	}
	elemSlot, ok := ownedReturnStorageSlot(info.ElemType, types)
	if !ok || elemSlot < 0 || elemSlot >= info.SlotCount-1 {
		return -1
	}
	return info.SlotCount - 1
}

func conditionalOwnedReturnGuardSlot(
	typeName string,
	returnSlot int,
	types map[string]*semantics.TypeInfo,
) (int, bool) {
	if tagSlot := optionalOwnedReturnTagSlot(typeName, types); tagSlot >= 0 {
		return tagSlot, true
	}
	info, ok := types[typeName]
	if !ok || info == nil || info.Kind != semantics.TypeEnum || returnSlot <= 0 || returnSlot >= info.SlotCount {
		return -1, false
	}
	return 0, true
}

func ownedThrowStorageSlot(typeName string, types map[string]*semantics.TypeInfo) (int, bool) {
	if slot, ok := ownedReturnStorageSlot(typeName, types); ok {
		return slot, true
	}
	info, ok := types[typeName]
	if !ok || info == nil || info.Kind != semantics.TypeOptional {
		return 0, false
	}
	elemSlot, ok := ownedReturnStorageSlot(info.ElemType, types)
	if !ok || elemSlot < 0 || elemSlot >= info.SlotCount-1 {
		return 0, false
	}
	return elemSlot, true
}

func ownedReturnStorageSlotInType(
	typeName string,
	types map[string]*semantics.TypeInfo,
	visiting map[string]bool,
) (int, bool) {
	if typeName == "ptr" {
		return 0, true
	}
	info, ok := types[typeName]
	if !ok || info == nil {
		return 0, false
	}
	if visiting[typeName] {
		return 0, false
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)

	if info.Kind == semantics.TypeEnum {
		ownedSlot := -1
		for _, enumCase := range info.EnumCases {
			payloadOffset := 1
			for i, payloadType := range enumCase.PayloadTypes {
				payloadSlots := 1
				if i < len(enumCase.PayloadSlots) {
					payloadSlots = enumCase.PayloadSlots[i]
				}
				payloadOwnedSlot, ok := ownedReturnStorageSlotInType(payloadType, types, visiting)
				if ok {
					if payloadOwnedSlot < 0 || payloadOwnedSlot >= payloadSlots {
						return 0, false
					}
					absoluteSlot := payloadOffset + payloadOwnedSlot
					if ownedSlot >= 0 && ownedSlot != absoluteSlot {
						return 0, false
					}
					ownedSlot = absoluteSlot
				}
				payloadOffset += payloadSlots
			}
		}
		if ownedSlot < 0 || ownedSlot >= info.SlotCount {
			return 0, false
		}
		return ownedSlot, true
	}
	if info.Kind == semantics.TypeOptional {
		elemSlot, ok := ownedReturnStorageSlotInType(info.ElemType, types, visiting)
		if !ok || elemSlot < 0 || elemSlot >= info.SlotCount-1 {
			return 0, false
		}
		return elemSlot, true
	}
	if info.Kind != semantics.TypeStruct {
		return 0, false
	}

	ownedSlot := -1
	for _, field := range info.Fields {
		fieldSlot, ok := ownedReturnStorageSlotInType(field.TypeName, types, visiting)
		if !ok {
			continue
		}
		if fieldSlot < 0 || fieldSlot >= field.SlotCount {
			continue
		}
		absoluteSlot := field.Offset + fieldSlot
		if ownedSlot >= 0 {
			return 0, false
		}
		ownedSlot = absoluteSlot
	}
	if ownedSlot < 0 || ownedSlot >= info.SlotCount {
		return 0, false
	}
	return ownedSlot, true
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
	ownedReturnSummaries := collectOwnedReturnSummaries(checked, Options{})
	ownedThrowSummaries := collectOwnedThrowSummaries(checked, Options{}, ownedReturnSummaries)
	for _, fn := range checked.Funcs {
		irFunc, err := lowerCheckedFuncWithOptions(
			fn,
			checked.Types,
			checked.FuncSigs,
			checked.GlobalsByModule[fn.Module],
			stagedTargets[fn.Name],
			callableTargets[fn.Name],
			ownedReturnSummaries,
			ownedThrowSummaries,
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
		nil,
		nil,
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
	ownedReturnSummaries map[string]ownedReturnSummary,
	ownedThrowSummaries map[string]ownedThrowSummary,
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
		ownedReturnSummaries:       ownedReturnSummaries,
		ownedThrowSummaries:        ownedThrowSummaries,
		allocationPlan:             allocationPlan,
		stackAllocationLowering:    opt.StackAllocationLowering,
		functionTempRegionLowering: opt.FunctionTempRegionLowering,
		ownedAllocDropLowering:     opt.OwnedAllocDropLowering,
		ownedLocalScopeDepth:       map[int]int{},
		movedOwnedLocals:           map[int]int{},
		nonOwnedEnumLocalTags:      map[int]int32{},
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
	for _, param := range fn.Decl.Params {
		if info, ok := l.locals[param.Name]; ok {
			l.rememberOwnedLocalScopeDepth(info)
			l.rememberOwnedAllocCleanupForConsumedParam(fn.Name, param, info)
		}
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
		OwnedParams: append([]ir.IROwnedParam(nil), l.ownedParams...),
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
	previousOwnedCleanupScopeDepth := l.ownedCleanupScopeDepth
	currentOwnedCleanupScopeDepth := previousOwnedCleanupScopeDepth + 1
	l.ownedCleanupScopeDepth = currentOwnedCleanupScopeDepth
	l.deferFrames = append(l.deferFrames, deferFrame{})
	for _, stmt := range stmts {
		if err := l.lowerStmt(stmt); err != nil {
			l.deferFrames = l.deferFrames[:frameIndex]
			l.forgetOwnedAllocCleanupsFromScope(currentOwnedCleanupScopeDepth)
			l.ownedCleanupScopeDepth = previousOwnedCleanupScopeDepth
			return err
		}
	}
	if err := l.emitDeferredFrame(frameIndex, pos); err != nil {
		l.deferFrames = l.deferFrames[:frameIndex]
		l.forgetOwnedAllocCleanupsFromScope(currentOwnedCleanupScopeDepth)
		l.ownedCleanupScopeDepth = previousOwnedCleanupScopeDepth
		return err
	}
	if l.blockMayFallthrough(stmts) {
		l.emitOwnedAllocCleanupFromScope(currentOwnedCleanupScopeDepth, pos)
	}
	l.forgetOwnedAllocCleanupsFromScope(currentOwnedCleanupScopeDepth)
	l.deferFrames = l.deferFrames[:frameIndex]
	l.ownedCleanupScopeDepth = previousOwnedCleanupScopeDepth
	return nil
}

func (l *lowerer) blockMayFallthrough(stmts []frontend.Stmt) bool {
	if len(stmts) == 0 {
		return true
	}
	return l.stmtMayFallthrough(stmts[len(stmts)-1])
}

func (l *lowerer) stmtMayFallthrough(stmt frontend.Stmt) bool {
	switch s := stmt.(type) {
	case *frontend.ReturnStmt, *frontend.ThrowStmt:
		return false
	case *frontend.IfStmt:
		if len(s.Else) == 0 {
			return true
		}
		return l.blockMayFallthrough(s.Then) || l.blockMayFallthrough(s.Else)
	case *frontend.IfLetStmt:
		if len(s.Else) == 0 {
			return true
		}
		return l.blockMayFallthrough(s.Then) || l.blockMayFallthrough(s.Else)
	case *frontend.MatchStmt:
		return l.matchStmtMayFallthrough(s)
	case *frontend.UnsafeStmt:
		return l.blockMayFallthrough(s.Body)
	default:
		return true
	}
}

func (l *lowerer) matchStmtMayFallthrough(s *frontend.MatchStmt) bool {
	for i := range s.Cases {
		c := &s.Cases[i]
		if l.blockMayFallthrough(c.Body) {
			return true
		}
	}
	return !l.matchStmtHasCompleteCoverage(s)
}

func (l *lowerer) matchStmtHasCompleteCoverage(s *frontend.MatchStmt) bool {
	if matchStmtHasUnguardedDefault(s) || l.matchStmtHasCompleteOptionalPatterns(s) {
		return true
	}
	return l.matchStmtHasCompleteEnumPatterns(s)
}

func matchStmtHasUnguardedDefault(s *frontend.MatchStmt) bool {
	for i := range s.Cases {
		c := &s.Cases[i]
		if c.Default && c.Guard == nil {
			return true
		}
	}
	return false
}

func (l *lowerer) matchStmtHasCompleteOptionalPatterns(s *frontend.MatchStmt) bool {
	scrutinee, ok := l.locals[s.ScrutineeLocal]
	if !ok {
		return false
	}
	info, ok := l.types[scrutinee.TypeName]
	if !ok || info.Kind != semantics.TypeOptional {
		return false
	}
	hasNone := false
	hasSome := false
	for i := range s.Cases {
		c := &s.Cases[i]
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		switch c.Pattern.(type) {
		case *frontend.NoneLitExpr:
			hasNone = true
		case *frontend.SomePatternExpr:
			hasSome = true
		default:
			return false
		}
	}
	return hasNone && hasSome
}

func (l *lowerer) matchStmtHasCompleteEnumPatterns(s *frontend.MatchStmt) bool {
	scrutinee, ok := l.locals[s.ScrutineeLocal]
	if !ok {
		return false
	}
	info, ok := l.types[scrutinee.TypeName]
	if !ok || info.Kind != semantics.TypeEnum || len(info.EnumCases) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(info.EnumCases))
	for i := range s.Cases {
		c := &s.Cases[i]
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		caseName := ""
		switch pat := c.Pattern.(type) {
		case *frontend.FieldAccessExpr:
			if pat.EnumType != scrutinee.TypeName {
				return false
			}
			caseName = pat.Field
		case *frontend.EnumCasePatternExpr:
			if pat.EnumType != scrutinee.TypeName {
				return false
			}
			caseName = pat.CaseName
		default:
			return false
		}
		caseInfo, ok := info.CaseMap[caseName]
		if !ok || len(caseInfo.PayloadTypes) != 0 {
			return false
		}
		seen[caseName] = struct{}{}
	}
	for _, enumCase := range info.EnumCases {
		if _, ok := seen[enumCase.Name]; !ok {
			return false
		}
	}
	return true
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
		returnedOwnedLocal, hasReturnedOwnedLocal, err := l.returnedOwnedAllocLocal(s.Value)
		if err != nil {
			return err
		}
		if err := l.rejectInlineOwnedReturnValue(s.Value, s.At); err != nil {
			return err
		}
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
			l.emitOwnedAllocCleanupExcept(s.At, returnedOwnedLocal, hasReturnedOwnedLocal)
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
			stagedSlots, stagedOwnedLocal, handled, err := l.lowerPartialOwnedExitExpr(s.Value, l.returnType)
			if err != nil {
				return err
			}
			if handled {
				slots = stagedSlots
				if stagedOwnedLocal >= 0 {
					returnedOwnedLocal = stagedOwnedLocal
					hasReturnedOwnedLocal = true
				}
			} else {
				slots, err = l.lowerExprAs(s.Value, l.returnType)
				if err != nil {
					return err
				}
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
		l.emitOwnedAllocCleanupExcept(s.At, returnedOwnedLocal, hasReturnedOwnedLocal)
		if l.throwsType == "" {
			l.emitInoutReturnSlots(s.At)
		}
		l.emitFunctionTempRegionReset(s.At)
		l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: s.At})
	case *frontend.ThrowStmt:
		if l.throwsType == "" {
			return fmt.Errorf("%s: throw is only allowed in throwing functions", frontend.FormatPos(s.At))
		}
		thrownOwnedLocal, hasThrownOwnedLocal, err := l.thrownOwnedAllocLocal(s.Value)
		if err != nil {
			return err
		}
		if err := l.rejectInlineOwnedThrowValue(s.Value, s.At); err != nil {
			return err
		}
		if l.stagedTaskTarget.SlotCount > 4 {
			slots := 0
			stagedSlots, stagedOwnedLocal, handled, err := l.lowerPartialOwnedExitExpr(s.Value, l.throwsType)
			if err != nil {
				return err
			}
			if handled {
				slots = stagedSlots
				if stagedOwnedLocal >= 0 {
					thrownOwnedLocal = stagedOwnedLocal
					hasThrownOwnedLocal = true
				}
			} else {
				slots, err = l.lowerExprAs(s.Value, l.throwsType)
				if err != nil {
					return err
				}
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
			l.emitOwnedAllocCleanupExcept(s.At, thrownOwnedLocal, hasThrownOwnedLocal)
			l.emitFunctionTempRegionReset(s.At)
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: s.At})
			l.emit(ir.IRInstr{Kind: ir.IRReturn, Pos: s.At})
			return nil
		}
		slots := 0
		if !l.throwCompact && exprContainsTryExpr(s.Value) {
			stagedSlots, stagedOwnedLocal, handled, err := l.lowerPartialOwnedExitExpr(s.Value, l.throwsType)
			if err != nil {
				return err
			}
			if handled {
				slots = stagedSlots
				if stagedOwnedLocal >= 0 {
					thrownOwnedLocal = stagedOwnedLocal
					hasThrownOwnedLocal = true
				}
			} else {
				slots, err = l.lowerExprAs(s.Value, l.throwsType)
				if err != nil {
					return err
				}
			}
			if slots != l.throwErrorSlots {
				return fmt.Errorf("%s: throw slot mismatch", frontend.FormatPos(s.At))
			}
			errBase := l.allocScratchSlots(l.throwErrorSlots)
			for slot := l.throwErrorSlots - 1; slot >= 0; slot-- {
				l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: errBase + slot, Pos: s.At})
			}
			l.emitZeroSlots(l.throwSuccessSlots, s.At)
			for slot := 0; slot < l.throwErrorSlots; slot++ {
				l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: errBase + slot, Pos: s.At})
			}
		} else {
			if !l.throwCompact {
				l.emitZeroSlots(l.throwSuccessSlots, s.At)
			}
			stagedSlots, stagedOwnedLocal, handled, err := l.lowerPartialOwnedExitExpr(s.Value, l.throwsType)
			if err != nil {
				return err
			}
			if handled {
				slots = stagedSlots
				if stagedOwnedLocal >= 0 {
					thrownOwnedLocal = stagedOwnedLocal
					hasThrownOwnedLocal = true
				}
			} else {
				slots, err = l.lowerExprAs(s.Value, l.throwsType)
				if err != nil {
					return err
				}
			}
		}
		if slots != l.throwErrorSlots {
			return fmt.Errorf("%s: throw slot mismatch", frontend.FormatPos(s.At))
		}
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 1, Pos: s.At})
		if err := l.emitDeferredFramesSince(0, s.At); err != nil {
			return err
		}
		l.emitCleanup(s.At)
		l.emitOwnedAllocCleanupExcept(s.At, thrownOwnedLocal, hasThrownOwnedLocal)
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
		l.emitOwnedAllocCleanupFromScope(loop.ownedScopeDepth, s.At)
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
		l.emitOwnedAllocCleanupFromScope(loop.ownedScopeDepth, s.At)
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
		if err := l.rejectLocalStagedOwnedEnumPayloadValue(s.Value, info.TypeName, s.At); err != nil {
			return err
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
		l.clearMovedOwnedLocalRange(info.Base, info.SlotCount)
		l.rememberOwnedLocalScopeDepth(info)
		l.rememberOwnedAllocCleanupForLet(s.Name, info, s.Value)
		l.rememberNonOwnedEnumLocalTag(info, s.Value)
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
			if err := l.rejectOwnedIndexStoreValue(s.Value, s.At); err != nil {
				return err
			}
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
				if err := l.rejectLocalStagedOwnedEnumPayloadValue(s.Value, g.TypeName, s.At); err != nil {
					return err
				}
				transferredOwnedLocal, hasTransferredOwnedLocal := l.ownedAllocCleanupLocalForExpr(s.Value)
				literalCleanup, literalDestSlot, hasLiteralCleanup := l.ownedAllocCleanupForStructLiteralField(
					s.Value,
					g.TypeName,
				)
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
				if hasTransferredOwnedLocal {
					l.markMovedOwnedLocal(transferredOwnedLocal, -1)
					l.forgetOwnedAllocCleanupLocal(transferredOwnedLocal)
				} else if hasLiteralCleanup && literalCleanup.local >= 0 &&
					literalDestSlot >= 0 && literalDestSlot < slotCount {
					l.markMovedOwnedLocal(literalCleanup.local, -1)
					l.forgetOwnedAllocCleanupLocal(literalCleanup.local)
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
		if err := l.rejectLocalStagedOwnedEnumPayloadValue(s.Value, target.TypeName, s.At); err != nil {
			return err
		}
		transferredCleanup, transferredTarget, hasTransferredCleanup := l.ownedAllocCleanupForExprWithTarget(s.Value)
		literalCleanup, literalDestSlot, hasLiteralCleanup := l.ownedAllocCleanupForStructLiteralField(
			s.Value,
			target.TypeName,
		)
		targetCleanup, hasTargetCleanup := l.ownedAllocCleanupForAssignedTarget(target)
		slots, err := l.lowerExprAs(s.Value, target.TypeName)
		if err != nil {
			return err
		}
		if slots != target.SlotCount {
			return fmt.Errorf("%s: slot mismatch for assignment", frontend.FormatPos(s.At))
		}
		if hasTargetCleanup && (!hasTransferredCleanup || transferredCleanup.local != targetCleanup.local) {
			l.emitOwnedAllocCleanupFor(targetCleanup, s.At)
			l.forgetOwnedAllocCleanupLocal(targetCleanup.local)
		}
		storeKind := ir.IRStoreLocal
		if target.Global {
			storeKind = ir.IRStoreGlobal
		}
		for i := target.SlotCount - 1; i >= 0; i-- {
			l.emit(ir.IRInstr{Kind: storeKind, Local: target.Base + i, Pos: s.At})
		}
		if target.Global && hasTransferredCleanup {
			if _, ok := ownedAllocCleanupTransferLocal(transferredCleanup, transferredTarget, target.Base, target.SlotCount); ok {
				l.markMovedOwnedLocal(transferredCleanup.local, -1)
				l.forgetOwnedAllocCleanupLocal(transferredCleanup.local)
			}
		} else if target.Global && hasLiteralCleanup && literalCleanup.local >= 0 &&
			literalDestSlot >= 0 && literalDestSlot < target.SlotCount {
			l.markMovedOwnedLocal(literalCleanup.local, -1)
			l.forgetOwnedAllocCleanupLocal(literalCleanup.local)
		} else if !target.Global && hasTransferredCleanup {
			if destLocal, ok := ownedAllocCleanupTransferLocal(transferredCleanup, transferredTarget, target.Base, target.SlotCount); ok {
				l.clearMovedOwnedLocalRange(target.Base, target.SlotCount)
				l.transferOwnedAllocCleanupToLocal(transferredCleanup, destLocal)
			}
		} else if !target.Global {
			l.clearMovedOwnedLocalRange(target.Base, target.SlotCount)
			l.rememberOwnedAllocCleanupForAssignedLocal(target, s.Value)
			l.rememberNonOwnedEnumLocalTag(
				semantics.LocalInfo{
					Base:      target.Base,
					SlotCount: target.SlotCount,
					TypeName:  target.TypeName,
				},
				s.Value,
			)
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
		ownedBranchState := l.snapshotOwnedAllocBranchState()
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
		thenOwnedState := l.snapshotOwnedAllocBranchState()
		elseState := branchState
		elseOwnedState := ownedBranchState
		if len(s.Else) > 0 {
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: s.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: elseLabel, Pos: s.At})
		if len(s.Else) > 0 {
			l.restoreRangeMetadata(branchState)
			l.restoreOwnedAllocBranchState(ownedBranchState)
			if err := l.lowerBlock(s.Else, s.At); err != nil {
				return err
			}
			elseState = l.snapshotRangeMetadata()
			elseOwnedState = l.snapshotOwnedAllocBranchState()
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		}
		l.mergeRangeMetadata(thenState, elseState)
		l.mergeOwnedAllocBranchState(thenOwnedState, elseOwnedState)
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
		if err := l.rejectLocalStagedOwnedEnumPayloadValue(s.Value, info.TypeName, s.At); err != nil {
			return err
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
		mergeOwnedCaseState := matchStmtOwnedReturnBranchMergeOK(s)
		ownedCaseBaseState := l.snapshotOwnedAllocBranchState()
		ownedCaseStates := []ownedAllocBranchState(nil)
		for i, c := range s.Cases {
			l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: caseLabels[i], Pos: c.At})
			if mergeOwnedCaseState {
				l.restoreOwnedAllocBranchState(ownedCaseBaseState)
			}
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
			if mergeOwnedCaseState {
				ownedCaseStates = append(ownedCaseStates, l.snapshotOwnedAllocBranchState())
			}
			l.emit(ir.IRInstr{Kind: ir.IRJmp, Label: endLabel, Pos: c.At})
		}
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: s.At})
		if mergeOwnedCaseState {
			l.mergeOwnedAllocCaseStates(ownedCaseStates)
		}
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
	ownedAllocDropLowering     bool
	ownedCleanupScopeDepth     int
	ownedAllocCleanups         []ownedAllocCleanup
	ownedLocalScopeDepth       map[int]int
	movedOwnedLocals           map[int]int
	nonOwnedEnumLocalTags      map[int]int32
	ownedReturnSummaries       map[string]ownedReturnSummary
	ownedThrowSummaries        map[string]ownedThrowSummary
	ownedParams                []ir.IROwnedParam
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

type ownedAllocCleanup struct {
	local                       int
	scopeDepth                  int
	conditionalLocal            int
	hasConditionalLocal         bool
	hasConditionalTagExactValue bool
	conditionalTagExactValue    int32
	layoutID                    string
	domain                      ir.IROwnershipDomain
	releaseKind                 ir.IRReleaseKind
}

type catchRelayLocalCleanupPlan struct {
	cleanups []ownedAllocCleanup
	selected []int
}

type ownedAllocBranchState struct {
	cleanups         []ownedAllocCleanup
	localScopeDepth  map[int]int
	movedLocals      map[int]int
	nonOwnedEnumTags map[int]int32
}

func cloneOwnedAllocCleanups(in []ownedAllocCleanup) []ownedAllocCleanup {
	out := make([]ownedAllocCleanup, len(in))
	copy(out, in)
	return out
}

func cloneLowerIntMap(in map[int]int) map[int]int {
	out := make(map[int]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneLowerInt32Map(in map[int]int32) map[int]int32 {
	out := make(map[int]int32, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (l *lowerer) snapshotOwnedAllocBranchState() ownedAllocBranchState {
	return ownedAllocBranchState{
		cleanups:         cloneOwnedAllocCleanups(l.ownedAllocCleanups),
		localScopeDepth:  cloneLowerIntMap(l.ownedLocalScopeDepth),
		movedLocals:      cloneLowerIntMap(l.movedOwnedLocals),
		nonOwnedEnumTags: cloneLowerInt32Map(l.nonOwnedEnumLocalTags),
	}
}

func (l *lowerer) restoreOwnedAllocBranchState(state ownedAllocBranchState) {
	l.ownedAllocCleanups = cloneOwnedAllocCleanups(state.cleanups)
	l.ownedLocalScopeDepth = cloneLowerIntMap(state.localScopeDepth)
	l.movedOwnedLocals = cloneLowerIntMap(state.movedLocals)
	l.nonOwnedEnumLocalTags = cloneLowerInt32Map(state.nonOwnedEnumTags)
}

func (l *lowerer) mergeOwnedAllocBranchState(thenState ownedAllocBranchState, elseState ownedAllocBranchState) {
	l.restoreOwnedAllocBranchState(l.commonOwnedAllocBranchState(thenState, elseState))
}

func (l *lowerer) mergeOwnedAllocCaseStates(states []ownedAllocBranchState) {
	if len(states) == 0 {
		return
	}
	merged := states[0]
	for _, state := range states[1:] {
		merged = l.commonOwnedAllocBranchState(merged, state)
	}
	l.restoreOwnedAllocBranchState(merged)
}

func (l *lowerer) commonOwnedAllocBranchState(a ownedAllocBranchState, b ownedAllocBranchState) ownedAllocBranchState {
	commonCleanups := commonOwnedAllocCleanups(a.cleanups, b.cleanups)
	commonCleanups = append(commonCleanups, l.conditionalOwnedAllocCleanups(a.cleanups, b.cleanups, b.nonOwnedEnumTags)...)
	commonCleanups = append(commonCleanups, l.conditionalOwnedAllocCleanups(b.cleanups, a.cleanups, a.nonOwnedEnumTags)...)
	return ownedAllocBranchState{
		cleanups:         commonCleanups,
		localScopeDepth:  commonLowerIntMap(a.localScopeDepth, b.localScopeDepth),
		movedLocals:      commonLowerIntMap(a.movedLocals, b.movedLocals),
		nonOwnedEnumTags: commonLowerInt32Map(a.nonOwnedEnumTags, b.nonOwnedEnumTags),
	}
}

func (l *lowerer) conditionalOwnedAllocCleanups(
	ownedBranch []ownedAllocCleanup,
	otherBranch []ownedAllocCleanup,
	otherNonOwnedTags map[int]int32,
) []ownedAllocCleanup {
	out := []ownedAllocCleanup(nil)
	for _, cleanup := range ownedBranch {
		if containsOwnedAllocCleanup(otherBranch, cleanup) {
			continue
		}
		conditionalCleanup, ok := l.conditionalOwnedAllocCleanupForNonOwnedBranch(cleanup, otherNonOwnedTags)
		if !ok {
			continue
		}
		out = append(out, conditionalCleanup)
	}
	return out
}

func containsOwnedAllocCleanup(cleanups []ownedAllocCleanup, target ownedAllocCleanup) bool {
	for _, cleanup := range cleanups {
		if cleanup == target {
			return true
		}
	}
	return false
}

func (l *lowerer) conditionalOwnedAllocCleanupForNonOwnedBranch(
	cleanup ownedAllocCleanup,
	otherNonOwnedTags map[int]int32,
) (ownedAllocCleanup, bool) {
	if cleanup.hasConditionalLocal {
		return ownedAllocCleanup{}, false
	}
	owner, ownedSlot, ok := l.enumLocalOwnerForOwnedSlot(cleanup.local)
	if !ok {
		return ownedAllocCleanup{}, false
	}
	tagSlot, ok := conditionalOwnedReturnGuardSlot(owner.TypeName, ownedSlot, l.types)
	if !ok {
		return ownedAllocCleanup{}, false
	}
	tagLocal := owner.Base + tagSlot
	nonOwnedTag, ok := otherNonOwnedTags[tagLocal]
	if !ok {
		return ownedAllocCleanup{}, false
	}
	ownedTag, ok := uniqueOwnedEnumTagValue(owner.TypeName, ownedSlot, l.types)
	if !ok || ownedTag == nonOwnedTag {
		return ownedAllocCleanup{}, false
	}
	cleanup.hasConditionalLocal = true
	cleanup.conditionalLocal = tagLocal
	cleanup.hasConditionalTagExactValue = true
	cleanup.conditionalTagExactValue = ownedTag
	return cleanup, true
}

func (l *lowerer) enumLocalOwnerForOwnedSlot(local int) (semantics.LocalInfo, int, bool) {
	if local < 0 {
		return semantics.LocalInfo{}, 0, false
	}
	for _, info := range l.locals {
		if local < info.Base || local >= info.Base+info.SlotCount {
			continue
		}
		ownedSlot, ok := ownedReturnStorageSlot(info.TypeName, l.types)
		if !ok || local-info.Base != ownedSlot {
			continue
		}
		if _, tagOK := uniqueOwnedEnumTagValue(info.TypeName, ownedSlot, l.types); !tagOK {
			continue
		}
		return info, ownedSlot, true
	}
	return semantics.LocalInfo{}, 0, false
}

func commonOwnedAllocCleanups(a []ownedAllocCleanup, b []ownedAllocCleanup) []ownedAllocCleanup {
	out := []ownedAllocCleanup(nil)
	used := make([]bool, len(b))
	for _, left := range a {
		for i, right := range b {
			if used[i] || left != right {
				continue
			}
			out = append(out, left)
			used[i] = true
			break
		}
	}
	return out
}

func commonLowerIntMap(a map[int]int, b map[int]int) map[int]int {
	out := map[int]int{}
	for key, left := range a {
		right, ok := b[key]
		if ok && right == left {
			out[key] = left
		}
	}
	return out
}

func commonLowerInt32Map(a map[int]int32, b map[int]int32) map[int]int32 {
	out := map[int]int32{}
	for key, left := range a {
		right, ok := b[key]
		if ok && right == left {
			out[key] = left
		}
	}
	return out
}

type ownedReturnSummary struct {
	returnSlot                  int
	conditional                 bool
	conditionalTagSlot          int
	hasConditionalTagExactValue bool
	conditionalTagExactValue    int32
	layoutID                    string
	domain                      ir.IROwnershipDomain
	releaseKind                 ir.IRReleaseKind
}

type ownedThrowSummary struct {
	errorSlot    int
	alwaysThrows bool
	layoutID     string
	domain       ir.IROwnershipDomain
	releaseKind  ir.IRReleaseKind
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
	continueLabel   int
	breakLabel      int
	cleanupDepth    int
	ownedScopeDepth int
	deferDepth      int
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

func (l *lowerer) emitOwnedAllocCleanup(pos frontend.Position) {
	l.emitOwnedAllocCleanupFromScope(1, pos)
}

func (l *lowerer) emitOwnedAllocCleanupExcept(
	pos frontend.Position,
	skipLocal int,
	hasSkip bool,
) {
	l.emitOwnedAllocCleanupFromScopeExcept(1, pos, skipLocal, hasSkip)
}

func (l *lowerer) emitOwnedAllocCleanupFromScope(minScopeDepth int, pos frontend.Position) {
	l.emitOwnedAllocCleanupFromScopeExcept(minScopeDepth, pos, -1, false)
}

func (l *lowerer) emitOwnedAllocCleanupFromScopeExcept(minScopeDepth int, pos frontend.Position, skipLocal int, hasSkip bool) {
	for i := len(l.ownedAllocCleanups) - 1; i >= 0; i-- {
		cleanup := l.ownedAllocCleanups[i]
		scopeDepth := cleanup.scopeDepth
		if scopeDepth == 0 {
			scopeDepth = 1
		}
		if scopeDepth < minScopeDepth {
			continue
		}
		if hasSkip && cleanup.local == skipLocal {
			continue
		}
		l.emitOwnedAllocCleanupFor(cleanup, pos)
	}
}

func (l *lowerer) emitOwnedAllocCleanupFor(cleanup ownedAllocCleanup, pos frontend.Position) {
	if cleanup.hasConditionalLocal {
		endLabel := l.newLabel()
		l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: cleanup.conditionalLocal, Pos: pos})
		if cleanup.hasConditionalTagExactValue {
			l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: cleanup.conditionalTagExactValue, Pos: pos})
			l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
		}
		l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: pos})
		l.emitOwnedAllocCleanupBody(cleanup, pos)
		l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: pos})
		return
	}
	l.emitOwnedAllocCleanupBody(cleanup, pos)
}

func (l *lowerer) emitOwnedAllocCleanupForEnumTag(
	cleanup ownedAllocCleanup,
	tagLocal int,
	tagValue int32,
	pos frontend.Position,
) {
	endLabel := l.newLabel()
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: tagLocal, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: tagValue, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
	l.emit(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: pos})
	l.emitOwnedAllocCleanupFor(cleanup, pos)
	l.emit(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: pos})
}

func (l *lowerer) emitOwnedAllocCleanupBody(cleanup ownedAllocCleanup, pos frontend.Position) {
	l.emit(ir.IRInstr{Kind: ir.IRLoadLocal, Local: cleanup.local, Pos: pos})
	l.emit(ir.IRInstr{
		Kind:            ir.IRDropOwned,
		LayoutID:        cleanup.layoutID,
		OwnershipDomain: cleanup.domain,
		ReleaseKind:     cleanup.releaseKind,
		Pos:             pos,
	})
	l.emit(ir.IRInstr{
		Kind:            ir.IRReleaseAllocation,
		LayoutID:        cleanup.layoutID,
		OwnershipDomain: cleanup.domain,
		ReleaseKind:     cleanup.releaseKind,
		Pos:             pos,
	})
}

func (l *lowerer) emitOwnedAllocCleanupRaw(pos frontend.Position) {
	for i := len(l.ownedAllocCleanups) - 1; i >= 0; i-- {
		cleanup := l.ownedAllocCleanups[i]
		l.emitOwnedAllocCleanupForRaw(cleanup, pos)
	}
}

func (l *lowerer) emitOwnedAllocCleanupForRaw(cleanup ownedAllocCleanup, pos frontend.Position) {
	if cleanup.hasConditionalLocal {
		endLabel := l.newLabel()
		l.emitRaw(ir.IRInstr{Kind: ir.IRLoadLocal, Local: cleanup.conditionalLocal, Pos: pos})
		if cleanup.hasConditionalTagExactValue {
			l.emitRaw(ir.IRInstr{Kind: ir.IRConstI32, Imm: cleanup.conditionalTagExactValue, Pos: pos})
			l.emitRaw(ir.IRInstr{Kind: ir.IRCmpEqI32, Pos: pos})
		}
		l.emitRaw(ir.IRInstr{Kind: ir.IRJmpIfZero, Label: endLabel, Pos: pos})
		l.emitOwnedAllocCleanupBodyRaw(cleanup, pos)
		l.emitRaw(ir.IRInstr{Kind: ir.IRLabel, Label: endLabel, Pos: pos})
		return
	}
	l.emitOwnedAllocCleanupBodyRaw(cleanup, pos)
}

func (l *lowerer) emitOwnedAllocCleanupBodyRaw(cleanup ownedAllocCleanup, pos frontend.Position) {
	l.emitRaw(ir.IRInstr{Kind: ir.IRLoadLocal, Local: cleanup.local, Pos: pos})
	l.emitRaw(ir.IRInstr{
		Kind:            ir.IRDropOwned,
		LayoutID:        cleanup.layoutID,
		OwnershipDomain: cleanup.domain,
		ReleaseKind:     cleanup.releaseKind,
		Pos:             pos,
	})
	l.emitRaw(ir.IRInstr{
		Kind:            ir.IRReleaseAllocation,
		LayoutID:        cleanup.layoutID,
		OwnershipDomain: cleanup.domain,
		ReleaseKind:     cleanup.releaseKind,
		Pos:             pos,
	})
}

func (l *lowerer) rememberOwnedLocalScopeDepth(info semantics.LocalInfo) {
	if !l.ownedAllocDropLowering {
		return
	}
	for slot := info.Base; slot < info.Base+info.SlotCount; slot++ {
		l.ownedLocalScopeDepth[slot] = l.currentOwnedCleanupScopeDepth()
	}
}

func (l *lowerer) rememberNonOwnedEnumLocalTag(info semantics.LocalInfo, value frontend.Expr) {
	if !l.ownedAllocDropLowering || !ownedReturnStorageType(info.TypeName, l.types) {
		return
	}
	ownedSlot, ok := ownedReturnStorageSlot(info.TypeName, l.types)
	if !ok {
		return
	}
	tagSlot, ok := conditionalOwnedReturnGuardSlot(info.TypeName, ownedSlot, l.types)
	if !ok || tagSlot < 0 || tagSlot >= info.SlotCount {
		return
	}
	tagLocal := info.Base + tagSlot
	nonOwnedTag, ok := enumConstructorNonOwnedTag(value, info.TypeName, l.types)
	if ok {
		l.nonOwnedEnumLocalTags[tagLocal] = nonOwnedTag
		return
	}
	delete(l.nonOwnedEnumLocalTags, tagLocal)
}

func (l *lowerer) currentOwnedCleanupScopeDepth() int {
	if l.ownedCleanupScopeDepth > 0 {
		return l.ownedCleanupScopeDepth
	}
	return 1
}

func (l *lowerer) ownedLocalScopeDepthForLocal(local int) int {
	if scopeDepth, ok := l.ownedLocalScopeDepth[local]; ok && scopeDepth > 0 {
		return scopeDepth
	}
	return l.currentOwnedCleanupScopeDepth()
}

func (l *lowerer) returnedOwnedAllocLocal(expr frontend.Expr) (int, bool, error) {
	return l.ownedAllocCleanupLocalForExitExpr(expr, l.returnType)
}

func (l *lowerer) thrownOwnedAllocLocal(expr frontend.Expr) (int, bool, error) {
	return l.ownedAllocCleanupLocalForExitExpr(expr, l.throwsType)
}

func (l *lowerer) ownedAllocCleanupLocalForExitExpr(expr frontend.Expr, destType string) (int, bool, error) {
	if cleanup, _, ok := l.ownedAllocCleanupForCatchRelayResult(expr); ok && cleanup.local >= 0 {
		return cleanup.local, true, nil
	}
	if cleanup, _, ok := l.ownedAllocCleanupForNonOwnedErrorCatchResult(expr); ok && cleanup.local >= 0 {
		return cleanup.local, true, nil
	}
	if cleanup, _, ok := l.ownedAllocCleanupForOwnedErrorMixedCatchResult(expr); ok && cleanup.local >= 0 {
		return cleanup.local, true, nil
	}
	if cleanup, _, ok := l.ownedAllocCleanupForStructLiteralField(expr, destType); ok && cleanup.local >= 0 {
		return cleanup.local, true, nil
	}
	if cleanup, _, ok := l.ownedAllocCleanupForEnumConstructorPayload(expr, destType); ok && cleanup.local >= 0 {
		return cleanup.local, true, nil
	}
	target, ok := l.ownedAllocCleanupTargetForExpr(expr)
	if !ok {
		return -1, false, nil
	}
	ownedLocal := -1
	ownedCount := 0
	for _, cleanup := range l.ownedAllocCleanups {
		if cleanup.local >= target.Base && cleanup.local < target.Base+target.SlotCount {
			if ownedLocal < 0 {
				ownedLocal = cleanup.local
			}
			ownedCount++
		}
	}
	if ownedCount > 1 {
		name := target.Name
		if name == "" {
			name = fmt.Sprintf("local%d", target.Base)
		}
		return -1, false, fmt.Errorf(
			"%s: returning aggregate '%s' with multiple owned return slots requires typed ownership summaries",
			frontend.FormatPos(expr.Pos()),
			name,
		)
	}
	if ownedLocal < 0 {
		return -1, false, nil
	}
	return ownedLocal, true, nil
}

func (l *lowerer) ownedAllocCleanupForCatchRelayResult(
	expr frontend.Expr,
) (ownedAllocCleanup, int, bool) {
	catchExpr, ok := expr.(*frontend.CatchExpr)
	if !ok || catchExpr == nil {
		return ownedAllocCleanup{}, 0, false
	}
	call, ok := catchExpr.Call.(*frontend.CallExpr)
	if !ok || call == nil {
		return ownedAllocCleanup{}, 0, false
	}
	summary, ok := l.ownedThrowSummaryForCallExpr(call)
	if !ok {
		return ownedAllocCleanup{}, 0, false
	}
	resultInfo, ok := l.locals[catchExpr.ResultLocal]
	if !ok {
		return ownedAllocCleanup{}, 0, false
	}
	resultSlot, ok := ownedReturnStorageSlot(catchExpr.ResultType, l.types)
	if !ok || resultSlot < 0 || resultSlot >= resultInfo.SlotCount {
		return ownedAllocCleanup{}, 0, false
	}
	callForReturn := lowerCallExprWithBuiltinAlias(call)
	successSummary, hasOwnedSuccess := l.ownedReturnSummaryForCallName(callForReturn.Name)
	if hasOwnedSuccess &&
		(successSummary.returnSlot < 0 || successSummary.returnSlot != resultSlot) {
		return ownedAllocCleanup{}, 0, false
	}
	if !summary.alwaysThrows && !hasOwnedSuccess {
		return ownedAllocCleanup{}, 0, false
	}
	if !catchExprRelaysOwnedError(
		catchExpr,
		catchExpr.ErrorType,
		catchExpr.ResultType,
		summary.errorSlot,
		l.types,
		func(value frontend.Expr, resultSlot int) bool {
			return l.catchCaseValueProducesOwnedResult(value, catchExpr.ResultType, resultSlot)
		},
	) {
		return ownedAllocCleanup{}, 0, false
	}
	cleanup := ownedAllocCleanup{
		local:       resultInfo.Base + resultSlot,
		scopeDepth:  l.ownedLocalScopeDepthForLocal(resultInfo.Base + resultSlot),
		layoutID:    summary.layoutID,
		domain:      summary.domain,
		releaseKind: summary.releaseKind,
	}
	if sourceCleanup, ok := l.catchRelayLocalCleanup(catchExpr, resultSlot); ok && summary.alwaysThrows {
		cleanup.layoutID = sourceCleanup.layoutID
		cleanup.domain = sourceCleanup.domain
		cleanup.releaseKind = sourceCleanup.releaseKind
	}
	return cleanup, resultSlot, true
}

func (l *lowerer) catchCaseValueProducesOwnedResult(
	value frontend.Expr,
	resultType string,
	resultSlot int,
) bool {
	if cleanup, target, ok := l.ownedAllocCleanupForExprWithTarget(value); ok {
		sourceSlot := cleanup.local - target.Base
		if sourceSlot == resultSlot {
			return true
		}
	}
	return catchCaseValueProducesOwnedResult(
		value,
		resultType,
		resultSlot,
		l.funcs,
		l.types,
		l.ownedReturnSummaries,
		nil,
	)
}

func (l *lowerer) catchRelayLocalCleanup(
	catchExpr *frontend.CatchExpr,
	resultSlot int,
) (ownedAllocCleanup, bool) {
	plan, ok := l.catchRelayLocalCleanupPlan(catchExpr, resultSlot)
	if !ok || len(plan.cleanups) != 1 {
		return ownedAllocCleanup{}, false
	}
	return plan.cleanups[0], true
}

func (l *lowerer) catchRelayLocalCleanupPlan(
	catchExpr *frontend.CatchExpr,
	resultSlot int,
) (catchRelayLocalCleanupPlan, bool) {
	if catchExpr == nil || len(catchExpr.Cases) == 0 {
		return catchRelayLocalCleanupPlan{}, false
	}
	plan := catchRelayLocalCleanupPlan{
		selected: make([]int, len(catchExpr.Cases)),
	}
	cleanupByLocal := map[int]int{}
	var relayDomain ir.IROwnershipDomain
	var relayReleaseKind ir.IRReleaseKind
	hasRelayMetadata := false
	for i, c := range catchExpr.Cases {
		if c.Default && c.Guard != nil {
			return catchRelayLocalCleanupPlan{}, false
		}
		cleanup, target, ok := l.ownedAllocCleanupForExprWithTarget(c.Value)
		if !ok {
			return catchRelayLocalCleanupPlan{}, false
		}
		sourceSlot := cleanup.local - target.Base
		if sourceSlot != resultSlot {
			return catchRelayLocalCleanupPlan{}, false
		}
		if !hasRelayMetadata {
			relayDomain = cleanup.domain
			relayReleaseKind = cleanup.releaseKind
			hasRelayMetadata = true
		} else if cleanup.domain != relayDomain || cleanup.releaseKind != relayReleaseKind {
			return catchRelayLocalCleanupPlan{}, false
		}
		cleanupIndex, ok := cleanupByLocal[cleanup.local]
		if ok {
			existing := plan.cleanups[cleanupIndex]
			if cleanup.layoutID != existing.layoutID ||
				cleanup.domain != existing.domain ||
				cleanup.releaseKind != existing.releaseKind {
				return catchRelayLocalCleanupPlan{}, false
			}
		} else {
			cleanupIndex = len(plan.cleanups)
			cleanupByLocal[cleanup.local] = cleanupIndex
			plan.cleanups = append(plan.cleanups, cleanup)
		}
		plan.selected[i] = cleanupIndex
	}
	return plan, len(plan.cleanups) > 0
}

func (l *lowerer) emitCatchRelayUnselectedLocalCleanups(
	plan catchRelayLocalCleanupPlan,
	caseIndex int,
	pos frontend.Position,
) {
	if caseIndex < 0 || caseIndex >= len(plan.selected) {
		return
	}
	selected := plan.selected[caseIndex]
	for i, cleanup := range plan.cleanups {
		if i == selected {
			continue
		}
		l.emitOwnedAllocCleanupFor(cleanup, pos)
	}
}

func (l *lowerer) emitCatchRelayAllLocalCleanups(plan catchRelayLocalCleanupPlan, pos frontend.Position) {
	for _, cleanup := range plan.cleanups {
		l.emitOwnedAllocCleanupFor(cleanup, pos)
	}
}

func (l *lowerer) forgetCatchRelayLocalCleanups(plan catchRelayLocalCleanupPlan) {
	for _, cleanup := range plan.cleanups {
		l.forgetOwnedAllocCleanupLocal(cleanup.local)
	}
}

func (l *lowerer) ownedAllocCleanupForNonOwnedErrorCatchResult(
	expr frontend.Expr,
) (ownedAllocCleanup, int, bool) {
	catchExpr, ok := expr.(*frontend.CatchExpr)
	if !ok || catchExpr == nil {
		return ownedAllocCleanup{}, 0, false
	}
	source, resultSlot, condition, ok := catchExprNonOwnedErrorOwnedResultSourceInfo(
		expr,
		catchExpr.ResultType,
		l.funcs,
		l.types,
		l.ownedReturnSummaries,
	)
	if !ok {
		return ownedAllocCleanup{}, 0, false
	}
	resultInfo, ok := l.locals[catchExpr.ResultLocal]
	if !ok {
		return ownedAllocCleanup{}, 0, false
	}
	if resultSlot < 0 || resultSlot >= resultInfo.SlotCount {
		return ownedAllocCleanup{}, 0, false
	}
	cleanup := ownedAllocCleanup{
		local:       resultInfo.Base + resultSlot,
		scopeDepth:  l.ownedLocalScopeDepthForLocal(resultInfo.Base + resultSlot),
		layoutID:    "layout:" + source,
		domain:      ir.IROwnershipDomainHeap,
		releaseKind: ir.IRReleaseKindLinuxMmap,
	}
	if condition.conditional {
		conditionalTagSlot, guardOK := conditionalOwnedReturnGuardSlot(catchExpr.ResultType, resultSlot, l.types)
		if !guardOK ||
			conditionalTagSlot < 0 ||
			conditionalTagSlot >= resultInfo.SlotCount ||
			conditionalTagSlot == resultSlot {
			return ownedAllocCleanup{}, 0, false
		}
		cleanup.conditionalLocal = resultInfo.Base + conditionalTagSlot
		cleanup.hasConditionalLocal = true
		cleanup.hasConditionalTagExactValue = condition.hasExactTagValue
		cleanup.conditionalTagExactValue = condition.exactTagValue
	}
	return cleanup, resultSlot, true
}

func (l *lowerer) ownedAllocCleanupForOwnedErrorMixedCatchResult(
	expr frontend.Expr,
) (ownedAllocCleanup, int, bool) {
	catchExpr, ok := expr.(*frontend.CatchExpr)
	if !ok || catchExpr == nil {
		return ownedAllocCleanup{}, 0, false
	}
	_, resultSlot, condition, ok := catchExprOwnedErrorMixedResultSourceInfo(
		expr,
		catchExpr.ResultType,
		l.funcs,
		l.types,
		l.ownedReturnSummaries,
		l.ownedThrowSummaries,
	)
	if !ok {
		return ownedAllocCleanup{}, 0, false
	}
	resultInfo, ok := l.locals[catchExpr.ResultLocal]
	if !ok {
		return ownedAllocCleanup{}, 0, false
	}
	if resultSlot < 0 || resultSlot >= resultInfo.SlotCount {
		return ownedAllocCleanup{}, 0, false
	}
	call, ok := catchExpr.Call.(*frontend.CallExpr)
	if !ok || call == nil {
		return ownedAllocCleanup{}, 0, false
	}
	call = lowerCallExprWithBuiltinAlias(call)
	summary, ok := l.ownedReturnSummaryForCallName(call.Name)
	if !ok ||
		summary.returnSlot < 0 ||
		summary.returnSlot != resultSlot {
		return ownedAllocCleanup{}, 0, false
	}
	cleanup := ownedAllocCleanup{
		local:       resultInfo.Base + resultSlot,
		scopeDepth:  l.ownedLocalScopeDepthForLocal(resultInfo.Base + resultSlot),
		layoutID:    summary.layoutID,
		domain:      summary.domain,
		releaseKind: summary.releaseKind,
	}
	if condition.conditional {
		conditionalTagSlot, guardOK := conditionalOwnedReturnGuardSlot(catchExpr.ResultType, resultSlot, l.types)
		if !guardOK ||
			conditionalTagSlot < 0 ||
			conditionalTagSlot >= resultInfo.SlotCount ||
			conditionalTagSlot == resultSlot {
			return ownedAllocCleanup{}, 0, false
		}
		cleanup.conditionalLocal = resultInfo.Base + conditionalTagSlot
		cleanup.hasConditionalLocal = true
		cleanup.hasConditionalTagExactValue = condition.hasExactTagValue
		cleanup.conditionalTagExactValue = condition.exactTagValue
	}
	return cleanup, resultSlot, true
}

func (l *lowerer) ownedAllocCleanupLocalForExpr(expr frontend.Expr) (int, bool) {
	target, ok := l.ownedAllocCleanupTargetForExpr(expr)
	if !ok {
		return -1, false
	}
	for _, cleanup := range l.ownedAllocCleanups {
		if cleanup.local >= target.Base && cleanup.local < target.Base+target.SlotCount {
			return cleanup.local, true
		}
	}
	return -1, false
}

func (l *lowerer) ownedAllocCleanupTargetForExpr(expr frontend.Expr) (lvalueInfo, bool) {
	if _, _, _, ok := splitFieldPathLower(expr); !ok {
		return lvalueInfo{}, false
	}
	target, err := l.resolveLValue(expr)
	if err != nil || target.Global || target.SlotCount <= 0 {
		return lvalueInfo{}, false
	}
	return target, true
}

func (l *lowerer) ownedAllocCleanupForAssignedTarget(target lvalueInfo) (ownedAllocCleanup, bool) {
	if target.Global || target.SlotCount <= 0 {
		return ownedAllocCleanup{}, false
	}
	for _, cleanup := range l.ownedAllocCleanups {
		if cleanup.local >= target.Base && cleanup.local < target.Base+target.SlotCount {
			return cleanup, true
		}
	}
	return ownedAllocCleanup{}, false
}

func (l *lowerer) forgetOwnedAllocCleanupLocal(local int) {
	out := l.ownedAllocCleanups[:0]
	for _, cleanup := range l.ownedAllocCleanups {
		if cleanup.local != local {
			out = append(out, cleanup)
		}
	}
	l.ownedAllocCleanups = out
}

func (l *lowerer) consumedOwnedAllocLocals(
	args []frontend.Expr,
	paramOwnership []string,
) []int {
	if !l.ownedAllocDropLowering {
		return nil
	}
	locals := []int(nil)
	for i, arg := range args {
		if i >= len(paramOwnership) || paramOwnership[i] != "consume" {
			continue
		}
		local, ok := l.ownedAllocCleanupLocalForExpr(arg)
		if !ok {
			continue
		}
		locals = append(locals, local)
	}
	return locals
}

func (l *lowerer) forgetOwnedAllocCleanupsFromScope(minScopeDepth int) {
	out := l.ownedAllocCleanups[:0]
	for _, cleanup := range l.ownedAllocCleanups {
		scopeDepth := cleanup.scopeDepth
		if scopeDepth == 0 {
			scopeDepth = 1
		}
		if scopeDepth < minScopeDepth {
			out = append(out, cleanup)
		}
	}
	l.ownedAllocCleanups = out
}

func (l *lowerer) transferOwnedAllocCleanupToLocal(cleanup ownedAllocCleanup, local int) {
	sourceLocal := cleanup.local
	conditionalDelta := cleanup.conditionalLocal - sourceLocal
	l.forgetOwnedAllocCleanupLocal(sourceLocal)
	if sourceLocal != local {
		l.forgetOwnedAllocCleanupLocal(local)
		l.markMovedOwnedLocal(sourceLocal, local)
	}
	l.clearMovedOwnedLocal(local)
	cleanup.local = local
	if cleanup.hasConditionalLocal {
		cleanup.conditionalLocal = local + conditionalDelta
	}
	cleanup.scopeDepth = l.ownedLocalScopeDepthForLocal(local)
	l.ownedAllocCleanups = append(l.ownedAllocCleanups, cleanup)
}

func (l *lowerer) rememberOwnedAllocCleanupAtLocal(
	cleanup ownedAllocCleanup,
	local int,
	conditionalLocal int,
	condition ownedReturnCondition,
) {
	l.forgetOwnedAllocCleanupLocal(local)
	l.clearMovedOwnedLocal(local)
	cleanup.local = local
	cleanup.scopeDepth = l.ownedLocalScopeDepthForLocal(local)
	cleanup.hasConditionalLocal = condition.conditional
	cleanup.hasConditionalTagExactValue = condition.hasExactTagValue
	cleanup.conditionalTagExactValue = condition.exactTagValue
	if condition.conditional {
		cleanup.conditionalLocal = conditionalLocal
	}
	l.ownedAllocCleanups = append(l.ownedAllocCleanups, cleanup)
}

func (l *lowerer) markMovedOwnedLocal(sourceLocal int, destLocal int) {
	if !l.ownedAllocDropLowering || sourceLocal < 0 {
		return
	}
	l.movedOwnedLocals[sourceLocal] = destLocal
}

func (l *lowerer) clearMovedOwnedLocal(local int) {
	if !l.ownedAllocDropLowering || local < 0 {
		return
	}
	delete(l.movedOwnedLocals, local)
}

func (l *lowerer) clearMovedOwnedLocalRange(base int, slotCount int) {
	if !l.ownedAllocDropLowering || base < 0 {
		return
	}
	if slotCount <= 0 {
		slotCount = 1
	}
	for slot := base; slot < base+slotCount; slot++ {
		delete(l.movedOwnedLocals, slot)
	}
}

func (l *lowerer) rejectMovedOwnedLocalUse(target lvalueInfo, pos frontend.Position) error {
	if !l.ownedAllocDropLowering || target.Global || len(l.movedOwnedLocals) == 0 {
		return nil
	}
	slotCount := target.SlotCount
	if slotCount <= 0 {
		slotCount = 1
	}
	movedSlot := -1
	destLocal := -1
	for slot := target.Base; slot < target.Base+slotCount; slot++ {
		dest, moved := l.movedOwnedLocals[slot]
		if !moved {
			continue
		}
		movedSlot = slot
		destLocal = dest
		break
	}
	if movedSlot < 0 {
		return nil
	}
	name := target.Name
	if name == "" {
		name = fmt.Sprintf("local%d", target.Base)
	}
	if destLocal >= 0 {
		return fmt.Errorf(
			"%s: use after move of owned local '%s' to local%d",
			frontend.FormatPos(pos),
			name,
			destLocal,
		)
	}
	if movedSlot != target.Base {
		return fmt.Errorf(
			"%s: use after move of owned local '%s' slot %d",
			frontend.FormatPos(pos),
			name,
			movedSlot,
		)
	}
	return fmt.Errorf("%s: use after move of owned local '%s'", frontend.FormatPos(pos), name)
}

func (l *lowerer) rejectOwnedIndexStoreValue(value frontend.Expr, pos frontend.Position) error {
	if !l.ownedAllocDropLowering || !l.exprContainsTrackedOwnedValue(value) {
		return nil
	}
	return fmt.Errorf(
		"%s: index store of owned value requires typed ownership summaries",
		frontend.FormatPos(pos),
	)
}

func (l *lowerer) rejectInlineOwnedReturnValue(value frontend.Expr, pos frontend.Position) error {
	if !l.ownedAllocDropLowering {
		return nil
	}
	if l.exprHasMultipleTrackedOwnedEnumPayloadSlots(value, l.returnType) {
		return fmt.Errorf(
			"%s: multiple owned enum payload slots require typed ownership summaries",
			frontend.FormatPos(pos),
		)
	}
	if !l.exprIsUnsupportedInlineOwnedAggregateExit(value) {
		return nil
	}
	if _, _, ok := l.ownedAllocCleanupForStructLiteralField(value, l.returnType); ok {
		return nil
	}
	return fmt.Errorf(
		"%s: inline owned return requires typed ownership summaries",
		frontend.FormatPos(pos),
	)
}

func (l *lowerer) rejectInlineOwnedThrowValue(value frontend.Expr, pos frontend.Position) error {
	if !l.ownedAllocDropLowering {
		return nil
	}
	if l.exprHasMultipleTrackedOwnedEnumPayloadSlots(value, l.throwsType) {
		return fmt.Errorf(
			"%s: multiple owned enum payload slots require typed ownership summaries",
			frontend.FormatPos(pos),
		)
	}
	if !l.exprIsUnsupportedInlineOwnedAggregateExit(value) {
		return nil
	}
	if _, _, ok := l.ownedAllocCleanupForStructLiteralField(value, l.throwsType); ok {
		return nil
	}
	return fmt.Errorf(
		"%s: inline owned throw requires typed ownership summaries",
		frontend.FormatPos(pos),
	)
}

func (l *lowerer) rejectLocalStagedOwnedEnumPayloadValue(
	value frontend.Expr,
	destType string,
	pos frontend.Position,
) error {
	if !l.ownedAllocDropLowering || !l.exprHasMultipleTrackedOwnedEnumPayloadSlots(value, destType) {
		return nil
	}
	return fmt.Errorf(
		"%s: multiple owned enum payload slots require typed ownership summaries",
		frontend.FormatPos(pos),
	)
}

func (l *lowerer) exprHasMultipleTrackedOwnedEnumPayloadSlots(expr frontend.Expr, destType string) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	typeName, caseInfo, ok := enumCaseConstructorInfo(call, destType, l.types)
	if !ok || typeName != destType {
		return false
	}
	staticOwnedCount := enumCaseOwnedPayloadSlotCount(caseInfo, l.types)
	ownedCount := 0
	for i, arg := range call.Args {
		if i >= len(caseInfo.PayloadTypes) || i >= len(caseInfo.PayloadSlots) {
			return false
		}
		ownedCount += l.trackedOwnedSlotCountForExpr(
			arg,
			caseInfo.PayloadTypes[i],
			caseInfo.PayloadSlots[i],
		)
		if ownedCount > 1 {
			return true
		}
	}
	return ownedCount > 0 && staticOwnedCount > 1
}

func enumCaseOwnedPayloadSlotCount(
	caseInfo semantics.EnumCaseInfo,
	types map[string]*semantics.TypeInfo,
) int {
	count := 0
	for _, payloadType := range caseInfo.PayloadTypes {
		count += ownedStorageSlotCount(payloadType, types)
		if count > 1 {
			return 2
		}
	}
	return count
}

func (l *lowerer) trackedOwnedSlotCountForExpr(expr frontend.Expr, typeName string, slotCount int) int {
	if expr == nil || slotCount <= 0 {
		return 0
	}
	if target, ok := l.ownedAllocCleanupTargetForExpr(expr); ok {
		count := 0
		for _, cleanup := range l.ownedAllocCleanups {
			if cleanup.local >= target.Base && cleanup.local < target.Base+target.SlotCount {
				count++
			}
		}
		return count
	}
	if typeName == "ptr" && isDirectAllocBytesCall(expr) {
		return 1
	}
	if summary, ok := l.ownedReturnSummaryForCallExpr(expr); ok {
		if summary.returnSlot >= 0 && summary.returnSlot < slotCount {
			return 1
		}
		return 0
	}
	if call, ok := expr.(*frontend.CallExpr); ok && call != nil {
		_, caseInfo, ok := enumCaseConstructorInfo(call, typeName, l.types)
		if !ok {
			return 0
		}
		total := 0
		for i, arg := range call.Args {
			if i >= len(caseInfo.PayloadTypes) || i >= len(caseInfo.PayloadSlots) {
				return 0
			}
			total += l.trackedOwnedSlotCountForExpr(arg, caseInfo.PayloadTypes[i], caseInfo.PayloadSlots[i])
		}
		return total
	}
	lit, ok := expr.(*frontend.StructLitExpr)
	if !ok || lit == nil {
		return 0
	}
	info, ok := l.types[typeName]
	if !ok || info == nil || info.Kind != semantics.TypeStruct {
		return 0
	}
	fieldValues := make(map[string]frontend.Expr, len(lit.Fields))
	for _, field := range lit.Fields {
		fieldValues[field.Name] = field.Value
	}
	total := 0
	for _, field := range info.Fields {
		value, ok := fieldValues[field.Name]
		if !ok {
			continue
		}
		total += l.trackedOwnedSlotCountForExpr(value, field.TypeName, field.SlotCount)
	}
	return total
}

func (l *lowerer) exprIsUnsupportedInlineOwnedAggregateExit(expr frontend.Expr) bool {
	if expr == nil {
		return false
	}
	lit, ok := expr.(*frontend.StructLitExpr)
	if !ok {
		return false
	}
	for _, field := range lit.Fields {
		if l.exprContainsTrackedOwnedValue(field.Value) {
			return true
		}
	}
	return false
}

func (l *lowerer) exprContainsTrackedOwnedValue(expr frontend.Expr) bool {
	if expr == nil {
		return false
	}
	if _, _, ok := l.ownedAllocCleanupForExprWithTarget(expr); ok {
		return true
	}
	if cleanup, _, ok := l.ownedAllocCleanupForCatchRelayResult(expr); ok && cleanup.local >= 0 {
		return true
	}
	if catchExpr, ok := expr.(*frontend.CatchExpr); ok && catchExpr != nil &&
		ownedReturnStorageType(catchExpr.ResultType, l.types) {
		if call, ok := catchExpr.Call.(*frontend.CallExpr); ok {
			if _, ok := l.ownedThrowSummaryForCallExpr(call); ok {
				return true
			}
		}
	}
	if _, ok := l.ownedReturnSummaryForCallExpr(expr); ok {
		return true
	}
	if isDirectAllocBytesCall(expr) {
		return true
	}
	switch e := expr.(type) {
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if l.exprContainsTrackedOwnedValue(field.Value) {
				return true
			}
		}
	}
	return false
}

func exprContainsTryExpr(expr frontend.Expr) bool {
	if expr == nil {
		return false
	}
	switch e := expr.(type) {
	case *frontend.TryExpr:
		return true
	case *frontend.AwaitExpr:
		return exprContainsTryExpr(e.X)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			if exprContainsTryExpr(arg) {
				return true
			}
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if exprContainsTryExpr(field.Value) {
				return true
			}
		}
	case *frontend.BinaryExpr:
		return exprContainsTryExpr(e.Left) || exprContainsTryExpr(e.Right)
	case *frontend.UnaryExpr:
		return exprContainsTryExpr(e.X)
	case *frontend.FieldAccessExpr:
		return exprContainsTryExpr(e.Base)
	case *frontend.IndexExpr:
		return exprContainsTryExpr(e.Base) || exprContainsTryExpr(e.Index)
	case *frontend.MatchExpr:
		if exprContainsTryExpr(e.Value) {
			return true
		}
		for _, c := range e.Cases {
			if exprContainsTryExpr(c.Pattern) || exprContainsTryExpr(c.Guard) ||
				exprContainsTryExpr(c.Value) {
				return true
			}
		}
	case *frontend.CatchExpr:
		if exprContainsTryExpr(e.Call) {
			return true
		}
		for _, c := range e.Cases {
			if exprContainsTryExpr(c.Pattern) || exprContainsTryExpr(c.Guard) ||
				exprContainsTryExpr(c.Value) {
				return true
			}
		}
	}
	return false
}

func (l *lowerer) rememberOwnedAllocCleanupForLet(
	name string,
	info semantics.LocalInfo,
	value frontend.Expr,
) {
	if !l.ownedAllocDropLowering || !ownedReturnStorageType(info.TypeName, l.types) {
		return
	}
	if sourceCleanup, sourceTarget, ok := l.ownedAllocCleanupForExprWithTarget(value); ok {
		if destLocal, ok := ownedAllocCleanupTransferLocal(sourceCleanup, sourceTarget, info.Base, info.SlotCount); ok {
			l.transferOwnedAllocCleanupToLocal(sourceCleanup, destLocal)
			return
		}
	}
	if sourceCleanup, destSlot, ok := l.ownedAllocCleanupForStructLiteralField(value, info.TypeName); ok {
		if destSlot >= 0 && destSlot < info.SlotCount {
			l.transferOwnedAllocCleanupToLocal(sourceCleanup, info.Base+destSlot)
			return
		}
	}
	if sourceCleanup, destSlot, ok := l.ownedAllocCleanupForEnumConstructorPayload(value, info.TypeName); ok {
		if destSlot >= 0 && destSlot < info.SlotCount {
			l.transferOwnedAllocCleanupToLocal(sourceCleanup, info.Base+destSlot)
			return
		}
	}
	if sourceCleanup, destSlot, conditionalTagSlot, conditional, ok := l.ownedAllocCleanupForMatchResult(value, info.TypeName); ok {
		if destSlot >= 0 && destSlot < info.SlotCount {
			l.rememberOwnedAllocCleanupAtLocal(sourceCleanup, info.Base+destSlot, info.Base+conditionalTagSlot, conditional)
			return
		}
	}
	if sourceCleanup, destSlot, ok := l.ownedAllocCleanupForCatchRelayResult(value); ok {
		if destSlot >= 0 && destSlot < info.SlotCount {
			l.transferOwnedAllocCleanupToLocal(sourceCleanup, info.Base+destSlot)
			return
		}
	}
	if sourceCleanup, destSlot, ok := l.ownedAllocCleanupForNonOwnedErrorCatchResult(value); ok {
		if destSlot >= 0 && destSlot < info.SlotCount {
			l.transferOwnedAllocCleanupToLocal(sourceCleanup, info.Base+destSlot)
			return
		}
	}
	if sourceCleanup, destSlot, ok := l.ownedAllocCleanupForOwnedErrorMixedCatchResult(value); ok {
		if destSlot >= 0 && destSlot < info.SlotCount {
			l.transferOwnedAllocCleanupToLocal(sourceCleanup, info.Base+destSlot)
			return
		}
	}
	if summary, ok := l.ownedReturnSummaryForCallExpr(value); ok {
		cleanup, ok := l.ownedAllocCleanupForReturnSummary(summary, info.TypeName, info.Base, info.SlotCount)
		if ok {
			l.ownedAllocCleanups = append(l.ownedAllocCleanups, cleanup)
		}
		return
	}
	if info.TypeName != "ptr" || !isDirectAllocBytesCall(value) {
		return
	}
	l.ownedAllocCleanups = append(l.ownedAllocCleanups, ownedAllocCleanup{
		local:       info.Base,
		scopeDepth:  l.ownedLocalScopeDepthForLocal(info.Base),
		layoutID:    "layout:core.alloc_bytes:" + name,
		domain:      ir.IROwnershipDomainHeap,
		releaseKind: ir.IRReleaseKindLinuxMmap,
	})
}

func (l *lowerer) rememberOwnedAllocCleanupForAssignedLocal(target lvalueInfo, value frontend.Expr) {
	if !l.ownedAllocDropLowering || !ownedReturnStorageType(target.TypeName, l.types) {
		return
	}
	if sourceCleanup, destSlot, ok := l.ownedAllocCleanupForStructLiteralField(value, target.TypeName); ok {
		if destSlot >= 0 && destSlot < target.SlotCount {
			l.transferOwnedAllocCleanupToLocal(sourceCleanup, target.Base+destSlot)
			return
		}
	}
	if sourceCleanup, destSlot, ok := l.ownedAllocCleanupForEnumConstructorPayload(value, target.TypeName); ok {
		if destSlot >= 0 && destSlot < target.SlotCount {
			l.transferOwnedAllocCleanupToLocal(sourceCleanup, target.Base+destSlot)
			return
		}
	}
	if sourceCleanup, destSlot, conditionalTagSlot, conditional, ok := l.ownedAllocCleanupForMatchResult(value, target.TypeName); ok {
		if destSlot >= 0 && destSlot < target.SlotCount {
			l.rememberOwnedAllocCleanupAtLocal(sourceCleanup, target.Base+destSlot, target.Base+conditionalTagSlot, conditional)
			return
		}
	}
	if sourceCleanup, destSlot, ok := l.ownedAllocCleanupForCatchRelayResult(value); ok {
		if destSlot >= 0 && destSlot < target.SlotCount {
			l.transferOwnedAllocCleanupToLocal(sourceCleanup, target.Base+destSlot)
			return
		}
	}
	if sourceCleanup, destSlot, ok := l.ownedAllocCleanupForNonOwnedErrorCatchResult(value); ok {
		if destSlot >= 0 && destSlot < target.SlotCount {
			l.transferOwnedAllocCleanupToLocal(sourceCleanup, target.Base+destSlot)
			return
		}
	}
	if sourceCleanup, destSlot, ok := l.ownedAllocCleanupForOwnedErrorMixedCatchResult(value); ok {
		if destSlot >= 0 && destSlot < target.SlotCount {
			l.transferOwnedAllocCleanupToLocal(sourceCleanup, target.Base+destSlot)
			return
		}
	}
	if summary, ok := l.ownedReturnSummaryForCallExpr(value); ok {
		cleanup, ok := l.ownedAllocCleanupForReturnSummary(summary, target.TypeName, target.Base, target.SlotCount)
		if ok {
			l.ownedAllocCleanups = append(l.ownedAllocCleanups, cleanup)
		}
		return
	}
	if target.TypeName != "ptr" || !isDirectAllocBytesCall(value) {
		return
	}
	name := target.Name
	if name == "" {
		name = fmt.Sprintf("local%d", target.Base)
	}
	l.ownedAllocCleanups = append(l.ownedAllocCleanups, ownedAllocCleanup{
		local:       target.Base,
		scopeDepth:  l.ownedLocalScopeDepthForLocal(target.Base),
		layoutID:    "layout:core.alloc_bytes:" + name,
		domain:      ir.IROwnershipDomainHeap,
		releaseKind: ir.IRReleaseKindLinuxMmap,
	})
}

func (l *lowerer) ownedAllocCleanupForReturnSummary(
	summary ownedReturnSummary,
	typeName string,
	base int,
	slotCount int,
) (ownedAllocCleanup, bool) {
	if summary.returnSlot < 0 || summary.returnSlot >= slotCount {
		return ownedAllocCleanup{}, false
	}
	local := base + summary.returnSlot
	cleanup := ownedAllocCleanup{
		local:       local,
		scopeDepth:  l.ownedLocalScopeDepthForLocal(local),
		layoutID:    summary.layoutID,
		domain:      summary.domain,
		releaseKind: summary.releaseKind,
	}
	if summary.conditional {
		tagSlot := summary.conditionalTagSlot
		if tagSlot < 0 || tagSlot >= slotCount || tagSlot == summary.returnSlot {
			return ownedAllocCleanup{}, false
		}
		cleanup.conditionalLocal = base + tagSlot
		cleanup.hasConditionalLocal = true
		cleanup.hasConditionalTagExactValue = summary.hasConditionalTagExactValue
		cleanup.conditionalTagExactValue = summary.conditionalTagExactValue
	}
	return cleanup, true
}

func (l *lowerer) ownedAllocCleanupForStructLiteralField(
	expr frontend.Expr,
	destType string,
) (ownedAllocCleanup, int, bool) {
	return l.ownedAllocCleanupForStructLiteralFieldInType(expr, destType, map[string]bool{})
}

func (l *lowerer) ownedAllocCleanupForEnumConstructorPayload(
	expr frontend.Expr,
	destType string,
) (ownedAllocCleanup, int, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return ownedAllocCleanup{}, 0, false
	}
	typeName, caseInfo, ok := enumCaseConstructorInfo(call, destType, l.types)
	if !ok || typeName != destType {
		return ownedAllocCleanup{}, 0, false
	}
	destOwnedSlot, ok := ownedReturnStorageSlot(destType, l.types)
	if !ok {
		return ownedAllocCleanup{}, 0, false
	}
	payloadOffset := 1
	for i, arg := range call.Args {
		if i >= len(caseInfo.PayloadTypes) || i >= len(caseInfo.PayloadSlots) {
			return ownedAllocCleanup{}, 0, false
		}
		payloadType := caseInfo.PayloadTypes[i]
		payloadSlots := caseInfo.PayloadSlots[i]
		payloadOwnedSlot, ok := ownedReturnStorageSlot(payloadType, l.types)
		if !ok || payloadOwnedSlot < 0 || payloadOwnedSlot >= payloadSlots {
			payloadOffset += payloadSlots
			continue
		}
		destSlot := payloadOffset + payloadOwnedSlot
		if destSlot != destOwnedSlot {
			return ownedAllocCleanup{}, 0, false
		}
		if payloadType == "ptr" && isDirectAllocBytesCall(arg) {
			return ownedAllocCleanup{
				local:       -1,
				layoutID:    fmt.Sprintf("layout:core.alloc_bytes:%s.%s", destType, caseInfo.Name),
				domain:      ir.IROwnershipDomainHeap,
				releaseKind: ir.IRReleaseKindLinuxMmap,
			}, destSlot, true
		}
		if summary, ok := l.ownedReturnSummaryForCallExpr(arg); ok {
			if summary.returnSlot != payloadOwnedSlot {
				return ownedAllocCleanup{}, 0, false
			}
			return ownedAllocCleanup{
				local:       -1,
				layoutID:    summary.layoutID,
				domain:      summary.domain,
				releaseKind: summary.releaseKind,
			}, destSlot, true
		}
		cleanup, sourceTarget, ok := l.ownedAllocCleanupForExprWithTarget(arg)
		if ok {
			sourceSlot := cleanup.local - sourceTarget.Base
			if sourceSlot != payloadOwnedSlot {
				return ownedAllocCleanup{}, 0, false
			}
			return cleanup, destSlot, true
		}
		nestedCleanup, nestedSlot, nestedOK := l.ownedAllocCleanupForStructLiteralField(arg, payloadType)
		if !nestedOK || nestedSlot != payloadOwnedSlot {
			return ownedAllocCleanup{}, 0, false
		}
		return nestedCleanup, destSlot, true
	}
	return ownedAllocCleanup{}, 0, false
}

func (l *lowerer) ownedAllocCleanupForMatchResult(
	expr frontend.Expr,
	destType string,
) (ownedAllocCleanup, int, int, ownedReturnCondition, bool) {
	matchExpr, ok := expr.(*frontend.MatchExpr)
	if !ok || matchExpr == nil || len(matchExpr.Cases) == 0 {
		return ownedAllocCleanup{}, 0, 0, ownedReturnCondition{}, false
	}
	destOwnedSlot, ok := ownedReturnStorageSlot(destType, l.types)
	if !ok {
		return ownedAllocCleanup{}, 0, 0, ownedReturnCondition{}, false
	}
	if !matchExprOwnedResultGuardCoverageOK(matchExpr.Cases) {
		return ownedAllocCleanup{}, 0, 0, ownedReturnCondition{}, false
	}
	var cleanup ownedAllocCleanup
	haveCleanup := false
	hasNonOwnedArm := false
	allNonOwnedTagsZero := true
	ownedTagValue := int32(-1)
	for _, c := range matchExpr.Cases {
		armCleanup, armSlot, ok := l.ownedAllocCleanupForEnumConstructorPayload(c.Value, destType)
		if !ok {
			nonOwnedTag, tagOK := enumConstructorNonOwnedTag(c.Value, destType, l.types)
			if !tagOK {
				return ownedAllocCleanup{}, 0, 0, ownedReturnCondition{}, false
			}
			if nonOwnedTag != 0 {
				allNonOwnedTagsZero = false
			}
			hasNonOwnedArm = true
			continue
		}
		if armSlot != destOwnedSlot || armCleanup.local >= 0 {
			return ownedAllocCleanup{}, 0, 0, ownedReturnCondition{}, false
		}
		if _, caseInfo, caseOK := enumConstructorInfoForValue(c.Value, destType, l.types); !caseOK {
			return ownedAllocCleanup{}, 0, 0, ownedReturnCondition{}, false
		} else if ownedTagValue < 0 {
			ownedTagValue = caseInfo.Ordinal
		} else if ownedTagValue != caseInfo.Ordinal {
			return ownedAllocCleanup{}, 0, 0, ownedReturnCondition{}, false
		}
		if !haveCleanup {
			cleanup = armCleanup
			haveCleanup = true
			continue
		}
		if cleanup.layoutID != armCleanup.layoutID ||
			cleanup.domain != armCleanup.domain ||
			cleanup.releaseKind != armCleanup.releaseKind {
			return ownedAllocCleanup{}, 0, 0, ownedReturnCondition{}, false
		}
	}
	if !haveCleanup {
		return ownedAllocCleanup{}, 0, 0, ownedReturnCondition{}, false
	}
	condition, conditionOK := ownedReturnConditionForEnumMixedResult(
		hasNonOwnedArm,
		allNonOwnedTagsZero,
		ownedTagValue,
	)
	if !conditionOK {
		return ownedAllocCleanup{}, 0, 0, ownedReturnCondition{}, false
	}
	conditionalTagSlot := -1
	if condition.conditional {
		var guardOK bool
		conditionalTagSlot, guardOK = conditionalOwnedReturnGuardSlot(destType, destOwnedSlot, l.types)
		if !guardOK {
			return ownedAllocCleanup{}, 0, 0, ownedReturnCondition{}, false
		}
	}
	return cleanup, destOwnedSlot, conditionalTagSlot, condition, true
}

func (l *lowerer) ownedAllocCleanupForStructLiteralFieldInType(
	expr frontend.Expr,
	destType string,
	visiting map[string]bool,
) (ownedAllocCleanup, int, bool) {
	lit, ok := expr.(*frontend.StructLitExpr)
	if !ok || lit == nil {
		return ownedAllocCleanup{}, 0, false
	}
	destOwnedSlot, ok := ownedReturnStorageSlot(destType, l.types)
	if !ok {
		return ownedAllocCleanup{}, 0, false
	}
	info, ok := l.types[destType]
	if !ok || info == nil || info.Kind != semantics.TypeStruct {
		return ownedAllocCleanup{}, 0, false
	}
	if visiting[destType] {
		return ownedAllocCleanup{}, 0, false
	}
	visiting[destType] = true
	defer delete(visiting, destType)

	fieldValues := make(map[string]frontend.Expr, len(lit.Fields))
	for _, field := range lit.Fields {
		fieldValues[field.Name] = field.Value
	}
	for _, field := range info.Fields {
		fieldOwnedSlot, ok := ownedReturnStorageSlot(field.TypeName, l.types)
		if !ok || fieldOwnedSlot < 0 || fieldOwnedSlot >= field.SlotCount {
			continue
		}
		destSlot := field.Offset + fieldOwnedSlot
		if destSlot != destOwnedSlot {
			continue
		}
		value, ok := fieldValues[field.Name]
		if !ok {
			return ownedAllocCleanup{}, 0, false
		}
		if field.TypeName == "ptr" && isDirectAllocBytesCall(value) {
			return ownedAllocCleanup{
				local:       -1,
				layoutID:    fmt.Sprintf("layout:core.alloc_bytes:%s.%s", destType, field.Name),
				domain:      ir.IROwnershipDomainHeap,
				releaseKind: ir.IRReleaseKindLinuxMmap,
			}, destSlot, true
		}
		if summary, ok := l.ownedReturnSummaryForCallExpr(value); ok {
			if summary.returnSlot != fieldOwnedSlot {
				return ownedAllocCleanup{}, 0, false
			}
			return ownedAllocCleanup{
				local:       -1,
				layoutID:    summary.layoutID,
				domain:      summary.domain,
				releaseKind: summary.releaseKind,
			}, destSlot, true
		}
		cleanup, sourceTarget, ok := l.ownedAllocCleanupForExprWithTarget(value)
		if ok {
			sourceSlot := cleanup.local - sourceTarget.Base
			if sourceSlot != fieldOwnedSlot {
				return ownedAllocCleanup{}, 0, false
			}
			return cleanup, destSlot, true
		}
		nestedCleanup, nestedSlot, nestedOK := l.ownedAllocCleanupForStructLiteralFieldInType(
			value,
			field.TypeName,
			visiting,
		)
		if !nestedOK || nestedSlot != fieldOwnedSlot {
			return ownedAllocCleanup{}, 0, false
		}
		return nestedCleanup, destSlot, true
	}
	return ownedAllocCleanup{}, 0, false
}

func (l *lowerer) rememberOwnedAllocCleanupForConsumedParam(
	fnName string,
	param frontend.ParamDecl,
	info semantics.LocalInfo,
) {
	if !l.ownedAllocDropLowering ||
		param.Ownership != "consume" ||
		info.TypeName != "ptr" ||
		info.SlotCount != 1 {
		return
	}
	layoutID := fmt.Sprintf("layout:consume_param:%s:%s", fnName, param.Name)
	cleanup := ownedAllocCleanup{
		local:       info.Base,
		scopeDepth:  l.ownedLocalScopeDepthForLocal(info.Base),
		layoutID:    layoutID,
		domain:      ir.IROwnershipDomainHeap,
		releaseKind: ir.IRReleaseKindLinuxMmap,
	}
	l.forgetOwnedAllocCleanupLocal(info.Base)
	l.ownedAllocCleanups = append(l.ownedAllocCleanups, cleanup)
	l.ownedParams = append(l.ownedParams, ir.IROwnedParam{
		Local:           info.Base,
		LayoutID:        cleanup.layoutID,
		OwnershipDomain: cleanup.domain,
		ReleaseKind:     cleanup.releaseKind,
	})
}

func (l *lowerer) ownedAllocCleanupForExpr(expr frontend.Expr) (ownedAllocCleanup, bool) {
	cleanup, _, ok := l.ownedAllocCleanupForExprWithTarget(expr)
	return cleanup, ok
}

func (l *lowerer) ownedAllocCleanupForExprWithTarget(expr frontend.Expr) (ownedAllocCleanup, lvalueInfo, bool) {
	target, ok := l.ownedAllocCleanupTargetForExpr(expr)
	if !ok {
		return ownedAllocCleanup{}, lvalueInfo{}, false
	}
	local, ok := l.ownedAllocCleanupLocalForExpr(expr)
	if !ok {
		return ownedAllocCleanup{}, lvalueInfo{}, false
	}
	for _, cleanup := range l.ownedAllocCleanups {
		if cleanup.local == local {
			return cleanup, target, true
		}
	}
	return ownedAllocCleanup{}, lvalueInfo{}, false
}

func ownedAllocCleanupTransferLocal(
	cleanup ownedAllocCleanup,
	sourceTarget lvalueInfo,
	destBase int,
	destSlotCount int,
) (int, bool) {
	if sourceTarget.SlotCount <= 0 || destSlotCount <= 0 {
		return -1, false
	}
	offset := cleanup.local - sourceTarget.Base
	if offset < 0 || offset >= sourceTarget.SlotCount || offset >= destSlotCount {
		return -1, false
	}
	return destBase + offset, true
}

func (l *lowerer) ownedReturnSummaryForCallExpr(expr frontend.Expr) (ownedReturnSummary, bool) {
	if !l.ownedAllocDropLowering {
		return ownedReturnSummary{}, false
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return ownedReturnSummary{}, false
	}
	call = lowerCallExprWithBuiltinAlias(call)
	sig, ok := l.funcs[call.Name]
	if !ok || sig.ThrowsType != "" {
		return ownedReturnSummary{}, false
	}
	summary, ok := l.ownedReturnSummaryForCallName(call.Name)
	if !ok || summary.returnSlot < 0 || summary.returnSlot >= sig.ReturnSlots {
		return ownedReturnSummary{}, false
	}
	for _, ownership := range sig.ParamOwnership {
		if ownership == "inout" {
			return ownedReturnSummary{}, false
		}
	}
	return summary, true
}

func (l *lowerer) ownedReturnSummaryForCallName(name string) (ownedReturnSummary, bool) {
	if !l.ownedAllocDropLowering || len(l.ownedReturnSummaries) == 0 {
		return ownedReturnSummary{}, false
	}
	summary, ok := l.ownedReturnSummaries[name]
	return summary, ok
}

func (l *lowerer) ownedThrowSummaryForCallExpr(expr frontend.Expr) (ownedThrowSummary, bool) {
	if !l.ownedAllocDropLowering {
		return ownedThrowSummary{}, false
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return ownedThrowSummary{}, false
	}
	call = lowerCallExprWithBuiltinAlias(call)
	sig, ok := l.funcs[call.Name]
	if !ok || sig.ThrowsType == "" {
		return ownedThrowSummary{}, false
	}
	summary, ok := l.ownedThrowSummaryForCallName(call.Name)
	if !ok || summary.errorSlot < 0 {
		return ownedThrowSummary{}, false
	}
	for _, ownership := range sig.ParamOwnership {
		if ownership == "inout" {
			return ownedThrowSummary{}, false
		}
	}
	return summary, true
}

func (l *lowerer) ownedThrowSummaryForCallName(name string) (ownedThrowSummary, bool) {
	if !l.ownedAllocDropLowering || len(l.ownedThrowSummaries) == 0 {
		return ownedThrowSummary{}, false
	}
	summary, ok := l.ownedThrowSummaries[name]
	return summary, ok
}

func isDirectAllocBytesCall(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok {
		return false
	}
	call = lowerCallExprWithBuiltinAlias(call)
	return call.Name == "core.alloc_bytes"
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
			l.emitOwnedAllocCleanupRaw(pos)
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
	l.emitOwnedAllocCleanupRaw(pos)
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
		if err := l.emitDeferredFramesSince(0, pos); err != nil {
			return 0, err
		}
		l.emitCleanup(pos)
		l.emitOwnedAllocCleanup(pos)
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
	if err := l.emitDeferredFramesSince(0, pos); err != nil {
		return 0, err
	}
	l.emitCleanup(pos)
	l.emitOwnedAllocCleanup(pos)
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
		continueLabel:   continueLabel,
		breakLabel:      breakLabel,
		cleanupDepth:    len(l.cleanupIslands),
		ownedScopeDepth: l.currentOwnedCleanupScopeDepth() + 1,
		deferDepth:      len(l.deferFrames),
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
	for i, param := range fn.OwnedParams {
		if param.Local < 0 || param.Local >= fn.LocalSlots {
			return irVerifierError(
				"ir verifier: %s owned param local %d out of range (index=%d locals=%d)",
				fn.Name,
				param.Local,
				i,
				fn.LocalSlots,
			)
		}
		if param.LayoutID == "" || param.OwnershipDomain == "" || param.ReleaseKind == "" {
			return irVerifierError(
				"ir verifier: %s owned param local %d missing typed release metadata",
				fn.Name,
				param.Local,
			)
		}
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
			hasOwnedReturnMetadata := instr.LayoutID != "" ||
				instr.OwnershipDomain != "" ||
				instr.ReleaseKind != ""
			if hasOwnedReturnMetadata {
				if instr.RetSlots <= 0 {
					return verifyError(
						fn,
						i,
						"call %q owned return metadata requires return slots",
						instr.Name,
					)
				}
				if instr.OwnedReturnSlot < 0 || instr.OwnedReturnSlot >= instr.RetSlots {
					return verifyError(
						fn,
						i,
						"call %q owned return slot %d out of range for %d return slots",
						instr.Name,
						instr.OwnedReturnSlot,
						instr.RetSlots,
					)
				}
				if instr.LayoutID == "" || instr.OwnershipDomain == "" || instr.ReleaseKind == "" {
					return verifyError(
						fn,
						i,
						"call %q owned return metadata is incomplete",
						instr.Name,
					)
				}
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
	case ir.IRDropOwned:
		return 1, 1, true
	case ir.IRReleaseAllocation:
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
