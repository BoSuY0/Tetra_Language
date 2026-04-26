# Roadmap v0.13 → v0.14 (Backlog)

This roadmap originally focused on soundness, runtime growth, and reducing remaining OS-specific duplication. v0.14 now
also includes Flow Syntax stabilization as the bridge toward the final Tetra language surface.

Status (as of 2025-12-30):
- P0.A (CI matrix), P0.B (smoke-report-driven checklists), and P1.C (shared link core) are implemented in the repo.

## P0 — Build confidence (process)

### A) Keep CI matrix green (Linux/Windows/macOS)
- Files: `.github/workflows/ci.yml`, `scripts/bootstrap.sh`, `scripts/test.sh`
- Acceptance:
  - CI runs `bootstrap` + `test` + `tetra smoke` on all three OS runners.
  - Status: DONE

### B) Automate checklist updates from smoke reports
- Files: `tools/cmd/smoke-report-to-checklist/main.go`, `docs/checklists/*`
- Acceptance:
  - A JSON report produced by `tetra smoke --report ...` updates the platform checklists deterministically.
  - Status: DONE

## P1 — Linker unification (hourglass completion)

### C) Extract shared x64 link core
- Files: `compiler/internal/linker/*`
- Deliverable:
  - A shared linker “plan/core” that:
    - loads TOBJ objects
    - resolves symbols
    - lays out `.text`/`.data`
    - applies relocations (`RelocCallRel32`, `RelocDataDisp32`, `RelocIATDisp32` where applicable)
  - Format-specific writers stay in `elf/macho/pe`.
- Acceptance:
  - `go test ./...` passes
  - `tetra smoke` passes for native targets in CI
  - Linker duplication between `linux.go` / `macos.go` / `windows.go` is significantly reduced.
  - Status: DONE

## P2 — Region typing (soundness & ergonomics)

### D) Expand interprocedural region inference (conservative)
- Files: `compiler/internal/semantics/checker.go`, `compiler/*_test.go`
- Acceptance:
  - New tests cover:
    - slice returned through helper chains
    - structs containing slices returned from helpers
    - clearer errors when inference fails (no unsound fallbacks)

## P3 — cap.mem (low-level stack)

### E) Extend cap.mem primitives beyond i32
- Files: `compiler/internal/semantics/checker.go`, `compiler/internal/lower/lower.go`, `compiler/internal/ir/ir.go`, `compiler/internal/backend/x64core/emit.go`, `examples/*`
- Acceptance:
  - Minimal set of additional primitives (e.g. `load_u8` / `store_u8`) with `unsafe` + `cap.mem` gating.
  - At least one small example (e.g. `memcpy`/`memset`) and smoke coverage.

## P4 — Actors runtime evolution (post-MVP)

### F) Prepare self-hosted runtime experiment (non-breaking)
- Deliverable:
  - A separate track for a Tetra-authored runtime module without regressing the current actors MVP.
  - A stable symbol surface (`__tetra_entry`, `__tetra_actor_*`, etc.) or an attribute mechanism for link names.

Next focus (v0.14+):
- Document the runtime ABI surface (`__tetra_*` symbols) as an explicit contract.
- Use the existing `@export`, `--emit=library`, and `--runtime-object` mechanisms to prototype a self-hosted runtime
  object, starting with actors.
  - Status: IN PROGRESS (PoC exists under `__rt/`, plus e2e tests)

## P5 — Flow Syntax Stabilization

### G) Stabilize Flow indentation syntax
- Files: `compiler/internal/frontend/flow.go`, `compiler/internal/frontend/parser_test.go`, `compiler/compiler_test.go`, `examples/flow_*.tetra`
- Acceptance:
  - Flow `func`, `struct`, `if`/`else`, `while`, `unsafe`, and scoped `island` blocks parse through the existing AST/IR path.
  - Comments and blank lines do not accidentally close Flow blocks.
  - Tabs and missing indentation produce clear diagnostics.
  - Flow examples are included in `tetra smoke`.
  - Status: IN PROGRESS
