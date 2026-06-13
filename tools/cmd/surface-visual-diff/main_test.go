package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestRunGeneratesValidSurfaceVisualReport(t *testing.T) {
	dir := t.TempDir()
	runtimePath := filepath.Join(dir, "surface-headless-block-system.json")
	outPath := filepath.Join(dir, "surface-visual-regression.json")
	if err := os.WriteFile(runtimePath, validVisualDiffRuntimeReportJSON(t, false), 0o644); err != nil {
		t.Fatalf("write runtime report: %v", err)
	}

	err := run([]string{
		"--runtime-report", runtimePath,
		"--required-target", "headless",
		"--git-head", "c0258b63a636775b114d69d31cb7832fc3991b05",
		"--out", outPath,
	})
	if err != nil {
		t.Fatalf("run surface-visual-diff: %v", err)
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read visual report: %v", err)
	}
	if err := surface.ValidateVisualReport(raw); err != nil {
		t.Fatalf("generated visual report failed validation: %v\n%s", err, raw)
	}
	var report surface.VisualRegressionReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode generated visual report: %v", err)
	}
	target := report.Apps[0].Targets[0]
	if target.RuntimeReport != runtimePath {
		t.Fatalf("runtime_report = %q, want %q", target.RuntimeReport, runtimePath)
	}
	if target.Renderer != "software-rgba" {
		t.Fatalf("renderer = %q, want software-rgba", target.Renderer)
	}
}

func TestRunRejectsMajorGoldenDrift(t *testing.T) {
	dir := t.TempDir()
	runtimePath := filepath.Join(dir, "surface-headless-block-system.json")
	outPath := filepath.Join(dir, "surface-visual-regression.json")
	if err := os.WriteFile(runtimePath, validVisualDiffRuntimeReportJSON(t, true), 0o644); err != nil {
		t.Fatalf("write runtime report: %v", err)
	}

	err := run([]string{
		"--runtime-report", runtimePath,
		"--required-target", "headless",
		"--git-head", "c0258b63a636775b114d69d31cb7832fc3991b05",
		"--out", outPath,
	})
	if err == nil {
		t.Fatalf("expected major golden drift to fail")
	}
	if !strings.Contains(err.Error(), "visual drift") {
		t.Fatalf("error = %v, want visual drift diagnostic", err)
	}
}

func TestRunGeneratesPolishedReferenceAppVisualEvidence(t *testing.T) {
	dir := t.TempDir()
	headlessPath := writeVisualDiffRuntimeReport(t, dir, "headless", "headless", false)
	linuxPath := writeVisualDiffRuntimeReport(t, dir, "linux-x64-real-window", "linux-x64", false)
	webPath := writeVisualDiffRuntimeReport(t, dir, "wasm32-web-browser-canvas", "wasm32-web", false)
	examplesPath := filepath.Join(dir, "surface-block-examples.json")
	if err := os.WriteFile(examplesPath, validBlockExamplesReportJSON(t, dir, false), 0o644); err != nil {
		t.Fatalf("write block examples report: %v", err)
	}
	outPath := filepath.Join(dir, "surface-visual-regression.json")

	err := run([]string{
		"--runtime-report", headlessPath,
		"--runtime-report", linuxPath,
		"--runtime-report", webPath,
		"--block-examples-report", examplesPath,
		"--required-target", "headless",
		"--required-target", "linux-x64-real-window",
		"--required-target", "wasm32-web-browser-canvas",
		"--git-head", "c0258b63a636775b114d69d31cb7832fc3991b05",
		"--out", outPath,
	})
	if err != nil {
		t.Fatalf("run surface-visual-diff: %v", err)
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read visual report: %v", err)
	}
	if err := surface.ValidateVisualReport(raw); err != nil {
		t.Fatalf("generated visual report failed validation: %v\n%s", err, raw)
	}
	var report surface.VisualRegressionReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode generated visual report: %v", err)
	}
	if got, want := len(report.Apps), 3; got != want {
		t.Fatalf("app count = %d, want %d", got, want)
	}
	for _, source := range []string{
		"examples/surface_block_command_palette.tetra",
		"examples/surface_block_settings.tetra",
	} {
		app := findVisualApp(report, source)
		if app == nil {
			t.Fatalf("missing polished reference app %s in visual report", source)
		}
		if got, want := len(app.Targets), 3; got != want {
			t.Fatalf("%s target count = %d, want %d", source, got, want)
		}
		if app.Targets[0].Frames[0].Checksum == findVisualApp(report, "examples/surface_block_system.tetra").Targets[0].Frames[0].Checksum {
			t.Fatalf("%s reused BlockSystem frame checksum", source)
		}
	}
}

func TestRunRejectsPolishedReferenceAppWithMissingEvidence(t *testing.T) {
	dir := t.TempDir()
	runtimePath := writeVisualDiffRuntimeReport(t, dir, "headless", "headless", false)
	examplesPath := filepath.Join(dir, "surface-block-examples.json")
	if err := os.WriteFile(examplesPath, validBlockExamplesReportJSON(t, dir, true), 0o644); err != nil {
		t.Fatalf("write block examples report: %v", err)
	}
	outPath := filepath.Join(dir, "surface-visual-regression.json")

	err := run([]string{
		"--runtime-report", runtimePath,
		"--block-examples-report", examplesPath,
		"--required-target", "headless",
		"--git-head", "c0258b63a636775b114d69d31cb7832fc3991b05",
		"--out", outPath,
	})
	if err == nil {
		t.Fatalf("expected missing polished reference app evidence to fail")
	}
	if !strings.Contains(err.Error(), "accessibility evidence") {
		t.Fatalf("error = %v, want accessibility evidence diagnostic", err)
	}
}

func writeVisualDiffRuntimeReport(t *testing.T, dir string, visualTargetName string, reportTarget string, drift bool) string {
	t.Helper()
	reportPath := filepath.Join(dir, visualTargetName+".json")
	var report surface.Report
	if err := json.Unmarshal(validVisualDiffRuntimeReportJSON(t, drift), &report); err != nil {
		t.Fatalf("decode runtime fixture: %v", err)
	}
	report.Target = reportTarget
	switch visualTargetName {
	case "linux-x64-real-window":
		report.Runtime = "surface-linux-x64"
		report.HostEvidence.RealWindow = true
	case "wasm32-web-browser-canvas":
		report.Runtime = "surface-wasm32-web"
		report.HostEvidence.BrowserCanvas = true
	default:
		report.Runtime = "surface-headless"
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal runtime fixture: %v", err)
	}
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write runtime report: %v", err)
	}
	return reportPath
}

func findVisualApp(report surface.VisualRegressionReport, source string) *surface.VisualRegressionAppReport {
	for i := range report.Apps {
		if report.Apps[i].Source == source {
			return &report.Apps[i]
		}
	}
	return nil
}

func validVisualDiffRuntimeReportJSON(t *testing.T, drift bool) []byte {
	t.Helper()
	checksum := "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	goldenChecksum := checksum
	if drift {
		goldenChecksum = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	}
	report := surface.Report{
		Schema:         surface.SchemaV1,
		Status:         "pass",
		Target:         "headless",
		Runtime:        "surface-headless",
		Source:         "examples/surface_block_system.tetra",
		BlockGraph:     &surface.BlockGraphReport{Schema: "tetra.surface.block-graph.v1"},
		VisualFeatures: []string{"tokens", "theme", "contrast"},
		Renderer:       &surface.RendererReport{Backend: "software-rgba"},
		LayoutConstraints: []surface.BlockLayoutConstraintReport{
			{ID: "root"},
		},
		LayoutPasses: []surface.BlockLayoutPassReport{
			{
				Order:    1,
				BlockID:  1,
				Mode:     "column",
				Checksum: "sha256:3333333333333333333333333333333333333333333333333333333333333333",
			},
		},
		LayoutScrolls: []surface.BlockLayoutScrollReport{
			{
				BlockID:  1,
				Checksum: "sha256:4444444444444444444444444444444444444444444444444444444444444444",
			},
		},
		BlockAccessibilityTree: &surface.BlockAccessibilityTreeReport{Schema: "tetra.surface.block-accessibility-tree.v1"},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     320,
				Height:    200,
				Stride:    1280,
				Checksum:  checksum,
				Presented: true,
			},
		},
		BlockSystem: &surface.BlockSystemReport{
			Schema:       "tetra.surface.block-system.v1",
			QualityLevel: "deterministic-headless-block-system-v1",
			Source:       "examples/surface_block_system.tetra",
			Renderer:     "software-rgba-headless",
			GoldenSet:    "surface-visual-regression-v1",
			FrameCount:   1,
			GoldenHash:   "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Frames: []surface.BlockSystemFrameReport{
				{
					Order:                 1,
					Label:                 "initial",
					Width:                 320,
					Height:                200,
					Stride:                1280,
					Checksum:              checksum,
					RepeatChecksum:        checksum,
					GoldenChecksum:        goldenChecksum,
					PaintEvidence:         true,
					LayoutEvidence:        true,
					AccessibilityEvidence: true,
				},
			},
			MemoryBudget: &surface.BlockMemoryBudgetReport{
				Schema: "tetra.surface.block-memory-budget.v1",
				Scope:  "surface-block-system-local-budget-v1",
			},
		},
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal runtime report: %v", err)
	}
	return raw
}

func validBlockExamplesReportJSON(t *testing.T, dir string, missingEvidence bool) []byte {
	t.Helper()
	artifactDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatalf("create artifact dir: %v", err)
	}
	examples := []map[string]any{}
	for _, name := range []string{"surface_block_command_palette", "surface_block_settings"} {
		artifact := filepath.Join(artifactDir, name)
		if err := os.WriteFile(artifact, []byte("artifact:"+name), 0o755); err != nil {
			t.Fatalf("write artifact: %v", err)
		}
		accessibility := true
		if missingEvidence && name == "surface_block_settings" {
			accessibility = false
		}
		examples = append(examples, map[string]any{
			"path":                   "examples/" + name + ".tetra",
			"block_only":             true,
			"compiles":               true,
			"runs":                   true,
			"exit_code":              0,
			"theme_tokens":           true,
			"paint_evidence":         true,
			"layout_evidence":        true,
			"text_evidence":          true,
			"asset_evidence":         true,
			"accessibility_evidence": accessibility,
			"hover_evidence":         true,
			"focus_evidence":         true,
			"pressed_evidence":       true,
			"motion_evidence":        true,
			"checksum_evidence":      true,
			"modules":                []string{"lib.core.surface", "lib.core.block"},
			"artifact":               artifact,
		})
	}
	report := map[string]any{
		"schema":        "tetra.surface.block-examples.v1",
		"quality_level": "block-only-polished-examples-v1",
		"example_count": len(examples),
		"examples":      examples,
		"negative_guards": map[string]bool{
			"core_widget_usage_rejected":        true,
			"missing_accessibility_rejected":    true,
			"missing_hover_focus_pressed_state": true,
		},
		"feature_totals": map[string]int{
			"accessibility":       2,
			"asset":               2,
			"checksum":            2,
			"hover_focus_pressed": 2,
			"layout":              2,
			"motion":              2,
			"paint":               2,
			"text":                2,
			"theme_tokens":        2,
		},
		"pass": true,
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block examples report: %v", err)
	}
	return raw
}
