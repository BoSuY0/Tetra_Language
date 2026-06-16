package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tetra_language/tools/internal/heaptelemetry"
	"tetra_language/tools/internal/zeroheapbench"
)

func TestValidateReportAcceptsCompleteZeroHeapSuite(t *testing.T) {
	dir := t.TempDir()
	report := validZeroHeapReport(t, dir)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := ValidateReportBytes(raw, dir); err != nil {
		t.Fatalf("ValidateReportBytes: %v", err)
	}
}

func TestValidateReportRejectsZeroHeapSuiteRuntimeHeapRegression(t *testing.T) {
	dir := t.TempDir()
	report := validZeroHeapReport(t, dir)
	results := report["results"].([]map[string]any)
	metadata := results[0]["tetra_metadata"].(map[string]any)
	metadata["heap_allocations"] = 1
	memory := metadata["memory_evidence"].(map[string]any)
	memory["heap_alloc_bytes"] = runtimeHeapMetricFixture(
		heapSidecarFixture(t, dir, results[0]["name"].(string), 0, 0, 64, 1),
		0,
		0,
		64,
		1,
	)

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "zero-heap") || !strings.Contains(err.Error(), "heap_allocations") {
		t.Fatalf("ValidateReportBytes heap regression = %v, want zero-heap heap_allocations rejection", err)
	}
}

func TestValidateReportRejectsNonTetraZeroHeapRow(t *testing.T) {
	dir := t.TempDir()
	report := validZeroHeapReport(t, dir)
	results := report["results"].([]map[string]any)
	results[0]["language"] = "c"

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "tetra-only") {
		t.Fatalf("ValidateReportBytes non-Tetra row = %v, want tetra-only rejection", err)
	}
}

func validZeroHeapReport(t *testing.T, dir string) map[string]any {
	t.Helper()
	results := make([]map[string]any, 0, len(zeroheapbench.Categories))
	for _, category := range zeroheapbench.Categories {
		name := zeroheapbench.Slug(category) + "_tetra"
		source := fixture(t, dir, filepath.Join("artifacts", "src", "zero-heap", zeroheapbench.Slug(category)+".tetra"), "source")
		binary := fixture(t, dir, filepath.Join("artifacts", "bin", name), "binary")
		heapSidecar := heapSidecarFixture(t, dir, name, 0, 0, 0, 0)
		alloc := fixture(t, dir, filepath.Join("artifacts", "bin", name+".alloc.json"), `{"totals":{"heap":0},"summary":{"bytes_requested":128,"bytes_reserved":128,"domains":[]}}`)
		result := map[string]any{
			"name":                 name,
			"category":             category,
			"algorithm_id":         "zero_heap." + zeroheapbench.Slug(category),
			"input_description":    "fixture zero-heap row",
			"language":             "tetra",
			"status":               "measured",
			"compiler_version":     "tetra test",
			"build_command":        []string{"tetra", "build"},
			"run_command":          []string{binary},
			"source_path":          source,
			"binary_path":          binary,
			"binary_size_bytes":    6,
			"compile_time_ms":      1.0,
			"run_measurements_ms":  []float64{1, 2, 3},
			"median_runtime_ms":    2.0,
			"raw_output_artifacts": []string{fixture(t, dir, filepath.Join("artifacts", "raw", name+".stdout.txt"), "stdout")},
			"tetra_metadata": map[string]any{
				"proof_report":        fixture(t, dir, filepath.Join("artifacts", "bin", name+".proof.json"), `{"kind":"proof"}`),
				"bounds_report":       fixture(t, dir, filepath.Join("artifacts", "bin", name+".bounds.json"), `{"totals":{"left":0}}`),
				"allocation_report":   alloc,
				"perf_blocker_report": fixture(t, dir, filepath.Join("artifacts", "bin", name+".perf.json"), `{"benchmarks":[]}`),
				"backend_report":      fixture(t, dir, filepath.Join("artifacts", "bin", name+".backend.json"), `{"summary":{"register_path":1}}`),
				"backend_path":        "register",
				"bounds_left":         0,
				"heap_allocations":    0,
				"perf_blockers":       []string{},
				"memory_evidence": map[string]any{
					"schema":                "tetra.local_benchmark.memory_evidence.v1",
					"heap_alloc_bytes":      runtimeHeapMetricFixture(heapSidecar, 0, 0, 0, 0),
					"bytes_requested":       allocationMetricFixture(128, alloc),
					"bytes_reserved":        allocationMetricFixture(128, alloc),
					"bytes_committed":       unsupportedMetricFixture("allocation report does not expose committed bytes"),
					"bytes_copied":          allocationMetricFixture(0, alloc),
					"rss_current":           unsupportedMetricFixture("zero-heap suite does not measure process RSS"),
					"rss_peak":              unsupportedMetricFixture("zero-heap suite does not measure process RSS"),
					"domain_bytes_evidence": unsupportedMetricFixture("allocation report summary does not include memory domains"),
					"domain_bytes":          []map[string]any{},
				},
			},
		}
		results = append(results, result)
	}
	return map[string]any{
		"schema":       zeroheapbench.Schema,
		"scope":        zeroheapbench.Scope,
		"generated_at": "2026-06-16T00:00:00Z",
		"policy": map[string]any{
			"suite":      "zero_heap_microbenchmarks",
			"iterations": 3,
		},
		"non_claims": []string{
			"no official benchmark claim",
			"no cross-language performance claim",
			"no zero RSS claim",
			"no universal zero heap claim",
		},
		"results": results,
	}
}

func runtimeHeapMetricFixture(sidecar string, current uint64, peak uint64, total uint64, count uint64) map[string]any {
	return map[string]any{
		"bytes":             peak,
		"current_bytes":     current,
		"peak_bytes":        peak,
		"total_alloc_bytes": total,
		"allocation_count":  count,
		"evidence_class":    "runtime_measured",
		"method":            heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		"source_artifact":   sidecar,
	}
}

func allocationMetricFixture(bytes uint64, source string) map[string]any {
	return map[string]any{
		"bytes":           bytes,
		"evidence_class":  "allocation_report_estimate",
		"method":          "allocation_report_summary",
		"source_artifact": source,
	}
}

func unsupportedMetricFixture(reason string) map[string]any {
	return map[string]any{
		"evidence_class":     "unsupported",
		"method":             "not_collected",
		"unsupported_reason": reason,
	}
}

func heapSidecarFixture(t *testing.T, dir string, name string, current uint64, peak uint64, total uint64, count uint64) string {
	t.Helper()
	data := map[string]any{
		"schema":                 heaptelemetry.Schema,
		"target":                 heaptelemetry.TargetLinuxX64,
		"method":                 heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		"program":                name,
		"pid":                    1234,
		"exit_status":            0,
		"heap_current_bytes":     current,
		"heap_peak_bytes":        peak,
		"heap_total_alloc_bytes": total,
		"heap_allocation_count":  count,
		"bytes_requested":        total,
		"bytes_reserved":         uint64(0),
	}
	if peak > 0 {
		data["bytes_reserved"] = uint64(4096)
	}
	if count > 0 {
		data["allocation_paths"] = map[string]uint64{"small_heap_bump": count}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal heap sidecar: %v", err)
	}
	return fixture(t, dir, filepath.Join("artifacts", "heap-telemetry", name, "iteration-01.heap.json"), string(raw))
}

func fixture(t *testing.T, dir string, rel string, content string) string {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", rel, err)
	}
	return rel
}
