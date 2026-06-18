#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-product-v1"
original_args=("$@")

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/product-gate.sh [--report-dir DIR]

Runs the scoped Tetra Surface product evidence gate for surface-v1-linux-web.
The gate executes the Surface v1 release gate in the requested report
directory, validates artifact hashes, validates Surface claims, validates the
generated manifest and docs, then writes a product-gate summary. P29 owns the
final PROD_STABLE_SCOPED verdict.
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
if [[ -z "${GOTELEMETRY:-}" ]]; then
	export GOTELEMETRY=off
fi
if [[ -z "${GOCACHE:-}" ]]; then
	export GOCACHE="$repo_root/.cache/go-build-surface-product-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
	export GOTMPDIR="$repo_root/.cache/go-tmp-surface-product-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir_arg="${report_dir%/}"
if [[ -z "$report_dir_arg" ]]; then
	report_dir_arg="$report_dir"
fi
surface_release_require_fresh_report_dir \
  "$report_dir_arg" \
  "$repo_root" \
  "surface_product_gate:" \
  >/dev/null
report_dir="$report_dir_arg"

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

bash scripts/release/surface/release-gate.sh --report-dir "$report_dir_arg"

go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-surface-claims --root "$repo_root" --report-dir "$report_dir"
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json

summary_path="$report_dir/surface-product-gate-summary.json"
product_summary_path="$report_dir/product-summary.json"
git_head="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
git_dirty=false
if ! git diff --quiet 2>/dev/null || \
  ! git diff --cached --quiet 2>/dev/null || \
  [[ -n "$(git ls-files --others --exclude-standard 2>/dev/null)" ]]; then
	git_dirty=true
fi
final_verdict="P29_FINAL_AUDIT_REQUIRED"
product_summary_status="product_gate_passed_p29_final_audit_required"
if [[ "$git_dirty" == true ]]; then
	final_verdict="BLOCKED_DIRTY_CHECKOUT"
	product_summary_status="product_gate_passed_clean_same_commit_blocked"
fi
generated_at_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
formatted_args="$(format_command "${original_args[@]}")"
command_line="bash scripts/release/surface/product-gate.sh"
if [[ -n "$formatted_args" ]]; then
	command_line+=" $formatted_args"
fi

write_product_category_summary() {
	local category="$1"
	local path="$2"
	local source_report="$3"
	local status="$4"
	local evidence="$5"
	mkdir -p "$(dirname "$path")"
	cat >"$path" <<JSON
{
  "schema": "tetra.surface.product-category-summary.v1",
  "release_scope": "surface-v1-linux-web",
  "category": $(json_string "$category"),
  "status": $(json_string "$status"),
  "source_report": $(json_string "$source_report"),
  "evidence": $(json_string "$evidence"),
  "git_head": $(json_string "$git_head"),
  "git_dirty": $git_dirty,
  "final_verdict_owner": "SURFACE-BEAUTY-P29",
  "final_verdict": $(json_string "$final_verdict"),
  "production_claim": false,
  "final_signoff": false
}
JSON
}

cat >"$summary_path" <<JSON
{
  "schema": "tetra.surface.product-gate.v1",
  "release_scope": "surface-v1-linux-web",
  "status": "p27_product_gate_passed",
  "producer": "scripts/release/surface/product-gate.sh",
  "git_head": $(json_string "$git_head"),
  "git_dirty": $git_dirty,
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "release_gate_report": "surface-release-summary.json",
  "artifact_hash_manifest": "artifact-hashes.json",
  "release_state": "validated",
  "artifact_hashes": "validated",
  "claim_scanner": "validated",
  "manifest": "validated",
  "docs": "validated",
  "ci_required_gate": true,
  "continue_on_error_bypass_allowed": false,
  "final_verdict_owner": "SURFACE-BEAUTY-P29",
  "nonclaims": [
    "not-final-PROD_STABLE_SCOPED-verdict",
    "not-P28-doc-governance-completion",
    "not-P29-final-audit"
  ]
}
JSON

cat >"$product_summary_path" <<JSON
{
  "schema": "tetra.surface.product-summary.v1",
  "release_scope": "surface-v1-linux-web",
  "status": $(json_string "$product_summary_status"),
  "producer": "scripts/release/surface/product-gate.sh",
  "git_head": $(json_string "$git_head"),
  "git_dirty": $git_dirty,
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "product_gate_summary": "surface-product-gate-summary.json",
  "release_gate_report": "surface-release-summary.json",
  "artifact_hash_manifest": "artifact-hashes.json",
  "release_state": "validated",
  "artifact_hashes": "validated",
  "claim_scanner": "validated",
  "manifest": "validated",
  "docs": "validated",
  "ci_required_gate": true,
  "continue_on_error_bypass_allowed": false,
  "final_verdict_owner": "SURFACE-BEAUTY-P29",
  "final_verdict": $(json_string "$final_verdict"),
  "production_claim": false,
  "final_signoff": false,
  "canonical_final_readiness_report": true,
  "final_readiness_source": "product-summary.json",
  "inner_release_summary_role": "prerequisite_evidence_not_final_signoff",
  "release_gate_report_final_signoff": false,
  "clean_same_commit_required": true,
  "clean_same_commit_proven": false,
  "target_matrix": [
    {
      "target": "headless",
      "status": "release-test-evidence",
      "tier": "evidence-target",
      "production_claim": false,
      "report": "surface-headless-release.json"
    },
    {
      "target": "linux-x64",
      "status": "current",
      "tier": "bounded-linux-web-scope",
      "production_claim": true,
      "report": "surface-linux-x64-release-app-shell.json"
    },
    {
      "target": "wasm32-web",
      "status": "current",
      "tier": "bounded-linux-web-scope",
      "production_claim": true,
      "report": "surface-wasm32-web-release-browser.json"
    },
    {
      "target": "macos-x64",
      "status": "unsupported",
      "tier": "UNSUPPORTED",
      "production_claim": false,
      "report": "surface-macos-x64-target-host-status.json"
    },
    {
      "target": "windows-x64",
      "status": "unsupported",
      "tier": "UNSUPPORTED",
      "production_claim": false,
      "report": "surface-windows-x64-target-host-status.json"
    },
    {
      "target": "wasm32-wasi",
      "status": "unsupported",
      "tier": "UNSUPPORTED",
      "production_claim": false,
      "report": "surface-release-summary.json"
    }
  ],
  "required_artifacts": {
    "product_summary": "product-summary.json",
    "artifact_hashes": "artifact-hashes.json",
    "visual": "visual/visual-summary.json",
    "accessibility": "accessibility/accessibility-summary.json",
    "performance": "performance/performance-budget.json",
    "app_shell": "app-shell/app-shell-summary.json",
    "package": "package/package-summary.json",
    "reference_apps": "reference-apps/reference-apps-summary.json",
    "claim_governance": "claim-governance/claims-summary.json"
  },
  "nonclaims": [
    "all-platform-surface-parity",
    "nonclaim-macos-surface-production-support",
    "nonclaim-windows-surface-production-support",
    "macos-surface-production-nonclaim",
    "windows-surface-production-nonclaim",
    "wasm32-wasi-surface-ui-runtime",
    "gpu-renderer",
    "full-rich-text-editor",
    "full-screen-reader-support",
    "official-benchmark-superiority",
    "electron-api-compatibility",
    "react-api-compatibility",
    "css-cascade-compatibility",
    "dom-authored-application-ui",
    "user-javascript-application-logic"
  ]
}
JSON

write_product_category_summary \
	"visual" \
	"$report_dir/visual/visual-summary.json" \
	"reference-visual/surface-visual-regression.json" \
	"validated-evidence-summary" \
	"visual gate evidence is present through release-gate reference visual reports"
	write_product_category_summary \
		"accessibility" \
		"$report_dir/accessibility/accessibility-summary.json" \
		"surface-linux-x64-release-accessibility.json" \
		"validated-evidence-summary" \
		"supported accessibility reports are present; screen-reader support is out of scope"
write_product_category_summary \
	"performance" \
	"$report_dir/performance/performance-budget.json" \
	"surface-linux-x64-release-app-shell.json" \
	"validated-evidence-summary" \
	"surface-performance-budget-v1 is validated as local deterministic budget evidence"
write_product_category_summary \
	"app-shell" \
	"$report_dir/app-shell/app-shell-summary.json" \
	"surface-linux-x64-release-app-shell.json" \
	"validated-evidence-summary" \
	"linux-app-shell-subset-v1 evidence is present for the bounded Linux scope"
write_product_category_summary \
	"package" \
	"$report_dir/package/package-summary.json" \
	"surface-package.json" \
	"validated-evidence-summary" \
	"surface-package-v1 evidence is present for the bounded Linux/web scope"
write_product_category_summary \
	"reference-apps" \
	"$report_dir/reference-apps/reference-apps-summary.json" \
	"surface-reference-apps.json" \
	"validated-evidence-summary" \
	"surface-reference-app-suite-v1 evidence is present for ten reference app shapes"
write_product_category_summary \
	"claim-governance" \
	"$report_dir/claim-governance/claims-summary.json" \
	"surface-product-gate-summary.json" \
	"validated-evidence-summary" \
	"claim scanner, manifest validator, and docs verifier passed for this product gate run"

go run ./tools/cmd/validate-artifact-hashes \
	--write \
	--root "$report_dir" \
	--out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-surface-product-summary --report-dir "$report_dir"
go run ./tools/cmd/validate-surface-claims --root "$repo_root" --report-dir "$report_dir"

echo "Surface product gate reports: $report_dir"
echo "Surface product gate summary: $summary_path"
echo "Surface product summary: $product_summary_path"
echo "Surface product gate artifact hashes: $report_dir/artifact-hashes.json"
