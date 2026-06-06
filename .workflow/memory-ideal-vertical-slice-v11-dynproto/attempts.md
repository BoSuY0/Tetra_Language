# MEM-DYNPROTO-011 Workflow Attempts

| Time | Attempt | Evidence | Result | Next Adjustment |
| --- | --- | --- | --- | --- |
| 2026-06-06 | Initialized workflow sidecar. | Root `GOAL.md`, `PLAN.md`, `ATTEMPTS.md`, `NOTES.md`, and `CONTROL.md` updated for v11. | Setup in progress. | Create active goal and begin RED evidence. |
| 2026-06-06 | Added first RED tests. | Targeted tools, MiniMemoryModel, and memoryfacts RED commands failed for missing v11 registration/vocabulary/projection. | RED confirmed. | Implement minimal GREEN support. |
| 2026-06-06 | Implemented first GREEN support. | Targeted v11 tools, MiniMemoryModel, and memoryfacts commands passed. | GREEN confirmed. | Add v11-specific report validator negative tests. |
| 2026-06-06 | Added v11 report integrity validator guard. | RED failed on missing v11 `cost_class`/`normal_build_check`; GREEN targeted report and full memoryfacts commands passed. | GREEN confirmed. | Add audit docs and manifest evidence. |
| 2026-06-06 | Added v11 docs/manifest evidence. | v11 correlation, v0-v11 correlation regression, manifest, and docs verification passed. | GREEN confirmed. | Run focused package gates. |
| 2026-06-06 | Ran focused package gates. | Full memoryfacts, full memorymodel, semantics/ownership focused regex, and tools package gates passed. | GREEN confirmed. | Run broad/CI/hygiene. |
| 2026-06-06 | Ran final gates. | Broad `go test` passed; CI ended `OK` with artifact `tetra.release.v0_4_0.go-test-suite.v1`; `git diff --check` passed; `git status --short` dirty output recorded; Graphify rebuilt `21397 nodes`, `66817 edges`, `1189 communities`. | GREEN with dirty-worktree blocker caveat. | Complete final report and goal. |
