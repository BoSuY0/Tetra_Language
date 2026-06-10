#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-prod/P25-visual-gate"
original_args=("$@")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/visual-gate.sh [--report-dir DIR]

Runs the deterministic Surface visual regression evidence gate:
canonical source scenes, software RGBA golden PNGs, current frame PNGs,
diff PNGs, frame/source/font/asset checksums, strict visual report validation,
fake-golden rejection coverage, and artifact hash integrity.
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
  export GOCACHE="$repo_root/.cache/go-build-surface-visual-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-surface-visual-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir_arg="${report_dir%/}"
report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_visual_gate:")"
report_path="$report_dir/surface-visual-report.json"

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
command_line="bash scripts/release/surface/visual-gate.sh"
if [[ -n "$formatted_args" ]]; then
  command_line+=" $formatted_args"
fi

go run -buildvcs=false ./tools/cmd/surface-golden --out "$report_dir"
go run -buildvcs=false ./tools/cmd/validate-surface-visual-report --report "$report_path"

summary_path="$report_dir/surface-visual-gate-summary.json"
cat > "$summary_path" <<JSON
{
  "schema": "tetra.surface.visual-gate.v1",
  "status": "current",
  "release_scope": "surface-visual-regression-experimental",
  "producer": "scripts/release/surface/visual-gate.sh",
  "git_head": $(json_string "$git_head"),
  "version": $(json_string "$version"),
  "git_dirty": $git_dirty,
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "schema_under_test": "tetra.surface.visual-regression.v1",
  "level": "surface-visual-golden-v1",
  "visual_report": "surface-visual-report.json",
  "required_scenes": [
    "command-palette",
    "dashboard",
    "settings",
    "editor",
    "glass"
  ],
  "same_commit_validated": true,
  "fake_golden_rejection": [
    "screenshot without scene hash",
    "changed golden without review marker",
    "missing baseline artifact",
    "tampered artifact checksum"
  ],
  "nonclaims": [
    "Electron or Chromium pixel parity",
    "CSS browser rendering parity",
    "GPU compositor parity",
    "production Block support"
  ]
}
JSON

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface visual gate reports: $report_dir"
echo "Surface visual report: $report_path"
echo "Surface visual artifact hashes: $report_dir/artifact-hashes.json"
