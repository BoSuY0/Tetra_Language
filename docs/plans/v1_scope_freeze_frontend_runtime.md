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

| Ref | Unresolved item | Classification | v1 freeze decision | Concrete prerequisite (for deferred/blocked) | Future verification command |
| --- | --- | --- | --- | --- | --- |
| 7.1 | Define the final Flow-only grammar for v1.0 | `implemented-now` | Grammar is frozen to current documented Flow MVP surface plus existing planned-feature diagnostics for excluded syntax. | n/a | `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt && go test ./compiler/internal/frontend ./compiler/... -run 'Flow|Parser|Lexer|Format'` |
| 7.2 | Finish argument labels | `deferred-post-v1` | Full call-site argument labels are explicitly out of v1 scope. | Final labeled-call grammar disambiguation and lowering rules that do not conflict with constructor labels. | `go test ./compiler/internal/frontend ./compiler/... -run 'Flow|Parser|Lexer|Format|Call'` |
| 8.1 | Complete multi-slot optionals | `deferred-post-v1` | v1 keeps one-slot optional payload semantics only. | Stable multi-slot value layout and ABI contract for optionals in frontend + lowering + backend. | `go test ./compiler/... -run 'Optional|Layout|Lower'` |
| 8.2 | Complete multi-slot typed errors | `deferred-post-v1` | v1 keeps one-slot success/error typed-throws semantics only. | Shared multi-slot result ABI for `throws`, `try`, lowering, and runtime interop. | `go test ./compiler/... -run 'TypedError|Throws|Try|Lower'` |
| 8.3 | Support generic functions across modules | `blocked-by-prerequisite` | Not in v1 until cross-module generic metadata and instantiation rules are stable. | Module export/import metadata for generic signatures plus cross-module instantiation pipeline. | `go test ./compiler/... -run 'Generic|Module|Import|Monomorph'` |
| 8.4 | Add extension conformance clauses | `blocked-by-prerequisite` | Not in v1 pending conformance-resolution design beyond current MVP. | Extension-conformance syntax + resolution rules integrated with protocol checker and module lookup. | `go test ./compiler/... -run 'Extension|Protocol|Conformance'` |
| 8.5 | Stabilize monomorphization names | `blocked-by-prerequisite` | v1 does not promise stable public monomorph symbol names. | Deterministic mangling spec (module path + canonical type args) with golden tests. | `go test ./compiler/... -run 'Monomorph|Mangle|Generic'` |
| 9.1 | Model local lifetimes and borrow scopes | `blocked-by-prerequisite` | Full lifetime solver is out of v1 implementation scope. | Approved local lifetime model and borrow-scope algorithm for checker integration. | `go test ./compiler/... -run 'Ownership|Borrow|Lifetime|Region'` |
| 9.2 | Define safe transfer rules for actor/task boundaries | `blocked-by-prerequisite` | v1 keeps current scoped-island safety slice; no expanded actor/task transfer contract. | Sendability/transfer model aligned with ownership regions and async runtime ABI. | `go test ./compiler/... -run 'Ownership|Actor|Task|Island'` |
| 10.1 | Extend `uses` into effect groups | `deferred-post-v1` | v1 keeps explicit flat effect names, no group abstraction. | Effect-group taxonomy and diagnostics contract that preserves current capability checks. | `go test ./compiler/... -run 'Effect|Capability|Unsafe'` |
| 10.2 | Propagate effects through generics | `blocked-by-prerequisite` | Out of v1 until effect-polymorphic generics are specified. | Effect-polymorphic type parameter model and inference/instantiation rules. | `go test ./compiler/... -run 'Effect|Generic|Inference'` |
| 10.3 | Propagate effects through protocols | `blocked-by-prerequisite` | Out of v1 until protocol effect requirements are specified. | Protocol effect requirement syntax and conformance-check propagation rules. | `go test ./compiler/... -run 'Effect|Protocol|Conformance'` |
| 10.4 | Add capability attenuation | `blocked-by-prerequisite` | Out of v1 pending capability-subsetting model. | Capability lattice/attenuation semantics with static checker rules. | `go test ./compiler/... -run 'Capability|Effect|Unsafe'` |
| 10.5 | Add capsule permission checks | `blocked-by-prerequisite` | Out of v1 pending capsule semantics. | Capsule declaration semantics and permission-check implementation points. | `go test ./compiler/... -run 'Capability|Capsule|Permission'` |
| 10.6 | Add secret/privacy types | `blocked-by-prerequisite` | Out of v1; privacy typing is gated behind prerequisite design. | Secret/privacy type model and boundary rules tied to capabilities. | `go test ./compiler/... -run 'Privacy|Capability|Type'` |
| 10.7 | Add consent-token MVP | `blocked-by-prerequisite` | Out of v1; consent-token enforcement is gated. | Consent token semantics for acquisition, propagation, and use-site checks. | `go test ./compiler/... -run 'Privacy|Consent|Capability'` |
| 10.8 | Add checked privacy clauses | `blocked-by-prerequisite` | Out of v1; checked privacy clauses are gated. | Clause grammar + checker implementation + runtime hooks where required. | `go test ./compiler/... -run 'Privacy|Clause|Capability'` |
| 10.9 | Add `budget`, `noalloc`, `noblock`, `realtime`, `nothrow` syntax or explicitly defer | `implemented-now` | Explicitly deferred from v1 enforcement; parser diagnostics posture remains. | n/a | `go test ./compiler/internal/frontend ./compiler/... -run 'Flow|Parser|Effect|Budget'` |
| 10.10 | Add runtime checks for the rest | `deferred-post-v1` | v1 does not add new runtime effect/privacy enforcement beyond current MVP checks. | Runtime contract for each dynamic-only policy plus ABI-safe error reporting path. | `go test ./compiler/... -run 'Effect|Capability|Budget|Privacy' && ./tetra smoke --target linux-x64 --run=true` |
| 11.1 | Define the v1.0 task ABI | `implemented-now` | ABI is frozen to current cooperative task MVP (`core.task_spawn_i32`, `core.task_join_i32`) for v1. | n/a | `go test ./compiler/... -run 'Async|Task|Actor|Runtime' && ./tetra build --runtime=selfhost -o reports/actors examples/actors_pingpong.tetra && ./tetra build --runtime=builtin -o reports/actors_builtin examples/actors_pingpong.tetra` |
| 11.2 | Implement structured task groups | `deferred-post-v1` | Structured concurrency groups are out of v1 scope. | Runtime scheduler/group lifecycle design and ABI update for group handles. | `go test ./compiler/... -run 'Async|Task|Runtime|Group'` |
| 11.3 | Implement cancellation | `deferred-post-v1` | Cancellation is out of v1 scope. | Cancellation token/state model and runtime polling/preemption semantics. | `go test ./compiler/... -run 'Async|Task|Runtime|Cancel'` |
| 11.4 | Add typed task handles | `blocked-by-prerequisite` | Not in v1 until ABI and type-system hooks are ready. | Stable task-handle ABI shape and generic/typed handle checker rules. | `go test ./compiler/... -run 'Async|Task|Type|Handle'` |
| 11.5 | Add typed async error propagation | `blocked-by-prerequisite` | Not in v1 until typed-error and async ABI integration is designed. | Multi-slot typed-error/runtime result contract compatible with async lowering. | `go test ./compiler/... -run 'Async|Task|TypedError|Throws'` |
| 11.6 | Expand actors beyond `i32` messages | `blocked-by-prerequisite` | v1 actor runtime stays on current `i32` message MVP. | Actor message ABI generalization plus ownership-safe transfer checks. | `go test ./compiler/... -run 'Actor|Runtime|Ownership|Task'` |

## Cross-Reference

- Flow syntax baseline: `docs/spec/flow_syntax_mvp.md`
- v1 feature policy decisions: `docs/spec/v1_feature_status.md`
