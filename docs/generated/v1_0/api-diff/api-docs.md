# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:4f36009e9450d9b077878d165c3ceda83d75923408a8fead580f7a57b140aee3","module_count":91,"entry_count":290} -->

## examples/actor_sleep_pingpong.tetra

### Functions

- `func delayed_pong() -> i32 uses actors, runtime`
- `func main() -> i32 uses actors, runtime`

## examples/actors_decl_spawn.tetra

### Functions

- `func Ponger.run() -> i32 uses actors`
- `func main() -> i32 uses actors`

## examples/actors_pingpong.tetra

### Functions

- `func pong() -> i32 uses actors`
- `func main() -> i32 uses actors`

## examples/actors_tagged_stress.tetra

### Globals

- `val ITERATIONS: i32`

### Functions

- `func worker() -> i32 uses actors`
- `func main() -> i32 uses actors`

## examples/async_smoke.tetra

### Functions

- `async func answer() -> i32`
- `async func caller() -> i32`
- `func main() -> i32`

## examples/bool_smoke.tetra

### Functions

- `func main() -> i32`

## examples/cap_mem_ptr_smoke.tetra

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples/cap_mem_smoke.tetra

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples/complex_control_flow_smoke.tetra

### Enums

- `Mode`: fast, slow

### Functions

- `func classify(x: i32) -> i32`
- `func main() -> i32`

## examples/compound_assignment_smoke.tetra

### Functions

- `func main() -> i32`

## examples/const_bool_smoke.tetra

### Globals

- `const enabled`

### Functions

- `func main() -> i32`

## examples/const_smoke.tetra

### Globals

- `const base: i32`
- `const delta`

### Functions

- `func main() -> i32`

## examples.core_async_smoke

### Functions

- `async func core_async_probe() -> i32`
- `func main() -> i32`

## examples.core_capability_smoke

### Functions

- `func main() -> i32 uses alloc, capability, io, mem, mmio`

## examples.core_collections_smoke

### Functions

- `func main() -> i32 uses alloc, mem`

## examples.core_crypto_smoke

### Functions

- `func main() -> i32 uses alloc, mem`

## examples.core_filesystem_smoke

### Functions

- `func main() -> i32 uses capability, io`

## examples.core_http_smoke

### Functions

- `func main() -> i32 uses alloc, mem`

## examples.core_io_smoke

### Functions

- `func main() -> i32 uses alloc, capability, io, mem, mmio`

## examples.core_json_smoke

### Functions

- `func main() -> i32 uses alloc, mem`

## examples.core_math_smoke

### Functions

- `func main() -> i32`

## examples.core_memory_negative_length_smoke

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples.core_memory_smoke

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples.core_net_smoke

### Functions

- `func main() -> i32 uses alloc, capability, io, mem`

## examples.core_networking_smoke

### Functions

- `func main() -> i32`

## examples.core_postgres_prepared_smoke

### Functions

- `func check_parse() -> i32 uses alloc, mem`
- `func check_bind() -> i32 uses alloc, mem`
- `func check_bind_two_params() -> i32 uses alloc, mem`
- `func check_describe_execute_sync() -> i32 uses alloc, mem`
- `func main() -> i32 uses alloc, mem`

## examples.core_postgres_result_smoke

### Functions

- `func write_world_row_description(dst: inout []u8) -> i32 uses mem`
- `func write_column(dst: inout []u8, start: i32, name: str, type_oid: i32) -> i32 uses mem`
- `func write_world_data_row(dst: inout []u8) -> i32 uses mem`
- `func check_row_description() -> i32 uses alloc, mem`
- `func check_data_row() -> i32 uses alloc, mem`
- `func check_command_and_ready() -> i32 uses alloc, mem`
- `func check_typed_frame_header() -> i32 uses alloc, mem`
- `func main() -> i32 uses alloc, mem`

## examples.core_postgres_smoke

### Functions

- `func main() -> i32 uses alloc, mem`

## examples.core_serialization_smoke

### Functions

- `func main() -> i32 uses alloc, mem`

## examples.core_slices_smoke

### Functions

- `func main() -> i32 uses alloc, islands, mem`

## examples.core_strings_smoke

### Functions

- `func main() -> i32`

## examples.core_sync_smoke

### Functions

- `func main() -> i32`

## examples.core_testing_smoke

### Functions

- `func main() -> i32`

## examples.core_time_smoke

### Functions

- `func main() -> i32`

## examples/ctx_switch_sysv_smoke.tetra

### Globals

- `var g_main_slot: ptr`
- `var g_fiber_slot: ptr`

### Functions

- `func trampoline() -> i32 uses capability, control, mem, runtime`
- `func main() -> i32 uses alloc, capability, control, link, mem, runtime`

## examples/ctx_switch_win64_smoke.tetra

### Globals

- `var g_main_slot: ptr`
- `var g_fiber_slot: ptr`

### Functions

- `func trampoline() -> i32 uses capability, control, mem, runtime`
- `func main() -> i32 uses alloc, capability, control, link, mem, runtime`

## examples/deadline_aware_waits_smoke.tetra

### Functions

- `func delayed_sender() -> i32 uses actors, runtime`
- `func delayed_task() -> i32 uses runtime`
- `func main() -> i32 uses actors, runtime`

## examples/effects_actors_smoke.tetra

### Functions

- `func main() -> i32 uses actors`

## examples/effects_io_smoke.tetra

### Functions

- `func main() -> i32 uses io`

## examples/effects_mem_smoke.tetra

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples/else_if_smoke.tetra

### Functions

- `func main() -> i32`

## examples/enum_exhaustive_match_smoke.tetra

### Enums

- `Color`: red, green

### Functions

- `func main() -> i32`

## examples/enum_match_smoke.tetra

### Enums

- `Color`: red, green, blue

### Functions

- `func main() -> i32`

## examples/enum_payload_smoke.tetra

### Enums

- `CounterMsg`: inc, reset
- `DecodeError`: invalid, eof

### Functions

- `func decode(flag: bool) -> CounterMsg`
- `func error_code(err: DecodeError) -> i32`
- `func bonus(msg: CounterMsg) -> i32`
- `func handle(msg: CounterMsg) -> i32`
- `func main() -> i32`

## examples.experimental_math_smoke

### Functions

- `func main() -> i32`

## examples.experimental_memcpy_smoke

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples/extension_smoke.tetra

### Structs

- `Vec2`
  - `x: i32`
  - `y: i32`

### Functions

- `func main() -> i32`

### Extensions

- `Vec2`
  - `func Vec2.sum(self: Vec2) -> i32`

## examples.flow_grammar_surface_smoke

### Structs

- `Box`
  - `value: i32`

### States

- `state CounterState`
  - `var count: i32`
  - `val title: str`

### Views

- `view CounterView(state: CounterState)`
  - `bind countValue: i32`
  - `bind titleText: str`
  - `event click -> increment`
  - `command increment`
  - `style width: i32`
  - `accessibility label: str`

### Enums

- `Mode`: fast, slow
- `ReadError`: eof

### Protocols

- `protocol Runner`
  - `func run(self: Box) -> i32`

### Globals

- `const answer: i32`

### Implementations

- `impl Box: Runner`

### Functions

- `func id<T>(x: T) -> T`
- `func borrow_one(x: borrow i32) -> i32`
- `func bump(x: inout i32) -> i32`
- `func take(x: consume i32) -> i32`
- `func maybe(flag: bool) -> i32?`
- `func unwrap_match(value: i32?) -> i32`
- `func read(flag: bool) -> i32 throws ReadError`
- `async func async_answer() -> i32`
- `func read_caller(flag: bool) -> i32 throws ReadError`
- `async func async_caller() -> i32`
- `func main() -> i32 uses alloc, budget, capability, io, islands, mem, runtime`

### Extensions

- `Box`
  - `func Box.run(self: Box) -> i32`

### Tests

- `grammar surface`

## examples/flow_hello.tetra

### Functions

- `func greeting() -> str`
- `func main() -> i32 uses io`

## examples/flow_islands_smoke.tetra

### Functions

- `func main() -> i32 uses alloc, io, islands, mem`

## examples/flow_struct_smoke.tetra

### Structs

- `Vec2`
  - `x: i32`
  - `y: i32`

### Functions

- `func main() -> i32`

## examples/flow_unsafe_cap_mem_smoke.tetra

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples/for_collection_smoke.tetra

### Functions

- `func main() -> i32 uses alloc, islands, mem`

## examples/for_collection_u8_smoke.tetra

### Functions

- `func main() -> i32 uses alloc, islands, mem`

## examples/for_range_smoke.tetra

### Functions

- `func main() -> i32`

## examples/generic_smoke.tetra

### Functions

- `func id<T>(x: T) -> T`
- `func main() -> i32`

## examples/generic_struct_smoke.tetra

### Structs

- `Box`
  - `value: T`

### Functions

- `func main() -> i32`

## examples/globals_smoke.tetra

### Globals

- `var g_x: i32`
- `val g_y: i32`
- `var g_p: ptr`
- `val g_null: ptr`

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples/hello.tetra

### Functions

- `func main() -> i32 uses io`

## examples/islands_double_free.tetra

### Functions

- `func alias(isl: island) -> island`
- `func main() -> i32 uses alloc, capability, islands, mem`

## examples/islands_hello.tetra

### Functions

- `func main() -> i32 uses alloc, io, islands, mem`

## examples/islands_i32.tetra

### Functions

- `func main() -> i32 uses alloc, islands, mem`

## examples/islands_overflow.tetra

### Functions

- `func main() -> i32 uses alloc, islands, mem`

## examples/local_const_smoke.tetra

### Functions

- `func main() -> i32`

## examples/loop_control_smoke.tetra

### Functions

- `func main() -> i32`

## examples/memset_smoke.tetra

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples/mmio_smoke.tetra

### Functions

- `func main() -> i32 uses alloc, capability, io, mem, mmio`

## examples/optional_match_smoke.tetra

### Functions

- `func maybe(flag: bool) -> i32?`
- `func main() -> i32`

## examples/optional_match_some_smoke.tetra

### Functions

- `func maybe(flag: bool) -> i32?`
- `func main() -> i32`

## examples/optional_smoke.tetra

### Functions

- `func maybe(flag: bool) -> i32?`
- `func unwrap(value: i32?) -> i32`
- `func main() -> i32`

## examples/ownership_smoke.tetra

### Structs

- `Pair`
  - `left: i32`
  - `right: i32`

### Enums

- `PairMsg`: both, empty

### Protocols

- `protocol Sink`
  - `func sink(self: consume Pair) -> i32`

### Implementations

- `impl Pair: Sink`

### Functions

- `func add_one(x: borrow i32) -> i32`
- `func take(x: consume i32) -> i32`
- `func bump(x: inout i32) -> i32`
- `func optional_ptr_score(value: borrow ptr?) -> i32`
- `func enum_score() -> i32`
- `func main() -> i32`

### Extensions

- `Pair`
  - `func Pair.sink(self: consume Pair) -> i32`

## examples/protocol_impl_smoke.tetra

### Structs

- `Vec2`
  - `x: i32`

### Protocols

- `protocol Renderable`
  - `func draw(self: Vec2) -> i32`

### Implementations

- `impl Vec2: Renderable`

### Functions

- `func main() -> i32`

### Extensions

- `Vec2`
  - `func Vec2.draw(self: Vec2) -> i32`

## examples.struct_ctor_smoke

### Structs

- `Vec2`
  - `x: i32`
  - `y: i32`

### Functions

- `func main() -> i32`

## examples/task_bounded_stress.tetra

### Globals

- `val ITERATIONS: i32`
- `val SEED: i32`
- `val EXPECTED: i32`

### Functions

- `func worker() -> i32`
- `func main() -> i32 uses runtime`

## examples/task_group_cancel_smoke.tetra

### Functions

- `func worker() -> i32 uses runtime`
- `func main() -> i32 uses runtime`

## examples/task_group_lifecycle_smoke.tetra

### Functions

- `func worker() -> i32 uses runtime`
- `func main() -> i32 uses runtime`

## examples/task_join_wait_smoke.tetra

### Functions

- `func worker() -> i32 uses runtime`
- `func main() -> i32 uses runtime`

## examples/task_sleep_deadline_smoke.tetra

### Functions

- `func worker() -> i32 uses runtime`
- `func main() -> i32 uses runtime`

## examples/task_smoke.tetra

### Functions

- `func worker() -> i32`
- `func main() -> i32 uses runtime`

## examples/time_sleep_smoke.tetra

### Functions

- `func main() -> i32 uses runtime`

## examples/tooling_tests.tetra

### Tests

- `math`

## examples/typed_errors_smoke.tetra

### Enums

- `ReadError`: eof

### Functions

- `func read(flag: bool) -> i32 throws ReadError`
- `func caller() -> i32 throws ReadError`
- `func main() -> i32`

## examples/ui_desktop_runtime_smoke.tetra

### States

- `state DesktopState`
  - `var saves: i32`
  - `var title: str`
  - `var name: str`
  - `var selected: str`

### Views

- `view DesktopView(state: DesktopState)`
  - `bind titleText: str`
  - `bind nameText: str`
  - `bind selectedText: str`
  - `event rename -> rename`
  - `event select -> selectSecond`
  - `event save -> save`
  - `command rename`
  - `command selectSecond`
  - `command save`
  - `style width: i32`
  - `style height: i32`
  - `accessibility role: str`
  - `accessibility description: str`

### Functions

- `func main() -> i32`

## examples/ui_native_shell_smoke.tetra

### States

- `state ShellState`
  - `var toggles: i32`
  - `var label: str`
  - `var source: i32`
  - `var textSource: str`

### Views

- `view ShellView(state: ShellState)`
  - `bind toggles: i32`
  - `bind labelText: str`
  - `event submit -> toggle`
  - `event reset -> decrement`
  - `event rename -> rename`
  - `event copy -> copy`
  - `event copyAfterToggle -> copyAfterToggle`
  - `event compound -> compound`
  - `command toggle`
  - `command decrement`
  - `command rename`
  - `command copy`
  - `command copyAfterToggle`
  - `command compound`
  - `style width: i32`
  - `style accent: str`
  - `accessibility role: str`
  - `accessibility description: str`

### Functions

- `func main() -> i32`

## examples/ui_web_smoke.tetra

### States

- `state CounterState`
  - `var count: i32`
  - `val title: str`

### Views

- `view CounterView(state: CounterState)`
  - `bind countValue: i32`
  - `bind titleText: str`
  - `event click -> increment`
  - `command increment`
  - `style width: i32`
  - `style theme: str`
  - `accessibility role: str`
  - `accessibility label: str`

### Functions

- `func main() -> i32`

## examples/unary_not_smoke.tetra

### Functions

- `func main() -> i32`

## examples/wait_composition_smoke.tetra

### Functions

- `func worker() -> i32 uses actors, runtime`
- `func main() -> i32 uses actors, runtime`

## examples/wasm_globals_smoke.tetra

### Globals

- `val greeting: str`
- `var title: str`
- `const base: i32`

### Functions

- `func main() -> i32`

## examples.wasm_multi_return_2_smoke

### Structs

- `Pair`
  - `left: i32`
  - `right: i32`

### Functions

- `func make_pair() -> Pair`
- `func main() -> i32`

## examples.wasm_multi_return_3_smoke

### Enums

- `Packet`: data, empty

### Functions

- `func make_packet() -> Packet`
- `func main() -> i32`

## examples.wasm_multi_return_4_smoke

### Structs

- `Quad`
  - `a: i32`
  - `b: i32`
  - `c: i32`
  - `d: i32`

### Functions

- `func make_quad() -> Quad`
- `func main() -> i32`

