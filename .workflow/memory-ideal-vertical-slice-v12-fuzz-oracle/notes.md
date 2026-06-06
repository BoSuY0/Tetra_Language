# MEM-FUZZ-012 Workflow Notes

- 2026-06-06: v11 accepted `validated_narrow`; v12 starts from dirty worktree
  caveat and must not claim clean release.
- 2026-06-06: Existing oracle has seven categories and seven invariants, but
  v12 requires explicit release-evidence coverage across v0-v11 plus artifact
  discipline for crash/miscompile and blocking memory failures.
- 2026-06-06: Final v12 gates passed. `git status --short` remains non-empty,
  so release/worktree decision stays `proceed_with_blockers`.
