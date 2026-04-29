# Ownership Markers v1

Ownership markers are part of the checked function-call contract in the current
conservative MVP. The MVP is intentionally narrow: it enforces local call-site
ownership markers and resource/transfer diagnostics, but it is not a full SSA
lifetime solver.

## Markers

- `borrow T`: read-only view. The parameter is immutable and values derived
  from the borrow cannot escape through returns, owned parameters, or `inout`
  assignment.
- `inout T`: exclusive mutable access to a mutable local. The argument must be a
  mutable local value, not a literal or expression.
- `consume T`: moves a local value into the callee. The source local cannot be
  reused, reassigned, or consumed again after the call.

## Aliasing Rules

Within a single call, the same local cannot be passed as both `inout` and
`borrow`, or as both `inout` and `consume`. The same local cannot satisfy two
`consume` parameters in one call.

## Resource Lifetime MVP

The current resource lifetime MVP conservatively tracks task handles,
task groups, island handles, region-backed slices, and structs containing those
resources through local scopes and common control-flow joins. It rejects double
join/close/use, use-after-transfer, ambiguous resource provenance on returns,
and ambiguous lifetime merges. This is a conservative MVP, not a full SSA
lifetime solver; future lifetime SSA work is planned separately and is not part
of the current support claim.

## Actor And Task Transfer

Actor/task transfer safety is a local MVP. It checks worker entrypoints,
sendable scalar and supported structural results, handle transfer, and
use-after-transfer diagnostics. It does not claim distributed actor safety, full
race-safety proofs, full cancellation semantics, or structured concurrency.

Actor/task worker entrypoints must be zero-argument synchronous user functions
returning `i32`. Worker signatures that borrow, mutate, throw, await, or touch
mutable global state are rejected. Sendable result types are limited to scalar
and recursively sendable structural values covered by the current semantics
checker.

## Current Limits

The checker is intentionally conservative. It tracks region-backed slices,
island handles, task handles, task groups, actor handles, and structs containing
them across local scopes and common control-flow merges, but it is not an SSA
lifetime solver. Ambiguous region/resource/lifetime merges are reported as
diagnostics and must be resolved by rewriting the code. Planned lifetime SSA
work may make those diagnostics more precise in the future, but the current
supported safety surface remains the conservative MVP.

## Epic 06 coverage

Ownership coverage is release-blocking in the focused safety slice:

```sh
go test ./compiler/... -run "Effect|Uses|Capability|Unsafe|Ownership|Borrow|Consume|Inout|Island|Region|Privacy|Budget" -count=1
```

The slice checks allowed borrow forwarding and distinct `borrow`/`inout`
locals, rejects reuse after `consume`, rejects borrowed values escaping through
returns, owned parameters, or `inout`, rejects double use of closed/joined
resources, rejects ambiguous resource provenance, and verifies actor/task
handles cannot be used after ownership transfer.
