# P1-semantics-registry

Packet ID: P1-semantics-registry

Objective: Identify the smallest semantics hook for B1-min representation
metadata registry and negative tests.

Context: Safe representation metadata names must be compiler-owned and not
assignable before lowering.

Files / sources:

- `compiler/internal/semantics`
- `compiler/tests/semantics`
- existing representation/slice/string metadata tests

Ownership: read-only.

Do: Locate assignment target resolution, field access/type model code, existing
diagnostics, and tests for `ptr`, `len`, nested wrapper, optional payload, and
reserved names. Recommend minimal files to edit and RED/GREEN tests.

Do not: Edit files or add broad type-system features.

Expected output: `.workflow/memory-ideal-vertical-slice-v0/results/P1-semantics-registry.md`
summary with evidence.

Verification: cite commands and file paths inspected.
