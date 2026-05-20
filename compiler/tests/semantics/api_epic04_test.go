package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestPublicCheckAPISingleSourcePositive(t *testing.T) {
	prog, err := compiler.Parse([]byte(`
fun main(): i32 {
  return 42
}
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if checked.MainName != "main" {
		t.Fatalf("main name = %q, want main", checked.MainName)
	}
}

func TestPublicCheckAPISingleSourceNegativeDiagnostic(t *testing.T) {
	prog, err := compiler.Parse([]byte(`
fun main(): i32 {
  let x: i32 = true
  return x
}
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSemantic {
		t.Fatalf("diagnostic code = %q, want %q", diag.Code, compiler.DiagnosticCodeSemantic)
	}
	if !strings.Contains(diag.Message, "type mismatch: expected 'i32', got 'bool'") {
		t.Fatalf("diagnostic message = %q", diag.Message)
	}
}

func TestPublicCheckAPICrossModuleWorldPositive(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/math.tetra": "module engine.math\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/main.tetra":    "module app.main\nimport engine.math as math\nfun main(): i32 {\n  return math.add_one(41)\n}\n",
	})
	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if checked.MainName != "app.main.main" {
		t.Fatalf("main name = %q, want app.main.main", checked.MainName)
	}
	if checked.FuncSigs["engine.math.add_one"].ReturnType != "i32" {
		t.Fatalf("unexpected imported signature: %#v", checked.FuncSigs["engine.math.add_one"])
	}
}

func TestPublicCheckAPIDisplayTextForBoundaryError(t *testing.T) {
	_, err := compiler.Check(nil)
	if err == nil {
		t.Fatalf("expected nil program boundary error")
	}
	if err.Error() != "no program provided" {
		t.Fatalf("error = %q, want no program provided", err.Error())
	}
}

func writeTestFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, contents := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile(%q): %v", path, err)
		}
	}
}
