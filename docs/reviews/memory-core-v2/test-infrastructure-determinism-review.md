reviewed_commit

- Source tree reviewed at `df3256b904f08b036c5378bcc003c36c0bed5c3e`.
- Evidence: `git rev-parse HEAD` in
  `/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2`
  returned `df3256b904f08b036c5378bcc003c36c0bed5c3e`.

ci_run_ids

- Push fan-in log:
  `/tmp/mcv2-push-fanin-failed.log`, `run_id=28020556066`,
  `job_id=82935734296`.
- PR fan-in log:
  `/tmp/mcv2-pr-fanin-failed.log`, `run_id=28020557965`,
  `job_id=82936970947`.
- Commit evidence in logs is not identical: the push log artifact env names
  include `df3256b904f08b036c5378bcc003c36c0bed5c3e`, while the PR log artifact
  env names include `a1cae0b93fa41cf0ab3319633dabd9c222e38287`.

failing_package

- Primary failed CI step in both logs: `full-platform-ui-runtime-gate-linux` /
  `baseline-tests`.
- Primary packages failing inside that step:
  `tetra_language/cli/cmd/tetra` and
  `tetra_language/tools/cmd/validate-v0-4-readiness`.
- Secondary fan-in package:
  `tetra_language/tools/scriptstest/workspace`, because
  `TestWorkspaceModules` reruns `go test ./... -count=1` in `cli` and `tools`.
- Additional differing surface:
  `tetra_language/tools/scriptstest/test_all`.

failing_test

- Both logs:
  `TestTargetMetadataCheck/wasi_runner_available`,
  `TestTargetsCommandJSON`,
  `TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape`, and
  `TestWorkspaceModules`.
- D-003 candidate:
  `TestTargetMetadataCheck/wasi_runner_available` and
  `TestTargetsCommandJSON` in `tetra_language/cli/cmd/tetra`.
- D-004 candidate:
  `TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape` in
  `tetra_language/tools/cmd/validate-v0-4-readiness`.
- Push-only `test_all` surface:
  `TestTestAllFullToolingSummaryFailsOnZeroByteRequiredArtifact`.
- PR-only `test_all` surface:
  `TestTestAllStabilizationToolingSummaryRequiresFocusedArtifacts`.

exact_error

- Both logs end the gate with `FAIL: baseline-tests (exit 1)` and
  `failed steps: baseline-tests:1`.
- Both logs show `cli/cmd/tetra` failing on linux-x32 host-probed metadata:
  `linux-x32 unsupported host-probed metadata = ... RunSupported:false,
  RunUnsupportedReason:"host linux/amd64 does not support Linux x32 ABI
  execution; no host fallback is allowed; probe command: tetra test
  --diagnostics=json --target x32 --format=json <runner-smoke.tetra>"`.
- Both logs show `tools/cmd/validate-v0-4-readiness` failing with:
  `expected native UI runtime-shaped evidence to pass readiness: decision
  ui.native-runtime evidence.docs path docs/user/surface/wasm_ui_guide.md is
  not readable`.
- Both logs then show `TestWorkspaceModules/cli` and
  `TestWorkspaceModules/tools` repeating the same `cli/cmd/tetra` and
  `validate-v0-4-readiness` failures from nested module test runs.
- The `test_all` error differs by log. Push reports:
  `summary missing failing tooling summary step` and the captured summary stops
  after `unsafe promotion blocker suite` failed. PR reports the same assertion
  text but the captured summary stops after
  `RAM contract fuzz oracle artifact gate` failed.

root_cause

- A single shared root cause in test infrastructure determinism/isolation is
  not confirmed.
- Observed chain: `scripts/release/full_platform/ui-runtime-gate.sh` runs
  `baseline-tests` as `go test ./compiler/... ./cli/... ./tools/... -count=1`.
  That broad command fails before the gate continues to later release steps.
- The workspace failure is secondary, not an independent proof of D-001:
  `tools/scriptstest/workspace/workspace_modules_test.go` runs nested
  `go test ./... -count=1` in `cli` and `tools`, so it re-surfaces the already
  visible package failures inside the same CI step.
- The readiness failure is tracked separately as D-004 pending a focused path
  review. This review only records the observed missing-readable-path error.
- The linux-x32 failure is tracked separately as D-003 pending a focused target
  contract review. This review only records the observed CI value and the fact
  that a local targeted run did not reproduce it on this host.
- The `test_all` failures are not confirmed as the same root cause: the two CI
  logs fail different `test_all` tests and local targeted runs of those two
  tests passed.
- D-001 is related as a prior hardening item for broad `tools/scriptstest`
  isolation, but the current logs do not show D-001's recorded signatures:
  no `$WORK/.../_pkg_.a` import archive failure and no GitHub-shaped remote URL
  failure.

shared_state

- Observed shared boundary: `TestWorkspaceModules` inherits `os.Environ()` for
  nested module test subprocesses and only appends
  `TETRA_WORKSPACE_MODULES_SUBPROCESS=1`, `GOCACHE`, `GOTMPDIR`, and `TMPDIR`.
- Observed host-sensitive value: the linux-x32 metadata assertion failed on the
  exact unsupported-reason string emitted in CI.
- Not confirmed: a shared temp/cache/order root cause across both logs.
  `TestWorkspaceModules` already uses per-module cache/tmp directories, and the
  logs do not contain D-001's missing `$WORK/.../_pkg_.a` archive errors.
- Not confirmed: a single fake-repo `test_all` shared-state cause. Push and PR
  show different `test_all` first-failing blocker steps, and the local targeted
  `test_all` diagnostics passed.

reproduction

- CI reproduction from logs:
  `bash scripts/release/full_platform/ui-runtime-gate.sh --report-dir
  reports/full-platform-ui-runtime` enters `baseline-tests`, which runs
  `go test ./compiler/... ./cli/... ./tools/... -count=1` and exits 1 in both
  logs.
- Local diagnostic command:
  `GOCACHE=${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-mcv2-subagent-e-readiness go test -buildvcs=false ./tools/cmd/validate-v0-4-readiness -run TestValidateReadinessAcceptsNativeUIRuntimeEvidenceShape -count=1`
  reproduced the readiness error exactly.
- Local diagnostic command:
  `GOCACHE=${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-mcv2-subagent-e-cli go test -buildvcs=false ./cli/cmd/tetra -run 'TestTargetMetadataCheck|TestTargetsCommandJSON$' -count=1`
  passed locally, so the linux-x32 CI failure was not reproduced on this host.
- Local diagnostic command:
  `GOCACHE=${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-mcv2-subagent-e-testall go test -buildvcs=false ./tools/scriptstest/test_all -run 'TestTestAllFullToolingSummaryFailsOnZeroByteRequiredArtifact|TestTestAllStabilizationToolingSummaryRequiresFocusedArtifacts' -count=1`
  passed locally, so the differing `test_all` CI surfaces are not confirmed as
  one shared root cause.
- Cleanup commands run after diagnostics:
  `GOCACHE=${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-mcv2-subagent-e-cli go clean -cache`,
  `GOCACHE=${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-mcv2-subagent-e-readiness go clean -cache`,
  and
  `GOCACHE=${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-mcv2-subagent-e-testall go clean -cache`.

minimal_fix

- Do not implement a broad rewrite for D-001 based on these two logs alone.
- No code fix is authorized by this review because the focused root causes are
  not confirmed here.
- Split the investigation into D-003 for the linux-x32 contract mismatch and
  D-004 for the readiness guide path failure.
- Treat the `tools/scriptstest/test_all` broad-run behavior as a separate
  follow-up investigation before changing it, because the two failing surfaces
  differ and the targeted local diagnostics passed.

forbidden_workarounds

- Do not mark D-001 as the confirmed root cause solely because
  `tools/scriptstest/workspace` appears in the fan-in logs.
- Do not silence or skip `baseline-tests` in the full-platform UI runtime gate.
- Do not serialize all packages, disable workspace tests, or globally unset
  environment variables without evidence that the specific state causes the
  current failures.
- Do not replace the failing fake-repo `test_all` assertions with looser
  summary checks without first reproducing the broad-run failure mechanism.
- Do not use a new GitHub CI rerun as a substitute for repo-grounded local
  evidence.

verdict

ROOT_CAUSE_NOT_CONFIRMED
