# Ownership Markers v1

Ownership markers are part of the checked v1 function-call contract.

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

## Actor And Task Transfer

Actor/task worker entrypoints must be zero-argument synchronous user functions
returning `i32`. Worker signatures that borrow, mutate, throw, await, or touch
mutable global state are rejected. Sendable result types are limited to scalar
and recursively sendable structural values covered by the current semantics
checker.

## Current Limits

The checker is intentionally conservative. It tracks region-backed slices,
island handles, and structs containing them across local scopes and common
control-flow merges, but it is not an SSA lifetime solver. Ambiguous region
merges are reported as diagnostics and must be resolved by rewriting the code.
