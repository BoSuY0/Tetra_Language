# Raw Pointer Bounds Metadata v1 Audit

Status: P15.3 evidence audit for the Ideal Master Plan.

## Summary

The P15.3 slice adds reportable raw pointer bounds metadata for unsafe
`core.alloc_bytes` while preserving conservative unsafe semantics. Verified
allocation roots can carry allocation-base metadata and derived-offset facts.
Verified-root `ptr_add` and raw load/store evidence distinguishes negative
offset, upper-bound, and access-width-overflow rejection markers.
Arbitrary raw pointers remain `checked_external_unknown`, and raw-slice
gateways remain `external_unknown` unless they are constructed from verified
allocation-root metadata with proven length bounds. Verified raw-slice views
report `raw_slice_verified_allocation_root` as `unsafe_checked` evidence, not
safe provenance; negative lengths and length-byte overflow use
`rejected_negative_length` and metadata-level `rejected_length_overflow`
evidence. Linux-x64 target byte-overflow traps are runtime evidence and may
remain conservative in reports until target-aware metadata is added.

## Evidence

| Check | Result |
| --- | --- |
| Focused runtime ABI, PLIR, allocplan, memoryprod, smoke, and validator tests | pass |
| Relevant lower, validation, backend, compiler, and semantics package tests | pass |
| Memory production smoke report generation | pass |
| Memory production report validation | pass |

## Report Markers

`reports/raw-pointer-bounds-metadata-v1/memory-production-linux-x64.json`
contains:

- process `raw pointer bounds metadata report build`;
- contract `raw pointer bounds metadata`;
- case `raw pointer bounds metadata report`;
- audit requirement `raw pointer bounds metadata`;
- evidence markers `allocation_base_metadata`, `derived_allocation_offset`,
  `rejected_negative_offset`, `rejected_upper_bound`,
  `rejected_access_width_overflow`, `checked_external_unknown`,
  `external_unknown`, and `raw_slice_verified_allocation_root`.

## Boundaries

This audit does not claim safe semantics for arbitrary unsafe pointers, a GC, or
removal of runtime raw pointer checks. It only claims metadata and report
coverage for verified `core.alloc_bytes` allocation roots, linux-x64 runtime
diagnostics for the listed rejection cases, and conservative unknown-pointer
handling. Pointer-width access checks are linux-x64 runtime evidence in this
audit; other targets remain build/lower scoped until target-aware raw pointer
metadata is available.
