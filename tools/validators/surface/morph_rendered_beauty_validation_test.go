package surface

import (
	"strings"
	"testing"
)

func TestValidateMorphRenderedBeautyReportAcceptsFirstClassReport(t *testing.T) {
	report := validMorphRenderedBeautySurfaceReportFixture()

	if err := ValidateMorphRenderedBeautyReportValue(report); err != nil {
		t.Fatalf("ValidateMorphRenderedBeautyReportValue failed: %v", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsMissingScenarioName(t *testing.T) {
	report := validMorphRenderedBeautySurfaceReportFixture()
	report.ScenarioName = ""

	err := ValidateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected missing scenario_name to fail")
	}
	if !strings.Contains(err.Error(), "scenario_name") {
		t.Fatalf("error = %v, want scenario_name diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsMissingPixelLink(t *testing.T) {
	report := validMorphRenderedBeautySurfaceReportFixture()
	report.PixelEvidence.RenderCommandStreamHash = ""

	err := ValidateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected missing render command stream pixel link to fail")
	}
	if !strings.Contains(err.Error(), "pixel_evidence.render_command_stream_hash") {
		t.Fatalf("error = %v, want pixel render_command_stream_hash diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsMissingGitCommitAlias(t *testing.T) {
	report := validMorphRenderedBeautySurfaceReportFixture()
	report.GitCommit = ""

	err := ValidateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected missing git_commit to fail")
	}
	if !strings.Contains(err.Error(), "git_commit") {
		t.Fatalf("error = %v, want git_commit diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsMismatchedGitCommit(t *testing.T) {
	report := validMorphRenderedBeautySurfaceReportFixture()
	report.GitCommit = strings.Repeat("a", 40)

	err := ValidateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected mismatched git_commit to fail")
	}
	if !strings.Contains(err.Error(), "git_commit must match git_head") {
		t.Fatalf("error = %v, want git_commit mismatch diagnostic", err)
	}
}

func TestValidateMorphRenderedBeautyReportRejectsProductClaimWithoutRendererOwnedStableProof(t *testing.T) {
	report := validMorphRenderedBeautySurfaceReportFixture()
	report.ProductClaim = true
	report.RendererStableProof.RendererOwned = false
	report.RendererStableProof.PixelOwner = "morph-evidence-bridge"

	err := ValidateMorphRenderedBeautyReportValue(report)
	if err == nil {
		t.Fatalf("expected product_claim without renderer-owned stable proof to fail")
	}
	if !strings.Contains(err.Error(), "renderer_owned stable proof") {
		t.Fatalf("error = %v, want renderer-owned stable proof diagnostic", err)
	}
}

func validMorphRenderedBeautySurfaceReportFixture() MorphRenderedBeautyReport {
	source := "examples/surface_morph_command_palette.tetra"
	blockHash := "sha256:" + strings.Repeat("5", 64)
	streamHash := "sha256:" + strings.Repeat("6", 64)
	frameChecksum := "sha256:" + strings.Repeat("8", 64)
	gitCommit := "95bfd4a887bab5032437cb22494d034e82ae6d35"
	return MorphRenderedBeautyReport{
		Schema:         MorphRenderedBeautyReportSchemaV1,
		Status:         "pass",
		SurfaceScope:   MorphRenderedBeautyScope,
		Target:         "headless",
		ScenarioName:   "headless-morph",
		GitHead:        gitCommit,
		GitCommit:      gitCommit,
		CorePrimitives: []string{"Block"},
		MorphEvidence: MorphRenderedBeautyMorphEvidence{
			Source:                 source,
			SourceSHA256:           "sha256:" + strings.Repeat("1", 64),
			CapsuleHash:            "sha256:" + strings.Repeat("2", 64),
			TokenGraphHash:         "sha256:" + strings.Repeat("3", 64),
			TokenCount:             22,
			TokenCategories:        []string{"color", "space", "radius", "typography", "motion", "assets"},
			RecipeCount:            3,
			RecipeExpansionCount:   19,
			RecipeNames:            []string{"control.action@1", "field.text@1", "command.item@1"},
			ResolvedMorphSceneHash: "sha256:" + strings.Repeat("4", 64),
			BlockSceneSnapshotHash: blockHash,
		},
		BlockSceneSnapshot: MorphRenderedBeautyBlockSceneSnapshot{
			Schema:               "tetra.surface.block-scene-snapshot.v1",
			SurfaceScope:         MorphRenderedBeautyScope,
			Source:               source,
			QualityLevel:         "rich-renderable-block-scene-v1",
			CorePrimitives:       []string{"Block"},
			RecipeExpansionCount: 19,
			NodeCount:            5,
			RichSpecHash:         "sha256:" + strings.Repeat("b", 64),
			BlockSceneHash:       blockHash,
			SpecCoverage: MorphRenderedBeautyBlockSceneSpecCoverage{
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
		RenderEvidence: MorphRenderedBeautyRenderEvidence{
			CommandStreamHash: streamHash,
			CommandCount:      10,
			Renderer:          "software-rgba-headless",
		},
		RendererStableProof: MorphRenderedBeautyRendererStableProof{
			Schema:                         "tetra.surface.renderer-stable-proof.v1",
			PixelOwner:                     "surface-renderer",
			RendererOwned:                  true,
			BridgeOwnedPixels:              false,
			BlockFirst:                     true,
			DerivedFromRenderCommandStream: true,
			RenderCommandStreamHash:        streamHash,
			BlockSceneHash:                 blockHash,
			FrameChecksum:                  frameChecksum,
			StablePromotionEligible:        true,
		},
		RenderCommandStream: MorphRenderedBeautyRenderCommandStream{
			Schema:                        "tetra.surface.render-command-stream.v1",
			Source:                        source,
			SurfaceScope:                  MorphRenderedBeautyScope,
			Producer:                      "surface-runtime-smoke",
			QualityLevel:                  "deterministic-render-command-stream-v1",
			Renderer:                      "software-rgba-headless",
			DerivedFromBlockSceneSnapshot: true,
			BlockSceneHash:                blockHash,
			FrameChecksum:                 frameChecksum,
			CommandStreamHash:             streamHash,
			CommandCount:                  10,
			SourceLinked:                  true,
			Commands: []MorphRenderedBeautyRenderCommand{
				morphRenderedBeautySurfaceCommandForTest(source, 1, "fill"),
				morphRenderedBeautySurfaceCommandForTest(source, 2, "gradient"),
				morphRenderedBeautySurfaceCommandForTest(source, 3, "image_fill"),
				morphRenderedBeautySurfaceCommandForTest(source, 4, "border"),
				morphRenderedBeautySurfaceCommandForTest(source, 5, "radius_clip"),
				morphRenderedBeautySurfaceCommandForTest(source, 6, "shadow"),
				morphRenderedBeautySurfaceCommandForTest(source, 7, "overlay"),
				morphRenderedBeautySurfaceCommandForTest(source, 8, "outline"),
				morphRenderedBeautySurfaceCommandForTest(source, 9, "text"),
				morphRenderedBeautySurfaceCommandForTest(source, 10, "icon"),
			},
		},
		PixelEvidence: MorphRenderedBeautyPixelEvidence{
			FrameArtifact:           "reports/surface/morph/headless/current.rgba",
			FrameArtifactSHA256:     "sha256:" + strings.Repeat("7", 64),
			FrameChecksum:           frameChecksum,
			FrameProducer:           "app",
			AppSource:               source,
			MorphRecipeHash:         "sha256:" + strings.Repeat("c", 64),
			BlockSceneHash:          blockHash,
			RenderCommandStreamHash: streamHash,
			GoldenArtifact:          "reports/surface/morph/headless/golden.rgba",
			GoldenArtifactSHA256:    "sha256:" + strings.Repeat("9", 64),
			GoldenChecksum:          "sha256:" + strings.Repeat("a", 64),
		},
		NegativeGuards: MorphRenderedBeautyNegativeGuards{
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

func morphRenderedBeautySurfaceCommandForTest(source string, order int, command string) MorphRenderedBeautyRenderCommand {
	item := MorphRenderedBeautyRenderCommand{
		Order:        order,
		Command:      command,
		Source:       source,
		SourceNodeID: "block:2",
		Recipe:       "morph.recipe",
		LayerID:      "block-2-layer-" + command,
		BlockID:      2,
		Quality:      "source-linked-block-render-command-v1",
		Checksum:     "sha256:" + strings.Repeat("d", 64),
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
		item.RasterHash = "sha256:" + strings.Repeat("e", 64)
		item.RasterWidth = 16
		item.RasterHeight = 16
		item.RasterCoverage = 24
	}
	if command == "icon" {
		item.RasterFormat = "builtin-icon-mask-raster-v1"
		item.RasterHash = "sha256:" + strings.Repeat("f", 64)
		item.RasterWidth = 16
		item.RasterHeight = 16
		item.RasterCoverage = 48
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
