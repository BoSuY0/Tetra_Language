# v0.3 Historical Boundary Guide

Status: historical user-facing map for the `v0.3.0` promotion boundary. The current public profile
is `v0.4.0`; use `docs/spec/core/current_supported_surface.md` for current release truth.

## Promoted In v0.3.0

- Enum payload match/catch/if-let: positional enum payload constructors and payload bindings are
  supported for the checked `v0.3.0` slice. Exhaustive unguarded enum match/catch diagnostics are
  part of the support boundary.
- Static protocol-bound generics: generic function type parameters with protocol bounds are
  validated during monomorphization, including same-module and cross-module conformance and
  visibility diagnostics.

## Still Experimental Or Planned

- Callable Level 1 remains experimental. The current support boundary is the Level 0 callable MVP
  and narrow symbol-backed non-capturing paths documented in
  `docs/spec/core/current_supported_surface.md`.
- Ownership/resource safety remains a conservative MVP. Full lifetime SSA is still planned.
- Capsule/Eco behavior remains local artifact and lock usability. Distributed EcoNet, production
  TetraHub, global trust scoring, and proof-carrying capsules remain outside the current support
  claim.
- WASI/Web runtime execution is runner-gated. `tetra run --target wasm32-wasi` is supported when a
  WASI runner is discoverable, and `tetra run --target wasm32-web` is supported when a
  Chromium-compatible browser runner is discoverable.

## Not Promoted By Preview Docs

Preview docs and smoke examples do not promote generic structs, explicit type arguments,
higher-ranked generics, runtime generic values, dynamic protocol dispatch, witness tables, trait
objects, captured closures, full first-class function values, distributed actors, native UI runtime
widgets, or full v1.0 language guarantees.

Use `compiler/compiler_facade.go`, `docs/generated/manifest.json`, and
`docs/spec/core/current_supported_surface.md` as the release-truth alignment points.
