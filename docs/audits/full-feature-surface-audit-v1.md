# Full Feature Surface Audit v1

Status: P22.0 evidence/report contract.

Schema: `tetra.language.feature_surface_audit.v1`
Scope: `p22.0_full_feature_surface_audit`

P22.0 records the current feature surface before Phase K promotion work. It
uses `FeatureRegistry()` as the canonical status source and copies those
statuses into report rows so the audit can catch drift between docs, manifest
truth, and implementation evidence.

This audit does not promote language behavior. It classifies each P22.0 surface
as bounded current support, static-only support, experimental gate,
unsupported, planned, or post-v1 until same-branch evidence exists.

## Categories

| Category | Current decision | Registry evidence | Kept boundary |
| --- | --- | --- | --- |
| First-class callables | keep current bounded | `language.callable-mvp`, `language.callable-level1`, `language.callable-level2`, `language.full-first-class-callables` | P22.1 owns any expansion beyond the bounded fnptr fast path and fixed 4-slot callable handle; mutable by-reference capture, pointer/resource capture, thread-boundary callable escape, and dynamic/generic callable movement stay gated. |
| Closures | keep safe by-value capture slice only | `language.callable-level2`, `language.full-first-class-callables` | Pointer/resource capture, mutable capture escape, generic closure capture, and thread movement require fresh ownership/lifetime/ABI evidence. |
| Protocols / trait objects | keep static conformance only | `language.protocol-conformance-mvp`, `language.protocol-bound-generics-static` | Witness tables, trait objects, runtime protocol values, existentials, and dynamic dispatch stay post-v1 unless P22.2 promotes them with same-branch evidence. |
| Runtime generics | keep static monomorphized generic functions only | `language.generics-mvp`, `language.protocol-bound-generics-static` | Runtime generic values, explicit type arguments, generic structs, higher-ranked generics, full protocol-bound dispatch, and broad specialization guarantees stay out of current support. |
| Advanced enums / pattern matching | keep positional enum payload slice only | `language.enum-payload-match` | Advanced ADT constructors, nested destructuring patterns, guard expansion, and richer payload algebra stay future/post-v1. |
| Async typed errors | keep `try await` boundary only | `language.task-handles-mvp`, `language.resource-lifetime-mvp` | Async typed-error behavior beyond `try await <call>()`, `await try`, full cancellation, and structured concurrency are not promoted. |
| Structured concurrency | keep local task/actor bounded | `actors.task-transfer-safety`, `language.task-handles-mvp`, `actors.distributed-runtime` | Existing local transfer/task evidence and Linux-x64 distributed actor evidence do not imply full cancellation, full race-safety proof, or broader structured-concurrency guarantees. |
| Modules / packages | keep local module/package/capsule metadata surface | `language.globals-properties-capsule-mvp`, `eco.local-package-lifecycle` | Capsule declarations are compile-time metadata; distributed EcoNet and proof-carrying capsule behavior remain post-v1. |
| Macros / metaprogramming | keep absent post-v1 | no current registry feature | No macro/metaprogramming system is current. Promotion requires a new registry ID plus parser, semantics, tooling, docs, manifest, and non-claim evidence in the same branch. |
| UI / Surface | keep Linux/web bounded and platform gate experimental | `ui.metadata-v1`, `ui.surface-core`, `ui.surface-linux-x64`, `ui.surface-web-wasm`, `ui.native-runtime`, `ui.platform-runtime`, unsupported Surface host rows | Linux-x64 and wasm32-web evidence stays bounded; macOS, Windows, wasm32-wasi, platform accessibility integration, and cross-platform production UI require real target-host reports. |
| Eco / capsules | keep local Eco and capsule metadata current, distributed Eco post-v1 | `language.globals-properties-capsule-mvp`, `eco.local-package-lifecycle`, `eco.distributed-network` | Distributed EcoNet, production publishing, global trust scoring, and proof-carrying capsules remain post-v1. |

## Validator Contract

`ValidateP22FeatureSurfaceAudit` rejects:

- missing or duplicate categories;
- unknown feature IDs or registry status drift;
- placeholder evidence;
- any report row that promotes a feature without same-branch evidence;
- full v1 guarantee claims;
- runtime generic value claims;
- trait object or runtime protocol value claims;
- macro/metaprogramming system claims;
- full structured-concurrency claims;
- cross-platform production UI runtime claims;
- distributed EcoNet or proof-carrying capsule claims;
- performance claims;
- safe-program semantic changes.

## Non-claims

- No full v1 language guarantee is claimed.
- No runtime generic values are claimed.
- No trait objects or runtime protocol values are claimed.
- No macro/metaprogramming system is claimed.
- No full structured concurrency guarantee is claimed.
- No cross-platform production UI runtime is claimed.
- No distributed EcoNet or proof-carrying capsule promotion is claimed.
- No performance claim is made.
- Safe-program semantics do not change.

