# Packet MPC12-S1: Semantic Boundary Audit

## Objective

Read-only audit of compiler-owned actor/task/request boundary checks for MPC-12.

## Context

MPC-12 requires borrowed slice/String values to be rejected across actor send and task spawn boundaries unless explicitly copied. Owned copies may cross. Unknown boundary behavior must be conservative.

## Files / Sources

- `compiler/internal/semantics/exprs.go`
- `compiler/internal/semantics/types.go`
- `compiler/internal/semantics/surface_lifetime.go`
- `compiler/tests/semantics/*`
- `compiler/tests/safety/*`
- `compiler/actors_test.go`
- `compiler/task_runtime_test.go`
- `docs/design/actor_region_transfer.md`
- `examples/safe_view_actor_copy_boundary.tetra`
- `examples/safe_view_task_copy_boundary.tetra`

## Do

- Identify current actor-send gates, task-spawn gates, and any async/request boundary gates.
- Identify exact RED-test targets for borrowed slice actor send, borrowed String actor send, borrowed slice task spawn, borrowed String task spawn, copied actor send success, and copied task spawn success.
- Report whether task spawn currently validates payload expressions or only function signatures.
- Report any existing owned-region move contract that must remain accepted.

## Do Not

- Do not edit files.
- Do not run broad test suites.
- Do not claim runtime production behavior from docs alone.

## Expected Output

Summarize findings with file:line anchors, suggested RED tests, commands run, and remaining unknowns.

## Verification

Read-only commands such as `rg`, `sed`, and focused existing `go test -run` probes are acceptable if useful.
