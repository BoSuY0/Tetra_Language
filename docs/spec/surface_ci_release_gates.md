# Surface CI Release Gates

Status: current production-gate contract for the scoped Surface production
claim path.

`scripts/release/surface/prod-gate.sh` is the final Surface CI/release
aggregator for `PROD_STABLE_SCOPED_LINUX_WEB_APP_UI`. It does not broaden the
Surface claim beyond the scoped Linux/web matrix. It allows the production tier
only when the same commit has passing release evidence, production gate
evidence, artifact hashes, and release workflow wiring.

## Report Contract

The production gate writes `tetra.surface.prod-gate-report.v1` at level
`surface-production-ci-release-gate-v1`. The report is stored under the Surface
release evidence directory as `surface-prod-gate-report.json`, and the final
artifact upload directory also writes `surface-prod-gate-summary.json` and
`artifact-hashes.json`.

The validator command is:

`go run -buildvcs=false ./tools/cmd/validate-surface-release-state --report-dir <surface-release-v1> --scope PROD_STABLE_SCOPED_LINUX_WEB_APP_UI`

That production scope remains invalid unless `surface-release-summary.json`,
Linux/web release target reports, Morph evidence, artifact hashes, docs
manifest state, and `surface-prod-gate-report.json` all validate together.

## Required CI Wiring

The required release job is `.github/workflows/release-packages.yml` job
`release-packages`. It must run:

`bash scripts/release/surface/prod-gate.sh --report-dir <out>/surface-production-final`

The job must upload `surface-production-final/**` as package evidence and must
not use `continue-on-error: true`. A missing production job, a missing
`surface-production-final` artifact upload, or a production gate hidden behind
`continue-on-error` is a release blocker.

## Aggregated Gates

The final gate aggregates these evidence classes:

- Surface v1 release evidence from `release-gate.sh`;
- Block System and Morph evidence through the release gate subdirectories;
- visual regression from `visual-gate.sh`;
- package distribution from `package-gate.sh`;
- security sandbox evidence from `security-gate.sh`;
- IPC/lifecycle evidence from `ipc-lifecycle-gate.sh`;
- crash diagnostics from `crash-gate.sh`;
- i18n/localization from `i18n-gate.sh`;
- performance/memory from `perf-gate.sh`;
- widget-to-Block/Morph migration from `migration-gate.sh`;
- production example suite from `example-suite-gate.sh`;
- API stability from `api-stability-gate.sh`;
- production claim governance from `validate-surface-prod-claim`.

Every aggregated gate must run, pass, avoid skip-as-pass behavior, and publish
an artifact hash manifest. Missing artifact hash manifests are release
blockers.

## Target Tiering

For this production tier, `linux-x64` and `wasm32-web` are the only production
runtime targets. `headless` remains release/test evidence. `windows-x64` and
`macos-x64` remain beta or unsupported target-host boundaries until real
target-host gates promote them. Skipped Linux/web production targets cannot be
counted as pass evidence.

## Negative Guards

The production gate report must keep these rejection guards true:

- missing production CI job rejected;
- `continue-on-error` production job rejected;
- skipped target counted as pass rejected;
- missing artifact hash manifest rejected.

These guards prevent a docs-only or CI-only production claim from passing.
Surface can claim `PROD_STABLE_SCOPED_LINUX_WEB_APP_UI` only when the
machine-readable gate proves the exact same tier.
