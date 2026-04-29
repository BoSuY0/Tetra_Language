# Tetra v0.2.0 Release Cut Guide

Status: current runbook for the `v0.2.0` release line.

Use this guide when cutting or validating a `v0.2.0` candidate. Version
promotion is complete only when `./tetra version` and `./t version` return
`v0.2.0` from freshly bootstrapped binaries.

## Prepare Branch

```bash
git fetch origin
git switch main
git pull --ff-only origin main
git switch -c release/v0.2.0-rc1
bash scripts/bootstrap.sh
./tetra version
./t version
```

Expected output:

```text
v0.2.0
```

## Generate Evidence

```bash
report_dir=/tmp/tetra-v0.2.0-rc1-gate
rm -rf "$report_dir"
TETRA_SECURITY_REVIEW_SIGNOFF=<path-to-signoff> \
  bash scripts/release_v0_2_0_gate.sh --report-dir "$report_dir"
```

The gate must produce:

- `$report_dir/summary.json` and `$report_dir/summary.md`
- `$report_dir/logs/*.log`
- `$report_dir/artifacts/test-all/summary.json`
- `$report_dir/artifacts/release-state.json`
- `$report_dir/artifacts/release-state.txt`
- `$report_dir/artifacts/artifact-hashes.json`
- `$report_dir/artifacts/security-review.md`
- `$report_dir/artifacts/reproducible-build.json`
- `$report_dir/artifacts/binary-size-thresholds.json`

Run the quick wrapper separately when preparing the candidate so the handoff can
distinguish local iteration coverage from the full nested gate:

```bash
bash scripts/test_all.sh --quick --report-dir "$report_dir/test-all-quick"
```

## Validate Archive

```bash
go run ./tools/cmd/validate-artifact-hashes \
  --manifest "$report_dir/artifacts/artifact-hashes.json"
go run ./tools/cmd/validate-release-state \
  --format=text \
  --report-dir "$report_dir"
git diff --check
```

## Failure Triage

- If `scripts/release_v0_2_0_gate.sh` blocks at version preflight, finish the
  intentional version promotion before rerunning the gate.
- If a release gate step fails, read `$report_dir/summary.json`, open the first
  failed step log under `$report_dir/logs/`, and rerun that exact command
  before rerunning the full gate.
- If `bash scripts/test_all.sh --quick` or the nested full wrapper fails, use
  the wrapper `summary.json` to find the named failed step and matching log.
- If release-state or artifact-hash validation fails, treat the named stale,
  dirty, missing, or mismatched artifact as a release blocker.

## Freshness Rules

- All command evidence must come from the exact commit being tagged.
- Reused reports from previous report directories are stale unless the handoff
  explicitly documents why they are historical context rather than release
  proof.
- Any changed generated artifact requires rerunning docs verification,
  release-state validation, artifact-hash validation, and `git diff --check`.
- The final handoff must cite the report directory, command result, and
  residual risk for every area in
  `docs/checklists/v0_2_0_release_gate.md#final-verification-matrix`.

## Tag Candidate

```bash
git status --short
git tag -a v0.2.0-rc1 -m "Tetra v0.2.0 release candidate 1"
git push origin release/v0.2.0-rc1
git push origin v0.2.0-rc1
```

## Tag Final

```bash
git switch release/v0.2.0-rc1
git status --short
git tag -a v0.2.0 -m "Tetra v0.2.0"
git push origin v0.2.0
```

## Roll Back Candidate Tag

```bash
git tag -d v0.2.0-rc1
git push origin :refs/tags/v0.2.0-rc1
```

## Handoff

Fill `docs/release/v0_2_0_final_handoff.md` with exact command outcomes and
report archive paths from the release commit.
