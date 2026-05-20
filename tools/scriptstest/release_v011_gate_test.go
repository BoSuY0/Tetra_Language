package scriptstest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV011GateDocumentsMandatoryTargets(t *testing.T) {
	root := repoRoot(t)
	assertLegacyFileRemoved(t, "scripts/release_v0_1_1_gate.sh", "scripts/release/v0_1_1/gate.sh directly")
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "v0_1_1", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.1.1 release gate: %v", err)
	}
	text := string(raw)
	for _, target := range []string{"linux-x64", "macos-x64", "windows-x64", "wasm32-wasi", "wasm32-web"} {
		if !strings.Contains(text, "--target "+target) {
			t.Fatalf("v0.1.1 release gate missing target %s", target)
		}
	}
}

func TestReleaseV011GateKeepsCurrentValidators(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v0_1_1", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.1.1 release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"bash scripts/ci/test-all.sh --full",
		"go run ./tools/cmd/validate-flow-only",
		"./tetra targets --format=json",
		"go run ./tools/cmd/validate-targets",
		"./tetra doctor --format=json",
		"go run ./tools/cmd/validate-doctor",
		"./tetra check examples/flow_hello.tetra",
		"./tetra doc examples",
		"./t version",
		"go run ./tools/cmd/validate-test-report",
		"go run ./tools/cmd/validate-manifest",
		"go run ./tools/cmd/verify-docs",
		"go run ./tools/cmd/smoke-report-to-checklist --validate-only",
		"go run ./tools/cmd/validate-web-ui-smoke",
		"./tetra smoke --list --format=json",
		"go run ./tools/cmd/validate-smoke-list",
		"go run ./tools/cmd/validate-api-docs",
		"bash scripts/release/v1_0/security-review.sh",
		"bash scripts/release/v1_0/binary-size.sh",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.1.1 release gate missing %q", want)
		}
	}
}

func TestReleaseV011GateRecordsBinarySizeEvidenceBeforeRepro(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v0_1_1", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.1.1 release gate: %v", err)
	}
	text := string(raw)
	sizeIdx := strings.Index(text, `run_step "binary size thresholds" check_binary_size_thresholds`)
	if sizeIdx < 0 {
		t.Fatalf("v0.1.1 release gate missing binary size threshold step")
	}
	reproIdx := strings.Index(text, `run_step "reproducible build proof" check_repro_build`)
	if reproIdx < 0 {
		t.Fatalf("v0.1.1 release gate missing repro proof step")
	}
	if sizeIdx > reproIdx {
		t.Fatalf("binary size evidence should be recorded before reproducibility proof")
	}
}

func TestReleaseV011GateValidatesJSONDiagnostics(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v0_1_1", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.1.1 release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "json diagnostic shape" check_json_diagnostic`,
		`check_json_diagnostic_case "invalid-diagnostic" "unknown function"`,
		`check_json_diagnostic_case "missing-effect-diagnostic" "uses effect 'io'"`,
		`check_json_diagnostic_case "tabs-diagnostic" "tabs are not supported"`,
		`check_json_diagnostic_case "planned-actor-diagnostic" "planned feature 'actor'"`,
		`go run ./tools/cmd/validate-diagnostic --diagnostic "$diagnostic" --severity error --contains "$contains" --require-position`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.1.1 release gate missing JSON diagnostic validation %q", want)
		}
	}
}

func TestReleaseV011GateRequiresSecurityReviewSignoff(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v0_1_1", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.1.1 release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`check_security_review_signoff()`,
		`TETRA_SECURITY_REVIEW_SIGNOFF`,
		`cp "$signoff_path" "$artifacts_dir/security-review.md"`,
		`run_step "security review signoff" check_security_review_signoff`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.1.1 release gate missing security review wiring %q", want)
		}
	}
}

func TestReleaseV011GateArchivesReleaseStateKnownIssuesAndHashes(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v0_1_1", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.1.1 release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`check_release_state()`,
		`go run ./tools/cmd/validate-release-state --expected-version "$release_version" --format=json --report-dir "$report_dir" >"$artifacts_dir/release-state.json"`,
		`go run ./tools/cmd/validate-release-state --expected-version "$release_version" --format=text --report-dir "$report_dir" >"$artifacts_dir/release-state.txt"`,
		`write_known_issues_artifact()`,
		`"$artifacts_dir/known_issues.md"`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$artifacts_dir" --out "$artifacts_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$artifacts_dir/artifact-hashes.json"`,
		`run_step "release state audit" check_release_state`,
		`run_step "known issues artifact" write_known_issues_artifact`,
		`run_step "artifact hash manifest" check_artifact_hash_manifest`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.1.1 release gate missing release evidence step %q", want)
		}
	}
}

func TestReleaseV011GateChecksGeneratedArtifactChurn(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v0_1_1", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.1.1 release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`capture_generated_artifact_state "$generated_state_before"`,
		`check_generated_artifact_churn()`,
		`git status --porcelain --untracked-files=no -- docs/generated docs/baselines`,
		`git diff --binary -- docs/generated docs/baselines`,
		`run_step "generated artifact churn check" check_generated_artifact_churn`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.1.1 release gate missing generated churn guard %q", want)
		}
	}
}

func TestReleaseV011GateRunsVersionPreflightBeforePackageTests(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v0_1_1", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.1.1 release gate: %v", err)
	}
	text := string(raw)
	versionIdx := strings.Index(text, `run_step "version preflight (v0.1.1 required)"`)
	if versionIdx < 0 {
		t.Fatalf("v0.1.1 release gate missing version preflight step")
	}
	goTestIdx := strings.Index(text, `run_step "go test packages"`)
	if goTestIdx < 0 {
		t.Fatalf("v0.1.1 release gate missing go test packages step")
	}
	if versionIdx > goTestIdx {
		t.Fatalf("v0.1.1 release gate must hard-block on version before package tests")
	}
}

func TestReleaseV011GateRejectsMissingReportDirArgument(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_1_1/gate.sh")

	out, err := runOldReleaseGate(t, root, "scripts/release/v0_1_1/gate.sh", "--report-dir")
	if err == nil {
		t.Fatalf("expected missing report-dir argument rejection\n%s", out)
	}
	if !strings.Contains(string(out), "release_v0_1_1_gate: --report-dir requires a directory") {
		t.Fatalf("missing report-dir argument output missing controlled error:\n%s", out)
	}
}

func TestReleaseV011GateRejectsNonDirectoryReportPathBeforeSideEffects(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_1_1/gate.sh")
	reportDir := filepath.Join(root, "report-file")
	if err := os.WriteFile(reportDir, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertOldGateRejectsNonDirectoryReportPath(t, root, "scripts/release/v0_1_1/gate.sh", "release_v0_1_1_gate:", reportDir)
}

func TestReleaseV011GateRejectsDanglingReportDirSymlinkBeforeSideEffects(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_1_1/gate.sh")
	reportDir := filepath.Join(root, "dangling-report-link")
	if err := os.Symlink(filepath.Join(root, "missing-report-target"), reportDir); err != nil {
		t.Fatal(err)
	}

	assertOldGateRejectsNonDirectoryReportPath(t, root, "scripts/release/v0_1_1/gate.sh", "release_v0_1_1_gate:", reportDir)
}

func TestReleaseV011GateRejectsNonEmptyReportDirBeforeSideEffects(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_1_1/gate.sh")
	reportDir := filepath.Join(root, "report")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertOldGateRejectsNonEmptyReportDir(t, root, "scripts/release/v0_1_1/gate.sh", "release_v0_1_1_gate:", reportDir)
}

func TestReleaseV011GateRejectsDashPrefixedNonEmptyReportDirBeforeSideEffects(t *testing.T) {
	root := releaseOldGateMinimalRoot(t, "scripts/release/v0_1_1/gate.sh")
	reportDirArg := "-stale-report"
	reportDir := filepath.Join(root, reportDirArg)
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertOldGateRejectsNonEmptyReportDirWithArg(t, root, "scripts/release/v0_1_1/gate.sh", "release_v0_1_1_gate:", reportDirArg, reportDir)
}

func releaseOldGateMinimalRoot(t *testing.T, gateRel string) string {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{
		"scripts/dev",
		filepath.Dir(gateRel),
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyFile(filepath.Join(repoRoot(t), filepath.FromSlash(gateRel)), filepath.Join(root, filepath.FromSlash(gateRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "dev", "bootstrap.sh"), []byte(`#!/usr/bin/env bash
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

func runOldReleaseGate(t *testing.T, root, gateRel string, args ...string) ([]byte, error) {
	t.Helper()

	cmdArgs := append([]string{filepath.ToSlash(gateRel)}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = root
	return cmd.CombinedOutput()
}

func assertOldGateRejectsNonDirectoryReportPath(t *testing.T, root, gateRel, prefix, reportDir string) {
	t.Helper()

	out, err := runOldReleaseGate(t, root, gateRel, "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected non-directory report dir to block gate\n%s", out)
	}
	for _, want := range []string{
		prefix + " refusing to use non-directory report path: " + reportDir,
		prefix + " choose a fresh --report-dir directory",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("non-directory report dir output missing %q:\n%s", want, out)
		}
	}
	assertOldGateRejectedBeforeSideEffects(t, out)
}

func assertOldGateRejectsNonEmptyReportDir(t *testing.T, root, gateRel, prefix, reportDir string) {
	t.Helper()

	assertOldGateRejectsNonEmptyReportDirWithArg(t, root, gateRel, prefix, reportDir, reportDir)
}

func assertOldGateRejectsNonEmptyReportDirWithArg(t *testing.T, root, gateRel, prefix, reportDirArg, reportDirPath string) {
	t.Helper()

	expectedReportDir := normalizeDashLeadingReportDirForTest(reportDirArg)
	out, err := runOldReleaseGate(t, root, gateRel, "--report-dir", reportDirArg)
	if err == nil {
		t.Fatalf("expected non-empty report dir to block gate\n%s", out)
	}
	for _, want := range []string{
		prefix + " refusing to reuse non-empty report directory: " + expectedReportDir,
		prefix + " choose a fresh --report-dir so stale reports cannot be reused",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("non-empty report dir output missing %q:\n%s", want, out)
		}
	}
	assertOldGateRejectedBeforeSideEffects(t, out)
	assertOldGateDidNotCreateReportSubdirs(t, reportDirPath)
}

func assertOldGateRejectedBeforeSideEffects(t *testing.T, out []byte) {
	t.Helper()

	for _, forbidden := range []string{
		"bootstrap should not run",
		"tetra should not run",
		"find:",
		"mkdir:",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf("gate should reject report dir before raw shell or workflow side effects:\n%s", out)
		}
	}
}

func assertOldGateDidNotCreateReportSubdirs(t *testing.T, reportDir string) {
	t.Helper()

	for _, name := range []string{"logs", "artifacts"} {
		path := filepath.Join(reportDir, name)
		if _, err := os.Lstat(path); err == nil {
			t.Fatalf("gate should reject stale report dir before creating %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat report side-effect path %s: %v", path, err)
		}
	}
}

func normalizeDashLeadingReportDirForTest(reportDir string) string {
	if strings.HasPrefix(reportDir, "-") {
		return "./" + reportDir
	}
	return reportDir
}
