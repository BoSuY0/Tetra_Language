#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/full-platform-ui-runtime"
failures=0
failed_steps=()

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/full_platform/ui-runtime-gate.sh [--report-dir DIR]

Runs the full-platform UI runtime production gate for Linux, Windows, macOS,
and Web. This gate fails unless every platform has fresh runtime-backed
evidence and artifact hashes validate.

For CI fan-in, set TETRA_WINDOWS_UI_RUNTIME_REPORT and
TETRA_MACOS_UI_RUNTIME_REPORT to validated target-host reports produced on real
Windows and macOS runners.

For diagnostic-only GitHub Actions startup blockers, set
TETRA_ACTIONS_STARTUP_BLOCKER_REPORT to a report validated by
tools/cmd/validate-actions-startup-blocker. This never replaces Windows/macOS
runtime evidence.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      if [[ $# -lt 2 ]]; then
        echo "error: --report-dir requires a value" >&2
        usage >&2
        exit 2
      fi
      report_dir="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

cd "$repo_root"

prepare_report_dir() {
  mkdir -p "$report_dir"
  rm -f "$report_dir/native-ui-linux-x64.json"
  rm -f "$report_dir/native-ui-runtime-linux-x64.integration.json"
  rm -f "$report_dir/ui-production-runtime-linux-x64.json"
  rm -f "$report_dir/windows-ui-runtime.json"
  rm -f "$report_dir/macos-ui-runtime.json"
  rm -f "$report_dir/web-smoke.json"
  rm -f "$report_dir/web-smoke.dom.html"
  rm -f "$report_dir/web-smoke.chromium.err"
  rm -f "$report_dir/web-smoke.ui.json"
  rm -f "$report_dir/web-smoke.ui.web.mjs"
  rm -f "$report_dir/github-actions-startup-blocker.json"
  rm -f "$report_dir/artifact-hashes.json"
  rm -f "$report_dir/full-platform-ui-runtime-gate.json"
}

run_step() {
  local name="$1"
  shift
  echo "==> ${name}"
  if "$@"; then
    echo "ok: ${name}"
  else
    local code=$?
    echo "FAIL: ${name} (exit ${code})" >&2
    failures=$((failures + 1))
    failed_steps+=("${name}:${code}")
  fi
}

write_gate_report() {
  local status="pass"
  if [[ "$failures" -ne 0 ]]; then
    status="fail"
  fi
  {
    printf '{\n'
    printf '  "schema": "tetra.full-platform-ui-runtime.gate.v1",\n'
    printf '  "artifact": "tetra.release.full_platform.ui_runtime.production-gate.v1",\n'
    printf '  "status": "%s",\n' "$status"
    printf '  "report_dir": "%s",\n' "$report_dir"
    printf '  "failed_steps": ['
    local first="true"
    local step
    for step in "${failed_steps[@]}"; do
      if [[ "$first" == "true" ]]; then
        first="false"
      else
        printf ', '
      fi
      printf '"%s"' "$step"
    done
    printf ']\n'
    printf '}\n'
  } >"$report_dir/full-platform-ui-runtime-gate.json"
}

prepare_report_dir

run_step baseline-tests go test ./compiler/... ./cli/... ./tools/... -count=1
run_step manifest-gen go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
run_step docs-verify go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
run_step manifest-validate go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
run_step targets-validate go run ./tools/cmd/validate-targets

run_step linux-native-ui-smoke bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir "$report_dir"
run_step linux-native-ui-validate go run ./tools/cmd/validate-native-ui-runtime --report "$report_dir/native-ui-linux-x64.json"
run_step linux-ui-production-smoke bash scripts/release/post_v0_4/ui-production-runtime-linux-x64-smoke.sh --report-dir "$report_dir"
run_step linux-ui-production-validate go run ./tools/cmd/validate-ui-production-runtime --report "$report_dir/ui-production-runtime-linux-x64.json"

run_step windows-ui-runtime-smoke bash scripts/release/full_platform/windows-ui-runtime-smoke.sh --report-dir "$report_dir"
run_step windows-ui-runtime-validate go run ./tools/cmd/validate-windows-ui-runtime --report "$report_dir/windows-ui-runtime.json"
run_step macos-ui-runtime-smoke bash scripts/release/full_platform/macos-ui-runtime-smoke.sh --report-dir "$report_dir"
run_step macos-ui-runtime-validate go run ./tools/cmd/validate-macos-ui-runtime --report "$report_dir/macos-ui-runtime.json"

run_step web-ui-smoke bash scripts/release/v1_0/web-smoke.sh --report "$report_dir/web-smoke.json"
run_step web-ui-validate go run ./tools/cmd/validate-web-ui-smoke --report "$report_dir/web-smoke.json"

run_step cross-platform-ui-validate go run ./tools/cmd/validate-cross-platform-ui-runtime \
  --linux "$report_dir/ui-production-runtime-linux-x64.json" \
  --windows "$report_dir/windows-ui-runtime.json" \
  --macos "$report_dir/macos-ui-runtime.json" \
  --web "$report_dir/web-smoke.json"

actions_startup_blocker_report="${TETRA_ACTIONS_STARTUP_BLOCKER_REPORT:-}"
if [[ -n "$actions_startup_blocker_report" ]]; then
  run_step actions-startup-blocker-import cp -- "$actions_startup_blocker_report" "$report_dir/github-actions-startup-blocker.json"
  run_step actions-startup-blocker-validate go run ./tools/cmd/validate-actions-startup-blocker --report "$report_dir/github-actions-startup-blocker.json"
fi

write_gate_report

echo "==> artifact-hashes-write"
if go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"; then
  echo "ok: artifact-hashes-write"
else
  code=$?
  echo "FAIL: artifact-hashes-write (exit ${code})" >&2
  failures=$((failures + 1))
  failed_steps+=("artifact-hashes-write:${code}")
  write_gate_report
  go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
fi

echo "==> artifact-hashes-validate"
if go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"; then
  echo "ok: artifact-hashes-validate"
else
  code=$?
  echo "FAIL: artifact-hashes-validate (exit ${code})" >&2
  failures=$((failures + 1))
  failed_steps+=("artifact-hashes-validate:${code}")
  write_gate_report
  go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
  go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
fi

if [[ "$failures" -ne 0 ]]; then
  echo "full-platform UI runtime gate failed with ${failures} failed step(s)" >&2
  printf 'failed steps: %s\n' "${failed_steps[*]}" >&2
  exit 1
fi

echo "full-platform UI runtime gate report dir: $report_dir"
echo "required artifact: tetra.release.full_platform.ui_runtime.production-gate.v1"
echo "artifact hashes: $report_dir/artifact-hashes.json"
