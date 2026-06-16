# Tetra Memory Zero-Heap MEM-8 MemoryBackend Evidence

Date: 2026-06-16

## Status

Complete for MEM-8.

## What Changed

- Allocation-plan rows now carry `memory_backend` evidence with schema
  `tetra.memory.backend-allocation.v1`.
- Runtime allocator evidence now distinguishes:
  - `small_heap`
  - `region`
  - `large_backend`
  - `none`
  - `external`
  - `conservative_heap`
  - `unknown`
- `memory_backend` evidence records:
  - `reserve_bytes`
  - `commit_bytes`
  - `release_bytes`
  - `footprint_current_bytes`
  - `footprint_peak_bytes`
  - backend operations such as `reserve`, `commit`, `release`, and
    `footprint`
  - `unsupported_reason` or `blocked_reason` when bytes are not claimed.
- Allocation report summaries now include:
  - `bytes_committed`
  - `bytes_released`
  - `memory_backend_classes`
  - `memory_backend_operations`
  - `memory_backend_evidence_classes`
- Tier 1 memory metadata now reports `bytes_committed` and `bytes_released`
  from allocation-report estimates instead of leaving committed bytes as
  unsupported.
- Tier 1 validation now reads each allocation report and rejects measured Tetra
  allocation rows without valid `memory_backend` evidence.
- RAM contract domain projection now preserves committed, released, current,
  and peak backend bytes when allocplan provides them.

## Fresh Benchmark Evidence

Fresh report:

```text
reports/benchmark-vnext-memory-baseline/tier1-after-memory-backend-evidence/report.json
```

Validator:

```text
GOCACHE=$(pwd)/.cache/go-build-mem8-validate go run ./tools/cmd/validate-local-benchmark-tier1 -report reports/benchmark-vnext-memory-baseline/tier1-after-memory-backend-evidence/report.json
```

Result: passed.

Report summary:

```text
tetra_rows=17
tetra_measured=17
tetra_bytes_committed_estimate=17
tetra_bytes_released_estimate=17
tetra_heap_reason_rows=5
allocation_rows_missing_memory_backend=0
```

Allocation-report backend classes in the fresh report:

```text
large_backend=2
none=6
region=1
small_heap=4
```

Allocation-report backend evidence classes:

```text
allocation_report_estimate=7
unsupported=6
```

Allocation-report backend operations:

```text
commit=7
footprint=7
release=7
reserve=7
```

The fresh Tier 1 report did not produce a `blocked` allocation row, but blocked
MemoryBackend semantics are covered by runtimeabi, allocplan, and Tier 1
validator tests.

## Tests

RED check was confirmed before implementation:

```text
GOCACHE=$(pwd)/.cache/go-build-mem8-red go test ./compiler/internal/runtimeabi ./compiler/internal/allocplan ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -run 'MemoryBackend|AllocatorEvidence|AllocationWithoutMemoryBackend|MemoryEvidenceFromAllocationReport|SmallHeapRuntimeAllocatorClass|PerCoreSmallHeapAllocatorEvidence|ExplicitIslandRuntimeAllocatorClass|FunctionTempRegion' -count=1
```

Expected failures showed missing `MemoryBackend`, `BytesCommitted`,
`BytesReleased`, and missing validator rejection for allocation rows without
`memory_backend`.

Focused tests passed:

```text
GOCACHE=$(pwd)/.cache/go-build-large-backend go test ./compiler/internal/runtimeabi ./compiler/internal/allocplan ./compiler ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -run 'Large|Mmap|MemoryBackend|Reserve|Commit|Release|Footprint|MemoryEvidence|AllocationWithoutMemoryBackend|RuntimeAllocator|FunctionTempRegion|ExplicitIsland' -count=1
```

Broader MEM-8/RAM-adjacent tests passed:

```text
GOCACHE=$(pwd)/.cache/go-build-mem8-ram go test ./compiler/internal/runtimeabi ./compiler/internal/allocplan ./compiler/internal/ramcontract ./tools/internal/ramvalidate ./tools/cmd/validate-memory-report ./compiler ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -run 'Memory|Backend|Domain|Commit|Release|Footprint|Allocation|RAM|Ram|Report' -count=1
```

## Nonclaims

- `bytes_committed` and `bytes_released` in this slice are allocation-report
  estimates, not runtime-measured allocator counters.
- This does not claim allocator performance superiority.
- This does not claim cross-target RSS parity.
- This does not implement actor memory domains or byte-based actor
  backpressure.
- This does not complete MEM-9 through MEM-12.

## Next

MEM-9: actor memory domains and byte-based backpressure.
