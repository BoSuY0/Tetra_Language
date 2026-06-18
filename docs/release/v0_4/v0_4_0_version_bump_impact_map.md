# Tetra v0.4.0 Version Bump Impact Map

Status: final for the scoped Linux-x64/no-EcoNet `v0.4.0` candidate.

This file records the version-promotion impact areas that were checked while
moving the repository from the older `v0.3.0` identity to the scoped `v0.4.0`
release candidate.

## Current Version Source

- `compiler/internal/version/version.go`
  - Current constant: `CompilerVersion = "v0.4.0"`.
- `./tetra version` and `./t version` report `v0.4.0`.

## Release Gate And Validator Impact

The `v0.4.0` release line has dedicated scoped infrastructure:

- `scripts/release/v0_4_0/gate.sh`
- `scripts/release/v0_4_0/security-review.sh`
- `tools/cmd/validate-v0-4-readiness`
- `tools/cmd/validate-v0-4-completion-audit`
- `tools/cmd/validate-v0-4-release-state`
- `tools/cmd/validate-memory-production`
- `tools/cmd/validate-parallel-production`
- `tools/cmd/validate-compiler-production`
- `docs/checklists/v0_4_0_release_gate.md`
- `docs/release/v0_4_0_final_handoff.md`

The gate artifact identity is:

- `tetra.release.v0_4_0.gate-report.v1`

The historical clean release snapshot evidence is:

- `reports/v0.4.0/release-gate-clean/summary.json`
- `reports/v0.4.0/release-gate-clean/artifact-hashes.json`
- `reports/v0.4.0/release-gate-clean/artifacts/release-state.json`

The expanded canonical gate additionally requires these report artifacts:

- `artifacts/memory-production-linux-x64.json`
- `artifacts/parallel-production-linux-x64.json`
- `artifacts/compiler-production-linux-x64.json`

## Scope Impact

The selected production scope is Linux-x64 only, with EcoNet explicitly
excluded. Non-Linux runtimes, WASM production runtime claims, and full v1.0
language guarantees are not part of this `v0.4.0` production claim.

Machine-readable scope:

- `docs/release/v0_4_0_scope_decisions.json`

Human-readable scope:

- `docs/spec/v0_4_scope.md`
- `docs/release/v0_4_0_scope_decisions.md`

## Generated Evidence Impact

The current generated evidence and release reports include:

- `docs/generated/manifest.json`
- `docs/generated/v1_0/manifest.json`
- `reports/v0.4.0/features.json`
- `reports/v0.4.0/targets.json`
- `reports/v0.4.0/linux-host-smoke.json`
- `reports/v0.4.0/distributed-actors-linux-x64.json`
- `reports/v0.4.0/native-ui-linux-x64.json`
- `reports/v0.4.0/security-review.md`

## Completion Rule

The scoped `v0.4.0` candidate is complete when the Linux-x64 gate passes from a
clean committed candidate with:

- selected feature/runtime readiness passing
- Linux host smoke passing
- memory production smoke and validation passing
- parallel production smoke and validation passing
- compiler production smoke and validation passing
- distributed actors Linux-x64 smoke passing
- native UI Linux-x64 smoke passing
- baseline compiler/CLI/tools tests passing
- docs/manifest verification passing
- security signoff passing
- artifact hash validation passing
- clean release-state passing

The final clean candidate branch supplies the fresh clean gate report required
by the expanded memory/parallel/compiler rule, so the version bump is final for
the scoped Linux-x64/no-EcoNet release contract.
