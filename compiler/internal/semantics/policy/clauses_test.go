package policy

import (
	"errors"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestParseFunctionClausePolicyCapturesBudgetAndConsent(t *testing.T) {
	fn := &frontend.FuncDecl{
		SemanticClauses: []frontend.SemanticClause{
			{Name: "noalloc"},
			{Name: "noblock"},
			{Name: "realtime"},
			{Name: "budget", Value: &frontend.NumberExpr{Value: 42}},
			{Name: "privacy"},
			{Name: "consent", Value: &frontend.IdentExpr{Name: "token"}},
		},
	}

	got, err := ParseFunctionClausePolicy(fn, constI32ForPolicyTests, nil)
	if err != nil {
		t.Fatalf("ParseFunctionClausePolicy returned error: %v", err)
	}

	if !got.HasNoAlloc || !got.HasNoBlock || !got.HasRealtime || !got.HasBudget || !got.HasPrivacy {
		t.Fatalf("policy flags = %#v, want all clause flags set", got)
	}
	if got.Budget != 42 {
		t.Fatalf("Budget = %d, want 42", got.Budget)
	}
	if got.ConsentParam != "token" {
		t.Fatalf("ConsentParam = %q, want token", got.ConsentParam)
	}
}

func TestValidateSemanticClausesUsesPrivacyDiagnosticForDuplicateConsent(t *testing.T) {
	sentinel := errors.New("privacy diagnostic")
	fn := &frontend.FuncDecl{
		SemanticClauses: []frontend.SemanticClause{
			{Name: "consent", Value: &frontend.IdentExpr{Name: "a"}},
			{Name: "consent", Value: &frontend.IdentExpr{Name: "b"}},
		},
	}

	err := ValidateSemanticClauses(fn, constI32ForPolicyTests, func(frontend.Position, string, ...interface{}) error {
		return sentinel
	}, nil)

	if !errors.Is(err, sentinel) {
		t.Fatalf("ValidateSemanticClauses error = %v, want privacy sentinel", err)
	}
}

func TestValidateSemanticClausesUsesBudgetDiagnosticForMissingBudgetValue(t *testing.T) {
	sentinel := errors.New("budget diagnostic")
	fn := &frontend.FuncDecl{
		SemanticClauses: []frontend.SemanticClause{
			{Name: "budget"},
		},
	}

	err := ValidateSemanticClauses(fn, constI32ForPolicyTests, nil, func(frontend.Position, string, ...interface{}) error {
		return sentinel
	})

	if !errors.Is(err, sentinel) {
		t.Fatalf("ValidateSemanticClauses error = %v, want budget sentinel", err)
	}
}

func constI32ForPolicyTests(expr frontend.Expr) (int32, bool) {
	n, ok := expr.(*frontend.NumberExpr)
	if !ok {
		return 0, false
	}
	return n.Value, true
}
