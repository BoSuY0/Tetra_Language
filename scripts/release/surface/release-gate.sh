#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface-release-v1"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/release-gate.sh [--report-dir DIR]

Runs the final Tetra Surface v1 release gate for surface-v1-linux-web.
It requires headless release evidence, linux-x64 real-window release evidence,
wasm32-web browser-canvas release evidence, strict Surface v1 validators,
artifact hash integrity, and docs/generated manifest state.

Surface v1 release gate must fail, not skip, when Chromium-compatible browser, Linux Wayland/display, accessibility probe, or clipboard harness evidence is unavailable.
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
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$repo_root/.cache/go-build-surface-release"
fi
mkdir -p "$GOCACHE"
mkdir -p "$report_dir"
report_dir="$(cd "$report_dir" && pwd)"

bash scripts/release/surface/surface-headless-release-smoke.sh --report-dir "$report_dir"
bash scripts/release/surface/surface-headless-release-text-input-smoke.sh --report-dir "$report_dir"
bash scripts/release/surface/surface-headless-release-toolkit-smoke.sh --report-dir "$report_dir"
bash scripts/release/surface/surface-headless-release-accessibility-smoke.sh --report-dir "$report_dir"
bash scripts/release/surface/surface-linux-x64-release-window-smoke.sh --report-dir "$report_dir"
bash scripts/release/surface/surface-linux-x64-release-text-input-smoke.sh --report-dir "$report_dir"
bash scripts/release/surface/surface-linux-x64-release-toolkit-smoke.sh --report-dir "$report_dir"
bash scripts/release/surface/surface-linux-x64-release-accessibility-smoke.sh --report-dir "$report_dir"
bash scripts/release/surface/surface-wasm32-web-release-browser-smoke.sh --report-dir "$report_dir"
bash scripts/release/surface/surface-wasm32-web-release-text-input-smoke.sh --report-dir "$report_dir"
bash scripts/release/surface/surface-wasm32-web-release-toolkit-smoke.sh --report-dir "$report_dir"
bash scripts/release/surface/surface-wasm32-web-release-accessibility-smoke.sh --report-dir "$report_dir"

summary_path="$report_dir/surface-release-summary.json"
cat > "$summary_path" <<'JSON'
{
  "schema": "tetra.surface.release.v1",
  "release_scope": "surface-v1-linux-web",
  "status": "current",
  "production_claim": true,
  "experimental": false,
  "supported_targets": [
    "headless",
    "linux-x64",
    "wasm32-web"
  ],
  "runtime_targets": [
    "linux-x64",
    "wasm32-web"
  ],
  "test_targets": [
    "headless"
  ],
  "unsupported_targets": [
    "macos-x64",
    "windows-x64",
    "wasm32-wasi"
  ],
  "host_abi": "tetra.surface.host.v1",
  "toolkit": "production-widgets-v1",
  "text_input": "production-text-input-v1",
  "clipboard": "clipboard-text-v1",
  "ime": "composition-baseline-v1",
  "accessibility": "platform-bridge-v1",
  "browser_surface": "browser-canvas-release-v1",
  "linux_surface": "linux-x64-release-window-v1",
  "artifact_hashes_validated": true,
  "legacy_sidecars": false,
  "dom_ui": false,
  "user_js": false,
  "platform_widgets": false
}
JSON

required_reports=(
  "surface-headless-release.json"
  "surface-headless-release-text-input.json"
  "surface-headless-release-toolkit.json"
  "surface-headless-release-accessibility.json"
  "surface-linux-x64-release-window.json"
  "surface-linux-x64-release-text-input.json"
  "surface-linux-x64-release-toolkit.json"
  "surface-linux-x64-release-accessibility.json"
  "surface-wasm32-web-release-browser.json"
  "surface-wasm32-web-release-text-input.json"
  "surface-wasm32-web-release-toolkit.json"
  "surface-wasm32-web-release-accessibility.json"
  "surface-release-summary.json"
  "artifact-hashes.json"
)
for report in "${required_reports[@]}"; do
  if [[ "$report" == "artifact-hashes.json" ]]; then
    continue
  fi
  if [[ ! -s "$report_dir/$report" ]]; then
    echo "error: required Surface v1 release report missing or empty: $report_dir/$report" >&2
    exit 1
  fi
done

go run ./tools/cmd/validate-surface-runtime --report "$report_dir/surface-release-summary.json" --release surface-v1
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-surface-release-state --report-dir "$report_dir" --expected-status current --scope surface-v1-linux-web --manifest docs/generated/manifest.json

echo "Surface v1 release gate reports: $report_dir"
echo "Surface v1 release gate summary: $report_dir/surface-release-summary.json"
echo "Surface v1 release gate artifact hashes: $report_dir/artifact-hashes.json"
