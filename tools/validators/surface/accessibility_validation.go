package surface

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type AccessibilityTreeReport struct {
	Schema                     string                            `json:"schema"`
	AccessibilityLevel         string                            `json:"accessibility_level"`
	ReleaseScope               string                            `json:"release_scope,omitempty"`
	Source                     string                            `json:"source"`
	Module                     string                            `json:"module"`
	WidgetModule               string                            `json:"widget_module"`
	Experimental               bool                              `json:"experimental"`
	ProductionClaim            bool                              `json:"production_claim"`
	PlatformHostIntegration    bool                              `json:"platform_host_integration"`
	DOMARIAIntegration         bool                              `json:"dom_aria_integration"`
	ScreenReaderEvidence       any                               `json:"screen_reader_evidence"`
	MetadataTree               bool                              `json:"metadata_tree,omitempty"`
	PlatformExport             bool                              `json:"platform_export,omitempty"`
	PlatformBridge             string                            `json:"platform_bridge,omitempty"`
	LinuxPlatformProbe         bool                              `json:"linux_platform_probe,omitempty"`
	LinuxProbeArtifact         string                            `json:"linux_probe_artifact,omitempty"`
	BrowserAccessibilitySnap   bool                              `json:"browser_accessibility_snapshot,omitempty"`
	BrowserAccessibilityMirror bool                              `json:"browser_accessibility_mirror,omitempty"`
	DerivedFromComponentTree   bool                              `json:"derived_from_component_tree"`
	UsesComponentTreeAPI       bool                              `json:"uses_component_tree_api"`
	UsesWidgetToolkit          bool                              `json:"uses_widget_toolkit"`
	ManualBookkeeping          bool                              `json:"manual_bookkeeping"`
	NoDOMUI                    bool                              `json:"no_dom_ui"`
	NoUserJS                   bool                              `json:"no_user_js"`
	NoPlatformWidgets          bool                              `json:"no_platform_widgets"`
	NoLegacySidecars           bool                              `json:"no_legacy_sidecars"`
	ComponentTreeSchema        string                            `json:"component_tree_schema"`
	ComponentTreeAPISchema     string                            `json:"component_tree_api_schema"`
	ToolkitSchema              string                            `json:"toolkit_schema"`
	NodeCount                  int                               `json:"node_count"`
	FocusableCount             int                               `json:"focusable_count"`
	LabelCount                 int                               `json:"label_count"`
	TextBoxCount               int                               `json:"textbox_count"`
	ButtonCount                int                               `json:"button_count"`
	StatusCount                int                               `json:"status_count"`
	RolesPresent               []string                          `json:"roles_present"`
	Nodes                      []AccessibilityNodeReport         `json:"nodes"`
	Relationships              []AccessibilityRelationshipReport `json:"relationships"`
	FocusOrder                 []string                          `json:"focus_order"`
	ReadingOrder               []string                          `json:"reading_order"`
	Actions                    []AccessibilityActionReport       `json:"actions"`
	Snapshots                  []AccessibilitySnapshotReport     `json:"snapshots"`
	NegativeGuards             AccessibilityNegativeGuardsReport `json:"negative_guards"`
}

type AccessibilityNodeReport struct {
	ID           int        `json:"id"`
	ComponentID  int        `json:"component_id"`
	ParentID     int        `json:"parent_id"`
	Name         string     `json:"name"`
	Role         string     `json:"role"`
	Bounds       RectReport `json:"bounds"`
	Visible      bool       `json:"visible"`
	Enabled      bool       `json:"enabled"`
	Focusable    bool       `json:"focusable"`
	Focused      bool       `json:"focused"`
	Editable     bool       `json:"editable"`
	Readonly     bool       `json:"readonly"`
	Required     bool       `json:"required"`
	Pressed      bool       `json:"pressed"`
	Invalid      bool       `json:"invalid"`
	LabelFor     string     `json:"label_for,omitempty"`
	LabelledBy   string     `json:"labelled_by,omitempty"`
	ValueKind    string     `json:"value_kind,omitempty"`
	ValueLen     int        `json:"value_len,omitempty"`
	Actions      []string   `json:"actions,omitempty"`
	FocusIndex   int        `json:"focus_index"`
	ReadingIndex int        `json:"reading_index"`
}

type AccessibilityRelationshipReport struct {
	Kind string `json:"kind"`
	From string `json:"from"`
	To   string `json:"to"`
}

type AccessibilityActionReport struct {
	Target   string `json:"target"`
	Action   string `json:"action"`
	Semantic string `json:"semantic"`
}

type AccessibilitySnapshotReport struct {
	Name                       string `json:"name"`
	Generation                 int    `json:"generation"`
	Focused                    string `json:"focused"`
	FocusedComponentID         int    `json:"focused_component_id"`
	FocusedAccessibilityNodeID int    `json:"focused_accessibility_node_id"`
	NameValueLen               int    `json:"name_value_len"`
	EmailValueLen              int    `json:"email_value_len"`
	StatusValue                string `json:"status_value"`
	BoundsChecksum             string `json:"bounds_checksum"`
	MetadataChecksum           string `json:"metadata_checksum"`
	FrameChecksum              string `json:"frame_checksum"`
}

type AccessibilityNegativeGuardsReport struct {
	NoBorrowedViewStorage       bool `json:"no_borrowed_view_storage"`
	ComponentIDAlignmentChecked bool `json:"component_id_alignment_checked"`
	BoundsAlignmentChecked      bool `json:"bounds_alignment_checked"`
	FocusOrderAlignmentChecked  bool `json:"focus_order_alignment_checked"`
	ReadingOrderChecked         bool `json:"reading_order_checked"`
	LabelRelationshipsChecked   bool `json:"label_relationships_checked"`
	StateUpdatesChecked         bool `json:"state_updates_checked"`
	ArtifactScanChecked         bool `json:"artifact_scan_checked"`
}

func validateAccessibilityTreeEvidence(report Report) []string {
	if !isAccessibilityMetadataReport(report) {
		return nil
	}
	var issues []string
	releaseWindow := isLinuxReleaseWindowReport(report)
	releaseAccessibility := isSurfaceReleaseAccessibilitySource(report.Source)
	if !isSurfaceAccessibilitySettingsSource(report.Source) && !releaseAccessibility && !releaseWindow {
		issues = append(issues, fmt.Sprintf("accessibility_tree source path must match examples/surface_accessibility_settings.tetra or examples/surface_release_accessibility.tetra, got %q", report.Source))
	}
	if report.Target == "wasm32-web" && !isBrowserCanvasHostEvidenceLevel(report.HostEvidence.Level) {
		issues = append(issues, "accessibility_tree browser evidence must use browser canvas native input, not Node-only wasm32-web evidence")
	}
	if report.AccessibilityTree == nil {
		return append(issues, "accessibility_tree evidence is required for examples/surface_accessibility_settings.tetra")
	}

	tree := report.AccessibilityTree
	if releaseWindow {
		issues = append(issues, validateLinuxReleaseWindowAccessibilityBridgeEvidence(report, tree)...)
		return issues
	}
	if tree.Schema != "tetra.surface.accessibility-tree.v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree schema is %q, want tetra.surface.accessibility-tree.v1", tree.Schema))
	}
	if tree.AccessibilityLevel != "metadata-tree-v1" && tree.AccessibilityLevel != "platform-bridge-v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree accessibility_level is %q, want metadata-tree-v1 or platform-bridge-v1", tree.AccessibilityLevel))
	}
	releaseAccessibility = releaseAccessibility || tree.AccessibilityLevel == "platform-bridge-v1"
	if normalizeEvidencePath(tree.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("accessibility_tree source %q must match report source %q", tree.Source, report.Source))
	}
	if tree.Module != "lib.core.accessibility" {
		issues = append(issues, fmt.Sprintf("accessibility_tree module is %q, want lib.core.accessibility", tree.Module))
	}
	if tree.WidgetModule != "lib.core.widgets" {
		issues = append(issues, fmt.Sprintf("accessibility_tree widget_module is %q, want lib.core.widgets", tree.WidgetModule))
	}
	if releaseAccessibility {
		issues = append(issues, validateReleaseAccessibilityBridgeEvidence(report, tree)...)
	} else {
		if tree.AccessibilityLevel != "metadata-tree-v1" {
			issues = append(issues, fmt.Sprintf("accessibility_tree accessibility_level is %q, want metadata-tree-v1", tree.AccessibilityLevel))
		}
		if !tree.Experimental {
			issues = append(issues, "accessibility_tree must declare experimental=true")
		}
		if tree.ProductionClaim {
			issues = append(issues, "accessibility_tree production_claim must be false")
		}
		if tree.PlatformHostIntegration {
			issues = append(issues, "accessibility_tree platform_host_integration must be false for metadata-only Surface evidence")
		}
		if tree.DOMARIAIntegration {
			issues = append(issues, "accessibility_tree dom_aria_integration must be false")
		}
		if screenReaderEvidenceTruthy(tree.ScreenReaderEvidence) {
			issues = append(issues, "accessibility_tree screen_reader_evidence must be false")
		}
	}
	if !tree.DerivedFromComponentTree || !tree.UsesComponentTreeAPI || !tree.UsesWidgetToolkit {
		issues = append(issues, "accessibility_tree must be derived from component_tree, component_tree_api, and widget toolkit evidence")
	}
	if tree.ManualBookkeeping {
		issues = append(issues, "accessibility_tree manual_bookkeeping must be false")
	}
	if !tree.NoDOMUI || !tree.NoUserJS || !tree.NoPlatformWidgets || !tree.NoLegacySidecars {
		issues = append(issues, "accessibility_tree must prove no_dom_ui, no_user_js, no_platform_widgets, and no_legacy_sidecars")
	}
	if tree.ComponentTreeSchema != "tetra.surface.component-tree.v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree component_tree_schema is %q, want tetra.surface.component-tree.v1", tree.ComponentTreeSchema))
	}
	if tree.ComponentTreeAPISchema != "tetra.surface.component-tree-api.v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree component_tree_api_schema is %q, want tetra.surface.component-tree-api.v1", tree.ComponentTreeAPISchema))
	}
	if tree.ToolkitSchema != "tetra.surface.toolkit.v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree toolkit_schema is %q, want tetra.surface.toolkit.v1", tree.ToolkitSchema))
	}

	issues = append(issues, validateAccessibilityToolkitEvidence(report)...)

	componentNodes := map[int]ComponentTreeNodeReport{}
	if report.ComponentTree == nil {
		issues = append(issues, "accessibility_tree requires component_tree evidence")
	} else {
		wantLevel := "accessibility-metadata-tree-v1"
		if releaseAccessibility {
			wantLevel = "platform-bridge-v1"
		}
		if report.ComponentTree.DynamicLevel != wantLevel {
			issues = append(issues, fmt.Sprintf("accessibility_tree component_tree dynamic_level is %q, want %s", report.ComponentTree.DynamicLevel, wantLevel))
		}
		if !intSlicesEqual(report.ComponentTree.FocusOrder, []int{5, 7, 9, 10}) {
			issues = append(issues, fmt.Sprintf("accessibility_tree component_tree focus_order = %v, want [5 7 9 10]", report.ComponentTree.FocusOrder))
		}
		for _, node := range report.ComponentTree.Nodes {
			componentNodes[node.ID] = node
		}
	}

	if tree.NodeCount != len(tree.Nodes) {
		issues = append(issues, fmt.Sprintf("accessibility_tree node_count = %d, want len(nodes) %d", tree.NodeCount, len(tree.Nodes)))
	}
	if tree.NodeCount != 12 || len(tree.Nodes) != 12 {
		issues = append(issues, fmt.Sprintf("accessibility_tree node_count = %d, want 12", tree.NodeCount))
	}

	allowedRoles := map[string]bool{
		"root": true, "panel": true, "column": true, "text": true, "label": true,
		"textbox": true, "row": true, "button": true, "status": true,
	}
	for role := range allowedRoles {
		if !contains(tree.RolesPresent, role) {
			issues = append(issues, fmt.Sprintf("accessibility_tree roles_present missing %s", role))
		}
	}

	nodesByID := map[int]AccessibilityNodeReport{}
	nodesByName := map[string]AccessibilityNodeReport{}
	componentIDs := map[int]string{}
	roleCounts := map[string]int{}
	focusedCount := 0
	for _, node := range tree.Nodes {
		if _, exists := nodesByID[node.ID]; exists {
			issues = append(issues, fmt.Sprintf("accessibility_tree duplicate node id %d", node.ID))
		}
		nodesByID[node.ID] = node
		if strings.TrimSpace(node.Name) == "" {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %d name is required", node.ID))
		} else if _, exists := nodesByName[node.Name]; exists {
			issues = append(issues, fmt.Sprintf("accessibility_tree duplicate node name %s", node.Name))
		}
		nodesByName[node.Name] = node
		if !allowedRoles[node.Role] {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s has unknown role %q", node.Name, node.Role))
		}
		roleCounts[node.Role]++
		if prev, exists := componentIDs[node.ComponentID]; exists {
			issues = append(issues, fmt.Sprintf("accessibility_tree duplicate component_id %d used by %s and %s", node.ComponentID, prev, node.Name))
		}
		componentIDs[node.ComponentID] = node.Name
		componentNode, ok := componentNodes[node.ComponentID]
		if !ok {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s component_id %d is not in component_tree", node.Name, node.ComponentID))
		} else {
			if componentNode.Name != node.Name {
				issues = append(issues, fmt.Sprintf("accessibility_tree node %s component_id %d names component_tree node %s", node.Name, node.ComponentID, componentNode.Name))
			}
			if componentNode.ParentID != node.ParentID {
				issues = append(issues, fmt.Sprintf("accessibility_tree node %s parent_id = %d, want component_tree parent_id %d", node.Name, node.ParentID, componentNode.ParentID))
			}
			if !rectsEqual(node.Bounds, componentNode.Bounds) {
				issues = append(issues, fmt.Sprintf("accessibility_tree node %s bounds %+v do not match component_tree bounds %+v", node.Name, node.Bounds, componentNode.Bounds))
			}
		}
		if node.Bounds.W <= 0 || node.Bounds.H <= 0 {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s bounds are required", node.Name))
		}
		if !node.Visible || !node.Enabled {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s must be visible and enabled", node.Name))
		}
		if node.Focused {
			focusedCount++
		}
		if node.Role == "textbox" && !node.Editable {
			issues = append(issues, fmt.Sprintf("accessibility_tree textbox %s must be editable", node.Name))
		}
		if node.Role != "textbox" && node.Editable {
			issues = append(issues, fmt.Sprintf("accessibility_tree non-textbox node %s must not be editable", node.Name))
		}
		if node.Readonly || node.Required || node.Pressed || node.Invalid {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s unexpected readonly/required/pressed/invalid state", node.Name))
		}
	}
	if focusedCount != 1 {
		issues = append(issues, fmt.Sprintf("accessibility_tree focused node count = %d, want exactly 1", focusedCount))
	}
	for _, count := range []struct {
		name string
		got  int
		want int
	}{
		{name: "focusable_count", got: tree.FocusableCount, want: 4},
		{name: "label_count", got: tree.LabelCount, want: 2},
		{name: "textbox_count", got: tree.TextBoxCount, want: 2},
		{name: "button_count", got: tree.ButtonCount, want: 2},
		{name: "status_count", got: tree.StatusCount, want: 1},
	} {
		if count.got != count.want {
			issues = append(issues, fmt.Sprintf("accessibility_tree %s = %d, want %d", count.name, count.got, count.want))
		}
	}
	if roleCounts["label"] != tree.LabelCount || roleCounts["textbox"] != tree.TextBoxCount || roleCounts["button"] != tree.ButtonCount || roleCounts["status"] != tree.StatusCount {
		issues = append(issues, "accessibility_tree role counts must match node roles")
	}

	for _, expected := range []struct {
		id      int
		parent  int
		name    string
		role    string
		focus   int
		reading int
		actions []string
	}{
		{id: 0, parent: -1, name: "AccessibilitySettingsApp", role: "root", focus: -1, reading: 0},
		{id: 1, parent: 0, name: "Panel", role: "panel", focus: -1, reading: 1},
		{id: 2, parent: 1, name: "Column", role: "column", focus: -1, reading: 2},
		{id: 3, parent: 2, name: "TitleText", role: "text", focus: -1, reading: 3},
		{id: 4, parent: 2, name: "NameLabel", role: "label", focus: -1, reading: 4},
		{id: 5, parent: 2, name: "NameTextBox", role: "textbox", focus: 0, reading: 5, actions: []string{"focus", "edit"}},
		{id: 6, parent: 2, name: "EmailLabel", role: "label", focus: -1, reading: 6},
		{id: 7, parent: 2, name: "EmailTextBox", role: "textbox", focus: 1, reading: 7, actions: []string{"focus", "edit"}},
		{id: 8, parent: 2, name: "ButtonRow", role: "row", focus: -1, reading: 8},
		{id: 9, parent: 8, name: "SaveButton", role: "button", focus: 2, reading: 9, actions: []string{"focus", "press", "save"}},
		{id: 10, parent: 8, name: "ResetButton", role: "button", focus: 3, reading: 10, actions: []string{"focus", "press", "reset"}},
		{id: 11, parent: 2, name: "StatusText", role: "status", focus: -1, reading: 11},
	} {
		node, ok := nodesByID[expected.id]
		if !ok {
			issues = append(issues, fmt.Sprintf("accessibility_tree missing node id %d (%s)", expected.id, expected.name))
			continue
		}
		if node.Name != expected.name || node.Role != expected.role || node.ParentID != expected.parent {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %d = %s/%s parent %d, want %s/%s parent %d", expected.id, node.Name, node.Role, node.ParentID, expected.name, expected.role, expected.parent))
		}
		if node.FocusIndex != expected.focus {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s focus_index = %d, want %d", expected.name, node.FocusIndex, expected.focus))
		}
		if node.ReadingIndex != expected.reading {
			issues = append(issues, fmt.Sprintf("accessibility_tree node %s reading_index = %d, want %d", expected.name, node.ReadingIndex, expected.reading))
		}
		for _, action := range expected.actions {
			if !contains(node.Actions, action) {
				issues = append(issues, fmt.Sprintf("accessibility_tree node %s actions missing %s", expected.name, action))
			}
		}
	}

	for _, relation := range []struct {
		kind string
		from string
		to   string
	}{
		{kind: "label_for", from: "NameLabel", to: "NameTextBox"},
		{kind: "labelled_by", from: "NameTextBox", to: "NameLabel"},
		{kind: "label_for", from: "EmailLabel", to: "EmailTextBox"},
		{kind: "labelled_by", from: "EmailTextBox", to: "EmailLabel"},
	} {
		if !hasAccessibilityRelationship(tree.Relationships, relation.kind, relation.from, relation.to) {
			issues = append(issues, fmt.Sprintf("accessibility_tree relationships missing %s %s %s", relation.from, relation.kind, relation.to))
		}
	}
	if nameBox, ok := nodesByName["NameTextBox"]; ok && nameBox.LabelledBy != "NameLabel" {
		issues = append(issues, fmt.Sprintf("accessibility_tree NameTextBox labelled_by is %q, want NameLabel", nameBox.LabelledBy))
	}
	if emailBox, ok := nodesByName["EmailTextBox"]; ok && emailBox.LabelledBy != "EmailLabel" {
		issues = append(issues, fmt.Sprintf("accessibility_tree EmailTextBox labelled_by is %q, want EmailLabel", emailBox.LabelledBy))
	}
	if nameLabel, ok := nodesByName["NameLabel"]; ok && nameLabel.LabelFor != "NameTextBox" {
		issues = append(issues, fmt.Sprintf("accessibility_tree NameLabel label_for is %q, want NameTextBox", nameLabel.LabelFor))
	}
	if emailLabel, ok := nodesByName["EmailLabel"]; ok && emailLabel.LabelFor != "EmailTextBox" {
		issues = append(issues, fmt.Sprintf("accessibility_tree EmailLabel label_for is %q, want EmailTextBox", emailLabel.LabelFor))
	}

	wantFocusOrder := []string{"NameTextBox", "EmailTextBox", "SaveButton", "ResetButton"}
	if !stringSlicesEqual(tree.FocusOrder, wantFocusOrder) {
		issues = append(issues, fmt.Sprintf("accessibility_tree focus_order = %v, want %v", tree.FocusOrder, wantFocusOrder))
	}
	wantReadingOrder := []string{"TitleText", "NameLabel", "NameTextBox", "EmailLabel", "EmailTextBox", "SaveButton", "ResetButton", "StatusText"}
	if !stringSlicesEqual(tree.ReadingOrder, wantReadingOrder) {
		issues = append(issues, fmt.Sprintf("accessibility_tree reading_order = %v, want %v", tree.ReadingOrder, wantReadingOrder))
	}

	for _, action := range []struct {
		target   string
		action   string
		semantic string
	}{
		{target: "NameTextBox", action: "edit", semantic: "text-input"},
		{target: "EmailTextBox", action: "edit", semantic: "text-input"},
		{target: "SaveButton", action: "press", semantic: "save"},
		{target: "ResetButton", action: "press", semantic: "reset"},
	} {
		if !hasAccessibilityAction(tree.Actions, action.target, action.action, action.semantic) {
			issues = append(issues, fmt.Sprintf("accessibility_tree actions missing %s %s/%s", action.target, action.action, action.semantic))
		}
	}

	issues = append(issues, validateAccessibilitySnapshots(tree.Snapshots)...)
	issues = append(issues, validateAccessibilityNegativeGuards(tree.NegativeGuards)...)

	for _, required := range []string{
		"accessibility metadata tree schema",
		"accessibility metadata roles labels values states",
		"accessibility metadata component tree alignment",
		"accessibility metadata focus order alignment",
		"accessibility metadata reading order",
		"accessibility metadata snapshots update",
		"accessibility metadata no DOM ARIA platform host claim",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("accessibility_tree report requires %s evidence", required))
		}
	}
	if releaseAccessibility {
		for _, required := range []string{
			"accessibility platform bridge v1 schema",
			"linux accessibility host bridge export",
			"accessibility release honest screen reader evidence",
		} {
			if !caseNameContains(report.Cases, required) {
				issues = append(issues, fmt.Sprintf("accessibility_tree release report requires %s evidence", required))
			}
		}
	}
	return issues
}

func validateReleaseAccessibilityBridgeEvidence(report Report, tree *AccessibilityTreeReport) []string {
	var issues []string
	if tree.AccessibilityLevel != "platform-bridge-v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree accessibility_level is %q, want platform-bridge-v1", tree.AccessibilityLevel))
	}
	if tree.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("accessibility_tree release_scope is %q, want surface-v1-linux-web", tree.ReleaseScope))
	}
	if tree.Experimental {
		issues = append(issues, "accessibility_tree experimental must be false for release accessibility reports")
	}
	if !tree.ProductionClaim {
		issues = append(issues, "accessibility_tree production_claim must be true for release accessibility reports")
	}
	if !tree.MetadataTree {
		issues = append(issues, "accessibility_tree metadata_tree must be true for platform-bridge-v1")
	}
	if !tree.PlatformExport {
		issues = append(issues, "accessibility_tree platform_export must be true for platform-bridge-v1")
	}
	screenReaderEvidence, ok := screenReaderEvidenceString(tree.ScreenReaderEvidence)
	if !ok || screenReaderEvidence == "" {
		issues = append(issues, "accessibility_tree screen_reader_evidence must name the exact platform-tree evidence")
	}
	if strings.Contains(screenReaderEvidence, "full") || strings.Contains(screenReaderEvidence, "screen-reader-support") {
		issues = append(issues, "accessibility_tree screen_reader_evidence must not claim full screen-reader support without a real probe")
	}
	switch report.Target {
	case "linux-x64":
		if report.HostEvidence.Level != "linux-x64-real-window" {
			issues = append(issues, "accessibility_tree linux release evidence must use linux-x64 real-window host evidence")
		}
		if !report.HostEvidence.AccessibilityBridge {
			issues = append(issues, "accessibility_tree linux release host_evidence.accessibility_bridge must be true")
		}
		if !tree.PlatformHostIntegration {
			issues = append(issues, "accessibility_tree platform_host_integration must be true for linux platform-bridge-v1")
		}
		if tree.PlatformBridge != "linux_accessibility_host_bridge_v1" {
			issues = append(issues, fmt.Sprintf("accessibility_tree platform_bridge is %q, want linux_accessibility_host_bridge_v1", tree.PlatformBridge))
		}
		if !tree.LinuxPlatformProbe {
			issues = append(issues, "accessibility_tree linux_platform_probe must be true for linux platform-bridge-v1")
		}
		if strings.TrimSpace(tree.LinuxProbeArtifact) == "" {
			issues = append(issues, "accessibility_tree linux_probe_artifact is required for linux platform-bridge-v1")
		}
		if screenReaderEvidence != "linux_accessibility_host_bridge_v1" {
			issues = append(issues, fmt.Sprintf("accessibility_tree screen_reader_evidence is %q, want linux_accessibility_host_bridge_v1", screenReaderEvidence))
		}
		if !hasRuntimeProcessName(report.Processes, "surface linux accessibility platform probe") {
			issues = append(issues, "accessibility_tree linux release requires surface linux accessibility platform probe process evidence")
		}
		if !artifactKindContains(report.Artifacts, "linux-accessibility-platform-probe") {
			issues = append(issues, "accessibility_tree linux release requires linux-accessibility-platform-probe artifact")
		}
	case "wasm32-web":
		if !isBrowserCanvasHostEvidenceLevel(report.HostEvidence.Level) {
			issues = append(issues, "accessibility_tree browser release evidence must be browser-canvas input, not Node-only wasm32-web evidence")
		}
		if !report.HostEvidence.BrowserAccessibilitySnapshot {
			issues = append(issues, "accessibility_tree browser release host_evidence.browser_accessibility_snapshot must be true")
		}
		if !report.HostEvidence.BrowserAccessibilityMirror {
			issues = append(issues, "accessibility_tree browser release host_evidence.browser_accessibility_mirror must be true")
		}
		if tree.PlatformBridge != "browser_accessibility_mirror_v1" {
			issues = append(issues, fmt.Sprintf("accessibility_tree platform_bridge is %q, want browser_accessibility_mirror_v1", tree.PlatformBridge))
		}
		if !tree.BrowserAccessibilitySnap {
			issues = append(issues, "accessibility_tree browser_accessibility_snapshot must be true for browser platform-bridge-v1")
		}
		if !tree.BrowserAccessibilityMirror {
			issues = append(issues, "accessibility_tree browser_accessibility_mirror must be true for browser platform-bridge-v1")
		}
		if screenReaderEvidence != "browser_accessibility_snapshot_v1" {
			issues = append(issues, fmt.Sprintf("accessibility_tree screen_reader_evidence is %q, want browser_accessibility_snapshot_v1", screenReaderEvidence))
		}
		if !hasRuntimeProcessName(report.Processes, "surface wasm32-web browser canvas trace") {
			issues = append(issues, "accessibility_tree browser release requires browser canvas trace process evidence")
		}
		if !artifactKindContains(report.Artifacts, "runner-trace") {
			issues = append(issues, "accessibility_tree browser release requires runner-trace artifact")
		}
	case "headless":
		if tree.PlatformBridge == "" {
			issues = append(issues, "accessibility_tree platform_bridge is required for headless platform-bridge-v1")
		}
		if tree.PlatformHostIntegration || tree.LinuxPlatformProbe || strings.TrimSpace(tree.LinuxProbeArtifact) != "" || tree.BrowserAccessibilitySnap || tree.BrowserAccessibilityMirror {
			issues = append(issues, "accessibility_tree headless platform bridge must not claim linux platform probe or browser accessibility mirror")
		}
	default:
		issues = append(issues, fmt.Sprintf("accessibility_tree release target %q is unsupported", report.Target))
	}
	return issues
}

func validateLinuxReleaseWindowAccessibilityBridgeEvidence(report Report, tree *AccessibilityTreeReport) []string {
	var issues []string
	if tree.Schema != "tetra.surface.accessibility-tree.v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree schema is %q, want tetra.surface.accessibility-tree.v1", tree.Schema))
	}
	if tree.AccessibilityLevel != "platform-bridge-v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree accessibility_level is %q, want platform-bridge-v1", tree.AccessibilityLevel))
	}
	if tree.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("accessibility_tree release_scope is %q, want surface-v1-linux-web", tree.ReleaseScope))
	}
	if normalizeEvidencePath(tree.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("accessibility_tree source %q must match report source %q", tree.Source, report.Source))
	}
	if tree.Module != "lib.core.accessibility" {
		issues = append(issues, fmt.Sprintf("accessibility_tree module is %q, want lib.core.accessibility", tree.Module))
	}
	if tree.WidgetModule != "lib.core.widgets" {
		issues = append(issues, fmt.Sprintf("accessibility_tree widget_module is %q, want lib.core.widgets", tree.WidgetModule))
	}
	if tree.Experimental || !tree.ProductionClaim {
		issues = append(issues, "accessibility_tree linux release window requires experimental=false and production_claim=true")
	}
	if !tree.PlatformHostIntegration || !tree.MetadataTree || !tree.PlatformExport {
		issues = append(issues, "accessibility_tree linux release window requires platform_host_integration, metadata_tree, and platform_export")
	}
	if tree.PlatformBridge != "linux_accessibility_host_bridge_v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree platform_bridge is %q, want linux_accessibility_host_bridge_v1", tree.PlatformBridge))
	}
	if !tree.LinuxPlatformProbe || strings.TrimSpace(tree.LinuxProbeArtifact) == "" {
		issues = append(issues, "accessibility_tree linux release window requires linux_platform_probe and linux_probe_artifact")
	}
	screenReaderEvidence, ok := screenReaderEvidenceString(tree.ScreenReaderEvidence)
	if !ok || screenReaderEvidence != "linux_accessibility_host_bridge_v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree screen_reader_evidence is %q, want linux_accessibility_host_bridge_v1", screenReaderEvidence))
	}
	if tree.DOMARIAIntegration || !tree.NoDOMUI || !tree.NoUserJS || !tree.NoPlatformWidgets || !tree.NoLegacySidecars {
		issues = append(issues, "accessibility_tree linux release window must not claim DOM ARIA, DOM UI, user JS, platform widgets, or legacy sidecars")
	}
	if !tree.DerivedFromComponentTree || !tree.UsesComponentTreeAPI || !tree.UsesWidgetToolkit {
		issues = append(issues, "accessibility_tree linux release window must derive from component_tree, component_tree_api, and toolkit evidence")
	}
	if tree.ComponentTreeSchema != "tetra.surface.component-tree.v1" ||
		tree.ComponentTreeAPISchema != "tetra.surface.component-tree-api.v1" ||
		tree.ToolkitSchema != "tetra.surface.toolkit.v1" {
		issues = append(issues, "accessibility_tree linux release window requires component-tree, component-tree-api, and toolkit schemas")
	}
	if report.ComponentTree == nil || report.ComponentTree.DynamicLevel != "production-widgets-v1" {
		issues = append(issues, "accessibility_tree linux release window requires production-widgets-v1 component_tree evidence")
	}
	if report.Toolkit == nil || report.Toolkit.ToolkitLevel != "production-widgets-v1" {
		issues = append(issues, "accessibility_tree linux release window requires production-widgets-v1 toolkit evidence")
	}
	for _, role := range []string{"root", "textbox", "checkbox", "button", "status"} {
		if !contains(tree.RolesPresent, role) {
			issues = append(issues, fmt.Sprintf("accessibility_tree linux release window roles_present missing %s", role))
		}
	}
	for _, focus := range []string{"NameTextBox", "EmailTextBox", "SubscribeCheckbox", "SaveButton", "ResetButton"} {
		if !contains(tree.FocusOrder, focus) {
			issues = append(issues, fmt.Sprintf("accessibility_tree linux release window focus_order missing %s", focus))
		}
	}
	return issues
}

func validateAccessibilityToolkitEvidence(report Report) []string {
	var issues []string
	if report.Toolkit == nil {
		return []string{"accessibility_tree requires toolkit evidence"}
	}
	toolkit := report.Toolkit
	if toolkit.Schema != "tetra.surface.toolkit.v1" {
		issues = append(issues, fmt.Sprintf("accessibility_tree toolkit schema is %q, want tetra.surface.toolkit.v1", toolkit.Schema))
	}
	if toolkit.ToolkitLevel != "toolkit-reuse-v1" || toolkit.ReuseLevel != "multi-form-widget-reuse-v1" {
		issues = append(issues, "accessibility_tree toolkit must reuse toolkit-reuse-v1 multi-form widgets")
	}
	if normalizeEvidencePath(toolkit.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("accessibility_tree toolkit source %q must match report source %q", toolkit.Source, report.Source))
	}
	if toolkit.Module != "lib.core.widgets" {
		issues = append(issues, fmt.Sprintf("accessibility_tree toolkit module is %q, want lib.core.widgets", toolkit.Module))
	}
	if !toolkit.Experimental || toolkit.ProductionClaim || !toolkit.UsesComponentTreeAPI || toolkit.ManualBookkeeping || toolkit.DemoSpecificWidgetStructs {
		issues = append(issues, "accessibility_tree toolkit must be experimental reusable component-tree evidence without production/manual/demo widget claims")
	}
	if !toolkit.NoMagicWidgets || !toolkit.NoPlatformWidgets || !toolkit.NoDOMUI || !toolkit.NoUserJS {
		issues = append(issues, "accessibility_tree toolkit must prove no magic, platform, DOM, or user-JS widgets")
	}
	if toolkit.ExampleCount < 3 || !contains(toolkit.Sources, "examples/surface_accessibility_settings.tetra") {
		issues = append(issues, "accessibility_tree toolkit example_count must include surface_accessibility_settings")
	}
	if toolkit.TextBoxCount < 2 || toolkit.ButtonCount < 2 || !toolkit.MultiTextBoxEvidence || !toolkit.MultiFormEvidence {
		issues = append(issues, "accessibility_tree toolkit requires multi TextBox and multi Button reuse evidence")
	}

	widgets := map[string]ToolkitWidgetReport{}
	for _, widget := range toolkit.Widgets {
		widgets[widget.Name] = widget
	}
	for _, required := range []struct {
		name     string
		kind     string
		nodeID   int
		role     string
		action   string
		editable bool
	}{
		{name: "Panel", kind: "Panel", nodeID: 1},
		{name: "Column", kind: "Column", nodeID: 2},
		{name: "TitleText", kind: "Text", nodeID: 3, role: "text"},
		{name: "NameLabel", kind: "Text", nodeID: 4, role: "label"},
		{name: "NameTextBox", kind: "TextBox", nodeID: 5, editable: true},
		{name: "EmailLabel", kind: "Text", nodeID: 6, role: "label"},
		{name: "EmailTextBox", kind: "TextBox", nodeID: 7, editable: true},
		{name: "ButtonRow", kind: "Row", nodeID: 8},
		{name: "SaveButton", kind: "Button", nodeID: 9, action: "save"},
		{name: "ResetButton", kind: "Button", nodeID: 10, action: "reset"},
		{name: "StatusText", kind: "Text", nodeID: 11, role: "status"},
	} {
		widget, ok := widgets[required.name]
		if !ok {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit widget evidence missing %s", required.name))
			continue
		}
		if widget.Kind != required.kind || widget.NodeID != required.nodeID {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit widget %s = %s/%d, want %s/%d", required.name, widget.Kind, widget.NodeID, required.kind, required.nodeID))
		}
		if required.role != "" && widget.Role != required.role {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit widget %s role is %q, want %q", required.name, widget.Role, required.role))
		}
		if required.action != "" && widget.Action != required.action {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit widget %s action is %q, want %q", required.name, widget.Action, required.action))
		}
		if required.editable && !widget.Editable {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit widget %s must prove editable=true", required.name))
		}
		if !widget.Reusable || !widget.OrdinaryTetraStruct {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit widget %s must be reusable ordinary Tetra struct evidence", required.name))
		}
	}
	for _, source := range []string{
		"lib/core/widgets.tetra:add_accessible_textbox",
		"lib/core/widgets.tetra:add_accessible_button",
		"lib/core/widgets.tetra:add_accessible_status",
	} {
		if !contains(toolkit.ReusableSources, source) {
			issues = append(issues, fmt.Sprintf("accessibility_tree toolkit reusable_sources missing %s", source))
		}
	}
	return issues
}

func validateAccessibilitySnapshots(snapshots []AccessibilitySnapshotReport) []string {
	var issues []string
	if len(snapshots) == 0 {
		return []string{"accessibility_tree snapshots are required"}
	}
	byName := map[string]AccessibilitySnapshotReport{}
	lastGeneration := 0
	for _, snapshot := range snapshots {
		if strings.TrimSpace(snapshot.Name) == "" {
			issues = append(issues, "accessibility_tree snapshot name is required")
		} else if _, exists := byName[snapshot.Name]; exists {
			issues = append(issues, fmt.Sprintf("accessibility_tree duplicate snapshot %s", snapshot.Name))
		}
		byName[snapshot.Name] = snapshot
		if snapshot.Generation <= lastGeneration {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s generation = %d, want greater than %d", snapshot.Name, snapshot.Generation, lastGeneration))
		}
		lastGeneration = snapshot.Generation
		if !validChecksumLike(snapshot.BoundsChecksum) {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s bounds_checksum is invalid", snapshot.Name))
		}
		if !validChecksumLike(snapshot.MetadataChecksum) {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s metadata_checksum is invalid", snapshot.Name))
		}
		if !validChecksumLike(snapshot.FrameChecksum) {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s frame_checksum is invalid", snapshot.Name))
		}
	}
	for _, expected := range []struct {
		name        string
		focused     string
		componentID int
		nodeID      int
		nameLen     int
		emailLen    int
		status      string
	}{
		{name: "initial", focused: "", componentID: -1, nodeID: -1, nameLen: 0, emailLen: 0, status: "idle"},
		{name: "after_name_focus", focused: "NameTextBox", componentID: 5, nodeID: 5, nameLen: 0, emailLen: 0, status: "idle"},
		{name: "after_name_text", focused: "NameTextBox", componentID: 5, nodeID: 5, nameLen: 3, emailLen: 0, status: "idle"},
		{name: "after_email_focus", focused: "EmailTextBox", componentID: 7, nodeID: 7, nameLen: 3, emailLen: 0, status: "idle"},
		{name: "after_email_text", focused: "EmailTextBox", componentID: 7, nodeID: 7, nameLen: 3, emailLen: 5, status: "idle"},
		{name: "after_save", focused: "SaveButton", componentID: 9, nodeID: 9, nameLen: 3, emailLen: 5, status: "saved"},
		{name: "after_reset", focused: "ResetButton", componentID: 10, nodeID: 10, nameLen: 0, emailLen: 0, status: "reset"},
		{name: "after_resize", focused: "NameTextBox", componentID: 5, nodeID: 5, nameLen: 0, emailLen: 0, status: "reset"},
	} {
		snapshot, ok := byName[expected.name]
		if !ok {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshots missing %s", expected.name))
			continue
		}
		if snapshot.Focused != expected.focused || snapshot.FocusedComponentID != expected.componentID || snapshot.FocusedAccessibilityNodeID != expected.nodeID {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s focused = %s/%d/%d, want %s/%d/%d", expected.name, snapshot.Focused, snapshot.FocusedComponentID, snapshot.FocusedAccessibilityNodeID, expected.focused, expected.componentID, expected.nodeID))
		}
		if snapshot.NameValueLen != expected.nameLen || snapshot.EmailValueLen != expected.emailLen {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s value lengths = %d/%d, want %d/%d", expected.name, snapshot.NameValueLen, snapshot.EmailValueLen, expected.nameLen, expected.emailLen))
		}
		if snapshot.StatusValue != expected.status {
			issues = append(issues, fmt.Sprintf("accessibility_tree snapshot %s status_value = %q, want %q", expected.name, snapshot.StatusValue, expected.status))
		}
	}
	if afterNameText, ok := byName["after_name_text"]; ok {
		if afterNameFocus, ok := byName["after_name_focus"]; ok && afterNameText.MetadataChecksum == afterNameFocus.MetadataChecksum {
			issues = append(issues, "accessibility_tree metadata_checksum must change after_name_text")
		}
	}
	if afterEmailText, ok := byName["after_email_text"]; ok {
		if afterEmailFocus, ok := byName["after_email_focus"]; ok && afterEmailText.MetadataChecksum == afterEmailFocus.MetadataChecksum {
			issues = append(issues, "accessibility_tree metadata_checksum must change after_email_text")
		}
	}
	if afterResize, ok := byName["after_resize"]; ok {
		if afterReset, ok := byName["after_reset"]; ok {
			if afterResize.BoundsChecksum == afterReset.BoundsChecksum {
				issues = append(issues, "accessibility_tree bounds_checksum must change after_resize")
			}
			if afterResize.FrameChecksum == afterReset.FrameChecksum {
				issues = append(issues, "accessibility_tree frame_checksum must change after_resize")
			}
		}
	}
	return issues
}

func validateAccessibilityNegativeGuards(guards AccessibilityNegativeGuardsReport) []string {
	var missing []string
	if !guards.NoBorrowedViewStorage {
		missing = append(missing, "no_borrowed_view_storage")
	}
	if !guards.ComponentIDAlignmentChecked {
		missing = append(missing, "component_id_alignment_checked")
	}
	if !guards.BoundsAlignmentChecked {
		missing = append(missing, "bounds_alignment_checked")
	}
	if !guards.FocusOrderAlignmentChecked {
		missing = append(missing, "focus_order_alignment_checked")
	}
	if !guards.ReadingOrderChecked {
		missing = append(missing, "reading_order_checked")
	}
	if !guards.LabelRelationshipsChecked {
		missing = append(missing, "label_relationships_checked")
	}
	if !guards.StateUpdatesChecked {
		missing = append(missing, "state_updates_checked")
	}
	if !guards.ArtifactScanChecked {
		missing = append(missing, "artifact_scan_checked")
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("accessibility_tree negative_guards missing %s", strings.Join(missing, ", "))}
}

func hasAccessibilityRelationship(relationships []AccessibilityRelationshipReport, kind string, from string, to string) bool {
	for _, relationship := range relationships {
		if relationship.Kind == kind && relationship.From == from && relationship.To == to {
			return true
		}
	}
	return false
}

func hasAccessibilityAction(actions []AccessibilityActionReport, target string, action string, semantic string) bool {
	for _, item := range actions {
		if item.Target == target && item.Action == action && item.Semantic == semantic {
			return true
		}
	}
	return false
}

func rectsEqual(a RectReport, b RectReport) bool {
	return a.X == b.X && a.Y == b.Y && a.W == b.W && a.H == b.H
}

func validChecksumLike(value string) bool {
	value = strings.TrimSpace(value)
	if validSHA256Digest(value) {
		return true
	}
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func normalizeAccessibilityRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

func containsNormalized(values []string, value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, current := range values {
		if strings.ToLower(strings.TrimSpace(current)) == value {
			return true
		}
	}
	return false
}

func isAccessibilityMetadataReport(report Report) bool {
	if isSurfaceAccessibilitySettingsSource(report.Source) || isSurfaceReleaseAccessibilitySource(report.Source) {
		return true
	}
	if isLinuxReleaseWindowReport(report) {
		return true
	}
	if report.AccessibilityTree != nil {
		return true
	}
	return caseNameContains(report.Cases, "accessibility metadata tree") || caseNameContains(report.Cases, "accessibility platform bridge")
}

func isSurfaceAccessibilitySettingsSource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_accessibility_settings.tetra")
}

func isSurfaceReleaseAccessibilitySource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_release_accessibility.tetra")
}

func isPlatformBridgeAccessibilityReport(report Report) bool {
	return report.AccessibilityTree != nil && report.AccessibilityTree.AccessibilityLevel == "platform-bridge-v1"
}

func screenReaderEvidenceTruthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.TrimSpace(typed) != ""
	default:
		return false
	}
}

func screenReaderEvidenceString(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), true
	case bool:
		if typed {
			return "true", true
		}
		return "", false
	default:
		return "", false
	}
}
