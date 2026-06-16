# Benchmark vNext + Memory vNext Baseline Plan

**Goal:** Produce fresh local Tier 1 benchmark evidence on the current HEAD and
extend Tetra benchmark rows with Memory vNext byte/RSS/domain evidence before
starting targeted optimization work.

**Context:** `reports/local-benchmark-tier1-v1/report.json` is valid Tier 1
local evidence, but it was generated on 2026-06-03 for commit
`5129f2623d9639990076a7d422e56f02b0ed3254`. The current Memory vNext work adds
memory evidence vocabulary and local validators, so the next benchmark pass
must establish a fresh baseline instead of optimizing from stale data.

**Execution:** Use `executing-plans` task-by-task. Do not start optimizer or
runtime changes until the fresh baseline and memory-augmented report schema are
validated.

## Post-Implementation Note

The later RSS track
`docs/plans/2026-06-13-benchmark-vnext-rss-sampling.md` implemented the
lightweight linux process RSS sampler for Tetra Tier 1 rows. The current
evidence path is
`reports/benchmark-vnext-memory-baseline/tier1-rss-current-head/report.json`.
Historical references below to RSS remaining `unsupported` describe the first
memory-aware pass, not the current RSS-enabled baseline.

## Current Ground Truth

- `tools/cmd/local-benchmark-tier1` generates
  `tetra.local_benchmark_tier1.v1` reports for 17 P20 categories and four
  languages: Tetra, C, C++, and Rust.
- `tools/cmd/validate-local-benchmark-tier1` validates the Tier 1 report
  schema, categories, rows, raw artifacts, Tetra metadata, classifications, and
  local-only nonclaims.
- `docs/plans/2026-06-03-p25.0-real-local-benchmark-execution-v1.md` is the
  existing Tier 1 local benchmark execution plan.
- `docs/benchmarks/truth_benchmark_harness.md` defines claim tiers and forbids
  fastest-language, official benchmark, cross-machine, TechEmpower, and broad
  C/C++/Rust parity claims without matching evidence.
- `docs/spec/memory_backend_vnext.md` and
  `docs/spec/memory_domains_vnext.md` define the Memory vNext evidence terms
  needed by this plan.
- `tools/cmd/memory-production-smoke` and
  `tools/cmd/validate-memory-production` already distinguish measured heap
  evidence from RSS unsupported/blocked evidence for memory production reports.
- `benchmarks/techempower/tetra` contains a local
  TechEmpower-compatible/SCRAM harness, but it is not an official TechEmpower
  submission or claim.

## Non-Goals

- Do not claim Tetra is globally faster than C, C++, Rust, Go, Java, or any
  other language.
- Do not claim official TechEmpower results.
- Do not claim cross-machine reproducibility from a single local machine.
- Do not turn allocation-report estimates into measured RSS.
- Do not add GitHub Actions wiring unless explicitly approved later.
- Do not optimize fallback backend, bounds checks, hash table heap allocation,
  or actor/runtime blockers before the fresh memory-aware baseline exists.

## Task 1 - Fresh Tier 1 Baseline On Current HEAD

**Goal:** Regenerate local Tier 1 benchmark evidence on the current HEAD and
record whether the existing P25.0 runner still works after Memory vNext.

**Files:**
- Inspect `tools/cmd/local-benchmark-tier1`.
- Inspect `tools/cmd/validate-local-benchmark-tier1`.
- Inspect `reports/local-benchmark-tier1-v1/report.json`.
- Write the fresh output to a new report directory, for example
  `reports/benchmark-vnext-memory-baseline/tier1-current-head/`.
- Do not overwrite the older `reports/local-benchmark-tier1-v1/` baseline until
  the new report validates and the replacement policy is explicit.

**Approach:**
- Run the current Tier 1 runner with a persistent Go cache and a fresh
  `--out-dir`.
- Capture the current git commit, host CPU, compiler versions, raw artifacts,
  binary sizes, compile times, runtime medians, and current classifications.
- Validate the generated `report.json`.
- Compare the fresh classification distribution against the old valid report:
  fallback backend, bounds checks, heap allocation, actor/runtime limitation,
  invalid/inconclusive, comparable, and locally faster categories.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-current-head --iterations 3
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-current-head/report.json
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext go clean -cache
```

**Done when:** A fresh report validates on the current HEAD, and a short
summary names the current measured row count, classification counts, and any
build/run failures.

**Notes:** If compilers or system tools are missing, record the exact blocker in
the report summary instead of downgrading the evidence silently.

## Task 2 - Benchmark Memory Evidence Schema

**Goal:** Extend local Tier 1 Tetra benchmark rows with Memory vNext fields
without changing benchmark semantics or non-Tetra language rows.

**Files:**
- Modify `tools/cmd/local-benchmark-tier1/types.go`.
- Modify `tools/cmd/local-benchmark-tier1/report.go` if summary output needs
  the new fields.
- Modify `tools/cmd/local-benchmark-tier1/metadata.go` or nearby metadata
  helpers after confirming where Tetra report artifacts are collected.
- Modify `tools/cmd/validate-local-benchmark-tier1/main.go`.
- Add or extend focused tests under `tools/cmd/local-benchmark-tier1` and
  `tools/cmd/validate-local-benchmark-tier1`.

**Approach:**
- Add an optional Tetra-only memory evidence object to `benchmarkRow` or
  `tetraMetadata`.
- Include separate fields for:
  - `heap_alloc_bytes`;
  - `bytes_requested`;
  - `bytes_reserved`;
  - `bytes_committed`, if a current artifact provides it; otherwise record an
    explicit unsupported/blocked sample;
  - `bytes_copied`;
  - `rss_current`;
  - `rss_peak`;
  - `domain_bytes`, grouped by domain id/kind when available.
- Every metric must carry an evidence class such as `runtime_measured`,
  `allocation_report_estimate`, `unsupported`, or `blocked`.
- Reuse Memory vNext vocabulary from `docs/spec/memory_backend_vnext.md` and
  `docs/spec/memory_domains_vnext.md`.
- Validators must reject reports that claim RSS is measured from allocation
  estimates or from `MemStats` alone.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext go test ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -run 'Memory|RSS|Domain|Benchmark|Report|Validate' -count=1
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext go clean -cache
```

**Done when:** The validator accepts a valid memory-aware Tier 1 fixture and
rejects missing, stale, fake, or overclaiming memory evidence.

**Notes:** This task defines evidence shape only. It must not claim hard RSS
thresholds or benchmark superiority.

## Task 3 - Attach Memory Evidence To Fresh Benchmark Rows

**Goal:** Populate the new memory evidence fields for every Tetra row in the
fresh Tier 1 report.

**Files:**
- Inspect Tetra artifacts emitted by `tetra build --target linux-x64
  --explain`.
- Inspect allocation, bounds, backend, and perf artifacts referenced by
  `tetraMetadata`.
- Inspect `tools/cmd/memory-production-smoke/report.go` for RAM/RSS evidence
  vocabulary, but do not force the full memory-production smoke into every
  microbenchmark unless the cost and meaning are acceptable.
- Modify `tools/cmd/local-benchmark-tier1` collection logic only after
  identifying the artifact source for each metric.

**Approach:**
- Prefer compiler-owned allocation reports for `bytes_requested`,
  `bytes_reserved`, `bytes_copied`, and domain grouping when available.
- Use runtime-measured heap/RSS samples only when the row runner can measure the
  actual benchmark process with an honest method.
- If a metric is not available for a benchmark row, record `unsupported` or
  `blocked` with a reason instead of omitting or estimating it silently.
- Keep non-Tetra rows unchanged except for summary comparison context.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-memory-current-head --iterations 3
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-memory-current-head/report.json
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext go clean -cache
```

**Done when:** Every Tetra row has memory evidence fields with explicit
evidence classes, and the report validates without weakening existing Tier 1
claim policy.

**Notes:** It is acceptable for RSS to remain unsupported/blocked for a first
memory-aware pass, but it must be visible and validated as such.

## Task 4 - Baseline Analysis And Blocker Ranking

**Goal:** Convert the fresh memory-aware report into an actionable optimization
queue.

**Files:**
- Add `docs/audits/benchmark-vnext-memory-baseline.md`.
- Read `reports/benchmark-vnext-memory-baseline/tier1-memory-current-head/report.json`.
- Read relevant Tetra `.bounds.json`, `.alloc.json`, `.perf.json`, and
  `.backend.json` artifacts referenced by the report.

**Approach:**
- Summarize benchmark categories by current classification.
- Rank blockers by expected leverage:
  1. fallback backend;
  2. bounds-check elimination;
  3. heap allocation in hash table;
  4. actor/runtime benchmark limitation.
- For each blocker, cite exact report rows and artifact paths.
- Separate runtime-measured memory from allocation-report estimates.
- Identify which rows are not meaningful full-service benchmarks, especially
  JSON/HTTP/PostgreSQL rows that remain invalid/inconclusive in Tier 1 helper
  form.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check docs/audits/benchmark-vnext-memory-baseline.md docs/plans/2026-06-13-benchmark-vnext-memory-baseline.md
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-docs go clean -cache
```

**Done when:** The audit names the next optimization target with evidence, not
guesswork, and preserves local-only benchmark nonclaims.

**Notes:** If the fresh baseline changes the blocker ordering, follow the new
evidence instead of the stale 2026-06-03 distribution.

## Task 5 - Fallback Backend Optimization Track

**Goal:** Open the first focused optimization track only after Task 4 identifies
the current fallback backend rows and artifact reasons.

**Files:**
- Inspect Tetra rows classified as `blocked by fallback backend`.
- Inspect the referenced `.backend.json`, `.perf.json`, and source artifacts.
- Inspect backend/codegen modules named by the artifacts before editing.

**Approach:**
- Pick one narrow fallback cause that appears in multiple benchmark rows.
- Write a separate design or implementation plan if the fix crosses optimizer,
  backend, and validation boundaries.
- Preserve correctness gates before any performance claim.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-backend go test ./compiler/internal/backend/... ./compiler/internal/validation/... -run 'Backend|Fallback|Register|Stack|Translation|Differential' -count=1
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-backend go clean -cache
```

**Done when:** There is a scoped follow-up plan or patch for the highest-value
fallback blocker, with fresh benchmark re-run requirements defined.

**Notes:** Do not optimize by weakening safety reports, bounds checks, or
translation validation.

## Task 6 - Bounds-Check Elimination Track

**Goal:** Open a focused bounds-check elimination track after the fresh baseline
identifies the exact remaining bounds checks.

**Files:**
- Inspect rows classified as `blocked by bounds check`.
- Inspect the referenced `.bounds.json`, `.proof.json`, and source artifacts.
- Inspect proof/validation/optimizer files named by those artifacts before
  editing.

**Approach:**
- Group remaining checks by reason, for example missing dominance, missing range
  proof, or unsupported loop shape.
- Pick one proof-preserving elimination case.
- Require translation/differential validation before promoting the result.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-bounds go test ./compiler/internal/opt ./compiler/internal/validation ./compiler/tests/safety -run 'Bounds|Proof|Range|Dominance|Translation|Differential' -count=1
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-bounds go clean -cache
```

**Done when:** There is a scoped follow-up plan or patch for one high-value
bounds blocker, with benchmark re-run criteria defined.

**Notes:** Do not remove checks unless proof and validation agree.

## Task 7 - Hash Table Heap Allocation Track

**Goal:** Investigate the hash table heap allocations and decide whether the
next move belongs in stdlib source, allocation planning, region/island support,
or benchmark workload design.

**Files:**
- Inspect the hash table Tetra benchmark source emitted by
  `tools/cmd/local-benchmark-tier1`.
- Inspect the row allocation report.
- Inspect `benchmarks/generic_collections/` and existing generic collection
  docs before designing changes.

**Approach:**
- Determine whether the heap allocation is inherent to the current source-level
  collection model or an avoidable lowering/reporting issue.
- Prefer region/island-backed or caller-owned storage evidence only if it
  matches current Tetra semantics.
- Keep C++/Rust parity and production stdlib claims out of scope.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-hash go test ./compiler/internal/allocplan ./compiler/internal/memoryfacts ./compiler/tests/semantics -run 'Hash|Generic|Allocation|Island|Region|Memory' -count=1
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-hash go clean -cache
```

**Done when:** The hash table blocker has a concrete owner and a next plan:
stdlib/source change, allocator/reporting change, or explicit nonclaim.

**Notes:** Do not hide heap allocations to make the benchmark look better.

## Task 8 - Actor Runtime Benchmark Limitation Track

**Goal:** Keep actor benchmarks honest while defining what would be required to
move from model/prep evidence toward local measured actor benchmark evidence.

**Files:**
- Inspect actor rows in the memory-aware Tier 1 report.
- Inspect `compiler/internal/parallelrt`, `compiler/internal/actorsrt`, and
  `compiler/internal/actorsafety`.
- Inspect `docs/spec/actors.md` and `docs/design/actor_region_transfer.md`.

**Approach:**
- Keep the current actor rows blocked if the runtime evidence is still bounded
  model/report evidence.
- Define the minimal local measured actor benchmark gate separately from
  distributed actor runtime or production scheduler promotion.
- Include `ActorMemoryDomain` byte limits, mailbox bytes, reclaimed bytes, and
  local domain ownership movement in the measurement design.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-actors go test ./compiler/internal/actorsrt ./compiler/internal/parallelrt ./compiler/internal/actorsafety -count=1
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-actors go clean -cache
```

**Done when:** Actor benchmark limitations remain explicit, and any promotion
path has its own acceptance criteria and nonclaims.

**Notes:** This plan must not promote the actor runtime to a full production
multi-threaded scheduler.

## Task 9 - TechEmpower- Compatible Track Kept Separate

**Goal:** Keep TechEmpower-compatible evidence as a separate local harness and
avoid mixing it into P25 Tier 1 microbenchmark claims.

**Files:**
- Inspect `benchmarks/techempower/tetra/README.md`.
- Inspect `benchmarks/techempower/tetra/run-scram-local-bench.sh`.
- Inspect `docs/benchmarks/techempower_scram_*`.
- Inspect `tools/cmd/validate-techempower-report` before changing any report
  semantics.

**Approach:**
- Treat SCRAM-backed TechEmpower-compatible runs as local service evidence.
- Validate existing checked-in reports before trusting them.
- If a fresh run is needed, write it to a fresh report directory and state
  Docker/PostgreSQL prerequisites.
- Do not merge official TechEmpower wording into Tier 1 local benchmark rows.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-te go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_local_report.json
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-te go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_single_query_matrix_local_report.json
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-te go run ./tools/cmd/validate-techempower-report --report docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-te go clean -cache
```

**Done when:** TechEmpower-compatible evidence is validated or separately
blocked, and no Tier 1 report claims official TechEmpower status.

**Notes:** A fresh live SCRAM run may require local PostgreSQL or Docker access;
record tool/daemon blockers honestly.

## Task 10 - Final Gates And Evidence Map

**Goal:** Close the Benchmark vNext baseline with reproducible local evidence
and a clear next optimization target.

**Files:**
- `reports/benchmark-vnext-memory-baseline/`
- `docs/audits/benchmark-vnext-memory-baseline.md`
- This plan.

**Approach:**
- Validate the final memory-aware Tier 1 report.
- Validate docs.
- Run focused tests for modified tools.
- Run `git diff --check`.
- Run `graphify update .` after code/doc changes.
- Write a final evidence map that links each task to report paths and commands.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-final go test ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -count=1
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-final go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-memory-current-head/report.json
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-final go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
graphify update .
GOCACHE=$(pwd)/.cache/go-build-benchmark-vnext-final go clean -cache
```

**Done when:** The fresh memory-aware local benchmark baseline validates, all
nonclaims are preserved, and the next optimization track is chosen from current
evidence rather than stale 2026-06-03 results.

## Acceptance Criteria

- Fresh Tier 1 report generated on current HEAD.
- Tetra rows expose memory evidence fields with evidence classes.
- RSS and heap/allocation estimates remain visibly separate.
- Domain bytes are recorded where available or explicitly unsupported/blocked.
- Top blockers are ranked from current report evidence.
- TechEmpower-compatible evidence remains a separate local service track.
- Validators reject fake, stale, missing, and overclaiming benchmark/memory
  evidence.
- No official benchmark, broad performance, cross-machine, or production claim
  is introduced.

## Open Decisions

- Whether to evolve `tetra.local_benchmark_tier1.v1` in place or introduce a
  `tetra.local_benchmark_tier1.memory_vnext.v1` schema.
- Per-row RSS is now measured by the lightweight linux process sampler in the
  Tier 1 runner for Tetra rows; cross-target and hard-threshold policy remain
  future decisions.
- Whether the older `reports/local-benchmark-tier1-v1/` artifact should remain
  historical evidence or be replaced after the new report validates.
- Whether hard memory thresholds belong in Tier 1 benchmark policy or only in a
  later release-gate policy after baseline evidence exists.
