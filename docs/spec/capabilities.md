# Capabilities (MVP)

Capabilities are opaque tokens that grant permission to perform specific unsafe
operations. They are meant to make low-level code explicit and auditable.

## Types

- `cap.io` — permission for MMIO-style operations
- `cap.mem` — permission for raw memory access

## Obtaining Capabilities

Capabilities are not constructible in safe code. They can only be obtained in
`unsafe` blocks via builtins. The containing function must also declare the
corresponding effects; `uses` is an audit declaration, not a token grant.

```tetra
fun main(): i32 uses capability, io, mem {
  unsafe {
    let io: cap.io = core.cap_io()
    let mem: cap.mem = core.cap_mem()
  }
  return 0
}
```

## Using Capabilities

Some builtins require a capability parameter in addition to `unsafe`:

```tetra
fun main(): i32 uses alloc, capability, io, mem, mmio {
  unsafe {
    let io: cap.io = core.cap_io()
    let mem: cap.mem = core.cap_mem()
    let p: ptr = core.alloc_bytes(4)
    let v: i32 = core.mmio_read_i32(p, io)
    let w: i32 = core.mmio_write_i32(p, v, io)
    let x: i32 = core.load_i32(p, mem)
    let y: i32 = core.store_i32(p, x, mem)
    let b: u8 = core.load_u8(p, mem)
    let c: u8 = core.store_u8(p, b, mem)
    let slot: ptr = core.alloc_bytes(8)
    let _sp: ptr = core.store_ptr(slot, p, mem)
    let p2: ptr = core.load_ptr(slot, mem)
  }
  return 0
}
```

## Safe Wrapper Pattern

Stable wrappers keep unsafe regions tiny and make the public effect contract
visible at the wrapper boundary. A wrapper may contain the `unsafe` block, but
callers still need the wrapper's declared effects:

```tetra
module lib.core.memory

func write_i32(dst: ptr, value: Int, mem: cap.mem) -> Int
uses mem:
  unsafe:
    return core.store_i32(dst, value, mem)
```

Callers cannot manufacture `cap.mem` through `uses mem`; they must receive a
capability from a reviewed unsafe boundary such as `lib.core.capability.mem()`
or another explicitly audited wrapper.

When callers opt into attenuated capability groups, raw memory access must also
carry `capsule.mem`; attenuated IO follows the same pattern with `capsule.io`.

## Memory Production Boundary

`cap.mem` is permission, not provenance. It proves that raw memory access crossed
an explicit unsafe/capability boundary, but it does not by itself prove pointer
validity, allocation lifetime, region size, alias exclusivity, or thread/actor
sendability.

The Memory Production Core requires deterministic runtime bounds diagnostics for
raw memory paths before they can be promoted as production memory evidence. Until
that evidence exists, wrappers that take `cap.mem` must still document the
caller-owned pointer validity and size obligations.

## Status

This is a compile-time gating mechanism with a minimal runtime implementation:
MMIO read/write and raw memory load/store map to normal memory loads/stores in the
current backend.

`uses mem` / `uses io` does not replace `unsafe` and does not create a
capability. Unsafe builtins still require an `unsafe` block and the relevant
`cap.mem` or `cap.io` argument.

Privacy capabilities are separate: `consent.token` is obtained through the
privacy surface and is documented in
[effects_capabilities_privacy_v1.md](./effects_capabilities_privacy_v1.md).

## MMIO Semantics (Volatile Contract)

Even though the current backend lowers MMIO operations to normal loads/stores, the
language contract is that MMIO operations are **observable** and must not be
removed, coalesced, or reordered across other MMIO operations by future compiler
optimizations.

## Epic 06 coverage

Capability coverage is release-blocking in the focused safety test slice:

```sh
go test ./compiler/... -run "Effect|Uses|Capability|Unsafe|Ownership|Borrow|Consume|Inout|Island|Region|Privacy|Budget" -count=1
```

The slice checks that safe code cannot manufacture capability tokens, that
unsafe raw-memory/MMIO calls require the right `cap.mem` or `cap.io` argument,
and that attenuated capability groups require the matching capsule permission.
