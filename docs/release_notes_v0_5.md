# Tetra v0.5.0 Integrated Alpha Release Notes

Tetra v0.5.0 is an Integrated Alpha release. It packages the staged compiler,
runtime, language, tooling, docs, and local ecosystem work into one coherent
local development profile. It is not the final Tetra platform or a v1.0
compatibility promise.

## Supported profile

- CLI commands: `version`, `build`, `run`, `smoke`, `fmt`, `test`, `clean`,
  `eco`, and `lsp`.
- Targets: `linux-x64`, `macos-x64`, and `windows-x64` build output. Native
  execution is supported when host and target match.
- Syntax: legacy brace syntax plus Flow indentation syntax for functions,
  structs, enums, blocks, `uses`, `unsafe`, and scoped islands.
- Core language: `bool`, `true`/`false`, range `for`, no-payload enums,
  statement `match`, optionals MVP, typed errors MVP, simple generics MVP,
  protocols MVP, extensions MVP, ownership markers, and async/task MVP.
- Effects: checked `uses` declarations for `io`, `mem`, `alloc`,
  `capability`, `islands`, `mmio`, `link`, `control`, `runtime`, and `actors`.
- Runtime/toolchain: self-host actor runtime selection, builtin fallback,
  object/library emission, repeatable TOBJ linking, build cache, and x64 output
  format checks.
- Stdlib: stable `lib.core.math`, `lib.core.capability`, and `lib.core.memory`
  helpers, with lower-level experimental modules retained under
  `lib.experimental`.
- Tooling: formatter, test runner, JSON diagnostics, doctest verification,
  generated API docs, and LSP-basic diagnostics/symbols/hover.
- Local ecosystem: capsule verify/pack/unpack, dependency graph lock/provenance
  JSON, and a local content-addressed Todex vault prototype.

## Deferred beyond v0.5

Payload enums, exhaustive match, collection iteration, closures, full ownership
and lifetime solving, full structured concurrency, protocol-bound generics,
production LSP, UI DSL/backends, package publishing, proof-carrying capsules,
EcoNet, distributed Todex mesh, trust scoring, and v1 stability guarantees are
post-v0.5 work.

## Release gate

The canonical release verification command is:

```bash
bash scripts/release_v0_5_gate.sh
```
