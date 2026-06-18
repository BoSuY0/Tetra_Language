# Roadmap v0.6 -> v1.0 (Maximal Production Release)

Status: historical roadmap. The current release truth is
`docs/spec/core/current_supported_surface.md`; the future v1 scope and gate evidence are tracked in
`docs/spec/flow/v1_scope.md` and `docs/checklists/v1_0_release_gate.md`.

Tetra 1.0 is the production line for the final platform profile: Flow-only syntax, stable
compiler/tooling, Rust-grade ownership safety, no data races in safe code, x64 plus WASM targets,
stable stdlib, UI model, local verifiable Eco/Todex, and beta network publishing.

## Decisions

- Flow syntax is the only official 1.0 syntax. Legacy brace syntax must be migrated before release
  and removed from the canonical compiler path.
- Mandatory 1.0 targets are `linux-x64`, `macos-x64`, `windows-x64`, `wasm32-wasi`, and
  `wasm32-web`.
- Safe Tetra must provide ownership safety and no data races. Unsafe and raw capability APIs remain
  explicit, effect-gated, and auditable.
- Local Eco/Todex workflows are stable in 1.0. Network EcoNet/TetraHub publishing is allowed only as
  an explicitly labeled beta surface.

## Wave 0: Release Tracking

- Keep the current-version stabilization gate green while new 1.0 work lands behind focused tests.
- Maintain `docs/spec/flow/v1_scope.md`, `docs/checklists/v1_0_release_gate.md`, and the draft
  release notes as the source of truth for 1.0 readiness.
- Keep `scripts/release/v1_0/gate.sh` intentionally failing until the release version and all
  mandatory capability checks are implemented.

## Wave 1: Flow-Only Frontend

- Make Flow syntax the canonical parser/frontend path.
- Add migration diagnostics and tooling for legacy brace syntax.
- Use `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt` as the
  release-profile scanner while migrating sources.
- Remove legacy examples from release coverage before 1.0.
- Finish argument labels, expression-bodied functions, `elif`, closures, payload enum syntax,
  exhaustive match, and semantic clauses.

## Wave 2: Stable Type System

- Complete multi-slot optionals and typed errors.
- Implement generic functions and generic structs across modules.
- Add protocol-bound generics, extension conformance clauses, and stable monomorphization names.
- Make pattern matching exhaustive for closed enums and optionals.

## Wave 3: Ownership And Race Freedom

- Turn `borrow`, `inout`, and `consume` into a real borrow/lifetime checker.
- Reject escaping borrowed locals, use-after-move, mutable aliasing, and unsafe island transfers in
  safe code.
- Enforce actor/task transfer rules and race-free shared state.

## Wave 4: Effects, Capabilities, Privacy, Budgets

- Extend `uses` into effect groups, effect propagation through generics and protocols, and stable
  diagnostics.
- Add capability attenuation and capsule permission checks.
- Add secret/privacy types, consent-token MVP, and checked privacy clauses.
- Enforce `budget`, `noalloc`, `noblock`, `realtime`, and `nothrow` where static checks are
  possible, with runtime checks for the rest.

## Wave 5: Time Runtime

- Replace async MVP lowering with a real cooperative runtime.
- Add structured task groups, cancellation, typed task handles, and typed async error propagation.
- Expand actors beyond `i32` messages and keep self-host x64 plus WASM runtime paths covered.

## Wave 6: Backends And ABI

- Stabilize native x64 ABI, object/library linking, runtime symbols, debug info, release
  optimization, and deterministic builds.
- Add `wasm32-wasi` and `wasm32-web` build paths with smoke coverage.
- Add incremental check/build cache validation and release-mode optimizer coverage.

## Wave 7: Stdlib 1.0

- Promote stable modules for collections, strings, slices, math, filesystem, networking, async,
  sync, testing, serialization, time, and crypto interfaces.
- Require docs, doctests, examples, formatter coverage, effects, and API diff metadata for every
  stable module.

## Wave 8: Developer Tooling

- Stabilize `tetra` and add the final `t` alias.
- Support `check`, `build`, `run`, `fmt`, `test`, `doc`, `lsp`, `eco`, `clean`, and `version`.
- Make formatter idempotent with full line/block comment preservation.
- Make JSON diagnostics, test reports, smoke reports, Eco reports, and LSP responses stable schemas.
- Complete LSP diagnostics, hover, go-to definition, references, rename, completion, formatting, and
  code actions.

## Wave 9: UI

- Implement stable `view`, `state`, binding, event, command, typed style, and accessibility syntax.
- Add Web backend through `wasm32-web`.
- Add native shell backend for desktop embedding.
- Add UI smoke applications for web and native shell.

## Wave 10: Eco And Publishing

- Stabilize Capsule manifest v1, dependency resolver, permission model, semantic lockfile, local
  Todex Vault, Seed import/export, NeedMap, TrustSnapshot, Materializer, reproducible builds, and
  API diff checker.
- Add beta package publishing, TetraHub, target-aware downloads, and trust metadata without claiming
  full distributed EcoNet stability.

## Release Rule

Tetra 1.0 cannot be labeled until:

```sh
bash scripts/ci/test-all.sh --full
bash scripts/release/v1_0/gate.sh
```

both pass on the release branch, and the generated docs manifest, release notes, checklist,
examples, stdlib docs, and API diff reports are current.
