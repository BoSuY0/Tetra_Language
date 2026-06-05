Packet ID: MPC15-S1
Objective: Audit existing fuzz/property/differential tooling and identify the smallest MPC-15 memory oracle API and RED-test insertion points.
Context: MPC-15 requires memory fuzzing with explicit oracle categories, Tier 1 short CI smoke, Tier 2 nightly, Tier 3 release-blocking focused memory fuzz, and no unsupported unsafe/target claim inflation.
Files / sources:
- `compiler/fuzz_suite.go`
- `compiler/fuzz_property_differential_v1.go`
- `compiler/fuzz_property_differential_v1_test.go`
- `compiler/internal/differential/differential.go`
- `compiler/tests/fuzz/fuzz_pipeline_test.go`
- `scripts/dev/fuzz-nightly.sh`
- `tools/cmd/validate-fuzz-summary/main.go`
- `docs/testing/fuzz_property_stress.md`
- `/home/tetra/Downloads/tetra_memory_production_core_v1_agent_plan_20260603.md` MPC-15 section
Ownership: read-only. Do not edit files.
Do:
- Identify existing oracle/result enums or places where oracle categories should be added.
- Identify whether MPC-15 should extend P23.1 reports or add a memory-specific report/validator.
- Find concrete RED-test locations for checker reject, runtime trap, reference equality, compiler crash bug, miscompile bug, unsafe_unknown optimized-as-safe bug, and report validation failure.
- Identify how Tier 1/2/3 should map to existing script/report conventions.
Do not:
- Modify files.
- Require long fuzz execution for Tier 1.
- Expand fuzz surface beyond supported MPC memory features.
Expected output:
- Accepted findings with file:line evidence.
- Rejected/non-issues.
- Recommended design and RED tests.
- Uncertainties.
Verification:
- Evidence must name inspected files and relevant symbols.
