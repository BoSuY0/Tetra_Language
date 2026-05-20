package main

import (
	"bytes"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ctarget "tetra_language/compiler/target"
)

func TestRunCommandJSONDiagnosticsForHostTargetMismatch(t *testing.T) {
	target := nonHostTarget(t)
	diag := runCLIJSONDiagnostic(t, []string{"run", "--diagnostics=json", "--target", target}, 2)
	if diag.Code != "TETRA0001" || diag.Severity != "error" || !strings.Contains(diag.Message, "cannot run target "+target) {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestRunCommandJSONDiagnosticsForWASMWebRuntimeUnsupported(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	restore := stubLookPath(func(name string) (string, error) {
		return "", exec.ErrNotFound
	})
	defer restore()

	diag := runCLIJSONDiagnostic(t, []string{"run", "--diagnostics=json", "--target", "wasm32-web", srcPath}, 1)
	for _, want := range []string{"cannot run target wasm32-web", "missing web runtime runner"} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
}

func TestExecWebProgramWithBrowserRunnerParsesBrowserExitResult(t *testing.T) {
	requireLocalTCPBind(t)

	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "app.wasm")
	if err := os.WriteFile(wasmPath, []byte("\x00asm\x01\x00\x00\x00"), 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.mjs"), []byte("export async function runTetra() { return 7; }\n"), 0o644); err != nil {
		t.Fatalf("write loader: %v", err)
	}
	browser := filepath.Join(dir, "fake-chromium")
	if err := os.WriteFile(browser, []byte(`#!/bin/sh
printf '<html><body><pre id="result">exit:7</pre></body></html>\n'
`), 0o755); err != nil {
		t.Fatalf("write fake browser: %v", err)
	}

	exit, err := execWebProgramWithBrowserRunner(wasmPath, browser, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("execWebProgramWithBrowserRunner: %v", err)
	}
	if exit != 7 {
		t.Fatalf("exit = %d, want 7", exit)
	}
}

func requireLocalTCPBind(t *testing.T) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local TCP bind unavailable in this environment: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close local TCP probe: %v", err)
	}
}

func TestRunCommandPropagatesProgramExitCode(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", mustHostTarget(t), srcPath}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunCommandWithoutOutputDoesNotLeaveDefaultBinary(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
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
	code := runCLI([]string{"run", "--target", mustHostTarget(t), srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	tgt, err := ctarget.Parse(mustHostTarget(t))
	if err != nil {
		t.Fatal(err)
	}
	defaultPath := filepath.Join(dir, defaultOutput(tgt, "exe"))
	if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
		t.Fatalf("run without -o should not leave %s, stat err=%v", defaultPath, err)
	}
}
