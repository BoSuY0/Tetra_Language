# Release Artifact Policy

Status: current release artifact policy for `v0.1.3`, with future v1.0 notes.
This file defines which artifacts are tracked and which are regenerated into a
report directory.

## Artifact Inventory

| Artifact | Tracked? | Regenerate command | Notes |
| --- | --- | --- | --- |
| `docs/generated/manifest.json` | Yes | `go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json` | Must match the current compiler metadata. |
| `docs/generated/v1_0/*` | Yes only for reviewed release-prep snapshots | `bash scripts/release_v0_1_3_gate.sh --report-dir <dir>` then copy reviewed outputs | Historical directory name retained for compatibility; do not refresh casually during feature work. |
| `docs/baselines/api-diff-baseline.v1alpha1.json` | Yes | `bash scripts/release_v1_0_api_diff.sh --write-baseline` | Baseline updates require review. |
| API diff reports | Report artifact by default | `bash scripts/release_v1_0_api_diff.sh --report-dir <dir>` | Track only release-prep snapshots. |
| Smoke reports | Report artifact by default | `./tetra smoke ... --report <path>` | Validate before archiving. |
| Web UI smoke report | Report artifact by default | `bash scripts/release_v1_0_web_smoke.sh --report <path>` then `go run ./tools/cmd/validate-web-ui-smoke --report <path>` | A `pass` report requires a UI-specific source and `ok:*` browser result. Host/browser limits are archived as validated `blocked` reports and remain release blockers. |
| Test-all summaries | Report artifact by default | `bash scripts/test_all.sh --full --keep-going --report-dir <dir>` | Keep logs with summary JSON/Markdown. |
| Reproducible build proof | Report artifact by default | `bash scripts/release_v1_0_repro.sh --report <path>` | Required for release candidate archive. |
| Security review signoff | Report artifact by default | `bash scripts/release_v1_0_security_review.sh --write-template <path>` then `bash scripts/release_v1_0_security_review.sh --signoff <path>` | Must name the reviewer, reviewed commit, report directory, evidence command results, decision for the current repository version, and residual risks. |
| Release-state audit | Report artifact by default | `go run ./tools/cmd/validate-release-state --format=json --report-dir <dir>` | Archives branch, version, git status, required artifact presence, manifest freshness, and last gate evidence. |
| Known issues | Report artifact by default, tracked only for reviewed release snapshots | `bash scripts/release_v0_1_3_gate.sh --report-dir <dir>` | Gate writes `<dir>/artifacts/known_issues.md`; reviewed snapshots may be copied to `docs/generated/v1_0/known_issues.md`. |
| Artifact hash manifest | Report artifact by default, tracked only for reviewed release snapshots | `go run ./tools/cmd/validate-artifact-hashes --write --root <dir>/artifacts --out <dir>/artifacts/artifact-hashes.json` | Hash manifest records path, sha256, size, and JSON schema where present; validate with `go run ./tools/cmd/validate-artifact-hashes --manifest <path>`. |

## Timestamp Policy

Tracked generated snapshots must not churn solely because a command was rerun.
Reports that need run time evidence should rely on the surrounding gate summary
timestamps, not embedded wall-clock fields. The reproducible build proof follows
this policy by omitting `generated_at` and recording `timestamp_policy` instead.

## Diff Check Policy

Run `git diff --check` after edits and before handoff. If generated artifacts
change, include the generator command and validation command in the evidence.

## Archive Location

Release candidate artifacts belong under a dedicated report directory first,
for example `/tmp/tetra-v0_1_3-release-gate`. Only reviewed snapshots should
be copied into `docs/generated/v1_0/`.

## Checklist Link

`docs/checklists/v0_1_3_release_gate.md` is the authoritative checklist for
which artifact evidence is mandatory before the current `v0.1.3` release tag.
`docs/checklists/v1_0_release_gate.md` is only a future placeholder until the
project intentionally promotes the version line to `v1.0.x`.
