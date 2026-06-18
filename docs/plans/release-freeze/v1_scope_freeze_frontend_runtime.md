# v1 Scope Freeze: Frontend, Type, Ownership, Effects, and Async Runtime

Date: 2026-04-26

This document closes historical frontend/type/ownership/effects/async-runtime
scope items now summarized by `docs/spec/v1_scope.md`.

Classification labels used below:

- `implemented-now`
- `deferred-post-v1`
- `blocked-by-prerequisite`

## Explicit v1 Decisions

1. Final Flow grammar for v1 is frozen now: Flow indentation syntax, expression
   bodies, `else if` spelling, current `uses` surface, and diagnostics-only
   behavior for deferred syntax (closures and semantic clauses).
2. Argument labels on regular function calls remain out of v1 scope; only the
   current type-like constructor label parsing remains in the v1 surface.
3. Multi-slot optionals and multi-slot typed errors are deferred post-v1.
4. Cross-module generic functions and extension conformance clauses are blocked
   behind prerequisite design and symbol/lowering work.
5. Monomorphization naming is blocked until a deterministic mangling contract is
   specified and tested across modules.
6. Full local lifetime/borrow-scope modeling is blocked until a formal checker
   model is approved.
7. Actor/task transfer rules are blocked until sendability and ownership
   transfer semantics are defined against that lifetime model.
8. v1 effects propagation set is frozen to explicit function-level `uses` checks
   over the current MVP effect names (`alloc`, `mem`, `io`, `mmio`, `islands`,
   `capability`, `link`, `control`, `runtime`, `actors`) without generic or
   protocol effect polymorphism.
9. Budget and privacy clauses are not part of enforced v1 language semantics:
   they remain deferred or blocked by prerequisites documented below.
10. v1 async task ABI is frozen to current cooperative runtime MVP entry points
    (`core.task_spawn_i32`, `core.task_join_i32`); task groups, cancellation,
    typed handles, and typed async errors are post-v1.

## Unresolved Item Freeze Table

### 7.1 - Define the final Flow-only grammar for v1.0

- Classification: `implemented-now`.
- v1 freeze decision: grammar is frozen to the current documented Flow MVP
  surface plus existing planned-feature diagnostics for excluded syntax.
- Concrete prerequisite: n/a.
- Future verification command:

  ```sh
  go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
  go test ./compiler/internal/frontend ./compiler/... \
    -run 'Flow|Parser|Lexer|Format'
  ```

### 7.2 - Finish argument labels

- Classification: `deferred-post-v1`.
- v1 freeze decision: full call-site argument labels are explicitly out of v1
  scope.
- Concrete prerequisite: final labeled-call grammar disambiguation and lowering
  rules that do not conflict with constructor labels.
- Future verification command:

  ```sh
  go test ./compiler/internal/frontend ./compiler/... \
    -run 'Flow|Parser|Lexer|Format|Call'
  ```

### 8.1 - Complete multi-slot optionals

- Classification: `deferred-post-v1`.
- v1 freeze decision: v1 keeps one-slot optional payload semantics only.
- Concrete prerequisite: stable multi-slot value layout and ABI contract for
  optionals in frontend, lowering, and backend.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Optional|Layout|Lower'
  ```

### 8.2 - Complete multi-slot typed errors

- Classification: `deferred-post-v1`.
- v1 freeze decision: v1 keeps one-slot success/error typed-throws semantics
  only.
- Concrete prerequisite: shared multi-slot result ABI for `throws`, `try`,
  lowering, and runtime interop.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'TypedError|Throws|Try|Lower'
  ```

### 8.3 - Support generic functions across modules

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: not in v1 until cross-module generic metadata and
  instantiation rules are stable.
- Concrete prerequisite: module export/import metadata for generic signatures
  plus cross-module instantiation pipeline.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Generic|Module|Import|Monomorph'
  ```

### 8.4 - Add extension conformance clauses

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: not in v1 pending conformance-resolution design beyond
  the current MVP.
- Concrete prerequisite: extension-conformance syntax and resolution rules
  integrated with protocol checker and module lookup.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Extension|Protocol|Conformance'
  ```

### 8.5 - Stabilize monomorphization names

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: v1 does not promise stable public monomorph symbol names.
- Concrete prerequisite: deterministic mangling spec with module path,
  canonical type args, and golden tests.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Monomorph|Mangle|Generic'
  ```

### 9.1 - Model local lifetimes and borrow scopes

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: full lifetime solver is out of v1 implementation scope.
- Concrete prerequisite: approved local lifetime model and borrow-scope
  algorithm for checker integration.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Ownership|Borrow|Lifetime|Region'
  ```

### 9.2 - Define safe transfer rules for actor/task boundaries

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: v1 keeps current scoped-island safety slice; no expanded
  actor/task transfer contract.
- Concrete prerequisite: sendability/transfer model aligned with ownership
  regions and async runtime ABI.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Ownership|Actor|Task|Island'
  ```

### 10.1 - Extend `uses` into effect groups

- Classification: `deferred-post-v1`.
- v1 freeze decision: v1 keeps explicit flat effect names, no group
  abstraction.
- Concrete prerequisite: effect-group taxonomy and diagnostics contract that
  preserves current capability checks.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Effect|Capability|Unsafe'
  ```

### 10.2 - Propagate effects through generics

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: out of v1 until effect-polymorphic generics are
  specified.
- Concrete prerequisite: effect-polymorphic type parameter model and
  inference/instantiation rules.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Effect|Generic|Inference'
  ```

### 10.3 - Propagate effects through protocols

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: out of v1 until protocol effect requirements are
  specified.
- Concrete prerequisite: protocol effect requirement syntax and
  conformance-check propagation rules.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Effect|Protocol|Conformance'
  ```

### 10.4 - Add capability attenuation

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: out of v1 pending capability-subsetting model.
- Concrete prerequisite: capability lattice/attenuation semantics with static
  checker rules.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Capability|Effect|Unsafe'
  ```

### 10.5 - Add capsule permission checks

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: out of v1 pending capsule semantics.
- Concrete prerequisite: capsule declaration semantics and permission-check
  implementation points.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Capability|Capsule|Permission'
  ```

### 10.6 - Add secret/privacy types

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: out of v1; privacy typing is gated behind prerequisite
  design.
- Concrete prerequisite: secret/privacy type model and boundary rules tied to
  capabilities.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Privacy|Capability|Type'
  ```

### 10.7 - Add consent-token MVP

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: out of v1; consent-token enforcement is gated.
- Concrete prerequisite: consent token semantics for acquisition, propagation,
  and use-site checks.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Privacy|Consent|Capability'
  ```

### 10.8 - Add checked privacy clauses

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: out of v1; checked privacy clauses are gated.
- Concrete prerequisite: clause grammar, checker implementation, and runtime
  hooks where required.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Privacy|Clause|Capability'
  ```

### 10.9 - Add policy syntax or explicitly defer

- Classification: `implemented-now`.
- v1 freeze decision: `budget`, `noalloc`, `noblock`, `realtime`, and
  `nothrow` are explicitly deferred from v1 enforcement; parser diagnostics
  posture remains.
- Concrete prerequisite: n/a.
- Future verification command:

  ```sh
  go test ./compiler/internal/frontend ./compiler/... \
    -run 'Flow|Parser|Effect|Budget'
  ```

### 10.10 - Add runtime checks for the rest

- Classification: `deferred-post-v1`.
- v1 freeze decision: v1 does not add new runtime effect/privacy enforcement
  beyond current MVP checks.
- Concrete prerequisite: runtime contract for each dynamic-only policy plus
  ABI-safe error reporting path.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Effect|Capability|Budget|Privacy'
  ./tetra smoke --target linux-x64 --run=true
  ```

### 11.1 - Define the v1.0 task ABI

- Classification: `implemented-now`.
- v1 freeze decision: ABI is frozen to the current cooperative task MVP
  (`core.task_spawn_i32`, `core.task_join_i32`) for v1.
- Concrete prerequisite: n/a.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Async|Task|Actor|Runtime'
  ./tetra build --runtime=selfhost \
    -o reports/actors examples/actors_pingpong.tetra
  ./tetra build --runtime=builtin \
    -o reports/actors_builtin examples/actors_pingpong.tetra
  ```

### 11.2 - Implement structured task groups

- Classification: `deferred-post-v1`.
- v1 freeze decision: structured concurrency groups are out of v1 scope.
- Concrete prerequisite: runtime scheduler/group lifecycle design and ABI
  update for group handles.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Async|Task|Runtime|Group'
  ```

### 11.3 - Implement cancellation

- Classification: `deferred-post-v1`.
- v1 freeze decision: cancellation is out of v1 scope.
- Concrete prerequisite: cancellation token/state model and runtime
  polling/preemption semantics.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Async|Task|Runtime|Cancel'
  ```

### 11.4 - Add typed task handles

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: not in v1 until ABI and type-system hooks are ready.
- Concrete prerequisite: stable task-handle ABI shape and generic/typed handle
  checker rules.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Async|Task|Type|Handle'
  ```

### 11.5 - Add typed async error propagation

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: not in v1 until typed-error and async ABI integration is
  designed.
- Concrete prerequisite: multi-slot typed-error/runtime result contract
  compatible with async lowering.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Async|Task|TypedError|Throws'
  ```

### 11.6 - Expand actors beyond `i32` messages

- Classification: `blocked-by-prerequisite`.
- v1 freeze decision: v1 actor runtime stays on current `i32` message MVP.
- Concrete prerequisite: actor message ABI generalization plus ownership-safe
  transfer checks.
- Future verification command:

  ```sh
  go test ./compiler/... -run 'Actor|Runtime|Ownership|Task'
  ```

## Cross-Reference

- Flow syntax baseline: `docs/spec/flow_syntax_mvp.md`
- v1 feature policy decisions: `docs/spec/v1_feature_status.md`
