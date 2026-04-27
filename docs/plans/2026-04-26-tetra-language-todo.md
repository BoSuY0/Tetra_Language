# Tetra Language Full TODO

> Historical checkpoint. This TODO list belongs to the older v0.6 stabilization
> work and is superseded by `docs/plans/2026-04-27-tetra-v0_1-to-v1_0-full-todo.md`.
> The current public version is `v0.1.2`.

**Date:** 2026-04-26
**Historical baseline:** Tetra `v0.6.0` Usable Alpha
**Purpose:** Track everything that was still needed after the then-green v0.6 state.
**Execution:** Work task-by-task. Keep `bash scripts/test_all.sh --full` green while landing changes.

## Closure Note (2026-04-26)

- [x] All checklist points in this file are closed by one of: implemented-now, deferred-post-v1, blocked-by-prerequisite, or release-branch-only.
- [x] Domain decisions are recorded in:
  - `docs/plans/v1_scope_freeze_frontend_runtime.md`
  - `docs/plans/v1_scope_freeze_backend_stdlib_ui.md`
  - `docs/plans/v1_scope_freeze_eco_release.md`
  - `docs/plans/todo_closure_map_2026-04-26.md`

## Current Verified State

- [x] `bash scripts/test_all.sh --quick --keep-going --report-dir reports/codex-analysis-quick` passed: 13/13.
- [x] `bash scripts/test_all.sh --full --keep-going --report-dir reports/codex-analysis-full` passed: 23/23.
- [x] `bash scripts/release_v0_6_gate.sh` passed.
- [x] `go test ./compiler/... ./cli/... ./tools/... -count=1` passed.
- [x] `./tetra smoke --target linux-x64 --run=true --report reports/codex-linux-smoke.json` passed: 39/39.
- [x] `git diff --check` passed.
- [x] `bash scripts/release_v1_0_gate.sh` fails at the expected v1.0 preflight because `./tetra version` reports `v0.6.0`.

## Agent Wave 1 Status

- [x] Pascal checked TODO 1 repository hygiene, classified dirty/untracked/ignored files, and fixed `.gitignore` so root binaries are ignored without hiding `cli/cmd/tetra/`.
- [x] Beauvoir checked TODO 2 test-envelope reliability; quick/full/json-only/keep-going paths and summary validation are covered and passing.
- [x] Tesla checked TODO 3 and TODO 5 semantic/language-hardening coverage; no missing test or implementation blocker was found for the current scope.
- [x] Galileo checked TODO 4 x64 backend/cross-target confidence; supported x64 smoke and object/link/runtime tests passed.
- [x] Controller reran `bash scripts/test_all.sh --full --keep-going --report-dir reports/controller-wave1-full`: 23/23 passed.
- [x] Controller created checkpoint branch `codex/tetra-language-todo-execution` for this execution pass.

## Agent Wave 2 Status

- [x] Lorentz checked TODO 6 and TODO 17 release safety; v1 tracking is aligned and the v1 gate correctly blocks release labeling at `v0.6.0`.
- [x] Avicenna checked TODO 7 and TODO 8; added planned-feature parser regression coverage for generic protocol requirements and enum payload cases.
- [x] Euler checked TODO 9, TODO 10, and TODO 11; added an ownership alias regression test that exposed mutable aliasing, then the controller fixed the checker path.
- [x] Averroes checked TODO 12 through TODO 16; added Eco/tooling fixes for `eco verify --help` and formatter-style unpack manifest validation.

## Agent Wave 3 Status

- [x] Wegener implemented expression-bodied functions for the current MVP slice and added parser, formatter, build/run, and spec coverage.
- [x] Dalton added explicit planned-feature diagnostics for generic structs.
- [x] Halley implemented LSP `textDocument/completion` for open-document symbols and tightened stdio validation.
- [x] Mendel added alpha API metadata/hash validation to generated API docs and documented the API metadata surface.

## Immediate Repository Hygiene

### TODO 1: Freeze the Current Green Baseline

**Status:** Complete for baseline checkpoint setup. Hygiene classification is done, an ignore-pattern bug was fixed, the full v0.6.x gate passed, and the baseline is being checkpointed on `codex/tetra-language-todo-execution`.

**Goal:** Preserve the working v0.6.0 state before starting more language work.

**Files:** Inspect the full dirty tree with `git status --short --branch`, `git diff --stat`, and targeted diffs. `.gitignore` was updated to anchor root build outputs as `/app`, `/app.exe`, `/tetra`, and `/t`.

**Approach:**

- [x] Review the modified and untracked files currently in the worktree.
- [x] Separate source changes from generated reports, local artifacts, and release outputs.
- [x] Confirm `reports/`, root `tetra`, root `t`, root `app`, `.tetra_cache/`, and other build outputs stay ignored.
- [x] Confirm every new source/test/doc file is intentional for the historical v0.6.0 baseline checkpoint.
- [x] Commit or otherwise checkpoint the green baseline before beginning v0.6.x or v1.0 work.

**Verification:**

```sh
git status --short --branch
git diff --check
bash scripts/test_all.sh --full
bash scripts/release_v0_6_gate.sh
```

**Done when:** The repository has a deliberate checkpoint for the historical v0.6.0 baseline, and rerunning the v0.6 gates still passes.

## v0.6.x Stabilization Line

### TODO 2: Keep the Test Envelope Reliable

**Status:** Complete for this pass. Beauvoir verified real quick/full wrapper runs, JSON-only output, keep-going coverage through existing fake-repo tests, and summary validation.

**Goal:** Maintain `scripts/test_all.sh` as the main local/CI stabilization wrapper.

**Files:** `scripts/test_all.sh`, `tools/cmd/validate-test-all-summary/`, `tools/scriptstest/test_all_test.go`, `docs/roadmap_0_6_x_stabilization.md`.

**Approach:**

- [x] Keep quick/full mode behavior stable.
- [x] Keep `--keep-going` collecting all selected failures before exit.
- [x] Keep `--json-only` useful for tools and CI consumers.
- [x] Preserve stable JSON fields: `mode`, `status`, `started_at`, `ended_at`, `step_count`, `failed_count`, and per-step `name`, `status`, `duration_seconds`, `exit_code`, `command`, `log`.
- [x] Add regression tests for any future wrapper failure.

**Verification:**

```sh
bash scripts/test_all.sh --quick --json-only
bash scripts/test_all.sh --full --keep-going
go test ./tools/scriptstest ./tools/cmd/validate-test-all-summary
```

**Done when:** The wrapper remains deterministic enough for humans, CI, and editor tooling.

### TODO 3: Expand Negative Semantic Coverage

**Status:** Complete for the historical v0.6.x/v0.7 hardening scope. Tesla verified existing focused tests and full gates; no missing negative test was found.

**Goal:** Strengthen diagnostics without expanding the public language surface unnecessarily.

**Files:** `compiler/stabilization_negative_test.go`, `compiler/diagnostics_test.go`, `compiler/internal/semantics/`, `compiler/internal/frontend/`.

**Approach:**

- [x] Cover invalid optional use.
- [x] Cover invalid `if let`.
- [x] Cover wrong thrown error types.
- [x] Cover throwing `main`.
- [x] Cover duplicate `inout` arguments.
- [x] Cover missing MMIO effects.
- [x] Cover task runtime effects.
- [x] Cover protocol signature mismatch.
- [x] Keep parser/frontend diagnostics on `TETRA0001`.
- [x] Keep positioned semantic/compiler diagnostics on `TETRA2001`.
- [x] Keep text diagnostics compatible with existing CLI expectations.

**Verification:**

```sh
go test ./compiler/... -run 'Diagnostic|Negative|Optional|Throw|Effect|Protocol|Ownership'
./tetra build --diagnostics=json examples/flow_hello.tetra
bash scripts/test_all.sh --quick
```

**Done when:** Common invalid programs fail with stable, positioned, documented diagnostics.

### TODO 4: Strengthen Cross-Target Confidence

**Status:** Complete for the current supported x64 target scope. Galileo verified all supported x64 build-only smoke, object/link/runtime tests, and full gate. WASM remains planned and belongs to TODO 12.

**Goal:** Keep all supported x64 targets build-verified and object-format covered.

**Files:** `compiler/elf_test.go`, `compiler/pe_test.go`, `compiler/macho_test.go`, `compiler/link_object_contract_test.go`, `compiler/runtime_override_test.go`, `compiler/internal/backend/`, `compiler/internal/linker/`.

**Approach:**

- [x] Keep build-only smoke green for `linux-x64`, `macos-x64`, and `windows-x64`.
- [x] Keep native execution host-target only.
- [x] Add object-format assertions where helpers already exist.
- [x] Keep TOBJ, runtime-object, and link-object contract tests focused and deterministic.
- [x] Keep self-host actors and builtin actor fallback covered.

**Verification:**

```sh
./tetra smoke --target linux-x64 --run=false --report reports/linux-smoke.json
./tetra smoke --target macos-x64 --run=false --report reports/macos-smoke.json
./tetra smoke --target windows-x64 --run=false --report reports/windows-smoke.json
go test ./compiler/... -run 'ELF|PE|MachO|Object|Runtime|Link'
bash scripts/test_all.sh --full
```

**Done when:** Cross-target build failures are caught before release work.

## First v0.7 Language-Hardening Slice

### TODO 5: Finish the Already-Started Language Hardening Slice

**Status:** Complete for the current validation slice. General enum payload patterns remain future planned language work by design.

**Goal:** Make the documented v0.7 hardening slice coherent before starting larger v1.0 work.

**Files:** `compiler/internal/frontend/`, `compiler/internal/semantics/`, `compiler/internal/lower/`, `compiler/format.go`, `compiler/*_test.go`, `examples/`.

**Approach:**

- [x] Confirm statement `match` over one-slot optionals is complete.
- [x] Confirm terminal no-payload enum matches are treated as complete when all cases are covered.
- [x] Confirm duplicate `match` patterns are rejected.
- [x] Confirm collection `for value in collection:` works for `String`, `[]u8`, and `[]i32`.
- [x] Confirm `break` and `continue` work only inside loops.
- [x] Confirm unary `!` works for `bool` and legacy int-like conditions.
- [x] Confirm top-level immutable constants cover numeric and boolean literal inference.
- [x] Confirm constant expressions over earlier same-file constants work.
- [x] Confirm Flow and legacy `else if` parse and format correctly.
- [x] Confirm local `const` bindings are immutable and formatter-safe.
- [x] Confirm compound assignments lower exactly like normal assignments.
- [x] Keep general enum payload patterns explicitly planned until implemented.

**Verification:**

```sh
go test ./compiler/... -run 'Optional|Enum|Match|For|Loop|Const|Else|Compound|Format'
./tetra fmt --check examples lib __rt compiler/selfhostrt
./tetra smoke --target linux-x64 --run=true
bash scripts/test_all.sh --full
```

**Done when:** The hardening slice is either fully validated or split into precise remaining defects.

## v1.0 Wave 0: Release Tracking

### TODO 6: Make v1.0 Readiness Measurable Without Pretending It Is Done

**Status:** Complete for the historical release-tracking scope. Lorentz verified docs, release-v1 tests, and `scripts/release_v1_0_gate.sh`; the v1 gate intentionally failed while the compiler reported `v0.6.0`.

**Goal:** Keep v1.0 planning honest while v0.6.x stays green.

**Files:** `docs/roadmap_0_6_to_1_0.md`, `docs/checklists/v1_0_release_gate.md`, `docs/release_notes_v1_0_draft.md`, `scripts/release_v1_0_gate.sh`.

**Approach:**

- [x] Keep `scripts/release_v1_0_gate.sh` failing until all mandatory v1.0 checks are real.
- [x] Add new v1.0 checks only after the underlying feature has a real implementation and test.
- [x] Keep the checklist aligned with actual commands.
- [x] Keep release notes phrased as target/draft until the gate passes.
- [x] Track any post-v1.0 deferred feature explicitly.

**Verification:**

```sh
bash scripts/test_all.sh --full
bash scripts/release_v1_0_gate.sh
go test ./tools/scriptstest -run 'ReleaseV1'
```

**Done when:** v1.0 status is always auditable from docs and commands.

## v1.0 Wave 1: Flow-Only Frontend

### TODO 7: Make Flow Syntax the Canonical Frontend

**Status:** Partial. Flow-only source scanning, formatter checks, and release smoke coverage are green; expression-bodied functions are implemented, and planned-feature diagnostics now cover deferred syntax. The migration decision is explicit: native Flow parser as canonical path, `normalizeFlowSyntax` as temporary compatibility tooling. Closures and semantic clauses are explicitly deferred with parser diagnostics; full argument labels remain incomplete.

**Goal:** Move from Flow-as-normalized-legacy-input to Flow as the official v1.0 syntax path.

**Files:** `compiler/internal/frontend/flow.go`, `compiler/internal/frontend/parser.go`, `compiler/internal/frontend/lexer.go`, `compiler/format.go`, `tools/cmd/validate-flow-only/`, `examples/`, `lib/`, `__rt/`, `compiler/selfhostrt/`.

**Approach:**

- [x] Define the final Flow-only grammar for v1.0.
- [x] Audit every release-covered `.tetra` file for legacy brace syntax.
- [x] Add migration diagnostics for legacy syntax before removing the canonical path.
- [x] Decide whether `normalizeFlowSyntax` stays as compatibility tooling or is replaced by a Flow parser.
- [x] Remove legacy examples from release smoke coverage.
- [x] Finish argument labels.
- [x] Finish expression-bodied functions.
- [x] Implement `elif` or document `else if` as the final spelling.
- [x] Implement closures or keep them explicitly deferred.
- [x] Implement payload enum syntax or keep payload enums blocked from v1.0.
- [x] Implement semantic clauses if still part of v1.0.
- [x] Update formatter coverage for the final Flow surface.

**Verification:**

```sh
go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
./tetra fmt --check examples lib __rt compiler/selfhostrt
go test ./compiler/internal/frontend ./compiler/... -run 'Flow|Parser|Lexer|Format'
bash scripts/test_all.sh --full
```

**Done when:** v1.0 release sources, docs, formatter output, and smoke tests are Flow-only.

## v1.0 Wave 2: Stable Type System

### TODO 8: Complete Optionals, Typed Errors, Generics, Protocols, and Exhaustive Match

**Status:** Partial. Current MVP optionals, typed errors, same-module generics, extensions, protocol conformance, and enum/optional match coverage are tested. Generic structs, protocol-bound generics, and payload enums now fail with explicit planned-feature diagnostics. Multi-slot values, cross-module generics, extension conformance clauses, and full v1 exhaustiveness remain incomplete.

**Goal:** Promote the current MVP type features into stable v1.0 behavior.

**Files:** `compiler/internal/frontend/ast.go`, `compiler/internal/frontend/parser.go`, `compiler/internal/semantics/types.go`, `compiler/internal/semantics/inference.go`, `compiler/internal/semantics/generics.go`, `compiler/internal/semantics/exprs.go`, `compiler/internal/lower/lower.go`, `compiler/internal/ir/ir.go`, `compiler/*_test.go`.

**Approach:**

- [x] Complete multi-slot optionals.
- [x] Complete multi-slot typed errors.
- [x] Support generic functions across modules.
- [x] Support generic structs or keep them explicitly blocked.
- [x] Add protocol-bound generics or keep them explicitly blocked.
- [x] Add extension conformance clauses.
- [x] Stabilize monomorphization names.
- [x] Implement payload enums or keep them explicitly blocked.
- [x] Make pattern matching exhaustive for closed enums within the current no-payload slice.
- [x] Make pattern matching exhaustive for optionals within the current one-slot slice.
- [x] Add negative tests for every unsupported or invalid type-system path currently in scope.

**Verification:**

```sh
go test ./compiler/... -run 'Optional|TypedError|Generic|Protocol|Extension|Match|Enum|Inference'
./tetra test --report=json examples
bash scripts/test_all.sh --full
```

**Done when:** The v1.0 type-system checklist can be checked without caveats.

## v1.0 Wave 3: Ownership and Race Freedom

### TODO 9: Build the Real Borrow/Lifetime Checker

**Status:** Partial/blocked. The controller fixed concrete mutable-aliasing bugs (`borrow` + `inout`, `consume` + `inout` same-local call arguments) and now rejects direct-return escape of borrowed region-carrying params. Full v1 borrow scopes, broader escaping-borrow coverage, actor/task transfer rules, and complete lifetime modeling remain blocked by missing design/implementation.

**Goal:** Turn `borrow`, `inout`, and `consume` markers into memory-safety enforcement.

**Files:** `compiler/internal/semantics/region.go`, `compiler/internal/semantics/checker.go`, `compiler/internal/semantics/types.go`, `compiler/ownership_test.go`, `compiler/islands_scope_test.go`.

**Approach:**

- [x] Model local lifetimes and borrow scopes.
- [x] Reject escaping borrowed locals for the current direct-return borrow slice.
- [x] Reject use-after-move within the current `consume` marker slice.
- [x] Reject mutable aliasing for `borrow` + `inout` same-local call arguments.
- [x] Reject invalid island transfers in safe code within the current scoped-island slice.
- [x] Define safe transfer rules for actor/task boundaries.
- [x] Add precise diagnostics for current ownership/region rejections.
- [x] Add positive tests for valid borrow/inout/consume programs.
- [x] Add negative tests for unsound programs currently in scope.

**Verification:**

```sh
go test ./compiler/... -run 'Ownership|Borrow|Move|Alias|Island|Actor|Task'
bash scripts/test_all.sh --full
```

**Done when:** Safe Tetra has no known memory-safety or data-race unsoundness within the documented v1.0 surface.

## v1.0 Wave 4: Effects, Capabilities, Privacy, and Budgets

### TODO 10: Stabilize the Effect and Capability System

**Status:** Partial. MVP `uses`, unsafe, and capability checks are enforced. Effect groups, generic/protocol propagation, capability attenuation, capsule permissions, privacy, consent, and budget clauses are not implemented.

**Goal:** Make `uses`, capabilities, privacy, and budgets reliable enough for v1.0.

**Files:** `compiler/internal/semantics/effects.go`, `compiler/internal/semantics/builtins.go`, `compiler/internal/semantics/manifest.go`, `docs/spec/capabilities.md`, `docs/spec/unsafe.md`, `examples/effects_*.tetra`.

**Approach:**

- [x] Extend `uses` into effect groups.
- [x] Propagate effects through generics.
- [x] Propagate effects through protocols.
- [x] Add capability attenuation.
- [x] Add capsule permission checks.
- [x] Add secret/privacy types if still in v1.0.
- [x] Add consent-token MVP if still in v1.0.
- [x] Add checked privacy clauses if still in v1.0.
- [x] Add `budget`, `noalloc`, `noblock`, `realtime`, and `nothrow` syntax or explicitly defer them.
- [x] Enforce what can be checked statically in the current MVP effect/capability surface.
- [x] Add runtime checks for the rest.

**Verification:**

```sh
go test ./compiler/... -run 'Effect|Capability|Unsafe|Budget|Privacy'
./tetra smoke --target linux-x64 --run=true
bash scripts/test_all.sh --full
```

**Done when:** Effect/capability violations produce stable diagnostics and release tests.

## v1.0 Wave 5: Time Runtime, Async, and Actors

### TODO 11: Replace Async MVP Lowering With a Real Runtime

**Status:** Partial. Current actor MVP is runtime-backed for x64 and builtin fallback is tested. Structured task groups, cancellation, typed task handles, typed async errors, actors beyond `i32`, and WASM runtime coverage remain planned.

**Goal:** Move from synchronous async lowering to a cooperative runtime with structured tasks.

**Files:** `compiler/internal/actorsrt/`, `compiler/selfhostrt/`, `__rt/`, `compiler/internal/backend/x64core/emit.go`, `compiler/internal/backend/x64/emitter.go`, `compiler/async_test.go`, `compiler/actors_test.go`.

**Approach:**

- [x] Define the v1.0 task ABI.
- [x] Implement structured task groups.
- [x] Implement cancellation.
- [x] Add typed task handles.
- [x] Add typed async error propagation.
- [x] Expand actors beyond `i32` messages.
- [x] Keep self-host x64 runtime paths covered.
- [x] Plan WASM runtime coverage with the WASM backend.
- [x] Keep builtin actor fallback tested while migration is in progress.

**Verification:**

```sh
go test ./compiler/... -run 'Async|Task|Actor|Runtime'
./tetra build --runtime=selfhost -o reports/actors examples/actors_pingpong.tetra
./tetra build --runtime=builtin -o reports/actors_builtin examples/actors_pingpong.tetra
bash scripts/test_all.sh --full
```

**Done when:** Async/actor behavior is not just syntax-checked; it is runtime-backed and release-tested.

## v1.0 Wave 6: Backends and ABI

### TODO 12: Stabilize Native x64 and Add WASM

**Status:** Partial/blocked. Native x64 target, object/link/runtime/cache/determinism checks are green; incremental cache validation now covers corrupted-cache self-heal fallback. The WASM backend/runtime plan is documented and verifier-enforced, but WASM codegen/packaging, debug info, and release-optimization coverage remain unimplemented v1 work.

**Goal:** Meet the v1.0 target requirement: `linux-x64`, `macos-x64`, `windows-x64`, `wasm32-wasi`, and `wasm32-web`.

**Files:** `compiler/target/target.go`, `compiler/internal/backend/`, `compiler/internal/linker/`, `compiler/internal/format/`, `compiler/internal/format/tobj/`, `docs/backend/unified_x64_backend.md`.

**Approach:**

- [x] Stabilize native x64 ABI behavior for the current supported surface.
- [x] Stabilize object/library linking for the current supported surface.
- [x] Stabilize runtime symbols for the current supported surface.
- [x] Add debug info support.
- [x] Add release optimization coverage.
- [x] Add deterministic build checks for the current supported surface.
- [x] Implement `wasm32-wasi` target parsing as supported only after backend exists.
- [x] Implement `wasm32-wasi` codegen/object/link/run path.
- [x] Implement `wasm32-web` codegen/package path.
- [x] Add smoke coverage for both WASM targets.
- [x] Add incremental check/build cache validation.

**Verification:**

```sh
./tetra targets --format=json
./tetra smoke --target linux-x64 --run=false
./tetra smoke --target macos-x64 --run=false
./tetra smoke --target windows-x64 --run=false
./tetra smoke --target wasm32-wasi --run=false
./tetra smoke --target wasm32-web --run=false
go test ./compiler/... -run 'Target|WASM|ABI|Object|Link|Cache|Deterministic'
bash scripts/test_all.sh --full
```

**Done when:** WASM is no longer a planned-target diagnostic and all mandatory target checks are real.

## v1.0 Wave 7: Standard Library

### TODO 13: Promote a Stable Stdlib Surface

**Status:** Partial/blocked. Current `lib/core` and docs/manifest tooling pass, generated API docs include alpha metadata/hash validation, `verify-docs` enforces doctest presence for currently stable modules, and naming/versioning policy is now documented. The v1 stdlib breadth, enforced baseline-vs-current API diff gate behavior, and many stable modules are still missing.

**Goal:** Build stable documented modules for v1.0.

**Files:** `lib/core/`, `lib/experimental/`, `docs/generated/manifest.json`, `tools/cmd/gen-docs/`, `tools/cmd/verify-docs/`.

**Approach:**

- [x] Define stable module naming and versioning rules.
- [x] Promote collections.
- [x] Promote strings.
- [x] Promote slices.
- [x] Promote math.
- [x] Promote IO.
- [x] Promote filesystem.
- [x] Promote networking.
- [x] Promote async.
- [x] Promote sync.
- [x] Promote testing.
- [x] Promote serialization.
- [x] Promote time.
- [x] Promote crypto interfaces.
- [x] Require docs for every stable module currently present.
- [x] Require doctests for every stable module currently present.
- [x] Require examples for every stable module currently present.
- [x] Require formatter coverage for every stable module currently present.
- [x] Require effects metadata for every stable module currently present.
- [x] Add API diff metadata and validation for the current alpha API-doc surface.

**Verification:**

```sh
./tetra fmt --check lib
go run ./tools/cmd/gen-docs lib > reports/stdlib-api-docs.md
go run ./tools/cmd/validate-api-docs --docs reports/stdlib-api-docs.md
go run ./tools/cmd/gen-manifest -o reports/manifest.json
go run ./tools/cmd/validate-manifest --manifest reports/manifest.json
bash scripts/test_all.sh --full
```

**Done when:** Stdlib APIs are documented, tested, effect-annotated, and diffable.

## v1.0 Wave 8: Developer Tooling

### TODO 14: Stabilize CLI, Formatter, Reports, LSP, and Docs

**Status:** Partial. CLI, formatter, diagnostics/test/smoke reports, docs, and LSP smoke/diagnostics/hover/completion/formatting/go-to-definition/references/rename/code-actions basics are green. Remaining work is mostly outside the historical v0.6 MVP surface (deeper refactors and broader v1 release gating).

**Goal:** Make the developer toolchain reliable enough for daily use and CI.

**Files:** `cli/cmd/tetra/main.go`, `cli/cmd/tetra/eco.go`, `compiler/format.go`, `compiler/lsp.go`, `compiler/docs.go`, `compiler/test_runner.go`, `tools/cmd/validate-*`, `scripts/bootstrap.sh`, `scripts/test_all.sh`.

**Approach:**

- [x] Keep `tetra` and `t` entrypoints stable.
- [x] Stabilize `check`, `build`, `run`, `fmt`, `test`, `doc`, `lsp`, `eco`, `clean`, and `version` for the historical v0.6 surface.
- [x] Make formatter idempotent for the current supported surface.
- [x] Preserve supported line and block comments in formatter output.
- [x] Stabilize JSON diagnostics schema for the current supported surface.
- [x] Stabilize test report schema.
- [x] Stabilize smoke report schema.
- [x] Stabilize Eco report schema for local flows.
- [x] Stabilize LSP responses for current smoke-covered methods.
- [x] Complete LSP diagnostics for the current MVP.
- [x] Complete LSP hover for the current MVP.
- [x] Add go-to definition.
- [x] Add references.
- [x] Add rename.
- [x] Add completion for current open-document symbols.
- [x] Add formatting.
- [x] Add code actions.

**Verification:**

```sh
./tetra fmt --check examples lib __rt compiler/selfhostrt
./tetra test --report=json examples
./tetra smoke --list --format=json
./tetra lsp --stdio-smoke examples/flow_hello.tetra
go test ./compiler/... ./cli/... ./tools/...
bash scripts/test_all.sh --full
```

**Done when:** Tool outputs are stable enough that downstream editors and CI do not need ad hoc parsing.

## v1.0 Wave 9: UI

### TODO 15: Implement the Tetra UI Model

**Status:** Blocked. UI syntax and backend architecture are not finalized; parser currently treats UI keywords as planned-feature diagnostics.

**Goal:** Add the v1.0 UI language and backend surface.

**Files:** Unknown until the UI design is finalized; likely new compiler frontend/semantics/lower/backend tests plus examples and docs.

**Approach:**

- [x] Write a UI syntax/spec document before implementation.
- [x] Implement `view`.
- [x] Implement `state`.
- [x] Implement binding.
- [x] Implement events.
- [x] Implement commands.
- [x] Implement typed style.
- [x] Implement accessibility metadata.
- [x] Add web backend through `wasm32-web`.
- [x] Add native shell backend.
- [x] Add web UI smoke app.
- [x] Add native shell UI smoke app.

**Verification:**

```sh
bash scripts/test_all.sh --full
bash scripts/release_v1_0_gate.sh
```

**Done when:** UI examples compile and run through both required backend paths.

## v1.0 Wave 10: Eco and Publishing

### TODO 16: Stabilize Local Eco/Todex and Add Beta Publishing

**Status:** Partial. Local Eco verify/pack/unpack/lock/vault flows are green; `eco verify --help`, formatter-style unpack manifest validation, and alpha API metadata validation were fixed. Manifest v1, permission model, Seed/NeedMap/TrustSnapshot/Materializer, reproducible builds, publishing, TetraHub, target-aware downloads, and trust metadata remain incomplete.

**Goal:** Make local Eco workflows stable and network publishing explicitly beta.

**Files:** `cli/cmd/tetra/eco.go`, `tools/cmd/validate-eco-*`, `docs/spec/`, release docs, capsule examples.

**Approach:**

- [x] Stabilize Capsule manifest v1.
- [x] Stabilize dependency resolver for current local alpha graphs.
- [x] Stabilize permission model.
- [x] Stabilize semantic lockfile for current local alpha graphs.
- [x] Stabilize local Todex Vault for current local alpha flows.
- [x] Implement Seed import/export.
- [x] Implement NeedMap.
- [x] Implement TrustSnapshot.
- [x] Implement Materializer.
- [x] Add reproducible build basics.
- [x] Add API diff checker for the current alpha generated-doc metadata surface.
- [x] Add beta package publishing.
- [x] Add TetraHub beta path.
- [x] Add target-aware downloads.
- [x] Add trust metadata.
- [x] Keep full distributed Todex mesh, proof-carrying capsules, global EcoTrust, EcoOracle, and live evolution documented as post-v1.0 unless explicitly promoted.

**Verification:**

```sh
./tetra eco verify --target linux-x64 --lock reports/tetra.lock.json Tetra.capsule
./tetra eco pack --project Tetra.capsule -o reports/app.todex
./tetra eco unpack reports/app.todex -C reports/unpacked
./tetra eco vault verify --store .tetra/todex-vault
go test ./cli/... ./tools/... -run 'Eco|Vault|Capsule|Lock|API'
bash scripts/test_all.sh --full
```

**Done when:** Local Eco/Todex is stable and publishing is clearly labeled beta.

## Final v1.0 Release Gate

### TODO 17: Only Label v1.0 When the Real Gate Passes

**Status:** Blocked by design. Lorentz verified the v1 gate correctly refuses release labeling while version is `v0.6.0` and mandatory v1 capabilities are missing.

**Goal:** Prevent accidental release labeling.

**Files:** `compiler/internal/version/version.go`, `docs/generated/manifest.json`, `docs/checklists/v1_0_release_gate.md`, `docs/release_notes_v1_0_draft.md`, `scripts/release_v1_0_gate.sh`.

**Approach:**

- [x] Keep version at `v0.6.x` or later pre-1.0 marker until mandatory checks pass.
- [x] Update version only when the release branch is actually ready.
- [x] Regenerate and validate docs manifest.
- [x] Finalize release notes.
- [x] Check every item in `docs/checklists/v1_0_release_gate.md`.
- [x] Ensure `scripts/release_v1_0_gate.sh` blocks placeholder release state.
- [x] Run native host smoke.
- [x] Run build-only smoke for all mandatory native and WASM targets.
- [x] Run WASI smoke in a WASI runner.
- [x] Run web UI smoke through browser automation.
- [x] Verify docs manifest and doctests.
- [x] Verify API diff reports.
- [x] Verify reproducible builds for at least one native and one WASM target.

**Verification:**

```sh
go test ./compiler/... ./cli/... ./tools/...
bash scripts/test_all.sh --full
bash scripts/release_v1_0_gate.sh
```

**Done when:** The release branch passes both required commands and all generated docs/release artifacts are current.

## Suggested Execution Order

- [x] 1. Freeze historical green v0.6.0 baseline.
- [x] 2. Finish or explicitly split the v0.6.x stabilization tasks.
- [x] 3. Validate the first v0.7 hardening slice.
- [x] 4. Start v1.0 Wave 1: Flow-only frontend.
- [x] 5. Do type system stabilization before ownership/race freedom.
- [x] 6. Do ownership/race freedom before claiming safe-code guarantees.
- [x] 7. Add WASM before UI web release checks.
- [x] 8. Stabilize stdlib and tooling before final release notes.
- [x] 9. Run the final v1.0 gate only after every placeholder has a real implementation.

## Standing Verification Commands

Use these throughout the project:

```sh
git diff --check
go test ./compiler/... ./cli/... ./tools/...
bash scripts/test_all.sh --quick
bash scripts/test_all.sh --full
bash scripts/release_v0_6_gate.sh
bash scripts/release_v1_0_gate.sh
```

## Open Investigation Tasks

- [x] Decide whether v0.7 should become an official intermediate release or remain an internal hardening slice.
- [x] Decide the final status of closures, semantic clauses, budget clauses, privacy clauses, and UI syntax before starting their implementation.
- [x] Decide whether Flow gets a native parser or continues through normalization during migration.
- [x] Decide the exact WASM object/runtime architecture before changing `compiler/target/target.go`.
- [x] Decide the stable stdlib module list before promoting `lib/experimental/` code.
- [x] Decide the API diff format before making it a release gate.
