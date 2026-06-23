## reviewed_commit

`70c07cbb48cbccacc2277172da298356d66d03f2`

Observed locally: `git rev-parse HEAD` in `/home/tetra/.codex/worktrees/Tetra_Language-stabilize-memory-core-v2` returned this exact commit, and `git status --short` was clean before this diagnostic document was written. Graphify report freshness also lists built commit `70c07cbb`.

## go_version

`go version go1.25.11 linux/amd64`

Go binary used by the recreated timeout run: `/home/tetra/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.11.linux-amd64/bin/go`.

## command

Primary timeout command under review:

```sh
go test -buildvcs=false ./cli/cmd/tetra -count=50 -timeout=10m -json
```

Observed artefacts:

- JSON timeout log: `/tmp/mcv2-cli-count50-timeout.json`
- Timing file: `/tmp/mcv2-cli-count50-timeout.time`
- Previous plain timeout log: `/tmp/mcv2-d003-d004-final-validation/01b-cli-count50-go125.log`

## package

`tetra_language/cli/cmd/tetra`

## default_timeout

`10m0s`

Observed facts:

- Previous plain timeout log: `panic: test timed out after 10m0s`, package `FAIL` elapsed `600.013s`.
- Recreated JSON timeout log: package `fail` elapsed `600.022s`, timing file `exit_status 1`, `wall_seconds 611`.

## test_count_multiplier

`-count=50`

The timeout command repeats the full package test list in-process 50 times. The JSON timeout log recorded `21852` individual test pass events before the package deadline fired.

## single_run_durations

Five independent package `count=1` runs, each in a new process with isolated `HOME`, `XDG_CACHE_HOME`, `GOCACHE`, `GOTMPDIR`, `TMPDIR`, `TMP`, and `TEMP`, and with `go clean -testcache` before the run:

| run | package result | package elapsed | wall seconds | test pass events |
| --- | --- | ---: | ---: | ---: |
| run_1 | PASS | 20.995s | 32s | 738 |
| run_2 | PASS | 21.733s | 33s | 738 |
| run_3 | PASS | 21.021s | 32s | 738 |
| run_4 | PASS | 22.763s | 35s | 738 |
| run_5 | PASS | 21.665s | 33s | 738 |

Median package elapsed: `21.665s`.

## projected_count_50_duration

`1083.250s`

Inference: `21.665s median_single_run_seconds * 50 = 1083.250s`, which is greater than the configured `600s` package timeout. This projection uses package elapsed, not dependency download time; the recorded wall seconds additionally include cold compile overhead.

## exact_test_result

Source inspection:

- `cli/cmd/tetra/tetra_suite_test.go:12356` defines `testCommand`, calls `findRepoRoot`, creates `exec.Command(name, args...)`, and sets `cmd.Dir = root`.
- `cli/cmd/tetra/tetra_suite_test.go:12561` defines `TestEcoLockFixtureRejectsGraphHashMismatch`.
- `cli/cmd/tetra/tetra_suite_test.go:12577` runs `go run ./tools/cmd/validate-eco-lock --lock <fixture>`.
- `cli/cmd/tetra/tetra_suite_test.go:12578` calls `cmd.CombinedOutput()`.
- Fixture path: `cli/cmd/tetra/testdata/eco_capsules/matrix/lock_mismatch/tetra.lock.json`.

Observed exact isolated results for `TestEcoLockFixtureRejectsGraphHashMismatch`:

- `count=1`: `go test -buildvcs=false ./cli/cmd/tetra -run '^TestEcoLockFixtureRejectsGraphHashMismatch$' -count=1 -timeout=2m -v` passed, package elapsed `0.230s`, shell wall `10s` cold compile.
- `count=100`: `/tmp/mcv2-eco-lock-count100.json` passed `100/100`, package elapsed `4.578s`; per-test min `0.040s`, median `0.040s`, max `0.270s`, mean `0.043s`; shell wall `15s` cold compile.
- `race count=20`: `/tmp/mcv2-eco-lock-race-count20.log` passed `20/20`, package PASS, package elapsed `5.131s`, shell wall `26s`; no `WARNING: DATA RACE` was present.
- In recreated package `count=50` timeout log, `TestEcoLockFixtureRejectsGraphHashMismatch` had already passed `30` times before timeout: min `0.040s`, median `0.040s`, max `0.050s`, mean `0.041s`.

## in_process_repeat_result

Observed in-process package repeats:

| command shape | package result | package elapsed | wall seconds | pass events | EcoLock median |
| --- | --- | ---: | ---: | ---: | ---: |
| `go test -buildvcs=false ./cli/cmd/tetra -count=2 -timeout=20m -json` | PASS | 40.595s | 51s | 1476 | 0.040s |
| `go test -buildvcs=false ./cli/cmd/tetra -count=5 -timeout=30m -json` | PASS | 102.748s | 114s | 3690 | 0.050s |

Inference: package elapsed scales approximately linearly against the `21.665s` median independent single-run duration: `count=2` is about `1.87x`, and `count=5` is about `4.74x`. No timeout, race, or deadlock strings were found in the `count=2`, `count=5`, exact `count=100`, race `count=20`, or independent `count=1` logs inspected.

## independent_process_result

Five independent package `count=1` runs all passed:

- `/tmp/mcv2-cli-count1-run1.json`: package PASS `20.995s`, `738` test pass events, wall `32s`.
- `/tmp/mcv2-cli-count1-run2.json`: package PASS `21.733s`, `738` test pass events, wall `33s`.
- `/tmp/mcv2-cli-count1-run3.json`: package PASS `21.021s`, `738` test pass events, wall `32s`.
- `/tmp/mcv2-cli-count1-run4.json`: package PASS `22.763s`, `738` test pass events, wall `35s`.
- `/tmp/mcv2-cli-count1-run5.json`: package PASS `21.665s`, `738` test pass events, wall `33s`.

Observed: no `panic: test timed out`, `WARNING: DATA RACE`, `fatal error`, or `FAIL` strings were found across these five independent JSON logs.

## timeout_stack_interpretation

Observed facts:

- Previous plain timeout log had no JSON pass events because it was a plain `go test` log. At the deadline it reported `running tests: TestEcoLockFixtureRejectsGraphHashMismatch (0s)`.
- That plain stack showed the EcoLock test waiting in `os.(*Process).Wait` via `os/exec.(*Cmd).CombinedOutput` at `cli/cmd/tetra/tetra_suite_test.go:12578`, plus an `IO wait` goroutine owned by `os/exec` output copying.
- Recreated JSON timeout log reported `running tests: TestNewSurfaceAppScaffoldCreatesRunnableBlockMorphProject (1s)`, not EcoLock.
- The recreated JSON stack had only three goroutine headers: the testing alarm goroutine `[running]`, goroutine 1 `[chan receive]` in the normal test runner wait path, and an active test goroutine `[runnable]`.
- The active recreated stack was in `tetra_language/compiler/internal/memorypipeline.normalizeFunction`, `normalizePLIR`, `(*State).ModulePlanDigest`, `compiler.planNativeModuleBuild`, `BuildFileWithStatsOpt`, `runRun`, and `TestNewSurfaceAppScaffoldCreatesRunnableBlockMorphProject` at `cli/cmd/tetra/tetra_suite_test.go:16138`.
- Source at `cli/cmd/tetra/tetra_suite_test.go:16138` is the scaffold test's `runCLI([]string{"run", "--target", mustHostTarget(t), appDir}, ...)` step.

Inference: the displayed active test at the instant the 10-minute package alarm fires is not stable across runs. One timeout snapshot caught EcoLock while it was waiting for a short-lived `go run` subprocess; the recreated timeout caught a different test doing runnable compiler digest work. This supports a cumulative package deadline classification rather than a repeated single-test hang.

## blocked_goroutine

No repeated blocked goroutine was confirmed.

Observed:

- Plain log: one timeout snapshot showed `TestEcoLockFixtureRejectsGraphHashMismatch` in `os.(*Process).Wait`/`CombinedOutput`, with an `IO wait` copier goroutine. That is a process-wait snapshot, not by itself evidence of an unreleased lock or channel deadlock.
- Recreated JSON log: no EcoLock process-wait goroutine appeared at the deadline; the active non-alarm goroutine was `[runnable]` in compiler/memorypipeline digest work.
- Exact EcoLock `count=100`, EcoLock race `count=20`, package `count=2`, package `count=5`, and five independent package `count=1` runs all passed.

## resource_leak_evidence

No confirmed resource leak evidence was found in the inspected artefacts.

Observed/provided facts:

- No orphan subprocesses matching `validate-eco-lock`, `validate-eco`, `go test`, or `tetra.test` remained after the repeat runs.
- Temp artifacts under isolated `TMPDIR` were Chromium temp dirs and scaled linearly: `count=2` had `tmp_files 60 / 80K`, `count=5` had `tmp_files 150 / 200K`.
- `GOTMPDIR` files were `0`.
- Exact EcoLock temp/GOTMP files were `0`.
- The logs inspected for `count=2`, `count=5`, exact EcoLock `count=100`, race `count=20`, and independent package `count=1` contained no timeout, data race, fatal error, or package failure evidence.

Inference: the available evidence does not support an orphan process, unreleased lock/channel, or accumulating temp-file leak as the timeout cause.

## root_cause

Observed root cause evidence: the full `./cli/cmd/tetra` package suite takes about `21-23s` per clean independent `count=1` package run, `count=2` and `count=5` scale approximately linearly, and the `count=50` command uses a `10m` package timeout. The projected package elapsed for `count=50` is `1083.250s`, well above the `600s` timeout. The actual timeout occurs at approximately the configured package deadline: `600.013s` in the plain log and `600.022s` in the recreated JSON log.

Inference: the `-count=50 -timeout=10m` command is under-budgeted for the cumulative runtime of this package. The timeout snapshot names whichever test is active when the global package alarm fires. EcoLock-specific hang is not supported by the exact targeted repeat, race repeat, in-process package repeats, independent package runs, or recreated timeout stack.

## verdict

CUMULATIVE_SUITE_TIMEOUT_CONFIRMED
