package semantics

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
)

func inferExprTypeForDecl(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, error) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return "i32", nil
	case *frontend.BoolLitExpr:
		return "bool", nil
	case *frontend.StringLitExpr:
		return "str", nil
	case *frontend.NoneLitExpr:
		return "", fmt.Errorf("cannot infer type from 'none'; add an optional type annotation")
	case *frontend.MatchExpr:
		return inferMatchExprType(e, locals, globals, funcs, types, module, imports)
	case *frontend.CatchExpr:
		return inferCatchExprType(e, locals, globals, funcs, types, module, imports)
	case *frontend.IdentExpr:
		if info, ok := locals[e.Name]; ok {
			if info.TypeName == "" {
				return "", fmt.Errorf("depends on '%s' which has no type annotation", e.Name)
			}
			return info.TypeName, nil
		}
		if g, ok := globals[e.Name]; ok {
			return g.TypeName, nil
		}
		return "", fmt.Errorf("unknown identifier '%s'", e.Name)
	case *frontend.UnaryExpr:
		switch e.Op {
		case frontend.TokenMinus:
			return "i32", nil
		case frontend.TokenBang:
			return "bool", nil
		default:
			return "", fmt.Errorf("unsupported unary operator")
		}
	case *frontend.BinaryExpr:
		switch e.Op {
		case frontend.TokenPlus, frontend.TokenMinus, frontend.TokenStar, frontend.TokenSlash, frontend.TokenPercent:
			return "i32", nil
		case frontend.TokenEqEq, frontend.TokenBangEq, frontend.TokenLess, frontend.TokenGreater, frontend.TokenGreaterEq, frontend.TokenLessEq,
			frontend.TokenAmpAmp, frontend.TokenPipePipe:
			return "bool", nil
		default:
			return "", fmt.Errorf("unsupported binary operator")
		}
	case *frontend.FieldAccessExpr:
		if typeName, _, ok, err := resolveEnumCaseExpr(e, locals, globals, types, module, imports); ok || err != nil {
			if err != nil {
				return "", err
			}
			return typeName, nil
		}
		_, targetType, err := ResolveFieldAccessType(e, locals, globals, types)
		if err != nil {
			return "", err
		}
		return targetType, nil
	case *frontend.IndexExpr:
		baseType, err := inferExprTypeForDecl(e.Base, locals, globals, funcs, types, module, imports)
		if err != nil {
			return "", err
		}
		info, err := ensureTypeInfo(baseType, types)
		if err != nil {
			return "", err
		}
		switch info.Kind {
		case TypeStr:
			return "u8", nil
		case TypeSlice:
			return info.ElemType, nil
		case TypeArray:
			return info.ElemType, nil
		default:
			return "", fmt.Errorf("cannot index '%s'", baseType)
		}
	case *frontend.StructLitExpr:
		resolved, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return "", err
		}
		return resolved, nil
	case *frontend.CallExpr:
		if enumType, _, ok, err := resolveEnumCaseConstructorCall(e, types, module, imports); ok || err != nil {
			if err != nil {
				return "", err
			}
			return enumType, nil
		}
		if builtin, ok := ResolveBuiltinAlias(e.Name); ok && builtin == "core.recv_typed" {
			if len(e.TypeArgs) != 1 {
				return "", fmt.Errorf("recv_typed expects one explicit type argument")
			}
			typeName, err := resolveTypeName(&e.TypeArgs[0], module, imports)
			if err != nil {
				return "", err
			}
			e.TypeArgs[0].Name = typeName
			return typeName, nil
		}
		if builtin, ok := ResolveBuiltinAlias(e.Name); ok && builtin == "core.send_typed" {
			return "i32", nil
		}
		if builtin, ok := ResolveBuiltinAlias(e.Name); ok && (builtin == "core.task_spawn_i32_typed" || builtin == "core.task_spawn_group_i32_typed") {
			if len(e.TypeArgs) != 1 {
				return "", fmt.Errorf("%s expects one explicit error type argument", builtin)
			}
			errorType, err := resolveTypeName(&e.TypeArgs[0], module, imports)
			if err != nil {
				return "", err
			}
			if err := validateTypedTaskErrorType(errorType, types, e.TypeArgs[0].At); err != nil {
				return "", err
			}
			e.TypeArgs[0].Name = errorType
			handleType, _, err := EnsureTypedTaskHandleType(errorType, types)
			if err != nil {
				return "", err
			}
			return handleType, nil
		}
		if builtin, ok := ResolveBuiltinAlias(e.Name); ok && (builtin == "core.task_join_i32_typed" || builtin == "core.task_join_group_i32_typed") {
			return "i32", nil
		}
		if ctorType, ok, err := resolveStructConstructorCallType(e, types, module, imports); ok {
			return ctorType, err
		}
		resolved := ""
		if local, ok := locals[e.Name]; ok {
			if local.FunctionValue == "" || (local.FunctionTypeValue && len(local.FunctionCaptures) == 0 && local.SlotCount == FnPtrSlotCount) {
				if !local.FunctionTypeValue {
					return "", fmt.Errorf("%s", unsupportedFunctionValueCallMessage(e.Name))
				}
				if len(local.FunctionCaptures) > 0 {
					return "", fmt.Errorf("function-typed callback '%s' captures local values; captured function values cannot be called through function type in this MVP", e.Name)
				}
				if len(e.Args) != len(local.FunctionParamTypes) {
					return "", fmt.Errorf("wrong argument count for callback '%s'", e.Name)
				}
				return local.FunctionReturnType, nil
			}
			if local.GenericFunctionValue {
				return "", fmt.Errorf("%s", genericClosureDirectCallRequirementMessage(e.Name))
			}
			if err := appendClosureCaptureArgs(e, local); err != nil {
				return "", err
			}
			resolved = local.FunctionValue
			e.Name = resolved
		} else if builtin, ok := ResolveBuiltinAlias(e.Name); ok {
			resolved = builtin
		} else if _, ok := funcs[e.Name]; ok {
			resolved = e.Name
		} else {
			name, err := resolveCallName(e.Name, module, imports, e.At)
			if err != nil {
				return "", err
			}
			resolved = name
		}
		sig, ok := funcs[resolved]
		if !ok {
			return "", fmt.Errorf("unknown function '%s'", resolved)
		}
		if sig.Generic {
			return "", fmt.Errorf("generic function '%s' could not be monomorphized; use inferable value arguments", e.Name)
		}
		return sig.ReturnType, nil
	case *frontend.ClosureExpr:
		return "ptr", nil
	case *frontend.TryExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			if await, awaitOK := e.X.(*frontend.AwaitExpr); awaitOK {
				call, ok = await.X.(*frontend.CallExpr)
			}
		}
		if !ok {
			return "", fmt.Errorf("try expects a throwing function call")
		}
		resolved := ""
		if builtin, ok := ResolveBuiltinAlias(call.Name); ok {
			resolved = builtin
		} else if _, ok := funcs[call.Name]; ok {
			resolved = call.Name
		} else {
			name, err := resolveCallName(call.Name, module, imports, call.At)
			if err != nil {
				return "", err
			}
			resolved = name
		}
		sig, ok := funcs[resolved]
		if !ok {
			return "", fmt.Errorf("unknown function '%s'", resolved)
		}
		if sig.ThrowsType == "" {
			return "", fmt.Errorf("try expects a throwing function call")
		}
		return sig.ReturnType, nil
	case *frontend.AwaitExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return "", fmt.Errorf("await expects an async function call")
		}
		resolved := ""
		if builtin, ok := ResolveBuiltinAlias(call.Name); ok {
			resolved = builtin
		} else if _, ok := funcs[call.Name]; ok {
			resolved = call.Name
		} else {
			name, err := resolveCallName(call.Name, module, imports, call.At)
			if err != nil {
				return "", err
			}
			resolved = name
		}
		sig, ok := funcs[resolved]
		if !ok {
			return "", fmt.Errorf("unknown function '%s'", resolved)
		}
		if !sig.Async {
			return "", fmt.Errorf("await expects an async function call")
		}
		return sig.ReturnType, nil
	default:
		return "", fmt.Errorf("unsupported expression for type inference")
	}
}

func resolveStructConstructorCallType(
	e *frontend.CallExpr,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, bool, error) {
	if len(e.Args) == 0 || len(e.ArgLabels) != len(e.Args) {
		return "", false, nil
	}
	for _, label := range e.ArgLabels {
		if label == "" {
			return "", false, nil
		}
	}

	ref := frontend.TypeRef{At: e.At, Kind: frontend.TypeRefNamed, Name: e.Name}
	resolved, err := resolveTypeName(&ref, module, imports)
	if err != nil {
		return "", false, nil
	}
	info, ok := types[resolved]
	if !ok || info.Kind != TypeStruct {
		return "", false, nil
	}
	if len(e.Args) != len(info.Fields) {
		return "", true, fmt.Errorf("wrong field count for '%s'", resolved)
	}

	seen := make(map[string]struct{}, len(e.ArgLabels))
	for _, label := range e.ArgLabels {
		if _, exists := seen[label]; exists {
			return "", true, fmt.Errorf("duplicate field '%s'", label)
		}
		seen[label] = struct{}{}
		if _, ok := info.FieldMap[label]; !ok {
			return "", true, fmt.Errorf("unknown field '%s'", label)
		}
	}
	for _, field := range info.Fields {
		if _, ok := seen[field.Name]; !ok {
			return "", true, fmt.Errorf("missing field '%s'", field.Name)
		}
	}
	return resolved, true, nil
}
