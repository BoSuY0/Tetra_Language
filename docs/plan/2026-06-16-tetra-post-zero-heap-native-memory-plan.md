# План Tetra після Zero-Heap: native backend, bounds, heap, actor memory, RSS

**Статус:** план реалізації, не доказ готової реалізації.  
**Дата:** 2026-06-16.  
**Обсяг:** fallback backend, bounds-check elimination, 5 heap rows, production actor memory, RSS reduction.  
**Базовий звіт:** `reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/report.json`.  
**Базовий аудит:** `docs/audits/tetra-memory-zero-heap-optimization-final.md`.

## 1. Мета

Зараз Tetra вже має чесну memory evidence базу: heap і RSS міряються, багато
простих рядків уже мають `0` heap. Але головні блокери тепер інші:

- багато рядків все ще йдуть через `fallback`, а не через нормальний
  register/native backend;
- частина рядків все ще має bounds checks;
- 5 рядків все ще мають runtime heap allocation;
- actor memory domains поки є як model/local evidence, але не як production
  runtime evidence;
- RSS міряється, але його ще треба зменшувати й ставити regression gates.

Фінальна ціль цього плану:

- fallback rows переводяться в реальний register/native backend;
- bounds checks прибираються тільки з proof evidence;
- 5 heap rows стають zero-heap або чесно залишаються blocked з exact escape
  evidence;
- actor memory domains міряються production runtime-ом;
- RSS залишається окремою метрикою від heap і має local regression gate;
- кожна заява підтверджена тестом, report, sidecar або validator.

Це не "нова memory model". Це наступний шар реалізації поверх уже існуючої
Tetra Memory Model.

## 2. Поточна база фактів

Поточний baseline:

```text
generated_at: 2026-06-16T16:18:13Z
git_commit: 95bfd4a887bab5032437cb22494d034e82ae6d35
host: linux/amd64
cpu: Intel(R) Core(TM) i9-14900HX
iterations: 5
categories: 17
rows: 68
Tetra rows: 17 measured
Tetra zero-heap rows: 12
Tetra heap rows: 5
```

Поточна таблиця Tetra rows:

| Category | Row | Backend path | Main blocker | Heap | Bounds left |
| --- | --- | --- | --- | ---: | ---: |
| integer loops | `integer_loops_tetra` | fallback | `unsupported_control_flow` | 0 | 0 |
| slice sum | `slice_sum_tetra` | fallback | `unsupported_effect_runtime_call` + bounds | 16384 B / 1 alloc | 1 |
| bounds-check loops | `bounds_check_loops_tetra` | fallback | `unsupported_effect_runtime_call` + bounds | 16384 B / 1 alloc | 2 |
| function calls | `function_calls_tetra` | register | немає в поточному report | 0 | 0 |
| recursion | `recursion_tetra` | fallback | `unsupported_control_flow` | 0 | 0 |
| matrix multiply | `matrix_multiply_tetra` | fallback | `unsupported_effect_runtime_call` + bounds | 0 | 7 |
| hash table | `hash_table_tetra` | fallback | `unsupported_control_flow`, `unsupported_effect_runtime_call` | 0 | 4 |
| allocation | `allocation_tetra` | fallback | `unsupported_effect_runtime_call` | 0 | 2 |
| region/island allocation | `region_island_allocation_tetra` | fallback | `unsupported_effect_runtime_call` | 0 | 2 |
| JSON parse/stringify | `json_parse_stringify_tetra` | fallback | `unsupported_aggregate_return`, `unsupported_call_abi` | 128 B / 1 alloc | 27 |
| HTTP plaintext/json | `http_plaintext_json_tetra` | fallback | `unsupported_aggregate_return`, `unsupported_call_abi` | 384 B / 2 allocs | 45 |
| PostgreSQL single/multiple/update | `postgresql_single_multiple_update_tetra` | fallback | `stack_fallback`, `unsupported_aggregate_return`, `unsupported_call_abi` | 64 B / 1 alloc | 8 |
| actor ping-pong | `actor_ping_pong_tetra` | fallback | actor/runtime limitation, `unsupported_effect_runtime_call` | 0 | 0 |
| parallel map/reduce | `parallel_map_reduce_tetra` | fallback | actor/runtime limitation, `unsupported_call_abi` | 0 | 0 |
| startup time | `startup_time_tetra` | register | немає в поточному report | 0 | 0 |
| binary size | `binary_size_tetra` | register | немає в поточному report | 0 | 0 |
| compile time | `compile_time_tetra` | fallback | `unsupported_control_flow` | 0 | 0 |

Поточний memory стан:

- `heap_alloc_bytes` уже runtime-measured для Tetra rows через
  `tetra.runtime.heap_telemetry.v1`;
- `rss_current` і `rss_peak` уже runtime-measured через
  `tetra.local_benchmark.process_rss_telemetry.v1`;
- `domain_bytes` поки здебільшого `allocation_report_estimate` або
  `unsupported`;
- actor domain bytes ще не production runtime sidecar evidence;
- RSS peak зараз приблизно 11-15 MiB на цьому host, навіть коли heap дорівнює
  `0`, бо RSS включає loader/code/runtime/stack/mapped pages.

## 3. Що не можна claim-ити

Цей план не дає права заявляти:

- `zero RSS`;
- official TechEmpower result;
- cross-machine RSS comparability;
- Linux RSS як semantics мови Tetra;
- production OS footprint;
- zero heap для будь-якої можливої програми;
- production actor memory без production `actorsrt` sidecar evidence;
- bounds-check elimination без proof IDs;
- native backend тільки тому, що label у report змінився.

## 4. Правила виконання

- Для code changes використовувати TDD.
- Спочатку RED test, потім implementation, потім GREEN.
- Fallback row вважається закритим тільки коли є реальний native/register path,
  а не просто змінений report label.
- Bounds check можна прибрати тільки з proof evidence.
- Heap, RSS і domain bytes не змішувати.
- Для Go verification використовувати persistent cache під `.cache/go-build-*`.
- Не ставити `GOCACHE` у `/tmp`.
- Після code changes запускати `graphify update .`.
- GitHub Actions не підключати без окремого дозволу.
- Не казати `DONE`, доки tests, benchmark report, validators і sidecars не
  збігаються.

## 5. Основні code surfaces

Fallback backend:

- `compiler/internal/buildreports/backend.go`
- `compiler/internal/machine/`
- `compiler/internal/backend/x64core/`
- `compiler/internal/backend/x64/`
- `compiler/internal/backend/x64abi/`
- `compiler/reports_backend.go`
- `compiler/reports_emit.go`
- `tools/cmd/local-benchmark-tier1/metadata.go`
- `tools/cmd/local-benchmark-tier1/classify.go`

Bounds-check elimination:

- `compiler/internal/lower/lower_rangeproof.go`
- `compiler/internal/lower/rangeproof/`
- `compiler/internal/rangeproof/`
- `compiler/internal/lower/proof_bce_test.go`
- `compiler/internal/validation/validation_translation.go`
- `compiler/reports_bounds.go`
- `tools/cmd/local-benchmark-tier1/metadata.go`

Heap/allocation lowering:

- `compiler/internal/allocplan/plan.go`
- `compiler/internal/allocplan/heap_reason_codes_test.go`
- `compiler/internal/validation/validation_allocation_lifetimes.go`
- `compiler/internal/memoryfacts/from_plir_allocplan.go`
- `compiler/internal/backend/x64core/emit_heap.go`
- `compiler/internal/backend/x64core/heap_telemetry.go`
- `docs/design/allocation_planner_lowering.md`
- `docs/design/runtime_allocation_contract.md`
- `docs/design/storage_classes.md`

Actor memory:

- `compiler/internal/actorsrt/linux_x64.go`
- `compiler/internal/actorsrt/linux_x64_emit.go`
- `compiler/internal/actorsrt/linux_x64_emit_send.go`
- `compiler/internal/actorsrt/linux_x64_emit_surface_recv.go`
- `compiler/internal/actorsrt/production_boundary.go`
- `compiler/internal/parallelrt/scheduler_model.go`
- `tools/internal/heaptelemetry/heaptelemetry.go`
- `tools/internal/ramvalidate/ramvalidate.go`
- `tools/cmd/local-benchmark-tier1/metadata.go`

RSS:

- `tools/internal/rsstelemetry/rsstelemetry.go`
- `tools/cmd/local-benchmark-tier1/command.go`
- `tools/cmd/local-benchmark-tier1/rss_linux.go`
- `docs/spec/process_rss_telemetry.md`
- `docs/spec/local_rss_budget_policy.md`
- `compiler/internal/buildruntime/runtime_object_plan.go`
- `compiler/internal/buildruntime/runtime_object.go`
- `compiler/reports_runtime_object.go`
- `reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/rss-budget-policy.local.json`

## 6. Phase 0 - зафіксувати baseline

### Goal

Перед оптимізаціями зафіксувати поточний стан, щоб не гадати, що саме
покращилось або зламалось.

### Files

- `reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/report.json`
- `reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/rss-budget-policy.local.json`
- `docs/audits/tetra-memory-zero-heap-optimization-final.md`
- новий audit/report artifact для цього треку

### Approach

1. Прогнати validator на поточному report.
2. Витягнути table по всіх Tetra rows:
   - category;
   - row name;
   - status;
   - classification;
   - backend path;
   - backend blockers;
   - heap allocation count;
   - heap reason codes;
   - bounds left.
3. Окремо записати:
   - fallback rows;
   - rows з `bounds_left > 0`;
   - rows з `heap_allocation_count > 0`;
   - actor rows без production actor byte evidence;
   - RSS peak values і policy.
4. Це baseline, не optimization patch.

### Verification

```sh
GOCACHE=$(pwd)/.cache/go-build-post-zero-baseline go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/report.json \
  --rss-budget-policy reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/rss-budget-policy.local.json

jq -r '.results[] as $r | $r.rows[] | select(.language=="tetra") | [.category, .name, .status, ($r.classification // ""), (.tetra_metadata.backend_path // ""), ((.tetra_metadata.backend_blockers // [])|join(",")), (.tetra_metadata.heap_allocations // 0), ((.tetra_metadata.heap_reason_codes // [])|join(",")), (.tetra_metadata.bounds_left // 0)] | @tsv' \
  reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/report.json

GOCACHE=$(pwd)/.cache/go-build-post-zero-baseline go clean -cache
```

### Done when

- validator проходить;
- blocker inventory збережено;
- відомо, з чого починається кожен track.

## 7. Phase 1 - Fallback backend track

### Goal

Найбільший blocker зараз не heap, а fallback backend. Треба перевести рядки з
fallback у реальний register/native backend.

Пріоритет:

1. `integer_loops_tetra`
2. `recursion_tetra`
3. `compile_time_tetra`
4. `hash_table_tetra`
5. `allocation_tetra`
6. `region_island_allocation_tetra`
7. actor rows після production actor memory
8. JSON/HTTP/PostgreSQL після call ABI і aggregate returns

### Поточна причина

`compiler/internal/buildreports/backend.go` має конкретні fallback categories:

- `unsupported_control_flow`
- `unsupported_effect_runtime_call`
- `unsupported_call_abi`
- `unsupported_aggregate_return`
- `unsupported_slice_string_return`
- `stack_fallback`

Це добре: кожну category можна закривати окремо, з tests і evidence.

### Правильна стратегія

Не можна просто прибрати blocker label. Треба додати реальну підтримку в
native path.

Найкращий шлях:

1. Залишити існуючі scalar/register fast paths.
2. Додати загальніший `StackIR/SSA -> MachineIR` selector для підтримуваного
   subset.
3. Навчити MachineIR і x64core basic blocks, branches, calls, return ABI.
4. Fallback classification послаблювати тільки тоді, коли функція проходить:
   - SSA gate;
   - MachineIR verifier;
   - register allocation verifier;
   - x64 emission/build;
   - local run.

### Task 1.1 - backend blocker inventory tests

**Goal:** не дати випадково приховати blockers.

**Files:**

- `compiler/internal/buildreports/backend.go`
- `compiler/internal/buildreports/*_test.go`
- `tools/cmd/local-benchmark-tier1/main_test.go`

**Approach:**

1. Додати tests на synthetic IR:
   - label/jump -> `unsupported_control_flow`;
   - `core.*` runtime call -> `unsupported_effect_runtime_call`;
   - забагато call args/returns -> `unsupported_call_abi`;
   - `ReturnSlots > 2` -> `unsupported_aggregate_return`.
2. Додати позитивний test: простий scalar/register випадок лишається
   `register`.
3. Якщо хтось змінить classification без реалізації, test має падати.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-fallback-inventory go test ./compiler/internal/buildreports ./tools/cmd/local-benchmark-tier1 \
  -run 'Backend|Fallback|Blocker|Classification' -count=1
GOCACHE=$(pwd)/.cache/go-build-fallback-inventory go clean -cache
```

**Done when:** blocker categories protected by tests.

### Task 1.2 - control-flow native path

**Goal:** simple loops і recursion мають компілюватись через native/register
path.

**Rows:**

- `integer_loops_tetra`
- `recursion_tetra`
- `compile_time_tetra`
- частина `hash_table_tetra`

**Files:**

- `compiler/internal/machine/ir.go`
- `compiler/internal/machine/scalar_loop.go`
- `compiler/internal/machine/scalar_call_loop.go`
- `compiler/internal/backend/x64core/emit.go`
- `compiler/internal/backend/x64core/scalar_loop_register.go`
- `compiler/internal/backend/x64core/scalar_call_loop_register.go`
- `compiler/internal/buildreports/backend.go`

**Approach:**

1. RED tests:
   - while loop з counter;
   - nested loop;
   - loop з accumulator;
   - direct recursive function;
   - call inside loop.
2. Перетворити labels/jumps у MachineIR blocks.
3. Emit x64 branches:
   - unconditional branch;
   - compare-zero branch;
   - fallthrough.
4. Перевірити liveness across blocks.
5. Перевірити linear scan across blocks.
6. Fallback лишається, якщо:
   - SSA conversion fails;
   - тип не підтримується;
   - register allocator rejects;
   - x64 emitter не може безпечно emit-ити.
7. Backend report міняти тільки після реального native path.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-control-flow go test ./compiler/internal/machine ./compiler/internal/backend/x64core ./compiler/internal/buildreports ./compiler \
  -run 'Control|Loop|Branch|Recursion|Machine|Backend|Register' -count=1
GOCACHE=$(pwd)/.cache/go-build-control-flow go clean -cache
```

Fresh benchmark:

```sh
GOCACHE=$(pwd)/.cache/go-build-control-flow-bench go run ./tools/cmd/local-benchmark-tier1 \
  -iterations 5 \
  -out-dir reports/benchmark-vnext-memory-baseline/tier1-after-fallback-control-flow

GOCACHE=$(pwd)/.cache/go-build-control-flow-bench go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report reports/benchmark-vnext-memory-baseline/tier1-after-fallback-control-flow/report.json

GOCACHE=$(pwd)/.cache/go-build-control-flow-bench go clean -cache
```

**Done when:**

- `integer_loops_tetra` більше не blocked by `unsupported_control_flow`;
- `recursion_tetra` більше не blocked by `unsupported_control_flow`;
- `compile_time_tetra` або native, або має менший точний blocker;
- binary runs;
- benchmark report validates.

### Task 1.3 - runtime effect call split

**Goal:** не вважати кожен `core.*` або `__tetra_*` call однаково
unsupported.

**Rows:**

- `slice_sum_tetra`
- `bounds_check_loops_tetra`
- `matrix_multiply_tetra`
- `hash_table_tetra`
- `allocation_tetra`
- `region_island_allocation_tetra`
- `actor_ping_pong_tetra`

**Files:**

- `compiler/internal/buildreports/backend.go`
- `compiler/internal/lower/callable_targets.go`
- `compiler/internal/lower/callable_lowering.go`
- `compiler/internal/backend/x64core/call_register.go`
- `compiler/internal/backend/x64core/emit_heap.go`
- `compiler/internal/runtimeabi/`

**Approach:**

1. Зробити call categories:
   - pure lowered builtin;
   - allocation builtin with native lowering;
   - runtime helper with supported ABI;
   - true unsupported runtime effect.
2. Не whitelist-ити тільки за prefix `core.`.
3. Додати call summaries:
   - arg slots;
   - return slots;
   - clobbers;
   - may allocate;
   - may escape input;
   - may touch actor/runtime state.
4. Native backend дозволяти тільки для calls із supported summary.
5. `unsupported_effect_runtime_call` лишати для справді непідтриманих runtime
   effects.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-effect-calls go test ./compiler/internal/lower ./compiler/internal/backend/x64core ./compiler/internal/buildreports ./compiler \
  -run 'Call|Builtin|Runtime|Effect|ABI|Backend' -count=1
GOCACHE=$(pwd)/.cache/go-build-effect-calls go clean -cache
```

**Done when:** row втрачає `unsupported_effect_runtime_call` тільки через
підтриманий call summary і native lowering.

### Task 1.4 - call ABI і aggregate returns

**Goal:** дати достатній call/return ABI, щоб JSON/HTTP/PostgreSQL не падали у
fallback через ABI.

**Rows:**

- `json_parse_stringify_tetra`
- `http_plaintext_json_tetra`
- `postgresql_single_multiple_update_tetra`
- `parallel_map_reduce_tetra`

**Files:**

- `compiler/internal/backend/x64abi/`
- `compiler/internal/backend/x64core/call_register.go`
- `compiler/internal/machine/ir.go`
- `compiler/internal/buildreports/backend.go`
- `compiler/internal/validation/`

**Approach:**

1. Використати існуючий `x64abi` classifier як source of truth.
2. Додати tests:
   - multi-argument scalar calls;
   - two-slot returns;
   - aggregate return через caller-owned output/sret;
   - call clobber preservation.
3. Реалізувати hidden return pointer або caller-owned result buffer там, де ABI
   цього вимагає.
4. Call site має report-ити:
   - target ABI;
   - arg slots;
   - ret slots;
   - clobbers;
   - stack adjustment.
5. `unsupported_call_abi` лишається, доки call не emitted і verified.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-call-abi go test ./compiler/internal/backend/x64abi ./compiler/internal/backend/x64core ./compiler/internal/machine ./compiler/internal/buildreports ./compiler \
  -run 'ABI|Call|Aggregate|Return|Clobber|Backend' -count=1
GOCACHE=$(pwd)/.cache/go-build-call-abi go clean -cache
```

**Done when:** JSON/HTTP/PostgreSQL втрачають ABI/aggregate fallback або мають
менший чесний blocker.

## 8. Phase 2 - Bounds-check elimination

### Goal

Прибрати remaining bounds checks тільки там, де compiler має доказ.

Target rows:

| Row | Current `bounds_left` |
| --- | ---: |
| `slice_sum_tetra` | 1 |
| `bounds_check_loops_tetra` | 2 |
| `matrix_multiply_tetra` | 7 |

JSON/HTTP/PostgreSQL мають багато `bounds_left`, але вони йдуть після ABI і
fallback work.

### Поточний стан

Уже є:

- `compiler/internal/lower/lower_rangeproof.go`;
- `compiler/internal/lower/rangeproof/`;
- `compiler/internal/rangeproof/`;
- `compiler/internal/lower/proof_bce_test.go`;
- `compiler/internal/validation/validation_translation.go`.

Проблема не в тому, що BCE відсутній. Проблема в тому, що proof coverage ще не
закриває benchmark patterns.

### Правильна стратегія

Bounds check можна прибрати тільки коли:

```text
index lower bound >= 0
index upper bound < collection length
collection length stable in proof scope
proof not invalidated by mutation/inout/alias/branch merge
unchecked IR has proof_id
translation validation preserves proof facts
```

### Task 2.1 - перенести benchmark patterns у BCE tests

**Goal:** бачити проблему без повного Tier 1 run.

**Files:**

- `tools/cmd/local-benchmark-tier1/tetra_sources.go`
- `compiler/internal/lower/proof_bce_test.go`
- `compiler/internal/lower/rangeproof/rangeproof_test.go`

**Approach:**

1. Взяти minimal source patterns для:
   - slice sum;
   - bounds-check loops;
   - matrix multiply.
2. Додати RED tests, які показують remaining checks.
3. Додати expected final tests:
   - safe index -> unchecked load/store з proof ID;
   - unsafe alias/mutation variant -> checked operation лишається.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-bce-patterns go test ./compiler/internal/lower ./compiler/internal/lower/rangeproof \
  -run 'Bounds|Proof|Slice|Matrix|Loop' -count=1
GOCACHE=$(pwd)/.cache/go-build-bce-patterns go clean -cache
```

**Done when:** tests фіксують саме ті missing proofs, які бачить benchmark.

### Task 2.2 - посилити range facts

**Goal:** довести common loop ranges.

**Files:**

- `compiler/internal/lower/lower_rangeproof.go`
- `compiler/internal/lower/rangeproof/rangeproof.go`
- `compiler/internal/rangeproof/rangeproof.go`

**Approach:**

Додати proof support для:

1. `i < xs.len`, де `i` стартує з `0` і росте на `1`.
2. `i <= xs.len - 1`.
3. Nested loops з незалежними indices.
4. Matrix flat index, де index виводиться з:
   - row;
   - column;
   - stride/width;
   - matrix length.
5. Stable length aliases:
   - `n = xs.len`;
   - `n` використано в loop condition;
   - `xs` не mutates у loop.
6. Branch join facts тільки якщо обидві гілки зберігають same safe bounds.

### Task 2.3 - proof-aware lowering

**Goal:** safe index operations lowering -> unchecked IR with proof IDs.

**Approach:**

1. Коли `activeWhileProofForIndex` successful, emit unchecked load/store.
2. Додати `proof_id`.
3. Mutation collection/index -> invalidate proof.
4. `inout` call, який може мутувати collection -> invalidate proof.
5. External/invalid provenance -> keep checked operation.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-bce go test ./compiler/internal/lower ./compiler/internal/validation ./compiler \
  -run 'Bounds|Proof|Translation|Unchecked|Matrix|Slice' -count=1
GOCACHE=$(pwd)/.cache/go-build-bce go clean -cache
```

Fresh benchmark:

```sh
GOCACHE=$(pwd)/.cache/go-build-bce-bench go run ./tools/cmd/local-benchmark-tier1 \
  -iterations 5 \
  -out-dir reports/benchmark-vnext-memory-baseline/tier1-after-bounds-elimination

GOCACHE=$(pwd)/.cache/go-build-bce-bench go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report reports/benchmark-vnext-memory-baseline/tier1-after-bounds-elimination/report.json

GOCACHE=$(pwd)/.cache/go-build-bce-bench go clean -cache
```

**Done when:**

- `slice_sum_tetra.bounds_left == 0`;
- `bounds_check_loops_tetra.bounds_left == 0`;
- `matrix_multiply_tetra.bounds_left == 0`;
- unsafe variants still keep checks;
- proof/bounds reports validate.

## 9. Phase 3 - добити 5 heap rows

### Goal

П'ять remaining heap rows мають перейти в zero runtime Tetra heap, якщо compiler
може довести non-heap lifetime.

Current heap rows:

| Row | Heap reason |
| --- | --- |
| `slice_sum_tetra` | `heap.required_large_object` |
| `bounds_check_loops_tetra` | `heap.required_large_object` |
| `json_parse_stringify_tetra` | `heap.required_unknown_call` |
| `http_plaintext_json_tetra` | `heap.required_unknown_call` |
| `postgresql_single_multiple_update_tetra` | `heap.required_unknown_call` |

### Правильна стратегія

Не можна просто збільшити `smallStackAllocationBytes`.

Зараз `smallStackAllocationBytes = 4096`, а два rows мають 16384 B allocation.
Якщо просто підняти threshold, ми сховаємо проблему і можемо роздути stack.

Правильний вибір storage:

```text
small fixed local object -> stack
large local bounded object -> function temp region або explicit region/island
returned object -> caller-owned result або heap, якщо escape реальний
unknown call -> heap, доки немає callee lifetime summary
```

### Task 3.1 - large local object region/stack path

**Goal:** прибрати `heap.required_large_object` для локальних benchmark arrays
без небезпечного stack growth.

**Rows:**

- `slice_sum_tetra`
- `bounds_check_loops_tetra`

**Files:**

- `compiler/internal/allocplan/plan.go`
- `compiler/internal/validation/validation_allocation_lifetimes.go`
- `compiler/internal/backend/x64core/emit_heap.go`
- `compiler/internal/backend/x64core/heap_telemetry.go`
- `compiler/internal/allocplan/heap_reason_codes_test.go`

**Approach:**

1. Розділити stack budget і region budget.
2. Для local bounded object > stack budget:
   - plan `FunctionTempRegion` або `ExplicitIsland`;
   - emit region enter/make/reset IR;
   - validate reset dominates all exits;
   - validate no return/call/global escape.
3. Для small fixed object:
   - лишити stack path.
4. Keep heap if:
   - lifetime crosses function boundary;
   - object returned;
   - object captured;
   - region lowering unavailable.
5. Heap reason codes міняти тільки через реальне storage decision.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-large-object go test ./compiler/internal/allocplan ./compiler/internal/validation ./compiler/internal/backend/x64core ./compiler \
  -run 'Large|Region|Stack|FunctionTemp|HeapReason|AllocationLowering' -count=1
GOCACHE=$(pwd)/.cache/go-build-large-object go clean -cache
```

**Done when:**

- target rows мають `heap_allocation_count == 0`;
- allocation report не має `heap.required_large_object` для них;
- runtime heap sidecar підтверджує zero heap;
- stack budget не порушено.

### Task 3.2 - unknown call lifetime summaries

**Goal:** прибрати `heap.required_unknown_call` тільки коли call boundary має
явні lifetime/escape facts.

**Rows:**

- `json_parse_stringify_tetra`
- `http_plaintext_json_tetra`
- `postgresql_single_multiple_update_tetra`

**Files:**

- `compiler/internal/allocplan/plan.go`
- `compiler/internal/memoryfacts/from_plir_allocplan.go`
- `compiler/internal/validation/validation_allocation_lifetimes.go`
- `compiler/internal/lower/callable_targets.go`
- `compiler/internal/lower/callable_lowering.go`
- `tools/cmd/local-benchmark-tier1/tetra_sources.go`

**Approach:**

1. Будувати local call summaries з PLIR:
   - callee не store-ить input глобально;
   - callee не return-ить input;
   - callee не передає input в unknown external call;
   - callee не sends input to actor/task boundary;
   - callee не тримає input після return.
2. Safe calls позначити як no-escape calls.
3. Calls без summary залишити heap fallback.
4. Для JSON/HTTP/PostgreSQL краще перейти на caller-owned buffers:
   - caller створює output buffer/region;
   - callee пише в нього;
   - callee повертає length/status, не heap-owned aggregate.
5. Negative tests:
   - unknown external call still heap;
   - returned/captured buffer still heap;
   - actor/task boundary окремо.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-call-lifetime go test ./compiler/internal/allocplan ./compiler/internal/memoryfacts ./compiler/internal/validation ./compiler/internal/lower ./compiler \
  -run 'UnknownCall|Lifetime|NoEscape|CallerOwned|HeapReason' -count=1
GOCACHE=$(pwd)/.cache/go-build-call-lifetime go clean -cache
```

Fresh benchmark:

```sh
GOCACHE=$(pwd)/.cache/go-build-zero-five-heap go run ./tools/cmd/local-benchmark-tier1 \
  -iterations 5 \
  -out-dir reports/benchmark-vnext-memory-baseline/tier1-after-five-heap-rows

GOCACHE=$(pwd)/.cache/go-build-zero-five-heap go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report reports/benchmark-vnext-memory-baseline/tier1-after-five-heap-rows/report.json

GOCACHE=$(pwd)/.cache/go-build-zero-five-heap go clean -cache
```

**Done when:**

- усі 5 target rows мають runtime-measured `heap_allocation_count == 0`;
- усі 5 target rows мають runtime-measured `heap_alloc_bytes == 0`;
- heap reason codes зникли тільки через proof/lifetime/storage evidence;
- якщо row не можна зробити zero-heap, він marked blocked з exact escape
  evidence.

## 10. Phase 4 - Production actor memory

### Goal

Перевести actor memory з model evidence у production runtime evidence.

Поточний стан:

- `compiler/internal/parallelrt/scheduler_model.go` уже моделює mailbox bytes,
  owned region moves, copy counts, byte backpressure;
- `compiler/internal/actorsrt/` має production x64 actor runtime;
- benchmark report ще не має production per-actor runtime byte sidecars.

### Правильна стратегія

`parallelrt` model - це specification/reference. Але production evidence має
йти з `actorsrt`, коли реально запускається compiled Tetra program.

Production actor memory має показувати:

```text
actor domain id
mailbox queued message count
mailbox queued bytes
mailbox peak queued bytes
message pool capacity bytes
message pool live bytes
message pool reclaimed bytes
owned region bytes
bytes copied
copy count
domain budget bytes
domain current bytes
domain peak bytes
backpressure status
backpressure reason
```

### Task 4.1 - actor runtime counter layout

**Goal:** додати стабільні byte counters у production `actorsrt`.

**Files:**

- `compiler/internal/actorsrt/linux_x64.go`
- `compiler/internal/actorsrt/linux_x64_emit.go`
- `compiler/internal/actorsrt/linux_x64_emit_send.go`
- `compiler/internal/actorsrt/linux_x64_emit_surface_recv.go`
- `compiler/internal/actorsrt/runtime_source_parity_test.go`
- `compiler/internal/actorsrt/production_boundary_test.go`

**Approach:**

1. Додати actor struct offsets для:
   - current mailbox bytes;
   - peak mailbox bytes;
   - reclaimed bytes;
   - bytes copied;
   - copy count;
   - budget bytes.
2. Додати scheduler/message pool counters:
   - pool capacity;
   - live bytes;
   - reclaimed bytes;
   - allocation failures.
3. Оновити parity tests, щоб offsets не drift-или silently.
4. Initialize counters on spawn.
5. Increment counters on send.
6. Decrement/reclaim counters on recv.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-actor-counters go test ./compiler/internal/actorsrt ./compiler \
  -run 'Actor|Mailbox|MessagePool|Counter|Runtime|Backpressure' -count=1
GOCACHE=$(pwd)/.cache/go-build-actor-counters go clean -cache
```

**Done when:** production actor runtime рахує bytes без зміни існуючих checked
failure semantics.

### Task 4.2 - byte backpressure у production runtime

**Goal:** actor runtime має reject-ити messages по byte budget, не тільки по
message count/pool exhaustion.

**Files:**

- `compiler/internal/actorsrt/linux_x64_emit_send.go`
- `compiler/internal/actorsrt/linux_x64_emit.go`
- `compiler/actors_test.go`

**Approach:**

1. Compute message footprint:
   - fixed message node bytes;
   - typed payload bytes;
   - owned region bytes when ownership moves.
2. Compare `current_bytes + message_bytes` з actor budget.
3. Return checked backpressure failure before allocation/enqueue.
4. Keep existing message-count limit.
5. Keep existing message-pool exhaustion behavior.
6. Test: failed byte backpressure does not enqueue partial payload.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-actor-byte-backpressure go test ./compiler/internal/actorsrt ./compiler \
  -run 'Actor.*Backpressure|Mailbox.*Byte|MessagePool|PartialPayload' -count=1
GOCACHE=$(pwd)/.cache/go-build-actor-byte-backpressure go clean -cache
```

**Done when:** byte backpressure є production runtime behavior і recover after
drain.

### Task 4.3 - runtime actor memory sidecar

**Goal:** benchmark rows мають отримувати actor domain bytes з реального run.

**Files:**

- `tools/internal/heaptelemetry/heaptelemetry.go`
- `compiler/internal/backend/x64core/heap_telemetry.go`
- `tools/cmd/local-benchmark-tier1/metadata.go`
- `tools/internal/ramvalidate/ramvalidate.go`

**Approach:**

1. Extend runtime heap telemetry для actor domain rows.
2. Process domain залишити для non-actor rows.
3. Actor domain rows:
   - `domain_id: actor:<id>`;
   - `kind: actor`;
   - `requested_bytes`;
   - `reserved_bytes`;
   - `committed_bytes`;
   - `current_bytes`;
   - `peak_bytes`;
   - `bytes_copied`.
4. Tier 1 metadata має prefer runtime-measured domain bytes over
   allocation-report estimates, якщо sidecar valid.
5. Для rows без runtime domain evidence лишати `allocation_report_estimate` або
   `unsupported`.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-actor-telemetry go test ./tools/internal/heaptelemetry ./tools/internal/ramvalidate ./tools/cmd/local-benchmark-tier1 ./compiler/internal/backend/x64core ./compiler \
  -run 'HeapTelemetry|DomainBytes|Actor|RuntimeMeasured|MemoryEvidence' -count=1
GOCACHE=$(pwd)/.cache/go-build-actor-telemetry go clean -cache
```

Fresh benchmark:

```sh
GOCACHE=$(pwd)/.cache/go-build-actor-memory-bench go run ./tools/cmd/local-benchmark-tier1 \
  -iterations 5 \
  -out-dir reports/benchmark-vnext-memory-baseline/tier1-after-production-actor-memory

GOCACHE=$(pwd)/.cache/go-build-actor-memory-bench go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report reports/benchmark-vnext-memory-baseline/tier1-after-production-actor-memory/report.json

GOCACHE=$(pwd)/.cache/go-build-actor-memory-bench go clean -cache
```

**Done when:**

- actor rows мають runtime-measured actor domain bytes;
- mailbox/message bytes видимі в evidence;
- byte backpressure - production runtime evidence;
- actor rows більше не описуються як тільки `parallelrt` model evidence.

## 11. Phase 5 - RSS reduction

### Goal

Зменшити process RSS і зробити RSS local regression gate.

### Поточна проблема

`zero heap != zero RSS`.

RSS включає:

- executable code;
- loader mappings;
- stack;
- runtime object;
- actor/runtime pools;
- pages touched during startup;
- linked libraries або target support code.

Поточний RSS policy artifact:

```text
reports/benchmark-vnext-memory-baseline/tier1-after-memory-zero-heap-optimization/rss-budget-policy.local.json
```

### Правильна стратегія

RSS - це local process footprint metric, не Tetra heap.

Порядок:

1. Measure і gate current RSS.
2. Reduce runtime object linking.
3. Reduce runtime initialization/touched pages.
4. Split actor/net/surface/filesystem runtime pieces.
5. Keep budgets host-pinned.

### Task 5.1 - RSS budget gate per row

**Goal:** local RSS regression має падати, якщо row перевищив pinned budget.

**Files:**

- `docs/spec/local_rss_budget_policy.md`
- `tools/cmd/validate-local-benchmark-tier1/`
- `reports/benchmark-vnext-memory-baseline/*/rss-budget-policy.local.json`

**Approach:**

1. `rss_peak` і `rss_current` лишити окремими metrics.
2. Budget applies only when:
   - target matches;
   - host profile matches;
   - row exists;
   - row measured;
   - RSS sidecar validates.
3. Required nonclaims:
   - local RSS budget only;
   - no cross-machine RSS claim;
   - no official benchmark claim.
4. Tests:
   - budget pass;
   - budget fail;
   - host mismatch -> not applicable;
   - missing sidecar -> fail/blocked, not pass.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-gate go test ./tools/cmd/validate-local-benchmark-tier1 ./tools/internal/rsstelemetry ./tools/cmd/local-benchmark-tier1 \
  -run 'RSS|Budget|Policy|Host|Sidecar' -count=1
GOCACHE=$(pwd)/.cache/go-build-rss-gate go clean -cache
```

**Done when:** RSS budget validation is reliable and host-pinned.

### Task 5.2 - runtime object feature splitting

**Goal:** link/init only runtime pieces actually used by a program.

**Files:**

- `compiler/internal/buildruntime/runtime_object_plan.go`
- `compiler/internal/buildruntime/runtime_object.go`
- `compiler/internal/buildruntime/selection.go`
- `compiler/reports_runtime_object.go`
- `compiler/internal/actorsrt/linux_x64.go`

**Approach:**

1. `runtimeObjectFeaturesRequired` - source of truth.
2. Split runtime features:
   - time;
   - actor core;
   - actor state;
   - task runtime;
   - filesystem;
   - net;
   - surface;
   - distributed actor runtime.
3. Simple integer loops must not link actor/net/surface.
4. Actor pools must not initialize if no actor runtime feature is required.
5. Report має показувати:
   - features required;
   - features linked;
   - features initialized;
   - lazy init blockers.
6. Tests compare feature usage with linked runtime object plan.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-runtime-split go test ./compiler/internal/buildruntime ./compiler \
  -run 'RuntimeObject|Feature|Lazy|Minimal|Linked|Initialized' -count=1
GOCACHE=$(pwd)/.cache/go-build-runtime-split go clean -cache
```

**Done when:** simple rows do not link/init unrelated runtime features.

### Task 5.3 - startup footprint audit

**Goal:** зрозуміти, що саме створює RSS до benchmark work.

**Files/tools:**

- `tools/cmd/local-benchmark-tier1/command.go`
- `tools/internal/rsstelemetry/rsstelemetry.go`
- generated binaries under benchmark artifacts
- `readelf`
- `size`
- `objdump`

**Approach:**

1. Якщо треба, додати optional startup RSS sampling window.
2. Порівнювати:
   - binary text/data/bss size;
   - runtime object linked features;
   - RSS current;
   - RSS peak.
3. Визначати source peak:
   - runtime startup;
   - actor stack/message pool mmap;
   - large static sections;
   - benchmark work.
4. Зменшувати одну причину за раз.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-audit go test ./tools/internal/rsstelemetry ./tools/cmd/local-benchmark-tier1 ./compiler/internal/buildruntime ./compiler \
  -run 'RSS|Startup|RuntimeObject|Binary|Telemetry' -count=1
GOCACHE=$(pwd)/.cache/go-build-rss-audit go clean -cache
```

Fresh RSS benchmark:

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-final go run ./tools/cmd/local-benchmark-tier1 \
  -iterations 5 \
  -out-dir reports/benchmark-vnext-memory-baseline/tier1-after-rss-reduction

GOCACHE=$(pwd)/.cache/go-build-rss-final go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report reports/benchmark-vnext-memory-baseline/tier1-after-rss-reduction/report.json \
  --rss-budget-policy reports/benchmark-vnext-memory-baseline/tier1-after-rss-reduction/rss-budget-policy.local.json

GOCACHE=$(pwd)/.cache/go-build-rss-final go clean -cache
```

**Done when:**

- RSS budget gates pass on pinned host profile;
- simple rows do not link unrelated runtime pieces;
- RSS changes explained by evidence, not guesses.

## 12. Phase 6 - final integrated gate

### Goal

Зібрати фінальний local evidence package після реалізації всіх accepted tracks.

### Required commands

```sh
GOCACHE=$(pwd)/.cache/go-build-post-zero-final go test ./compiler/... ./tools/... \
  -run 'Backend|Fallback|Bounds|Proof|Allocation|Heap|Actor|RSS|RuntimeObject|Telemetry|Benchmark|Validate' -count=1

GOCACHE=$(pwd)/.cache/go-build-post-zero-final go run ./tools/cmd/local-benchmark-tier1 \
  -iterations 5 \
  -out-dir reports/benchmark-vnext-memory-baseline/tier1-after-native-memory-complete

GOCACHE=$(pwd)/.cache/go-build-post-zero-final go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report reports/benchmark-vnext-memory-baseline/tier1-after-native-memory-complete/report.json \
  --rss-budget-policy reports/benchmark-vnext-memory-baseline/tier1-after-native-memory-complete/rss-budget-policy.local.json

graphify update .

git diff --check

GOCACHE=$(pwd)/.cache/go-build-post-zero-final go clean -cache
```

### Final acceptance criteria

Fallback:

- `integer_loops_tetra` not fallback;
- `recursion_tetra` not fallback;
- `compile_time_tetra` not fallback або має менший exact blocker;
- `hash_table_tetra`, `allocation_tetra`,
  `region_island_allocation_tetra` втрачають fallback тільки через real native
  support.

Bounds:

- `slice_sum_tetra.bounds_left == 0`;
- `bounds_check_loops_tetra.bounds_left == 0`;
- `matrix_multiply_tetra.bounds_left == 0`;
- proof reports і translation validation pass.

Heap:

- 5 target heap rows runtime-measured zero heap або exact blocked escape;
- heap reason code не видалений без storage/lifetime evidence.

Actors:

- actor rows мають production runtime actor domain byte evidence;
- mailbox/message bytes measured або explicitly blocked;
- byte backpressure runtime evidence, not only model evidence.

RSS:

- RSS sidecars validate;
- RSS budget policy applies on pinned host;
- simple rows do not link/init unrelated runtime pieces.

Docs/evidence:

- final report path recorded;
- final audit explains measured/estimated/unsupported/blocked;
- nonclaims present.

## 13. Рекомендований порядок виконання

Порядок важливий, бо fallback зараз найбільший blocker:

1. Phase 0: baseline lock.
2. Phase 1.1: blocker inventory tests.
3. Phase 1.2: control-flow native path для `integer_loops_tetra`.
4. Phase 2.1-2.2: BCE tests і range facts для slice/bounds/matrix.
5. Phase 3.1: large local object region/stack path.
6. Phase 1.3: runtime effect call split.
7. Phase 3.2: unknown call lifetime summaries.
8. Phase 1.4: call ABI і aggregate returns.
9. Phase 4: production actor memory.
10. Phase 5: RSS reduction and gates.
11. Phase 6: final integrated benchmark gate.

Перший вузький implementation goal:

```text
Make integer_loops_tetra leave fallback by implementing the needed
control-flow/register backend path, with RED/GREEN tests, backend report
evidence, local benchmark evidence, and validation.
```

## 14. Stop rules

Зупинитися і записати blocker, якщо:

- той самий fix падає двічі;
- native backend emits code, але губить validation metadata;
- bounds check можна прибрати тільки без proof ID;
- heap стає zero тільки тому, що telemetry перестала рахувати allocations;
- RSS падає тільки тому, що benchmark work вимкнули;
- actor byte evidence є тільки в `parallelrt`, а не в production `actorsrt`;
- row змінив classification, але raw sidecars не підтверджують це.

## 15. Definition of Done

План реалізований тільки коли:

- code changes є для всіх accepted phases;
- affected compiler/runtime/tool tests pass;
- fresh Tier 1 benchmark report exists;
- fresh report validates;
- RSS budget policy validates on pinned host profile;
- final audit explains remaining unsupported/blocked items;
- `graphify update .` run after code changes;
- `git diff --check` passes;
- no unrelated scratch artifacts remain.

До цього чесний статус: `PARTIAL`.
