package scriptstest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV10ReproWorkflowLivesInVersionedReleaseScript(t *testing.T) {
	root := repoRoot(t)
	versionedPath := filepath.Join(root, "scripts", "release", "v1_0", "reproducible-build.sh")
	assertLegacyFileRemoved(t, "scripts/release_v1_0_repro.sh", "scripts/release/v1_0/reproducible-build.sh directly")
	versionedRaw, err := os.ReadFile(versionedPath)
	if err != nil {
		t.Fatalf("read versioned repro script: %v", err)
	}
	versionedText := string(versionedRaw)
	for _, want := range []string{
		"Usage: bash scripts/release/v1_0/reproducible-build.sh",
		`"schema": "tetra.reproducible-build-proof.v1alpha1"`,
		"./tetra build --target",
		"record_artifact_pair()",
	} {
		if !strings.Contains(versionedText, want) {
			t.Fatalf("scripts/release/v1_0/reproducible-build.sh missing %q", want)
		}
	}
	assertNoLegacyMention(t, versionedText, "scripts/release_v1_0_repro.sh", "scripts/release/v1_0/reproducible-build.sh")
}

func TestReleaseV10ReproRejectsMissingReportArgument(t *testing.T) {
	root := t.TempDir()
	copyReleaseV10ReproScripts(t, root)

	out, err := runReleaseV10Repro(t, root, "--report")
	if err == nil {
		t.Fatalf("expected missing --report argument rejection\n%s", out)
	}
	if !strings.Contains(string(out), "release/v1_0/reproducible-build: --report requires a path") {
		t.Fatalf("missing report argument output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
}

func TestReleaseV10ReproRejectsDirectoryReportBeforeBuild(t *testing.T) {
	root := releaseV10ReproOutputPathFakeRepo(t)
	reportPath := filepath.Join(root, "report-dir")
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runReleaseV10Repro(t, root, "--report", reportPath)
	if err == nil {
		t.Fatalf("expected directory report path rejection\n%s", out)
	}
	if !strings.Contains(string(out), "release/v1_0/reproducible-build: refusing to use directory report path: "+reportPath) {
		t.Fatalf("directory report output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	if _, err := os.Stat(filepath.Join(root, "tetra-build.log")); !os.IsNotExist(err) {
		t.Fatalf("directory report path should block before build, stat err = %v", err)
	}
}

func TestReleaseV10ReproAcceptsDashPrefixedReportPath(t *testing.T) {
	root := releaseV10ReproOutputPathFakeRepo(t)
	reportArg := "-repro.json"

	out, err := runReleaseV10Repro(t, root, "--report", reportArg)
	if err != nil {
		t.Fatalf("dash-prefixed report path should work: %v\n%s", err, out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	if _, err := os.Stat(filepath.Join(root, reportArg)); err != nil {
		t.Fatalf("dash-prefixed report was not written: %v\n%s", err, out)
	}
}

func TestReleaseV10ReproAllowsExistingRegularReportOverwrite(t *testing.T) {
	root := releaseV10ReproOutputPathFakeRepo(t)
	reportPath := filepath.Join(root, "repro.json")
	if err := os.WriteFile(reportPath, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runReleaseV10Repro(t, root, "--report", reportPath)
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

func TestReleaseV10ReproProofDoesNotChurnWallClockTimestamp(t *testing.T) {
	root := t.TempDir()
	copyReleaseV10ReproScripts(t, root)
	if err := os.MkdirAll(filepath.Join(root, "examples"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "ui_web_smoke.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
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
    case "$target" in
      wasm32-wasi)
        base="${out%.wasm}"
        printf 'ui:%s\n' "$target" >"$base.ui.json"
        ;;
      wasm32-web)
        base="${out%.wasm}"
        printf 'loader:%s:%s\n' "$target" "$(basename "$out")" >"$base.mjs"
        printf 'ui:%s\n' "$target" >"$base.ui.json"
        printf 'ui-module:%s\n' "$target" >"$base.ui.web.mjs"
        printf 'ui-html:%s\n' "$target" >"$base.ui.html"
        ;;
    esac
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
		cmd := exec.Command("bash", "scripts/release/v1_0/reproducible-build.sh", "--report", report)
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
		Source          string   `json:"source"`
		SourceSHA256    string   `json:"source_sha256"`
		TargetCount     int      `json:"target_count"`
		MatchedCount    int      `json:"matched_count"`
		MismatchedCount int      `json:"mismatched_count"`
		Targets         []string `json:"targets"`
		NativeTargets   []struct {
			Target string `json:"target"`
			Match  bool   `json:"match"`
		} `json:"native_targets"`
		Artifacts []struct {
			Target    string `json:"target"`
			Extension string `json:"extension"`
			Match     bool   `json:"match"`
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
	if proof.Source != "examples/ui_web_smoke.tetra" {
		t.Fatalf("source = %q", proof.Source)
	}
	if !strings.HasPrefix(proof.SourceSHA256, "sha256:") {
		t.Fatalf("source_sha256 missing sha256 prefix: %q", proof.SourceSHA256)
	}
	if proof.TargetCount != 5 || proof.MatchedCount != 10 || proof.MismatchedCount != 0 {
		t.Fatalf("unexpected match counts: %+v", proof)
	}
	if len(proof.NativeTargets) != 3 {
		t.Fatalf("native_targets = %#v, want linux/macos/windows target-aware entries", proof.NativeTargets)
	}
	seenNative := map[string]bool{}
	for _, artifact := range proof.NativeTargets {
		if !artifact.Match {
			t.Fatalf("native target %s did not match", artifact.Target)
		}
		seenNative[artifact.Target] = true
	}
	for _, target := range []string{"linux-x64", "macos-x64", "windows-x64"} {
		if !seenNative[target] {
			t.Fatalf("repro proof missing native target %s in native_targets: %#v", target, proof.NativeTargets)
		}
	}
	seen := map[string]bool{}
	seenExtensions := map[string]bool{}
	for _, artifact := range proof.Artifacts {
		if !artifact.Match {
			t.Fatalf("artifact %s did not match", artifact.Target)
		}
		seen[artifact.Target] = true
		seenExtensions[artifact.Target+" "+artifact.Extension] = true
	}
	for _, target := range []string{"linux-x64", "macos-x64", "windows-x64", "wasm32-wasi", "wasm32-web"} {
		if !seen[target] {
			t.Fatalf("repro proof missing target %s in artifacts: %#v", target, proof.Artifacts)
		}
	}
	for _, want := range []string{
		"wasm32-wasi .wasm",
		"wasm32-wasi .ui.json",
		"wasm32-web .wasm",
		"wasm32-web .mjs",
		"wasm32-web .ui.json",
		"wasm32-web .ui.web.mjs",
		"wasm32-web .ui.html",
	} {
		if !seenExtensions[want] {
			t.Fatalf("repro proof missing sidecar coverage %s in artifacts: %#v", want, proof.Artifacts)
		}
	}
}

func TestReleaseV10ReproProofFailsForNonDeterministicBuildOutput(t *testing.T) {
	root := t.TempDir()
	copyReleaseV10ReproScripts(t, root)
	if err := os.MkdirAll(filepath.Join(root, "examples"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "ui_web_smoke.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
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
	cmd := exec.Command("bash", "scripts/release/v1_0/reproducible-build.sh", "--report", report)
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

func TestReleaseV10ReproProofFailsForNonDeterministicSidecar(t *testing.T) {
	root := t.TempDir()
	copyReleaseV10ReproScripts(t, root)
	if err := os.MkdirAll(filepath.Join(root, "examples"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "ui_web_smoke.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
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
    if [[ "$target" == "wasm32-web" ]]; then
      counter="${REPRO_COUNTER_FILE:-counter.txt}"
      n=0
      if [[ -f "$counter" ]]; then
        n="$(cat "$counter")"
      fi
      n=$((n + 1))
      printf '%s' "$n" >"$counter"
      base="${out%.wasm}"
      printf 'loader:%s:%s\n' "$target" "$n" >"$base.mjs"
    fi
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
	cmd := exec.Command("bash", "scripts/release/v1_0/reproducible-build.sh", "--report", report)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "REPRO_COUNTER_FILE="+counterPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected reproducibility proof to fail for non-deterministic sidecar\n%s", out)
	}

	raw, readErr := os.ReadFile(report)
	if readErr != nil {
		t.Fatalf("read report: %v", readErr)
	}
	var proof struct {
		Status          string `json:"status"`
		MismatchedCount int    `json:"mismatched_count"`
		Artifacts       []struct {
			Target    string `json:"target"`
			Extension string `json:"extension"`
			Sidecar   bool   `json:"sidecar"`
			Match     bool   `json:"match"`
		} `json:"artifacts"`
	}
	if unmarshalErr := json.Unmarshal(raw, &proof); unmarshalErr != nil {
		t.Fatalf("unmarshal report: %v\n%s", unmarshalErr, string(raw))
	}
	if proof.Status != "fail" || proof.MismatchedCount == 0 {
		t.Fatalf("unexpected failure report: %+v\n%s", proof, string(raw))
	}
	for _, artifact := range proof.Artifacts {
		if artifact.Target == "wasm32-web" && artifact.Extension == ".mjs" && artifact.Sidecar && !artifact.Match {
			return
		}
	}
	t.Fatalf("missing mismatched wasm32-web .mjs sidecar in report:\n%s", string(raw))
}

func TestReleaseV10ReproProofFailsWhenExpectedUISidecarsAreMissing(t *testing.T) {
	root := t.TempDir()
	copyReleaseV10ReproScripts(t, root)
	if err := os.MkdirAll(filepath.Join(root, "examples"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "ui_web_smoke.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
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

	report := filepath.Join(root, "repro.json")
	cmd := exec.Command("bash", "scripts/release/v1_0/reproducible-build.sh", "--report", report)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected reproducibility proof to fail when expected UI sidecars are missing\n%s", out)
	}

	raw, readErr := os.ReadFile(report)
	if readErr != nil {
		t.Fatalf("read report: %v", readErr)
	}
	var proof struct {
		Status string `json:"status"`
	}
	if unmarshalErr := json.Unmarshal(raw, &proof); unmarshalErr != nil {
		t.Fatalf("unmarshal report: %v\n%s", unmarshalErr, string(raw))
	}
	if proof.Status == "pass" {
		t.Fatalf("repro proof passed despite missing expected UI sidecars:\n%s", string(raw))
	}
}

func copyReleaseV10ReproScripts(t *testing.T, root string) {
	t.Helper()
	for _, dir := range []string{
		"scripts",
		"scripts/release/v1_0",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "reproducible-build.sh"), filepath.Join(root, "scripts", "release", "v1_0", "reproducible-build.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func releaseV10ReproOutputPathFakeRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyReleaseV10ReproScripts(t, root)
	if err := os.MkdirAll(filepath.Join(root, "examples"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "ui_web_smoke.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
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
    case "$target" in
      wasm32-wasi)
        base="${out%.wasm}"
        printf 'ui:%s\n' "$target" >"$base.ui.json"
        ;;
      wasm32-web)
        base="${out%.wasm}"
        printf 'loader:%s\n' "$target" >"$base.mjs"
        printf 'ui:%s\n' "$target" >"$base.ui.json"
        printf 'ui-module:%s\n' "$target" >"$base.ui.web.mjs"
        printf 'ui-html:%s\n' "$target" >"$base.ui.html"
        ;;
    esac
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

func runReleaseV10Repro(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{"scripts/release/v1_0/reproducible-build.sh"}, args...)...)
	cmd.Dir = root
	return cmd.CombinedOutput()
}
