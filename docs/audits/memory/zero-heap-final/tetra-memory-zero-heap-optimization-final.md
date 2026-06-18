# Tetra Memory Zero-Heap Optimization Final Audit

Date: 2026-06-16. Status: final memory optimization audit with MEM-11 report evidence and MEM-12
verification evidence recorded in the workflow kernel.

## Primary Evidence

- Final Tier 1 report:
  `reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/report.json`
- Final Tier 1 summary:
  `reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/summary.md`
- Host-pinned local RSS budget policy:

```text
reports/benchmark-vnext-memory-baseline/
tier1-after-memory-zero-heap-optimization/rss-budget-policy.local.json
```

- Report generated at: `2026-06-16T16:18:13Z`.
- Host: linux/amd64, 32 CPUs, `Intel(R) Core(TM) i9-14900HX`.
- Git commit recorded by the report: `95bfd4a887bab5032437cb22494d034e82ae6d35`.
- Policy: `tier1_local_benchmark_evidence`, 5 iterations, comparable threshold `0.20`.

The report validates with `tools/cmd/validate-local-benchmark-tier1`.

## Matrix Shape

| Metric                      | Value |
| --------------------------- | ----: |
| Categories                  |    17 |
| Rows                        |    68 |
| Tetra rows                  |    17 |
| Measured Tetra rows         |    17 |
| Tetra zero-heap rows        |    12 |
| Tetra rows still using heap |     5 |

## Zero-Heap Rows

These Tetra rows have runtime-measured heap total bytes `0`, heap allocation count `0`, and heap
peak bytes `0` in the final report:

Record: integer loops.
Tetra row: `integer_loops_tetra`.
Classification: blocked by fallback backend.
Backend path: fallback.
Remaining blocker: `unsupported_control_flow`.

Record: function calls.
Tetra row: `function_calls_tetra`.
Classification: faster than C/C++/Rust locally.
Backend path: register.
Remaining blocker: none in backend report.

Record: recursion.
Tetra row: `recursion_tetra`.
Classification: blocked by fallback backend.
Backend path: fallback.
Remaining blocker: `unsupported_control_flow`.

Record: matrix multiply.
Tetra row: `matrix_multiply_tetra`.
Classification: blocked by bounds check.
Backend path: fallback.
Remaining blocker: `unsupported_effect_runtime_call`; 7 bounds checks left.

Record: hash table.
Tetra row: `hash_table_tetra`.
Classification: blocked by fallback backend.
Backend path: fallback.
Remaining blockers: `unsupported_control_flow`,
`unsupported_effect_runtime_call`.

Record: allocation.
Tetra row: `allocation_tetra`.
Classification: blocked by fallback backend.
Backend path: fallback.
Remaining blocker: `unsupported_effect_runtime_call`.

Record: region/island allocation.
Tetra row: `region_island_allocation_tetra`.
Classification: blocked by fallback backend.
Backend path: fallback.
Remaining blocker: `unsupported_effect_runtime_call`.

Record: actor ping-pong.
Tetra row: `actor_ping_pong_tetra`.
Classification: blocked by actor/runtime limitation.
Backend path: fallback.
Remaining blocker: bounded actor runtime evidence only.

Record: parallel map/reduce.
Tetra row: `parallel_map_reduce_tetra`.
Classification: blocked by actor/runtime limitation.
Backend path: fallback.
Remaining blocker: bounded task runtime evidence only.

Record: startup time.
Tetra row: `startup_time_tetra`.
Classification: faster than C/C++/Rust locally.
Backend path: register.
Remaining blocker: none in backend report.

Record: binary size.
Tetra row: `binary_size_tetra`.
Classification: comparable.
Backend path: register.
Remaining blocker: none in backend report.

Record: compile time.
Tetra row: `compile_time_tetra`.
Classification: faster than C/C++/Rust locally.
Backend path: fallback.
Remaining blocker: `unsupported_control_flow`.

Important: zero Tetra heap is not zero RSS and not a universal zero-heap claim. Several zero-heap
rows still have fallback backend, bounds-check, or actor/task runtime limitations.

## Rows Still Using Heap

These Tetra rows still allocate runtime heap in the final report. Every heap row has explicit heap
reason codes.

Record: slice sum.
Tetra row: `slice_sum_tetra`.
Heap total bytes: `16384`.
Heap count: `1`.
Heap reason codes: `heap.required_large_object`.

Record: bounds-check loops.
Tetra row: `bounds_check_loops_tetra`.
Heap total bytes: `16384`.
Heap count: `1`.
Heap reason codes: `heap.required_large_object`.

Record: JSON parse/stringify.
Tetra row: `json_parse_stringify_tetra`.
Heap total bytes: `128`.
Heap count: `1`.
Heap reason codes: `heap.required_unknown_call`.

Record: HTTP plaintext/json.
Tetra row: `http_plaintext_json_tetra`.
Heap total bytes: `384`.
Heap count: `2`.
Heap reason codes: `heap.required_unknown_call`.

Record: PostgreSQL single/multiple/update.
Tetra row: `postgresql_single_multiple_update_tetra`.
Heap total bytes: `64`.
Heap count: `1`.
Heap reason codes: `heap.required_unknown_call`.

Current total runtime-measured Tetra heap allocation across these five rows is `33344` bytes and `6`
allocations.

## Region And Island Status

`region_island_allocation_tetra` is now measured, not build-failed.

Final report evidence:

- runtime heap total bytes: `0`;
- runtime heap allocation count: `0`;
- bytes requested/reserved/committed/released from allocation-report evidence: `64/64/64/64`;
- domain kind: `island`;
- domain evidence class: `allocation_report_estimate`;
- classification remains `blocked by fallback backend`.

This proves the first explicit island allocation path is closed for the local Tier 1 row. It does
not prove production region allocation for all programs.

## Actor-Domain Status

MEM-9 implemented local `parallelrt` actor memory-domain evidence and byte-based backpressure. That
evidence lives in:

- `docs/audits/memory/zero-heap-runtime/tetra-memory-zero-heap-mem9-actor-domains.md`;
- `.workflow/tetra-memory-zero-heap-optimization-goal/verification/mem9-actor-domains.md`.

The final Tier 1 actor/task rows remain bounded benchmark evidence:

Record: actor ping-pong.
Runtime object feature: `actor_runtime`.
Tier 1 domain evidence: unsupported.
Status: blocked by actor/runtime limitation.

Record: parallel map/reduce.
Runtime object feature: `task_runtime`.
Tier 1 domain evidence: unsupported.
Status: blocked by actor/runtime limitation.

This is intentional. The final report does not claim production actor-runtime per-actor memory
sampling or distributed zero-copy.

## RSS Budget Status

Process RSS is measured separately from Tetra heap.

Final Tetra row RSS peak range:

- minimum: `integer_loops_tetra`, `11247616` bytes;
- maximum: rows with `15769600` bytes, including `compile_time_tetra`.

The final report directory includes a host-pinned local RSS policy:

```text
reports/benchmark-vnext-memory-baseline/
tier1-after-memory-zero-heap-optimization/rss-budget-policy.local.json
```

That policy pins:

- target `linux-x64`;
- host `linux/amd64`;
- 32 CPUs;
- `Intel(R) Core(TM) i9-14900HX`;
- git commit `95bfd4a887bab5032437cb22494d034e82ae6d35`;
- per-category Tetra `rss_peak` budgets with `5%` local variance.

The policy validates against the final report with `--rss-budget-policy`.

This is a local regression gate. It is not a cross-machine RSS comparison and not an official
benchmark result.

## Bytes Copied Summary

Final Tier 1 Tetra memory evidence reports:

```text
bytes_copied rows = 17
rows with explicit bytes_copied.bytes = 0
positive bytes_copied rows = 0
interpreted bytes_copied total = 0
evidence class = allocation_report_estimate for all 17 Tetra rows
```

These are allocation-report estimates, not runtime copy counters. The report schema uses `omitempty`
for zero-valued byte fields, so the final audit treats missing `bytes_copied.bytes` as zero only at
report-summary level and does not promote it to runtime-measured copy evidence.

## Domain Evidence Summary

Final Tier 1 Tetra rows:

| Domain evidence class      | Row count |
| -------------------------- | --------: |
| allocation_report_estimate |         9 |
| unsupported                |         8 |

Domain kinds observed in Tier 1 allocation-report evidence:

- `process`;
- `island`.

No final Tier 1 row claims runtime-measured domain bytes. Runtime actor-domain work remains
represented by the separate MEM-9 local `parallelrt` evidence.

## Known Target Limitations

- Fallback backend remains a blocker for multiple rows.
- Bounds-check elimination remains incomplete for `slice sum`, `bounds-check loops`, and
  `matrix multiply`.
- JSON, HTTP, and PostgreSQL rows are deterministic local helper kernels, not full service/database
  benchmarks.
- Actor/task rows are bounded local evidence, not production runtime benchmark claims.
- RSS budgets are local host-pinned gates only.
- Domain bytes in Tier 1 are either allocation-report estimates or unsupported.
- Heap-free Tier 1 rows do not imply heap-free behavior for every Tetra program.

## Nonclaims

- No universal zero heap.
- No zero RSS.
- No official benchmark result.
- No cross-machine RSS claim.
- No production OS memory claim.
- No allocator superiority claim.
- No production actor-runtime memory sampler claim.
- No distributed actor zero-copy claim.
- No Linux RSS behavior promoted into Tetra language semantics.

## Verification Commands

The final report and local RSS budget policy were checked with:

```sh
report_dir="reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization"
policy="$report_dir/rss-budget-policy.local.json"

GOCACHE=$(pwd)/.cache/go-build-memory-final \
go run ./tools/cmd/local-benchmark-tier1 \
  --out-dir "$report_dir" \
  --iterations 5

GOCACHE=$(pwd)/.cache/go-build-memory-final \
go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report "$report_dir/report.json"

GOCACHE=$(pwd)/.cache/go-build-memory-final \
go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report "$report_dir/report.json" \
  --rss-budget-policy "$policy"
```

Result: passed.
