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
	Status          string `json:"status"`
	StepCount       int    `json:"step_count"`
	FailedCount     int    `json:"failed_count"`
	ReleaseArtifact string `json:"release_artifact"`
	Steps           []struct {
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
	if summary.ReleaseArtifact != "tetra.release.v0_2_0.test-all-summary.v1" {
		t.Fatalf("release_artifact = %q", summary.ReleaseArtifact)
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

func TestTestAllRejectsMalformedReportDirFlag(t *testing.T) {
	root := testAllFakeRepo(t, false)
	out, err := runTestAll(t, root, nil, "--report-dir")
	if err == nil {
		t.Fatalf("expected malformed --report-dir failure\n%s", out)
	}
	if !strings.Contains(string(out), "--report-dir requires a directory") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestTestAllRunsFromNestedWorkingDirectory(t *testing.T) {
	root := testAllFakeRepo(t, false)
	nested := filepath.Join(root, "sub", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := runTestAllFromWorkingDir(t, root, nested, nil, "--quick", "--json-only", "--report-dir", filepath.Join(root, "report"))
	if err != nil {
		t.Fatalf("test_all run from nested dir failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "pass" {
		t.Fatalf("summary status = %q", summary.Status)
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
		`./tetra build --target wasm32-wasi -o "$wasm_out" examples/hello.tetra`,
		`test "$(od -An -tx1 -N4 "$wasm_out" | tr -d ' \n')" = "0061736d"`,
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
	run := `run_release_smoke_target "linux-x64" "true" "$report_dir/host-smoke.json"`
	if !strings.Contains(text, run) {
		t.Fatalf("test_all missing host smoke report command %q", run)
	}
	validate := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$report_dir/host-smoke.json"`
	if !strings.Contains(text, validate) {
		t.Fatalf("test_all missing host smoke report validation %q", validate)
	}
	if strings.Contains(text, `./tetra smoke --target linux-x64 --run=true`) {
		t.Fatalf("test_all should not use legacy full smoke coverage for host run")
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
		`check_json_diagnostic_case "planned-actor-diagnostic" "actor declarations currently support state fields and func methods only"`,
		`--require-position`,
		`./tetra build --target wasm32-wasi -o "$wasm_out" examples/hello.tetra`,
		`test "$(od -An -tx1 -N4 "$wasm_out" | tr -d ' \n')" = "0061736d"`,
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
		`write_release_smoke_list "$report_dir/smoke-list.json"`,
		`go run ./tools/cmd/validate-flow-only examples/flow_hello.tetra examples/flow_struct_smoke.tetra examples/flow_islands_smoke.tetra examples/flow_unsafe_cap_mem_smoke.tetra`,
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

func TestTestAllFullValidatesPerformanceReportWhenPresent(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "test_all.sh"))
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`check_performance_report()`,
		`go run ./tools/cmd/validate-performance-report --report "$report"`,
		`run_step "performance report schema" check_performance_report`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing performance report wiring %q", want)
		}
	}
}

func TestTestAllFullRunsDocsManifestDiffStep(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, nil, "--full", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all full failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	for _, step := range summary.Steps {
		if step.Name == "docs manifest diff" && step.Status == "pass" {
			return
		}
	}
	t.Fatalf("full test_all summary missing passing docs manifest diff step: %#v", summary.Steps)
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

func TestReleaseV012GateFormatterCoversRuntimeSources(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v0_1_2_gate.sh"))
	if err != nil {
		t.Fatalf("read v0.1.2 release gate: %v", err)
	}
	want := `./tetra fmt --check examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("v0.1.2 release gate missing runtime formatter coverage %q", want)
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
	for _, report := range []string{"linux-smoke.json", "macos-smoke.json", "windows-smoke.json", "wasm32-wasi-smoke.json", "wasm32-web-smoke.json"} {
		want := "run ./tools/cmd/smoke-report-to-checklist --validate-only --report "
		if !strings.Contains(log, want) || !strings.Contains(log, report) {
			t.Fatalf("full gate missing validate-only call for %s in log:\n%s", report, log)
		}
	}
}

func TestTestAllStabilizationRunsFocusedGates(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	goLog := filepath.Join(root, "go.log")
	out, err := runTestAll(t, root, []string{"TETRA_FAKE_GO_LOG=" + goLog}, "--stabilization", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all stabilization failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "pass" {
		t.Fatalf("summary status = %q", summary.Status)
	}
	if !hasTestAllStep(summary, "compiler pipeline focused gate") || !hasTestAllStep(summary, "api diff no-change") {
		t.Fatalf("stabilization summary missing focused gates: %#v", summary.Steps)
	}
	if !hasTestAllStep(summary, "wasi runner smoke") || !hasTestAllStep(summary, "web ui browser smoke") {
		t.Fatalf("stabilization summary missing smoke gates: %#v", summary.Steps)
	}
	if summary.StepCount <= 24 {
		t.Fatalf("stabilization should extend full gate step count, got %d", summary.StepCount)
	}
	raw, err := os.ReadFile(goLog)
	if err != nil {
		t.Fatalf("read fake go log: %v", err)
	}
	log := string(raw)
	for _, want := range []string{
		"test ./tools/cmd/validate-lsp-stdio/...",
		"./tools/cmd/validate-api-docs/...",
		"./tools/cmd/verify-docs/...",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("stabilization missing validator go invocation %q in log:\n%s", want, log)
		}
	}
}

func hasTestAllStep(summary testAllSummary, name string) bool {
	for _, step := range summary.Steps {
		if step.Name == name && step.Status == "pass" {
			return true
		}
	}
	return false
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
	if err := os.WriteFile(filepath.Join(root, "scripts", "release_v1_0_wasi_smoke.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\nreport=\"\"\nwhile [[ $# -gt 0 ]]; do case \"$1\" in --report) report=\"$2\"; shift 2 ;; *) shift ;; esac; done\nmkdir -p \"$(dirname \"$report\")\"\nprintf '{\"status\":\"pass\",\"cases\":[]}\\n' >\"$report\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "release_v1_0_web_smoke.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\nreport=\"\"\nwhile [[ $# -gt 0 ]]; do case \"$1\" in --report) report=\"$2\"; shift 2 ;; *) shift ;; esac; done\nmkdir -p \"$(dirname \"$report\")\"\nprintf '{\"status\":\"pass\",\"ui_schema\":\"tetra.ui.bundle.v1\",\"cases\":[]}\\n' >\"$report\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "release_v1_0_api_diff.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\nreport_dir=\"\"\nwhile [[ $# -gt 0 ]]; do case \"$1\" in --report-dir) report_dir=\"$2\"; shift 2 ;; *) shift ;; esac; done\nmkdir -p \"$report_dir\"\nprintf '{\"review\":{\"status\":\"clean\"},\"diff\":{\"added\":[],\"removed\":[],\"changed\":[]}}\\n' >\"$report_dir/api-diff.json\"\n"), 0o755); err != nil {
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
    echo "v0.2.0"
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
          *planned-actor-diagnostic.tetra*) echo '{"code":"TETRA0001","message":"actor declarations currently support state fields and func methods only","severity":"error"}' >&2 ;;
          *) echo '{"code":"TETRA2001","message":"unknown function missing_call","severity":"error"}' >&2 ;;
        esac
        exit 1
      fi
    done
    ;;
  doc)
    printf '%s\n' '# Tetra API Docs' ''
    printf '%s\n' '<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:ede46e5e34948c25f6ec38b0b963a2d8d42f5aa09071128581ee08271e966459","module_count":1,"entry_count":1} -->' ''
    printf '%s\n' '## examples' '' '### Functions' ''
    printf '%b\n' '- \x60func main() -> Int\x60'
    ;;
  build)
    out=""
    prev=""
    for arg in "$@"; do
      if [[ "$prev" == "-o" ]]; then
        out="$arg"
      fi
      prev="$arg"
    done
    if [[ -n "$out" ]]; then
      mkdir -p "$(dirname "$out")"
      printf '\x00\x61\x73\x6d\x01\x00\x00\x00' >"$out"
    fi
    ;;
  targets)
    printf '{"supported":["linux-x64","windows-x64","macos-x64"],"build_only":["wasm32-wasi","wasm32-web"],"planned":[]}\n'
    ;;
  doctor)
    printf '{"status":"pass","checks":[{"name":"version","status":"pass"},{"name":"supported targets","status":"pass"},{"name":"build-only targets","status":"pass"},{"name":"planned targets","status":"pass"},{"name":"repo root","status":"pass"},{"name":"__rt/actors_sysv.tetra","status":"pass"},{"name":"__rt/actors_win64.tetra","status":"pass"},{"name":"compiler/selfhostrt/actors_sysv.tetra","status":"pass"},{"name":"compiler/selfhostrt/actors_win64.tetra","status":"pass"},{"name":"examples/flow_hello.tetra","status":"pass"},{"name":"docs/generated/manifest.json","status":"pass"},{"name":"docs manifest version","status":"pass"},{"name":"docs manifest surface","status":"pass"},{"name":"smoke sources","status":"pass"},{"name":"runtime exports","status":"pass"}]}\n'
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
    case "$sub" in
      verify)
        lock=""
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --lock)
              lock="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        if [[ -n "$lock" ]]; then
          mkdir -p "$(dirname "$lock")"
          cat >"$lock" <<'JSON'
{
  "schema": "tetra.eco.lock.v1",
  "manifest_schema": "tetra.capsule.v1",
  "permissions_model": "tetra.eco.permissions.v1",
  "graph_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "capsules": [
    {
      "id": "tetra://app",
      "name": "App",
      "version": "0.1.0",
      "path": "Tetra.capsule",
      "targets": ["linux-x64"],
      "permissions": ["io"]
    }
  ]
}
JSON
        fi
        ;;
      seed)
        action="${1:-}"
        shift || true
        case "$action" in
          export)
            out="tetra.seed.json"
            while [[ $# -gt 0 ]]; do
              case "$1" in
                --out)
                  out="$2"
                  shift 2
                  ;;
                *)
                  shift
                  ;;
              esac
            done
            mkdir -p "$(dirname "$out")"
            printf '{}\n' >"$out"
            ;;
          import)
            lock=""
            capsules_dir=""
            while [[ $# -gt 0 ]]; do
              case "$1" in
                --lock)
                  lock="$2"
                  shift 2
                  ;;
                --capsules-dir)
                  capsules_dir="$2"
                  shift 2
                  ;;
                *)
                  shift
                  ;;
              esac
            done
            if [[ -n "$lock" ]]; then
              mkdir -p "$(dirname "$lock")"
              printf '{}\n' >"$lock"
            fi
            if [[ -n "$capsules_dir" ]]; then
              mkdir -p "$capsules_dir"
              printf 'capsule App:\n  id "tetra://app"\n  version "0.1.0"\n' >"$capsules_dir/App.capsule"
            fi
            ;;
        esac
        ;;
      needmap)
        out="tetra.needmap.json"
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
        mkdir -p "$(dirname "$out")"
        printf '{}\n' >"$out"
        ;;
      pack)
        out=""
        while [[ $# -gt 0 ]]; do
          case "$1" in
            -o|--out)
              out="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        if [[ -n "$out" ]]; then
          mkdir -p "$(dirname "$out")"
          printf 'todex\n' >"$out"
        fi
        ;;
      unpack)
        out=""
        while [[ $# -gt 0 ]]; do
          case "$1" in
            -C|--dir)
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
          printf 'capsule App:\n  id "tetra://app"\n  version "0.1.0"\n  target "linux-x64"\n' >"$out/Tetra.capsule"
          printf 'func main() -> Int:\n    return 0\n' >"$out/src/main.tetra"
          printf '{"schema":"tetra.eco.package.v1","compression":"gzip","mtime_unix":0,"file_count":2,"files":[{"path":"Tetra.capsule","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":68},{"path":"src/main.tetra","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":32}]}\n' >"$out/tetra.package.json"
        fi
        ;;
      vault)
        action="${1:-}"
        shift || true
        store=".tetra/todex-vault"
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --store)
              store="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        mkdir -p "$store/objects/sha256"
        printf '{}' >"$store/records.json"
        if [[ "$action" == "add" ]]; then
          echo "Vault added: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa source fixture"
        fi
        if [[ "$action" == "verify" ]]; then
          echo "Vault OK: 1 records"
        fi
        ;;
      trust)
        action="${1:-}"
        shift || true
        if [[ "$action" == "snapshot" ]]; then
          out="tetra.trust-snapshot.json"
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
          mkdir -p "$(dirname "$out")"
          printf '{}\n' >"$out"
        fi
        ;;
      materialize)
        out="."
        while [[ $# -gt 0 ]]; do
          case "$1" in
            -C|--dir)
              out="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        mkdir -p "$out"
        printf '{}\n' >"$out/tetra.materialization.json"
        ;;
      publish)
        registry=".tetra/registry-beta"
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --registry)
              registry="$2"
              shift 2
              ;;
            *)
              shift
              ;;
          esac
        done
        mkdir -p "$registry/packages/tetra_app/0.1.0/linux-x64"
        printf 'todex\n' >"$registry/packages/tetra_app/0.1.0/linux-x64/package.todex"
        printf '{"schema":"tetra.eco.publish.v1beta","channel":"beta","hub":"local-beta","published_at_unix":0,"capsule":{"id":"tetra://app","name":"App","version":"0.1.0","target":"linux-x64","targets":["linux-x64"],"permissions":["io"]},"package":{"file":"package.todex","size":6,"sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"downloads":[{"target":"linux-x64","path":"packages/tetra_app/0.1.0/linux-x64/package.todex"}]}\n' >"$registry/packages/tetra_app/0.1.0/linux-x64/metadata.json"
        ;;
      download)
        out=""
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
          mkdir -p "$(dirname "$out")"
          printf 'todex\n' >"$out"
        fi
        ;;
      tetrahub)
        action="${1:-}"
        shift || true
        if [[ "$action" == "publish" ]]; then
          store=".tetra/tetrahub-beta"
          while [[ $# -gt 0 ]]; do
            case "$1" in
              --store)
                store="$2"
                shift 2
                ;;
              *)
                shift
                ;;
            esac
          done
          mkdir -p "$store/packages/tetra_app/0.1.0/linux-x64"
          printf 'todex\n' >"$store/packages/tetra_app/0.1.0/linux-x64/package.todex"
        elif [[ "$action" == "download" ]]; then
          out=""
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
            mkdir -p "$(dirname "$out")"
            printf 'todex\n' >"$out"
          fi
        fi
        ;;
    esac
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

func runTestAllFromWorkingDir(t *testing.T, root string, workingDir string, env []string, args ...string) ([]byte, error) {
	t.Helper()
	script := filepath.Join(root, "scripts", "test_all.sh")
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Dir = workingDir
	cmd.Env = append(os.Environ(), env...)
	cmd.Env = append(cmd.Env, "PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	return cmd.CombinedOutput()
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
