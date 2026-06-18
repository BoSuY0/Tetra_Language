# Benchmark vNext Fallback Backend Track

Status: follow-up plan opened from the fresh memory-aware Tier 1 baseline.

Primary audit: `docs/audits/memory/zero-heap-final/benchmark-vnext-memory-baseline.md`.

## Goal

Reduce the highest-leverage `blocked by fallback backend` rows without weakening validation, safety
reporting, bounds checks, or memory evidence.

## Current Evidence

Fresh report: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/report.json`.

Rows:

| Row                    | Current cause                                                           | Artifact                                                                                                         |
| ---------------------- | ----------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `integer_loops_tetra`  | `unsupported_control_flow`, one stack fallback function                 | `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/integer_loops_tetra.backend.json`  |
| `function_calls_tetra` | `main` uses `unsupported_control_flow`; helper is already register path | `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/function_calls_tetra.backend.json` |
| `recursion_tetra`      | `fib` and `main` use `unsupported_control_flow`                         | `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/recursion_tetra.backend.json`      |
| `allocation_tetra`     | `unsupported_effect_runtime_call`                                       | `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/allocation_tetra.backend.json`     |

Secondary fallback rows exist in invalid/inconclusive helper categories, but those should not drive
this track before the primary matrix rows.

## Proposed Slice

Start with scalar control-flow fallback, not runtime calls.

Acceptance for the first patch should target one narrow behavior:

- `integer_loops_tetra` moves from stack fallback to register path, or
- `function_calls_tetra.main` moves from stack fallback to register path while keeping `mix`
  unchanged as a passing control.

The first slice should not include actor calls, task spawn ABI, heap runtime calls, multi-slot
returns, or service benchmark helper rows.

## Implementation Boundary

Likely files to inspect before editing:

- `compiler/compiler_reports.go`
- `compiler/compiler_suite_test.go`
- `compiler/internal/backend/linux_x86/codegen*.go`
- `compiler/internal/backend/x64core/emit*.go`
- `compiler/internal/lower`
- `compiler/internal/validation`
- `tools/internal/localbenchmarktier1/specs/tetra_sources.go`

Required behavior:

- Preserve translation/differential validation.
- Keep stack fallback visible when a function is still outside the register backend subset.
- Preserve backend artifact reasons; do not relabel stack fallback as register evidence.

## Tests First

Add or extend focused tests that fail before the implementation:

- backend report test for a simple loop function that should be register-path eligible;
- benchmark-row regression that confirms `integer_loops_tetra` no longer reports
  `unsupported_control_flow` after the patch;
- negative test for an unsupported runtime call still reporting stack fallback.

## Verification

Focused:

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-backend go test ./compiler/internal/backend/... ./compiler/internal/validation/... ./compiler -run 'Backend|Fallback|Register|Stack|Translation|Differential' -count=1
```

Benchmark:

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-backend go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-after-fallback-track --iterations 3
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-backend go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-fallback-track/report.json
```

## Nonclaims

- No speed claim from a backend-path change alone.
- No actor/runtime-call promotion in this first slice.
- No weakening of bounds, allocation, memory, or validator evidence to improve classification.
