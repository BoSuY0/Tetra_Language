# Tetra Memory Zero-Heap MEM-7 Heap Reason Codes

Date: 2026-06-16

## Status

Complete for MEM-7.

## What Changed

- Allocation-plan rows now carry machine-readable reason-code evidence:
  - `reason_codes`
  - `heap_reason_codes`
- Heap allocation rows are invalid without `heap_reason_codes`.
- The reason-code vocabulary is explicit:
  - `heap.required_escape_return`
  - `heap.required_unknown_call`
  - `heap.required_actor_boundary`
  - `heap.required_task_boundary`
  - `heap.required_dynamic_lifetime`
  - `heap.required_large_object`
  - `heap.required_ffi_external`
  - `heap.required_backend_lowering_unavailable`
  - `heap.required_region_lowering_unavailable`
- Tier 1 benchmark metadata now mirrors allocation-report `summary.heap_reason_codes` as
  `tetra_metadata.heap_reason_codes`.
- Tier 1 validator now checks the allocation report itself: every heap allocation row must have
  known `heap_reason_codes`, those codes must also appear in `reason_codes`, and metadata must match
  the allocation report summary.
- RAM contract rows and heap blocker rows now preserve heap reason codes.
- Paired memory-report validation rejects heap allocation artifacts without heap reason codes.
- Legacy `BuildAllocReport` heap rows also carry reason-code fields.

## Fresh Benchmark Evidence

Fresh report:

```text
reports/benchmark-vnext-memory-baseline/tier1-after-heap-reason-codes/report.json
```

Validator:

```text
GOCACHE=$(pwd)/.cache/go-build-mem7-validate go run ./tools/cmd/validate-local-benchmark-tier1 -report reports/benchmark-vnext-memory-baseline/tier1-after-heap-reason-codes/report.json
```

Result: passed.

Report summary:

```text
tetra_rows=17
tetra_rows_with_heap_reason_metadata=5
heap_allocation_rows=5
heap_reason_codes=heap.required_large_object,heap.required_unknown_call
```

Rows with heap allocations:

```text
slice_sum_tetra                         1  heap.required_large_object
bounds_check_loops_tetra                1  heap.required_large_object
json_parse_stringify_tetra              1  heap.required_unknown_call
http_plaintext_json_tetra               2  heap.required_unknown_call
postgresql_single_multiple_update_tetra 1  heap.required_unknown_call
```

Direct allocation-report audit:

```text
slice_sum_tetra.alloc.json                         heap rows 1  missing codes 0
bounds_check_loops_tetra.alloc.json                heap rows 1  missing codes 0
json_parse_stringify_tetra.alloc.json              heap rows 1  missing codes 0
http_plaintext_json_tetra.alloc.json               heap rows 2  missing codes 0
postgresql_single_multiple_update_tetra.alloc.json heap rows 1  missing codes 0
```

## Tests

Focused tests passed:

```text
GOCACHE=$(pwd)/.cache/go-build-mem7-focused go test ./compiler/internal/allocplan ./compiler/internal/ramcontract ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 ./tools/cmd/validate-memory-report ./compiler -run 'HeapReason|Allocation|Alloc|RAMContract|HeapBlocker|MemoryReportWithAllocReport|ValidateReportAcceptsCompleteP25Tier1Matrix|ValidateReportRejectsHeapAllocationWithoutReasonCodes|ValidateReportRejectsHeapReasonMetadataMismatch|CollectTetraMetadataAttachesHeapReasonCodes' -count=1
```

Result: passed.

## Nonclaims

- This does not remove the remaining heap allocations.
- This does not claim zero heap for excluded or complex rows.
- This does not merge heap, RSS, domain bytes, or allocation estimates.
- This does not complete MEM-8 allocator classes or MemoryBackend evidence.

## Next

MEM-8: runtime allocator classes and MemoryBackend evidence must distinguish small heap, region,
large backend, reserve, commit, release, footprint, and unsupported/blocked states.
