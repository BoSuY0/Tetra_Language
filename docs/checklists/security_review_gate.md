# v1.0 Security Review Gate

Status: future v1.0 evidence checklist. This checklist is required before
`v1.0.0` signoff, but it is not a claim that the current `v0.3.0` baseline
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

Each checked row must fill all evidence fields in the table. Empty cells,
template text, stale report directories, or command names without outcomes do
not close the row.

| Item | Evidence source | Command | Required evidence fields |
| --- | --- | --- | --- |
| Unsafe builtins inventory is current | `docs/spec/unsafe.md` | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` | commit, command exit code, docs verifier output, reviewer/date |
| Capability boundary map is current | `docs/spec/capabilities.md`; `docs/spec/effects_capabilities_privacy_v1.md` | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` | commit, command exit code, cited docs paths, reviewer/date |
| Safe/unsafe boundary tests pass | compiler safety packages | `go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1` | commit, command exit code, package summary, failing test names if any |
| Capsule permission failure tests pass | CLI/tooling tests | `go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1` | commit, command exit code, package summary, failing test names if any |
| Eco package trust threat model is reviewed | `docs/spec/eco_publishing_v1.md` | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` | commit, command exit code, accepted trust boundary decision, reviewer/date |
| WASM/WASI/web host threat model is reviewed | `docs/backend/wasm_backend_plan.md`; `docs/user/wasm_ui_guide.md` | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` | commit, command exit code, WASI report path, web UI report path, reviewer/date |
| Privacy/consent/resource-budget release decision is reviewed | `docs/spec/effects_capabilities_privacy_v1.md`; `docs/spec/v1_scope.md` | `go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`; `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` | commit, both exit codes, resource-budget decision, reviewer/date |
| Named reviewer signoff validates | release report directory | `bash scripts/release/v1_0/security-review.sh --signoff REPORT_DIR/artifacts/security-review.md` | reviewer identity, report directory, commit, command exit code, accepted residual risks |
| Signoff lists release artifact hashes | `REPORT_DIR/artifacts/security-review.md` | `bash scripts/release/v1_0/security-review.sh --signoff REPORT_DIR/artifacts/security-review.md` | artifact file name, `sha256:<64 lowercase hex chars>`, command exit code |

## Known Residual Risks

This table is the minimum summary required for the security, performance, and
fuzz/stress surfaces. A final release can replace `accepted for RC review` only
with a concrete owner decision from the release report.

| Surface | Known residual risk | Current evidence | Release decision field |
| --- | --- | --- | --- |
| Security | WASM build-only reports are not host-isolation proof; browser/WASI runtime proof must come from dedicated smoke scripts. | `reports/plan250/backend/web-ui-smoke.json`; `reports/plan250/backend/wasi-smoke.json` | owner/date plus accepted/blocking decision |
| Performance | Plan250 benchmark snapshot used `-count=1`, not the release-candidate `-count=5` threshold run. | `docs/performance/v1_0_thresholds.md` | owner/date plus accepted/blocking decision |
| Fuzz/stress | Short fuzz smoke can miss unstable seeds; crashers must become deterministic regression tests before promotion. | `docs/testing/fuzz_property_stress.md` | owner/date plus accepted/blocking decision |

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
