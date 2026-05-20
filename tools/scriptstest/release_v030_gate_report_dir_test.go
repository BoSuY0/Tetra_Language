package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV030GateRejectsExistingReportArtifacts(t *testing.T) {
	root := releaseV030FakeRepo(t)
	reportDir := filepath.Join(root, "report")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runReleaseV030Gate(t, root, "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected stale report-dir rejection\n%s", out)
	}
	if !strings.Contains(string(out), "refusing to reuse non-empty report directory: "+reportDir) {
		t.Fatalf("unexpected stale report-dir output:\n%s", out)
	}
}

func TestReleaseV030GateRejectsSymlinkToExistingReportArtifacts(t *testing.T) {
	root := releaseV030FakeRepo(t)
	targetDir := filepath.Join(root, "stale-target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(root, "report-link")
	if err := os.Symlink(targetDir, reportDir); err != nil {
		t.Fatalf("create report-dir symlink: %v", err)
	}
	out, err := runReleaseV030Gate(t, root, "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected stale symlinked report-dir rejection\n%s", out)
	}
	if !strings.Contains(string(out), "refusing to reuse non-empty report directory: "+reportDir) {
		t.Fatalf("unexpected stale symlinked report-dir output:\n%s", out)
	}
}

func TestReleaseV030GateRejectsDashPrefixedExistingReportArtifacts(t *testing.T) {
	root := releaseV030FakeRepo(t)
	reportDir := "-stale-report"
	if err := os.MkdirAll(filepath.Join(root, reportDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, reportDir, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runReleaseV030Gate(t, root, "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected dash-prefixed stale report-dir rejection\n%s", out)
	}
	if !strings.Contains(string(out), "refusing to reuse non-empty report directory: "+reportDir) {
		t.Fatalf("unexpected dash-prefixed stale report-dir output:\n%s", out)
	}
	if strings.Contains(string(out), "find:") {
		t.Fatalf("dash-prefixed report-dir should not be parsed as a find option:\n%s", out)
	}
}

func TestReleaseV030GateRejectsNonDirectoryReportPath(t *testing.T) {
	root := releaseV030FakeRepo(t)
	reportDir := filepath.Join(root, "report-file")
	if err := os.WriteFile(reportDir, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertReleaseV030RejectsNonDirectoryReportPath(t, root, reportDir)
}

func TestReleaseV030GateRejectsDanglingReportDirSymlink(t *testing.T) {
	root := releaseV030FakeRepo(t)
	reportDir := filepath.Join(root, "dangling-report-link")
	if err := os.Symlink(filepath.Join(root, "missing-report-target"), reportDir); err != nil {
		t.Fatal(err)
	}

	assertReleaseV030RejectsNonDirectoryReportPath(t, root, reportDir)
}

func TestReleaseV030GateRejectsReportDirSymlinkToFile(t *testing.T) {
	root := releaseV030FakeRepo(t)
	target := filepath.Join(root, "report-target-file")
	if err := os.WriteFile(target, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(root, "report-file-link")
	if err := os.Symlink(target, reportDir); err != nil {
		t.Fatal(err)
	}

	assertReleaseV030RejectsNonDirectoryReportPath(t, root, reportDir)
}

func assertReleaseV030RejectsNonDirectoryReportPath(t *testing.T, root, reportDir string) {
	t.Helper()

	out, err := runReleaseV030Gate(t, root, "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected non-directory report-dir rejection\n%s", out)
	}
	for _, want := range []string{
		"release_v0_3_0_gate: refusing to use non-directory report path: " + reportDir,
		"release_v0_3_0_gate: choose a fresh --report-dir directory",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("non-directory report-dir output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "mkdir:") || strings.Contains(string(out), "bootstrapping") {
		t.Fatalf("gate should reject invalid report dir before raw shell or workflow side effects:\n%s", out)
	}
}
