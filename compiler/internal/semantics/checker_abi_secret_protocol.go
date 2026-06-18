package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
	semanticspolicy "tetra_language/compiler/internal/semantics/policy"
)

func validateExportedConsentTokenABISignature(_ string, fn *frontend.FuncDecl, paramTypes map[string]string, returnType string, types map[string]*TypeInfo) error {
	if fn == nil || fn.ExportName == "" {
		return nil
	}
	for _, param := range fn.Params {
		paramType := paramTypes[param.Name]
		if isForgeableConsentTokenType(paramType) {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose forgeable consent token '%s' in parameter '%s'",
				fn.Name,
				paramType,
				param.Name,
			)
		}
		if exposure, ok := exportedConsentTokenABIExposureForType(paramType, types); ok {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose %s '%s' through parameter '%s' type '%s'",
				fn.Name,
				exposure.Kind,
				exposure.TypeName,
				param.Name,
				paramType,
			)
		}
	}
	if isForgeableConsentTokenType(returnType) {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose forgeable consent token '%s' in return type",
			fn.Name,
			returnType,
		)
	}
	if exposure, ok := exportedConsentTokenABIExposureForType(returnType, types); ok {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose %s '%s' through return type '%s'",
			fn.Name,
			exposure.Kind,
			exposure.TypeName,
			returnType,
		)
	}
	return nil
}

func validateExportedThrowingABISignature(_ string, fn *frontend.FuncDecl, throwsType string) error {
	if fn == nil || fn.ExportName == "" || strings.TrimSpace(throwsType) == "" {
		return nil
	}
	return effectDiagnosticf(
		fn.Throws.At,
		"exported function '%s' cannot throw typed error '%s'; export a non-throwing wrapper with an explicit result type",
		fn.Name,
		throwsType,
	)
}

func exportedConsentTokenABIExposureForType(typeName string, types map[string]*TypeInfo) (exportedOpaqueABIExposure, bool) {
	return exportedConsentTokenABIExposureForTypeVisiting(strings.TrimSpace(typeName), types, map[string]bool{})
}

func exportedConsentTokenABIExposureForTypeVisiting(typeName string, types map[string]*TypeInfo, visiting map[string]bool) (exportedOpaqueABIExposure, bool) {
	typeName = strings.TrimSpace(typeName)
	if isForgeableConsentTokenType(typeName) {
		return exportedOpaqueABIExposure{Kind: "forgeable consent token", TypeName: typeName}, true
	}
	if elem, ok := optionalElemName(typeName); ok {
		return exportedConsentTokenABIExposureForTypeVisiting(elem, types, visiting)
	}
	if _, elem, ok := parseArrayTypeName(typeName); ok {
		return exportedConsentTokenABIExposureForTypeVisiting(elem, types, visiting)
	}
	info, ok := types[typeName]
	if !ok {
		return exportedOpaqueABIExposure{}, false
	}
	switch info.Kind {
	case TypeStruct:
		if visiting[typeName] {
			return exportedOpaqueABIExposure{}, false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			if exposure, ok := exportedConsentTokenABIExposureForTypeVisiting(field.TypeName, types, visiting); ok {
				return exposure, true
			}
		}
	case TypeEnum:
		if visiting[typeName] {
			return exportedOpaqueABIExposure{}, false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, enumCase := range info.EnumCases {
			for _, payload := range enumCase.PayloadTypes {
				if exposure, ok := exportedConsentTokenABIExposureForTypeVisiting(payload, types, visiting); ok {
					return exposure, true
				}
			}
		}
	case TypeArray, TypeOptional:
		return exportedConsentTokenABIExposureForTypeVisiting(info.ElemType, types, visiting)
	}
	return exportedOpaqueABIExposure{}, false
}

func isInternalRuntimeABIExport(module string, fn *frontend.FuncDecl) bool {
	if fn == nil || !strings.HasPrefix(fn.ExportName, "__tetra_") {
		return false
	}
	return module == "__rt" || strings.HasPrefix(module, "__rt.")
}

type exportedOpaqueABIExposure struct {
	Kind     string
	TypeName string
}

func exportedOpaqueABIExposureForType(typeName string, types map[string]*TypeInfo, allowRuntimeHandles bool) (exportedOpaqueABIExposure, bool) {
	return exportedOpaqueABIExposureForTypeVisiting(strings.TrimSpace(typeName), types, allowRuntimeHandles, map[string]bool{})
}

func exportedDefaultStructABIExposureForType(typeName string, types map[string]*TypeInfo) (exportedOpaqueABIExposure, bool) {
	return exportedDefaultStructABIExposureForTypeVisiting(strings.TrimSpace(typeName), types, map[string]bool{})
}

func exportedDefaultStructABIExposureForTypeVisiting(typeName string, types map[string]*TypeInfo, visiting map[string]bool) (exportedOpaqueABIExposure, bool) {
	typeName = strings.TrimSpace(typeName)
	if elem, ok := optionalElemName(typeName); ok {
		return exportedDefaultStructABIExposureForTypeVisiting(elem, types, visiting)
	}
	if _, elem, ok := parseArrayTypeName(typeName); ok {
		return exportedDefaultStructABIExposureForTypeVisiting(elem, types, visiting)
	}
	info, ok := types[typeName]
	if !ok {
		return exportedOpaqueABIExposure{}, false
	}
	switch info.Kind {
	case TypeStruct:
		if info.Repr != frontend.StructReprC {
			return exportedOpaqueABIExposure{Kind: "default-layout struct", TypeName: info.Name}, true
		}
		if visiting[typeName] {
			return exportedOpaqueABIExposure{}, false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			if exposure, ok := exportedDefaultStructABIExposureForTypeVisiting(field.TypeName, types, visiting); ok {
				return exposure, true
			}
		}
	case TypeArray, TypeOptional:
		return exportedDefaultStructABIExposureForTypeVisiting(info.ElemType, types, visiting)
	}
	return exportedOpaqueABIExposure{}, false
}

func exportedOpaqueABIExposureForTypeVisiting(typeName string, types map[string]*TypeInfo, allowRuntimeHandles bool, visiting map[string]bool) (exportedOpaqueABIExposure, bool) {
	if isOpaqueCapabilityTokenType(typeName) {
		return exportedOpaqueABIExposure{Kind: "opaque capability token", TypeName: typeName}, true
	}
	if isOpaqueIslandHandleType(typeName) {
		return exportedOpaqueABIExposure{Kind: "opaque island handle", TypeName: typeName}, true
	}
	if isFunctionTypedABIValueType(typeName) {
		return exportedOpaqueABIExposure{Kind: "function-typed value", TypeName: typeName}, true
	}
	if exposure, ok := exportedRawViewABIExposureForType(typeName, types); ok {
		return exposure, true
	}
	if !allowRuntimeHandles {
		if exposure, ok := exportedBoolABIExposureForType(typeName, types); ok {
			return exposure, true
		}
	}
	if !allowRuntimeHandles && isOpaqueRuntimeHandleType(typeName) {
		return exportedOpaqueABIExposure{Kind: "opaque runtime handle", TypeName: typeName}, true
	}
	if elem, ok := optionalElemName(typeName); ok {
		if exposure, ok := exportedOpaqueABIExposureForTypeVisiting(elem, types, allowRuntimeHandles, visiting); ok {
			return exposure, true
		}
		return exportedOpaqueABIExposure{Kind: "forgeable optional presence tag", TypeName: typeName}, true
	}
	if _, elem, ok := parseArrayTypeName(typeName); ok {
		return exportedOpaqueABIExposureForTypeVisiting(elem, types, allowRuntimeHandles, visiting)
	}
	info, ok := types[typeName]
	if !ok {
		return exportedOpaqueABIExposure{}, false
	}
	switch info.Kind {
	case TypeStruct:
		if visiting[typeName] {
			return exportedOpaqueABIExposure{}, false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			if exposure, ok := exportedOpaqueABIExposureForTypeVisiting(field.TypeName, types, allowRuntimeHandles, visiting); ok {
				return exposure, true
			}
		}
	case TypeEnum:
		if visiting[typeName] {
			return exportedOpaqueABIExposure{}, false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, enumCase := range info.EnumCases {
			for _, payload := range enumCase.PayloadTypes {
				if exposure, ok := exportedOpaqueABIExposureForTypeVisiting(payload, types, allowRuntimeHandles, visiting); ok {
					return exposure, true
				}
			}
		}
		return exportedOpaqueABIExposure{Kind: "forgeable enum discriminant", TypeName: info.Name}, true
	case TypeArray:
		return exportedOpaqueABIExposureForTypeVisiting(info.ElemType, types, allowRuntimeHandles, visiting)
	case TypeOptional:
		if exposure, ok := exportedOpaqueABIExposureForTypeVisiting(info.ElemType, types, allowRuntimeHandles, visiting); ok {
			return exposure, true
		}
		return exportedOpaqueABIExposure{Kind: "forgeable optional presence tag", TypeName: info.Name}, true
	}
	return exportedOpaqueABIExposure{}, false
}

func isOpaqueIslandHandleType(typeName string) bool {
	return strings.TrimSpace(typeName) == "island"
}

func isForgeableConsentTokenType(typeName string) bool {
	return strings.TrimSpace(typeName) == "consent.token"
}

func isFunctionTypedABIValueType(typeName string) bool {
	return strings.TrimSpace(typeName) == "fnptr"
}

func exportedBoolABIExposureForType(typeName string, types map[string]*TypeInfo) (exportedOpaqueABIExposure, bool) {
	typeName = strings.TrimSpace(typeName)
	info, ok := types[typeName]
	if !ok || info.Kind != TypeBool {
		return exportedOpaqueABIExposure{}, false
	}
	return exportedOpaqueABIExposure{Kind: "unnormalized bool", TypeName: info.Name}, true
}

func exportedRawViewABIExposureForType(typeName string, types map[string]*TypeInfo) (exportedOpaqueABIExposure, bool) {
	typeName = strings.TrimSpace(typeName)
	info, ok := types[typeName]
	if !ok {
		return exportedOpaqueABIExposure{}, false
	}
	switch info.Kind {
	case TypeStr:
		return exportedOpaqueABIExposure{Kind: "raw string view", TypeName: info.Name}, true
	case TypeSlice:
		return exportedOpaqueABIExposure{Kind: "raw slice view", TypeName: info.Name}, true
	case TypeArray:
		return exportedOpaqueABIExposure{Kind: "raw fixed-array view", TypeName: info.Name}, true
	default:
		return exportedOpaqueABIExposure{}, false
	}
}

func isOpaqueCapabilityTokenType(typeName string) bool {
	switch strings.TrimSpace(typeName) {
	case "cap.io", "cap.mem":
		return true
	default:
		return false
	}
}

func isOpaqueRuntimeHandleType(typeName string) bool {
	switch strings.TrimSpace(typeName) {
	case "actor", "task.group", "task.i32":
		return true
	default:
		return false
	}
}

func firstForbiddenEffect(have map[string]struct{}, forbidden []string) string {
	return semanticspolicy.FirstForbiddenEffect(have, forbidden)
}

func typeUsesSecret(typeName string, types map[string]*TypeInfo) bool {
	return typeUsesSecretVisited(strings.TrimSpace(typeName), types, map[string]struct{}{})
}

func functionDeclSignatureUsesSecret(fn *frontend.FuncDecl, types map[string]*TypeInfo) bool {
	if fn == nil {
		return false
	}
	if typeRefUsesSecret(fn.ReturnType, types) {
		return true
	}
	if fn.HasThrows && typeRefUsesSecret(fn.Throws, types) {
		return true
	}
	for _, param := range fn.Params {
		if typeRefUsesSecret(param.Type, types) {
			return true
		}
	}
	return false
}

func typeRefUsesSecret(ref frontend.TypeRef, types map[string]*TypeInfo) bool {
	return typeRefUsesSecretVisited(ref, types, map[string]struct{}{})
}

func typeRefUsesSecretVisited(ref frontend.TypeRef, types map[string]*TypeInfo, visiting map[string]struct{}) bool {
	switch ref.Kind {
	case frontend.TypeRefFunction:
		for _, param := range ref.Params {
			if typeRefUsesSecretVisited(param, types, visiting) {
				return true
			}
		}
		if ref.Return != nil && typeRefUsesSecretVisited(*ref.Return, types, visiting) {
			return true
		}
		return ref.Throws != nil && typeRefUsesSecretVisited(*ref.Throws, types, visiting)
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		if ref.Elem != nil {
			return typeRefUsesSecretVisited(*ref.Elem, types, visiting)
		}
	}
	return typeUsesSecretVisited(strings.TrimSpace(ref.Name), types, visiting)
}

func functionSignatureUsesSecretVisited(paramTypes []string, returnType string, throwsType string, types map[string]*TypeInfo, visiting map[string]struct{}) bool {
	for _, paramType := range paramTypes {
		if typeUsesSecretVisited(paramType, types, visiting) {
			return true
		}
	}
	return typeUsesSecretVisited(returnType, types, visiting) || typeUsesSecretVisited(throwsType, types, visiting)
}

func functionTypedFieldUsesSecret(field FieldInfo, types map[string]*TypeInfo, visiting map[string]struct{}) bool {
	return field.FunctionTypeValue &&
		functionSignatureUsesSecretVisited(field.FunctionParamTypes, field.FunctionReturnType, field.FunctionThrowsType, types, visiting)
}

func enumPayloadFunctionUsesSecret(enumCase EnumCaseInfo, index int, types map[string]*TypeInfo, visiting map[string]struct{}) bool {
	if index < 0 || index >= len(enumCase.PayloadFunctionTypes) || !enumCase.PayloadFunctionTypes[index] {
		return false
	}
	return functionSignatureUsesSecretVisited(
		functionPayloadParamsAt(enumCase, index),
		functionPayloadReturnAt(enumCase, index),
		functionPayloadThrowsAt(enumCase, index),
		types,
		visiting,
	)
}

func functionPayloadParamsAt(enumCase EnumCaseInfo, index int) []string {
	if index >= 0 && index < len(enumCase.PayloadFunctionParams) {
		return enumCase.PayloadFunctionParams[index]
	}
	return nil
}

func functionPayloadReturnAt(enumCase EnumCaseInfo, index int) string {
	if index >= 0 && index < len(enumCase.PayloadFunctionReturns) {
		return enumCase.PayloadFunctionReturns[index]
	}
	return ""
}

func functionPayloadThrowsAt(enumCase EnumCaseInfo, index int) string {
	if index >= 0 && index < len(enumCase.PayloadFunctionThrows) {
		return enumCase.PayloadFunctionThrows[index]
	}
	return ""
}

func exprSecretTainted(
	expr frontend.Expr,
	exprType string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	analysis *functionAnalysisState,
) (bool, error) {
	if expr == nil {
		return false, nil
	}
	if typeUsesSecret(exprType, types) {
		return true, nil
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if analysis.localSecretTainted(e.Name) {
			return true, nil
		}
		if g, ok := globals[e.Name]; ok {
			return typeUsesSecret(g.TypeName, types), nil
		}
		return false, nil
	case *frontend.ClosureExpr:
		for _, capture := range e.Captures {
			if analysis.localSecretTainted(capture.Name) {
				return true, nil
			}
			if local, ok := locals[capture.Name]; ok && typeUsesSecret(local.TypeName, types) {
				return true, nil
			}
		}
		if len(e.Captures) == 0 && e.Decl != nil {
			for name := range collectClosureCaptures(e.Decl, locals) {
				if analysis.localSecretTainted(name) {
					return true, nil
				}
				if local, ok := locals[name]; ok && typeUsesSecret(local.TypeName, types) {
					return true, nil
				}
			}
		}
		return false, nil
	case *frontend.FieldAccessExpr:
		baseType := ""
		if targetInfo, _, err := ResolveFieldAccessType(e, locals, globals, types); err == nil {
			baseType = targetInfo.TypeName
		}
		return exprSecretTainted(e.Base, baseType, locals, globals, funcs, types, module, imports, analysis)
	case *frontend.IndexExpr:
		baseTainted, err := exprSecretTainted(e.Base, "", locals, globals, funcs, types, module, imports, analysis)
		if err != nil || baseTainted {
			return baseTainted, err
		}
		return exprSecretTainted(e.Index, "", locals, globals, funcs, types, module, imports, analysis)
	case *frontend.CallExpr:
		if e.Name == "core.secret_unseal_i32" || e.Name == "secret_unseal_i32" {
			return true, nil
		}
		if local, ok := locals[e.Name]; ok && local.FunctionTypeValue && analysis.localSecretTainted(e.Name) {
			return true, nil
		}
		if enumType, _, ok, err := resolveEnumCaseConstructorCall(e, types, module, imports); ok || err != nil {
			if err != nil {
				return false, err
			}
			if typeUsesSecret(enumType, types) {
				return true, nil
			}
			for _, arg := range e.Args {
				tainted, err := exprSecretTainted(arg, "", locals, globals, funcs, types, module, imports, analysis)
				if err != nil || tainted {
					return tainted, err
				}
			}
			return false, nil
		}
		if info, ok := types[exprType]; ok && info.Kind == TypeStruct && e.Name == exprType {
			for _, arg := range e.Args {
				tainted, err := exprSecretTainted(arg, "", locals, globals, funcs, types, module, imports, analysis)
				if err != nil || tainted {
					return tainted, err
				}
			}
			return false, nil
		}
		resolved, err := resolveCheckedCallName(e.Name, funcs, module, imports, e.At)
		if err != nil {
			for _, arg := range e.Args {
				tainted, taintErr := exprSecretTainted(arg, "", locals, globals, funcs, types, module, imports, analysis)
				if taintErr != nil {
					return false, taintErr
				}
				if tainted {
					return false, privacyDiagnosticf(e.At, "secret-tainted value cannot be passed through unknown callback target '%s'", e.Name)
				}
			}
			return false, nil
		}
		if resolved == "core.secret_unseal_i32" {
			return true, nil
		}
		taintedArgIndexes := make([]int, 0, len(e.Args))
		for idx, arg := range e.Args {
			tainted, err := exprSecretTainted(arg, "", locals, globals, funcs, types, module, imports, analysis)
			if err != nil {
				return false, err
			}
			if tainted {
				taintedArgIndexes = append(taintedArgIndexes, idx)
			}
		}
		if len(taintedArgIndexes) > 0 {
			if actorMailboxSendHasSecretPayload(resolved, taintedArgIndexes) {
				return false, privacyDiagnosticf(e.At, "secret-tainted value cannot be sent through actor mailbox")
			}
			if rawMemoryStoreHasSecretPayload(resolved, taintedArgIndexes) {
				return false, privacyDiagnosticf(e.At, "secret-tainted value cannot be stored through raw memory")
			}
			if runtimeTimeControlHasSecretPayload(resolved, taintedArgIndexes) {
				return false, privacyDiagnosticf(e.At, "secret-tainted value cannot control runtime time")
			}
			if mmioWriteHasSecretPayload(resolved, taintedArgIndexes) {
				return false, privacyDiagnosticf(e.At, "secret-tainted value cannot be written through MMIO")
			}
			if strings.HasPrefix(resolved, "core.") {
				return true, nil
			}
			sig, ok := funcs[resolved]
			if !ok {
				return false, privacyDiagnosticf(e.At, "secret-tainted value cannot be passed through unknown callback target '%s'", e.Name)
			}
			if analysis != nil {
				for _, idx := range taintedArgIndexes {
					if idx >= 0 && idx < len(sig.ParamNames) {
						analysis.markFunctionParamSecretTaint(resolved, sig.ParamNames[idx])
					}
				}
			}
			return true, nil
		}
		if analysis != nil && analysis.funcReturnSecretTaint != nil && analysis.funcReturnSecretTaint[resolved] {
			return true, nil
		}
		return false, nil
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			tainted, err := exprSecretTainted(field.Value, "", locals, globals, funcs, types, module, imports, analysis)
			if err != nil || tainted {
				return tainted, err
			}
		}
		return false, nil
	case *frontend.UnaryExpr:
		return exprSecretTainted(e.X, "", locals, globals, funcs, types, module, imports, analysis)
	case *frontend.BinaryExpr:
		left, err := exprSecretTainted(e.Left, "", locals, globals, funcs, types, module, imports, analysis)
		if err != nil || left {
			return left, err
		}
		return exprSecretTainted(e.Right, "", locals, globals, funcs, types, module, imports, analysis)
	case *frontend.TryExpr:
		return exprSecretTainted(e.X, exprType, locals, globals, funcs, types, module, imports, analysis)
	case *frontend.AwaitExpr:
		return exprSecretTainted(e.X, exprType, locals, globals, funcs, types, module, imports, analysis)
	case *frontend.MatchExpr:
		scrutTainted, err := exprSecretTainted(e.Value, "", locals, globals, funcs, types, module, imports, analysis)
		if err != nil || scrutTainted {
			return scrutTainted, err
		}
		for _, c := range e.Cases {
			if c.Guard != nil {
				guardTainted, err := exprSecretTainted(c.Guard, "", locals, globals, funcs, types, module, imports, analysis)
				if err != nil || guardTainted {
					return guardTainted, err
				}
			}
			tainted, err := exprSecretTainted(c.Value, exprType, locals, globals, funcs, types, module, imports, analysis)
			if err != nil || tainted {
				return tainted, err
			}
		}
	case *frontend.CatchExpr:
		callTainted, err := exprSecretTainted(e.Call, exprType, locals, globals, funcs, types, module, imports, analysis)
		if err != nil || callTainted {
			return callTainted, err
		}
		for _, c := range e.Cases {
			tainted, err := exprSecretTainted(c.Value, exprType, locals, globals, funcs, types, module, imports, analysis)
			if err != nil || tainted {
				return tainted, err
			}
		}
	}
	return false, nil
}

func actorMailboxSendHasSecretPayload(resolved string, taintedArgIndexes []int) bool {
	for _, idx := range taintedArgIndexes {
		switch resolved {
		case "core.send", "core.send_typed":
			if idx == 1 {
				return true
			}
		case "core.send_msg":
			if idx == 1 || idx == 2 {
				return true
			}
		}
	}
	return false
}

func rawMemoryStoreHasSecretPayload(resolved string, taintedArgIndexes []int) bool {
	switch resolved {
	case "core.store_i32", "core.store_u8", "core.store_ptr":
	default:
		return false
	}
	for _, idx := range taintedArgIndexes {
		if idx == 1 {
			return true
		}
	}
	return false
}

func runtimeTimeControlHasSecretPayload(resolved string, taintedArgIndexes []int) bool {
	switch resolved {
	case "core.sleep_ms", "core.sleep_until":
	default:
		return false
	}
	for _, idx := range taintedArgIndexes {
		if idx == 0 {
			return true
		}
	}
	return false
}

func mmioWriteHasSecretPayload(resolved string, taintedArgIndexes []int) bool {
	if resolved != "core.mmio_write_i32" {
		return false
	}
	for _, idx := range taintedArgIndexes {
		if idx == 1 {
			return true
		}
	}
	return false
}

func bindPatternSecretTaintLocals(pattern frontend.Expr, fallbackName string, tainted bool, analysis *functionAnalysisState) {
	if !tainted || analysis == nil {
		return
	}
	if fallbackName != "" {
		analysis.setLocalSecretTaint(fallbackName, true)
	}
	switch p := pattern.(type) {
	case *frontend.IdentExpr:
		analysis.setLocalSecretTaint(p.Name, true)
	case *frontend.SomePatternExpr:
		analysis.setLocalSecretTaint(p.Name, true)
	case *frontend.EnumCasePatternExpr:
		for _, binding := range p.Bindings {
			analysis.setLocalSecretTaint(binding, true)
		}
	}
}

func typeUsesSecretVisited(typeName string, types map[string]*TypeInfo, visiting map[string]struct{}) bool {
	if typeName == "" {
		return false
	}
	if strings.HasPrefix(typeName, "secret.") {
		return true
	}
	if _, seen := visiting[typeName]; seen {
		return false
	}
	visiting[typeName] = struct{}{}
	defer delete(visiting, typeName)

	if info, ok := types[typeName]; ok {
		switch info.Kind {
		case TypeStruct:
			for _, field := range info.Fields {
				if functionTypedFieldUsesSecret(field, types, visiting) || typeUsesSecretVisited(field.TypeName, types, visiting) {
					return true
				}
			}
		case TypeEnum:
			for _, enumCase := range info.EnumCases {
				for index, payloadType := range enumCase.PayloadTypes {
					if enumPayloadFunctionUsesSecret(enumCase, index, types, visiting) ||
						typeUsesSecretVisited(payloadType, types, visiting) {
						return true
					}
				}
			}
		case TypeArray, TypeOptional, TypeSlice:
			return typeUsesSecretVisited(info.ElemType, types, visiting)
		}
	}
	if strings.HasSuffix(typeName, "?") {
		return typeUsesSecretVisited(strings.TrimSuffix(typeName, "?"), types, visiting)
	}
	if strings.HasPrefix(typeName, "[]") {
		return typeUsesSecretVisited(strings.TrimPrefix(typeName, "[]"), types, visiting)
	}
	if _, elem, ok := parseArrayTypeName(typeName); ok {
		return typeUsesSecretVisited(elem, types, visiting)
	}
	return false
}

func findFuncDecl(world *module.World, name string) (*frontend.FuncDecl, string, map[string]string, error) {
	for _, file := range world.Files {
		for _, fn := range file.Funcs {
			if qualifyName(file.Module, fn.Name) == name || fn.Name == name {
				imports, err := collectImportAliases(file)
				if err != nil {
					return nil, "", nil, err
				}
				return fn, file.Module, imports, nil
			}
		}
	}
	return nil, "", nil, nil
}

func compareProtocolRequirement(typeName, protoName string, req frontend.FuncSigDecl, method *frontend.FuncDecl, methodModule string, methodImports map[string]string) error {
	if len(req.TypeParams) != len(method.TypeParams) {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': generic parameter count differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	typeParamMap := make(map[string]string, len(req.TypeParams))
	for i := range req.TypeParams {
		typeParamMap[req.TypeParams[i]] = method.TypeParams[i]
	}
	methodTypeParams := make(map[string]struct{}, len(method.TypeParams))
	for _, name := range method.TypeParams {
		methodTypeParams[name] = struct{}{}
	}
	methodParamTypes := make([]frontend.TypeRef, len(method.Params))
	for i := range method.Params {
		normalized, err := normalizeProtocolComparisonTypeRef(method.Params[i].Type, methodModule, methodImports, methodTypeParams)
		if err != nil {
			return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': parameter %d type differs: %v", frontend.FormatPos(method.Params[i].At), method.Name, protoName, req.Name, i+1, err)
		}
		methodParamTypes[i] = normalized
	}
	methodReturnType, err := normalizeProtocolComparisonTypeRef(method.ReturnType, methodModule, methodImports, methodTypeParams)
	if err != nil {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': return type differs: %v", frontend.FormatPos(method.ReturnType.At), method.Name, protoName, req.Name, err)
	}
	methodThrows := method.Throws
	if method.HasThrows {
		normalized, err := normalizeProtocolComparisonTypeRef(method.Throws, methodModule, methodImports, methodTypeParams)
		if err != nil {
			return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': throws type differs: %v", frontend.FormatPos(method.Throws.At), method.Name, protoName, req.Name, err)
		}
		methodThrows = normalized
	}
	if len(req.Params) != len(method.Params) {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': parameter count differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	if len(req.Params) == 0 {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': missing self parameter", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	if req.Params[0].Name != "self" {
		return fmt.Errorf("%s: protocol '%s' requirement '%s': first parameter must be 'self'", frontend.FormatPos(req.Params[0].At), protoName, req.Name)
	}
	if method.Params[0].Name != "self" {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': first parameter must be 'self'", frontend.FormatPos(method.Params[0].At), method.Name, protoName, req.Name)
	}
	if genericTypeName(req.Params[0].Type) != typeName {
		return fmt.Errorf("%s: protocol '%s' requirement '%s': self parameter type must be '%s'", frontend.FormatPos(req.Params[0].At), protoName, req.Name, typeName)
	}
	if genericTypeName(methodParamTypes[0]) != typeName {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': self parameter type must be '%s'", frontend.FormatPos(method.Params[0].At), method.Name, protoName, req.Name, typeName)
	}
	for i := range req.Params {
		if req.Params[i].Ownership != method.Params[i].Ownership {
			return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': parameter %d ownership differs: expected '%s', got '%s'", frontend.FormatPos(method.Params[i].At), method.Name, protoName, req.Name, i+1, ownershipDisplay(req.Params[i].Ownership), ownershipDisplay(method.Params[i].Ownership))
		}
		if !protocolTypeRefsEquivalent(req.Params[i].Type, methodParamTypes[i], typeParamMap) {
			return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': parameter %d type differs", frontend.FormatPos(method.Params[i].At), method.Name, protoName, req.Name, i+1)
		}
	}
	if !protocolTypeRefsEquivalent(req.ReturnType, methodReturnType, typeParamMap) {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': return type differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	if req.Async != method.Async {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': async marker differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	if req.HasThrows != method.HasThrows || !protocolTypeRefsEquivalent(req.Throws, methodThrows, typeParamMap) {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': throws type differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	reqEffects, err := normalizeEffects(req.Uses, req.At)
	if err != nil {
		return fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.At), protoName, req.Name, err)
	}
	methodEffects, err := normalizeEffects(method.Uses, method.Pos)
	if err != nil {
		return err
	}
	missing := missingRequiredEffects(reqEffects, methodEffects)
	if len(missing) > 0 {
		return fmt.Errorf("%s: method '%s' for type '%s' does not match protocol '%s' requirement '%s': missing required effects %s", frontend.FormatPos(method.Pos), method.Name, typeName, protoName, req.Name, strings.Join(missing, ", "))
	}
	return nil
}

func normalizeProtocolComparisonTypeRef(ref frontend.TypeRef, module string, imports map[string]string, typeParams map[string]struct{}) (frontend.TypeRef, error) {
	normalized := ref
	name, _, err := resolveProtocolRequirementTypeRef(&normalized, module, imports, typeParams)
	if err != nil {
		return normalized, err
	}
	normalized.Name = name
	return normalized, nil
}

func protocolTypeRefsEquivalent(req frontend.TypeRef, method frontend.TypeRef, typeParamMap map[string]string) bool {
	if req.Kind != method.Kind {
		return false
	}
	switch req.Kind {
	case frontend.TypeRefNamed:
		reqName := genericTypeName(req)
		methodName := genericTypeName(method)
		if mapped, ok := typeParamMap[reqName]; ok {
			return mapped == methodName
		}
		if len(req.TypeArgs) != len(method.TypeArgs) {
			return false
		}
		if req.Name != method.Name {
			return false
		}
		for i := range req.TypeArgs {
			if !protocolTypeRefsEquivalent(req.TypeArgs[i], method.TypeArgs[i], typeParamMap) {
				return false
			}
		}
		return true
	case frontend.TypeRefSlice, frontend.TypeRefOptional:
		if req.Elem == nil || method.Elem == nil {
			return req.Elem == nil && method.Elem == nil
		}
		return protocolTypeRefsEquivalent(*req.Elem, *method.Elem, typeParamMap)
	case frontend.TypeRefArray:
		if req.Len != method.Len {
			return false
		}
		if req.Elem == nil || method.Elem == nil {
			return req.Elem == nil && method.Elem == nil
		}
		return protocolTypeRefsEquivalent(*req.Elem, *method.Elem, typeParamMap)
	case frontend.TypeRefFunction:
		if len(req.Params) != len(method.Params) {
			return false
		}
		for i := range req.Params {
			if ownershipAt(req.ParamOwnership, i) != ownershipAt(method.ParamOwnership, i) {
				return false
			}
		}
		for i := range req.Params {
			if !protocolTypeRefsEquivalent(req.Params[i], method.Params[i], typeParamMap) {
				return false
			}
		}
		if req.Return == nil || method.Return == nil {
			return req.Return == nil && method.Return == nil
		}
		if !protocolTypeRefsEquivalent(*req.Return, *method.Return, typeParamMap) {
			return false
		}
		reqEffects, err := normalizeEffects(req.Uses, req.At)
		if err != nil {
			return false
		}
		methodEffects, err := normalizeEffects(method.Uses, method.At)
		if err != nil {
			return false
		}
		return len(missingRequiredEffects(reqEffects, methodEffects)) == 0 && len(missingRequiredEffects(methodEffects, reqEffects)) == 0
	default:
		return genericTypeName(req) == genericTypeName(method)
	}
}

func missingRequiredEffects(required []string, declared []string) []string {
	if len(required) == 0 {
		return nil
	}
	have := make(map[string]struct{}, len(declared))
	for _, effect := range declared {
		have[effect] = struct{}{}
	}
	var missing []string
	for _, effect := range required {
		if _, ok := have[effect]; ok {
			continue
		}
		missing = append(missing, effect)
	}
	return missing
}

func resolveProtocolRequirementTypeRef(ref *frontend.TypeRef, module string, imports map[string]string, typeParams map[string]struct{}) (string, bool, error) {
	if ref == nil {
		return "", false, fmt.Errorf("missing type")
	}
	switch ref.Kind {
	case frontend.TypeRefNamed:
		if ref.Name == "" {
			return "", false, fmt.Errorf("missing type name")
		}
		if _, ok := typeParams[ref.Name]; ok {
			if len(ref.TypeArgs) > 0 {
				return "", false, fmt.Errorf("generic type parameter '%s' cannot have type arguments", ref.Name)
			}
			return ref.Name, true, nil
		}
		for i := range ref.TypeArgs {
			argName, _, err := resolveProtocolRequirementTypeRef(&ref.TypeArgs[i], module, imports, typeParams)
			if err != nil {
				return "", false, err
			}
			ref.TypeArgs[i].Name = argName
		}
		resolved, err := resolveTypeName(ref, module, imports)
		if err != nil {
			return "", false, err
		}
		ref.Name = resolved
		return resolved, false, nil
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		if ref.Elem == nil {
			return "", false, fmt.Errorf("missing element type")
		}
		elemName, elemGeneric, err := resolveProtocolRequirementTypeRef(ref.Elem, module, imports, typeParams)
		if err != nil {
			return "", false, err
		}
		ref.Elem.Name = elemName
		if elemGeneric {
			return genericTypeName(*ref), true, nil
		}
		resolved, err := resolveTypeName(ref, module, imports)
		if err != nil {
			return "", false, err
		}
		return resolved, false, nil
	case frontend.TypeRefFunction:
		anyGeneric := false
		for i := range ref.Params {
			paramName, paramGeneric, err := resolveProtocolRequirementTypeRef(&ref.Params[i], module, imports, typeParams)
			if err != nil {
				return "", false, err
			}
			ref.Params[i].Name = paramName
			anyGeneric = anyGeneric || paramGeneric
		}
		if ref.Return == nil {
			return "", false, fmt.Errorf("missing function return type")
		}
		retName, retGeneric, err := resolveProtocolRequirementTypeRef(ref.Return, module, imports, typeParams)
		if err != nil {
			return "", false, err
		}
		ref.Return.Name = retName
		if _, err := normalizeEffects(ref.Uses, ref.At); err != nil {
			return "", false, err
		}
		anyGeneric = anyGeneric || retGeneric
		if anyGeneric {
			return genericTypeName(*ref), true, nil
		}
		return "ptr", false, nil
	default:
		return "", false, fmt.Errorf("unsupported type reference in protocol requirement")
	}
}

func validateGenericTypeRef(ref frontend.TypeRef, params map[string]struct{}) error {
	switch ref.Kind {
	case frontend.TypeRefNamed:
		if ref.Name == "" {
			return fmt.Errorf("missing type name")
		}
		if _, ok := params[ref.Name]; ok {
			return nil
		}
		if _, ok := canonicalBuiltinType(ref.Name); ok {
			return nil
		}
		if strings.Contains(ref.Name, ".") {
			return nil
		}
		return nil
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		if ref.Elem == nil {
			return fmt.Errorf("%s: missing element type", frontend.FormatPos(ref.At))
		}
		return validateGenericTypeRef(*ref.Elem, params)
	case frontend.TypeRefFunction:
		for _, param := range ref.Params {
			if err := validateGenericTypeRef(param, params); err != nil {
				return err
			}
		}
		if ref.Return == nil {
			return fmt.Errorf("%s: missing function return type", frontend.FormatPos(ref.At))
		}
		if _, err := normalizeEffects(ref.Uses, ref.At); err != nil {
			return err
		}
		return validateGenericTypeRef(*ref.Return, params)
	default:
		return fmt.Errorf("%s: unsupported generic type reference kind %d", frontend.FormatPos(ref.At), ref.Kind)
	}
}

func genericParamTypeNames(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		out = append(out, formatGenericTypeRef(param.Type))
	}
	return out
}

func genericParamNames(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		out = append(out, param.Name)
	}
	return out
}

func genericParamOwnership(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		out = append(out, param.Ownership)
	}
	return out
}

func genericParamFunctionKinds(params []frontend.ParamDecl) []bool {
	out := make([]bool, 0, len(params))
	for _, param := range params {
		out = append(out, param.Type.Kind == frontend.TypeRefFunction)
	}
	return out
}

func genericParamFunctionParamTypes(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		row := make([]string, 0, len(param.Type.Params))
		for _, p := range param.Type.Params {
			row = append(row, formatGenericTypeRef(p))
		}
		out = append(out, row)
	}
	return out
}

func genericParamFunctionOwnership(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		out = append(out, functionTypeRefParamOwnership(param.Type))
	}
	return out
}

func genericParamFunctionReturnTypes(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction || param.Type.Return == nil {
			out = append(out, "")
			continue
		}
		out = append(out, formatGenericTypeRef(*param.Type.Return))
	}
	return out
}

func genericParamFunctionThrowsTypes(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction || param.Type.Throws == nil {
			out = append(out, "")
			continue
		}
		out = append(out, formatGenericTypeRef(*param.Type.Throws))
	}
	return out
}

func genericParamFunctionEffectTypes(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		effects, err := normalizeEffects(param.Type.Uses, param.Type.At)
		if err != nil {
			out = append(out, nil)
			continue
		}
		out = append(out, effects)
	}
	return out
}

func paramFunctionKinds(params []frontend.ParamDecl) []bool {
	out := make([]bool, 0, len(params))
	for _, param := range params {
		out = append(out, param.Type.Kind == frontend.TypeRefFunction)
	}
	return out
}

func genericParamFunctionReturnOwnership(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, "")
			continue
		}
		out = append(out, functionTypeRefReturnOwnership(param.Type))
	}
	return out
}

func paramFunctionParamTypes(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		row := make([]string, 0, len(param.Type.Params))
		for _, p := range param.Type.Params {
			row = append(row, p.Name)
		}
		out = append(out, row)
	}
	return out
}

func paramFunctionOwnership(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		out = append(out, functionTypeRefParamOwnership(param.Type))
	}
	return out
}

func paramFunctionReturnTypes(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction || param.Type.Return == nil {
			out = append(out, "")
			continue
		}
		out = append(out, param.Type.Return.Name)
	}
	return out
}

func paramFunctionReturnOwnership(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, "")
			continue
		}
		out = append(out, functionTypeRefReturnOwnership(param.Type))
	}
	return out
}

func paramFunctionThrowsTypes(params []frontend.ParamDecl, module string, imports map[string]string) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		throwsType, err := functionTypeRefThrowsType(param.Type, module, imports)
		if err != nil {
			out = append(out, "")
			continue
		}
		out = append(out, throwsType)
	}
	return out
}

func paramFunctionEffectTypes(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		effects, err := normalizeEffects(param.Type.Uses, param.Type.At)
		if err != nil {
			out = append(out, nil)
			continue
		}
		out = append(out, effects)
	}
	return out
}

func throwingReturnSlots(successSlots, errorSlots int) int {
	if successSlots == 1 && errorSlots == 1 {
		// Preserve the compact v0.5 layout for existing single-slot typed errors.
		return 2
	}
	return successSlots + errorSlots + 1
}

func throwingScratchSlots(errorSlots int) int {
	if errorSlots <= 0 {
		return 0
	}
	return errorSlots
}

func formatGenericTypeRef(ref frontend.TypeRef) string {
	switch ref.Kind {
	case frontend.TypeRefSlice:
		if ref.Elem == nil {
			return "[]?"
		}
		return "[]" + formatGenericTypeRef(*ref.Elem)
	case frontend.TypeRefArray:
		if ref.Elem == nil {
			return fmt.Sprintf("[%d]?", ref.Len)
		}
		return fmt.Sprintf("[%d]%s", ref.Len, formatGenericTypeRef(*ref.Elem))
	case frontend.TypeRefOptional:
		if ref.Elem == nil {
			return "?"
		}
		return formatGenericTypeRef(*ref.Elem) + "?"
	case frontend.TypeRefFunction:
		parts := make([]string, 0, len(ref.Params))
		for i, param := range ref.Params {
			formatted := formatGenericTypeRef(param)
			if i < len(ref.ParamOwnership) && ref.ParamOwnership[i] != "" {
				formatted = ref.ParamOwnership[i] + " " + formatted
			}
			parts = append(parts, formatted)
		}
		ret := "?"
		if ref.Return != nil {
			ret = formatGenericTypeRef(*ref.Return)
		}
		out := "fn(" + strings.Join(parts, ", ") + ") -> " + ret
		if ref.Throws != nil {
			out += " throws " + formatGenericTypeRef(*ref.Throws)
		}
		if len(ref.Uses) > 0 {
			out += " uses " + strings.Join(ref.Uses, ", ")
		}
		return out
	default:
		if len(ref.TypeArgs) > 0 {
			args := make([]string, 0, len(ref.TypeArgs))
			for _, arg := range ref.TypeArgs {
				args = append(args, formatGenericTypeRef(arg))
			}
			return ref.Name + "<" + strings.Join(args, ", ") + ">"
		}
		return ref.Name
	}
}
