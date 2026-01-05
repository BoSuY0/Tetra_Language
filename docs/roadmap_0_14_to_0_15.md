# Roadmap v0.14 → v0.15 (Backlog)

Focus: productionize the self-hosted runtime path and continue reducing remaining OS-specific duplication.

## P0 — Documentation hygiene

- Update `docs/spec/capabilities.md` to match the current cap.mem primitives (`load/store_i32/u8/ptr`, `ptr_add`) and the
  MMIO volatile contract.
- Add a short spec for globals (top-level `var/val`) and their current initialization limits.

## P1 — Self-host actors runtime (3 OS)

- Make the self-hosted actors runtime runnable on `windows-x64` (in addition to SysV hosts), and run it in CI.
- Promote a canonical runtime source file under `__rt/` (replacing PoC naming) with a stable exported surface per
  `docs/spec/runtime_abi.md`.

## P2 — Self-host runtime as a first-class build path

- Add an optional build mode where the compiler auto-builds and links the canonical `__rt` runtime module when actors are
  used (keeping `--runtime-object` as an override).
- Deprecate the Go-emitted `actorsrt/*_emit.go` scheduler once the self-host runtime becomes stable.

## P3 — Library linking ergonomics

- Add a general `--link-object path.tobj` flag (in addition to `--runtime-object`) so users can link arbitrary TOBJ
  libraries without special runtime semantics.
- Add e2e tests that validate `@export`-driven symbol aliases across object boundaries.

