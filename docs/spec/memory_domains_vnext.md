# Memory Domains vNext

Status: vNext report/runtime substrate contract for
`docs/plans/2026-06-13-tetra-memory-model-vnext.md`.

This document extends the current Tetra Memory Model. It does not replace
ownership, islands, allocation planning, runtime allocation contracts, or RAM
contract reports.

## Goal

`MemoryDomain` names the owner and lifetime boundary for memory bytes before
the runtime has a full target-specific allocator. It lets compiler reports,
runtime reports, and future actor memory work speak in one vocabulary without
turning Linux RSS behavior into language semantics.

Domains are accounting scopes. They are not proof by themselves. A row can be
assigned to a domain only with its existing allocation, lifetime, and placement
evidence.

## Domain Kinds

| Kind | Meaning |
| --- | --- |
| `process` | Default process-wide domain for ordinary heap, stack, static, or eliminated allocation rows. |
| `task` | Task-owned lifetime region or task runtime accounting scope. |
| `actor` | Actor-owned memory scope for owned regions, message slabs, and mailbox pools. |
| `island` | Explicit island scope with island lifetime evidence. |
| `request` | Request-owned domain for future server/request pipeline accounting. |
| `external` | Memory owned by a host, FFI boundary, browser, kernel, or target adapter. |

`process`, `task`, `actor`, `island`, and `request` are target-neutral model
terms. They do not imply a Linux process, pthread, epoll loop, or OS-specific
primitive.

## Fields

Every domain record has:

| Field | Meaning |
| --- | --- |
| `domain_id` | Stable report-local identifier such as `domain:process` or `domain:island:main`. |
| `parent_domain_id` | Optional parent accounting scope. |
| `kind` | One of the domain kinds above. |
| `owner_kind` | Owner class, for example `process`, `task`, `actor`, `island`, or `external`. |
| `owner_id` | Stable owner identifier inside the report. |
| `lifetime` | Lifetime boundary inherited from allocation or runtime evidence. |
| `budget_bytes` | Budget charged to this domain row or aggregate. |
| `requested_bytes` | Bytes requested by source/runtime allocation intent. |
| `reserved_bytes` | Bytes reserved by allocation class, region, chunk, or backend reservation. |
| `committed_bytes` | Bytes committed by the backend when measured or reported. |
| `released_bytes` | Bytes released back to the backend or host. |
| `current_bytes` | Current measured or estimated domain footprint. |
| `peak_bytes` | Peak measured or estimated domain footprint. |
| `copy_count` | Number of copy events charged to the domain row or aggregate. |
| `bytes_copied` | Bytes copied for copy rows charged to the domain. |

`current_bytes` and `peak_bytes` require separate footprint evidence from the
`MemoryBackend` contract before they can be treated as measured RSS or measured
runtime footprint.

## RAM Contract Projection

`tetra.ram-contract-report.v1` may attach optional `domain` metadata to each
row. The current projection is conservative:

- ordinary heap/default rows are charged to `domain:process`;
- explicit island rows are charged to `domain:<region_id>`;
- task-region rows are charged to a `task` domain;
- actor-move-region rows are charged to an `actor` domain;
- external rows are charged to an `external` domain;
- copy rows record `copy_count` and `bytes_copied`.

The report `summary.domains` field is derived from rows, sorted
deterministically, and validated as part of the normal summary equality check.

## Allocation Report Projection

Allocation plan reports may attach the same nested `domain` object to allocation
rows. This is allocator evidence, not footprint/RSS evidence:

- per-core small-heap, heap, stack, register, eliminated, and large-mmap rows
  are charged to `domain:process`;
- explicit island rows are charged to their island domain;
- external rows are charged to an external domain;
- planned function-temp and actor-move storage remains conservative until a
  runtime ownership-transfer report proves stronger domain movement.

The allocation summary may include `domains[]` aggregates. These aggregates are
derived from allocation rows and carry requested/reserved accounting only unless
a separate runtime/backend sample provides committed/current/peak evidence.

## Actor Memory Domain Direction

Actor domains are the substrate for future actor memory work:

- mailbox pools;
- message slabs;
- actor-owned regions;
- per-actor budget and peak bytes;
- byte-based backpressure;
- zero-copy ownership transfer as a domain owner change.

The current report vocabulary does not claim runtime actor mailbox pooling,
message-slab reuse, or distributed actor zero-copy.

## Nonclaims

- This is not a new memory model.
- This does not claim zero heap for all programs.
- This does not claim RSS in bytes for all targets.
- This does not claim actor zero-copy is implemented end to end.
- This does not make allocation-report estimates equal to measured runtime
  footprint.
- This does not claim production allocator performance.

## Promotion Requirements

Before domain memory can be promoted from report vocabulary to runtime evidence,
the implementation must provide:

1. Runtime or backend samples with explicit evidence class and method.
2. Validator coverage for measured, estimated, unsupported, and blocked domain
   samples.
3. Per-domain budget and peak checks in release gates.
4. Actor mailbox and message-byte backpressure evidence before actor domains
   can claim runtime enforcement.
5. Target adapter evidence for each target that claims measured footprint.
