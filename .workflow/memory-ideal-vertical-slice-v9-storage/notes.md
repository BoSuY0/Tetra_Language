# Notes: Memory Ideal Vertical Slice v9 Storage

## Chronological Notes

- 2026-06-06: User accepted v8 / `MEM-REPORT-008` as completed
  `validated_narrow`, with explicit nonclaims for Memory 100%, new memory
  semantics, optimizer/target/performance/FFI/runtime proof, and arbitrary
  external pointer safety.
- 2026-06-06: Highest next risk shifted to storage decision mismatch:
  Stack/Region/Task/Actor trusted storage must not be validated when escape
  evidence exists.
- 2026-06-06: Existing `allocplan.VerifyPlan` already rejects escaping
  allocations that use trusted local storage classes. v9 should elevate this
  into exact Memory Ideal evidence with report/correlation/docs discipline.
- 2026-06-06: Existing memoryfacts tests preserve stack/function-temp heap
  fallback rows as non-validated projection evidence. v9 should add exact
  source/reason preservation and boundary conservatism requirements.
- 2026-06-06: Preserve unrelated dirty worktree changes. Record
  `git status --short` in final evidence because `git diff --check` is not a
  clean-worktree proof.

## Nonclaims To Repeat

- No "Memory 100% complete".
- No full region inference.
- No new broad memory semantics.
- No optimizer-wide allocation correctness proof.
- No arbitrary external pointer safety.
- No arbitrary FFI/runtime lifetime proof.
- No production actor runtime proof.
- No full async lifetime system.
- No target parity.
- No performance claim.
- No clean-release claim while `git status --short` is dirty.
