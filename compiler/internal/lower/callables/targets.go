package callables

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func TargetFromAssignedExpr(
	expr frontend.Expr,
	caller semantics.CheckedFunc,
	funcs map[string]semantics.FuncSig,
	globals map[string]semantics.GlobalInfo,
) (string, bool) {
	if target, ok := FunctionFieldTargetFromExpr(expr, caller.Locals); ok {
		return target, true
	}
	if target, ok := FunctionTypedGlobalFieldTargetFromExpr(expr, globals); ok {
		return target, true
	}
	if target, ok := ImportedFunctionTargetFromExpr(expr, caller.Imports, funcs); ok {
		return target, true
	}
	switch e := expr.(type) {
	case *frontend.ClosureExpr:
		return ClosureTargetName(caller, e, funcs), e.Name != ""
	case *frontend.CallExpr:
		resolved := e.Name
		if builtin, ok := semantics.ResolveBuiltinAlias(resolved); ok {
			resolved = builtin
		}
		if sig, ok := funcs[resolved]; ok && sig.ReturnFunctionType && sig.ReturnFunctionSymbol != "" {
			return sig.ReturnFunctionSymbol, true
		}
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok {
		return "", false
	}
	if local, ok := caller.Locals[id.Name]; ok && local.FunctionValue != "" {
		return local.FunctionValue, true
	}
	if global, ok := globals[id.Name]; ok && global.FunctionTypeValue &&
		global.FunctionValue != "" {
		return global.FunctionValue, true
	}
	if _, ok := funcs[id.Name]; ok {
		return id.Name, true
	}
	return "", false
}

func ClosureTargetName(
	caller semantics.CheckedFunc,
	closure *frontend.ClosureExpr,
	funcs map[string]semantics.FuncSig,
) string {
	if closure == nil || closure.Name == "" {
		return ""
	}
	if _, ok := funcs[closure.Name]; ok {
		return closure.Name
	}
	if caller.Module != "" {
		qualified := caller.Module + "." + closure.Name
		if _, ok := funcs[qualified]; ok {
			return qualified
		}
	}
	return closure.Name
}

func EnumPayloadTargetsFromExpr(
	expr frontend.Expr,
	caller semantics.CheckedFunc,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
) map[string]semantics.FunctionFieldInfo {
	if id, ok := expr.(*frontend.IdentExpr); ok {
		if local, exists := caller.Locals[id.Name]; exists && len(local.EnumPayloadFunctions) > 0 {
			return local.EnumPayloadFunctions
		}
		return nil
	}
	if payloads := EnumPayloadFieldTargetsFromExpr(expr, caller); len(payloads) > 0 {
		return payloads
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok {
		return nil
	}
	if sig, ok := funcs[call.Name]; ok && len(sig.ReturnEnumPayloadFunctions) > 0 {
		return sig.ReturnEnumPayloadFunctions
	}
	_, caseInfo, ok := EnumCaseConstructorInfoForTargets(call, types)
	if !ok {
		return nil
	}
	out := map[string]semantics.FunctionFieldInfo{}
	for i, arg := range call.Args {
		if i >= len(caseInfo.PayloadFunctionTypes) || !caseInfo.PayloadFunctionTypes[i] {
			continue
		}
		target, ok := TargetFromAssignedExpr(arg, caller, funcs, nil)
		if !ok {
			continue
		}
		out[EnumPayloadTargetKey(caseInfo.Ordinal, i)] = EnumPayloadTargetInfo(caseInfo, i, target)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func EnumPayloadFieldTargetsFromExpr(
	expr frontend.Expr,
	caller semantics.CheckedFunc,
) map[string]semantics.FunctionFieldInfo {
	name := FunctionTypedFieldNameFromExpr(expr)
	if name == "" {
		return nil
	}
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return nil
	}
	local, ok := caller.Locals[parts[0]]
	if !ok || len(local.EnumPayloadFields) == 0 {
		return nil
	}
	prefix := strings.Join(parts[1:], ".") + "#"
	out := map[string]semantics.FunctionFieldInfo{}
	for fieldName, fieldInfo := range local.EnumPayloadFields {
		if !strings.HasPrefix(fieldName, prefix) {
			continue
		}
		payloadKey := strings.TrimPrefix(fieldName, prefix)
		if payloadKey == "" {
			continue
		}
		out[payloadKey] = fieldInfo
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func EnumPayloadTargetKey(ordinal int32, index int) string {
	return fmt.Sprintf("%d:%d", ordinal, index)
}

func EnumPayloadTargetInfo(
	caseInfo semantics.EnumCaseInfo,
	index int,
	target string,
) semantics.FunctionFieldInfo {
	info := semantics.FunctionFieldInfo{FunctionValue: target}
	if index >= 0 && index < len(caseInfo.PayloadFunctionParams) {
		info.FunctionParamTypes = append([]string(nil), caseInfo.PayloadFunctionParams[index]...)
	}
	if index >= 0 && index < len(caseInfo.PayloadFunctionOwns) {
		info.FunctionParamOwnership = append([]string(nil), caseInfo.PayloadFunctionOwns[index]...)
	}
	if index >= 0 && index < len(caseInfo.PayloadFunctionReturns) {
		info.FunctionReturnType = caseInfo.PayloadFunctionReturns[index]
	}
	if index >= 0 && index < len(caseInfo.PayloadFunctionEffects) {
		info.FunctionEffects = append([]string(nil), caseInfo.PayloadFunctionEffects[index]...)
	}
	return info
}

func EnumCaseConstructorInfoForTargets(
	call *frontend.CallExpr,
	types map[string]*semantics.TypeInfo,
) (string, semantics.EnumCaseInfo, bool) {
	if call.ResolvedType != "" {
		parts := strings.Split(call.Name, ".")
		if len(parts) >= 2 {
			caseName := parts[len(parts)-1]
			if info, ok := types[call.ResolvedType]; ok && info.Kind == semantics.TypeEnum {
				if caseInfo, ok := info.CaseMap[caseName]; ok {
					return call.ResolvedType, caseInfo, true
				}
			}
		}
	}
	parts := strings.Split(call.Name, ".")
	if len(parts) < 2 {
		return "", semantics.EnumCaseInfo{}, false
	}
	typeName := strings.Join(parts[:len(parts)-1], ".")
	caseName := parts[len(parts)-1]
	info, ok := types[typeName]
	if !ok || info.Kind != semantics.TypeEnum {
		if altName, altInfo, found := FindUniqueEnumByShortName(typeName, types); found {
			typeName = altName
			info = altInfo
		} else {
			return "", semantics.EnumCaseInfo{}, false
		}
	}
	caseInfo, ok := info.CaseMap[caseName]
	if !ok {
		return "", semantics.EnumCaseInfo{}, false
	}
	return typeName, caseInfo, true
}

func EnumCasePatternInfoForTargets(
	pattern *frontend.EnumCasePatternExpr,
	types map[string]*semantics.TypeInfo,
) (semantics.EnumCaseInfo, bool) {
	typeName := pattern.EnumType
	if typeName == "" {
		typeName = pattern.TypeName
	}
	info, ok := types[typeName]
	if !ok || info.Kind != semantics.TypeEnum {
		if altName, altInfo, found := FindUniqueEnumByShortName(typeName, types); found {
			typeName = altName
			info = altInfo
		} else {
			return semantics.EnumCaseInfo{}, false
		}
	}
	caseInfo, ok := info.CaseMap[pattern.CaseName]
	return caseInfo, ok
}

func ResolvedFunctionName(name string, funcs map[string]semantics.FuncSig) (string, bool) {
	resolved := name
	if builtin, ok := semantics.ResolveBuiltinAlias(resolved); ok {
		resolved = builtin
	}
	if _, ok := funcs[resolved]; !ok {
		return "", false
	}
	return resolved, true
}

func TrimFunctionFields(
	fields map[string]semantics.FunctionFieldInfo,
	prefix string,
) map[string]semantics.FunctionFieldInfo {
	out := map[string]semantics.FunctionFieldInfo{}
	for name, field := range fields {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		trimmed := strings.TrimPrefix(name, prefix)
		if trimmed == "" {
			continue
		}
		out[trimmed] = field
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ResolveFunctionFieldName(
	name string,
	locals map[string]semantics.LocalInfo,
) (semantics.FunctionFieldInfo, bool, error) {
	if !strings.Contains(name, ".") {
		return semantics.FunctionFieldInfo{}, false, nil
	}
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return semantics.FunctionFieldInfo{}, false, nil
	}
	local, ok := locals[parts[0]]
	if !ok || len(local.FunctionFields) == 0 {
		return semantics.FunctionFieldInfo{}, false, nil
	}
	field, ok := local.FunctionFields[strings.Join(parts[1:], ".")]
	return field, ok, nil
}

func FunctionTypedFieldNameFromExpr(expr frontend.Expr) string {
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok {
		return ""
	}
	base := FunctionTypedFieldNameFromExpr(field.Base)
	if base == "" {
		if id, ok := field.Base.(*frontend.IdentExpr); ok {
			base = id.Name
		}
	}
	if base == "" || field.Field == "" {
		return ""
	}
	return base + "." + field.Field
}

func FunctionFieldTargetFromExpr(
	expr frontend.Expr,
	locals map[string]semantics.LocalInfo,
) (string, bool) {
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok {
		return "", false
	}
	name := FunctionTypedFieldNameFromExpr(field)
	info, ok, _ := ResolveFunctionFieldName(name, locals)
	if !ok || info.FunctionValue == "" {
		return "", false
	}
	return info.FunctionValue, true
}

func FunctionTypedGlobalFieldTargetFromExpr(
	expr frontend.Expr,
	globals map[string]semantics.GlobalInfo,
) (string, bool) {
	fieldName := FunctionTypedFieldNameFromExpr(expr)
	if fieldName == "" {
		return "", false
	}
	global, ok := globals[fieldName]
	if !ok || !global.FunctionTypeValue || global.FunctionValue == "" {
		return "", false
	}
	return global.FunctionValue, true
}

func ImportedFunctionTargetFromExpr(
	expr frontend.Expr,
	imports map[string]string,
	funcs map[string]semantics.FuncSig,
) (string, bool) {
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok {
		return "", false
	}
	base, ok := field.Base.(*frontend.IdentExpr)
	if !ok {
		return "", false
	}
	module, ok := imports[base.Name]
	if !ok || module == "" {
		return "", false
	}
	name := module + "." + field.Field
	if _, ok := funcs[name]; !ok {
		return "", false
	}
	return name, true
}

func FindUniqueEnumByShortName(
	shortName string,
	types map[string]*semantics.TypeInfo,
) (string, *semantics.TypeInfo, bool) {
	if shortName == "" {
		return "", nil, false
	}
	var foundName string
	var foundInfo *semantics.TypeInfo
	for name, info := range types {
		if info == nil || info.Kind != semantics.TypeEnum {
			continue
		}
		if name == shortName || strings.HasSuffix(name, "."+shortName) {
			if foundInfo != nil {
				return "", nil, false
			}
			foundName = name
			foundInfo = info
		}
	}
	if foundInfo == nil {
		return "", nil, false
	}
	return foundName, foundInfo, true
}
