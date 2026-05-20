package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/internal/testkit"
	"tetra_language/compiler/target"
)

func TestLinkObjectTargetMismatch(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	other := "windows-x64"
	if tgt.Triple == "windows-x64" {
		other = "linux-x64"
	}

	tmp := t.TempDir()
	objPath := filepath.Join(tmp, "lib.tobj")
	if err := WriteObject(objPath, &Object{
		Target:  other,
		Module:  "__testlib",
		Code:    []byte{0xC3},
		Symbols: []Symbol{{Name: "__testlib", Offset: 0}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}

	outPath := filepath.Join(tmp, "app"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(filepath.Join("..", "examples", "hello.tetra"), outPath, tgt.Triple, BuildOptions{
		LinkObjectPaths: []string{objPath},
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "link object target mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLinksInterfaceDependencyWithMatchingImplementationObject(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	libSrc := filepath.Join(tmp, filepath.FromSlash("lib/math/core.t4"))
	libObj := filepath.Join(tmp, "math.tobj")
	if err := writeFile(libSrc, string(src)); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildFileWithStatsOpt(libSrc, libObj, tgt.Triple, BuildOptions{Emit: EmitLibrary}); err != nil {
		t.Fatalf("build library: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(src, libSrc)
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(tmp, filepath.FromSlash("app/math/core.t4i")), string(iface)); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(tmp, "app-bin"+tgt.ExeExt)
	stats, err := BuildFileWithStatsOpt(appSrc, outPath, tgt.Triple, BuildOptions{LinkObjectPaths: []string{libObj}})
	if err != nil {
		t.Fatalf("build with .t4i + matching .tobj: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("missing output: %v", err)
	}
	testkit.AssertModules(t, stats.InterfaceModules, []string{"math.core"})
}

func TestBuildLinksGeneratedInterfaceExtensionWithMatchingImplementationObject(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module engine.vec

pub struct Vec2:
    x: Int
    y: Int

pub extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x + self.y
`)
	libSrc := filepath.Join(tmp, filepath.FromSlash("lib/engine/vec.t4"))
	libObj := filepath.Join(tmp, "vec.tobj")
	if err := writeFile(libSrc, string(src)); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildFileWithStatsOpt(libSrc, libObj, tgt.Triple, BuildOptions{Emit: EmitLibrary}); err != nil {
		t.Fatalf("build library: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(src, libSrc)
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, `module app.main
import engine.vec as vec

func main() -> Int:
    let v: vec.Vec2 = vec.Vec2(x: 40, y: 2)
    return vec.Vec2.sum(v)
`); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(tmp, filepath.FromSlash("app/engine/vec.t4i")), string(iface)); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(tmp, "app-bin"+tgt.ExeExt)
	stats, err := BuildFileWithStatsOpt(appSrc, outPath, tgt.Triple, BuildOptions{LinkObjectPaths: []string{libObj}})
	if err != nil {
		t.Fatalf("build generated .t4i extension with matching .tobj: %v\ninterface:\n%s", err, iface)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("missing output: %v", err)
	}
	testkit.AssertModules(t, stats.InterfaceModules, []string{"engine.vec"})
}

func TestBuildRejectsInterfaceDependencyWithoutImplementationObject(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	iface, err := GenerateInterfaceFromSource(src, filepath.Join(tmp, filepath.FromSlash("math/core.t4")))
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/main.t4"))
	if err := writeFile(appSrc, "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(tmp, filepath.FromSlash("math/core.t4i")), string(iface)); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(appSrc, filepath.Join(tmp, "app"+tgt.ExeExt), tgt.Triple, BuildOptions{})
	if err == nil {
		t.Fatalf("expected missing implementation object error")
	}
	if !strings.Contains(err.Error(), "missing implementation object for interface module 'math.core'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsInterfaceImplementationAPIHashMismatch(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	interfaceSrc := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	implSrc := []byte(`module math.core

pub func add(a: Int, b: Bool) -> Int:
    return a
`)
	libSrc := filepath.Join(tmp, filepath.FromSlash("lib/math/core.t4"))
	libObj := filepath.Join(tmp, "math.tobj")
	if err := writeFile(libSrc, string(implSrc)); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildFileWithStatsOpt(libSrc, libObj, tgt.Triple, BuildOptions{Emit: EmitLibrary}); err != nil {
		t.Fatalf("build library: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(interfaceSrc, filepath.Join(tmp, filepath.FromSlash("app/math/core.t4")))
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(tmp, filepath.FromSlash("app/math/core.t4i")), string(iface)); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(appSrc, filepath.Join(tmp, "app"+tgt.ExeExt), tgt.Triple, BuildOptions{LinkObjectPaths: []string{libObj}})
	if err == nil {
		t.Fatalf("expected API hash mismatch")
	}
	if !strings.Contains(err.Error(), "public API hash mismatch for interface module 'math.core'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsDuplicateInterfaceImplementationObjects(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	libSrc := filepath.Join(tmp, filepath.FromSlash("lib/math/core.t4"))
	if err := writeFile(libSrc, string(src)); err != nil {
		t.Fatal(err)
	}
	var objs []string
	for _, name := range []string{"math-a.tobj", "math-b.tobj"} {
		objPath := filepath.Join(tmp, name)
		if _, err := BuildFileWithStatsOpt(libSrc, objPath, tgt.Triple, BuildOptions{Emit: EmitLibrary}); err != nil {
			t.Fatalf("build library %s: %v", name, err)
		}
		objs = append(objs, objPath)
	}
	iface, err := GenerateInterfaceFromSource(src, libSrc)
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(tmp, filepath.FromSlash("app/math/core.t4i")), string(iface)); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(appSrc, filepath.Join(tmp, "app"+tgt.ExeExt), tgt.Triple, BuildOptions{LinkObjectPaths: objs})
	if err == nil {
		t.Fatalf("expected duplicate implementation provider error")
	}
	if !strings.Contains(err.Error(), "duplicate implementation object for interface module 'math.core'") && !strings.Contains(err.Error(), "duplicate symbol 'math.core.add'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsInterfaceImplementationMissingSymbol(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	apiHash, err := InterfaceFingerprintFromSource(src, filepath.Join(tmp, filepath.FromSlash("math/core.t4")))
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	objPath := filepath.Join(tmp, "math.tobj")
	if err := WriteObject(objPath, &Object{
		Target:          tgt.Triple,
		Module:          "math.core",
		CompilerVersion: Version(),
		PublicAPIHash:   apiHash,
		Code:            []byte{0xC3},
		Symbols:         []Symbol{{Name: "math.core.other", Offset: 0}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(src, filepath.Join(tmp, filepath.FromSlash("app/math/core.t4")))
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(tmp, filepath.FromSlash("app/math/core.t4i")), string(iface)); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(appSrc, filepath.Join(tmp, "app-bin"+tgt.ExeExt), tgt.Triple, BuildOptions{LinkObjectPaths: []string{objPath}})
	if err == nil {
		t.Fatalf("expected missing implementation symbol error")
	}
	if !strings.Contains(err.Error(), "implementation object for interface module 'math.core' missing exported symbol 'math.core.add'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsInterfaceImplementationSignatureMismatch(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	apiHash, err := InterfaceFingerprintFromSource(src, filepath.Join(tmp, filepath.FromSlash("math/core.t4")))
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	objPath := filepath.Join(tmp, "math-wrong-abi.tobj")
	if err := WriteObject(objPath, &Object{
		Target:          tgt.Triple,
		Module:          "math.core",
		CompilerVersion: Version(),
		PublicAPIHash:   apiHash,
		Code:            []byte{0xC3},
		Symbols: []Symbol{{
			Name:         "math.core.add",
			Offset:       0,
			HasSignature: true,
			ParamSlots:   1,
			ReturnSlots:  1,
		}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(src, filepath.Join(tmp, filepath.FromSlash("app/math/core.t4")))
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(tmp, filepath.FromSlash("app/math/core.t4i")), string(iface)); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(appSrc, filepath.Join(tmp, "app-bin"+tgt.ExeExt), tgt.Triple, BuildOptions{LinkObjectPaths: []string{objPath}})
	if err == nil {
		t.Fatalf("expected implementation signature mismatch error")
	}
	if !strings.Contains(err.Error(), "implementation object for interface module 'math.core' symbol 'math.core.add' signature mismatch") ||
		!strings.Contains(err.Error(), "params=1 want=2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsInterfaceImplementationMissingSignatureMetadata(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	apiHash, err := InterfaceFingerprintFromSource(src, filepath.Join(tmp, filepath.FromSlash("math/core.t4")))
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	objPath := filepath.Join(tmp, "math-nosig.tobj")
	if err := WriteObject(objPath, &Object{
		Target:          tgt.Triple,
		Module:          "math.core",
		CompilerVersion: Version(),
		PublicAPIHash:   apiHash,
		Code:            []byte{0xC3},
		Symbols: []Symbol{{
			Name:   "math.core.add",
			Offset: 0,
		}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(src, filepath.Join(tmp, filepath.FromSlash("app/math/core.t4")))
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(tmp, filepath.FromSlash("app/math/core.t4i")), string(iface)); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(appSrc, filepath.Join(tmp, "app-bin"+tgt.ExeExt), tgt.Triple, BuildOptions{LinkObjectPaths: []string{objPath}})
	if err == nil {
		t.Fatalf("expected missing implementation signature metadata error")
	}
	if !strings.Contains(err.Error(), "implementation object for interface module 'math.core' symbol 'math.core.add' missing signature metadata") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsInterfaceImplementationGenericExport(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module lib.generic

pub func id<T>(x: T) -> T:
    return x
`)
	apiHash, err := InterfaceFingerprintFromSource(src, filepath.Join(tmp, filepath.FromSlash("lib/generic.t4")))
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	objPath := filepath.Join(tmp, "generic.tobj")
	if err := WriteObject(objPath, &Object{
		Target:          tgt.Triple,
		Module:          "lib.generic",
		CompilerVersion: Version(),
		PublicAPIHash:   apiHash,
		Code:            []byte{0xC3},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(src, filepath.Join(tmp, filepath.FromSlash("app/lib/generic.t4")))
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, "module app.main\nimport lib.generic as generic\nfunc main() -> Int:\n    return 0\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(tmp, filepath.FromSlash("app/lib/generic.t4i")), string(iface)); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(appSrc, filepath.Join(tmp, "app-bin"+tgt.ExeExt), tgt.Triple, BuildOptions{LinkObjectPaths: []string{objPath}})
	if err == nil {
		t.Fatalf("expected unsupported generic export error")
	}
	if !strings.Contains(err.Error(), "implementation object for interface module 'lib.generic' cannot satisfy generic export 'lib.generic.id'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadLinkObjectsRejectsDuplicatePath(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	objPath := filepath.Join(tmp, "lib.tobj")
	if err := WriteObject(objPath, &Object{
		Target:  tgt.Triple,
		Module:  "dup.path",
		Code:    []byte{0xC3},
		Symbols: []Symbol{{Name: "dup.path.entry", Offset: 0}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}

	_, err := readLinkObjects([]string{objPath, filepath.Join(tmp, ".", "lib.tobj")}, tgt.Triple)
	if err == nil {
		t.Fatalf("expected duplicate path error")
	}
	if !strings.Contains(err.Error(), "duplicate link object path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadLinkObjectsRejectsMissingModuleIdentity(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	objPath := filepath.Join(tmp, "nomodule.tobj")
	if err := WriteObject(objPath, &Object{
		Target:  tgt.Triple,
		Code:    []byte{0xC3},
		Symbols: []Symbol{{Name: "nomodule.entry", Offset: 0}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}

	_, err := readLinkObjects([]string{objPath}, tgt.Triple)
	if err == nil {
		t.Fatalf("expected missing module identity error")
	}
	if !strings.Contains(err.Error(), "link object has no module identity") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadLinkObjectsRejectsDuplicateSymbolsBeforeLinking(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	paths := []string{
		filepath.Join(tmp, "a.tobj"),
		filepath.Join(tmp, "b.tobj"),
	}
	for i, path := range paths {
		if err := WriteObject(path, &Object{
			Target:  tgt.Triple,
			Module:  []string{"a", "b"}[i],
			Code:    []byte{0xC3},
			Symbols: []Symbol{{Name: "shared.symbol", Offset: 0}},
		}); err != nil {
			t.Fatalf("write object %s: %v", path, err)
		}
	}

	_, err := readLinkObjects(paths, tgt.Triple)
	if err == nil {
		t.Fatalf("expected duplicate symbol error")
	}
	if !strings.Contains(err.Error(), "duplicate symbol 'shared.symbol' in link objects") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadLinkObjectsRejectsDuplicateSymbolsInsideObject(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	objPath := filepath.Join(tmp, "dup-symbol.tobj")
	if err := WriteObject(objPath, &Object{
		Target: tgt.Triple,
		Module: "dup.symbol",
		Code:   []byte{0xC3},
		Symbols: []Symbol{
			{Name: "dup.entry", Offset: 0},
			{Name: "dup.entry", Offset: 0},
		},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}

	_, err := readLinkObjects([]string{objPath}, tgt.Triple)
	if err == nil {
		t.Fatalf("expected duplicate symbol error")
	}
	if !strings.Contains(err.Error(), "duplicate symbol 'dup.entry' inside link object") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeFile(path string, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func TestLinkObjectLibraryBuildPath(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	libSrc := filepath.Join(tmp, "lib.tetra")
	libObj := filepath.Join(tmp, "lib.tobj")
	if err := os.WriteFile(libSrc, []byte("@export(\"linked_answer\")\nfun answer(): i32 { return 42 }\n"), 0o644); err != nil {
		t.Fatalf("write library source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(libSrc, libObj, tgt.Triple, BuildOptions{Emit: EmitLibrary}); err != nil {
		t.Fatalf("build library: %v", err)
	}

	outPath := filepath.Join(tmp, "app"+tgt.ExeExt)
	if _, err := BuildFileWithStatsOpt(filepath.Join("..", "examples", "hello.tetra"), outPath, tgt.Triple, BuildOptions{
		LinkObjectPaths: []string{libObj},
	}); err != nil {
		t.Fatalf("build with link object: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("missing output: %v", err)
	}
}

func TestRepeatedLinkObjectsAreAccepted(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	var objs []string
	for _, name := range []string{"one", "two"} {
		srcPath := filepath.Join(tmp, name+".tetra")
		objPath := filepath.Join(tmp, name+".tobj")
		src := "@export(\"linked_" + name + "\")\nfun " + name + "(): i32 { return 1 }\n"
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatalf("write source: %v", err)
		}
		if _, err := BuildFileWithStatsOpt(srcPath, objPath, tgt.Triple, BuildOptions{Emit: EmitLibrary}); err != nil {
			t.Fatalf("build library %s: %v", name, err)
		}
		objs = append(objs, objPath)
	}

	outPath := filepath.Join(tmp, "app"+tgt.ExeExt)
	if _, err := BuildFileWithStatsOpt(filepath.Join("..", "examples", "hello.tetra"), outPath, tgt.Triple, BuildOptions{
		LinkObjectPaths: objs,
	}); err != nil {
		t.Fatalf("build with repeated link objects: %v", err)
	}
}

func TestLinkObjectDuplicateSymbolDiagnostic(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	var objs []string
	for _, name := range []string{"a", "b"} {
		srcPath := filepath.Join(tmp, name+".tetra")
		objPath := filepath.Join(tmp, name+".tobj")
		if err := os.WriteFile(srcPath, []byte("@export(\"dup_symbol\")\nfun "+name+"(): i32 { return 1 }\n"), 0o644); err != nil {
			t.Fatalf("write source: %v", err)
		}
		if _, err := BuildFileWithStatsOpt(srcPath, objPath, tgt.Triple, BuildOptions{Emit: EmitLibrary}); err != nil {
			t.Fatalf("build library %s: %v", name, err)
		}
		objs = append(objs, objPath)
	}

	outPath := filepath.Join(tmp, "app"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(filepath.Join("..", "examples", "hello.tetra"), outPath, tgt.Triple, BuildOptions{
		LinkObjectPaths: objs,
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate symbol 'dup_symbol'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkObjectMissingSymbolDiagnostic(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	objPath := filepath.Join(tmp, "missing.tobj")
	if err := WriteObject(objPath, &Object{
		Target:  tgt.Triple,
		Module:  "__missing_ref",
		Code:    []byte{0xE8, 0, 0, 0, 0, 0xC3},
		Symbols: []Symbol{{Name: "__missing_ref_entry", Offset: 0}},
		Relocs:  []Reloc{{Kind: RelocCallRel32, At: 1, Name: "missing.symbol"}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}

	outPath := filepath.Join(tmp, "app"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(filepath.Join("..", "examples", "hello.tetra"), outPath, tgt.Triple, BuildOptions{
		LinkObjectPaths: []string{objPath},
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unresolved symbol 'missing.symbol'") {
		t.Fatalf("unexpected error: %v", err)
	}
}
