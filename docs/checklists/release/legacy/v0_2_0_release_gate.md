# v0.2.0 Release Gate Checklist

Status: green gate evidence integrated for the `v0.2.0` release line. This
checklist is release proof only for checked rows that cite fresh evidence from
the exact candidate archive.

Current green gate evidence:
`reports/v0_2_0_candidate_20260428-rerun-210701`.
Gate summary: `reports/v0_2_0_candidate_20260428-rerun-210701/summary.json`
records `status: pass`, `failed_count: 0`, and `step_count: 35`.
Gate log: `reports/v0_2_0_candidate_20260428-rerun-210701/gate.log`.

Scope contract: `docs/spec/v0_2_scope.md`.
Current release truth: `docs/spec/current_supported_surface.md`.
Artifact policy: `docs/release/artifact_policy.md`.
Cut guide: `docs/release/v0_2_0_release_cut_guide.md`.
Final handoff: `docs/release/v0_2_0_final_handoff.md`.

## Hard Blockers

- [x] Version preflight: `./tetra version` and `./t version` report `v0.2.0`
      via gate steps `02-version-preflight-v0-2-0-required` and
      `03-short-alias-version-parity`.
- [x] `scripts/release/v0_2_0/gate.sh` runs to completion on the exact release
      commit and writes a fresh report archive:
      `reports/v0_2_0_candidate_20260428-rerun-210701`.
- [x] `docs/release-notes/v0_2_0.md` contains reviewed release notes for the
      exact candidate commit.
- [x] Security signoff exists for the exact release commit under review:
      `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/security-review.md`
      and `reports/v0_2_0_security_review_20260428-180419.md`.

## Required Command Evidence

- [x] `go test ./compiler/... ./cli/... ./tools/... -count=1`
      (`logs/04-go-test-packages.log`)
- [x] Epic 06 safety gate:

  ```sh
  SAFETY_RE="Effect|Uses|Capability|Unsafe|Ownership|Borrow|Consume"
  SAFETY_RE="${SAFETY_RE}|Inout|Island|Region|Privacy|Budget"
  go test ./compiler/... -run "$SAFETY_RE" -count=1
  ```

- [x] Formatter closure gate:

  ```sh
  ./tetra fmt --check examples lib __rt compiler/selfhostrt && \
    go test ./compiler/... ./cli/... -run "Format|Formatter|Comment" -count=1
  ```

      (gate log records this formatter closure gate; package tests and
      formatter check passed in `logs/04-go-test-packages.log` and
      `logs/10-formatter-check.log`)
- [x] `bash scripts/ci/test.sh`
      (`artifacts/test-all/logs/02-repo-test-script.log`)
- [x] `bash scripts/dev/bootstrap.sh`
      (`logs/01-bootstrap-tetra-binaries.log`)
- [x] `bash scripts/ci/test-all.sh --quick --report-dir <dir>/test-all-quick`
- [x] `bash scripts/ci/test-all.sh --full --keep-going --report-dir <dir>/test-all`
      (`logs/05-full-stabilization-wrapper.log`,
      `artifacts/test-all/summary.json`)
- [x] `./tetra fmt --check examples lib __rt compiler/selfhostrt`
      (`logs/10-formatter-check.log`)
- [x] `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`
      (`logs/06-flow-only-source-scan.log`)
- [x] `./tetra targets --format=json > <dir>/artifacts/targets.json`
      (`logs/07-targets-report-validation.log`, `artifacts/targets.json`)
- [x] `go run ./tools/cmd/validate-targets --report <dir>/artifacts/targets.json`
      (`logs/07-targets-report-validation.log`)
- [x] `./tetra doctor --format=json > <dir>/artifacts/doctor.json`
      (`logs/08-doctor-report-validation.log`, `artifacts/doctor.json`)
- [x] `go run ./tools/cmd/validate-doctor --report <dir>/artifacts/doctor.json`
      (`logs/08-doctor-report-validation.log`)
- [x] `./tetra test --report=json examples > <dir>/artifacts/tetra-test-report.json`
      (`logs/11-tetra-test-examples-json.log`, `artifacts/tetra-test-report.json`)
- [x] `go run ./tools/cmd/validate-test-report --report <dir>/artifacts/tetra-test-report.json`
      (`logs/11-tetra-test-examples-json.log`)
- [x] `./tetra smoke --list --format=json > <dir>/artifacts/smoke-list.json`
      (`logs/16-smoke-list-validation.log`, `artifacts/smoke-list.json`)
- [x] Smoke list report validation:

  ```sh
  go run ./tools/cmd/validate-smoke-list \
    --report <dir>/artifacts/smoke-list.json \
    --examples-root examples
  ```

      (`logs/16-smoke-list-validation.log`)
- [x] `go run ./tools/cmd/gen-docs examples > <dir>/artifacts/api-docs.md`
- [x] `go run ./tools/cmd/validate-api-docs --docs <dir>/artifacts/api-docs.md`
- [x] `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
      (`logs/13-docs-verification-and-doctests.log`)
- [x] `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
      (`logs/12-docs-manifest-regenerate-validate.log`)
- [x] `go run ./tools/cmd/validate-release-state --format=text --report-dir <dir>`
      (`logs/34-release-state-audit.log`, `artifacts/release-state.txt`)
- [x] `go run ./tools/cmd/validate-artifact-hashes --manifest <dir>/artifacts/artifact-hashes.json`
      (`logs/32-artifact-hash-manifest.log`,
      `logs/35-artifact-hash-manifest-refresh.log`,
      `artifacts/artifact-hashes.json`)
- [x] `bash scripts/release/v1_0/security-review.sh --signoff <dir>/artifacts/security-review.md`
      (covered by gate step `security review signoff`,
      `logs/25-security-review-signoff.log`)
- [x] Reproducible build proof:

  ```sh
  bash scripts/release/v1_0/reproducible-build.sh \
    --report <dir>/artifacts/reproducible-build.json
  ```

      (covered by gate step `reproducible build proof`,
      `logs/29-reproducible-build-proof.log`)
- [x] Binary size thresholds:

  ```sh
  bash scripts/release/v1_0/binary-size.sh \
    --report <dir>/artifacts/binary-size-thresholds.json
  ```

      (covered by gate step `binary size thresholds`,
      `logs/28-binary-size-thresholds.log`)
- [x] Performance regression validation:

  ```sh
  go run ./tools/cmd/validate-performance-report \
    --report <dir>/artifacts/performance-regression.json
  ```

      (`logs/27-performance-regression-evidence.log`)
- [x] `git diff --check`

## Required Artifact Paths

- [x] `<report-dir>/summary.json`
- [x] `<report-dir>/summary.md`
- [x] `<report-dir>/logs/*.log`
- [x] `<report-dir>/artifacts/release-state.json`
- [x] `<report-dir>/artifacts/release-state.txt`
- [x] `<report-dir>/artifacts/artifact-hashes.json`
- [x] `<report-dir>/artifacts/known_issues.md`
- [x] `<report-dir>/artifacts/security-review.md`
- [x] `<report-dir>/artifacts/reproducible-build.json`
- [x] `<report-dir>/artifacts/binary-size-thresholds.json`
- [x] `<report-dir>/artifacts/performance-regression.json`
- [x] `<report-dir>/artifacts/targets.json`
- [x] `<report-dir>/artifacts/doctor.json`
- [x] `<report-dir>/artifacts/tetra-test-report.json`
- [x] `<report-dir>/artifacts/smoke-list.json`
- [x] `<report-dir>/artifacts/api-docs.md`
- [x] `<report-dir>/artifacts/api-diff/api-docs.md`
- [x] `<report-dir>/artifacts/api-diff/api-diff.json`
- [x] `<report-dir>/artifacts/tetra-docs.md`
- [x] `<report-dir>/artifacts/test-all/summary.json`
- [x] `<report-dir>/artifacts/test-all/summary.md`
- [x] `<report-dir>/test-all-quick/summary.json`
- [x] `<report-dir>/test-all-quick/summary.md`

## Final Verification Matrix

This matrix is the release-facing closure record for `V020-0951` through
`V020-1000`. It does not mark the TODO plan complete; it defines the concrete
checks that must exist before the handoff may claim the range is closed.

Current evidence archive for checked rows:
`reports/v0_2_0_candidate_20260428-rerun-210701`.

### Go Workspace Tests

- Tasks: `V020-0951`..`V020-0955`
- Required command or gate step:

  ```sh
  go test ./compiler/... ./cli/... ./tools/... -count=1
  ```

- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/04-go-test-packages.log`
  - handoff command row
- Pass condition: exit `0` with no failing package.
- Failure triage: start from the first failing package or test name in the log.
  Rerun the narrow package with `-run` before rerunning the full command.
- Freshness block: missing or non-current command evidence blocks the handoff.

### Quick Scripts

- Tasks: `V020-0956`..`V020-0960`
- Required command or gate step:

  ```sh
  bash scripts/ci/test.sh
  bash scripts/dev/bootstrap.sh
  bash scripts/ci/test-all.sh --quick --report-dir <report-dir>/test-all-quick
  ```

- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/test-all/logs/02-repo-test-script.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/01-bootstrap-tetra-binaries.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/test-all-quick/summary.json`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/test-all-quick/summary.md`
- Pass condition: quick summary has `status: pass` and `failed_count: 0`.
  Bootstrap refreshes `./tetra` and `./t`.
- Failure triage: inspect the named failed quick step in `summary.md`.
  Then use its linked log.
- Freshness block: summaries from another commit, branch, or version are stale.

### Full Test Wrapper

- Tasks: `V020-0961`..`V020-0965`
- Required command or gate step:

  ```sh
  bash scripts/ci/test-all.sh --full --keep-going \
    --report-dir <report-dir>/artifacts/test-all
  ```

- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/test-all/summary.json`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/test-all/summary.md`
  - logs under the same directory
- Pass condition: summary has `mode: full`, `status: pass`, and `failed_count: 0`.
- Failure triage: use the first failed step in `summary.json`.
  Rerun that command before the full wrapper.
- Freshness block: missing summaries, failures, or copied summaries block release.

### Formatter And Flow Gates

- Tasks: `V020-0966`..`V020-0970`
- Required command or gate step:

  ```sh
  ./tetra fmt --check examples lib __rt compiler/selfhostrt
  go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
  ```

- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/10-formatter-check.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/06-flow-only-source-scan.log`
- Pass condition: formatter exits `0`; Flow-only scan exits `0`.
- Failure triage: formatter diagnostics identify the source path.
  Flow-only failures name the legacy syntax path.
- Freshness block: the handoff must cite logs from the final candidate commit.

### CLI Report Gates

- Tasks: `V020-0971`..`V020-0975`
- Required command or gate step:

  ```sh
  ./tetra targets --format=json
  go run ./tools/cmd/validate-targets
  ./tetra doctor --format=json
  go run ./tools/cmd/validate-doctor
  ./tetra test --report=json examples
  go run ./tools/cmd/validate-test-report
  ```

- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/targets.json`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/doctor.json`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/tetra-test-report.json`
- Pass condition: each JSON report validates with its paired validator.
- Failure triage: regenerate the failing JSON report.
  Then run the paired validator alone for a focused error.
- Freshness block: reports without matching validator logs are stale.

### Smoke Report Gates

- Tasks: `V020-0976`..`V020-0980`
- Required command or gate step:

  ```sh
  ./tetra smoke --list --format=json
  go run ./tools/cmd/validate-smoke-list
  ```

- Required gate step: target smoke steps from the release gate.
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/smoke-list.json`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/*-smoke*.json`
  - smoke logs
- Pass condition: smoke list validator passes.
  Each target smoke report validates or is an explicit blocker.
- Failure triage: inspect the target-specific smoke report and matching log.
- Freshness block: old WASM or cross-target reports cannot be reused.

### Docs/API Gates

- Tasks: `V020-0981`..`V020-0985`
- Required command or gate step:

  ```sh
  go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
  ```

- Required gate steps: docs manifest validation, tetra-doc output validation,
  and API diff gate.
- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/12-docs-manifest-regenerate-validate.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/13-docs-verification-and-doctests.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/14-tetra-doc-output-validation.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/26-api-diff-gate.log`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/api-diff/api-docs.md`
- Pass condition: docs verifier exits `0`; generated docs and API diff validate.
- Failure triage: fix the path named by `verify-docs`.
  For API docs, inspect generated Markdown before changing code or examples.
- Freshness block: any manifest or generated docs change requires rerunning this row.

### Release-State And Hash Gates

- Tasks: `V020-0986`..`V020-0990`
- Required command or gate step:

  ```sh
  go run ./tools/cmd/validate-release-state \
    --format=text \
    --report-dir <report-dir>
  go run ./tools/cmd/validate-artifact-hashes \
    --manifest <report-dir>/artifacts/artifact-hashes.json
  ```

- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/release-state.json`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/release-state.txt`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/artifact-hashes.json`
- Pass condition: release-state status is `pass`.
  Hash manifest validates every artifact by size and SHA256.
- Failure triage: release-state issues name stale version, dirty worktree,
  missing artifact, or freshness failure.
- Failure triage: hash failures name the mismatched artifact.
- Freshness block: this row is the stale-evidence blocker for final signoff.

### Security, Repro, And Performance Gates

- Tasks: `V020-0991`..`V020-0995`
- Required command or gate step:

  ```sh
  bash scripts/release/v1_0/security-review.sh \
    --signoff <report-dir>/artifacts/security-review.md
  bash scripts/release/v1_0/reproducible-build.sh \
    --report <report-dir>/artifacts/reproducible-build.json
  bash scripts/release/v1_0/binary-size.sh \
    --report <report-dir>/artifacts/binary-size-thresholds.json
  go run ./tools/cmd/validate-performance-report \
    --report <report-dir>/artifacts/performance-regression.json
  ```

- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/security-review.md`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/reproducible-build.json`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/binary-size-thresholds.json`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/performance-regression.json`
- Pass condition: security signoff is reviewed.
  Repro, binary-size, and performance reports validate.
- Failure triage: security failures need reviewer action.
  Repro and performance failures start from generated JSON and script logs.
- Freshness block: missing reviewer identity, commit, or report directory blocks release.

### v0.2.0 Final Gate

- Tasks: `V020-0996`..`V020-1000`
- Required command or gate step:

  ```sh
  TETRA_SECURITY_REVIEW_SIGNOFF=<path> \
    bash scripts/release/v0_2_0/gate.sh --report-dir <report-dir>
  ```

- Evidence:
  - `reports/v0_2_0_candidate_20260428-rerun-210701/summary.json`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/summary.md`
  - `reports/v0_2_0_candidate_20260428-rerun-210701/logs/*.log`
  - all required artifacts above
- Pass condition: gate summary has `status: pass` and `failed_count: 0`.
  Version preflight proves `v0.2.0`, and handoff cites the report directory.
- Failure triage: if version preflight blocks, finish version promotion first.
  Otherwise inspect the first failed step in `summary.json`.
- Freshness block: only the final release commit gate can support tag creation.

## Source Of Truth Guardrails

- [x] No checkbox is marked complete without a command or artifact path.
- [x] Historical v0.5/v0.6/v1 placeholders are not presented as current release proof.
- [x] Unsupported/planned features stay labeled as planned or deferred.
- [x] `docs/release/v0_2_0_final_handoff.md` names the exact report directory,
      command outcomes, changed files, and residual risks for the release
      commit before any tag is created.
