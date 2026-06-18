# Local Benchmark Tier 1 V1 Audit

Status: P25.0 local benchmark evidence artifact.

This audit records a local-only execution of the P20 matrix. It does not claim Tetra is the fastest
language, does not claim an official benchmark result, does not claim cross-machine reproduction,
does not claim TechEmpower publication, and does not claim production readiness.

Primary artifact:
`reports/benchmark-vnext-memory-baseline/tier1-after-hash-table-lookup-native/report.json`.

Summary artifact:
`reports/benchmark-vnext-memory-baseline/tier1-after-hash-table-lookup-native/summary.md`.

## Classifications

- `integer loops`: `comparable` — Tetra median 0.928 ms is within 20% of the fastest local
  competitor median 1.125 ms.
- `slice sum`: `blocked by fallback backend` — Tetra backend report selected stack/fallback path for
  at least one function.
- `bounds-check loops`: `comparable` — Tetra median 0.888 ms is within 20% of the fastest local
  competitor median 0.794 ms.
- `function calls`: `faster than C/C++/Rust locally` — Tetra median 1.548 ms is more than 20% below
  the fastest local competitor median 2.466 ms.
- `recursion`: `faster than C/C++/Rust locally` — Tetra median 0.247 ms is more than 20% below the
  fastest local competitor median 0.644 ms.
- `matrix multiply`: `blocked by fallback backend` — Tetra backend report selected stack/fallback
  path for at least one function.
- `hash table`: `blocked by fallback backend` — Tetra backend report selected stack/fallback path
  for at least one function.
- `allocation`: `faster than C/C++/Rust locally` — Tetra median 0.388 ms is more than 20% below the
  fastest local competitor median 0.701 ms.
- `region/island allocation`: `blocked by fallback backend` — Tetra backend report selected
  stack/fallback path for at least one function.
- `JSON parse/stringify`: `invalid/inconclusive` — This Tier 1 run measures deterministic local
  helper kernels, not a full local service/database benchmark for this category.
- `HTTP plaintext/json`: `invalid/inconclusive` — This Tier 1 run measures deterministic local
  helper kernels, not a full local service/database benchmark for this category.
- `PostgreSQL single/multiple/update`: `invalid/inconclusive` — This Tier 1 run measures
  deterministic local helper kernels, not a full local service/database benchmark for this category.
- `actor ping-pong`: `blocked by actor/runtime limitation` — Current local actor/task runtime
  evidence is bounded and not a production parallel benchmark claim. Backend path is "fallback", not
  register. Backend blockers: unsupported_effect_runtime_call. Perf blockers:
  actor_copy.borrowed_data_boundary.
- `parallel map/reduce`: `blocked by actor/runtime limitation` — Current local actor/task runtime
  evidence is bounded and not a production parallel benchmark claim. Backend path is "fallback", not
  register. Backend blockers: unsupported_call_abi. Perf blockers:
  actor_copy.borrowed_data_boundary, register_spill.live_range_pressure. Actor-domain memory
  evidence is missing or unsupported.
- `startup time`: `faster than C/C++/Rust locally` — Tetra median 0.252 ms is more than 20% below
  the fastest local competitor median 0.640 ms.
- `binary size`: `comparable` — binary_size_bytes local evidence: Tetra=10177, C=15832, C++=15840,
  Rust=445728; no binary-size superiority or production-size claim is promoted.
- `compile time`: `faster than C/C++/Rust locally` — Tetra compile_time_ms 10.961 is more than 20%
  below the fastest local competitor compile_time_ms 46.863.

## Required Verification

- `go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-hash-table-lookup-native/report.json`
- `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
- `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
- `git diff --check`
- `graphify update .`
- `go test ./compiler/... ./cli/... ./tools/... -count=1`
