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
- [x] Epic 06 safety gate: `go test ./compiler/... -run "Effect|Uses|Capability|Unsafe|Ownership|Borrow|Consume|Inout|Island|Region|Privacy|Budget" -count=1`
- [x] `./tetra fmt --check examples lib __rt compiler/selfhostrt && go test ./compiler/... ./cli/... -run "Format|Formatter|Comment" -count=1`
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
- [x] `go run ./tools/cmd/validate-smoke-list --report <dir>/artifacts/smoke-list.json --examples-root examples`
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
- [x] `bash scripts/release/v1_0/reproducible-build.sh --report <dir>/artifacts/reproducible-build.json`
      (covered by gate step `reproducible build proof`,
      `logs/29-reproducible-build-proof.log`)
- [x] `bash scripts/release/v1_0/binary-size.sh --report <dir>/artifacts/binary-size-thresholds.json`
      (covered by gate step `binary size thresholds`,
      `logs/28-binary-size-thresholds.log`)
- [x] `go run ./tools/cmd/validate-performance-report --report <dir>/artifacts/performance-regression.json`
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

| Area | Tasks | Required command or gate step | Evidence path | Pass condition | Failure triage | Freshness block |
| --- | --- | --- | --- | --- | --- | --- |
| Go workspace tests | `V020-0951`..`V020-0955` | `go test ./compiler/... ./cli/... ./tools/... -count=1` | `reports/v0_2_0_candidate_20260428-rerun-210701/logs/04-go-test-packages.log`; handoff command row | Exit `0` with no failing package. | Start from the first failing package or test name in the log; rerun the narrow package with `-run` before rerunning the full command. | Missing or non-current command evidence blocks the `v0.2.0` release handoff. |
| Quick scripts | `V020-0956`..`V020-0960` | `bash scripts/ci/test.sh`; `bash scripts/dev/bootstrap.sh`; `bash scripts/ci/test-all.sh --quick --report-dir <report-dir>/test-all-quick` | `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/test-all/logs/02-repo-test-script.log`; `reports/v0_2_0_candidate_20260428-rerun-210701/logs/01-bootstrap-tetra-binaries.log`; `reports/v0_2_0_candidate_20260428-rerun-210701/test-all-quick/summary.json`; `reports/v0_2_0_candidate_20260428-rerun-210701/test-all-quick/summary.md` | Quick wrapper summary has `status: pass` and `failed_count: 0`; bootstrap refreshes `./tetra` and `./t`. | Inspect the named failed quick step in `summary.md`, then use its linked log. | A quick summary from another commit, branch, or version is stale and blocks signoff. |
| Full test wrapper | `V020-0961`..`V020-0965` | `bash scripts/ci/test-all.sh --full --keep-going --report-dir <report-dir>/artifacts/test-all` | `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/test-all/summary.json`; `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/test-all/summary.md`; logs under the same directory | Summary has `mode: full`, `status: pass`, and `failed_count: 0`. | Use the first failed step in `summary.json`; rerun that command before the full wrapper. | Missing summary files, `failed_count > 0`, or copied historical summaries block release. |
| Formatter and Flow gates | `V020-0966`..`V020-0970` | `./tetra fmt --check examples lib __rt compiler/selfhostrt`; `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt` | `reports/v0_2_0_candidate_20260428-rerun-210701/logs/10-formatter-check.log`; `reports/v0_2_0_candidate_20260428-rerun-210701/logs/06-flow-only-source-scan.log` | Formatter exits `0`; Flow-only scan exits `0`. | Formatter diagnostics identify the source path; Flow-only failures name the legacy syntax path. | The handoff must cite logs from the final candidate commit. |
| CLI report gates | `V020-0971`..`V020-0975` | `./tetra targets --format=json`; `go run ./tools/cmd/validate-targets`; `./tetra doctor --format=json`; `go run ./tools/cmd/validate-doctor`; `./tetra test --report=json examples`; `go run ./tools/cmd/validate-test-report` | `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/targets.json`; `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/doctor.json`; `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/tetra-test-report.json` | Each JSON report validates with its paired validator. | Regenerate the failing JSON report, then run the paired validator alone for a focused error. | Reports without matching validator logs are stale. |
| Smoke report gates | `V020-0976`..`V020-0980` | `./tetra smoke --list --format=json`; `go run ./tools/cmd/validate-smoke-list`; target smoke steps from the release gate | `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/smoke-list.json`; `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/*-smoke*.json`; smoke logs | Smoke list validator passes; each target smoke report validates or is an explicit blocker. | Inspect the target-specific smoke report and the matching build/run log. | Build-only WASM or cross-target reports cannot be reused from older gate runs. |
| Docs/API gates | `V020-0981`..`V020-0985` | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`; docs manifest validation; tetra-doc output validation; API diff gate | `reports/v0_2_0_candidate_20260428-rerun-210701/logs/12-docs-manifest-regenerate-validate.log`; `reports/v0_2_0_candidate_20260428-rerun-210701/logs/13-docs-verification-and-doctests.log`; `reports/v0_2_0_candidate_20260428-rerun-210701/logs/14-tetra-doc-output-validation.log`; `reports/v0_2_0_candidate_20260428-rerun-210701/logs/26-api-diff-gate.log`; `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/api-diff/api-docs.md` | Docs verifier exits `0`; generated docs and API diff validate. | Fix the path named by `verify-docs`; for API docs, inspect the generated markdown before changing code or examples. | Any manifest or generated docs change requires rerunning this row. |
| Release-state and hash gates | `V020-0986`..`V020-0990` | `go run ./tools/cmd/validate-release-state --format=text --report-dir <report-dir>`; `go run ./tools/cmd/validate-artifact-hashes --manifest <report-dir>/artifacts/artifact-hashes.json` | `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/release-state.json`; `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/release-state.txt`; `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/artifact-hashes.json` | Release-state status is `pass`; hash manifest validates every artifact by size and SHA256. | Release-state issues name stale version, dirty worktree, missing artifact, or freshness failure; hash failures name the mismatched artifact. | This row is the canonical stale-evidence blocker for final signoff. |
| Security, repro, and performance gates | `V020-0991`..`V020-0995` | `bash scripts/release/v1_0/security-review.sh --signoff <report-dir>/artifacts/security-review.md`; `bash scripts/release/v1_0/reproducible-build.sh --report <report-dir>/artifacts/reproducible-build.json`; `bash scripts/release/v1_0/binary-size.sh --report <report-dir>/artifacts/binary-size-thresholds.json`; `go run ./tools/cmd/validate-performance-report --report <report-dir>/artifacts/performance-regression.json` | `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/security-review.md`; `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/reproducible-build.json`; `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/binary-size-thresholds.json`; `reports/v0_2_0_candidate_20260428-rerun-210701/artifacts/performance-regression.json` | Security signoff is reviewed; repro, binary-size, and performance reports validate. | Security failures need reviewer action; repro/performance failures should be triaged from their generated JSON and script log. | Missing reviewer identity, reviewed commit, or report directory blocks release. |
| v0.2.0 final gate | `V020-0996`..`V020-1000` | `TETRA_SECURITY_REVIEW_SIGNOFF=<path> bash scripts/release/v0_2_0/gate.sh --report-dir <report-dir>` | `reports/v0_2_0_candidate_20260428-rerun-210701/summary.json`; `reports/v0_2_0_candidate_20260428-rerun-210701/summary.md`; `reports/v0_2_0_candidate_20260428-rerun-210701/logs/*.log`; all required artifacts above | Gate summary has `status: pass`, `failed_count: 0`, version preflight proves `v0.2.0`, and handoff cites the report directory. | If the gate blocks at version preflight, finish version promotion first; otherwise inspect the first failed step in `summary.json`. | A final gate from any commit other than the release commit is stale and blocks tag creation. |

## Source Of Truth Guardrails

- [x] No checkbox is marked complete without a command or artifact path.
- [x] Historical v0.5/v0.6/v1 placeholders are not presented as current release proof.
- [x] Unsupported/planned features stay labeled as planned or deferred.
- [x] `docs/release/v0_2_0_final_handoff.md` names the exact report directory,
      command outcomes, changed files, and residual risks for the release
      commit before any tag is created.
