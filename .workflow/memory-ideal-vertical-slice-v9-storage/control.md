# Control: Memory Ideal Vertical Slice v9 Storage

primary_priority: evidence_quality
secondary_priority: behavior_preservation

## Allowed Files

- `compiler/internal/allocplan/**`
- `compiler/internal/validation/**`
- `compiler/internal/lower/**`
- `compiler/internal/memoryfacts/**`
- `compiler/internal/memorymodel/**`
- `compiler/tests/semantics/**`
- `compiler/tests/ownership/**`
- `tools/cmd/validate-memory-report/**`
- `tools/cmd/validate-memory-correlation/**`
- `docs/audits/memory-ideal-vslice-v9-storage-*.md`
- `docs/spec/memory_report_schema_v1.md`
- `docs/design/memory_production_core_v1.md`
- `docs/generated/manifest.json`
- `.workflow/memory-ideal-vertical-slice-v9-storage/**`
- `GOAL.md`

## Protected Files

Unrelated dirty files outside v9 scope. Do not revert, clean, or normalize them.

## Require Approval For

- scope expansion beyond `MEM-STORAGE-001` through `MEM-STORAGE-004`
- dependency changes
- broad schema redesign
- performance, target parity, production actor runtime, full async lifetime, or
  arbitrary FFI/runtime safety claims
- destructive cleanup of dirty worktree

## Execution Knobs

- max_parallel_jobs: use parallel reads/tests when independent
- avoid running broad CI concurrently with other heavy gates
- preserve persistent Go cache discipline under `.cache/`

This file may narrow priorities or require approval, but it cannot silently
weaken `GOAL.md`, `done_when`, or v9 nonclaims.
