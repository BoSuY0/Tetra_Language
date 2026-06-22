# P7 P7.5 Compiler RSS Samples5 Gate

Date: 2026-06-20

## Goal

Promote the P7.5 compiler RSS matrix from smoke coverage to same-host multi-sample evidence without
changing the report-bound formula or treating a narrower local check as final P7 completion.

## Measurement Fixes

- Run one compiler-process warmup before measured scenarios. The warmup uses a removed
  `.process-warmup-*` scratch directory under the bundle root and does not warm any scenario
  `.tetra_cache`.
- Keep `p7_5` report-off/report-on scenarios adjacent for the same module graph, jobs, cache mode,
  and outcome so report-bound comparisons are not biased by unrelated in-process scenario drift.
- Release Go-retained compiler memory before the `object_retention_link` phase snapshot while
  native object references are still retained and counted.

## Evidence

The generated bundle is:

```text
reports/stabilization/tetra-ram-p7-compiler-rss-b452638a8af7-p75-samples5/
```

It contains 96 scenarios, 5 samples per scenario, 48 expected compile-error scenarios, and 24
report-off/report-on comparisons. All 24 report comparisons pass. The largest positive report-on
delta is `p75_medium_success_reports_jobs_cpu_cold`, which remains inside the sample-derived bound.

## Boundary

This is Linux x64-first synthetic compiler RSS evidence. It does not claim final P7 completion for
full repository/representative-large-build RSS or cross-target parity by itself.
