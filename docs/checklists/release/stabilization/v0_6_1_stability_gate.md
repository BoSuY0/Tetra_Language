# v0.6.1 Stability Gate

Use this checklist before labeling a build or branch as the first 0.6.x
stabilization point release.

## Wrapper

- [x] `bash scripts/ci/test-all.sh --help` documents `--keep-going`, `--json-only`,
      `--report-dir`, and exit codes.
- [x] `bash scripts/ci/test-all.sh --quick --json-only` emits valid JSON.
- [x] `summary.json` records each step `exit_code` for pass/fail CI consumers.
- [x] `summary.json` records top-level `step_count` and `failed_count`.
- [x] `summary.json` is validated against step counts, exit codes, and log
      artifacts before emission.
- [x] `summary.json` rejects duplicate step names and duplicate step log paths.
- [x] Failure-path summaries are still emitted when summary validation itself
      fails, with a warning instead of masking the original failed check.
- [x] `--keep-going --json-only` is covered by a fake-repo CLI test and exits
      `1` after recording later steps.

## Regression Coverage

- [x] Optional, typed-error, ownership, effects, protocol, task, and JSON
      diagnostic negative tests are present.
- [x] Eco duplicate ID, target mismatch, and corrupt vault object tests are
      present.
- [x] Smoke report JSON shape is decoded in CLI tests.
- [x] `tetra test --report=json` is validated for array shape and aggregate
      counts in full/release gates.
- [x] `tetra test --report=json` validation rejects duplicate test names per
      file and negative durations.

## Required Commands

- [ ] `go test ./compiler/... ./cli/... ./tools/...`
- [ ] `bash scripts/ci/test-all.sh --quick`
- [ ] `bash scripts/ci/test-all.sh --full`
- [ ] `bash scripts/release/v0_6/gate.sh`
- [ ] `git diff --check`
