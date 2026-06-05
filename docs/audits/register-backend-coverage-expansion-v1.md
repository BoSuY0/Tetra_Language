# Register Backend Coverage Expansion v1

Status: P16.2 evidence audit for the Ideal Master Plan.

## Summary

The register backend now accepts scalar `IRDivI32` and `IRModI32` in the
SSA-gated Machine IR path and verifies Linux x64 native output against the stack
fallback. Machine backend reports also expose instruction-selection and
validation metadata for promoted paths, plus ABI boundary policy for multi-slot
return cases that are still stack fallback. Backend coverage summaries now
include selected ordinary-corpus evidence for non-runtime-heavy benchmark rows
that reached Machine IR without push/pop stack churn.

## Evidence

| Check | Result |
| --- | --- |
| Machine IR accepts scalar div/mod | pass |
| Linux x64 register div/mod matches stack fallback | pass |
| Backend report includes `div`/`mod` instruction selection | pass |
| Backend report validates machine verifier and allocation verifier status | pass |
| Backend report exposes stack-churn count, spill/reload status, and call-clobber status | pass |
| Backend report exposes multi-slot return ABI policy without false promotion | pass |
| Backend report exposes bounded ABI value class and boundary status for header/pair/aggregate multi-slot returns | pass |
| Backend summary exposes `ordinary_corpus` and no-stack-churn majority evidence for the selected non-runtime-heavy corpus | pass |
| Backend summary exposes `abi_boundaries` counts for multi-slot return and call-return fallbacks | pass |

## Boundaries

This audit does not promote unsupported aggregate returns, slice/String
multi-slot returns, or runtime-heavy code to the register backend. Those paths
remain stack fallback or explicit unsupported ABI classifications until later
evidence promotes them safely. The ordinary-corpus majority is a coverage
statement for the selected report rows, not a performance claim.
