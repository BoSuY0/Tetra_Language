# MPC15-S1 Result: fuzz-oracle-tooling-audit

Status: integrated
Agent: Pascal (`019e90e4-e6c7-74e2-876b-b6e8433c3bcc`)
Scope: read-only audit of existing fuzz/property/differential tooling and the smallest MPC-15 oracle/report insertion points.

## Accepted Findings

- Existing P23.1 fuzz/property artifacts did not provide explicit memory oracle categories; integrated a memory-specific oracle report instead of overloading the generic fuzz summary in `compiler/memory_fuzz_oracle_v1.go:12` through `compiler/memory_fuzz_oracle_v1.go:25`.
- Oracle classification needed to distinguish expected checker rejects, expected runtime traps, reference-output equality, compiler crash bugs, miscompile bugs, `unsafe_unknown` promoted as safe bugs, and report validation failures; integrated in `compiler/memory_fuzz_oracle_v1.go:121` and covered by `compiler/memory_fuzz_oracle_v1_test.go:71`.
- Tier 1 needed to stay short and deterministic, while Tier 2 and Tier 3 remain declared fuzz scopes rather than long CI work; integrated in the oracle report and Tier 1 command at `compiler/memory_fuzz_oracle_v1.go:178` and `tools/cmd/memory-fuzz-short/main.go:30`.
- The existing generic `validate-fuzz-summary` path is intentionally separate; MPC-15 now has a memory-specific validator in `tools/cmd/validate-memory-fuzz-oracle/main.go:26`.
- Generator scope needed explicit boundaries for supported-now safe memory features, narrow support, conservative/rejected unsafe/target behavior, and future-only scope; integrated in `docs/audits/memory-fuzz-oracle-v1.md` and projected into `reports/memory-fuzz-short/mpc15/memory-fuzz-oracle.json`.

## RED Tests Integrated

- Oracle category and invariant coverage: `compiler/memory_fuzz_oracle_v1_test.go:8`.
- Per-observation classification for checker reject, runtime trap, reference equality, compiler crash, miscompile, unsafe promotion, and report validation failure: `compiler/memory_fuzz_oracle_v1_test.go:71`.
- Drift rejection for missing categories, missing invariants, wrong expected result, unsafe boundary inflation, and metadata mutation wording: `compiler/memory_fuzz_oracle_v1_test.go:96`.
- Tier 1 artifact emission and Tier 2 rejection in the short command: `tools/cmd/memory-fuzz-short/main_test.go:13`.
- CLI validation accepts compiler-owned oracle reports and rejects invalid JSON reports: `tools/cmd/validate-memory-fuzz-oracle/main_test.go:13`.

## Rejected / Non-Issues

- No long-running fuzz execution was added to Tier 1; Tier 1 is a deterministic oracle smoke artifact.
- No unsupported unsafe or cross-target behavior was promoted by generated program output; unsupported behavior remains conservative/rejected generator scope.
- The generic P23.1 fuzz summary remains separate because MPC-15 needs memory-specific invariants and report validation against MemoryFactGraph.

## Verification Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc15-red go test -p=1 ./compiler ./tools/cmd/memory-fuzz-short ./tools/cmd/validate-memory-fuzz-oracle ./tools/cmd/verify-docs ./compiler/tests/semantics -run 'MemoryFuzz|FuzzOracle|MemoryProductionContractDocs|FeatureRegistry' -count=1` failed before implementation on missing oracle API, short command, validator, and docs gates.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc15-red go test -p=1 ./compiler ./tools/cmd/memory-fuzz-short ./tools/cmd/validate-memory-fuzz-oracle ./tools/cmd/verify-docs ./compiler/tests/semantics -run 'MemoryFuzz|FuzzOracle|MemoryProductionContractDocs|FeatureRegistry' -count=1` passed after implementation.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc15-short go run ./tools/cmd/memory-fuzz-short --tier=1 --report-dir reports/memory-fuzz-short/mpc15` passed and produced `reports/memory-fuzz-short/mpc15/memory-fuzz-oracle.json` plus `reports/memory-fuzz-short/mpc15/summary.md`.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc15-short go run ./tools/cmd/validate-memory-fuzz-oracle --report reports/memory-fuzz-short/mpc15/memory-fuzz-oracle.json` passed.
