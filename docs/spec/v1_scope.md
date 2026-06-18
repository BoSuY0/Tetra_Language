# Tetra v1.0 Canonical Scope

Status: pre-release scope contract. This document defines what must be true before a build can be
labeled `v1.0.0`; it is not a claim that the current `v0.4.0` profile, or any separately gated
post-v0.4 production evidence, already satisfies the scope.

The current release gate is `scripts/release/v0_4_0/gate.sh`, with separate post-v0.4 Linux-x64
Memory/Parallelism/UI gates under `scripts/release/post_v0_4/`. A true `v1.0.0` gate remains
`scripts/release/v1_0/gate.sh` and must close from this contract when the version is promoted to
`v1.0.x` and every mandatory artifact below has fresh evidence. The matching release checklist is
`docs/checklists/v1_0_release_gate.md`, and the final evidence handoff schema is
`docs/release/v1_0_final_handoff.md`.

In this document, `Required` means required before a future `v1.0.0` release label can close. It
does not promote any `planned` feature-registry entry, such as `language.full-v1-guarantees`, into
current support. Entries that are already current in the `v0.4.0` manifest, such as `ui.metadata-v1`
and `wasm.runtime-execution`, keep their registry-limited scope and do not close the full v1 target
matrix.

## Mandatory Language Scope

Each item below records the v1.0 decision, evidence, gates, and owner.

- Flow syntax as canonical source syntax:
  - Decision: Required.
  - Evidence: Flow-only scan and formatter check over release source roots.
  - Gates:
    - `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`
    - `./tetra fmt --check examples lib __rt compiler/selfhostrt`
  - Owner: frontend agent.
- Parser and diagnostics for supported Flow forms:
  - Decision: Required.
  - Evidence: frontend parser/diagnostic tests and docs verification.
  - Gate: `go test ./compiler/internal/frontend/... -count=1`.
  - Owner: frontend agent.
- Function-type/callable Level 0 MVP boundary:
  - Decision: required as constrained MVP.
  - Evidence: `fn(T...) -> R` parsing/checking and direct-local callable subset.
  - Non-goal: full first-class function-value behavior.
  - Gate terms: `Closure`, `FunctionType`, `Callable`, `Type`.
  - Owner: frontend/semantics agent.
- Callable Level 1 non-capturing expansion:
  - Decision: experimental until promoted.
  - Evidence: symbol-backed non-capturing callable expansion stays labeled.
  - Gate terms: `Closure`, `FunctionType`, `Callable`.
  - Docs gate: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
  - Owner: frontend/semantics/docs agents.
- Callable Level 2 captured closure and escape model:
  - Decision: planned/experimental.
  - Evidence: future compiler/runtime ABI gates plus docs verification.
  - Non-goal: full first-class function values in the current baseline.
  - Owner: frontend/semantics/runtime agents.
- Top-level `capsule` metadata declaration MVP:
  - Decision: required as metadata-only surface.
  - Evidence: parser/semantic validation for capsule key/value metadata.
  - Gates: frontend package tests plus compiler tests for `Capsule` and `Property`.
  - Owner: frontend/semantics agent.
- Static monomorphized generic functions:
  - Decision: required as constrained MVP.
  - Evidence: parsed, checked, formatted, documented, and static specialization.
  - Non-goals: explicit type args, generic structs, higher-ranked generics.
  - Gate terms: `Generic`, `Monomorph`, `Module`, `Inference`.
  - Owner: semantics/docs agents.
- Static protocol conformance:
  - Decision: required as constrained MVP.
  - Evidence: protocol declarations and `impl Type: Protocol` are static checked.
  - Non-goals: witness tables, trait objects, and dynamic dispatch.
  - Gate terms: `Protocol`, `Conformance`, `Extension`, `Generic`.
  - Owner: semantics/docs agents.
- Primitive, structural, optional, typed-error, enum, extension, and module contracts:
  - Decision: required as the promoted positional enum payload slice only.
  - Evidence: compiler tests and spec alignment for same-module enum payloads.
  - Non-goals: richer constructors, nested destructuring, and guard expansion.
  - Gate terms: `Type`, `Inference`, `Enum`, `Optional`, `Extension`, `Module`.
  - Owner: semantics agent.
- Ownership markers MVP:
  - Decision: required as conservative MVP.
  - Evidence: `borrow`, `inout`, and `consume` call-site marker checks.
  - Gate terms: `Ownership`, `Borrow`, `Consume`, `Inout`.
  - Owner: safety/docs agents.
- Resource lifetime MVP:
  - Decision: required as conservative MVP.
  - Evidence: task, island, region slice, and containing-struct lifetime checks.
  - Gate terms: `Lifetime`, `Resource`, `Island`, `Task`.
  - Owner: safety/docs agents.
- Actor/task transfer safety MVP:
  - Decision: required as conservative local MVP.
  - Evidence: worker entrypoint, sendable-result, transfer, and use-after-transfer.
  - Non-goals: distributed actors and full race-safety proofs.
  - Gate terms: `Actor`, `Task`, `Ownership`, `Transfer`.
  - Owner: safety/runtime/docs agents.
- Lifetime SSA local join solver:
  - Decision: current since `v0.4.0`.
  - Evidence: local/control-flow snapshots for consume and finalization state.
  - Non-goals: richer interprocedural proofs and broad alias/race proofs.
  - Gate terms: `Ownership`, `Borrow`, `Consume`, `Inout`, `Lifetime`, `Task`.
  - Owner: safety agent.
- Ownership, lifetime, island, actor/task transfer, and race-safety checks:
  - Decision: required before release label.
  - Evidence: negative tests for moves, borrows, transfers, and actor/task races.
  - Gate terms: ownership, effects, privacy, capability, budget, MMIO, and memory.
  - Owner: safety/docs agents.
- Effects, capabilities, unsafe boundaries, and public diagnostics:
  - Decision: Required.
  - Evidence: specs, module effect metadata audit, and diagnostic shape tests.
  - Gate terms: `Unsafe`, `Capability`, `Effect`, `Privacy`, `Consent`, `Mem`.
  - Extra gate: `go test ./tools/cmd/validate-diagnostic/... -count=1`.
  - Owner: safety/tooling agent.
- Privacy, consent, and budget contract:
  - Decision: required as static v1 MVP.
  - Evidence: privacy clauses, consent-token signatures, and budget guards.
  - Gate terms: `Privacy`, `Consent`, `Budget`, `Effect`.
  - Owner: safety/tooling agent.
- Async function MVP:
  - Decision: required as checked synchronous lowering.
  - Evidence: `async func`, `await`, and supported `try await <call>()`.
  - Gate terms: `Async`, `Await`, `Task`, `TypedError`.
  - Owner: runtime agent.
- Task runtime MVP:
  - Decision: required for local typed task handles.
  - Evidence: spawn/join/group builtins, `uses runtime`, docs, and stress tests.
  - Gate terms: `Task`, `Runtime`, `Async`, `Stress`.
  - Owner: runtime agent.
- Actors runtime MVP:
  - Decision: required for local actor runtime on supported native targets.
  - Evidence: tagged messages, runtime selection, parity, ownership, and targets.
  - Non-goals: non-Linux-x64 distributed actors and broad structured concurrency.
  - Gate terms: `Actor`, `Actors`, `Runtime`, `Ownership`.
  - Owner: runtime agent.
- Runtime ABI and TOBJ linking:
  - Decision: Required.
  - Evidence: reserved symbols, TOBJ metadata, overrides, and diagnostics.
  - Gate terms: `Runtime`, `ABI`, `Object`, `Link`.
  - Owner: runtime agent.
- UI syntax and accessibility metadata:
  - Decision: required as metadata UI surface.
  - Evidence: UI specs/tests plus native shell sidecar and web smoke evidence.
  - Gate terms: `UI`, `View`, `State`, `Style`, `Accessibility`, `NativeShell`.
  - Extra gates: `bash scripts/release/v1_0/web-smoke.sh`.
  - Extra gate: `./tetra smoke --target linux-x64 --run=false`.
  - Owner: UI agent.

## Mandatory Tooling, CLI, LSP, Docs, And Eco Scope

- CLI commands `check`, `build`, `run`, `fmt`, `test`, `doc`, `lsp`, `eco`,
  `clean`, and `version`:
  - Decision: Required.
  - Evidence: CLI package tests and release gate command coverage.
  - Gates: `go test ./cli/... -count=1`; `bash scripts/release/v0_4_0/gate.sh`.
  - Future gate: v1 gate is blocked until promotion.
  - Owner: CLI agent.
- Formatter contract:
  - Decision: Required.
  - Evidence: idempotence and comment-preservation coverage.
  - Gate terms: `Format`, `Formatter`, `Comment`.
  - Extra gate: `./tetra fmt --check examples lib __rt compiler/selfhostrt`.
  - Owner: tooling agent.
- Docs manifest, doctests, and generated API docs:
  - Decision: Required.
  - Evidence: manifest validation, docs verification, and API docs validation.
  - Gate: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
  - Extra gate: `go run ./tools/cmd/validate-api-docs --docs <generated-docs>`.
  - Owner: docs agent.
- JSON diagnostics, test reports, target reports, doctor reports, and smoke reports:
  - Decision: Required.
  - Evidence: schema validator tests and release gate validator steps.
  - Gates: `go test ./tools/... -count=1`; `bash scripts/ci/test-all.sh --full`.
  - Owner: tools agent.
- LSP stdio baseline:
  - Decision: Required.
  - Evidence: LSP validator and transcript coverage.
  - Gate: `go test ./tools/cmd/validate-lsp-stdio/...`.
  - Extra gate: `go test ./tools/cmd/validate-lsp-smoke/... -count=1`.
  - Owner: LSP agent.
- Local Eco package lifecycle:
  - Decision: Required.
  - Evidence: capsule verify/pack/unpack/vault/publish metadata fixtures.
  - Extra evidence: lock generation and validation through `--lock` workflows.
  - Gate: relevant Eco validator tests.
  - Extra gate: `bash scripts/ci/test-all.sh --full --keep-going`.
  - Owner: Eco agent.

## Evidence Artifact Map

Every mandatory v1 feature row must map to a fresh command result and a concrete artifact path
before the release checklist can close. Paths below are the expected archive locations under the
same `<report-dir>` used by `bash scripts/release/v1_0/gate.sh --report-dir <report-dir>`.

- Frontend:
  - Feature rows: Flow syntax, parser diagnostics, formatter, callable/capsule.
  - Commands: frontend tests, flow-only validation, and formatter check.
  - Artifacts: `<report-dir>/logs/*frontend*`, `*flow-only*`, and `*formatter*`.
- Semantics:
  - Feature rows: types, generics, protocols, enums, modules, typed errors.
  - Commands: compiler semantic tests and docs verification.
  - Artifacts: `<report-dir>/logs/*semantic*`, `*docs*`, and `tetra-docs.md`.
- Safety:
  - Feature rows: ownership, lifetimes, actors, effects, privacy, budgets, memory.
  - Commands: compiler safety tests, docs verification, and diagnostic validation.
  - Artifacts: `<report-dir>/logs/*safety*`, `*docs*`, and `*diagnostic*.json`.
- Runtime:
  - Feature rows: async, tasks, actors, runtime ABI, TOBJ linking.
  - Commands: runtime compiler tests and Linux host smoke when supported.
  - Artifacts: `<report-dir>/logs/*runtime*` and runtime smoke JSON files.
- Backend:
  - Feature rows: native targets, WASI/Web preflight, UI metadata, target smoke.
  - Commands: backend compiler tests plus v1 WASI and web smoke scripts.
  - Artifacts: backend logs and `wasm32-*`, WASI, and web UI smoke JSON.
- CLI/tools:
  - Feature rows: CLI contracts, JSON reports, validators, release-state audit.
  - Commands: CLI tests, tools tests, and release-state validator.
  - Artifacts: CLI/tools logs, release-state text, and JSON reports.
- Docs/LSP/Eco:
  - Feature rows: docs manifest, API docs, examples index, LSP, local Eco.
  - Commands: docs verifier, API-doc validator, LSP tests, and test-all report.
  - Artifacts: docs/LSP logs, API docs, and `test-all/summary.json`.

## Target Matrix

- `linux-x64`: native build and host smoke when running on Linux.
  - Evidence: `./tetra smoke --target linux-x64 --run=true --report <path>`.
- `macos-x64`: build-only cross-target smoke.
  - Evidence: `./tetra smoke --target macos-x64 --run=false --report <path>`.
- `windows-x64`: build-only cross-target smoke.
  - Evidence: `./tetra smoke --target windows-x64 --run=false --report <path>`.
- `wasm32-wasi`: artifact/import preflight plus WASI runner smoke.
  - Evidence: `bash scripts/release/v1_0/wasi-smoke.sh --report <path>`.
- `wasm32-web`: artifact/import preflight plus browser runtime smoke.
  - Evidence: `bash scripts/release/v1_0/web-smoke.sh --report <path>`.

## Explicitly Post-v1 Unless Promoted By Review

Promotion requires `docs/release/post_v1_promotion_checklist.md` evidence in the same branch state
as the implementation, tests, docs, gates, compatibility notes, and security review when
applicable.

- Distributed EcoNet and TetraHub production publishing.
- Proof-carrying capsules and global trust scoring.
- EcoOracle, live evolution, time-travel execution, and multiverse optimizer features.
- Advanced AI/model types and model-runtime integration.
- Callable Level 2 captured closure and escape semantics, broader callback movement, and full
  first-class function-value behavior unless promoted with lifetime and ABI evidence.
- Distributed actors beyond the release actor/task safety contract.
- Async typed-error behavior beyond the supported `try await <call>()` synchronous-lowering
  boundary, plus cancellation and structured concurrency.
- Runtime generic values, generic structs, explicit type arguments, higher-ranked generics, full
  protocol-bound generic dispatch, and specialization optimization beyond the static monomorphized
  generic-function MVP.
- Protocol witness tables, trait objects, runtime protocol values, protocol existential containers,
  and dynamic dispatch beyond static conformance checks.
- Advanced ADT work beyond the promoted positional enum payload slice: arbitrary constructors,
  nested destructuring patterns, guard expansion, richer payload pattern algebra, and match/catch
  coverage outside the gated enum payload promotion.
- Distributed privacy/consent enforcement and runtime-wide resource-budget accounting beyond
  deterministic local guard lowering.
- Real macOS/Windows host execution evidence for actor/runtime binaries when collecting it from
  non-matching Linux hosts.
- Cross-platform native widget rendering, platform accessibility integration, and runtime UI event
  dispatch/layout beyond the Linux-x64 post-v0.4 desktop runtime evidence and the UI v0.4.0
  metadata artifacts in `docs/spec/ui_v0.4.0.md`.
- Any feature still labeled `planned`, `beta`, `deferred-post-v1`, or `blocked-by-prerequisite` in
  the release checklist.

## Release Closure Rule

The release checklist, release notes, and artifact archive must cite this document. A checkbox may
be marked complete only when the implementation, tests, documentation, and artifact evidence exist
in the same branch state.
