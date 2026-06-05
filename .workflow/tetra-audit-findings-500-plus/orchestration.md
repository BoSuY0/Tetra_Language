# Orchestration: Tetra audit findings 500 plus

## Execution Rules

- Keep the original objective intact.
- Ask for approval before risky, expensive, external, or destructive actions.
- Keep immediate blocking work local.
- Delegate only bounded, disjoint, materially useful packets.
- Integrate packet results before final verification.

## Branching Rules
- If a finding is live code/script behavior, reproduce or add a static RED guard, then patch and verify.
- If a finding references an ignored report/artifact path, classify by tracked/untracked/ignored/missing and avoid generating bulky ignored artifacts unless the final evidence policy requires it.
- If a finding is dump-only, close only with regenerated dump evidence or an explicit limitation.
- If release-state evidence is historical by design, do not overwrite it without a release-lane decision; add a policy/validator or classify as historical.
- If sub-agent results disagree, inspect the authoritative local file and command output.

## Packet Prompts
- P-A: read-only audit for Build/test failure, Toolchain/version mismatch, Shell/release-script robustness.
- P-B: read-only audit for Dump integrity, Unverifiable binary artifact, Release evidence contradiction, Stale/generated evidence.
- P-C: read-only grouping for Missing/unverifiable referenced artifact/path.
- P-D: read-only grouping for Placeholder/fake markers and documented bug ledgers.
- P-I: local integration and implementation of accepted live fixes.
- P-V: final verification and completion audit.

## Completion Audit
- Every F-ID has status/evidence in the triage artifact.
- Workflow `state.json` packet statuses are complete or blocked with reason.
- `final-report.md` summarizes accepted, rejected, conflicts, verification, and remaining risks.
