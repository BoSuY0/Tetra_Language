# Tetra Memory Zero-Heap Optimization Plan

**Status:** planning document, not implementation evidence. **Date:** 2026-06-16. **Owner:**
compiler/runtime/memory benchmark track. **Requested by:** user request to make Tetra memory usage
move toward bytes/KB for simple programs and controlled low-RAM behavior for large programs.

## 1. Goal

Make Tetra memory behavior predictable, measurable, and aggressively small.

Target end state:

- simple programs allocate zero Tetra heap bytes at runtime;
- simple benchmark rows fail validation if they regress into heap allocation;
- compiler storage selection prefers `eliminated`, `register`, `stack`, and `region` before `heap`;
- complex programs use explicit memory domains and budgets instead of one opaque heap;
- actor/task/request memory can be measured per owner;
- RSS is reduced by lazy runtime linking/init, not confused with heap bytes;
- every memory claim is backed by sidecar/report evidence.

## 2. Definitions

Use these terms exactly.

- `heap_alloc_bytes`: Tetra runtime heap bytes measured by Tetra heap telemetry.
- `heap_allocation_count`: counted Tetra heap allocations during program run.
- `bytes_requested`: logical bytes requested by source/runtime allocation intent.
- `bytes_reserved`: bytes reserved by allocator/region/backend.
- `bytes_committed`: bytes committed/usable by the backend when measured.
- `bytes_copied`: bytes copied because ownership was not moved or reused.
- `rss_current`: current OS process resident memory sample.
- `rss_peak`: peak OS process resident memory sample.
- `domain_bytes`: bytes charged to `process`, `actor`, `task`, `island`,
  `request`, or `external` domain.

Important boundary:

```text
heap_alloc_bytes != RSS
allocation_report_estimate != runtime_measured
NoEscape != automatically Stack
zero heap != zero RSS
```

## 3. Current Ground Truth

Observed current facts:

- Runtime heap telemetry exists in `docs/spec/telemetry/runtime_heap_telemetry.md`.
- Process RSS telemetry exists in `docs/spec/telemetry/process_rss_telemetry.md`.
- MemoryBackend vocabulary exists in `docs/spec/memory/memory_backend_vnext.md`.
- MemoryDomain vocabulary exists in `docs/spec/memory/memory_domains_vnext.md`.
- Allocation planner/lowering readiness is documented in
  `docs/design/memory/allocation_planner_lowering.md`.
- Current benchmark final report exists at
  `.workflow/benchmark-vnext-optimization-goal/final-report.md`.
- `hash_table_tetra` now proves `keys` and `values` as `NoEscape`, but keeps storage as `Heap`
  because local call stack/region lowering is not proven.
- `region/island allocation` remains a separate missing-feature/build-failure follow-up.
- Actor benchmark rows remain blocked because backend blockers and actor-domain byte evidence are
  still missing.
- `compiler/internal/runtimeabi/memory_domain.go` already defines memory domain structs and summary
  aggregation.

Current main blocker shape:

```text
NoEscape + Heap
```

Desired next shape:

```text
NoEscape + bounded lifetime + proven local use
=> Stack, Region, ExplicitIsland, Register, or Eliminated
```

## 4. Non-goals

Do not claim:

- zero heap for every possible Tetra program;
- zero RSS for OS processes;
- production OS memory usage;
- official benchmark results;
- cross-target RSS parity;
- distributed actor zero-copy;
- production actor runtime;
- Linux-specific behavior as language semantics;
- allocator performance superiority before benchmark evidence.

Do not wire GitHub Actions unless explicitly approved.

## 5. Success Metrics

### Simple programs

For simple benchmark categories, target:

```text
heap_allocation_count == 0
heap_alloc_bytes == 0
bytes_copied == 0 where ownership move/elimination is possible
```

Initial simple categories:

- `integer loops`
- `function calls`
- `slice sum`
- `bounds-check loops`
- small fixed-size local arrays
- simple read-only local calls
- simple struct/scalar copies

### Memory-heavy programs

For larger programs, target:

```text
heap allocations are justified
regions/domains have budgets
bytes_copied is visible
rss_peak has a local budget gate
```

### Actor/task programs

Target:

```text
actor mailbox bytes measured
message slab bytes measured
owned region bytes measured
per-actor peak bytes measured
byte-based backpressure validated
```

## 6. Execution Rules

- Use TDD for every compiler/runtime behavior change.
- First write a RED test that proves the current bad behavior.
- Keep heap/RSS/domain evidence separate.
- Use persistent Go caches under `.cache/go-build-*`, never `/tmp`.
- Run `graphify update .` after code changes.
- Use `git diff --check` before every completion claim.
- Keep reports honest: if a metric is not measured, mark it `unsupported` or `blocked`, not
  fake-measured.
- Do not delete or revert unrelated dirty worktree changes.

## 7. Phase Order

Execute phases in this order:

1. Baseline lock for memory optimization.
2. Call-aware stack/region lowering.
3. Zero-heap benchmark gates.
4. Region/island allocation closure.
5. Lazy runtime linking and initialization.
6. Runtime allocator classes.
7. Actor memory domains and byte backpressure.
8. RSS budget gates.
9. Final release-quality memory audit.

Each phase below is intentionally small enough to execute as a separate goal or checkpointed plan
batch.

## 8. Phase 0 - Baseline Lock

### Task 0.1 - Lock Current Memory Evidence

**Goal:** Start from known current evidence, not memory.

**Inspect:**

- `.workflow/benchmark-vnext-optimization-goal/final-report.md`
- `reports/benchmark-vnext-memory-baseline/tier1-after-actor-track/report.json`
- `reports/benchmark-vnext-memory-baseline/tier1-after-hash-track/report.json`
- `docs/audits/memory/zero-heap-final/benchmark-vnext-memory-baseline.md`

**Approach:**

- Record current classifications.
- Record current heap/RSS/domain support.
- Record current `hash_table_tetra` state:
  - `keys` and `values` are `NoEscape`;
  - storage remains `Heap`;
  - blocker is local call heap fallback.
- Record current `region/island allocation` state:
  - build failed or missing-feature.
- Record current actor state:
  - actor rows blocked;
  - actor-domain bytes unsupported.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-memory-zero-baseline"
REPORT="reports/benchmark-vnext-memory-baseline/tier1-after-actor-track/report.json"
GOCACHE="$CACHE" \
  go run ./tools/cmd/validate-local-benchmark-tier1 --report "$REPORT"
jq '
  .results[]
  | select(
      .category=="hash table"
      or .category=="region/island allocation"
      or .category=="actor ping-pong"
      or .category=="parallel map/reduce"
    )
  | {category, classification, classification_reason}
' "$REPORT"
git diff --check docs/plan/2026-06-16-tetra-memory-zero-heap-optimization-plan.md
GOCACHE="$CACHE" go clean -cache
```

**Done when:** A fresh baseline note exists and the current report validates.

**Notes:** This phase must not optimize anything.

## 9. Phase 1 - Call-Aware Stack/Region Lowering

This is the highest-value next optimization.

Current problem:

```text
read-only local call summary proves NoEscape
but storage stays Heap
```

Target:

```text
NoEscape + fixed/bounded local allocation + proven read-only local call
=> Stack or Region
```

### Task 1.1 - Audit The Current NoEscape Heap Fallback

**Goal:** Find the exact reason `NoEscape` allocations crossing local read-only calls still lower to
heap.

**Inspect:**

- `compiler/internal/allocplan/plan.go`
- `compiler/internal/allocplan/plan_test.go`
- `docs/design/memory/allocation_planner_lowering.md`
- Allocation artifact:
  - Directory: `reports/benchmark-vnext-memory-baseline/tier1-after-hash-track`
  - File: `artifacts/bin/hash_table_tetra.alloc.json`

**Approach:**

- Trace `classifyEscape`.
- Trace storage selection for `NoEscape`.
- Trace validation rules that reject stack storage across calls.
- Identify whether the missing proof belongs in:
  - allocation planner;
  - lowered IR validator;
  - call summary model;
  - storage selection;
  - backend lowering.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-call-aware-audit"
GOCACHE="$CACHE" \
  go test ./compiler/internal/allocplan \
  -run 'ReadOnly|Call|NoEscape|Stack|Region' \
  -count=1
GOCACHE="$CACHE" go clean -cache
```

**Done when:** The audit names one precise blocker and one first safe slice.

**Notes:** Do not change storage selection in this task.

### Task 1.2 - RED Test: Read-Only Local Call Can Be Trusted For Stack

**Goal:** Add a failing test for the first safe stack-lowering shape.

**Modify:**

- `compiler/internal/allocplan/plan_test.go`

**Test shape:**

```tetra
func consume(xs: []u8) -> Int
uses mem:
    return xs.len

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    return consume(xs)
```

Expected future behavior:

```text
escape = NoEscape
planned_storage = Stack
actual_lowering_storage = Stack
reason includes read-only local call proof
```

Current behavior should fail because storage is still heap.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-call-aware-stack"
GOCACHE="$CACHE" \
  go test ./compiler/internal/allocplan \
  -run TestPlannerStackLowersReadOnlyLocalCallAllocation \
  -count=1
```

**Done when:** The new test fails for the expected reason:

```text
want Stack
got Heap
```

**Notes:** If actual lowering cannot be stack yet, split the test:

```text
planned_storage = Stack
actual_lowering_storage = Heap
```

Then add a second lowering task later. Do not fake actual stack lowering.

### Task 1.3 - Implement Planner Proof Status

**Goal:** Add a distinct proof status for read-only local-call no-escape.

**Modify likely files:**

- `compiler/internal/allocplan/plan.go`
- `compiler/internal/allocplan/plan_test.go`

**Approach:**

- Keep unknown calls conservative.
- Keep unsafe/global/return/actor/task calls conservative.
- Add a proof marker such as:

```text
validated_read_only_local_call_no_escape
```

or reuse the existing proof vocabulary only if it already expresses this exact case.

- The proof must carry:
  - callee identity;
  - parameter indexes proven read-only;
  - allocation ids covered by the proof;
  - reason text.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-call-aware-stack"
GOCACHE="$CACHE" \
  go test ./compiler/internal/allocplan \
  -run 'ReadOnly|Call|NoEscape|Stack|Summary' \
  -count=1
```

**Done when:** The planner can distinguish:

```text
NoEscape intra-function
NoEscape read-only local call
EscapesCallUnknown
EscapesActor
EscapesTask
EscapesUnsafe
EscapesGlobal
EscapesReturn
```

**Notes:** Do not allow actor/task local calls to inherit read-only stack trust.

### Task 1.4 - Lowered IR Escape Validation For Stack Across Local Calls

**Goal:** Teach validation that a proven read-only local call does not make a stack-backed
allocation escape.

**Inspect:**

- `compiler/internal/validation`
- `compiler/internal/lower`
- `compiler/internal/plir`
- `docs/design/memory/allocation_planner_lowering.md`

**Approach:**

- Identify how stack-backed allocation tags are propagated.
- Add call-summary-aware validation:

```text
passing stack-backed header to proven read-only local callee is allowed
passing to unknown/external/actor/task/global/return path is rejected
```

- Keep validation as the safety gate. Planner proof alone is not enough.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-call-aware-validation"
GOCACHE="$CACHE" \
  go test \
  ./compiler/internal/validation/... \
  ./compiler/internal/lower \
  ./compiler/internal/plir \
  ./compiler/internal/allocplan \
  -run 'ReadOnly|Call|Stack|Escape|Translation|Validation' \
  -count=1
```

**Done when:** A forged or unsafe stack plan still fails validation, while the read-only local-call
case passes.

**Notes:** If validation cannot see enough call summary data, add an investigation task before
changing storage.

### Task 1.5 - Actual Stack Lowering For Fixed Small Local Calls

**Goal:** Move the first fixed-size read-only local-call allocation from heap to stack.

**Modify likely files:**

- `compiler/internal/allocplan/plan.go`
- `compiler/internal/lower`
- `compiler/internal/backend/x64core`
- `compiler/internal/validation`
- related tests in those packages

**Approach:**

- Restrict first slice:

  - fixed positive length;
  - known element size;
  - under existing stack threshold;
  - local read-only call only;
  - no actor/task/unsafe/global/return path;
  - no stored closure or aggregate escape.

- Do not handle dynamic large arrays yet.
- Do not handle recursive call graphs yet.
- Do not handle generic call summaries yet unless already supported.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-call-aware-lowering"
GOCACHE="$CACHE" \
  go test \
  ./compiler/internal/allocplan \
  ./compiler/internal/lower \
  ./compiler/internal/validation \
  ./compiler/internal/backend/... \
  ./compiler \
  -run 'ReadOnly|Call|Stack|Allocation|Lower|Backend|Translation' \
  -count=1
GOCACHE="$CACHE" go clean -cache
```

**Done when:** A fixed-size read-only local-call allocation reports:

```text
escape = NoEscape
planned_storage = Stack
actual_lowering_storage = Stack
heap_allocations unchanged or reduced
validation_status is compiler-owned proof
```

### Task 1.6 - Promote Hash Table From Heap Fallback When Safe

**Goal:** Apply call-aware storage to `hash_table_tetra` only if all safety conditions are met.

**Inspect:**

- `tools/internal/localbenchmarktier1/specs/tetra_sources.go`
- Allocation artifact:
  - Directory: `reports/benchmark-vnext-memory-baseline/tier1-after-hash-track`
  - File: `artifacts/bin/hash_table_tetra.alloc.json`
- `compiler/internal/allocplan`
- `compiler/internal/validation`

**Approach:**

- Confirm `keys` and `values` are fixed/bounded enough for stack.
- If fixed-size stack is safe, lower to stack.
- If dynamic length is not stack-safe, lower to function-temp region instead.
- If neither is proven, keep heap and record the exact missing proof.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-call-aware-hash"
OUT="reports/benchmark-vnext-memory-baseline/tier1-after-call-aware-stack-track"
GOCACHE="$CACHE" \
  go test \
  ./compiler/internal/allocplan/... \
  ./compiler/internal/validation/... \
  ./compiler \
  -run 'Hash|ReadOnly|Call|Stack|Region|Allocation|PerformanceReport' \
  -count=1
GOCACHE="$CACHE" \
  go run ./tools/cmd/local-benchmark-tier1 --out-dir "$OUT" --iterations 3
GOCACHE="$CACHE" \
  go run ./tools/cmd/validate-local-benchmark-tier1 --report "$OUT/report.json"
jq '
  .results[]
  | select(.category=="hash table")
  | {
      classification,
      classification_reason,
      tetra: (.rows[] | select(.language=="tetra") | .tetra_metadata)
    }
' "$OUT/report.json"
GOCACHE="$CACHE" go clean -cache
```

**Done when:** `hash table` is no longer blocked by heap allocation, or the report explains one
precise remaining blocker.

**Notes:** This is the first real memory optimization target after the completed benchmark goal.

## 10. Phase 2 - Zero-Heap Benchmark Gates

### Task 2.1 - Define Zero-Heap Policy

**Goal:** Define which benchmark rows must stay zero heap.

**Add:**

- `docs/spec/telemetry/zero_heap_benchmark_policy.md`

**Inspect:**

- `tools/cmd/local-benchmark-tier1`
- `tools/cmd/validate-local-benchmark-tier1`
- `docs/spec/telemetry/runtime_heap_telemetry.md`

**Initial policy:**

Rows that should be zero heap after current compiler support:

```text
integer loops
function calls
slice sum
bounds-check loops
startup time, if no user allocation exists
small fixed local allocation tests, once added
```

Rows not initially zero-heap-required:

```text
hash table, until call-aware stack/region closes
allocation
region/island allocation, until build failure closes
JSON/HTTP/PostgreSQL helper rows
actor ping-pong
parallel map/reduce
```

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-zero-heap-policy"
GOCACHE="$CACHE" \
  go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check docs/spec/telemetry/zero_heap_benchmark_policy.md
GOCACHE="$CACHE" go clean -cache
```

**Done when:** The policy separates required zero-heap rows from excluded rows.

### Task 2.2 - Add Validator Gate For Zero-Heap Rows

**Goal:** Fail local benchmark validation when a zero-heap-required row reports heap allocation.

**Modify:**

- `tools/cmd/validate-local-benchmark-tier1/main.go`
- `tools/cmd/validate-local-benchmark-tier1/main_test.go`

**Approach:**

- Add a zero-heap-required category list.
- For each successful Tetra row in that list, require:

```text
memory_evidence.heap_alloc_bytes.evidence_class == runtime_measured
heap_alloc_bytes.total_alloc_bytes == 0
heap_alloc_bytes.allocation_count == 0
tetra_metadata.heap_allocations == 0
```

- If heap evidence is missing, `unsupported`, or `blocked`, fail validation.
- If the row is excluded by policy, do not fail it.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-zero-heap-validator"
GOCACHE="$CACHE" \
  go test ./tools/cmd/validate-local-benchmark-tier1 \
  -run 'ZeroHeap|MemoryEvidence|Heap' \
  -count=1
GOCACHE="$CACHE" \
  go test \
  ./tools/cmd/local-benchmark-tier1 \
  ./tools/cmd/validate-local-benchmark-tier1 \
  -count=1
GOCACHE="$CACHE" go clean -cache
```

**Done when:** Validator rejects:

```text
zero-heap category with heap_allocation_count > 0
zero-heap category with heap_total_alloc_bytes > 0
zero-heap category with unsupported heap evidence
```

### Task 2.3 - Add Dedicated Zero-Heap Microbenchmarks

**Goal:** Add small benchmarks whose only purpose is to protect zero-heap compiler paths.

**Modify likely files:**

- `tools/internal/localbenchmarktier1/types.go`
- `tools/internal/localbenchmarktier1/specs/specs.go`
- `tools/internal/localbenchmarktier1/specs/tetra_sources.go`
- `tools/internal/localbenchmarktier1/specs/c_sources.go`
- `tools/internal/localbenchmarktier1/specs/rust_sources.go`
- `tools/internal/localbenchmarktier1/core_test.go`
- `tools/cmd/validate-local-benchmark-tier1/main.go`
- `tools/cmd/validate-local-benchmark-tier1/main_test.go`

**Candidate categories:**

```text
fixed local array sum
read-only local call slice
small struct copy
borrowed view sum
copy eliminated unused
```

**Approach:**

- Add one category at a time.
- Each category must have Tetra/C/C++/Rust rows or be explicitly local-Tetra optimizer evidence
  outside the Tier 1 comparable matrix.
- Prefer a separate zero-heap suite if adding categories to Tier 1 would distort the P20 matrix.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-zero-heap-suite"
OUT="reports/benchmark-vnext-memory-baseline/tier1-after-zero-heap-gates"
GOCACHE="$CACHE" \
  go test \
  ./tools/cmd/local-benchmark-tier1 \
  ./tools/cmd/validate-local-benchmark-tier1 \
  -run 'ZeroHeap|BuildSpecs|ValidateReport|MemoryEvidence' \
  -count=1
GOCACHE="$CACHE" \
  go run ./tools/cmd/local-benchmark-tier1 --out-dir "$OUT" --iterations 3
GOCACHE="$CACHE" \
  go run ./tools/cmd/validate-local-benchmark-tier1 --report "$OUT/report.json"
GOCACHE="$CACHE" go clean -cache
```

**Done when:** Zero-heap categories validate and fail on intentional heap regressions.

## 11. Phase 3 - Region/Island Allocation Closure

### Task 3.1 - Reproduce Region/Island Build Failure

**Goal:** Identify why `region/island allocation` is not a measured Tetra row.

**Inspect:**

- `tools/internal/localbenchmarktier1/specs/tetra_sources.go`
- `compiler/internal/allocplan`
- `compiler/internal/lower`
- `compiler/internal/runtimeabi`
- `docs/spec/memory/islands.md`
- `docs/design/memory/allocation_planner_lowering.md`

**Approach:**

- Run the single category build path.
- Capture the exact compiler diagnostic/build failure.
- Decide if the blocker is syntax, semantics, lowering, runtime ABI, or report validation.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-region-island-repro"
OUT="reports/benchmark-vnext-memory-baseline/tier1-region-island-repro"
GOCACHE="$CACHE" \
  go test ./tools/cmd/local-benchmark-tier1 \
  -run 'Region|Island|BuildSpecs' \
  -count=1
GOCACHE="$CACHE" \
  go run ./tools/cmd/local-benchmark-tier1 --out-dir "$OUT" --iterations 1
jq '.results[] | select(.category=="region/island allocation")' "$OUT/report.json"
GOCACHE="$CACHE" go clean -cache
```

**Done when:** The failure has one named root cause.

### Task 3.2 - RED Test For Minimal Island Allocation

**Goal:** Add the smallest failing compiler/runtime test for explicit island allocation.

**Modify likely files:**

- `compiler/internal/allocplan/plan_test.go`
- `compiler/internal/lower`
- `compiler/internal/validation`
- `compiler/tests/semantics`

**Expected behavior:**

```text
core.island_make_u8 / island_make_i32
=> planned_storage = ExplicitIsland
=> actual_lowering_storage = ExplicitIsland
=> heap allocation count does not increase
```

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-region-island-red"
GOCACHE="$CACHE" \
  go test \
  ./compiler/internal/allocplan \
  ./compiler/internal/lower \
  ./compiler/internal/validation \
  ./compiler/tests/semantics \
  -run 'Island|Region|Allocation|ExplicitIsland' \
  -count=1
```

**Done when:** The RED test fails for the missing island lowering/build reason.

### Task 3.3 - Implement Minimal ExplicitIsland Lowering

**Goal:** Make the benchmark's region/island row build and run with honest memory evidence.

**Modify likely files:**

- `compiler/internal/lower`
- `compiler/internal/validation`
- `compiler/internal/runtimeabi`
- `compiler/internal/allocplan`
- target backend files only after identifying the real missing lowering path

**Approach:**

- Keep first slice explicit island only.
- Do not introduce implicit region inference yet.
- Require island scope dominance.
- Reject returned island-backed pointers unless ownership/lifetime proves them safe.
- Report domain kind `island`.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-region-island"
OUT="reports/benchmark-vnext-memory-baseline/tier1-after-region-island-track"
GOCACHE="$CACHE" \
  go test \
  ./compiler/internal/allocplan/... \
  ./compiler/internal/lower \
  ./compiler/internal/validation/... \
  ./compiler/internal/runtimeabi \
  ./compiler \
  -run 'Island|Region|Allocation|Domain|Lower|Translation' \
  -count=1
GOCACHE="$CACHE" \
  go run ./tools/cmd/local-benchmark-tier1 --out-dir "$OUT" --iterations 3
GOCACHE="$CACHE" \
  go run ./tools/cmd/validate-local-benchmark-tier1 --report "$OUT/report.json"
jq '
  .results[]
  | select(.category=="region/island allocation")
  | {classification, rows}
' "$OUT/report.json"
GOCACHE="$CACHE" go clean -cache
```

**Done when:** `region/island allocation` is no longer `blocked by missing feature`, or the report
names one smaller remaining blocker.

## 12. Phase 4 - Lazy Runtime Linking And Initialization

### Task 4.1 - Runtime Dependency Audit

**Goal:** Identify which runtime pieces are linked/initialized for each simple program.

**Inspect:**

- `compiler`
- `compiler/internal/backend`
- `compiler/internal/runtimeabi`
- `compiler/internal/buildruntime`
- `compiler/internal/buildreports`
- `compiler/compiler_reports.go`
- generated benchmark binaries under current report artifacts

**Approach:**

- For each simple benchmark, record whether it uses:

  - heap runtime;
  - actor runtime;
  - task runtime;
  - IO runtime;
  - scheduler;
  - net runtime;
  - island runtime.

- Use existing build reports if possible.
- If build reports do not expose runtime dependencies, add a report-only task before changing
  linking.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-lazy-runtime-audit"
GOCACHE="$CACHE" \
  go test ./compiler \
  -run 'BuildReport|Runtime|Link|Explain' \
  -count=1
GOCACHE="$CACHE" go clean -cache
```

**Done when:** A table exists:

```text
benchmark -> runtime features linked -> runtime features initialized
```

### Task 4.2 - Add Runtime Feature Report

**Goal:** Make runtime linkage visible in reports.

**Modify likely files:**

- `compiler/compiler_reports.go`
- `compiler/internal/buildreports`
- `compiler/compiler_suite_test.go`
- `tools/internal/localbenchmarktier1/metadata.go`
- `tools/internal/localbenchmarktier1/types.go`

**Report fields:**

```text
runtime_features_required
runtime_features_linked
runtime_features_initialized
runtime_lazy_init_blockers
```

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-lazy-runtime-report"
GOCACHE="$CACHE" \
  go test \
  ./compiler \
  ./tools/cmd/local-benchmark-tier1 \
  ./tools/cmd/validate-local-benchmark-tier1 \
  -run 'Runtime|Feature|Report|Benchmark|Validate' \
  -count=1
GOCACHE="$CACHE" go clean -cache
```

**Done when:** Reports can prove when a simple program does not pull actor/task runtime.

### Task 4.3 - Lazy Link Actor/Task/Heap Runtime

**Goal:** Do not link or initialize unused runtime subsystems.

**Approach:**

- Start with actor runtime:

```text
no actor syntax/effects/calls
=> no actor runtime symbols linked
```

- Then task runtime:

```text
no task/async calls
=> no task runtime symbols linked
```

- Then heap runtime:

```text
no heap allocations after lowering
=> no heap runtime init
```

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-lazy-runtime"
OUT="reports/benchmark-vnext-memory-baseline/tier1-after-lazy-runtime-track"
GOCACHE="$CACHE" \
  go test \
  ./compiler \
  ./compiler/internal/backend/... \
  -run 'Runtime|Link|Actor|Task|Heap|Symbol|Smoke' \
  -count=1
GOCACHE="$CACHE" \
  go run ./tools/cmd/local-benchmark-tier1 --out-dir "$OUT" --iterations 3
GOCACHE="$CACHE" \
  go run ./tools/cmd/validate-local-benchmark-tier1 --report "$OUT/report.json"
GOCACHE="$CACHE" go clean -cache
```

**Done when:** Simple programs show fewer linked/initialized runtime features and no benchmark truth
validator weakens.

**Notes:** This phase reduces RSS. It does not directly prove heap reductions.

## 13. Phase 5 - Runtime Allocator Classes

### Task 5.1 - Keep Heap As Last Resort

**Goal:** Make the allocation path explicit for every heap use.

**Inspect:**

- `compiler/internal/runtimeabi`
- `compiler/internal/allocplan`
- `docs/spec/memory/memory_backend_vnext.md`

**Approach:**

Every heap allocation must answer:

```text
Why not eliminated?
Why not register?
Why not stack?
Why not region/island?
Why not actor/task/request domain?
```

Add reason codes for remaining heap:

```text
heap.required_escape_return
heap.required_unknown_call
heap.required_actor_boundary
heap.required_task_boundary
heap.required_dynamic_lifetime
heap.required_large_object
heap.required_ffi_external
```

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-heap-reasons"
GOCACHE="$CACHE" \
  go test \
  ./compiler/internal/allocplan \
  ./compiler/internal/runtimeabi \
  ./compiler \
  -run 'Heap|Allocation|Reason|Report' \
  -count=1
GOCACHE="$CACHE" go clean -cache
```

**Done when:** No heap allocation in allocation reports lacks a reason code.

### Task 5.2 - Function/Request Region Allocator

**Goal:** Add region allocation for short-lived temporary memory that is bigger or more dynamic than
stack.

**Approach:**

- Start with function-temp region.
- Later add request region.
- Free/reset the whole region at boundary.
- Track:

```text
region_id
lifetime
requested_bytes
reserved_bytes
released_bytes
peak_bytes when measured
```

**Modify likely files:**

- `compiler/internal/runtimeabi`
- `compiler/internal/allocplan`
- `compiler/internal/lower`
- `compiler/internal/validation`
- benchmark report tooling

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-region-allocator"
GOCACHE="$CACHE" \
  go test \
  ./compiler/internal/runtimeabi \
  ./compiler/internal/allocplan \
  ./compiler/internal/lower \
  ./compiler/internal/validation \
  ./compiler \
  -run 'Region|FunctionTemp|Request|Allocation|Domain|Release' \
  -count=1
GOCACHE="$CACHE" go clean -cache
```

**Done when:** A temporary dynamic allocation can use a region without heap and without escaping the
region lifetime.

### Task 5.3 - Large Object Backend Path

**Goal:** Keep huge allocations out of the small heap/region path.

**Approach:**

- Define threshold.
- Route large allocations to `large_mmap` or target equivalent.
- Report reserve/commit/release using `MemoryBackend` terms.
- Keep target adapter evidence separate from language semantics.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-large-backend"
GOCACHE="$CACHE" \
  go test \
  ./compiler/internal/runtimeabi \
  ./compiler/internal/allocplan \
  ./compiler \
  -run 'Large|Mmap|MemoryBackend|Reserve|Commit|Release|Footprint' \
  -count=1
GOCACHE="$CACHE" go clean -cache
```

**Done when:** Large allocation rows are not mixed with small heap rows and carry backend evidence.

## 14. Phase 6 - Actor Memory Domains

### Task 6.1 - Actor Domain Runtime Accounting

**Goal:** Turn actor domain vocabulary into runtime evidence for local actor benchmarks.

**Inspect:**

- `compiler/internal/parallelrt`
- `compiler/internal/actorsrt`
- `compiler/internal/actorsafety`
- `tools/cmd/parallel-production-smoke`
- `tools/validators/parallelprod`
- `tools/cmd/local-benchmark-tier1`

**Track per actor:**

```text
mailbox_pool_bytes
message_slab_bytes
owned_region_bytes
queued_message_bytes
current_bytes
peak_bytes
bytes_copied
copy_count
budget_bytes
```

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-actor-domain"
GOCACHE="$CACHE" \
  go test \
  ./compiler/internal/parallelrt/... \
  ./compiler/internal/actorsrt/... \
  ./compiler/internal/actorsafety/... \
  ./tools/validators/parallelprod/... \
  ./tools/cmd/parallel-production-smoke \
  -run 'Actor|MemoryDomain|Mailbox|Budget|Bytes|Backpressure|ZeroCopy|Claim' \
  -count=1
GOCACHE="$CACHE" go clean -cache
```

**Done when:** Actor domain bytes are measured or explicitly blocked with a runtime reason, not just
allocation-report estimates.

### Task 6.2 - Byte-Based Actor Backpressure

**Goal:** Backpressure actors by bytes, not only message count.

**Approach:**

Current:

```text
mailbox full by message count
```

Target:

```text
mailbox full by message count OR queued bytes
```

Rules:

- small messages can fit many;
- huge messages consume byte budget;
- owned-region transfer charges bytes to receiver domain;
- copied payload increments `bytes_copied`.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-actor-byte-backpressure"
GOCACHE="$CACHE" \
  go test \
  ./compiler/internal/parallelrt/... \
  ./compiler/internal/actorsrt/... \
  ./compiler/internal/actorsafety/... \
  -run 'Actor|Mailbox|Backpressure|Bytes|Budget|Domain' \
  -count=1
GOCACHE="$CACHE" go clean -cache
```

**Done when:** An actor can reject or delay messages because of byte budget.

### Task 6.3 - Zero-Copy Ownership Move As Domain Transfer

**Goal:** Make local actor zero-copy a domain owner change, not a copy.

**Approach:**

Desired model:

```text
sender actor domain owns region
send owned region
receiver actor domain owns region
bytes_copied stays 0
domain transfer event is recorded
```

Reject:

```text
borrowed data transfer
unknown lifetime transfer
distributed zero-copy claim
cross-runtime zero-copy claim
```

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-actor-zero-copy"
GOCACHE="$CACHE" \
  go test \
  ./compiler/internal/parallelrt/... \
  ./compiler/internal/actorsafety/... \
  ./tools/validators/parallelprod/... \
  -run 'ZeroCopy|OwnedRegion|Actor|Domain|Transfer|Claim' \
  -count=1
GOCACHE="$CACHE" go clean -cache
```

**Done when:** Local owned-region actor transfer records owner movement and does not increment copy
bytes.

## 15. Phase 7 - RSS Budget Gates

### Task 7.1 - Define Local RSS Budgets

**Goal:** Add local RSS budgets without pretending RSS is cross-machine stable.

**Add:**

- `docs/spec/telemetry/local_rss_budget_policy.md`

**Approach:**

- Budgets are local gates, not global claims.
- Store:

```text
host info
target
benchmark category
rss_peak budget
allowed variance
reason for budget
```

- Do not compare RSS across machines unless host profile is pinned.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-rss-budget-docs"
GOCACHE="$CACHE" \
  go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check docs/spec/telemetry/local_rss_budget_policy.md
GOCACHE="$CACHE" go clean -cache
```

**Done when:** RSS policy says exactly what can and cannot be claimed.

### Task 7.2 - Add RSS Budget Validator

**Goal:** Fail local benchmark validation when `rss_peak` exceeds a configured local budget.

**Modify likely files:**

- `tools/cmd/validate-local-benchmark-tier1/main.go`
- `tools/cmd/validate-local-benchmark-tier1/main_test.go`
- optional budget config file under `docs/` or `tools/`

**Approach:**

- Require `rss_peak.runtime_measured`.
- Compare only against local policy.
- If host profile does not match, mark budget check `blocked`, not failed.
- Keep functional validator separate from budget validator if necessary.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-rss-budget"
GOCACHE="$CACHE" \
  go test ./tools/cmd/validate-local-benchmark-tier1 \
  -run 'RSS|Budget|Peak|Host|Policy' \
  -count=1
GOCACHE="$CACHE" go clean -cache
```

**Done when:** RSS regressions can fail a local gate without creating a global RSS claim.

## 16. Phase 8 - Final Memory Release Gate

### Task 8.1 - Memory Optimization Completion Audit

**Goal:** Produce a final audit that maps every memory claim to evidence.

**Add:**

- `docs/audits/memory/zero-heap-final/tetra-memory-zero-heap-optimization-final.md`

**Must include:**

- zero-heap rows;
- rows still using heap and why;
- region/island status;
- actor-domain status;
- RSS budget status;
- bytes copied summary;
- known target limitations;
- nonclaims.

**Verification:**

```sh
CACHE="$(pwd)/.cache/go-build-memory-final-docs"
GOCACHE="$CACHE" \
  go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check docs/audits/memory/zero-heap-final/tetra-memory-zero-heap-optimization-final.md
GOCACHE="$CACHE" go clean -cache
```

**Done when:** A reader can tell which memory improvements are implemented, which are measured, and
which are still planned.

### Task 8.2 - Final Benchmark Matrix

**Goal:** Generate one final memory optimization report.

**Command:**

```sh
CACHE="$(pwd)/.cache/go-build-memory-final"
OUT="reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization"
GOCACHE="$CACHE" \
  go run ./tools/cmd/local-benchmark-tier1 --out-dir "$OUT" --iterations 5
GOCACHE="$CACHE" \
  go run ./tools/cmd/validate-local-benchmark-tier1 --report "$OUT/report.json"
```

**Extra checks:**

```sh
OUT="reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization"
REPORT="$OUT/report.json"
jq '.results[] | {category, classification, classification_reason}' "$REPORT"
jq '
  [
    .results[].rows[]
    | select(.language=="tetra")
    | {
        name,
        category,
        heap: .tetra_metadata.memory_evidence.heap_alloc_bytes,
        rss_peak: .tetra_metadata.memory_evidence.rss_peak,
        domain: .tetra_metadata.memory_evidence.domain_bytes_evidence
      }
  ]
' "$REPORT"
```

**Done when:** The final report validates and shows which rows reached zero heap.

### Task 8.3 - Final Verification

**Goal:** Close the track only after local evidence proves the claims.

**Run:**

```sh
CACHE="$(pwd)/.cache/go-build-memory-final"
TOOLS_CACHE="$(pwd)/.cache/go-build-memory-final-tools"
DOCS_CACHE="$(pwd)/.cache/go-build-memory-final-docs"
GOCACHE="$CACHE" \
  go test \
  ./compiler/internal/allocplan/... \
  ./compiler/internal/runtimeabi/... \
  ./compiler/internal/ramcontract/... \
  ./compiler/internal/memoryfacts/... \
  ./compiler/internal/validation/... \
  ./compiler \
  -run 'Allocation|Memory|Region|Island|Domain|Heap|Stack|RSS|Backend|Actor' \
  -count=1
GOCACHE="$CACHE" \
  go test \
  ./compiler/internal/allocplan/... \
  -run 'Task|ZeroCopy|Bounds|Translation' \
  -count=1
GOCACHE="$TOOLS_CACHE" \
  go test \
  ./tools/internal/rsstelemetry \
  ./tools/internal/heaptelemetry \
  ./tools/cmd/local-benchmark-tier1 \
  ./tools/cmd/validate-local-benchmark-tier1 \
  -count=1
GOCACHE="$DOCS_CACHE" \
  go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
graphify update .
```

**Clean caches:**

```sh
CACHE="$(pwd)/.cache/go-build-memory-final"
TOOLS_CACHE="$(pwd)/.cache/go-build-memory-final-tools"
DOCS_CACHE="$(pwd)/.cache/go-build-memory-final-docs"
GOCACHE="$CACHE" go clean -cache
GOCACHE="$TOOLS_CACHE" go clean -cache
GOCACHE="$DOCS_CACHE" go clean -cache
rm -rf \
  .cache/go-build-memory-final \
  .cache/go-build-memory-final-tools \
  .cache/go-build-memory-final-docs
```

**Done when:** All commands pass and the final audit says exactly what is done and what is not done.

## 17. Recommended First Implementation Goal

Start with one focused goal:

```text
Implement call-aware stack/region lowering for the first safe read-only local
call allocation shape, prove it with RED/GREEN allocplan/lowering/validation
tests, regenerate a fresh benchmark report, and keep unknown/unsafe/actor/task
escapes conservative.
```

First success target:

```text
read_only_call(xs: make_u8(4))
NoEscape + read-only local call summary
Heap -> Stack
```

Second success target:

```text
hash_table_tetra keys/values
NoEscape + Heap -> Stack or Region
```

Do not start with allocator internals. First remove unnecessary allocations.

## 18. Stop Rules

Stop and report `PARTIAL` if:

- stack lowering crosses a call without validation proof;
- an unknown/unsafe/actor/task/global/return escape becomes stack or region;
- RSS is reported from heap telemetry;
- allocation estimate is reported as runtime measured;
- actor-domain bytes are claimed without runtime/domain evidence;
- `region/island allocation` is marked implemented while still build-failing;
- zero-heap validator can be bypassed by missing heap sidecars;
- any final report weakens benchmark nonclaims.

## 19. Expected Result If All Phases Succeed

Simple programs:

```text
heap_allocation_count = 0
heap_alloc_bytes = 0
bytes_requested = 0 or explicitly justified
bytes_copied = 0 where avoidable
```

Medium programs:

```text
stack and region replace temporary heap
heap remains only for justified escape/lifetime cases
```

Actor programs:

```text
actor memory domains show mailbox/message/owned-region bytes
byte backpressure protects memory
local owned-region move can avoid copies
```

Process RSS:

```text
lower because unused runtime pieces are not linked/initialized
still never claimed as zero
```

The final product claim should be:

```text
Tetra has evidence-backed zero-heap paths for simple programs and domain-based
low-memory control for larger programs.
```

Not:

```text
Tetra always uses zero memory.
```
