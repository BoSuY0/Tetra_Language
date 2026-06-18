package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
)

func validateFunctionTypeReturnCallBinding(
	name string,
	declared frontend.TypeRef,
	init *frontend.CallExpr,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
) (string, error) {
	resolvedCall, err := resolveCheckedCallName(init.Name, funcs, module, imports, init.At)
	if err != nil {
		return "", unsupportedFunctionTypedLocalInitializerReturnCallSourceError(init.At, name, init.Name)
	}
	init.Name = resolvedCall
	callSig, ok := funcs[resolvedCall]
	if !ok {
		return "", unsupportedFunctionTypedLocalInitializerReturnCallSourceError(init.At, name, init.Name)
	}
	if !callSig.ReturnFunctionType {
		return "", unsupportedFunctionTypedLocalInitializerReturnCallSourceError(init.At, name, init.Name)
	}
	if callSig.ReturnFunctionSymbol != "" {
		targetSig, ok := funcs[callSig.ReturnFunctionSymbol]
		if !ok {
			return "", fmt.Errorf("%s: unknown returned function symbol '%s'", frontend.FormatPos(init.At), callSig.ReturnFunctionSymbol)
		}
		if targetSig.Generic {
			return "", unsupportedGenericFunctionTypedLocalInitializerError(init.At, callSig.ReturnFunctionSymbol, name)
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
	if err := validateFunctionTypeSymbolSignature(name, declared, returnedSig, module, imports, init.At); err != nil {
		return "", err
	}
	return callSig.ReturnFunctionSymbol, nil
}

func resolveImportedFunctionFieldAccess(
	value *frontend.FieldAccessExpr,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
) (string, bool) {
	base, ok := value.Base.(*frontend.IdentExpr)
	if !ok {
		return "", false
	}
	importedModule, ok := imports[base.Name]
	if !ok || importedModule == "" {
		return "", false
	}
	name := importedModule + "." + value.Field
	sig, ok := funcs[name]
	if !ok {
		return "", false
	}
	if err := ensureFuncVisible(name, sig, module, value.At); err != nil {
		return "", false
	}
	return name, true
}

func resolveFunctionTypedGlobalFieldAccess(
	value *frontend.FieldAccessExpr,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
) (GlobalInfo, FuncSig, bool, error) {
	name := callbackArgumentName(value)
	global, ok := globals[name]
	if !ok {
		return GlobalInfo{}, FuncSig{}, false, nil
	}
	if !global.FunctionTypeValue || global.FunctionValue == "" {
		if global.Mutable {
			return GlobalInfo{}, FuncSig{}, true, unsupportedImportedMutableFunctionTypedGlobalUseError(value.At, name)
		}
		return GlobalInfo{}, FuncSig{}, true, unsupportedFunctionTypedGlobalTargetError(value.At, name)
	}
	sig, ok := funcs[global.FunctionValue]
	if !ok {
		return GlobalInfo{}, FuncSig{}, true, fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(value.At), global.FunctionValue)
	}
	return global, sig, true, nil
}

func resolveImportedFunctionGlobalInitializer(
	value *frontend.FieldAccessExpr,
	world *module.World,
	currentModule string,
	imports map[string]string,
	types map[string]*TypeInfo,
) (string, FuncSig, bool, error) {
	base, ok := value.Base.(*frontend.IdentExpr)
	if !ok {
		return "", FuncSig{}, false, nil
	}
	importedModule, ok := imports[base.Name]
	if !ok || importedModule == "" {
		return "", FuncSig{}, false, nil
	}
	file := world.ByModule[importedModule]
	if file == nil {
		return "", FuncSig{}, false, nil
	}
	var target *frontend.FuncDecl
	for _, fn := range file.Funcs {
		if fn != nil && fn.Name == value.Field {
			target = fn
			break
		}
	}
	if target == nil {
		return "", FuncSig{}, false, nil
	}
	name := importedModule + "." + target.Name
	sig, err := funcSigFromDeclForGlobalInitializer(file, target, importedModule, types)
	if err != nil {
		return "", FuncSig{}, false, err
	}
	if err := ensureFuncVisible(name, sig, currentModule, value.At); err != nil {
		return "", FuncSig{}, false, err
	}
	return name, sig, true, nil
}

func funcSigFromDeclForGlobalInitializer(file *frontend.FileAST, fn *frontend.FuncDecl, currentModule string, types map[string]*TypeInfo) (FuncSig, error) {
	if err := validateFunctionParamNames(fn); err != nil {
		return FuncSig{}, err
	}
	imports, err := collectImportAliases(file)
	if err != nil {
		return FuncSig{}, err
	}
	effects, err := normalizeEffects(fn.Uses, fn.Pos)
	if err != nil {
		return FuncSig{}, err
	}
	retName, err := resolveTypeName(&fn.ReturnType, currentModule, imports)
	if err != nil {
		return FuncSig{}, err
	}
	throwsType := ""
	if fn.HasThrows {
		throwsType, err = resolveTypeName(&fn.Throws, currentModule, imports)
		if err != nil {
			return FuncSig{}, err
		}
	}
	paramTypes := make([]string, 0, len(fn.Params))
	paramOwnership := make([]string, 0, len(fn.Params))
	for i := range fn.Params {
		param := &fn.Params[i]
		resolved, err := resolveTypeName(&param.Type, currentModule, imports)
		if err != nil {
			return FuncSig{}, err
		}
		if _, err := ensureTypeInfo(resolved, types); err != nil {
			return FuncSig{}, fmt.Errorf("%s: %v", frontend.FormatPos(param.At), err)
		}
		paramTypes = append(paramTypes, resolved)
		paramOwnership = append(paramOwnership, param.Ownership)
	}
	return FuncSig{
		Generic:         len(fn.TypeParams) > 0,
		Public:          declarationIsPublic(file, fn.Public),
		ParamTypes:      paramTypes,
		ParamOwnership:  paramOwnership,
		ReturnType:      retName,
		ReturnOwnership: fn.ReturnOwnership,
		ThrowsType:      throwsType,
		Effects:         effects,
	}, nil
}

func enumPayloadFunctionKey(ordinal int32, index int) string {
	return fmt.Sprintf("%d:%d", ordinal, index)
}

func enumPayloadFunctionInfo(caseInfo EnumCaseInfo, index int, functionValue string) FunctionFieldInfo {
	field := FunctionFieldInfo{FunctionValue: functionValue}
	if index >= 0 && index < len(caseInfo.PayloadFunctionParams) {
		field.FunctionParamTypes = append([]string(nil), caseInfo.PayloadFunctionParams[index]...)
	}
	if index >= 0 && index < len(caseInfo.PayloadFunctionOwns) {
		field.FunctionParamOwnership = append([]string(nil), caseInfo.PayloadFunctionOwns[index]...)
	}
	if index >= 0 && index < len(caseInfo.PayloadFunctionReturns) {
		field.FunctionReturnType = caseInfo.PayloadFunctionReturns[index]
	}
	if index >= 0 && index < len(caseInfo.PayloadFunctionReturnOwns) {
		field.FunctionReturnOwnership = caseInfo.PayloadFunctionReturnOwns[index]
	}
	if index >= 0 && index < len(caseInfo.PayloadFunctionThrows) {
		field.FunctionThrowsType = caseInfo.PayloadFunctionThrows[index]
	}
	if index >= 0 && index < len(caseInfo.PayloadFunctionEffects) {
		field.FunctionEffects = append([]string(nil), caseInfo.PayloadFunctionEffects[index]...)
	}
	return field
}

func validateFunctionTypeEnumPayloadBinding(
	enumType string,
	caseInfo EnumCaseInfo,
	index int,
	init frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (FunctionFieldInfo, error) {
	if index < 0 || index >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[index] {
		return FunctionFieldInfo{}, nil
	}
	label := fmt.Sprintf("%s.%s[%d]", displayTypeName(enumType, module), caseInfo.Name, index+1)
	resolved := ""
	paramName := ""
	captures := []frontend.ClosureCapture(nil)
	escapeCaptures := []frontend.ClosureCapture(nil)
	directSnapshotAlias := false
	escapeKind := CallableEscapeKind("")
	handleValue := false
	switch value := init.(type) {
	case *frontend.IdentExpr:
		source, localIdent := locals[value.Name]
		var err error
		resolved, err = validateFunctionTypeNamedSymbolBinding(label, caseInfo.PayloadFunctionRefs[index], value, locals, globals, funcs, types, module, imports, true, unsupportedGenericFunctionTypedEnumPayloadInitializerError)
		if err != nil {
			return FunctionFieldInfo{}, err
		}
		captures = append([]frontend.ClosureCapture(nil), source.FunctionCaptures...)
		escapeCaptures = append([]frontend.ClosureCapture(nil), source.FunctionEscapeCaptures...)
		directSnapshotAlias = source.FunctionDirectSnapshotAlias
		escapeKind = source.FunctionEscapeKind
		handleValue = source.FunctionHandleValue
		if source.FunctionTypeValue && source.FunctionValue == "" {
			paramName = source.FunctionParamName
			if paramName == "" {
				paramName = value.Name
			}
		}
		if resolved != "" && !localIdent {
			value.Name = resolved
		}
	case *frontend.FieldAccessExpr:
		fieldInfo, ok, err := resolveFunctionFieldArgument(value, locals)
		target := ""
		fieldTargetInfo := FunctionFieldInfo{}
		fieldTargetInfoOK := false
		if err != nil {
			return FunctionFieldInfo{}, err
		}
		if ok && fieldInfo.FunctionValue != "" {
			target = fieldInfo.FunctionValue
			fieldTargetInfo = fieldInfo
			fieldTargetInfoOK = true
			captures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionCaptures...)
			escapeCaptures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionEscapeCaptures...)
			directSnapshotAlias = fieldInfo.FunctionDirectSnapshotAlias
			escapeKind = fieldInfo.FunctionEscapeKind
			handleValue = fieldInfo.FunctionHandleValue
		} else if ok && functionFieldInfoHasTargetSet(fieldInfo) {
			paramName = fieldInfo.FunctionParamName
			captures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionCaptures...)
			escapeCaptures = append([]frontend.ClosureCapture(nil), fieldInfo.FunctionEscapeCaptures...)
			directSnapshotAlias = fieldInfo.FunctionDirectSnapshotAlias
			escapeKind = fieldInfo.FunctionEscapeKind
			handleValue = fieldInfo.FunctionHandleValue
			if err := validateFunctionInfoAssignable(label, enumPayloadLocalInfo(caseInfo, index), functionFieldInfoSig(fieldInfo), value.At); err != nil {
				return FunctionFieldInfo{}, err
			}
		} else if globalInfo, _, globalOK, err := resolveFunctionTypedGlobalFieldAccess(value, globals, funcs); err != nil {
			return FunctionFieldInfo{}, err
		} else if globalOK {
			target = globalInfo.FunctionValue
		} else if imported, importedOK := resolveImportedFunctionFieldAccess(value, funcs, module, imports); importedOK {
			target = imported
		} else {
			return FunctionFieldInfo{}, unsupportedFunctionTypedEnumPayloadInitializerSourceError(value.At, label)
		}
		if target != "" {
			targetSig, ok := funcs[target]
			if !ok {
				return FunctionFieldInfo{}, fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(value.At), target)
			}
			if targetSig.Generic {
				return FunctionFieldInfo{}, unsupportedGenericFunctionTypedEnumPayloadInitializerError(value.At, callbackArgumentName(value), label)
			}
			if fieldTargetInfoOK {
				if err := validateFunctionInfoAssignable(label, enumPayloadLocalInfo(caseInfo, index), functionFieldInfoSig(fieldTargetInfo), value.At); err != nil {
					return FunctionFieldInfo{}, err
				}
			} else {
				if err := validateFunctionTypeSymbolSignature(label, caseInfo.PayloadFunctionRefs[index], targetSig, module, imports, value.At); err != nil {
					return FunctionFieldInfo{}, err
				}
			}
			resolved = target
		}
	case *frontend.CallExpr:
		var err error
		resolved, err = validateFunctionTypeReturnCallBinding(label, caseInfo.PayloadFunctionRefs[index], value, funcs, module, imports)
		if err != nil {
			return FunctionFieldInfo{}, err
		}
		metadataValue, metadataCaptures, metadataEscapeCaptures, metadataParamName, err := functionAssignmentMetadataWithReturnParamRefs(value, locals, globals, funcs, types, module, imports)
		if err != nil {
			return FunctionFieldInfo{}, err
		}
		if metadataValue != "" {
			resolved = metadataValue
		}
		paramName = metadataParamName
		captures = append([]frontend.ClosureCapture(nil), metadataCaptures...)
		escapeCaptures = append([]frontend.ClosureCapture(nil), metadataEscapeCaptures...)
		if callSig, ok := funcs[value.Name]; ok && callSig.ReturnFunctionHandleValue {
			escapeKind = callSig.ReturnFunctionEscapeKind
			handleValue = callSig.ReturnFunctionHandleValue
		}
	case *frontend.ClosureExpr:
		target := enumPayloadLocalInfo(caseInfo, index)
		if err := validateFunctionTypedClosureAssignment(label, target, value, locals, funcs, types, module, imports, value.At); err != nil {
			return FunctionFieldInfo{}, err
		}
		if len(value.Captures) > 0 {
			captureSlots, err := functionCaptureSlotCount(value.Captures, types)
			if err != nil {
				return FunctionFieldInfo{}, err
			}
			if captureSlots > FnPtrEnvSlotCount {
				var err error
				escapeKind, handleValue, err = classifyCallableEscape(callableBoundaryEnumPayload, value.Captures, types)
				if err != nil {
					return FunctionFieldInfo{}, err
				}
			}
		}
		resolved = closureFunctionValueName(value, funcs, module)
		captures = append([]frontend.ClosureCapture(nil), value.Captures...)
		directSnapshotAlias = len(value.Captures) > 0
	default:
		return FunctionFieldInfo{}, unsupportedFunctionTypedEnumPayloadInitializerSourceError(init.Pos(), label)
	}
	if index >= len(caseInfo.PayloadFunctionRefs) {
		return FunctionFieldInfo{}, fmt.Errorf("%s: function-typed enum payload '%s' is missing function type metadata", frontend.FormatPos(init.Pos()), label)
	}
	info := enumPayloadFunctionInfo(caseInfo, index, resolved)
	info.FunctionParamName = paramName
	info.FunctionCaptures = captures
	info.FunctionEscapeCaptures = escapeCaptures
	info.FunctionReturnSnapshotAlias = isFunctionReturnSnapshotAlias(init, funcs, captures, escapeCaptures, paramName)
	info.FunctionDirectSnapshotAlias = directSnapshotAlias
	info.FunctionEscapeKind = escapeKind
	info.FunctionHandleValue = handleValue
	touchesMutableGlobals, err := functionAssignmentValueTouchesMutableGlobals(init, locals, globals, funcs, types, module, imports)
	if err != nil {
		return FunctionFieldInfo{}, err
	}
	info.FunctionTouchesMutableGlobals = touchesMutableGlobals
	return info, nil
}

func unsupportedFunctionTypedEnumPayloadInitializerSourceError(pos frontend.Position, label string) error {
	return fmt.Errorf(
		"%s: function-typed enum payload '%s' initializer must be a supported fnptr source: closure literal, function-typed local/global/struct field, direct named function/closure symbol, or function-typed return call",
		frontend.FormatPos(pos),
		label,
	)
}

func enumPayloadLocalInfo(caseInfo EnumCaseInfo, index int) LocalInfo {
	info := enumPayloadFunctionInfo(caseInfo, index, "")
	return LocalInfo{
		SlotCount:               FnPtrSlotCount,
		TypeName:                "fnptr",
		FunctionTypeValue:       true,
		FunctionParamTypes:      append([]string(nil), info.FunctionParamTypes...),
		FunctionParamOwnership:  append([]string(nil), info.FunctionParamOwnership...),
		FunctionReturnType:      info.FunctionReturnType,
		FunctionReturnOwnership: info.FunctionReturnOwnership,
		FunctionThrowsType:      info.FunctionThrowsType,
		FunctionEffects:         append([]string(nil), info.FunctionEffects...),
	}
}

func enumPayloadFunctionsFromConstructor(
	info *TypeInfo,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]FunctionFieldInfo, error) {
	if info == nil || info.Kind != TypeEnum {
		return nil, nil
	}
	call, ok := value.(*frontend.CallExpr)
	if !ok {
		return nil, nil
	}
	enumType, caseInfo, ok, err := resolveEnumCaseConstructorCall(call, types, module, imports)
	if err != nil {
		return nil, err
	}
	if !ok || enumType != info.Name {
		return nil, nil
	}
	out := map[string]FunctionFieldInfo{}
	for i, arg := range call.Args {
		if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
			continue
		}
		payloadInfo, err := validateFunctionTypeEnumPayloadBinding(enumType, caseInfo, i, arg, locals, globals, funcs, types, module, imports)
		if err != nil {
			return nil, err
		}
		out[enumPayloadFunctionKey(caseInfo.Ordinal, i)] = payloadInfo
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func enumPayloadFunctionsFromReturnedEnumExpr(
	returnType string,
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]FunctionFieldInfo, error) {
	info, ok := types[returnType]
	if !ok || info.Kind != TypeEnum {
		return nil, nil
	}
	payloads, err := enumPayloadFunctionsFromConstructor(info, value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return nil, err
	}
	if len(payloads) == 0 {
		payloads = enumPayloadFunctionValuesForExpr(value, locals)
	}
	if len(payloads) == 0 {
		payloads, err = enumPayloadFunctionsFromReturnCall(value, locals, globals, funcs, types, module, imports, returnType)
		if err != nil {
			return nil, err
		}
	}
	return payloads, nil
}

func enumPayloadFunctionsFromReturnCall(
	value frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	expectedType string,
) (map[string]FunctionFieldInfo, error) {
	call, ok := value.(*frontend.CallExpr)
	if !ok {
		return nil, nil
	}
	resolved, err := resolveCheckedCallName(call.Name, funcs, module, imports, call.At)
	if err != nil {
		return nil, nil
	}
	sig, ok := funcs[resolved]
	if !ok || sig.ReturnType != expectedType {
		return nil, nil
	}
	if len(sig.ReturnEnumPayloadFunctions) == 0 {
		argumentCaptures, err := capturedFunctionTypedCallArgumentMetadata(sig, call, locals, globals, funcs, types, module, imports)
		if err != nil {
			return nil, err
		}
		enumArgumentCaptures, err := capturedFunctionTypedEnumCallArgumentMetadata(sig, call, locals, globals, funcs, types, module, imports)
		if err != nil {
			return nil, err
		}
		argumentCaptures = append(argumentCaptures, enumArgumentCaptures...)
		enumArgumentPayloads, err := functionTypedEnumCallArgumentPayloadMetadata(sig, call, locals, globals, funcs, types, module, imports)
		if err != nil {
			return nil, err
		}
		argumentParamName := ""
		for i, functionParam := range sig.ParamFunctionTypes {
			if !functionParam || i >= len(call.Args) {
				continue
			}
			_, _, _, paramName, err := functionAssignmentMetadataWithReturnParamRefs(call.Args[i], locals, globals, funcs, types, module, imports)
			if err != nil {
				return nil, err
			}
			if paramName != "" {
				argumentParamName = paramName
				break
			}
		}
		if len(argumentCaptures) == 0 && argumentParamName == "" && len(enumArgumentPayloads) == 0 {
			return nil, nil
		}
		payloads := declaredEnumPayloadFunctionsForType(expectedType, types)
		for key, payload := range payloads {
			payload.FunctionParamName = argumentParamName
			payload.FunctionEscapeCaptures = append([]frontend.ClosureCapture(nil), argumentCaptures...)
			payloads[key] = payload
		}
		if payloads == nil && len(enumArgumentPayloads) > 0 {
			payloads = map[string]FunctionFieldInfo{}
		}
		for key, payload := range enumArgumentPayloads {
			mergeFunctionFieldInfoIntoMap(payloads, key, payload)
		}
		return payloads, nil
	}
	payloads := cloneFunctionFieldMap(sig.ReturnEnumPayloadFunctions)
	for payloadKey, payload := range payloads {
		if payload.FunctionParamName == "" {
			payloads[payloadKey] = functionFieldInfoAsReturnSnapshot(payload)
			continue
		}
		resolvedPayload, found, err := functionTypedReturnParamRefMetadata(sig, payload.FunctionParamName, call, locals, globals, funcs, types, module, imports)
		if err != nil || !found {
			return nil, err
		}
		if resolvedPayload.FunctionValue != "" {
			payload.FunctionValue = resolvedPayload.FunctionValue
		}
		if resolvedPayload.FunctionParamName != "" {
			payload.FunctionParamName = resolvedPayload.FunctionParamName
		}
		payload.FunctionEscapeCaptures = append([]frontend.ClosureCapture(nil), resolvedPayload.FunctionCaptures...)
		payload.FunctionEscapeCaptures = append(payload.FunctionEscapeCaptures, resolvedPayload.FunctionEscapeCaptures...)
		payloads[payloadKey] = functionFieldInfoAsReturnSnapshot(payload)
	}
	return payloads, nil
}

func functionTypedEnumCallArgumentPayloadMetadata(
	sig FuncSig,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (map[string]FunctionFieldInfo, error) {
	out := map[string]FunctionFieldInfo{}
	for i, typeName := range sig.ParamTypes {
		if i >= len(call.Args) {
			continue
		}
		info, ok := types[typeName]
		if !ok || info.Kind != TypeEnum {
			continue
		}
		payloads := enumPayloadFunctionValuesForExpr(call.Args[i], locals)
		if len(payloads) == 0 {
			var err error
			payloads, err = enumPayloadFunctionsFromConstructor(info, call.Args[i], locals, globals, funcs, types, module, imports)
			if err != nil {
				return nil, err
			}
		}
		if len(payloads) == 0 {
			var err error
			payloads, err = enumPayloadFunctionsFromReturnCall(call.Args[i], locals, globals, funcs, types, module, imports, typeName)
			if err != nil {
				return nil, err
			}
		}
		for payloadKey, payload := range payloads {
			if !functionFieldInfoHasTargetSet(payload) {
				continue
			}
			mergeFunctionFieldInfoIntoMap(out, payloadKey, payload)
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func capturedFunctionTypedEnumCallArgumentMetadata(
	sig FuncSig,
	call *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) ([]frontend.ClosureCapture, error) {
	var out []frontend.ClosureCapture
	for i, typeName := range sig.ParamTypes {
		if i >= len(call.Args) {
			continue
		}
		info, ok := types[typeName]
		if !ok || info.Kind != TypeEnum {
			continue
		}
		payloads := enumPayloadFunctionValuesForExpr(call.Args[i], locals)
		if len(payloads) == 0 {
			var err error
			payloads, err = enumPayloadFunctionsFromConstructor(info, call.Args[i], locals, globals, funcs, types, module, imports)
			if err != nil {
				return nil, err
			}
		}
		if len(payloads) == 0 {
			var err error
			payloads, err = enumPayloadFunctionsFromReturnCall(call.Args[i], locals, globals, funcs, types, module, imports, typeName)
			if err != nil {
				return nil, err
			}
		}
		for _, payload := range payloads {
			out = append(out, payload.FunctionCaptures...)
			out = append(out, payload.FunctionEscapeCaptures...)
		}
	}
	return out, nil
}

func enumPayloadFunctionsFromAlias(value frontend.Expr, locals map[string]LocalInfo) map[string]FunctionFieldInfo {
	if id, ok := value.(*frontend.IdentExpr); ok {
		local, ok := locals[id.Name]
		if !ok || len(local.EnumPayloadFunctions) == 0 {
			return nil
		}
		return cloneFunctionFieldMap(local.EnumPayloadFunctions)
	}
	return enumPayloadFunctionsFromStructFieldExpr(value, locals)
}

func functionLocalInfoForEnumPayload(caseInfo EnumCaseInfo, index int, value FunctionFieldInfo) LocalInfo {
	info := LocalInfo{
		SlotCount:                     1,
		TypeName:                      "ptr",
		Mutable:                       false,
		FunctionValue:                 value.FunctionValue,
		FunctionParamName:             value.FunctionParamName,
		FunctionCaptures:              append([]frontend.ClosureCapture(nil), value.FunctionCaptures...),
		FunctionEscapeCaptures:        append([]frontend.ClosureCapture(nil), value.FunctionEscapeCaptures...),
		FunctionTouchesMutableGlobals: value.FunctionTouchesMutableGlobals,
		FunctionReturnSnapshotAlias:   value.FunctionReturnSnapshotAlias,
		FunctionDirectSnapshotAlias:   value.FunctionDirectSnapshotAlias,
		FunctionEscapeKind:            value.FunctionEscapeKind,
		FunctionHandleValue:           value.FunctionHandleValue,
		FunctionEnumPayload:           true,
		FunctionTypeValue:             true,
		FunctionParamTypes:            append([]string(nil), value.FunctionParamTypes...),
		FunctionParamOwnership:        append([]string(nil), value.FunctionParamOwnership...),
		FunctionReturnType:            value.FunctionReturnType,
		FunctionReturnOwnership:       value.FunctionReturnOwnership,
		FunctionThrowsType:            value.FunctionThrowsType,
		FunctionEffects:               append([]string(nil), value.FunctionEffects...),
	}
	if value.FunctionValue == "" {
		info.FunctionValue = ""
		info.FunctionParamTypes = append([]string(nil), caseInfo.PayloadFunctionParams[index]...)
		info.FunctionParamOwnership = append([]string(nil), caseInfo.PayloadFunctionOwns[index]...)
		info.FunctionReturnType = caseInfo.PayloadFunctionReturns[index]
		if index < len(caseInfo.PayloadFunctionReturnOwns) {
			info.FunctionReturnOwnership = caseInfo.PayloadFunctionReturnOwns[index]
		}
		if index < len(caseInfo.PayloadFunctionThrows) {
			info.FunctionThrowsType = caseInfo.PayloadFunctionThrows[index]
		}
		info.FunctionEffects = append([]string(nil), caseInfo.PayloadFunctionEffects[index]...)
	}
	return info
}

func enumPayloadFunctionValuesForExpr(expr frontend.Expr, locals map[string]LocalInfo) map[string]FunctionFieldInfo {
	if id, ok := expr.(*frontend.IdentExpr); ok {
		local, ok := locals[id.Name]
		if !ok || len(local.EnumPayloadFunctions) == 0 {
			return nil
		}
		return local.EnumPayloadFunctions
	}
	return enumPayloadFunctionsFromStructFieldExpr(expr, locals)
}

func enumPayloadFunctionsFromStructFieldExpr(expr frontend.Expr, locals map[string]LocalInfo) map[string]FunctionFieldInfo {
	name := callbackArgumentName(expr)
	if name == "" {
		return nil
	}
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return nil
	}
	local, ok := locals[parts[0]]
	if !ok || len(local.EnumPayloadFields) == 0 {
		return nil
	}
	prefix := strings.Join(parts[1:], ".") + "#"
	out := map[string]FunctionFieldInfo{}
	for fieldName, fieldInfo := range local.EnumPayloadFields {
		if !strings.HasPrefix(fieldName, prefix) {
			continue
		}
		payloadKey := strings.TrimPrefix(fieldName, prefix)
		if payloadKey == "" {
			continue
		}
		out[payloadKey] = cloneFunctionFieldInfo(fieldInfo)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func enumPayloadFunctionValuesForMatchExpr(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	scrutType string,
) (map[string]FunctionFieldInfo, error) {
	if payloads := enumPayloadFunctionValuesForExpr(expr, locals); len(payloads) > 0 {
		return payloads, nil
	}
	return enumPayloadFunctionsFromReturnCall(expr, locals, globals, funcs, types, module, imports, scrutType)
}

func bindEnumPatternFunctionPayloadLocals(pattern frontend.Expr, payloads map[string]FunctionFieldInfo, locals map[string]LocalInfo, types map[string]*TypeInfo, module string, imports map[string]string) error {
	if len(payloads) == 0 {
		return nil
	}
	enumPat, ok := pattern.(*frontend.EnumCasePatternExpr)
	if !ok {
		return nil
	}
	_, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	for i, binding := range enumPat.Bindings {
		if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
			continue
		}
		value, ok := payloads[enumPayloadFunctionKey(caseInfo.Ordinal, i)]
		if !ok {
			continue
		}
		hasMetadata := value.FunctionValue != "" ||
			value.FunctionParamName != "" ||
			len(value.FunctionCaptures) > 0 ||
			len(value.FunctionEscapeCaptures) > 0
		if !hasMetadata {
			continue
		}
		localInfo := functionLocalInfoForEnumPayload(caseInfo, i, value)
		if existing, exists := locals[binding]; exists {
			localInfo.Base = existing.Base
			localInfo.SlotCount = existing.SlotCount
			localInfo.TypeName = existing.TypeName
		}
		locals[binding] = localInfo
	}
	return nil
}
