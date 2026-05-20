package scriptstest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func releaseV030FakeRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts", "dev"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts", "ci"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts", "release", "v1_0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts", "release", "v0_3_0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release/v0_3_0/gate.sh"), filepath.Join(root, "scripts", "release/v0_3_0/gate.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release", "v0_3_0", "security-review.sh"), filepath.Join(root, "scripts", "release", "v0_3_0", "security-review.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	bootstrap := `#!/usr/bin/env bash
set -euo pipefail
exit 0
`
	if err := os.WriteFile(filepath.Join(root, "scripts", "dev", "bootstrap.sh"), []byte(bootstrap), 0o755); err != nil {
		t.Fatal(err)
	}
	tetra := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "version" ]]; then
  echo "v0.2.0"
  exit 0
fi
echo "unexpected tetra command: $*" >&2
exit 2
`
	if err := os.WriteFile(filepath.Join(root, "tetra"), []byte(tetra), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func releaseV030RunnableFakeRepo(t *testing.T, unstableSeedRows []string) string {
	t.Helper()
	root := releaseV030FakeRepo(t)
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tetra"), []byte(`#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "version" ]]; then
  echo "v0.3.0"
  exit 0
fi
echo "unexpected tetra command: $*" >&2
exit 2
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "t"), []byte(`#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "version" ]]; then
  echo "v0.3.0"
  exit 0
fi
echo "unexpected t command: $*" >&2
exit 2
`), 0o755); err != nil {
		t.Fatal(err)
	}
	realGo, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("look up go: %v", err)
	}
	fakeGo := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "run" && "${2:-}" == */validate_residual_risks.go ]]; then
  exec ` + shellSingleQuote(realGo) + ` "$@"
fi
exit 0
`
	if err := os.WriteFile(filepath.Join(root, "bin", "go"), []byte(fakeGo), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "git"), []byte(`#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "rev-parse" && "${2:-}" == "--short" && "${3:-}" == "HEAD" ]]; then
  echo "fake-head-for-release-v030-test"
  exit 0
fi
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	fuzzNightly := `#!/usr/bin/env bash
set -euo pipefail
out_dir=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --out-dir)
      out_dir="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
if [[ -z "$out_dir" ]]; then
  echo "missing --out-dir" >&2
  exit 2
fi
mkdir -p "$out_dir/logs"
printf '{"status":"pass","artifacts":{"unstable_seed_log":"%s"}}\n' "$out_dir/unstable-seeds.md" >"$out_dir/summary.json"
printf '# Fuzz Nightly Summary\n' >"$out_dir/summary.md"
{
  printf '# Unstable Fuzz Seeds\n\n'
  printf '| package | fuzz target | seed/crasher path | status | owner | next command |\n'
  printf '| --- | --- | --- | --- | --- | --- |\n'
`
	for _, row := range unstableSeedRows {
		fuzzNightly += "  printf '%s\\n' " + shellSingleQuote(row) + "\n"
	}
	fuzzNightly += `} >"$out_dir/unstable-seeds.md"
`
	if err := os.WriteFile(filepath.Join(root, "scripts", "dev", "fuzz-nightly.sh"), []byte(fuzzNightly), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "ci", "test-all.sh"), []byte(`#!/usr/bin/env bash
set -euo pipefail
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "release", "v1_0", "security-review.sh"), []byte(`#!/usr/bin/env bash
set -euo pipefail
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "generated"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "generated", "manifest.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "security-review.md"), []byte("# Security Review\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func runReleaseV030Gate(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{"scripts/release/v0_3_0/gate.sh"}, args...)...)
	cmd.Dir = root
	return cmd.CombinedOutput()
}

func runReleaseV030RunnableGate(t *testing.T, root, reportDir string) ([]byte, error) {
	t.Helper()
	return runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
	})
}

func runReleaseV030RunnableGateWithEnv(t *testing.T, root, reportDir string, env []string) ([]byte, error) {
	t.Helper()
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	cmd := exec.Command("bash", "scripts/release/v0_3_0/gate.sh", "--report-dir", reportDir)
	cmd.Dir = root
	cmd.Env = append(filteredReleaseV030GateEnv(),
		"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	if !envHasPrefix(env, "TETRA_MACOS_RUNTIME_SMOKE_REPORT=") {
		cmd.Env = append(cmd.Env, "TETRA_MACOS_RUNTIME_SMOKE_REPORT="+macosReport)
	}
	if !envHasPrefix(env, "TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=") {
		cmd.Env = append(cmd.Env, "TETRA_WINDOWS_RUNTIME_SMOKE_REPORT="+windowsReport)
	}
	cmd.Env = append(cmd.Env, env...)
	return cmd.CombinedOutput()
}

func envHasPrefix(env []string, prefix string) bool {
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return true
		}
	}
	return false
}

func filteredReleaseV030GateEnv() []string {
	blocked := map[string]bool{
		"TETRA_SECURITY_REVIEW_SIGNOFF":                        true,
		"TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF": true,
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT":                     true,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT":                   true,
		"TETRA_RESIDUAL_RISKS_JSON":                            true,
	}
	out := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if ok && blocked[key] {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func writeReleaseV030RuntimeSmokeReports(t *testing.T, root string) (string, string) {
	t.Helper()
	dir := filepath.Join(root, "runtime-smoke")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(target string) string {
		t.Helper()
		path := filepath.Join(dir, target+".json")
		raw := `{
  "timestamp": "2026-04-30T12:00:00Z",
  "target": "` + target + `",
  "host": "` + target + `",
  "version": "v0.3.0",
  "git_head": "fake-head-for-release-v030-test",
  "islands_debug": false,
  "total": 8,
  "passed": 8,
  "failed": 0,
  "cases": [
    {"name":"actors_pingpong","src_path":"examples/actors_pingpong.tetra","out_path":"/tmp/actors_pingpong","expected_exit":0,"actual_exit":0,"ran":true,"pass":true},
    {"name":"actor_sleep_pingpong","src_path":"examples/actor_sleep_pingpong.tetra","out_path":"/tmp/actor_sleep_pingpong","expected_exit":0,"actual_exit":0,"ran":true,"pass":true},
    {"name":"task_smoke","src_path":"examples/task_smoke.tetra","out_path":"/tmp/task_smoke","expected_exit":42,"actual_exit":42,"ran":true,"pass":true},
    {"name":"time_sleep_smoke","src_path":"examples/time_sleep_smoke.tetra","out_path":"/tmp/time_sleep_smoke","expected_exit":0,"actual_exit":0,"ran":true,"pass":true},
    {"name":"task_sleep_deadline_smoke","src_path":"examples/task_sleep_deadline_smoke.tetra","out_path":"/tmp/task_sleep_deadline_smoke","expected_exit":0,"actual_exit":0,"ran":true,"pass":true},
    {"name":"task_join_wait_smoke","src_path":"examples/task_join_wait_smoke.tetra","out_path":"/tmp/task_join_wait_smoke","expected_exit":5,"actual_exit":5,"ran":true,"pass":true},
    {"name":"deadline_aware_waits_smoke","src_path":"examples/deadline_aware_waits_smoke.tetra","out_path":"/tmp/deadline_aware_waits_smoke","expected_exit":0,"actual_exit":0,"ran":true,"pass":true},
    {"name":"wait_composition_smoke","src_path":"examples/wait_composition_smoke.tetra","out_path":"/tmp/wait_composition_smoke","expected_exit":0,"actual_exit":0,"ran":true,"pass":true}
  ]
}
`
		if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	return write("macos-x64"), write("windows-x64")
}

func installReleaseV030SummaryEchoingGo(t *testing.T, root string) {
	t.Helper()
	goScript := `#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" != "run" || "${2:-}" != "./tools/cmd/validate-release-state" ]]; then
  exit 0
fi

format="text"
report_dir=""
shift 2
while [[ $# -gt 0 ]]; do
  case "$1" in
    --format=*)
      format="${1#--format=}"
      shift
      ;;
    --format)
      format="$2"
      shift 2
      ;;
    --report-dir)
      report_dir="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

if [[ -z "$report_dir" ]]; then
  echo "missing --report-dir" >&2
  exit 2
fi

summary="$report_dir/summary.json"
status="$(sed -n 's/.*"status": "\([^"]*\)".*/\1/p' "$summary" | head -n 1)"
step_count="$(sed -n 's/.*"step_count": \([0-9][0-9]*\).*/\1/p' "$summary" | head -n 1)"
failed_count="$(sed -n 's/.*"failed_count": \([0-9][0-9]*\).*/\1/p' "$summary" | head -n 1)"

if [[ "$format" == "json" ]]; then
  printf '{"status":"pass","last_gate_evidence":{"status":"%s","step_count":%s,"failed_count":%s}}\n' "$status" "$step_count" "$failed_count"
else
  printf 'status: pass\n'
  printf 'last gate evidence: %s (%s failed of %s steps, %s)\n' "$status" "$failed_count" "$step_count" "$summary"
fi
`
	if err := os.WriteFile(filepath.Join(root, "bin", "go"), []byte(goScript), 0o755); err != nil {
		t.Fatal(err)
	}
}

func installReleaseV030CanonicalArtifactGo(t *testing.T, root string) {
	t.Helper()
	goScript := `#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" != "run" ]]; then
  exit 0
fi

tool="${2:-}"
shift 2
case "$tool" in
  "./tools/cmd/validate-release-state")
    format="text"
    report_dir=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --format=*)
          format="${1#--format=}"
          shift
          ;;
        --format)
          format="$2"
          shift 2
          ;;
        --report-dir)
          report_dir="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    if [[ "$format" == "json" ]]; then
      printf '{"schema":"tetra.release-state.v1alpha1","status":"pass","report_dir":"%s","security_review":{"status":"deferred"}}\n' "$report_dir"
    else
      printf 'status: pass\nreport_dir: %s\n' "$report_dir"
      printf 'security review evidence: deferred (%s/artifacts/security-review.md)\n' "$report_dir"
    fi
    ;;
  "./tools/cmd/validate-artifact-hashes")
    root=""
    out=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --root)
          root="$2"
          shift 2
          ;;
        --out)
          out="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    if [[ -n "$out" ]]; then
      mkdir -p "$(dirname "$out")"
      {
        printf '{"schema":"tetra.release-artifact-hashes.v1alpha1","root":".","artifacts":['
        first=1
        while IFS= read -r file; do
          rel="${file#"$root"/}"
          rel="${rel#./}"
          if [[ "$rel" == "$(basename "$out")" ]]; then
            continue
          fi
          if [[ "$first" -eq 0 ]]; then
            printf ','
          fi
          first=0
          printf '{"path":"%s","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":1}' "$rel"
        done < <(find "$root" -type f | sort)
        printf ']}\n'
      } >"$out"
    fi
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(root, "bin", "go"), []byte(goScript), 0o755); err != nil {
		t.Fatal(err)
	}
}

func installReleaseV030CIMissingSignoffFailingFinalArtifactHashGo(t *testing.T, root string) {
	t.Helper()
	realGo, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("look up go: %v", err)
	}
	counterPath := filepath.Join(root, "artifact-hashes-count")
	goScript := `#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" != "run" ]]; then
  exit 0
fi

tool="${2:-}"
shift 2
case "$tool" in
  */validate_residual_risks.go)
    exec ` + shellSingleQuote(realGo) + ` run "$tool" "$@"
    ;;
  "./tools/cmd/validate-release-state")
    format="text"
    report_dir=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --format=*)
          format="${1#--format=}"
          shift
          ;;
        --format)
          format="$2"
          shift 2
          ;;
        --report-dir)
          report_dir="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    summary="$report_dir/summary.json"
    status="$(sed -n 's/.*"status": "\([^"]*\)".*/\1/p' "$summary" | head -n 1)"
    step_count="$(sed -n 's/.*"step_count": \([0-9][0-9]*\).*/\1/p' "$summary" | head -n 1)"
    failed_count="$(sed -n 's/.*"failed_count": \([0-9][0-9]*\).*/\1/p' "$summary" | head -n 1)"
    if [[ "$format" == "json" ]]; then
      printf '{"status":"pass","last_gate_evidence":{"status":"%s","step_count":%s,"failed_count":%s}}\n' "$status" "$step_count" "$failed_count"
    else
      printf 'status: pass\n'
      printf 'last gate evidence: %s (%s failed of %s steps, %s)\n' "$status" "$failed_count" "$step_count" "$summary"
    fi
    ;;
  "./tools/cmd/validate-artifact-hashes")
    count=0
    if [[ -f ` + shellSingleQuote(counterPath) + ` ]]; then
      count="$(cat ` + shellSingleQuote(counterPath) + `)"
    fi
    count=$((count + 1))
    printf '%s\n' "$count" >` + shellSingleQuote(counterPath) + `
    if [[ "$count" -ge 3 ]]; then
      echo "forced CI final artifact-hashes failure" >&2
      exit 1
    fi

    root=""
    out=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --root)
          root="$2"
          shift 2
          ;;
        --out)
          out="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    if [[ -n "$out" ]]; then
      mkdir -p "$(dirname "$out")"
      printf '{"schema":"tetra.release-artifact-hashes.v1alpha1","root":".","artifacts":[]}\n' >"$out"
    fi
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(root, "bin", "go"), []byte(goScript), 0o755); err != nil {
		t.Fatal(err)
	}
}

func installReleaseV030FailingFinalArtifactHashGo(t *testing.T, root string) {
	t.Helper()
	realGo, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("look up go: %v", err)
	}
	counterPath := filepath.Join(root, "artifact-hashes-count")
	goScript := `#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" != "run" ]]; then
  exit 0
fi

tool="${2:-}"
shift 2
case "$tool" in
  */validate_residual_risks.go)
    exec ` + shellSingleQuote(realGo) + ` run "$tool" "$@"
    ;;
  "./tools/cmd/validate-release-state")
    format="text"
    report_dir=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --format=*)
          format="${1#--format=}"
          shift
          ;;
        --format)
          format="$2"
          shift 2
          ;;
        --report-dir)
          report_dir="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    if [[ "$format" == "json" ]]; then
      printf '{"schema":"tetra.release-state.v1alpha1","status":"pass","report_dir":"%s","security_review":{"status":"deferred"}}\n' "$report_dir"
    else
      printf 'status: pass\nreport_dir: %s\n' "$report_dir"
      printf 'security review evidence: deferred (%s/artifacts/security-review.md)\n' "$report_dir"
    fi
    ;;
  "./tools/cmd/validate-artifact-hashes")
    count=0
    if [[ -f ` + shellSingleQuote(counterPath) + ` ]]; then
      count="$(cat ` + shellSingleQuote(counterPath) + `)"
    fi
    count=$((count + 1))
    printf '%s\n' "$count" >` + shellSingleQuote(counterPath) + `
    if [[ "$count" -ge 5 ]]; then
      echo "forced final artifact-hashes failure" >&2
      exit 1
    fi

    root=""
    out=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --root)
          root="$2"
          shift 2
          ;;
        --out)
          out="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    if [[ -n "$out" ]]; then
      mkdir -p "$(dirname "$out")"
      {
        printf '{"schema":"tetra.release-artifact-hashes.v1alpha1","root":".","artifacts":['
        first=1
        while IFS= read -r file; do
          rel="${file#"$root"/}"
          rel="${rel#./}"
          if [[ "$rel" == "$(basename "$out")" ]]; then
            continue
          fi
          if [[ "$first" -eq 0 ]]; then
            printf ','
          fi
          first=0
          printf '{"path":"%s","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":1}' "$rel"
        done < <(find "$root" -type f | sort)
        printf ']}\n'
      } >"$out"
    fi
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(root, "bin", "go"), []byte(goScript), 0o755); err != nil {
		t.Fatal(err)
	}
}

func installReleaseV030FailingSecurityReviewSha256(t *testing.T, root string) {
	t.Helper()
	realSha256sum, err := exec.LookPath("sha256sum")
	if err != nil {
		t.Fatalf("look up sha256sum: %v", err)
	}
	shaScript := `#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  */artifacts/security-review.md)
    echo "forced detached security-review hash failure" >&2
    exit 1
    ;;
esac
exec ` + shellSingleQuote(realSha256sum) + ` "$@"
`
	if err := os.WriteFile(filepath.Join(root, "bin", "sha256sum"), []byte(shaScript), 0o755); err != nil {
		t.Fatal(err)
	}
}

func installReleaseV030PortablePythonCanonicalizers(t *testing.T, root string) string {
	t.Helper()
	binDir := filepath.Join(root, "bin")
	markerPath := filepath.Join(root, "python3-canonicalizer.log")
	pythonScript := `#!/usr/bin/env bash
set -euo pipefail
echo "bare python should not be required for artifact manifest canonicalization" >&2
exit 127
`
	if err := os.WriteFile(filepath.Join(binDir, "python"), []byte(pythonScript), 0o755); err != nil {
		t.Fatal(err)
	}
	python3Script := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" != "-" || -z "${2:-}" ]]; then
  echo "unexpected python3 canonicalizer args: $*" >&2
  exit 2
fi
cat >/dev/null
manifest_path="$2"
printf 'python3 canonicalized %s\n' "$manifest_path" >>` + shellSingleQuote(markerPath) + `
cat >"$manifest_path" <<'JSON'
{
  "schema": "tetra.release-artifact-hashes.v1alpha1",
  "root": ".",
  "artifacts": [
    {
      "path": "summary.json",
      "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "size": 1
    }
  ]
}
JSON
`
	if err := os.WriteFile(filepath.Join(binDir, "python3"), []byte(python3Script), 0o755); err != nil {
		t.Fatal(err)
	}
	return markerPath
}
