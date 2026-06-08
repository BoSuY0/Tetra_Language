#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
report_dir=""
check_timeout="${SURFACE_UI_TRUTH_AUDIT_TIMEOUT:-20}"

usage() {
  cat <<'USAGE'
Usage: bash scripts/analysis/surface-ui-truth-audit.sh --report-dir DIR

Collects reproducible Surface/UI production truth-audit evidence. This script is
not a release gate and never claims production readiness. It records PASS, FAIL,
BLOCKED, or SKIPPED outcomes for discovery, focused tests, validators, and
Surface release-gate probes.
USAGE
}

reject_report_dir() {
  local message="$1"
  echo "refusing unsafe report dir: $message" >&2
  exit 2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      if [[ $# -lt 2 || -z "${2:-}" ]]; then
        echo "--report-dir requires a value" >&2
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
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_dir" ]]; then
  echo "--report-dir requires a value" >&2
  usage >&2
  exit 2
fi
if [[ "$report_dir" = /* ]]; then
  reject_report_dir "absolute paths are not accepted ($report_dir)"
fi
if [[ "$report_dir" == "." || "$report_dir" == "./" ]]; then
  reject_report_dir "repo root is not an audit report directory"
fi
IFS='/' read -r -a report_parts <<<"$report_dir"
for part in "${report_parts[@]}"; do
  if [[ "$part" == ".." ]]; then
    reject_report_dir "parent traversal is not accepted ($report_dir)"
  fi
done

cd "$repo_root"
report_path="$repo_root/$report_dir"
if [[ -L "$report_path" ]]; then
  reject_report_dir "symlink report dirs are not accepted ($report_dir)"
fi
if [[ -e "$report_path" && ! -d "$report_path" ]]; then
  reject_report_dir "report path exists and is not a directory ($report_dir)"
fi
if [[ -d "$report_path" ]]; then
  first_entry="$(find -H -- "$report_path" -mindepth 1 -print -quit)"
  if [[ -n "$first_entry" ]]; then
    reject_report_dir "non-empty report dirs are not accepted ($report_dir)"
  fi
fi

export GOTELEMETRY="${GOTELEMETRY:-off}"
export GOCACHE="${GOCACHE:-$repo_root/.cache/go-build-surface-ui-truth-audit}"
mkdir -p "$GOCACHE"
mkdir -p "$report_path"

rg_index="$report_path/surface-rg-index.txt"
script_index="$report_path/surface-script-index.txt"
tool_index="$report_path/surface-tool-index.txt"
example_index="$report_path/surface-example-index.txt"
focused_log="$report_path/focused-tests.log"
gates_log="$report_path/surface-gates.log"
validators_log="$report_path/validators.log"
summary="$report_path/truth-summary.md"
checks_tsv="$report_path/checks.tsv"
gate_runs_dir="$report_path/gate-runs"

: >"$focused_log"
: >"$gates_log"
: >"$validators_log"
: >"$checks_tsv"
mkdir -p "$gate_runs_dir"

{
  echo "# Surface/UI grep index"
  echo
  rg -n --no-heading 'Surface|surface|UI|ui|Draw|draw|Accessibility|TextBox|wasm32-web|linux-x64' \
    lib compiler tools scripts docs examples .github 2>/dev/null | head -n 800 || true
} >"$rg_index"

{
  echo "# Surface/UI script index"
  echo
  find scripts/release/surface scripts/release/safe-view-lifetime scripts/ci .github/workflows \
    -maxdepth 3 -type f -print 2>/dev/null | LC_ALL=C sort || true
} >"$script_index"

{
  echo "# Surface/UI tool index"
  echo
  rg --files tools/cmd tools/validators 2>/dev/null | rg 'surface|wasm-import|artifact-hashes|api-docs|manifest|docs' || true
} >"$tool_index"

{
  echo "# Surface/UI example index"
  echo
  rg --files examples 2>/dev/null | rg 'surface_.*[.]tetra$' || true
} >"$example_index"

format_command() {
  local formatted=""
  local quoted=""
  local arg
  for arg in "$@"; do
    printf -v quoted "%q" "$arg"
    if [[ -z "$formatted" ]]; then
      formatted="$quoted"
    else
      formatted+=" $quoted"
    fi
  done
  printf "%s" "$formatted"
}

status_for_failure() {
  local code="$1"
  local log="$2"
  if [[ "$code" == "124" || "$code" == "137" ]]; then
    printf "BLOCKED"
    return
  fi
  if [[ -f "$log" ]] && grep -Eqi 'WAYLAND_DISPLAY|DISPLAY|Chromium|browser.*unavailable|runner.*unavailable|timed out|timeout|target host|unavailable' "$log"; then
    printf "BLOCKED"
    return
  fi
  printf "FAIL"
}

run_check() {
  local category="$1"
  local name="$2"
  local log="$3"
  shift 3

  local command
  command="$(format_command "$@")"
  mkdir -p "$(dirname "$log")" "$report_path"
  : >>"$log"
  {
    echo "== $name =="
    echo "command: $command"
  } >>"$log"

  local code=0
  if command -v timeout >/dev/null 2>&1; then
    timeout --kill-after=2s "${check_timeout}s" "$@" >>"$log" 2>&1 || code="$?"
  else
    "$@" >>"$log" 2>&1 || code="$?"
  fi

  local status="PASS"
  if [[ "$code" != "0" ]]; then
    status="$(status_for_failure "$code" "$log")"
  fi
  mkdir -p "$(dirname "$log")" "$report_path"
  : >>"$log"
  : >>"$checks_tsv"
  printf '%s\t%s\t%s\t%s\n' "$category" "$name" "$status" "$command" >>"$checks_tsv"
  echo "status: $status (exit $code)" >>"$log"
  echo >>"$log"
}

run_check "focused" "surface validators package" "$focused_log" \
  go test -buildvcs=false ./tools/validators/surface -count=1
run_check "focused" "surface runtime validator package" "$focused_log" \
  go test -buildvcs=false ./tools/cmd/validate-surface-runtime -count=1
run_check "focused" "surface release state validator package" "$focused_log" \
  go test -buildvcs=false ./tools/cmd/validate-surface-release-state -count=1
run_check "focused" "surface release script tests" "$focused_log" \
  go test -buildvcs=false ./tools/scriptstest -run 'TestReleaseSurface|TestCurrentSupportedSurface|TestCIWorkflowIncludesSurface|TestSurface(Tree|Toolkit|Accessibility)' -count=1

run_check "validators" "wasm imports validator package" "$validators_log" \
  go test -buildvcs=false ./tools/cmd/validate-wasm-imports -count=1
run_check "validators" "artifact hashes validator package" "$validators_log" \
  go test -buildvcs=false ./tools/cmd/validate-artifact-hashes -count=1
run_check "validators" "manifest validation" "$validators_log" \
  go run -buildvcs=false ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
run_check "validators" "docs verification" "$validators_log" \
  go run -buildvcs=false ./tools/cmd/verify-docs --manifest docs/generated/manifest.json

run_check "surface-gate" "api stability gate" "$gates_log" \
  bash scripts/release/surface/api-stability-gate.sh --report-dir "$report_dir/gate-runs/surface-api-stability-v1"
run_check "surface-gate" "headless release smoke" "$gates_log" \
  bash scripts/release/surface/surface-headless-release-smoke.sh --report-dir "$report_dir/gate-runs/headless-release"
run_check "surface-gate" "headless text input release smoke" "$gates_log" \
  bash scripts/release/surface/surface-headless-release-text-input-smoke.sh --report-dir "$report_dir/gate-runs/headless-release-text-input"
run_check "surface-gate" "linux x64 release window smoke" "$gates_log" \
  bash scripts/release/surface/surface-linux-x64-release-window-smoke.sh --report-dir "$report_dir/gate-runs/linux-x64-release-window"
run_check "surface-gate" "wasm32 web release browser smoke" "$gates_log" \
  bash scripts/release/surface/surface-wasm32-web-release-browser-smoke.sh --report-dir "$report_dir/gate-runs/wasm32-web-release-browser"
run_check "surface-gate" "safe view lifetime gate" "$gates_log" \
  bash scripts/release/safe-view-lifetime/gate.sh --report-dir "$report_dir/gate-runs/safe-view-lifetime"

pass_count="$(awk -F '\t' '$3 == "PASS" { count++ } END { print count + 0 }' "$checks_tsv")"
fail_count="$(awk -F '\t' '$3 == "FAIL" { count++ } END { print count + 0 }' "$checks_tsv")"
blocked_count="$(awk -F '\t' '$3 == "BLOCKED" { count++ } END { print count + 0 }' "$checks_tsv")"
skipped_count="$(awk -F '\t' '$3 == "SKIPPED" { count++ } END { print count + 0 }' "$checks_tsv")"

{
  echo "# Surface/UI Truth Audit"
  echo
  echo "- production_ready_claim: false"
  echo "- report_dir: \`$report_dir\`"
  echo "- git_head: \`$(git rev-parse HEAD 2>/dev/null || echo unknown)\`"
  echo "- git_available: $([[ -d .git || -f .git ]] && echo true || echo false)"
  echo "- generated_at_utc: \`$(date -u +%Y-%m-%dT%H:%M:%SZ)\`"
  echo "- pass_count: \`$pass_count\`"
  echo "- fail_count: \`$fail_count\`"
  echo "- blocked_count: \`$blocked_count\`"
  echo "- skipped_count: \`$skipped_count\`"
  echo
  echo "This is an evidence collection report, not a Surface/UI production"
  echo "readiness claim. FAIL and BLOCKED checks remain blockers until fixed or"
  echo "explicitly accepted by the production plan."
  echo
  echo "## Artifacts"
  echo
  for artifact in \
    surface-rg-index.txt \
    surface-script-index.txt \
    surface-tool-index.txt \
    surface-example-index.txt \
    focused-tests.log \
    surface-gates.log \
    validators.log \
    truth-summary.md; do
    echo "- \`$report_dir/$artifact\`"
  done
  echo
  echo "## Checks"
  echo
  echo "| category | name | status | command |"
  echo "| --- | --- | --- | --- |"
  awk -F '\t' '{ printf "| `%s` | `%s` | `%s` | `%s` |\n", $1, $2, $3, $4 }' "$checks_tsv"
} >"$summary"

echo "Surface/UI truth audit summary: $summary"
echo "production_ready_claim: false"
