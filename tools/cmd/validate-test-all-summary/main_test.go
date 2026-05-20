package main

import (
	"fmt"
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
  "step_count": 7,
  "failed_count": 0,
  "steps": [
    {"name":"go test all packages","status":"pass","duration_seconds":0,"exit_code":0,"command":"go test ./compiler/... ./cli/... ./tools/... -count=1","log":"logs/01-step.log"},
    {"name":"json diagnostic shape","status":"pass","duration_seconds":1,"exit_code":0,"command":"check_json_diagnostic","log":"logs/02-step.log"},
    {"name":"host smoke linux-x64","status":"pass","duration_seconds":1,"exit_code":0,"command":"check_host_smoke","log":"logs/03-step.log"},
    {"name":"docs manifest diff","status":"pass","duration_seconds":1,"exit_code":0,"command":"check_docs_manifest","log":"logs/04-step.log"},
    {"name":"safety readiness evidence","status":"pass","duration_seconds":1,"exit_code":0,"command":"check_safety_readiness","log":"logs/05-step.log"},
    {"name":"ownership production audit","status":"pass","duration_seconds":1,"exit_code":0,"command":"validate-ownership-audit","log":"logs/06-step.log"},
    {"name":"tooling summary aggregation","status":"pass","duration_seconds":1,"exit_code":0,"command":"write_tooling_summary","log":"logs/07-step.log"}
  ]
}`)
	out, err := runSummaryValidator(t, dir)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateTestAllSummaryAcceptsStabilizationReport(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "stabilization",
  "status": "pass",
  "started_at": "2026-04-25T13:00:00Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 14,
  "failed_count": 0,
  "release_artifact": "tetra.release.v0_2_0.test-all-summary.v1",
  "steps": [
    {"name":"go test all packages","status":"pass","duration_seconds":0,"exit_code":0,"command":"go test ./compiler/... ./cli/... ./tools/... -count=1","log":"logs/01-step.log"},
    {"name":"json diagnostic shape","status":"pass","duration_seconds":1,"exit_code":0,"command":"check_json_diagnostic","log":"logs/02-step.log"},
    {"name":"host smoke linux-x64","status":"pass","duration_seconds":1,"exit_code":0,"command":"check_host_smoke","log":"logs/03-step.log"},
    {"name":"docs manifest diff","status":"pass","duration_seconds":1,"exit_code":0,"command":"check_docs_manifest","log":"logs/04-step.log"},
    {"name":"safety readiness evidence","status":"pass","duration_seconds":1,"exit_code":0,"command":"check_safety_readiness","log":"logs/05-step.log"},
    {"name":"ownership production audit","status":"pass","duration_seconds":1,"exit_code":0,"command":"validate-ownership-audit","log":"logs/06-step.log"},
    {"name":"tooling summary aggregation","status":"pass","duration_seconds":1,"exit_code":0,"command":"write_tooling_summary","log":"logs/07-step.log"},
    {"name":"frontend callable focused gate","status":"pass","duration_seconds":1,"exit_code":0,"command":"go test callable","log":"logs/08-step.log"},
    {"name":"safety runtime focused gate","status":"pass","duration_seconds":1,"exit_code":0,"command":"go test safety","log":"logs/09-step.log"},
    {"name":"lowering ir focused gate","status":"pass","duration_seconds":1,"exit_code":0,"command":"go test lower","log":"logs/10-step.log"},
    {"name":"wasi runner smoke","status":"pass","duration_seconds":1,"exit_code":0,"command":"wasi-smoke","log":"logs/11-step.log"},
    {"name":"web runtime browser smoke","status":"pass","duration_seconds":1,"exit_code":0,"command":"web-smoke","log":"logs/12-step.log"},
    {"name":"api diff no-change","status":"pass","duration_seconds":1,"exit_code":0,"command":"api-diff","log":"logs/13-step.log"},
    {"name":"working tree whitespace audit","status":"pass","duration_seconds":1,"exit_code":0,"command":"git diff --check","log":"logs/14-step.log"}
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

func TestValidateTestAllSummaryRejectsPassingReportWithNoSteps(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "full",
  "status": "pass",
  "started_at": "2026-04-25T13:00:00Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 0,
  "failed_count": 0,
  "steps": []
}`)
	out, err := runSummaryValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure for empty passing report\n%s", out)
	}
	if !strings.Contains(string(out), "at least one step") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateTestAllSummaryRejectsFullPassMissingSafetyOwnershipSteps(t *testing.T) {
	dir := makeSummaryReport(t, `{
  "mode": "full",
  "status": "pass",
  "started_at": "2026-04-25T13:00:00Z",
  "ended_at": "2026-04-25T13:00:01Z",
  "step_count": 4,
  "failed_count": 0,
  "steps": [
    {"name":"go test all packages","status":"pass","duration_seconds":0,"exit_code":0,"command":"go test","log":"logs/01-step.log"},
    {"name":"json diagnostic shape","status":"pass","duration_seconds":1,"exit_code":0,"command":"check_json_diagnostic","log":"logs/02-step.log"},
    {"name":"host smoke linux-x64","status":"pass","duration_seconds":1,"exit_code":0,"command":"check_host_smoke","log":"logs/03-step.log"},
    {"name":"docs manifest diff","status":"pass","duration_seconds":1,"exit_code":0,"command":"check_docs_manifest","log":"logs/04-step.log"}
  ]
}`)
	out, err := runSummaryValidator(t, dir)
	if err == nil {
		t.Fatalf("expected validator failure for incomplete full pass report\n%s", out)
	}
	if !strings.Contains(string(out), `missing required step "safety readiness evidence"`) {
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
  "step_count": 3,
  "failed_count": 0,
  "release_artifact": "tetra.release.unknown",
  "steps": [
    {"name":"go test all packages","status":"pass","duration_seconds":0,"exit_code":0,"command":"go test","log":"logs/01-step.log"},
    {"name":"json diagnostic shape","status":"pass","duration_seconds":1,"exit_code":0,"command":"check_json_diagnostic","log":"logs/02-step.log"},
    {"name":"host smoke linux-x64","status":"pass","duration_seconds":1,"exit_code":0,"command":"check_host_smoke","log":"logs/03-step.log"}
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
	for i := 1; i <= 32; i++ {
		name := fmt.Sprintf("%02d-step.log", i)
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
