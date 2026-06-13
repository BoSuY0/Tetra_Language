# AGENTS.md

## Language

- Always communicate with the user in Ukrainian.
- Preserve English identifiers, commands, paths, and code symbols exactly.


## Completion integrity / Definition of Done

- Interpret user wording such as `повністю`, `до кінця`, `100%`, `final`, `complete`, and `end-to-end` as the whole requested outcome, not one local metric, file, validator, or test.
- Never claim `DONE`, `готово`, `complete`, `final`, or `100%` from a local success signal. A green validator, one passing test, one fixed endpoint, or one score at 100% is only `LOCAL` evidence.
- Use completion levels when reporting progress:
  - `LOCAL`: one file/module/check is valid;
  - `INTEGRATION`: affected parts are wired together;
  - `END_TO_END`: the real user/system flow works;
  - `FINAL`: all acceptance criteria and validation gates pass.
- Final task status must be exactly one of: `DONE`, `PARTIAL`, or `BLOCKED`.
- Mark `DONE` only when all acceptance criteria are satisfied, affected surfaces were reviewed, relevant validation passed or was explicitly justified, and no known blocker remains.
- If anything remains unknown, unimplemented, untested, or blocked, report `PARTIAL` or `BLOCKED` and name the exact remaining gap.
- Final answers for non-trivial work must include: `Status`, `Completed`, `Scope covered`, `Validation`, `Evidence`, and `Not verified / risks`.

## Scope discovery for agentic coding

Before substantive implementation or before claiming completion, inspect the surfaces that can affect the requested outcome:

- files/modules/packages;
- APIs, CLI commands, background jobs, and service boundaries;
- storage/schema/migrations and persistence behavior;
- UI/user-facing flows when relevant;
- config/env/permissions/privacy constraints;
- tests, validators, scorers, and smoke-test paths;
- docs or `GOAL.md` acceptance criteria when present.

Do not silently reduce a feature request to a narrower local patch. If only the local patch is possible, say so and keep status `PARTIAL`.

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
- Do not mark a goal complete because one local metric, validator, or test reached 100%; completion requires every acceptance item and relevant end-to-end validation.
- Only call `update_goal { status: "complete" }` when every `GOAL.md` requirement is checked off with evidence. Budget exhaustion, local success, or plausible progress is not completion.
- Optional `.goal-loop/` SQLite is only a machine-readable mirror. If it disagrees with `GOAL.md`, `GOAL.md` wins.
- If the same fix fails twice, stop and write the blocker into `GOAL.md`; do not try a third variant without new evidence.
