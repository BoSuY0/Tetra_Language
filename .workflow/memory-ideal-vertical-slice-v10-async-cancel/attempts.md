# MEM-ASYNC-010 Workflow Attempts

| Time | Attempt | Evidence | Result | Next Adjustment |
| --- | --- | --- | --- | --- |
| 2026-06-06 | Initialized workflow from user-provided v10 scope. | `GOAL.md`, `PLAN.md`, `CONTROL.md`; Graphify MCP query/neighbors/path. | Pending verification. | Begin RED tests after concrete file inspection. |
| 2026-06-06 | Added first RED tests for correlation and MiniMemoryModel v10 vocabulary. | Targeted `go test` commands with `.cache/go-build-memory-v10-async-*-red`. | RED: `MEM-ASYNC-*` unexpected; v10 model constants undefined. | Register v10 rows and add model vocabulary/evaluation. |
| 2026-06-06 | Added minimal v10 correlation and model support. | Targeted `go test` commands with `.cache/go-build-memory-v10-async-tools` and `.cache/go-build-memory-v10-async-mini`. | GREEN for first cluster. | Move to memoryfacts/report projection RED. |
| 2026-06-06 | Added v10 graph/report projection RED then GREEN. | Targeted memoryfacts test with `.cache/go-build-memory-v10-async-memoryfacts-red` then `.cache/go-build-memory-v10-async-memoryfacts`. | GREEN for five source/report rows. | Create v10 audit docs and manifest evidence. |
| 2026-06-06 | Ran focused gates and repaired README release-marker drift found by broad gate. | Focused gates passed; broad failed in `tools/scriptstest`; targeted README alignment test passed after marker fix. | Broad rerun pending. | Rerun broad gate. |
| 2026-06-06 | Final gates passed and final report written. | Broad, CI, hygiene, status, and Graphify evidence recorded in final report. | All goal gates passed with dirty-worktree caveat. | Completion audit. |
