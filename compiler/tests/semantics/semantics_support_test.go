package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/testkit"
)

// ---- build_options_helpers_test.go ----

func buildAndRunWithOptions(t *testing.T, src string, opt compiler.BuildOptions) (string, int) {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")

	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", opt); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func buildAndRunFile(t *testing.T, srcPath string) (string, int) {
	t.Helper()
	outPath := filepath.Join(t.TempDir(), "app")
	if err := compiler.BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func buildOnlyFiles(t *testing.T, files map[string]string, entry string) error {
	t.Helper()
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, files)
	entryPath := filepath.Join(tmp, filepath.FromSlash(entry))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return compiler.BuildFile(entryPath, outPath, "linux-x64")
}

// ---- ir_helpers_test.go ----

func findIRFunc(t *testing.T, funcs []compiler.IRFunc, name string) compiler.IRFunc {
	t.Helper()
	for _, fn := range funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("missing IR function %q", name)
	return compiler.IRFunc{}
}

func hasIRCall(fn compiler.IRFunc, name string) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == name {
			return true
		}
	}
	return false
}

// ---- world_test_helpers_test.go ----

func requireCheckWorldFilesErrorContains(
	t *testing.T,
	files map[string]string,
	entry string,
	want string,
) {
	t.Helper()

	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, files)

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash(entry)))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}
