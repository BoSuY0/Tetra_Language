# Tetra v1.0 Production Release Notes Draft

This draft tracks the release-note shape for Tetra v1.0. The current public release line remains
`v0.4.0`; the authoritative future v1.0 scope contract is `docs/spec/flow/v1_scope.md`.

Canonical scope and release process:

- `docs/spec/flow/v1_scope.md`
- `docs/checklists/v1_0_release_gate.md`
- `docs/release/policy/artifact_policy.md`
- `docs/release/policy/rc_process.md`

## Target Profile

- Flow-only syntax.
- Stable ownership/lifetime safety and no data races in safe code.
- Stable type system with payload enums, exhaustive pattern matching, optionals, typed errors,
  generics, protocols, extensions, and modules.
- Stable effects, capabilities, privacy clauses, and resource budget clauses.
- Native x64 and WASM targets.
- Stable formatter, test runner, JSON diagnostics, LSP, docs generator, and API diff tooling.
- Stable stdlib for core systems, IO, networking, async/sync, serialization, time, testing, and
  crypto interfaces.
- Stable UI model with web and native shell backends.
- Linux-x64 v1 native actor platform target remains gated by the final native actor platform
  validator. Current actor runtime foundation evidence is scoped and must stay behind the strict
  Linux-x64 gate until every required v1 capability report exists.
- Stable local Eco/Todex plus beta network publishing.

## Current Actor Runtime Evidence

The current actor runtime foundation evidence is bounded by the strict Linux-x64 gate and the
report set named by the actor capability manifest:

- `actor-runtime-foundation-manifest.json`
- `distributed-actors-linux-x64/distributed-actors-linux-x64.json`
- `parallel-production-linux-x64/parallel-production-linux-x64.json`

The V1-P01 system-message lane claim is scoped: source-level system-message API and isolated
runtime system lane implemented for Linux-x64 builtin runtime.

Required actor foundation nonclaims:

- no full Erlang/OTP actor runtime claim
- no cluster membership or reconnect/retry production claim
- no non-Linux distributed actor runtime support claim
- no distributed zero-copy pointer or region transfer claim
- no formal race proof claim

## Compatibility

Legacy brace syntax is a migration-only concern before 1.0 and is not part of the 1.0 language
profile. 1.0 source examples, docs, formatter output, and release smoke tests use Flow syntax only.
The release gate tracks this with
`go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`.

## Deferred Beyond 1.0

Full distributed Todex mesh, proof-carrying capsules, global EcoTrust scoring, EcoOracle,
time-travel/live evolution, exactly-once actor messaging, highly available consensus control plane
behavior, AI model types, and the multiverse optimizer remain post-1.0 work unless separately
promoted and stabilized before release candidate freeze.
