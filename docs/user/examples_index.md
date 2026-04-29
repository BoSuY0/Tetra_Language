# Examples Index

Status: release-covered examples index. Validate with:

```sh
./tetra smoke --list --format=json > reports/smoke-list-linux-x64.json
go run ./tools/cmd/validate-example-index --smoke-list reports/smoke-list-linux-x64.json --index docs/user/examples_index.md
```

| Example | Purpose | Target group | Expected behavior |
| --- | --- | --- | --- |
| `examples/hello.tetra` | Minimal legacy hello-world program. | wasm | build-only exits 0 contract (excluded from native smoke profile) |
| `examples/islands_hello.tetra` | Minimal island program. | native | exits 0 |
| `examples/islands_i32.tetra` | Island integer access. | native | exits 55 |
| `examples/islands_overflow.tetra` | Island bounds diagnostic smoke. | native | exits 1 |
| `examples/mmio_smoke.tetra` | MMIO builtin smoke. | native | exits 123 |
| `examples/cap_mem_smoke.tetra` | Memory capability smoke. | native | exits 77 |
| `examples/memset_smoke.tetra` | Memory set helper smoke. | native | exits 88 |
| `examples/actors_pingpong.tetra` | Actor ping-pong runtime smoke. | native | exits 0 |
| `examples/actor_sleep_pingpong.tetra` | Actor timer wake smoke. | native | exits 0 |
| `examples/flow_hello.tetra` | Minimal canonical Flow program. | native | exits 0 |
| `examples/flow_struct_smoke.tetra` | Flow struct syntax and field access. | native | exits 42 |
| `examples/flow_islands_smoke.tetra` | Flow syntax with islands. | native | exits 0 |
| `examples/flow_unsafe_cap_mem_smoke.tetra` | Flow unsafe capability memory path. | native | exits 42 |
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
| `examples/compound_assignment_smoke.tetra` | Compound assignment smoke. | native | exits 42 |
| `examples/else_if_smoke.tetra` | Else-if lowering smoke. | native | exits 42 |
| `examples/enum_match_smoke.tetra` | Enum match smoke. | native | exits 42 |
| `examples/enum_exhaustive_match_smoke.tetra` | Exhaustive enum match smoke. | native | exits 42 |
| `examples/effects_io_smoke.tetra` | IO effect declaration smoke. | native wasm | exits 0 |
| `examples/effects_mem_smoke.tetra` | Memory effect declaration smoke. | native | exits 17 |
| `examples/effects_actors_smoke.tetra` | Actor effect declaration smoke. | native | exits 0 |
| `examples/optional_smoke.tetra` | Optional value smoke. | native | exits 42 |
| `examples/optional_match_smoke.tetra` | Optional none match smoke. | native | exits 42 |
| `examples/optional_match_some_smoke.tetra` | Optional some match smoke. | native | exits 42 |
| `examples/ownership_smoke.tetra` | Ownership transfer smoke. | native | exits 42 |
| `examples/typed_errors_smoke.tetra` | Typed error syntax smoke. | native | exits 42 |
| `examples/async_smoke.tetra` | Async and await smoke. | native | exits 42 |
| `examples/task_smoke.tetra` | Task runtime handle smoke. | native | exits 42 |
| `examples/time_sleep_smoke.tetra` | Logical runtime sleep/deadline smoke. | native | exits 0 |
| `examples/task_sleep_deadline_smoke.tetra` | Task sleep deadline ordering smoke. | native | exits 0 |
| `examples/task_join_wait_smoke.tetra` | Task join waiter wake smoke. | native | exits 5 |
| `examples/deadline_aware_waits_smoke.tetra` | Deadline-aware sleep, task join, and actor receive smoke. | native | exits 0 |
| `examples/wait_composition_smoke.tetra` | Poll, yield, timer-ready, tagged receive deadline, and task/timer select smoke. | native | exits 0 |
| `examples/core_math_smoke.tetra` | Stable core math module smoke. | native | exits 42 |
| `examples/core_memory_smoke.tetra` | Stable core memory module smoke. | native | exits 42 |
| `examples/extension_smoke.tetra` | Extension method smoke. | native | exits 42 |
| `examples/generic_smoke.tetra` | Generic function smoke. | native | exits 42 |
| `examples/generic_struct_smoke.tetra` | Generic struct instantiation and field access smoke. | native | exits 42 |
| `examples/protocol_impl_smoke.tetra` | Protocol implementation smoke. | native | exits 42 |
| `examples/projects/hello_t4/src/main.t4` | Minimal project-first `.t4` app with `Capsule.t4`. | native | exits 0 |
| `examples/projects/dogfood_cli/src/main.tetra` | Dogfood CLI project build smoke. | native | exits 0 |
| `examples/projects/dogfood_actor_task/src/main.tetra` | Dogfood actor/task project smoke. | native | exits 0 |
| `examples/projects/eco_dogfood/src/main.tetra` | Eco dogfood project baseline smoke. | native | exits 0 (excluded from linux-x64 smoke profile) |
| `examples/ui_web_smoke.tetra` | UI metadata web smoke. | wasm | build-only or exits 0 |
| `examples/projects/dogfood_wasi/src/main.tetra` | Dogfood WASI project smoke. | wasm | build-only or exits 0 |
| `examples/projects/dogfood_web_ui/src/main.tetra` | Dogfood web UI project smoke. | wasm | build-only or exits 0 |

## Epic 15 Verification Commands

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

## Validator Notes (V020-0751..0800)

- `validate-example-index` (`tetra.release.v0_2_0.examples-index.v1`) validates strict smoke JSON shape, portable `examples/...` paths, and index coverage/exclusion consistency.
- `validate-smoke-list` (`tetra.release.v0_2_0.smoke-list.v1`) enforces deterministic smoke profiles and optional full examples-root assignment checks.
- `validate-test-report` (`tetra.release.v0_2_0.test-report.v1`) enforces deterministic test-report ordering/counts and failure evidence shape.

## Troubleshooting Notes (Epic 15)

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

- `generic`, `protocol`, and `extension` examples are required in native smoke and should fail loudly on semantic regressions.
- Do not classify semantic type-check failures as unsupported unless release docs explicitly defer that feature.

### Safety/runtime examples (`V020-0731..0735`)

- `ownership`, `async`, `task`, `time_sleep`, `task_sleep_deadline`, `task_join_wait`, `deadline_aware_waits`, `actors_pingpong`, and `actor_sleep_pingpong` are native release-covered examples with deterministic exits.
- Scheduling-related nondeterminism is considered broken for these smokes; unsupported status must be documented as an exclusion.

### Memory/capability examples (`V020-0736..0740`)

- `islands_*`, `cap_mem`, `mmio`, and `memset` examples are split between required smoke cases and profile exclusions by design.
- If an excluded example appears as a failing smoke case unexpectedly, verify smoke-list config drift before changing code.

### UI/WASM examples (`V020-0741..0745`)

- `ui_web` and dogfood wasm/web examples are allowed as build-only evidence on wasm targets.
- Native smoke exclusion for wasm-specific examples is expected; compile/link failures on wasm targets remain regressions.

### Project dogfood examples (`V020-0746..0750`)

- `dogfood_cli` and `dogfood_actor_task` are required native smoke entries with exit `0`; failures are regressions.
- `eco_dogfood` is intentionally excluded from linux-x64 smoke profile; local `./tetra run --target linux-x64 examples/projects/eco_dogfood/src/main.tetra` is the fallback check.
