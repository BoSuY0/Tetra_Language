package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"tetra_language/compiler"
)

const memoryFuzzTestHead = "0123456789abcdef0123456789abcdef01234567"

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

func TestValidateMemoryFuzzOracleReportFileRejectsMissingGitHeadWhenSameCommitRequired(t *testing.T) {
	dir := t.TempDir()
	writeTier1ArtifactBundle(t, dir)
	path := filepath.Join(dir, "memory-fuzz-oracle.json")
	err := validateMemoryFuzzOracleReportFileWithOptions(memoryFuzzOracleValidationOptions{
		ReportPath:     path,
		ArtifactDir:    dir,
		CurrentGitHead: memoryFuzzTestHead,
	})
	if err == nil {
		t.Fatalf("expected missing git_head to fail when same-commit validation is required")
	}
	if !strings.Contains(err.Error(), "git_head") {
		t.Fatalf("validateMemoryFuzzOracleReportFileWithOptions error = %v, want git_head rejection", err)
	}
}

func TestValidateMemoryFuzzOracleReportFileAcceptsSameCommitGitHead(t *testing.T) {
	dir := t.TempDir()
	writeTier1ArtifactBundle(t, dir)
	path := filepath.Join(dir, "memory-fuzz-oracle.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report compiler.MemoryFuzzOracleReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse report: %v", err)
	}
	report.GitHead = memoryFuzzTestHead
	raw, err = json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	writeTier1ArtifactHashManifest(t, dir)

	if err := validateMemoryFuzzOracleReportFileWithOptions(memoryFuzzOracleValidationOptions{
		ReportPath:     path,
		ArtifactDir:    dir,
		CurrentGitHead: memoryFuzzTestHead,
	}); err != nil {
		t.Fatalf("validateMemoryFuzzOracleReportFileWithOptions same commit: %v", err)
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

func TestValidateMemoryFuzzOracleReportFileRejectsMissingRequiredReproducerDirs(t *testing.T) {
	dir := t.TempDir()
	writeTier1ArtifactBundle(t, dir)
	if err := os.RemoveAll(filepath.Join(dir, "reproducers", "compiler-crash")); err != nil {
		t.Fatalf("remove compiler crash reproducer dir: %v", err)
	}
	writeTier1ArtifactHashManifest(t, dir)

	path := filepath.Join(dir, "memory-fuzz-oracle.json")
	err := validateMemoryFuzzOracleReportFile(path, dir)
	if err == nil {
		t.Fatalf("expected missing required reproducer dir to fail")
	}
	if !strings.Contains(err.Error(), "reproducers/compiler-crash") {
		t.Fatalf("validateMemoryFuzzOracleReportFile error = %v, want required reproducer dir rejection", err)
	}
}

func TestValidateMemoryFuzzOracleReportFileRejectsMissingFailureClassificationCounts(t *testing.T) {
	dir := t.TempDir()
	writeTier1ArtifactBundle(t, dir)
	summaryPath := filepath.Join(dir, "summary.json")
	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary json: %v", err)
	}
	raw = []byte(strings.Replace(string(raw), `  "unclassified_failures": 0,`+"\n", "", 1))
	if err := os.WriteFile(summaryPath, raw, 0o644); err != nil {
		t.Fatalf("write summary json: %v", err)
	}
	writeTier1ArtifactHashManifest(t, dir)

	path := filepath.Join(dir, "memory-fuzz-oracle.json")
	err = validateMemoryFuzzOracleReportFile(path, dir)
	if err == nil {
		t.Fatalf("expected missing unclassified failure count to fail")
	}
	if !strings.Contains(err.Error(), "unclassified_failures") {
		t.Fatalf("validateMemoryFuzzOracleReportFile error = %v, want unclassified_failures rejection", err)
	}
}

func TestValidateMemoryFuzzOracleReportFileRejectsUnclassifiedFailures(t *testing.T) {
	dir := t.TempDir()
	writeTier1ArtifactBundle(t, dir)
	summaryPath := filepath.Join(dir, "summary.json")
	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary json: %v", err)
	}
	raw = []byte(strings.Replace(string(raw), `"unclassified_failures": 0`, `"unclassified_failures": 1`, 1))
	if err := os.WriteFile(summaryPath, raw, 0o644); err != nil {
		t.Fatalf("write summary json: %v", err)
	}
	writeTier1ArtifactHashManifest(t, dir)

	path := filepath.Join(dir, "memory-fuzz-oracle.json")
	err = validateMemoryFuzzOracleReportFile(path, dir)
	if err == nil {
		t.Fatalf("expected unclassified failure count to fail")
	}
	if !strings.Contains(err.Error(), "unclassified_failures") {
		t.Fatalf("validateMemoryFuzzOracleReportFile error = %v, want unclassified_failures rejection", err)
	}
}

func TestValidateMemoryFuzzOracleReportFileRejectsMissingReproducibilitySeeds(t *testing.T) {
	dir := t.TempDir()
	writeTier1ArtifactBundle(t, dir)
	summaryPath := filepath.Join(dir, "summary.json")
	raw, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary json: %v", err)
	}
	var summary map[string]any
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatalf("parse summary json: %v", err)
	}
	delete(summary, "reproducibility_seeds")
	raw, err = json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(summaryPath, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write summary json: %v", err)
	}
	writeTier1ArtifactHashManifest(t, dir)

	path := filepath.Join(dir, "memory-fuzz-oracle.json")
	err = validateMemoryFuzzOracleReportFile(path, dir)
	if err == nil {
		t.Fatalf("expected missing reproducibility seeds to fail")
	}
	if !strings.Contains(err.Error(), "reproducibility_seeds") {
		t.Fatalf("validateMemoryFuzzOracleReportFile error = %v, want reproducibility_seeds rejection", err)
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
  "observed_failures": 0,
  "classified_failures": 0,
  "unclassified_failures": 0,
  "release_blocking_failures": 0,
  "reproducibility_seeds": [
    "memory-fuzz:v0:seed:1000",
    "memory-fuzz:v1:seed:1001",
    "memory-fuzz:v2:seed:1002",
    "memory-fuzz:v3:seed:1003",
    "memory-fuzz:v4:seed:1004",
    "memory-fuzz:v5:seed:1005",
    "memory-fuzz:v6:seed:1006",
    "memory-fuzz:v7:seed:1007",
    "memory-fuzz:v8:seed:1008",
    "memory-fuzz:v9:seed:1009",
    "memory-fuzz:v10:seed:1010",
    "memory-fuzz:v11:seed:1011"
  ],
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
	writeTier1RequiredReproducerDirs(t, dir)
	writeTier1ArtifactHashManifest(t, dir)
}

func writeTier1RequiredReproducerDirs(t *testing.T, dir string) {
	t.Helper()
	for _, rel := range []string{"reproducers/compiler-crash", "reproducers/miscompile", "reducers/miscompile"} {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("create required reproducer dir %s: %v", rel, err)
		}
		if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("required release evidence slot for "+rel+"\n"), 0o644); err != nil {
			t.Fatalf("write required reproducer marker %s: %v", rel, err)
		}
	}
}

func writeTier1ArtifactHashManifest(t *testing.T, dir string) {
	t.Helper()
	type hashedArtifact struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
		Schema string `json:"schema,omitempty"`
	}
	var artifacts []hashedArtifact
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "artifact-hashes.json" {
			return nil
		}
		raw, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			return fmt.Errorf("read artifact %s: %w", rel, err)
		}
		sum := sha256.Sum256(raw)
		artifacts = append(artifacts, hashedArtifact{
			Path:   rel,
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Size:   int64(len(raw)),
			Schema: testArtifactJSONSchema(raw),
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
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
