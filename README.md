# Tetra Language (v0.4.0)

Tetra is a systems programming language with region-based memory management
through Islands. This repository is the working local compiler/toolchain, not
the full future Tetra platform.

The current public profile is **v0.4.0**. It keeps the verified local compiler,
tooling, T4 source format, interface, runtime, validator, release-gate
foundation, and the promoted v0.4 language/tooling slices. Older v0.5/v0.6
labels are historical checkpoints, and `v1.0.0` is a future release label.

## Current Truth

- Current supported surface: `docs/spec/current_supported_surface.md`
- Previous minor scope: `docs/spec/v0_3_scope.md`
- Current minor scope: `docs/spec/v0_4_scope.md`
- Future v1 contract: `docs/spec/v1_scope.md`
- Current release checklist: `docs/checklists/v0_4_0_release_gate.md`
- Current release gate: `scripts/release/v0_4_0/gate.sh`
- Current release handoff: `docs/release/v0_4_0_final_handoff.md`

Do not treat future or compatibility `v1_0` filenames as proof that this branch
is ready for the `v1.0.0` label.

## Quick Start

```sh
bash scripts/dev/bootstrap.sh
./tetra version
./t version
./tetra check examples/flow_hello.tetra
./tetra run examples/flow_hello.tetra
```

Bootstrap writes both local entrypoints: `./tetra` and the short alias `./t`.

## Common Commands

```sh
./tetra fmt --check examples lib __rt compiler/selfhostrt
./tetra test examples
./tetra targets --format=json
./tetra doctor --format=json
./tetra smoke --target linux-x64 --run=true --report reports/smoke-linux.json
```

For a compact user reference, see `docs/user/cli_cheatsheet.md`. The normative
command, exit-code, diagnostics, and JSON report contract is
`docs/spec/cli_contracts.md`.

## Project Workflow

New source files should use the T4 source format (`.t4`). Legacy `.tetra` files
remain accepted for existing examples and smoke coverage.

```sh
./tetra new app --lock DemoApp
./tetra project sync --check DemoApp
./tetra check DemoApp
./tetra build DemoApp
./tetra run DemoApp
./tetra test DemoApp
```

Project roots use `Capsule.t4`. Generated interface files use `.t4i` and carry a
public API hash validated by interface, check, build, and graph-loading paths.

## Verification

Fast local checks:

```sh
bash scripts/ci/test.sh
bash scripts/ci/test-all.sh --quick
```

Broader stabilization checks:

```sh
bash scripts/ci/test-all.sh --full --keep-going
bash scripts/ci/test-all.sh --stabilization --keep-going
bash scripts/dev/fuzz-nightly.sh --short --out-dir reports/fuzz-nightly-smoke
```

Docs and manifest checks:

```sh
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
```

`scripts/ci/test.sh` is a non-mutating Go test and formatting gate. Use explicit
formatting commands when you intend to rewrite files.

## Documentation Map

- Getting started: `docs/user/getting_started.md`
- Current status: `docs/user/status.md`
- v0.3 preview boundaries: `docs/user/v0_3_preview.md`
- Tutorial path: `docs/user/tutorial_path.md`
- CLI cheatsheet: `docs/user/cli_cheatsheet.md`
- Troubleshooting: `docs/user/troubleshooting.md`
- Examples index: `docs/user/examples_index.md`
- Language tour: `docs/user/language_tour.md`
- Standard library guide: `docs/user/standard_library_guide.md`
- Standard library spec: `docs/spec/stdlib.md`
- Flow syntax: `docs/spec/flow_syntax_v1.md`
- Ownership and effects: `docs/user/ownership_effects_guide.md`
- Async and actors: `docs/user/async_actors_guide.md`
- WASM/UI guide: `docs/user/wasm_ui_guide.md`
- Eco package guide: `docs/user/eco_package_guide.md`
- Compiler pipeline map: `docs/contributing/compiler_pipeline.md`

## Developer Utilities

Create a curated repository dump for agents:

```sh
bash scripts/dev/dump-project.sh
```

The wrapper runs `go run ./tools/cmd/dump-project`, prints the release artifact
marker, and writes the timestamped dump under `dumps/`.
Useful flags include `--all`, `--only <prefix>`, and
`--exclude-prefix <prefix>`.
