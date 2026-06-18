#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/surface"

usage() {
	cat <<'USAGE'
Usage: bash scripts/release/surface/surface-dev-workflow-smoke.sh [--report-dir DIR]

Runs deterministic Tetra Surface developer workflow smoke.
The report validates tetra.surface.dev-workflow.v1
surface-dev-workflow-v1 evidence for a scoped fast rebuild loop across
token/recipe/source changes and attaches Morph-to-pixels evidence from the
rendered beauty report. It is a fast rebuild report with no hot reload claim.
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
report_path="$report_dir/surface-dev-workflow.json"
artifact_dir="$report_dir/dev-artifacts"
mrb_dir="$report_dir/morph-rendered-beauty"
mrb_runtime_report="$mrb_dir/runtime.json"
mrb_visual_report="$mrb_dir/visual.json"
mrb_report="$mrb_dir/morph-rendered-beauty.json"
mrb_golden_dir="$mrb_dir/goldens/headless"
source_path="examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"
tokens_path="lib/core/morph/morph.tetra"
recipes_path="lib/core/morph/morph.tetra"

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

mkdir -p "$artifact_dir" "$mrb_dir" "$mrb_golden_dir"
go run ./tools/cmd/surface-runtime-smoke \
	--mode headless-morph \
	--source "$source_path" \
	--report "$mrb_runtime_report"
mrb_artifact_dir="$mrb_dir/surface-headless-morph-artifacts"
write_drift_golden \
  "$mrb_artifact_dir/surface-morph-rendered-beauty-frame-order-1-initial.rgba" \
  "$mrb_golden_dir/order-1-initial.rgba"
write_drift_golden \
  "$mrb_artifact_dir/surface-morph-rendered-beauty-frame-order-2-focused.rgba" \
  "$mrb_golden_dir/order-2-focused.rgba"
write_drift_golden \
  "$mrb_artifact_dir/surface-morph-rendered-beauty-frame-order-3-motion.rgba" \
  "$mrb_golden_dir/order-3-motion.rgba"
go run ./tools/cmd/surface-visual-diff \
	--runtime-report "$mrb_runtime_report" \
	--required-target headless \
	--golden-artifact "$source_path,headless,1,$mrb_golden_dir/order-1-initial.rgba" \
	--golden-artifact "$source_path,headless,2,$mrb_golden_dir/order-2-focused.rgba" \
	--golden-artifact "$source_path,headless,3,$mrb_golden_dir/order-3-motion.rgba" \
	--out "$mrb_visual_report"
go run ./tools/cmd/surface-runtime-smoke \
	--mode headless-morph \
	--source "$source_path" \
	--report "$mrb_runtime_report" \
	--visual-report "$mrb_visual_report" \
	--morph-rendered-beauty-report "$mrb_report"
go run ./tools/cmd/validate-surface-morph-rendered-beauty --report "$mrb_report"

go run ./cli/cmd/tetra surface dev \
	--source "$source_path" \
	--target linux-x64 \
	--out-dir "$artifact_dir" \
	--report "$report_path" \
	--morph-rendered-beauty-report "$mrb_report" \
	--change-file "token:$tokens_path" \
	--change-file "recipe:$recipes_path" \
	--change-file "source:$source_path"
go run ./tools/cmd/validate-surface-dev-workflow --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes \
	--write \
	--root "$report_dir" \
	--out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface dev workflow smoke report: $report_path"
echo "Surface dev workflow Morph-to-pixels report: $mrb_report"
echo "Surface dev workflow artifact hashes: $report_dir/artifact-hashes.json"
