Packet ID: MPC15-S2
Objective: Audit memory production smoke, memory report validation, and docs/manifest gates for MPC-15 fuzz oracle integration.
Context: MPC-15 requires generated-program invariants: no safe metadata mutation, no borrowed escape, no unsafe_unknown -> safe_known, no removed bounds check without proof id, no stack/region storage if escape exists, and reports validate against MemoryFactGraph plus MPC-14 cost model.
Files / sources:
- `tools/cmd/memory-production-smoke/main.go`
- `tools/cmd/memory-production-smoke/main_test.go`
- `tools/validators/memoryprod/report.go`
- `tools/validators/memoryprod/report_test.go`
- `tools/cmd/validate-memory-production/main.go`
- `compiler/internal/memoryfacts/report.go`
- `compiler/internal/memoryfacts/graph.go`
- `compiler/reports.go`
- `tools/cmd/verify-docs/main.go`
- `tools/cmd/validate-manifest/main.go`
- `docs/design/memory_production_core_v1.md`
- `docs/audits/memory-production-core-v1-supported-surface.md`
- `/home/tetra/Downloads/tetra_memory_production_core_v1_agent_plan_20260603.md` MPC-15 section
Ownership: read-only. Do not edit files.
Do:
- Identify how a Tier 1 memory fuzz smoke should emit report artifacts without changing normal program semantics.
- Identify where memoryprod or a new validator should require oracle categories and invariant checks.
- Find RED-test locations for missing oracle category coverage, missing report validation failure case, unsafe_unknown upgraded as safe, and missing fuzz oracle docs.
- Identify docs/manifest insertion points for `docs/audits/memory-fuzz-oracle-v1.md`.
Do not:
- Modify files.
- Require linux-x64 runtime smoke for every unit test.
- Reconstruct compiler-owned truth from report text.
Expected output:
- Accepted findings with file:line evidence.
- Rejected/non-issues.
- Recommended design and RED tests.
- Uncertainties.
Verification:
- Evidence must name inspected files and relevant symbols.
