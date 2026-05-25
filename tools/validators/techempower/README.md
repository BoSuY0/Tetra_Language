# TechEmpower Report Validator

`ValidateReport` checks both `tetra.techempower.benchmark.v1` semantic reports
emitted by `compiler/cmd/tetra-techempower-bench` and
`tetra.techempower.single_query_matrix.v1` matrix reports emitted by the SCRAM
runner. It rejects weak placeholder evidence, missing integrity metadata,
incomplete endpoint sets, inconsistent counters, and reports that omit latency
percentiles (`p50`, `p90`, `p95`, `p99`, `p99.9`, `max`), positive and unique
matrix run repeat metadata with contiguous per-grid repeat coverage, zeroed
warmup repeat metadata, RPS evidence derived from successes over elapsed
seconds, soak RPS evidence derived from successes over duration seconds,
elapsed timing evidence that covers the declared run duration, observed
content types, endpoint semantic check lists, SCRAM evidence, matrix semantic
probes, timestamped resource snapshots inside the report resource window,
declared matrix grid coverage, matrix command provenance, command/artifact
report-path consistency, command/artifact grid consistency, command/run
duration consistency, command/run repeat consistency, command/warmup
consistency, command/soak consistency, or clean shutdown evidence when a soak is
present.

Use the CLI release gate from the repository root:

```sh
go run ./tools/cmd/validate-techempower-report \
  --report reports/techempower/tetra-local-benchmark.json
```

Reports generated with `--skip-db` are smoke evidence only and must opt in
explicitly:

```sh
go run ./tools/cmd/validate-techempower-report \
  --report docs/benchmarks/techempower_local_smoke_skip_db_report.json \
  --allow-skip-db
```

The reproducible local SCRAM runner writes a validator-compatible six-endpoint
report plus separate matrix reports:

```sh
benchmarks/techempower/tetra/run-scram-local-bench.sh
go run ./tools/cmd/validate-techempower-report \
  --report docs/benchmarks/techempower_scram_single_query_local_report.json
go run ./tools/cmd/validate-techempower-report \
  --report docs/benchmarks/techempower_scram_single_query_matrix_local_report.json
```

Matrix reports are also validated by the SCRAM runner before it exits; the CLI
validator gives release gates and stabilization evidence a standalone check for
SCRAM evidence, semantic probes, p99.9 latency, resource snapshots, optional
soak evidence, and shutdown cleanup checks.
