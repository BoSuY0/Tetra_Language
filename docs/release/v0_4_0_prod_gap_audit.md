# Tetra v0.4.0 Production Gap Audit

Status: final audit for the scoped Linux-x64 `v0.4.0` production objective.

This document does not tag the repository. It records the selected production
claim and the release evidence that closes it.

Machine-readable gap matrix:
`docs/release/v0_4_0_prod_gap_matrix.json`.

Production backlog:
`docs/release/v0_4_0_prod_backlog.md`.

Scope decision files:
`docs/release/v0_4_0_scope_decisions.md` and
`docs/release/v0_4_0_scope_decisions.json`.

Scope decision status: Linux-x64 production scope selected. EcoNet, full v1.0
language guarantees, WASM production runtime targets, Windows, and macOS are
excluded from the scoped `v0.4.0` production claim.

## Objective

Finish Tetra as production-grade `0.4.0` for Linux x64 first, with all selected
behavior implemented as real production behavior and no mock/fake claims.

Concrete completion means:

- `./tetra version` and `./t version` print `v0.4.0`.
- The feature registry has no production-claimed selected feature left as
  `experimental`, `planned`, or `post-v1`.
- `linux-x64` runtime execution is proven by fresh smoke evidence.
- Memory, parallelism, and compiler production cores are proven by executable
  Linux-x64 smoke artifacts and validators.
- Docs no longer market placeholder, metadata-only, build-only, preview, or
  excluded behavior as scoped `v0.4.0` production.
- Production gates prove every promoted feature through implementation, runtime
  execution where applicable, docs verification, security review, artifact
  hashes, and clean worktree.
- Excluded features are named out-of-scope for `v0.4.0`.

## Current Evidence

| Requirement | Current evidence | Status |
| --- | --- | --- |
| Version is `v0.4.0` | local version metadata and generated manifest report `v0.4.0`. | pass |
| Feature registry is `v0.4.0` | `reports/v0.4.0/features.json` reports `v0.4.0`; selected scoped features are current. | pass |
| Target matrix includes runnable Linux | `reports/v0.4.0/targets.json` reports `linux-x64` as supported and runnable on the current host. | pass |
| Linux host runtime smoke | `reports/v0.4.0/linux-host-smoke.json` passes 64/64 cases. | pass |
| Memory production core | Canonical gate requires `artifacts/memory-production-linux-x64.json` with schema `tetra.memory.production.v1`. | pass in expanded gate |
| Parallel production core | Canonical gate requires `artifacts/parallel-production-linux-x64.json` with schema `tetra.parallel.production.v1`. | pass in expanded gate |
| Compiler production core | Canonical gate requires `artifacts/compiler-production-linux-x64.json` with schema `tetra.compiler.production.v1`. | pass in expanded gate |
| Linux distributed actors | `reports/v0.4.0/distributed-actors-linux-x64.json`. | pass |
| Linux native UI runtime | `reports/v0.4.0/native-ui-linux-x64.json`. | pass |
| Readiness preflight | `go run ./tools/cmd/validate-v0-4-readiness --expected-version v0.4.0 --features reports/v0.4.0/features.json --targets reports/v0.4.0/targets.json --manifest docs/generated/manifest.json --scope-decisions docs/release/v0_4_0_scope_decisions.json` exits 0. | pass |
| Final release gate | `bash scripts/release/v0_4_0/gate.sh --report-dir /tmp/tetra-v0.4.0-final-production-gate --require-clean` validates the expanded scoped gate from the clean candidate branch. | pass for clean candidate gate |
| Security signoff | `reports/v0.4.0/security-review.md` validates with `scripts/release/v0_4_0/security-review.sh`. | pass |
| Worktree can be tagged | The final gate runs with `--require-clean` and writes a release-state artifact with `git.clean: true`. | pass for clean candidate gate |

## Selected Production Surface

| Feature/target | Status |
| --- | --- |
| `language.callable-level1` | selected/current |
| `language.callable-level2` | selected/current |
| `language.full-first-class-callables` | selected/current |
| `language.lifetime-ssa` | selected/current |
| `memory.production-core` | selected; canonical gate evidence required |
| `parallel.production-core` | selected; canonical gate evidence required |
| `compiler.production-core` | selected; canonical gate evidence required |
| `stdlib.experimental-mirrors` | selected/current |
| `ui.metadata-v1` | selected/current |
| `actors.distributed-runtime` | selected/current for Linux-x64 |
| `ui.native-runtime` | selected/current for Linux-x64 |
| `linux-x64` | selected/supported/runnable |

## Excluded From v0.4.0 Production

- `eco.distributed-network`
- `language.full-v1-guarantees`
- `wasm.runtime-execution`
- `windows-x64`
- `macos-x64`
- `wasm32-wasi`
- `wasm32-web`

These may exist as future, supported, experimental, or non-production surfaces,
but they are not part of the scoped `v0.4.0` production claim.

## Current Conclusion

The old all-platform/all-feature objective is no longer the release contract.
The active contract is Linux-x64 production without EcoNet. The scoped
readiness, runtime/security evidence, memory production, parallel production,
compiler production, artifact hashes, and clean release-state evidence are all
required by the expanded canonical gate.
