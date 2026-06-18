# Benchmark vNext Actor Runtime Track

Status: follow-up plan opened from the fresh memory-aware Tier 1 baseline.

Primary audit: `docs/audits/memory/zero-heap-final/benchmark-vnext-memory-baseline.md`.

## Goal

Keep current actor/runtime benchmark limitations explicit and define a narrow promotion path for
measured local actor evidence without turning prototype prep rows into production or distributed
claims.

## Current Evidence

Fresh report: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/report.json`.

Rows:

| Row                         | Current limitation                                                                    | Artifact                                                                                                              |
| --------------------------- | ------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `actor_ping_pong_tetra`     | actor runtime calls use stack fallback; Tier 1 row remains a bounded local limitation | `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/actor_ping_pong_tetra.backend.json`     |
| `parallel_map_reduce_tetra` | workers are register path, but `main` falls back on task-spawn multi-slot ABI         | `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/parallel_map_reduce_tetra.backend.json` |

Related current evidence:

- `compiler/internal/parallelrt/scheduler_model.go` has `PrototypeBenchmarks` prep rows.
- `compiler/internal/parallelrt/scheduler_model.go` has `ActorMemoryDomainReport` and validation
  helpers.
- `tools/cmd/parallel-production-smoke` emits actor benchmark prep rows as Tier 0 local smoke only.
- `docs/benchmarks/truth_benchmark_harness.md` forbids promoting prep rows to measured benchmark or
  superiority claims.

## Boundary

Current actor evidence may say:

- local actor/task runtime evidence exists in bounded forms;
- typed mailbox and actor memory-domain model evidence exists;
- prep rows exist for actor benchmark shapes.

Current actor evidence must not say:

- measured actor throughput is published for Tier 1;
- actor runtime is production parallel/distributed;
- actor benchmark parity with C++/Rust/Go/Erlang is proven;
- zero-copy move is distributed or general-purpose;
- actor memory domains are full runtime RSS accounting.

## Proposed Promotion Path

Do not mix this with scalar fallback or bounds work. Actor promotion needs a separate local harness.

Phase 1:

- add a local actor benchmark harness that actually runs actor ping-pong with bounded iterations,
  warmup, repeat count, median latency/throughput, and raw output artifacts;
- attach explicit memory evidence with actor-domain bytes when the source is actor domain
  accounting, otherwise mark runtime RSS/heap unsupported;
- validate that the report cannot claim official, production, distributed, or cross-runtime results.

Phase 2:

- add measured `parallel map/reduce` only after task-spawn ABI and scheduler evidence are explicit;
- keep actor/domain byte backpressure separate from message-count backpressure until the runtime has
  byte-aware enforcement.

Phase 3:

- after local measured rows exist and validate, decide whether those rows belong in Tier 1 or in a
  separate actor benchmark suite.

## Likely Files

- `compiler/internal/parallelrt/scheduler_model.go`
- `compiler/internal/parallelrt/scheduler_model_test.go`
- `compiler/internal/actorsrt`
- `compiler/internal/actorsafety`
- `tools/cmd/parallel-production-smoke`
- `tools/validators/parallelprod`
- `docs/benchmarks/truth_benchmark_harness.md`
- `tools/cmd/local-benchmark-tier1`

## Tests First

Add or extend focused tests that fail before implementation:

- validator rejects actor benchmark superiority/production/distributed claims;
- actor harness report requires raw artifacts, repeat count, and measurement method;
- actor memory-domain evidence validates as domain accounting, not RSS;
- prep rows remain Tier 0/Tier 1 preparation-only unless measured fields are present.

## Verification

Focused:

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-actor go test ./compiler/internal/parallelrt/... ./compiler/internal/actorsrt/... ./compiler/internal/actorsafety/... ./tools/validators/parallelprod/... ./tools/cmd/parallel-production-smoke -run 'Actor|Benchmark|MemoryDomain|ZeroCopy|Claim|Tier' -count=1
```

Benchmark or harness, once added:

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-actor go run ./tools/cmd/parallel-production-smoke
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-actor go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-after-actor-track --iterations 3
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-actor go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-actor-track/report.json
```

## Nonclaims

- No production actor runtime claim.
- No official actor benchmark claim.
- No distributed zero-copy claim.
- No actor RSS claim without a process-level RSS sampler.
