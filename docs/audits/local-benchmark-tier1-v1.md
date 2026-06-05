# Local Benchmark Tier 1 V1 Audit

Status: P25.0 local benchmark evidence artifact.

This audit records a local-only execution of the P20 matrix. It does not claim Tetra is the fastest language, does not claim an official benchmark result, does not claim cross-machine reproduction, does not claim TechEmpower publication, and does not claim production readiness.

Primary artifact: `reports/local-benchmark-tier1-v1/report.json`.

Summary artifact: `reports/local-benchmark-tier1-v1/summary.md`.

## Classifications

- `integer loops`: `blocked by fallback backend` — Tetra backend report selected stack/fallback path for at least one function.
- `slice sum`: `blocked by bounds check` — Tetra bounds report records 2 bounds checks left.
- `bounds-check loops`: `blocked by bounds check` — Tetra bounds report records 2 bounds checks left.
- `function calls`: `blocked by fallback backend` — Tetra backend report selected stack/fallback path for at least one function.
- `recursion`: `blocked by fallback backend` — Tetra backend report selected stack/fallback path for at least one function.
- `matrix multiply`: `blocked by bounds check` — Tetra bounds report records 7 bounds checks left.
- `hash table`: `blocked by heap allocation` — Tetra allocation report records 2 heap allocations.
- `allocation`: `blocked by fallback backend` — Tetra backend report selected stack/fallback path for at least one function.
- `region/island allocation`: `blocked by fallback backend` — Tetra backend report selected stack/fallback path for at least one function.
- `JSON parse/stringify`: `invalid/inconclusive` — This Tier 1 run measures deterministic local helper kernels, not a full local service/database benchmark for this category.
- `HTTP plaintext/json`: `invalid/inconclusive` — This Tier 1 run measures deterministic local helper kernels, not a full local service/database benchmark for this category.
- `PostgreSQL single/multiple/update`: `invalid/inconclusive` — This Tier 1 run measures deterministic local helper kernels, not a full local service/database benchmark for this category.
- `actor ping-pong`: `blocked by actor/runtime limitation` — Current local actor/task runtime evidence is bounded and not a production parallel benchmark claim.
- `parallel map/reduce`: `blocked by actor/runtime limitation` — Current local actor/task runtime evidence is bounded and not a production parallel benchmark claim.
- `startup time`: `faster than C/C++/Rust locally` — Tetra median 0.336 ms is more than 20% below the fastest local competitor median 0.816 ms.
- `binary size`: `comparable` — binary_size_bytes local evidence: Tetra=4096, C=15832, C++=15840, Rust=445728; no binary-size superiority or production-size claim is promoted.
- `compile time`: `faster than C/C++/Rust locally` — Tetra compile_time_ms 7.363 is more than 20% below the fastest local competitor compile_time_ms 60.828.

## Required Verification

- `go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/local-benchmark-tier1-v1/report.json`
- `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
- `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
- `git diff --check`
- `graphify update .`
- `go test ./compiler/... ./cli/... ./tools/... -count=1`
