# Tetra Language v1.0 Implementation Roadmap (Real Delivery)

**Date:** 2026-04-26  
**Current version:** `v0.6.0`  
**Goal:** reach a true `v1.0.x` release state where `bash scripts/release_v1_0_gate.sh` passes without scope-freeze exceptions.

## Rules For This Roadmap

- [ ] A TODO is considered closed only when implementation + tests + release gates pass.
- [ ] `deferred-post-v1`, `blocked-by-prerequisite`, and `scope-freeze` notes do not count as v1.0 completion.
- [ ] Keep `bash scripts/test_all.sh --full` green after every merge batch.
- [ ] No version bump to `v1.0.x` until all mandatory v1.0 checks are green.

## Baseline Commands (Must Stay Green)

- [x] `go test ./compiler/... ./cli/... ./tools/... -count=1`
- [x] `bash scripts/test_all.sh --quick`
- [x] `bash scripts/test_all.sh --full`
- [x] `bash scripts/release_v0_6_gate.sh`
- [x] `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`

---

## Wave 1: Canonical Flow Frontend

### 1.1 Final Grammar

- [ ] Freeze final v1 Flow grammar spec (single canonical grammar source).
- [x] Remove canonical reliance on `normalizeFlowSyntax` from compile/check/fmt paths.
- [ ] Keep normalization only as migration tooling (optional command/tool path), not canonical frontend.

### 1.2 Missing Syntax Features

- [x] Implement function call argument labels without ambiguity with struct constructors.
- [ ] Implement closures (`fn`/`fun` expression form) with parser + semantics + lowering support.
- [ ] Implement semantic clauses needed for v1 syntax (or remove from v1 spec if not required).

### 1.3 Frontend Validation

- [x] Add positive tests for each newly implemented syntax path.
- [x] Add negative tests for each invalid syntax path.
- [ ] Keep formatter coverage for full Flow surface.

**Verification**

```sh
go test ./compiler/internal/frontend ./compiler/... -run 'Flow|Parser|Lexer|Format'
go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
./tetra fmt --check examples lib __rt compiler/selfhostrt
```

---

## Wave 2: Type System Completion

### 2.1 Optionals / Typed Errors

- [ ] Implement multi-slot optionals.
- [ ] Implement multi-slot typed errors.

### 2.2 Generics / Protocols / Extensions

- [ ] Implement generic functions across modules.
- [ ] Implement extension conformance clauses.
- [ ] Stabilize monomorphization naming (deterministic + stable ABI-facing names).

### 2.3 Type-System Hardening

- [ ] Add exhaustive tests for optionals/enums with expanded payload model.
- [ ] Ensure diagnostics are stable for unsupported/invalid generic/protocol paths.

**Verification**

```sh
go test ./compiler/... -run 'Optional|TypedError|Generic|Protocol|Extension|Match|Enum|Inference'
./tetra test --report=json examples
```

---

## Wave 3: Ownership And Race Freedom

### 3.1 Lifetime Model

- [ ] Model local lifetimes and borrow scopes in checker.
- [ ] Enforce escaping-borrow rejection across all relevant flows (not just narrow slices).

### 3.2 Concurrency Safety Rules

- [ ] Define safe transfer rules for actor/task boundaries.
- [ ] Enforce sendability/ownership transfer checks in actor/task APIs.

### 3.3 Ownership Coverage

- [ ] Add complete positive/negative ownership test matrix for borrow/inout/consume/actor/task paths.

**Verification**

```sh
go test ./compiler/... -run 'Ownership|Borrow|Move|Alias|Island|Actor|Task'
```

---

## Wave 4: Effects, Capabilities, Privacy, Budgets

### 4.1 Effects Propagation

- [ ] Extend `uses` into effect groups.
- [ ] Propagate effects through generics.
- [ ] Propagate effects through protocols.

### 4.2 Capability Model

- [ ] Implement capability attenuation.
- [ ] Implement capsule permission checks.

### 4.3 Privacy / Consent / Budgets

- [ ] Implement secret/privacy type system paths needed for v1.
- [ ] Implement consent-token MVP.
- [ ] Implement checked privacy clauses.
- [ ] Implement `budget`, `noalloc`, `noblock`, `realtime`, `nothrow` syntax + semantics.
- [ ] Add runtime checks for non-static guarantees.

**Verification**

```sh
go test ./compiler/... -run 'Effect|Capability|Unsafe|Budget|Privacy'
./tetra smoke --target linux-x64 --run=true
```

---

## Wave 5: Async Runtime And Actors

### 5.1 Runtime Contract

- [ ] Define and implement stable v1 task ABI.
- [ ] Implement structured task groups.
- [ ] Implement cancellation semantics.
- [ ] Implement typed task handles.
- [ ] Implement typed async error propagation.
- [ ] Expand actors beyond `i32` message-only model.

### 5.2 Runtime Validation

- [ ] Keep self-host and builtin runtime paths passing during migration.
- [ ] Add actor/runtime stress coverage.

**Verification**

```sh
go test ./compiler/... -run 'Async|Task|Actor|Runtime'
./tetra build --runtime=selfhost -o reports/actors examples/actors_pingpong.tetra
./tetra build --runtime=builtin -o reports/actors_builtin examples/actors_pingpong.tetra
```

---

## Wave 6: Backend, ABI, WASM

### 6.1 Native Backend Finalization

- [ ] Add debug info support.
- [ ] Add release optimization coverage.
- [x] Keep deterministic build checks stable.

### 6.2 WASM Targets (Mandatory For v1)

- [x] Implement `wasm32-wasi` target parsing as supported.
- [ ] Implement `wasm32-wasi` codegen/object/link/run path.
- [ ] Implement `wasm32-web` codegen/package path.
- [ ] Add smoke coverage for both WASM targets.
- [ ] Validate incremental build/check cache behavior for native and WASM paths.

**Verification**

```sh
./tetra targets --format=json
./tetra smoke --target linux-x64 --run=false
./tetra smoke --target macos-x64 --run=false
./tetra smoke --target windows-x64 --run=false
./tetra smoke --target wasm32-wasi --run=false
./tetra smoke --target wasm32-web --run=false
./tetra smoke --target wasm32-wasi --run=true
go test ./compiler/... -run 'Target|WASM|ABI|Object|Link|Cache|Deterministic'
```

---

## Wave 7: Stable Stdlib v1 Surface

### 7.1 Promotion

- [ ] Promote `collections`.
- [x] Promote `strings`.
- [x] Promote `slices`.
- [x] Promote `math`.
- [ ] Promote `io`.
- [ ] Promote `filesystem`.
- [ ] Promote `networking`.
- [ ] Promote `async`.
- [ ] Promote `sync`.
- [ ] Promote `testing`.
- [ ] Promote `serialization`.
- [ ] Promote `time`.
- [ ] Promote `crypto interfaces`.

### 7.2 Quality Gates

- [x] Require docs for each stable module.
- [x] Require doctests for each stable module.
- [x] Require examples for each stable module.
- [x] Require formatter compliance for each stable module.
- [x] Require effects metadata for each stable module.
- [x] Require API diff compatibility check against baseline.

**Verification**

```sh
./tetra fmt --check lib
go run ./tools/cmd/gen-docs lib > reports/stdlib-api-docs.md
go run ./tools/cmd/validate-api-docs --docs reports/stdlib-api-docs.md
go run ./tools/cmd/gen-manifest -o reports/manifest.json
go run ./tools/cmd/validate-manifest --manifest reports/manifest.json
```

---

## Wave 8: Toolchain And Developer Experience

### 8.1 CLI / LSP / Reports

- [x] Keep `tetra` and `t` entrypoints stable across v1 feature surface.
- [x] Keep diagnostics/test/smoke/docs schemas stable.
- [x] Keep LSP diagnostics/hover/completion/definition/references/rename/formatting/code-actions stable across expanded language surface.

### 8.2 Formatter

- [ ] Keep formatter idempotent for full v1 syntax.
- [ ] Preserve supported line/block comments for full v1 syntax.

**Verification**

```sh
./tetra lsp --stdio-smoke examples/flow_hello.tetra
go test ./compiler/... ./cli/... ./tools/...
```

---

## Wave 9: UI Language (v1 If Required By Product Scope)

- [ ] Finalize UI syntax/spec.
- [ ] Implement `view`.
- [ ] Implement `state`.
- [ ] Implement bindings.
- [ ] Implement events.
- [ ] Implement commands.
- [ ] Implement typed style.
- [ ] Implement accessibility metadata.
- [ ] Add web backend through `wasm32-web`.
- [ ] Add native shell backend.
- [ ] Add web UI smoke app.
- [ ] Add native shell UI smoke app.

**Verification**

```sh
bash scripts/test_all.sh --full
bash scripts/release_v1_0_gate.sh
```

---

## Wave 10: Eco / Publishing

### 10.1 Core Model

- [ ] Stabilize Capsule manifest v1.
- [ ] Stabilize permission model.
- [ ] Implement Seed import/export.
- [ ] Implement NeedMap.
- [ ] Implement TrustSnapshot.
- [ ] Implement Materializer.
- [ ] Add reproducible build basics.

### 10.2 Beta Distribution

- [ ] Add beta package publishing.
- [ ] Add TetraHub beta path.
- [ ] Add target-aware downloads.
- [ ] Add trust metadata.
- [ ] Decide and document which distributed mesh/EcoTrust/EcoOracle features are in v1 vs post-v1.

**Verification**

```sh
./tetra eco verify --target linux-x64 --lock reports/tetra.lock.json Tetra.capsule
./tetra eco pack --project Tetra.capsule -o reports/app.todex
./tetra eco unpack reports/app.todex -C reports/unpacked
./tetra eco vault verify --store .tetra/todex-vault
go test ./cli/... ./tools/... -run 'Eco|Vault|Capsule|Lock|API'
```

---

## Final v1.0 Release Execution

### Release Checklist

- [ ] Update version to `v1.0.x` on release branch only when all mandatory checks pass.
- [ ] Regenerate and validate docs manifest.
- [ ] Finalize release notes.
- [ ] Complete every item in `docs/checklists/v1_0_release_gate.md`.
- [ ] Run build-only smoke for all mandatory native and WASM targets.
- [ ] Run WASI smoke in a WASI runner.
- [ ] Run web UI smoke via browser automation (if UI is in v1 scope).
- [ ] Verify docs manifest + doctests.
- [ ] Verify API diff reports.
- [ ] Verify reproducible build proof for at least one native and one WASM target.

### Mandatory Final Gate

- [ ] `go test ./compiler/... ./cli/... ./tools/...`
- [ ] `bash scripts/test_all.sh --full`
- [ ] `bash scripts/release_v1_0_gate.sh` (must pass fully)

---

## Definition Of Done (v1.0)

- [ ] `./tetra version` returns `v1.0.x` on release branch.
- [ ] `scripts/release_v1_0_gate.sh` passes end-to-end.
- [ ] No mandatory v1 TODO in this roadmap remains open.
- [ ] All generated release artifacts (docs/manifest/release notes/api diff proofs) are current and verified.
