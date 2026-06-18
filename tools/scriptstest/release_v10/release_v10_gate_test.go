package release_v10

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV10GateUsesRealV1Boundary(t *testing.T) {
	root := repoRoot(t)
	assertLegacyFileRemoved(
		t,
		"scripts/release_v1_0_gate.sh",
		"scripts/release/v1_0/gate.sh directly",
	)
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "v1_0", "gate.sh"))
	if err != nil {
		t.Fatalf("read v1.0 release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/v1_0/gate.sh [--report-dir DIR]",
		`release_version="v1.0.0"`,
		`release_artifact="tetra.release.v1_0.gate-report.v1"`,
		`bash scripts/dev/bootstrap.sh`,
		`run_step "bootstrap tetra binaries" bash scripts/dev/bootstrap.sh`,
		`run_step "go test packages" env \`,
		`-u TETRA_SECURITY_REVIEW_SIGNOFF \`,
		`-u TETRA_TEST_ALL_RELEASE_VERSION \`,
		`-u TETRA_TEST_ALL_RELEASE_ARTIFACT \`,
		`go test ./compiler/... ./cli/... ./tools/... -count=1`,
		`run_step "full stabilization wrapper" env \`,
		`TETRA_TEST_ALL_RELEASE_VERSION="$release_version" \`,
		`TETRA_TEST_ALL_RELEASE_ARTIFACT="tetra.release.v1_0_0.test-all-summary.v1" \`,
		`bash scripts/ci/test-all.sh \`,
		`--report-dir "$artifacts_dir/test-all"`,
		`if [[ "$version" != "$release_version" ]]`,
		`expected ./tetra version to be $release_version`,
		`release_gate_command="bash scripts/release/v1_0/gate.sh"`,
		`TETRA_TEST_ALL_RELEASE_ARTIFACT="tetra.release.v1_0_0.test-all-summary.v1"`,
		`run_step "WASI runner smoke" check_wasi_runner_smoke`,
		`run_step "Web runtime browser smoke" check_web_runtime_smoke`,
		`run_step "security review detached hash" write_security_review_detached_hash`,
		`run_step "handoff signoff lint" check_handoff_signoff_lint`,
		`run_step "WASI artifact/import smoke"`,
		`go run ./tools/cmd/validate-wasi-smoke-report \`,
		`--mode artifact`,
		`--report "$report"`,
		`run_step "Web artifact/import smoke"`,
		`run_step "build-only smoke linux-x64"`,
		`run_step "build-only smoke macos-x64"`,
		`run_step "build-only smoke windows-x64"`,
		`run_step "backend summary artifact" check_backend_summary`,
		`run_step "API diff gate" check_api_diff`,
		`go run ./tools/cmd/validate-performance-report --report "$dst" --stamp-git-head "$current_head"`,
		`run_step "reproducible build proof" check_repro_build`,
		`run_step "release state audit" check_release_state`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v1.0 release gate missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"compatibility alias",
		"release/v0_1_3/gate.sh",
		"release/v0_2_0/gate.sh",
		`TETRA_RELEASE_GATE_VERSION`,
		`TETRA_RELEASE_GATE_ARTIFACT`,
		`TETRA_RELEASE_GATE_COMMAND`,
		`exec bash "$script_dir/release/v0_2_0/gate.sh" "$@"`,
		`scripts/release_v1_0_gate.sh`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("v1.0 release gate still contains alias marker %q", forbidden)
		}
	}
	summaryBeforeReleaseState := `check_release_state() {
  if [[ "$failed_count" -gt 0 ]]; then
    write_summary "blocked"
  else
    write_summary "pass"
  fi
  go run ./tools/cmd/validate-release-state`
	if !strings.Contains(text, summaryBeforeReleaseState) {
		t.Fatalf("v1.0 release-state audit must refresh summary before validate-release-state")
	}
	artifactIdx := strings.Index(
		text,
		`run_step "artifact hash manifest" check_artifact_hash_manifest`,
	)
	releaseStateIdx := strings.Index(text, `run_step "release state audit" check_release_state`)
	if artifactIdx == -1 || releaseStateIdx == -1 || artifactIdx > releaseStateIdx {
		t.Fatalf("v1.0 release gate must build artifact hash manifest before release-state audit")
	}
	for _, want := range []string{
		`write_summary "pass"
if ! check_artifact_hash_manifest; then`,
		`if ! check_release_state; then`,
		`release/v1_0/gate: blocked: final artifact hash refresh failed`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v1.0 release gate final refresh sequence missing %q", want)
		}
	}
}

func TestReleaseV10GateRunsDedicatedV1Workflow(t *testing.T) {
	root := releaseV10GateFakeRepo(t)
	reportDir := filepath.Join(root, "report")
	cmd := exec.Command("bash", "scripts/release/v1_0/gate.sh", "--report-dir", reportDir)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TETRA_SECURITY_REVIEW_SIGNOFF="+filepath.Join(root, "security-review.md"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("v1.0 gate should run dedicated fake workflow: %v\n%s", err, out)
	}

	rawSummary, err := os.ReadFile(filepath.Join(reportDir, "summary.json"))
	if err != nil {
		t.Fatalf("read v1.0 gate summary: %v\n%s", err, out)
	}
	var summary struct {
		Status             string `json:"status"`
		ReleaseVersion     string `json:"release_version"`
		ReleaseArtifact    string `json:"release_artifact"`
		ReleaseGateCommand string `json:"release_gate_command"`
		Steps              []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Command string `json:"command"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(rawSummary, &summary); err != nil {
		t.Fatalf("unmarshal v1.0 gate summary: %v\n%s", err, string(rawSummary))
	}
	if summary.Status != "pass" {
		t.Fatalf("summary status = %q\n%s", summary.Status, string(rawSummary))
	}
	if summary.ReleaseVersion != "v1.0.0" {
		t.Fatalf("release_version = %q", summary.ReleaseVersion)
	}
	if summary.ReleaseArtifact != "tetra.release.v1_0.gate-report.v1" {
		t.Fatalf("release_artifact = %q", summary.ReleaseArtifact)
	}
	if summary.ReleaseGateCommand != "bash scripts/release/v1_0/gate.sh" {
		t.Fatalf("release_gate_command = %q", summary.ReleaseGateCommand)
	}

	seen := map[string]bool{}
	for _, step := range summary.Steps {
		seen[step.Name] = true
		if strings.Contains(step.Command, "release/v0_1_3/gate.sh") {
			t.Fatalf("v1.0 gate delegated to v0.1.3 in step %q: %s", step.Name, step.Command)
		}
	}
	for _, want := range []string{
		"bootstrap tetra binaries",
		"WASI runner smoke",
		"Web runtime browser smoke",
		"WASI artifact/import smoke",
		"Web artifact/import smoke",
		"build-only smoke linux-x64",
		"build-only smoke macos-x64",
		"build-only smoke windows-x64",
		"backend summary artifact",
		"handoff signoff lint",
		"API diff gate",
		"reproducible build proof",
		"release state audit",
	} {
		if !seen[want] {
			t.Fatalf(
				"v1.0 dedicated workflow missing step %q in summary:\n%s",
				want,
				string(rawSummary),
			)
		}
	}
	for _, artifact := range []string{
		"logs/01-bootstrap-tetra-binaries.log",
		"artifacts/wasi-smoke.json",
		"artifacts/web-ui-smoke.json",
		"artifacts/backend-summary.md",
		"artifacts/api-diff/api-diff.json",
		"artifacts/reproducible-build.json",
	} {
		if _, err := os.Stat(filepath.Join(reportDir, filepath.FromSlash(artifact))); err != nil {
			t.Fatalf("missing v1.0 gate artifact %s: %v", artifact, err)
		}
	}
	releaseStateRaw, err := os.ReadFile(filepath.Join(reportDir, "artifacts", "release-state.json"))
	if err != nil {
		t.Fatalf("read v1.0 release-state artifact: %v", err)
	}
	var releaseState struct {
		LastGateEvidence struct {
			Status      string `json:"status"`
			StepCount   int    `json:"step_count"`
			FailedCount int    `json:"failed_count"`
		} `json:"last_gate_evidence"`
	}
	if err := json.Unmarshal(releaseStateRaw, &releaseState); err != nil {
		t.Fatalf("decode v1.0 release-state artifact: %v\n%s", err, releaseStateRaw)
	}
	if releaseState.LastGateEvidence.Status != summary.Status {
		t.Fatalf(
			"release-state status %q contradicts final summary status %q",
			releaseState.LastGateEvidence.Status,
			summary.Status,
		)
	}
	if releaseState.LastGateEvidence.StepCount != len(summary.Steps) {
		t.Fatalf(
			"release-state step_count %d contradicts final summary steps %d",
			releaseState.LastGateEvidence.StepCount,
			len(summary.Steps),
		)
	}
	if releaseState.LastGateEvidence.FailedCount != 0 {
		t.Fatalf(
			"release-state failed_count = %d, want 0",
			releaseState.LastGateEvidence.FailedCount,
		)
	}

	hashRaw, err := os.ReadFile(filepath.Join(reportDir, "artifacts", "artifact-hashes.json"))
	if err != nil {
		t.Fatalf("read v1.0 artifact hash manifest: %v", err)
	}
	var hashManifest struct {
		Artifacts []struct {
			Path string `json:"path"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal(hashRaw, &hashManifest); err != nil {
		t.Fatalf("decode v1.0 artifact hash manifest: %v\n%s", err, hashRaw)
	}
	hashed := map[string]bool{}
	for _, artifact := range hashManifest.Artifacts {
		hashed[artifact.Path] = true
	}
	for _, want := range []string{"release-state.json", "release-state.txt"} {
		if !hashed[want] {
			t.Fatalf(
				"artifact-hashes.json must include %s after final release-state refresh:\n%s",
				want,
				hashRaw,
			)
		}
	}
}

func TestReleaseV10GateRejectsMissingReportDirArgument(t *testing.T) {
	root := releaseV10MinimalGateRoot(t)

	cmd := exec.Command("bash", "scripts/release/v1_0/gate.sh", "--report-dir")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected missing report-dir argument rejection\n%s", out)
	}
	if !strings.Contains(string(out), "release/v1_0/gate: --report-dir requires a directory") {
		t.Fatalf("missing report-dir argument output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	for _, forbidden := range []string{
		"bootstrap should not run",
		"tetra should not run",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf(
				"missing report-dir argument should reject before workflow side effects:\n%s",
				out,
			)
		}
	}
}

func TestReleaseV10GateRejectsNonDirectoryReportPathBeforeSideEffects(t *testing.T) {
	root := releaseV10MinimalGateRoot(t)
	reportDir := filepath.Join(root, "report-file")
	if err := os.WriteFile(reportDir, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertReleaseV10RejectsNonDirectoryReportPath(t, root, reportDir)
}

func TestReleaseV10GateRejectsDanglingReportDirSymlinkBeforeSideEffects(t *testing.T) {
	root := releaseV10MinimalGateRoot(t)
	reportDir := filepath.Join(root, "dangling-report-link")
	if err := os.Symlink(filepath.Join(root, "missing-report-target"), reportDir); err != nil {
		t.Fatal(err)
	}

	assertReleaseV10RejectsNonDirectoryReportPath(t, root, reportDir)
}

func TestReleaseV10GateRejectsNonEmptyReportDirBeforeSideEffects(t *testing.T) {
	root := releaseV10MinimalGateRoot(t)
	reportDir := filepath.Join(root, "report")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(reportDir, "summary.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release/v1_0/gate.sh", "--report-dir", reportDir)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-empty report dir to block gate\n%s", out)
	}
	for _, want := range []string{
		"release/v1_0/gate: refusing to reuse non-empty report directory: " + reportDir,
		"release/v1_0/gate: choose a fresh --report-dir so stale reports cannot be reused",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("non-empty report dir output missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{
		"bootstrap should not run",
		"tetra should not run",
		"mkdir:",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf(
				"gate should reject stale report dir before raw shell or workflow side effects:\n%s",
				out,
			)
		}
	}
}

func TestReleaseV10GateAcceptsDashPrefixedFreshReportDirBeforeBootstrapSentinel(t *testing.T) {
	root := releaseV10MinimalGateRoot(t)
	reportDirArg := "-fresh-v10-report"

	cmd := exec.Command("bash", "scripts/release/v1_0/gate.sh", "--report-dir", reportDirArg)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf(
			"minimal fake root should fail at bootstrap/version sentinel after accepting report dir\n%s",
			out,
		)
	}
	if !strings.Contains(
		string(out),
		"release/v1_0/gate: bootstrapping local binaries before v1 version preflight",
	) {
		t.Fatalf(
			"dash-prefixed fresh report dir should reach the bootstrap stage, not fail in path setup:\n%s",
			out,
		)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	reportDir := filepath.Join(root, reportDirArg)
	for _, rel := range []string{"logs", "artifacts"} {
		if _, err := os.Stat(filepath.Join(reportDir, rel)); err != nil {
			t.Fatalf(
				"dash-prefixed report dir should create normalized %s directory: %v\n%s",
				rel,
				err,
				out,
			)
		}
	}
}

func TestReleaseV10GateRejectsSymlinkReportDirBeforeSideEffects(t *testing.T) {
	root := releaseV10MinimalGateRoot(t)
	targetDir := filepath.Join(root, "report-target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(root, "report-link")
	if err := os.Symlink(targetDir, reportDir); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release/v1_0/gate.sh", "--report-dir", reportDir)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected symlink report dir rejection\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"release/v1_0/gate: refusing to use symlink report path: "+reportDir,
	) {
		t.Fatalf("symlink report dir output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	for _, forbidden := range []string{
		"bootstrap should not run",
		"tetra should not run",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf("symlink report dir should reject before workflow side effects:\n%s", out)
		}
	}
	for _, rel := range []string{"logs", "artifacts"} {
		if _, err := os.Stat(filepath.Join(targetDir, rel)); !os.IsNotExist(err) {
			t.Fatalf(
				"symlink report dir should not write through target %s, stat err = %v\n%s",
				rel,
				err,
				out,
			)
		}
	}
}

func TestReleaseV10GateAcceptsDashPrefixedSecuritySignoff(t *testing.T) {
	root := releaseV10GateFakeRepo(t)
	reportDir := filepath.Join(root, "report")
	signoffPath := "-security-review.md"
	if err := os.WriteFile(
		filepath.Join(root, signoffPath),
		[]byte("# Security Review\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release/v1_0/gate.sh", "--report-dir", reportDir)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TETRA_SECURITY_REVIEW_SIGNOFF="+signoffPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf(
			"v1.0 gate should accept dash-prefixed security signoff source path: %v\n%s",
			err,
			out,
		)
	}
	if _, err := os.Stat(filepath.Join(reportDir, "artifacts", "security-review.md")); err != nil {
		t.Fatalf(
			"security review artifact was not archived from dash-prefixed source: %v\n%s",
			err,
			out,
		)
	}
	reviewRaw, err := os.ReadFile(filepath.Join(reportDir, "artifacts", "security-review.md"))
	if err != nil {
		t.Fatalf("read archived security review: %v", err)
	}
	hashRaw, err := os.ReadFile(filepath.Join(reportDir, "artifacts", "security-review.md.sha256"))
	if err != nil {
		t.Fatalf("security review detached hash was not archived: %v\n%s", err, out)
	}
	wantHash := fmt.Sprintf("%x  artifacts/security-review.md\n", sha256.Sum256(reviewRaw))
	if string(hashRaw) != wantHash {
		t.Fatalf("security review detached hash = %q, want %q", string(hashRaw), wantHash)
	}
}

func TestReleaseV10GateWritesBlockedSecurityArtifactWhenSignoffMissing(t *testing.T) {
	root := releaseV10GateFakeRepo(t)
	reportDir := filepath.Join(root, "report")

	cmd := exec.Command("bash", "scripts/release/v1_0/gate.sh", "--report-dir", reportDir)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("v1.0 gate should block when security signoff is missing\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"release/v1_0/gate: missing TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>",
	) {
		t.Fatalf("missing signoff output did not explain the blocker:\n%s", out)
	}
	reviewPath := filepath.Join(reportDir, "artifacts", "security-review.md")
	reviewRaw, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatalf("blocked security review artifact was not written: %v\n%s", err, out)
	}
	for _, want := range []string{
		"Decision: blocked",
		"Reason: missing TETRA_SECURITY_REVIEW_SIGNOFF for the exact v1.0.0 candidate.",
		"bash scripts/release/v1_0/security-review.sh --write-template <security-review.md>",
	} {
		if !strings.Contains(string(reviewRaw), want) {
			t.Fatalf("blocked security review artifact missing %q:\n%s", want, reviewRaw)
		}
	}
	hashRaw, err := os.ReadFile(filepath.Join(reportDir, "artifacts", "security-review.md.sha256"))
	if err != nil {
		t.Fatalf("blocked security review detached hash was not written: %v\n%s", err, out)
	}
	wantHash := fmt.Sprintf("%x  artifacts/security-review.md\n", sha256.Sum256(reviewRaw))
	if string(hashRaw) != wantHash {
		t.Fatalf("blocked security review detached hash = %q, want %q", string(hashRaw), wantHash)
	}
}

func TestReleaseV10GateRejectsSecuritySignoffPlaceholders(t *testing.T) {
	root := releaseV10GateFakeRepo(t)
	reportDir := filepath.Join(root, "report")
	signoffPath := filepath.Join(root, "security-review.md")
	if err := os.WriteFile(
		signoffPath,
		[]byte("# Security Review\n\nDecision: approved\nEvidence: <fill-me>\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/release/v1_0/gate.sh", "--report-dir", reportDir)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TETRA_SECURITY_REVIEW_SIGNOFF="+signoffPath,
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("v1.0 gate should reject placeholder security signoff\n%s", out)
	}
	if !strings.Contains(string(out), "handoff/signoff lint found unresolved placeholders") {
		t.Fatalf("placeholder signoff output did not explain lint blocker:\n%s", out)
	}
}

func releaseV10MinimalGateRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{
		"scripts/dev",
		"scripts/release/v1_0",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyFile(
		filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "gate.sh"),
		filepath.Join(root, "scripts", "release", "v1_0", "gate.sh"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(
		root,
		"scripts",
		"dev",
		"bootstrap.sh",
	), []byte(`#!/usr/bin/env bash
set -euo pipefail
echo "bootstrap should not run when report dir is invalid" >&2
exit 2
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tetra"), []byte(`#!/usr/bin/env bash
set -euo pipefail
echo "tetra should not run when report dir is invalid" >&2
exit 2
`), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func assertReleaseV10RejectsNonDirectoryReportPath(t *testing.T, root, reportDir string) {
	t.Helper()

	cmd := exec.Command("bash", "scripts/release/v1_0/gate.sh", "--report-dir", reportDir)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-directory report dir to block gate\n%s", out)
	}
	for _, want := range []string{
		"release/v1_0/gate: refusing to use non-directory report path: " + reportDir,
		"release/v1_0/gate: choose a fresh --report-dir directory",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("non-directory report dir output missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{
		"bootstrap should not run",
		"tetra should not run",
		"mkdir:",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf(
				"gate should reject invalid report dir before raw shell or workflow side effects:\n%s",
				out,
			)
		}
	}
}
