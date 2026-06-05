# Stable Generic Collections v1

Goal slice: P19.1 Stable Generic Collections.

Baseline: `tetra.truthful-performance-core.baseline.20260602.v1`.

Status: complete for the bounded source-level API, evidence report, and checked
P19.1 dry-run benchmark-equivalent artifact. The benchmark row is evidence
only, not performance parity.

## Scope

P19.1 promotes a narrow Tetra-source generic collection surface:

- `lib.core.collections.Vec<T>` as a view over caller-owned `[]T` storage.
- `lib.core.collections.HashMap<K,V>` as a view over caller-owned parallel
  key/value slices.
- Generic operations that can be statically monomorphized from value arguments.
- Concrete lookup specializations for `HashMap<Int,Int>` and
  `HashMap<UInt8,Int>`.
- Allocation evidence through the existing slice allocation-plan reports.

## Implemented Rows

| Row | Status | Evidence |
|---|---|---|
| Tetra-source API | `implemented_narrow` | `Vec<T>`, `HashMap<K,V>`, caller-owned slices |
| Generic value representation | `implemented_narrow` | `genericTypeName`, `mangleGenericName`, `[]T` substitution |
| Monomorphized operations | `implemented_narrow` | `vec_from_slice<T>`, `hash_map_from_slices<K,V>`, concrete before lowering |
| Common specializations | `implemented_narrow` | `hash_map_get_i32_i32_or`, `hash_map_get_u8_i32_or` |
| Allocation reports | `evidence_only` | `core.make_*` / `core.island_make_*` allocation-plan paths |
| Benchmark gate | `evidence_only` | `p19.1_generic_collections` dry-run hash-table Tetra/C++/Rust equivalent artifact; no parity claim |

## Code Changes

- `lib/core/collections.tetra` adds generic source views and helpers.
- `compiler/internal/semantics/generics.go` infers generic function type
  arguments from monomorphized generic struct parameter names.
- `compiler/internal/stdlibrt/stable_generic_collections.go` adds
  `tetra.stdlib.generic_collections.v1` evidence rows and validator.
- Focused tests cover the source API, generic struct parameter inference, and
  fake-claim rejection.
- `tools/cmd/truth-bench-harness` accepts the narrow
  `p19.1_generic_collections` scope and rejects missing Tetra/C++/Rust rows,
  mismatched algorithm/input metadata, fake C++/Rust parity claims, and fake
  official benchmark claims.
- `reports/stable-generic-collections-v1/benchmarks/generic-collections-hash-table-manifest.json`
  and
  `reports/stable-generic-collections-v1/benchmarks/generic-collections-hash-table-report.json`
  record a checked dry-run hash-table equivalent artifact with Tetra proof,
  allocation, bounds, and performance report paths.

## Verification Evidence

RED evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/stdlibrt ./compiler/tests/semantics -run 'TestStableGenericCollectionsCoverageCoversP19PlanList|TestStableGenericCollectionsCoverageRejectsFakeClaims|TestGenericFunctionInfersThroughGenericStructParameter|TestStableGenericCollectionSourceAPIMonomorphizesVecAndHashMap' -count=1
```

Initial result: failed for the intended reasons:
`StableGenericCollectionsCoverage`, `ValidateStableGenericCollectionsCoverage`,
and the `lib.core.collections.Vec` source API did not exist.

Focused GREEN evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/stdlibrt ./compiler/tests/semantics -run 'TestStableGenericCollectionsCoverageCoversP19PlanList|TestStableGenericCollectionsCoverageRejectsFakeClaims|TestGenericFunctionInfersThroughGenericStructParameter|TestStableGenericCollectionSourceAPIMonomorphizesVecAndHashMap' -count=1
```

Result: pass.

Benchmark gate GREEN evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./tools/cmd/truth-bench-harness ./compiler/internal/stdlibrt -run 'TestP19GenericCollectionsScopeRequiresTetraCppRustHashTableEquivalents|TestP19GenericCollectionsScopeRejectsMissingEquivalenceAndParityClaim|TestStableGenericCollectionsCoverageCoversP19PlanList|TestStableGenericCollectionsCoverageRejectsFakeClaims' -count=1
```

Result: pass.

Artifact generation:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/truth-bench-harness --manifest reports/stable-generic-collections-v1/benchmarks/generic-collections-hash-table-manifest.json --out reports/stable-generic-collections-v1/benchmarks/generic-collections-hash-table-report.json
```

Result: pass; generated report schema `tetra.truth.benchmark.v1`, scope
`p19.1_generic_collections`, three rows, and `ran=false`.

Equivalent source checks:

```bash
./tetra check benchmarks/generic_collections/hash_table.tetra
clang++ -O3 benchmarks/generic_collections/hash_table.cpp -o .cache/p19.1-generic-collections-bench/hash_table_cpp
rustc -C opt-level=3 benchmarks/generic_collections/hash_table.rs -o .cache/p19.1-generic-collections-bench/hash_table_rust
.cache/p19.1-generic-collections-bench/hash_table_cpp
.cache/p19.1-generic-collections-bench/hash_table_rust
```

Result: pass. Both native equivalents print `530944` and exit `0`; this is a
source-equivalence check, not a harness runtime measurement or speed claim.

## Non-Claims

- This is not an allocator-backed production `Vec<T>`/`HashMap<K,V>` runtime.
- This does not add generic hashing or equality protocols.
- This does not claim resizing, collision handling, sorting, or iterator
  objects.
- This does not claim C++/Rust parity, broad stdlib completeness, or an
  official benchmark result.
- The checked benchmark artifact is dry-run evidence only; it records source
  equivalence and report shape, not a runtime measurement or speed comparison.
