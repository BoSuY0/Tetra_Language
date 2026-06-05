#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir="$repo_root/reports/safe-view-lifetime"

usage() {
  cat <<'USAGE'
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
tmp_dir="$(mktemp -d)"

export GOCACHE="${GOCACHE:-${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-safe-view-lifetime-gate}"

cleanup() {
  rm -rf "$tmp_dir"
  GOCACHE="$GOCACHE" go clean -cache >/dev/null 2>&1 || true
}
trap cleanup EXIT

run_go_test() {
  GOCACHE="$GOCACHE" go test "$@"
}

run_go() {
  GOCACHE="$GOCACHE" go run "$@"
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

echo "== Focused Go tests =="
run_go_test ./compiler/... -run 'Borrow|Copy|Lifetime|Ownership|Actor|Task|String|Slice|PLIR|Alloc|Proof' -count=1
run_go_test ./cli/... -run 'Borrow|Copy|Lifetime|Diagnostics' -count=1
run_go_test ./tools/... -run 'Manifest|Docs|PLIR|Proof|Alloc|SafeView' -count=1

echo "== Docs manifest =="
run_go ./tools/cmd/gen-manifest -o "$tmp_dir/manifest.json"
run_go ./tools/cmd/validate-manifest --manifest "$tmp_dir/manifest.json"
diff -u docs/generated/manifest.json "$tmp_dir/manifest.json"
run_go ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
run_go ./tools/cmd/verify-docs --manifest docs/generated/manifest.json

echo "== Safe view proof/allocation report artifacts =="
run_go ./cli/cmd/tetra build --target linux-x64 --emit-proof --emit-alloc-report -o "$report_dir/safe-view-borrow-return" examples/safe_view_borrow_return.tetra
run_go ./cli/cmd/tetra build --target linux-x64 --emit-proof --emit-alloc-report -o "$report_dir/safe-view-copy-escape" examples/safe_view_copy_escape.tetra

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
: >"$boundary_log"

cat >"$tmp_dir/bad-actor.tetra" <<'TETRA'
enum Msg:
    case bytes([]u8)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(1)
    return core.send_typed(core.self(), Msg.bytes(xs.borrow()))
TETRA
if run_go ./cli/cmd/tetra check "$tmp_dir/bad-actor.tetra" >>"$boundary_log" 2>&1; then
  echo "expected borrowed actor boundary check to fail" >&2
  exit 1
fi

cat >"$tmp_dir/bad-task.tetra" <<'TETRA'
enum TaskErr:
    case bytes([]u8)

func worker() -> Int throws TaskErr:
    return 0

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)
TETRA
if run_go ./cli/cmd/tetra check "$tmp_dir/bad-task.tetra" >>"$boundary_log" 2>&1; then
  echo "expected borrowed task boundary check to fail" >&2
  exit 1
fi

require_contains "$boundary_log" "cannot cross actor boundary without copy"
require_contains "$boundary_log" "typed task error payload must be sendable across task boundary"

cat >"$report_dir/safe-view-lifetime-summary.json" <<JSON
{
  "schema": "tetra.safe-view-lifetime.gate.v1",
  "status": "pass",
  "report_dir": "$report_dir",
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
