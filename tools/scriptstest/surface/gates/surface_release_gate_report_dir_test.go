package surface_gates

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSurfaceReleaseGateScriptDocumentsGuardAndMetadata(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate: %v", err)
	}
	gate := string(raw)
	for _, want := range []string{
		"source \"$script_dir/report-dir-guard.sh\"",
		("surface_release_require_fresh_report_dir \"$report_dir\" \"$repo_" +
			"root\" \"surface_release_gate:\""),
		"\"producer\": \"scripts/release/surface/release-gate.sh\"",
		"\"git_head\":",
		"\"git_dirty\":",
		"\"host_os\":",
		"\"host_arch\":",
		"\"generated_at_utc\":",
		"\"command_line\":",
		"\"version\":",
	} {
		if !strings.Contains(gate, want) {
			t.Fatalf("Surface release gate missing %q", want)
		}
	}
}

func TestSurfaceReleaseGateRejectsStaleReportDir(t *testing.T) {
	root := surfaceReleaseGateFakeRoot(t)
	reportRel := filepath.ToSlash(filepath.Join("reports", "surface-release-v1"))
	reportPath := filepath.Join(root, filepath.FromSlash(reportRel))
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(reportPath, "surface-release-summary.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	out, err := runSurfaceReleaseGate(t, root, "--report-dir", reportRel)
	if err == nil {
		t.Fatalf("expected stale report-dir rejection\n%s", out)
	}
	for _, want := range []string{
		"surface_release_gate: refusing to reuse non-empty report directory: " + reportRel,
		"surface_release_gate: choose a fresh --report-dir so stale reports cannot be reused",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("stale report-dir output missing %q:\n%s", want, out)
		}
	}
	assertSurfaceReleaseGateRejectedBeforeSideEffects(t, out)
}

func TestSurfaceReleaseGateRejectsSymlinkReportDir(t *testing.T) {
	root := surfaceReleaseGateFakeRoot(t)
	targetDir := filepath.Join(root, "reports", "stale-target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	reportRel := filepath.ToSlash(filepath.Join("reports", "surface-release-link"))
	reportPath := filepath.Join(root, filepath.FromSlash(reportRel))
	if err := os.Symlink(targetDir, reportPath); err != nil {
		t.Fatalf("create report-dir symlink: %v", err)
	}

	out, err := runSurfaceReleaseGate(t, root, "--report-dir", reportRel)
	if err == nil {
		t.Fatalf("expected symlink report-dir rejection\n%s", out)
	}
	for _, want := range []string{
		"surface_release_gate: refusing to use symlink report directory: " + reportRel,
		"surface_release_gate: choose a real fresh --report-dir",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("symlink report-dir output missing %q:\n%s", want, out)
		}
	}
	assertSurfaceReleaseGateRejectedBeforeSideEffects(t, out)
}

func TestSurfaceReleaseGateRejectsTraversalReportDir(t *testing.T) {
	root := surfaceReleaseGateFakeRoot(t)
	reportDir := filepath.ToSlash(filepath.Join("..", "surface-release-outside"))

	out, err := runSurfaceReleaseGate(t, root, "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected traversal report-dir rejection\n%s", out)
	}
	for _, want := range []string{
		("surface_release_gate: refusing unsafe report directory: parent " +
			"traversal is not accepted (") + reportDir + ")",
		"surface_release_gate: choose a fresh repo-relative --report-dir",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("traversal report-dir output missing %q:\n%s", want, out)
		}
	}
	assertSurfaceReleaseGateRejectedBeforeSideEffects(t, out)
}

func TestSurfaceReleaseGateRejectsUnsafeReportDirForms(t *testing.T) {
	root := surfaceReleaseGateFakeRoot(t)
	fileRel := filepath.ToSlash(filepath.Join("reports", "not-a-dir"))
	filePath := filepath.Join(root, filepath.FromSlash(fileRel))
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name      string
		reportDir string
		want      string
	}{
		{name: "empty", reportDir: "", want: "surface_release_gate: --report-dir requires a value"},
		{name: "absolute", reportDir: filepath.Join(
			root,
			"reports",
			"absolute",
		), want: ("surface_release_gate: refusing unsafe report directory: " +
			"absolute paths are not accepted")},
		{
			name:      "repo-root",
			reportDir: ".",
			want: ("surface_release_gate: refusing unsafe report directory: repo " +
				"root is not a release report directory"),
		},
		{
			name:      "non-directory",
			reportDir: fileRel,
			want:      "surface_release_gate: refusing to use non-directory report path: " + fileRel,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runSurfaceReleaseGate(t, root, "--report-dir", tc.reportDir)
			if err == nil {
				t.Fatalf("expected unsafe report-dir rejection\n%s", out)
			}
			if !strings.Contains(string(out), tc.want) {
				t.Fatalf("unsafe report-dir output missing %q:\n%s", tc.want, out)
			}
			assertSurfaceReleaseGateRejectedBeforeSideEffects(t, out)
		})
	}
}

func surfaceReleaseGateFakeRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	srcDir := filepath.Join(repoRoot(t), "scripts", "release", "surface")
	dstDir := filepath.Join(root, "scripts", "release", "surface")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatalf("read Surface release script dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sh") {
			continue
		}
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())
		if err := copyFile(src, dst, 0o755); err != nil {
			t.Fatalf("copy %s: %v", entry.Name(), err)
		}
	}
	return root
}

func runSurfaceReleaseGate(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()

	cmdArgs := append([]string{"scripts/release/surface/release-gate.sh"}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOTELEMETRY=off",
		"GOCACHE="+filepath.Join(root, ".cache", "go-build-surface-p01-scriptstest"),
	)
	return cmd.CombinedOutput()
}

func assertSurfaceReleaseGateRejectedBeforeSideEffects(t *testing.T, out []byte) {
	t.Helper()

	for _, forbidden := range []string{
		"surface-headless-release-smoke.sh",
		"surface-runtime-smoke",
		"go:",
		"stat ",
		"mkdir:",
		"find:",
		"No such file or directory",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf(
				"release gate should reject report dir before sub-gate or raw shell side effects:\n%s",
				out,
			)
		}
	}
}
