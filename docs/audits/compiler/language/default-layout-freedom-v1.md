# Default Layout Freedom v1

Status: P21.0 evidence slice.

## Scope

P21.0 records the layout/ABI boundary as explicit compiler evidence:

- default struct layout is compiler-owned;
- `repr(C)` locks layout;
- public ABI/exported FFI aggregate boundaries require explicit `repr(C)`;
- `.layout.json` reports show per-struct decisions.

The compiler now emits `.layout.json` schema version 2 with policy
`p21.0_default_layout_freedom_v1`. Each struct row records its representation, source field order,
current checked field layout, ABI-lock status, public ABI status, allowed layout-transform names for
default structs, denied transform names for `repr(C)`, a decision such as `compiler_owned_default`
or `abi_locked_repr_c`, and the reason text behind that decision.

## Public ABI Boundary

An `@export` function that exposes a default-layout struct directly, through an array/optional
wrapper, or through fields of an exposed `repr(C)` struct is rejected before codegen. The diagnostic
says the exported parameter or return type requires explicit `repr(C)` because the default Tetra
layout is compiler-owned and has no public ABI.

`repr(C)` aggregate exports still flow to the target-specific ABI validators. Those validators
remain responsible for per-target aggregate ABI support and can reject shapes even after the
explicit-repr gate passes.

## Validator

`ValidateLayoutReport` rejects:

- schema, kind, target, policy, or summary drift;
- a `repr(C)` row that allows `field_reordering`, `padding_removal`, `hot_cold_splitting`,
  `scalar_replacement`, or `aos_to_soa`;
- a default-layout row that claims an ABI lock;
- an exported public ABI row that is missing explicit `repr(C)`;
- incomplete or placeholder report text.

This keeps layout reports as evidence, not a hidden optimization mode.

## Non-Claims

This slice does not implement field reordering, padding removal, hot/cold splitting, scalar
replacement, or AoS-to-SoA transformation. It does not claim a C ABI for default structs, a
performance change, a runtime behavior change, or a user-visible layout optimization.

## Verification

Focused evidence command:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/semantics ./compiler -run 'TestExportedDefaultStructRequiresExplicitRepr|TestExportedReprCStructPassesExplicitReprGate|TestBuildLayoutReportRecordsP21|TestValidateLayoutReportRejectsFakeP21|TestNativeTargetsRejectExportedAggregateFFI|TestBuildExplainReportsTruthProofAndAllocationArtifacts' -count=1
```
