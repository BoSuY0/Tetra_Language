# Current Status

Status: user-facing release status summary for the current branch.

The current public profile is `v0.4.0`. Treat this page as a navigation layer;
the release-truth documents remain `docs/spec/current_supported_surface.md` and
`docs/spec/v0_4_scope.md`.

## Candidate Status

The current branch carries `v0.4.0` release evidence:

- `reports/v0.4.0/features.json`
- `reports/v0.4.0/targets.json`
- `reports/v0.4.0/linux-host-smoke.json`
- `reports/v0.4.0/release-gate-clean/summary.json`

The selected production objective is Linux x64. macOS, Windows, and WASM
runtime claims remain bounded by the target evidence in
`docs/spec/current_supported_surface.md`.

## Supported Today

- Core user compiler workflow: `check` and `build`. Other commands remain
  supported tooling around that workflow: `run`, `fmt`, `test`, `doc`,
  `doctor`, `targets`, `features`, `formats`, `new`, `interface`, `project`,
  `workspace`, `smoke`, `eco`, `clean`, `version`, and `lsp`.
- Flow indentation syntax for release-covered examples, standard library,
  runtime sources, and self-host runtime snippets.
- Native Linux build/run smoke plus macOS and Windows build-only cross-target
  smoke.
- WASM artifact and runtime evidence for `wasm32-wasi` and `wasm32-web`, with
  runtime execution conditional on discoverable WASI/browser runners.
- Positional enum payload constructors/bindings for match/catch/if-let.
- Static protocol-bound generic validation without runtime dynamic dispatch.
- Conservative ownership/resource safety MVP, not a full SSA lifetime solver.
- Local Eco package lifecycle workflows, not distributed production TetraHub.

## Preview Boundaries

`docs/release-notes/v0_4_0.md` explains the selected `v0.4.0` production
scope and the areas that remain excluded from current claims.

## Future Work

`docs/spec/v1_scope.md` is the future major-release contract. It must not be
read as current `v1.0.0` readiness while the branch remains on `v0.4.0`.

## Where To Go Next

- Start here: `docs/user/getting_started.md`
- Commands: `docs/user/cli_cheatsheet.md`
- Current truth: `docs/spec/current_supported_surface.md`
- v0.4 release notes: `docs/release-notes/v0_4_0.md`
- Examples: `docs/user/examples_index.md`
- Troubleshooting: `docs/user/troubleshooting.md`
