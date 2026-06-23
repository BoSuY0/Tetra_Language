## reviewed_commit

`30f1f7bd71c1972bc37ef937ee913ade3b3cbb80`

The source tree reviewed was the docs-only diagnostic HEAD in
`/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2`.
Graphify artifacts were present but built from `df3256b9`; concrete findings
below are from direct source and CI-log inspection at `30f1f7bd...`.

## ci_run_id

- Push: `28020556066`, `full-platform-ui-runtime-gate-linux`,
  `/tmp/d003-push-full.log`.
- Pull request: `28020557965`, `full-platform-ui-runtime-gate-linux`,
  `/tmp/d004-pr-full.log`.

The same D-003 linux-x32 host-probed metadata failure is visible in the failed
excerpts `/tmp/mcv2-push-fanin-failed.log` and
`/tmp/mcv2-pr-fanin-failed.log`.

## failing_test

- `tetra_language/cli/cmd/tetra.TestTargetMetadataCheck/wasi_runner_available`
  at `cli/cmd/tetra/tetra_suite_test.go:9367`.
- `tetra_language/cli/cmd/tetra.TestTargetsCommandJSON` at
  `cli/cmd/tetra/tetra_suite_test.go:15126`.

`TestWorkspaceModules/cli` repeats the same package failures through nested
module test execution; it is not a distinct linux-x32 contract failure.

## actual_value

The failing field is `run_unsupported_reason`, not the target registry
`unsupported_reason`.

Actual CI value:

`host linux/amd64 does not support Linux x32 ABI execution; no host fallback is allowed; probe command: tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>`

The adjacent `unsupported_reason` field is the long linux-x32 capability/detail
reason from the target registry and includes
`host-probed source run/test execution` plus
`available when the Linux kernel supports the x32 ABI`.

## expected_value

The two failing CLI assertions expected substring containment, not exact string
equality:

- `host does not support Linux x32 ABI execution`
- `no host fallback`

The first expected substring no longer matches the actual host-qualified value
because actual begins `host linux/amd64 does not...`.

## source_of_truth

- Target capability registry: `compiler/target/target.go:589` defines
  `linux-x32` as `build_only`, `RunModeHostProbed`, x32 data model, x64
  register width, memory metadata, probe command, and the long
  `UnsupportedReason`.
- Production source of actual `run_unsupported_reason`:
  `cli/cmd/tetra/tetra_core.go:972` `buildOnlyNativeRunUnsupportedReason`,
  which uses `runtime.GOOS + "/" + runtime.GOARCH` and formats
  `host <goos>/<goarch> does not support Linux x32 ABI execution; no host fallback is allowed; probe command: <probe>`.
- Metadata call path: `cli/cmd/tetra/tetra_commands.go:1091` routes
  host-probed build-only targets through `canRunBuildOnlyNativeTargetOnHost`;
  when the probe fails, it returns `buildOnlyNativeRunUnsupportedReason(tgt)`.
- Contract validators treat this as human-readable diagnostic detail with
  required contract fragments, not an exact stable literal. In particular,
  `tools/cmd/validate-targets/main.go:928` requires `no host fallback`,
  `host `, and `probe command:`.

## duplicate_sources

- One production formatter generates the host-probed run reason:
  `buildOnlyNativeRunUnsupportedReason`.
- The stale expected substring appears twice in CLI metadata tests:
  `cli/cmd/tetra/tetra_suite_test.go:9362` and
  `cli/cmd/tetra/tetra_suite_test.go:15121`.
- The same test file also contains newer host-qualified expectations for
  run/test diagnostics at `cli/cmd/tetra/tetra_suite_test.go:17051` and
  `cli/cmd/tetra/tetra_suite_test.go:19546`.
- Validators and fixtures already accept or encode host-qualified metadata:
  `tools/cmd/validate-targets/main_test.go:98`,
  `tools/cmd/validate-linux-native-targets/main.go:1060`, and
  `tools/cmd/validate-linux-native-targets/main_test.go:1380`.
- `docs/generated/manifest.json` duplicates the target registry
  `unsupported_reason`, not the host-probed `run_unsupported_reason`.

## host_dependencies

The failing branch depends on the executing machine.

- `canRunLinuxX32OnHost` in `cli/cmd/tetra/tetra_core.go:918` returns false
  unless `runtime.GOOS == "linux"` and `runtime.GOARCH == "amd64"`, then runs a
  real x32 execution probe.
- `GOHOSTOS` and `GOHOSTARCH` are not read by the production code. Runtime
  behavior uses `runtime.GOOS` and `runtime.GOARCH`; in normal `go test`
  execution these match the running host. CI showed linux/amd64.
- On the reviewed local host, `go env GOOS GOARCH GOHOSTOS GOHOSTARCH` returned
  `linux`, `amd64`, `linux`, `amd64`, and
  `go run ./cli/cmd/tetra targets --format=json` reported
  `linux-x32.run_supported=true`. Therefore the unsupported branch did not run
  locally, and the focused metadata tests passed.
- On CI linux/amd64, the x32 probe failed, so
  `run_supported=false` and the host-qualified reason was emitted.

## reproduction

Evidence commands run in the reviewed worktree:

- `rg -n -- '--- FAIL|linux-x32 unsupported host-probed metadata|RunUnsupportedReason' /tmp/d003-push-full.log /tmp/d004-pr-full.log /tmp/mcv2-push-fanin-failed.log /tmp/mcv2-pr-fanin-failed.log`
  found the direct failures and actual `RunUnsupportedReason` in both push and
  PR logs.
- `go test ./cli/cmd/tetra -run 'Test(TargetMetadataCheck|TargetsCommandJSON)$' -count=1 -v`
  passed locally because this host reports `linux-x32.run_supported=true`.
- `go run ./cli/cmd/tetra targets --format=json | jq -r '.targets[] | select(.triple=="linux-x32") | {triple,run_supported,run_unsupported_reason,unsupported_reason}'`
  showed `run_supported: true` and no `run_unsupported_reason` locally.
- `go test ./cli/cmd/tetra -run 'Test(RunCommandJSONDiagnosticsForLinuxX32HostUnsupported|TestCommandJSONDiagnosticsForLinuxX32HostUnsupported)$' -count=1 -v`
  passed, confirming existing diagnostics tests expect host-qualified x32
  unsupported text.
- `go test ./tools/cmd/validate-targets -run TestValidateRunContract -count=1 -v`
  and
  `go test ./tools/cmd/validate-linux-native-targets -run 'TestValidateLinuxNativeTargets(AcceptsBuildOnlyNoHostFallbackRunnerDiagnostic|RejectsNoHostRunnerDiagnosticWhenMetadataRunSupported)$' -count=1 -v`
  passed, confirming validator-side acceptance of the current contract shape.

## root_cause

Root cause is test expectation drift in two CLI metadata assertions.

Production now emits a host-qualified host-probed run diagnostic:
`host <GOOS>/<GOARCH> does not support Linux x32 ABI execution...`. That is
consistent with validators requiring host identity and with newer diagnostics
tests in the same suite. The two failing metadata assertions still look for the
older unqualified substring `host does not support Linux x32 ABI execution`.

The production classification is not shown to be incorrect: `linux-x32` remains
build-only/host-probed, and `run_supported=false` is correct on a linux/amd64
host whose kernel cannot execute x32 binaries. The failure only appears on hosts
where the x32 probe fails because the tests do not stub the host-probe result.

## minimal_fix

Do not change production classification or the emitted reason. Update only the
two stale CLI metadata assertions to use the production
`buildOnlyNativeRunUnsupportedReason` constructor as the canonical source of
truth. Force the affected tests through `linuxX32HostSupport(false)` so the
unsupported branch is covered deterministically on hosts that can execute x32.

## regression_test

Make the unsupported metadata branch deterministic by stubbing
`linuxX32HostSupport(false)` around
`TestTargetMetadataCheck/wasi_runner_available` and `TestTargetsCommandJSON`,
then assert exact equality with the canonical production reason constructor.

Relevant focused verification after a fix:

- `go test ./cli/cmd/tetra -run 'Test(TargetMetadataCheck|TargetsCommandJSON)$' -count=1 -v`
- `go test ./tools/cmd/validate-targets -run TestValidateRunContract -count=1 -v`
- `go test ./tools/cmd/validate-linux-native-targets -run 'TestValidateLinuxNativeTargets(AcceptsBuildOnlyNoHostFallbackRunnerDiagnostic|RejectsNoHostRunnerDiagnosticWhenMetadataRunSupported)$' -count=1 -v`

## resolution

status: resolved

fix_commit: `7e9184aaa2c220590d67f3d369d9598b62861088`

before_behavior:
- CI linux/amd64 hosts without x32 execution support emitted
  `host linux/amd64 does not support Linux x32 ABI execution; no host fallback is allowed; probe command: tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>`.
- `TestTargetMetadataCheck/wasi_runner_available` and `TestTargetsCommandJSON`
  rejected that value because they still looked for the stale unqualified
  substring `host does not support Linux x32 ABI execution`.

after_behavior:
- The two affected metadata tests force `linuxX32HostSupport(false)`.
- Both tests compare `run_unsupported_reason` with
  `buildOnlyNativeRunUnsupportedReason(ctarget.Parse("linux-x32"))`.
- Existing linux-x32 run/test diagnostics reuse the same expected value.

regression_tests:
- RED:
  `go test -buildvcs=false ./cli/cmd/tetra -run '^(TestTargetMetadataCheck|TestTargetsCommandJSON)$' -count=1 -v`
  failed after the unsupported branch was forced while the old assertions were
  still present.
- GREEN:
  `go test -buildvcs=false ./cli/cmd/tetra -run '^(TestTargetMetadataCheck|TestTargetsCommandJSON|TestRunCommandJSONDiagnosticsForLinuxX32HostUnsupported|TestTestCommandJSONDiagnosticsForBuildOnlyRuntimeUnsupported)$' -count=20 -v`
  passed.

merge_recommendation: pending full local validation and new exact-HEAD CI; do
not mark PR ready until all requested gates pass.

## forbidden_fixes

- Do not remove host identity from `buildOnlyNativeRunUnsupportedReason`.
- Do not mark `linux-x32` as host-native, fully supported, or runnable by
  fallback on hosts that fail the x32 probe.
- Do not make `run_unsupported_reason` empty when `run_supported=false`.
- Do not weaken validators that require `no host fallback`, host identity, or
  the probe command.
- Do not change the target registry `UnsupportedReason` to paper over a
  `RunUnsupportedReason` assertion drift.
- Do not skip the tests or spoof CI/kernel capability as the fix.

## verdict

ROOT_CAUSE_CONFIRMED
