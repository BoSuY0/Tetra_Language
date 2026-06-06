# MEM-RELEASE-013 Execution Plan

Canonical goal: `GOAL.md`
Workflow directory: `.workflow/memory-release-v13/`

## Goal

Produce a release-grade memory evidence freeze for v0-v12 and classify the
dirty worktree blocker without destructive cleanup or widened safety claims.

## Current Strategy

Treat v13 as an evidence/triage slice. First freeze `git status --short`, then
classify every entry, then regenerate v13 fuzz evidence and run the existing
memory/report/docs gates from the frozen state. Preserve v12 as accepted
`validated_narrow`, but keep clean-release blocked until status is clean or
every dirty entry has explicit triage.

## Phases

- [x] Inspect active goal state, `GOAL.md`, `AGENTS.md`, current dirty status,
  and Graphify memory/release context.
- [x] Compile user-provided v13 requirements into `GOAL.md`.
- [x] Capture `git status --short` into
  `reports/memory-release-v13/git-status-short.txt`.
- [x] Classify every status entry in `reports/memory-release-v13/triage.md`.
- [x] Build `reports/memory-release-v13/evidence-packet.md` for v0-v12.
- [x] Regenerate and validate `reports/memory-fuzz-short/v13/`.
- [x] Add release-summary lint evidence for broad-claim rejection.
- [x] Run correlation, docs/manifest, broad, CI, hygiene, and Graphify gates.
- [x] Write `.workflow/memory-release-v13/final-report.md`.
- [x] Complete only when every `GOAL.md done_when` item has evidence.

## Open Decisions

- None for product intent. Stop and ask only if v13 requires destructive
  cleanup, broad fuzz runtime, target parity, arbitrary unsafe proof, long
  nightly run as mandatory Tier 1, or automatic resolution of unrelated dirty
  files.

## Current Iteration

- Completion audit passed against every `GOAL.md done_when` item. Ready to mark
  the active goal complete.
