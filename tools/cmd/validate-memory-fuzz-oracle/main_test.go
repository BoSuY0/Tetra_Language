package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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

func TestValidateMemoryFuzzOracleReportFileRejectsMissingIslandProofFuzzSummary(t *testing.T) {
	dir := t.TempDir()
	writeTier1ArtifactBundle(t, dir)
	_ = os.Remove(filepath.Join(dir, "island-proof-fuzz-summary.json"))
	path := filepath.Join(dir, "memory-fuzz-oracle.json")
	err := validateMemoryFuzzOracleReportFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "island-proof-fuzz-summary.json") {
		t.Fatalf("validateMemoryFuzzOracleReportFile error = %v, want missing island proof fuzz summary", err)
	}
}

func TestValidateMemoryFuzzOracleReportFileRejectsMissingArtifactHashes(t *testing.T) {
	dir := t.TempDir()
	writeTier1ArtifactBundle(t, dir)
	if err := os.Remove(filepath.Join(dir, "artifact-hashes.json")); err != nil {
		t.Fatalf("remove artifact hashes: %v", err)
	}
	path := filepath.Join(dir, "memory-fuzz-oracle.json")
	err := validateMemoryFuzzOracleReportFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "artifact-hashes.json") {
		t.Fatalf("validateMemoryFuzzOracleReportFile error = %v, want missing artifact hashes", err)
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

func TestValidateMemoryFuzzOracleReportFileRejectsUnknownSummaryField(t *testing.T) {
	dir := t.TempDir()
	writeTier1ArtifactBundle(t, dir)
	summaryPath := filepath.Join(dir, "summary.json")
	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary json: %v", err)
	}
	raw = []byte(strings.Replace(string(raw), `"status": "pass",`, `"status": "pass", "unexpected": true,`, 1))
	if err := os.WriteFile(summaryPath, raw, 0o644); err != nil {
		t.Fatalf("write summary json: %v", err)
	}
	writeTier1ArtifactHashManifest(t, dir)
	path := filepath.Join(dir, "memory-fuzz-oracle.json")
	err = validateMemoryFuzzOracleReportFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("validateMemoryFuzzOracleReportFile error = %v, want strict summary json rejection", err)
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
    "artifact_hashes": "artifact-hashes.json",
    "island_proof_fuzz_summary": "island-proof-fuzz-summary.json",
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
	proofSummary := `{
  "schema_version": "tetra.island-proof-fuzz-summary.v1",
  "status": "pass",
  "corpus": "deterministic-short",
  "total": 11,
  "rejected": 11,
  "accepted": 0,
  "cases": [
    {"name": "malformed_proof_json", "status": "rejected"},
    {"name": "stale_epoch", "status": "rejected"},
    {"name": "mismatched_island_id", "status": "rejected"},
    {"name": "wrong_base_allocation", "status": "rejected"},
    {"name": "broken_dominance", "status": "rejected"},
    {"name": "missing_proof_id", "status": "rejected"},
    {"name": "wrong_operation", "status": "rejected"},
    {"name": "unsafe_unknown_promotion", "status": "rejected"},
    {"name": "noalias_broad_proof", "status": "rejected"},
    {"name": "storage_heap_fallback", "status": "rejected"},
    {"name": "transform_lost_metadata", "status": "rejected"}
  ]
}
`
	if err := os.WriteFile(filepath.Join(dir, "island-proof-fuzz-summary.json"), []byte(proofSummary), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTier1ArtifactHashManifest(t, dir)
}

func writeTier1ArtifactHashManifest(t *testing.T, dir string) {
	t.Helper()
	type hashedArtifact struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
		Schema string `json:"schema,omitempty"`
	}
	paths := []string{
		"island-proof-fuzz-summary.json",
		"memory-fuzz-oracle.json",
		"summary.json",
		"summary.md",
	}
	sort.Strings(paths)
	artifacts := make([]hashedArtifact, 0, len(paths))
	for _, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatalf("read artifact %s: %v", rel, err)
		}
		sum := sha256.Sum256(raw)
		artifacts = append(artifacts, hashedArtifact{
			Path:   rel,
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Size:   int64(len(raw)),
			Schema: testArtifactJSONSchema(raw),
		})
	}
	manifest := struct {
		Schema    string           `json:"schema"`
		Root      string           `json:"root"`
		Artifacts []hashedArtifact `json:"artifacts"`
	}{
		Schema:    "tetra.release-artifact-hashes.v1alpha1",
		Root:      ".",
		Artifacts: artifacts,
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "artifact-hashes.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write artifact hashes: %v", err)
	}
}

func testArtifactJSONSchema(raw []byte) string {
	var envelope struct {
		Schema        string `json:"schema"`
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	if envelope.Schema != "" {
		return envelope.Schema
	}
	return envelope.SchemaVersion
}
