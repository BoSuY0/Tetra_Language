# Global Architecture Refactor Plan

**Goal:** Refactor the whole Tetra_Language repository toward a maintainable
structure without changing public behavior unintentionally.

**Context:** The repository is a Go workspace with root, `compiler`, `cli`, and
`tools` modules. The worktree is already heavily dirty, so every slice must stay
scoped, preserve unrelated changes, and report exactly which files it touches.

**Execution:** Use small TDD/verification slices. After code changes, run
focused tests, relevant package tests, formatting checks, required validators,
and `graphify update .`.

## Architecture Inventory Evidence

- `graphify god_nodes` highlighted broad coupling around helpers such as
  `contains`, `writeFile`, `buildAndRun`, `runCLI`, and compiler diagnostic
  helpers.
- `graphify query` for global architecture refactor connected the current
  pressure points to split CLI command tests, `cli/cmd/tetra/main.go`,
  `compiler/internal/semantics/checker.go`, `compiler/internal/lower/lower.go`,
  `compiler/tests/callables/function_typed_callable_test.go`, split versioned release script tests,
  `tools/cmd/verify-docs`, and the existing full-project refactor plan.
- `find . -name go.mod` found module roots at `.`, `cli`, `compiler`, and
  `tools`.
- `go env GOMOD GOWORK` confirmed the active root module and `go.work`.
- `GOCACHE=/tmp/tetra-language-go-cache go list ./...` from the repository root
  lists only the root `tetra_language` package; package-specific verification
  must run inside or through the workspace module paths.
- `find ... -name '*.go' | xargs wc -l` identified the largest Go files and test
  files.
- `git status --short` showed extensive pre-existing modified and untracked
  files; refactor slices must not revert unrelated work.
- `docs/architecture/project_structure.md` now records the directory-based
  target structure for compiler, CLI, tools, scripts, docs, examples, and test
  suites.

## Current Hotspot Map

| Area | Evidence | Risk | First safe move |
| --- | --- | --- | --- |
| CLI tests | Historical monolithic CLI command tests were about 14.4k lines before being split across `cli/cmd/tetra/*_test.go`. | Mixed command coverage, shared fixtures, brittle navigation. | Add source-structure guards, then split cohesive command test groups without changing test logic. |
| Function-typed callable tests | `compiler/tests/callables/function_typed_callable_test.go` started from the historical monolithic callable suite and is now 6932 lines. | Large scenario matrix hides subdomain ownership. | Continue splitting by callable domain: direct symbols, captures, struct/enum payloads, cross-module/interface metadata. |
| Semantic checker | `compiler/internal/semantics/checker.go` is about 11.6k lines. | Central semantic state and diagnostics are tightly coupled. | Same-package extraction of pure helpers only after dependency/impact checks and focused compiler tests. |
| Safety and ownership tests | `compiler/tests/safety/safety_diagnostics_test.go` and `compiler/tests/ownership/ownership_test.go` exceed 7.5k and 6.2k lines. | Broad diagnostic/evidence matrices are hard to review. | Extract fixtures and table groups; preserve diagnostic text and JSON evidence. |
| Lowering | `compiler/internal/lower/lower.go` remains about 4.3k lines after earlier callable extraction. | Mixed lowering responsibilities. | Continue same-package helper extraction with focused lower tests. |
| CLI Eco | `cli/cmd/tetra/eco.go` is about 4.4k lines. | Package, trust, lock, vault, and publish concerns sit together. | Inventory command clusters before extraction; keep public CLI output stable. |
| Release script tests | The historical monolithic v1 release script test file started at about 4.1k lines. | Gate fixtures and artifact validators were intertwined. The file has now been removed after domain splits. | Keep the structure guard and continue with other oversized test hotspots. |
| Tools validators | Many `tools/cmd/validate-*` commands repeat JSON decode and report validation patterns. | Premature shared libraries could couple unrelated validators. | Extract only after proving at least three commands share the same pure helper contract. |
| Docs/release evidence | Existing generated and release docs are numerous and partly historical. | Proxy green docs can be mistaken for release completeness. | Keep completion audits explicit; validators must distinguish blocker evidence from complete claims. |
| Generated artifacts | `docs/generated/**` and reports contain large machine-written files. | Refactors should not churn generated evidence accidentally. | Modify only when the generator/validator slice requires it. |
| Project directories | `compiler/tests`, `cli/tests`, `tools/release`, `tools/validators`, `docs/architecture`, `docs/audits`, `scripts/ci`, `scripts/dev`, and versioned `scripts/release/**` now exist with README contracts. | Empty structure can become decorative if migrations do not follow. | Extract `testkit` helpers first, then move one domain test/script group at a time. |

## Prompt-To-Artifact Checklist

| Objective requirement | Required artifact or command | Current evidence | Status |
| --- | --- | --- | --- |
| Global architecture audit | This plan plus Graphify and repo inventory commands. | Graphify queries and file-size inventory identified current hotspots. | started |
| Compiler structure | `compiler/**`; focused compiler package tests. | Hotspots identified in semantic checker, lowerer, function callable tests, ownership/safety tests. This pass split the initial function-typed throwing/direct-try cluster, throwing captured-closure return/cross-module cluster, full callable capture stress cluster, eight-slot captured-closure return stress cluster, local captured-closure direct/callback/matrix cluster, mutable-global captured-closure starter cluster, returned mutable-global/local captured-closure cluster, struct/nested-struct mutable-global captured-closure cluster, local enum-payload mutable-global captured-closure cluster, basic captured-closure callable cluster including captured return/direct-call/callback scenarios, parameter-return captured-closure storage, mutable/local/struct reassignment, enum-payload reassignment, returned struct/enum payload, and parameter field/payload return clusters, local direct-symbol/multi-target callable cluster, local return/parameter-alias/multi-target reassignment callable cluster, cross-module callback callable cluster including multi-target callback params and imported callback-argument returns, cross-module return/storage callable cluster, imported enum-payload callable cluster, cross-module multi-target direct/reassignment callable cluster, and cross-module direct-storage callable cluster into guarded files. `compiler/internal/testkit`, `compiler/tests/**`, and `compiler/testdata/callables/**` now define the directory target for follow-up moves. | in progress |
| CLI structure | `cli/cmd/tetra/**`; CLI package tests. | Existing plan already reduced `main.go`; this pass split `main_test.go` LSP, new app/project info, and all current fmt/format groups behind `TestCLITestsAreSplitByCommandSurface`. `cli/testkit` and `cli/tests/**` now define the directory target. `eco.go` remains a hotspot. | in progress |
| Tools structure | `tools/cmd/**`; `tools/scriptstest/**`; tools tests. | Validator and release-script hotspots identified. Ownership validator was already refactored in a prior goal. This pass split release script fixtures and started assertion splits with guarded v0.1.1 and v1.0 release tests. `tools/validators`, `tools/release/**`, `tools/testkit`, and versioned `tools/scriptstest/**` now define the directory target. | in progress |
| Docs structure | `docs/**`; docs validators. | Ownership audit was already made readable. `docs/architecture` and `docs/audits` now separate architecture/audit artifacts from plans and release evidence. Broader generated release-doc policy remains to audit. | in progress |
| Release gates | `scripts/ci/test.sh`, `scripts/ci/test-all.sh`, release gate scripts, scriptstest. | Current dirty tree includes release gate work; release truth was not rewritten. This pass moved test fixtures and v1.0 gate/smoke assertions, verified `tools/scriptstest`, moved the mutating formatter, bootstrap binary builder, bounded fuzz nightly, and project dump implementations to `scripts/dev/**`, moved the canonical Go test-suite plus summarized test-all implementations to `scripts/ci/**`, and moved all release workflows/gates into `scripts/release/**`. `find scripts -maxdepth 1 -type f -print` now returns only `scripts/README.md`; remaining legacy path strings are negative test guards or documented generated-evidence snapshot exceptions. | in progress |
| Generated artifacts | `docs/generated/**`, `reports/**`, generated validators. | Treat as evidence outputs, not primary refactor targets unless a generator slice requires updates. | open |
| Examples | `examples/**`; smoke/test commands. | Examples are part of release and docs evidence; no example-specific refactor selected yet. | open |
| Test suites | `*_test.go`, `tools/scriptstest`, focused package tests. | CLI test structure guard now enforces split ownership for metadata, doctor, clean, LSP, new app/project info, and fmt/format groups. Compiler function-typed callable throwing, throwing captured-closure return/cross-module, full-capture stress, eight-slot captured-return stress, local captured-closure direct/callback/matrix, mutable-global captured-closure starter, returned mutable-global/local captured-closure, struct/nested-struct mutable-global captured-closure, local enum-payload mutable-global captured-closure, basic captured-closure including captured return/direct-call/callback scenarios, parameter-return captured-closure storage, mutable/local/struct reassignment, enum-payload reassignment, returned struct/enum payload, and parameter field/payload return, local direct-symbol/multi-target, local return/parameter-alias/multi-target reassignment, cross-module callback including multi-target callback params and imported callback-argument returns, cross-module return/storage, imported enum-payload, cross-module multi-target direct/reassignment, and cross-module direct-storage tests now have guarded domain splits. `compiler/tests/**`, `cli/tests/**`, and versioned `tools/scriptstest/**` now define the directory target once `testkit` helpers are extracted. Release script tests now have guarded v0.1.1, v0.1.2, v0.1.3, v0.2.0, static v0.3.0 gate/checklist/security wrapper assertions, v0.3.0 evidence/preflight assertions, v0.3.0 stale report-dir assertions, v0.3.0 residual-risk assertions, v0.3.0 security-signoff policy assertions, v0.3.0 security-signoff acceptance/detached-hash assertions, v0.3.0 final-summary/artifact-hash assertions, v0.3.0 runtime-smoke evidence/archiving assertions, v0.3.0 runtime-smoke schema/type assertions, v0.4.0 gate readiness assertions, current surface docs guard, and bootstrap guard split out. The historical monolithic v1 release script test file was removed after the split. Larger compiler test hotspots remain open. | in progress |
| TDD/verification slices | Red/green/refactor loop per implementation slice. | Required before behavior-preserving code/test structure changes. | required |
| Graphify freshness | `graphify update .` after code changes. | Graphify is available at `graphify-out/`; update after each code slice. | required |
| No mock/placeholder claims | Completion audits and validators. | Keep claims tied to real files and command output. | required |
| Final completion audit | Prompt-to-artifact checklist mapping every requirement to evidence. | Not complete until all rows are closed or explicitly documented as intentionally out of scope. | open |

## Slice Queue

### Slice 1: CLI Main Test Structure Guard And Split

**Goal:** Start reducing split CLI command test files by introducing a
source-structure guard and moving a small cohesive command-test group into a
focused file.

**Files:**

- Inspect/modify split CLI command test files
- Add one focused `*_test.go` file in `cli/cmd/tetra`

**Approach:**

- Identify an already cohesive group by existing `Test...` names.
- Add a RED structural test that requires that group to live in the focused file.
- Move the tests mechanically without changing assertions, fixtures, command
  behavior, or output strings.

**Verification:**

- Focused CLI test regex for the moved group and structural guard.
- `GOCACHE=/tmp/tetra-language-go-cache go test ./cli/cmd/tetra -count=1`
- `git diff --check -- cli/cmd/tetra`
- `graphify update .`

**Done when:** The focused tests pass, the CLI package passes, and the file
split is enforced by a regression guard.

**2026-05-08 execution evidence:**

- Added/updated `cli/cmd/tetra/test_structure_test.go` as the structure guard.
- Moved LSP tests and helpers to `cli/cmd/tetra/lsp_test.go`.
- Moved new app and project info tests to `cli/cmd/tetra/new_app_test.go`.
- Moved the initial contiguous fmt/format group and later fmt diagnostics
  tests to `cli/cmd/tetra/fmt_test.go`.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./cli/cmd/tetra -run 'TestCLITestsAreSplitByCommandSurface|TestLSP' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./cli/cmd/tetra -run 'TestCLITestsAreSplitByCommandSurface|TestNewApp|TestProjectInfoCommandJSON' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./cli/cmd/tetra -run 'TestCLITestsAreSplitByCommandSurface|TestFmtCommandCheckAndStdout|TestCollectTetraFiles|TestFormatCommand' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./cli/cmd/tetra -run 'TestCLITestsAreSplitByCommandSurface|TestFmt|TestFormatCommand|TestCollectTetraFiles' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./cli/cmd/tetra -count=1`,
  `git diff --check -- cli/cmd/tetra/fmt_test.go cli/cmd/tetra/lsp_test.go cli/cmd/tetra/new_app_test.go cli/cmd/tetra/test_structure_test.go`,
  and `graphify update .`.

### Slice 2: Function-Typed Callable Test Matrix Split

**Goal:** Split `compiler/tests/callables/function_typed_callable_test.go` into cohesive
subdomain files.

**Verification:** Focused `go test ./compiler -run 'FunctionTyped|Callable'`
plus `go test ./compiler -count=1`.

**2026-05-08 execution evidence:**

- Added `compiler/tests/callables/function_typed_callable_structure_test.go` as a domain split
  guard.
- Moved the initial throwing/direct-try cluster to
  `compiler/tests/callables/function_typed_callable_throwing_test.go`.
- Moved the throwing captured-closure return/cross-module callable cluster into
  `compiler/tests/callables/function_typed_callable_throwing_test.go`.
- Moved the full callable nine/twelve capture stress cluster to
  `compiler/tests/callables/function_typed_callable_full_capture_test.go`.
- Moved the eight-slot captured-closure cross-module return stress tests to
  `compiler/tests/callables/function_typed_callable_full_capture_test.go`.
- Moved the local captured-closure direct-call, callback matrix, five-slot ptr,
  optional/enum/composite capture, and mutable snapshot tests to
  `compiler/tests/callables/function_typed_callable_captured_closure_test.go`.
- Moved the mutable-global captured-closure direct/callback starter cluster to
  `compiler/tests/callables/function_typed_callable_mutable_global_test.go`.
- Moved the returned mutable-global/local captured-closure reassignment cluster
  to `compiler/tests/callables/function_typed_callable_mutable_global_test.go`.
- Moved the struct-field, whole-struct, and nested-struct mutable-global
  captured-closure reassignment/snapshot cluster to
  `compiler/tests/callables/function_typed_callable_mutable_global_test.go`.
- Moved the local enum-payload and returned-enum-payload mutable-global
  captured-closure cluster to
  `compiler/tests/callables/function_typed_callable_mutable_global_test.go`.
- Moved the basic captured pointer closure callable cluster to
  `compiler/tests/callables/function_typed_callable_captured_closure_test.go`.
- Moved the captured-closure return/direct-call callable tests into
  `compiler/tests/callables/function_typed_callable_captured_closure_test.go`.
- Moved the captured-closure callback-return callable tests into
  `compiler/tests/callables/function_typed_callable_captured_closure_test.go`.
- Moved the parameter-return captured-closure storage callable cluster to
  `compiler/tests/callables/function_typed_callable_parameter_return_capture_test.go`.
- Moved the parameter-return captured-closure mutable/local/struct reassignment
  callable cluster to
  `compiler/tests/callables/function_typed_callable_parameter_return_reassignment_test.go`.
- Moved the parameter-return captured-closure enum-payload reassignment callable
  cluster to
  `compiler/tests/callables/function_typed_callable_parameter_return_enum_reassignment_test.go`.
- Moved the returned struct/enum payload callable cluster to
  `compiler/tests/callables/function_typed_callable_returned_struct_enum_payload_test.go`.
- Moved the parameter field/payload return captured-closure callable cluster to
  `compiler/tests/callables/function_typed_callable_parameter_field_return_test.go`.
- Moved the local direct-symbol, multi-target return, and argument-label
  callable cluster to `compiler/tests/callables/function_typed_callable_direct_symbol_test.go`.
- Moved the local return, parameter alias, multi-target return, and direct
  callback-argument callable cluster to
  `compiler/tests/callables/function_typed_callable_local_return_alias_test.go`.
- Moved the local multi-target return alias and mutable-local reassignment
  callable tests into `compiler/tests/callables/function_typed_callable_local_return_alias_test.go`.
- Moved the first cross-module callback callable cluster to
  `compiler/tests/callables/function_typed_callable_cross_module_callback_test.go`.
- Moved the cross-module multi-target callback-param test into
  `compiler/tests/callables/function_typed_callable_cross_module_callback_test.go`.
- Moved the imported/cross-module callback-argument return tests into
  `compiler/tests/callables/function_typed_callable_cross_module_callback_test.go`.
- Moved the first cross-module direct return/storage callable cluster to
  `compiler/tests/callables/function_typed_callable_cross_module_return_test.go`.
- Moved the imported enum-payload callable parameter cluster to
  `compiler/tests/callables/function_typed_callable_imported_enum_payload_test.go`.
- Moved the cross-module multi-target direct call and reassignment callable
  cluster to `compiler/tests/callables/function_typed_callable_cross_module_multi_target_test.go`.
- Moved the cross-module direct named-symbol struct/enum storage callable
  cluster to `compiler/tests/callables/function_typed_callable_cross_module_direct_storage_test.go`.
- Left later cross-module/global callable scenarios in
  `compiler/tests/callables/function_typed_callable_test.go` for future domain splits.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run TestFunctionTypedCallableTestsAreSplitByDomain -count=1` failed RED before the move,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTypedThrowing.*Smoke' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTypedThrowingCapturedClosure(ReturnCrossModuleDirectTrySmoke|ReturnCrossModuleDirectCallbackArgumentSmoke|ReturnedStructFieldCrossModuleDirectTrySmoke|ReturnedEnumPayloadCrossModuleDirectTrySmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFullCallable' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTypedCapturedClosureEightSlot(ReturnCrossModuleCallbackSmoke|EnumReturnCrossModuleCallbackSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTypedCapturedClosure(LocalDirectCallSmoke|LocalDirectCallAllowsArgumentLabelsSmoke|CompositeCaptureMatrixCallbackSmoke|EnumCaptureMatrixCallbackSmoke|OptionalCaptureMatrixCallbackSmoke)|TestBuildFunctionTypedCapturedPtrClosureFiveSlotDirectAndCallbackSmoke|TestBuildFunctionTypedMutableCaptureSnapshotsAtBindingSmoke' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTypedCaptured(Closure|PtrClosure)MutableGlobalReassignment(DirectCallSmoke|CallbackArgumentSmoke|ReturnDirectCallSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTypedCapturedClosureReturn(CallMutableGlobalReassignmentDirectCallSmoke|LocalMutableGlobalReassignmentDirectCallSmoke|MutableLocalReassignmentDirectCallSmoke|MutableLocalMutableGlobalReassignmentDirectCallSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTypedCapturedClosure(ReturnStructFieldMutableGlobalReassignmentDirectCallSmoke|StructFieldMutableGlobalSnapshotSmoke|ReturnNestedStructFieldMutableGlobalReassignmentDirectCallSmoke|ReturnWholeStructMutableGlobalReassignmentDirectCallSmoke|WholeStructMutableGlobalSnapshotSmoke|ReturnWholeNestedStructMutableGlobalReassignmentDirectCallSmoke|WholeNestedStructMutableGlobalSnapshotSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTypedCapturedClosure(ReturnEnumPayloadMutableGlobalReassignmentDirectCallSmoke|EnumPayloadMutableGlobalSnapshotSmoke|WholeEnumMutableGlobalSnapshotSmoke|ReturnedStructEnumPayloadMutableGlobalSnapshotSmoke|ReturnedEnumPayloadMutableGlobalSnapshotSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuild(FunctionTypedLocalAliasCapturedPtrClosureSmoke|CapturedPtrClosureLabeledDirectCallSmoke|CapturedPtrClosureDirectCallbackArgumentSmoke|CapturedPtrClosureReturnedFunctionValueSmoke|FunctionTypedMutableLocalReassignCapturedPtrClosureSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(ReturnMultiTargetCapturedClosureDirectCallSmoke|ReturnMultiTargetCapturedPtrClosureDirectCallSmoke|ReturnCapturedPtrClosureDirectCallbackArgumentSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(ReturnMultiTargetCapturedClosureCallbackSmoke|CapturedClosureReturnCrossModuleCallbackSmoke|CapturedClosureReturnCrossModuleDirectCallbackArgumentSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(ParameterReturnCapturedPtrClosureSmoke|ParameterReturnCapturedPtrClosureCrossModuleSmoke|ReturnCallStructFieldCapturedPtrClosureSmoke|ReturnCallEnumPayloadCapturedPtrClosureSmoke|ImportedParameterReturnStructFieldCapturedPtrClosureSmoke|ImportedParameterReturnEnumPayloadCapturedPtrClosureSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(ParameterReturnMutableLocalReassignmentCapturedPtrClosureSmoke|ParameterReturnStructFieldReassignmentCapturedPtrClosureSmoke|ImportedParameterReturnMutableLocalReassignmentCapturedPtrClosureSmoke|ImportedParameterReturnStructFieldReassignmentCapturedPtrClosureSmoke|ImportedParameterReturnNestedStructFieldReassignmentCapturedPtrClosureSmoke|ImportedParameterReturnWholeStructReassignmentCapturedPtrClosureSmoke|ImportedParameterReturnStructValuedFieldReassignmentCapturedPtrClosureSmoke|ImportedParameterReturnWholeNestedStructReassignmentCapturedPtrClosureSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(ParameterReturnEnumPayloadReassignmentCapturedPtrClosureSmoke|ImportedParameterReturnEnumPayloadReassignmentCapturedPtrClosureSmoke|ImportedParameterReturnStructFieldEnumPayloadReassignmentCapturedPtrClosureSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(ReturnedStructEnumPayloadCapturedPtrClosureSmoke|ImportedReturnedStructEnumPayloadCapturedPtrClosureSmoke|ReturnedStructEnumPayloadDirectFieldMatchCapturedPtrClosureSmoke|ReturnedStructEnumPayloadWholeStructReassignmentCapturedPtrClosureSmoke|NestedReturnedStructEnumPayloadCapturedPtrClosureSmoke|ReturnedStructEnumPayloadMultiTargetDirectCallSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(StructParameterFieldReturnCapturedClosureDirectCallSmoke|EnumParameterPayloadReturnCapturedClosureDirectCallSmoke|StructParameterFieldReturnCapturedClosureCallbackArgumentSmoke|NestedStructParameterFieldReturnCapturedClosureDirectCallSmoke|StructParameterWholeReturnCapturedClosureDirectCallSmoke|EnumParameterWholeReturnCapturedClosureDirectCallSmoke|EnumParameterPayloadReturnCapturedClosureCallbackArgumentSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(CallableParamDirectNamedSymbolSmoke|CallableParamMultiTargetSmoke|CallableParamMultiTargetStringReturnSmoke|CallableParamMultiTargetStructReturnSmoke|CallbackCallAllowsArgumentLabelsSmoke|StructFieldCallAllowsArgumentLabelsSmoke|GlobalCallAllowsArgumentLabelsSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(ReturnSymbolBackedValueSmoke|ReturnParameterValueSmoke|ReturnParameterDirectCallbackArgumentSmoke|ParameterAliasDirectCallSmoke|ParameterAliasCallbackArgumentSmoke|ReturnDirectNamedSymbolSmoke|ReturnMultiTargetDirectCallSmoke|ReturnMultiTargetCallbackSmoke|ReturnDirectCallbackArgumentSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(ReturnMultiTargetLocalAliasSmoke|MutableLocalReassignmentFromMultiTargetReturnSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(CallableParamCrossModuleSmoke|StructFieldCrossModuleCallbackSmoke|EnumPayloadCrossModuleCallbackSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTypedCallableParamMultiTargetCrossModuleSmoke' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(ReturnDirectCallbackArgumentCrossModuleSmoke|ImportedParameterReturnCapturedPtrClosureDirectCallbackArgumentSmoke|ImportedReturnIgnoresCapturedCallbackArgumentSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(CallableParamDirectNamedSymbolCrossModuleSmoke|ReturnDirectNamedSymbolCrossModuleSmoke|ReturnMultiTargetCrossModuleCallbackSmoke|StructFieldFromMultiTargetCrossModuleReturnSmoke|StructFieldFromCapturedCrossModuleReturnSmoke|EnumPayloadFromMultiTargetCrossModuleReturnSmoke|ParameterReturnedEnumPayloadCrossModuleSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(ImportedEnumPayloadParamCapturedClosureSmoke|ImportedEnumPayloadParamDirectReturnCapturedClosureSmoke|ImportedEnumPayloadParamDirectConstructorCapturedClosureSmoke|SelectiveImportedEnumPayloadParamDirectConstructorCapturedClosureSmoke|ImportedEnumPayloadParamDirectConstructorClosureLiteralSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(ReturnMultiTargetCrossModuleDirectCallSmoke|MutableLocalReassignmentFromMultiTargetCrossModuleReturnSmoke|StructFieldReassignmentFromMultiTargetCrossModuleReturnSmoke|MutableEnumPayloadReassignmentFromMultiTargetCrossModuleReturnSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -run 'TestFunctionTypedCallableTestsAreSplitByDomain|TestBuildFunctionTyped(StructFieldDirectNamedSymbolCrossModuleSmoke|EnumPayloadDirectNamedSymbolCrossModuleSmoke)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./compiler -count=1`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_throwing_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_full_capture_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_captured_closure_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_parameter_return_capture_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_parameter_return_reassignment_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_parameter_return_enum_reassignment_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_returned_struct_enum_payload_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_parameter_field_return_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_direct_symbol_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_local_return_alias_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_cross_module_callback_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_cross_module_return_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_imported_enum_payload_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_cross_module_multi_target_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  `git diff --check -- compiler/tests/callables/function_typed_callable_test.go compiler/tests/callables/function_typed_callable_cross_module_direct_storage_test.go compiler/tests/callables/function_typed_callable_structure_test.go`,
  and `graphify update .` (latest code graph: 10763 nodes, 38907 edges, 311 communities).

### Slice 2b: Directory-Based Project Structure Scaffold

**Goal:** Create the directory-based target structure before moving more tests,
scripts, fixtures, and evidence files.

**2026-05-08 execution evidence:**

- Added `docs/architecture/project_structure.md` as the canonical structure map.
- Added README contracts for `compiler/internal/testkit`,
  `compiler/tests/**`, `compiler/testdata/callables/**`, `cli/testkit`,
  `cli/tests/**`, `tools/validators`, `tools/release/**`, `tools/testkit`,
  `tools/scriptstest/fixtures`, versioned `tools/scriptstest/**`,
  `docs/architecture`, `docs/audits`, and `examples/smoke`.
- Added compatibility wrappers under `scripts/ci`, `scripts/dev`, and
  versioned `scripts/release/**` while keeping root-level scripts intact.
- Verification:
  `find scripts/ci scripts/dev scripts/release -type f -name '*.sh' -print0 | xargs -0 -n1 sh -n`,
  `git diff --check -- docs/plans/2026-05-08-global-architecture-refactor-plan.md docs/architecture/project_structure.md compiler/internal/testkit/README.md compiler/tests/README.md cli/testkit/README.md cli/tests/README.md tools/release/README.md tools/validators/README.md tools/testkit/README.md scripts/README.md`,
  and `graphify update .` (latest code graph: 10763 nodes, 38907 edges, 311 communities).

### Slice 3: Semantic Checker Same-Package Helper Extraction

**Goal:** Reduce `compiler/internal/semantics/checker.go` by extracting a pure,
well-bounded helper cluster.

**Verification:** Dependency/impact inspection, focused semantics tests,
compiler tests for the touched behavior, and `go test ./compiler/internal/semantics -count=1`.

### Slice 3a: Dev Script Migration

**Goal:** Move real developer script implementations into the domain directory
while preserving the legacy root entrypoints.

**2026-05-08 execution evidence:**

- Updated `tools/scriptstest/shell_portability_test.go` so
  `TestFormattingWorkflowSeparatesCheckAndWrite` requires
  `scripts/dev/format.sh` to own the mutating `gofmt -w` workflow.
- Moved the existing formatter implementation to `scripts/dev/format.sh`.
- Updated `scripts/README.md` and `scripts/dev/README.md` to document the
  implementation-vs-wrapper rule.
- Updated `tools/scriptstest/release_bootstrap_test.go` so
  `TestBootstrapBuildsTetraAndTAlias` requires `scripts/dev/bootstrap.sh` to own
  the binary build workflow and `scripts/bootstrap.sh` to delegate to it.
- Observed the expected RED failure before the bootstrap move:
  `scripts/bootstrap.sh must stay a compatibility wrapper; build logic belongs in scripts/dev/bootstrap.sh`.
- Moved the existing bootstrap implementation to `scripts/dev/bootstrap.sh`.
- Replaced `scripts/bootstrap.sh` with a compatibility wrapper that execs
  `scripts/dev/bootstrap.sh` (later removed in Slice 3g).
- Updated `tools/scriptstest/fuzz_nightly_test.go` so
  `TestFuzzNightlyWrapperDocumentsBoundedCommands` requires
  `scripts/dev/fuzz-nightly.sh` to own the bounded fuzz/property/stress workflow
  and `scripts/fuzz_nightly.sh` to delegate to it.
- Observed the expected RED failure before the fuzz nightly move:
  `scripts/fuzz_nightly.sh must delegate to scripts/dev/fuzz-nightly.sh`.
- Moved the existing fuzz nightly implementation to
  `scripts/dev/fuzz-nightly.sh`.
- Replaced `scripts/fuzz_nightly.sh` with a compatibility wrapper that execs
  `scripts/dev/fuzz-nightly.sh`.
- Added `TestDumpProjectWorkflowLivesInDevScript` to require
  `scripts/dev/dump-project.sh` to own the dump workflow.
- Moved the existing dump implementation to `scripts/dev/dump-project.sh`.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestFormattingWorkflowSeparatesCheckAndWrite -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestBootstrapBuildsTetraAndTAlias -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestFuzzNightlyWrapperDocumentsBoundedCommands -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestFuzzNightly -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestDumpProjectWorkflowLivesInDevScript -count=1`,
  `gofmt -w tools/scriptstest/shell_portability_test.go tools/scriptstest/release_bootstrap_test.go tools/scriptstest/fuzz_nightly_test.go`,
  `bash -n scripts/dev/format.sh scripts/bootstrap.sh scripts/dev/bootstrap.sh scripts/dev/fuzz-nightly.sh scripts/dev/fuzz-nightly.sh scripts/dev/dump-project.sh`,
  and `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`.

### Slice 3b: CI Test Script Migration

**Goal:** Move the canonical Go test-suite implementation into the CI domain
directory while preserving legacy root entrypoints.

**2026-05-08 execution evidence:**

- Updated `TestCanonicalTestScriptDoesNotMutateFormatting` so it requires
  `scripts/ci/test.sh` to own the non-mutating `gofmt -l` and `go test`
  workflow while `scripts/test.sh` delegates to it.
- Observed the expected RED failure before the move:
  `scripts/test.sh must delegate to scripts/ci/test.sh`.
- Moved the existing test-suite implementation to `scripts/ci/test.sh`.
- Replaced `scripts/test.sh` with a compatibility wrapper that execs
  `scripts/ci/test.sh` (later removed in Slice 3i).
- Updated the temp-repo setup in `tools/scriptstest/test_script_test.go` so
  wrapper behavior is tested with the canonical CI script present.
- Updated `scripts/ci/README.md` to document the canonical entrypoint and
  compatibility wrapper.
- Added `TestTestAllWorkflowLivesInCIEntryPoint` so `scripts/ci/test-all.sh`
  owns the summarized release/stabilization runner and `scripts/test_all.sh`
  delegates to it.
- Observed the expected RED failure before the test-all move:
  `scripts/test_all.sh must delegate to scripts/ci/test-all.sh`.
- Moved the existing test-all implementation to `scripts/ci/test-all.sh`.
- Replaced `scripts/test_all.sh` with a compatibility wrapper that execs
  `scripts/ci/test-all.sh` (later removed in Slice 3j).
- Updated `tools/scriptstest/test_all_test.go` so static assertions read the
  canonical CI script while execution still goes through the legacy wrapper.
- Fixed `scripts/ci/test.sh` and `scripts/ci/test-all.sh` repo-root discovery
  to climb from `scripts/ci` to the repository root.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestCanonicalTestScriptDoesNotMutateFormatting -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestCanonicalTestScript' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestTestAllWorkflowLivesInCIEntryPoint -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestTestAll -count=1`,
  `gofmt -w tools/scriptstest/shell_portability_test.go tools/scriptstest/test_script_test.go tools/scriptstest/release_bootstrap_test.go tools/scriptstest/fuzz_nightly_test.go`,
  `bash -n scripts/test.sh scripts/ci/test.sh scripts/test_all.sh scripts/ci/test-all.sh scripts/dev/format.sh scripts/bootstrap.sh scripts/dev/bootstrap.sh scripts/fuzz_nightly.sh scripts/dev/fuzz-nightly.sh scripts/dev/dump-project.sh`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `git diff --check -- scripts/test.sh scripts/ci/test.sh scripts/test_all.sh scripts/ci/test-all.sh scripts/dev/format.sh scripts/bootstrap.sh scripts/dev/bootstrap.sh scripts/fuzz_nightly.sh scripts/dev/fuzz-nightly.sh scripts/dev/dump-project.sh tools/scriptstest/shell_portability_test.go tools/scriptstest/test_script_test.go tools/scriptstest/test_all_test.go tools/scriptstest/release_bootstrap_test.go tools/scriptstest/fuzz_nightly_test.go scripts/README.md scripts/dev/README.md scripts/ci/README.md docs/plans/2026-05-08-global-architecture-refactor-plan.md`,
  and `graphify update .`.

### Slice 3f: Remove Dev Legacy Root Entrypoints

**Goal:** Apply the no-wrapper rule to the already-migrated formatter and
project dump workflows.

**2026-05-08 execution evidence:**

- Added `TestDevScriptsHaveNoLegacyRootEntrypoints` to require removal of
  `scripts/format.sh` and `scripts/dump.sh`.
- Observed the expected RED failure before deletion:
  `scripts/format.sh must be removed; use the canonical dev script path`.
- Deleted `scripts/format.sh` and `scripts/dump.sh`.
- Updated canonical help text and `scripts/dev/README.md` so
  `scripts/dev/format.sh` and `scripts/dev/dump-project.sh` no longer
  advertise removed root-level wrappers.

### Slice 3g: Remove Bootstrap Legacy Root Entrypoint

**Goal:** Apply the no-wrapper rule to the already-migrated bootstrap workflow.

**2026-05-08 execution evidence:**

- Added a RED assertion in `TestBootstrapBuildsTetraAndTAlias` requiring
  `scripts/bootstrap.sh` to be absent.
- Observed the expected RED failure before deletion:
  `scripts/bootstrap.sh must be removed; use scripts/dev/bootstrap.sh`.
- Updated CI, release gates, docs, and script test fixtures from
  `scripts/bootstrap.sh` to `scripts/dev/bootstrap.sh`.
- Deleted `scripts/bootstrap.sh`.
- Updated `scripts/dev/bootstrap.sh` help and `scripts/dev/README.md` so they
  do not advertise a root-level compatibility wrapper.

### Slice 3h: Remove Fuzz Nightly Legacy Root Entrypoint

**Goal:** Apply the no-wrapper rule to the already-migrated fuzz nightly
workflow.

**2026-05-08 execution evidence:**

- Added a RED assertion in `TestFuzzNightlyWrapperDocumentsBoundedCommands`
  requiring `scripts/fuzz_nightly.sh` to be absent.
- Observed the expected RED failure before deletion:
  `scripts/fuzz_nightly.sh must be removed; use scripts/dev/fuzz-nightly.sh`.
- Updated CI, v0.3 release gate docs/tests, fuzz docs, README, and cheatsheet
  references from `scripts/fuzz_nightly.sh` to `scripts/dev/fuzz-nightly.sh`.
- Deleted `scripts/fuzz_nightly.sh`.
- Updated `scripts/dev/fuzz-nightly.sh` help and `scripts/dev/README.md` so
  they do not advertise a root-level compatibility wrapper.

### Slice 3i: Remove CI Test Legacy Root Entrypoint

**Goal:** Apply the no-wrapper rule to the already-migrated fast Go test
workflow.

**2026-05-08 execution evidence:**

- Added a RED assertion in `TestCanonicalTestScriptDoesNotMutateFormatting`
  requiring `scripts/test.sh` to be absent.
- Observed the expected RED failure before deletion:
  `scripts/test.sh must be removed; use scripts/ci/test.sh`.
- Updated CI, release gates, docs, temp-repo script tests, and
  `scripts/ci/test-all.sh` from `scripts/test.sh` to `scripts/ci/test.sh`.
- Deleted `scripts/test.sh`.
- Updated `scripts/ci/test.sh` help and `scripts/ci/README.md` so they do not
  advertise a root-level compatibility wrapper.

### Slice 3j: Remove CI Test-All Legacy Root Entrypoint

**Goal:** Apply the no-wrapper rule to the already-migrated summarized
release/stabilization runner.

**2026-05-08 execution evidence:**

- Added a RED assertion in `TestTestAllWorkflowLivesInCIEntryPoint` requiring
  `scripts/test_all.sh` to be absent.
- Observed the expected RED failure before deletion:
  `scripts/test_all.sh must be removed; use scripts/ci/test-all.sh`.
- Updated CI, release gates, docs, CLI tests, script tests, validators, and
  fake release repos from `scripts/test_all.sh` to `scripts/ci/test-all.sh`.
- Deleted `scripts/test_all.sh`.
- Updated `scripts/ci/test-all.sh` help and `scripts/ci/README.md` so they do
  not advertise a root-level compatibility wrapper.

### Slice 3c: Versioned Release API Diff Script Migration

**Goal:** Move one release workflow implementation into the versioned release
directory and then remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Updated `TestReleaseV10APIDiffWorkflowLivesInVersionedReleaseScript` so
  `scripts/release/v1_0/api-diff.sh` owns the API diff workflow and the removed
  root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper:
  the root API diff wrapper must not exist, and callers must use
  `scripts/release/v1_0/api-diff.sh` directly.
- Updated CI, release gates, docs, script tests, and fake release repos from the
  root API diff wrapper to `scripts/release/v1_0/api-diff.sh`.
- Deleted the root API diff wrapper.
- Removed the legacy usage line from `scripts/release/v1_0/api-diff.sh` while
  preserving the artifact mapping.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV10APIDiffWorkflowLivesInVersionedReleaseScript|TestReleaseV10APIDiff|TestAPIDiff|TestTestAllWorkflowLivesInCIEntryPoint|TestReleaseV10GateUsesRealV1Boundary' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `bash -n scripts/release/v1_0/api-diff.sh scripts/release/v0_1_1/gate.sh scripts/release/v0_1_2/gate.sh scripts/release/v0_1_3/gate.sh scripts/release/v1_0/gate.sh scripts/ci/test-all.sh`,
  `bash scripts/release/v1_0/api-diff.sh --help`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./cli/cmd/tetra -run TestTestAllScript -count=1`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3d: Versioned Release Binary Size Script Migration

**Goal:** Move another v1.0 release workflow implementation into the versioned
release directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Updated `TestReleaseV10BinarySizeWorkflowLivesInVersionedReleaseScript` so
  `scripts/release/v1_0/binary-size.sh` owns the binary-size threshold workflow
  and the removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root
  binary-size wrapper must not exist, and callers must use
  `scripts/release/v1_0/binary-size.sh` directly.
- Updated release gates, docs, script tests, and fake release repos from the
  root binary-size wrapper to `scripts/release/v1_0/binary-size.sh`.
- Deleted the root binary-size wrapper.
- Updated `scripts/release/README.md` to document the canonical versioned
  entrypoint.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV10BinarySizeWorkflowLivesInVersionedReleaseScript|TestReleaseV10BinarySize|TestReleaseV011Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `bash -n scripts/release/v1_0/binary-size.sh scripts/release/v0_1_1/gate.sh scripts/release/v0_1_2/gate.sh scripts/release/v0_1_3/gate.sh scripts/release/v1_0/gate.sh`,
  `bash scripts/release/v1_0/binary-size.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3e: Versioned Release Reproducible Build Script Migration

**Goal:** Move the reproducible-build proof workflow into the versioned release
directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Updated `TestReleaseV10ReproWorkflowLivesInVersionedReleaseScript` so
  `scripts/release/v1_0/reproducible-build.sh` owns the reproducible-build
  proof workflow and the removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root repro
  wrapper must not exist, and callers must use
  `scripts/release/v1_0/reproducible-build.sh` directly.
- Updated release gates, docs, script tests, and fake release repos from the
  root repro wrapper to `scripts/release/v1_0/reproducible-build.sh`.
- Deleted the root repro wrapper.
- Updated `scripts/release/README.md` to document the canonical versioned
  entrypoint.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV10ReproWorkflowLivesInVersionedReleaseScript|TestReleaseV10Repro' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `bash -n scripts/release/v1_0/reproducible-build.sh scripts/release/v0_1_1/gate.sh scripts/release/v0_1_2/gate.sh scripts/release/v0_1_3/gate.sh scripts/release/v1_0/gate.sh`,
  `bash scripts/release/v1_0/reproducible-build.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3f: Versioned Release Security Review Script Migration

**Goal:** Move the v1.0 security review validator implementation into the
versioned release directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Added `TestReleaseV10SecurityReviewWorkflowLivesInVersionedReleaseScript` so
  `scripts/release/v1_0/security-review.sh` owns the validator implementation
  and the removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root
  security review wrapper must not exist, and callers must use
  `scripts/release/v1_0/security-review.sh` directly.
- Moved the validator implementation into
  `scripts/release/v1_0/security-review.sh`.
- Updated release gates, the v0.3 security wrapper, docs, validators, script
  tests, and fake release repos from the root security wrapper to the canonical
  versioned script.
- Deleted the root security review wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV10SecurityReviewWorkflowLivesInVersionedReleaseScript|TestSecurityReviewSignoffValidator|TestReleaseV030SecurityReviewWrapperUsesV030Name|TestReleaseV011Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-state -count=1`,
  `bash -n scripts/release/v1_0/security-review.sh scripts/release/v0_1_1/gate.sh scripts/release/v0_1_2/gate.sh scripts/release/v0_1_3/gate.sh scripts/release/v1_0/gate.sh scripts/release/v0_3_0/security-review.sh`,
  `bash scripts/release/v1_0/security-review.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3g: Versioned Release WASI Smoke Script Migration

**Goal:** Move the v1.0 WASI smoke workflow implementation into the versioned
release directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Added `TestReleaseV10WASISmokeWorkflowLivesInVersionedReleaseScript` so
  `scripts/release/v1_0/wasi-smoke.sh` owns the workflow implementation and the
  removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root WASI
  smoke wrapper must not exist, and callers must use
  `scripts/release/v1_0/wasi-smoke.sh` directly.
- Moved the WASI smoke implementation into
  `scripts/release/v1_0/wasi-smoke.sh`.
- Updated release gates, CI test-all, docs, security-review evidence templates,
  script tests, and fake release repos from the root WASI smoke wrapper to the
  canonical versioned script.
- Deleted the root WASI smoke wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV10WASISmokeWorkflowLivesInVersionedReleaseScript|TestReleaseV10WASISmoke|TestReleaseV10SmokeScriptsHaveDefaultReportPaths|TestSmokeSourceSetsUseUnifiedRegistry|TestTestAllWorkflowLivesInCIEntryPoint|TestSecurityReviewSignoffValidator|TestReleaseV030Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `bash -n scripts/release/v1_0/wasi-smoke.sh scripts/release/v0_1_1/gate.sh scripts/release/v0_1_2/gate.sh scripts/release/v0_1_3/gate.sh scripts/release/v1_0/gate.sh scripts/ci/test-all.sh scripts/release/v1_0/security-review.sh`,
  `bash scripts/release/v1_0/wasi-smoke.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3h: Versioned Release Web Smoke Script Migration

**Goal:** Move the v1.0 web smoke workflow implementation into the versioned
release directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Added `TestReleaseV10WebSmokeWorkflowLivesInVersionedReleaseScript` so
  `scripts/release/v1_0/web-smoke.sh` owns the workflow implementation and the
  removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root web
  smoke wrapper must not exist, and callers must use
  `scripts/release/v1_0/web-smoke.sh` directly.
- Moved the web smoke implementation into
  `scripts/release/v1_0/web-smoke.sh`.
- Updated release gates, CI test-all, docs, security-review evidence templates,
  script tests, v0.4 readiness fixtures, and fake release repos from the root
  web smoke wrapper to the canonical versioned script.
- Deleted the root web smoke wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV10WebSmokeWorkflowLivesInVersionedReleaseScript|TestReleaseV10WebSmokeScript|Test_release_v1_0_web_smoke|TestReleaseV10SmokeScriptsHaveDefaultReportPaths|TestSmokeSourceSetsUseUnifiedRegistry|TestTestAllWorkflowLivesInCIEntryPoint|TestSecurityReviewSignoffValidator|TestReleaseV030Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-v0-4-readiness -count=1`,
  `bash -n scripts/release/v1_0/web-smoke.sh scripts/release/v0_1_1/gate.sh scripts/release/v0_1_2/gate.sh scripts/release/v0_1_3/gate.sh scripts/release/v1_0/gate.sh scripts/ci/test-all.sh scripts/release/v1_0/security-review.sh`,
  `bash scripts/release/v1_0/web-smoke.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3i: Versioned Release v1.0 Gate Migration

**Goal:** Move the v1.0 release gate implementation into the versioned release
directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Updated `TestReleaseV10GateUsesRealV1Boundary` so
  `scripts/release/v1_0/gate.sh` owns the dedicated v1 gate implementation and
  the removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root v1
  gate wrapper must not exist, and callers must use
  `scripts/release/v1_0/gate.sh` directly.
- Moved the v1 gate implementation into `scripts/release/v1_0/gate.sh`.
- Updated the v1 gate release command identity, release-state validator
  expectations, script tests, fake release repos, release docs, backend docs,
  roadmap/planning docs, and generated known-issues text from the root v1 gate
  path to the canonical versioned script.
- Deleted the root v1 gate wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV10Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-state -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `bash -n scripts/release/v1_0/gate.sh`,
  `bash scripts/release/v1_0/gate.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3j: Versioned Release v0.3.0 Security Review Migration

**Goal:** Move the v0.3.0 security review validator wrapper into the versioned
release directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Updated `TestReleaseV030SecurityReviewWrapperUsesV030Name` so
  `scripts/release/v0_3_0/security-review.sh` owns the v0.3.0-scoped
  validator wrapper and the removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root v0.3
  security-review wrapper must not exist, and callers must use
  `scripts/release/v0_3_0/security-review.sh` directly.
- Moved the v0.3 security-review implementation into
  `scripts/release/v0_3_0/security-review.sh` and updated its shared v1
  validator delegation for the nested script location.
- Updated v0.3 gate wiring, docs, script tests, fake release repos, and
  release-state validator expectations from the root v0.3 security-review path
  to the canonical versioned script.
- Deleted the root v0.3 security-review wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV030SecurityReviewWrapperUsesV030Name|TestReleaseV030Gate|TestSecurityReviewSignoffValidator' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-state -run 'TestReleaseState' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-state -count=1`,
  `bash -n scripts/release/v0_3_0/security-review.sh scripts/release/v0_3_0/gate.sh`,
  `bash scripts/release/v0_3_0/security-review.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3k: Versioned Release v0.4.0 Security Review Migration

**Goal:** Move the v0.4.0 security review template/blocking validator into the
versioned release directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Added `TestReleaseV040SecurityReviewWorkflowLivesInVersionedReleaseScript` so
  `scripts/release/v0_4_0/security-review.sh` owns the v0.4.0 security-review
  workflow and the removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root v0.4
  security-review wrapper must not exist, and callers must use
  `scripts/release/v0_4_0/security-review.sh` directly.
- Moved the v0.4 security-review implementation into
  `scripts/release/v0_4_0/security-review.sh`.
- Updated docs and script tests from the root v0.4 security-review path to the
  canonical versioned script.
- Deleted the root v0.4 security-review wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV040SecurityReview|TestSecurityReviewSignoffValidator' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `bash -n scripts/release/v0_4_0/security-review.sh`,
  `bash scripts/release/v0_4_0/security-review.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3l: Versioned Release v0.1.1 Gate Migration

**Goal:** Move the v0.1.1 release gate implementation into the versioned
release directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Updated `TestReleaseV011GateDocumentsMandatoryTargets` so
  `scripts/release/v0_1_1/gate.sh` owns the v0.1.1 release gate implementation
  and the removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root v0.1.1
  gate wrapper must not exist, and callers must use
  `scripts/release/v0_1_1/gate.sh` directly.
- Moved the v0.1.1 gate implementation into
  `scripts/release/v0_1_1/gate.sh`.
- Updated docs, checklist references, known-issues text emitted by the gate, and
  script tests from the root v0.1.1 gate path to the canonical versioned script.
- Deleted the root v0.1.1 gate wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV011Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `bash -n scripts/release/v0_1_1/gate.sh`,
  `bash scripts/release/v0_1_1/gate.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3m: Versioned Release v0.1.2 Gate Migration

**Goal:** Move the v0.1.2 release gate implementation into the versioned
release directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Updated `TestReleaseV012GateArchivesReleaseStateWithExpectedVersion` so
  `scripts/release/v0_1_2/gate.sh` owns the v0.1.2 release gate implementation
  and the removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root v0.1.2
  gate wrapper must not exist, and callers must use
  `scripts/release/v0_1_2/gate.sh` directly.
- Moved the v0.1.2 gate implementation into
  `scripts/release/v0_1_2/gate.sh`.
- Updated docs, checklist references, known-issues text emitted by the gate, and
  script tests from the root v0.1.2 gate path to the canonical versioned script.
- Deleted the root v0.1.2 gate wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV012Gate|TestReleaseV012GateFormatterCoversRuntimeSources' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `bash -n scripts/release/v0_1_2/gate.sh`,
  `bash scripts/release/v0_1_2/gate.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3n: Versioned Release v0.1.3 Gate Migration

**Goal:** Move the v0.1.3 release gate implementation into the versioned
release directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Updated `TestReleaseV013GateIsCanonicalPatchGate` so
  `scripts/release/v0_1_3/gate.sh` owns the v0.1.3 release gate implementation
  and the removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root v0.1.3
  gate wrapper must not exist, and callers must use
  `scripts/release/v0_1_3/gate.sh` directly.
- Moved the v0.1.3 gate implementation into
  `scripts/release/v0_1_3/gate.sh`.
- Updated the gate's nested repo-root resolution, default release command,
  v0.2.0 delegation, release-state validator expectations, docs, generated
  v1.0 README references, and script tests from the root v0.1.3 gate path to
  the canonical versioned script.
- Deleted the root v0.1.3 gate wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestReleaseV013GateIsCanonicalPatchGate -count=1` failed RED before the move,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV013Gate|TestReleaseV020GateDelegatesWithV020Boundary|TestReleaseV10Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-state -run TestReleaseState -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-state -count=1`,
  `bash -n scripts/release/v0_1_3/gate.sh scripts/release/v0_2_0/gate.sh`,
  `bash scripts/release/v0_1_3/gate.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3o: Versioned Release v0.2.0 Gate Migration

**Goal:** Move the v0.2.0 release gate implementation into the versioned
release directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Updated `TestReleaseV020GateDelegatesWithV020Boundary` so
  `scripts/release/v0_2_0/gate.sh` owns the v0.2.0 release gate implementation
  and the removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root v0.2.0
  gate wrapper must not exist, and callers must use
  `scripts/release/v0_2_0/gate.sh` directly.
- Moved the v0.2.0 gate implementation into
  `scripts/release/v0_2_0/gate.sh`.
- Updated the gate's nested repo-root resolution, v0.1.3 delegation, release
  command identity, release-state validator expectations, release-gate-summary
  test fixture, docs, and script tests from the root v0.2.0 gate path to the
  canonical versioned script.
- Deleted the root v0.2.0 gate wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestReleaseV020GateDelegatesWithV020Boundary -count=1` failed RED before the move,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV020GateDelegatesWithV020Boundary|TestReleaseV10Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-state -run TestReleaseState -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-gate-summary -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-state -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-gate-summary -count=1`,
  `bash -n scripts/release/v0_2_0/gate.sh`,
  `bash scripts/release/v0_2_0/gate.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3p: Versioned Release v0.3.0 Gate Migration

**Goal:** Move the v0.3.0 release gate implementation into the versioned
release directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Updated `TestReleaseV030GateUsesDedicatedV030Boundary` so
  `scripts/release/v0_3_0/gate.sh` owns the v0.3.0 release gate implementation
  and the removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root v0.3.0
  gate wrapper must not exist, and callers must use
  `scripts/release/v0_3_0/gate.sh` directly.
- Moved the v0.3.0 gate implementation into
  `scripts/release/v0_3_0/gate.sh`.
- Updated the gate's nested repo-root resolution, release command identity,
  runnable fake repos, CI release gate job, release-state and release-gate
  summary validators, generated release-state text/json references, docs, and
  script tests from the root v0.3.0 gate path to the canonical versioned script.
- Deleted the root v0.3.0 gate wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestReleaseV030GateUsesDedicatedV030Boundary -count=1` failed RED before the move,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV030' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestCIWorkflowIncludesCanonicalV030ReleaseGateJob|TestReleaseV030' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-state -run TestReleaseState -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-gate-summary -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-state -count=1`,
  `bash -n scripts/release/v0_3_0/gate.sh scripts/release/v0_3_0/security-review.sh`,
  `bash scripts/release/v0_3_0/gate.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3q: Versioned Release v0.4.0 Gate Migration

**Goal:** Move the v0.4.0 release gate implementation into the versioned
release directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Updated `TestReleaseV040GateUsesDedicatedReadinessPreflight` so
  `scripts/release/v0_4_0/gate.sh` owns the v0.4.0 release gate implementation
  and the removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root v0.4.0
  gate wrapper must not exist, and callers must use
  `scripts/release/v0_4_0/gate.sh` directly.
- Moved the v0.4.0 gate implementation into
  `scripts/release/v0_4_0/gate.sh`.
- Updated the gate's nested repo-root resolution, release command identity,
  fake blocked-readiness repos, security-review template text, release-state
  validator expectations, README/current surface guards, docs, and script tests
  from the root v0.4.0 gate path to the canonical versioned script.
- Deleted the root v0.4.0 gate wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestReleaseV040GateUsesDedicatedReadinessPreflight -count=1` failed RED before the move,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV040Gate|TestReleaseV040SecurityReview|TestSecurityReview' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestCurrentSupportedSurfaceDocumentIsReleaseAligned|TestReleaseV040Gate|TestReleaseV040SecurityReview|TestSecurityReview' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-state -run TestReleaseState -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-gate-summary -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-state -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/cmd/validate-release-gate-summary -count=1`,
  `bash -n scripts/release/v0_4_0/gate.sh scripts/release/v0_4_0/security-review.sh`,
  `bash scripts/release/v0_4_0/gate.sh --help`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3r: Versioned Release v0.5 Gate Migration

**Goal:** Move the v0.5 release gate implementation into the versioned release
directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Updated `TestReleaseV05GateValidatesJSONReports` so
  `scripts/release/v0_5/gate.sh` owns the v0.5 release gate implementation and
  the removed root wrapper path is rejected.
- Observed the expected RED failure before deleting the wrapper: the root v0.5
  gate wrapper must not exist, and callers must use
  `scripts/release/v0_5/gate.sh` directly.
- Moved the v0.5 gate implementation into `scripts/release/v0_5/gate.sh`.
- Added nested repo-root resolution to the versioned v0.5 gate and updated docs
  and script tests from the root v0.5 gate path to the canonical versioned
  script.
- Deleted the root v0.5 gate wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestReleaseV05GateValidatesJSONReports -count=1` failed RED before the move,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestReleaseV05GateValidatesJSONReports -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `bash -n scripts/release/v0_5/gate.sh`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3s: Versioned Release v0.6 Gate Migration

**Goal:** Move the v0.6 release gate implementation into the versioned release
directory and remove its root compatibility wrapper.

**2026-05-08 execution evidence:**

- Updated the v0.6 release gate static assertions to read
  `scripts/release/v0_6/gate.sh` through `readReleaseV06GateScript` and reject
  the removed root wrapper path.
- Observed the expected RED failure before deleting the wrapper: the root v0.6
  gate wrapper must not exist, and callers must use
  `scripts/release/v0_6/gate.sh` directly.
- Moved the v0.6 gate implementation into `scripts/release/v0_6/gate.sh`.
- Added nested repo-root resolution to the versioned v0.6 gate and updated docs
  and script tests from the root v0.6 gate path to the canonical versioned
  script.
- Updated `TestShellScriptsUsePortableBashSafetyHeader` to scan nested
  canonical script directories recursively now that root `scripts/*.sh`
  entrypoints are gone.
- Deleted the root v0.6 gate wrapper.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestReleaseV06GateValidatesHostSmokeReport -count=1` failed RED before the move,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV06Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestShellScriptsUsePortableBashSafetyHeader|TestReleaseV06Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `bash -n scripts/release/v0_6/gate.sh`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 3t: Root Script Residual Reference Audit

**Goal:** Verify that removed root script entrypoints no longer have live
non-generated callsites, and document intentional historical exceptions.

**2026-05-08 execution evidence:**

- Verified `find scripts -maxdepth 1 -type f -print | sort` returns only
  `scripts/README.md`; no root-level shell entrypoints remain.
- Verified old root script references outside `tools/scriptstest/**`,
  `docs/generated/**`, and this plan no longer point at live callsites. The
  remaining script-side matches are diagnostic labels such as
  `release_v0_3_0_gate:` and `release_v0_3_0_security_review:`, not executable
  paths.
- Intentional exception: `tools/scriptstest/**` keeps legacy path strings only
  as negative guards that fail if removed root entrypoints reappear. Owner:
  Release Engineering. Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`.
- Intentional exception: `docs/generated/v1_0/{release-state.*,release_gate_summary.*,test_all_full_summary.*,test-all/summary.*}`
  preserves reviewed historical command strings from older release evidence
  snapshots, including removed root entrypoints. Owner: Release Engineering.
  Reason: these files are tracked release evidence snapshots, not executable
  callsites; `docs/generated/v1_0/README.md` identifies the directory as a
  compatibility snapshot and the canonical gate archive as the source of truth.
  Verification:
  `rg -n "scripts/(release_[A-Za-z0-9_]+|test_all|test|bootstrap|format|dump|fuzz_nightly)\\.sh|bash scripts/(release_[A-Za-z0-9_]+|test_all|test|bootstrap|format|dump|fuzz_nightly)\\.sh|release_v[0-9_]+_.*\\.sh" docs/generated/v1_0`.
- Verification:
  `rg -n "scripts/(release_[A-Za-z0-9_]+|test_all|test|bootstrap|format|dump|fuzz_nightly)\\.sh|bash scripts/(release_[A-Za-z0-9_]+|test_all|test|bootstrap|format|dump|fuzz_nightly)\\.sh|release_v[0-9_]+_.*\\.sh" --glob '!graphify-out/**' --glob '!tools/scriptstest/**' --glob '!docs/generated/**' --glob '!docs/plans/2026-05-08-global-architecture-refactor-plan.md'`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 4: Release Script Test Fixture Split

**Goal:** Separate fake repo/report fixtures from assertions in
the historical monolithic v1 release script test file.

**Verification:** Focused release script tests and `go test ./tools/scriptstest -count=1`.

**2026-05-08 execution evidence:**

- Added `tools/scriptstest/release_v1_structure_test.go` as the fixture split
  guard.
- Moved `releaseV030FakeRepo`, `runReleaseV030Gate`,
  `releaseV030RunnableFakeRepo`, `runReleaseV030RunnableGate`,
  `runReleaseV030RunnableGateWithEnv`, `envHasPrefix`,
  `filteredReleaseV030GateEnv`, and
  `writeReleaseV030RuntimeSmokeReports`, and
  `installReleaseV030SummaryEchoingGo`, and
  `installReleaseV030CanonicalArtifactGo`, and
  `installReleaseV030CIMissingSignoffFailingFinalArtifactHashGo`, and
  `installReleaseV030FailingFinalArtifactHashGo`, and
  `installReleaseV030FailingSecurityReviewSha256`, and
  `installReleaseV030PortablePythonCanonicalizers` to
  `tools/scriptstest/release_v030_fixtures_test.go`.
- Moved `releaseV10GateFakeRepo`, `releaseV10WASISmokeFakeRepo`,
  `releaseV10WebSmokeFakeRepo`, `writeToolWrapper`,
  `writeReleaseV10FakeBrowser`, `runReleaseV10WebSmoke`, and
  `readWebSmokeReport` to `tools/scriptstest/release_v10_fixtures_test.go`.
- Moved shared `shellSingleQuote` to
  `tools/scriptstest/release_helpers_test.go`.
- Moved v1.0 web smoke assertion tests to
  `tools/scriptstest/release_v10_web_smoke_test.go`.
- Moved v1.0 WASI smoke assertion tests to
  `tools/scriptstest/release_v10_wasi_smoke_test.go`.
- Moved v1.0 gate assertion tests to
  `tools/scriptstest/release_v10_gate_test.go`.
- Moved v1.0 policy/default report assertion tests to
  `tools/scriptstest/release_v10_policy_test.go`.
- Moved v0.1.1 gate assertion tests to
  `tools/scriptstest/release_v011_gate_test.go`.
- Moved v0.1.2 gate assertion tests to
  `tools/scriptstest/release_v012_gate_test.go`.
- Moved v0.1.3 gate assertion tests to
  `tools/scriptstest/release_v013_gate_test.go`.
- Moved v0.2.0 gate assertion tests to
  `tools/scriptstest/release_v020_gate_test.go`.
- Moved static v0.3.0 gate/checklist/security wrapper assertion tests to
  `tools/scriptstest/release_v030_gate_static_test.go`.
- Moved v0.3.0 evidence/preflight assertion tests and their local
  `finalReleaseStateRefreshFollowsSummary` helper to
  `tools/scriptstest/release_v030_gate_evidence_test.go`.
- Moved v0.3.0 stale report-dir rejection tests to
  `tools/scriptstest/release_v030_gate_report_dir_test.go`.
- Moved v0.3.0 residual-risk assertion tests to
  `tools/scriptstest/release_v030_gate_residual_risks_test.go`.
- Moved v0.3.0 security-signoff policy assertion tests to
  `tools/scriptstest/release_v030_gate_security_signoff_test.go`.
- Moved v0.3.0 security-signoff acceptance and detached-hash assertion tests to
  `tools/scriptstest/release_v030_gate_security_signoff_acceptance_test.go`.
- Moved v0.3.0 final-summary/artifact-hash assertion tests and
  `countReleaseGateFailedSteps` to
  `tools/scriptstest/release_v030_gate_final_summary_test.go`.
- Moved v0.3.0 runtime-smoke evidence/archiving assertion tests to
  `tools/scriptstest/release_v030_gate_runtime_smoke_test.go`.
- Moved v0.3.0 runtime-smoke schema/type validation assertion tests to
  `tools/scriptstest/release_v030_gate_runtime_smoke_schema_test.go`.
- Moved v0.4.0 release gate readiness assertion tests to
  `tools/scriptstest/release_v040_gate_test.go`.
- Moved the current surface doc alignment guard to
  `tools/scriptstest/release_current_surface_test.go`.
- Moved the bootstrap binary alias guard to
  `tools/scriptstest/release_bootstrap_test.go`.
- Removed the historical monolithic v1 release script test file after all tests and helpers
  were moved behind `tools/scriptstest/release_v1_structure_test.go`.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestReleaseV1TestsAreSplitByFixtureDomain -count=1` failed RED before the move,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030GateRejectsExistingReportArtifacts|TestReleaseV030GateRejectsSymlinkToExistingReportArtifacts|TestReleaseV030GateRejectsDashPrefixedExistingReportArtifacts' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030RunnableGateFiltersAmbientResidualRisksEnv' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030RunnableGateFiltersAmbientResidualRisksEnv|TestReleaseV030GateRejectsBuildOnlyRuntimeSmokeEvidence' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030GateRejectsBuildOnlyRuntimeSmokeEvidence|TestReleaseV030GateRejectsWrongVersionRuntimeSmokeEvidence|TestReleaseV030GateRejectsMissingRequiredRuntimeSmokeCase' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030GateRejectsUntriagedUnstableSeeds|TestReleaseV030GateAcceptsTriagedUnstableSeeds|TestReleaseV030GateRejectsBuildOnlyRuntimeSmokeEvidence' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV10GateRunsDedicatedV1Workflow' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV10WASISmokeRunsUnifiedCLIAndValidatesReport' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|Test_release_v1_0_web_smoke(DiscoversGoogleChromeFallback|BrowserArgOverridesDiscovery|MissingExplicitBrowserWritesBlockedReport|ValidateWASMImportsFailureWritesStructuredFailReport)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030GateCIModeAllowsMissingSecuritySignoffWithArtifact|TestReleaseV030GateRefreshesReleaseStateAfterFinalSummary' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030Gate(CIMissingSignoffWritesDetachedHashOutsideCanonicalManifest|AcceptsSameRunSecuritySignoffWithFreshReportArtifacts|AcceptsSecuritySignoffPathStartingWithDash|WritesDetachedSecurityReviewHashOutsideCanonicalManifest|BlocksFinalSummaryWhenDetachedSecurityHashFails|CanonicalizesArtifactManifestWithPython3WhenPythonIsUnavailable)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030GateRecordsCIMissingSignoffFinalArtifactHashRefreshFailure' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030GateBlocksFinalSummaryWhenPostSummaryArtifactHashCheckFails' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030GateBlocksFinalSummaryWhenDetachedSecurityHashFails' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030GateCanonicalizesArtifactManifestWithPython3WhenPythonIsUnavailable' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030GateDoesNotArchivePartialRuntimeEvidenceWhenRuntimeCopyFails|Test_release_v1_0_web_smokeDiscoversGoogleChromeFallback' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV10WebSmokeScript|Test_release_v1_0_web_smoke' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV10WASISmoke' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV10Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV10SmokeScriptsHaveDefaultReportPaths|TestRoadmapV10RecordsExplicitCompatibilityAndSafetyPolicy' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV011Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV012Gate|TestReleaseV013Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV013Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV020Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030(GateUsesDedicatedV030Boundary|ChecklistIsNonClaimingAndVersionScoped|ChecklistAndGateRequireSecuritySignoff|GateGoTestStepUnsetsReleaseInputEnv|SecurityReviewWrapperUsesV030Name)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030Gate(RequireCleanRejectsDirtyWorktree|ValidatesFuzzArtifactsAfterShortFuzz|RefreshesReleaseStateAfterFinalSummaryWrite|WritesBlockedReleaseStateBeforeCIMissingSignoffExit|ValidatesGateSummaryArtifacts|HashesEntireReportDirectory)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030GateRejects(ExistingReportArtifacts|SymlinkToExistingReportArtifacts|DashPrefixedExistingReportArtifacts)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030(GateRejectsUntriagedUnstableSeeds|GateAcceptsTriagedUnstableSeeds|GateWritesResidualRisksJSONArtifact|GateAcceptsResidualRisksSourcePathStartingWithDash|GateRejectsUnownedHighMediumResidualRisk|GateRejectsResidualRisksJSONForWrongReleaseVersion|GateRejectsMalformedResidualRisksJSON|GateRejectsNullResidualRisksArray|GateRejectsResidualRiskMissingRequiredFields|RunnableGateFiltersAmbientResidualRisksEnv)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030Gate(CIModeAllowsMissingSecuritySignoffWithArtifact|CIMissingSignoffWritesDetachedHashOutsideCanonicalManifest|RequiresSecuritySignoffOutsideCIMode|RequireCleanRequiresSecuritySignoffEvenInCIMode)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030Gate(AcceptsSameRunSecuritySignoffWithFreshReportArtifacts|AcceptsSecuritySignoffPathStartingWithDash|WritesDetachedSecurityReviewHashOutsideCanonicalManifest)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030Gate(BlocksFinalSummaryWhenPostSummaryArtifactHashCheckFails|BlocksFinalSummaryWhenDetachedSecurityHashFails|RecordsCIMissingSignoffFinalArtifactHashRefreshFailure|CanonicalizesArtifactManifestWithPython3WhenPythonIsUnavailable|RefreshesReleaseStateAfterFinalSummary)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030Gate(RejectsBuildOnlyRuntimeSmokeEvidence|RejectsWrongVersionRuntimeSmokeEvidence|DoesNotArchivePartialRuntimeEvidenceWhenWindowsReportInvalid|DoesNotArchivePartialRuntimeEvidenceWhenRuntimeCopyFails|AcceptsRuntimeSmokeSourcePathStartingWithDash)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV030GateRejects(WrongGitHeadRuntimeSmokeEvidence|RunnerRuntimeSmokeEvidence|InvalidTimestampRuntimeSmokeEvidence|LooseTimestampRuntimeSmokeEvidence|MissingRequiredRuntimeSmokeCase|RuntimeSmokeCaseErrorText|EmptyRuntimeSmokeCaseName|NonStringRuntimeSmokeCaseName|NonBooleanRuntimeSmokeCaseStatus|NonIntegerRuntimeSmokeCaseExitFields|NonIntegerRuntimeSmokeCounts|NonBooleanRuntimeSmokeIslandsDebug|NonBooleanRuntimeSmokeBuildOnly|NonStringRuntimeSmokeRunner)' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestReleaseV040Gate' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestCurrentSupportedSurfaceDocumentIsReleaseAligned|TestBootstrapBuildsTetraAndTAlias' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `git diff --check -- tools/scriptstest/release_v011_gate_test.go tools/scriptstest/release_v030_fixtures_test.go tools/scriptstest/release_v10_fixtures_test.go tools/scriptstest/release_helpers_test.go tools/scriptstest/release_v10_gate_test.go tools/scriptstest/release_v10_policy_test.go tools/scriptstest/release_v10_web_smoke_test.go tools/scriptstest/release_v10_wasi_smoke_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_v012_gate_test.go tools/scriptstest/release_v013_gate_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_v013_gate_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_v020_gate_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_v030_gate_static_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_v030_gate_evidence_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_v030_gate_report_dir_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_v030_gate_residual_risks_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_v030_gate_security_signoff_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_v030_gate_security_signoff_acceptance_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_v030_gate_final_summary_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_v030_gate_runtime_smoke_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_v030_gate_runtime_smoke_schema_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_v040_gate_test.go tools/scriptstest/release_v1_structure_test.go`,
  `git diff --check -- tools/scriptstest/release_current_surface_test.go tools/scriptstest/release_bootstrap_test.go tools/scriptstest/release_v1_structure_test.go`,
  and `graphify update .`.

### Slice 4a: Scriptstest Shared Helper Extraction

**Goal:** Remove shared helper ownership from the large `test_all_test.go`
surface so release and script fake repos depend on a dedicated helper file.

**2026-05-08 execution evidence:**

- Extended `TestReleaseV1TestsAreSplitByFixtureDomain` so
  `tools/scriptstest/release_helpers_test.go` owns `copyFile` and `repoRoot`,
  not `tools/scriptstest/test_all_test.go`.
- Observed the expected RED failure before the move:
  `release_helpers_test.go must contain copyFile`.
- Moved `copyFile` and `repoRoot` into
  `tools/scriptstest/release_helpers_test.go` without changing their signatures.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestReleaseV1TestsAreSplitByFixtureDomain -count=1` failed RED before the move,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestTestAllQuickJSONIncludesStepExitCodes|TestReleaseV030GateRejectsExistingReportArtifacts|TestReleaseV10GateUsesRealV1Boundary' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 4b: Scriptstest Test-All Helper Extraction

**Goal:** Remove test-all run/read/summary helper ownership from the large
`test_all_test.go` assertion surface.

**2026-05-08 execution evidence:**

- Extended `TestReleaseV1TestsAreSplitByFixtureDomain` so
  `tools/scriptstest/test_all_helpers_test.go` owns `hasTestAllStep`,
  `testAllStepLog`, `readTestAllScript`, `readReleaseV06GateScript`,
  `runTestAll`, `runTestAllSplit`, `runTestAllFromWorkingDir`, and
  `decodeTestAllSummary`.
- Observed the expected RED failure before the move:
  `read test_all_helpers_test.go: no such file or directory`.
- Moved those helpers into `tools/scriptstest/test_all_helpers_test.go` without
  changing signatures or callsites.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestReleaseV1TestsAreSplitByFixtureDomain -count=1` failed RED before the move,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestTestAllQuickJSONIncludesStepExitCodes|TestTestAllFullAggregatesToolingSummary|TestTestAllRunsFromNestedWorkingDirectory' -count=1`,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -count=1`,
  `git diff --check -- ...`,
  `graphify update .`.

### Slice 4c: Scriptstest Test-All Summary Helper Ownership

**Goal:** Make the test-all helper file own its summary schema and formatter
step name constant so the large `test_all_test.go` file keeps only assertions.

**2026-05-08 execution evidence:**

- Extended `TestReleaseV1TestsAreSplitByFixtureDomain` so
  `tools/scriptstest/test_all_helpers_test.go` owns `testAllSummary` and
  `testAllFormatterStepName`.
- Observed the expected RED failure before the move:
  `test_all_helpers_test.go must contain type testAllSummary struct`.
- Moved `testAllSummary` and `testAllFormatterStepName` into
  `tools/scriptstest/test_all_helpers_test.go` without changing their callers.
- Verification:
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run TestReleaseV1TestsAreSplitByFixtureDomain -count=1` failed RED before the move,
  `GOCACHE=/tmp/tetra-language-go-cache go test ./tools/scriptstest -run 'TestReleaseV1TestsAreSplitByFixtureDomain|TestTestAllQuickJSONIncludesStepExitCodes|TestTestAllFormatterCoversRuntimeSources' -count=1`.

## Completion Criteria

This global goal is complete only after a final completion audit shows:

- each hotspot row is either refactored or documented with an intentional
  boundary and owner,
- no public behavior changed without tests and docs,
- focused and relevant package tests pass for every touched area,
- required docs/release/tool validators pass,
- Graphify artifacts are fresh after code changes,
- dirty worktree scope is reviewed so unrelated changes are not claimed,
- no generated artifact churn is accidental, and
- every explicit objective requirement maps to concrete files and command output.
