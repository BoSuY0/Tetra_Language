package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfaceelectron"
)

func TestSurfaceElectronComparisonCommandWritesValidReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "surface-electron-comparison-report.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runSurfaceElectronComparison([]string{"--out", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := surfaceelectron.ValidateReport(raw); err != nil {
		t.Fatalf("generated report failed validation: %v", err)
	}
}

func TestSurfaceElectronComparisonCommandRejectsOfficialBenchmarkFlag(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "surface-electron-comparison-report.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runSurfaceElectronComparison([]string{"--out", reportPath, "--claim-official-benchmark"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected command rejection, stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "official benchmark") {
		t.Fatalf("stderr = %q, want official benchmark rejection", stderr.String())
	}
}
