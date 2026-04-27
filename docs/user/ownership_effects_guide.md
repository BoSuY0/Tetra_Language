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

## Verification

```sh
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go test ./tools/cmd/validate-diagnostic/... -count=1
```
