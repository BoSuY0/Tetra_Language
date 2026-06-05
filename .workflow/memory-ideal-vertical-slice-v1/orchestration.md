# Orchestration: Memory Ideal Vertical Slice v1

## Execution Rules

- Keep the original objective intact.
- Ask for approval before risky, expensive, external, or destructive actions.
- Keep immediate blocking work local.
- Delegate only bounded, disjoint, materially useful packets.
- Integrate packet results before final verification.
- Preserve unrelated dirty worktree changes.
- Use simulated packet notes only unless the user explicitly authorizes
  subagents/delegation.

## Branching Rules

- If enum/generic wrapper closure requires interfaces, function-typed values,
  callbacks, async, actor/task, raw pointer semantics, target parity, or broad
  noalias, stop and record a blocker instead of broadening.
- If `unsafe_unknown` cannot be proven safe, reject or classify conservative.
- If full gates fail from unrelated dirty state, classify failures and keep
  focused evidence for touched areas.

## Packet Prompts

### P0-discovery

Read-only. Inspect v0 correlation, memoryfacts/report projection, memory report
validator, MiniMemoryModel, and semantics borrow tests. Identify exact files,
symbols, and test commands for v1. Do not edit files.

### P1-tests-model

TDD packet. Add RED tests for memoryfacts/report validator and MiniMemoryModel
v1 enum/generic wrapper cases. Verify RED for the intended missing behavior,
then implement the smallest GREEN changes in owned files.

### P2-semantics

TDD packet. Add RED semantics tests for borrowed enum payload returned/stored,
borrowed generic wrapper returned/stored, branch owner mismatch,
`unsafe_unknown`, local valid use, and `.copy()` owned escape. Implement only
narrow checker propagation needed for those tests.

### P3-docs-audit

Update v1 correlation matrix, report schema docs, manifest references, and final
audit after code behavior is proven. Keep matrix to exactly two v1 rows.

### P4-final-verification

Run focused gates, full gates, `git diff --check`, and `graphify update .`.
Write final workflow report with accepted/rejected/conflicts/decisions/changes/
verification/remaining risks.

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
