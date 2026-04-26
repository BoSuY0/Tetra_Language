# v1 Feature Status Decisions

This document resolves TODO 658 from `docs/plans/2026-04-26-tetra-language-todo.md`
for the current v1 planning baseline.

Companion freeze decisions for unresolved TODO 12/13/15 implementation items:
`../plans/v1_scope_freeze_backend_stdlib_ui.md`.

Decision labels used here:

- `implement in v1`: required in the v1 delivery scope.
- `defer post-v1`: explicitly out of v1 scope.
- `block behind prerequisite`: implementation must not start until listed
  prerequisites are complete; status is re-evaluated afterward.

## Decisions (2026-04-26)

| Feature | Decision | Notes / prerequisites |
| --- | --- | --- |
| Closures | `defer post-v1` | `fn`/`fun` literals already produce planned-feature diagnostics in the Flow MVP parser. Closure capture semantics should not land before the borrow/lifetime surface is stabilized. |
| Semantic clauses (`noalloc`, `noblock`, `realtime`, `nothrow`) | `defer post-v1` | The Flow MVP currently documents these as deferred and diagnostics-only. v1 keeps this posture instead of introducing partial enforcement semantics. |
| Budget clauses (`budget`) | `block behind prerequisite` | Prerequisites: (1) effect propagation through generics/protocols is stable, (2) runtime budget accounting and diagnostics contract exist, (3) release-profile tests cover both static and runtime budget failures. |
| Privacy clauses (checked privacy + consent/privacy types) | `block behind prerequisite` | Prerequisites: (1) secret/privacy type model is specified, (2) consent-token MVP semantics are specified, (3) capability attenuation/capsule permission checks exist to enforce privacy boundaries. |
| UI syntax (`view`, `state`, binding/events/commands, typed style, accessibility metadata) | `block behind prerequisite` | Prerequisites: (1) a dedicated UI syntax spec exists, (2) backend architecture for `wasm32-web` and native shell is finalized, (3) UI smoke apps are defined for release verification. |

## Spec Alignment

- Flow/Core syntax remains as documented in
  [flow_syntax_mvp.md](./flow_syntax_mvp.md): closures and semantic clauses are
  diagnostics-only in the current MVP.
- `unsafe` and capability behavior remain as documented in
  [unsafe.md](./unsafe.md) and [capabilities.md](./capabilities.md); privacy
  clauses are not part of the currently enforced v1-safe surface until their
  prerequisites are complete.
