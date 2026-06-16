package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportRejectsPrecomputedProductVisualFrame(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		attachProductVisualChainForTest(report)
		report.Frames[3].Producer = "surface-runtime-smoke"
		report.Frames[3].EvidenceRole = "product_visual"
		report.Frames[3].AppSource = report.Source
		report.Frames[3].Precomputed = true
		report.Frames[3].MorphRecipeHash = "sha256:" + strings.Repeat("a", 64)
		report.Frames[3].BlockSceneHash = report.BlockSceneSnapshot.BlockSceneHash
		report.Frames[3].RenderCommandStreamHash = report.RenderCommandStream.CommandStreamHash
	})

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected precomputed product visual frame to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "precomputed") ||
		!strings.Contains(strings.ToLower(err.Error()), "product visual") {
		t.Fatalf("error = %v, want precomputed product visual diagnostic", err)
	}
}

func TestValidateReportRejectsProductVisualFrameWithoutSourceLinks(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		attachProductVisualChainForTest(report)
		report.Frames[3].Producer = "app"
		report.Frames[3].EvidenceRole = "product_visual"
	})

	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected unlinked product visual frame to fail")
	}
	for _, want := range []string{"app_source", "morph_recipe_hash", "block_scene_hash", "render_command_stream_hash"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %s diagnostic", err, want)
		}
	}
}

func TestValidateReportAcceptsSourceLinkedProductVisualFrame(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		attachProductVisualChainForTest(report)
		report.Frames[3].Producer = "app"
		report.Frames[3].EvidenceRole = "product_visual"
		report.Frames[3].AppSource = report.Source
		report.Frames[3].MorphRecipeHash = "sha256:" + strings.Repeat("a", 64)
		report.Frames[3].BlockSceneHash = report.BlockSceneSnapshot.BlockSceneHash
		report.Frames[3].RenderCommandStreamHash = report.RenderCommandStream.CommandStreamHash
	})

	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v", err)
	}
}

func TestValidateReportAcceptsPrecomputedHostProbeOnlyFrame(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		report.Frames[3].Producer = "host_probe"
		report.Frames[3].EvidenceRole = "host_probe_only"
		report.Frames[3].Precomputed = true
	})

	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v", err)
	}
}

func TestBlockSystemRejectsProductVisualFrameFromPrecomputedRenderer(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		attachProductVisualChainForTest(report)
		report.Frames[3].Producer = "host_probe"
		report.Frames[3].EvidenceRole = "product_visual"
		report.Frames[3].AppSource = report.Source
		report.Frames[3].Precomputed = true
		report.Frames[3].MorphRecipeHash = "sha256:" + strings.Repeat("a", 64)
		report.Frames[3].BlockSceneHash = report.BlockSceneSnapshot.BlockSceneHash
		report.Frames[3].RenderCommandStreamHash = report.RenderCommandStream.CommandStreamHash
		report.BlockSystem.Frames[3].Producer = "host_probe"
		report.BlockSystem.Frames[3].EvidenceRole = "product_visual"
		report.BlockSystem.Frames[3].Precomputed = true
	})

	var decoded Report
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected precomputed Block-system product visual frame to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "block_system") ||
		!strings.Contains(strings.ToLower(err.Error()), "product visual") {
		t.Fatalf("error = %v, want Block-system product visual diagnostic", err)
	}
}

func attachProductVisualChainForTest(report *Report) {
	nodes := make([]BlockSceneNodeReport, 0, len(report.BlockGraph.Nodes))
	for _, graphNode := range report.BlockGraph.Nodes {
		nodes = append(nodes, BlockSceneNodeReport{
			BlockID:  graphNode.ID,
			ParentID: graphNode.ParentID,
			Recipe:   "morph.product_visual@1",
			Name:     graphNode.Name,
			Layout: &BlockSceneLayoutSpecReport{
				Mode: "absolute",
				X:    graphNode.Bounds.X,
				Y:    graphNode.Bounds.Y,
				W:    graphNode.Bounds.W,
				H:    graphNode.Bounds.H,
			},
			Paint: &BlockScenePaintSpecReport{
				LayerCount: 1,
				Layers: []BlockScenePaintLayerSpecReport{
					{Kind: "fill", Color: "#202733ff", Radius: 4, Opacity: 255},
				},
			},
			Text:  &BlockSceneTextSpecReport{TextLen: 4, Color: "#f4f7fbff", Size: 14, Weight: 500},
			Image: &BlockSceneImageSpecReport{AssetID: "search-icon", Mode: "template", Tint: "#b7c4d6ff", Opacity: 255},
			Input: &BlockSceneInputSpecReport{
				Kind:      "button",
				Focusable: graphNode.Focusable,
				Editable:  false,
			},
			Event:         &BlockSceneEventSpecReport{PointerAction: "press", KeyAction: "activate"},
			State:         &BlockSceneStateSpecReport{Variant: "focused", Enabled: true, Focused: graphNode.Focusable},
			Motion:        &BlockSceneMotionSpecReport{DurationMS: 120, Easing: "standard", ReducedMotionSafe: true},
			Accessibility: &BlockSceneAccessibilitySpecReport{Role: graphNode.AccessibilityRole, LabelLen: 4, ReadingIndex: graphNode.ID},
		})
	}
	blockSceneHash := "sha256:" + strings.Repeat("c", 64)
	report.BlockSceneSnapshot = &BlockSceneSnapshotReport{
		Schema:               "tetra.surface.block-scene-snapshot.v1",
		Source:               report.Source,
		SurfaceScope:         "surface-morph-rendered-beauty-linux-web",
		Producer:             "surface-runtime-smoke",
		QualityLevel:         "rich-renderable-block-scene-v1",
		CorePrimitives:       []string{"Block"},
		CompactPropsOnly:     false,
		RecipeExpansionCount: len(nodes),
		NodeCount:            len(nodes),
		RichSpecHash:         "sha256:" + strings.Repeat("b", 64),
		BlockSceneHash:       blockSceneHash,
		SpecCoverage: BlockSceneSpecCoverageReport{
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
	frame := report.Frames[len(report.Frames)-1]
	report.RenderCommandStream = &RenderCommandStreamReport{
		Schema:                        "tetra.surface.render-command-stream.v1",
		Source:                        report.Source,
		SurfaceScope:                  "surface-morph-rendered-beauty-linux-web",
		Producer:                      "surface-runtime-smoke",
		QualityLevel:                  "deterministic-render-command-stream-v1",
		Renderer:                      "software-rgba-headless",
		DerivedFromBlockSceneSnapshot: true,
		BlockSceneHash:                blockSceneHash,
		FrameChecksum:                 frame.Checksum,
		CommandStreamHash:             "sha256:" + strings.Repeat("d", 64),
		CommandCount:                  10,
		SourceLinked:                  true,
		HandcraftedFixture:            false,
		Commands: []RenderCommandReport{
			productVisualRenderCommandForTest(1, "fill"),
			productVisualRenderCommandForTest(2, "gradient"),
			productVisualRenderCommandForTest(3, "image_fill"),
			productVisualRenderCommandForTest(4, "border"),
			productVisualRenderCommandForTest(5, "radius_clip"),
			productVisualRenderCommandForTest(6, "shadow"),
			productVisualRenderCommandForTest(7, "overlay"),
			productVisualRenderCommandForTest(8, "outline"),
			productVisualRenderCommandForTest(9, "text"),
			productVisualRenderCommandForTest(10, "icon"),
		},
	}
}

func productVisualRenderCommandForTest(order int, command string) RenderCommandReport {
	item := RenderCommandReport{
		Order:        order,
		Command:      command,
		Source:       "examples/surface_block_system.tetra",
		SourceNodeID: "block:2",
		Recipe:       "morph.product_visual@1",
		LayerID:      command + "-layer",
		BlockID:      2,
		Rect:         RectReport{X: 12, Y: 12, W: 296, H: 176},
		Clip:         RectReport{X: 12, Y: 12, W: 296, H: 176},
		Radius:       8,
		Opacity:      255,
		Quality:      "source-linked-block-render-command-v1",
		AssetID:      "search-icon",
		TextLen:      12,
		Checksum:     "sha256:" + strings.Repeat("e", 64),
	}
	if command != "radius_clip" {
		item.Color = productVisualRenderCommandColorForTest(command)
	}
	if command == "border" || command == "outline" {
		item.Width = 1
	}
	if command == "shadow" {
		item.Blur = 8
		item.OffsetY = 2
	}
	if command == "text" {
		item.RasterFormat = "builtin-5x7-alpha-mask-v1"
		item.RasterHash = "sha256:" + strings.Repeat("f", 64)
		item.RasterWidth = 296
		item.RasterHeight = 176
		item.RasterCoverage = 204
	}
	if command == "icon" {
		item.RasterFormat = "builtin-icon-mask-raster-v1"
		item.RasterHash = "sha256:" + strings.Repeat("a", 64)
		item.RasterWidth = 296
		item.RasterHeight = 176
		item.RasterCoverage = 1024
	}
	return item
}

func productVisualRenderCommandColorForTest(command string) string {
	switch command {
	case "fill":
		return "#202733ff"
	case "gradient":
		return "#2c3848ff"
	case "image_fill":
		return "#ffffff22"
	case "shadow":
		return "#00000040"
	case "overlay":
		return "#10182066"
	default:
		return "#6eaef4ff"
	}
}
