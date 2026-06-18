# Tetra Native Memory Completion Master Plan

**Status:** execution plan, not implementation evidence.  
**Date:** 2026-06-17.  
**Scope:** fallback backend, bounds-check regression safety, heap zero-regression safety, production
actor memory, RSS reduction.  
**Current truth report:**
`reports/benchmark-vnext-memory-baseline/tier1-after-postgresql-inout-writer-native/report.json`.  
**Workflow kernel:** `.workflow/post-zero-heap-native-memory/`.  
**Previous plan:** `docs/plan/2026-06-16-tetra-post-zero-heap-native-memory-plan.md`.

## 1. Plain Goal

The goal is to make Tetra's memory and native-backend story real, stable, and hard to fake.

This does not mean "make one report look nice". It means:

1. benchmark rows leave `fallback` only when the compiler emits a real native/register path;
2. removed bounds checks always have proof evidence;
3. heap stays at runtime-measured zero for the already-clean benchmark rows;
4. actor memory is measured and enforced in the production actor runtime;
5. RSS is reduced where possible and guarded with host-pinned local regression budgets.

Important separation:

- Metric: `heap_alloc_bytes`
  - Meaning: bytes allocated through Tetra/runtime heap paths.
  - Can it be zero for simple rows: yes.
  - Notes: already zero for all 17 Tetra Tier 1 rows in the current report.
- Metric: `bounds_left`
  - Meaning: remaining runtime bounds checks after proof/lowering.
  - Can it be zero for simple rows: yes.
  - Notes: already zero for all 17 Tetra Tier 1 rows in the current report.
- Metric: `backend_path`
  - Meaning: whether a row/function uses register/native or fallback path.
  - Can it be zero for simple rows: yes for some rows.
  - Notes: still the biggest blocker.
- Metric: `domain_bytes`
  - Meaning: bytes attributed to process/actor/request/domain ownership.
  - Can it be zero for simple rows: yes, but must be honest.
  - Notes: actor budget/backpressure is not complete.
- Metric: `rss_peak`
  - Meaning: OS resident memory, including loader/code/runtime/stack/libs/pages.
  - Can it be zero for simple rows: no honest zero claim.
  - Notes: must be treated as local, host-specific evidence.

## 2. Current Truth From P72

Source:
`reports/benchmark-vnext-memory-baseline/tier1-after-postgresql-inout-writer-native/report.json`

Generated at: `2026-06-17T10:34:06Z`  
Host: `linux/amd64`, CPU `Intel(R) Core(TM) i9-14900HX`  
Git commit in report host metadata: `95bfd4a887bab5032437cb22494d034e82ae6d35`

Current Tetra summary:

| Fact | Value |
| --- | ---: |
| Tetra rows | 17 |
| Runtime-measured zero-heap rows | 17 |
| Rows with `bounds_left == 0` | 17 |
| Rows still in `fallback` | 8 |

Current row records:

- `integer_loops_tetra`:
  - Category: integer loops.
  - Backend: `register`.
  - Backend blockers: none.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `slice_sum_tetra`:
  - Category: slice sum.
  - Backend: `fallback`.
  - Backend blockers: `unsupported_control_flow`.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `bounds_check_loops_tetra`:
  - Category: bounds-check loops.
  - Backend: `register`.
  - Backend blockers: none.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `function_calls_tetra`:
  - Category: function calls.
  - Backend: `register`.
  - Backend blockers: none.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `recursion_tetra`:
  - Category: recursion.
  - Backend: `register`.
  - Backend blockers: none.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `matrix_multiply_tetra`:
  - Category: matrix multiply.
  - Backend: `fallback`.
  - Backend blockers: `unsupported_control_flow`.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `hash_table_tetra`:
  - Category: hash table.
  - Backend: `fallback`.
  - Backend blockers: `unsupported_control_flow`.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `allocation_tetra`:
  - Category: allocation.
  - Backend: `register`.
  - Backend blockers: none.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `region_island_allocation_tetra`:
  - Category: region/island allocation.
  - Backend: `fallback`.
  - Backend blockers: `unsupported_effect_runtime_call`.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `json_parse_stringify_tetra`:
  - Category: JSON parse/stringify.
  - Backend: `fallback`.
  - Backend blockers:
    - `unsupported_aggregate_return`
    - `unsupported_call_abi`
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `http_plaintext_json_tetra`:
  - Category: HTTP plaintext/json.
  - Backend: `fallback`.
  - Backend blockers:
    - `unsupported_aggregate_return`
    - `unsupported_call_abi`
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `postgresql_single_multiple_update_tetra`:
  - Category: PostgreSQL ops.
  - Backend: `register`.
  - Backend blockers: none.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `actor_ping_pong_tetra`:
  - Category: actor ping-pong.
  - Backend: `fallback`.
  - Backend blockers: `unsupported_effect_runtime_call`.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `parallel_map_reduce_tetra`:
  - Category: parallel map/reduce.
  - Backend: `fallback`.
  - Backend blockers: `unsupported_call_abi`.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `startup_time_tetra`:
  - Category: startup time.
  - Backend: `register`.
  - Backend blockers: none.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `binary_size_tetra`:
  - Category: binary size.
  - Backend: `register`.
  - Backend blockers: none.
  - Heap allocs: `0`.
  - Bounds left: `0`.
- `compile_time_tetra`:
  - Category: compile time.
  - Backend: `register`.
  - Backend blockers: none.
  - Heap allocs: `0`.
  - Bounds left: `0`.

What changed compared to the older "5 heap rows" framing:

- the original five heap-positive rows are no longer heap-positive in the current report;
- the current heap task is no longer "make five rows zero";
- the current heap task is "keep all 17 rows zero with validator gates and exact lifetime evidence";
- the active implementation blocker is now mostly backend/ABI/runtime, not heap.

P72-specific truth:

- `p25.postgresql_single_multiple_update.frame_type_at` remains function-level `register`;
- `p25.postgresql_single_multiple_update.write_i32_be_at` is now `register`
  through `machine-ir-postgresql-inout-writer`;
- `p25.postgresql_single_multiple_update.write_i16_be_at` is now `register`
  through `machine-ir-postgresql-inout-writer`;
- `p25.postgresql_single_multiple_update.main` is now `register` through
  `machine-ir-postgresql-inout-writer-main`;
- PostgreSQL row-level `backend_path` is now `register`;
- PostgreSQL backend sidecar has `function_count=6`, `register_path=6`, and
  `stack_fallback=0`;
- global generic SysV/Win64 return slots were not widened; the exact
  PostgreSQL writer path records `return_slots=3` with
  `max_register_return_slots=1` and
  `multi_slot_return_policy=single_slot_register_return`;
- evidence is recorded in
  `.workflow/post-zero-heap-native-memory/verification/p72-postgresql-inout-writer-native-slice.md`.

## 3. Non-Claims

Do not claim any of these until separately proven:

- zero RSS;
- official TechEmpower result;
- cross-machine RSS comparability;
- production OS footprint;
- all possible Tetra programs are zero-heap;
- production actor memory budget/backpressure is complete;
- a row is native/register because the label changed;
- bounds-check elimination without proof IDs and negative tests;
- ABI support just because one helper was promoted;
- final completion while any track below is still missing evidence.

## 4. Execution Rules

Use small slices. One capability or one row family per packet.

Every implementation packet must follow this order:

1. inspect the current sidecars and source shape;
2. write RED tests for the exact missing behavior;
3. implement the smallest compiler/runtime change;
4. run targeted tests;
5. run a fresh local benchmark slice or full Tier 1 when row classification changes;
6. validate the report with `validate-local-benchmark-tier1`;
7. run `graphify update .` after code changes;
8. record evidence under `.workflow/post-zero-heap-native-memory/`;
9. keep status `PARTIAL` unless every final acceptance gate passes.

Forbidden shortcuts:

- editing report JSON to change outcomes;
- relabeling fallback as register without backend sidecar evidence;
- removing bounds checks without proof evidence;
- hiding heap by disabling telemetry;
- comparing RSS across machines as a language-level claim;
- using `GOCACHE=/tmp/...`;
- wiring GitHub Actions unless explicitly requested.

Persistent Go cache convention:

```sh
GOCACHE=$(pwd)/.cache/go-build-native-memory-<slug> go test ./...
GOCACHE=$(pwd)/.cache/go-build-native-memory-<slug> go clean -cache
```

## 5. Main Surfaces

Backend and fallback truth:

- `compiler/internal/buildreports/backend.go`
- `compiler/internal/buildreports/types.go`
- `compiler/internal/machine/machine_core.go`
- `compiler/internal/machine/machine_suite_test.go`
- `compiler/internal/backend/x64core/x64core_core.go`
- `compiler/internal/backend/x64core/x64core_suite_test.go`
- `compiler/internal/backend/x64/emitter.go`
- `compiler/internal/backend/x64/emitter_ext.go`
- `compiler/internal/backend/x64abi/abi.go`
- `compiler/internal/backend/x64abi/classifier.go`
- `compiler/compiler_reports.go`
- `tools/internal/localbenchmarktier1/classify.go`
- `tools/internal/localbenchmarktier1/metadata.go`

Bounds and proof truth:

- `compiler/internal/buildreports/bounds.go`
- `compiler/internal/lower/lower_expressions.go`
- `compiler/internal/lower/lower_suite_test.go`
- `compiler/internal/lower/rangeproof/rangeproof.go`
- `compiler/internal/rangeproof/rangeproof.go`
- `compiler/internal/machine/bounds/bounds_check_loops.go`
- `compiler/internal/backend/x64core/bounds/bounds_check_loops_register.go`

Heap/allocation truth:

- `compiler/internal/allocplan/plan.go`
- `compiler/internal/allocplan/report.go`
- `compiler/internal/allocplan/verify.go`
- `compiler/internal/allocplan/heap_reason_codes_test.go`
- `tools/internal/heaptelemetry/heaptelemetry.go`
- `tools/internal/localbenchmarktier1/metadata.go`
- `tools/cmd/validate-local-benchmark-tier1/evidence_validation.go`

Actor memory truth:

- `compiler/internal/actorsrt/actorsrt_core.go`
- `compiler/internal/actorsrt/actorsrt_suite_test.go`
- `compiler/internal/parallelrt/scheduler_model.go`
- `compiler/internal/parallelrt/per_core_scheduler.go`
- `compiler/internal/actorsafety/ownership_transfer.go`
- `compiler/internal/buildreports/actor_transfer.go`
- `tools/internal/heaptelemetry/heaptelemetry.go`
- `tools/internal/localbenchmarktier1/metadata.go`

RSS/runtime object truth:

- `tools/internal/rsstelemetry/rsstelemetry.go`
- `tools/internal/rsstelemetry/process_linux.go`
- `tools/internal/localbenchmarktier1/command.go`
- `compiler/internal/buildruntime/runtime_object.go`
- `compiler/internal/buildruntime/runtime_object_plan.go`
- `compiler/internal/buildruntime/runtime_usage.go`
- `compiler/internal/buildruntime/selection.go`
- `tools/cmd/validate-local-benchmark-tier1/main.go`

Benchmark and validation entry points:

- `tools/cmd/local-benchmark-tier1/main.go`
- `tools/cmd/validate-local-benchmark-tier1/main.go`
- P72 report:
  - Directory: `reports/benchmark-vnext-memory-baseline/tier1-after-postgresql-inout-writer-native`
  - File: `report.json`
- RSS budget policy:
  - Directory: `reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization`
  - File: `rss-budget-policy.local.json`

## 6. Phase 0 - Re-lock The Current Baseline

### Goal

Make the P72 report the explicit starting point before new implementation work.

### Steps

1. Validate the P72 report with the current RSS policy.
2. Extract a Tetra-only row inventory.
3. Confirm:
   - `17/17` Tetra rows are heap-clean;
   - `17/17` Tetra rows are bounds-clean;
   - exactly 8 Tetra rows remain fallback;
   - PostgreSQL row is `register` with no backend blockers;
   - JSON and HTTP rows still honestly retain `unsupported_aggregate_return`
     and `unsupported_call_abi`.
4. Save the inventory under `.workflow/post-zero-heap-native-memory/verification/`.
5. Update `.workflow/post-zero-heap-native-memory/GOAL.md` and `state.json`.

### Commands

```sh
REPORT_ROOT="reports/benchmark-vnext-memory-baseline"
P72_DIR="$REPORT_ROOT/tier1-after-postgresql-inout-writer-native"
POLICY_DIR="$REPORT_ROOT/tier1-after-memory-zero-heap-optimization"

GOCACHE=$(pwd)/.cache/go-build-p73-baseline \
  go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report "$P72_DIR/report.json" \
  --rss-budget-policy "$POLICY_DIR/rss-budget-policy.local.json"

jq -r '.results[] as $r | $r.rows[] | select(.language=="tetra") |
  [.category,.name,.status,(.tetra_metadata.backend_path // ""),
   ((.tetra_metadata.backend_blockers // [])|join(",")),
   (.tetra_metadata.heap_allocations // 0),
   ((.tetra_metadata.heap_reason_codes // [])|join(",")),
   (.tetra_metadata.bounds_left // 0)] | @tsv' \
  "$P72_DIR/report.json"

jq -r '[.results[].rows[] | select(.language=="tetra")] |
  {tetra_rows:length,
   zero_heap:map(select((.tetra_metadata.heap_allocations // 0)==0))|length,
   zero_bounds:map(select((.tetra_metadata.bounds_left // 0)==0))|length,
   fallback:map(select((.tetra_metadata.backend_path // "")=="fallback"))|length}' \
  "$P72_DIR/report.json"

GOCACHE=$(pwd)/.cache/go-build-p73-baseline go clean -cache
```

### Done When

- validator exits `0`;
- inventory is recorded;
- next packet starts from P72 truth, not from the older P69 PostgreSQL-fallback
  baseline.

## 7. Track A - Fallback Backend

### Track Goal

Move rows out of `fallback` only through real backend support.

Current fallback rows:

1. `json_parse_stringify_tetra`
   Blockers: `unsupported_aggregate_return`, `unsupported_call_abi`.
   Order: same `inout []u8`/multi-slot family as P72, smaller than HTTP.
2. `http_plaintext_json_tetra`
   Blockers: `unsupported_aggregate_return`, `unsupported_call_abi`.
   Order: same ABI family, two response writers plus `main`.
3. `hash_table_tetra`
   Blocker: `unsupported_control_flow`.
   Order: heap/bounds clean; lookup had prior progress; remaining `main` is composite.
4. `slice_sum_tetra`
   Blocker: `unsupported_control_flow`.
   Order: heap/bounds clean; loop and local storage composition.
5. `matrix_multiply_tetra`
   Blocker: `unsupported_control_flow`.
   Order: heap/bounds clean; nested loops and register pressure.
6. `region_island_allocation_tetra`
   Blocker: `unsupported_effect_runtime_call`.
   Order: needs runtime effect/domain primitive separation.
7. `actor_ping_pong_tetra`
   Blocker: `unsupported_effect_runtime_call`.
   Order: needs production actor runtime semantics.
8. `parallel_map_reduce_tetra`
   Blocker: `unsupported_call_abi`.
   Order: task spawn/join ABI and actor/task memory boundary.

### A0. Backend Truth Contract

**Goal:** reports cannot claim native/register unless backend sidecars prove it.

**Approach:**

1. Add or strengthen tests around backend report classification.
2. Require per-function evidence when a row is promoted through helper-local native support.
3. Keep row-level fallback if any called helper or `main` remains unsupported.
4. Report blockers must remain exact:
   - `unsupported_control_flow`;
   - `unsupported_effect_runtime_call`;
   - `unsupported_aggregate_return`;
   - `unsupported_call_abi`;
   - no vague "optimized" label without sidecar proof.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-backend-truth go test \
  ./compiler/internal/buildreports \
  ./compiler/internal/machine \
  ./compiler/internal/backend/x64core \
  ./compiler \
  -run 'Backend|Register|Fallback|PostgreSQL|Hash|Matrix|Slice|ABI' -count=1

GOCACHE=$(pwd)/.cache/go-build-backend-truth go clean -cache
```

**Done when:**

- every row promotion has backend sidecar evidence;
- partial function-level promotions do not silently promote the whole row.

### A1. PostgreSQL Writer ABI Discovery

**Completed packets:** `P70-postgresql-writer-abi-discovery`,
`P71-postgresql-inout-multislot-abi-design`.

**Goal:** decide whether `write_i32_be_at` or `write_i16_be_at` can be promoted without broad
multi-slot ABI support.

**Inspect:**

- Backend sidecar:
  `reports/benchmark-vnext-memory-baseline/`
  `tier1-after-postgresql-inout-writer-native/artifacts/bin/`
  `postgresql_single_multiple_update_tetra.backend.json`
- PLIR sidecar:
  `reports/benchmark-vnext-memory-baseline/`
  `tier1-after-postgresql-inout-writer-native/artifacts/bin/`
  `postgresql_single_multiple_update_tetra.plir.json`
- Source:
  `reports/benchmark-vnext-memory-baseline/`
  `tier1-after-postgresql-inout-writer-native/artifacts/src/p25/`
  `postgresql_single_multiple_update.tetra`
- `compiler/internal/lower/`
- `compiler/internal/backend/x64abi/`
- `compiler/internal/machine/`
- `compiler/internal/backend/x64core/`

**Questions:**

1. Is `return start + N` encoded as a multi-slot aggregate, or is the report classifier wrong?
2. What exact IR shape causes `return_slots=3`?
3. Is there already a scalar-return rewrite pattern that can preserve semantics?
4. Can one writer helper become register while the row honestly stays fallback?
5. What near-miss cases must remain fallback?

**Done when:**

- `.workflow/post-zero-heap-native-memory/results/P70-postgresql-writer-abi-discovery.md` exists;
- it names the exact P71 implementation slice or redirects to ABI design;
- no compiler/runtime code is edited by the read-only packet.

**P70/P71 result:** the writer helpers are real 3-slot shapes because lowered
IR returns scalar `Int` plus hidden `inout []u8` writeback slots. The safe first
implementation was narrowed to exact PostgreSQL writer helpers and row-local
`main`, without generic aggregate-return support and without widening global
SysV/Win64 return slots.

### A2. PostgreSQL Writer Implementation

**Status:** completed by P72.

**Implemented safe slice:**

- promote `p25.postgresql_single_multiple_update.write_i32_be_at`;
- promote `p25.postgresql_single_multiple_update.write_i16_be_at`;
- promote row-local `p25.postgresql_single_multiple_update.main`;
- keep JSON/HTTP aggregate-return rows fallback;
- make no broad aggregate-return ABI claim;
- do not widen generic SysV/Win64 `MaxRetSlots`.

**Approach:**

1. Add RED tests in `compiler/internal/machine/machine_suite_test.go` for the exact writer shape.
2. Add RED tests in `compiler/internal/backend/x64core/x64core_suite_test.go` for register emission.
3. Add report tests in `compiler/compiler_suite_test.go` or backend report tests.
4. Implement scalar-return lowering if the helper is semantically scalar.
5. If the IR truly requires multi-slot aggregate return, stop and do A3 instead.
6. Run a fresh Tier 1 report.

**Verification:**

```sh
REPORT_ROOT="reports/benchmark-vnext-memory-baseline"
PG_WRITER_DIR="$REPORT_ROOT/tier1-after-postgresql-writer-native"
POLICY_DIR="$REPORT_ROOT/tier1-after-memory-zero-heap-optimization"

GOCACHE=$(pwd)/.cache/go-build-pg-writer go test \
  ./compiler/internal/machine \
  ./compiler/internal/backend/x64core \
  ./compiler/internal/backend/linux_x64 \
  ./compiler \
  -run 'PostgreSQL|Writer|WriteI32|WriteI16|Aggregate|CallABI|Backend' -count=1

GOCACHE=$(pwd)/.cache/go-build-pg-writer go run ./tools/cmd/local-benchmark-tier1 \
  --out-dir "$PG_WRITER_DIR" \
  --iterations 3

GOCACHE=$(pwd)/.cache/go-build-pg-writer \
  go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report "$PG_WRITER_DIR/report.json" \
  --rss-budget-policy "$POLICY_DIR/rss-budget-policy.local.json"

graphify update .
git diff --check
GOCACHE=$(pwd)/.cache/go-build-pg-writer go clean -cache
```

**Done when:**

- both targeted writer helper sidecars show `backend_path=register`;
- PostgreSQL row-level `backend_path` becomes `register` only after every
  PostgreSQL function in the row has sidecar proof;
- heap and bounds stay `0`;
- no unrelated JSON/HTTP/actor behavior changes.

**P72 result:** fresh report
`reports/benchmark-vnext-memory-baseline/tier1-after-postgresql-inout-writer-native/report.json`
shows `postgresql_single_multiple_update_tetra.backend_path=register`,
`heap_allocations=0`, and `bounds_left=0`. PostgreSQL backend sidecar shows
`function_count=6`, `register_path=6`, `stack_fallback=0`. Evidence:
`.workflow/post-zero-heap-native-memory/verification/p72-postgresql-inout-writer-native-slice.md`.

### A3. Aggregate Return And Call ABI Design

**Goal:** make JSON/HTTP fallback blockers represent real ABI support, not
benchmark exceptions. PostgreSQL is now closed by P72 for the exact benchmark
row and should become a template only after separate read-only design proves
which parts generalize safely.

**Required ABI concepts:**

- scalar return in register;
- small aggregate return if the target ABI supports it;
- caller-provided output buffer for writer/string/slice results;
- explicit ownership for returned strings/slices/views;
- `ret_slots` and `arg_slots` limits in reports;
- exact blocker when ABI is still unsupported.

**Approach:**

1. Write design notes in workflow result or a focused doc before broad implementation.
2. Add classifier tests in `compiler/internal/backend/x64abi/abi_test.go`.
3. Add lowering tests for calls returning:
   - scalar;
   - fixed small aggregate;
   - caller-output buffer;
   - borrowed view;
   - owned region/request-domain value.
4. Implement only one ABI shape at a time.
5. Keep JSON/HTTP and any future ABI row fallback until all called functions
   and `main` are covered.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-abi-design go test \
  ./compiler/internal/backend/x64abi \
  ./compiler/internal/lower \
  ./compiler/internal/machine \
  ./compiler/internal/backend/x64core \
  ./compiler \
  -run 'ABI|Aggregate|Call|Return|Writer|JSON|HTTP|PostgreSQL' -count=1

GOCACHE=$(pwd)/.cache/go-build-abi-design go clean -cache
```

**Done when:**

- unsupported ABI blockers decrease because supported ABI shapes exist;
- near-miss unsupported ABI shapes still report precise blockers;
- no row is promoted by string matching alone.

### A4. JSON And HTTP Backend Tail

**Goal:** after A3, promote JSON and HTTP helpers through real ABI/lifetime support.

**Approach:**

1. Start with JSON because it is smaller than HTTP.
2. Promote exactly one helper or writer shape per packet.
3. For HTTP, handle plaintext and JSON writers separately if their IR differs.
4. Keep request/response domain memory separate from heap telemetry.
5. Run fresh Tier 1 after each row-level blocker changes.

**Done when:**

- JSON/HTTP row fallback blockers are either gone or exact;
- heap stays `0`;
- bounds stay `0`;
- sidecars show which helper/main functions are still fallback.

### A5. Control-Flow Native Backend

**Rows:** `hash_table_tetra`, `slice_sum_tetra`, `matrix_multiply_tetra`.

**Goal:** support loop/indexed-memory control-flow shapes without benchmark-only hacks.

**Implementation order:**

1. `hash_table_tetra.main`
   - current row is heap/bounds clean;
   - lookup helper already had previous native progress;
   - remaining blocker is composite `main` control flow.
2. `slice_sum_tetra.main`
   - local storage + fill loop + repeated sum loop;
   - needs no-alias/vector blocker to stay separate from backend blocker.
3. `matrix_multiply_tetra.main`
   - nested loops, affine indices, register pressure;
   - do last among these three.

**Approach:**

1. Inspect backend sidecar for the exact unsupported IR kind.
2. Add machine IR recognizer tests.
3. Add x64core emitter tests.
4. Add negative tests for near-miss loops:
   - unknown trip count;
   - unsafe index;
   - aliasing not proven;
   - unsupported nested break/continue;
   - unsupported call inside loop.
5. Implement one recognized loop family.
6. Keep vectorization and no-alias optimization as separate perf work.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-control-flow-native go test \
  ./compiler/internal/machine \
  ./compiler/internal/backend/x64core \
  ./compiler/internal/buildreports \
  ./compiler \
  -run 'Hash|SliceSum|Matrix|ControlFlow|Loop|Indexed|Backend' -count=1

GOCACHE=$(pwd)/.cache/go-build-control-flow-native go clean -cache
```

**Done when:**

- a target row leaves fallback with backend sidecar proof;
- semantic output stays correct;
- no near-miss unsafe control-flow shape is accidentally promoted.

### A6. Region/Island Runtime Effect Backend

**Row:** `region_island_allocation_tetra`.

**Goal:** replace generic `unsupported_effect_runtime_call` with either native/domain primitive
support or a precise runtime blocker.

**Approach:**

1. Inspect the row backend sidecar and lowered IR.
2. Separate these cases:
   - local region allocation that can lower to stack/domain storage;
   - island/domain accounting that needs runtime support;
   - real effectful runtime call that must remain fallback.
3. Add classifier tests so stack/domain primitives are not reported as generic runtime calls.
4. If safe, lower one exact primitive to native/domain operation.
5. Keep domain bytes visible in memory evidence.

**Done when:**

- generic blocker is gone or replaced with a precise blocker;
- heap and bounds remain `0`;
- domain accounting remains honest.

### A7. Actor And Parallel Backend Tail

**Rows:** `actor_ping_pong_tetra`, `parallel_map_reduce_tetra`.

**Goal:** do not promote actor/task rows until production actor memory and call ABI are real.

**Approach:**

1. Finish Track D actor byte budget/backpressure first.
2. Inspect actor runtime calls:
   - `__tetra_actor_recv`;
   - `__tetra_actor_spawn`;
   - task spawn/join helpers.
3. Define supported runtime-call ABI shapes.
4. Add tests for actor/task call lowering and fallback blockers.
5. Promote only exact runtime-call shapes.

**Done when:**

- actor/task rows leave fallback only with production runtime evidence;
- memory/domain evidence remains present;
- task/actor copied/moved bytes stay visible.

## 8. Track B - Bounds-Check Elimination

### Current State

All 17 Tetra rows currently have `bounds_left == 0`.

So this track is now a stability track, not a "remove remaining checks" track.

### B1. Bounds Regression Gate

**Goal:** prevent future packets from reintroducing bounds checks into Tier 1 rows.

**Approach:**

1. Extend or add validator expectations for the current Tier 1 Tetra rows.
2. For each row, require:
   - `bounds_left == 0`;
   - bounds report exists;
   - eliminated checks have proof evidence where applicable.
3. If a future row intentionally needs a check, it must be recorded as an exact exception.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-bounds-gate go test \
  ./tools/cmd/validate-local-benchmark-tier1 ./tools/internal/localbenchmarktier1 \
  -run 'Bounds|Proof|Evidence|Validate' -count=1

GOCACHE=$(pwd)/.cache/go-build-bounds-gate go clean -cache
```

**Done when:**

- validator fails on a fixture/report where any current Tier 1 Tetra row regresses to
  `bounds_left > 0`;
- accepted exceptions require explicit reason codes.

### B2. Proof ID Hygiene

**Goal:** make zero bounds auditable.

**Approach:**

1. Confirm bounds reports include enough proof IDs or proof metadata.
2. Add negative tests where proof must fail.
3. Keep known proof families separate:
   - constant length;
   - modulo capacity;
   - affine nested loop;
   - call-boundary length contract;
   - helper offset contract.

**Done when:**

- zero bounds is explainable by proofs, not by removed instrumentation.

### B3. Future Backend Work Must Preserve BCE

**Goal:** backend promotion must not reintroduce checked paths.

**Approach:**

1. Each Track A worker must compare before/after `bounds_left`.
2. If a row leaves fallback but `bounds_left` increases, the worker is incomplete.
3. Row-level native support must preserve the existing bounds proof contract.

**Done when:**

- every native/backend packet includes a bounds invariant in its verification artifact.

## 9. Track C - Heap Zero-Regression And Lifetime Evidence

### Current State

All 17 Tetra rows currently have `heap_allocations == 0`.

The old five heap rows are now closed at benchmark-report level:

| Original row | Old reason | Current state |
| --- | --- | --- |
| `slice_sum_tetra` | `heap.required_large_object` | zero heap |
| `bounds_check_loops_tetra` | `heap.required_large_object` | zero heap |
| `json_parse_stringify_tetra` | `heap.required_unknown_call` | zero heap |
| `http_plaintext_json_tetra` | `heap.required_unknown_call` | zero heap |
| `postgresql_single_multiple_update_tetra` | `heap.required_unknown_call` | zero heap |

### C1. Zero-Heap Validator Gate

**Goal:** make zero heap a regression gate for current Tier 1 Tetra rows.

**Approach:**

1. Add validator fixtures:
   - one report with a Tetra row at `heap_allocations > 0`;
   - one report with `heap.required_unknown_call`;
   - one report missing runtime heap evidence.
2. Validator should reject these unless an explicit exception policy is supplied.
3. Keep the gate local to benchmark claims; do not claim all possible programs are zero-heap.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-zero-heap-gate go test \
  ./tools/cmd/validate-local-benchmark-tier1 \
  ./tools/internal/heaptelemetry \
  ./tools/internal/localbenchmarktier1 \
  -run 'Heap|ZeroHeap|MemoryEvidence|UnknownCall|Validate' -count=1

GOCACHE=$(pwd)/.cache/go-build-zero-heap-gate go clean -cache
```

**Done when:**

- current zero heap is enforced by validator tests;
- missing heap sidecars fail validation;
- unknown call heap reason cannot silently return.

### C2. Lifetime Summary Stability

**Goal:** keep JSON/HTTP/PostgreSQL zero heap tied to call/lifetime facts.

**Approach:**

1. Ensure helper summaries distinguish:
   - no allocation;
   - borrowed input view;
   - caller-owned output slot;
   - request/domain-owned buffer;
   - true heap escape.
2. Tests must prove near-miss unknown calls still require heap or exact blockers.
3. ABI work in Track A must not erase lifetime evidence.

**Done when:**

- helper lifetime facts survive backend/ABI changes;
- heap remains `0` because evidence says no heap, not because reporting is muted.

### C3. Large Local Storage Stability

**Rows:** `slice_sum_tetra`, `bounds_check_loops_tetra`.

**Goal:** keep large local arrays on stack/region/local domain when no escape is proven.

**Approach:**

1. Keep allocplan tests for large no-escape `i32` local arrays.
2. Keep lowerer tests proving no `IRMakeSliceI32` heap path appears.
3. Add negative tests:
   - escaping slice;
   - unknown length;
   - address stored globally;
   - returned borrowed view with invalid lifetime.

**Done when:**

- old `heap.required_large_object` rows cannot regress silently.

## 10. Track D - Production Actor Memory

### Track Goal

Actor memory must be real production runtime behavior:

- per-actor byte counters;
- mailbox/message byte accounting;
- owned region accounting;
- copied/moved byte accounting;
- byte budgets;
- byte-based backpressure;
- Tier 1 sidecar evidence and validator checks.

Current limitation:

- actor row has runtime-measured domain evidence;
- production byte budget and byte backpressure are not complete.

### D1. Runtime Actor Memory Domain

**Goal:** actor runtime owns byte counters directly.

**Fields to support:**

- actor id;
- mailbox current bytes;
- mailbox peak bytes;
- message slab current bytes;
- message slab peak bytes;
- owned region current bytes;
- owned region peak bytes;
- bytes copied;
- bytes moved zero-copy;
- budget bytes;
- over-budget count;
- backpressure events.

**Approach:**

1. Add or extend runtime structs in `compiler/internal/actorsrt/actorsrt_core.go`.
2. Add tests in `compiler/internal/actorsrt/actorsrt_suite_test.go`.
3. Keep counters independent from benchmark tooling.
4. Ensure zero-message actor starts at zero dynamic actor bytes.

**Done when:**

- actor runtime tests can read current/peak counters without Tier 1 tooling.

### D2. Mailbox And Message Byte Accounting

**Goal:** send/receive changes bytes correctly.

**Approach:**

1. Define message size calculation.
2. On send:
   - receiver mailbox bytes increase;
   - copied bytes increase if payload is copied;
   - moved bytes increase if ownership transfers.
3. On receive:
   - mailbox bytes decrease;
   - active/owned bytes update if the actor retains payload.
4. On drop:
   - owned bytes decrease.
5. Add tests for current and peak behavior.

**Done when:**

- actor tests prove send, receive, drop, copy, and move byte accounting.

### D3. Actor Byte Budget And Backpressure

**Goal:** actor pressure is based on bytes, not only message count.

**Approach:**

1. Add per-actor budget configuration.
2. Define send behavior when budget would be exceeded:
   - wait/yield;
   - reject;
   - or return explicit backpressure status.
3. Record backpressure events.
4. Receiving messages must free budget.
5. Zero-copy move must change owner without copy count.

**Done when:**

- tests prove under-budget send succeeds;
- over-budget send triggers the designed behavior;
- backpressure counters are visible;
- receiving frees budget.

### D4. Tier 1 Actor Evidence

**Goal:** benchmark report uses production actor runtime sidecars.

**Approach:**

1. Extend heap/domain sidecar schema if needed.
2. Ingest actor fields in `tools/internal/localbenchmarktier1/metadata.go`.
3. Add validator checks for actor row:
   - actor domain evidence exists;
   - mailbox/message bytes exist;
   - budget/backpressure fields exist;
   - copied/moved bytes exist.
4. Keep the claim local: this is memory evidence, not parallel performance proof.

**Verification:**

```sh
REPORT_ROOT="reports/benchmark-vnext-memory-baseline"
ACTOR_DIR="$REPORT_ROOT/tier1-after-actor-byte-budget"
POLICY_DIR="$REPORT_ROOT/tier1-after-memory-zero-heap-optimization"

GOCACHE=$(pwd)/.cache/go-build-actor-memory go test \
  ./compiler/internal/actorsrt \
  ./compiler/internal/parallelrt \
  ./tools/internal/heaptelemetry \
  ./tools/internal/localbenchmarktier1 \
  ./tools/cmd/validate-local-benchmark-tier1 \
  -run 'Actor|Mailbox|Message|Budget|Backpressure|Domain|Bytes' -count=1

GOCACHE=$(pwd)/.cache/go-build-actor-memory go run ./tools/cmd/local-benchmark-tier1 \
  --out-dir "$ACTOR_DIR" \
  --iterations 3

GOCACHE=$(pwd)/.cache/go-build-actor-memory \
  go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report "$ACTOR_DIR/report.json" \
  --rss-budget-policy "$POLICY_DIR/rss-budget-policy.local.json"

graphify update .
GOCACHE=$(pwd)/.cache/go-build-actor-memory go clean -cache
```

**Done when:**

- `actor_ping_pong_tetra` has production actor memory evidence;
- validator fails if actor byte evidence disappears;
- fallback/native status stays honest.

## 11. Track E - RSS Reduction

### Track Goal

RSS must become smaller where feasible and guarded locally.

RSS includes:

- executable code;
- dynamic loader;
- runtime object;
- libc/syscall wrappers;
- stack pages;
- mapped telemetry/runtime pages;
- page rounding;
- optional actor/http/postgres pieces if linked.

No honest plan should claim "RSS in bytes" for a Linux process that still uses loader/code/stack.
The correct goal is lower local RSS and regression gates.

### E1. RSS Breakdown Audit

**Goal:** know what creates the RSS floor for tiny rows.

**Approach:**

1. For tiny rows, record:
   - binary size;
   - linked runtime features;
   - initialized runtime features;
   - RSS current;
   - RSS peak;
   - heap peak;
   - runtime object plan.
2. Use existing sidecars first.
3. Do not claim exact RSS component attribution unless measured.
4. Save audit under `.workflow/post-zero-heap-native-memory/verification/`.

**Done when:**

- each RSS optimization target is based on evidence, not a guess.

### E2. Minimal Runtime Object Linking

**Goal:** scalar rows should not link unused actor/http/postgres/runtime pieces.

**Approach:**

1. Inspect `runtime_features_required`, `runtime_features_linked`, and runtime object plans.
2. Add tests in `compiler/internal/buildruntime/tests/`.
3. Split optional runtime pieces behind feature selection:
   - actor runtime;
   - HTTP runtime;
   - PostgreSQL runtime;
   - telemetry runtime;
   - domain telemetry.
4. Ensure heap/RSS telemetry does not force unrelated runtime pieces.

**Done when:**

- tiny scalar rows prove minimal runtime object linking;
- no required telemetry disappears.

### E3. Startup Footprint

**Goal:** reduce startup RSS for tiny programs.

**Approach:**

1. Avoid eager initialization for unused runtime features.
2. Avoid large static buffers in generic startup.
3. Lazy-init actor/http/postgres pieces.
4. Keep telemetry scoped to benchmark/evidence builds.
5. Measure after each change with fresh Tier 1.

**Done when:**

- startup/binary-size rows show stable or lower local RSS;
- report still carries heap/RSS evidence.

### E4. Row-Specific RSS Budget Policy

**Goal:** local validator fails on RSS regressions.

**Approach:**

1. Use host-pinned policy only.
2. Keep budgets per row, not one global number.
3. Include tolerance bands for local noise.
4. Validator must say the policy is local, not cross-machine.
5. Store policy beside the fresh report.

**Verification:**

```sh
REPORT_ROOT="reports/benchmark-vnext-memory-baseline"
RSS_DIR="$REPORT_ROOT/tier1-after-rss-runtime-object-reduction"

GOCACHE=$(pwd)/.cache/go-build-rss-reduction go test \
  ./tools/internal/rsstelemetry \
  ./tools/internal/localbenchmarktier1 \
  ./tools/cmd/validate-local-benchmark-tier1 \
  ./compiler/internal/buildruntime/... \
  -run 'RSS|RuntimeObject|Feature|Budget|Policy|Startup' -count=1

GOCACHE=$(pwd)/.cache/go-build-rss-reduction go run ./tools/cmd/local-benchmark-tier1 \
  --out-dir "$RSS_DIR" \
  --iterations 5

GOCACHE=$(pwd)/.cache/go-build-rss-reduction \
  go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report "$RSS_DIR/report.json" \
  --rss-budget-policy "$RSS_DIR/rss-budget-policy.local.json"

graphify update .
GOCACHE=$(pwd)/.cache/go-build-rss-reduction go clean -cache
```

**Done when:**

- RSS policy is row-specific;
- validator enforces it;
- final report avoids cross-machine RSS claims.

## 12. Track F - Benchmark And Evidence Discipline

### F1. Fresh Report After Every Classification Change

Any change to these fields requires a fresh report:

- `backend_path`;
- `backend_blockers`;
- `heap_allocations`;
- `heap_reason_codes`;
- `bounds_left`;
- actor domain evidence;
- RSS policy fields.

### F2. Required Evidence For Every Worker Packet

Each worker result must include:

- changed files;
- exact target row/function;
- RED tests added;
- commands run;
- fresh report path if classification changed;
- validator result;
- sidecar snippets or summary;
- non-targets;
- remaining blockers;
- Graphify update result after code changes.

### F3. Dirty Worktree Rule

The repo currently has many unrelated dirty and untracked files. Do not revert unrelated changes.
Packet write scopes must be narrow. If a required file has unrelated edits, inspect and work with
the current content.

## 13. Work Packets From Here

### P70 - PostgreSQL Writer ABI Discovery

Owner: read-only `explorer` or `explorer_fast`.

Goal:

- inspect writer helper IR/report shape;
- decide whether P71 can safely promote one writer helper;
- produce exact write scope and RED tests.

Output:

- `.workflow/post-zero-heap-native-memory/results/P70-postgresql-writer-abi-discovery.md`

Done when:

- recommendation is exact;
- no code edits.

### P71 - PostgreSQL Writer ABI Design

Status: completed as read-only design.

Goal:

- define the smallest honest ABI contract for PostgreSQL `inout []u8`
  writer helpers.

Done when:

- design rejects generic aggregate-return widening;
- P72 worker scope is exact and bounded.

### P72 - Exact PostgreSQL Inout Writer Native Slice

Status: completed by `P72-postgresql-inout-writer-native-slice`.

Goal:

- prove the exact PostgreSQL `inout []u8` writer ABI path without generic ABI
  widening.

Done when:

- PostgreSQL writer helper and row-local `main` backend rows are `register`;
- JSON/HTTP aggregate-return rows remain fallback;
- near-miss unsupported shapes remain fallback;
- fresh Tier 1 validates.

### P73 - Post-P72 ABI/Fallback Discovery

Owner: read-only `explorer` or `explorer_fast`.

Goal:

- start from the P72 report and decide the next smallest safe backend packet:
  JSON exact writer, HTTP exact writer, or a non-ABI control-flow row.

Done when:

- `.workflow/post-zero-heap-native-memory/results/P73-post-p72-fallback-abi-discovery.md`
  ranks the eight remaining fallback rows;
- it names one safe P74 worker slice or records why no worker slice is safe;
- no compiler/runtime code is edited by the read-only packet.

### P74 - Next Backend Worker Slice

Owner: `worker`.

Goal:

- implement only the safe slice named by P73.

Done when:

- the targeted row/function backend blocker decreases with sidecar proof;
- request-domain memory stays honest.

### P75 - Control-Flow Native Row

Owner: read-only discovery, then `worker`.

Goal:

- choose one of hash/slice/matrix and implement one exact control-flow backend family.

Done when:

- one row leaves fallback through real native/register support;
- no bounds/heap regression.

### P76 - Region/Island Effect Primitive

Owner: read-only discovery, then `worker`.

Goal:

- separate domain primitives from real runtime calls.

Done when:

- `region_island_allocation_tetra` blocker becomes precise or row moves forward with evidence.

### P77 - Production Actor Byte Budget

Owner: `worker`.

Goal:

- implement runtime actor budget/backpressure and benchmark ingestion.

Done when:

- actor tests and Tier 1 actor evidence prove mailbox/message bytes and backpressure.

### P78 - RSS Runtime Object Reduction

Owner: discovery, then `worker`.

Goal:

- reduce linked runtime pieces and enforce row-specific local RSS budgets.

Done when:

- fresh report validates with local RSS policy;
- tiny rows show no unrelated runtime pieces linked.

### P79 - Final Integrated Gate

Owner: coordinator.

Goal:

- produce final report and final audit.

Done when:

- all acceptance criteria below pass;
- otherwise status remains `PARTIAL` with exact remaining blockers.

## 14. Final Integrated Gate

Run targeted commands first for changed areas. Then run the integrated gate.

```sh
REPORT_ROOT="reports/benchmark-vnext-memory-baseline"
FINAL_DIR="$REPORT_ROOT/tier1-native-memory-final"

GOCACHE=$(pwd)/.cache/go-build-native-memory-final go test \
  ./compiler/internal/allocplan \
  ./compiler/internal/lower \
  ./compiler/internal/buildreports \
  ./compiler/internal/backend/x64abi \
  ./compiler/internal/backend/x64core \
  ./compiler/internal/machine \
  ./compiler/internal/actorsrt \
  ./compiler/internal/buildruntime/... \
  ./tools/internal/heaptelemetry \
  ./tools/internal/rsstelemetry \
  ./tools/internal/localbenchmarktier1 \
  ./tools/cmd/local-benchmark-tier1 \
  ./tools/cmd/validate-local-benchmark-tier1 \
  -count=1

GOCACHE=$(pwd)/.cache/go-build-native-memory-final \
  go run ./tools/cmd/local-benchmark-tier1 \
  --out-dir "$FINAL_DIR" \
  --iterations 5

GOCACHE=$(pwd)/.cache/go-build-native-memory-final \
  go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report "$FINAL_DIR/report.json" \
  --rss-budget-policy "$FINAL_DIR/rss-budget-policy.local.json"

graphify update .
git diff --check
GOCACHE=$(pwd)/.cache/go-build-native-memory-final go clean -cache
```

If `tier1-native-memory-final/rss-budget-policy.local.json` is not generated yet, validate against
the latest approved local policy and keep RSS completion `PARTIAL`.

## 15. Final Acceptance Criteria

Final status may be `DONE` only when every item is true:

- P72 or later baseline is locked and recorded;
- all current Tier 1 Tetra rows remain `heap_allocations == 0`;
- all current Tier 1 Tetra rows remain `bounds_left == 0`;
- zero-heap and zero-bounds are enforced by validator tests;
- fallback rows targeted by this plan leave fallback only through sidecar-proven backend support;
- remaining fallback rows, if any, have exact blockers and keep global status `PARTIAL`;
- JSON/HTTP ABI blockers are resolved or explicitly blocked by exact ABI
  evidence, and PostgreSQL remains register/native in fresh sidecars;
- actor memory has production mailbox/message bytes, copied/moved bytes, budget, and backpressure
  evidence;
- RSS has row-specific host-pinned local budget policy;
- fresh final Tier 1 report validates;
- `graphify update .` ran after code changes;
- `git diff --check` passes;
- workflow kernel records final evidence;
- final audit states non-claims clearly.

## 16. Best Implementation Strategy

Best path from the P72 baseline:

1. Start P73 as read-only discovery from the fresh P72 report.
2. Pick the smallest safe next backend slice instead of assuming P72 generalizes
   to JSON/HTTP.
3. Keep heap and bounds as regression gates, not active optimization claims.
4. Delay actor/parallel backend promotion until production actor byte budgets exist.
5. Delay RSS claims until runtime object audit proves what actually changed.
6. Prefer exact blockers over optimistic labels.

This keeps the system clean: fewer benchmark hacks, more reusable compiler/runtime facts, and no
claim that outruns the evidence.
