# Notes: Memory Ideal Vertical Slice v8 Report Integrity

## Chronological Notes

- 2026-06-05: User accepted v7 / `MEM-FFI-007` as completed
  `validated_narrow`, with explicit nonclaims for Memory 100%, FFI lifetime
  safety, and arbitrary external pointer safety.
- 2026-06-05: Highest next risk shifted to report/correlation/docs claim drift
  after v0-v7 accumulated exact evidence slices.
- 2026-06-05: Existing validators already reject many bad report shapes:
  missing/duplicate `source_fact_id`, unknown/missing `cost_class`, missing
  `normal_build_check` for dynamic checks, bad parent discipline,
  unsafe_unknown promotion, broad noalias, and trusted unsafe storage/lowering.
  v8 should make graph/report projection identity and completeness explicit.
- 2026-06-05: Preserve unrelated dirty worktree changes. Record
  `git status --short` in final evidence because `git diff --check` is not a
  clean-worktree proof.

## Nonclaims To Repeat

- No "Memory 100% complete".
- No new memory semantics.
- No optimizer rewrite.
- No arbitrary external pointer safety.
- No FFI/runtime lifetime proof.
- No target parity.
- No performance claim.
- No production runtime/ABI proof.
- No clean-release claim while `git status --short` is dirty.
