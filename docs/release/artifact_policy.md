# Release Artifact Policy

Status: current release artifact policy for `v0.4.0`, with future v1.0 notes.
This file defines which artifacts are tracked and which are regenerated into a
report directory.

## Artifact Inventory

| Artifact | Tracked? | Regenerate command | Notes |
| --- | --- | --- | --- |
| `docs/generated/manifest.json` | Yes | `go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json` | Must match the current compiler metadata. |
| `docs/generated/v1_0/*` | Yes only for reviewed release-prep snapshots | Future v1 gate with a fresh `<dir>` after version promotion | Historical directory name retained for compatibility; do not refresh casually during feature work. |
| `docs/baselines/api-diff-baseline.v1alpha1.json` | Yes | `bash scripts/release/v1_0/api-diff.sh --write-baseline` | Baseline updates require review. |
| API diff reports | Report artifact by default | `bash scripts/release/v1_0/api-diff.sh --report-dir <dir>` | Track only release-prep snapshots. |
| Smoke reports | Report artifact by default | `./tetra smoke ... --report <path>` | Validate before archiving. |
| Web UI smoke report | Report artifact by default | `bash scripts/release/v1_0/web-smoke.sh --report <path>` then `go run ./tools/cmd/validate-web-ui-smoke --report <path>` | A `pass` report requires a UI-specific source, `ok:*` browser result, UI sidecars/DOM evidence, and `ui-event-dispatch:web-command-dispatch` in `runtime_trace`. Host/browser limits are archived as validated `blocked` reports and remain release blockers. |
| Native UI shell trace | Report artifact when native UI sidecars are generated | `go run ./tools/cmd/validate-native-ui-smoke --report <output>.ui.shell.json` | Validates `tetra.ui.native-shell.v1` sidecars for command-dispatch runtime identity, state/view evidence, event operation traces, post-dispatch bindings, and binding/action widgets. |
| Test-all summaries | Report artifact by default | `bash scripts/ci/test-all.sh --full --keep-going --report-dir <dir>` | Keep logs with summary JSON/Markdown. |
| Reproducible build proof | Report artifact by default | `bash scripts/release/v1_0/reproducible-build.sh --report <path>` | Required for release candidate archive. |
| Security review signoff | Report artifact by default | `bash scripts/release/v1_0/security-review.sh --write-template <path>` then `bash scripts/release/v1_0/security-review.sh --signoff <path>` | Must name the reviewer, reviewed commit, report directory, evidence command results, decision for the current repository version, and residual risks. |
| Release-state audit | Report artifact by default | `go run ./tools/cmd/validate-release-state --format=json --report-dir <dir>` | Archives branch, version, git status, required artifact presence, manifest freshness, and last gate evidence. |
| `v0.4.0` completion audit | Yes for the scoped v0.4.0 objective | `go run ./tools/cmd/validate-v0-4-completion-audit --audit docs/release/v0_4_0_completion_audit.md --expected-status achieved` | Validates the prompt-to-artifact checklist, achieved result classifications, and evidence summary for the current scoped audit. |
| Known issues | Report artifact by default, tracked only for reviewed release snapshots | `bash scripts/release/v0_4_0/gate.sh --report-dir <dir>` for the current line; future v1 gate after promotion | Gate writes `<dir>/artifacts/known_issues.md`; reviewed snapshots may be copied to `docs/generated/v1_0/known_issues.md` only for v1 release prep. |
| Artifact hash manifest | Report artifact by default, tracked only for reviewed release snapshots | `go run ./tools/cmd/validate-artifact-hashes --write --root <dir>/artifacts --out <dir>/artifacts/artifact-hashes.json` | Hash manifest records path, sha256, size, and JSON schema where present; validate with `go run ./tools/cmd/validate-artifact-hashes --manifest <path>`. |
| `v0.2.0` release gate artifacts | Report artifact by default | `TETRA_SECURITY_REVIEW_SIGNOFF=<path> bash scripts/release/v0_2_0/gate.sh --report-dir <dir>` | The archive must satisfy `docs/checklists/v0_2_0_release_gate.md#final-verification-matrix`. |
| `v0.2.0` quick-wrapper evidence | Report artifact by default | `bash scripts/ci/test-all.sh --quick --report-dir <dir>/test-all-quick` | Captured separately from the nested full gate so local iteration evidence and final release evidence remain distinguishable. |
| `v0.3.0` release gate artifacts | Report artifact by default | `bash scripts/release/v0_3_0/gate.sh --report-dir <dir>` | The archive must satisfy `docs/checklists/v0_3_0_release_gate.md`. |

## Timestamp Policy

Tracked generated snapshots must not churn solely because a command was rerun.
Reports that need run time evidence should rely on the surrounding gate summary
timestamps, not embedded wall-clock fields. The reproducible build proof follows
this policy by omitting `generated_at` and recording `timestamp_policy` instead.

## Diff Check Policy

Run `git diff --check` after edits and before handoff. If generated artifacts
change, include the generator command and validation command in the evidence.

## Docs QA Notes

Release-facing docs must not contain stale release claims. A document that uses
phrases such as current release line, supported today, stable, production, or
complete must either name the current `v0.4.0` surface or link to
`docs/spec/current_supported_surface.md`.

Evidence dates are meaningful only together with a version, commit, and report
directory. A dated note without those bindings is an audit breadcrumb, not
release proof.

## Archive Location

Release candidate artifacts belong under a dedicated report directory first,
for example `/tmp/tetra-v0_4_0-release-gate`. Only reviewed snapshots should
be copied into `docs/generated/v1_0/`.

## Checklist Link

`docs/checklists/v0_4_0_release_gate.md` is the authoritative checklist for
which artifact evidence is mandatory before the current `v0.4.0` release tag.
`docs/checklists/v0_3_0_release_gate.md` remains the archived checklist for the
older `v0.3.0` tag.
`docs/checklists/v0_2_0_release_gate.md` remains the archived checklist for the
older `v0.2.0` tag.
`docs/checklists/v0_1_3_release_gate.md` remains the archived checklist for the
older `v0.1.3` tag.
`docs/checklists/v1_0_release_gate.md` is only a future placeholder until the
project intentionally promotes the version line to `v1.0.x`.
