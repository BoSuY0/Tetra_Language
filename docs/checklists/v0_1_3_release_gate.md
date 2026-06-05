# v0.1.3 Production Release Gate

Status: checked against the canonical `v0.1.3` release evidence archive.
Required checkboxes below are tied to
`reports/codex-v0_1_3-post-bump-release-gate-2`, produced by
`scripts/release/v0_1_3/gate.sh` with `33` passing steps and `0` failures.
Tracked snapshots live in `docs/generated/v1_0`.

Canonical scope: `docs/spec/v1_scope.md`.
Artifact policy: `docs/release/artifact_policy.md`.
RC process: `docs/release/rc_process.md`.
Security review gate: `docs/checklists/security_review_gate.md`.
Robustness suite: `docs/testing/fuzz_property_stress.md`.
Performance thresholds: `docs/performance/v1_0_thresholds.md`.

Gate evidence archive layout:

- Summary and step logs: `<report-dir>/summary.json`,
  `<report-dir>/summary.md`, and `<report-dir>/logs/*.log`.
- Release-state audit: `<report-dir>/artifacts/release-state.json` and
  `<report-dir>/artifacts/release-state.txt`.
- Artifact integrity: `<report-dir>/artifacts/artifact-hashes.json`.
- Known issues: `<report-dir>/artifacts/known_issues.md`.
- Targets, diagnostics, doctor, test, smoke, docs, API diff, security, and
  reproducibility artifacts are named next to the checkbox that they satisfy.

## Hard Blockers

- [x] Version preflight: `./tetra version` reports `v0.1.3`.
  - Evidence command: `bash scripts/dev/bootstrap.sh && ./tetra version && ./t version`.
  - Evidence artifacts: `<report-dir>/logs/01-bootstrap-tetra-binaries.log`,
    `<report-dir>/logs/02-version-preflight-v0-1-3-required.log`, and
    `<report-dir>/artifacts/release-state.json`.
  - Blocking gate: `scripts/release/v0_1_3/gate.sh`.
- [x] API diff no-change policy passes against the reviewed baseline.
  - Evidence command: `bash scripts/release/v1_0/api-diff.sh --report-dir <dir>/api-diff --baseline docs/baselines/api-diff-baseline.v1alpha1.json --enforce no-change`.
  - Evidence artifacts: `<dir>/api-diff/api-docs.md` and
    `<dir>/api-diff/api-diff.json`; the diff report includes
    `review.status`, `review.checklist`, and per-change `review_status` fields.
  - Blocking gate: `scripts/release/v0_1_3/gate.sh`.
- [x] No required checklist item is checked without an artifact, log, or test
      command recorded in the release evidence archive.
  - Evidence command: inspect `<report-dir>/summary.json`, `<report-dir>/summary.md`, and artifact paths listed below.
  - Blocking gate: release reviewer signoff.

## Required Language And Safety

- [x] Flow syntax is the only official release syntax in examples, docs,
      formatter output, and release smoke coverage.
  - Evidence command: `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`.
  - Evidence artifact: `<report-dir>/logs/*-flow-only-source-scan.log`.
- [x] Parser, formatter, and diagnostics cover the supported Flow syntax
      families.
  - Evidence command: `go test ./compiler/internal/frontend/... -count=1`.
  - Evidence artifacts: `<report-dir>/logs/*-go-test-packages.log`,
    `<report-dir>/logs/*-formatter-check.log`,
    `<report-dir>/artifacts/invalid-diagnostic.json`,
    `<report-dir>/artifacts/missing-effect-diagnostic.json`,
    `<report-dir>/artifacts/tabs-diagnostic.json`, and
    `<report-dir>/artifacts/planned-actor-diagnostic.json`.
  - Blocking gate: `scripts/release/v0_1_3/gate.sh` runs `tetra fmt --check`
    and `tetra check --diagnostics=json` cases through
    `tools/cmd/validate-diagnostic`.
- [x] Stable type system covers the mandatory v1 language scope in
      `docs/spec/v1_scope.md`.
  - Evidence command: `go test ./compiler/... -run 'Type|Inference|Enum|Optional|Protocol|Extension|Module' -count=1`.
  - Evidence artifact: `<report-dir>/logs/*-go-test-packages.log`.
- [x] Ownership, lifetime, island, actor/task transfer, and race-safety rules
      have positive and negative release tests.
  - Evidence command: `go test ./compiler/... -run 'Ownership|Borrow|Lifetime|Island|Actor|Task' -count=1`.
  - Evidence artifact: `<report-dir>/logs/*-go-test-packages.log`.
- [x] Effects, capabilities, unsafe boundaries, privacy/resource-budget
      decisions, and public diagnostics are documented and tested.
  - Evidence command: `go test ./compiler/... -run 'Unsafe|Capability|Effect|Privacy|Consent|Budget|MMIO|Mem' -count=1`.
  - Evidence command: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
  - Evidence artifacts: `<report-dir>/logs/*-go-test-packages.log` and
    `<report-dir>/logs/*-docs-verification-and-doctests.log`.
  - Evidence note: docs verification audits stable `lib/core` effect metadata
    against the module `uses` declarations.
- [x] Unsafe/capability/privacy/Eco/WASM security review has complete evidence
      and reviewer signoff.
  - Evidence checklist: `docs/checklists/security_review_gate.md`.
  - Evidence workflow: `bash scripts/release/v1_0/security-review.sh --write-template <report-dir>/security-review.md`, then `bash scripts/release/v1_0/security-review.sh --signoff <report-dir>/security-review.md`.
  - Evidence artifact: `<report-dir>/artifacts/security-review.md`.
  - Blocking gate: release reviewer signoff.

## Required Compiler, Targets, And Runtime

- [x] Native release builds pass for `linux-x64`, `macos-x64`, and
      `windows-x64`.
  - Evidence command: `./tetra smoke --target <target> --run=false --report <path>`.
  - Evidence artifacts: `<report-dir>/artifacts/linux-smoke.json`,
    `<report-dir>/artifacts/macos-smoke.json`, and
    `<report-dir>/artifacts/windows-smoke.json`.
- [x] Native host smoke runs on the host platform.
  - Evidence command: `./tetra smoke --target linux-x64 --run=true --report <path>` on Linux.
  - Evidence artifact: `<report-dir>/artifacts/host-smoke.json`.
- [x] Runtime ABI, actor runtime override, and TOBJ link-object compatibility
      matrix has fresh build evidence.
  - Evidence command: `go test ./compiler/... -run 'Runtime|ABI|Object|Link|Actor|Actors' -count=1`.
  - Evidence artifact: `<report-dir>/logs/*-go-test-packages.log`.
  - Evidence note: non-host `macos-x64` and `windows-x64` runtime binaries are
    build-only evidence unless executed on matching hosts.
- [x] WASM build-only smoke passes for `wasm32-wasi` and `wasm32-web`.
  - Evidence command: `./tetra smoke --target wasm32-wasi --run=false --report <path>` and `./tetra smoke --target wasm32-web --run=false --report <path>`.
  - Evidence artifacts: `<report-dir>/artifacts/wasm32-wasi-smoke.json` and
    `<report-dir>/artifacts/wasm32-web-smoke.json`.
- [x] WASI runner smoke produces a validated report.
  - Evidence command: `bash scripts/release/v1_0/wasi-smoke.sh --report <path>`.
  - Evidence artifact: `<report-dir>/artifacts/wasi-smoke.json`.
- [x] Web UI/browser smoke produces a validated UI-specific report.
  - Evidence command: `bash scripts/release/v1_0/web-smoke.sh --report <path>`.
  - Evidence artifact: `<report-dir>/artifacts/web-ui-smoke.json`.
  - Host policy: missing or crashing headless Chromium records a `blocked`
    report, fails the gate, and does not satisfy this checkbox.
- [x] Native shell UI emits a deterministic metadata sidecar for the native
      smoke source.
  - Evidence command: `./tetra smoke --target linux-x64 --run=false --report <path>` on Linux.
  - Evidence artifact: `<report-dir>/artifacts/linux-smoke.json`.
  - Scope note: native shell UI is a v1 metadata preview sidecar, not a full
    native widget toolkit.
  - Evidence validation: `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report <path>`.
- [x] Reproducible build proof exists for at least one native target and one
      WASM target.
  - Evidence command: `bash scripts/release/v1_0/reproducible-build.sh --report <path>`.
  - Evidence artifact: `<report-dir>/artifacts/reproducible-build.json`.
  - Timestamp policy: the proof JSON omits wall-clock timestamps; release gate
    summaries provide run timestamps so tracked proof snapshots do not churn.

## Required Stdlib, Tooling, Docs, And Eco

- [x] Stable stdlib modules have API docs and example doctests where required,
      effects metadata, formatter coverage, and API diff metadata.
  - Evidence command: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
  - Evidence artifacts: `<report-dir>/artifacts/tetra-docs.md`,
    `<report-dir>/artifacts/api-diff/api-docs.md`, and
    `<report-dir>/artifacts/api-diff/api-diff.json`.
  - Scope note: `lib.experimental.*` modules are labeled experimental in
    generated docs and stable examples must import `lib.core.*` directly.
- [x] CLI commands required by `docs/spec/v1_scope.md` are tested.
  - Evidence command: `go test ./cli/... -count=1`.
  - Evidence artifacts: `<report-dir>/logs/*-go-test-packages.log`,
    `<report-dir>/artifacts/targets.json`, and
    `<report-dir>/artifacts/doctor.json`.
- [x] JSON diagnostics, test reports, target reports, doctor reports, smoke
      reports, and API docs are validated.
  - Evidence command: `go test ./tools/... -count=1`.
  - Evidence artifacts: `<report-dir>/artifacts/tetra-test-report.json`,
    `<report-dir>/artifacts/targets.json`,
    `<report-dir>/artifacts/doctor.json`,
    `<report-dir>/artifacts/smoke-list.json`, and
    `<report-dir>/artifacts/tetra-docs.md`.
  - Blocking gate: `scripts/release/v0_1_3/gate.sh` includes the
    `json diagnostic shape` step.
- [x] Fuzz/property/stress smoke suite passes and nightly fuzz commands are
      documented.
  - Evidence command: `go test ./compiler/... ./cli/... ./tools/... -run 'Fuzz|Property|Stress' -count=1`.
  - Evidence docs: `docs/testing/fuzz_property_stress.md`.
  - Evidence artifacts: `<report-dir>/logs/*-go-test-packages.log` and
    `<report-dir>/artifacts/test-all/summary.json`.
- [x] Performance benchmarks have RC baselines, thresholds, and reviewer
      decisions for regressions.
  - Evidence command: `go test ./compiler/... -bench='Benchmark(CompileRepresentativeExamples|FormatRepresentativeSources|GenerateAPIDocsDogfoodProjects|BinarySizeBaselines)' -run '^$' -count=5`.
  - Evidence docs: `docs/performance/v1_0_thresholds.md`.
  - Evidence artifact: `<report-dir>/logs/*-go-test-packages.log`.
- [x] LSP stdio baseline has validated diagnostics/symbol/hover evidence.
  - Evidence command: `bash scripts/ci/test-all.sh --full --keep-going --report-dir <dir>/test-all`.
  - Evidence artifact: `<report-dir>/artifacts/test-all/summary.json`.
- [x] Local Eco package lifecycle covers verify, dependency lock, pack/unpack,
      vault, and publish metadata fixtures.
  - Evidence command: `bash scripts/ci/test-all.sh --full --keep-going --report-dir <dir>/test-all`.
  - Evidence artifact: `<report-dir>/artifacts/test-all/summary.json`.
- [x] Release documentation set is complete: user docs, contributor docs,
      artifact policy, RC process, maintenance policy, release notes, and known
      issues template.
  - Evidence command: `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
  - Evidence artifacts: `<report-dir>/logs/*-docs-verification-and-doctests.log`,
    `<report-dir>/artifacts/known_issues.md`,
    `<report-dir>/artifacts/release-state.json`, and
    `<report-dir>/artifacts/artifact-hashes.json`.

## Required Aggregate Commands

- [x] `go test ./compiler/... ./cli/... ./tools/... -count=1`
  - Evidence artifact: `<report-dir>/logs/*-go-test-packages.log`.
- [x] `bash scripts/ci/test-all.sh --full --keep-going --report-dir <dir>/test-all`
  - Evidence artifact: `<report-dir>/artifacts/test-all/summary.json`.
- [x] `bash scripts/release/v0_1_3/gate.sh --report-dir <dir>/release-gate`
  - Evidence artifacts: `<report-dir>/summary.json` and
    `<report-dir>/artifacts/release-state.json`.
- [x] `git diff --check`
  - Evidence artifact: final handoff command output.

## Optional Or Beta Surface

- [x] Network package publishing is explicitly labeled beta if present.
- [x] TetraHub integration is explicitly labeled beta if present.
- [x] Target-aware downloads and trust metadata are explicitly labeled beta if
      present.

Optional/beta items must not be required for `v0.1.3` unless promoted through
scope review and added to `docs/spec/v1_scope.md`.

## Informational Post-v1 Items

These are not required for `v0.1.3`:

- Distributed EcoNet and production TetraHub publishing.
- Proof-carrying capsules and global trust scoring.
- EcoOracle, live evolution, time-travel execution, and multiverse optimizer.
- Advanced AI/model types and model-runtime integration.
- Distributed actors beyond the release actor/task safety contract.

## Expected Current State

On the `v0.1.3` release branch:

- `bash scripts/dev/bootstrap.sh && ./tetra version && ./t version` reports
  `v0.1.3`.
- `TETRA_SECURITY_REVIEW_SIGNOFF=<current-signoff> bash scripts/release/v0_1_3/gate.sh --report-dir <dir>/release-gate`
  must pass before the release label is attached.
- Required checkboxes above must stay linked to evidence produced in the same
  branch state and archived with the release report.
