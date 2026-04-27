# v1.0 Release Candidate Process

Status: future release process policy. The current public release is `v0.1.1`;
no `v1.0.0` release candidate may be created while mandatory scope remains
blocked.

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

## Signoff

Signoff requires a current release gate report, a reviewed artifact diff, an
updated known issues list, and explicit confirmation that no required checklist
item was checked without evidence.
