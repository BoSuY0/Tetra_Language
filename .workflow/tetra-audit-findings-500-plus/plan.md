# Tetra audit findings 500 plus

## Goal
Resolve the actionable live-checkout issues from `/home/tetra/Downloads/tetra_audit_findings_500plus.md` and classify the rest with evidence.

## Success Criteria
- All 748 F-IDs are represented in a triage artifact with status and evidence.
- Live code/script issues are fixed with targeted RED/GREEN or static guard evidence.
- Dump-only, historical, ignored-artifact, and external-evidence findings are not mislabeled as code fixes.
- Sub-agent packet results are integrated and conflicts are resolved against the live repo.
- Final verification includes focused touched checks, `git diff --check`, and `graphify update .`.

## Current Context
- Active `/goal` objective asks to fix all findings with sub-agents.
- The audit file has 748 findings: 3 critical, 146 high, 527 medium, 72 low.
- Category counts: 430 missing/unverifiable references, 195 documented bug/regression ledger entries, 90 placeholder/fake markers, 15 binary dump artifacts, 5 shell robustness, 4 Go toolchain mismatch, 3 dump integrity, 3 stale/generated evidence, 2 release evidence contradiction, 1 build/test failure.
- Worktree is heavily dirty; preserve unrelated changes.

## Constraints
- Follow `AGENTS.md`: Ukrainian user communication, persistent Go cache, Graphify before codebase/architecture decisions, `graphify update .` after code changes.
- Use sub-agents only for bounded disjoint packets; require evidence.
- Do not delete historical/generated evidence without approval.
- Do not claim dump-only issues are fixed unless the live checkout and artifact policy prove it.

## Risks
- Many findings are evidence-policy issues, not local source bugs.
- `reports/` and `artifacts/` are ignored; generating hundreds of files could add noise without solving tracked evidence integrity.
- `docs/generated/v1_0` is explicitly a mixed historical compatibility workspace.
- The repo already has extensive unrelated changes; broad tests may expose unrelated failures.

## Approval Required
- Required before deleting, mass-moving, or regenerating historical release archives.
- Required before changing release-line policy for `docs/generated/v1_0`.
- Not required for read-only sub-agents, focused tests, local script hardening, or triage artifacts.

## Work Packets
- P-A build/toolchain/scripts: `F-0001..F-0005`, `F-0744..F-0748`.
- P-B generated release/dump/binary evidence: `F-0006..F-0028`.
- P-C missing referenced artifacts: 430 findings, grouped by referenced path and source doc.
- P-D placeholders and bug ledgers: 90 placeholder/fake markers and 195 documented bug ledger findings.
- P-I integration: build a full triage artifact and apply live fixes.
- P-V verification: focused tests, broad checks, diff/graph gates.

## Integration Policy
- Accept sub-agent findings only after direct live-file inspection or command evidence.
- Prefer small source/script fixes for live bugs.
- Prefer documented classification plus validator policy for ignored/external/historical evidence findings.
- Record rejected findings with the reason: dump-only, stale audit location, already fixed, historical snapshot, or needs release decision.

## Verification
- Static RED/GREEN for Go floor compatibility: no `testing.T.Chdir` in `_test.go`.
- Focused Go package tests for touched Go packages.
- Static/script checks for shell release scripts.
- Docs/generated evidence checks or blockers for release-state findings.
- Final `git diff --check` and `graphify update .`.

## Reusable Artifacts
- `.workflow/tetra-audit-findings-500-plus/results/*.md`
- planned triage artifact for all F-IDs
