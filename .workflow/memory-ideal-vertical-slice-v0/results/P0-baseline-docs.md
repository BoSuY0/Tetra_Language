# P0-baseline-docs Result

Status: completed_read_only

Sub-agent: Boyle (`019e91ed-30fd-7eb1-8714-9a72a8fbcb39`)

## Accepted Leads

- All seven A0-lite required baseline documents exist.
- The ten required baseline assertions are supported by the current docs:
  `MemoryFactGraph` truth source, report projection boundaries,
  `unsafe_unknown` conservative behavior, pre-lowering metadata assignment
  rejection, borrow/copy/copy_into support, conservative inout/alias evidence,
  and nonclaims for Rust-like borrow parity, arbitrary unsafe pointers, full
  actor runtime, and target parity.
- Manifest/docs hooks for existing memory production docs live in
  `docs/generated/manifest.json`, `tools/cmd/validate-manifest/main.go`, and
  `tools/cmd/verify-docs/main.go`.

## Decisions

- A0-lite classification is `validated_with_gaps`, not `blocked`.
- The gaps are not stop conditions for B1/B2/B3. They are exactly the bounded
  gaps this v0 slice is meant to reduce.

## Risks

- New `memory-ideal-vslice-v0-*` docs are not automatically required by existing
  manifest/docs validators; later audit/manifest work must decide whether to add
  explicit hooks.
- The old MPC baseline date and the 2026-06-04 slice plan are distinct evidence
  points; this v0 baseline doc records the current re-check.
