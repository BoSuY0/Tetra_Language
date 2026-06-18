# Tetra Memory Zero-Heap MEM-2 Hash Re-Evaluation

Status: MEM-2 fresh hash re-evaluation after MEM-1.

This note records the current `hash_table_tetra` state after the first
call-aware stack-lowering slice. It is local benchmark evidence only.

## Source Report

- Report:
  `reports/benchmark-vnext-memory-baseline/tier1-after-call-aware-stack-track/report.json`
- Generated at: `2026-06-16T13:02:39Z`
- Git commit:
  `95bfd4a887bab5032437cb22494d034e82ae6d35`
- Scope: `tetra.local_benchmark_tier1.v1`

## Validation

Validated locally:

```sh
GOCACHE=$(pwd)/.cache/go-build-call-aware-hash go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-after-call-aware-stack-track --iterations 3
GOCACHE=$(pwd)/.cache/go-build-call-aware-hash go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-call-aware-stack-track/report.json
```

Both commands exited 0.

## Report Shape

- categories: 17
- rows: 68
- Tetra rows: 17
- measured Tetra rows: 16
- build-failed Tetra rows: 1
- runtime-measured heap rows: 16
- blocked heap rows: 1
- runtime-measured RSS peak rows: 16

## `hash_table_tetra` Current State

Category classification:

- classification: `blocked by heap allocation`
- classification reason:
  `Tetra allocation report records 2 heap allocations.`

Tetra row:

- status: `measured`
- backend path: `fallback`
- backend blockers:
  - `unsupported_control_flow`
  - `unsupported_effect_runtime_call`
- bounds left: 4
- heap allocations: 2
- perf blockers:
  - `allocation.local_call_heap_fallback`
  - `inline.code_size_budget`
- runtime heap:
  - current bytes: 2048
  - peak bytes: 2048
  - total alloc bytes: 2048
  - allocation count: 2
- runtime heap source:
  `reports/benchmark-vnext-memory-baseline/tier1-after-call-aware-stack-track/artifacts/heap-telemetry/hash_table_tetra/iteration-01.heap.json`
- RSS peak: 13176832 bytes
- domain evidence: `allocation_report_estimate`
- domain: `domain:process`

Allocation report:

```text
allocation_count: 2
storage_classes: Heap = 2
actual_lowering_storage_classes: Heap = 2
runtime_paths: heap = 2
```

`keys` and `values` both have:

```text
length_expr: n
length_status: runtime_guarded
escape: NoEscape
planned_storage: Heap
actual_lowering_storage: Heap
reason: no-escape allocation crosses a local call boundary but is not fixed-small
```

## Exact Remaining Blocker

The first call-aware slice works for fixed-small allocations, but this benchmark
does not yet expose `n` as a constant allocation length.

Source:

```tetra
let n: Int = 256
var keys: []i32 = core.make_i32(n)
var values: []i32 = core.make_i32(n)
```

PLIR still records both allocation intents with:

```text
length_expr: n
length_const_known: false
```

The allocation planner therefore cannot compute:

```text
256 * 4 bytes = 1024 bytes
```

and cannot apply the fixed-small read-only-local-call Stack path.

## MEM-2 Decision

MEM-2 is satisfied as a re-evaluation step: `hash_table_tetra` did not leave
heap-blocked status, and the exact remaining blocker is now narrowed to missing
constant-length propagation from immutable local constants into allocation
intents.

Next implementation target before claiming hash-table zero heap:

```text
let n: Int = 256
make_i32(n)
=> AllocIntent.LengthConstKnown = true, LengthConst = 256
```

This must preserve guardrails:

- mutable or reassigned lengths stay runtime-guarded;
- parameter lengths stay runtime-guarded;
- unknown expressions stay runtime-guarded;
- overflow/negative checks stay intact;
- actor/task/unsafe/global/return escapes stay conservative.
