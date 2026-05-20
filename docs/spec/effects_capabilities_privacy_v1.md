# Effects, Capabilities, Privacy, And Budget v1

This document records the v1 checked surface for effects and policy clauses.

## Effects

Canonical `uses` effect names are `actors`, `alloc`, `budget`, `capability`,
`control`, `io`, `islands`, `link`, `mem`, `mmio`, `privacy`, `runtime`,
`capsule.io`, and `capsule.mem`.

Accepted aliases in `uses` are canonicalized by the checker:

- `cap.io` aliases `io`
- `cap.mem` aliases `mem`

Permission keys are separate from effect aliases:

- `capsule.io` and `capsule.mem` are permission keys, not aliases for `io`/`mem`
- declaring `capsule.io` alone does not satisfy `uses io` requirements
- declaring `capsule.mem` alone does not satisfy `uses mem` requirements

Stable groups expand deterministically:

| Group | Members |
| --- | --- |
| `effects.memory` | `alloc`, `islands`, `mem` |
| `effects.cap.io` | `capability`, `io`, `mmio` |
| `effects.cap.mem` | `capability`, `mem` |
| `effects.policy` | `budget`, `privacy` |
| `effects.runtime` | `actors`, `control`, `link`, `runtime` |
| `effects.all` | `actors`, `alloc`, `budget`, `capability`, `control`, `io`, `islands`, `link`, `mem`, `mmio`, `privacy`, `runtime` |

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
Pointer/MMIO/memory operations require matching `uses` effects, explicit
unsafe syntax where the builtin policy requires it, and the corresponding
capability token argument.

Capability attenuation permission checks apply only when a function declares an
attenuation group (`effects.cap.mem`, `effects.cap.io`, or `effects.all`) and
then calls attenuation-sensitive builtins. In that mode:

- memory attenuation diagnostics reference `capsule.mem`
- IO attenuation diagnostics reference `capsule.io`
- declaring `mem`/`io` explicitly (including `cap.mem`/`cap.io` aliases)
  satisfies the boundary without an additional capsule permission
- group-expanded `mem`/`io` from attenuation groups are not treated as explicit
  for this bypass; declare `mem`/`io` directly to bypass the capsule permission
- declaring `capsule.mem`/`capsule.io` alone does not clear attenuation
  diagnostics in the current checker path

Top-level source-language `capsule` metadata declarations are compile-time
validated only:

- keys must be dot-separated lower-case segments matching
  `[a-z][a-z0-9_]*` (for each segment)
- duplicate keys are rejected per capsule declaration
- values must be literals (`string`, `number`, or `bool`)
- this metadata has no runtime/codegen coupling in the current MVP

## Unsafe Policy Public API Boundary

Unsafe policy is exported in the public manifest builtins surface as
`unsafe_policy` (`never`, `always`, `conditional`) plus optional
`unsafe_details` for conditional cases.

Current conditional policy is `core.island_make_*`: unsafe is required when the
island argument is not a tracked scoped-island variable. Other unsafe-only
builtins are `always`.

Boundary: this manifest policy is per builtin entry. It does not add a
transitive function-level `unsafe` effect to user function signatures; `unsafe`
is enforced at direct operation sites in checker semantics.

## Privacy And Consent

The v1 privacy MVP is compiler-enforced:

- `uses privacy` requires the semantic clause `privacy`.
- Secret-bearing signature detection is recursive for function parameter,
  return, and throws types: the checker trims whitespace, unwraps optional
  (`?`) and slice (`[]`) layers, and then matches a `secret.` prefix.
- The current concrete secret type surface remains `secret.i32`/`SecretInt`.
- Secret-bearing signatures require `consent(<token>)`.
- The referenced consent parameter must exist and have type `consent.token`.
- `core.secret_seal_i32` and `core.secret_unseal_i32` require the privacy
  effect and a consent token.

Runtime secret storage is intentionally minimal; the v1 guarantee is static
auditing and call-shape enforcement, not cryptographic isolation.

Public privacy lowering boundary for the current surface:

- `core.consent_token()` lowers to an opaque compiler/runtime sentinel in the
  local lowering/runtime path, rather than the forgeable public value `1`.
- Consent clauses lower to exact sentinel validation; non-zero integers are not
  accepted as valid consent tokens. The representation remains a local
  one-slot runtime token for this MVP, not distributed consent enforcement.
- `core.secret_seal_i32` and `core.secret_unseal_i32` lower to arithmetic that
  preserves the first argument value while still consuming/evaluating the token
  argument in the lowered expression path.
- This is a public static-policy + lowering-shape contract for the current
  surface, not a cryptographic secret-storage or runtime secrecy guarantee.

## Budget

`budget(<non-negative integer constant>)` requires `uses budget`. Lowering
emits deterministic budget guard instructions for functions with the clause.
The current cross-function guardrail is static and conservative: a direct call,
`core.spawn`, or `core.task_spawn_*` edge into a `budget(N)` function/worker
requires the caller to declare a non-zero budget context of at least `N`. The
existing lowered `IRCall` guard still charges the caller for entering the edge.
This prevents a smaller caller budget from launching a larger local budget
context, while preserving `budget(0)` as a deterministic local failure-before-
call path. It is not aggregate runtime-wide accounting across repeated calls,
task scheduling, or distributed actors. Full runtime-wide and distributed budget
enforcement remain post-v1.

The v1 budget charge table is explicit and local to lowered IR. Each chosen
cost-bearing instruction costs one budget unit and is guarded immediately before
the instruction, preserving any operand stack values through scratch locals when
needed.

| Lowered IR operation | Surface sources | Cost |
| --- | --- | ---: |
| `IRWrite` | `print` | 1 |
| `IRCall` | user calls and runtime builtin calls | 1 |
| `IRAllocBytes`, `IRMakeSliceU8`, `IRMakeSliceU16`, `IRMakeSliceI32` | allocation and slice constructors | 1 |
| `IRIndexLoadI32`, `IRIndexLoadU8`, `IRIndexLoadU16` | slice/index reads | 1 |
| `IRIndexStoreI32`, `IRIndexStoreU8`, `IRIndexStoreU16` | slice/index writes | 1 |
| `IRIslandNew`, `IRIslandMakeSliceU8`, `IRIslandMakeSliceU16`, `IRIslandMakeSliceI32`, `IRIslandFree` | island allocation, island slices, and island cleanup | 1 |
| `IRCapIO`, `IRCapMem` | capability token construction | 1 |
| `IRMemReadI32`, `IRMemReadU8`, `IRMemReadPtr` | unsafe memory reads | 1 |
| `IRMemWriteI32`, `IRMemWriteU8`, `IRMemWritePtr`, `IRPtrAdd` | unsafe memory writes and pointer arithmetic | 1 |
| `IRMmioReadI32`, `IRMmioWriteI32` | MMIO reads and writes | 1 |
| `IRSymAddr` | symbol address materialization and function values | 1 |
| `IRCtxSwitch` | explicit context switch operation | 1 |

Pure local stack, arithmetic, comparison, local/global load/store, labels,
branches, string literal materialization, and returns are not budget-charged in
v1. The cost table is intentionally deterministic; changing either coverage or
cost is a language/runtime policy change that must update this table and the
lowering/verifier tests together.

Budget exhaustion uses the stable local policy-failure ABI:

- non-throwing functions return their declared result slot shape filled with
  zero/default scalar slots
- throwing compact results return a zero/default error payload followed by trap
  status `1`
- throwing non-compact results return zero/default success slots,
  zero/default error payload slots, and trap status `1`
- staged typed-task throwing results use the same zero/default payload and trap
  status shape in the staged result buffer

The trap status is part of the local lowered result ABI, not a process abort,
host exception, or distributed cancellation signal. For enum error payloads,
the zero/default payload means ordinal `0` plus zero/default payload slots.

The deterministic guarantee is local to compiler lowering plus the static
cross-edge budget-context guardrail above: repeated checking/lowering of the
same program emits the same diagnostics, budget guard instruction shape, and
local-slot metadata. It is not a promise of aggregate runtime-wide propagation,
distributed budget accounting, or wall-clock resource accounting.

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
non-literal, async, throwing, wrong-shape, mutable-global, and under-budgeted
worker targets.

The actor MVP is local-process actor spawn/send/receive with tagged-message
support. Distributed actors and non-host runtime execution evidence remain
release-lab or post-v1 items, not a language guarantee for `v1.0.0`.

Actor/task transfer safety is still part of the conservative local safety MVP.
The checker validates zero-argument synchronous worker targets, rejects
wrong-shape and mutable-global targets, checks typed actor message payloads
after module resolution, and preserves use-after-transfer diagnostics for
actors, tasks, islands, and structs containing those handles.

## Epic 06 release evidence

The release-blocking safety slice is:

```sh
go test ./compiler/... -run "Effect|Uses|Capability|Unsafe|Ownership|Borrow|Consume|Inout|Island|Region|Privacy|Budget" -count=1
```

That slice includes positive and negative checks for transitive `uses`
propagation, capability attenuation, unsafe-only builtins, ownership transfer,
region escape prevention, privacy consent clauses, and budget lowering guards.

The Plan250 Epic 04 regression slice is intentionally narrower but more
boundary-focused:

```sh
go test ./compiler -run "Plan250Safety|Plan250Runtime|Plan250Link" -count=1
```

It covers branch/loop ownership merges, cross-module actor message sendability,
reserved runtime symbol diagnostics, TOBJ metadata diagnostics, and stable
panic/exit result reporting.
