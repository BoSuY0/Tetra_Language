package surface_gates

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

func workflowJobSection(workflow, job string) string {
	lines := strings.Split(workflow, "\n")
	var section []string
	inJob := false
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
			if inJob {
				break
			}
			inJob = line == "  "+job
		}
		if inJob {
			section = append(section, line)
		}
	}
	return strings.Join(section, "\n")
}

func parseBashStringArray(t *testing.T, script, name string) []string {
	t.Helper()
	start := name + "=("
	inArray := false
	var values []string
	for lineNo, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if !inArray {
			if trimmed == start {
				inArray = true
			}
			continue
		}
		if trimmed == ")" {
			return values
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		value, err := strconv.Unquote(trimmed)
		if err != nil {
			t.Fatalf("parse %s entry on line %d: %v", name, lineNo+1, err)
		}
		values = append(values, value)
	}
	if !inArray {
		t.Fatalf("missing bash array %s", name)
	}
	t.Fatalf("bash array %s is not closed", name)
	return nil
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

func assertOrderedFragments(t *testing.T, text string, fragments ...string) {
	t.Helper()
	last := -1
	for _, fragment := range fragments {
		idx := strings.Index(text, fragment)
		if idx < 0 {
			t.Fatalf("missing ordered fragment %q", fragment)
		}
		if idx < last {
			t.Fatalf("fragment %q appears out of order", fragment)
		}
		last = idx
	}
}
