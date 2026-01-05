package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

func TestLinkCrossModuleCall(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add(a: i32, b: i32): i32 {\n  return a + b\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as r\nfun main(): i32 {\n  return r.add(20, 22)\n}\n",
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))

	objs, mainName := buildObjectsForEntry(t, entry)
	img, err := LinkLinuxX64(objs, mainName)
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := WriteELF64LinuxX64(outPath, img); err != nil {
		t.Fatalf("write ELF: %v", err)
	}
	stdout, exitCode := runBinary(t, outPath)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestLinkCrossModuleCallSevenPlusArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/math.tetra": "module engine.math\nfun sum7(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32): i32 {\n  return a + b + c + d + e + f + g\n}\n",
		"app/game.tetra":    "module app.game\nimport engine.math as m\nfun main(): i32 {\n  return m.sum7(1, 2, 3, 4, 5, 6, 7)\n}\n",
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))

	objs, mainName := buildObjectsForEntry(t, entry)
	img, err := LinkLinuxX64(objs, mainName)
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := WriteELF64LinuxX64(outPath, img); err != nil {
		t.Fatalf("write ELF: %v", err)
	}
	stdout, exitCode := runBinary(t, outPath)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 28 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestLinkUnresolvedSymbol(t *testing.T) {
	obj := &Object{
		Target: "linux-x64",
		Module: "app.game",
		Code:   []byte{0xE8, 0x00, 0x00, 0x00, 0x00, 0xC3},
		Symbols: []Symbol{
			{Name: "app.game.main", Offset: 0},
		},
		Relocs: []Reloc{
			{Kind: RelocCallRel32, At: 1, Name: "missing.func", Addend: 0},
		},
	}

	if _, err := LinkLinuxX64([]*Object{obj}, "app.game.main"); err == nil {
		t.Fatalf("expected linker error")
	}
}

func buildObjectsForEntry(t *testing.T, entryPath string) ([]*Object, string) {
	t.Helper()

	world, err := LoadWorld(entryPath)
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("check world: %v", err)
	}
	irModules, err := LowerModules(checked)
	if err != nil {
		t.Fatalf("lower: %v", err)
	}

	modules := make([]string, 0, len(world.ByModule))
	for module := range world.ByModule {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	var objects []*Object
	for _, module := range modules {
		obj, err := CodegenObjectLinuxX64(irModules[module])
		if err != nil {
			t.Fatalf("codegen object: %v", err)
		}
		obj.Module = module
		obj.Target = "linux-x64"
		objects = append(objects, obj)
	}
	return objects, checked.MainName
}
