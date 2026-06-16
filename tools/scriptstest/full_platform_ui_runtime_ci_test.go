package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFullPlatformUIRuntimeWorkflowProducesTargetHostReports(t *testing.T) {
	path := filepath.Join(repoRoot(t), ".github", "workflows", "full-platform-ui-runtime.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read full-platform UI runtime workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"name: full-platform-ui-runtime",
		"workflow_dispatch:",
		"contents: read",
		"target-host-ui-runtime:",
		"fail-fast: false",
		"runs-on: ${{ matrix.os }}",
		`case "${{ runner.os }}" in`,
		`Windows) tetra_bin="$RUNNER_TEMP/tetra.exe" ;;`,
		`*) tetra_bin="$RUNNER_TEMP/tetra" ;;`,
		`go build -o "$tetra_bin" ./cli/cmd/tetra`,
		`"$tetra_bin" version`,
		"os: windows-2025",
		"target: windows-x64",
		"report: windows-ui-runtime.json",
		"os: macos-15-intel",
		"target: macos-x64",
		"report: macos-ui-runtime.json",
		"go run ./tools/cmd/platform-ui-runtime-smoke --target \"${{ matrix.target }}\" --report \"${{ matrix.report }}\"",
		"expected_version=\"$(go run ./cli/cmd/tetra version)\"",
		"expected_git_head=\"$(git rev-parse HEAD)\"",
		"go run ./tools/cmd/validate-windows-ui-runtime --report windows-ui-runtime.json --expected-version \"$expected_version\" --expected-git-head \"$expected_git_head\"",
		"go run ./tools/cmd/validate-macos-ui-runtime --report macos-ui-runtime.json --expected-version \"$expected_version\" --expected-git-head \"$expected_git_head\"",
		"uses: actions/upload-artifact@v4",
		"name: tetra-full-platform-ui-runtime-${{ github.sha }}-${{ matrix.target }}",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("full-platform UI runtime workflow missing %q", want)
		}
	}
}

func TestFullPlatformUIRuntimeWorkflowRunsOnBranchPush(t *testing.T) {
	path := filepath.Join(repoRoot(t), ".github", "workflows", "full-platform-ui-runtime.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read full-platform UI runtime workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"on:",
		"push:",
		"pull_request:",
		"workflow_dispatch:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("full-platform UI runtime workflow missing trigger %q", want)
		}
	}
}

func TestFullPlatformUIRuntimeWorkflowFetchesBranchParentAncestryForDocVerification(t *testing.T) {
	path := filepath.Join(repoRoot(t), ".github", "workflows", "full-platform-ui-runtime.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read full-platform UI runtime workflow: %v", err)
	}
	text := string(raw)
	if got := strings.Count(text, "fetch-depth: 3"); got < 2 {
		t.Fatalf("full-platform UI runtime workflow must fetch PR branch-parent ancestry for HEAD^2^ docs verification; got %d fetch-depth: 3 markers", got)
	}
}

func TestFullPlatformUIRuntimeWorkflowAggregatesTargetHostReports(t *testing.T) {
	path := filepath.Join(repoRoot(t), ".github", "workflows", "full-platform-ui-runtime.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read full-platform UI runtime workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"full-platform-ui-runtime-gate-linux:",
		"needs: target-host-ui-runtime",
		"runs-on: ubuntu-24.04",
		"TETRA_WINDOWS_UI_RUNTIME_REPORT: reports/full-platform-ui-runtime-targets/tetra-full-platform-ui-runtime-${{ github.sha }}-windows-x64/windows-ui-runtime.json",
		"TETRA_MACOS_UI_RUNTIME_REPORT: reports/full-platform-ui-runtime-targets/tetra-full-platform-ui-runtime-${{ github.sha }}-macos-x64/macos-ui-runtime.json",
		"uses: actions/download-artifact@v4",
		"Build CLI for aggregation gate",
		"go build -o ./tetra ./cli/cmd/tetra",
		"pattern: tetra-full-platform-ui-runtime-${{ github.sha }}-*",
		"path: reports/full-platform-ui-runtime-targets",
		"bash scripts/release/full_platform/ui-runtime-gate.sh --report-dir reports/full-platform-ui-runtime",
		"name: tetra-full-platform-ui-runtime-${{ github.sha }}-gate",
		"path: reports/full-platform-ui-runtime",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("full-platform UI runtime workflow missing aggregation detail %q", want)
		}
	}
}

func TestFullPlatformUIRuntimeWorkflowAllowsCurrentGitHubMacOSIntelLabel(t *testing.T) {
	path := filepath.Join(repoRoot(t), ".github", "actionlint.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read actionlint config: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "macos-15-intel") {
		t.Fatalf("actionlint config must allow GitHub's current Intel macOS runner label")
	}
}

func TestMainCIWorkflowRunsFullPlatformUIRuntimeFanIn(t *testing.T) {
	path := filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"full-platform-ui-runtime-target-host:",
		"full-platform-ui-runtime-gate-linux:",
		"needs: full-platform-ui-runtime-target-host",
		"os: windows-2025",
		"target: windows-x64",
		"os: macos-15-intel",
		"target: macos-x64",
		"go run ./tools/cmd/platform-ui-runtime-smoke --target \"${{ matrix.target }}\" --report \"${{ matrix.report }}\"",
		"expected_version=\"$(go run ./cli/cmd/tetra version)\"",
		"expected_git_head=\"$(git rev-parse HEAD)\"",
		"go run ./tools/cmd/validate-windows-ui-runtime --report windows-ui-runtime.json --expected-version \"$expected_version\" --expected-git-head \"$expected_git_head\"",
		"go run ./tools/cmd/validate-macos-ui-runtime --report macos-ui-runtime.json --expected-version \"$expected_version\" --expected-git-head \"$expected_git_head\"",
		"TETRA_WINDOWS_UI_RUNTIME_REPORT: reports/full-platform-ui-runtime-targets/tetra-full-platform-ui-runtime-${{ github.sha }}-windows-x64/windows-ui-runtime.json",
		"TETRA_MACOS_UI_RUNTIME_REPORT: reports/full-platform-ui-runtime-targets/tetra-full-platform-ui-runtime-${{ github.sha }}-macos-x64/macos-ui-runtime.json",
		"Build CLI for aggregation gate",
		"go build -o ./tetra ./cli/cmd/tetra",
		"bash scripts/release/full_platform/ui-runtime-gate.sh --report-dir reports/full-platform-ui-runtime",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("CI workflow missing full-platform UI runtime detail %q", want)
		}
	}
}
