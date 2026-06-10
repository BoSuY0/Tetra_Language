package scriptstest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleasePostV04MemoryProductionSmokeScriptRunsExecutableValidator(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "post_v0_4", "memory-production-linux-x64-smoke.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read memory production smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh [--report-dir DIR]",
		`memory-production-linux-x64.json`,
		`go run ./tools/cmd/memory-production-smoke`,
		`go run ./tools/cmd/validate-memory-production`,
		`targets_path="$report_dir/targets.json"`,
		`go run ./cli/cmd/tetra targets --format=json > "$targets_path"`,
		`go run ./tools/cmd/validate-targets --report "$targets_path"`,
		`memory_fuzz_dir="$report_dir/memory-fuzz-tier1"`,
		`go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir "$memory_fuzz_dir"`,
		`go run ./tools/cmd/validate-memory-fuzz-oracle --report "$memory_fuzz_dir/memory-fuzz-oracle.json" --artifact-dir "$memory_fuzz_dir"`,
		`ram_contract_dir="$report_dir/ram-contract"`,
		`bash "$script_dir/ram-contract-linux-x64-smoke.sh" --report-dir "$ram_contract_dir"`,
		`ram-contract/ram-contract-report.json`,
		`tetra.ram-contract-report.v1`,
		`ram-contract/fuzz/ram-contract-fuzz-oracle.json`,
		`memory_release_manifest_path="$report_dir/memory-release-manifest.json"`,
		`git_head="$(git rev-parse --verify HEAD)"`,
		`generated_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"`,
		`memory-release-manifest.json`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-memory-production --report "$report_path" --manifest "$memory_release_manifest_path" --report-dir "$report_dir" --current-git-head "$git_head"`,
		`targets.json`,
		`$memory_fuzz_dir/memory-fuzz-oracle.json`,
		`$memory_fuzz_dir/summary.json`,
		`artifact-hashes.json`,
		`memory production release manifest: $memory_release_manifest_path`,
		`tetra.memory.production.v1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("memory production smoke script missing %q", want)
		}
	}
	targetsIdx := strings.Index(text, `go run ./tools/cmd/validate-targets --report "$targets_path"`)
	fuzzIdx := strings.Index(text, `go run ./tools/cmd/validate-memory-fuzz-oracle --report "$memory_fuzz_dir/memory-fuzz-oracle.json" --artifact-dir "$memory_fuzz_dir"`)
	hashIdx := strings.Index(text, `go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`)
	if targetsIdx < 0 || hashIdx < 0 || targetsIdx > hashIdx {
		t.Fatalf("memory production smoke script must validate targets before writing artifact hashes")
	}
	if fuzzIdx < 0 || hashIdx < 0 || fuzzIdx > hashIdx {
		t.Fatalf("memory production smoke script must validate memory fuzz Tier 1 artifacts before writing artifact hashes")
	}
	ramIdx := strings.Index(text, `bash "$script_dir/ram-contract-linux-x64-smoke.sh" --report-dir "$ram_contract_dir"`)
	if ramIdx < 0 || hashIdx < 0 || ramIdx > hashIdx {
		t.Fatalf("memory production smoke script must run RAM contract gate before writing artifact hashes")
	}
	manifestWriteIdx := strings.Index(text, `cat > "$memory_release_manifest_path" <<MANIFEST`)
	if manifestWriteIdx < 0 || hashIdx < 0 || manifestWriteIdx > hashIdx {
		t.Fatalf("memory production smoke script must write release manifest before writing artifact hashes")
	}
	hashValidateIdx := strings.Index(text, `go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`)
	manifestValidateIdx := strings.Index(text, `go run ./tools/cmd/validate-memory-production --report "$report_path" --manifest "$memory_release_manifest_path" --report-dir "$report_dir" --current-git-head "$git_head"`)
	if hashValidateIdx < 0 || manifestValidateIdx < 0 || hashValidateIdx > manifestValidateIdx {
		t.Fatalf("memory production smoke script must validate artifact hashes before validating release manifest provenance")
	}
}

func TestReleasePostV04MemoryProductionSmokeRejectsStaleReportDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash release script test")
	}

	root := repoRoot(t)
	reportDir := filepath.Join(t.TempDir(), "memory-report")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("mkdir report dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "stale.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write stale artifact: %v", err)
	}

	out, err := runMemoryProductionSmokeScript(t, root, "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected stale report-dir rejection\n%s", out)
	}
	for _, want := range []string{
		"refusing to reuse non-empty report directory: " + reportDir,
		"choose a fresh --report-dir so stale reports cannot be reused",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("stale report-dir output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "memory production linux-x64 smoke report:") {
		t.Fatalf("script should reject stale report-dir before writing success output:\n%s", out)
	}
}

func TestReleasePostV04MemoryProductionSmokeRejectsSymlinkReportDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink release script test")
	}

	root := repoRoot(t)
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	reportDir := filepath.Join(tempDir, "memory-report-link")
	if err := os.Symlink(targetDir, reportDir); err != nil {
		t.Fatalf("create report-dir symlink: %v", err)
	}

	out, err := runMemoryProductionSmokeScript(t, root, "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected symlink report-dir rejection\n%s", out)
	}
	for _, want := range []string{
		"refusing to use symlink report directory: " + reportDir,
		"choose a real fresh --report-dir",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("symlink report-dir output missing %q:\n%s", want, out)
		}
	}
}

func runMemoryProductionSmokeScript(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()
	script := filepath.Join(root, "scripts", "release", "post_v0_4", "memory-production-linux-x64-smoke.sh")
	cmdArgs := append([]string{script}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = root
	return cmd.CombinedOutput()
}

func TestReleasePostV04ScopeDocsAdvertiseMemoryProductionSmokeScript(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "docs", "release", "post_v0_4_linux_x64_memory_parallel_ui_scope.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read post-v0.4 scope docs: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir <dir>`,
		`go run ./tools/cmd/memory-production-smoke --report <path>`,
		`go run ./tools/cmd/validate-memory-production --report <path>`,
		`go run ./cli/cmd/tetra targets --format=json > <dir>/targets.json`,
		`go run ./tools/cmd/validate-targets --report <dir>/targets.json`,
		`go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir <dir>/memory-fuzz-tier1`,
		`go run ./tools/cmd/validate-memory-fuzz-oracle --report <dir>/memory-fuzz-tier1/memory-fuzz-oracle.json --artifact-dir <dir>/memory-fuzz-tier1`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest <dir>/artifact-hashes.json`,
		`go run ./tools/cmd/validate-memory-production --report <dir>/memory-production-linux-x64.json --manifest <dir>/memory-release-manifest.json --report-dir <dir> --current-git-head <head>`,
		`memory-release-manifest.json`,
		`fresh --report-dir`,
		`memory-fuzz-oracle.json`,
		`summary.json`,
		`not an exhaustive fuzz proof`,
		`quick evidence is not full, stabilization, nightly, or release proof`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("post-v0.4 scope docs missing %q", want)
		}
	}
}
