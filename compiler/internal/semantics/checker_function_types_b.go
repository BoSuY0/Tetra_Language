package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	semanticsfunctiontypes "tetra_language/compiler/internal/semantics/functiontypes"
)

func functionTypedClosureCaptureBoundaryPhrase(context, targetName string) string {
	if context == "callback argument" {
		return fmt.Sprintf("callback argument '%s'", targetName)
	}
	if targetName == "return" {
		return "function-typed return 'closure literal'"
	}
	if context == "" || context == "function-typed assignment" {
		return fmt.Sprintf("function-typed storage '%s'", targetName)
	}
	return fmt.Sprintf("%s '%s'", context, targetName)
}

func validateFunctionTypedReturnCallAssignment(
	targetName string,
	targetInfo LocalInfo,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	allowCapturedLocalStorage bool,
) error {
	resolvedCall, err := resolveCheckedCallName(call.Name, funcs, module, imports, call.At)
	if err != nil {
		return unsupportedFunctionTypedAssignmentReturnCallSourceError(call.At, targetName, call.Name)
	}
	call.Name = resolvedCall
	callSig, ok := funcs[resolvedCall]
	if !ok || !callSig.ReturnFunctionType {
		return unsupportedFunctionTypedAssignmentReturnCallSourceError(call.At, targetName, call.Name)
	}
	if callSig.ReturnFunctionSymbol != "" {
		targetSig, ok := funcs[callSig.ReturnFunctionSymbol]
		if !ok {
			return fmt.Errorf("%s: unknown returned function symbol '%s'", frontend.FormatPos(call.At), callSig.ReturnFunctionSymbol)
		}
		if targetSig.Generic {
			return unsupportedGenericFunctionTypedAssignmentError(call.At, callSig.ReturnFunctionSymbol, targetName)
		}
	}
	if !allowCapturedLocalStorage && len(callSig.ReturnFunctionCaptures) > 0 {
		if err := rejectMutableGlobalFunctionCaptures(callSig.ReturnFunctionCaptures, nil); err != nil {
			return err
		}
		captureSlots, err := functionCaptureSlotCount(callSig.ReturnFunctionCaptures, types)
		if err != nil {
			return err
		}
		if captureSlots < 1 || captureSlots > FnPtrEnvSlotCount {
			return unsupportedFunctionTypedStorageCaptureError(call.At, targetName, captureSlots)
		}
	}
	if !allowCapturedLocalStorage && callSig.ReturnFunctionParamName != "" {
		returnInfo, found, err := functionTypedReturnParamRefMetadata(callSig, callSig.ReturnFunctionParamName, call, locals, globals, funcs, types, module, imports)
		if err != nil || !found {
			return err
		}
		if returnInfo.FunctionParamName != "" {
			return unsupportedGlobalFunctionParameterStorageError(call.At, targetName, returnInfo.FunctionParamName)
		}
		if len(returnInfo.FunctionCaptures) > 0 || len(returnInfo.FunctionEscapeCaptures) > 0 {
			return unsupportedGlobalFunctionCaptureStorageError(call.At, targetName)
		}
		if returnInfo.FunctionValue == "" && strings.Contains(callSig.ReturnFunctionParamName, "#") {
			return unsupportedGlobalFunctionParameterStorageError(call.At, targetName, callSig.ReturnFunctionParamName)
		}
	}
	if !allowCapturedLocalStorage && callSig.ReturnFunctionSymbol == "" && callSig.ReturnFunctionParamName == "" {
		for i, functionParam := range callSig.ParamFunctionTypes {
			if !functionParam || i >= len(call.Args) {
				continue
			}
			captured, err := functionTypedCallArgumentHasCaptures(callSig, i, call.Args[i], locals, globals, funcs, types, module, imports)
			if err != nil {
				return err
			}
			if captured {
				return unsupportedGlobalFunctionCaptureStorageError(call.At, targetName)
			}
		}
	}
	returnedSig := FuncSig{
		ParamTypes:      append([]string(nil), callSig.ReturnFunctionParams...),
		ParamOwnership:  append([]string(nil), callSig.ReturnFunctionParamOwnership...),
		ReturnType:      callSig.ReturnFunctionReturn,
		ReturnOwnership: callSig.ReturnFunctionReturnOwnership,
		ThrowsType:      callSig.ReturnFunctionThrows,
		Effects:         append([]string(nil), callSig.ReturnFunctionEffects...),
	}
	return validateFunctionInfoAssignable(targetName, targetInfo, returnedSig, call.At)
}

func functionTypedCallArgumentHasCaptures(
	callSig FuncSig,
	index int,
	arg frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (bool, error) {
	captures, escapeCaptures, err := functionTypedCallArgumentCaptureMetadata(callSig, index, arg, locals, globals, funcs, types, module, imports)
	if err != nil {
		return false, err
	}
	return len(captures) > 0 || len(escapeCaptures) > 0, nil
}

func functionTypedReturnParamRefHasCaptures(
	callSig FuncSig,
	paramRef string,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (bool, error) {
	captures, escapeCaptures, err := functionTypedReturnParamRefCaptureMetadata(callSig, paramRef, call, locals, globals, funcs, types, module, imports)
	if err != nil {
		return false, err
	}
	return len(captures) > 0 || len(escapeCaptures) > 0, nil
}

func functionTypedReturnParamRefCaptureMetadata(
	callSig FuncSig,
	paramRef string,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) ([]frontend.ClosureCapture, []frontend.ClosureCapture, error) {
	info, ok, err := functionTypedReturnParamRefMetadata(callSig, paramRef, call, locals, globals, funcs, types, module, imports)
	if err != nil || !ok {
		return nil, nil, err
	}
	return append([]frontend.ClosureCapture(nil), info.FunctionCaptures...), append([]frontend.ClosureCapture(nil), info.FunctionEscapeCaptures...), nil
}

func functionTypedReturnParamRefMetadata(
	callSig FuncSig,
	paramRef string,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (FunctionFieldInfo, bool, error) {
	if paramRef == "" {
		return FunctionFieldInfo{}, false, nil
	}
	for i, name := range callSig.ParamNames {
		if i >= len(call.Args) {
			continue
		}
		if name == paramRef {
			captures, escapeCaptures, err := functionTypedCallArgumentCaptureMetadata(callSig, i, call.Args[i], locals, globals, funcs, types, module, imports)
			if err != nil {
				return FunctionFieldInfo{}, false, err
			}
			functionValue, _, _, functionParamName := functionAssignmentMetadata(call.Args[i], locals, globals, funcs, module, imports)
			if functionValue == "" {
				metadataValue, _, _, metadataParamName, err := functionAssignmentMetadataWithReturnParamRefs(call.Args[i], locals, globals, funcs, types, module, imports)
				if err != nil {
					return FunctionFieldInfo{}, false, err
				}
				functionValue = metadataValue
				if functionParamName == "" {
					functionParamName = metadataParamName
				}
			}
			touchesMutableGlobals, err := functionAssignmentValueTouchesMutableGlobals(call.Args[i], locals, globals, funcs, types, module, imports)
			if err != nil {
				return FunctionFieldInfo{}, false, err
			}
			return FunctionFieldInfo{
				FunctionValue:                 functionValue,
				FunctionParamName:             functionParamName,
				FunctionCaptures:              captures,
				FunctionEscapeCaptures:        escapeCaptures,
				FunctionTouchesMutableGlobals: touchesMutableGlobals,
			}, true, nil
		}
		payloadPrefix := name + "#"
		if strings.HasPrefix(paramRef, payloadPrefix) {
			payloadKey := strings.TrimPrefix(paramRef, payloadPrefix)
			payloadInfo, ok, err := functionEnumPayloadInfoFromCallArgument(payloadKey, callSig, i, call.Args[i], locals, globals, funcs, types, module, imports)
			if err != nil {
				return FunctionFieldInfo{}, false, err
			}
			if !ok {
				return FunctionFieldInfo{}, false, nil
			}
			return payloadInfo, true, nil
		}
		prefix := name + "."
		if !strings.HasPrefix(paramRef, prefix) {
			continue
		}
		fieldPath := strings.TrimPrefix(paramRef, prefix)
		fieldInfo, ok, err := functionFieldInfoFromCallArgument(fieldPath, callSig, i, call.Args[i], locals, globals, funcs, types, module, imports)
		if err != nil {
			return FunctionFieldInfo{}, false, err
		}
		if !ok {
			return FunctionFieldInfo{}, false, nil
		}
		return fieldInfo, true, nil
	}
	return FunctionFieldInfo{}, false, nil
}

func functionFieldInfoFromCallArgument(
	fieldPath string,
	callSig FuncSig,
	index int,
	arg frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (FunctionFieldInfo, bool, error) {
	fields := functionFieldsFromStructAlias(arg, locals)
	if len(fields) == 0 && index >= 0 && index < len(callSig.ParamTypes) {
		if info, ok := types[callSig.ParamTypes[index]]; ok && info.Kind == TypeStruct {
			var err error
			fields, err = functionFieldsFromStructLiteral("<argument>", info, arg, locals, globals, funcs, types, module, imports)
			if err != nil {
				return FunctionFieldInfo{}, false, err
			}
		}
	}
	if len(fields) == 0 && index >= 0 && index < len(callSig.ParamTypes) {
		var err error
		fields, err = functionFieldsFromReturnCall(arg, locals, globals, funcs, types, module, imports, callSig.ParamTypes[index])
		if err != nil {
			return FunctionFieldInfo{}, false, err
		}
	}
	if len(fields) == 0 {
		return FunctionFieldInfo{}, false, nil
	}
	fieldInfo, ok := fields[fieldPath]
	if !ok {
		return FunctionFieldInfo{}, false, nil
	}
	return fieldInfo, true, nil
}

func functionEnumPayloadInfoFromCallArgument(
	payloadKey string,
	callSig FuncSig,
	index int,
	arg frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (FunctionFieldInfo, bool, error) {
	payloads := enumPayloadFunctionsFromAlias(arg, locals)
	if len(payloads) == 0 && index >= 0 && index < len(callSig.ParamTypes) {
		info, ok := types[callSig.ParamTypes[index]]
		if ok && info.Kind == TypeEnum {
			var err error
			payloads, err = enumPayloadFunctionsFromConstructor(info, arg, locals, globals, funcs, types, module, imports)
			if err != nil {
				return FunctionFieldInfo{}, false, err
			}
		}
	}
	if len(payloads) == 0 && index >= 0 && index < len(callSig.ParamTypes) {
		var err error
		payloads, err = enumPayloadFunctionsFromReturnCall(arg, locals, globals, funcs, types, module, imports, callSig.ParamTypes[index])
		if err != nil {
			return FunctionFieldInfo{}, false, err
		}
	}
	if len(payloads) == 0 {
		return FunctionFieldInfo{}, false, nil
	}
	payloadInfo, ok := payloads[payloadKey]
	if !ok {
		return FunctionFieldInfo{}, false, nil
	}
	return payloadInfo, true, nil
}

func functionTypedCallArgumentCaptureMetadata(
	callSig FuncSig,
	index int,
	arg frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) ([]frontend.ClosureCapture, []frontend.ClosureCapture, error) {
	if closure, ok := arg.(*frontend.ClosureExpr); ok {
		paramInfo := functionParamLocalInfo(callSig, index)
		if err := validateFunctionTypedClosureAssignment("closure literal", paramInfo, closure, locals, funcs, types, module, imports, closure.At, "callback argument"); err != nil {
			return nil, nil, err
		}
	}
	if call, ok := arg.(*frontend.CallExpr); ok {
		if resolved, err := resolveCheckedCallName(call.Name, funcs, module, imports, call.At); err == nil {
			call.Name = resolved
		}
	}
	_, captures, escapeCaptures, _, err := functionAssignmentMetadataWithReturnParamRefs(arg, locals, globals, funcs, types, module, imports)
	if err != nil {
		return nil, nil, err
	}
	return captures, escapeCaptures, nil
}

func validateFunctionInfoAssignable(targetName string, targetInfo LocalInfo, sig FuncSig, pos frontend.Position) error {
	return validateFunctionInfoAssignableWithContext(targetName, targetInfo, sig, pos, "function-typed assignment")
}

func validateFunctionInfoAssignableWithContext(targetName string, targetInfo LocalInfo, sig FuncSig, pos frontend.Position, context string) error {
	if len(targetInfo.FunctionParamTypes) != len(sig.ParamTypes) {
		if context == "function-typed assignment" {
			return fmt.Errorf("%s: function-typed assignment to '%s' parameter count mismatch: expected %d, got %d", frontend.FormatPos(pos), targetName, len(targetInfo.FunctionParamTypes), len(sig.ParamTypes))
		}
		return fmt.Errorf("%s: %s '%s' parameter count mismatch: expected %d, got %d", frontend.FormatPos(pos), context, targetName, len(targetInfo.FunctionParamTypes), len(sig.ParamTypes))
	}
	if err := validateFunctionTypeParamOwnership(targetInfo.FunctionParamOwnership, sig.ParamOwnership, len(targetInfo.FunctionParamTypes), pos, context, targetName); err != nil {
		return err
	}
	for i := range targetInfo.FunctionParamTypes {
		if targetInfo.FunctionParamTypes[i] != sig.ParamTypes[i] {
			if context == "function-typed assignment" {
				return fmt.Errorf("%s: function-typed assignment to '%s' parameter %d type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), targetName, i+1, targetInfo.FunctionParamTypes[i], sig.ParamTypes[i])
			}
			return fmt.Errorf("%s: %s '%s' parameter %d type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), context, targetName, i+1, targetInfo.FunctionParamTypes[i], sig.ParamTypes[i])
		}
	}
	if targetInfo.FunctionReturnType != sig.ReturnType {
		if context == "function-typed assignment" {
			return fmt.Errorf("%s: function-typed assignment to '%s' return type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), targetName, targetInfo.FunctionReturnType, sig.ReturnType)
		}
		return fmt.Errorf("%s: %s '%s' return type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), context, targetName, targetInfo.FunctionReturnType, sig.ReturnType)
	}
	if targetInfo.FunctionReturnOwnership != sig.ReturnOwnership {
		if context == "function-typed assignment" {
			return fmt.Errorf("%s: function-typed assignment to '%s' return ownership mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), targetName, ownershipDisplay(targetInfo.FunctionReturnOwnership), ownershipDisplay(sig.ReturnOwnership))
		}
		return fmt.Errorf("%s: %s '%s' return ownership mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), context, targetName, ownershipDisplay(targetInfo.FunctionReturnOwnership), ownershipDisplay(sig.ReturnOwnership))
	}
	if targetInfo.FunctionThrowsType != sig.ThrowsType {
		if context == "function-typed assignment" {
			return fmt.Errorf("%s: function-typed assignment to '%s' throws type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), targetName, targetInfo.FunctionThrowsType, sig.ThrowsType)
		}
		return fmt.Errorf("%s: %s '%s' throws type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), context, targetName, targetInfo.FunctionThrowsType, sig.ThrowsType)
	}
	return validateFunctionTypeCallableEffects(targetInfo.FunctionEffects, sig.Effects, pos, context, targetName)
}

func validateFunctionTypeCallableEffects(declaredEffects []string, targetEffects []string, pos frontend.Position, context, rawName string) error {
	missing := missingRequiredEffects(targetEffects, declaredEffects)
	if len(missing) > 0 {
		return fmt.Errorf("%s: %s '%s' requires effects %s but function type does not declare them", frontend.FormatPos(pos), context, rawName, strings.Join(missing, ", "))
	}
	return nil
}

func validateFunctionTypeParamOwnership(expected []string, actual []string, count int, pos frontend.Position, context, rawName string) error {
	for i := 0; i < count; i++ {
		want := ownershipAt(expected, i)
		got := ownershipAt(actual, i)
		if want != got {
			return ownershipDiagnosticf(pos, "%s '%s' parameter %d ownership mismatch: expected '%s', got '%s'", context, rawName, i+1, ownershipDisplay(want), ownershipDisplay(got))
		}
	}
	return nil
}

func ownershipAt(ownership []string, index int) string {
	if index < 0 || index >= len(ownership) {
		return ""
	}
	return ownership[index]
}

func ownershipDisplay(ownership string) string {
	if ownership == "" {
		return "owned"
	}
	return ownership
}

func validateFunctionTypeSymbolSignature(
	localName string,
	declared frontend.TypeRef,
	sig FuncSig,
	module string,
	imports map[string]string,
	pos frontend.Position,
) error {
	if len(declared.Params) != len(sig.ParamTypes) {
		return fmt.Errorf("%s: function-typed local '%s' parameter count mismatch: expected %d, got %d", frontend.FormatPos(pos), localName, len(declared.Params), len(sig.ParamTypes))
	}
	if err := validateFunctionTypeParamOwnership(functionTypeRefParamOwnership(declared), sig.ParamOwnership, len(declared.Params), pos, "function-typed local", localName); err != nil {
		return err
	}
	for i := range declared.Params {
		want, err := resolveTypeName(&declared.Params[i], module, imports)
		if err != nil {
			return err
		}
		got := sig.ParamTypes[i]
		if want != got {
			return fmt.Errorf("%s: function-typed local '%s' parameter %d type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), localName, i+1, want, got)
		}
	}
	if declared.Return == nil {
		return fmt.Errorf("%s: missing function return type", frontend.FormatPos(declared.At))
	}
	wantRet, err := resolveTypeName(declared.Return, module, imports)
	if err != nil {
		return err
	}
	if wantRet != sig.ReturnType {
		return fmt.Errorf("%s: function-typed local '%s' return type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), localName, wantRet, sig.ReturnType)
	}
	if declared.ReturnOwnership != sig.ReturnOwnership {
		return fmt.Errorf("%s: function-typed local '%s' return ownership mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), localName, ownershipDisplay(declared.ReturnOwnership), ownershipDisplay(sig.ReturnOwnership))
	}
	wantThrows := ""
	if declared.Throws != nil {
		wantThrows, err = resolveTypeName(declared.Throws, module, imports)
		if err != nil {
			return err
		}
	}
	if wantThrows != sig.ThrowsType {
		return fmt.Errorf("%s: function-typed local '%s' throws type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), localName, wantThrows, sig.ThrowsType)
	}
	declaredEffects, err := functionTypeRefEffects(declared, declared.At)
	if err != nil {
		return err
	}
	if err := validateFunctionTypeCallableEffects(declaredEffects, sig.Effects, pos, "function-typed local", localName); err != nil {
		return err
	}
	return nil
}

func validateReturnedFunctionSignature(
	callerSig FuncSig,
	returnedSig FuncSig,
	pos frontend.Position,
	rawName string,
) error {
	if !callerSig.ReturnFunctionType {
		return nil
	}
	if len(callerSig.ReturnFunctionParams) != len(returnedSig.ParamTypes) {
		return fmt.Errorf("%s: returned function symbol '%s' has incompatible parameter count: expected %d, got %d", frontend.FormatPos(pos), rawName, len(callerSig.ReturnFunctionParams), len(returnedSig.ParamTypes))
	}
	if err := validateFunctionTypeParamOwnership(callerSig.ReturnFunctionParamOwnership, returnedSig.ParamOwnership, len(callerSig.ReturnFunctionParams), pos, "returned function symbol", rawName); err != nil {
		return err
	}
	for i := range callerSig.ReturnFunctionParams {
		if callerSig.ReturnFunctionParams[i] != returnedSig.ParamTypes[i] {
			return fmt.Errorf(
				"%s: returned function symbol '%s' parameter %d type mismatch: expected '%s', got '%s'",
				frontend.FormatPos(pos),
				rawName,
				i+1,
				callerSig.ReturnFunctionParams[i],
				returnedSig.ParamTypes[i],
			)
		}
	}
	if callerSig.ReturnFunctionReturn != returnedSig.ReturnType {
		return fmt.Errorf(
			"%s: returned function symbol '%s' return type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			rawName,
			callerSig.ReturnFunctionReturn,
			returnedSig.ReturnType,
		)
	}
	if callerSig.ReturnFunctionReturnOwnership != returnedSig.ReturnOwnership {
		return fmt.Errorf(
			"%s: returned function symbol '%s' return ownership mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			rawName,
			ownershipDisplay(callerSig.ReturnFunctionReturnOwnership),
			ownershipDisplay(returnedSig.ReturnOwnership),
		)
	}
	if callerSig.ReturnFunctionThrows != returnedSig.ThrowsType {
		return fmt.Errorf(
			"%s: returned function symbol '%s' throws type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			rawName,
			callerSig.ReturnFunctionThrows,
			returnedSig.ThrowsType,
		)
	}
	if err := validateFunctionTypeCallableEffects(callerSig.ReturnFunctionEffects, returnedSig.Effects, pos, "returned function symbol", rawName); err != nil {
		return err
	}
	return nil
}

func functionTypeRefSignature(ref frontend.TypeRef, module string, imports map[string]string) ([]string, string, error) {
	if ref.Kind != frontend.TypeRefFunction {
		return nil, "", nil
	}
	paramTypes := make([]string, 0, len(ref.Params))
	for i := range ref.Params {
		tname, err := resolveTypeName(&ref.Params[i], module, imports)
		if err != nil {
			return nil, "", err
		}
		paramTypes = append(paramTypes, tname)
	}
	if ref.Return == nil {
		return nil, "", fmt.Errorf("%s: missing function return type", frontend.FormatPos(ref.At))
	}
	retType, err := resolveTypeName(ref.Return, module, imports)
	if err != nil {
		return nil, "", err
	}
	return paramTypes, retType, nil
}

func functionTypeRefParamOwnership(ref frontend.TypeRef) []string {
	return semanticsfunctiontypes.ParamOwnership(ref)
}

func functionTypeRefReturnOwnership(ref frontend.TypeRef) string {
	return semanticsfunctiontypes.ReturnOwnership(ref)
}

func functionTypeRefEffects(ref frontend.TypeRef, pos frontend.Position) ([]string, error) {
	return semanticsfunctiontypes.Effects(ref, pos, effectDiagnosticf)
}

func functionTypeRefSignatureAndEffects(ref frontend.TypeRef, module string, imports map[string]string) ([]string, string, []string, error) {
	params, ret, err := functionTypeRefSignature(ref, module, imports)
	if err != nil {
		return nil, "", nil, err
	}
	effects, err := functionTypeRefEffects(ref, ref.At)
	if err != nil {
		return nil, "", nil, err
	}
	return params, ret, effects, nil
}

func functionTypeRefThrowsType(ref frontend.TypeRef, module string, imports map[string]string) (string, error) {
	if ref.Kind != frontend.TypeRefFunction || ref.Throws == nil {
		return "", nil
	}
	return resolveTypeName(ref.Throws, module, imports)
}

func cloneBoolMap(src map[string]bool) map[string]bool {
	if len(src) == 0 {
		return make(map[string]bool)
	}
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func borrowedPtrOwnerFromExpr(expr frontend.Expr, state *regionState, borrowedParams map[string]struct{}) (string, bool) {
	id, ok := expr.(*frontend.IdentExpr)
	if ok {
		if _, borrowed := borrowedParams[id.Name]; borrowed {
			return id.Name, true
		}
		if owner, borrowed := state.borrowedPtrAliasOwner(id.Name); borrowed {
			return owner, true
		}
	}
	if path, ok := resourcePathForExpr(expr); ok {
		if owner, borrowed := state.borrowedPtrAliasOwnerInTree(path); borrowed {
			return owner, true
		}
	}
	if field, ok := expr.(*frontend.FieldAccessExpr); ok {
		if owner, borrowed := borrowedPtrOwnerFromExpr(field.Base, state, borrowedParams); borrowed {
			return owner, true
		}
	}
	if index, ok := expr.(*frontend.IndexExpr); ok {
		if owner, borrowed := borrowedPtrOwnerFromExpr(index.Base, state, borrowedParams); borrowed {
			return owner, true
		}
		if source, ok := resourcePathForExpr(index.Base); ok {
			if owner, borrowed := state.borrowedPtrAliasOwnerInTree(source); borrowed {
				return owner, true
			}
		}
	}
	return "", false
}

func bindBorrowedPtrAliasFromExpr(name string, typeName string, expr frontend.Expr, types map[string]*TypeInfo, module string, imports map[string]string, state *regionState, borrowedParams map[string]struct{}) {
	if state == nil || name == "" {
		return
	}
	state.clearBorrowedPtrAliasTree(name)
	if typeName == "ptr" {
		if owner, borrowed := borrowedPtrOwnerFromExpr(expr, state, borrowedParams); borrowed {
			state.bindBorrowedPtrAlias(name, owner)
		}
		return
	}
	if owner, borrowed := borrowedPtrOwnerFromExpr(expr, state, borrowedParams); borrowed {
		if typeMayContainPtr(typeName, types) {
			state.bindBorrowedPtrAlias(name, owner)
		}
	}
	if !typeMayContainPtr(typeName, types) {
		return
	}
	if info, ok := types[typeName]; ok && info.Kind == TypeOptional {
		bindBorrowedPtrAliasFromExpr(resourceFieldPath(name, "$elem"), info.ElemType, expr, types, module, imports, state, borrowedParams)
		return
	}
	if sourcePath, ok := resourcePathForExpr(expr); ok {
		prefix := sourcePath + "."
		for path, owner := range state.borrowedPtrAliases {
			if owner == "" || !strings.HasPrefix(path, prefix) {
				continue
			}
			suffix := strings.TrimPrefix(path, prefix)
			state.bindBorrowedPtrAlias(joinResourcePath(name, suffix), owner)
		}
		return
	}
	info, ok := types[typeName]
	if !ok {
		return
	}
	if info.Kind == TypeEnum {
		call, ok := expr.(*frontend.CallExpr)
		if !ok {
			return
		}
		enumType, caseInfo, found, err := resolveEnumCaseConstructorCall(call, types, module, imports)
		if err != nil || !found || enumType != typeName {
			return
		}
		for i, arg := range call.Args {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			bindBorrowedPtrAliasFromExpr(resourceEnumPayloadPath(name, caseInfo.Ordinal, i), caseInfo.PayloadTypes[i], arg, types, module, imports, state, borrowedParams)
		}
		return
	}
	if info.Kind != TypeStruct {
		return
	}
	fieldTypes := make(map[string]string, len(info.Fields))
	for _, field := range info.Fields {
		fieldTypes[field.Name] = field.TypeName
	}
	lit, ok := expr.(*frontend.StructLitExpr)
	if lit != nil {
		for _, field := range lit.Fields {
			fieldType, ok := fieldTypes[field.Name]
			if !ok {
				continue
			}
			bindBorrowedPtrAliasFromExpr(joinResourcePath(name, field.Name), fieldType, field.Value, types, module, imports, state, borrowedParams)
		}
		return
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || len(call.Args) == 0 || len(call.ArgLabels) != len(call.Args) {
		return
	}
	for i, arg := range call.Args {
		label := call.ArgLabels[i]
		if label == "" {
			return
		}
		fieldType, ok := fieldTypes[label]
		if !ok {
			continue
		}
		bindBorrowedPtrAliasFromExpr(joinResourcePath(name, label), fieldType, arg, types, module, imports, state, borrowedParams)
	}
}

func compoundIndexTargetHasSideEffects(expr frontend.Expr) bool {
	switch e := expr.(type) {
	case *frontend.IndexExpr:
		return exprMayHaveRuntimeSideEffects(e.Base) || exprMayHaveRuntimeSideEffects(e.Index)
	case *frontend.FieldAccessExpr:
		return compoundIndexTargetHasSideEffects(e.Base)
	default:
		return false
	}
}

func exprMayHaveRuntimeSideEffects(expr frontend.Expr) bool {
	switch e := expr.(type) {
	case nil:
		return false
	case *frontend.CallExpr, *frontend.TryExpr, *frontend.AwaitExpr, *frontend.CatchExpr:
		return true
	case *frontend.FieldAccessExpr:
		return exprMayHaveRuntimeSideEffects(e.Base)
	case *frontend.IndexExpr:
		return exprMayHaveRuntimeSideEffects(e.Base) || exprMayHaveRuntimeSideEffects(e.Index)
	case *frontend.UnaryExpr:
		return exprMayHaveRuntimeSideEffects(e.X)
	case *frontend.BinaryExpr:
		return exprMayHaveRuntimeSideEffects(e.Left) || exprMayHaveRuntimeSideEffects(e.Right)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if exprMayHaveRuntimeSideEffects(field.Value) {
				return true
			}
		}
		return false
	case *frontend.MatchExpr:
		if exprMayHaveRuntimeSideEffects(e.Value) {
			return true
		}
		for _, c := range e.Cases {
			if exprMayHaveRuntimeSideEffects(c.Pattern) || exprMayHaveRuntimeSideEffects(c.Guard) || exprMayHaveRuntimeSideEffects(c.Value) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func checkBorrowedReturnContract(
	expr frontend.Expr,
	returnType string,
	callerSig FuncSig,
	callerSigOK bool,
	borrowedParams map[string]struct{},
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
	pos frontend.Position,
) error {
	if callerSigOK && callerSig.ReturnOwnership == "borrow" {
		borrowedName, borrowed, err := borrowedOwnerFromExpr(expr, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return err
		}
		if !borrowed {
			return nil
		}
		if borrowedName == "<borrow>" {
			kind, _ := borrowedReturnTypeLabels(returnType, types)
			return lifetimeDiagnosticf(pos, "borrowed %s return requires caller-visible borrow source", kind)
		}
		if _, ok := borrowedParams[borrowedName]; ok {
			if err := recordBorrowedReturnOwner(analysis, borrowedName, pos); err != nil {
				return err
			}
			return nil
		}
		kind, _ := borrowedReturnTypeLabels(returnType, types)
		return lifetimeDiagnosticf(pos, "borrowed %s return derives from local owner '%s'", kind, borrowedName)
	}
	if err := checkBorrowedAggregateEscape(expr, returnType, "escape through owned return", locals, globals, funcs, types, module, imports, state, effects, analysis, pos); err != nil {
		return err
	}
	return checkBorrowedEscape(expr, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
		kind, display, directView := borrowedReturnDirectViewLabels(returnType, types)
		if directView {
			return lifetimeDiagnosticf(pos, "borrowed %s return requires '-> borrow %s' or '.copy()'", kind, display)
		}
		return lifetimeDiagnosticf(pos, "borrowed local '%s' cannot escape via return", borrowedName)
	})
}

func recordBorrowedReturnOwner(analysis *functionAnalysisState, owner string, pos frontend.Position) error {
	if analysis == nil || owner == "" {
		return nil
	}
	if analysis.borrowedReturnOwner == "" {
		analysis.borrowedReturnOwner = owner
		return nil
	}
	if analysis.borrowedReturnOwner != owner {
		return lifetimeDiagnosticf(pos, "borrowed return has multiple possible owner sources ('%s', '%s'); named lifetimes are not supported in v1", analysis.borrowedReturnOwner, owner)
	}
	return nil
}

func checkBorrowedAggregateEscape(
	expr frontend.Expr,
	typeName string,
	context string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
	pos frontend.Position,
) error {
	if explicitCopyResultExpr(expr) {
		return nil
	}
	info, ok := types[typeName]
	if !ok {
		return nil
	}
	switch info.Kind {
	case TypeStruct:
		for _, field := range structFieldExprs(expr, info) {
			if err := checkBorrowedAggregateFieldEscape(info.Name, field.name, field.typeName, field.value, context, locals, globals, funcs, types, module, imports, state, effects, analysis, pos); err != nil {
				return err
			}
		}
	case TypeEnum:
		if call, ok := expr.(*frontend.CallExpr); ok {
			_, caseInfo, found, err := resolveEnumCaseConstructorCall(call, types, module, imports)
			if err != nil {
				return err
			}
			if !found {
				return nil
			}
			for i, arg := range call.Args {
				if i >= len(caseInfo.PayloadTypes) {
					break
				}
				label := fmt.Sprintf("%s.%s[%d]", displayTypeName(typeName, module), caseInfo.Name, i+1)
				if err := checkBorrowedAggregateFieldEscape(displayTypeName(typeName, module), label, caseInfo.PayloadTypes[i], arg, context, locals, globals, funcs, types, module, imports, state, effects, analysis, pos); err != nil {
					return err
				}
			}
		}
	case TypeOptional:
		if call, ok := expr.(*frontend.CallExpr); ok {
			for _, arg := range call.Args {
				if err := checkBorrowedAggregateFieldEscape(displayTypeName(typeName, module), "$elem", info.ElemType, arg, context, locals, globals, funcs, types, module, imports, state, effects, analysis, pos); err != nil {
					return err
				}
			}
		} else if borrowedEscapeShouldInspect(info.ElemType, types) {
			if err := checkBorrowedAggregateFieldEscape(displayTypeName(typeName, module), "$elem", info.ElemType, expr, context, locals, globals, funcs, types, module, imports, state, effects, analysis, pos); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkBorrowedAggregateFieldEscape(
	aggregateName string,
	fieldName string,
	fieldType string,
	value frontend.Expr,
	context string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
	pos frontend.Position,
) error {
	if !borrowedEscapeShouldInspect(fieldType, types) {
		return nil
	}
	kind, _, directView := borrowedReturnDirectViewLabels(fieldType, types)
	if directView {
		if _, borrowed, err := borrowedOwnerFromExpr(value, locals, globals, funcs, types, module, imports, state, effects, analysis); err != nil {
			return err
		} else if borrowed {
			return lifetimeDiagnosticf(pos, "aggregate '%s' contains borrowed %s field '%s' that cannot %s", displayTypeName(aggregateName, module), kind, fieldName, context)
		}
	}
	return checkBorrowedAggregateEscape(value, fieldType, context, locals, globals, funcs, types, module, imports, state, effects, analysis, pos)
}

type structFieldExpr struct {
	name     string
	typeName string
	value    frontend.Expr
}

func structFieldExprs(expr frontend.Expr, info *TypeInfo) []structFieldExpr {
	if info == nil || info.Kind != TypeStruct || expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *frontend.StructLitExpr:
		out := make([]structFieldExpr, 0, len(e.Fields))
		for _, field := range e.Fields {
			fieldInfo, ok := info.FieldMap[field.Name]
			if !ok {
				continue
			}
			out = append(out, structFieldExpr{name: field.Name, typeName: fieldInfo.TypeName, value: field.Value})
		}
		return out
	case *frontend.CallExpr:
		out := make([]structFieldExpr, 0, len(e.Args))
		if len(e.ArgLabels) == len(e.Args) {
			for i, arg := range e.Args {
				label := e.ArgLabels[i]
				if label == "" {
					continue
				}
				fieldInfo, ok := info.FieldMap[label]
				if !ok {
					continue
				}
				out = append(out, structFieldExpr{name: label, typeName: fieldInfo.TypeName, value: arg})
			}
			return out
		}
		for i, arg := range e.Args {
			if i >= len(info.Fields) {
				break
			}
			field := info.Fields[i]
			out = append(out, structFieldExpr{name: field.Name, typeName: field.TypeName, value: arg})
		}
		return out
	}
	return nil
}

func borrowedReturnTypeLabels(typeName string, types map[string]*TypeInfo) (kind string, display string) {
	kind, display, _ = borrowedReturnDirectViewLabels(typeName, types)
	return kind, display
}

func borrowedReturnDirectViewLabels(typeName string, types map[string]*TypeInfo) (kind string, display string, directView bool) {
	if info, ok := types[typeName]; ok {
		switch info.Kind {
		case TypeStr:
			return "String", "String", true
		case TypeSlice:
			return "slice", typeName, true
		}
	}
	if typeName == "str" || typeName == "String" {
		return "String", "String", true
	}
	return "value", typeName, false
}
