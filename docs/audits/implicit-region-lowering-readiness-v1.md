# Implicit Region Lowering Readiness v1 Closure

Goal slice: P15.0 Implicit Region Lowering Readiness.

Baseline: `tetra.truthful-performance-core.baseline.20260602.v1`.

Status: complete for slice after focused implementation and verification.

## Scope

This slice turns the prior `planned_storage: Region` plus heap fallback model
into an explicit, opt-gated function-temp region lowering path at the planner,
IR, lowerer, validator, and allocation-report boundary. It is a readiness slice:
production target lowering remains unchanged unless the new
`FunctionTempRegionLowering` option is explicitly enabled in lowerer tests.

## Implemented Rules

- Rule: function-temp region ABI has explicit IR instructions.
  - Instructions: enter/reset and region make-slice.
  - Evidence:
    - `compiler/internal/ir/ir.go`
    - `compiler/internal/lower/verify.go`
- Rule: planner can report `ActualLoweringStorage=Region`.
  - Scope: bounded function-local temporary copies with region lowering enabled.
  - Evidence: `TestPlannerReportsActualFunctionTempRegionLoweringWhenEnabled`
- Rule: existing region planning without lowering still reports heap fallback.
  - Evidence: `TestPlannerSelectsFunctionTempRegionForTemporaryCopyWhenEnabled`
- Rule: lowering emits supported function-temp region IR.
  - Instructions: `IRRegionEnter`, `IRRegionMakeSlice*`, and `IRRegionReset`.
  - Scope: supported non-escaping temporary copies.
  - Evidence: `TestLowerFunctionTempRegionCopyEmitsEnterMakeAndReset`
- Rule: escaping region-backed slices are rejected by validation.
  - Evidence: `TestValidateAllocationLoweringRejectsReturnedFunctionTempRegionAllocation`
- Rule: validation rejects stale actual-region claims without matching IR.
  - Evidence: `TestValidateAllocationLoweringRejectsMissingFunctionTempRegionIR`
- Rule: validation requires region reset to dominate returns.
  - Evidence:
    - `validateFunctionTempRegionResets`
    - validation package gates
- Rule: allocation reports expose function-temp region rows and summaries.
  - Evidence: `TestWrapAllocationPlanReportV2IncludesFunctionTempRegionSummary`

## Code Changes

- `compiler/internal/ir/ir.go` adds `IRRegionEnter`,
  `IRRegionMakeSliceU8`, `IRRegionMakeSliceU16`, `IRRegionMakeSliceI32`, and
  `IRRegionReset`.
- `compiler/internal/allocplan/plan.go` adds `EnableRegionLowering` and reports
  `function_temp_region_lowering` only when actual lowering is explicitly
  enabled.
- `compiler/internal/lower/lower.go` adds `FunctionTempRegionLowering`, emits
  function-temp region IR for supported copy allocations, and inserts reset
  before return paths.
- `compiler/internal/validation/validation.go` validates matching region IR,
  no escape, and reset-before-return dominance for function-temp region
  allocations.
- `compiler/reports_internal_test.go` adds report-row coverage for
  function-temp region summaries.

## Graphify Navigation Evidence

Graphify MCP was used before concrete file inspection:

```text
query_graph:
  P15.0 Implicit Region Lowering Readiness function-temp region ABI
  region enter reset allocation planner region report rows
  dominance validation non-escaping temporary copies
  escaping region-backed slice
get_neighbors: FromPLIRWithOptions()
get_neighbors: ValidateAllocationLowering()
get_neighbors: .lowerStackCopyLet()
shortest_path: FromPLIRWithOptions() -> ValidateAllocationLowering()
```

The graph identified `compiler/internal/allocplan/plan.go`,
`compiler/internal/lower/lower.go`, `compiler/internal/validation/validation.go`,
`compiler/internal/ir/ir.go`, existing stack-copy lowering tests, and allocation
report tests as the relevant boundary.

## Verification Evidence

RED evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler/internal/allocplan \
  -run 'FunctionTempRegion|TemporaryCopyWhenEnabled' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler/internal/lower \
  -run 'FunctionTempRegionCopy' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler/internal/validation \
  -run 'FunctionTempRegion' -count=1
```

Initial result: failed at compile time for the right reason: the requested
`EnableRegionLowering`/`FunctionTempRegionLowering` options and `IRRegion*`
instructions did not exist.

Focused GREEN evidence:

```bash
REGION_RUN='FunctionTempRegion|TemporaryCopyWhenEnabled'
REGION_RUN="$REGION_RUN|BudgetChargeModelIsExplicit"
REGION_RUN="$REGION_RUN|WrapAllocationPlanReportV2IncludesFunctionTempRegionSummary"

GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test \
  ./compiler/internal/allocplan \
  ./compiler/internal/lower \
  ./compiler/internal/validation \
  ./compiler \
  -run "$REGION_RUN" -count=1
```

Result: pass.

Relevant package evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/allocplan -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/lower -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/validation -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler -run 'Allocation|Reports|Explain' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test \
  ./compiler/internal/backend/x64core \
  ./compiler/internal/backend/x64abi \
  ./compiler/internal/backend/linux_x86 \
  ./compiler/internal/backend/wasm32_wasi \
  ./compiler/internal/backend/wasm32_web \
  -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler/internal/ir ./compiler/internal/runtimeabi -count=1
```

Result: pass.

Final hygiene evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test \
  ./compiler/internal/allocplan \
  ./compiler/internal/lower \
  ./compiler/internal/validation \
  ./compiler/internal/runtimeabi \
  ./compiler/internal/backend/x64core \
  ./compiler/internal/backend/x64abi \
  ./compiler/internal/backend/linux_x86 \
  ./compiler/internal/backend/wasm32_wasi \
  ./compiler/internal/backend/wasm32_web \
  ./compiler \
  -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
git diff --check
graphify update .
```

Result: pass. Graphify rebuilt `18942 nodes, 60798 edges, 1083 communities`.

Additional final checks:

```bash
rg -n '[[:blank:]]$' \
  GOAL.md PLAN.md ATTEMPTS.md NOTES.md CONTROL.md \
  reports/implicit-region-lowering-readiness-v1/closure.md \
  docs/audits/implicit-region-lowering-readiness-v1.md \
  compiler/internal/ir/ir.go \
  compiler/internal/allocplan/plan.go \
  compiler/internal/allocplan/plan_test.go \
  compiler/internal/lower/lower.go \
  compiler/internal/lower/verify.go \
  compiler/internal/lower/verify_test.go \
  compiler/internal/lower/allocation_stack_test.go \
  compiler/internal/validation/validation.go \
  compiler/internal/validation/validation_test.go \
  compiler/reports_internal_test.go \
  docs/generated/manifest.json
rg -n \
  -e 'tetra_surface_release_promotion_v1_full_plan' \
  -e 'source_plan: /home/tetra/Downloads/tetra_surface_release' \
  -e 'Active slice: Section' \
  -e 'Surface Release Promotion v1' \
  GOAL.md PLAN.md ATTEMPTS.md NOTES.md CONTROL.md
```

Result: whitespace scan passed. Drift scan found sidecars overwritten to the
stale Surface Release Promotion goal again before Graphify update; sidecars were
recreated to Ideal with P15.0 complete and P15.1 active before continuing.

## Non-Claims

- P15.0 does not enable function-temp region lowering in the default production
  target path.
- P15.0 does not implement request/task regions; that remains P15.1.
- P15.0 does not replace the heap allocator or implement per-core allocators;
  that remains P15.2.
- P15.0 does not permit escaping region-backed slices; validation rejects them.
- P15.0 does not claim backend object code support for the new implicit region
  IR outside the explicit lowerer readiness path.
