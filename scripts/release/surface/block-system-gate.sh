#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-block/gate"
original_args=("$@")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/block-system-gate.sh [--report-dir DIR]

Runs the strict Tetra Surface Block System release evidence gate.
It requires headless golden/checksum evidence, linux-x64 real-window evidence,
wasm32-web browser-canvas evidence, five polished Block-only example scenes,
bounded Block frame/cache budget evidence, same-commit Block report validation,
and final artifact hash integrity.

The gate must fail, not skip, when Wayland/display or Chromium-compatible
browser evidence is unavailable; blocked reports are evidence of an unmet host
precondition, not release readiness.
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
source "$script_dir/report-dir-guard.sh"
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$repo_root/.cache/go-build-surface-block-gate"
fi
mkdir -p "$GOCACHE"

report_dir_arg="${report_dir%/}"
report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_block_system_gate:")"
headless_report_dir="$report_dir_arg/headless"
linux_report_dir="$report_dir_arg/linux-x64-real-window"
wasm_report_dir="$report_dir_arg/wasm32-web-browser-canvas"

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

json_string() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '"%s"' "$value"
}

git_head="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
git_dirty=false
if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null || [[ -n "$(git ls-files --others --exclude-standard 2>/dev/null)" ]]; then
  git_dirty=true
fi
version="$(go list -m 2>/dev/null || echo tetra_language)"
host_os="$(go env GOOS 2>/dev/null || uname -s)"
host_arch="$(go env GOARCH 2>/dev/null || uname -m)"
generated_at_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
formatted_args="$(format_command "${original_args[@]}")"
command_line="bash scripts/release/surface/block-system-gate.sh"
if [[ -n "$formatted_args" ]]; then
  command_line+=" $formatted_args"
fi

bash scripts/release/surface/surface-headless-block-system-smoke.sh --report-dir "$headless_report_dir"
go run ./tools/cmd/validate-surface-block-report --report "$report_dir/headless/surface-headless-block-system.json" --same-commit "$git_head"
go run ./tools/cmd/validate-surface-block-contract --contract docs/spec/surface_block_contract.json --report "$report_dir/headless/surface-headless-block-system.json"

bash scripts/release/surface/surface-linux-x64-real-window-block-system-smoke.sh --report-dir "$linux_report_dir"
go run ./tools/cmd/validate-surface-block-report --report "$report_dir/linux-x64-real-window/surface-block-system-linux-x64.json" --same-commit "$git_head"
go run ./tools/cmd/validate-surface-block-contract --contract docs/spec/surface_block_contract.json --report "$report_dir/linux-x64-real-window/surface-block-system-linux-x64.json"

bash scripts/release/surface/surface-wasm32-web-browser-canvas-block-system-smoke.sh --report-dir "$wasm_report_dir"
go run ./tools/cmd/validate-surface-block-report --report "$report_dir/wasm32-web-browser-canvas/surface-block-system-wasm32-web.json" --same-commit "$git_head"
go run ./tools/cmd/validate-surface-block-contract --contract docs/spec/surface_block_contract.json --report "$report_dir/wasm32-web-browser-canvas/surface-block-system-wasm32-web.json"

required_reports=(
  "headless/surface-headless-block-system.json"
  "headless/surface-block-examples.json"
  "headless/artifact-hashes.json"
  "linux-x64-real-window/surface-block-system-linux-x64.json"
  "linux-x64-real-window/artifact-hashes.json"
  "wasm32-web-browser-canvas/surface-block-system-wasm32-web.json"
  "wasm32-web-browser-canvas/artifact-hashes.json"
)
for report in "${required_reports[@]}"; do
  if [[ ! -s "$report_dir/$report" ]]; then
    echo "error: required Surface Block System gate report missing or empty: $report_dir/$report" >&2
    exit 1
  fi
done

summary_path="$report_dir/surface-block-system-gate-summary.json"
cat > "$summary_path" <<JSON
{
  "schema": "tetra.surface.block-system.gate.v1",
  "status": "current",
  "release_scope": "surface-block-system-linux-web",
  "producer": "scripts/release/surface/block-system-gate.sh",
  "git_head": $(json_string "$git_head"),
  "version": $(json_string "$version"),
  "git_dirty": $git_dirty,
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "block_primitive": "Block",
  "schema_under_test": "tetra.surface.block-system.v1",
  "same_commit_validated": true,
  "headless_report": "headless/surface-headless-block-system.json",
  "examples_report": "headless/surface-block-examples.json",
  "linux_real_window_report": "linux-x64-real-window/surface-block-system-linux-x64.json",
  "wasm_browser_canvas_report": "wasm32-web-browser-canvas/surface-block-system-wasm32-web.json",
  "target_evidence": [
    "headless",
    "linux-x64-real-window",
    "wasm32-web-browser-canvas"
  ],
  "core_primitives": [
    "Block"
  ],
  "forbidden_core_primitives": [
    "Button",
    "Card",
    "TextField",
    "TextBox",
    "Sidebar",
    "Modal"
  ],
  "artifact_hashes_validated": true
}
JSON

go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface Block System gate reports: $report_dir"
echo "Surface Block System gate summary: $summary_path"
echo "Surface Block System gate artifact hashes: $report_dir/artifact-hashes.json"
