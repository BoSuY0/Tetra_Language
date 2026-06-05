# Notes: Memory Ideal Vertical Slice v7 FFI

## Chronological Notes

- 2026-06-05: User accepted v6 / `MEM-BOUNDS-006` as completed
  `validated_narrow`, not "Memory 100%".
- 2026-06-05: Highest next risk shifted from bounds-check elimination to
  external/FFI/raw pointer boundary trust.
- 2026-06-05: Existing schema/docs already mention `checked_external_unknown`,
  `external_unknown`, `unsafe_unknown`, broad noalias rejection, and
  unsafe_unknown bounds-elimination rejection. v7 should convert this into an
  exact Memory Ideal correlation slice.
- 2026-06-05: Preserve unrelated dirty worktree changes. Record
  `git status --short` in final evidence because `git diff --check` is not a
  clean-worktree proof.
- 2026-06-05: v7 source/model/validator GREEN cluster passed after adding
  narrow PLIR-derived rows for `ffi_pointer_external_unknown`,
  `ffi_call_may_retain_borrow`,
  `ffi_noalias_invalidated_by_external_call`,
  `safe_wrapper_promotion_rejected_without_contract`, and
  `external_pointer_provenance_rejected`.
- 2026-06-05: v7 docs/schema/manifest cluster added exact four-row
  correlation and final audit docs, schema parent discipline, design/unsafe
  boundaries, and manifest docs entries. `validate-memory-correlation`,
  `validate-manifest`, and `verify-docs` passed with v7 cache names.
- 2026-06-05: focused gates, v0-v7 correlation regression, broad `go test`,
  canonical `scripts/ci/test.sh`, `graphify update .`, and `git diff --check`
  passed. `git status --short` remains heavily dirty/non-empty; final report
  records this as a release caveat rather than a clean-worktree claim.

## Nonclaims To Repeat

- No "Memory 100% complete".
- No arbitrary external pointer safety.
- No C/FFI lifetime safety.
- No safe wrapper promotion.
- No broad unsafe noalias or universal noalias.
- No target parity.
- No performance claim.
- No arbitrary external allocator provenance.
- No production runtime/ABI proof.
