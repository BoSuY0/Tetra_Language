# Formal Core v1

Status: P23.2 evidence/report closure for the Ideal Master Plan.

Schema: `tetra.formal_core.v1`

Scope: `p23.2_formal_core_v1`

## Summary

Formal Core v1 records the current small machine-checkable semantic core. It is
not a full formalization of Tetra. The report connects the internal
`compiler/internal/formalcore` rule inventory to live witnesses for values,
borrows and owned/copy, provenance and regions, bounds proof id semantics,
allocation length contracts, allocation intent lowering, raw pointer bounds
metadata, and check-elimination validity.

## Coverage Rows

| Row | Evidence | Boundary |
| --- | --- | --- |
| `values` | `differential.CheckBackendMatrix` compares supported scalar i32 values across source, Stack IR, optimized Stack IR, SSA, and Machine IR lanes. | Current supported scalar i32 subset only. |
| `borrows_owned_copy` | `compiler.Parse`, `compiler.Check`, `BuildPLIR`, and `plir.VerifyProgram` accept real `window().borrow().copy()` evidence with borrowed/no-escape and owned/provenance facts. | Current PLIR borrow/copy facts only. |
| `provenance_regions` | PLIR records island provenance and explicit regions for region-backed values, derived views, and borrows. | Internal PLIR evidence, not a complete region calculus. |
| `bounds_proof_id_semantics` | `validation.CheckBoundsProofsWithPLIR` accepts removed checks only when the proof id exists in PLIR proof guards. | Proof-tagged removed index checks only. |
| `allocation_length_contract` | `allocplan.FromPLIR` classifies valid empty, normal, rejected negative, and rejected overflow length contracts. | Planner evidence, not platform build/run evidence. |
| `allocation_intent_lowering` | `validation.ValidateAllocationLowering` checks allocation plans against lowered IR and rejects drift. | Current allocation-plan/lowering validators only. |
| `raw_pointer_bounds_metadata` | `runtimeabi.NewRawAllocationBounds`, `DeriveRawPointerBounds`, and `RawSliceBoundsFromParts` cover allocation-base, derived-offset, rejected, and checked external/unknown metadata. | Unsafe policy does not change. |
| `check_elimination_validity` | Unchecked lowered index operations stay valid only with a preserved proof id accepted by PLIR proof guards. | No broad theorem prover is claimed. |

## Validator Contract

`ValidateP23FormalCoreV1Report` rejects:

- wrong schema or scope;
- missing or duplicate rows;
- missing witness references;
- placeholder evidence;
- missing formal spec validation;
- missing values evidence;
- missing borrow/copy or provenance/regions facts;
- missing bounds proof id or check-elimination evidence;
- missing allocation length or allocation-intent lowering evidence;
- missing raw pointer bounds metadata evidence;
- fake full formal proof claims;
- fake broad language proof claims;
- fake unsafe-policy-change claims;
- fake runtime-behavior-change claims;
- fake safe-semantics-change claims;
- performance claims.

## Non-claims

- No full formal proof of Tetra is claimed.
- No broad language theorem prover is claimed.
- No public source interpreter or backend selector is introduced.
- Unsafe policy does not change.
- Runtime behavior does not change.
- Safe-program semantics do not change.
- No performance claim is made.

## Verification

Focused evidence:

```text
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler ./compiler/internal/formalcore -run 'P23FormalCore|FormalCore' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/validation ./compiler/internal/plir ./compiler/internal/allocplan ./compiler/internal/runtimeabi -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics -run 'FeatureRegistry' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
```
