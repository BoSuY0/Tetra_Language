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

| Surface | Builtins | Required effects | Capability | Review notes |
| --- | --- | --- | --- | --- |
| Raw allocation | `core.alloc_bytes` | `alloc`, `mem` | none | Runtime allocation contracts reject invalid sizes before allocator entry and expose report hooks. |
| Islands | `core.island_new`, `core.island_make_u8`, `core.island_make_u16`, `core.island_make_i32`, `core.island_make_bool` | `alloc`, `islands`, `mem` | none | `island_make_*` is conditional when the island is not tracked as scoped. |
| Capability acquisition | `core.cap_io`, `core.cap_mem` | `capability`, `io` or `mem` | returns token | Tokens are constructed only inside unsafe blocks. |
| Raw slices | `core.raw_slice_u8_from_parts`, `core.raw_slice_u16_from_parts`, `core.raw_slice_i32_from_parts`, `core.raw_slice_bool_from_parts` | `mem` | `cap.mem` | Unknown or external pointers are not promoted to verified allocation roots. |
| Raw loads/stores | `core.load_i32`, `core.store_i32`, `core.load_u8`, `core.store_u8`, `core.load_ptr`, `core.store_ptr` | `mem` | `cap.mem` | Direct visible allocation-base accesses use raw-pointer bounds metadata where supported. |
| Pointer arithmetic | `core.ptr_add` | `mem` | `cap.mem` | Negative offsets, upper-bound offsets, and access-width overflow are rejected for verified allocation roots. |
| MMIO | `core.mmio_read_i32`, `core.mmio_write_i32` | `io`, `mmio` | `cap.io` | MMIO remains observable and must not be removed, coalesced, or reordered across MMIO operations. |
| Symbol address | `core.sym_addr` | `link` | none | Address materialization remains unsafe and link-effect gated. |
| Context switch | `core.ctx_switch` | `control`, `runtime` | `cap.mem` | Runtime control transfer remains unsafe and explicitly effect-gated. |
| Atomics | `core.atomic_*` builtins listed in `docs/spec/unsafe.md` | memory/control policy per builtin family | architecture-specific | Architecture-pointer and atomic operations are unsafe-only and must remain in the unsafe inventory. |
| Architecture pointer store | `core.store_arch_ptr` | architecture-pointer policy | memory capability where required | Pointer width behavior is target-specific; x32 paths zero-extend 32-bit pointer writes. |

## Target Boundary

The current WASM policy blocks raw unsafe host/runtime paths including
`core.alloc_bytes`, `core.cap_io`, `core.cap_mem`, raw load/store, `core.ptr_add`,
MMIO, and `core.ctx_switch`. Supported WASM paths remain compile-compatible safe
paths, not raw host memory access.

## Runtime Evidence

| Runtime evidence | File | Security relevance |
| --- | --- | --- |
| Allocation contracts | `compiler/internal/runtimeabi/allocation_contract.go` | Alignment, invalid-size guards, overflow guards, failure behavior, debug hooks, and report hooks. |
| Raw-pointer bounds ABI | `compiler/internal/runtimeabi/raw_pointer_bounds.go` | Allocation roots, derived offsets, unknown external pointers, raw slice metadata, and rejected impossible `ptr_add`. |
| Release safety slice | `docs/checklists/security_review_gate.md` | Release checklist names focused unsafe/capability/effect tests and required evidence fields. |

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
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/runtimeabi -run 'Allocation|Region|SmallHeap|RawPointer' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```
