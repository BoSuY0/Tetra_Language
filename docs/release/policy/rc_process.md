# v1.0 Release Candidate Process

Status: future release process policy. The current public release is `v0.4.0`;
no `v1.0.0` release candidate may be created while mandatory scope remains
blocked.

For the current minor line, use `docs/release/v0_4_0_final_handoff.md` and
`docs/checklists/v0_4_0_release_gate.md`.
Keep the same evidence discipline: no release-candidate checkbox is valid
without artifacts from the exact branch state under review.

## RC Entry Criteria

- `docs/spec/v1_scope.md` has no mandatory open blockers.
- `docs/checklists/v1_0_release_gate.md` has been replaced with a real v1.0
  checklist instead of the current placeholder.
- `./tetra version` and `./t version` report the intended `v1.0.0-rcN` or
  release-candidate branch version policy.
- The future v1.0 gate reaches all mandatory steps and records the result.

## Feature Freeze

After RC entry, only release-blocking fixes, documentation corrections,
deterministic artifact regeneration, and approved test updates may land.

## Scope Drift Policy

Scope changes after RC entry are blocked by default. A change may proceed only
when it has all of the following in the same branch state:

- A `docs/spec/v1_scope.md` delta that names the feature, decision, evidence
  command, and artifact path.
- A matching `docs/checklists/v1_0_release_gate.md` row or explicit post-v1
  deferral.
- Fresh verification output in the release report directory.
- Reviewer approval recorded in the final handoff.

If any item is missing, keep the feature experimental/planned/post-v1 and do
not treat examples, generated docs, or old reports as release proof.

## Experimental User Docs Policy

User docs may mention experimental slices only when the text labels them as
experimental, planned, build-only, reporting-only, or post-v1 in the same
paragraph or table row. Preview docs must link back to
`docs/spec/current_supported_surface.md` and must not describe planned behavior
as stable support.

## Allowed RC Changes

- Fixes for failing mandatory gates.
- Documentation updates that clarify known limitations.
- Release-script fixes that improve evidence without skipping required work.
- Artifact regeneration from reviewed commands.

## Rollback Plan

If a release candidate exposes a blocker, mark the candidate rejected in the
known issues list, keep its artifact archive, revert or fix the offending
change through normal review, and rerun the full release gate for the next RC.

## Known Limitations Format

Each limitation needs: title, affected component, user impact, workaround,
release-blocker status, owner, and evidence link.

## Evidence Archive

Archive the release gate summary, test-all summary, API diff report, WASI/web
smoke reports, reproducible build proof, and any platform-specific logs in the
same report directory.

## Minimum Reproducibility Checks

Before a release candidate is approved, the report directory must contain these
same-commit checks:

| Check | Required evidence |
| --- | --- |
| Commit identity | `git rev-parse HEAD` recorded in the gate summary, handoff, security signoff, release-state report, and artifact-hash manifest. |
| Version identity | `./tetra version` and `./t version` output recorded before the gate runs. |
| Deterministic generated docs | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` exit code and log. |
| Diagnostic contract | `go test ./tools/cmd/validate-diagnostic/... -count=1` exit code and package summary. |
| Reproducible builds | `bash scripts/release/v1_0/reproducible-build.sh --report REPORT_DIR/artifacts/reproducible-build.json` plus validator result when available. |
| Artifact hashes | `go run ./tools/cmd/validate-artifact-hashes --manifest REPORT_DIR/artifacts/artifact-hashes.json` exit code and missing-artifact count. |
| Flake reruns | Exact package/test rerun commands from `docs/testing/fuzz_property_stress.md`, with results and owner. |

If any field points at a different commit, version, report directory, or
generated artifact state, reject the candidate and rerun the affected gate
segment.

## No Stale Evidence Reuse

Evidence is stale when it was collected from a different commit, branch,
version, report directory, target, or generated artifact state. Anti-patterns:

- Copying a passing older release-gate row into a future `v1.0.0` handoff.
- Citing `docs/generated/manifest.json` after changing compiler metadata but
  before rerunning `go run ./tools/cmd/gen-manifest`.
- Treating a build-only WASM smoke report as browser or WASI runner execution.
- Marking a checklist row complete because a matching command passed in an older
  report archive.

## Signoff

Signoff requires a current release gate report, a reviewed artifact diff, an
updated known issues list, and explicit confirmation that no required checklist
item was checked without evidence.
