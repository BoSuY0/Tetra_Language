# Memory Ideal Vertical Slice v5 Correlation

Status: validated_narrow

This matrix intentionally has exactly four rows. It extends the Memory Ideal Vertical Slice
v0/v1/v2/v3/v4 correlation pattern to raw pointer unsafe contracts without making arbitrary unsafe
memory safe. `MemoryFactGraph` remains the source of truth; this document is only a projection/audit
correlation.

| requirement_id | claim                                                                                     | source_fact_id                                                            | validator                             | report_row                           | negative_test                                                                                                                                                               | target_level     | status           |
| -------------- | ----------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- | ------------------------------------- | ------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------- | ---------------- |
| MEM-UNSAFE-001 | unsafe_unknown raw pointer cannot produce safe_known/provenance_known/noalias facts       | plir:rawUnsafeV5:op_ptr_unknown:unsafe:unsafe_unknown_rejected_safe_facts | unsafe_unknown_fact_validator         | unsafe_unknown_rejected_safe_facts   | TestMiniMemoryModelV5RawPointerUnsafeContractCases,TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown,TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaims | linux-x64:narrow | rejected         |
| MEM-UNSAFE-002 | unsafe_verified_root from core.alloc_bytes may produce bounded allocation-base facts      | allocplan:rawUnsafeV5:p:unsafe_verified_root_allocation_base              | unsafe_verified_root_bounds_validator | unsafe_verified_root_allocation_base | TestMiniMemoryModelV5RawPointerUnsafeContractCases,TestMemoryIdealV5ProjectsRawPointerUnsafeContractFacts                                                                   | linux-x64:narrow | validated_narrow |
| MEM-UNSAFE-003 | runtime-checkable unsafe contracts may validate nonnull/alignment/length only             | plir:rawUnsafeV5:op_runtime_contract:unsafe_contract_runtime_checkable    | unsafe_runtime_contract_validator     | unsafe_contract_runtime_checkable    | TestMiniMemoryModelV5RawPointerUnsafeContractCases,TestMemoryIdealV5ProjectsRawPointerUnsafeContractFacts                                                                   | linux-x64:narrow | validated_narrow |
| MEM-UNSAFE-004 | unsafe noalias/lifetime/region contracts remain static-untrusted unless separately proven | plir:rawUnsafeV5:op_static_contract:unsafe_contract_static_untrusted      | unsafe_static_contract_validator      | unsafe_contract_static_untrusted     | TestMiniMemoryModelV5RawPointerUnsafeContractCases,TestValidateMemoryReportRejectsValidatedNoAliasWithUnknownAliasState,TestValidateMemoryReportRejectsBroadNoAliasClaim    | linux-x64:narrow | conservative     |

## Validator

Run:

```bash
go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory/ideal-v5-v7/memory-ideal-vslice-v5-correlation.md
```

The validator checks this v5 table shape:

- every row has a `requirement_id`;
- every row has a `source_fact_id`;
- every row names a `validator`;
- every row names at least one `negative_test`;
- `status` is one of `validated`, `validated_narrow`, `conservative`, `rejected`, `future`, or
  `explicit_non_goal`;
- the row set is exactly `MEM-UNSAFE-001`, `MEM-UNSAFE-002`, `MEM-UNSAFE-003`, and `MEM-UNSAFE-004`.

## Update Policy

Unknown external raw pointers remain `unsafe_unknown` and may not produce `safe_known`,
`provenance_known`, or `no_alias` facts. Verified `core.alloc_bytes` roots may produce bounded
allocation-base metadata, but the metadata remains unsafe-origin evidence. Runtime-checkable unsafe
contracts are limited to nonnull, alignment, and length/bounds. Unsafe noalias, lifetime, and region
contracts remain static-untrusted unless a separate later proof exists.

This document does not claim arbitrary external pointer safety, an FFI lifetime system, broad unsafe
noalias, safe wrapper promotion, actor/task/runtime expansion, target parity, or performance.
