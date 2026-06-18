# Capsule Project System v1 Implementation Plan

**Goal:** implement `Capsule.t4` as the practical project root for Tetra.
**Context:** T4 file formats exist; the next foundation is making project
discovery, source roots, and Eco metadata real.
**Execution:** TDD task-by-task in the current session.

## Task 1: Project-Aware Module Loading

**Goal:** allow the compiler to load modules from a project root and multiple
source roots.

**Files:** `compiler/internal/module/loader.go`, `compiler/api.go`,
`compiler/compiler.go`, nearby module/compiler tests.

**Approach:** add load options for project root/source roots, keep existing
`LoadWorld` behavior as the compatibility default, and route build paths through
the new loader when build options include project metadata.

**Verification:** `go test ./compiler/internal/module ./compiler`

**Done when:** imports resolve from declared source roots and `.t4` remains
preferred over legacy `.tetra`.

## Task 2: Structured Capsule Parser

**Goal:** support `entry`, `sources:`, `targets:`, `deps:`, `allow:`, and
`policy:` in `Capsule.t4`.

**Files:** `cli/cmd/tetra/eco.go`, Eco tests, lock validator if lock shape
changes.

**Approach:** extend the existing line parser without breaking flat manifest
fields. Normalize target aliases, permissions, exact dependencies, and policy
keys.

**Verification:** `go test ./cli/cmd/tetra ./tools/cmd/validate-eco-lock`

**Done when:** structured `Capsule.t4` verifies, writes a lock, and legacy
capsules still pass.

## Task 3: CLI Project Discovery

**Goal:** make `tetra check/build/run/test` discover `Capsule.t4` and use its
entry/source roots when no explicit input is given.

**Files:** `cli/cmd/tetra/main.go`, optionally a new CLI helper file,
`cli/cmd/tetra/main_test.go`.

**Approach:** discover `Capsule.t4` upward from cwd or explicit input, resolve
entry defaults, pass project root/source roots into compiler load/build options,
and keep non-project single-file behavior unchanged.

**Verification:** `go test ./cli/cmd/tetra`

**Done when:** project-level check/build work without passing `src/main.t4`.

## Task 4: Docs and Release Surface

**Goal:** document the project root rules and keep generated manifests valid.

**Files:** `docs/spec/t4_formats.md`, `docs/spec/eco_publishing_v1.md`,
`docs/user/getting_started.md`, `docs/user/eco_package_guide.md`,
`docs/generated/manifest.json` if needed.

**Approach:** update docs only for implemented behavior.

**Verification:** `go test ./compiler/... ./cli/... ./tools/...` and
`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`

**Done when:** tests pass and docs describe the actual behavior.
