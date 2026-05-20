# Tetra v0.3.0 Completion Audit

Status: active completion audit for the `v0.3.0` release objective.

This file maps the release objective to concrete evidence. It is not a release
handoff and does not make the branch tag-ready by itself.

## Objective

Finish Tetra as a complete `v0.3.0` release version.

Concrete completion means:

- The current public profile is `v0.3.0`.
- The `v0.3.0` supported surface and promoted slices are implemented,
  documented, and verified.
- The `v0.3.0` release gate has a fresh passing evidence archive.
- The release-state validator passes for that archive.
- The tag-ready pass is clean: `git status --porcelain --untracked-files=all`
  prints no entries at the intended tag commit.

## Prompt-To-Artifact Checklist

| Requirement | Required evidence | Current evidence | Status |
| --- | --- | --- | --- |
| Release version is `v0.3.0` | `./tetra version`; `./t version` | Both commands printed `v0.3.0` during audit. | pass |
| Full Go package suite | `go test ./compiler/... ./cli/... ./tools/... -count=1` | Command exited 0 during audit. | pass |
| Current truth documents name `v0.3.0` | `README.md`; `docs/spec/current_supported_surface.md`; `docs/spec/v0_3_scope.md` | Documents exist and describe `v0.3.0` as the current public profile. | pass |
| Promoted enum payload match slice | `go test ./compiler/... -run 'Enum|Match|TypedError' -count=1` | Covered by focused compiler promotion run using the combined v0.3 regex; command exited 0. | pass |
| Promoted static protocol-bound generics slice | `go test ./compiler/... -run 'Generic|Protocol|Conformance|Extension' -count=1` | Covered by focused compiler promotion run using the combined v0.3 regex; command exited 0. | pass |
| Callable Level 1 remains experimental, not promoted | `compiler/features.go`; current supported surface docs | Feature registry and docs keep Level 1 experimental and Level 2 planned. | pass |
| Ownership/resource safety remains conservative MVP | `docs/spec/current_supported_surface.md`; focused safety tests | Focused compiler promotion run covered ownership/resource/task/island regex and exited 0. | pass |
| CLI/Eco/project/workspace artifact workflows | `go test ./cli/... ./tools/... -run 'Eco|Project|Workspace|Artifact|Capsule|Lock' -count=1` | Command exited 0 during audit. | pass |
| Docs and manifest verification | `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json` | Command exited 0 during audit. | pass |
| Whitespace/diff hygiene | `git diff --check` | Command exited 0 during audit. | pass |
| Short fuzz smoke evidence | `bash scripts/dev/fuzz-nightly.sh --short --out-dir <dir>` | `GOCACHE=/tmp/tetra-v0.3-go-cache GOENV=off bash scripts/dev/fuzz-nightly.sh --short --out-dir /tmp/tetra-v0.3-fuzz-short-audit-writable-cache` exited 0 during audit; `/tmp/tetra-v0.3-fuzz-short-audit-writable-cache/summary.md` reports all 6 steps pass. | pass |
| Linux native runtime smoke | `./tetra smoke --target linux-x64 --run=true --report <report>` plus smoke report validation | `./tetra smoke --target linux-x64 --run=true --report /tmp/tetra-v0.3-linux-runtime-smoke.json` exited 0; the report has `target: linux-x64`, `host: linux-x64`, `version: v0.3.0`, `total: 62`, `passed: 62`, `failed: 0`, and runtime cases have `ran: true`. `go run ./tools/cmd/smoke-report-to-checklist --validate-only --report /tmp/tetra-v0.3-linux-runtime-smoke.json` exited 0. | pass |
| Web UI browser smoke | `bash scripts/release/v1_0/web-smoke.sh --report <report>` | `bash scripts/release/v1_0/web-smoke.sh --report /tmp/tetra-v0.3-web-ui-smoke-rerun.json` exited 0 outside the sandbox restriction; the report has `status: pass`, `result: ok:0:ui=1:runtime=ok`, and a runtime trace covering boundary metadata, main exit, stdout, nonzero exit, failure propagation, repeated instantiation, and main instantiation. | pass |
| Fresh stabilization evidence archive | `bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir <dir>` | `bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir /tmp/tetra-v0.3-stabilization-audit-rerun` exited 0. `/tmp/tetra-v0.3-stabilization-audit-rerun/summary.md` reports `status: pass`, `step_count: 38`, `failed_count: 0`, `release_version: v0.3.0`, and `release_artifact: tetra.release.v0_3_0.test-all-summary.v1`. | pass |
| macOS runtime smoke | `./tetra smoke --target macos-x64 --run=true --report <report>` on macOS host or matching CI runner | `validate-release-state` reports `artifacts/macos-runtime-smoke.json` missing. | missing |
| Windows runtime smoke | `./tetra smoke --target windows-x64 --run=true --report <report>` on Windows host or matching CI runner | `validate-release-state` reports `artifacts/windows-runtime-smoke.json` missing. | missing |
| Human security signoff | `TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>` plus `bash scripts/release/v0_3_0/security-review.sh --signoff <file>` | `validate-release-state` reports `artifacts/security-review.md` and `.sha256` missing. | missing |
| Fresh v0.3.0 gate archive | `bash scripts/release/v0_3_0/gate.sh --report-dir <dir>` with required env inputs | `env TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1 TETRA_MACOS_RUNTIME_SMOKE_REPORT=docs/generated/v1_0/macos-smoke.json TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=docs/generated/v1_0/windows-smoke.json bash scripts/release/v0_3_0/gate.sh --report-dir /tmp/tetra-v0.3-gate-audit-crossbuild-runtime-blocked` produced a fresh `v0.3.0` identity-matched blocked gate report. It passed version, Go tests, stabilization, fuzz, docs, manifest, residual risks, and whitespace, but failed runtime evidence because the supplied macOS report has `host: linux-x64`, not `macos-x64`; release-state also reports both archived runtime execution artifacts missing and security signoff incomplete. | blocked |
| Release-state validator | `go run ./tools/cmd/validate-release-state --expected-version v0.3.0 --format=text --report-dir <dir>` | Running against `/tmp/tetra-v0.3-gate-audit-crossbuild-runtime-blocked` exits 1. It confirms gate identity is correct, required artifacts are present, and freshness checks pass, but runtime execution evidence is `0/2 pass`, security review evidence fails, and last gate evidence status is `blocked`. | blocked |
| Tag-ready clean worktree | `bash scripts/release/v0_3_0/gate.sh --report-dir <dir> --require-clean`; empty `git status --porcelain --untracked-files=all` | Current worktree has many tracked and untracked entries. | blocked |

## Current Blockers

The branch is not a complete `v0.3.0` release yet.

For the current Linux-only objective, macOS and Windows runtime evidence are
out of scope. The Linux-only release profile is audited in
`docs/release/v0_3_0_linux_release_audit.md`. The local candidate has version,
Linux runtime, stabilization, fuzz, docs, and web smoke evidence, but it does
not satisfy the older cross-platform release gate contract. The
machine-readable local candidate summary is
`docs/release/v0_3_0_local_candidate_summary.json`. The repo-local evidence
bundle is `reports/release-v0.3.0-local-candidate/summary.md`, with packaged
archive `reports/release-v0.3.0-local-candidate.tar.gz`.

Blocking evidence gaps:

- macOS and Windows host runtime smoke reports are missing.
- Human security signoff and detached hash are missing.
- Fresh `v0.3.0` release gate archive exists only as a blocked audit report:
  `/tmp/tetra-v0.3-gate-audit-crossbuild-runtime-blocked`.
- Worktree is not clean, so a tag-ready `--require-clean` pass cannot run.

Dirty-state classification is tracked in
`docs/release/v0_3_0_dirty_worktree_inventory.md`.

The exact external unblock packet for macOS/Windows runtime evidence and the
human security signoff is tracked in
`docs/release/v0_3_0_unblock_packet.md`.

## Next Concrete Actions

1. Review and classify the dirty worktree into release changes, generated
   evidence, historical cleanup, and unrelated local artifacts.
2. Obtain macOS and Windows runtime smoke reports from matching native hosts or
   CI runners.
3. Obtain the human `v0.3.0` security signoff.
4. Run `scripts/release/v0_3_0/gate.sh` with the runtime smoke and security
   signoff inputs.
5. Clean or commit the final branch state and rerun the gate with
   `--require-clean` before tagging.
