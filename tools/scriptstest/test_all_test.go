package scriptstest

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type testAllSummary struct {
	Status      string `json:"status"`
	StepCount   int    `json:"step_count"`
	FailedCount int    `json:"failed_count"`
	Steps       []struct {
		Name     string `json:"name"`
		Status   string `json:"status"`
		ExitCode *int   `json:"exit_code"`
		Command  string `json:"command"`
	} `json:"steps"`
}

const testAllFormatterStepName = "formatter check examples lib runtime"

func TestTestAllQuickJSONIncludesStepExitCodes(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all quick failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "pass" || len(summary.Steps) == 0 {
		t.Fatalf("summary = %#v", summary)
	}
	if summary.StepCount != len(summary.Steps) || summary.FailedCount != 0 {
		t.Fatalf("summary counts = steps:%d failed:%d len:%d", summary.StepCount, summary.FailedCount, len(summary.Steps))
	}
	for _, step := range summary.Steps {
		if step.Status != "pass" || step.ExitCode == nil || *step.ExitCode != 0 {
			t.Fatalf("step missing pass exit code: %#v", step)
		}
		if step.Command == "" {
			t.Fatalf("step missing command: %#v", step)
		}
	}
}

func TestTestAllKeepGoingJSONRecordsFailingExitCode(t *testing.T) {
	root := testAllFakeRepo(t, true)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, []string{"TETRA_FAIL_FMT=1"}, "--quick", "--keep-going", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected failing test_all run\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" {
		t.Fatalf("status = %q, want fail", summary.Status)
	}
	if summary.StepCount != len(summary.Steps) || summary.FailedCount != 1 {
		t.Fatalf("summary counts = steps:%d failed:%d len:%d", summary.StepCount, summary.FailedCount, len(summary.Steps))
	}
	var sawFmt bool
	var sawLaterStep bool
	for _, step := range summary.Steps {
		if step.Name == testAllFormatterStepName {
			sawFmt = true
			if step.Status != "fail" || step.ExitCode == nil || *step.ExitCode != 7 {
				t.Fatalf("formatter step = %#v", step)
			}
		}
		if step.Name == "host smoke linux-x64" && step.Status == "pass" && step.ExitCode != nil && *step.ExitCode == 0 {
			sawLaterStep = true
		}
	}
	if !sawFmt || !sawLaterStep {
		t.Fatalf("summary did not record failing and later steps: %#v", summary.Steps)
	}
}

func TestTestAllFailurePathPreservesSummaryWhenSummaryValidatorFails(t *testing.T) {
	root := testAllFakeRepo(t, true)
	reportDir := filepath.Join(root, "report")
	stdout, stderr, err := runTestAllSplit(t, root, []string{
		"TETRA_FAIL_FMT=1",
		"TETRA_FAIL_SUMMARY_VALIDATOR=1",
	}, "--quick", "--json-only", "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected failing test_all run\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(string(stderr), "warning: summary validation failed; preserving original test failure") {
		t.Fatalf("stderr missing best-effort validator warning:\n%s", stderr)
	}
	summary := decodeTestAllSummary(t, stdout)
	if summary.Status != "fail" {
		t.Fatalf("status = %q, want fail", summary.Status)
	}
	if summary.StepCount != len(summary.Steps) || summary.FailedCount != 1 {
		t.Fatalf("summary counts = steps:%d failed:%d len:%d", summary.StepCount, summary.FailedCount, len(summary.Steps))
	}
	if got := len(summary.Steps); got == 0 {
		t.Fatalf("summary has no steps")
	}
	step := summary.Steps[len(summary.Steps)-1]
	if step.Name != testAllFormatterStepName || step.Status != "fail" || step.ExitCode == nil || *step.ExitCode != 7 {
		t.Fatalf("last failing step = %#v", step)
	}
}

func TestReleaseV06GateValidatesCrossTargetSmokeReports(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, report := range []string{"linux-smoke.json", "macos-smoke.json", "windows-smoke.json"} {
		want := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/` + report + `"`
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing smoke report validation %q", want)
		}
	}
}

func TestReleaseV06GateValidatesHostSmokeReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	run := `./tetra smoke --target linux-x64 --run=true --report "$tmp_dir/host-smoke.json"`
	if !strings.Contains(text, run) {
		t.Fatalf("release gate missing host smoke report command %q", run)
	}
	validate := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/host-smoke.json"`
	if !strings.Contains(text, validate) {
		t.Fatalf("release gate missing host smoke report validation %q", validate)
	}
}

func TestReleaseV05GateValidatesJSONReports(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_5_gate.sh"))
	if err != nil {
		t.Fatalf("read v0.5 release gate: %v", err)
	}
	text := string(raw)
	generatedManifest := `go run ./tools/cmd/validate-manifest --manifest "$tmp_dir/manifest.json"`
	if !strings.Contains(text, generatedManifest) {
		t.Fatalf("v0.5 release gate missing generated manifest validation %q", generatedManifest)
	}
	canonicalManifest := `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
	if !strings.Contains(text, canonicalManifest) {
		t.Fatalf("v0.5 release gate missing canonical manifest validation %q", canonicalManifest)
	}
	testReport := `go run ./tools/cmd/validate-test-report --report "$tmp_dir/tetra-test-report.json"`
	if !strings.Contains(text, testReport) {
		t.Fatalf("v0.5 release gate missing test report validation %q", testReport)
	}
	lspReport := `go run ./tools/cmd/validate-lsp-smoke --report "$tmp_dir/lsp.json"`
	if !strings.Contains(text, lspReport) {
		t.Fatalf("v0.5 release gate missing LSP smoke validation %q", lspReport)
	}
	apiDocs := `go run ./tools/cmd/validate-api-docs --docs "$tmp_dir/api-docs.md"`
	if !strings.Contains(text, apiDocs) {
		t.Fatalf("v0.5 release gate missing API docs validation %q", apiDocs)
	}
	ecoLock := `go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.lock.json"`
	if !strings.Contains(text, ecoLock) {
		t.Fatalf("v0.5 release gate missing Eco lock validation %q", ecoLock)
	}
	ecoVault := `go run ./tools/cmd/validate-eco-vault --store "$tmp_dir/vault"`
	if !strings.Contains(text, ecoVault) {
		t.Fatalf("v0.5 release gate missing Eco vault validation %q", ecoVault)
	}
	hostSmoke := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/host-smoke.json"`
	if !strings.Contains(text, hostSmoke) {
		t.Fatalf("v0.5 release gate missing host smoke validation %q", hostSmoke)
	}
	for _, report := range []string{"linux-smoke.json", "macos-smoke.json", "windows-smoke.json"} {
		want := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/` + report + `"`
		if !strings.Contains(text, want) {
			t.Fatalf("v0.5 release gate missing smoke report validation %q", want)
		}
	}
}

func TestReleaseV06GateValidatesTestRunnerJSONReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-test-report --report "$tmp_dir/tetra-test-report.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing test report validation %q", want)
	}
}

func TestReleaseV06GateValidatesDocsManifests(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`go run ./tools/cmd/validate-manifest --manifest "$tmp_dir/manifest.json"`,
		`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing manifest validation %q", want)
		}
	}
}

func TestReleaseV06GateChecksShortAliasVersion(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`short_version="$(./t version)"`,
		`expected ./t version to match ./tetra version`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing short alias version check %q", want)
		}
	}
}

func TestReleaseV06GateValidatesLSPSmokeJSONReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-lsp-smoke --report "$tmp_dir/lsp-smoke.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing LSP smoke validation %q", want)
	}
}

func TestReleaseV06GateValidatesLSPStdioTranscript(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-lsp-stdio --transcript "$tmp_dir/lsp-stdio.out"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing LSP stdio validation %q", want)
	}
}

func TestReleaseV06GateValidatesGeneratedAPIDocs(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-api-docs --docs "$tmp_dir/api-docs.md"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing API docs validation %q", want)
	}
}

func TestReleaseV06GateFormatterCoversRuntimeSources(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `./tetra fmt --check examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing runtime formatter coverage %q", want)
	}
}

func TestReleaseV06GateScansFlowOnlySources(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing Flow-only scan %q", want)
	}
}

func TestReleaseV06GateValidatesTargetsJSONReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`./tetra targets --format=json >"$tmp_dir/targets.json"`,
		`go run ./tools/cmd/validate-targets --report "$tmp_dir/targets.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing targets validation %q", want)
		}
	}
}

func TestReleaseV06GateValidatesDoctorJSONReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`./tetra doctor --format=json >"$tmp_dir/doctor.json"`,
		`go run ./tools/cmd/validate-doctor --report "$tmp_dir/doctor.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing doctor validation %q", want)
		}
	}
}

func TestReleaseV06GateRunsCheckAndDocCommands(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`./tetra check examples/flow_hello.tetra`,
		`./tetra doc examples >"$tmp_dir/tetra-docs.md"`,
		`go run ./tools/cmd/validate-api-docs --docs "$tmp_dir/tetra-docs.md"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing CLI command coverage %q", want)
		}
	}
}

func TestReleaseV06GateValidatesJSONDiagnosticShape(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`check_json_diagnostic_case "invalid-diagnostic" "unknown function"`,
		`check_json_diagnostic_case "missing-effect-diagnostic" "uses effect 'io'"`,
		`check_json_diagnostic_case "tabs-diagnostic" "tabs are not supported"`,
		`check_json_diagnostic_case "planned-actor-diagnostic" "planned feature 'actor'"`,
		`--require-position`,
		`./tetra build --diagnostics=json --target wasm32-wasi examples/flow_hello.tetra`,
		`go run ./tools/cmd/validate-diagnostic --diagnostic "$tmp_dir/wasm-target-diagnostic.json" --severity error --contains "planned target not implemented: wasm32-wasi"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing JSON diagnostic validation %q", want)
		}
	}
}

func TestReleaseV06GateValidatesSmokeListJSONReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`./tetra smoke --list --format=json >"$tmp_dir/smoke-list.json"`,
		`go run ./tools/cmd/validate-smoke-list --report "$tmp_dir/smoke-list.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing smoke list validation %q", want)
		}
	}
}

func TestReleaseV06GateValidatesEcoLock(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.lock.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing Eco lock validation %q", want)
	}
}

func TestReleaseV06GateValidatesEcoUnpack(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-unpack --dir "$tmp_dir/unpacked"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing Eco unpack validation %q", want)
	}
}

func TestReleaseV06GateValidatesEcoVault(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_6_gate.sh"))
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-vault --store "$tmp_dir/vault"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing Eco vault validation %q", want)
	}
}

func TestTestAllValidatesTestRunnerJSONReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-test-report --report "$report_dir/tetra-test-report.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing test report validation %q", want)
	}
}

func TestTestAllValidatesSummaryArtifacts(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-test-all-summary --summary "$summary_json" --report-dir "$report_dir"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing summary validation %q", want)
	}
}

func TestTestAllChecksShortAliasVersion(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "short alias version" check_short_alias_version`,
		`short_version="$(./t version)"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing short alias version check %q", want)
		}
	}
}

func TestTestAllValidatesHostSmokeReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	run := `./tetra smoke --target linux-x64 --run=true --report "$report_dir/host-smoke.json"`
	if !strings.Contains(text, run) {
		t.Fatalf("test_all missing host smoke report command %q", run)
	}
	validate := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$report_dir/host-smoke.json"`
	if !strings.Contains(text, validate) {
		t.Fatalf("test_all missing host smoke report validation %q", validate)
	}
}

func TestTestAllFormatterCoversRuntimeSources(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `run_step "formatter check examples lib runtime" ./tetra fmt --check examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing runtime formatter coverage %q", want)
	}
}

func TestTestAllScansFlowOnlySources(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `run_step "flow-only source scan" go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing Flow-only scan %q", want)
	}
}

func TestTestAllValidatesTargetsJSONReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "targets json report" check_targets_report`,
		`./tetra targets --format=json >"$report_dir/targets.json"`,
		`go run ./tools/cmd/validate-targets --report "$report_dir/targets.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing targets validation %q", want)
		}
	}
}

func TestTestAllValidatesDoctorJSONReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "doctor json report" check_doctor_report`,
		`./tetra doctor --format=json >"$report_dir/doctor.json"`,
		`go run ./tools/cmd/validate-doctor --report "$report_dir/doctor.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing doctor validation %q", want)
		}
	}
}

func TestTestAllRunsCheckAndDocCommands(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "tetra check flow hello" ./tetra check examples/flow_hello.tetra`,
		`run_step "tetra doc examples" check_tetra_doc`,
		`./tetra doc examples >"$report_dir/tetra-docs.md"`,
		`go run ./tools/cmd/validate-api-docs --docs "$report_dir/tetra-docs.md"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing CLI command coverage %q", want)
		}
	}
}

func TestTestAllValidatesJSONDiagnosticShape(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "json diagnostic shape" check_json_diagnostic`,
		`check_json_diagnostic_case "invalid-diagnostic" "unknown function"`,
		`check_json_diagnostic_case "missing-effect-diagnostic" "uses effect 'io'"`,
		`check_json_diagnostic_case "tabs-diagnostic" "tabs are not supported"`,
		`check_json_diagnostic_case "planned-actor-diagnostic" "planned feature 'actor'"`,
		`--require-position`,
		`./tetra build --diagnostics=json --target wasm32-wasi examples/flow_hello.tetra`,
		`go run ./tools/cmd/validate-diagnostic --diagnostic "$report_dir/wasm-target-diagnostic.json" --severity error --contains "planned target not implemented: wasm32-wasi"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing JSON diagnostic validation %q", want)
		}
	}
}

func TestTestAllValidatesSmokeListJSONReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "smoke list json report" check_smoke_list`,
		`./tetra smoke --list --format=json >"$report_dir/smoke-list.json"`,
		`go run ./tools/cmd/validate-smoke-list --report "$report_dir/smoke-list.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing smoke list validation %q", want)
		}
	}
}

func TestTestAllValidatesDocsManifests(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`go run ./tools/cmd/validate-manifest --manifest "$tmp_dir/manifest.json"`,
		`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing manifest validation %q", want)
		}
	}
}

func TestTestAllValidatesLSPSmokeJSONReport(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `./tetra lsp --stdio-smoke examples/flow_hello.tetra >"$report_dir/lsp-smoke.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing persisted LSP smoke report %q", want)
	}
	validator := `go run ./tools/cmd/validate-lsp-smoke --report "$report_dir/lsp-smoke.json"`
	if !strings.Contains(string(raw), validator) {
		t.Fatalf("test_all missing LSP smoke validation %q", validator)
	}
}

func TestTestAllValidatesLSPStdioTranscript(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-lsp-stdio --transcript "$tmp_dir/lsp-stdio.out"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing LSP stdio validation %q", want)
	}
}

func TestTestAllValidatesGeneratedAPIDocs(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	gen := `go run ./tools/cmd/gen-docs examples >"$report_dir/api-docs.md"`
	if !strings.Contains(string(raw), gen) {
		t.Fatalf("test_all missing persisted generated API docs %q", gen)
	}
	validate := `go run ./tools/cmd/validate-api-docs --docs "$report_dir/api-docs.md"`
	if !strings.Contains(string(raw), validate) {
		t.Fatalf("test_all missing API docs validation %q", validate)
	}
}

func TestReleaseV10GateFormatterCoversRuntimeSources(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v1_0_gate.sh"))
	if err != nil {
		t.Fatalf("read v1.0 release gate: %v", err)
	}
	want := `./tetra fmt --check examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("v1.0 release gate missing runtime formatter coverage %q", want)
	}
}

func TestTestAllValidatesEcoLock(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.lock.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing Eco lock validation %q", want)
	}
}

func TestTestAllValidatesEcoUnpack(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-unpack --dir "$tmp_dir/unpacked"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing Eco unpack validation %q", want)
	}
}

func TestTestAllValidatesEcoVault(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-vault --store "$tmp_dir/vault"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing Eco vault validation %q", want)
	}
}

func TestTestAllFullValidatesCrossTargetSmokeReports(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	goLog := filepath.Join(root, "go.log")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_GO_LOG=" + goLog}, "--full", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all full failed: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(goLog)
	if err != nil {
		t.Fatalf("read fake go log: %v", err)
	}
	log := string(raw)
	for _, report := range []string{"linux-smoke.json", "macos-smoke.json", "windows-smoke.json"} {
		want := "run ./tools/cmd/smoke-report-to-checklist --validate-only --report "
		if !strings.Contains(log, want) || !strings.Contains(log, report) {
			t.Fatalf("full gate missing validate-only call for %s in log:\n%s", report, log)
		}
	}
}

func testAllFakeRepo(t *testing.T, failFmt bool) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"), filepath.Join(root, "scripts", "test_all.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "bootstrap.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\ncp ./tetra ./t\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "test.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "generated"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "generated", "manifest.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goScript := `#!/usr/bin/env bash
set -euo pipefail
if [[ -n "${TETRA_FAKE_GO_LOG:-}" ]]; then
  printf '%s\n' "$*" >>"$TETRA_FAKE_GO_LOG"
fi
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/validate-test-all-summary" && "${TETRA_FAIL_SUMMARY_VALIDATOR:-}" == "1" ]]; then
  echo "summary validator unavailable" >&2
  exit 23
fi
if [[ "${1:-}" == "run" && "${2:-}" == "./tools/cmd/gen-manifest" ]]; then
  out=""
  shift 2
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -o)
        out="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ -n "$out" ]]; then
    printf '{}\n' >"$out"
  fi
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte(goScript), 0o755); err != nil {
		t.Fatal(err)
	}
	tetra := `#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-}"
shift || true
case "$cmd" in
  version)
    echo "v0.6.0"
    ;;
  fmt)
    if [[ "${TETRA_FAIL_FMT:-}" == "1" ]]; then
      echo "format mismatch" >&2
      exit 7
    fi
    ;;
  test)
    for arg in "$@"; do
      if [[ "$arg" == "--report=json" ]]; then
        echo '{"total":0,"passed":0,"failed":0,"files":[],"results":[]}'
        exit 0
      fi
    done
    ;;
  check)
    for arg in "$@"; do
      if [[ "$arg" == "--diagnostics=json" ]]; then
        case "$*" in
          *missing-effect-diagnostic.tetra*) echo '{"code":"TETRA2001","message":"function main uses effect '\''io'\'' but does not declare it","severity":"error"}' >&2 ;;
          *tabs-diagnostic.tetra*) echo '{"code":"TETRA0001","message":"tabs are not supported in Flow indentation","severity":"error"}' >&2 ;;
          *planned-actor-diagnostic.tetra*) echo '{"code":"TETRA0001","message":"planned feature '\''actor'\'' is not implemented","severity":"error"}' >&2 ;;
          *) echo '{"code":"TETRA2001","message":"unknown function missing_call","severity":"error"}' >&2 ;;
        esac
        exit 1
      fi
    done
    ;;
  doc)
    printf '%s\n' '# Tetra API Docs' '' '## examples' '' '### Functions' ''
    printf '%b\n' '- \x60func main() -> Int\x60'
    ;;
  build)
    for arg in "$@"; do
      if [[ "$arg" == "wasm32-wasi" ]]; then
        echo '{"code":"TETRA0001","message":"planned target not implemented: wasm32-wasi","severity":"error"}' >&2
        exit 2
      fi
    done
    ;;
  targets)
    printf '{"supported":["linux-x64","windows-x64","macos-x64"],"planned":["wasm32-wasi","wasm32-web"]}\n'
    ;;
  doctor)
    printf '{"status":"pass","checks":[{"name":"version","status":"pass"},{"name":"supported targets","status":"pass"},{"name":"planned targets","status":"pass"},{"name":"repo root","status":"pass"},{"name":"__rt/actors_sysv.tetra","status":"pass"},{"name":"__rt/actors_win64.tetra","status":"pass"},{"name":"compiler/selfhostrt/actors_sysv.tetra","status":"pass"},{"name":"compiler/selfhostrt/actors_win64.tetra","status":"pass"},{"name":"examples/flow_hello.tetra","status":"pass"},{"name":"docs/generated/manifest.json","status":"pass"},{"name":"docs manifest version","status":"pass"},{"name":"docs manifest surface","status":"pass"},{"name":"smoke sources","status":"pass"},{"name":"runtime exports","status":"pass"}]}\n'
    ;;
  smoke)
    report=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --report)
          report="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    if [[ -n "$report" ]]; then
      printf '{"target":"linux-x64","cases":[]}\n' >"$report"
    fi
    echo "Smoke linux-x64: 0/0 passed"
    ;;
  lsp)
    if [[ "${1:-}" == "--stdio" ]]; then
      cat >/dev/null
      printf '{"result":{"capabilities":{}}}\n'
      printf '{"method":"textDocument/publishDiagnostics","params":{"diagnostics":[]}}\n'
    else
      printf '{"diagnostics":[]}\n'
    fi
    ;;
  eco)
    sub="${1:-}"
    shift || true
    if [[ "$sub" == "unpack" ]]; then
      out=""
      while [[ $# -gt 0 ]]; do
        case "$1" in
          -C)
            out="$2"
            shift 2
            ;;
          *)
            shift
            ;;
        esac
      done
      if [[ -n "$out" ]]; then
        mkdir -p "$out/src"
        printf 'func main() -> Int:\n    return 0\n' >"$out/src/main.tetra"
      fi
    fi
    ;;
  *)
    ;;
esac
`
	if failFmt && len(tetra) == 0 {
		t.Fatal("unreachable")
	}
	if err := os.WriteFile(filepath.Join(root, "tetra"), []byte(tetra), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func runTestAll(t *testing.T, root string, env []string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{"scripts/test_all.sh"}, args...)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	cmd.Env = append(cmd.Env, "PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	return cmd.CombinedOutput()
}

func runTestAllSplit(t *testing.T, root string, env []string, args ...string) ([]byte, []byte, error) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{"scripts/test_all.sh"}, args...)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	cmd.Env = append(cmd.Env, "PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func decodeTestAllSummary(t *testing.T, raw []byte) testAllSummary {
	t.Helper()
	var summary testAllSummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatalf("decode summary: %v\n%s", err, string(raw))
	}
	return summary
}

func copyFile(src, dst string, mode os.FileMode) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, raw, mode)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
