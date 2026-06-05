package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

func rewriteSliceViewMethodCall(e *frontend.CallExpr, locals map[string]LocalInfo, globals map[string]GlobalInfo, types map[string]*TypeInfo) (bool, error) {
	if method, ok := syntheticViewMethodName(e.Name); ok {
		return rewriteSyntheticViewMethodCall(e, method, locals, globals, types)
	}
	receiverParts, method, ok := sliceViewMethodParts(e.Name)
	if !ok {
		return false, nil
	}
	if len(receiverParts) == 0 {
		return false, nil
	}
	root := receiverParts[0]
	if _, ok := locals[root]; !ok {
		if _, ok := globals[root]; !ok {
			return false, nil
		}
	}
	if len(e.TypeArgs) > 0 {
		return true, fmt.Errorf("%s: slice view method '%s' does not accept explicit type arguments", frontend.FormatPos(e.At), method)
	}
	wantArgs := viewMethodArgCount(method)
	if len(e.Args) != wantArgs {
		return true, fmt.Errorf("%s: slice view method '%s' expects %d argument(s)", frontend.FormatPos(e.At), method, wantArgs)
	}
	receiverType, err := sliceViewReceiverType(receiverParts, locals, globals, types, e.At)
	if err != nil {
		return true, err
	}
	builtin, ok := sliceViewBuiltin(receiverType, method)
	if !ok {
		return true, unsupportedViewReceiverError(e.At, method, receiverType)
	}
	receiver := exprFromPathParts(receiverParts, e.At)
	args := make([]frontend.Expr, 0, len(e.Args)+1)
	args = append(args, receiver)
	args = append(args, e.Args...)
	e.Name = builtin
	e.Args = args
	e.ArgLabels = nil
	return true, nil
}

func syntheticViewMethodName(name string) (string, bool) {
	const prefix = "__method."
	if !strings.HasPrefix(name, prefix) {
		return "", false
	}
	method := strings.TrimPrefix(name, prefix)
	if isViewMethod(method) {
		return method, true
	}
	return "", false
}

func rewriteSyntheticViewMethodCall(e *frontend.CallExpr, method string, locals map[string]LocalInfo, globals map[string]GlobalInfo, types map[string]*TypeInfo) (bool, error) {
	if len(e.Args) == 0 {
		return true, fmt.Errorf("%s: view method '%s' is missing receiver", frontend.FormatPos(e.At), method)
	}
	if len(e.TypeArgs) > 0 {
		return true, fmt.Errorf("%s: view method '%s' does not accept explicit type arguments", frontend.FormatPos(e.At), method)
	}
	wantArgs := viewMethodArgCount(method)
	if len(e.Args)-1 != wantArgs {
		return true, fmt.Errorf("%s: view method '%s' expects %d argument(s)", frontend.FormatPos(e.At), method, wantArgs)
	}
	receiverType, err := viewReceiverTypeFromExpr(e.Args[0], locals, globals, types, e.At)
	if err != nil {
		return true, err
	}
	builtin, ok := sliceViewBuiltin(receiverType, method)
	if !ok {
		return true, unsupportedViewReceiverError(e.At, method, receiverType)
	}
	e.Name = builtin
	e.ArgLabels = nil
	return true, nil
}

func viewReceiverTypeFromExpr(expr frontend.Expr, locals map[string]LocalInfo, globals map[string]GlobalInfo, types map[string]*TypeInfo, pos frontend.Position) (string, error) {
	switch receiver := expr.(type) {
	case *frontend.StringLitExpr:
		return "str", nil
	case *frontend.IdentExpr:
		if local, ok := locals[receiver.Name]; ok {
			return local.TypeName, nil
		}
		if global, ok := globals[receiver.Name]; ok {
			return global.TypeName, nil
		}
		return "", fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(receiver.At), receiver.Name)
	case *frontend.FieldAccessExpr:
		_, targetType, err := ResolveFieldAccessType(receiver, locals, globals, types)
		return targetType, err
	case *frontend.CallExpr:
		if _, err := rewriteSliceViewMethodCall(receiver, locals, globals, types); err != nil {
			return "", err
		}
		if elem, _, ok := sliceViewElemFromBuiltin(receiver.Name); ok {
			if elem == "str" {
				return "str", nil
			}
			return "[]" + elem, nil
		}
		sigs, err := builtinFuncSigs(types)
		if err != nil {
			return "", err
		}
		if sig, ok := sigs[receiver.Name]; ok {
			return sig.ReturnType, nil
		}
		return "", fmt.Errorf("%s: invalid view method receiver", frontend.FormatPos(pos))
	default:
		return "", fmt.Errorf("%s: invalid view method receiver", frontend.FormatPos(pos))
	}
}

func unsupportedViewReceiverError(pos frontend.Position, method string, receiverType string) error {
	return fmt.Errorf("%s: view method '%s' expects []u8, []u16, []i32, []bool, or String receiver, got '%s'", frontend.FormatPos(pos), method, receiverType)
}

func sliceViewMethodParts(name string) ([]string, string, bool) {
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return nil, "", false
	}
	method := parts[len(parts)-1]
	if isViewMethod(method) {
		return parts[:len(parts)-1], method, true
	}
	return nil, "", false
}

func isViewMethod(method string) bool {
	switch method {
	case "window", "prefix", "suffix", "borrow", "copy", "copy_into":
		return true
	default:
		return false
	}
}

func viewMethodArgCount(method string) int {
	switch method {
	case "window":
		return 2
	case "prefix", "suffix", "copy_into":
		return 1
	default:
		return 0
	}
}

func sliceViewReceiverType(parts []string, locals map[string]LocalInfo, globals map[string]GlobalInfo, types map[string]*TypeInfo, pos frontend.Position) (string, error) {
	if len(parts) == 1 {
		if local, ok := locals[parts[0]]; ok {
			return local.TypeName, nil
		}
		if global, ok := globals[parts[0]]; ok {
			return global.TypeName, nil
		}
		return "", fmt.Errorf("%s: unknown identifier '%s'", frontend.FormatPos(pos), parts[0])
	}
	expr := exprFromPathParts(parts, pos)
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok {
		return "", fmt.Errorf("%s: invalid slice view receiver", frontend.FormatPos(pos))
	}
	_, targetType, err := ResolveFieldAccessType(field, locals, globals, types)
	return targetType, err
}

func exprFromPathParts(parts []string, pos frontend.Position) frontend.Expr {
	if len(parts) == 0 {
		return &frontend.IdentExpr{At: pos, Name: ""}
	}
	expr := frontend.Expr(&frontend.IdentExpr{At: pos, Name: parts[0]})
	for _, field := range parts[1:] {
		expr = &frontend.FieldAccessExpr{At: pos, Base: expr, Field: field}
	}
	return expr
}

func sliceViewBuiltin(typeName, method string) (string, bool) {
	suffix := ""
	switch typeName {
	case "[]u8":
		suffix = "u8"
	case "[]u16":
		suffix = "u16"
	case "[]i32":
		suffix = "i32"
	case "[]bool":
		suffix = "bool"
	case "str", "String":
		return "core.string_" + method, true
	default:
		return "", false
	}
	return "core.slice_" + method + "_" + suffix, true
}

func sliceViewElemFromBuiltin(name string) (elem string, method string, ok bool) {
	if !strings.HasPrefix(name, "core.slice_") {
		if strings.HasPrefix(name, "core.string_") {
			method := strings.TrimPrefix(name, "core.string_")
			switch method {
			case "window", "prefix", "suffix", "borrow", "copy", "copy_into":
				return "str", method, true
			}
		}
		return "", "", false
	}
	rest := strings.TrimPrefix(name, "core.slice_")
	for _, candidate := range []string{"window", "prefix", "suffix", "borrow", "copy", "copy_into"} {
		prefix := candidate + "_"
		if strings.HasPrefix(rest, prefix) {
			elem = strings.TrimPrefix(rest, prefix)
			switch elem {
			case "u8", "u16", "i32", "bool":
				return elem, candidate, true
			}
			return "", "", false
		}
	}
	return "", "", false
}
