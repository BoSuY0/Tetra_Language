# Eco Publishing Model v1

Status: Originally accepted for Wave 10 completion on 2026-04-26; synchronized
with the current `v0.4.0` release truth on 2026-05-20. This spec applies to
the local Eco/Todex contract baseline for the current `v0.4.0` release line.

This document stabilizes local Eco/Todex v1 contracts and defines which network/distributed
capabilities are beta vs post-v1.

## 1) Capsule Manifest v1

Canonical schema identifier: `tetra.capsule.v1`.

Supported fields:
- `manifest "tetra.capsule.v1"` (optional for backward compatibility; defaults to v1)
- `capsule <Name>:`
- `id "tetra://..."`
- `version "x.y.z"`
- `entry "<relative .t4/.tetra path>"`
- `source "<relative source root>"` or a `sources:` block
- `target "<triple>"` (repeatable)
- `targets:` block with bare aliases or triples (`linux`, `windows`, `macOS`,
  `web`, `wasi`, `linux-x64`, `wasm32-web`, ...)
- `effect "<effect>"` (legacy-compatible, normalized)
- `permission "<permission>"` (repeatable)
- `allow:` block for permissions such as `ui` or `fs.readWrite.userData`
- `dependency "<id>" "<version>" ["<local capsule path>"]`
- `deps:` block with `<id> <version> [local capsule path]` entries
- `artifact "<kind>" "<relative path>"` or an `artifacts:` block with
  `<kind> <relative path>` entries. Supported kinds are `interface` (`.t4i`),
  `object` (`.tobj`), and `seed` (`.t4s`). Object artifacts may be target-aware:
  `object linux-x64 artifacts/math/core.linux-x64.tobj`.
- `policy:` block with semantic project policy keys:
  `unsafe deny|allow` and `reproducible required|preferred|off`

Backward compatibility rule: manifests without explicit `manifest` are interpreted as
`tetra.capsule.v1`.

CLI project discovery rule: `tetra check`, `tetra build`, `tetra run`,
`tetra test`, and `tetra doc` walk upward from the current/input path, prefer
`Capsule.t4`, and fall back to legacy `Tetra.capsule`. When a project is found,
`entry` selects the default program and `sources` become module lookup roots. If
`sources` is omitted, the CLI uses the conventional roots `src`, `ui`, `tests`,
`drivers`, `kernel`, `game`, and project root. Dependencies with local paths add
that capsule's own source roots to module lookup, so `import other.module` can
resolve across local workspace capsules without copying source files. Interface
artifacts add `.t4i` module roots, object artifacts matching the active target
are passed to native linking, and seed artifacts are lock-tracked offline inputs.

If a discovered project contains `Tetra.lock`, `tetra check`, `tetra build`, and
`tetra run` validate the current capsule graph and artifact hashes against the
lock before compiler work begins. `tetra project sync` is the project-first
command that expands local path dependencies, refreshes `Tetra.lock`, and
generates local dependency artifacts when native object targets are available.
`tetra project sync --check` reports pending lock/artifact writes without
modifying project files. Build-only targets such as WASM use lock-only sync and
skip native `.tobj` generation. The lower-level `tetra eco verify --lock
Tetra.lock Capsule.t4` remains available when only the lock graph should be
refreshed.

`tetra eco artifacts build --target <native-triple> --lock Tetra.lock Capsule.t4`
builds local path dependencies into project artifacts, appends missing
`artifacts:` entries to `Capsule.t4`, writes `.t4i` interfaces under
`interfaces/`, `.tobj` implementation objects under `artifacts/`, a dependency
seed under `seeds/`, and then refreshes `Tetra.lock`.

`tetra eco artifacts check` validates the same expected artifact set without
writing files. It reports missing/stale generated interfaces, wrong-target or
stale object files, stale dependency seeds, and stale semantic locks with a
repair command. `tetra eco artifacts build --check` is the dry-run form of the
builder and reports pending writes as `would generate ...`. `tetra eco artifacts
build --all-targets` emits native object artifacts for every native target in
`Capsule.t4` and skips runner-gated WASM object outputs.
`tetra build --artifacts=auto` runs the repair step before compiling; the
default strict build mode only validates declared artifacts and reports
diagnostics.

Example:

```t4 unsupported
manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    entry "src/main.t4"

    sources:
        src
        ui

    targets:
        linux
        web

    allow:
        io
        ui

    policy:
        unsafe deny
        reproducible required

    deps:
        tetra://core 0.1.0 ../tetra-core

    artifacts:
        interface interfaces/core/api.t4i
        object linux-x64 artifacts/core-linux-x64.tobj
        seed seeds/tetra-core.t4s
```

## 2) Permission Model v1

Canonical permission model identifier: `tetra.eco.permissions.v1`.

Rules:
- `permission` is authoritative.
- `effect` entries are normalized and added to permissions for compatibility.
- Dependency checks enforce no permission escalation: a depender must include every permission
  required by each dependency.
- Existing effect-based checks remain in place.

`Tetra.lock` also records each capsule `policy` map and artifact list when
present. Artifact lock entries include SHA-256 and, when available, target,
module, and public API hash metadata. The lock graph hash includes policy keys,
dependency edges, artifact paths, targets, modules, public API hashes, and
artifact SHA-256 values, so changing `unsafe`, `reproducible`, a dependency, or
a tracked `.t4i`/`.tobj`/`.t4s` artifact changes the semantic lock.

## 3) Seed Import/Export v1

Seed schema identifier: `tetra.eco.seed.v1`.

Commands:
- `tetra eco seed export --out <seed.json> [capsules...]`
- `tetra eco seed import --seed <seed.json> --lock <lock.json> [--capsules-dir <dir>]`

Seed carries a lock snapshot plus capsule entries for reproducible graph exchange.

## 4) NeedMap v1

NeedMap schema identifier: `tetra.eco.needmap.v1`.

Command:
- `tetra eco needmap --lock <lock.json> -o <needmap.json>`

NeedMap captures:
- capsule nodes
- dependency edges
- transitive need sets
- target set and lock hash reference

## 5) TrustSnapshot v1

Trust snapshot schema identifier: `tetra.eco.trust-snapshot.v1`.

Command:
- `tetra eco trust snapshot --lock <lock.json> --store <vault> -o <snapshot.json>`

Snapshot captures:
- lock hash
- vault hash
- capsule-level trust tier/score/reasons

## 6) Materializer v1

Materialization schema identifier: `tetra.eco.materialization.v1`.

Command:
- `tetra eco materialize <package.todex> --target <triple> --trust <snapshot.json> -C <out>`

Materializer unpacks and writes deterministic `tetra.materialization.json` metadata for the
materialized target selection.

## 7) Reproducible Build Basics v1

`eco pack` v1 reproducibility rules:
- stable sorted archive path order
- normalized gzip/tar timestamps (`mtime = 0`)
- normalized uid/gid/uname/gname fields
- deterministic package metadata digest (`build_inputs_sha256`)

Package metadata remains `tetra.eco.package.v1`.

Minimal unpacked project bundle metadata example:

```json
{
  "schema": "tetra.eco.package.v1",
  "compression": "gzip",
  "mtime_unix": 0,
  "reproducible": true,
  "build_inputs_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "manifest_schema": "tetra.capsule.v1",
  "permissions_model": "tetra.eco.permissions.v1",
  "file_count": 2,
  "files": [
    {
      "path": "Tetra.capsule",
      "sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
      "size": 128
    },
    {
      "path": "src/main.tetra",
      "sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
      "size": 32
    }
  ]
}
```

## 8) Package Publishing

Publishing metadata schemas:

- `tetra.eco.publish.v1` for stable local production metadata
- `tetra.eco.publish.v1beta` for beta metadata

Command:
- `tetra eco publish --package <pkg.todex> --registry <dir> --target <triple> [--trust <snapshot.json>] [--channel beta|stable]`

Contract:
- `channel = "stable"` writes `tetra.eco.publish.v1`
- `channel = "beta"` writes `tetra.eco.publish.v1beta`
- published metadata records package hash/size and optional trust snapshot linkage
- metadata is target-specific and must point at the package file for that target
- validators reject unknown metadata fields, unsafe paths, size/hash mismatches,
  schema/channel mismatches, and mismatched download entries

Stable local metadata fields:

```json
{
  "schema": "tetra.eco.publish.v1",
  "channel": "stable",
  "hub": "production",
  "published_at_unix": 0,
  "capsule": {
    "id": "tetra://demo",
    "name": "Demo",
    "version": "0.1.0",
    "target": "linux-x64",
    "targets": ["linux-x64"],
    "permissions": ["io"]
  },
  "package": {
    "file": "package.todex",
    "size": 4096,
    "sha256": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
  },
  "trust": {
    "snapshot_file": "trust.snapshot.json",
    "snapshot_sha256": "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
    "trust_tier": "high"
  },
  "downloads": [
    {
      "target": "linux-x64",
      "path": "packages/tetra_demo/0.1.0/linux-x64/package.todex"
    }
  ]
}
```

## 9) TetraHub Beta Path

Commands:
- `tetra eco tetrahub publish ... [--channel beta|stable]`
- `tetra eco tetrahub download ...`
- `tetra eco tetrahub mirror --from <store> --to <store> --id <id> --version <x.y.z> --target <triple> -o <report.json>`
- `tetra eco tetrahub fetch --url <http-url> --to <store> --id <id> --version <x.y.z> --target <triple> -o <report.json>`

`tetra eco tetrahub publish --channel stable` writes local production metadata
with schema `tetra.eco.publish.v1` and `hub = "tetrahub-stable"`. The default
channel remains beta over the local/store-backed metadata layout.

`tetra eco tetrahub mirror` is a local store-to-store synchronization primitive,
not a network mesh. It validates the source package metadata, package bytes, and
optional trust snapshot before copying them byte-for-byte into the destination
store. The command emits `tetra.eco.mirror.v1`, validated by
`tools/cmd/validate-eco-mirror`.

`tetra eco tetrahub fetch` is the matching single-origin HTTP(S) fetch primitive.
It downloads `metadata.json`, `package.todex`, and the optional trust snapshot
from one TetraHub store URL, validates size/hash/schema fields before accepting
the package, writes the verified bytes into a local destination store, and emits
the same `tetra.eco.mirror.v1` report. This proves network transport integrity
for a concrete store entry; it is not hub discovery, federation, consensus, or a
distributed mesh.

Mirror report fields:

```json
{
  "schema": "tetra.eco.mirror.v1",
  "mirrored_at_unix": 0,
  "source_store": ".tetra/tetrahub-stable",
  "destination_store": ".tetra/tetrahub-mirror",
  "id": "tetra://demo",
  "version": "0.1.0",
  "target": "linux-x64",
  "channel": "stable",
  "hub": "tetrahub-stable",
  "package_path": "packages/tetra_demo/0.1.0/linux-x64/package.todex",
  "package_sha256": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
  "metadata_path": "packages/tetra_demo/0.1.0/linux-x64/metadata.json",
  "metadata_sha256": "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
  "trust_snapshot_path": "packages/tetra_demo/0.1.0/linux-x64/trust.snapshot.json",
  "trust_snapshot_sha256": "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
}
```

## 10) Target-Aware Downloads

Command:
- `tetra eco download --id <id> --version <x.y.z> --target <triple> --registry <dir> -o <pkg.todex>`

Behavior:
- resolves by capsule id/version/target
- returns explicit available-target diagnostics on mismatch

## 11) Trust Metadata

Trust metadata is published and consumed via:
- `trust.snapshot_sha256` in publish metadata
- capsule trust tier and score from TrustSnapshot

This is intentionally local metadata and not a global trust network claim.

Trust snapshot example:

```json
{
  "schema": "tetra.eco.trust-snapshot.v1",
  "generated_at_unix": 0,
  "lock_sha256": "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
  "vault_sha256": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
  "record_count": 1,
  "capsules": [
    {
      "id": "tetra://demo",
      "version": "0.1.0",
      "permissions": ["io"],
      "trust_tier": "high",
      "trust_score": 95,
      "trust_reasons": ["permissions=io"]
    }
  ]
}
```

## 12) v1 vs post-v1 boundaries

In v1:
- local manifest/lock/seed/needmap/trust snapshot/materialization flows
- local vault verification
- stable local publish metadata and target-aware downloads
- stable local TetraHub store metadata and downloads
- local TetraHub mirror reports and byte-preserving store-to-store copies
- single-origin HTTP(S) TetraHub fetch into a local verified store
- beta publishing and target-aware downloads through local/TetraHub beta paths

Post-v1 (explicitly out of v1 contract):
- distributed Todex mesh synchronization beyond local store-to-store mirroring
  and single-origin HTTP(S) fetch
- global EcoTrust network scoring
- EcoOracle federation/consensus
- proof-carrying capsule mesh enforcement and live-evolution protocol

Promotion rule: any post-v1 capability needs an explicit schema/version and release-gate command
before being treated as v1-stable.

## 13) v0.3.0 release evidence

`v0.3.0` treats Eco/Todex as local-only workflows with validator-backed
metadata. It covers local package lifecycle validation for verify,
lock-generation and lock-validation through `--lock` workflows, pack/unpack,
vault, and publish metadata fixtures. `v0.4.0` adds stable local publish
metadata through `tetra.eco.publish.v1`, local TetraHub mirror reports through
`tetra.eco.mirror.v1`, and single-origin HTTP(S) TetraHub fetch with the same
integrity checks. Hosted production TetraHub, distributed Todex mesh
synchronization beyond local mirroring/fetch, and global trust federation remain
outside the current support claim.

Focused verification command:

`go test ./cli/... ./tools/... -run 'Eco|Project|Workspace|Artifact|Capsule|Lock' -count=1`

Gate workflow command:

`bash scripts/release/v0_3_0/gate.sh`

That gate invokes the stabilization wrapper around `scripts/ci/test-all.sh` and
must pass before claiming current Eco/Todex release health. For user-facing
command coverage, keep this spec aligned with `docs/user/eco_package_guide.md`
and the current release-truth layer in `docs/spec/current_supported_surface.md`.
