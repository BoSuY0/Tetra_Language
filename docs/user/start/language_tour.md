# Tetra Language Tour

Status: user-facing tour for the current `v0.4.0` profile with future v1.0 notes. This guide calls
out planned or blocked areas instead of implying they are complete.

## Source Shape

The current profile uses Flow indentation syntax for release-covered sources. The full v1.0 language
contract remains future scope. Release sources are checked with:

```sh
go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
```

## Modules And Functions

Source files define modules, imports, functions, declarations, and tests. Use `./tetra check <file>`
for fast feedback and `./tetra fmt --check <paths>` to verify formatting without rewriting files.

Module imports support namespace aliases and selective public imports:

```tetra
import lib.core.math as math
import app.ui.{CounterView, render}
```

Use `pub` to define the public surface of a module. A `pub import` re-exports selected public names
through the current module.

## Types

The current profile supports the subset described in `docs/spec/core/current_supported_surface.md`.
It includes static monomorphized generic functions, static protocol conformance, static
protocol-bound generic validation, and positional enum payload match/catch/if-let support. Full
generic struct support, dynamic protocol dispatch, full first-class callable semantics, and full
v1.0 guarantees remain outside the current profile.

## Control Flow

The supported Flow surface includes ordinary blocks, conditionals, loops, test blocks, and
release-covered match forms. Any syntax that still emits a planned-feature diagnostic is not part of
the final v1.0 support claim until the release gate has evidence for it.

## Callable And Preview Boundaries

Function types, callable Level 1/Level 2, and selected safe first-class callable behavior are
current only within the bounded scope described by `docs/spec/core/current_supported_surface.md`,
`docs/generated/manifest.json`, and `compiler/compiler_facade.go`. Anything outside that bounded
scope remains future.

For a compact map of current and excluded `v0.4.0` behavior, see
`docs/spec/core/current_supported_surface.md` and `docs/release-notes/v0_4_0.md`.

## Diagnostics

For humans, use the default CLI output. For tools, use JSON diagnostics where a command supports
them and validate reports through the matching validator under `tools/cmd/`.
