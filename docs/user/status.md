# Current Status

Status: user-facing release status summary for the current branch.

The current public profile is `v0.4.0`. Treat this page as a navigation layer;
the release-truth documents remain `docs/spec/current_supported_surface.md` and
`docs/spec/v0_4_scope.md`.

## Candidate Status

The current branch has a `v0.4.0` Linux-x64 release gate path plus separate
post-v0.4 WASM/UI/GUI promotion evidence.

This candidate is verified for local Linux development and testing. For the
current Linux-x64 objective, macOS and Windows runtime evidence are out of
scope. The post-v0.4 WASM/UI/GUI gate is not full `v1.0.0` readiness.

## Supported Today

- Local compiler and CLI workflows: `check`, `build`, `run`, `fmt`, `test`,
  `doc`, `doctor`, `targets`, `smoke`, `eco`, `clean`, and `version`.
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

`docs/spec/current_supported_surface.md` explains the promoted `v0.4.0` slices
and the areas that remain experimental, planned, or reporting-only.

## Future Work

`docs/spec/v1_scope.md` is the future major-release contract. It must not be
read as current `v1.0.0` readiness while the branch remains on `v0.4.0`.

## Where To Go Next

- Start here: `docs/user/getting_started.md`
- Commands: `docs/user/cli_cheatsheet.md`
- Current truth: `docs/spec/current_supported_surface.md`
- WASM/UI guide: `docs/user/wasm_ui_guide.md`
- Examples: `docs/user/examples_index.md`
- Troubleshooting: `docs/user/troubleshooting.md`
