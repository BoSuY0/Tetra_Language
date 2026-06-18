package scriptstest

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func workflowJobSection(workflow, job string) string {
	start := strings.Index(workflow, job)
	if start < 0 {
		return ""
	}
	rest := workflow[start+len(job):]
	next := strings.Index(rest, "\n  ")
	if next < 0 {
		return rest
	}
	return rest[:next]
}
