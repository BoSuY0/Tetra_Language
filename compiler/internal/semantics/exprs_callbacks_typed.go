package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	semanticsexpressions "tetra_language/compiler/internal/semantics/expressions"
	semanticspolicy "tetra_language/compiler/internal/semantics/policy"
)

var noblockForbiddenCallEffects = semanticspolicy.NoblockForbiddenCallEffects

var realtimeForbiddenCallEffects = semanticspolicy.RealtimeForbiddenCallEffects

func currentCallerSignature(effects *effectContext, funcs map[string]FuncSig) (FuncSig, bool) {
	if effects == nil || effects.funcName == "" {
		return FuncSig{}, false
	}
	sig, ok := funcs[effects.funcName]
	if !ok {
		return FuncSig{}, false
	}
	return sig, true
}

func validateCallAgainstSemanticClauses(callerSig FuncSig, calleeSig FuncSig, calleeName string, pos frontend.Position) error {
	return validateCallAgainstSemanticClauseTarget(callerSig, calleeSig, fmt.Sprintf("call to '%s'", calleeName), pos)
}

func validateCallAgainstSemanticClauseTarget(callerSig FuncSig, calleeSig FuncSig, calleePhrase string, pos frontend.Position) error {
	if err := validateBudgetedSemanticCallTarget(callerSig, calleeSig, calleePhrase, pos); err != nil {
		return err
	}
	if callerSig.HasRealtime {
		if blocked := firstFuncSigForbiddenEffect(calleeSig, realtimeForbiddenCallEffects); blocked != "" {
			return fmt.Errorf("%s: semantic clause 'realtime' forbids %s because it is not realtime-safe (effect '%s')", frontend.FormatPos(pos), calleePhrase, blocked)
		}
	}
	if callerSig.HasNoAlloc && funcSigHasEffect(calleeSig, "alloc") {
		return fmt.Errorf("%s: semantic clause 'noalloc' forbids %s because it may allocate", frontend.FormatPos(pos), calleePhrase)
	}
	if callerSig.HasNoBlock {
		if blocked := firstFuncSigForbiddenEffect(calleeSig, noblockForbiddenCallEffects); blocked != "" {
			return fmt.Errorf("%s: semantic clause 'noblock' forbids %s because it may block (effect '%s')", frontend.FormatPos(pos), calleePhrase, blocked)
		}
	}
	return nil
}

func validateBudgetedSemanticCallTarget(callerSig FuncSig, calleeSig FuncSig, calleePhrase string, pos frontend.Position) error {
	if !calleeSig.HasBudget {
		return nil
	}
	required := calleeSig.Budget
	if !callerSig.HasBudget {
		return budgetDiagnosticf(pos, "budget context for %s requires caller budget at least %d", calleePhrase, required)
	}
	if callerSig.Budget < required {
		return budgetDiagnosticf(pos, "budget context for %s requires caller budget at least %d, got %d", calleePhrase, required, callerSig.Budget)
	}
	return nil
}

func validateCallbackClauseCompatibility(
	callbackSig FuncSig,
	calleeSig FuncSig,
	calleeName string,
	pos frontend.Position,
	callbackName string,
) error {
	if err := validateBudgetedSemanticCallTarget(calleeSig, callbackSig, fmt.Sprintf("callback function symbol '%s' for callee '%s'", callbackName, calleeName), pos); err != nil {
		return err
	}
	if calleeSig.HasRealtime {
		if blocked := firstFuncSigForbiddenEffect(callbackSig, realtimeForbiddenCallEffects); blocked != "" {
			return fmt.Errorf("%s: callback function symbol '%s' is not realtime-safe (effect '%s') for callee '%s'", frontend.FormatPos(pos), callbackName, blocked, calleeName)
		}
	}
	if calleeSig.HasNoAlloc && funcSigHasEffect(callbackSig, "alloc") {
		return fmt.Errorf("%s: callback function symbol '%s' may allocate but callee '%s' has semantic clause 'noalloc'", frontend.FormatPos(pos), callbackName, calleeName)
	}
	if calleeSig.HasNoBlock {
		if blocked := firstFuncSigForbiddenEffect(callbackSig, noblockForbiddenCallEffects); blocked != "" {
			return fmt.Errorf("%s: callback function symbol '%s' may block (effect '%s') but callee '%s' has semantic clause 'noblock'", frontend.FormatPos(pos), callbackName, blocked, calleeName)
		}
	}
	return nil
}

func funcSigHasEffect(sig FuncSig, effect string) bool {
	return semanticspolicy.FuncSigHasEffect(sig, effect)
}

func actorTaskWorkerBoundaryEffect(sig FuncSig) string {
	return semanticspolicy.ActorTaskWorkerBoundaryEffect(sig)
}

func firstFuncSigForbiddenEffect(sig FuncSig, forbidden []string) string {
	return semanticspolicy.FirstFuncSigForbiddenEffect(sig, forbidden)
}

func hasStrictSemanticCallClauses(sig FuncSig) bool {
	return semanticspolicy.HasStrictSemanticCallClauses(sig)
}

func firstStrictSemanticCallClause(sig FuncSig) string {
	return semanticspolicy.FirstStrictSemanticCallClause(sig)
}

func resolveCallbackArgumentType(
	arg frontend.Expr,
	calleeName string,
	calleeSig FuncSig,
	paramIndex int,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
	deferEffectValidation bool,
) (string, string, error) {
	if closure, ok := arg.(*frontend.ClosureExpr); ok {
		targetInfo := functionParamLocalInfo(calleeSig, paramIndex)
		if err := validateFunctionTypedClosureAssignment("closure literal", targetInfo, closure, locals, funcs, types, module, imports, closure.At, "callback argument"); err != nil {
			return "", "", err
		}
		if len(closure.Captures) > 0 {
			captureSlots, err := functionCaptureSlotCount(closure.Captures, types)
			if err != nil {
				return "", "", err
			}
			if captureSlots > FnPtrEnvSlotCount {
				if _, _, err := classifyCallableEscape(callableBoundaryCallback, closure.Captures, types); err != nil {
					return "", "", err
				}
			}
		}
		callbackSig, ok := funcs[closure.Name]
		if !ok {
			callbackSig = FuncSig{
				ParamTypes:     append([]string(nil), targetInfo.FunctionParamTypes...),
				ParamOwnership: append([]string(nil), targetInfo.FunctionParamOwnership...),
				ReturnType:     targetInfo.FunctionReturnType,
				Effects:        append([]string(nil), targetInfo.FunctionEffects...),
			}
		}
		if !deferEffectValidation {
			if err := validateCallbackClauseCompatibility(callbackSig, calleeSig, calleeName, closure.At, callbackArgumentName(arg)); err != nil {
				return "", "", err
			}
		}
		return targetInfo.TypeName, closure.Name, nil
	}
	if call, ok := arg.(*frontend.CallExpr); ok {
		argType, _, err := checkExprWithEffects(call, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", "", err
		}
		callSig, ok := funcs[call.Name]
		if !ok || !callSig.ReturnFunctionType {
			return "", "", fmt.Errorf("%s: callback argument call '%s' does not return a function type", frontend.FormatPos(call.At), call.Name)
		}
		if err := markFunctionTypedReturnCallMutableGlobalUse(callSig, call, locals, globals, funcs, types, module, imports, analysis); err != nil {
			return "", "", err
		}
		returnedSig := FuncSig{
			ParamTypes:     append([]string(nil), callSig.ReturnFunctionParams...),
			ParamOwnership: append([]string(nil), callSig.ReturnFunctionParamOwnership...),
			ReturnType:     callSig.ReturnFunctionReturn,
			ThrowsType:     callSig.ReturnFunctionThrows,
			Effects:        append([]string(nil), callSig.ReturnFunctionEffects...),
		}
		if err := validateCallbackSignature(returnedSig, calleeSig, paramIndex, call.At, callbackArgumentName(call), deferEffectValidation); err != nil {
			return "", "", err
		}
		if err := validateCallbackClauseCompatibility(returnedSig, calleeSig, calleeName, call.At, callbackArgumentName(call)); err != nil {
			return "", "", err
		}
		return argType, "", nil
	}
	if fieldInfo, ok, err := resolveFunctionFieldArgument(arg, locals); err != nil {
		return "", "", err
	} else if ok {
		if analysis != nil && fieldInfo.FunctionTouchesMutableGlobals {
			analysis.touchesMutableGlobals = true
		}
		if fieldInfo.FunctionValue == "" {
			paramSlots, err := functionParamSlotCount(fieldInfo.FunctionParamTypes, types)
			if err != nil {
				return "", "", err
			}
			callbackSig := FuncSig{
				ParamTypes:     append([]string(nil), fieldInfo.FunctionParamTypes...),
				ParamOwnership: append([]string(nil), fieldInfo.FunctionParamOwnership...),
				ParamSlots:     paramSlots,
				ReturnType:     fieldInfo.FunctionReturnType,
				ThrowsType:     fieldInfo.FunctionThrowsType,
				Effects:        append([]string(nil), fieldInfo.FunctionEffects...),
			}
			if err := validateCallbackSignatureForField(callbackSig, fieldInfo, calleeSig, paramIndex, arg.Pos(), callbackArgumentName(arg), types, deferEffectValidation); err != nil {
				return "", "", err
			}
			if err := validateCallbackClauseCompatibility(callbackSig, calleeSig, calleeName, arg.Pos(), callbackArgumentName(arg)); err != nil {
				return "", "", err
			}
			return "fnptr", "", nil
		}
		localSig, ok := funcs[fieldInfo.FunctionValue]
		if !ok {
			return "", "", fmt.Errorf("%s: unknown callback function symbol '%s'", frontend.FormatPos(arg.Pos()), fieldInfo.FunctionValue)
		}
		if err := validateCallbackSignatureForField(localSig, fieldInfo, calleeSig, paramIndex, arg.Pos(), callbackArgumentName(arg), types, deferEffectValidation); err != nil {
			return "", "", err
		}
		if err := validateCallbackClauseCompatibility(localSig, calleeSig, calleeName, arg.Pos(), callbackArgumentName(arg)); err != nil {
			return "", "", err
		}
		return "fnptr", fieldInfo.FunctionValue, nil
	}
	if fieldAccess, ok := arg.(*frontend.FieldAccessExpr); ok {
		globalInfo, globalSig, globalOK, err := resolveFunctionTypedGlobalFieldAccess(fieldAccess, globals, funcs)
		if err != nil {
			return "", "", err
		}
		if globalOK {
			globalLocal := LocalInfo{
				TypeName:               globalInfo.TypeName,
				FunctionTypeValue:      true,
				FunctionValue:          globalInfo.FunctionValue,
				FunctionParamTypes:     append([]string(nil), globalInfo.FunctionParamTypes...),
				FunctionParamOwnership: append([]string(nil), globalInfo.FunctionParamOwnership...),
				FunctionReturnType:     globalInfo.FunctionReturnType,
				FunctionThrowsType:     globalInfo.FunctionThrowsType,
				FunctionEffects:        append([]string(nil), globalInfo.FunctionEffects...),
			}
			if err := validateCallbackSignatureForLocal(globalSig, globalLocal, calleeSig, paramIndex, arg.Pos(), callbackArgumentName(arg), types, deferEffectValidation); err != nil {
				return "", "", err
			}
			if err := validateCallbackClauseCompatibility(globalSig, calleeSig, calleeName, arg.Pos(), callbackArgumentName(arg)); err != nil {
				return "", "", err
			}
			return globalInfo.TypeName, globalInfo.FunctionValue, nil
		}
	}
	id, ok := arg.(*frontend.IdentExpr)
	if !ok {
		return "", "", unsupportedCallbackArgumentSourceError(arg.Pos(), calleeName)
	}
	if localInfo, ok := locals[id.Name]; ok {
		if analysis != nil && localInfo.FunctionTouchesMutableGlobals {
			analysis.touchesMutableGlobals = true
		}
		if !localInfo.FunctionTypeValue {
			if localInfo.FunctionValue == "" {
				return "", "", unsupportedCallbackArgumentSourceError(arg.Pos(), calleeName)
			}
			if localInfo.GenericFunctionValue {
				return "", "", unsupportedGenericCallbackSymbolError(arg.Pos(), id.Name)
			}
			localSig, ok := funcs[localInfo.FunctionValue]
			if !ok {
				return "", "", fmt.Errorf("%s: unknown callback function symbol '%s'", frontend.FormatPos(arg.Pos()), localInfo.FunctionValue)
			}
			if localSig.ThrowsType != "" && callbackExpectedThrowsType(calleeSig, paramIndex) == "" {
				return "", "", unsupportedThrowingCallbackSymbolError(arg.Pos(), id.Name)
			}
			targetInfo := functionParamLocalInfo(calleeSig, paramIndex)
			targetInfo.FunctionValue = localInfo.FunctionValue
			targetInfo.FunctionCaptures = append([]frontend.ClosureCapture(nil), localInfo.FunctionCaptures...)
			if len(targetInfo.FunctionCaptures) > 0 {
				captureSlots, err := functionCaptureSlotCount(targetInfo.FunctionCaptures, types)
				if err != nil {
					return "", "", err
				}
				if captureSlots > FnPtrEnvSlotCount {
					escapeKind, handleValue, err := classifyCallableEscape(callableBoundaryCallback, targetInfo.FunctionCaptures, types)
					if err != nil {
						return "", "", err
					}
					targetInfo.FunctionEscapeKind = escapeKind
					targetInfo.FunctionHandleValue = handleValue
				}
			}
			if err := validateCallbackSignatureForLocal(localSig, targetInfo, calleeSig, paramIndex, arg.Pos(), id.Name, types, deferEffectValidation); err != nil {
				return "", "", err
			}
			if err := validateCallbackClauseCompatibility(localSig, calleeSig, calleeName, arg.Pos(), id.Name); err != nil {
				return "", "", err
			}
			return targetInfo.TypeName, localInfo.FunctionValue, nil
		}
		if localInfo.FunctionValue == "" {
			paramSlots, err := functionParamSlotCount(localInfo.FunctionParamTypes, types)
			if err != nil {
				return "", "", err
			}
			callbackSig := FuncSig{
				ParamTypes:     append([]string(nil), localInfo.FunctionParamTypes...),
				ParamOwnership: append([]string(nil), localInfo.FunctionParamOwnership...),
				ParamSlots:     paramSlots,
				ReturnType:     localInfo.FunctionReturnType,
				ThrowsType:     localInfo.FunctionThrowsType,
				Effects:        append([]string(nil), localInfo.FunctionEffects...),
			}
			if err := validateCallbackSignatureForLocal(callbackSig, localInfo, calleeSig, paramIndex, arg.Pos(), id.Name, types, deferEffectValidation); err != nil {
				return "", "", err
			}
			if err := validateCallbackClauseCompatibility(callbackSig, calleeSig, calleeName, arg.Pos(), id.Name); err != nil {
				return "", "", err
			}
			return "fnptr", "", nil
		}
		if localInfo.GenericFunctionValue {
			return "", "", unsupportedGenericCallbackSymbolError(arg.Pos(), id.Name)
		}
		localSig, ok := funcs[localInfo.FunctionValue]
		if !ok {
			return "", "", fmt.Errorf("%s: unknown callback function symbol '%s'", frontend.FormatPos(arg.Pos()), localInfo.FunctionValue)
		}
		if err := validateCallbackSignatureForLocal(localSig, localInfo, calleeSig, paramIndex, arg.Pos(), id.Name, types, deferEffectValidation); err != nil {
			return "", "", err
		}
		if err := validateCallbackClauseCompatibility(localSig, calleeSig, calleeName, arg.Pos(), id.Name); err != nil {
			return "", "", err
		}
		return localInfo.TypeName, localInfo.FunctionValue, nil
	}
	if globalInfo, ok := globals[id.Name]; ok {
		if analysis != nil && globalInfo.Mutable {
			analysis.touchesMutableGlobals = true
		}
		if !globalInfo.FunctionTypeValue || globalInfo.FunctionValue == "" {
			return "", "", unsupportedCallbackArgumentSourceError(arg.Pos(), calleeName)
		}
		globalSig, ok := funcs[globalInfo.FunctionValue]
		if !ok {
			return "", "", fmt.Errorf("%s: unknown callback function symbol '%s'", frontend.FormatPos(arg.Pos()), globalInfo.FunctionValue)
		}
		globalLocal := LocalInfo{
			TypeName:               globalInfo.TypeName,
			FunctionTypeValue:      true,
			FunctionValue:          globalInfo.FunctionValue,
			FunctionParamTypes:     append([]string(nil), globalInfo.FunctionParamTypes...),
			FunctionParamOwnership: append([]string(nil), globalInfo.FunctionParamOwnership...),
			FunctionReturnType:     globalInfo.FunctionReturnType,
			FunctionThrowsType:     globalInfo.FunctionThrowsType,
			FunctionEffects:        append([]string(nil), globalInfo.FunctionEffects...),
		}
		if err := validateCallbackSignatureForLocal(globalSig, globalLocal, calleeSig, paramIndex, arg.Pos(), id.Name, types, deferEffectValidation); err != nil {
			return "", "", err
		}
		if err := validateCallbackClauseCompatibility(globalSig, calleeSig, calleeName, arg.Pos(), id.Name); err != nil {
			return "", "", err
		}
		return globalInfo.TypeName, globalInfo.FunctionValue, nil
	}

	resolved, err := resolveCheckedCallName(id.Name, funcs, module, imports, id.At)
	if err != nil {
		return "", "", unsupportedCallbackArgumentSourceError(arg.Pos(), calleeName)
	}
	sig, ok := funcs[resolved]
	if !ok {
		return "", "", unsupportedCallbackArgumentSourceError(arg.Pos(), calleeName)
	}
	if err := ensureFuncVisible(resolved, sig, module, id.At); err != nil {
		return "", "", err
	}
	if sig.Generic {
		return "", "", unsupportedGenericCallbackSymbolError(arg.Pos(), id.Name)
	}
	if sig.ThrowsType != "" && callbackExpectedThrowsType(calleeSig, paramIndex) == "" {
		return "", "", unsupportedThrowingCallbackSymbolError(arg.Pos(), id.Name)
	}
	if err := validateCallbackSignature(sig, calleeSig, paramIndex, arg.Pos(), id.Name, deferEffectValidation); err != nil {
		return "", "", err
	}
	if err := validateCallbackClauseCompatibility(sig, calleeSig, calleeName, arg.Pos(), id.Name); err != nil {
		return "", "", err
	}
	return "fnptr", resolved, nil
}

func unsupportedCallbackArgumentSourceError(pos frontend.Position, calleeName string) error {
	return fmt.Errorf(
		"%s: callback argument for '%s' must be a supported fnptr source: closure literal, function-typed local/global/struct field, direct named function/closure symbol, or function-typed return call",
		frontend.FormatPos(pos),
		calleeName,
	)
}

func unsupportedCallbackCaptureError(pos frontend.Position, rawName string, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf("%s: callback argument '%s' captures %d environment slots; captured callback arguments support at most %d fnptr environment slots within the supported fnptr ABI", frontend.FormatPos(pos), rawName, envSlots, FnPtrEnvSlotCount)
	}
	return fmt.Errorf("%s: callback argument '%s' captures local values; captured function values cannot be passed as callback arguments in this MVP; closure lifetime/ABI evidence is only available for local direct calls", frontend.FormatPos(pos), rawName)
}

func unsupportedClosureLiteralCallbackCaptureError(pos frontend.Position, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf("%s: callback argument 'closure literal' captures %d environment slots; captured callback arguments support at most %d fnptr environment slots within the supported fnptr ABI", frontend.FormatPos(pos), envSlots, FnPtrEnvSlotCount)
	}
	return fmt.Errorf("%s: callback argument 'closure literal' captures local values; captured function values cannot be passed as callback arguments in this MVP; closure lifetime/ABI evidence is only available for local direct calls", frontend.FormatPos(pos))
}

func unsupportedFunctionTypedCallCaptureError(pos frontend.Position, rawName string, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf("%s: function-typed callback '%s' captures %d environment slots; direct function-typed calls support at most %d fnptr environment slots within the supported fnptr ABI", frontend.FormatPos(pos), rawName, envSlots, FnPtrEnvSlotCount)
	}
	return fmt.Errorf("%s: function-typed callback '%s' has unsupported captured environment size", frontend.FormatPos(pos), rawName)
}

func unsupportedFunctionFieldCallCaptureError(pos frontend.Position, rawName string, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf("%s: function-typed struct field call '%s' captures %d environment slots; direct struct-field calls support at most %d fnptr environment slots within the supported fnptr ABI", frontend.FormatPos(pos), rawName, envSlots, FnPtrEnvSlotCount)
	}
	return fmt.Errorf("%s: function-typed struct field call '%s' has unsupported captured environment size", frontend.FormatPos(pos), rawName)
}

func unsupportedEnumPayloadCallCaptureError(pos frontend.Position, rawName string, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return fmt.Errorf("%s: function-typed enum payload binding '%s' captures %d environment slots; direct enum-payload calls support at most %d fnptr environment slots within the supported fnptr ABI", frontend.FormatPos(pos), rawName, envSlots, FnPtrEnvSlotCount)
	}
	return fmt.Errorf("%s: function-typed enum payload binding '%s' has unsupported captured environment size", frontend.FormatPos(pos), rawName)
}

func validateCallbackSignatureForLocal(
	callbackSig FuncSig,
	localInfo LocalInfo,
	calleeSig FuncSig,
	paramIndex int,
	pos frontend.Position,
	rawName string,
	types map[string]*TypeInfo,
	deferEffectValidation bool,
) error {
	if localInfo.FunctionReturnType != "" {
		explicitSlots, err := functionParamSlotCount(localInfo.FunctionParamTypes, types)
		if err != nil {
			return err
		}
		hiddenSlots := callbackSig.ParamSlots - explicitSlots
		if hiddenSlots < 0 || (hiddenSlots > FnPtrEnvSlotCount && !localInfo.FunctionHandleValue) {
			return unsupportedCallbackCaptureError(pos, rawName, hiddenSlots)
		}
		visibleSig := callbackSig
		visibleSig.ParamTypes = append([]string(nil), localInfo.FunctionParamTypes...)
		visibleSig.ParamOwnership = append([]string(nil), localInfo.FunctionParamOwnership...)
		visibleSig.ParamSlots = explicitSlots
		visibleSig.ReturnType = localInfo.FunctionReturnType
		return validateCallbackSignature(visibleSig, calleeSig, paramIndex, pos, rawName, deferEffectValidation)
	}
	if len(localInfo.FunctionCaptures) == 0 {
		return validateCallbackSignature(callbackSig, calleeSig, paramIndex, pos, rawName, deferEffectValidation)
	}
	captureSlots := 0
	for _, capture := range localInfo.FunctionCaptures {
		info, err := ensureTypeInfo(capture.Type.Name, types)
		if err != nil {
			return err
		}
		captureSlots += info.SlotCount
	}
	if captureSlots < 1 || captureSlots > FnPtrEnvSlotCount {
		return unsupportedCallbackCaptureError(pos, rawName, captureSlots)
	}
	trimmed := callbackSig
	trimmed.ParamTypes = append([]string(nil), localInfo.FunctionParamTypes...)
	trimmed.ParamOwnership = append([]string(nil), localInfo.FunctionParamOwnership...)
	trimmed.ParamSlots -= captureSlots
	return validateCallbackSignature(trimmed, calleeSig, paramIndex, pos, rawName, deferEffectValidation)
}

func validateCallbackSignatureForField(
	callbackSig FuncSig,
	fieldInfo FunctionFieldInfo,
	calleeSig FuncSig,
	paramIndex int,
	pos frontend.Position,
	rawName string,
	types map[string]*TypeInfo,
	deferEffectValidation bool,
) error {
	if fieldInfo.FunctionReturnType == "" {
		return validateCallbackSignature(callbackSig, calleeSig, paramIndex, pos, rawName, deferEffectValidation)
	}
	explicitSlots, err := functionParamSlotCount(fieldInfo.FunctionParamTypes, types)
	if err != nil {
		return err
	}
	hiddenSlots := callbackSig.ParamSlots - explicitSlots
	if hiddenSlots < 0 || (hiddenSlots > FnPtrEnvSlotCount && !fieldInfo.FunctionHandleValue) {
		return unsupportedCallbackCaptureError(pos, rawName, hiddenSlots)
	}
	visibleSig := callbackSig
	visibleSig.ParamTypes = append([]string(nil), fieldInfo.FunctionParamTypes...)
	visibleSig.ParamOwnership = append([]string(nil), fieldInfo.FunctionParamOwnership...)
	visibleSig.ParamSlots = explicitSlots
	visibleSig.ReturnType = fieldInfo.FunctionReturnType
	return validateCallbackSignature(visibleSig, calleeSig, paramIndex, pos, rawName, deferEffectValidation)
}

func functionParamSlotCount(typeNames []string, types map[string]*TypeInfo) (int, error) {
	slots := 0
	for _, typeName := range typeNames {
		info, err := ensureTypeInfo(typeName, types)
		if err != nil {
			return 0, err
		}
		slots += info.SlotCount
	}
	return slots, nil
}

func resolveFunctionFieldCall(name string, locals map[string]LocalInfo) (FunctionFieldInfo, bool, error) {
	if !strings.Contains(name, ".") {
		return FunctionFieldInfo{}, false, nil
	}
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return FunctionFieldInfo{}, false, nil
	}
	local, ok := locals[parts[0]]
	if !ok || len(local.FunctionFields) == 0 {
		return FunctionFieldInfo{}, false, nil
	}
	field, ok := local.FunctionFields[strings.Join(parts[1:], ".")]
	return field, ok, nil
}

func resolveFunctionFieldArgument(expr frontend.Expr, locals map[string]LocalInfo) (FunctionFieldInfo, bool, error) {
	name := callbackArgumentName(expr)
	if name == "" {
		return FunctionFieldInfo{}, false, nil
	}
	return resolveFunctionFieldCall(name, locals)
}

func callbackArgumentName(expr frontend.Expr) string {
	return semanticsexpressions.CallbackArgumentName(expr)
}

func validateCallbackSignature(
	callbackSig FuncSig,
	calleeSig FuncSig,
	paramIndex int,
	pos frontend.Position,
	rawName string,
	deferEffectValidation bool,
) error {
	if paramIndex >= len(calleeSig.ParamFunctionParams) || paramIndex >= len(calleeSig.ParamFunctionReturns) {
		return nil
	}
	wantParams := calleeSig.ParamFunctionParams[paramIndex]
	wantReturn := calleeSig.ParamFunctionReturns[paramIndex]
	wantReturnOwnership := ""
	if paramIndex < len(calleeSig.ParamFunctionReturnOwnership) {
		wantReturnOwnership = calleeSig.ParamFunctionReturnOwnership[paramIndex]
	}
	wantThrows := callbackExpectedThrowsType(calleeSig, paramIndex)
	wantEffects := []string(nil)
	if paramIndex < len(calleeSig.ParamFunctionEffects) {
		wantEffects = calleeSig.ParamFunctionEffects[paramIndex]
	}
	if len(wantParams) != len(callbackSig.ParamTypes) {
		return fmt.Errorf("%s: callback function symbol '%s' has incompatible parameter count: expected %d, got %d", frontend.FormatPos(pos), rawName, len(wantParams), len(callbackSig.ParamTypes))
	}
	wantOwnership := []string(nil)
	if paramIndex < len(calleeSig.ParamFunctionOwnership) {
		wantOwnership = calleeSig.ParamFunctionOwnership[paramIndex]
	}
	if err := validateFunctionTypeParamOwnership(wantOwnership, callbackSig.ParamOwnership, len(wantParams), pos, "callback function symbol", rawName); err != nil {
		return err
	}
	for i := range wantParams {
		if wantParams[i] != callbackSig.ParamTypes[i] {
			return fmt.Errorf("%s: callback function symbol '%s' parameter %d type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), rawName, i+1, wantParams[i], callbackSig.ParamTypes[i])
		}
	}
	if wantReturn != "" && wantReturn != callbackSig.ReturnType {
		return fmt.Errorf("%s: callback function symbol '%s' return type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), rawName, wantReturn, callbackSig.ReturnType)
	}
	if wantReturnOwnership != callbackSig.ReturnOwnership {
		return fmt.Errorf("%s: callback function symbol '%s' return ownership mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), rawName, ownershipDisplay(wantReturnOwnership), ownershipDisplay(callbackSig.ReturnOwnership))
	}
	if wantThrows != callbackSig.ThrowsType {
		return fmt.Errorf("%s: callback function symbol '%s' throws type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), rawName, wantThrows, callbackSig.ThrowsType)
	}
	if !deferEffectValidation {
		if err := validateFunctionTypeCallableEffects(wantEffects, callbackSig.Effects, pos, "callback function symbol", rawName); err != nil {
			return err
		}
	}
	return nil
}

func callbackExpectedThrowsType(calleeSig FuncSig, paramIndex int) string {
	if paramIndex >= 0 && paramIndex < len(calleeSig.ParamFunctionThrows) {
		return calleeSig.ParamFunctionThrows[paramIndex]
	}
	return ""
}

func validateFunctionTypedThrowCall(throwsType string, e *frontend.CallExpr, state *regionState) error {
	isTryCall := state != nil && state.allowThrowDepth > 0 && state.allowThrowCall == e
	isCatchCall := state != nil && state.allowCatchDepth > 0 && state.allowCatchCall == e
	if throwsType == "" {
		if isTryCall {
			return fmt.Errorf("%s: try expects a throwing function call", frontend.FormatPos(e.At))
		}
		if isCatchCall {
			return fmt.Errorf("%s: catch expects a throwing function call", frontend.FormatPos(e.At))
		}
		return nil
	}
	if !isTryCall && !isCatchCall {
		return fmt.Errorf("%s: call to throwing function '%s' requires try", frontend.FormatPos(e.At), e.Name)
	}
	if isTryCall && state.throwType == "" {
		return fmt.Errorf("%s: try is only allowed in throwing functions", frontend.FormatPos(e.At))
	}
	if isTryCall && !typesCompatibleWithNullPtr(state.throwType, throwsType, e) {
		return fmt.Errorf("%s: thrown error type mismatch: expected '%s', got '%s'", frontend.FormatPos(e.At), state.throwType, throwsType)
	}
	return nil
}

func checkTypedActorBuiltin(
	e *frontend.CallExpr,
	resolved string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) (string, int, error) {
	if err := effects.requireAll(e.At, []string{"actors"}); err != nil {
		return "", regionNone, err
	}
	switch resolved {
	case "core.send_typed":
		if len(e.TypeArgs) != 0 {
			return "", regionNone, fmt.Errorf("%s: send_typed does not accept explicit type arguments", frontend.FormatPos(e.At))
		}
		if len(e.Args) != 2 {
			return "", regionNone, fmt.Errorf("%s: send_typed expects 2 arguments", frontend.FormatPos(e.At))
		}
		targetType, _, err := checkExprWithEffects(e.Args[0], locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		if targetType != "actor" {
			return "", regionNone, fmt.Errorf("%s: type mismatch for 'core.send_typed' arg 1", frontend.FormatPos(e.Args[0].Pos()))
		}
		msgType, _, err := checkExprWithEffects(e.Args[1], locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		info, ok := types[msgType]
		if !ok || info.Kind != TypeEnum {
			return "", regionNone, fmt.Errorf("%s: send_typed expects an enum message", frontend.FormatPos(e.Args[1].Pos()))
		}
		if err := validateTypedActorMessageType(msgType, types, map[string]bool{}); err != nil {
			return "", regionNone, fmt.Errorf("%s: %v", frontend.FormatPos(e.Args[1].Pos()), err)
		}
		transferOwners := actorTransferOwnerPayloads(e.Args[1], msgType, types, module, imports)
		if err := validateActorBoundaryPayloadExpr(e.Args[1], msgType, types, module, imports, state, transferOwners); err != nil {
			return "", regionNone, err
		}
		if err := checkBorrowedEscape(e.Args[1], locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
			return ownershipDiagnosticf(e.Args[1].Pos(), "cannot send borrowed view across actor boundary; use .copy() (borrowed value derived from '%s' cannot cross actor boundary without copy)", borrowedName)
		}); err != nil {
			return "", regionNone, err
		}
		if err := consumeTypedActorTransferPayloads(e.Args[1], msgType, locals, types, module, imports, state); err != nil {
			return "", regionNone, err
		}
		e.Name = resolved
		return "i32", regionNone, nil
	case "core.recv_typed":
		if len(e.Args) != 0 {
			return "", regionNone, fmt.Errorf("%s: recv_typed expects 0 arguments", frontend.FormatPos(e.At))
		}
		if len(e.TypeArgs) != 1 {
			return "", regionNone, fmt.Errorf("%s: recv_typed expects one explicit type argument", frontend.FormatPos(e.At))
		}
		typeName, err := resolveTypeName(&e.TypeArgs[0], module, imports)
		if err != nil {
			return "", regionNone, err
		}
		e.TypeArgs[0].Name = typeName
		info, ok := types[typeName]
		if !ok || info.Kind != TypeEnum {
			return "", regionNone, fmt.Errorf("%s: recv_typed expects an enum type argument", frontend.FormatPos(e.TypeArgs[0].At))
		}
		if err := validateTypedActorMessageType(typeName, types, map[string]bool{}); err != nil {
			return "", regionNone, fmt.Errorf("%s: %v", frontend.FormatPos(e.TypeArgs[0].At), err)
		}
		e.Name = resolved
		return typeName, regionNone, nil
	default:
		return "", regionNone, fmt.Errorf("%s: unknown typed actor builtin '%s'", frontend.FormatPos(e.At), resolved)
	}
}

func checkTypedTaskBuiltin(
	e *frontend.CallExpr,
	resolved string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) (string, int, error) {
	if err := effects.requireAll(e.At, []string{"runtime"}); err != nil {
		return "", regionNone, err
	}
	if len(e.TypeArgs) != 1 {
		return "", regionNone, fmt.Errorf("%s: %s expects one explicit error type argument", frontend.FormatPos(e.At), resolved)
	}
	errorType, err := resolveTypeName(&e.TypeArgs[0], module, imports)
	if err != nil {
		return "", regionNone, err
	}
	e.TypeArgs[0].Name = errorType
	if err := validateTypedTaskErrorType(errorType, types, e.TypeArgs[0].At); err != nil {
		return "", regionNone, err
	}
	handleType, handleInfo, err := EnsureTypedTaskHandleType(errorType, types)
	if err != nil {
		return "", regionNone, fmt.Errorf("%s: %v", frontend.FormatPos(e.TypeArgs[0].At), err)
	}

	switch resolved {
	case "core.task_spawn_i32_typed", "core.task_spawn_group_i32_typed":
		if resolved == "core.task_spawn_i32_typed" && len(e.Args) != 1 {
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32_typed expects 1 argument", frontend.FormatPos(e.At))
		}
		if resolved == "core.task_spawn_group_i32_typed" && len(e.Args) != 2 {
			return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32_typed expects 2 arguments", frontend.FormatPos(e.At))
		}
		workerArg := 0
		if resolved == "core.task_spawn_group_i32_typed" {
			groupType, _, err := checkExprWithEffects(e.Args[0], locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return "", regionNone, err
			}
			if groupType != "task.group" {
				return "", regionNone, fmt.Errorf("%s: type mismatch for 'core.task_spawn_group_i32_typed' arg 1", frontend.FormatPos(e.Args[0].Pos()))
			}
			if err := checkResourceCallArg(resolved, "task.group", e.Args[0], funcs, module, imports, state); err != nil {
				return "", regionNone, err
			}
			workerArg = 1
		}
		lit, ok := e.Args[workerArg].(*frontend.StringLitExpr)
		if !ok {
			if resolved == "core.task_spawn_group_i32_typed" {
				return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32_typed expects a string literal worker name", frontend.FormatPos(e.At))
			}
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32_typed expects a string literal", frontend.FormatPos(e.At))
		}
		raw := string(lit.Value)
		if raw == "" {
			if resolved == "core.task_spawn_group_i32_typed" {
				return "", regionNone, fmt.Errorf("%s: task_spawn_group_i32_typed expects a non-empty name", frontend.FormatPos(e.At))
			}
			return "", regionNone, fmt.Errorf("%s: task_spawn_i32_typed expects a non-empty name", frontend.FormatPos(e.At))
		}
		target, err := resolveKnownCallName(raw, funcs, module, imports, e.At)
		if err != nil {
			return "", regionNone, err
		}
		if strings.HasPrefix(target, "core.") {
			return "", regionNone, fmt.Errorf("%s: %s target must be a user function, got '%s'", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), target)
		}
		targetSig, ok := funcs[target]
		if !ok {
			return "", regionNone, fmt.Errorf("%s: unknown function '%s'", frontend.FormatPos(e.At), target)
		}
		if len(targetSig.ParamTypes) != 0 || targetSig.ReturnType != "i32" {
			if handleInfo.SlotCount > 4 {
				return "", regionNone, fmt.Errorf("%s: %s target must have shape func %s() -> i32", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), target)
			}
			return "", regionNone, fmt.Errorf("%s: %s target must have shape func %s() -> i32 throws %s", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), target, displayTypeName(errorType, module))
		}
		if targetSig.Async {
			return "", regionNone, fmt.Errorf("%s: %s target must be synchronous", frontend.FormatPos(e.At), taskTypedSpawnName(resolved))
		}
		if handleInfo.SlotCount > 4 {
			if targetSig.ThrowsType != "" && targetSig.ThrowsType != errorType {
				return "", regionNone, fmt.Errorf("%s: %s target must throw '%s' (or be non-throwing in staged mode)", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), displayTypeName(errorType, module))
			}
		} else if targetSig.ThrowsType != errorType {
			return "", regionNone, fmt.Errorf("%s: %s target must throw '%s'", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), displayTypeName(errorType, module))
		}
		if targetSig.TouchesMutableGlobals {
			return "", regionNone, fmt.Errorf("%s: %s target '%s' touches mutable global state and cannot cross task boundary", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), target)
		}
		if blocked := actorTaskWorkerBoundaryEffect(targetSig); blocked != "" {
			return "", regionNone, fmt.Errorf("%s: %s target '%s' uses effect '%s' and cannot cross task boundary", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), target, blocked)
		}
		if !funcSigActorTaskTransferSafe(targetSig, types) {
			return "", regionNone, fmt.Errorf("%s: %s target '%s' is not sendable across task boundary", frontend.FormatPos(e.At), taskTypedSpawnName(resolved), target)
		}
		lit.Value = []byte(target)
		e.Name = resolved
		return handleType, regionNone, nil
	case "core.task_join_i32_typed", "core.task_join_group_i32_typed":
		if len(e.Args) != 1 {
			return "", regionNone, fmt.Errorf("%s: task_join_i32_typed expects 1 argument", frontend.FormatPos(e.At))
		}
		argType, _, err := checkExprWithEffects(e.Args[0], locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return "", regionNone, err
		}
		if !typesCompatibleWithNullPtr(handleType, argType, e.Args[0]) {
			return "", regionNone, fmt.Errorf("%s: type mismatch for '%s' arg 1: expected '%s', got '%s'", frontend.FormatPos(e.Args[0].Pos()), resolved, handleType, argType)
		}
		isTryCall := state != nil && state.allowThrowDepth > 0 && state.allowThrowCall == e
		isCatchCall := state != nil && state.allowCatchDepth > 0 && state.allowCatchCall == e
		if !isTryCall && !isCatchCall {
			return "", regionNone, fmt.Errorf("%s: call to throwing function '%s' requires try", frontend.FormatPos(e.At), resolved)
		}
		if isTryCall && state.throwType == "" {
			return "", regionNone, fmt.Errorf("%s: try is only allowed in throwing functions", frontend.FormatPos(e.At))
		}
		if isTryCall && state.throwType != errorType {
			return "", regionNone, fmt.Errorf("%s: thrown error type mismatch: expected '%s', got '%s'", frontend.FormatPos(e.At), state.throwType, errorType)
		}
		markTaskHandleJoined(e.Args[0], funcs, module, imports, state)
		e.Name = resolved
		return "i32", regionNone, nil
	default:
		return "", regionNone, fmt.Errorf("%s: unknown typed task builtin '%s'", frontend.FormatPos(e.At), resolved)
	}
}
