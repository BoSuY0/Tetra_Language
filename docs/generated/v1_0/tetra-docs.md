# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:5cb44c97ab2689809ebba302fed59847c144ae4f76b90be05aab20263e4872b1","module_count":64,"entry_count":139} -->

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

## examples.core_collections_smoke

### Functions

- `func main() -> i32 uses alloc, mem`

## examples.core_crypto_smoke

### Functions

- `func main() -> i32 uses alloc, mem`

## examples.core_filesystem_smoke

### Functions

- `func main() -> i32`

## examples.core_io_smoke

### Functions

- `func main() -> i32 uses alloc, capability, io, mem, mmio`

## examples.core_math_smoke

### Functions

- `func main() -> i32`

## examples.core_memory_smoke

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples.core_networking_smoke

### Functions

- `func main() -> i32`

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

## examples/flow_hello.tetra

### Functions

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

### Functions

- `func add_one(x: borrow i32) -> i32`
- `func take(x: consume i32) -> i32`
- `func bump(x: inout i32) -> i32`
- `func main() -> i32`

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

## examples/task_smoke.tetra

### Functions

- `func worker() -> i32`
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

## examples/ui_native_shell_smoke.tetra

### States

- `state ShellState`
  - `var toggles: i32`
  - `val label: str`

### Views

- `view ShellView(state: ShellState)`
  - `bind toggles: i32`
  - `bind labelText: str`
  - `event submit -> toggle`
  - `command toggle`
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

