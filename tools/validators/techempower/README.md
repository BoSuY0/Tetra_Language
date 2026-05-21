# TechEmpower Report Validator

`ValidateReport` checks `tetra.techempower.benchmark.v1` JSON reports emitted by
`compiler/cmd/tetra-techempower-bench`. It rejects weak placeholder evidence,
missing integrity metadata, incomplete endpoint sets, inconsistent counters, and
reports that omit latency percentiles (`p50`, `p90`, `p95`, `p99`, `max`).

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
report plus a separate `/db` matrix:

```sh
benchmarks/techempower/tetra/run-scram-local-bench.sh
go run ./tools/cmd/validate-techempower-report \
  --report docs/benchmarks/techempower_scram_single_query_local_report.json
```
