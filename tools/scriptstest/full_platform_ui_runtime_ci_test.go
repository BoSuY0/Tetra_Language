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
		"go run ./tools/cmd/validate-windows-ui-runtime --report windows-ui-runtime.json",
		"go run ./tools/cmd/validate-macos-ui-runtime --report macos-ui-runtime.json",
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

func TestFullPlatformUIRuntimeWorkflowDocumentsCurrentActionlintRunnerLabel(t *testing.T) {
	path := filepath.Join(repoRoot(t), ".github", "actionlint.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read actionlint config: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		".github/workflows/full-platform-ui-runtime.yml:",
		`label "macos-15-intel" is unknown`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("actionlint config missing %q", want)
		}
	}
}
