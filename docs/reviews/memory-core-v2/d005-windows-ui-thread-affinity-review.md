# D-005 Windows UI Thread-Affinity Review

reviewed_commit: `3b8c02b0579cbd778a628f7f3245d7badef956e5`

pr_merge_commit: `8e68d34c303479b9fd1754592b15923be2873936`

ci_run_id: `28052545600`

ci_job_id: `83046625456`

workflow_timeout: `45 minutes`

failing_step: `Target-host UI runtime smoke`

last_confirmed_stage:
`Build CLI` completed and printed `v0.4.0`. The failing step started
`go run ./tools/cmd/platform-ui-runtime-smoke --target "windows-x64" --report
"windows-ui-runtime.json"` and was later cancelled by the GitHub runner with
`exit status 0xc000013a` and `The operation was canceled`.

window_creator_thread_contract:
The Windows probe creates the parent window with `CreateWindowExW` and then
creates child controls with more `CreateWindowExW` calls. Win32 assigns each
window to the thread that created it, and that owner thread is responsible for
retrieving and dispatching messages for the window.

go_thread_pinning:
At the failing commit, `tools/cmd/platform-ui-runtime-smoke/platform_probe_windows.go`
did not call `runtime.LockOSThread` before registering the class, creating the
window, creating controls, sending messages, dispatching messages, redrawing,
destroying the window, or unregistering the class.

synchronous_win32_calls:
The probe uses synchronous User32 calls after window creation:

- `SendMessageW` for edit/list/button/label operations.
- `DispatchMessageW` for the retrieved timer message.
- `RedrawWindow` with `RDW_UPDATENOW`, which asks User32 to process pending
  painting immediately.

message_queue_owner:
The failing implementation had only a one-shot `PostMessageW`/`PeekMessageW`/
`DispatchMessageW` sequence in the same goroutine that created the window. If
that goroutine moved to another OS thread after window creation, the original
window-owner thread had no independent message pump.

child_process_timeout: `false`

build_process_timeout: `false`

branch_merge_relevant_code_identical: `true`

exact_blocking_call_proven: `false`

contract_violation_confirmed: `true`

root_cause:
The Windows platform UI probe violated the Win32 UI thread-affinity contract.
The code created HWNDs and then made synchronous User32 lifecycle calls without
pinning the Go goroutine to one OS thread. Go may move an unlocked goroutine
between OS threads at syscall boundaries. If a later `SendMessageW`,
`PeekMessageW`, `DispatchMessageW`, `RedrawWindow`, `DestroyWindow`, or
`UnregisterClassW` ran on a different OS thread than the one that created the
window, the probe could enter a cross-thread synchronous wait while the owner
thread had no guaranteed message retrieval loop. The observed PR-only Windows
timeout is consistent with that nondeterministic migration. The `0xc000013a`
status is treated as a consequence of GitHub runner cancellation, not as a
separate application-level error.

minimal_fix:
Commit `a93dede994b29a86af5efde3434951410e737a34` pins the Windows probe with
`runtime.LockOSThread` immediately after target validation and defers
`runtime.UnlockOSThread` until after the full User32 lifecycle has completed.
The lock covers `RegisterClassExW`, `CreateWindowExW`, child control creation,
`ShowWindow`, `UpdateWindow`, `SetFocus`, all `SendMessageW` calls,
`SetTimer`, `PostMessageW`, `PeekMessageW`, `TranslateMessage`,
`DispatchMessageW`, `RedrawWindow`, `DestroyWindow`, and `UnregisterClassW`.

defense_in_depth:
The same fix commit adds fail-closed local deadlines to the nested platform
runtime build and child execution. The nested build is bounded to `5m0s`; the
child runtime is bounded to `1m0s`. Timeout failures return a nonzero result,
populate a failed platform UI report, record a blocker, and allow the outer
smoke command to write JSON before the workflow-level `45m` job timeout.

regression_test:
The fix adds a Windows-only helper-subprocess regression,
`TestWindowsPlatformProbeCompletesUnderSchedulerPressure`, which runs the real
Win32 probe under scheduler pressure with a `15s` parent timeout and checks the
required Win32 markers. It also adds
`TestPlatformRuntimeChildTimeoutIsBounded`, which injects a blocking child
command and verifies that the runner fails closed quickly with a JSON-marshalable
failed report and a child-timeout blocker.

forbidden_fixes:
No workflow timeout increase, retry loop, `t.Skip`, optional Windows job,
synthetic report, placeholder artifact, removed Win32 probe, removed target, or
validator relaxation was used.

verdict: `ROOT_CAUSE_CONFIRMED`

