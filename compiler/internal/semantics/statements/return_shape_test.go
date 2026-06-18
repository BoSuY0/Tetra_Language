package statements

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestListEndsWithReturnRecognizesReturnAndThrow(t *testing.T) {
	if !ListEndsWithReturn([]frontend.Stmt{&frontend.ReturnStmt{}}) {
		t.Fatalf("ListEndsWithReturn did not recognize return")
	}
	if !ListEndsWithReturn([]frontend.Stmt{&frontend.ThrowStmt{}}) {
		t.Fatalf("ListEndsWithReturn did not recognize throw")
	}
	if ListEndsWithReturn(nil) {
		t.Fatalf("ListEndsWithReturn(nil) = true, want false")
	}
}

func TestEndsWithReturnRequiresBothIfBranches(t *testing.T) {
	stmt := &frontend.IfStmt{
		Then: []frontend.Stmt{&frontend.ReturnStmt{}},
		Else: []frontend.Stmt{&frontend.ThrowStmt{}},
	}

	if !EndsWithReturn(stmt) {
		t.Fatalf("EndsWithReturn did not accept both returning branches")
	}
	stmt.Else = nil
	if EndsWithReturn(stmt) {
		t.Fatalf("EndsWithReturn accepted missing else branch")
	}
}

func TestEndsWithReturnRecognizesCompleteOptionalMatch(t *testing.T) {
	stmt := &frontend.MatchStmt{
		Cases: []frontend.MatchCase{
			{Pattern: &frontend.NoneLitExpr{}, Body: []frontend.Stmt{&frontend.ReturnStmt{}}},
			{Pattern: &frontend.SomePatternExpr{}, Body: []frontend.Stmt{&frontend.ThrowStmt{}}},
		},
	}

	if !EndsWithReturn(stmt) {
		t.Fatalf("EndsWithReturn did not accept none/some match")
	}
	stmt.Cases[1].Body = nil
	if EndsWithReturn(stmt) {
		t.Fatalf("EndsWithReturn accepted non-returning match case")
	}
}
