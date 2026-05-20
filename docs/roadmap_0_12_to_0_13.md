# Roadmap v0.12 → v0.13 (Done)

This file turned the v0.12 planning notes into a concrete backlog with acceptance criteria.

Status: completed as of `v0.13` (2025-12-30). Next: `docs/roadmap_0_13_to_0_14.md`.

## P0 — Workspace + DX

### A) Fix `tetra run -o app` on Unix
- Files: `cli/cmd/tetra/main.go`
- Acceptance:
  - `./tetra run --target linux-x64 -o app examples/hello.tetra` runs without requiring `./app`.

### B) Add `tetra version`
- Files: `cli/cmd/tetra/main.go`, `compiler/version.go`, `compiler/internal/version/version.go`
- Acceptance:
  - `./tetra version` prints `tetra_language <compilerVersion>` (and git HEAD when available).

### C) Scripts policy (bootstrap/test/dump)
- Files: `scripts/dev/bootstrap.sh`, `scripts/ci/test.sh`, `scripts/dev/dump-project.sh`
- Acceptance:
  - `bash scripts/dev/bootstrap.sh` produces `./tetra`.
  - `bash scripts/ci/test.sh` runs `go test` for `./compiler/...`, `./cli/...`, `./tools/...`.
  - `bash scripts/dev/dump-project.sh` produces a dump via `tools/cmd/dump-project`.

### D) Automated smoke runner
- Files: `cli/cmd/tetra/main.go`
- Acceptance:
  - `./tetra smoke --target linux-x64` builds and runs:
    - `examples/islands_hello.tetra` (exit 0)
    - `examples/islands_i32.tetra` (exit 55)
    - `examples/islands_overflow.tetra` (exit 1)
    - `examples/mmio_smoke.tetra` (exit 123)
  - `./tetra smoke --target windows-x64 --run=false` builds all smoke targets.

## P1 — Cross-platform confidence

### E) CI (Linux/Windows/macOS)
- Acceptance:
  - Each OS runs `go test` per module and `tetra smoke` for its native target.

## P2 — Actors MVP (design + runtime)

### F) Actors spec
- Deliverable: a spec in `docs/spec/actors.md`.

### G) Minimal runtime + builtins
- Acceptance:
  - `spawn` + `send` + `recv` implemented with a single-thread cooperative scheduler.
