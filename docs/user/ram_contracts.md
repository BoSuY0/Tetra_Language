# Using RAM Contracts

RAM contracts are opt-in compiler reports for memory allocation and copy evidence. They are useful
when a release gate, library boundary, or application profile wants an explicit RAM budget.

## Build Flags

- `--emit-ram-contract-report` writes the RAM contract artifact bundle beside the output.
- `--fail-if-heap` fails the build when the report contains heap blockers.
- `--fail-if-copy` fails the build when the report contains copy blockers.
- `--fail-if-unbounded` fails the build when the report contains unbounded rows.
- `--memory-budget <bytes>` fails the build when reported RAM budget bytes exceed the budget.
- `--ram-contract <path>` supplies a RAM contract policy file for the build.

Enforcement failures use `TETRA4100`.

## Validation

Use the release validator when checking a complete bundle:

```sh
go run ./tools/cmd/validate-ram-contract-release --report-dir reports/ram-contract-release
```

Use the individual validators for focused iteration:

```sh
go run ./tools/cmd/validate-ram-contract-report --report reports/ram-contract-release/ram-contract-report.json
go run ./tools/cmd/validate-memory-grade-report --report reports/ram-contract-release/memory-grade-report.json
go run ./tools/cmd/validate-proof-store-summary --report reports/ram-contract-release/proof-store-summary.json
go run ./tools/cmd/validate-validation-pipeline-coverage --report reports/ram-contract-release/validation-pipeline-coverage.json
go run ./tools/cmd/validate-heap-blockers --report reports/ram-contract-release/heap-blockers.json
go run ./tools/cmd/validate-copy-blockers --report reports/ram-contract-release/copy-blockers.json
go run ./tools/cmd/validate-ram-contract-fuzz-oracle --report reports/ram-contract-release/fuzz/ram-contract-fuzz-oracle.json --artifact-dir reports/ram-contract-release
```

## Nonclaims

RAM contracts are evidence and enforcement controls, not a global optimizer promise: no zero heap
for all programs claim, no zero-copy for all programs claim, no all-target RAM parity claim, and no
performance claim.
