# Local Benchmark Tier 1 V1 Audit

Status: P25.0 local benchmark evidence artifact.

This audit records a local-only execution of the P20 matrix. It does not claim Tetra is the fastest language, does not claim an official benchmark result, does not claim cross-machine reproduction, does not claim TechEmpower publication, and does not claim production readiness.

Primary artifact: `reports/benchmark-vnext-memory-baseline/tier1-after-stack-slice-effect-summary/report.json`.

Summary artifact: `reports/benchmark-vnext-memory-baseline/tier1-after-stack-slice-effect-summary/summary.md`.

## Classifications

- `integer loops`: `comparable` — Tetra median 0.907 ms is within 20% of the fastest local competitor median 1.034 ms.
- `slice sum`: `blocked by bounds check` — Tetra bounds report records 1 bounds checks left.
- `bounds-check loops`: `blocked by bounds check` — Tetra bounds report records 2 bounds checks left.
- `function calls`: `faster than C/C++/Rust locally` — Tetra median 1.585 ms is more than 20% below the fastest local competitor median 2.086 ms.
- `recursion`: `faster than C/C++/Rust locally` — Tetra median 0.272 ms is more than 20% below the fastest local competitor median 0.615 ms.
- `matrix multiply`: `blocked by bounds check` — Tetra bounds report records 7 bounds checks left.
- `hash table`: `blocked by fallback backend` — Tetra backend report selected stack/fallback path for at least one function.
- `allocation`: `blocked by fallback backend` — Tetra backend report selected stack/fallback path for at least one function.
- `region/island allocation`: `blocked by fallback backend` — Tetra backend report selected stack/fallback path for at least one function.
- `JSON parse/stringify`: `invalid/inconclusive` — This Tier 1 run measures deterministic local helper kernels, not a full local service/database benchmark for this category.
- `HTTP plaintext/json`: `invalid/inconclusive` — This Tier 1 run measures deterministic local helper kernels, not a full local service/database benchmark for this category.
- `PostgreSQL single/multiple/update`: `invalid/inconclusive` — This Tier 1 run measures deterministic local helper kernels, not a full local service/database benchmark for this category.
- `actor ping-pong`: `blocked by actor/runtime limitation` — Current local actor/task runtime evidence is bounded and not a production parallel benchmark claim. Backend path is "fallback", not register. Backend blockers: unsupported_effect_runtime_call. Perf blockers: actor_copy.borrowed_data_boundary. Actor-domain memory evidence is missing or unsupported.
- `parallel map/reduce`: `blocked by actor/runtime limitation` — Current local actor/task runtime evidence is bounded and not a production parallel benchmark claim. Backend path is "fallback", not register. Backend blockers: unsupported_call_abi. Perf blockers: actor_copy.borrowed_data_boundary, register_spill.live_range_pressure. Actor-domain memory evidence is missing or unsupported.
- `startup time`: `slower` — Tetra median 1.004 ms is more than 20% above the fastest local competitor median 0.552 ms.
- `binary size`: `comparable` — binary_size_bytes local evidence: Tetra=6075, C=15832, C++=15840, Rust=445728; no binary-size superiority or production-size claim is promoted.
- `compile time`: `faster than C/C++/Rust locally` — Tetra compile_time_ms 15.416 is more than 20% below the fastest local competitor compile_time_ms 71.040.

## Required Verification

- `go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-stack-slice-effect-summary/report.json`
- `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
- `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
- `git diff --check`
- `graphify update .`
- `go test ./compiler/... ./cli/... ./tools/... -count=1`
