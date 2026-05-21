package scriptstest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseFullPlatformUIRuntimeGateRunsMandatoryEvidence(t *testing.T) {
	path := filepath.Join(repoRoot(t), "scripts", "release", "full_platform", "ui-runtime-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read full-platform UI gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/full_platform/ui-runtime-gate.sh [--report-dir DIR]",
		"prepare_report_dir",
		"go test ./compiler/... ./cli/... ./tools/... -count=1",
		"go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json",
		"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
		"go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
		"go run ./tools/cmd/validate-targets",
		`bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-native-ui-runtime --report "$report_dir/native-ui-linux-x64.json"`,
		`bash scripts/release/post_v0_4/ui-production-runtime-linux-x64-smoke.sh --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-ui-production-runtime --report "$report_dir/ui-production-runtime-linux-x64.json"`,
		`bash scripts/release/full_platform/windows-ui-runtime-smoke.sh --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-windows-ui-runtime --report "$report_dir/windows-ui-runtime.json"`,
		`bash scripts/release/full_platform/macos-ui-runtime-smoke.sh --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-macos-ui-runtime --report "$report_dir/macos-ui-runtime.json"`,
		`bash scripts/release/v1_0/web-smoke.sh --report "$report_dir/web-smoke.json"`,
		`go run ./tools/cmd/validate-web-ui-smoke --report "$report_dir/web-smoke.json"`,
		"go run ./tools/cmd/validate-cross-platform-ui-runtime",
		"TETRA_ACTIONS_STARTUP_BLOCKER_REPORT",
		"actions_startup_blocker_report_snapshot",
		"mktemp -d",
		"actions-startup-blocker-validate",
		`go run ./tools/cmd/validate-actions-startup-blocker --report "$report_dir/github-actions-startup-blocker.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		"tetra.release.full_platform.ui_runtime.production-gate.v1",
		"TETRA_WINDOWS_UI_RUNTIME_REPORT",
		"TETRA_MACOS_UI_RUNTIME_REPORT",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("full-platform UI gate missing %q", want)
		}
	}
}

func TestReleaseFullPlatformSmokeScriptsExist(t *testing.T) {
	for _, rel := range []string{
		"scripts/release/full_platform/README.md",
		"scripts/release/full_platform/actions-availability-preflight.sh",
		"scripts/release/full_platform/github-actions-startup-diagnostic.sh",
		"scripts/release/full_platform/target-host-evidence-request.sh",
		"scripts/release/full_platform/target-host-ui-runtime-smoke.sh",
		"scripts/release/full_platform/windows-ui-runtime-smoke.ps1",
		"scripts/release/full_platform/windows-ui-runtime-smoke.sh",
		"scripts/release/full_platform/macos-ui-runtime-smoke.sh",
	} {
		info, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("%s must exist: %v", rel, err)
		}
		if info.IsDir() || info.Size() == 0 {
			t.Fatalf("%s must be a non-empty file", rel)
		}
	}
}

func TestReleaseFullPlatformActionsAvailabilityPreflightIsNotRuntimeEvidence(t *testing.T) {
	scriptPath := filepath.Join(repoRoot(t), "scripts", "release", "full_platform", "actions-availability-preflight.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read GitHub Actions availability preflight: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/full_platform/actions-availability-preflight.sh [--repo OWNER/REPO] [--branch BRANCH] [--report FILE]",
		"tetra.actions.availability.v1",
		"gh run list",
		"gh api \"repos/$repo/actions/runs/$run_id\"",
		"gh api \"repos/$repo/actions/runs/$run_id/jobs\"",
		"gh api \"repos/$repo/actions/runs/$run_id/logs\"",
		"gh api \"repos/$repo/check-suites/$run_check_suite_id\"",
		"production_evidence: false",
		"go run ./tools/cmd/validate-actions-availability --report \"$report_path\"",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Actions availability preflight missing %q", want)
		}
	}

	readmePath := filepath.Join(repoRoot(t), "scripts", "release", "full_platform", "README.md")
	readmeRaw, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read full-platform README: %v", err)
	}
	readme := string(readmeRaw)
	for _, want := range []string{
		"actions-availability-preflight.sh",
		"validate-actions-availability",
		"not runtime evidence",
		"zero jobs",
		"`BuildFailed`",
		"`macos-13`",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("full-platform README missing Actions availability detail %q", want)
		}
	}
}

func TestReleaseFullPlatformGitHubActionsStartupDiagnosticIsBlockedOnly(t *testing.T) {
	scriptPath := filepath.Join(repoRoot(t), "scripts", "release", "full_platform", "github-actions-startup-diagnostic.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read GitHub Actions startup diagnostic: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/full_platform/github-actions-startup-diagnostic.sh [--repo OWNER/REPO] [--branch BRANCH] [--report FILE] [--canary-branch BRANCH]",
		"gh run list",
		"gh api \"repos/$repo/actions/permissions\"",
		"gh api \"repos/$repo/actions/runners\"",
		"gh api \"users/$billing_owner/settings/billing/actions\"",
		"minimal_canary",
		"startup_failure",
		"tetra.actions.startup-blocker.v1",
		"remote_url=\"\"",
		"go run ./tools/cmd/validate-actions-startup-blocker --report \"$report_path\"",
		"manual or self-hosted target-host Windows/macOS reports",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("GitHub Actions startup diagnostic missing %q", want)
		}
	}

	readmePath := filepath.Join(repoRoot(t), "scripts", "release", "full_platform", "README.md")
	readmeRaw, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read full-platform README: %v", err)
	}
	readme := string(readmeRaw)
	for _, want := range []string{
		"github-actions-startup-diagnostic.sh",
		"validate-actions-startup-blocker",
		"diagnostic only",
		"--canary-branch codex/actions-canary",
		"billing_actions_status",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("full-platform README missing startup diagnostic detail %q", want)
		}
	}
}

func TestReleaseFullPlatformTargetHostHelperDocumentsManualEvidence(t *testing.T) {
	scriptPath := filepath.Join(repoRoot(t), "scripts", "release", "full_platform", "target-host-ui-runtime-smoke.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read target-host helper: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/full_platform/target-host-ui-runtime-smoke.sh [--target TARGET] [--report FILE]",
		"windows-x64",
		"macos-x64",
		"go run ./tools/cmd/platform-ui-runtime-smoke",
		"go run ./tools/cmd/validate-windows-ui-runtime --report \"$report_path\"",
		"go run ./tools/cmd/validate-macos-ui-runtime --report \"$report_path\"",
		"requires a real Windows or macOS target host",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("target-host helper missing %q", want)
		}
	}

	readmePath := filepath.Join(repoRoot(t), "scripts", "release", "full_platform", "README.md")
	readmeRaw, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read full-platform README: %v", err)
	}
	readme := string(readmeRaw)
	for _, want := range []string{
		"Manual target-host evidence",
		"target-host-evidence-request.sh",
		"bash scripts/release/full_platform/target-host-ui-runtime-smoke.sh",
		"pwsh -File scripts/release/full_platform/windows-ui-runtime-smoke.ps1",
		"TETRA_WINDOWS_UI_RUNTIME_REPORT=/path/windows-ui-runtime.json",
		"TETRA_MACOS_UI_RUNTIME_REPORT=/path/macos-ui-runtime.json",
		"-ExpectedVersion",
		"-ExpectedGitHead",
		"--expected-version",
		"--expected-git-head",
		"same Git commit",
		"startup_failure",
		"does not relax",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("full-platform README missing manual evidence detail %q", want)
		}
	}
}

func TestReleaseFullPlatformTargetHostEvidenceRequestBundle(t *testing.T) {
	scriptPath := filepath.Join(repoRoot(t), "scripts", "release", "full_platform", "target-host-evidence-request.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read target-host evidence request helper: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/full_platform/target-host-evidence-request.sh [--out-dir DIR] [--repo OWNER/REPO] [--branch BRANCH]",
		"tetra.ui.target-host-evidence-request.v1",
		"production_evidence: false",
		"--expected-version $version --expected-git-head $git_head",
		"windows-ui-runtime-smoke.ps1",
		"target-host-ui-runtime-smoke.sh --target macos-x64",
		"TETRA_WINDOWS_UI_RUNTIME_REPORT",
		"TETRA_MACOS_UI_RUNTIME_REPORT",
		"not runtime evidence",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("target-host evidence request helper missing %q", want)
		}
	}

	outDir := t.TempDir()
	cmd := exec.Command("bash", scriptPath, "--out-dir", outDir, "--repo", "BoSuY0/Tetra_Language", "--branch", "codex/full-platform-ui-runtime")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("target-host evidence request helper failed: %v\n%s", err, output)
	}
	reportPath := filepath.Join(outDir, "target-host-evidence-request.json")
	reportRaw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read target-host evidence request json: %v", err)
	}
	var report struct {
		Schema             string `json:"schema"`
		ProductionEvidence bool   `json:"production_evidence"`
		ExpectedVersion    string `json:"expected_version"`
		ExpectedGitHead    string `json:"expected_git_head"`
		Targets            []struct {
			Target  string `json:"target"`
			Command string `json:"command"`
		} `json:"targets"`
		Aggregation struct {
			Command string `json:"command"`
		} `json:"aggregation"`
	}
	if err := json.Unmarshal(reportRaw, &report); err != nil {
		t.Fatalf("decode target-host evidence request json: %v", err)
	}
	if report.Schema != "tetra.ui.target-host-evidence-request.v1" {
		t.Fatalf("schema = %q", report.Schema)
	}
	if report.ProductionEvidence {
		t.Fatalf("target-host evidence request must not be production evidence")
	}
	if report.ExpectedVersion != "v0.4.0" || strings.TrimSpace(report.ExpectedGitHead) == "" {
		t.Fatalf("request missing expected version/head: %#v", report)
	}
	if len(report.Targets) != 2 {
		t.Fatalf("target count = %d, want 2", len(report.Targets))
	}
	if !strings.Contains(report.Aggregation.Command, "ui-runtime-gate.sh") {
		t.Fatalf("aggregation command missing gate: %q", report.Aggregation.Command)
	}
	for _, target := range report.Targets {
		wants := []string{}
		switch target.Target {
		case "windows-x64":
			wants = []string{
				"-ExpectedVersion v0.4.0",
				"-ExpectedGitHead " + report.ExpectedGitHead,
			}
		case "macos-x64":
			wants = []string{
				"--expected-version v0.4.0",
				"--expected-git-head " + report.ExpectedGitHead,
			}
		default:
			t.Fatalf("unexpected target in request: %q", target.Target)
		}
		for _, want := range wants {
			if !strings.Contains(target.Command, want) {
				t.Fatalf("%s command missing %q: %q", target.Target, want, target.Command)
			}
		}
	}
	readmeRaw, err := os.ReadFile(filepath.Join(outDir, "README.md"))
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	for _, want := range []string{"not runtime evidence", "same Git commit", "--expected-version v0.4.0", "--expected-git-head", "windows-ui-runtime-smoke.ps1", "macos-ui-runtime.json"} {
		if !strings.Contains(string(readmeRaw), want) {
			t.Fatalf("generated README missing %q", want)
		}
	}
}

func TestReleaseFullPlatformWindowsPowerShellTargetHostHelper(t *testing.T) {
	scriptPath := filepath.Join(repoRoot(t), "scripts", "release", "full_platform", "windows-ui-runtime-smoke.ps1")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Windows PowerShell target-host helper: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		"Usage: pwsh -File scripts/release/full_platform/windows-ui-runtime-smoke.ps1 [-Report FILE] [-ReportDir DIR]",
		"Windows_NT",
		"OSArchitecture",
		"IsPathRooted",
		"windows-x64 UI runtime production evidence requires a real Windows x64 host",
		"ExpectedVersion",
		"ExpectedGitHead",
		"go run ./tools/cmd/platform-ui-runtime-smoke --target windows-x64 --report",
		"go run ./tools/cmd/validate-windows-ui-runtime --report $Report --expected-version $ExpectedVersion --expected-git-head $ExpectedGitHead",
		"target-host UI runtime report",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Windows PowerShell target-host helper missing %q", want)
		}
	}
}

func TestReleaseFullPlatformSmokeScriptsAcceptValidatedExternalEvidence(t *testing.T) {
	for _, rel := range []struct {
		path      string
		env       string
		validator string
	}{
		{
			path:      "scripts/release/full_platform/windows-ui-runtime-smoke.sh",
			env:       "TETRA_WINDOWS_UI_RUNTIME_REPORT",
			validator: `go run ./tools/cmd/validate-windows-ui-runtime --report "$report_path" --expected-version "$expected_version" --expected-git-head "$expected_git_head"`,
		},
		{
			path:      "scripts/release/full_platform/macos-ui-runtime-smoke.sh",
			env:       "TETRA_MACOS_UI_RUNTIME_REPORT",
			validator: `go run ./tools/cmd/validate-macos-ui-runtime --report "$report_path" --expected-version "$expected_version" --expected-git-head "$expected_git_head"`,
		},
	} {
		raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(rel.path)))
		if err != nil {
			t.Fatalf("read %s: %v", rel.path, err)
		}
		text := string(raw)
		for _, want := range []string{rel.env, `cp -- "$external_report" "$report_path"`, rel.validator} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing %q", rel.path, want)
			}
		}
	}
}
