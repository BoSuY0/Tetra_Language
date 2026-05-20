# v1 Scope Freeze: Eco, Release Gate, and Execution Order

Status: approved scope-decision artifact for historical planning closure.
Date: 2026-04-26.
Applies to: historical Eco/release/execution-order scope items now summarized
by `docs/spec/v1_scope.md`.

## Purpose

This document closes remaining TODO planning points by decision, not by pretending implementation is complete.

Decision labels:
- `implemented-now`: already implemented and covered by current commands.
- `deferred-post-v1`: explicitly out of v1.0 scope; tracked for post-v1 promotion.
- `blocked-by-prerequisite`: remains in v1 intent but cannot be completed until prerequisite gates are real.

## Evidence Baseline

Current command coverage proves local Eco alpha flows and release-gate scaffolding are real:

```sh
bash scripts/ci/test-all.sh --full
bash scripts/release/v1_0/gate.sh
```

`test_all --full` currently covers `eco verify`, lock validation, pack/unpack validation, and vault validation.
`scripts/release/v1_0/gate.sh` intentionally blocks v1 labeling while `./tetra version` is not `v1.0.x`.

## Decision Matrix

| Item | Decision | Prerequisite / scope rule | Concrete gate command |
| --- | --- | --- | --- |
| Capsule manifest v1 stabilization | `blocked-by-prerequisite` | Finalize and version a manifest-v1 contract in parser/validator path before promoting as stable. | `./tetra eco verify --target linux-x64 --lock reports/tetra.lock.json Tetra.capsule && go run ./tools/cmd/validate-eco-lock --lock reports/tetra.lock.json` |
| Permission model stabilization | `blocked-by-prerequisite` | Define enforceable permission semantics and diagnostics for capsule graph validation. | `go test ./cli/... ./tools/... -run 'Eco|Vault|Capsule|Lock' && bash scripts/ci/test-all.sh --full` |
| Seed import/export | `deferred-post-v1` | Keep out of v1.0; only promote in a post-v1 wave with dedicated CLI/tests. | `rg -n "Seed import/export|post-v1" docs/plans/v1_scope_freeze_eco_release.md docs/release_notes_v1_0_draft.md` |
| NeedMap | `deferred-post-v1` | Keep out of v1.0 until data model and compatibility policy are specified. | `rg -n "NeedMap|post-v1" docs/plans/v1_scope_freeze_eco_release.md docs/release_notes_v1_0_draft.md` |
| TrustSnapshot | `deferred-post-v1` | Keep out of v1.0 until trust snapshot format/signing rules exist. | `rg -n "TrustSnapshot|post-v1" docs/plans/v1_scope_freeze_eco_release.md docs/release_notes_v1_0_draft.md` |
| Materializer | `deferred-post-v1` | Keep out of v1.0 until deterministic materialization contract is designed. | `rg -n "Materializer|post-v1" docs/plans/v1_scope_freeze_eco_release.md docs/release_notes_v1_0_draft.md` |
| Reproducible build basics | `blocked-by-prerequisite` | Define reproducibility baseline format and acceptance criteria for one native and one WASM path. | `bash scripts/release/v1_0/gate.sh` |
| Beta package publishing | `deferred-post-v1` | Keep network publishing out of v1.0 release gate; allow only after post-v1 promotion. | `rg -n "deferred-post-v1|beta package publishing" docs/plans/v1_scope_freeze_eco_release.md` |
| TetraHub beta path | `deferred-post-v1` | Keep out of v1.0 release gate pending service/API contract and rollout policy. | `rg -n "TetraHub|deferred-post-v1" docs/plans/v1_scope_freeze_eco_release.md docs/release_notes_v1_0_draft.md` |
| Target-aware downloads | `deferred-post-v1` | Keep out of v1.0 until signed metadata and target policy are finalized. | `rg -n "target-aware downloads|deferred-post-v1" docs/plans/v1_scope_freeze_eco_release.md` |
| Trust metadata | `deferred-post-v1` | Keep out of v1.0 until trust schema and verification path are defined. | `rg -n "trust metadata|deferred-post-v1" docs/plans/v1_scope_freeze_eco_release.md` |
| Distributed Todex mesh / proof-carrying capsules / EcoTrust / EcoOracle / live evolution | `implemented-now` | Already documented as post-v1 unless explicitly promoted. | `rg -n "post-1.0|distributed Todex mesh|EcoTrust|EcoOracle|live evolution" docs/release_notes_v1_0_draft.md docs/checklists/v1_0_release_gate.md` |

## Final v1.0 Release Gate TODO (unresolved bullets) Decisions

| TODO gate item | Decision | Prerequisite / scope rule | Concrete gate command |
| --- | --- | --- | --- |
| Update version only when release branch is ready | `blocked-by-prerequisite` | Do not bump until all mandatory v1 checks are real and green. | `./tetra version && bash scripts/release/v1_0/gate.sh` |
| Regenerate and validate docs manifest | `implemented-now` | Command exists and is already part of full/release checks. | `go run ./tools/cmd/gen-manifest -o reports/manifest.json && go run ./tools/cmd/validate-manifest --manifest reports/manifest.json && go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` |
| Finalize release notes | `blocked-by-prerequisite` | Keep draft language until v1 release branch readiness is proven. | `bash scripts/release/v1_0/gate.sh` |
| Check every release-gate checklist item | `blocked-by-prerequisite` | Checklist cannot be fully checked while blocked capabilities remain. | `bash scripts/release/v1_0/gate.sh` |
| Build-only smoke for all mandatory native and WASM targets | `blocked-by-prerequisite` | Native path exists; WASM targets remain planned in current implementation. | `./tetra smoke --target linux-x64 --run=false --report reports/linux.json && ./tetra smoke --target macos-x64 --run=false --report reports/macos.json && ./tetra smoke --target windows-x64 --run=false --report reports/windows.json && ./tetra smoke --target wasm32-wasi --run=false --report reports/wasi.json && ./tetra smoke --target wasm32-web --run=false --report reports/web.json` |
| Run WASI smoke in WASI runner | `blocked-by-prerequisite` | Requires implemented WASI backend and runner integration. | `bash scripts/release/v1_0/gate.sh` |
| Run web UI smoke through browser automation | `blocked-by-prerequisite` | Requires implemented UI model + web backend + automation harness. | `bash scripts/release/v1_0/gate.sh` |
| Verify docs manifest and doctests | `implemented-now` | Verification path is real today. | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` |
| Verify API diff reports | `implemented-now` | API docs validation path exists and runs in full/release checks. | `go run ./tools/cmd/gen-docs examples > reports/api-docs.md && go run ./tools/cmd/validate-api-docs --docs reports/api-docs.md` |
| Verify reproducible builds for native + WASM | `blocked-by-prerequisite` | Reproducibility policy and WASM implementation are not complete. | `bash scripts/release/v1_0/gate.sh` |

## Suggested Execution Order 1-9 Decisions

| Order item | Decision | Prerequisite / scope rule | Concrete gate command |
| --- | --- | --- | --- |
| 1. Freeze historical green v0.6.0 baseline | `implemented-now` | Keep v0.6 gate authoritative during scope freeze. | `bash scripts/release/v0_6/gate.sh` |
| 2. Finish or explicitly split v0.6.x stabilization tasks | `implemented-now` | This document is the explicit split/closure decision artifact. | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` |
| 3. Validate first v0.7 hardening slice | `implemented-now` | Validation slice already marked complete in TODO 5. | `go test ./compiler/... -run 'Optional|Enum|Match|For|Loop|Const|Else|Compound|Format'` |
| 4. Start v1.0 Wave 1: Flow-only frontend | `implemented-now` | Wave started; remaining feature work tracked separately in Wave 1 TODOs. | `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt` |
| 5. Type system stabilization before ownership/race freedom | `blocked-by-prerequisite` | Ownership/race claims remain blocked until type-stability criteria pass. | `bash scripts/release/v1_0/gate.sh` |
| 6. Ownership/race freedom before safe-code guarantees | `blocked-by-prerequisite` | Safe-code guarantee stays blocked until ownership/race checks are complete. | `bash scripts/release/v1_0/gate.sh` |
| 7. Add WASM before UI web release checks | `blocked-by-prerequisite` | UI web release checks remain blocked while WASM targets are planned. | `bash scripts/release/v1_0/gate.sh` |
| 8. Stabilize stdlib/tooling before final release notes | `blocked-by-prerequisite` | Release notes remain draft until stdlib/tooling gate items are real. | `bash scripts/release/v1_0/gate.sh` |
| 9. Run final v1.0 gate only after placeholders are real | `blocked-by-prerequisite` | Keep intentional v1 gate refusal until real implementation replaces placeholders. | `bash scripts/release/v1_0/gate.sh` |

## v0.7 Intermediate-Release Decision (TODO line 657)

Decision: `implemented-now` as a scope choice.

- v0.7 remains an internal hardening slice, not an official public release label.
- Public release labeling remains `v0.1.x` until v1.0 gate criteria are genuinely satisfied.

Concrete gate commands:

```sh
bash scripts/ci/test-all.sh --full
bash scripts/release/v1_0/gate.sh
```

The first command preserves the active public baseline; the second enforces that v1 cannot be labeled early.

## Checklist-Closure Rule

For unresolved TODO/checklist bullets covered by this artifact:

- close as `implemented-now` when command coverage is already real;
- close as `deferred-post-v1` when scope intentionally excludes the feature from v1.0;
- close as `blocked-by-prerequisite` when the item stays in v1 intent but cannot complete yet.

This closes planning ambiguity now without claiming unfinished implementation is done.
