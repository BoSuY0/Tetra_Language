package postv04_memory

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func copyFile(src, dst string, mode os.FileMode) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, raw, mode)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

func assertEqualOrderedStrings(t *testing.T, got, want []string, label string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf(
			"%s length = %d, want %d\ngot:  %q\nwant: %q",
			label,
			len(got),
			len(want),
			got,
			want,
		)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf(
				"%s[%d] = %q, want %q\ngot:  %q\nwant: %q",
				label,
				i,
				got[i],
				want[i],
				got,
				want,
			)
		}
	}
}
