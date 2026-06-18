# Roadmap v0.18 → v0.5 (Integrated Alpha)

> Historical checkpoint. This roadmap describes the completed v0.5 profile and is superseded by
> `docs/spec/flow/v1_scope.md` and `docs/checklists/v1_0_release_gate.md`. Public release truth for
> this branch lives in `docs/spec/core/current_supported_surface.md` (`v0.2.0`). The v1.0 scope
> remains a future contract.

Status: completed as the v0.5.0 Integrated Alpha profile.

Focus: the staged v0.14-v0.18 compiler/toolchain work is now one coherent Integrated Alpha profile
with the MVP slices needed before the v0.7 beta track. v0.5 is not the full future Tetra language,
UI stack, package ecosystem, or distributed runtime.

## Profile Definition

The v0.5 profile includes the v0.18 baseline surface:

- native builds for `linux-x64`, `windows-x64`, and `macos-x64`;
- legacy brace syntax plus the accepted Flow/Core profile syntax;
- checked `uses` effects for the current observable effect names;
- Islands and unsafe/capability boundaries documented as the safety baseline;
- actor runtime selection through `--runtime=auto|selfhost|builtin`;
- TOBJ library emission and repeatable object linking;
- developer tooling alpha commands: `fmt`, `test`, structured diagnostics, and docs doctests;
- local Eco/Capsule verification, Todex pack/unpack, and a local content-addressed vault prototype
  as experimental tooling, not a published ecosystem contract.

v0.5 adds:

- optionals MVP (`T?`, `none`, implicit one-slot `some`, `if let`) as the first accepted v0.5
  language slice;
- typed errors MVP (`throws`, `throw`, `try`) for one-slot success/error functions, with
  non-throwing `main`;
- ownership markers MVP (`borrow`, `inout`, `consume`) with local diagnostics;
- LSP-basic JSON analysis and generated API docs;
- pattern-match expansion;
- same-module generic function monomorphization MVP;
- protocols MVP as typed requirement declarations plus `impl Type: Protocol` conformance checks;
- extensions MVP as namespaced static functions;
- ownership markers plus local borrow-checker rules;
- effects v2 and a stable `lib/core` surface for math, capabilities, and memory;
- async syntax MVP plus a cooperative single-slot task runtime MVP;
- dependency-aware local Capsule/Todex graphing and local vault records;
- LSP basics, API docs generation, and formatter/LSP compatibility;
- release hardening across the supported x64 target matrix.

## Completed Baseline Alignment

- Keep README, roadmap, specs, examples, and CLI help aligned around the staged profile instead of
  older single-version MVP language.
- Preserve the existing compiler version convention unless the release process already defines an
  explicit version-string bump.
- Treat v0.14-v0.18 changes as the baseline; do not roll back feature slices while integrating v0.5
  documentation and gates.

## Completed Release Gates

- Add a concise v0.5 release gate checklist for docs, tests, smoke coverage, generated docs manifest
  verification, and known-risk signoff.
- Keep generated manifest updates deterministic and intentional.
- Verify docs without changing `docs/generated/manifest.json` unless compiler metadata has actually
  changed.

## Completed Integration Confidence

- Confirm examples and smoke flows cover the staged profile: Flow hello, bool/range/enum/match,
  effects, Islands/capabilities, object linking, actors, formatting, tests, and diagnostics.
- Keep platform-specific runtime and object-linking notes visible in docs so failures can be triaged
  against the intended profile.
- Make deferred work explicit so v0.5 does not blur into future language claims.

## Deferred Beyond v0.5

Payload enums, exhaustive match checking, collection `for`, closures and comprehensions, full
Rust-grade ownership, full structured concurrency, UI DSL, web/native UI backends, production
package publishing, proof-carrying capsules, richer effect inference or polymorphism, and the
complete EcoNet/Todex ecosystem remain post-v0.5 work.
