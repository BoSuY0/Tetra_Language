package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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
