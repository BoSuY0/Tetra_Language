#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/wasm-ui-gui"
artifact_id="tetra.release.post_v0_4.wasm_ui_gui.production-gate.v1"

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/post_v0_4/wasm-ui-gui-production-gate.sh [--report-dir DIR]

Runs the ordered post-v0.4 production evidence gate for:
  1. wasm32-wasi artifact/import/runtime execution
  2. wasm32-web artifact/import/runtime execution
  3. browser-backed Web UI production runtime smoke
  4. Linux-x64 native UI/GUI runtime smoke
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
    -h | --help)
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
  if [[ -z "$report_dir" || "$report_dir" == "/" || "$report_dir" == "." || "$report_dir" == ".." ]]; then
    echo "error: refusing to clear unsafe report directory: ${report_dir:-<empty>}" >&2
    exit 2
  fi
  if [[ -L "$report_dir" ]]; then
    echo "error: refusing to clear symlink report directory: $report_dir" >&2
    exit 2
  fi
  if [[ -e "$report_dir" && ! -d "$report_dir" ]]; then
    echo "error: refusing to use non-directory report path: $report_dir" >&2
    exit 2
  fi
  mkdir -p "$report_dir"
  local find_report_dir="$report_dir"
  if [[ "$find_report_dir" == -* ]]; then
    find_report_dir="./$find_report_dir"
  fi
  find "$find_report_dir" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +
}

prepare_report_dir

./tetra smoke --target wasm32-wasi --run=false --report "$report_dir/wasi-artifact.json"
go run ./tools/cmd/validate-wasm-imports --target wasm32-wasi --report "$report_dir/wasi-artifact.json"
./tetra smoke --target wasm32-wasi --run=true --report "$report_dir/wasi-runtime.json"

./tetra smoke --target wasm32-web --run=false --report "$report_dir/web-artifact.json"
go run ./tools/cmd/validate-wasm-imports --target wasm32-web --report "$report_dir/web-artifact.json"
./tetra smoke --target wasm32-web --run=true --report "$report_dir/web-runtime.json"

bash scripts/release/v1_0/wasi-smoke.sh --report "$report_dir/wasi-smoke.json"
bash scripts/release/v1_0/web-smoke.sh --report "$report_dir/web-smoke.json"
go run ./tools/cmd/validate-web-ui-smoke --report "$report_dir/web-smoke.json"

bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir "$report_dir"
go run ./tools/cmd/validate-native-ui-runtime --report "$report_dir/native-ui-linux-x64.json"

bash scripts/release/post_v0_4/ui-production-runtime-linux-x64-smoke.sh --report-dir "$report_dir"
go run ./tools/cmd/validate-ui-production-runtime --report "$report_dir/ui-production-runtime-linux-x64.json"

summary_path="$report_dir/wasm-ui-gui-production-gate.json"
generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
cat > "$summary_path" << JSON
{
  "schema": "tetra.release.post_v0_4.wasm_ui_gui.production-gate.summary.v1",
  "artifact": "$artifact_id",
  "generated_at": "$generated_at",
  "status": "pass",
  "report_dir": "$report_dir",
  "evidence": [
    "wasi-artifact.json",
    "wasi-runtime.json",
    "wasi-smoke.json",
    "web-artifact.json",
    "web-runtime.json",
    "web-smoke.json",
    "native-ui-linux-x64.json",
    "ui-production-runtime-linux-x64.json"
  ]
}
JSON

go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "post-v0.4 WASM/UI/GUI production gate report dir: $report_dir"
echo "required artifact: $artifact_id"
echo "summary: $summary_path"
echo "artifact hashes: $report_dir/artifact-hashes.json"
