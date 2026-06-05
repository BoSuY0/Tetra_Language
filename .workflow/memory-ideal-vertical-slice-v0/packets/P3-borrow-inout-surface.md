# P3-borrow-inout-surface

Packet ID: P3-borrow-inout-surface

Objective: Discover current borrow through aggregate/optional and inout alias
surface for B2a/B3a.

Context: B2a supports only struct field and optional payload local borrow
propagation/copy escape. B3a supports only unique local and sequential inout
with narrow noalias wording.

Files / sources:

- `compiler`
- `compiler/internal/semantics`
- `compiler/internal/plir`
- `compiler/internal/validation`
- `compiler/tests/semantics`
- `compiler/tests/ownership`
- `examples/safe_view_*.tetra`

Ownership: read-only.

Do: Identify existing positive/negative tests, syntax examples, diagnostics,
and likely gaps. Separate supported v0 forms from future non-goals.

Do not: Edit files, broaden to enum/generic/function/interface/async/actor/raw
pointer support.

Expected output: `.workflow/memory-ideal-vertical-slice-v0/results/P3-borrow-inout-surface.md`
summary with evidence.

Verification: cite commands and file paths inspected.
