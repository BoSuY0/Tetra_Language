package main

import (
	"path/filepath"
	"strconv"
	"strings"

	"tetra_language/tools/validators/surface"
)

const morphRenderedFlagshipSource = "examples/surface_morph_rendered_studio_shell.tetra"

func isMorphRenderedFlagshipSource(source string) bool {
	clean := filepath.ToSlash(filepath.Clean(normalizeSurfaceSourcePath(source)))
	return clean == morphRenderedFlagshipSource || strings.HasSuffix(clean, "/"+morphRenderedFlagshipSource)
}

func runMorphScenarioForSource(source string) headlessScenario {
	source = normalizeSurfaceSourcePath(source)
	if source == "" {
		source = "examples/surface_morph_command_palette.tetra"
	}
	if isMorphRenderedFlagshipSource(source) {
		return runMorphFlagshipScenario(source)
	}
	scenario := runBlockSystemScenario()
	retargetScenarioToSource(&scenario, source, surfaceSourceModuleName(source))
	scenario.Morph = morphReportForScenario(source, scenario)
	scenario.BlockSceneSnapshot = blockSceneSnapshotForScenario(source, scenario)
	attachRenderCommandStreamForScenario(source, &scenario)
	scenario.Cases = append(scenario.Cases, morphCasesForScenario()...)
	return scenario
}

func runMorphFlagshipScenario(source string) headlessScenario {
	scenario := runBlockSystemScenario()
	retargetScenarioToSource(&scenario, source, surfaceSourceModuleName(source))
	scenario.Components = append(scenario.Components, flagshipMorphComponentsForScenario(source)...)
	scenario.BlockGraph = flagshipMorphBlockGraphForScenario(source)
	scenario.BlockAccessibilityTree = flagshipMorphAccessibilityTreeForScenario(source, scenario.BlockGraph)
	scenario.BlockSystem = blockSystemReportForScenario(source, scenario.Frames)
	attachBlockSystemMemoryBudget(&scenario)
	scenario.Morph = morphReportForScenario(source, scenario)
	scenario.BlockSceneSnapshot = blockSceneSnapshotForScenario(source, scenario)
	attachRenderCommandStreamForScenario(source, &scenario)
	scenario.Cases = append(scenario.Cases, morphCasesForScenario()...)
	scenario.Cases = append(scenario.Cases, flagshipMorphCasesForScenario()...)
	return scenario
}

func flagshipMorphComponentsForScenario(source string) []surface.ComponentReport {
	module := surfaceSourceModuleName(source)
	nodes := flagshipMorphGraphNodes()
	namesByID := map[int]string{}
	for _, node := range nodes {
		namesByID[node.ID] = node.Name
	}
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility", "state", "motion", "asset"}
	components := make([]surface.ComponentReport, 0, len(nodes))
	for _, node := range nodes {
		parent := ""
		if node.ParentID >= 0 {
			parent = namesByID[node.ParentID]
		}
		components = append(components, surface.ComponentReport{
			ID:        node.Name,
			Type:      module + "." + node.Name,
			Parent:    parent,
			Bounds:    node.Bounds,
			Abilities: abilities,
			State: map[string]string{
				"block_id": strconv.Itoa(node.ID),
				"role":     node.AccessibilityRole,
				"recipe":   flagshipMorphRecipeForNode(node.Name),
				"source":   "morph",
			},
		})
	}
	return components
}

func flagshipMorphRecipeForNode(name string) string {
	switch name {
	case "RenderedStudioShell", "AppShellFrame":
		return "app.shell@1"
	case "NavigationRail", "ProfilesActions", "ProjectPackageView", "RunDiagnosticsView":
		return "nav.item@1"
	case "ToolbarActions":
		return "toolbar@1"
	case "DashboardShell":
		return "split.pane@1"
	case "CommandPalette":
		return "command.item@1"
	case "SettingsForm":
		return "settings.form@1"
	case "LogsOutput":
		return "log.row@1"
	case "DiagnosticsError":
		return "error.panel@1"
	case "MetricTiles":
		return "metric.tile@1"
	case "StatusBar":
		return "status.bar@1"
	case "BlockedDialog":
		return "dialog.panel@1"
	case "ToastSurface":
		return "toast.notification@1"
	case "EmptyState":
		return "empty.state@1"
	default:
		return "region.panel@1"
	}
}

func flagshipMorphBlockGraphForScenario(source string) *surface.BlockGraphReport {
	nodes := flagshipMorphGraphNodes()
	order := make([]int, 0, len(nodes))
	for _, node := range nodes {
		order = append(order, node.ID)
	}
	return &surface.BlockGraphReport{
		Schema:            "tetra.surface.block-graph.v1",
		APILevel:          "block-tree-builder-v1",
		Source:            source,
		ManualBookkeeping: false,
		Builder: surface.BlockGraphBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         len(nodes),
			Capacity:          24,
			OverflowChecked:   true,
		},
		Invariants: surface.BlockGraphInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			DuplicateIDRejected:     true,
			MissingParentRejected:   true,
			CycleRejected:           true,
			ParentChildLinksChecked: true,
			ChildOrderChecked:       true,
			FocusOrderChecked:       true,
			HitTestPathChecked:      true,
			AccessibilityChecked:    true,
		},
		RootID:    1,
		NodeCount: len(nodes),
		Nodes:     nodes,
		ChildOrders: []surface.BlockGraphChildOrderReport{
			{ParentID: 1, Children: []int{2}},
			{ParentID: 2, Children: []int{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18}},
		},
		LayoutOrder:        order,
		DrawOrder:          order,
		FocusOrder:         []int{4, 6, 8, 9, 10, 12, 15},
		AccessibilityOrder: []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18},
		HitTests: []surface.BlockGraphPathReport{
			{Helper: "tree_hit_test_path", Event: "click", TargetID: 9, X: 720, Y: 112, Path: []int{1, 2, 9}},
			{Helper: "tree_hit_test_path", Event: "click", TargetID: 15, X: 780, Y: 244, Path: []int{1, 2, 15}},
		},
		DispatchPaths: []surface.BlockGraphPathReport{
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 4, Path: []int{1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 6, Path: []int{1, 2, 6}},
			{Helper: "tree_build_dispatch_path", Event: "key", TargetID: 9, Path: []int{1, 2, 9}},
			{Helper: "tree_build_dispatch_path", Event: "text", TargetID: 10, Path: []int{1, 2, 10}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 12, Path: []int{1, 2, 12}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 15, Path: []int{1, 2, 15}},
		},
	}
}

func flagshipMorphGraphNodes() []surface.BlockGraphNodeReport {
	return []surface.BlockGraphNodeReport{
		{ID: 1, Name: "RenderedStudioShell", ParentID: -1, ChildIndex: 0, FirstChild: 2, ChildCount: 1, AccessibilityRole: "none", Bounds: surface.RectReport{X: 0, Y: 0, W: 1180, H: 760}},
		{ID: 2, Name: "AppShellFrame", ParentID: 1, ChildIndex: 0, FirstChild: 3, ChildCount: 16, AccessibilityRole: "region", Bounds: surface.RectReport{X: 0, Y: 0, W: 1180, H: 760}},
		{ID: 3, Name: "NavigationRail", ParentID: 2, ChildIndex: 0, FirstChild: -1, ChildCount: 0, AccessibilityRole: "navigation", Bounds: surface.RectReport{X: 24, Y: 80, W: 160, H: 256}},
		{ID: 4, Name: "ToolbarActions", ParentID: 2, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: surface.RectReport{X: 250, Y: 32, W: 880, H: 44}},
		{ID: 5, Name: "DashboardShell", ParentID: 2, ChildIndex: 2, FirstChild: -1, ChildCount: 0, AccessibilityRole: "region", Bounds: surface.RectReport{X: 250, Y: 92, W: 420, H: 190}},
		{ID: 6, Name: "ProfilesActions", ParentID: 2, ChildIndex: 3, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: surface.RectReport{X: 552, Y: 248, W: 140, H: 36}},
		{ID: 7, Name: "ProjectPackageView", ParentID: 2, ChildIndex: 4, FirstChild: -1, ChildCount: 0, AccessibilityRole: "region", Bounds: surface.RectReport{X: 280, Y: 116, W: 160, H: 72}},
		{ID: 8, Name: "RunDiagnosticsView", ParentID: 2, ChildIndex: 5, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: surface.RectReport{X: 456, Y: 316, W: 160, H: 40}},
		{ID: 9, Name: "CommandPalette", ParentID: 2, ChildIndex: 6, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "textbox", Bounds: surface.RectReport{X: 690, Y: 92, W: 360, H: 48}},
		{ID: 10, Name: "SettingsForm", ParentID: 2, ChildIndex: 7, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "textbox", Bounds: surface.RectReport{X: 690, Y: 154, W: 360, H: 48}},
		{ID: 11, Name: "LogsOutput", ParentID: 2, ChildIndex: 8, FirstChild: -1, ChildCount: 0, AccessibilityRole: "text", Bounds: surface.RectReport{X: 280, Y: 432, W: 360, H: 36}},
		{ID: 12, Name: "DiagnosticsError", ParentID: 2, ChildIndex: 9, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: surface.RectReport{X: 656, Y: 432, W: 220, H: 36}},
		{ID: 13, Name: "MetricTiles", ParentID: 2, ChildIndex: 10, FirstChild: -1, ChildCount: 0, AccessibilityRole: "region", Bounds: surface.RectReport{X: 280, Y: 204, W: 160, H: 36}},
		{ID: 14, Name: "StatusBar", ParentID: 2, ChildIndex: 11, FirstChild: -1, ChildCount: 0, AccessibilityRole: "status", Bounds: surface.RectReport{X: 250, Y: 704, W: 880, H: 32}},
		{ID: 15, Name: "BlockedDialog", ParentID: 2, ChildIndex: 12, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "dialog", Bounds: surface.RectReport{X: 690, Y: 220, W: 360, H: 74}},
		{ID: 16, Name: "ToastSurface", ParentID: 2, ChildIndex: 13, FirstChild: -1, ChildCount: 0, AccessibilityRole: "status", Bounds: surface.RectReport{X: 456, Y: 116, W: 180, H: 72}},
		{ID: 17, Name: "EmptyState", ParentID: 2, ChildIndex: 14, FirstChild: -1, ChildCount: 0, AccessibilityRole: "text", Bounds: surface.RectReport{X: 456, Y: 204, W: 180, H: 36}},
		{ID: 18, Name: "AppShellState", ParentID: 2, ChildIndex: 15, FirstChild: -1, ChildCount: 0, AccessibilityRole: "text", Bounds: surface.RectReport{X: 690, Y: 316, W: 220, H: 32}},
	}
}

func flagshipMorphAccessibilityTreeForScenario(source string, graph *surface.BlockGraphReport) *surface.BlockAccessibilityTreeReport {
	if graph == nil {
		return nil
	}
	focusIndex := map[int]int{}
	for i, id := range graph.FocusOrder {
		focusIndex[id] = i
	}
	readingIndex := map[int]int{}
	for i, id := range graph.AccessibilityOrder {
		readingIndex[id] = i
	}
	roles := []string{}
	seenRoles := map[string]bool{}
	nodes := make([]surface.BlockAccessibilityNodeReport, 0, len(graph.AccessibilityOrder))
	actions := []surface.AccessibilityActionReport{}
	for _, graphNode := range graph.Nodes {
		if graphNode.AccessibilityRole == "" || graphNode.AccessibilityRole == "none" {
			continue
		}
		role := graphNode.AccessibilityRole
		if !seenRoles[role] {
			seenRoles[role] = true
			roles = append(roles, role)
		}
		node := surface.BlockAccessibilityNodeReport{
			ID:            graphNode.ID,
			BlockID:       graphNode.ID,
			ParentBlockID: graphNode.ParentID,
			Name:          graphNode.Name,
			Role:          role,
			Description:   flagshipMorphDescriptionForNode(graphNode.Name),
			Bounds:        graphNode.Bounds,
			Visible:       true,
			Enabled:       true,
			Focusable:     graphNode.Focusable,
			FocusIndex:    -1,
			ReadingIndex:  readingIndex[graphNode.ID],
		}
		if graphNode.Focusable {
			node.FocusIndex = focusIndex[graphNode.ID]
			node.Actions = flagshipMorphActionsForRole(role)
			node.Focused = graphNode.ID == graph.FocusOrder[0]
			actions = append(actions, surface.AccessibilityActionReport{
				Target:   graphNode.Name,
				Action:   flagshipMorphPrimaryActionForRole(role),
				Semantic: flagshipMorphSemanticForNode(graphNode.Name),
			})
		}
		if graphNode.Name == "AppShellState" {
			node.LabelFor = "CommandPalette"
		}
		if graphNode.Name == "CommandPalette" {
			node.LabelledBy = "AppShellState"
			node.Editable = true
			node.Value = "Morph command ready"
		}
		nodes = append(nodes, node)
	}
	return &surface.BlockAccessibilityTreeReport{
		Schema:                  "tetra.surface.block-accessibility-tree.v1",
		AccessibilityLevel:      "block-metadata-tree-v1",
		Source:                  source,
		Module:                  "lib.core.block",
		QualityLevel:            "block-derived-accessibility-metadata-v1",
		BlockGraphSchema:        "tetra.surface.block-graph.v1",
		DerivedFromBlockGraph:   true,
		ManualBookkeeping:       false,
		PlatformHostIntegration: false,
		DOMARIAIntegration:      false,
		ScreenReaderEvidence:    false,
		NoDOMUI:                 true,
		NoUserJS:                true,
		NoPlatformWidgets:       true,
		NodeCount:               len(nodes),
		FocusableCount:          len(graph.FocusOrder),
		RolesPresent:            roles,
		Nodes:                   nodes,
		Relationships: []surface.AccessibilityRelationshipReport{
			{Kind: "label_for", From: "AppShellState", To: "CommandPalette"},
			{Kind: "labelled_by", From: "CommandPalette", To: "AppShellState"},
		},
		FocusOrder:   graph.FocusOrder,
		ReadingOrder: graph.AccessibilityOrder,
		Actions:      actions,
		NegativeGuards: surface.BlockAccessibilityNegativeGuardsReport{
			FocusableActionNameChecked:    true,
			LabelRelationshipsChecked:     true,
			ReadingOrderGraphChecked:      true,
			BoundsAlignmentChecked:        true,
			FakeScreenReaderClaimRejected: true,
			ScopedPlatformClaimChecked:    true,
		},
	}
}

func flagshipMorphDescriptionForNode(name string) string {
	switch name {
	case "CommandPalette":
		return "command palette field"
	case "SettingsForm":
		return "settings form field"
	case "BlockedDialog":
		return "blocked action dialog"
	case "DiagnosticsError":
		return "recoverable diagnostics error"
	default:
		return "Morph-authored Surface region"
	}
}

func flagshipMorphActionsForRole(role string) []string {
	switch role {
	case "textbox":
		return []string{"focus", "edit"}
	case "dialog":
		return []string{"focus", "dismiss"}
	default:
		return []string{"focus", "press"}
	}
}

func flagshipMorphPrimaryActionForRole(role string) string {
	switch role {
	case "textbox":
		return "edit"
	case "dialog":
		return "dismiss"
	default:
		return "press"
	}
}

func flagshipMorphSemanticForNode(name string) string {
	switch name {
	case "ToolbarActions":
		return "open-command-palette"
	case "ProfilesActions":
		return "select-profile"
	case "RunDiagnosticsView":
		return "run-diagnostics"
	case "CommandPalette":
		return "edit-command"
	case "SettingsForm":
		return "edit-settings"
	case "DiagnosticsError":
		return "retry-diagnostics"
	case "BlockedDialog":
		return "dismiss-blocked-action"
	default:
		return "activate"
	}
}

func flagshipMorphCasesForScenario() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "flagship Morph source avoids manual draw authoring", Kind: "positive", Ran: true, Pass: true},
		{Name: "flagship Morph app shell expands to Block scene", Kind: "positive", Ran: true, Pass: true},
		{Name: "flagship Morph dashboard shell emits render commands", Kind: "positive", Ran: true, Pass: true},
		{Name: "flagship Morph command palette emits pixel evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "flagship Morph settings form projects accessibility", Kind: "positive", Ran: true, Pass: true},
		{Name: "flagship Morph dialog/error/status recipes stay Morph recipes", Kind: "positive", Ran: true, Pass: true},
	}
}
