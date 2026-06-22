# P7 Compiler RSS Median Evidence Plan

**Goal:** Extend the P7 compiler RSS evidence harness from one-sample diagnostics to a repeatable
sample protocol that records raw runs, medians, dispersion, and compile-error coverage.

**Context:** `source-plan.md` requires P7 phase profiling, a test matrix across jobs/reports/cache
modes/module sizes, success and compile-error paths, at least 5 valid samples when claiming gates,
and final acceptance based on medians rather than selected best runs. The existing
`tools/cmd/ram-p7-compiler-rss` bundle is useful but currently records one synthetic sample per
scenario and does not cover compile-error paths.

## Task 1 - Add Sample Summary Data

- **Goal:** Record every sample for each scenario plus median RSS and dispersion fields.
- **Files:** `tools/internal/ramcompilerrss/collector.go`,
  `tools/internal/ramcompilerrss/collector_test.go`.
- **Approach:** Add a small sample count option with a conservative default for normal CLI runs.
  Keep every raw sample in `scenario-summary.json`; compute median from all valid samples rather
  than choosing a best run.
- **Verification:** Targeted `go test -count=1 ./tools/internal/ramcompilerrss -v` with a RED test
  that requires sample records and median fields.
- **Done when:** The summary schema exposes raw samples, `sample_count`, median RSS, and
  min/max/dispersion values for successful scenarios.

## Task 2 - Add Compile-Error Scenario Evidence

- **Goal:** Cover P7.5 compile-error paths without pretending they produce executable hashes.
- **Files:** `tools/internal/ramcompilerrss/collector.go`,
  `tools/internal/ramcompilerrss/collector_test.go`.
- **Approach:** Add a scenario mode that intentionally writes an invalid module and records the
  compiler error plus the phase profile generated before failure if available. Validator output
  should require the error path to be observed for scenarios marked compile-error.
- **Verification:** Targeted RED/GREEN test requiring a compile-error scenario in the bundle.
- **Done when:** The bundle includes raw error-path evidence and validator output passes only when
  the expected compile failure is observed.

## Task 3 - CLI And Bundle Regeneration

- **Goal:** Make the evidence reproducible from the CLI without heavy default runtime.
- **Files:** `tools/cmd/ram-p7-compiler-rss/main.go`,
  `tools/cmd/ram-p7-compiler-rss/main_test.go`,
  `docs/spec/telemetry/process_rss_telemetry.md`.
- **Approach:** Add a `--samples` flag. Use a small default for quick diagnostics and allow
  `--samples 5` or `--samples 7` for final P7 gates. Regenerate a non-ignored bundle with
  at least a focused sample count appropriate for this slice.
- **Verification:** CLI unit tests, focused tools/hash/docs tests, artifact hash validation,
  `git diff --check`, `graphify update .`, and persistent Go cache cleanup.
- **Done when:** A new bundle exists under `reports/stabilization/tetra-ram-p7-compiler-rss-*/`
  with sample summary fields, compile-error evidence, validator output, and artifact hashes.

## Non-Claims

- This slice does not claim final P7 completion.
- This slice does not claim baseline-vs-candidate RSS improvement unless a same-host baseline
  comparison bundle is generated and validated later.
- This slice does not modify P6 production mailbox payload ABI or descriptor/destructor behavior.
