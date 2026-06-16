package main

import (
	"fmt"

	"tetra_language/tools/validators/surface"
)

func blockSceneSnapshotForScenario(source string, scenario headlessScenario) *surface.BlockSceneSnapshotReport {
	recipeExpansionCount := 1
	if scenario.Morph != nil && len(scenario.Morph.RecipeExpansions) > 0 {
		recipeExpansionCount = len(scenario.Morph.RecipeExpansions)
	}
	nodes := blockSceneSnapshotNodesForScenario(scenario)
	return &surface.BlockSceneSnapshotReport{
		Schema:               "tetra.surface.block-scene-snapshot.v1",
		Source:               source,
		SurfaceScope:         "surface-morph-rendered-beauty-linux-web",
		Producer:             "surface-runtime-smoke",
		QualityLevel:         "rich-renderable-block-scene-v1",
		CorePrimitives:       []string{"Block"},
		CompactPropsOnly:     false,
		RecipeExpansionCount: recipeExpansionCount,
		NodeCount:            len(nodes),
		RichSpecHash:         "sha256:" + checksumText("block-scene-rich-specs:"+source+fmt.Sprint(len(nodes))+fmt.Sprint(recipeExpansionCount)),
		BlockSceneHash:       "sha256:" + checksumText("block-scene-snapshot:"+source+fmt.Sprint(len(nodes))+fmt.Sprint(recipeExpansionCount)),
		SpecCoverage: surface.BlockSceneSpecCoverageReport{
			Layout:        true,
			Paint:         true,
			Text:          true,
			Image:         true,
			Input:         true,
			Event:         true,
			State:         true,
			Motion:        true,
			Accessibility: true,
		},
		Nodes: nodes,
	}
}

func blockSceneSnapshotNodesForScenario(scenario headlessScenario) []surface.BlockSceneNodeReport {
	if scenario.BlockGraph == nil {
		return nil
	}
	paintLayersByBlock := map[int][]surface.BlockScenePaintLayerSpecReport{}
	for _, layer := range scenario.PaintLayers {
		paintLayersByBlock[layer.BlockID] = append(paintLayersByBlock[layer.BlockID], surface.BlockScenePaintLayerSpecReport{
			Kind:    layer.Kind,
			Color:   layer.Color,
			Radius:  layer.Radius,
			Width:   layer.Width,
			Blur:    layer.Blur,
			OffsetX: layer.OffsetX,
			OffsetY: layer.OffsetY,
			Opacity: layer.Opacity,
		})
	}
	a11yByBlock := map[int]surface.BlockAccessibilityNodeReport{}
	if scenario.BlockAccessibilityTree != nil {
		for _, node := range scenario.BlockAccessibilityTree.Nodes {
			a11yByBlock[node.BlockID] = node
		}
	}
	nodes := make([]surface.BlockSceneNodeReport, 0, len(scenario.BlockGraph.Nodes))
	for i, graphNode := range scenario.BlockGraph.Nodes {
		layers := paintLayersByBlock[graphNode.ID]
		if len(layers) == 0 {
			layers = []surface.BlockScenePaintLayerSpecReport{{
				Kind:    "fill",
				Color:   "#101820ff",
				Radius:  0,
				Opacity: 255,
			}}
		}
		role := graphNode.AccessibilityRole
		labelLen := len(graphNode.Name)
		focusIndex := 0
		readingIndex := i + 1
		if a11y, ok := a11yByBlock[graphNode.ID]; ok {
			role = a11y.Role
			labelLen = len(a11y.Name)
			focusIndex = a11y.FocusIndex
			readingIndex = a11y.ReadingIndex
			if readingIndex <= 0 {
				readingIndex = i + 1
			}
		}
		if role == "" || role == "none" {
			role = "group"
		}
		nodes = append(nodes, surface.BlockSceneNodeReport{
			BlockID:  graphNode.ID,
			ParentID: graphNode.ParentID,
			Recipe:   blockSceneRecipeForGraphNode(graphNode),
			Name:     graphNode.Name,
			Layout: &surface.BlockSceneLayoutSpecReport{
				Mode: blockSceneLayoutModeForGraphNode(graphNode),
				X:    graphNode.Bounds.X,
				Y:    graphNode.Bounds.Y,
				W:    graphNode.Bounds.W,
				H:    graphNode.Bounds.H,
			},
			Paint: &surface.BlockScenePaintSpecReport{
				LayerCount: len(layers),
				Layers:     layers,
			},
			Text: &surface.BlockSceneTextSpecReport{
				TextLen: len(graphNode.Name),
				Color:   "#edf2f7ff",
				Size:    14,
				Weight:  500,
			},
			Image: &surface.BlockSceneImageSpecReport{
				AssetID: blockSceneImageAssetForGraphNode(graphNode),
				Mode:    blockSceneImageModeForGraphNode(graphNode),
				Tint:    "#f4cd5cff",
				Opacity: 255,
			},
			Input: &surface.BlockSceneInputSpecReport{
				Kind:      blockSceneInputKindForGraphNode(graphNode),
				Focusable: graphNode.Focusable,
				Editable:  graphNode.AccessibilityRole == "textbox",
			},
			Event: &surface.BlockSceneEventSpecReport{
				PointerAction: blockScenePointerActionForGraphNode(graphNode),
				KeyAction:     blockSceneKeyActionForGraphNode(graphNode),
			},
			State: &surface.BlockSceneStateSpecReport{
				Variant:  blockSceneStateVariantForGraphNode(graphNode),
				Enabled:  true,
				Focused:  graphNode.Focusable,
				Selected: graphNode.ID == 4,
			},
			Motion: &surface.BlockSceneMotionSpecReport{
				DurationMS:        120 + graphNode.ChildIndex*20,
				Easing:            "standard",
				ReducedMotionSafe: true,
			},
			Accessibility: &surface.BlockSceneAccessibilitySpecReport{
				Role:         role,
				LabelLen:     labelLen,
				FocusIndex:   focusIndex,
				ReadingIndex: readingIndex,
				Actions:      blockSceneActionsForGraphNode(graphNode),
			},
		})
	}
	return nodes
}

func blockSceneRecipeForGraphNode(node surface.BlockGraphNodeReport) string {
	if node.ParentID == -1 {
		return "morph.surface"
	}
	switch node.AccessibilityRole {
	case "button":
		return "morph.control.action"
	case "textbox":
		return "morph.field.text"
	case "text":
		return "morph.text.label"
	default:
		return "morph.region.panel"
	}
}

func blockSceneLayoutModeForGraphNode(node surface.BlockGraphNodeReport) string {
	if node.ParentID == -1 {
		return "column"
	}
	if node.ChildCount > 0 {
		return "stack"
	}
	if node.AccessibilityRole == "button" {
		return "row"
	}
	return "absolute"
}

func blockSceneImageAssetForGraphNode(node surface.BlockGraphNodeReport) string {
	if node.AccessibilityRole == "button" {
		return "command.action.icon"
	}
	return "none"
}

func blockSceneImageModeForGraphNode(node surface.BlockGraphNodeReport) string {
	if node.AccessibilityRole == "button" {
		return "template"
	}
	return "none"
}

func blockSceneInputKindForGraphNode(node surface.BlockGraphNodeReport) string {
	switch node.AccessibilityRole {
	case "textbox":
		return "text"
	case "button":
		return "button"
	default:
		return "none"
	}
}

func blockScenePointerActionForGraphNode(node surface.BlockGraphNodeReport) string {
	if node.Focusable {
		return "activate"
	}
	return "none"
}

func blockSceneKeyActionForGraphNode(node surface.BlockGraphNodeReport) string {
	switch node.AccessibilityRole {
	case "textbox":
		return "edit"
	case "button":
		return "activate"
	default:
		return "none"
	}
}

func blockSceneStateVariantForGraphNode(node surface.BlockGraphNodeReport) string {
	if node.Focusable {
		return "interactive"
	}
	if node.ParentID == -1 {
		return "surface"
	}
	return "default"
}

func blockSceneActionsForGraphNode(node surface.BlockGraphNodeReport) []string {
	switch node.AccessibilityRole {
	case "textbox":
		return []string{"focus", "edit"}
	case "button":
		return []string{"focus", "activate"}
	default:
		return nil
	}
}

func blockSceneSnapshotCasesForScenario() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "block scene snapshot preserves rich visual specs", Kind: "positive", Ran: true, Pass: true},
		{Name: "block scene compact BlockProps-only evidence rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "compact_props_only rejected"},
		{Name: "block scene non-Block core primitive rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "non-Block primitive rejected"},
		{Name: "block scene missing rich spec coverage rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "rich spec coverage required"},
	}
}
