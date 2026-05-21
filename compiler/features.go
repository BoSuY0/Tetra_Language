package compiler

// FeatureStatus is the release-truth lifecycle label for a public Tetra feature.
type FeatureStatus string

const (
	FeatureStatusCurrent      FeatureStatus = "current"
	FeatureStatusExperimental FeatureStatus = "experimental"
	FeatureStatusPlanned      FeatureStatus = "planned"
	FeatureStatusPostV1       FeatureStatus = "post-v1"
)

// FeatureInfo is a machine-readable release-truth registry entry.
type FeatureInfo struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Status    FeatureStatus `json:"status"`
	Since     string        `json:"since,omitempty"`
	Scope     string        `json:"scope"`
	Stability string        `json:"stability"`
	Docs      []string      `json:"docs"`
}

// FeatureRegistry returns the canonical feature status registry for the current
// compiler/tooling surface. Keep this list conservative: current entries must
// reflect the supported surface, while future work stays planned or post-v1
// until promoted with release-gate evidence.
func FeatureRegistry() []FeatureInfo {
	features := []FeatureInfo{
		{
			ID:        "cli.core",
			Name:      "Core CLI workflows",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "check/build/run/fmt/test/doc/doctor/targets/features/formats/new/interface/project/workspace/smoke/eco/clean/version/lsp local workflows",
			Stability: "supported in the current v0.4.0 local profile",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/cli_contracts.md", "docs/user/cli_cheatsheet.md"},
		},
		{
			ID:        "targets.native",
			Name:      "Native target builds",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "linux-x64 build/run plus macos-x64 and windows-x64 build-only release coverage",
			Stability: "supported target metadata is validated by release checks",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/cli_contracts.md"},
		},
		{
			ID:        "targets.wasm-artifact-preflight",
			Name:      "WASM artifact/import preflight",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "wasm32-wasi and wasm32-web artifact/import validation through smoke --run=false, with runtime execution covered by wasm.runtime-execution",
			Stability: "current deterministic artifact/import validation; this is not runtime proof by itself",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/backend/wasm_backend_plan.md"},
		},
		{
			ID:        "language.flow",
			Name:      "Flow syntax profile",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "release-covered indentation syntax in examples, stdlib, runtime, and self-host snippets",
			Stability: "supported source syntax for the current release gate",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md"},
		},
		{
			ID:        "language.generics-mvp",
			Name:      "Static monomorphized generics MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "generic functions with inferred value arguments are statically monomorphized across modules; no runtime generic values or dynamic dispatch",
			Stability: "supported static MVP; explicit type arguments, generic structs, higher-ranked generics, full protocol-bound generic dispatch, and specialization optimization remain future/post-v1",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"},
		},
		{
			ID:        "language.protocol-conformance-mvp",
			Name:      "Static protocol conformance MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "protocol declarations and impl conformance are checked statically against extension/static methods, including generic requirement signature shape; no witness tables, trait objects, or dynamic dispatch model",
			Stability: "supported static conformance MVP; runtime polymorphism and dynamic dispatch remain post-v1 unless separately gated",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"},
		},
		{
			ID:        "language.callable-mvp",
			Name:      "Callable/function type MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "Level 0 callable surface: function type references, narrow symbol-backed non-capturing callable paths, and legacy ptr closure local direct calls",
			Stability: "current constrained MVP; captured closure escape, storage, and full first-class function values remain out of scope",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_feature_status.md"},
		},
		{
			ID:        "language.callable-level1",
			Name:      "Callable Level 1 non-capturing expansion",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "production non-capturing symbol-backed callable Level 1: function-typed locals, aliases, callbacks, including target-set-backed function-typed parameter aliases, function-typed parameter storage into struct fields with direct field calls or synchronous callback arguments, function-typed parameter storage into enum payloads with direct payload calls, reassignment, returned enum propagation, or synchronous callback arguments, optional argument labels on function-typed value calls including captured fnptr locals with mixed labeled/unlabeled lists rejected, symbol-backed returns, declared function-typed local binding, symbol-backed function-typed globals for same-module or namespace/selective imported public direct calls plus local initialization/reassignment/direct callback arguments, non-capturing closure-literal function-typed globals, same-module mutable global reassignment with direct calls, synchronous callback arguments, function-typed returns, generated .t4i function-typed parameter local-alias return metadata, and local or nested local struct-field/enum-payload storage/reassignment/returned-aggregate propagation, imported mutable function-typed global boundary diagnostics, actor/task boundary diagnostics across core.spawn, core.task_spawn_i32, core.task_spawn_i32_typed, core.task_spawn_group_i32, and core.task_spawn_group_i32_typed for workers that directly dispatch through same-module or imported immutable function-typed globals whose targets touch mutable globals, pass mutable function-typed globals as synchronous callback arguments, pass same-module or imported symbol-backed callback arguments whose targets touch mutable globals, pass same-module or imported direct function-typed return-call callback arguments whose returned targets or multi-return target sets touch mutable globals, preserve that classification through local/field alias returns and returned struct/enum aggregate fields or payloads across module boundaries, directly call function-typed locals/struct fields/enum payloads whose targets touch mutable globals, reassign them into function-typed locals or local struct fields/enum payloads, store them into local function-typed struct fields/enum payloads, return them from function-typed return helpers, or write mutable function-typed globals, and inferable same-module/imported generic-symbol initializers, non-capturing generic closure literal binding/direct callback/return/mutable local or nested struct field reassignment/nested struct field initializer/enum payload initializer or reassignment, function-typed returns including target-set-backed function-typed parameter returns and direct returned-call callback arguments, mutable local and nested struct field reassignment, function-typed nested struct field initializers, and enum payload initializers for inferable same-module or imported generic symbols, and signature-compatible mutable local reassignment with stable diagnostics",
			Stability: "current constrained Level 1; generic callable movement is limited to declared local initializers, symbol-backed function-typed global initializers, same-module mutable global reassignment/returns and local or nested local struct-field/enum-payload storage/reassignment/returned-aggregate propagation, direct callback arguments, function-typed returns, mutable local or nested struct field reassignment, struct field initializers, and enum payload initializers; captured closure escape beyond the fnptr Level 2 slice, captured/global-escaping callable storage beyond the same-module symbol-backed mutable global snapshot/reassignment/return slice, and full first-class function values remain out of scope",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_feature_status.md"},
		},
		{
			ID:        "language.callable-level2",
			Name:      "Callable Level 2 captured closure fnptr values",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "production captured closure Level 2 slice: local Int/Bool/String/simple-struct/enum/optional captures without ptr/resource payloads may enter fnptr-backed function-typed locals, captured ptr closure aliases into function-typed locals, mutable function-typed local reassignment, same-module mutable function-typed global snapshot reassignment from direct closure literals, let-bound captured ptr closure locals, direct same-module/imported function-typed return calls, immutable local aliases initialized from those return calls, mutable function-typed locals, local/nested struct fields, local enum payloads, whole local or nested structs with function fields reassigned from struct literals containing direct closure literals or direct return calls, whole local enums reassigned from enum constructors containing direct closure literals or direct return calls, or same-module or source-imported returned enum payloads or returned struct enum payloads carrying direct closure literals, with generated `.t4i` interface-only returned direct enum or aggregate stubs preserving payload metadata for API-only validation, or return alias chains that return captured closure snapshots with later direct calls, synchronous callback arguments, same-module or cross-module function-typed returns, direct callback arguments after cross-module returns, mutable local reassignments after cross-module returns, local or cross-module returned struct-field initializer/reassignment, local or cross-module returned enum-payload initializer/reassignment, or throwing direct-try dispatch through that global, direct synchronous callback arguments including direct closure literals passed to imported callbacks, function-typed returns including direct return of let-bound captured ptr closure values, local struct fields or enum payloads including direct closure-literal container initializers in module-aware lowering, direct calls including labeled direct calls on captured ptr closures, synchronous callback parameters including imported parameter-return callbacks, cross-module returned captured closures used through locals or direct callback arguments, cross-module struct-parameter function-field dispatch including namespace/selective imported direct struct constructors carrying closure literals or captured ptr closure locals, cross-module enum-parameter function-payload dispatch including direct namespace/selective imported enum constructor arguments, immutable local struct fields or enum payloads with up to eight by-value snapshot environment slots, explicitly declared immutable local direct-try bindings to throwing function symbols or captured throwing closure literals, captured throwing closure literals in mutable local reassignment, direct callback arguments, function-typed returns, immutable local struct-field or enum-payload direct-try dispatch and aliases, and mutable local struct-field or enum-payload reassignment direct-try dispatch, declared function-typed returns of a concrete throwing symbol followed by local direct-try dispatch, immutable local struct-field and enum-payload direct-try dispatch for concrete throwing symbols, immutable same-module or imported-public function-typed global direct-try dispatch/local alias/mutable local reassignment/direct callback/struct-field initializer/struct-field reassignment/enum-payload reassignment paths for concrete throwing symbols, same-module mutable function-typed global direct-try dispatch, direct throwing callback arguments, and local struct-field/enum-payload storage direct-try after compatible concrete throwing-symbol initialization or reassignment, and direct synchronous throwing callback-parameter dispatch through `try cb(...)` when the callback parameter type declares the same throws type",
			Stability: "current constrained Level 2 fast path; larger immutable environments are promoted under language.full-first-class-callables, while by-reference mutable capture, pointer/resource capture, thread escape, unsupported assignment sources, and generic/runtime callable polymorphism beyond statically inferred function-type surfaces report stable diagnostics or remain governed by explicit future features",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_feature_status.md"},
		},
		{
			ID:        "language.semantic-clauses-mvp",
			Name:      "Semantic clause checker MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "phase-1 noalloc/noblock/realtime checks on resolved direct and supported callable paths",
			Stability: "static checker MVP; proof-level guarantees remain future work",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/v1_feature_status.md"},
		},
		{
			ID:        "safety.effects-mvp",
			Name:      "Effects and uses checker MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.3.0",
			Scope:     "stable uses effect names and groups with transitive call propagation across resolved direct, generic, protocol, and supported callable paths; missing uses declarations are diagnostics",
			Stability: "supported static MVP; no effect inference or proof-level effect system guarantee is claimed",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/effects_capabilities_privacy_v1.md", "docs/spec/capabilities.md"},
		},
		{
			ID:        "safety.capabilities-mvp",
			Name:      "Capabilities and unsafe boundary MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.3.0",
			Scope:     "cap.io and cap.mem opaque tokens are obtained only inside unsafe blocks; raw memory/MMIO operations require the matching uses effects, unsafe boundary, capability argument, and capsule permissions for attenuated groups",
			Stability: "supported compile-time gating MVP; not a broad safe-code capability construction model and current MMIO/raw-memory lowering remains minimal",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/capabilities.md", "docs/spec/unsafe.md", "docs/spec/effects_capabilities_privacy_v1.md"},
		},
		{
			ID:        "safety.privacy-consent-mvp",
			Name:      "Privacy and consent checker MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.3.0",
			Scope:     "uses privacy requires privacy semantic clauses; secret.i32/SecretInt signatures and privacy builtins require a consent token parameter with consent.token type",
			Stability: "supported static auditing and call-shape MVP; not cryptographic isolation, and distributed consent enforcement remains post-v1",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/effects_capabilities_privacy_v1.md", "docs/spec/stdlib.md"},
		},
		{
			ID:        "safety.budget-mvp",
			Name:      "Budget clause lowering MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.3.0",
			Scope:     "budget(<non-negative integer constant>) requires uses budget, lowers to deterministic budget guard instructions with stable local-slot metadata, and enforces conservative direct-call/task/actor budget context guardrails",
			Stability: "supported local lowering plus static edge guardrail MVP; not cross-function runtime-wide aggregate accounting, and distributed budget enforcement remains post-v1",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/effects_capabilities_privacy_v1.md"},
		},
		{
			ID:        "safety.production-core",
			Name:      "Production safety core",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "production local safety model for ownership/lifetime/borrow/consume/inout checks, resource finalization, callable escape diagnostics, effects/capabilities/privacy/consent/budget policy, unsafe boundaries, actor/task transfer safety, and pointer/MMIO/memory capability gates",
			Stability: "release-gated current profile with explicit diagnostics for unsupported distributed, cryptographic, formal-proof, and runtime-wide guarantees",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/effects_capabilities_privacy_v1.md"},
		},
		{
			ID:        "language.globals-properties-capsule-mvp",
			Name:      "Top-level globals, properties, and capsule metadata MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "constant global initializers, property declarations, and compile-time capsule metadata validation",
			Stability: "supported MVP with explicit initializer/runtime limitations",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md"},
		},
		{
			ID:        "language.slice-mvp",
			Name:      "Native-first slice MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "[]u16 and []bool helpers including make_* and island compile-compatible fallback paths",
			Stability: "supported MVP with documented layout/runtime constraints",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/stdlib.md"},
		},
		{
			ID:        "language.ownership-markers-mvp",
			Name:      "Ownership markers MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "conservative borrow/inout/consume marker checks for local calls, same-module/cross-module struct-field and enum-payload partial consume with whole-value call/let/return and enum wrapper-constructor rejection plus stable TETRA2101 diagnostics including same-module/cross-module CLI JSON evidence, same-module/cross-module whole-copy rejection after partial struct/enum consume with stable TETRA2101 CLI JSON evidence, mutable struct-field/whole-struct/whole-enum reinitialization after partial consume, aliasing, use-after-consume, and borrow escape diagnostics for scalar ptr including same-module/cross-module scalar ptr consume and inout assignment plus match/catch-expression return escapes and typed-error throw ptr/region payload escapes, same-module/cross-module borrowed scalar ptr escapes through ptr-containing struct inout assignment, same-module/cross-module fixed-array alias return plus direct global assignment, optional global assignment, and inout assignment escapes with stable TETRA2102 diagnostic evidence, borrowed string alias return/global assignment escapes with stable TETRA2102 CLI JSON evidence, slice-containing struct literal/alias/nested struct/enum-payload return and inout assignment escapes plus slice-containing enum direct/alias return escapes with stable same-module/cross-module TETRA2102 CLI JSON evidence, slice-containing struct/enum owned/consume/inout call escapes with stable same-module/cross-module and imported direct TETRA2101 CLI JSON evidence, function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence, ptr/slice optional assignment return/owned/consume/inout escape with stable same-module/cross-module TETRA2101/TETRA2102 CLI JSON evidence for slice optional assignment, same-module/cross-module slice optional payload binding owned/consume/inout call, inout-assignment, and global assignment escapes with stable TETRA2101/TETRA2102 CLI JSON evidence, same-module/cross-module ptr optional assignment if-let/match global escape with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module ptr enum alias return escape with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module slice optional-payload inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module nested slice enum-payload return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module nested slice struct return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module direct slice global assignment with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module optional ptr global assignment with stable TETRA2102 JSON diagnostic evidence, and same-module/cross-module optional aggregate global assignment with stable TETRA2102 JSON diagnostic evidence, and same-module/cross-module ptr-containing aggregate whole/field/alias/nested-field return escapes with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module whole-aggregate global assignment with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module ptr-containing enum whole-value global assignment with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module global field target assignment with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module aggregate and nested-aggregate global field escapes with stable TETRA2102 JSON diagnostic evidence, same-module/cross-module ptr-containing and nested ptr-containing aggregates plus ptr-containing enum aggregates including whole-aggregate, whole-enum, global field target, and global field escapes with stable TETRA2102 CLI JSON evidence, optional ptr payloads including same-module/cross-module whole-optional use-after-payload-consume diagnostics with stable TETRA2101 CLI JSON evidence and same-module/cross-module optional-payload whole-value rejection after payload consume/free with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module pattern-bound enum payload and if-let/match optional payload return, owned/consume/inout call, inout-assignment, and global escapes with same-module/cross-module ptr enum-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence and same-module/cross-module ptr optional-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence, plus same-module/cross-module ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module ptr enum-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module ptr optional-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence, and same-module/cross-module slice optional-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module generic aggregate and optional-ptr owned/consume/inout instantiations including slice-containing struct/enum aggregate instantiations with stable TETRA2101 CLI JSON evidence, same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence, same-module/cross-module protocol parameter ownership matching plus same-module/cross-module protocol impl parameter ownership mismatch diagnostics with stable TETRA2001 CLI JSON evidence and same-module/cross-module generic protocol requirement parameter ownership mismatch diagnostics with stable TETRA2001 JSON diagnostic evidence, same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence, imported direct owned/consume/inout call boundaries including struct, enum-payload, and nested ptr-containing aggregate arguments, with imported direct ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence, and supported mutable global assignment boundaries",
			Stability: "supported conservative MVP; this is not a full SSA lifetime solver and ambiguous lifetime merges remain diagnostics",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"},
		},
		{
			ID:        "language.resource-lifetime-mvp",
			Name:      "Resource lifetime MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "conservative resource finalization checks for task handles, task groups, island handles, region-backed slices, structs containing them, branch/match/loop task-handle maybe-joined, task-group maybe-closed, and island maybe-freed merge diagnostics; branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence, stable ownership safety JSON diagnostics for resource use-after-free, double-join, and ambiguous-provenance cases including same-module/cross-module struct-field and enum-payload alias use-after-free, same-module/cross-module struct-field and enum-payload alias use-after-free with stable TETRA2101 JSON diagnostic evidence, plus task-group use-after-close, struct-field aliases and enum-payload aliases including same-module/cross-module task-handle/task-group struct-field/enum-payload join/close aliases, same-module/cross-module task-handle struct-field/enum-payload alias join diagnostics with stable TETRA2101 JSON diagnostic evidence, and same-module/cross-module task-group struct-field/enum-payload alias close diagnostics with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module enum-constructor return resource aliases with stable TETRA2101 CLI JSON evidence, same-module typed-error throw/catch and rethrow-through-try enum-payload resource aliases with stable TETRA2101 JSON diagnostic evidence, generated .t4i direct/local/aggregate-local-alias/aggregate-field-access/aggregate-field-local-alias resource return, assignment/let/direct-if-let/direct-match/field-local/if-let/match optional and nested/field-local nested optional resource return, typed-error direct/field-local-alias throw, and rethrow-through-try direct/field-local-alias provenance stubs, same-module/cross-module monomorphized generic struct task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence, if-let/match optional-payload return aliases including nested struct-field and enum-payload wrappers with stable same-module/cross-module TETRA2101 CLI JSON evidence, same-module/cross-module task-handle/task-group if-let/match optional-payload join/close aliases with stable TETRA2101 CLI JSON evidence, same-module/cross-module island whole-optional use-after-payload-free diagnostics with stable TETRA2101 CLI JSON evidence, same-module/cross-module transitive interprocedural task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence, same-module and cross-module transitive interprocedural resource alias double-use, and ambiguous provenance diagnostics",
			Stability: "supported conservative MVP; tracks common local scope and control-flow merge cases, but is not a full SSA lifetime solver",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"},
		},
		{
			ID:        "actors.task-transfer-safety",
			Name:      "Actor/task transfer safety MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "conservative actor/task ownership transfer checks for worker entrypoints, sendable results, handle transfer, branch/match/loop actor consume reuse diagnostics with stable TETRA2101 CLI JSON evidence, actor/task use-after-transfer diagnostics with stable TETRA2101 CLI JSON evidence, island transfer non-local-payload rejection with stable TETRA2101 CLI JSON evidence, same-module/cross-module transitive actor consume alias diagnostics with stable TETRA2101 CLI JSON evidence, same-module/cross-module monomorphized generic struct actor consume alias diagnostics with stable TETRA2101 CLI JSON evidence, same-module/cross-module task_group_cancel return provenance diagnostics with stable TETRA2101 CLI JSON evidence, same-module/cross-module actor if-let/match optional-payload, struct-field, and enum-payload consume alias diagnostics, same-module/cross-module actor struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module actor/task if-let/match optional-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence, same-module/cross-module task-handle struct-field/enum-payload alias join diagnostics with stable TETRA2101 JSON diagnostic evidence, release-covered cooperative task_group_cancel wake/join behavior, and task group lifecycle status/close smokes",
			Stability: "supported conservative local MVP; distributed actors, full race-safety proofs, full cancellation semantics, and structured concurrency remain outside the current support claim",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md", "docs/user/async_actors_guide.md"},
		},
		{
			ID:        "language.lifetime-ssa",
			Name:      "Lifetime SSA local join solver",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "production SSA-like local lifetime join analysis for ownership consume state, resource finalization state, branch/match/loop flow snapshots, branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence, optional region-wrapper escapes with stable TETRA2102 diagnostics, same-module and interface-only cross-module per-field interprocedural region summaries for aggregate returns from multiple island parameters, including optional aggregate wrappers, enum payload wrappers, branch aggregate wrappers, match aggregate wrappers, if-let aggregate wrappers, mixed safe/provenance aggregate branch and match returns, and optional mixed safe/provenance aggregate branch merges, and maybe-consumed diagnostics",
			Stability: "current local/control-flow solver; richer interprocedural lifetime proofs, broad alias modeling, race proofs, and full formal lifetime guarantees remain under full-v1 scope",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"},
		},
		{
			ID:        "language.task-handles-mvp",
			Name:      "Typed task handle wrappers MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "typed task handle wrappers for slot counts 2..8 in the current runtime path",
			Stability: "supported MVP; layouts above 8 are rejected",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/user/async_actors_guide.md"},
		},
		{
			ID:        "eco.local-package-lifecycle",
			Name:      "Local Eco package lifecycle",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "local verify, lock generation/validation, pack/unpack, vault, stable and beta publish metadata, target-aware download, stable/beta TetraHub store fixtures, local mirror reports, and single-origin HTTP(S) fetch into a verified local store",
			Stability: "local tooling support with stable publish, mirror, and HTTP fetch integrity metadata; distributed network ecosystem is not implied",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/user/eco_package_guide.md", "docs/spec/eco_publishing_v1.md"},
		},
		{
			ID:        "stdlib.core-current",
			Name:      "Core standard library current profile",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "release-covered lib.core helper modules with a capability-gated linux-x64 filesystem exists slice, executable Linux TCP socket client/server I/O helpers with recv/send, SO_REUSEPORT, TCP_NODELAY, nonblocking accept convenience, and epoll add/mod/delete plus wait-one readiness flag capture and predicates, stable crypto interface helpers, stable networking endpoint policy helpers, executable HTTP/1.1 String and byte-buffer request-line routing, request-head framing, and response byte-buffer helpers, and executable JSON byte-buffer response helpers",
			Stability: "current import paths and smoke coverage; filesystem exists is host-backed on linux-x64, net socket open/bind/connect/listen/accept/read/recv/write/send/nonblocking/close plus SO_REUSEPORT, TCP_NODELAY, SOCK_NONBLOCK/SOCK_CLOEXEC accept helpers, and epoll create/add-read/add-read-write/mod-read/mod-read-write/delete/wait-one/wait-one-into helpers with EPOLLIN/EPOLLOUT/EPOLLERR/EPOLLHUP predicates are host-backed on linux-x64, crypto exposes deterministic interface helpers, networking exposes deterministic endpoint policy helpers, HTTP helpers classify TechEmpower request lines from String text or caller-owned byte buffers, locate CRLFCRLF request-head boundaries for pipelined buffers, and write compact response payloads into caller-owned buffers, and JSON helpers write compact response bodies into caller-owned buffers",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/stdlib.md", "docs/user/standard_library_guide.md"},
		},
		{
			ID:        "stdlib.experimental-mirrors",
			Name:      "Standard-library compatibility mirrors",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "production compatibility mirrors under lib.experimental.* forward to lib.core.* modules for legacy source compatibility",
			Stability: "current compatibility bridge; stable callers should import lib.core.* directly, and no broader host API guarantee is implied beyond the mirrored lib.core surface",
			Docs:      []string{"docs/spec/stdlib.md", "docs/spec/stdlib_naming_versioning.md", "docs/user/standard_library_guide.md"},
		},
		{
			ID:        "language.enum-payload-match",
			Name:      "Enum payload constructors and exhaustive match/catch",
			Status:    FeatureStatusCurrent,
			Since:     "v0.3.0",
			Scope:     "positional enum payload constructors and payload bindings for match/catch/if-let, with exhaustive unguarded enum match/catch coverage and stable diagnostics for arity, type, duplicate, default-order, and payload-syntax errors",
			Stability: "supported v0.3.0 static/runtime slice; cross-module enum constructor/match paths are checked and lowered, while advanced ADT constructors, nested destructuring patterns, richer payload algebra, and guard expansion remain future/post-v1",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v0_3_scope.md"},
		},
		{
			ID:        "language.protocol-bound-generics-static",
			Name:      "Static protocol-bound generics",
			Status:    FeatureStatusCurrent,
			Since:     "v0.3.0",
			Scope:     "generic function type parameters with protocol bounds are validated statically during monomorphization, including same-module and cross-module impl conformance with parameter ownership markers, requirement signature shape, and visibility diagnostics",
			Stability: "supported v0.3.0 static conformance slice; calling protocol requirements through generic bounds, witness tables, trait objects, runtime protocol values, and dynamic dispatch remain unsupported",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/v0_3_scope.md", "docs/spec/flow_syntax_v1.md"},
		},
		{
			ID:        "ui.metadata-v1",
			Name:      "UI metadata v0.4.0 surface",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "production UI metadata contract for checked view/state declarations, deterministic tetra.ui.v0.4.0 JSON, browser-backed web command-dispatch runtime artifacts, style metadata preview attributes, accessibility metadata preview attributes, and native shell command-dispatch text plus JSON trace sidecars with deterministic widget-tree artifacts",
			Stability: "current metadata plus wasm32-web command dispatch covered by post-v0.4 Web UI runtime smoke and native shell command dispatch/widget-tree traces for lowered scalar state operations; style and accessibility metadata are preview attributes only, while executable Linux-x64 native runtime evidence is tracked by ui.native-runtime",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_v0.4.0.md", "docs/spec/v1_feature_status.md", "docs/user/wasm_ui_guide.md"},
		},
		{
			ID:        "ui.toolkit-core",
			Name:      "UI Toolkit Core contract runtime",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "production platform-independent UI Toolkit Core contract for tetra.ui.toolkit.v1 with widget model, layout model, style model, accessibility model, event model, state binding model, widget tree construction, layout measurement and placement, event dispatch, state binding/update, focus traversal, timer/async command/redraw/error recovery evidence, deterministic compiler .ui.toolkit.json emission, and validator-gated runtime trace artifacts",
			Stability: "current toolkit core only; validators reject metadata-only, preview-only, runtime-less, native-shell sidecar-only, web-only, docs-only, build-only, fake/mock/placeholder evidence, and this does not claim GTK/Qt/OS platform backend production, Windows/macOS GUI production, or full cross-platform UI",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_toolkit_core.md", "docs/spec/ui_v0.4.0.md"},
		},
		{
			ID:        "wasm.runtime-execution",
			Name:      "WASM runtime execution",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "production WASI runner execution through wasmtime or the Node WASI fallback plus browser-backed wasm32-web execution through discovered Chromium-compatible runners",
			Stability: "current runner-backed WASM runtime support with explicit missing-runner diagnostics; browser UI command dispatch evidence remains separated from Linux-x64 native UI runtime evidence",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/backend/wasm_backend_plan.md", "docs/user/wasm_ui_guide.md"},
		},
		{
			ID:        "language.full-v1-guarantees",
			Name:      "Full v1.0 language guarantees",
			Status:    FeatureStatusPlanned,
			Scope:     "complete v1.0 release contract after mandatory release-gate evidence",
			Stability: "future label while repository remains on the v0.4.0 profile",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/v1_scope.md"},
		},
		{
			ID:        "language.full-first-class-callables",
			Name:      "Full first-class callable/function-value semantics",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "production first-class callable/function-value semantics for safe by-value captures: the bounded fnptr fast path remains for up to eight environment slots, and larger immutable Int/Bool/String/simple-aggregate captures use a fixed 4-slot callable handle for local storage, mutable local reassignment, returns, same-module global snapshots, struct fields, enum payloads, synchronous callback arguments, cross-module returned values, aliases, generated .t4i function-typed parameter local-alias return metadata, and generated .t4i metadata",
			Stability: "current v0.4.0 safe-capture model with explicit escape classification and stable JSON diagnostics for mutable by-reference captures including callable mutable-capture global-escape and callable mutable-capture heap-escape, callable pointer/resource capture escape, function-typed storage/return unsupported capture rejection, captured callable/function-typed parameter global-storage escape, unsupported function-value escape outside the fnptr ABI, unsupported function-value call, capturing closure raw-ptr escape, captured closure explicit type-arg rejection, function-typed explicit type-arg rejection, generic closure capture and generic callback-closure capture rejection, generic closure pointer/direct-call rejection, thread-boundary callable escape, imported mutable function-typed global boundary, imported mutable global-data ABI gaps, and unsupported dynamic/generic callable movement",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/v1_feature_status.md", "docs/release/v0_4_0_callable_evidence_map.md"},
		},
		{
			ID:        "eco.distributed-network",
			Name:      "Distributed EcoNet and production publishing",
			Status:    FeatureStatusPostV1,
			Scope:     "distributed EcoNet, production TetraHub publishing, global trust scoring, proof-carrying capsules",
			Stability: "deferred post-v1 unless explicitly promoted",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/release/post_v1_promotion_checklist.md"},
		},
		{
			ID:        "actors.distributed-runtime",
			Name:      "Distributed actor runtime for Linux x64",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "production Linux-x64 distributed actor runtime path with actornet loopback TCP broker, distributed node identity, remote actor handles, network mailbox send/receive for i32, tagged, and typed frames, missing-node failure/status propagation, and compatibility with existing task cancel/join handles",
			Stability: "current Linux-x64 runtime/lowering slice with executable tetra.actors.distributed-runtime.v1 smoke evidence and strict validator rejection for transport-only or fake reports; non-Linux-x64 targets, multi-threaded scheduling, and broader structured-concurrency guarantees remain outside this claim",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/actors.md", "docs/user/async_actors_guide.md"},
		},
		{
			ID:        "ui.native-runtime",
			Name:      "Linux-x64 native UI runtime",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "production Linux-x64 native UI runtime path that loads the checked tetra.ui.v0.4.0/native-shell widget tree, creates native runtime widget instances with IDs, hierarchy, bounds, text/value, enabled, and visible state, dispatches click/activate events to lowered command operations, propagates state and widget updates, records lifecycle close, and reports negative invalid widget, malformed metadata, unsupported event, and command failure cases",
			Stability: "current Linux-x64 deterministic native runtime slice with executable tetra.ui.native-runtime.v1 smoke evidence and strict validator rejection for metadata-only, web-only, native-shell sidecar-only, fake, mock, or placeholder evidence; macOS/Windows, GTK/Qt/OS widget backend claims, platform accessibility integration, and broad input/change/focus behavior remain outside this claim until host-native reports exist",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_v0.4.0.md", "docs/user/wasm_ui_guide.md"},
		},
	}
	out := make([]FeatureInfo, len(features))
	copy(out, features)
	for i := range out {
		out[i].Docs = append([]string(nil), features[i].Docs...)
	}
	return out
}
