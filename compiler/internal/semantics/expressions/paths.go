package expressions

import "tetra_language/compiler/internal/frontend"

func CallbackArgumentName(expr frontend.Expr) string {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.FieldAccessExpr:
		base := CallbackArgumentName(e.Base)
		if base != "" && e.Field != "" {
			return base + "." + e.Field
		}
	case *frontend.CallExpr:
		if e.Name != "" {
			return e.Name + "()"
		}
	}
	return ""
}

func SplitFieldPath(expr frontend.Expr) (string, []string, frontend.Position, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name, nil, e.At, true
	case *frontend.FieldAccessExpr:
		baseName, fields, pos, ok := SplitFieldPath(e.Base)
		if !ok {
			return "", nil, pos, false
		}
		fields = append(fields, e.Field)
		return baseName, fields, e.At, true
	default:
		return "", nil, expr.Pos(), false
	}
}
