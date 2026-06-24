# reviewed_commit

Reviewed commit: `3d101fbc3e1d8d9a9710c44725372ea086287c9c`.

Scope inspected for this current-state audit:

- `compiler/internal/semantics/policy/**`
- `compiler/internal/semantics/model/types.go`
- `compiler/internal/plir/**`
- `compiler/internal/memoryfacts/**`
- `compiler/internal/allocplan/**`
- `compiler/internal/buildreports/**`
- current `FeatureRegistry` location: `compiler/compiler_facade.go`
- `tools/cmd/validate-diagnostic/**`

Graphify note: this worktree has no `graphify-out/` directory, so concrete evidence was verified with `rg`/`sed`/`git` read-only inspection.

# canonical_effect_vocabulary

The canonical effect vocabulary is currently centralized in `compiler/internal/semantics/policy/effects.go`: `canonicalEffects` contains `actors`, `alloc`, `budget`, `capability`, `control`, `io`, `islands`, `link`, `mem`, `mmio`, `privacy`, `runtime`, and `surface` at `compiler/internal/semantics/policy/effects.go:18`.

Aliases and permission marker names are separate from the canonical set: `cap.io -> io`, `cap.mem -> mem`, and marker effects `capsule.io`/`capsule.mem` are accepted by `CanonicalizeEffectName` at `compiler/internal/semantics/policy/effects.go:34` and `compiler/internal/semantics/policy/effects.go:39`.

Effect groups are also in the same file: `effects.all`, `effects.cap.io`, `effects.cap.mem`, `effects.memory`, `effects.policy`, and `effects.runtime` are defined at `compiler/internal/semantics/policy/effects.go:44`. `NormalizeEffects` delegates to `NormalizeEffectDecl` and returns `SortedEffectSet(normalized.Declared)`, so the current normalized list is sorted by this helper at `compiler/internal/semantics/policy/effects.go:86` and `compiler/internal/semantics/policy/effects.go:138`.

The semantics package delegates normalization to the policy package rather than keeping a separate vocabulary: `normalizeEffects`, `normalizeEffectDecl`, and `sortedEffectSet` forward to `semanticspolicy` at `compiler/internal/semantics/semantics_memory_resources.go:689`.

Semantic clauses are modeled separately from effect names. `FunctionClausePolicy` stores `noalloc`, `noblock`, `realtime`, `budget`, `privacy`, and `consent` data at `compiler/internal/semantics/policy/clauses.go:11`; `ValidateSemanticClauses` accepts and validates those clause names at `compiler/internal/semantics/policy/clauses.go:21`; `ParseFunctionClausePolicy` projects them into booleans/value fields at `compiler/internal/semantics/policy/clauses.go:133`.

Call-policy helpers use `model.FuncSig.Effects` directly. `NoblockForbiddenCallEffects` and `RealtimeForbiddenCallEffects` are listed at `compiler/internal/semantics/policy/calls.go:5`, `FuncSigHasEffect` reads `sig.Effects` at `compiler/internal/semantics/policy/calls.go:26`, and strict semantic call clauses are detected from `FuncSig` booleans at `compiler/internal/semantics/policy/calls.go:50`.

# function_summary_sources

The canonical in-memory function contract shape is currently `model.FuncSig`, reachable through `CheckedProgram.FuncSigs map[string]FuncSig` at `compiler/internal/semantics/model/types.go:5`. `FuncSig` contains public/generic flags, semantic clause flags, param/return/callable fields, region/resource summaries, effects, and mutable-global facts at `compiler/internal/semantics/model/types.go:140`.

Function declaration signatures are constructed directly in semantics. Generic function signatures are assigned to `checked.FuncSigs[fullName] = FuncSig{...}` at `compiler/internal/semantics/semantics_checker.go:926`; non-generic declared function signatures are assigned similarly at `compiler/internal/semantics/semantics_checker.go:1068`.

Core builtins are another direct source of `FuncSig` data: `sigs := map[string]FuncSig{...}` starts at `compiler/internal/semantics/semantics_core.go:137`.

Function-typed field/global/helper sources can also synthesize reduced `FuncSig` values. `functionFieldInfoSig` returns a `FuncSig` from `FunctionFieldInfo` at `compiler/internal/semantics/semantics_expressions.go:8915`, and `funcSigFromDeclForGlobalInitializer` returns a declaration-derived `FuncSig` for imported function global initializers at `compiler/internal/semantics/semantics_expressions.go:9147`.

Semantic policy fields are parsed through `parseFunctionClausePolicy`, which maps `semanticspolicy.ParseFunctionClausePolicy` into local booleans and consent metadata at `compiler/internal/semantics/semantics_checker.go:13380`. Function policy validation then checks budget/noalloc/noblock/realtime/privacy/consent compatibility against declared effects and signature types at `compiler/internal/semantics/semantics_checker.go:13396`.

Direct and callback call checks consume `FuncSig` directly. `validateCallAgainstSemanticClauseTarget` checks realtime/noalloc/noblock/budget constraints at `compiler/internal/semantics/semantics_expressions.go:1442`; `validateFunctionTypeCallableEffects` compares function-typed callable declared effects against target effects at `compiler/internal/semantics/semantics_checker.go:9339`.

# function_summary_projections

`plir.FunctionSummary` is the current PLIR projection type. It includes generic/public/async flags, param metadata, return/throws metadata, `Effects`, `TouchesMutableGlobals`, and return/throw region/resource summaries at `compiler/internal/plir/plir.go:33`.

PLIR generation receives source summaries from `checked.FuncSigs`: `FromCheckedProgram` stores `checked.FuncSigs` into the builder at `compiler/internal/plir/plir.go:367`, and each emitted `Function` receives `Summary: b.functionSummary()` at `compiler/internal/plir/plir.go:619`.

`b.functionSummary()` copies selected fields from `semantics.FuncSig` into `FunctionSummary`: param names/types/ownership, return/throws, effects, mutable-global flag, region/resource unknown flags, and cloned region/resource summaries at `compiler/internal/plir/plir.go:643`.

Return ownership is partly normalized during PLIR projection: `summaryReturnOwnership` defaults borrowed memory returns to `"borrow"` when return-region data exists, even if `sig.ReturnOwnership` is empty, at `compiler/internal/plir/plir.go:672`.

`memoryfacts/fromplir` consumes the PLIR summary and validates summary completeness before adding facts: `Build` calls `validatePLIRFunctionSummaryCompleteness` and then `addPLIRFacts` at `compiler/internal/memoryfacts/fromplir/from_plir.go:9`; the validator delegates to `plir.VerifyFunctionSummaryCompleteness` at `compiler/internal/memoryfacts/fromplir/from_plir.go:28`.

Function-summary facts are produced through multiple `fromplir` projections. `addFunctionSummaryFacts` chains return-summary, declared-summary, operation-summary, and fact-kind-summary projections at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:10`.

Declared PLIR summary fields become memory facts in `addDeclaredSummaryFacts`: return-region, return-resource, throw-resource, effects, capability requirements, cap.mem authorization, and mutable-global facts are emitted from `fn.Summary` at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:112`.

Operation and value scans also project summary-like facts: return operations can emit `returns_unknown_unsafe`, `returns_owned_new_allocation`, or `returns_borrow_from_param` at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:28`; operation scans emit global-store, actor, closure, task, unknown-external, FFI, and pointer-retention facts at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:271`.

`allocplan` and reports are downstream projections. `allocplan.Build` consumes `memoryfacts.Snapshot` plus PLIR values at `compiler/internal/allocplan/build.go:17`; `WrapAllocationPlanReport` wraps an `allocplan.Plan` and recomputes report summary with `allocplan.Summarize` at `compiler/internal/buildreports/types.go:151`.

# memory_fact_consumers

`memoryfacts.Graph` is the current memory fact container. `Fact` stores function/value/site/type, provenance, region/domain/transfer, borrow/escape/alias, storage, proof, validation, source stage, parent/derived facts, claim, decision code, validator, and cost class fields at `compiler/internal/memoryfacts/facts.go:147`.

`memoryfacts/fromplir.Build` is a PLIR-to-graph consumer/adapter: it creates a graph, adds representation metadata, validates PLIR summaries, adds PLIR facts, validates the graph, and returns it at `compiler/internal/memoryfacts/fromplir/from_plir.go:9`.

`memoryfacts.Snapshot` provides immutable indexed access for downstream consumers. `Graph.Snapshot` clones facts and indexes them by value/allocation/proof/parent at `compiler/internal/memoryfacts/snapshot.go:100`; `Snapshot.Digest` produces a deterministic graph digest from sorted facts at `compiler/internal/memoryfacts/snapshot.go:318`.

`allocplan.Build` is a downstream memory-fact consumer. It takes `Program *plir.Program` and `Snapshot memoryfacts.Snapshot` at `compiler/internal/allocplan/build.go:17`, resolves allocation evidence per PLIR allocation value at `compiler/internal/allocplan/build.go:39`, verifies PLIR/evidence consistency at `compiler/internal/allocplan/build.go:51`, and assigns allocation plan digests at `compiler/internal/allocplan/build.go:70`.

`resolvePlannerAllocation` merges facts for a value into `memoryfacts.AllocationEvidence`, collecting provenance/unsafe class, region/island/domain/transfer/lifetime/escape/proof data and `SourceFactIDs` at `compiler/internal/allocplan/build.go:268`.

Lowering evidence is fed back into memoryfacts through `memoryfacts/fromlowering`: `AddFacts` builds a delta, applies it to the graph, and validates at `compiler/internal/memoryfacts/fromlowering/from_lowering.go:11`; each lowering fact uses the first source fact as parent and records storage/lowered artifact fields at `compiler/internal/memoryfacts/fromlowering/from_lowering.go:33`.

`allocplan/plan.go` also reads PLIR function summaries directly for explicit-island handle slot discovery: it checks `fn.Summary.ParamNames` and `fn.Summary.ParamTypes` at `compiler/internal/allocplan/plan.go:574`.

# direct_funcsig_constructors

Direct production `FuncSig{...}` construction remains present. A read-only `rg` scan found production occurrences in `compiler/internal/semantics`, `compiler/internal/lower`, and `compiler/internal/opt`; representative examples follow.

Declared and generic source functions are direct literals in `semantics_checker`: generic signatures at `compiler/internal/semantics/semantics_checker.go:926` and non-generic signatures at `compiler/internal/semantics/semantics_checker.go:1068`.

Core builtin signatures are direct literals in `semantics_core`: the builtin signature map starts at `compiler/internal/semantics/semantics_core.go:137`.

Function-typed helper paths synthesize direct literals: `functionFieldInfoSig` at `compiler/internal/semantics/semantics_expressions.go:8915`, returned function type validation at `compiler/internal/semantics/semantics_expressions.go:9028`, and imported global initializer signatures at `compiler/internal/semantics/semantics_expressions.go:9189`.

Lowering creates temporary `semantics.FuncSig` values for dynamic function values in `try` lowering at `compiler/internal/lower/lower_expressions.go:1113`.

Optimizer support code contains direct `semantics.FuncSig` map literals for module-summary fixtures at `compiler/internal/opt/opt_core.go:4638`.

No centralized `buildDeclaredFuncSig`, `buildBuiltinFuncSig`, `ValidateFuncSigContract`, or `CloneFuncSig` symbol was found in the reviewed worktree by read-only symbol search.

# summary_copy_helpers

PLIR summary projection uses defensive slice/map copies for copied fields: param names/types/ownership and effects are copied with `append([]string(nil), ...)` at `compiler/internal/plir/plir.go:651`; return-region maps are copied by `cloneIntMap` at `compiler/internal/plir/plir.go:686`; return/throw resource summaries are copied by `cloneResourceSummary` at `compiler/internal/plir/plir.go:697`.

`memoryfacts` snapshots clone facts when exposing or indexing graph state. `factsForIDs` returns cloned facts at `compiler/internal/memoryfacts/snapshot.go:398`; `Graph.clone` clones graph order and fact contents at `compiler/internal/memoryfacts/snapshot.go:439`; `cloneFact` copies derived fact IDs and param-index pointers at `compiler/internal/memoryfacts/graph.go:454`.

Vocabulary list accessors return defensive copies through `copyStrings`, including source stages, provenance classes, unsafe classes, alias states, storage classes, report claims, and memory fuzz statuses at `compiler/internal/memoryfacts/vocabulary.go:599`.

PLIR proof-state helpers clone local proof maps across branches: `snapshotLocalProofState`, `restoreLocalProofState`, and `mergeLocalProofState` use map clone helpers at `compiler/internal/plir/plir_proofs.go:229`; `cloneBoolMap`, `cloneInt64Map`, and `cloneStringMap` are at `compiler/internal/plir/plir_proofs.go:304`.

# duplicate_or_shadow_projections

There is one direct PLIR summary projection from `FuncSig`, but multiple downstream memoryfact projections from PLIR summary, PLIR operations, PLIR values, and PLIR facts.

The direct projection is `b.functionSummary()` copying `semantics.FuncSig` into `plir.FunctionSummary` at `compiler/internal/plir/plir.go:643`.

`addDeclaredSummaryFacts` projects declared `fn.Summary` fields into memory facts at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:112`.

`addReturnSummaryFacts` independently scans return operations and PLIR value/fact provenance to emit return summary facts at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:28`.

`addOperationSummaryFacts` independently scans operation kinds and operation notes to emit global-store, actor/task/closure, unknown-external, FFI, noalias-invalidation, and pointer-retention facts at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:271`.

`addFactKindSummaryFacts` independently scans PLIR facts and values to emit `may_consume_param` and `may_mutate_inout` facts at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:667`.

Allocation escape summaries are another projection surface. `addAllocationIntentSummaryFacts` emits summary facts from allocation values and classified escape state at `compiler/internal/memoryfacts/fromplir/from_plir_allocation_escape.go:15`; `classifyAllocationEscapeForPLIR` derives escape from PLIR value state, operations, unsafe/aggregate/closure boundaries, and local call-summary proofs at `compiler/internal/memoryfacts/fromplir/from_plir_allocation_escape.go:295`.

Local no-escape call summaries are inferred by scanning callee PLIR summaries and operations, not by a contract digest: `buildAllocationReadOnlyCallSummaries` reads `fn.Summary` plus operation scans at `compiler/internal/memoryfacts/fromplir/from_plir_allocation_escape.go:415`.

Backend reports include report-only runtime/effect inference from IR op kinds and runtime call names. `classifyBackendFallback` uses `backendCallLooksRuntimeEffect` and effect-runtime IR kind checks at `compiler/internal/buildreports/backend.go:206`; `backendCallLooksRuntimeEffect` classifies names by prefixes `__tetra_`, `runtime.`, and `core.` at `compiler/internal/buildreports/backend.go:249`.

# report_only_dependencies

`FeatureRegistry` currently lives in `compiler/compiler_facade.go`, not `compiler/features.go`. The registry types and defensive-copy return are at `compiler/compiler_facade.go:1773` and `compiler/compiler_facade.go:3919`.

Feature registry entries mark effects/capabilities/privacy/budget/production safety as current release-truth metadata. `safety.effects-mvp` states stable effect names/groups, transitive call propagation, diagnostics, and PLIR optimizer facts at `compiler/compiler_facade.go:2512`; `safety.capabilities-mvp`, `safety.privacy-consent-mvp`, `safety.budget-mvp`, and `safety.production-core` follow at `compiler/compiler_facade.go:2533`, `compiler/compiler_facade.go:2552`, `compiler/compiler_facade.go:2569`, and `compiler/compiler_facade.go:2586`.

`BuildAllocReport` is a legacy/report surface over IR allocation instructions and emits conservative allocation rows directly from IR kinds at `compiler/internal/buildreports/types.go:101`. The newer allocation-plan report wraps an `allocplan.Plan`, derives summary from `allocplan.Summarize`, and validates report equality against the plan at `compiler/internal/buildreports/types.go:151` and `compiler/internal/buildreports/types.go:179`.

Backend reports derive runtime feature requirements from lowered IR, not `FuncSig` effects. `collectBackendRuntimeFeatures` scans IR calls/op kinds at `compiler/internal/buildreports/backend.go:448`; `backendRuntimeFeaturesForIRKind` maps allocation/region/island/io/memory IR kinds to runtime feature names at `compiler/internal/buildreports/backend.go:544`; `backendRuntimeFeatureForCall` maps runtime call names to runtime feature names by string patterns at `compiler/internal/buildreports/backend.go:639`.

Layout reports read `checked.FuncSigs` only to find exported ABI type uses: `exportedLayoutABITypeUses` walks exported functions and their signature param/return types at `compiler/internal/buildreports/layout.go:237`.

`tools/cmd/validate-diagnostic` validates diagnostic JSON shape and requested code/severity/message content. It parses strict JSON at `tools/cmd/validate-diagnostic/main.go:68` and validates required code/message/severity/position fields at `tools/cmd/validate-diagnostic/main.go:76`; it does not consume `FuncSig`, PLIR summary, memoryfacts, or effect vocabulary directly.

# confirmed_gaps

No `FunctionContractV1`, `ProjectFunctionContractV1`, `ContractDigest`, `ContractSchema`, `FunctionSummaryFromFuncSig`, `ValidateFuncSigContract`, or `CloneFuncSig` implementation is present in the reviewed worktree by read-only symbol search.

`model.FuncSig` has semantic contract fields but no schema/digest fields at `compiler/internal/semantics/model/types.go:140`.

`plir.FunctionSummary` has no `ContractSchema` or `ContractDigest` fields at `compiler/internal/plir/plir.go:33`.

PLIR summary construction is a builder method, `b.functionSummary()`, rather than a standalone constructor such as `FunctionSummaryFromFuncSig`; the method copies fields directly from `sig` at `compiler/internal/plir/plir.go:643`.

Memoryfacts encode summary-derived claims but do not retain a source function-contract digest. `memoryfacts.Fact` has no digest/source-contract field in its current schema at `compiler/internal/memoryfacts/facts.go:147`, and `functionSummaryFact` constructs facts without contract metadata at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:731`.

Direct production `FuncSig{...}` literals remain in multiple packages; representative current constructors are at `compiler/internal/semantics/semantics_checker.go:926`, `compiler/internal/semantics/semantics_checker.go:1068`, `compiler/internal/semantics/semantics_core.go:137`, `compiler/internal/semantics/semantics_expressions.go:9189`, `compiler/internal/lower/lower_expressions.go:1113`, and `compiler/internal/opt/opt_core.go:4638`.

Multiple memoryfact projection surfaces can shadow or duplicate declared PLIR summaries: declared summary facts at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:112`, return-operation facts at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:28`, operation-summary facts at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:271`, fact-kind summary facts at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:667`, and allocation escape summary facts at `compiler/internal/memoryfacts/fromplir/from_plir_allocation_escape.go:15`.

Backend report code still infers runtime/effect-like categories from runtime call names and IR kinds, not semantic metadata, at `compiler/internal/buildreports/backend.go:206`, `compiler/internal/buildreports/backend.go:249`, and `compiler/internal/buildreports/backend.go:448`.

`allocplan` has per-allocation `PlanDigest`, but this digest covers allocation plan/options, not a source `FuncSig` contract digest: `PlanDigest` is an allocation field at `compiler/internal/allocplan/plan.go:98`, and `allocationPlanDigest` serializes schema `tetra.allocplan.v2`, function, options, and allocation at `compiler/internal/allocplan/build.go:771`.
