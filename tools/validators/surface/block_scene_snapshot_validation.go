package surface

import (
	"fmt"
	"strings"
)

type BlockSceneSnapshotReport struct {
	Schema               string                       `json:"schema"`
	Source               string                       `json:"source"`
	SurfaceScope         string                       `json:"surface_scope"`
	Producer             string                       `json:"producer"`
	QualityLevel         string                       `json:"quality_level"`
	CorePrimitives       []string                     `json:"core_primitives"`
	CompactPropsOnly     bool                         `json:"compact_props_only"`
	RecipeExpansionCount int                          `json:"recipe_expansion_count"`
	NodeCount            int                          `json:"node_count"`
	RichSpecHash         string                       `json:"rich_spec_hash"`
	BlockSceneHash       string                       `json:"block_scene_hash"`
	SpecCoverage         BlockSceneSpecCoverageReport `json:"spec_coverage"`
	Nodes                []BlockSceneNodeReport       `json:"nodes"`
}

type BlockSceneSpecCoverageReport struct {
	Layout        bool `json:"layout"`
	Paint         bool `json:"paint"`
	Text          bool `json:"text"`
	Image         bool `json:"image"`
	Input         bool `json:"input"`
	Event         bool `json:"event"`
	State         bool `json:"state"`
	Motion        bool `json:"motion"`
	Accessibility bool `json:"accessibility"`
}

type BlockSceneNodeReport struct {
	BlockID       int                                `json:"block_id"`
	ParentID      int                                `json:"parent_id"`
	Recipe        string                             `json:"recipe"`
	Name          string                             `json:"name"`
	Layout        *BlockSceneLayoutSpecReport        `json:"layout"`
	Paint         *BlockScenePaintSpecReport         `json:"paint"`
	Text          *BlockSceneTextSpecReport          `json:"text"`
	Image         *BlockSceneImageSpecReport         `json:"image"`
	Input         *BlockSceneInputSpecReport         `json:"input"`
	Event         *BlockSceneEventSpecReport         `json:"event"`
	State         *BlockSceneStateSpecReport         `json:"state"`
	Motion        *BlockSceneMotionSpecReport        `json:"motion"`
	Accessibility *BlockSceneAccessibilitySpecReport `json:"accessibility"`
}

type BlockSceneLayoutSpecReport struct {
	Mode string `json:"mode"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	W    int    `json:"w"`
	H    int    `json:"h"`
}

type BlockScenePaintSpecReport struct {
	LayerCount int                              `json:"layer_count"`
	Layers     []BlockScenePaintLayerSpecReport `json:"layers"`
}

type BlockScenePaintLayerSpecReport struct {
	Kind    string `json:"kind"`
	Color   string `json:"color,omitempty"`
	Radius  int    `json:"radius,omitempty"`
	Width   int    `json:"width,omitempty"`
	Blur    int    `json:"blur,omitempty"`
	OffsetX int    `json:"offset_x,omitempty"`
	OffsetY int    `json:"offset_y,omitempty"`
	Opacity int    `json:"opacity,omitempty"`
}

type BlockSceneTextSpecReport struct {
	TextLen int    `json:"text_len"`
	HintLen int    `json:"hint_len,omitempty"`
	Color   string `json:"color,omitempty"`
	Size    int    `json:"size,omitempty"`
	Weight  int    `json:"weight,omitempty"`
}

type BlockSceneImageSpecReport struct {
	AssetID string `json:"asset_id"`
	Mode    string `json:"mode"`
	Tint    string `json:"tint,omitempty"`
	Opacity int    `json:"opacity"`
}

type BlockSceneInputSpecReport struct {
	Kind      string `json:"kind"`
	Focusable bool   `json:"focusable"`
	Editable  bool   `json:"editable"`
}

type BlockSceneEventSpecReport struct {
	PointerAction string `json:"pointer_action"`
	KeyAction     string `json:"key_action"`
}

type BlockSceneStateSpecReport struct {
	Variant  string `json:"variant"`
	Enabled  bool   `json:"enabled"`
	Focused  bool   `json:"focused,omitempty"`
	Selected bool   `json:"selected,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
	Error    bool   `json:"error,omitempty"`
	Loading  bool   `json:"loading,omitempty"`
}

type BlockSceneMotionSpecReport struct {
	DurationMS        int    `json:"duration_ms"`
	Easing            string `json:"easing"`
	ReducedMotionSafe bool   `json:"reduced_motion_safe"`
}

type BlockSceneAccessibilitySpecReport struct {
	Role         string   `json:"role"`
	LabelLen     int      `json:"label_len"`
	FocusIndex   int      `json:"focus_index,omitempty"`
	ReadingIndex int      `json:"reading_index"`
	Actions      []string `json:"actions,omitempty"`
}

func validateBlockSceneSnapshotEvidence(report Report) []string {
	if report.BlockSceneSnapshot == nil {
		return nil
	}

	snapshot := report.BlockSceneSnapshot
	var issues []string
	if snapshot.Schema != "tetra.surface.block-scene-snapshot.v1" {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot schema is %q, want tetra.surface.block-scene-snapshot.v1", snapshot.Schema))
	}
	if normalizeEvidencePath(snapshot.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot source %q must match report source %q", snapshot.Source, report.Source))
	}
	if snapshot.SurfaceScope != "surface-morph-rendered-beauty-linux-web" {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot surface_scope is %q, want surface-morph-rendered-beauty-linux-web", snapshot.SurfaceScope))
	}
	if strings.TrimSpace(snapshot.Producer) == "" {
		issues = append(issues, "block_scene_snapshot producer is required")
	}
	if snapshot.QualityLevel != "rich-renderable-block-scene-v1" {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot quality_level is %q, want rich-renderable-block-scene-v1", snapshot.QualityLevel))
	}
	if len(snapshot.CorePrimitives) != 1 || !stringSliceContainsFold(snapshot.CorePrimitives, "Block") {
		issues = append(issues, "block_scene_snapshot core_primitives must contain only Block")
	}
	for _, primitive := range snapshot.CorePrimitives {
		primitive = strings.TrimSpace(primitive)
		if !strings.EqualFold(primitive, "Block") {
			issues = append(issues, fmt.Sprintf("block_scene_snapshot core_primitives must not include %s", primitive))
		}
	}
	if snapshot.CompactPropsOnly {
		issues = append(issues, "block_scene_snapshot compact_props_only must be false; rich visual specs are required")
	}
	if snapshot.RecipeExpansionCount <= 0 {
		issues = append(issues, "block_scene_snapshot recipe_expansion_count must be positive")
	}
	if snapshot.NodeCount != len(snapshot.Nodes) {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node_count = %d, want len(nodes) %d", snapshot.NodeCount, len(snapshot.Nodes)))
	}
	if snapshot.NodeCount == 0 {
		issues = append(issues, "block_scene_snapshot nodes evidence is required")
	}
	if !validSHA256Digest(snapshot.RichSpecHash) {
		issues = append(issues, "block_scene_snapshot rich_spec_hash must be sha256 evidence")
	}
	if !validSHA256Digest(snapshot.BlockSceneHash) {
		issues = append(issues, "block_scene_snapshot block_scene_hash must be sha256 evidence")
	}
	issues = append(issues, validateBlockSceneSpecCoverage(snapshot.SpecCoverage)...)
	issues = append(issues, validateBlockSceneNodes(snapshot.Nodes, report.BlockGraph)...)
	return issues
}

func validateBlockSceneSpecCoverage(coverage BlockSceneSpecCoverageReport) []string {
	var issues []string
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{"layout", coverage.Layout},
		{"paint", coverage.Paint},
		{"text", coverage.Text},
		{"image", coverage.Image},
		{"input", coverage.Input},
		{"event", coverage.Event},
		{"state", coverage.State},
		{"motion", coverage.Motion},
		{"accessibility", coverage.Accessibility},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("block_scene_snapshot spec_coverage missing %s", check.name))
		}
	}
	return issues
}

func validateBlockSceneNodes(nodes []BlockSceneNodeReport, graph *BlockGraphReport) []string {
	var issues []string
	graphNodes := map[int]BlockGraphNodeReport{}
	if graph != nil {
		for _, node := range graph.Nodes {
			graphNodes[node.ID] = node
		}
	}
	seen := map[int]bool{}
	rootCount := 0
	for _, node := range nodes {
		if node.BlockID <= 0 {
			issues = append(issues, "block_scene_snapshot node block_id must be positive")
			continue
		}
		if seen[node.BlockID] {
			issues = append(issues, fmt.Sprintf("block_scene_snapshot duplicate node block_id %d", node.BlockID))
		}
		seen[node.BlockID] = true
		if node.ParentID == -1 {
			rootCount++
		}
		if strings.TrimSpace(node.Name) == "" {
			issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d name is required", node.BlockID))
		}
		if strings.TrimSpace(node.Recipe) == "" {
			issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d recipe is required", node.BlockID))
		}
		if graph != nil {
			graphNode, ok := graphNodes[node.BlockID]
			if !ok {
				issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d is not in block_graph", node.BlockID))
			} else {
				if node.ParentID != graphNode.ParentID {
					issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d parent_id = %d, want block_graph parent_id %d", node.BlockID, node.ParentID, graphNode.ParentID))
				}
				if node.Layout != nil {
					rect := RectReport{X: node.Layout.X, Y: node.Layout.Y, W: node.Layout.W, H: node.Layout.H}
					if !rectsEqual(rect, graphNode.Bounds) {
						issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d layout bounds %+v must match block_graph bounds %+v", node.BlockID, rect, graphNode.Bounds))
					}
				}
			}
		}
		issues = append(issues, validateBlockSceneRichSpecs(node)...)
	}
	if len(nodes) > 0 && rootCount != 1 {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot root count = %d, want 1", rootCount))
	}
	for _, node := range nodes {
		if node.ParentID >= 0 && !seen[node.ParentID] {
			issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d parent_id %d is not in snapshot nodes", node.BlockID, node.ParentID))
		}
	}
	return issues
}

func validateBlockSceneRichSpecs(node BlockSceneNodeReport) []string {
	var issues []string
	if node.Layout == nil {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d layout spec is required", node.BlockID))
	} else if !validLayoutMode(node.Layout.Mode) || node.Layout.W <= 0 || node.Layout.H <= 0 {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d layout spec must include supported mode and positive bounds", node.BlockID))
	}
	if node.Paint == nil {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d paint spec is required", node.BlockID))
	} else {
		if node.Paint.LayerCount != len(node.Paint.Layers) {
			issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d paint layer_count = %d, want len(layers) %d", node.BlockID, node.Paint.LayerCount, len(node.Paint.Layers)))
		}
		if node.Paint.LayerCount == 0 {
			issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d paint layers are required", node.BlockID))
		}
		for _, layer := range node.Paint.Layers {
			if strings.TrimSpace(layer.Kind) == "" {
				issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d paint layer kind is required", node.BlockID))
			}
		}
	}
	if node.Text == nil {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d text spec is required", node.BlockID))
	} else if node.Text.TextLen < 0 || node.Text.HintLen < 0 {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d text lengths must be non-negative", node.BlockID))
	}
	if node.Image == nil {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d image spec is required", node.BlockID))
	} else if strings.TrimSpace(node.Image.AssetID) == "" || strings.TrimSpace(node.Image.Mode) == "" {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d image asset_id and mode are required", node.BlockID))
	}
	if node.Input == nil {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d input spec is required", node.BlockID))
	} else if strings.TrimSpace(node.Input.Kind) == "" {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d input kind is required", node.BlockID))
	}
	if node.Event == nil {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d event spec is required", node.BlockID))
	} else if strings.TrimSpace(node.Event.PointerAction) == "" || strings.TrimSpace(node.Event.KeyAction) == "" {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d event actions are required", node.BlockID))
	}
	if node.State == nil {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d state spec is required", node.BlockID))
	} else if strings.TrimSpace(node.State.Variant) == "" {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d state variant is required", node.BlockID))
	}
	if node.Motion == nil {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d motion spec is required", node.BlockID))
	} else if node.Motion.DurationMS <= 0 || strings.TrimSpace(node.Motion.Easing) == "" || !node.Motion.ReducedMotionSafe {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d motion spec requires positive duration, easing, and reduced_motion_safe", node.BlockID))
	}
	if node.Accessibility == nil {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d accessibility spec is required", node.BlockID))
	} else if strings.TrimSpace(node.Accessibility.Role) == "" || node.Accessibility.LabelLen <= 0 || node.Accessibility.ReadingIndex <= 0 {
		issues = append(issues, fmt.Sprintf("block_scene_snapshot node %d accessibility spec requires role, label_len, and reading_index", node.BlockID))
	}
	return issues
}
