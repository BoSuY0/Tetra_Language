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
perf state, and the Morph-to-pixels rendered beauty chain.
It emits a static tool report and does not depend on browser devtools,
React devtools, DOM runtime UI, or hidden state.
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
morph_rendered_beauty_report="$input_dir/surface-headless-morph-rendered-beauty.json"
morph_rendered_beauty_runtime="$input_dir/morph-rendered-beauty-runtime.json"
morph_rendered_beauty_visual="$input_dir/morph-rendered-beauty-visual.json"
morph_rendered_beauty_goldens="$input_dir/morph-rendered-beauty-goldens/headless"
app_model_report="$input_dir/surface-headless-app-model.json"
accessibility_report="$input_dir/surface-headless-release-accessibility.json"
events_report="$input_dir/surface-headless-block-events.json"
report_path="$report_dir/surface-inspector.json"
html_path="$report_dir/surface-inspector.html"
morph_rendered_beauty_source="examples/surface_morph_rendered_studio_shell.tetra"

write_drift_golden() {
  local current="$1"
  local golden="$2"
  mkdir -p "$(dirname "$golden")"
  python3 - "$current" "$golden" <<'PY'
import sys

src, dst = sys.argv[1], sys.argv[2]
data = bytearray(open(src, "rb").read())
if not data:
    raise SystemExit(f"empty frame artifact: {src}")
data[0] ^= 1
open(dst, "wb").write(data)
PY
}

go run ./tools/cmd/surface-runtime-smoke --mode headless-block-system --source examples/surface_block_system.tetra --report "$block_report"
go run ./tools/cmd/surface-runtime-smoke --mode headless-morph --source examples/surface_morph_command_palette.tetra --report "$morph_report"
go run ./tools/cmd/surface-runtime-smoke --mode headless-morph --source "$morph_rendered_beauty_source" --report "$morph_rendered_beauty_runtime"
morph_rendered_beauty_artifacts="$input_dir/surface-headless-morph-artifacts"
write_drift_golden "$morph_rendered_beauty_artifacts/surface-morph-rendered-beauty-frame-order-1-initial.rgba" "$morph_rendered_beauty_goldens/order-1-initial.rgba"
write_drift_golden "$morph_rendered_beauty_artifacts/surface-morph-rendered-beauty-frame-order-2-focused.rgba" "$morph_rendered_beauty_goldens/order-2-focused.rgba"
write_drift_golden "$morph_rendered_beauty_artifacts/surface-morph-rendered-beauty-frame-order-3-motion.rgba" "$morph_rendered_beauty_goldens/order-3-motion.rgba"
go run ./tools/cmd/surface-visual-diff \
  --runtime-report "$morph_rendered_beauty_runtime" \
  --required-target headless \
  --golden-artifact "$morph_rendered_beauty_source,headless,1,$morph_rendered_beauty_goldens/order-1-initial.rgba" \
  --golden-artifact "$morph_rendered_beauty_source,headless,2,$morph_rendered_beauty_goldens/order-2-focused.rgba" \
  --golden-artifact "$morph_rendered_beauty_source,headless,3,$morph_rendered_beauty_goldens/order-3-motion.rgba" \
  --out "$morph_rendered_beauty_visual"
go run ./tools/cmd/surface-runtime-smoke \
  --mode headless-morph \
  --source "$morph_rendered_beauty_source" \
  --report "$morph_rendered_beauty_runtime" \
  --visual-report "$morph_rendered_beauty_visual" \
  --morph-rendered-beauty-report "$morph_rendered_beauty_report"
go run ./tools/cmd/validate-surface-morph-rendered-beauty --report "$morph_rendered_beauty_report"
go run ./tools/cmd/surface-runtime-smoke --mode headless-app-model --source examples/surface_app_model.tetra --report "$app_model_report"
go run ./tools/cmd/surface-runtime-smoke --mode headless-release-accessibility --source examples/surface_release_accessibility.tetra --report "$accessibility_report"
go run ./tools/cmd/surface-runtime-smoke --mode headless-block-events --source examples/surface_block_events.tetra --report "$events_report"
go run ./tools/cmd/surface-inspector \
  --runtime-report "block:$block_report" \
  --runtime-report "morph:$morph_report" \
  --runtime-report "morph-rendered-beauty:$morph_rendered_beauty_report" \
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
echo "Surface inspector Morph-to-pixels report: $morph_rendered_beauty_report"
echo "Surface inspector artifact hashes: $report_dir/artifact-hashes.json"
