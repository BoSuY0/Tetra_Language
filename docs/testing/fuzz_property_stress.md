# Fuzz, Property, And Stress Suite

This is the short release-candidate robustness suite. It is intentionally small
enough for normal CI while leaving long fuzzing to nightly/release-candidate
runs.

## Normal Suite

```sh
go test ./compiler/... ./cli/... ./tools/... -run 'Fuzz|Property|Stress' -count=1
```

Coverage:

- lexer/parser seed fuzzing in `compiler/internal/frontend`
- linker object fuzzing in `compiler/internal/linker/linkcore`
- HTTP request-line parsing fuzzing in `compiler/internal/httprt`
- JSON string escaping fuzzing in `compiler/internal/jsonrt`
- PostgreSQL wire-frame parsing fuzzing in `compiler/internal/pgrt`
- manifest parser fuzz and negative property checks in `tools/cmd/validate-manifest`
- Eco capsule parser fuzzing in `cli/cmd/tetra`
- formatter idempotence property checks in `compiler`
- bounded actor/task stress examples in `compiler`

## Nightly Fuzz Commands

Run the one-command wrapper for agent and CI reproducibility:

```sh
bash scripts/dev/fuzz-nightly.sh --out-dir reports/fuzz-nightly
bash scripts/dev/fuzz-nightly.sh --short --out-dir reports/fuzz-nightly-smoke
```

The wrapper writes one log per surface under `logs/`, records `summary.md`,
creates `unstable-seeds.md`, and uses Go's package-local crasher archive path:
`<package>/testdata/fuzz/<FuzzName>/`.

The wrapper executes each fuzz command through fixed argv entries, not a
shell-assembled command string. Treat `--fuzztime` as untrusted input: it is
passed only as the literal `-fuzztime=<duration>` argument, and printable command
summaries are evidence strings, not shell input.

## Triage Protocol

Every crash, hang, timeout, or flaky seed must be triaged before a release
candidate can use the fuzz run as evidence:

- copy stable Go fuzz crashers from `<package>/testdata/fuzz/<FuzzName>/` into
  the smallest deterministic regression test that fails without fuzz mode;
- record unstable or non-deterministic seeds in `unstable-seeds.md` with package,
  fuzz target, seed/crasher path, owner, status, and next command;
- rerun the exact package with `go test <package> -run '^$' -fuzz=<FuzzName>
  -fuzztime=<duration>` to confirm the seed still reproduces before filing a
  release blocker;
- when a normal test flakes outside fuzz mode, rerun only the failing package
  with `go test <package> -run '<ExactTestName>' -count=3` before broad reruns;
- when the exact rerun passes all three attempts, keep the issue open as
  `flaky-unreproduced` with the original log path, rerun command, owner, and
  next observation window;
- when any exact rerun fails, reduce the failure to a deterministic regression
  test or mark the release candidate blocked with the failing command and log;
- mark a release candidate blocked if an unstable seed lacks either a
  deterministic regression test or an explicit owner and rerun command.

The underlying package commands are run one at a time so crashers are attributed
to the right surface:

```sh
go test ./compiler/internal/frontend -run '^$' -fuzz=FuzzLexer -fuzztime=10m
go test ./compiler/internal/frontend -run '^$' -fuzz=FuzzParser -fuzztime=10m
go test ./compiler/internal/linker/linkcore -run '^$' -fuzz=. -fuzztime=10m
go test ./compiler/internal/linker/linkcore -run '^$' -fuzz=FuzzLinkX64ObjectsDoesNotPanic -fuzztime=10m
go test ./compiler/internal/httprt -run '^$' -fuzz=FuzzHTTPParseRequest -fuzztime=10m
go test ./compiler/internal/jsonrt -run '^$' -fuzz=FuzzAppendStringProducesValidJSON -fuzztime=10m
go test ./compiler/internal/pgrt -run '^$' -fuzz=FuzzReadFrameDoesNotPanic -fuzztime=10m
go test ./tools/cmd/validate-manifest -run '^$' -fuzz=. -fuzztime=10m
go test ./cli/cmd/tetra -run '^$' -fuzz=FuzzParseCapsuleDoesNotPanic -fuzztime=10m
```

Crashers must be committed as deterministic regression tests before a release
candidate is promoted.

## Short Verification

```sh
bash scripts/dev/fuzz-nightly.sh --short --fuzztime 2s --out-dir reports/fuzz-nightly-smoke
```

For short verification, review `reports/fuzz-nightly-smoke/unstable-seeds.md`
even when all commands pass. An empty table means no unstable seeds were
observed in that run; it is not reusable evidence for a later branch state.

## Deterministic Rerun Evidence

Every fuzz, property, stress, or flaky-test triage note must include:

```text
original_command:
original_log:
package:
test_or_fuzz_target:
seed_or_case:
exact_rerun_command:
exact_rerun_result:
deterministic_regression_test:
owner:
status:
```

The `exact_rerun_result` field must state `failed reproducibly`, `passed 3/3
exact reruns`, or `blocked before rerun` with the blocking reason. Do not close a
release-candidate robustness row with only a broad suite rerun.
