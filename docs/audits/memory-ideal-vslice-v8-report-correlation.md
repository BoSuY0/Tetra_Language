# Memory Ideal Vertical Slice v8 Report Integrity Correlation

Status: accepted narrow correlation target for `MEM-REPORT-008`.

This table is intentionally exact. It adds only graph/report projection and
claim-drift integrity evidence. It does not add new memory semantics, optimizer
behavior, FFI/runtime proof, target parity, performance evidence, arbitrary
external pointer safety, or a "Memory 100%" claim.

| requirement_id | claim | source_fact_id | validator | report_row | negative_test | target_level | status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| MEM-REPORT-001 | every report row maps to a MemoryFactGraph source fact | report:v8:source-map | report_graph_projection_validator | report_graph_projection | TestValidateReportProjectionRejectsUnknownSourceFactID | all:report | validated_narrow |
| MEM-REPORT-002 | every graph fact requiring projection appears in the report | report:v8:projection-complete | report_projection_completeness_validator | report_projection_completeness | TestValidateReportProjectionRejectsMissingProjectedGraphFact | all:report | validated_narrow |
| MEM-REPORT-003 | report projection preserves cost_class and normal_build_check | report:v8:projection-fields | cost_class_preservation_validator,normal_build_check_preservation_validator | report_projection_field_preservation | TestValidateReportProjectionRejectsAlteredCostClass,TestValidateReportProjectionRejectsDroppedNormalBuildCheck | all:report | validated_narrow |
| MEM-REPORT-004 | correlation docs reject extra missing or widened rows | report:v8:correlation-exact | correlation_exact_row_validator | correlation_exact_row_set | TestValidateMemoryCorrelationRejectsV8MissingClaimDriftRow,TestValidateMemoryCorrelationRejectsV8ExtraRow | all:docs | validated_narrow |
| MEM-REPORT-005 | memory release or audit docs cannot claim broad safety from conservative or rejected rows | report:v8:claim-drift | memory_claim_drift_validator | memory_claim_drift | TestValidateMemoryCorrelationRejectsV8BroadSafetyClaimDrift | all:docs | rejected |

## Notes

- `MemoryFactGraph` remains the truth source.
- `tetra.memory-report.v1` rows remain projections.
- `ValidateReportProjection` validates a report against graph facts and rejects
  unknown `source_fact_id`, missing projected facts, altered projection fields,
  and changed `cost_class` or `normal_build_check`.
- `validate-memory-correlation` recognizes the exact v8 row set and rejects
  missing, extra, widened, or broad-safety claim-drift rows for this slice.
