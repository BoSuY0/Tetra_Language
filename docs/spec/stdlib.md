# Standard Library Spec Notes

This page anchors stdlib-specific spec policies.

- Naming and versioning policy (normative for release gating):
  [stdlib_naming_versioning.md](./stdlib_naming_versioning.md)

## Stable Core Surface (Current)

The current stable stdlib modules under `lib/core` are:

- `lib.core.async`
- `lib.core.capability`
- `lib.core.collections`
- `lib.core.crypto`
- `lib.core.filesystem`
- `lib.core.io`
- `lib.core.math`
- `lib.core.memory`
- `lib.core.networking`
- `lib.core.serialization`
- `lib.core.slices`
- `lib.core.strings`
- `lib.core.sync`
- `lib.core.testing`
- `lib.core.time`

## Promotion Notes

Wave 7 promotions in this repository include:

- `lib.experimental.io` -> `lib.core.io`
- `lib.experimental.collections` -> `lib.core.collections`
- `lib.experimental.filesystem` -> `lib.core.filesystem`
- `lib.experimental.networking` -> `lib.core.networking`
- `lib.experimental.async` -> `lib.core.async`
- `lib.experimental.sync` -> `lib.core.sync`
- `lib.experimental.serialization` -> `lib.core.serialization`
- `lib.experimental.slices` -> `lib.core.slices`
- `lib.experimental.strings` -> `lib.core.strings`
- `lib.experimental.testing` -> `lib.core.testing`
- `lib.experimental.time` -> `lib.core.time`
- `lib.experimental.crypto` -> `lib.core.crypto`

`lib.experimental.*` entries remain available as compatibility shims, but stable
code should import `lib.core.*` directly. Generated API docs label
`lib.experimental.*` modules as experimental so they are not confused with v1
compatibility promises.

## Stable Type Display

Public compiler output and generated API docs use canonical builtin names:

| Source aliases | Canonical name | Notes |
| --- | --- | --- |
| `Int`, `i32` | `i32` | Default integer literal type. |
| `UInt8`, `Byte`, `u8` | `u8` | Slice element supported by `[]u8` and string storage. |
| `UInt16`, `u16` | `u16` | Native-first slice element supported by `[]u16`. |
| `Bool`, `bool` | `bool` | Boolean literal and condition type. |
| `String`, `str` | `str` | Two-slot UTF-8 string/slice shape. |
| `ConsentToken` | `consent.token` | Privacy/consent capability token. |
| `SecretInt` | `secret.i32` | Privacy-protected integer wrapper. |

Structural types use deterministic field order and slot counts. `str`, `[]u8`,
`[]u16`, `[]i32`, and `[]bool` are two-slot values (`ptr`, `len`). `T?` adds one presence tag slot
to the payload slots. Opaque handles such as `ptr`, `island`, `actor`,
`cap.io`, `cap.mem`, and `task.*` are not interchangeable even when they occupy
one slot.

## Semantic Type Model Boundaries (v0.2 profile)

The current semantic checker intentionally enforces these boundaries:

- Arrays (`[N]T`) are not part of the checked type model yet.
- Slice element support is currently limited to `[]u8`, `[]u16`, `[]i32`, and `[]bool`.
- Local inference does not infer a type from bare `none`; optional payload type
  must be explicit (for example `let v: i32? = none`).
- Global type inference is limited to constant numeric/bool expressions used by
  immutable globals (`val`/`const`), and top-level `var` initializers are
  limited to explicit `i32`/`bool` constant expressions.

## Stable Module Quality Gates

Stable `lib.core.*` modules are required to include:

- top-of-file docs comments
- at least one `tetra doctest` block
- an `// Effects: ...` metadata line (`none` or a comma-separated list)
- a checked smoke example under `examples/core_*_smoke.tetra`

`tools/cmd/verify-docs` enforces these requirements.

Stable examples used as release evidence must import `lib.core.*` directly and
must not import `lib.experimental.*`.
