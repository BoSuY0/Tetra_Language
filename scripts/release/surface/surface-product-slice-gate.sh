#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-product-slice/product-gate"
product_claim=false
final_signoff=false
original_args=("$@")

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/surface-product-slice-gate.sh [--report-dir DIR] [--product-claim --final-signoff]

Runs the focused Tetra Surface product-slice evidence gate for the Tetra Studio
Shell flagship. The gate validates flagship headless/linux/web runtime reports,
flagship developer-loop evidence, flagship package/update evidence, Surface
claims, generated manifest/docs, artifact hashes, and
the integrated Morph rendered beauty gate before writing
tetra.surface.product-slice-summary.v1.
Product claim/final signoff are explicit promotion-mode flags and require a
clean checkout plus an integrated Morph rendered beauty promotion gate.
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
	export GOCACHE="$repo_root/.cache/go-build-surface-product-slice-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
	export GOTMPDIR="$repo_root/.cache/go-tmp-surface-product-slice-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir_arg="${report_dir%/}"
surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_product_slice_gate:" >/dev/null
report_dir="$report_dir_arg"
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

write_category_summary() {
	local name="$1"
	local path="$2"
	local source_report="$3"
	local evidence="$4"
	mkdir -p "$(dirname "$path")"
	cat >"$path" <<JSON
{
  "schema": "tetra.surface.product-slice-category.v1",
  "release_scope": "surface-v1-linux-web",
  "name": $(json_string "$name"),
  "status": "validated",
  "source_report": $(json_string "$source_report"),
  "evidence": $(json_string "$evidence"),
  "pass": true
}
JSON
}

flagship_source="examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"
flagship_dir="$report_dir/flagship"
dev_dir="$report_dir/dev-workflow"
package_dir="$report_dir/package"
mrb_gate_dir="$report_dir/morph-rendered-beauty"
claims_dir="$report_dir/claims"
docs_dir="$report_dir/docs-manifest"
categories_dir="$report_dir/categories"
mkdir -p "$flagship_dir" "$dev_dir" "$claims_dir" "$docs_dir" "$categories_dir"
mrb_gate_args=()
if [[ "$product_claim" == true ]]; then
	mrb_gate_args+=("--product-claim")
fi
if [[ "$final_signoff" == true ]]; then
	mrb_gate_args+=("--final-signoff")
fi
mrb_category_evidence="integrated Morph rendered beauty gate validated before product-slice summary; product claim remains false until clean checkout audit and final signoff"
if [[ "$product_claim" == true && "$final_signoff" == true ]]; then
	mrb_category_evidence="Morph rendered beauty gate passed with product claim and signoff"
fi

headless_report="$flagship_dir/headless-block-system.json"
linux_report="$flagship_dir/linux-x64-real-window-block-system.json"
wasm_report="$flagship_dir/wasm32-web-browser-canvas-block-system.json"

go run ./tools/cmd/surface-runtime-smoke --mode headless-block-system --source "$flagship_source" --report "$headless_report"
go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-block-system --source "$flagship_source" --report "$linux_report"
go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-browser-canvas-block-system --source "$flagship_source" --report "$wasm_report"

dev_report="$dev_dir/surface-dev-workflow.json"
go run ./cli/cmd/tetra surface dev \
	--source "$flagship_source" \
	--target linux-x64 \
	--out-dir "$dev_dir/dev-artifacts" \
	--report "$dev_report" \
	--change-file "token:lib/core/morph/morph.tetra" \
	--change-file "recipe:lib/core/morph/morph.tetra" \
	--change-file "source:$flagship_source"
go run ./tools/cmd/validate-surface-dev-workflow --report "$dev_report"

bash scripts/release/surface/surface-package-smoke.sh \
	--report-dir "$package_dir" \
	--source "$flagship_source" \
	--app-id studio-shell \
	--app-title "Tetra Studio Shell" \
	--expected-exit-code 0
go run ./tools/cmd/validate-surface-package --report "$package_dir/surface-package.json"

bash scripts/release/surface/morph-rendered-beauty-gate.sh --report-dir "$mrb_gate_dir" "${mrb_gate_args[@]}"

go run ./tools/cmd/validate-surface-claims --root "$repo_root" --report-dir "$report_dir"
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json

cat >"$flagship_dir/flagship-runtime-summary.json" <<JSON
{
  "schema": "tetra.surface.product-slice.flagship-runtime.v1",
  "release_scope": "surface-v1-linux-web",
  "flagship_source": $(json_string "$flagship_source"),
  "reports": {
    "headless": "flagship/headless-block-system.json",
    "linux_x64_real_window": "flagship/linux-x64-real-window-block-system.json",
    "wasm32_web_browser_canvas": "flagship/wasm32-web-browser-canvas-block-system.json"
  },
  "validated": true,
  "pass": true
}
JSON

cat >"$claims_dir/claim-governance-summary.json" <<JSON
{
  "schema": "tetra.surface.product-slice.claim-governance.v1",
  "release_scope": "surface-v1-linux-web",
  "claim_scanner": "validated",
  "comparison_doc": "docs/user/surface/surface_electron_comparison.md",
  "pass": true
}
JSON

cat >"$docs_dir/docs-manifest-summary.json" <<JSON
{
  "schema": "tetra.surface.product-slice.docs-manifest.v1",
  "release_scope": "surface-v1-linux-web",
  "manifest": "validated",
  "docs": "validated",
  "manifest_path": "docs/generated/manifest.json",
  "pass": true
}
JSON

write_category_summary \
	"flagship-runtime" \
	"$categories_dir/flagship-runtime.json" \
	"flagship/flagship-runtime-summary.json" \
	"headless, linux-x64 real-window, and wasm32-web browser-canvas flagship runtime reports validated"
write_category_summary \
	"developer-loop" \
	"$categories_dir/developer-loop.json" \
	"dev-workflow/surface-dev-workflow.json" \
	"tetra.surface.dev-workflow.v1 flagship fast rebuild report validated"
write_category_summary \
	"package-update" \
	"$categories_dir/package-update.json" \
	"package/surface-package.json" \
	"tetra.surface.package.v1 flagship package/update evidence validated"
write_category_summary \
	"morph-rendered-beauty" \
	"$categories_dir/morph-rendered-beauty.json" \
	"morph-rendered-beauty/morph-rendered-beauty-gate-summary.json" \
	"$mrb_category_evidence"
write_category_summary \
	"claim-governance" \
	"$categories_dir/claim-governance.json" \
	"claims/claim-governance-summary.json" \
	"Surface claim scanner passed with product-slice reports"
write_category_summary \
	"docs-manifest" \
	"$categories_dir/docs-manifest.json" \
	"docs-manifest/docs-manifest-summary.json" \
	"generated manifest and docs verification passed"

summary_path="$report_dir/surface-product-slice-summary.json"
generated_at_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
formatted_args="$(format_command "${original_args[@]}")"
command_line="bash scripts/release/surface/surface-product-slice-gate.sh"
if [[ -n "$formatted_args" ]]; then
	command_line+=" $formatted_args"
fi

cat > "$summary_path" <<JSON
{
  "schema": "tetra.surface.product-slice-summary.v1",
  "release_scope": "surface-v1-linux-web",
  "producer": "scripts/release/surface/surface-product-slice-gate.sh",
  "git_head": $(json_string "$git_head"),
  "git_commit": $(json_string "$git_head"),
  "git_dirty": $git_dirty,
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "flagship_source": $(json_string "$flagship_source"),
  "app_id": "studio-shell",
  "artifact_hash_manifest": "artifact-hashes.json",
  "claim_scanner": "validated",
  "manifest": "validated",
  "docs": "validated",
  "morph_rendered_beauty": "validated",
  "product_claim": $product_claim,
  "final_signoff": $final_signoff,
  "categories": [
    {"name": "flagship-runtime", "status": "validated", "source_report": "flagship/flagship-runtime-summary.json", "evidence": "flagship runtime reports validated", "pass": true},
    {"name": "developer-loop", "status": "validated", "source_report": "dev-workflow/surface-dev-workflow.json", "evidence": "flagship developer-loop report validated", "pass": true},
    {"name": "package-update", "status": "validated", "source_report": "package/surface-package.json", "evidence": "flagship package/update report validated", "pass": true},
    {"name": "morph-rendered-beauty", "status": "validated", "source_report": "morph-rendered-beauty/morph-rendered-beauty-gate-summary.json", "evidence": $(json_string "$mrb_category_evidence"), "pass": true},
    {"name": "claim-governance", "status": "validated", "source_report": "claims/claim-governance-summary.json", "evidence": "claim scanner accepted product-slice wording and reports", "pass": true},
    {"name": "docs-manifest", "status": "validated", "source_report": "docs-manifest/docs-manifest-summary.json", "evidence": "manifest and docs validators passed", "pass": true}
  ],
  "required_artifacts": {
    "product_slice_summary": "surface-product-slice-summary.json",
    "artifact_hashes": "artifact-hashes.json",
    "flagship_runtime_summary": "flagship/flagship-runtime-summary.json",
    "flagship_headless": "flagship/headless-block-system.json",
    "flagship_linux": "flagship/linux-x64-real-window-block-system.json",
    "flagship_wasm": "flagship/wasm32-web-browser-canvas-block-system.json",
    "developer_loop": "dev-workflow/surface-dev-workflow.json",
    "package": "package/surface-package.json",
    "morph_rendered_beauty_gate": "morph-rendered-beauty/morph-rendered-beauty-gate-summary.json",
    "claim_governance": "claims/claim-governance-summary.json",
    "docs_manifest": "docs-manifest/docs-manifest-summary.json",
    "category_flagship_runtime": "categories/flagship-runtime.json",
    "category_developer_loop": "categories/developer-loop.json",
    "category_package_update": "categories/package-update.json",
    "category_morph_beauty": "categories/morph-rendered-beauty.json",
    "category_claim_governance": "categories/claim-governance.json",
    "category_docs_manifest": "categories/docs-manifest.json"
  },
  "nonclaims": [
    "no-electron-api-compatibility",
    "no-react-runtime-claim",
    "no-css-runtime-claim",
    "no-dom-authored-application-ui",
    "nonclaim-macos-surface-production-support",
    "nonclaim-windows-surface-production-support",
    "no-gpu-renderer-parity",
    "no-native-widget-parity",
    "no-signing-or-notarization-claim",
    "no-automatic-network-update-claim"
  ],
  "validations": {
    "flagship_runtime": "validated",
    "developer_loop": "validated",
    "package": "validated",
    "morph_rendered_beauty": "validated",
    "claims": "validated",
    "manifest": "validated",
    "docs": "validated",
    "artifact_hashes": "validated"
  },
  "pass": true
}
JSON

go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-surface-product-slice --report-dir "$report_dir"
go run ./tools/cmd/validate-surface-claims --root "$repo_root" --report-dir "$report_dir"

echo "Surface product-slice gate reports: $report_dir"
echo "Surface product-slice summary: $summary_path"
echo "Surface product-slice artifact hashes: $report_dir/artifact-hashes.json"
