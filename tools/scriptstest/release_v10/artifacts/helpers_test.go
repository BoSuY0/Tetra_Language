package release_v10_artifacts

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func copyFile(src, dst string, mode os.FileMode) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, raw, mode)
}

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

func assertOutputAvoidsRawPathUtilityErrors(t *testing.T, out []byte) {
	t.Helper()
	for _, forbidden := range []string{
		"unbound variable",
		"mkdir:",
		"dirname:",
		"find:",
		"cp:",
		"cat:",
		"sed:",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf(
				"output should use controlled path hygiene errors, not raw shell utility failures:\n%s",
				out,
			)
		}
	}
}

func normalizeDashLeadingPathForTest(path string) string {
	if strings.HasPrefix(path, "-") {
		return "./" + path
	}
	return path
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}
