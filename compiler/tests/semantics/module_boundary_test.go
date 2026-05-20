package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestModuleBoundaryAllowsPublicImportedFunction(t *testing.T) {
	tmp := t.TempDir()
	writeCompilerModuleFiles(t, tmp, map[string]string{
		"engine/math.t4": `module engine.math
pub func add(a: Int, b: Int) -> Int:
    return a + b
func hidden() -> Int:
    return 99
`,
		"app/main.t4": `module app.main
import engine.math as math
func main() -> Int:
    return math.add(40, 2)
`,
	})

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
}

func TestModuleBoundaryRejectsPrivateImportedFunction(t *testing.T) {
	tmp := t.TempDir()
	writeCompilerModuleFiles(t, tmp, map[string]string{
		"engine/math.t4": `module engine.math
pub func add(a: Int, b: Int) -> Int:
    return a + b
func hidden() -> Int:
    return 99
`,
		"app/main.t4": `module app.main
import engine.math as math
func main() -> Int:
    return math.hidden()
`,
	})

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected private function diagnostic")
	}
	if !strings.Contains(err.Error(), "private function 'engine.math.hidden'") {
		t.Fatalf("error = %v", err)
	}
}

func TestSelectiveImportResolvesPublicFunctionAndType(t *testing.T) {
	tmp := t.TempDir()
	writeCompilerModuleFiles(t, tmp, map[string]string{
		"engine/math.t4": `module engine.math
pub struct Vec { x: Int }
pub func add(a: Int, b: Int) -> Int:
    return a + b
`,
		"app/main.t4": `module app.main
import engine.math.{add, Vec}
struct Holder { value: Vec }
func main() -> Int:
    return add(40, 2)
`,
	})

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if got := checked.FuncSigs["app.main.main"].ReturnType; got != "i32" {
		t.Fatalf("main return = %q, want i32", got)
	}
	if _, ok := checked.Types["engine.math.Vec"]; !ok {
		t.Fatalf("missing selected imported type engine.math.Vec")
	}
}

func TestSelectiveImportRejectsDuplicateImportedSymbol(t *testing.T) {
	tmp := t.TempDir()
	writeCompilerModuleFiles(t, tmp, map[string]string{
		"a/one.t4": `module a.one
pub func pick() -> Int:
    return 1
`,
		"b/two.t4": `module b.two
pub func pick() -> Int:
    return 2
`,
		"app/main.t4": `module app.main
import a.one.{pick}
import b.two.{pick}
func main() -> Int:
    return pick()
`,
	})

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected duplicate selective import diagnostic")
	}
	if !strings.Contains(err.Error(), "duplicate import alias 'pick'") {
		t.Fatalf("error = %v", err)
	}
}

func TestPublicReExportSupportsSelectiveImport(t *testing.T) {
	tmp := t.TempDir()
	writeCompilerModuleFiles(t, tmp, map[string]string{
		"math/core.t4": `module math.core
pub func add(a: Int, b: Int) -> Int:
    return a + b
`,
		"math/prelude.t4": `module math.prelude
pub import math.core.{add}
`,
		"app/main.t4": `module app.main
import math.prelude.{add}
func main() -> Int:
    return add(40, 2)
`,
	})

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
}

func writeCompilerModuleFiles(t *testing.T, base string, files map[string]string) {
	t.Helper()
	for rel, src := range files {
		path := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
}
