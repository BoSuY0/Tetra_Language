package surface

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repo root %s missing go.mod: %v", root, err)
	}
	return root
}

func currentReleaseVersion(t *testing.T) string {
	t.Helper()

	raw, err := os.ReadFile(
		filepath.Join(repoRoot(t), "compiler", "internal", "version", "version.go"),
	)
	if err != nil {
		t.Fatalf("read version.go: %v", err)
	}
	matches := regexp.MustCompile(`CompilerVersion = "([^"]+)"`).FindSubmatch(raw)
	if matches == nil {
		t.Fatalf("CompilerVersion not found in version.go")
	}
	return string(matches[1])
}
