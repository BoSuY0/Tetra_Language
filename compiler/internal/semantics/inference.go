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
		_, targetType, err := ResolveFieldAccessType(e, locals, types)
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
		resolved := ""
		if builtin, ok := ResolveBuiltinAlias(e.Name); ok {
			resolved = builtin
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
			return "", fmt.Errorf("generic function '%s' could not be monomorphized in v0.5; use a same-module call with inferable value arguments", e.Name)
		}
		return sig.ReturnType, nil
	case *frontend.TryExpr:
		call, ok := e.X.(*frontend.CallExpr)
		if !ok {
			return "", fmt.Errorf("try expects a throwing function call")
		}
		resolved := ""
		if builtin, ok := ResolveBuiltinAlias(call.Name); ok {
			resolved = builtin
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
