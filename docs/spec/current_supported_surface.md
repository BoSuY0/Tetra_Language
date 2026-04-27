# Tetra Current Supported Surface

Status: current for `v0.1.2`.

This document is the short release-truth layer for the current public Tetra
profile. It records what the repository may describe as supported now, and what
must still be described as future or planned.

`v1.0.0` is a future label. The future scope contract remains
`docs/spec/v1_scope.md`, but the current user-facing and release-facing truth is
the `v0.1.2` local compiler/tooling profile.

## Current Release Gate

- Current gate: `scripts/release_v0_1_2_gate.sh`.
- Current checklist: `docs/checklists/v0_1_2_release_gate.md`.
- Compatibility alias: `scripts/release_v1_0_gate.sh` delegates to the current
  `v0.1.2` gate and must not be treated as proof of `v1.0.0` readiness.
- Historical gate: `scripts/release_v0_1_1_gate.sh` remains for the immutable
  `v0.1.1` tag.

## Supported Now

- Flow indentation syntax for the examples, standard library sources, runtime
  sources, and self-hosted runtime snippets covered by the release gate.
- Local compiler and CLI workflows: `check`, `build`, `run`, `fmt`, `test`,
  `doc`, `doctor`, `targets`, `smoke`, `eco`, `clean`, and `version`.
- Native build/smoke coverage for `linux-x64`, plus build-only coverage for
  `macos-x64` and `windows-x64`.
- Build-only WASM target smoke for `wasm32-wasi` and `wasm32-web`, with release
  smoke reports validated by the gate.
- Local Eco package lifecycle validation for verify, lock, pack/unpack, vault,
  and publish metadata fixtures.
- JSON reports and validators for diagnostics, tests, smoke lists, targets,
  doctor output, web UI smoke, artifact hashes, and release state.

## Future Or Limited

- Full `v1.0.0` language guarantees remain future work.
- Distributed EcoNet, production TetraHub publishing, global trust scoring, and
  proof-carrying capsules remain post-v1 unless explicitly promoted.
- Distributed actors, full async cancellation/structured concurrency, full UI
  runtime event dispatch, and native widget rendering remain outside the
  current `v0.1.2` support claim.
- Any feature labeled `planned`, `beta`, `deferred-post-v1`, or
  `blocked-by-prerequisite` in release docs must not be marketed as stable.

## Patch-Line Rule

`v0.1.x` releases are allowed to clean, stabilize, document, and harden the
existing profile. Breaking language or project compatibility changes belong in
a later `x.0.0` line, and large feature updates belong in a later `0.x.0` line.
