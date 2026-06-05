# P1-semantics-registry Result

Status: completed_read_only

Sub-agent: Ramanujan (`019e91ed-31a3-7e42-a5b0-d92e7cde22c2`)

## Accepted Leads

- The main assignment cutoffs are in
  `compiler/internal/semantics/resolution.go`, especially
  `resolveAssignTarget`, `rejectRepresentationMetadataExprAssignment`,
  `rejectCollectionInternalAssignment`, `rejectRepresentationMetadataAssignment`,
  and `isReservedRepresentationMetadataField`.
- `compiler/internal/semantics/checker.go` also calls the metadata assignment
  guard during assignment collection, giving a second pre-lowering path.
- Existing tests already cover `ptr`/`len` metadata behavior across
  `compiler/internal/semantics/representation_metadata_test.go`,
  `compiler/tests/semantics/array_mvp_test.go`,
  `compiler/tests/semantics/slice_bool_test.go`,
  `compiler/tests/semantics/slice_view_test.go`, and
  `compiler/tests/semantics/string_metadata_test.go`.

## Implementation Guidance

- Prefer a centralized registry in semantics over scattered string checks.
- Keep existing diagnostics and `ptr`/`len` behavior stable.
- Add tests for reserved names beyond `ptr`/`len`, especially `owner_id`,
  `region_id`, `provenance_id`, `borrow_source`, `storage_class`, and
  `unsafe_class` where syntactically expressible.

## Risks

- Reserved metadata names beyond `ptr`/`len` appear to have less direct test
  evidence.
- User structs with similarly named ordinary fields may need careful policy:
  the v0 plan requires compiler-owned representation metadata to be reserved,
  but should not accidentally ban unrelated user fields unless the current type
  model treats them as representation metadata.
