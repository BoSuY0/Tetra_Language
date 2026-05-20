package scriptstest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV10BinarySizeWorkflowLivesInVersionedReleaseScript(t *testing.T) {
	root := repoRoot(t)
	versionedPath := filepath.Join(root, "scripts", "release", "v1_0", "binary-size.sh")
	assertLegacyFileRemoved(t, "scripts/release_v1_0_binary_size.sh", "scripts/release/v1_0/binary-size.sh directly")
	versionedRaw, err := os.ReadFile(versionedPath)
	if err != nil {
		t.Fatalf("read versioned binary-size script: %v", err)
	}
	versionedText := string(versionedRaw)
	for _, want := range []string{
		"Usage: bash scripts/release/v1_0/binary-size.sh",
		`"schema": "tetra.binary-size-thresholds.v1alpha1"`,
		"./tetra build --target",
		"soft_limit()",
		"hard_limit()",
	} {
		if !strings.Contains(versionedText, want) {
			t.Fatalf("scripts/release/v1_0/binary-size.sh missing %q", want)
		}
	}
	assertNoLegacyMention(t, versionedText, "scripts/release_v1_0_binary_size.sh", "scripts/release/v1_0/binary-size.sh")
}

func TestReleaseV10BinarySizeRejectsMissingReportArgument(t *testing.T) {
	root := t.TempDir()
	copyReleaseV10BinarySizeScripts(t, root)

	out, err := runReleaseV10BinarySize(t, root, "--report")
	if err == nil {
		t.Fatalf("expected missing --report argument rejection\n%s", out)
	}
	if !strings.Contains(string(out), "release/v1_0/binary-size: --report requires a path") {
		t.Fatalf("missing report argument output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
}

func TestReleaseV10BinarySizeRejectsDirectoryReportBeforeBuild(t *testing.T) {
	root := releaseV10BinarySizeOutputPathFakeRepo(t)
	reportPath := filepath.Join(root, "report-dir")
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runReleaseV10BinarySize(t, root, "--report", reportPath)
	if err == nil {
		t.Fatalf("expected directory report path rejection\n%s", out)
	}
	if !strings.Contains(string(out), "release/v1_0/binary-size: refusing to use directory report path: "+reportPath) {
		t.Fatalf("directory report output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	if _, err := os.Stat(filepath.Join(root, "tetra-build.log")); !os.IsNotExist(err) {
		t.Fatalf("directory report path should block before build, stat err = %v", err)
	}
}

func TestReleaseV10BinarySizeAcceptsDashPrefixedReportPath(t *testing.T) {
	root := releaseV10BinarySizeOutputPathFakeRepo(t)
	reportArg := "-binary-size.json"

	out, err := runReleaseV10BinarySize(t, root, "--report", reportArg)
	if err != nil {
		t.Fatalf("dash-prefixed report path should work: %v\n%s", err, out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	if _, err := os.Stat(filepath.Join(root, reportArg)); err != nil {
		t.Fatalf("dash-prefixed report was not written: %v\n%s", err, out)
	}
}

func TestReleaseV10BinarySizeAllowsExistingRegularReportOverwrite(t *testing.T) {
	root := releaseV10BinarySizeOutputPathFakeRepo(t)
	reportPath := filepath.Join(root, "size.json")
	if err := os.WriteFile(reportPath, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runReleaseV10BinarySize(t, root, "--report", reportPath)
	if err != nil {
		t.Fatalf("existing regular report should remain overwritable: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "old\n" {
		t.Fatalf("existing report was not overwritten:\n%s", string(raw))
	}
}

func TestReleaseV10BinarySizeReportIsDeterministicForIdenticalBuilds(t *testing.T) {
	root := t.TempDir()
	copyReleaseV10BinarySizeScripts(t, root)
	if err := os.MkdirAll(filepath.Join(root, "examples"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "flow_struct_smoke.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tetra := `#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-}"
shift || true
case "$cmd" in
  version)
    echo "v0.2.0"
    ;;
  build)
    target=""
    out=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --target)
          target="$2"
          shift 2
          ;;
        -o)
          out="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    case "$target" in
      linux-x64) size=1024 ;;
      macos-x64) size=2048 ;;
      windows-x64) size=3072 ;;
      wasm32-wasi) size=512 ;;
      wasm32-web) size=256 ;;
      *) size=64 ;;
    esac
    dd if=/dev/zero of="$out" bs=1 count="$size" status=none
    ;;
  *)
    echo "unexpected tetra command: $cmd $*" >&2
    exit 2
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(root, "tetra"), []byte(tetra), 0o755); err != nil {
		t.Fatal(err)
	}

	first := filepath.Join(root, "first.json")
	second := filepath.Join(root, "second.json")
	for _, report := range []string{first, second} {
		cmd := exec.Command("bash", "scripts/release/v1_0/binary-size.sh", "--report", report)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("binary size report failed: %v\n%s", err, out)
		}
	}
	firstRaw, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondRaw, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstRaw) != string(secondRaw) {
		t.Fatalf("binary size report should be byte-stable across identical runs\nfirst:\n%s\nsecond:\n%s", string(firstRaw), string(secondRaw))
	}

	var report struct {
		Status         string `json:"status"`
		TargetCount    int    `json:"target_count"`
		PassCount      int    `json:"pass_count"`
		WarnCount      int    `json:"warn_count"`
		FailCount      int    `json:"fail_count"`
		TotalSizeBytes int    `json:"total_size_bytes"`
		MaxSizeBytes   int    `json:"max_size_bytes"`
		Targets        []string
	}
	if err := json.Unmarshal(firstRaw, &report); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if report.Status != "pass" {
		t.Fatalf("status = %q", report.Status)
	}
	if report.TargetCount != 5 || report.PassCount != 5 || report.WarnCount != 0 || report.FailCount != 0 {
		t.Fatalf("unexpected counters: %+v", report)
	}
	if report.TotalSizeBytes != (1024 + 2048 + 3072 + 512 + 256) {
		t.Fatalf("total_size_bytes = %d", report.TotalSizeBytes)
	}
	if report.MaxSizeBytes != 3072 {
		t.Fatalf("max_size_bytes = %d", report.MaxSizeBytes)
	}
}

func TestReleaseV10BinarySizeReportFailsOnHardLimitBreach(t *testing.T) {
	root := t.TempDir()
	copyReleaseV10BinarySizeScripts(t, root)
	if err := os.MkdirAll(filepath.Join(root, "examples"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "flow_struct_smoke.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tetra := `#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-}"
shift || true
case "$cmd" in
  version)
    echo "v0.2.0"
    ;;
  build)
    target=""
    out=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --target)
          target="$2"
          shift 2
          ;;
        -o)
          out="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    size=128
    if [[ "$target" == "linux-x64" ]]; then
      size=4194305
    fi
    dd if=/dev/zero of="$out" bs=1 count="$size" status=none
    ;;
  *)
    echo "unexpected tetra command: $cmd $*" >&2
    exit 2
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(root, "tetra"), []byte(tetra), 0o755); err != nil {
		t.Fatal(err)
	}

	reportPath := filepath.Join(root, "size.json")
	cmd := exec.Command("bash", "scripts/release/v1_0/binary-size.sh", "--report", reportPath)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected hard-limit failure\n%s", out)
	}
	raw, readErr := os.ReadFile(reportPath)
	if readErr != nil {
		t.Fatalf("read report: %v", readErr)
	}
	var report struct {
		Status    string `json:"status"`
		FailCount int    `json:"fail_count"`
	}
	if unmarshalErr := json.Unmarshal(raw, &report); unmarshalErr != nil {
		t.Fatalf("unmarshal report: %v\n%s", unmarshalErr, string(raw))
	}
	if report.Status != "fail" || report.FailCount == 0 {
		t.Fatalf("unexpected failure report: %+v\n%s", report, string(raw))
	}
}

func copyReleaseV10BinarySizeScripts(t *testing.T, root string) {
	t.Helper()
	for _, dir := range []string{
		"scripts",
		"scripts/release/v1_0",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "binary-size.sh"), filepath.Join(root, "scripts", "release", "v1_0", "binary-size.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func releaseV10BinarySizeOutputPathFakeRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyReleaseV10BinarySizeScripts(t, root)
	if err := os.MkdirAll(filepath.Join(root, "examples"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "flow_struct_smoke.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tetra := `#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-}"
shift || true
case "$cmd" in
  version)
    echo "v0.2.0"
    ;;
  build)
    target=""
    out=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --target)
          target="$2"
          shift 2
          ;;
        -o)
          out="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    printf '%s\n' "$target" >>tetra-build.log
    printf 'artifact:%s\n' "$target" >"$out"
    ;;
  *)
    echo "unexpected tetra command: $cmd $*" >&2
    exit 2
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(root, "tetra"), []byte(tetra), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func runReleaseV10BinarySize(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{"scripts/release/v1_0/binary-size.sh"}, args...)...)
	cmd.Dir = root
	return cmd.CombinedOutput()
}
