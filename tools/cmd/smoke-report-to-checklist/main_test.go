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

func nativeSmokeReportForTest(omit string) *smokeReport {
	cases := []struct {
		name         string
		srcPath      string
		expectedExit int
	}{
		{"flow_hello", "examples/flow_hello.tetra", 0},
		{"flow_struct_smoke", "examples/flow_struct_smoke.tetra", 42},
		{"flow_islands_smoke", "examples/flow_islands_smoke.tetra", 0},
		{"flow_unsafe_cap_mem_smoke", "examples/flow_unsafe_cap_mem_smoke.tetra", 42},
		{"core_async_smoke", "examples/core_async_smoke.tetra", 42},
		{"core_capability_smoke", "examples/core_capability_smoke.tetra", 42},
		{"core_collections_smoke", "examples/core_collections_smoke.tetra", 42},
		{"core_crypto_smoke", "examples/core_crypto_smoke.tetra", 42},
		{"core_filesystem_smoke", "examples/core_filesystem_smoke.tetra", 42},
		{"core_io_smoke", "examples/core_io_smoke.tetra", 42},
		{"core_math_smoke", "examples/core_math_smoke.tetra", 42},
		{"core_memory_smoke", "examples/core_memory_smoke.tetra", 42},
		{"core_networking_smoke", "examples/core_networking_smoke.tetra", 42},
		{"core_serialization_smoke", "examples/core_serialization_smoke.tetra", 42},
		{"core_slices_smoke", "examples/core_slices_smoke.tetra", 42},
		{"core_strings_smoke", "examples/core_strings_smoke.tetra", 42},
		{"core_sync_smoke", "examples/core_sync_smoke.tetra", 42},
		{"core_testing_smoke", "examples/core_testing_smoke.tetra", 42},
		{"core_time_smoke", "examples/core_time_smoke.tetra", 42},
	}
	reportCases := make([]smokeCaseReport, 0, len(cases))
	for _, c := range cases {
		if c.name == omit {
			continue
		}
		reportCases = append(reportCases, smokeCaseReport{
			Name:         c.name,
			SrcPath:      c.srcPath,
			ExpectedExit: c.expectedExit,
			Pass:         true,
		})
	}
	total := len(reportCases)
	passed := total
	failed := 0
	return &smokeReport{
		Target:       "linux-x64",
		Host:         "linux-x64",
		Version:      "v0.6.0",
		IslandsDebug: false,
		Total:        &total,
		Passed:       &passed,
		Failed:       &failed,
		Cases:        reportCases,
	}
}

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

func TestValidateSmokeReportRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{
  "timestamp": "2026-04-27T00:00:00Z",
  "target": "linux-x64",
  "host": "linux-x64",
  "version": "v1.0.0",
  "islands_debug": false,
  "total": 4,
  "passed": 4,
  "failed": 0,
  "cases": [
    {"name":"flow_hello","src_path":"examples/flow_hello.tetra","expected_exit":0,"ran":false,"pass":true,"extra":true},
    {"name":"flow_struct_smoke","src_path":"examples/flow_struct_smoke.tetra","expected_exit":42,"ran":false,"pass":true},
    {"name":"flow_islands_smoke","src_path":"examples/flow_islands_smoke.tetra","expected_exit":0,"ran":false,"pass":true},
    {"name":"flow_unsafe_cap_mem_smoke","src_path":"examples/flow_unsafe_cap_mem_smoke.tetra","expected_exit":42,"ran":false,"pass":true}
  ]
}`)
	if _, err := parseSmokeReport(raw); err == nil {
		t.Fatalf("expected unknown field failure")
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

func TestValidateSmokeReportShapeAcceptsNativeRequiredProfile(t *testing.T) {
	if err := validateSmokeReport(nativeSmokeReportForTest("")); err != nil {
		t.Fatalf("validate native smoke report: %v", err)
	}
}

func TestValidateSmokeReportShapeRejectsMissingCoreStdlibCase(t *testing.T) {
	err := validateSmokeReport(nativeSmokeReportForTest("core_crypto_smoke"))
	if err == nil {
		t.Fatalf("expected missing core stdlib smoke case")
	}
	if !strings.Contains(err.Error(), "core_crypto_smoke") {
		t.Fatalf("missing core stdlib error = %v", err)
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

func TestValidateSmokeReportShapeAcceptsWASMSupportedArtifactTarget(t *testing.T) {
	total := 5
	passed := 5
	failed := 0
	report := &smokeReport{
		Target:  "wasm32-web",
		Host:    "linux-x64",
		Version: "v0.6.0",
		Total:   &total,
		Passed:  &passed,
		Failed:  &failed,
		Cases: []smokeCaseReport{
			{Name: "legacy_hello", SrcPath: "examples/hello.tetra", ExpectedExit: 0, Pass: true},
			{Name: "effects_io_smoke", SrcPath: "examples/effects_io_smoke.tetra", ExpectedExit: 0, Pass: true},
			{Name: "ui_web_smoke", SrcPath: "examples/ui_web_smoke.tetra", ExpectedExit: 0, Pass: true},
			{Name: "dogfood_wasi", SrcPath: "examples/projects/dogfood_wasi/src/main.tetra", ExpectedExit: 0, Pass: true},
			{Name: "dogfood_web_ui", SrcPath: "examples/projects/dogfood_web_ui/src/main.tetra", ExpectedExit: 0, Pass: true},
		},
	}
	if err := validateSmokeReport(report); err != nil {
		t.Fatalf("validateSmokeReport(wasm32-web): %v", err)
	}
}

func TestValidateSmokeReportShapeRejectsWASMMissingDogfoodProfile(t *testing.T) {
	total := 3
	passed := 3
	failed := 0
	report := &smokeReport{
		Target:  "wasm32-web",
		Host:    "linux-x64",
		Version: "v0.6.0",
		Total:   &total,
		Passed:  &passed,
		Failed:  &failed,
		Cases: []smokeCaseReport{
			{Name: "flow_hello", SrcPath: "examples/flow_hello.tetra", ExpectedExit: 0, Pass: true},
			{Name: "effects_io_smoke", SrcPath: "examples/effects_io_smoke.tetra", ExpectedExit: 0, Pass: true},
			{Name: "ui_web_smoke", SrcPath: "examples/ui_web_smoke.tetra", ExpectedExit: 0, Pass: true},
		},
	}
	err := validateSmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "missing required smoke profile") {
		t.Fatalf("expected missing dogfood profile error, got %v", err)
	}
}

func TestValidateSmokeReportShapeRejectsStaleBuildOnlyFlagForSupportedWASMTarget(t *testing.T) {
	total := 5
	passed := 5
	failed := 0
	report := &smokeReport{
		Target:    "wasm32-wasi",
		BuildOnly: true,
		Host:      "linux-x64",
		Version:   "v0.6.0",
		Total:     &total,
		Passed:    &passed,
		Failed:    &failed,
		Cases: []smokeCaseReport{
			{Name: "legacy_hello", SrcPath: "examples/hello.tetra", ExpectedExit: 0, Pass: true},
			{Name: "effects_io_smoke", SrcPath: "examples/effects_io_smoke.tetra", ExpectedExit: 0, Pass: true},
			{Name: "ui_web_smoke", SrcPath: "examples/ui_web_smoke.tetra", ExpectedExit: 0, Pass: true},
			{Name: "dogfood_wasi", SrcPath: "examples/projects/dogfood_wasi/src/main.tetra", ExpectedExit: 0, Pass: true},
			{Name: "dogfood_web_ui", SrcPath: "examples/projects/dogfood_web_ui/src/main.tetra", ExpectedExit: 0, Pass: true},
		},
	}
	err := validateSmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "build_only = true, want false") {
		t.Fatalf("expected stale build_only error, got %v", err)
	}
}

func TestValidateSmokeReportShapeAcceptsRanWASMCaseForSupportedTarget(t *testing.T) {
	total := 5
	passed := 5
	failed := 0
	report := &smokeReport{
		Target:  "wasm32-web",
		Host:    "linux-x64",
		Version: "v0.6.0",
		Total:   &total,
		Passed:  &passed,
		Failed:  &failed,
		Cases: []smokeCaseReport{
			{Name: "legacy_hello", SrcPath: "examples/hello.tetra", ExpectedExit: 0, Pass: true},
			{Name: "effects_io_smoke", SrcPath: "examples/effects_io_smoke.tetra", ExpectedExit: 0, Pass: true},
			{Name: "ui_web_smoke", SrcPath: "examples/ui_web_smoke.tetra", ExpectedExit: 0, Ran: true, ActualExit: intPtr(0), Pass: true},
			{Name: "dogfood_wasi", SrcPath: "examples/projects/dogfood_wasi/src/main.tetra", ExpectedExit: 0, Pass: true},
			{Name: "dogfood_web_ui", SrcPath: "examples/projects/dogfood_web_ui/src/main.tetra", ExpectedExit: 0, Pass: true},
		},
	}
	if err := validateSmokeReport(report); err != nil {
		t.Fatalf("validateSmokeReport(wasm32-web ran case): %v", err)
	}
}

func TestValidateSmokeReportShapeAcceptsRanWASMCaseWithRunner(t *testing.T) {
	total := 5
	passed := 5
	failed := 0
	report := &smokeReport{
		Target:  "wasm32-web",
		Runner:  "chromium",
		Host:    "linux-x64",
		Version: "v0.6.0",
		Total:   &total,
		Passed:  &passed,
		Failed:  &failed,
		Cases: []smokeCaseReport{
			{Name: "legacy_hello", SrcPath: "examples/hello.tetra", ExpectedExit: 0, Ran: true, ActualExit: intPtr(0), Pass: true},
			{Name: "effects_io_smoke", SrcPath: "examples/effects_io_smoke.tetra", ExpectedExit: 0, Ran: true, ActualExit: intPtr(0), Pass: true},
			{Name: "ui_web_smoke", SrcPath: "examples/ui_web_smoke.tetra", ExpectedExit: 0, Ran: true, ActualExit: intPtr(0), Pass: true},
			{Name: "dogfood_wasi", SrcPath: "examples/projects/dogfood_wasi/src/main.tetra", ExpectedExit: 0, Ran: true, ActualExit: intPtr(0), Pass: true},
			{Name: "dogfood_web_ui", SrcPath: "examples/projects/dogfood_web_ui/src/main.tetra", ExpectedExit: 0, Ran: true, ActualExit: intPtr(0), Pass: true},
		},
	}
	if err := validateSmokeReport(report); err != nil {
		t.Fatalf("validateSmokeReport(wasm32-web with runner): %v", err)
	}
}

func TestValidateSmokeReportShapeRejectsMissingWASMRequiredCase(t *testing.T) {
	total := 1
	passed := 1
	failed := 0
	report := &smokeReport{
		Target:  "wasm32-wasi",
		Host:    "linux-x64",
		Version: "v0.6.0",
		Total:   &total,
		Passed:  &passed,
		Failed:  &failed,
		Cases: []smokeCaseReport{
			{Name: "flow_hello", SrcPath: "examples/flow_hello.tetra", ExpectedExit: 0, Pass: true},
		},
	}
	err := validateSmokeReport(report)
	if err == nil || !strings.Contains(err.Error(), "missing required smoke profile") {
		t.Fatalf("expected missing required case error, got %v", err)
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
	raw, err := json.Marshal(nativeSmokeReportForTest(""))
	if err != nil {
		t.Fatalf("marshal native smoke report: %v", err)
	}
	if err := os.WriteFile(report, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--validate-only", "--report", report)
	cmd.Dir = smokeReportToolDir(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("validate-only failed: %v\n%s", err, out)
	}
}

func TestSmokeReportToChecklistUpdatesTargetSection(t *testing.T) {
	dir := t.TempDir()
	checklist := filepath.Join(dir, "islands.md")
	if err := os.WriteFile(checklist, []byte(`Date:
Target version:
Git HEAD:
Compiler version (compilerVersion):

## Linux x64 (sanity)

- [ ] build examples/islands_hello.tetra
- [ ] run ./islands_hello

## Windows x64

- [ ] build examples/islands_hello.tetra
`), 0o644); err != nil {
		t.Fatal(err)
	}
	total, passed, failed := 1, 1, 0
	report := &smokeReport{
		Timestamp: "2026-04-27T12:00:00Z",
		Target:    "linux-x64",
		Host:      "linux-x64",
		Version:   "v0.1.0",
		GitHead:   "abc123",
		Total:     &total,
		Passed:    &passed,
		Failed:    &failed,
		Cases: []smokeCaseReport{{
			Name:         "islands_hello",
			SrcPath:      "examples/islands_hello.tetra",
			ExpectedExit: 0,
			ActualExit:   intPtr(0),
			Ran:          true,
			Pass:         true,
		}},
	}
	updates := []checkboxUpdate{
		{Contains: "examples/islands_hello.tetra", Checked: true},
		{Contains: "./islands_hello", Checked: true},
	}
	if err := applyToChecklist(checklist, report, updates); err != nil {
		t.Fatalf("applyToChecklist: %v", err)
	}
	raw, err := os.ReadFile(checklist)
	if err != nil {
		t.Fatal(err)
	}
	out := string(raw)
	for _, want := range []string{
		"Date: 2026-04-27",
		"Target version: linux-x64",
		"Git HEAD: abc123",
		"Compiler version (compilerVersion): v0.1.0",
		"- [x] build examples/islands_hello.tetra",
		"- [x] run ./islands_hello",
		"## Windows x64\n\n- [ ] build examples/islands_hello.tetra",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("checklist missing %q:\n%s", want, out)
		}
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
