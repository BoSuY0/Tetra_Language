# Packet P3: Raw Bounds Runtime Path

## Objective

Read-only audit of raw pointer bounds behavior for verified `core.alloc_bytes` roots and unknown external pointers.

## Context

The current slice must add red/green coverage for:

- negative `ptr_add`
- upper bound at allocation length
- access-width overflow for `load_i32`/pointer-width access
- unknown external pointer remains conservative, not `safe_known`
- raw slice from unknown pointer remains external_unknown

## Files / Sources

Start with:

- `compiler/internal/runtimeabi/raw_pointer_bounds.go`
- `compiler/internal/runtimeabi/allocation_contract.go`
- `compiler/internal/semantics/builtins.go`
- `compiler/internal/lower/lower.go`
- `compiler/internal/backend/linux_x64/`
- `compiler/internal/backend/x64core/`
- `compiler/tests/safety`, `compiler/tests/runtime`, and `compiler/*raw*/*bounds*` tests

## Ownership

Read-only. Do not edit files.

## Do

- Find existing raw pointer and allocation bounds code paths.
- Find existing tests for `alloc_bytes`, `ptr_add`, raw load/store, and raw slice gateways.
- Identify the narrowest test locations for the red/green closure.
- Cite exact files/lines and any likely target limitations.

## Do Not

- Do not implement the closure.
- Do not claim arbitrary unsafe pointer safety.

## Expected Output

Markdown report with current behavior, missing red tests, proposed minimal test files, evidence, and uncertainty.

## Verification

Read-only probes only; no broad test runs.
