package scriptstest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReleaseV10BinarySizeReportIsDeterministicForIdenticalBuilds(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release_v1_0_binary_size.sh"), filepath.Join(root, "scripts", "release_v1_0_binary_size.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
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
		cmd := exec.Command("bash", "scripts/release_v1_0_binary_size.sh", "--report", report)
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
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release_v1_0_binary_size.sh"), filepath.Join(root, "scripts", "release_v1_0_binary_size.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
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
	cmd := exec.Command("bash", "scripts/release_v1_0_binary_size.sh", "--report", reportPath)
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
