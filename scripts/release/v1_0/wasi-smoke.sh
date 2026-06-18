#!/usr/bin/env bash
set -euo pipefail

report_path=""

if [[ -z "${GOCACHE:-}" ]]; then
  cache_home="${XDG_CACHE_HOME:-${HOME:?HOME must be set}/.cache}"
  GOCACHE="$cache_home/tetra-language/go-build"
fi
mkdir -p "$GOCACHE"
export GOCACHE

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/v1_0/wasi-smoke.sh [--report PATH]

Runs WASI smoke through the unified CLI runtime path.
Also records an artifact/import preflight report; that report is not runtime
proof and must not be used as the production WASI claim.
Default report: docs/generated/v1_0/wasi-smoke.json
USAGE
}

require_flag_value() {
  local flag="$1"
  local value="${2:-}"
  if [[ -z "$value" ]]; then
    echo "release/v1_0/wasi-smoke: ${flag} requires a path" >&2
    exit 2
  fi
}

normalize_relative_dash_path() {
  local path="$1"
  if [[ "$path" == -* ]]; then
    printf './%s' "$path"
  else
    printf '%s' "$path"
  fi
}

prepare_output_file_path() {
  if [[ -d "$report_path" || -L "$report_path" ]]; then
    echo "release/v1_0/wasi-smoke: refusing to use directory report path: $report_path" >&2
    exit 2
  fi
  local parent_dir
  parent_dir="$(dirname "$report_path")"
  mkdir -p -- "$parent_dir"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report)
      require_flag_value "$1" "${2:-}"
      report_path="$2"
      shift 2
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "release/v1_0/wasi-smoke: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_path" ]]; then
  report_path="docs/generated/v1_0/wasi-smoke.json"
fi
report_path="$(normalize_relative_dash_path "$report_path")"
prepare_output_file_path

if ! command -v node > /dev/null 2>&1; then
  echo "release/v1_0/wasi-smoke: runtime prerequisite unavailable: node" >&2
  exit 1
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

smoke_source_for_case() {
  local list_path="$1"
  local case_name="$2"
  node - "$list_path" "$case_name" << 'JS'
const fs = require('fs');
const listPath = process.argv[2];
const caseName = process.argv[3];
const report = JSON.parse(fs.readFileSync(listPath, 'utf8'));
const found = (report.cases || []).find((item) => item.name === caseName);
if (!found || !found.src_path) {
  console.error(`smoke case ${caseName} not found in ${listPath}`);
  process.exit(1);
}
process.stdout.write(found.src_path);
JS
}

smoke_list="$tmp_dir/wasm32-wasi-smoke-list.json"
./tetra smoke --list --target wasm32-wasi --format=json > "$smoke_list"
go run ./tools/cmd/validate-smoke-list --report "$smoke_list"

artifact_report="$tmp_dir/wasm32-wasi-artifact-preflight.json"
./tetra smoke --target wasm32-wasi --run=false --report "$artifact_report"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$artifact_report"
go run ./tools/cmd/validate-wasi-smoke-report --mode artifact --report "$artifact_report"

cp -- "$artifact_report" "${report_path%.json}.artifact.json"

wasi_dogfood_src="$(smoke_source_for_case "$smoke_list" "dogfood_wasi")"
wasi_ui_probe="$tmp_dir/dogfood_wasi_probe"
if ./tetra build --target wasm32-wasi -o "$wasi_ui_probe" "$wasi_dogfood_src" > "$tmp_dir/dogfood_wasi_build.out" 2> "$tmp_dir/dogfood_wasi_build.err"; then
  go run ./tools/cmd/validate-wasm-imports --target wasm32-wasi "$wasi_ui_probe"
  for sidecar in "$wasi_ui_probe.ui.json" "$wasi_ui_probe.ui.web.mjs" "$wasi_ui_probe.ui.html" "$wasi_ui_probe.ui.shell.txt"; do
    if [[ -f "$sidecar" ]]; then
      echo "release/v1_0/wasi-smoke: unexpected UI sidecar for WASI dogfood: $sidecar" >&2
      exit 1
    fi
  done
else
  echo "release/v1_0/wasi-smoke: failed to build WASI dogfood source $wasi_dogfood_src" >&2
  cat "$tmp_dir/dogfood_wasi_build.err" >&2 || true
  exit 1
fi

wasi_ui_src="$(smoke_source_for_case "$smoke_list" "ui_web_smoke")"
wasi_ui_policy_probe="$tmp_dir/wasi_ui_policy_probe"
if ./tetra build --target wasm32-wasi -o "$wasi_ui_policy_probe" "$wasi_ui_src" > "$tmp_dir/wasi_ui_policy_build.out" 2> "$tmp_dir/wasi_ui_policy_build.err"; then
  go run ./tools/cmd/validate-wasm-imports --target wasm32-wasi "$wasi_ui_policy_probe"
  if [[ ! -f "$wasi_ui_policy_probe.ui.json" ]]; then
    echo "release/v1_0/wasi-smoke: expected WASI UI metadata sidecar: $wasi_ui_policy_probe.ui.json" >&2
    exit 1
  fi
  for sidecar in "$wasi_ui_policy_probe.ui.web.mjs" "$wasi_ui_policy_probe.ui.html" "$wasi_ui_policy_probe.ui.shell.txt"; do
    if [[ -f "$sidecar" ]]; then
      echo "release/v1_0/wasi-smoke: unexpected WASI runtime UI sidecar: $sidecar" >&2
      exit 1
    fi
  done
else
  echo "release/v1_0/wasi-smoke: failed to build WASI UI source $wasi_ui_src" >&2
  cat "$tmp_dir/wasi_ui_policy_build.err" >&2 || true
  exit 1
fi

./tetra smoke --target wasm32-wasi --run=true --report "$report_path"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$report_path"
go run ./tools/cmd/validate-wasi-smoke-report --mode runtime --report "$report_path"

echo "wasi smoke report: $report_path"
