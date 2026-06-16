package semantics

import (
	"tetra_language/compiler/internal/frontend"
	semanticspolicy "tetra_language/compiler/internal/semantics/policy"
)

type effectContext struct {
	funcName         string
	declared         map[string]struct{}
	explicitDeclared map[string]struct{}
	capsulePerms     map[string]struct{}
	allowMissing     bool
	hasCapGroup      bool
}

type normalizedEffects struct {
	declared    map[string]struct{}
	explicit    map[string]struct{}
	hasCapGroup bool
}

func canonicalizeEffectName(name string) (string, bool) {
	return semanticspolicy.CanonicalizeEffectName(name)
}

func normalizeEffects(raw []string, pos frontend.Position) ([]string, error) {
	return semanticspolicy.NormalizeEffects(raw, pos, effectDiagnosticf)
}

func normalizeEffectDecl(raw []string, pos frontend.Position) (normalizedEffects, error) {
	normalized, err := semanticspolicy.NormalizeEffectDecl(raw, pos, effectDiagnosticf)
	if err != nil {
		return normalizedEffects{}, err
	}
	return normalizedEffects{
		declared:    normalized.Declared,
		explicit:    normalized.Explicit,
		hasCapGroup: normalized.HasCapGroup,
	}, nil
}

func sortedEffectSet(set map[string]struct{}) []string {
	return semanticspolicy.SortedEffectSet(set)
}

func effectSet(effects []string) map[string]struct{} {
	return semanticspolicy.EffectSet(effects)
}

func newEffectContext(funcName string, effects []string, raw []string, allowMissing bool) *effectContext {
	explicitDeclared := make(map[string]struct{}, len(effects))
	hasCapGroup := false
	if normalized, err := normalizeEffectDecl(raw, frontend.Position{}); err == nil {
		explicitDeclared = normalized.explicit
		hasCapGroup = normalized.hasCapGroup
	} else {
		for _, effect := range effects {
			explicitDeclared[effect] = struct{}{}
		}
	}
	return &effectContext{
		funcName:         funcName,
		declared:         effectSet(effects),
		explicitDeclared: explicitDeclared,
		allowMissing:     allowMissing,
		hasCapGroup:      hasCapGroup,
	}
}

func (ctx *effectContext) require(pos frontend.Position, effect string) error {
	if ctx == nil || ctx.allowMissing {
		return nil
	}
	if _, ok := ctx.declared[effect]; ok {
		return nil
	}
	return effectDiagnosticf(pos, "function '%s' uses effect '%s' but does not declare it", ctx.funcName, effect)
}

func (ctx *effectContext) requireAll(pos frontend.Position, effects []string) error {
	for _, effect := range effects {
		if err := ctx.require(pos, effect); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *effectContext) requireCapsulePermission(pos frontend.Position, permission string, attenuatedEffect string) error {
	if ctx == nil || ctx.allowMissing {
		return nil
	}
	if !ctx.hasCapGroup {
		return nil
	}
	if _, ok := ctx.explicitDeclared[attenuatedEffect]; ok {
		return nil
	}
	if _, ok := ctx.capsulePerms[permission]; ok {
		return nil
	}
	return effectDiagnosticf(pos, "function '%s' requires capsule permission '%s' for attenuated effect '%s'", ctx.funcName, permission, attenuatedEffect)
}
