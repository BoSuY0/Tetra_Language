# AGENTS.md

## Language

- Always communicate with the user in Ukrainian.
- Preserve English identifiers, commands, paths, and code symbols exactly.

## Host temp/cache discipline

- Never set `GOCACHE` to `/tmp/...` in this repo; `/tmp` is tmpfs and repeated validators/TDD slices can exhaust RAM-backed temp space.
- For isolated Go verification, use a persistent cache under `$(pwd)/.cache/go-build-<slug>` or `${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-<slug>`.
- After the evidence run, clean it with `GOCACHE=<that-path> go clean -cache`.
- Evidence recorded in `reports/stabilization` must keep the same persistent-cache convention so copied commands do not recreate tmpfs pressure.

## graphify

Knowledge graph path: `graphify-out/`.

- Before architecture/codebase answers, read `graphify-out/GRAPH_REPORT.md` for god nodes and community structure.
- If `graphify-out/wiki/index.md` exists, navigate it instead of raw files.
- For cross-module “how does X relate to Y” questions, prefer `graphify query "<question>"`, `graphify path "<A>" "<B>"`, or `graphify explain "<concept>"` over grep; these traverse EXTRACTED + INFERRED edges.
- After modifying code in this session, run `graphify update .` to keep the graph current (AST-only, no API cost).

## Goal-Loop Conventions

When an active Codex `/goal` objective references `GOAL.md`, `GOAL.md` is canonical. The slash command is only the launcher; durable state and evidence live in repo files.

- Re-read `GOAL.md` at every continuation start before choosing work.
- Keep `/goal` short and consistent with `GOAL.md`, `AGENTS.md`, and the current master plan.
- Maintain `## Progress` in `GOAL.md`. Before each iteration, record:
  - completed items with `file:line`, artifact paths, or command output evidence;
  - in-progress work with a Bridge note naming the acceptance item it feeds;
  - blockers/open questions detailed enough for a fresh session.
- Keep progress terse: one line per state change; evidence by reference, not transcript. Replace stale in-progress state instead of duplicating it.
- Before marking acceptance complete, run the test, inspect the diff, and confirm output. No “should work” claims in `GOAL.md`.
- Only call `update_goal { status: "complete" }` when every `GOAL.md` requirement is checked off with evidence. Budget exhaustion or plausible progress is not completion.
- Optional `.goal-loop/` SQLite is only a machine-readable mirror. If it disagrees with `GOAL.md`, `GOAL.md` wins.
- If the same fix fails twice, stop and write the blocker into `GOAL.md`; do not try a third variant without new evidence.
