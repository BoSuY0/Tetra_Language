# P7 Report-On/Off RSS Bound Comparison Plan

**Goal:** Extend the P7 compiler RSS bundle so report-on scenarios are paired with matching
report-off baselines and the summary records a same-host, sample-derived bound comparison.

**Context:** P7.7 requires report-on peak RSS to be bounded relative to report-off using a
baseline-derived gate. The current samples5 bundle records raw samples and medians, but does not
publish a structured report-on/report-off comparison or bound basis.

## Task 1 - RED Comparison Coverage

- **Goal:** Prove the bundle summary exposes report-on/report-off comparison rows.
- **Files:** `tools/internal/ramcompilerrss/collector_test.go`.
- **Approach:** Add a pure comparison test with deterministic synthetic summaries, plus a `Run`
  assertion that a matching off/on scenario pair writes `report_comparisons` into
  `scenario-summary.json`.
- **Verification:** `GOTELEMETRY=off GOCACHE="$(pwd)/.cache/go-build-ram-p7-report-bound-red" go test -count=1 ./tools/internal/ramcompilerrss -run 'TestReportComparison|TestRunWritesCompilerRSSBundle' -v`.

## Task 2 - Implement Same-Host Bound Metadata

- **Goal:** Record comparison status without claiming final P7 completion.
- **Files:** `tools/internal/ramcompilerrss/collector.go`,
  `docs/spec/telemetry/process_rss_telemetry.md`.
- **Approach:** Pair non-error report-off/report-on scenarios by module count, jobs, warm-cache,
  and memory budget. Compute a bound as report-off median plus observed off/on dispersion. Mark
  rows as `pass`, `fail`, or `insufficient_samples`, and keep validator semantics focused on
  artifact integrity rather than final acceptance.
- **Verification:** Targeted GREEN plus focused harness/CLI/hash/docs checks.

## Task 3 - Evidence And Kernel Update

- **Goal:** Produce a non-ignored bundle containing report comparison metadata and record limits.
- **Files:** `reports/stabilization/...`, `.workflow/tetra-ram-optimization-master-plan/**`.
- **Verification:** bundle generation with `--samples 5`, artifact hash write/validate,
  `./tools/internal/ramcompilerrss`, `./tools/cmd/ram-p7-compiler-rss`, docs verifier,
  `git diff --check`, `graphify update .`, and persistent Go cache cleanup.
