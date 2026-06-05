# Safe Borrow Returns v1 Closure

Goal slice: P14.1 Safe Borrow Returns v1.

Baseline: `tetra.truthful-performance-core.baseline.20260602.v1`.

Status: complete for slice after focused implementation and verification.

## Scope

This slice closes safe borrowed return semantics for supported slice/String
borrowed views. It makes `-> borrow` useful only when the returned borrowed
view is tied to a caller-visible borrowed source, and it keeps unsafe or
unknown provenance conservative.

## Implemented Rules

| Rule | Evidence |
|---|---|
| Borrowed return signatures parse and preserve `ReturnOwnership`. | `compiler/internal/frontend/parser.go`, `compiler/internal/frontend/parser_test.go` |
| Borrowed returns must originate from caller-visible sources. | `compiler/internal/semantics/checker.go`, `compiler/tests/semantics/borrow_copy_test.go` |
| Local owned allocation cannot be returned as borrow. | `TestBorrowedSliceAndStringBorrowedReturnContracts` |
| Unsafe unknown provenance cannot be returned as borrow. | `TestBorrowedReturnRejectsUnsafeUnknownProvenance` |
| Ambiguous branch sources are rejected unless both sources tie to the same caller-visible source. | `TestBorrowedReturnBranchOriginConsistency` |
| Function-typed borrowed return ownership is checked. | `TestFunctionTypedBorrowedReturnOwnershipContract`, `TestBorrowedReturnForwardingRequiresBorrowReturn` |
| Module/interface boundaries preserve borrowed return contracts. | `TestGenerateInterfaceFromSourcePreservesBorrowedReturnContract`, `compiler/interface.go` |
| PLIR preserves borrow/provenance/no-escape facts for safe borrowed views. | `compiler/internal/plir/plir.go`, `compiler/internal/plir/plir_test.go` |
| Reports expose borrowed-return no-allocation/proof facts and report flags do not change semantics. | `compiler/explain_reports_test.go` |

## Code Changes

- `compiler/tests/semantics/borrow_copy_test.go` adds
  `TestBorrowedReturnRejectsUnsafeUnknownProvenance`.
- `compiler/internal/semantics/checker.go` now rejects explicit borrowed
  returns whose source collapses to the synthetic `"<borrow>"` marker, with
  diagnostic text:

```text
borrowed slice return requires caller-visible borrow source
```

This turns expression-level unsafe raw-slice borrowed returns into an explicit
borrow-source failure instead of a later generic region mismatch.

## Verification Evidence

RED evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics -run TestBorrowedReturnRejectsUnsafeUnknownProvenance -count=1
```

Initial result: failed before implementation because the fixture exposed a
missing explicit caller-visible borrow source through existing region
validation.

GREEN/focused evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics -run TestBorrowedReturnRejectsUnsafeUnknownProvenance -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/frontend -run 'Borrow|ReturnOwnership' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics -run 'Borrowed|GenerateInterfaceFromSourcePreservesBorrowedReturnContract|RawSliceFromParts' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/plir -run 'Borrow|RawSliceExternal|RawDerived|VerifierRejects.*Provenance' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'BuildReportsShowBorrowedReturnNoAllocationAndCopyOwnership|ReportFlagsDoNotChangeBorrowedReturnFailure|BuildReportsShowBorrowCopyProvenanceAndAllocationIntent' -count=1
```

Relevant package evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/frontend ./compiler/internal/semantics ./compiler/tests/semantics ./compiler/internal/plir ./compiler -count=1
```

Result: pass.

Final hygiene evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
graphify update .
```

Result: pass. Graphify rebuilt `18876 nodes, 60608 edges, 1101 communities`.

Additional final checks:

```bash
git diff --check
python3 <targeted trailing-whitespace scan for sidecars/P14.1 files>
rg -n 'tetra_surface_release_promotion_v1_full_plan|source_plan: /home/tetra/Downloads/tetra_surface_release|Active slice: Section|Surface Release Promotion v1' GOAL.md PLAN.md ATTEMPTS.md NOTES.md CONTROL.md || true
```

Result: pass. The drift scan found only the explicit drift-guard references in
`GOAL.md` and `CONTROL.md`; no sidecar names the Surface plan as the active
goal.

## Non-Claims

- P14.1 does not implement named lifetimes.
- P14.1 does not implement a full borrow graph or noalias optimizer contract;
  that remains P14.2.
- P14.1 does not promote raw or external provenance to known.
- P14.1 does not remove runtime checks or change safe semantics through report
  flags.
