# v1 Scope Freeze: Eco, Release Gate, and Execution Order

Status: approved scope-decision artifact for historical planning closure.
Date: 2026-04-26.
Applies to: historical Eco/release/execution-order scope items now summarized
by `docs/spec/v1_scope.md`.

## Purpose

This document closes remaining TODO planning points by decision. It does not
pretend unfinished implementation is complete.

Decision labels:

- `implemented-now`: already implemented and covered by current commands.
- `deferred-post-v1`: explicitly out of v1.0 scope; tracked for post-v1
  promotion.
- `blocked-by-prerequisite`: remains in v1 intent but cannot be completed until
  prerequisite gates are real.

## Evidence Baseline

Current command coverage proves local Eco alpha flows and release-gate
scaffolding are real:

```sh
bash scripts/ci/test-all.sh --full
bash scripts/release/v1_0/gate.sh
```

`test_all --full` currently covers `eco verify`, lock validation, pack/unpack
validation, and vault validation.

`scripts/release/v1_0/gate.sh` intentionally blocks v1 labeling while
`./tetra version` is not `v1.0.x`.

## Decision Matrix

### Capsule manifest v1 stabilization

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  finalize and version a manifest-v1 contract in parser/validator path before
  promoting as stable.
- Gate:

```sh
./tetra eco verify \
  --target linux-x64 \
  --lock reports/tetra.lock.json \
  Tetra.capsule
go run ./tools/cmd/validate-eco-lock \
  --lock reports/tetra.lock.json
```

### Permission model stabilization

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  define enforceable permission semantics and diagnostics for capsule graph
  validation.
- Gate:

```sh
go test ./cli/... ./tools/... \
  -run 'Eco|Vault|Capsule|Lock'
bash scripts/ci/test-all.sh --full
```

### Seed import/export

- Decision: `deferred-post-v1`.
- Scope rule:
  keep out of v1.0; only promote in a post-v1 wave with dedicated CLI/tests.
- Gate:

```sh
rg -n \
  "Seed import/export|post-v1" \
  docs/plans/v1_scope_freeze_eco_release.md \
  docs/release_notes_v1_0_draft.md
```

### NeedMap

- Decision: `deferred-post-v1`.
- Scope rule:
  keep out of v1.0 until data model and compatibility policy are specified.
- Gate:

```sh
rg -n \
  "NeedMap|post-v1" \
  docs/plans/v1_scope_freeze_eco_release.md \
  docs/release_notes_v1_0_draft.md
```

### TrustSnapshot

- Decision: `deferred-post-v1`.
- Scope rule:
  keep out of v1.0 until trust snapshot format/signing rules exist.
- Gate:

```sh
rg -n \
  "TrustSnapshot|post-v1" \
  docs/plans/v1_scope_freeze_eco_release.md \
  docs/release_notes_v1_0_draft.md
```

### Materializer

- Decision: `deferred-post-v1`.
- Scope rule:
  keep out of v1.0 until deterministic materialization contract is designed.
- Gate:

```sh
rg -n \
  "Materializer|post-v1" \
  docs/plans/v1_scope_freeze_eco_release.md \
  docs/release_notes_v1_0_draft.md
```

### Reproducible build basics

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  define reproducibility baseline format and acceptance criteria for one native
  and one WASM path.
- Gate: `bash scripts/release/v1_0/gate.sh`.

### Beta package publishing

- Decision: `deferred-post-v1`.
- Scope rule:
  keep network publishing out of v1.0 release gate; allow only after post-v1
  promotion.
- Gate:

```sh
rg -n \
  "deferred-post-v1|beta package publishing" \
  docs/plans/v1_scope_freeze_eco_release.md
```

### TetraHub beta path

- Decision: `deferred-post-v1`.
- Scope rule:
  keep out of v1.0 release gate pending service/API contract and rollout policy.
- Gate:

```sh
rg -n \
  "TetraHub|deferred-post-v1" \
  docs/plans/v1_scope_freeze_eco_release.md \
  docs/release_notes_v1_0_draft.md
```

### Target-aware downloads

- Decision: `deferred-post-v1`.
- Scope rule:
  keep out of v1.0 until signed metadata and target policy are finalized.
- Gate:

```sh
rg -n \
  "target-aware downloads|deferred-post-v1" \
  docs/plans/v1_scope_freeze_eco_release.md
```

### Trust metadata

- Decision: `deferred-post-v1`.
- Scope rule:
  keep out of v1.0 until trust schema and verification path are defined.
- Gate:

```sh
rg -n \
  "trust metadata|deferred-post-v1" \
  docs/plans/v1_scope_freeze_eco_release.md
```

### Distributed Todex mesh and post-v1 eco systems

- Item:
  `Distributed Todex mesh / proof-carrying capsules / EcoTrust / EcoOracle`
  plus live evolution.
- Decision: `implemented-now`.
- Scope rule:
  already documented as post-v1 unless explicitly promoted.
- Gate:

```sh
rg -n \
  "post-1.0|distributed Todex mesh|EcoTrust|EcoOracle|live evolution" \
  docs/release_notes_v1_0_draft.md \
  docs/checklists/v1_0_release_gate.md
```

## Final v1.0 Release Gate TODO Decisions

### Update version only when release branch is ready

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  do not bump until all mandatory v1 checks are real and green.
- Gate:

```sh
./tetra version
bash scripts/release/v1_0/gate.sh
```

### Regenerate and validate docs manifest

- Decision: `implemented-now`.
- Scope rule:
  command exists and is already part of full/release checks.
- Gate:

```sh
go run ./tools/cmd/gen-manifest \
  -o reports/manifest.json
go run ./tools/cmd/validate-manifest \
  --manifest reports/manifest.json
go run ./tools/cmd/verify-docs \
  --manifest docs/generated/manifest.json
```

### Finalize release notes

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  keep draft language until v1 release branch readiness is proven.
- Gate: `bash scripts/release/v1_0/gate.sh`.

### Check every release-gate checklist item

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  checklist cannot be fully checked while blocked capabilities remain.
- Gate: `bash scripts/release/v1_0/gate.sh`.

### Build-only smoke for all mandatory native and WASM targets

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  native path exists; WASM targets remain planned in current implementation.
- Gate:

```sh
./tetra smoke --target linux-x64 --run=false --report reports/linux.json
./tetra smoke --target macos-x64 --run=false --report reports/macos.json
./tetra smoke --target windows-x64 --run=false --report reports/windows.json
./tetra smoke --target wasm32-wasi --run=false --report reports/wasi.json
./tetra smoke --target wasm32-web --run=false --report reports/web.json
```

### Run WASI smoke in WASI runner

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  requires implemented WASI backend and runner integration.
- Gate: `bash scripts/release/v1_0/gate.sh`.

### Run web UI smoke through browser automation

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  requires implemented UI model, web backend, and automation harness.
- Gate: `bash scripts/release/v1_0/gate.sh`.

### Verify docs manifest and doctests

- Decision: `implemented-now`.
- Scope rule:
  verification path is real today.
- Gate:

```sh
go run ./tools/cmd/verify-docs \
  --manifest docs/generated/manifest.json
```

### Verify API diff reports

- Decision: `implemented-now`.
- Scope rule:
  API docs validation path exists and runs in full/release checks.
- Gate:

```sh
go run ./tools/cmd/gen-docs examples > reports/api-docs.md
go run ./tools/cmd/validate-api-docs \
  --docs reports/api-docs.md
```

### Verify reproducible builds for native + WASM

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  reproducibility policy and WASM implementation are not complete.
- Gate: `bash scripts/release/v1_0/gate.sh`.

## Suggested Execution Order 1-9 Decisions

### 1. Freeze historical green v0.6.0 baseline

- Decision: `implemented-now`.
- Scope rule:
  keep v0.6 gate authoritative during scope freeze.
- Gate: `bash scripts/release/v0_6/gate.sh`.

### 2. Finish or explicitly split v0.6.x stabilization tasks

- Decision: `implemented-now`.
- Scope rule:
  this document is the explicit split/closure decision artifact.
- Gate:

```sh
go run ./tools/cmd/verify-docs \
  --manifest docs/generated/manifest.json
```

### 3. Validate first v0.7 hardening slice

- Decision: `implemented-now`.
- Scope rule:
  validation slice already marked complete in TODO 5.
- Gate:

```sh
go test ./compiler/... \
  -run 'Optional|Enum|Match|For|Loop|Const|Else|Compound|Format'
```

### 4. Start v1.0 Wave 1: Flow-only frontend

- Decision: `implemented-now`.
- Scope rule:
  wave started; remaining feature work is tracked separately in Wave 1 TODOs.
- Gate:

```sh
go run ./tools/cmd/validate-flow-only \
  examples \
  lib \
  __rt \
  compiler/selfhostrt
```

### 5. Type system stabilization before ownership/race freedom

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  ownership/race claims remain blocked until type-stability criteria pass.
- Gate: `bash scripts/release/v1_0/gate.sh`.

### 6. Ownership/race freedom before safe-code guarantees

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  safe-code guarantee stays blocked until ownership/race checks are complete.
- Gate: `bash scripts/release/v1_0/gate.sh`.

### 7. Add WASM before UI web release checks

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  UI web release checks remain blocked while WASM targets are planned.
- Gate: `bash scripts/release/v1_0/gate.sh`.

### 8. Stabilize stdlib/tooling before final release notes

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  release notes remain draft until stdlib/tooling gate items are real.
- Gate: `bash scripts/release/v1_0/gate.sh`.

### 9. Run final v1.0 gate only after placeholders are real

- Decision: `blocked-by-prerequisite`.
- Scope rule:
  keep intentional v1 gate refusal until real implementation replaces
  placeholders.
- Gate: `bash scripts/release/v1_0/gate.sh`.

## v0.7 Intermediate-Release Decision (TODO line 657)

Decision: `implemented-now` as a scope choice.

- v0.7 remains an internal hardening slice, not an official public release
  label.
- Public release labeling remains `v0.1.x` until v1.0 gate criteria are
  genuinely satisfied.

Concrete gate commands:

```sh
bash scripts/ci/test-all.sh --full
bash scripts/release/v1_0/gate.sh
```

The first command preserves the active public baseline; the second enforces
that v1 cannot be labeled early.

## Checklist-Closure Rule

For unresolved TODO/checklist bullets covered by this artifact:

- close as `implemented-now` when command coverage is already real;
- close as `deferred-post-v1` when scope intentionally excludes the feature
  from v1.0;
- close as `blocked-by-prerequisite` when the item stays in v1 intent but
  cannot complete yet.

This closes planning ambiguity now without claiming unfinished implementation
is done.
