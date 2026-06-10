# RAM Contract Report Schema

Status: current scoped schema contract.

## Schemas

- `tetra.ram-contract-report.v1` is written as `ram-contract-report.json`.
- `tetra.memory-grade-report.v1` is written as `memory-grade-report.json`.
- `tetra.proof-store-summary.v1` is written as `proof-store-summary.json`.
- `tetra.validation-pipeline-coverage.v1` is written as `validation-pipeline-coverage.json`.
- `tetra.ram-blockers.v1` is written as `heap-blockers.json` and `copy-blockers.json`.
- `ram-contract-fuzz-oracle.json` is the deterministic RAM contract fuzz oracle artifact.

## Validators

- `go run ./tools/cmd/validate-ram-contract-report --report ram-contract-report.json`
- `go run ./tools/cmd/validate-memory-grade-report --report memory-grade-report.json`
- `go run ./tools/cmd/validate-proof-store-summary --report proof-store-summary.json`
- `go run ./tools/cmd/validate-validation-pipeline-coverage --report validation-pipeline-coverage.json`
- `go run ./tools/cmd/validate-heap-blockers --report heap-blockers.json`
- `go run ./tools/cmd/validate-copy-blockers --report copy-blockers.json`
- `go run ./tools/cmd/validate-ram-contract-fuzz-oracle --report ram-contract-fuzz-oracle.json --artifact-dir .`

## Required Fields

`tetra.ram-contract-report.v1` records `entrypoint`, `target`, `git_head`, `rows`, `blockers`, and `summary`. Rows carry storage class, heap/copy byte counts, boundedness, proof references, and blocker IDs. Blocker IDs must resolve to concrete blocker rows; fake or missing blocker explanations are invalid.

`tetra.proof-store-summary.v1` records proof IDs and invalid references. A stale proof hash or a proof that promotes `unsafe_unknown` as trusted evidence is invalid.

## Nonclaims

The schema records report evidence. It makes no zero heap for all programs claim, no zero-copy for all programs claim, no full formal proof claim, and no all-target RAM parity claim.
