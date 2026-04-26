# Unsafe Blocks

`unsafe { ... }` marks a block that is allowed to perform operations that are not
safe by default. Unsafe code can violate memory safety if misused, so it should
be small, explicit, and well-reviewed.

## What Requires `unsafe`

The following operations are gated behind `unsafe`:

- `core.alloc_bytes`
- `core.island_new`
- `core.island_make_u8` / `core.island_make_i32` when the island is not a tracked
  region (scoped island or region-carrying parameter)
- `free(<island>)` when called explicitly
- `core.cap_io` / `core.cap_mem`
- `core.load_i32` / `core.store_i32`
- `core.load_u8` / `core.store_u8`
- `core.load_ptr` / `core.store_ptr`
- `core.ptr_add`
- `core.mmio_read_i32` / `core.mmio_write_i32`
- `core.sym_addr`
- `core.ctx_switch`

Scoped islands remain safe: `island(size) as isl { ... }` injects `free` automatically.

## Relationship to `uses`

`unsafe` and `uses` are separate gates. `unsafe` marks code that may use
operations outside the safe subset; `uses` declares the effects a function may
perform. For example, raw memory code typically needs both:

```tetra
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let _: Int = core.store_i32(p, 42, mem)
        return core.load_i32(p, mem)
    return 0
```

## Debug Runtime Mode

The build flag `--islands-debug` enables runtime checks for islands:
- double-free detection (exit code 2)
- data page protection to catch use-after-free
