# Benchmark vNext Hash Table Heap Track

Status: follow-up plan opened from the fresh memory-aware Tier 1 baseline.

Primary audit: `docs/audits/memory/zero-heap-final/benchmark-vnext-memory-baseline.md`.

## Goal

Identify and remove the first heap-allocation blocker in the `hash table` row only when compiler
evidence proves the allocations do not escape.

## Current Evidence

Fresh report: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/report.json`.

Row:

- `hash_table_tetra`
  - Classification: `blocked by heap allocation`
  - Allocation artifact:
    `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/hash_table_tetra.alloc.json`
  - Bounds artifact:
    `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/hash_table_tetra.bounds.json`
  - Backend artifact:
    `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/hash_table_tetra.backend.json`

Allocation artifact summary:

- `allocation_count`: 2
- `storage_classes.Heap`: 2
- `runtime_paths.heap`: 2
- allocations: `keys`, `values`
- builtin: `core.make_i32`
- escape: `EscapesCallUnknown`
- reason: unknown call escape requires conservative heap fallback because the allocation is passed
  to a call without interprocedural escape facts.
- domain: `domain:process`

## Owner

The immediate owner is compiler escape/interprocedural summary evidence.

This is not primarily an allocator problem. The runtime allocator is doing what the current
allocation plan asks it to do. The allocation planner falls back to heap because it cannot prove
that `lookup(keys, values, n, key)` only reads the slices and does not store, return, transfer, or
otherwise escape them.

There is a secondary reporting gap: the report currently records `bytes_requested: 0` and
`bytes_reserved: 0` for `length_expr: n`, even though the benchmark source binds `let n: Int = 256`.
That reporting gap should not be confused with the heap-allocation owner.

## Proposed Slice

First acceptance target:

- add a conservative function summary for calls whose slice parameters are only read and not
  returned/stored/transferred;
- consume that summary in allocation planning so `keys` and `values` no longer become
  `EscapesCallUnknown` for this benchmark shape;
- keep heap fallback for unknown calls, calls that return a slice, global stores, actor/task
  transfers, unsafe exposure, and aggregate escapes.

Only after no-escape is proven should storage choice be revisited:

- fixed-size stack lowering if the length is proven constant and within stack policy;
- function-temp region if dynamic but scoped and region lowering is enabled;
- explicit heap fallback if neither is proven.

## Likely Files

- `compiler/internal/allocplan/plan.go`
- `compiler/internal/allocplan/plan_test.go`
- `compiler/internal/memoryfacts/fromplir/from_plir_summary.go`
- `compiler/internal/memoryfacts/fromplir/from_plir_allocplan.go`
- `compiler/internal/lower`
- `compiler/internal/semantics`
- `compiler/compiler_suite_test.go`
- `tools/internal/localbenchmarktier1/specs/tetra_sources.go`

## Tests First

Add or extend focused tests that fail before implementation:

- no-escape call summary for a read-only function receiving a slice;
- negative call-summary tests for return/global/actor/task/unsafe escape;
- allocation report test proving the old `EscapesCallUnknown` reason is gone only for the read-only
  hash-table shape;
- memory evidence test confirming the report still classifies heap/RSS truth honestly.

## Verification

Focused:

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-hash go test ./compiler/internal/allocplan/... ./compiler/internal/memoryfacts/... ./compiler/internal/semantics/... ./compiler -run 'Escape|Allocation|Call|Summary|Hash|Memory' -count=1
```

Benchmark:

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-hash go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-after-hash-track --iterations 3
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-hash go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-hash-track/report.json
```

## Nonclaims

- No zero-heap-for-all-programs claim.
- No allocator/RSS claim from an escape-summary change.
- No removal of heap fallback for unknown or unsafe calls.
