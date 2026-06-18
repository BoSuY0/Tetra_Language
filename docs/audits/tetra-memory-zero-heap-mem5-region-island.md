# Tetra Memory Zero-Heap MEM-5 Region/Island Closure

Status: complete for MEM-5.
Date: 2026-06-16.

## Scope

MEM-5 covers only the `region/island allocation` Tier 1 blocker.

It does not claim:

- production OS memory usage;
- zero RSS;
- universal zero heap;
- implicit region inference;
- actor memory domains;
- final memory optimization completion.

## Reproduction

Fresh repro command:

```sh
GOCACHE=$(pwd)/.cache/go-build-region-island-repro go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-region-island-repro --iterations 1
```

Fresh repro result:

- `region_island_allocation_tetra` status: `build_failed`.
- category classification: `blocked by missing feature`.
- build stderr:

```text
allocation lowering validation: p25.region_island_allocation.main instruction 14 explicit island allocation "xs" use after free via operands of island:p25.region_island_allocation.main:10
```

## Root Cause

Boundary: lowered IR -> allocation lifetime validator.

The scoped island workload creates and frees an island inside a loop:

```tetra
while r < 256:
    island(256) as isl:
        var xs: []i32 = core.island_make_i32(isl, 16)
```

The validator tags an explicit island handle by function and `IRIslandNew`
instruction index, for example:

```text
island:p25.region_island_allocation.main:10
```

On loop back, the same `IRIslandNew` instruction creates a fresh runtime island
handle, but the validator still had that tag marked as freed from the previous
iteration. The next `IRIslandMakeSliceI32` therefore saw a false
use-after-free.

## Fix

When validation steps over `IRIslandNew`, it now clears the freed marker for the
fresh handle tag before pushing that handle onto the abstract stack.

Regression test:

```text
TestValidateAllocationLoweringAcceptsExplicitIslandRecreatedAtLoopSite
```

The existing explicit-island use-after-free and double-free tests still pass.

## Fresh Result

Fresh report:

```text
reports/benchmark-vnext-memory-baseline/tier1-after-region-island-track/report.json
```

Result:

- `region_island_allocation_tetra` is now `measured`.
- `tetra_metadata.heap_allocations == 0`.
- `heap_alloc_bytes.evidence_class == runtime_measured`.
- runtime heap sidecars report current/peak/total/count as `0`.
- allocation report shows:
  - `storage_classes.ExplicitIsland == 1`;
  - `actual_lowering_storage_classes.ExplicitIsland == 1`;
  - `runtime_paths.explicit_island == 1`;
  - `allocator_classes.region_bump_16 == 1`;
  - `totals.heap == 0`;
  - domain kind `island`, requested/reserved bytes `64`.

The category is no longer `blocked by missing feature`; it is now
`blocked by fallback backend`, which is a later benchmark/backend blocker.

## Verification

```sh
GOCACHE=$(pwd)/.cache/go-build-region-island-repro go test ./tools/cmd/local-benchmark-tier1 -run 'Region|Island|BuildSpecs' -count=1
GOCACHE=$(pwd)/.cache/go-build-region-island-repro go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-region-island-repro --iterations 1
GOCACHE=$(pwd)/.cache/go-build-region-island-red go test ./compiler/internal/validation -run TestValidateAllocationLoweringAcceptsExplicitIslandRecreatedAtLoopSite -count=1
GOCACHE=$(pwd)/.cache/go-build-region-island go test ./compiler/internal/allocplan/... ./compiler/internal/lower ./compiler/internal/validation/... ./compiler/internal/runtimeabi ./compiler -run 'Island|Region|Allocation|Domain|Lower|Translation' -count=1
GOCACHE=$(pwd)/.cache/go-build-region-island go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-after-region-island-track --iterations 3
GOCACHE=$(pwd)/.cache/go-build-region-island go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-region-island-track/report.json
GOCACHE=$(pwd)/.cache/go-build-region-island go test ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -run 'Region|Island|ValidateReport|MemoryEvidence|BuildSpecs' -count=1
```

All commands above passed after the fix, except the intentional RED run before
the fix:

```text
TestValidateAllocationLoweringAcceptsExplicitIslandRecreatedAtLoopSite
failed with false use-after-free via island:main:2
```

## Remaining Risk

MEM-5 does not close the fallback backend blocker. It only closes the
build-failed/missing-feature blocker for the explicit island allocation row.
