package testall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSmokeSourceSetsUseUnifiedRegistry(t *testing.T) {
	root := repoRoot(t)
	files := map[string][]string{
		"scripts/ci/test-all.sh": {
			`./tetra smoke --list --target linux-x64 --format=json >"$report_dir/smoke-list.json"`,
			`run_tetra_smoke_target "linux-x64" "true" "$report_dir/host-smoke.json"`,
			`run_tetra_smoke_target "wasm32-wasi" "false" "$report_dir/wasm32-wasi-artifact-smoke.json"`,
			`run_tetra_smoke_target "wasm32-web" "false" "$report_dir/wasm32-web-artifact-smoke.json"`,
		},
		"scripts/release/v1_0/wasi-smoke.sh": {
			`./tetra smoke --list --target wasm32-wasi --format=json >"$smoke_list"`,
			`smoke_source_for_case "$smoke_list" "dogfood_wasi"`,
			`smoke_source_for_case "$smoke_list" "ui_web_smoke"`,
		},
		"scripts/release/v1_0/web-smoke.sh": {
			`./tetra smoke --list --target wasm32-web --format=json >"$smoke_list"`,
			`smoke_source_for_case "$smoke_list" "dogfood_web_ui"`,
			`smoke_source_for_case "$smoke_list" "ui_web_smoke"`,
		},
	}
	for rel, wants := range files {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(raw)
		for _, want := range wants {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing unified smoke registry contract %q", rel, want)
			}
		}
		for _, forbidden := range []string{
			`release_smoke_cases=`,
			`examples/projects/dogfood_wasi/src/main.tetra`,
			`examples/projects/dogfood_web_ui/src/main.tetra`,
			`examples/ui/ui_web_smoke.tetra`,
			`examples/flow/flow_struct_smoke.tetra`,
		} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s still hard-codes smoke source %q", rel, forbidden)
			}
		}
	}
}

func TestTestAllWASMSchemaChecksUseArtifactSmokeReports(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "ci", "test-all.sh"))
	if err != nil {
		t.Fatalf("read test-all.sh: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`local report="$report_dir/$target-artifact-smoke.json"`,
		`go run ./tools/cmd/validate-wasi-smoke-report --mode artifact --report "$report"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test-all WASM schema check missing %q", want)
		}
	}
}

func TestTestAllValidatesDocsManifests(t *testing.T) {
	raw, err := readTestAllScript(t)
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
	raw, err := readTestAllScript(t)
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

func TestTestAllFullValidatesTechEmpowerReportsWhenPresent(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`check_techempower_reports()`,
		`docs/benchmarks/techempower_local_smoke_skip_db_report.json`,
		`docs/benchmarks/techempower_scram_single_query_local_report.json`,
		`docs/benchmarks/techempower_scram_single_query_matrix_local_report.json`,
		`docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json`,
		`go run ./tools/cmd/validate-techempower-report --report "$report"`,
		`go run ./tools/cmd/validate-techempower-report --report "$report" --allow-skip-db`,
		`run_step "techempower report schemas" check_techempower_reports`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all missing TechEmpower report wiring %q", want)
		}
	}
}

func TestTestAllFullRunsTechEmpowerReportValidationForPresentReports(t *testing.T) {
	root := testAllFakeRepo(t, false)
	for _, report := range []string{
		"docs/benchmarks/techempower_local_smoke_skip_db_report.json",
		"docs/benchmarks/techempower_scram_single_query_local_report.json",
		"docs/benchmarks/techempower_scram_single_query_matrix_local_report.json",
		"docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json",
	} {
		path := filepath.Join(root, report)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reportDir := filepath.Join(root, "report")
	goLog := filepath.Join(root, "go.log")
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_GO_LOG=" + goLog},
		"--full",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err != nil {
		t.Fatalf("test_all full failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if !hasTestAllStep(summary, "techempower report schemas") {
		t.Fatalf("full test_all summary missing TechEmpower report schema step: %#v", summary.Steps)
	}
	rawLog, err := os.ReadFile(goLog)
	if err != nil {
		t.Fatalf("read fake go log: %v", err)
	}
	log := string(rawLog)
	for _, want := range []string{
		`run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_local_smoke_skip_db_report.json --allow-skip-db`,
		`run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_local_report.json`,
		`run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_matrix_local_report.json`,
		`run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json`,
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("full gate missing TechEmpower validator call %q in log:\n%s", want, log)
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
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `./tetra lsp --stdio-smoke examples/flow/flow_hello.tetra >"$report_dir/lsp-smoke.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing persisted LSP smoke report %q", want)
	}
	validator := `go run ./tools/cmd/validate-lsp-smoke --report "$report_dir/lsp-smoke.json"`
	if !strings.Contains(string(raw), validator) {
		t.Fatalf("test_all missing LSP smoke validation %q", validator)
	}
}

func TestTestAllValidatesLSPStdioTranscript(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-lsp-stdio --transcript "$tmp_dir/lsp-stdio.out"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing LSP stdio validation %q", want)
	}
}

func TestTestAllValidatesGeneratedAPIDocs(t *testing.T) {
	raw, err := readTestAllScript(t)
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
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v0_1_2", "gate.sh"))
	if err != nil {
		t.Fatalf("read v0.1.2 release gate: %v", err)
	}
	want := `./tetra fmt --check examples lib __rt compiler/selfhostrt`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("v0.1.2 release gate missing runtime formatter coverage %q", want)
	}
}

func TestTestAllValidatesEcoLock(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.lock.json"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing Eco lock validation %q", want)
	}
}

func TestTestAllValidatesEcoUnpack(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-unpack --dir "$tmp_dir/unpacked"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing Eco unpack validation %q", want)
	}
}

func TestTestAllValidatesEcoVault(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	want := `go run ./tools/cmd/validate-eco-vault --store "$tmp_dir/vault"`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("test_all missing Eco vault validation %q", want)
	}
}

func Test_test_all_validates_tool011_eco_reports(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	for _, want := range []string{
		`go run ./tools/cmd/validate-eco-seed --seed "$tmp_dir/tetra.seed.json"`,
		`go run ./tools/cmd/validate-eco-needmap --needmap "$tmp_dir/tetra.needmap.json"`,
		`go run ./tools/cmd/validate-eco-trust --trust "$tmp_dir/tetra.trust-snapshot.json"`,
		`go run ./tools/cmd/validate-eco-materialization --materialization "$tmp_dir/materialized/tetra.materialization.json"`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("test_all missing TOOL-011 Eco validation %q", want)
		}
	}
}

func TestTestAllFullValidatesCrossTargetSmokeReports(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	goLog := filepath.Join(root, "go.log")
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_GO_LOG=" + goLog},
		"--full",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err != nil {
		t.Fatalf("test_all full failed: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(goLog)
	if err != nil {
		t.Fatalf("read fake go log: %v", err)
	}
	log := string(raw)
	for _, report := range []string{
		"linux-smoke.json",
		"macos-smoke.json",
		"windows-smoke.json",
		"wasm32-wasi-artifact-smoke.json",
		"wasm32-web-artifact-smoke.json",
	} {
		want := "run ./tools/cmd/smoke-report-to-checklist --validate-only --report "
		if !strings.Contains(log, want) || !strings.Contains(log, report) {
			t.Fatalf("full gate missing validate-only call for %s in log:\n%s", report, log)
		}
	}
	if !strings.Contains(
		log,
		"run ./tools/cmd/validate-wasi-smoke-report --mode artifact --report ",
	) ||
		!strings.Contains(log, "wasm32-wasi-artifact-smoke.json") {
		t.Fatalf("full gate missing strict WASI artifact/import report validation in log:\n%s", log)
	}
	summary := decodeTestAllSummary(t, out)
	for _, step := range []string{"wasm32-wasi smoke schema", "wasm32-web smoke schema"} {
		if !hasTestAllStep(summary, step) {
			t.Fatalf("full gate missing %q step: %#v", step, summary.Steps)
		}
	}
}

func TestTestAllWasmSchemaValidationUsesPersistedArtifactReports(t *testing.T) {
	raw, err := readTestAllScript(t)
	if err != nil {
		t.Fatalf("read test_all: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`local report="$report_dir/$target-artifact-smoke.json"`,
		`test -s "$report" || return 1`,
		`go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$report" || return 1`,
		`go run ./tools/cmd/validate-wasi-smoke-report --mode artifact --report "$report" || return 1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("test_all wasm schema validation missing %q", want)
		}
	}
}

func TestTestAllFullAggregatesToolingSummary(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, nil, "--full", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all full failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if !hasTestAllStep(summary, "tooling summary aggregation") {
		t.Fatalf("full gate missing tooling summary aggregation step: %#v", summary.Steps)
	}
	raw, err := os.ReadFile(filepath.Join(reportDir, "tooling-summary.json"))
	if err != nil {
		t.Fatalf("read tooling summary: %v", err)
	}
	for _, want := range []string{
		`"schema": "tetra.tooling-summary.v1alpha1"`,
		`"targets.json"`,
		`"doctor.json"`,
		`"safety-readiness.json"`,
		`"wasm32-wasi-artifact-smoke.json"`,
		`"wasm32-web-artifact-smoke.json"`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("tooling summary missing %q:\n%s", want, raw)
		}
	}
}

func TestTestAllFullToolingSummaryFailsOnZeroByteRequiredArtifact(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_ZERO_DOCTOR_REPORT=1"},
		"--full",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err == nil {
		t.Fatalf("expected full tooling summary to fail on zero-byte required artifact\n%s", out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	var sawToolingFailure bool
	for _, step := range summary.Steps {
		if step.Name == "tooling summary aggregation" && step.Status == "fail" {
			sawToolingFailure = true
		}
	}
	if !sawToolingFailure {
		t.Fatalf("summary missing failing tooling summary step: %#v", summary.Steps)
	}
	logRaw, err := os.ReadFile(
		filepath.Join(reportDir, testAllStepLog(t, summary, "tooling summary aggregation")),
	)
	if err != nil {
		t.Fatalf("read tooling summary log: %v", err)
	}
	log := string(logRaw)
	if !strings.Contains(log, "required artifact is zero-byte: doctor.json") {
		t.Fatalf("tooling summary log missing zero-byte artifact detail:\n%s", log)
	}
}

func TestTestAllStabilizationToolingSummaryRequiresFocusedArtifacts(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_SKIP_WEB_UI_SMOKE_REPORT=1"},
		"--stabilization",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err == nil {
		t.Fatalf(
			"expected stabilization tooling summary to fail on missing focused artifact\n%s",
			out,
		)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	var sawToolingFailure bool
	for _, step := range summary.Steps {
		if step.Name == "tooling summary aggregation" && step.Status == "fail" {
			sawToolingFailure = true
		}
	}
	if !sawToolingFailure {
		t.Fatalf("summary missing failing tooling summary step: %#v", summary.Steps)
	}
	logRaw, err := os.ReadFile(
		filepath.Join(reportDir, testAllStepLog(t, summary, "tooling summary aggregation")),
	)
	if err != nil {
		t.Fatalf("read tooling summary log: %v", err)
	}
	log := string(logRaw)
	if !strings.Contains(log, "required artifact missing: web-ui-smoke.json") {
		t.Fatalf("tooling summary log missing required artifact detail:\n%s", log)
	}
}

func TestTestAllStabilizationRunsFocusedGates(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	goLog := filepath.Join(root, "go.log")
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_GO_LOG=" + goLog},
		"--stabilization",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err != nil {
		t.Fatalf("test_all stabilization failed: %v\n%s", err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "pass" {
		t.Fatalf("summary status = %q", summary.Status)
	}
	if !hasTestAllStep(summary, "compiler pipeline focused gate") ||
		!hasTestAllStep(summary, "api diff no-change") {
		t.Fatalf("stabilization summary missing focused gates: %#v", summary.Steps)
	}
	if !hasTestAllStep(summary, "wasi runner smoke") ||
		!hasTestAllStep(summary, "web runtime browser smoke") {
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
		"./tools/cmd/validate-wasi-smoke-report/...",
		"./tools/cmd/verify-docs/...",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("stabilization missing validator go invocation %q in log:\n%s", want, log)
		}
	}
}
