# Memory Ideal Vertical Slice v5 Final Audit

Status: validated_narrow

This audit closes the raw pointer unsafe contract slice for the current
supported surface. It extends the v0/v1/v2/v3/v4 memory correlation pattern
without implementing arbitrary external pointer safety, an FFI lifetime system,
broad unsafe noalias, safe wrapper promotion, actor/task/runtime expansion,
target parity, or performance claims. `MemoryFactGraph` remains the truth
source; reports remain projections.

## Row Classifications

| requirement_id | classification | evidence | boundary |
| --- | --- | --- | --- |
| MEM-UNSAFE-001 | rejected | `compiler/internal/memoryfacts/from_plir_test.go` proves `unsafe_unknown_rejected_safe_facts`; `compiler/internal/memoryfacts/report_test.go` and `tools/cmd/validate-memory-report/main_test.go` reject unsafe_unknown safe/noalias promotion; `compiler/internal/memorymodel/mini_test.go` covers unknown pointer safe/noalias rejection. | Unknown external raw pointers may remain checked or conservative, but never become safe-known, provenance-known, or noalias facts. |
| MEM-UNSAFE-002 | validated_narrow | `compiler/internal/memoryfacts/from_plir_test.go` proves `unsafe_verified_root_allocation_base` with `unsafe_verified_root_bounds_validator`; `compiler/tests/semantics/memory_ideal_v5_raw_pointer_test.go` covers current `core.alloc_bytes` + in-bounds `core.ptr_add` surface; `compiler/internal/memorymodel/mini_test.go` covers verified-root bounds and too-large raw-slice rejection. | Verified `core.alloc_bytes` roots may project bounded allocation-base metadata only as unsafe-origin evidence. |
| MEM-UNSAFE-003 | validated_narrow | `compiler/internal/memoryfacts/from_plir_test.go` proves `unsafe_contract_runtime_checkable`; `compiler/internal/memoryfacts/report_test.go` and `tools/cmd/validate-memory-report/main_test.go` accept only runtime-checkable contract projection with `normal_build_check`; `compiler/internal/memorymodel/mini_test.go` covers nonnull/alignment/length runtime contract validation. | Runtime-checkable unsafe contracts are limited to nonnull, alignment, and length/bounds. |
| MEM-UNSAFE-004 | conservative | `compiler/internal/memoryfacts/from_plir_test.go` proves `unsafe_contract_static_untrusted` with `alias_state: invalidated_by_call`; `compiler/internal/memorymodel/mini_test.go` covers unsafe noalias and lifetime/region static-untrusted cases; report validators continue rejecting validated noalias with unknown alias state and broad noalias claims. | Unsafe noalias, lifetime, and region contracts remain static-untrusted unless separately proven. |

## Minimal Report Projection

Projected v5 claims:

| claim | source stage | validator | notes |
| --- | --- | --- | --- |
| `unsafe_unknown_rejected_safe_facts` | plir | `unsafe_unknown_fact_validator` | Derived from unsafe unknown raw pointer facts and projected as rejected evidence against safe/noalias promotion. |
| `unsafe_verified_root_allocation_base` | allocplan | `unsafe_verified_root_bounds_validator` | Derived from validated `core.alloc_bytes` allocation-base metadata; projected as unsafe verified-root evidence, not safe provenance. |
| `unsafe_contract_runtime_checkable` | plir | `unsafe_runtime_contract_validator` | Validates only nonnull, alignment, and length/bounds contracts and keeps normal-build checks. |
| `unsafe_contract_static_untrusted` | plir | `unsafe_static_contract_validator` | Projects unsafe noalias/lifetime/region contracts as conservative/static-untrusted. |

Validators reject derived v5 rows without `parent_fact_id`, safe claims from
`unsafe_unknown`, generic safe/lifetime claims from `unsafe_verified_root`, and
broad noalias wording. Reports remain projections from compiler-owned facts.

## Positive Coverage

- `core.alloc_bytes` root plus in-bounds `core.ptr_add` type-checks on the
  current unsafe gateway surface.
- MemoryFactGraph projects `unsafe_verified_root_allocation_base` from the
  validated allocation root.
- Runtime-checkable nonnull/alignment/length unsafe contracts project as
  validated dynamic-check evidence.

## Negative Coverage

- Unknown pointers cannot become `safe_known`.
- Unknown pointers cannot emit noalias.
- Unsafe noalias contracts cannot become validated noalias.
- Unsafe lifetime/region contracts cannot become safe lifetime evidence.
- `raw_slice_from_parts` over unknown pointer remains `external_unknown`.
- `raw_slice_from_parts` over a verified root with too-large length is rejected
  or conservative.

## Nonclaims

- No arbitrary external pointer safety.
- No FFI lifetime system.
- No broad unsafe noalias.
- No safe wrapper promotion.
- No actor/task/runtime expansion.
- No target parity.
- No performance claim.
