package surface

import (
	"fmt"
	"strings"
)

type BlockStateSelectorReport struct {
	Order    int    `json:"order"`
	Name     string `json:"name"`
	BlockID  int    `json:"block_id"`
	Flags    int    `json:"flags"`
	Hovered  bool   `json:"hovered,omitempty"`
	Pressed  bool   `json:"pressed,omitempty"`
	Focused  bool   `json:"focused,omitempty"`
	Selected bool   `json:"selected,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
	Error    bool   `json:"error,omitempty"`
	Loading  bool   `json:"loading,omitempty"`
}

type BlockStateResolutionReport struct {
	Order        int    `json:"order"`
	BlockID      int    `json:"block_id"`
	Selector     string `json:"selector"`
	ResolverStep string `json:"resolver_step"`
	Property     string `json:"property"`
	Before       string `json:"before"`
	After        string `json:"after"`
	Applied      bool   `json:"applied"`
}

func validateBlockStateEvidence(report Report) []string {
	if !hasBlockStateEvidence(report) {
		return nil
	}

	var issues []string
	if report.BlockStateQualityLevel != "deterministic-block-state-resolver-v1" {
		issues = append(issues, fmt.Sprintf("block_state_quality_level is %q, want deterministic-block-state-resolver-v1", report.BlockStateQualityLevel))
	}
	if report.BlockStateUnsupportedCSSPseudos {
		issues = append(issues, "block_state unsupported css pseudo parity claim must be false")
	}
	expectedOrder := []string{"base", "variant", "hover", "pressed", "focused", "selected", "disabled", "error", "loading", "motion"}
	if !normalizedStringListEqual(report.BlockStateResolverOrder, expectedOrder) {
		issues = append(issues, fmt.Sprintf("block_state resolver order = %v, want %v", report.BlockStateResolverOrder, expectedOrder))
	}
	if len(report.BlockStateSelectors) == 0 {
		issues = append(issues, "block_state_selectors evidence is required")
	}
	if len(report.BlockStateResolutions) == 0 {
		issues = append(issues, "block_state_resolutions evidence is required")
	}

	expectedSelectors := map[string]struct {
		flag  int
		check func(BlockStateSelectorReport) bool
	}{
		"hover":    {flag: 1, check: func(selector BlockStateSelectorReport) bool { return selector.Hovered }},
		"pressed":  {flag: 2, check: func(selector BlockStateSelectorReport) bool { return selector.Pressed }},
		"focused":  {flag: 4, check: func(selector BlockStateSelectorReport) bool { return selector.Focused }},
		"selected": {flag: 8, check: func(selector BlockStateSelectorReport) bool { return selector.Selected }},
		"disabled": {flag: 16, check: func(selector BlockStateSelectorReport) bool { return selector.Disabled }},
		"error":    {flag: 32, check: func(selector BlockStateSelectorReport) bool { return selector.Error }},
		"loading":  {flag: 64, check: func(selector BlockStateSelectorReport) bool { return selector.Loading }},
	}
	seenSelectors := map[string]bool{}
	lastSelectorOrder := 0
	for _, selector := range report.BlockStateSelectors {
		if selector.Order <= lastSelectorOrder {
			issues = append(issues, fmt.Sprintf("block_state_selectors order %d is not strictly greater than previous order %d", selector.Order, lastSelectorOrder))
		}
		lastSelectorOrder = selector.Order
		name := normalizeStateToken(selector.Name)
		spec, ok := expectedSelectors[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_state_selectors[%d] name is %q, want supported Block state selector", selector.Order, selector.Name))
			continue
		}
		if selector.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("block_state_selectors[%d] block_id must be positive", selector.Order))
		}
		if selector.Flags != spec.flag {
			issues = append(issues, fmt.Sprintf("block_state_selectors[%d] %s flags = %d, want %d", selector.Order, name, selector.Flags, spec.flag))
		}
		if !spec.check(selector) {
			issues = append(issues, fmt.Sprintf("block_state_selectors[%d] %s selector boolean evidence is missing", selector.Order, name))
		}
		seenSelectors[name] = true
	}
	for name := range expectedSelectors {
		if !seenSelectors[name] {
			issues = append(issues, fmt.Sprintf("block_state_selectors require %s selector evidence", name))
		}
	}

	requiredProperties := map[string]map[string]bool{
		"hover":    {"paint.fill": false},
		"pressed":  {"layout.scale": false},
		"focused":  {"paint.outline": false},
		"selected": {"accessibility.selected": false},
		"disabled": {"input.disabled": false, "text.opacity": false},
		"error":    {"paint.outline_color": false},
		"loading":  {"text.content": false},
		"motion":   {"motion.transition_ms": false},
	}
	lastResolutionOrder := 0
	for _, resolution := range report.BlockStateResolutions {
		if resolution.Order <= lastResolutionOrder {
			issues = append(issues, fmt.Sprintf("block_state_resolutions order %d is not strictly greater than previous order %d", resolution.Order, lastResolutionOrder))
		}
		lastResolutionOrder = resolution.Order
		selector := normalizeStateToken(resolution.Selector)
		step := normalizeStateToken(resolution.ResolverStep)
		property := normalizeStateProperty(resolution.Property)
		properties, ok := requiredProperties[selector]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] selector is %q, want supported selector or motion", resolution.Order, resolution.Selector))
			continue
		}
		if resolution.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] block_id must be positive", resolution.Order))
		}
		if step == "" {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] resolver_step is required", resolution.Order))
		} else if step != selector {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] resolver_step is %q, want selector %q", resolution.Order, resolution.ResolverStep, resolution.Selector))
		}
		if property == "" {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] property is required", resolution.Order))
		}
		if strings.TrimSpace(resolution.Before) == "" || strings.TrimSpace(resolution.After) == "" {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] before and after values are required", resolution.Order))
		}
		if !resolution.Applied {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] %s %s override must be applied", resolution.Order, selector, property))
		}
		if resolution.Applied && strings.TrimSpace(resolution.Before) == strings.TrimSpace(resolution.After) {
			issues = append(issues, fmt.Sprintf("block_state_resolutions[%d] %s %s before/after must change", resolution.Order, selector, property))
		}
		if _, want := properties[property]; want {
			properties[property] = true
		}
	}
	for selector, properties := range requiredProperties {
		for property, seen := range properties {
			if !seen {
				issues = append(issues, fmt.Sprintf("block_state_resolutions require %s %s evidence", selector, property))
			}
		}
	}

	if !hasEventTargetKind(report.Events, "StateBlock", "mouse_move") ||
		!hasEventTargetKind(report.Events, "StateBlock", "mouse_down") ||
		!hasEventTargetKind(report.Events, "StateBlock", "key_down") {
		issues = append(issues, "block_state evidence requires StateBlock hover/press/focus events")
	}
	for _, field := range []string{"selector_flags", "resolved_fill", "resolved_scale", "disabled", "error", "loading"} {
		if !hasTransition(report.StateTransitions, "StateBlock", field) {
			issues = append(issues, fmt.Sprintf("block_state evidence requires StateBlock %s state transition", field))
		}
	}
	if len(report.Frames) < 2 || strings.TrimSpace(report.Frames[0].Checksum) == "" || report.Frames[0].Checksum == report.Frames[1].Checksum {
		issues = append(issues, "block_state frame checksum evidence must show state-driven visual change")
	}
	for _, required := range []string{
		"block state selector resolver order",
		"block state hover fill override",
		"block state pressed scale override",
		"block state focus selected metadata",
		"block state disabled error loading overrides",
		"block state frame checksum changed",
		"block state no css pseudo parity",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("block_state report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockStateEvidence(report Report) bool {
	return len(report.BlockStateSelectors) > 0 ||
		len(report.BlockStateResolutions) > 0 ||
		len(report.BlockStateResolverOrder) > 0 ||
		strings.TrimSpace(report.BlockStateQualityLevel) != "" ||
		report.BlockStateUnsupportedCSSPseudos
}

func normalizeStateToken(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "-", "_")))
}

func normalizeStateProperty(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "-", "_")))
}

func normalizedStringListEqual(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if normalizeStateToken(got[i]) != normalizeStateToken(want[i]) {
			return false
		}
	}
	return true
}
