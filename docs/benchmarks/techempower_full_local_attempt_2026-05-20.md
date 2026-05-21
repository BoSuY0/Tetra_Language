# TechEmpower Full Local Attempt - 2026-05-20

Status: blocked by local Docker daemon availability.

Command attempted from the repository root:

```sh
TETRA_TE_BENCH_REPORT=reports/techempower/tetra-full-local-benchmark.json \
TETRA_TE_BENCH_REQUESTS=8 \
TETRA_TE_BENCH_CONCURRENCY=2 \
TETRA_TE_BENCH_MIN_RPS=1 \
benchmarks/techempower/tetra/run-full-local.sh
```

Observed failure:

```text
unable to get image 'postgres:16-alpine': failed to connect to the docker API at unix:///var/run/docker.sock; check if the path is correct and if the daemon is running: dial unix /var/run/docker.sock: connect: no such file or directory
```

Follow-up evidence from `docker info`:

```text
Server:
failed to connect to the docker API at unix:///var/run/docker.sock; check if the path is correct and if the daemon is running: dial unix /var/run/docker.sock: connect: no such file or directory
```

Conclusion: Docker CLI and Compose are installed, and `docker compose config`
validates the benchmark stack, but the Docker daemon is not reachable in this
environment. Full six-endpoint DB-backed benchmark evidence is therefore not
claimed by this artifact. The durable no-DB smoke evidence remains
`docs/benchmarks/techempower_local_smoke_skip_db_report.json`.
