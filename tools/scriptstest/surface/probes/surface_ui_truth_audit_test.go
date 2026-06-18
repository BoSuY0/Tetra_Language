package surface_probes

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestSurfaceUITruthAuditScriptContract(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "analysis", "surface-ui-truth-audit.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Surface UI truth audit script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/analysis/surface-ui-truth-audit.sh --report-dir DIR",
		"set -euo pipefail",
		"surface-rg-index.txt",
		"surface-script-index.txt",
		"surface-tool-index.txt",
		"surface-example-index.txt",
		"focused-tests.log",
		"surface-gates.log",
		"validators.log",
		"truth-summary.md",
		"production_ready_claim: false",
		"PASS",
		"FAIL",
		"BLOCKED",
		"SKIPPED",
		"gate-runs",
		("TestReleaseSurface|TestCurrentSupportedSurface|TestCIWorkflowInc" +
			"ludesSurface|TestSurface(Tree|Toolkit|Accessibility)"),
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Surface UI truth audit script missing %q", want)
		}
	}
	if strings.Contains(text, "-run 'Surface|surface'") {
		t.Fatalf(
			"Surface UI truth audit script must not recursively run its own artifact-writing tests",
		)
	}
}

func TestSurfaceUITruthAuditRejectsUnsafeReportDirs(t *testing.T) {
	root := repoRoot(t)
	script := filepath.Join(root, "scripts", "analysis", "surface-ui-truth-audit.sh")
	for _, reportDir := range []string{
		"",
		"../surface-ui-audit",
		"/tmp/surface-ui-audit",
		".",
	} {
		t.Run(reportDir, func(t *testing.T) {
			cmd := exec.Command("bash", script, "--report-dir", reportDir)
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected unsafe report dir %q to fail\n%s", reportDir, out)
			}
			if !strings.Contains(string(out), "refusing unsafe report dir") &&
				!strings.Contains(string(out), "--report-dir requires a value") {
				t.Fatalf(
					"expected controlled report-dir rejection for %q, got:\n%s",
					reportDir,
					out,
				)
			}
			assertOutputAvoidsRawPathUtilityErrors(t, out)
		})
	}
}

func TestSurfaceUITruthAuditWritesRequiredArtifacts(t *testing.T) {
	root := repoRoot(t)
	script := filepath.Join(root, "scripts", "analysis", "surface-ui-truth-audit.sh")
	reportRel := filepath.ToSlash(
		filepath.Join(
			"reports",
			"surface-ui-production-audit",
			"scriptstest-p00-"+strconv.Itoa(os.Getpid()),
		),
	)
	reportDir := filepath.Join(root, filepath.FromSlash(reportRel))
	if err := os.RemoveAll(reportDir); err != nil {
		t.Fatalf("clean report dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(reportDir); err != nil {
			t.Logf("clean report dir: %v", err)
		}
	})

	cmd := exec.Command("bash", script, "--report-dir", reportRel)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOTELEMETRY=off",
		"GOCACHE="+filepath.Join(root, ".cache", "go-build-surface-p00-scriptstest"),
		"SURFACE_UI_TRUTH_AUDIT_TIMEOUT=2",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Surface UI truth audit failed: %v\n%s", err, out)
	}

	for _, rel := range []string{
		"surface-rg-index.txt",
		"surface-script-index.txt",
		"surface-tool-index.txt",
		"surface-example-index.txt",
		"focused-tests.log",
		"surface-gates.log",
		"validators.log",
		"truth-summary.md",
	} {
		info, err := os.Stat(filepath.Join(reportDir, rel))
		if err != nil {
			t.Fatalf("expected audit artifact %s: %v\noutput:\n%s", rel, err, out)
		}
		if info.IsDir() || info.Size() == 0 {
			t.Fatalf("expected non-empty audit artifact %s", rel)
		}
	}
	summary, err := os.ReadFile(filepath.Join(reportDir, "truth-summary.md"))
	if err != nil {
		t.Fatalf("read truth summary: %v", err)
	}
	for _, want := range []string{
		"# Surface/UI Truth Audit",
		"- production_ready_claim: false",
		"- report_dir: `" + reportRel + "`",
		"## Checks",
	} {
		if !strings.Contains(string(summary), want) {
			t.Fatalf("truth summary missing %q:\n%s", want, summary)
		}
	}
}
