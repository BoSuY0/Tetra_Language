# Tetra TechEmpower-Compatible Web Benchmark

This directory contains the local TechEmpower-compatible packaging for the
in-repo Tetra web runtime.

It is not an official upstream TechEmpower submission yet. The upstream
FrameworkBenchmarks repository was archived in March 2026, and official
submission/round mechanics may require a fork-specific review path.

## Endpoints

- `/plaintext`
- `/json`
- `/db`
- `/queries?queries=N`
- `/updates?queries=N`
- `/fortunes`

## Run Locally

Start a PostgreSQL database with the TechEmpower `hello_world` schema, then run:

```sh
benchmarks/techempower/tetra/run-local.sh
```

For a self-contained Docker Compose run with PostgreSQL, schema initialization,
the app, and a six-endpoint benchmark report:

```sh
benchmarks/techempower/tetra/run-full-local.sh
```

Useful overrides:

```sh
TETRA_TE_HOST=127.0.0.1 \
TETRA_TE_PORT=8080 \
TETRA_TE_WORKERS=4 \
TETRA_TE_PG_HOST=127.0.0.1 \
TETRA_TE_PG_PORT=5432 \
TETRA_TE_PG_USER=benchmarkdbuser \
TETRA_TE_PG_DATABASE=hello_world \
TETRA_TE_PG_PASSWORD=benchmarkdbpass \
TETRA_TE_PG_POOL=256 \
benchmarks/techempower/tetra/run-local.sh
```

`TETRA_TE_WORKERS` controls how many independent nonblocking event loops bind
the same port with `SO_REUSEPORT`. Leave it unset to use Go's current
`GOMAXPROCS` value.

`TETRA_TE_PG_PASSWORD` is optional for trust-auth local setups. When PostgreSQL
requests password authentication, the runtime supports cleartext
PasswordMessage and SCRAM-SHA-256 SASL startup flows. SCRAM uses a secure client
nonce, validates the server nonce extension, sends the client proof, and rejects
bad server signatures. The current runtime does not implement SASLprep for
non-ASCII credentials or SCRAM-SHA-256-PLUS channel binding.

Quick smoke probes:

```sh
curl -s http://127.0.0.1:8080/plaintext
curl -s http://127.0.0.1:8080/json
curl -s http://127.0.0.1:8080/fortunes
```

If `wrk` is installed:

```sh
wrk -t4 -c128 -d15s http://127.0.0.1:8080/plaintext
wrk -t4 -c128 -d15s http://127.0.0.1:8080/json
```

## Local Benchmark Report

With the server running, produce a machine-readable correctness and load report:

```sh
benchmarks/techempower/tetra/run-bench.sh
```

The report schema is `tetra.techempower.benchmark.v1` and defaults to:

```sh
reports/techempower/tetra-local-benchmark.json
```

Each endpoint row records the observed content type, semantic checks,
p50/p90/p95/p99/p99.9/max latency, request counters, and threshold decision.

Useful overrides:

```sh
TETRA_TE_BENCH_BASE_URL=http://127.0.0.1:8080 \
TETRA_TE_BENCH_REQUESTS=1000 \
TETRA_TE_BENCH_CONCURRENCY=64 \
TETRA_TE_BENCH_MIN_RPS=1 \
benchmarks/techempower/tetra/run-bench.sh
```

For a no-database smoke of only `/plaintext` and `/json`, set
`TETRA_TE_BENCH_SKIP_DB=true`. Full TechEmpower-compatible evidence should run
all six endpoints against PostgreSQL.

For a reproducible local SCRAM-backed run without Docker, use:

```sh
benchmarks/techempower/tetra/run-scram-local-bench.sh
```

This builds release binaries, starts embedded PostgreSQL 16.9 with
`scram-sha-256` host authentication, seeds `World` and `Fortune`, validates all
six endpoints semantically, and writes both a six-endpoint report and a `/db`
Single Query matrix. Use longer durations for release gates:

```sh
benchmarks/techempower/tetra/run-scram-local-bench.sh \
  --duration 60s \
  --warmup 10s \
  --soak 120s \
  --repeats 1 \
  --levels 8:8,16:16 \
  --worker-levels 1,2 \
  --endpoints db
```

The checked-in SCRAM-SHA-256 local evidence lives at
`docs/benchmarks/techempower_scram_single_query_local_report.json`, with run
notes in `docs/benchmarks/techempower_scram_single_query_local_2026-05-21.md`.
It used PostgreSQL 16.9 with `scram-sha-256` host authentication and includes a
passing Single Query `/db` matrix at
`docs/benchmarks/techempower_scram_single_query_matrix_local_report.json`. A
separate endpoint matrix for `/queries`, `/updates`, and `/fortunes` lives at
`docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json`. Validate
both semantic and matrix JSON reports with
`go run ./tools/cmd/validate-techempower-report --report <path>`.

For Linux `perf` profiling, pass `--profile-build` to
`run-scram-local-bench.sh`. This opt-in mode builds the app and benchmark helper
without `-trimpath` or `-ldflags=-s -w`, and adds `-gcflags=all=-N -l` so
profiles preserve Go symbols. Do not use profile-build RPS numbers as
competitive benchmark claims; use it only to collect profiling evidence, then
rerun release builds for performance comparisons.

For live server Go pprof evidence, pass `--pprof-dir <dir>`. The runner binds a
private loopback-only pprof endpoint for the benchmark server, records a CPU
profile during the first DB-backed matrix load, captures a heap profile
afterward, and writes artifact paths into the matrix report. Keep this opt-in
for profiling runs only; default release/benchmark runs do not expose pprof.

## Docker

Build the local image from the repository root:

```sh
docker build -f benchmarks/techempower/tetra/Dockerfile -t tetra-techempower .
```

Run it on a Docker network that can resolve the PostgreSQL host:

```sh
docker run --rm --network tfb -p 8080:8080 tetra-techempower
```

Or run the compose stack:

```sh
docker compose -f benchmarks/techempower/tetra/docker-compose.yml up --build
```

Run the Compose benchmark profile after the app is healthy:

```sh
docker compose -f benchmarks/techempower/tetra/docker-compose.yml --profile benchmark up --build --abort-on-container-exit
```

The Compose database uses `POSTGRES_INITDB_ARGS` with SCRAM host/local auth and
`password_encryption=scram-sha-256`; the app receives
`TETRA_TE_PG_PASSWORD=benchmarkdbpass`.

In the 2026-05-21 local environment, Compose static validation passed but the
stack could not be executed because `docker info` could not connect to the
Docker daemon at `/var/run/docker.sock`.
