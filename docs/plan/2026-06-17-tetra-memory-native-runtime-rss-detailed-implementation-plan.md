# Детальний план реалізації Tetra memory/native/runtime/RSS

**Статус:** план виконання, не доказ реалізації.  
**Дата:** 2026-06-17.  
**Головна ціль:** довести memory/native історію Tetra до стабільного стану без фейкових claim-ів:
реальний native/register backend, нуль heap-regression, нуль bounds-regression, production actor
memory evidence і локальні RSS gates.  
**Поточна truth-база:**  
`reports/benchmark-vnext-memory-baseline/tier1-after-postgresql-inout-writer-native/report.json`  
**Пов'язаний master plan:**  
`docs/plan/2026-06-17-tetra-native-memory-completion-plan.md`  
**Workflow evidence:**  
`.workflow/post-zero-heap-native-memory/`

## 1. Коротка правда перед стартом

Старе формулювання було правильним для попереднього baseline, але зараз стан кращий.
Тому реалізацію треба вести від поточного P72 report, а не від старого списку blockers.

Поточний стан Tetra Tier 1:

| Метрика | Поточний стан |
| --- | ---: |
| Tetra rows | 17 |
| Runtime-measured zero heap | 17/17 |
| `bounds_left == 0` | 17/17 |
| Row-level fallback | 8/17 |
| Row-level register/native | 9/17 |

Поточні fallback rows:

| Row | Backend blocker | Heap | Bounds |
| --- | --- | ---: | ---: |
| `slice_sum_tetra` | `unsupported_control_flow` | 0 | 0 |
| `matrix_multiply_tetra` | `unsupported_control_flow` | 0 | 0 |
| `hash_table_tetra` | `unsupported_control_flow` | 0 | 0 |
| `region_island_allocation_tetra` | `unsupported_effect_runtime_call` | 0 | 0 |
| `json_parse_stringify_tetra` | `unsupported_aggregate_return`, `unsupported_call_abi` | 0 | 0 |
| `http_plaintext_json_tetra` | `unsupported_aggregate_return`, `unsupported_call_abi` | 0 | 0 |
| `actor_ping_pong_tetra` | `unsupported_effect_runtime_call` | 0 | 0 |
| `parallel_map_reduce_tetra` | `unsupported_call_abi` | 0 | 0 |

Rows, які вже не треба називати поточними blockers:

- `integer_loops_tetra` вже `register`;
- `recursion_tetra` вже `register`;
- `bounds_check_loops_tetra` вже `register`;
- `allocation_tetra` вже `register`;
- `compile_time_tetra` вже `register`;
- `postgresql_single_multiple_update_tetra` вже `register`;
- старі 5 heap-positive rows зараз уже мають `heap_allocations == 0`.

Висновок:

- найбільший активний blocker зараз справді `fallback backend`;
- bounds і heap зараз не основна оптимізаційна задача, а regression-safety задача;
- actor memory ще не завершена як production runtime budget/backpressure;
- RSS треба зменшувати окремо, не змішуючи його з heap.

## 2. Що означає "готово"

Фінальний `DONE` можна ставити тільки якщо виконані всі пункти:

1. Fresh Tier 1 report згенерований після останньої зміни backend/heap/bounds/actor/RSS.
2. `17/17` Tetra rows мають `heap_allocations == 0`.
3. `17/17` Tetra rows мають `bounds_left == 0`.
4. Кожен row, який став `register`, має backend sidecar proof.
5. Жоден row не став `register` через зміну report label.
6. JSON/HTTP ABI blockers або реально закриті, або залишені з точним blocker reason.
7. Actor memory має production runtime counters:
   - mailbox bytes;
   - message bytes;
   - owned region bytes;
   - copied bytes;
   - moved bytes;
   - byte budget;
   - backpressure events.
8. RSS має host-pinned row-specific budget policy.
9. Validator падає на підроблених або неповних memory/backend evidence fixtures.
10. Final audit явно пише non-claims:
    - no zero RSS claim;
    - no official TechEmpower claim;
    - no cross-machine RSS claim;
    - no "all programs zero heap" claim;
    - no production OS footprint claim.

Якщо хоча б один пункт не виконаний, статус має бути `PARTIAL`, не `DONE`.

## 3. Головний принцип реалізації

Правильний шлях:

```text
compiler/runtime capability -> sidecar evidence -> benchmark report -> validator gate -> audit claim
```

Неправильний шлях:

```text
змінити report label -> сказати, що backend/memory готові
```

Тобто:

- backend path міняється тільки після реального machine/x64core support;
- heap стає нульовим тільки через runtime telemetry або alloc/lifetime proof;
- bounds checks прибираються тільки через range proof;
- actor memory рахується в production runtime, не тільки в benchmark metadata;
- RSS gate є локальним і host-specific.

## 4. Базові правила виконання

Для кожної code-зміни:

1. Спершу прочитати поточний sidecar/report/source shape.
2. Додати RED test для конкретного пропущеного capability.
3. Реалізувати мінімальну зміну.
4. Запустити targeted tests.
5. Якщо змінився `backend_path`, `bounds_left`, heap або actor/RSS metadata,
   згенерувати fresh Tier 1 report.
6. Прогнати `validate-local-benchmark-tier1`.
7. Після code changes запустити `graphify update .`.
8. Прогнати `git diff --check`.
9. Записати evidence у `.workflow/post-zero-heap-native-memory/`.

Go cache тільки в persistent path:

```sh
GOCACHE=$(pwd)/.cache/go-build-native-memory-<slug> go test ./...
GOCACHE=$(pwd)/.cache/go-build-native-memory-<slug> go clean -cache
```

Не використовувати `GOCACHE=/tmp/...`.

## 5. Track 1: Fallback backend

### 5.1. Ціль track

Перевести поточні 8 fallback rows у native/register там, де це реально можливо.
Якщо неможливо, blocker має стати точнішим, а не оптимістичнішим.

Поточні blocker families:

### ABI / Aggregate Return

- Rows: JSON, HTTP, parallel.
- Реальний зміст: compiler ще не має чесного support для конкретної
  call/return ABI shape.

### Composite Control Flow

- Rows: slice, matrix, hash main.
- Реальний зміст: machine backend не складає цілу функцію з кількох
  loop/call/branch regions.

### Runtime Effect Call

- Rows: region/island, actor.
- Реальний зміст: треба відділити domain/runtime primitive від справжнього
  runtime call.

### 5.2. Найкращий порядок

Не починати з широкого "generic control-flow backend". Це ризиковано.

Рекомендований порядок:

1. JSON/HTTP ABI feasibility.
2. JSON exact writer helper.
3. HTTP exact writer helpers.
4. Parallel call ABI, якщо вона така сама або простіша.
5. Region/island primitive split.
6. Composite control-flow composer for slice/hash/matrix.
7. Actor backend тільки після production actor memory budget/backpressure.

Причина:

- P72 уже довів один exact `inout []u8` writer path на PostgreSQL.
- JSON і HTTP мають дуже схожі `inout []u8 -> Int` writer helpers.
- Але P72 не можна автоматично generalize-ити: треба окремо перевірити
  PLIR/sidecar shape.
- Hash/slice/matrix потребують ширшої композиції control-flow, і P76 показав,
  що benchmark-source split не закриває row-level fallback.

### 5.3. Task 1A: JSON/HTTP ABI feasibility

**Goal:** вирішити, чи можна зробити один безпечний worker slice для JSON або HTTP ABI.

**Read files:**

### JSON Backend Sidecar

- Directory:
  `reports/benchmark-vnext-memory-baseline/tier1-after-postgresql-inout-writer-native/artifacts/bin`
- File: `json_parse_stringify_tetra.backend.json`

### HTTP Backend Sidecar

- Directory:
  `reports/benchmark-vnext-memory-baseline/tier1-after-postgresql-inout-writer-native/artifacts/bin`
- File: `http_plaintext_json_tetra.backend.json`

### PostgreSQL Backend Sidecar

- Directory:
  `reports/benchmark-vnext-memory-baseline/tier1-after-postgresql-inout-writer-native/artifacts/bin`
- File: `postgresql_single_multiple_update_tetra.backend.json`

### JSON Source Fixture

- Directory:
  `reports/benchmark-vnext-memory-baseline/tier1-after-postgresql-inout-writer-native/artifacts/src`
- Subdirectory: `p25`
- File: `json_parse_stringify.tetra`

### HTTP Source Fixture

- Directory:
  `reports/benchmark-vnext-memory-baseline/tier1-after-postgresql-inout-writer-native/artifacts/src`
- Subdirectory: `p25`
- File: `http_plaintext_json.tetra`

### PostgreSQL Source Fixture

- Directory:
  `reports/benchmark-vnext-memory-baseline/tier1-after-postgresql-inout-writer-native/artifacts/src`
- Subdirectory: `p25`
- File: `postgresql_single_multiple_update.tetra`
- `compiler/internal/backend/x64abi/`
- `compiler/internal/machine/`
- `compiler/internal/backend/x64core/`
- `compiler/internal/buildreports/backend.go`

**Current facts:**

- JSON has 2 functions:
  - `p25.json_parse_stringify.write_message_object` ->
    `unsupported_aggregate_return`, `return_slots=3`;
  - `p25.json_parse_stringify.main` -> `unsupported_call_abi`, calls helper with `ret_slots=3`.
- HTTP has 3 functions:
  - `write_plaintext_response` -> `unsupported_aggregate_return`, `return_slots=3`;
  - `write_json_response` -> `unsupported_aggregate_return`, `return_slots=3`;
  - `main` -> `unsupported_call_abi`.
- PostgreSQL now has 6/6 `register`, but this was exact writer handling, not generic ABI widening.

**Approach:**

1. Compare PLIR shape of JSON writer with PostgreSQL `write_i32_be_at`.
2. Confirm whether JSON/HTTP `return_slots=3` means:
   - scalar `Int` return plus hidden `inout []u8` writeback slots;
   - or real aggregate return requiring ABI support.
3. Check whether existing P72 implementation can safely accept JSON writer shape.
4. Decide the first worker:
   - JSON writer only;
   - JSON writer plus `main`;
   - HTTP one writer only;
   - no worker, broader ABI design needed.
5. Write result before editing compiler code.

**Done when:**

- one exact implementation slice is named, or a blocker is recorded;
- no row-level promotion is claimed yet;
- near-miss cases are listed.

**Stop if:**

- JSON/HTTP IR differs from P72 in ownership/lifetime meaning;
- support would require generic multi-slot ABI widening;
- row-level promotion would require more than one unrelated capability.

### 5.4. Task 1B: JSON exact writer native/register

**Goal:** promote the smallest JSON writer path through real backend support.

**Likely write files:**

- `compiler/internal/machine/machine_core.go`
- `compiler/internal/machine/machine_suite_test.go`
- `compiler/internal/backend/x64core/x64core_core.go`
- `compiler/internal/backend/x64core/x64core_suite_test.go`
- `compiler/internal/buildreports/backend.go`
- `compiler/compiler_suite_test.go`

**Approach:**

1. RED test: JSON writer remains fallback today because of `return_slots=3`.
2. RED test: unsupported multi-slot aggregate near-miss stays fallback.
3. Add exact recognizer for JSON writer shape only if it is semantically
   scalar return plus `inout` writeback.
4. Add x64core emission test.
5. Add backend report test:
   - helper becomes `register`;
   - `main` remains fallback until call ABI is supported, unless the same
     exact safe slice covers it.
6. Run targeted tests.
7. Generate fresh Tier 1 if row/function classification changes.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-json-writer go test \
  ./compiler/internal/machine \
  ./compiler/internal/backend/x64core \
  ./compiler/internal/buildreports \
  ./compiler \
  -run 'JSON|Writer|Inout|Aggregate|CallABI|Backend' -count=1

GOCACHE=$(pwd)/.cache/go-build-json-writer go clean -cache
```

**Done when:**

- JSON helper sidecar shows `backend_path=register` only if real machine path exists;
- row stays fallback if `main` still cannot call the helper;
- heap and bounds stay zero.

### 5.5. Task 1C: JSON main call ABI

**Goal:** allow `main` to call the promoted JSON writer without stack fallback.

**Approach:**

1. Test call ABI from `main` to exact promoted `inout []u8 -> Int` writer.
2. Preserve caller-owned buffer semantics.
3. Ensure hidden writeback slots do not become heap escape.
4. Keep generic multi-slot aggregate calls unsupported.
5. Promote row-level JSON only when all JSON functions are register.

**Done when:**

- `json_parse_stringify_tetra.backend.json` has `function_count=2`,
  `register_path=2`, `stack_fallback=0`;
- report row becomes `backend_path=register`;
- heap and bounds stay zero.

### 5.6. Task 1D: HTTP writer/main ABI

**Goal:** repeat JSON path for HTTP, but only after JSON proves the pattern.

**Approach:**

1. Handle `write_plaintext_response`.
2. Handle `write_json_response`.
3. Then handle `main`.
4. Do not promote HTTP row until all 3 functions have sidecar proof.

**Done when:**

- HTTP backend sidecar has `function_count=3`, `register_path=3`, `stack_fallback=0`;
- row-level HTTP is `register`;
- no new heap/bounds regression.

### 5.7. Task 1E: Parallel map/reduce call ABI

**Goal:** close `parallel_map_reduce_tetra` only if its remaining
`unsupported_call_abi` is a known supported shape.

**Current fact:**

- Sidecar has 4 functions.
- 3 are already `register`.
- 1 remains `stack` due `unsupported_call_abi`.

**Approach:**

1. Inspect the remaining stack function and call detail.
2. Decide if it is:
   - same `inout` writer family;
   - task/spawn/join ABI;
   - another unsupported call shape.
3. If task runtime ABI is involved, defer until actor/task runtime memory work is done.

**Done when:**

- either row becomes register with exact ABI evidence;
- or blocker becomes more precise than generic `unsupported_call_abi`.

### 5.8. Task 1F: Region/island primitive split

**Goal:** stop treating every region/island operation as generic unsupported runtime effect.

**Write files likely involved:**

- `compiler/internal/buildreports/backend.go`
- `compiler/internal/machine/`
- `compiler/internal/backend/x64core/`
- `compiler/internal/allocplan/`
- memory/domain evidence ingestion in `tools/internal/localbenchmarktier1/metadata.go`

**Approach:**

1. Inspect `region_island_allocation_tetra.backend.json` and PLIR.
2. Split cases:
   - local region allocation that can lower to stack/domain primitive;
   - domain accounting operation;
   - true runtime call.
3. Add report categories:
   - exact domain primitive supported;
   - domain primitive unsupported;
   - true runtime effect call.
4. Implement one primitive only after tests.
5. Keep domain bytes visible in memory evidence.

**Done when:**

- blocker is no longer vague;
- if row becomes register, sidecar proves real primitive support;
- heap stays zero.

### 5.9. Task 1G: Composite control-flow backend

**Rows:** `slice_sum_tetra`, `hash_table_tetra`, `matrix_multiply_tetra`.

**Important decision:** do not use benchmark-source reshaping as the main solution.
P76 found source-level helper extraction for `slice_sum_tetra`, but that would not close row-level
fallback if `main` remains fallback. It can be a later non-claim experiment, not the best main path.

**Goal:** teach backend to compose already-safe machine pieces into larger structured functions.

**Recommended order:**

1. `slice_sum_tetra.main`
2. `hash_table_tetra.main`
3. `matrix_multiply_tetra.main`

**Why this order:**

- slice has one local array, fill loop, repeated sum loop, final branch;
- hash has two arrays, fill loop, helper call, final branch;
- matrix has nested loops and higher register pressure.

**Approach:**

1. Read exact `*.backend.json`, `*.plir.txt`, `*.bounds.json`, `*.alloc.json`.
2. Define a structured machine-function composer:
   - block labels;
   - unconditional branch;
   - conditional branch;
   - loop-carried locals;
   - stack/local array reads and writes;
   - safe helper call only if call ABI is already supported.
3. Add strict shape recognizers first.
4. Add negative tests:
   - unknown trip count;
   - missing bounds proof;
   - escaping local array;
   - unsupported call inside loop;
   - nested control flow beyond the implemented pattern.
5. Emit x64 only for the exact recognized shape.
6. Keep vectorization separate from native backend support.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-composite-control-flow go test \
  ./compiler/internal/machine \
  ./compiler/internal/backend/x64core \
  ./compiler/internal/buildreports \
  ./compiler \
  -run 'SliceSum|Hash|Matrix|ControlFlow|Loop|Backend|Register' -count=1

GOCACHE=$(pwd)/.cache/go-build-composite-control-flow go clean -cache
```

**Done when:**

- one target row leaves fallback;
- sidecar shows every function required by the row is register;
- heap/bounds remain zero;
- near-miss shapes still fallback.

### 5.10. Task 1H: Actor backend tail

**Goal:** do not promote actor row until actor memory is production-ready.

**Reason:**

- `actor_ping_pong_tetra` is not only a backend row.
- It is also the proof that actor mailbox/message/domain memory is real.

**Approach:**

1. Finish Track 4 first.
2. Inspect exact runtime calls:
   - actor spawn;
   - actor send;
   - actor receive;
   - actor drop/finish;
   - task join if present.
3. Define exact ABI for runtime calls.
4. Add tests for supported runtime-call ABI and unsupported runtime-call fallback.

**Done when:**

- actor row leaves fallback only after production actor memory sidecar evidence exists;
- copied/moved/domain bytes remain visible.

## 6. Track 2: Bounds-check elimination

### 6.1. Current state

All 17 Tetra rows currently have `bounds_left == 0`.

So this track is no longer "remove remaining checks".
It is "never let checks silently return".

### 6.2. Task 2A: Validator zero-bounds gate

**Goal:** validator must fail if a current Tier 1 row regresses to `bounds_left > 0`.

**Files:**

- `tools/cmd/validate-local-benchmark-tier1/main.go`
- `tools/cmd/validate-local-benchmark-tier1/evidence_validation.go`
- validator fixture files if this repo uses them for this command
- `tools/internal/localbenchmarktier1/metadata.go`

**Approach:**

1. Add fixture/report with one Tetra row at `bounds_left=1`.
2. Validator must reject it unless an explicit exception policy exists.
3. Add fixture/report where bounds report is missing.
4. Validator must reject missing proof evidence for a zero-bounds claim.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-bounds-gate go test \
  ./tools/cmd/validate-local-benchmark-tier1 \
  ./tools/internal/localbenchmarktier1 \
  -run 'Bounds|Proof|Evidence|Validate' -count=1

GOCACHE=$(pwd)/.cache/go-build-bounds-gate go clean -cache
```

**Done when:**

- fake zero-bounds reports fail;
- real P72 report still validates.

### 6.3. Task 2B: Proof hygiene

**Goal:** every eliminated check remains explainable.

**Files:**

- `compiler/internal/lower/rangeproof/`
- `compiler/internal/rangeproof/`
- `compiler/internal/buildreports/bounds.go`
- `compiler/internal/lower/lower_suite_test.go`
- `compiler/internal/machine/bounds/`

**Approach:**

1. Audit proof families already used:
   - constant length;
   - modulo capacity;
   - affine loop;
   - call-boundary length contract;
   - helper offset contract.
2. Ensure bounds reports expose proof IDs or equivalent metadata.
3. Add negative tests for each proof family.
4. Keep unchecked lowering impossible without proof.

**Done when:**

- each zero-bounds row has inspectable proof evidence;
- invalid proof shapes fail tests.

### 6.4. Task 2C: Backend promotion must preserve BCE

**Goal:** Track 1 must not reintroduce bounds checks.

**Approach:**

1. Every backend worker records before/after `bounds_left`.
2. If a row becomes `register` but `bounds_left` increases, worker is incomplete.
3. Native emitter must use proof-checked indexed ops only when proof exists.

**Done when:**

- every native promotion packet includes a bounds invariant.

## 7. Track 3: Heap rows and lifetime evidence

### 7.1. Current state

The old 5 heap rows are currently zero heap:

### Slice Sum

- Old row: `slice_sum_tetra`.
- Old blocker: `heap.required_large_object`.
- Current duty: prevent regression.

### Bounds Check Loops

- Old row: `bounds_check_loops_tetra`.
- Old blocker: `heap.required_large_object`.
- Current duty: prevent regression.

### JSON Parse/Stringify

- Old row: `json_parse_stringify_tetra`.
- Old blocker: `heap.required_unknown_call`.
- Current duty: preserve noescape/caller-owned facts.

### HTTP Plaintext JSON

- Old row: `http_plaintext_json_tetra`.
- Old blocker: `heap.required_unknown_call`.
- Current duty: preserve noescape/caller-owned facts.

### PostgreSQL Single/Multiple Update

- Old row: `postgresql_single_multiple_update_tetra`.
- Old blocker: `heap.required_unknown_call`.
- Current duty: preserve P72 writer facts.

### 7.2. Task 3A: Zero-heap validator gate

**Goal:** zero heap must be enforced, not trusted by convention.

**Files:**

- `tools/internal/heaptelemetry/heaptelemetry.go`
- `tools/internal/heaptelemetry/heaptelemetry_test.go`
- `tools/cmd/validate-local-benchmark-tier1/main.go`
- `tools/cmd/validate-local-benchmark-tier1/evidence_validation.go`
- `tools/internal/localbenchmarktier1/metadata.go`

**Approach:**

1. Add fixture where a current Tetra row has `heap_allocations > 0`.
2. Add fixture with `heap.required_unknown_call`.
3. Add fixture with missing runtime heap sidecar.
4. Validator must reject them for rows that are now required zero-heap.
5. Keep exception mechanism explicit if future benchmark needs heap.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-zero-heap-gate go test \
  ./tools/internal/heaptelemetry \
  ./tools/internal/localbenchmarktier1 \
  ./tools/cmd/validate-local-benchmark-tier1 \
  -run 'Heap|ZeroHeap|UnknownCall|MemoryEvidence|Validate' -count=1

GOCACHE=$(pwd)/.cache/go-build-zero-heap-gate go clean -cache
```

**Done when:**

- fake heap-positive report fails;
- P72 report passes;
- missing heap evidence cannot be mistaken for zero heap.

### 7.3. Task 3B: Large local storage stability

**Goal:** large local arrays stay stack/region/local-domain when no escape is proven.

**Files:**

- `compiler/internal/allocplan/`
- `compiler/internal/lower/`
- `compiler/internal/memoryfacts/`
- `compiler/internal/backend/x64core/`

**Approach:**

1. Keep tests for `core.make_i32` and `core.make_u8` noescape local arrays.
2. Add negative tests:
   - returned slice;
   - stored globally;
   - passed to unknown escaping call;
   - unknown length;
   - lifetime longer than owning frame/domain.
3. Ensure negative cases still use heap or report exact blocker.

**Done when:**

- `heap.required_large_object` cannot silently return for slice/bounds rows.

### 7.4. Task 3C: Unknown call noescape stability

**Goal:** JSON/HTTP/PostgreSQL stay zero-heap because lifetime facts are real.

**Approach:**

1. Track `inout []u8` as caller-owned output, not heap escape.
2. Track helper summaries:
   - no allocation;
   - writes into caller buffer;
   - scalar return;
   - no retained reference;
   - no returned borrowed invalid view.
3. Add near-miss tests:
   - helper stores buffer in global;
   - helper returns borrowed view that escapes;
   - helper calls unknown external function;
   - helper allocates internally.

**Done when:**

- ABI backend work does not erase memory/lifetime evidence;
- unknown call heap reason stays available for unsafe cases.

## 8. Track 4: Production actor memory

### 8.1. Ціль

Actor memory має бути не "гарною ідеєю в моделі", а runtime evidence:

- кожен actor має memory domain;
- mailbox має current/peak bytes;
- message slabs мають current/peak bytes;
- owned regions мають current/peak bytes;
- copy/move bytes рахуються;
- budget enforced;
- backpressure visible;
- Tier 1 actor row бере ці дані з production runtime sidecar.

### 8.2. Task 4A: Runtime actor memory structs

**Files:**

- `compiler/internal/actorsrt/actorsrt_core.go`
- `compiler/internal/actorsrt/actorsrt_suite_test.go`

**Fields:**

- `actor_id`;
- `mailbox_current_bytes`;
- `mailbox_peak_bytes`;
- `message_current_bytes`;
- `message_peak_bytes`;
- `owned_region_current_bytes`;
- `owned_region_peak_bytes`;
- `bytes_copied`;
- `bytes_moved`;
- `budget_bytes`;
- `over_budget_count`;
- `backpressure_events`.

**Approach:**

1. Add pure runtime counters first.
2. Do not wire benchmark ingestion yet.
3. Tests should create actors, send messages, receive messages, drop payloads.
4. Prove current and peak behavior.

**Done when:**

- `actorsrt` tests can inspect counters without benchmark runner.

### 8.3. Task 4B: Mailbox/message byte accounting

**Approach:**

1. Define message size calculation:
   - header bytes;
   - payload bytes;
   - copied payload bytes;
   - moved owned-region bytes.
2. On send:
   - receiver mailbox current bytes increases;
   - peak updates if needed;
   - copied/moved counters update.
3. On receive:
   - mailbox current bytes decreases;
   - owned bytes update if actor retains data.
4. On drop:
   - owned bytes decreases.

**Done when:**

- tests prove send/receive/drop accounting.

### 8.4. Task 4C: Byte budget and backpressure

**Approach:**

1. Add per-actor memory budget.
2. Define behavior when send exceeds budget:
   - reject with explicit status; or
   - yield/wait; or
   - queue only if policy permits.
3. Record backpressure events.
4. Receiving/dropping must free budget.
5. Zero-copy move changes owner, not copy count.

**Done when:**

- under-budget send succeeds;
- over-budget send triggers designed behavior;
- backpressure counters are visible;
- budget is released after receive/drop.

### 8.5. Task 4D: Actor memory sidecar schema

**Files:**

- `tools/internal/heaptelemetry/heaptelemetry.go`
- `tools/internal/heaptelemetry/heaptelemetry_test.go`
- `tools/internal/localbenchmarktier1/metadata.go`
- `tools/cmd/validate-local-benchmark-tier1/main.go`

**Approach:**

1. Extend or reuse `domain_bytes` with actor-specific fields.
2. Add sidecar sample for actor runtime counters.
3. Validate:
   - actor domain exists;
   - mailbox/message fields exist;
   - copy/move fields exist;
   - budget/backpressure fields exist.
4. Make missing actor evidence fail for actor row.

**Done when:**

- actor row has production actor memory evidence in report;
- validator rejects missing actor memory fields.

### 8.6. Task 4E: Actor row integration

**Approach:**

1. Run fresh Tier 1.
2. Check `actor_ping_pong_tetra`:
   - heap stays zero;
   - bounds stays zero;
   - actor domain bytes exist;
   - budget/backpressure fields exist;
   - backend remains fallback unless Track 1H also proves native path.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-actor-memory go test \
  ./compiler/internal/actorsrt \
  ./compiler/internal/parallelrt \
  ./tools/internal/heaptelemetry \
  ./tools/internal/localbenchmarktier1 \
  ./tools/cmd/validate-local-benchmark-tier1 \
  -run 'Actor|Mailbox|Message|Budget|Backpressure|Domain|Bytes' -count=1

GOCACHE=$(pwd)/.cache/go-build-actor-memory go clean -cache
```

**Done when:**

- actor memory is production runtime evidence, not only benchmark metadata.

## 9. Track 5: RSS reduction

### 9.1. Ціль

RSS не може чесно бути "0 bytes" для Linux process.
RSS включає:

- loader;
- executable pages;
- runtime object;
- stack pages;
- mapped pages;
- libc/syscall support if linked;
- telemetry process sampling overhead;
- page rounding.

Тому правильна ціль:

- зменшувати RSS floor;
- прибирати непотрібні runtime pieces;
- робити row-specific local regression gates;
- не робити cross-machine claim.

### 9.2. Task 5A: RSS floor audit

**Files:**

- `tools/internal/rsstelemetry/rsstelemetry.go`
- `tools/internal/rsstelemetry/process_linux.go`
- `tools/internal/localbenchmarktier1/command.go`
- current `*.backend.json` runtime object plan fields

**Approach:**

1. For tiny rows record:
   - binary size;
   - runtime features required;
   - runtime features linked;
   - runtime features initialized;
   - RSS current;
   - RSS peak;
   - runtime object plan.
2. Produce artifact under `.workflow/post-zero-heap-native-memory/verification/`.
3. Do not claim exact component attribution unless measured.

**Done when:**

- each RSS optimization target has evidence.

### 9.3. Task 5B: Minimal runtime object linking

**Files:**

- `compiler/internal/buildruntime/runtime_object.go`
- `compiler/internal/buildruntime/runtime_object_plan.go`
- `compiler/internal/buildruntime/runtime_usage.go`
- `compiler/internal/buildruntime/selection.go`
- `compiler/internal/buildruntime/tests/`
- `compiler/internal/buildruntime/linuxrt/`
- `compiler/internal/buildruntime/actors/`

**Approach:**

1. For scalar rows, verify no actor/http/postgres pieces are linked.
2. Split runtime features:
   - base syscall/minimal;
   - heap telemetry;
   - RSS telemetry support if needed;
   - actor runtime;
   - task runtime;
   - HTTP runtime;
   - PostgreSQL runtime;
   - domain telemetry.
3. Make feature selection explicit in runtime object plan.
4. Add tests that scalar rows link only minimal required features.
5. Do not remove telemetry required for evidence.

**Done when:**

- tiny rows show minimal runtime feature set;
- validator can see required/linked/initialized feature lists.

### 9.4. Task 5C: Lazy runtime initialization

**Approach:**

1. Avoid actor/task/http/postgres init for rows that do not use them.
2. Avoid large static buffers in generic startup.
3. Initialize telemetry only when benchmark/evidence build needs it.
4. Keep startup correctness tests.

**Done when:**

- startup rows have lower or stable RSS;
- runtime object evidence explains why.

### 9.5. Task 5D: Row-specific RSS policy

**Files:**

- `tools/cmd/validate-local-benchmark-tier1/main.go`
- `reports/benchmark-vnext-memory-baseline/*/rss-budget-policy.local.json`

**Approach:**

1. Generate local RSS policy from a pinned host profile:
   - GOOS;
   - GOARCH;
   - CPU;
   - git commit;
   - per-row RSS peak.
2. Add allowed variance per row.
3. Validator must fail when row exceeds budget.
4. Policy must include non-claims:
   - local only;
   - not comparable across machines;
   - not a language semantics claim.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-policy go test \
  ./tools/internal/rsstelemetry \
  ./tools/internal/localbenchmarktier1 \
  ./tools/cmd/validate-local-benchmark-tier1 \
  ./compiler/internal/buildruntime/... \
  -run 'RSS|Budget|Policy|RuntimeObject|Feature' -count=1

GOCACHE=$(pwd)/.cache/go-build-rss-policy go clean -cache
```

**Done when:**

- RSS regression fails locally;
- final report does not claim cross-machine RSS.

## 10. Benchmark and report discipline

Run fresh Tier 1 after any change to:

- `backend_path`;
- `backend_blockers`;
- `heap_allocations`;
- `heap_reason_codes`;
- `bounds_left`;
- actor domain evidence;
- RSS policy fields;
- runtime feature linking fields.

Fresh report command:

```sh
RSS_POLICY_DIR=reports/benchmark-vnext-memory-baseline
RSS_POLICY="$RSS_POLICY_DIR/tier1-after-memory-zero-heap-optimization/rss-budget-policy.local.json"

GOCACHE=$(pwd)/.cache/go-build-tier1-native-memory \
  go run ./tools/cmd/local-benchmark-tier1 \
  --out-dir reports/benchmark-vnext-memory-baseline/tier1-<slug> \
  --iterations 3

GOCACHE=$(pwd)/.cache/go-build-tier1-native-memory \
  go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report reports/benchmark-vnext-memory-baseline/tier1-<slug>/report.json \
  --rss-budget-policy "$RSS_POLICY"

GOCACHE=$(pwd)/.cache/go-build-tier1-native-memory go clean -cache
```

For final report use `--iterations 5` if runtime is acceptable.

## 11. Final integrated gate

```sh
FINAL_DIR=reports/benchmark-vnext-memory-baseline/tier1-native-memory-final
FINAL_REPORT="$FINAL_DIR/report.json"
FINAL_POLICY="$FINAL_DIR/rss-budget-policy.local.json"

GOCACHE=$(pwd)/.cache/go-build-native-memory-final \
  go test \
  ./compiler/internal/allocplan \
  ./compiler/internal/lower \
  ./compiler/internal/buildreports \
  ./compiler/internal/backend/x64abi \
  ./compiler/internal/backend/x64core \
  ./compiler/internal/machine \
  ./compiler/internal/actorsrt \
  ./compiler/internal/parallelrt \
  ./compiler/internal/buildruntime/... \
  ./tools/internal/heaptelemetry \
  ./tools/internal/rsstelemetry \
  ./tools/internal/localbenchmarktier1 \
  ./tools/cmd/local-benchmark-tier1 \
  ./tools/cmd/validate-local-benchmark-tier1 \
  -count=1

GOCACHE=$(pwd)/.cache/go-build-native-memory-final \
  go run ./tools/cmd/local-benchmark-tier1 \
  --out-dir reports/benchmark-vnext-memory-baseline/tier1-native-memory-final \
  --iterations 5

GOCACHE=$(pwd)/.cache/go-build-native-memory-final \
  go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report "$FINAL_REPORT" \
  --rss-budget-policy "$FINAL_POLICY"

graphify update .
git diff --check
GOCACHE=$(pwd)/.cache/go-build-native-memory-final go clean -cache
```

If final RSS policy is not generated yet, final status is `PARTIAL`.

## 12. Найкращий наступний крок

Найкращий next step після P72/P76:

```text
P77: JSON/HTTP ABI feasibility
```

Чому:

- P72 уже закрив PostgreSQL exact `inout []u8` writer.
- JSON/HTTP мають схожий writer pattern.
- Це може прибрати два важливі fallback rows або дати точний ABI blocker.
- Це краще, ніж source-level split `slice_sum_tetra`, бо source split не
  закриває row-level fallback.

P77 має бути read-only:

1. Порівняти JSON/HTTP writers із PostgreSQL P72.
2. Вирішити, чи є safe worker.
3. Якщо safe worker є, назвати exact files/tests.
4. Якщо safe worker немає, зафіксувати ABI blocker і перейти до region/island
   або control-flow composer design.

## 13. Речі, які не робити

Не робити:

- не міняти report JSON вручну;
- не relabel-ити fallback у register;
- не розширювати generic ABI без negative tests;
- не прибирати bounds checks без proof;
- не вимикати heap telemetry, щоб отримати zero heap;
- не називати RSS "bytes memory of language";
- не claim-ити production actor memory без production runtime counters;
- не змішувати benchmark-source reshaping з compiler backend capability;
- не підключати GitHub Actions без окремого дозволу.

## 14. Короткий execution checklist

Для кожного packet:

- [ ] Названий один target row/function.
- [ ] Є current sidecar/report/source evidence.
- [ ] Є RED tests.
- [ ] Є minimal implementation.
- [ ] Є targeted tests.
- [ ] Є fresh Tier 1, якщо змінилась classification.
- [ ] Є validator run.
- [ ] Є `graphify update .` після code changes.
- [ ] Є `git diff --check`.
- [ ] Є workflow evidence artifact.
- [ ] Є non-claims.
- [ ] Status залишається `PARTIAL`, якщо хоч один final gate ще не закритий.
