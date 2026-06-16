#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
report_dir_arg="reports/actor-runtime-foundation/final"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh [--report-dir DIR]

Runs the scoped Linux-x64 actor runtime foundation production gate. The gate
requires distributed actor runtime smoke, parallel production smoke, focused
actor diagnostics tests, race actor slice, docs/manifest checks, same-commit
metadata, scoped nonclaims, and final artifact hash integrity.
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
gate_contract="scripts/release/post_v0_4/contracts/actor-runtime-foundation-linux-x64.json"
source "$repo_root/scripts/release/surface/report-dir-guard.sh"
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$repo_root/.cache/go-build-actor-runtime-foundation-gate"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-actor-runtime-foundation-gate"
fi
mkdir -p "$GOCACHE" "$GOTMPDIR"

go run ./tools/cmd/run-gate --contract "$gate_contract" --report-dir "$report_dir_arg" --dry-run >/dev/null

report_dir="$(surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "actor_runtime_foundation_gate:")"
distributed_report_dir="$report_dir_arg/distributed-actors-linux-x64"
parallel_report_dir="$report_dir_arg/parallel-production-linux-x64"
manifest_path="$report_dir/actor-runtime-foundation-manifest.json"
log_dir="$report_dir/logs"
mkdir -p "$log_dir"

json_string() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '"%s"' "$value"
}

json_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '%s' "$value"
}

actor_gate_run() {
  local name="$1"
  shift
  "$@" >"$log_dir/$name.log" 2>&1
}

actor_gate_run distributed-actors-smoke bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir "$distributed_report_dir"
actor_gate_run parallel-production-smoke bash "$script_dir/parallel-production-linux-x64-smoke.sh" --report-dir "$parallel_report_dir"
actor_gate_run focused-actor-tests go test -buildvcs=false ./cli/cmd/tetra ./compiler/tests/ownership ./compiler -run 'Diagnostic|Actor|Backpressure|Invalid|Closed|Transfer' -count=1
actor_gate_run race-actor-slice go test -race -buildvcs=false ./compiler ./cli/internal/actornet -run 'Actor|Broker' -count=1
actor_gate_run validate-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
actor_gate_run verify-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json

git_head="$(git rev-parse --verify HEAD)"

cat > "$manifest_path" <<MANIFEST
{
  "schema": "tetra.actor.production_foundation.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "git_head": $(json_string "$git_head"),
  "report_dir": ".",
  "artifact_hashes": "artifact-hashes.json",
  "claims": [
    "linux-x64 scoped actor/task runtime foundation evidence"
  ],
  "nonclaims": [
    "no full Erlang/OTP actor runtime claim",
    "no cluster membership or reconnect/retry production claim",
    "no non-Linux distributed actor runtime support claim",
    "no distributed zero-copy pointer or region transfer claim",
    "no formal race proof claim"
  ],
  "commands": [
    {"name":"distributed-actors-smoke","command":"bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir $(json_escape "$distributed_report_dir")","status":"pass","log":"logs/distributed-actors-smoke.log"},
    {"name":"parallel-production-smoke","command":"bash scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh --report-dir $(json_escape "$parallel_report_dir")","status":"pass","log":"logs/parallel-production-smoke.log"},
    {"name":"focused-actor-tests","command":"go test -buildvcs=false ./cli/cmd/tetra ./compiler/tests/ownership ./compiler -run 'Diagnostic|Actor|Backpressure|Invalid|Closed|Transfer' -count=1","status":"pass","log":"logs/focused-actor-tests.log"},
    {"name":"race-actor-slice","command":"go test -race -buildvcs=false ./compiler ./cli/internal/actornet -run 'Actor|Broker' -count=1","status":"pass","log":"logs/race-actor-slice.log"},
    {"name":"validate-manifest","command":"go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json","status":"pass","log":"logs/validate-manifest.log"},
    {"name":"verify-docs","command":"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json","status":"pass","log":"logs/verify-docs.log"},
    {"name":"artifact-hashes-write","command":"go run ./tools/cmd/validate-artifact-hashes --write --root $(json_escape "$report_dir_arg") --out $(json_escape "$report_dir_arg")/artifact-hashes.json","status":"pass","log":"logs/artifact-hashes-write.log"},
    {"name":"artifact-hashes-validate","command":"go run ./tools/cmd/validate-artifact-hashes --manifest $(json_escape "$report_dir_arg")/artifact-hashes.json","status":"pass","log":"stdout"},
    {"name":"actor-foundation-validator","command":"go run ./tools/cmd/validate-actor-runtime-foundation --report-dir $(json_escape "$report_dir_arg") --current-git-head $git_head","status":"pass","log":"stdout"}
  ],
  "artifacts": [
    {"path":"actor-runtime-foundation-manifest.json","kind":"foundation_manifest","schema":"tetra.actor.production_foundation.v1"},
    {"path":"parallel-production-linux-x64/parallel-production-linux-x64.json","kind":"parallel_production_report","schema":"tetra.parallel.production.v1"},
    {"path":"parallel-production-linux-x64/artifact-hashes.json","kind":"parallel_hash_manifest","schema":"tetra.release-artifact-hashes.v1alpha1"},
    {"path":"distributed-actors-linux-x64/distributed-actors-linux-x64.json","kind":"distributed_actor_runtime_report","schema":"tetra.actors.distributed-runtime.v1"},
    {"path":"distributed-actors-linux-x64/artifact-hashes.json","kind":"distributed_hash_manifest","schema":"tetra.release-artifact-hashes.v1alpha1"},
    {"path":"artifact-hashes.json","kind":"foundation_hash_manifest","schema":"tetra.release-artifact-hashes.v1alpha1"}
  ]
}
MANIFEST

actor_gate_run artifact-hashes-write go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
go run ./tools/cmd/validate-actor-runtime-foundation --report-dir "$report_dir" --current-git-head "$git_head"

echo "Actor runtime foundation gate reports: $report_dir"
echo "Actor runtime foundation manifest: $manifest_path"
echo "Actor runtime foundation artifact hashes: $report_dir/artifact-hashes.json"
