# Capsule Artifact Build v1 Implementation Plan

**Goal:** make local capsule dependencies produce the `.t4i`, `.tobj`, `.t4s`,
and `Tetra.lock` artifacts consumed by project builds.

**Context:** `Capsule.t4` already supports `deps:` and `artifacts:`. Project
`check`/`build`/`run` already validate present locks and consume interface/object
artifacts. The missing foundation is a repeatable command that creates those
artifacts from local path dependencies.

## Task 1: Artifact Build CLI

**Goal:** add `tetra eco artifacts build`.

**Files:** `cli/cmd/tetra/eco.go`, `cli/cmd/tetra/main_test.go`.

**Approach:** add an `eco artifacts` dispatcher with a `build` subcommand.
Inputs are a project capsule path, `--target`, and optional `--lock`. The command
expands local path dependencies from the project capsule and rejects non-native
object targets.

**Verification:** focused CLI tests for command dispatch, generated files, and
error diagnostics.

**Done when:** a project with a local dependency can run the command and receive
interface/object/seed artifacts plus an updated lock.

## Task 2: Target-Aware Artifact Manifest Entries

**Goal:** allow object artifacts to declare a target.

**Files:** `cli/cmd/tetra/eco.go`, `cli/cmd/tetra/project.go`,
`tools/cmd/validate-eco-lock/main.go`.

**Approach:** parse `object <target> <path.tobj>` while retaining
`object <path.tobj>` compatibility. Build/run should link only object artifacts
whose target is empty or matches the active target.

**Verification:** focused build test that generated target-aware object artifacts
are consumed without manual `--link-object`.

**Done when:** cross-target object artifacts are not accidentally linked into the
wrong target build.

## Task 3: Lock Metadata

**Goal:** make locks useful for stale artifact diagnostics.

**Files:** `cli/cmd/tetra/eco.go`, `tools/cmd/validate-eco-lock/main.go`,
nearby tests.

**Approach:** enrich lock artifact entries with `target`, `module`, and
`public_api_hash` when available. Interface metadata comes from `.t4i`; object
metadata comes from `.tobj`.

**Verification:** validator accepts enriched artifacts and rejects malformed
paths/targets.

**Done when:** changing generated artifacts changes `graph_sha256` and the lock
names the module/API surface.

## Task 4: Docs

**Goal:** document only implemented behavior.

**Files:** `README.md`, `docs/spec/cli_contracts.md`,
`docs/spec/eco_publishing_v1.md`, `docs/user/eco_package_guide.md`.

**Verification:** docs validators plus broad Go tests.

**Done when:** users can discover the artifact build command and understand the
lock/update flow.
