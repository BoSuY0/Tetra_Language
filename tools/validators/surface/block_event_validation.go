package surface

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type BlockEventRouteReport struct {
	Order          int    `json:"order"`
	Kind           string `json:"kind"`
	Policy         string `json:"policy"`
	TargetID       int    `json:"target_id"`
	TargetName     string `json:"target_name"`
	HitTestPath    []int  `json:"hit_test_path,omitempty"`
	DispatchPath   []int  `json:"dispatch_path"`
	CapturePath    []int  `json:"capture_path,omitempty"`
	BubblePath     []int  `json:"bubble_path,omitempty"`
	DirectTargetID int    `json:"direct_target_id"`
	Delivered      bool   `json:"delivered"`
	Rejected       bool   `json:"rejected"`
	RejectReason   string `json:"reject_reason,omitempty"`
	FocusedID      int    `json:"focused_id,omitempty"`
	Editable       bool   `json:"editable,omitempty"`
	Disabled       bool   `json:"disabled,omitempty"`
	TextLen        int    `json:"text_len,omitempty"`
	TextBytesHex   string `json:"text_bytes_hex,omitempty"`
}

type BlockFocusTransitionReport struct {
	Order        int    `json:"order"`
	Helper       string `json:"helper"`
	BeforeID     int    `json:"before_id"`
	AfterID      int    `json:"after_id"`
	Direction    string `json:"direction"`
	GraphDerived bool   `json:"graph_derived"`
	Wrapped      bool   `json:"wrapped"`
}

func validateBlockEventFocusEvidence(report Report) []string {
	if !hasBlockEventFocusEvidence(report) {
		return nil
	}

	var issues []string
	if report.BlockEventQualityLevel != "deterministic-block-events-v1" {
		issues = append(issues, fmt.Sprintf("block_event_quality_level is %q, want deterministic-block-events-v1", report.BlockEventQualityLevel))
	}
	if report.BlockEventPolicy != "capture-bubble-direct-v1" {
		issues = append(issues, fmt.Sprintf("block_event_policy is %q, want capture-bubble-direct-v1", report.BlockEventPolicy))
	}
	if report.BlockEventUnsupportedDragDrop {
		issues = append(issues, "block_event unsupported drag-and-drop claim must be false")
	}
	if report.BlockGraph == nil {
		issues = append(issues, "block_event evidence requires block_graph")
		return issues
	}

	nodes := map[int]BlockGraphNodeReport{}
	for _, node := range report.BlockGraph.Nodes {
		nodes[node.ID] = node
	}
	for _, kind := range []string{"pointer_enter", "pointer_leave", "pointer_move", "pointer_down", "pointer_up", "click", "double_click", "key", "text", "focus", "blur", "scroll", "resize", "close", "frame"} {
		if !containsNormalizedEventKind(report.BlockEventKinds, kind) {
			issues = append(issues, fmt.Sprintf("block_event_kinds require %s", kind))
		}
	}
	if len(report.BlockEventRoutes) == 0 {
		issues = append(issues, "block_event_routes evidence is required")
	}
	if len(report.BlockFocusTransitions) == 0 {
		issues = append(issues, "block_focus_transitions evidence is required")
	}

	lastOrder := 0
	hasNestedHit := false
	hasCaptureBubbleDirect := false
	hasDisabledReject := false
	hasUnfocusedTextReject := false
	hasFocusedTextDeliver := false
	for _, route := range report.BlockEventRoutes {
		if route.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("block_event_routes order %d is not strictly greater than previous order %d", route.Order, lastOrder))
		}
		lastOrder = route.Order
		kind := normalizeEventToken(route.Kind)
		if kind == "" {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] kind is required", route.Order))
		}
		if !validBlockEventKind(kind) {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] kind is %q, want supported Block event kind", route.Order, route.Kind))
		}
		node, ok := nodes[route.TargetID]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] target_id %d is not in block_graph", route.Order, route.TargetID))
			continue
		}
		if strings.TrimSpace(route.TargetName) == "" {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] target_name is required", route.Order))
		} else if route.TargetName != node.Name {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] target_name is %q, want block_graph node name %q", route.Order, route.TargetName, node.Name))
		}
		wantPath, ok := blockGraphPathToRoot(route.TargetID, nodes)
		if !ok {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] target_id %d is not reachable from root", route.Order, route.TargetID))
			continue
		}
		if !intSlicesEqual(route.DispatchPath, wantPath) {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] dispatch_path = %v, want %v", route.Order, route.DispatchPath, wantPath))
		}
		if len(route.HitTestPath) > 0 {
			if !intSlicesEqual(route.HitTestPath, wantPath) {
				issues = append(issues, fmt.Sprintf("block_event_routes[%d] hit_test_path = %v, want %v", route.Order, route.HitTestPath, wantPath))
			}
			if len(route.HitTestPath) >= 3 {
				hasNestedHit = true
			}
		}
		if route.DirectTargetID != route.TargetID {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] direct_target_id = %d, want target_id %d", route.Order, route.DirectTargetID, route.TargetID))
		}
		if route.Delivered == route.Rejected {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] must be exactly one of delivered or rejected", route.Order))
		}
		if route.Rejected && strings.TrimSpace(route.RejectReason) == "" {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] rejected route requires reject_reason", route.Order))
		}
		if strings.TrimSpace(route.Policy) == "" {
			issues = append(issues, fmt.Sprintf("block_event_routes[%d] policy is required", route.Order))
		}
		if normalizeEventPolicy(route.Policy) == "capture-bubble-direct-v1" {
			if !blockEventCapturePathMatches(route.CapturePath, wantPath) {
				issues = append(issues, fmt.Sprintf("block_event_routes[%d] capture_path = %v, want ancestors", route.Order, route.CapturePath))
			}
			if !blockEventBubblePathMatches(route.BubblePath, wantPath) {
				issues = append(issues, fmt.Sprintf("block_event_routes[%d] bubble_path = %v, want reverse ancestors", route.Order, route.BubblePath))
			}
			hasCaptureBubbleDirect = true
		}
		reason := strings.ToLower(route.RejectReason)
		if kind == "click" && route.Disabled && route.Rejected && !route.Delivered && strings.Contains(reason, "disabled") {
			hasDisabledReject = true
		}
		if kind == "text" && route.Editable && route.Rejected && !route.Delivered && strings.Contains(reason, "unfocused") && route.FocusedID != route.TargetID {
			hasUnfocusedTextReject = true
		}
		if kind == "text" && route.Editable && route.Delivered && !route.Rejected && route.FocusedID == route.TargetID && route.TextLen > 0 && strings.TrimSpace(route.TextBytesHex) != "" {
			payload, err := hex.DecodeString(route.TextBytesHex)
			if err != nil {
				issues = append(issues, fmt.Sprintf("block_event_routes[%d] text_bytes_hex is not valid hex", route.Order))
			} else if len(payload) != route.TextLen {
				issues = append(issues, fmt.Sprintf("block_event_routes[%d] text_len = %d, want decoded payload length %d", route.Order, route.TextLen, len(payload)))
			}
			hasFocusedTextDeliver = true
		}
	}
	if !hasNestedHit {
		issues = append(issues, "block_event_routes require nested hit_test_path evidence")
	}
	if !hasCaptureBubbleDirect {
		issues = append(issues, "block_event_routes require capture-bubble-direct policy evidence")
	}
	if !hasDisabledReject {
		issues = append(issues, "block_event_routes require disabled click rejection evidence")
	}
	if !hasUnfocusedTextReject {
		issues = append(issues, "block_event_routes require unfocused text rejection evidence")
	}
	if !hasFocusedTextDeliver {
		issues = append(issues, "block_event_routes require focused editable text delivery evidence")
	}

	expectedFocus := report.BlockGraph.FocusOrder
	if len(expectedFocus) < 2 {
		issues = append(issues, "block_focus_transitions require at least two focusable Block IDs")
	}
	hasGraphDerived := false
	hasWrap := false
	lastFocusOrder := 0
	for _, transition := range report.BlockFocusTransitions {
		if transition.Order <= lastFocusOrder {
			issues = append(issues, fmt.Sprintf("block_focus_transitions order %d is not strictly greater than previous order %d", transition.Order, lastFocusOrder))
		}
		lastFocusOrder = transition.Order
		if transition.Helper != "tree_focus_next" && transition.Helper != "tree_focus_prev" {
			issues = append(issues, fmt.Sprintf("block_focus_transitions[%d] helper is %q, want tree_focus_next or tree_focus_prev", transition.Order, transition.Helper))
		}
		if !transition.GraphDerived {
			issues = append(issues, fmt.Sprintf("block_focus_transitions[%d] must prove graph_derived", transition.Order))
		}
		if !containsInt(expectedFocus, transition.BeforeID) || !containsInt(expectedFocus, transition.AfterID) {
			issues = append(issues, fmt.Sprintf("block_focus_transitions[%d] before/after must be in block_graph focus_order %v", transition.Order, expectedFocus))
		}
		if transition.GraphDerived {
			hasGraphDerived = true
		}
		if transition.Wrapped && len(expectedFocus) >= 2 {
			if transition.BeforeID != expectedFocus[len(expectedFocus)-1] || transition.AfterID != expectedFocus[0] {
				issues = append(issues, fmt.Sprintf("block_focus_transitions[%d] wrap = %d -> %d, want %d -> %d", transition.Order, transition.BeforeID, transition.AfterID, expectedFocus[len(expectedFocus)-1], expectedFocus[0]))
			}
			hasWrap = true
		}
	}
	if !hasGraphDerived || !hasWrap {
		issues = append(issues, "block_focus_transitions require graph-derived tab wrap evidence")
	}

	for _, required := range []string{
		"block event nested hit-test path",
		"block event capture bubble direct policy",
		"block event disabled click rejected",
		"block event text input focused only",
		"block focus tab order graph-derived",
		"block event no complex drag claim",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("block_event report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockEventFocusEvidence(report Report) bool {
	return len(report.BlockEventRoutes) > 0 ||
		len(report.BlockFocusTransitions) > 0 ||
		len(report.BlockEventKinds) > 0 ||
		strings.TrimSpace(report.BlockEventPolicy) != "" ||
		strings.TrimSpace(report.BlockEventQualityLevel) != "" ||
		report.BlockEventUnsupportedDragDrop
}

func normalizeEventToken(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "-", "_")))
}

func normalizeEventPolicy(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func validBlockEventKind(value string) bool {
	switch normalizeEventToken(value) {
	case "pointer_enter", "pointer_leave", "pointer_move", "pointer_down", "pointer_up", "click", "double_click", "key", "text", "focus", "blur", "scroll", "resize", "close", "frame":
		return true
	default:
		return false
	}
}

func containsNormalizedEventKind(values []string, want string) bool {
	want = normalizeEventToken(want)
	for _, value := range values {
		if normalizeEventToken(value) == want {
			return true
		}
	}
	return false
}

func blockEventCapturePathMatches(got []int, fullPath []int) bool {
	if len(fullPath) < 2 {
		return false
	}
	return intSlicesEqual(got, fullPath[:len(fullPath)-1])
}

func blockEventBubblePathMatches(got []int, fullPath []int) bool {
	if len(fullPath) < 2 || len(got) != len(fullPath)-1 {
		return false
	}
	for i := range got {
		if got[i] != fullPath[len(fullPath)-2-i] {
			return false
		}
	}
	return true
}
