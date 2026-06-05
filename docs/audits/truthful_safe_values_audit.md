# Truthful Safe Values Audit

Status: 2026-06-01 implementation slice.

## Invariant

Safe code cannot mutate slice or string representation metadata.

## Central Gate

`compiler/internal/semantics/resolution.go` rejects assignment to `ptr` and
`len` when any field projection step reaches a fixed array, slice, or string
representation.

## Covered Regression Tests

- `TestSliceMetadataAssignmentRejectsLen`
- `TestSliceMetadataAssignmentRejectsPtr`
- `TestSliceMetadataAssignmentRejectsNestedLen`
- `TestSliceMetadataAssignmentRejectsNestedPtr`
- `TestSliceMetadataAssignmentRejectsGenericNestedPtr`
- `TestSliceMetadataAssignmentRejectsInoutLen`
- `TestSliceMetadataAssignmentRejectsOptionalPayloadLen`
- `TestSliceMetadataAssignmentRejectsEnumPayloadPtr`
- `TestRawSliceFromPartsRequiresUnsafe`
- `TestRawSliceFromPartsUnsafeGatewayTypeChecks`
- `TestBuildRawSliceFromPartsSmoke`
- `TestSliceViewConstructorsTypeCheckForSupportedSlices`
- `TestBuildSliceWindowPrefixSuffixSmoke`
- `TestBuildSliceViewConstructorsAllElementKindsSmoke`
- `TestSliceWindowRejectsInvalidRangesBeforeConstruction`
- `TestForSliceWindowLoopUsesProofTaggedUncheckedIndexLoad`
- `TestFromCheckedProgramRecordsSliceWindowProvenanceAndRange`
- `TestBuildBoundsReportShowsWindowLoopCheckRemoval`
- `TestStringMetadataAssignmentRejectsLen`
- `TestStringMetadataAssignmentRejectsPtr`
- `TestStringMetadataAssignmentRejectsNestedLen`
- `TestStringMetadataAssignmentRejectsNestedPtr`
- `TestStringMetadataAssignmentRejectsGenericNestedLen`

## Current Limitations

Safe `window`, `prefix`, and `suffix` are implemented for `[]u8`, `[]u16`,
`[]i32`, and `[]bool`. String views, `copy`, and `borrow` remain future work.
Raw slice construction is available only through the audited
`core.raw_slice_*_from_parts` unsafe gateway family and carries conservative
external provenance in PLIR.
