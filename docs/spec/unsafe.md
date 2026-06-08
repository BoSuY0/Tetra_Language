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
| `core.island_reset` | always | `islands`, `mem` | island handle |
| `core.island_make_u8` | conditional when the island is not a tracked scoped island | `alloc`, `islands`, `mem` | none |
| `core.island_make_u16` | conditional when the island is not a tracked scoped island | `alloc`, `islands`, `mem` | none |
| `core.island_make_i32` | conditional when the island is not a tracked scoped island | `alloc`, `islands`, `mem` | none |
| `core.island_make_bool` | conditional when the island is not a tracked scoped island | `alloc`, `islands`, `mem` | none |
| explicit `free(<island>)` | always | `islands`, `mem` | island handle |
| `core.cap_io` | always | `capability`, `io` | returns `cap.io` |
| `core.cap_mem` | always | `capability`, `mem` | returns `cap.mem` |
| `core.raw_slice_u8_from_parts` | always | `mem` | `cap.mem` |
| `core.raw_slice_u16_from_parts` | always | `mem` | `cap.mem` |
| `core.raw_slice_i32_from_parts` | always | `mem` | `cap.mem` |
| `core.raw_slice_bool_from_parts` | always | `mem` | `cap.mem` |
| `core.load_i32` / `core.store_i32` | always | `mem` | `cap.mem` |
| `core.load_u8` / `core.store_u8` | always | `mem` | `cap.mem` |
| `core.load_ptr` / `core.store_ptr` | always | `mem` | `cap.mem` |
| `core.ptr_add` | always | `mem` | `cap.mem` |
| `core.mmio_read_i32` / `core.mmio_write_i32` | always | `io`, `mmio` | `cap.io` |
| `core.sym_addr` | always | `link` | none |
| `core.ctx_switch` | always | `control`, `runtime` | `cap.mem` |

Safe slice views are not raw gateways. `xs.window(start, count)`,
`xs.prefix(count)`, and `xs.suffix(start)` require no capability token and
trap/reject invalid ranges before constructing a view; they derive provenance
from the source slice instead of accepting caller-supplied raw parts.

## Memory Report Boundary

Memory Production Core v1 exposes `tetra.memory-report.v1` through
`--emit-memory-report`. The report is a projection of compiler-owned memory
facts, not a source of truth and not an unsafe permission grant.

Unsafe rows must keep their origin visible. Verified `core.alloc_bytes` roots
may report `unsafe_verified_root` allocation-base metadata, while arbitrary raw
external pointers and raw slices from unknown parts remain `unsafe_unknown` or
`checked_external_unknown`. A report row must not convert unknown unsafe
provenance into `safe_known`; validators reject that shape. MPC-7 also rejects
`unsafe_unknown` rows that try to claim `provenance_known`, noalias,
`index_in_range`, bounds-check-elimination, or trusted stack/region/island
lowering. Raw allocation, pointer arithmetic, raw load/store, and raw-slice
gateways carry explicit PLIR unsafe classes before report projection. Those
rows may remain useful diagnostics, but they are never optimization permission.
For verified `core.alloc_bytes` roots, pointer arithmetic and raw memory access
may report `derived_allocation_offset`, `rejected_negative_offset`,
`rejected_upper_bound`, or `rejected_access_width_overflow` as `unsafe_checked`
evidence. Unknown raw pointers still report `checked_external_unknown` or
`external_unknown`; they do not become safe facts or trusted bounds proofs.
Memory Ideal v7 extends this boundary to FFI calls: external pointers project
`ffi_pointer_external_unknown`, external calls may retain borrowed pointers,
safe wrapper promotion without compiler-owned proof is rejected, and external
calls invalidate broad noalias. These rows do not prove C-side lifetimes,
arbitrary external allocator provenance, or safe wrapper promotion.
Raw slice construction from a verified allocation root may report bounded
`raw_slice_verified_allocation_root` evidence only when the constant
`len * sizeof(T)` and `offset + lenBytes` fit inside allocation metadata.
Negative raw-slice lengths trap before view construction on linux-x64 and report
`rejected_negative_length` when statically classified. Metadata arithmetic
overflow reports `rejected_length_overflow`; target byte-overflow traps are
linux-x64 runtime evidence and may remain conservative in reports until
target-aware raw-slice metadata is added. None of these rows produce
`safe_known`, noalias, `index_in_range`, or bounds-check-elimination permission.

Atomic and architecture-pointer raw memory builtins are also unsafe-only. The
manifest expands them by operation, value width, and memory order; this literal
registry keeps the generated surface auditable:

```text
core.atomic_compare_exchange_i32_acq_rel
core.atomic_compare_exchange_i32_acquire
core.atomic_compare_exchange_i32_relaxed
core.atomic_compare_exchange_i32_release
core.atomic_compare_exchange_i32_seq_cst
core.atomic_compare_exchange_i64_acq_rel
core.atomic_compare_exchange_i64_acquire
core.atomic_compare_exchange_i64_relaxed
core.atomic_compare_exchange_i64_release
core.atomic_compare_exchange_i64_seq_cst
core.atomic_compare_exchange_ptr_acq_rel
core.atomic_compare_exchange_ptr_acquire
core.atomic_compare_exchange_ptr_relaxed
core.atomic_compare_exchange_ptr_release
core.atomic_compare_exchange_ptr_seq_cst
core.atomic_compare_exchange_u16_acq_rel
core.atomic_compare_exchange_u16_acquire
core.atomic_compare_exchange_u16_relaxed
core.atomic_compare_exchange_u16_release
core.atomic_compare_exchange_u16_seq_cst
core.atomic_compare_exchange_u8_acq_rel
core.atomic_compare_exchange_u8_acquire
core.atomic_compare_exchange_u8_relaxed
core.atomic_compare_exchange_u8_release
core.atomic_compare_exchange_u8_seq_cst
core.atomic_compare_exchange_weak_i32_acq_rel
core.atomic_compare_exchange_weak_i32_acquire
core.atomic_compare_exchange_weak_i32_relaxed
core.atomic_compare_exchange_weak_i32_release
core.atomic_compare_exchange_weak_i32_seq_cst
core.atomic_compare_exchange_weak_i64_acq_rel
core.atomic_compare_exchange_weak_i64_acquire
core.atomic_compare_exchange_weak_i64_relaxed
core.atomic_compare_exchange_weak_i64_release
core.atomic_compare_exchange_weak_i64_seq_cst
core.atomic_compare_exchange_weak_ptr_acq_rel
core.atomic_compare_exchange_weak_ptr_acquire
core.atomic_compare_exchange_weak_ptr_relaxed
core.atomic_compare_exchange_weak_ptr_release
core.atomic_compare_exchange_weak_ptr_seq_cst
core.atomic_compare_exchange_weak_u16_acq_rel
core.atomic_compare_exchange_weak_u16_acquire
core.atomic_compare_exchange_weak_u16_relaxed
core.atomic_compare_exchange_weak_u16_release
core.atomic_compare_exchange_weak_u16_seq_cst
core.atomic_compare_exchange_weak_u8_acq_rel
core.atomic_compare_exchange_weak_u8_acquire
core.atomic_compare_exchange_weak_u8_relaxed
core.atomic_compare_exchange_weak_u8_release
core.atomic_compare_exchange_weak_u8_seq_cst
core.atomic_exchange_i32_acq_rel
core.atomic_exchange_i32_acquire
core.atomic_exchange_i32_relaxed
core.atomic_exchange_i32_release
core.atomic_exchange_i32_seq_cst
core.atomic_exchange_i64_acq_rel
core.atomic_exchange_i64_acquire
core.atomic_exchange_i64_relaxed
core.atomic_exchange_i64_release
core.atomic_exchange_i64_seq_cst
core.atomic_exchange_ptr_acq_rel
core.atomic_exchange_ptr_acquire
core.atomic_exchange_ptr_relaxed
core.atomic_exchange_ptr_release
core.atomic_exchange_ptr_seq_cst
core.atomic_exchange_u16_acq_rel
core.atomic_exchange_u16_acquire
core.atomic_exchange_u16_relaxed
core.atomic_exchange_u16_release
core.atomic_exchange_u16_seq_cst
core.atomic_exchange_u8_acq_rel
core.atomic_exchange_u8_acquire
core.atomic_exchange_u8_relaxed
core.atomic_exchange_u8_release
core.atomic_exchange_u8_seq_cst
core.atomic_fence_acq_rel
core.atomic_fence_acquire
core.atomic_fence_relaxed
core.atomic_fence_release
core.atomic_fence_seq_cst
core.atomic_fetch_add_i32_acq_rel
core.atomic_fetch_add_i32_acquire
core.atomic_fetch_add_i32_relaxed
core.atomic_fetch_add_i32_release
core.atomic_fetch_add_i32_seq_cst
core.atomic_fetch_add_i64_acq_rel
core.atomic_fetch_add_i64_acquire
core.atomic_fetch_add_i64_relaxed
core.atomic_fetch_add_i64_release
core.atomic_fetch_add_i64_seq_cst
core.atomic_fetch_add_ptr_acq_rel
core.atomic_fetch_add_ptr_acquire
core.atomic_fetch_add_ptr_relaxed
core.atomic_fetch_add_ptr_release
core.atomic_fetch_add_ptr_seq_cst
core.atomic_fetch_add_u16_acq_rel
core.atomic_fetch_add_u16_acquire
core.atomic_fetch_add_u16_relaxed
core.atomic_fetch_add_u16_release
core.atomic_fetch_add_u16_seq_cst
core.atomic_fetch_add_u8_acq_rel
core.atomic_fetch_add_u8_acquire
core.atomic_fetch_add_u8_relaxed
core.atomic_fetch_add_u8_release
core.atomic_fetch_add_u8_seq_cst
core.atomic_fetch_and_i32_acq_rel
core.atomic_fetch_and_i32_acquire
core.atomic_fetch_and_i32_relaxed
core.atomic_fetch_and_i32_release
core.atomic_fetch_and_i32_seq_cst
core.atomic_fetch_and_i64_acq_rel
core.atomic_fetch_and_i64_acquire
core.atomic_fetch_and_i64_relaxed
core.atomic_fetch_and_i64_release
core.atomic_fetch_and_i64_seq_cst
core.atomic_fetch_and_ptr_acq_rel
core.atomic_fetch_and_ptr_acquire
core.atomic_fetch_and_ptr_relaxed
core.atomic_fetch_and_ptr_release
core.atomic_fetch_and_ptr_seq_cst
core.atomic_fetch_and_u16_acq_rel
core.atomic_fetch_and_u16_acquire
core.atomic_fetch_and_u16_relaxed
core.atomic_fetch_and_u16_release
core.atomic_fetch_and_u16_seq_cst
core.atomic_fetch_and_u8_acq_rel
core.atomic_fetch_and_u8_acquire
core.atomic_fetch_and_u8_relaxed
core.atomic_fetch_and_u8_release
core.atomic_fetch_and_u8_seq_cst
core.atomic_fetch_or_i32_acq_rel
core.atomic_fetch_or_i32_acquire
core.atomic_fetch_or_i32_relaxed
core.atomic_fetch_or_i32_release
core.atomic_fetch_or_i32_seq_cst
core.atomic_fetch_or_i64_acq_rel
core.atomic_fetch_or_i64_acquire
core.atomic_fetch_or_i64_relaxed
core.atomic_fetch_or_i64_release
core.atomic_fetch_or_i64_seq_cst
core.atomic_fetch_or_ptr_acq_rel
core.atomic_fetch_or_ptr_acquire
core.atomic_fetch_or_ptr_relaxed
core.atomic_fetch_or_ptr_release
core.atomic_fetch_or_ptr_seq_cst
core.atomic_fetch_or_u16_acq_rel
core.atomic_fetch_or_u16_acquire
core.atomic_fetch_or_u16_relaxed
core.atomic_fetch_or_u16_release
core.atomic_fetch_or_u16_seq_cst
core.atomic_fetch_or_u8_acq_rel
core.atomic_fetch_or_u8_acquire
core.atomic_fetch_or_u8_relaxed
core.atomic_fetch_or_u8_release
core.atomic_fetch_or_u8_seq_cst
core.atomic_fetch_sub_i32_acq_rel
core.atomic_fetch_sub_i32_acquire
core.atomic_fetch_sub_i32_relaxed
core.atomic_fetch_sub_i32_release
core.atomic_fetch_sub_i32_seq_cst
core.atomic_fetch_sub_i64_acq_rel
core.atomic_fetch_sub_i64_acquire
core.atomic_fetch_sub_i64_relaxed
core.atomic_fetch_sub_i64_release
core.atomic_fetch_sub_i64_seq_cst
core.atomic_fetch_sub_ptr_acq_rel
core.atomic_fetch_sub_ptr_acquire
core.atomic_fetch_sub_ptr_relaxed
core.atomic_fetch_sub_ptr_release
core.atomic_fetch_sub_ptr_seq_cst
core.atomic_fetch_sub_u16_acq_rel
core.atomic_fetch_sub_u16_acquire
core.atomic_fetch_sub_u16_relaxed
core.atomic_fetch_sub_u16_release
core.atomic_fetch_sub_u16_seq_cst
core.atomic_fetch_sub_u8_acq_rel
core.atomic_fetch_sub_u8_acquire
core.atomic_fetch_sub_u8_relaxed
core.atomic_fetch_sub_u8_release
core.atomic_fetch_sub_u8_seq_cst
core.atomic_fetch_xor_i32_acq_rel
core.atomic_fetch_xor_i32_acquire
core.atomic_fetch_xor_i32_relaxed
core.atomic_fetch_xor_i32_release
core.atomic_fetch_xor_i32_seq_cst
core.atomic_fetch_xor_i64_acq_rel
core.atomic_fetch_xor_i64_acquire
core.atomic_fetch_xor_i64_relaxed
core.atomic_fetch_xor_i64_release
core.atomic_fetch_xor_i64_seq_cst
core.atomic_fetch_xor_ptr_acq_rel
core.atomic_fetch_xor_ptr_acquire
core.atomic_fetch_xor_ptr_relaxed
core.atomic_fetch_xor_ptr_release
core.atomic_fetch_xor_ptr_seq_cst
core.atomic_fetch_xor_u16_acq_rel
core.atomic_fetch_xor_u16_acquire
core.atomic_fetch_xor_u16_relaxed
core.atomic_fetch_xor_u16_release
core.atomic_fetch_xor_u16_seq_cst
core.atomic_fetch_xor_u8_acq_rel
core.atomic_fetch_xor_u8_acquire
core.atomic_fetch_xor_u8_relaxed
core.atomic_fetch_xor_u8_release
core.atomic_fetch_xor_u8_seq_cst
core.atomic_load_i32_acquire
core.atomic_load_i32_relaxed
core.atomic_load_i32_seq_cst
core.atomic_load_i64_acquire
core.atomic_load_i64_relaxed
core.atomic_load_i64_seq_cst
core.atomic_load_ptr_acquire
core.atomic_load_ptr_relaxed
core.atomic_load_ptr_seq_cst
core.atomic_load_u16_acquire
core.atomic_load_u16_relaxed
core.atomic_load_u16_seq_cst
core.atomic_load_u8_acquire
core.atomic_load_u8_relaxed
core.atomic_load_u8_seq_cst
core.atomic_store_i32_relaxed
core.atomic_store_i32_release
core.atomic_store_i32_seq_cst
core.atomic_store_i64_relaxed
core.atomic_store_i64_release
core.atomic_store_i64_seq_cst
core.atomic_store_ptr_relaxed
core.atomic_store_ptr_release
core.atomic_store_ptr_seq_cst
core.atomic_store_u16_relaxed
core.atomic_store_u16_release
core.atomic_store_u16_seq_cst
core.atomic_store_u8_relaxed
core.atomic_store_u8_release
core.atomic_store_u8_seq_cst
core.store_arch_ptr
```

Atomic store builtins return the stored value truncated to their width. Atomic
exchange, compare-exchange, and fetch builtins return the observed old value.
All narrow `u8`/`u16` atomic results are zero-extended into the Tetra scalar
slot.

Pointer-width raw stores and pointer atomics use the target pointer width, not
the architectural register width. On `linux-x32`, `core.store_ptr` and
`core.atomic_store_ptr_*` write 32-bit pointer values and zero-extend returned
pointer results into the 64-bit machine slot; pointer compare-exchange results
follow the same zero-extension rule on both success and failure paths.

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
