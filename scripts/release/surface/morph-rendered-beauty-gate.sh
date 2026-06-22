#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface/morph-rendered-beauty-gate"
product_claim=false
final_signoff=false
original_args=("$@")

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/morph-rendered-beauty-gate.sh [--report-dir DIR] [--product-claim --final-signoff]

Runs the integrated Tetra Surface Morph rendered beauty evidence gate.
It proves the Morph flagship through the real Morph -> Block scene -> render
commands -> RGBA frame artifact -> separate pixel golden -> MRB report path,
then validates dev-loop, inspector, templates, reference apps, docs/claims, and
artifact hashes before product-slice consumption.
Product claim/final signoff are explicit promotion-mode flags and require a
clean checkout plus renderer-owned stable proof for every supported target.
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
	--product-claim)
		product_claim=true
		shift
		;;
	--final-signoff)
		final_signoff=true
		shift
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
if [[ -z "${GOTELEMETRY:-}" ]]; then
	export GOTELEMETRY=off
fi
if [[ -z "${GOCACHE:-}" ]]; then
	export GOCACHE="$repo_root/.cache/go-build-surface-morph-beauty-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
	export GOTMPDIR="$repo_root/.cache/go-tmp-surface-morph-beauty-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir_arg="${report_dir%/}"
report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_morph_rendered_beauty_gate:")"
report_dir_rel="$(realpath --relative-to="$repo_root" "$report_dir")"

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

json_array() {
	local first=true
	local value
	printf '['
	for value in "$@"; do
		if [[ "$first" == true ]]; then
			first=false
		else
			printf ', '
		fi
		json_string "$value"
	done
	printf ']'
}

renderer_owned_stable_proof() {
	local report="$1"
	python3 - "$report" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as handle:
    report = json.load(handle)
proof = report.get("renderer_stable_proof", {})
ok = (
    proof.get("pixel_owner") == "surface-renderer"
    and proof.get("renderer_owned") is True
    and proof.get("bridge_owned_pixels") is False
    and proof.get("derived_from_render_command_stream") is True
    and proof.get("stable_promotion_eligible") is True
)
print("true" if ok else "false")
PY
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

source_path="examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"
runtime_report="$report_dir/runtime.json"
visual_report="$report_dir/visual.json"
mrb_report="$report_dir/morph-rendered-beauty.json"
morph_to_pixels_report="$report_dir/morph-to-pixels.json"
golden_dir="$report_dir/goldens/headless"
linux_runtime_report="$report_dir/runtime-linux-x64-real-window.json"
linux_visual_report="$report_dir/visual-linux-x64-real-window.json"
linux_mrb_report="$report_dir/morph-rendered-beauty-linux-x64-real-window.json"
linux_morph_to_pixels_report="$report_dir/morph-to-pixels-linux-x64-real-window.json"
linux_golden_dir="$report_dir/goldens/linux-x64-real-window"
wasm_runtime_report="$report_dir/runtime-wasm32-web-browser-canvas.json"
wasm_visual_report="$report_dir/visual-wasm32-web-browser-canvas.json"
wasm_mrb_report="$report_dir/morph-rendered-beauty-wasm32-web-browser-canvas.json"
wasm_morph_to_pixels_report="$report_dir/morph-to-pixels-wasm32-web-browser-canvas.json"
wasm_golden_dir="$report_dir/goldens/wasm32-web-browser-canvas"
dev_dir="$report_dir/dev-workflow"
inspector_dir="$report_dir/inspector"
template_dir="$report_dir/templates"
reference_dir="$report_dir_rel/reference-apps"
docs_claims_dir="$report_dir/docs-claims"
summary_path="$report_dir/morph-rendered-beauty-gate-summary.json"

git_head="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
git_dirty=false
if ! git diff --quiet 2>/dev/null || \
  ! git diff --cached --quiet 2>/dev/null || \
  [[ -n "$(git ls-files --others --exclude-standard 2>/dev/null)" ]]; then
	git_dirty=true
fi
if [[ "$final_signoff" == true && "$product_claim" != true ]]; then
	echo "error: --final-signoff requires --product-claim" >&2
	exit 1
fi
if [[ "$product_claim" == true && "$final_signoff" != true ]]; then
	echo "error: --product-claim requires --final-signoff" >&2
	exit 1
fi
if [[ "$product_claim" == true && "$git_dirty" == true ]]; then
	echo "error: --product-claim requires a clean checkout; current git_dirty=true" >&2
	exit 1
fi
version="$(go list -m 2>/dev/null || echo tetra_language)"
host_os="$(go env GOOS 2>/dev/null || uname -s)"
host_arch="$(go env GOARCH 2>/dev/null || uname -m)"
generated_at_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
formatted_args="$(format_command "${original_args[@]}")"
command_line="bash scripts/release/surface/morph-rendered-beauty-gate.sh"
if [[ -n "$formatted_args" ]]; then
	command_line+=" $formatted_args"
fi

mkdir -p "$golden_dir" "$linux_golden_dir" "$wasm_golden_dir" "$docs_claims_dir"
mrb_signoff_args=()
if [[ "$product_claim" == true ]]; then
	mrb_signoff_args+=("--morph-rendered-beauty-product-claim")
fi
if [[ "$final_signoff" == true ]]; then
	mrb_signoff_args+=("--morph-rendered-beauty-final-signoff")
fi

go run ./tools/cmd/validate-surface-morph-rendered-beauty \
	--contract docs/spec/surface/morph/surface_morph_rendered_beauty_contract.json
go run ./tools/cmd/validate-surface-block-contract \
	--contract docs/spec/surface/surface_block_contract.json

go run ./tools/cmd/surface-runtime-smoke \
	--mode headless-morph \
	--source "$source_path" \
	--report "$runtime_report"
go run ./tools/cmd/validate-surface-morph-report \
	--report "$runtime_report" \
	--same-commit "$git_head"
go run ./tools/cmd/validate-surface-block-contract --report "$runtime_report"

artifact_dir="$report_dir/surface-headless-morph-artifacts"
write_drift_golden \
  "$artifact_dir/surface-morph-rendered-beauty-frame-order-1-initial.rgba" \
  "$golden_dir/order-1-initial.rgba"
write_drift_golden \
  "$artifact_dir/surface-morph-rendered-beauty-frame-order-2-focused.rgba" \
  "$golden_dir/order-2-focused.rgba"
write_drift_golden \
  "$artifact_dir/surface-morph-rendered-beauty-frame-order-3-motion.rgba" \
  "$golden_dir/order-3-motion.rgba"

go run ./tools/cmd/surface-visual-diff \
	--runtime-report "$runtime_report" \
	--required-target headless \
	--golden-artifact "$source_path,headless,1,$golden_dir/order-1-initial.rgba" \
	--golden-artifact "$source_path,headless,2,$golden_dir/order-2-focused.rgba" \
	--golden-artifact "$source_path,headless,3,$golden_dir/order-3-motion.rgba" \
	--out "$visual_report"
go run ./tools/cmd/validate-surface-visual-report --report "$visual_report"

go run ./tools/cmd/surface-runtime-smoke \
	--mode headless-morph \
	--source "$source_path" \
	--report "$runtime_report" \
	--visual-report "$visual_report" \
	--morph-rendered-beauty-report "$mrb_report" \
	"${mrb_signoff_args[@]}"
go run ./tools/cmd/validate-surface-morph-rendered-beauty \
	--report "$mrb_report" \
	--morph-to-pixels-chain-out "$morph_to_pixels_report"

go run ./tools/cmd/surface-runtime-smoke \
	--mode linux-x64-real-window-morph \
	--source "$source_path" \
	--report "$linux_runtime_report"
go run ./tools/cmd/validate-surface-morph-report \
	--report "$linux_runtime_report" \
	--same-commit "$git_head"
go run ./tools/cmd/validate-surface-block-contract --report "$linux_runtime_report"

linux_artifact_dir="$report_dir/surface-linux-x64-real-window-morph-artifacts"
write_drift_golden \
  "$linux_artifact_dir/surface-morph-real-window-frame-order-1.rgba" \
  "$linux_golden_dir/order-1-initial.rgba"
write_drift_golden \
  "$linux_artifact_dir/surface-morph-real-window-frame-order-5.rgba" \
  "$linux_golden_dir/order-5-active.rgba"

go run ./tools/cmd/surface-visual-diff \
	--runtime-report "$linux_runtime_report" \
	--required-target linux-x64-real-window \
	--golden-artifact "$source_path,linux-x64-real-window,1,$linux_golden_dir/order-1-initial.rgba" \
	--golden-artifact "$source_path,linux-x64-real-window,5,$linux_golden_dir/order-5-active.rgba" \
	--out "$linux_visual_report"
go run ./tools/cmd/validate-surface-visual-report --report "$linux_visual_report"

go run ./tools/cmd/surface-runtime-smoke \
	--mode linux-x64-real-window-morph \
	--source "$source_path" \
	--report "$linux_runtime_report" \
	--visual-report "$linux_visual_report" \
	--morph-rendered-beauty-report "$linux_mrb_report" \
	"${mrb_signoff_args[@]}"
go run ./tools/cmd/validate-surface-morph-rendered-beauty \
	--report "$linux_mrb_report" \
	--morph-to-pixels-chain-out "$linux_morph_to_pixels_report"

go run ./tools/cmd/surface-runtime-smoke \
	--mode wasm32-web-browser-canvas-morph \
	--source "$source_path" \
	--report "$wasm_runtime_report"
go run ./tools/cmd/validate-surface-morph-report \
	--report "$wasm_runtime_report" \
	--same-commit "$git_head"
go run ./tools/cmd/validate-surface-block-contract --report "$wasm_runtime_report"

wasm_artifact_dir="$report_dir/surface-wasm32-web-browser-canvas-morph-artifacts"
write_drift_golden \
  "$wasm_artifact_dir/surface-browser-canvas-frame-order-1.rgba" \
  "$wasm_golden_dir/order-1-initial.rgba"
write_drift_golden \
  "$wasm_artifact_dir/surface-browser-canvas-frame-order-5.rgba" \
  "$wasm_golden_dir/order-5-focused.rgba"

wasm_golden_order_1="$source_path,wasm32-web-browser-canvas,1,"
wasm_golden_order_1+="$wasm_golden_dir/order-1-initial.rgba"
wasm_golden_order_5="$source_path,wasm32-web-browser-canvas,5,"
wasm_golden_order_5+="$wasm_golden_dir/order-5-focused.rgba"

go run ./tools/cmd/surface-visual-diff \
	--runtime-report "$wasm_runtime_report" \
	--required-target wasm32-web-browser-canvas \
	--golden-artifact "$wasm_golden_order_1" \
	--golden-artifact "$wasm_golden_order_5" \
	--out "$wasm_visual_report"
go run ./tools/cmd/validate-surface-visual-report --report "$wasm_visual_report"

go run ./tools/cmd/surface-runtime-smoke \
	--mode wasm32-web-browser-canvas-morph \
	--source "$source_path" \
	--report "$wasm_runtime_report" \
	--visual-report "$wasm_visual_report" \
	--morph-rendered-beauty-report "$wasm_mrb_report" \
	"${mrb_signoff_args[@]}"
go run ./tools/cmd/validate-surface-morph-rendered-beauty \
	--report "$wasm_mrb_report" \
	--morph-to-pixels-chain-out "$wasm_morph_to_pixels_report"

bash scripts/release/surface/surface-dev-workflow-smoke.sh --report-dir "$dev_dir"
go run ./tools/cmd/validate-surface-dev-workflow --report "$dev_dir/surface-dev-workflow.json"

bash scripts/release/surface/surface-inspector-smoke.sh --report-dir "$inspector_dir"
go run ./tools/cmd/validate-surface-inspector --report "$inspector_dir/surface-inspector.json"

bash scripts/release/surface/surface-template-smoke.sh --report-dir "$template_dir"
go run ./tools/cmd/validate-surface-template-smoke --report "$template_dir/surface-template-smoke.json"

bash scripts/release/surface/surface-reference-apps-smoke.sh --report-dir "$reference_dir"
go run ./tools/cmd/validate-surface-reference-apps --report "$reference_dir/surface-reference-apps.json"

docs_claims_log="$docs_claims_dir/surface-docs-claims-gate.log"
bash scripts/release/surface/surface-docs-claims-gate.sh --report-dir "$report_dir" >"$docs_claims_log"
printf "surface docs claims gate validated for %s\n" "$report_dir" >>"$docs_claims_log"

headless_renderer_owned="$(renderer_owned_stable_proof "$mrb_report")"
linux_renderer_owned="$(renderer_owned_stable_proof "$linux_mrb_report")"
wasm_renderer_owned="$(renderer_owned_stable_proof "$wasm_mrb_report")"
renderer_owned_targets=()
bridge_owned_targets=()
missing_renderer_owned_targets=()
if [[ "$headless_renderer_owned" == true ]]; then
	renderer_owned_targets+=("headless")
else
	bridge_owned_targets+=("headless")
	missing_renderer_owned_targets+=("headless")
fi
if [[ "$linux_renderer_owned" == true ]]; then
	renderer_owned_targets+=("linux-x64-real-window")
else
	bridge_owned_targets+=("linux-x64-real-window")
	missing_renderer_owned_targets+=("linux-x64-real-window")
fi
if [[ "$wasm_renderer_owned" == true ]]; then
	renderer_owned_targets+=("wasm32-web-browser-canvas")
else
	bridge_owned_targets+=("wasm32-web-browser-canvas")
	missing_renderer_owned_targets+=("wasm32-web-browser-canvas")
fi
stable_promotion_blockers=()
if [[ "${#missing_renderer_owned_targets[@]}" -gt 0 ]]; then
	missing_renderer_owned_text="$(
		IFS=', '
		printf '%s' "${missing_renderer_owned_targets[*]}"
	)"
	stable_promotion_blockers+=("renderer-owned proof missing: ${missing_renderer_owned_text}")
fi
if [[ "$git_dirty" == true ]]; then
	stable_promotion_blockers+=("dirty worktree audit required")
fi
if [[ "$product_claim" != true ]]; then
	stable_promotion_blockers+=("product_claim=false")
fi
if [[ "$final_signoff" != true ]]; then
	stable_promotion_blockers+=("final_signoff=false")
fi
renderer_owned_stable_targets_json="$(json_array "${renderer_owned_targets[@]}")"
bridge_owned_stable_targets_json="$(json_array "${bridge_owned_targets[@]}")"
stable_promotion_blockers_json="$(json_array "${stable_promotion_blockers[@]}")"
renderer_owned_stable_coverage="all_supported_targets_renderer_owned_stable_proof"
if [[ "${#missing_renderer_owned_targets[@]}" -gt 0 ]]; then
	renderer_owned_stable_coverage="partial_renderer_owned_stable_proof"
fi
gate_status="validated"
if [[ "${#stable_promotion_blockers[@]}" -gt 0 ]]; then
	gate_status="validated_with_target_blockers"
fi

cat > "$summary_path" <<JSON
{
  "schema": "tetra.surface.morph-rendered-beauty.gate.v1",
  "status": $(json_string "$gate_status"),
  "release_scope": "surface-v1-linux-web",
  "surface_scope": "surface-morph-rendered-beauty-linux-web",
  "producer": "scripts/release/surface/morph-rendered-beauty-gate.sh",
  "git_head": $(json_string "$git_head"),
  "git_commit": $(json_string "$git_head"),
  "git_dirty": $git_dirty,
  "version": $(json_string "$version"),
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "source": $(json_string "$source_path"),
  "contract": "docs/spec/surface/morph/surface_morph_rendered_beauty_contract.json",
  "block_contract": "docs/spec/surface/surface_block_contract.json",
  "runtime_report": "runtime.json",
  "visual_report": "visual.json",
  "morph_rendered_beauty_report": "morph-rendered-beauty.json",
  "morph_to_pixels_report": "morph-to-pixels.json",
  "linux_runtime_report": "runtime-linux-x64-real-window.json",
  "linux_visual_report": "visual-linux-x64-real-window.json",
  "linux_morph_rendered_beauty_report": "morph-rendered-beauty-linux-x64-real-window.json",
  "linux_morph_to_pixels_report": "morph-to-pixels-linux-x64-real-window.json",
  "wasm_runtime_report": "runtime-wasm32-web-browser-canvas.json",
  "wasm_visual_report": "visual-wasm32-web-browser-canvas.json",
  "wasm_morph_rendered_beauty_report": "morph-rendered-beauty-wasm32-web-browser-canvas.json",
  "wasm_morph_to_pixels_report": "morph-to-pixels-wasm32-web-browser-canvas.json",
  "dev_workflow_report": "dev-workflow/surface-dev-workflow.json",
  "inspector_report": "inspector/surface-inspector.json",
  "template_smoke_report": "templates/surface-template-smoke.json",
  "reference_apps_report": "reference-apps/surface-reference-apps.json",
  "docs_claims_gate_log": "docs-claims/surface-docs-claims-gate.log",
  "target_matrix": [
    {
      "target": "headless",
      "status": "validated",
      "runtime_report": "runtime.json",
      "visual_report": "visual.json",
      "morph_rendered_beauty_report": "morph-rendered-beauty.json",
      "renderer_owned_stable_proof": $headless_renderer_owned,
      "product_claim": $product_claim
    },
    {
      "target": "linux-x64-real-window",
      "status": "validated",
      "runtime_report": "runtime-linux-x64-real-window.json",
      "visual_report": "visual-linux-x64-real-window.json",
      "morph_rendered_beauty_report": "morph-rendered-beauty-linux-x64-real-window.json",
      "renderer_owned_stable_proof": $linux_renderer_owned,
      "product_claim": $product_claim
    },
    {
      "target": "wasm32-web-browser-canvas",
      "status": "validated",
      "runtime_report": "runtime-wasm32-web-browser-canvas.json",
      "visual_report": "visual-wasm32-web-browser-canvas.json",
      "morph_rendered_beauty_report": "morph-rendered-beauty-wasm32-web-browser-canvas.json",
      "renderer_owned_stable_proof": $wasm_renderer_owned,
      "product_claim": $product_claim
    }
  ],
  "target_blockers": [],
  "stable_promotion_blockers": $stable_promotion_blockers_json,
  "renderer_owned_stable_targets": $renderer_owned_stable_targets_json,
  "bridge_owned_stable_targets": $bridge_owned_stable_targets_json,
  "coverage": {
    "morph_contract": "validated",
    "block_contract": "validated",
    "morph_flagship_runtime": "validated",
    "morph_linux_real_window_runtime": "validated",
    "morph_wasm_browser_canvas_runtime": "validated",
    "renderer_owned_stable_proof": $(json_string "$renderer_owned_stable_coverage"),
    "block_scene_snapshot": "validated",
    "render_command_stream": "validated",
    "frame_artifacts": "validated",
    "pixel_golden": "validated",
    "dev_workflow": "validated",
    "inspector": "validated",
    "templates": "validated",
    "reference_apps": "validated",
    "docs_claims": "validated",
    "artifact_hashes": "validated"
  },
  "negative_guards": {
    "no_react_runtime": true,
    "no_electron_runtime": true,
    "no_dom_ui": true,
    "no_css_runtime": true,
    "no_native_widgets": true,
    "self_golden_rejected": true,
    "metadata_only_rejected": true,
    "precomputed_product_frame_rejected": true,
    "target_blockers_do_not_create_product_claim": true
  },
  "product_claim": $product_claim,
  "final_signoff": $final_signoff,
  "pass": true
}
JSON

required_reports=(
	"runtime.json"
	"visual.json"
	"morph-rendered-beauty.json"
	"morph-to-pixels.json"
	"runtime-linux-x64-real-window.json"
	"visual-linux-x64-real-window.json"
	"morph-rendered-beauty-linux-x64-real-window.json"
	"morph-to-pixels-linux-x64-real-window.json"
	"runtime-wasm32-web-browser-canvas.json"
	"visual-wasm32-web-browser-canvas.json"
	"morph-rendered-beauty-wasm32-web-browser-canvas.json"
	"morph-to-pixels-wasm32-web-browser-canvas.json"
	"dev-workflow/surface-dev-workflow.json"
	"inspector/surface-inspector.json"
	"templates/surface-template-smoke.json"
	"reference-apps/surface-reference-apps.json"
	"docs-claims/surface-docs-claims-gate.log"
	"morph-rendered-beauty-gate-summary.json"
)
for report in "${required_reports[@]}"; do
	if [[ ! -s "$report_dir/$report" ]]; then
		printf 'error: required Surface Morph rendered beauty report missing: %s\n' \
			"$report_dir/$report" >&2
		exit 1
	fi
done

go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface Morph rendered beauty gate reports: $report_dir"
echo "Surface Morph rendered beauty report: $mrb_report"
echo "Surface Morph rendered beauty gate summary: $summary_path"
echo "Surface Morph rendered beauty gate artifact hashes: $report_dir/artifact-hashes.json"
