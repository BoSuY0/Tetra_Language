# Tetra v1.0 Release Cut Guide

Status: future maintainer runbook for non-interactive v1.0 release candidates
and v1.0.x patch releases. This guide is not executable for the current
`v0.4.0` profile, or for separately gated post-v0.4 production evidence, until
the repository version is intentionally promoted to `v1.0.x` and the v1 gate
passes version preflight.

Canonical scope: `docs/spec/v1_scope.md`.
Release gate: `scripts/release/v1_0/gate.sh`.
Artifact policy: `docs/release/artifact_policy.md`.

## Prepare Branch

```bash
git fetch origin
git switch main
git pull --ff-only origin main
git switch -c release/v1.0.0-rc1
bash scripts/dev/bootstrap.sh
./tetra version
./t version
```

Expected version output for the final v1.0.0 branch:

```text
v1.0.0
```

## Generate Evidence

```bash
report_dir=/tmp/tetra-v1.0.0-rc1-gate
rm -rf "$report_dir"
GOCACHE=/tmp/tetra-go-build \
  TETRA_SECURITY_REVIEW_SIGNOFF=docs/generated/v1_0/security-review.md \
  bash scripts/release/v1_0/gate.sh --report-dir "$report_dir"
```

Required archive entry points:

```text
$report_dir/summary.json
$report_dir/summary.md
$report_dir/artifacts/release-state.json
$report_dir/artifacts/known_issues.md
$report_dir/artifacts/artifact-hashes.json
```

Validate integrity before tagging:

```bash
GOCACHE=/tmp/tetra-go-build \
  go run ./tools/cmd/validate-artifact-hashes \
  --manifest "$report_dir/artifacts/artifact-hashes.json"
GOCACHE=/tmp/tetra-go-build \
  go run ./tools/cmd/validate-release-state \
  --format=text \
  --report-dir "$report_dir"
git diff --check
```

## Tag Release Candidate

```bash
git status --short
git tag -a v1.0.0-rc1 -m "Tetra v1.0.0 release candidate 1"
git push origin release/v1.0.0-rc1
git push origin v1.0.0-rc1
```

## Cut Final Release

```bash
git switch release/v1.0.0-rc1
git status --short
git tag -a v1.0.0 -m "Tetra v1.0.0"
git push origin v1.0.0
```

## Roll Back A Candidate Tag

Use this only before announcing the candidate externally.

```bash
git tag -d v1.0.0-rc1
git push origin :refs/tags/v1.0.0-rc1
```

## Create Patch Branch

```bash
git fetch origin --tags
git switch -c release/v1.0.x v1.0.0
bash scripts/dev/bootstrap.sh
./tetra version
```

Patch branches must regenerate the full release evidence archive and update
`docs/generated/v1_0/known_issues.md` only after reviewer approval.
