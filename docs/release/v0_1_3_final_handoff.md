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
- Canonical release archive for the prep branch:
  `reports/codex-v0_1_3-post-bump-release-gate-2`.
- Tracked compatibility evidence snapshot: `docs/generated/v1_0`. The
  directory name is retained for validator compatibility; exact-commit
  signoff artifacts remain in the release archive.

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

- `scripts/release_v0_1_3_gate.sh` archives the external `security-review.md`
  signoff into the release evidence artifacts before hashing the archive.
- `scripts/release_v1_0_security_review.sh` validates the signoff against the
  current repository version and exact commit under review.
- `tools/cmd/validate-release-state` now rejects stale release summaries with
  fewer than `33` steps and validates the tracked artifact hash manifest.
- `docs/generated/v1_0` is a tracked compatibility snapshot of reviewed
  release evidence. The exact final signoff and release-state audit must be
  preserved from the report directory produced for the tagged commit.
- `.gitignore` excludes root-level native UI smoke sidecars and local Codex
  scratch files from release commits.

## Remaining Release Action

The code is release-gate clean except for the intentionally external security
review signoff when no `TETRA_SECURITY_REVIEW_SIGNOFF` is supplied. Final
tagging requires a fresh signoff for the exact commit, rerunning
`scripts/release_v0_1_3_gate.sh` with that signoff, and preserving the report
archive.
