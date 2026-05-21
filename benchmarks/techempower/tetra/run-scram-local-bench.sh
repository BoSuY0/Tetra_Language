#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)

cd "$ROOT_DIR"
GOWORK=off go run ./benchmarks/techempower/tetra/cmd/scram-local-bench "$@"
