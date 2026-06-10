#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-prod/P36-electron-comparison-gate"
original_args=("$@")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/electron-comparison-gate.sh [--report-dir DIR]

Builds method-first Surface-vs-Electron comparison evidence for the scoped
Linux/web Surface production claim. This gate publishes comparison method,
sample/variance/environment rows, app-shape parity, public positioning, and
fake-claim rejections. It does not claim official benchmark superiority.
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
  export GOCACHE="$repo_root/.cache/go-build-surface-electron-comparison-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-surface-electron-comparison-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir_arg="${report_dir%/}"
report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_electron_comparison_gate:")"
method_dir="$report_dir/method"
apps_dir="$report_dir/apps"
electron_app_dir="$apps_dir/electron-command-palette"
mkdir -p "$method_dir" "$apps_dir" "$electron_app_dir"

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
command_line="bash scripts/release/surface/electron-comparison-gate.sh"
if [[ -n "$formatted_args" ]]; then
  command_line+=" $formatted_args"
fi

cat > "$method_dir/surface-vs-electron-method.md" <<'MD'
# Surface vs Electron Method

This method compares equivalent app shapes, identical scripted interactions,
same cold/warm state, same OS/target, same measurement tool, and at least five
samples with variance reporting. It is a local scoped comparison method, not an
official benchmark and not proof that arbitrary Electron apps migrate to
Surface.
MD

cp examples/surface_prod_command_palette.tetra "$apps_dir/surface-command-palette.tetra"
cat > "$electron_app_dir/package.json" <<'JSON'
{
  "name": "surface-electron-comparison-command-palette",
  "private": true,
  "description": "Method fixture for equivalent Electron command palette app shape. Not executed as an official benchmark.",
  "main": "main.js",
  "dependencies": {
    "electron": "method-fixture-only"
  }
}
JSON

report_path="$report_dir/surface-electron-comparison-report.json"

go test -buildvcs=false ./tools/validators/surfaceelectron ./tools/cmd/validate-surface-electron-comparison-report ./tools/cmd/surface-electron-comparison -run 'ElectronComparison|Surface.*Benchmark' -count=1
go run -buildvcs=false ./tools/cmd/surface-electron-comparison --out "$report_path"
go run -buildvcs=false ./tools/cmd/validate-surface-electron-comparison-report --report "$report_path"

summary_path="$report_dir/surface-electron-comparison-gate-summary.json"
cat > "$summary_path" <<JSON
{
  "schema": "tetra.surface.electron-comparison-gate.v1",
  "status": "pass",
  "release_scope": "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
  "producer": "scripts/release/surface/electron-comparison-gate.sh",
  "git_head": $(json_string "$git_head"),
  "version": $(json_string "$version"),
  "git_dirty": $git_dirty,
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "schema_under_test": "tetra.surface.electron-comparison-report.v1",
  "level": "surface-electron-comparison-method-v1",
  "comparison_report": "surface-electron-comparison-report.json",
  "method_artifact": "method/surface-vs-electron-method.md",
  "public_positioning": "competitive with Electron in the supported Linux/web scope",
  "same_commit_validated": true,
  "fake_claim_rejections": [
    "official benchmark claim",
    "cherry-picked hardware",
    "missing variance",
    "missing environment",
    "unfair app shape",
    "single-smoke faster-than-Electron claim"
  ]
}
JSON

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface Electron comparison gate reports: $report_dir"
echo "Surface Electron comparison report: $report_path"
echo "Surface Electron comparison artifact hashes: $report_dir/artifact-hashes.json"
