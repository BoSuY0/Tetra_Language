package scriptstest

import (
	"os"
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
		"scripts/release/full_platform/github-actions-startup-diagnostic.sh",
		"scripts/release/full_platform/target-host-ui-runtime-smoke.sh",
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
		"bash scripts/release/full_platform/target-host-ui-runtime-smoke.sh",
		"TETRA_WINDOWS_UI_RUNTIME_REPORT=/path/windows-ui-runtime.json",
		"TETRA_MACOS_UI_RUNTIME_REPORT=/path/macos-ui-runtime.json",
		"same Git commit",
		"startup_failure",
		"does not relax",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("full-platform README missing manual evidence detail %q", want)
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
			validator: `go run ./tools/cmd/validate-windows-ui-runtime --report "$report_path"`,
		},
		{
			path:      "scripts/release/full_platform/macos-ui-runtime-smoke.sh",
			env:       "TETRA_MACOS_UI_RUNTIME_REPORT",
			validator: `go run ./tools/cmd/validate-macos-ui-runtime --report "$report_path"`,
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
