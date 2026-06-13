package scriptstest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSurfaceVisualGateRunsBlockSystemVisualEvidence(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "surface", "visual-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Surface visual gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/visual-gate.sh [--report-dir DIR]",
		`source "$script_dir/report-dir-guard.sh"`,
		`surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_visual_gate:"`,
		`block_system_report_dir="$report_dir_arg/block-system"`,
		`visual_report_path="$report_dir/surface-visual-regression.json"`,
		`bash scripts/release/surface/block-system-gate.sh --report-dir "$block_system_report_dir"`,
		`go run ./tools/cmd/surface-visual-diff`,
		`--block-examples-report "$report_dir/block-system/headless/surface-block-examples.json"`,
		`--runtime-report "$report_dir/block-system/headless/surface-headless-block-system.json"`,
		`--runtime-report "$report_dir/block-system/linux-x64-real-window/surface-block-system-linux-x64.json"`,
		`--runtime-report "$report_dir/block-system/wasm32-web-browser-canvas/surface-block-system-wasm32-web.json"`,
		`--required-target headless`,
		`--required-target linux-x64-real-window`,
		`--required-target wasm32-web-browser-canvas`,
		`go run ./tools/cmd/validate-surface-visual-report --report "$visual_report_path"`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Surface visual gate missing %q", want)
		}
	}
	assertOrderedFragments(t, text,
		`block-system-gate.sh --report-dir "$block_system_report_dir"`,
		`surface-visual-diff`,
		`validate-surface-visual-report --report "$visual_report_path"`,
		`validate-artifact-hashes --write --root "$report_dir"`,
		`validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
	)
	for _, forbidden := range []string{"continue-on-error", "|| true", "set +e"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Surface visual gate must not contain bypass marker %q", forbidden)
		}
	}
}

func TestSurfaceVisualGateRejectsStaleReportDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash release script test")
	}

	root := surfaceVisualGateFakeRoot(t)
	reportRel := filepath.ToSlash(filepath.Join("reports", "surface-visual"))
	reportPath := filepath.Join(root, filepath.FromSlash(reportRel))
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportPath, "stale.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runSurfaceVisualGate(t, root, "--report-dir", reportRel)
	if err == nil {
		t.Fatalf("expected stale report-dir rejection\n%s", out)
	}
	for _, want := range []string{
		"surface_visual_gate: refusing to reuse non-empty report directory: " + reportRel,
		"surface_visual_gate: choose a fresh --report-dir so stale reports cannot be reused",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("stale visual report-dir output missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{
		"block-system-gate.sh",
		"surface-visual-diff",
		"go run",
		"stat ",
		"mkdir:",
		"find:",
		"No such file or directory",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf("visual gate should reject report dir before sub-gates or raw shell side effects:\n%s", out)
		}
	}
}

func surfaceVisualGateFakeRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	srcDir := filepath.Join(repoRoot(t), "scripts", "release", "surface")
	dstDir := filepath.Join(root, "scripts", "release", "surface")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"visual-gate.sh", "report-dir-guard.sh"} {
		if err := copyFile(filepath.Join(srcDir, name), filepath.Join(dstDir, name), 0o755); err != nil {
			t.Fatalf("copy %s: %v", name, err)
		}
	}
	return root
}

func runSurfaceVisualGate(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()

	cmdArgs := append([]string{"scripts/release/surface/visual-gate.sh"}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOTELEMETRY=off",
		"GOCACHE="+filepath.Join(root, ".cache", "go-build-surface-visual-gate-scriptstest"),
	)
	return cmd.CombinedOutput()
}
