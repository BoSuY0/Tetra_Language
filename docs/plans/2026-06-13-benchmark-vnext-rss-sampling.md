# Benchmark vNext Process RSS Sampling Implementation Plan

Status: implementation plan, not implemented yet.

Primary predecessor:
`docs/plans/2026-06-13-benchmark-vnext-runtime-heap-sampling.md`.

Current runtime heap baseline:
`reports/benchmark-vnext-memory-baseline/tier1-runtime-heap-current-head/report.json`.

Prompting contract applied:
`/home/tetra/Downloads/gpt55-prompting-final-files/gpt-5-5-prompting-guide-en-final.md`.

## Goal

Make local Tier 1 linux-x64 Tetra benchmark rows report real process RSS
evidence for the benchmarked process, backed by raw artifacts collected by the
local benchmark runner.

This plan is complete only when RSS evidence is separated from Tetra heap
evidence, allocation-report estimates, compiler memory, and Go runner memory.

## Outcome Contract

The final outcome is not "some RSS-looking number exists". The final outcome
is:

- successful linux-x64 Tetra Tier 1 benchmark rows include artifact-backed
  `rss_peak.runtime_measured`;
- successful linux-x64 Tetra Tier 1 benchmark rows include artifact-backed
  `rss_current.runtime_measured` when a live-process sample was actually
  observed;
- if a live current-RSS sample cannot be observed for a row, the row must use
  `blocked` for `rss_current` with a concrete reason rather than copying peak
  RSS into current RSS;
- RSS artifacts are written by the runner while executing the benchmarked
  process, not by the Tetra compiler, not by Go `runtime.MemStats`, and not by
  the Tetra heap sidecar;
- validators reject missing, stale, impossible, or substituted RSS evidence;
- benchmark timing behavior remains reviewable, including the RSS sampler
  overhead/nonclaim;
- docs and audits explain what RSS means and what it does not mean.

## Current Ground Truth

- `tools/cmd/local-benchmark-tier1/types.go` already has
  `memoryEvidence.RSSCurrent` and `memoryEvidence.RSSPeak`.
- `tools/cmd/local-benchmark-tier1/metadata.go` currently emits both RSS
  fields as `unsupported` with reason:
  `Tier 1 runner does not measure process RSS per benchmark row`.
- `tools/cmd/local-benchmark-tier1/command.go` currently runs processes via
  `exec.CommandContext(...).Run()`, which captures elapsed time and output but
  does not expose live `/proc/<pid>` sampling points.
- `tools/cmd/local-benchmark-tier1/command.go` already has per-iteration
  collection machinery for runtime heap sidecars.
- `tools/cmd/validate-local-benchmark-tier1/main.go` currently rejects some
  RSS overclaims, including allocation-report estimates and Go `MemStats`, but
  it does not require a raw RSS artifact for `runtime_measured` RSS.
- `docs/spec/runtime_heap_telemetry.md` explicitly states that heap telemetry
  is not RSS evidence.
- The fresh runtime heap report has 17 Tetra rows: 16 successful measured rows
  and 1 `region_island_allocation_tetra` build-failed/blocked row.

## Truth Boundaries

`heap_alloc_bytes` and RSS are different metrics:

- `heap_alloc_bytes` is Tetra runtime heap allocation telemetry from the Tetra
  binary/runtime path.
- `rss_peak` is peak resident set size for the benchmarked OS process.
- `rss_current` is the last observed live resident set size sample while that
  process still existed.

`rss_current` is not "RSS after the process exited". After exit, current RSS is
not a useful process metric. Do not fill `rss_current` with `rss_peak` unless a
future spec explicitly renames that field. If the sampler cannot observe a live
RSS sample, `rss_current` must be `blocked`.

## Non-Goals

- Do not optimize memory usage in this plan.
- Do not change Tetra heap telemetry semantics.
- Do not claim official TechEmpower results.
- Do not require RSS evidence for C, C++, or Rust rows in the first pass.
- Do not make cross-target RSS claims for macOS, Windows, wasm, C backend, C++,
  or Rust backend rows.
- Do not fix `region_island_allocation_tetra` build failure here; keep it
  `blocked` unless a separate implementation track fixes it.
- Do not wire GitHub Actions unless a later task explicitly approves it.

## Target Design

Add a linux process RSS telemetry layer to the local Tier 1 runner:

- a shared parser/validator package, likely
  `tools/internal/rsstelemetry`;
- a raw per-iteration RSS artifact under
  `artifacts/rss-telemetry/<benchmark>/iteration-XX.rss.json`;
- a row summary under
  `artifacts/rss-telemetry/<benchmark>/summary.json`;
- runner code that starts the benchmark process, samples `/proc/<pid>/status`
  or `/proc/<pid>/statm` while the process is alive, waits for the process, and
  records `ProcessState.SysUsage()` / `wait4` peak RSS when available;
- memory evidence mapping from the selected RSS artifact into
  `rss_current` and `rss_peak`;
- validator gates that require matching raw artifacts for
  `runtime_measured` RSS.

Recommended method names:

- `linux_procfs_wait4_rss_sampler_v1` for the combined raw sidecar;
- `linux_procfs_status_vmrss_v1` inside the sidecar for live current samples;
- `linux_wait4_rusage_maxrss_v1` inside the sidecar for peak RSS.

## RSS Sidecar Schema

Add a documented schema, for example
`tetra.local_benchmark.process_rss_telemetry.v1`, with at least:

- `schema`;
- `method`;
- `program`;
- `pid`;
- `target_os`;
- `target_arch`;
- `started_unix_nano`;
- `finished_unix_nano`;
- `exit_status`;
- `sample_interval_micros`;
- `sample_count`;
- `rss_current_bytes`;
- `rss_peak_bytes`;
- `rss_peak_source`;
- `ru_maxrss_raw`;
- `ru_maxrss_unit`;
- `samples`, possibly capped or summarized;
- `notes`.

Required invariants:

- `target_os == linux` for this implementation;
- `method == linux_procfs_wait4_rss_sampler_v1`;
- `rss_peak_bytes >= rss_current_bytes` when `sample_count > 0`;
- `rss_peak_bytes > 0` for successful measured linux rows unless the OS
  source is unavailable and the metric is blocked;
- `sample_count == 0` cannot support `rss_current.runtime_measured`;
- sidecar path must resolve inside the benchmark report artifact root.

## Evidence Mapping

For successful linux-x64 Tetra rows:

- `rss_peak`:
  - `evidence_class`: `runtime_measured`;
  - `method`: `linux_wait4_rusage_maxrss_v1`;
  - `bytes` and `peak_bytes`: selected sidecar `rss_peak_bytes`;
  - `source_artifact`: selected raw RSS artifact.
- `rss_current`:
  - `evidence_class`: `runtime_measured` only when `sample_count > 0`;
  - `method`: `linux_procfs_status_vmrss_v1`;
  - `bytes` and `current_bytes`: selected sidecar `rss_current_bytes`;
  - `source_artifact`: selected raw RSS artifact.

For build-failed or run-failed Tetra rows:

- both RSS metrics must be `blocked` with a concrete reason.

For non-linux hosts:

- RSS metrics remain `unsupported` or `blocked` with explicit OS reason.

## Definition Of Done

Status may be `DONE` only when all of these are true:

- RSS telemetry semantics are documented, including the `rss_current` truth
  boundary.
- The RSS sidecar parser validates schema, method, OS, path, and byte
  invariants.
- The validator rejects fake `rss_current` and `rss_peak` evidence:
  allocation-report substitutes, heap sidecar substitutes, Go `MemStats`,
  missing artifacts, stale artifacts, wrong program, impossible byte totals,
  and current-without-live-sample claims.
- The Tier 1 runner collects per-iteration RSS artifacts for successful
  linux-x64 Tetra rows.
- The runner emits row summaries and maps selected RSS evidence into
  `memory_evidence.rss_current` and `memory_evidence.rss_peak`.
- A fresh local Tier 1 report validates.
- Successful linux-x64 Tetra rows have `rss_peak.runtime_measured`.
- Successful linux-x64 Tetra rows have `rss_current.runtime_measured`, or any
  row without an observed live sample is explicitly `blocked` and final status
  is not overclaimed.
- Docs/audits/nonclaims are updated.
- Focused tests, fresh report validation, `verify-docs`, `git diff --check`,
  and `graphify update .` pass.
- Persistent Go caches used for evidence are cleaned.

If any item is missing, final status is `PARTIAL` or `BLOCKED`, not `DONE`.

## Task 0 - Scope And Baseline Lock

**Owner:** implementation agent.

**Dependency:** none.

**Goal:** Preserve the current RSS unsupported baseline before changing
behavior.

**Files:**
- Read `reports/benchmark-vnext-memory-baseline/tier1-runtime-heap-current-head/report.json`.
- Read `docs/audits/benchmark-vnext-memory-baseline.md`.
- Read `tools/cmd/local-benchmark-tier1/metadata.go`.
- Read `tools/cmd/validate-local-benchmark-tier1/main.go`.

**Approach:**
- Confirm the exact current RSS evidence class for each successful Tetra row.
- Confirm the build-failed/blocked Tetra row remains blocked.
- Record baseline counts in the workflow/evidence notes before edits.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-runtime-heap-current-head/report.json
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go clean -cache
```

**Done when:** The existing report validates and the old RSS unsupported state
is documented before implementation.

**Notes:** This prevents treating a schema rewrite as measurement progress.

## Task 1 - RSS Telemetry Spec

**Owner:** implementation agent.

**Dependency:** Task 0.

**Goal:** Define RSS artifact semantics before code changes.

**Files:**
- Add `docs/spec/process_rss_telemetry.md`.
- Update `docs/spec/runtime_heap_telemetry.md` only for cross-reference.
- Update `docs/audits/benchmark-vnext-memory-baseline.md` after fresh evidence
  exists.

**Approach:**
- Document `tetra.local_benchmark.process_rss_telemetry.v1`.
- Define `rss_peak` as process peak RSS from OS process accounting.
- Define `rss_current` as last observed live RSS sample, not post-exit RSS.
- State invalid evidence sources: Go `MemStats`, Tetra heap sidecars,
  allocation reports, binary size, compiler memory, runner heap, and copied
  `rss_peak` values.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-sampling-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-rss-sampling-docs go clean -cache
```

**Done when:** A reader can tell exactly what each RSS field means and what
cannot be used as evidence.

**Notes:** This task intentionally keeps heap and RSS as separate evidence
classes.

## Task 2 - RSS Parser And Validator Tests First

**Owner:** implementation agent.

**Dependency:** Task 1.

**Goal:** Make fake RSS evidence fail before runner sampling is trusted.

**Files:**
- Add `tools/internal/rsstelemetry/rsstelemetry.go`.
- Add `tools/internal/rsstelemetry/rsstelemetry_test.go`.
- Modify `tools/cmd/validate-local-benchmark-tier1/main_test.go`.
- Modify `tools/cmd/validate-local-benchmark-tier1/main.go`.

**Approach:**
- Implement `ReadFile(path, artifactRoot)` similar to heap telemetry.
- Validate schema, method, target OS, program, path containment, timestamps,
  `sample_count`, and byte invariants.
- Add validator tests that fail these overclaims:
  - `rss_peak.runtime_measured` without `source_artifact`;
  - missing RSS sidecar;
  - stale sidecar with wrong `program`;
  - RSS metric pointed at heap sidecar;
  - RSS metric pointed at allocation report;
  - `MemStats` method;
  - `rss_current.runtime_measured` with `sample_count == 0`;
  - `rss_peak_bytes < rss_current_bytes`.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go test ./tools/internal/rsstelemetry ./tools/cmd/validate-local-benchmark-tier1 -run 'RSS|Rss|Memory|Validate' -count=1
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go clean -cache
```

**Done when:** The validator can distinguish real RSS artifacts from every
known fake or substituted source.

**Notes:** Do not weaken existing heap validator gates while adding RSS gates.

## Task 3 - Runner RSS Sampler

**Owner:** implementation agent.

**Dependency:** Task 2.

**Goal:** Collect process RSS while running benchmark binaries.

**Files:**
- Modify `tools/cmd/local-benchmark-tier1/command.go`.
- Add linux-specific helper file if needed, for example
  `tools/cmd/local-benchmark-tier1/rss_linux.go`.
- Add non-linux fallback helper if build tags require it.
- Modify `tools/cmd/local-benchmark-tier1/main_test.go`.

**Approach:**
- Add a runner path that uses `cmd.Start()`, not `cmd.Run()`, so the parent can
  observe `cmd.Process.Pid`.
- While the child is alive, sample `/proc/<pid>/status` `VmRSS` or
  `/proc/<pid>/statm`.
- After `cmd.Wait()`, read `cmd.ProcessState.SysUsage()` and use Linux
  `ru_maxrss` as peak RSS evidence.
- Preserve timeout behavior, stdout/stderr capture, exit-code handling, and
  elapsed-time measurement.
- Write one raw RSS sidecar per iteration.
- Record sampler notes, including sample interval and whether current RSS was
  observed.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go test ./tools/cmd/local-benchmark-tier1 -run 'RSS|Rss|RunCommand|Iteration|Memory' -count=1
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go clean -cache
```

**Done when:** A test command can execute a real child process and produce a
valid RSS sidecar with peak RSS and, for a long-enough child, at least one live
current RSS sample.

**Notes:** If the sampler changes timing overhead materially, document that the
local Tier 1 report is still a local evidence report and not an official
benchmark claim.

## Task 4 - Tier 1 RSS Integration

**Owner:** implementation agent.

**Dependency:** Task 3.

**Goal:** Wire RSS sidecars into Tetra row memory evidence.

**Files:**
- Modify `tools/cmd/local-benchmark-tier1/types.go`.
- Modify `tools/cmd/local-benchmark-tier1/metadata.go`.
- Modify `tools/cmd/local-benchmark-tier1/specs.go`.
- Modify `tools/cmd/local-benchmark-tier1/report.go` if audit output needs RSS
  counts.
- Modify `tools/cmd/local-benchmark-tier1/main_test.go`.

**Approach:**
- Extend the existing runtime heap collection flow into a combined memory
  telemetry flow, or add a parallel RSS collection flow beside it.
- Store raw artifacts under
  `artifacts/rss-telemetry/<benchmark>/iteration-XX.rss.json`.
- Store a row summary under
  `artifacts/rss-telemetry/<benchmark>/summary.json`.
- Select row-level RSS evidence by highest `rss_peak_bytes`; if tied, prefer
  the sample with live current RSS and then the larger `sample_count`.
- Map selected evidence into `RSSPeak` and `RSSCurrent`.
- Keep build/run failures as `blocked`.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go test ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 ./tools/internal/rsstelemetry -run 'RSS|Rss|Memory|Report|Validate|Tier1' -count=1
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go clean -cache
```

**Done when:** Focused tests prove the report rows contain RSS evidence backed
by raw RSS artifacts and accepted by the validator.

**Notes:** Do not move RSS evidence into `heap_alloc_bytes`.

## Task 5 - Fresh Local Tier 1 RSS Report

**Owner:** implementation agent.

**Dependency:** Task 4.

**Goal:** Generate fresh evidence from the current HEAD.

**Files:**
- Write report artifacts under
  `reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/`.
- Update `docs/audits/local-benchmark-tier1-v1.md`.
- Update `docs/audits/benchmark-vnext-memory-baseline.md`.

**Approach:**
- Run Tier 1 locally with telemetry enabled.
- Validate the fresh report.
- Inspect representative RSS sidecars for:
  - at least one Tetra row with live `rss_current` sample;
  - peak RSS from `wait4`/rusage;
  - blocked build-failed row still blocked.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-rss-current-head --iterations 3 --timeout 20s
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/report.json
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go clean -cache
```

**Done when:** The fresh report validates and successful linux-x64 Tetra rows
have artifact-backed RSS evidence according to the Definition of Done.

**Notes:** If `rss_current` cannot be captured for short-lived rows, record that
as a blocker/nonclaim. Do not copy `rss_peak` into `rss_current`.

## Task 6 - Docs, Audits, And Nonclaims

**Owner:** implementation agent.

**Dependency:** Task 5.

**Goal:** Make the evidence contract public and reviewable.

**Files:**
- Update `docs/spec/process_rss_telemetry.md`.
- Update `docs/spec/runtime_heap_telemetry.md` cross-reference.
- Update `docs/audits/benchmark-vnext-memory-baseline.md`.
- Update `docs/audits/local-benchmark-tier1-v1.md`.
- Update `docs/plans/2026-06-13-benchmark-vnext-memory-baseline.md` only if it
  has stale RSS claims.

**Approach:**
- State measured counts and blocked counts.
- Explain that RSS is process memory, not Tetra heap memory.
- Explain sampler method and overhead boundary.
- Keep nonclaims for official TechEmpower, cross-target RSS, and
  region/island build failure.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-sampling-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-rss-sampling-docs go clean -cache
```

**Done when:** The docs and audits match the fresh report and do not overclaim.

**Notes:** Use exact report paths and counts from the fresh run, not expected
counts.

## Task 7 - Final Hygiene And Graph Update

**Owner:** implementation agent.

**Dependency:** Task 6.

**Goal:** Close the implementation cleanly.

**Files:**
- Workflow/evidence files if executing under a goal loop.
- `graphify-out/` after graph update.

**Approach:**
- Run final validation.
- Run diff hygiene.
- Update Graphify after code changes.
- Clean repo-local Go caches used by evidence commands.
- Inspect the final diff for unrelated changes and avoid reverting user work.

**Verification:**

```sh
git diff --check
graphify update .
GOCACHE=$(pwd)/.cache/go-build-rss-sampling go clean -cache
GOCACHE=$(pwd)/.cache/go-build-rss-sampling-docs go clean -cache
rm -rf .cache/go-build-rss-sampling .cache/go-build-rss-sampling-docs
```

**Done when:** Final checks pass, cache directories from this run are removed,
and the final report lists exact evidence and remaining risks.

**Notes:** `graphify update .` may skip `graph.html` if the graph is above the
HTML renderer node limit; that is acceptable if `graph.json` and
`GRAPH_REPORT.md` update successfully.

## Execution Recommendation

Use `executing-plans` for a single-agent checkpointed run, or
`subagent-driven-development` only if the work is split into independent
parser/validator, runner, and docs tracks with explicit integration review.

Recommended execution order:

1. Tasks 0-2: lock semantics and validator anti-fake gates.
2. Tasks 3-4: implement runner sampling and report integration.
3. Task 5: generate and validate fresh evidence.
4. Tasks 6-7: docs, audits, graph, cache cleanup, and final handoff.

## Stop Rules

- Stop with `BLOCKED` if Linux process RSS cannot be sampled on the host.
- Stop with `PARTIAL` if only `rss_peak` is reliable and `rss_current` cannot
  be observed for successful rows.
- Stop with `PARTIAL` if the report validates but docs/audits are stale.
- Stop with `PARTIAL` if only focused tests pass but no fresh Tier 1 report was
  generated.
- Never mark `DONE` from one passing validator, one passing test, or one row
  with RSS evidence.
