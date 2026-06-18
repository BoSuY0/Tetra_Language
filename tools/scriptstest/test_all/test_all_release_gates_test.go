package testall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV06GateValidatesCrossTargetSmokeReports(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
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
	assertLegacyFileRemoved(
		t,
		"scripts/release_v0_6_gate.sh",
		"scripts/release/v0_6/gate.sh directly",
	)
	raw, err := readReleaseV06GateScript(t)
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
	root := repoRoot(t)
	assertLegacyFileRemoved(
		t,
		"scripts/release_v0_5_gate.sh",
		"scripts/release/v0_5/gate.sh directly",
	)
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "v0_5", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.5 release gate: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, `repo_root="$(cd "$script_dir/../../.." && pwd)"`) {
		t.Fatalf("v0.5 release gate should resolve the repo root from its versioned script path")
	}
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
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-test-report --report "$tmp_dir/tetra-test-report.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing test report validation %q", want)
	}
}

func TestReleaseV06GateValidatesDocsManifests(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
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
	raw, err := readReleaseV06GateScript(t)
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
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-lsp-smoke --report "$tmp_dir/lsp-smoke.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing LSP smoke validation %q", want)
	}
}

func TestReleaseV06GateValidatesLSPStdioTranscript(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-lsp-stdio --transcript "$tmp_dir/lsp-stdio.out"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing LSP stdio validation %q", want)
	}
}

func TestReleaseV06GateValidatesGeneratedAPIDocs(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-api-docs --docs "$tmp_dir/api-docs.md"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing API docs validation %q", want)
	}
}

func TestReleaseV06GateFormatterCoversRuntimeSources(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `./tetra fmt --check examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing runtime formatter coverage %q", want)
	}
}

func TestReleaseV06GateScansFlowOnlySources(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing Flow-only scan %q", want)
	}
}

func TestReleaseV06GateValidatesTargetsJSONReport(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
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
	raw, err := readReleaseV06GateScript(t)
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
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`./tetra check examples/flow/flow_hello.tetra`,
		`./tetra doc examples >"$tmp_dir/tetra-docs.md"`,
		`go run ./tools/cmd/validate-api-docs --docs "$tmp_dir/tetra-docs.md"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing CLI command coverage %q", want)
		}
	}
}

func TestReleaseV06GateValidatesJSONDiagnosticShape(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
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
		`./tetra build --target wasm32-wasi -o "$wasm_out" examples/smoke/basic/hello.tetra`,
		`test "$(od -An -tx1 -N4 "$wasm_out" | tr -d ' \n')" = "0061736d"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release gate missing JSON diagnostic validation %q", want)
		}
	}
}

func TestReleaseV06GateValidatesSmokeListJSONReport(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
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
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.lock.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing Eco lock validation %q", want)
	}
}

func TestReleaseV06GateValidatesEcoUnpack(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-unpack --dir "$tmp_dir/unpacked"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing Eco unpack validation %q", want)
	}
}

func TestReleaseV06GateValidatesEcoVault(t *testing.T) {
	raw, err := readReleaseV06GateScript(t)
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-vault --store "$tmp_dir/vault"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("release gate missing Eco vault validation %q", want)
	}
}

func TestTestAllValidatesTestRunnerJSONReport(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-test-report --report "$report_dir/tetra-test-report.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing test report validation %q", want)
	}
}

func TestTestAllValidatesSummaryArtifacts(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-test-all-summary --summary "$summary_json" --report-dir "$report_dir" --format=json`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing summary validation %q", want)
	}
}

func TestTestAllTopLevelGoTestBypassesCache(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `run_step "go test all packages" env -u TETRA_TEST_ALL_RELEASE_VERSION -u TETRA_TEST_ALL_RELEASE_ARTIFACT -u TETRA_SECURITY_REVIEW_SIGNOFF go test ./compiler/... ./cli/... ./tools/... -count=1`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all top-level go test should bypass cache with %q", want)
	}
	wantRepo := `run_step "repo test script" env -u TETRA_TEST_ALL_RELEASE_VERSION -u TETRA_TEST_ALL_RELEASE_ARTIFACT -u TETRA_SECURITY_REVIEW_SIGNOFF bash scripts/ci/test.sh`
	if !strings.Contains(string(raw), wantRepo) {
		t.Fatalf("test_all repo script should clear release signoff env with %q", wantRepo)
	}
}

func TestTestAllChecksShortAliasVersion(t *testing.T) {
	raw, err := readTestAllScript(t)
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
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	run := `run_tetra_smoke_target "linux-x64" "true" "$report_dir/host-smoke.json"`
	if !strings.Contains(text, run) {
		t.Fatalf("test_all missing host smoke report command %q", run)
	}
	validate := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$report_path" --format=json`
	if !strings.Contains(text, validate) {
		t.Fatalf("test_all missing host smoke report validation %q", validate)
	}
	toonValidate := `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$toon_report_path" --format=toon`
	if !strings.Contains(text, toonValidate) {
		t.Fatalf("test_all missing host smoke TOON validation %q", toonValidate)
	}
	if strings.Contains(text, `release_smoke_cases=`) {
		t.Fatalf("test_all should not keep a local release smoke case array")
	}
}

func TestTestAllFormatterCoversRuntimeSources(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `run_step "formatter check examples lib runtime" ./tetra fmt --check examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing runtime formatter coverage %q", want)
	}
}

func TestTestAllScansFlowOnlySources(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `run_step "flow-only source scan" go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing Flow-only scan %q", want)
	}
}

func TestTestAllValidatesTargetsJSONReport(t *testing.T) {
	raw, err := readTestAllScript(t)
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
	raw, err := readTestAllScript(t)
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
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "tetra check flow hello" ./tetra check examples/flow/flow_hello.tetra`,
		`run_step "tetra doc examples" check_tetra_doc`,
		`./tetra doc examples >"$report_dir/tetra-docs.md"`,
		`go run ./tools/cmd/validate-api-docs --docs "$report_dir/tetra-docs.md"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing CLI command coverage %q", want)
		}
	}
}

func TestTestAllFullRunsSafetyReadinessGate(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "safety readiness evidence" check_safety_readiness`,
		`go run ./tools/cmd/validate-safety-readiness`,
		`--out "$report_dir/safety-readiness.json"`,
		`go test ./compiler/... -run 'Ownership|Borrow|Consume|Inout|Lifetime|Resource|Island|Actor|Task|Unsafe|Capability|Effect|Privacy|Consent|Budget|MMIO|Mem' -count=1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing safety readiness coverage %q", want)
		}
	}
}

func TestTestAllFullFailsOnSafetyReadinessValidatorFailure(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAIL_SAFETY_READINESS=1"},
		"--full",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err == nil {
		t.Fatalf("expected full gate to fail on safety readiness validator failure\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	var sawSafetyFailure bool
	for _, step := range summary.Steps {
		if step.Name == "safety readiness evidence" && step.Status == "fail" {
			sawSafetyFailure = true
		}
		if step.Name == "tooling summary aggregation" {
			t.Fatalf(
				"tooling summary should not run after safety readiness failure without keep-going: %#v",
				summary.Steps,
			)
		}
	}
	if !sawSafetyFailure {
		t.Fatalf("summary missing failing safety readiness step: %#v", summary.Steps)
	}
	logRaw, err := os.ReadFile(
		filepath.Join(reportDir, testAllStepLog(t, summary, "safety readiness evidence")),
	)
	if err != nil {
		t.Fatalf("read safety readiness log: %v", err)
	}
	if !strings.Contains(string(logRaw), "safety readiness failed") {
		t.Fatalf("safety readiness log missing validator failure detail:\n%s", logRaw)
	}
}

func TestTestAllFullRunsOwnershipAuditGate(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "ownership production audit" go run ./tools/cmd/validate-ownership-audit --audit docs/release/production/ownership_production_audit.md --expected-status achieved`,
		`docs/release/production/ownership_production_audit.md`,
		`--expected-status achieved`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing ownership audit coverage %q", want)
		}
	}
}

func TestTestAllValidatesJSONDiagnosticShape(t *testing.T) {
	raw, err := readTestAllScript(t)
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
		`./tetra build --target wasm32-wasi -o "$wasm_out" examples/smoke/basic/hello.tetra`,
		`test "$(od -An -tx1 -N4 "$wasm_out" | tr -d ' \n')" = "0061736d"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing JSON diagnostic validation %q", want)
		}
	}
}

func TestTestAllValidatesSmokeListJSONReport(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`run_step "smoke list json report" check_smoke_list`,
		`./tetra smoke --list --target linux-x64 --format=json >"$report_dir/smoke-list.json"`,
		`go run ./tools/cmd/validate-smoke-list --report "$report_dir/smoke-list.json" --examples-root examples --format=json`,
		`./tetra smoke --list --target linux-x64 --format=toon >"$report_dir/smoke-list.toon"`,
		`go run ./tools/cmd/validate-smoke-list --report "$report_dir/smoke-list.toon" --examples-root examples --format=toon`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing smoke list validation %q", want)
		}
	}
}
