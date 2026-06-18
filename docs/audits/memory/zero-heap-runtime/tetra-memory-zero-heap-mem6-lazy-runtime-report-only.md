# Tetra Memory Zero-Heap MEM-6 Lazy Runtime Report-Only Baseline

Status: historical PARTIAL checkpoint for MEM-6.

Superseded by: `docs/audits/memory/zero-heap-core/tetra-memory-zero-heap-mem6-lazy-runtime.md`.

This audit records the first MEM-6 slice: runtime feature evidence is now reported in compiler
backend reports and mirrored into local Tier 1 benchmark metadata. This is not yet a full
binary-level lazy-link proof.

## Implemented

- Backend reports now expose:
  - `runtime_features_required`
  - `runtime_features_linked`
  - `runtime_features_initialized`
  - `runtime_lazy_init_blockers`
  - `runtime_feature_evidence_class`
  - `runtime_feature_evidence_method`
- The evidence class is `lowered_ir_static_plan`.
- The evidence method is `backend_report_lowered_ir_scan_v1`.
- `tools/cmd/local-benchmark-tier1` copies the backend runtime feature fields into each Tetra
  benchmark row metadata with `source_artifact` pointing at the backend report.
- `tools/cmd/validate-local-benchmark-tier1` rejects measured Tetra rows when runtime feature
  metadata is missing or does not match the backend report.

## Fresh Report

Fresh local Tier 1 artifact:

```text
reports/benchmark-vnext-memory-baseline/tier1-after-lazy-runtime-track/report.json
```

Validator:

```text
GOCACHE=$(pwd)/.cache/go-build-lazy-runtime-validate go run ./tools/cmd/validate-local-benchmark-tier1 -report reports/benchmark-vnext-memory-baseline/tier1-after-lazy-runtime-track/report.json
```

Result: exit 0.

## Runtime Feature Summary

Fresh report facts:

- Tetra rows: 17.
- Rows with `lowered_ir_static_plan` runtime feature evidence: 17.
- Rows requiring actor/task/heap runtime features: 7.
- Rows requiring `unknown_runtime`: 0.

Simple zero-heap-required rows:

| Row                    | heap_allocations | heap total alloc bytes | heap allocation count | runtime_features_required |
| ---------------------- | ---------------: | ---------------------: | --------------------: | ------------------------- |
| `integer_loops_tetra`  |                0 |                      0 |                     0 | empty                     |
| `function_calls_tetra` |                0 |                      0 |                     0 | empty                     |
| `hash_table_tetra`     |                0 |                      0 |                     0 | empty                     |
| `startup_time_tetra`   |                0 |                      0 |                     0 | empty                     |

Representative runtime feature rows:

| Row                              | runtime_features_required |
| -------------------------------- | ------------------------- |
| `slice_sum_tetra`                | `heap_runtime`            |
| `region_island_allocation_tetra` | `island_allocator`        |
| `actor_ping_pong_tetra`          | `actor_runtime`           |
| `parallel_map_reduce_tetra`      | `task_runtime`            |

## Verification

Focused RED/GREEN and report tests:

```text
GOCACHE=$(pwd)/.cache/go-build-mem6-report go test ./compiler ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -run 'BackendReportRuntimeFeatures|BackendCoverageSummaryCountsRowsAndCategories|CollectTetraMetadataAttachesRuntimeFeatureEvidence|MissingTetraMetadataAttachesBlockedMemoryEvidence|ValidateReportRejectsMissingRuntimeFeatureEvidence|ValidateReportRejectsRuntimeFeatureMetadataMismatch|ValidateReportAcceptsCompleteP25Tier1Matrix|ValidateReportAcceptsBuildFailedTetraRowWithMissingBuildArtifacts' -count=1
```

Result: exit 0.

MEM-6 representative report command:

```text
GOCACHE=$(pwd)/.cache/go-build-lazy-runtime-report go test ./compiler ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -run 'Runtime|Feature|Report|Benchmark|Validate' -count=1
```

Result: exit 0.

Fresh report generation:

```text
GOCACHE=$(pwd)/.cache/go-build-lazy-runtime-bench go run ./tools/cmd/local-benchmark-tier1 -out-dir reports/benchmark-vnext-memory-baseline/tier1-after-lazy-runtime-track -iterations 1 -timeout 20s
```

Result: exit 0.

## Nonclaims / Remaining Gap

- This is not an ELF symbol scan proof.
- Current Tetra binaries are static ELF files with no section header and no symbols; `nm` reports
  `no symbols`, and `readelf -s` has no dynamic symbol information.
- Therefore `runtime_features_linked` and `runtime_features_initialized` are static compiler-plan
  evidence, not binary symbol evidence.
- MEM-6 remains open until the compiler emits a runtime object/link/init sidecar or another
  target-owned artifact that proves which runtime object pieces were actually linked and
  initialized.

## Next

Add runtime object/link/init evidence near the native link path so MEM-6 can be closed without
relying on unavailable ELF symbols.
