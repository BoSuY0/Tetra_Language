# P3-borrow-inout-surface Result

Status: completed_read_only

Sub-agent: Plato (`019e91ed-3354-76a3-823c-24a0ecf9a08a`)

## Accepted Leads

- Slice/String view rewrites and native mappings for `window`, `prefix`,
  `suffix`, `borrow`, `copy`, and `copy_into` are centered in
  `compiler/internal/semantics/slice_views.go`.
- Borrowed return/escape checks, including aggregate escape diagnostics, are in
  `compiler/internal/semantics/checker.go`.
- Actor/task boundary borrowed-view checks are in
  `compiler/internal/semantics/exprs.go`.
- PLIR records borrow/copy/view operations and noalias/inout facts in
  `compiler/internal/plir/plir.go`, with verification in
  `compiler/internal/plir/verify.go`.
- Existing test sources include `compiler/tests/semantics/borrow_copy_test.go`,
  `compiler/tests/ownership/ownership_test.go`, and
  `compiler/internal/plir/plir_test.go`, plus safe-view examples.

## Implementation Guidance

- Keep B2a limited to struct and optional borrow propagation/copy escape.
- Keep B3a limited to unique local and sequential inout exclusivity.
- Do not expand async/reentrant/generic/interface/raw/actor/task semantics in
  this slice.

## Risks

- Semantics and PLIR evidence are split; focused tests must cover both layers.
- Raw exposure/noalias invalidation should stay conservative.
- Existing broader tests may mention async or actor paths, but those are
  non-goals for v0 implementation.
