#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)
COMPOSE_FILE="$ROOT_DIR/benchmarks/techempower/tetra/docker-compose.yml"

: "${TETRA_TE_BENCH_BASE_URL:=http://127.0.0.1:8080}"
: "${TETRA_TE_BENCH_REPORT:=reports/techempower/tetra-full-local-benchmark.json}"
: "${TETRA_TE_BENCH_REQUESTS:=256}"
: "${TETRA_TE_BENCH_CONCURRENCY:=32}"
: "${TETRA_TE_BENCH_MIN_RPS:=1}"

cd "$ROOT_DIR"

if ! docker info >/dev/null 2>&1; then
  echo "Docker daemon is not reachable; start Docker and rerun this script." >&2
  exit 1
fi

docker compose -f "$COMPOSE_FILE" up -d --build tfb-database tetra-techempower

cleanup() {
  if [ "${TETRA_TE_KEEP_COMPOSE:-false}" != "true" ]; then
    docker compose -f "$COMPOSE_FILE" down
  fi
}
trap cleanup EXIT INT TERM

for _ in $(seq 1 60); do
  if curl -fsS "$TETRA_TE_BENCH_BASE_URL/plaintext" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

TETRA_TE_BENCH_SKIP_DB=false \
TETRA_TE_BENCH_BASE_URL="$TETRA_TE_BENCH_BASE_URL" \
TETRA_TE_BENCH_REPORT="$TETRA_TE_BENCH_REPORT" \
TETRA_TE_BENCH_REQUESTS="$TETRA_TE_BENCH_REQUESTS" \
TETRA_TE_BENCH_CONCURRENCY="$TETRA_TE_BENCH_CONCURRENCY" \
TETRA_TE_BENCH_MIN_RPS="$TETRA_TE_BENCH_MIN_RPS" \
benchmarks/techempower/tetra/run-bench.sh
