# Runtime Heap Telemetry

Status: linux-x64 implementation contract for local benchmark evidence.

## Purpose

Runtime heap telemetry records Tetra heap activity observed while a compiled
Tetra binary executes. It is evidence for local benchmark rows, not a global
memory claim.

This spec defines the sidecar artifact used by
`tools/cmd/local-benchmark-tier1` and
`tools/cmd/validate-local-benchmark-tier1`.

## Schema

Runtime heap sidecars use:

```text
tetra.runtime.heap_telemetry.v1
```

The current measured method is:

```text
tetra_linux_x64_heap_telemetry_v1
```

The first target is:

```text
linux-x64
```

## Required Fields

- `schema`: must be `tetra.runtime.heap_telemetry.v1`.
- `target`: must be `linux-x64` for this implementation.
- `method`: must be `tetra_linux_x64_heap_telemetry_v1`.
- `program`: benchmark/program identity.
- `pid`: process id when available.
- `exit_status`: runtime exit status captured by the telemetry path.
- `heap_current_bytes`: current live Tetra heap bytes known to the telemetry
  path at sidecar write time.
- `heap_peak_bytes`: peak live Tetra heap bytes known to the telemetry path.
- `heap_total_alloc_bytes`: cumulative Tetra heap bytes allocated by counted
  heap paths.
- `heap_allocation_count`: number of counted heap allocations.
- `bytes_requested`: logical bytes requested by counted heap allocation calls.
- `bytes_reserved`: bytes reserved from the backend/OS by counted heap paths.
- `allocation_paths`: count by allocation path, such as `small_heap_bump`,
  `small_heap_refill`, `large_mmap`, or `alloc_bytes_mmap`.
- `domain_bytes`: optional per-domain byte summary.
- `actor_snapshot_record_count`: optional actor-telemetry record count when
  actor domains are emitted; this includes live actors plus reusable/done slots
  that still carry retained or released accounting state.
- `actor_live_count`: optional actor-telemetry live slot count when actor
  domains are emitted; this excludes `done` and `free` actor slots from the live
  total.
- `notes`: optional limitations or implementation notes.

`started_unix_nano` and `finished_unix_nano` are optional because some minimal
native runtimes may not have a time source wired into the telemetry path.

## Invariants

- `heap_peak_bytes >= heap_current_bytes`.
- `heap_total_alloc_bytes >= heap_peak_bytes`.
- `heap_allocation_count == 0` is valid only when all heap byte totals are
  zero.
- if `bytes_reserved` is non-zero, it must be greater than or equal to
  `heap_peak_bytes`.
- every `allocation_paths` key must be non-empty and every count must be
  positive.
- every `domain_bytes` item must have a non-empty `domain_id` and `kind`.
- per-domain `peak_bytes` must be greater than or equal to per-domain
  `current_bytes`.
- when present, `actor_snapshot_record_count >= actor_live_count`;
  reusable/done actor slots must not be counted as live actors merely because
  they still have snapshot records for retained or released stack accounting.

## Evidence Mapping

In local Tier 1 benchmark memory evidence:

- `heap_alloc_bytes.evidence_class == runtime_measured` means the value came
  from a `tetra.runtime.heap_telemetry.v1` sidecar for the benchmarked Tetra
  binary.
- `heap_alloc_bytes.method` must be
  `tetra_linux_x64_heap_telemetry_v1`.
- `heap_alloc_bytes.source_artifact` must point to the raw or row-summary heap
  telemetry artifact inside the report directory.
- If the legacy `bytes` field is present for `heap_alloc_bytes`, it means
  `heap_peak_bytes`.
- `current_bytes`, `peak_bytes`, `total_alloc_bytes`, and
  `allocation_count` should be present when the report schema supports them.
- When a benchmark has multiple iterations, the row-level
  `heap_alloc_bytes` metric points at the collected sidecar with the maximum
  observed `heap_peak_bytes`.
- A successful row may report zero heap bytes when the telemetry sidecar proves
  that the optimized binary performed no counted Tetra heap allocation at
  runtime. Zero is not a replacement for a missing sidecar.

## Explicit Nonclaims

The following are not valid evidence for
`heap_alloc_bytes.runtime_measured`:

- Go `runtime.MemStats` from the benchmark runner or compiler process.
- process RSS from `/proc`, `getrusage`, or another process sampler.
- allocation-plan reports or compiler estimates.
- binary size.
- `mmap` reserved bytes alone.
- C, C++, Rust, wasm, linux-x86, linux-x32, macOS, or Windows observations.

RSS is reported separately by `docs/spec/process_rss_telemetry.md` when a
process-level sampler is enabled, but RSS is not Tetra heap telemetry.

## Failure Semantics

If a Tetra row builds or runs successfully on linux-x64 but lacks a valid heap
sidecar, the row must not claim runtime-measured heap evidence.

Valid alternatives are:

- `blocked` with a concrete build/run/telemetry reason;
- `unsupported` only for targets or configurations that the telemetry contract
  explicitly does not support.

For successful linux-x64 Tier 1 Tetra rows, `unsupported` heap evidence is a
failure once this feature is enabled.
