package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmitLibraryAllowsNoMainAndWritesTOBJ(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "lib.tetra")
	outPath := filepath.Join(tmp, "lib.tobj")

	src := "fun foo(): i32 { return 0 }\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Emit: EmitLibrary}); err != nil {
		t.Fatalf("build library object: %v", err)
	}
	obj, err := ReadObject(outPath)
	if err != nil {
		t.Fatalf("read object: %v", err)
	}
	if obj.Target != "linux-x64" {
		t.Fatalf("target mismatch: %q", obj.Target)
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

func TestEmitObjectRequiresMain(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "obj.tetra")
	outPath := filepath.Join(tmp, "obj.tobj")

	src := "fun foo(): i32 { return 0 }\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Emit: EmitObject}); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
