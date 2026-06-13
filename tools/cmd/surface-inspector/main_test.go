package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	reportPath := filepath.Join(dir, "surface-inspector.json")
	htmlPath := filepath.Join(dir, "surface-inspector.html")
	if err := run([]string{
		"--runtime-report", "block:" + blockPath,
		"--runtime-report", "morph:" + morphPath,
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
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	for _, want := range []string{"block_tree", "morph_tokens", "layout", "paint", "accessibility", "event_routes", "focus", "perf_counters"} {
		got := report.Sections[want]
		if !got.Present || got.Count == 0 {
			t.Fatalf("section %s = %#v, want present with count", want, got)
		}
	}
}

func TestRunRejectsHiddenStateInInputReports(t *testing.T) {
	dir := t.TempDir()
	blockPath := filepath.Join(dir, "block.json")
	if err := os.WriteFile(blockPath, []byte(`{"schema":"tetra.surface.runtime.v1","status":"pass","target":"headless","source":"examples/surface_block_system.tetra","hidden_state":true}`), 0o644); err != nil {
		t.Fatalf("write block report: %v", err)
	}
	err := run([]string{"--runtime-report", "block:" + blockPath, "--out", filepath.Join(dir, "surface-inspector.json")})
	if err == nil {
		t.Fatalf("expected hidden state input to fail")
	}
}

func minimalInspectorInputBlockJSON() string {
	return `{"schema":"tetra.surface.runtime.v1","status":"pass","target":"headless","source":"examples/surface_block_system.tetra","block_graph":{"nodes":[{"id":1},{"id":2}]},"layout_passes":[{"order":1},{"order":2}],"paint_commands":[{"order":1},{"order":2}],"block_event_routes":[{"order":1},{"order":2}],"block_focus_transitions":[{"order":1}],"block_accessibility_tree":{"nodes":[{"id":1},{"id":2}]},"surface_performance_budget":{"schema":"tetra.surface.performance-budget.v1","model":"surface-performance-budget-v1"}}`
}

func minimalInspectorInputMorphJSON() string {
	return `{"schema":"tetra.surface.runtime.v1","status":"pass","target":"headless","source":"examples/surface_morph_command_palette.tetra","morph":{"schema":"tetra.surface.morph.v1","token_graph":{"tokens":[{"name":"color.accent"},{"name":"space.2"}]},"recipes":[{"name":"panel"}]}}`
}

func minimalInspectorInputAppModelJSON() string {
	return `{"schema":"tetra.surface.runtime.v1","status":"pass","target":"headless","source":"examples/surface_app_model.tetra","app_model":{"schema":"tetra.surface.app-model.v1","event_bindings":[{"event":"key"}],"focus_scopes":[{"id":"modal"}],"async_tasks":[{"name":"load"}]}}`
}

func minimalInspectorInputAccessibilityJSON() string {
	return `{"schema":"tetra.surface.runtime.v1","status":"pass","target":"headless","source":"examples/surface_release_accessibility.tetra","accessibility_tree":{"schema":"tetra.surface.accessibility-tree.v1","nodes":[{"id":1},{"id":2}],"snapshots":[{"name":"initial"}]}}`
}
