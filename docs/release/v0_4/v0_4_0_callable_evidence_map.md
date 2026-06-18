# Tetra v0.4.0 Callable Evidence Map

Status: evidence map for the callable production epic. Callable Level 1,
Callable Level 2, and safe full first-class callables are promoted in the
v0.4.0 feature registry.

This file records the inspected evidence for the selected callable scope in
`docs/spec/v0_4_scope.md`. It records the evidence that allows
`language.callable-level1`, `language.callable-level2`, and
`language.full-first-class-callables` to move to `current` under the v0.4.0
release boundary while keeping unsafe mutable/resource/thread escapes explicit.

## Selected Callable Scope

The `v0.4.0` scope selects all callable gaps for implementation:

- `language.callable-level1`
- `language.callable-level2`
- `language.full-first-class-callables`

The current feature registry reports:

- `language.callable-mvp`: `current`
- `language.callable-level1`: `current` since `v0.4.0`
- `language.callable-level2`: `current` since `v0.4.0`
- `language.full-first-class-callables`: `current` since `v0.4.0`

## Existing Evidence

The repository already has real implementation and tests for the current
constrained callable MVP and several non-capturing symbol-backed paths:

- `compiler/tests/callables/function_typed_callable_test.go`
- `compiler/tests/semantics/closures_semantic_clauses_test.go`
- `compiler/internal/lower/callable_test.go`
- `compiler/tests/lowering/lowering_public_api_test.go`
- `compiler/tests/lowering/wasm_backend_mvp_test.go`
- `compiler/tests/safety/effects_test.go`
- `compiler/tests/safety/epic06_safety_test.go`

Covered behavior includes:

- function type parsing/checking
- non-capturing function-typed local binding
- non-capturing generic closure literals may initialize declared function-typed
  locals when every type parameter is inferred from the declared
  `fn(...) -> ...` parameter and return types; generic closure captures remain
  rejected with a stable diagnostic
- non-capturing generic closure literals may be passed as direct callback
  arguments when every type parameter is inferred from the callee's
  function-typed callback parameter; generic closure captures remain rejected
  with a stable diagnostic
- non-capturing generic closure literals may initialize function-typed struct
  fields, including nested struct literal initializers, when every type
  parameter is inferred from the field's declared `fn(...) -> ...` type; generic
  closure captures remain rejected with a stable diagnostic
- non-capturing generic closure literals may initialize function-typed enum
  payloads when every type parameter is inferred from the payload's declared
  `fn(...) -> ...` type; generic closure captures remain rejected with a stable
  diagnostic
- non-capturing generic closure literals may reassign mutable enum payload
  values when every type parameter is inferred from the payload's declared
  `fn(...) -> ...` type; generic closure captures remain rejected with a stable
  diagnostic
- non-capturing generic closure literals may be returned directly from
  function-typed return paths when every type parameter is inferred from the
  declared return `fn(...) -> ...` type; generic closure captures remain
  rejected with a stable diagnostic
- non-capturing generic closure literals may be assigned into mutable
  function-typed locals when every type parameter is inferred from the target's
  declared `fn(...) -> ...` type; generic closure captures remain rejected with
  a stable diagnostic
- non-capturing generic closure literals may be assigned into mutable
  function-typed struct fields, including nested local field paths such as
  `box.holder.cb`, when every type parameter is inferred from the target
  field's declared `fn(...) -> ...` type; generic closure captures remain
  rejected with a stable diagnostic
- generic closure literals outside the supported direct-call shape now report
  the generic direct-call closure ABI boundary: the closure must be let-bound,
  called locally and directly, and have inferable concrete arguments
- same-module or imported generic function symbols may initialize declared
  function-typed locals when every type parameter is inferred from the declared
  `fn(...) -> ...` parameter and return types; the monomorphizer rewrites the
  initializer to the concrete specialization before semantic checking and
  lowering
- same-module or imported generic function symbols may be passed as direct
  callback arguments when every type parameter is inferred from the callee's
  function-typed callback parameter; the monomorphizer rewrites the argument to
  the concrete specialization before semantic checking and lowering
- same-module or imported generic function symbols may be returned directly
  from function-typed return paths when every type parameter is inferred from
  the declared return
  `fn(...) -> ...` type; the monomorphizer rewrites the returned value to the
  concrete specialization before semantic checking and lowering
- same-module or imported generic function symbols may be assigned into mutable
  function-typed locals or mutable function-typed nested struct fields,
  including imported generic symbols on nested local field paths such as
  `box.holder.cb` and imported type-only modules that provide the nested struct
  declarations, when every type parameter is inferred from the target's
  declared `fn(...) -> ...` type; the monomorphizer rewrites the assignment
  value to the concrete specialization before semantic checking and lowering,
  and object generation skips imported dependency modules that contain no
  runtime functions, including type-only and unused global-only modules
- same-module or imported generic function symbols may initialize
  function-typed struct fields, including nested struct literal initializers,
  when every type parameter is inferred from the field's declared
  `fn(...) -> ...` type; the monomorphizer rewrites the field initializer to the
  concrete specialization before semantic checking and lowering
- same-module or imported generic function symbols may initialize
  function-typed enum payloads when every type parameter is inferred from the
  payload's declared `fn(...) -> ...`
  type; the monomorphizer rewrites the payload initializer to the concrete
  specialization before semantic checking and lowering
- same-module or imported generic function symbols may reassign mutable enum
  payload values when every type parameter is inferred from the payload's
  declared `fn(...) -> ...` type; the monomorphizer rewrites the payload
  reassignment to the concrete specialization before semantic checking and
  lowering
- function-typed local aliases, including target-set-backed aliases of
  function-typed parameters and snapshot copies from mutable function-typed
  locals into immutable aliases with dynamic dispatch over known target sets
- function-typed parameter storage into local struct fields with direct field
  calls and synchronous callback arguments, backed by propagated call-site
  target sets and dependency collection that skips those field calls as external
  function symbols
- function-typed parameter storage into enum payloads with direct payload calls,
  mutable local enum reassignment, returned enum propagation, and synchronous
  callback arguments backed by propagated call-site target sets
- signature-compatible mutable local reassignment among supported
  function-typed values, including direct calls and callback arguments after
  reassignment, assignment from known function-typed returns including
  same-module or imported target-set-backed parameter-return calls such as
  `identity(captured)` or `callbacks.identity(captured)`,
  multi-target return target sets with mutable-global-target classification, closure literals, and
  immutable
  function-typed struct fields; imported parameter-return reassignment keeps
  captured `fnptr` metadata for subsequent local direct calls and diagnostics
  at later global escape boundaries
- direct named function/closure symbol callbacks
- optional argument labels on function-typed value calls, including local
  callback parameters, captured `fnptr` locals, and function-typed struct
  fields, function-typed enum-payload bindings, and function-typed globals;
  labels are accepted as call-site documentation because function type syntax
  does not carry parameter names, while positional type and ownership checks
  still apply; mixed labeled/unlabeled argument lists are rejected with stable
  diagnostics for callback, function-typed struct-field, and function-typed
  global calls
- direct function-typed struct-field calls enforce positional ownership markers
  (`borrow`, `consume`, `inout`) with the same aliasing and mutability checks as
  local function-typed callback calls
- direct function-typed enum-payload calls enforce positional ownership markers
  (`borrow`, `consume`, `inout`) through the same pattern-bound callback path,
  including aliasing and mutability checks
- function-typed globals initialized from same-module/imported direct named
  function symbols, inferable same-module/imported generic function symbols, or
  non-capturing closure literals may be called directly; public immutable
  function-typed globals imported through a namespace alias or selective import
  may also be called directly across the module boundary; namespace and
  selective imports are both covered for local initialization, mutable local
  reassignment, local struct-field reassignment, and direct synchronous
  callback arguments; their declared
  function type is checked against the target symbol at the call site, including
  argument count, positional type checks, positional ownership markers, and
  semantic-clause/effect compatibility with user-visible global-call
  diagnostics, and explicit type arguments are rejected against the
  user-visible global callable name; the
  same symbol-backed global values may
  initialize local function-typed values or reassign mutable local
  function-typed values, reassign supported local struct fields, or be passed
  directly as synchronous callback arguments with stable dynamic target-set
  propagation; dependency collection follows the
  backing function symbol rather than the imported global pseudo-symbol;
  same-module mutable globals may
  be reassigned to compatible direct function symbols and then called directly
  or passed through synchronous callback arguments, returned from function-typed
  return paths, and their current fnptr value may be stored into local struct
  fields, nested local struct fields, or enum payloads, or reassigned through supported local
  struct-field/enum-payload paths, for supported direct calls or synchronous
  callback arguments, including through known returned struct fields or enum
  payloads; imported mutable function-typed globals are recognized as public
  callable globals but rejected with a stable cross-module global-data ABI
  diagnostic instead of falling through to `unknown function`; actor/task
  workers that directly dispatch through same-module mutable function-typed
  globals, imported immutable function-typed globals whose targets touch
  mutable globals, pass them as synchronous callback arguments, pass
  same-module or imported symbol-backed callback arguments whose targets touch
  mutable globals, pass same-module or imported direct function-typed
  return-call callback arguments whose returned targets or multi-return target
  sets touch mutable globals, preserve that classification through local/field
  alias returns and returned struct/enum aggregate fields or payloads across
  module boundaries, directly call function-typed locals/struct fields/enum
  payloads whose targets touch mutable globals, reassign them into
  function-typed locals or local struct fields/enum payloads, store them into
  local function-typed struct fields/enum payloads, return them from
  function-typed return helpers, or write mutable function-typed globals are rejected as
  mutable-global boundary crossings for `core.spawn`,
  `core.task_spawn_i32`, `core.task_spawn_i32_typed`,
  `core.task_spawn_group_i32`, and `core.task_spawn_group_i32_typed`; captured values
  and arbitrary global function value escape remain outside this slice.
  `TestReleaseTraceabilityCrossModuleImmutableCallableGlobalMutableTargetBoundary`
  covers task, typed-task, task-group, typed-task-group, and actor boundary diagnostics for
  imported immutable function-typed globals with mutable-global targets,
  including direct calls and synchronous callback arguments.
  `TestReleaseTraceabilityCrossModuleReturnedAggregateCallableMutableTargetBoundary`
  covers task, typed-task, task-group, typed-task-group, and actor boundary diagnostics for
  imported returned structs and enums that carry callable fields or payloads
  whose known targets touch mutable globals, including the full
  `*_returned_enum_payload_direct_closure` task, typed-task, task-group,
  typed-task-group, and actor boundary matrix where an imported returned enum
  payload carries a direct closure that mutates module global state.
  `TestReleaseTraceabilityCrossModuleCallableMutableTargetBoundary` covers the
  same task, typed-task, task-group, typed-task-group, and actor boundary set for direct imported
  function-typed return-call callback arguments
  `TestReleaseTraceabilityLifetimeAndRaceSafetyNegativeActorTaskOwnership` also
  pins the same-module returned enum payload direct-closure cases
  `RaceSafetyRejectsTaskTargetReturnedEnumPayloadDirectClosureWithMutableGlobalTarget`,
  `RaceSafetyRejectsTypedTaskTargetReturnedEnumPayloadDirectClosureWithMutableGlobalTarget`,
  `RaceSafetyRejectsTaskGroupTargetReturnedEnumPayloadDirectClosureWithMutableGlobalTarget`,
  `RaceSafetyRejectsTypedTaskGroupTargetReturnedEnumPayloadDirectClosureWithMutableGlobalTarget`,
  and
  `RaceSafetyRejectsActorTargetReturnedEnumPayloadDirectClosureWithMutableGlobalTarget`,
  where the returned payload closure itself mutates global state before a worker
  can cross a task, typed-task, task-group, typed-task-group, or actor boundary
- symbol-backed function-typed returns, including direct named function symbols
  and supported symbol-backed aliases
- captured closure literals assigned to function-typed locals and called
  directly, with captures materialized as `fnptr` environment slots
- captured `ptr` closure direct calls may use labels for explicit closure
  parameters; hidden capture arguments are appended internally with matching
  capture labels before ordinary positional/type/effect validation, and mixed
  labeled/unlabeled calls are rejected against the user-visible captured
  closure local rather than the synthetic closure symbol; explicit type
  arguments on captured closure direct calls are also rejected before the call
  is rewritten to its synthetic closure symbol
- let-bound captured `ptr` closure values may alias into function-typed locals
  or reassign compatible mutable function-typed locals, store into local
  struct fields or enum payloads, return directly from function-typed return
  paths, and pass as direct synchronous callback arguments when their by-value
  environment fits the eight-slot `fnptr` envelope; the same `fnptr` value may
  cross an imported function-typed parameter-return boundary such as
  `identity(cb) -> cb` without losing the caller module's synthetic closure
  symbol
- direct captured closure literals, let-bound captured `ptr` closure locals,
  direct same-module/imported function-typed return calls, immutable local
  aliases initialized from those return calls, mutable function-typed locals,
  local/nested struct fields, local enum payloads, whole local or nested
  structs with function fields reassigned from struct literals containing
  direct closure literals or direct return calls, or whole local enums reassigned
  from enum constructors containing direct closure literals or direct return calls, or return alias
  chains that return captured closure snapshots assigned
  into same-module mutable global
  function-typed values are stored as bounded by-value `fnptr`
  snapshots and may be called later through that
  global, passed as synchronous callback arguments, returned from same-module
  or imported functions, passed as callback arguments or reassigned into
  mutable locals after cross-module returns, stored in local
  struct fields or enum payloads, or dispatched through `try cb(...)` when the
  global type declares the same throws type;
  captured `fnptr` values reached through mutable function-typed
  whole-struct reassignments not backed by direct closure or direct return-call
  field initializers, including unsupported assignment sources or parameter
  escapes, remain outside the current production claim;
  function-typed
  parameters also cannot be stored into mutable global function-typed values and
  report a dedicated parameter-to-global escape diagnostic even when first
  routed through local aliases, mutable local reassignments, direct same-module
  or imported function-typed return calls, helper return aliases, helper
  struct-field returns, local struct fields, enum payload bindings,
  same-module returned struct fields, same-module or imported returned nested
  struct field paths, same-module or imported whole struct-parameter returns,
  same-module or imported whole enum-parameter returns, or same-module or
  imported returned enum payloads,
  and captured values passed through direct, inline, imported source, or
  generated `.t4i` interface-only function-typed parameter-return calls such
  as `identity(f) -> f`, through
  same-module, imported source, or generated `.t4i` interface-only
  struct-parameter field returns such as `pick(holder) -> holder.cb` and
  nested paths such as
  `pick(box) -> box.holder.cb`, through same-module, imported source, or
  generated `.t4i` interface-only whole struct-parameter returns such as
  `echo(box) -> box` that preserve nested function-field target sets, through
  same-module or imported enum-parameter payload returns or whole generated
  `.t4i` interface-only enum-parameter returns such as
  enum-parameter returns such as `echo(choice) -> choice`, including inline imported struct/enum
  constructors carrying
  captured closure literals, with those returned captured `fnptr` values usable
  for local direct calls or direct synchronous callback arguments, or through
  direct function-typed returns through local struct-field aliases or
  reassignments, enum-payload bindings or reassignments, returned struct fields
  including nested paths, and enum payloads built from
  function-typed parameters, local aliases of those parameters, or local
  struct-field aliases carrying those parameters, including returned structs
  such as `makeBox(f) -> Box(choice: MaybeCallback.some(f))`, are rejected at
  the global assignment boundary. This keeps local
  `fnptr` environments from escaping without a heap/lifetime model
- direct captured closure-literal callback arguments lowered as nine-slot
  `fnptr` values, including imported callback callees where synthetic closure
  target symbols are qualified in the caller module before target-set dispatch
- handle-backed function-typed local callback arguments with larger captured
  environments lowered through the same fixed nine-slot callback ABI, covered
  by `TestBuildFullCallableLocalCallbackArgumentNineCaptureSmoke`
- cross-module returned nine-capture callable handles used through local calls,
  local struct fields, local enum payloads, direct callback arguments, and local
  return aliases, covered by
  `TestBuildFullCallableCrossModuleReturnedNineCaptureMatrixSmoke`
- generated `.t4i` stubs for direct returned function values preserve capture
  count, heap escape kind, handle flag, function target identity, and 4-slot
  return handle metadata for nine-capture closures, covered by
  `TestGenerateInterfaceFromSourcePreservesReturnedFunctionHandleMetadata`
- local aliases of larger captured closure handles are first-class values for
  function-typed returns, same-module mutable global snapshots, and synchronous
  callback arguments, covered by
  `TestBuildFullCallableReturnAliasTwelveCaptureSmoke`,
  `TestBuildFullCallableGlobalAliasTwelveCaptureSmoke`, and
  `TestBuildFullCallableCallbackAliasTwelveCaptureSmoke`
- mutable/resource escape diagnostics now include source-level heap mutable,
  heap resource, and global resource captured-callable rejections in
  `TestBuildFunctionTypedCallableMVPRejectsUnsupportedForms`, thread-boundary
  mutable/resource classifier diagnostics in
  `TestClassifyCallableEscapeRejectsMutableCaptureAcrossThreadBoundary` and
  `TestClassifyCallableEscapeRejectsResourceCaptureAcrossThreadBoundary`,
  plus existing imported mutable function-typed global ABI and unsupported
  generic callable movement diagnostics
- direct captured closure-literal struct-field and enum-payload initializers
  preserve module-qualified synthetic closure targets in function-field/payload
  metadata, including imported struct types and module-local enum payloads
- imported functions may accept structs with function-typed fields and dispatch
  through those fields when the caller passes a known local struct value or
  namespace/selective imported direct struct constructor carrying a direct
  closure literal or captured `ptr` closure local; target-set propagation maps the caller-side field
  metadata onto the callee's
  `param.field` stored-function call
- imported functions may accept enums with function-typed payloads and dispatch
  through pattern-bound payload callbacks when the caller passes a known local
  enum value, direct enum-returning call, or direct namespace/selective imported
  enum constructor argument carrying a captured closure target; target-set propagation maps the
  caller-side payload metadata onto the callee's enum parameter payload binding,
  and dependency collection treats namespace/selective imported enum
  constructors as type-level construction rather than external function symbols
- captured function-typed locals with up to eight by-value snapshot capture
  slots passed through synchronous function-typed callback parameters via the
  `fnptr` environment
- semantic clauses validate function-typed callback parameters and
  target-set-backed callback arguments against declared function-type effects
  when no single concrete callback symbol is available, while unsafe concrete
  callbacks, direct function-typed local calls, direct function-typed
  struct-field calls, and direct function-typed global calls are still rejected
  at call sites with user-visible callable diagnostics; function-typed local,
  struct-field, and immutable/mutable global callback arguments also report the
  visible argument name instead of the backing function symbol, and
  function-typed return-call callback arguments report the visible call form
  such as `pick()`; direct closure literal callback arguments report
  signature/effect/unsupported-throwing diagnostics and generic capture
  rejections as `closure literal`; unsupported callback argument source shapes
  now name the supported `fnptr` source forms instead of reporting an unnamed
  MVP subset
- captured function-typed locals with up to eight by-value snapshot capture
  slots returned from function-typed return paths, called directly after
  return, or passed through synchronous callback parameters
- function-typed local and direct callback closure literals validate exact
  parameter arity before lowering, so extra closure parameters cannot be hidden
  behind a narrower declared `fn(...) -> ...` ABI
- direct calls through function-typed locals, including captured `fnptr` locals,
  report unsupported explicit type arguments, wrong arity, type mismatches, and
  mixed labeled/unlabeled argument lists against the visible callback name; captured
  `fnptr` local semantic-clause violations use the same visible phrase, for
  example `function-typed callback 'f'`
- direct function-typed struct-field calls report wrong arity and explicit
  type-argument rejections against the visible field path, for example
  `function-typed struct field call 'holder.cb'`
- pattern-bound function-typed enum-payload calls report wrong arity, type
  mismatches, labels, explicit type-argument rejections, and ownership/aliasing
  diagnostics against the visible payload binding, for example
  `function-typed enum payload call 'cb'`; semantic-clause violations on direct
  enum-payload calls use the same visible binding phrase
- function-typed returns may collect multiple known return-path targets from
  direct symbols, local aliases, captured closure literals, or function-typed
  parameters and propagate them through direct local calls or synchronous
  callback arguments, and the same target sets may be stored into supported
  local struct fields or enum payload constructors or reassigned through
  supported mutable local, struct, and enum-payload paths, including across
  imported module boundaries
- function-typed return call expressions may be passed directly as synchronous
  callback arguments, including imported `math.pick(...)` style calls, when the
  returned `fn(...) -> ...` signature matches the callee's function-typed
  parameter; diagnostics name the visible return call, and lowering propagates
  the returned target set into the callee's callback parameter instead of
  requiring a temporary local alias; imported parameter-return calls such as
  `callbacks.identity(captured)` preserve captured `fnptr` metadata as direct
  callback arguments, while imported returns that ignore a captured callback and
  return a concrete symbol such as `add0` do not inherit the argument's captures
- direct captured closure-literal returns from function-typed return paths,
  lowered as nine-slot `fnptr` values with the same eight-slot environment
  limit; oversized direct closure-literal and named captured `ptr` closure
  returns report the concrete environment slot count at the return boundary.
  `TestBuildFunctionTypedCapturedClosureEightSlotReturnCrossModuleCallbackSmoke`
  exercises the maximum eight-capture payload as a cross-module function-typed
  return immediately consumed by a synchronous callback argument, proving the
  current nine-slot `fnptr` return ABI at the production boundary
- throwing function-typed values returned through supported aggregate shapes
  preserve their throws type and captured `fnptr` metadata for direct `try`
  dispatch. `TestBuildFunctionTypedThrowingReturnedStructFieldDirectTrySmoke`,
  `TestBuildFunctionTypedThrowingCapturedClosureReturnedStructFieldDirectTrySmoke`,
  `TestBuildFunctionTypedThrowingCapturedClosureReturnedEnumPayloadDirectTrySmoke`,
  `TestBuildFunctionTypedThrowingCapturedClosureReturnedStructFieldCrossModuleDirectTrySmoke`,
  and
  `TestBuildFunctionTypedThrowingCapturedClosureReturnedEnumPayloadCrossModuleDirectTrySmoke`
  pin same-module and source-imported returned struct-field/enum-payload
  direct-try dispatch for concrete throwing symbols and captured throwing
  closure literals
- direct function-typed return-call snapshots assigned into same-module mutable
  globals remain bounded through local aliases, mutable locals, local/nested
  struct fields, local enum payloads, whole local struct reassignments from
  struct literals, and whole local enum reassignments from enum constructors
  containing those return calls. The focused runtime and IR
  evidence is
  `TestBuildFunctionTypedCapturedClosureReturnWholeStructMutableGlobalReassignmentDirectCallSmoke`,
  `TestBuildFunctionTypedCapturedClosureReturnWholeNestedStructMutableGlobalReassignmentDirectCallSmoke`,
  `TestCapturedFunctionTypedReturnWholeStructReassignmentCanSnapshotIntoGlobalFunctionValue`,
  `TestCapturedFunctionTypedReturnNestedStructReassignmentCanSnapshotIntoGlobalFunctionValue`,
  and
  `TestLowerCallableCapturedReturnWholeStructGlobalAssignmentPropagatesTargetAcrossFuncsIR` /
  `TestLowerCallableCapturedReturnWholeNestedStructGlobalAssignmentPropagatesTargetAcrossFuncsIR`;
  the parameter-return guard tests keep function-typed parameter escapes
  rejected through the same container shapes
- direct captured closure literals stored first in local struct fields or enum
  payloads may also snapshot into same-module mutable function-typed globals
  when their environment fits the fixed `fnptr` envelope.
  `TestBuildFunctionTypedCapturedClosureStructFieldMutableGlobalSnapshotSmoke`,
  `TestBuildFunctionTypedCapturedClosureEnumPayloadMutableGlobalSnapshotSmoke`,
  `TestBuildFunctionTypedCapturedClosureWholeEnumMutableGlobalSnapshotSmoke`,
  `TestBuildFunctionTypedCapturedClosureWholeStructMutableGlobalSnapshotSmoke`,
  `TestBuildFunctionTypedCapturedClosureWholeNestedStructMutableGlobalSnapshotSmoke`,
  `TestCapturedFunctionTypedStructFieldCanSnapshotIntoGlobalFunctionValue`,
  `TestCapturedFunctionTypedEnumPayloadCanSnapshotIntoGlobalFunctionValue`,
  `TestCapturedFunctionTypedDirectClosureWholeEnumReassignmentCanSnapshotIntoGlobalFunctionValue`,
  `TestCapturedFunctionTypedDirectClosureWholeStructReassignmentCanSnapshotIntoGlobalFunctionValue`,
  `TestCapturedFunctionTypedDirectClosureWholeNestedStructReassignmentCanSnapshotIntoGlobalFunctionValue`,
  `TestLowerCallableCapturedStructFieldGlobalAssignmentPropagatesTargetAcrossFuncsIR`,
  `TestLowerCallableCapturedEnumPayloadGlobalAssignmentPropagatesTargetAcrossFuncsIR`,
  `TestLowerCallableCapturedWholeEnumGlobalAssignmentPropagatesTargetAcrossFuncsIR`,
  `TestLowerCallableCapturedWholeStructGlobalAssignmentPropagatesTargetAcrossFuncsIR`,
  and
  `TestLowerCallableCapturedWholeNestedStructGlobalAssignmentPropagatesTargetAcrossFuncsIR`
  pin the runtime, semantic, and IR evidence for these direct container
  snapshot paths
- same-module or source-imported returned enum-payload snapshots, either as a
  direct enum return or through a returned struct field, carrying direct closure
  literals may also be assigned into same-module mutable function-typed globals
  after local match binding.
  `TestBuildFunctionTypedCapturedClosureReturnedStructEnumPayloadMutableGlobalSnapshotSmoke`,
  `TestBuildFunctionTypedCapturedClosureReturnedEnumPayloadMutableGlobalSnapshotSmoke`,
  `TestBuildFunctionTypedCapturedClosureImportedReturnedStructEnumPayloadMutableGlobalSnapshotSmoke`,
  `TestBuildFunctionTypedCapturedClosureImportedReturnedEnumPayloadMutableGlobalSnapshotSmoke`,
  `TestImportedReturnedStructEnumPayloadDirectClosureMetadata`,
  `TestImportedReturnedEnumPayloadDirectClosureMetadata`,
  `TestInterfaceReturnedStructEnumPayloadInlineClosureMetadata`,
  `TestInterfaceReturnedEnumPayloadInlineClosureMetadata`,
  `TestInterfaceReturnedStructEnumPayloadInlineThrowingClosureMetadata`,
  `TestInterfaceReturnedStructFieldInlineThrowingClosureMetadata`,
  `TestInterfaceReturnedEnumPayloadInlineThrowingClosureMetadata`,
  `TestGenerateInterfaceFromSourcePreservesReturnedAggregateClosureStub`,
  `TestGenerateInterfaceFromSourcePreservesReturnedEnumClosureStub`,
  `TestGenerateInterfaceFromSourcePreservesReturnedThrowingAggregateClosureStub`,
  `TestGenerateInterfaceFromSourcePreservesReturnedThrowingStructFieldClosureStub`,
  `TestGenerateInterfaceFromSourcePreservesReturnedThrowingEnumClosureStub`,
  `TestBuildInterfaceOnlyModeReturnedAggregateClosurePayloadStub`,
  `TestBuildInterfaceOnlyModeReturnedEnumClosurePayloadStub`,
  `TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadStub`,
  `TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadRequiresTryDiagnostic`,
  `TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureStub`,
  `TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureRequiresTryDiagnostic`,
  `TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureCallbackStub`,
  `TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureCallbackThrowsMismatchDiagnostic`,
  `TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadStub`,
  `TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadRequiresTryDiagnostic`,
  `TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadCallbackStub`,
  `TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadCallbackStub`,
  `TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadCallbackThrowsMismatchDiagnostic`,
  `TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadCallbackThrowsMismatchDiagnostic`,
  `TestCapturedFunctionTypedReturnedStructEnumPayloadCanSnapshotIntoGlobalFunctionValue`,
  `TestCapturedFunctionTypedReturnedEnumPayloadCanSnapshotIntoGlobalFunctionValue`,
  `TestImportedCapturedFunctionTypedReturnedEnumPayloadCanSnapshotIntoGlobalFunctionValue`,
  and
  `TestLowerCallableCapturedReturnedEnumPayloadGlobalAssignmentPropagatesTargetAcrossFuncsIR` /
  `TestLowerCallableCapturedReturnedStructEnumPayloadGlobalAssignmentPropagatesTargetAcrossFuncsIR`
  pin the runtime, semantic, IR, generated `.t4i`, and interface-only evidence
  for this shape. The `.t4i` path preserves direct enum constructor and
  aggregate constructor shape plus function-payload metadata, including direct
  returned struct function fields and declared `throws` types on inline closure
  stubs for returned aggregate payloads, with semantics metadata preserving
  `FunctionThrowsType` through both return signatures and caller-side local
  struct fields for API-only validation without exposing the original closure
  body or captures
- `TestBuildFunctionTypedCapturedClosureEightSlotEnumReturnCrossModuleCallbackSmoke`
  exercises the corresponding enum-payload return path: enum tag plus the
  maximum eight-capture `fnptr` payload returns across a module boundary and is
  consumed through a synchronous callback, proving the current ten-slot native
  return layout for callable enum payloads
- mutable local `Int` captures are snapshotted by value at `fnptr` binding
  points for function-typed locals, immutable struct fields, and enum payloads
- `TestBuildFunctionTypedCapturedClosureCompositeCaptureMatrixCallbackSmoke`
  covers the current by-value capture matrix beyond `Int` by executing a
  function-typed callback that captures `Bool`, `String`, and a simple struct
  without pointer/resource fields
- `TestBuildFunctionTypedCallableMVPRejectsUnsupportedForms` now also covers
  boundary-specific unsupported capture diagnostics for direct closure-literal
  callback arguments, function-typed local storage, function-typed returns,
  struct-field storage, and enum-payload storage, including initializer and
  reassignment paths, when a captured simple-looking struct contains a `ptr`
  field. These cases name the concrete boundary and captured local instead of
  falling back to a generic closure-capture message
- `TestBuildFunctionTypedCrossModuleUnsupportedCaptureDiagnostics` covers the
  same unsupported capture family across imported callback parameters and
  imported enum constructor payloads, imported struct constructor function
  fields in both local storage and direct argument positions, plus imported
  function-typed return producers, including qualified captured-local and
  callable boundary names in diagnostics where no local storage name exists
- mutable captures in direct `ptr` closure calls are rejected with a stable
  diagnostic because that lowering path would otherwise observe mutable locals
  by reference; the supported snapshot route is an explicit function-typed
  `fnptr` binding
- direct `ptr` closure capture diagnostics for unsupported aggregate capture
  types and capturing non-let-bound closure literals name the direct
  ptr-closure capture ABI instead of describing the boundary as an unnamed MVP
- non-capturing symbol-backed function-typed values and captured `fnptr` values
  stored in local struct fields and called directly through
  `value.field(...)`, aliased into function-typed locals, or passed as supported
  callback arguments, including nested local field paths such as
  `box.holder.cb`; nested local field paths may also be reassigned from
  supported named functions. These field values may also be initialized from
  direct closure literals, other immutable symbol-backed struct fields,
  symbol-backed enum payload bindings, or from known function-typed returns with
  stable targets or target-set-backed function-typed parameter-return calls such
  as `Holder(cb: identity(captured))` or
  `Holder(cb: callbacks.identity(captured))`, including multi-target return target sets with
  mutable-global-target classification, returned from
  function-typed return paths, preserved when a known
  struct return carries stable function-field metadata, including after a local
  struct field reassignment before return and through nested struct literal
  initializers such as `Box(holder: makeHolder())`; known struct returns may
  also collect multiple function-field targets across return paths and preserve
  them for subsequent direct field calls or synchronous callback arguments.
  Function-typed fields may be reassigned on mutable local structs from
  supported named functions, closure literals, known function-typed returns, or
  target-set-backed parameter-return calls such as `holder.cb = identity(captured)`,
  including imported forms such as `holder.cb = callbacks.identity(captured)`,
  and nested local field paths such as
  `box.holder.cb = callbacks.identity(captured)`,
  with dynamic dispatch over known target sets, including when the reassigned
  field is later snapshotted into another local struct field, preserved through
  whole-struct local aliases or whole-struct local reassignments such as
  `holder = Holder(cb: callbacks.identity(captured))`, struct-valued field
  reassignments such as `box.holder = Holder(cb: callbacks.identity(captured))`,
  whole nested-struct reassignments such as
  `box = Box(holder: Holder(cb: callbacks.identity(captured)))`,
  snapshotted into a function-typed local alias, or
  passed as a synchronous callback argument; captured `fnptr` struct-field
  values returned directly from function-typed return paths preserve their
  environment slots instead of degrading to a bare function symbol; imported
  functions that return structs carrying captured `fnptr` fields preserve that
  metadata for caller-side direct field calls and imported callee dispatch
- non-capturing symbol-backed function-typed values and captured `fnptr` values
  stored in enum payloads and called directly, aliased into function-typed
  locals, or passed as supported callback arguments after pattern binding from a
  known immutable local enum value; whole-enum local aliases preserve
  function-payload metadata before pattern binding. Payloads may also be
  initialized from direct closure literals, immutable symbol-backed struct
  fields, symbol-backed enum payload bindings, or known function-typed returns
  with stable targets or target-set-backed function-typed parameter-return calls
  such as `MaybeCallback.some(identity(captured))` or
  `MaybeCallback.some(callbacks.identity(captured))`, including multi-target return target sets with
  mutable-global-target classification, preserved
  through known enum returns carrying stable
  function-payload metadata for local bindings and direct `match makeChoice()`
  scrutinees, including multiple known targets collected across enum-producing
  return paths and later passed through synchronous callback arguments, and
  eight-slot captured closure payloads returned through enum-producing functions
  exercise the ten-slot native return path for enum tag plus `fnptr`. Payload
  values may be reassigned from same-module or imported parameter-return calls
  such as `MaybeCallback.some(identity(captured))` and
  `MaybeCallback.some(callbacks.identity(captured))`, preserving captured
  metadata for direct `match` calls and for global-escape diagnostics. The same
  payload metadata is preserved when an enum value is stored behind a mutable
  local struct field and reassigned with imported parameter-return forms such as
  `box.choice = MaybeCallback.some(callbacks.identity(captured))`, then
  snapshotted through `let choice: MaybeCallback = box.choice` before pattern
  binding. Returned structs whose fields contain enum payloads, such as
  `makeBox(f) -> Box(choice: MaybeCallback.some(f))`, preserve the same
  payload metadata after call-site substitution from imported parameter-return
  arguments, including after whole-struct local reassignment such as
  `box = makeBox(callbacks.identity(captured))` and through nested returned
  struct initializers such as `makeOuter(f) -> Outer(box: makeBox(f))`.
  Returned-struct enum-payload target sets also collect multiple known
  return-path targets, for example `makeBox(useSecond)` returning
  `Box(choice: MaybeCallback.some(add2))` or
  `Box(choice: MaybeCallback.some(add1))`, and dispatch the pattern-bound
  payload through direct `match box.choice` calls using the target selected by
  the runtime tag. Payload
  bindings, including captured `fnptr` payload bindings, may be returned from
  function-typed return paths without exposing hidden capture slots as visible
  callback parameters, including through imported enum-producing functions.
  Mutable
  local enum values may be reassigned from supported enum
  constructors carrying direct named functions, direct closure literals, known
  function-typed returns with stable targets including multi-target return
  target sets, or whole-enum aliases before a local `match`; multiple known
  branch targets dispatch through the same stable symbol-address target-set path
  used by callback values, including when the pattern-bound payload is passed to
  a synchronous function-typed callback parameter
- cross-module callback calls through supported function-typed struct fields and
  enum payload bindings, including dependency-cache handling for same-module
  enum constructors
- multi-target callback lowering branches
- selected effect propagation through function-typed values
- stable diagnostics for unsupported assignment sources, unsupported generic
  symbols, throwing callable movement outside the explicitly declared
  `fn(...) -> R throws E` direct-try slice, oversized environments,
  by-reference mutable capture, imported
  mutable function-typed globals that would require cross-module global-data
  ABI, captured function-typed local/struct-field/enum-payload/return-call
  global escape, direct, local-alias/reassignment-routed, return-call-routed,
  or container-routed function-typed parameter global escape, and unsupported
  escape/storage.
  `TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadCallbackThrowsMismatchDiagnostic`
  and
  `TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadCallbackThrowsMismatchDiagnostic`
  pin the generated `.t4i` returned struct-field/enum-payload paths so a
  throwing callable payload cannot be passed to a non-throwing synchronous
  callback parameter without the stable
  `callback function symbol 'local' throws type mismatch` diagnostic.
  `TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureRequiresTryDiagnostic`
  pins the same generated `.t4i` direct returned struct-field path so direct
  dispatch without `try` reports the stable
  `call to throwing function 'holder.cb' requires try` diagnostic.
  `TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureCallbackStub`
  and
  `TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureCallbackThrowsMismatchDiagnostic`
  pin the same direct returned struct-field path when `holder.cb` is passed as
  a synchronous throwing callback argument, including the stable
  `callback function symbol 'holder.cb' throws type mismatch` diagnostic for
  non-throwing callback parameters.
  `TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadRequiresTryDiagnostic`
  and
  `TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadRequiresTryDiagnostic`
  pin the generated `.t4i` returned enum-payload paths so pattern-bound
  throwing payload dispatch without `try` reports the stable
  `call to throwing function 'local' requires try` diagnostic.
  `TestFunctionTypedParameterCannotEscapeIntoGlobalFunctionValue`,
  `TestFunctionTypedParameterLocalAliasCannotEscapeIntoGlobalFunctionValue`,
  `TestFunctionTypedParameterMutableLocalReassignmentCannotEscapeIntoGlobalFunctionValue`,
  `TestFunctionTypedParameterReturnCallCannotEscapeIntoGlobalFunctionValue`,
  `TestImportedFunctionTypedParameterReturnCannotEscapeIntoGlobalFunctionValue`,
  `TestBuildInterfaceOnlyModeFunctionTypedParameterReturnGlobalEscapeDiagnostic`,
  `TestGenerateInterfaceFromSourcePreservesFunctionTypedParameterReturnStub`,
  `TestBuildInterfaceOnlyModeFunctionTypedStructFieldReturnGlobalEscapeDiagnostic`,
  `TestGenerateInterfaceFromSourcePreservesFunctionTypedStructFieldReturnStub`,
  `TestBuildInterfaceOnlyModeFunctionTypedNestedStructFieldReturnGlobalEscapeDiagnostic`,
  `TestGenerateInterfaceFromSourcePreservesFunctionTypedNestedStructFieldReturnStub`,
  `TestBuildInterfaceOnlyModeFunctionTypedStructParameterWholeReturnGlobalEscapeDiagnostic`,
  `TestGenerateInterfaceFromSourcePreservesFunctionTypedStructParameterWholeReturnStub`,
  `TestBuildInterfaceOnlyModeFunctionTypedEnumParameterWholeReturnGlobalEscapeDiagnostic`,
  `TestGenerateInterfaceFromSourcePreservesFunctionTypedEnumParameterWholeReturnStub`,
  `TestFunctionTypedParameterAliasReturnCannotEscapeIntoGlobalFunctionValue`,
  `TestFunctionTypedParameterFieldReturnCannotEscapeIntoGlobalFunctionValue`,
  `TestFunctionTypedParameterStructFieldCannotEscapeIntoGlobalFunctionValue`,
  `TestFunctionTypedParameterEnumPayloadCannotEscapeIntoGlobalFunctionValue`,
  `TestFunctionTypedParameterReturnedStructFieldCannotEscapeIntoGlobalFunctionValue`,
  `TestFunctionTypedParameterReturnedNestedStructFieldCannotEscapeIntoGlobalFunctionValue`,
  `TestFunctionTypedStructParameterWholeReturnCannotEscapeIntoGlobalFunctionValue`,
  `TestImportedFunctionTypedParameterReturnedStructFieldCannotEscapeIntoGlobalFunctionValue`,
  `TestImportedFunctionTypedParameterReturnedNestedStructFieldCannotEscapeIntoGlobalFunctionValue`,
  `TestImportedFunctionTypedStructParameterWholeReturnCannotEscapeIntoGlobalFunctionValue`,
  `TestFunctionTypedEnumParameterWholeReturnCannotEscapeIntoGlobalFunctionValue`,
  `TestImportedFunctionTypedEnumParameterWholeReturnCannotEscapeIntoGlobalFunctionValue`,
  `TestFunctionTypedParameterReturnedEnumPayloadCannotEscapeIntoGlobalFunctionValue`,
  and `TestImportedFunctionTypedParameterReturnedEnumPayloadCannotEscapeIntoGlobalFunctionValue`
  pin the function-typed parameter diagnostics so the rejected heap/global
  escape paths do not fall back to generic symbol-backed or captured-value
  assignment errors
- `TestFullCallableGlobalEscapeRejectsMutableCaptureDiagnostic`,
  `TestFullCallableGlobalEscapeRejectsMutableFunctionTypedAliasDiagnostic`,
  `TestFullCallableGlobalEscapeRejectsMutableStructFieldDiagnostic`,
  `TestFullCallableGlobalEscapeRejectsMutableEnumPayloadDiagnostic`,
  `TestFullCallableGlobalEscapeRejectsMutableReturnedClosureDiagnostic`,
  `TestFullCallableGlobalEscapeRejectsMutableReturnedStructFieldDiagnostic`,
  `TestFullCallableGlobalEscapeRejectsMutableReturnedEnumPayloadDiagnostic`,
  `TestImportedFullCallableGlobalEscapeRejectsMutableReturnedClosureDiagnostic`,
  `TestImportedFullCallableGlobalEscapeRejectsMutableReturnedStructFieldDiagnostic`,
  and
  `TestImportedFullCallableGlobalEscapeRejectsMutableReturnedEnumPayloadDiagnostic`
  pin mutable by-reference capture diagnostics for captured callable values
  that try to escape into global function-typed storage through direct
  closure literals, function-typed aliases, struct fields, enum payloads,
  same-module returned values, and imported returned values
- oversized safe by-value captured callback arguments and captured aliases use
  the fixed 4-slot handle path instead of the eight-slot `fnptr` envelope;
  unsafe mutable/resource captures still report stable classifier diagnostics
- `TestBuildFullCallableLocalNineCaptureSmoke`,
  `TestBuildFullCallableMutableLocalReassignNineCaptureSmoke`,
  `TestBuildFullCallableEscapedNineCaptureReturnSmoke`,
  `TestBuildFullCallableStructFieldNineCaptureSmoke`,
  `TestBuildFullCallableStructFieldReassignNineCaptureSmoke`,
  `TestBuildFullCallableEnumPayloadNineCaptureSmoke`,
  `TestBuildFullCallableEnumPayloadReassignNineCaptureSmoke`,
  `TestBuildFullCallableCallbackArgumentNineCaptureSmoke`, and
  `TestBuildFullCallableEscapedGlobalNineCaptureSmoke` cover the first
  handle-backed oversized environment slice: direct closure literals with nine
  immutable scalar captures may bind directly to a local callable and dispatch,
  reassign a mutable local callable and dispatch, return through a
  function-typed return and bind to a local callable, initialize an immutable
  local struct field and dispatch through that field, reassign a mutable local
  struct field and dispatch through that field, initialize or reassign a local
  enum payload and dispatch through a pattern-bound payload binding, pass
  directly as a synchronous callback argument, or be snapshotted into
  same-module mutable global function-typed storage and dispatched later without
  expanding the bounded nine-slot `fnptr` layout.
  `TestFullCallableStructFieldNineCapturePassesSemanticClassification` and
  `TestFullCallableStructFieldNineCaptureLowersHandleEnvironment` pin the
  struct-field escape metadata and handle-env lowering.
  `TestFullCallableEnumPayloadNineCapturePassesSemanticClassification` and
  `TestFullCallableEnumPayloadNineCaptureLowersHandleEnvironment` pin the
  enum-payload escape metadata and handle-env lowering. The twelve-capture
  alias smokes above prove the handle path is not capped to the initial
  nine-capture smoke slice.

Focused evidence command:

```sh
go test ./compiler/... -run 'Closure|Callable|FunctionType' -count=1
```

## Production Boundary Diagnostics

The selected `v0.4.0` callable scope is current for the safe first-class
callable model: non-capturing symbol-backed values, the bounded `fnptr` fast
path, and the fixed 4-slot handle path for larger immutable by-value captured
environments. That handle path covers local aliases, mutable local storage,
same-module global snapshots, function-typed returns, local and cross-module
returned struct fields and enum payloads, synchronous callback arguments, and
generated `.t4i` metadata.

Remaining rejected forms are production-boundary diagnostics, not missing
release blockers for the promoted safe model. Mutable by-reference captures,
pointer/resource captures, imported mutable global-data escape, unsupported
heap/thread transfer, higher-order generic callable escape, and callable
movement without stable target/capture metadata are rejected before lowering.
Thread-boundary checks currently pin the classifier diagnostics for mutable or
resource captures until the language exposes a source-level
function-value-to-thread transfer surface.

Unsupported function-value escape now reports that the value cannot escape
outside the supported callable ABI and points users toward declared `fn(...)`
parameters, function-typed returns, local storage, struct fields, enum payloads,
or supported same-module global snapshots; captured closure escape as raw
`ptr` points users toward a declared `fn(...)` binding for the by-value snapshot
ABI. Unsupported global escape of captured function values or function-typed
parameters reports the supported direct snapshot/handle requirements. Explicit
type arguments on function-typed struct-field, enum-payload, global, and
callback dispatch report the monomorphic callable dispatch boundary instead of
MVP-era wording. Oversized safe immutable captures are promoted to the handle
path; unsupported capture kinds name the rejected boundary directly.
Generic closure literals with captures now report the production callable
limitation and guide users to a non-generic closure or explicit captured-state
parameters. Generic closure pointer escape and mutable-binding call diagnostics
now name the generic direct-call closure ABI instead of treating the boundary as
an unnamed MVP subset.
Generic and throwing function symbols passed as callback arguments now report
the callback `fnptr` ABI requirements for monomorphic targets and declared
throws-type compatibility. Unsupported callback argument source diagnostics now
list the supported `fnptr` source forms: closure literals, function-typed
locals/globals/struct fields, direct named function/closure symbols, and
function-typed return calls. Imported mutable function-typed globals now report
the cross-module mutable global-data ABI boundary without MVP-era wording.
Function-typed global initializer diagnostics now name the supported `fnptr`
ABI source requirements for same-module symbols, imported public symbols, and
direct symbol or closure-literal initializer shapes. Function-typed local
initializer diagnostics now name the supported symbol-backed, target-set-backed,
direct-symbol, and closure-literal `fnptr` sources instead of MVP-era wording.
Literal or otherwise non-callable function-typed local initializer fallbacks now
reuse that supported `fnptr` source diagnostic.
Function-typed local initializer return-call diagnostics now require the call to
resolve to a function-typed return for the supported `fnptr` ABI.
Function-typed local initializers from non-function globals or unresolved
symbols now reuse the supported `fnptr` source diagnostic instead of older
symbol-kind-specific MVP wording.
Function-typed local initializers from non-function struct-field paths now also
reuse the supported `fnptr` source diagnostic.
Function-typed struct-field and enum-payload initializer source diagnostics now
name the same supported `fnptr` source forms for literal or unsupported
initializer expressions.
Function-typed assignment source diagnostics now also name the supported
`fnptr` source forms for mutable local, struct-field, enum-payload, and
same-module global reassignment fallback paths.
Function-typed assignment return-call diagnostics now require the initializer
call to resolve to a function-typed return for the supported `fnptr` ABI.
Function-typed return source diagnostics now list the supported `fnptr` source
forms instead of falling back to older symbol-backed-only wording.
Generic struct instantiation with function-typed type arguments now reports
that generic structs cannot carry function-typed values under the supported
`fnptr` ABI.
Throwing symbols used to initialize non-throwing function-typed locals now
report the local `fnptr` throws-type compatibility boundary.
Generic function symbols that cannot initialize function-typed locals now
report the local `fnptr` monomorphic-target boundary instead of MVP-era
wording.
Generic function symbols assigned into function-typed mutable locals, fields,
payloads, or globals now report the assignment `fnptr` monomorphic-target
boundary instead of MVP-era wording.
Throwing function symbols assigned into non-throwing function-typed targets now
report the assignment `fnptr` throws-type compatibility boundary instead of
MVP-era wording.
Generic function symbols returned as function-typed values now report the
return `fnptr` monomorphic-target boundary instead of MVP-era wording.
Calls through raw local function values without supported callable metadata now
report the supported `fnptr` call ABI source boundary instead of MVP-era
closure-literal-only wording.
Generic function symbols that cannot initialize function-typed struct fields or
enum payloads now report the respective struct-field or enum-payload `fnptr`
monomorphic-target boundary instead of local-initializer or MVP-era wording.
Lowering-time function-typed parameter calls without a known target set now
report the direct `fnptr` target-set requirement instead of MVP-era callable
wording.
Callback arguments forwarded under strict semantic clauses without a known
target now report the missing stable `fnptr` target-set requirement instead of
MVP-era wording.

## Promotion Boundary

`language.callable-level1` may be `current` only under the `v0.4.0` release
boundary, together with:

- updated feature registry status and `since` metadata
- updated docs that no longer call the promoted slice experimental
- updated validators that expect the promoted slice to be `current`
- release-gate evidence from the same intended release commit
- a clear boundary between promoted Level 1 behavior and still-missing Level 2
  or full first-class callable behavior

The current safe truth is: callable MVP, Callable Level 1, Callable Level 2,
and `language.full-first-class-callables` are current for the v0.4.0 safe
by-value capture model. Mutable by-reference, pointer/resource, and
thread-boundary callable escapes remain explicit diagnostics rather than
silently accepted behavior.
