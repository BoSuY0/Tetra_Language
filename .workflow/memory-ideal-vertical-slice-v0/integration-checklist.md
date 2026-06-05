# Integration Checklist: memory-ideal-vertical-slice-v0

## P0 Baseline Docs

# P0-baseline-docs Result
## Accepted Leads
- All seven A0-lite required baseline documents exist.
- The ten required baseline assertions are supported by the current docs:
- Manifest/docs hooks for existing memory production docs live in
## Decisions
- A0-lite classification is `validated_with_gaps`, not `blocked`.
- The gaps are not stop conditions for B1/B2/B3. They are exactly the bounded
## Risks
- New `memory-ideal-vslice-v0-*` docs are not automatically required by existing
- The old MPC baseline date and the 2026-06-04 slice plan are distinct evidence

## P1 Semantics Registry

# P1-semantics-registry Result
## Accepted Leads
- The main assignment cutoffs are in
- `compiler/internal/semantics/checker.go` also calls the metadata assignment
- Existing tests already cover `ptr`/`len` metadata behavior across
## Implementation Guidance
- Prefer a centralized registry in semantics over scattered string checks.
- Keep existing diagnostics and `ptr`/`len` behavior stable.
- Add tests for reserved names beyond `ptr`/`len`, especially `owner_id`,
## Risks
- Reserved metadata names beyond `ptr`/`len` appear to have less direct test
- User structs with similarly named ordinary fields may need careful policy:

## P2 Memoryfacts Report

# P2-memoryfacts-report Result
## Accepted Leads
- `compiler/internal/memoryfacts/report.go`, `facts.go`, and `validate.go` are
- `tools/cmd/validate-memory-report/main.go` is the existing CLI report
- `compiler/internal/memoryfacts/from_plir.go` already contains borrow/copy and
- Existing tests to extend or preserve include
## Risks
- The current report validator is stronger for generic report invariants than
- `MEM-ALIAS-001` evidence is split across PLIR verification, memoryfacts
- The sub-agent did not run validators; this result is static discovery only.

## P3 Borrow Inout Surface

# P3-borrow-inout-surface Result
## Accepted Leads
- Slice/String view rewrites and native mappings for `window`, `prefix`,
- Borrowed return/escape checks, including aggregate escape diagnostics, are in
- Actor/task boundary borrowed-view checks are in
- PLIR records borrow/copy/view operations and noalias/inout facts in
`compiler/internal/plir/plir.go`, with verification in
- Existing test sources include `compiler/tests/semantics/borrow_copy_test.go`,
## Implementation Guidance
- Keep B2a limited to struct and optional borrow propagation/copy escape.
- Keep B3a limited to unique local and sequential inout exclusivity.
- Do not expand async/reentrant/generic/interface/raw/actor/task semantics in
## Risks
- Semantics and PLIR evidence are split; focused tests must cover both layers.
- Raw exposure/noalias invalidation should stay conservative.
- Existing broader tests may mention async or actor paths, but those are

## P4 Final Review

# P4-final-review Result
## Accepted
- P0/P1/P2/P3 were used as read-only discovery packets and integrated against
- B1-min, MiniMemoryModel v0, B2a, B3a, report projection, docs, and manifest
- The optional copied payload gap was reproduced as a failing acceptance test
- Required focused and full gates passed with persistent `GOCACHE` paths under
## Rejected
- No broad borrow checker, broad noalias model, enum/generic/interface/callable
performance claim was accepted into this slice.
## Remaining Risk
- The worktree had substantial pre-existing dirty state. Verification passed in

## Integration Decisions

Accepted:

Rejected:

Conflicts:

Remaining risks:

Verification still needed:
