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
token/recipe/source changes. It is a fast rebuild report with no hot reload claim.
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
    -h|--help)
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
fixture_dir="$report_dir/dev-fixture"
artifact_dir="$report_dir/dev-artifacts"
source_path="$fixture_dir/app/main.tetra"
tokens_path="$fixture_dir/design/tokens.tetra"
recipes_path="$fixture_dir/design/recipes.tetra"

mkdir -p "$fixture_dir/app" "$fixture_dir/design" "$artifact_dir"
cat >"$tokens_path" <<'TETRA'
module design.tokens
func accent() -> Int:
    return 17
TETRA
cat >"$recipes_path" <<'TETRA'
module design.recipes
func card() -> Int:
    return 25
TETRA
cat >"$source_path" <<'TETRA'
module app.main
import design.tokens as tokens
import design.recipes as recipes
func main() -> Int:
    return tokens.accent() + recipes.card()
TETRA

go run ./cli/cmd/tetra surface dev --source "$source_path" --target linux-x64 --out-dir "$artifact_dir" --report "$report_path" --change-file "token:$tokens_path" --change-file "recipe:$recipes_path" --change-file "source:$source_path"
go run ./tools/cmd/validate-surface-dev-workflow --report "$report_path"
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface dev workflow smoke report: $report_path"
echo "Surface dev workflow artifact hashes: $report_dir/artifact-hashes.json"
