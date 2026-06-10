package scriptstest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleasePostV04RAMContractSmokeScriptRunsStrictValidators(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "post_v0_4", "ram-contract-linux-x64-smoke.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read RAM contract smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh [--report-dir DIR]",
		`source "$repo_root/scripts/release/surface/report-dir-guard.sh"`,
		`surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "ram_contract_gate:"`,
		`go run ./cli/cmd/tetra build --target linux-x64 --emit-ram-contract-report --emit-memory-report --emit-alloc-report`,
		`ram-contract-report.json`,
		`memory-grade-report.json`,
		`proof-store-summary.json`,
		`validation-pipeline-coverage.json`,
		`heap-blockers.json`,
		`copy-blockers.json`,
		`go run ./tools/cmd/validate-ram-contract-report --report "$report_dir/ram-contract-report.json"`,
		`go run ./tools/cmd/validate-memory-grade-report --report "$report_dir/memory-grade-report.json"`,
		`go run ./tools/cmd/validate-proof-store-summary --report "$report_dir/proof-store-summary.json"`,
		`go run ./tools/cmd/validate-validation-pipeline-coverage --report "$report_dir/validation-pipeline-coverage.json"`,
		`go run ./tools/cmd/validate-heap-blockers --report "$report_dir/heap-blockers.json"`,
		`go run ./tools/cmd/validate-copy-blockers --report "$report_dir/copy-blockers.json"`,
		`go run ./tools/cmd/ram-contract-fuzz-short --report-dir "$report_dir/fuzz" --git-head "$git_head"`,
		`go run ./tools/cmd/validate-ram-contract-fuzz-oracle --report "$report_dir/fuzz/ram-contract-fuzz-oracle.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-ram-contract-release --report-dir "$report_dir" --current-git-head "$git_head"`,
		`tetra.ram-contract.release-manifest.v1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("RAM contract smoke script missing %q", want)
		}
	}
	assertOrderedFragments(t, text,
		`go run ./cli/cmd/tetra build --target linux-x64 --emit-ram-contract-report`,
		`go run ./tools/cmd/validate-ram-contract-report`,
		`go run ./tools/cmd/ram-contract-fuzz-short`,
		`cat > "$manifest_path" <<MANIFEST`,
		`go run ./tools/cmd/validate-artifact-hashes --write`,
		`go run ./tools/cmd/validate-ram-contract-release`,
	)
}

func TestReleasePostV04RAMContractSmokeRejectsStaleReportDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash release script test")
	}
	root := ramContractGateFakeRoot(t)
	reportRel := filepath.ToSlash(filepath.Join("reports", "ram-contract"))
	reportPath := filepath.Join(root, filepath.FromSlash(reportRel))
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportPath, "stale.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runRAMContractGate(t, root, "--report-dir", reportRel)
	if err == nil {
		t.Fatalf("expected stale report-dir rejection\n%s", out)
	}
	if !strings.Contains(string(out), "ram_contract_gate: refusing to reuse non-empty report directory: "+reportRel) {
		t.Fatalf("stale report-dir output missing guard:\n%s", out)
	}
}

func ramContractGateFakeRoot(t *testing.T) string {
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
			src: filepath.Join(repo, "scripts", "release", "post_v0_4", "ram-contract-linux-x64-smoke.sh"),
			dst: filepath.Join(root, "scripts", "release", "post_v0_4", "ram-contract-linux-x64-smoke.sh"),
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

func runRAMContractGate(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()
	cmdArgs := append([]string{"scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh"}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOTELEMETRY=off",
		"GOCACHE="+filepath.Join(root, ".cache", "go-build-ram-scriptstest"),
		"GOTMPDIR="+filepath.Join(root, ".cache", "go-tmp-ram-scriptstest"),
	)
	return cmd.CombinedOutput()
}
