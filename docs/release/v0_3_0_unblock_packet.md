# Tetra v0.3.0 Release Unblock Packet

Status: actionable packet for the remaining non-Linux release blockers.

This file records the exact artifacts needed to turn the current blocked
`v0.3.0` audit into a release evidence pass. It does not approve the release by
itself.

## Current Blocker Summary

Fresh local evidence already collected on 2026-05-04:

- `reports/release-v0.3.0-local-candidate/summary.md`: repo-local local
  candidate evidence bundle.
- `/tmp/tetra-v0.3-linux-runtime-smoke.json`: `pass`, 62 Linux runtime cases,
  `host: linux-x64`, `ran: true`.
- `/tmp/tetra-v0.3-stabilization-audit-rerun/summary.md`: `pass`, 38 checks,
  0 failures.
- `/tmp/tetra-v0.3-gate-audit-crossbuild-runtime-blocked/summary.md`:
  identity-matched `v0.3.0` gate report, but `blocked`.

The blocked gate confirms:

- Gate identity is correct:
  `tetra.release.v0_3_0.gate-report.v1`.
- The supplied `docs/generated/v1_0/macos-smoke.json` and
  `docs/generated/v1_0/windows-smoke.json` are not valid runtime execution
  evidence because they were produced on `host: linux-x64`.
- Release-state validation still requires native `--run=true` reports and a
  same-run security review signoff.

## Native Runtime Reports

Run these commands from the same Git commit intended for the release.

On a macOS host or CI macOS runner:

```sh
bash scripts/dev/bootstrap.sh
./tetra version
./t version
./tetra smoke --target macos-x64 --run=true --report /tmp/tetra-v0.3-macos-runtime-smoke.json
```

On a Windows host or CI Windows runner:

```sh
bash scripts/dev/bootstrap.sh
./tetra version
./t version
./tetra smoke --target windows-x64 --run=true --report /tmp/tetra-v0.3-windows-runtime-smoke.json
```

Each report must satisfy these checks:

- `target` matches the requested target.
- `host` equals the same target.
- `version` is `v0.3.0`.
- `git_head` matches the release candidate commit short hash.
- `build_only` is absent or `false`.
- `unsupported` is absent or `false`.
- Required actor/task smoke cases, including `actors_pingpong`,
  `actor_sleep_pingpong`, and `task_smoke`, have `ran: true`, `pass: true`,
  and matching `actual_exit` / `expected_exit`.

## GitHub CI Runtime Artifact Path

The repository already has `.github/workflows/ci.yml` configured to run smoke
on `ubuntu-latest`, `windows-latest`, and `macos-latest`, then upload artifacts
named:

- `tetra-v0.3.0-<git-sha>-smoke-macOS`
- `tetra-v0.3.0-<git-sha>-smoke-Windows`

When GitHub authentication is available, the native runtime reports can be
collected with:

```sh
gh auth status
gh workflow run ci.yml --ref <release-branch-or-sha>
gh run list --workflow ci.yml --limit 10
gh run watch <run-id>
gh run download <run-id> \
  --pattern 'tetra-v0.3.0-*-smoke-macOS' \
  --pattern 'tetra-v0.3.0-*-smoke-Windows' \
  --dir reports/ci-smoke-runtime
```

Expected downloaded files:

```text
reports/ci-smoke-runtime/tetra-v0.3.0-<git-sha>-smoke-macOS/smoke_macos-x64.json
reports/ci-smoke-runtime/tetra-v0.3.0-<git-sha>-smoke-Windows/smoke_windows-x64.json
```

Use those two JSON files as `TETRA_MACOS_RUNTIME_SMOKE_REPORT` and
`TETRA_WINDOWS_RUNTIME_SMOKE_REPORT` for the final gate.

## Security Signoff

Template command:

```sh
bash scripts/release/v0_3_0/security-review.sh --write-template /tmp/tetra-v0.3-security-review.md
```

For the final gate run, the signoff must be a named human approval for the same
candidate and must include:

- `Reviewer: <name and contact>`
- `Reviewed commit: <full commit sha>`
- `Report directory: <final release gate report directory>`
- `Decision: approved for v0.3.0 release`
- A `## Artifact Hashes` section. The gate rewrites same-run canonical hashes
  for `summary.json`, `artifact-hashes.json`, and
  `artifacts/release-state.json` before archiving the final signoff.

The final archived signoff is validated with:

```sh
bash scripts/release/v0_3_0/security-review.sh --signoff <report-dir>/artifacts/security-review.md
```

## Final Evidence Gate

After native runtime reports and the human signoff exist:

```sh
TETRA_MACOS_RUNTIME_SMOKE_REPORT=/tmp/tetra-v0.3-macos-runtime-smoke.json \
TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=/tmp/tetra-v0.3-windows-runtime-smoke.json \
TETRA_SECURITY_REVIEW_SIGNOFF=/tmp/tetra-v0.3-security-review.md \
bash scripts/release/v0_3_0/gate.sh --report-dir reports/release-v0.3.0-gate
```

The release is tag-ready only after the evidence pass is current for the
intended tag commit and this clean-worktree gate also passes:

```sh
bash scripts/release/v0_3_0/gate.sh --report-dir reports/release-v0.3.0-gate-clean --require-clean
git status --porcelain --untracked-files=all
```

The `git status` command must print no entries.
