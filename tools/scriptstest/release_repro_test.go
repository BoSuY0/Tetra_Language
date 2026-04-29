package scriptstest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV10ReproProofDoesNotChurnWallClockTimestamp(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release_v1_0_repro.sh"), filepath.Join(root, "scripts", "release_v1_0_repro.sh"), 0o755); err != nil {
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

	first := filepath.Join(root, "first.json")
	second := filepath.Join(root, "second.json")
	for _, report := range []string{first, second} {
		cmd := exec.Command("bash", "scripts/release_v1_0_repro.sh", "--report", report)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("repro proof failed: %v\n%s", err, out)
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
	if strings.Contains(string(firstRaw), "generated_at") {
		t.Fatalf("proof should not include wall-clock generated_at:\n%s", string(firstRaw))
	}
	if string(firstRaw) != string(secondRaw) {
		t.Fatalf("proof should be byte-stable across identical runs\nfirst:\n%s\nsecond:\n%s", string(firstRaw), string(secondRaw))
	}

	var proof struct {
		CompilerVersion string   `json:"compiler_version"`
		SourceSHA256    string   `json:"source_sha256"`
		TargetCount     int      `json:"target_count"`
		MatchedCount    int      `json:"matched_count"`
		MismatchedCount int      `json:"mismatched_count"`
		Targets         []string `json:"targets"`
		Artifacts       []struct {
			Target string `json:"target"`
			Match  bool   `json:"match"`
		} `json:"artifacts"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(firstRaw, &proof); err != nil {
		t.Fatalf("unmarshal proof: %v", err)
	}
	if proof.Status != "pass" {
		t.Fatalf("proof status = %q", proof.Status)
	}
	if proof.CompilerVersion != "v0.2.0" {
		t.Fatalf("compiler_version = %q", proof.CompilerVersion)
	}
	if !strings.HasPrefix(proof.SourceSHA256, "sha256:") {
		t.Fatalf("source_sha256 missing sha256 prefix: %q", proof.SourceSHA256)
	}
	if proof.TargetCount != 5 || proof.MatchedCount != 5 || proof.MismatchedCount != 0 {
		t.Fatalf("unexpected match counts: %+v", proof)
	}
	seen := map[string]bool{}
	for _, artifact := range proof.Artifacts {
		if !artifact.Match {
			t.Fatalf("artifact %s did not match", artifact.Target)
		}
		seen[artifact.Target] = true
	}
	for _, target := range []string{"linux-x64", "macos-x64", "windows-x64", "wasm32-wasi", "wasm32-web"} {
		if !seen[target] {
			t.Fatalf("repro proof missing target %s in artifacts: %#v", target, proof.Artifacts)
		}
	}
}

func TestReleaseV10ReproProofFailsForNonDeterministicBuildOutput(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release_v1_0_repro.sh"), filepath.Join(root, "scripts", "release_v1_0_repro.sh"), 0o755); err != nil {
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
    counter="${REPRO_COUNTER_FILE:-counter.txt}"
    n=0
    if [[ -f "$counter" ]]; then
      n="$(cat "$counter")"
    fi
    n=$((n + 1))
    printf '%s' "$n" >"$counter"
    printf 'artifact:%s:%s\n' "$target" "$n" >"$out"
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

	report := filepath.Join(root, "repro.json")
	counterPath := filepath.Join(root, "counter.txt")
	cmd := exec.Command("bash", "scripts/release_v1_0_repro.sh", "--report", report)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "REPRO_COUNTER_FILE="+counterPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected reproducibility proof to fail for non-deterministic build\n%s", out)
	}

	raw, readErr := os.ReadFile(report)
	if readErr != nil {
		t.Fatalf("read report: %v", readErr)
	}
	var proof struct {
		Status          string `json:"status"`
		MismatchedCount int    `json:"mismatched_count"`
	}
	if unmarshalErr := json.Unmarshal(raw, &proof); unmarshalErr != nil {
		t.Fatalf("unmarshal report: %v\n%s", unmarshalErr, string(raw))
	}
	if proof.Status != "fail" || proof.MismatchedCount == 0 {
		t.Fatalf("unexpected failure report: %+v\n%s", proof, string(raw))
	}
}
