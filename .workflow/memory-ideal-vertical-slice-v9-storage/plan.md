# Plan: Memory Ideal Vertical Slice v9 Storage

## Goal

Implement the narrow Memory Ideal v9 escape-aware storage/lowering evidence
slice.

## Current Strategy

Treat v8 as accepted `validated_narrow` integrity evidence, then add exact v9
rows around escape-aware storage rejection, no-escape proof requirements, heap
fallback source/reason preservation, and async/task/actor/FFI boundary
conservatism. Keep this as a storage evidence layer only; do not claim full
region inference or performance.

## Phases

- [x] Compile v9 goal contract from the 2026-06-06 v8 acceptance roadmap.
- [x] Inspect Graphify and concrete storage/escape/lowering paths.
- [x] Add RED tests for escaped trusted storage, missing no-escape proof,
  fallback source/reason drift, boundary conservatism, and v9 correlation
  exactness.
- [x] Implement v9 validators and exact correlation row set.
- [x] Add v9 docs, schema/design notes, manifest entries, and final audit.
- [x] Run focused gates.
- [x] Run v0-v9 regression, docs/manifest, broad/CI, hygiene, and Graphify
  gates.
- [x] Close final report and update goal completion evidence.

## Open Decisions

- Validator names may use exact requested names or clearly equivalent local
  names where the repo already has an idiomatic boundary.
- `MEM-STORAGE-004` should reuse existing async/task/actor/FFI/unknown-call
  escape evidence where possible; split only if a new runtime proof would be
  required.
- Dirty worktree remains a release caveat, not a blocker for v9 evidence.

## Next Action

All v9 acceptance gates passed. Next action is to mark the active goal complete
after final sanity checks.
