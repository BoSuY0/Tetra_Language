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
- manifest parser fuzz and negative property checks in `tools/cmd/validate-manifest`
- Eco capsule parser fuzzing in `cli/cmd/tetra`
- formatter idempotence property checks in `compiler`
- bounded actor/task stress examples in `compiler`

## Nightly Fuzz Commands

Run the one-command wrapper for agent and CI reproducibility:

```sh
bash scripts/fuzz_nightly.sh --out-dir reports/fuzz-nightly
bash scripts/fuzz_nightly.sh --short --out-dir reports/fuzz-nightly-smoke
```

The wrapper writes one log per surface under `logs/`, records `summary.md`, and
uses Go's package-local crasher archive path:
`<package>/testdata/fuzz/<FuzzName>/`.

The underlying package commands are run one at a time so crashers are attributed
to the right surface:

```sh
go test ./compiler/internal/frontend -run '^$' -fuzz=FuzzLexer -fuzztime=10m
go test ./compiler/internal/frontend -run '^$' -fuzz=FuzzParser -fuzztime=10m
go test ./compiler/internal/linker/linkcore -run '^$' -fuzz=. -fuzztime=10m
go test ./tools/cmd/validate-manifest -run '^$' -fuzz=. -fuzztime=10m
go test ./cli/cmd/tetra -run '^$' -fuzz=FuzzParseCapsuleDoesNotPanic -fuzztime=10m
```

Crashers must be committed as deterministic regression tests before a release
candidate is promoted.

## Short Verification

```sh
bash scripts/fuzz_nightly.sh --short --fuzztime 1s --out-dir reports/fuzz-nightly-smoke
```
