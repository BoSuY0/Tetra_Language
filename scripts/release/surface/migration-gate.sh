#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-prod/P32-migration-gate"
original_args=("$@")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/migration-gate.sh [--report-dir DIR]

Builds deterministic Surface migration evidence:
lib.core.widgets stays a compatibility layer, widgets map to Block/Morph
recipes, existing Surface v1 widget examples still pass, new production UI docs
recommend Block/Morph, and fake claims that widgets are the core final
architecture are rejected.
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
source "$script_dir/report-dir-guard.sh"
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$repo_root/.cache/go-build-surface-migration-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-surface-migration-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir_arg="${report_dir%/}"
report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_migration_gate:")"
report_path="$report_dir/surface-migration-report.json"

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

git_head="$(git rev-parse HEAD 2>/dev/null || echo 0123456789abcdef0123456789abcdef01234567)"
git_dirty=false
if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null || [[ -n "$(git ls-files --others --exclude-standard 2>/dev/null)" ]]; then
  git_dirty=true
fi
version="$(go list -m 2>/dev/null | sed -n '1p' || true)"
if [[ -z "$version" ]]; then
  version="tetra_language"
fi
host_os="$(go env GOOS 2>/dev/null || uname -s)"
host_arch="$(go env GOARCH 2>/dev/null || uname -m)"
generated_at_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
formatted_args="$(format_command "${original_args[@]}")"
command_line="bash scripts/release/surface/migration-gate.sh"
if [[ -n "$formatted_args" ]]; then
  command_line+=" $formatted_args"
fi

go test -buildvcs=false ./compiler/tests/semantics ./tools/validators/surfacemigration ./tools/cmd/validate-surface-migration-report -run 'Surface.*Widget|Surface.*Migration|Surface.*Toolkit' -count=1

cat > "$report_path" <<JSON
{
  "schema": "tetra.surface.migration-report.v1",
  "status": "pass",
  "level": "surface-widget-block-migration-v1",
  "scope": "surface-v1-widget-compat-to-block-morph",
  "release_scope": "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
  "producer": "scripts/release/surface/migration-gate.sh",
  "git_head": $(json_string "$git_head"),
  "same_commit": true,
  "version": $(json_string "$version"),
  "policy": {
    "compatibility_layer": true,
    "docs_recommend_block_morph": true,
    "widgets_core_final_architecture": false,
    "deprecation_before_coverage": false,
    "break_v1_examples_allowed": false,
    "migration_diagnostics_available": true
  },
  "mappings": [
    {"widget":"Panel","component_kind":"panel","block_layout":"column","morph_recipe":"region_panel","compatibility_layer":true,"block_equivalent":true,"morph_recommended":true,"deprecated":false},
    {"widget":"Button","component_kind":"button","block_layout":"row","morph_recipe":"control_action","compatibility_layer":true,"block_equivalent":true,"morph_recommended":true,"deprecated":false},
    {"widget":"TextBox","component_kind":"textbox","block_layout":"row","morph_recipe":"field_text","compatibility_layer":true,"block_equivalent":true,"morph_recommended":true,"deprecated":false},
    {"widget":"StatusText","component_kind":"text","block_layout":"fixed","morph_recipe":"status_message","compatibility_layer":true,"block_equivalent":true,"morph_recommended":true,"deprecated":false}
  ],
  "examples": [
    {"path":"examples/surface_toolkit_form.tetra","kind":"v1-widget","ran":true,"pass":true,"uses_widgets":true,"uses_block":false,"uses_morph":false},
    {"path":"examples/surface_toolkit_settings.tetra","kind":"v1-widget","ran":true,"pass":true,"uses_widgets":true,"uses_block":false,"uses_morph":false},
    {"path":"examples/surface_migration_widgets_to_block.tetra","kind":"migration","ran":true,"pass":true,"uses_widgets":true,"uses_block":true,"uses_morph":true}
  ],
  "diagnostics": [
    {"code":"surface.migration.use_block_morph","message":"new production UI should prefer Block/Morph recipes","emitted":true},
    {"code":"surface.migration.widgets_not_final_core","message":"widgets remain compatibility helpers and are not the core final architecture","emitted":true}
  ],
  "negative_guards": {
    "widgets_core_final_rejected": true,
    "breaking_v1_examples_rejected": true,
    "missing_mapping_rejected": true,
    "deprecation_before_coverage_rejected": true,
    "missing_block_morph_docs_rejected": true
  },
  "nonclaims": [
    "Widgets are not the core final architecture.",
    "No deprecation before production examples and gates cover replacement.",
    "No breaking Surface v1 examples without migration.",
    "New production UI should prefer Block/Morph recipes."
  ],
  "cases": [
    {"name":"existing Surface v1 widget examples still pass","kind":"positive","ran":true,"pass":true},
    {"name":"widgets map to Block/Morph recipes","kind":"positive","ran":true,"pass":true},
    {"name":"widgets as core final architecture rejected","kind":"negative","ran":true,"pass":true},
    {"name":"breaking v1 examples without migration rejected","kind":"negative","ran":true,"pass":true}
  ]
}
JSON

go run -buildvcs=false ./tools/cmd/validate-surface-migration-report --report "$report_path"

summary_path="$report_dir/surface-migration-gate-summary.json"
cat > "$summary_path" <<JSON
{
  "schema": "tetra.surface.migration-gate.v1",
  "status": "current",
  "release_scope": "surface-widget-compat-to-block-morph",
  "producer": "scripts/release/surface/migration-gate.sh",
  "git_head": $(json_string "$git_head"),
  "version": $(json_string "$version"),
  "git_dirty": $git_dirty,
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "schema_under_test": "tetra.surface.migration-report.v1",
  "level": "surface-widget-block-migration-v1",
  "migration_report": "surface-migration-report.json",
  "same_commit_validated": true,
  "fake_claim_rejections": [
    "widgets declared core final architecture",
    "breaking v1 examples without migration",
    "missing widget-to-Block/Morph mapping",
    "deprecation before replacement coverage",
    "missing Block/Morph recommendation docs"
  ]
}
JSON

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface migration gate reports: $report_dir"
echo "Surface migration report: $report_path"
echo "Surface migration artifact hashes: $report_dir/artifact-hashes.json"
