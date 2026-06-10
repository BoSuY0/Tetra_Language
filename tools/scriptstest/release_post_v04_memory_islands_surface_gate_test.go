package scriptstest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleasePostV04MemoryIslandsSurfaceProductionGateRunsStrictOrderedEvidence(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "post_v0_4", "memory-islands-surface-production-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Memory/Islands/Surface integrated production gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/post_v0_4/memory-islands-surface-production-gate.sh [--report-dir DIR]",
		`source "$repo_root/scripts/release/surface/report-dir-guard.sh"`,
		`surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "memory_islands_surface_gate:"`,
		`memory_report_dir="$report_dir_arg/memory"`,
		`surface_release_dir="$report_dir_arg/surface-release-v1"`,
		`surface_experimental_dir="$report_dir_arg/surface-experimental-regression"`,
		`safe_view_dir="$report_dir_arg/safe-view-lifetime"`,
		`surface_api_dir="$report_dir_arg/surface-api-stability-v1"`,
		`islands_debug_report="$report_dir/islands-debug-smoke.json"`,
		`bash "$script_dir/memory-production-linux-x64-smoke.sh" --report-dir "$memory_report_dir"`,
		`go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --islands-debug --report "$islands_debug_report"`,
		`go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$islands_debug_report"`,
		`bash scripts/release/surface/release-gate.sh --report-dir "$surface_release_dir"`,
		`bash scripts/release/surface/gate.sh --report-dir "$surface_experimental_dir"`,
		`bash scripts/release/safe-view-lifetime/gate.sh --report-dir "$safe_view_dir"`,
		`bash scripts/release/surface/api-stability-gate.sh --report-dir "$surface_api_dir"`,
		`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`,
		`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`,
		`memory-islands-surface-production-manifest.json`,
		`tetra.memory-islands-surface.production-gate.v1`,
		`tetra.memory.production.v1`,
		`memory/ram-contract/ram-contract-report.json`,
		`tetra.ram-contract-report.v1`,
		`memory/ram-contract/fuzz/ram-contract-fuzz-oracle.json`,
		`tetra.island.proof.v1`,
		`tetra.island-proof-fuzz-summary.v1`,
		`tetra.surface.release.v1`,
		`tetra.safe-view-lifetime.gate.v1`,
		`tetra.surface.api-stability.v1`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-memory-islands-surface-production --report-dir "$report_dir" --current-git-head "$git_head"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("integrated production gate missing %q", want)
		}
	}

	assertOrderedFragments(t, text,
		`memory-production-linux-x64-smoke.sh`,
		`go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --islands-debug`,
		`scripts/release/surface/release-gate.sh`,
		`scripts/release/surface/gate.sh`,
		`scripts/release/safe-view-lifetime/gate.sh`,
		`scripts/release/surface/api-stability-gate.sh`,
		`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`,
		`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`,
		`cat > "$integrated_manifest_path" <<MANIFEST`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-memory-islands-surface-production --report-dir "$report_dir" --current-git-head "$git_head"`,
	)
	for _, forbidden := range []string{
		"continue-on-error",
		"|| true",
		"set +e",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("integrated production gate must not contain bypass marker %q", forbidden)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, `"command":`) && strings.Contains(line, "json_string") {
			t.Fatalf("integrated manifest command lines must not embed JSON string literals inside JSON strings: %s", line)
		}
	}
}

func TestReleasePostV04MemoryIslandsSurfaceProductionGateRejectsStaleReportDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash release script test")
	}

	root := integratedGateFakeRoot(t)
	reportRel := filepath.ToSlash(filepath.Join("reports", "integrated"))
	reportPath := filepath.Join(root, filepath.FromSlash(reportRel))
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportPath, "stale.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runIntegratedGate(t, root, "--report-dir", reportRel)
	if err == nil {
		t.Fatalf("expected stale report-dir rejection\n%s", out)
	}
	for _, want := range []string{
		"memory_islands_surface_gate: refusing to reuse non-empty report directory: " + reportRel,
		"memory_islands_surface_gate: choose a fresh --report-dir so stale reports cannot be reused",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("stale report-dir output missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{
		"memory-production-linux-x64-smoke.sh",
		"surface/release-gate.sh",
		"safe-view-lifetime/gate.sh",
		"go run",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf("integrated gate should reject stale report-dir before sub-gates:\n%s", out)
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

func integratedGateFakeRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repo := repoRoot(t)
	for _, dir := range []string{
		filepath.Join("scripts", "release", "post_v0_4"),
		filepath.Join("scripts", "release", "surface"),
	} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	copies := []struct {
		src string
		dst string
	}{
		{
			src: filepath.Join(repo, "scripts", "release", "post_v0_4", "memory-islands-surface-production-gate.sh"),
			dst: filepath.Join(root, "scripts", "release", "post_v0_4", "memory-islands-surface-production-gate.sh"),
		},
		{
			src: filepath.Join(repo, "scripts", "release", "surface", "report-dir-guard.sh"),
			dst: filepath.Join(root, "scripts", "release", "surface", "report-dir-guard.sh"),
		},
	}
	for _, copy := range copies {
		if err := copyFile(copy.src, copy.dst, 0o755); err != nil {
			t.Fatalf("copy %s: %v", filepath.Base(copy.src), err)
		}
	}
	return root
}

func runIntegratedGate(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()

	cmdArgs := append([]string{"scripts/release/post_v0_4/memory-islands-surface-production-gate.sh"}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOTELEMETRY=off",
		"GOCACHE="+filepath.Join(root, ".cache", "go-build-integrated-scriptstest"),
		"GOTMPDIR="+filepath.Join(root, ".cache", "go-tmp-integrated-scriptstest"),
	)
	return cmd.CombinedOutput()
}
