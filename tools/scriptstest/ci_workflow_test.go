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
		"govulncheck ./compiler/... ./cli/... ./tools/...",
		"fuzz-short-linux:",
		"bash scripts/dev/fuzz-nightly.sh --short --out-dir \"$RUNNER_TEMP/fuzz-short\"",
		"fuzz-nightly-linux:",
		"bash scripts/dev/fuzz-nightly.sh --fuzztime 10m --out-dir \"$RUNNER_TEMP/fuzz-nightly\"",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ci workflow missing %q", want)
		}
	}
	for _, blocked := range []string{
		"  push:",
		"  pull_request:",
		"  schedule:",
	} {
		if strings.Contains(text, blocked) {
			t.Fatalf("ci workflow must not auto-trigger while GitHub Actions billing is locked; found %q", blocked)
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
		"release-v0-4-0-readiness-linux:",
		"surface-release-readiness-linux:",
		"techempower-report-schemas-linux:",
		"lint-workflows-and-shell-linux:",
	} {
		if !workflowJobHasField(text, job, "timeout-minutes:") {
			t.Fatalf("ci workflow job %s missing job-level timeout-minutes", job)
		}
	}
}

func TestCIWorkflowIncludesSurfaceReleaseReadinessJob(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"surface-release-readiness-linux:",
		"github.event_name == 'workflow_dispatch' || github.event_name == 'schedule'",
		"runs-on: ubuntu-latest",
		"timeout-minutes: 90",
		"actions/checkout@v4",
		"actions/setup-go@v5",
		"go-version: \"1.20.x\"",
		"name: Bootstrap",
		"bash scripts/dev/bootstrap.sh",
		"name: Surface release gate",
		"bash scripts/release/surface/release-gate.sh --report-dir reports/surface-release-v1",
		"name: Surface experimental regression gate",
		"bash scripts/release/surface/gate.sh --report-dir reports/surface-experimental-regression",
		"name: Safe view lifetime gate",
		"bash scripts/release/safe-view-lifetime/gate.sh --report-dir reports/safe-view-lifetime",
		"name: Surface API stability gate",
		"bash scripts/release/surface/api-stability-gate.sh --report-dir reports/surface-api-stability-v1",
		"name: Full tests",
		"go test ./compiler/... ./cli/... ./tools/... -count=1",
		"go test ./... ./compiler/... ./cli/... ./tools/... -count=1",
		"bash scripts/ci/test.sh",
		"name: Docs and manifest",
		"go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json",
		"go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
		"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
		"git diff --exit-code -- docs/generated/manifest.json",
		"name: Upload release reports",
		"if: always()",
		"uses: actions/upload-artifact@v4",
		"name: tetra-surface-release-v1-${{ github.sha }}",
		"path: |",
		"reports/surface-release-v1",
		"reports/surface-experimental-regression",
		"reports/safe-view-lifetime",
		"reports/surface-api-stability-v1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ci workflow missing Surface release readiness detail %q", want)
		}
	}
	if strings.Contains(text, "continue-on-error: true") {
		t.Fatalf("Surface release readiness job must not silently continue after missing production dependencies")
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
		"govulncheck ./compiler/... ./cli/... ./tools/...",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ci workflow missing supply-chain vulnerability scan detail %q", want)
		}
	}
}

func TestCIWorkflowIncludesCurrentV040ReleaseReadinessJob(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"release-v0-4-0-readiness-linux:",
		"github.event_name == 'schedule' || github.event_name == 'workflow_dispatch'",
		"needs: test",
		"mkdir -p reports/ci-release-v0.4.0-readiness",
		"./tetra features --format=json > reports/ci-release-v0.4.0-readiness/features.json",
		"./tetra targets --format=json > reports/ci-release-v0.4.0-readiness/targets.json",
		"go run ./tools/cmd/validate-v0-4-readiness \\",
		"--expected-version v0.4.0 \\",
		"--features reports/ci-release-v0.4.0-readiness/features.json \\",
		"--targets reports/ci-release-v0.4.0-readiness/targets.json \\",
		"--manifest docs/generated/manifest.json \\",
		"--scope-decisions docs/release/v0_4_0_scope_decisions.json",
		"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
		"go run ./tools/cmd/validate-v0-4-completion-audit --audit docs/release/v0_4_0_completion_audit.md --expected-status achieved",
		"name: tetra-v0.4.0-${{ github.sha }}-release-readiness-linux",
		"path: reports/ci-release-v0.4.0-readiness",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ci workflow missing current v0.4.0 release readiness detail %q", want)
		}
	}
	if strings.Contains(text, "v0.3.0") || strings.Contains(text, "v0_3_0") {
		t.Fatalf("ci workflow contains stale v0.3 release assumptions")
	}
}

func TestCIWorkflowValidatesTechEmpowerReportSchemas(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"techempower-report-schemas-linux:",
		"github.event_name != 'schedule'",
		"timeout-minutes: 20",
		"name: Validate TechEmpower report schemas",
		"go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_local_smoke_skip_db_report.json --allow-skip-db",
		"go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_local_report.json",
		"go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_matrix_local_report.json",
		"go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ci workflow missing TechEmpower report schema detail %q", want)
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
		"tetra-v0.4.0-${{ github.sha }}-smoke-${{ runner.os }}",
		"tetra-v0.4.0-${{ github.sha }}-test-all-quick-linux",
		"tetra-v0.4.0-${{ github.sha }}-test-all-stabilization-linux",
		"tetra-v0.4.0-${{ github.sha }}-release-readiness-linux",
		"tetra-v0.4.0-${{ github.sha }}-coverage-linux",
		"tetra-v0.4.0-${{ github.sha }}-fuzz-short-linux",
		"tetra-v0.4.0-${{ github.sha }}-fuzz-nightly-linux",
		"tetra-surface-release-v1-${{ github.sha }}",
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
		wantParts := []string{"v0.4.0", "${{ github.sha }}"}
		if name == "tetra-surface-release-v1-${{ github.sha }}" {
			wantParts = []string{"surface-release-v1", "${{ github.sha }}"}
		}
		for _, wantPart := range wantParts {
			if !strings.Contains(name, wantPart) {
				t.Fatalf("ci workflow artifact name %q missing release-aware metadata %q", name, wantPart)
			}
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
