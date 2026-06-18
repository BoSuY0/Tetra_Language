package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		releaseGateSummaryExpectations{
			ReleaseVersion:     "v0.4.0",
			ReleaseArtifact:    "tetra.release.v0_4_0.gate-report.v1",
			ReleaseGateCommand: "bash scripts/release/v0_4_0/gate.sh",
		},
	)
	if err != nil {
		t.Fatalf("validator failed: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithoutCompilerProductionArtifact(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(
		t,
		dir,
		v040ArtifactHashesManifestExcept("artifacts/compiler-production-linux-x64.json"),
	)
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		releaseGateSummaryExpectations{
			ReleaseVersion:     "v0.4.0",
			ReleaseArtifact:    "tetra.release.v0_4_0.gate-report.v1",
			ReleaseGateCommand: "bash scripts/release/v0_4_0/gate.sh",
		},
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "artifacts/compiler-production-linux-x64.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithoutTechEmpowerReportStep(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReportExcept(t, "techempower report schemas")
	writeReleaseGateArtifactHashes(t, dir, v040ProductionArtifactHashesManifest())
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), `missing required step "techempower report schemas"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithoutDocsVerificationStep(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReportExcept(t, "docs verification")
	writeReleaseGateArtifactHashes(t, dir, v040ProductionArtifactHashesManifest())
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), `missing required step "docs verification"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithReorderedRequiredSteps(
	t *testing.T,
) {
	order := v040GateStepNames()
	order[3], order[4] = order[4], order[3]
	dir := makeV040PassingReleaseGateSummaryReportInOrder(t, order)
	writeReleaseGateArtifactHashes(t, dir, v040ArtifactHashesManifestForSummary(t, dir))
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(
		err.Error(),
		`required step 04 = "techempower report schemas", want "docs verification"`,
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithWrongRequiredStepCommand(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	mutateV040ReleaseGateSummaryStep(
		t,
		dir,
		"validate memory production",
		func(step *releaseGateStep) {
			step.Command = "go run ./tools/cmd/validate-memory-production"
		},
	)
	writeReleaseGateArtifactHashes(t, dir, v040ArtifactHashesManifestForSummary(t, dir))
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), `required step "validate memory production" command`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithWrongRequiredStepLogPath(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	mutateV040ReleaseGateSummaryStep(t, dir, "docs verification", func(step *releaseGateStep) {
		step.Log = "logs/04-docs.log"
	})
	writeReleaseGateLog(t, dir, "logs/04-docs.log")
	writeReleaseGateArtifactHashes(t, dir, v040ArtifactHashesManifestForSummary(t, dir))
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(
		err.Error(),
		`required step "docs verification" log = "logs/04-docs.log", want "logs/04-docs-verification.log"`,
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithMismatchedReportDir(t *testing.T) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	spoofedReportDir := "reports/spoofed-v0.4.0-gate"
	mutateV040ReleaseGateSummary(t, dir, func(summary *releaseGateSummary) {
		summary.ReportDir = spoofedReportDir
		for i := range summary.Steps {
			if command, ok := expectedV040RequiredStepCommand(summary.Steps[i].Name, spoofedReportDir); ok {
				summary.Steps[i].Command = command
			}
		}
	})
	writeReleaseGateArtifactHashes(t, dir, v040ArtifactHashesManifestForSummary(t, dir))
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), `report_dir = "reports/spoofed-v0.4.0-gate"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithNonDotArtifactHashRoot(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(t, dir, v040ArtifactHashesManifestWithRoot(t, dir, "artifacts"))
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), `artifact-hashes.json root = "artifacts", want "."`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithDuplicateArtifactHashPath(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(
		t,
		dir,
		v040ArtifactHashesManifestWithDuplicatePath(t, dir, "artifacts/features.json"),
	)
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(
		err.Error(),
		`duplicate artifact path "artifacts/features.json" in artifact-hashes.json`,
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithUnsafeArtifactHashPath(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(
		t,
		dir,
		v040ArtifactHashesManifestWithExtraPath(t, dir, "../outside.txt"),
	)
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(
		err.Error(),
		`unsafe artifact path "../outside.txt" in artifact-hashes.json`,
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithUnsortedArtifactHashPaths(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(t, dir, v040ArtifactHashesManifestWithUnsortedPaths(t, dir))
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "artifact-hashes.json artifacts must be sorted by path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithInvalidArtifactHashDigest(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(
		t,
		dir,
		v040ArtifactHashesManifestWithMutatedArtifact(
			t,
			dir,
			"artifacts/features.json",
			func(artifact *releaseHashArtifact) {
				artifact.SHA256 = "not-a-digest"
			},
		),
	)
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(
		err.Error(),
		`artifact artifacts/features.json has invalid sha256 format`,
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithNegativeArtifactHashSize(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(
		t,
		dir,
		v040ArtifactHashesManifestWithMutatedArtifact(
			t,
			dir,
			"artifacts/features.json",
			func(artifact *releaseHashArtifact) {
				artifact.Size = -1
			},
		),
	)
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), `artifact artifacts/features.json has negative size`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithMismatchedSecurityReviewDetachedHash(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeV040SecurityReviewDetachedHash(
		t,
		dir,
		("sha256:fffffffffffffffffffffffffffffffffffffffffffffffffffffffff" +
			"fffffff  artifacts/security-review.md"),
	)
	writeReleaseGateArtifactHashes(t, dir, v040ArtifactHashesManifestForSummary(t, dir))
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "security-review.md.sha256") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithoutFeaturesArtifact(t *testing.T) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(
		t,
		dir,
		v040ArtifactHashesManifestExcept("artifacts/features.json"),
	)
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "artifacts/features.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithoutStepLogArtifact(t *testing.T) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(
		t,
		dir,
		v040ArtifactHashesManifestExcept("logs/04-docs-verification.log"),
	)
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "logs/04-docs-verification.log") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithoutMemoryProductionArtifact(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(
		t,
		dir,
		v040ArtifactHashesManifestExcept("artifacts/memory-production-linux-x64.json"),
	)
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "artifacts/memory-production-linux-x64.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryRejectsPassingV040ReportWithoutParallelProductionArtifact(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(
		t,
		dir,
		v040ArtifactHashesManifestExcept("artifacts/parallel-production-linux-x64.json"),
	)
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		v040ReleaseGateSummaryExpectations(),
	)
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "artifacts/parallel-production-linux-x64.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReleaseGateSummaryAcceptsPassingV040ReportWithCompilerProductionArtifact(
	t *testing.T,
) {
	dir := makeV040PassingReleaseGateSummaryReport(t)
	writeReleaseGateArtifactHashes(t, dir, v040ProductionArtifactHashesManifest())
	err := validateReleaseGateSummaryFileWithExpectations(
		filepath.Join(dir, "summary.json"),
		dir,
		releaseGateSummaryExpectations{
			ReleaseVersion:     "v0.4.0",
			ReleaseArtifact:    "tetra.release.v0_4_0.gate-report.v1",
			ReleaseGateCommand: "bash scripts/release/v0_4_0/gate.sh",
		},
	)
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
		logs = []string{
			"logs/01-version-preflight.log",
			"logs/02-docs-verification.log",
			"logs/01-one.log",
		}
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

func writeReleaseGateLog(t *testing.T, dir, log string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(log))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeV040SecurityReviewDetachedHash(t *testing.T, dir, line string) {
	t.Helper()
	path := filepath.Join(dir, "artifacts", "security-review.md.sha256")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func makeV040PassingReleaseGateSummaryReport(t *testing.T) string {
	t.Helper()
	return makeV040PassingReleaseGateSummaryReportExcept(t)
}

func makeV040PassingReleaseGateSummaryReportExcept(t *testing.T, omittedSteps ...string) string {
	t.Helper()
	omitted := make(map[string]bool, len(omittedSteps))
	for _, step := range omittedSteps {
		omitted[step] = true
	}

	var order []string
	for _, fixture := range v040GateStepFixtures {
		if omitted[fixture.Name] {
			continue
		}
		order = append(order, fixture.Name)
	}
	return makeV040PassingReleaseGateSummaryReportInOrder(t, order)
}

func makeV040PassingReleaseGateSummaryReportInOrder(t *testing.T, order []string) string {
	t.Helper()
	dir := t.TempDir()
	reportDir := filepath.ToSlash(dir)
	steps, logs := v040GateStepsForOrder(t, order, reportDir)
	summary := releaseGateSummary{
		Status:             "pass",
		ReleaseVersion:     "v0.4.0",
		ReleaseArtifact:    "tetra.release.v0_4_0.gate-report.v1",
		ReleaseGateCommand: "bash scripts/release/v0_4_0/gate.sh",
		StartedAt:          "2026-05-20T10:00:00Z",
		EndedAt:            "2026-05-20T10:00:02Z",
		StepCount:          len(steps),
		FailedCount:        0,
		ReportDir:          reportDir,
		Steps:              steps,
	}
	for _, log := range logs {
		writeReleaseGateLog(t, dir, log)
	}
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "summary.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	writeV040SecurityReviewDetachedHash(
		t,
		dir,
		v040FixtureSHA(v040SecurityReviewArtifactPath)+"  "+v040SecurityReviewArtifactPath,
	)
	return dir
}

func mutateV040ReleaseGateSummaryStep(
	t *testing.T,
	dir string,
	name string,
	mutate func(*releaseGateStep),
) {
	t.Helper()
	mutateV040ReleaseGateSummary(t, dir, func(summary *releaseGateSummary) {
		for i := range summary.Steps {
			if summary.Steps[i].Name == name {
				mutate(&summary.Steps[i])
				return
			}
		}
		t.Fatalf("summary missing step %q", name)
	})
}

func mutateV040ReleaseGateSummary(t *testing.T, dir string, mutate func(*releaseGateSummary)) {
	t.Helper()
	path := filepath.Join(dir, "summary.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var summary releaseGateSummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatal(err)
	}
	mutate(&summary)
	raw, err = json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func v040GateStepsForOrder(
	t *testing.T,
	order []string,
	reportDir string,
) ([]releaseGateStep, []string) {
	t.Helper()
	var steps []releaseGateStep
	var logs []string
	for _, name := range order {
		command, ok := expectedV040RequiredStepCommand(name, reportDir)
		if !ok {
			t.Fatalf("unknown v0.4 command fixture %q", name)
		}
		log := fmt.Sprintf("logs/%02d-%s.log", len(steps)+1, stepLogSlug(name))
		steps = append(steps, releaseGateStep{
			Name:            name,
			Status:          "pass",
			DurationSeconds: 1,
			ExitCode:        0,
			Command:         command,
			Log:             log,
		})
		logs = append(logs, log)
	}
	return steps, logs
}

func v040ReleaseGateSummaryExpectations() releaseGateSummaryExpectations {
	return releaseGateSummaryExpectations{
		ReleaseVersion:     "v0.4.0",
		ReleaseArtifact:    "tetra.release.v0_4_0.gate-report.v1",
		ReleaseGateCommand: "bash scripts/release/v0_4_0/gate.sh",
	}
}

var v040GateStepFixtures = []struct {
	Name string
}{
	{Name: "readiness preflight"},
	{Name: "version parity"},
	{Name: "readiness validator tests"},
	{Name: "docs verification"},
	{Name: "techempower report schemas"},
	{Name: "compiler cli tools baseline"},
	{Name: "memory production linux x64 smoke"},
	{Name: "validate memory production"},
	{Name: "parallel production linux x64 smoke"},
	{Name: "validate parallel production"},
	{Name: "compiler production linux x64 smoke"},
	{Name: "validate compiler production"},
	{Name: "linux host smoke"},
	{Name: "distributed actors linux x64 smoke"},
	{Name: "validate distributed actor runtime"},
	{Name: "native ui linux x64 smoke"},
	{Name: "validate native ui runtime"},
	{Name: "readiness final"},
	{Name: "completion audit validation"},
	{Name: "release state"},
	{Name: "security review signoff"},
	{Name: "security review detached hash"},
	{Name: "diff check"},
}

func v040GateStepNames() []string {
	names := make([]string, 0, len(v040GateStepFixtures))
	for _, fixture := range v040GateStepFixtures {
		names = append(names, fixture.Name)
	}
	return names
}

func stepLogSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.NewReplacer(" ", "-", "/", "-", ".", "-").Replace(slug)
	return strings.Trim(slug, "-")
}

func v040ProductionArtifactHashesManifest() string {
	return v040ArtifactHashesManifestExcept()
}

func v040ArtifactHashesManifestForSummary(t *testing.T, dir string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var summary releaseGateSummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatal(err)
	}
	var logs []string
	for _, step := range summary.Steps {
		logs = append(logs, step.Log)
	}
	return v040ArtifactHashesManifestWithLogs(logs)
}

func v040ArtifactHashesManifestWithRoot(t *testing.T, dir string, root string) string {
	t.Helper()
	raw := v040ArtifactHashesManifestForSummary(t, dir)
	var manifest releaseArtifactHashesManifest
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Root = root
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func v040ArtifactHashesManifestWithDuplicatePath(t *testing.T, dir string, path string) string {
	t.Helper()
	raw := v040ArtifactHashesManifestForSummary(t, dir)
	var manifest releaseArtifactHashesManifest
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range manifest.Artifacts {
		if artifact.Path == path {
			manifest.Artifacts = append(manifest.Artifacts, artifact)
			out, err := json.MarshalIndent(manifest, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			return string(out)
		}
	}
	t.Fatalf("artifact path %q not found", path)
	return ""
}

func v040ArtifactHashesManifestWithExtraPath(t *testing.T, dir string, path string) string {
	t.Helper()
	raw := v040ArtifactHashesManifestForSummary(t, dir)
	var manifest releaseArtifactHashesManifest
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Artifacts = append(manifest.Artifacts, releaseHashArtifact{
		Path:   path,
		SHA256: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Size:   1,
	})
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func v040ArtifactHashesManifestWithUnsortedPaths(t *testing.T, dir string) string {
	t.Helper()
	raw := v040ArtifactHashesManifestForSummary(t, dir)
	var manifest releaseArtifactHashesManifest
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		t.Fatal(err)
	}
	sort.Slice(manifest.Artifacts, func(i, j int) bool {
		return manifest.Artifacts[i].Path < manifest.Artifacts[j].Path
	})
	if len(manifest.Artifacts) < 2 {
		t.Fatalf("need at least two artifacts to make unsorted manifest")
	}
	manifest.Artifacts[0], manifest.Artifacts[1] = manifest.Artifacts[1], manifest.Artifacts[0]
	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func v040ArtifactHashesManifestWithMutatedArtifact(
	t *testing.T,
	dir string,
	path string,
	mutate func(*releaseHashArtifact),
) string {
	t.Helper()
	raw := v040ArtifactHashesManifestForSummary(t, dir)
	var manifest releaseArtifactHashesManifest
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		t.Fatal(err)
	}
	for i := range manifest.Artifacts {
		if manifest.Artifacts[i].Path == path {
			mutate(&manifest.Artifacts[i])
			out, err := json.MarshalIndent(manifest, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			return string(out)
		}
	}
	t.Fatalf("artifact path %q not found", path)
	return ""
}

func v040ArtifactHashesManifestExcept(omittedPaths ...string) string {
	omitted := make(map[string]bool, len(omittedPaths))
	for _, path := range omittedPaths {
		omitted[path] = true
	}
	return v040ArtifactHashesManifestWithOmissionsAndLogs(omitted, v040CanonicalStepLogs())
}

func v040ArtifactHashesManifestWithLogs(logs []string) string {
	return v040ArtifactHashesManifestWithOmissionsAndLogs(map[string]bool{}, logs)
}

func v040ArtifactHashesManifestWithOmissionsAndLogs(omitted map[string]bool, logs []string) string {
	manifest := releaseArtifactHashesManifest{
		Schema: releaseArtifactHashesSchema,
		Root:   ".",
	}
	for i, fixture := range v040ArtifactHashFixtures {
		if omitted[fixture.Path] {
			continue
		}
		artifact := fixture
		artifact.SHA256 = fmt.Sprintf("sha256:%064x", i+1)
		artifact.Size = int64(i + 1)
		manifest.Artifacts = append(manifest.Artifacts, artifact)
	}
	offset := len(v040ArtifactHashFixtures)
	for i, path := range logs {
		if omitted[path] {
			continue
		}
		manifest.Artifacts = append(manifest.Artifacts, releaseHashArtifact{
			Path:   path,
			SHA256: fmt.Sprintf("sha256:%064x", offset+i+1),
			Size:   int64(offset + i + 1),
		})
	}
	sort.Slice(manifest.Artifacts, func(i, j int) bool {
		return manifest.Artifacts[i].Path < manifest.Artifacts[j].Path
	})
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func v040CanonicalStepLogs() []string {
	logs := make([]string, 0, len(v040GateStepFixtures))
	for i, fixture := range v040GateStepFixtures {
		path := fmt.Sprintf("logs/%02d-%s.log", i+1, stepLogSlug(fixture.Name))
		logs = append(logs, path)
	}
	return logs
}

var v040ArtifactHashFixtures = []releaseHashArtifact{
	{Path: "summary.json"},
	{Path: "summary.md"},
	{Path: "artifacts/features.json", Schema: "tetra.features.v1"},
	{Path: "artifacts/targets.json"},
	{Path: "artifacts/linux-host-smoke.json"},
	{Path: "artifacts/memory-production-linux-x64.json", Schema: "tetra.memory.production.v1"},
	{Path: "artifacts/parallel-production-linux-x64.json", Schema: "tetra.parallel.production.v1"},
	{Path: "artifacts/compiler-production-linux-x64.json", Schema: "tetra.compiler.production.v1"},
	{
		Path:   "artifacts/distributed-actors-linux-x64.json",
		Schema: "tetra.actors.distributed-runtime.v1",
	},
	{Path: "artifacts/native-ui-linux-x64.json", Schema: "tetra.ui.native-runtime.v1"},
	{Path: "artifacts/release-state.json", Schema: "tetra.release.v0_4_0.release-state.v1"},
	{Path: "artifacts/release-state.txt"},
	{Path: v040SecurityReviewArtifactPath},
	{Path: v040SecurityReviewDetachedHashPath},
}

func v040FixtureSHA(path string) string {
	for i, fixture := range v040ArtifactHashFixtures {
		if fixture.Path == path {
			return fmt.Sprintf("sha256:%064x", i+1)
		}
	}
	panic(fmt.Sprintf("missing v0.4 artifact fixture %q", path))
}

func writeReleaseGateArtifactHashes(t *testing.T, dir string, manifest string) {
	t.Helper()
	if err := os.WriteFile(
		filepath.Join(dir, "artifact-hashes.json"),
		[]byte(manifest),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
}
