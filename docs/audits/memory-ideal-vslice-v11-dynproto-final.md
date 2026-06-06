# Memory Ideal Vertical Slice v11 Dynamic Protocol Final Audit

Date: 2026-06-06

Decision: accepted

Status: validated_narrow

## Summary

`MEM-DYNPROTO-011` adds a narrow dynamic protocol / witness-table memory
conservatism slice to the existing v0-v10 Memory Ideal evidence chain.
`MemoryFactGraph` remains the truth source, `tetra.memory-report.v1` remains a
projection, and this slice does not add a full trait-object/existential runtime
proof, complete witness-table ABI proof, production dynamic dispatch runtime
safety, target parity, performance, broad noalias, unsafe/external pointer
promotion, clean-release, or "Memory 100%" claim.

## Row Classification

| Requirement | Status | Decision-grade interpretation |
| --- | --- | --- |
| `MEM-DYNPROTO-001` | `conservative` | Dynamic existential/protocol borrow carriers remain conservative unless statically resolved. |
| `MEM-DYNPROTO-002` | `validated_narrow` | Static witness/conformance proof may carry borrow facts only with a compiler-owned parent fact. |
| `MEM-DYNPROTO-003` | `rejected` | Dynamic protocol dispatch cannot validate broad noalias. |
| `MEM-DYNPROTO-004` | `rejected` | Witness/conformance table lookup cannot promote unsafe/dynamic/unknown provenance to `safe_known`. |
| `MEM-DYNPROTO-005` | `validated_narrow` | Protocol/existential dispatch report rows preserve `source_fact_id`, `cost_class`, and `normal_build_check`. |

## Validator Map

| Validator | Evidence |
| --- | --- |
| `dynamic_existential_borrow_conservative_validator` | `MiniMemoryModel` and `MemoryFactGraph` keep dynamic existential/protocol carriers conservative. |
| `static_witness_parent_fact_validator` | Static witness/conformance rows require compiler-owned parent facts. |
| `dynamic_protocol_noalias_rejection_validator` | Dynamic protocol dispatch noalias claims are rejected. |
| `witness_provenance_promotion_validator` | Witness lookup cannot promote unsafe/unknown provenance to safe-known. |
| `protocol_dispatch_report_integrity_validator` | Report rows preserve `source_fact_id`, `cost_class`, and `normal_build_check`. |

## RED Evidence

Observed RED failures:

- `validate-memory-correlation` treated `MEM-DYNPROTO-*` rows as unexpected v0
  rows.
- `MiniMemoryModel` lacked v11 dynamic protocol / witness vocabulary.
- `MemoryFactGraph` did not project v11 dynamic protocol report rows.
- Standalone `ValidateReport` accepted `protocol_dispatch_report_integrity`
  without the required `cost_class` / `normal_build_check` fields.

## Current GREEN Evidence

Focused evidence commands already observed during implementation:

```bash
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-tools go test ./tools/cmd/validate-memory-correlation -run 'V11|DynProto|Dynamic|Protocol|Witness|Conformance' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-mini go test ./compiler/internal/memorymodel -run 'V11|DynamicProtocolWitness' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-memoryfacts go test ./compiler/internal/memoryfacts -run 'V11|DynamicProtocolWitness' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-report go test ./compiler/internal/memoryfacts -run 'V11.*Report|V11Derived|ProtocolDispatchIntegrity' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v11-dynproto-memoryfacts go test ./compiler/internal/memoryfacts -count=1
```

Final gates passed and are recorded in
`.workflow/memory-ideal-vertical-slice-v11-dynproto/final-report.md`.

## Nonclaims

- No "Memory 100% complete" claim.
- No full trait-object/existential runtime proof.
- No complete witness-table ABI safety proof.
- No production dynamic dispatch runtime safety claim.
- No target parity.
- No performance claim.
- No broad noalias.
- No arbitrary unsafe/external pointer promotion.
- No clean-release claim while `git status --short` remains dirty.
