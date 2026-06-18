package module

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModuleLoadWorldResolvesImportGraph(t *testing.T) {
	tmp := t.TempDir()
	writeModuleFiles(t, tmp, map[string]string{
		"engine/math.tetra": "module engine.math\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/main.tetra":    "module app.main\nimport engine.math as math\nfun main(): i32 {\n  return math.add_one(41)\n}\n",
	})

	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	if got := world.EntryModule; got != "app.main" {
		t.Fatalf("entry module = %q, want app.main", got)
	}
	if _, ok := world.ByModule["engine.math"]; !ok {
		t.Fatalf("engine.math module missing from world graph")
	}
}

func TestModuleLoadWorldResolvesT4ImportGraph(t *testing.T) {
	tmp := t.TempDir()
	writeModuleFiles(t, tmp, map[string]string{
		"engine/math.t4": "module engine.math\nfunc add_one(x: Int) -> Int:\n    return x + 1\n",
		"app/main.t4":    "module app.main\nimport engine.math as math\nfunc main() -> Int:\n    return math.add_one(41)\n",
	})

	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	if got := world.EntryModule; got != "app.main" {
		t.Fatalf("entry module = %q, want app.main", got)
	}
	if got := filepath.Base(world.ByModule["engine.math"].Path); got != "math.t4" {
		t.Fatalf("engine.math path = %q, want math.t4", got)
	}
}

func TestModuleLoadWorldPrefersT4OverLegacyTetraImport(t *testing.T) {
	tmp := t.TempDir()
	writeModuleFiles(t, tmp, map[string]string{
		"engine/math.t4":    "module engine.math\nfunc add_one(x: Int) -> Int:\n    return x + 1\n",
		"engine/math.tetra": "module engine.math\nfunc add_one(x: Int) -> Int:\n    return x + 100\n",
		"app/main.t4":       "module app.main\nimport engine.math as math\nfunc main() -> Int:\n    return math.add_one(41)\n",
	})

	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	if got := filepath.Ext(world.ByModule["engine.math"].Path); got != ".t4" {
		t.Fatalf("engine.math extension = %q, want .t4", got)
	}
}

func TestModuleLoadWorldMergesSameModuleFragments(t *testing.T) {
	tmp := t.TempDir()
	writeModuleFiles(t, tmp, map[string]string{
		"engine/math.t4":             "module engine.math\nfunc base() -> Int:\n    return 40\n",
		"engine/math.parts/extra.t4": "module engine.math\nfunc add_two(x: Int) -> Int:\n    return x + 2\n",
		"app/main.t4":                "module app.main\nimport engine.math as math\nfunc main() -> Int:\n    return math.add_two(math.base())\n",
	})

	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	file := world.ByModule["engine.math"]
	if file == nil {
		t.Fatalf("engine.math module missing")
	}
	if got := len(file.Funcs); got != 2 {
		t.Fatalf("engine.math funcs = %d, want 2", got)
	}
	if got := filepath.Base(file.Path); got != "math.t4" {
		t.Fatalf("engine.math primary path = %q, want math.t4", got)
	}
	if src := string(file.Src); !strings.Contains(src, "func base") ||
		!strings.Contains(src, "func add_two") {
		t.Fatalf("merged source missing primary or fragment function:\n%s", src)
	}
}

func TestModuleLoadWorldWithSourceRootsResolvesImportsAcrossProjectRoots(t *testing.T) {
	tmp := t.TempDir()
	writeModuleFiles(t, tmp, map[string]string{
		"src/app/main.t4":          "module app.main\nimport components.counter as counter\nfunc main() -> Int:\n    return counter.value()\n",
		"ui/components/counter.t4": "module components.counter\nfunc value() -> Int:\n    return 42\n",
	})

	world, err := LoadWorldOpt(
		filepath.Join(tmp, filepath.FromSlash("src/app/main.t4")),
		LoadOptions{
			Root:        tmp,
			SourceRoots: []string{"src", "ui"},
		},
	)
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	file := world.ByModule["components.counter"]
	if file == nil {
		t.Fatalf("components.counter module missing")
	}
	if got := filepath.ToSlash(file.Path); !strings.HasSuffix(got, "ui/components/counter.t4") {
		t.Fatalf("components.counter path = %q, want ui/components/counter.t4", got)
	}
}

func TestModuleLoadWorldRejectsDuplicateModuleAcrossSourceRoots(t *testing.T) {
	tmp := t.TempDir()
	writeModuleFiles(t, tmp, map[string]string{
		"src/app/main.t4":     "module app.main\nimport shared.util as util\nfunc main() -> Int:\n    return util.value()\n",
		"src/shared/util.t4":  "module shared.util\nfunc value() -> Int:\n    return 1\n",
		"lib/shared/util.t4":  "module shared.util\nfunc value() -> Int:\n    return 2\n",
		"test/shared/util.t4": "module shared.util\nfunc value() -> Int:\n    return 3\n",
	})

	_, err := LoadWorldOpt(filepath.Join(tmp, filepath.FromSlash("src/app/main.t4")), LoadOptions{
		Root:        tmp,
		SourceRoots: []string{"src", "lib", "test"},
	})
	if err == nil {
		t.Fatalf("expected duplicate module diagnostic")
	}
	if !strings.Contains(err.Error(), "duplicate module 'shared.util'") {
		t.Fatalf("error = %v", err)
	}
}

func TestModuleLoadWorldWithDependencyRootsResolvesCrossCapsuleImport(t *testing.T) {
	tmp := t.TempDir()
	appRoot := filepath.Join(tmp, "App")
	mathRoot := filepath.Join(tmp, "Math")
	writeModuleFiles(t, appRoot, map[string]string{
		"src/app/main.t4": "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n",
	})
	writeModuleFiles(t, mathRoot, map[string]string{
		"src/math/core.t4": "module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n",
	})

	world, err := LoadWorldOpt(
		filepath.Join(appRoot, filepath.FromSlash("src/app/main.t4")),
		LoadOptions{
			Root:        appRoot,
			SourceRoots: []string{"src"},
			DependencyRoots: []ModuleRoot{{
				Root:        mathRoot,
				SourceRoots: []string{"src"},
			}},
		},
	)
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	file := world.ByModule["math.core"]
	if file == nil {
		t.Fatalf("math.core module missing")
	}
	if got := filepath.ToSlash(file.Path); !strings.HasSuffix(got, "Math/src/math/core.t4") {
		t.Fatalf("math.core path = %q, want dependency source", got)
	}
}

func TestModuleLoadWorldFallsBackToT4InterfaceForDependencyImport(t *testing.T) {
	tmp := t.TempDir()
	appRoot := filepath.Join(tmp, "App")
	mathRoot := filepath.Join(tmp, "Math")
	writeModuleFiles(t, appRoot, map[string]string{
		"src/app/main.t4": "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n",
	})
	writeModuleFiles(t, mathRoot, map[string]string{
		"interfaces/math/core.t4i": hashedT4I(
			"module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return 0\n",
		),
	})

	world, err := LoadWorldOpt(
		filepath.Join(appRoot, filepath.FromSlash("src/app/main.t4")),
		LoadOptions{
			Root:        appRoot,
			SourceRoots: []string{"src"},
			DependencyRoots: []ModuleRoot{{
				Root:        mathRoot,
				SourceRoots: []string{"src", "interfaces"},
			}},
		},
	)
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	if !world.InterfaceModules["math.core"] {
		t.Fatalf("math.core should be marked as interface-only")
	}
	if world.InterfaceHashes["math.core"] == "" {
		t.Fatalf("math.core should record interface hash")
	}
}

func TestModuleLoadWorldRejectsTamperedT4InterfaceHash(t *testing.T) {
	tmp := t.TempDir()
	writeModuleFiles(t, tmp, map[string]string{
		"app/main.t4":   "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n",
		"math/core.t4i": "// t4i-hash: sha256:0000000000000000000000000000000000000000000000000000000000000000\nmodule math.core\nfunc add(a: Int, b: Int) -> Int:\n    return 0\n",
	})

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err == nil {
		t.Fatalf("expected invalid interface hash diagnostic")
	}
	if !strings.Contains(err.Error(), "invalid .t4i hash") {
		t.Fatalf("error = %v", err)
	}
}

func TestModuleLoadWorldDiagnosticForModuleDeclarationMismatch(t *testing.T) {
	tmp := t.TempDir()
	writeModuleFiles(t, tmp, map[string]string{
		"engine/math.tetra": "module engine.wrong\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/main.tetra":    "module app.main\nimport engine.math as math\nfun main(): i32 {\n  return math.add_one(41)\n}\n",
	})

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err == nil {
		t.Fatalf("expected module declaration mismatch error")
	}
	if !strings.Contains(
		err.Error(),
		"module declaration 'engine.wrong' does not match import 'engine.math'",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestModuleLoadWorldDiagnosticForDuplicateImportPath(t *testing.T) {
	tmp := t.TempDir()
	writeModuleFiles(t, tmp, map[string]string{
		"engine/math.tetra": "module engine.math\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/main.tetra":    "module app.main\nimport engine.math as math\nimport engine.math as m\nfun main(): i32 {\n  return math.add_one(41)\n}\n",
	})

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err == nil {
		t.Fatalf("expected duplicate import error")
	}
	if !strings.Contains(err.Error(), "duplicate import 'engine.math'") {
		t.Fatalf("error = %v", err)
	}
}

func TestModuleLoadWorldDiagnosticForImportCycle(t *testing.T) {
	tmp := t.TempDir()
	writeModuleFiles(t, tmp, map[string]string{
		"app/main.tetra": "module app.main\nimport mod.a as a\nfun main(): i32 {\n  return a.ping()\n}\n",
		"mod/a.tetra":    "module mod.a\nimport mod.b as b\nfun ping(): i32 {\n  return b.pong()\n}\n",
		"mod/b.tetra":    "module mod.b\nimport mod.a as a\nfun pong(): i32 {\n  return 1\n}\n",
	})

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	if !strings.Contains(err.Error(), "import cycle detected at 'mod.a'") {
		t.Fatalf("error = %v", err)
	}
}

func TestModuleRootFromEntryBoundaryPathMustMatchModule(t *testing.T) {
	tmp := t.TempDir()
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))
	root, err := rootFromEntry(entry, "app.wrong")
	if err == nil {
		t.Fatalf("expected rootFromEntry mismatch error, got root=%q", root)
	}
	if !strings.Contains(err.Error(), "module 'app.wrong' must be in app/wrong.t4") {
		t.Fatalf("error = %v", err)
	}
}

func writeModuleFiles(t *testing.T, base string, files map[string]string) {
	t.Helper()
	for rel, src := range files {
		full := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", full, err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
}

func hashedT4I(body string) string {
	sum := sha256.Sum256([]byte(body))
	return "// t4i-hash: sha256:" + hex.EncodeToString(sum[:]) + "\n" + body
}
