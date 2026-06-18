#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/surface-linux-x64-real-window-text-focus-input-smoke.sh [--report-dir DIR]

Runs the linux-x64-real-window Tetra Surface TextBox focus/text input smoke.
The gate builds the pure-Tetra Surface TextBox app, presents real-window RGBA
frame evidence, records native-input promotion evidence, and validates focus,
keyboard, text payload, caret, backspace/delete, and resize behavior.
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
report_path="$report_dir/surface-linux-x64-real-window-text-focus-input.json"

go run ./tools/cmd/surface-runtime-smoke \
	--mode linux-x64-real-window-text-focus-input \
	--source examples/surface/runtime/surface_textbox_app.tetra \
	--report "$report_path"
go run ./tools/cmd/validate-surface-runtime --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes \
	--write \
	--root "$report_dir" \
	--out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface linux-x64-real-window text-focus-input runtime smoke report: $report_path"
echo "Surface linux-x64-real-window text-focus-input artifact hashes: $report_dir/artifact-hashes.json"
