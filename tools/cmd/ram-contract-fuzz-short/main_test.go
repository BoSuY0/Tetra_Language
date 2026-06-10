package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunRAMContractFuzzShortWritesValidatedArtifacts(t *testing.T) {
	dir := t.TempDir()
	if err := runRAMContractFuzzShort(dir, "e2c19b8ee276158f8eb2c54cf61e11bd84952893"); err != nil {
		t.Fatalf("runRAMContractFuzzShort: %v", err)
	}
	for _, name := range []string{
		"ram-contract-report.json",
		"memory-grade-report.json",
		"proof-store-summary.json",
		"validation-pipeline-coverage.json",
		"heap-blockers.json",
		"copy-blockers.json",
		"ram-contract-fuzz-oracle.json",
		"ram-contract-fuzz-summary.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}
