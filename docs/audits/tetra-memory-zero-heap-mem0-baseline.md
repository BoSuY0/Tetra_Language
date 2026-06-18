# Tetra Memory Zero-Heap MEM-0 Baseline

Status: MEM-0 baseline lock.

This note records current local evidence before starting memory optimization
work. It is not an optimization patch and does not claim the final zero-heap
goal is achieved.

## Source Reports

- Active baseline report:
  `reports/benchmark-vnext-memory-baseline/tier1-after-actor-track/report.json`
- Cross-check report:
  `reports/benchmark-vnext-memory-baseline/tier1-after-hash-track/report.json`
- Previous memory audit:
  `docs/audits/benchmark-vnext-memory-baseline.md`
- Canonical plan:
  `docs/plan/2026-06-16-tetra-memory-zero-heap-optimization-plan.md`

## Validation

Validated locally on 2026-06-16:

```sh
GOCACHE=$(pwd)/.cache/go-build-memory-zero-baseline go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-actor-track/report.json
GOCACHE=$(pwd)/.cache/go-build-memory-zero-baseline go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-hash-track/report.json
```

Both commands exited 0.

## Report Shape

`tier1-after-actor-track/report.json`:

- generated_at: `2026-06-16T10:24:39Z`
- host: `linux/amd64`, 32 CPUs, `Intel(R) Core(TM) i9-14900HX`
- git_commit: `95bfd4a887bab5032437cb22494d034e82ae6d35`
- categories: 17
- rows: 68
- Tetra rows: 17
- measured Tetra rows: 16
- build-failed Tetra rows: 1
- Tetra rows with memory evidence: 17
- runtime-measured heap rows: 16
- blocked heap rows: 1
- runtime-measured RSS peak rows: 16
- runtime-measured domain rows: 0
- allocation-estimate domain rows: 8
- unsupported domain rows: 8
- blocked domain rows: 1

`tier1-after-hash-track/report.json` has the same 17 categories, 68 rows,
17 Tetra rows, 16 measured Tetra rows, 1 build-failed Tetra row, 16
runtime-measured heap rows, and 1 blocked heap row.

## Current Key Classifications

| Category | Classification | Current blocker |
|---|---|---|
| `hash table` | `blocked by heap allocation` | `hash_table_tetra` records 2 heap allocations. |
| `region/island allocation` | `blocked by missing feature` | Tetra build fails before runtime memory artifacts are produced. |
| `actor ping-pong` | `blocked by actor/runtime limitation` | Actor-domain memory evidence is unsupported; backend is fallback. |
| `parallel map/reduce` | `blocked by actor/runtime limitation` | Actor-domain memory evidence is unsupported; backend is fallback. |

## Heap / RSS / Domain Boundary

Current reports already separate these metrics:

- `heap_alloc_bytes` is runtime-measured for successful linux-x64 Tetra rows
  with method `tetra_linux_x64_heap_telemetry_v1`.
- `rss_peak` is runtime-measured for successful linux-x64 Tetra rows with
  method `linux_wait4_rusage_maxrss_v1`.
- `rss_current` is runtime-measured from `/proc/<pid>/status` VmRSS samples.
- `bytes_requested`, `bytes_reserved`, and `bytes_copied` are still
  allocation-report estimates where present.
- `bytes_committed` is unsupported because allocation reports do not expose
  committed bytes.
- Actor-domain bytes are not runtime-measured in this baseline.
- Build-failed rows use blocked memory evidence with
  `missing_build_artifacts`.

This baseline can support local runtime heap and RSS evidence. It cannot
support cross-machine RSS claims, official benchmark claims, or production OS
memory claims.

## Hash Table State

`hash_table_tetra` remains the first memory-specific optimization target.

Evidence from `tier1-after-actor-track/report.json`:

- status: `measured`
- classification: `blocked by heap allocation`
- classification reason: `Tetra allocation report records 2 heap allocations.`
- backend path: `fallback`
- backend blockers: `unsupported_control_flow`,
  `unsupported_effect_runtime_call`
- bounds left: 4
- heap allocations: 2
- perf blockers: `allocation.local_call_heap_fallback`,
  `inline.code_size_budget`
- runtime heap: 2048 bytes current, peak, and total alloc bytes
- runtime heap allocation count: 2
- runtime heap source:
  `reports/benchmark-vnext-memory-baseline/tier1-after-actor-track/artifacts/heap-telemetry/hash_table_tetra/iteration-01.heap.json`
- RSS peak: 14086144 bytes
- RSS peak source:
  `reports/benchmark-vnext-memory-baseline/tier1-after-actor-track/artifacts/rss-telemetry/hash_table_tetra/iteration-01.rss.json`
- domain evidence: `allocation_report_estimate`
- domain: `domain:process`

Allocation summary:

```text
allocation_count: 2
storage_classes: Heap = 2
actual_lowering_storage_classes: Heap = 2
runtime_paths: heap = 2
domain: domain:process
```

Interpretation: this is not an allocator performance claim. The current blocker
is compiler proof/lowering: local-call heap fallback keeps `keys` and `values`
on heap even though this track expects the next optimization to prove the safe
stack/region case.

## Region / Island State

`region_island_allocation_tetra` is build-failed in the baseline.

Evidence:

- status: `build_failed`
- classification: `blocked by missing feature`
- memory evidence: `blocked`
- method: `missing_build_artifacts`
- blocked reason: `Tetra build failed before memory artifacts were produced`
- build stderr:
  `allocation lowering validation: p25.region_island_allocation.main instruction 14 explicit island allocation "xs" use after free via operands of island:p25.region_island_allocation.main:10`

Interpretation: this is not a measured heap allocation problem yet. MEM-5 must
reproduce and root-cause the island lowering validation failure before claiming
region/island memory evidence.

## Actor / Parallel State

`actor_ping_pong_tetra`:

- status: `measured`
- classification: `blocked by actor/runtime limitation`
- backend path: `fallback`
- backend blocker: `unsupported_effect_runtime_call`
- perf blocker: `actor_copy.borrowed_data_boundary`
- heap allocations: 0
- runtime heap sidecar reports 0 current, peak, total alloc bytes, and
  allocation count 0
- RSS peak: 14086144 bytes
- domain bytes evidence: `unsupported`
- unsupported reason: `allocation report summary does not include memory domains`

`parallel_map_reduce_tetra`:

- status: `measured`
- classification: `blocked by actor/runtime limitation`
- backend path: `fallback`
- backend blocker: `unsupported_call_abi`
- perf blockers: `actor_copy.borrowed_data_boundary`,
  `register_spill.live_range_pressure`
- heap allocations: 0
- runtime heap sidecar reports 0 current, peak, total alloc bytes, and
  allocation count 0
- RSS peak: 14086144 bytes
- domain bytes evidence: `unsupported`
- unsupported reason: `allocation report summary does not include memory domains`

Interpretation: actor rows are not heap-blocked in this baseline, but they are
not actor-memory-domain-ready. MEM-9 must add runtime-backed actor domain bytes
or keep the actor-domain claim blocked.

## MEM-0 Decision

MEM-0 is locked from current evidence.

Current next optimization target remains MEM-1:

```text
read_only_call(xs: make_u8(4))
NoEscape + read-only local call summary
Heap -> Stack or Region
```

Do not start with allocator internals. The immediate memory win is removing
unnecessary heap placement when compiler proof is strong enough.
