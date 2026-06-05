# Memory Ideal Vertical Slice v2

## Goal
Extend the narrow memory correlation pattern to function-typed values and
callback boundaries for exactly `MEM-BORROW-004`, `MEM-BORROW-005`, and
`MEM-ALIAS-002`.

## Success Criteria
- v2 correlation and final audit docs exist and validate.
- `MemoryFactGraph` exposes only the narrow facts/projections:
  `function_value_contains_borrow`, `callback_arg_contains_borrow`, and
  `callback_inout_conservative`.
- Validators exist for function-value borrow escape, callback borrow escape, and
  callback alias conservatism.
- MiniMemoryModel and semantics tests cover required positive/negative cases.
- Focused gates, full gates, manifest/docs validators, `git diff --check`, and
  `graphify update .` pass before completion.

## Current Context
- v0/v1 patterns already exist for struct, optional, enum payload, and generic
  wrapper borrow carriers.
- Graphify MCP identified `function_types.go` as the semantic hub for
  function-typed locals, struct fields, enum payloads, and captured call
  metadata.
- The worktree is dirty from prior slices; preserve unrelated changes.

## Constraints
- Do not implement full first-class callable ABI.
- Do not broaden captured/escaping closures, protocol/interface values,
  async/task/actor boundaries, raw pointer semantics, target parity,
  performance claims, or broad noalias.
- Unknown callback targets remain conservative and must not emit trusted facts.
- Use persistent Go caches under `.cache/`.

## Risks
- Existing function-typed syntax may already reject some cases; tests must
  classify that as conservative instead of widening callable support.
- Report projection and correlation validator may have row-specific exactness
  checks inherited from v0/v1.
- Manifest tests often mirror `docs/generated/manifest.json`; fixtures may need
  synchronized updates.

## Approval Required
No external, destructive, or irreversible steps are planned. Ask before any
destructive Git operation, mass rename, force push, external write, or broad
codemod.

## Work Packets
- P0 discovery: map v1 patterns and callback/function-typed semantic surface.
- P1 memoryfacts/model: RED/GREEN for source facts, projections, validators,
  and MiniMemoryModel v2.
- P2 semantics: RED/GREEN positive and negative callback/function-typed cases.
- P3 docs/audit/manifest: correlation, final audit, schema, design doc if
  needed, manifest and fixtures.
- P4 verification: focused gates, broad gates, docs/manifest, diff check,
  graphify update, workflow final report.

## Integration Policy
Accept packet findings only when backed by file paths, test output, or command
evidence. If a packet would require excluded scope, reject that expansion and
record the row as conservative where allowed.

## Verification
Run the exact focused and full gate list from `GOAL.md` with persistent caches.
Do not mark complete until all required checks pass or unrelated failures are
explicitly classified with evidence.

## Reusable Artifacts
The v2 workflow directory, packet results, final report, audit docs, and updated
schema/manifest become the reusable recipe for later memory vertical slices.
