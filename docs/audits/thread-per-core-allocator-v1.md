# Thread-local / Per-core Allocator v1 Audit

Status: P15.2 evidence audit for the Ideal Master Plan.

## Summary

The P15.2 slice defines a per-core small heap allocator ABI and report model for
safe slice allocations. Small constant-size safe-slice heap rows now report
`runtime_path: per_core_small_heap`, deterministic allocator scope `core:0`,
`allocator_reuse_policy: same_core_same_size_class_free_list`, and chunk size
evidence. The runtime ABI model includes generation-guarded handles so same
core/same class reuse is explicit and stale or double frees are rejected.

The memory production smoke gate builds a generated benchmark, reads the
schema-v2 allocation summary, requires `per_core_small_heap` rows with the reuse
policy, and records an estimated syscall reduction from mmap-per-allocation to
64 KiB chunk refills.

## Changed Areas

- `compiler/internal/runtimeabi/small_heap.go`
- `compiler/internal/runtimeabi/small_heap_test.go`
- `compiler/internal/runtimeabi/allocation_contract.go`
- `compiler/internal/allocplan/plan.go`
- `compiler/internal/allocplan/plan_test.go`
- `tools/cmd/memory-production-smoke/main.go`
- `tools/cmd/memory-production-smoke/main_test.go`
- `tools/validators/memoryprod/report.go`
- `tools/validators/memoryprod/report_test.go`
- `tools/validators/postv04prod/report_test.go`
- `docs/design/runtime_allocation_contract.md`
- `docs/design/storage_classes.md`
- `docs/design/explainable_one_build.md`
- `docs/design/truthful_intent_architecture.md`
- `docs/design/allocation_planner_lowering.md`
- `tools/validators/memoryprod/README.md`
- `reports/thread-per-core-allocator-v1/memory-production-linux-x64.json`

## Evidence

| Check | Result |
| --- | --- |
| Focused runtime/planner/memory validator tests | pass |
| Compiler allocation report and feature registry slice | pass |
| Memory production smoke report generation | pass |
| Memory production report validation | pass |

## Memory Benchmark Evidence

`reports/thread-per-core-allocator-v1/memory-production-linux-x64.json`
contains:

- benchmark name: `small heap allocation syscall reduction`;
- baseline value: `64`;
- measured value: `1`;
- improvement ratio: `64`;
- evidence text naming `per_core_small_heap` and
  `same_core_same_size_class_free_list`.

## Boundaries

Unsafe `core.alloc_bytes` remains conservative until P15.3 raw pointer bounds
metadata. The P15.2 report model does not claim a tracing GC, broad safe-slice
free API, non-Linux-x64 allocator production behavior, or official benchmark
leadership.
