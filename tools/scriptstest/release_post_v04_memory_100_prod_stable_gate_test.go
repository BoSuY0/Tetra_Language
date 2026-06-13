package scriptstest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleasePostV04Memory100ProdStableGateRunsStrictOrderedEvidence(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "post_v0_4", "memory-100-prod-stable-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Memory100 production stable gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/post_v0_4/memory-100-prod-stable-gate.sh [--report-dir DIR]",
		`source "$repo_root/scripts/release/surface/report-dir-guard.sh"`,
		`surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "memory100_prod_stable_gate:"`,
		`memory_production_dir="$report_dir_arg/memory-production"`,
		`ram_contract_dir="$report_dir_arg/ram-contract"`,
		`integrated_dir="$report_dir_arg/integrated"`,
		`aggregate_manifest_path="$report_dir/memory-100-prod-stable-manifest.json"`,
		`bash "$script_dir/memory-production-linux-x64-smoke.sh" --report-dir "$memory_production_dir"`,
		`bash "$script_dir/ram-contract-linux-x64-smoke.sh" --report-dir "$ram_contract_dir"`,
		`bash "$script_dir/memory-islands-surface-production-gate.sh" --report-dir "$integrated_dir"`,
		`tetra.memory-100.prod-stable.v1`,
		`memory-production/memory-release-manifest.json`,
		`ram-contract/ram-contract-release-manifest.json`,
		`memory100_verdict="MEMORY100_SCOPED_READY_LOCAL"`,
		`memory100_verdict="MEMORY100_SCOPED_READY_DIRTY"`,
		`"verdict": "$memory100_verdict"`,
		`raw-memory-contract/raw-memory-contract.json`,
		`allocation-lowering/allocation-lowering-report.json`,
		`proof-store/proof-store-summary.json`,
		`proof-transition/proof-transition-report.json`,
		`runtime-memory/runtime-memory-contract.json`,
		`memory-fuzz/memory-fuzz-oracle.json`,
		`leak-resource/leak-resource-report.json`,
		`integrated/memory-islands-surface-production-manifest.json`,
		`docs-manifest/claim-policy.json`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-memory-100-prod-stable --report-dir "$report_dir" --current-git-head "$git_head"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Memory100 production stable gate missing %q", want)
		}
	}

	assertOrderedFragments(t, text,
		`memory-production-linux-x64-smoke.sh`,
		`ram-contract-linux-x64-smoke.sh`,
		`memory-islands-surface-production-gate.sh`,
		`cat > "$aggregate_manifest_path" <<MANIFEST`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-memory-100-prod-stable --report-dir "$report_dir" --current-git-head "$git_head"`,
	)
	for _, forbidden := range []string{"continue-on-error", "|| true", "set +e"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Memory100 gate must not contain bypass marker %q", forbidden)
		}
	}
}

func TestReleasePostV04Memory100ProdStableGateRejectsStaleReportDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash release script test")
	}

	root := memory100GateFakeRoot(t)
	reportRel := filepath.ToSlash(filepath.Join("reports", "memory-100", "stale"))
	reportPath := filepath.Join(root, filepath.FromSlash(reportRel))
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportPath, "stale.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runMemory100Gate(t, root, "--report-dir", reportRel)
	if err == nil {
		t.Fatalf("expected stale report-dir rejection\n%s", out)
	}
	for _, want := range []string{
		"memory100_prod_stable_gate: refusing to reuse non-empty report directory: " + reportRel,
		"memory100_prod_stable_gate: choose a fresh --report-dir so stale reports cannot be reused",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("stale report-dir output missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{
		"memory-production-linux-x64-smoke.sh",
		"ram-contract-linux-x64-smoke.sh",
		"memory-islands-surface-production-gate.sh",
		"go run",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf("Memory100 gate should reject stale report-dir before sub-gates:\n%s", out)
		}
	}
}

func memory100GateFakeRoot(t *testing.T) string {
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
	for _, copy := range []struct {
		src string
		dst string
	}{
		{
			src: filepath.Join(repo, "scripts", "release", "post_v0_4", "memory-100-prod-stable-gate.sh"),
			dst: filepath.Join(root, "scripts", "release", "post_v0_4", "memory-100-prod-stable-gate.sh"),
		},
		{
			src: filepath.Join(repo, "scripts", "release", "surface", "report-dir-guard.sh"),
			dst: filepath.Join(root, "scripts", "release", "surface", "report-dir-guard.sh"),
		},
	} {
		if err := copyFile(copy.src, copy.dst, 0o755); err != nil {
			t.Fatalf("copy %s: %v", filepath.Base(copy.src), err)
		}
	}
	return root
}

func runMemory100Gate(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()
	cmdArgs := append([]string{"scripts/release/post_v0_4/memory-100-prod-stable-gate.sh"}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOTELEMETRY=off",
		"GOCACHE="+filepath.Join(root, ".cache", "go-build-memory100-scriptstest"),
		"GOTMPDIR="+filepath.Join(root, ".cache", "go-tmp-memory100-scriptstest"),
	)
	return cmd.CombinedOutput()
}
