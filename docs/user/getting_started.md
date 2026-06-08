# Getting Started With Tetra

Status: user guide for the current development baseline. Commands should be run
from the repository root.

## Bootstrap

If you are new to this repository, first read `docs/user/status.md` and follow
the ordered path in `docs/user/tutorial_path.md`.

Build the CLI pair:

```sh
bash scripts/dev/bootstrap.sh
```

Check the active version:

```sh
./tetra version
./t version
```

The current public release scope is `docs/spec/current_supported_surface.md`.
Use `docs/spec/v1_scope.md` only as the future major-release contract. Old
v0.5/v0.6 documents are historical unless they explicitly say otherwise.

## Run A First Program

Use the tracked hello example as the first smoke input:

```sh
./tetra check examples/flow_hello.tetra
./tetra build --target linux-x64 -o app examples/flow_hello.tetra
```

`check` and `build` are the normal user loop. `check` gives parser and semantic
feedback without writing an artifact; `build` applies the same safety contract
and emits the program. Use `run` as a convenience command when you want build
plus host execution in one step.

## Daily Commands

For a compact command reference, see `docs/user/cli_cheatsheet.md`.

```sh
./tetra fmt --check examples lib __rt compiler/selfhostrt
./tetra test examples
./tetra targets
./tetra doctor
```

For repository-level confidence, use:

```sh
bash scripts/ci/test-all.sh --full --keep-going
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

## Multi-Module Layout

Use `Capsule.t4` as the project root when a program has more than one source
root. The CLI discovers it automatically for `check`, `build`, `run`, `test`,
and `doc`.

Start a new project with:

```sh
tetra new app --lock DemoApp
cd DemoApp
tetra project sync --check .
tetra project deps list .
tetra workspace init ..
tetra workspace add DemoApp --workspace ..
tetra workspace check ..
tetra check .
tetra build .
tetra project info --format=json .
tetra doctor --format=json .
```

Add a local capsule dependency with:

```sh
tetra project deps add --path ../Math .
tetra project deps check .
tetra project sync .
```

`deps add` writes the `deps:` entry in `Capsule.t4`; `project sync` refreshes
`Tetra.lock` and generated dependency artifacts after the manifest changes.

Manage multiple local capsules with:

```sh
tetra workspace init
tetra workspace add App
tetra workspace add Math
tetra workspace check
tetra workspace graph --format=json
tetra workspace sync
tetra workspace build --target linux-x64
tetra workspace test --target linux-x64
tetra workspace run App
```

`Tetra.workspace` is a local member list. Dependency declarations still live in
each member's `Capsule.t4`; `workspace sync`, `workspace build`, and
`workspace test` run members in dependency order, while `workspace run <member>`
runs one selected app from the workspace root.

Example layout:

```text
Capsule.t4
src/app/main.t4
ui/components/counter.t4
```

```tetra
// Capsule.t4
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/app/main.t4"

    sources:
        src
        ui
```

When the project has generated dependencies, `Capsule.t4` can also bind them:

```tetra
artifacts:
    interface interfaces/math/core.t4i
    object linux-x64 artifacts/math-core.linux-x64.tobj
    seed seeds/tetra-core.t4s
```

Run `tetra eco artifacts build --target linux-x64 --lock Tetra.lock Capsule.t4`
to generate dependency artifacts from local path dependencies, or prefer the
project-first wrapper `tetra project sync --target linux-x64 .`. Run
`tetra project sync .` when you only need the normal discovered project flow:
it refreshes `Tetra.lock` and generated local dependency artifacts together. If
`Tetra.lock` is present, `check`, `build`, and `run` validate it before
compiling.

Useful artifact maintenance commands:

```sh
tetra project deps list .
tetra project deps check .
tetra workspace check .
tetra workspace sync .
tetra workspace build --target linux-x64 .
tetra workspace test --target linux-x64 .
tetra project sync .
tetra project sync --check .
tetra project sync --all-targets .
tetra eco artifacts check --target linux-x64 --lock Tetra.lock Capsule.t4
tetra eco artifacts build --check --target linux-x64 --lock Tetra.lock Capsule.t4
tetra eco artifacts build --all-targets --lock Tetra.lock Capsule.t4
tetra build --artifacts=auto
```

`check` validates freshness and prints repair commands. `build --check` is a
dry-run and writes nothing. `--all-targets` generates native object artifacts
for every native target in `Capsule.t4`, skipping WASM object targets.
`build --artifacts=auto` repairs project artifacts before compiling;
plain `build` remains strict and reports stale declared artifacts.

```tetra
// src/app/main.t4
module app.main
import components.counter.{value}

func main() -> Int:
    return value()
```

```tetra
// ui/components/counter.t4
module components.counter

pub func value() -> Int:
    return 42
```

From that directory, this is enough:

```sh
./tetra check
./tetra build
```

The path-to-module rule is strict inside each source root: `app.main` maps to
`src/app/main.t4`, and `components.counter` maps to
`ui/components/counter.t4`. Legacy `.tetra` files remain accepted for existing
projects. Missing imports, duplicate module declarations, and import cycles are
rejected during `./tetra check` / `./tetra build`.

`pub` defines the module surface. Modules without any `pub` declarations keep
the older public-by-default behavior, but once a module uses `pub`, its other
functions and types are private to that module. Use `pub import x.y.{Name}` to
re-export selected public symbols through a facade module.

Generate a lightweight interface file when another project only needs the
surface. The generated `.t4i` includes a stable public API hash:

```sh
./tetra interface -o ui/components/counter.t4i ui/components/counter.t4
./tetra interface --check -o ui/components/counter.t4i ui/components/counter.t4
```

Use interface-only checks for API graphs that intentionally do not have an
executable entry point or full source implementation available:

```sh
./tetra check --interface-only ui/components/counter.t4
./tetra build --interface-only ui/components/counter.t4
```

## Examples In Documentation

Examples in user docs are either real repository paths or are marked as
illustrative. Release-critical examples must be covered by the release gate
before they can be described as supported.
