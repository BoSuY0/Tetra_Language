# Truthful Safe Values

Status: v0 implementation slice.

A truthful safe value is a value whose safe-language representation cannot be
forged by ordinary user code. For memory-backed values, this means the compiler
knows where the memory came from, how long the value can be used, and which
facts are safe to reuse for optimization.

## Slice And String Metadata

`[]T.ptr`, `[]T.len`, `String.ptr`, and `String.len` are representation
metadata. Safe code may read `len`, index through checked operations, and use
safe view operations as they are introduced. Safe code must not assign to
`ptr` or `len`, including through nested structs, generic wrappers, optional
payloads, enum payloads, or `inout` parameters.

Current diagnostic concept:

```text
cannot assign to slice internals ('ptr'/'len'); assign elements via index instead
cannot assign to string internals ('ptr'/'len')
```

## Provenance

Provenance identifies the root of memory:

- allocation
- island
- stack
- literal
- external/FFI
- unknown unsafe
- actor transfer
- parameter

Unsafe code may create values with unknown or external provenance. Safe code
may use those values only through checked operations until trusted facts are
re-established by an audited gateway.

## Raw Slice Gateway

The current audited unsafe gateway is the concrete builtin family:

- `core.raw_slice_u8_from_parts(ptr, len, cap.mem)`
- `core.raw_slice_u16_from_parts(ptr, len, cap.mem)`
- `core.raw_slice_i32_from_parts(ptr, len, cap.mem)`
- `core.raw_slice_bool_from_parts(ptr, len, cap.mem)`

Each builtin is allowed only in an `unsafe` block, requires the `mem` effect
and a `cap.mem` token, and returns a slice header from the supplied raw parts.
PLIR records the result with external/unknown provenance, so the optimizer must
not reuse allocation, noalias, or range facts for it unless a later audited
proof establishes those facts explicitly.

## Safe Slice View Constructors

Safe code now uses method syntax for checked slice views instead of forging
`ptr`/`len` pairs:

- `xs.window(start, count)`
- `xs.prefix(count)`
- `xs.suffix(start)`

The v0 surface covers `[]u8`, `[]u16`, `[]i32`, and `[]bool`. Construction
rejects negative `start` or `count`, `start > xs.len`, and
`count > xs.len - start` before returning a new header. The returned view keeps
pointer provenance derived from the source slice, records a PLIR
`derived_window` range, and receives `len_stable` only when the source
provenance is known. The methods do not expose writable metadata; safe code
still cannot assign `ptr` or `len`.

## Safe View Lifetime Contracts v1

`borrow()` creates a borrowed view. It does not allocate, and the compiler
tracks the result as `borrowed_imm` with `no_escape` and derived provenance.
`copy()` creates owned storage with new known provenance. `copy_into(dst)`
writes bytes or elements into caller-owned destination storage and does not
create a fresh allocation intent.

Functions may now declare borrowed view returns for the supported surface:

```tetra
func view_bytes(xs: borrow []u8) -> borrow []u8:
    return xs.window(1, 2).borrow()

func view_text(text: borrow String) -> borrow String:
    return text.window(1, 3).borrow()
```

The v1 relation is deliberately conservative. A borrowed return must be tied
to a single safe source such as a parameter, a compatible borrowed return from
another function, or an already modeled static-safe source. Borrowed returns
from local allocation, copied local storage, unsafe unknown provenance, or
different branch owner sources are rejected.

Borrowed views may be read locally, passed to `borrow` parameters, or copied.
They may not escape through an owned return, mutable global storage, actor or
task boundaries, closure escape, consume parameters, or hidden aggregate
payloads. Structs, enum payloads, optionals, and generic wrappers containing a
borrowed slice or String are treated as borrowed for escape checks; copying the
field or payload removes that borrow restriction by creating owned storage.

This is not a named-lifetime system, generic lifetime parameter model, full
Rust-like borrow checker, arbitrary borrowed aggregate return feature, Unicode
text lifetime model, or production FFI lifetime contract.

## Allocation Intent

`make<T>(n)` is an allocation intent, not a command to use one storage class.
The current backend remains conservative, but reports expose the storage
decision so future planner changes are reviewable.

Required contract:

- `n == 0` creates a valid empty slice without allocator access on supported
  native ABI paths.
- `n < 0` traps or rejects before allocator or island metadata access.
- `n * sizeof(T)` byte-size overflow traps or rejects before allocator or
  island metadata access.
- the returned slice length is always the logical element count, not the byte
  count.
- storage decisions must not alter safe semantics.

The contract applies to `core.make_u8`, `core.make_u16`, `core.make_i32`,
`core.make_bool`, and the matching `core.island_make_*` constructors. The
current bool slice representation uses i32-width slots, so PLIR records
`element_size = 4` for `bool` allocation intents.

Allocation reports distinguish `valid_empty_allocation`, `normal_allocation`,
`rejected_negative_length`, and `rejected_byte_size_overflow`. These reports
also carry stable `site_id`, `planned_storage`, `actual_lowering_storage`,
validation status, and lowering status. They are explanatory only: enabling
`--explain` or `--emit-alloc-report` must not change the safe allocation
contract.
On x64 native targets, P2.1 may lower a fixed small no-escape local
constructor to stack-frame backing, but the same zero/negative/overflow and
logical-length rules remain mandatory.
