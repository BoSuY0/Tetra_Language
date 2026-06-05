# Memory Ideal Vertical Slice v3

## Goal

Extend the narrow memory correlation pattern to interface/protocol/existential-
like boundaries for exactly `MEM-BORROW-006`, `MEM-BORROW-007`, and
`MEM-ALIAS-003`.

## Success Criteria

- v3 correlation and final audit docs exist and validate.
- `MemoryFactGraph` exposes only the narrow facts/projections:
  `interface_value_contains_borrow`,
  `protocol_dispatch_borrow_conservative`, and
  `protocol_dispatch_noalias_conservative`.
- Validators exist for interface borrow escape, protocol dispatch borrow
  conservatism, and protocol dispatch alias conservatism.
- MiniMemoryModel and semantics tests cover required positive/negative cases.
- Focused gates, full gates, manifest/docs validators, `git diff --check`, and
  `graphify update .` pass before completion.

## Current Context

- v0/v1/v2 patterns already exist for struct/optional, enum/generic wrappers,
  function-typed values, and callback boundaries.
- Current supported surface documents static protocol conformance and explicitly
  rejects runtime protocol values, trait objects, witness tables, dynamic
  dispatch, conformance-table lookup, and runtime existential ABI.
- The worktree is dirty from prior slices; preserve unrelated changes.

## Constraints

- Do not implement full trait-object/protocol existential runtime.
- Do not implement or promote full dynamic dispatch.
- Do not add witness tables, conformance-table lookup, async/task/actor
  boundary expansion, raw pointer expansion, target parity, broad noalias, or
  performance claims.
- Unknown/dynamic dispatch remains conservative and must not emit trusted facts.
- Use persistent Go caches under `.cache/`.

## Risks

- Existing syntax may reject runtime protocol/existential values. Treat that as
  conservative evidence instead of widening the language/runtime.
- Statically known protocol targets may be represented through generic/static
  conformance or direct calls rather than actual existential values.
- Report projection and correlation validators may need exact-row v3 handling
  while preserving v0/v1/v2 behavior.
- Manifest tests mirror `docs/generated/manifest.json`; fixtures may need
  synchronized updates.

## Approval Required

No external, destructive, or irreversible steps are planned. Ask before any
destructive Git operation, mass rename, force push, dependency change, public
ABI change, or broad codemod.

## Work Packets

- P0 discovery: map v2 patterns and current interface/protocol/static
  conformance surface.
- P1 memoryfacts/model: RED/GREEN for v3 source facts, projections, validators,
  and MiniMemoryModel cases.
- P2 semantics: RED/GREEN positive and negative interface/protocol cases,
  keeping unsupported dynamic/existential behavior conservative.
- P3 docs/audit/manifest: correlation, final audit, schema, design doc if
  needed, generated manifest and fixtures.
- P4 verification: focused gates, broad gates, docs/manifest, diff check,
  Graphify update, workflow final report.

## Integration Policy

Accept packet findings only when backed by file paths, test output, or command
evidence. If a packet would require excluded scope, reject that expansion and
record the row as conservative where allowed.

## Verification

Run the exact focused and full gate list from `GOAL.md` with persistent caches.
Do not mark complete until all required checks pass or unrelated failures are
explicitly classified with evidence.

## Reusable Artifacts

The v3 workflow directory, packet results, final report, audit docs, and
updated schema/manifest become the reusable recipe for later memory vertical
slices.
