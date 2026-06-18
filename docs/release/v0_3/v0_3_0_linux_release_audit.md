# Tetra v0.3.0 Linux-Only Release Audit

Status: Linux-only `v0.3.0` release profile audit.

This audit intentionally excludes macOS and Windows runtime execution evidence.
It supersedes the earlier cross-platform release-gate blocker for the current
Linux-only objective, but it does not claim a clean tag-ready release commit.

## Objective

Ship the current Tetra `v0.3.0` profile for Linux first, without Windows or Mac.

Concrete completion for this Linux-only profile means:

- `./tetra version` and `./t version` report `v0.3.0`.
- Linux native runtime smoke executes on `host: linux-x64`.
- Stabilization passes for the current `v0.3.0` profile.
- Docs verification and diff hygiene pass after release-status updates.
- A packaged Linux/local evidence bundle exists with hashes.
- macOS and Windows runtime execution evidence are explicitly out of scope.

## Prompt-To-Artifact Checklist

| Requirement | Evidence | Status |
| --- | --- | --- |
| Version identity | `./tetra version` and `./t version` both printed `v0.3.0`. | pass |
| Linux runtime execution | `./tetra smoke --target linux-x64 --run=true --report /tmp/tetra-v0.3-linux-runtime-smoke.json` exited 0; `62/62` passed on `host: linux-x64`; all non-unsupported runtime cases ran. | pass |
| Linux smoke report validation | `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report /tmp/tetra-v0.3-linux-runtime-smoke.json` exited 0. | pass |
| Stabilization | `/tmp/tetra-v0.3-stabilization-audit-rerun/summary.json` reports `status: pass`, `step_count: 38`, `failed_count: 0`, and `release_version: v0.3.0`. | pass |
| Packaged Linux/local evidence | `reports/release-v0.3.0-local-candidate.tar.gz` exists with detached SHA256 at `reports/release-v0.3.0-local-candidate.tar.gz.sha256`; `sha256sum -c` reports `OK`. | pass |
| Bundle internal hashes | `reports/release-v0.3.0-local-candidate/artifact-hashes.json` verified 7 artifacts. | pass |
| Docs verification | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` exited 0 after Linux-only status updates. | pass |
| Diff hygiene | `git diff --check` exited 0 after Linux-only status updates. | pass |
| macOS runtime evidence | Explicitly out of scope for this Linux-only objective. | out-of-scope |
| Windows runtime evidence | Explicitly out of scope for this Linux-only objective. | out-of-scope |

## Remaining Non-Linux / Tag-Ready Caveats

- The cross-platform `scripts/release/v0_3_0/gate.sh` remains blocked because
  it still requires macOS and Windows native runtime evidence.
- The current worktree is not clean, so this audit must not be treated as a
  clean tag-ready release commit.
- A future cross-platform release can resume from
  `docs/release/v0_3_0_unblock_packet.md`.

## Conclusion

For the current objective, Tetra `v0.3.0` is ready as a Linux-only release
profile with packaged evidence. It is not a cross-platform or clean tag-ready
release.
