# Memory Audit Index

Status: canonical index for memory audit evidence.

This index keeps historical memory reports readable without rewriting them.
Memory Core v2 current evidence is anchored by:

- spec: `docs/spec/memory/memory_core_v2.md`;
- T15 evidence schema validator: `tools/cmd/validate-memory-core-v2`;
- T15 release subgate: `scripts/release/memory/memory-core-v2-gate.sh`;
- checksum inventory:
  `reports/stabilization/memory-core-v2/memory-audit-checksums.tsv`.

## Current Supported Surface

The current Memory Core v2 surface is narrow:

- canonical memory state is built in the normal compiler path;
- reports are projections of decisions already used by the build;
- island domain direct-route and lifecycle evidence is required;
- Linux-x64 backend operation evidence is required for supported backend rows;
- optimizer memory rewrites require canonical proof IDs;
- cache hits require memory plan/lowering attestation.

Explicit boundaries:

- no universal memory safety claim;
- no universal performance claim;
- no zero heap for all programs claim;
- no all-target memory support claim;
- no all-target backend runtime claim.

## Canonical Families

- Production core v1 history:
  `docs/audits/memory/production/`
- Island and Memory 100 history:
  `docs/audits/memory/islands/`
- RAM/raw memory history:
  `docs/audits/memory/ram-raw/`
- Ideal vertical slices:
  `docs/audits/memory/ideal-v0-v1/`,
  `docs/audits/memory/ideal-v2-v4/`,
  `docs/audits/memory/ideal-v5-v7/`,
  `docs/audits/memory/ideal-v8-v9/`, and
  `docs/audits/memory/ideal-v10-v11/`
- Zero-heap historical work:
  `docs/audits/memory/zero-heap-core/`,
  `docs/audits/memory/zero-heap-runtime/`, and
  `docs/audits/memory/zero-heap-final/`

Top-level `docs/audits/*memory*.md` files are historical aliases or earlier
placement paths unless this index names them as canonical. They remain available
for inbound links and audit traceability.

## Duplicate Cleanup

Checksum inventory was built from `docs/audits/**/*memory*.md`. The inventory
found no byte-identical duplicate groups at T15 time, so no audit file was
deleted and no historical report was rewritten.

Historical reports with different results are intentionally preserved. The
status correction at
`docs/audits/memory/zero-heap-final/post-zero-heap-native-memory-dump-status-correction-2026-06-18.md`
remains authoritative for the affected zero-heap/native-memory dump status until
new same-commit evidence passes the Memory Core v2 gate.
