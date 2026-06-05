Packet ID: MPC14-S1
Objective: Audit memory report cost_class insertion points and validator gaps for MPC-14.
Context: MPC-14 requires memory report rows to include `cost_class`, cost classes to be compiler-owned or conservative, validators to reject unknown classes, `dynamic_check_required` optimization claims without a remaining check, and `unsafe_unknown` optimized-as-trusted claims.
Files / sources:
- `compiler/internal/memoryfacts/facts.go`
- `compiler/internal/memoryfacts/report.go`
- `compiler/internal/memoryfacts/from_plir.go`
- `compiler/internal/memoryfacts/report_test.go`
- `compiler/internal/memoryfacts/from_plir_test.go`
- `tools/cmd/validate-memory-report/main.go`
- `tools/cmd/validate-memory-report/main_test.go`
- `compiler/reports.go`
- `/home/tetra/Downloads/tetra_memory_production_core_v1_agent_plan_20260603.md` MPC-14 section
Ownership: read-only. Do not edit files.
Do:
- Identify the minimal fields/enums/helpers needed for `cost_class`.
- Identify which existing claims should map to `zero_cost_proven`, `dynamic_check_required`, `instrumentation_only`, `unsupported_rejected`, or `conservative_fallback`.
- Find validator gaps and precise RED test locations.
- Check whether report generation is already optional/artifact-only.
Do not:
- Modify files.
- Propose runtime semantic changes.
- Claim cost facts from report-only reconstruction.
Expected output:
- Accepted findings with file:line evidence.
- Rejected/non-issues.
- Recommended RED tests and likely implementation points.
- Uncertainties.
Verification:
- Evidence must name inspected files and relevant symbols.
