# Packet MPC12-S2: Report And Runtime Claims Audit

## Objective

Read-only audit of MPC-12 report/runtime/docs claims around actor/task zero-copy, owned-region move, and evidence-only conservatism.

## Context

MPC-12 does not require a full actor runtime rewrite. Reports must not claim production zero-copy or validated boundary transfer unless runtime/lowering evidence exists. Unsupported or unknown boundary behavior must be conservative or evidence-only.

## Files / Sources

- `compiler/internal/memoryfacts/*`
- `tools/cmd/validate-memory-report/*`
- `compiler/reports.go`
- `compiler/internal/actorsafety/*`
- `compiler/internal/actorsrt/production_boundary.go`
- `compiler/internal/parallelrt/*`
- `docs/design/actor_region_transfer.md`
- `docs/audits/typed-actor-ownership-transfer-v1.md`
- `docs/audits/actor-runtime-production-boundary-v1.md`
- `docs/audits/memory-production-core-v1-supported-surface.md`
- `docs/audits/memory-production-core-v1-baseline.md`
- `docs/spec/memory_report_schema_v1.md`

## Do

- Identify any report rows or docs that claim actor/task zero-copy as production-validated.
- Identify validator policy for evidence-only versus validated actor/task boundary rows.
- Identify whether owned region move has a narrow validated runtime/lowering contract, and where tests should assert conservatism.
- Recommend minimal docs/schema/report tests for MPC-12.

## Do Not

- Do not edit files.
- Do not run broad test suites.
- Do not weaken existing validated facts unless evidence says they are inflated.

## Expected Output

Summarize findings with file:line anchors, suggested RED tests, commands run, and remaining unknowns.

## Verification

Read-only commands such as `rg`, `sed`, and focused existing `go test -run` probes are acceptable if useful.
