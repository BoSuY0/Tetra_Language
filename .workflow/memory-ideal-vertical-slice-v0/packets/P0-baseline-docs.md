# P0-baseline-docs

Packet ID: P0-baseline-docs

Objective: Verify A0-lite baseline docs and identify docs/manifest hooks for
Memory Ideal Vertical Slice v0.

Context: The source plan requires these docs to exist and support ten baseline
assertions before implementation proceeds.

Files / sources:

- `/home/tetra/Downloads/tetra_memory_ideal_vertical_slice_plan_20260604.md`
- `docs/audits/memory-production-core-v1-final.md`
- `docs/audits/memory-production-core-v1-artifact-map.md`
- `docs/audits/memory-production-core-v1-nonclaims.md`
- `docs/audits/memory-production-core-v1-supported-surface.md`
- `docs/audits/memory-production-core-v1-gap-map.md`
- `docs/spec/memory_report_schema_v1.md`
- `docs/design/memory_production_core_v1.md`
- `docs/generated/manifest.json`
- docs validators under `tools/cmd/validate-manifest` and
  `tools/cmd/verify-docs`

Ownership: read-only.

Do: Report whether each required doc exists, which lines support each baseline
assertion, what the baseline classification should be, and how to update
manifest/docs safely.

Do not: Edit files, broaden scope, or treat report rows as truth.

Expected output: `.workflow/memory-ideal-vertical-slice-v0/results/P0-baseline-docs.md`
summary with evidence.

Verification: cite commands and file paths inspected.
