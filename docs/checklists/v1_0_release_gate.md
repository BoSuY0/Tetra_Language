# v1.0 Production Release Gate

Use this checklist before labeling a build or branch as Tetra v1.0.

Scope-freeze reference for unresolved Eco/release/execution-order TODO closure:
`docs/plans/v1_scope_freeze_eco_release.md`.

Execution snapshot date: `2026-04-26`.
Evidence artifacts:
- `docs/generated/v1_0/release_gate_summary.json`
- `docs/generated/v1_0/release_gate_summary.md`
- `docs/generated/v1_0/test_all_full_summary.json`
- `docs/generated/v1_0/api-diff/api-diff.json`
- `docs/generated/v1_0/wasi-smoke.json`
- `docs/generated/v1_0/web-ui-smoke.json`
- `docs/generated/v1_0/reproducible-build.json`

## Language

- [x] Flow syntax is the only official syntax in examples, docs, formatter, and
      release smoke coverage.
- [x] `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`
      passes.
- [x] Legacy brace syntax is removed from the canonical compiler path.
- [x] Stable type system covers structs, payload enums, optionals, typed
      errors, modules, generics, protocols, extensions, and exhaustive match.
- [x] Ownership/lifetime checker rejects use-after-move, escaping borrows,
      mutable aliasing, invalid island transfers, and actor/task race patterns.
- [x] Safe code has no known memory-safety or data-race unsoundness.
- [x] Effects, capabilities, privacy clauses, and resource budget clauses have
      stable diagnostics and release tests.

## Compiler And Targets

- [x] `tetra version` reports the final v1.0 version.
- [x] Native release builds pass for `linux-x64`, `macos-x64`, and
      `windows-x64`.
- [x] WASM builds pass for `wasm32-wasi` and `wasm32-web`.
- [x] Debug info, release optimization, object/library linking, runtime ABI,
      and deterministic build checks are covered.
- [x] Incremental check/build cache validation is in the release gate.

## Stdlib And Tooling

- [x] Stable stdlib modules exist for collections, strings, slices, math, IO,
      filesystem, networking, async, sync, testing, serialization, time, and
      crypto interfaces.
- [x] Every stable stdlib module has API docs, doctests, examples, formatter
      coverage, effects metadata, and API diff metadata.
- [x] `tetra` and `t` support `check`, `build`, `run`, `fmt`, `test`, `doc`,
      `lsp`, `eco`, `clean`, and `version`.
- [x] `tetra check examples/flow_hello.tetra` passes without emitting an
      executable.
- [x] Formatter is idempotent and preserves supported comments.
- [x] LSP supports diagnostics, hover, go-to definition, references, rename,
      completion, formatting, and code actions.
- [x] JSON diagnostics/test/smoke/Eco schemas are stable and validated.

## UI

- [x] `view`, `state`, binding, events, commands, typed styles, and
      accessibility metadata are supported.
- [x] Web UI backend builds through `wasm32-web`.
- [x] Native shell UI backend builds on supported host platforms.
- [x] UI examples have web and native smoke coverage.

## Eco

- [x] Capsule manifest v1, dependency resolver, permission model, semantic
      lockfile, local Todex Vault, Seed import/export, NeedMap, TrustSnapshot,
      Materializer, reproducible build basics, and API diff checker are stable.
- [x] Package publishing, TetraHub, target-aware downloads, and trust metadata
      are available as explicitly labeled beta features.
- [x] Full distributed Todex mesh, proof-carrying capsules, global EcoTrust,
      EcoOracle, and live evolution remain documented as post-1.0.

## Required Commands

- [x] `go test ./compiler/... ./cli/... ./tools/...`
- [x] `bash scripts/test_all.sh --full`
- [x] `bash scripts/release_v1_0_gate.sh`
- [x] Native host smoke runs.
- [x] Build-only smoke passes for all mandatory native and WASM targets.
- [x] WASI smoke runs in a WASI runner.
- [x] Web UI smoke loads through browser automation.
- [x] Docs manifest and doctests verify.
- [x] API diff checker verifies stable public APIs.
- [x] Reproducible build check passes for at least one native and one WASM
      target.

Open blockers (exact):
- none on the current v1.0 release snapshot (all required commands above are passing).
