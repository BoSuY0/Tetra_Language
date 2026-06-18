package surface

import (
	"fmt"
	"strings"
)

type BlockAccessibilityTreeReport struct {
	Schema                  string                                 `json:"schema"`
	AccessibilityLevel      string                                 `json:"accessibility_level"`
	Source                  string                                 `json:"source"`
	Module                  string                                 `json:"module"`
	QualityLevel            string                                 `json:"quality_level"`
	BlockGraphSchema        string                                 `json:"block_graph_schema"`
	DerivedFromBlockGraph   bool                                   `json:"derived_from_block_graph"`
	ManualBookkeeping       bool                                   `json:"manual_bookkeeping"`
	PlatformHostIntegration bool                                   `json:"platform_host_integration"`
	DOMARIAIntegration      bool                                   `json:"dom_aria_integration"`
	ScreenReaderEvidence    any                                    `json:"screen_reader_evidence"`
	NoDOMUI                 bool                                   `json:"no_dom_ui"`
	NoUserJS                bool                                   `json:"no_user_js"`
	NoPlatformWidgets       bool                                   `json:"no_platform_widgets"`
	NodeCount               int                                    `json:"node_count"`
	FocusableCount          int                                    `json:"focusable_count"`
	RolesPresent            []string                               `json:"roles_present"`
	Nodes                   []BlockAccessibilityNodeReport         `json:"nodes"`
	Relationships           []AccessibilityRelationshipReport      `json:"relationships"`
	FocusOrder              []int                                  `json:"focus_order"`
	ReadingOrder            []int                                  `json:"reading_order"`
	Actions                 []AccessibilityActionReport            `json:"actions"`
	NegativeGuards          BlockAccessibilityNegativeGuardsReport `json:"negative_guards"`
}

type BlockAccessibilityNodeReport struct {
	ID            int        `json:"id"`
	BlockID       int        `json:"block_id"`
	ParentBlockID int        `json:"parent_block_id"`
	Name          string     `json:"name"`
	Role          string     `json:"role"`
	Description   string     `json:"description,omitempty"`
	Value         string     `json:"value,omitempty"`
	State         string     `json:"state,omitempty"`
	Bounds        RectReport `json:"bounds"`
	Visible       bool       `json:"visible"`
	Enabled       bool       `json:"enabled"`
	Focusable     bool       `json:"focusable"`
	Focused       bool       `json:"focused"`
	Editable      bool       `json:"editable"`
	LabelFor      string     `json:"label_for,omitempty"`
	LabelledBy    string     `json:"labelled_by,omitempty"`
	Actions       []string   `json:"actions,omitempty"`
	FocusIndex    int        `json:"focus_index"`
	ReadingIndex  int        `json:"reading_index"`
}

type BlockAccessibilityNegativeGuardsReport struct {
	FocusableActionNameChecked    bool `json:"focusable_action_name_checked"`
	LabelRelationshipsChecked     bool `json:"label_relationships_checked"`
	ReadingOrderGraphChecked      bool `json:"reading_order_graph_checked"`
	BoundsAlignmentChecked        bool `json:"bounds_alignment_checked"`
	FakeScreenReaderClaimRejected bool `json:"fake_screen_reader_claim_rejected"`
	ScopedPlatformClaimChecked    bool `json:"scoped_platform_claim_checked"`
}

func validateBlockAccessibilityEvidence(report Report) []string {
	if report.BlockAccessibilityTree == nil {
		return nil
	}

	tree := report.BlockAccessibilityTree
	var issues []string
	if report.BlockGraph == nil {
		return []string{"block_accessibility_tree requires block_graph evidence"}
	}
	graph := report.BlockGraph
	if !isSurfaceBlockAccessibilitySource(report.Source) {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree source path must match a Block accessibility/system example, got %q", report.Source))
	}
	if tree.Schema != "tetra.surface.block-accessibility-tree.v1" {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree schema is %q, want tetra.surface.block-accessibility-tree.v1", tree.Schema))
	}
	if tree.AccessibilityLevel != "block-metadata-tree-v1" {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree accessibility_level is %q, want block-metadata-tree-v1", tree.AccessibilityLevel))
	}
	if normalizeEvidencePath(tree.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree source %q must match report source %q", tree.Source, report.Source))
	}
	if tree.Module != "lib.core.block" {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree module is %q, want lib.core.block", tree.Module))
	}
	if tree.QualityLevel != "block-derived-accessibility-metadata-v1" {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree quality_level is %q, want block-derived-accessibility-metadata-v1", tree.QualityLevel))
	}
	if tree.BlockGraphSchema != "tetra.surface.block-graph.v1" {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree block_graph_schema is %q, want tetra.surface.block-graph.v1", tree.BlockGraphSchema))
	}
	if normalizeEvidencePath(graph.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree block_graph source %q must match report source %q", graph.Source, report.Source))
	}
	if !tree.DerivedFromBlockGraph {
		issues = append(issues, "block_accessibility_tree must declare derived_from_block_graph=true")
	}
	if tree.ManualBookkeeping {
		issues = append(issues, "block_accessibility_tree manual_bookkeeping must be false")
	}
	if tree.PlatformHostIntegration {
		issues = append(issues, "block_accessibility_tree platform_host_integration must be false for metadata-only Block evidence")
	}
	if tree.DOMARIAIntegration {
		issues = append(issues, "block_accessibility_tree dom_aria_integration must be false")
	}
	if screenReaderEvidenceTruthy(tree.ScreenReaderEvidence) {
		issues = append(issues, "block_accessibility_tree screen_reader_evidence must be false without platform assistive-tech proof")
	}
	if !tree.NoDOMUI || !tree.NoUserJS || !tree.NoPlatformWidgets {
		issues = append(issues, "block_accessibility_tree must prove no_dom_ui, no_user_js, and no_platform_widgets")
	}
	if tree.NodeCount != len(tree.Nodes) {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree node_count = %d, want len(nodes) %d", tree.NodeCount, len(tree.Nodes)))
	}
	if tree.NodeCount == 0 {
		issues = append(issues, "block_accessibility_tree nodes evidence is required")
	}

	graphNodes := map[int]BlockGraphNodeReport{}
	for _, node := range graph.Nodes {
		graphNodes[node.ID] = node
	}
	expectedFocus := graph.FocusOrder
	expectedReading := graph.AccessibilityOrder
	if !intSlicesEqual(tree.FocusOrder, expectedFocus) {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree focus_order = %v, want block_graph focus_order %v", tree.FocusOrder, expectedFocus))
	}
	if !intSlicesEqual(tree.ReadingOrder, expectedReading) {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree reading_order = %v, want block_graph accessibility_order %v", tree.ReadingOrder, expectedReading))
	}

	nodesByName := map[string]BlockAccessibilityNodeReport{}
	nodesByBlockID := map[int]BlockAccessibilityNodeReport{}
	roleCounts := map[string]int{}
	focusableCount := 0
	focusedCount := 0
	for _, node := range tree.Nodes {
		if node.ID != node.BlockID {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node id %d must match block_id %d", node.ID, node.BlockID))
		}
		if strings.TrimSpace(node.Name) == "" {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %d name is required", node.BlockID))
		} else if _, exists := nodesByName[node.Name]; exists {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree duplicate node name %s", node.Name))
		}
		nodesByName[node.Name] = node
		if _, exists := nodesByBlockID[node.BlockID]; exists {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree duplicate block_id %d", node.BlockID))
		}
		nodesByBlockID[node.BlockID] = node
		graphNode, ok := graphNodes[node.BlockID]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s block_id %d is not in block_graph", node.Name, node.BlockID))
		} else {
			if node.ParentBlockID != graphNode.ParentID {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s parent_block_id = %d, want block_graph parent_id %d", node.Name, node.ParentBlockID, graphNode.ParentID))
			}
			if !rectsEqual(node.Bounds, graphNode.Bounds) {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s bounds %+v do not match block_graph bounds %+v", node.Name, node.Bounds, graphNode.Bounds))
			}
			if normalizeAccessibilityRole(node.Role) != normalizeAccessibilityRole(graphNode.AccessibilityRole) {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s role is %q, want block_graph role %q", node.Name, node.Role, graphNode.AccessibilityRole))
			}
			if node.Focusable != graphNode.Focusable {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s focusable = %t, want block_graph focusable %t", node.Name, node.Focusable, graphNode.Focusable))
			}
		}
		role := normalizeAccessibilityRole(node.Role)
		if role == "" || role == "none" {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s role is required", node.Name))
		}
		roleCounts[role]++
		if !containsNormalized(tree.RolesPresent, role) {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree roles_present missing %s", role))
		}
		if node.Bounds.W <= 0 || node.Bounds.H <= 0 {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s bounds are required", node.Name))
		}
		if !node.Visible || !node.Enabled {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s must be visible and enabled", node.Name))
		}
		if node.Focusable {
			focusableCount++
			if node.FocusIndex < 0 {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree focusable node %s requires focus_index", node.Name))
			}
		} else if node.FocusIndex >= 0 {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree non-focusable node %s focus_index = %d, want -1", node.Name, node.FocusIndex))
		}
		if node.Focused {
			focusedCount++
		}
		if node.ReadingIndex < 0 {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s requires reading_index", node.Name))
		}
		if (node.Focusable || len(node.Actions) > 0) && strings.TrimSpace(node.Name) == "" {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree actionable focusable block %d requires accessible name", node.BlockID))
		}
		if node.Focusable && len(node.Actions) == 0 {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree focusable node %s requires actions", node.Name))
		}
	}
	if tree.FocusableCount != focusableCount {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree focusable_count = %d, want %d", tree.FocusableCount, focusableCount))
	}
	if focusedCount > 1 {
		issues = append(issues, fmt.Sprintf("block_accessibility_tree focused node count = %d, want at most 1", focusedCount))
	}
	for _, blockID := range expectedReading {
		if _, ok := nodesByBlockID[blockID]; !ok {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree missing block_graph accessibility node %d", blockID))
		}
	}
	for i, blockID := range expectedFocus {
		node, ok := nodesByBlockID[blockID]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree missing block_graph focus node %d", blockID))
			continue
		}
		if node.FocusIndex != i {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s focus_index = %d, want %d", node.Name, node.FocusIndex, i))
		}
	}
	for i, blockID := range expectedReading {
		node, ok := nodesByBlockID[blockID]
		if !ok {
			continue
		}
		if node.ReadingIndex != i {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree node %s reading_index = %d, want %d", node.Name, node.ReadingIndex, i))
		}
	}
	for role, count := range roleCounts {
		if count > 0 && !containsNormalized(tree.RolesPresent, role) {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree roles_present missing %s", role))
		}
	}
	issues = append(issues, validateBlockAccessibilityRelationships(tree.Relationships, nodesByName)...)
	issues = append(issues, validateBlockAccessibilityActions(tree.Actions, nodesByName)...)
	issues = append(issues, validateBlockAccessibilityNegativeGuards(tree.NegativeGuards)...)
	for _, required := range []string{
		"block accessibility tree derived from block graph",
		"block accessibility focusable actionable name required",
		"block accessibility label relationship mismatch rejected",
		"block accessibility reading order graph mismatch rejected",
		"block accessibility screen-reader claim without platform proof rejected",
		"block accessibility platform claim scoped metadata only",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree report requires %s evidence", required))
		}
	}
	return issues
}

func validateBlockAccessibilityRelationships(relationships []AccessibilityRelationshipReport, nodes map[string]BlockAccessibilityNodeReport) []string {
	if len(relationships) == 0 {
		return []string{"block_accessibility_tree label relationships evidence is required"}
	}
	var issues []string
	for _, relationship := range relationships {
		from, fromOK := nodes[relationship.From]
		to, toOK := nodes[relationship.To]
		if !fromOK {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree relationship from %q is not a node", relationship.From))
			continue
		}
		if !toOK {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree relationship to %q is not a node", relationship.To))
			continue
		}
		switch relationship.Kind {
		case "label_for":
			if from.LabelFor != to.Name || to.LabelledBy != from.Name {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree label relationship mismatch: %s label_for %s must match reciprocal labelled_by", from.Name, to.Name))
			}
		case "labelled_by":
			if from.LabelledBy != to.Name || to.LabelFor != from.Name {
				issues = append(issues, fmt.Sprintf("block_accessibility_tree label relationship mismatch: %s labelled_by %s must match reciprocal label_for", from.Name, to.Name))
			}
		default:
			issues = append(issues, fmt.Sprintf("block_accessibility_tree relationship kind %q is unsupported", relationship.Kind))
		}
	}
	return issues
}

func validateBlockAccessibilityActions(actions []AccessibilityActionReport, nodes map[string]BlockAccessibilityNodeReport) []string {
	if len(actions) == 0 {
		return []string{"block_accessibility_tree actions evidence is required"}
	}
	var issues []string
	for _, action := range actions {
		node, ok := nodes[action.Target]
		if !ok {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree action target %q is not a node", action.Target))
			continue
		}
		if !contains(node.Actions, action.Action) {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree action %s missing from node %s actions", action.Action, action.Target))
		}
		if strings.TrimSpace(action.Semantic) == "" {
			issues = append(issues, fmt.Sprintf("block_accessibility_tree action %s requires semantic", action.Target))
		}
	}
	return issues
}

func validateBlockAccessibilityNegativeGuards(guards BlockAccessibilityNegativeGuardsReport) []string {
	var missing []string
	if !guards.FocusableActionNameChecked {
		missing = append(missing, "focusable_action_name_checked")
	}
	if !guards.LabelRelationshipsChecked {
		missing = append(missing, "label_relationships_checked")
	}
	if !guards.ReadingOrderGraphChecked {
		missing = append(missing, "reading_order_graph_checked")
	}
	if !guards.BoundsAlignmentChecked {
		missing = append(missing, "bounds_alignment_checked")
	}
	if !guards.FakeScreenReaderClaimRejected {
		missing = append(missing, "fake_screen_reader_claim_rejected")
	}
	if !guards.ScopedPlatformClaimChecked {
		missing = append(missing, "scoped_platform_claim_checked")
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("block_accessibility_tree negative_guards missing %s", strings.Join(missing, ", "))}
}

func isSurfaceBlockAccessibilitySource(source string) bool {
	source = normalizeEvidencePath(source)
	return strings.HasSuffix(source, "examples/surface_block_accessibility.tetra") ||
		strings.HasSuffix(source, "examples/surface_block_system.tetra") ||
		strings.HasSuffix(source, "examples/surface_morph_command_palette.tetra") ||
		strings.HasSuffix(source, "examples/surface_morph_rendered_studio_shell.tetra") ||
		isSurfaceReferenceAppSource(source) ||
		strings.HasSuffix(source, "examples/surface_migration_tetra_control_center.tetra") ||
		isSurfaceProjectTemplateSource(source)
}
