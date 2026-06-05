# Memory Production Core v1 Baseline

Status: MPC-0 baseline for the 2026-06-03 Memory Production Core v1 plan.

Core claim target: sound for the supported safe surface, conservative for
unknown unsafe memory, and validated for every claimed lowering. This audit does
not claim perfect memory, full Rust-like borrow checker parity, arbitrary unsafe
pointer safety, full target parity, or fastest-language performance.

## Baseline Table

| Area | Status | Current claim | Evidence |
| --- | --- | --- | --- |
| safe representation metadata | complete_narrow_slice | User-visible slice/String `ptr`/`len` metadata is not assignable supported safe state, including direct, nested, generic, optional/enum payload, inout, and index-through-metadata assignment targets. Broader future metadata names are rejected as representation metadata when they appear on collection assignment paths, but no full Rust-like lifetime model is claimed. | `docs/spec/ownership_v1.md`, `docs/design/truthful_safe_values.md`, `compiler/internal/semantics/representation_metadata_test.go`, `compiler/tests/semantics/slice_view_test.go`, `compiler/tests/semantics/string_metadata_test.go` |
| safe slice/String views | complete_narrow_slice | `window`, `prefix`, `suffix`, `borrow`, `copy`, and `copy_into` exist for the supported slice/String byte-view surface with checked construction and PLIR facts. | `docs/design/provenance_lifetime_ir.md`, `compiler/tests/semantics/slice_view_test.go`, `compiler/tests/semantics/string_view_test.go` |
| borrow/copy/copy_into | complete_narrow_slice | Borrowed views preserve source provenance, `copy()` creates owned allocation intent, and `copy_into(dst)` uses caller-owned destination storage. MPC-4 memory reports project bounded `borrow_owner`, `borrow_source_fact_id`, `copy_owned`, `copy_source_fact_id`, and `copy_into_destination_fact_id` rows without claiming named lifetimes. | `docs/design/provenance_lifetime_ir.md`, `docs/design/allocation_planner_lowering.md`, `docs/spec/memory_report_schema_v1.md` |
| borrowed return syntax | complete_narrow_slice | Borrowed returns are supported for the documented safe slice/String byte views. Named lifetimes and arbitrary borrowed aggregate returns are not claimed. | `docs/design/provenance_lifetime_ir.md`, safe view lifetime tests |
| hidden borrowed aggregate escape diagnostics | complete_narrow_slice | Recursive escape diagnostics exist for supported structs, enums, optionals, and generic wrappers carrying borrowed views. | `compiler/tests/safety/safety_diagnostics_test.go`, ownership marker tests |
| allocation length contract | complete | `make_*` and island allocation lengths classify zero, negative, overflow, and dynamic guarded lengths before storage selection. | `docs/design/allocation_planner_lowering.md`, `docs/spec/runtime_abi.md` |
| raw pointer bounds metadata | complete_narrow_slice | Verified `core.alloc_bytes` roots carry bounded metadata, derived offsets, and split rejection markers for negative offsets, upper bounds, and access-width overflow; arbitrary external raw pointers remain conservative. Linux-x64 has runtime evidence, while other targets are build/lower scoped unless separately validated. | `docs/audits/raw-pointer-bounds-metadata-v1.md`, `compiler/internal/runtimeabi/raw_pointer_bounds.go` |
| unsafe fact classes | complete_narrow_slice | MPC-7 separates `unsafe_unknown`, `unsafe_checked`, and `unsafe_verified_root` in PLIR and MemoryFactGraph rows. `unsafe_unknown` cannot authorize safe provenance, noalias, bounds-check elimination, or trusted stack/region-style lowering; verified allocation roots keep only bounded unsafe-origin metadata. | `compiler/internal/memoryfacts`, `compiler/internal/plir`, `tools/cmd/validate-memory-report`, `docs/spec/memory_report_schema_v1.md` |
| raw slice gateway policy | complete_narrow_slice | `raw_slice_from_parts` is unsafe-only. Unknown raw parts remain external/unknown; verified allocation roots may emit bounded unsafe-origin evidence only when constant length bytes fit allocation metadata. Linux-x64 traps negative length and element-size byte overflow before constructing the view. | `docs/spec/unsafe.md`, `compiler/internal/runtimeabi/raw_pointer_bounds.go` |
| explicit island safety | complete_narrow_slice | Explicit island allocation facts and lowering evidence exist for supported local scopes; full cross-target island runtime parity is not claimed. | `docs/design/allocation_planner_lowering.md`, island safety tests |
| implicit region lowering | complete_narrow_slice | Linux-x64 supports a narrow `FunctionTempRegion` path for one active function-local temporary copy buffer per function when planned storage, actual lowering storage, lowered IR enter/make/reset evidence, and validator results all agree. Unsupported retention, actor/task/global/closure/unknown-call, multiple active temp-region allocations, broad control-flow, target-parity, and heap-fallback promotion remain conservative or future work. | `docs/design/allocation_planner_lowering.md`, `compiler/internal/allocplan`, `compiler/internal/lower`, `compiler/internal/validation` |
| allocation planner lowering | complete_narrow_slice | Planner and actual lowering storage are separated, and stack/island/eliminated claims require validation for supported subsets. | `compiler/internal/allocplan`, `compiler/internal/validation` |
| inout/mutable aliasing | partial | Supported mutable/inout ownership diagnostics are conservative. Full mutable alias model is not claimed. | `docs/spec/ownership_v1.md`, ownership marker tests |
| mutable alias/inout report rows | complete_narrow_slice | MPC-5 memory reports project `no_alias`, `mutable_exclusive`, `start_inout_exclusive`, and `end_inout_exclusive` rows for supported `inout` PLIR facts. Unknown, maybe, or call-invalidated alias state is not a validated noalias claim. | `compiler/internal/plir`, `compiler/internal/memoryfacts`, `docs/spec/memory_report_schema_v1.md` |
| provenance/resource summaries | complete_narrow_slice | MPC-6 memory reports project bounded function summary facts for owned returns, borrowed returns from parameters, unknown unsafe returns, global store, actor/task escape, closure capture, pointer retention, return-region/resource provenance, thrown resources, consumed parameters, inout mutation, required effects, and required capabilities. Unknown external calls and unknown unsafe/resource summaries remain conservative and are not optimization permission. | `compiler/internal/plir`, `compiler/internal/memoryfacts`, `docs/design/provenance_lifetime_ir.md`, `docs/spec/memory_report_schema_v1.md` |
| cross-module resource summaries | complete_narrow_slice | Interface metadata preserves currently supported borrowed-return and resource summary metadata where the checker already exposes it, and PLIR `FunctionSummary` carries that bounded metadata into reports. Unsupported resource/generic lifetime shapes and broad FFI lifetime summaries remain conservative/outside scope. | `docs/design/provenance_lifetime_ir.md`, interface tests |
| task/actor/request boundaries | complete_narrow_slice | Actor payload expressions reject borrowed slice/String values unless explicitly copied, and the narrow local typed-mailbox owned-region slice move is report-visible as `claim_level: evidence_only` with `production_runtime_validated: false`. Typed task spawn has no payload expression in the current API, so task String/slice boundary transfer remains conservative instead of a validated cross-task copy path. Request/task region views are explicit entry-scope evidence and cannot escape without later lifetime modeling. | `docs/design/actor_region_transfer.md`, `docs/audits/request-task-region-v1.md`, actor/task safety tests |
| memory reports | complete_narrow_slice | PLIR, proof, bounds, and allocation reports exist. Schema-v1 memory reports project compiler-owned `MemoryFactGraph` facts for raw-bounds, representation metadata, bounded borrow/copy/copy_into evidence, alias/inout evidence, and MPC-6 function summaries. | `docs/design/explainable_one_build.md`, `docs/spec/memory_report_schema_v1.md` |
| target support | partial | Linux-x64 has runtime-backed memory evidence; other targets are build/lower/artifact scoped unless separately validated. | `docs/spec/current_supported_surface.md`, `docs/spec/runtime_abi.md` |
| fuzz/stress coverage | partial | Property/differential and stress artifacts exist for selected compiler/runtime paths, but exhaustive memory fuzzing remains MPC-15. | `docs/audits/fuzz-property-differential-v1.md` |

## Non-Goals Preserved

- no named lifetime parameters such as `'a`;
- no generic lifetime parameter system;
- no full Rust-like borrow checker parity;
- no arbitrary borrowed aggregate return model;
- no production FFI lifetime system;
- no safety claim for arbitrary unsafe external pointers;
- no cross-target memory production claim without target evidence;
- no report flag that changes safe-program semantics.

## Slice Added By This Goal

This slice introduced `compiler/internal/memoryfacts`, the
`tetra.memory-report.v1` schema, `tools/cmd/validate-memory-report`, and
`--emit-memory-report`. MPC-3 then added the validated
`safe_representation_metadata: not_user_assignable` report row. MPC-4 then
added bounded borrow/copy/copy_into report rows, and MPC-5 added bounded
alias/inout report rows. MPC-6 added bounded function summary rows for
provenance/resource evidence and conservative unknown external/unsafe
summaries. MPC-7 hardened unsafe fact classes so `unsafe_unknown` cannot become
optimization or trusted-lowering permission. The report is a projection of
compiler-owned facts and is not optimization permission by itself. MPC-11 added
narrow linux-x64 `FunctionTempRegion` actual-lowering evidence and projection
hardening so `FunctionTempRegion` plans that lower as `Heap` cannot be
reported as validated storage lowering. MPC-12 added conservative
actor/task/request boundary hardening and report honesty: actor zero-copy move
rows remain evidence-only unless a future production actor-runtime validation
slice proves a broader claim.
