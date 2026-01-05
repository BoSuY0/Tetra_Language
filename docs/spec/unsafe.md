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

## Debug Runtime Mode

The build flag `--islands-debug` enables runtime checks for islands:
- double-free detection (exit code 2)
- data page protection to catch use-after-free
