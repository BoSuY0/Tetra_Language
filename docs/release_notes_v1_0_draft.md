# Tetra v1.0 Production Release Notes Draft

This draft tracks the target shape for Tetra v1.0. It is not a claim that the
current v0.6 compiler already implements the 1.0 surface.

## Target Profile

- Flow-only syntax.
- Stable ownership/lifetime safety and no data races in safe code.
- Stable type system with payload enums, exhaustive pattern matching,
  optionals, typed errors, generics, protocols, extensions, and modules.
- Stable effects, capabilities, privacy clauses, and resource budget clauses.
- Native x64 and WASM targets.
- Stable formatter, test runner, JSON diagnostics, LSP, docs generator, and API
  diff tooling.
- Stable stdlib for core systems, IO, networking, async/sync, serialization,
  time, testing, and crypto interfaces.
- Stable UI model with web and native shell backends.
- Stable local Eco/Todex plus beta network publishing.

## Compatibility

Legacy brace syntax is a migration-only concern before 1.0 and is not part of
the 1.0 language profile. 1.0 source examples, docs, formatter output, and
release smoke tests use Flow syntax only. The release gate tracks this with
`go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`.

## Deferred Beyond 1.0

Full distributed Todex mesh, proof-carrying capsules, global EcoTrust scoring,
EcoOracle, time-travel/live evolution, distributed actors, AI model types, and
the multiverse optimizer remain post-1.0 work unless separately promoted and
stabilized before release candidate freeze.
