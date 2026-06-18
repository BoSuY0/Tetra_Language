# Benchmark vNext Runtime Heap Sampling Finalization Plan

Status: implementation plan, not implemented yet.

Primary baseline: `docs/audits/memory/zero-heap-final/benchmark-vnext-memory-baseline.md`.

Prompting contract applied:
`/home/tetra/Downloads/gpt55-prompting-final-files/gpt-5-5-prompting-guide-en-final.md`.

## Goal

Make local Tier 1 Tetra benchmark rows report real runtime heap evidence for the benchmark process
itself, so `heap_alloc_bytes` is no longer an unsupported placeholder for successful linux-x64 Tetra
rows.

This plan is complete only when the runner, compiler/runtime instrumentation, validator, reports,
docs, and fresh evidence agree on the same truth model.

## Outcome Contract

The final outcome is not "some memory number exists". The final outcome is:

- successful linux-x64 Tetra Tier 1 benchmark rows include a runtime-measured Tetra heap sample;
- every runtime heap sample is backed by a raw sidecar artifact written by the benchmarked Tetra
  binary or by a runtime hook inside that binary;
- validators reject fake heap evidence, missing sidecars, stale sidecars, RSS substitutes, Go runner
  `MemStats`, and allocation-report-only estimates;
- the report still separates Tetra heap bytes, allocation-plan estimates, and process RSS;
- build-failed or run-failed rows are explicitly `blocked`, not silently counted as measured;
- docs and audits explain the remaining nonclaims.

## Current Ground Truth

- `tools/internal/localbenchmarktier1/command.go` runs benchmark binaries with
  `exec.CommandContext`, captures stdout/stderr, and records elapsed time. It does not sample
  target-process heap bytes.
- `tools/internal/localbenchmarktier1/specs/specs.go` builds Tetra rows and then calls
  `collectTetraMetadata`.
- `tools/internal/localbenchmarktier1/metadata.go` currently sets `heap_alloc_bytes` to
  `unsupported` with the reason that the Tier 1 runner does not measure runtime heap bytes per
  benchmark process.
- The same metadata path reads allocation reports and can populate `bytes_requested`,
  `bytes_reserved`, `bytes_copied`, and `domain_bytes` as `allocation_report_estimate`.
- `tools/internal/localbenchmarktier1/types.go` defines `tetra.local_benchmark.memory_evidence.v1`,
  but the current metric object is too small to express current/peak/total heap semantics cleanly.
- `tools/cmd/validate-local-benchmark-tier1/main.go` validates memory evidence classes, but it does
  not yet require a raw heap sidecar for `heap_alloc_bytes.runtime_measured`.
- `tools/cmd/memory-production-smoke/report.go` uses Go `runtime.MemStats` for a smoke process. That
  is not valid evidence for a compiled Tetra benchmark binary.
- `compiler/internal/runtimeabi/allocation_contract.go`,
  `compiler/internal/runtimeabi/smallheap/small_heap.go`,
  `compiler/internal/backend/x64core/x64core_core.go`, and
  `compiler/internal/backend/x64core/x64core_core.go` are the likely runtime allocation/emit
  surfaces for native heap telemetry.
- `compiler/internal/backend/linux_x64/codegen.go` enables the current small heap path for
  linux-x64, so the first implementation target should be linux-x64.

## Truth Boundaries

`heap_alloc_bytes` means Tetra heap usage observed while the benchmarked Tetra binary executes. It
does not mean:

- Go runner heap usage;
- compiler heap usage;
- allocation-report estimates;
- process RSS;
- binary size;
- `mmap` bytes alone;
- fastest-language or official benchmark claims.

The report may keep allocation estimates beside the runtime sample, but the evidence class and
method must make the difference machine-checkable.

## Non-Goals

- Do not implement RSS sampling in this plan. RSS needs a separate process sampler and threshold
  policy.
- Do not optimize allocations, fallback backend, bounds checks, hash table escape behavior, or
  actor/runtime benchmarks here.
- Do not make cross-target claims. The first target is native linux-x64.
- Do not require C, C++, or Rust rows to expose heap samples.
- Do not claim zero heap usage unless a runtime sidecar proves zero for that exact benchmark row.
- Do not wire GitHub Actions unless a later task explicitly approves it.

## Definition Of Done

Status may be `DONE` only when all of these are true:

- `tetra build --target linux-x64` can opt into runtime heap telemetry through explicit build
  options.
- A telemetry-enabled Tetra binary writes a raw JSON sidecar on successful benchmark execution.
- The sidecar schema is documented and validated.
- The local Tier 1 runner collects per-iteration sidecars and emits a stable row-level heap summary.
- `heap_alloc_bytes` for every successful linux-x64 Tetra row in the fresh Tier 1 report is
  `runtime_measured`.
- Build-failed or run-failed Tetra rows use `blocked` with a concrete reason.
- The validator rejects unsupported, fake, stale, missing, or mismatched heap evidence for rows that
  claim runtime measurement.
- Docs and audits state the exact nonclaims.
- Focused tests, report validation, docs validation, `git diff --check`, and `graphify update .`
  pass.

If any item is missing, final status is `PARTIAL` or `BLOCKED`, not `DONE`.

## Target Design

Introduce a native Tetra heap telemetry path for linux-x64:

- build flag: `--emit-runtime-heap-telemetry`;
- build flag: `--runtime-heap-telemetry-dir <dir>`;
- compiler option fields that carry those flags to the linux-x64 backend;
- backend/runtime counters for Tetra heap allocation paths;
- a normal-exit hook that writes one JSON sidecar per process;
- Tier 1 runner collection that renames/copies raw sidecars into the benchmark report artifact tree;
- row-level summary generation from raw per-iteration sidecars;
- strict validation of both summary and raw sidecars.

The sidecar directory approach avoids per-iteration recompiles. The runner can clean the telemetry
directory before each iteration, run the binary, detect the new sidecar, and store it as a
deterministic artifact path.

## Sidecar Schema

Add a documented schema, for example `tetra.runtime.heap_telemetry.v1`, with at least:

- `schema`;
- `target`;
- `method`: `tetra_linux_x64_heap_telemetry_v1`;
- `program`;
- `pid`;
- `started_unix_nano` if available;
- `finished_unix_nano` if available;
- `exit_status`;
- `heap_current_bytes`;
- `heap_peak_bytes`;
- `heap_total_alloc_bytes`;
- `heap_allocation_count`;
- `bytes_requested`;
- `bytes_reserved`;
- `allocation_paths`;
- `domain_bytes`;
- `notes`.

Required invariants:

- `heap_peak_bytes >= heap_current_bytes`;
- `heap_total_alloc_bytes >= heap_peak_bytes`;
- `heap_allocation_count == 0` is allowed only when all heap byte totals are zero;
- `target == linux-x64` for this implementation;
- `method == tetra_linux_x64_heap_telemetry_v1`;
- sidecar path must be inside the benchmark report directory after collection.

## Evidence Mapping

`heap_alloc_bytes` in the Tier 1 memory evidence should move to a v2-compatible shape that can
distinguish:

- `current_bytes`;
- `peak_bytes`;
- `total_alloc_bytes`;
- `allocation_count`;
- `evidence_class`;
- `method`;
- `source_artifact`;
- `unsupported_reason`;
- `blocked_reason`.

Compatibility rule:

- if the existing `bytes` field remains, define it as `peak_bytes` for `heap_alloc_bytes` only, and
  document this explicitly.

## Task 0 - Evidence Lock Before Edits

**Goal:** Preserve the current unsupported baseline before changing behavior.

**Files:**

- Read `docs/audits/memory/zero-heap-final/benchmark-vnext-memory-baseline.md`.
- Read the current fresh report under
  `reports/benchmark-vnext-memory-baseline/tier1-memory-current-head/report.json`.
- Read `tools/internal/localbenchmarktier1/metadata.go`.
- Read `tools/cmd/validate-local-benchmark-tier1/main.go`.

**Approach:**

- Confirm which rows currently have `heap_alloc_bytes.unsupported`.
- Confirm which row is still build-failed or blocked.
- Record the exact old reason string in the implementation notes or audit update.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-memory-current-head/report.json
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go clean -cache
```

**Done when:** The implementation starts from a validated baseline and no old unsupported state is
lost or rewritten without replacement evidence.

**Notes:** This task is a guard against accidentally claiming progress from a schema rename.

## Task 1 - Runtime Heap Telemetry Spec

**Goal:** Define the heap sidecar schema and report semantics before code.

**Files:**

- Add `docs/spec/telemetry/runtime_heap_telemetry.md`.
- Update `docs/spec/memory/memory_backend_vnext.md` only if terminology needs a cross-reference.
- Update `docs/spec/memory/memory_domains_vnext.md` only if domain byte mapping needs a
  cross-reference.

**Approach:**

- Document `tetra.runtime.heap_telemetry.v1`.
- Document `tetra.local_benchmark.memory_evidence.v2` or a backward-compatible v1 extension.
- State that Go `MemStats`, allocation reports, and RSS are not acceptable methods for
  `heap_alloc_bytes.runtime_measured`.
- Define how build/run failures become `blocked`.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-heap-sampling-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-heap-sampling-docs go clean -cache
```

**Done when:** A reader can tell exactly what a heap byte means and what it does not mean.

**Notes:** This spec is part of the product surface, not just internal tooling.

## Task 2 - Validator Tests First

**Goal:** Make fake heap evidence fail before instrumentation exists.

**Files:**

- Modify `tools/cmd/validate-local-benchmark-tier1/main_test.go`.
- Modify `tools/cmd/validate-local-benchmark-tier1/main.go`.
- Add shared fixtures under the existing test fixture style, if present.

**Approach:**

- Add failing tests for:
  - runtime-measured heap evidence without `source_artifact`;
  - runtime-measured heap evidence whose sidecar path does not exist;
  - stale sidecar where row/category/binary identity does not match;
  - method `MemStats`;
  - method `allocation_report_summary`;
  - method `linux_proc_status`;
  - sidecar with invalid byte invariants;
  - successful linux-x64 Tetra row that still says heap unsupported.
- Keep build-failed rows allowed to report `blocked`.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go test ./tools/cmd/validate-local-benchmark-tier1 -run 'Heap|Memory|Runtime|Sidecar|Validate' -count=1
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go clean -cache
```

**Done when:** Tests prove the validator will not accept the exact false completion modes this plan
forbids.

**Notes:** This is the main anti-lie gate.

## Task 3 - Shared Heap Telemetry Parser

**Goal:** Avoid ad hoc JSON parsing in runner and validator.

**Files:**

- Add a small shared package, likely `tools/internal/heaptelemetry`.
- Add tests under that package.
- Wire the package into `tools/cmd/validate-local-benchmark-tier1`.

**Approach:**

- Define Go structs for the sidecar schema.
- Parse, normalize, and validate invariants in one place.
- Expose clear errors for missing fields, bad method, bad target, negative bytes, impossible totals,
  and artifact-path problems.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go test ./tools/internal/heaptelemetry ./tools/cmd/validate-local-benchmark-tier1 -count=1
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go clean -cache
```

**Done when:** The validator and later runner can rely on one schema parser.

**Notes:** Keep this package limited to artifact parsing and validation. Do not put benchmark policy
there unless the policy is schema-level.

## Task 4 - CLI And Compiler Option Plumbing

**Goal:** Add an explicit opt-in path from `tetra build` to native backend telemetry.

**Files:**

- Modify `cli/cmd/tetra/tetra_core.go`.
- Modify `compiler/compiler_facade.go`.
- Modify any build option helper tests under `cli/cmd/tetra`.
- Modify linux-x64 backend option structs, likely under `compiler/internal/backend/x64` and
  `compiler/internal/backend/linux_x64`.

**Approach:**

- Add `--emit-runtime-heap-telemetry`.
- Add `--runtime-heap-telemetry-dir <dir>`.
- Reject a telemetry dir without telemetry enabled.
- Reject telemetry for unsupported targets with a clear diagnostic.
- Pass options through `compiler.BuildOptions` to the linux-x64 backend.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go test ./cli/cmd/tetra ./compiler ./compiler/internal/backend/linux_x64 ./compiler/internal/backend/x64 -run 'Heap|Telemetry|Build|Option|Target' -count=1
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go clean -cache
```

**Done when:** The build command accepts valid linux-x64 telemetry options and rejects unsupported
or inconsistent option combinations.

**Notes:** The default build must stay unchanged.

## Task 5 - Native linux-x64 Heap Counters

**Goal:** Count heap allocation behavior inside the generated Tetra binary.

**Files:**

- Inspect and modify `compiler/internal/backend/x64core/x64core_core.go`.
- Inspect and modify `compiler/internal/backend/x64core/x64core_core.go`.
- Inspect linux-x64 ABI/syscall helpers before adding new emit code.
- Add focused backend/codegen tests.

**Approach:**

- Add telemetry storage only when the build flag is enabled.
- Increment counters on Tetra heap allocation paths:
  - current live heap bytes when supported by the allocator path;
  - peak live heap bytes;
  - total allocated heap bytes;
  - allocation count;
  - bytes requested;
  - bytes reserved;
  - allocation path buckets.
- For paths that allocate but do not yet support live decrement, report that limitation in `notes`
  and keep `heap_peak_bytes` conservative.
- Do not count stack allocations as heap.
- Do not count Go compiler/runner allocation.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go test ./compiler/internal/backend/x64core ./compiler/internal/backend/linux_x64 ./compiler -run 'Heap|Telemetry|Alloc|SmallHeap|MakeSlice' -count=1
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go clean -cache
```

**Done when:** Generated telemetry-enabled binaries contain deterministic heap counter paths for the
allocation mechanisms used by Tier 1 rows.

**Notes:** If an allocation path cannot be counted honestly, mark that row or field blocked rather
than inventing a number.

## Task 6 - Sidecar Write On Normal Exit

**Goal:** Persist heap counters from the Tetra binary as raw evidence.

**Files:**

- Inspect linux-x64 program entry/exit emission.
- Modify the relevant linux-x64 backend or ABI helper.
- Add executable smoke tests that build and run tiny Tetra programs.

**Approach:**

- Embed the telemetry directory path into the binary when telemetry is enabled.
- On normal exit, write one JSON file to that directory using linux syscalls.
- Include process identity or a unique suffix so repeated runs do not collide.
- Ensure no sidecar is written outside the configured directory.
- Treat failed sidecar writes as a runtime telemetry failure that the runner can detect.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go test ./compiler ./cli/cmd/tetra -run 'Heap|Telemetry|Sidecar|LinuxX64|Run' -count=1
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go clean -cache
```

Manual smoke inside the implementation branch:

```sh
mkdir -p reports/tmp/heap-telemetry-smoke
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go run ./cli/cmd/tetra build --target linux-x64 --emit-runtime-heap-telemetry --runtime-heap-telemetry-dir reports/tmp/heap-telemetry-smoke -o reports/tmp/heap-telemetry-smoke/smoke examples/smoke/basic/hello.tetra
reports/tmp/heap-telemetry-smoke/smoke
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go clean -cache
```

**Done when:** A real compiled Tetra linux-x64 binary writes a valid sidecar on successful
execution.

**Notes:** The final implementation should remove or keep any `reports/tmp` scratch according to
repo cleanliness rules before completion.

## Task 7 - Tier 1 Runner Collection

**Goal:** Collect per-iteration heap sidecars and emit row-level memory evidence.

**Files:**

- Modify `tools/internal/localbenchmarktier1/command.go`.
- Modify `tools/internal/localbenchmarktier1/specs/specs.go`.
- Modify `tools/internal/localbenchmarktier1/metadata.go`.
- Modify `tools/internal/localbenchmarktier1/types.go`.
- Modify `tools/internal/localbenchmarktier1/core_test.go`.

**Approach:**

- Build Tetra rows with telemetry enabled and a row-specific telemetry directory.
- Before each iteration, clean only that row/iteration telemetry directory.
- After each successful iteration, find exactly one new sidecar.
- Copy or rename it into deterministic artifact paths such as
  `artifacts/memory/<row>.heap.iter<N>.json`.
- Write a row summary artifact such as `artifacts/memory/<row>.heap.summary.json`.
- Populate `heap_alloc_bytes` from the summary as runtime-measured evidence.
- Preserve existing allocation-report estimates for requested/reserved/copied bytes and domain bytes
  unless superseded by better runtime fields.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go test ./tools/internal/heaptelemetry ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 -run 'Heap|Memory|Runtime|Sidecar|Tier1|Report' -count=1
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go clean -cache
```

**Done when:** Unit tests prove successful Tetra rows receive sidecar-backed heap evidence and
failed rows produce blocked evidence.

**Notes:** Keep raw stdout/stderr artifacts unchanged.

## Task 8 - Strict End-To-End Report Validation

**Goal:** Prove the full Tier 1 flow produces and validates real heap evidence.

**Files:**

- Read generated report artifacts.
- Update `docs/audits/memory/zero-heap-final/benchmark-vnext-memory-baseline.md`.
- Add a new audit or section for runtime heap sampling evidence.

**Approach:**

- Run a fresh local Tier 1 report into a new directory, for example:
  `reports/benchmark-vnext-memory-baseline/tier1-runtime-heap-current-head/`.
- Validate it with `validate-local-benchmark-tier1`.
- Inspect all Tetra rows:
  - successful rows must have `heap_alloc_bytes.runtime_measured`;
  - build/run failed rows must have `blocked`;
  - source artifacts must exist;
  - sidecar method must be `tetra_linux_x64_heap_telemetry_v1`.
- Compare old unsupported heap evidence against new measured evidence.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-runtime-heap-current-head --iterations 3
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-runtime-heap-current-head/report.json
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go clean -cache
```

**Done when:** The fresh report validates and contains no unsupported `heap_alloc_bytes` for
successful linux-x64 Tetra rows.

**Notes:** This is the first point where the task can claim END_TO_END for heap sampling.

## Task 9 - Documentation, Cleanliness, And Graph Refresh

**Goal:** Finish cleanly without stale docs, caches, or graph artifacts.

**Files:**

- Update `docs/audits/memory/zero-heap-final/benchmark-vnext-memory-baseline.md`.
- Update benchmark truth docs only if they mention memory evidence semantics.
- Update this plan with completion notes if the project convention expects it.
- Update generated graph artifacts through `graphify update .`.

**Approach:**

- Document exact evidence counts:
  - total categories;
  - total rows;
  - successful Tetra rows with runtime heap evidence;
  - blocked/build-failed Tetra rows;
  - unsupported RSS rows, if RSS remains out of scope.
- Remove temporary report directories not intended as evidence.
- Clean the persistent Go cache used for verification.
- Run whitespace/diff checks.

**Verification:**

```sh
GOCACHE=$(pwd)/.cache/go-build-heap-sampling-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
graphify update .
GOCACHE=$(pwd)/.cache/go-build-heap-sampling-docs go clean -cache
```

**Done when:** Docs, validators, final report, and graph all reflect the same runtime heap sampling
truth.

**Notes:** Do not mark the overall task done if only docs or only tests pass.

## Final Validation Bundle

Before reporting `DONE`, run:

```sh
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go test ./tools/internal/heaptelemetry ./tools/cmd/local-benchmark-tier1 ./tools/cmd/validate-local-benchmark-tier1 ./cli/cmd/tetra ./compiler ./compiler/internal/backend/linux_x64 ./compiler/internal/backend/x64 ./compiler/internal/backend/x64core -count=1
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go run ./tools/cmd/local-benchmark-tier1 --out-dir reports/benchmark-vnext-memory-baseline/tier1-runtime-heap-current-head --iterations 3
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/benchmark-vnext-memory-baseline/tier1-runtime-heap-current-head/report.json
GOCACHE=$(pwd)/.cache/go-build-heap-sampling-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
graphify update .
GOCACHE=$(pwd)/.cache/go-build-heap-sampling go clean -cache
GOCACHE=$(pwd)/.cache/go-build-heap-sampling-docs go clean -cache
```

## Stop Rules

Stop and report `PARTIAL` if:

- the validator can be made green only by accepting estimates as runtime heap evidence;
- the runner can collect RSS but not Tetra heap bytes;
- a successful Tetra row lacks a sidecar;
- the sidecar is written by the Go runner instead of the Tetra binary/runtime;
- a target other than linux-x64 needs major design work before the linux-x64 path is complete;
- the implementation requires broad allocator rewrites unrelated to sampling.

Report `BLOCKED` if:

- linux-x64 binaries cannot write sidecar files with the available runtime syscall support after
  focused investigation;
- the existing backend has no reliable normal-exit hook and adding one would be a separate runtime
  architecture project;
- required source files or benchmark artifacts are missing.

## Completion Levels

- `LOCAL`: sidecar parser and validator tests pass.
- `INTEGRATION`: `tetra build` can emit a telemetry-enabled linux-x64 binary and the runner can
  collect sidecars in unit tests.
- `END_TO_END`: a fresh local Tier 1 report validates with runtime-measured heap evidence for
  successful Tetra rows.
- `FINAL`: docs, report artifacts, validators, graph update, cache cleanup, and nonclaim audit are
  all complete.

Only `FINAL` may be reported as `DONE`.

## Risks

- Some allocation paths may not currently expose enough information for exact live heap decrement.
  If so, the implementation must clearly distinguish peak/total allocation evidence from current
  live heap evidence.
- Normal-exit sidecar emission may require backend work beyond a simple counter insertion.
- The current dirty worktree is large. The implementation must isolate its own changes and avoid
  reverting unrelated files.
- RSS remains out of scope for this plan. A later RSS sampler must use process evidence, not heap
  sidecars.
