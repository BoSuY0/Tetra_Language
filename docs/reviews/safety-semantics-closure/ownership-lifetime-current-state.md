# SSC-T01 SUBAGENT-A current-state audit: ownership/lifetime flow

## reviewed_commit

- Reviewed commit: `3d101fbc3e1d8d9a9710c44725372ea086287c9c`.
- Target worktree: `/home/tetra/.codex/worktrees/Tetra_Language/safety-semantics-closure-v1`.
- Reviewed source/test surface requested for this packet:
  `compiler/internal/semantics/semantics_checker.go`,
  `compiler/internal/semantics/semantics_memory_resources.go`,
  `compiler/internal/semantics/flow/**`,
  `compiler/internal/semantics/resources/**`,
  `compiler/internal/semantics/regions/**`,
  `compiler/tests/ownership/**`,
  `compiler/tests/runtime/resource_finalization/**`, and
  `compiler/tests/semantics/semantics_async_ownership_test.go`.
- Graphify artifacts were not present in this worktree at `graphify-out/GRAPH_REPORT.md` or
  `graphify-out/wiki/index.md` during the audit. No graph rebuild was run because this packet is
  read-only except for this deliverable.

## existing_state_maps

- `scopeInfo` already carries per-scope ownership/region/resource state maps:
  `regionVars`, `resourceVars`, `consumedVars`, `maybeConsumedVars`, and
  `borrowedPtrAliases` (`compiler/internal/semantics/semantics_memory_resources.go:1176`).
  `newScopeInfo` initializes those maps for each scope (`compiler/internal/semantics/semantics_memory_resources.go:1193`).
- `regionState` is the main mutable lifetime/resource state. It includes region maps
  (`regionVars`, `exprRegionTrees`, `paramRegionIndex`, `borrowedParamRegion`), resource maps
  (`resourceParamIndex`, `resourceParamPath`, `resourceVars`, `consumedResources`,
  `unknownResources`, `finalizedResources`), ownership maps (`consumedVars`,
  `maybeConsumedVars`, `ownershipAliases`, `borrowedPtrAliases`, `ownedRegionSliceOwners`),
  async invalidation state (`awaitInvalidatedBorrow`), defer-capture frames, loop-flow state,
  and return/throw summary state (`compiler/internal/semantics/semantics_memory_resources.go:1231`).
  `newRegionState` initializes these maps and stacks (`compiler/internal/semantics/semantics_memory_resources.go:1324`).
- Control-flow snapshots preserve reachability plus ownership, alias, await invalidation, resource,
  and finalization maps (`compiler/internal/semantics/semantics_checker.go:13875`).
  `snapshotFlow` clones these maps (`compiler/internal/semantics/semantics_checker.go:13901`), and
  `restoreFlow` restores them (`compiler/internal/semantics/semantics_checker.go:13917`).
- Function-level summaries are stored back into signatures after body analysis:
  return region summaries are persisted at `compiler/internal/semantics/semantics_checker.go:1430`,
  return resource summaries at `compiler/internal/semantics/semantics_checker.go:1465`, and
  throw resource summaries at `compiler/internal/semantics/semantics_checker.go:1495`.

## existing_path_encodings

- Resource paths are plain strings. `PathForExpr` derives identifier/field paths
  (`compiler/internal/semantics/resources/paths.go:10`), `FieldPath` appends `.field`
  (`compiler/internal/semantics/resources/paths.go:22`), `EnumPayloadPath` emits
  `$caseN.payloadM` (`compiler/internal/semantics/resources/paths.go:29`), and `JoinPath`
  concatenates non-empty path parts with `.` (`compiler/internal/semantics/resources/paths.go:33`).
- The resource path helper still has prefix/string-tail behavior via `LeafTail`, including
  `strings.HasPrefix` (`compiler/internal/semantics/resources/paths.go:43`).
- Region/resource tree collection uses the same string wire shape: array/optional payloads use
  `$elem`, structs use field names, and enum payloads use `$caseN.payloadM`
  (`compiler/internal/semantics/semantics_memory_resources.go:2458`,
  `compiler/internal/semantics/semantics_memory_resources.go:2610`,
  `compiler/internal/semantics/semantics_checker.go:14878`).
- The checker wraps the helper API as `resourcePathForExpr`, `resourceFieldPath`,
  `resourceEnumPayloadPath`, and `joinResourcePath`
  (`compiler/internal/semantics/semantics_checker.go:15013`).
- Ownership-access paths are also string paths in expression checking, including field/index
  extension and parent/prefix helpers (`compiler/internal/semantics/semantics_expressions.go:5074`,
  `compiler/internal/semantics/semantics_expressions.go:5149`).

## existing_join_functions

- Region variable joins are centralized in `regions.MergeVars`: matching regions are preserved,
  missing/differing regions become `Unknown` (`compiler/internal/semantics/regions/tree.go:25`).
  Region tree joins/common-region helpers are in `Join`, `CommonFromTree`, and
  `ConstructorFromTree` (`compiler/internal/semantics/regions/tree.go:54`).
- `semantics_memory_resources.go` delegates region merge/join helpers to the `regions` package
  (`compiler/internal/semantics/semantics_memory_resources.go:2434`).
- Flow summary clone/equality helpers exist for return region/resource summaries
  (`compiler/internal/semantics/flow/return_summaries.go:5`,
  `compiler/internal/semantics/flow/return_summaries.go:29`).
- Branch/control joins live in the checker. `mergeFlowWithLabels` merges reachability,
  consumed/maybe-consumed maps, aliases, await invalidation, resources, and finalization
  (`compiler/internal/semantics/semantics_checker.go:13935`). `mergeControlFlowWithLabels`
  handles one-sided reachability, then merges regions, flow, and taint state
  (`compiler/internal/semantics/semantics_checker.go:14057`).
- The current implementation has local merge helpers in `semantics_checker.go`; it does not expose
  a canonical `FlowState` object in `compiler/internal/semantics/flow` beyond the summary helper
  file present in that package.

## loop_behavior

- Loop frames collect `break` and `continue` snapshots, keyed by labels, using
  `pushLoopFlowFrame`, `recordLoopFlowExit`, `popLoopFlowFrame`, and `mergeLoopFlowExits`
  (`compiler/internal/semantics/semantics_checker.go:13985`).
- `break` and `continue` statements record current flow, check pending defer captures, and make the
  current path unreachable (`compiler/internal/semantics/semantics_checker.go:16346`).
- `while` analysis snapshots the pre-loop flow, analyzes the body once, merges reachable body flow
  with `continue` exits, then merges that with `break` exits and the zero-iteration pre-loop path
  (`compiler/internal/semantics/semantics_checker.go:17770`). `for` uses the same single-pass
  pattern (`compiler/internal/semantics/semantics_checker.go:17881`).
- Tests confirm loop joins can create maybe-consumed state:
  `TestConditionalMoveMarksMaybeConsumed` covers maybe-consumed after control flow
  (`compiler/tests/ownership/ownership_escape_assignment_test.go:56`),
  resource finalization tests cover maybe-freed/closed state after loops
  (`compiler/tests/runtime/resource_finalization/resource_finalization_island_core_test.go:128`,
  `compiler/tests/runtime/resource_finalization/resource_finalization_task_group_test.go:366`),
  and loop break/continue label handling has coverage
  (`compiler/tests/ownership/ownership_scalar_escape_test.go:824`).
- No fixed-point/digest loop iteration is visible in the reviewed loop implementation; the loop body
  is analyzed once and joined with collected exits.

## partial_move_behavior

- Direct consumption records either a resource path/resource id or an ownership path
  (`compiler/internal/semantics/semantics_memory_resources.go:1373`).
  `consumedPath` checks exact paths, ancestors, ownership aliases, and resource aliases
  (`compiler/internal/semantics/semantics_memory_resources.go:1756`).
- Whole-value availability checks reject use of an aggregate after a consumed descendant path,
  including consumed resource descendants (`compiler/internal/semantics/semantics_memory_resources.go:1652`).
  Synthetic path segments such as `$case0.payload0` are preserved for diagnostics where applicable
  (`compiler/internal/semantics/semantics_memory_resources.go:1781`).
- Expression availability checks inspect aggregate constructor arguments and all resource leaves for
  ownership/resource-containing values (`compiler/internal/semantics/semantics_expressions.go:5222`).
- Tests cover struct field partial moves and sibling access:
  sibling field still usable (`compiler/tests/ownership/ownership_escape_assignment_test.go:160`),
  consumed field reuse rejected (`compiler/tests/ownership/ownership_escape_assignment_test.go:176`),
  whole struct use after partial consume rejected (`compiler/tests/ownership/ownership_escape_assignment_test.go:192`),
  and cross-module struct field cases (`compiler/tests/ownership/ownership_escape_assignment_test.go:372`).
- Tests cover enum/optional path encodings:
  enum payload sibling use is accepted (`compiler/tests/ownership/ownership_escape_assignment_test.go:516`),
  whole enum use after payload consume is rejected with `$case0.payload0`
  (`compiler/tests/ownership/ownership_escape_assignment_test.go:563`), and optional payload reuse is
  rejected with `$elem` (`compiler/tests/ownership/ownership_optional_payload_escape_test.go:728`).

## reinitialization_behavior

- Reinitialization clears consumed/maybe-consumed state for a path subtree via `clearConsumedTree`,
  including ownership aliases and resource aliases under that path
  (`compiler/internal/semantics/semantics_memory_resources.go:1397`).
- Assignment first checks that a target path is assignable; a child path cannot be assigned if an
  ancestor has already been consumed (`compiler/internal/semantics/semantics_memory_resources.go:1432`,
  `compiler/internal/semantics/semantics_checker.go:17007`).
- Assignment rebinds region/resource state for the target and then clears consumed state for the
  assigned ownership path (`compiler/internal/semantics/semantics_checker.go:17342`,
  `compiler/internal/semantics/semantics_checker.go:17398`).
  `let` initialization clears prior consumed state for the local name
  (`compiler/internal/semantics/semantics_checker.go:16621`).
- Tests confirm consumed field reassignment is accepted
  (`compiler/tests/ownership/ownership_escape_assignment_test.go:267`), whole struct reassignment
  after partial consume is accepted (`compiler/tests/ownership/ownership_escape_assignment_test.go:284`),
  whole enum reassignment after payload consume is accepted
  (`compiler/tests/ownership/ownership_escape_assignment_test.go:918`), and assigning a field after
  whole-root consume is rejected (`compiler/tests/ownership/ownership_escape_assignment_test.go:301`).
- Resource finalization tests also cover assignment-based reopening of a task group after close
  (`compiler/tests/runtime/resource_finalization/resource_finalization_task_group_test.go:351`).

## return_region_summary_behavior

- `recordReturnRegionSummary` records path-to-region provenance for returns and rejects unknown,
  scoped, or non-parameter region escapes. It also rejects a path that merges different parameter
  regions (`compiler/internal/semantics/semantics_memory_resources.go:2893`).
- After body analysis, the checker stores a cloned `ReturnRegionSummary` on the function signature
  and also derives `ReturnRegionParam` when every returned path maps to the same parameter
  (`compiler/internal/semantics/semantics_checker.go:1430`).
- Interface/callable metadata paths can synthesize return region summaries from an expression
  (`compiler/internal/semantics/semantics_checker.go:3987`) and apply interface metadata back into
  function signatures (`compiler/internal/semantics/semantics_checker.go:3796`).
- Return-region summaries are cloned/compared through helpers in `semantics/flow`
  (`compiler/internal/semantics/flow/return_summaries.go:5`).
- Tests cover borrowed return rejection/acceptance for slices, pointers, aliases, and cross-module
  aggregate cases (`compiler/tests/ownership/ownership_test.go:140`,
  `compiler/tests/ownership/ownership_cross_module_aggregate_test.go:855`,
  `compiler/tests/ownership/ownership_cross_module_aggregate_test.go:1170`), plus async borrowed
  return cases (`compiler/tests/semantics/semantics_async_ownership_test.go:171`).

## return_resource_summary_behavior

- Resource parameter provenance is tracked through `resourceParamIndex` and `resourceParamPath`
  (`compiler/internal/semantics/semantics_memory_resources.go:1262`), initialized for resource
  parameters during function analysis (`compiler/internal/semantics/semantics_memory_resources.go:2698`).
- `returnResourceSummaryForExpr` maps returned resource leaves to one or more
  `ResourceProvenance{ParamIndex, ParamPath}` entries
  (`compiler/internal/semantics/semantics_checker.go:15042`). `recordReturnResourceSummary` records
  this map, rejects mixed provenance for the same leaf, and sets root `ReturnResourceParam` /
  `ReturnResourcePath` when the root has a single provenance
  (`compiler/internal/semantics/semantics_memory_resources.go:2955`).
- The checker stores the summary on the function signature after body analysis
  (`compiler/internal/semantics/semantics_checker.go:1465`) and fails closed for uninferred
  resource returns that have neither a root provenance nor a summary
  (`compiler/internal/semantics/semantics_checker.go:1595`).
- Call-result binding applies a callee `ReturnResourceSummary` back to returned aggregates
  (`compiler/internal/semantics/semantics_checker.go:14474`), and resource-source lookup handles
  root and subpath return provenance (`compiler/internal/semantics/semantics_checker.go:13744`).
- Tests cover task-group/resource wrappers across modules
  (`compiler/tests/runtime/resource_finalization/resource_finalization_actor_consume_test.go:9`),
  interprocedural struct resource leaves
  (`compiler/tests/ownership/ownership_scalar_escape_test.go:741`), interprocedural double-free
  through returned resource fields
  (`compiler/tests/runtime/resource_finalization/resource_finalization_island_core_test.go:355`),
  and ambiguous/uninferred resource returns
  (`compiler/tests/runtime/resource_finalization/resource_finalization_island_core_test.go:487`,
  `compiler/tests/runtime/resource_finalization/resource_finalization_island_core_test.go:550`).

## throw_resource_summary_behavior

- Throw summaries are held in `regionState.throwResourceSummary`
  (`compiler/internal/semantics/semantics_memory_resources.go:1288`) and recorded with
  `recordThrowResourceSummary` (`compiler/internal/semantics/semantics_memory_resources.go:3002`).
- A `throw` statement records a resource summary when the thrown type contains a resource handle and
  the expression's resource provenance is known; the throw path then becomes unreachable
  (`compiler/internal/semantics/semantics_checker.go:16427`,
  `compiler/internal/semantics/semantics_checker.go:16497`).
- Try-call propagation maps a callee `ThrowResourceSummary` through the actual arguments
  (`compiler/internal/semantics/semantics_checker.go:15167`), and catch binding can attach thrown
  resource leaves to the caught error value (`compiler/internal/semantics/semantics_checker.go:15108`).
- After body analysis, the checker clones the state summary into `sig.ThrowResourceSummary`
  (`compiler/internal/semantics/semantics_checker.go:1495`). Interface metadata update paths also
  write explicit throw summaries and try-rethrow summaries
  (`compiler/internal/semantics/semantics_checker.go:3670`,
  `compiler/internal/semantics/semantics_checker.go:3713`).
- Reviewed tests cover borrow escape/use-after-consume behavior around `throw`
  (`compiler/tests/ownership/ownership_scalar_escape_test.go:290`,
  `compiler/tests/ownership/ownership_scalar_escape_test.go:320`), but no reviewed test path showed a
  resource-bearing typed error validating `ThrowResourceSummary` provenance end to end.

## await_invalidation_behavior

- Await invalidation state is stored in `awaitInvalidatedBorrow`
  (`compiler/internal/semantics/semantics_memory_resources.go:1268`) and included in flow snapshots
  and joins (`compiler/internal/semantics/semantics_checker.go:13882`,
  `compiler/internal/semantics/semantics_checker.go:13966`).
- `invalidateBorrowedRegionsAfterAwait` marks all locals whose region is derived from a borrowed
  parameter after an await position (`compiler/internal/semantics/semantics_memory_resources.go:2186`).
  `checkBorrowedRegionAfterAwait` emits the lifetime diagnostic when a marked value is used
  (`compiler/internal/semantics/semantics_memory_resources.go:2205`), and `checkRegionUsable`
  invokes that check (`compiler/internal/semantics/semantics_memory_resources.go:2594`).
- Supporting call sites in expression analysis invoke invalidation after successful `try await` and
  `await` expressions (`compiler/internal/semantics/semantics_expressions.go:511`,
  `compiler/internal/semantics/semantics_expressions.go:561`).
- Tests confirm a borrowed view is usable before await
  (`compiler/tests/semantics/semantics_async_ownership_test.go:942`) and rejected after both `await`
  and `try await` (`compiler/tests/semantics/semantics_async_ownership_test.go:958`,
  `compiler/tests/semantics/semantics_async_ownership_test.go:973`). Cross-module async borrowed
  return cases are also covered (`compiler/tests/semantics/semantics_async_ownership_test.go:1801`).

## defer_capture_behavior

- Defer capture state is stored as a stack of capture frames on `regionState`
  (`compiler/internal/semantics/semantics_memory_resources.go:1284`). The state has helpers to push,
  pop, register, and check pending captures (`compiler/internal/semantics/semantics_memory_resources.go:2260`).
- `deferredCaptureConsumedAt` checks captured locals against consumed ownership/resource state,
  ownership aliases, and resource aliases (`compiler/internal/semantics/semantics_memory_resources.go:2312`).
- `checkStmts` pushes/pops a defer capture frame for statement lists
  (`compiler/internal/semantics/semantics_checker.go:16298`), checks pending defer captures before
  `break`/`continue` flow exits (`compiler/internal/semantics/semantics_checker.go:16346`), and checks
  pending defer captures after each reachable statement (`compiler/internal/semantics/semantics_checker.go:18307`).
- `DeferStmt` validates cleanup-body control flow, analyzes the cleanup body, and registers captures
  (`compiler/internal/semantics/semantics_checker.go:16545`). Defer body control rejects
  `return`, `throw`, nested `defer`, and invalid `break`/`continue` in cleanup bodies
  (`compiler/internal/semantics/semantics_checker.go:4444`).
- Reviewed test paths did not contain `defer` coverage. The save/restore block for defer-body
  analysis restores region, consumed, resource, and finalized maps
  (`compiler/internal/semantics/semantics_checker.go:4519`), but the visible restored fields do not
  include `returnResourceParam`, `returnResourcePath`, `returnResourceSummary`, `returnResourceSet`,
  `returnResourceUnknown`, or `throwResourceSummary`
  (`compiler/internal/semantics/semantics_checker.go:4546`).

## confirmed_gaps

- Typed path API is not present in the reviewed implementation. Current path encodings remain plain
  strings with helper functions and at least one prefix/tail helper
  (`compiler/internal/semantics/resources/paths.go:10`,
  `compiler/internal/semantics/resources/paths.go:43`).
- A canonical `FlowState` package with join laws/digest behavior is not present. Current flow joins
  are implemented locally in `semantics_checker.go`, while `compiler/internal/semantics/flow` only
  contains return-summary clone/equality helpers in the reviewed tree
  (`compiler/internal/semantics/semantics_checker.go:13875`,
  `compiler/internal/semantics/flow/return_summaries.go:5`).
- Loop analysis is not a fixed-point/digest loop. Reviewed `while`/`for` code analyzes the loop body
  once and joins it with zero-iteration, break, and continue flows
  (`compiler/internal/semantics/semantics_checker.go:17770`,
  `compiler/internal/semantics/semantics_checker.go:17881`).
- Interprocedural analysis is an order/changed-loop over functions with `maxIter := funcCount + 1`,
  not an explicit call-graph SCC/digest fixed point
  (`compiler/internal/semantics/semantics_checker.go:1213`,
  `compiler/internal/semantics/semantics_checker.go:1588`).
- Throw resource summaries exist, but unknown throw-resource provenance is not represented as an
  unknown summary in the reviewed throw path. Unknown throw provenance is skipped at the local throw
  site and during try-call propagation (`compiler/internal/semantics/semantics_checker.go:16497`,
  `compiler/internal/semantics/semantics_checker.go:15203`).
- The reviewed tests cover many partial move and reinitialization cases, but they do not cover the
  full roadmap matrix across all combinations of same-module/cross-module/interface-only,
  callable/generic, resource aggregate, and recursion surfaces.
- Await invalidation is implemented for borrowed-parameter-derived views after `await`/`try await`,
  but the reviewed test surface does not show the full expanded matrix for nested optional/struct
  paths, `inout`, callable captures, resource captures, and external pointer/source-vs-interface
  combinations.
- Defer capture safety primitives exist in implementation code, but no reviewed test path matched
  `defer`; defer/resource/throw summary isolation is therefore not end-to-end demonstrated by the
  reviewed tests.

## no_longer_missing_roadmap_items

- Function signatures already receive return region summaries, return resource summaries, and throw
  resource summaries during checker analysis
  (`compiler/internal/semantics/semantics_checker.go:1430`,
  `compiler/internal/semantics/semantics_checker.go:1465`,
  `compiler/internal/semantics/semantics_checker.go:1495`).
- Local control-flow joins already preserve and merge ownership/resource/finalization/await state
  across branches, matches, and loop exits
  (`compiler/internal/semantics/semantics_checker.go:13935`,
  `compiler/internal/semantics/semantics_checker.go:14057`).
- Return/throw terminal branches do not poison fallthrough ownership state in reviewed tests
  (`compiler/tests/ownership/ownership_scalar_escape_test.go:798`).
- Field, enum-payload, and optional/array-element path encodings already exist and are visible in
  tests/diagnostics as `.field`, `$caseN.payloadM`, and `$elem`
  (`compiler/internal/semantics/resources/paths.go:22`,
  `compiler/internal/semantics/resources/paths.go:29`,
  `compiler/tests/ownership/ownership_optional_payload_escape_test.go:728`).
- Partial move behavior for struct fields, enum payloads, optional payloads, and basic
  reinitialization already has reviewed implementation and tests
  (`compiler/tests/ownership/ownership_escape_assignment_test.go:160`,
  `compiler/tests/ownership/ownership_escape_assignment_test.go:516`,
  `compiler/tests/ownership/ownership_optional_payload_escape_test.go:728`,
  `compiler/tests/ownership/ownership_escape_assignment_test.go:267`).
- Resource-return provenance already supports aggregate resource leaves, wrappers, and cross-module
  cases in the reviewed implementation/tests
  (`compiler/internal/semantics/semantics_checker.go:15042`,
  `compiler/tests/runtime/resource_finalization/resource_finalization_actor_consume_test.go:9`,
  `compiler/tests/ownership/ownership_scalar_escape_test.go:741`).
- Borrow-after-await invalidation already exists and has direct async tests
  (`compiler/internal/semantics/semantics_memory_resources.go:2186`,
  `compiler/tests/semantics/semantics_async_ownership_test.go:958`).
- Defer capture checking already has implementation hooks in statement flow, even though reviewed
  tests do not yet demonstrate the behavior
  (`compiler/internal/semantics/semantics_checker.go:16298`,
  `compiler/internal/semantics/semantics_checker.go:18307`).
