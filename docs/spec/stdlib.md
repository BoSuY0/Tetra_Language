# Standard Library Spec Notes

This page anchors stdlib-specific spec policies.

- Naming and versioning policy (normative for release gating):
  [stdlib_naming_versioning.md](./stdlib_naming_versioning.md)

## Stable Core Surface (`v0.4.0` Current Profile)

The current `v0.4.0` stable stdlib modules under `lib/core` are:

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

## Stable Smoke Evidence and Profile Semantics

Every stable `lib.core.*` module has a checked stable-core smoke example under
`examples/core_*_smoke.tetra`. In this spec, "checked" means the example exists,
imports the stable `lib.core.*` module directly, is covered by the stdlib
documentation/completeness workflow, and is accepted by `tools/cmd/verify-docs`.
It does not mean that every stable-core smoke is an active case in the default
`./tetra smoke --list --format=json` linux-x64 profile.

The default linux-x64 smoke profile is a narrower runtime smoke list. It
currently includes `examples/core_math_smoke.tetra` and
`examples/core_memory_smoke.tetra` as active stdlib smoke cases. Other
stable-core smoke examples may appear in the smoke-list report's
`excluded_examples` array with the reason `not part of linux-x64 smoke profile`.
That exclusion is profile scope, not a withdrawal of stable stdlib evidence.

Stable stdlib release evidence therefore comes from the separate stdlib
completeness workflow documented in
`docs/user/standard_library_guide.md#verification`: generated API docs for
`lib/core`, `lib/experimental`, and all `examples/core_*_smoke.tetra` files,
followed by `go run ./tools/cmd/verify-docs --manifest
docs/generated/manifest.json`. The smoke-list contract is validated separately
with `validate-smoke-list`, which requires examples to be either active smoke
cases or explicit `excluded_examples`.

## Stable Core Function Matrix

This matrix is the function-level `lib.core.*` surface for the current
`v0.4.0` profile. `Effects` records the function's declared `uses` clause;
`none` means the function has no per-function `uses` clause. `Stability` means
stable within the current release profile and should not be read as a broader
runtime, host, or security guarantee.

| Module | Function signature | Effects | Stability | Contract notes |
| --- | --- | --- | --- | --- |
| `lib.core.async` | `async func ready(value: Int) -> Int` | none | stable `v0.3.0` core | Async helper surface; no scheduler/runtime progress guarantee is implied. |
| `lib.core.async` | `async func pair_sum(lhs: Int, rhs: Int) -> Int` | none | stable `v0.3.0` core | Async helper surface; awaits local helper calls only. |
| `lib.core.async` | `func select_or(value: Int, fallback: Int) -> Int` | none | stable `v0.3.0` core | Pure fallback helper. |
| `lib.core.capability` | `func mem() -> cap.mem` | `capability, mem` | stable `v0.3.0` core | Unsafe capability wrapper; callers still need matching effects. |
| `lib.core.capability` | `func io() -> cap.io` | `capability, io` | stable `v0.3.0` core | Unsafe capability wrapper; callers still need matching effects. |
| `lib.core.collections` | `func len_i32(values: []i32) -> Int` | `mem` | stable `v0.3.0` core | `[]i32` scan helper only; not a generic collection API. |
| `lib.core.collections` | `func contains_i32(values: []i32, needle: Int) -> Bool` | `mem` | stable `v0.3.0` core | `[]i32` scan helper only; not a generic collection API. |
| `lib.core.collections` | `func count_i32(values: []i32, needle: Int) -> Int` | `mem` | stable `v0.3.0` core | `[]i32` scan helper only; not a generic collection API. |
| `lib.core.collections` | `func first_or_i32(values: []i32, fallback: Int) -> Int` | `mem` | stable `v0.3.0` core | `[]i32` scan helper only; returns fallback for an empty slice. |
| `lib.core.crypto` | `func interface_strength() -> Int` | none | stable `v0.3.0` core | Stable interface marker for examples and API docs. |
| `lib.core.crypto` | `func mix_seed(seed: Int, value: Int) -> Int` | none | stable `v0.3.0` core | Deterministic mixer for reproducible interface tests; no encryption or authentication claim. |
| `lib.core.crypto` | `func checksum_u8(values: []u8) -> Int` | `mem` | stable `v0.3.0` core | Deterministic byte checksum for examples and API-shape tests. |
| `lib.core.crypto` | `func constant_time_eq_u8(lhs: []u8, rhs: []u8) -> Bool` | `mem` | stable `v0.3.0` core | Equality helper for byte slices; scans equal-length inputs without early value mismatch exit. |
| `lib.core.filesystem` | `func exists(path: String, io_cap: cap.io) -> Bool` | `io` | stable `v0.4.0` linux-x64 slice | Host-backed existence check through `__tetra_fs_exists`; requires an explicit `cap.io` token and returns false for missing, invalid, or unsupported paths. |
| `lib.core.filesystem` | `func has_leading_slash(path: String) -> Bool` | none | stable `v0.3.0` core | Pure string-path utility; no host access. |
| `lib.core.filesystem` | `func ends_with_slash(path: String) -> Bool` | none | stable `v0.3.0` core | Pure string-path utility; no host access. |
| `lib.core.filesystem` | `func is_root(path: String) -> Bool` | none | stable `v0.3.0` core | Pure string-path utility; treats `/` as root. |
| `lib.core.filesystem` | `func slash_count(path: String) -> Int` | none | stable `v0.3.0` core | Pure string-path utility; counts slash bytes. |
| `lib.core.filesystem` | `func directory_depth(path: String) -> Int` | none | stable `v0.3.0` core | Pure string-path utility; counts non-empty path segments. |
| `lib.core.io` | `func capability_io() -> cap.io` | `capability, io` | stable `v0.3.0` core | Unsafe capability wrapper; no host permission is granted by import alone. |
| `lib.core.io` | `func mmio_read_i32(addr: ptr, io_cap: cap.io) -> Int` | `io, mmio` | stable `v0.3.0` core | Unsafe MMIO wrapper over caller-selected address and token. |
| `lib.core.io` | `func mmio_write_i32(addr: ptr, value: Int, io_cap: cap.io) -> Int` | `io, mmio` | stable `v0.3.0` core | Unsafe MMIO wrapper over caller-selected address and token. |
| `lib.core.math` | `func add_i32(a: Int, b: Int) -> Int` | none | stable `v0.3.0` core | Pure integer addition helper. |
| `lib.core.math` | `func min_i32(a: Int, b: Int) -> Int` | none | stable `v0.3.0` core | Pure integer minimum helper. |
| `lib.core.math` | `func max_i32(a: Int, b: Int) -> Int` | none | stable `v0.3.0` core | Pure integer maximum helper. |
| `lib.core.math` | `func clamp_i32(value: Int, lo: Int, hi: Int) -> Int` | none | stable `v0.3.0` core | Pure integer helper; assumes caller chooses sensible bounds. |
| `lib.core.memory` | `func memset_u8(dst: ptr, v: UInt8, n: Int, mem: cap.mem) -> Int` | `mem` | stable `v0.3.0` core | Unsafe byte helper; no allocation, bounds validation, or permission grant. |
| `lib.core.memory` | `func memcpy_u8(dst: ptr, src: ptr, n: Int, mem: cap.mem) -> Int` | `mem` | stable `v0.3.0` core | Unsafe byte helper; caller owns pointer validity and overlap assumptions. |
| `lib.core.networking` | `func default_port_http() -> Int` | none | stable `v0.3.0` core | Standard HTTP port constant. |
| `lib.core.networking` | `func default_port_https() -> Int` | none | stable `v0.3.0` core | Standard HTTPS port constant. |
| `lib.core.networking` | `func clamp_port(port: Int) -> Int` | none | stable `v0.3.0` core | Deterministic port-range policy helper. |
| `lib.core.networking` | `func is_valid_port(port: Int) -> Bool` | none | stable `v0.3.0` core | Port-range validation helper for endpoint configuration. |
| `lib.core.networking` | `func choose_port(preferred: Int, fallback: Int) -> Int` | none | stable `v0.3.0` core | Endpoint configuration helper; `0` preferred port falls back. |
| `lib.core.networking` | `func retry_backoff_ms(attempt: Int, base_ms: Int, max_ms: Int) -> Int` | none | stable `v0.3.0` core | Deterministic retry backoff policy helper. |
| `lib.core.serialization` | `func clamp_u8(value: Int) -> Int` | none | stable `v0.3.0` core | Pure packing helper; not a general serializer. |
| `lib.core.serialization` | `func pack_u8_pair(high: Int, low: Int) -> Int` | none | stable `v0.3.0` core | Pure two-byte packing helper; not a wire-format guarantee. |
| `lib.core.serialization` | `func unpack_u8_high(packed: Int) -> Int` | none | stable `v0.3.0` core | Pure unpack helper; negative packed input returns `0`. |
| `lib.core.serialization` | `func unpack_u8_low(packed: Int) -> Int` | none | stable `v0.3.0` core | Pure unpack helper; negative packed input returns `0`. |
| `lib.core.serialization` | `func checksum_u8(values: []u8) -> Int` | `mem` | stable `v0.3.0` core | Simple checksum helper; not authentication or encryption. |
| `lib.core.slices` | `func sum_i32(values: []i32) -> Int` | `mem` | stable `v0.3.0` core | Slice scan helper for supported element type only. |
| `lib.core.slices` | `func weighted_sum_i32(values: []i32) -> Int` | `mem` | stable `v0.3.0` core | Slice scan helper for supported element type only. |
| `lib.core.slices` | `func sum_u8(values: []u8) -> Int` | `mem` | stable `v0.3.0` core | Slice scan helper for supported element type only. |
| `lib.core.strings` | `func ascii_len(text: String) -> Int` | none | stable `v0.3.0` core | ASCII/byte-oriented helper; no Unicode normalization guarantee. |
| `lib.core.strings` | `func ascii_sum(text: String) -> Int` | none | stable `v0.3.0` core | ASCII/byte-oriented helper; no Unicode normalization guarantee. |
| `lib.core.strings` | `func is_empty(text: String) -> Bool` | none | stable `v0.3.0` core | ASCII/byte-oriented helper built from `ascii_len`. |
| `lib.core.sync` | `func merge_status(lhs: Int, rhs: Int) -> Int` | none | stable `v0.3.0` core | Pure status helper; not a runtime synchronization primitive. |
| `lib.core.sync` | `func all_ready(lhs: Bool, rhs: Bool) -> Bool` | none | stable `v0.3.0` core | Pure boolean helper; not a runtime synchronization primitive. |
| `lib.core.sync` | `func spin_countdown(start: Int, ticks: Int) -> Int` | none | stable `v0.3.0` core | Pure countdown helper; no sleeping or scheduling behavior. |
| `lib.core.sync` | `func barrier_target(workers: Int) -> Int` | none | stable `v0.3.0` core | Pure clamp helper; not a runtime barrier. |
| `lib.core.testing` | `func assert_true(value: Bool) -> Int` | none | stable `v0.3.0` core | Status-code helper; `0` means pass and `1` means fail. |
| `lib.core.testing` | `func assert_false(value: Bool) -> Int` | none | stable `v0.3.0` core | Status-code helper; `0` means pass and `1` means fail. |
| `lib.core.testing` | `func assert_eq_i32(actual: Int, expected: Int) -> Int` | none | stable `v0.3.0` core | Status-code helper; `0` means pass and `1` means fail. |
| `lib.core.testing` | `func combine(lhs: Int, rhs: Int) -> Int` | none | stable `v0.3.0` core | Status-code helper; returns first non-zero status. |
| `lib.core.time` | `func millis_from_seconds(seconds: Int) -> Int` | none | stable `v0.3.0` core | Pure duration arithmetic; negative input clamps to `0`. |
| `lib.core.time` | `func seconds_from_millis(milliseconds: Int) -> Int` | none | stable `v0.3.0` core | Pure duration arithmetic; negative input clamps to `0`. |
| `lib.core.time` | `func clamp_timeout_ms(value: Int, lo: Int, hi: Int) -> Int` | none | stable `v0.3.0` core | Pure duration arithmetic; assumes caller chooses sensible bounds. |
| `lib.core.time` | `func add_duration_ms(base: Int, delta: Int) -> Int` | none | stable `v0.3.0` core | Pure duration arithmetic; negative result clamps to `0`. |

## `lib.core.math`

`lib.core.math` is a stable, side-effect-free integer helper module covered by
`examples/core_math_smoke.tetra`. The smoke imports `lib.core.math` directly and
exercises `add_i32`, `min_i32`, `max_i32`, and `clamp_i32` while preserving the
stable exit value `42`.

Function behavior:

- `add_i32(a, b)` returns `a + b`.
- `min_i32(a, b)` returns the smaller argument, returning `b` when `a` is not
  less than `b`.
- `max_i32(a, b)` returns the larger argument, returning `b` when `a` is not
  greater than `b`.
- `clamp_i32(value, lo, hi)` returns `lo` when `value < lo`, `hi` when
  `value > hi`, and otherwise `value`.

## `lib.core.memory`

`lib.core.memory` is a stable capability-bound byte helper module covered by
`examples/core_memory_smoke.tetra`. The smoke imports `lib.core.memory` directly
and exercises both `memset_u8` and `memcpy_u8` through an explicit `cap.mem`
token while preserving the stable exit value `42`.

Calls require a `uses mem` effect and a caller-provided `cap.mem` token. These
helpers operate on caller-selected memory regions; they do not allocate memory,
validate pointers, add bounds checks, or grant host permissions.

### `lib.core.memory` Production Boundary

The Memory Production Core treats this module as a thin safe-wrapper surface
around reviewed unsafe byte loops. `memcpy_u8` and `memset_u8` do not allocate,
do not perform bounds checks, do not track ownership, and do not prove that two
regions are non-overlapping. A production memory report must therefore pair
uses of this module with separate allocator, ownership, and runtime bounds
diagnostics evidence.

For verifier stability: this module does not allocate and does not perform bounds checks.

Function behavior:

- `memset_u8(dst, v, n, mem)` writes the `UInt8` value `v` to `n` bytes starting
  at `dst` and returns `0`. Passing `0` is the documented clear/fill pattern for
  a valid writable region.
- `memcpy_u8(dst, src, n, mem)` copies `n` bytes from `src` to `dst` in
  increasing byte order and returns `0`. The caller remains responsible for
  choosing valid readable and writable regions and for any overlap assumptions.

## Generated Docs Rendering Policy

Generated docs render a source unit by declared module path when the file has a
`module ...` declaration. For example, `lib/core/math.tetra` renders as
`lib.core.math`, and `examples/core_math_smoke.tetra` renders as
`examples.core_math_smoke` because that smoke declares that module name.

Generated docs render a portable repository file path when a source file has no
module declaration. Example-only programs such as `examples/flow_hello.tetra`
therefore keep the `examples/...` spelling in generated docs. This mixed
rendering is intentional: dotted names are module identities, while slash paths
are file identities.

## Promotion Notes

For the `v0.4.0` release line, each current `lib/experimental/*.tetra` file is
a production compatibility mirror that imports the matching `lib.core.*` module
and forwards to it. Stable code should import `lib.core.*` directly; the
`lib.experimental.*` namespace is retained only for legacy source
compatibility.

| Experimental mirror | Stable target | `v0.4.0` status |
| --- | --- | --- |
| `lib.experimental.async` | `lib.core.async` | Compatibility mirror; source note directs stable callers to `lib.core.async`. |
| `lib.experimental.collections` | `lib.core.collections` | Compatibility mirror; source note directs stable callers to `lib.core.collections`. |
| `lib.experimental.crypto` | `lib.core.crypto` | Compatibility mirror for stable crypto interface helpers. |
| `lib.experimental.filesystem` | `lib.core.filesystem` | Compatibility mirror for filesystem helpers, including the linux-x64 host-backed `exists` slice. |
| `lib.experimental.io` | `lib.core.io` | Compatibility mirror for capability/MMIO wrappers; callers still need matching effects. |
| `lib.experimental.math` | `lib.core.math` | Compatibility mirror; explicitly present in `lib/experimental/math.tetra`. |
| `lib.experimental.memory` | `lib.core.memory` | Compatibility mirror; explicitly present in `lib/experimental/memory.tetra` and still requires `cap.mem`. |
| `lib.experimental.networking` | `lib.core.networking` | Compatibility mirror for stable endpoint policy helpers. |
| `lib.experimental.serialization` | `lib.core.serialization` | Compatibility mirror for packing/checksum helpers; not a general serializer guarantee. |
| `lib.experimental.slices` | `lib.core.slices` | Compatibility mirror; source note directs stable callers to `lib.core.slices`. |
| `lib.experimental.strings` | `lib.core.strings` | Compatibility mirror; source note directs stable callers to `lib.core.strings`. |
| `lib.experimental.sync` | `lib.core.sync` | Compatibility mirror for pure status/countdown helpers; not runtime synchronization. |
| `lib.experimental.testing` | `lib.core.testing` | Compatibility mirror; source note directs stable callers to `lib.core.testing`. |
| `lib.experimental.time` | `lib.core.time` | Compatibility mirror for pure duration arithmetic; no clock or scheduler access. |

Generated API docs label `lib.experimental.*` modules as experimental so they
are not confused with the stable `lib.core.*` API namespace. The production
claim is limited to mirror forwarding compatibility, not new stable APIs under
`lib.experimental.*`.

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

## Semantic Type Model Boundaries (v0.3 profile)

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
- a checked stable-core smoke example under `examples/core_*_smoke.tetra`

`tools/cmd/verify-docs` enforces these requirements. Default smoke-list
membership is tracked separately by `./tetra smoke --list --format=json`; stable
core smoke examples that are not active linux-x64 smoke cases must remain listed
as `excluded_examples` there.

Stable examples used as release evidence must import `lib.core.*` directly and
must not import `lib.experimental.*`.
