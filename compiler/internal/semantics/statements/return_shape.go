package statements

import "tetra_language/compiler/internal/frontend"

func ListEndsWithReturn(stmts []frontend.Stmt) bool {
	if len(stmts) == 0 {
		return false
	}
	return EndsWithReturn(stmts[len(stmts)-1])
}

func EndsWithReturn(stmt frontend.Stmt) bool {
	switch s := stmt.(type) {
	case *frontend.ReturnStmt:
		return true
	case *frontend.ThrowStmt:
		return true
	case *frontend.IfStmt:
		return len(s.Then) > 0 && len(s.Else) > 0 && ListEndsWithReturn(s.Then) && ListEndsWithReturn(s.Else)
	case *frontend.IfLetStmt:
		return len(s.Then) > 0 && len(s.Else) > 0 && ListEndsWithReturn(s.Then) && ListEndsWithReturn(s.Else)
	case *frontend.MatchStmt:
		hasDefault := false
		hasNone := false
		hasSome := false
		for _, c := range s.Cases {
			if c.Default {
				hasDefault = true
			} else if _, ok := c.Pattern.(*frontend.NoneLitExpr); ok {
				hasNone = true
			} else if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				hasSome = true
			}
			if !ListEndsWithReturn(c.Body) {
				return false
			}
		}
		return hasDefault || (hasNone && hasSome)
	case *frontend.UnsafeStmt:
		return ListEndsWithReturn(s.Body)
	default:
		return false
	}
}
