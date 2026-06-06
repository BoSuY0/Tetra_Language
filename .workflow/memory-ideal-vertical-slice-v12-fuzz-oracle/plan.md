# MEM-FUZZ-012 Workflow Plan

Canonical goal: `../../GOAL.md`

## Current Strategy

RED-first on existing `tetra.memory-fuzz.oracle.v1`, then minimal GREEN support
for v12 deterministic Tier 1 release evidence and Tier 2/Tier 3 boundary
classification.

## Checklist

- [x] Create v12 workflow directory.
- [x] Record v11 accepted baseline and dirty-worktree caveat.
- [x] Add RED tests for v12 oracle/report/validator drift.
- [x] Implement minimal GREEN path.
- [x] Generate `reports/memory-fuzz-short/v12/`.
- [x] Run acceptance gates.
- [x] Write final report.
