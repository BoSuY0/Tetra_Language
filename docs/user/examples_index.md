# Examples Index

Status: release-covered examples index. The current support boundary is
`docs/spec/current_supported_surface.md`. Validate with:

```sh
./tetra smoke --list --format=json > reports/smoke-list-linux-x64.json
go run ./tools/cmd/validate-example-index --smoke-list reports/smoke-list-linux-x64.json --index docs/user/examples_index.md
```

## Generated Docs Naming Policy

Generated docs may show examples with two spellings. If an example source file
declares `module ...`, generated docs render its dotted module path, such as
`examples.core_math_smoke`. If an example source file has no module declaration,
generated docs render its portable file path, such as `examples/flow_hello.tetra`.

This index always lists repository file paths under `examples/` so smoke-list
validation and release evidence stay portable. When comparing this index with
generated docs, map dotted `examples.*` module names back to their source files
before treating the rendering difference as drift.

| Example | Purpose | Target group | Expected behavior |
| --- | --- | --- | --- |
| `examples/hello.tetra` | Minimal legacy hello-world program. | wasm | build-only exits 0 contract (excluded from native smoke profile) |
| `examples/islands_hello.tetra` | Minimal island program. | native | exits 0 |
| `examples/islands_i32.tetra` | Island integer access. | native | exits 55 |
| `examples/islands_overflow.tetra` | Island bounds diagnostic smoke. | native | exits 1 |
| `examples/islands_double_free.tetra` | Island debug double-free diagnostic smoke. | native debug-only | exits 2 with `--islands-debug`; excluded from normal run smoke |
| `examples/mmio_smoke.tetra` | MMIO builtin smoke. | native | exits 123 |
| `examples/cap_mem_smoke.tetra` | Memory capability smoke. | native | exits 77 |
| `examples/cap_mem_ptr_smoke.tetra` | Pointer load/store through `cap.mem`. | native | exits 77 |
| `examples/memset_smoke.tetra` | Memory set helper smoke. | native | exits 88 |
| `examples/actors_pingpong.tetra` | Actor ping-pong runtime smoke. | native | exits 0 |
| `examples/actor_sleep_pingpong.tetra` | Actor timer wake smoke. | native | exits 0 |
| `examples/actors_decl_spawn.tetra` | Actor declaration spawn target smoke. | native | exits 0 |
| `examples/actors_tagged_stress.tetra` | Tagged actor message stress smoke. | native | exits 0 |
| `examples/flow_hello.tetra` | Minimal canonical Flow program. | native | exits 0 |
| `examples/flow_struct_smoke.tetra` | Flow struct syntax and field access. | native | exits 42 |
| `examples/flow_islands_smoke.tetra` | Flow syntax with islands. | native | exits 0 |
| `examples/flow_unsafe_cap_mem_smoke.tetra` | Flow unsafe capability memory path. | native | exits 42 |
| `examples/flow_grammar_surface_smoke.tetra` | Broad Flow grammar surface and test-block smoke. | native | exits 128 in linux/amd64 compiler evidence; `tetra test` block passes |
| `examples/ui_native_shell_smoke.tetra` | UI metadata native shell smoke. | native | exits 0 |
| `examples/bool_smoke.tetra` | Boolean branch smoke. | native | exits 42 |
| `examples/for_range_smoke.tetra` | Range loop smoke. | native | exits 55 |
| `examples/for_collection_smoke.tetra` | Collection loop smoke. | native | exits 42 |
| `examples/for_collection_u8_smoke.tetra` | Byte collection loop smoke. | native | exits 42 |
| `examples/loop_control_smoke.tetra` | Break and continue control flow. | native | exits 42 |
| `examples/complex_control_flow_smoke.tetra` | Nested control flow coverage. | native | exits 42 |
| `examples/unary_not_smoke.tetra` | Unary boolean negation. | native | exits 42 |
| `examples/const_smoke.tetra` | Global const expression smoke. | native | exits 42 |
| `examples/const_bool_smoke.tetra` | Boolean constant smoke. | native | exits 42 |
| `examples/local_const_smoke.tetra` | Local const binding smoke. | native | exits 42 |
| `examples/globals_smoke.tetra` | Top-level `var`/`val` global storage smoke. | native | exits 49 |
| `examples/compound_assignment_smoke.tetra` | Compound assignment smoke. | native | exits 42 |
| `examples/else_if_smoke.tetra` | Else-if lowering smoke. | native | exits 42 |
| `examples/enum_match_smoke.tetra` | Enum match smoke. | native | exits 42 |
| `examples/enum_exhaustive_match_smoke.tetra` | Exhaustive enum match smoke. | native | exits 42 |
| `examples/enum_payload_smoke.tetra` | Enum payload constructor and match smoke. | native | exits 42 |
| `examples/effects_io_smoke.tetra` | IO effect declaration smoke. | native wasm | exits 0 |
| `examples/effects_mem_smoke.tetra` | Memory effect declaration smoke. | native | exits 17 |
| `examples/effects_actors_smoke.tetra` | Actor effect declaration smoke. | native | exits 0 |
| `examples/optional_smoke.tetra` | Optional value smoke. | native | exits 42 |
| `examples/optional_match_smoke.tetra` | Optional none match smoke. | native | exits 42 |
| `examples/optional_match_some_smoke.tetra` | Optional some match smoke. | native | exits 42 |
| `examples/ownership_smoke.tetra` | Ownership transfer and optional borrow smoke. | native | exits 42 |
| `examples/typed_errors_smoke.tetra` | Typed error syntax smoke. | native | exits 42 |
| `examples/async_smoke.tetra` | Async and await smoke. | native | exits 42 |
| `examples/task_smoke.tetra` | Task runtime handle smoke. | native | exits 42 |
| `examples/time_sleep_smoke.tetra` | Logical runtime sleep/deadline smoke. | native | exits 0 |
| `examples/task_sleep_deadline_smoke.tetra` | Task sleep deadline ordering smoke. | native | exits 0 |
| `examples/task_join_wait_smoke.tetra` | Task join waiter wake smoke. | native | exits 5 |
| `examples/task_group_cancel_smoke.tetra` | Task group cancellation wakes a sleeping child before its timer and returns cancellation error. | native | exits 1 |
| `examples/task_group_lifecycle_smoke.tetra` | Task group open, spawn/join, close, status, and canceled-state lifecycle smoke. | native | exits 42 |
| `examples/task_bounded_stress.tetra` | Bounded cooperative task spawn/join stress smoke. | native | exits 42 |
| `examples/deadline_aware_waits_smoke.tetra` | Deadline-aware sleep, task join, and actor receive smoke. | native | exits 0 |
| `examples/wait_composition_smoke.tetra` | Poll, yield, timer-ready, tagged receive deadline, and task/timer select smoke. | native | exits 0 |
| `examples/ctx_switch_sysv_smoke.tetra` | `core.ctx_switch` SysV x64 stack-switch smoke. | native linux-x64 macos-x64 | exits 66 |
| `examples/ctx_switch_win64_smoke.tetra` | `core.ctx_switch` Win64 stack-switch smoke. | native windows-x64 | exits 66; excluded from linux-x64 smoke profile by target |
| `examples/core_async_smoke.tetra` | Current v0.3.0 core async helper smoke for `select_or`, with `pair_sum` probe coverage kept compile-visible. | native | exits 42 through the deterministic `select_or` path; does not claim broader async runtime coverage |
| `examples/core_capability_smoke.tetra` | Current v0.3.0 core capability token acquisition smoke for `cap.mem` and `cap.io`. | native | exits 42 using only caller-owned heap memory and local MMIO storage; does not imply host permission grant |
| `examples/core_collections_smoke.tetra` | Current v0.3.0 core collections helper smoke for length, contains, count, and first-or behavior. | native | exits 42 |
| `examples/core_crypto_smoke.tetra` | Current v0.3.0 core crypto placeholder smoke for checksum, seed mixing, and equality branches. | native | exits 42; placeholder helpers are not cryptographic primitives |
| `examples/core_filesystem_smoke.tetra` | Current v0.3.0 core filesystem placeholder smoke for path-string helper behavior. | native | exits 42; does not perform host filesystem access |
| `examples/core_http_smoke.tetra` | Current v0.4.0 core HTTP/1.1 String and byte-buffer request-line routing, request-head framing, and response byte-buffer helper smoke for TechEmpower paths. | native | exits 42 using caller-owned heap memory; does not open sockets, parse full request bodies, or talk to PostgreSQL |
| `examples/core_io_smoke.tetra` | Current v0.3.0 core IO capability/MMIO helper smoke. | native | exits 42 using caller-owned local MMIO storage; does not imply host IO permission grant |
| `examples/core_json_smoke.tetra` | Current v0.4.0 core JSON byte-buffer helper smoke for compact response object writing and escaping. | native | exits 42 using caller-owned heap memory; does not perform HTTP or network IO |
| `examples/core_math_smoke.tetra` | Current v0.3.0 core math module smoke for `add_i32`, `min_i32`, `max_i32`, and `clamp_i32`. | native | exits 42 |
| `examples/core_memory_smoke.tetra` | Current v0.3.0 core memory module smoke for capability-bound `memset_u8` and `memcpy_u8`. | native | exits 42 |
| `examples/core_memory_negative_length_smoke.tetra` | Current v0.3.0 core memory negative-length diagnostic smoke for capability-bound `memset_u8` and `memcpy_u8`. | native | exits 2 when both helpers reject negative lengths |
| `examples/core_net_smoke.tetra` | Current v0.4.0 core networking runtime smoke for real linux-x64 TCP socket open, nonblocking mode, `SO_REUSEPORT`, `TCP_NODELAY`, loopback bind/listen, epoll create/add-read/add-read-write/mod-read-write/mod-read/delete/wait-zero/wait-one-into-zero, fd/flag extraction, event predicates, and close helpers; compiler integration separately covers loopback connect plus read/recv/write/send payload exchange. | native linux-x64 | exits 42; does not accept clients, read/write payloads, run a full event-loop abstraction, or talk to PostgreSQL |
| `examples/core_networking_smoke.tetra` | Current v0.3.0 core networking placeholder smoke for port and retry-backoff helpers. | native | exits 42; does not perform network IO |
| `examples/core_postgres_smoke.tetra` | Current v0.4.0 core PostgreSQL wire-frame byte-buffer helper smoke for startup, Simple Query, Terminate, and big-endian length fields. | native | exits 42 using caller-owned heap memory; does not open sockets, authenticate, parse server frames, or pool connections |
| `examples/core_postgres_prepared_smoke.tetra` | Current v0.4.0 core PostgreSQL prepared-statement wire-frame smoke for Parse, Bind, Describe, Execute, Sync, one- and two-parameter text binds, and i16/i32 length fields. | native | exits 42 using caller-owned heap memory; does not open sockets, authenticate, parse server frames, manage prepared statement state, or pool connections |
| `examples/core_postgres_result_smoke.tetra` | Current v0.4.0 core PostgreSQL result-frame smoke for typed frame headers, RowDescription type OIDs, DataRow value offsets/lengths, ASCII integer values, CommandComplete affected rows, and ReadyForQuery status bytes. | native | exits 42 using caller-owned heap memory; does not open sockets, authenticate, own connection state, manage prepared statements, or pool connections |
| `examples/core_serialization_smoke.tetra` | Current v0.3.0 core serialization helper smoke for byte-pair packing and checksum behavior. | native | exits 42 |
| `examples/core_slices_smoke.tetra` | Current v0.3.0 core slices helper smoke for `sum_i32`, `weighted_sum_i32`, and `sum_u8`. | native | exits 42 |
| `examples/core_strings_smoke.tetra` | Current v0.3.0 core strings helper smoke for `ascii_len`, `ascii_sum`, and `is_empty`. | native | exits 42 |
| `examples/core_sync_smoke.tetra` | Current v0.3.0 core sync helper smoke for status merge, countdown, barrier target, and readiness behavior. | native | exits 42 |
| `examples/core_testing_smoke.tetra` | Current v0.3.0 core testing helper smoke for assertion status composition. | native | exits 42 |
| `examples/core_time_smoke.tetra` | Current v0.3.0 core time helper smoke for deterministic duration arithmetic. | native | exits 42; does not claim wall-clock runtime behavior |
| `examples/experimental_math_smoke.tetra` | Experimental stdlib math mirror smoke; evidence only, not a stable support claim. | native | experimental evidence only; Excluded from linux-x64 smoke profile; exits 42 in linux/amd64 compiler test evidence |
| `examples/experimental_memcpy_smoke.tetra` | Experimental stdlib memory mirror memcpy/memset smoke; evidence only, not a stable support claim. | native | experimental evidence only; Excluded from linux-x64 smoke profile; exits 93 in linux/amd64 compiler test evidence |
| `examples/extension_smoke.tetra` | Extension method smoke. | native | exits 42 |
| `examples/generic_smoke.tetra` | Generic function smoke. | native | exits 42 |
| `examples/generic_struct_smoke.tetra` | Experimental generic struct smoke; evidence only, not a stable support claim. | native | exits 42 when experimental path is covered |
| `examples/struct_ctor_smoke.tetra` | Call-style struct constructor smoke. | native | exits 94 |
| `examples/protocol_impl_smoke.tetra` | Protocol implementation smoke. | native | exits 42 |
| `examples/tooling_tests.tetra` | Minimal `tetra test` block smoke. | native test-only | `tetra test` passes; no `main`, so excluded from run smoke |
| `examples/projects/hello_t4/src/main.t4` | Minimal project-first `.t4` app with `Capsule.t4`. | native | exits 0 |
| `examples/projects/hello_t4/Capsule.t4` | Project-first capsule manifest for the hello `.t4` app. | native project metadata | declares `src/main.t4`, `src`, `tests`, `linux-x64`, and `io`; not a runnable entry itself |
| `examples/projects/hello_t4/tests/main_test.t4` | Project-first `.t4` test block for `hello_t4`. | native test-only | `tetra test .` passes; no `main`, so excluded from run smoke |
| `examples/projects/dogfood_cli/src/main.tetra` | Dogfood CLI project build smoke. | native | exits 0 |
| `examples/projects/dogfood_actor_task/src/main.tetra` | Dogfood actor/task project smoke. | native | exits 0 |
| `examples/projects/eco_dogfood/src/main.tetra` | Eco dogfood project baseline smoke. | native | exits 0 (excluded from linux-x64 smoke profile) |
| `examples/ui_web_smoke.tetra` | UI metadata web smoke. | wasm | artifact/import preflight by default; runtime exit 0 only with explicit browser gate evidence |
| `examples/projects/dogfood_wasi/src/main.tetra` | Dogfood WASI project smoke. | wasm | artifact/import preflight by default; runtime exit 0 only with explicit runner evidence |
| `examples/projects/dogfood_web_ui/src/main.tetra` | Dogfood web UI project smoke. | wasm | artifact/import preflight by default; runtime exit 0 only with explicit browser gate evidence |

## Excluded from linux-x64 smoke profile

The `./tetra smoke --list --format=json` report also emits `excluded_examples`.
These examples are intentionally outside the default linux-x64 smoke profile, but
remain visible here with the exact exclusion reason reported by the smoke list.

| Example | Reason |
| --- | --- |
| `examples/actors_decl_spawn.tetra` | not part of linux-x64 smoke profile |
| `examples/actors_tagged_stress.tetra` | not part of linux-x64 smoke profile |
| `examples/cap_mem_ptr_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/ctx_switch_sysv_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/ctx_switch_win64_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/enum_payload_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/experimental_math_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/experimental_memcpy_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/flow_grammar_surface_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/generic_struct_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/globals_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/hello.tetra` | not part of linux-x64 smoke profile |
| `examples/islands_double_free.tetra` | not part of linux-x64 smoke profile |
| `examples/projects/dogfood_wasi/src/main.tetra` | not part of linux-x64 smoke profile |
| `examples/projects/dogfood_web_ui/src/main.tetra` | not part of linux-x64 smoke profile |
| `examples/projects/eco_dogfood/src/main.tetra` | not part of linux-x64 smoke profile |
| `examples/projects/hello_t4/Capsule.t4` | not part of linux-x64 smoke profile |
| `examples/projects/hello_t4/src/main.t4` | not part of linux-x64 smoke profile |
| `examples/projects/hello_t4/tests/main_test.t4` | not part of linux-x64 smoke profile |
| `examples/struct_ctor_smoke.tetra` | not part of linux-x64 smoke profile |
| `examples/task_bounded_stress.tetra` | not part of linux-x64 smoke profile |
| `examples/tooling_tests.tetra` | not part of linux-x64 smoke profile |
| `examples/ui_web_smoke.tetra` | not part of linux-x64 smoke profile |

## Epic 14 Verification Commands

```sh
./tetra fmt --check examples
./tetra smoke --list --format=json > reports/smoke-list-linux-x64.json
go run ./tools/cmd/validate-smoke-list --report reports/smoke-list-linux-x64.json --examples-root examples
./tetra test --report=json examples > reports/examples-test-report.json
go run ./tools/cmd/validate-test-report --report reports/examples-test-report.json
go run ./tools/cmd/validate-example-index --smoke-list reports/smoke-list-linux-x64.json --index docs/user/examples_index.md
./tetra run --target linux-x64 examples/projects/dogfood_cli/src/main.tetra
./tetra run --target linux-x64 examples/projects/dogfood_actor_task/src/main.tetra
./tetra run --target linux-x64 examples/projects/eco_dogfood/src/main.tetra
```

## Validator Notes

- Validator schema IDs may retain historical artifact names even when the current
  release profile advances. `validate-example-index`, `validate-smoke-list`, and
  `validate-test-report` enforce strict JSON shape, deterministic smoke profiles,
  and failure evidence shape for the current branch state.

## Troubleshooting Notes (Epic 14)

Use these notes to separate unsupported profile boundaries from real regressions.

### Basic language examples (`V020-0701..0705`)

- `examples/hello.tetra` is intentionally excluded from linux-x64 smoke matrix; this is unsupported profile scope, not a compiler/runtime break.
- If `examples/flow_hello.tetra` or `examples/bool_smoke.tetra` stop compiling/running on native, treat as a regression and rerun `./tetra smoke --list --format=json`.

### Control-flow examples (`V020-0706..0710`)

- Loop/control examples should keep deterministic exits (`42` or `55`) in native smoke; any parser or lowering failure is a regression.
- If only `examples/for_collection_u8_smoke.tetra` fails while others pass, suspect byte-collection semantics rather than global smoke config.

### Const and assignment examples (`V020-0711..0715`)

- `const` and compound assignment failures are regressions when they fail formatting, parsing, or expected exit checks in smoke/test.
- Unsupported behavior should be documented explicitly; silent drift in exit codes is treated as broken behavior.

### Enum/match examples (`V020-0716..0720`)

- `examples/enum_match_smoke.tetra` and `examples/enum_exhaustive_match_smoke.tetra` are required native smoke coverage.
- Missing exhaustiveness diagnostics or changed exit contracts indicate a regression, not an unsupported target limitation.

### Optional/error examples (`V020-0721..0725`)

- `optional` and `typed error` smoke examples are release-covered on native and must keep stable expected exits.
- If only one optional variant fails, verify matcher semantics before changing smoke profiles.

### Generic/protocol/extension examples (`V020-0726..0730`)

- `generic`, `protocol`, and `extension` MVP examples are required in native smoke and should fail loudly on semantic regressions.
- `generic_struct` coverage is experimental evidence only unless the feature registry promotes generic structs to `current`.
- Enum payload constructor/match examples are current only for the narrow current positional match/catch/if-let slice.

### Safety/runtime examples (`V020-0731..0735`)

- `ownership`, `async`, `task`, `time_sleep`, `task_sleep_deadline`, `task_join_wait`, `task_group_cancel`, `deadline_aware_waits`, `actors_pingpong`, and `actor_sleep_pingpong` are native release-covered examples with deterministic exits.
- Scheduling-related nondeterminism is considered broken for these smokes; unsupported status must be documented as an exclusion.

### Memory/capability examples (`V020-0736..0740`)

- `islands_*`, `cap_mem`, `mmio`, and `memset` examples are split between required smoke cases and profile exclusions by design.
- If an excluded example appears as a failing smoke case unexpectedly, verify smoke-list config drift before changing code.

### UI/WASM examples (`V020-0741..0745`)

- `ui_web` and dogfood wasm/web examples are allowed as artifact/import preflight evidence on wasm targets.
- Native smoke exclusion for wasm-specific examples is expected; compile/link failures on wasm targets remain regressions.

### Project dogfood examples (`V020-0746..0750`)

- `dogfood_cli` and `dogfood_actor_task` are required native smoke entries with exit `0`; failures are regressions.
- `eco_dogfood` is intentionally excluded from linux-x64 smoke profile; local `./tetra run --target linux-x64 examples/projects/eco_dogfood/src/main.tetra` is the fallback check.
