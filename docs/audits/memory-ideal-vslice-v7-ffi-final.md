# Memory Ideal Vertical Slice v7 FFI Final Audit

Status: validated_narrow

This audit closes the external pointer and FFI lifetime quarantine slice for
the current supported surface. It extends the v0/v1/v2/v3/v4/v5/v6 memory
correlation pattern without implementing arbitrary external pointer safety,
C/FFI lifetime safety, safe wrapper promotion, broad unsafe noalias, target
parity, performance, arbitrary external allocator provenance, or full
runtime/ABI proof. `MemoryFactGraph` remains the truth source; reports remain
projections.

## Row Classifications

| requirement_id | classification | evidence | boundary |
| --- | --- | --- | --- |
| MEM-FFI-001 | conservative | `compiler/internal/memoryfacts/from_plir_test.go` proves `ffi_pointer_external_unknown`; `tools/cmd/validate-memory-report/main_test.go` keeps unsafe/external provenance from becoming `provenance_known`; existing report validators reject unsafe/external safe/noalias and bounds-elimination claims. | External pointers remain `unsafe_unknown` or `external_unknown` unless compiler-owned provenance exists. |
| MEM-FFI-002 | conservative | `compiler/internal/memoryfacts/from_plir_test.go` proves `ffi_call_may_retain_borrow`; `compiler/internal/memorymodel/mini_test.go` covers FFI calls that may retain borrowed pointers. | External calls may retain borrowed pointers unless a compiler-owned contract proves otherwise. |
| MEM-FFI-003 | rejected | `compiler/internal/memoryfacts/from_plir_test.go` proves `safe_wrapper_promotion_rejected_without_contract`; `compiler/internal/memorymodel/mini_test.go` rejects safe-wrapper promotion from external pointers; report validators reject unsafe_unknown safe promotion. | Safe wrapper promotion from raw or external pointers is rejected without compiler-owned proof. |
| MEM-FFI-004 | conservative | `compiler/internal/memoryfacts/from_plir_test.go` proves `ffi_noalias_invalidated_by_external_call`; `compiler/internal/memorymodel/mini_test.go` covers external-call noalias invalidation; report validators reject broad noalias claims. | External calls invalidate broad noalias unless a later narrow validator proves a specific fact. |

## Minimal Report Projection

Projected v7 claims:

| claim | source stage | validator | notes |
| --- | --- | --- | --- |
| `ffi_pointer_external_unknown` | PLIR | `external_pointer_provenance_validator` | Records that an external pointer at an FFI boundary remains unsafe/external unknown and conservative. |
| `ffi_call_may_retain_borrow` | PLIR | `ffi_lifetime_conservative_validator` | Derived from borrowed parent evidence passed to an external call; requires `parent_fact_id` and remains conservative. |
| `ffi_noalias_invalidated_by_external_call` | PLIR | `ffi_noalias_conservative_validator` | Derived from noalias parent evidence at an external call; requires `parent_fact_id`, carries `alias_state: invalidated_by_call`, and remains conservative. |
| `safe_wrapper_promotion_rejected_without_contract` | PLIR | `safe_wrapper_promotion_validator` | Derived from raw/external unsafe parent evidence; requires `parent_fact_id` and remains rejected. |
| `external_pointer_provenance_rejected` | PLIR | `external_pointer_provenance_validator` | Supporting rejected evidence that external pointer provenance cannot be promoted without compiler-owned proof; requires `parent_fact_id`. |

Validators reject derived v7 rows without `parent_fact_id`, unsafe_unknown
safe/provenance/noalias promotion, unsafe/external bounds-check elimination,
dynamic-check rows without `normal_build_check`, and broad noalias wording.
Reports remain projections from compiler-owned facts.

## Positive Coverage

- External pointers project as `ffi_pointer_external_unknown` /
  `external_unknown` conservative evidence.
- External calls emit conservative may-retain rows for borrowed pointer
  arguments.
- Explicit owned-copy crossing remains owned only where the existing safe copy
  path already supports it.
- The v7 correlation row set validates exactly `MEM-FFI-001` through
  `MEM-FFI-004`.

## Negative Coverage

- Raw/external pointers cannot become `safe_known`.
- External pointers cannot become `provenance_known`.
- Safe wrappers from external pointers are rejected without compiler-owned
  contracts.
- Borrowed locals passed to FFI are not treated as non-escaping.
- FFI calls cannot produce broad or validated noalias.
- Unsafe/external pointers cannot authorize `bounds_check_eliminated` or
  `index_in_range`.

## Nonclaims

- No "Memory 100% complete".
- No arbitrary external pointer safety.
- No C/FFI lifetime safety.
- No safe wrapper promotion.
- No broad unsafe noalias.
- No target parity.
- No performance claim.
- No arbitrary external allocator provenance.
- No full runtime/ABI proof.
