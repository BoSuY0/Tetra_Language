package surface_gates

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSurfaceBlockSystemGateRunsStrictOrderedEvidence(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "surface", "block-system-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Surface Block-system gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/block-system-gate.sh [--report-dir DIR]",
		`source "$script_dir/report-dir-guard.sh"`,
		`surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_block_system_gate:"`,
		`headless_report_dir="$report_dir_arg/headless"`,
		`linux_report_dir="$report_dir_arg/linux-x64-real-window"`,
		`wasm_report_dir="$report_dir_arg/wasm32-web-browser-canvas"`,
		`bash scripts/release/surface/surface-headless-block-system-smoke.sh --report-dir "$headless_report_dir"`,
		`bash scripts/release/surface/surface-linux-x64-real-window-block-system-smoke.sh --report-dir "$linux_report_dir"`,
		`bash scripts/release/surface/surface-wasm32-web-browser-canvas-block-system-smoke.sh --report-dir "$wasm_report_dir"`,
		`go run ./tools/cmd/validate-surface-block-report --report "$report_dir/headless/surface-headless-block-system.json" --same-commit "$git_head"`,
		`go run ./tools/cmd/validate-surface-block-report --report "$report_dir/linux-x64-real-window/surface-block-system-linux-x64.json" --same-commit "$git_head"`,
		`go run ./tools/cmd/validate-surface-block-report --report "$report_dir/wasm32-web-browser-canvas/surface-block-system-wasm32-web.json" --same-commit "$git_head"`,
		`surface-block-system-gate-summary.json`,
		`tetra.surface.block-system.gate.v1`,
		`surface-block-examples.json`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Surface Block-system gate missing %q", want)
		}
	}

	assertOrderedFragments(
		t,
		text,
		`surface-headless-block-system-smoke.sh`,
		`validate-surface-block-report --report "$report_dir/headless/surface-headless-block-system.json"`,
		`surface-linux-x64-real-window-block-system-smoke.sh`,
		`validate-surface-block-report --report "$report_dir/linux-x64-real-window/surface-block-system-linux-x64.json"`,
		`surface-wasm32-web-browser-canvas-block-system-smoke.sh`,
		`validate-surface-block-report --report "$report_dir/wasm32-web-browser-canvas/surface-block-system-wasm32-web.json"`,
		`cat > "$summary_path" <<JSON`,
		`validate-artifact-hashes --write --root "$report_dir"`,
		`validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
	)
	for _, forbidden := range []string{"continue-on-error", "|| true", "set +e"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Surface Block-system gate must not contain bypass marker %q", forbidden)
		}
	}
}

func TestSurfaceReleaseGateRequiresBlockSystemSubgateBeforeSummary(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "surface", "release-gate.sh"))
	if err != nil {
		t.Fatalf("read Surface release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`report_dir_arg="${report_dir%/}"`,
		`block_system_report_dir="$report_dir_arg/block-system"`,
		`bash scripts/release/surface/block-system-gate.sh --report-dir "$block_system_report_dir"`,
		`"block_system": "block-system"`,
		`"block_system_gate": "tetra.surface.block-system.gate.v1"`,
		`"block-system/surface-block-system-gate-summary.json"`,
		`"block-system/headless/surface-headless-block-system.json"`,
		`"block-system/headless/surface-block-examples.json"`,
		`"block-system/linux-x64-real-window/surface-block-system-linux-x64.json"`,
		`"block-system/wasm32-web-browser-canvas/surface-block-system-wasm32-web.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Surface release gate missing Block-system release evidence detail %q", want)
		}
	}
	assertOrderedFragments(
		t,
		text,
		`surface-wasm32-web-release-accessibility-smoke.sh`,
		`block-system-gate.sh --report-dir "$block_system_report_dir"`,
		`summary_path="$report_dir/surface-release-summary.json"`,
		`validate-surface-runtime --report "$report_dir/surface-release-summary.json" --release surface-v1`,
	)
}

func TestSurfaceBlockSystemGateRejectsStaleReportDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash release script test")
	}

	root := surfaceBlockSystemGateFakeRoot(t)
	reportRel := filepath.ToSlash(filepath.Join("reports", "surface-block-system"))
	reportPath := filepath.Join(root, filepath.FromSlash(reportRel))
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(reportPath, "stale.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	out, err := runSurfaceBlockSystemGate(t, root, "--report-dir", reportRel)
	if err == nil {
		t.Fatalf("expected stale report-dir rejection\n%s", out)
	}
	for _, want := range []string{
		"surface_block_system_gate: refusing to reuse non-empty report directory: " + reportRel,
		"surface_block_system_gate: choose a fresh --report-dir so stale reports cannot be reused",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("stale Block-system report-dir output missing %q:\n%s", want, out)
		}
	}
	assertSurfaceBlockSystemGateRejectedBeforeSideEffects(t, out)
}

func TestSurfaceBlockSystemGateRejectsSymlinkReportDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash release script test")
	}

	root := surfaceBlockSystemGateFakeRoot(t)
	targetDir := filepath.Join(root, "reports", "surface-block-stale-target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	reportRel := filepath.ToSlash(filepath.Join("reports", "surface-block-link"))
	reportPath := filepath.Join(root, filepath.FromSlash(reportRel))
	if err := os.Symlink(targetDir, reportPath); err != nil {
		t.Fatalf("create report-dir symlink: %v", err)
	}

	out, err := runSurfaceBlockSystemGate(t, root, "--report-dir", reportRel)
	if err == nil {
		t.Fatalf("expected symlink report-dir rejection\n%s", out)
	}
	for _, want := range []string{
		"surface_block_system_gate: refusing to use symlink report directory: " + reportRel,
		"surface_block_system_gate: choose a real fresh --report-dir",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("symlink Block-system report-dir output missing %q:\n%s", want, out)
		}
	}
	assertSurfaceBlockSystemGateRejectedBeforeSideEffects(t, out)
}

func TestSurfaceReleaseReadinessCIUploadsBlockSystemReports(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	section := workflowJobSection(string(raw), "surface-release-readiness-linux:")
	for _, want := range []string{
		"surface-release-readiness-linux:",
		"name: Surface product gate",
		"bash scripts/release/surface/product-gate.sh --report-dir reports/surface-product-v1",
		"reports/surface-product-v1/block-system",
	} {
		if !strings.Contains(section, want) {
			t.Fatalf("Surface release readiness job missing Block-system CI detail %q", want)
		}
	}
	if strings.Contains(section, "continue-on-error") {
		t.Fatalf("Surface release readiness Block-system evidence must not use continue-on-error")
	}
}

func surfaceBlockSystemGateFakeRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	srcDir := filepath.Join(repoRoot(t), "scripts", "release", "surface")
	dstDir := filepath.Join(root, "scripts", "release", "surface")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"block-system-gate.sh", "report-dir-guard.sh"} {
		if err := copyFile(filepath.Join(srcDir, name), filepath.Join(dstDir, name), 0o755); err != nil {
			t.Fatalf("copy %s: %v", name, err)
		}
	}
	return root
}

func runSurfaceBlockSystemGate(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()

	cmdArgs := append([]string{"scripts/release/surface/block-system-gate.sh"}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOTELEMETRY=off",
		"GOCACHE="+filepath.Join(root, ".cache", "go-build-surface-block-gate-scriptstest"),
	)
	return cmd.CombinedOutput()
}

func assertSurfaceBlockSystemGateRejectedBeforeSideEffects(t *testing.T, out []byte) {
	t.Helper()

	for _, forbidden := range []string{
		"surface-headless-block-system-smoke.sh",
		"surface-runtime-smoke",
		"go run",
		"stat ",
		"mkdir:",
		"find:",
		"No such file or directory",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf(
				"Block-system gate should reject report dir before sub-gates or raw shell side effects:\n%s",
				out,
			)
		}
	}
}
