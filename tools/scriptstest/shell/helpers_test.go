package shell

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func assertLegacyFileRemoved(t *testing.T, rel, mustUse string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(rel))); err == nil {
		t.Fatalf("%s must be removed; use %s", rel, mustUse)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", rel, err)
	}
}

func assertNoLegacyMention(t *testing.T, text, legacy, where string) {
	t.Helper()
	if strings.Contains(text, legacy) {
		t.Fatalf("%s must not advertise removed root-level wrapper %s", where, legacy)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
