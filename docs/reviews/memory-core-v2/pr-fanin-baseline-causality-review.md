# PR Fan-in Baseline Causality Review

reviewed_head

`48b4b45e03ef356ef4bfc65748700e6ac1eb5064`

reviewed_merge_commit

`df54bc2ff05f2a14c097705cebce56bc0c651d7f`

ci_run_id

`28082310881`

ci_job_id

`83140236906`

gate_step

`Full-platform UI runtime gate`, nested step `baseline-tests`.

failing_package

`tetra_language/tools/scriptstest/test_all`

failing_test

Primary: `TestTestAllQuickFailsWhenUnsafePromotionBlockerSuiteMissing`.
Downstream: `TestTestAllAllowsDashPrefixedFreshReportDir`.

exact_error

CI log `/tmp/mcv2-pr-fanin-full.log` shows
`test_all_test.go:128: summary status/counts = "fail"/2, want fail/1`.
The summary contains the expected failing `unsafe promotion blocker suite` and
an unexpected failing `RAM contract fuzz oracle artifact gate`.

first_primary_failure

The first visible failure inside `baseline-tests` is
`TestTestAllQuickFailsWhenUnsafePromotionBlockerSuiteMissing`; earlier packages
in the broad baseline were `ok` or had no test files. The gate artifact only
records the aggregate `failed_steps: ["baseline-tests:1"]`, so the gate-level
summary is not the root cause.

downstream_failures

`TestTestAllAllowsDashPrefixedFreshReportDir` fails with `exit status 1` and a
quick-mode summary that stops after `unsafe promotion blocker suite`. This is
the same ambient fake-control leak seen by the primary failure, expressed in a
non-`--keep-going` test-all invocation.

nested_commands

- Gate command: `bash scripts/release/full_platform/ui-runtime-gate.sh --report-dir reports/full-platform-ui-runtime`
- Baseline command: `go test ./compiler/... ./cli/... ./tools/... -count=1`
- Failing helper command: `bash scripts/ci/test-all.sh --quick --keep-going --json-only --report-dir <tmp>/report`
- Nested RAM command: `go test ./tools/cmd/ram-contract-fuzz-short -list 'RAMContract|Fuzz|ReportDir'`

buildvcs_dependency

Not causal. The exact primary test passed 20/20 with buildvcs enabled and 20/20
with `-buildvcs=false` before the fix.

environment_dependency

Confirmed. `runTestAll`, `runTestAllSplit`, and
`runTestAllFromWorkingDir` built subprocess environments from `os.Environ()`
plus test-specific env overrides. With ambient
`TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST=1` and
`TETRA_FAKE_SKIP_RAM_CONTRACT_LIST=1`, the actual PR merge worktree reproduced
the CI primary error exactly.

shared_state_dependency

Confirmed as ambient process-environment state in the test harness. No
production `scripts/ci/test-all.sh` behavior is required to reproduce it, and no
`tools/scriptstest/test_all` source mutation via `os.Setenv`, `os.Unsetenv`, or
`t.Setenv` was found for these variables outside explicit tests.

root_cause

`tools/scriptstest/test_all` fake-repo subprocess helpers inherited ambient
test-only `TETRA_FAKE_*` and `TETRA_FAIL_*` controls, so an outer process env
could turn a focused negative test that expected one fake blocker into a
multi-blocker failure.

minimal_fix

Filter ambient `TETRA_FAKE_*` and `TETRA_FAIL_*` entries before constructing
fake-repo subprocess environments, then append each test's explicit env slice so
intentional negative tests still work.

regression_test

`TestTestAllQuickIgnoresAmbientFakeFailureControls` seeds ambient fake failure
controls and verifies a normal quick fake-repo run still passes. Existing
explicit-env negative tests continue to verify intentional fake failures.

verdict

ROOT_CAUSE_CONFIRMED
