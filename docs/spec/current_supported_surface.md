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
- Static monomorphized generic functions: generic functions with inferred value
  arguments are parsed, checked, formatted, documented, and specialized with
  deterministic names across modules. The current truth boundary excludes
  runtime generic values, explicit type arguments, generic structs,
  higher-ranked generics, full protocol-bound generic dispatch, specialization
  optimization, and any dynamic dispatch claim.
- Static protocol conformance: protocol declarations and `impl Type: Protocol`
  are checked against extension/static methods, including compatible effects,
  async, throws, params, return types, and MVP generic requirement signature
  shape (`func req<T>(...)`). This is static conformance only: no witness
  tables, trait objects, runtime protocol values, or dynamic dispatch model are
  introduced.
- Generic protocol requirement parsing/checking in MVP form (`func req<T>(...)`)
  with signature-shape conformance checks and no new runtime dispatch model.
- Function type references in type positions (`fn(T1, T2) -> R`) plus the
  current Level 0 callable MVP for direct local calls of let-bound
  non-capturing closure values, plus callback-parameter calls in callees when
  the call-site passes a known symbol-backed function-typed local or a direct
  named non-generic non-throwing function/closure symbol (immutable in this MVP
  path; reassignment of function-typed locals is rejected). The current safe
  subset also allows returning symbol-backed non-generic non-throwing function
  values from functions with function-typed returns, and immutable
  function-typed local-to-local binding (`let g: fn(...) -> ... = f`) when
  signatures match. This is not a full first-class function-value model.
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
- Ownership markers MVP for `borrow`, `inout`, and `consume` call-site
  contracts. The current checker is conservative: it covers local-call marker
  validation, same-call alias rejection, use-after-`consume`, and borrow escape
  diagnostics, but it is not a full SSA lifetime solver.
- Resource lifetime MVP for task handles, task groups, island handles,
  region-backed slices, and structs containing those resources. Common local
  scopes and control-flow merges are checked conservatively; double-use,
  ambiguous provenance, and ambiguous lifetime merges are diagnostics rather
  than proof obligations solved by a full SSA analysis.
- Actor/task transfer safety MVP for local worker entrypoints, sendable scalar
  and supported structural results, handle transfer, and use-after-transfer
  diagnostics. This is a conservative local MVP; it does not claim distributed
  actor safety, full race-safety proofs, full cancellation semantics, or
  structured concurrency.
- Typed task handle wrappers support slot counts `2..8` in the current runtime
  path (`2..4` direct, `5..8` staged). Layouts above `8` are rejected.

## Future Or Limited

- Full `v1.0.0` language guarantees remain future work.
- Distributed EcoNet, production TetraHub publishing, global trust scoring, and
  proof-carrying capsules remain post-v1 unless explicitly promoted.
- Callable Level 1 is experimental: non-capturing, symbol-backed callable
  expansion beyond the Level 0 MVP may be documented or tested behind explicit
  experimental labels, but it is not part of the stable `v0.2.0` baseline.
  Callable Level 2 is planned/experimental design work for captured closures,
  broader callback movement, lifetime validation, and ABI evidence; it must not
  be marketed as current support.
- Full first-class callable/function-pointer semantics (arbitrary
  escape/passing/storing, complete capture matrix, and ABI redesign) remain
  outside the current supported callable MVP.
- Generic structs, explicit type arguments, higher-ranked generics, runtime
  generic values, protocol-bound generic dispatch, specialization optimization,
  witness tables, trait objects, runtime protocol values, and protocol dynamic
  dispatch remain outside the current `v0.2.0` support claim unless separately
  promoted by a later gate.
- Enum payload constructors and exhaustive enum match/catch coverage are an
  experimental next-cycle promotion slice. The represented slice is limited to
  same-module enum constructors with positional payload arguments/bindings and
  tested exhaustive enum match/catch behavior; it is not a `v0.2.0` stable
  baseline claim. Advanced ADT constructors, nested destructuring patterns,
  richer payload algebra, and guard expansion remain future/post-v1 unless
  separately promoted.
- Lifetime SSA solving is planned future work: the current ownership/resource
  safety implementation is a conservative MVP, not a full SSA lifetime solver.
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
