# Project Structure

This repository is moving from flat, oversized package directories toward
domain-owned directories with small migration slices.

## Compiler

```text
compiler/
  internal/
    testkit/
  tests/
    backend/
    callables/
    frontend/
    lowering/
    semantics/
  testdata/
    callables/
      captures/
      cross_module/
      mutable_global/
      throwing/
```

`compiler/internal/testkit` is the bridge that lets domain tests move out of the
flat `compiler/` package without duplicating build/run helpers. Move helpers
first, then move test groups.

## CLI

```text
cli/
  testkit/
  tests/
    commands/
    eco/
    lsp/
    smoke/
```

`cli/testkit` owns shared command execution, fixture project setup, stdout/stderr
assertions, and JSON helpers. Domain directories own command-specific behavior.

## Tools And Release

```text
tools/
  validators/
  release/
    gates/
    security/
    smoke/
  testkit/
  scriptstest/
    fixtures/
    v0_3_0/
    v1_0_0/
```

Validator helper extraction should happen only after at least three validators
prove a shared helper boundary. Release script tests should separate fake repos,
evidence fixtures, and assertions.

## Scripts

```text
scripts/
  ci/
  dev/
  release/
    shared/
    smoke/
    v0_1_1/
    v0_1_2/
    v0_1_3/
    v0_2_0/
    v0_3_0/
    v0_4_0/
    v0_5/
    v0_6/
    v1_0/
  tools/
```

Root-level scripts remain compatibility entrypoints until docs, CI, and release
gates migrate to the directory-based names.

## Docs And Examples

```text
docs/
  architecture/
  audits/
  generated/
  plans/
  release/

examples/
  projects/
  smoke/
```

Architecture and audits are separated from execution plans and release-specific
evidence. Smoke examples are separated from larger dogfood projects.

## Migration Rules

- Do not move package-local Go tests into subdirectories until shared helpers
  are available in the relevant `testkit`.
- Keep compatibility wrappers for renamed scripts until every reference is
  migrated.
- Each migration slice must have a RED guard, focused test, relevant package
  test, hygiene check, and Graphify update when code changes.
- Generated artifacts and historical release evidence must not churn unless the
  generator or validator slice requires it.

