package compiler

import (
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
