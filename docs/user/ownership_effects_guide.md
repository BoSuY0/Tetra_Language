# Ownership And Effects Guide

Status: user guide for the safety model required by the v1.0 scope.

## Ownership

The v1.0 release cannot claim safe-code memory safety until ownership and
lifetime checks reject use-after-move, escaping borrows, invalid mutable
aliasing, invalid island transfers, and actor/task race patterns.

Use diagnostics from `./tetra check <file>` as the source of truth. If a rule is
documented here but not covered by tests and release evidence, it remains a
release blocker rather than a user guarantee.

## Effects And Capabilities

Effects describe observable operations such as IO, allocation, memory access,
runtime features, and unsafe capability use. Capability and unsafe boundaries
are specified in:

- `docs/spec/capabilities.md`
- `docs/spec/unsafe.md`
- `docs/spec/islands.md`

## User Workflow

1. Declare only the effects a function needs.
2. Keep unsafe or capability-bearing code narrow.
3. Run `./tetra check <file>` and fix diagnostics before using `run` or
   release smoke commands.

## Allowed patterns

- Forward borrowed values only to other borrowed parameters.
- Pass `inout` only to mutable locals that are not simultaneously borrowed or
  consumed in the same call.
- Move `consume` values exactly once and do not reuse or reassign the source
  local after the move.
- Use scoped islands for safe region allocation, and keep returned slices inside
  the island scope.
- Keep raw memory and capability creation inside small `unsafe` blocks with
  matching `uses` effects.

## Forbidden patterns

- Returning a borrowed value or an alias derived from a borrow.
- Passing a borrowed value to an owned or `inout` parameter.
- Reusing a consumed actor, task, island-backed slice, or scalar local.
- Letting scoped island handles or slices escape their scope.
- Treating `uses mem`, `uses io`, or an effect group as permission to create a
  capability token in safe code.

## Verification

```sh
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go test ./tools/cmd/validate-diagnostic/... -count=1
go test ./compiler/... -run "Effect|Uses|Capability|Unsafe|Ownership|Borrow|Consume|Inout|Island|Region|Privacy|Budget" -count=1
```
