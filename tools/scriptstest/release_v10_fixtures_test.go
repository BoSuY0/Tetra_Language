package scriptstest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func releaseV10GateFakeRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{
		"bin",
		"docs/baselines",
		"docs/generated/v1_0",
		"docs/release",
		"examples",
		"scripts",
		"scripts/ci",
		"scripts/dev",
		"scripts/release/v1_0",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "gate.sh"), filepath.Join(root, "scripts", "release", "v1_0", "gate.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "flow_hello.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "ui_web_smoke.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "generated", "manifest.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "generated", "v1_0", "performance-regression.json"), []byte("{\"status\":\"pass\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "baselines", "api-diff-baseline.v1alpha1.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "release", "v1_0_final_handoff.md"), []byte("# Tetra v1.0 Final Handoff\n\nStatus: ready for lint fixture.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "security-review.md"), []byte("# Security Review\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	writeScript := func(rel, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	writeScript("scripts/dev/bootstrap.sh", "#!/usr/bin/env bash\nset -euo pipefail\nexit 0\n")
	writeScript("scripts/ci/test-all.sh", `#!/usr/bin/env bash
set -euo pipefail
report_dir=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      report_dir="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
if [[ -n "$report_dir" ]]; then
  mkdir -p "$report_dir"
  printf '{"status":"pass","steps":[]}\n' >"$report_dir/summary.json"
fi
exit 0
`)
	writeScript("scripts/release/v1_0/wasi-smoke.sh", `#!/usr/bin/env bash
set -euo pipefail
report=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --report)
      report="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
mkdir -p "$(dirname "$report")"
printf '{"status":"pass","runner":"fake","cases":[]}\n' >"$report"
`)
	writeScript("scripts/release/v1_0/web-smoke.sh", `#!/usr/bin/env bash
set -euo pipefail
report=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --report)
      report="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
mkdir -p "$(dirname "$report")"
printf '{"status":"pass","ui_schema":"tetra.ui.v1","cases":[]}\n' >"$report"
`)
	writeScript("scripts/release/v1_0/api-diff.sh", `#!/usr/bin/env bash
set -euo pipefail
report_dir=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      report_dir="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
mkdir -p "$report_dir"
printf '# API Docs\n' >"$report_dir/api-docs.md"
printf '{"review":{"status":"clean"},"diff":{"added":[],"removed":[],"changed":[]}}\n' >"$report_dir/api-diff.json"
`)
	for _, script := range []string{
		"release/v1_0/binary-size.sh",
		"release/v1_0/reproducible-build.sh",
	} {
		writeScript("scripts/"+script, `#!/usr/bin/env bash
set -euo pipefail
report=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --report)
      report="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
mkdir -p "$(dirname "$report")"
printf '{"status":"pass"}\n' >"$report"
`)
	}
	writeScript("scripts/release/v1_0/security-review.sh", "#!/usr/bin/env bash\nset -euo pipefail\nexit 0\n")

	tetra := `#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-}"
shift || true
case "$cmd" in
  version)
    echo "v1.0.0"
    ;;
  targets|doctor)
    printf '{"status":"pass"}\n'
    ;;
  check|fmt)
    if [[ "$cmd" == "check" && "${1:-}" == "--diagnostics=json" ]]; then
      printf '{"severity":"error","message":"fake diagnostic","range":{"start":{"line":1,"column":1},"end":{"line":1,"column":2}}}\n' >&2
      exit 1
    fi
    exit 0
    ;;
  test)
    printf '{"status":"pass","tests":[]}\n'
    ;;
  doc)
    printf '# Tetra Docs\n'
    ;;
  smoke)
    report=""
    list_mode="false"
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --list)
          list_mode="true"
          shift
          ;;
        --report)
          report="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    if [[ "$list_mode" == "true" && -z "$report" ]]; then
      printf '{"status":"pass","cases":[]}\n'
      exit 0
    fi
    mkdir -p "$(dirname "$report")"
    printf '{"status":"pass","cases":[]}\n' >"$report"
    ;;
  *)
    echo "unexpected tetra command: $cmd $*" >&2
    exit 2
    ;;
esac
`
	writeScript("tetra", tetra)
	writeScript("t", tetra)

	writeScript("bin/git", `#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  ls-files)
    printf 'examples/flow_hello.tetra\nexamples/ui_web_smoke.tetra\n'
    ;;
  status|diff)
    exit 0
    ;;
  branch)
    printf 'main\n'
    ;;
  *)
    exit 0
    ;;
esac
`)
	writeScript("bin/go", `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "run" ]]; then
  shift
  tool="${1:-}"
  shift || true
  case "$tool" in
    ./tools/cmd/gen-manifest)
      out=""
      while [[ $# -gt 0 ]]; do
        case "$1" in
          -o)
            out="$2"
            shift 2
            ;;
          *)
            shift
            ;;
        esac
      done
      mkdir -p "$(dirname "$out")"
      printf '{}\n' >"$out"
      ;;
    ./tools/cmd/gen-docs)
      printf '# API Docs\n'
      ;;
    ./tools/cmd/validate-release-state)
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
      if [[ ! -s "$summary" ]]; then
        echo "validate-release-state: missing current summary $summary" >&2
        exit 1
      fi
      status="$(sed -n 's/.*"status": "\([^"]*\)".*/\1/p' "$summary" | head -n 1)"
      step_count="$(sed -n 's/.*"step_count": \([0-9][0-9]*\).*/\1/p' "$summary" | head -n 1)"
      failed_count="$(sed -n 's/.*"failed_count": \([0-9][0-9]*\).*/\1/p' "$summary" | head -n 1)"
      if [[ "$format" == "json" ]]; then
        printf '{"schema":"tetra.release-state.v1alpha1","status":"pass","last_gate_evidence":{"status":"%s","step_count":%s,"failed_count":%s}}\n' "$status" "$step_count" "$failed_count"
      else
        printf 'status: pass\nlast gate evidence: %s (%s failed of %s steps)\n' "$status" "$failed_count" "$step_count"
      fi
      ;;
    ./tools/cmd/validate-artifact-hashes)
      out=""
      root=""
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
        if [[ -z "$root" ]]; then
          root="$(dirname "$out")"
        fi
        mkdir -p "$(dirname "$out")"
        {
          printf '{"schema":"tetra.release-artifact-hashes.v1alpha1","root":".","artifacts":['
          first="true"
          while IFS= read -r file; do
            rel="${file#"$root"/}"
            if [[ "$rel" == "artifact-hashes.json" ]]; then
              continue
            fi
            if [[ "$first" == "true" ]]; then
              first="false"
            else
              printf ','
            fi
            size="$(wc -c <"$file" | tr -d ' ')"
            printf '{"path":"%s","sha256":"sha256:0000000000000000000000000000000000000000000000000000000000000000","size":%s}' "$rel" "$size"
          done < <(find "$root" -type f | sort)
          printf ']}\n'
        } >"$out"
      fi
      ;;
  esac
fi
exit 0
`)

	return root
}

func releaseV10WASISmokeFakeRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{
		"bin",
		"examples",
		"examples/projects/dogfood_wasi/src",
		"scripts",
		"scripts/release/v1_0",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "wasi-smoke.sh"), filepath.Join(root, "scripts", "release", "v1_0", "wasi-smoke.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "ui_web_smoke.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "projects", "dogfood_wasi", "src", "main.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	writeScript := func(rel, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	writeScript("tetra", `#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-}"
shift || true
case "$cmd" in
  smoke)
    printf '%s\n' "$*" >>tetra-smoke.log
    for arg in "$@"; do
      if [[ "$arg" == "--list" ]]; then
        printf '{"target":"wasm32-wasi","cases":[{"name":"dogfood_wasi","src_path":"examples/projects/dogfood_wasi/src/main.tetra"},{"name":"ui_web_smoke","src_path":"examples/ui_web_smoke.tetra"}]}\n'
        exit 0
      fi
    done
    report=""
    run=""
    list_mode="false"
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --list)
          list_mode="true"
          shift
          ;;
        --report)
          report="$2"
          shift 2
          ;;
        --run=*)
          run="${1#--run=}"
          shift
          ;;
        *)
          shift
          ;;
      esac
    done
    if [[ "$list_mode" == "true" ]]; then
      printf '{"target":"wasm32-wasi","build_only":true,"run_supported":true,"total":5,"islands_debug":false,"cases":[{"name":"legacy_hello","src_path":"examples/hello.tetra","target_group":"wasm","expected_exit":0},{"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","target_group":"wasm","expected_exit":0},{"name":"ui_web_smoke","src_path":"examples/ui_web_smoke.tetra","target_group":"wasm","expected_exit":0},{"name":"dogfood_wasi","src_path":"examples/projects/dogfood_wasi/src/main.tetra","target_group":"wasm","expected_exit":0},{"name":"dogfood_web_ui","src_path":"examples/projects/dogfood_web_ui/src/main.tetra","target_group":"wasm","expected_exit":0}]}\n'
      exit 0
    fi
    if [[ -z "$report" ]]; then
      echo "missing --report" >&2
      exit 2
    fi
    mkdir -p "$(dirname "$report")"
    if [[ "$run" == "true" ]]; then
      printf '{"target":"wasm32-wasi","runner":"unified-cli","total":1,"passed":1,"failed":0,"cases":[]}\n' >"$report"
    else
      printf '{"target":"wasm32-wasi","build_only":false,"total":1,"passed":1,"failed":0,"cases":[]}\n' >"$report"
    fi
    ;;
  build)
    out=""
    src=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        -o)
          out="$2"
          shift 2
          ;;
        *)
          src="$1"
          shift
          ;;
      esac
    done
    if [[ -z "$out" ]]; then
      echo "missing -o" >&2
      exit 2
    fi
    printf 'wasm:%s\n' "$src" >"$out"
    if [[ "$src" == "examples/ui_web_smoke.tetra" ]]; then
      printf '{"schema":"tetra.ui.v1"}\n' >"$out.ui.json"
    fi
    ;;
  *)
    echo "unexpected tetra command: $cmd $*" >&2
    exit 2
    ;;
esac
`)
	writeScript("bin/go", `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" != "run" ]]; then
  exit 0
fi
shift
tool="${1:-}"
shift || true
printf '%s %s\n' "$(basename "$tool")" "$*" >>validator.log
case "$tool" in
  ./tools/cmd/smoke-report-to-checklist)
    report=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --report)
          report="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    if [[ -z "$report" || ! -s "$report" ]]; then
      echo "missing smoke report: $report" >&2
      exit 1
    fi
    ;;
esac
exit 0
`)
	return root
}

type webSmokeScriptReport struct {
	Automation string `json:"automation"`
	Status     string `json:"status"`
	Blocker    string `json:"blocker"`
}

func releaseV10WebSmokeFakeRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"bin", "examples", "scripts/release/v1_0"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "web-smoke.sh"), filepath.Join(root, "scripts", "release", "v1_0", "web-smoke.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "flow_hello.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cat", "cp", "dirname", "find", "grep", "head", "mkdir", "mktemp", "node", "rm", "sed", "sleep", "sort"} {
		writeToolWrapper(t, root, name)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "git"), []byte(`#!/bin/sh
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "go"), []byte(`#!/bin/sh
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "python3"), []byte(`#!/bin/sh
sleep 60
`), 0o755); err != nil {
		t.Fatal(err)
	}
	tetra := `#!/bin/sh
if [ "${1:-}" = "smoke" ]; then
  shift
  list_mode="false"
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --list)
        list_mode="true"
        shift
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ "$list_mode" = "true" ]; then
    printf '%s\n' '{"target":"wasm32-web","build_only":true,"run_supported":false,"total":5,"islands_debug":false,"cases":[{"name":"legacy_hello","src_path":"examples/hello.tetra","target_group":"wasm","expected_exit":0},{"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","target_group":"wasm","expected_exit":0},{"name":"ui_web_smoke","src_path":"examples/ui_web_smoke.tetra","target_group":"wasm","expected_exit":0},{"name":"dogfood_wasi","src_path":"examples/projects/dogfood_wasi/src/main.tetra","target_group":"wasm","expected_exit":0},{"name":"dogfood_web_ui","src_path":"examples/projects/dogfood_web_ui/src/main.tetra","target_group":"wasm","expected_exit":0}]}'
    exit 0
  fi
fi
if [ "${1:-}" = "build" ]; then
  out=""
  while [ "$#" -gt 0 ]; do
    if [ "$1" = "-o" ]; then
      out="$2"
      shift 2
    else
      shift
    fi
  done
  if [ -z "$out" ]; then
    echo "missing -o" >&2
    exit 2
  fi
  printf '%s\n' "build" > tetra-build.log
  printf '%s\n' "export async function runTetra() { return 0; }" > "$out.mjs"
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

func writeToolWrapper(t *testing.T, root, name string) {
	t.Helper()
	realPath, err := exec.LookPath(name)
	if err != nil {
		t.Fatalf("look up %s: %v", name, err)
	}
	wrapper := "#!/bin/sh\nexec " + shellSingleQuote(realPath) + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(root, "bin", name), []byte(wrapper), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeReleaseV10FakeBrowser(t *testing.T, root, name string) {
	t.Helper()
	browser := `#!/bin/sh
printf '%s\n' '<html><body><pre id="result">ok:0</pre></body></html>'
`
	if err := os.WriteFile(filepath.Join(root, "bin", name), []byte(browser), 0o755); err != nil {
		t.Fatal(err)
	}
}

func runReleaseV10WebSmoke(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Fatalf("look up bash: %v", err)
	}
	cmd := exec.Command(bashPath, append([]string{"scripts/release/v1_0/web-smoke.sh"}, args...)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"PATH="+filepath.Join(root, "bin"),
		"GOCACHE="+filepath.Join(root, "gocache"),
	)
	return cmd.CombinedOutput()
}

func readWebSmokeReport(t *testing.T, reportPath string) webSmokeScriptReport {
	t.Helper()
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read web smoke report: %v", err)
	}
	var report webSmokeScriptReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode web smoke report: %v\n%s", err, string(raw))
	}
	return report
}
