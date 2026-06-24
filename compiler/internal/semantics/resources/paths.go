package resources

import "tetra_language/compiler/internal/frontend"

func PathForExpr(expr frontend.Expr) (string, bool) {
	base, fields, ok := splitFieldPath(expr)
	if !ok {
		return "", false
	}
	path := base
	for _, field := range fields {
		path = FieldPath(path, field)
	}
	return path, true
}

func FieldPath(prefix string, field string) string {
	if prefix == "" {
		return field
	}
	return prefix + "." + field
}

func EnumPayloadPath(prefix string, ordinal int32, index int) string {
	return Path(prefix).EnumPayload(ordinal, index).String()
}

func LeafTail(leaf string, head string) (string, bool) {
	tail, ok := Path(leaf).RelativeTo(Path(head))
	return tail.String(), ok
}

func splitFieldPath(expr frontend.Expr) (string, []string, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name, nil, true
	case *frontend.FieldAccessExpr:
		baseName, fields, ok := splitFieldPath(e.Base)
		if !ok {
			return "", nil, false
		}
		return baseName, append(fields, e.Field), true
	default:
		return "", nil, false
	}
}
