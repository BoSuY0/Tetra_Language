# Getting Started With Tetra

Status: user guide for the current development baseline. Commands should be run
from the repository root.

## Bootstrap

Build the CLI pair:

```sh
bash scripts/bootstrap.sh
```

Check the active version:

```sh
./tetra version
./t version
```

The current public release scope is `docs/spec/v1_scope.md`. Do not treat old
v0.5/v0.6 documents as the current release truth unless they are explicitly
marked historical.

## Run A First Program

Use the tracked hello example as the first smoke input:

```sh
./tetra check examples/flow_hello.tetra
./tetra run examples/flow_hello.tetra
```

If you only need parser/semantic feedback, prefer `check`. Use `run` when you
also want host execution.

## Daily Commands

```sh
./tetra fmt --check examples lib __rt compiler/selfhostrt
./tetra test examples
./tetra targets
./tetra doctor
```

For repository-level confidence, use:

```sh
bash scripts/test_all.sh --full --keep-going
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

## Examples In Documentation

Examples in user docs are either real repository paths or are marked as
illustrative. Release-critical examples must be covered by the release gate
before they can be described as supported.
