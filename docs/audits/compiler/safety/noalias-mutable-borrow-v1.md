# NoAlias Mutable Borrow v1 Closure

Goal slice: P14.2 NoAlias / Mutable Borrow v1.

Baseline: `tetra.truthful-performance-core.baseline.20260602.v1`.

Status: complete for slice after focused implementation and verification.

## Scope

This slice adds a narrow, verifier-backed `no_alias` PLIR fact for exclusive mutable
slice/String-like parameters. It does not enable a new optimizer transformation. Optimizer use
remains gated: current optimizer paths do not consume PLIR `no_alias`, and report guidance continues
to leave vectorization blocked by `vector.alias_unknown` until a validator-backed pass exists.

## Implemented Rules

- Rule: `inout []T` is represented as a mutable borrow in PLIR.
  - Evidence:
    - `compiler/internal/plir/plir.go`
    - `TestFromCheckedProgramRecordsNoAliasForExclusiveInoutSliceParam`
- Rule: `no_alias` is emitted only for memory-backed `inout` params.
  - Constraints: parameter provenance, bounded function lifetime, `borrowed_mut`.
  - Extra constraint: `region_alive`.
  - Evidence:
    - `compiler/internal/plir/plir.go`
    - `compiler/internal/plir/verify.go`
- Rule: overlapping mutable borrows are rejected.
  - Evidence:
    - `TestOwnershipRejectsOverlappingMutableInoutSliceBorrow`
    - `compiler/internal/semantics/semantics_expressions.go`
- Rule: immutable borrow plus mutable borrow aliases are rejected.
  - Evidence:
    - `TestOwnershipRejectsBorrowInoutAlias`
    - `TestOwnershipAllowsBorrowInoutWithDistinctLocals`
- Rule: raw unsafe pointer exposure kills `no_alias` for the exposed root.
  - Evidence: `TestFromCheckedProgramDoesNotClaimNoAliasAfterRawInoutExposure`
- Rule: forged `no_alias` facts are rejected by the PLIR verifier.
  - Evidence:
    - `TestVerifierRejectsNoAliasWithoutExclusiveMutableBorrow`
    - `TestVerifierRejectsNoAliasForExternalProvenance`
- Rule: reports expose `no_alias` as evidence without changing semantics.
  - Evidence: `TestBuildReportsShowInoutNoAliasProofFact`
- Rule: optimizer use remains validation-gated.
  - Evidence:
    - `compiler/internal/opt`
    - `compiler/internal/validation` package gates passed
    - no optimizer pass consumes PLIR `no_alias` in this slice

## Code Changes

- `compiler/internal/plir/plir.go` now records raw pointer exposure roots from
  `core.raw_slice_*_from_parts(xs.ptr, ...)` and emits `FactNoAlias` only after the function body
  has been scanned.
- `compiler/internal/plir/verify.go` now rejects `no_alias` unless the value is a mutable borrowed
  parameter with parameter provenance, bounded lifetime, `borrowed_mut`, `region_alive`, and
  `provenance_known` evidence.
- `compiler/internal/plir/plir_test/plir_test.go`, `compiler/tests/ownership/ownership_test.go`, and
  `compiler/compiler_external_test.go` add focused P14.2 coverage.

## Verification Evidence

RED evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler/internal/plir \
  -run 'NoAlias|RawInout' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler \
  -run 'BuildReportsShowInoutNoAliasProofFact' -count=1
```

Initial result: failed because PLIR emitted `borrowed_mut` but no `no_alias`, and the verifier
accepted a forged `no_alias` fact.

Focused GREEN evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler/internal/plir \
  -run 'NoAlias|RawInout' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler/tests/ownership \
  -run 'OverlappingMutableInoutSliceBorrow' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler \
  -run 'BuildReportsShowInoutNoAliasProofFact' -count=1
```

Result: pass.

Relevant package evidence:

```bash
PLIR_RUN='Borrow|RawSlice|RawDerived|NoAlias'
PLIR_RUN="$PLIR_RUN|VerifierRejects.*Provenance|VerifierRejectsNoAlias"
OWNERSHIP_RUN='Inout|BorrowInout|OverlappingMutable'
OWNERSHIP_RUN="$OWNERSHIP_RUN|BorrowedProjectionAsInout|BorrowDerivedValueAsInout"
COMPILER_RUN='BuildReportsShowInoutNoAliasProofFact'
COMPILER_RUN="$COMPILER_RUN|BuildReportsShowBorrowedReturnNoAllocationAndCopyOwnership"
COMPILER_RUN="$COMPILER_RUN|ReportFlagsDoNotChangeBorrowedReturnFailure"
COMPILER_RUN="$COMPILER_RUN|BuildReportsShowBorrowCopyProvenanceAndAllocationIntent"

GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler/internal/plir \
  -run "$PLIR_RUN" -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler/tests/ownership \
  -run "$OWNERSHIP_RUN" -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler/tests/semantics \
  -run 'Borrowed|RawSliceFromParts|SliceRaw|BorrowCopy' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler \
  -run "$COMPILER_RUN" -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go test ./compiler/internal/opt ./compiler/internal/validation -count=1
```

Result: pass.

Final hygiene evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test \
  ./compiler/internal/plir \
  ./compiler/tests/ownership \
  ./compiler/tests/semantics \
  ./compiler/internal/opt \
  ./compiler/internal/validation \
  ./compiler \
  -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan \
  go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
git diff --check
graphify update .
```

Result: pass. Graphify rebuilt `18902 nodes, 60679 edges, 1077 communities`. The sidecar drift scan
found only explicit drift-guard references in `GOAL.md` and `CONTROL.md`.

## Non-Claims

- P14.2 does not implement a broad interprocedural alias analysis.
- P14.2 does not allow safe code to forge noalias, provenance, or lifetime facts.
- P14.2 does not enable vectorization or any optimizer transformation from `no_alias`; optimizer
  consumption remains a future validator-backed slice.
- P14.2 does not turn raw or external provenance into known provenance.
