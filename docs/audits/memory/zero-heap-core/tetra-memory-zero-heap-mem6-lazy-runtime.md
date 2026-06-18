# Tetra Memory Zero-Heap MEM-6 Lazy Runtime Evidence

Status: COMPLETE for MEM-6.

This audit closes MEM-6 with compiler-owned runtime object/link/init evidence. It does not claim ELF
symbol-table proof.

## Implemented

- Backend reports expose static runtime feature evidence:
  - `runtime_features_required`
  - `runtime_features_linked`
  - `runtime_features_initialized`
  - `runtime_lazy_init_blockers`
  - evidence class `lowered_ir_static_plan`
  - evidence method `backend_report_lowered_ir_scan_v1`
- Backend reports also expose native runtime object plan evidence:
  - `summary.runtime_object_plan.evidence_class = native_runtime_object_plan`
  - `summary.runtime_object_plan.evidence_method = native_link_runtime_object_plan_v1`
  - `runtime_used`
  - `runtime_object_linked`
  - `runtime_object_initialized`
  - `runtime_object_features_required`
  - `runtime_object_features_linked`
  - `runtime_object_features_initialized`
  - `runtime_object_lazy_init_blockers`
- `runtime_object_plan` is built from the same semantic runtime usage collectors and
  `buildruntime.DecideRuntimeObjectPlan` decision path used by native linking.
- Tier 1 metadata mirrors both evidence groups.
- Tier 1 validator rejects missing or mismatched runtime feature/object-plan evidence.

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

## Runtime Object Plan Facts

Fresh report facts:

- Tetra rows: 17.
- Rows with `lowered_ir_static_plan`: 17.
- Rows with `native_runtime_object_plan`: 17.
- Rows with `runtime_object_plan.runtime_used = true`: 2.
- Rows with `unknown_runtime`: 0.

Rows with runtime object usage:

| Row                         | runtime_used | linked | initialized | runtime_object_features_required |
| --------------------------- | -----------: | -----: | ----------: | -------------------------------- |
| `actor_ping_pong_tetra`     |         true |   true |        true | `actor_runtime`                  |
| `parallel_map_reduce_tetra` |         true |   true |        true | `task_runtime`                   |

Protected simple rows:

| Row                    | heap_allocations | heap total alloc bytes | heap allocation count | runtime_used | linked runtime object features |
| ---------------------- | ---------------: | ---------------------: | --------------------: | -----------: | ------------------------------ |
| `integer_loops_tetra`  |                0 |                      0 |                     0 |        false | empty                          |
| `function_calls_tetra` |                0 |                      0 |                     0 |        false | empty                          |
| `hash_table_tetra`     |                0 |                      0 |                     0 |        false | empty                          |
| `startup_time_tetra`   |                0 |                      0 |                     0 |        false | empty                          |

## Verification

Focused sidecar tests:

```text
GOCACHE=$(pwd)/.cache/go-build-mem6-sidecar go test ./compiler ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -run 'RuntimeObjectPlan|RuntimeFeature|ValidateReportAcceptsCompleteP25Tier1Matrix|ValidateReportAcceptsBuildFailedTetraRowWithMissingBuildArtifacts|CollectTetraMetadataAttachesRuntimeFeatureEvidence|MissingTetraMetadataAttachesBlockedMemoryEvidence' -count=1
```

Result: exit 0.

Representative MEM-6 command:

```text
GOCACHE=$(pwd)/.cache/go-build-lazy-runtime-report go test ./compiler ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -run 'Runtime|Feature|Report|Benchmark|Validate|Link|Init|Object' -count=1
```

Result: exit 0.

Fresh report generation:

```text
GOCACHE=$(pwd)/.cache/go-build-lazy-runtime-bench go run ./tools/cmd/local-benchmark-tier1 -out-dir reports/benchmark-vnext-memory-baseline/tier1-after-lazy-runtime-track -iterations 1 -timeout 20s
```

Result: exit 0.

Fresh report validator:

```text
GOCACHE=$(pwd)/.cache/go-build-lazy-runtime-validate go run ./tools/cmd/validate-local-benchmark-tier1 -report reports/benchmark-vnext-memory-baseline/tier1-after-lazy-runtime-track/report.json
```

Result: exit 0.

## Nonclaims

- This is not an ELF symbol-table proof.
- Current generated Tetra binaries are static ELF files with no section header and no symbols, so
  `nm` and `readelf -s` cannot prove absence of runtime symbols.
- The claim is narrower and compiler-owned: the native runtime object decision path reports no
  runtime object linked or initialized for simple protected rows, and the validator requires that
  report evidence.
- This does not optimize RSS by itself; it makes lazy runtime link/init evidence visible and
  enforceable in local benchmark artifacts.

## Next

Proceed to MEM-7: remaining heap allocations need explicit reason codes explaining why they could
not become eliminated, register, stack, region/island, or domain allocations.
