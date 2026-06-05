Packet ID: MPC14-S2
Objective: Audit performance blocker, docs, manifest, and claim wording gaps for MPC-14.
Context: MPC-14 requires `docs/design/memory_cost_model.md`, docs/manifest gates, performance blocker rows mapped to the same cost classes, and validator rejection of fake zero-cost or trusted unsafe optimization claims.
Files / sources:
- `compiler/reports.go`
- `compiler/reports_internal_test.go`
- `tools/cmd/validate-performance-report/main.go`
- `tools/cmd/validate-performance-report/main_test.go`
- `tools/cmd/verify-docs/main.go`
- `tools/cmd/verify-docs/main_test.go`
- `tools/cmd/validate-manifest/main.go`
- `compiler/features.go`
- `compiler/tests/semantics/features_test.go`
- `docs/design/memory_production_core_v1.md`
- `docs/audits/memory-production-core-v1-supported-surface.md`
- `docs/audits/performance-blocker-reports-v1.md`
- `/home/tetra/Downloads/tetra_memory_production_core_v1_agent_plan_20260603.md` MPC-14 section
Ownership: read-only. Do not edit files.
Do:
- Identify how `.perf.json` blocker rows can carry `cost_class`.
- Identify which blockers are memory blockers and expected cost classes.
- Find docs/manifest gate insertion points for `docs/design/memory_cost_model.md`.
- Identify claim wording validators that should reject fake zero-cost/trusted unsafe claims.
Do not:
- Modify files.
- Expand P20 performance claim scope.
- Require benchmark measurement for MPC-14.
Expected output:
- Accepted findings with file:line evidence.
- Rejected/non-issues.
- Recommended RED tests and likely implementation points.
- Uncertainties.
Verification:
- Evidence must name inspected files and relevant symbols.
