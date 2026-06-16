# Eco Package Guide

Status: user guide for `v0.4.0` Eco/Todex workflows.

This guide describes what is release-covered today for local capsule/package
work plus the constrained HTTP TetraHub fetch contract. It does not claim a
distributed mesh, a hosted production TetraHub service, or global trust
federation. The current support boundary is `docs/spec/current_supported_surface.md`.

Terminology note:
- `Capsule.t4` / `Tetra.capsule` in this guide are Eco project/package manifest
  files.
- They are distinct from source-language top-level `capsule Name: ...`
  declarations, which are compile-time language metadata.
- Todex package files use `.tdx` as the primary extension. `.todex` remains a
  compatibility alias for existing local package paths; new examples and docs
  should prefer `.tdx`.

## Supported Scope (v0.4.0)

Supported local commands:

- `./tetra formats`
- `./tetra project deps list [path]`
- `./tetra project deps add --path <dep-project-path> [path]`
- `./tetra project deps remove --id <id> [path]`
- `./tetra project deps check [path]`
- `./tetra project sync [path]`
- `./tetra project sync --check [path]`
- `./tetra project sync --target <triple> [path]`
- `./tetra project sync --all-targets [path]`
- `./tetra workspace init [path]`
- `./tetra workspace add <member-path> [--workspace <workspace-path>]`
- `./tetra workspace remove <member-path> [--workspace <workspace-path>]`
- `./tetra workspace list|check|graph [path] [--format=json]`
- `./tetra workspace sync [path] [--target <triple>] [--all-targets] [--check] [--jobs <n>]`
- `./tetra workspace build [path] [--target <triple>] [--all-targets] [--format=json] [-o <out-dir>]`
- `./tetra workspace test [path] [--target <triple>] [--fail-fast] [--format=json]`
- `./tetra workspace run <member-path> [--workspace <workspace-path>] [--target <triple>]`
- `./tetra eco verify --target <triple> --lock Tetra.lock <capsules...>`
- `./tetra eco verify --lock Tetra.lock Capsule.t4` for a project capsule with
  local path dependencies
- `./tetra eco artifacts build --target linux-x64 --lock Tetra.lock Capsule.t4`
- `./tetra eco artifacts build --check --target linux-x64 --lock Tetra.lock Capsule.t4`
- `./tetra eco artifacts build --all-targets --lock Tetra.lock Capsule.t4`
- `./tetra eco artifacts check --target linux-x64 --lock Tetra.lock Capsule.t4`
- `./tetra build --artifacts=auto`
- `./tetra eco seed export --out tetra-core.t4s <capsules...>`
- `./tetra eco seed import --seed tetra-core.t4s --lock Tetra.lock`
- `./tetra eco needmap --lock Tetra.lock -o missing.tneed`
- `./tetra eco pack Capsule.t4 -o manifest-only.tdx`
- `./tetra eco pack --project Capsule.t4 -o app.tdx`
- `./tetra eco unpack app.tdx -C out`
- `./tetra eco vault add|list|verify --store .tetra/todex-vault ...`
- `./tetra eco trust snapshot --lock Tetra.lock --store .tetra/todex-vault -o trust.snapshot.json`
- `./tetra eco materialize app.tdx [--target <triple>] --trust trust.snapshot.json -C out`
- `./tetra eco publish --package app.tdx --registry .tetra/registry-beta [--target <triple>] [--trust trust.snapshot.json]`
- `./tetra eco publish --package app.tdx --registry .tetra/registry-stable --channel stable [--target <triple>] [--trust trust.snapshot.json]`
- `./tetra eco download --id <id> --version <x.y.z> --target <triple> --registry .tetra/registry-beta -o app.tdx`
- `./tetra eco tetrahub publish|download ... --store .tetra/tetrahub-beta`
- `./tetra eco tetrahub publish --channel stable ... --store .tetra/tetrahub-stable`
- `./tetra eco tetrahub mirror --from .tetra/tetrahub-stable --to .tetra/tetrahub-mirror --id <id> --version <x.y.z> --target <triple> -o tetra.eco.mirror.json`
- `./tetra eco tetrahub fetch --url http://127.0.0.1:8080/tetrahub --to .tetra/tetrahub-cache --id <id> --version <x.y.z> --target <triple> -o tetra.eco.mirror.json`

Publishing boundary:

- `tetra eco publish --channel stable` writes local production metadata with
  schema `tetra.eco.publish.v1`.
- `tetra eco tetrahub publish --channel stable` writes the same production
  metadata schema with a `tetrahub-stable` hub label.
- `tetra eco tetrahub mirror` validates a local TetraHub source package,
  preserves package/metadata/trust bytes in the destination store, and writes a
  `tetra.eco.mirror.v1` report.
- `tetra eco tetrahub fetch` performs the same package/metadata/trust integrity
  checks over a single HTTP(S) TetraHub store URL, then writes the validated
  bytes into a local destination store and emits the same `tetra.eco.mirror.v1`
  report schema.
- beta publish/download and `tetra eco tetrahub` remain local beta
  metadata/store paths.
- none of these commands are claims about global discovery, federation,
  consensus, or a distributed mesh.

## Validator Contracts

Release checks use machine-validated artifacts:

- `go run ./tools/cmd/validate-eco-lock --lock Tetra.lock`
- `go run ./tools/cmd/validate-eco-unpack --dir out`
- `go run ./tools/cmd/validate-eco-vault --store .tetra/todex-vault`
- `go run ./tools/cmd/validate-eco-publish --registry .tetra/registry-beta --id <id> --version <x.y.z> --target <triple>`
- `go run ./tools/cmd/validate-eco-publish --registry .tetra/registry-stable --id <id> --version <x.y.z> --target <triple> --channel stable`
- `go run ./tools/cmd/validate-eco-mirror --mirror tetra.eco.mirror.json`

The validators enforce schema/path/hash constraints, reject malformed metadata,
and keep local workflows deterministic.

Selected Eco report artifacts support TOON as an opt-in mirror or input:

- use `--lock-format=json|toon|both` with `eco verify`;
- use `--format=json|toon|both` with `eco seed export`, `eco needmap`,
  `eco trust snapshot`, and `eco tetrahub mirror|fetch`;
- use `--seed-format=auto|json|toon` with `eco seed import`;
- use `--metadata-format json|toon|both` with `eco materialize`.

Todex archive metadata, TetraHub `metadata.json`, `tetra.package.json`, and
vault `records.json` remain canonical JSON compatibility files.

## Packing Modes

`tetra eco pack Capsule.t4 -o manifest-only.tdx` writes a manifest-only Todex
archive. Use it when you only need to move or inspect the package manifest. It
does not include source roots such as `src/`, so unpacking it is not expected to
pass `go run ./tools/cmd/validate-eco-unpack --dir out`.

`tetra eco pack --project Capsule.t4 -o app.tdx` writes a project bundle. It
includes the manifest, project sources, and package metadata. Use this mode for
the pack/unpack workflow that is validated by:

```sh
./tetra eco pack --project Capsule.t4 -o app.tdx
./tetra eco unpack app.tdx -C out
go run ./tools/cmd/validate-eco-unpack --dir out
```

`tetra eco publish --target <triple>` is optional. When omitted, publish uses
the first `target` declared by the packaged `Capsule.t4`; packages with no
declared targets publish under the target bucket `any`. When provided, the
target must be one declared by the package manifest.

`tetra eco materialize --target <triple>` is optional. When omitted,
materialization is unscoped and `tetra.materialization.json` records an empty
`target`. When provided, and the package declares targets, the target must be
one declared by the package manifest.

`tetra project deps` manages local path dependencies in `Capsule.t4`.
`deps add --path ../Math` reads the dependency capsule ID/version by default,
stores a project-relative path in `deps:`, and prints the follow-up
`tetra project sync` command. `deps check` validates dependency paths,
ID/version matches, and dependency cycles before artifacts are refreshed.

`tetra workspace` manages a local `Tetra.workspace` member list for
multi-capsule repositories. `workspace add/remove` edits membership without
touching member capsules. `workspace check` validates members, duplicate capsule
IDs, dependency mismatches, and cycles. `workspace graph --format=json` emits a
machine-readable member/dependency graph, `workspace sync` refreshes member
locks/artifacts in dependency order, `workspace build` builds members in that
order, `workspace test` runs member tests in that order, and the
`workspace run <member>` command runs one selected app. With `-o <out-dir>`,
`workspace build` writes outputs below per-member subdirectories, and
`workspace build/test --format=json` emits a machine-readable workspace summary
with per-member status/details.

`tetra project sync` is the project-first maintenance command. It discovers the
nearest `Capsule.t4`, writes or refreshes `Tetra.lock`, and generates `.t4i`,
`.tobj`, and `.t4s` artifacts for local path dependencies when native object
targets are available. `tetra project sync --check` is the dry-run form and
reports pending lock/artifact writes without changing `interfaces/`,
`artifacts/`, `seeds/`, `Capsule.t4`, or `Tetra.lock`. For runner-gated WASM targets, `project sync` still refreshes the lock and skips native `.tobj`
generation.

Project locks are active when present. If `Tetra.lock` exists next to
`Capsule.t4`, `tetra check`, `tetra build`, and `tetra run` validate the current
capsule graph and tracked artifact hashes before compiling. `artifacts:` entries
can track `interface <path.t4i>`, `object <path.tobj>`, and `seed <path.t4s>`.
For native object artifacts, prefer `object <target> <path.tobj>` so one
capsule can hold artifacts for multiple targets without cross-linking them.

Use `tetra eco artifacts check` before a strict build to verify the complete
generated artifact set. It reports missing interface/object/seed entries, stale
`.t4i` public API hashes, wrong-target or stale `.tobj` files, stale seeds, and
stale locks with a concrete `tetra eco artifacts build ...` repair command.
Use `tetra eco artifacts build --check` for a dry-run that reports what would be
generated without writing `interfaces/`, `artifacts/`, `seeds/`, `Capsule.t4`,
or `Tetra.lock`. Use `tetra eco artifacts build --all-targets` to produce native
object artifacts for every native target declared by `Capsule.t4`; build-only
WASM targets are skipped for `.tobj` output. `tetra build --artifacts=auto`
runs the same repair step before compiling, while the default strict build mode
only validates and reports stale declared artifacts.

## Gate Evidence

`bash scripts/ci/test-all.sh --full --keep-going` includes an `eco graph bundle
vault` step that exercises:

- verify + lock
- seed export/import
- needmap
- pack/unpack
- vault add/list/verify
- trust snapshot
- materialize
- publish/download
- TetraHub beta publish/download

The current release gate runs that coverage through its stabilization wrapper:

```sh
env TETRA_TEST_ALL_RELEASE_VERSION="$release_version" bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir "$artifacts_dir/test-all"
```

That command is invoked by the active release gate named in
`docs/spec/current_supported_surface.md`.
