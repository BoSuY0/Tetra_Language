#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/surface-headless-release-toolkit-smoke.sh [--report-dir DIR]

Runs deterministic headless Tetra Surface production toolkit smoke.
The report validates tetra.surface.toolkit.v1 production-widgets-v1 for
examples/surface/release/surface_release_form.tetra, including style/theme evidence, Text,
Label, StatusText, Button, TextBox, Checkbox, Row, Column, Panel, Stack, Scroll,
Spacer, safe text storage, and changed RGBA frame checksums.
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
report_path="$report_dir/surface-headless-release-toolkit.json"

go run ./tools/cmd/surface-runtime-smoke \
	--mode headless-release-toolkit \
	--source examples/surface/release/surface_release_form.tetra \
	--report "$report_path"
go run ./tools/cmd/validate-surface-runtime --report "$report_path" --release surface-v1
go run ./tools/cmd/validate-artifact-hashes \
	--write \
	--root "$report_dir" \
	--out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface headless release toolkit runtime smoke report: $report_path"
echo "Surface headless release toolkit artifact hashes: $report_dir/artifact-hashes.json"
