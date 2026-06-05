# Vectorization v1

Status: P17.3 progress audit for the Ideal Master Plan.

## Summary

`compiler/internal/opt.VectorizationCoverage()` emits schema
`tetra.optimizer.vectorization.v1` for the P17.3 initial target list. The
current bounded slice recognizes proof-tagged `sum []i32` as a
range-proof-backed candidate, proves noalias is not required for this read-only
reduction, lowers it to an abstract safe-unaligned `i32x4` Machine IR plan with
a structural `vector_can_load_i32x4` lane guard, and emits a linux-x64 native
SIMD path with scalar tail handling and stack-fallback differential validation.
It also recognizes proof-tagged `copy []u8` as a range-proof-backed candidate,
requires source/dest disjoint owned-copy noalias evidence, lowers it to an
abstract safe-unaligned `u8x16` Machine IR plan with scalar tail and scalar
fallback, and emits a linux-x64 native SIMD copy path with stack-fallback
differential validation. It now recognizes a proof-tagged simple map over
`[]i32` that adds constant `1` in place, proves noalias is not required for a
single mutable slice in-place map, and lowers that shape to an abstract
safe-unaligned `i32x4` Machine IR plan with scalar tail handling and a
`scalar-i32-map` fallback, then emits a linux-x64 native SIMD path with
stack-fallback differential validation.

It also recognizes a proof-tagged `memset_zero_u8` helper as a zero-fill
`memset` candidate, lowers it to an abstract safe-unaligned `u8x16` Machine IR
store plan with scalar tail handling and a `scalar-u8-memset-zero` fallback,
and emits a linux-x64 native SIMD zero-fill path with stack-fallback
differential validation. The `memcpy` helper portion is bounded to the existing
proof-tagged `copy []u8` native SIMD evidence.

This audit does not claim broad SIMD auto-vectorization, broader map-shape
vectorization, arbitrary non-zero `memset`, overlapping `memcpy`, libc/runtime
helper lowering, throughput, or C/Rust parity.

## Coverage

| Target | Status | Decision | Evidence | Boundary |
| --- | --- | --- | --- | --- |
| `sum []i32` | `implemented_narrow` | `vectorized` | `CoreHotLoopShapeEvidence`, `ScalarI32SliceSumLoopPlanFromStackIR`, `VectorI32x4SliceSumLoopPlanFromStackIR`, `TestVectorI32x4SliceSumLoopFromStackIRUsesSafeUnalignedTailAndScalarFallback`, `emitVectorSliceSumRegisterFunction`, `TestCodegenObjectLinuxX64UsesVectorSliceSumPathForProofLoop`, and `TestCodegenObjectLinuxX64VectorSliceSumMatchesStackFallbackWithTail` | Proof-tagged range proof exists; noalias is not required because this is a read-only reduction with no slice memory stores; abstract `vector-i32x4-slice-sum-plan` records `vector_can_load_i32x4`, safe unaligned vector load, scalar tail handling, and scalar fallback; linux-x64 native SIMD codegen emits executable `pxor`/`movdqu`/`paddd`/`pshufd`/`movd` lowering and the tail run matches stack fallback. Scope is limited to proof-tagged step=1 `sum []i32` on linux-x64 machine paths. |
| `copy []u8` | `implemented_narrow` | `vectorized` | `addCopyLoopRangeProof`, `TestFromCheckedProgramRecordsBorrowCopyFacts`, `VectorU8x16CopyLoopPlanFromStackIR`, `TestVectorU8x16CopyLoopFromStackIRRequiresRangeNoAliasSafeUnalignedTailAndFallback`, `emitVectorCopyU8RegisterFunction`, `TestCodegenObjectLinuxX64UsesVectorCopyU8PathForProofLoop`, and `TestCodegenObjectLinuxX64VectorCopyU8MatchesStackFallbackWithTail` | Proof-tagged copy-loop range proof exists; noalias is required and bounded to source/dest disjoint owned-copy-result evidence; abstract `vector-u8x16-copy-plan` records `vector_can_copy_u8x16`, safe unaligned vector load/store, scalar tail handling, and scalar fallback; linux-x64 native SIMD codegen emits executable safe-unaligned `movdqu` load/store lowering and the 19-byte tail run matches stack fallback. Scope is limited to proof-tagged `copy []u8` on linux-x64 machine paths and excludes checked/no-proof copy and overlapping slices. |
| simple map over `[]i32` | `implemented_narrow` | `vectorized` | `VectorI32x4MapAddConstPlanFromStackIR`, `TestVectorI32x4MapAddConstFromStackIRRequiresRangeSafeUnalignedTailAndFallback`, `emitVectorMapI32AddConstRegisterFunction`, `TestCodegenObjectLinuxX64UsesVectorMapI32AddConstPathForProofLoop`, and `TestCodegenObjectLinuxX64VectorMapI32AddConstMatchesStackFallbackWithTail` | Proof-tagged map-loop range proof exists for an in-place `xs[i] = xs[i] + 1` shape; noalias is not required because there is one mutable slice base; abstract `vector-i32x4-map-add-const-plan` records `vector_can_map_i32x4`, safe unaligned vector load/store, scalar tail handling, and a `scalar-i32-map` fallback; linux-x64 native SIMD codegen emits executable `movd`/`pshufd`/`movdqu`/`paddd` lowering plus scalar tail handling and the 7-element tail run matches stack fallback. Scope is limited to proof-tagged in-place add-constant-1 `map []i32` on linux-x64 machine paths and excludes checked/no-proof map loops and broader map shapes. |
| `memset/memcpy helpers` | `implemented_narrow` | `vectorized` | `VectorU8x16MemsetZeroPlanFromStackIR`, `TestVectorU8x16MemsetZeroHelperFromStackIRRequiresRangeSafeUnalignedTailAndFallback`, `emitVectorMemsetZeroU8RegisterFunction`, `TestCodegenObjectLinuxX64UsesVectorMemsetZeroU8PathForProofHelper`, `TestCodegenObjectLinuxX64VectorMemsetZeroU8MatchesStackFallbackWithTail`, and existing `copy []u8` vector evidence for the `memcpy` helper portion | Proof-tagged `memset_zero_u8` has memset-loop range proof; noalias is not required for a single mutable slice zero-fill helper; abstract `vector-u8x16-memset-zero-plan` records `vector_can_memset_u8x16`, safe unaligned vector store, scalar tail handling, and `scalar-u8-memset-zero` fallback; linux-x64 native SIMD codegen emits executable `pxor`/`movdqu` zero-store lowering and the 19-byte tail run matches stack fallback. `memcpy` helper evidence is limited to the existing proof-tagged source/dest-disjoint `copy []u8` path. Scope excludes arbitrary non-zero `memset`, checked/no-proof helpers, overlapping `memcpy`, and libc/runtime helper lowering. |

## Non-Claims

- No broad SIMD auto-vectorization or target-independent native vector backend
  is claimed by this slice.
- No throughput or C/Rust performance parity claim is made.
- No vector path may be selected without range proof, required noalias or
  noalias-not-required evidence, alignment or safe unaligned-vector evidence,
  tail handling, scalar fallback, native codegen, and translation/differential
  validation.
- No checked/no-proof `sum []i32`, constant-stride sum, checked/no-proof
  `copy []u8`, overlapping-slice copy, checked/no-proof map,
  broader map-shape vectorization, arbitrary non-zero `memset`, overlapping
  `memcpy`, checked/no-proof helper, or libc/runtime helper lowering claim is
  made.
