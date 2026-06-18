#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/surface-headless-app-model-smoke.sh [--report-dir DIR]

Runs deterministic headless Tetra Surface app-model release smoke.
The report validates tetra.surface.app-model.v1 explicit command/reducer
evidence for examples/surface/toolkit/surface_app_model.tetra with navigation, focus scopes,
async cancellation, undo/redo, and no React hooks or DOM event model.
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
report_path="$report_dir/surface-headless-app-model.json"

go run ./tools/cmd/surface-runtime-smoke \
	--mode headless-app-model \
	--source examples/surface/toolkit/surface_app_model.tetra \
	--report "$report_path"
go run ./tools/cmd/validate-surface-runtime --report "$report_path" --release app-model
go run ./tools/cmd/validate-artifact-hashes \
	--write \
	--root "$report_dir" \
	--out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface headless app-model runtime smoke report: $report_path"
echo "Surface headless app-model artifact hashes: $report_dir/artifact-hashes.json"
