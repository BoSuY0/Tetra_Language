# Stdlib Module Naming And Versioning Policy

This document defines the normative naming and versioning rules for the stable
Tetra standard library surface used by release gates.

## Scope

- Stable stdlib modules: `lib.core.*`
- Experimental stdlib modules: `lib.experimental.*`
- Internal runtime/toolchain modules: names that start with `__`

## Naming Rules (Normative)

`STDLIB-NAME-001` Stable modules MUST declare names under `lib.core`.

- Allowed shape: `module lib.core.<name>`
- `<name>` MUST match `^[a-z][a-z0-9_]*$`.
- Version suffixes in stable module names are forbidden (for example:
  `lib.core.math_v2`, `lib.core.v2.math`).

`STDLIB-NAME-002` Experimental modules MUST declare names under
`lib.experimental`.

- Allowed shape: `module lib.experimental.<name>`
- `<name>` MUST match `^[a-z][a-z0-9_]*$`.

`STDLIB-NAME-003` Internal modules (names starting with `__`) MUST NOT be part
of the stable stdlib contract.

`STDLIB-NAME-004` File path and module declaration MUST match:

- `lib/core/<name>.tetra` -> `module lib.core.<name>`
- `lib/experimental/<name>.tetra` -> `module lib.experimental.<name>`

## Versioning Rules (Normative)

`STDLIB-VER-001` Stable stdlib modules (`lib.core.*`) are versioned by the
language release line, not by per-module semver.

- Example: `lib.core.math` in Tetra `v1.0.x` is the same stable module name
  across all `v1.x` releases.

`STDLIB-VER-002` In a major release line (for example `v1.x`), stable module
changes MUST be backward compatible:

- Allowed: additive APIs, bug fixes, clarifications.
- Not allowed: removals, incompatible signature/behavior changes.

`STDLIB-VER-003` Breaking changes to `lib.core.*` MUST wait for the next major
release line.

`STDLIB-VER-004` `lib.experimental.*` has no compatibility guarantee and MAY
change between minor/patch releases.

`STDLIB-VER-005` Promotion from experimental to stable MUST keep a stable name
without version suffixes and MUST be called out in release notes/checklists.

`STDLIB-VER-006` Generated API docs MUST label `lib.experimental.*` modules as
experimental. Experimental docs entries are discoverability aids, not stable
API compatibility promises.

## Release Gate Checks

Release gating MUST fail if any rule above is violated.

Recommended checks:

1. Validate module declarations and path matching for all `lib/core/*.tetra`
   and `lib/experimental/*.tetra`.
2. Reject stable module names that contain explicit version segments/suffixes.
3. Confirm stable stdlib docs/doctests/examples/metadata checks pass through the
   existing docs + API-doc validation pipeline.
4. Reject release-evidence examples for stable modules when they import
   `lib.experimental.*`.

Example command snippets for gate scripts:

```sh
rg -n '^module lib\.core\.[a-z][a-z0-9_]*$' lib/core/*.tetra
rg -n '^module lib\.experimental\.[a-z][a-z0-9_]*$' lib/experimental/*.tetra
! rg -n '^module lib\.core\..*(v[0-9]+|_[vV][0-9]+)' lib/core/*.tetra
```

Generated API docs must be produced from the same branch state before this
policy is used as release evidence:

```sh
./tetra doc examples > REPORT_DIR/artifacts/tetra-docs.md
go run ./tools/cmd/validate-api-docs --docs REPORT_DIR/artifacts/tetra-docs.md
```
