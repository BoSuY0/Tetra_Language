#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-visual/gate"
original_args=("$@")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/visual-gate.sh [--report-dir DIR]

Runs the Tetra Surface visual regression evidence gate.
It first produces strict Block System headless, linux-x64 real-window, and
wasm32-web browser-canvas evidence, then converts those runtime reports into a
tetra.surface.visual-regression.v1 report with deterministic frame/golden/diff,
token/theme, layout, accessibility, and performance evidence.
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
  export GOCACHE="$repo_root/.cache/go-build-surface-visual-gate"
fi
mkdir -p "$GOCACHE"

report_dir_arg="${report_dir%/}"
report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_visual_gate:")"
block_system_report_dir="$report_dir_arg/block-system"
visual_report_path="$report_dir/surface-visual-regression.json"
summary_path="$report_dir/surface-visual-gate-summary.json"

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
command_line="bash scripts/release/surface/visual-gate.sh"
if [[ -n "$formatted_args" ]]; then
  command_line+=" $formatted_args"
fi

bash scripts/release/surface/block-system-gate.sh --report-dir "$block_system_report_dir"

go run ./tools/cmd/surface-visual-diff \
  --runtime-report "$report_dir/block-system/headless/surface-headless-block-system.json" \
  --runtime-report "$report_dir/block-system/linux-x64-real-window/surface-block-system-linux-x64.json" \
  --runtime-report "$report_dir/block-system/wasm32-web-browser-canvas/surface-block-system-wasm32-web.json" \
  --block-examples-report "$report_dir/block-system/headless/surface-block-examples.json" \
  --required-target headless \
  --required-target linux-x64-real-window \
  --required-target wasm32-web-browser-canvas \
  --git-head "$git_head" \
  --out "$visual_report_path"

go run ./tools/cmd/validate-surface-visual-report --report "$visual_report_path"

cat > "$summary_path" <<JSON
{
  "schema": "tetra.surface.visual-regression.gate.v1",
  "status": "current",
  "producer": "scripts/release/surface/visual-gate.sh",
  "git_head": $(json_string "$git_head"),
  "version": $(json_string "$version"),
  "git_dirty": $git_dirty,
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "visual_report": "surface-visual-regression.json",
  "block_system_gate_summary": "block-system/surface-block-system-gate-summary.json",
  "required_targets": [
    "headless",
    "linux-x64-real-window",
    "wasm32-web-browser-canvas"
  ],
  "visual_schema": "tetra.surface.visual-regression.v1",
  "screenshot_only_evidence_rejected": true,
  "artifact_hashes_validated": true
}
JSON

required_reports=(
  "surface-visual-regression.json"
  "surface-visual-gate-summary.json"
  "block-system/surface-block-system-gate-summary.json"
  "block-system/headless/surface-headless-block-system.json"
  "block-system/linux-x64-real-window/surface-block-system-linux-x64.json"
  "block-system/wasm32-web-browser-canvas/surface-block-system-wasm32-web.json"
)
for report in "${required_reports[@]}"; do
  if [[ ! -s "$report_dir/$report" ]]; then
    echo "error: required Surface visual gate report missing or empty: $report_dir/$report" >&2
    exit 1
  fi
done

go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface visual gate reports: $report_dir"
echo "Surface visual regression report: $visual_report_path"
echo "Surface visual gate summary: $summary_path"
echo "Surface visual gate artifact hashes: $report_dir/artifact-hashes.json"
