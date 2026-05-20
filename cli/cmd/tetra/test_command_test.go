package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTestCommandJSONDiagnosticsForWASMRuntimeUnsupported(t *testing.T) {
	diag := runCLIJSONDiagnostic(t, []string{"test", "--diagnostics=json", "--target", "wasm32-web"}, 2)
	for _, want := range []string{"cannot run tests for target wasm32-web", "WASM test runner is not part of the current production runtime contract", "smoke/runtime reports"} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
}

func TestTestCommandJSONDiagnosticsForHostTargetMismatch(t *testing.T) {
	target := nonHostTarget(t)
	diag := runCLIJSONDiagnostic(t, []string{"test", "--diagnostics=json", "--target", target}, 2)
	if diag.Code != "TETRA0001" || diag.Severity != "error" || !strings.Contains(diag.Message, "cannot run tests for target "+target) {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestTestCommandJSONDiagnosticsForUnsupportedReportFormat(t *testing.T) {
	diag := runCLIJSONDiagnostic(t, []string{"test", "--diagnostics=json", "--report=yaml"}, 2)
	if diag.Code != "TETRA0001" || diag.Message != "unsupported --report format" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestTestCommandRunsTetraTests(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "test \"math\":\n    expect 40 + 2 == 42\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", mustHostTarget(t), srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "1/1 passed") {
		t.Fatalf("test stdout = %q", stdout.String())
	}
}

func TestTestCommandDiscoversCapsuleSourceRoots(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")
	writeCLIProjectFile(t, dir, "src/passes.t4", "test \"project ok\":\n    expect 40 + 2 == 42\n")
	writeCLIProjectFile(t, dir, "other/fails.t4", "test \"should not run\":\n    expect 1 == 2\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", mustHostTarget(t)}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "1/1 passed") || strings.Contains(stdout.String(), "should not run") {
		t.Fatalf("test stdout = %q", stdout.String())
	}
}

func TestTestCommandExplicitProjectDirectoryUsesSourceRootsAndImports(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/main.t4"
    sources:
        src
        tests
`)
	writeCLIProjectFile(t, dir, "src/main.t4", "func main() -> Int:\n    return 0\n")
	writeCLIProjectFile(t, dir, "src/app/util.t4", "module app.util\nfunc answer() -> Int:\n    return 42\n")
	writeCLIProjectFile(t, dir, "tests/util_test.t4", "module util_test\nimport app.util as util\ntest \"imports app util\":\n    expect util.answer() == 42\n")
	writeCLIProjectFile(t, dir, "other/fails.t4", "test \"should not run\":\n    expect 1 == 2\n")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", mustHostTarget(t), dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "1/1 passed") {
		t.Fatalf("test stdout = %q", stdout.String())
	}
}

func TestTestCommandRunsModuleFileWithImportsAndMain(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	srcPath := filepath.Join("..", "..", "..", "examples", "projects", "dogfood_cli", "src", "main.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing dogfood source %s: %v", srcPath, err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", mustHostTarget(t), srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "PASS cli status code") {
		t.Fatalf("test stdout = %q", stdout.String())
	}
}

func TestTestCommandJSONReport(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "test \"math\":\n    expect 40 + 2 == 42\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Total      int   `json:"total"`
		Passed     int   `json:"passed"`
		Failed     int   `json:"failed"`
		DurationMS int64 `json:"duration_ms"`
		Files      []struct {
			Filename   string `json:"filename"`
			Total      int    `json:"total"`
			Passed     int    `json:"passed"`
			Failed     int    `json:"failed"`
			DurationMS int64  `json:"duration_ms"`
		} `json:"files"`
		Results []struct {
			Name       string `json:"name"`
			Passed     bool   `json:"passed"`
			DurationMS int64  `json:"duration_ms"`
		} `json:"results"`
	}
	runCLIJSONStdout(t, []string{"test", "--target", mustHostTarget(t), "--report=json", srcPath}, 0, &report)
	if report.Total != 1 || report.Passed != 1 || report.Failed != 0 || len(report.Results) != 1 || report.Results[0].Name != "math" || !report.Results[0].Passed {
		t.Fatalf("report = %#v", report)
	}
	if report.DurationMS <= 0 || report.Results[0].DurationMS <= 0 {
		t.Fatalf("durations missing: %#v", report)
	}
	if len(report.Files) != 1 || report.Files[0].Filename != srcPath || report.Files[0].Total != 1 || report.Files[0].Passed != 1 || report.Files[0].Failed != 0 {
		t.Fatalf("file report = %#v", report.Files)
	}
	if report.Files[0].DurationMS != report.Results[0].DurationMS || report.DurationMS != report.Results[0].DurationMS {
		t.Fatalf("duration aggregation mismatch: %#v", report)
	}
}

func TestTestCommandJSONReportMultipleBlocks(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := `test "first":
    expect 1 + 1 == 2

test "second":
    expect 2 + 2 == 4
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Total   int `json:"total"`
		Passed  int `json:"passed"`
		Failed  int `json:"failed"`
		Results []struct {
			Name         string `json:"name"`
			Index        int    `json:"index"`
			FunctionName string `json:"function_name"`
			Passed       bool   `json:"passed"`
		} `json:"results"`
	}
	runCLIJSONStdout(t, []string{"test", "--target", mustHostTarget(t), "--report=json", srcPath}, 0, &report)
	if report.Total != 2 || report.Passed != 2 || report.Failed != 0 || len(report.Results) != 2 {
		t.Fatalf("report = %#v", report)
	}
	if report.Results[0].Name != "first" || report.Results[0].Index != 0 || report.Results[0].FunctionName != "__tetra_test_0_first" || !report.Results[0].Passed {
		t.Fatalf("first result = %#v", report.Results[0])
	}
	if report.Results[1].Name != "second" || report.Results[1].Index != 1 || report.Results[1].FunctionName != "__tetra_test_1_second" || !report.Results[1].Passed {
		t.Fatalf("second result = %#v", report.Results[1])
	}
}

func TestTestCommandReportsFailingExpectText(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "test \"bad math\":\n    expect 40 + 2 == 41\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", mustHostTarget(t), srcPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected failing test, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "FAIL bad math") || !strings.Contains(out, "exit code 1") || !strings.Contains(out, "0/1 passed") {
		t.Fatalf("test stdout = %q", out)
	}
}

func TestTestCommandJSONReportIncludesFailureError(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "test \"bad math\":\n    expect 40 + 2 == 41\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Total   int `json:"total"`
		Passed  int `json:"passed"`
		Failed  int `json:"failed"`
		Results []struct {
			Name     string `json:"name"`
			ExitCode int    `json:"exit_code"`
			Passed   bool   `json:"passed"`
			Error    string `json:"error"`
		} `json:"results"`
	}
	runCLIJSONStdout(t, []string{"test", "--target", mustHostTarget(t), "--report=json", srcPath}, 1, &report)
	if report.Total != 1 || report.Passed != 0 || report.Failed != 1 || len(report.Results) != 1 {
		t.Fatalf("report = %#v", report)
	}
	result := report.Results[0]
	if result.Name != "bad math" || result.Passed || result.ExitCode != 1 || result.Error != "exit code 1" {
		t.Fatalf("result = %#v", result)
	}
}

func TestTestCommandJSONReportUsesEmptyArraysWhenNoTestsExist(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "func main() -> Int:\n    return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Total   int               `json:"total"`
		Passed  int               `json:"passed"`
		Failed  int               `json:"failed"`
		Files   []json.RawMessage `json:"files"`
		Results []json.RawMessage `json:"results"`
	}
	rawReport := runCLIJSONStdout(t, []string{"test", "--target", mustHostTarget(t), "--report=json", srcPath}, 0, &report)
	if report.Total != 0 || report.Passed != 0 || report.Failed != 0 {
		t.Fatalf("report counts = %#v", report)
	}
	if report.Files == nil || len(report.Files) != 0 || report.Results == nil || len(report.Results) != 0 {
		t.Fatalf("empty arrays should be present, report = %#v\n%s", report, rawReport)
	}
}
