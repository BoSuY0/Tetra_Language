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

P0 phase-aligned sidecars use:

```text
tetra.local_benchmark.process_rss_telemetry.v2
```

The current sidecar method is:

```text
linux_procfs_wait4_rss_sampler_v1
```

The P0 v2 sidecar method is:

```text
linux_procfs_phase_rss_sampler_v2
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

## V2 Phase-Aligned Fields

`tetra.local_benchmark.process_rss_telemetry.v2` adds phase-aligned RSS evidence for steady-state
memory work:

- `workload_kind`: use `steady_state` for allocation/deallocation churn workloads.
- `mapping_count`: current process mapping count when sampled.
- each `samples[]` entry may include `phase`; Linux v2 phase samples must include
  `mapping_count`.
- Linux harnesses should keep raw `/proc/<pid>/smaps_rollup` snapshots beside the sidecar when
  that procfs file is available.

For `workload_kind == steady_state`, the sidecar must contain this ordered phase sequence:

```text
startup
post_warmup
steady_round_N
post_drain
pre_exit
```

`steady_round_N` means at least one numbered steady-state round such as `steady_round_1`.

## Invariants

- `target_os == linux`.
- v1 `method == linux_procfs_wait4_rss_sampler_v1`.
- v2 `method == linux_procfs_phase_rss_sampler_v2`.
- `program` must be non-empty.
- `pid` must be non-negative.
- `exit_status` must be non-negative for normally observed child exits.
- `finished_unix_nano >= started_unix_nano` when both timestamps are present.
- `rss_peak_bytes >= rss_current_bytes` when `sample_count > 0`.
- `sample_count == 0` cannot support `rss_current.runtime_measured`.
- `rss_peak_bytes > 0` is required for successful measured linux rows unless the RSS metric is
  `blocked`.
- every sample item must have a positive RSS byte value.
- Linux v2 sidecars must include top-level `mapping_count` and per-sample `mapping_count`.
- v2 steady-state workloads must include the required phase sequence.
- sidecar paths used by benchmark reports must resolve inside the report artifact root.

## P0 Baseline Harness

`tools/cmd/ram-p0-baseline` generates a `tetra.local_benchmark.process_rss_telemetry.v2`
sidecar under `reports/stabilization/tetra-ram-p0-baseline-<sha>/` with host fingerprint,
command manifest, raw phase samples, optional `smaps_rollup` snapshots, validator output, and
artifact hashes. It is a baseline harness only; it does not claim allocator reuse, release, target
parity, or performance improvement.

## Compiler Phase Profile

The compiler P7 diagnostic artifact uses a separate schema:

```text
tetra.compiler.phase-profile.v1
```

When `BuildOptions.EmitCompilerPhaseReport` is enabled for native executable builds, the compiler
writes `<output>.compiler-profile.json` by default, or `BuildOptions.CompilerPhaseReportPath` when
provided. The artifact samples the compiler process itself, not the emitted Tetra program.

The profile records local snapshots for these P7 phases:

```text
source_loading_parsing
semantic_analysis
plir_construction
allocation_planning
ir_lowering
module_codegen
object_retention_link
report_generation
final_cleanup
```

Each phase records Go heap allocation/sys bytes, Linux `/proc/self/status` RSS when available,
module/object/IR counts, retained source/semantic graph counts, worker count and reason, report
mode, target, and requested jobs. The `source_file_count` field records retained loaded source AST
files. The `checked_function_count` and `checked_type_count` fields record retained semantic graph
coverage from `CheckedProgram`. Native executable builds release the loaded source graph after
module codegen no longer needs `world.Root`, and release the checked semantic graph after UI/report
generation. Therefore `final_cleanup` must record `source_file_count`,
`checked_function_count`, and `checked_type_count` as `0`.

The `transient_ir_function_count` field records PLIR or lowering-summary function coverage while
those temporary IR objects are retained. The `allocation_plan_function_count` field records
allocation plan function coverage while the allocation plan is retained. Native module codegen
drops these temporary references before the `module_codegen` snapshot, so `module_codegen`,
`object_retention_link`, `report_generation`, and `final_cleanup` must record both transient
counts as `0`. When `MemoryBudgetBytes` is set, the native module build chooses a worker count no
larger than the budget-derived conservative per-worker capacity and records that reason in the
profile.
The `object_retention_link` phase records native object references while linking. After a
successful link, the native executable path drops module object and linked-object references before
UI/report generation; `report_generation` and `final_cleanup` therefore record `object_count: 0`
for this released-object boundary.
Before the retained-state `object_retention_link` snapshot, profiled native builds ask the Go
runtime to release already-dead compiler allocations while native object references are still
retained and counted. This keeps the snapshot focused on live object/link retention rather than
earlier temporary compiler pages.
Compiler JSON report emission streams through a file-backed JSON encoder into a temporary file in
the report directory, then publishes the finished report with an atomic rename. Failed encodes keep
the previous destination intact and remove temporary output instead of leaving partial report files.
When only `BuildOptions.EmitPLIR` is requested, report generation verifies and writes the PLIR
artifacts, then returns before constructing the allocation plan, full lowered IR, or bounds report
intermediates. `Explain`, proof, allocation, memory, and RAM-contract report modes still build the
intermediates they need for validation and report contents.
Compiler phase-profile JSON uses the same file-backed writer, so phase-profile emission does not
build one full JSON byte buffer before publication.
Profiled failed native builds also release retained compiler pages before the failure
`final_cleanup` snapshot and reset retained source/semantic/object/transient counts to zero in
that final row. The profile keeps a failure note so the sample is not mistaken for a successful
build.

This artifact is diagnostic evidence for compiler max-RSS work only; it does not by itself claim a
compiler RSS reduction, a benchmark improvement, or cross-host RSS comparability.

`tools/cmd/ram-p7-compiler-rss` generates a non-ignored diagnostic bundle under
`reports/stabilization/tetra-ram-p7-compiler-rss-<sha>/`. The bundle records synthetic and
selected real-source compiler workload scenarios, their `tetra.compiler.phase-profile.v1`
artifacts, executable/report outputs, host fingerprint, command manifest, validator output, and
`artifact-hashes.json`. The v2 bundle summary keeps every raw sample per scenario, reports
median/min/max/dispersion RSS fields, and can include expected compile-error scenarios so the P7.5
error path is covered explicitly. Repo-backed batch scenarios use `source_paths` plus
`source_count`, `compiler_profile_count`, and `executable_count` so multiple entry points compiled
inside one measured sample are not collapsed into a single opaque source.
Each measured sample runs after the harness asks the Go runtime to release retained pages, so
scenario ordering and earlier samples do not inflate the next sample's `source_loading_parsing`
snapshot. Report-enabled compiler builds also release transient report-generation allocations before
the `report_generation` phase snapshot, so that row represents retained compiler/report state after
report files are emitted rather than unreclaimed encoder/report temporaries.
The harness also performs one compiler-process warmup before measured scenarios. The warmup builds a
tiny valid program in a removed `.process-warmup-*` scratch directory under the bundle root and then
releases Go-retained pages. It warms the Go/compiler process only; it does not warm measured
scenario `.tetra_cache` directories or change cold/warm scenario cache semantics.

When matching report-off and report-on scenarios share module count, worker count, warm-cache mode,
and memory budget, the v2 summary also emits `report_comparisons`. Each comparison records both
scenario names, median RSS values, observed dispersions, delta, ratio, peak phases, and a
`bound_rss_bytes` value computed as:

```text
report_off_rss_median_bytes
+ report_off_rss_dispersion_bytes
+ report_on_rss_dispersion_bytes
```

`evaluation_status` is `pass`, `fail`, or `insufficient_samples`. This comparison is a same-host
diagnostic gate input for P7.7 report-on/report-off work; it is not by itself a final compiler RSS
improvement claim.
The default P7 harness includes comparable report-off/report-on pairs for small cold, medium cold,
and large warm-cache workloads so the default bundle exercises more than one report-bound point.

Passing `--matrix p7_5` selects the Linux x64-first P7.5 matrix. It covers small, medium, and
large synthetic module graphs; jobs `1`, `2`, `4`, and `runtime.NumCPU()` with duplicate worker
counts deduplicated; reports off/on; cold/warm compiler cache; and both successful builds and
expected compile-error paths. Warm expected compile-error scenarios first build a valid entry point
to warm unchanged dependency modules, then restore the failing entry point before the measured build,
so `warm_cache: true` remains truthful for error-path evidence. A one-sample `p7_5` bundle is only
smoke/proof-of-path evidence; the P7.7 median RSS and report-bound gates still require the source
plan's multi-sample protocol on the same host/configuration. The `p7_5` generator keeps matching
report-off/report-on scenarios adjacent for the same size/jobs/cache/outcome config so the
same-host report-bound comparison is not biased by unrelated in-process scenario drift.

The current Linux x64-first multi-sample P7.5 evidence bundle is:

```text
reports/stabilization/tetra-ram-p7-compiler-rss-b452638a8af7-p75-samples5/
```

It has 96 scenarios, five samples per scenario, 48 expected compile-error scenarios, and 24
report-off/report-on comparisons. All 24 report comparisons pass in that same-host bundle.

Passing `--matrix representative` selects the current real-source representative Linux x64-first
matrix. It builds
`examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra` as a report-off/on
pair with five samples when requested. Each sample copies the entry source into the bundle-local
`src/` project root and uses the repository root only as a dependency root, so measured compiler
cache state is sample-local and removed before the bundle is finalized. Scenario summaries and
report comparisons include `source_path` to keep real-source evidence distinct from synthetic
module-count scenarios.

The current representative evidence bundle is:

```text
reports/stabilization/tetra-ram-p7-compiler-rss-b452638a8af7-representative-samples5/
```

It has two scenarios, five samples per scenario, one passing report-off/report-on comparison, and
compiled module coverage for the Surface Morph flagship entry plus `lib.core.surface`,
`lib.core.block`, and `lib.core.morph`. This is representative-source evidence, not a full-repo or
cross-target compiler RSS claim.

Passing `--matrix full_repo` selects the current Linux x64 release smoke-profile batch workload.
Each measured sample compiles the current 71 linux-x64 smoke entry points in the same compiler
process, with one `tetra.compiler.phase-profile.v1` and one executable/report output set per entry
point. Scenario summaries and report comparisons include `source_paths` so the exact entry set is
machine-readable. This matrix is a full-repository release smoke-profile workload, not a claim that
all `examples/` entrypoint candidates, libraries, runtime sources, tests, or excluded negative/
target-specific examples were successfully compiled.

The current full-repo smoke-profile evidence bundle is:

```text
reports/stabilization/tetra-ram-p7-compiler-rss-b452638a8af7-full-repo-smoke-samples2/
```

It has two scenarios, two samples per scenario, 71 source paths per scenario, 284 compiler phase
profiles, validated `artifact-hashes.json`, `artifact-hashes-validation.txt`, and one passing
report-off/report-on comparison: report-off median `72699904`, report-on median `70897664`, bound
`75960320`, delta `-1802240`, and ratio `0.9752`. This closes the Linux x64 full-repo
smoke-profile compiler RSS evidence slice only; target-specific parity and all non-smoke
repository entrypoint candidates remain outside this claim.

The bundle also includes `target-scope.json` with schema:

```text
tetra.ram.p7-compiler-rss-target-scope.v1
```

`compiler-rss-manifest.json` points to this artifact through `target_scope`. The current artifact
records `host_target: linux/amd64` and `compiler_target: linux-x64`; `linux-x64` is the only
target with `host_rss_measured` status in this Linux procfs process-RSS harness. Windows x64,
macOS x64/arm64, Linux x86/x32, wasm32-wasi, and wasm32-web are explicit `non_claim` targets in
this evidence bundle because their reserve/commit/release, RSS, or linear-memory semantics require
target-specific measurement. The artifact prevents Linux-only compiler RSS evidence from being used
as a cross-target memory lifecycle claim.
The bundle validator checks both `scenario-summary.json` and `target-scope.json`: it rejects
missing required non-claim targets, empty non-claim reasons, any host RSS claim for a target other
than `linux-x64`, and any `host_rss_measured` row when the host is not `linux/amd64`.

The same tool can also compare a baseline bundle with a candidate bundle:

```text
go run ./tools/cmd/ram-p7-compiler-rss \
  --compare-baseline-dir <baseline-bundle> \
  --compare-candidate-dir <candidate-bundle> \
  --compare-out <candidate-bundle>/baseline-candidate-comparison.json
```

The comparison artifact schema is:

```text
tetra.ram.p7-compiler-rss-baseline-comparison.v1
```

It records the reproducible comparison command, bundle paths, git heads from each bundle manifest,
same-host/config status, host mismatch fields when present, candidate-only scenarios, and one
scenario comparison per baseline scenario. Scenarios match by name and must keep compatible module
count, jobs, report mode, warm-cache setting, memory budget, and expected compile-error flag.
When both sides include per-sample `tetra.compiler.phase-profile.v1` paths, each matched scenario
also records `phase_comparisons`: intersected compiler phase names sorted by descending median RSS
delta, with baseline/candidate sample counts, median RSS values, delta bytes, and ratio. These
phase rows are diagnostic evidence for locating RSS growth; the scenario-level median gate remains
the acceptance input.

For each matched scenario, `bound_rss_bytes` is computed as:

```text
baseline_rss_median_bytes
+ baseline_rss_dispersion_bytes
+ candidate_rss_dispersion_bytes
```

Scenario `evaluation_status` is:

- `improved` when the candidate median is below the baseline median;
- `flat` when the candidate median is at or above the baseline median but not above the bound;
- `regressed` when the candidate median is above the bound;
- `insufficient_samples` when either side has fewer than the required valid samples;
- `config_mismatch`, `missing_candidate`, or `incompatible_host` when the comparison is not a
  same-host/same-config acceptance input.

`overall_status` is `pass`, `fail`, `insufficient_samples`, or `incompatible`. A failing comparison
is still useful evidence because it identifies the scenarios that prevent the P7 median-RSS
acceptance gate from being claimed.

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
