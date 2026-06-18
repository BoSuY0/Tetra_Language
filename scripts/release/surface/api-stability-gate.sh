#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface-api-stability-v1"

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/api-stability-gate.sh [--report-dir DIR]

Validates the Surface v1 stable API documentation boundary. The gate checks
stable lib.core Surface module names, rejects experimental/versioned stable
module suffixes, rejects lib.experimental imports in Surface release examples,
generates same-branch API docs, validates them, and reruns docs/manifest gates.
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
if [[ -z "${GOCACHE:-}" ]]; then
	export GOCACHE="$repo_root/.cache/go-build-surface-release"
fi
mkdir -p "$GOCACHE"
mkdir -p "$report_dir/artifacts"
report_dir="$(cd "$report_dir" && pwd)"
artifacts_dir="$report_dir/artifacts"

stable_modules=(
	"lib.core.surface"
	"lib.core.draw"
	"lib.core.component"
	"lib.core.widgets"
	"lib.core.accessibility"
	"lib.core.text"
	"lib.core.style"
)

module_report="$report_dir/stable-surface-modules.txt"
stable_module_pattern='^module lib\.core\.'
stable_module_pattern+='(surface|draw|component|widgets|accessibility|text|style)$'
rg -n "$stable_module_pattern" lib/core >"$module_report"
module_count="$(wc -l <"$module_report" | tr -d ' ')"
if [[ "$module_count" != "${#stable_modules[@]}" ]]; then
	echo "error: expected ${#stable_modules[@]} stable Surface modules, got $module_count" >&2
	cat "$module_report" >&2
	exit 1
fi

if rg -n '^module lib\.core\..*(experimental|v[0-9]+|_[vV][0-9]+)' lib/core; then
	echo "error: stable lib.core Surface modules must not use experimental/version suffixes" >&2
	exit 1
fi

if rg -n '^import lib\.experimental(\.|$)' examples/surface_release_*.tetra; then
	echo "error: Surface release examples must not import lib.experimental.*" >&2
	exit 1
fi

docs_path="$artifacts_dir/tetra-docs.md"
./tetra doc lib/core examples >"$docs_path"
go run ./tools/cmd/validate-api-docs --docs "$docs_path"

for module in "${stable_modules[@]}"; do
	if ! rg -n "^## ${module}$" "$docs_path" >/dev/null; then
		echo "error: generated API docs missing stable module ${module}" >&2
		exit 1
	fi
done

public_api_summary="$report_dir/public-surface-api-summary.txt"
{
	echo "schema: tetra.surface.public-api-summary.v1"
	echo "release_scope: surface-v1-linux-web"
	echo "stable_modules:"
	for module in "${stable_modules[@]}"; do
		echo "- $module"
	done
	echo "supported_targets:"
	echo "- headless release evidence target"
	echo "- linux-x64 real-window"
	echo "- wasm32-web browser-canvas"
	echo "unsupported_targets:"
	echo "- macos-x64"
	echo "- windows-x64"
	echo "- wasm32-wasi"
	echo "nonclaims:"
	echo "- GPU rendering"
	echo "- platform-native widgets"
	echo "- DOM/React/user-JS application UI"
	echo "- dynamic trait-object widgets or witness-table component dispatch"
	echo "- full rich text editor"
	echo "- full AT-SPI/screen-reader support"
} >"$public_api_summary"

go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json

cat >"$report_dir/surface-api-stability-summary.json" <<JSON
{
  "schema": "tetra.surface.api-stability.v1",
  "status": "pass",
  "release_scope": "surface-v1-linux-web",
  "stable_modules": [
    "lib.core.surface",
    "lib.core.draw",
    "lib.core.component",
    "lib.core.widgets",
    "lib.core.accessibility",
    "lib.core.text",
    "lib.core.style"
  ],
  "api_docs": "artifacts/tetra-docs.md",
  "public_api_summary": "public-surface-api-summary.txt",
  "module_report": "stable-surface-modules.txt",
  "release_examples_import_experimental": false,
  "docs_manifest_validated": true
}
JSON

echo "Surface API stability report: $report_dir/surface-api-stability-summary.json"
echo "Surface API docs: $docs_path"
