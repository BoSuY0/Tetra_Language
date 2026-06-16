package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestSurfaceDevCommandWritesFastRebuildReport(t *testing.T) {
	target := mustHostTarget(t)
	if target != "linux-x64" {
		t.Skip("Surface dev fast rebuild cache evidence is currently linux-x64 scoped")
	}
	dir := t.TempDir()
	entry, tokens, recipes := writeSurfaceDevFixture(t, dir)
	reportPath := filepath.Join(dir, "surface-dev-workflow.json")
	morphRenderedBeautyReportPath := filepath.Join(dir, "surface-morph-rendered-beauty.json")
	writeSurfaceDevMorphRenderedBeautyReport(t, morphRenderedBeautyReportPath, entry)
	outDir := filepath.Join(dir, "dist")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{
		"surface", "dev",
		"--source", entry,
		"--target", target,
		"--out-dir", outDir,
		"--report", reportPath,
		"--morph-rendered-beauty-report", morphRenderedBeautyReportPath,
		"--change-file", "token:" + tokens,
		"--change-file", "recipe:" + recipes,
		"--change-file", "source:" + entry,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("surface dev exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report struct {
		Schema                 string `json:"schema"`
		Model                  string `json:"model"`
		ReleaseScope           string `json:"release_scope"`
		Command                string `json:"command"`
		Mode                   string `json:"mode"`
		ReloadSemantics        string `json:"reload_semantics"`
		ProcessRestartRequired bool   `json:"process_restart_required"`
		HotReloadClaim         bool   `json:"hot_reload_claim"`
		Pass                   bool   `json:"pass"`
		Steps                  []struct {
			Name            string   `json:"name"`
			Kind            string   `json:"kind"`
			ChangedPath     string   `json:"changed_path"`
			DurationMS      int64    `json:"duration_ms"`
			CompiledModules []string `json:"compiled_modules"`
			CacheHits       []string `json:"cache_hits"`
			Pass            bool     `json:"pass"`
		} `json:"steps"`
		SourceDiagnostics []struct {
			Kind     string `json:"kind"`
			Path     string `json:"path"`
			Line     int    `json:"line"`
			Column   int    `json:"column"`
			Severity string `json:"severity"`
			Pass     bool   `json:"pass"`
		} `json:"source_diagnostics"`
		MorphToPixels struct {
			ChainID                 string `json:"chain_id"`
			ReportPath              string `json:"report_path"`
			Source                  string `json:"source"`
			TokenCount              int    `json:"token_count"`
			RecipeCount             int    `json:"recipe_count"`
			RecipeExpansionCount    int    `json:"recipe_expansion_count"`
			BlockSceneHash          string `json:"block_scene_hash"`
			RenderCommandStreamHash string `json:"render_command_stream_hash"`
			RenderCommandCount      int    `json:"render_command_count"`
			FrameArtifact           string `json:"frame_artifact"`
			GoldenArtifact          string `json:"golden_artifact"`
			DiffPixels              int    `json:"diff_pixels"`
			Pass                    bool   `json:"pass"`
		} `json:"morph_to_pixels"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, string(raw))
	}
	if report.Schema != "tetra.surface.dev-workflow.v1" ||
		report.Model != "surface-dev-workflow-v1" ||
		report.ReleaseScope != "surface-v1-linux-web" ||
		report.Command != "tetra surface dev" ||
		report.Mode != "fast-rebuild" ||
		report.ReloadSemantics != "fast-rebuild" ||
		!report.ProcessRestartRequired ||
		report.HotReloadClaim ||
		!report.Pass {
		t.Fatalf("unexpected report header = %#v", report)
	}
	steps := map[string]struct {
		compiled int
		cache    int
		pass     bool
	}{}
	for _, step := range report.Steps {
		if step.DurationMS < 0 || !step.Pass {
			t.Fatalf("bad step = %#v", step)
		}
		steps[step.Kind] = struct {
			compiled int
			cache    int
			pass     bool
		}{compiled: len(step.CompiledModules), cache: len(step.CacheHits), pass: step.Pass}
	}
	for _, want := range []string{"initial", "warm-cache", "token-change", "recipe-change", "source-change"} {
		if !steps[want].pass {
			t.Fatalf("missing or failed rebuild step %q in %#v", want, report.Steps)
		}
	}
	if steps["warm-cache"].compiled != 0 || steps["warm-cache"].cache == 0 {
		t.Fatalf("warm-cache step = %#v, want zero compiled modules and cache hits", steps["warm-cache"])
	}
	for _, want := range []string{"token-change", "recipe-change", "source-change"} {
		if steps[want].compiled == 0 {
			t.Fatalf("%s step = %#v, want changed module compilation", want, steps[want])
		}
	}
	diagnosticKinds := map[string]bool{}
	for _, diag := range report.SourceDiagnostics {
		if diag.Path == "" || diag.Line <= 0 || diag.Column <= 0 || diag.Severity == "" || !diag.Pass {
			t.Fatalf("bad source diagnostic = %#v", diag)
		}
		diagnosticKinds[diag.Kind] = true
	}
	for _, want := range []string{"token", "recipe", "source"} {
		if !diagnosticKinds[want] {
			t.Fatalf("missing %s source diagnostic in %#v", want, report.SourceDiagnostics)
		}
	}
	if !report.MorphToPixels.Pass ||
		report.MorphToPixels.ChainID == "" ||
		filepath.Clean(report.MorphToPixels.Source) != filepath.Clean(entry) ||
		report.MorphToPixels.TokenCount == 0 ||
		report.MorphToPixels.RecipeCount == 0 ||
		report.MorphToPixels.RecipeExpansionCount < report.MorphToPixels.RecipeCount ||
		report.MorphToPixels.RenderCommandCount == 0 ||
		report.MorphToPixels.BlockSceneHash == "" ||
		report.MorphToPixels.RenderCommandStreamHash == "" ||
		report.MorphToPixels.FrameArtifact == "" ||
		report.MorphToPixels.GoldenArtifact == "" {
		t.Fatalf("morph_to_pixels = %#v, want source-linked Morph-to-pixels chain", report.MorphToPixels)
	}
}

func TestSurfaceDevCommandJSONDiagnosticIncludesSurfacePath(t *testing.T) {
	target := mustHostTarget(t)
	if target != "linux-x64" {
		t.Skip("Surface dev diagnostic smoke is currently linux-x64 scoped")
	}
	dir := t.TempDir()
	entry := filepath.Join(dir, "app", "main.tetra")
	writeCLIProjectFile(t, dir, "app/main.tetra", "module app.main\nimport lib.core.morph as morph\nfunc main() -> Int:\n    let x: Int =\n    return 0\n")
	reportPath := filepath.Join(dir, "surface-dev-workflow.json")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{
		"surface", "dev",
		"--source", entry,
		"--target", target,
		"--diagnostics", "json",
		"--report", reportPath,
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("surface dev exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("stdout = %q, want empty on JSON diagnostic failure", stdout.String())
	}
	var cliDiag cliJSONDiagnostic
	if err := json.Unmarshal(stderr.Bytes(), &cliDiag); err != nil {
		t.Fatalf("decode CLI diagnostic: %v\n%s", err, stderr.String())
	}
	if filepath.Clean(cliDiag.File) != filepath.Clean(entry) || cliDiag.Line <= 0 || cliDiag.Column <= 0 || cliDiag.Severity != "error" {
		t.Fatalf("CLI diagnostic = %#v, want positioned source error for %s", cliDiag, entry)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report struct {
		Pass              bool `json:"pass"`
		SourceDiagnostics []struct {
			Kind     string `json:"kind"`
			Path     string `json:"path"`
			Line     int    `json:"line"`
			Column   int    `json:"column"`
			Severity string `json:"severity"`
			Pass     bool   `json:"pass"`
		} `json:"source_diagnostics"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, string(raw))
	}
	if report.Pass || len(report.SourceDiagnostics) == 0 {
		t.Fatalf("report = %#v, want failing source diagnostic", report)
	}
	first := report.SourceDiagnostics[0]
	if first.Kind != "morph" || filepath.Clean(first.Path) != filepath.Clean(entry) || first.Line <= 0 || first.Column <= 0 || first.Severity != "error" || first.Pass {
		t.Fatalf("source diagnostic = %#v, want Morph-positioned failing diagnostic", first)
	}
}

func writeSurfaceDevFixture(t *testing.T, dir string) (entry string, tokens string, recipes string) {
	t.Helper()
	unique := strings.ReplaceAll(filepath.Base(dir), "-", "_")
	tokens = filepath.Join(dir, "design", "tokens.tetra")
	recipes = filepath.Join(dir, "design", "recipes.tetra")
	entry = filepath.Join(dir, "app", "main.tetra")
	writeCLIProjectFile(t, dir, "design/tokens.tetra", "module design.tokens\n// "+unique+"\nfunc accent() -> Int:\n    return 17\n")
	writeCLIProjectFile(t, dir, "design/recipes.tetra", "module design.recipes\n// "+unique+"\nfunc card() -> Int:\n    return 25\n")
	writeCLIProjectFile(t, dir, "app/main.tetra", "module app.main\n// "+unique+"\nimport design.tokens as tokens\nimport design.recipes as recipes\nfunc main() -> Int:\n    return tokens.accent() + recipes.card()\n")
	return entry, tokens, recipes
}

func writeSurfaceDevMorphRenderedBeautyReport(t *testing.T, path string, source string) {
	t.Helper()
	report := validSurfaceDevMorphRenderedBeautyReport(source)
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

func validSurfaceDevMorphRenderedBeautyReport(source string) surface.MorphRenderedBeautyReport {
	blockSceneHash := surfaceDevTestSHA(5)
	commandStreamHash := surfaceDevTestSHA(7)
	frameHash := surfaceDevTestSHA(60)
	goldenHash := surfaceDevTestSHA(61)
	commands := []string{"fill", "gradient", "image_fill", "border", "radius_clip", "shadow", "overlay", "outline", "text", "icon"}
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
			Checksum:     surfaceDevTestSHA(100 + i),
		}
		if command == "text" {
			item.RasterFormat = "builtin-5x7-alpha-mask-v1"
			item.RasterHash = surfaceDevTestSHA(210)
			item.RasterWidth = 5
			item.RasterHeight = 7
			item.RasterCoverage = 20
		}
		if command == "icon" {
			item.RasterFormat = "builtin-icon-mask-raster-v1"
			item.RasterHash = surfaceDevTestSHA(211)
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
		CorePrimitives: []string{"Block"},
		MorphEvidence: surface.MorphRenderedBeautyMorphEvidence{
			Source:                 source,
			SourceSHA256:           surfaceDevTestSHA(1),
			CapsuleHash:            surfaceDevTestSHA(2),
			TokenGraphHash:         surfaceDevTestSHA(3),
			TokenCount:             6,
			TokenCategories:        []string{"color", "space", "radius", "typography", "motion", "assets"},
			RecipeCount:            3,
			RecipeExpansionCount:   4,
			RecipeNames:            []string{"studio_shell", "hero_panel", "toolbar"},
			ResolvedMorphSceneHash: surfaceDevTestSHA(4),
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
			RichSpecHash:         surfaceDevTestSHA(6),
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
			FrameArtifact:           "reports/surface/dev-frame.rgba",
			FrameArtifactSHA256:     frameHash,
			FrameChecksum:           frameHash,
			FrameProducer:           "app",
			AppSource:               source,
			MorphRecipeHash:         surfaceDevTestSHA(8),
			BlockSceneHash:          blockSceneHash,
			RenderCommandStreamHash: commandStreamHash,
			GoldenArtifact:          "reports/surface/dev-golden.rgba",
			GoldenArtifactSHA256:    goldenHash,
			GoldenChecksum:          goldenHash,
			DiffPixels:              1,
			MaxChannelDelta:         1,
		},
		NegativeGuards: surface.MorphRenderedBeautyNegativeGuards{
			MetadataOnlyRejected: true, SelfGoldenRejected: true, PrecomputedFrameRejected: true, MissingFrameArtifactRejected: true,
			NoDOMUI: true, NoCSSRuntime: true, NoReactRuntime: true, NoElectronRuntime: true, NoNativeWidgets: true, NoHiddenAppState: true,
			NonBlockOutputRejected: true, DirtyCheckoutProductionRejected: true, UnsupportedTargetRejected: true,
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

func surfaceDevTestSHA(seed int) string {
	return "sha256:" + fmt.Sprintf("%064x", seed)
}
