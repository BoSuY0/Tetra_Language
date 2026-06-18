# Tetra Memory Zero-Heap Optimization Final Audit

Date: 2026-06-16.
Status: final memory optimization audit with MEM-11 report evidence and MEM-12
verification evidence recorded in the workflow kernel.

## Primary Evidence

- Final Tier 1 report:
  `reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/report.json`
- Final Tier 1 summary:
  `reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/summary.md`
- Host-pinned local RSS budget policy:
  `reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/rss-budget-policy.local.json`
- Report generated at: `2026-06-16T16:18:13Z`.
- Host: linux/amd64, 32 CPUs,
  `Intel(R) Core(TM) i9-14900HX`.
- Git commit recorded by the report:
  `95bfd4a887bab5032437cb22494d034e82ae6d35`.
- Policy: `tier1_local_benchmark_evidence`, 5 iterations,
  comparable threshold `0.20`.

The report validates with `tools/cmd/validate-local-benchmark-tier1`.

## Matrix Shape

| Metric | Value |
| --- | ---: |
| Categories | 17 |
| Rows | 68 |
| Tetra rows | 17 |
| Measured Tetra rows | 17 |
| Tetra zero-heap rows | 12 |
| Tetra rows still using heap | 5 |

## Zero-Heap Rows

These Tetra rows have runtime-measured heap total bytes `0`, heap allocation
count `0`, and heap peak bytes `0` in the final report:

| Category | Tetra row | Classification | Backend path | Remaining blocker |
| --- | --- | --- | --- | --- |
| integer loops | `integer_loops_tetra` | blocked by fallback backend | fallback | `unsupported_control_flow` |
| function calls | `function_calls_tetra` | faster than C/C++/Rust locally | register | none in backend report |
| recursion | `recursion_tetra` | blocked by fallback backend | fallback | `unsupported_control_flow` |
| matrix multiply | `matrix_multiply_tetra` | blocked by bounds check | fallback | `unsupported_effect_runtime_call`; 7 bounds checks left |
| hash table | `hash_table_tetra` | blocked by fallback backend | fallback | `unsupported_control_flow`, `unsupported_effect_runtime_call` |
| allocation | `allocation_tetra` | blocked by fallback backend | fallback | `unsupported_effect_runtime_call` |
| region/island allocation | `region_island_allocation_tetra` | blocked by fallback backend | fallback | `unsupported_effect_runtime_call` |
| actor ping-pong | `actor_ping_pong_tetra` | blocked by actor/runtime limitation | fallback | bounded actor runtime evidence only |
| parallel map/reduce | `parallel_map_reduce_tetra` | blocked by actor/runtime limitation | fallback | bounded task runtime evidence only |
| startup time | `startup_time_tetra` | faster than C/C++/Rust locally | register | none in backend report |
| binary size | `binary_size_tetra` | comparable | register | none in backend report |
| compile time | `compile_time_tetra` | faster than C/C++/Rust locally | fallback | `unsupported_control_flow` |

Important: zero Tetra heap is not zero RSS and not a universal zero-heap claim.
Several zero-heap rows still have fallback backend, bounds-check, or actor/task
runtime limitations.

## Rows Still Using Heap

These Tetra rows still allocate runtime heap in the final report. Every heap row
has explicit heap reason codes.

| Category | Tetra row | Heap total bytes | Heap count | Heap reason codes |
| --- | --- | ---: | ---: | --- |
| slice sum | `slice_sum_tetra` | 16384 | 1 | `heap.required_large_object` |
| bounds-check loops | `bounds_check_loops_tetra` | 16384 | 1 | `heap.required_large_object` |
| JSON parse/stringify | `json_parse_stringify_tetra` | 128 | 1 | `heap.required_unknown_call` |
| HTTP plaintext/json | `http_plaintext_json_tetra` | 384 | 2 | `heap.required_unknown_call` |
| PostgreSQL single/multiple/update | `postgresql_single_multiple_update_tetra` | 64 | 1 | `heap.required_unknown_call` |

Current total runtime-measured Tetra heap allocation across these five rows is
`33344` bytes and `6` allocations.

## Region And Island Status

`region_island_allocation_tetra` is now measured, not build-failed.

Final report evidence:

- runtime heap total bytes: `0`;
- runtime heap allocation count: `0`;
- bytes requested/reserved/committed/released from allocation-report evidence:
  `64/64/64/64`;
- domain kind: `island`;
- domain evidence class: `allocation_report_estimate`;
- classification remains `blocked by fallback backend`.

This proves the first explicit island allocation path is closed for the local
Tier 1 row. It does not prove production region allocation for all programs.

## Actor-Domain Status

MEM-9 implemented local `parallelrt` actor memory-domain evidence and
byte-based backpressure. That evidence lives in:

- `docs/audits/tetra-memory-zero-heap-mem9-actor-domains.md`;
- `.workflow/tetra-memory-zero-heap-optimization-goal/verification/mem9-actor-domains.md`.

The final Tier 1 actor/task rows remain bounded benchmark evidence:

| Category | Runtime object feature | Tier 1 domain evidence | Status |
| --- | --- | --- | --- |
| actor ping-pong | `actor_runtime` | unsupported | blocked by actor/runtime limitation |
| parallel map/reduce | `task_runtime` | unsupported | blocked by actor/runtime limitation |

This is intentional. The final report does not claim production actor-runtime
per-actor memory sampling or distributed zero-copy.

## RSS Budget Status

Process RSS is measured separately from Tetra heap.

Final Tetra row RSS peak range:

- minimum: `integer_loops_tetra`, `11247616` bytes;
- maximum: rows with `15769600` bytes, including `compile_time_tetra`.

The final report directory includes a host-pinned local RSS policy:

```text
reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/rss-budget-policy.local.json
```

That policy pins:

- target `linux-x64`;
- host `linux/amd64`;
- 32 CPUs;
- `Intel(R) Core(TM) i9-14900HX`;
- git commit `95bfd4a887bab5032437cb22494d034e82ae6d35`;
- per-category Tetra `rss_peak` budgets with `5%` local variance.

The policy validates against the final report with
`--rss-budget-policy`.

This is a local regression gate. It is not a cross-machine RSS comparison and
not an official benchmark result.

## Bytes Copied Summary

Final Tier 1 Tetra memory evidence reports:

```text
bytes_copied rows = 17
rows with explicit bytes_copied.bytes = 0
positive bytes_copied rows = 0
interpreted bytes_copied total = 0
evidence class = allocation_report_estimate for all 17 Tetra rows
```

These are allocation-report estimates, not runtime copy counters. The report
schema uses `omitempty` for zero-valued byte fields, so the final audit treats
missing `bytes_copied.bytes` as zero only at report-summary level and does not
promote it to runtime-measured copy evidence.

## Domain Evidence Summary

Final Tier 1 Tetra rows:

| Domain evidence class | Row count |
| --- | ---: |
| allocation_report_estimate | 9 |
| unsupported | 8 |

Domain kinds observed in Tier 1 allocation-report evidence:

- `process`;
- `island`.

No final Tier 1 row claims runtime-measured domain bytes. Runtime actor-domain
work remains represented by the separate MEM-9 local `parallelrt` evidence.

## Known Target Limitations

- Fallback backend remains a blocker for multiple rows.
- Bounds-check elimination remains incomplete for `slice sum`,
  `bounds-check loops`, and `matrix multiply`.
- JSON, HTTP, and PostgreSQL rows are deterministic local helper kernels, not
  full service/database benchmarks.
- Actor/task rows are bounded local evidence, not production runtime benchmark
  claims.
- RSS budgets are local host-pinned gates only.
- Domain bytes in Tier 1 are either allocation-report estimates or unsupported.
- Heap-free Tier 1 rows do not imply heap-free behavior for every Tetra
  program.

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
GOCACHE=$(pwd)/.cache/go-build-memory-final go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization --iterations 5
GOCACHE=$(pwd)/.cache/go-build-memory-final go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/report.json
GOCACHE=$(pwd)/.cache/go-build-memory-final go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/report.json --rss-budget-policy reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/rss-budget-policy.local.json
```

Result: passed.
