# Callable Interface Current-State Audit

## reviewed_commit

Reviewed worktree: `/home/tetra/.codex/worktrees/Tetra_Language/safety-semantics-closure-v1`

Reviewed commit: `3d101fbc3e1d8d9a9710c44725372ea086287c9c`

Source plan: `/home/tetra/Downloads/Tetra_Safety_Semantics_Closure_v1_Implementation_Plan.md`

Audit scope was the current implementation only. No production code, tests, workflow files,
generated files, or git history were modified for this audit.

Evidence:

- `git rev-parse HEAD` in the reviewed worktree returned
  `3d101fbc3e1d8d9a9710c44725372ea086287c9c`.
- The target worktree does not contain `graphify-out/`, so this audit used normal
  read-only source inspection.
- Callable ABI constants and metadata live in semantic model structs:
  `LocalInfo`, `FunctionFieldInfo`, `GlobalInfo`, and `FuncSig` carry function value,
  capture, escape-kind, handle, effect, field, and enum-payload metadata
  (`compiler/internal/semantics/model/types.go:47`,
  `compiler/internal/semantics/model/types.go:80`,
  `compiler/internal/semantics/model/types.go:113`,
  `compiler/internal/semantics/model/types.go:140`).

## fnptr_fast_path

The current fast path is a 9-slot function pointer value: one symbol slot plus eight
environment slots. The constants are `FnPtrEnvSlotCount = 8` and
`FnPtrSlotCount = 1 + FnPtrEnvSlotCount` (`compiler/internal/semantics/model/types.go:98`).

The semantic classifier returns `CallableEscapeLocalSnapshot` with `handle=false` when the
capture slot count is at most eight and the boundary is not `thread`
(`compiler/internal/semantics/semantics_memory_resources.go:27`,
`compiler/internal/semantics/semantics_memory_resources.go:44`). Capture slot counts are
computed from each captured type's `SlotCount`
(`compiler/internal/semantics/semantics_memory_resources.go:218`).

`configureClosureCaptures` collects closure captures, appends hidden parameters to the
closure declaration/signature, and increments `sig.ParamSlots` by captured slots
(`compiler/internal/semantics/semantics_memory_resources.go:233`,
`compiler/internal/semantics/semantics_memory_resources.go:300`,
`compiler/internal/semantics/semantics_memory_resources.go:334`). The fast-path supported
capture subset is immutable local `Int`/`Bool`/`String`, simple structs, enums, and
optionals without `ptr`/resource fields
(`compiler/internal/semantics/semantics_memory_resources.go:340`,
`compiler/internal/semantics/semantics_memory_resources.go:364`).

Lowering emits fast-path values as `IRSymAddr` followed by exactly eight env slots,
zero-filling missing env entries (`compiler/internal/lower/lower_callables.go:24`).
Bounded capture extraction returns `nil` when more than eight env slots are needed
(`compiler/internal/lower/lower_callables.go:66`). Local function-typed assignment uses the
fast path when the unbounded env length is not greater than eight
(`compiler/internal/lower/lower_callables.go:97`).

Test coverage includes an eight-capture returned closure passed cross-module to a callback,
expecting exit code 42 (`compiler/tests/callables/captures/function_typed_callable_full_capture_test.go:8`).

## callable_handle_path

The current handle path is a 4-slot value, with `CallableHandleSlotCount = 4`
(`compiler/internal/semantics/model/types.go:98`). When capture slots exceed eight, the
classifier selects `heap` by default, `global` for global boundaries, and `thread` for the
thread boundary, then returns `handle=true` after mutable/resource validation
(`compiler/internal/semantics/semantics_memory_resources.go:48`,
`compiler/internal/semantics/semantics_memory_resources.go:65`,
`compiler/internal/semantics/semantics_memory_resources.go:75`,
`compiler/internal/semantics/semantics_memory_resources.go:83`).

Lowering emits a handle by allocating an unbounded environment block, storing captured
locals into it, then emitting symbol address, env pointer, capture count, and a reserved
zero slot (`compiler/internal/lower/lower_callables.go:36`). Local assignment and callback
argument lowering switch to this handle emission when capture env length is greater than
eight, then zero-fill the surrounding fnptr-width storage when required
(`compiler/internal/lower/lower_callables.go:102`,
`compiler/internal/lower/lower_callables.go:186`).

Callable dispatch uses handle metadata at call sites. Function-typed parameter calls branch
to `lowerCallableHandleLocalCall` when `local.FunctionHandleValue` is true
(`compiler/internal/lower/lower_callables.go:312`). Direct and multi-target dispatch load
hidden captures from handle memory when hidden slots exceed the eight-slot fast-path limit
(`compiler/internal/lower/lower_callables.go:372`,
`compiler/internal/lower/lower_callables.go:408`). Current handle lowering requires a single
stable target (`compiler/internal/lower/lower_callables.go:446`). Stored field/global
function calls also require stable targets and load hidden capture slots from the handle
environment when needed (`compiler/internal/lower/lower_callables.go:522`,
`compiler/internal/lower/lower_callables.go:626`,
`compiler/internal/lower/lower_callables.go:693`).

Test coverage includes nine-capture returned/local/global/callback cases and twelve-capture
alias cases (`compiler/tests/callables/captures/function_typed_callable_full_capture_test.go:100`,
`compiler/tests/callables/captures/function_typed_callable_full_capture_test.go:128`,
`compiler/tests/callables/captures/function_typed_callable_full_capture_test.go:312`,
`compiler/tests/callables/captures/function_typed_callable_full_capture_test.go:346`,
`compiler/tests/callables/captures/function_typed_callable_full_capture_test.go:402`,
`compiler/tests/callables/captures/function_typed_callable_full_capture_test.go:434`,
`compiler/tests/callables/captures/function_typed_callable_full_capture_test.go:473`).

## capture_classifier

The concrete classifier is `classifyCallableEscape`. It combines boundary selection, slot
counting, surface-ephemeral rejection, island-kernel acceptance, mutable-capture rejection,
and pointer/resource-capture rejection
(`compiler/internal/semantics/semantics_memory_resources.go:27`,
`compiler/internal/semantics/semantics_memory_resources.go:36`,
`compiler/internal/semantics/semantics_memory_resources.go:55`,
`compiler/internal/semantics/semantics_memory_resources.go:65`,
`compiler/internal/semantics/semantics_memory_resources.go:75`).

Capture collection is separate from escape classification. `configureClosureCaptures`
collects locals referenced by the closure body, rejects mutable captures unless the caller
allows value captures, rejects surface frame pixel escape, records captures, and mutates
the closure signature with hidden capture params
(`compiler/internal/semantics/semantics_memory_resources.go:245`,
`compiler/internal/semantics/semantics_memory_resources.go:277`,
`compiler/internal/semantics/semantics_memory_resources.go:285`,
`compiler/internal/semantics/semantics_memory_resources.go:300`).

The current implementation does not have a single canonical "classify once per closure
value" site. Production call sites re-run or propagate classification across interface
return metadata, local bindings, assignments, returns, callbacks, struct fields, and enum
payloads: examples include `compiler/internal/semantics/semantics_checker.go:3878`,
`compiler/internal/semantics/semantics_checker.go:8267`,
`compiler/internal/semantics/semantics_checker.go:11279`,
`compiler/internal/semantics/semantics_checker.go:11336`,
`compiler/internal/semantics/semantics_checker.go:15609`,
`compiler/internal/semantics/semantics_checker.go:15768`,
`compiler/internal/semantics/semantics_expressions.go:1627`,
`compiler/internal/semantics/semantics_expressions.go:1878`,
`compiler/internal/semantics/semantics_expressions.go:7644`, and
`compiler/internal/semantics/semantics_expressions.go:9440`.

Negative tests confirm current rejection boundaries for ptr/resource captures and generic
callable aliases (`compiler/tests/callables/unsupported/function_typed_callable_unsupported_diagnostics_test.go:903`,
`compiler/tests/callables/unsupported/function_typed_callable_unsupported_diagnostics_test.go:925`,
`compiler/tests/callables/unsupported/function_typed_callable_unsupported_diagnostics_test.go:943`,
`compiler/tests/callables/unsupported/function_typed_callable_unsupported_diagnostics_test.go:965`,
`compiler/tests/callables/unsupported/function_typed_callable_unsupported_diagnostics_test.go:1038`).

## escape_boundaries

The declared callable escape boundaries are local, return, global, struct-field,
enum-payload, callback, and thread
(`compiler/internal/semantics/semantics_memory_resources.go:15`).

Local binding paths configure captures and classify oversized local function-typed values
under `callableBoundaryLocal`
(`compiler/internal/semantics/semantics_checker.go:11262`,
`compiler/internal/semantics/semantics_checker.go:11279`,
`compiler/internal/semantics/semantics_checker.go:11336`). Assignment metadata takes an
explicit boundary and reuses stored escape/handle metadata from locals, fields, and
function-typed return calls (`compiler/internal/semantics/semantics_checker.go:8248`,
`compiler/internal/semantics/semantics_checker.go:8267`,
`compiler/internal/semantics/semantics_checker.go:8270`,
`compiler/internal/semantics/semantics_checker.go:8273`,
`compiler/internal/semantics/semantics_checker.go:8281`).

Return paths classify oversized closure returns and captured local returns under
`callableBoundaryReturn` (`compiler/internal/semantics/semantics_checker.go:15596`,
`compiler/internal/semantics/semantics_checker.go:15609`,
`compiler/internal/semantics/semantics_checker.go:15759`,
`compiler/internal/semantics/semantics_checker.go:15768`). Callback argument paths classify
oversized closure literals and local callable aliases under `callableBoundaryCallback`
(`compiler/internal/semantics/semantics_expressions.go:1605`,
`compiler/internal/semantics/semantics_expressions.go:1627`,
`compiler/internal/semantics/semantics_expressions.go:1844`,
`compiler/internal/semantics/semantics_expressions.go:1878`).

Struct-field and enum-payload initializer paths classify oversized captures under their
own boundaries and store the resulting escape kind/handle flag into `FunctionFieldInfo`
(`compiler/internal/semantics/semantics_expressions.go:7622`,
`compiler/internal/semantics/semantics_expressions.go:7644`,
`compiler/internal/semantics/semantics_expressions.go:7676`,
`compiler/internal/semantics/semantics_expressions.go:9418`,
`compiler/internal/semantics/semantics_expressions.go:9440`,
`compiler/internal/semantics/semantics_expressions.go:9466`).

Thread exists as an enum/classifier boundary (`compiler/internal/semantics/model/types.go:104`,
`compiler/internal/semantics/semantics_memory_resources.go:24`,
`compiler/internal/semantics/semantics_memory_resources.go:52`). In the reviewed production
paths, task spawning takes a string literal worker name and validates a named function's
shape, synchrony, throwing behavior, mutable-global use, effects, and sendability
(`compiler/internal/semantics/semantics_expressions.go:2675`,
`compiler/internal/semantics/semantics_expressions.go:2745`,
`compiler/internal/semantics/semantics_expressions.go:2771`,
`compiler/internal/semantics/semantics_expressions.go:2791`,
`compiler/internal/semantics/semantics_expressions.go:2808`,
`compiler/internal/semantics/semantics_expressions.go:2832`,
`compiler/internal/semantics/semantics_expressions.go:2849`,
`compiler/internal/semantics/semantics_expressions.go:4682`). No reviewed production call
site transfers a function-typed callable value itself across the thread boundary.

## cross_module_metadata

Cross-module callable metadata is currently represented in semantic structs and reconstructed
from `.t4i` interface stubs, not from an explicit serialized callable contract block.

`FuncSig` stores returned function metadata including returned function params, captures,
mutable-global touch flag, escape kind, handle flag, function fields, and enum payload
functions (`compiler/internal/semantics/model/types.go:140`). Locals and fields store
parallel metadata for function values and function-typed storage
(`compiler/internal/semantics/model/types.go:47`,
`compiler/internal/semantics/model/types.go:80`).

During checking, interface modules are detected in the fixed-point analysis loop and
`applyInterfaceFunctionReturnMetadata` is applied to functions from those modules
(`compiler/internal/semantics/semantics_checker.go:1218`,
`compiler/internal/semantics/semantics_checker.go:1233`). For returned closures in
interface modules, that function rebuilds stub locals, configures closure captures,
classifies oversized captures, and updates `ReturnFunctionSymbol`,
`ReturnFunctionCaptures`, `ReturnFunctionEscapeKind`, `ReturnFunctionHandleValue`, and
`ReturnSlots` (`compiler/internal/semantics/semantics_checker.go:3836`,
`compiler/internal/semantics/semantics_checker.go:3848`,
`compiler/internal/semantics/semantics_checker.go:3871`,
`compiler/internal/semantics/semantics_checker.go:3878`,
`compiler/internal/semantics/semantics_checker.go:3887`,
`compiler/internal/semantics/semantics_checker.go:3895`).

Lowering helper code resolves callable targets from assigned expressions, including function
fields, imported functions, call-return signatures, locals, and globals
(`compiler/internal/lower/callables/targets.go:11`). Enum payload callable targets are
resolved from local payload metadata, field payloads, function return metadata, or constructor
arguments (`compiler/internal/lower/callables/targets.go:75`). Tests assert target
qualification, import/field target resolution, and enum payload metadata cloning
(`compiler/internal/lower/callables/targets_test.go:11`,
`compiler/internal/lower/callables/targets_test.go:21`,
`compiler/internal/lower/callables/targets_test.go:55`).

Interface tests confirm current cross-module preservation for a returned nine-capture
handle: the generated `.t4i` is parsed as an interface module, the checked signature keeps
nine captures, `heap`, handle `true`, and four return slots, and the app local receives the
same handle metadata (`compiler/tests/semantics/semantics_types_protocols_test.go:3888`,
`compiler/tests/semantics/semantics_types_protocols_test.go:3922`,
`compiler/tests/semantics/semantics_types_protocols_test.go:3934`,
`compiler/tests/semantics/semantics_types_protocols_test.go:3946`,
`compiler/tests/semantics/semantics_types_protocols_test.go:3958`).

## synthetic_t4i_body_dependencies

The current interface generator is source-AST based. `GenerateInterfaceFromSource` parses
the original source file and writes a public-surface `.t4i` body, then prefixes it with a
hash header; it does not consume a checked semantic program or encode a callable contract
block (`compiler/compiler_facade.go:5797`, `compiler/compiler_facade.go:5805`,
`compiler/compiler_facade.go:5870`, `compiler/compiler_facade.go:5880`).

The only implementation currently under `compiler/internal/t4iface` is hash handling:
hash prefix generation, hash splitting, and hash validation
(`compiler/internal/t4iface/hash.go:11`,
`compiler/internal/t4iface/hash.go:18`,
`compiler/internal/t4iface/hash.go:30`,
`compiler/internal/t4iface/hash.go:51`). The module loader validates the `.t4i` hash, parses
the file, and stores `InterfaceHash`; it does not decode callable contract metadata
(`compiler/internal/module/loader.go:197`,
`compiler/internal/module/loader.go:203`,
`compiler/internal/module/loader.go:210`,
`compiler/internal/module/loader.go:215`).

Function interface bodies are synthetic summaries. `interfaceFunctionBody` chooses between
match-return stubs, returned-closure capture stubs, throw stubs, borrowed-return comments,
or a synthetic return expression (`compiler/compiler_facade.go:6978`). Borrowed-return
metadata is represented as a comment line in the generated body
(`compiler/compiler_facade.go:6994`). Returned closure captures are reconstructed as `let`
or `var` stubs plus a synthetic captured closure literal
(`compiler/compiler_facade.go:7175`,
`compiler/compiler_facade.go:7210`,
`compiler/compiler_facade.go:7225`). Function-typed return paths are recovered by scanning
body statements, aliases, callback argument names, match cases, and payload bindings
(`compiler/compiler_facade.go:7732`,
`compiler/compiler_facade.go:7790`,
`compiler/compiler_facade.go:7918`).

On import, semantic checking then re-analyzes these synthetic bodies to derive region,
resource, and callable return metadata
(`compiler/internal/semantics/semantics_checker.go:3657`,
`compiler/internal/semantics/semantics_checker.go:3796`,
`compiler/internal/semantics/semantics_checker.go:3836`,
`compiler/internal/semantics/semantics_checker.go:4000`,
`compiler/internal/semantics/semantics_checker.go:4074`,
`compiler/internal/semantics/semantics_checker.go:4185`).

## async_suspension_dependencies

Async suspension tracking currently lives in region state. `regionState` records
`awaitInvalidatedBorrow`, async flags, and the active await call
(`compiler/internal/semantics/semantics_memory_resources.go:1240`,
`compiler/internal/semantics/semantics_memory_resources.go:1251`,
`compiler/internal/semantics/semantics_memory_resources.go:1278`). Flow snapshots preserve
and merge the await-invalidated borrow map
(`compiler/internal/semantics/semantics_checker.go:13876`,
`compiler/internal/semantics/semantics_checker.go:13905`,
`compiler/internal/semantics/semantics_checker.go:13935`,
`compiler/internal/semantics/semantics_checker.go:4988`).

Both `try await` and plain `await` check an async call and, when successful, call
`invalidateBorrowedRegionsAfterAwait`
(`compiler/internal/semantics/semantics_expressions.go:511`,
`compiler/internal/semantics/semantics_expressions.go:535`,
`compiler/internal/semantics/semantics_expressions.go:555`,
`compiler/internal/semantics/semantics_expressions.go:579`,
`compiler/internal/semantics/semantics_expressions.go:586`,
`compiler/internal/semantics/semantics_expressions.go:603`). Invalidation walks region vars,
marks regions owned by borrowed params, and later rejects use of those borrowed views after
the suspension (`compiler/internal/semantics/semantics_memory_resources.go:2186`,
`compiler/internal/semantics/semantics_memory_resources.go:2193`,
`compiler/internal/semantics/semantics_memory_resources.go:2205`,
`compiler/internal/semantics/semantics_memory_resources.go:2216`).

Tests cover borrowed view use after `await` and `try await`
(`compiler/tests/semantics/semantics_async_ownership_test.go:958`,
`compiler/tests/semantics/semantics_async_ownership_test.go:973`). They also cover named
task spawn shape/runtime/synchrony restrictions, not callable-value transfer
(`compiler/tests/semantics/semantics_async_ownership_test.go:369`,
`compiler/tests/semantics/semantics_async_ownership_test.go:409`,
`compiler/tests/semantics/semantics_async_ownership_test.go:430`,
`compiler/tests/semantics/semantics_async_ownership_test.go:455`).

Within reviewed paths, no await handler consults callable capture metadata such as
`FunctionCaptures`, `FunctionEscapeCaptures`, `FunctionHandleValue`, or a callable capture
contract. Current async suspension safety is therefore region-borrow based, not
callable-contract based.

## confirmed_gaps

1. No explicit `.t4i` callable contract schema/API is present in reviewed paths. The t4i
   package contains only hash helpers (`compiler/internal/t4iface/hash.go:11`,
   `compiler/internal/t4iface/hash.go:51`), the loader only validates the hash and parses
   the file (`compiler/internal/module/loader.go:197`), and interface generation writes a
   source-derived stub body (`compiler/compiler_facade.go:5805`). A repository search for
   names such as `CallableTypeContract`, `CallableValueContract`, `FunctionContract`, and
   `function-contract` found no implementation in reviewed compiler paths.

2. Imported interface callable metadata is still dependent on synthetic `.t4i` bodies and
   semantic re-analysis. Returned closures, borrowed returns, function-typed return paths,
   match/payload paths, and region/resource metadata are reconstructed from generated body
   stubs and comments (`compiler/compiler_facade.go:6978`,
   `compiler/compiler_facade.go:6994`,
   `compiler/compiler_facade.go:7175`,
   `compiler/compiler_facade.go:7732`,
   `compiler/internal/semantics/semantics_checker.go:3657`,
   `compiler/internal/semantics/semantics_checker.go:3836`,
   `compiler/internal/semantics/semantics_checker.go:4000`).

3. Capture classification is not a single canonical decision made once per closure value.
   The same classifier is called from multiple production contexts, including interface
   return metadata, local binding, assignment propagation, return handling, callbacks,
   struct fields, and enum payloads (`compiler/internal/semantics/semantics_checker.go:3878`,
   `compiler/internal/semantics/semantics_checker.go:8267`,
   `compiler/internal/semantics/semantics_checker.go:11279`,
   `compiler/internal/semantics/semantics_checker.go:15609`,
   `compiler/internal/semantics/semantics_expressions.go:1627`,
   `compiler/internal/semantics/semantics_expressions.go:7644`,
   `compiler/internal/semantics/semantics_expressions.go:9440`).

4. Current semantic structs carry capture slices, mutable-global flags, escape kind, and
   handle flags, but no explicit callable contract digest field in the reviewed definitions
   (`compiler/internal/semantics/model/types.go:47`,
   `compiler/internal/semantics/model/types.go:80`,
   `compiler/internal/semantics/model/types.go:140`). The digest hits found in compiler
   internals are unrelated memory-plan/lowering/profile digests, not callable contract
   digests.

5. Thread boundary support is declared in the classifier model but is not wired to a
   production callable-value transfer path in reviewed code. Task spawn paths validate named
   string-literal worker functions and do not accept or transfer a function-typed callable
   value (`compiler/internal/semantics/semantics_memory_resources.go:24`,
   `compiler/internal/semantics/semantics_memory_resources.go:52`,
   `compiler/internal/semantics/semantics_expressions.go:2745`,
   `compiler/internal/semantics/semantics_expressions.go:4689`).

6. Async suspension safety does not currently inspect callable capture contracts at `await`.
   The implemented mechanism invalidates borrowed regions after suspension
   (`compiler/internal/semantics/semantics_expressions.go:555`,
   `compiler/internal/semantics/semantics_expressions.go:603`,
   `compiler/internal/semantics/semantics_memory_resources.go:2186`), and tests assert
   borrowed-view rejection after await (`compiler/tests/semantics/semantics_async_ownership_test.go:958`).

## forbidden_feature_expansions

- Do not expand or reinterpret the current callable ABI as supporting more than the declared
  fast-path and handle widths. The declared constants are eight env slots, nine fnptr slots,
  and four handle slots (`compiler/internal/semantics/model/types.go:98`), and lowering
  emits exactly those forms (`compiler/internal/lower/lower_callables.go:24`,
  `compiler/internal/lower/lower_callables.go:36`).

- Do not treat mutable, pointer, resource, surface, or unsupported aggregate captures as
  accepted callable captures. Current capture configuration and classification reject these
  categories (`compiler/internal/semantics/semantics_memory_resources.go:277`,
  `compiler/internal/semantics/semantics_memory_resources.go:285`,
  `compiler/internal/semantics/semantics_memory_resources.go:65`,
  `compiler/internal/semantics/semantics_memory_resources.go:75`), with negative tests for
  ptr-field captures in callback/local/field/payload/reassignment paths
  (`compiler/tests/callables/unsupported/function_typed_callable_unsupported_diagnostics_test.go:903`,
  `compiler/tests/callables/unsupported/function_typed_callable_unsupported_diagnostics_test.go:925`,
  `compiler/tests/callables/unsupported/function_typed_callable_unsupported_diagnostics_test.go:943`,
  `compiler/tests/callables/unsupported/function_typed_callable_unsupported_diagnostics_test.go:965`,
  `compiler/tests/callables/unsupported/function_typed_callable_unsupported_diagnostics_test.go:988`).

- Do not claim dynamic dispatch or arbitrary multi-target callable handles. Function-typed
  parameter lowering requires a known target set (`compiler/internal/lower/lower_callables.go:256`),
  and handle lowering currently requires a single stable target
  (`compiler/internal/lower/lower_callables.go:446`). Stored function calls also reject
  missing stable targets (`compiler/internal/lower/lower_callables.go:527`).

- Do not claim runtime generic callable polymorphism. Generic callable aliases are rejected
  for function-typed assignment, and return/callback handling rejects generic function values
  (`compiler/tests/callables/unsupported/function_typed_callable_unsupported_diagnostics_test.go:1038`,
  `compiler/internal/semantics/semantics_checker.go:15681`,
  `compiler/internal/semantics/semantics_checker.go:15744`,
  `compiler/internal/semantics/semantics_expressions.go:1852`).

- Do not claim function-typed callable values can cross task/thread boundaries. Current task
  spawn code accepts named worker functions via string literals and validates named function
  properties, while callable `thread` escape is only a declared classifier boundary in
  reviewed production paths (`compiler/internal/semantics/semantics_expressions.go:2745`,
  `compiler/internal/semantics/semantics_expressions.go:2771`,
  `compiler/internal/semantics/semantics_expressions.go:4689`,
  `compiler/internal/semantics/semantics_memory_resources.go:24`).

- Do not treat missing `.t4i` callable metadata as safe or canonical. The loader currently
  validates only the hash and parses the stub body (`compiler/internal/module/loader.go:203`),
  and current callable metadata comes from generated bodies plus semantic re-analysis rather
  than an explicit metadata contract (`compiler/compiler_facade.go:6978`,
  `compiler/internal/semantics/semantics_checker.go:3836`).
