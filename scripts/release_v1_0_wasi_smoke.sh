#!/usr/bin/env bash
set -euo pipefail

report_path=""

: "${GOCACHE:=/tmp/tetra-go-cache}"
mkdir -p "$GOCACHE"
export GOCACHE

usage() {
  cat <<'USAGE'
Usage: bash scripts/release_v1_0_wasi_smoke.sh [--report PATH]

Runs WASI smoke with a real runner when available.
Runner preference: wasmtime -> node-wasi fallback.
Default report: docs/generated/v1_0/wasi-smoke.json
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report)
      report_path="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "release_v1_0_wasi_smoke: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_path" ]]; then
  report_path="docs/generated/v1_0/wasi-smoke.json"
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

build_report="$tmp_dir/wasm32-wasi-build-only.json"
./tetra smoke --target wasm32-wasi --run=false --report "$build_report"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$build_report"

mkdir -p "$(dirname "$report_path")"
cp "$build_report" "${report_path%.json}.build-only.json"

wasi_dogfood_src="examples/projects/dogfood_wasi/src/main.tetra"
wasi_ui_probe="$tmp_dir/dogfood_wasi_probe"
if ./tetra build --target wasm32-wasi -o "$wasi_ui_probe" "$wasi_dogfood_src" >"$tmp_dir/dogfood_wasi_build.out" 2>"$tmp_dir/dogfood_wasi_build.err"; then
  for sidecar in "$wasi_ui_probe.ui.json" "$wasi_ui_probe.ui.web.mjs" "$wasi_ui_probe.ui.html" "$wasi_ui_probe.ui.shell.txt"; do
    if [[ -f "$sidecar" ]]; then
      echo "release_v1_0_wasi_smoke: unexpected UI sidecar for WASI dogfood: $sidecar" >&2
      exit 1
    fi
  done
else
  echo "release_v1_0_wasi_smoke: failed to build WASI dogfood source $wasi_dogfood_src" >&2
  cat "$tmp_dir/dogfood_wasi_build.err" >&2 || true
  exit 1
fi

runner=""
if command -v wasmtime >/dev/null 2>&1; then
  runner="wasmtime"
elif command -v node >/dev/null 2>&1; then
  runner="node-wasi"
else
  node - "$build_report" "$report_path" <<'JS'
const fs = require('fs');
const build = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
const cases = Array.isArray(build.cases) ? build.cases : [];
const outCases = cases.map((c) => ({
  name: c.name,
  src_path: c.src_path,
  out_path: c.out_path || '',
  expected_exit: Number(c.expected_exit || 0),
  ran: false,
  pass: false,
  error: "missing WASI runner: need wasmtime or node",
}));
const report = {
  timestamp: new Date().toISOString(),
  target: build.target || 'wasm32-wasi',
  host: build.host || '',
  version: build.version || '',
  git_head: build.git_head || '',
  islands_debug: Boolean(build.islands_debug),
  total: outCases.length,
  passed: 0,
  failed: outCases.length,
  runner: 'none',
  cases: outCases,
};
fs.writeFileSync(process.argv[3], JSON.stringify(report, null, 2) + '\n');
JS
  echo "release_v1_0_wasi_smoke: missing WASI runner (wasmtime/node)" >&2
  exit 1
fi

node scripts/tools/run_wasi_smoke_from_report.mjs --build-report "$build_report" --out "$report_path" --runner "$runner" --work-dir "$tmp_dir/rebuilt-cases"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$report_path"

if [[ "$runner" == "node-wasi" ]]; then
  echo "release_v1_0_wasi_smoke: using node-wasi fallback runner" >&2
fi

echo "wasi smoke report: $report_path"
