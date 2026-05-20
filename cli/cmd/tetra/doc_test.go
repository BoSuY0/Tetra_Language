package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocCommandWritesAPIDocsToStdout(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "api.tetra")
	if err := os.WriteFile(srcPath, []byte("func answer() -> Int:\n    return 42\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"doc", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doc exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "# Tetra API Docs") || !strings.Contains(stdout.String(), "`func answer() -> i32`") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestDocCommandDiscoversCapsuleProjectSources(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/app/main.t4", "module app.main\nfunc answer() -> Int:\n    return 42\n")

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
	code := runCLI([]string{"doc"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doc exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "## app.main") || !strings.Contains(stdout.String(), "`func answer() -> i32`") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestDocCommandWritesAPIDocsToFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "api.tetra")
	outPath := filepath.Join(dir, "docs", "api.md")
	if err := os.WriteFile(srcPath, []byte("func answer() -> Int:\n    return 42\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"doc", "-o", outPath, srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doc exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read docs: %v", err)
	}
	if !strings.Contains(string(raw), "`func answer() -> i32`") {
		t.Fatalf("docs = %s", raw)
	}
}

func TestDocCommandGeneratedOutputPassesAPIValidator(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "api.tetra")
	outPath := filepath.Join(dir, "api.md")
	src := `module docs.api

func answer() -> Int:
    return 42

test "answer":
    expect answer() == 42
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"doc", "-o", outPath, srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doc exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	cmd := exec.Command("go", "run", "./tools/cmd/validate-api-docs", "--docs", outPath)
	cmd.Dir = filepath.Join("..", "..", "..")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("validate-api-docs failed: %v\n%s", err, out)
	}
}

func TestDocCommandJSONDiagnostics(t *testing.T) {
	diag := runCLIJSONDiagnostic(t, []string{"doc", "--diagnostics=json", "/tmp/does-not-exist.tetra"}, 1)
	if diag.Code != "TETRA0001" || diag.Severity != "error" || !strings.Contains(diag.Message, "no such file or directory") {
		t.Fatalf("diagnostic = %#v", diag)
	}
}
