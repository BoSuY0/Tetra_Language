package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateTestAllSummaryAcceptsPassingReport(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "full",
  "status": "pass",
  "started_at": "2026-04-25T13:00:00Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 2,
  "failed_count": 0,
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/01-one.log"},
    {"name":"two","status":"pass","duration_seconds":1,"exit_code":0,"command":"true","log":"logs/02-two.log"}
  ]
}`)
	out, err := runSummaryValidator(t, dir)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateTestAllSummaryRejectsCountMismatch(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "quick",
  "status": "pass",
  "started_at": "2026-04-25T13:00:00Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 3,
  "failed_count": 0,
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/01-one.log"}
  ]
}`)
	out, err := runSummaryValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "step_count mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestAllSummaryRejectsUnknownFields(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "full",
  "status": "pass",
  "started_at": "2026-04-25T13:00:00Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 1,
  "failed_count": 0,
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/01-one.log","extra":true}
  ]
}`)
	out, err := runSummaryValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestAllSummaryRejectsPassWithNonZeroExit(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "quick",
  "status": "pass",
  "started_at": "2026-04-25T13:00:00Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 1,
  "failed_count": 0,
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":7,"command":"true","log":"logs/01-one.log"}
  ]
}`)
	out, err := runSummaryValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "pass step one has non-zero exit code") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestAllSummaryRejectsMissingLog(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "quick",
  "status": "pass",
  "started_at": "2026-04-25T13:00:00Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 1,
  "failed_count": 0,
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/01-missing.log"}
  ]
}`)
	out, err := runSummaryValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing log file") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestAllSummaryRejectsMissingCommand(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "quick",
  "status": "pass",
  "started_at": "2026-04-25T13:00:00Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 1,
  "failed_count": 0,
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"log":"logs/01-one.log"}
  ]
}`)
	out, err := runSummaryValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing command") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestAllSummaryRejectsDuplicateStepNameAndLog(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "quick",
  "status": "pass",
  "started_at": "2026-04-25T13:00:00Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 2,
  "failed_count": 0,
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/01-one.log"},
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/02-two.log"}
  ]
}`)
	out, err := runSummaryValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate step name") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestAllSummaryRejectsUnsafeLogPath(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "quick",
  "status": "pass",
  "started_at": "2026-04-25T13:00:00Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 1,
  "failed_count": 0,
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"../outside.log"}
  ]
}`)
	out, err := runSummaryValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unsafe log path") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestAllSummaryRejectsOutOfOrderLogOrdinal(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "quick",
  "status": "pass",
  "started_at": "2026-04-25T13:00:00Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 2,
  "failed_count": 0,
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/02-two.log"},
    {"name":"two","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/01-one.log"}
  ]
}`)
	out, err := runSummaryValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "log ordinal") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestAllSummaryRejectsInvalidTimestampOrder(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "quick",
  "status": "pass",
  "started_at": "2026-04-25T13:00:02Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 1,
  "failed_count": 0,
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/01-one.log"}
  ]
}`)
	out, err := runSummaryValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "ended_at must not be before started_at") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestAllSummaryRejectsUnknownReleaseArtifact(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "quick",
  "status": "pass",
  "started_at": "2026-04-25T13:00:00Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 1,
  "failed_count": 0,
  "release_artifact": "tetra.release.unknown",
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/01-one.log"}
  ]
}`)
	out, err := runSummaryValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "release_artifact") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func makeSummaryReport(t *testing.T, summary string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"01-one.log", "02-two.log"} {
		if err := os.WriteFile(filepath.Join(dir, "logs", name), []byte("ok\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "summary.json"), []byte(summary), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func runSummaryValidator(t *testing.T, reportDir string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("go", "run", ".", "--summary", filepath.Join(reportDir, "summary.json"), "--report-dir", reportDir)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
