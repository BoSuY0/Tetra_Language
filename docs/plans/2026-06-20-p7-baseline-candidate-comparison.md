# P7 Baseline/Candidate Compiler RSS Comparison Slice

Date: 2026-06-20

## Goal

Add a same-host bundle-to-bundle comparison for P7 compiler RSS evidence. The comparison must use
existing `ram-p7-compiler-rss` bundles, preserve all raw-sample evidence in the source bundles, and
emit a structured verdict based on median RSS plus observed dispersion rather than a hard-coded
universal percentage.

## Scope

- Add `ramcompilerrss.CompareBundles` for comparing a baseline bundle directory with a candidate
  bundle directory.
- Match scenarios by scenario name and verify scenario configuration compatibility.
- Require same host/config metadata before treating the comparison as an acceptance input.
- Compare median RSS for each baseline scenario against a baseline-derived bound:
  `baseline_median + baseline_dispersion + candidate_dispersion`.
- Record scenario-level verdicts plus candidate-only scenario notes.
- Keep this as evidence input only. A failed comparison is useful evidence and must not be reported
  as final P7 completion.

## Verification

1. RED: focused `ramcompilerrss` test fails before the comparison API exists.
2. GREEN: focused package tests pass.
3. Generate a comparison artifact for the existing samples5 baseline and report-bound candidate.
4. Run focused tools/docs gates, full compiler gate if implementation touches compiler paths, then
   `git diff --check`, `graphify update .`, persistent Go cache cleanup, and workflow kernel update.
