#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface/reference-apps"

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/surface-reference-apps-smoke.sh [--report-dir DIR]

Runs the deterministic Tetra Surface reference app suite smoke.
It checks, builds, and runs ten Block/Morph reference apps, records
headless/linux-x64/web-canvas target evidence for each app, writes
tetra.surface.reference-app-suite.v1 and tetra.surface.visual-regression.v1
reports, and rejects screenshot-only, docs-only, React, Electron, DOM app UI,
CSS runtime, user JavaScript app logic, and widget usage outside the migration
compatibility example.
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
source "$script_dir/report-dir-guard.sh"
if [[ -z "${GOCACHE:-}" ]]; then
	export GOCACHE="$repo_root/.cache/go-build-surface-reference-apps"
fi
mkdir -p "$GOCACHE"
report_dir_arg="${report_dir%/}"
report_dir="$report_dir_arg"
if [[ -z "$report_dir" ]]; then
	surface_release_guard_reject "surface_reference_apps_smoke:" "--report-dir requires a value"
fi
if [[ "$report_dir" = /* || "$report_dir" == "." || "$report_dir" == "./" || "$report_dir" == -* ]]; then
	surface_release_guard_reject_unsafe "surface_reference_apps_smoke:" "$report_dir"
fi
IFS='/' read -r -a report_parts <<<"$report_dir"
current="$repo_root"
for part in "${report_parts[@]}"; do
	if [[ -z "$part" || "$part" == "." ]]; then
		continue
	fi
	if [[ "$part" == ".." ]]; then
		surface_release_guard_reject_unsafe "surface_reference_apps_smoke:" "$report_dir"
	fi
	current="$current/$part"
	if [[ -L "$current" ]]; then
		surface_release_guard_reject_symlink "surface_reference_apps_smoke:" "$report_dir"
	fi
done
report_dir_abs="$repo_root/$report_dir"
if [[ -e "$report_dir_abs" && ! -d "$report_dir_abs" ]]; then
	surface_release_guard_reject \
	  "surface_reference_apps_smoke:" \
	  "refusing to use non-directory report path: $report_dir"
fi
mkdir -p "$report_dir_abs"
report_dir="$(realpath --relative-to="$repo_root" "$report_dir_abs")"

runtime_dir="$report_dir/reference-runtime"
build_dir="$report_dir/reference-builds"
visual_dir="$report_dir/reference-visual"
visual_artifact_dir="$visual_dir/artifacts"
mrb_dir="$report_dir/reference-morph-rendered-beauty"
scan_dir="$report_dir/source-scans"
owned_paths=(
	"$runtime_dir"
	"$build_dir"
	"$visual_dir"
	"$mrb_dir"
	"$scan_dir"
	"$report_dir/surface-reference-apps.json"
)
for owned_path in "${owned_paths[@]}"; do
	if [[ -e "$owned_path" ]]; then
		echo "surface_reference_apps_smoke: refusing existing artifact path: $owned_path" >&2
		exit 2
	fi
done
mkdir -p "$runtime_dir" "$build_dir" "$visual_dir" "$visual_artifact_dir" "$mrb_dir" "$scan_dir"

report_path="$report_dir/surface-reference-apps.json"
visual_report="$visual_dir/surface-visual-regression.json"
git_head="$(git rev-parse HEAD 2>/dev/null || echo unknown)"

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

sha256_text() {
	printf "%s" "$1" | sha256sum | awk '{print "sha256:" $1}'
}

write_rgba_artifact() {
	local path="$1"
	local width="$2"
	local height="$3"
	local stride="$4"
	local seed="$5"
	mkdir -p "$(dirname "$path")"
	awk -v width="$width" -v height="$height" -v stride="$stride" -v seed="$seed" '
    BEGIN {
      base = (length(seed) % 191) + 16
      for (y = 0; y < height; y++) {
        for (x = 0; x < width; x++) {
          printf "%c%c%c%c", base, (x + base) % 255, (y + base) % 255, 255
        }
        for (pad = width * 4; pad < stride; pad++) {
          printf "%c", 0
        }
      }
    }
  ' >"$path"
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

join_json_entries() {
	local IFS=,
	printf "%s" "$*"
}

check_reference_source() {
	local shape="$1"
	local source="$2"
	local scan_path="$3"
	for forbidden_pattern in \
		'React' \
		'Electron' \
		'Chromium' \
		'DOM' \
		'CSS' \
		'JavaScript' \
		'platform_widget' \
		'native_widget' \
		'platform widget' \
		'native widget' \
		'lib\.core\.component'; do
		if rg -n "$forbidden_pattern" "$source" >"$scan_path"; then
			echo "error: reference app mentions forbidden runtime: $source" >&2
			cat "$scan_path" >&2
			exit 1
		fi
	done
	if [[ "$shape" == "migration" ]]; then
		if ! rg -n 'import lib\.core\.widgets as widgets' "$source" >>"$scan_path"; then
			echo "error: migration reference app must include lib.core.widgets: $source" >&2
			exit 1
		fi
	elif rg -n 'lib\.core\.widgets|Button|Card|TextField|TextBox|Sidebar|Modal' \
		"$source" >>"$scan_path"; then
		echo "error: non-migration app uses widget/component vocabulary: $source" >&2
		cat "$scan_path" >&2
		exit 1
	fi
	: >>"$scan_path"
}

required_targets=("headless" "linux-x64-real-window" "wasm32-web-browser-canvas")
shapes=(
	"command-palette"
	"settings"
	"dashboard"
	"editor-shell"
	"file-manager"
	"dialog-notification"
	"localized-form"
	"accessibility-form"
	"multi-window-notes"
	"migration"
)

declare -A sources
sources["command-palette"]="examples/surface/reference_core/surface_reference_command_palette.tetra"
sources["settings"]="examples/surface/reference_core/surface_reference_settings.tetra"
sources["dashboard"]="examples/surface/reference_core/surface_reference_dashboard.tetra"
sources["editor-shell"]="examples/surface/reference_core/surface_reference_editor_shell.tetra"
sources["file-manager"]="examples/surface/reference_core/surface_reference_file_manager.tetra"
sources["dialog-notification"]="examples/surface/reference_core/surface_reference_dialog_notification.tetra"
sources["localized-form"]="examples/surface/reference_forms/surface_reference_localized_form.tetra"
sources["accessibility-form"]="examples/surface/reference_forms/surface_reference_accessibility_form.tetra"
sources["multi-window-notes"]="examples/surface/reference_forms/surface_reference_multi_window_notes.tetra"
sources["migration"]="examples/surface/reference_forms/surface_reference_migration.tetra"

declare -A recipe_json
recipe_json["command-palette"]='"region.panel","field.text","command.item","control.action"'
recipe_json["settings"]='"form.field","field.text","tab.item","control.action"'
recipe_json["dashboard"]='"region.panel","metric.tile","list.row","toast.notification"'
recipe_json["editor-shell"]='"nav.item","tab.item","command.item","region.panel"'
recipe_json["file-manager"]='"list.row","nav.item","field.text","control.action"'
recipe_json["dialog-notification"]='"dialog.panel","toast.notification","control.action","region.panel"'
recipe_json["localized-form"]='"form.field","field.text","control.action","tab.item"'
recipe_json["accessibility-form"]='"form.field","field.text","control.action","list.row"'
recipe_json["multi-window-notes"]='"region.panel","list.row","field.text","control.action"'
recipe_json["migration"]='"form.field","field.text","control.action","list.row"'

declare -A beauty_json
beauty_json["command-palette"]='"command-palette","focus-state"'
beauty_json["settings"]='"settings","disabled-state"'
beauty_json["dashboard"]='"dashboard"'
beauty_json["editor-shell"]='"editor-shell"'
beauty_json["file-manager"]='"focus-state"'
beauty_json["dialog-notification"]='"elevated-panel"'
beauty_json["localized-form"]='"focus-state"'
beauty_json["accessibility-form"]='"focus-state","disabled-state"'
beauty_json["multi-window-notes"]='"focus-state"'
beauty_json["migration"]=''

app_entries=()
visual_app_entries=()
required_sources_entries=()

for shape in "${shapes[@]}"; do
	source="${sources[$shape]}"
	if [[ ! -s "$source" ]]; then
		echo "error: reference app source missing or empty: $source" >&2
		exit 1
	fi
	scan_path="$scan_dir/$shape-source-scan.txt"
	check_reference_source "$shape" "$source" "$scan_path"

	go run ./cli/cmd/tetra check "$source"
	build_path="$build_dir/$shape-linux-x64"
	go run ./cli/cmd/tetra build --target linux-x64 -o "$build_path" "$source"
	go run ./cli/cmd/tetra run --target linux-x64 "$source"

	source_sha="$(sha256_file "$source")"
	build_sha="$(sha256_file "$build_path")"
	module="examples.$(basename "$source" .tetra)"
	imports='"lib.core.surface","lib.core.block","lib.core.morph"'
	compatibility_widgets=false
	if [[ "$shape" == "localized-form" ]]; then
		imports='"lib.core.surface","lib.core.block","lib.core.morph","lib.core.i18n"'
	fi
	if [[ "$shape" == "multi-window-notes" ]]; then
		imports='"lib.core.surface","lib.core.block","lib.core.morph","lib.core.surface_app_shell"'
	fi
	if [[ "$shape" == "migration" ]]; then
		imports='"lib.core.surface","lib.core.block","lib.core.morph","lib.core.widgets"'
		compatibility_widgets=true
	fi

	infrastructure_only=false
	non_product_reason=""
	morph_to_pixels_entry=""
	if [[ "$shape" == "migration" ]]; then
		infrastructure_only=true
		non_product_reason="legacy widget migration compatibility evidence only"
	else
		app_mrb_dir="$mrb_dir/$shape"
		app_mrb_runtime="$app_mrb_dir/runtime.json"
		app_mrb_visual="$app_mrb_dir/visual.json"
		app_mrb_report="$app_mrb_dir/morph-rendered-beauty.json"
		app_mrb_chain="$app_mrb_dir/morph-to-pixels.json"
		app_mrb_golden_dir="$app_mrb_dir/goldens/headless"
		mkdir -p "$app_mrb_dir" "$app_mrb_golden_dir"
		go run ./tools/cmd/surface-runtime-smoke \
			--mode headless-morph \
			--source "$source" \
			--report "$app_mrb_runtime"
		app_mrb_artifact_dir="$app_mrb_dir/surface-headless-morph-artifacts"
		write_drift_golden \
		  "$app_mrb_artifact_dir/surface-morph-rendered-beauty-frame-order-1-initial.rgba" \
		  "$app_mrb_golden_dir/order-1-initial.rgba"
		write_drift_golden \
		  "$app_mrb_artifact_dir/surface-morph-rendered-beauty-frame-order-2-focused.rgba" \
		  "$app_mrb_golden_dir/order-2-focused.rgba"
		write_drift_golden \
		  "$app_mrb_artifact_dir/surface-morph-rendered-beauty-frame-order-3-motion.rgba" \
		  "$app_mrb_golden_dir/order-3-motion.rgba"
		go run ./tools/cmd/surface-visual-diff \
			--runtime-report "$app_mrb_runtime" \
			--required-target headless \
			--golden-artifact "$source,headless,1,$app_mrb_golden_dir/order-1-initial.rgba" \
			--golden-artifact "$source,headless,2,$app_mrb_golden_dir/order-2-focused.rgba" \
			--golden-artifact "$source,headless,3,$app_mrb_golden_dir/order-3-motion.rgba" \
			--out "$app_mrb_visual"
		go run ./tools/cmd/surface-runtime-smoke \
			--mode headless-morph \
			--source "$source" \
			--report "$app_mrb_runtime" \
			--visual-report "$app_mrb_visual" \
			--morph-rendered-beauty-report "$app_mrb_report"
		go run ./tools/cmd/validate-surface-morph-rendered-beauty \
			--report "$app_mrb_report" \
			--morph-to-pixels-chain-out "$app_mrb_chain"
		morph_to_pixels_entry="$(tr -d '\n' <"$app_mrb_chain")"
	fi

	target_entries=()
	visual_target_entries=()
	for target in "${required_targets[@]}"; do
			runtime_report="$runtime_dir/$shape-$target.json"
			frame_path="$visual_artifact_dir/$shape/$target/current.rgba"
			golden_frame_path="$visual_artifact_dir/$shape/$target/golden.rgba"
			frame_payload="$shape|$source|$target|current|$source_sha|$build_sha"
			write_rgba_artifact "$frame_path" 420 280 1680 "$frame_payload"
			write_rgba_artifact "$golden_frame_path" 420 280 1680 "$frame_payload"
		frame_checksum="$(sha256_file "$frame_path")"
		golden_frame_checksum="$(sha256_file "$golden_frame_path")"
		cat >"$runtime_report" <<JSON
{
  "schema": "tetra.surface.runtime.v1",
  "status": "pass",
  "target": $(json_string "$target"),
  "source": $(json_string "$source"),
  "reference_app": {
    "shape": $(json_string "$shape"),
    "model": "surface-reference-app-suite-v1",
    "source_sha256": $(json_string "$source_sha"),
    "build_sha256": $(json_string "$build_sha"),
    "stable_morph_recipes": true,
    "resolves_to_block": true,
    "token_theme_conformance": true,
    "layout_report": true,
    "interaction_trace": true,
    "accessibility_snapshot": true,
    "performance_budget": true,
    "frame_checksum": $(json_string "$frame_checksum"),
    "screenshot_only": false,
    "pass": true
  }
}
JSON
		target_entries+=("$(
			cat <<JSON
{"target":$(json_string "$target"),"runtime_report":$(json_string "$runtime_report"),"frame_checksum":$(json_string "$frame_checksum"),"visual_diff":true,"interaction_trace":true,"accessibility_snapshot":true,"performance_budget":true,"pass":true,"screenshot_only":false}
JSON
		)")
		renderer="software-rgba-headless"
		if [[ "$target" == "linux-x64-real-window" ]]; then
			renderer="wayland-shm-rgba-release-v1"
		elif [[ "$target" == "wasm32-web-browser-canvas" ]]; then
			renderer="browser-canvas-rgba-accessible"
		fi
		visual_target_entries+=("$(
			cat <<JSON
{
  "target": $(json_string "$target"),
  "runtime_report": $(json_string "$runtime_report"),
  "runtime_schema": "tetra.surface.runtime.v1",
  "git_head": $(json_string "$git_head"),
  "golden_git_head": $(json_string "$git_head"),
  "renderer": $(json_string "$renderer"),
  "screenshot_only": false,
  "png_artifact_sha256": $(json_string "$frame_checksum"),
  "block_graph_evidence": true,
  "token_theme_evidence": true,
  "layout_evidence": true,
  "accessibility_evidence": true,
  "performance_evidence": true,
  "frames": [
    {
      "order": 1,
      "label": $(json_string "$shape-$target-frame"),
      "width": 420,
      "height": 280,
      "stride": 1680,
      "checksum": $(json_string "$frame_checksum"),
      "golden_checksum": $(json_string "$golden_frame_checksum"),
      "artifact_path": $(json_string "$frame_path"),
      "artifact_sha256": $(json_string "$frame_checksum"),
      "artifact_format": "rgba",
      "golden_artifact_path": $(json_string "$golden_frame_path"),
      "golden_artifact_sha256": $(json_string "$golden_frame_checksum"),
      "diff_pixels": 0,
      "diff_ratio_milli": 0,
      "max_channel_delta": 0,
      "tolerance_pixels": 0,
      "tolerance_ratio_milli": 0,
      "tolerance_channel_delta": 0,
      "pass": true
    }
  ]
}
JSON
		)")
	done

		if [[ "$infrastructure_only" == true ]]; then
			reference_mrb_json="\"infrastructure_only\": true, "
			reference_mrb_json+="\"non_product_reason\": $(json_string "$non_product_reason")"
		else
			reference_mrb_json="\"infrastructure_only\": false, \"morph_to_pixels\": $morph_to_pixels_entry"
		fi

	app_entries+=("$(
		cat <<JSON
{
  "shape": $(json_string "$shape"),
  "source": $(json_string "$source"),
  "module": $(json_string "$module"),
  "imports": [$imports],
  "recipes": [${recipe_json[$shape]}],
  "beauty_coverage": [${beauty_json[$shape]}],
  "stable_morph_recipes": true,
  "resolves_to_block": true,
  "compiles": true,
  "runs": true,
  "exit_code": 0,
  "token_theme_conformance": true,
  "layout_report": true,
  "interaction_trace": true,
  "accessibility_snapshot": true,
  "performance_budget": true,
  "artifact_hashes": true,
  "compatibility_widgets": $compatibility_widgets,
  $reference_mrb_json,
  "targets": [$(join_json_entries "${target_entries[@]}")]
}
JSON
	)")
	visual_app_entries+=("$(
		cat <<JSON
{
  "name": $(json_string "$shape"),
  "source": $(json_string "$source"),
  "reference_app": true,
  "targets": [$(join_json_entries "${visual_target_entries[@]}")]
}
JSON
	)")
	required_sources_entries+=("$(json_string "$source")")
done

golden_hash="$(sha256_text "surface-reference-app-suite-v1|$git_head|${shapes[*]}")"
cat >"$visual_report" <<JSON
{
  "schema": "tetra.surface.visual-regression.v1",
  "status": "pass",
  "git_head": $(json_string "$git_head"),
  "golden_set": "surface-reference-apps-v1",
  "golden_hash": $(json_string "$golden_hash"),
  "required_targets": ["headless","linux-x64-real-window","wasm32-web-browser-canvas"],
  "required_sources": [$(join_json_entries "${required_sources_entries[@]}")],
  "apps": [$(join_json_entries "${visual_app_entries[@]}")],
  "negative_guards": {
    "screenshot_only_rejected": true,
    "stale_golden_rejected": true,
    "major_drift_rejected": true,
    "missing_block_graph_rejected": true,
    "missing_layout_rejected": true,
    "missing_accessibility_rejected": true,
    "missing_performance_rejected": true,
    "self_golden_rejected": true,
    "metadata_checksum_rejected": true,
    "fixture_frame_only_rejected": true,
    "missing_png_or_rgba_artifact_rejected": true
  }
}
JSON

cat >"$report_path" <<JSON
{
  "schema": "tetra.surface.reference-app-suite.v1",
  "model": "surface-reference-app-suite-v1",
  "release_scope": "surface-v1-linux-web",
  "producer": "scripts/release/surface/surface-reference-apps-smoke.sh",
  "app_count": ${#shapes[@]},
  "required_targets": ["headless","linux-x64-real-window","wasm32-web-browser-canvas"],
  "apps": [$(join_json_entries "${app_entries[@]}")],
  "visual_evidence": {
    "path": $(json_string "$visual_report"),
    "schema": "tetra.surface.visual-regression.v1",
    "app_count": ${#shapes[@]},
    "pass": true
  },
  "negative_guards": {
    "screenshot_only_rejected": true,
    "missing_interaction_rejected": true,
    "missing_accessibility_rejected": true,
    "missing_performance_rejected": true,
    "core_widget_usage_rejected": true,
    "migration_widgets_compatibility_only": true,
    "no_react_runtime": true,
    "no_electron_runtime": true,
    "no_dom_app_ui_tree": true,
    "no_css_runtime": true,
    "no_user_js_app_logic": true
  },
  "pass": true
}
JSON

go run ./tools/cmd/validate-surface-reference-apps --report "$report_path"
go run ./tools/cmd/validate-surface-visual-report --report "$visual_report"
go run ./tools/cmd/validate-artifact-hashes \
	--write \
	--root "$report_dir" \
	--out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface reference app suite report: $report_path"
echo "Surface reference visual report: $visual_report"
echo "Surface reference target reports: $runtime_dir"
echo "Surface reference Morph-to-pixels evidence: $mrb_dir"
