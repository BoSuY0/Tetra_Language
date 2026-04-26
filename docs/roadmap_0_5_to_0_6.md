# Roadmap v0.5 → v0.6 (Usable Alpha)

Status: completed as the v0.6.0 hardening profile.

v0.6 turns the broad v0.5 Integrated Alpha surface into a more usable local
alpha. The release focuses on reliability and day-to-day tooling rather than
large new language syntax.

## Completed Focus

- Version and manifest identity move to `v0.6.0`.
- Formatter coverage expands from a single smoke file to all `examples` and
  `lib` sources.
- LSP-basic grows from `--stdio-smoke` into a minimal stdio JSON-RPC loop with
  initialize, shutdown, didOpen diagnostics, document symbols, and hover.
- Eco pack gains project bundle mode while preserving single-manifest packs.
- Release gating is captured in `scripts/release_v0_6_gate.sh`.

## Still Deferred

Payload enums, exhaustive match, collection iteration, closures, full ownership
and lifetime solving, full structured concurrency, protocol-bound generics,
production-grade LSP, UI DSL/backends, package publishing, proof-carrying
capsules, EcoNet, distributed Todex mesh, trust scoring, and v1 stability
guarantees remain post-v0.6 work.
