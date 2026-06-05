# NoAlias Mutable Borrow v1 Closure

Goal slice: P14.2 NoAlias / Mutable Borrow v1.

Baseline: `tetra.truthful-performance-core.baseline.20260602.v1`.

Status: complete for slice after focused implementation and verification.

## Scope

This slice adds a narrow, verifier-backed `no_alias` PLIR fact for exclusive
mutable slice/String-like parameters. It does not enable a new optimizer
transformation. Optimizer use remains gated: current optimizer paths do not
consume PLIR `no_alias`, and report guidance continues to leave vectorization
blocked by `vector.alias_unknown` until a validator-backed pass exists.

## Implemented Rules

| Rule | Evidence |
|---|---|
| `inout []T` is represented as a mutable borrow in PLIR. | `compiler/internal/plir/plir.go`, `TestFromCheckedProgramRecordsNoAliasForExclusiveInoutSliceParam` |
| `no_alias` is emitted only for memory-backed `inout` params with parameter provenance, bounded function lifetime, `borrowed_mut`, and `region_alive`. | `compiler/internal/plir/plir.go`, `compiler/internal/plir/verify.go` |
| Overlapping mutable borrows are rejected by existing active call-argument ownership tracking. | `TestOwnershipRejectsOverlappingMutableInoutSliceBorrow`, `compiler/internal/semantics/exprs.go` |
| Immutable borrow plus mutable borrow aliases are rejected; distinct roots remain allowed. | `TestOwnershipRejectsBorrowInoutAlias`, `TestOwnershipAllowsBorrowInoutWithDistinctLocals` |
| Raw unsafe pointer exposure kills `no_alias` for the exposed root. | `TestFromCheckedProgramDoesNotClaimNoAliasAfterRawInoutExposure` |
| Forged `no_alias` facts are rejected by the PLIR verifier. | `TestVerifierRejectsNoAliasWithoutExclusiveMutableBorrow`, `TestVerifierRejectsNoAliasForExternalProvenance` |
| Reports expose `no_alias` as evidence without changing semantics. | `TestBuildReportsShowInoutNoAliasProofFact` |
| Optimizer use remains validation-gated. | `compiler/internal/opt` and `compiler/internal/validation` package gates passed; no optimizer pass consumes PLIR `no_alias` in this slice |

## Code Changes

- `compiler/internal/plir/plir.go` now records raw pointer exposure roots from
  `core.raw_slice_*_from_parts(xs.ptr, ...)` and emits `FactNoAlias` only after
  the function body has been scanned.
- `compiler/internal/plir/verify.go` now rejects `no_alias` unless the value is
  a mutable borrowed parameter with parameter provenance, bounded lifetime,
  `borrowed_mut`, `region_alive`, and `provenance_known` evidence.
- `compiler/internal/plir/plir_test.go`, `compiler/tests/ownership/ownership_test.go`,
  and `compiler/explain_reports_test.go` add focused P14.2 coverage.

## Verification Evidence

RED evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/plir -run 'NoAlias|RawInout' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'BuildReportsShowInoutNoAliasProofFact' -count=1
```

Initial result: failed because PLIR emitted `borrowed_mut` but no `no_alias`,
and the verifier accepted a forged `no_alias` fact.

Focused GREEN evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/plir -run 'NoAlias|RawInout' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/ownership -run 'OverlappingMutableInoutSliceBorrow' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'BuildReportsShowInoutNoAliasProofFact' -count=1
```

Result: pass.

Relevant package evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/plir -run 'Borrow|RawSlice|RawDerived|NoAlias|VerifierRejects.*Provenance|VerifierRejectsNoAlias' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/ownership -run 'Inout|BorrowInout|OverlappingMutable|BorrowedProjectionAsInout|BorrowDerivedValueAsInout' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics -run 'Borrowed|RawSliceFromParts|SliceRaw|BorrowCopy' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'BuildReportsShowInoutNoAliasProofFact|BuildReportsShowBorrowedReturnNoAllocationAndCopyOwnership|ReportFlagsDoNotChangeBorrowedReturnFailure|BuildReportsShowBorrowCopyProvenanceAndAllocationIntent' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/opt ./compiler/internal/validation -count=1
```

Result: pass.

Final hygiene evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/plir ./compiler/tests/ownership ./compiler/tests/semantics ./compiler/internal/opt ./compiler/internal/validation ./compiler -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
git diff --check
graphify update .
```

Result: pass. Graphify rebuilt `18902 nodes, 60679 edges, 1077 communities`.
The sidecar drift scan found only explicit drift-guard references in
`GOAL.md` and `CONTROL.md`.

## Non-Claims

- P14.2 does not implement a broad interprocedural alias analysis.
- P14.2 does not allow safe code to forge noalias, provenance, or lifetime
  facts.
- P14.2 does not enable vectorization or any optimizer transformation from
  `no_alias`; optimizer consumption remains a future validator-backed slice.
- P14.2 does not turn raw or external provenance into known provenance.
