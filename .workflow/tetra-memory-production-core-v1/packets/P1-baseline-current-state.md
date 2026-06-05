# Packet P1: Baseline Current State

## Objective

Read-only audit of the current repo against the MPC-0 baseline rows.

## Context

The orchestrator is implementing the immediate Memory Production Core v1 slice from `/home/tetra/Downloads/tetra_memory_production_core_v1_agent_plan_20260603.md`.

Focus only on MPC-0 classifications:

- safe representation metadata
- safe slice/String views
- borrow/copy/copy_into
- borrowed return syntax
- hidden borrowed aggregate escape diagnostics
- allocation length contract
- raw pointer bounds metadata
- raw slice gateway policy
- explicit island safety
- implicit region lowering
- allocation planner lowering
- inout/mutable aliasing
- cross-module resource summaries
- task/actor/request boundaries
- memory reports
- target support
- fuzz/stress coverage

## Files / Sources

Start with `compiler/internal/semantics`, `compiler/internal/lower`, `compiler/internal/plir`, `compiler/internal/allocplan`, `compiler/internal/runtimeabi`, `compiler/tests`, `docs/audits`, `docs/spec`, and `tools/cmd/memory-production-smoke`.

## Ownership

Read-only. Do not edit files.

## Do

- Classify each MPC-0 row as complete, complete_narrow_slice, partial, evidence_only, future, blocked, or explicit_non_goal.
- Cite concrete files and line references where possible.
- Note existing tests or commands that prove a row.
- Identify obvious gaps the orchestrator should document.

## Do Not

- Do not modify code/docs.
- Do not make broad implementation suggestions outside the immediate slice.
- Do not assume dirty files are correct; inspect them.

## Expected Output

Markdown report with: files inspected, commands run, classification table, evidence, uncertainty.

## Verification

At minimum run read-only `rg`/`go test -list` style probes if useful. Do not run broad tests.
