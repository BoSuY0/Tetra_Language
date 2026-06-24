# D-001 Post-Fix CI Causality Review

reviewed_commit: `db821745b09a44c46f0c70246b87b5a4c3f3996f`

ci_runs:
- push: `28086494161`
- pull_request: `28086496364`

ci_jobs:
- push: `83153880490`, `full-platform-ui-runtime-gate-linux`
- pull_request: `83153905910`, `full-platform-ui-runtime-gate-linux`

push_failure:
`TestTestAllQuickFailsWhenHostLeakBlockerSuiteMissing` expected one explicit
host-leak failure, but the summary had two failures: the expected
`host leak blocker suite` plus an unexpected `bounds proof blocker suite`.

pr_failures:
- `TestTestAllFullRunsDocsManifestDiffStep` passed no fake failure control, but
  failed first at `unsafe promotion blocker suite` before reaching the docs
  manifest diff assertion.
- `TestTestAllFullValidatesCrossTargetSmokeReports` passed only
  `TETRA_FAKE_GO_LOG`, but failed first at `bounds proof blocker suite` before
  reaching cross-target smoke report validation.

explicit_test_controls:
- The push witness explicitly passed `TETRA_FAKE_SKIP_HOST_LEAK_LIST=1`.
- The docs-manifest witness passed no fake failure control.
- The cross-target witness explicitly passed only `TETRA_FAKE_GO_LOG`.
- No `t.Parallel` usage was found in `tools/scriptstest/test_all`.

unexpected_controls:
- Effective unexpected `TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST` behavior in the push
  witness.
- Effective unexpected `TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST` behavior in the
  PR docs-manifest witness.
- Effective unexpected `TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST` behavior in the PR
  cross-target witness.

ambient_channels:
At the reviewed commit, fake-repo helpers still inherited `os.Environ()` minus
only `TETRA_FAKE_*` and `TETRA_FAIL_*`. Ambient channels still included
`BASH_ENV`, `ENV`, `HOME`, `XDG_CACHE_HOME`, `GOENV`, `GOFLAGS`, `GOCACHE`,
`GOTMPDIR`, `TMPDIR`, `TMP`, `TEMP`, `CI`, `GITHUB_*`, and target-host report
variables from the fan-in job.

filesystem_channels:
Fake repos, report dirs, and fake-go logs are created under per-test temp
directories, so the logs did not prove shared report-dir or fake-go-log
contamination. Shared host `HOME`, cache, Go temp, and shell startup inheritance
remained valid open channels at the reviewed commit.

process_channels:
GitHub Actions bash step ->
`scripts/release/full_platform/ui-runtime-gate.sh` ->
baseline `go test ./compiler/... ./cli/... ./tools/... -count=1` ->
`tools/scriptstest/test_all` test binary ->
`bash scripts/ci/test-all.sh` ->
fake repo tools.

tests_are_witnesses: true

independent_new_defect: false

root_cause:
D-001 isolation remained incomplete after commit `9ca65b6...`. The partial fix
blocked direct ambient `TETRA_FAKE_*` and `TETRA_FAIL_*` inheritance, but the
fake-repo subprocess environment still inherited non-hermetic shell, Go, Git,
cache, temp, CI, and report-path state.

remaining_uncertainty:
The CI logs do not include a full sanitized child environment snapshot, so they
do not prove the exact carrier variable for each effective fake control. This
does not block the fix because the reviewed contract allowed multiple ambient
channels and the requested closure is hermetic construction rather than a wider
denylist.

verdict: `D001_INCOMPLETE_ISOLATION_CONFIRMED`
