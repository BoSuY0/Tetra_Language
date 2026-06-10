#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-prod/P24-dev-loop-gate"
original_args=("$@")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/dev-loop-gate.sh [--report-dir DIR]

Runs the deterministic Surface developer-loop evidence gate:
template scaffold, compiler check, baseline state, real source edit, reload
report, dev-loop validator, package wrapper, and artifact hashes.
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
  export GOCACHE="$repo_root/.cache/go-build-surface-dev-loop-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-surface-dev-loop-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir_arg="${report_dir%/}"
report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_dev_loop_gate:")"
scaffold_dir="$report_dir/scaffold/SurfaceDesk"
state_path="$scaffold_dir/.tetra/surface-dev-state.json"
warmup_report="$scaffold_dir/.tetra/surface-dev-warmup.json"
reload_report="$report_dir/surface-dev-report.json"
package_path="$report_dir/surface-desk.tdx"

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
command_line="bash scripts/release/surface/dev-loop-gate.sh"
if [[ -n "$formatted_args" ]]; then
  command_line+=" $formatted_args"
fi

go run -buildvcs=false ./cli/cmd/tetra new surface-app --template surface-dashboard "$scaffold_dir"
go run -buildvcs=false ./cli/cmd/tetra check "$scaffold_dir"
go run -buildvcs=false ./cli/cmd/tetra surface dev --project "$scaffold_dir" --once --state "$state_path" --report "$warmup_report"
printf '\n// reload-change: dev-loop-gate\n' >> "$scaffold_dir/src/main.t4"
go run -buildvcs=false ./cli/cmd/tetra surface dev --project "$scaffold_dir" --once --state "$state_path" --report "$reload_report"
go run -buildvcs=false ./tools/cmd/validate-surface-dev-report --report "$reload_report"
go run -buildvcs=false ./cli/cmd/tetra surface package "$scaffold_dir" -o "$package_path"

summary_path="$report_dir/surface-dev-loop-gate-summary.json"
cat > "$summary_path" <<JSON
{
  "schema": "tetra.surface.dev-loop.gate.v1",
  "status": "current",
  "release_scope": "surface-dev-loop-experimental",
  "producer": "scripts/release/surface/dev-loop-gate.sh",
  "git_head": $(json_string "$git_head"),
  "version": $(json_string "$version"),
  "git_dirty": $git_dirty,
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "template": "surface-dashboard",
  "schema_under_test": "tetra.surface.dev-loop.v1",
  "level": "surface-fast-dev-loop-v1",
  "scaffold": "scaffold/SurfaceDesk",
  "reload_report": "surface-dev-report.json",
  "package": "surface-desk.tdx",
  "same_commit_validated": true,
  "source_change_trace_required": true,
  "nonclaims": [
    "Electron dev-server parity",
    "React Fast Refresh",
    "CSS HMR",
    "DOM hot reload",
    "browser devtools parity",
    "production signing/notarization"
  ]
}
JSON

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface dev-loop gate reports: $report_dir"
echo "Surface dev-loop report: $reload_report"
echo "Surface dev-loop package: $package_path"
echo "Surface dev-loop artifact hashes: $report_dir/artifact-hashes.json"
