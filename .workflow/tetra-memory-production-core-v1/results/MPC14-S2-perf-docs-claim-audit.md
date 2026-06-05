# MPC14-S2 Result: perf-docs-claim-audit

Status: integrated
Agent: Russell (`019e90ca-96cb-7793-9b84-2cdef13919e4`)
Scope: read-only audit of performance blocker reports, docs/manifest gates, and optimization claim wording.

## Accepted Findings

- Performance blocker rows lacked `cost_class`; integrated in `compiler/reports.go:240` and populated for P20 blockers in `compiler/reports.go:712`.
- `ValidatePerformanceBlockerReport` did not validate cost classes or exact memory-blocker mapping; integrated in `compiler/reports.go:830` through `compiler/reports.go:873`.
- Performance blocker claim validation did not reject fake zero-cost or trusted `unsafe_unknown` wording; integrated in `compiler/reports.go:935`.
- Memory production docs gates did not require `docs/design/memory_cost_model.md`; integrated in `tools/cmd/verify-docs/main.go:205`, `tools/cmd/verify-docs/main.go:348`, `tools/cmd/validate-manifest/main.go:645`, and `compiler/features.go:247`.
- The new cost model needed contract docs and feature-surface linkage; integrated in `docs/design/memory_cost_model.md`, `docs/spec/memory_report_schema_v1.md:57`, `docs/audits/performance-blocker-reports-v1.md:42`, and `docs/audits/memory-production-core-v1-supported-surface.md:119`.

## RED Tests Integrated

- Performance blockers require valid cost classes and exact mapping in `compiler/reports_internal_test.go:915`, `compiler/reports_internal_test.go:949`, and `compiler/reports_internal_test.go:1057`.
- Fake `dynamic_check_required` zero-cost wording and trusted `unsafe_unknown` wording are rejected in `compiler/reports_internal_test.go`.
- Docs/manifest gates require the cost model doc in `tools/cmd/verify-docs/main_test.go`, `tools/cmd/validate-manifest/main_test.go`, and `compiler/tests/semantics/features_test.go:84`.

## Rejected / Non-Issues

- `tools/cmd/validate-performance-report` validates the separate performance-regression schema, not `.perf.json` P20 blocker rows; no MPC-14 change was needed there.
- No benchmark measurement scope was added for MPC-14; this slice only classifies blocker cost and rejects unsafe claims.

## Verification Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc14-perf go test -p=1 ./tools/cmd/validate-performance-report ./tools/validators/memoryprod ./compiler/internal/runtimeabi -run 'Cost|Blocker|Memory|Report|Runtime|Validator' -count=1` passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc14-docs go test -p=1 ./tools/cmd/verify-docs ./tools/cmd/validate-manifest -run 'Cost|Memory|Manifest|Docs' -count=1` passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc14-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc14-docs-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json` passed.
