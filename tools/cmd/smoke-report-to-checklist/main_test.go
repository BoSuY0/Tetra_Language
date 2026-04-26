package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateSmokeReportCountsAcceptsConsistentCounts(t *testing.T) {
	total := 2
	passed := 1
	failed := 1
	report := &smokeReport{
		Total:  &total,
		Passed: &passed,
		Failed: &failed,
		Cases: []smokeCaseReport{
			{Name: "ok", Pass: true},
			{Name: "bad", Pass: false},
		},
	}
	if err := validateSmokeReportCounts(report); err != nil {
		t.Fatalf("validateSmokeReportCounts: %v", err)
	}
}

func TestValidateSmokeReportCountsRejectsMismatchedCounts(t *testing.T) {
	total := 2
	passed := 2
	failed := 0
	report := &smokeReport{
		Total:  &total,
		Passed: &passed,
		Failed: &failed,
		Cases: []smokeCaseReport{
			{Name: "ok", Pass: true},
			{Name: "bad", Pass: false},
		},
	}
	err := validateSmokeReportCounts(report)
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
}

func TestValidateSmokeReportCountsAcceptsLegacyReportsWithoutCounts(t *testing.T) {
	report := &smokeReport{
		Cases: []smokeCaseReport{
			{Name: "ok", Pass: true},
			{Name: "bad", Pass: false},
		},
	}
	if err := validateSmokeReportCounts(report); err != nil {
		t.Fatalf("legacy report should remain accepted: %v", err)
	}
}

func TestValidateSmokeReportCountsRejectsExplicitZeroCountsWithCases(t *testing.T) {
	var report smokeReport
	raw := []byte(`{
  "total": 0,
  "passed": 0,
  "failed": 0,
  "cases": [
    {"name": "ok", "pass": true}
  ]
}`)
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if err := validateSmokeReportCounts(&report); err == nil {
		t.Fatalf("expected explicit zero-count mismatch")
	}
}

func TestValidateSmokeReportShapeRejectsMissingCaseSource(t *testing.T) {
	total := 1
	passed := 1
	failed := 0
	report := &smokeReport{
		Target:  "linux-x64",
		Host:    "linux-x64",
		Version: "v0.6.0",
		Total:   &total,
		Passed:  &passed,
		Failed:  &failed,
		Cases: []smokeCaseReport{
			{Name: "bad", Pass: true, Ran: true, ActualExit: intPtr(0)},
		},
	}
	if err := validateSmokeReport(report); err == nil {
		t.Fatalf("expected missing src_path error")
	}
}

func TestValidateSmokeReportShapeRejectsDuplicateCaseName(t *testing.T) {
	total := 2
	passed := 2
	failed := 0
	report := &smokeReport{
		Target:  "linux-x64",
		Host:    "linux-x64",
		Version: "v0.6.0",
		Total:   &total,
		Passed:  &passed,
		Failed:  &failed,
		Cases: []smokeCaseReport{
			{Name: "dup", SrcPath: "examples/one.tetra", Pass: true, Ran: true, ActualExit: intPtr(0)},
			{Name: "dup", SrcPath: "examples/two.tetra", Pass: true, Ran: true, ActualExit: intPtr(0)},
		},
	}
	if err := validateSmokeReport(report); err == nil {
		t.Fatalf("expected duplicate case error")
	}
}

func TestValidateSmokeReportShapeRejectsRanCaseWithoutActualExit(t *testing.T) {
	total := 1
	passed := 1
	failed := 0
	report := &smokeReport{
		Target:  "linux-x64",
		Host:    "linux-x64",
		Version: "v0.6.0",
		Total:   &total,
		Passed:  &passed,
		Failed:  &failed,
		Cases: []smokeCaseReport{
			{Name: "bad", SrcPath: "examples/bad.tetra", Pass: true, Ran: true},
		},
	}
	if err := validateSmokeReport(report); err == nil {
		t.Fatalf("expected missing actual_exit error")
	}
}

func TestValidateSmokeReportShapeRejectsUnsupportedTarget(t *testing.T) {
	total := 1
	passed := 1
	failed := 0
	report := &smokeReport{
		Target:  "plan9-x64",
		Host:    "linux-x64",
		Version: "v0.6.0",
		Total:   &total,
		Passed:  &passed,
		Failed:  &failed,
		Cases: []smokeCaseReport{
			{Name: "ok", SrcPath: "examples/ok.tetra", ExpectedExit: 0, Pass: true},
		},
	}
	err := validateSmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "unsupported target") {
		t.Fatalf("expected unsupported target error, got %v", err)
	}
}

func TestValidateSmokeReportShapeAcceptsWASMBuildOnlyTarget(t *testing.T) {
	total := 1
	passed := 1
	failed := 0
	report := &smokeReport{
		Target:  "wasm32-web",
		Host:    "linux-x64",
		Version: "v0.6.0",
		Total:   &total,
		Passed:  &passed,
		Failed:  &failed,
		Cases: []smokeCaseReport{
			{Name: "ok", SrcPath: "examples/ok.tetra", ExpectedExit: 0, Pass: true},
		},
	}
	if err := validateSmokeReport(report); err != nil {
		t.Fatalf("validateSmokeReport(wasm32-web): %v", err)
	}
}

func TestValidateSmokeReportShapeRejectsDuplicateSource(t *testing.T) {
	total := 2
	passed := 2
	failed := 0
	report := &smokeReport{
		Target:  "linux-x64",
		Host:    "linux-x64",
		Version: "v0.6.0",
		Total:   &total,
		Passed:  &passed,
		Failed:  &failed,
		Cases: []smokeCaseReport{
			{Name: "one", SrcPath: "examples/same.tetra", ExpectedExit: 0, Pass: true},
			{Name: "two", SrcPath: "examples/same.tetra", ExpectedExit: 0, Pass: true},
		},
	}
	err := validateSmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "duplicate smoke report src_path") {
		t.Fatalf("expected duplicate source error, got %v", err)
	}
}

func TestValidateSmokeReportShapeRejectsInvalidExitCode(t *testing.T) {
	total := 1
	passed := 1
	failed := 0
	report := &smokeReport{
		Target:  "linux-x64",
		Host:    "linux-x64",
		Version: "v0.6.0",
		Total:   &total,
		Passed:  &passed,
		Failed:  &failed,
		Cases: []smokeCaseReport{
			{Name: "bad", SrcPath: "examples/bad.tetra", ExpectedExit: 300, Pass: true},
		},
	}
	err := validateSmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "expected_exit") {
		t.Fatalf("expected invalid exit error, got %v", err)
	}
}

func TestValidateSmokeReportShapeRejectsPassedRunWithWrongExit(t *testing.T) {
	total := 1
	passed := 1
	failed := 0
	report := &smokeReport{
		Target:  "linux-x64",
		Host:    "linux-x64",
		Version: "v0.6.0",
		Total:   &total,
		Passed:  &passed,
		Failed:  &failed,
		Cases: []smokeCaseReport{
			{Name: "bad", SrcPath: "examples/bad.tetra", ExpectedExit: 42, ActualExit: intPtr(0), Ran: true, Pass: true},
		},
	}
	err := validateSmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "passed with actual_exit") {
		t.Fatalf("expected wrong exit error, got %v", err)
	}
}

func TestSmokeReportToChecklistValidateOnly(t *testing.T) {
	dir := t.TempDir()
	report := filepath.Join(dir, "smoke.json")
	if err := os.WriteFile(report, []byte(`{
  "target": "linux-x64",
  "host": "linux-x64",
  "version": "v0.6.0",
  "total": 1,
  "passed": 1,
  "failed": 0,
  "cases": [
    {"name": "islands_hello", "src_path": "examples/islands_hello.tetra", "actual_exit": 0, "ran": true, "pass": true}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--validate-only", "--report", report)
	cmd.Dir = smokeReportToolDir(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("validate-only failed: %v\n%s", err, out)
	}
}

func smokeReportToolDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

func intPtr(v int) *int {
	return &v
}
