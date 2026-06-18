package lower

import (
	"fmt"
	"strings"
	"tetra_language/compiler/internal/frontend"
	lowercallables "tetra_language/compiler/internal/lower/callables"
	"tetra_language/compiler/internal/semantics"
)

func callableTargetFromAssignedExpr(expr frontend.Expr, caller semantics.CheckedFunc, funcs map[string]semantics.FuncSig, globals map[string]semantics.GlobalInfo) (string, bool) {
	return lowercallables.TargetFromAssignedExpr(expr, caller, funcs, globals)
}

func callableClosureTargetName(caller semantics.CheckedFunc, closure *frontend.ClosureExpr, funcs map[string]semantics.FuncSig) string {
	return lowercallables.ClosureTargetName(caller, closure, funcs)
}

func enumPayloadTargetsFromExpr(
	expr frontend.Expr,
	caller semantics.CheckedFunc,
	funcs map[string]semantics.FuncSig,
	types map[string]*semantics.TypeInfo,
) map[string]semantics.FunctionFieldInfo {
	return lowercallables.EnumPayloadTargetsFromExpr(expr, caller, funcs, types)
}

func enumPayloadFieldTargetsFromExpr(expr frontend.Expr, caller semantics.CheckedFunc) map[string]semantics.FunctionFieldInfo {
	return lowercallables.EnumPayloadFieldTargetsFromExpr(expr, caller)
}

func enumPayloadTargetKey(ordinal int32, index int) string {
	return lowercallables.EnumPayloadTargetKey(ordinal, index)
}

func enumPayloadTargetInfo(caseInfo semantics.EnumCaseInfo, index int, target string) semantics.FunctionFieldInfo {
	return lowercallables.EnumPayloadTargetInfo(caseInfo, index, target)
}

func enumCaseConstructorInfoForTargets(call *frontend.CallExpr, types map[string]*semantics.TypeInfo) (string, semantics.EnumCaseInfo, bool) {
	return lowercallables.EnumCaseConstructorInfoForTargets(call, types)
}

func enumCasePatternInfoForTargets(pattern *frontend.EnumCasePatternExpr, types map[string]*semantics.TypeInfo) (semantics.EnumCaseInfo, bool) {
	return lowercallables.EnumCasePatternInfoForTargets(pattern, types)
}

func resolvedCallableFunctionName(name string, funcs map[string]semantics.FuncSig) (string, bool) {
	return lowercallables.ResolvedFunctionName(name, funcs)
}

func trimFunctionFields(fields map[string]semantics.FunctionFieldInfo, prefix string) map[string]semantics.FunctionFieldInfo {
	return lowercallables.TrimFunctionFields(fields, prefix)
}

func resolveFunctionFieldName(name string, locals map[string]semantics.LocalInfo) (semantics.FunctionFieldInfo, bool, error) {
	return lowercallables.ResolveFunctionFieldName(name, locals)
}

func functionTypedFieldNameFromExpr(expr frontend.Expr) string {
	return lowercallables.FunctionTypedFieldNameFromExpr(expr)
}

func functionFieldTargetFromExpr(expr frontend.Expr, locals map[string]semantics.LocalInfo) (string, bool) {
	return lowercallables.FunctionFieldTargetFromExpr(expr, locals)
}

func functionTypedGlobalFieldTargetFromExpr(expr frontend.Expr, globals map[string]semantics.GlobalInfo) (string, bool) {
	return lowercallables.FunctionTypedGlobalFieldTargetFromExpr(expr, globals)
}

func (l *lowerer) functionFieldCallSource(name string, pos frontend.Position) (semantics.FunctionFieldInfo, int, bool, error) {
	parts := strings.Split(name, ".")
	if len(parts) != 2 {
		if len(parts) < 2 {
			return semantics.FunctionFieldInfo{}, 0, false, nil
		}
	}
	local, ok := l.locals[parts[0]]
	if !ok || len(local.FunctionFields) == 0 {
		return semantics.FunctionFieldInfo{}, 0, false, nil
	}
	fieldPath := parts[1:]
	field, ok := local.FunctionFields[strings.Join(fieldPath, ".")]
	if !ok {
		return semantics.FunctionFieldInfo{}, 0, false, nil
	}
	_, slotCount, base, err := resolveFieldChainLower(local.TypeName, local.Base, fieldPath, l.types, pos)
	if err != nil {
		return semantics.FunctionFieldInfo{}, 0, false, err
	}
	if slotCount != semantics.FnPtrSlotCount {
		return semantics.FunctionFieldInfo{}, 0, false, fmt.Errorf("%s: function-typed struct field '%s' slot mismatch", frontend.FormatPos(pos), name)
	}
	return field, base, true, nil
}

func importedFunctionTargetFromExpr(expr frontend.Expr, imports map[string]string, funcs map[string]semantics.FuncSig) (string, bool) {
	return lowercallables.ImportedFunctionTargetFromExpr(expr, imports, funcs)
}
