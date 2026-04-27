# Eco Publishing Model v1

Status: Accepted for Wave 10 completion on 2026-04-26.

This document stabilizes local Eco/Todex v1 contracts and defines which network/distributed capabilities are beta vs post-v1.

## 1) Capsule Manifest v1

Canonical schema identifier: `tetra.capsule.v1`.

Supported fields:
- `manifest "tetra.capsule.v1"` (optional for backward compatibility; defaults to v1)
- `capsule <Name>:`
- `id "tetra://..."`
- `version "x.y.z"`
- `target "<triple>"` (repeatable)
- `effect "<effect>"` (legacy-compatible, normalized)
- `permission "<permission>"` (repeatable)
- `dependency "<id>" "<version>"`

Backward compatibility rule: manifests without explicit `manifest` are interpreted as `tetra.capsule.v1`.

Example:

```tetra
manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
    target "linux-x64"
    permission "io"
    dependency "tetra://core" "0.1.0"
```

## 2) Permission Model v1

Canonical permission model identifier: `tetra.eco.permissions.v1`.

Rules:
- `permission` is authoritative.
- `effect` entries are normalized and added to permissions for compatibility.
- Dependency checks enforce no permission escalation: a depender must include every permission required by each dependency.
- Existing effect-based checks remain in place.

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

Materializer unpacks and writes deterministic `tetra.materialization.json` metadata for the materialized target selection.

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

## 8) Beta Package Publishing

Publishing metadata schema: `tetra.eco.publish.v1beta`.

Command:
- `tetra eco publish --package <pkg.todex> --registry <dir> --target <triple> [--trust <snapshot.json>]`

Contract:
- channel is beta-only in v1 (`channel = "beta"`)
- published metadata records package hash/size and optional trust snapshot linkage
- metadata is target-specific and must point at the package file for that target
- validators reject unknown metadata fields, unsafe paths, size/hash mismatches, and mismatched download entries

Stable local metadata fields:

```json
{
  "schema": "tetra.eco.publish.v1beta",
  "channel": "beta",
  "hub": "local-beta",
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
- `tetra eco tetrahub publish ...`
- `tetra eco tetrahub download ...`

`tetra eco tetrahub` is an explicit beta path over local/store-backed metadata layout.

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

This is intentionally local/beta metadata and not a global trust network claim.

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
- beta publishing and target-aware downloads through local/TetraHub beta paths

Post-v1 (explicitly out of v1 contract):
- distributed Todex mesh synchronization
- global EcoTrust network scoring
- EcoOracle federation/consensus
- proof-carrying capsule mesh enforcement and live-evolution protocol

Promotion rule: any post-v1 capability needs an explicit schema/version and release-gate command before being treated as v1-stable.
