package reportdir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFreshRejectsUnsafeReportDirs(t *testing.T) {
	repoRoot := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(repoRoot, "file-report"),
		[]byte("not a directory"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	nonEmpty := filepath.Join(repoRoot, "non-empty")
	if err := os.MkdirAll(nonEmpty, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nonEmpty, "old.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	linkParentTarget := filepath.Join(repoRoot, "real-parent")
	if err := os.MkdirAll(linkParentTarget, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.Symlink(linkParentTarget, filepath.Join(repoRoot, "link-parent")); err != nil {
		t.Fatalf("Symlink parent: %v", err)
	}
	linkFinalTarget := filepath.Join(repoRoot, "real-final")
	if err := os.MkdirAll(linkFinalTarget, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.Symlink(linkFinalTarget, filepath.Join(repoRoot, "link-final")); err != nil {
		t.Fatalf("Symlink final: %v", err)
	}

	cases := []struct {
		name      string
		reportDir string
		want      string
	}{
		{name: "empty", reportDir: "", want: "empty"},
		{name: "absolute", reportDir: filepath.Join(repoRoot, "abs"), want: "absolute"},
		{name: "repo root dot", reportDir: ".", want: "repo root"},
		{name: "repo root slash dot", reportDir: "./", want: "repo root"},
		{name: "dash prefixed", reportDir: "-release", want: "dash"},
		{name: "parent traversal", reportDir: "reports/../escape", want: "parent traversal"},
		{name: "symlink parent", reportDir: "link-parent/report", want: "symlink"},
		{name: "symlink final", reportDir: "link-final", want: "symlink"},
		{name: "existing non directory", reportDir: "file-report", want: "non-directory"},
		{name: "existing non empty directory", reportDir: "non-empty", want: "non-empty"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := ValidateFresh(repoRoot, tc.reportDir); err == nil {
				t.Fatalf("ValidateFresh(%q) = %q, nil error", tc.reportDir, got)
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateFresh(%q) error = %v, want substring %q", tc.reportDir, err, tc.want)
			}
		})
	}
}

func TestPrepareFreshCreatesNestedRepoRelativeDirectory(t *testing.T) {
	repoRoot := t.TempDir()
	got, err := PrepareFresh(repoRoot, "reports/surface/release")
	if err != nil {
		t.Fatalf("PrepareFresh: %v", err)
	}

	want := filepath.Join(repoRoot, "reports", "surface", "release")
	if got != want {
		t.Fatalf("PrepareFresh path = %q, want %q", got, want)
	}
	info, err := os.Stat(got)
	if err != nil {
		t.Fatalf("Stat created directory: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("PrepareFresh created non-directory mode %v", info.Mode())
	}
}

func TestValidateFreshAcceptsExistingEmptyDirectoryWithoutCreatingMissingPath(t *testing.T) {
	repoRoot := t.TempDir()
	emptyDir := filepath.Join(repoRoot, "reports", "empty")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	got, err := ValidateFresh(repoRoot, "reports/empty")
	if err != nil {
		t.Fatalf("ValidateFresh rejected existing empty directory: %v", err)
	}
	if got != emptyDir {
		t.Fatalf("ValidateFresh path = %q, want %q", got, emptyDir)
	}

	missing := filepath.Join(repoRoot, "reports", "not-created")
	if _, err := ValidateFresh(repoRoot, "reports/not-created"); err != nil {
		t.Fatalf("ValidateFresh rejected missing fresh path: %v", err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("ValidateFresh created %q or got unexpected stat error %v", missing, err)
	}
}
