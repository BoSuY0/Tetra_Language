# Benchmark vNext Memory Baseline Audit

Status: current local evidence audit for
`docs/plans/2026-06-13-benchmark-vnext-memory-baseline.md`.

This audit is based on the fresh memory-aware Tier 1 report with runtime heap
and process RSS sidecars at
`reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/report.json`.
It is local benchmark evidence only. It does not claim Tetra is globally
faster than another language, does not claim cross-machine reproducibility, does
not claim production readiness, and does not claim official TechEmpower
results.

## Fresh Evidence

- Report: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/report.json`
- Generated at: `2026-06-13T20:51:54Z`
- Git commit: `95bfd4a887bab5032437cb22494d034e82ae6d35`
- Categories: 17
- Rows: 68
- Measured rows: 67
- Build failed rows: 1
- Tetra rows: 17
- Tetra rows with `memory_evidence`: 17
- Successful Tetra rows with `heap_alloc_bytes.runtime_measured`: 16
- Successful Tetra rows with `rss_current.runtime_measured`: 16
- Successful Tetra rows with `rss_peak.runtime_measured`: 16
- Build-failed Tetra rows with `heap_alloc_bytes.blocked`: 1
- Build-failed Tetra rows with `rss_current.blocked`: 1
- Build-failed Tetra rows with `rss_peak.blocked`: 1
- Raw RSS sidecars: 48

The report validates with:

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/report.json
```

## Classification Summary

| Classification | Count | Categories |
| --- | ---: | --- |
| `blocked by fallback backend` | 4 | `integer loops`, `function calls`, `recursion`, `allocation` |
| `blocked by bounds check` | 3 | `slice sum`, `bounds-check loops`, `matrix multiply` |
| `blocked by heap allocation` | 1 | `hash table` |
| `blocked by actor/runtime limitation` | 2 | `actor ping-pong`, `parallel map/reduce` |
| `blocked by missing feature` | 1 | `region/island allocation` |
| `invalid/inconclusive` | 3 | `JSON parse/stringify`, `HTTP plaintext/json`, `PostgreSQL single/multiple/update` |
| `comparable` | 1 | `binary size` |
| `faster than C/C++/Rust locally` | 2 | `startup time`, `compile time` |

The `JSON parse/stringify`, `HTTP plaintext/json`, and
`PostgreSQL single/multiple/update` rows remain helper-kernel evidence, not
full service/database benchmark evidence.

## Memory Evidence Boundary

Every Tetra row carries
`tetra.local_benchmark.memory_evidence.v1`.

Current memory evidence is intentionally split by evidence source:

- `heap_alloc_bytes` for successful linux-x64 Tetra rows is
  `runtime_measured` with method `tetra_linux_x64_heap_telemetry_v1`.
  The source artifact is a raw
  `tetra.runtime.heap_telemetry.v1` sidecar written by the benchmarked Tetra
  binary. The row-level metric uses the selected per-iteration sidecar with the
  maximum observed `heap_peak_bytes`.
- Some successful rows report zero runtime heap bytes. That is a measured
  runtime sidecar result for the optimized binary, not an unsupported
  placeholder.
- `bytes_requested`, `bytes_reserved`, `bytes_copied`, and allocation-domain
  bytes still use `allocation_report_estimate` when backed by an allocation
  report.
- `rss_current` and `rss_peak` for successful linux-x64 Tetra rows are
  `runtime_measured` from raw
  `tetra.local_benchmark.process_rss_telemetry.v1` sidecars collected by the
  Tier 1 runner while the benchmarked process executes. `rss_current` uses
  live `/proc/<pid>/status` `VmRSS` samples; `rss_peak` uses Linux
  `wait4`/`ru_maxrss` process accounting.
- `bytes_committed` remains `unsupported` because current allocation reports do
  not expose committed bytes.
- Build-failed Tetra rows use `blocked` memory metrics with an explicit build
  artifact reason.

This means the fresh report can support sidecar-backed Tetra runtime heap
evidence, sidecar-backed process RSS evidence, and allocation-domain estimates,
but it still cannot support hard RSS thresholds, cross-machine RSS claims, or
official benchmark memory claims.

## Blocker Ranking

### 1. Fallback Backend

Leverage: highest. It blocks four primary Tier 1 categories and appears in
simple scalar programs before any memory-specific optimization can matter.

Rows:

- `integer_loops_tetra`
  - Classification: `blocked by fallback backend`
  - Backend: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/integer_loops_tetra.backend.json`
  - Evidence: `unsupported_control_flow`, one stack fallback function.
- `function_calls_tetra`
  - Backend: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/function_calls_tetra.backend.json`
  - Evidence: helper function is register path, `main` falls back through
    `unsupported_control_flow`.
- `recursion_tetra`
  - Backend: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/recursion_tetra.backend.json`
  - Evidence: `fib` and `main` both fall back through
    `unsupported_control_flow`.
- `allocation_tetra`
  - Backend: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/allocation_tetra.backend.json`
  - Evidence: `unsupported_effect_runtime_call`.

Recommended follow-up:
`docs/plans/2026-06-13-benchmark-vnext-fallback-backend-track.md`.

### 2. Bounds-Check Elimination

Leverage: high. It blocks three core numeric/memory categories and accounts for
11 remaining checks across the primary bounded-loop rows.

Rows:

- `slice_sum_tetra`
  - Bounds: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/slice_sum_tetra.bounds.json`
  - Evidence: 2 checks left, both `left_missing_dominance`.
- `bounds_check_loops_tetra`
  - Bounds: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/bounds_check_loops_tetra.bounds.json`
  - Evidence: 2 checks left.
- `matrix_multiply_tetra`
  - Bounds: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/matrix_multiply_tetra.bounds.json`
  - Evidence: 7 checks left.

Recommended follow-up:
`docs/plans/2026-06-13-benchmark-vnext-bounds-check-track.md`.

### 3. Hash Table Heap Allocation

Leverage: medium and memory-specific. It is the first clear heap blocker in the
primary comparable matrix, but it is narrower than fallback backend or bounds.

Row:

- `hash_table_tetra`
  - Allocation: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/hash_table_tetra.alloc.json`
  - Evidence: 2 heap allocations, `keys` and `values`.
  - Reason: unknown call escape through `lookup` requires conservative heap
    fallback.
  - Domain: `domain:process`.

The owner is not the runtime allocator itself. The immediate owner is compiler
escape/interprocedural summary evidence: the allocation planner cannot yet prove
that passing `keys` and `values` to `lookup` does not store, return, or transfer
the slices.

Recommended follow-up:
`docs/plans/2026-06-13-benchmark-vnext-hash-table-heap-track.md`.

### 4. Actor Runtime Benchmark Limitation

Leverage: important for the long-term runtime story, but it must stay separate
from Tier 1 scalar/memory optimizer work.

Rows:

- `actor_ping_pong_tetra`
  - Backend: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/actor_ping_pong_tetra.backend.json`
  - Evidence: actor runtime calls such as `__tetra_actor_recv` and
    `__tetra_actor_spawn` use stack fallback.
- `parallel_map_reduce_tetra`
  - Backend: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/parallel_map_reduce_tetra.backend.json`
  - Evidence: worker functions are register path, `main` falls back on
    `__tetra_task_spawn_i32` multi-slot return ABI.

Current actor benchmark prep rows in `compiler/internal/parallelrt` and
`tools/cmd/parallel-production-smoke` are Tier 0/Tier 1 preparation-only rows.
They do not publish measured throughput, parity, production scheduler, or
distributed zero-copy claims.

Recommended follow-up:
`docs/plans/2026-06-13-benchmark-vnext-actor-runtime-track.md`.

## Separate Blockers

- `region/island allocation` is a missing-feature/build-failure row, not a
  benchmark optimization row. The build fails before artifacts are produced, so
  memory evidence is explicitly `blocked`.
- TechEmpower-compatible reports remain outside Tier 1 language benchmark
  claims. They must be validated separately with
  `tools/cmd/validate-techempower-report`.

## Verification Requirements

Before promoting any blocker as fixed:

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go test ./tools/internal/rsstelemetry ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -count=1
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-rss-current-head --iterations 3
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/report.json
GOCACHE=$(pwd)/.cache/go-build-rss-sampling-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```

No optimization track may replace blocked evidence with an unsupported claim.
