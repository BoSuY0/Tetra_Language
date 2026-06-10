package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
)

func TestRunMemoryFuzzShortWritesValidatedArtifacts(t *testing.T) {
	dir := t.TempDir()
	if err := runMemoryFuzzShort(memoryFuzzShortOptions{Tier: "1", ReportDir: dir}); err != nil {
		t.Fatalf("runMemoryFuzzShort: %v", err)
	}
	reportPath := filepath.Join(dir, "memory-fuzz-oracle.json")
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report compiler.MemoryFuzzOracleReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse report: %v\n%s", err, raw)
	}
	if err := compiler.ValidateMemoryFuzzOracleReport(report); err != nil {
		t.Fatalf("ValidateMemoryFuzzOracleReport: %v\n%s", err, raw)
	}
	if len(report.Requirements) != 5 {
		t.Fatalf("requirements count = %d, want 5: %#v", len(report.Requirements), report.Requirements)
	}
	if len(report.SliceCoverage) != 12 {
		t.Fatalf("slice coverage count = %d, want v0-v11 coverage: %#v", len(report.SliceCoverage), report.SliceCoverage)
	}
	summary, err := os.ReadFile(filepath.Join(dir, "summary.md"))
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	for _, want := range []string{"# Memory Fuzz Short Summary", "tetra.memory-fuzz.oracle.v1", "Tier 1", "memory-fuzz-oracle.json", "MEM-FUZZ-001", "v0-v11"} {
		if !strings.Contains(string(summary), want) {
			t.Fatalf("summary missing %q:\n%s", want, summary)
		}
	}
	summaryJSONRaw, err := os.ReadFile(filepath.Join(dir, "summary.json"))
	if err != nil {
		t.Fatalf("read summary json: %v", err)
	}
	var summaryJSON struct {
		SchemaVersion string `json:"schema_version"`
		Kind          string `json:"kind"`
		Tier          string `json:"tier"`
		Status        string `json:"status"`
		Commands      []struct {
			Name    string `json:"name"`
			Command string `json:"command"`
			Status  string `json:"status"`
		} `json:"commands"`
		Artifacts map[string]string `json:"artifacts"`
	}
	if err := json.Unmarshal(summaryJSONRaw, &summaryJSON); err != nil {
		t.Fatalf("parse summary json: %v\n%s", err, summaryJSONRaw)
	}
	if summaryJSON.SchemaVersion != "tetra.memory-fuzz-short.summary.v1" || summaryJSON.Kind != "tier1_short_ci_smoke" || summaryJSON.Tier != "tier1_short_ci_smoke" || summaryJSON.Status != "pass" {
		t.Fatalf("summary json identity/status = %#v", summaryJSON)
	}
	for _, want := range []string{"oracle_report", "summary_md", "summary_json", "artifact_hashes"} {
		if summaryJSON.Artifacts[want] == "" {
			t.Fatalf("summary json missing artifact %q: %#v", want, summaryJSON.Artifacts)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "artifact-hashes.json")); err != nil {
		t.Fatalf("memory fuzz artifact hashes missing: %v", err)
	}
	var sawRunner, sawValidator bool
	for _, command := range summaryJSON.Commands {
		if command.Name == "memory-fuzz-short" && command.Status == "pass" && strings.Contains(command.Command, "go run ./tools/cmd/memory-fuzz-short") && strings.Contains(command.Command, "--report-dir") {
			sawRunner = true
		}
		if command.Name == "validate-memory-fuzz-oracle" && command.Status == "pass" && strings.Contains(command.Command, "go run ./tools/cmd/validate-memory-fuzz-oracle") && strings.Contains(command.Command, "--artifact-dir") {
			sawValidator = true
		}
	}
	if !sawRunner || !sawValidator {
		t.Fatalf("summary json commands missing runner/validator provenance: %#v", summaryJSON.Commands)
	}
	proofSummaryRaw, err := os.ReadFile(filepath.Join(dir, "island-proof-fuzz-summary.json"))
	if err != nil {
		t.Fatalf("read island proof fuzz summary: %v", err)
	}
	var proofSummary struct {
		SchemaVersion string `json:"schema_version"`
		Status        string `json:"status"`
		Total         int    `json:"total"`
		Rejected      int    `json:"rejected"`
		Accepted      int    `json:"accepted"`
		Cases         []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(proofSummaryRaw, &proofSummary); err != nil {
		t.Fatalf("parse island proof fuzz summary: %v\n%s", err, proofSummaryRaw)
	}
	if proofSummary.SchemaVersion != "tetra.island-proof-fuzz-summary.v1" || proofSummary.Status != "pass" {
		t.Fatalf("island proof fuzz identity/status = %#v", proofSummary)
	}
	if proofSummary.Total < 10 || proofSummary.Rejected != proofSummary.Total || proofSummary.Accepted != 0 {
		t.Fatalf("island proof fuzz counts = total %d rejected %d accepted %d", proofSummary.Total, proofSummary.Rejected, proofSummary.Accepted)
	}
	seenCases := map[string]bool{}
	for _, c := range proofSummary.Cases {
		if c.Status != "rejected" {
			t.Fatalf("island proof fuzz case %s status = %q", c.Name, c.Status)
		}
		seenCases[c.Name] = true
	}
	for _, want := range []string{"storage_heap_fallback", "transform_lost_metadata"} {
		if !seenCases[want] {
			t.Fatalf("island proof fuzz summary missing case %s: %#v", want, proofSummary.Cases)
		}
	}
}

func TestRunMemoryFuzzShortRejectsUnsupportedTier(t *testing.T) {
	err := runMemoryFuzzShort(memoryFuzzShortOptions{Tier: "2", ReportDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "Tier 1") {
		t.Fatalf("runMemoryFuzzShort tier error = %v, want Tier 1 rejection", err)
	}
}

func TestRunMemoryFuzzShortRejectsStaleReportDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stale-summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write stale artifact: %v", err)
	}
	err := runMemoryFuzzShort(memoryFuzzShortOptions{Tier: "1", ReportDir: dir})
	if err == nil || !strings.Contains(err.Error(), "fresh --report-dir") {
		t.Fatalf("runMemoryFuzzShort stale report dir error = %v, want fresh-dir rejection", err)
	}
}
