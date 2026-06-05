# Plan: Memory Ideal Vertical Slice v6 Bounds

## Goal

Implement the narrow Memory Ideal v6 bounds-check proof-id correlation slice.

## Current Strategy

Use the existing proof-id substrate (`validation.CheckBoundsProofsWithPLIR`,
`plir.VerifyProgram`, and lowering BCE tests) as compiler evidence, then add
MemoryFactGraph/report/correlation rows around it. Keep unsafe/raw uncertainty
dynamic, conservative, or rejected; do not claim broad optimizer correctness.

## Phases

- [x] Compile v6 goal contract from the 2026-06-05 audit.
- [ ] Inspect v0-v5 patterns and current proof-id substrate.
- [x] Add RED tests for v6 model/report/correlation/proof behavior.
- [x] Implement v6 facts, validators, and projections.
- [x] Add v6 docs, schema notes, manifest entries, and final audit.
- [x] Run focused gates.
- [x] Run broad/docs/hygiene/Graphify gates.
- [x] Close final report and update goal completion evidence.

## Open Decisions

- Default proof linkage should remain in `source_fact_id`/`parent_fact_id` and
  fact metadata. Add a dedicated report `proof_id` field only if implementation
  evidence shows the existing linkage is insufficient.
- Fuzz artifact policy is not part of v6 except for keeping the existing
  `validate-memory-fuzz-oracle` gate green.
- Target parity remains an explicit nonclaim.

## Next Action

Completion audit is ready. Verify `GOAL.md`, final report, and final hygiene
checks before calling `update_goal complete`.
