# Standard Library Spec Notes

This page anchors stdlib-specific spec policies.

- Naming and versioning policy (normative for release gating):
  [stdlib_naming_versioning.md](./stdlib_naming_versioning.md)

## Stable Core Surface (Current)

The current stable stdlib modules under `lib/core` are:

- `lib.core.capability`
- `lib.core.math`
- `lib.core.memory`
- `lib.core.slices`
- `lib.core.strings`

## Promotion Notes

Wave 7 promotions in this repository include:

- `lib.experimental.slices` -> `lib.core.slices`
- `lib.experimental.strings` -> `lib.core.strings`

`lib.experimental.*` entries remain available as compatibility shims, but stable
code should import `lib.core.*` directly.

## Stable Module Quality Gates

Stable `lib.core.*` modules are required to include:

- top-of-file docs comments
- at least one `tetra doctest` block
- an `// Effects: ...` metadata line (`none` or a comma-separated list)
- a checked smoke example under `examples/core_*_smoke.tetra`

`tools/cmd/verify-docs` enforces these requirements.
