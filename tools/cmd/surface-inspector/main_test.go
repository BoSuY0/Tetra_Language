package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestRunWritesInspectorReportFromRuntimeReports(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "inputs")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatalf("mkdir inputs: %v", err)
	}
	blockPath := filepath.Join(inputDir, "block.json")
	morphPath := filepath.Join(inputDir, "morph.json")
	appPath := filepath.Join(inputDir, "app-model.json")
	a11yPath := filepath.Join(inputDir, "accessibility.json")
	morphRenderedBeautyPath := filepath.Join(inputDir, "morph-rendered-beauty.json")
	for path, raw := range map[string]string{
		blockPath: minimalInspectorInputBlockJSON(),
		morphPath: minimalInspectorInputMorphJSON(),
		appPath:   minimalInspectorInputAppModelJSON(),
		a11yPath:  minimalInspectorInputAccessibilityJSON(),
	} {
		if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeInspectorMorphRenderedBeautyReport(
		t,
		morphRenderedBeautyPath,
		"examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra",
	)

	reportPath := filepath.Join(dir, "surface-inspector.json")
	htmlPath := filepath.Join(dir, "surface-inspector.html")
	if err := run([]string{
		"--runtime-report", "block:" + blockPath,
		"--runtime-report", "morph:" + morphPath,
		"--runtime-report", "morph-rendered-beauty:" + morphRenderedBeautyPath,
		"--runtime-report", "app-model:" + appPath,
		"--runtime-report", "accessibility:" + a11yPath,
		"--out", reportPath,
		"--html", htmlPath,
	}); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if err := surface.ValidateInspectorReport(raw); err != nil {
		t.Fatalf("ValidateInspectorReport failed: %v\n%s", err, raw)
	}
	if _, err := os.Stat(htmlPath); err != nil {
		t.Fatalf("expected HTML tool report: %v", err)
	}
	var report struct {
		Sections map[string]struct {
			Present bool `json:"present"`
			Count   int  `json:"count"`
		} `json:"sections"`
		MorphToPixels struct {
			ChainID                 string `json:"chain_id"`
			Source                  string `json:"source"`
			RecipeExpansionCount    int    `json:"recipe_expansion_count"`
			BlockSceneNodeCount     int    `json:"block_scene_node_count"`
			RenderCommandStreamHash string `json:"render_command_stream_hash"`
			RenderCommandCount      int    `json:"render_command_count"`
			FrameArtifact           string `json:"frame_artifact"`
			GoldenArtifact          string `json:"golden_artifact"`
			Pass                    bool   `json:"pass"`
		} `json:"morph_to_pixels"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	for _, want := range []string{
		"block_tree",
		"morph_tokens",
		"layout",
		"paint",
		"accessibility",
		"event_routes",
		"focus",
		"perf_counters",
		"recipe_expansions",
		"block_scene_nodes",
		"render_commands",
		"frame_artifacts",
		"golden_diff",
	} {
		got := report.Sections[want]
		if !got.Present || got.Count == 0 {
			t.Fatalf("section %s = %#v, want present with count", want, got)
		}
	}
	if !report.MorphToPixels.Pass ||
		report.MorphToPixels.ChainID == "" ||
		report.MorphToPixels.Source != ("examples/surface/morph_flagship/surface_morph_rendered_"+
			"studio_shell.tetra") ||
		report.MorphToPixels.RecipeExpansionCount == 0 ||
		report.MorphToPixels.BlockSceneNodeCount == 0 ||
		report.MorphToPixels.RenderCommandCount == 0 ||
		report.MorphToPixels.RenderCommandStreamHash == "" ||
		report.MorphToPixels.FrameArtifact == "" ||
		report.MorphToPixels.GoldenArtifact == "" {
		t.Fatalf("morph_to_pixels = %#v, want Morph-to-pixels chain summary", report.MorphToPixels)
	}
}

func TestRunRejectsHiddenStateInInputReports(t *testing.T) {
	dir := t.TempDir()
	blockPath := filepath.Join(dir, "block.json")
	if err := os.WriteFile(
		blockPath,
		[]byte(`{"schema":"tetra.surface.runtime.v1","status":"pass","target":"headless","source":"examples/surface/block_core/surface_block_system.tetra","hidden_state":true}`),
		0o644,
	); err != nil {
		t.Fatalf("write block report: %v", err)
	}
	err := run(
		[]string{
			"--runtime-report",
			"block:" + blockPath,
			"--out",
			filepath.Join(dir, "surface-inspector.json"),
		},
	)
	if err == nil {
		t.Fatalf("expected hidden state input to fail")
	}
}

func minimalInspectorInputBlockJSON() string {
	return `{"schema":"tetra.surface.runtime.v1","status":"pass","target":"headless","source":"examples/surface/block_core/surface_block_system.tetra","block_graph":{"nodes":[{"id":1},{"id":2}]},"layout_passes":[{"order":1},{"order":2}],"paint_commands":[{"order":1},{"order":2}],"block_event_routes":[{"order":1},{"order":2}],"block_focus_transitions":[{"order":1}],"block_accessibility_tree":{"nodes":[{"id":1},{"id":2}]},"surface_performance_budget":{"schema":"tetra.surface.performance-budget.v1","model":"surface-performance-budget-v1"}}`
}

func minimalInspectorInputMorphJSON() string {
	return `{"schema":"tetra.surface.runtime.v1","status":"pass","target":"headless","source":"examples/surface/morph_core/surface_morph_command_palette.tetra","morph":{"schema":"tetra.surface.morph.v1","token_graph":{"tokens":[{"name":"color.accent"},{"name":"space.2"}]},"recipes":[{"name":"panel"}]}}`
}

func minimalInspectorInputAppModelJSON() string {
	return `{"schema":"tetra.surface.runtime.v1","status":"pass","target":"headless","source":"examples/surface/toolkit/surface_app_model.tetra","app_model":{"schema":"tetra.surface.app-model.v1","event_bindings":[{"event":"key"}],"focus_scopes":[{"id":"modal"}],"async_tasks":[{"name":"load"}]}}`
}

func minimalInspectorInputAccessibilityJSON() string {
	return `{"schema":"tetra.surface.runtime.v1","status":"pass","target":"headless","source":"examples/surface/release/surface_release_accessibility.tetra","accessibility_tree":{"schema":"tetra.surface.accessibility-tree.v1","nodes":[{"id":1},{"id":2}],"snapshots":[{"name":"initial"}]}}`
}

func writeInspectorMorphRenderedBeautyReport(t *testing.T, path string, source string) {
	t.Helper()
	report := validInspectorMorphRenderedBeautyReport(source)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal Morph rendered beauty report: %v", err)
	}
	if err := surface.ValidateMorphRenderedBeautyReport(raw); err != nil {
		t.Fatalf("test Morph rendered beauty report invalid: %v\n%s", err, raw)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write Morph rendered beauty report: %v", err)
	}
}

func validInspectorMorphRenderedBeautyReport(source string) surface.MorphRenderedBeautyReport {
	blockSceneHash := inspectorTestSHA(5)
	commandStreamHash := inspectorTestSHA(7)
	frameHash := inspectorTestSHA(60)
	goldenHash := inspectorTestSHA(61)
	commands := []string{
		"fill",
		"gradient",
		"image_fill",
		"border",
		"radius_clip",
		"shadow",
		"overlay",
		"outline",
		"text",
		"icon",
	}
	renderCommands := make([]surface.MorphRenderedBeautyRenderCommand, 0, len(commands))
	for i, command := range commands {
		item := surface.MorphRenderedBeautyRenderCommand{
			Order:        i + 1,
			Command:      command,
			Source:       source,
			SourceNodeID: fmt.Sprintf("node-%d", i+1),
			Recipe:       "studio_shell",
			LayerID:      "layer-main",
			BlockID:      i + 1,
			Quality:      "deterministic",
			Checksum:     inspectorTestSHA(100 + i),
		}
		if command != "radius_clip" {
			item.Color = inspectorMorphRenderedBeautyCommandColor(command)
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
			item.RasterHash = inspectorTestSHA(210)
			item.RasterWidth = 5
			item.RasterHeight = 7
			item.RasterCoverage = 20
		}
		if command == "icon" {
			item.RasterFormat = "builtin-icon-mask-raster-v1"
			item.RasterHash = inspectorTestSHA(211)
			item.RasterWidth = 16
			item.RasterHeight = 16
			item.RasterCoverage = 96
		}
		renderCommands = append(renderCommands, item)
	}
	return surface.MorphRenderedBeautyReport{
		Schema:         surface.MorphRenderedBeautyReportSchemaV1,
		Status:         "pass",
		SurfaceScope:   surface.MorphRenderedBeautyScope,
		Target:         "headless",
		ScenarioName:   "headless-morph:" + source,
		GitHead:        strings.Repeat("1", 40),
		GitCommit:      strings.Repeat("1", 40),
		CorePrimitives: []string{"Block"},
		MorphEvidence: surface.MorphRenderedBeautyMorphEvidence{
			Source:         source,
			SourceSHA256:   inspectorTestSHA(1),
			CapsuleHash:    inspectorTestSHA(2),
			TokenGraphHash: inspectorTestSHA(3),
			TokenCount:     6,
			TokenCategories: []string{
				"color",
				"space",
				"radius",
				"typography",
				"motion",
				"assets",
			},
			RecipeCount:            3,
			RecipeExpansionCount:   4,
			RecipeNames:            []string{"studio_shell", "hero_panel", "toolbar"},
			ResolvedMorphSceneHash: inspectorTestSHA(4),
			BlockSceneSnapshotHash: blockSceneHash,
		},
		BlockSceneSnapshot: surface.MorphRenderedBeautyBlockSceneSnapshot{
			Schema:               "tetra.surface.block-scene-snapshot.v1",
			SurfaceScope:         surface.MorphRenderedBeautyScope,
			Source:               source,
			QualityLevel:         "rich-renderable-block-scene-v1",
			CorePrimitives:       []string{"Block"},
			RecipeExpansionCount: 4,
			NodeCount:            12,
			RichSpecHash:         inspectorTestSHA(6),
			BlockSceneHash:       blockSceneHash,
			SpecCoverage: surface.MorphRenderedBeautyBlockSceneSpecCoverage{
				Layout: true, Paint: true, Text: true, Image: true, Input: true, Event: true, State: true, Motion: true, Accessibility: true,
			},
		},
		RenderEvidence: surface.MorphRenderedBeautyRenderEvidence{
			CommandStreamHash: commandStreamHash,
			CommandCount:      len(renderCommands),
			Renderer:          "software-rgba-headless",
		},
		RendererStableProof: surface.MorphRenderedBeautyRendererStableProof{
			Schema:                         "tetra.surface.renderer-stable-proof.v1",
			PixelOwner:                     "surface-renderer",
			RendererOwned:                  true,
			BridgeOwnedPixels:              false,
			BlockFirst:                     true,
			DerivedFromRenderCommandStream: true,
			RenderCommandStreamHash:        commandStreamHash,
			BlockSceneHash:                 blockSceneHash,
			FrameChecksum:                  frameHash,
			StablePromotionEligible:        true,
		},
		RenderCommandStream: surface.MorphRenderedBeautyRenderCommandStream{
			Schema:                        "tetra.surface.render-command-stream.v1",
			Source:                        source,
			SurfaceScope:                  surface.MorphRenderedBeautyScope,
			Producer:                      "surface-runtime-smoke",
			QualityLevel:                  "deterministic-render-command-stream-v1",
			Renderer:                      "software-rgba-headless",
			DerivedFromBlockSceneSnapshot: true,
			BlockSceneHash:                blockSceneHash,
			FrameChecksum:                 frameHash,
			CommandStreamHash:             commandStreamHash,
			CommandCount:                  len(renderCommands),
			SourceLinked:                  true,
			Commands:                      renderCommands,
		},
		PixelEvidence: surface.MorphRenderedBeautyPixelEvidence{
			FrameArtifact:           "reports/surface/inspector-frame.rgba",
			FrameArtifactSHA256:     frameHash,
			FrameChecksum:           frameHash,
			FrameProducer:           "app",
			AppSource:               source,
			MorphRecipeHash:         inspectorTestSHA(8),
			BlockSceneHash:          blockSceneHash,
			RenderCommandStreamHash: commandStreamHash,
			GoldenArtifact:          "reports/surface/inspector-golden.rgba",
			GoldenArtifactSHA256:    goldenHash,
			GoldenChecksum:          goldenHash,
			DiffPixels:              1,
			MaxChannelDelta:         1,
		},
		NegativeGuards: surface.MorphRenderedBeautyNegativeGuards{
			MetadataOnlyRejected: true, SelfGoldenRejected: true, PrecomputedFrameRejected: true, MissingFrameArtifactRejected: true,
			NoDOMUI: true, NoCSSRuntime: true, NoReactRuntime: true, NoElectronRuntime: true, NoNativeWidgets: true, NoHiddenAppState: true,
			NonBlockOutputRejected: true, DirtyCheckoutProductionRejected: true, UnsupportedTargetRejected: true, RendererOwnedStableProofRequired: true,
		},
		NonClaims: []string{
			"no Electron runtime claim",
			"no React runtime claim",
			"no CSS runtime claim",
			"no DOM-authored UI claim",
			"no GPU renderer production claim",
			"no macOS production claim",
			"no Windows production claim",
		},
	}
}

func inspectorMorphRenderedBeautyCommandColor(command string) string {
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

func inspectorTestSHA(seed int) string {
	return "sha256:" + fmt.Sprintf("%064x", seed)
}
