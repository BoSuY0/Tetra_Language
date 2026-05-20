package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestEmitLibraryAllowsNoMainAndWritesTOBJ(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "lib.tetra")
	outPath := filepath.Join(tmp, "lib.tobj")

	src := "fun foo(): i32 { return 0 }\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{Emit: compiler.EmitLibrary}); err != nil {
		t.Fatalf("build library object: %v", err)
	}
	obj, err := compiler.ReadObject(outPath)
	if err != nil {
		t.Fatalf("read object: %v", err)
	}
	if obj.Target != "linux-x64" {
		t.Fatalf("target mismatch: %q", obj.Target)
	}
	if obj.CompilerVersion != compiler.Version() {
		t.Fatalf("compiler version = %q, want %q", obj.CompilerVersion, compiler.Version())
	}
	if obj.PublicAPIHash == "" {
		t.Fatalf("missing public API hash")
	}
	found := false
	for _, sym := range obj.Symbols {
		if sym.Name == "foo" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing foo symbol")
	}
}

func TestEmitLibraryWritesPublicAPIHash(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, filepath.FromSlash("math/core.t4"))
	outPath := filepath.Join(tmp, "math.tobj")
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(srcPath, src, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	wantHash, err := compiler.InterfaceFingerprintFromSource(src, srcPath)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{Emit: compiler.EmitLibrary}); err != nil {
		t.Fatalf("build library object: %v", err)
	}
	obj, err := compiler.ReadObject(outPath)
	if err != nil {
		t.Fatalf("read object: %v", err)
	}
	if obj.Module != "math.core" {
		t.Fatalf("module = %q, want math.core", obj.Module)
	}
	if obj.CompilerVersion != compiler.Version() {
		t.Fatalf("compiler version = %q, want %q", obj.CompilerVersion, compiler.Version())
	}
	if obj.PublicAPIHash != wantHash {
		t.Fatalf("public API hash = %q, want %q", obj.PublicAPIHash, wantHash)
	}
	if !emitObjectHasSymbol(obj, "math.core.add") {
		t.Fatalf("object missing math.core.add symbol: %#v", obj.Symbols)
	}
}

func emitObjectHasSymbol(obj *compiler.Object, name string) bool {
	if obj == nil {
		return false
	}
	for _, sym := range obj.Symbols {
		if strings.EqualFold(sym.Name, name) || sym.Name == name {
			return true
		}
	}
	return false
}

func TestEmitObjectRequiresMain(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "obj.tetra")
	outPath := filepath.Join(tmp, "obj.tobj")

	src := "fun foo(): i32 { return 0 }\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{Emit: compiler.EmitObject}); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestEmitLibraryBuildOnlyAcrossNativeTargets(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "lib.tetra")
	if err := os.WriteFile(srcPath, []byte("fun foo(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32): i32 { return a + b + c + d + e + f + g }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	targets := []string{"linux-x64", "macos-x64", "windows-x64"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			outPath := filepath.Join(tmp, "lib-"+target+".tobj")
			if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, target, compiler.BuildOptions{Emit: compiler.EmitLibrary}); err != nil {
				t.Fatalf("emit library for %s: %v", target, err)
			}
			obj, err := compiler.ReadObject(outPath)
			if err != nil {
				t.Fatalf("read object: %v", err)
			}
			if obj.Target != target {
				t.Fatalf("target mismatch: got %q want %q", obj.Target, target)
			}
		})
	}
}

func TestEmitLibraryObjectRelocProfileByTarget(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "libio.tetra")
	src := "fun main(): i32 uses io {\n  print(\"ObjectReloc\\n\")\n  return 0\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	targets := []string{"linux-x64", "macos-x64", "windows-x64"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			outPath := filepath.Join(tmp, "obj-"+target+".tobj")
			if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, target, compiler.BuildOptions{Emit: compiler.EmitLibrary}); err != nil {
				t.Fatalf("emit library for %s: %v", target, err)
			}
			obj, err := compiler.ReadObject(outPath)
			if err != nil {
				t.Fatalf("read object: %v", err)
			}
			hasData := false
			hasIAT := false
			for _, reloc := range obj.Relocs {
				if reloc.Kind == compiler.RelocDataDisp32 {
					hasData = true
				}
				if reloc.Kind == compiler.RelocIATDisp32 {
					hasIAT = true
				}
			}
			if !hasData {
				t.Fatalf("expected data relocation for %s", target)
			}
			if target == "windows-x64" && !hasIAT {
				t.Fatalf("expected IAT relocation for windows-x64")
			}
			if target != "windows-x64" && hasIAT {
				t.Fatalf("unexpected IAT relocation for %s", target)
			}
		})
	}
}
