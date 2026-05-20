# Compiler Pipeline Map

Status: contributor guide for navigating the current compiler/tooling layout.

This document is descriptive, not a language specification. Public behavior is
defined by the specs under `docs/spec/` and by release gates.

## Pipeline

| Stage | Main paths | Notes |
| --- | --- | --- |
| CLI dispatch | `cli/cmd/tetra/main.go`, `project.go`, `workspace.go`, `eco.go` | Parses command flags, discovers projects, and calls compiler/tooling APIs. |
| Source formats | `compiler/internal/formats`, `docs/spec/t4_formats.md` | Defines `.t4`, legacy `.tetra`, `.t4i`, `.tobj`, and related artifact roles. |
| Frontend | `compiler/internal/frontend` | Lexes, parses, normalizes Flow migration paths, and emits frontend diagnostics. |
| Module graph | `compiler/internal/module/loader.go` | Loads entry files, imports, source roots, dependency roots, and `.t4i` interfaces. |
| Semantics | `compiler/internal/semantics` | Checks types, effects, capabilities, ownership/resource state, generics, protocols, actors, tasks, and UI metadata. |
| Lowering | `compiler/internal/lower` | Converts checked programs to target-neutral IR and UI bundles. |
| IR verification | `compiler/internal/lower/verify.go`, `compiler/internal/ir/ir.go` | Rejects invalid stack/slot/control-flow IR before codegen. |
| Backends | `compiler/internal/backend`, `compiler/internal/format`, `compiler/internal/linker` | Emits native objects/executables and build-only WASM artifacts. |
| Validators | `tools/cmd/*` | Validate release reports, docs, manifests, smoke lists, diagnostics, and artifacts. |

## Public API Boundary

`compiler/api.go` is the public hourglass for parser, module loading, semantic
checking, lowering, IR verification, codegen, linking, and object read/write
helpers. Prefer extending this boundary deliberately instead of importing
internal packages from new external callers.

## Hotspot Guidance

- Keep behavior-changing work in small focused slices with tests first.
- Large same-package splits are acceptable when they are mechanical and verified
  by focused package tests.
- Avoid changing IR emission order unless the corresponding verifier, backend,
  and smoke tests are updated together.
- Do not promote planned features by editing docs alone; update implementation,
  tests, feature registry, manifest, and release evidence in the same branch
  state.

## Useful Verification Commands

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
bash scripts/ci/test.sh
bash scripts/ci/test-all.sh --quick
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```
