# MPC15-S2 Result: memory-report-invariant-audit

Status: integrated
Agent: Fermat (`019e90e4-e779-7683-aea4-e574d0c37490`)
Scope: read-only audit of memory production reports, invariant validation, and docs/manifest gates for MPC-15 fuzz oracle integration.

## Accepted Findings

- Memory reports already validate through compiler-owned `MemoryFactGraph` paths, so MPC-15 should reference those checks instead of reconstructing truth from report text; integrated the oracle invariant witness in `compiler/memory_fuzz_oracle_v1.go:426` through `compiler/memory_fuzz_oracle_v1.go:437`.
- The fuzz oracle needed invariant rows for no safe metadata mutation, no borrowed escape, no `unsafe_unknown -> safe_known`, no removed bounds check without proof id, no stack/region storage when escape exists, report validation against MemoryFactGraph, and MPC-14 cost model preservation; integrated in `compiler/memory_fuzz_oracle_v1.go:217` through `compiler/memory_fuzz_oracle_v1.go:229`.
- Tier 1 artifacts should be report-only and must not alter normal program semantics; integrated as a standalone command that builds and validates an oracle report in `tools/cmd/memory-fuzz-short/main.go:30`.
- Docs and manifest gates needed a dedicated `docs/audits/memory-fuzz-oracle-v1.md`; integrated through `tools/cmd/verify-docs/main.go:206`, `tools/cmd/verify-docs/main.go:226`, `compiler/features.go:249`, and `tools/cmd/validate-manifest/main.go:671`.
- The generated artifact now exposes oracle categories and invariants under `reports/memory-fuzz-short/mpc15/memory-fuzz-oracle.json`, with a mirrored summary in `reports/memory-fuzz-short/mpc15/summary.md`.

## RED Tests Integrated

- Report drift checks reject missing oracle categories, missing invariant coverage, wrong expected results, unsafe boundary inflation, and missing metadata-mutation wording in `compiler/memory_fuzz_oracle_v1_test.go:96`.
- Docs/manifest gates require the memory fuzz oracle audit in `tools/cmd/verify-docs/main_test.go`, `tools/cmd/validate-manifest/main_test.go`, and `compiler/tests/semantics/features_test.go`.
- The short Tier 1 command validates its own output through the compiler-owned oracle validator in `tools/cmd/memory-fuzz-short/main_test.go:13`.
- The standalone validator rejects invalid/mutated oracle reports in `tools/cmd/validate-memory-fuzz-oracle/main_test.go:31`.

## Rejected / Non-Issues

- `tools/validators/memoryprod` was not expanded into a fuzz oracle validator; it remains responsible for memory production smoke reports, while `validate-memory-fuzz-oracle` owns MPC-15 oracle schema checks.
- No linux-x64 runtime smoke was required inside every oracle unit test; runtime trap categories are oracle classifications, not a demand for long runtime fuzz in short CI.
- No report text is trusted as source of truth; the oracle report is valid only if it preserves compiler-owned invariants and MemoryFactGraph validation evidence.

## Verification Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc15-fuzz go test -p=1 ./compiler ./compiler/internal/memoryfacts ./tools/cmd/memory-production-smoke ./tools/validators/memoryprod ./tools/cmd/memory-fuzz-short ./tools/cmd/validate-memory-fuzz-oracle -run 'Fuzz|Oracle|Memory|Report|Invariant|Property|Stress' -count=1` passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc15-docs go test -p=1 ./tools/cmd/verify-docs ./tools/cmd/validate-manifest ./compiler/tests/semantics ./compiler -run 'MemoryFuzz|FuzzOracle|MemoryProductionContractDocs|FeatureRegistry|Manifest|Docs' -count=1` passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc15-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc15-docs go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json` passed.
