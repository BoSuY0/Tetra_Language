# Claim Tiers V1

Status: P20.2 bounded claim-policy evidence slice.

This audit records the public performance wording contract for the master-plan P20.2 gate. It
defines the allowed tiers and validates that current P20.0/P20.1 evidence stays at Tier 0
local-smoke wording only.

It does not run benchmarks, does not promote dry-run artifacts to measured performance, does not
claim C/C++/Rust parity, does not claim official TechEmpower, and does not change optimizer,
runtime, or safe-program semantics.

## Policy Surface

The implementation lives in:

- `tools/cmd/truth-bench-harness/claim_tiers.go`
- `tools/cmd/truth-bench-harness/main.go`
- schema `tetra.performance.claim_tiers.v1`
- scope `p20.2_claim_tiers`

The exact allowed tiers are:

- `tier0_local_smoke_only`: Tier 0: local smoke only.
- `tier1_local_benchmark_evidence`: Tier 1: local benchmark evidence.
- `tier2_reproducible_cross_machine_benchmark`: Tier 2: reproducible cross-machine benchmark.
- `tier3_independent_reproduced_benchmark`: Tier 3: independent reproduced benchmark.
- `tier4_official_upstream_benchmark_submission`: Tier 4: official upstream benchmark submission.

## Checked Artifact

The checked report is:

- `reports/claim-tiers-v1/claim-tier-report.json`

It records:

- schema `tetra.performance.claim_tiers.v1`
- scope `p20.2_claim_tiers`
- five exact policy rows
- current claim `p20_current_local_smoke_only`
- tier `tier0_local_smoke_only`
- evidence classes `local_smoke`, `dry_run_matrix`, and `performance_blocker_report`
- non-claims for measured speed, C++/Rust parity, official benchmark, official TechEmpower,
  cross-machine reproduction, independent reproduction, throughput advantage, and latency advantage

The artifact is generated through:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/truth-bench-harness --claim-tiers-out reports/claim-tiers-v1/claim-tier-report.json
```

## Validation

`ValidateClaimTierReport` rejects:

- missing or unknown schemas and scopes
- missing, duplicate, or drifted tier policy rows
- unknown public-claim tiers
- missing required evidence classes for a declared tier
- placeholder claim text or placeholder evidence
- local benchmark wording without Tier 1 evidence
- cross-machine wording without Tier 2 evidence
- independent reproduction wording without Tier 3 evidence
- official/upstream/TechEmpower wording without Tier 4 evidence
- fastest-language, C++/Rust parity, production database benchmark, measured speed, throughput
  advantage, and latency advantage wording unless explicit non-claim text is used

The existing unstructured `validateClaims` path now uses the same tier wording guard for public
report claim notes while preserving row-specific benchmark claim wording that cites a concrete
row/report/target/runtime comparison.

## Current Boundary

Current P20.0 and P20.1 evidence maps only to Tier 0:

- P20.0 proves a dry-run benchmark-matrix contract with `ran=false`.
- P20.1 proves actionable performance-blocker explanations.
- Neither artifact proves local measured benchmark performance, cross-machine reproducibility,
  independent reproduction, or official upstream submission.

Future public wording must move through the next tier only when the matching evidence class exists
and the validator accepts the report.

## Verification

Focused evidence commands:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./tools/cmd/truth-bench-harness -run 'TestP20ClaimTier|TestValidateP20ClaimTier|TestValidateClaimsRejectsFakeHigherTierWording|TestValidateReportRejectsBroadClaims' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/truth-bench-harness --claim-tiers-out reports/claim-tiers-v1/claim-tier-report.json
```
