package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInterfaceCommandWritesT4IFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "math.t4")
	outPath := filepath.Join(dir, "math.t4i")
	if err := os.WriteFile(srcPath, []byte("module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"interface", "-o", outPath, srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("interface exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read interface: %v", err)
	}
	if !strings.Contains(string(raw), "func add(a: i32, b: i32) -> i32:") {
		t.Fatalf("interface output = %s", raw)
	}
}

func TestInterfaceCommandCheckReportsStalePublicAPI(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, filepath.FromSlash("math/core.t4"))
	writeCLIProjectFile(t, dir, "math/core.t4", `module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	outPath := filepath.Join(dir, "math.t4i")
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"interface", "-o", outPath, srcPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("interface write exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	writeCLIProjectFile(t, dir, "math/core.t4", `module math.core

pub func add(a: Int, b: Bool) -> Int:
    return a
`)

	stdout.Reset()
	stderr.Reset()
	code := runCLI([]string{"interface", "--check", "-o", outPath, srcPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected stale interface check failure, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "public API mismatch") {
		t.Fatalf("stderr = %q, want public API mismatch", stderr.String())
	}
}

func TestCheckCommandInterfaceOnlyDoesNotRequireMain(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, filepath.FromSlash("math/core.t4"))
	writeCLIProjectFile(t, dir, "math/core.t4", `module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check", "--interface-only", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("check --interface-only exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestBuildCommandInterfaceOnlyDoesNotRequireMain(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, filepath.FromSlash("math/core.t4"))
	writeCLIProjectFile(t, dir, "math/core.t4", `module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)

	outPath := filepath.Join(dir, "out", "app")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"build", "--interface-only", "--target", "linux-x64", "-o", outPath, srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("build --interface-only exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("build --interface-only should not emit %s, stat err=%v", outPath, err)
	}
	if !strings.Contains(stdout.String(), "Interface-only build checked") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
