# Effects, Capabilities, Privacy, And Budget v1

This document records the v1 checked surface for effects and policy clauses.

## Effects

Stable effect names are `actors`, `alloc`, `budget`, `capability`, `control`,
`io`, `islands`, `link`, `mem`, `mmio`, `privacy`, and `runtime`.

Stable groups expand deterministically:

| Group | Members |
| --- | --- |
| `effects.memory` | `alloc`, `islands`, `mem` |
| `effects.cap.io` | `capability`, `io`, `mmio` |
| `effects.cap.mem` | `capability`, `mem` |
| `effects.policy` | `budget`, `privacy` |
| `effects.runtime` | `actors`, `control`, `link`, `runtime` |
| `effects.all` | all stable effects |

Function calls propagate callee effects transitively. Missing `uses`
declarations are diagnostics; effect inference is intentionally out of v1
scope.

Stable `lib/core` modules carry a top-level `// Effects:` metadata line. The
docs verifier parses those modules and fails if the metadata does not match the
actual union of public `uses` declarations, so API docs and release review do
not drift from the compiler surface.

## Capabilities And Unsafe

`cap.io` and `cap.mem` are opaque capability tokens. They can only be obtained
inside `unsafe` blocks through `core.cap_io()` and `core.cap_mem()`. Raw memory,
MMIO, pointer arithmetic, symbol-address, context-switch, and manual island
allocation/free operations require both the relevant `uses` effects and an
`unsafe` block.

Capability attenuation groups require explicit capsule permissions for raw
memory or IO when `effects.cap.mem` or `effects.cap.io` are used.

## Privacy And Consent

The v1 privacy MVP is compiler-enforced:

- `uses privacy` requires the semantic clause `privacy`.
- Secret-bearing signatures use `secret.i32`/`SecretInt`.
- Secret-bearing signatures require `consent(<token>)`.
- The referenced consent parameter must exist and have type `consent.token`.
- `core.secret_seal_i32` and `core.secret_unseal_i32` require the privacy
  effect and a consent token.

Runtime secret storage is intentionally minimal; the v1 guarantee is static
auditing and call-shape enforcement, not cryptographic isolation.

## Budget

`budget(<non-negative integer constant>)` requires `uses budget`. Lowering
emits deterministic budget guard instructions for functions with the clause.
Cross-function runtime accounting policy and distributed budget enforcement are
post-v1.

## Async, Task, And Actor Policy Boundaries

The v1 async surface is a checked synchronous lowering MVP: `async func` and
`await` are parsed, type-checked, lowered, and tested, but cancellation,
structured concurrency, and async typed-error behavior beyond the tested
boundary remain post-v1. The supported boundary form is `try await <call>()`
for propagating an async throwing call through the current synchronous lowering
path. The alternate spelling `await try <call>()` intentionally produces a
stable diagnostic pointing to `try await`; no broader async/error runtime ABI is
claimed here.

The task MVP is `core.task_spawn_i32`, `core.task_spawn_group_i32`,
`core.task_join_i32`, `core.task_join_result_i32`, and task group open/cancel/
close. These builtins require `uses runtime`, accept typed handles, and reject
non-literal, async, throwing, wrong-shape, and mutable-global worker targets.

The actor MVP is local-process actor spawn/send/receive with tagged-message
support. Distributed actors and non-host runtime execution evidence remain
release-lab or post-v1 items, not a language guarantee for `v1.0.0`.

## Epic 06 release evidence

The release-blocking safety slice is:

```sh
go test ./compiler/... -run "Effect|Uses|Capability|Unsafe|Ownership|Borrow|Consume|Inout|Island|Region|Privacy|Budget" -count=1
```

That slice includes positive and negative checks for transitive `uses`
propagation, capability attenuation, unsafe-only builtins, ownership transfer,
region escape prevention, privacy consent clauses, and budget lowering guards.
