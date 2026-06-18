package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestValidateMorphRenderedBeautyContractAcceptsValidContract(t *testing.T) {
	contract := validMorphRenderedBeautyContractFixture()
	if err := validateMorphRenderedBeautyContractValue(contract); err != nil {
		t.Fatalf("validateMorphRenderedBeautyContractValue failed: %v", err)
	}
}

func TestValidateMorphRenderedBeautyContractRejectsCoreButtonPrimitive(t *testing.T) {
	contract := validMorphRenderedBeautyContractFixture()
	contract.CorePrimitives = []string{"Block", "Button"}

	err := validateMorphRenderedBeautyContractValue(contract)
	if err == nil {
		t.Fatalf("expected Button core primitive rejection")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "button") {
		t.Fatalf("error = %v, want Button diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyContractRejectsMissingSelfGoldenGuard(t *testing.T) {
	contract := validMorphRenderedBeautyContractFixture()
	contract.NegativeGuards.SelfGoldenRejected = false

	err := validateMorphRenderedBeautyContractValue(contract)
	if err == nil {
		t.Fatalf("expected missing self_golden_rejected guard to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "self_golden") {
		t.Fatalf("error = %v, want self_golden diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportAcceptsValidReport(t *testing.T) {
	report := validMorphRenderedBeautyReportFixture()
	if err := validateMorphRenderedBeautyReportValue(report); err != nil {
		t.Fatalf("validateMorphRenderedBeautyReportValue failed: %v", err)
	}
}

func TestWriteMorphToPixelsChainFile(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "morph-rendered-beauty.json")
	chainPath := filepath.Join(dir, "morph-to-pixels.json")
	report := validMorphRenderedBeautyReportFixture()

	if err := writeMorphToPixelsChainFile(reportPath, chainPath, report); err != nil {
		t.Fatalf("writeMorphToPixelsChainFile failed: %v", err)
	}
	raw, err := os.ReadFile(chainPath)
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}
	var chain surface.MorphToPixelsChainReport
	if err := json.Unmarshal(raw, &chain); err != nil {
		t.Fatalf("decode chain: %v\n%s", err, raw)
	}
	if err := surface.ValidateMorphToPixelsChainReport(
		chain,
		report.MorphEvidence.Source,
	); err != nil {
		t.Fatalf("ValidateMorphToPixelsChainReport failed: %v\n%s", err, raw)
	}
	if chain.ReportPath != filepath.ToSlash(reportPath) ||
		chain.Source != report.MorphEvidence.Source ||
		!chain.Pass {
		t.Fatalf("chain = %#v, want report path, source, and pass", chain)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsMissingRenderCommandStream(t *testing.T) {
	report := validMorphRenderedBeautyReportFixture()
	report.RenderCommandStream = morphRenderedBeautyRenderCommandStream{}

	err := validateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected missing render command stream to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "render_command_stream") {
		t.Fatalf("error = %v, want render_command_stream diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsUnlinkedRenderCommandStream(t *testing.T) {
	report := validMorphRenderedBeautyReportFixture()
	report.RenderCommandStream.SourceLinked = false
	report.RenderCommandStream.HandcraftedFixture = true

	err := validateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected unlinked handcrafted render command stream to fail")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "source_linked") || !strings.Contains(lower, "handcrafted") {
		t.Fatalf("error = %v, want source_linked and handcrafted diagnostics", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsRenderCommandStreamHashMismatch(t *testing.T) {
	report := validMorphRenderedBeautyReportFixture()
	report.RenderCommandStream.CommandStreamHash = "sha256:" + strings.Repeat("e", 64)

	err := validateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected render command stream hash mismatch to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "command_stream_hash") {
		t.Fatalf("error = %v, want command_stream_hash diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsMarkerOnlyTextIconRaster(t *testing.T) {
	report := validMorphRenderedBeautyReportFixture()
	report.RenderCommandStream.Commands[8].MarkerOnly = true
	report.RenderCommandStream.Commands[8].RasterHash = ""
	report.RenderCommandStream.Commands[9].MarkerOnly = true
	report.RenderCommandStream.Commands[9].RasterHash = ""

	err := validateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected marker-only text/icon raster stream to fail")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "marker") || !strings.Contains(lower, "raster") {
		t.Fatalf("error = %v, want marker and raster diagnostics", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsSelfGolden(t *testing.T) {
	report := validMorphRenderedBeautyReportFixture()
	report.PixelEvidence.GoldenArtifactSHA256 = report.PixelEvidence.FrameArtifactSHA256
	report.PixelEvidence.GoldenChecksum = report.PixelEvidence.FrameChecksum

	err := validateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected self-golden report to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "self-golden") {
		t.Fatalf("error = %v, want self-golden diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsMetadataOnlyPixels(t *testing.T) {
	report := validMorphRenderedBeautyReportFixture()
	report.PixelEvidence.FrameArtifact = ""
	report.PixelEvidence.GoldenArtifact = ""

	err := validateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected metadata-only pixel evidence to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "artifact") {
		t.Fatalf("error = %v, want artifact diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsPrecomputedProductFrame(t *testing.T) {
	report := validMorphRenderedBeautyReportFixture()
	report.PixelEvidence.PrecomputedFixtureFrame = true

	err := validateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected precomputed fixture frame to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "precomputed") {
		t.Fatalf("error = %v, want precomputed diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsSyntheticFrameArtifactWithoutPrecomputedFlag(
	t *testing.T,
) {
	report := validMorphRenderedBeautyReportFixture()
	report.PixelEvidence.FrameArtifact = "fixtures/precomputed/surface-block-system-frame.rgba"
	report.PixelEvidence.PrecomputedFixtureFrame = false

	err := validateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected synthetic frame artifact to fail")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "fixture") && !strings.Contains(lower, "precomputed") {
		t.Fatalf("error = %v, want fixture/precomputed diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsPixelEvidenceWithoutSourceLinks(t *testing.T) {
	report := validMorphRenderedBeautyReportFixture()
	report.PixelEvidence.FrameProducer = ""
	report.PixelEvidence.AppSource = ""
	report.PixelEvidence.MorphRecipeHash = ""
	report.PixelEvidence.BlockSceneHash = ""
	report.PixelEvidence.RenderCommandStreamHash = ""

	err := validateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected unlinked pixel evidence to fail")
	}
	for _, want := range []string{
		"frame_producer",
		"app_source",
		"morph_recipe_hash",
		"block_scene_hash",
		"render_command_stream_hash",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %s diagnostic", err, want)
		}
	}
}

func TestValidateMorphRenderedBeautyReportRejectsPixelEvidenceHashMismatch(t *testing.T) {
	report := validMorphRenderedBeautyReportFixture()
	report.PixelEvidence.FrameChecksum = "sha256:" + strings.Repeat("d", 64)
	report.PixelEvidence.BlockSceneHash = "sha256:" + strings.Repeat("e", 64)
	report.PixelEvidence.RenderCommandStreamHash = "sha256:" + strings.Repeat("f", 64)

	err := validateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected mismatched pixel evidence hashes to fail")
	}
	for _, want := range []string{"frame_checksum", "block_scene_hash", "render_command_stream_hash"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %s diagnostic", err, want)
		}
	}
}

func TestValidateMorphRenderedBeautyReportRejectsDOMRuntime(t *testing.T) {
	report := validMorphRenderedBeautyReportFixture()
	report.NegativeGuards.NoDOMUI = false

	err := validateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected DOM runtime guard failure")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "dom") {
		t.Fatalf("error = %v, want DOM diagnostic", err)
	}
}

func validMorphRenderedBeautyContractFixture() morphRenderedBeautyContract {
	return morphRenderedBeautyContract{
		Schema:       "tetra.surface.morph-rendered-beauty.contract.v1",
		Status:       "experimental-contract",
		ReportSchema: "tetra.surface.morph-rendered-beauty.v1",
		SurfaceScope: "surface-morph-rendered-beauty-linux-web",
		Pipeline: []string{
			"morph_source",
			"token_graph",
			"recipe_expansions",
			"resolved_morph_scene",
			"block_scene_snapshot",
			"render_command_stream",
			"frame_artifact",
			"pixel_golden_comparison",
			"product_claim_gate",
		},
		CorePrimitives: []string{"Block"},
		ForbiddenCorePrimitives: []string{
			"Button",
			"Card",
			"TextField",
			"TextBox",
			"Sidebar",
			"Modal",
		},
		SupportedTargets: []string{
			"headless",
			"linux-x64-real-window",
			"wasm32-web-browser-canvas",
		},
		UnsupportedTargets: []string{"macos", "windows", "wasm32-wasi"},
		RequiredEvidence: []string{
			"morph_source_hash",
			"token_graph_hash",
			"token_coverage",
			"recipe_coverage",
			"recipe_expansions",
			"resolved_morph_scene_hash",
			"block_scene_snapshot_hash",
			"block_scene_snapshot_rich_specs",
			"render_command_stream_hash",
			"source_linked_render_command_stream",
			"text_icon_raster_evidence",
			"app_produced_frame",
			"morph_recipe_hash",
			"pixel_block_scene_hash",
			"pixel_render_command_stream_hash",
			"frame_artifact_sha256",
			"golden_artifact_sha256",
			"pixel_diff_metrics",
			"renderer_owned_stable_proof",
			"target_and_scenario_name",
			"same_commit_git_head",
			"same_commit_git_commit",
		},
		NegativeGuards: morphRenderedBeautyNegativeGuards{
			MetadataOnlyRejected:             true,
			SelfGoldenRejected:               true,
			PrecomputedFrameRejected:         true,
			MissingFrameArtifactRejected:     true,
			NoDOMUI:                          true,
			NoCSSRuntime:                     true,
			NoReactRuntime:                   true,
			NoElectronRuntime:                true,
			NoNativeWidgets:                  true,
			NoHiddenAppState:                 true,
			NonBlockOutputRejected:           true,
			DirtyCheckoutProductionRejected:  true,
			UnsupportedTargetRejected:        true,
			RendererOwnedStableProofRequired: true,
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

func validMorphRenderedBeautyReportFixture() morphRenderedBeautyReport {
	return morphRenderedBeautyReport{
		Schema:         "tetra.surface.morph-rendered-beauty.v1",
		Status:         "pass",
		SurfaceScope:   "surface-morph-rendered-beauty-linux-web",
		Target:         "headless",
		ScenarioName:   "surface-morph-rendered-studio-shell",
		GitHead:        "95bfd4a887bab5032437cb22494d034e82ae6d35",
		GitCommit:      "95bfd4a887bab5032437cb22494d034e82ae6d35",
		GitDirty:       false,
		ProductClaim:   false,
		FinalSignoff:   false,
		CorePrimitives: []string{"Block"},
		MorphEvidence: morphRenderedBeautyMorphEvidence{
			Source:         "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra",
			SourceSHA256:   "sha256:" + strings.Repeat("1", 64),
			CapsuleHash:    "sha256:" + strings.Repeat("2", 64),
			TokenGraphHash: "sha256:" + strings.Repeat("3", 64),
			TokenCount:     22,
			TokenCategories: []string{
				"color",
				"space",
				"radius",
				"typography",
				"motion",
				"assets",
			},
			RecipeCount:          16,
			RecipeExpansionCount: 16,
			RecipeNames: []string{
				"control.action@1",
				"field.text@1",
				"command.item@1",
				"region.panel@1",
				"form.field@1",
				"nav.item@1",
				"metric.tile@1",
				"dialog.panel@1",
				"toast.notification@1",
				"tab.item@1",
				"list.row@1",
				"app.shell@1",
				"toolbar@1",
				"split.pane@1",
				"status.bar@1",
				"settings.form@1",
			},
			ResolvedMorphSceneHash: "sha256:" + strings.Repeat("4", 64),
			BlockSceneSnapshotHash: "sha256:" + strings.Repeat("5", 64),
		},
		BlockSceneSnapshot: morphRenderedBeautyBlockSceneSnapshot{
			Schema:       "tetra.surface.block-scene-snapshot.v1",
			SurfaceScope: "surface-morph-rendered-beauty-linux-web",
			Source: ("examples/surface/morph_flagship/surface_morph_rendered_studio_" +
				"shell.tetra"),
			QualityLevel:         "rich-renderable-block-scene-v1",
			CorePrimitives:       []string{"Block"},
			CompactPropsOnly:     false,
			RecipeExpansionCount: 16,
			NodeCount:            24,
			RichSpecHash:         "sha256:" + strings.Repeat("b", 64),
			BlockSceneHash:       "sha256:" + strings.Repeat("5", 64),
			SpecCoverage: morphRenderedBeautyBlockSceneSpecCoverage{
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
		},
		RenderEvidence: morphRenderedBeautyRenderEvidence{
			CommandStreamHash: "sha256:" + strings.Repeat("6", 64),
			CommandCount:      10,
			Renderer:          "software-rgba-headless",
		},
		RendererStableProof: morphRenderedBeautyRendererStableProof{
			Schema:                         "tetra.surface.renderer-stable-proof.v1",
			PixelOwner:                     "surface-renderer",
			RendererOwned:                  true,
			BridgeOwnedPixels:              false,
			BlockFirst:                     true,
			DerivedFromRenderCommandStream: true,
			RenderCommandStreamHash:        "sha256:" + strings.Repeat("6", 64),
			BlockSceneHash:                 "sha256:" + strings.Repeat("5", 64),
			FrameChecksum:                  "sha256:" + strings.Repeat("8", 64),
			StablePromotionEligible:        true,
		},
		RenderCommandStream: morphRenderedBeautyRenderCommandStream{
			Schema: "tetra.surface.render-command-stream.v1",
			Source: ("examples/surface/morph_flagship/surface_morph_" +
				"rendered_studio_shell.tetra"),
			SurfaceScope:                  "surface-morph-rendered-beauty-linux-web",
			Producer:                      "surface-runtime-smoke",
			QualityLevel:                  "deterministic-render-command-stream-v1",
			Renderer:                      "software-rgba-headless",
			DerivedFromBlockSceneSnapshot: true,
			BlockSceneHash:                "sha256:" + strings.Repeat("5", 64),
			FrameChecksum:                 "sha256:" + strings.Repeat("8", 64),
			CommandStreamHash:             "sha256:" + strings.Repeat("6", 64),
			CommandCount:                  10,
			SourceLinked:                  true,
			HandcraftedFixture:            false,
			Commands: []morphRenderedBeautyRenderCommand{
				morphRenderedBeautyRenderCommandForTest(1, "fill"),
				morphRenderedBeautyRenderCommandForTest(2, "gradient"),
				morphRenderedBeautyRenderCommandForTest(3, "image_fill"),
				morphRenderedBeautyRenderCommandForTest(4, "border"),
				morphRenderedBeautyRenderCommandForTest(5, "radius_clip"),
				morphRenderedBeautyRenderCommandForTest(6, "shadow"),
				morphRenderedBeautyRenderCommandForTest(7, "overlay"),
				morphRenderedBeautyRenderCommandForTest(8, "outline"),
				morphRenderedBeautyRenderCommandForTest(9, "text"),
				morphRenderedBeautyRenderCommandForTest(10, "icon"),
			},
		},
		PixelEvidence: morphRenderedBeautyPixelEvidence{
			FrameArtifact:       "frames/studio-shell-headless.rgba",
			FrameArtifactSHA256: "sha256:" + strings.Repeat("7", 64),
			FrameChecksum:       "sha256:" + strings.Repeat("8", 64),
			FrameProducer:       "app",
			AppSource: ("examples/surface/morph_flagship/surface_morph_rendered_" +
				"studio_shell.tetra"),
			MorphRecipeHash:         "sha256:" + strings.Repeat("c", 64),
			BlockSceneHash:          "sha256:" + strings.Repeat("5", 64),
			RenderCommandStreamHash: "sha256:" + strings.Repeat("6", 64),
			GoldenArtifact:          "goldens/studio-shell-headless.rgba",
			GoldenArtifactSHA256:    "sha256:" + strings.Repeat("9", 64),
			GoldenChecksum:          "sha256:" + strings.Repeat("a", 64),
			DiffPixels:              0,
			DiffRatioMilli:          0,
			MaxChannelDelta:         0,
			PrecomputedFixtureFrame: false,
		},
		NegativeGuards: validMorphRenderedBeautyContractFixture().NegativeGuards,
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

func morphRenderedBeautyRenderCommandForTest(
	order int,
	command string,
) morphRenderedBeautyRenderCommand {
	item := morphRenderedBeautyRenderCommand{
		Order:        order,
		Command:      command,
		Source:       "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra",
		SourceNodeID: "block:2",
		Recipe:       "morph.search_input",
		LayerID:      "block-2-layer-" + command,
		BlockID:      2,
		Quality:      "source-linked-block-render-command-v1",
		Checksum:     morphRenderedBeautyChecksumForOrder(order),
	}
	if command != "radius_clip" {
		item.Color = morphRenderedBeautyCommandColorForTest(command)
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
		item.RasterHash = "sha256:" + strings.Repeat("b", 64)
		item.RasterWidth = 288
		item.RasterHeight = 168
		item.RasterCoverage = 204
		item.MarkerOnly = false
	}
	if command == "icon" {
		item.RasterFormat = "builtin-icon-mask-raster-v1"
		item.RasterHash = "sha256:" + strings.Repeat("c", 64)
		item.RasterWidth = 288
		item.RasterHeight = 168
		item.RasterCoverage = 16128
		item.MarkerOnly = false
	}
	return item
}

func morphRenderedBeautyCommandColorForTest(command string) string {
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

func morphRenderedBeautyChecksumForOrder(order int) string {
	digits := "abcdef"
	index := order % len(digits)
	return "sha256:" + strings.Repeat(digits[index:index+1], 64)
}
