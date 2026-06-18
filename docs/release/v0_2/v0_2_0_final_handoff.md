# Tetra v0.2.0 Final Handoff

Status: archived green release-gate handoff for the historical `v0.2.0` cut.

The `v0.2.0` version metadata was the release line for this archived handoff.
Fresh release-gate evidence for that cut is archived under
`reports/v0_2_0_candidate_20260428-rerun-210701` with `status: pass`,
`failed_count: 0`, and `step_count: 35`.

Checklist source of truth: `docs/checklists/v0_2_0_release_gate.md`.
Scope contract: `docs/spec/v0_2_scope.md`.
Cut guide: `docs/release/v0_2_0_release_cut_guide.md`.

## Release State

- Date: 2026-04-28 gate run, `started_at: 2026-04-28T18:07:01Z`,
  `ended_at: 2026-04-28T18:07:16Z`.
- Branch: `main`, from
  `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/release-state.txt`.
- Version: `v0.2.0`; version preflight and short-alias parity passed in
  `reports/v0_2_0_candidate_20260428-rerun-210701/logs/02-version-preflight-v0-2-0-required.log`
  and
  `reports/v0_2_0_candidate_20260428-rerun-210701/logs/03-short-alias-version-parity.log`.
- Release archive path: `reports/v0_2_0_candidate_20260428-rerun-210701`.
- Security signoff path:
  `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/security-review.md`;
  source signoff is `reports/v0_2_0_security_review_20260428-180419.md`.
- Git status evidence:
  `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/release-state.txt`
  reports `status: pass`, `dirty tracked files: 215`,
  `untracked release artifacts: 3`, `required artifacts: 36`, and
  `missing artifacts: 0`.

## Fresh Verification

All completed rows below cite the current release archive
`reports/v0_2_0_candidate_20260428-rerun-210701`.

### Bootstrap Binaries

- Command or gate step: `bash scripts/dev/bootstrap.sh`
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/01-bootstrap-tetra-binaries.log`
- Result: pass, exit `0`.

### Version Preflight

- Command or gate step: `check_release_version`; `check_short_alias_version`
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/02-version-preflight-v0-2-0-required.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/03-short-alias-version-parity.log`
- Result: pass, `v0.2.0`.

### Go Workspace Tests

- Command or gate step:

  ```sh
  go test ./compiler/... ./cli/... ./tools/... -count=1
  ```

- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/04-go-test-packages.log`
- Result: pass, exit `0`.

### Full Wrapper

- Command or gate step:

  ```sh
  bash scripts/ci/test-all.sh --full --keep-going \
    --report-dir reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/test-all
  ```

- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/test-all/summary.json`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/05-full-stabilization-wrapper.log`
- Result: pass, `mode: full`, `failed_count: 0`, `step_count: 24`.

### Quick Wrapper

- Command or gate step:

  ```sh
  bash scripts/ci/test-all.sh --quick --report-dir <dir>/test-all-quick
  ```

- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/test-all-quick/summary.json`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/test-all-quick/summary.md`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/H-test-all-quick.log.exit`
- Result: pass, `mode: quick`, `failed_count: 0`, exit `0`.

### Formatter And Flow Gates

- Command or gate step:

  ```sh
  ./tetra fmt --check examples lib __rt compiler/selfhostrt
  go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
  ```

- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/10-formatter-check.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/06-flow-only-source-scan.log`
- Result: pass, exit `0`.

### CLI Report Gates

- Command or gate step: targets, doctor, and tetra-test JSON validation steps.
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/07-targets-report-validation.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/08-doctor-report-validation.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/11-tetra-test-examples-json.log`
- Result: pass, exit `0`.

### Smoke Gates

- Command or gate step: smoke list, host, cross-target, WASI, and Web UI smoke steps.
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/16-smoke-list-validation.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/17-native-host-smoke-linux-x64.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/18-build-only-smoke-linux-x64.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/19-build-only-smoke-macos-x64.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/20-build-only-smoke-windows-x64.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/21-build-only-smoke-wasm32-wasi.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/22-build-only-smoke-wasm32-web.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/23-wasi-runner-smoke.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/24-web-ui-browser-smoke.log`
- Result: pass, exit `0`.

### Docs Verification

- Command or gate step:

  ```sh
  go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
  ```

- Command or gate step: docs manifest validation.
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/12-docs-manifest-regenerate-validate.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/13-docs-verification-and-doctests.log`
- Result: pass, exit `0`.

### API Diff Gate

- Command or gate step: `check_api_diff`
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/26-api-diff-gate.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/api-diff/api-diff.json`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/api-diff/api-docs.md`
- Result: pass; baseline `docs/baselines/api-diff-baseline.v1alpha1.json`.

### Security Signoff

- Command or gate step: `check_security_review_signoff`
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/25-security-review-signoff.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/security-review.md`
- Result: pass; decision approved for `v0.2.0` release.

### Performance Regression

- Command or gate step: `check_performance_regression_artifact`
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/27-performance-regression-evidence.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/performance-regression.json`
- Result: pass; `metric_count: 11`.

### Binary Size Thresholds

- Command or gate step: `check_binary_size_thresholds`
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/28-binary-size-thresholds.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/binary-size-thresholds.json`
- Result: pass; `pass_count: 5`, `fail_count: 0`.

### Reproducible Build Proof

- Command or gate step: `check_repro_build`
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/29-reproducible-build-proof.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/reproducible-build.json`
- Result: pass; `matched_count: 5`, `mismatched_count: 0`.

### Release State

- Command or gate step: `check_release_state`
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/34-release-state-audit.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/release-state.txt`
- Result: pass; `missing artifacts: 0`.

### Artifact Hashes

- Command or gate step: `check_artifact_hash_manifest`
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/32-artifact-hash-manifest.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/35-artifact-hash-manifest-refresh.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/artifact-hashes.json`
- Result: pass; manifest contains 69 artifacts.

### Final Release Gate

- Command or gate step:

  ```sh
  bash scripts/release/v0_2_0/gate.sh \
    --report-dir reports/v0_2_0_candidate_20260428-rerun-210701
  ```

- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/summary.json`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/summary.md`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/gate.log`
- Result: pass; `failed_count: 0`, `step_count: 35`.

## Verification Matrix Signoff

The final verification matrix in `docs/checklists/v0_2_0_release_gate.md` now
cites `reports/v0_2_0_candidate_20260428-rerun-210701` as the current green
gate evidence. The checklist has no unchecked entries after the quick wrapper
evidence was added.

The green gate covers these release-facing areas:

- Go workspace tests.
- Quick test wrapper.
- Full test wrapper.
- Formatter and Flow gates.
- CLI report gates.
- Smoke report gates.
- Docs/API gates covered by the gate's manifest verification, docs verifier,
  tetra-doc output validation, and API diff step.
- Release-state and artifact hash gates.
- Security, reproducible build, binary-size, and performance gates.
- Final `v0.2.0` gate.

Stale evidence rule: copied summaries, generated artifacts, or smoke reports
from another commit, branch, version, or report directory are not acceptable
release proof.

## Integration Notes

- API baseline refresh: the API diff gate passed and recorded
  `docs/baselines/api-diff-baseline.v1alpha1.json` as the baseline, with
  report artifacts at
  `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/api-diff/api-diff.json`
  and
  `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/api-diff/api-docs.md`.
- Artifact-hash refresh: the final hash refresh step passed in
  `reports/v0_2_0_candidate_20260428-rerun-210701/logs/35-artifact-hash-manifest-refresh.log`;
  `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/artifact-hashes.json`
  contains 69 artifact entries.
- Gate pass: `reports/v0_2_0_candidate_20260428-rerun-210701/summary.json`
  records `status: pass`, `failed_count: 0`, and `step_count: 35`.
- Known issues: `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/known_issues.md`
  says no known issues were emitted automatically by the release gate.

## Remaining Release Actions

None for the archived 35-step green gate closure. This handoff only claims
checks with direct evidence in
`reports/v0_2_0_candidate_20260428-rerun-210701`; the checklist currently has
no unchecked entries.
