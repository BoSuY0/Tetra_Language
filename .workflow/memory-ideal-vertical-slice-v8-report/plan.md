# Plan: Memory Ideal Vertical Slice v8 Report Integrity

## Goal

Implement the narrow Memory Ideal v8 graph/report projection and claim-drift
integrity slice.

## Current Strategy

Treat v7 as accepted `validated_narrow` evidence, then add exact v8 rows around
report projection identity, graph-to-report completeness, cost/normal-build
preservation, correlation exactness, and memory claim drift. Keep this as an
integrity layer only; do not add new memory semantics.

## Phases

- [x] Compile v8 goal contract from the 2026-06-05 v7 acceptance roadmap.
- [x] Inspect graph/report/correlation validator implementation and current
  schema/docs wording. Current bridge: Graphify and `rg` surfaced
  `compiler/internal/memoryfacts/report.go`, `validate-memory-report`, and
  `validate-memory-correlation`. 2026-06-06 inspection found one-way
  `BuildReportFromGraph` projection and no explicit graph/report projection
  validator yet.
- [x] Add RED tests for report graph projection identity/completeness,
  cost-class and normal-build preservation, correlation drift, and claim drift.
- [x] Implement v8 validators and exact correlation row set.
- [x] Add v8 docs, schema notes, manifest entries, and final audit.
- [x] Run focused gates.
- [x] Run v0-v8 regression, docs/manifest, broad/CI, hygiene, and Graphify
  gates.
- [x] Close final report and update goal completion evidence.

## Open Decisions

- Validator naming may use exact requested names or clearly equivalent local
  names if the codebase already has a more idiomatic API boundary.
- Claim-drift linting should stay memory-audit scoped and reject broad safety
  claims derived from conservative/rejected rows; it must not become a general
  prose style checker.
- Dirty worktree remains a release caveat, not a blocker for v8 evidence.

## Next Action

Complete. Preserve v8 as `validated_narrow` report-integrity evidence with
dirty-worktree release caveat.
