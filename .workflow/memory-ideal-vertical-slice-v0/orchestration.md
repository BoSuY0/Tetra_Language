# Orchestration: Memory Ideal Vertical Slice v0

## Execution Rules

- Keep the original objective intact: implement the v0 vertical slice, not the
  whole ideal memory system.
- Ask for approval before risky, expensive, external, or destructive actions.
- Keep immediate blocking work local.
- Delegate only bounded, disjoint, materially useful packets.
- Integrate packet results before final verification.
- Preserve unrelated dirty worktree changes.

## Branching Rules

- If A0-lite baseline is `blocked`, stop B1/B2/B3 implementation and record the
  blocker with a split recommendation.
- If B2 requires enum/generic/function/interface/async/actor propagation, stop
  and split.
- If B3 requires callback/reentrant/async/concurrency/raw-pointer semantics,
  stop and split.
- If report schema migration blocks semantic progress, keep the minimal
  projection and record the broader migration as future work.
- If full gates fail from unrelated dirty state, classify the failure and keep
  focused evidence for touched areas.

## Packet Prompts

### P0-baseline-docs

Read-only. Verify the baseline docs required by Task 0, inspect manifest/doc
validator expectations, and report missing/stale/conflicting evidence. Do not
edit files.

### P1-semantics-registry

Read-only. Inspect semantics field-assignment/type-resolution paths and existing
representation metadata tests. Identify the smallest hook for a centralized
metadata registry and RED/GREEN tests. Do not edit files.

### P2-memoryfacts-report

Read-only. Inspect `compiler/internal/memoryfacts`, `compiler/internal/plir`,
`compiler/internal/validation`, `tools/cmd/validate-memory-report`, and existing
report validators. Identify minimal facts/report rows/validator changes for the
three requirements. Do not edit files.

### P3-borrow-inout-surface

Read-only. Inspect borrow/copy/copy_into/inout syntax and tests in compiler and
semantics packages. Identify existing support, gaps, and unsupported forms that
must stay conservative. Do not edit files.

### P4-final-review

Read-only after implementation. Check spec compliance first, then correctness
risks, missing tests, and verification gaps. Cite files and commands.

## Completion Audit

`final-report.md` must include:

```text
Accepted:
Rejected:
Conflicts:
Decisions:
Final changes:
Verification:
Remaining risks:
```
