# Eco Package Guide

Status: user guide for v0.2.0 local Eco/Todex workflows.

This guide describes what is release-covered today for local capsule/package
work. It does not claim distributed publishing, a production TetraHub network,
or global trust federation.

Terminology note:
- `Capsule.t4` / `Tetra.capsule` in this guide are Eco project/package manifest
  files.
- They are distinct from source-language top-level `capsule Name: ...`
  declarations, which are compile-time language metadata.

## Local-Only Scope (v0.2.0)

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
- `./tetra workspace list|check|graph [path]`
- `./tetra workspace sync [path]`
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
- `./tetra eco pack --project Capsule.t4 -o app.tdx`
- `./tetra eco unpack app.tdx -C out`
- `./tetra eco vault add|list|verify --store .tetra/todex-vault ...`
- `./tetra eco trust snapshot --lock Tetra.lock --store .tetra/todex-vault -o trust.snapshot.json`
- `./tetra eco materialize app.tdx --target <triple> --trust trust.snapshot.json -C out`
- `./tetra eco publish --package app.tdx --registry .tetra/registry-beta --target <triple> [--trust trust.snapshot.json]`
- `./tetra eco download --id <id> --version <x.y.z> --target <triple> --registry .tetra/registry-beta -o app.tdx`
- `./tetra eco tetrahub publish|download ... --store .tetra/tetrahub-beta`

Beta boundary:

- publish/download and `tetra eco tetrahub` are local beta metadata/store paths.
- they are not claims about a production online hub.

## Validator Contracts

Release checks use machine-validated artifacts:

- `go run ./tools/cmd/validate-eco-lock --lock Tetra.lock`
- `go run ./tools/cmd/validate-eco-unpack --dir out`
- `go run ./tools/cmd/validate-eco-vault --store .tetra/todex-vault`
- `go run ./tools/cmd/validate-eco-publish --registry .tetra/registry-beta --id <id> --version <x.y.z> --target <triple>`

The validators enforce schema/path/hash constraints, reject malformed metadata,
and keep local workflows deterministic.

`tetra project deps` manages local path dependencies in `Capsule.t4`.
`deps add --path ../Math` reads the dependency capsule ID/version by default,
stores a project-relative path in `deps:`, and prints the follow-up
`tetra project sync` command. `deps check` validates dependency paths,
ID/version matches, and dependency cycles before artifacts are refreshed.

`tetra workspace` manages a local `Tetra.workspace` member list for
multi-capsule repositories. `workspace check` validates members, duplicate
capsule IDs, dependency mismatches, and cycles. `workspace graph --format=json`
emits a machine-readable member/dependency graph, `workspace sync` refreshes
member locks/artifacts in dependency order, `workspace build/test` executes all
members in that order, and `workspace run <member>` runs one selected app.

`tetra project sync` is the project-first maintenance command. It discovers the
nearest `Capsule.t4`, writes or refreshes `Tetra.lock`, and generates `.t4i`,
`.tobj`, and `.t4s` artifacts for local path dependencies when native object
targets are available. `tetra project sync --check` is the dry-run form and
reports pending lock/artifact writes without changing `interfaces/`,
`artifacts/`, `seeds/`, `Capsule.t4`, or `Tetra.lock`. For build-only targets
such as WASM, `project sync` still refreshes the lock and skips native `.tobj`
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

`bash scripts/test_all.sh --full --keep-going` includes an `eco graph bundle
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

The same full script is executed inside the release gate
`TETRA_SECURITY_REVIEW_SIGNOFF=<path> bash scripts/release_v0_2_0_gate.sh`.
