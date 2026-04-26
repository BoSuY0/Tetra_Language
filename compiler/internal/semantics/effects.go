package semantics

import (
	"fmt"
	"sort"

	"tetra_language/compiler/internal/frontend"
)

var knownEffects = map[string]string{
	"actors":     "actors",
	"alloc":      "alloc",
	"cap.io":     "io",
	"cap.mem":    "mem",
	"capability": "capability",
	"control":    "control",
	"io":         "io",
	"islands":    "islands",
	"link":       "link",
	"mem":        "mem",
	"mmio":       "mmio",
	"runtime":    "runtime",
}

type effectContext struct {
	funcName     string
	declared     map[string]struct{}
	allowMissing bool
}

func normalizeEffects(raw []string, pos frontend.Position) ([]string, error) {
	seen := make(map[string]struct{}, len(raw))
	for _, name := range raw {
		canonical, ok := knownEffects[name]
		if !ok {
			return nil, fmt.Errorf("%s: unknown effect '%s'", frontend.FormatPos(pos), name)
		}
		seen[canonical] = struct{}{}
	}
	return sortedEffectSet(seen), nil
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

func newEffectContext(funcName string, effects []string, allowMissing bool) *effectContext {
	return &effectContext{
		funcName:     funcName,
		declared:     effectSet(effects),
		allowMissing: allowMissing,
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
