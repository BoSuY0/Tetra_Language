package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReleaseGateSummaryAcceptsPassingV030Report(t *testing.T) {
	dir := makeReleaseGateSummaryReport(t, `{
  "status": "pass",
  "release_version": "v0.3.0",
  "release_artifact": "tetra.release.v0_3_0.gate-report.v1",
  "release_gate_command": "bash scripts/release/v0_3_0/gate.sh",
  "started_at": "2026-04-29T10:00:00Z",
  "ended_at": "2026-04-29T10:00:02Z",
  "step_count": 2,
  "failed_count": 0,
  "report_dir": "reports/release-v0.3.0-gate",
  "steps": [
    {"name":"version preflight (v0.3.0 required)","status":"pass","duration_seconds":0,"exit_code":0,"command":"check_release_version","log":"logs/01-version-preflight.log"},
    {"name":"docs verification","status":"pass","duration_seconds":2,"exit_code":0,"command":"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json","log":"logs/02-docs-verification.log"}
  ]
}`)
	if err := validateReleaseGateSummaryFile(filepath.Join(dir, "summary.json"), dir); err != nil {
		t.Fatalf("validator failed: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsBlockedReportWithoutFailingStep(t *testing.T) {
	dir := makeReleaseGateSummaryReport(t, `{
  "status": "blocked",
  "release_version": "v0.3.0",
  "release_artifact": "tetra.release.v0_3_0.gate-report.v1",
  "release_gate_command": "bash scripts/release/v0_3_0/gate.sh",
  "started_at": "2026-04-29T10:00:00Z",
  "ended_at": "2026-04-29T10:00:01Z",
  "step_count": 0,
  "failed_count": 0,
  "report_dir": "reports/release-v0.3.0-gate",
  "steps": []
}`)
	err := validateReleaseGateSummaryFile(filepath.Join(dir, "summary.json"), dir)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "blocked summary contains no failing steps") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryAcceptsV040ReportWithExpectedIdentity(t *testing.T) {
	dir := makeReleaseGateSummaryReport(t, `{
  "status": "blocked",
  "release_version": "v0.4.0",
  "release_artifact": "tetra.release.v0_4_0.gate-report.v1",
  "release_gate_command": "bash scripts/release/v0_4_0/gate.sh",
  "started_at": "2026-05-04T14:00:00Z",
  "ended_at": "2026-05-04T14:00:01Z",
  "step_count": 1,
  "failed_count": 1,
  "report_dir": "reports/release-v0.4.0-gate",
  "steps": [
    {"name":"readiness preflight","status":"fail","duration_seconds":1,"exit_code":1,"command":"go run ./tools/cmd/validate-v0-4-readiness","log":"logs/01-readiness-preflight.log"}
  ]
}`, "logs/01-readiness-preflight.log")
	err := validateReleaseGateSummaryFileWithExpectations(filepath.Join(dir, "summary.json"), dir, releaseGateSummaryExpectations{
		ReleaseVersion:     "v0.4.0",
		ReleaseArtifact:    "tetra.release.v0_4_0.gate-report.v1",
		ReleaseGateCommand: "bash scripts/release/v0_4_0/gate.sh",
	})
	if err != nil {
		t.Fatalf("validator failed: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithoutCompilerProductionArtifact(t *testing.T) {
	dir := makeReleaseGateSummaryReport(t, `{
  "status": "pass",
  "release_version": "v0.4.0",
  "release_artifact": "tetra.release.v0_4_0.gate-report.v1",
  "release_gate_command": "bash scripts/release/v0_4_0/gate.sh",
  "started_at": "2026-05-20T10:00:00Z",
  "ended_at": "2026-05-20T10:00:02Z",
  "step_count": 1,
  "failed_count": 0,
  "report_dir": "reports/release-v0.4.0-gate",
  "steps": [
    {"name":"readiness preflight","status":"pass","duration_seconds":1,"exit_code":0,"command":"go run ./tools/cmd/validate-v0-4-readiness","log":"logs/01-readiness-preflight.log"}
  ]
}`, "logs/01-readiness-preflight.log")
	writeReleaseGateArtifactHashes(t, dir, `{
  "schema": "tetra.release-artifact-hashes.v1alpha1",
  "root": ".",
  "artifacts": [
    {
      "path": "artifacts/memory-production-linux-x64.json",
      "sha256": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
      "size": 0,
      "schema": "tetra.memory.production.v1"
    },
    {
      "path": "artifacts/parallel-production-linux-x64.json",
      "sha256": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
      "size": 0,
      "schema": "tetra.parallel.production.v1"
    }
  ]
}`)
	err := validateReleaseGateSummaryFileWithExpectations(filepath.Join(dir, "summary.json"), dir, releaseGateSummaryExpectations{
		ReleaseVersion:     "v0.4.0",
		ReleaseArtifact:    "tetra.release.v0_4_0.gate-report.v1",
		ReleaseGateCommand: "bash scripts/release/v0_4_0/gate.sh",
	})
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "artifacts/compiler-production-linux-x64.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithoutMemoryProductionArtifact(t *testing.T) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(t, dir, `{
  "schema": "tetra.release-artifact-hashes.v1alpha1",
  "root": ".",
  "artifacts": [
    {
      "path": "artifacts/compiler-production-linux-x64.json",
      "sha256": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
      "size": 0,
      "schema": "tetra.compiler.production.v1"
    },
    {
      "path": "artifacts/parallel-production-linux-x64.json",
      "sha256": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
      "size": 0,
      "schema": "tetra.parallel.production.v1"
    }
  ]
}`)
	err := validateReleaseGateSummaryFileWithExpectations(filepath.Join(dir, "summary.json"), dir, v040ReleaseGateSummaryExpectations())
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "artifacts/memory-production-linux-x64.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithoutParallelProductionArtifact(t *testing.T) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(t, dir, `{
  "schema": "tetra.release-artifact-hashes.v1alpha1",
  "root": ".",
  "artifacts": [
    {
      "path": "artifacts/memory-production-linux-x64.json",
      "sha256": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
      "size": 0,
      "schema": "tetra.memory.production.v1"
    },
    {
      "path": "artifacts/compiler-production-linux-x64.json",
      "sha256": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
      "size": 0,
      "schema": "tetra.compiler.production.v1"
    }
  ]
}`)
	err := validateReleaseGateSummaryFileWithExpectations(filepath.Join(dir, "summary.json"), dir, v040ReleaseGateSummaryExpectations())
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "artifacts/parallel-production-linux-x64.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryAcceptsPassingV040ReportWithCompilerProductionArtifact(t *testing.T) {
	dir := makeReleaseGateSummaryReport(t, `{
  "status": "pass",
  "release_version": "v0.4.0",
  "release_artifact": "tetra.release.v0_4_0.gate-report.v1",
  "release_gate_command": "bash scripts/release/v0_4_0/gate.sh",
  "started_at": "2026-05-20T10:00:00Z",
  "ended_at": "2026-05-20T10:00:02Z",
  "step_count": 1,
  "failed_count": 0,
  "report_dir": "reports/release-v0.4.0-gate",
  "steps": [
    {"name":"validate compiler production","status":"pass","duration_seconds":1,"exit_code":0,"command":"go run ./tools/cmd/validate-compiler-production --report reports/release-v0.4.0-gate/artifacts/compiler-production-linux-x64.json","log":"logs/01-validate-compiler-production.log"}
  ]
}`, "logs/01-validate-compiler-production.log")
	writeReleaseGateArtifactHashes(t, dir, `{
  "schema": "tetra.release-artifact-hashes.v1alpha1",
  "root": ".",
  "artifacts": [
    {
      "path": "artifacts/memory-production-linux-x64.json",
      "sha256": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
      "size": 0,
      "schema": "tetra.memory.production.v1"
    },
    {
      "path": "artifacts/parallel-production-linux-x64.json",
      "sha256": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
      "size": 0,
      "schema": "tetra.parallel.production.v1"
    },
    {
      "path": "artifacts/compiler-production-linux-x64.json",
      "sha256": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
      "size": 0,
      "schema": "tetra.compiler.production.v1"
    }
  ]
}`)
	err := validateReleaseGateSummaryFileWithExpectations(filepath.Join(dir, "summary.json"), dir, releaseGateSummaryExpectations{
		ReleaseVersion:     "v0.4.0",
		ReleaseArtifact:    "tetra.release.v0_4_0.gate-report.v1",
		ReleaseGateCommand: "bash scripts/release/v0_4_0/gate.sh",
	})
	if err != nil {
		t.Fatalf("validator failed: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsStaleReleaseIdentity(t *testing.T) {
	dir := makeReleaseGateSummaryReport(t, `{
  "status": "pass",
  "release_version": "v0.2.0",
  "release_artifact": "tetra.release.v0_2_0.gate-report.v1",
  "release_gate_command": "bash scripts/release/v0_2_0/gate.sh",
  "started_at": "2026-04-29T10:00:00Z",
  "ended_at": "2026-04-29T10:00:01Z",
  "step_count": 1,
  "failed_count": 0,
  "report_dir": "reports/release-v0.3.0-gate",
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/01-one.log"}
  ]
}`)
	err := validateReleaseGateSummaryFile(filepath.Join(dir, "summary.json"), dir)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	for _, want := range []string{"release_version", "v0.3.0"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}

func TestValidateReleaseGateSummaryRejectsUnknownFields(t *testing.T) {
	dir := makeReleaseGateSummaryReport(t, `{
  "status": "pass",
  "release_version": "v0.3.0",
  "release_artifact": "tetra.release.v0_3_0.gate-report.v1",
  "release_gate_command": "bash scripts/release/v0_3_0/gate.sh",
  "started_at": "2026-04-29T10:00:00Z",
  "ended_at": "2026-04-29T10:00:01Z",
  "step_count": 1,
  "failed_count": 0,
  "report_dir": "reports/release-v0.3.0-gate",
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/01-one.log","extra":true}
  ]
}`)
	err := validateReleaseGateSummaryFile(filepath.Join(dir, "summary.json"), dir)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsCountMismatch(t *testing.T) {
	dir := makeReleaseGateSummaryReport(t, `{
  "status": "pass",
  "release_version": "v0.3.0",
  "release_artifact": "tetra.release.v0_3_0.gate-report.v1",
  "release_gate_command": "bash scripts/release/v0_3_0/gate.sh",
  "started_at": "2026-04-29T10:00:00Z",
  "ended_at": "2026-04-29T10:00:01Z",
  "step_count": 2,
  "failed_count": 0,
  "report_dir": "reports/release-v0.3.0-gate",
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/01-one.log"}
  ]
}`)
	err := validateReleaseGateSummaryFile(filepath.Join(dir, "summary.json"), dir)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "step_count mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsFailedCountMismatch(t *testing.T) {
	dir := makeReleaseGateSummaryReport(t, `{
  "status": "blocked",
  "release_version": "v0.3.0",
  "release_artifact": "tetra.release.v0_3_0.gate-report.v1",
  "release_gate_command": "bash scripts/release/v0_3_0/gate.sh",
  "started_at": "2026-04-29T10:00:00Z",
  "ended_at": "2026-04-29T10:00:01Z",
  "step_count": 1,
  "failed_count": 0,
  "report_dir": "reports/release-v0.3.0-gate",
  "steps": [
    {"name":"one","status":"fail","duration_seconds":0,"exit_code":1,"command":"false","log":"logs/01-one.log"}
  ]
}`)
	err := validateReleaseGateSummaryFile(filepath.Join(dir, "summary.json"), dir)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	for _, want := range []string{"failed_count mismatch", "computed 1"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}

func TestValidateReleaseGateSummaryRejectsPassSummaryWithFailingSteps(t *testing.T) {
	dir := makeReleaseGateSummaryReport(t, `{
  "status": "pass",
  "release_version": "v0.3.0",
  "release_artifact": "tetra.release.v0_3_0.gate-report.v1",
  "release_gate_command": "bash scripts/release/v0_3_0/gate.sh",
  "started_at": "2026-04-29T10:00:00Z",
  "ended_at": "2026-04-29T10:00:01Z",
  "step_count": 1,
  "failed_count": 1,
  "report_dir": "reports/release-v0.3.0-gate",
  "steps": [
    {"name":"one","status":"fail","duration_seconds":0,"exit_code":1,"command":"false","log":"logs/01-one.log"}
  ]
}`)
	err := validateReleaseGateSummaryFile(filepath.Join(dir, "summary.json"), dir)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "pass summary contains failing steps") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsMissingLog(t *testing.T) {
	dir := makeReleaseGateSummaryReport(t, `{
  "status": "pass",
  "release_version": "v0.3.0",
  "release_artifact": "tetra.release.v0_3_0.gate-report.v1",
  "release_gate_command": "bash scripts/release/v0_3_0/gate.sh",
  "started_at": "2026-04-29T10:00:00Z",
  "ended_at": "2026-04-29T10:00:01Z",
  "step_count": 1,
  "failed_count": 0,
  "report_dir": "reports/release-v0.3.0-gate",
  "steps": [
    {"name":"one","status":"pass","duration_seconds":0,"exit_code":0,"command":"true","log":"logs/01-missing.log"}
  ]
}`, "logs/01-one.log")
	err := validateReleaseGateSummaryFile(filepath.Join(dir, "summary.json"), dir)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "missing log file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func makeReleaseGateSummaryReport(t *testing.T, summary string, logs ...string) string {
	t.Helper()
	dir := t.TempDir()
	if len(logs) == 0 {
		logs = []string{"logs/01-version-preflight.log", "logs/02-docs-verification.log", "logs/01-one.log"}
	}
	if err := os.MkdirAll(filepath.Join(dir, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, log := range logs {
		path := filepath.Join(dir, filepath.FromSlash(log))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("ok\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "summary.json"), []byte(summary), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func makeV040PassingReleaseGateSummaryReport(t *testing.T) string {
	t.Helper()
	return makeReleaseGateSummaryReport(t, `{
  "status": "pass",
  "release_version": "v0.4.0",
  "release_artifact": "tetra.release.v0_4_0.gate-report.v1",
  "release_gate_command": "bash scripts/release/v0_4_0/gate.sh",
  "started_at": "2026-05-20T10:00:00Z",
  "ended_at": "2026-05-20T10:00:02Z",
  "step_count": 1,
  "failed_count": 0,
  "report_dir": "reports/release-v0.4.0-gate",
  "steps": [
    {"name":"readiness preflight","status":"pass","duration_seconds":1,"exit_code":0,"command":"go run ./tools/cmd/validate-v0-4-readiness","log":"logs/01-readiness-preflight.log"}
  ]
}`, "logs/01-readiness-preflight.log")
}

func v040ReleaseGateSummaryExpectations() releaseGateSummaryExpectations {
	return releaseGateSummaryExpectations{
		ReleaseVersion:     "v0.4.0",
		ReleaseArtifact:    "tetra.release.v0_4_0.gate-report.v1",
		ReleaseGateCommand: "bash scripts/release/v0_4_0/gate.sh",
	}
}

func writeReleaseGateArtifactHashes(t *testing.T, dir string, manifest string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "artifact-hashes.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}
