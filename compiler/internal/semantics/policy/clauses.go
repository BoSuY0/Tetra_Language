package policy

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
)

type ConstI32Func func(frontend.Expr) (int32, bool)

type FunctionClausePolicy struct {
	HasNoAlloc   bool
	HasNoBlock   bool
	HasRealtime  bool
	HasBudget    bool
	Budget       int32
	HasPrivacy   bool
	ConsentParam string
}

func ValidateSemanticClauses(
	fn *frontend.FuncDecl,
	constI32 ConstI32Func,
	privacyDiagnostic DiagnosticFunc,
	budgetDiagnostic DiagnosticFunc,
) error {
	seen := map[string]frontend.Position{}
	for _, clause := range fn.SemanticClauses {
		if first, exists := seen[clause.Name]; exists {
			if clause.Name == "privacy" || clause.Name == "consent" {
				return diagnosticOrError(
					privacyDiagnostic,
					clause.At,
					"duplicate semantic clause '%s' (first at %s)",
					clause.Name,
					frontend.FormatPos(first),
				)
			}
			return fmt.Errorf(
				"%s: duplicate semantic clause '%s' (first at %s)",
				frontend.FormatPos(clause.At),
				clause.Name,
				frontend.FormatPos(first),
			)
		}
		seen[clause.Name] = clause.At
		switch clause.Name {
		case "noalloc", "noblock", "realtime":
			if clause.Value != nil {
				return fmt.Errorf(
					"%s: semantic clause '%s' does not take arguments",
					frontend.FormatPos(clause.At),
					clause.Name,
				)
			}
		case "privacy":
			if clause.Value != nil {
				return diagnosticOrError(
					privacyDiagnostic,
					clause.At,
					"semantic clause 'privacy' does not take arguments",
				)
			}
		case "nothrow":
			if clause.Value != nil {
				return fmt.Errorf(
					"%s: semantic clause 'nothrow' does not take arguments",
					frontend.FormatPos(clause.At),
				)
			}
			if fn.HasThrows {
				return fmt.Errorf(
					"%s: semantic clause 'nothrow' conflicts with explicit throws type",
					frontend.FormatPos(clause.At),
				)
			}
		case "budget":
			if clause.Value == nil {
				return diagnosticOrError(
					budgetDiagnostic,
					clause.At,
					"semantic clause 'budget' requires an integer argument",
				)
			}
			if constI32 == nil {
				return diagnosticOrError(
					budgetDiagnostic,
					clause.Value.Pos(),
					"semantic clause 'budget' expects an integer constant argument",
				)
			}
			v, ok := constI32(clause.Value)
			if !ok {
				return diagnosticOrError(
					budgetDiagnostic,
					clause.Value.Pos(),
					"semantic clause 'budget' expects an integer constant argument",
				)
			}
			if v < 0 {
				return diagnosticOrError(
					budgetDiagnostic,
					clause.Value.Pos(),
					"semantic clause 'budget' requires a non-negative value",
				)
			}
		case "consent":
			if clause.Value == nil {
				return diagnosticOrError(
					privacyDiagnostic,
					clause.At,
					"semantic clause 'consent' requires a token parameter name",
				)
			}
			if _, ok := clause.Value.(*frontend.IdentExpr); !ok {
				return diagnosticOrError(
					privacyDiagnostic,
					clause.Value.Pos(),
					"semantic clause 'consent' expects an identifier argument",
				)
			}
		default:
			return fmt.Errorf(
				"%s: unknown semantic clause '%s'",
				frontend.FormatPos(clause.At),
				clause.Name,
			)
		}
	}
	return nil
}

func ParseFunctionClausePolicy(
	fn *frontend.FuncDecl,
	constI32 ConstI32Func,
	privacyDiagnostic DiagnosticFunc,
) (FunctionClausePolicy, error) {
	policy := FunctionClausePolicy{}
	for _, clause := range fn.SemanticClauses {
		switch clause.Name {
		case "noalloc":
			policy.HasNoAlloc = true
		case "noblock":
			policy.HasNoBlock = true
		case "realtime":
			policy.HasRealtime = true
		case "budget":
			policy.HasBudget = true
			if constI32 != nil {
				if v, ok := constI32(clause.Value); ok {
					policy.Budget = v
				}
			}
		case "privacy":
			policy.HasPrivacy = true
		case "consent":
			ident, ok := clause.Value.(*frontend.IdentExpr)
			if !ok {
				return FunctionClausePolicy{}, diagnosticOrError(
					privacyDiagnostic,
					clause.At,
					"semantic clause 'consent' expects an identifier argument",
				)
			}
			policy.ConsentParam = ident.Name
		}
	}
	return policy, nil
}

func diagnosticOrError(
	diagnostic DiagnosticFunc,
	pos frontend.Position,
	format string,
	args ...interface{},
) error {
	if diagnostic != nil {
		return diagnostic(pos, format, args...)
	}
	return fmt.Errorf("%s: %s", frontend.FormatPos(pos), fmt.Sprintf(format, args...))
}
