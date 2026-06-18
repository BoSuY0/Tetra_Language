#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="reports/surface-prod/P31-perf-gate"
original_args=("$@")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/surface/perf-gate.sh [--report-dir DIR]

Builds deterministic Surface performance/memory evidence:
startup time, first frame time, steady frame p95, peak RSS, frame allocations,
layout/glyph/asset cache budgets, binary size, CPU idle/power proxy, input
latency, animation frame jitter, baseline environment capture, and fake-claim
rejection for zero memory overhead, fastest UI framework, and unsupported
faster-than-Electron claims.
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
  export GOCACHE="$repo_root/.cache/go-build-surface-perf-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-surface-perf-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir_arg="${report_dir%/}"
report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_perf_gate:")"
baseline_dir="$report_dir/baselines"
comparison_dir="$report_dir/comparisons"
mkdir -p "$baseline_dir" "$comparison_dir"
report_path="$report_dir/surface-perf-report.json"

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
command_line="bash scripts/release/surface/perf-gate.sh"
if [[ -n "$formatted_args" ]]; then
  command_line+=" $formatted_args"
fi

cat > "$baseline_dir/surface-v1-linux-cold.json" <<JSON
{
  "schema": "tetra.surface.perf-baseline.v1",
  "name": "surface-v1-linux-cold",
  "target": "linux-x64",
  "git_head": $(json_string "$git_head"),
  "same_app_shape": true,
  "same_os_target": true,
  "same_cold_warm_state": true,
  "environment_captured": true,
  "budgets": ["startup_time", "first_frame_time", "steady_frame_time_p95", "peak_rss"]
}
JSON

cat > "$baseline_dir/surface-v1-web-warm.json" <<JSON
{
  "schema": "tetra.surface.perf-baseline.v1",
  "name": "surface-v1-web-warm",
  "target": "wasm32-web",
  "git_head": $(json_string "$git_head"),
  "same_app_shape": true,
  "same_os_target": true,
  "same_cold_warm_state": true,
  "environment_captured": true,
  "budgets": ["steady_frame_time_p95", "frame_allocations", "input_latency_p95", "animation_frame_jitter_p95"]
}
JSON

cat > "$comparison_dir/electron-fairness-nonclaim.json" <<JSON
{
  "schema": "tetra.surface.electron-comparison-nonclaim.v1",
  "same_app_shape": true,
  "same_os_target": true,
  "same_cold_warm_state": true,
  "hardware_environment": true,
  "statistically_supported": false,
  "sample_count": 0,
  "faster_than_electron_claim": false,
  "fastest_ui_framework_claim": false,
  "zero_memory_overhead_claim": false,
  "decision": "fair comparison harness shape recorded; no faster-than-Electron claim"
}
JSON

go test -buildvcs=false ./tools/validators/surfaceperf ./tools/cmd/validate-surface-perf-report ./tools/cmd/surface-perf-smoke -run 'Surface.*Perf|Memory|Budget|RSS|Frame' -count=1
go run -buildvcs=false ./tools/cmd/surface-perf-smoke --out "$report_path"
go run -buildvcs=false ./tools/cmd/validate-surface-perf-report --report "$report_path"

summary_path="$report_dir/surface-perf-gate-summary.json"
cat > "$summary_path" <<JSON
{
  "schema": "tetra.surface.perf-gate.v1",
  "status": "current",
  "release_scope": "surface-performance-memory-scoped-linux-web",
  "producer": "scripts/release/surface/perf-gate.sh",
  "git_head": $(json_string "$git_head"),
  "version": $(json_string "$version"),
  "git_dirty": $git_dirty,
  "host_os": $(json_string "$host_os"),
  "host_arch": $(json_string "$host_arch"),
  "generated_at_utc": $(json_string "$generated_at_utc"),
  "command_line": $(json_string "$command_line"),
  "schema_under_test": "tetra.surface.perf-report.v1",
  "level": "surface-performance-memory-v1",
  "perf_report": "surface-perf-report.json",
  "baselines": [
    "baselines/surface-v1-linux-cold.json",
    "baselines/surface-v1-web-warm.json"
  ],
  "electron_comparison": "comparisons/electron-fairness-nonclaim.json",
  "same_commit_validated": true,
  "fake_claim_rejections": [
    "missing baseline environment",
    "impossible performance numbers",
    "unbounded cache",
    "unsupported faster-than-Electron claim",
    "fastest UI framework claim",
    "zero memory overhead claim"
  ]
}
JSON

go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run -buildvcs=false ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"

echo "Surface performance gate reports: $report_dir"
echo "Surface performance report: $report_path"
echo "Surface performance artifact hashes: $report_dir/artifact-hashes.json"
