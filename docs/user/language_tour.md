# Tetra Language Tour

Status: user-facing tour for the v1.0 scope. This guide describes the intended
stable profile and calls out planned or blocked areas instead of implying they
are complete.

## Source Shape

Tetra v1.0 uses Flow indentation syntax as the canonical source style. Release
sources are checked with:

```sh
go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
```

## Modules And Functions

Source files define modules, imports, functions, declarations, and tests. Use
`./tetra check <file>` for fast feedback and `./tetra fmt --check <paths>` to
verify formatting without rewriting files.

## Types

The v1.0 scope requires stable behavior for primitive values, structs, slices,
strings, optionals, typed errors, enums, generics, protocols, extensions, and
modules. The exact release contract is tracked in `docs/spec/v1_scope.md` and
the detailed syntax rules live in `docs/spec/flow_syntax_v1.md`.

## Control Flow

The supported Flow surface includes ordinary blocks, conditionals, loops, test
blocks, and release-covered match forms. Any syntax that still emits a
planned-feature diagnostic is not part of the final v1.0 support claim until
the release gate has evidence for it.

## Diagnostics

For humans, use the default CLI output. For tools, use JSON diagnostics where a
command supports them and validate reports through the matching validator under
`tools/cmd/`.
