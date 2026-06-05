# MPC14-S1 Result: cost-report-validator-audit

Status: integrated
Agent: Anscombe (`019e90ca-9614-77c0-877b-9991ea1fe40d`)
Scope: read-only audit of memory report cost classes, compiler-owned fact projection, and validator gaps.

## Accepted Findings

- Memory facts and report rows lacked `cost_class` and `normal_build_check`; integrated with `CostClass` and `NormalBuildCheck` on `Fact` and `ReportRow` in `compiler/internal/memoryfacts/facts.go:115` and `compiler/internal/memoryfacts/report.go:44`.
- Cost class validation belonged in the graph and report validators, not report-only reconstruction; integrated in `compiler/internal/memoryfacts/graph.go:197`, `compiler/internal/memoryfacts/validate.go:49`, and `compiler/internal/memoryfacts/report.go:188`.
- PLIR and allocplan facts needed explicit cost projection at existing compiler-owned insertion points; integrated in `compiler/internal/memoryfacts/from_plir.go:151`, `compiler/internal/memoryfacts/from_plir.go:768`, `compiler/internal/memoryfacts/from_plir.go:1183`, and `compiler/internal/memoryfacts/from_plir.go:1198`.
- CLI schema validation needed lockstep `cost_class` and dynamic-check rules; integrated in `tools/cmd/validate-memory-report/main.go`.
- Report generation was already optional/artifact-only through existing build report flags; no runtime semantic path was changed.

## RED Tests Integrated

- Missing and unknown `cost_class` rejection: `compiler/internal/memoryfacts/report_test.go:92`, `compiler/internal/memoryfacts/report_test.go:102`, and mirrored CLI tests.
- Dynamic optimization claims require `normal_build_check`: `compiler/internal/memoryfacts/report_test.go:112`.
- `unsafe_unknown` cannot claim zero-cost/trusted optimization: `compiler/internal/memoryfacts/report_test.go` and `tools/cmd/validate-memory-report/main_test.go`.
- PLIR unsafe gateway vocabulary now asserts cost-class mapping and normal-build checks in `compiler/internal/memoryfacts/from_plir_test.go:600`.

## Rejected / Non-Issues

- No need to make report generation mandatory; existing report flags keep memory reports artifact-only.
- No report-only source of truth was introduced; reports still project compiler-owned facts.

## Verification Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc14-cost go test -p=1 ./compiler/internal/memoryfacts ./tools/cmd/validate-memory-report ./compiler ./tools/cmd/memory-production-smoke -run 'Cost|Memory|Report|Dynamic|Instrumentation|Conservative|Unsafe|Optimization|Bounds' -count=1` passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc14-touched go test -p=1 ./compiler/internal/memoryfacts ./tools/cmd/validate-memory-report ./compiler ./tools/cmd/verify-docs ./tools/cmd/validate-manifest ./compiler/tests/semantics -count=1` passed.
