# Memory Production Core v1 Baseline

Status: MPC-0 baseline for the 2026-06-03 Memory Production Core v1 plan.

Core claim target: sound for the supported safe surface, conservative for
unknown unsafe memory, and validated for every claimed lowering. This audit does
not claim perfect memory, full Rust-like borrow checker parity, arbitrary unsafe
pointer safety, full target parity, or fastest-language performance.

## Baseline Table

### Safe Representation Metadata

- Status: `complete_narrow_slice`.
- Claim: user-visible slice/String `ptr`/`len` metadata is not assignable
  supported safe state across direct, nested, generic, optional/enum payload,
  inout, and index-through-metadata assignment targets.
- Boundary: broader future metadata names are rejected as representation
  metadata on collection assignment paths, but no full Rust-like lifetime
  model is claimed.
- Evidence: `docs/spec/ownership_v1.md`,
  `docs/design/truthful_safe_values.md`,
  `compiler/internal/semantics/representation_metadata_test.go`,
  `compiler/tests/semantics/slice_view_test.go`, and
  `compiler/tests/semantics/string_metadata_test.go`.

### Safe Slice/String Views

- Status: `complete_narrow_slice`.
- Claim: `window`, `prefix`, `suffix`, `borrow`, `copy`, and `copy_into` exist
  for the supported slice/String byte-view surface.
- Boundary: construction is checked and expressed as PLIR facts.
- Evidence: `docs/design/provenance_lifetime_ir.md`,
  `compiler/tests/semantics/slice_view_test.go`, and
  `compiler/tests/semantics/string_view_test.go`.

### Borrow/Copy/Copy Into

- Status: `complete_narrow_slice`.
- Claim: borrowed views preserve source provenance, `copy()` creates owned
  allocation intent, and `copy_into(dst)` uses caller-owned destination
  storage.
- Report rows: bounded `borrow_owner`, `borrow_source_fact_id`, `copy_owned`,
  `copy_source_fact_id`, and `copy_into_destination_fact_id`.
- Boundary: named lifetimes are not claimed.
- Evidence: provenance/lifetime IR, allocation planner lowering, and
  `docs/spec/memory_report_schema_v1.md`.

### Borrowed Return Syntax

- Status: `complete_narrow_slice`.
- Claim: borrowed returns are supported for documented safe slice/String byte
  views.
- Boundary: named lifetimes and arbitrary borrowed aggregate returns are not
  claimed.
- Evidence: `docs/design/provenance_lifetime_ir.md` and safe view lifetime
  tests.

### Hidden Borrowed Aggregate Escape Diagnostics

- Status: `complete_narrow_slice`.
- Claim: recursive escape diagnostics exist for supported structs, enums,
  optionals, and generic wrappers carrying borrowed views.
- Evidence: `compiler/tests/safety/safety_diagnostics_test.go` and ownership
  marker tests.

### Allocation Length Contract

- Status: `complete`.
- Claim: `make_*` and island allocation lengths classify zero, negative,
  overflow, and dynamic guarded lengths before storage selection.
- Evidence: `docs/design/allocation_planner_lowering.md` and
  `docs/spec/runtime_abi.md`.

### Raw Pointer Bounds Metadata

- Status: `complete_narrow_slice`.
- Claim: verified `core.alloc_bytes` roots carry bounded metadata, derived
  offsets, and split rejection markers.
- Boundary: arbitrary external raw pointers remain conservative; linux-x64 has
  runtime evidence, while other targets need separate validation.
- Evidence: `docs/audits/raw-pointer-bounds-metadata-v1.md` and
  `compiler/internal/runtimeabi/raw_pointer_bounds.go`.

### Unsafe Fact Classes

- Status: `complete_narrow_slice`.
- Claim: MPC-7 separates `unsafe_unknown`, `unsafe_checked`, and
  `unsafe_verified_root` in PLIR and `MemoryFactGraph` rows.
- Boundary: `unsafe_unknown` cannot authorize safe provenance, noalias,
  bounds-check elimination, or trusted stack/region-style lowering.
- Evidence: `compiler/internal/memoryfacts`, `compiler/internal/plir`,
  `tools/cmd/validate-memory-report`, and
  `docs/spec/memory_report_schema_v1.md`.

### Raw Slice Gateway Policy

- Status: `complete_narrow_slice`.
- Claim: `raw_slice_from_parts` is unsafe-only.
- Boundary: unknown raw parts remain external/unknown; verified allocation
  roots may emit bounded unsafe-origin evidence only when constant byte length
  fits allocation metadata.
- Runtime note: linux-x64 traps negative length and element-size byte overflow
  before constructing the view.
- Evidence: `docs/spec/unsafe.md` and
  `compiler/internal/runtimeabi/raw_pointer_bounds.go`.

### Explicit Island Safety

- Status: `complete_narrow_slice`.
- Claim: explicit island allocation facts and lowering evidence exist for
  supported local scopes.
- Boundary: full cross-target island runtime parity is not claimed.
- Evidence: `docs/design/allocation_planner_lowering.md` and island safety
  tests.

### Implicit Region Lowering

- Status: `complete_narrow_slice`.
- Claim: linux-x64 supports a narrow `FunctionTempRegion` path for one active
  function-local temporary copy buffer per function.
- Required evidence: planned storage, actual lowering storage, lowered IR
  enter/make/reset evidence, and validator results must all agree.
- Boundary: unsupported retention, actor/task/global/closure/unknown-call,
  multiple active temp-region allocations, broad control-flow, target parity,
  and heap-fallback promotion remain conservative or future work.
- Evidence: `docs/design/allocation_planner_lowering.md`,
  `compiler/internal/allocplan`, `compiler/internal/lower`, and
  `compiler/internal/validation`.

### Allocation Planner Lowering

- Status: `complete_narrow_slice`.
- Claim: planner and actual lowering storage are separated.
- Boundary: stack/island/eliminated claims require validation for supported
  subsets.
- Evidence: `compiler/internal/allocplan` and
  `compiler/internal/validation`.

### Inout/Mutable Aliasing

- Status: `partial`.
- Claim: supported mutable/inout ownership diagnostics are conservative.
- Boundary: full mutable alias model is not claimed.
- Evidence: `docs/spec/ownership_v1.md` and ownership marker tests.

### Mutable Alias/Inout Report Rows

- Status: `complete_narrow_slice`.
- Claim: MPC-5 memory reports project `no_alias`, `mutable_exclusive`,
  `start_inout_exclusive`, and `end_inout_exclusive` rows for supported
  `inout` PLIR facts.
- Boundary: unknown, maybe, or call-invalidated alias state is not a validated
  noalias claim.
- Evidence: `compiler/internal/plir`, `compiler/internal/memoryfacts`, and
  `docs/spec/memory_report_schema_v1.md`.

### Provenance/Resource Summaries

- Status: `complete_narrow_slice`.
- Claim: MPC-6 memory reports project bounded function summary facts for owned
  returns, borrowed parameter returns, unknown unsafe returns, global store,
  actor/task escape, closure capture, pointer retention, return-region/resource
  provenance, thrown resources, consumed parameters, inout mutation, required
  effects, and required capabilities.
- Boundary: unknown external calls and unknown unsafe/resource summaries remain
  conservative and are not optimization permission.
- Evidence: `compiler/internal/plir`, `compiler/internal/memoryfacts`,
  `docs/design/provenance_lifetime_ir.md`, and
  `docs/spec/memory_report_schema_v1.md`.

### Cross-Module Resource Summaries

- Status: `complete_narrow_slice`.
- Claim: interface metadata preserves currently supported borrowed-return and
  resource summary metadata where the checker already exposes it.
- Report projection: PLIR `FunctionSummary` carries bounded metadata into
  reports.
- Boundary: unsupported resource/generic lifetime shapes and broad FFI lifetime
  summaries remain conservative/outside scope.
- Evidence: `docs/design/provenance_lifetime_ir.md` and interface tests.

### Task/Actor/Request Boundaries

- Status: `complete_narrow_slice`.
- Claim: actor payload expressions reject borrowed slice/String values unless
  explicitly copied.
- Report projection: the narrow local typed-mailbox owned-region slice move is
  report-visible as `claim_level: evidence_only` with
  `production_runtime_validated: false`.
- Boundary: typed task spawn has no payload expression in the current API, so
  task String/slice boundary transfer remains conservative.
- Boundary: request/task region views are explicit entry-scope evidence and
  cannot escape without later lifetime modeling.
- Evidence: `docs/design/actor_region_transfer.md`,
  `docs/audits/request-task-region-v1.md`, and actor/task safety tests.

### Memory Reports

- Status: `complete_narrow_slice`.
- Claim: PLIR, proof, bounds, and allocation reports exist.
- Report projection: schema-v1 memory reports project compiler-owned facts for
  raw bounds, representation metadata, borrow/copy/copy_into evidence,
  alias/inout evidence, and MPC-6 function summaries.
- Evidence: `docs/design/explainable_one_build.md` and
  `docs/spec/memory_report_schema_v1.md`.

### Target Support

- Status: `partial`.
- Claim: linux-x64 has runtime-backed memory evidence.
- Boundary: other targets are build/lower/artifact scoped unless separately
  validated.
- Evidence: `docs/spec/current_supported_surface.md` and
  `docs/spec/runtime_abi.md`.

### Fuzz/Stress Coverage

- Status: `partial`.
- Claim: property/differential and stress artifacts exist for selected
  compiler/runtime paths.
- Boundary: exhaustive memory fuzzing remains MPC-15.
- Evidence: `docs/audits/fuzz-property-differential-v1.md`.

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
