package main

import (
	"strings"
	"testing"
)

func TestValidateWASISmokeReportAcceptsRuntimeRunnerReport(t *testing.T) {
	raw := runtimeReport(`"runner":"node-wasi",`, runtimeCases(true, true))
	if err := validateWASISmokeReport([]byte(raw), validationModeRuntime); err != nil {
		t.Fatalf("validate runtime report: %v", err)
	}
}

func TestValidateWASISmokeReportAcceptsArtifactPreflightReport(t *testing.T) {
	raw := runtimeReport("", artifactCases())
	if err := validateWASISmokeReport([]byte(raw), validationModeArtifact); err != nil {
		t.Fatalf("validate artifact/import preflight report: %v", err)
	}
}

func TestValidateWASISmokeReportRejectsRuntimeReportWithoutRunner(t *testing.T) {
	raw := runtimeReport("", artifactCases())
	err := validateWASISmokeReport([]byte(raw), validationModeRuntime)
	if err == nil || !strings.Contains(err.Error(), "runtime report missing WASI runner") {
		t.Fatalf("error = %v, want missing WASI runner", err)
	}
}

func TestValidateWASISmokeReportRejectsArtifactReportThatRanCases(t *testing.T) {
	raw := runtimeReport(`"runner":"wasmtime",`, runtimeCases(true, true))
	err := validateWASISmokeReport([]byte(raw), validationModeArtifact)
	if err == nil || !strings.Contains(err.Error(), "artifact/import preflight report cannot include runner") {
		t.Fatalf("error = %v, want artifact/import preflight runner rejection", err)
	}
}

func TestValidateWASISmokeReportRejectsRuntimeCaseWithoutActualExit(t *testing.T) {
	raw := runtimeReport(`"runner":"wasmtime",`, runtimeCases(true, false))
	err := validateWASISmokeReport([]byte(raw), validationModeRuntime)
	if err == nil || !strings.Contains(err.Error(), "ran without actual_exit") {
		t.Fatalf("error = %v, want missing actual_exit rejection", err)
	}
}

func TestValidateWASISmokeReportRejectsUnexpectedSourceContract(t *testing.T) {
	raw := strings.Replace(runtimeReport(`"runner":"node-wasi",`, runtimeCases(true, true)), "examples/effects_io_smoke.tetra", "examples/hello.tetra", 1)
	err := validateWASISmokeReport([]byte(raw), validationModeRuntime)
	if err == nil || !strings.Contains(err.Error(), "src_path") {
		t.Fatalf("error = %v, want src_path contract rejection", err)
	}
}

func TestValidateWASISmokeReportRejectsMissingMultiReturnEvidence(t *testing.T) {
	raw := strings.Replace(runtimeReport(`"runner":"node-wasi",`, runtimeCases(true, true)), caseJSON("wasm_multi_return_3_smoke", "examples/wasm_multi_return_3_smoke.tetra", 0, "true", `"actual_exit":0,`), caseJSON("dogfood_web_ui", "examples/projects/dogfood_web_ui/src/main.tetra", 0, "true", `"actual_exit":0,`), 1)
	err := validateWASISmokeReport([]byte(raw), validationModeRuntime)
	if err == nil || !strings.Contains(err.Error(), "wasm_multi_return_3_smoke") {
		t.Fatalf("error = %v, want missing multi-return evidence rejection", err)
	}
}

func TestValidateWASISmokeReportRejectsUnsupportedDiagnosticDrift(t *testing.T) {
	raw := strings.Replace(runtimeReport(`"runner":"node-wasi",`, runtimeCases(true, true)), `"expected_diagnostic":"runtime not supported on wasm32"`, `"expected_diagnostic":"runtime supported on wasm32"`, 1)
	err := validateWASISmokeReport([]byte(raw), validationModeRuntime)
	if err == nil || !strings.Contains(err.Error(), "time_sleep_smoke") || !strings.Contains(err.Error(), "expected_diagnostic") {
		t.Fatalf("error = %v, want unsupported diagnostic contract rejection", err)
	}
}

func TestValidateWASISmokeReportRejectsUnknownFields(t *testing.T) {
	raw := strings.Replace(runtimeReport(`"runner":"node-wasi",`, runtimeCases(true, true)), `"failed":0,`, `"failed":0,"extra":true,`, 1)
	err := validateWASISmokeReport([]byte(raw), validationModeRuntime)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error = %v, want unknown field rejection", err)
	}
}

func runtimeReport(runner string, cases string) string {
	return `{
		"timestamp":"2026-04-30T10:40:50Z",
		"target":"wasm32-wasi",
		"build_only":false,
		` + runner + `
		"host":"linux-x64",
		"version":"v0.3.0",
		"git_head":"b884653",
		"islands_debug":false,
		"total":13,
		"passed":13,
		"failed":0,
		"cases":[` + cases + `]
	}`
}

func runtimeCases(ran bool, includeActualExit bool) string {
	actual := ""
	if includeActualExit {
		actual = `"actual_exit":0,`
	}
	ranLiteral := "false"
	if ran {
		ranLiteral = "true"
	}
	return caseJSON("legacy_hello", "examples/hello.tetra", 0, ranLiteral, actual) + "," +
		caseJSON("effects_io_smoke", "examples/effects_io_smoke.tetra", 0, ranLiteral, actual) + "," +
		caseJSON("ui_web_smoke", "examples/ui_web_smoke.tetra", 0, ranLiteral, actual) + "," +
		caseJSON("core_slices_smoke", "examples/core_slices_smoke.tetra", 0, ranLiteral, actualForExpected(includeActualExit, 0)) + "," +
		caseJSON("wasm_globals_smoke", "examples/wasm_globals_smoke.tetra", 0, ranLiteral, actual) + "," +
		caseJSON("wasm_multi_return_2_smoke", "examples/wasm_multi_return_2_smoke.tetra", 0, ranLiteral, actual) + "," +
		caseJSON("wasm_multi_return_3_smoke", "examples/wasm_multi_return_3_smoke.tetra", 0, ranLiteral, actual) + "," +
		caseJSON("wasm_multi_return_4_smoke", "examples/wasm_multi_return_4_smoke.tetra", 0, ranLiteral, actual) + "," +
		caseJSON("dogfood_wasi", "examples/projects/dogfood_wasi/src/main.tetra", 0, ranLiteral, actual) + "," +
		caseJSON("dogfood_web_ui", "examples/projects/dogfood_web_ui/src/main.tetra", 0, ranLiteral, actual) + "," +
		unsupportedCaseJSON("time_sleep_smoke", "examples/time_sleep_smoke.tetra", 0, "runtime not supported on wasm32") + "," +
		unsupportedCaseJSON("task_smoke", "examples/task_smoke.tetra", 42, "runtime not supported on wasm32") + "," +
		unsupportedCaseJSON("actors_pingpong", "examples/actors_pingpong.tetra", 0, "runtime not supported on wasm32")
}

func artifactCases() string {
	return runtimeCases(false, false)
}

func actualForExpected(include bool, expected int) string {
	if !include {
		return ""
	}
	if expected == 42 {
		return `"actual_exit":42,`
	}
	return `"actual_exit":0,`
}

func caseJSON(name string, srcPath string, expectedExit int, ran string, actual string) string {
	return `{
		"name":"` + name + `",
		"src_path":"` + srcPath + `",
		"out_path":"docs/generated/v1_0/wasi-smoke-artifacts/` + name + `.wasm",
		"expected_exit":` + intLiteral(expectedExit) + `,
		` + actual + `
		"ran":` + ran + `,
		"pass":true
	}`
}

func unsupportedCaseJSON(name string, srcPath string, expectedExit int, expectedDiagnostic string) string {
	return `{
		"name":"` + name + `",
		"src_path":"` + srcPath + `",
		"out_path":"",
		"expected_exit":` + intLiteral(expectedExit) + `,
		"unsupported":true,
		"expected_diagnostic":"` + expectedDiagnostic + `",
		"diagnostic":"wasm backend: function 'main' calls ` + expectedDiagnostic + `",
		"ran":false,
		"pass":true
	}`
}

func intLiteral(v int) string {
	if v == 42 {
		return "42"
	}
	return "0"
}
