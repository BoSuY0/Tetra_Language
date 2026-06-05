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
	summary, err := os.ReadFile(filepath.Join(dir, "summary.md"))
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	for _, want := range []string{"# Memory Fuzz Short Summary", "tetra.memory-fuzz.oracle.v1", "Tier 1", "memory-fuzz-oracle.json"} {
		if !strings.Contains(string(summary), want) {
			t.Fatalf("summary missing %q:\n%s", want, summary)
		}
	}
}

func TestRunMemoryFuzzShortRejectsUnsupportedTier(t *testing.T) {
	err := runMemoryFuzzShort(memoryFuzzShortOptions{Tier: "2", ReportDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "Tier 1") {
		t.Fatalf("runMemoryFuzzShort tier error = %v, want Tier 1 rejection", err)
	}
}
