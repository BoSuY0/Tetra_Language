#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/surface-wasm32-web-browser-canvas-component-tree-smoke.sh [--report-dir DIR]

Runs the wasm32-web browser-canvas/input Tetra Surface component-tree smoke.
The gate builds the pure-Tetra tree app, runs it in a real Chromium-compatible
browser canvas, reads back RGBA pixels, records pointer/key/resize/beforeinput
tree dispatch evidence, and rejects Node-only, DOM UI, user JS, or legacy
sidecar promotion evidence.
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
mkdir -p "$report_dir"
report_dir="$(cd "$report_dir" && pwd)"
report_path="$report_dir/surface-wasm32-web-browser-canvas-component-tree.json"
wasm_path="$report_dir/surface-wasm32-web-browser-canvas-component-tree-artifacts/surface-tree-app.wasm"

go run ./tools/cmd/surface-runtime-smoke \
	--mode wasm32-web-browser-canvas-component-tree \
	--source examples/surface/toolkit/surface_tree_app.tetra \
	--report "$report_path"
go run ./tools/cmd/validate-wasm-imports --target wasm32-web "$wasm_path"
go run ./tools/cmd/validate-surface-runtime --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes \
	--write \
	--root "$report_dir" \
	--out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface wasm32-web browser-canvas component-tree runtime smoke report: $report_path"
echo "Surface wasm32-web browser-canvas component-tree artifact hashes: $report_dir/artifact-hashes.json"
