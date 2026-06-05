# Memory Cost Model

Status: Memory Production Core v1 cost-class contract.

This document defines the bounded memory cost vocabulary used by
`tetra.memory-report.v1` rows and by performance blocker reports. The model is
evidence and validation metadata. It does not introduce a runtime mode, a
benchmark claim, or an optimizer promotion by itself.

## Cost Classes

`zero_cost_proven`

: A compiler-owned compile-time proof removes a runtime memory cost. Examples
  include validated allocation-base metadata, provenance facts, noalias facts,
  and storage-lowering facts whose actual lowering matches the trusted plan.

`dynamic_check_required`

: Proof is unavailable or incomplete, so a check remains in the normal build.
  Rows with this class must carry `normal_build_check: true`. A validator must
  reject an optimization claim with `dynamic_check_required` unless the check
  remains in the normal build.

`instrumentation_only`

: Debug, audit, explanation, or report instrumentation. It is not required for
  default safe-program semantics and must not be used as optimization proof.

`unsupported_rejected`

: The compiler rejects the unsupported memory path before treating it as a safe
  or optimized path. Rejected raw-pointer and raw-slice cases use this class.

`conservative_fallback`

: The compiler keeps a safe but possibly slower fallback, such as heap storage,
  explicit copy, checked path, or unknown unsafe provenance.

## Required Rules

- A normal build does not run heavy validators at runtime.
- Compile-time validators are allowed.
- Report generation is optional and artifact-only.
- report generation is optional and artifact-only.
- Safe proven paths avoid redundant runtime checks only when compiler-owned
  facts prove the check unnecessary.
- `unsafe_unknown` may be checked, trapped, or conservative, but never optimized
  as trusted.
- unsafe_unknown may be checked, trapped, or conservative, but never optimized as trusted.
- `cost_class` is required on memory report rows.
- `normal_build_check` is required when `cost_class` is
  `dynamic_check_required`.

## Report Boundary

`cost_class` is projected from compiler-owned memory facts or conservative
validator state. Validators may reject malformed reports, but they must not
invent a stronger cost class from report text alone.

Performance blocker rows use the same cost vocabulary so that blockers such as
missing dominance, unknown call escape, no noalias proof, stack fallback, and
actor-boundary copies are explanation metadata rather than hidden performance
claims.
