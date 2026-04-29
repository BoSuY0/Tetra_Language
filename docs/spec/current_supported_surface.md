# Tetra Current Supported Surface

Status: current for `v0.2.0`.

This document is the short release-truth layer for the current public Tetra
profile. It records what the repository may describe as supported now, and what
must still be described as future or planned.

`v1.0.0` is a future label. The future scope contract remains
`docs/spec/v1_scope.md`, but the current user-facing and release-facing truth is
the `v0.2.0` local compiler/tooling profile.

## Current Minor Scope

The current minor line is `v0.2.0`. Its release identity and verification
surface are tracked here:

- Scope contract: `docs/spec/v0_2_scope.md`
- Release checklist: `docs/checklists/v0_2_0_release_gate.md`
- Release gate script: `scripts/release_v0_2_0_gate.sh`
- Release notes: `docs/release-notes/v0_2_0.md`
- Final handoff: `docs/release/v0_2_0_final_handoff.md`

The version metadata is promoted to `v0.2.0`. Tagging still requires a fresh
green `scripts/release_v0_2_0_gate.sh` report and matching handoff evidence.

## Current Release Gate

- Current gate: `scripts/release_v0_2_0_gate.sh`.
- Current checklist: `docs/checklists/v0_2_0_release_gate.md`.
- Future gate: `scripts/release_v1_0_gate.sh` is blocked by a `v1.0.0`
  version preflight before mandatory release checks run and must not be treated
  as proof of `v1.0.0` readiness while the repository remains on `v0.2.0`.
- Historical gate: `scripts/release_v0_1_3_gate.sh` remains for the immutable
  `v0.1.3` tag.
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
- Local Eco package lifecycle validation for verify, lock generation/validation
  through `--lock` workflows, pack/unpack, vault, and publish metadata
  fixtures.
- JSON reports and validators for diagnostics, tests, smoke lists, targets,
  doctor output, web UI smoke, artifact hashes, and release state.
- Target-neutral IR verification before lowering results reach public codegen:
  main metadata, function slot metadata, branch labels, stack heights, local
  slots, returns, calls, unknown instructions, and unsupported lowering paths
  are reported with structured diagnostics.
- Generic protocol requirement parsing/checking in MVP form (`func req<T>(...)`)
  with signature-shape conformance checks and no new runtime dispatch model.
- Function type references in type positions (`fn(T1, T2) -> R`) plus callable
  MVP for direct local calls of let-bound non-capturing closure values, plus
  callback-parameter calls in callees when the call-site passes a known
  symbol-backed function-typed local or a direct named non-generic
  non-throwing function/closure symbol (immutable in this MVP path;
  reassignment of function-typed locals is rejected). The current safe subset
  also allows returning symbol-backed non-generic non-throwing function values
  from functions with function-typed returns, and immutable function-typed
  local-to-local binding (`let g: fn(...) -> ... = f`) when signatures match.
- Semantic-clause checker phase 1 for `noalloc`/`noblock`/`realtime`:
  resolved direct calls, closure-symbol calls, and function-typed callback
  arguments are validated against clause contracts; `realtime` requires
  `noalloc` and `noblock`.
- Top-level globals (`var`/`val`/`property`) in the current global pipeline:
  compile-time constant initializers for scalar MVP types plus `String`/`str`
  when the initializer is a string literal; non-constant/non-literal and
  unsupported-type initializers remain rejected.
- Top-level `property` declarations mapped onto the current global pipeline.
- Top-level language `capsule` declarations accepted as compile-time metadata
  only (duplicate-key/key-shape/value-shape checks; no runtime/codegen impact).
- Native-first `[]u16` slice support including `make_u16` and
  `core.island_make_u16`.
- `[]bool` slice support including `make_bool` and `core.island_make_bool`.
  In the current MVP lowering path, bool-slice allocation reuses the existing
  i32-width slice layout.
  `make_bool` is available on native and build-only WASM targets, while
  `core.island_make_bool` follows the current island runtime boundary (native
  runtime scope); build-only WASM targets provide compile-compatible island IR
  fallback (`island_new` handle token, `island_make_*` mapped to linear heap
  slice allocation by element width, `island_free` no-op).
- Typed task handle wrappers support slot counts `2..8` in the current runtime
  path (`2..4` direct, `5..8` staged). Layouts above `8` are rejected.

## Future Or Limited

- Full `v1.0.0` language guarantees remain future work.
- Distributed EcoNet, production TetraHub publishing, global trust scoring, and
  proof-carrying capsules remain post-v1 unless explicitly promoted.
- Full first-class callable/function-pointer semantics (escape/passing/storing,
  complete capture matrix, and ABI redesign) remain outside the current
  supported callable MVP.
- Distributed actors, full async cancellation/structured concurrency, full UI
  runtime event dispatch, and native widget rendering remain outside the
  current `v0.2.0` support claim.
- Any feature labeled `planned`, `beta`, `deferred-post-v1`, or
  `blocked-by-prerequisite` in release docs must not be marketed as stable.

Language note:
- Source-language `capsule ...` declarations are not Eco package manifests.
  Eco packaging still uses project manifest files (`Capsule.t4`,
  `Tetra.capsule`) and corresponding `tetra eco` workflows.

## Patch-Line Rule

`v0.2.x` releases are allowed to clean, stabilize, document, and harden the
current profile. Breaking language or project compatibility changes belong in
a later `x.0.0` line, and large feature updates belong in a later `0.x.0` line.
