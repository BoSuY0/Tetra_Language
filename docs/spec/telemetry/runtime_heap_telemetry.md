# Runtime Heap Telemetry

Status: linux-x64 implementation contract for local benchmark evidence.

## Purpose

Runtime heap telemetry records Tetra heap activity observed while a compiled Tetra binary executes.
It is evidence for local benchmark rows, not a global memory claim.

This spec defines the sidecar artifact used by `tools/cmd/local-benchmark-tier1` and
`tools/cmd/validate-local-benchmark-tier1`.

## Schema

Runtime heap sidecars use:

```text
tetra.runtime.heap_telemetry.v1
```

P0 truthful-evidence sidecars use:

```text
tetra.runtime.heap_telemetry.v2
```

The current measured method is:

```text
tetra_linux_x64_heap_telemetry_v1
```

The P0 v2 method is:

```text
tetra_linux_x64_heap_telemetry_v2
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
- `heap_current_bytes`: current live Tetra heap bytes known to the telemetry path at sidecar write
  time.
- `heap_peak_bytes`: peak live Tetra heap bytes known to the telemetry path.
- `heap_total_alloc_bytes`: cumulative Tetra heap bytes allocated by counted heap paths.
- `heap_allocation_count`: number of counted heap allocations.
- `bytes_requested`: logical bytes requested by counted heap allocation calls.
- `bytes_reserved`: bytes reserved from the backend/OS by counted heap paths.
- `allocation_paths`: count by allocation path, such as `small_heap_bump`, `small_heap_refill`,
  `large_mmap`, or `alloc_bytes_mmap`.
- `domain_bytes`: optional per-domain byte summary.
- `actor_snapshot_record_count`: optional actor-telemetry record count when actor domains are
  emitted; this includes live actors plus reusable/done slots that still carry retained or released
  accounting state.
- `actor_live_count`: optional actor-telemetry live slot count when actor domains are emitted; this
  excludes `done` and `free` actor slots from the live total.
- `notes`: optional limitations or implementation notes.

## V2 Truth Fields

`tetra.runtime.heap_telemetry.v2` preserves v1 compatibility where useful, but its claim-bearing
fields are separated from compile-time estimates and report labels:

- `allocator_mode`: truthful emitted allocator mode, for example `process_bump_small_heap_v0`.
- `allocator_state_scope`: ownership of allocator state, for example `process`.
- `allocator_claims`: optional capability labels; labels such as `per_core` must match real emitted
  state scope.
- `successful_alloc_payload_bytes`: payload bytes actually accepted by counted runtime allocation
  paths.
- `successful_drop_payload_bytes`: payload bytes actually dropped/freed by counted runtime paths.
- `payload_transfer_current_delta_bytes`: signed adjustment for documented ownership transfers that
  preserve global totals.
- `payload_live_current_bytes`: measured current payload bytes.
- `free_count`: successful runtime free/drop count.
- `reuse_count`: successful runtime reuse count.
- `released_total_bytes`: bytes reported as released/decommitted only after successful OS/runtime
  release.
- `os_release_attempt_count`, `os_release_success_count`, `os_release_success_bytes`: release
  operation counters at the runtime memory ABI boundary.
- `metric_sources`: per-metric provenance. Claim-bearing numeric fields must be
  `runtime_measured` or `os_measured`, not `allocation_plan_estimate`, `planned`, or `estimated`.
- `unsupported_metrics`: metrics unavailable for the current target/configuration; unsupported
  metrics must be absent rather than encoded as numeric zero.
- `not_sampled_metrics`: metrics omitted because the run did not sample them; not-sampled metrics
  must be absent rather than encoded as numeric zero.

`started_unix_nano` and `finished_unix_nano` are optional because some minimal native runtimes may
not have a time source wired into the telemetry path.

## Invariants

- `heap_peak_bytes >= heap_current_bytes`.
- `heap_total_alloc_bytes >= heap_peak_bytes`.
- `heap_allocation_count == 0` is valid only when all heap byte totals are zero.
- if `bytes_reserved` is non-zero, it must be greater than or equal to `heap_peak_bytes`.
- every `allocation_paths` key must be non-empty and every count must be positive.
- every `domain_bytes` item must have a non-empty `domain_id` and `kind`.
- per-domain `peak_bytes` must be greater than or equal to per-domain `current_bytes`.
- when present, `actor_snapshot_record_count >= actor_live_count`; reusable/done actor slots must
  not be counted as live actors merely because they still have snapshot records for retained or
  released stack accounting.
- v2 `payload_live_current_bytes` must reconcile with successful alloc/drop counters and documented
  transfer deltas.
- v2 `released_total_bytes` must not exceed successful OS/runtime release bytes and must not be
  positive when `os_release_success_count` is zero or absent.
- v2 `free_count` must not exceed counted successful allocations.
- v2 `per_core` claims are invalid for process-global allocator state.
- v2 claim-bearing measured fields must not use allocation-plan or compiler-estimate provenance.
- v2 unsupported or not-sampled metrics must be omitted/null rather than encoded as zero.

## Evidence Mapping

In local Tier 1 benchmark memory evidence:

- `heap_alloc_bytes.evidence_class == runtime_measured` means the value came from a
  `tetra.runtime.heap_telemetry.v1` sidecar for the benchmarked Tetra binary.
- `heap_alloc_bytes.method` must be `tetra_linux_x64_heap_telemetry_v1`.
- `heap_alloc_bytes.source_artifact` must point to the raw or row-summary heap telemetry artifact
  inside the report directory.
- If the legacy `bytes` field is present for `heap_alloc_bytes`, it means `heap_peak_bytes`.
- `current_bytes`, `peak_bytes`, `total_alloc_bytes`, and `allocation_count` should be present when
  the report schema supports them.
- When a benchmark has multiple iterations, the row-level `heap_alloc_bytes` metric points at the
  collected sidecar with the maximum observed `heap_peak_bytes`.
- A successful row may report zero heap bytes when the telemetry sidecar proves that the optimized
  binary performed no counted Tetra heap allocation at runtime. Zero is not a replacement for a
  missing sidecar.

## Explicit Nonclaims

The following are not valid evidence for `heap_alloc_bytes.runtime_measured`:

- Go `runtime.MemStats` from the benchmark runner or compiler process.
- process RSS from `/proc`, `getrusage`, or another process sampler.
- allocation-plan reports or compiler estimates.
- allocation-plan reports or compiler estimates in v2 `metric_sources`.
- binary size.
- `mmap` reserved bytes alone.
- C, C++, Rust, wasm, linux-x86, linux-x32, macOS, or Windows observations.

RSS is reported separately by `docs/spec/telemetry/process_rss_telemetry.md` when a process-level
sampler is enabled, but RSS is not Tetra heap telemetry.

## Failure Semantics

If a Tetra row builds or runs successfully on linux-x64 but lacks a valid heap sidecar, the row must
not claim runtime-measured heap evidence.

Valid alternatives are:

- `blocked` with a concrete build/run/telemetry reason;
- `unsupported` only for targets or configurations that the telemetry contract explicitly does not
  support.

For successful linux-x64 Tier 1 Tetra rows, `unsupported` heap evidence is a failure once this
feature is enabled.
