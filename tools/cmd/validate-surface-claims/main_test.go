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

const (
	surfaceClaimTestGitHead      = "0123456789abcdef0123456789abcdef01234567"
	surfaceClaimTestStaleGitHead = "fedcba9876543210fedcba9876543210fedcba98"
)

func TestValidateSurfaceClaimsRejectsFullElectronReplacement(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"docs/spec/core/current_supported_surface.md",
		`# Fake Surface Claim

Surface is a full Electron replacement for production desktop applications.
`,
	)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted a full Electron replacement claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "electron") {
		t.Fatalf("error = %v, want Electron diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsReactAndCSSReplacement(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "docs/user/surface/surface_guide.md", `# Fake Surface Claim

Surface is a React replacement and CSS replacement for production app UI.
`)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted React/CSS replacement claims")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "react") || !strings.Contains(lower, "css") {
		t.Fatalf("error = %v, want React and CSS diagnostics", err)
	}
}

func TestValidateSurfaceClaimsRejectsProductionMorphExperimentalReport(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"reports/surface-morph/headless/surface-headless-morph.json",
		`{
  "schema": "tetra.surface.runtime.v1",
  "target": "headless",
  "claim": "production Morph is ready",
  "experimental": true
}
`,
	)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:       root,
		ReportDirs: []string{filepath.Join(root, "reports")},
	})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted a production Morph claim with experimental=true")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "morph") {
		t.Fatalf("error = %v, want Morph diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsMorphProductionBeautyWithoutMRBEvidence(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"docs/release/surface/surface_v1_release_notes.md",
		`# Fake Surface Claim

Morph production beauty is now guaranteed for Surface.
`,
	)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:    root,
		GitHead: surfaceClaimTestGitHead,
	})
	if err == nil {
		t.Fatalf(
			"validateSurfaceClaims accepted a Morph production beauty claim without MRB evidence",
		)
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "beauty") || !strings.Contains(lower, "morph") {
		t.Fatalf("error = %v, want Morph beauty diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsElectronQualityUIWithoutMRBEvidence(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"docs/user/surface/surface_electron_comparison.md",
		`# Fake Surface Claim

The UI is Electron-quality for the current Surface release.
`,
	)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:    root,
		GitHead: surfaceClaimTestGitHead,
	})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted an Electron-quality UI claim without MRB evidence")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "quality") {
		t.Fatalf("error = %v, want quality diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsReactQualityUIWithoutMRBEvidence(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"docs/user/surface/surface_electron_comparison.md",
		`# Fake Surface Claim

Our React-quality Surface UI is production-grade.
`,
	)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:    root,
		GitHead: surfaceClaimTestGitHead,
	})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted a React-quality UI claim without MRB evidence")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "quality") {
		t.Fatalf("error = %v, want quality diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsPixelPerfectSurfaceWithoutMRBEvidence(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "docs/spec/surface/surface_v1.md", `# Fake Surface Claim

Pixel-perfect Surface rendering is ready.
`)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:    root,
		GitHead: surfaceClaimTestGitHead,
	})
	if err == nil {
		t.Fatalf(
			"validateSurfaceClaims accepted a pixel-perfect Surface claim without MRB evidence",
		)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "quality") {
		t.Fatalf("error = %v, want quality diagnostic", err)
	}
}

func TestValidateSurfaceClaimsAllowsBeautyClaimWithSameCommitMRBEvidence(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"docs/spec/surface/morph/surface_morph_rendered_beauty.md",
		`# Fake Surface Claim

Surface has Morph rendered beauty evidence for the checked report.
`,
	)
	writeSurfaceClaimMRBReport(
		t,
		root,
		"reports/surface-morph-rendered-beauty/morph-rendered-beauty.json",
		surfaceClaimTestGitHead,
		false,
		false,
	)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:       root,
		ReportDirs: []string{filepath.Join(root, "reports")},
		GitHead:    surfaceClaimTestGitHead,
	})
	if err != nil {
		t.Fatalf("validateSurfaceClaims rejected same-commit MRB evidence: %v", err)
	}
}

func TestValidateSurfaceClaimsRejectsBeautyClaimWithStaleMRBEvidence(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"docs/spec/surface/morph/surface_morph_rendered_beauty.md",
		`# Fake Surface Claim

Surface has Morph rendered beauty evidence for the checked report.
`,
	)
	writeSurfaceClaimMRBReport(
		t,
		root,
		"reports/surface-morph-rendered-beauty/morph-rendered-beauty.json",
		surfaceClaimTestStaleGitHead,
		false,
		false,
	)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:       root,
		ReportDirs: []string{filepath.Join(root, "reports")},
		GitHead:    surfaceClaimTestGitHead,
	})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted stale MRB evidence")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "same-commit") {
		t.Fatalf("error = %v, want same-commit diagnostic", err)
	}
}

func TestValidateSurfaceClaimsAllowsProductionReadyMorphWithProductMRBSignoff(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"docs/release/surface/surface_v1_release_notes.md",
		`# Fake Surface Claim

Morph is production-ready for the signed Surface rendered beauty scope.
`,
	)
	writeSurfaceClaimMRBReport(
		t,
		root,
		"reports/surface-morph-rendered-beauty/morph-rendered-beauty.json",
		surfaceClaimTestGitHead,
		true,
		true,
	)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:       root,
		ReportDirs: []string{filepath.Join(root, "reports")},
		GitHead:    surfaceClaimTestGitHead,
	})
	if err != nil {
		t.Fatalf(
			"validateSurfaceClaims rejected production-ready Morph with product MRB signoff: %v",
			err,
		)
	}
}

func TestValidateSurfaceClaimsAllowsProductionWordInMorphArtifactPaths(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		("reports/surface-electron-react-beauty-production/P07/morph-gate/" +
			"headless/surface-headless-morph.json"),
		`{
  "schema": "tetra.surface.runtime.v1",
  "path": "/repo/reports/surface-electron-react-beauty-production/P07/morph-gate/headless/surface-morph-command-palette",
  "root": "/repo/reports/surface-electron-react-beauty-production/P07/morph-gate/headless/surface-headless-morph-artifacts",
  "command_line": "bash scripts/release/surface/morph-gate.sh --report-dir reports/surface-electron-react-beauty-production/P07/morph-gate",
  "morph": {
    "experimental": true,
    "production_claim": false
  }
}
`,
	)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:       root,
		ReportDirs: []string{filepath.Join(root, "reports")},
	})
	if err != nil {
		t.Fatalf(
			"validateSurfaceClaims rejected artifact paths as Morph production claims: %v",
			err,
		)
	}
}

func TestValidateSurfaceClaimsRejectsUnsupportedGPUProductionClaim(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"docs/release/surface/surface_v1_release_notes.md",
		`# Fake Surface Claim

Surface GPU rendering is production supported for the current release.
`,
	)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted unsupported GPU production claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "gpu") {
		t.Fatalf("error = %v, want GPU diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsMixedGPUProductionWithoutEvidenceClause(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"docs/release/surface/surface_v1_release_notes.md",
		`# Fake Surface Claim

Surface GPU rendering is production supported without additional target-host evidence.
`,
	)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err == nil {
		t.Fatalf(
			"validateSurfaceClaims accepted mixed GPU production claim with without-evidence wording",
		)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "gpu") {
		t.Fatalf("error = %v, want GPU diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsStaleProductionEvidence(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "reports/surface-release-v1/stale-summary.json", `{
  "schema": "tetra.surface.release.v1",
  "release_scope": "surface-v1-linux-web",
  "production_claim": true,
  "same_commit_validated": false
}
`)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:       root,
		ReportDirs: []string{filepath.Join(root, "reports")},
	})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted stale production evidence")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "stale") {
		t.Fatalf("error = %v, want stale evidence diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsDocsOnlyProductionClaim(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "docs/spec/surface/surface_v1.md", `# Fake Surface Claim

Surface production support is proven by docs-only evidence.
`)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted docs-only production claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "docs-only") {
		t.Fatalf("error = %v, want docs-only diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsNativeSurfaceHostPromotionWithoutStrictEvidence(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"docs/release/surface/surface_v1_release_notes.md",
		`# Fake Surface Claim

Native Surface Host v1 is current and release-ready for Linux Surface apps.
`,
	)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err == nil {
		t.Fatalf(
			"validateSurfaceClaims accepted Native Surface Host v1 promotion without strict evidence",
		)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "native surface host") {
		t.Fatalf("error = %v, want Native Surface Host diagnostic", err)
	}
}

func TestValidateSurfaceClaimsAllowsNativeSurfaceHostBoundaryNonclaim(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "docs/spec/surface/native_surface_host_v1.md", `# Honest Boundary

Native Surface Host v1 is not current release support until strict
linux-x64-native-surface-host-v1 evidence exists.
`)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err != nil {
		t.Fatalf("validateSurfaceClaims rejected Native Surface Host boundary: %v", err)
	}
}

func TestValidateSurfaceClaimsAllowsNativeSurfaceHostPromotionWithStrictEvidence(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"docs/release/surface/surface_v1_release_notes.md",
		`# Fake Surface Claim

Native Surface Host v1 is current for the strict checked Linux Surface scope.
`,
	)
	writeSurfaceClaimFixture(
		t,
		root,
		"reports/surface-native/surface-linux-x64-native-host.json",
		strictNativeSurfaceHostClaimReportJSON(1),
	)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:       root,
		ReportDirs: []string{filepath.Join(root, "reports")},
	})
	if err != nil {
		t.Fatalf("validateSurfaceClaims rejected strict Native Surface Host evidence: %v", err)
	}
}

func TestValidateSurfaceClaimsRejectsNativeSurfaceHostPromotionWithPointerlessReport(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"docs/release/surface/surface_v1_release_notes.md",
		`# Fake Surface Claim

Native Surface Host v1 is supported for Linux Surface apps.
`,
	)
	writeSurfaceClaimFixture(
		t,
		root,
		"reports/surface-native/surface-linux-x64-native-host.json",
		strictNativeSurfaceHostClaimReportJSON(0),
	)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:       root,
		ReportDirs: []string{filepath.Join(root, "reports")},
	})
	if err == nil {
		t.Fatalf(
			"validateSurfaceClaims accepted Native Surface Host promotion with no pointer evidence",
		)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "native surface host") {
		t.Fatalf("error = %v, want Native Surface Host diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsWindowsMacOSProductionWithoutTargetHostEvidence(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(
		t,
		root,
		"docs/release/surface/surface_v1_release_notes.md",
		`# Fake Surface Claim

Windows Surface and macOS Surface are production supported real-window targets.
`,
	)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err == nil {
		t.Fatalf(
			"validateSurfaceClaims accepted Windows/macOS production support without target-host evidence",
		)
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "windows") || !strings.Contains(lower, "macos") {
		t.Fatalf("error = %v, want Windows and macOS diagnostics", err)
	}
}

func TestValidateSurfaceClaimsAllowsScopedNonClaims(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "README.md", `# Honest Surface Scope

Surface v1 is PROD_STABLE_SCOPED for surface-v1-linux-web.
Surface is not an Electron replacement, not a React replacement, and no CSS replacement claim is made.
Morph remains EXPERIMENTAL; no production Morph claim is made.
No Electron-quality UI, React-quality UI, pixel-perfect Surface, or production-ready Morph claim is made.
Windows Surface and macOS Surface are UNSUPPORTED until BETA_TARGET_HOST reports exist.
`)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err != nil {
		t.Fatalf("validateSurfaceClaims rejected scoped nonclaims: %v", err)
	}
}

func writeSurfaceClaimMRBReport(
	t *testing.T,
	root string,
	rel string,
	gitHead string,
	productClaim bool,
	finalSignoff bool,
) {
	t.Helper()
	report := validSurfaceClaimMRBReport(gitHead, productClaim, finalSignoff)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal MRB report: %v", err)
	}
	writeSurfaceClaimFixture(t, root, rel, string(raw)+"\n")
}

func validSurfaceClaimMRBReport(
	gitHead string,
	productClaim bool,
	finalSignoff bool,
) surface.MorphRenderedBeautyReport {
	source := "examples/surface/morph_core/surface_morph_command_palette.tetra"
	blockHash := "sha256:" + strings.Repeat("5", 64)
	streamHash := "sha256:" + strings.Repeat("6", 64)
	frameChecksum := "sha256:" + strings.Repeat("8", 64)
	return surface.MorphRenderedBeautyReport{
		Schema:         surface.MorphRenderedBeautyReportSchemaV1,
		Status:         "pass",
		SurfaceScope:   surface.MorphRenderedBeautyScope,
		Target:         "headless",
		ScenarioName:   "headless-morph",
		GitHead:        gitHead,
		GitCommit:      gitHead,
		ProductClaim:   productClaim,
		FinalSignoff:   finalSignoff,
		CorePrimitives: []string{"Block"},
		MorphEvidence: surface.MorphRenderedBeautyMorphEvidence{
			Source:         source,
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
			RecipeCount:            3,
			RecipeExpansionCount:   19,
			RecipeNames:            []string{"control.action@1", "field.text@1", "command.item@1"},
			ResolvedMorphSceneHash: "sha256:" + strings.Repeat("4", 64),
			BlockSceneSnapshotHash: blockHash,
		},
		BlockSceneSnapshot: surface.MorphRenderedBeautyBlockSceneSnapshot{
			Schema:               "tetra.surface.block-scene-snapshot.v1",
			SurfaceScope:         surface.MorphRenderedBeautyScope,
			Source:               source,
			QualityLevel:         "rich-renderable-block-scene-v1",
			CorePrimitives:       []string{"Block"},
			RecipeExpansionCount: 19,
			NodeCount:            5,
			RichSpecHash:         "sha256:" + strings.Repeat("b", 64),
			BlockSceneHash:       blockHash,
			SpecCoverage: surface.MorphRenderedBeautyBlockSceneSpecCoverage{
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
		RenderEvidence: surface.MorphRenderedBeautyRenderEvidence{
			CommandStreamHash: streamHash,
			CommandCount:      10,
			Renderer:          "software-rgba-headless",
		},
		RendererStableProof: surface.MorphRenderedBeautyRendererStableProof{
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
		RenderCommandStream: surface.MorphRenderedBeautyRenderCommandStream{
			Schema:                        "tetra.surface.render-command-stream.v1",
			Source:                        source,
			SurfaceScope:                  surface.MorphRenderedBeautyScope,
			Producer:                      "surface-runtime-smoke",
			QualityLevel:                  "deterministic-render-command-stream-v1",
			Renderer:                      "software-rgba-headless",
			DerivedFromBlockSceneSnapshot: true,
			BlockSceneHash:                blockHash,
			FrameChecksum:                 frameChecksum,
			CommandStreamHash:             streamHash,
			CommandCount:                  10,
			SourceLinked:                  true,
			Commands: []surface.MorphRenderedBeautyRenderCommand{
				surfaceClaimMRBCommand(source, 1, "fill"),
				surfaceClaimMRBCommand(source, 2, "gradient"),
				surfaceClaimMRBCommand(source, 3, "image_fill"),
				surfaceClaimMRBCommand(source, 4, "border"),
				surfaceClaimMRBCommand(source, 5, "radius_clip"),
				surfaceClaimMRBCommand(source, 6, "shadow"),
				surfaceClaimMRBCommand(source, 7, "overlay"),
				surfaceClaimMRBCommand(source, 8, "outline"),
				surfaceClaimMRBCommand(source, 9, "text"),
				surfaceClaimMRBCommand(source, 10, "icon"),
			},
		},
		PixelEvidence: surface.MorphRenderedBeautyPixelEvidence{
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
			DiffPixels:              1,
			MaxChannelDelta:         1,
		},
		NegativeGuards: surface.MorphRenderedBeautyNegativeGuards{
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

func strictNativeSurfaceHostClaimReportJSON(pointerEvents int) string {
	return fmt.Sprintf(`{
  "schema": "tetra.surface.runtime.v1",
  "target": "linux-x64",
  "runtime": "surface-linux-x64",
  "host_evidence": {
    "level": "linux-x64-native-surface-host-v1",
    "backend": "wayland-surface-host-v1",
    "framebuffer": true,
    "real_window": true,
    "native_input": true,
    "user_facing_platform_widgets": false
  },
  "native_surface_host": {
    "schema": "tetra.surface.native-host.v1",
    "host": "wayland",
    "protocol": "tetra.surface.host-ipc.v1",
    "app_process_kind": "compiled-linux-x64-tetra-app",
    "host_process_kind": "tetra-surface-host-wayland",
    "app_pid": 4242,
    "host_pid": 4243,
    "surface_open_from_app": true,
    "poll_event_from_host": true,
    "present_from_app_rgba": true,
    "app_loop_observed": true,
    "real_window": true,
    "real_close_event": true,
    "real_pointer_event_count": %d,
    "real_key_event_count": 1,
    "presented_frame_count": 2,
    "pre_rendered_frame_source": false,
    "delivery_path": "compiled-tetra-app-to-wayland-surface"
  }
}
`, pointerEvents)
}

func surfaceClaimMRBCommand(
	source string,
	order int,
	command string,
) surface.MorphRenderedBeautyRenderCommand {
	item := surface.MorphRenderedBeautyRenderCommand{
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
		item.Color = surfaceClaimMRBCommandColor(command)
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

func surfaceClaimMRBCommandColor(command string) string {
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

func writeSurfaceClaimFixture(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
