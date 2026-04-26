# Roadmap v0.14 → v0.15 (Core Language MVP)

Focus: stabilize the first Core Alpha language surface after the v0.14 Flow
bridge, without changing backend, linker, ABI, runtime, package manager, or IR.

## P0 — Core language surface

- Add real `bool` with `true`/`false`; keep integer conditions accepted as a
  legacy-compatible bridge.
- Add exclusive integer range loops: `for i in start..<end:`.
- Add no-payload enums and same-module enum case expressions such as
  `Color.green`.
- Add statement-level `match` over enum or integer values with enum cases,
  integer literal cases, and `_` default.

## P1 — Diagnostics and compatibility

- Remove `enum`, `for`, and `match` from planned-feature diagnostics.
- Keep planned-feature diagnostics for `protocol`, `extension`, `actor`, `view`,
  `state`, `test`, `property`, and `capsule`.
- Report typed errors for bool/numeric mismatches, duplicate enum cases, unknown
  enum cases, invalid match patterns, and misplaced match defaults.
- Keep legacy brace syntax and v0.14 Flow examples passing.

## P2 — Examples, smoke, and docs

- Add `bool_smoke`, `for_range_smoke`, and `enum_match_smoke` examples.
- Extend `tetra smoke` so the native smoke suite covers the new examples.
- Update README and the Flow/Core MVP spec to show v0.15 as the bool/range
  for/enum/match MVP.

## Deferred runtime backlog

The previous v0.15 runtime backlog is intentionally deferred to v0.16 or a
parallel runtime track:

- capabilities/globals documentation hygiene;
- self-host actors runtime productionization across all OS targets;
- self-host runtime auto-build/link mode;
- broader library linking ergonomics beyond the current `--link-object` path.
