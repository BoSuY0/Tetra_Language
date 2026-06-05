# AGENTS.md

## Language

- Always communicate with the user in Ukrainian.
- Keep technical terms clear and natural; use English identifiers, commands,
  file paths, and code symbols exactly as they appear in the project.

## Host temp/cache discipline

- Do not set `GOCACHE` to `/tmp/...` in this repo. On this workstation `/tmp`
  is tmpfs, and repeated validator/TDD slices can fill RAM-backed temp space.
- For isolated Go verification, prefer a persistent cache under
  `$(pwd)/.cache/go-build-<slug>` or
  `${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-<slug>`, then clean
  it with `GOCACHE=<that-path> go clean -cache` when the evidence run is done.
- When recording command evidence in `reports/stabilization`, keep the same
  persistent-cache convention so copied commands do not recreate tmpfs pressure.

## graphify

This project has a graphify knowledge graph at graphify-out/.

Rules:
- Before answering architecture or codebase questions, read graphify-out/GRAPH_REPORT.md for god nodes and community structure
- If graphify-out/wiki/index.md exists, navigate it instead of reading raw files
- For cross-module "how does X relate to Y" questions, prefer `graphify query "<question>"`, `graphify path "<A>" "<B>"`, or `graphify explain "<concept>"` over grep — these traverse the graph's EXTRACTED + INFERRED edges instead of scanning files
- After modifying code files in this session, run `graphify update .` to keep the graph current (AST-only, no API cost)

## Goal-Loop Conventions

When a thread has an active Codex `/goal` whose objective references `GOAL.md`,
treat `GOAL.md` as the source of truth for the loop. The slash-command
objective is only the launcher; durable state and evidence live in repo files.

Re-read `GOAL.md` at the start of every continuation turn before deciding what
to do. Keep the `/goal` command short and consistent with `GOAL.md`, `AGENTS.md`,
and the current master plan.

Maintain the `## Progress` section in `GOAL.md`. Each iteration, before taking
action, update it with:

- completed items with `file:line`, artifact paths, or command output evidence;
- in-progress work with a Bridge note naming the acceptance item it feeds;
- blockers or open questions with enough detail for a fresh session to resume.

Keep progress terse: one line per state change, evidence by reference, not by
transcript. Replace stale in-progress state instead of appending duplicate
status when nothing materially changed.

Verify before marking any acceptance item complete. Run the test, inspect the
diff, and confirm the output. No "should work" claims in `GOAL.md`.

Only call `update_goal { status: "complete" }` when every requirement in
`GOAL.md` is checked off with evidence. Budget exhaustion or plausible progress
is not a completion signal.

If optional local SQLite state exists under `.goal-loop/`, use it only as a
machine-readable mirror. If SQLite and `GOAL.md` disagree, treat `GOAL.md` as
canonical.

If the same fix fails twice, stop and write the blocker into `GOAL.md` instead
of trying a third variant without new evidence.
