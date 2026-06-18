#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/ui-toolkit-core"
artifact_id="tetra.release.post_v0_4.ui_toolkit_core.production-gate.v1"

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/post_v0_4/ui-toolkit-core-production-gate.sh [--report-dir DIR]

Runs the ordered post-v0.4 production evidence gate for tetra.ui.toolkit.v1:
  1. baseline compiler/CLI/tools tests
  2. manifest/docs/target validation
  3. UI Toolkit Core runtime smoke and validator
  4. artifact hash write and verification
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

prepare_report_dir() {
  if [[ -z "$report_dir" || "$report_dir" == "/" || "$report_dir" == "." || "$report_dir" == ".." ]]; then
    echo "error: refusing to clear unsafe report directory: ${report_dir:-<empty>}" >&2
    exit 2
  fi
  if [[ -L "$report_dir" ]]; then
    echo "error: refusing to clear symlink report directory: $report_dir" >&2
    exit 2
  fi
  if [[ -e "$report_dir" && ! -d "$report_dir" ]]; then
    echo "error: refusing to use non-directory report path: $report_dir" >&2
    exit 2
  fi
  mkdir -p "$report_dir"
  local find_report_dir="$report_dir"
  if [[ "$find_report_dir" == -* ]]; then
    find_report_dir="./$find_report_dir"
  fi
  find "$find_report_dir" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +
}

prepare_report_dir

go test ./compiler/... ./cli/... ./tools/... -count=1
go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-targets

./tetra features --format=json > "$report_dir/features.json"
./tetra targets --format=json > "$report_dir/targets.json"

go run ./tools/cmd/ui-toolkit-core-smoke --report "$report_dir/ui-toolkit-core.json"
go run ./tools/cmd/validate-ui-toolkit-core --report "$report_dir/ui-toolkit-core.json"

summary_path="$report_dir/ui-toolkit-core-production-gate.json"
generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
cat > "$summary_path" << JSON
{
  "schema": "tetra.release.post_v0_4.ui_toolkit_core.production-gate.summary.v1",
  "artifact": "$artifact_id",
  "generated_at": "$generated_at",
  "status": "pass",
  "report_dir": "$report_dir",
  "evidence": [
    "features.json",
    "targets.json",
    "ui-toolkit-core.bundle.json",
    "ui-toolkit-core.trace.json",
    "ui-toolkit-core.json"
  ]
}
JSON

go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --root "$report_dir" --out "$report_dir/artifact-hashes.json"

echo "post-v0.4 UI Toolkit Core production gate report dir: $report_dir"
echo "required artifact: $artifact_id"
echo "summary: $summary_path"
echo "artifact hashes: $report_dir/artifact-hashes.json"
