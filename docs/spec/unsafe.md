# Unsafe Blocks

`unsafe { ... }` marks a block that is allowed to perform operations that are not
safe by default. Unsafe code can violate memory safety if misused, so it should
be small, explicit, and well-reviewed.

## Unsafe-Only Builtins Registry

The following operations are gated behind `unsafe`. The generated manifest
records the same policy in each builtin's `unsafe_policy` field, and
`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
checks that this registry stays current.
This registry describes safety policy only; target/backend support can still
vary (for example, build-only WASM targets use a compile-compatible fallback
for island IR paths rather than native island runtime semantics).

| Builtin | Unsafe policy | Required effects | Capability argument |
| --- | --- | --- | --- |
| `core.alloc_bytes` | always | `alloc`, `mem` | none |
| `core.island_new` | always | `alloc`, `islands`, `mem` | none |
| `core.island_make_u8` | conditional when the island is not a tracked scoped island | `alloc`, `islands`, `mem` | none |
| `core.island_make_u16` | conditional when the island is not a tracked scoped island | `alloc`, `islands`, `mem` | none |
| `core.island_make_i32` | conditional when the island is not a tracked scoped island | `alloc`, `islands`, `mem` | none |
| `core.island_make_bool` | conditional when the island is not a tracked scoped island | `alloc`, `islands`, `mem` | none |
| explicit `free(<island>)` | always | `islands`, `mem` | island handle |
| `core.cap_io` | always | `capability`, `io` | returns `cap.io` |
| `core.cap_mem` | always | `capability`, `mem` | returns `cap.mem` |
| `core.load_i32` / `core.store_i32` | always | `mem` | `cap.mem` |
| `core.load_u8` / `core.store_u8` | always | `mem` | `cap.mem` |
| `core.load_ptr` / `core.store_ptr` | always | `mem` | `cap.mem` |
| `core.ptr_add` | always | `mem` | `cap.mem` |
| `core.mmio_read_i32` / `core.mmio_write_i32` | always | `io`, `mmio` | `cap.io` |
| `core.sym_addr` | always | `link` | none |
| `core.ctx_switch` | always | `control`, `runtime` | `cap.mem` |

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

## Epic 06 coverage

Unsafe policy is covered by the release-blocking safety slice:

```sh
go test ./compiler/... -run "Effect|Uses|Capability|Unsafe|Ownership|Borrow|Consume|Inout|Island|Region|Privacy|Budget" -count=1
```

The slice checks both sides of the boundary: safe scoped islands remain
available without `unsafe`, while raw allocation, direct island creation,
manual `free`, capability construction, raw memory access, MMIO, symbol lookup,
and context switching stay rejected unless they appear inside an `unsafe` block
with the required `uses` effects.
