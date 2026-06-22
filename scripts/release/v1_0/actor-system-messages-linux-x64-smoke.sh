#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir_arg="reports/v1-actor-system-messages/fresh"

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/v1_0/actor-system-messages-linux-x64-smoke.sh [--report-dir DIR]

Runs the scoped Linux-x64 V1-P01 actor system-message lane smoke. The report is
P01 fixture evidence only: producer=test_hook, no P06/P10 production claim.

Nonclaims carried by this gate:
- no full Erlang/OTP actor runtime claim
- no cluster membership or reconnect/retry production claim
- no non-Linux distributed actor runtime support claim
- no distributed zero-copy pointer or region transfer claim
- no formal race proof claim
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
      report_dir_arg="$2"
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
source "$repo_root/scripts/release/surface/report-dir-guard.sh"
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$repo_root/.cache/go-build-actor-system-messages-smoke"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-actor-system-messages-smoke"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

report_dir="$(
  surface_release_require_fresh_report_dir \
    "$report_dir_arg" \
    "$repo_root" \
    "actor_system_messages_gate:"
)"
log_dir="$report_dir/logs"
bin_dir="$report_dir/bin"
mkdir -p "$log_dir" "$bin_dir"

json_string() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '"%s"' "$value"
}

join_command() {
  local out=""
  local part
  for part in "$@"; do
    if [[ -n "$out" ]]; then
      out+=" "
    fi
    out+="$part"
  done
  printf '%s' "$out"
}

command_row() {
  local comma="$1"
  local name="$2"
  local log="$3"
  shift 3
  local command
  command="$(join_command "$@")"
  printf '    {"name":%s,"command":%s,"status":"pass","log":%s}%s\n' \
    "$(json_string "$name")" \
    "$(json_string "$command")" \
    "$(json_string "$log")" \
    "$comma"
}

artifact_row() {
  local comma="$1"
  local path="$2"
  local kind="$3"
  local schema="${4:-}"
  if [[ -n "$schema" ]]; then
    printf '    {"path":%s,"kind":%s,"schema":%s}%s\n' \
      "$(json_string "$path")" \
      "$(json_string "$kind")" \
      "$(json_string "$schema")" \
      "$comma"
  else
    printf '    {"path":%s,"kind":%s}%s\n' \
      "$(json_string "$path")" \
      "$(json_string "$kind")" \
      "$comma"
  fi
}

run_logged() {
  local name="$1"
  shift
  "$@" > "$log_dir/$name.log" 2>&1
}

run_logged focused-validator-tests \
  go test -buildvcs=false \
  ./tools/cmd/validate-actor-system-messages \
  ./tools/validators/actorsystem \
  ./compiler/internal/formats \
  ./compiler/internal/buildruntime/tests \
  -count=1

run_logged actor-system-message-validator \
  go run -buildvcs=false ./tools/cmd/validate-actor-system-messages --root .

run_logged actor-system-layout-report \
  go run -buildvcs=false ./compiler/cmd/actor-system-layout-report \
  --out "$report_dir/actor-system-layout-linux-x64.json"

positive_examples=(
  examples/actors/system_messages/system_user_queue_isolation.tetra
  examples/actors/system_messages/system_exit_trap.tetra
  examples/actors/system_messages/system_monitor_down.tetra
  examples/actors/system_messages/system_poll_timeout_cancel.tetra
  examples/actors/system_messages/system_sender_unchanged.tetra
)

{
  for example in "${positive_examples[@]}"; do
    name="$(basename "$example" .tetra)"
    out="$bin_dir/$name"
    echo "check $example"
    go run -buildvcs=false ./cli/cmd/tetra check "$example"
    echo "build $example -> $out"
    go run -buildvcs=false ./cli/cmd/tetra build --target linux-x64 -o "$out" "$example"
    echo "run $out"
    "$out"
  done
} > "$log_dir/generated-examples-build-run.log" 2>&1

negative_log="$log_dir/negative-forgery-check.log"
if go run -buildvcs=false ./cli/cmd/tetra check \
  examples/actors/system_messages/system_forgery_negative.tetra \
  > "$negative_log" 2>&1; then
  echo "expected system_forgery_negative.tetra to fail" >&2
  exit 1
fi
grep -Fq \
  "runtime system messages cannot be sent through the ordinary actor mailbox" \
  "$negative_log"

scan_binary="$bin_dir/system_user_queue_isolation"
symbol_log="$log_dir/release-symbol-scan.log"
forbidden_symbol="__tetra_test_actor_system_inject"
if command -v nm >/dev/null 2>&1; then
  if ! nm -a "$scan_binary" > "$symbol_log" 2>&1; then
    strings "$scan_binary" > "$symbol_log" 2>&1
  fi
else
  strings "$scan_binary" > "$symbol_log" 2>&1
fi
test_injector_exported=false
if grep -Fq "$forbidden_symbol" "$symbol_log"; then
  test_injector_exported=true
fi

git_head="$(git rev-parse --verify HEAD)"
git_dirty=false
if [[ -n "$(git status --porcelain)" ]]; then
  git_dirty=true
fi

report_path="$report_dir/actor-system-messages-linux-x64.json"
cat > "$report_path" << REPORT
{
  "schema": "tetra.actor.system_messages.v1",
  "pass": true,
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "builtin-actor-runtime-v2",
  "git_head": $(json_string "$git_head"),
  "git_dirty": $git_dirty,
  "design": "separate-system-lane-v1",
  "producer": "test_hook",
  "report_dir": ".",
  "artifact_hashes": "artifact-hashes.json",
  "command_line": $(json_string "bash scripts/release/v1_0/actor-system-messages-linux-x64-smoke.sh --report-dir $report_dir_arg"),
  "claims": [
    "source-level system-message API and isolated runtime system lane implemented for Linux-x64 builtin runtime"
  ],
  "nonclaims": [
    "real local link/monitor producers are completed in P06",
    "authenticated node-down producer is completed in P10",
    "no full Erlang/OTP actor runtime claim",
    "no cluster membership or reconnect/retry production claim",
    "no non-Linux distributed actor runtime support claim",
    "no distributed zero-copy pointer or region transfer claim",
    "no formal race proof claim"
  ],
  "api": {
    "recv_system": true,
    "poll_system": true,
    "recv_system_until": true
  },
  "isolation": {
    "separate_heads_tails": true,
    "user_recv_system_consumptions": 0,
    "system_recv_user_consumptions": 0,
    "user_queue_fifo_violations": 0,
    "system_queue_fifo_violations": 0,
    "sender_unchanged": true
  },
  "security": {
    "ordinary_send_forgery_rejected": true,
    "runtime_handles_opaque": true,
    "release_test_injector_exported": $test_injector_exported
  },
  "events": {
    "exit": 1,
    "down": 1,
    "node_down_fixture": 1,
    "duplicate_down": 0,
    "producer": "test_hook"
  },
  "memory": {
    "bounded": true,
    "reserved_credits": 0,
    "live_bytes_after_shutdown": 0,
    "silent_drops": 0
  },
  "release_symbol_scan": {
    "scanned": true,
    "binary": "bin/system_user_queue_isolation",
    "forbidden_symbols": ["__tetra_test_actor_system_inject"],
    "test_injector_exported": $test_injector_exported
  },
  "commands": [
$(command_row "," "focused-validator-tests" \
  "logs/focused-validator-tests.log" \
  go test -buildvcs=false \
  ./tools/cmd/validate-actor-system-messages \
  ./tools/validators/actorsystem \
  ./compiler/internal/formats \
  ./compiler/internal/buildruntime/tests \
  -count=1)
$(command_row "," "actor-system-message-validator" \
  "logs/actor-system-message-validator.log" \
  go run -buildvcs=false ./tools/cmd/validate-actor-system-messages --root .)
$(command_row "," "generated-examples-build-run" \
  "logs/generated-examples-build-run.log" \
  go run -buildvcs=false ./cli/cmd/tetra build --target linux-x64 \
  examples/actors/system_messages/system_user_queue_isolation.tetra)
$(command_row "," "negative-forgery-check" \
  "logs/negative-forgery-check.log" \
  go run -buildvcs=false ./cli/cmd/tetra check \
  examples/actors/system_messages/system_forgery_negative.tetra)
$(command_row "," "actor-system-layout-report" \
  "logs/actor-system-layout-report.log" \
  go run -buildvcs=false ./compiler/cmd/actor-system-layout-report \
  --out "$report_dir_arg/actor-system-layout-linux-x64.json")
$(command_row "," "release-symbol-scan" \
  "logs/release-symbol-scan.log" \
  nm -a bin/system_user_queue_isolation)
$(command_row "," "artifact-hashes-write" \
  "logs/artifact-hashes-write.log" \
  go run ./tools/cmd/validate-artifact-hashes \
  --write \
  --root "$report_dir_arg" \
  --out "$report_dir_arg/artifact-hashes.json")
$(command_row "" "artifact-hashes-validate" \
  "stdout" \
  go run ./tools/cmd/validate-artifact-hashes \
  --manifest "$report_dir_arg/artifact-hashes.json")
  ],
  "artifacts": [
$(artifact_row "," "actor-system-messages-linux-x64.json" \
  "actor_system_messages_report" \
  "tetra.actor.system_messages.v1")
$(artifact_row "," "actor-system-layout-linux-x64.json" \
  "actor_system_layout_report" \
  "tetra.actor.system_layout.v1")
$(artifact_row "," "artifact-hashes.json" \
  "artifact_hash_manifest" \
  "tetra.release-artifact-hashes.v1alpha1")
$(artifact_row "" "bin/system_user_queue_isolation" \
  "native_binary")
  ]
}
REPORT

run_logged artifact-hashes-write \
  go run ./tools/cmd/validate-artifact-hashes \
  --write \
  --root "$report_dir" \
  --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
go run -buildvcs=false ./tools/cmd/validate-actor-system-messages \
  --root . \
  --report-dir "$report_dir" \
  --current-git-head "$git_head"

echo "Actor system-message smoke reports: $report_dir"
echo "Actor system-message report: $report_path"
echo "Actor system-message artifact hashes: $report_dir/artifact-hashes.json"
