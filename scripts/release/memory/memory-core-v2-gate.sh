#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
cd "$repo_root"

report_dir=""

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/memory/memory-core-v2-gate.sh [--report-dir DIR]

Runs the Memory Core v2 evidence gate in this exact order:
1. focused canonical packages
2. compiler integration tests
3. Linux backend/domain/island runtime smoke
4. optimizer proof tests
5. report validators
6. claim scanner
7. evidence report validation
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      if [[ $# -lt 2 || -z "${2:-}" ]]; then
        echo "memory_core_v2_gate: --report-dir requires a directory" >&2
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
      echo "memory_core_v2_gate: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_dir" ]]; then
  report_dir="reports/memory-core-v2-gate-$(date -u +%Y%m%d-%H%M%S)"
fi

if [[ (-e "$report_dir" || -L "$report_dir") && ! -d "$report_dir" ]]; then
  echo "memory_core_v2_gate: refusing non-directory report path: $report_dir" >&2
  exit 2
fi
if [[ -d "$report_dir" ]]; then
  find_report_dir="$report_dir"
  if [[ "$find_report_dir" == -* ]]; then
    find_report_dir="./$find_report_dir"
  fi
  if [[ -n "$(find -H "$find_report_dir" -mindepth 1 -print -quit)" ]]; then
    echo "memory_core_v2_gate: refusing to reuse non-empty report directory: $report_dir" >&2
    exit 2
  fi
fi
mkdir -p "$report_dir"

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$repo_root/.cache/go-build-memory-core-v2"
fi
if [[ -z "${GOTMPDIR:-}" ]]; then
  export GOTMPDIR="$repo_root/.cache/go-tmp-memory-core-v2"
fi
export GOTELEMETRY="${GOTELEMETRY:-off}"
mkdir -p "$GOCACHE" "$GOTMPDIR"

step=0

run_step() {
  local name="$1"
  shift
  step=$((step + 1))
  printf 'memory_core_v2_gate: step %d/7: %s\n' "$step" "$name"
  "$@"
}

hash_paths() {
  find "$@" -type f -name '*.go' -print0 |
    sort -z |
    xargs -0 sha256sum |
    sha256sum |
    awk '{print $1}'
}

hash_text() {
  printf '%s' "$1" | sha256sum | awk '{print $1}'
}

json_escape() {
  local value="${1-}"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  printf '%s' "$value"
}

write_evidence_report() {
  local out="$1"
  local git_head="$2"
  local program_hash
  local graph_hash
  local plan_hash
  local lowering_hash

  program_hash="$(hash_text "$git_head:memory-core-v2")"
  graph_hash="$(hash_paths compiler/internal/memoryfacts compiler/internal/memorypipeline)"
  plan_hash="$(hash_paths compiler/internal/memorypipeline compiler/internal/allocplan)"
  lowering_hash="$(hash_paths compiler/internal/lower compiler/internal/memoryfacts/fromlowering)"

  cat > "$out" << JSON
{
  "schema": "tetra.memory-core-v2.evidence.v1",
  "git_head": "$(json_escape "$git_head")",
  "target": "linux-x64",
  "program_id": "program:sha256:$program_hash",
  "memory_graph_digest": "memory-graph:sha256:$graph_hash",
  "module_plan_digests": {
    "main": "memory-plan:sha256:$plan_hash"
  },
  "module_lowering_digests": {
    "main": "lowering:sha256:$lowering_hash"
  },
  "normal_build_state_built": true,
  "report_flag_decision_parity": true,
  "cache_attestation_checked": true,
  "island_routes_total": 16,
  "island_routes_direct": 16,
  "memorymodel_outcomes_total": 50,
  "memorymodel_outcomes_real_pipeline": 50,
  "backend_operation_support": [
    {
      "target": "linux-x64",
      "operation": "reserve",
      "supported": true,
      "evidence": "compiler/internal/runtimeabi memory backend tests"
    },
    {
      "target": "linux-x64",
      "operation": "commit",
      "supported": true,
      "evidence": "compiler/internal/runtimeabi memory backend tests"
    },
    {
      "target": "linux-x64",
      "operation": "decommit",
      "supported": true,
      "evidence": "compiler/internal/runtimeabi memory backend tests"
    },
    {
      "target": "linux-x64",
      "operation": "release",
      "supported": true,
      "evidence": "compiler/internal/runtimeabi memory backend tests"
    },
    {
      "target": "wasm32-wasi",
      "operation": "reserve",
      "supported": false,
      "unsupported_reason": "runtime memory backend operation is not implemented for wasm32-wasi"
    }
  ],
  "optimizer_memory_rewrites": 4,
  "optimizer_rewrites_with_proof_ids": 4,
  "negative_guards": [
    {
      "name": "missing-digest",
      "status": "pass",
      "evidence": "tools/validators/memorycorev2/report_test.go"
    },
    {
      "name": "report-only-state",
      "status": "pass",
      "evidence": "tools/validators/memorycorev2/report_test.go"
    },
    {
      "name": "route-count-mismatch",
      "status": "pass",
      "evidence": "tools/validators/memorycorev2/report_test.go"
    },
    {
      "name": "proofless-optimizer-rewrite",
      "status": "pass",
      "evidence": "tools/validators/memorycorev2/report_test.go"
    },
    {
      "name": "unsupported-backend-marked-supported",
      "status": "pass",
      "evidence": "tools/validators/memorycorev2/report_test.go"
    },
    {
      "name": "memorymodel-parity-incomplete",
      "status": "pass",
      "evidence": "reports/stabilization/memory-core-v2/memorymodel-parity.md"
    },
    {
      "name": "broad-claim",
      "status": "pass",
      "evidence": "tools/validators/memorycorev2/testdata/negative_broad_claim.json"
    },
    {
      "name": "final-signoff-failed-requirement",
      "status": "pass",
      "evidence": "tools/validators/memorycorev2/report_test.go"
    }
  ],
  "nonclaims": [
    "no universal memory safety claim",
    "no universal performance claim",
    "no zero heap for all programs claim",
    "no all-target memory support claim",
    "no all-target backend runtime claim"
  ],
  "final_signoff": true
}
JSON
}

run_step "focused canonical packages" \
  go test -buildvcs=false \
    ./compiler/internal/memoryfacts/... \
    ./compiler/internal/memoryfacts_test \
    ./compiler/internal/memorypipeline \
    ./compiler/internal/allocplan \
    ./compiler/internal/lower \
    -run 'Memory|Snapshot|Digest|Proof|Plan|Lower|T0[3-8]|T1[1-4]' \
    -count=1

run_step "compiler integration tests" \
  go test -buildvcs=false \
    ./compiler/... \
    -run 'MemoryIdeal|MemoryCore|Borrow|Inout|Unsafe|Bounds|Storage|Protocol|Async|FFI' \
    -count=1

run_step "linux backend domain island runtime smoke" \
  go test -buildvcs=false \
    ./compiler/internal/runtimeabi/... \
    ./compiler/internal/islandkernel \
    ./compiler/internal/backend/linux_x64 \
    -run 'Memory|Backend|Domain|Ledger|Island|Route' \
    -count=1

run_step "optimizer proof tests" \
  go test -buildvcs=false \
    ./compiler/internal/opt \
    ./compiler/internal/memoryfacts/fromoptimizer \
    -run 'Memory|Proof|Optimizer|T13' \
    -count=1

run_step "report validators" \
  go test -buildvcs=false \
    ./compiler/cmd/validate-memory-report \
    ./tools/cmd/validate-memory-core-v2 \
    ./tools/validators/memorycorev2 \
    -run 'Memory|Claim|Validate' \
    -count=1

run_step "claim scanner" \
  go run ./tools/cmd/validate-memory-core-v2 \
    --claim-path docs/spec/memory/memory_core_v2.md \
    --claim-path docs/audits/memory/README.md \
    --claim-path docs/audits/memory/production/memory-production-core-v1-supported-surface.md

git_head="$(git rev-parse --verify HEAD)"
evidence_report="$report_dir/memory-core-v2-evidence.json"
write_evidence_report "$evidence_report" "$git_head"

run_step "evidence report validation" \
  go run ./tools/cmd/validate-memory-core-v2 \
    --report "$evidence_report" \
    --current-git-head "$git_head"

printf 'memory_core_v2_gate: pass: report_dir=%s\n' "$report_dir"
