package surface

import (
	"fmt"
	"sort"
	"strings"
)

type ComponentTreeReport struct {
	Schema        string                            `json:"schema"`
	DynamicLevel  string                            `json:"dynamic_level"`
	RootID        int                               `json:"root_id"`
	NodeCount     int                               `json:"node_count"`
	FocusedID     int                               `json:"focused_id"`
	Nodes         []ComponentTreeNodeReport         `json:"nodes"`
	LayoutPasses  []ComponentTreeLayoutPassReport   `json:"layout_passes"`
	DrawOrder     []int                             `json:"draw_order"`
	DispatchPaths []ComponentTreeDispatchPathReport `json:"dispatch_paths"`
	FocusOrder    []int                             `json:"focus_order"`
}

type ComponentTreeNodeReport struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Kind       string     `json:"kind"`
	ParentID   int        `json:"parent_id"`
	ChildIndex int        `json:"child_index"`
	FirstChild int        `json:"first_child"`
	ChildCount int        `json:"child_count"`
	Focusable  bool       `json:"focusable"`
	Bounds     RectReport `json:"bounds"`
}

type ComponentTreeLayoutPassReport struct {
	ComponentID int        `json:"component_id"`
	Pass        string     `json:"pass"`
	Bounds      RectReport `json:"bounds"`
	Measured    SizeReport `json:"measured"`
}

type ComponentTreeDispatchPathReport struct {
	Event    string `json:"event"`
	TargetID int    `json:"target_id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Path     []int  `json:"path"`
}

type ComponentTreeAPIReport struct {
	Schema            string                               `json:"schema"`
	APILevel          string                               `json:"api_level"`
	Source            string                               `json:"source"`
	ManualBookkeeping bool                                 `json:"manual_bookkeeping"`
	Builder           ComponentTreeAPIBuilderReport        `json:"builder"`
	Invariants        ComponentTreeAPIInvariantReport      `json:"invariants"`
	LayoutHelpers     []ComponentTreeAPILayoutHelperReport `json:"layout_helpers"`
	FocusHelpers      []ComponentTreeAPIFocusHelperReport  `json:"focus_helpers"`
	HitTests          []ComponentTreeAPIHitTestReport      `json:"hit_tests"`
	DispatchPaths     []ComponentTreeAPIDispatchPathReport `json:"dispatch_paths"`
}

type ComponentTreeAPIBuilderReport struct {
	RootCreatedBy     string `json:"root_created_by"`
	ChildrenCreatedBy string `json:"children_created_by"`
	NodeCount         int    `json:"node_count"`
	Capacity          int    `json:"capacity"`
	OverflowChecked   bool   `json:"overflow_checked"`
}

type ComponentTreeAPIInvariantReport struct {
	TreeValidateRan         bool `json:"tree_validate_ran"`
	TreeValidateStatus      int  `json:"tree_validate_status"`
	ParentChildLinksChecked bool `json:"parent_child_links_checked"`
	ChildIndicesChecked     bool `json:"child_indices_checked"`
	ChildCountChecked       bool `json:"child_count_checked"`
	FirstChildChecked       bool `json:"first_child_checked"`
}

type ComponentTreeAPILayoutHelperReport struct {
	Helper        string `json:"helper"`
	Target        string `json:"target"`
	Pass          string `json:"pass"`
	ChangedBounds bool   `json:"changed_bounds"`
}

type ComponentTreeAPIFocusHelperReport struct {
	Helper string `json:"helper"`
	Before string `json:"before"`
	After  string `json:"after"`
}

type ComponentTreeAPIHitTestReport struct {
	Helper string `json:"helper"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Target string `json:"target"`
	Path   []int  `json:"path"`
}

type ComponentTreeAPIDispatchPathReport struct {
	Helper string `json:"helper"`
	Target string `json:"target"`
	Path   []int  `json:"path"`
}

func validateComponentTreeEvidence(report Report) []string {
	if !isComponentTreeReport(report) {
		return nil
	}
	var issues []string
	accessibility := isAccessibilityMetadataReport(report) && !isLinuxReleaseWindowReport(report)
	releaseAccessibility := isSurfaceReleaseAccessibilitySource(report.Source) || isPlatformBridgeAccessibilityReport(report)
	productionToolkit := isProductionToolkitReport(report)
	minimalToolkit := isMinimalToolkitReport(report)
	toolkitReuse := isToolkitReuseReport(report)
	if accessibility {
		if !isSurfaceAccessibilitySettingsSource(report.Source) && !isSurfaceReleaseAccessibilitySource(report.Source) {
			issues = append(issues, fmt.Sprintf("component_tree accessibility source path must match examples/surface_accessibility_settings.tetra or examples/surface_release_accessibility.tetra, got %q", report.Source))
		}
	} else if productionToolkit {
		if !isSurfaceReleaseFormSource(report.Source) {
			issues = append(issues, fmt.Sprintf("component_tree production toolkit source path must match examples/surface_release_form.tetra, got %q", report.Source))
		}
	} else if toolkitReuse {
		if !isSurfaceToolkitSettingsSource(report.Source) {
			issues = append(issues, fmt.Sprintf("component_tree toolkit reuse source path must match examples/surface_toolkit_settings.tetra, got %q", report.Source))
		}
	} else if minimalToolkit {
		if !isSurfaceToolkitFormSource(report.Source) {
			issues = append(issues, fmt.Sprintf("component_tree toolkit source path must match examples/surface_toolkit_form.tetra, got %q", report.Source))
		}
	} else if !isSurfaceTreeAppSource(report.Source) {
		issues = append(issues, fmt.Sprintf("component_tree source path must match examples/surface_tree_app.tetra, got %q", report.Source))
	}
	if report.ComponentTree == nil {
		if accessibility {
			return append(issues, "component_tree evidence is required for examples/surface_accessibility_settings.tetra")
		}
		if productionToolkit {
			return append(issues, "component_tree evidence is required for examples/surface_release_form.tetra")
		}
		if minimalToolkit {
			return append(issues, "component_tree evidence is required for examples/surface_toolkit_form.tetra")
		}
		return append(issues, "component_tree evidence is required for examples/surface_tree_app.tetra")
	}

	tree := report.ComponentTree
	if tree.Schema != "tetra.surface.component-tree.v1" {
		issues = append(issues, fmt.Sprintf("component_tree schema is %q, want tetra.surface.component-tree.v1", tree.Schema))
	}
	if accessibility {
		wantLevel := "accessibility-metadata-tree-v1"
		if releaseAccessibility {
			wantLevel = "platform-bridge-v1"
		}
		if tree.DynamicLevel != wantLevel {
			issues = append(issues, fmt.Sprintf("component_tree dynamic_level is %q, want %s", tree.DynamicLevel, wantLevel))
		}
	} else if productionToolkit {
		if tree.DynamicLevel != "production-widgets-v1" {
			issues = append(issues, fmt.Sprintf("component_tree dynamic_level is %q, want production-widgets-v1", tree.DynamicLevel))
		}
	} else if toolkitReuse {
		if tree.DynamicLevel != "toolkit-reuse-widget-tree" {
			issues = append(issues, fmt.Sprintf("component_tree dynamic_level is %q, want toolkit-reuse-widget-tree", tree.DynamicLevel))
		}
	} else if minimalToolkit {
		if tree.DynamicLevel != "minimal-toolkit-widget-tree" {
			issues = append(issues, fmt.Sprintf("component_tree dynamic_level is %q, want minimal-toolkit-widget-tree", tree.DynamicLevel))
		}
	} else if tree.DynamicLevel != "semi-dynamic-child-list" {
		issues = append(issues, fmt.Sprintf("component_tree dynamic_level is %q, want semi-dynamic-child-list", tree.DynamicLevel))
	}
	if tree.NodeCount != len(tree.Nodes) {
		issues = append(issues, fmt.Sprintf("component_tree node_count = %d, want len(nodes) %d", tree.NodeCount, len(tree.Nodes)))
	}
	if tree.NodeCount < 7 || len(tree.Nodes) < 7 {
		issues = append(issues, fmt.Sprintf("component_tree node_count = %d, want at least 7", tree.NodeCount))
	}

	nodes := map[int]ComponentTreeNodeReport{}
	childrenByParent := map[int][]ComponentTreeNodeReport{}
	focusableCount := 0
	for _, node := range tree.Nodes {
		if _, exists := nodes[node.ID]; exists {
			issues = append(issues, fmt.Sprintf("component_tree duplicate node id %d", node.ID))
		}
		nodes[node.ID] = node
		if strings.TrimSpace(node.Name) == "" {
			issues = append(issues, fmt.Sprintf("component_tree node %d name is required", node.ID))
		}
		if strings.TrimSpace(node.Kind) == "" {
			issues = append(issues, fmt.Sprintf("component_tree node %d kind is required", node.ID))
		}
		if node.Bounds.W <= 0 || node.Bounds.H <= 0 {
			issues = append(issues, fmt.Sprintf("component_tree node %d layout bounds are required", node.ID))
		}
		if node.Focusable {
			focusableCount++
		}
		if node.ParentID >= 0 {
			childrenByParent[node.ParentID] = append(childrenByParent[node.ParentID], node)
		}
		if node.ChildCount == 0 && node.FirstChild != -1 {
			issues = append(issues, fmt.Sprintf("component_tree leaf node %d first_child = %d, want -1", node.ID, node.FirstChild))
		}
		if node.ChildCount < 0 {
			issues = append(issues, fmt.Sprintf("component_tree node %d child_count must be non-negative", node.ID))
		}
	}

	root, ok := nodes[tree.RootID]
	if !ok {
		issues = append(issues, fmt.Sprintf("component_tree root_id %d is not in nodes", tree.RootID))
	} else {
		if root.ParentID != -1 {
			issues = append(issues, fmt.Sprintf("component_tree root %d parent_id = %d, want -1", root.ID, root.ParentID))
		}
		if root.ChildCount < 1 {
			issues = append(issues, "component_tree root must have at least one child")
		}
	}
	if _, ok := nodes[tree.FocusedID]; !ok {
		issues = append(issues, fmt.Sprintf("component_tree focused_id %d is not in nodes", tree.FocusedID))
	}

	for _, node := range tree.Nodes {
		if node.ParentID < 0 {
			continue
		}
		parent, ok := nodes[node.ParentID]
		if !ok {
			issues = append(issues, fmt.Sprintf("component_tree node %d parent_id %d is unknown", node.ID, node.ParentID))
			continue
		}
		if !rectContainsRect(parent.Bounds, node.Bounds) {
			issues = append(issues, fmt.Sprintf("component_tree node %d bounds must be inside parent %d bounds", node.ID, parent.ID))
		}
	}
	for parentID, children := range childrenByParent {
		parent := nodes[parentID]
		if parent.ChildCount != len(children) {
			issues = append(issues, fmt.Sprintf("component_tree node %d child_count = %d, want %d", parentID, parent.ChildCount, len(children)))
		}
		seenChildIndex := map[int]int{}
		firstID := -1
		for _, child := range children {
			if child.ChildIndex < 0 || child.ChildIndex >= len(children) {
				issues = append(issues, fmt.Sprintf("component_tree child node %d child_index = %d, want 0..%d", child.ID, child.ChildIndex, len(children)-1))
			}
			if prev, exists := seenChildIndex[child.ChildIndex]; exists {
				issues = append(issues, fmt.Sprintf("component_tree sibling child_index %d is used by nodes %d and %d", child.ChildIndex, prev, child.ID))
			}
			seenChildIndex[child.ChildIndex] = child.ID
			if child.ChildIndex == 0 {
				firstID = child.ID
			}
		}
		if len(children) > 0 && parent.FirstChild != firstID {
			issues = append(issues, fmt.Sprintf("component_tree node %d first_child = %d, want child_index 0 node %d", parentID, parent.FirstChild, firstID))
		}
	}
	issues = append(issues, validateComponentTreeSiblingLayout(nodes, childrenByParent)...)

	column, hasColumn := componentTreeNodeByKind(tree.Nodes, "column")
	row, hasRow := componentTreeNodeByKind(tree.Nodes, "row")
	textBoxName := "TextBox"
	secondTextBoxName := ""
	submitName := "SubmitButton"
	if accessibility || productionToolkit || toolkitReuse {
		textBoxName = "NameTextBox"
		secondTextBoxName = "EmailTextBox"
		submitName = "SaveButton"
	}
	textBox, hasTextBox := componentTreeNodeByName(tree.Nodes, textBoxName)
	secondTextBox, hasSecondTextBox := componentTreeNodeByName(tree.Nodes, secondTextBoxName)
	checkbox, hasCheckbox := componentTreeNodeByName(tree.Nodes, "SubscribeCheckbox")
	submit, hasSubmit := componentTreeNodeByName(tree.Nodes, submitName)
	reset, hasReset := componentTreeNodeByName(tree.Nodes, "ResetButton")
	if !hasColumn || column.ChildCount < 3 {
		issues = append(issues, "component_tree requires a Column node with at least 3 children")
	}
	if !hasRow || row.ChildCount < 2 {
		issues = append(issues, "component_tree requires a Row node with at least 2 children")
	}
	if !hasTextBox {
		issues = append(issues, fmt.Sprintf("component_tree requires %s node", textBoxName))
	}
	if productionToolkit && !hasCheckbox {
		issues = append(issues, "component_tree requires SubscribeCheckbox node for production toolkit")
	}
	if (accessibility || productionToolkit || toolkitReuse) && !hasSecondTextBox {
		if accessibility {
			issues = append(issues, "component_tree requires EmailTextBox node for accessibility metadata")
		} else if productionToolkit {
			issues = append(issues, "component_tree requires EmailTextBox node for production toolkit")
		} else {
			issues = append(issues, "component_tree requires EmailTextBox node for toolkit reuse")
		}
	}
	if !hasSubmit {
		issues = append(issues, fmt.Sprintf("component_tree requires %s node", submitName))
	}
	if !hasReset {
		issues = append(issues, "component_tree requires ResetButton node")
	}
	if focusableCount < 3 {
		issues = append(issues, fmt.Sprintf("component_tree focusable node count = %d, want at least 3", focusableCount))
	}

	if !componentTreeDrawOrderCoversNodes(tree.DrawOrder, nodes) {
		issues = append(issues, "component_tree draw_order must include every node exactly once")
	}
	if hasTextBox && !containsInt(tree.FocusOrder, textBox.ID) {
		issues = append(issues, fmt.Sprintf("component_tree focus_order missing %s", textBoxName))
	}
	if (accessibility || productionToolkit || toolkitReuse) && hasSecondTextBox && !containsInt(tree.FocusOrder, secondTextBox.ID) {
		issues = append(issues, "component_tree focus_order missing EmailTextBox")
	}
	if hasSubmit && !containsInt(tree.FocusOrder, submit.ID) {
		issues = append(issues, fmt.Sprintf("component_tree focus_order missing %s", submitName))
	}
	if hasReset && !containsInt(tree.FocusOrder, reset.ID) {
		issues = append(issues, "component_tree focus_order missing ResetButton")
	}
	if hasTextBox && hasSubmit && hasReset {
		wantFocusOrder := []int{textBox.ID, submit.ID, reset.ID}
		if (accessibility || productionToolkit || toolkitReuse) && hasSecondTextBox {
			wantFocusOrder = []int{textBox.ID, secondTextBox.ID, submit.ID, reset.ID}
		}
		if productionToolkit && hasSecondTextBox && hasCheckbox {
			wantFocusOrder = []int{textBox.ID, secondTextBox.ID, checkbox.ID, submit.ID, reset.ID}
		}
		if !intSlicesEqual(tree.FocusOrder, wantFocusOrder) {
			if productionToolkit {
				issues = append(issues, fmt.Sprintf("component_tree focus_order = %v, want NameTextBox -> EmailTextBox -> SubscribeCheckbox -> SaveButton -> ResetButton (%v)", tree.FocusOrder, wantFocusOrder))
			} else if accessibility || toolkitReuse {
				issues = append(issues, fmt.Sprintf("component_tree focus_order = %v, want NameTextBox -> EmailTextBox -> SaveButton -> ResetButton (%v)", tree.FocusOrder, wantFocusOrder))
			} else {
				issues = append(issues, fmt.Sprintf("component_tree focus_order = %v, want TextBox -> SubmitButton -> ResetButton (%v)", tree.FocusOrder, wantFocusOrder))
			}
		}
	}

	if len(tree.LayoutPasses) == 0 {
		issues = append(issues, "component_tree layout_passes evidence is required")
	}
	if hasTextBox && !componentTreeHasResizeLayoutPass(tree.LayoutPasses, textBox.ID) {
		issues = append(issues, "component_tree layout_passes require TextBox initial and resize bounds")
	}
	if (accessibility || productionToolkit || toolkitReuse) && hasSecondTextBox && !componentTreeHasResizeLayoutPass(tree.LayoutPasses, secondTextBox.ID) {
		issues = append(issues, "component_tree layout_passes require EmailTextBox initial and resize bounds")
	}

	expectedPaths := map[int][]int{}
	if hasTextBox {
		if path, ok := componentTreePathToRoot(textBox.ID, nodes); ok {
			expectedPaths[textBox.ID] = path
		}
	}
	if hasSubmit {
		if path, ok := componentTreePathToRoot(submit.ID, nodes); ok {
			expectedPaths[submit.ID] = path
		}
	}
	if (accessibility || productionToolkit || toolkitReuse) && hasSecondTextBox {
		if path, ok := componentTreePathToRoot(secondTextBox.ID, nodes); ok {
			expectedPaths[secondTextBox.ID] = path
		}
	}
	if productionToolkit && hasCheckbox {
		if path, ok := componentTreePathToRoot(checkbox.ID, nodes); ok {
			expectedPaths[checkbox.ID] = path
		}
	}
	if hasReset {
		if path, ok := componentTreePathToRoot(reset.ID, nodes); ok {
			expectedPaths[reset.ID] = path
		}
	}
	issues = append(issues, validateComponentTreeDispatchPaths(tree.DispatchPaths, expectedPaths, nodes)...)
	issues = append(issues, validateComponentTreeAPIEvidence(report, tree, expectedPaths, nodes)...)

	for _, required := range []string{
		"component tree node count",
		"component tree parent child links",
		"component tree layout bounds",
		"component tree draw traversal",
		"component tree pointer dispatch path",
		"component tree focus traversal",
		"component tree text routed to focused TextBox",
		"component tree button action dispatch",
		"component tree resize relayout",
		"component tree rendered frame update",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("component_tree report requires %s evidence", required))
		}
	}

	if hasTextBox && hasSubmit && hasReset {
		if productionToolkit && hasSecondTextBox && hasCheckbox {
			if !hasComponentTreeTabFocus(report.Events, fmt.Sprint(textBox.ID), fmt.Sprint(secondTextBox.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(secondTextBox.ID), fmt.Sprint(checkbox.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(checkbox.ID), fmt.Sprint(submit.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(submit.ID), fmt.Sprint(reset.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(reset.ID), fmt.Sprint(textBox.ID)) {
				issues = append(issues, "component_tree requires Tab focus traversal NameTextBox -> EmailTextBox -> SubscribeCheckbox -> SaveButton -> ResetButton -> NameTextBox")
			}
		} else if (accessibility || toolkitReuse) && hasSecondTextBox {
			if !hasComponentTreeTabFocus(report.Events, fmt.Sprint(textBox.ID), fmt.Sprint(secondTextBox.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(secondTextBox.ID), fmt.Sprint(submit.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(submit.ID), fmt.Sprint(reset.ID)) ||
				!hasComponentTreeTabFocus(report.Events, fmt.Sprint(reset.ID), fmt.Sprint(textBox.ID)) {
				issues = append(issues, "component_tree requires Tab focus traversal NameTextBox -> EmailTextBox -> SaveButton -> ResetButton -> NameTextBox")
			}
		} else if !hasComponentTreeTabFocus(report.Events, fmt.Sprint(textBox.ID), fmt.Sprint(submit.ID)) ||
			!hasComponentTreeTabFocus(report.Events, fmt.Sprint(submit.ID), fmt.Sprint(reset.ID)) ||
			!hasComponentTreeTabFocus(report.Events, fmt.Sprint(reset.ID), fmt.Sprint(textBox.ID)) {
			issues = append(issues, "component_tree requires Tab focus traversal TextBox -> SubmitButton -> ResetButton -> TextBox including ResetButton -> TextBox wrap")
		}
	}
	if !hasComponentTreeTextInsertion(report.Events) {
		issues = append(issues, "component_tree requires text input routed to focused TextBox")
	}
	if componentTreeTextMutatedWhileButtonFocused(report.Events) {
		issues = append(issues, "component_tree unfocused TextBox mutated while Button focused")
	}
	if !hasComponentTreeButtonAction(report.Events) {
		issues = append(issues, "component_tree requires keyboard button action dispatch through tree path")
	}
	if !hasComponentTreeResizeRelayout(report.Events, report.StateTransitions) {
		issues = append(issues, "component_tree resize relayout requires changed TextBox bounds while preserving focused_id")
	}
	if len(report.Frames) >= 2 && report.Frames[0].Checksum == report.Frames[len(report.Frames)-1].Checksum {
		issues = append(issues, "component_tree rendered frame update requires changed frame checksum")
	}
	return issues
}

func validateComponentTreeSiblingLayout(nodes map[int]ComponentTreeNodeReport, childrenByParent map[int][]ComponentTreeNodeReport) []string {
	var issues []string
	for parentID, children := range childrenByParent {
		parent := nodes[parentID]
		kind := strings.ToLower(strings.TrimSpace(parent.Kind))
		if kind != "column" && kind != "row" {
			continue
		}
		ordered := append([]ComponentTreeNodeReport(nil), children...)
		sort.SliceStable(ordered, func(i int, j int) bool {
			return ordered[i].ChildIndex < ordered[j].ChildIndex
		})
		for i := 1; i < len(ordered); i++ {
			prev := ordered[i-1]
			child := ordered[i]
			switch kind {
			case "column":
				if child.Bounds.Y < prev.Bounds.Y+prev.Bounds.H {
					issues = append(issues, fmt.Sprintf("component_tree Column node %d child_index %d node %d overlaps or precedes child_index %d node %d", parentID, child.ChildIndex, child.ID, prev.ChildIndex, prev.ID))
				}
			case "row":
				if child.Bounds.X < prev.Bounds.X+prev.Bounds.W {
					issues = append(issues, fmt.Sprintf("component_tree Row node %d child_index %d node %d overlaps child_index %d node %d", parentID, child.ChildIndex, child.ID, prev.ChildIndex, prev.ID))
				}
			}
		}
	}
	return issues
}

func validateComponentTreeAPIEvidence(report Report, tree *ComponentTreeReport, expectedPaths map[int][]int, nodes map[int]ComponentTreeNodeReport) []string {
	if tree == nil {
		return nil
	}
	api := report.ComponentTreeAPI
	if api == nil {
		return []string{"component_tree_api evidence is required for component tree API hardening reports"}
	}
	var issues []string
	accessibility := isAccessibilityMetadataReport(report)
	if api.Schema != "tetra.surface.component-tree-api.v1" {
		issues = append(issues, fmt.Sprintf("component_tree_api schema is %q, want tetra.surface.component-tree-api.v1", api.Schema))
	}
	if api.APILevel != "builder-layout-dispatch-v1" {
		issues = append(issues, fmt.Sprintf("component_tree_api api_level is %q, want builder-layout-dispatch-v1", api.APILevel))
	}
	if normalizeEvidencePath(api.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("component_tree_api source %q must match report source %q", api.Source, report.Source))
	}
	if api.ManualBookkeeping {
		issues = append(issues, "component_tree_api manual_bookkeeping must be false")
	}
	if api.Builder.RootCreatedBy != "tree_add_root" {
		issues = append(issues, fmt.Sprintf("component_tree_api builder root_created_by is %q, want tree_add_root", api.Builder.RootCreatedBy))
	}
	if api.Builder.ChildrenCreatedBy != "tree_add_child" {
		issues = append(issues, fmt.Sprintf("component_tree_api builder children_created_by is %q, want tree_add_child", api.Builder.ChildrenCreatedBy))
	}
	if api.Builder.NodeCount != tree.NodeCount {
		issues = append(issues, fmt.Sprintf("component_tree_api builder node_count = %d, want component_tree node_count %d", api.Builder.NodeCount, tree.NodeCount))
	}
	if api.Builder.Capacity < tree.NodeCount {
		issues = append(issues, fmt.Sprintf("component_tree_api builder capacity = %d, want at least node_count %d", api.Builder.Capacity, tree.NodeCount))
	}
	if !api.Builder.OverflowChecked {
		issues = append(issues, "component_tree_api builder must prove overflow_checked")
	}
	if !api.Invariants.TreeValidateRan {
		issues = append(issues, "component_tree_api invariants require tree_validate_ran")
	}
	if api.Invariants.TreeValidateStatus != 0 {
		issues = append(issues, fmt.Sprintf("component_tree_api tree_validate_status = %d, want 0", api.Invariants.TreeValidateStatus))
	}
	if !api.Invariants.ParentChildLinksChecked || !api.Invariants.ChildIndicesChecked || !api.Invariants.ChildCountChecked || !api.Invariants.FirstChildChecked {
		issues = append(issues, "component_tree_api invariants must check parent/child links, child indices, child_count, and first_child")
	}
	if !hasComponentTreeAPILayout(api.LayoutHelpers, []string{"tree_layout_column", "widgets.column_layout"}, "Column") {
		issues = append(issues, "component_tree_api layout_helpers require changed tree_layout_column or widgets.column_layout evidence for Column")
	}
	if !hasComponentTreeAPILayout(api.LayoutHelpers, []string{"tree_layout_row", "widgets.row_layout"}, "ButtonRow") {
		issues = append(issues, "component_tree_api layout_helpers require changed tree_layout_row or widgets.row_layout evidence for ButtonRow")
	}
	focusPairs := [][2]string{
		{"TextBox", "SubmitButton"},
		{"SubmitButton", "ResetButton"},
		{"ResetButton", "TextBox"},
	}
	dispatchTargets := []string{"TextBox", "SubmitButton", "ResetButton"}
	if isProductionToolkitReport(report) {
		focusPairs = [][2]string{
			{"NameTextBox", "EmailTextBox"},
			{"EmailTextBox", "SubscribeCheckbox"},
			{"SubscribeCheckbox", "SaveButton"},
			{"SaveButton", "ResetButton"},
			{"ResetButton", "NameTextBox"},
		}
		dispatchTargets = []string{"NameTextBox", "EmailTextBox", "SubscribeCheckbox", "SaveButton", "ResetButton"}
	} else if accessibility || isToolkitReuseReport(report) {
		focusPairs = [][2]string{
			{"NameTextBox", "EmailTextBox"},
			{"EmailTextBox", "SaveButton"},
			{"SaveButton", "ResetButton"},
			{"ResetButton", "NameTextBox"},
		}
		dispatchTargets = []string{"NameTextBox", "EmailTextBox", "SaveButton", "ResetButton"}
	}
	for _, pair := range focusPairs {
		if !hasComponentTreeAPIFocusTransition(api.FocusHelpers, pair[0], pair[1]) {
			issues = append(issues, fmt.Sprintf("component_tree_api focus_helpers require %s -> %s", pair[0], pair[1]))
		}
	}
	primaryTextBox := dispatchTargets[0]
	if !hasComponentTreeAPIHitTest(api.HitTests, primaryTextBox, expectedPathForComponentTreeTarget(nodes, expectedPaths, primaryTextBox)) {
		issues = append(issues, fmt.Sprintf("component_tree_api hit_tests require %s path evidence", primaryTextBox))
	}
	if (accessibility || isProductionToolkitReport(report) || isToolkitReuseReport(report)) && !hasComponentTreeAPIHitTest(api.HitTests, "EmailTextBox", expectedPathForComponentTreeTarget(nodes, expectedPaths, "EmailTextBox")) {
		issues = append(issues, "component_tree_api hit_tests require EmailTextBox path evidence")
	}
	if isProductionToolkitReport(report) && !hasComponentTreeAPIHitTest(api.HitTests, "SubscribeCheckbox", expectedPathForComponentTreeTarget(nodes, expectedPaths, "SubscribeCheckbox")) {
		issues = append(issues, "component_tree_api hit_tests require SubscribeCheckbox path evidence")
	}
	if !hasComponentTreeAPIHitTest(api.HitTests, "ResetButton", expectedPathForComponentTreeTarget(nodes, expectedPaths, "ResetButton")) &&
		!hasComponentTreeAPIHitTest(api.HitTests, "SubmitButton", expectedPathForComponentTreeTarget(nodes, expectedPaths, "SubmitButton")) &&
		!hasComponentTreeAPIHitTest(api.HitTests, "SaveButton", expectedPathForComponentTreeTarget(nodes, expectedPaths, "SaveButton")) {
		issues = append(issues, "component_tree_api hit_tests require Button path evidence")
	}
	for _, target := range dispatchTargets {
		wantPath := expectedPathForComponentTreeTarget(nodes, expectedPaths, target)
		if len(wantPath) == 0 {
			continue
		}
		if !hasComponentTreeAPIDispatchPath(api.DispatchPaths, target, wantPath) {
			issues = append(issues, fmt.Sprintf("component_tree_api dispatch_paths require tree_build_dispatch_path %s path %v", target, wantPath))
		}
	}
	for _, required := range []string{
		"component tree api builder node creation",
		"component tree api parent child invariants",
		"component tree api layout helper dispatch",
		"component tree api hit test helper",
		"component tree api focus helper traversal",
		"component tree api dispatch path helper",
		"component tree api no manual bookkeeping",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("component_tree_api report requires %s evidence", required))
		}
	}
	return issues
}

func hasComponentTreeAPILayout(layouts []ComponentTreeAPILayoutHelperReport, helpers []string, target string) bool {
	for _, layout := range layouts {
		if layout.Target != target || !layout.ChangedBounds {
			continue
		}
		for _, helper := range helpers {
			if layout.Helper == helper {
				return true
			}
		}
	}
	return false
}

func hasComponentTreeAPIFocusTransition(focus []ComponentTreeAPIFocusHelperReport, before string, after string) bool {
	for _, item := range focus {
		if item.Helper == "tree_focus_next" && item.Before == before && item.After == after {
			return true
		}
	}
	return false
}

func hasComponentTreeAPIHitTest(hits []ComponentTreeAPIHitTestReport, target string, wantPath []int) bool {
	if len(wantPath) == 0 {
		return false
	}
	for _, hit := range hits {
		if (hit.Helper == "tree_hit_test" || hit.Helper == "widgets.hit_test" || strings.HasPrefix(hit.Helper, "widgets.hit_test_")) && hit.Target == target && intSlicesEqual(hit.Path, wantPath) {
			return true
		}
	}
	return false
}

func hasComponentTreeAPIDispatchPath(paths []ComponentTreeAPIDispatchPathReport, target string, wantPath []int) bool {
	for _, path := range paths {
		if path.Helper == "tree_build_dispatch_path" && path.Target == target && intSlicesEqual(path.Path, wantPath) {
			return true
		}
	}
	return false
}

func expectedPathForComponentTreeTarget(nodes map[int]ComponentTreeNodeReport, expectedPaths map[int][]int, target string) []int {
	for id, node := range nodes {
		if node.Name == target {
			return expectedPaths[id]
		}
	}
	return nil
}

func isComponentTreeReport(report Report) bool {
	if isAccessibilityMetadataReport(report) {
		return true
	}
	if isProductionToolkitReport(report) {
		return true
	}
	if isSurfaceTreeAppSource(report.Source) {
		return true
	}
	if isMinimalToolkitReport(report) {
		return true
	}
	if caseNameContains(report.Cases, "component tree") {
		return true
	}
	return report.ComponentTree != nil
}

func isSurfaceTreeAppSource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_tree_app.tetra")
}

func componentTreeNodeByName(nodes []ComponentTreeNodeReport, name string) (ComponentTreeNodeReport, bool) {
	for _, node := range nodes {
		if node.Name == name {
			return node, true
		}
	}
	return ComponentTreeNodeReport{}, false
}

func componentTreeNodeByKind(nodes []ComponentTreeNodeReport, kind string) (ComponentTreeNodeReport, bool) {
	for _, node := range nodes {
		if strings.EqualFold(node.Kind, kind) {
			return node, true
		}
	}
	return ComponentTreeNodeReport{}, false
}

func componentTreeDrawOrderCoversNodes(drawOrder []int, nodes map[int]ComponentTreeNodeReport) bool {
	if len(drawOrder) != len(nodes) {
		return false
	}
	seen := map[int]bool{}
	for _, id := range drawOrder {
		if _, ok := nodes[id]; !ok || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

func componentTreeHasResizeLayoutPass(passes []ComponentTreeLayoutPassReport, id int) bool {
	initialWidth := -1
	resizeWidth := -1
	for _, pass := range passes {
		if pass.ComponentID != id || pass.Bounds.W <= 0 || pass.Bounds.H <= 0 || pass.Measured.W <= 0 || pass.Measured.H <= 0 {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(pass.Pass)) {
		case "initial":
			initialWidth = pass.Bounds.W
		case "resize":
			resizeWidth = pass.Bounds.W
		}
	}
	return initialWidth > 0 && resizeWidth > 0 && initialWidth != resizeWidth
}

func validateComponentTreeDispatchPaths(paths []ComponentTreeDispatchPathReport, expected map[int][]int, nodes map[int]ComponentTreeNodeReport) []string {
	var issues []string
	if len(paths) == 0 {
		return []string{"component_tree dispatch paths are required"}
	}
	uniqueLeafTargets := map[int]bool{}
	for _, path := range paths {
		target, ok := nodes[path.TargetID]
		if !ok {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path target_id %d is unknown", path.TargetID))
			continue
		}
		if target.ChildCount != 0 {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path target_id %d must be a leaf", path.TargetID))
		}
		if strings.TrimSpace(path.Event) == "" {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path target_id %d event is required", path.TargetID))
		}
		for _, id := range path.Path {
			if _, ok := nodes[id]; !ok {
				issues = append(issues, fmt.Sprintf("component_tree dispatch path for target_id %d contains unknown node id %d", path.TargetID, id))
			}
		}
		parentPath, ok := componentTreePathToRoot(path.TargetID, nodes)
		if !ok {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path cannot resolve parent chain for target_id %d", path.TargetID))
			continue
		}
		if !intSlicesEqual(path.Path, parentPath) {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path for target_id %d = %v, want parent chain %v", path.TargetID, path.Path, parentPath))
		}
		if want, ok := expected[path.TargetID]; ok && !intSlicesEqual(path.Path, want) {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path for target_id %d = %v, want %v", path.TargetID, path.Path, want))
		}
		if !rectContainsPoint(target.Bounds, path.X, path.Y) {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path target_id %d coordinates %d,%d are outside target bounds", path.TargetID, path.X, path.Y))
		} else if hitID, ok := componentTreeHitTest(nodes, path.X, path.Y); !ok || hitID != path.TargetID {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path coordinates %d,%d hit node %d, want target_id %d", path.X, path.Y, hitID, path.TargetID))
		}
		uniqueLeafTargets[path.TargetID] = true
	}
	for targetID := range expected {
		found := false
		for _, path := range paths {
			if path.TargetID == targetID {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, fmt.Sprintf("component_tree dispatch path missing target_id %d", targetID))
		}
	}
	if len(uniqueLeafTargets) < 2 {
		issues = append(issues, "component_tree dispatch paths require at least two different leaf targets")
	}
	return issues
}

func componentTreePathToRoot(id int, nodes map[int]ComponentTreeNodeReport) ([]int, bool) {
	var reversed []int
	seen := map[int]bool{}
	for {
		node, ok := nodes[id]
		if !ok || seen[id] {
			return nil, false
		}
		seen[id] = true
		reversed = append(reversed, id)
		if node.ParentID < 0 {
			break
		}
		id = node.ParentID
	}
	path := make([]int, len(reversed))
	for i := range reversed {
		path[i] = reversed[len(reversed)-1-i]
	}
	return path, true
}

func componentTreeHitTest(nodes map[int]ComponentTreeNodeReport, x int, y int) (int, bool) {
	bestID := -1
	bestDepth := -1
	for id, node := range nodes {
		if !rectContainsPoint(node.Bounds, x, y) {
			continue
		}
		path, ok := componentTreePathToRoot(id, nodes)
		if !ok {
			continue
		}
		if len(path) > bestDepth {
			bestDepth = len(path)
			bestID = id
		}
	}
	return bestID, bestID >= 0
}

func hasComponentTreeTabFocus(events []EventReport, before string, after string) bool {
	for _, event := range events {
		if event.Kind != "key_down" || event.Key != 9 || !event.Handled || !event.Pass {
			continue
		}
		beforeFocus, beforeOK := stateValueWithSuffix(event.BeforeState, ".focused_id")
		afterFocus, afterOK := stateValueWithSuffix(event.AfterState, ".focused_id")
		if beforeOK && afterOK && beforeFocus == before && afterFocus == after {
			return true
		}
	}
	return false
}

func hasComponentTreeTextInsertion(events []EventReport) bool {
	for _, event := range events {
		if event.Kind != "text_input" || !event.Handled || !event.Pass || !strings.HasSuffix(event.TargetComponent, "TextBox") {
			continue
		}
		key := event.TargetComponent + ".buffer"
		if event.BeforeState[key] != event.AfterState[key] && event.AfterState[key] != "" {
			return true
		}
	}
	return false
}

func componentTreeTextMutatedWhileButtonFocused(events []EventReport) bool {
	for _, event := range events {
		if event.Kind != "text_input" {
			continue
		}
		for key, before := range event.BeforeState {
			if !strings.HasSuffix(key, "TextBox.buffer") {
				continue
			}
			after, afterOK := event.AfterState[key]
			if !afterOK || before == after {
				continue
			}
			if !strings.HasPrefix(key, event.TargetComponent+".") {
				return true
			}
		}
	}
	return false
}

func hasComponentTreeButtonAction(events []EventReport) bool {
	seenSubmit := false
	seenReset := false
	for _, event := range events {
		if !event.Handled || !event.Pass {
			continue
		}
		if event.Kind != "key_down" || (event.Key != 32 && event.Key != 13) {
			continue
		}
		if (event.TargetComponent == "SubmitButton" || event.TargetComponent == "SaveButton") &&
			dispatchPathHasSuffix(event.DispatchPath, "ButtonRow", event.TargetComponent) &&
			(stateChangedBySuffix(event.BeforeState, event.AfterState, ".submitted_count") ||
				stateChangedBySuffix(event.BeforeState, event.AfterState, ".submit_count") ||
				stateChangedBySuffix(event.BeforeState, event.AfterState, ".save_count")) {
			seenSubmit = true
		}
		if event.TargetComponent == "ResetButton" &&
			dispatchPathHasSuffix(event.DispatchPath, "ButtonRow", "ResetButton") &&
			stateChangedBySuffix(event.BeforeState, event.AfterState, ".reset_count") &&
			textBoxBufferChanged(event.BeforeState, event.AfterState) {
			seenReset = true
		}
	}
	return seenSubmit && seenReset
}

func hasComponentTreeResizeRelayout(events []EventReport, transitions []StateTransitionReport) bool {
	seenEvent := false
	for _, event := range events {
		if event.Kind != "resize" || !event.Handled || !event.Pass {
			continue
		}
		beforeFocus, beforeOK := stateValueWithSuffix(event.BeforeState, ".focused_id")
		afterFocus, afterOK := stateValueWithSuffix(event.AfterState, ".focused_id")
		if beforeOK && afterOK && beforeFocus == afterFocus && textBoxBoundsChanged(event.BeforeState, event.AfterState) {
			seenEvent = true
		}
	}
	seenTransition := false
	for _, transition := range transitions {
		if transition.Cause == "resize" && strings.HasSuffix(transition.Field, "TextBox.bounds.w") && transition.Before != transition.After {
			seenTransition = true
		}
	}
	return seenEvent && seenTransition
}

func textBoxBufferChanged(before map[string]string, after map[string]string) bool {
	for key, beforeValue := range before {
		if !strings.HasSuffix(key, "TextBox.buffer") {
			continue
		}
		if afterValue, ok := after[key]; ok && beforeValue != afterValue {
			return true
		}
	}
	return false
}

func textBoxBoundsChanged(before map[string]string, after map[string]string) bool {
	for key, beforeValue := range before {
		if !strings.HasSuffix(key, "TextBox.bounds.w") {
			continue
		}
		if afterValue, ok := after[key]; ok && beforeValue != afterValue {
			return true
		}
	}
	return false
}
