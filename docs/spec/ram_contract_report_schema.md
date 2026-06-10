# RAM Contract Report Schema

Status: current scoped schema contract.

## Schemas

- `tetra.ram-contract-report.v1` is written as `ram-contract-report.json`.
- `tetra.memory-grade-report.v1` is written as `memory-grade-report.json`.
- `tetra.proof-store-summary.v1` is written as `proof-store-summary.json`.
- `tetra.validation-pipeline-coverage.v1` is written as `validation-pipeline-coverage.json`.
- `tetra.ram-blockers.v1` is written as `heap-blockers.json` and `copy-blockers.json`.
- `fuzz/ram-contract-fuzz-oracle.json` is the deterministic RAM contract fuzz oracle artifact in release bundles.

## Validators

- `go run ./tools/cmd/validate-ram-contract-report --report ram-contract-report.json`
- `go run ./tools/cmd/validate-memory-grade-report --report memory-grade-report.json`
- `go run ./tools/cmd/validate-proof-store-summary --report proof-store-summary.json`
- `go run ./tools/cmd/validate-validation-pipeline-coverage --report validation-pipeline-coverage.json`
- `go run ./tools/cmd/validate-heap-blockers --report heap-blockers.json`
- `go run ./tools/cmd/validate-copy-blockers --report copy-blockers.json`
- `go run ./tools/cmd/validate-ram-contract-fuzz-oracle --report fuzz/ram-contract-fuzz-oracle.json --artifact-dir .`
- `go run ./tools/cmd/validate-ram-contract-release --report-dir reports/ram-contract-release --current-git-head "$(git rev-parse HEAD)"`

## Required Fields

`tetra.ram-contract-report.v1` records `schema_version`, `git_head`,
`target`, `generated_by`, optional `generated_at`, optional `functions`,
`rows`, optional `proofs`, `summary`, and `non_claims`.

Rows record `site_id`, `value_id`, `function`, optional `source_span`,
`intent`, `requested_bytes`, `bounded`, `owner`, `lifetime`, `escape_status`,
`placement`, `proof_ids`, `blockers`, optional `copy_reason`, optional
`free_point`, `contract_grade`, `validation_status`, and optional
`source_fact_id`. Heap placements require blocker explanations. Copy intents
require `copy_reason`. Trusted validated placements require usable proof IDs.

`summary` records `row_count`, `artifact_grade`, `heap_rows`, `copy_rows`,
`unbounded_rows`, and `budget_bytes`; it must match the rows exactly.

`tetra.memory-grade-report.v1` records `artifact_grade`, `functions`,
`summary`, and `non_claims`. In a release bundle, its top-level
`artifact_grade` and full `summary` must match `ram-contract-report.json`.

`tetra.proof-store-summary.v1` records proof IDs, kind, subject, stable hash,
status, and status counts. Duplicate proofs, missing proof fields, stale RAM
references, rejected references, unknown references, and a proof that promotes
`unsafe_unknown` as trusted evidence are invalid.

`tetra.validation-pipeline-coverage.v1` records release-profile entrypoints.
`validated_by_pipeline` entries require validators and `artifact_path`.
Non-exercised release entrypoints require specific
`formal_exemption_with_reason` entries.

`tetra.ram-blockers.v1` is emitted separately for heap and copy blockers. In a
release bundle, heap rows and copy rows are indexed by `site_id` in both
directions against `ram-contract-report.json`; extra blocker rows and missing
blocker rows are invalid.

## Nonclaims

The schema records report evidence. It makes no zero heap for all programs
claim, no zero-copy for all programs claim, no full formal proof claim, and no
all-target RAM parity claim.
