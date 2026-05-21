package compiler_test

import (
	"bytes"
	"encoding/binary"
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

	targets := []string{"linux-x86", "linux-x64", "macos-x64", "windows-x64"}
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

func TestEmitLibraryLinuxX32WritesRealX32TOBJ(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "libx32.tetra")
	outPath := filepath.Join(tmp, "libx32.tobj")
	src := "fun say(): i32 uses io {\n  print(\"x32 object\\n\")\n  return 0\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "x32", compiler.BuildOptions{Emit: compiler.EmitLibrary}); err != nil {
		t.Fatalf("emit linux-x32 library object: %v", err)
	}
	obj, err := compiler.ReadObject(outPath)
	if err != nil {
		t.Fatalf("read object: %v", err)
	}
	if obj.Target != "linux-x32" {
		t.Fatalf("target mismatch: got %q want linux-x32", obj.Target)
	}
	if !emitObjectHasSymbol(obj, "say") {
		t.Fatalf("object missing say symbol: %#v", obj.Symbols)
	}
	if !containsMovEaxImm32(obj.Code, 0x40000001) {
		t.Fatalf("missing x32 write syscall number in object code: % x", obj.Code)
	}
	if containsMovEaxImm32(obj.Code, 1) {
		t.Fatalf("linux-x32 object emitted plain x64 write syscall number: % x", obj.Code)
	}
	for _, reloc := range obj.Relocs {
		if reloc.Kind == compiler.RelocIATDisp32 {
			t.Fatalf("linux-x32 object unexpectedly has Windows IAT reloc: %#v", obj.Relocs)
		}
	}
}

func TestEmitLibraryLinuxX32BuildsI64AndWeakAtomicObject(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "atomic_x32.tetra")
	outPath := filepath.Join(tmp, "atomic_x32.tobj")
	src := `
func atomic_probe() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let loaded: i64 = core.atomic_load_i64_acquire(p, mem)
        let exchanged: i64 = core.atomic_exchange_i64_seq_cst(p, loaded, mem)
        let weak64: i64 = core.atomic_compare_exchange_weak_i64_seq_cst(p, loaded, exchanged, mem)
        var ignored_store: i64 = core.atomic_store_i64_release(p, weak64, mem)
        return core.atomic_compare_exchange_weak_i32_seq_cst(p, 0, 1, mem)
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "x32", compiler.BuildOptions{Emit: compiler.EmitLibrary}); err != nil {
		t.Fatalf("emit linux-x32 atomic library object: %v", err)
	}
	obj, err := compiler.ReadObject(outPath)
	if err != nil {
		t.Fatalf("read object: %v", err)
	}
	if obj.Target != "linux-x32" {
		t.Fatalf("target mismatch: got %q want linux-x32", obj.Target)
	}
	if !emitObjectHasSymbol(obj, "atomic_probe") {
		t.Fatalf("object missing atomic_probe symbol: %#v", obj.Symbols)
	}
	if !bytes.Contains(obj.Code, []byte{0xF0, 0x4C, 0x0F, 0xB1, 0x07}) {
		t.Fatalf("missing qword weak-CAS codegen for i64 atomic on x32: % x", obj.Code)
	}
	if !bytes.Contains(obj.Code, []byte{0xF0, 0x44, 0x0F, 0xB1, 0x07}) {
		t.Fatalf("missing dword weak-CAS codegen for i32 atomic on x32: % x", obj.Code)
	}
}

func TestEmitLibraryLinuxX86WritesRealI386TOBJ(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "libx86.tetra")
	outPath := filepath.Join(tmp, "libx86.tobj")
	if err := os.WriteFile(srcPath, []byte("fun foo(a: i32, b: i32): i32 { return a + b }\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "x86", compiler.BuildOptions{Emit: compiler.EmitLibrary}); err != nil {
		t.Fatalf("emit linux-x86 library object: %v", err)
	}
	obj, err := compiler.ReadObject(outPath)
	if err != nil {
		t.Fatalf("read object: %v", err)
	}
	if obj.Target != "linux-x86" {
		t.Fatalf("target mismatch: got %q want linux-x86", obj.Target)
	}
	if !emitObjectHasSymbol(obj, "foo") {
		t.Fatalf("object missing foo symbol: %#v", obj.Symbols)
	}
	if !bytes.Contains(obj.Code, []byte{0x55, 0x89, 0xE5}) {
		t.Fatalf("missing i386 frame prologue in object code: % x", obj.Code)
	}
	if !bytes.Contains(obj.Code, []byte{0xC9, 0xC3}) {
		t.Fatalf("missing i386 leave/ret epilogue in object code: % x", obj.Code)
	}
}

func containsMovEaxImm32(buf []byte, imm uint32) bool {
	for i := 0; i+5 <= len(buf); i++ {
		if buf[i] == 0xB8 && binary.LittleEndian.Uint32(buf[i+1:i+5]) == imm {
			return true
		}
	}
	return false
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
