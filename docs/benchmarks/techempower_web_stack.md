# TechEmpower-Compatible Web Stack

Status: local compatible runtime and benchmark app slice for Tetra v0.4.0.

This is a real TCP/HTTP/PostgreSQL path implemented inside the current Tetra
runtime codebase. It is not an official upstream TechEmpower submission yet.
The upstream FrameworkBenchmarks repository was archived on March 24, 2026, so
official publication mechanics may require a fork or successor process.

## Runtime Pieces

- `compiler/internal/netrt`: Linux TCP sockets and epoll-backed event polling.
- `compiler/internal/httprt`: HTTP/1.1 parser, static and parameterized
  router, middleware wrappers, request body limits, and response writer.
- `compiler/internal/jsonrt`: byte-buffer JSON serializers and generic
  deterministic JSON value parse/write helpers.
- `compiler/internal/htmlrt`: Fortunes HTML escaping and rendering.
- `compiler/internal/pgrt`: PostgreSQL wire protocol client, prepared statement
  execution, TCP dial path, connection pool, and pool stats.
- `lib/core/postgres.tetra`: executable Tetra-source PostgreSQL startup,
  Simple Query, Parse/Bind/Describe/Execute/Sync,
  RowDescription/DataRow/CommandComplete/ReadyForQuery, Terminate, and endian
  byte-buffer helpers used to grow the stdlib-facing DB surface.
- `compiler/internal/webrt`: HTTP server and TechEmpower endpoint handlers.
- `compiler/cmd/tetra-techempower`: runnable benchmark server executable.

The backend API surface is summarized in `docs/user/backend_web_platform.md`.

## Endpoints

- `/plaintext`
- `/json`
- `/db`
- `/queries?queries=N`
- `/updates?queries=N`
- `/fortunes`

## Local Packaging

The local benchmark packaging lives in `benchmarks/techempower/tetra/`:

- `benchmark_config.json`
- `Dockerfile`
- `docker-compose.yml`
- `run-local.sh`
- `run-bench.sh`
- `run-full-local.sh`
- `run-scram-local-bench.sh`
- `setup-postgres.sql`
- `README.md`

Run the app from the repository root:

```sh
benchmarks/techempower/tetra/run-local.sh
```

The app expects a PostgreSQL instance with the TechEmpower `hello_world` schema.
Use environment variables such as `TETRA_TE_WORKERS`, `TETRA_TE_PG_HOST`,
`TETRA_TE_PG_PORT`, `TETRA_TE_PG_USER`, `TETRA_TE_PG_DATABASE`,
`TETRA_TE_PG_PASSWORD`, and `TETRA_TE_PG_POOL` to tune per-core event-loop
workers and point at the database. `TETRA_TE_PG_PASSWORD` may be empty for
trust-auth local setups; when PostgreSQL requests password authentication,
`compiler/internal/pgrt` supports cleartext PasswordMessage and SCRAM-SHA-256
SASL startup flows.

For a self-contained local six-endpoint run:

```sh
benchmarks/techempower/tetra/run-full-local.sh
```

This starts PostgreSQL 16, initializes the `World` and `Fortune` tables from
`setup-postgres.sql`, starts the Tetra benchmark app, and runs the benchmark
harness without `--skip-db`.

Generate a local benchmark/stress report against a running server:

```sh
benchmarks/techempower/tetra/run-bench.sh
```

The harness writes `tetra.techempower.benchmark.v1` JSON with one row per
endpoint, correctness validation, observed content types, semantic check lists,
concurrent request counts, latency summaries including p99.9, and threshold
decisions. The default artifact path is
`reports/techempower/tetra-local-benchmark.json`.
The validator rejects non-monotonic latency percentile evidence, so endpoint and
matrix-run reports must keep `p50 <= p90 <= p95 <= p99 <= p99.9 <= max`.
Matrix run evidence must also carry a positive `repeat` number; warmup evidence
uses `repeat=0` and is not counted as a matrix run. The validator rejects
warmup evidence with any other repeat value. Matrix run identities must be
unique by endpoint, worker count, concurrency/connection level, and repeat.
For repeatability, every declared matrix grid cell must carry the same
contiguous repeat sequence starting at `1`. Matrix run RPS must match
`successes / elapsed_seconds` within validator tolerance, and
`elapsed_seconds` must not be shorter than `duration_seconds`.
Soak evidence carries tail latency only and must keep
`p99 <= p99.9 <= max`. Soak RPS must match
`successes / duration_seconds` within validator tolerance.
Soak reports also require positive duration and concurrency/connection levels
with non-negative timing metrics, internally consistent request counters, and
zero failures.
Matrix run and soak endpoint identities must match the SCRAM matrix harness
allowlist: `/db`, `/queries?queries=2`, `/updates?queries=2`, or `/fortunes`
with their expected endpoint names and run kinds.
Matrix artifacts must include non-empty `semantic_report`, `matrix_report`,
`endpoints`, `levels`, and `worker_levels` entries so the reported benchmark
can be traced back to its generated evidence shape. The validator parses the
declared endpoint, worker, and concurrency/connection levels and rejects matrix
reports whose `runs` omit or exceed that declared grid. Matrix report `command`
provenance must include `scram-local-bench`, and the command's
`--semantic-report` and `--matrix-report` paths must match the recorded
artifact paths. The command's `--endpoints`, `--levels`, and
`--worker-levels` grid flags must also match the recorded artifact grid.
Matrix resource snapshots require RFC3339 timestamps, live positive-PID process
evidence, and non-negative TCP, CPU, and goroutine counters. Matrix start/end
resource spans must have increasing timestamps and non-regressing CPU counters;
per-run, warmup, and soak resource timestamps must fall within the report
resource window.

Validate a checked report before treating it as release evidence:

```sh
go run ./tools/cmd/validate-techempower-report \
  --report reports/techempower/tetra-local-benchmark.json
```

Only local no-database smoke reports may use the explicit allowance:

```sh
go run ./tools/cmd/validate-techempower-report \
  --report docs/benchmarks/techempower_local_smoke_skip_db_report.json \
  --allow-skip-db
```

Generate the reproducible local SCRAM/PostgreSQL evidence without Docker:

```sh
benchmarks/techempower/tetra/run-scram-local-bench.sh \
  --duration 60s \
  --warmup 10s \
  --soak 120s \
  --repeats 1 \
  --levels 8:8,16:16 \
  --worker-levels 1,2 \
  --endpoints db \
  --semantic-requests 64 \
  --semantic-concurrency 8 \
  --workers 1 \
  --pool 64 \
  --semantic-report docs/benchmarks/techempower_scram_single_query_local_report.json \
  --matrix-report docs/benchmarks/techempower_scram_single_query_matrix_local_report.json
```

For endpoint coverage beyond `/db`, use `--endpoints queries,updates,fortunes`
with a separate matrix report.

Current local evidence:

- generated smoke artifact:
  `reports/techempower/tetra-db-unavailable-benchmark.json`
- durable checked-in copy: `docs/benchmarks/techempower_local_smoke_skip_db_report.json`
- DB-backed SCRAM-SHA-256 local run:
  `docs/benchmarks/techempower_scram_single_query_local_report.json`
- DB-backed SCRAM-SHA-256 `/db` matrix:
  `docs/benchmarks/techempower_scram_single_query_matrix_local_report.json`
- DB-backed SCRAM-SHA-256 `/queries`, `/updates`, `/fortunes` matrix:
  `docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json`
- DB-backed SCRAM-SHA-256 run notes:
  `docs/benchmarks/techempower_scram_single_query_local_2026-05-21.md`
- full DB-backed local attempt log:
  `docs/benchmarks/techempower_full_local_attempt_2026-05-20.md`

The no-database smoke artifact intentionally uses `--skip-db`, so it covers only
`/plaintext` and `/json`. The SCRAM local run used PostgreSQL 16.9.0 with
`scram-sha-256` host authentication, `password_encryption=scram-sha-256`, and a
role verifier prefix of `SCRAM-SHA-256`. It passed all six endpoints and the
matrix semantic probe verified real DB reads, `queries` clamping, update
persistence, Fortune insertion, HTML escaping, and sorted Fortune rendering.
The 60s `/db` Single Query matrix completed 7294387 total requests, 0 failures,
best run 44435.284989964974 rps, worst p99 2.309208 ms, and worst p99.9
4.76986 ms across 1/2 worker levels and 8/16 concurrency/connection levels. A
120s `/db` soak completed 2194528 requests with 0 failures, RSS 14256 KB to
14244 KB, 0 open sockets after shutdown, and clean shutdown evidence. The 60s
endpoint matrix covered `/queries?queries=2`, `/updates?queries=2`, and
`/fortunes` at 2 workers / 8 connections with 0 failures. Full release evidence
should still add an external baseline and larger official-style hardware before
making competitive performance claims.

Backend hardening beyond the original benchmark slice now includes:

- HTTP request body limits with `413 Payload Too Large`;
- explicit unsupported-transfer diagnostics for chunked request bodies;
- route parameters via `PathValue`;
- router middleware extension points;
- deterministic generic JSON object/array/string/number/bool/null parse/write;
- PostgreSQL pool stats for leak checks.

Docker Compose status in this environment:

```sh
docker compose -f benchmarks/techempower/tetra/docker-compose.yml config
docker compose -f benchmarks/techempower/tetra/docker-compose.yml --profile benchmark config
```

Both static config validations passed. The actual Compose stack was not run
because `docker info` could not connect to `/var/run/docker.sock`.

## Verification

Relevant checks:

```sh
go test ./compiler/internal/netrt ./compiler/internal/httprt ./compiler/internal/jsonrt ./compiler/internal/htmlrt ./compiler/internal/webrt ./compiler/internal/pgrt -count=1
go test ./compiler/cmd/tetra-techempower -count=1
go test ./compiler/cmd/tetra-techempower-bench -count=1
go test ./tools/validators/techempower ./tools/cmd/validate-techempower-report -count=1
go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_local_smoke_skip_db_report.json --allow-skip-db
go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_local_report.json
go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_matrix_local_report.json
go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json
go run ./tools/cmd/validate-techempower-report --report reports/techempower/tetra-scram-endpoints-semantic-benchmark.json
GOWORK=off go test ./benchmarks/techempower/tetra/cmd/scram-local-bench -count=1
docker compose -f benchmarks/techempower/tetra/docker-compose.yml config
docker compose -f benchmarks/techempower/tetra/docker-compose.yml --profile benchmark config
go test ./compiler/internal/pgrt ./compiler/internal/webrt -run 'Prepared|DBEndpoint|QueriesEndpoint|UpdatesEndpoint|FortunesEndpoint' -count=1
go test ./compiler/internal/webrt -run 'Stress|SlowHeader' -count=1
go build -o /tmp/tetra-techempower ./compiler/cmd/tetra-techempower
go build -o /tmp/tetra-techempower-bench ./compiler/cmd/tetra-techempower-bench
```

Short fuzz smoke checks:

```sh
go test ./compiler/internal/httprt -run '^$' -fuzz=FuzzHTTPParseRequest -fuzztime=1s
go test ./compiler/internal/jsonrt -run '^$' -fuzz=FuzzAppendStringProducesValidJSON -fuzztime=1s
go test ./compiler/internal/htmlrt -run '^$' -fuzz=FuzzAppendEscapedRemovesRawHTMLSpecials -fuzztime=1s
go test ./compiler/internal/pgrt -run '^$' -fuzz=FuzzReadFrameDoesNotPanic -fuzztime=1s
```

## PostgreSQL Authentication

The DB-backed TechEmpower path now covers PostgreSQL cleartext password auth and
SCRAM-SHA-256. The SCRAM client parses `AuthenticationSASL`,
`AuthenticationSASLContinue`, and `AuthenticationSASLFinal`, generates a secure
nonce, computes the client proof with PBKDF2-HMAC-SHA-256, verifies the server
signature before accepting startup completion, and rejects malformed messages,
nonce mismatches, missing server-final messages, and bad signatures.

Known unsupported cases are explicit rather than silent fallbacks:
SASL mechanisms other than `SCRAM-SHA-256`, SCRAM-SHA-256-PLUS channel binding,
and SASLprep normalization for non-ASCII user names or passwords.
