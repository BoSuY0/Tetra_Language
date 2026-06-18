# P24.0 Unsafe Surface Map

Status: current-branch P24.0 audit artifact for schema
`tetra.security.review_gate.v1` and scope `p24.0_security_review_gate`.

Source of truth: `docs/spec/unsafe.md`.

## Entry Rules

Unsafe-only operations require explicit `unsafe` syntax. Observable effects must
also be declared through `uses`, and raw memory or MMIO operations require the
matching capability token where the policy demands one.

`unsafe` and `uses` are separate gates. `uses` records permitted effects; it
does not grant `cap.mem`, `cap.io`, pointer provenance, allocation lifetime,
alias exclusivity, or actor sendability.

## Unsafe Builtin Map

### Raw Allocation

- Builtins: `core.alloc_bytes`.
- Required effects: `alloc`, `mem`.
- Capability: none.
- Review notes: runtime allocation contracts reject invalid sizes before
  allocator entry and expose report hooks.

### Islands

- Builtins: `core.island_new`, `core.island_make_u8`,
  `core.island_make_u16`, `core.island_make_i32`,
  `core.island_make_bool`.
- Required effects: `alloc`, `islands`, `mem`.
- Capability: none.
- Review notes: `island_make_*` is conditional when the island is not tracked
  as scoped.

### Capability Acquisition

- Builtins: `core.cap_io`, `core.cap_mem`.
- Required effects: `capability`, `io` or `mem`.
- Capability: returns token.
- Review notes: tokens are constructed only inside unsafe blocks.

### Raw Slices

- Builtins: `core.raw_slice_u8_from_parts`, `core.raw_slice_u16_from_parts`.
- Builtins: `core.raw_slice_i32_from_parts`, `core.raw_slice_bool_from_parts`.
- Required effects: `mem`.
- Capability: `cap.mem`.
- Review notes: unknown or external pointers are not promoted to verified
  allocation roots.

### Raw Loads/Stores

- Builtins: `core.load_i32`, `core.store_i32`, `core.load_u8`,
  `core.store_u8`, `core.load_ptr`, `core.store_ptr`.
- Required effects: `mem`.
- Capability: `cap.mem`.
- Review notes: direct visible allocation-base accesses use raw-pointer bounds
  metadata where supported.

### Pointer Arithmetic

- Builtins: `core.ptr_add`.
- Required effects: `mem`.
- Capability: `cap.mem`.
- Review notes: negative offsets, upper-bound offsets, and access-width
  overflow are rejected for verified allocation roots.

### MMIO

- Builtins: `core.mmio_read_i32`, `core.mmio_write_i32`.
- Required effects: `io`, `mmio`.
- Capability: `cap.io`.
- Review notes: MMIO remains observable and must not be removed, coalesced, or
  reordered across MMIO operations.

### Symbol Address

- Builtins: `core.sym_addr`.
- Required effects: `link`.
- Capability: none.
- Review notes: address materialization remains unsafe and link-effect gated.

### Context Switch

- Builtins: `core.ctx_switch`.
- Required effects: `control`, `runtime`.
- Capability: `cap.mem`.
- Review notes: runtime control transfer remains unsafe and explicitly
  effect-gated.

### Atomics

- Builtins: `core.atomic_*` builtins listed in `docs/spec/unsafe.md`.
- Required effects: memory/control policy per builtin family.
- Capability: architecture-specific.
- Review notes: architecture-pointer and atomic operations are unsafe-only and
  must remain in the unsafe inventory.

### Architecture Pointer Store

- Builtins: `core.store_arch_ptr`.
- Required effects: architecture-pointer policy.
- Capability: memory capability where required.
- Review notes: pointer width behavior is target-specific; x32 paths
  zero-extend 32-bit pointer writes.

## Target Boundary

The current WASM policy blocks raw unsafe host/runtime paths including
`core.alloc_bytes`, `core.cap_io`, `core.cap_mem`, raw load/store, `core.ptr_add`,
MMIO, and `core.ctx_switch`. Supported WASM paths remain compile-compatible safe
paths, not raw host memory access.

## Runtime Evidence

### Allocation Contracts

- File: `compiler/internal/runtimeabi/allocation_contract.go`.
- Security relevance: alignment, invalid-size guards, overflow guards, failure
  behavior, debug hooks, and report hooks.

### Raw-Pointer Bounds ABI

- File: `compiler/internal/runtimeabi/raw_pointer_bounds.go`.
- Security relevance: allocation roots, derived offsets, unknown external
  pointers, raw slice metadata, and rejected impossible `ptr_add`.

### Release Safety Slice

- File: `docs/checklists/security_review_gate.md`.
- Security relevance: release checklist names focused unsafe/capability/effect
  tests and required evidence fields.

## Residual Risks

- `cap.mem` authorizes raw operation entry but does not prove pointer validity,
  lifetime, bounds, aliasing, or actor sendability.
- External or unknown raw pointers remain unknown unless constructed from
  verified allocation-root metadata.
- The unsafe inventory must stay aligned with manifest `unsafe_policy` entries
  and generated docs.
- This map does not claim a complete formal memory-safety proof.

## Focused Verification

```sh
CACHE="$PWD/.cache/go-build-ideal-plan"
GOCACHE="$CACHE" \
  go test ./compiler/... \
    -run 'Unsafe|Capability|Effect|MMIO|Mem' \
    -count=1
GOCACHE="$CACHE" \
  go test ./compiler/internal/runtimeabi \
    -run 'Allocation|Region|SmallHeap|RawPointer' \
    -count=1
GOCACHE="$CACHE" \
  go run ./tools/cmd/verify-docs \
    --manifest docs/generated/manifest.json
```
