# v1 Scope Freeze Decisions: Backend, Stdlib, UI

Date: 2026-04-26

This document closes unresolved implementation items from
`docs/plans/2026-04-26-tetra-language-todo.md` (TODO 12, TODO 13, TODO 15;
unresolved checklist lines 405-541) by explicit v1 scope-freeze decisions.

It records scope decisions only. It does not edit the original TODO plan file.

## Decision Labels

- `implemented-now`: completed now in this v1 closure pass.
- `defer post-v1`: intentionally out of v1 scope; revisit in post-v1 planning.
- `block behind prerequisite`: do not start implementation until prerequisites
  are complete, then re-evaluate status.

## Classification Summary

- `implemented-now`: none.
- `defer post-v1`: stdlib promotions (`collections` through `crypto`), UI syntax
  spec authoring.
- `block behind prerequisite`: backend debug/release coverage, WASM
  implementation/smokes, and UI implementation/backends/smokes.

## TODO 12 (Backends/ABI) Unresolved Items

| Item | Decision | Prerequisite(s) | Intended future gate command(s) |
| --- | --- | --- | --- |
| Add debug info support. | `block behind prerequisite` | Target-specific debug-info contract is finalized for native + WASM backend outputs and covered by backend tests. | `go test ./compiler/... -run 'Target|WASM|ABI|Object|Link|Cache|Deterministic'`; `bash scripts/release_v1_0_gate.sh` |
| Add release optimization coverage. | `block behind prerequisite` | Release-profile codegen/runtime expectations are specified and exercised with deterministic regression coverage. | `bash scripts/test_all.sh --full`; `bash scripts/release_v1_0_gate.sh` |
| Implement `wasm32-wasi` target parsing as supported only after backend exists. | `block behind prerequisite` | `wasm32-wasi` backend path emits real artifacts and target metadata validation no longer treats the target as planned-placeholder state. | `./tetra targets --format=json`; `./tetra smoke --target wasm32-wasi --run=false`; `bash scripts/release_v1_0_gate.sh` |
| Implement `wasm32-wasi` codegen/object/link/run path. | `block behind prerequisite` | WASI codegen + packaging + runner integration are complete for the architecture contract in `docs/backend/wasm_architecture.md`. | `./tetra smoke --target wasm32-wasi --run=false`; `./tetra smoke --target wasm32-wasi --run=true`; `bash scripts/release_v1_0_gate.sh` |
| Implement `wasm32-web` codegen/package path. | `block behind prerequisite` | `wasm32-web` module + deterministic loader output exist and follow the `tetra_web_v1` import contract. | `./tetra smoke --target wasm32-web --run=false`; `bash scripts/release_v1_0_gate.sh` |
| Add smoke coverage for both WASM targets. | `block behind prerequisite` | Both WASM targets are implemented and release harness can execute required WASI/web smoke checks. | `./tetra smoke --target wasm32-wasi --run=false`; `./tetra smoke --target wasm32-web --run=false`; `./tetra smoke --target wasm32-wasi --run=true`; `bash scripts/release_v1_0_gate.sh` |

## TODO 13 (Stdlib) Unresolved Items

| Item | Decision | Prerequisite(s) | Intended future gate command(s) |
| --- | --- | --- | --- |
| Promote stable stdlib modules: `collections`, `strings`, `slices`, `math`, `IO`, `filesystem`, `networking`, `async`, `sync`, `testing`, `serialization`, `time`, `crypto interfaces`. | `defer post-v1` | Post-v1 module-promotion slices are approved and each promoted module lands with stable API docs/doctests/examples/effects metadata and API-diff metadata in `lib/core`. | `./tetra fmt --check lib`; `go run ./tools/cmd/gen-docs lib > reports/stdlib-api-docs.md`; `go run ./tools/cmd/validate-api-docs --docs reports/stdlib-api-docs.md`; `go run ./tools/cmd/gen-manifest -o reports/manifest.json`; `go run ./tools/cmd/validate-manifest --manifest reports/manifest.json`; `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`; `bash scripts/release_v1_0_gate.sh` |

## TODO 15 (UI) Unresolved Items

| Item | Decision | Prerequisite(s) | Intended future gate command(s) |
| --- | --- | --- | --- |
| Write a UI syntax/spec document before implementation. | `defer post-v1` | Post-v1 UI language design slice is approved with parser + semantics sign-off and release-check expectations. | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`; `bash scripts/release_v1_0_gate.sh` |
| Implement `view`, `state`, binding, events, commands, typed style, accessibility metadata. | `block behind prerequisite` | UI syntax spec is accepted and UI type/effect/diagnostics contracts are finalized. | `bash scripts/test_all.sh --full`; `bash scripts/release_v1_0_gate.sh` |
| Add web backend through `wasm32-web`. | `block behind prerequisite` | `wasm32-web` backend path from TODO 12 is real and UI runtime/web bindings are defined. | `./tetra smoke --target wasm32-web --run=false`; `bash scripts/release_v1_0_gate.sh` |
| Add native shell backend. | `block behind prerequisite` | Native shell ABI/runtime contract and host-shell harness are finalized for mandatory host targets. | `bash scripts/test_all.sh --full`; `bash scripts/release_v1_0_gate.sh` |
| Add web UI smoke app. | `block behind prerequisite` | UI web backend exists and browser-automation smoke harness is wired into the release gate. | `bash scripts/release_v1_0_gate.sh` |
| Add native shell UI smoke app. | `block behind prerequisite` | Native shell backend exists and native shell smoke harness is wired into the release gate. | `bash scripts/release_v1_0_gate.sh` |

