# Memory Ideal Vertical Slice v8 Report Integrity Final Audit

Status: `validated_narrow` for the bounded report-integrity surface.

Decision: proceed for v8 evidence. This slice accepts only graph/report projection integrity and
claim-drift rejection. It is not "Memory 100%", not a new memory semantics slice, not optimizer
correctness, not target parity, not performance evidence, and not arbitrary unsafe or FFI safety.

## Requirement Results

| requirement_id   | status             | evidence                                                                                                                                                                                                                                                      |
| ---------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `MEM-REPORT-001` | `validated_narrow` | `ValidateReportProjection` rejects report rows whose `source_fact_id` does not exist in the supplied `MemoryFactGraph`; negative test: `TestValidateReportProjectionRejectsUnknownSourceFactID`.                                                              |
| `MEM-REPORT-002` | `validated_narrow` | `ValidateReportProjection` rejects reports that omit graph facts projected by `BuildReportFromGraph`; negative test: `TestValidateReportProjectionRejectsMissingProjectedGraphFact`.                                                                          |
| `MEM-REPORT-003` | `validated_narrow` | Projection comparison rejects altered fields, including `cost_class` and `normal_build_check`; negative tests: `TestValidateReportProjectionRejectsAlteredCostClass`, `TestValidateReportProjectionRejectsDroppedNormalBuildCheck`.                           |
| `MEM-REPORT-004` | `validated_narrow` | `validate-memory-correlation` recognizes exactly the five `MEM-REPORT-*` rows and rejects missing, extra, or widened v8 rows; negative tests: `TestValidateMemoryCorrelationRejectsV8MissingClaimDriftRow`, `TestValidateMemoryCorrelationRejectsV8ExtraRow`. |
| `MEM-REPORT-005` | `rejected`         | `memory_claim_drift_validator` rejects broad-safety wording such as "Memory 100%" or broad safety proven from conservative/rejected evidence; negative test: `TestValidateMemoryCorrelationRejectsV8BroadSafetyClaimDrift`.                                   |

## Validator Map

| validator                                   | implementation                                                                               |
| ------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `report_graph_projection_validator`         | `compiler/internal/memoryfacts.ValidateReportProjection` unknown-source check.               |
| `report_projection_completeness_validator`  | `compiler/internal/memoryfacts.ValidateReportProjection` graph fact completeness check.      |
| `cost_class_preservation_validator`         | `compiler/internal/memoryfacts.ValidateReportProjection` canonical `rowFromFact` comparison. |
| `normal_build_check_preservation_validator` | `compiler/internal/memoryfacts.ValidateReportProjection` canonical `rowFromFact` comparison. |
| `correlation_exact_row_validator`           | `tools/cmd/validate-memory-correlation` v8 required row set and status checks.               |
| `memory_claim_drift_validator`              | `tools/cmd/validate-memory-correlation` broad memory claim drift checks.                     |

## Nonclaims

- No "Memory 100% complete" claim.
- No new memory semantics.
- No optimizer rewrite or broad optimizer correctness proof.
- No arbitrary external pointer safety.
- No FFI/runtime lifetime proof.
- No target parity.
- No performance claim.
- No production runtime/ABI proof.
- No clean-release claim while `git status --short` remains dirty.

## Current Gate Evidence

Focused RED was observed before implementation:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-memoryfacts-red go test ./compiler/internal/memoryfacts -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-tools-red go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1
```

Focused GREEN has passed:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-memoryfacts go test ./compiler/internal/memoryfacts -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v8-report-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation ./tools/cmd/validate-memory-fuzz-oracle -count=1
```

Final broad, docs, manifest, CI, hygiene, and Graphify evidence is recorded in
`.workflow/memory-ideal-vertical-slice-v8-report/final-report.md`.
