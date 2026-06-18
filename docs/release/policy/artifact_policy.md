# Release Artifact Policy

Status: current release artifact policy for `v0.4.0`, with future v1.0 notes.
This file defines which artifacts are tracked and which are regenerated into a
report directory.

## Artifact Inventory

### Generated Manifest

- Artifact: `docs/generated/manifest.json`.
- Tracked: yes.
- Regenerate command:
  `go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json`.
- Notes: must match the current compiler metadata.

### v1.0 Generated Snapshots

- Artifact: `docs/generated/v1_0/*`.
- Tracked: yes, only for reviewed release-prep snapshots.
- Regenerate command: future v1 gate with a fresh `<dir>` after version
  promotion.
- Notes: historical directory name retained for compatibility; do not refresh
  casually during feature work.

### API Diff Baseline

- Artifact: `docs/baselines/api-diff-baseline.v1alpha1.json`.
- Tracked: yes.
- Regenerate command: `bash scripts/release/v1_0/api-diff.sh --write-baseline`.
- Notes: baseline updates require review.

### API Diff Reports

- Artifact: API diff reports.
- Tracked: report artifact by default.
- Regenerate command: `bash scripts/release/v1_0/api-diff.sh --report-dir <dir>`.
- Notes: track only release-prep snapshots.

### Smoke Reports

- Artifact: smoke reports.
- Tracked: report artifact by default.
- Regenerate command: `./tetra smoke ... --report <path>`.
- Notes: validate before archiving.

### Web UI Smoke Report

- Artifact: web UI smoke report.
- Tracked: report artifact by default.
- Regenerate command: `bash scripts/release/v1_0/web-smoke.sh --report <path>`.
- Validation command:
  `go run ./tools/cmd/validate-web-ui-smoke --report <path>`.
- Notes: a `pass` report requires a UI-specific source, `ok:*` browser result,
  UI sidecars/DOM evidence, and `ui-event-dispatch:web-command-dispatch` in
  `runtime_trace`.
- Notes: host/browser limits are archived as validated `blocked` reports and
  remain release blockers.

### Native UI Shell Trace

- Artifact: native UI shell trace.
- Tracked: report artifact when native UI sidecars are generated.
- Regenerate command:
  `go run ./tools/cmd/validate-native-ui-smoke --report <output>.ui.shell.json`.
- Notes: validates `tetra.ui.native-shell.v1` sidecars for command-dispatch
  runtime identity, state/view evidence, event operation traces,
  post-dispatch bindings, and binding/action widgets.

### Test-All Summaries

- Artifact: test-all summaries.
- Tracked: report artifact by default.
- Regenerate command:
  `bash scripts/ci/test-all.sh --full --keep-going --report-dir <dir>`.
- Notes: keep logs with summary JSON/Markdown.

### Reproducible Build Proof

- Artifact: reproducible build proof.
- Tracked: report artifact by default.
- Regenerate command:
  `bash scripts/release/v1_0/reproducible-build.sh --report <path>`.
- Notes: required for release candidate archive.

### Security Review Signoff

- Artifact: security review signoff.
- Tracked: report artifact by default.
- Regenerate command:
  `bash scripts/release/v1_0/security-review.sh --write-template <path>`.
- Signoff command:
  `bash scripts/release/v1_0/security-review.sh --signoff <path>`.
- Notes: must name the reviewer, reviewed commit, report directory, evidence
  command results, decision for the current repository version, and residual
  risks.

### Release-State Audit

- Artifact: release-state audit.
- Tracked: report artifact by default.
- Regenerate command:
  `go run ./tools/cmd/validate-release-state --format=json --report-dir <dir>`.
- Notes: archives branch, version, git status, required artifact presence,
  manifest freshness, and last gate evidence.

### v0.4.0 Completion Audit

- Artifact: `v0.4.0` completion audit.
- Tracked: yes for the scoped v0.4.0 objective.
- Regenerate command:
  `go run ./tools/cmd/validate-v0-4-completion-audit`.
- Required flags:
  `--audit docs/release/v0_4_0_completion_audit.md --expected-status achieved`.
- Notes: validates prompt-to-artifact checklist and achieved result
  classifications.
- Notes: validates the six-column release evidence matrix, positive and
  negative test evidence, manifest/docs/report/Graphify/CI evidence keys, and
  no dirty-green pass rows.

### Known Issues

- Artifact: known issues.
- Tracked: report artifact by default, tracked only for reviewed release
  snapshots.
- Regenerate command: `bash scripts/release/v0_4_0/gate.sh --report-dir <dir>`.
- Future command: future v1 gate after promotion.
- Notes: gate writes `<dir>/artifacts/known_issues.md`.
- Notes: reviewed snapshots may be copied to
  `docs/generated/v1_0/known_issues.md` only for v1 release prep.

### Artifact Hash Manifest

- Artifact: artifact hash manifest.
- Tracked: report artifact by default, tracked only for reviewed release
  snapshots.
- Regenerate command:
  `go run ./tools/cmd/validate-artifact-hashes --write`.
- Required flags:
  `--root <dir>/artifacts --out <dir>/artifacts/artifact-hashes.json`.
- Notes: hash manifest records path, sha256, size, and JSON schema where
  present.
- Validation command:
  `go run ./tools/cmd/validate-artifact-hashes --manifest <path>`.

### v0.2.0 Release Gate Artifacts

- Artifact: `v0.2.0` release gate artifacts.
- Tracked: report artifact by default.
- Regenerate command:
  `TETRA_SECURITY_REVIEW_SIGNOFF=<path> bash scripts/release/v0_2_0/gate.sh`.
- Required flags: `--report-dir <dir>`.
- Notes: the archive must satisfy
  `docs/checklists/v0_2_0_release_gate.md#final-verification-matrix`.

### v0.2.0 Quick-Wrapper Evidence

- Artifact: `v0.2.0` quick-wrapper evidence.
- Tracked: report artifact by default.
- Regenerate command:
  `bash scripts/ci/test-all.sh --quick --report-dir <dir>/test-all-quick`.
- Notes: captured separately from the nested full gate so local iteration
  evidence and final release evidence remain distinguishable.

### v0.3.0 Release Gate Artifacts

- Artifact: `v0.3.0` release gate artifacts.
- Tracked: report artifact by default.
- Regenerate command: `bash scripts/release/v0_3_0/gate.sh --report-dir <dir>`.
- Notes: the archive must satisfy `docs/checklists/v0_3_0_release_gate.md`.

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
