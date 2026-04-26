package semantics

import (
	"fmt"
	"sort"

	"tetra_language/compiler/internal/frontend"
)

var knownEffects = map[string]string{
	"actors":      "actors",
	"alloc":       "alloc",
	"budget":      "budget",
	"cap.io":      "io",
	"cap.mem":     "mem",
	"capability":  "capability",
	"capsule.io":  "capsule.io",
	"capsule.mem": "capsule.mem",
	"control":     "control",
	"io":          "io",
	"islands":     "islands",
	"link":        "link",
	"mem":         "mem",
	"mmio":        "mmio",
	"privacy":     "privacy",
	"runtime":     "runtime",
}

var effectGroups = map[string][]string{
	"effects.all":     {"actors", "alloc", "budget", "capability", "control", "io", "islands", "link", "mem", "mmio", "privacy", "runtime"},
	"effects.cap.io":  {"capability", "io", "mmio"},
	"effects.cap.mem": {"capability", "mem"},
	"effects.memory":  {"alloc", "islands", "mem"},
	"effects.policy":  {"budget", "privacy"},
	"effects.runtime": {"actors", "control", "link", "runtime"},
}

var capabilityAttenuationGroups = map[string]struct{}{
	"effects.all":     {},
	"effects.cap.io":  {},
	"effects.cap.mem": {},
}

type effectContext struct {
	funcName         string
	declared         map[string]struct{}
	explicitDeclared map[string]struct{}
	allowMissing     bool
	hasCapGroup      bool
}

type normalizedEffects struct {
	declared    map[string]struct{}
	explicit    map[string]struct{}
	hasCapGroup bool
}

func normalizeEffects(raw []string, pos frontend.Position) ([]string, error) {
	normalized, err := normalizeEffectDecl(raw, pos)
	if err != nil {
		return nil, err
	}
	return sortedEffectSet(normalized.declared), nil
}

func normalizeEffectDecl(raw []string, pos frontend.Position) (normalizedEffects, error) {
	declared := make(map[string]struct{}, len(raw))
	explicit := make(map[string]struct{}, len(raw))
	hasCapGroup := false
	for _, name := range raw {
		canonical, ok := knownEffects[name]
		if ok {
			declared[canonical] = struct{}{}
			explicit[canonical] = struct{}{}
			continue
		}
		members, groupOK := effectGroups[name]
		if !groupOK {
			return normalizedEffects{}, fmt.Errorf("%s: unknown effect '%s'", frontend.FormatPos(pos), name)
		}
		if _, ok := capabilityAttenuationGroups[name]; ok {
			hasCapGroup = true
		}
		if err := expandEffectGroup(name, members, declared, map[string]struct{}{name: {}}); err != nil {
			return normalizedEffects{}, fmt.Errorf("%s: %v", frontend.FormatPos(pos), err)
		}
	}
	return normalizedEffects{
		declared:    declared,
		explicit:    explicit,
		hasCapGroup: hasCapGroup,
	}, nil
}

func expandEffectGroup(name string, members []string, out map[string]struct{}, visiting map[string]struct{}) error {
	for _, member := range members {
		if canonical, ok := knownEffects[member]; ok {
			out[canonical] = struct{}{}
			continue
		}
		nested, ok := effectGroups[member]
		if !ok {
			return fmt.Errorf("effect group '%s' contains unknown member '%s'", name, member)
		}
		if _, seen := visiting[member]; seen {
			return fmt.Errorf("effect group '%s' has a cycle via '%s'", name, member)
		}
		visiting[member] = struct{}{}
		if err := expandEffectGroup(member, nested, out, visiting); err != nil {
			return err
		}
		delete(visiting, member)
	}
	return nil
}

func sortedEffectSet(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func effectSet(effects []string) map[string]struct{} {
	out := make(map[string]struct{}, len(effects))
	for _, effect := range effects {
		out[effect] = struct{}{}
	}
	return out
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
	return fmt.Errorf("%s: function '%s' uses effect '%s' but does not declare it", frontend.FormatPos(pos), ctx.funcName, effect)
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
	if _, ok := ctx.declared[permission]; ok {
		return nil
	}
	return fmt.Errorf("%s: function '%s' requires capsule permission '%s' for attenuated effect '%s'", frontend.FormatPos(pos), ctx.funcName, permission, attenuatedEffect)
}
