# v1 Feature Status Decisions

This document resolves TODO 658 from `docs/plans/2026-04-26-tetra-language-todo.md`
for the current v1 planning baseline.

Companion freeze decisions for unresolved TODO 12/13/15 implementation items:
`../plans/v1_scope_freeze_backend_stdlib_ui.md`.

Canonical release scope for the current v0.1.3-to-v1.0.0 plan:
`./v1_scope.md`.

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
| Budget clauses (`budget`) | `implement in v1` | Static v1 MVP: `uses budget` plus `budget(<non-negative integer constant>)` is checked and lowered to deterministic local guards. Cross-function/runtime-wide and distributed budget accounting remains post-v1 in `v1_scope.md`. |
| Privacy clauses (checked privacy + consent/privacy types) | `implement in v1` | Static v1 MVP: privacy clauses, `secret.i32`/`SecretInt`, consent-token signatures, and privacy builtins are checked. Cryptographic isolation and distributed consent enforcement remain post-v1 in `v1_scope.md`. |
| UI syntax (`view`, `state`, binding/events/commands, typed style, accessibility metadata) | `implement in v1` | Scope is the metadata UI surface in `ui_v1.md`: checked declarations, deterministic UI JSON, web preview artifacts, native shell text sidecar, and smoke evidence. Full native widgets and runtime UI event dispatch are post-v1. |

## Spec Alignment

- Canonical Flow grammar/source surface is documented in
  [flow_syntax_v1.md](./flow_syntax_v1.md). The previous
  [flow_syntax_mvp.md](./flow_syntax_mvp.md) path is retained only as an alias.
- UI syntax, lowered metadata, web preview artifacts, and native shell sidecars
  are documented in [ui_v1.md](./ui_v1.md).
- `unsafe`, capability, privacy, consent, and budget behavior remain as
  documented in [unsafe.md](./unsafe.md), [capabilities.md](./capabilities.md),
  and [effects_capabilities_privacy_v1.md](./effects_capabilities_privacy_v1.md).
  The v1 guarantee is the static checked MVP; runtime-wide/distributed policy
  enforcement remains explicitly post-v1.
