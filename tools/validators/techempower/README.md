# TechEmpower Report Validator

`ValidateReport` checks `tetra.techempower.benchmark.v1` JSON reports emitted by
`compiler/cmd/tetra-techempower-bench`. It rejects weak placeholder evidence,
missing integrity metadata, incomplete endpoint sets, inconsistent counters, and
reports that omit latency percentiles (`p50`, `p90`, `p95`, `p99`, `p99.9`,
`max`), observed content types, or endpoint semantic check lists.

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
```

The matrix reports are validated by the SCRAM runner itself and include SCRAM
evidence, semantic probes, p99.9 latency, resource snapshots, optional soak
evidence, and shutdown cleanup checks.
