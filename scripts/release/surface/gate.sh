#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/gate.sh [--report-dir DIR]

Runs the complete experimental Tetra Surface release gate.
It emits and validates tetra.surface.runtime.v1 reports for headless,
Linux-x64 starter, linux-x64-real-window, wasm32-web starter, and wasm32-web
browser-canvas/input Surface evidence plus TextBox focus/text input evidence,
component-tree evidence, component-tree API hardening evidence, the
minimal reusable widget toolkit, toolkit-reuse-v1 evidence, and
accessibility metadata tree evidence for headless, linux real-window, and
browser-canvas targets. The gate requires
pure-Tetra component runtime evidence, no legacy UI sidecar artifacts,
real-window/browser-canvas promotion evidence, and final artifact hash
integrity for the report directory.
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

surface_smokes=(
	scripts/release/surface/surface-headless-smoke.sh
	scripts/release/surface/surface-linux-x64-smoke.sh
	scripts/release/surface/surface-linux-x64-real-window-smoke.sh
	scripts/release/surface/surface-wasm32-web-smoke.sh
	scripts/release/surface/surface-wasm32-web-browser-canvas-smoke.sh
	scripts/release/surface/surface-headless-text-focus-input-smoke.sh
	scripts/release/surface/surface-linux-x64-real-window-text-focus-input-smoke.sh
	scripts/release/surface/surface-wasm32-web-browser-canvas-text-focus-input-smoke.sh
	scripts/release/surface/surface-headless-component-tree-smoke.sh
	scripts/release/surface/surface-linux-x64-real-window-component-tree-smoke.sh
	scripts/release/surface/surface-wasm32-web-browser-canvas-component-tree-smoke.sh
	scripts/release/surface/surface-headless-component-tree-api-smoke.sh
	scripts/release/surface/surface-linux-x64-real-window-component-tree-api-smoke.sh
	scripts/release/surface/surface-wasm32-web-browser-canvas-component-tree-api-smoke.sh
	scripts/release/surface/surface-headless-minimal-toolkit-smoke.sh
	scripts/release/surface/surface-linux-x64-real-window-minimal-toolkit-smoke.sh
	scripts/release/surface/surface-wasm32-web-browser-canvas-minimal-toolkit-smoke.sh
	scripts/release/surface/surface-headless-toolkit-reuse-smoke.sh
	scripts/release/surface/surface-linux-x64-real-window-toolkit-reuse-smoke.sh
	scripts/release/surface/surface-wasm32-web-browser-canvas-toolkit-reuse-smoke.sh
	scripts/release/surface/surface-headless-accessibility-metadata-smoke.sh
	scripts/release/surface/surface-linux-x64-real-window-accessibility-metadata-smoke.sh
	scripts/release/surface/surface-wasm32-web-browser-canvas-accessibility-metadata-smoke.sh
)

for smoke_script in "${surface_smokes[@]}"; do
	bash "$smoke_script" --report-dir "$report_dir"
done

surface_reports=(
	surface-headless.json
	surface-linux-x64.json
	surface-linux-x64-real-window.json
	surface-wasm32-web.json
	surface-wasm32-web-browser-canvas.json
	surface-headless-text-focus-input.json
	surface-linux-x64-real-window-text-focus-input.json
	surface-wasm32-web-browser-canvas-text-focus-input.json
	surface-headless-component-tree.json
	surface-linux-x64-real-window-component-tree.json
	surface-wasm32-web-browser-canvas-component-tree.json
	surface-headless-component-tree-api.json
	surface-linux-x64-real-window-component-tree-api.json
	surface-wasm32-web-browser-canvas-component-tree-api.json
	surface-headless-minimal-toolkit.json
	surface-linux-x64-real-window-minimal-toolkit.json
	surface-wasm32-web-browser-canvas-minimal-toolkit.json
	surface-headless-toolkit-reuse.json
	surface-linux-x64-real-window-toolkit-reuse.json
	surface-wasm32-web-browser-canvas-toolkit-reuse.json
	surface-headless-accessibility-metadata.json
	surface-linux-x64-real-window-accessibility-metadata.json
	surface-wasm32-web-browser-canvas-accessibility-metadata.json
)

for report_name in "${surface_reports[@]}"; do
	go run ./tools/cmd/validate-surface-runtime \
		--report "$report_dir/$report_name"
done

go run ./tools/cmd/validate-artifact-hashes \
	--write \
	--root "$report_dir" \
	--out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface release gate reports: $report_dir"
echo "Surface release gate artifact hashes: $report_dir/artifact-hashes.json"
