package release_v030_static

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV030GateRequireCleanRejectsDirtyWorktree(t *testing.T) {
	root := releaseV030FakeRepo(t)
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	git := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "status" ]]; then
  printf ' M README.md\n'
  printf '?? scratch.txt\n'
  exit 0
fi
echo "unexpected git command: $*" >&2
exit 2
`
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(git), 0o755); err != nil {
		t.Fatal(err)
	}

	reportDir := filepath.Join(root, "report")
	cmd := exec.Command(
		"bash",
		"scripts/release/v0_3_0/gate.sh",
		"--require-clean",
		"--report-dir",
		reportDir,
	)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected --require-clean to reject dirty worktree\n%s", out)
	}
	for _, want := range []string{
		"tag-ready clean worktree required",
		"git status --porcelain --untracked-files=all",
		" M README.md",
		"?? scratch.txt",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("--require-clean dirty output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "bootstrapping local binaries") {
		t.Fatalf("--require-clean should block before release gate side effects:\n%s", out)
	}
}

func TestReleaseV030GateValidatesFuzzArtifactsAfterShortFuzz(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release/v0_3_0/gate.sh"))
	if err != nil {
		t.Fatalf("read v0.3.0 release gate: %v", err)
	}
	text := string(raw)
	shortFuzz := `run_step "short fuzz smoke" bash scripts/dev/fuzz-nightly.sh --short --out-dir "$artifacts_dir/fuzz-short"`
	validateFuzz := `run_step "fuzz artifact validation" check_short_fuzz_summary`
	if !strings.Contains(text, `check_short_fuzz_summary()`) {
		t.Fatalf("v0.3.0 release gate missing fuzz summary validator function")
	}
	if !strings.Contains(text, validateFuzz) {
		t.Fatalf("v0.3.0 release gate missing fuzz artifact validation step")
	}
	if strings.Index(text, validateFuzz) < strings.Index(text, shortFuzz) {
		t.Fatalf("v0.3.0 release gate validates fuzz artifacts before producing them")
	}
}

func TestReleaseV030GateRefreshesReleaseStateAfterFinalSummaryWrite(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release/v0_3_0/gate.sh"))
	if err != nil {
		t.Fatalf("read v0.3.0 release gate: %v", err)
	}
	text := string(raw)
	if !finalReleaseStateRefreshFollowsSummary(text) {
		t.Fatalf(
			("v0.3.0 release gate must refresh release-state after the final " +
				"pass summary is written and validated"),
		)
	}
	regressed := `if check_release_state; then
  :
fi
write_summary "pass"
validate_summary`
	if finalReleaseStateRefreshFollowsSummary(regressed) {
		t.Fatalf("regression guard failed: release-state refresh before final summary was accepted")
	}
}

func TestReleaseV030GateWritesBlockedReleaseStateBeforeCIMissingSignoffExit(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release/v0_3_0/gate.sh"))
	if err != nil {
		t.Fatalf("read v0.3.0 release gate: %v", err)
	}
	text := string(raw)
	summaryBeforeCIBranch := `if [[ "$failed_count" -eq 0 && "$ci_missing_security_signoff" -eq 0 ]]; then
  write_summary "pass"
else
  write_summary "blocked"
fi
validate_summary
if [[ "$failed_count" -eq 0 && "$ci_missing_security_signoff" -eq 1 ]]; then`
	if !strings.Contains(text, summaryBeforeCIBranch) {
		t.Fatalf(
			"v0.3.0 release gate must write and validate summary.json before CI missing-signoff branch",
		)
	}
	ciBlockedReleaseState := `write_summary "blocked"
  validate_summary
  if check_release_state; then`
	if !strings.Contains(text, ciBlockedReleaseState) {
		t.Fatalf(
			"v0.3.0 release gate must write blocked release-state artifacts before CI missing-signoff exit",
		)
	}
	if strings.Index(
		text,
		`write_summary "pass"`,
	) > strings.Index(
		text,
		`run_step "release state audit" check_release_state`,
	) {
		t.Fatalf("v0.3.0 release gate writes pass summary after release-state audit")
	}
	if strings.Index(
		text,
		`validate_summary`,
	) > strings.Index(
		text,
		`run_step "release state audit" check_release_state`,
	) {
		t.Fatalf("v0.3.0 release gate validates summary after release-state audit")
	}
}

func TestReleaseV030GateValidatesGateSummaryArtifacts(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release/v0_3_0/gate.sh"))
	if err != nil {
		t.Fatalf("read v0.3.0 release gate: %v", err)
	}
	text := string(raw)
	wantFunction := `validate_summary() {
  go run ./tools/cmd/validate-release-gate-summary --summary "$summary_json" --report-dir "$report_dir"
}`
	if !strings.Contains(text, wantFunction) {
		t.Fatalf("v0.3.0 release gate missing summary validator function")
	}
	if !strings.Contains(text, `write_summary "pass"
validate_summary
if check_release_state; then`) {
		t.Fatalf(
			"v0.3.0 release gate must validate final summary before guarded final release-state refresh",
		)
	}
	if !strings.Contains(text, `if check_artifact_hash_manifest; then`) {
		t.Fatalf("v0.3.0 release gate must guard final artifact hash validation")
	}
	if !strings.Contains(text, `if write_security_review_detached_hash; then`) {
		t.Fatalf("v0.3.0 release gate must guard detached security hash generation")
	}
}

func TestReleaseV030GateHashesEntireReportDirectory(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release/v0_3_0/gate.sh"))
	if err != nil {
		t.Fatalf("read v0.3.0 release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.3.0 release gate must hash the full report directory, missing %q", want)
		}
	}
	for _, stale := range []string{
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$artifacts_dir" --out "$artifacts_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$artifacts_dir/artifact-hashes.json"`,
	} {
		if strings.Contains(text, stale) {
			t.Fatalf("v0.3.0 release gate still hashes only artifacts/: %q", stale)
		}
	}
}

func finalReleaseStateRefreshFollowsSummary(text string) bool {
	finalSummary := `write_summary "pass"
validate_summary`
	summaryIdx := strings.LastIndex(text, finalSummary)
	refreshIdx := strings.LastIndex(text, `if check_release_state; then`)
	return summaryIdx >= 0 && refreshIdx > summaryIdx
}
