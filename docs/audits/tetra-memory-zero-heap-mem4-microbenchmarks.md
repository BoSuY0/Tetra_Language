# MEM-4 Zero-Heap Microbenchmarks Audit

Date: 2026-06-16.
Scope: MEM-4 of
`docs/plan/2026-06-16-tetra-memory-zero-heap-optimization-plan.md`.

## Result

MEM-4 is implemented as a dedicated Tetra-only zero-heap microbenchmark suite
outside the Tier 1 P20 comparable matrix.

Implementation:

```text
tools/internal/zeroheapbench
tools/cmd/local-benchmark-zero-heap
tools/cmd/validate-local-benchmark-zero-heap
```

Policy placement:

```text
docs/spec/zero_heap_benchmark_policy.md
```

## Why Outside Tier 1

Tier 1 is a P20-style comparable matrix with four required languages:

```text
tetra
c
cpp
rust
```

Adding Tetra-only compiler guardrail rows directly to Tier 1 would distort that
matrix and could imply a cross-language benchmark claim. The zero-heap
microbenchmarks are therefore local Tetra optimizer/runtime guardrails, not
performance comparisons.

## Current Microbenchmarks

The dedicated categories are:

```text
zero heap fixed local array sum
zero heap read-only local call slice
zero heap small struct copy
zero heap borrowed view sum
zero heap copy eliminated unused
```

Each category has one Tetra zero-heap spec with:

- `Language == "tetra"`;
- `BuildArgs` containing `tetra build --target linux-x64 --explain`;
- source path under `artifacts/src/zero_heap`;
- a Tetra source snippet focused on one zero-heap compiler/runtime path.

## Guardrail

`TestZeroHeapMicrobenchSpecsStayOutsideTier1Matrix` ensures:

- the suite is not empty;
- there is exactly one Tetra spec per dedicated category;
- no dedicated category appears in `requiredP20Categories`;
- source and build metadata exist for every spec.

## Verification

RED confirmed:

```sh
GOCACHE=$(pwd)/.cache/go-build-zero-heap-mem4-red go test ./tools/cmd/local-benchmark-tier1 -run 'ZeroHeap|BuildSpecs' -count=1
GOCACHE=$(pwd)/.cache/go-build-zero-heap-standalone-red go test ./tools/cmd/validate-local-benchmark-zero-heap -run 'ValidateReport' -count=1
```

The first RED failed on missing zero-heap spec symbols. The second RED failed
on missing `ValidateReportBytes` in the standalone validator package.

Passed:

```sh
GOCACHE=$(pwd)/.cache/go-build-zero-heap-mem4-green go test ./tools/cmd/local-benchmark-tier1 -run 'ZeroHeap|BuildSpecs' -count=1
GOCACHE=$(pwd)/.cache/go-build-zero-heap-mem4-final go test ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -run 'ZeroHeap|BuildSpecs|ValidateReport|MemoryEvidence' -count=1
GOCACHE=$(pwd)/.cache/go-build-zero-heap-mem4-final go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-after-zero-heap-microbenchmarks --iterations 3
GOCACHE=$(pwd)/.cache/go-build-zero-heap-mem4-final go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-zero-heap-microbenchmarks/report.json
GOCACHE=$(pwd)/.cache/go-build-zero-heap-standalone-run go test ./tools/internal/zeroheapbench ./tools/cmd/local-benchmark-zero-heap ./tools/cmd/validate-local-benchmark-zero-heap ./tools/cmd/local-benchmark-tier1 -run 'ZeroHeap|BuildSpecs|ValidateReport' -count=1
GOCACHE=$(pwd)/.cache/go-build-zero-heap-standalone-run go run ./tools/cmd/local-benchmark-zero-heap --out-dir reports/benchmark-vnext-memory-baseline/zero-heap-microbenchmarks --iterations 3
GOCACHE=$(pwd)/.cache/go-build-zero-heap-standalone-run go run ./tools/cmd/validate-local-benchmark-zero-heap --report reports/benchmark-vnext-memory-baseline/zero-heap-microbenchmarks/report.json
```

Fresh Tier 1 report after MEM-4 still has 17 categories and no category whose
name starts with `zero heap`.

Fresh standalone zero-heap report:

```text
reports/benchmark-vnext-memory-baseline/zero-heap-microbenchmarks/report.json
```

All five rows are `measured` and report:

```text
heap_alloc_bytes.total_alloc_bytes == 0
heap_alloc_bytes.allocation_count == 0
runtime heap sidecar heap_current/peak/total/count == 0
```
