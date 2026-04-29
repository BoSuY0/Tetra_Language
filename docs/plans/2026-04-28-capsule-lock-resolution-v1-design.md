# Capsule Lock Resolution v1 Design

**Goal:** make `Capsule.t4`, `Tetra.lock`, `.t4i`, `.tobj`, and `.t4s`
work as one local project system.

## Observed Facts

- `Capsule.t4` is already preferred over legacy `Tetra.capsule`.
- CLI project discovery already resolves entry files, source roots, targets, and
  local path dependencies.
- `.t4i` files are already loadable as interface-only modules.
- Native builds already accept repeatable `--link-object` and validate matching
  `.tobj` providers for `.t4i` modules.
- Eco lock files already carry capsule graph metadata and a graph hash.

## Design

`Capsule.t4` gains an `artifacts:` section:

```tetra
artifacts:
    interface interfaces/math/core.t4i
    object artifacts/math-core.tobj
    seed seeds/tetra-core.t4s
```

Artifact paths are project-relative, stay inside the capsule root, and use
forward slashes. `interface` artifacts are `.t4i` files; project discovery reads
their module declaration and adds the containing interface root to module
resolution. `object` artifacts are `.tobj` files; `build` and `run` append them
to the existing `--link-object` list. `seed` artifacts are lock-tracked package
inputs for offline workflows; they are validated by hash but not linked.

`eco verify --lock` expands a single project capsule with path dependencies into
the full local capsule graph before writing `Tetra.lock`.

When a discovered project root contains `Tetra.lock`, `check`, `build`, and
`run` validate the current capsule graph and artifact hashes against the lock
before compiler work begins. Missing locks remain allowed for early local work;
present locks are authoritative.

## Non-Goals

- No network resolver or semver range solving.
- No `.t4s` materialization during build.
- No global package cache.

## Verification

- RED/GREEN CLI tests for project lock validation, path dependency lock
  expansion, and artifact-driven interface/object linking.
- Focused `go test ./cli/cmd/tetra`.
- Broad `go test ./compiler/... ./cli/... ./tools/...`.
- Docs validators and final `go test ./...`.
