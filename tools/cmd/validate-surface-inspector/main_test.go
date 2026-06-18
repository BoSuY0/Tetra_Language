package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidatesSurfaceInspectorReport(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-inspector.json")
	if err := os.WriteFile(reportPath, []byte(validSurfaceInspectorReportJSON()), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := run([]string{"--report", reportPath}); err != nil {
		t.Fatalf("run failed: %v", err)
	}
}

func TestRunRejectsDOMRuntimeDependency(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-inspector.json")
	raw := strings.Replace(
		validSurfaceInspectorReportJSON(),
		`"no_dom_runtime_dependency":true`,
		`"no_dom_runtime_dependency":false`,
		1,
	)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := run([]string{"--report", reportPath})
	if err == nil {
		t.Fatalf("expected DOM runtime dependency to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "dom") {
		t.Fatalf("error = %v, want DOM dependency diagnostic", err)
	}
}

func validSurfaceInspectorReportJSON() string {
	return `{"schema":"tetra.surface.inspector.v1","model":"surface-inspector-v1","release_scope":"surface-v1-linux-web","producer":"tools/cmd/surface-inspector","source":"examples/surface/block_core/surface_block_system.tetra","target":"headless","mode":"static-tool-report","input_reports":[{"kind":"block","path":"reports/surface-inspector/inputs/surface-headless-block-system.json","schema":"tetra.surface.runtime.v1","source":"examples/surface/block_core/surface_block_system.tetra","target":"headless","pass":true},{"kind":"morph","path":"reports/surface-inspector/inputs/surface-headless-morph.json","schema":"tetra.surface.runtime.v1","source":"examples/surface/morph_core/surface_morph_command_palette.tetra","target":"headless","pass":true},{"kind":"accessibility","path":"reports/surface-inspector/inputs/surface-headless-release-accessibility.json","schema":"tetra.surface.runtime.v1","source":"examples/surface/release/surface_release_accessibility.tetra","target":"headless","pass":true},{"kind":"app-model","path":"reports/surface-inspector/inputs/surface-headless-app-model.json","schema":"tetra.surface.runtime.v1","source":"examples/surface/toolkit/surface_app_model.tetra","target":"headless","pass":true}],"source_locations":[{"kind":"block","path":"examples/surface/block_core/surface_block_system.tetra","line":1,"column":1},{"kind":"morph","path":"examples/surface/morph_core/surface_morph_command_palette.tetra","line":1,"column":1},{"kind":"accessibility","path":"examples/surface/release/surface_release_accessibility.tetra","line":1,"column":1},{"kind":"app-model","path":"examples/surface/toolkit/surface_app_model.tetra","line":1,"column":1}],"sections":{"block_tree":{"present":true,"count":6,"source":"block_graph.nodes"},"morph_tokens":{"present":true,"count":22,"source":"morph.token_graph.tokens"},"layout":{"present":true,"count":6,"source":"layout_passes"},"paint":{"present":true,"count":10,"source":"paint_commands"},"accessibility":{"present":true,"count":12,"source":"accessibility_tree.nodes"},"event_routes":{"present":true,"count":5,"source":"block_event_routes"},"focus":{"present":true,"count":3,"source":"block_focus_transitions"},"perf_counters":{"present":true,"count":4,"source":"surface_performance_budget"}},"static_artifacts":{"json":"reports/surface-inspector/surface-inspector.json","html":"reports/surface-inspector/surface-inspector.html","html_tool_report":true},"hidden_state":{"scanned":true,"findings":[]},"negative_guards":{"no_dom_runtime_dependency":true,"no_browser_devtools_dependency":true,"no_react_devtools_dependency":true,"static_html_tool_report_only":true,"no_hidden_state":true},"pass":true}` + "\n"
}
