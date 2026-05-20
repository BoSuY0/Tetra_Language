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
vary (for example, WASM targets use a compile-compatible fallback
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

## WASM Target Policy

`wasm32-wasi` and `wasm32-web` accept the safe scalar/control-flow, print,
slice, string, global, call, `core.sym_addr`, and compile-compatible scoped
island IR paths. The current WASM policy blocks raw unsafe host/runtime paths
before backend emission:

| Builtin/IR family | WASM policy |
| --- | --- |
| `core.alloc_bytes` | blocked |
| `core.cap_io` / `core.cap_mem` | blocked |
| `core.load_i32` / `core.store_i32` / `core.load_u8` / `core.store_u8` | blocked |
| `core.load_ptr` / `core.store_ptr` / `core.ptr_add` | blocked |
| `core.mmio_read_i32` / `core.mmio_write_i32` | blocked |
| `core.ctx_switch` | blocked |

Unsupported WASM unsafe/capability paths are compile-time target diagnostics
with `TETRA3003`; they are not lowered to generic backend "unsupported IR"
failures.

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

## Memory Production Contract Boundary

`cap.mem` authorizes the raw operation, but it does not prove pointer validity or bounds.
A function can hold a valid `cap.mem` token and still pass a stale,
null, undersized, aliased, or otherwise invalid pointer to a raw memory builtin.

`lib.core.memory.memcpy_u8` and `lib.core.memory.memset_u8` keep unsafe byte
loops behind a stable helper surface, but `memcpy_u8` and `memset_u8` inherit
the same boundary: the caller must choose valid readable/writable regions and
sizes until the Memory Production Core runtime supplies deterministic runtime
bounds diagnostics and allocator failure semantics for the exercised path.
The invalid allocation sizes are part of that runtime boundary: the current
Linux-x64 allocator rejects `core.alloc_bytes` sizes less than one before
calling into the host allocator.

The current Linux-x64 runtime slice rejects negative `core.ptr_add` offsets
before returning a derived pointer. This catches lower-bound pointer arithmetic
violations, but it does not prove the resulting non-negative offset is within
the allocation.

For pointers returned directly by `core.alloc_bytes`, allocation-base `core.ptr_add` upper bounds
also reject offsets greater than or equal to the requested allocation size. This
protects helper loops and direct `core.load_*`/`core.store_*` calls whose
address is a visible `core.ptr_add(base, offset, mem)`, while arbitrary
derived-pointer provenance remains outside the current slice.

For pointers returned directly by `core.alloc_bytes`, allocation-base `core.store_i32` width bounds
reject a 4-byte raw store when the requested allocation size is smaller than 4
bytes. The Linux-x64 backend shares that allocation-base check for
`core.load_i32`; direct visible base+offset raw accesses get offset+width
checks, but arbitrary stored derived-pointer word-access bounds still need
provenance.

For pointers returned directly by `core.alloc_bytes`, allocation-base `core.store_ptr` width bounds
reject an 8-byte raw pointer store when the requested allocation size is
smaller than 8 bytes. The Linux-x64 backend shares that allocation-base check
for `core.load_ptr`; direct visible base+offset pointer loads/stores get
offset+width checks, but arbitrary stored derived-pointer pointer slot bounds
still need provenance.

The stable memory helpers reject negative `memcpy_u8` and `memset_u8` lengths
by returning status `2` before entering their unsafe byte loops. That status is
an explicit helper precondition result; it does not prove pointer provenance or
replace runtime bounds diagnostics for positive lengths.

For the Memory Production Core, compile-time checks cover unsafe/effect/capability
gates plus statically visible ownership and borrow escape errors. The runtime bounds diagnostics
are separate evidence requirements and must be proven by
`tetra.memory.production.v1` reports rather than inferred from the existence of
an `unsafe` block or a `cap.mem` token.

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
