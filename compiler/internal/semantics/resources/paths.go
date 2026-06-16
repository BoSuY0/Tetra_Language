package resources

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

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
	return FieldPath(prefix, fmt.Sprintf("$case%d.payload%d", ordinal, index))
}

func JoinPath(prefix string, leaf string) string {
	if leaf == "" {
		return prefix
	}
	if prefix == "" {
		return leaf
	}
	return prefix + "." + leaf
}

func LeafTail(leaf string, head string) (string, bool) {
	if leaf == head {
		return "", true
	}
	prefix := head + "."
	if strings.HasPrefix(leaf, prefix) {
		return strings.TrimPrefix(leaf, prefix), true
	}
	return "", false
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
