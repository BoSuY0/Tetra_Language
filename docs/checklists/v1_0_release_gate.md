# v1.0 Production Release Gate

Use this checklist before labeling a build or branch as Tetra v1.0.

Scope-freeze reference for unresolved Eco/release/execution-order TODO closure:
`docs/plans/v1_scope_freeze_eco_release.md`.

## Language

- [ ] Flow syntax is the only official syntax in examples, docs, formatter, and
      release smoke coverage.
- [ ] `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`
      passes.
- [ ] Legacy brace syntax is removed from the canonical compiler path.
- [ ] Stable type system covers structs, payload enums, optionals, typed
      errors, modules, generics, protocols, extensions, and exhaustive match.
- [ ] Ownership/lifetime checker rejects use-after-move, escaping borrows,
      mutable aliasing, invalid island transfers, and actor/task race patterns.
- [ ] Safe code has no known memory-safety or data-race unsoundness.
- [ ] Effects, capabilities, privacy clauses, and resource budget clauses have
      stable diagnostics and release tests.

## Compiler And Targets

- [ ] `tetra version` reports the final v1.0 version.
- [ ] Native release builds pass for `linux-x64`, `macos-x64`, and
      `windows-x64`.
- [ ] WASM builds pass for `wasm32-wasi` and `wasm32-web`.
- [ ] Debug info, release optimization, object/library linking, runtime ABI,
      and deterministic build checks are covered.
- [ ] Incremental check/build cache validation is in the release gate.

## Stdlib And Tooling

- [ ] Stable stdlib modules exist for collections, strings, slices, math, IO,
      filesystem, networking, async, sync, testing, serialization, time, and
      crypto interfaces.
- [ ] Every stable stdlib module has API docs, doctests, examples, formatter
      coverage, effects metadata, and API diff metadata.
- [ ] `tetra` and `t` support `check`, `build`, `run`, `fmt`, `test`, `doc`,
      `lsp`, `eco`, `clean`, and `version`.
- [ ] `tetra check examples/flow_hello.tetra` passes without emitting an
      executable.
- [ ] Formatter is idempotent and preserves supported comments.
- [ ] LSP supports diagnostics, hover, go-to definition, references, rename,
      completion, formatting, and code actions.
- [ ] JSON diagnostics/test/smoke/Eco schemas are stable and validated.

## UI

- [ ] `view`, `state`, binding, events, commands, typed styles, and
      accessibility metadata are supported.
- [ ] Web UI backend builds through `wasm32-web`.
- [ ] Native shell UI backend builds on supported host platforms.
- [ ] UI examples have web and native smoke coverage.

## Eco

- [ ] Capsule manifest v1, dependency resolver, permission model, semantic
      lockfile, local Todex Vault, Seed import/export, NeedMap, TrustSnapshot,
      Materializer, reproducible build basics, and API diff checker are stable.
- [ ] Package publishing, TetraHub, target-aware downloads, and trust metadata
      are available as explicitly labeled beta features.
- [ ] Full distributed Todex mesh, proof-carrying capsules, global EcoTrust,
      EcoOracle, and live evolution remain documented as post-1.0.

## Required Commands

- [ ] `go test ./compiler/... ./cli/... ./tools/...`
- [ ] `bash scripts/test_all.sh --full`
- [ ] `bash scripts/release_v1_0_gate.sh`
- [ ] Native host smoke runs.
- [ ] Build-only smoke passes for all mandatory native and WASM targets.
- [ ] WASI smoke runs in a WASI runner.
- [ ] Web UI smoke loads through browser automation.
- [ ] Docs manifest and doctests verify.
- [ ] API diff checker verifies stable public APIs.
- [ ] Reproducible build check passes for at least one native and one WASM
      target.
