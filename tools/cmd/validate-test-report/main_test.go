package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateTestReportAcceptsValidReport(t *testing.T) {
	report := `{
  "total": 2,
  "passed": 1,
  "failed": 1,
  "duration_ms": 12,
  "files": [
    {"filename": "a.tetra", "total": 2, "passed": 1, "failed": 1, "duration_ms": 12}
  ],
  "results": [
    {"name": "ok", "filename": "a.tetra", "index": 0, "function_name": "__tetra_test_0_ok", "exit_code": 0, "passed": true, "duration_ms": 5},
    {"name": "bad", "filename": "a.tetra", "index": 1, "function_name": "__tetra_test_1_bad", "exit_code": 1, "passed": false, "duration_ms": 7, "error": "exit code 1"}
  ]
}`
	out, err := runValidator(t, report)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateTestReportRejectsNullArrays(t *testing.T) {
	report := `{"total":0,"passed":0,"failed":0,"duration_ms":0,"files":null,"results":null}`
	out, err := runValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "files must be an array") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestReportRejectsCountMismatch(t *testing.T) {
	report := `{
  "total": 1,
  "passed": 1,
  "failed": 0,
  "duration_ms": 1,
  "files": [{"filename": "a.tetra", "total": 1, "passed": 1, "failed": 0, "duration_ms": 1}],
  "results": [{"name": "bad", "filename": "a.tetra", "index": 0, "function_name": "__tetra_test_0_bad", "exit_code": 1, "passed": false, "duration_ms": 1, "error": "exit code 1"}]
}`
	out, err := runValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "report counts mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestReportRejectsPassedResultWithNonZeroExit(t *testing.T) {
	report := `{
  "total": 1,
  "passed": 1,
  "failed": 0,
  "duration_ms": 1,
  "files": [{"filename": "a.tetra", "total": 1, "passed": 1, "failed": 0, "duration_ms": 1}],
  "results": [{"name": "impossible", "filename": "a.tetra", "index": 0, "function_name": "__tetra_test_0_impossible", "exit_code": 1, "passed": true, "duration_ms": 1}]
}`
	out, err := runValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "passed result has non-zero exit code") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestReportRejectsFailedResultWithoutFailureDetail(t *testing.T) {
	report := `{
  "total": 1,
  "passed": 0,
  "failed": 1,
  "duration_ms": 1,
  "files": [{"filename": "a.tetra", "total": 1, "passed": 0, "failed": 1, "duration_ms": 1}],
  "results": [{"name": "silent", "filename": "a.tetra", "index": 0, "function_name": "__tetra_test_0_silent", "exit_code": 0, "passed": false, "duration_ms": 1}]
}`
	out, err := runValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "failed result must include a non-zero exit code or error") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestReportRejectsDuplicateResultNameInFile(t *testing.T) {
	report := `{
  "total": 2,
  "passed": 2,
  "failed": 0,
  "duration_ms": 2,
  "files": [{"filename": "a.tetra", "total": 2, "passed": 2, "failed": 0, "duration_ms": 2}],
  "results": [
    {"name": "math", "filename": "a.tetra", "index": 0, "function_name": "__tetra_test_0_math", "exit_code": 0, "passed": true, "duration_ms": 1},
    {"name": "math", "filename": "a.tetra", "index": 1, "function_name": "__tetra_test_1_math", "exit_code": 0, "passed": true, "duration_ms": 1}
  ]
}`
	out, err := runValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate test result") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestReportRejectsNegativeDurations(t *testing.T) {
	report := `{
  "total": 1,
  "passed": 1,
  "failed": 0,
  "duration_ms": -1,
  "files": [{"filename": "a.tetra", "total": 1, "passed": 1, "failed": 0, "duration_ms": -1}],
  "results": [{"name": "math", "filename": "a.tetra", "index": 0, "function_name": "__tetra_test_0_math", "exit_code": 0, "passed": true, "duration_ms": -1}]
}`
	out, err := runValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "negative duration") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestReportRejectsDuplicateIndexInFile(t *testing.T) {
	report := `{
  "total": 2,
  "passed": 2,
  "failed": 0,
  "duration_ms": 2,
  "files": [{"filename": "a.tetra", "total": 2, "passed": 2, "failed": 0, "duration_ms": 2}],
  "results": [
    {"name": "a", "filename": "a.tetra", "index": 0, "function_name": "__tetra_test_0_a", "exit_code": 0, "passed": true, "duration_ms": 1},
    {"name": "b", "filename": "a.tetra", "index": 0, "function_name": "__tetra_test_0_b", "exit_code": 0, "passed": true, "duration_ms": 1}
  ]
}`
	out, err := runValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate test index") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestReportRejectsMissingSyntheticFunction(t *testing.T) {
	report := `{
  "total": 1,
  "passed": 1,
  "failed": 0,
  "duration_ms": 1,
  "files": [{"filename": "a.tetra", "total": 1, "passed": 1, "failed": 0, "duration_ms": 1}],
  "results": [{"name": "math", "filename": "a.tetra", "index": 0, "exit_code": 0, "passed": true, "duration_ms": 1}]
}`
	out, err := runValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing function_name") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestReportRejectsNonSequentialIndex(t *testing.T) {
	report := `{
  "total": 2,
  "passed": 2,
  "failed": 0,
  "duration_ms": 2,
  "files": [{"filename": "a.tetra", "total": 2, "passed": 2, "failed": 0, "duration_ms": 2}],
  "results": [
    {"name": "a", "filename": "a.tetra", "index": 0, "function_name": "__tetra_test_0_a", "exit_code": 0, "passed": true, "duration_ms": 1},
    {"name": "b", "filename": "a.tetra", "index": 2, "function_name": "__tetra_test_2_b", "exit_code": 0, "passed": true, "duration_ms": 1}
  ]
}`
	out, err := runValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing test index 1") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func runValidator(t *testing.T, report string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")
	if err := os.WriteFile(path, []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--report", path)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
