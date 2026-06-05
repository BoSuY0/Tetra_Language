# Production PostgreSQL Driver / Pool V1

Status: P19.3 bounded closure evidence slice.

This audit records the current PostgreSQL driver/pool foundation, the
source-first DB benchmark gate, and the checked local SCRAM/PostgreSQL
benchmark honesty closure. It is not an official TechEmpower result, not a
production database benchmark, not a P20 performance matrix, not a C++/Rust
parity claim, not a measured speed-comparison claim, and not a public
source-level full PostgreSQL driver API promotion.

## Coverage

The machine-readable coverage API is:

- `compiler/internal/pgrt/production_postgres_coverage.go`
- schema `tetra.stdlib.postgresql.production_driver.v1`
- validator `ValidateProductionPostgresCoverage`

Rows covered by the validator:

- startup and SCRAM-SHA-256 authentication
- prepared statements and extended query protocol
- binary int4 Bind/decode helpers
- connection pooling and backpressure
- borrowed DataRow decode evidence
- local `/db`, `/queries`, `/updates`, and `/fortunes` endpoint workloads
- source-first PostgreSQL benchmark gate
- live local SCRAM benchmark honesty gate

## Benchmark Gate

The checked dry-run benchmark artifact is:

- manifest:
  `reports/production-postgres-v1/benchmarks/postgres-source-first-manifest.json`
- report:
  `reports/production-postgres-v1/benchmarks/postgres-source-first-report.json`
- scope: `p19.3_postgres_source_first`

The scope requires Tetra-only `DB single query`, `DB multiple queries`,
`DB updates`, and `DB fortunes` rows, source build commands with
`tetra build ... --explain`, algorithm/input metadata, and Tetra
proof/allocation/bounds/P19.3 coverage artifacts. The generated report has
`ran=false`, so it records source-first gate coverage, not measured speed.

## Live Local Benchmark Honesty

The checked local SCRAM/PostgreSQL evidence is:

- semantic report:
  `docs/benchmarks/techempower_scram_single_query_local_report.json`
- `/db` matrix report:
  `docs/benchmarks/techempower_scram_single_query_matrix_local_report.json`
- `/queries`, `/updates`, and `/fortunes` matrix report:
  `docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json`

All three are accepted by `tools/cmd/validate-techempower-report` without
`--allow-skip-db`. The semantic report covers all six local endpoints. The
matrix reports record SCRAM-SHA-256 PostgreSQL metadata, command provenance
through the SCRAM local benchmark harness, resource snapshots, zero-failure
matrix runs, and endpoint coverage for `/db`, `/queries`, `/updates`, and
`/fortunes`.

The validator also has negative tests for weak SCRAM metadata, spoofed command
provenance, missing or invalid git heads, artifact/grid mismatches, command
duration/repeat/warmup/soak/pool mismatches, invalid resource snapshots,
summary mismatches, weak placeholder evidence, and endpoint identity drift.

## Boundaries

- `compiler/internal/pgrt` supports local startup, cleartext password, and
  SCRAM-SHA-256 authentication evidence through fake wire-server tests.
- `PreparedQueryFormat` and `AppendBindFormat` prove extended-protocol
  prepared statements and binary int4 parameter helpers for the current
  TechEmpower runtime path.
- `Pool.Checkout` uses a capped local pool and returns `ErrPoolExhausted` for
  backpressure. It does not claim adaptive production queueing.
- `DecodeDataRowBorrowed` returns borrowed cell views tied to the frame payload
  lifetime. It does not claim external region allocator integration.
- `compiler/internal/webrt` covers local `/db`, `/queries`, `/updates`, and
  `/fortunes` correctness with fake PostgreSQL wire servers and existing local
  SCRAM benchmark artifacts.
- DB-backed SCRAM local benchmark reports are closure evidence for honest local
  runtime/PostgreSQL measurement only; they do not become official upstream
  TechEmpower results or production database benchmark claims.
- P20 owns broader benchmark matrices, external baselines, measured speed, and
  performance comparison claims.

## Verification

Focused evidence commands:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/pgrt -run 'TestProductionPostgresCoverage' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./tools/cmd/truth-bench-harness -run 'TestP19PostgresSourceFirst' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/truth-bench-harness --manifest reports/production-postgres-v1/benchmarks/postgres-source-first-manifest.json --out reports/production-postgres-v1/benchmarks/postgres-source-first-report.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_local_report.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_matrix_local_report.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json
```
