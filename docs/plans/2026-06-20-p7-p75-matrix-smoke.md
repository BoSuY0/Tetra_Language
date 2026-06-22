# P7 P7.5 Compiler RSS Matrix Smoke

Date: 2026-06-20

## Goal

Extend `tools/cmd/ram-p7-compiler-rss` beyond the default quick matrix so the P7.5 source-plan
dimensions can be executed as a reproducible Linux x64-first diagnostic bundle.

## Matrix

`--matrix p7_5` covers the Cartesian product of:

- module graph size: small, medium, large;
- jobs: 1, 2, 4, and `runtime.NumCPU()` with duplicate worker counts deduplicated;
- reports: off and on;
- compiler cache: cold and warm;
- result path: successful build and expected compile error.

Warm expected compile-error scenarios first warm the dependency cache with a valid entry point, then
restore the failing entry point before the measured build. This keeps `warm_cache: true` meaningful
for error-path evidence.

## Evidence Boundary

The first generated bundle is a smoke run:

```text
reports/stabilization/tetra-ram-p7-compiler-rss-b452638a8af7-p75-smoke-samples1/
```

It proves that every P7.5 synthetic scenario is runnable and hash-validated. It does not prove the
P7.7 median RSS/report-bound acceptance gate because each scenario has one sample. The final P7.7
gate still requires at least five valid samples per scenario on the same host/configuration.
