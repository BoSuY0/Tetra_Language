# Linux-x64 Memory, Parallelism, And UI Production Plan

**Goal:** execute the active post-`v0.4.0` goal in the order Memory,
Parallelism, then UI, with Linux-x64 production evidence for each layer.

**Design:** `docs/plans/2026-05-17-linux-x64-memory-parallel-ui-production-design.md`

**Execution:** use test-driven development for each implementation task. Do not
mark the goal complete until the completion audit maps every objective
requirement to concrete evidence.

## Task 0: Baseline And Scope Lock

**Goal:** create a machine-readable target scope and prevent accidental
metadata-only production claims.

**Files to inspect:**

- `docs/spec/current_supported_surface.md`
- `docs/spec/v0_4_scope.md`
- `docs/release/v0_4_0_scope_decisions.json`
- `reports/v0.4.0/features.json`
- `scripts/release/v0_4_0/gate.sh`

**Approach:**

- Add a new post-`v0.4.0` scope artifact for Linux-x64 memory, parallelism, and
  UI production.
- Define explicit exclusions: non-Linux runtime promotion, WASM/browser
  production UI, EcoNet, and broad v1.0 guarantees.
- Add planned evidence IDs for memory, parallelism, and UI.

**Verification:**

```sh
jq empty <new-scope-artifact>
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```

**Done when:** the next production line has a clear scope artifact and docs do
not imply unsupported production behavior.

## Task 1: Memory Evidence Validator Skeleton

**Goal:** define the evidence contract before implementing new memory behavior.

**Files to inspect:**

- `tools/validators/nativeui/report.go`
- `tools/validators/actordist/report.go`
- `tools/cmd/validate-native-ui-runtime/main.go`
- `tools/cmd/validate-distributed-actor-runtime/main.go`

**Files to add or modify:**

- `tools/validators/memoryprod/report.go`
- `tools/cmd/validate-memory-production/main.go`
- matching tests under the same packages

**Approach:**

- Create `tetra.memory.production.v1`.
- Require Linux-x64 target and host.
- Require process evidence, positive memory cases, negative safety cases,
  stress cases, and explicit allocator/bounds diagnostics.
- Reject fake, mock, placeholder, docs-only, build-only, and metadata-only
  evidence markers.

**Verification:**

```sh
go test ./tools/validators/memoryprod ./tools/cmd/validate-memory-production -count=1
```

**Done when:** fake and incomplete memory reports fail, while a complete fixture
passes.

## Task 2: Memory Runtime Contract

**Goal:** document and test the Linux-x64 allocator and bounds ABI before
backend changes.

**Files to inspect:**

- `docs/spec/runtime_abi.md`
- `docs/spec/ownership_v1.md`
- `lib/core/memory.tetra`
- `compiler/internal/semantics/builtins.go`
- `compiler/internal/actorsrt/linux_x64.go`
- `compiler/internal/actorsrt/linux_x64_emit.go`

**Approach:**

- Specify allocator symbols, pointer ownership rules, bounds failure semantics,
  and `cap.mem` requirements.
- Decide which operations are compile-time diagnostics and which are runtime
  checked failures.
- Keep the first allocator contract narrow: deterministic Linux-x64 behavior
  before any cross-target claim.

**Verification:**

```sh
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```

**Done when:** the ABI and safety docs are precise enough to write failing tests
without guessing.

## Task 3: Memory TDD Slice

**Goal:** implement the first executable memory production slice.

**Files to inspect or modify:**

- `compiler/internal/semantics/region.go`
- `compiler/internal/semantics/checker.go`
- `compiler/internal/semantics/builtins.go`
- `compiler/internal/actorsrt/linux_x64.go`
- `compiler/internal/actorsrt/linux_x64_emit.go`
- `lib/core/memory.tetra`
- `compiler/tests/ownership/`
- `compiler/tests/runtime/`
- `cli/cmd/tetra/check_diagnostics_ownership_*.go`
- `examples/core_memory_smoke.tetra`
- `tools/cmd/memory-production-smoke/main.go`

**Approach:**

- Write failing tests for allocator use, invalid free, double free,
  use-after-free, borrow escape through heap/slice/struct/closure, and
  actor/task transfer of memory-bearing values.
- Implement the smallest runtime and semantic changes needed for each test.
- Add deterministic stress and fuzz-like cases that fit CI, including a
  `memcpy_u8`/`memset_u8` length sweep with sentinel checks.
- Generate a memory production report and validate it.
- Embed a completion audit in `tetra.memory.production.v1` that maps Memory
  Production Core requirements to concrete artifacts, commands, tests, and
  docs.
- Write and validate `artifact-hashes.json` for the memory release-gate output.

**Verification:**

```sh
go test ./compiler/... -run "Ownership|Borrow|Consume|Inout|Region|Memory|Unsafe|Capability|Actor|Task" -count=1
go test ./cli/... -run "Ownership|Memory|Diagnostics" -count=1
bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir reports/post-v0.4-memory
go run ./tools/cmd/validate-artifact-hashes --manifest reports/post-v0.4-memory/artifact-hashes.json
go test ./tools/validators/memoryprod ./tools/cmd/validate-memory-production -count=1
```

**Done when:** `tetra.memory.production.v1` exists and validates against real
Linux-x64 execution and negative safety cases.

## Task 4: Parallel Evidence Validator Skeleton

**Goal:** define production evidence for scheduler, actor, and task behavior.

**Files to inspect:**

- `tools/validators/actordist/report.go`
- `tools/cmd/validate-distributed-actor-runtime/main.go`
- `compiler/task_runtime_test.go`
- `compiler/tests/ownership/actor_task_stress_test.go`

**Files to add or modify:**

- `tools/validators/parallelprod/report.go`
- `tools/cmd/validate-parallel-production/main.go`
- matching tests

**Approach:**

- Create `tetra.parallel.production.v1`.
- Require cases for scheduler fairness, join/cancel/deadline/select/group
  lifecycle, mailbox capacity/backpressure, invalid handles, cancellation
  storms, timeouts, and transfer diagnostics.
- Require a completion audit that maps scheduler, lifecycle, mailbox,
  transfer, race-safety, stress, docs, and release-gate requirements to
  concrete evidence.
- Reject evidence that only proves transport, build, or docs.

**Verification:**

```sh
go test ./tools/validators/parallelprod ./tools/cmd/validate-parallel-production -count=1
```

**Done when:** complete fixtures pass and partial actor/task reports fail.

## Task 5: Parallel Runtime TDD Slice

**Goal:** promote cooperative task/actor runtime behavior to a production
contract.

**Files to inspect or modify:**

- `docs/spec/actors.md`
- `docs/spec/runtime_abi.md`
- `docs/user/async_actors_guide.md`
- `compiler/internal/actorsrt/linux_x64.go`
- `compiler/internal/actorsrt/linux_x64_emit.go`
- `compiler/task_runtime_test.go`
- `compiler/actors_test.go`
- `compiler/distributed_actor_runtime_test.go`
- `examples/task_bounded_stress.tetra`
- `examples/actors_tagged_stress.tetra`
- `tools/cmd/parallel-production-smoke/main.go`
- `scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh`

**Approach:**

- Turn unspecified mailbox overflow and invalid handle behavior into stable
  diagnostics or deterministic runtime status.
- Add stress examples for many tasks, many messages, cancellation storms, and
  deadline-heavy waits.
- Add transfer and race-safety diagnostics for unsupported shared mutable
  state across task/actor boundaries.
- Generate and validate `tetra.parallel.production.v1`.
- Write and validate `artifact-hashes.json` for the parallel release-gate
  output.

**Verification:**

```sh
go test ./compiler/... -run "Task|Actor|Actors|Runtime|Scheduler|Deadline|Cancel|Stress|Ownership" -count=1
bash scripts/release/post_v0_4/parallel-production-linux-x64-smoke.sh --report-dir reports/post-v0.4-parallel
go run ./tools/cmd/validate-artifact-hashes --manifest reports/post-v0.4-parallel/artifact-hashes.json
go test ./tools/validators/parallelprod ./tools/cmd/validate-parallel-production -count=1
```

**Done when:** parallel production evidence proves real Linux-x64 runtime
behavior and rejects unsupported race/capacity paths.

## Task 6: UI Evidence Validator Upgrade

**Goal:** define a full desktop runtime evidence contract beyond current native
UI smoke.

**Files to inspect:**

- `tools/validators/nativeui/report.go`
- `tools/cmd/native-ui-runtime-smoke/main.go`
- `tools/cmd/validate-native-ui-runtime/main.go`
- `docs/spec/ui_v1.md`

**Files to add or modify:**

- `tools/validators/uiprod/report.go`
- `tools/cmd/validate-ui-production-runtime/main.go`
- matching tests

**Approach:**

- Create `tetra.ui.desktop-runtime.v1`.
- Require window lifecycle, layout, controls, input, event loop, async command,
  timer, redraw/update, and error/crash cases.
- Require Linux-x64 build, app, runtime, and stress process evidence, plus an
  embedded completion audit for every UI production requirement.
- Keep strict rejection for mock, placeholder, metadata-only,
  native-shell-only, sidecar-only, web-only, build-only, and runtime-less
  reports.

**Verification:**

```sh
go test ./tools/validators/uiprod ./tools/cmd/validate-ui-production-runtime -count=1
```

**Done when:** `tools/validators/uiprod/report.go` and
`tools/cmd/validate-ui-production-runtime/main.go` accept a complete
`tetra.ui.desktop-runtime.v1` fixture and reject runtime-less or partial UI
production evidence.

## Task 7: UI Runtime TDD Slice

**Goal:** implement the first real Linux-x64 desktop UI runtime.

**Files to inspect or modify:**

- `docs/spec/ui_v1.md`
- `compiler/internal/semantics/ui.go`
- `compiler/internal/lower/ui.go`
- `compiler/internal/backend/native_shell/codegen.go`
- `tools/cmd/native-ui-runtime-smoke/main.go`
- `tools/validators/nativeui/report.go`
- `examples/ui_native_shell_smoke.tetra`
- `examples/projects/dogfood_web_ui/src/main.tetra`

**Approach:**

- Start with a narrow but real runtime: one window, deterministic layout,
  text/value/input/action widgets, event loop, timers, state binding, redraw,
  and controlled error handling.
- Add async UI command support only after Task 5 proves transfer and scheduler
  safety.
- Add a dogfood desktop app that uses state, input, async command, timer, and
  redraw.
- Generate and validate `tetra.ui.desktop-runtime.v1`.
- Write and validate `artifact-hashes.json` for the UI production runtime
  release-gate output.

**Verification:**

```sh
go test ./compiler/... -run "UI|Native|Task|Actor|Runtime" -count=1
bash scripts/release/post_v0_4/ui-production-runtime-linux-x64-smoke.sh --report-dir reports/post-v0.4-ui
go run ./tools/cmd/validate-artifact-hashes --manifest reports/post-v0.4-ui/artifact-hashes.json
go test ./tools/validators/nativeui ./tools/validators/uiprod ./tools/cmd/validate-ui-production-runtime -count=1
```

**Done when:** a real Linux-x64 desktop Tetra app runs through the production
UI runtime and the validator rejects metadata-only evidence.

## Task 8: Combined Production Gate

**Goal:** create one gate that proves the active goal end-to-end.

**Files to inspect or modify:**

- `scripts/release/v0_4_0/gate.sh`
- `tools/cmd/validate-release-gate-summary`
- `tools/cmd/validate-artifact-hashes`
- `docs/release/`
- `reports/`

**Approach:**

- Add a new versioned gate for the memory/parallel/UI production line.
- Include memory, parallelism, and UI validators.
- Preserve the required order: Memory, then Parallelism, then UI.
- Revalidate each layer artifact and write a final hash manifest over the
  combined report directory.
- Require full baseline tests, docs verification, artifact hashes, security
  review, and clean release-state.
- Add a completion audit that maps every active goal requirement to concrete
  evidence.

**Verification:**

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
bash scripts/release/post_v0_4/memory-parallel-ui-production-linux-x64-gate.sh --report-dir <fresh-report-dir>
```

**Done when:** a clean Linux-x64 snapshot passes the combined gate and the
completion audit has no missing or weakly verified requirements.
