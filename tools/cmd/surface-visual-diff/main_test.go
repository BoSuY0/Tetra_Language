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

func TestRunGeneratesValidSurfaceVisualReport(t *testing.T) {
	dir := t.TempDir()
	runtimePath := filepath.Join(dir, "surface-headless-block-system.json")
	outPath := filepath.Join(dir, "surface-visual-regression.json")
	artifactEvidence := writeVisualArtifactArgs(t, dir, "examples/surface_block_system.tetra", "headless", 1, false)
	if err := os.WriteFile(runtimePath, validVisualDiffRuntimeReportJSON(t, false, artifactEvidence.CurrentSHA), 0o644); err != nil {
		t.Fatalf("write runtime report: %v", err)
	}

	args := []string{
		"--runtime-report", runtimePath,
		"--required-target", "headless",
		"--git-head", "c0258b63a636775b114d69d31cb7832fc3991b05",
		"--out", outPath,
	}
	args = append(args, artifactEvidence.Args...)
	err := run(args)
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
	artifactEvidence := writeVisualArtifactArgs(t, dir, "examples/surface_block_system.tetra", "headless", 1, true)
	if err := os.WriteFile(runtimePath, validVisualDiffRuntimeReportJSON(t, true, artifactEvidence.CurrentSHA), 0o644); err != nil {
		t.Fatalf("write runtime report: %v", err)
	}

	args := []string{
		"--runtime-report", runtimePath,
		"--required-target", "headless",
		"--git-head", "c0258b63a636775b114d69d31cb7832fc3991b05",
		"--out", outPath,
	}
	args = append(args, artifactEvidence.Args...)
	err := run(args)
	if err == nil {
		t.Fatalf("expected major golden drift to fail")
	}
	if !strings.Contains(err.Error(), "visual drift") {
		t.Fatalf("error = %v, want visual drift diagnostic", err)
	}
}

func TestRunRejectsMetadataOnlyVisualEvidence(t *testing.T) {
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
	if err == nil {
		t.Fatalf("expected metadata-only visual evidence to fail")
	}
	if !strings.Contains(err.Error(), "frame artifact") {
		t.Fatalf("error = %v, want frame artifact diagnostic", err)
	}
}

func TestRunRejectsSelfGoldenArtifact(t *testing.T) {
	dir := t.TempDir()
	runtimePath := filepath.Join(dir, "surface-headless-block-system.json")
	outPath := filepath.Join(dir, "surface-visual-regression.json")
	currentPath := writeRGBAArtifact(t, filepath.Join(dir, "frames", "self.rgba"), 320, 200, 1280, "self-golden", false)
	currentSHA, err := sha256File(currentPath)
	if err != nil {
		t.Fatalf("hash current artifact: %v", err)
	}
	if err := os.WriteFile(runtimePath, validVisualDiffRuntimeReportJSON(t, false, currentSHA), 0o644); err != nil {
		t.Fatalf("write runtime report: %v", err)
	}

	err = run([]string{
		"--runtime-report", runtimePath,
		"--required-target", "headless",
		"--frame-artifact", fmt.Sprintf("%s,%s,%d,%s", "examples/surface_block_system.tetra", "headless", 1, currentPath),
		"--golden-artifact", fmt.Sprintf("%s,%s,%d,%s", "examples/surface_block_system.tetra", "headless", 1, currentPath),
		"--git-head", "c0258b63a636775b114d69d31cb7832fc3991b05",
		"--out", outPath,
	})
	if err == nil {
		t.Fatalf("expected self-golden artifact to fail")
	}
	if !strings.Contains(err.Error(), "self-golden") {
		t.Fatalf("error = %v, want self-golden diagnostic", err)
	}
}

func TestRunUsesRuntimeFrameArtifactPathWithBareChecksum(t *testing.T) {
	dir := t.TempDir()
	runtimePath := filepath.Join(dir, "surface-headless-block-system.json")
	outPath := filepath.Join(dir, "surface-visual-regression.json")
	currentPath := writeRGBAArtifact(t, filepath.Join(dir, "runtime", "initial.rgba"), 320, 200, 1280, "runtime-artifact-path", false)
	goldenPath := writeRGBAArtifact(t, filepath.Join(dir, "goldens", "initial.rgba"), 320, 200, 1280, "runtime-artifact-path", false)
	currentSHA, err := sha256File(currentPath)
	if err != nil {
		t.Fatalf("hash current artifact: %v", err)
	}
	bareCurrentSHA := strings.TrimPrefix(currentSHA, "sha256:")
	var report surface.Report
	if err := json.Unmarshal(validVisualDiffRuntimeReportJSON(t, false, bareCurrentSHA), &report); err != nil {
		t.Fatalf("decode runtime fixture: %v", err)
	}
	report.Frames[0].ArtifactPath = currentPath
	report.BlockSystem.Frames[0].ArtifactPath = currentPath
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal runtime fixture: %v", err)
	}
	if err := os.WriteFile(runtimePath, raw, 0o644); err != nil {
		t.Fatalf("write runtime report: %v", err)
	}

	err = run([]string{
		"--runtime-report", runtimePath,
		"--required-target", "headless",
		"--golden-artifact", fmt.Sprintf("%s,%s,%d,%s", "examples/surface_block_system.tetra", "headless", 1, goldenPath),
		"--git-head", "c0258b63a636775b114d69d31cb7832fc3991b05",
		"--out", outPath,
	})
	if err != nil {
		t.Fatalf("run surface-visual-diff with runtime artifact_path: %v", err)
	}
	raw, err = os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read visual report: %v", err)
	}
	var visual surface.VisualRegressionReport
	if err := json.Unmarshal(raw, &visual); err != nil {
		t.Fatalf("decode visual report: %v", err)
	}
	frame := visual.Apps[0].Targets[0].Frames[0]
	if frame.ArtifactPath != currentPath {
		t.Fatalf("artifact_path = %q, want runtime frame artifact %q", frame.ArtifactPath, currentPath)
	}
	if frame.Checksum != currentSHA {
		t.Fatalf("checksum = %q, want normalized artifact checksum %q", frame.Checksum, currentSHA)
	}
	if frame.ArtifactSHA256 != currentSHA {
		t.Fatalf("artifact_sha256 = %q, want normalized artifact SHA %q", frame.ArtifactSHA256, currentSHA)
	}
	if !strings.HasPrefix(frame.Checksum, "sha256:") || !strings.HasPrefix(frame.ArtifactSHA256, "sha256:") {
		t.Fatalf("visual frame checksums must be sha256-prefixed: checksum=%q artifact_sha256=%q", frame.Checksum, frame.ArtifactSHA256)
	}
}

func TestRunWritesGoldenOnlyWithExplicitMode(t *testing.T) {
	dir := t.TempDir()
	runtimePath := filepath.Join(dir, "surface-headless-block-system.json")
	outPath := filepath.Join(dir, "surface-visual-regression.json")
	currentPath := writeRGBAArtifact(t, filepath.Join(dir, "frames", "current.rgba"), 320, 200, 1280, "write-golden", false)
	currentSHA, err := sha256File(currentPath)
	if err != nil {
		t.Fatalf("hash current artifact: %v", err)
	}
	if err := os.WriteFile(runtimePath, validVisualDiffRuntimeReportJSON(t, false, currentSHA), 0o644); err != nil {
		t.Fatalf("write runtime report: %v", err)
	}
	goldenPath := filepath.Join(dir, "goldens", "initial.rgba")
	commonArgs := []string{
		"--runtime-report", runtimePath,
		"--required-target", "headless",
		"--frame-artifact", fmt.Sprintf("%s,%s,%d,%s", "examples/surface_block_system.tetra", "headless", 1, currentPath),
		"--golden-artifact", fmt.Sprintf("%s,%s,%d,%s", "examples/surface_block_system.tetra", "headless", 1, goldenPath),
		"--git-head", "c0258b63a636775b114d69d31cb7832fc3991b05",
		"--out", outPath,
	}
	if err := run(commonArgs); err == nil {
		t.Fatalf("expected missing golden artifact without --write-golden to fail")
	}
	args := append([]string{}, commonArgs...)
	args = append(args, "--write-golden")
	if err := run(args); err != nil {
		t.Fatalf("run surface-visual-diff --write-golden: %v", err)
	}
	if _, err := os.Stat(goldenPath); err != nil {
		t.Fatalf("expected write-golden to create golden artifact: %v", err)
	}
}

func TestRunGeneratesPolishedReferenceAppVisualEvidence(t *testing.T) {
	dir := t.TempDir()
	blockSystemEvidence := map[string]visualArtifactTestEvidence{}
	for _, target := range []string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"} {
		blockSystemEvidence[target] = writeVisualArtifactArgs(t, dir, "examples/surface_block_system.tetra", target, 1, false)
	}
	headlessPath := writeVisualDiffRuntimeReport(t, dir, "headless", "headless", false, blockSystemEvidence["headless"].CurrentSHA)
	linuxPath := writeVisualDiffRuntimeReport(t, dir, "linux-x64-real-window", "linux-x64", false, blockSystemEvidence["linux-x64-real-window"].CurrentSHA)
	webPath := writeVisualDiffRuntimeReport(t, dir, "wasm32-web-browser-canvas", "wasm32-web", false, blockSystemEvidence["wasm32-web-browser-canvas"].CurrentSHA)
	examplesPath := filepath.Join(dir, "surface-block-examples.json")
	if err := os.WriteFile(examplesPath, validBlockExamplesReportJSON(t, dir, false), 0o644); err != nil {
		t.Fatalf("write block examples report: %v", err)
	}
	outPath := filepath.Join(dir, "surface-visual-regression.json")

	args := []string{
		"--runtime-report", headlessPath,
		"--runtime-report", linuxPath,
		"--runtime-report", webPath,
		"--block-examples-report", examplesPath,
		"--required-target", "headless",
		"--required-target", "linux-x64-real-window",
		"--required-target", "wasm32-web-browser-canvas",
		"--git-head", "c0258b63a636775b114d69d31cb7832fc3991b05",
		"--out", outPath,
	}
	for _, target := range []string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"} {
		args = append(args, blockSystemEvidence[target].Args...)
		for _, source := range []string{
			"examples/surface_block_command_palette.tetra",
			"examples/surface_block_settings.tetra",
		} {
			args = append(args, writeVisualArtifactArgs(t, dir, source, target, 1, false).Args...)
		}
	}
	err := run(args)
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
	blockSystemEvidence := writeVisualArtifactArgs(t, dir, "examples/surface_block_system.tetra", "headless", 1, false)
	runtimePath := writeVisualDiffRuntimeReport(t, dir, "headless", "headless", false, blockSystemEvidence.CurrentSHA)
	examplesPath := filepath.Join(dir, "surface-block-examples.json")
	if err := os.WriteFile(examplesPath, validBlockExamplesReportJSON(t, dir, true), 0o644); err != nil {
		t.Fatalf("write block examples report: %v", err)
	}
	outPath := filepath.Join(dir, "surface-visual-regression.json")

	args := []string{
		"--runtime-report", runtimePath,
		"--block-examples-report", examplesPath,
		"--required-target", "headless",
		"--git-head", "c0258b63a636775b114d69d31cb7832fc3991b05",
		"--out", outPath,
	}
	args = append(args, blockSystemEvidence.Args...)
	err := run(args)
	if err == nil {
		t.Fatalf("expected missing polished reference app evidence to fail")
	}
	if !strings.Contains(err.Error(), "accessibility evidence") {
		t.Fatalf("error = %v, want accessibility evidence diagnostic", err)
	}
}

func writeVisualDiffRuntimeReport(t *testing.T, dir string, visualTargetName string, reportTarget string, drift bool, checksumOverride ...string) string {
	t.Helper()
	reportPath := filepath.Join(dir, visualTargetName+".json")
	var report surface.Report
	if err := json.Unmarshal(validVisualDiffRuntimeReportJSON(t, drift, checksumOverride...), &report); err != nil {
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

type visualArtifactTestEvidence struct {
	Args       []string
	CurrentSHA string
	GoldenSHA  string
}

func writeVisualArtifactArgs(t *testing.T, dir string, source string, target string, order int, drift bool) visualArtifactTestEvidence {
	t.Helper()
	currentPath := writeRGBAArtifact(t, filepath.Join(dir, "frames", safeArtifactName(source), target, fmt.Sprintf("%d.rgba", order)), 320, 200, 1280, source+"|"+target, false)
	goldenPath := writeRGBAArtifact(t, filepath.Join(dir, "goldens", safeArtifactName(source), target, fmt.Sprintf("%d.rgba", order)), 320, 200, 1280, source+"|"+target, drift)
	currentSHA, err := sha256File(currentPath)
	if err != nil {
		t.Fatalf("hash current artifact: %v", err)
	}
	goldenSHA, err := sha256File(goldenPath)
	if err != nil {
		t.Fatalf("hash golden artifact: %v", err)
	}
	return visualArtifactTestEvidence{
		Args: []string{
			"--frame-artifact", fmt.Sprintf("%s,%s,%d,%s", source, target, order, currentPath),
			"--golden-artifact", fmt.Sprintf("%s,%s,%d,%s", source, target, order, goldenPath),
		},
		CurrentSHA: currentSHA,
		GoldenSHA:  goldenSHA,
	}
}

func writeRGBAArtifact(t *testing.T, path string, width int, height int, stride int, seed string, drift bool) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create artifact dir: %v", err)
	}
	pixels := make([]byte, height*stride)
	base := byte(len(seed)%191 + 16)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := y*stride + x*4
			pixels[offset] = base
			pixels[offset+1] = byte((x + int(base)) % 255)
			pixels[offset+2] = byte((y + int(base)) % 255)
			pixels[offset+3] = 255
		}
	}
	if drift {
		for i := 0; i < len(pixels); i += 4 {
			pixels[i] = 255 - pixels[i]
			pixels[i+1] = 255 - pixels[i+1]
			pixels[i+2] = 255 - pixels[i+2]
		}
	}
	if err := os.WriteFile(path, pixels, 0o644); err != nil {
		t.Fatalf("write RGBA artifact: %v", err)
	}
	return path
}

func safeArtifactName(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ".", "_", ":", "_")
	return replacer.Replace(value)
}

func validVisualDiffRuntimeReportJSON(t *testing.T, drift bool, checksumOverride ...string) []byte {
	t.Helper()
	checksum := "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	if len(checksumOverride) > 0 && checksumOverride[0] != "" {
		checksum = checksumOverride[0]
	}
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
