package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCIWorkflowIncludesStabilizationAndRobustnessJobs(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"workflow_dispatch:",
		"test-all-quick-linux:",
		"github.event_name != 'schedule'",
		"bash scripts/ci/test-all.sh --quick --keep-going --report-dir reports/ci-test-all-quick",
		"stabilization-linux:",
		"github.event_name == 'schedule' || github.event_name == 'workflow_dispatch'",
		"bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir reports/ci-test-all-stabilization",
		"test-all-stabilization-linux",
		"coverage-linux:",
		"go test ./compiler/... ./cli/... ./tools/... -covermode=atomic -coverprofile=coverage.out -count=1",
		"if: always()",
		"race-linux:",
		"go test -race ./compiler/... ./cli/... ./tools/... -count=1",
		"supply-chain-vulnerability-scan-linux:",
		"go install golang.org/x/vuln/cmd/govulncheck@v1.1.2",
		"govulncheck ./...",
		"fuzz-short-linux:",
		"bash scripts/dev/fuzz-nightly.sh --short --out-dir \"$RUNNER_TEMP/fuzz-short\"",
		"fuzz-nightly-linux:",
		"bash scripts/dev/fuzz-nightly.sh --fuzztime 10m --out-dir \"$RUNNER_TEMP/fuzz-nightly\"",
		"schedule:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ci workflow missing %q", want)
		}
	}
}

func TestCIWorkflowHasLeastPrivilegeConcurrencyAndTimeouts(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"permissions:\n  contents: read",
		"concurrency:\n  group: ${{ github.workflow }}-${{ github.ref }}",
		"cancel-in-progress: true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ci workflow missing %q", want)
		}
	}
	for _, job := range []string{
		"test:",
		"test-all-quick-linux:",
		"stabilization-linux:",
		"coverage-linux:",
		"race-linux:",
		"supply-chain-vulnerability-scan-linux:",
		"fuzz-short-linux:",
		"fuzz-nightly-linux:",
		"release-v0-3-0-gate-linux:",
		"lint-workflows-and-shell-linux:",
	} {
		if !workflowJobHasField(text, job, "timeout-minutes:") {
			t.Fatalf("ci workflow job %s missing job-level timeout-minutes", job)
		}
	}
}

func TestCIWorkflowIncludesSupplyChainVulnerabilityScan(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"supply-chain-vulnerability-scan-linux:",
		"github.event_name != 'schedule'",
		"runs-on: ubuntu-latest",
		"timeout-minutes: 20",
		"actions/checkout@v4",
		"actions/setup-go@v5",
		"go-version: \"1.20.x\"",
		"name: Install govulncheck",
		"go install golang.org/x/vuln/cmd/govulncheck@v1.1.2",
		"name: govulncheck",
		"shell: bash",
		"govulncheck ./...",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ci workflow missing supply-chain vulnerability scan detail %q", want)
		}
	}
}

func TestCIWorkflowIncludesCanonicalV030ReleaseGateJob(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"release-v0-3-0-gate-linux:",
		"github.event_name == 'schedule' || github.event_name == 'workflow_dispatch'",
		"needs: test",
		"TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF: \"1\"",
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT: reports/ci-smoke-runtime/tetra-v0.3.0-${{ github.sha }}-smoke-macOS/smoke_macos-x64.json",
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT: reports/ci-smoke-runtime/tetra-v0.3.0-${{ github.sha }}-smoke-Windows/smoke_windows-x64.json",
		"uses: actions/download-artifact@v4",
		"pattern: tetra-v0.3.0-${{ github.sha }}-smoke-*",
		"bash scripts/release/v0_3_0/gate.sh --report-dir reports/ci-release-v0.3.0-gate",
		"name: tetra-v0.3.0-${{ github.sha }}-release-gate-linux",
		"path: reports/ci-release-v0.3.0-gate",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ci workflow missing canonical v0.3.0 release gate detail %q", want)
		}
	}
}

func TestCIWorkflowArtifactNamesAreReleaseAware(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	names := workflowUploadArtifactNames(string(raw))
	want := []string{
		"tetra-v0.3.0-${{ github.sha }}-smoke-${{ runner.os }}",
		"tetra-v0.3.0-${{ github.sha }}-test-all-quick-linux",
		"tetra-v0.3.0-${{ github.sha }}-test-all-stabilization-linux",
		"tetra-v0.3.0-${{ github.sha }}-release-gate-linux",
		"tetra-full-platform-ui-runtime-${{ github.sha }}-${{ matrix.target }}",
		"tetra-full-platform-ui-runtime-${{ github.sha }}-gate",
		"tetra-v0.3.0-${{ github.sha }}-coverage-linux",
		"tetra-v0.3.0-${{ github.sha }}-fuzz-short-linux",
		"tetra-v0.3.0-${{ github.sha }}-fuzz-nightly-linux",
	}
	if len(names) != len(want) {
		t.Fatalf("ci workflow upload-artifact names = %v, want %v", names, want)
	}
	for _, wantName := range want {
		if !stringSliceContains(names, wantName) {
			t.Fatalf("ci workflow upload-artifact names = %v, missing %q", names, wantName)
		}
	}
	for _, name := range names {
		if !strings.Contains(name, "${{ github.sha }}") {
			t.Fatalf("ci workflow artifact name %q missing git SHA metadata", name)
		}
		if !strings.Contains(name, "v0.3.0") && !strings.Contains(name, "full-platform-ui-runtime") {
			t.Fatalf("ci workflow artifact name %q missing release-aware scope", name)
		}
	}
}

func TestCIWorkflowIncludesMinimalActionAndShellLinting(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"lint-workflows-and-shell-linux:",
		"bash -n",
		"go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ci workflow missing linting detail %q", want)
		}
	}
}

func TestCIWorkflowValidatesSmokeJSONReportsBeforeUpload(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(raw)
	for _, report := range []string{
		`smoke_${target}.json`,
		`smoke_${target}_islands_debug.json`,
	} {
		run := `--report "` + report + `"`
		validate := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "` + report + `"`
		runIndex := strings.Index(text, run)
		if runIndex < 0 {
			t.Fatalf("ci workflow missing smoke report generation %q", run)
		}
		validateIndex := strings.Index(text, validate)
		if validateIndex < 0 {
			t.Fatalf("ci workflow missing smoke report validation %q", validate)
		}
		if validateIndex < runIndex {
			t.Fatalf("ci workflow validates %q before it is generated", report)
		}
	}
	uploadIndex := strings.Index(text, "uses: actions/upload-artifact@v4")
	if uploadIndex < 0 {
		t.Fatalf("ci workflow missing smoke report upload")
	}
	validateIndex := strings.LastIndex(text, "go run ./tools/cmd/smoke-report-to-checklist --validate-only --report")
	if validateIndex < 0 || uploadIndex < validateIndex {
		t.Fatalf("ci workflow must validate smoke JSON reports before upload")
	}
}

func workflowJobHasField(workflow, job, field string) bool {
	inJob := false
	for _, line := range strings.Split(workflow, "\n") {
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
			inJob = line == "  "+job
			continue
		}
		if inJob && strings.Contains(line, "    "+field) {
			return true
		}
	}
	return false
}

func workflowUploadArtifactNames(workflow string) []string {
	lines := strings.Split(workflow, "\n")
	var names []string
	for i, line := range lines {
		if !strings.Contains(line, "uses: actions/upload-artifact@v4") {
			continue
		}
		for _, candidate := range lines[i+1:] {
			if strings.HasPrefix(candidate, "      - ") {
				break
			}
			trimmed := strings.TrimSpace(candidate)
			if strings.HasPrefix(trimmed, "name: ") {
				names = append(names, strings.TrimPrefix(trimmed, "name: "))
				break
			}
		}
	}
	return names
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
