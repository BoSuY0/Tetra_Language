# Memory Ideal Vertical Slice v6 Bounds Final Audit

Status: validated_narrow

This audit closes the bounds-check proof-id slice for the current supported
surface. It extends the v0/v1/v2/v3/v4/v5 memory correlation pattern without
implementing broad optimizer correctness, target parity, performance,
arbitrary unsafe pointer arithmetic proof, arbitrary external pointer safety,
an FFI lifetime system, or a full theorem prover. `MemoryFactGraph` remains
the truth source; reports remain projections.

## Row Classifications

| requirement_id | classification | evidence | boundary |
| --- | --- | --- | --- |
| MEM-BOUNDS-001 | validated_narrow | `compiler/internal/memoryfacts/from_validation_test.go` proves `bounds_check_retained_dynamic`; `compiler/internal/memorymodel/mini_test.go` covers retained dynamic normal-build checks; report validators reject dynamic rows without `normal_build_check`. | Missing proof keeps a bounds check in the normal build rather than authorizing elimination. |
| MEM-BOUNDS-002 | validated_narrow | `compiler/internal/validation/validation_test.go` proves missing/unknown proof-id rejection and live proof acceptance; `compiler/internal/memoryfacts/from_validation_test.go` proves `bounds_check_removed_with_proof_id`; `compiler/internal/lower/proof_bce_test.go` covers proof-tagged unchecked lowering cases. | Removed checks are valid only when tied to compiler-owned proof ids and parent proof evidence. |
| MEM-BOUNDS-003 | rejected | `compiler/internal/memoryfacts/report_test.go` and `tools/cmd/validate-memory-report/main_test.go` keep unsafe_unknown optimization claims rejected; `compiler/internal/memorymodel/mini_test.go` covers unsafe_unknown bounds elimination rejection. | `unsafe_unknown` cannot authorize `bounds_check_eliminated`, `index_in_range`, or zero-cost bounds removal. |
| MEM-BOUNDS-004 | conservative | `compiler/internal/memoryfacts/from_plir_test.go` proves `raw_bounds_runtime_check_normal_build`; `compiler/internal/runtimeabi/raw_pointer_bounds_test.go` covers raw bounds overflow/rejection paths; `compiler/internal/memorymodel/mini_test.go` covers raw overflow check/trap conservatism. | Raw target-width and overflow uncertainty remains a normal-build check/trap or rejected/conservative row. |

## Minimal Report Projection

Projected v6 claims:

| claim | source stage | validator | notes |
| --- | --- | --- | --- |
| `bounds_check_retained_dynamic` | validation | `normal_build_bounds_check_validator` | Records retained bounds checks when no compiler-owned proof exists; keeps `normal_build_check`. |
| `bounds_check_removed_with_proof_id` | validation | `bounds_proof_id_validator` | Derived from proof-id parent evidence and projected as narrow zero-cost proof only when validated. |
| `bounds_check_removal_rejected_missing_proof_id` | validation | `bounds_proof_id_validator` | Rejected evidence for removed checks without compiler-owned proof ids. |
| `raw_bounds_runtime_check_normal_build` | plir | `raw_bounds_width_validator` | Derived from unsafe-checked raw bounds evidence and keeps a normal-build check/trap. |

Validators reject derived v6 rows without `parent_fact_id`, unsafe_unknown
optimization claims, dynamic-check rows without `normal_build_check`, and
missing/mismatched proof-id removal. Reports remain projections from
compiler-owned facts.

## Positive Coverage

- Proof-tagged lowered index checks are accepted by validation and projected
  through MemoryFactGraph/report as `bounds_check_removed_with_proof_id`.
- Retained dynamic bounds checks project with `dynamic_check_required` and
  `normal_build_check`.

## Negative Coverage

- Lowered unchecked index without proof id is rejected.
- Mismatched or unknown proof id is rejected.
- `unsafe_unknown` cannot authorize eliminated bounds checks.
- Dynamic optimization/check rows without `normal_build_check` are rejected.
- Raw overflow/target-width uncertainty cannot become a zero-cost eliminated
  bounds check.

## Nonclaims

- No "Memory 100% complete".
- No broad optimizer correctness.
- No target parity.
- No performance claim.
- No arbitrary unsafe pointer arithmetic proof.
- No arbitrary external pointer safety.
- No FFI lifetime model.
- No full theorem prover.
