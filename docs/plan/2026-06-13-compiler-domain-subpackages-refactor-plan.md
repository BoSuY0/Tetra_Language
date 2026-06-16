# План: compiler domain subpackages + facade refactor

**Status:** planning document, not implementation evidence.  
**Date:** 2026-06-13.  
**Owner:** compiler architecture refactor.  
**Requested by:** user request to turn the current focused-file split into more
real directories around `compiler/internal/semantics`,
`compiler/internal/lower`, and root `compiler/`.

## 1. Goal

The goal is to continue the completed 1400-line refactor by turning selected
large same-package clusters into real domain-owned directories.

The previous refactor successfully reduced every `.go` and `.tetra` source file
to at most 1400 physical lines. It mostly did that with same-package focused
files because that was the safest way to preserve behavior.

This plan is the next architectural step:

- keep public behavior and public import paths stable;
- create more real directories, not only more files;
- avoid Go import cycles;
- preserve package-level compiler behavior through facade packages;
- move domain logic gradually into subpackages with clear dependency direction;
- keep every moved or new `.go` file below 1400 lines;
- prove every slice with focused package tests and a final broad gate.

The end state should make these areas easier to navigate:

- semantic checking;
- lowering;
- root compiler build orchestration;
- report generation;
- runtime/build capability selection;
- link-object and native-target handling.

## 2. Non-goals

This plan does not propose a behavior rewrite.

Out of scope:

- changing the Tetra language semantics;
- changing generated IR or PLIR behavior intentionally;
- changing public `compiler` package APIs unless an explicit compatibility plan
  is approved;
- renaming public commands;
- moving CLI/tools/test suites unrelated to the three requested areas;
- optimizing performance as a primary objective;
- deleting existing dirty worktree changes;
- claiming remote CI, release, packaging, security, or performance readiness.

## 3. Observed current facts

These facts were inspected from the repo before writing this plan.

### 3.1 Existing architecture guidance

`docs/architecture/project_structure.md` says the repository is moving from
flat oversized package directories toward domain-owned directories with small
migration slices.

Important existing migration rules:

- move helpers first, then move test groups;
- do not move package-local Go tests into subdirectories until shared helpers
  exist in the relevant `testkit`;
- keep compatibility wrappers for renamed scripts until references migrate;
- each migration slice needs a guard, focused test, relevant package test,
  hygiene check, and Graphify update when code changes.

### 3.2 Current package sizes

Current inspected package/file counts:

```text
compiler/internal/semantics  54 Go files
compiler/internal/lower      39 Go files
compiler/                    117 Go files
```

`go list` reports:

```text
tetra_language/compiler/internal/semantics  package semantics  40 GoFiles 14 TestGoFiles
tetra_language/compiler/internal/lower      package lower      18 GoFiles 21 TestGoFiles
tetra_language/compiler                     package compiler   55 GoFiles 56 TestGoFiles
```

The difference between `find` and `go list` counts is expected because `go list`
counts package files after build constraints and package selection.

### 3.3 Previous refactor result

The completed project-wide refactor recorded:

```text
Final source files counted: 1958
Files above 1400 lines:    0
```

The previous structure ledger is:

```text
.workflow/project-wide-refactor-1400/verification/structure-ledger.md
```

That ledger shows the current state is still mostly same-package focused files.
This plan intentionally moves beyond that.

### 3.4 Why a direct file move is not safe

In Go, files in a subdirectory are a different package/import path. Therefore
this is not a valid mechanical refactor:

```text
compiler/internal/semantics/checker_world_globals.go
  -> compiler/internal/semantics/checker/world_globals.go
```

If the file remains `package semantics`, Go will not treat it as part of the
parent package because it is in a different directory. If it becomes
`package checker`, it loses access to unexported parent-package types and
helpers.

The same applies to:

- `compiler/internal/lower/lower_*.go`;
- `compiler/internal/lower/callable_*.go`;
- `compiler/compiler_*.go`;
- `compiler/reports_*.go`.

So the correct target is not "move files into folders". The correct target is
"extract real packages with explicit APIs and keep facade code in the old
package".

## 4. Current pain points by area

### 4.1 `compiler/internal/semantics`

Current focused files include:

```text
checker.go
checker_world_globals.go
checker_stmt_return.go
checker_entry_helpers.go
checker_declarations.go
checker_policy.go
checker_abi_secret_protocol.go
checker_analysis_flow.go
checker_resource_tracking.go
checker_resource_sources.go
checker_function_types_a.go
checker_function_types_b.go
checker_stmts.go
checker_locals.go
exprs.go
exprs_callbacks_typed.go
exprs_calls.go
exprs_ownership.go
exprs_resources_actors.go
function_types.go
function_types_enum_payload.go
generics.go
generics_clone_types.go
generics_monomorphize.go
region.go
region_tree_summary.go
```

The issue is not only file length. The issue is that one package owns many
domains:

- public semantic facade: `Check`, `CheckWorld`, `CheckWorldOpt`;
- model/result types such as `CheckedProgram`, `FuncSig`, `TypeInfo`,
  `GlobalInfo`, `LocalInfo`;
- declaration collection;
- expression checking;
- statement checking;
- function-type validation;
- generic monomorphization;
- ABI/privacy/protocol checks;
- resource ownership tracking;
- flow/secret/surface-frame analysis;
- region summaries.

The biggest architectural risk is package cycles. A child package under
`compiler/internal/semantics/...` cannot import parent package `semantics` if
the parent imports the child.

Therefore semantic extraction needs a shared model package before deeper domain
packages can exist.

### 4.2 `compiler/internal/lower`

Current focused files include:

```text
lower.go
lower_types_tasks.go
lower_stmts.go
lower_lets_match.go
lower_expr.go
lower_expr_calls.go
lower_lvalues_copy.go
lower_rangeproof.go
lower_constructors.go
callable_target_edges.go
callable_targets.go
callable_lowering.go
```

The central coupling point is the unexported `lowerer` type. Many files define
methods on `*lowerer`, so direct extraction into subpackages is not mechanical.

The target should be:

- keep the orchestration receiver in package `lower`;
- extract pure decision logic into subpackages;
- leave thin adapter methods in `lower` where stateful access to `lowerer` is
  required;
- introduce shared DTO/model packages only when needed.

### 4.3 root `compiler`

Current focused files include:

```text
compiler.go
compiler_native_link.go
compiler_runtime_caps.go
compiler_wasm_ui.go
compiler_link_objects.go
compiler_actor_usage.go
compiler_runtime_usage.go
compiler_actor_dispatch.go
reports_emit.go
reports_layout_perf.go
reports_backend.go
reports_bounds.go
reports_actor_helpers.go
abi_suite.go
abi_suite_classifiers.go
abi_suite_ffi.go
abi_suite_runtime_smoke.go
abi_suite_x64_runtime.go
abi_suite_runtime_boundaries.go
```

This is the best first area for real subpackages because the public root
`compiler` package can remain a facade while internal implementation moves
behind it.

Root `compiler` currently owns several domains:

- public build API;
- build options/stats;
- module build planning;
- native backend selection;
- runtime capability selection;
- link-object loading/validation;
- actor/runtime usage analysis;
- WASM/UI object building;
- report envelope/type generation;
- ABI suite and runtime smoke helpers.

## 5. Proposed target architecture

The target uses three principles.

1. Old public packages remain facade packages.
2. New domain packages live under `compiler/internal/...`.
3. Shared types move into model packages before behavior moves.

### 5.1 Target root overview

Proposed target structure:

```text
compiler/
  compiler.go
  facade_build.go
  facade_reports.go
  facade_abi.go
  compatibility_stability_v1.go
  runtime_hardening_v1.go
  *_test.go

  internal/
    buildapi/
      options.go
      stats.go
      targets.go
      objects.go

    buildplan/
      module_plan.go
      module_jobs.go
      cache_inputs.go

    buildruntime/
      capabilities.go
      runtime_usage.go
      actor_usage.go
      actor_dispatch.go

    buildlink/
      link_objects.go
      native_link.go
      object_validation.go

    buildwasm/
      wasm_ui.go
      wasm_runtime_policy.go

    buildreports/
      envelope.go
      emit.go
      layout_perf.go
      backend.go
      bounds.go
      actor_helpers.go

    abisuite/
      checks.go
      classifiers.go
      ffi.go
      runtime_smoke.go
      x64_runtime.go
      runtime_boundaries.go

    semantics/
      facade.go
      aliases.go
      checker.go
      model/
        checked_program.go
        types.go
        funcs.go
        globals.go
        locals.go
        resources.go
      checkerworld/
        world.go
        imports.go
        globals.go
      checkerdecl/
        structs.go
        enums.go
        actors.go
        protocols.go
      checkerflow/
        analysis.go
        secret_taint.go
        surface_frames.go
      checkerpolicy/
        clauses.go
        privacy.go
        abi_secret_protocol.go
      checkerresources/
        tracking.go
        sources.go
        ownership.go
      checkerfuncs/
        function_types.go
        enum_payload.go
        callable_fields.go
      checkerstmts/
        statements.go
        returns.go
        locals.go
      checkerexprs/
        exprs.go
        calls.go
        ownership.go
        callbacks_typed.go
        resources_actors.go
      generics/
        monomorphize.go
        clone_types.go
      regions/
        region.go
        tree_summary.go

    lower/
      facade.go
      lowerer.go
      adapters.go
      model/
        options.go
        locals.go
        policy.go
        loops.go
      stmts/
        block.go
        statement_kinds.go
      exprs/
        expr.go
        calls.go
        lvalues.go
        copy.go
      lets/
        let_copy.go
        match.go
        raw_offset.go
      constructors/
        structs.go
        enums.go
        function_fields.go
      rangeproof/
        proofs.go
        metadata.go
        conditions.go
      callables/
        target_edges.go
        targets.go
        lowering.go
      tasks/
        typed_tasks.go
        joins.go
      cleanup/
        defer_frames.go
        zeroing.go
```

Important: this tree is the architectural target, not a single patch. Some
directories may be introduced later if dependency analysis shows they should be
merged or renamed.

## 6. Dependency direction rules

These rules are the core of the plan.

### 6.1 Root compiler packages

Allowed direction:

```text
compiler
  -> compiler/internal/buildapi
  -> compiler/internal/buildplan
  -> compiler/internal/buildruntime
  -> compiler/internal/buildlink
  -> compiler/internal/buildwasm
  -> compiler/internal/buildreports
  -> compiler/internal/abisuite
```

Internal packages may import lower-level compiler internals such as:

```text
compiler/internal/frontend
compiler/internal/module
compiler/internal/semantics
compiler/internal/lower
compiler/internal/plir
compiler/internal/ir
compiler/internal/validation
compiler/internal/backend/...
compiler/target
```

Internal packages must not import parent package `compiler`.

If a type is currently in `compiler` and needed by an internal package, choose
one of these:

1. move it to `compiler/internal/buildapi`;
2. re-export it from `compiler` via type alias;
3. pass only the fields needed through a small DTO;
4. keep that domain in `compiler` until a safer seam exists.

Preferred compatibility pattern:

```go
// in compiler package
type BuildOptions = buildapi.BuildOptions
type BuildStats = buildapi.BuildStats
```

Use aliases only after verifying docs/tests do not rely on reflection of package
paths or type names in unexpected ways.

### 6.2 Semantics packages

Allowed direction:

```text
semantics
  -> semantics/model
  -> semantics/checkerworld
  -> semantics/checkerdecl
  -> semantics/checkerflow
  -> semantics/checkerpolicy
  -> semantics/checkerresources
  -> semantics/checkerfuncs
  -> semantics/checkerstmts
  -> semantics/checkerexprs
  -> semantics/generics
  -> semantics/regions
```

Domain packages may import:

```text
compiler/internal/frontend
compiler/internal/module
compiler/internal/semantics/model
compiler/internal/t4iface
```

Domain packages must not import parent package:

```text
compiler/internal/semantics
```

If a domain package needs a type currently in `semantics`, move that type to
`semantics/model` first and leave a parent alias:

```go
// in semantics package
type CheckedProgram = model.CheckedProgram
type TypeInfo = model.TypeInfo
type FuncSig = model.FuncSig
```

Do not start by moving checker behavior. Start by moving model types.

### 6.3 Lowering packages

Allowed direction:

```text
lower
  -> lower/model
  -> lower/rangeproof
  -> lower/callables
  -> lower/constructors
  -> lower/stmts
  -> lower/exprs
  -> lower/lets
  -> lower/tasks
  -> lower/cleanup
```

Because `lowerer` is stateful and currently unexported, the first extraction
must avoid moving receiver methods that deeply mutate `lowerer`.

Preferred initial pattern:

```text
lower/lower_rangeproof.go
  remains in package lower as adapter methods

lower/rangeproof/
  contains pure proof detection, metadata merge helpers, and condition
  classifiers that do not need lowerer internals
```

If a helper needs many `lowerer` fields, leave it in package `lower` until a
small interface can be introduced.

Do not export `lowerer` casually. Exporting it would turn a local implementation
detail into a cross-package contract.

## 7. Public compatibility strategy

### 7.1 Public import paths that should remain stable

These import paths must remain valid:

```text
tetra_language/compiler
tetra_language/compiler/internal/semantics
tetra_language/compiler/internal/lower
```

The plan may add imports such as:

```text
tetra_language/compiler/internal/buildreports
tetra_language/compiler/internal/semantics/model
tetra_language/compiler/internal/semantics/checkerflow
tetra_language/compiler/internal/lower/rangeproof
```

But existing callers should not be forced to import them unless they are internal
compiler tests or explicitly migrated implementation code.

### 7.2 Facade rule

Every old package keeps the entrypoints that existing code uses.

Root `compiler` keeps:

```text
BuildFile
BuildFileWithStats
Build
BuildOptions
BuildStats
EmitMode
RuntimeMode
```

`semantics` keeps:

```text
Check
CheckWorld
CheckWorldOpt
CheckOptions
CheckedProgram
TypeInfo
FuncSig
LocalInfo
GlobalInfo
```

`lower` keeps the public lowering entrypoints already used by root compiler and
tests. Exact entrypoint names must be confirmed before implementation.

### 7.3 Alias rule

When moving types, prefer type aliases in the old package for compatibility.

Example:

```go
package semantics

import semmodel "tetra_language/compiler/internal/semantics/model"

type CheckedProgram = semmodel.CheckedProgram
type TypeInfo = semmodel.TypeInfo
```

This preserves assignability and avoids a broad API churn wave.

### 7.4 Adapter rule

When moving behavior, leave a thin old-package adapter first.

Example shape:

```go
func CheckWorldOpt(world *module.World, opt CheckOptions) (*CheckedProgram, error) {
    return checkerworld.CheckWorldOpt(world, opt)
}
```

This is only valid after the moved package owns all required model types without
importing parent `semantics`.

## 8. Proposed migration phases

Each phase should be small enough to revert without losing the whole refactor.

### Phase 0: Baseline and guard setup

**Goal:** Capture the current dependency and behavior baseline before moving
anything.

**Files to inspect:**

```text
compiler/compiler.go
compiler/compiler_*.go
compiler/reports*.go
compiler/abi_suite*.go
compiler/internal/semantics/*.go
compiler/internal/lower/*.go
docs/architecture/project_structure.md
.workflow/project-wide-refactor-1400/verification/structure-ledger.md
```

**Commands:**

```sh
go list -deps ./compiler/internal/semantics ./compiler/internal/lower ./compiler
go list -f '{{.ImportPath}} {{.Name}} {{.GoFiles}} {{.TestGoFiles}}' ./compiler/internal/semantics ./compiler/internal/lower ./compiler
rg --files -g '*.go' -g '*.tetra' | while IFS= read -r file; do wc -l "$file"; done | sort -nr | head -40
```

**Verification:**

```sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/internal/semantics ./compiler/internal/lower ./compiler -count=1
git diff --check
```

**Done when:**

- baseline package tests are known;
- current line counts are recorded;
- package dependency direction is documented;
- no code movement has started without evidence.

**Notes:**

- If this baseline fails, stop and debug before any package extraction.
- Clean cache after evidence:

```sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages go clean -cache
rm -rf .cache/go-tmp-domain-subpackages
```

### Phase 1: Root `compiler` model/API extraction

**Goal:** Prepare root `compiler` for subpackages by extracting stable API/model
types first.

**Target structure:**

```text
compiler/internal/buildapi/
  options.go
  stats.go
  targets.go
  objects.go
```

**Candidate moves:**

From `compiler/compiler.go`:

```text
EmitMode
RuntimeMode
BuildOptions
BuildStats
linkedObject
nativeCodegenFunc
nativeExecutableBackend
nativeBuildTarget
checkedBuildWorld
moduleBuildJob
moduleBuildPlan
```

Only move a type if it does not create import cycles. Public types should be
re-exported as aliases from `compiler`.

**Expected facade shape:**

```go
package compiler

import buildapi "tetra_language/compiler/internal/buildapi"

type BuildOptions = buildapi.BuildOptions
type BuildStats = buildapi.BuildStats
type EmitMode = buildapi.EmitMode
type RuntimeMode = buildapi.RuntimeMode
```

**Verification:**

```sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler -run '^$' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler -run 'Test(Pipeline|BuildWASM|BuildNativeUI|BuildCache|LinkObject|BuildLinks|BuildRejectsInterface|BuildInterfaceOnly)' -count=1
git diff --check
```

**Done when:**

- root package compiles;
- public API remains source-compatible;
- no internal package imports parent `compiler`;
- all new files stay under 1400 lines.

**Risks:**

- constants using `iota` need alias-compatible handling;
- tests may compare string output that includes type names;
- internal helper types may be better left in `compiler` until later.

### Phase 2: Root build domain packages

**Goal:** Move implementation domains out of root `compiler/` while preserving
root facade functions.

**Target structure:**

```text
compiler/internal/buildplan/
  module_plan.go
  module_jobs.go
  cache_inputs.go

compiler/internal/buildruntime/
  capabilities.go
  runtime_usage.go
  actor_usage.go
  actor_dispatch.go

compiler/internal/buildlink/
  link_objects.go
  native_link.go
  object_validation.go

compiler/internal/buildwasm/
  wasm_ui.go
  wasm_runtime_policy.go
```

**Candidate source files:**

```text
compiler/compiler.go
compiler/compiler_native_link.go
compiler/compiler_runtime_caps.go
compiler/compiler_wasm_ui.go
compiler/compiler_link_objects.go
compiler/compiler_actor_usage.go
compiler/compiler_runtime_usage.go
compiler/compiler_actor_dispatch.go
```

**Approach:**

1. Extract pure helper groups first.
2. Keep root `BuildFile`, `BuildFileWithStats`, and the main build pipeline as
   facade/orchestrator until all dependencies are clean.
3. Move one domain at a time:
   - runtime capabilities;
   - actor usage;
   - link objects;
   - native link;
   - WASM/UI object helpers;
   - build planning.
4. Use small DTOs from `buildapi` to avoid importing parent `compiler`.

**Verification per domain:**

```sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler -run 'Runtime|Actor|Link|Build|WASM|Interface' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler -count=1
git diff --check
```

**Done when:**

- root `compiler/` no longer owns most build-domain implementation details;
- root files become facade/orchestration files;
- new packages have no import cycle;
- root tests pass.

### Phase 3: Reports package extraction

**Goal:** Move report construction out of root `compiler` into a dedicated
report package.

**Target structure:**

```text
compiler/internal/buildreports/
  envelope.go
  emit.go
  layout_perf.go
  backend.go
  bounds.go
  actor_helpers.go
```

**Candidate source files:**

```text
compiler/reports.go
compiler/reports_emit.go
compiler/reports_layout_perf.go
compiler/reports_backend.go
compiler/reports_bounds.go
compiler/reports_actor_helpers.go
```

**Approach:**

1. Move private report structs to `buildreports`.
2. Export only the report builder functions needed by root `compiler`.
3. Keep root functions as wrappers if tests or docs reference old function
   names.
4. Keep JSON schema version behavior unchanged.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler -run 'ExplainReports|BoundsReport|AllocReport|MemoryReport|BackendReport|Performance|P24Compatibility' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler -count=1
git diff --check
```

**Done when:**

- report code lives in `compiler/internal/buildreports`;
- root compiler keeps only wrappers/facade glue;
- schema/version witness tests still pass.

### Phase 4: ABI suite extraction

**Goal:** Move ABI and runtime smoke helper implementation out of root
`compiler`.

**Target structure:**

```text
compiler/internal/abisuite/
  checks.go
  classifiers.go
  ffi.go
  runtime_smoke.go
  x64_runtime.go
  runtime_boundaries.go
```

**Candidate source files:**

```text
compiler/abi_suite.go
compiler/abi_suite_classifiers.go
compiler/abi_suite_ffi.go
compiler/abi_suite_runtime_smoke.go
compiler/abi_suite_x64_runtime.go
compiler/abi_suite_runtime_boundaries.go
```

**Approach:**

1. Move private ABI helper types first.
2. Preserve root-facing API through wrappers or aliases.
3. Keep test fixtures unchanged unless imports force updates.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler -run 'ABI|ABICheck|RunTargetABIChecks|RuntimeSmoke|Boundary' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler -count=1
git diff --check
```

**Done when:**

- ABI suite implementation is in `compiler/internal/abisuite`;
- root `compiler` retains compatibility surface;
- ABI-focused tests pass.

### Phase 5: Semantics model extraction

**Goal:** Create an acyclic foundation for semantic subpackages.

**Target structure:**

```text
compiler/internal/semantics/model/
  checked_program.go
  types.go
  funcs.go
  globals.go
  locals.go
  resources.go
```

**Candidate moved types:**

Exact locations must be confirmed during implementation, but likely candidates
include:

```text
CheckedProgram
TypeInfo
FuncSig
LocalInfo
GlobalInfo
ActorStateField
FunctionFieldInfo
EnumCaseInfo
Resource facts used across checker files
```

**Approach:**

1. Move data-only types first.
2. Keep aliases in parent `semantics`.
3. Do not move behavior in this phase.
4. Fix imports inside `semantics` package to use `model` only where needed.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/internal/semantics -run '^$' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/internal/semantics -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/tests/semantics ./compiler/tests/callables ./compiler/tests/ownership ./compiler/tests/safety -count=1
git diff --check
```

**Done when:**

- model package exists;
- parent semantic aliases compile;
- no semantics child package imports parent `semantics`;
- semantics-related tests pass.

**Risk:**

This is the highest leverage and highest-risk step. If a type has methods that
depend on many parent-package helpers, move only the data type and leave methods
temporarily in parent package as functions.

### Phase 6: Semantics checker domain extraction

**Goal:** Turn focused checker files into real checker domain packages.

**Target structure:**

```text
compiler/internal/semantics/checkerworld/
  world.go
  imports.go
  globals.go

compiler/internal/semantics/checkerdecl/
  structs.go
  enums.go
  actors.go
  protocols.go

compiler/internal/semantics/checkerpolicy/
  clauses.go
  privacy.go
  abi_secret_protocol.go

compiler/internal/semantics/checkerflow/
  analysis.go
  secret_taint.go
  surface_frames.go

compiler/internal/semantics/checkerresources/
  tracking.go
  sources.go
  ownership.go

compiler/internal/semantics/checkerfuncs/
  function_types.go
  enum_payload.go
  callable_fields.go

compiler/internal/semantics/checkerstmts/
  statements.go
  returns.go
  locals.go

compiler/internal/semantics/checkerexprs/
  exprs.go
  calls.go
  ownership.go
  callbacks_typed.go
  resources_actors.go
```

**Candidate source files:**

```text
compiler/internal/semantics/checker_world_globals.go
compiler/internal/semantics/checker_declarations.go
compiler/internal/semantics/checker_policy.go
compiler/internal/semantics/checker_abi_secret_protocol.go
compiler/internal/semantics/checker_analysis_flow.go
compiler/internal/semantics/checker_resource_tracking.go
compiler/internal/semantics/checker_resource_sources.go
compiler/internal/semantics/checker_function_types_a.go
compiler/internal/semantics/checker_function_types_b.go
compiler/internal/semantics/checker_stmts.go
compiler/internal/semantics/checker_stmt_return.go
compiler/internal/semantics/checker_locals.go
compiler/internal/semantics/exprs*.go
```

**Approach:**

Do not move all checker files at once.

Recommended order:

1. `checkerflow`: move state structs and pure snapshot/merge helpers.
2. `checkerresources`: move resource source classification helpers.
3. `checkerpolicy`: move clause/policy validation helpers.
4. `checkerfuncs`: move function-type validators once model types are stable.
5. `checkerdecl`: move declaration collection and duplicate validation helpers.
6. `checkerworld`: move world-level orchestration only after the previous
   packages are stable.
7. `checkerstmts` and `checkerexprs`: move last, because they usually touch the
   broadest surface.

**Facade rule:**

Parent `semantics.CheckWorldOpt` should remain the canonical entrypoint until
all internal packages stabilize.

**Verification per slice:**

```sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/internal/semantics -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/tests/semantics -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/tests/callables ./compiler/tests/ownership ./compiler/tests/safety -count=1
git diff --check
```

**Done when:**

- checker logic is grouped by real packages;
- parent `semantics` remains the compatibility facade;
- no import cycles exist;
- all semantics-facing package tests pass.

### Phase 7: Semantics generics and regions extraction

**Goal:** Move non-checker semantic domains into separate packages after checker
model extraction is stable.

**Target structure:**

```text
compiler/internal/semantics/generics/
  monomorphize.go
  clone_types.go

compiler/internal/semantics/regions/
  region.go
  tree_summary.go
```

**Candidate source files:**

```text
compiler/internal/semantics/generics.go
compiler/internal/semantics/generics_monomorphize.go
compiler/internal/semantics/generics_clone_types.go
compiler/internal/semantics/region.go
compiler/internal/semantics/region_tree_summary.go
```

**Approach:**

1. Move pure generic clone/monomorphization helpers first.
2. Move region summary helpers after model types are stable.
3. Leave parent wrappers if existing tests call package-private helpers.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/internal/semantics -run 'Generics|Region|Ownership|Surface' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/tests/semantics ./compiler/tests/ownership -count=1
git diff --check
```

**Done when:**

- generics and regions are real packages;
- parent semantics facade remains stable;
- region/ownership/generic tests pass.

### Phase 8: Lowering model extraction

**Goal:** Prepare `lower` for real subpackages without exposing the whole
`lowerer` implementation.

**Target structure:**

```text
compiler/internal/lower/model/
  options.go
  locals.go
  policy.go
  loops.go
```

**Candidate moved types:**

```text
Options
runtimePolicy
budgetCharge
scalarSliceLocal
whileRangeProof
rawPtrOffsetLocal
typedTaskWrapper
typedTaskStagedTarget
inoutReturnLocal
inoutWriteback
loopLabels
deferFrame
```

**Approach:**

1. Move data-only types with aliases where needed.
2. Keep `lowerer` in parent package initially.
3. Do not move receiver methods in this phase.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/internal/lower -run '^$' -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/internal/lower -count=1
git diff --check
```

**Done when:**

- `lower/model` exists;
- parent `lower.Options` remains source-compatible;
- lower package tests pass.

### Phase 9: Lowering domain extraction

**Goal:** Move lower-domain pure logic into subpackages while leaving stateful
adapter methods in package `lower`.

**Target structure:**

```text
compiler/internal/lower/rangeproof/
  proofs.go
  metadata.go
  conditions.go

compiler/internal/lower/callables/
  target_edges.go
  targets.go
  lowering.go

compiler/internal/lower/constructors/
  structs.go
  enums.go
  function_fields.go

compiler/internal/lower/stmts/
  block.go
  statement_kinds.go

compiler/internal/lower/exprs/
  expr.go
  calls.go
  lvalues.go
  copy.go

compiler/internal/lower/lets/
  let_copy.go
  match.go
  raw_offset.go

compiler/internal/lower/tasks/
  typed_tasks.go
  joins.go

compiler/internal/lower/cleanup/
  defer_frames.go
  zeroing.go
```

**Recommended order:**

1. `rangeproof`: likely best first target because many helpers classify
   conditions/ranges.
2. `callables`: extract target edge structs and pure target graph building.
3. `constructors`: extract constructor resolution helpers.
4. `tasks`: extract typed task staging helpers.
5. `lets` and `exprs`: move later because they touch many `lowerer` fields.
6. `stmts` and `cleanup`: move last because they are orchestration-heavy.

**Adapter pattern:**

Parent package keeps methods like:

```go
func (l *lowerer) whileRangeProof(stmt *frontend.WhileStmt) (model.WhileRangeProof, bool) {
    return rangeproof.While(stmt, rangeproof.Context{...})
}
```

The subpackage should receive explicit inputs, not the whole `*lowerer`, unless
a narrow interface is introduced and tested.

**Verification per slice:**

```sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/internal/lower -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/tests/lowering ./compiler/tests/semantics -count=1
git diff --check
```

**Done when:**

- lower has real domain subpackages;
- parent package is mostly orchestration/adapters;
- no subpackage imports parent `lower`;
- lower and lowering-related tests pass.

### Phase 10: Test relocation and testkit cleanup

**Goal:** Move package-local tests only after implementation package boundaries
are real and shared helpers exist.

**Target structure:**

```text
compiler/internal/testkit/
  buildrun/
  fixtures/
  assertions/

compiler/tests/
  backend/
  callables/
  frontend/
  lowering/
  semantics/
  compiler/
```

**Approach:**

1. Audit package-local tests that only need public APIs.
2. Move helper functions into `compiler/internal/testkit`.
3. Move behavior tests into `compiler/tests/<domain>`.
4. Keep white-box tests in package directories only when they truly require
   unexported internals.

**Verification:**

```sh
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/internal/... ./compiler/tests/... ./compiler -count=1
git diff --check
```

**Done when:**

- package-local tests are not moved prematurely;
- new test directories use shared testkit helpers;
- no duplicate fixture helper piles appear.

### Phase 11: Final integration gate

**Goal:** Prove the directory refactor works end-to-end across affected compiler
surfaces.

**Required commands:**

```sh
rg --files -g '*.go' -g '*.tetra' | while IFS= read -r file; do wc -l "$file"; done | awk '{print $1 "\t" substr($0, index($0,$2))}' | sort -nr > reports/domain-subpackages-line-counts.tsv
awk '$1 > 1400 { print }' reports/domain-subpackages-line-counts.tsv > reports/domain-subpackages-over-1400.tsv
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/internal/semantics ./compiler/internal/lower ./compiler -count=1
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages GOTMPDIR=$(pwd)/.cache/go-tmp-domain-subpackages go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1
git diff --check
graphify update .
GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-domain-subpackages go clean -cache
rm -rf .cache/go-tmp-domain-subpackages
```

Use a `.workflow/<slug>/verification/` directory instead of `reports/` if this
is executed as a sustained goal.

**Done when:**

- zero `.go`/`.tetra` files above 1400 lines;
- focused compiler packages pass;
- broad `./compiler/... ./cli/... ./tools/...` gate passes;
- `git diff --check` passes;
- `graphify update .` passes after code changes;
- final report maps every acceptance criterion to evidence.

## 9. Detailed desired end-state by original user area

### 9.1 Requested area: `compiler/internal/semantics`

Current user-highlighted files:

```text
checker.go
checker_world_globals.go
checker_stmt_return.go
checker_entry_helpers.go
checker_declarations.go
checker_policy.go
checker_abi_secret_protocol.go
checker_analysis_flow.go
checker_resource_tracking.go
checker_resource_sources.go
checker_function_types_a.go
checker_function_types_b.go
checker_stmts.go
checker_locals.go
```

Desired final directory shape:

```text
compiler/internal/semantics/
  facade.go
  aliases.go
  checker.go

  model/
    checked_program.go
    types.go
    funcs.go
    globals.go
    locals.go
    resources.go

  checkerworld/
    world.go
    imports.go
    globals.go

  checkerdecl/
    structs.go
    enums.go
    actors.go
    protocols.go

  checkerpolicy/
    clauses.go
    privacy.go
    abi_secret_protocol.go

  checkerflow/
    analysis.go
    secret_taint.go
    surface_frames.go

  checkerresources/
    tracking.go
    sources.go
    ownership.go

  checkerfuncs/
    function_types.go
    enum_payload.go
    callable_fields.go

  checkerstmts/
    statements.go
    returns.go
    locals.go
```

Mapping:

```text
checker_world_globals.go          -> checkerworld/globals.go
checker_stmt_return.go            -> checkerstmts/returns.go
checker_entry_helpers.go          -> checkerworld/world.go or model/helpers.go after inspection
checker_declarations.go           -> checkerdecl/*
checker_policy.go                 -> checkerpolicy/clauses.go and checkerpolicy/privacy.go
checker_abi_secret_protocol.go    -> checkerpolicy/abi_secret_protocol.go
checker_analysis_flow.go          -> checkerflow/*
checker_resource_tracking.go      -> checkerresources/tracking.go
checker_resource_sources.go       -> checkerresources/sources.go
checker_function_types_a.go        -> checkerfuncs/function_types.go
checker_function_types_b.go        -> checkerfuncs/enum_payload.go or checkerfuncs/callable_fields.go
checker_stmts.go                  -> checkerstmts/statements.go
checker_locals.go                 -> checkerstmts/locals.go
```

Some mappings are intentionally provisional because exact helper ownership must
be confirmed during implementation.

### 9.2 Requested area: `compiler/internal/lower`

Current user-highlighted files:

```text
lower.go
lower_types_tasks.go
lower_stmts.go
lower_lets_match.go
lower_expr.go
lower_expr_calls.go
lower_lvalues_copy.go
lower_rangeproof.go
lower_constructors.go
callable_target_edges.go
callable_targets.go
callable_lowering.go
```

Desired final directory shape:

```text
compiler/internal/lower/
  facade.go
  lowerer.go
  adapters.go

  model/
    options.go
    locals.go
    policy.go
    loops.go

  stmts/
    block.go
    statement_kinds.go

  exprs/
    expr.go
    calls.go
    lvalues.go
    copy.go

  lets/
    let_copy.go
    match.go
    raw_offset.go

  constructors/
    structs.go
    enums.go
    function_fields.go

  rangeproof/
    proofs.go
    metadata.go
    conditions.go

  callables/
    target_edges.go
    targets.go
    lowering.go

  tasks/
    typed_tasks.go
    joins.go

  cleanup/
    defer_frames.go
    zeroing.go
```

Mapping:

```text
lower_types_tasks.go        -> lowerer.go, model/*, tasks/*, cleanup/*
lower_stmts.go              -> stmts/*
lower_lets_match.go         -> lets/* plus constructors or expr adapters where needed
lower_expr.go               -> exprs/expr.go
lower_expr_calls.go         -> exprs/calls.go
lower_lvalues_copy.go       -> exprs/lvalues.go and exprs/copy.go
lower_rangeproof.go         -> rangeproof/*
lower_constructors.go       -> constructors/*
callable_target_edges.go    -> callables/target_edges.go
callable_targets.go         -> callables/targets.go
callable_lowering.go        -> callables/lowering.go
```

The first implementation should not move all receiver methods. It should move
pure helpers and leave adapters until smaller interfaces exist.

### 9.3 Requested area: root `compiler`

Current user-highlighted files:

```text
compiler.go
compiler_native_link.go
compiler_runtime_caps.go
compiler_wasm_ui.go
compiler_link_objects.go
compiler_actor_usage.go
compiler_runtime_usage.go
compiler_actor_dispatch.go
reports_emit.go
reports_layout_perf.go
reports_backend.go
reports_bounds.go
reports_actor_helpers.go
```

Desired final directory shape:

```text
compiler/
  compiler.go
  facade_build.go
  facade_reports.go
  facade_abi.go

compiler/internal/buildapi/
  options.go
  stats.go
  targets.go
  objects.go

compiler/internal/buildplan/
  module_plan.go
  module_jobs.go
  cache_inputs.go

compiler/internal/buildruntime/
  capabilities.go
  runtime_usage.go
  actor_usage.go
  actor_dispatch.go

compiler/internal/buildlink/
  link_objects.go
  native_link.go
  object_validation.go

compiler/internal/buildwasm/
  wasm_ui.go
  wasm_runtime_policy.go

compiler/internal/buildreports/
  envelope.go
  emit.go
  layout_perf.go
  backend.go
  bounds.go
  actor_helpers.go
```

Mapping:

```text
compiler.go                    -> facade_build.go + buildapi/* + buildplan/*
compiler_native_link.go         -> buildlink/native_link.go
compiler_runtime_caps.go        -> buildruntime/capabilities.go
compiler_wasm_ui.go             -> buildwasm/wasm_ui.go
compiler_link_objects.go        -> buildlink/link_objects.go
compiler_actor_usage.go         -> buildruntime/actor_usage.go
compiler_runtime_usage.go       -> buildruntime/runtime_usage.go
compiler_actor_dispatch.go      -> buildruntime/actor_dispatch.go
reports_emit.go                 -> buildreports/emit.go
reports_layout_perf.go          -> buildreports/layout_perf.go
reports_backend.go              -> buildreports/backend.go
reports_bounds.go               -> buildreports/bounds.go
reports_actor_helpers.go        -> buildreports/actor_helpers.go
```

Root `compiler` remains the public package. Internal packages are implementation
details.

## 10. Guardrails and stop rules

Stop immediately if:

- `go list` reports an import cycle;
- a moved package needs to import its parent package;
- a facade alias changes public API behavior unexpectedly;
- the same extraction attempt fails twice;
- line count grows above 1400 in any `.go` or `.tetra` file;
- broad tests fail for a reason not understood;
- the refactor starts changing compiler behavior instead of structure.

When stopped:

- record the blocker in the workflow kernel if running under `/goal`;
- keep status `PARTIAL`, not `DONE`;
- do not try a third variant without new evidence.

## 11. Suggested execution style

This refactor is broad and should not be one giant patch.

Recommended execution:

```text
1. Create .workflow/compiler-domain-subpackages/
2. Record baseline and acceptance criteria.
3. Execute one package/domain slice at a time.
4. After every slice:
   - run focused tests;
   - run git diff --check;
   - check line counts;
   - update structure ledger.
5. After all slices:
   - run broad compiler/cli/tools gate;
   - run graphify update .;
   - write final report.
```

Recommended slice order:

```text
1. compiler/internal/buildapi
2. compiler/internal/buildruntime
3. compiler/internal/buildlink
4. compiler/internal/buildreports
5. compiler/internal/abisuite
6. compiler/internal/semantics/model
7. compiler/internal/semantics/checkerflow
8. compiler/internal/semantics/checkerresources
9. compiler/internal/semantics/checkerpolicy
10. compiler/internal/semantics/checkerfuncs
11. compiler/internal/semantics/checkerdecl
12. compiler/internal/semantics/checkerworld
13. compiler/internal/semantics/checkerstmts/checkerexprs
14. compiler/internal/semantics/generics and regions
15. compiler/internal/lower/model
16. compiler/internal/lower/rangeproof
17. compiler/internal/lower/callables
18. compiler/internal/lower/constructors
19. compiler/internal/lower/tasks
20. compiler/internal/lower/exprs/lets/stmts cleanup
21. test relocation/testkit cleanup
22. final integration gate
```

## 12. Acceptance criteria

This refactor is complete only when all of these are true:

- root `compiler` remains a stable facade package;
- `compiler/internal/semantics` has real domain subpackages, not only many
  same-package files;
- `compiler/internal/lower` has real domain subpackages, with parent adapters
  where stateful lowering remains centralized;
- report/build/runtime/link/ABI implementation has moved out of root
  `compiler/` where safe;
- no package imports its parent package;
- no import cycles exist;
- no `.go` or `.tetra` source file exceeds 1400 lines;
- focused tests for every touched package pass;
- broad `go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1`
  passes;
- `git diff --check` passes;
- `graphify update .` is run after code changes;
- final documentation records the new structure and explicit nonclaims.

## 13. Evidence to produce during implementation

If this plan is executed as a goal, create:

```text
.workflow/compiler-domain-subpackages/
  GOAL.md
  PLAN.md
  ATTEMPTS.md
  NOTES.md
  CONTROL.md
  state.json
  verification/
    baseline-package-graph.md
    phase01-buildapi.md
    phase02-buildruntime-buildlink.md
    phase03-buildreports.md
    phase04-abisuite.md
    phase05-semantics-model.md
    phase06-semantics-checker-domains.md
    phase07-semantics-generics-regions.md
    phase08-lower-model.md
    phase09-lower-domains.md
    final-line-counts.md
    final-broad-go-test.log
    final-diff-check.log
    final-graphify-update.log
  final-report.md
```

For each phase, record:

```text
Acceptance criterion:
Changed files/directories:
Dependency graph result:
Focused command:
Exit status:
Line-count result:
Residual risk:
```

## 14. Open implementation questions

These must be answered during Phase 0 or the first relevant phase:

- Which root `compiler` public types can safely become aliases to
  `compiler/internal/buildapi`?
- Which semantic model types can move without changing JSON/report/test
  expectations?
- Which checker helpers are pure enough to move before the full model split?
- Which lower helpers can move without exporting `lowerer`?
- Should `compiler/internal/buildreports` own report structs only, or also
  report file writing?
- Should ABI suite remain partly root-level because tests treat it as compiler
  package internals?
- Which package-local tests can later move to `compiler/tests/*` after
  `compiler/internal/testkit` is stronger?

## 15. Recommendation

Start with root `compiler` extraction, not `semantics`.

Reason:

- root `compiler` has the clearest facade boundary;
- public API compatibility can be preserved with aliases/wrappers;
- build/report/runtime/link domains are easier to isolate than semantic checker
  state;
- early success creates the shared patterns needed for the harder semantics and
  lower extractions.

After root compiler extraction, move to `semantics/model`, then semantics
checker domains, then `lower/model`, then lowering domain packages.

The riskiest packages are:

```text
compiler/internal/semantics/checkerstmts
compiler/internal/semantics/checkerexprs
compiler/internal/lower/exprs
compiler/internal/lower/lets
compiler/internal/lower/stmts
```

Those should be last, because they are most likely to depend on broad package
state.

## 16. Implementation handoff

This plan is ready to execute with a workflow that can handle many small
verified slices.

Recommended modes:

1. `subagent-driven-development` for same-session execution with per-task review.
2. `executing-plans` for checkpointed batch execution.
3. `/goal` with a new workflow kernel if the refactor should run
   autonomously across long context windows.

Do not execute this as one broad mechanical move. The safe path is small
package-boundary extraction slices with tests after each one.
