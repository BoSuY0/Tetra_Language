#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface-block/headless"

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/surface-headless-block-system-smoke.sh [--report-dir DIR]

Runs deterministic headless Tetra Surface Block-system smoke.
The report validates tetra.surface.block-system.v1 golden/checksum evidence
for examples/surface/block_core/surface_block_system.tetra, including deterministic repeat
checksums and paint/layout/accessibility evidence gates. The same gate also
writes surface-block-examples.json for five polished Block-only example scenes:
examples/surface/block_apps/surface_block_command_palette.tetra,
examples/surface/block_apps/surface_block_project_dashboard.tetra,
examples/surface/block_apps/surface_block_settings.tetra,
examples/surface/block_apps/surface_block_editor_shell.tetra, and
examples/surface/block_apps/surface_block_glass_panel.tetra.
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
report_path="$report_dir/surface-headless-block-system.json"
examples_report_path="$report_dir/surface-block-examples.json"

go run ./tools/cmd/surface-runtime-smoke \
	--mode headless-block-system \
	--source examples/surface/block_core/surface_block_system.tetra \
	--report "$report_path"
go run ./tools/cmd/validate-surface-runtime --report "$report_path"
go run ./tools/cmd/validate-surface-block-examples --report "$examples_report_path" \
	examples/surface/block_apps/surface_block_command_palette.tetra \
	examples/surface/block_apps/surface_block_project_dashboard.tetra \
	examples/surface/block_apps/surface_block_settings.tetra \
	examples/surface/block_apps/surface_block_editor_shell.tetra \
	examples/surface/block_apps/surface_block_glass_panel.tetra
go run ./tools/cmd/validate-artifact-hashes \
	--write \
	--root "$report_dir" \
	--out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface headless Block-system runtime smoke report: $report_path"
echo "Surface Block-only polished examples report: $examples_report_path"
echo "Surface headless Block-system artifact hashes: $report_dir/artifact-hashes.json"
