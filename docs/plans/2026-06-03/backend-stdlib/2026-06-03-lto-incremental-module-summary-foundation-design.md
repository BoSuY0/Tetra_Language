# P17.4 LTO Incremental Module Summary Foundation Design

Status: Approved by active `GOAL.md` P17.4 Bridge. This document narrows the
LTO/incremental batch to evidence and contracts only.

## Goal

Promote the P17.4 `lto_incremental_module_summary` row from `not_yet_covered`
to `implemented_narrow` by adding a deterministic internal module-summary
schema, dependency hash contract evidence, cross-module validation rows,
incremental negative tests, and an explicit non-consumer boundary.

## Observed Facts

- `compiler/internal/cache` owns cache keys, source hashes, dependency
  signature hashes, `DepSigHashFromDepsWithInterfaceHashes`, object cache
  load/store, and compiler/cache ABI discriminators.
- `compiler/compiler.go::planNativeModuleBuild` computes per-module source
  hashes, dependency hashes, public API hashes, cache hits, and compile jobs.
- `compiler/internal/format/tobj.Object` stores `SrcHash`, `WorldSigHash`, and
  `PublicAPIHash`.
- `compiler/internal/opt/pgo_lto.go` owns the P17.4 coverage matrix.

## Design

Add an internal `tetra.incremental.module_summary.v1` summary in
`compiler/internal/cache`. A summary records:

- module, target, and build tag;
- source hash as `sha256:...`;
- dependency hash as `sha256:...`;
- public API hash;
- sorted external callees and type dependencies;
- validation rows proving source hash, dependency hash, public API hash,
  cross-module signature inputs, and non-consumer boundary;
- explicit `codegen_consumer=false` and `linker_consumer=false`.

The summary is not an LTO optimizer input and is not a cache mode. It is a
machine-checkable evidence artifact that documents the contract already used by
incremental object caching.

## Implementation Plan

1. Add RED cache tests for deterministic summary construction,
   `DepSigHashFromDepsWithInterfaceHashes` sensitivity, schema validation, and
   negative rejection of missing hashes or any codegen/linker consumer.
2. Add RED P17.4 coverage tests expecting `lto_incremental_module_summary` to
   be `implemented_narrow` with no optimizer input and no safe-semantics
   change.
3. Implement the smallest `compiler/internal/cache` summary builder and
   validator.
4. Promote only the P17.4 LTO summary row in `compiler/internal/opt/pgo_lto.go`.
5. Update feature text, audit/progress docs, manifest, and goal sidecars while
   preserving all non-claims.

## Verification

Focused RED/GREEN:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/cache ./compiler/internal/opt ./compiler/tests/semantics -run 'TestIncrementalModuleSummaryV1RecordsDependencyHashContractAndRejectsConsumers|TestPGOLTOTargetCPUCoverageAuditsP17PlanList|TestFeatureRegistryCoversReleaseStatusesAndKeyBoundaries' -count=1
```

Broad relevant gate:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/cache ./compiler/internal/opt ./compiler/internal/validation ./compiler ./compiler/tests/semantics ./tools/cmd/validate-manifest ./tools/cmd/verify-docs -count=1
```

After code changes, run `graphify update .`, `git diff --check`, stale/overclaim
scans, scratch scan, and `GOCACHE=$(pwd)/.cache/go-build-ideal-plan go clean
-cache`.

## Non-Claims

- No LTO optimizer, cross-module inlining, linker consumer, codegen consumer,
  incremental speedup, cache performance claim, or C/Rust parity claim is made.
- No public PGO/profile/LTO/target-cpu flag changes safe-program semantics.
