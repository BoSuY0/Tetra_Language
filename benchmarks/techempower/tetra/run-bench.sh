#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)

: "${TETRA_TE_BENCH_BASE_URL:=http://127.0.0.1:8080}"
: "${TETRA_TE_BENCH_REPORT:=reports/techempower/tetra-local-benchmark.json}"
: "${TETRA_TE_BENCH_REQUESTS:=256}"
: "${TETRA_TE_BENCH_CONCURRENCY:=32}"
: "${TETRA_TE_BENCH_MIN_RPS:=1}"
: "${TETRA_TE_BENCH_SKIP_DB:=false}"

cd "$ROOT_DIR"
mkdir -p "$(dirname "$TETRA_TE_BENCH_REPORT")"

args="
  --base-url $TETRA_TE_BENCH_BASE_URL
  --report $TETRA_TE_BENCH_REPORT
  --requests $TETRA_TE_BENCH_REQUESTS
  --concurrency $TETRA_TE_BENCH_CONCURRENCY
  --min-rps $TETRA_TE_BENCH_MIN_RPS
"

if [ "$TETRA_TE_BENCH_SKIP_DB" = "true" ]; then
  args="$args --skip-db"
fi

# shellcheck disable=SC2086
go run ./compiler/cmd/tetra-techempower-bench $args
