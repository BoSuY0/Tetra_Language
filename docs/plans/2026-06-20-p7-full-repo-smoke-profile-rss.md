# P7 Full-Repo Smoke-Profile Compiler RSS Slice

## Goal

Add a Linux x64 compiler RSS matrix that batches the current release smoke-profile entry points in
one measured compiler process per sample, then generate non-ignored evidence with report-off/on
comparison and target-scope non-claims.

## Implemented

- `tools/cmd/ram-p7-compiler-rss --matrix full_repo` selects a report-off/on pair named
  `full_repo_linux_x64_smoke_profile_*`.
- Each scenario compiles 71 current linux-x64 smoke entry points per sample.
- `scenario-summary.json` records `source_paths`, `source_count`, `compiler_profile_count`, and
  `executable_count` for batch scenarios.
- `validate-artifact-hashes` now emits structured success output and ignores the known
  `artifact-hashes-validation.txt` sidecar to avoid an impossible self-hash fixed point.

## Evidence

Bundle:

```text
reports/stabilization/tetra-ram-p7-compiler-rss-b452638a8af7-full-repo-smoke-samples2/
```

Key facts:

- 2 scenarios, 2 samples per scenario.
- 71 source paths per scenario.
- 284 compiler phase profiles.
- `validator-output.txt`: `result: pass`.
- `artifact-hashes-validation.txt`: `result: pass`.
- report comparison `full_repo_linux_x64_smoke_profile_reports_jobs_cpu_cold`: `pass`.
- report-off median `72699904`; report-on median `70897664`; bound `75960320`; delta `-1802240`;
  ratio `0.9752`.

## Boundary

This is a full-repository release smoke-profile workload for Linux x64 compiler RSS evidence. It is
not a claim that every `examples/` entrypoint candidate, library/runtime file, negative fixture,
test file, or target-specific example is a buildable linux-x64 entry point. It is also not a P8
target-parity claim; non-linux targets remain explicit non-claims in `target-scope.json`.
