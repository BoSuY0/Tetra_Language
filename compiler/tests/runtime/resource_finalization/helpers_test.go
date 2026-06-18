package compiler_test

import (
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func requireCheckWorldFilesErrorContains(
	t *testing.T,
	files map[string]string,
	entry string,
	want string,
) {
	t.Helper()
	tmp := t.TempDir()
	testkit.WriteFiles(t, tmp, files)
	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash(entry)))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}
