# Backend Coverage Audit v1

Status: P16.0 evidence audit for the Ideal Master Plan.

## Summary

The backend report now emits a coverage row for every lowered IR function and
classifies why each function follows the register path or falls back to the
stack backend. The audit also tags rows with a static hotness rank when the
function name appears in the checked-in benchmark corpus map.

## Report Fields

Each `.backend.json` function row records:

- `backend_path`: `register` or `stack`;
- `category`: one of `register_path`, `stack_fallback`,
  `unsupported_aggregate_return`, `unsupported_slice_string_return`,
  `unsupported_call_abi`, `unsupported_effect_runtime_call`, or
  `unsupported_control_flow`;
- `reason`: stable human-readable fallback or eligibility reason;
- `hotness_rank`: benchmark-corpus rank, with `0` meaning no corpus match;
- `hotness_source`: corpus artifact path or `not_in_benchmark_corpus`.

The report summary records function totals, register-path count, stack-fallback
count, category counts, and the hotness source policy.

## Evidence

| Check | Result |
| --- | --- |
| RED tests for category/hotness fields | fail before implementation |
| Focused backend report tests | pass |
| Existing explain-only backend report contract | pass |

## Boundaries

This audit is diagnostic only. It does not expose a backend mode, change build
semantics, remove runtime checks, or make performance claims. The hotness rank
is a static prioritization signal from checked-in benchmark corpus identifiers,
not a measured benchmark result.
