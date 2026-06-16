# Tetra Memory Zero-Heap Const-Length Hash Follow-Up

Status: completed local follow-up after MEM-2.

This note records the implementation and evidence that closed the exact
`hash_table_tetra` heap blocker found in MEM-2. It is local benchmark evidence
only.

## Source Report

- Report:
  `reports/benchmark-vnext-memory-baseline/tier1-after-const-length-track-final/report.json`
- Generated at: `2026-06-16T13:18:18Z`
- Git commit:
  `95bfd4a887bab5032437cb22494d034e82ae6d35`
- Scope: `tetra.local_benchmark_tier1.v1`

## Implementation

The compiler now propagates immutable local integer constants into allocation
length evidence:

```tetra
let n: Int = 256
var keys: []i32 = core.make_i32(n)
```

now records:

```text
LengthConstKnown = true
LengthConst = 256
```

The lowerer now uses the same immutable-local constant evidence when emitting
stack slice IR, so the allocation plan and lowered IR agree.

Guardrails preserved:

- mutable local lengths remain runtime guarded;
- unknown expressions remain runtime guarded;
- stack lowering still requires allocation-plan `ActualLoweringStorage=Stack`;
- runtime heap telemetry reports stack-backed allocations as zero Tetra heap.

## Validation

Validated locally:

```sh
GOCACHE=$(pwd)/.cache/go-build-const-length-final go test ./compiler/internal/plir ./compiler/internal/allocplan ./compiler/internal/lower ./compiler/internal/validation ./compiler ./tools/cmd/local-benchmark-tier1 -run 'AllocationLength|ImmutableLength|MutableLocal|ReadOnly|Call|Stack|Hash|AllocReport|ExplainReports|AllocationLowering|RuntimeHeapTelemetry|ResolvedLocalCallHeapFallback' -count=1
GOCACHE=$(pwd)/.cache/go-build-const-length-final-report go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-after-const-length-track-final --iterations 3
GOCACHE=$(pwd)/.cache/go-build-const-length-final-report go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-const-length-track-final/report.json
```

All commands exited 0.

## Report Shape

- categories: 17
- rows: 68
- Tetra rows: 17
- measured Tetra rows: 16
- build-failed Tetra rows: 1

## `hash_table_tetra` Current State

Category classification:

- classification: `blocked by fallback backend`
- classification reason:
  `Tetra backend report selected stack/fallback path for at least one function.`

Tetra row:

- status: `measured`
- heap allocations: 0
- bounds left: 4
- backend path: `fallback`
- backend blockers:
  - `unsupported_control_flow`
  - `unsupported_effect_runtime_call`
- perf blockers:
  - `inline.code_size_budget`
- runtime heap:
  - current bytes: 0
  - peak bytes: 0
  - total alloc bytes: 0
  - allocation count: 0
- runtime heap source:
  `reports/benchmark-vnext-memory-baseline/tier1-after-const-length-track-final/artifacts/heap-telemetry/hash_table_tetra/iteration-01.heap.json`
- RSS peak: 11792384 bytes
- domain evidence: `allocation_report_estimate`
- domain: `domain:process`

Allocation report:

```text
allocation_count: 2
storage_classes: Stack = 2
actual_lowering_storage_classes: Stack = 2
runtime_paths: stack_frame = 2
heap: 0
stack: 2
bytes_requested: 2048
bytes_reserved: 2048
```

`keys` and `values` both have:

```text
length_expr: n
length_status: normal_allocation
escape: NoEscape
planned_storage: Stack
actual_lowering_storage: Stack
runtime_path: stack_frame
byte_size: 1024
reason: fixed_small_read_only_local_call_no_escape
```

## Decision

The MEM-2 exact blocker is closed for `hash_table_tetra`: the row no longer has
Tetra runtime heap allocation. The remaining category blocker is backend
fallback, not heap allocation.

Next target: MEM-3 zero-heap gates. The validator should make this kind of
zero-heap result enforceable instead of relying on manual audit.
