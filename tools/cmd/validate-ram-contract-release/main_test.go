package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRAMContractReleaseRejectsMissingReport(t *testing.T) {
	dir := t.TempDir()
	err := validateRAMContractRelease(dir, "")
	if err == nil || !strings.Contains(err.Error(), "ram-contract-report.json") {
		t.Fatalf("validateRAMContractRelease error = %v, want missing report", err)
	}
}

func TestValidateReleaseHashManifestRejectsMissingRAMArtifact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact-hashes.json")
	raw := `{"schema":"tetra.release-artifact-hashes.v1alpha1","root":".","artifacts":[]}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateReleaseHashManifest(path)
	if err == nil || !strings.Contains(err.Error(), "ram-contract-report.json") {
		t.Fatalf("validateReleaseHashManifest error = %v, want RAM artifact rejection", err)
	}
}
