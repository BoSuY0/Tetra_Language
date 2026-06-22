# P7 Compiler RSS Sample Isolation And Report Release Slice

Date: 2026-06-20

## Goal

Reduce the current P7 compiler RSS evidence failures without changing the acceptance
definition:

- isolate each measured compiler RSS sample from earlier samples/scenarios in the
  in-process harness;
- release transient report-generation memory before the compiler phase profile's
  retained-state `report_generation` snapshot;
- regenerate the same-host P7 bundle, report-on/off comparison, and baseline/candidate
  comparison with validated hashes.

## Evidence Standard

- RED/GREEN coverage for harness sample isolation.
- RED/GREEN coverage for compiler post-report memory release before phase-profile
  snapshot.
- Regenerated non-ignored evidence under
  `reports/stabilization/tetra-ram-p7-compiler-rss-b452638a8af7-report-bound-samples5/`.
- all default `report_comparisons[].evaluation_status == "pass"` for small, medium, and large
  comparable report-off/report-on pairs.
- `baseline-candidate-comparison.json.overall_status == "pass"`.
- Focused tools/docs checks, full `./compiler/...`, `git diff --check`, `graphify update .`,
  and persistent-cache cleanup.

## Non-Goals

- Do not tune RSS bounds.
- Do not claim full P7 completion.
- Do not claim full RAM optimization plan completion.
