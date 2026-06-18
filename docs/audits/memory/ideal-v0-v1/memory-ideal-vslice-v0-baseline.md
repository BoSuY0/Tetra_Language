# Memory Ideal Vertical Slice v0 Baseline

Status: validated_with_gaps

This A0-lite audit re-validates the Memory Production Core v1 baseline before starting Memory Ideal
Vertical Slice v0. It does not claim broader memory soundness; it only confirms that the current
tree still has the documented source-of-truth, report-projection, unsafe, representation,
borrow/copy, alias, nonclaim, and target-tier boundaries needed for the v0 slice.

## Required Documents

All required baseline documents exist in the live checkout:

### Final Audit

- Document: `docs/audits/memory/production/memory-production-core-v1-final.md`.
- A0-lite role: final MPC v1 classification and command evidence.

### Artifact Map

- Document:
  `docs/audits/memory/production/memory-production-core-v1-artifact-map.md`.
- A0-lite role: artifact map for fact graph, validators, docs, and evidence.

### Nonclaims

- Document:
  `docs/audits/memory/production/memory-production-core-v1-nonclaims.md`.
- A0-lite role: explicit nonclaims and overclaim boundaries.

### Supported Surface

- Document:
  `docs/audits/memory/production/memory-production-core-v1-supported-surface.md`.
- A0-lite role: supported safe surface, unsafe boundary, and report surface.

### Gap Map

- Document: `docs/audits/memory/production/memory-production-core-v1-gap-map.md`.
- A0-lite role: remaining gaps and narrow/partial classifications.

### Report Schema

- Document: `docs/spec/memory/memory_report_schema_v1.md`.
- A0-lite role: memory report projection schema and validator invariants.

### Design Law

- Document: `docs/design/memory/memory_production_core_v1.md`.
- A0-lite role: design law for compiler-owned facts and report projection.

Verification command:

```bash
ls \
  docs/audits/memory/production/memory-production-core-v1-final.md \
  docs/audits/memory/production/memory-production-core-v1-artifact-map.md \
  docs/audits/memory/production/memory-production-core-v1-nonclaims.md \
  docs/audits/memory/production/memory-production-core-v1-supported-surface.md \
  docs/audits/memory/production/memory-production-core-v1-gap-map.md \
  docs/spec/memory/memory_report_schema_v1.md \
  docs/design/memory/memory_production_core_v1.md
```

Result: all seven paths were present.

## Baseline Assertions

### MemoryFactGraph Truth Source

- Status: validated.
- Evidence: `docs/design/memory/memory_production_core_v1.md:8` maps compiler
  facts to `MemoryFactGraph`.
- Evidence: `docs/audits/memory/production/memory-production-core-v1-final.md:34`
  classifies the graph as the truth source for report projection.

### Reports Are Projections

- Status: validated.
- Evidence: `docs/design/memory/memory_production_core_v1.md:12` says reports
  must not reconstruct facts the compiler did not own.
- Evidence: `docs/spec/memory/memory_report_schema_v1.md:5` says the report is
  a projection and not a source of truth.

### Unsafe Unknown Boundary

- Status: validated.
- Evidence: `docs/design/memory/memory_production_core_v1.md:27` rejects
  unsafe-to-safe promotion.
- Evidence: `docs/spec/memory/memory_report_schema_v1.md:171` rejects safe
  provenance paired with `unsafe_unknown`.

### Safe Metadata Assignment

- Status: validated.
- Evidence: `docs/design/memory/memory_production_core_v1.md:139` says
  `FieldInfo` entries are not `UserAssignable`.
- Evidence: `docs/design/memory/memory_production_core_v1.md:141` says
  assignment-target guards reject writes before lowering.

### Borrow/Copy/Copy Into

- Status: validated.
- Evidence:
  `docs/audits/memory/production/memory-production-core-v1-supported-surface.md:11`
  lists `borrow`, `copy`, and `copy_into`.
- Evidence: `docs/design/memory/memory_production_core_v1.md:54` describes the
  supported borrow/copy/copy_into projection.

### Mutable Alias/Inout

- Status: validated.
- Evidence: `docs/design/memory/memory_production_core_v1.md:64` scopes mutable
  alias/inout as conservative.
- Evidence: `docs/audits/memory/production/memory-production-core-v1-final.md:39`
  classifies MPC-5 as `conservative`.

### Borrow Checker Parity Nonclaim

- Status: validated.
- Evidence: `docs/audits/memory/production/memory-production-core-v1-nonclaims.md:8`
  lists full Rust-like borrow checker parity as a nonclaim.
- Evidence:
  `docs/audits/memory/production/memory-production-core-v1-supported-surface.md:134`
  repeats that boundary.

### Arbitrary Unsafe Pointer Safety Nonclaim

- Status: validated.
- Evidence:
  `docs/audits/memory/production/memory-production-core-v1-nonclaims.md:10`
  excludes arbitrary unsafe external pointer safety.
- Evidence:
  `docs/audits/memory/production/memory-production-core-v1-supported-surface.md:143`
  repeats that boundary.

### Full Actor Runtime Nonclaim

- Status: validated.
- Evidence:
  `docs/audits/memory/production/memory-production-core-v1-nonclaims.md:12`
  excludes full production actor runtime.
- Evidence: `docs/audits/memory/production/memory-production-core-v1-final.md:57`
  repeats that full actor runtime guarantees are explicit non-goals.

### Target Parity Nonclaim

- Status: validated.
- Evidence:
  `docs/audits/memory/production/memory-production-core-v1-nonclaims.md:13`
  excludes full target runtime parity.
- Evidence:
  `docs/audits/memory/production/memory-production-core-v1-supported-surface.md:144`
  excludes cross-target runtime memory parity without target evidence.

## Classification

Baseline classification: `validated_with_gaps`.

No A0-lite blocker was found. The baseline is true enough to proceed, but it is not gap-free: MPC v1
already records partial/generalization gaps around the representation metadata namespace, mutable
alias model, and future target/runtime claims. B1-min, MiniMemoryModel v0, B2a, B3a, and the minimal
report/correlation work may proceed, as long as they keep the v0 scope and nonclaims intact.

## Nonclaims Preserved

- This baseline is not a claim of perfect memory.
- This baseline is not a full borrow checker, full mutable alias model, raw pointer safety model,
  full actor/task runtime model, target parity, or performance claim.
- Future slices must still validate their own facts, reports, and tests.
