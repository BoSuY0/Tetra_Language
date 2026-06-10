package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfaceperf"
)

func TestSurfacePerfSmokeWritesValidReport(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-perf-report.json")

	var stdout, stderr bytes.Buffer
	code := runSurfacePerfSmoke([]string{"--out", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := surfaceperf.ValidateReport(raw); err != nil {
		t.Fatalf("generated report did not validate: %v", err)
	}
	if !strings.Contains(stdout.String(), "surface performance smoke report") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestSurfacePerfSmokeRejectsUnsupportedSpeedClaimMode(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-perf-report.json")

	var stdout, stderr bytes.Buffer
	code := runSurfacePerfSmoke([]string{"--out", reportPath, "--claim-faster-than-electron"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected nonzero exit, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "Electron") {
		t.Fatalf("stderr = %q, want Electron claim rejection", stderr.String())
	}
	if raw, err := os.ReadFile(reportPath); err == nil {
		var report surfaceperf.Report
		if json.Unmarshal(raw, &report) == nil && report.ElectronComparison.FasterThanElectronClaim {
			t.Fatalf("unsupported report was written with faster-than-Electron claim")
		}
	}
}
