# P7 Representative Compiler RSS Gate

Date: 2026-06-20

## Goal

Add a real repository-source compiler RSS matrix to complement the synthetic default and P7.5
matrices. The slice must preserve sample isolation, avoid writing compiler cache state into the
repository root, and keep the report-bound formula unchanged.

## Implementation Plan

- Add `--matrix representative` to `tools/cmd/ram-p7-compiler-rss`.
- Use `examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra` as the current
  representative source. It imports `lib.core.surface`, `lib.core.block`, and `lib.core.morph`.
- Copy the representative entry source into each sample-local project root, then use the repository
  root only as a dependency root. This keeps the sample `.tetra_cache` local and removable.
- Record `source_path` in scenario summaries and report comparisons so real-source scenarios cannot
  collide with synthetic scenarios that have the same module/job/cache shape.
- Keep report-off/report-on as a same-host samples5 pair and evaluate it with the existing
  sample-derived bound.

## Evidence

The generated bundle is:

```text
reports/stabilization/tetra-ram-p7-compiler-rss-b452638a8af7-representative-samples5/
```

It contains two scenarios, five samples per scenario, copied representative entry sources in every
sample, validated artifact hashes, and one report-off/report-on comparison. The comparison passes:

```text
report-off median: 56918016
report-on median:  58900480
bound:             68329472
ratio:             1.0348
```

## Boundary

This is representative-source Linux x64 evidence for one Surface Morph flagship source. It is not a
full-repository compiler RSS run, not cross-target parity evidence, and not final P7 completion by
itself.
