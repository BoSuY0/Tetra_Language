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

### Release Version

- Required evidence: `./tetra version`; `./t version`.
- Current evidence: both commands printed `v0.3.0` during audit.
- Status: pass.

### Full Go Package Suite

- Required evidence: `go test ./compiler/... ./cli/... ./tools/... -count=1`.
- Current evidence: command exited 0 during audit.
- Status: pass.

### Current Truth Documents

- Required evidence:
  `README.md`; `docs/spec/current_supported_surface.md`; `docs/spec/v0_3_scope.md`.
- Current evidence: documents exist and describe `v0.3.0` as the current public
  profile.
- Status: pass.

### Promoted Enum Payload Match Slice

- Required evidence: `go test ./compiler/... -run 'Enum|Match|TypedError'`.
- Current evidence: covered by focused compiler promotion run using the combined
  v0.3 regex; command exited 0.
- Status: pass.

### Promoted Static Protocol-Bound Generics Slice

- Required evidence:
  `go test ./compiler/... -run 'Generic|Protocol|Conformance|Extension'`.
- Current evidence: covered by focused compiler promotion run using the combined
  v0.3 regex; command exited 0.
- Status: pass.

### Callable Level 1 Scope

- Required evidence: `compiler/features.go`; current supported surface docs.
- Current evidence: feature registry and docs keep Level 1 experimental and
  Level 2 planned.
- Status: pass.

### Ownership And Resource Safety

- Required evidence: `docs/spec/current_supported_surface.md`; focused safety
  tests.
- Current evidence: focused compiler promotion run covered the ownership,
  resource, task, and island regex; command exited 0.
- Status: pass.

### CLI/Eco/Project/Workspace Workflows

- Required evidence:
  `go test ./cli/... ./tools/... -run 'Eco|Project|Workspace|Artifact|Capsule|Lock'`.
- Current evidence: command exited 0 during audit.
- Status: pass.

### Docs And Manifest Verification

- Required evidence:
  `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- Current evidence: command exited 0 during audit.
- Status: pass.

### Whitespace And Diff Hygiene

- Required evidence: `git diff --check`.
- Current evidence: command exited 0 during audit.
- Status: pass.

### Short Fuzz Smoke Evidence

- Required evidence:
  `bash scripts/dev/fuzz-nightly.sh --short --out-dir <dir>`.
- Current evidence: historical audit command with `GOENV=off` exited 0.
- Current evidence:
  `/tmp/tetra-v0.3-fuzz-short-audit-writable-cache/summary.md` reports all
  6 steps pass.
- Status: pass.

### Linux Native Runtime Smoke

- Required evidence:
  `./tetra smoke --target linux-x64 --run=true --report <report>`.
- Required evidence: smoke report validation.
- Current evidence: `/tmp/tetra-v0.3-linux-runtime-smoke.json` exited 0.
- Current evidence: report has `target: linux-x64`, `host: linux-x64`,
  `version: v0.3.0`, `total: 62`, `passed: 62`, and `failed: 0`.
- Current evidence: runtime cases have `ran: true`.
- Current evidence: smoke-report-to-checklist validation exited 0.
- Status: pass.

### Web UI Browser Smoke

- Required evidence: `bash scripts/release/v1_0/web-smoke.sh --report <report>`.
- Current evidence: historical rerun exited 0 outside the sandbox restriction.
- Current evidence: report has `status: pass` and `result: ok:0:ui=1:runtime=ok`.
- Current evidence: trace covers boundary metadata, main exit, stdout, nonzero
  exit, failure propagation, repeated instantiation, and main instantiation.
- Status: pass.

### Fresh Stabilization Evidence Archive

- Required evidence:
  `bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir <dir>`.
- Current evidence: historical rerun exited 0.
- Current evidence: summary reports `status: pass`, `step_count: 38`,
  `failed_count: 0`, `release_version: v0.3.0`, and release artifact
  `tetra.release.v0_3_0.test-all-summary.v1`.
- Status: pass.

### macOS Runtime Smoke

- Required evidence:
  `./tetra smoke --target macos-x64 --run=true --report <report>` on macOS host.
- Current evidence: `validate-release-state` reports
  `artifacts/macos-runtime-smoke.json` missing.
- Status: missing.

### Windows Runtime Smoke

- Required evidence:
  `./tetra smoke --target windows-x64 --run=true --report <report>` on Windows.
- Current evidence: `validate-release-state` reports
  `artifacts/windows-runtime-smoke.json` missing.
- Status: missing.

### Human Security Signoff

- Required evidence: `TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>`.
- Required evidence: `bash scripts/release/v0_3_0/security-review.sh --signoff`.
- Current evidence: `validate-release-state` reports
  `artifacts/security-review.md` and `.sha256` missing.
- Status: missing.

### Fresh v0.3.0 Gate Archive

- Required evidence:
  `bash scripts/release/v0_3_0/gate.sh --report-dir <dir>` with required env.
- Current evidence: historical cross-build gate produced a fresh identity-matched
  blocked report.
- Current evidence: it passed version, Go tests, stabilization, fuzz, docs,
  manifest, residual risks, and whitespace.
- Current evidence: it failed runtime evidence because the supplied macOS report
  has `host: linux-x64`, not `macos-x64`.
- Current evidence: release-state also reports missing archived runtime execution
  artifacts and incomplete security signoff.
- Status: blocked.

### Release-State Validator

- Required evidence:
  `go run ./tools/cmd/validate-release-state --expected-version v0.3.0`.
- Required evidence: `--format=text --report-dir <dir>`.
- Current evidence: running against the blocked cross-build report exits 1.
- Current evidence: gate identity is correct, artifacts are present, and
  freshness checks pass.
- Current evidence: runtime execution evidence is `0/2 pass`, security review
  fails, and last gate evidence status is `blocked`.
- Status: blocked.

### Tag-Ready Clean Worktree

- Required evidence:
  `bash scripts/release/v0_3_0/gate.sh --report-dir <dir> --require-clean`.
- Required evidence: empty `git status --porcelain --untracked-files=all`.
- Current evidence: current worktree has many tracked and untracked entries.
- Status: blocked.

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
