# Memory Ideal Vertical Slice v7 FFI Correlation

Status: validated_narrow

This matrix intentionally has exactly four rows. It extends the Memory Ideal Vertical Slice
v0/v1/v2/v3/v4/v5/v6 correlation pattern to external pointer and FFI lifetime quarantine without
claiming arbitrary external pointer safety, C/FFI lifetime safety, safe wrapper promotion, broad
unsafe noalias, target parity, performance, arbitrary external allocator provenance, or full
runtime/ABI proof. `MemoryFactGraph` remains the source of truth; this document is only a
projection/audit correlation.

| requirement_id | claim                                                                                               | source_fact_id                                                                                 | validator                             | report_row                                       | negative_test                                                                                                  | target_level     | status       |
| -------------- | --------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ------------------------------------- | ------------------------------------------------ | -------------------------------------------------------------------------------------------------------------- | ---------------- | ------------ |
| MEM-FFI-001    | external pointers remain unsafe_unknown or external_unknown unless compiler-owned provenance exists | plir:ffiV7:f_external_unknown:op_ffi:ffi_pointer_external_unknown                              | external_pointer_provenance_validator | ffi_pointer_external_unknown                     | TestMemoryIdealV7ProjectsFFICallExternalFacts,TestValidateMemoryReportRejectsUnsafeUnknownProvenanceKnownClaim | linux-x64:narrow | conservative |
| MEM-FFI-002    | external calls may retain borrowed pointers unless compiler-owned contract proves otherwise         | plir:ffiV7:f_borrowed:op_ffi:ffi_call_may_retain_borrow                                        | ffi_lifetime_conservative_validator   | ffi_call_may_retain_borrow                       | TestMiniMemoryModelV7FFICases,TestMemoryIdealV7ProjectsFFICallExternalFacts                                    | linux-x64:narrow | conservative |
| MEM-FFI-003    | safe wrapper promotion from raw or external pointer rejects without compiler-owned proof            | plir:ffiV7:f_external_unknown:op_safe_wrapper:safe_wrapper_promotion_rejected_without_contract | safe_wrapper_promotion_validator      | safe_wrapper_promotion_rejected_without_contract | TestMiniMemoryModelV7FFICases,TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown                        | linux-x64:narrow | rejected     |
| MEM-FFI-004    | external calls invalidate broad noalias unless narrow validator proves otherwise                    | plir:ffiV7:f_noalias:op_ffi:ffi_noalias_invalidated_by_external_call                           | ffi_noalias_conservative_validator    | ffi_noalias_invalidated_by_external_call         | TestMiniMemoryModelV7FFICases,TestValidateMemoryReportRejectsBroadNoAliasClaim                                 | linux-x64:narrow | conservative |

## Validator

Run:

```bash
go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory/ideal-v5-v7/memory-ideal-vslice-v7-ffi-correlation.md
```

The validator checks this v7 table shape:

- every row has a `requirement_id`;
- every row has a `source_fact_id`;
- every row names a `validator`;
- every row names at least one `negative_test`;
- `status` is one of `validated`, `validated_narrow`, `conservative`, `rejected`, `future`, or
  `explicit_non_goal`;
- the row set is exactly `MEM-FFI-001`, `MEM-FFI-002`, `MEM-FFI-003`, and `MEM-FFI-004`.

## Update Policy

External and unknown pointer provenance cannot become safe provenance, provenance-known, noalias,
bounds-check-elimination, or trusted storage evidence. External calls are modeled as possibly
retaining borrowed pointers unless a compiler-owned contract exists in the supported surface. Safe
wrapper promotion from raw or external pointers remains rejected without compiler-owned proof.
External calls invalidate broad noalias claims and remain conservative unless a later narrow
validator proves a specific fact.

This document does not claim arbitrary external pointer safety, C/FFI lifetime safety, safe wrapper
promotion, broad unsafe noalias, target parity, performance, arbitrary external allocator
provenance, or full runtime/ABI proof.
