# MPC13-S1 Target Evidence Audit

## Purpose

Read-only audit for MPC-13 Target capability matrix.

Find the concrete repo evidence that distinguishes build, lower, run, raw diagnostics, region lowering, alignment semantics, and claim level per target. Focus on existing target validators, runtime ABI metadata, memory production smoke reports, target codegen tests, and release artifacts.

## Scope

- `tools/cmd/validate-targets/`
- `tools/cmd/validate-memory-production/`
- `tools/validators/memoryprod/`
- `compiler/target/`
- `compiler/internal/runtimeabi/`
- `compiler/internal/backend/`
- `reports/memory-production-core-v1/`
- target-related docs under `docs/spec/`, `docs/audits/`, and `docs/release/`

## Questions

1. Which targets have build evidence, lowering evidence, and actual runtime evidence?
2. Where does linux-x64 memory production runtime evidence enter the repo?
3. Which existing validators could reject claim inflation with the least new surface area?
4. Which fake claims should RED tests exercise for runtime, raw diagnostics, region lowering, and alignment?
5. Which files/tests should the orchestrator inspect before editing?

## Output Contract

Return a concise Markdown result with:

- files inspected;
- commands run, if any;
- evidence table by target;
- recommended validator/test insertion points;
- uncertainties or gaps.

Do not edit files.
