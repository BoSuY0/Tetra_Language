# Process RSS Telemetry

Status: linux implementation contract for local benchmark evidence.

## Purpose

Process RSS telemetry records resident set size for the benchmarked OS process while
`tools/cmd/local-benchmark-tier1` executes a local Tier 1 benchmark row. It is process memory
evidence, not Tetra heap evidence and not an official benchmark claim.

This spec defines the RSS sidecar artifact consumed by `tools/cmd/local-benchmark-tier1` and
`tools/cmd/validate-local-benchmark-tier1`.

## Schema

RSS sidecars use:

```text
tetra.local_benchmark.process_rss_telemetry.v1
```

The current sidecar method is:

```text
linux_procfs_wait4_rss_sampler_v1
```

Metric-level methods are:

```text
linux_procfs_status_vmrss_v1
linux_wait4_rusage_maxrss_v1
```

The first supported OS is:

```text
linux
```

## Required Fields

- `schema`: must be `tetra.local_benchmark.process_rss_telemetry.v1`.
- `method`: must be `linux_procfs_wait4_rss_sampler_v1`.
- `program`: benchmark/program identity.
- `pid`: process id of the benchmarked child process when available.
- `target_os`: must be `linux` for this implementation.
- `target_arch`: host architecture string for the executed process.
- `started_unix_nano`: wall-clock start timestamp captured by the runner.
- `finished_unix_nano`: wall-clock finish timestamp captured by the runner.
- `exit_status`: child process exit status.
- `sample_interval_micros`: live RSS sampler interval.
- `sample_count`: number of live RSS samples observed before process exit.
- `rss_current_bytes`: last live RSS sample in bytes.
- `rss_peak_bytes`: peak RSS in bytes from OS process accounting when available.
- `rss_peak_source`: source for `rss_peak_bytes`.
- `ru_maxrss_raw`: raw `ru_maxrss` value from Linux `wait4`/`getrusage` semantics when available.
- `ru_maxrss_unit`: unit used by `ru_maxrss_raw`; Linux uses `kilobytes`.
- `samples`: optional capped live RSS samples.
- `notes`: optional limitations or implementation notes.

## Invariants

- `target_os == linux`.
- `method == linux_procfs_wait4_rss_sampler_v1`.
- `program` must be non-empty.
- `pid` must be non-negative.
- `exit_status` must be non-negative for normally observed child exits.
- `finished_unix_nano >= started_unix_nano` when both timestamps are present.
- `rss_peak_bytes >= rss_current_bytes` when `sample_count > 0`.
- `sample_count == 0` cannot support `rss_current.runtime_measured`.
- `rss_peak_bytes > 0` is required for successful measured linux rows unless the RSS metric is
  `blocked`.
- every sample item must have a positive RSS byte value.
- sidecar paths used by benchmark reports must resolve inside the report artifact root.

## Evidence Mapping

In local Tier 1 benchmark memory evidence:

- `rss_peak.evidence_class == runtime_measured` means the value came from a valid
  `tetra.local_benchmark.process_rss_telemetry.v1` sidecar for the benchmarked child process.
- `rss_peak.method` must be `linux_wait4_rusage_maxrss_v1`.
- `rss_peak.source_artifact` must point to the raw RSS sidecar inside the report directory.
- if the legacy `bytes` field is present for `rss_peak`, it means `rss_peak_bytes`.
- `rss_current.evidence_class == runtime_measured` means the runner observed at least one live
  `/proc/<pid>/status` or equivalent RSS sample before the process exited.
- `rss_current.method` must be `linux_procfs_status_vmrss_v1`.
- if the legacy `bytes` field is present for `rss_current`, it means `rss_current_bytes`.
- when a benchmark has multiple iterations, the row-level RSS metric points at the collected sidecar
  with the maximum `rss_peak_bytes`; ties should prefer a sidecar with a live current sample, then
  the larger `sample_count`.

## RSS Current Boundary

`rss_current` is the last live resident set size sample observed while the benchmarked process still
existed. It is not post-exit RSS.

If a process exits before the sampler observes a live RSS sample, the row must not claim
`rss_current.runtime_measured`. It must use `blocked` with a concrete reason, even when `rss_peak`
is available.

Do not copy `rss_peak_bytes` into `rss_current_bytes`.

## Explicit Nonclaims

The following are not valid evidence for process RSS runtime measurements:

- Go `runtime.MemStats` from the benchmark runner or compiler process.
- Tetra heap sidecars such as `tetra.runtime.heap_telemetry.v1`.
- allocation-plan reports or compiler estimates.
- binary size.
- Tetra heap allocation counts.
- RSS from another process.
- C, C++, Rust, wasm, macOS, Windows, or non-linux targets for this first implementation.

RSS can explain the OS process footprint, but it does not prove Tetra heap allocation behavior.
Tetra heap evidence remains covered by `docs/spec/telemetry/runtime_heap_telemetry.md`.

## Failure Semantics

If a successful linux Tier 1 Tetra row lacks valid RSS sidecar evidence, it must not claim
runtime-measured RSS.

Valid alternatives are:

- `blocked` with a concrete sampler/build/run reason;
- `unsupported` only for targets or configurations outside this contract.

For successful linux-x64 Tier 1 Tetra rows, `rss_peak.unsupported` is a failure once this feature is
enabled. `rss_current.blocked` remains acceptable only when no live sample was observed and that
condition is recorded honestly.
