#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-morph/gate"
original_args=("$@")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/morph-gate.sh [--report-dir DIR]

Runs the experimental Tetra Surface Morph Capsule evidence gate.
It requires deterministic headless Morph evidence, Block System dependency
evidence in the same runtime envelope, same-commit validation, and final
artifact hash integrity.
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
gate_contract="scripts/release/surface/contracts/morph-gate.json"
source "$script_dir/report-dir-guard.sh"
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$repo_root/.cache/go-build-surface-morph-gate"
fi
mkdir -p "$GOCACHE"

report_dir_arg="${report_dir%/}"
go run ./tools/cmd/run-gate --contract "$gate_contract" --report-dir "$report_dir_arg" --dry-run >/dev/null
report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_morph_gate:")"
headless_report_dir="$report_dir_arg/headless"

format_command() {
  local formatted=""
  local quoted=""
  local arg
  for arg in "$@"; do
    printf -v quoted "%q" "$arg"
    if [[ -z "$formatted" ]]; then
      formatted="$quoted"
    else
      formatted+=" $quoted"
    fi
  done
  printf "%s" "$formatted"
}

json_string() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '"%s"' "$value"
}

git_head="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
git_dirty=false
if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null || [[ -n "$(git ls-files --others --exclude-standard 2>/dev/null)" ]]; then
  git_dirty=true
fi
version="$(go list -m 2>/dev/null || echo tetra_language)"
host_os="$(go env GOOS 2>/dev/null || uname -s)"
host_arch="$(go env GOARCH 2>/dev/null || uname -m)"
generated_at_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
formatted_args="$(format_command "${original_args[@]}")"
command_line="bash scripts/release/surface/morph-gate.sh"
if [[ -n "$formatted_args" ]]; then
  command_line+=" $formatted_args"
fi

bash scripts/release/surface/surface-headless-morph-smoke.sh --report-dir "$headless_report_dir"
go run ./tools/cmd/validate-surface-morph-report --report "$report_dir/headless/surface-headless-morph.json" --same-commit "$git_head"
go run ./tools/cmd/validate-surface-token-graph --contract docs/spec/surface_token_graph_contract.json --report "$report_dir/headless/surface-headless-morph.json" --root "$repo_root"

required_reports=(
  "headless/surface-headless-morph.json"
  "headless/artifact-hashes.json"
)
for report in "${required_reports[@]}"; do
  if [[ ! -s "$report_dir/$report" ]]; then
    echo "error: required Surface Morph gate report missing or empty: $report_dir/$report" >&2
    exit 1
  fi
done

summary_path="$report_dir/surface-morph-gate-summary.json"
cat > "$summary_path" <<JSON
{
  "schema": "tetra.surface.morph.gate.v1",
  "status": "current",
  "release_scope": "surface-morph-experimental-linux-web",
  "producer": "scripts/release/surface/morph-gate.sh",
  "git_head": $(json_string "$git_head"),
  "version": $(json_string "$version"),
  "git_dirty": $git_dirty,
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "source": "examples/surface_morph_command_palette.tetra",
  "module": "lib.core.morph",
  "schema_under_test": "tetra.surface.morph.v1",
  "token_graph_contract": "docs/spec/surface_token_graph_contract.json",
  "token_graph_validator": "validate-surface-token-graph",
  "recipe_authoring_validator": "validate-surface-morph-report",
  "recipe_expansion_report": "headless/surface-headless-morph.json#morph.recipe_expansions",
  "recipe_count": 19,
  "reference_recipe_apps": [
    "examples/surface_morph_command_palette.tetra",
    "examples/surface_morph_project_dashboard.tetra",
    "examples/surface_morph_settings.tetra",
    "examples/surface_morph_editor_shell.tetra",
    "examples/surface_morph_glass_panel.tetra",
    "examples/surface_morph_studio_shell.tetra"
  ],
  "dependency_gate": "tetra.surface.block-system.gate.v1",
  "same_commit_validated": true,
  "headless_report": "headless/surface-headless-morph.json",
  "target_evidence": [
    "headless"
  ],
  "core_primitives": [
    "Block"
  ],
  "forbidden_core_primitives": [
    "Button",
    "Card",
    "TextField",
    "TextBox",
    "Sidebar",
    "Modal"
  ],
  "artifact_hashes_validated": true
}
JSON

go run ./tools/cmd/validate-surface-morph-gate-summary --summary "$report_dir/surface-morph-gate-summary.json" --same-commit "$git_head"
go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

final_required_reports=(
  "surface-morph-gate-summary.json"
  "headless/surface-headless-morph.json"
  "headless/artifact-hashes.json"
  "artifact-hashes.json"
)
for report in "${final_required_reports[@]}"; do
  if [[ ! -s "$report_dir/$report" ]]; then
    echo "error: required Surface Morph gate report missing or empty: $report_dir/$report" >&2
    exit 1
  fi
done

echo "Surface Morph Capsule gate reports: $report_dir"
echo "Surface Morph Capsule gate summary: $summary_path"
echo "Surface Morph Capsule gate artifact hashes: $report_dir/artifact-hashes.json"
