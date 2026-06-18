# MEM-3 Zero-Heap Gates Audit

Date: 2026-06-16. Scope: MEM-3 of
`docs/plan/2026-06-16-tetra-memory-zero-heap-optimization-plan.md`.

## Result

MEM-3 is implemented for the approved simple Tier 1 rows.

The current zero-heap-required categories are defined in
`docs/spec/telemetry/zero_heap_benchmark_policy.md`:

```text
integer loops
function calls
hash table
startup time
```

The validator gate is implemented in `tools/cmd/validate-local-benchmark-tier1/main.go`.

## Validator Behavior

For a measured Tetra row in a zero-heap-required category, validation now requires:

```text
tetra_metadata.heap_allocations == 0
heap_alloc_bytes.evidence_class == runtime_measured
heap_alloc_bytes.total_alloc_bytes == 0
heap_alloc_bytes.allocation_count == 0
heap_alloc_bytes bytes/current/peak fields == 0
runtime heap sidecar total/count/current/peak fields == 0
```

Excluded rows still require truthful memory evidence, but they do not fail merely because they
allocate heap.

## Fresh Report Evidence

Fresh local report:

```text
reports/benchmark-vnext-memory-baseline/tier1-after-zero-heap-gates/report.json
```

Protected Tetra row evidence:

| Category       | Row                    | Status   | Metadata heap | Heap evidence    | Total alloc bytes | Allocation count |
| -------------- | ---------------------- | -------- | ------------: | ---------------- | ----------------: | ---------------: |
| integer loops  | `integer_loops_tetra`  | measured |             0 | runtime_measured |                 0 |                0 |
| function calls | `function_calls_tetra` | measured |             0 | runtime_measured |                 0 |                0 |
| hash table     | `hash_table_tetra`     | measured |             0 | runtime_measured |                 0 |                0 |
| startup time   | `startup_time_tetra`   | measured |             0 | runtime_measured |                 0 |                0 |

Runtime heap sidecars for those rows report:

```text
heap_current_bytes = 0
heap_peak_bytes = 0
heap_total_alloc_bytes = 0
heap_allocation_count = 0
```

Rows deliberately excluded by policy still include measured heap allocation in the fresh report, for
example:

| Category                          | Heap allocations | Total alloc bytes | Allocation count |
| --------------------------------- | ---------------: | ----------------: | ---------------: |
| slice sum                         |                1 |             16384 |                1 |
| bounds-check loops                |                1 |             16384 |                1 |
| JSON parse/stringify              |                1 |               128 |                1 |
| HTTP plaintext/json               |                2 |               384 |                2 |
| PostgreSQL single/multiple/update |                1 |                64 |                1 |

## Verification

Passed:

```sh
GOCACHE=$(pwd)/.cache/go-build-zero-heap-validator-green go test ./tools/cmd/validate-local-benchmark-tier1 -run 'ZeroHeap|MemoryEvidence|Heap' -count=1
GOCACHE=$(pwd)/.cache/go-build-zero-heap-validator-green go test ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -count=1
GOCACHE=$(pwd)/.cache/go-build-zero-heap-validator-green go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-const-length-track-final/report.json
GOCACHE=$(pwd)/.cache/go-build-zero-heap-suite go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-after-zero-heap-gates --iterations 3
GOCACHE=$(pwd)/.cache/go-build-zero-heap-suite go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-zero-heap-gates/report.json
git diff --check -- tools/cmd/validate-local-benchmark-tier1/main.go tools/cmd/validate-local-benchmark-tier1/main_test.go docs/spec/telemetry/zero_heap_benchmark_policy.md
graphify update .
```

Failed outside MEM-3 scope:

```sh
GOCACHE=$(pwd)/.cache/go-build-zero-heap-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

Failure:

```text
verify-docs: lib/core/block/block.tetra: effects metadata mismatch: got alloc, mem want none
```

The failure is not caused by the zero-heap validator or policy files, but it remains a final
verification risk for MEM-12.

## Remaining Work

MEM-4 is still open: dedicated zero-heap microbenchmarks must either be added outside the comparable
Tier 1 matrix or the plan must record a final placement decision with evidence.
