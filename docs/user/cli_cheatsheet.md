# Tetra CLI Cheatsheet

Status: user-facing quick reference. The normative command contract lives in
`docs/spec/cli_contracts.md`.

Run commands from the repository root unless a project path is shown.

The core user compiler workflow is intentionally small: use `check` for fast
semantic feedback and `build` for artifacts. The other commands below are
tooling around that workflow; they are not extra safety levels.

## Bootstrap And Version

```sh
bash scripts/dev/bootstrap.sh
./tetra version
./t version
```

`./t` is the short alias written by bootstrap.

## Single-File Workflow

```sh
./tetra check examples/flow_hello.tetra
./tetra build --target linux-x64 -o app examples/flow_hello.tetra
```

Use `run` only when you want build plus host execution in one command:

```sh
./tetra run --target linux-x64 examples/flow_hello.tetra
```

## Project Workflow

```sh
./tetra new app --lock DemoApp
./tetra project sync --check DemoApp
./tetra check DemoApp
./tetra build DemoApp
```

Project discovery prefers `Capsule.t4`. If no explicit input is given, the CLI
looks for the project entry and then `main.t4` before legacy `main.tetra`.

## Workspace Workflow

```sh
./tetra workspace init examples/projects
./tetra workspace add hello_t4 --workspace examples/projects
./tetra workspace list examples/projects --format=json
./tetra workspace check examples/projects
./tetra workspace build --target linux-x64 examples/projects
./tetra workspace test --target linux-x64 examples/projects
./tetra workspace run hello_t4 --workspace examples/projects --target linux-x64 --artifacts=auto
```

Workspace commands run members in dependency order when the member graph is
available. `workspace run --artifacts=auto` refreshes the selected member's
lock and generated local dependency artifacts before running it; the default is
strict validation.

## Formatting, Tests, And Docs

```sh
./tetra fmt --check examples lib __rt compiler/selfhostrt
./tetra test examples
./tetra test --report=json examples
./tetra doc examples
./tetra doc -o docs/api.md examples
```

Use `./tetra fmt --write <paths>` only when you intentionally want to rewrite
sources.

## Targets, Doctor, And Smoke

```sh
./tetra targets
./tetra targets --format=json
./tetra doctor
./tetra doctor --format=json
./tetra actor-net --addr 127.0.0.1:47777 --report reports/actornet.json
./tetra smoke --list --format=json
./tetra smoke --list --format=toon
./tetra smoke --target linux-x64 --run=true --report reports/smoke-linux.json
./tetra smoke --target linux-x64 --run=false --report reports/smoke-linux.json --report-format=both
```

The current target truth is documented in
`docs/spec/current_supported_surface.md`.
In JSON target reports, `run_supported` means `tetra run --target <triple>` is
runnable in the current environment. Native targets require a matching host.
`wasm32-wasi` becomes `run_supported: true` when a supported WASI runner is
discoverable. `wasm32-web` becomes `run_supported: true` when a
Chromium-compatible browser runner is discoverable. Missing runners are
reported with `run_unsupported_reason`.

`actor-net` runs the loopback TCP broker used by Linux-x64 distributed actor
runtime smokes. Its report is broker telemetry; the production distributed
runtime gate also needs executable node-process evidence validated by
`go run ./tools/cmd/validate-distributed-actor-runtime --report <path>`.

## Cache Cleanup

```sh
./tetra clean
./tetra clean --target linux-x64
```

Use `clean` to remove local `.tetra_cache` and `tetra_cache` directories from
the current working directory. Use `clean --target <triple>` for
target-specific cleanup; it removes only `.tetra_cache/<triple>` and
`tetra_cache/<triple>`, leaving cache entries for other targets in place.

## Interfaces And Objects

```sh
./tetra interface -o examples/flow_hello.t4i examples/flow_hello.tetra
./tetra interface --check -o examples/flow_hello.t4i examples/flow_hello.tetra
./tetra build --emit=object -o app.tobj examples/flow_hello.tetra
./tetra build --link-object app.tobj -o app examples/flow_hello.tetra
```

`.t4i` files carry public API hashes and are validated during graph loading.

## Eco And Artifacts

```sh
./tetra eco verify --target linux-x64 --lock Tetra.lock Capsule.t4
./tetra eco artifacts check --target linux-x64 --lock Tetra.lock Capsule.t4
./tetra eco artifacts build --check --target linux-x64 --lock Tetra.lock Capsule.t4
./tetra eco artifacts build --all-targets --lock Tetra.lock Capsule.t4
./tetra workspace run App --workspace examples/projects --artifacts=auto
tmp_dir="$(mktemp -d)"
./tetra eco pack --project examples/projects/hello_t4/Capsule.t4 -o "$tmp_dir/package.tdx"
./tetra eco unpack "$tmp_dir/package.tdx" -C "$tmp_dir/unpacked"
go run ./tools/cmd/validate-eco-unpack --dir "$tmp_dir/unpacked"
```

Use `--check` for dry-run artifact validation. Use build modes only when you
intend to create or refresh local artifacts. `workspace run --artifacts=auto`
does the same refresh for the selected workspace member before execution.
For `eco pack`, the default without `--project` writes only the manifest into
the Todex archive; use `--project` for a source-bearing project bundle that
passes `validate-eco-unpack`. `.todex` remains accepted as a compatibility
alias for local package paths.

## Diagnostics And Reports

```sh
./tetra check --diagnostics=json examples/flow_hello.tetra
./tetra build --diagnostics=json examples/flow_hello.tetra
./tetra test --report=json examples
./tetra test --report=toon examples
```

JSON diagnostics use stable fields for tooling: `code`, `message`, `file`,
`line`, `column`, `severity`, and optional `hint`.

Validated diagnostic JSON is a single object. Use:

```sh
go test ./tools/cmd/validate-diagnostic/... -count=1
```

The validator accepts severities `error`, `warning`, `info`, and `hint`, and it
requires `file`, `line`, and `column` to appear together when a source position
is present.

TOON is opt-in for selected structured reports. JSON remains the default and
canonical format, while `--report-format=both` writes a `.toon` mirror beside
the JSON path when that command supports it.

## Plan250 CLI Evidence

The Wave-A docs closure cites:

- `reports/plan250/cli-tools/targets.json`: supported targets
  `linux-x64`, `windows-x64`, `macos-x64`; runner-gated WASM targets
  `wasm32-wasi`, `wasm32-web`; no planned targets.
- `reports/plan250/cli-tools/smoke-list.json`: `linux-x64` smoke list with
  `total: 62`, `build_only: false`, and `run_supported: true`.
- `reports/plan250/cli-tools/tetra-test-report.json`: valid empty test report
  with `total: 0`, `passed: 0`, `failed: 0`.

## Release-Maintainer Commands

```sh
bash scripts/ci/test.sh
bash scripts/ci/test-all.sh --quick
bash scripts/ci/test-all.sh --full --keep-going
bash scripts/ci/test-all.sh --stabilization --keep-going
bash scripts/dev/fuzz-nightly.sh --short --out-dir reports/fuzz-nightly-smoke
```

Final release proof must use the release checklist and gate for the active line,
not this cheatsheet.
