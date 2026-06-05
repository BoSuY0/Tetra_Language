# Benchmark Matrix Hardening V1

Status: P20.0 bounded foundation evidence slice.

This audit records the checked benchmark-matrix contract for the master-plan
P20.0 gate. It hardens scope, row identity, equivalence metadata, compiler
baselines, raw-output artifacts, Tetra report artifacts, and target-CPU
consistency. It is not a measured speed comparison, not C/C++/Rust parity, not
an official benchmark result, not an official TechEmpower result, and maps only
to P20.2 Tier 0 local-smoke wording.

## Harness Scope

The checked scope is implemented in:

- `tools/cmd/truth-bench-harness/main.go`
- schema `tetra.truth.benchmark.v1`
- scope `p20.0_benchmark_matrix`

The scope requires four language rows for every category:

- Tetra through `tetra build ... --explain`
- C through `clang -O3`
- C++ through `clang++ -O3`
- Rust through `rustc -C opt-level=3`

The scope requires these master-plan categories:

- integer loops
- slice sum
- bounds-check loops
- function calls
- recursion
- matrix multiply
- hash table
- allocation
- region/island allocation
- JSON parse/stringify
- HTTP plaintext/json
- PostgreSQL single/multiple/update
- actor ping-pong
- parallel map/reduce
- startup time
- binary size
- compile time

## Evidence Contract

Every P20.0 row must record:

- benchmark name, category, and language
- compiler version
- exact build and run commands
- same `algorithm_id` and `input_description` for equivalent language rows
- binary artifact path and positive binary size
- raw output artifact path
- row target CPU matching report host target CPU

Every Tetra row must also record existing proof, allocation, bounds, and
performance report artifacts. The harness validator rejects missing matrix
rows, missing equivalence metadata, missing raw output artifacts, missing Tetra
report artifacts, target-CPU drift, fake C++/Rust parity claims, broad
fastest-language claims, and fake official benchmark claims.

## Checked Artifact

The checked dry-run artifact is:

- manifest:
  `reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-hardening-manifest.json`
- report:
  `reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-hardening-report.json`
- artifact directory:
  `reports/benchmark-matrix-hardening-v1/benchmarks/artifacts/`

The report has:

- schema `tetra.truth.benchmark.v1`
- scope `p20.0_benchmark_matrix`
- 68 rows: 17 categories times 4 languages
- languages `tetra`, `c`, `cpp`, and `rust`
- raw output artifacts on every row
- Tetra performance report artifacts on all 17 Tetra rows
- `ran=false` for all rows

Because `ran=false`, this artifact proves the matrix and evidence contract. It
does not prove throughput, latency, startup speed, binary-size advantage,
compile-time advantage, C/C++/Rust parity, or production database performance.

The Tetra performance artifact path now points at the P20.1 blocker report:

- `reports/benchmark-matrix-hardening-v1/benchmarks/artifacts/p20-matrix-hardening.perf.json`

That report has schema version `3`, kind `perf`, the exact eight P20.1 blocker
reasons, and 17 P20.0 Tetra benchmark explanation rows. It explains blockers;
it does not run benchmarks or make measured performance claims.

## Boundaries

- P20.0 owns the complete benchmark matrix contract.
- P20.1 owns blocker closure for any still-missing benchmark execution or
  benchmark-environment prerequisites.
- P20.2 owns claim-tier promotion, external comparison policy, and any measured
  performance claims. Current P20.0 dry-run evidence validates only Tier 0
  local-smoke wording in `reports/claim-tiers-v1/claim-tier-report.json`.
- Existing P19 source-first and local SCRAM/PostgreSQL evidence remain bounded
  to their own scopes and do not become P20 performance claims.

## Verification

Focused evidence commands:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./tools/cmd/truth-bench-harness -run 'TestP20BenchmarkMatrix' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/truth-bench-harness --manifest reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-hardening-manifest.json --out reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-hardening-report.json
```

Relevant broader gates:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./tools/cmd/truth-bench-harness ./compiler/tests/semantics ./tools/cmd/verify-docs ./tools/cmd/validate-manifest -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
```
