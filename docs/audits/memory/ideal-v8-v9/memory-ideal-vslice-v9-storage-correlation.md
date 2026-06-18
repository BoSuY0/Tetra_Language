# Memory Ideal Vertical Slice v9 Storage Correlation

Status: accepted narrow correlation target for `MEM-STORAGE-009`.

This table is intentionally exact. It adds only escape-aware storage/lowering integrity evidence. It
does not add full region inference, optimizer-wide allocation correctness, production actor runtime
proof, full async lifetime proof, FFI lifetime safety, arbitrary external pointer safety, target
parity, performance evidence, or a "Memory 100%" claim.

| requirement_id  | claim                                                                           | source_fact_id                                                        | validator                               | report_row                               | negative_test                                                                                                     | target_level     | status           |
| --------------- | ------------------------------------------------------------------------------- | --------------------------------------------------------------------- | --------------------------------------- | ---------------------------------------- | ----------------------------------------------------------------------------------------------------------------- | ---------------- | ---------------- |
| MEM-STORAGE-001 | escaped value cannot lower as trusted stack region task actor or island storage | allocplan:storageV9:escape:storage_escape_rejected                    | storage_escape_validator                | storage_escape_rejected                  | TestVerifyPlanRejectsEscapedActualTrustedLowering                                                                 | linux-x64:narrow | rejected         |
| MEM-STORAGE-002 | trusted stack or region storage requires compiler-owned no-escape proof         | allocplan:storageV9:noescape:trusted_storage_requires_no_escape_proof | storage_no_escape_proof_validator       | trusted_storage_requires_no_escape_proof | TestVerifyPlanRejectsTrustedStorageWithoutNoEscapeProof                                                           | linux-x64:narrow | validated_narrow |
| MEM-STORAGE-003 | heap or conservative fallback preserves source_fact_id and reason               | allocplan:storageV9:fallback:heap_fallback_reason_preserved           | heap_fallback_reason_validator          | heap_fallback_reason_preserved           | TestFromPLIRAndAllocPlanRejectsHeapFallbackWithoutReason,TestValidateMemoryReportRejectsHeapFallbackWithoutReason | linux-x64:narrow | validated_narrow |
| MEM-STORAGE-004 | async task actor FFI or unknown-call escape keeps storage conservative          | allocplan:storageV9:boundary:boundary_storage_conservative            | boundary_storage_conservative_validator | boundary_storage_conservative            | TestVerifyPlanRejectsEscapedActualTrustedLowering,TestMiniMemoryModelV9StorageCases                               | linux-x64:narrow | conservative     |

## Notes

- `MemoryFactGraph` remains the truth source.
- `tetra.memory-report.v1` rows remain projections.
- `allocplan.VerifyPlan` rejects escaped allocations whose planned, reported, or actual lowering
  storage uses trusted stack, register, region, function-temp region, explicit island, task-region,
  actor-move-region, or non-empty eliminated storage without the required proof state.
- `compiler/internal/memoryfacts` and `ValidateReport` reject heap or conservative trusted-storage
  fallbacks without a reviewable `reason`; report schema validation already requires
  `source_fact_id`.
- `validate-memory-correlation` recognizes exactly the four `MEM-STORAGE-*` rows and rejects
  missing, extra, or widened v9 rows.
