# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:b7cdf34de1eda56ca8ebf2f6795660d3081bf5ae40581c1e3e6eb8695e655007","module_count":62,"entry_count":113} -->

## examples/actors_pingpong.tetra

### Functions

- `func pong() -> i32 uses actors`
- `func main() -> i32 uses actors`

## examples/actors_tagged_stress.tetra

### Globals

- `val ITERATIONS: i32`

### Functions

- `func worker() -> Int uses actors`
- `func main() -> Int uses actors`

## examples/async_smoke.tetra

### Functions

- `async func answer() -> Int`
- `async func caller() -> Int`
- `func main() -> Int`

## examples/bool_smoke.tetra

### Functions

- `func main() -> Int`

## examples/cap_mem_ptr_smoke.tetra

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples/cap_mem_smoke.tetra

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples/compound_assignment_smoke.tetra

### Functions

- `func main() -> Int`

## examples/const_bool_smoke.tetra

### Globals

- `const enabled`

### Functions

- `func main() -> Int`

## examples/const_smoke.tetra

### Globals

- `const base: i32`
- `const delta`

### Functions

- `func main() -> Int`

## examples.core_async_smoke

### Functions

- `async func core_async_probe() -> Int`
- `func main() -> Int`

## examples.core_collections_smoke

### Functions

- `func main() -> Int uses alloc, mem`

## examples.core_crypto_smoke

### Functions

- `func main() -> Int uses alloc, mem`

## examples.core_filesystem_smoke

### Functions

- `func main() -> Int`

## examples.core_io_smoke

### Functions

- `func main() -> Int uses alloc, capability, io, mem, mmio`

## examples.core_math_smoke

### Functions

- `func main() -> Int`

## examples.core_memory_smoke

### Functions

- `func main() -> Int uses alloc, capability, mem`

## examples.core_networking_smoke

### Functions

- `func main() -> Int`

## examples.core_serialization_smoke

### Functions

- `func main() -> Int uses alloc, mem`

## examples.core_slices_smoke

### Functions

- `func main() -> Int uses alloc, islands, mem`

## examples.core_strings_smoke

### Functions

- `func main() -> Int`

## examples.core_sync_smoke

### Functions

- `func main() -> Int`

## examples.core_testing_smoke

### Functions

- `func main() -> Int`

## examples.core_time_smoke

### Functions

- `func main() -> Int`

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

- `func main() -> Int uses actors`

## examples/effects_io_smoke.tetra

### Functions

- `func main() -> Int uses io`

## examples/effects_mem_smoke.tetra

### Functions

- `func main() -> Int uses alloc, capability, mem`

## examples/else_if_smoke.tetra

### Functions

- `func main() -> Int`

## examples/enum_exhaustive_match_smoke.tetra

### Enums

- `Color`: red, green

### Functions

- `func main() -> Int`

## examples/enum_match_smoke.tetra

### Enums

- `Color`: red, green, blue

### Functions

- `func main() -> Int`

## examples.experimental_math_smoke

### Functions

- `func main() -> i32`

## examples.experimental_memcpy_smoke

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples/extension_smoke.tetra

### Structs

- `Vec2`
  - `x: Int`
  - `y: Int`

### Functions

- `func main() -> Int`

### Extensions

- `Vec2`
  - `func Vec2.sum(self: Vec2) -> Int`

## examples/flow_hello.tetra

### Functions

- `func main() -> Int uses io`

## examples/flow_islands_smoke.tetra

### Functions

- `func main() -> Int uses alloc, io, islands, mem`

## examples/flow_struct_smoke.tetra

### Structs

- `Vec2`
  - `x: Int`
  - `y: Int`

### Functions

- `func main() -> Int`

## examples/flow_unsafe_cap_mem_smoke.tetra

### Functions

- `func main() -> Int uses alloc, capability, mem`

## examples/for_collection_smoke.tetra

### Functions

- `func main() -> Int uses alloc, islands, mem`

## examples/for_collection_u8_smoke.tetra

### Functions

- `func main() -> Int uses alloc, islands, mem`

## examples/for_range_smoke.tetra

### Functions

- `func main() -> Int`

## examples/generic_smoke.tetra

### Functions

- `func id<T>(x: T) -> T`
- `func main() -> Int`

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

- `func main() -> Int`

## examples/loop_control_smoke.tetra

### Functions

- `func main() -> Int`

## examples/memset_smoke.tetra

### Functions

- `func main() -> i32 uses alloc, capability, mem`

## examples/mmio_smoke.tetra

### Functions

- `func main() -> i32 uses alloc, capability, io, mem, mmio`

## examples/optional_match_smoke.tetra

### Functions

- `func maybe(flag: Bool) -> Int?`
- `func main() -> Int`

## examples/optional_match_some_smoke.tetra

### Functions

- `func maybe(flag: Bool) -> Int?`
- `func main() -> Int`

## examples/optional_smoke.tetra

### Functions

- `func maybe(flag: Bool) -> Int?`
- `func unwrap(value: Int?) -> Int`
- `func main() -> Int`

## examples/ownership_smoke.tetra

### Functions

- `func add_one(x: borrow Int) -> Int`
- `func take(x: consume Int) -> Int`
- `func bump(x: inout Int) -> Int`
- `func main() -> Int`

## examples/protocol_impl_smoke.tetra

### Structs

- `Vec2`
  - `x: Int`

### Protocols

- `protocol Renderable`
  - `func draw(self: Vec2) -> Int`

### Implementations

- `impl Vec2: Renderable`

### Functions

- `func main() -> Int`

### Extensions

- `Vec2`
  - `func Vec2.draw(self: Vec2) -> Int`

## examples.struct_ctor_smoke

### Structs

- `Vec2`
  - `x: i32`
  - `y: i32`

### Functions

- `func main() -> i32`

## examples/task_smoke.tetra

### Functions

- `func worker() -> Int`
- `func main() -> Int uses runtime`

## examples/tooling_tests.tetra

### Tests

- `math`

## examples/typed_errors_smoke.tetra

### Enums

- `ReadError`: eof

### Functions

- `func read(flag: Bool) -> Int throws ReadError`
- `func caller() -> Int throws ReadError`
- `func main() -> Int`

## examples/unary_not_smoke.tetra

### Functions

- `func main() -> Int`

