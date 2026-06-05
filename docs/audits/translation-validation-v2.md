# Translation Validation v2

Status: P23.0 evidence/report closure for the Ideal Master Plan.

Schema: `tetra.translation.validation.v2`

Scope: `p23.0_translation_validation_v2`

## Summary

Translation Validation v2 records the current supported optimizer-validation
subset as machine-checkable evidence. It builds on the existing
`validation.ValidateTranslation` and `opt.Manager` contract and adds an
explicit P23.0 report/validator for scalar symbolic equivalence, supported i32
slice memory samples, loop and call/inlining differential samples, bounds proof
preservation, allocation-plan preservation, and sha256 before/after IR hashes.

## Coverage Rows

| Row | Evidence | Boundary |
| --- | --- | --- |
| `registered_passes` | `opt.RegisteredPasses`, `opt.ValidatePassContract`, and `opt.NewManager` require `translation_validation`, `validation.ValidateTranslation`, report rows, negative tests, and validation metadata. | Limited to currently registered optimizer passes. |
| `symbolic_scalar_equivalence` | `validation.ValidateTranslation` checks supported scalar i32 local arithmetic/comparison rewrites and rejects semantic mismatch. | Not a full scalar theorem prover. |
| `memory_equivalence` | `differential.CheckBackendMatrix` compares supported proof-tagged i32 slice samples across source, Stack IR, optimized Stack IR, SSA, and Machine IR lanes. | Not a broad memory model or alias model. |
| `bounds_proof_preservation` | `CheckBoundsProofs` and proof-fact multiset comparison reject missing or changed proof ids. | Only proof-tagged removed bounds checks in current IR are covered. |
| `allocation_plan_preservation` | `validation.ValidateAllocationLowering` checks allocation plan rows against emitted Stack IR allocation lowering and rejects drift. | Evidence-bound to current `allocplan` and lowering validators. |
| `machine_checkable_hashes` | `BuildOptimizationValidationMetadata` records distinct sha256 before/after IR hashes and compared functions. | Hashes are evidence, not proof by themselves. |

## Validator Contract

`ValidateP23TranslationValidationV2` rejects:

- wrong schema or scope;
- missing or duplicate rows;
- missing witness references;
- placeholder evidence;
- incomplete registered-pass coverage;
- missing scalar, memory, loop, call, proof, allocation, or hash evidence;
- fake full formal proof claims;
- fake exhaustive optimizer completeness claims;
- fake broad memory model or loop theorem-prover claims;
- performance claims;
- runtime behavior changes;
- safe-program semantics changes.

## Non-claims

- No full formal proof is claimed.
- No exhaustive optimizer completeness is claimed.
- No broad memory model or alias model is claimed.
- No broad loop theorem prover is claimed.
- No performance claim is made.
- Runtime behavior does not change.
- Safe-program semantics do not change.

## Verification

Focused evidence:

```text
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'P23TranslationValidationV2|ValidateP23TranslationValidationV2' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/validation -run 'ValidateTranslation|OptimizationValidationMetadata|ValidateAllocationLowering' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/opt -run 'Manager|BasicScalar|SCCP|Mem2Reg|Inline|Loop|LICM' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/differential -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
```
