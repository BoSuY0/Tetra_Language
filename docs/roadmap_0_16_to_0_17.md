# Roadmap v0.16 → v0.17 (Effects / `uses` Enforcement)

Focus: make the already parsed `uses` clause a checked MVP effect surface while
leaving backend, IR, ABI, runtime, ownership, async, protocols, and Eco out of
scope.

## P0 — Enforced effect declarations

- User functions declare direct and callee effects with `uses`.
- Builtin effects come from the existing manifest metadata.
- `print` requires `io`; scoped islands require `alloc, islands, mem`; actor
  builtins require `actors`.
- Internal `__rt.*` runtime modules are exempt from missing-uses diagnostics but
  still reject unknown effect names.

## P1 — Safety boundary stays separate

- `uses` is not a permission token and does not replace `unsafe`.
- Capability-producing and raw memory builtins still require `unsafe` plus the
  appropriate `cap.mem` or `cap.io` value.

## P2 — Deferred work

Effect inference, effect polymorphism, ownership, async, protocols, typed
errors, UI, and Eco remain v0.18+ work.
