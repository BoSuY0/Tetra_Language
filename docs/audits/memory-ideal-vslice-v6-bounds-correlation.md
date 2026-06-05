# Memory Ideal Vertical Slice v6 Bounds Correlation

Status: validated_narrow

This matrix intentionally has exactly four rows. It extends the Memory Ideal
Vertical Slice v0/v1/v2/v3/v4/v5 correlation pattern to bounds-check proof IDs
without claiming broad optimizer correctness, target parity, performance, or
arbitrary unsafe pointer arithmetic safety. `MemoryFactGraph` remains the
source of truth; this document is only a projection/audit correlation.

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-BOUNDS-001 | retained dynamic bounds checks remain normal-build checks when no proof exists | validation:bounds:retained_dynamic | normal_build_bounds_check_validator | bounds_check_retained_dynamic | TestMiniMemoryModelV6BoundsProofCases,TestValidateMemoryReportRejectsDynamicOptimizationClaimWithoutNormalBuildCheck | linux-x64:narrow | validated_narrow |
| MEM-BOUNDS-002 | removed bounds check requires compiler-owned proof id | validation:sum:3:proof:while:i:xs:1:1:proof_guard:bounds_check_removed_with_proof_id | bounds_proof_id_validator | bounds_check_removed_with_proof_id | TestCheckBoundsProofsRejectsRemovedCheckWithoutProofID,TestCheckBoundsProofsWithPLIRRejectsUnknownLiveProof | linux-x64:narrow | validated_narrow |
| MEM-BOUNDS-003 | unsafe_unknown cannot authorize eliminated bounds checks | validation:sum:bounds:sum:4:missing_proof | bounds_proof_id_validator | bounds_check_removal_rejected_missing_proof_id | TestMiniMemoryModelV6BoundsProofCases,TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaims | linux-x64:narrow | rejected |
| MEM-BOUNDS-004 | raw bounds target-width or overflow uncertainty keeps normal-build check or trap | plir:main:op_raw_load:unsafe:raw_bounds_runtime_check_normal_build | raw_bounds_width_validator | raw_bounds_runtime_check_normal_build | TestMiniMemoryModelV6BoundsProofCases,TestMemoryIdealV6ProjectsRawPointerBoundsMetadata | linux-x64:narrow | conservative |

## Validator

Run:

```bash
go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v6-bounds-correlation.md
```

The validator checks this v6 table shape:

- every row has a `requirement_id`;
- every row has a `source_fact_id`;
- every row names a `validator`;
- every row names at least one `negative_test`;
- `status` is one of `validated`, `validated_narrow`, `conservative`,
  `rejected`, `future`, or `explicit_non_goal`;
- the row set is exactly `MEM-BOUNDS-001`, `MEM-BOUNDS-002`,
  `MEM-BOUNDS-003`, and `MEM-BOUNDS-004`.

## Update Policy

Removed bounds checks require compiler-owned proof ids linked to PLIR proof
guards or equivalent compiler-owned proof metadata. Missing or mismatched proof
ids reject. Unknown unsafe/raw provenance cannot authorize eliminated bounds
checks. Raw target-width and overflow uncertainty remains a normal-build check,
trap, conservative row, or rejected row.

This document does not claim broad optimizer correctness, target parity,
performance, arbitrary unsafe pointer arithmetic proof, arbitrary external
pointer safety, an FFI lifetime system, or a full theorem prover.
