# MPC13-S2 Docs Manifest Audit

## Purpose

Read-only audit for MPC-13 documentation and manifest gates.

Identify how `docs/audits/memory-target-capability-matrix.md` should be linked through docs validation without inflating claims beyond target-specific evidence.

## Scope

- `docs/generated/manifest.json`
- `tools/cmd/verify-docs/`
- `tools/cmd/validate-manifest/`
- `docs/audits/`
- `docs/spec/current_supported_surface.md`
- `docs/spec/runtime_abi.md`
- `docs/design/memory_production_core_v1.md`
- release audit docs that mention memory production target scope

## Questions

1. Is a target capability matrix already required by docs validators or manifest?
2. Which docs should link to the new matrix?
3. What exact wording avoids cross-target parity claims?
4. Which docs/manifest RED tests should fail before the matrix exists or is linked?
5. Are there stale claims that need narrowing for macOS, Windows, wasm, linux-x86, or linux-x32?

## Output Contract

Return a concise Markdown result with:

- files inspected;
- commands run, if any;
- docs/manifest insertion points;
- claim wording risks;
- recommended RED tests;
- uncertainties or gaps.

Do not edit files.
