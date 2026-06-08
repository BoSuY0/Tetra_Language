package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
)

func TestValidateMemoryFuzzOracleReportFileAcceptsCompilerReport(t *testing.T) {
	report, err := compiler.BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "memory-fuzz-oracle.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMemoryFuzzOracleReportFile(path); err != nil {
		t.Fatalf("validateMemoryFuzzOracleReportFile: %v", err)
	}
}

func TestValidateMemoryFuzzOracleReportFileAcceptsTier1ArtifactBundle(t *testing.T) {
	dir := t.TempDir()
	writeTier1ArtifactBundle(t, dir)
	path := filepath.Join(dir, "memory-fuzz-oracle.json")
	if err := validateMemoryFuzzOracleReportFile(path, dir); err != nil {
		t.Fatalf("validateMemoryFuzzOracleReportFile artifact bundle: %v", err)
	}
}

func TestValidateMemoryFuzzOracleReportFileRejectsInvalidReport(t *testing.T) {
	report, err := compiler.BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	report.Rows = report.Rows[1:]
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "memory-fuzz-oracle.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	err = validateMemoryFuzzOracleReportFile(path)
	if err == nil || !strings.Contains(err.Error(), "missing oracle_category") {
		t.Fatalf("validateMemoryFuzzOracleReportFile error = %v, want missing oracle_category", err)
	}
}

func TestValidateMemoryFuzzOracleReportFileRejectsMissingV12ReleaseEvidence(t *testing.T) {
	report, err := compiler.BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	report.Requirements = report.Requirements[1:]
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "memory-fuzz-oracle.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	err = validateMemoryFuzzOracleReportFile(path)
	if err == nil || !strings.Contains(err.Error(), "missing requirement MEM-FUZZ-001") {
		t.Fatalf("validateMemoryFuzzOracleReportFile error = %v, want missing MEM-FUZZ-001", err)
	}
}

func TestValidateMemoryFuzzOracleReportFileRejectsMissingArtifactSummary(t *testing.T) {
	dir := t.TempDir()
	writeTier1ArtifactBundle(t, dir)
	if err := os.Remove(filepath.Join(dir, "summary.md")); err != nil {
		t.Fatalf("remove summary: %v", err)
	}
	path := filepath.Join(dir, "memory-fuzz-oracle.json")
	err := validateMemoryFuzzOracleReportFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "summary.md") {
		t.Fatalf("validateMemoryFuzzOracleReportFile error = %v, want missing summary.md", err)
	}
}

func TestValidateMemoryFuzzOracleReportFileRejectsMissingValidatorProvenance(t *testing.T) {
	dir := t.TempDir()
	writeTier1ArtifactBundle(t, dir)
	summaryPath := filepath.Join(dir, "summary.json")
	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary json: %v", err)
	}
	raw = []byte(strings.ReplaceAll(string(raw), "--artifact-dir", "--missing-artifact-dir"))
	if err := os.WriteFile(summaryPath, raw, 0o644); err != nil {
		t.Fatalf("write summary json: %v", err)
	}
	path := filepath.Join(dir, "memory-fuzz-oracle.json")
	err = validateMemoryFuzzOracleReportFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "validate-memory-fuzz-oracle") {
		t.Fatalf("validateMemoryFuzzOracleReportFile error = %v, want validator command provenance rejection", err)
	}
}

func writeTier1ArtifactBundle(t *testing.T, dir string) {
	t.Helper()
	report, err := compiler.BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(dir, "memory-fuzz-oracle.json")
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "summary.md"), []byte("# Memory Fuzz Short Summary\n\n- tier: `Tier 1 short CI smoke`\n- report: `"+filepath.ToSlash(reportPath)+"`\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	summaryJSON := `{
  "schema_version": "tetra.memory-fuzz-short.summary.v1",
  "kind": "tier1_short_ci_smoke",
  "tier": "tier1_short_ci_smoke",
  "status": "pass",
  "artifacts": {
    "oracle_report": "memory-fuzz-oracle.json",
    "summary_md": "summary.md",
    "summary_json": "summary.json"
  },
  "commands": [
    {"name": "memory-fuzz-short", "command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir <artifact-dir>", "status": "pass"},
    {"name": "validate-memory-fuzz-oracle", "command": "go run ./tools/cmd/validate-memory-fuzz-oracle --report <artifact-dir>/memory-fuzz-oracle.json --artifact-dir <artifact-dir>", "status": "pass"}
  ]
}
`
	if err := os.WriteFile(filepath.Join(dir, "summary.json"), []byte(summaryJSON), 0o644); err != nil {
		t.Fatal(err)
	}
}
