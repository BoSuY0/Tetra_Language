# Roadmap v0.17 → v0.18 (Developer Tooling Alpha)

Focus: make the language usable day to day after Flow/Core/effects work by
adding local tooling rather than new runtime or type-system features.

## P0 — CLI tooling

- `tetra fmt` formats the supported MVP syntax in canonical Flow style.
- `tetra test` discovers `.tetra` files, runs `test` blocks on the host target,
  and reports pass/fail counts.
- `--diagnostics=text|json` starts the structured diagnostics surface for build,
  run, fmt, and test workflows.

## P1 — Language test blocks

- Top-level `test "name":` blocks are parsed and ignored by normal app builds.
- `expect <bool>` is test-runner-only syntax.
- v0.18 does not add fixtures, mocking, coverage, property tests, or async tests.

## P2 — Deferred work

Full LSP, complete comment-preserving formatting, ownership, protocols, async,
UI, package publishing, and EcoNet remain v0.19+ work.
