# RAM Contract Compiler Handoff

Release gate: `scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh`
CI job: `ram-contract-release-readiness-linux`
Package workflow: `ram-contract-linux-x64`

## Required Artifacts

Required artifacts:

- `ram-contract-report.json`
- `memory-grade-report.json`
- `proof-store-summary.json`
- `validation-pipeline-coverage.json`
- `heap-blockers.json`
- `copy-blockers.json`
- `ram-contract-fuzz-oracle.json`
- `artifact-hashes.json`

## Operator Notes

Run the release gate with a fresh report directory. The gate runs the compiler with `--emit-ram-contract-report`, validates the report bundle, runs the RAM contract fuzz oracle, and validates the artifact hashes. Reusing a non-empty report directory is rejected so stale reports cannot satisfy the gate.

## Handoff Boundaries

This handoff gives downstream release packaging a scoped RAM contract evidence bundle. It does not claim production object memory, production persistent memory, zero heap for all programs, zero-copy for all programs, full formal proof, or target parity across all targets.

- nonclaim: no all-target RAM parity claim
