#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)

: "${TETRA_TE_HOST:=127.0.0.1}"
: "${TETRA_TE_PORT:=8080}"
: "${TETRA_TE_WORKERS:=}"
: "${TETRA_TE_PG_HOST:=127.0.0.1}"
: "${TETRA_TE_PG_PORT:=5432}"
: "${TETRA_TE_PG_USER:=benchmarkdbuser}"
: "${TETRA_TE_PG_DATABASE:=hello_world}"
: "${TETRA_TE_PG_PASSWORD:=}"
: "${TETRA_TE_PG_POOL:=256}"

export TETRA_TE_HOST
export TETRA_TE_PORT
export TETRA_TE_WORKERS
export TETRA_TE_PG_HOST
export TETRA_TE_PG_PORT
export TETRA_TE_PG_USER
export TETRA_TE_PG_DATABASE
export TETRA_TE_PG_PASSWORD
export TETRA_TE_PG_POOL

cd "$ROOT_DIR"
go run ./compiler/cmd/tetra-techempower
