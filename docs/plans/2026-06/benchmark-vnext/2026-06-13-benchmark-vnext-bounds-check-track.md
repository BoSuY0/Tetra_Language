# Benchmark vNext Bounds-Check Track

Status: follow-up plan opened from the fresh memory-aware Tier 1 baseline.

Primary audit: `docs/audits/memory/zero-heap-final/benchmark-vnext-memory-baseline.md`.

## Goal

Eliminate proven redundant bounds checks in the primary Tier 1 rows while preserving typed proof
evidence and translation validation.

## Current Evidence

Fresh report: `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/report.json`.

Rows:

| Row                        | Checks left | Artifact                                                                                                            |
| -------------------------- | ----------: | ------------------------------------------------------------------------------------------------------------------- |
| `slice_sum_tetra`          |           2 | `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/slice_sum_tetra.bounds.json`          |
| `bounds_check_loops_tetra` |           2 | `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/bounds_check_loops_tetra.bounds.json` |
| `matrix_multiply_tetra`    |           7 | `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/artifacts/bin/matrix_multiply_tetra.bounds.json`    |

The `slice_sum_tetra` artifact shows `left_missing_dominance` for both the store and load sites.
This is the best first target because the loop shape is simple: `i` starts at zero and increments
while `i < n`.

## Proposed Slice

Start with `slice_sum_tetra` loop dominance/range proof preservation.

First acceptance target:

- remove or reduce the two `slice_sum_tetra` checks when the loop guard dominates `xs[i]`;
- preserve a proof id/reason for every removed check;
- keep retained checks explicit when proof is missing.

Do not start with matrix multiply or modulo-based `bounds-check loops`; those need broader
affine/modulo reasoning.

## Reporting Boundary

The current `.bounds.json` artifacts expose `reason`, such as `left_missing_dominance`, but the
fresh audit had to inspect sites manually. The follow-up patch should keep reason strings stable
enough to group blockers by cause and should avoid reporting a removed check without compiler-owned
proof metadata.

## Likely Files

- `compiler/compiler_reports.go`
- `compiler/compiler_reports.go`
- `compiler/compiler_suite_test.go`
- `compiler/internal/validation/validation.go`
- `compiler/internal/validation/validation_translation.go`
- `compiler/internal/opt/opt_core.go`
- `compiler/internal/memoryfacts`
- `tools/internal/localbenchmarktier1/specs/tetra_sources.go`

## Tests First

Add or extend focused tests that fail before the implementation:

- a validation/proof test for a `while i < n` loop that removes `xs[i]` bounds checks with a proof
  id;
- a report test that keeps `left_missing_dominance` visible for unsupported shapes;
- a translation/differential test that rejects removed checks without proof.

## Verification

Focused:

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-bounds go test ./compiler/internal/validation/... ./compiler/internal/opt/... ./compiler/internal/memoryfacts/... ./compiler -run 'Bounds|Range|Dominance|Proof|Translation' -count=1
```

Benchmark:

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-bounds go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-after-bounds-track --iterations 3
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-bounds go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-after-bounds-track/report.json
```

## Nonclaims

- No global bounds-check-free claim.
- No unsafe removal without typed proof and validator evidence.
- No performance claim until the fresh benchmark is rerun and validated.
