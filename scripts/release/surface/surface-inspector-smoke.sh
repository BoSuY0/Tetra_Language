#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/surface-inspector-smoke.sh [--report-dir DIR]

Runs deterministic Tetra Surface inspector smoke.
The report validates tetra.surface.inspector.v1 surface-inspector-v1 evidence
for Block tree, Morph tokens, layout, paint, accessibility, event route, focus,
and perf state. It emits a static tool report and does not depend on browser
devtools, React devtools, DOM runtime UI, or hidden state.
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
mkdir -p "$report_dir"
report_dir="$(cd "$report_dir" && pwd)"
input_dir="$report_dir/inspector-inputs"
mkdir -p "$input_dir"

block_report="$input_dir/surface-headless-block-system.json"
morph_report="$input_dir/surface-headless-morph.json"
app_model_report="$input_dir/surface-headless-app-model.json"
accessibility_report="$input_dir/surface-headless-release-accessibility.json"
events_report="$input_dir/surface-headless-block-events.json"
report_path="$report_dir/surface-inspector.json"
html_path="$report_dir/surface-inspector.html"

go run ./tools/cmd/surface-runtime-smoke --mode headless-block-system --source examples/surface_block_system.tetra --report "$block_report"
go run ./tools/cmd/surface-runtime-smoke --mode headless-morph --source examples/surface_morph_command_palette.tetra --report "$morph_report"
go run ./tools/cmd/surface-runtime-smoke --mode headless-app-model --source examples/surface_app_model.tetra --report "$app_model_report"
go run ./tools/cmd/surface-runtime-smoke --mode headless-release-accessibility --source examples/surface_release_accessibility.tetra --report "$accessibility_report"
go run ./tools/cmd/surface-runtime-smoke --mode headless-block-events --source examples/surface_block_events.tetra --report "$events_report"
go run ./tools/cmd/surface-inspector \
  --runtime-report "block:$block_report" \
  --runtime-report "morph:$morph_report" \
  --runtime-report "app-model:$app_model_report" \
  --runtime-report "accessibility:$accessibility_report" \
  --runtime-report "events:$events_report" \
  --out "$report_path" \
  --html "$html_path"
go run ./tools/cmd/validate-surface-inspector --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface inspector smoke report: $report_path"
echo "Surface inspector static HTML tool report: $html_path"
echo "Surface inspector artifact hashes: $report_dir/artifact-hashes.json"
