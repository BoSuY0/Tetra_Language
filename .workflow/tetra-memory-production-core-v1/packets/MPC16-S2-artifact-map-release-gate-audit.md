Packet ID: MPC16-S2
Objective: Audit MPC-16 artifact map, nonclaims, docs/manifest gates, and required release command evidence.
Context: MPC-16 requires `docs/audits/memory-production-core-v1-artifact-map.md`, `docs/audits/memory-production-core-v1-nonclaims.md`, and final commands including `scripts/ci/test-all.sh --quick --keep-going --report-dir reports/memory-production-core-v1/test-all-quick`.
Files / sources:
- `/home/tetra/Downloads/tetra_memory_production_core_v1_agent_plan_20260603.md` MPC-16 section
- `tools/cmd/verify-docs/main.go`
- `tools/cmd/validate-manifest/main.go`
- `docs/generated/manifest.json`
- `compiler/features.go`
- `docs/audits/memory-production-core-v1-final.md`
- `docs/audits/memory-production-core-v1-artifact-map.md`
- `docs/audits/memory-production-core-v1-nonclaims.md`
- `scripts/ci/test.sh`
- `scripts/ci/test-all.sh`
- `reports/memory-production-core-v1/`
- `reports/memory-fuzz-short/mpc15/`
Ownership: read-only. Do not edit files.
Do:
- Identify docs/manifest insertion points for the three final audit docs.
- Identify required nonclaims and exact wording risks.
- Identify what `test-all --quick` emits under the requested report directory and how the artifact map should reference it.
- Identify RED-test locations for missing final audit docs/nonclaims/artifact-map entries.
Do not:
- Modify files.
- Run long release commands.
- Convert test reports into safety claims beyond the evidence they actually provide.
Expected output:
- Accepted findings with file:line evidence.
- Rejected/non-issues.
- Recommended docs/manifest/test evidence checks.
- Uncertainties/blockers.
Verification:
- Evidence must name inspected files, commands, and artifact paths.
