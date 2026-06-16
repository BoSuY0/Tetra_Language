package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

func resolveCheckedCallName(name string, funcs map[string]FuncSig, module string, imports map[string]string, pos frontend.Position) (string, error) {
	if builtin, ok := ResolveBuiltinAlias(name); ok {
		return builtin, nil
	}
	return resolveKnownCallName(name, funcs, module, imports, pos)
}

func paramDeclOwnership(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		out = append(out, param.Ownership)
	}
	return out
}

func validateFunctionTypeNamedSymbolBinding(
	name string,
	declared frontend.TypeRef,
	init *frontend.IdentExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	allowCapturedAlias bool,
	genericErrorOverride ...func(frontend.Position, string, string) error,
) (string, error) {
	genericError := unsupportedGenericFunctionTypedLocalInitializerError
	if len(genericErrorOverride) > 0 && genericErrorOverride[0] != nil {
		genericError = genericErrorOverride[0]
	}
	if declared.Kind != frontend.TypeRefFunction {
		return "", nil
	}
	if init == nil {
		return "", unsupportedFunctionTypedLocalInitializerSourceError(declared.At, name)
	}
	if localInfo, ok := locals[init.Name]; ok {
		if !localInfo.FunctionTypeValue && localInfo.FunctionValue != "" {
			return validateFunctionTypeClosurePointerBinding(name, declared, init, localInfo, funcs, types, module, imports, genericError)
		}
		if len(localInfo.FunctionCaptures) > 0 {
			if !allowCapturedAlias {
				return "", unsupportedFunctionTypedCaptureAliasError(init.At, name, init.Name)
			}
			captureSlots, err := functionCaptureSlotCount(localInfo.FunctionCaptures, types)
			if err != nil {
				return "", err
			}
			if captureSlots < 1 {
				return "", unsupportedFunctionTypedStorageCaptureError(init.At, name, captureSlots)
			}
			if captureSlots > FnPtrEnvSlotCount && !localInfo.FunctionHandleValue {
				if _, _, err := classifyCallableEscape(callableBoundaryLocal, localInfo.FunctionCaptures, types); err != nil {
					return "", err
				}
			}
			if !localInfo.FunctionTypeValue {
				return "", unsupportedFunctionTypedCaptureAliasError(init.At, name, init.Name)
			}
		}
		if !localInfo.FunctionTypeValue {
			return "", unsupportedFunctionTypedLocalInitializerSourceError(init.At, name)
		}
		if localInfo.FunctionValue == "" {
			validationSig := FuncSig{
				ParamTypes:      append([]string(nil), localInfo.FunctionParamTypes...),
				ParamOwnership:  append([]string(nil), localInfo.FunctionParamOwnership...),
				ReturnType:      localInfo.FunctionReturnType,
				ReturnOwnership: localInfo.FunctionReturnOwnership,
				ThrowsType:      localInfo.FunctionThrowsType,
				Effects:         append([]string(nil), localInfo.FunctionEffects...),
			}
			if err := validateFunctionTypeSymbolSignature(name, declared, validationSig, module, imports, init.At); err != nil {
				return "", err
			}
			return "", nil
		}
		sig, ok := funcs[localInfo.FunctionValue]
		if !ok {
			return "", fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(init.At), localInfo.FunctionValue)
		}
		if localInfo.GenericFunctionValue || sig.Generic {
			return "", genericError(init.At, init.Name, name)
		}
		if sig.ThrowsType != "" && localInfo.FunctionReturnType != "" && declared.Throws == nil {
			return "", unsupportedThrowingFunctionTypedLocalInitializerError(init.At, init.Name, name)
		}
		validationSig := sig
		if localInfo.FunctionReturnType != "" {
			explicitSlots, err := functionParamSlotCount(localInfo.FunctionParamTypes, types)
			if err != nil {
				return "", err
			}
			hiddenSlots := sig.ParamSlots - explicitSlots
			if hiddenSlots < 0 || (hiddenSlots > FnPtrEnvSlotCount && !localInfo.FunctionHandleValue) {
				return "", unsupportedFunctionTypedCaptureAliasError(init.At, name, init.Name)
			}
			if hiddenSlots > 0 && !allowCapturedAlias {
				return "", unsupportedFunctionTypedCaptureAliasError(init.At, name, init.Name)
			}
			validationSig.ParamTypes = append([]string(nil), localInfo.FunctionParamTypes...)
			validationSig.ParamOwnership = append([]string(nil), localInfo.FunctionParamOwnership...)
			validationSig.ReturnType = localInfo.FunctionReturnType
			validationSig.ReturnOwnership = localInfo.FunctionReturnOwnership
		}
		if err := validateFunctionTypeSymbolSignature(name, declared, validationSig, module, imports, init.At); err != nil {
			return "", err
		}
		if localInfo.Mutable {
			return "", nil
		}
		return localInfo.FunctionValue, nil
	}
	if globalInfo, ok := globals[init.Name]; ok {
		if !globalInfo.FunctionTypeValue || globalInfo.FunctionValue == "" {
			return "", unsupportedFunctionTypedLocalInitializerSourceError(init.At, name)
		}
		sig, ok := funcs[globalInfo.FunctionValue]
		if !ok {
			return "", fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(init.At), globalInfo.FunctionValue)
		}
		if sig.Generic {
			return "", genericError(init.At, init.Name, name)
		}
		if sig.ThrowsType != "" && declared.Throws == nil {
			return "", unsupportedThrowingFunctionTypedLocalInitializerError(init.At, init.Name, name)
		}
		if err := validateFunctionTypeSymbolSignature(name, declared, sig, module, imports, init.At); err != nil {
			return "", err
		}
		if globalInfo.Mutable {
			return "", nil
		}
		return globalInfo.FunctionValue, nil
	}
	resolved, err := resolveCheckedCallName(init.Name, funcs, module, imports, init.At)
	if err != nil {
		return "", unsupportedFunctionTypedLocalInitializerSourceError(init.At, name)
	}
	sig, ok := funcs[resolved]
	if !ok {
		return "", unsupportedFunctionTypedLocalInitializerSourceError(init.At, name)
	}
	if err := ensureFuncVisible(resolved, sig, module, init.At); err != nil {
		return "", err
	}
	if sig.Generic {
		return "", genericError(init.At, init.Name, name)
	}
	if sig.ThrowsType != "" && declared.Throws == nil {
		return "", unsupportedThrowingFunctionTypedLocalInitializerError(init.At, init.Name, name)
	}
	if err := validateFunctionTypeSymbolSignature(name, declared, sig, module, imports, init.At); err != nil {
		return "", err
	}
	return resolved, nil
}

func validateFunctionTypeClosurePointerBinding(
	name string,
	declared frontend.TypeRef,
	init *frontend.IdentExpr,
	localInfo LocalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	genericError func(frontend.Position, string, string) error,
) (string, error) {
	if genericError == nil {
		genericError = unsupportedGenericFunctionTypedLocalInitializerError
	}
	sig, ok := funcs[localInfo.FunctionValue]
	if !ok {
		return "", fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(init.At), localInfo.FunctionValue)
	}
	if localInfo.GenericFunctionValue || sig.Generic {
		return "", genericError(init.At, init.Name, name)
	}
	if sig.ThrowsType != "" {
		return "", unsupportedThrowingFunctionTypedLocalInitializerError(init.At, init.Name, name)
	}
	visibleSig := sig
	if len(localInfo.FunctionCaptures) > 0 {
		captureSlots, err := functionCaptureSlotCount(localInfo.FunctionCaptures, types)
		if err != nil {
			return "", err
		}
		if captureSlots < 1 {
			return "", unsupportedFunctionTypedStorageCaptureError(init.At, name, captureSlots)
		}
		if captureSlots > FnPtrEnvSlotCount {
			if _, _, err := classifyCallableEscape(callableBoundaryLocal, localInfo.FunctionCaptures, types); err != nil {
				return "", err
			}
		}
		paramTypes, returnType, _, err := functionTypeRefSignatureAndEffects(declared, module, imports)
		if err != nil {
			return "", err
		}
		explicitSlots, err := functionParamSlotCount(paramTypes, types)
		if err != nil {
			return "", err
		}
		if sig.ParamSlots-explicitSlots != captureSlots {
			return "", unsupportedFunctionTypedCaptureAliasError(init.At, name, init.Name)
		}
		visibleSig.ParamTypes = paramTypes
		visibleSig.ParamOwnership = functionTypeRefParamOwnership(declared)
		visibleSig.ParamSlots = explicitSlots
		visibleSig.ReturnType = returnType
	}
	if err := validateFunctionTypeSymbolSignature(name, declared, visibleSig, module, imports, init.At); err != nil {
		return "", err
	}
	return localInfo.FunctionValue, nil
}

func validateFunctionTypeClosurePointerAssignment(
	targetName string,
	targetInfo LocalInfo,
	init *frontend.IdentExpr,
	sourceInfo LocalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	boundary callableEscapeBoundary,
) (string, error) {
	sig, ok := funcs[sourceInfo.FunctionValue]
	if !ok {
		return "", fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(init.At), sourceInfo.FunctionValue)
	}
	if sourceInfo.GenericFunctionValue || sig.Generic {
		return "", unsupportedGenericFunctionTypedAssignmentError(init.At, init.Name, targetName)
	}
	if sig.ThrowsType != "" {
		return "", unsupportedThrowingFunctionTypedAssignmentError(init.At, init.Name, targetName)
	}
	visibleSig := sig
	if len(sourceInfo.FunctionCaptures) > 0 {
		captureSlots, err := functionCaptureSlotCount(sourceInfo.FunctionCaptures, types)
		if err != nil {
			return "", err
		}
		if captureSlots < 1 {
			return "", unsupportedFunctionTypedStorageCaptureError(init.At, targetName, captureSlots)
		}
		if captureSlots > FnPtrEnvSlotCount {
			if _, _, err := classifyCallableEscape(boundary, sourceInfo.FunctionCaptures, types); err != nil {
				return "", err
			}
		}
		explicitSlots, err := functionParamSlotCount(targetInfo.FunctionParamTypes, types)
		if err != nil {
			return "", err
		}
		if sig.ParamSlots-explicitSlots != captureSlots {
			return "", unsupportedFunctionTypedCaptureAliasError(init.At, targetName, init.Name)
		}
		visibleSig.ParamTypes = append([]string(nil), targetInfo.FunctionParamTypes...)
		visibleSig.ParamOwnership = append([]string(nil), targetInfo.FunctionParamOwnership...)
		visibleSig.ParamSlots = explicitSlots
		visibleSig.ReturnType = targetInfo.FunctionReturnType
		visibleSig.ReturnOwnership = targetInfo.FunctionReturnOwnership
	}
	if err := validateFunctionInfoAssignable(targetName, targetInfo, visibleSig, init.At); err != nil {
		return "", err
	}
	return sourceInfo.FunctionValue, nil
}

func validateFunctionTypedAssignmentValue(
	targetName string,
	targetInfo LocalInfo,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	pos frontend.Position,
	allowCapturedLocalStorage bool,
	boundary callableEscapeBoundary,
) error {
	switch v := value.(type) {
	case *frontend.ClosureExpr:
		if err := validateFunctionTypedClosureAssignment(targetName, targetInfo, v, locals, funcs, types, module, imports, pos); err != nil {
			return err
		}
		if allowCapturedLocalStorage && len(v.Captures) > 0 {
			captureSlots, err := functionCaptureSlotCount(v.Captures, types)
			if err != nil {
				return err
			}
			if captureSlots > FnPtrEnvSlotCount {
				if _, _, err := classifyCallableEscape(boundary, v.Captures, types); err != nil {
					return err
				}
				return nil
			}
		}
		if !allowCapturedLocalStorage && len(v.Captures) > 0 {
			return unsupportedGlobalFunctionCaptureStorageError(v.At, targetName)
		}
		return nil
	case *frontend.CallExpr:
		return validateFunctionTypedReturnCallAssignment(targetName, targetInfo, v, locals, globals, funcs, types, module, imports, allowCapturedLocalStorage)
	case *frontend.FieldAccessExpr:
		fieldInfo, ok, err := resolveFunctionFieldArgument(v, locals)
		if err != nil {
			return err
		}
		if ok && !allowCapturedLocalStorage && (len(fieldInfo.FunctionCaptures) > 0 || len(fieldInfo.FunctionEscapeCaptures) > 0) {
			return unsupportedGlobalFunctionCaptureStorageError(v.At, targetName)
		}
		if ok && !allowCapturedLocalStorage && fieldInfo.FunctionValue == "" && functionFieldInfoHasTargetSet(fieldInfo) {
			if fieldInfo.FunctionParamName != "" {
				return unsupportedGlobalFunctionParameterStorageError(v.At, targetName, fieldInfo.FunctionParamName)
			}
			return unsupportedGlobalFunctionCaptureStorageError(v.At, targetName)
		}
		if ok && fieldInfo.FunctionValue == "" && functionFieldInfoHasTargetSet(fieldInfo) && allowCapturedLocalStorage {
			return validateFunctionInfoAssignable(targetName, targetInfo, functionFieldInfoSig(fieldInfo), v.At)
		}
		if !ok || fieldInfo.FunctionValue == "" {
			if _, globalSig, globalOK, err := resolveFunctionTypedGlobalFieldAccess(v, globals, funcs); err != nil {
				return err
			} else if globalOK {
				return validateFunctionInfoAssignable(targetName, targetInfo, globalSig, v.At)
			}
			return unsupportedFunctionTypedAssignmentSourceError(v.At, targetName)
		}
		fieldSig := FuncSig{
			ParamTypes:      append([]string(nil), fieldInfo.FunctionParamTypes...),
			ParamOwnership:  append([]string(nil), fieldInfo.FunctionParamOwnership...),
			ReturnType:      fieldInfo.FunctionReturnType,
			ReturnOwnership: fieldInfo.FunctionReturnOwnership,
			ThrowsType:      fieldInfo.FunctionThrowsType,
			Effects:         append([]string(nil), fieldInfo.FunctionEffects...),
		}
		return validateFunctionInfoAssignable(targetName, targetInfo, fieldSig, v.At)
	}
	id, ok := value.(*frontend.IdentExpr)
	if !ok {
		return unsupportedFunctionTypedAssignmentSourceError(value.Pos(), targetName)
	}
	if sourceInfo, ok := locals[id.Name]; ok {
		if !allowCapturedLocalStorage && (len(sourceInfo.FunctionCaptures) > 0 || len(sourceInfo.FunctionEscapeCaptures) > 0) {
			return unsupportedGlobalFunctionCaptureStorageError(id.At, targetName)
		}
		if !allowCapturedLocalStorage && sourceInfo.FunctionTypeValue && sourceInfo.FunctionValue == "" {
			paramName := sourceInfo.FunctionParamName
			if paramName == "" {
				paramName = id.Name
			}
			return unsupportedGlobalFunctionParameterStorageError(id.At, targetName, paramName)
		}
		if !sourceInfo.FunctionTypeValue && sourceInfo.FunctionValue != "" {
			if _, err := validateFunctionTypeClosurePointerAssignment(targetName, targetInfo, id, sourceInfo, funcs, types, boundary); err != nil {
				return err
			}
			return nil
		}
		if sourceInfo.FunctionTypeValue && sourceInfo.FunctionValue == "" && allowCapturedLocalStorage {
			sourceSig := FuncSig{
				ParamTypes:      append([]string(nil), sourceInfo.FunctionParamTypes...),
				ParamOwnership:  append([]string(nil), sourceInfo.FunctionParamOwnership...),
				ReturnType:      sourceInfo.FunctionReturnType,
				ReturnOwnership: sourceInfo.FunctionReturnOwnership,
				ThrowsType:      sourceInfo.FunctionThrowsType,
				Effects:         append([]string(nil), sourceInfo.FunctionEffects...),
			}
			return validateFunctionInfoAssignable(targetName, targetInfo, sourceSig, id.At)
		}
		if !sourceInfo.FunctionTypeValue || sourceInfo.FunctionValue == "" {
			return unsupportedFunctionTypedAssignmentSourceError(id.At, targetName)
		}
		sourceSig := FuncSig{
			ParamTypes:      append([]string(nil), sourceInfo.FunctionParamTypes...),
			ParamOwnership:  append([]string(nil), sourceInfo.FunctionParamOwnership...),
			ReturnType:      sourceInfo.FunctionReturnType,
			ReturnOwnership: sourceInfo.FunctionReturnOwnership,
			ThrowsType:      sourceInfo.FunctionThrowsType,
			Effects:         append([]string(nil), sourceInfo.FunctionEffects...),
		}
		return validateFunctionInfoAssignable(targetName, targetInfo, sourceSig, id.At)
	}
	if globalInfo, ok := globals[id.Name]; ok {
		if !globalInfo.FunctionTypeValue || globalInfo.FunctionValue == "" {
			return unsupportedFunctionTypedAssignmentSourceError(id.At, targetName)
		}
		sig, ok := funcs[globalInfo.FunctionValue]
		if !ok {
			return fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(id.At), globalInfo.FunctionValue)
		}
		if sig.Generic {
			return unsupportedGenericFunctionTypedAssignmentError(pos, id.Name, targetName)
		}
		return validateFunctionInfoAssignable(targetName, targetInfo, sig, id.At)
	}
	resolved, err := resolveCheckedCallName(id.Name, funcs, module, imports, id.At)
	if err != nil {
		return unsupportedFunctionTypedAssignmentSourceError(id.At, targetName)
	}
	sig, ok := funcs[resolved]
	if !ok {
		return unsupportedFunctionTypedAssignmentSourceError(id.At, targetName)
	}
	if err := ensureFuncVisible(resolved, sig, module, id.At); err != nil {
		return err
	}
	if sig.Generic {
		return unsupportedGenericFunctionTypedAssignmentError(pos, id.Name, targetName)
	}
	if err := validateFunctionInfoAssignable(targetName, targetInfo, sig, id.At); err != nil {
		return err
	}
	id.Name = resolved
	return nil
}

func allowCapturedGlobalFunctionSnapshot(value frontend.Expr, locals map[string]LocalInfo, types map[string]*TypeInfo, state *regionState) (bool, error) {
	if closure, ok := value.(*frontend.ClosureExpr); ok {
		if err := rejectMutableGlobalFunctionCaptures(closure.Captures, locals); err != nil {
			return false, err
		}
		if err := rejectBorrowedFunctionCaptures(closure.Captures, state); err != nil {
			return false, err
		}
		if _, _, err := classifyCallableEscape(callableBoundaryGlobal, closure.Captures, types); err != nil {
			return false, err
		}
		return true, nil
	}
	if field, ok := value.(*frontend.FieldAccessExpr); ok {
		fieldInfo, found, err := resolveFunctionFieldArgument(field, locals)
		if err != nil || !found {
			return false, err
		}
		return allowFunctionFieldGlobalSnapshot(fieldInfo, value.Pos(), locals, types, state)
	}
	id, ok := value.(*frontend.IdentExpr)
	if !ok {
		return false, nil
	}
	source, ok := locals[id.Name]
	if !ok {
		return false, nil
	}
	if source.FunctionParamName != "" || source.FunctionValue == "" {
		return false, nil
	}
	if !source.FunctionTypeValue &&
		len(source.FunctionCaptures) > 0 &&
		len(source.FunctionEscapeCaptures) == 0 {
		if err := rejectMutableGlobalFunctionCaptures(source.FunctionCaptures, locals); err != nil {
			return false, err
		}
		if err := rejectBorrowedFunctionCaptures(source.FunctionCaptures, state); err != nil {
			return false, err
		}
		captureSlots, err := functionCaptureSlotCount(source.FunctionCaptures, types)
		if err != nil {
			return false, err
		}
		if captureSlots < 1 {
			return false, nil
		}
		if captureSlots > FnPtrEnvSlotCount {
			if _, _, err := classifyCallableEscape(callableBoundaryGlobal, source.FunctionCaptures, types); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	if source.FunctionTypeValue &&
		source.FunctionDirectSnapshotAlias &&
		len(source.FunctionCaptures) > 0 &&
		len(source.FunctionEscapeCaptures) == 0 {
		if err := rejectMutableGlobalFunctionCaptures(source.FunctionCaptures, locals); err != nil {
			return false, err
		}
		if err := rejectBorrowedFunctionCaptures(source.FunctionCaptures, state); err != nil {
			return false, err
		}
		captureSlots, err := functionCaptureSlotCount(source.FunctionCaptures, types)
		if err != nil {
			return false, err
		}
		if captureSlots < 1 {
			return false, nil
		}
		if captureSlots > FnPtrEnvSlotCount {
			if _, _, err := classifyCallableEscape(callableBoundaryGlobal, source.FunctionCaptures, types); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	if source.FunctionTypeValue &&
		source.FunctionReturnSnapshotAlias &&
		len(source.FunctionCaptures) == 0 &&
		len(source.FunctionEscapeCaptures) > 0 {
		if err := rejectMutableGlobalFunctionCaptures(source.FunctionEscapeCaptures, locals); err != nil {
			return false, err
		}
		if err := rejectBorrowedFunctionCaptures(source.FunctionEscapeCaptures, state); err != nil {
			return false, err
		}
		captureSlots, err := functionCaptureSlotCount(source.FunctionEscapeCaptures, types)
		if err != nil {
			return false, err
		}
		if captureSlots < 1 {
			return false, nil
		}
		if captureSlots > FnPtrEnvSlotCount {
			if _, _, err := classifyCallableEscape(callableBoundaryGlobal, source.FunctionEscapeCaptures, types); err != nil {
				return false, err
			}
		}
		return true, nil
	}
	return false, nil
}

func rejectBorrowedFunctionCaptures(captures []frontend.ClosureCapture, state *regionState) error {
	if state == nil {
		return nil
	}
	for _, capture := range captures {
		for path, regionID := range state.regionVars {
			if path != capture.Name && !strings.HasPrefix(path, capture.Name+".") {
				continue
			}
			if _, borrowed := state.borrowedParamOwner(regionID); !borrowed {
				continue
			}
			name := path
			if name == "" {
				name = capture.Name
			}
			return lifetimeDiagnosticf(capture.At, "borrowed local '%s' cannot escape via function capture", name)
		}
	}
	return nil
}

func rejectMutableGlobalFunctionCaptures(captures []frontend.ClosureCapture, locals map[string]LocalInfo) error {
	for _, capture := range captures {
		if capture.Mutable {
			return unsupportedGlobalFunctionMutableCaptureStorageError(capture.At, capture.Name)
		}
		if local, ok := locals[capture.Name]; ok && local.Mutable {
			return unsupportedGlobalFunctionMutableCaptureStorageError(capture.At, capture.Name)
		}
	}
	return nil
}

func allowFunctionFieldGlobalSnapshot(info FunctionFieldInfo, pos frontend.Position, locals map[string]LocalInfo, types map[string]*TypeInfo, state *regionState) (bool, error) {
	if info.FunctionParamName != "" || info.FunctionValue == "" {
		return false, nil
	}
	captures := info.FunctionEscapeCaptures
	if !info.FunctionReturnSnapshotAlias && info.FunctionDirectSnapshotAlias {
		captures = info.FunctionCaptures
	}
	if !info.FunctionReturnSnapshotAlias && !info.FunctionDirectSnapshotAlias {
		return false, nil
	}
	if len(captures) == 0 {
		return false, nil
	}
	if info.FunctionReturnSnapshotAlias && len(info.FunctionCaptures) != 0 {
		return false, nil
	}
	if !info.FunctionReturnSnapshotAlias && len(info.FunctionEscapeCaptures) != 0 {
		return false, nil
	}
	if err := rejectMutableGlobalFunctionCaptures(captures, locals); err != nil {
		return false, err
	}
	if err := rejectBorrowedFunctionCaptures(captures, state); err != nil {
		return false, err
	}
	captureSlots, err := functionCaptureSlotCount(captures, types)
	if err != nil {
		return false, err
	}
	if captureSlots < 1 {
		return false, nil
	}
	if captureSlots > FnPtrEnvSlotCount {
		if _, _, err := classifyCallableEscape(callableBoundaryGlobal, captures, types); err != nil {
			return false, err
		}
	}
	return true, nil
}

func unsupportedGlobalFunctionCaptureStorageError(pos frontend.Position, targetName string) error {
	return lifetimeDiagnosticf(pos, "captured function value cannot be stored in global function-typed value '%s'; global escape requires a direct fnptr snapshot with known captures and bounded environment slots", targetName)
}

func unsupportedGlobalFunctionMutableCaptureStorageError(pos frontend.Position, captureName string) error {
	return lifetimeDiagnosticf(pos, "global-escaped function value captures mutable local '%s'; mutable by-reference captures require a proven lifetime and synchronization model", captureName)
}

func unsupportedGlobalFunctionParameterStorageError(pos frontend.Position, targetName, paramName string) error {
	return lifetimeDiagnosticf(pos, "function-typed parameter '%s' cannot be stored in global function-typed value '%s'; global escape requires a direct fnptr snapshot with known captures and bounded environment slots", paramName, targetName)
}

func functionParamNameForParam(name string, functionTypeValue bool) string {
	if functionTypeValue {
		return name
	}
	return ""
}

func functionAssignmentMetadata(
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
) (string, []frontend.ClosureCapture, []frontend.ClosureCapture, string) {
	switch v := value.(type) {
	case *frontend.ClosureExpr:
		return closureFunctionValueName(v, funcs, module), append([]frontend.ClosureCapture(nil), v.Captures...), nil, ""
	case *frontend.CallExpr:
		if callSig, ok := funcs[v.Name]; ok {
			return callSig.ReturnFunctionSymbol, nil, append([]frontend.ClosureCapture(nil), callSig.ReturnFunctionCaptures...), callSig.ReturnFunctionParamName
		}
	case *frontend.FieldAccessExpr:
		if fieldInfo, ok, err := resolveFunctionFieldArgument(v, locals); err == nil && ok {
			return fieldInfo.FunctionValue, append([]frontend.ClosureCapture(nil), fieldInfo.FunctionCaptures...), append([]frontend.ClosureCapture(nil), fieldInfo.FunctionEscapeCaptures...), fieldInfo.FunctionParamName
		}
		if globalInfo, _, ok, err := resolveFunctionTypedGlobalFieldAccess(v, globals, funcs); err == nil && ok {
			return globalInfo.FunctionValue, nil, nil, ""
		}
	case *frontend.IdentExpr:
		if sourceInfo, ok := locals[v.Name]; ok {
			paramName := sourceInfo.FunctionParamName
			if paramName == "" && sourceInfo.FunctionTypeValue && sourceInfo.FunctionValue == "" {
				paramName = v.Name
			}
			return sourceInfo.FunctionValue, append([]frontend.ClosureCapture(nil), sourceInfo.FunctionCaptures...), append([]frontend.ClosureCapture(nil), sourceInfo.FunctionEscapeCaptures...), paramName
		}
		if globalInfo, ok := globals[v.Name]; ok {
			return globalInfo.FunctionValue, nil, nil, ""
		}
		if resolved, err := resolveCheckedCallName(v.Name, funcs, module, imports, v.At); err == nil {
			return resolved, nil, nil, ""
		}
	}
	return "", nil, nil, ""
}

func functionDirectSnapshotAliasForExpr(value frontend.Expr, locals map[string]LocalInfo) bool {
	switch v := value.(type) {
	case *frontend.ClosureExpr:
		return len(v.Captures) > 0
	case *frontend.IdentExpr:
		source, ok := locals[v.Name]
		return ok && source.FunctionDirectSnapshotAlias
	case *frontend.FieldAccessExpr:
		fieldInfo, ok, err := resolveFunctionFieldArgument(v, locals)
		return err == nil && ok && fieldInfo.FunctionDirectSnapshotAlias
	default:
		return false
	}
}

func functionSymbolTouchesMutableGlobals(name string, funcs map[string]FuncSig) bool {
	if name == "" {
		return false
	}
	if sig, ok := funcs[name]; ok {
		return sig.TouchesMutableGlobals
	}
	return false
}

func functionFieldInfoTouchesMutableGlobals(info FunctionFieldInfo, funcs map[string]FuncSig) bool {
	return info.FunctionTouchesMutableGlobals || functionSymbolTouchesMutableGlobals(info.FunctionValue, funcs)
}

func functionAssignmentValueTouchesMutableGlobals(
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (bool, error) {
	switch v := value.(type) {
	case *frontend.ClosureExpr:
		return functionSymbolTouchesMutableGlobals(closureFunctionValueName(v, funcs, module), funcs), nil
	case *frontend.CallExpr:
		if resolved, err := resolveCheckedCallName(v.Name, funcs, module, imports, v.At); err == nil {
			v.Name = resolved
		}
		callSig, ok := funcs[v.Name]
		if !ok || !callSig.ReturnFunctionType {
			return false, nil
		}
		if callSig.ReturnFunctionTouchesMutableGlobals || functionSymbolTouchesMutableGlobals(callSig.ReturnFunctionSymbol, funcs) {
			return true, nil
		}
		if callSig.ReturnFunctionParamName == "" {
			return false, nil
		}
		returnInfo, found, err := functionTypedReturnParamRefMetadata(callSig, callSig.ReturnFunctionParamName, v, locals, globals, funcs, types, module, imports)
		if err != nil || !found {
			return false, err
		}
		return functionFieldInfoTouchesMutableGlobals(returnInfo, funcs), nil
	case *frontend.FieldAccessExpr:
		if fieldInfo, ok, err := resolveFunctionFieldArgument(v, locals); err != nil {
			return false, err
		} else if ok {
			return functionFieldInfoTouchesMutableGlobals(fieldInfo, funcs), nil
		}
		if globalInfo, _, ok, err := resolveFunctionTypedGlobalFieldAccess(v, globals, funcs); err != nil {
			return false, err
		} else if ok {
			return functionSymbolTouchesMutableGlobals(globalInfo.FunctionValue, funcs), nil
		}
	case *frontend.IdentExpr:
		if sourceInfo, ok := locals[v.Name]; ok {
			return sourceInfo.FunctionTouchesMutableGlobals || functionSymbolTouchesMutableGlobals(sourceInfo.FunctionValue, funcs), nil
		}
		if globalInfo, ok := globals[v.Name]; ok {
			return functionSymbolTouchesMutableGlobals(globalInfo.FunctionValue, funcs), nil
		}
		if resolved, err := resolveCheckedCallName(v.Name, funcs, module, imports, v.At); err == nil {
			return functionSymbolTouchesMutableGlobals(resolved, funcs), nil
		}
	}
	return false, nil
}

func functionAssignmentMetadataWithReturnParamRefs(
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, []frontend.ClosureCapture, []frontend.ClosureCapture, string, error) {
	functionValue, captures, escapeCaptures, functionParamName := functionAssignmentMetadata(value, locals, globals, funcs, module, imports)
	call, ok := value.(*frontend.CallExpr)
	if !ok {
		return functionValue, captures, escapeCaptures, functionParamName, nil
	}
	if resolved, err := resolveCheckedCallName(call.Name, funcs, module, imports, call.At); err == nil {
		call.Name = resolved
	}
	callSig, ok := funcs[call.Name]
	if !ok || callSig.ReturnFunctionParamName == "" {
		if ok && callSig.ReturnFunctionType && callSig.ReturnFunctionSymbol == "" {
			fallbackValue, fallbackCaptures, fallbackEscapeCaptures, fallbackParamName, err := functionTypedReturnUnknownParamCaptureMetadata(callSig, call, locals, globals, funcs, types, module, imports)
			if err != nil {
				return functionValue, captures, escapeCaptures, functionParamName, err
			}
			if len(fallbackCaptures) > 0 || len(fallbackEscapeCaptures) > 0 {
				if fallbackValue != "" {
					functionValue = fallbackValue
				}
				captures = append([]frontend.ClosureCapture(nil), fallbackCaptures...)
				escapeCaptures = append(escapeCaptures, fallbackEscapeCaptures...)
				functionParamName = fallbackParamName
			}
		}
		return functionValue, captures, escapeCaptures, functionParamName, nil
	}
	returnInfo, found, err := functionTypedReturnParamRefMetadata(callSig, callSig.ReturnFunctionParamName, call, locals, globals, funcs, types, module, imports)
	if err != nil || !found {
		return functionValue, captures, escapeCaptures, functionParamName, err
	}
	if returnInfo.FunctionValue != "" {
		functionValue = returnInfo.FunctionValue
	}
	functionParamName = returnInfo.FunctionParamName
	captures = append([]frontend.ClosureCapture(nil), returnInfo.FunctionCaptures...)
	escapeCaptures = append(escapeCaptures, returnInfo.FunctionEscapeCaptures...)
	return functionValue, captures, escapeCaptures, functionParamName, nil
}

func functionTypedReturnUnknownParamCaptureMetadata(
	callSig FuncSig,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, []frontend.ClosureCapture, []frontend.ClosureCapture, string, error) {
	functionValue := ""
	var captures []frontend.ClosureCapture
	var escapeCaptures []frontend.ClosureCapture
	functionParamName := ""
	for i, functionParam := range callSig.ParamFunctionTypes {
		if !functionParam || i >= len(call.Args) {
			continue
		}
		argCaptures, argEscapeCaptures, err := functionTypedCallArgumentCaptureMetadata(callSig, i, call.Args[i], locals, globals, funcs, types, module, imports)
		if err != nil {
			return "", nil, nil, "", err
		}
		if len(argCaptures) == 0 && len(argEscapeCaptures) == 0 {
			continue
		}
		argValue, _, _, argParamName := functionAssignmentMetadata(call.Args[i], locals, globals, funcs, module, imports)
		if functionValue == "" {
			functionValue = argValue
		}
		if functionParamName == "" {
			functionParamName = argParamName
		}
		captures = append(captures, argCaptures...)
		escapeCaptures = append(escapeCaptures, argEscapeCaptures...)
	}
	return functionValue, captures, escapeCaptures, functionParamName, nil
}

func updateFunctionTypedLocalAssignmentMetadata(
	name string,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	localInfo, ok := locals[name]
	if !ok {
		return nil
	}
	functionValue, captures, escapeCaptures, functionParamName, err := functionAssignmentMetadataWithReturnParamRefs(value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return err
	}
	escapeKind, handleValue, err := functionAssignmentEscapeMetadata(value, locals, funcs, types, module, imports, callableBoundaryLocal)
	if err != nil {
		return err
	}
	touchesMutableGlobals, err := functionAssignmentValueTouchesMutableGlobals(value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return err
	}
	localInfo.FunctionValue = functionValue
	localInfo.FunctionParamName = functionParamName
	localInfo.FunctionCaptures = captures
	localInfo.FunctionEscapeCaptures = escapeCaptures
	localInfo.FunctionTouchesMutableGlobals = touchesMutableGlobals
	localInfo.FunctionReturnSnapshotAlias = isFunctionReturnSnapshotAlias(value, funcs, captures, escapeCaptures, functionParamName)
	localInfo.FunctionDirectSnapshotAlias = functionDirectSnapshotAliasForExpr(value, locals)
	localInfo.FunctionEscapeKind = escapeKind
	localInfo.FunctionHandleValue = handleValue
	locals[name] = localInfo
	return nil
}

func functionAssignmentEscapeMetadata(
	value frontend.Expr,
	locals map[string]LocalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	boundary callableEscapeBoundary,
) (CallableEscapeKind, bool, error) {
	switch v := value.(type) {
	case *frontend.ClosureExpr:
		if len(v.Captures) == 0 {
			return "", false, nil
		}
		captureSlots, err := functionCaptureSlotCount(v.Captures, types)
		if err != nil {
			return "", false, err
		}
		if captureSlots > FnPtrEnvSlotCount {
			return classifyCallableEscape(boundary, v.Captures, types)
		}
	case *frontend.IdentExpr:
		if source, ok := locals[v.Name]; ok {
			return source.FunctionEscapeKind, source.FunctionHandleValue, nil
		}
	case *frontend.FieldAccessExpr:
		fieldInfo, ok, err := resolveFunctionFieldArgument(v, locals)
		if err != nil {
			return "", false, err
		}
		if ok {
			return fieldInfo.FunctionEscapeKind, fieldInfo.FunctionHandleValue, nil
		}
	case *frontend.CallExpr:
		resolved, err := resolveCheckedCallName(v.Name, funcs, module, imports, v.At)
		if err != nil {
			return "", false, nil
		}
		if callSig, ok := funcs[resolved]; ok && callSig.ReturnFunctionHandleValue {
			return callSig.ReturnFunctionEscapeKind, callSig.ReturnFunctionHandleValue, nil
		}
	}
	return "", false, nil
}

func isFunctionReturnSnapshotAlias(
	value frontend.Expr,
	funcs map[string]FuncSig,
	captures []frontend.ClosureCapture,
	escapeCaptures []frontend.ClosureCapture,
	functionParamName string,
) bool {
	call, ok := value.(*frontend.CallExpr)
	if !ok {
		return false
	}
	callSig, ok := funcs[call.Name]
	return ok &&
		callSig.ReturnFunctionType &&
		callSig.ReturnFunctionParamName == "" &&
		len(captures) == 0 &&
		len(escapeCaptures) > 0 &&
		functionParamName == ""
}

func updateFunctionTypedFieldAssignmentMetadata(
	targetName string,
	targetInfo LocalInfo,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	parts := strings.Split(targetName, ".")
	if len(parts) < 2 {
		return nil
	}
	base := parts[0]
	fieldPath := strings.Join(parts[1:], ".")
	localInfo, ok := locals[base]
	if !ok {
		return nil
	}
	if localInfo.FunctionFields == nil {
		localInfo.FunctionFields = map[string]FunctionFieldInfo{}
	}
	functionValue, captures, escapeCaptures, functionParamName, err := functionAssignmentMetadataWithReturnParamRefs(value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return err
	}
	escapeKind, handleValue, err := functionAssignmentEscapeMetadata(value, locals, funcs, types, module, imports, callableBoundaryStructField)
	if err != nil {
		return err
	}
	touchesMutableGlobals, err := functionAssignmentValueTouchesMutableGlobals(value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return err
	}
	localInfo.FunctionFields[fieldPath] = FunctionFieldInfo{
		FunctionValue:                 functionValue,
		FunctionParamName:             functionParamName,
		FunctionCaptures:              captures,
		FunctionEscapeCaptures:        escapeCaptures,
		FunctionTouchesMutableGlobals: touchesMutableGlobals,
		FunctionReturnSnapshotAlias:   isFunctionReturnSnapshotAlias(value, funcs, captures, escapeCaptures, functionParamName),
		FunctionDirectSnapshotAlias:   functionDirectSnapshotAliasForExpr(value, locals),
		FunctionEscapeKind:            escapeKind,
		FunctionHandleValue:           handleValue,
		FunctionParamTypes:            append([]string(nil), targetInfo.FunctionParamTypes...),
		FunctionParamOwnership:        append([]string(nil), targetInfo.FunctionParamOwnership...),
		FunctionReturnType:            targetInfo.FunctionReturnType,
		FunctionReturnOwnership:       targetInfo.FunctionReturnOwnership,
		FunctionThrowsType:            targetInfo.FunctionThrowsType,
		FunctionEffects:               append([]string(nil), targetInfo.FunctionEffects...),
	}
	locals[base] = localInfo
	return nil
}

func updateFunctionTypedStructFieldAssignmentMetadata(
	target frontend.Expr,
	targetType string,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	base, fields, _, ok := splitFieldPath(target)
	if !ok || len(fields) == 0 {
		return nil
	}
	localInfo, ok := locals[base]
	if !ok || !localInfo.Mutable {
		return nil
	}
	fieldsForValue, err := functionFieldsFromReturnedStructExpr(targetType, value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return err
	}
	prefix := strings.Join(fields, ".")
	prefixWithDot := prefix + "."
	hadExisting := false
	for fieldName := range localInfo.FunctionFields {
		if fieldName == prefix || strings.HasPrefix(fieldName, prefixWithDot) {
			hadExisting = true
			break
		}
	}
	if len(fieldsForValue) == 0 && !hadExisting {
		return nil
	}
	updated := cloneFunctionFieldMap(localInfo.FunctionFields)
	if updated == nil {
		updated = map[string]FunctionFieldInfo{}
	}
	for fieldName := range updated {
		if fieldName == prefix || strings.HasPrefix(fieldName, prefixWithDot) {
			delete(updated, fieldName)
		}
	}
	for fieldName, fieldInfo := range fieldsForValue {
		updated[prefixWithDot+fieldName] = cloneFunctionFieldInfo(fieldInfo)
	}
	localInfo.FunctionFields = updated
	locals[base] = localInfo
	return nil
}

func updateEnumPayloadStructFieldAssignmentMetadata(
	target frontend.Expr,
	targetType string,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	base, fields, _, ok := splitFieldPath(target)
	if !ok || len(fields) == 0 {
		return nil
	}
	localInfo, ok := locals[base]
	if !ok || !localInfo.Mutable {
		return nil
	}
	info, ok := types[targetType]
	if !ok {
		return nil
	}
	prefix := strings.Join(fields, ".")
	updates := map[string]FunctionFieldInfo{}
	switch info.Kind {
	case TypeEnum:
		payloads, err := enumPayloadFunctionsFromConstructor(info, value, locals, globals, funcs, types, module, imports)
		if err != nil {
			return err
		}
		if len(payloads) == 0 {
			payloads = enumPayloadFunctionsFromAlias(value, locals)
		}
		if len(payloads) == 0 {
			var err error
			payloads, err = enumPayloadFunctionsFromReturnCall(value, locals, globals, funcs, types, module, imports, targetType)
			if err != nil {
				return err
			}
		}
		for payloadKey, payload := range payloads {
			updates[enumPayloadFieldKey(prefix, payloadKey)] = cloneFunctionFieldInfo(payload)
		}
	case TypeStruct:
		fieldsForValue, err := enumPayloadFieldsFromReturnedStructExpr(targetType, value, locals, globals, funcs, types, module, imports)
		if err != nil {
			return err
		}
		for fieldName, fieldInfo := range fieldsForValue {
			updates[prefix+"."+fieldName] = cloneFunctionFieldInfo(fieldInfo)
		}
	default:
		return nil
	}
	hadExisting := false
	for fieldName := range localInfo.EnumPayloadFields {
		if enumPayloadFieldMatchesPrefix(fieldName, prefix) {
			hadExisting = true
			break
		}
	}
	if len(updates) == 0 && !hadExisting {
		return nil
	}
	updated := cloneFunctionFieldMap(localInfo.EnumPayloadFields)
	if updated == nil {
		updated = map[string]FunctionFieldInfo{}
	}
	for fieldName := range updated {
		if enumPayloadFieldMatchesPrefix(fieldName, prefix) {
			delete(updated, fieldName)
		}
	}
	for fieldName, fieldInfo := range updates {
		updated[fieldName] = cloneFunctionFieldInfo(fieldInfo)
	}
	localInfo.EnumPayloadFields = updated
	locals[base] = localInfo
	return nil
}

func validateFunctionTypedClosureAssignment(
	targetName string,
	targetInfo LocalInfo,
	closure *frontend.ClosureExpr,
	locals map[string]LocalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	pos frontend.Position,
	contextOverride ...string,
) error {
	context := "function-typed assignment"
	if len(contextOverride) > 0 && contextOverride[0] != "" {
		context = contextOverride[0]
	}
	targetPhrase := functionTypedClosureAssignmentTargetPhrase(context, targetName)
	if closure == nil || closure.Decl == nil {
		return fmt.Errorf("%s: %s must use a closure literal with a body", frontend.FormatPos(pos), targetPhrase)
	}
	if len(closure.Decl.TypeParams) > 0 {
		return fmt.Errorf("%s: generic closure literals are not supported for %s in this MVP", frontend.FormatPos(closure.At), targetPhrase)
	}
	explicitParams := explicitClosureParams(closure)
	if len(explicitParams) != len(targetInfo.FunctionParamTypes) {
		return fmt.Errorf("%s: %s parameter count mismatch: expected %d, got %d", frontend.FormatPos(closure.At), targetPhrase, len(targetInfo.FunctionParamTypes), len(explicitParams))
	}
	closureSig := FuncSig{
		ParamTypes:      make([]string, 0, len(explicitParams)),
		ParamOwnership:  paramDeclOwnership(explicitParams),
		ReturnType:      "",
		ReturnOwnership: closure.Decl.ReturnOwnership,
	}
	for _, param := range explicitParams {
		typeName, err := resolveTypeName(&param.Type, module, imports)
		if err != nil {
			return err
		}
		closureSig.ParamTypes = append(closureSig.ParamTypes, typeName)
	}
	returnType, err := resolveTypeName(&closure.Decl.ReturnType, module, imports)
	if err != nil {
		return err
	}
	closureSig.ReturnType = returnType
	if closure.Decl.HasThrows {
		throwsType, err := resolveTypeName(&closure.Decl.Throws, module, imports)
		if err != nil {
			return err
		}
		closureSig.ThrowsType = throwsType
	}
	closureEffects, err := normalizeEffects(closure.Decl.Uses, closure.Decl.Pos)
	if err != nil {
		return err
	}
	closureSig.Effects = closureEffects
	if err := validateFunctionInfoAssignableWithContext(targetName, targetInfo, closureSig, closure.At, context); err != nil {
		return err
	}
	return configureClosureCaptures(closure, locals, funcs, types, module, true, functionTypedClosureCaptureBoundaryPhrase(context, targetName))
}

func functionTypedClosureAssignmentTargetPhrase(context, targetName string) string {
	if context == "" || context == "function-typed assignment" {
		return fmt.Sprintf("function-typed assignment to '%s'", targetName)
	}
	return fmt.Sprintf("%s '%s'", context, targetName)
}
