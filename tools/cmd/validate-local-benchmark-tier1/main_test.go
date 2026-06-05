package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReportAcceptsCompleteP25Tier1Matrix(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := ValidateReportBytes(raw, dir); err != nil {
		t.Fatalf("ValidateReportBytes: %v", err)
	}
}

func TestValidateReportRejectsMissingMatrixRow(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	results[0]["rows"] = rows[:len(rows)-1]
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "missing matrix row") {
		t.Fatalf("ValidateReportBytes missing row = %v, want missing matrix row", err)
	}
}

func TestValidateReportRejectsMissingTetraMetadata(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	delete(rows[0], "tetra_metadata")
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "tetra metadata") {
		t.Fatalf("ValidateReportBytes missing Tetra metadata = %v, want tetra metadata", err)
	}
}

func TestValidateReportRejectsWeakClaimsAndUnknownClassification(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	report["non_claims"] = []string{"Tetra is the fastest language."}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "fastest-language") {
		t.Fatalf("ValidateReportBytes weak non-claims = %v, want fastest-language rejection", err)
	}

	report = validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	results[0]["classification"] = "wins everything"
	raw, err = json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "classification") {
		t.Fatalf("ValidateReportBytes unknown classification = %v, want classification rejection", err)
	}
}

func validTier1Report(t *testing.T, dir string) map[string]any {
	t.Helper()
	optimizer := fixture(t, dir, "artifacts/optimizer-validation.json", `{"status":"current_supported_subset"}`)
	results := make([]map[string]any, 0, len(requiredP20Categories))
	for _, category := range requiredP20Categories {
		rows := make([]map[string]any, 0, len(requiredLanguages))
		for _, language := range requiredLanguages {
			name := slug(category) + "_" + language
			row := map[string]any{
				"name":                 name,
				"category":             category,
				"language":             language,
				"status":               "measured",
				"compiler_version":     language + " compiler",
				"build_command":        []string{language, "build"},
				"run_command":          []string{filepath.Join("artifacts", name+".bin")},
				"source_path":          fixture(t, dir, "artifacts/"+name+".src", "source"),
				"binary_path":          fixture(t, dir, "artifacts/"+name+".bin", "binary"),
				"binary_size_bytes":    6,
				"compile_time_ms":      1.0,
				"run_measurements_ms":  []float64{1, 2, 3},
				"median_runtime_ms":    2.0,
				"raw_output_artifacts": []string{fixture(t, dir, "artifacts/"+name+".stdout.txt", "stdout"), fixture(t, dir, "artifacts/"+name+".stderr.txt", "")},
			}
			if language == "tetra" {
				row["tetra_metadata"] = map[string]any{
					"proof_report":                  fixture(t, dir, "artifacts/"+name+".proof.json", `{"kind":"proof"}`),
					"bounds_report":                 fixture(t, dir, "artifacts/"+name+".bounds.json", `{"kind":"bounds","totals":{"left":0}}`),
					"allocation_report":             fixture(t, dir, "artifacts/"+name+".alloc.json", `{"kind":"allocation_plan","totals":{"heap":0}}`),
					"perf_blocker_report":           fixture(t, dir, "artifacts/"+name+".perf.json", `{"kind":"perf","benchmarks":[]}`),
					"backend_report":                fixture(t, dir, "artifacts/"+name+".backend.json", `{"kind":"backend","summary":{"register_path":1,"stack_fallback":0}}`),
					"backend_path":                  "register",
					"bounds_left":                   0,
					"heap_allocations":              0,
					"perf_blockers":                 []string{},
					"optimizer_validation_metadata": map[string]any{"status": "current_supported_subset", "artifact": optimizer},
				}
			}
			rows = append(rows, row)
		}
		results = append(results, map[string]any{
			"category":              category,
			"algorithm_id":          "p25." + slug(category),
			"input_description":     "deterministic local Tier 1 fixture",
			"classification":        "comparable",
			"classification_reason": "fixture rows are within the comparable threshold",
			"rows":                  rows,
		})
	}
	return map[string]any{
		"schema":       schemaLocalBenchmarkTier1,
		"scope":        scopeP25RealLocalBenchmark,
		"generated_at": "2026-06-03T00:00:00Z",
		"host": map[string]any{
			"goos":       "linux",
			"goarch":     "amd64",
			"cpus":       8,
			"target_cpu": "test cpu",
			"git_commit": "abcdef",
		},
		"policy": map[string]any{
			"tier":                 "tier1_local_benchmark_evidence",
			"comparable_threshold": 0.20,
			"iterations":           3,
		},
		"non_claims": []string{
			"no fastest-language claim",
			"no official benchmark claim",
			"no cross-machine claim",
			"no TechEmpower claim",
			"no production claim",
		},
		"optimizer_validation": map[string]any{"status": "current_supported_subset", "artifact": optimizer},
		"results":              results,
	}
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
