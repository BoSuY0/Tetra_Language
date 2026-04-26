# Capabilities (MVP)

Capabilities are opaque tokens that grant permission to perform specific unsafe
operations. They are meant to make low-level code explicit and auditable.

## Types

- `cap.io` — permission for MMIO-style operations
- `cap.mem` — permission for raw memory access

## Obtaining Capabilities

Capabilities are not constructible in safe code. They can only be obtained in
`unsafe` blocks via builtins. In v0.17 the containing function must also
declare the corresponding effects; `uses` is an audit declaration, not a token
grant.

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

## Status

This is a compile-time gating mechanism with a minimal runtime implementation:
MMIO read/write and raw memory load/store map to normal memory loads/stores in the
current backend.

`uses mem` / `uses io` does not replace `unsafe` and does not create a
capability. Unsafe builtins still require an `unsafe` block and the relevant
`cap.mem` or `cap.io` argument.

## MMIO Semantics (Volatile Contract)

Even though the current backend lowers MMIO operations to normal loads/stores, the
language contract is that MMIO operations are **observable** and must not be
removed, coalesced, or reordered across other MMIO operations by future compiler
optimizations.
