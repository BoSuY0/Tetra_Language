# v1.0 Security Review Gate

Status: future v1.0 evidence checklist. This checklist is required before
`v1.0.0` signoff, but it is not a claim that the current `v0.2.0` baseline
satisfies the future v1.0 security signoff.

## Scope

Security review covers the release surfaces that can bypass normal safety,
touch host capabilities, or move package trust across boundaries:

- unsafe builtins and scoped `unsafe` usage
- capability tokens and effect declarations
- Eco/Capsule permission and trust metadata
- WASM/WASI/web host boundaries
- privacy, consent, and resource budget decisions

## Required Evidence

- [ ] Unsafe builtins inventory is current.
  - Evidence source: `docs/spec/unsafe.md`.
  - Evidence command: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- [ ] Capability boundary map is current.
  - Evidence source: `docs/spec/capabilities.md` and `docs/spec/effects_capabilities_privacy_v1.md`.
  - Evidence command: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- [ ] Safe/unsafe boundary tests pass.
  - Evidence command: `go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`.
- [ ] Capsule permission failure tests pass.
  - Evidence command: `go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`.
- [ ] Eco package trust threat model is reviewed.
  - Evidence source: `docs/spec/eco_publishing_v1.md`.
  - Evidence command: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- [ ] WASM/WASI/web host threat model is reviewed.
  - Evidence source: `docs/backend/wasm_backend_plan.md` and `docs/user/wasm_ui_guide.md`.
  - Evidence command: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- [ ] Privacy/consent/resource-budget release decision is reviewed.
  - Evidence source: `docs/spec/effects_capabilities_privacy_v1.md`.
  - Evidence source: `docs/spec/v1_scope.md`.
  - Evidence command: `go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`.
  - Evidence command: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- [ ] A named reviewer signs off with the report directory, command output, and
      accepted residual risks.
  - Evidence location: release report directory, usually `<report-dir>/security-review.md`.
  - Template command: `bash scripts/release_v1_0_security_review.sh --write-template <report-dir>/security-review.md`.
  - Validation command: `bash scripts/release_v1_0_security_review.sh --signoff <report-dir>/security-review.md`.
- [ ] The signoff lists release artifact hashes.
  - Evidence format: `- <artifact file name>: sha256:<64 lowercase hex chars>`.
  - Validation command: `bash scripts/release_v1_0_security_review.sh --signoff <report-dir>/security-review.md`.

## Threat Model Notes

Unsafe review must verify that every unsafe-only builtin is documented, remains
inside explicit unsafe syntax, and has the matching `uses` declaration where an
effect is observable.

Capability review must verify that `uses` declarations do not grant tokens by
themselves, that capability tokens are obtained only through the documented
unsafe builtins, and that missing capabilities produce diagnostics or failing
tests.

Effects review must verify that stable `lib/core` module `// Effects:`
metadata matches the compiler-parsed `uses` declarations. The docs verifier
performs this audit and should fail the release if wrapper metadata drifts.

Eco review must verify that local capsule permissions do not escalate through
dependencies, that trust metadata is treated as beta/local evidence rather than
a global trust network, and that post-v1 trust claims remain out of the v1
contract.

WASM/WASI/web review must verify that build-only smoke is not treated as runtime
isolation evidence, that host imports are documented, and that browser/WASI
runner reports are archived before release signoff.

## Closure Rule

Do not check this gate complete until all required evidence commands have been
run in the same branch state and the reviewer signoff file validates in the
release artifact archive.
