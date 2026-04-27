package lower

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestLowerUnsupportedStatementNamesFeature(t *testing.T) {
	err := (&lowerer{}).lowerStmt(&frontend.ExpectStmt{
		At: frontend.Position{File: "lower.tetra", Line: 4, Col: 3},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	for _, want := range []string{"lower.tetra:4:3", "unsupported statement kind", "ExpectStmt"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
}

func TestLowerUnsupportedExpressionNamesFeature(t *testing.T) {
	errExpr := &frontend.SomePatternExpr{At: frontend.Position{File: "lower.tetra", Line: 5, Col: 9}}
	_, err := (&lowerer{}).lowerExpr(errExpr)
	if err == nil {
		t.Fatalf("expected error")
	}
	for _, want := range []string{"lower.tetra:5:9", "unsupported expression kind", "SomePatternExpr"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
}

func TestLowerUnsupportedOperatorsNameOperator(t *testing.T) {
	pos := frontend.Position{File: "lower.tetra", Line: 6, Col: 5}
	l := &lowerer{}
	_, err := l.lowerExpr(&frontend.UnaryExpr{
		At: pos,
		Op: frontend.TokenQuestion,
		X:  &frontend.NumberExpr{At: pos, Value: 1},
	})
	if err == nil {
		t.Fatalf("expected unary operator error")
	}
	if !strings.Contains(err.Error(), "unsupported unary operator '?'") {
		t.Fatalf("error = %v", err)
	}

	l = &lowerer{}
	_, err = l.lowerExpr(&frontend.BinaryExpr{
		At:    pos,
		Op:    frontend.TokenQuestion,
		Left:  &frontend.NumberExpr{At: pos, Value: 1},
		Right: &frontend.NumberExpr{At: pos, Value: 2},
	})
	if err == nil {
		t.Fatalf("expected binary operator error")
	}
	if !strings.Contains(err.Error(), "unsupported binary operator '?'") {
		t.Fatalf("error = %v", err)
	}
}

func TestLowerInferUnsupportedExpressionNamesFeature(t *testing.T) {
	errExpr := &frontend.SomePatternExpr{At: frontend.Position{File: "infer.tetra", Line: 8, Col: 13}}
	_, err := (&lowerer{}).inferExprType(errExpr)
	if err == nil {
		t.Fatalf("expected error")
	}
	for _, want := range []string{"infer.tetra:8:13", "unsupported expression kind", "SomePatternExpr"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
}
