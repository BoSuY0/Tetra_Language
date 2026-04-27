# Tetra v0.1.2 Stabilization Start

Status: completed patch-line planning note for the `v0.1.2` release tag.

`v0.1.2` is a patch/stabilization line under the project version policy:
cleanup, diagnostics, flaky test reduction, documentation fixes, and release
tooling hardening are allowed; large language changes or compatibility breaks
belong in a later minor or major line.

## Goals

- Keep `v0.1.1` tag immutable and reproducible.
- Reduce naming debt left by historical `v1_0` artifact paths without breaking
  existing tools.
- Tighten diagnostics and focused regression coverage around the already
  supported language/profile.
- Improve docs that still mix current release truth with future v1.0 scope.
- Keep every change backed by focused tests before broad gates.

## Initial Tasks

- [x] Audit docs for remaining current-release references that should point to
      `docs/checklists/v0_1_2_release_gate.md` or
      `scripts/release_v0_1_2_gate.sh`.
- [x] Decide whether `docs/generated/v1_0` should remain the compatibility
      archive path for all `v0.1.x` snapshots or gain a new `v0_1` directory in
      a reviewed migration.
      Decision: keep `docs/generated/v1_0` as the compatibility archive path
      during `v0.1.x`; migrate only through a reviewed post-release cleanup.
- [x] Add focused tests for the compatibility alias
      `scripts/release_v1_0_gate.sh` so future cleanup cannot silently remove
      the current gate path.
- [x] Review `docs/spec/v1_scope.md` and split future-only v1 requirements from
      current `v0.1.x` release claims.
- [x] Run `go test ./compiler/... ./cli/... ./tools/... -count=1` and
      `bash scripts/test_all.sh --full --keep-going` before any `v0.1.2`
      version bump.

## Non-goals

- Do not bump `CompilerVersion` beyond `v0.1.2` until at least one real patch
  fix exists and the release gate has been rerun.
- Do not mark Tetra as `v1.0.0`.
- Do not remove compatibility filenames until all scripts, docs, tests, and
  generated evidence paths have a reviewed migration.
