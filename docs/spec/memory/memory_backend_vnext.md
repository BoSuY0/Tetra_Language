# Memory Backend vNext

Status: vNext contract for
`docs/plans/2026-06/actors-memory/2026-06-13-tetra-memory-model-vnext.md`.

This document extends the existing Tetra Memory Model. It does not replace ownership, islands,
allocation planning, runtime allocation contracts, or RAM contract reports.

## Goal

`MemoryBackend` is the target-neutral substrate that lets runtime and reporting layers describe
memory reservation, commitment, release, trimming, and footprint evidence without making
Linux-specific semantics part of the language model.

Linux-x64 is the first concrete measured adapter. Other targets may report `unsupported` or
`blocked` until they have their own adapter and validator evidence.

## Operations

Every backend contract exposes the same operation names:

| Operation   | Meaning                                                                 |
| ----------- | ----------------------------------------------------------------------- |
| `reserve`   | Reserve address space or target-equivalent backing capacity.            |
| `commit`    | Make reserved bytes usable by the runtime.                              |
| `decommit`  | Return committed pages/segments to an uncommitted state when supported. |
| `release`   | Release the reservation or target-equivalent storage owner.             |
| `trim`      | Ask the backend to return idle memory to the host when supported.       |
| `footprint` | Report current and peak process/domain footprint evidence.              |

The operation names are stable reporting terms. A target adapter may implement them with `mmap`,
`VirtualAlloc`, linear memory, host APIs, or another target primitive, but those primitives are
adapter details.

## Byte Terms

| Field             | Meaning                                                                   |
| ----------------- | ------------------------------------------------------------------------- |
| `requested_bytes` | Bytes requested by source/runtime allocation intent.                      |
| `reserved_bytes`  | Bytes reserved by allocator class, region, chunk, or backend reservation. |
| `committed_bytes` | Bytes committed/usable according to the backend adapter.                  |
| `released_bytes`  | Bytes released back to the backend or host.                               |
| `current_bytes`   | Current measured or estimated footprint for the sample scope.             |
| `peak_bytes`      | Peak measured or estimated footprint for the sample scope.                |

`requested_bytes` and `reserved_bytes` may come from allocation reports. They are not RSS by
themselves. `current_bytes` and `peak_bytes` must carry an evidence class and method.

## Evidence Classes

| Evidence class               | Meaning                                                               |
| ---------------------------- | --------------------------------------------------------------------- |
| `runtime_measured`           | Runtime or host adapter measured the sample directly.                 |
| `allocation_report_estimate` | Compiler/report summary estimated bytes or syscalls.                  |
| `unsupported`                | Target cannot provide the metric in the current adapter boundary.     |
| `blocked`                    | Metric could be provided, but the tool or permission was unavailable. |

Measured, estimated, unsupported, and blocked samples are different evidence. Validators must reject
reports that use an estimate as measured RSS/footprint evidence.

`tetra.memory.ram-measurement.v1` applies the same rule with required metric samples for
`heap_alloc_bytes`, `bytes_requested`, `bytes_reserved`, `bytes_copied`, `rss_current`, `rss_peak`,
and `per_actor_domain_bytes`. MemStats can measure heap allocation fields, but it is not accepted as
process-RSS evidence.

## Current Runtime ABI Model

`compiler/internal/runtimeabi` owns the current in-repo data model:

- schema: `tetra.memory.backend-contract.v1`;
- operations: `reserve`, `commit`, `decommit`, `release`, `trim`, `footprint`;
- evidence classes: `runtime_measured`, `allocation_report_estimate`, `unsupported`, and `blocked`;
- first measured target policy: `linux-x64` with method `linux_proc_status`;
- WASM targets currently report unsupported footprint evidence because host RSS is unavailable at
  the current linear-memory boundary.

The contract preserves existing allocation paths: `heap`, `process_bump_small_heap_v0`,
`large_mmap`, `explicit_island`, `scoped_single_mapping_v0`, `region`, `stack_frame`, and
`eliminated`. `per_core_small_heap` remains model-only/future evidence until emitted runtime code has
matching per-core state and reuse.

## Nonclaims

- This is not a second memory model.
- This does not claim zero heap for all programs.
- This does not claim all-target RAM/RSS parity.
- This does not turn allocation-report estimates into runtime measurements.
- This does not claim performance superiority.

## Promotion Requirements

Before a target can claim measured footprint support, it must have:

1. A target adapter method with stable evidence class and method name.
2. Validator coverage for measured, unsupported, and blocked samples.
3. Report docs that distinguish heap/allocation bytes from process or domain footprint bytes.
4. Release-gate evidence that rejects stale, fake, and overclaiming reports.
