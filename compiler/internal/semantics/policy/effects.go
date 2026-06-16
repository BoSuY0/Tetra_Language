package policy

import (
	"fmt"
	"sort"

	"tetra_language/compiler/internal/frontend"
)

type DiagnosticFunc func(frontend.Position, string, ...interface{}) error

type NormalizedEffects struct {
	Declared    map[string]struct{}
	Explicit    map[string]struct{}
	HasCapGroup bool
}

var canonicalEffects = map[string]struct{}{
	"actors":     {},
	"alloc":      {},
	"budget":     {},
	"capability": {},
	"control":    {},
	"io":         {},
	"islands":    {},
	"link":       {},
	"mem":        {},
	"mmio":       {},
	"privacy":    {},
	"runtime":    {},
	"surface":    {},
}

var effectAliases = map[string]string{
	"cap.io":  "io",
	"cap.mem": "mem",
}

var permissionMarkerEffects = map[string]struct{}{
	"capsule.io":  {},
	"capsule.mem": {},
}

var effectGroups = map[string][]string{
	"effects.all":     {"actors", "alloc", "budget", "capability", "control", "io", "islands", "link", "mem", "mmio", "privacy", "runtime", "surface"},
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

func CanonicalizeEffectName(name string) (string, bool) {
	if canonical, ok := effectAliases[name]; ok {
		return canonical, true
	}
	if _, ok := permissionMarkerEffects[name]; ok {
		return name, true
	}
	if _, ok := canonicalEffects[name]; ok {
		return name, true
	}
	return "", false
}

func NormalizeEffects(raw []string, pos frontend.Position, diagnostic DiagnosticFunc) ([]string, error) {
	normalized, err := NormalizeEffectDecl(raw, pos, diagnostic)
	if err != nil {
		return nil, err
	}
	return SortedEffectSet(normalized.Declared), nil
}

func NormalizeEffectDecl(raw []string, pos frontend.Position, diagnostic DiagnosticFunc) (NormalizedEffects, error) {
	declared := make(map[string]struct{}, len(raw))
	explicit := make(map[string]struct{}, len(raw))
	hasCapGroup := false
	for _, name := range raw {
		canonical, ok := CanonicalizeEffectName(name)
		if ok {
			declared[canonical] = struct{}{}
			explicit[canonical] = struct{}{}
			continue
		}
		members, groupOK := effectGroups[name]
		if !groupOK {
			if diagnostic != nil {
				return NormalizedEffects{}, diagnostic(pos, "unknown effect '%s'", name)
			}
			return NormalizedEffects{}, fmt.Errorf("%s: unknown effect '%s'", frontend.FormatPos(pos), name)
		}
		if _, ok := capabilityAttenuationGroups[name]; ok {
			hasCapGroup = true
		}
		if err := expandEffectGroup(name, members, declared, map[string]struct{}{name: {}}); err != nil {
			return NormalizedEffects{}, fmt.Errorf("%s: %v", frontend.FormatPos(pos), err)
		}
	}
	return NormalizedEffects{
		Declared:    declared,
		Explicit:    explicit,
		HasCapGroup: hasCapGroup,
	}, nil
}

func SortedEffectSet(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func EffectSet(effects []string) map[string]struct{} {
	out := make(map[string]struct{}, len(effects))
	for _, effect := range effects {
		out[effect] = struct{}{}
	}
	return out
}

func expandEffectGroup(name string, members []string, out map[string]struct{}, visiting map[string]struct{}) error {
	for _, member := range members {
		if canonical, ok := CanonicalizeEffectName(member); ok {
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
