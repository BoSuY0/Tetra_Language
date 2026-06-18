package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidatesSurfacePerformanceBudgetReport(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-performance-budget.json")
	if err := os.WriteFile(
		reportPath,
		[]byte(validSurfacePerformanceBudgetReportJSON()),
		0o644,
	); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := run([]string{"--report", reportPath}); err != nil {
		t.Fatalf("run failed: %v", err)
	}
}

func TestRunRejectsFasterThanElectronClaim(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-performance-budget.json")
	raw := strings.Replace(
		validSurfacePerformanceBudgetReportJSON(),
		`"performance_claim":"none"`,
		`"performance_claim":"faster than Electron"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"electron_comparison":"none"`,
		`"electron_comparison":"faster than Electron"`,
		1,
	)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := run([]string{"--report", reportPath})
	if err == nil {
		t.Fatalf("expected faster-than-Electron claim to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "faster than electron") {
		t.Fatalf("error = %v, want faster than Electron diagnostic", err)
	}
}

func validSurfacePerformanceBudgetReportJSON() string {
	return `{"schema":"tetra.surface.performance-budget.v1","model":"surface-performance-budget-v1","release_scope":"surface-v1-linux-web","source":"examples/surface/toolkit/surface_linux_app_shell_notes.tetra","target":"linux-x64","runtime":"surface-linux-x64","production_claim":true,"experimental":false,"git_head":"0123456789abcdef0123456789abcdef01234567","performance_claim":"none","startup":{"launch_to_first_frame_ms":18,"budget_ms":250,"trace":"local-startup-trace-v1","pass":true},"frame":{"frame_count":3,"p50_build_ms":4,"p95_build_ms":7,"p50_present_ms":3,"p95_present_ms":6,"budget_ms":16,"idle_loop_count":24,"work_loop_count":6,"pass":true},"scene":{"block_count":3,"recipe_expansion_count":0,"paint_command_count":10,"layout_pass_count":4,"text_run_count":2},"memory":{"glyph_cache_bytes":4096,"asset_cache_bytes":5376,"layout_cache_bytes":4096,"paint_cache_bytes":10240,"framebuffer_peak_bytes":1555200,"framebuffer_total_bytes":2880000,"rss_measured":false,"peak_rss_bytes":0,"allocation_count":42,"allocation_bytes":2903808,"bounded_caches":true,"unbounded_cache_rejected":true,"pass":true},"binary":{"artifact_path":"/tmp/surface-artifacts/surface-linux-app-shell-notes","size_bytes":90001,"budget_bytes":16777216,"pass":true},"cpu_power_proxy":{"idle_loop_count":24,"work_loop_count":6,"idle_frame_count":2,"work_frame_count":1,"real_power_measured":false,"pass":true},"cache":{"glyph_cache_budget_bytes":65536,"asset_cache_budget_bytes":65536,"layout_cache_budget_bytes":65536,"paint_cache_budget_bytes":65536,"total_cache_bytes":23808,"total_cache_budget_bytes":262144,"eviction":"bounded-lru","pass":true},"methodology":{"kind":"local-deterministic-budget-v1","electron_comparison":"none","official_benchmark":false,"cross_machine":false,"fair_comparison_required_for_electron_claim":true},"unsupported_claims":["faster-than-electron","lower-power-than-electron","official-benchmark-result","cross-machine-benchmark","electron-parity-performance"],"negative_guards":{"bounded_caches":true,"unbounded_cache_rejected":true,"stale_report_rejected":true,"no_faster_than_electron_claim":true,"no_benchmark_parity_claim":true,"peak_memory_field_required":true,"no_official_benchmark_claim":true}}` + "\n"
}
