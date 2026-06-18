#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/safe-view-lifetime"

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/safe-view-lifetime/gate.sh [--report-dir DIR]

Runs the focused Safe View Lifetime Contracts v1 gate and writes proof,
allocation, boundary-negative, and summary artifacts into DIR.
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
tmp_dir="$(mktemp -d)"

export GOCACHE="${GOCACHE:-${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-safe-view-lifetime-gate}"

safe_view_lifetime_active_pid=""

cleanup() {
  if [[ -n "${safe_view_lifetime_active_pid:-}" ]] && kill -0 "$safe_view_lifetime_active_pid" 2> /dev/null; then
    kill -TERM "$safe_view_lifetime_active_pid" 2> /dev/null || true
    sleep 1
    kill -KILL "$safe_view_lifetime_active_pid" 2> /dev/null || true
  fi
  rm -rf "$tmp_dir"
  GOCACHE="$GOCACHE" go clean -cache > /dev/null 2>&1 || true
}
trap safe_view_lifetime_cleanup EXIT

safe_view_lifetime_cleanup() {
  cleanup
}

safe_view_lifetime_timeout_seconds() {
  local timeout_seconds="${SAFE_VIEW_LIFETIME_TIMEOUT_SECONDS:-180}"
  if ! command -v timeout > /dev/null 2>&1; then
    echo "error: missing timeout command for bounded safe-view lifetime gate" >&2
    exit 1
  fi
  if ! [[ "$timeout_seconds" =~ ^[0-9]+$ ]] || [[ "$timeout_seconds" -le 0 ]]; then
    echo "error: SAFE_VIEW_LIFETIME_TIMEOUT_SECONDS must be a positive integer" >&2
    exit 2
  fi
  printf '%s' "$timeout_seconds"
}

safe_view_lifetime_slug() {
  local value="$1"
  value="${value// /-}"
  value="${value//\//-}"
  value="${value//:/-}"
  printf '%s' "$value"
}

safe_view_lifetime_run_step() {
  local label="$1"
  shift
  local timeout_seconds
  timeout_seconds="$(safe_view_lifetime_timeout_seconds)"
  local slug
  slug="$(safe_view_lifetime_slug "$label")"
  local log_path="$report_dir/safe-view-step-${slug}.log"

  echo "== $label =="
  set +e
  timeout --kill-after=5s "${timeout_seconds}s" "$@" > "$log_path" 2>&1 &
  safe_view_lifetime_active_pid=$!
  wait "$safe_view_lifetime_active_pid"
  local status=$?
  safe_view_lifetime_active_pid=""
  set -e

  if [[ "$status" -ne 0 ]]; then
    if [[ "$status" -eq 124 || "$status" -eq 137 ]]; then
      echo "error: $label timed out after ${timeout_seconds}s; see $log_path" >&2
    else
      echo "error: $label failed with exit $status; see $log_path" >&2
    fi
    tail -n 80 "$log_path" >&2 || true
    exit "$status"
  fi
}

safe_view_lifetime_run_expected_failure() {
  local label="$1"
  local output_path="$2"
  shift 2
  local timeout_seconds
  timeout_seconds="$(safe_view_lifetime_timeout_seconds)"
  local slug
  slug="$(safe_view_lifetime_slug "$label")"
  local log_path="$report_dir/safe-view-step-${slug}.log"

  echo "== $label =="
  set +e
  timeout --kill-after=5s "${timeout_seconds}s" "$@" > "$log_path" 2>&1 &
  safe_view_lifetime_active_pid=$!
  wait "$safe_view_lifetime_active_pid"
  local status=$?
  safe_view_lifetime_active_pid=""
  set -e

  cat "$log_path" >> "$output_path"
  if [[ "$status" -eq 0 ]]; then
    echo "error: expected $label to fail; see $log_path" >&2
    exit 1
  fi
  if [[ "$status" -eq 124 || "$status" -eq 137 ]]; then
    echo "error: expected $label diagnostic failure, but it timed out after ${timeout_seconds}s; see $log_path" >&2
    exit "$status"
  fi
}

run_go_test() {
  local label="$1"
  shift
  safe_view_lifetime_run_step "$label" env GOCACHE="$GOCACHE" go test "$@"
}

run_go() {
  local label="$1"
  shift
  safe_view_lifetime_run_step "$label" env GOCACHE="$GOCACHE" go run "$@"
}

require_contains() {
  local path="$1"
  local text="$2"
  if ! grep -Fq "$text" "$path"; then
    echo "expected $path to contain: $text" >&2
    exit 1
  fi
}

require_not_contains() {
  local path="$1"
  local text="$2"
  if grep -Fq "$text" "$path"; then
    echo "expected $path not to contain: $text" >&2
    exit 1
  fi
}

echo "== Surface lifetime/resource cleanup tests =="
run_go_test "surface-close-frame-event-resize-resource-cleanup" -buildvcs=false ./compiler/tests/semantics ./compiler/internal/semantics ./tools/scriptstest -run 'SurfaceClose|FrameAfterClose|DoubleClose|BeginPresent|ResizeAfterClose|ResourceCleanup|BrowserProcessCleanup|SafeViewLifetime' -count=1

echo "== Safe view proof/allocation report artifacts =="
run_go "safe-view-borrow-return-build" ./cli/cmd/tetra build --target linux-x64 --emit-proof --emit-alloc-report -o "$report_dir/safe-view-borrow-return" examples/memory/safe_view/safe_view_borrow_return.tetra
run_go "safe-view-copy-escape-build" ./cli/cmd/tetra build --target linux-x64 --emit-proof --emit-alloc-report -o "$report_dir/safe-view-copy-escape" examples/memory/safe_view/safe_view_copy_escape.tetra

for artifact in \
  "$report_dir/safe-view-borrow-return.proof.json" \
  "$report_dir/safe-view-borrow-return.alloc.json" \
  "$report_dir/safe-view-copy-escape.proof.json" \
  "$report_dir/safe-view-copy-escape.alloc.json"; do
  test -s "$artifact"
done

require_contains "$report_dir/safe-view-borrow-return.proof.json" '"kind": "borrowed_imm"'
require_contains "$report_dir/safe-view-borrow-return.proof.json" '"kind": "no_escape"'
require_contains "$report_dir/safe-view-borrow-return.proof.json" '"kind": "derived_window"'
require_not_contains "$report_dir/safe-view-borrow-return.alloc.json" '"value_id": "alloc_intent:v"'
require_contains "$report_dir/safe-view-copy-escape.proof.json" '"kind": "owned"'
require_contains "$report_dir/safe-view-copy-escape.proof.json" '"kind": "provenance_known"'
require_contains "$report_dir/safe-view-copy-escape.alloc.json" '"value_id": "alloc_intent:$return"'
require_contains "$report_dir/safe-view-copy-escape.alloc.json" '"length_expr": "borrowed.len"'

echo "== Boundary negative diagnostics =="
boundary_log="$report_dir/safe-view-boundary-negative.txt"
: > "$boundary_log"

cat > "$tmp_dir/bad-actor.tetra" << 'TETRA'
enum Msg:
    case bytes([]u8)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(1)
    return core.send_typed(core.self(), Msg.bytes(xs.borrow()))
TETRA
safe_view_lifetime_run_expected_failure "boundary-negative-bad-actor" "$boundary_log" env GOCACHE="$GOCACHE" go run ./cli/cmd/tetra check "$tmp_dir/bad-actor.tetra"

cat > "$tmp_dir/bad-task.tetra" << 'TETRA'
enum TaskErr:
    case bytes([]u8)

func worker() -> Int throws TaskErr:
    return 0

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)
TETRA
safe_view_lifetime_run_expected_failure "boundary-negative-bad-task" "$boundary_log" env GOCACHE="$GOCACHE" go run ./cli/cmd/tetra check "$tmp_dir/bad-task.tetra"

require_contains "$boundary_log" "cannot cross actor boundary without copy"
require_contains "$boundary_log" "typed task error payload must be sendable across task boundary"

cat > "$report_dir/safe-view-lifetime-summary.json" << JSON
{
  "schema": "tetra.safe-view-lifetime.gate.v1",
  "status": "pass",
  "bounded": true,
  "release_blocking": true,
  "timeout_seconds": ${SAFE_VIEW_LIFETIME_TIMEOUT_SECONDS:-180},
  "report_dir": "$report_dir",
  "resource_cleanup": "pass",
  "surface_lifecycle": "surface-close-frame-event-resize-resource-cleanup",
  "step_logs": [
    "safe-view-step-surface-close-frame-event-resize-resource-cleanup.log",
    "safe-view-step-safe-view-borrow-return-build.log",
    "safe-view-step-safe-view-copy-escape-build.log",
    "safe-view-step-boundary-negative-bad-actor.log",
    "safe-view-step-boundary-negative-bad-task.log"
  ],
  "artifacts": [
    "safe-view-borrow-return.proof.json",
    "safe-view-borrow-return.alloc.json",
    "safe-view-copy-escape.proof.json",
    "safe-view-copy-escape.alloc.json",
    "safe-view-boundary-negative.txt"
  ]
}
JSON

echo "Safe view lifetime gate reports: $report_dir"
