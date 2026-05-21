package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

func TestTestCommandJSONDiagnosticsForBuildOnlyRuntimeUnsupported(t *testing.T) {
	restore := stubLinuxX32HostSupport(false)
	defer restore()

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	if err := os.WriteFile(srcPath, []byte("test \"math\":\n    expect 40 + 2 == 42\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"test", "--diagnostics=json", "--target", "x32", srcPath}, 2)
	for _, want := range []string{"cannot run tests for target linux-x32", "host does not support Linux x32 ABI execution", "no host fallback"} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
}

func TestTestCommandRunsLinuxX32SourceTestsWhenProbePasses(t *testing.T) {
	restoreHost := stubLinuxX32HostSupport(true)
	defer restoreHost()
	restoreExec := stubNativeExec(func(path string, stdout io.Writer, stderr io.Writer) int {
		if err := requireX32ExecutableFile(path); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	})
	defer restoreExec()

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	if err := os.WriteFile(srcPath, []byte("test \"math\":\n    expect 40 + 2 == 42\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x32", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("test stderr = %q", stderr.String())
	}
	for _, want := range []string{"PASS math", "Tetra tests: 1/1 passed"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("test stdout missing %q: %q", want, stdout.String())
		}
	}
}

func TestTestCommandRunsDefaultTargetSuitesWithoutProject(t *testing.T) {
	for _, tc := range []struct {
		target string
		want   []string
	}{
		{
			target: "x86",
			want: []string{
				"PASS x86 target model",
				"PASS x86 i386 SysV classifier",
				"PASS x86 varargs and sret ABI",
				"PASS x86 pointer/native-libc FFI diagnostics",
				"PASS x86 source native scalar diagnostics",
				"PASS x86 stdlib runtime boundary diagnostics",
				"PASS x86 target runtime boundary diagnostics",
				"PASS x86 pointer atomic ABI width",
				"PASS x86 object ABI smoke",
				"PASS x86 atomic ABI object",
				"PASS x86 executable matrix smoke",
				"Tetra tests: 11/11 passed",
			},
		},
		{
			target: "x64",
			want: []string{
				"PASS x64 target model",
				"PASS x64 SysV classifier",
				"PASS x64 SysV varargs and aggregates",
				"PASS x64 source native scalar diagnostics",
				"PASS x64 pointer atomic ABI width",
				"PASS x64 object ABI smoke",
				"PASS x64 atomic ABI object",
				"PASS x64 executable matrix smoke",
				"Tetra tests: 8/8 passed",
			},
		},
		{
			target: "x32",
			want: []string{
				"PASS x32 target model",
				"PASS x32 SysV classifier",
				"PASS x32 SysV varargs and aggregates",
				"PASS x32 pointer/native-libc FFI diagnostics",
				"PASS x32 source native scalar diagnostics",
				"PASS x32 stdlib runtime boundary diagnostics",
				"PASS x32 target runtime boundary diagnostics",
				"PASS x32 pointer atomic ABI width",
				"PASS x32 object ABI smoke",
				"PASS x32 atomic ABI object",
				"PASS x32 executable matrix smoke",
				"Tetra tests: 11/11 passed",
			},
		},
	} {
		t.Run(tc.target, func(t *testing.T) {
			dir := t.TempDir()
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
			code := runCLI([]string{"test", "--target", tc.target}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("test stderr = %q", stderr.String())
			}
			out := stdout.String()
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Fatalf("test stdout missing %q: %q", want, out)
				}
			}
		})
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

func TestTestCommandRunsAllTargetsBrutalSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--all-targets", "--brutal"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("test stderr = %q", stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x86 target model",
		"PASS x86 pointer/native-libc FFI diagnostics",
		"PASS x86 source native scalar diagnostics",
		"PASS x86 stdlib runtime boundary diagnostics",
		"PASS x86 target runtime boundary diagnostics",
		"PASS x86 pointer atomic ABI width",
		"PASS x64 atomic object matrix",
		"PASS x64 pointer atomic object width",
		"PASS x64 source native scalar diagnostics",
		"PASS x64 pointer atomic ABI width",
		"PASS x32 layout fuzz",
		"PASS x64 layout fuzz",
		"PASS x64 object signature fuzz",
		"PASS x86 atomic validation matrix",
		"PASS x86 atomic object matrix",
		"PASS x86 pointer atomic object width",
		"PASS x86 layout fuzz",
		"PASS x86 object signature fuzz",
		"PASS x32 SysV classifier",
		"PASS x32 SysV varargs and aggregates",
		"PASS x32 pointer/native-libc FFI diagnostics",
		"PASS x32 source native scalar diagnostics",
		"PASS x32 stdlib runtime boundary diagnostics",
		"PASS x32 target runtime boundary diagnostics",
		"PASS x32 pointer atomic ABI width",
		"PASS x32 pointer atomic object width",
		"PASS macos-x64 SysV classifier",
		"PASS macos-x64 object ABI smoke",
		"PASS macos-x64 source native scalar diagnostics",
		"PASS macos-x64 pointer atomic ABI width",
		"PASS windows-x64 Win64 classifier",
		"PASS windows-x64 Win64 varargs and aggregates",
		"PASS windows-x64 object ABI smoke",
		"PASS windows-x64 source native scalar diagnostics",
		"PASS windows-x64 pointer atomic ABI width",
		"PASS macos-x64 atomic object matrix",
		"PASS macos-x64 pointer atomic object width",
		"PASS windows-x64 atomic object matrix",
		"PASS windows-x64 pointer atomic object width",
		"PASS x32 atomic concurrency stress oracle",
		"PASS macos-x64 object signature fuzz",
		"PASS windows-x64 object signature fuzz",
		"Tetra tests: 82/82 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
	if strings.Contains(out, "FAIL x64 fuzz") {
		t.Fatalf("test stdout still reports x64 fuzz as unsupported: %q", out)
	}
}

func TestTestCommandAllTargetsBrutalJSONUsesTargetSpecificFiles(t *testing.T) {
	var report struct {
		Total  int `json:"total"`
		Passed int `json:"passed"`
		Failed int `json:"failed"`
		Files  []struct {
			Filename string `json:"filename"`
		} `json:"files"`
		Results []struct {
			Name         string `json:"name"`
			Filename     string `json:"filename"`
			Index        int    `json:"index"`
			FunctionName string `json:"function_name"`
			Passed       bool   `json:"passed"`
		} `json:"results"`
	}
	runCLIJSONStdout(t, []string{"test", "--all-targets", "--brutal", "--report=json"}, 0, &report)
	if report.Total != 82 || report.Passed != 82 || report.Failed != 0 || len(report.Results) != 82 {
		t.Fatalf("report = %#v", report)
	}
	files := map[string]bool{}
	for _, file := range report.Files {
		files[file.Filename] = true
	}
	for _, want := range []string{
		"tetra:x64-abi",
		"tetra:macos-x64-abi",
		"tetra:windows-x64-abi",
		"tetra:x64-atomic-stress",
		"tetra:macos-x64-atomic-stress",
		"tetra:windows-x64-atomic-stress",
		"tetra:x64-fuzz",
		"tetra:macos-x64-fuzz",
		"tetra:windows-x64-fuzz",
	} {
		if !files[want] {
			t.Fatalf("report files missing %q: %#v", want, report.Files)
		}
	}
	wantFilenameByName := map[string]string{
		"x64 SysV classifier":                          "tetra:x64-abi",
		"x64 pointer atomic ABI width":                 "tetra:x64-abi",
		"macos-x64 SysV classifier":                    "tetra:macos-x64-abi",
		"macos-x64 object ABI smoke":                   "tetra:macos-x64-abi",
		"macos-x64 pointer atomic ABI width":           "tetra:macos-x64-abi",
		"windows-x64 Win64 classifier":                 "tetra:windows-x64-abi",
		"windows-x64 object ABI smoke":                 "tetra:windows-x64-abi",
		"windows-x64 pointer atomic ABI width":         "tetra:windows-x64-abi",
		"x64 atomic object matrix":                     "tetra:x64-atomic-stress",
		"x64 pointer atomic object width":              "tetra:x64-atomic-stress",
		"x64 atomic concurrency stress oracle":         "tetra:x64-atomic-stress",
		"macos-x64 atomic object matrix":               "tetra:macos-x64-atomic-stress",
		"macos-x64 pointer atomic object width":        "tetra:macos-x64-atomic-stress",
		"macos-x64 atomic concurrency stress oracle":   "tetra:macos-x64-atomic-stress",
		"windows-x64 atomic object matrix":             "tetra:windows-x64-atomic-stress",
		"windows-x64 pointer atomic object width":      "tetra:windows-x64-atomic-stress",
		"windows-x64 atomic concurrency stress oracle": "tetra:windows-x64-atomic-stress",
		"x64 object signature fuzz":                    "tetra:x64-fuzz",
		"macos-x64 object signature fuzz":              "tetra:macos-x64-fuzz",
		"windows-x64 object signature fuzz":            "tetra:windows-x64-fuzz",
	}
	for name, wantFile := range wantFilenameByName {
		found := false
		for _, result := range report.Results {
			if result.Name == name {
				found = true
				if result.Filename != wantFile || !result.Passed {
					t.Fatalf("result %q = %#v, want filename %q and passed", name, result, wantFile)
				}
				if !strings.HasPrefix(result.FunctionName, "__tetra_test_") {
					t.Fatalf("result %q function_name = %q, want __tetra_test_ prefix", name, result.FunctionName)
				}
			}
		}
		if !found {
			t.Fatalf("report missing result %q: %#v", name, report.Results)
		}
	}
	prevOrderKey := ""
	for _, result := range report.Results {
		orderKey := fmt.Sprintf("%s\x00%08d", result.Filename, result.Index)
		if prevOrderKey != "" && orderKey < prevOrderKey {
			t.Fatalf("results are not sorted by filename then index: previous=%q current=%q", prevOrderKey, orderKey)
		}
		prevOrderKey = orderKey
	}
}

func TestTestCommandJSONDiagnosticsForTargetSpecificSuiteUnsupported(t *testing.T) {
	diag := runCLIJSONDiagnostic(t, []string{"test", "--diagnostics=json", "--target", "x32", "--abi", "--atomic-stress"}, 2)
	for _, want := range []string{"--abi", "--atomic-stress", "linux-x32", "ABI torture", "atomic stress", "not implemented yet", "no fake or skipped tests"} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
}

func TestTestCommandRunsX32FuzzSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x32", "--fuzz"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x32 layout fuzz",
		"PASS x32 object signature fuzz",
		"PASS x32 target alias fuzz",
		"Tetra tests: 3/3 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX64FuzzSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x64", "--fuzz"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x64 layout fuzz",
		"PASS x64 object signature fuzz",
		"PASS x64 target alias fuzz",
		"Tetra tests: 3/3 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX86FuzzSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x86", "--fuzz"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x86 layout fuzz",
		"PASS x86 object signature fuzz",
		"PASS x86 target alias fuzz",
		"Tetra tests: 3/3 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX32AtomicStressSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x32", "--atomic-stress"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x32 atomic validation matrix",
		"PASS x32 atomic object matrix",
		"PASS x32 pointer atomic object width",
		"PASS x32 atomic concurrency stress oracle",
		"PASS x32 atomic diagnostics",
		"Tetra tests: 5/5 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX64AtomicStressSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x64", "--atomic-stress"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x64 atomic validation matrix",
		"PASS x64 atomic object matrix",
		"PASS x64 pointer atomic object width",
		"PASS x64 atomic concurrency stress oracle",
		"PASS x64 atomic diagnostics",
		"Tetra tests: 5/5 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX86AtomicStressSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x86", "--atomic-stress"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x86 atomic validation matrix",
		"PASS x86 atomic object matrix",
		"PASS x86 pointer atomic object width",
		"PASS x86 atomic concurrency stress oracle",
		"PASS x86 atomic diagnostics",
		"Tetra tests: 5/5 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX32ABISuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x32", "--abi"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x32 target model",
		"PASS x32 SysV classifier",
		"PASS x32 SysV varargs and aggregates",
		"PASS x32 pointer/native-libc FFI diagnostics",
		"PASS x32 source native scalar diagnostics",
		"PASS x32 stdlib runtime boundary diagnostics",
		"PASS x32 target runtime boundary diagnostics",
		"PASS x32 pointer atomic ABI width",
		"PASS x32 object ABI smoke",
		"PASS x32 atomic ABI object",
		"PASS x32 executable matrix smoke",
		"Tetra tests: 11/11 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX86ABISuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x86", "--abi"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x86 target model",
		"PASS x86 i386 SysV classifier",
		"PASS x86 varargs and sret ABI",
		"PASS x86 pointer/native-libc FFI diagnostics",
		"PASS x86 source native scalar diagnostics",
		"PASS x86 stdlib runtime boundary diagnostics",
		"PASS x86 target runtime boundary diagnostics",
		"PASS x86 pointer atomic ABI width",
		"PASS x86 object ABI smoke",
		"PASS x86 atomic ABI object",
		"PASS x86 executable matrix smoke",
		"Tetra tests: 11/11 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsLinuxX86SourceTestsWhenKernelSupports(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sample.tetra")
	src := "func add(a: Int, b: Int) -> Int:\n    return a + b\n\ntest \"math\":\n    expect add(40, 2) == 42\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("test stderr = %q", stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"PASS math", "Tetra tests: 1/1 passed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandRunsX64ABISuite(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"test", "--target", "x64", "--abi"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"PASS x64 target model",
		"PASS x64 SysV classifier",
		"PASS x64 SysV varargs and aggregates",
		"PASS x64 source native scalar diagnostics",
		"PASS x64 pointer atomic ABI width",
		"PASS x64 object ABI smoke",
		"PASS x64 atomic ABI object",
		"PASS x64 executable matrix smoke",
		"Tetra tests: 8/8 passed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("test stdout missing %q: %q", want, out)
		}
	}
}

func TestTestCommandX32ABISuiteJSONReport(t *testing.T) {
	var report struct {
		Total  int `json:"total"`
		Passed int `json:"passed"`
		Failed int `json:"failed"`
		Files  []struct {
			Filename string `json:"filename"`
			Total    int    `json:"total"`
			Passed   int    `json:"passed"`
			Failed   int    `json:"failed"`
		} `json:"files"`
		Results []struct {
			Name     string `json:"name"`
			Filename string `json:"filename"`
			Passed   bool   `json:"passed"`
		} `json:"results"`
	}
	runCLIJSONStdout(t, []string{"test", "--target", "x32", "--abi", "--report=json"}, 0, &report)
	if report.Total != 11 || report.Passed != 11 || report.Failed != 0 || len(report.Results) != 11 {
		t.Fatalf("report = %#v", report)
	}
	if len(report.Files) != 1 || report.Files[0].Filename != "tetra:x32-abi" || report.Files[0].Total != 11 || report.Files[0].Passed != 11 || report.Files[0].Failed != 0 {
		t.Fatalf("files = %#v", report.Files)
	}
	wantNames := []string{"x32 target model", "x32 SysV classifier", "x32 SysV varargs and aggregates", "x32 pointer/native-libc FFI diagnostics", "x32 source native scalar diagnostics", "x32 stdlib runtime boundary diagnostics", "x32 target runtime boundary diagnostics", "x32 pointer atomic ABI width", "x32 object ABI smoke", "x32 atomic ABI object", "x32 executable matrix smoke"}
	for i, want := range wantNames {
		if report.Results[i].Name != want || report.Results[i].Filename != "tetra:x32-abi" || !report.Results[i].Passed {
			t.Fatalf("result[%d] = %#v", i, report.Results[i])
		}
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
