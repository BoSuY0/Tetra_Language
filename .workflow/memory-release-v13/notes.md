# MEM-RELEASE-013 Workflow Notes

- v13 is a release/evidence hygiene slice, not a new memory semantics slice.
- Dirty worktree state is a release blocker unless clean or fully triaged.
- Do not clean, delete, revert, or archive unrelated dirty entries without
  explicit human approval.
- Preserve v12 as accepted `validated_narrow` with `proceed_with_blockers`.
