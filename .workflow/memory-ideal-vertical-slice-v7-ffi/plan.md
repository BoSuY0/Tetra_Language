# Plan: Memory Ideal Vertical Slice v7 FFI

## Goal

Implement the narrow Memory Ideal v7 external pointer and FFI lifetime
quarantine correlation slice.

## Current Strategy

Treat v6 as accepted `validated_narrow` evidence, then add exact v7 rows around
current external/raw pointer guardrails. Keep external pointers conservative or
rejected unless compiler-owned provenance exists. Do not prove arbitrary C-side
lifetimes, safe-wrapper promotion, broad noalias, target parity, or
performance.

## Phases

- [x] Compile v7 goal contract from the 2026-06-05 v6 acceptance roadmap.
- [x] Inspect v5/v6 patterns and current FFI/raw/external surfaces.
- [x] Add RED tests for v7 model/report/correlation/semantics behavior.
- [x] Implement v7 facts, validators, and projections.
- [x] Add v7 docs, schema notes, manifest entries, and final audit.
- [x] Run focused gates.
- [x] Run broad/docs/hygiene/Graphify gates.
- [x] Close final report and update goal completion evidence.

## Open Decisions

- Correlation should stay exactly four rows, `MEM-FFI-001` through
  `MEM-FFI-004`. Supporting source facts may include
  `external_pointer_provenance_rejected` if it is necessary for negative
  evidence.
- Owned-copy crossing FFI is positive evidence only where current semantics
  already support it. It must not become a broad FFI lifetime claim.
- Fuzz artifact policy remains outside v7 except for keeping the existing
  `validate-memory-fuzz-oracle` gate green.
- Target parity and performance remain explicit nonclaims.

## Next Action

Run the focused gate set from `GOAL.md`, starting with memoryfacts,
MiniMemoryModel, semantics, ownership, runtime ABI, tools, and v7 correlation.
