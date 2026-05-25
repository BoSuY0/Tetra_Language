# TechEmpower SCRAM Local Evidence - 2026-05-21

Status: pass for local DB-backed TechEmpower-compatible validation.

This is local harness evidence, not an official TechEmpower publication. The
runner builds the Tetra server and benchmark harness in release mode, starts a
real PostgreSQL 16.9 process from embedded PostgreSQL binaries, rewrites
`pg_hba.conf` to require `scram-sha-256`, seeds the TechEmpower `hello_world`
schema, validates all six endpoints semantically, and runs release benchmark
matrices.

Environment:

- local time: `2026-05-21T16:47:20+03:00` for the 60s `/db` matrix
- UTC time: `2026-05-21T13:47:20Z` for the 60s `/db` matrix
- host: `tetra-4`
- OS/kernel: Linux `7.0.5-2-cachyos` x86_64
- Go: `go1.26.3-X:nodwarf5 linux/amd64`
- git head: `be39ab2ab4e1`
- git worktree: dirty
- Tetra app: release binary built with `go build -trimpath -ldflags=-s -w`
- Benchmark harness: release binary built with `go build -trimpath -ldflags=-s -w`
- PostgreSQL: 16.9.0, local loopback, `password_encryption=scram-sha-256`
- User/database: `benchmarkdbuser` / `hello_world`
- Auth evidence: role password verifier prefix `SCRAM-SHA-256`
- Seed evidence: `World=10000`, `Fortune=12`
- Pool size: `TETRA_TE_PG_POOL=64`

Primary 60s `/db` command:

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

Endpoint matrix command:

```sh
benchmarks/techempower/tetra/run-scram-local-bench.sh \
  --duration 60s \
  --warmup 5s \
  --repeats 1 \
  --levels 8:8 \
  --worker-levels 2 \
  --endpoints queries,updates,fortunes \
  --semantic-requests 32 \
  --semantic-concurrency 4 \
  --workers 2 \
  --pool 64 \
  --semantic-report reports/techempower/tetra-scram-endpoints-semantic-benchmark.json \
  --matrix-report docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json
```

Durable reports:

```sh
docs/benchmarks/techempower_scram_single_query_local_report.json
docs/benchmarks/techempower_scram_single_query_matrix_local_report.json
docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json
```

Semantic report:

- generated at: `2026-05-21T16:47:20+03:00`
- endpoints: `/plaintext`, `/json`, `/db`, `/queries?queries=2`,
  `/updates?queries=2`, `/fortunes`
- requests: 384
- successes: 384
- failures: 0
- each endpoint records `observed_content_type` and `semantic_checks`

Semantic probe evidence in the matrix report:

- `/plaintext`: status, `text/plain` body, `Date`, and `Server` headers.
- `/json`: JSON object shape and content type.
- `/db`: response matched a real PostgreSQL `World` row.
- `/queries`: `queries` parameter clamps to `1..500`.
- `/updates`: response values persisted back into PostgreSQL.
- `/fortunes`: request-time Fortune insertion, HTML escaping, and sorted order.

Observed `/db` Single Query matrix:

| workers | concurrency | connections | repeat | requests | failures | rps | avg ms | p50 ms | p90 ms | p95 ms | p99 ms | p99.9 ms | max ms | RSS KB | FD |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | 8 | 8 | 1 | 1108444 | 0 | 18473.87478037671 | 0.4319324313470054 | 0.382641 | 0.677937 | 0.803452 | 1.124632 | 1.764247 | 16.875712 | 14092 | 132 |
| 1 | 16 | 16 | 1 | 1172117 | 0 | 19535.0510947854 | 0.8179399507736856 | 0.72013 | 1.220962 | 1.452921 | 2.309208 | 4.76986 | 25.375004 | 14256 | 148 |
| 2 | 8 | 8 | 1 | 2347648 | 0 | 39122.4630959888 | 0.20354224388749934 | 0.184287 | 0.340789 | 0.405096 | 0.591746 | 0.935977 | 13.708611 | 15192 | 20 |
| 2 | 16 | 16 | 1 | 2666178 | 0 | 44435.284989964974 | 0.35919239768125005 | 0.338973 | 0.537602 | 0.632137 | 0.843595 | 1.275039 | 5.602574 | 15008 | 36 |

Matrix summary: 7294387 total `/db` requests, 0 failures, best run
44435.284989964974 rps, worst p99 2.309208 ms, worst p99.9 4.76986 ms.

Soak evidence:

- endpoint: `/db`
- duration: 120s
- requests: 2194528
- failures: 0
- rps: 18287.698915731577
- avg latency: 0.43638156438878883 ms
- p99: 1.113969 ms
- p99.9: 1.622909 ms
- max: 4.45315 ms
- RSS start/end: 14256 KB / 14244 KB
- open sockets after shutdown: 0
- shutdown clean: true

Additional 60s endpoint matrix:

| endpoint | workers | concurrency | requests | failures | rps | avg ms | p99 ms | p99.9 ms | max ms |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `/queries?queries=2` | 2 | 8 | 1180000 | 0 | 19665.706007744328 | 0.40562214076694914 | 1.523879 | 5.657515 | 17.428599 |
| `/updates?queries=2` | 2 | 8 | 661365 | 0 | 11021.61024830387 | 0.7244700978672897 | 2.7451 | 8.281976 | 21.288297 |
| `/fortunes` | 2 | 8 | 1199025 | 0 | 19982.058293634254 | 0.39914159810929717 | 1.180501 | 3.338541 | 11.85496 |

Docker status:

- `docker info` failed with
  `dial unix /var/run/docker.sock: connect: no such file or directory`.
- `docker compose -f benchmarks/techempower/tetra/docker-compose.yml config`
  passed.
- `docker compose -f benchmarks/techempower/tetra/docker-compose.yml --profile benchmark config`
  passed.
- The Compose stack was not executed because the Docker daemon was not
  reachable in this environment.

Limitations:

- This is local harness evidence, not an official TechEmpower publication.
- Docker Compose packaging is statically valid but was not executed because the
  Docker daemon was not reachable.
- The endpoint matrix covers `/queries`, `/updates`, and `/fortunes` at one
  worker/concurrency level; expand it before making competitive claims.
- No mature external baseline was run in this environment.
- SCRAM-SHA-256-PLUS channel binding and SASLprep for non-ASCII credentials
  remain unsupported.
