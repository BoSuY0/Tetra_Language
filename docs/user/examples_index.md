# Examples Index

Status: release-covered examples index. Validate with:

```sh
./tetra smoke --list --format=json > reports/smoke-list-linux-x64.json
go run ./tools/cmd/validate-example-index --smoke-list reports/smoke-list-linux-x64.json --index docs/user/examples_index.md
```

| Example | Purpose | Target group | Expected behavior |
| --- | --- | --- | --- |
| `examples/islands_hello.tetra` | Minimal island program. | native | exits 0 |
| `examples/islands_i32.tetra` | Island integer access. | native | exits 55 |
| `examples/islands_overflow.tetra` | Island bounds diagnostic smoke. | native | exits 1 |
| `examples/mmio_smoke.tetra` | MMIO builtin smoke. | native | exits 123 |
| `examples/cap_mem_smoke.tetra` | Memory capability smoke. | native | exits 77 |
| `examples/memset_smoke.tetra` | Memory set helper smoke. | native | exits 88 |
| `examples/actors_pingpong.tetra` | Actor ping-pong runtime smoke. | native | exits 0 |
| `examples/flow_hello.tetra` | Minimal canonical Flow program. | native wasm | exits 0 |
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
| `examples/core_math_smoke.tetra` | Stable core math module smoke. | native | exits 42 |
| `examples/core_memory_smoke.tetra` | Stable core memory module smoke. | native | exits 42 |
| `examples/extension_smoke.tetra` | Extension method smoke. | native | exits 42 |
| `examples/generic_smoke.tetra` | Generic function smoke. | native | exits 42 |
| `examples/protocol_impl_smoke.tetra` | Protocol implementation smoke. | native | exits 42 |
| `examples/projects/dogfood_cli/src/main.tetra` | Dogfood CLI project build smoke. | native | exits 0 |
| `examples/projects/dogfood_actor_task/src/main.tetra` | Dogfood actor/task project smoke. | native | exits 0 |
| `examples/ui_web_smoke.tetra` | UI metadata web smoke. | wasm | build-only or exits 0 |
| `examples/projects/dogfood_wasi/src/main.tetra` | Dogfood WASI project smoke. | wasm | build-only or exits 0 |
| `examples/projects/dogfood_web_ui/src/main.tetra` | Dogfood web UI project smoke. | wasm | build-only or exits 0 |

