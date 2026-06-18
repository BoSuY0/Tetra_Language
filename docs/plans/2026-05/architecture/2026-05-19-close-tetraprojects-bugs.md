# Close TetraProjects_Bugs.md

Goal: close all confirmed `BUG-001` through `BUG-075` in
`/home/tetra/Desktop/Projects/Tetra_Language/TetraProjects_Bugs.md` with a
RED/GREEN test or equivalent repro evidence for each cluster, then update
Graphify.

## Working Rules

- Treat `TetraProjects_Bugs.md` as the source of truth for confirmed bug IDs.
- Prefer existing tests when they already encode the repro; otherwise add the
  smallest focused regression test before changing production code.
- Keep fixes grouped by subsystem so verification stays local and failures are
  easy to attribute.
- Do not clean or revert unrelated dirty worktree changes.
- After code changes, run `graphify update .` before final handoff.

## Clusters

1. Stdlib and runtime guardrails: `BUG-001`, `BUG-004`, `BUG-005`,
   `BUG-009`, `BUG-012`, `BUG-013`.
   Verify with focused stdlib/runtime tests and, where needed, `go run
   ./cli/cmd/tetra check/run` probes.

2. CLI artifact, capsule, and eco packaging: `BUG-002`, `BUG-007`, `BUG-008`.
   Verify with `go test ./compiler`, `go test ./cli/cmd/tetra`, and stale
   binary/build-contract checks.

3. Actor, task, and transport boundaries: `BUG-003`, `BUG-006`, `BUG-014`,
   `BUG-016`, `BUG-017`, `BUG-032`, `BUG-033`, `BUG-037`, `BUG-065`.
   Verify with `go test ./cli/internal/actornet ./cli/cmd/tetra` plus targeted
   compiler safety tests.

4. Numeric, literals, arrays, slices, strings, and lowering correctness:
   `BUG-010`, `BUG-011`, `BUG-015`, `BUG-018`, `BUG-019`, `BUG-020`,
   `BUG-023`, `BUG-024`, `BUG-025`, `BUG-026`, `BUG-027`, `BUG-028`,
   `BUG-029`, `BUG-030`, `BUG-031`, `BUG-056`.
   Verify with `go test ./compiler/tests/semantics`, backend/lowering tests,
   and focused run/build probes.

5. Flow, defer, try, privacy, and budget semantics: `BUG-021`, `BUG-022`,
   `BUG-034`, `BUG-035`, `BUG-036`, `BUG-038`, `BUG-039`, `BUG-040`,
   `BUG-041`, `BUG-042`, `BUG-043`, `BUG-063`, `BUG-064`.
   Verify with `go test ./compiler/tests/semantics ./compiler/tests/safety`
   and direct probes for lowering/runtime-only bugs.

6. Export/ABI/object metadata and symbol validation: `BUG-044` through
   `BUG-062`, `BUG-066` through `BUG-072`.
   Verify with `go test ./compiler/tests/safety`, TOBJ format tests, object
   metadata readers, and malformed symbol tests.

7. WASM import/export/symbol address behavior: `BUG-073`, `BUG-074`,
   `BUG-075`.
   Verify with `go test ./compiler/internal/backend/wasm32_wasi
   ./compiler/internal/backend/wasm32_web` and WASM validation probes.

## Current Evidence

- Baseline safety/export subset: passing.
- Baseline TOBJ/WASM symbol subset: passing.
- Baseline actor-net/eco subset: passing.
- Initial semantics subset failed only on `BUG-063`.
- `BUG-063` RED/GREEN: `TestDeferRejectsLaterConsumeOfCapturedDescendant`
  now passes with `TestDeferAllowsSiblingCaptureAfterDescendantConsume`.

## Done Criteria

- Every `BUG-001` through `BUG-075` has either a passing regression test or a
  documented no-longer-reproducing command.
- `TetraProjects_Bugs.md` is updated with closure evidence for fixed bugs.
- Relevant package tests pass for all touched subsystems.
- `graphify update .` completes after code edits.
