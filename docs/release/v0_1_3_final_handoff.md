# Tetra v0.1.3 Final Handoff

Date: 2026-04-27
Branch: `main`
Version: `v0.1.3`

## Release State

- Stabilization backlog: no open checkboxes in
  `docs/plans/2026-04-27-tetra-real-stabilization-agent-backlog.md`.
- v1.0 TODO plan: no open checkboxes in
  `docs/plans/2026-04-27-tetra-v0_1-to-v1_0-full-todo.md`.
- Release checklist: no open checkboxes in
  `docs/checklists/v0_1_3_release_gate.md`.
- Canonical tracked evidence snapshot: `docs/generated/v1_0`, mirrored from
  `reports/codex-v0_1_3-post-bump-release-gate-2`.
- Final verification release gate:
  `reports/codex-v0_1_3-post-bump-release-gate-2`.

## Fresh Verification

- `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`:
  pass.
- `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`:
  pass.
- `go run ./tools/cmd/validate-artifact-hashes --manifest docs/generated/v1_0/artifact-hashes.json`:
  pass.
- `go test ./compiler/... ./cli/... ./tools/... -count=1`:
  pass.
- `bash scripts/test_all.sh --full --report-dir reports/codex-v0_1_3-post-bump-full-2`:
  pass, 23 full checks.
- `TETRA_SECURITY_REVIEW_SIGNOFF=reports/codex-current-security-review.md bash scripts/release_v0_1_3_gate.sh --report-dir reports/codex-v0_1_3-post-bump-release-gate-2`:
  pass, 33 release-gate checks.
- `go run ./tools/cmd/validate-release-state --format=text --report-dir reports/codex-v0_1_3-post-bump-release-gate-2`:
  pass, `36` required artifacts, `0` missing artifacts, artifact hash manifest
  valid.
- `git diff --check`: pass.

## Integration Notes

- `scripts/release_v0_1_3_gate.sh` now archives the external
  `security-review.md` signoff into the release evidence artifacts before
  hashing the archive.
- `tools/cmd/validate-release-state` now rejects stale release summaries with
  fewer than `33` steps and validates the tracked artifact hash manifest.
- `docs/generated/v1_0` now mirrors the release evidence artifact set, not only
  a small subset of summary files.
- `.gitignore` excludes root-level native UI smoke sidecars and local Codex
  scratch files from release commits.

## Remaining Release Action

The code and evidence are release-gate clean for the release-prep branch
state. Final tagging still requires rerunning `scripts/release_v0_1_3_gate.sh`
on the exact tagged commit and preserving its artifact archive.
