package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfaceelectron"
)

func TestValidateSurfaceElectronComparisonCommandAcceptsValidReport(t *testing.T) {
	reportPath := writeComparisonFixture(t, commandComparisonReport())
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runValidateSurfaceElectronComparisonReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "surface electron comparison report OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestValidateSurfaceElectronComparisonCommandRejectsOfficialBenchmarkClaim(t *testing.T) {
	report := commandComparisonReport()
	report.Positioning.OfficialBenchmarkClaim = true
	report.NegativeGuards.OfficialBenchmarkClaimRejected = false
	reportPath := writeComparisonFixture(t, report)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runValidateSurfaceElectronComparisonReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected official benchmark claim rejection")
	}
	if !strings.Contains(stderr.String(), "official benchmark") {
		t.Fatalf("stderr = %q, want official benchmark rejection", stderr.String())
	}
}

func writeComparisonFixture(t *testing.T, report surfaceelectron.Report) string {
	t.Helper()
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "surface-electron-comparison-report.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func commandComparisonReport() surfaceelectron.Report {
	return surfaceelectron.ValidFixtureReport()
}
