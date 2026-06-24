# D-001 Test-All Environment Contract Review

reviewed_commit: `db821745b09a44c46f0c70246b87b5a4c3f3996f`

previous_fix_commit: `9ca65b6b820e477059513a31ebc190f5062f84b5`

run_helpers:
At the reviewed commit, `runTestAll`, `runTestAllSplit`, and
`runTestAllFromWorkingDir` all executed `bash scripts/ci/test-all.sh`, set
`cmd.Dir`, and built `cmd.Env` from `append(filteredTestAllEnv(), explicit...)`
plus a fake-repo `PATH`. `filteredTestAllEnv()` started from `os.Environ()` and
removed only ambient `TETRA_FAKE_*` and `TETRA_FAIL_*` entries.

ambient_environment_source:
The helper contract still used the parent process environment as the base. That
left non-denylisted shell startup, Go state, Git state, CI variables, report-path
variables, `HOME`, `XDG_*`, temp variables, and arbitrary host keys available to
fake-repo subprocesses. Independent `test_script_test.go` fake-repo probes also
used `append(os.Environ(), ...)`.

explicit_environment_keys:
- `TETRA_TEST_GO_LOG`
- `TETRA_TEST_GOFMT_LOG`
- `TETRA_FAKE_GO_LOG`
- `TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST`
- `TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST`
- `TETRA_FAKE_SKIP_RAM_CONTRACT_LIST`
- `TETRA_FAKE_SKIP_MEMORY_FUZZ_ORACLE_LIST`
- `TETRA_FAKE_SKIP_HOST_LEAK_LIST`
- `TETRA_FAKE_ZERO_DOCTOR_REPORT`
- `TETRA_FAKE_SKIP_WEB_UI_SMOKE_REPORT`
- `TETRA_FAKE_TETRA_VERSION`
- `TETRA_FAIL_FMT`
- `TETRA_FAIL_SUMMARY_VALIDATOR`
- `TETRA_FAIL_SAFETY_READINESS`
- `TETRA_TEST_ALL_RELEASE_VERSION`
- `TETRA_TEST_ALL_RELEASE_ARTIFACT`

script_environment_reads:
- `scripts/ci/test-all.sh` reads `TETRA_TEST_ALL_RELEASE_VERSION`.
- `scripts/ci/test-all.sh` reads `TETRA_TEST_ALL_RELEASE_ARTIFACT`.
- It unsets `TETRA_TEST_ALL_RELEASE_VERSION`, `TETRA_TEST_ALL_RELEASE_ARTIFACT`,
  and `TETRA_SECURITY_REVIEW_SIGNOFF` for two nested production-style steps.
- It relies indirectly on `PATH`, `HOME`, temp variables, Go environment, Git
  environment, and bash startup behavior.

fixture_environment_reads:
- Fake `web-smoke.sh`: `TETRA_FAKE_SKIP_WEB_UI_SMOKE_REPORT`.
- Fake `go`: `TETRA_FAKE_GO_LOG`, `TETRA_FAIL_SUMMARY_VALIDATOR`,
  `TETRA_FAIL_SAFETY_READINESS`, `TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST`,
  `TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST`,
  `TETRA_FAKE_SKIP_MEMORY_FUZZ_ORACLE_LIST`,
  `TETRA_FAKE_SKIP_HOST_LEAK_LIST`, and
  `TETRA_FAKE_SKIP_RAM_CONTRACT_LIST`.
- Fake `git`: no environment reads found.
- Fake `tetra`: `TETRA_FAKE_TETRA_VERSION`, `TETRA_FAIL_FMT`,
  `TETRA_FAKE_ZERO_DOCTOR_REPORT`, and `TETRA_FAKE_SMOKE_REPORT_FAIL`.
- `test_script_test.go` fake tools: `TETRA_TEST_GO_LOG` and
  `TETRA_TEST_GOFMT_LOG`.

ci_environment_keys:
The full-platform fan-in jobs define target-host report paths:
`TETRA_WINDOWS_UI_RUNTIME_REPORT` and `TETRA_MACOS_UI_RUNTIME_REPORT`. Logs also
show standard GitHub Actions key names such as `GITHUB_*` and runner/setup-go
metadata; secret values were not inspected or recorded.

shell_startup_channels:
- `BASH_ENV`
- `ENV`
- `SHELLOPTS`
- `BASHOPTS`
- `CDPATH`
- parent-side lookup of `bash`
- fixture shebang lookup via `/usr/bin/env bash`

go_state_channels:
- `GOENV`
- `GOFLAGS`
- `GOCACHE`
- `GOMODCACHE`
- `GOTMPDIR`
- `GOWORK`
- `GOTOOLCHAIN`
- `GOPATH`
- `GOROOT`
- `GOTELEMETRY`
- `HOME`
- `XDG_CACHE_HOME`

git_state_channels:
- `GIT_DIR`
- `GIT_WORK_TREE`
- `GIT_INDEX_FILE`
- `GIT_CONFIG_GLOBAL`
- `GIT_CONFIG_SYSTEM`
- `GIT_CONFIG_NOSYSTEM`
- `GIT_CONFIG_COUNT` / `GIT_CONFIG_KEY_*` / `GIT_CONFIG_VALUE_*`
- `GIT_CEILING_DIRECTORIES`
- `GIT_SSH_COMMAND`
- `GIT_TRACE*`
- `HOME`
- `XDG_CONFIG_HOME`

report_path_channels:
- `--report-dir`
- test-all per-step log and summary paths
- `TETRA_WINDOWS_UI_RUNTIME_REPORT`
- `TETRA_MACOS_UI_RUNTIME_REPORT`
- `TETRA_ACTIONS_STARTUP_BLOCKER_REPORT`
- `TETRA_FAKE_GO_LOG`
- `TETRA_TEST_GO_LOG`
- `TETRA_TEST_GOFMT_LOG`
- `GITHUB_STEP_SUMMARY`

duplicate_key_behavior:
`exec.Cmd.Env` is last-value-wins for duplicate keys. The previous helpers put
explicit test env after the filtered ambient env and appended fake `PATH` last,
so explicit test keys usually won, but duplicates, non-denylisted ambient keys,
and shell-startup reinjection remained possible.

protected_keys:
The previous fix only protected ambient `TETRA_FAKE_*` and `TETRA_FAIL_*` via a
prefix denylist. It did not protect `BASH_ENV`, `ENV`, Go state keys, Git config
keys, `HOME`, `XDG_*`, temp keys, fan-in report path keys, or arbitrary
non-fake `TETRA_*` keys.

required_host_keys:
- A controlled `PATH` that puts `<fake repo>/bin` first and then uses the host
  `PATH` only as a suffix.
- Windows-only process keys when needed: `SystemRoot`, `ComSpec`, `PATHEXT`.
- No secret-bearing GitHub, SSH, or host Git configuration keys are required for
  fake test-all subprocesses.

root_cause:
The previous fix was an ambient denylist, not a hermetic execution contract.
Push and PR CI still showed extra fake blocker-suite failures in
`tools/scriptstest/test_all` after commit `9ca65b6...`, while the reviewed
helpers still inherited nearly all parent environment state.

required_contract:
Build fake-repo subprocess environments from a synthetic allowlist map, not from
`os.Environ()`. Set controlled `HOME`, `XDG_CACHE_HOME`, `GOCACHE`, `GOTMPDIR`,
`TMPDIR`, `TMP`, `TEMP`, `GOENV=off`, `GOWORK=off`, empty `GOFLAGS`,
`GOTELEMETRY=off`, `LANG=C`, `LC_ALL=C`, `TZ=UTC`, and fake-repo-first `PATH`.
Reject malformed entries, duplicate explicit keys, protected isolation keys, and
unknown explicit keys.

minimal_fix:
Replace `filteredTestAllEnv()` with one canonical hermetic env builder shared by
the fake `test-all` run helpers, resolve `bash` before spawning the child, move
test controls into an explicit allowlist, and add regressions for ambient
control leakage, shell startup injection, target-host report path leakage,
duplicate/key ordering, and concurrent fake-repo isolation.

verdict: `DENYLIST_CONTRACT_INSUFFICIENT`
