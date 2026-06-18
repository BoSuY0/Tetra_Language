# Memory Ideal Vertical Slice v11 Dynamic Protocol Correlation

Status: accepted narrow correlation target for `MEM-DYNPROTO-011`.

This table is the exact v11 row set. `MemoryFactGraph` remains the truth source;
`tetra.memory-report.v1` rows are projections.

| requirement_id   | claim                                                                                                  | source_fact_id                                                        | validator                                         | report_row                              | negative_test                                                                                                                                                                                                                                       | target_level     | status           |
| ---------------- | ------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------- | ------------------------------------------------- | --------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------- | ---------------- |
| MEM-DYNPROTO-001 | dynamic existential or protocol borrow carriers remain conservative unless statically resolved         | memorymodel:dynprotoV11:dynamic:existential_borrow_conservative       | dynamic_existential_borrow_conservative_validator | dynamic_existential_borrow_conservative | TestMiniMemoryModelV11DynamicProtocolWitnessCases,TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts                                                                                                                                             | linux-x64:narrow | conservative     |
| MEM-DYNPROTO-002 | static witness or conformance proof may carry borrow facts only with compiler-owned parent fact        | memorymodel:dynprotoV11:witness:static_witness_parent_fact            | static_witness_parent_fact_validator              | static_witness_borrow_parent_validated  | TestMiniMemoryModelV11DynamicProtocolWitnessCases,TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts,TestValidateMemoryReportRejectsV11DerivedRowsWithoutParent                                                                                  | linux-x64:narrow | validated_narrow |
| MEM-DYNPROTO-003 | dynamic protocol dispatch cannot validate broad noalias                                                | memorymodel:dynprotoV11:dispatch:dynamic_protocol_noalias_rejected    | dynamic_protocol_noalias_rejection_validator      | dynamic_protocol_noalias_rejected       | TestMiniMemoryModelV11DynamicProtocolWitnessCases,TestValidateMemoryReportRejectsBroadNoAliasClaim                                                                                                                                                  | linux-x64:narrow | rejected         |
| MEM-DYNPROTO-004 | witness or conformance table lookup cannot promote unsafe dynamic unknown provenance to safe_known     | memorymodel:dynprotoV11:witness:witness_provenance_promotion_rejected | witness_provenance_promotion_validator            | witness_provenance_promotion_rejected   | TestMiniMemoryModelV11DynamicProtocolWitnessCases,TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown                                                                                                                                         | linux-x64:narrow | rejected         |
| MEM-DYNPROTO-005 | protocol or existential dispatch report rows preserve source_fact_id cost_class and normal_build_check | report:v11:dynproto:protocol_dispatch_report_integrity                | protocol_dispatch_report_integrity_validator      | protocol_dispatch_report_integrity      | TestMemoryIdealV11ProjectsDynamicProtocolWitnessFacts,TestValidateMemoryReportRejectsV11ProtocolDispatchIntegrityWithoutReportFields,TestValidateReportProjectionRejectsAlteredCostClass,TestValidateReportProjectionRejectsDroppedNormalBuildCheck | linux-x64:narrow | validated_narrow |

## Validator Discipline

- `validate-memory-correlation` recognizes exactly the five `MEM-DYNPROTO-*` rows and rejects
  missing, extra, or widened v11 rows.
- Dynamic existential/protocol carriers stay conservative unless statically resolved by
  compiler-owned evidence.
- Static witness/conformance rows require a parent fact and do not carry broad runtime or ABI
  claims.
- Dynamic protocol noalias and unsafe/unknown witness provenance promotion are rejected.
- Report integrity rows preserve `source_fact_id`, `cost_class`, and `normal_build_check`.

## Nonclaims

No full trait-object/existential runtime proof, complete witness-table ABI safety proof, production
dynamic dispatch runtime safety claim, target parity, performance claim, broad noalias, arbitrary
unsafe/external pointer promotion, "Memory 100%", or clean-release claim while the worktree is
dirty.
