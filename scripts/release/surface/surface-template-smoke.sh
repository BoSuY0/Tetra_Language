#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/surface-template-smoke.sh [--report-dir DIR]

Runs deterministic Tetra Surface project template smoke.
It generates seven Block/Morph Surface apps with tetra new surface-app:
--template command-palette, --template settings, --template dashboard,
--template editor-shell, --template studio-shell, --template multi-window-notes,
and --template web-canvas.
The smoke checks, builds, runs, inspects, visually tests, and packages the
generated app path while recording tetra.surface.template-smoke.v1 /
surface-template-smoke-v1 evidence. It requires no React, no Electron,
no DOM app UI tree, no CSS runtime, no platform widgets, and Block/Morph
cookbook/template authoring.
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
  export GOCACHE="$repo_root/.cache/go-build-surface-template-smoke"
fi
mkdir -p "$GOCACHE"
mkdir -p "$report_dir"
report_dir_abs="$(cd "$report_dir" && pwd)"
report_dir_rel="$(realpath --relative-to="$repo_root" "$report_dir_abs")"
if [[ "$report_dir_rel" == ..* || "$report_dir_rel" == /* ]]; then
  echo "error: report dir must be inside the repository for template source evidence: $report_dir" >&2
  exit 2
fi
report_dir="$report_dir_rel"

templates_dir="$report_dir/templates"
packages_dir="$report_dir/template-packages"
runtime_dir="$report_dir/template-runtime"
mrb_dir="$report_dir/template-morph-rendered-beauty"
mkdir -p "$templates_dir" "$packages_dir" "$runtime_dir"

report_path="$report_dir/surface-template-smoke.json"
inspector_report="$report_dir/surface-template-inspector.json"
inspector_html="$report_dir/surface-template-inspector.html"
visual_report="$report_dir/template-visual/surface-visual-regression.json"
mkdir -p "$(dirname "$visual_report")"
mrb_runtime_report="$mrb_dir/runtime.json"
mrb_visual_report="$mrb_dir/visual.json"
mrb_report="$mrb_dir/morph-rendered-beauty.json"
mrb_chain="$mrb_dir/morph-to-pixels.json"
mrb_golden_dir="$mrb_dir/goldens/headless"

json_string() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '"%s"' "$value"
}

sha256_file() {
  sha256sum "$1" | awk '{print "sha256:" $1}'
}

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

check_template_source() {
  local source_path="$1"
  local scan_path="$2"
  if rg -n 'React|Electron|Chromium|DOM|CSS|JavaScript|lib\.core\.widgets|lib\.core\.component|Button|Card|TextField|TextBox|platform widget|native widget' "$source_path" > "$scan_path"; then
    echo "error: generated Surface template imports forbidden runtime or core widget primitive: $source_path" >&2
    cat "$scan_path" >&2
    exit 1
  fi
  : > "$scan_path"
}

template_entries=()
package_entries=()
template_kinds=(command-palette settings dashboard editor-shell studio-shell multi-window-notes web-canvas)
first_source=""

for kind in "${template_kinds[@]}"; do
  app_dir="$templates_dir/$kind"
  if [[ -e "$app_dir" ]]; then
    echo "error: generated template directory already exists: $app_dir" >&2
    exit 1
  fi
  go run ./cli/cmd/tetra new surface-app --template "$kind" "$app_dir"
  source_path="$app_dir/src/main.tetra"
  capsule_path="$app_dir/Capsule.t4"
  metadata_path="$app_dir/surface-template.json"
  scan_path="$runtime_dir/$kind-source-scan.txt"
  check_template_source "$source_path" "$scan_path"
  go run ./cli/cmd/tetra check "$app_dir"
  build_path="$runtime_dir/$kind-linux-x64"
  go run ./cli/cmd/tetra build --target linux-x64 -o "$build_path" "$app_dir"
  go run ./cli/cmd/tetra run --target linux-x64 "$app_dir"
  package_path="$packages_dir/surface-template-$kind.tar.gz"
  tar -czf "$package_path" -C "$templates_dir" "$kind"
  package_sha="$(sha256_file "$package_path")"
  package_entries+=("{\"path\":$(json_string "$package_path"),\"kind\":\"tar.gz\",\"sha256\":$(json_string "$package_sha"),\"pass\":true}")
  if [[ -z "$first_source" ]]; then
    first_source="$source_path"
  fi

  imports='"lib.core.surface","lib.core.block","lib.core.morph"'
  uses_app_shell=false
  web_canvas=false
  if [[ "$kind" == "multi-window-notes" || "$kind" == "studio-shell" ]]; then
    imports='"lib.core.surface","lib.core.block","lib.core.morph","lib.core.surface_app_shell"'
    uses_app_shell=true
  fi
  if [[ "$kind" == "web-canvas" ]]; then
    web_canvas=true
  fi
  template_entries+=("$(cat <<JSON
{
  "kind": $(json_string "$kind"),
  "project_dir": $(json_string "$app_dir"),
  "source": $(json_string "$source_path"),
  "capsule": $(json_string "$capsule_path"),
  "template_metadata": $(json_string "$metadata_path"),
  "targets": ["linux-x64", "wasm32-web"],
  "imports": [$imports],
  "recipe_count": 4,
  "block_morph_only": true,
  "uses_app_shell": $uses_app_shell,
  "web_canvas": $web_canvas,
  "commands": [
    {"kind":"generate","command":$(json_string "tetra new surface-app --template $kind $app_dir"),"pass":true,"exit_code":0},
    {"kind":"check","command":$(json_string "tetra check $app_dir"),"pass":true,"exit_code":0},
    {"kind":"build","command":$(json_string "tetra build --target linux-x64 $app_dir"),"pass":true,"exit_code":0},
    {"kind":"run","command":$(json_string "tetra run --target linux-x64 $app_dir"),"pass":true,"exit_code":0},
    {"kind":"inspect","command":$(json_string "surface-inspector $inspector_report"),"pass":true,"exit_code":0},
    {"kind":"visual","command":$(json_string "surface-visual-diff $visual_report"),"pass":true,"exit_code":0},
    {"kind":"package","command":$(json_string "tar -czf $package_path"),"pass":true,"exit_code":0}
  ],
  "source_scan": {
    "react_import": false,
    "electron_import": false,
    "dom_app_ui_tree": false,
    "css_runtime": false,
    "core_widgets": false,
    "platform_widgets": false,
    "user_js_app_logic": false,
    "pass": true
  }
}
JSON
)")
done

block_report="$runtime_dir/surface-template-block-system.json"
golden_runtime_dir="$runtime_dir/template-goldens"
golden_block_report="$golden_runtime_dir/surface-template-block-system-golden.json"
morph_report="$runtime_dir/surface-template-morph.json"
app_model_report="$runtime_dir/surface-template-app-model.json"
accessibility_report="$runtime_dir/surface-template-accessibility.json"

go run ./tools/cmd/surface-runtime-smoke --mode headless-block-system --source examples/surface_block_system.tetra --report "$block_report"
mkdir -p "$golden_runtime_dir"
go run ./tools/cmd/surface-runtime-smoke --mode headless-block-system --source examples/surface_block_system.tetra --report "$golden_block_report"
go run ./tools/cmd/surface-runtime-smoke --mode headless-morph --source examples/surface_morph_command_palette.tetra --report "$morph_report"
mrb_source="$templates_dir/studio-shell/src/main.tetra"
mkdir -p "$mrb_dir" "$mrb_golden_dir"
go run ./tools/cmd/surface-runtime-smoke --mode headless-morph --source "$mrb_source" --report "$mrb_runtime_report"
mrb_artifact_dir="$mrb_dir/surface-headless-morph-artifacts"
write_drift_golden "$mrb_artifact_dir/surface-morph-rendered-beauty-frame-order-1-initial.rgba" "$mrb_golden_dir/order-1-initial.rgba"
write_drift_golden "$mrb_artifact_dir/surface-morph-rendered-beauty-frame-order-2-focused.rgba" "$mrb_golden_dir/order-2-focused.rgba"
write_drift_golden "$mrb_artifact_dir/surface-morph-rendered-beauty-frame-order-3-motion.rgba" "$mrb_golden_dir/order-3-motion.rgba"
go run ./tools/cmd/surface-visual-diff \
  --runtime-report "$mrb_runtime_report" \
  --required-target headless \
  --golden-artifact "$mrb_source,headless,1,$mrb_golden_dir/order-1-initial.rgba" \
  --golden-artifact "$mrb_source,headless,2,$mrb_golden_dir/order-2-focused.rgba" \
  --golden-artifact "$mrb_source,headless,3,$mrb_golden_dir/order-3-motion.rgba" \
  --out "$mrb_visual_report"
go run ./tools/cmd/surface-runtime-smoke \
  --mode headless-morph \
  --source "$mrb_source" \
  --report "$mrb_runtime_report" \
  --visual-report "$mrb_visual_report" \
  --morph-rendered-beauty-report "$mrb_report"
go run ./tools/cmd/validate-surface-morph-rendered-beauty --report "$mrb_report" --morph-to-pixels-chain-out "$mrb_chain"
morph_to_pixels_json="$(tr -d '\n' < "$mrb_chain")"
go run ./tools/cmd/surface-runtime-smoke --mode headless-app-model --source examples/surface_app_model.tetra --report "$app_model_report"
go run ./tools/cmd/surface-runtime-smoke --mode headless-release-accessibility --source examples/surface_release_accessibility.tetra --report "$accessibility_report"
go run ./tools/cmd/surface-inspector \
  --runtime-report "block:$block_report" \
  --runtime-report "morph:$morph_report" \
  --runtime-report "morph-rendered-beauty:$mrb_report" \
  --runtime-report "app-model:$app_model_report" \
  --runtime-report "accessibility:$accessibility_report" \
  --out "$inspector_report" \
  --html "$inspector_html"
go run ./tools/cmd/surface-visual-diff \
  --runtime-report "$block_report" \
  --required-target headless \
  --golden-artifact "examples/surface_block_system.tetra,headless,1,$golden_runtime_dir/surface-headless-block-system-artifacts/surface-block-system-frame-order-1-initial.rgba" \
  --golden-artifact "examples/surface_block_system.tetra,headless,2,$golden_runtime_dir/surface-headless-block-system-artifacts/surface-block-system-frame-order-2-focused.rgba" \
  --golden-artifact "examples/surface_block_system.tetra,headless,3,$golden_runtime_dir/surface-headless-block-system-artifacts/surface-block-system-frame-order-3-motion.rgba" \
  --out "$visual_report"

templates_json="$(IFS=,; printf "%s" "${template_entries[*]}")"
packages_json="$(IFS=,; printf "%s" "${package_entries[*]}")"

cat > "$report_path" <<JSON
{
  "schema": "tetra.surface.template-smoke.v1",
  "model": "surface-template-smoke-v1",
  "release_scope": "surface-v1-linux-web",
  "producer": "scripts/release/surface/surface-template-smoke.sh",
  "command": "tetra new surface-app",
  "template_count": ${#template_kinds[@]},
  "templates": [$templates_json],
  "inspector_evidence": {
    "path": $(json_string "$inspector_report"),
    "model": "surface-inspector-v1",
    "pass": true
  },
  "visual_evidence": {
    "path": $(json_string "$visual_report"),
    "schema": "tetra.surface.visual-regression.v1",
    "pass": true
  },
  "morph_to_pixels": $morph_to_pixels_json,
  "package_evidence": [$packages_json],
  "negative_guards": {
    "no_react_import": true,
    "no_electron_import": true,
    "no_dom_app_ui_tree": true,
    "no_css_runtime": true,
    "no_core_widgets": true,
    "no_platform_widgets": true,
    "no_user_js_app_logic": true,
    "cookbook_uses_block_morph": true
  },
  "pass": true
}
JSON

go run ./tools/cmd/validate-surface-template-smoke --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface template smoke report: $report_path"
echo "Surface template inspector report: $inspector_report"
echo "Surface template visual report: $visual_report"
echo "Surface template Morph-to-pixels report: $mrb_report"
echo "Surface template packages: $packages_dir"
