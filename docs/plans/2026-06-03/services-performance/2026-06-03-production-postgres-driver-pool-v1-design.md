# P19.3 Production PostgreSQL Driver / Pool V1 Foundation Design

Status: approved local slice design for the active Ideal Master Plan goal loop.

## Scope

This slice records bounded P19.3 evidence for the existing PostgreSQL runtime
foundation. It does not rewrite the driver or claim measured performance.

Covered evidence:

- startup and SCRAM-SHA-256 authentication through `compiler/internal/pgrt`
- prepared statements and extended query protocol
- binary int4 Bind/decode helpers
- connection pooling, pool cap, and backpressure rejection
- borrowed DataRow/region-view decode evidence
- local TechEmpower `/db`, `/queries`, `/updates`, and `/fortunes` handlers
- a Tetra-source-first DB benchmark gate in `truth-bench-harness`

## Non-Claims

- no official TechEmpower result
- no measured PostgreSQL throughput or P20 performance matrix
- no C++/Rust parity
- no external production database deployment claim
- no runtime behavior change
- no complete source-level `lib.core.postgres` driver API promotion

## Implementation Shape

Add a `compiler/internal/pgrt` coverage report with schema
`tetra.stdlib.postgresql.production_driver.v1`. Its validator must require one
row for every P19.3 feature and reject fake production, official benchmark,
P20, parity, and runtime-behavior claims.

Add `p19.3_postgres_source_first` to `tools/cmd/truth-bench-harness`. The
scope requires Tetra-only rows for `DB single query`, `DB multiple queries`,
`DB updates`, and `DB fortunes`, source build commands with `tetra build ...
--explain`, algorithm/input metadata, and Tetra proof/allocation/bounds/P19.3
coverage artifacts. Dry-run reports are gate evidence only.

Update docs, feature registry text, generated manifest evidence, and GOAL
progress after focused tests prove the slice.

## Verification Plan

Focused RED/GREEN:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/pgrt -run 'TestProductionPostgresCoverage' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./tools/cmd/truth-bench-harness -run 'TestP19PostgresSourceFirst' -count=1
```

Focused current-state gates:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/pgrt ./compiler/internal/webrt ./tools/cmd/truth-bench-harness ./compiler/tests/semantics ./tools/cmd/verify-docs ./tools/cmd/validate-manifest -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
git diff --check
graphify update .
```
