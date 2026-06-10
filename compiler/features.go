package compiler

// FeatureStatus is the release-truth lifecycle label for a public Tetra feature.
type FeatureStatus string

const (
	FeatureStatusCurrent             FeatureStatus = "current"
	FeatureStatusExperimental        FeatureStatus = "experimental"
	FeatureStatusReleaseCandidate    FeatureStatus = "release_candidate"
	FeatureStatusUnsupported         FeatureStatus = "unsupported"
	FeatureStatusLegacyCompatibility FeatureStatus = "legacy_compatibility"
	FeatureStatusPlanned             FeatureStatus = "planned"
	FeatureStatusPostV1              FeatureStatus = "post-v1"
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
			Scope:     "linux-x64 build/run plus macos-x64, windows-x64, linux-x86, and linux-x32 build-only release coverage with pointer, rawptr, nullable_ptr, ref, c_int, c_uint, and the complete ILP32 native/libc scalar FFI object evidence set, x86/x32 allocator success/failure and island/free executable ABI smoke evidence, current x86/x32 core.net runtime ABI evidence, and explicit x86/x32 no-host-fallback, bounded two-spawn x86/x32 self-host scheduler evidence, function-pointer FFI diagnostics, remaining source target-layout scalar diagnostics, Surface, distributed actors, and actor-fanout diagnostics",
			Stability: "supported target metadata is validated by release checks; linux-x64 keeps pointer plus c_int/c_uint @export object regression smokes, linux-x86 and linux-x32 now build canonical ptr/rawptr/nullable_ptr/ref plus c_int/c_uint plus the complete ILP32 native/libc scalar @export object smoke set and target-specific allocator success/failure plus island/free executable ABI smokes, and both build-only targets remain unpromoted until their remaining FFI/runtime/stdlib runner gates pass",
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
			Scope:     "generic functions with inferred value arguments are statically monomorphized across modules; tiny generic identity/wrapper calls may disappear through the internal small-pure inliner after monomorphization; no runtime generic values or dynamic dispatch",
			Stability: "supported static MVP; explicit type arguments, generic structs, higher-ranked generics, full protocol-bound generic dispatch, and broad specialization optimization remain future/post-v1",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"},
		},
		{
			ID:        "language.layout-abi-policy",
			Name:      "Struct layout and ABI representation policy",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "default structs carry Tetra representation metadata and do not promise C layout; repr(C) struct declarations parse and check into ABI-locked metadata; public ABI/exported FFI aggregate boundaries require explicit repr(C)",
			Stability: "current P21.0 default layout freedom v1 metadata/report contract with .layout.json schema_version 2, policy p21.0_default_layout_freedom_v1, decision rows compiler_owned_default, abi_locked_repr_c, exported_ffi_explicit_repr_c, and validator rejection for fake layout freedoms; field_reordering, padding_removal, hot_cold_splitting, scalar_replacement, and aos_to_soa freedoms are explicitly unavailable for repr(C), while the compiler-owned default layout freedom is report evidence only and no field reordering, packing, hot/cold splitting, scalar replacement, AoS-to-SoA transform, performance change, runtime behavior change, or public ABI layout without repr(C) is claimed",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/design/truthful_intent_architecture.md", "docs/design/explainable_one_build.md", "docs/audits/default-layout-freedom-v1.md"},
		},
		{
			ID:        "compiler.abi-verification",
			Name:      "ABI verification v1",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "P21.1 ABI verification v1 report schema tetra.abi.verification.v1 with scope p21.1_abi_verification covers linux-x64 SysV, linux-x86 i386 SysV, linux-x32 x32 SysV, macos-x64 SysV, windows-x64 Win64, wasm32-wasi, and wasm32-web target rows; task coverage includes abi_test_corpus, struct_enum_slice_string_return_validation, call_boundary_validation, and ffi_repr_c_tests; native rows reuse x86/x32/x64 classifier, aggregate, object, and FFI repr(C) diagnostics; wasm rows validate compiler-owned i32 slot ABI metadata and backend IRCall arg/return slot matching",
			Stability: "current evidence/report contract only; no runtime execution claim for build-only or wasm targets, no C ABI claim for default structs, no native C aggregate ABI claim for wasm targets, no performance claim, and no safe-program semantics change",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/design/truthful_intent_architecture.md", "docs/design/explainable_one_build.md", "docs/audits/abi-verification-v1.md"},
		},
		{
			ID:        "compiler.feature-surface-audit",
			Name:      "Full feature surface audit v1",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "P22.0 full feature surface audit report schema tetra.language.feature_surface_audit.v1 with scope p22.0_full_feature_surface_audit covers first-class callables, closures, protocols/trait objects, runtime generics, advanced enums/pattern matching, async typed errors, structured concurrency, modules/packages, macros/metaprogramming, UI/surface, and Eco/capsules; rows copy current FeatureRegistry statuses and preserve keep-current-bounded, keep-static-only, keep-post-v1, unsupported, or experimental-gate decisions without promoting a feature unless same-branch evidence exists",
			Stability: "current evidence/report contract only; no full v1 language guarantee, runtime generic values, trait objects, runtime protocol values, macro/metaprogramming system, full structured concurrency, cross-platform production UI runtime, distributed EcoNet, proof-carrying capsules, performance claim, runtime behavior change, or safe-program semantics change is claimed",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/design/truthful_intent_architecture.md", "docs/design/explainable_one_build.md", "docs/audits/full-feature-surface-audit-v1.md"},
		},
		{
			ID:        "compiler.ram-contracts",
			Name:      "RAM Contract Compiler reports",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "RAM Contract Compiler report evidence for linux-x64 build outputs with tetra.ram-contract-report.v1, tetra.memory-grade-report.v1, tetra.proof-store-summary.v1, tetra.validation-pipeline-coverage.v1, heap-blockers.json, copy-blockers.json, ram-contract-fuzz-oracle.json, --emit-ram-contract-report, --fail-if-heap, --fail-if-copy, --fail-if-unbounded, --memory-budget, --ram-contract, TETRA4100 diagnostics, validate-ram-contract-report, validate-memory-grade-report, validate-proof-store-summary, validate-validation-pipeline-coverage, validate-heap-blockers, validate-copy-blockers, validate-ram-contract-fuzz-oracle, validate-ram-contract-release, and scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh evidence",
			Stability: "current report/gate contract only; no zero heap for all programs claim, no zero-copy for all programs claim, no full formal proof claim, no all-target RAM parity claim, no production object memory claim, no production persistent memory claim, no runtime behavior change, no performance claim, and no safe-program semantics change is claimed",
			Docs: []string{
				"docs/design/ram_contract_compiler.md",
				"docs/spec/ram_contract_report_schema.md",
				"docs/user/ram_contracts.md",
				"docs/audits/ram-contract-compiler-readiness.md",
				"docs/audits/ram-contract-compiler-handoff.md",
			},
		},
		{
			ID:        "compiler.first-class-callables-v1",
			Name:      "First-class callables v1 evidence",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "P22.1 first-class callables v1 report schema tetra.language.first_class_callables.v1 with scope p22.1_first_class_callables_v1 covers the bounded fnptr fast path, fat callable handle, capture safety classifier, mutable capture escape diagnostics, resource/thread escape diagnostics, fixed ABI width, cross-module interface metadata, and storage/callback paths; witnesses parse, check, and lower a one-capture 9-slot fnptr value without heap environment allocation plus a nine-capture fixed 4-slot handle value with IRAllocBytes, nine IRMemWritePtrOffset writes, nine IRMemReadPtrOffset reads, and call ArgSlots 10 RetSlots 1; generated .t4i metadata preserves ReturnFunctionHandleValue, heap escape kind, capture count, target identity, and ReturnSlots = 4",
			Stability: "current evidence/report contract only for the existing safe by-value callable model; no variable-width callable ABI, exploding return slots, mutable by-reference capture support, pointer/resource capture support, thread-boundary callable transfer, runtime generic callable polymorphism, dynamic callable dispatch, unsafe lifetime relaxation, performance claim, runtime behavior change, or safe-program semantics change is claimed",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/design/truthful_intent_architecture.md", "docs/design/explainable_one_build.md", "docs/audits/first-class-callables-v1.md", "docs/release/v0_4_0_callable_evidence_map.md"},
		},
		{
			ID:        "compiler.protocol-trait-object-decision",
			Name:      "Protocol / trait object decision v1",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "P22.2 protocol / trait object decision report schema tetra.language.protocol_trait_object_decision.v1 with scope p22.2_protocol_trait_object_decision records decision keep_static_conformance_only; rows cover static conformance fast path, static protocol-bound generics, runtime existential decision, explicit dynamic-dispatch gate, specialization static abstraction, witness-table boundary, trait-object boundary, and registry/docs alignment; witnesses parse, check, and lower a static protocol impl direct Vec2.draw IRCall, a protocol-bound generic concrete id__T_Vec2 direct call, runtime protocol value rejection with unknown type 'Drawable', generic-bound requirement-call rejection, and P17/P21 known-direct specialization evidence",
			Stability: "current evidence/report decision only; runtime protocol values, trait objects, witness tables, dynamic dispatch, conformance-table lookup, runtime existential ABI, broad protocol specialization, performance, runtime behavior change, and safe-program semantics change are not promoted or claimed",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/design/truthful_intent_architecture.md", "docs/design/explainable_one_build.md", "docs/audits/protocol-trait-object-decision-v1.md", "docs/audits/inlining-specialization-v1.md", "docs/audits/specialization-machine-code-v1.md"},
		},
		{
			ID:        "compiler.verified-track",
			Name:      "Long-term verified track evidence",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "internal P11/P16/P17/P18/P19 verified track: differential scalar-i32 stable IR interpreter compares source interpreter, stack backend, register backend, and optimized backend results; backend differential matrix v1 compares supported source, Stack IR, optimized Stack IR, SSA, Machine IR, and native execution lanes for scalar, slice-sum, branch/loop, and call-loop rows; optimizer pass contract v1 requires registered pass names, input/output verifier evidence, proof preservation or invalidation rules, translation validation hooks, stable report rows, negative-test markers, and validation metadata with sha256 before/after hashes, function set, proof facts, semantic checks, and differential samples; optimizer core coverage v1 records a bounded evidence-backed P17.1 closure with narrow safe const-denominator div_i32/mod_i32 constant folding plus same-local comparison algebraic simplification, narrow SCCP constant-condition, known-local and stored safe unary neg_i32 plus safe constant-expression facts including safe const-denominator div_i32/mod_i32, constant unary neg_i32 and binary-expression branch folding including safe const-denominator div_i32/mod_i32 with unary min-int and denominator 0 and -1 rejected, immediate and forward-terminated single-predecessor label propagation plus folded zero-branch target propagation for labels with one incoming edge and no fallthrough predecessor, folded nonzero-branch fallthrough propagation through immediate labels with no explicit incoming branch/jump edges, dynamic load-local zero-target and nonzero-fallthrough path facts, dynamic zero-comparison eq/ne zero/nonzero path facts, fallthrough-predecessor rejection, explicit-incoming fallthrough-label rejection, and fallthrough pruning, narrow Stack IR adjacent and stack-neutral separated single-assignment mem2reg temp promotion including bounded comparison-expression, safe const unary neg_i32, safe known-local unary neg_i32, safe const add_i32/sub_i32/mul_i32 arithmetic, safe known-local add_i32/sub_i32/mul_i32 arithmetic, safe const-denominator div_i32/mod_i32 producer temps, and safe known-local div_i32/mod_i32 producer temps with unary min-int, arithmetic overflow, source-local mutation, and denominator 0 and -1 rejected, bounded DCE for simple dead local stores, non-trapping comparison-expression stores, safe known-local unary neg_i32 stores, safe known-local add_i32/sub_i32/mul_i32 stores, safe const-denominator div_i32/mod_i32 stores, and safe known-local div_i32/mod_i32 stores with unary min-int, arithmetic overflow, and denominator 0 and -1 rejected, a narrow exact/commutative/mirrored-comparison local-load, local-load/constant, unary local neg_i32, safe known-local unary neg_i32 value, safe known-local add_i32/sub_i32/mul_i32 value, safe known-local cmp_*_i32 value, safe known-local div_i32/mod_i32 value, and safe const-denominator div_i32/mod_i32 CSE/GVN slice in basic-scalar including commutative add/mul/eq/ne and mirrored lt/gt/le/ge operand canonicalization, narrow proof-tagged LICM pure invariant comparison, add/sub/mul arithmetic, known-local add_i32/sub_i32/mul_i32 left-or-right operand hoisting, known-local cmp_*_i32 left-or-right operand hoisting, safe const-denominator div_i32/mod_i32 hoisting, and safe known-local div_i32/mod_i32 denominator hoisting, and bounded hot-loop shape evidence for scalar sum, scalar constant-stride sum, scalar sum-of-squares, scalar product reduction bounded to product *= index + 1, scalar branchy max reduction, scalar affine sum with compile-time scale and bias 1..127, scalar countdown, proof-tagged slice sum, proof-tagged slice constant-stride sum, and call-loop machine IR rows; inlining specialization coverage v1 records P17.2 target rows with narrow monomorphized generic identity/wrapper, small-pure inline-small-pure, payload enum known-case match and proven-some optional match sccp-constant-branch evidence, statically checked protocol/conformance direct-call inline-small-pure evidence, statically resolved extension-call inline-small-pure evidence, inlined/not_inlined report reasons, the same 8-instruction body cap, translation validation, constant_stack_store tag tracking, known direct Stack IR function symbol boundaries, and explicit non-claims for protocol-bound requirement calls, witness tables, trait objects, runtime protocol values, dynamic dispatch, and conformance-table lookup; vectorization coverage v1 records P17.3 initial target rows with proof-tagged sum []i32 candidate recognition, range-proof evidence, noalias-not-required read-only reduction evidence, safe unaligned i32x4 vector backend lowering through vector-i32x4-slice-sum-plan, linux-x64 native SIMD lowering for proof-tagged step=1 sum []i32, scalar tail handling, scalar-i32-slice-sum fallback, translation/differential validation against stack fallback, proof-tagged copy []u8 vector backend lowering through vector-u8x16-copy-plan, noalias required source/dest disjoint owned-copy-result evidence, safe unaligned u8x16 load/store, scalar-u8-copy fallback, linux-x64 native SIMD lowering for proof-tagged copy []u8, copy []u8 translation/differential validation against stack fallback, proof-tagged simple map over []i32 guarded vector backend lowering through vector-i32x4-map-add-const-plan, single mutable slice in-place noalias-not-required evidence, safe unaligned i32x4 map load/store, scalar-i32-map fallback, linux-x64 native SIMD lowering for proof-tagged in-place add-constant-1 map []i32, map []i32 translation/differential validation against stack fallback, proof-tagged memset/memcpy helper evidence through vector-u8x16-memset-zero-plan, single mutable slice zero-fill noalias-not-required evidence, safe unaligned u8x16 zero-store, scalar-u8-memset-zero fallback, linux-x64 native SIMD zero-fill lowering for proof-tagged memset_zero_u8, memset_zero_u8 translation/differential validation against stack fallback, memcpy helper via copy []u8 evidence, and explicit no broad SIMD auto-vectorization, checked/no-proof copy, overlapping copy, checked/no-proof map, broader map-shape vectorization, arbitrary non-zero memset, overlapping memcpy, checked/no-proof helper, libc/runtime helper lowering, or performance claim; PGO/LTO/target-cpu evidence v1 records tetra.optimizer.profile.v1 canonical JSON profile collection format with duplicate and negative counter rejection, internal Options.ProfileInput optimizer profile input API, profile_input_policy pass-contract metadata, profile digest validation metadata, translation validation for profile-input foundation runs, profile-guided rewrite policy rejection, profile parsing is evidence-only, target-cpu feature detection foundation with portable baseline target-feature model, guarded codegen contract, no target-specific rewrite, LTO/incremental module summary foundation with tetra.incremental.module_summary.v1 dependency hash contract and non-consumer boundary, no LTO optimizer or incremental speedup claim, final safe-semantics closure validator rejects fake semantic-changing coverage, profile-guided rewrite policy, target-specific optimization evidence, and LTO/codegen/linker consumers, and no PGO, LTO, target-cpu, or profile flag changes safe-program semantics; actor runtime production-boundary audit v1 records tetra.runtime.actor.production_boundary.v1 rows for current actor runtime limits, scheduler prototype features, production runtime acceptance, and full claim blockers, with fake full production actor runtime claim rejection and explicit non-claims for production multi-threaded actor scheduling, non-Linux-x64 distributed actor runtime targets, message-pool exhaustion/reclamation, full cancellation and structured concurrency, full race-safety proof, and production broker deployment evidence; async I/O reactor v1 records tetra.runtime.io_reactor.v1 rows for Linux epoll v1, io_uring future boundary, kqueue macOS boundary, IOCP Windows boundary, WASI/web adapter boundary, nonblocking accept/read/write, readiness polling, task wakeups from I/O readiness, timer integration, cancellation, backpressure, reactor report rows, HTTP smoke, DB smoke, stress evidence, fake full production web-stack rejection, fake cross-platform reactor parity rejection, fake io_uring rejection, fake runtime-behavior-change rejection, and clear production boundary per platform; region-aware stdlib v1 records tetra.stdlib.region_aware.v1 rows for byte-oriented StringBuilder, VecBytes, fixed-capacity HashMapBytes, ByteBuffer, RingBuffer, borrowed JSON/HTTP views, PostgreSQL protocol helper reports, copy-only-when-needed reports, hidden-heap rejection, and fake production web/db/result claim rejection; no full production web stack, cross-platform reactor parity, io_uring support, runtime behavior change, official TechEmpower result, production HTTP/PostgreSQL stack promotion, broad generic collection API, or public stdlib mode is claimed; self-hosting gate requires register backend, optimizer, allocator, and stdlib evidence before a self-hosting claim; formal core spec covers values, provenance, borrow/copy, bounds proofs, allocation intent, raw pointer bounds metadata, and check-elimination validity",
			Stability: "current internal evidence only; not a public backend selector, source interpreter mode, release optimization mode, full self-hosting claim, or full formal proof of the language",
			Docs: []string{
				"docs/spec/current_supported_surface.md",
				"docs/design/explainable_one_build.md",
				"docs/design/formal_core_semantics.md",
				"docs/design/truthful_intent_architecture.md",
				"docs/audits/master-plan-final-20260602.md",
				"docs/audits/master-plan-final-20260602-artifact-map.md",
				"docs/audits/truthful-performance-core-baseline.md",
				"docs/audits/safe-borrow-returns-v1.md",
				"docs/audits/noalias-mutable-borrow-v1.md",
				"docs/audits/lifetime-module-boundaries-v1.md",
				"docs/audits/implicit-region-lowering-readiness-v1.md",
				"docs/audits/request-task-region-v1.md",
				"docs/audits/thread-per-core-allocator-v1.md",
				"docs/audits/raw-pointer-bounds-metadata-v1.md",
				"docs/audits/backend-coverage-audit-v1.md",
				"docs/audits/value-ssa-ir-v1.md",
				"docs/audits/register-backend-coverage-expansion-v1.md",
				"docs/audits/backend-differential-validation-v1.md",
				"docs/audits/optimizer-pass-contract-v1.md",
				"docs/audits/optimizer-core-coverage-v1.md",
				"docs/audits/inlining-specialization-v1.md",
				"docs/audits/vectorization-v1.md",
				"docs/audits/pgo-lto-target-cpu-v1.md",
				"docs/audits/actor-runtime-production-boundary-v1.md",
				"docs/audits/typed-actor-ownership-transfer-v1.md",
				"docs/audits/per-core-scheduler-v1.md",
				"docs/audits/async-io-reactor-v1.md",
				"docs/audits/region-aware-stdlib-v1.md",
			},
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
			Scope:     "stable uses effect names and groups with transitive call propagation across resolved direct, generic, protocol, and supported callable paths; missing uses declarations are diagnostics; PLIR exposes checker-enforced optimizer facts for pure/no-alloc/no-mem-write/no-actor-send/no-unknown-escape cases",
			Stability: "supported static MVP; no effect inference or proof-level effect system guarantee is claimed, and optimizer facts are emitted only from checked declared effects",
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
			Scope:     "production local safety model for ownership/lifetime/borrow/consume/inout checks, resource finalization, callable escape diagnostics, effects/capabilities/privacy/consent/budget policy, unsafe boundaries, actor/task transfer safety, pointer/MMIO/memory capability gates, Memory Production Core v1 report evidence through compiler-owned facts rather than report-reconstructed truth, a memory cost model with zero_cost_proven, dynamic_check_required, instrumentation_only, unsupported_rejected, and conservative_fallback report classes, a memory fuzz oracle with Tier 1 short CI smoke, Tier 2 nightly fuzz, Tier 3 release-blocking focused memory fuzz, explicit oracle categories, MEM-FUZZ-012 deterministic v0-v11 release evidence rows, required crash/miscompile repro artifacts, release-blocking unsafe/bounds/storage/report classifications, memory production final audit with artifact map and explicit nonclaims, validate-island-proof independent-ish verifier evidence, --islands-debug sanitizer smoke, island-proof-fuzz-summary deterministic mutation evidence, leak/resource finalization evidence, and an integrated Memory/Islands/Surface release gate with memory-islands-surface-production-manifest.json and artifact-hashes.json, and no Memory 100% claim or unsupported unsafe pointer safety claim",
			Stability: "release-gated current profile with explicit diagnostics for unsupported distributed, cryptographic, formal-proof, runtime-wide guarantees, arbitrary unsafe external pointer safety, full target parity, all-target Surface support, clean release-candidate checkout claims, and no production object memory or production persistent memory claim",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/effects_capabilities_privacy_v1.md", "docs/spec/unsafe.md", "docs/spec/memory_report_schema_v1.md", "docs/spec/islands.md", "docs/design/memory_production_core_v1.md", "docs/design/memory_cost_model.md", "docs/audits/memory-fuzz-oracle-v1.md", "docs/testing/fuzz_property_stress.md", "docs/audits/memory-production-core-v1-baseline.md", "docs/audits/memory-production-core-v1-gap-map.md", "docs/audits/memory-production-core-v1-supported-surface.md", "docs/audits/memory-target-capability-matrix.md", "docs/audits/memory-production-core-v1-final.md", "docs/audits/memory-production-core-v1-artifact-map.md", "docs/audits/memory-production-core-v1-nonclaims.md", "docs/release/memory_islands_surface_scope.md", "docs/audits/memory-ideal-vslice-v0-baseline.md", "docs/audits/memory-ideal-vslice-v0-correlation.md", "docs/audits/memory-ideal-vslice-v0-final.md", "docs/audits/memory-ideal-vslice-v1-correlation.md", "docs/audits/memory-ideal-vslice-v1-final.md", "docs/audits/memory-ideal-vslice-v2-correlation.md", "docs/audits/memory-ideal-vslice-v2-final.md", "docs/audits/memory-ideal-vslice-v3-correlation.md", "docs/audits/memory-ideal-vslice-v3-final.md", "docs/audits/memory-ideal-vslice-v4-correlation.md", "docs/audits/memory-ideal-vslice-v4-final.md", "docs/audits/memory-ideal-vslice-v5-correlation.md", "docs/audits/memory-ideal-vslice-v5-final.md", "docs/audits/memory-ideal-vslice-v6-bounds-correlation.md", "docs/audits/memory-ideal-vslice-v6-bounds-final.md", "docs/audits/memory-ideal-vslice-v7-ffi-correlation.md", "docs/audits/memory-ideal-vslice-v7-ffi-final.md", "docs/audits/memory-ideal-vslice-v8-report-correlation.md", "docs/audits/memory-ideal-vslice-v8-report-final.md", "docs/audits/memory-ideal-vslice-v9-storage-correlation.md", "docs/audits/memory-ideal-vslice-v9-storage-final.md", "docs/audits/memory-ideal-vslice-v10-async-cancel-correlation.md", "docs/audits/memory-ideal-vslice-v10-async-cancel-final.md", "docs/audits/memory-ideal-vslice-v11-dynproto-correlation.md", "docs/audits/memory-ideal-vslice-v11-dynproto-final.md"},
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
			Scope:     "[]u8/[]u16/[]i32/[]bool helpers including make_* and island_make_* allocation-length contracts, island compile-compatible fallback paths, checked slice window/prefix/suffix safe view constructors, proof-tagged for-loop and supported while-loop bounds-check removal through PLIR CFG/dominance/range facts, explicit borrow/copy/copy_into methods, and checked String byte window/prefix/suffix/borrow/copy/copy_into methods with provenance-aware PLIR facts, allocation/proof/bounds report evidence, and actor-boundary copy diagnostics",
			Stability: "supported MVP with documented layout/runtime constraints",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/stdlib.md"},
		},
		{
			ID:        "language.safe-view-lifetime-contracts-v1",
			Name:      "Safe View Lifetime Contracts v1",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "borrowed return signatures for supported slice/String byte views, cross-module borrowed return preservation, single-source borrowed return validation, recursive hidden-borrow escape checks for structs/enums/optionals/generic wrappers, actor and typed-task copy-required boundaries, and PLIR/proof/alloc evidence for borrow/copy/borrowed-return facts",
			Stability: "current conservative lifetime contract for safe view surfaces; named lifetimes, generic lifetime parameters, arbitrary borrowed aggregate returns, full Unicode String lifetime semantics, Rust-like borrow checking, and production FFI lifetime contracts remain outside this claim",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/design/truthful_safe_values.md", "docs/design/provenance_lifetime_ir.md", "docs/design/truthful_intent_architecture.md", "docs/user/examples_index.md"},
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
			Scope:     "release-covered lib.core helper modules with a capability-gated linux-x64 filesystem exists slice plus filesystem+scheduler composition and scheduler-restriction regression smokes, x86 and x32 no-runtime stdout/string-literal executable smokes, x86 and x32 stderr fd runtime smokes through core.net_write(2), x86 and x32 allocator success/failure executable smokes for core.alloc_bytes plus raw store/load and checked invalid-size/mmap-error exit lowering, x86 and x32 island/free executable smokes for scoped island allocation/free and debug free guard lowering, x86 and x32 filesystem+scheduler self-host composition smokes, x86 and x32 bounded two-spawn actors/task/task-group self-host smokes, x86 and x32 typed-task self-host smokes, x86 and x32 staged typed-task self-host smokes, x86 and x32 typed task-group self-host smokes, and pure fs_exists linux-x86/linux-x32 smokes, executable Linux TCP socket client/server I/O helpers with recv/send, SO_REUSEPORT, TCP_NODELAY, nonblocking accept convenience, and epoll add/mod/delete plus wait-one readiness flag capture and predicates, stable crypto interface helpers, stable networking endpoint policy helpers, executable HTTP/1.1 String and byte-buffer request-line routing, request-head framing, and response byte-buffer helpers, executable JSON byte-buffer response helpers, and internal P7/P19 runtime evidence for region-aware collection/buffer storage planning, P19 byte-oriented StringBuilder/VecBytes/HashMapBytes/RingBuffer helpers, borrowed JSON parsing, borrowed HTTP request-head parsing, and PostgreSQL borrowed/binary row helpers",
			Stability: "current import paths and smoke coverage; filesystem exists is host-backed on linux-x64 including filesystem+scheduler composition and scheduler-restriction regression smokes, x86 and x32 no-runtime stdout/string-literal executables plus core.net_write fd=2 stderr runtime executables, core.alloc_bytes allocator success/failure executable smokes, and scoped island/free executable smokes are covered by ABI smokes, composable with the x86 and x32 self-host scheduler slices, x86 and x32 two-spawn actors/task/task-group flows are covered by self-host runtime smokes, x86 and x32 typed-task handles are covered by self-host runtime smokes with staged typed-task coverage, x86 and x32 typed task-group composition are covered by self-host runtime smokes, and pure fs_exists linux-x86/linux-x32 smokes remain covered; full x86/x32 allocator/free/panic parity remains unpromoted. net socket open/bind/connect/listen/accept/read/recv/write/send/nonblocking/close plus SO_REUSEPORT, TCP_NODELAY, SOCK_NONBLOCK/SOCK_CLOEXEC accept helpers, and epoll create/add-read/add-read-write/mod-read/mod-read-write/delete/wait-one/wait-one-into helpers with EPOLLIN/EPOLLOUT/EPOLLERR/EPOLLHUP predicates are host-backed on linux-x64, crypto exposes deterministic interface helpers, networking exposes deterministic endpoint policy helpers, HTTP helpers classify TechEmpower request lines from String text or caller-owned byte buffers, locate CRLFCRLF request-head boundaries for pipelined buffers, and write compact response payloads into caller-owned buffers, JSON helpers write compact response bodies into caller-owned buffers, and P7/P19 internal runtime helpers provide checked storage/provenance/copy evidence without promoting broad generic collection APIs, production web/db stacks, or official TechEmpower claims",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/stdlib.md", "docs/user/standard_library_guide.md", "docs/audits/region-aware-stdlib-v1.md"},
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
			Scope:     "legacy metadata compatibility surface preserving the production UI metadata contract for checked view/state declarations, deterministic tetra.ui.v0.4.0 JSON, browser-backed web command-dispatch runtime artifacts, style metadata preview attributes, accessibility metadata preview attributes, and native shell command-dispatch text plus JSON trace sidecars with deterministic widget-tree artifacts",
			Stability: "current metadata plus wasm32-web command dispatch covered by post-v0.4 Web UI runtime smoke and native shell command dispatch/widget-tree traces for lowered scalar state operations; it is not the new Tetra Surface runtime, not the pure-Tetra component model, and not a basis for new Surface host claims; style and accessibility metadata are preview attributes only, while executable Linux-x64 native runtime evidence is tracked by ui.native-runtime",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_v1.md", "docs/spec/ui_v0.4.0.md", "docs/spec/v1_feature_status.md", "docs/user/wasm_ui_guide.md"},
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
			ID:        "ui.surface-core",
			Name:      "Tetra Surface core",
			Status:    FeatureStatusCurrent,
			Since:     "surface-v1",
			Scope:     "surface-v1-linux-web current release scope: pure-Tetra UI, tiny Surface Host ABI, software RGBA framebuffer presentation, owned/copy-safe event and text buffers, and release evidence for headless, linux-x64 real-window, and wasm32-web browser-canvas targets",
			Stability: "current only for the bounded Surface v1 linux/web release scope; macOS, Windows, wasm32-wasi UI, GPU rendering, platform widgets, DOM/user-JS app logic, dynamic trait-object widgets, witness-table component dispatch, and rich text editor claims remain unsupported or future work",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/release/surface_v1_release_contract.md", "docs/release/surface_v1_release_notes.md", "docs/release/surface_v1_release_audit.md"},
		},
		{
			ID:        "ui.surface-block-system",
			Name:      "Tetra Surface Block System",
			Status:    FeatureStatusExperimental,
			Scope:     "Block-first Surface architecture implementation track with `lib.core.block` data model support for Block as the core Surface primitive for layout, paint, text, assets, input/events, states, motion, and accessibility; existing Button/Card/TextField-like helpers are recipes/compatibility over Block rather than core widget primitives; scoped `tetra.surface.block-system.gate.v1` reports include `block_system.memory_budget` evidence under reports/surface-block/p18-budget",
			Stability: "experimental and not current Surface v1 production support, with same-commit target evidence for headless, linux-x64 real-window, and wasm32-web browser-canvas Block-system reports, validators, artifact hashes, and release-gate integration; not production support and no production Block claim, Electron, React, DOM, CSS runtime, user JavaScript, Chromium, platform-native widget, GPU renderer, or cross-platform desktop replacement claim is implied",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/user/examples_index.md", "docs/release/surface_v1_release_contract.md", "docs/release/surface_v1_release_notes.md", "docs/release/surface_v1_release_audit.md"},
		},
		{
			ID:        "ui.surface-morph-capsule",
			Name:      "Tetra Surface Morph Capsule",
			Status:    FeatureStatusExperimental,
			Scope:     "experimental Morph Capsule authoring layer over the Surface Block System; `lib.core.morph` defines scoped capsule tokens, materials, affordances, state lenses, motion presets, and recipe algebra that expands into Block evidence for `examples/surface_morph_command_palette.tetra`; `tetra.surface.morph.gate.v1` records deterministic headless same-commit Morph reports plus artifact hashes",
			Stability: "experimental evidence layer and not Surface v1 production support; Morph does not add core widget primitives, platform widgets, CSS cascade, DOM app logic, React/Electron runtime, GPU renderer, or cross-target desktop replacement support",
			Docs:      []string{"docs/spec/surface_morph.md", "docs/spec/current_supported_surface.md", "docs/user/surface_guide.md", "docs/user/examples_index.md", "docs/user/standard_library_guide.md", "docs/release/surface_v1_release_contract.md", "docs/release/surface_v1_release_notes.md"},
		},
		{
			ID:        "ui.surface-headless",
			Name:      "Headless Tetra Surface runtime",
			Status:    FeatureStatusCurrent,
			Since:     "surface-v1",
			Scope:     "release-test target for deterministic Surface runtime, text/input, toolkit, accessibility, artifact-hash, and validator evidence under surface-v1-linux-web",
			Stability: "current as a release evidence target, not as an end-user platform claim; reports are validated by strict Surface v1 release validators and artifact hashes",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/release/surface_v1_release_contract.md"},
		},
		{
			ID:        "ui.surface-linux-x64",
			Name:      "Linux-x64 Tetra Surface host",
			Status:    FeatureStatusCurrent,
			Since:     "surface-v1",
			Scope:     "current linux-x64-release-window-v1 Surface target using Wayland shm RGBA real-window evidence, native event pump, text input, clipboard, IME/composition trace, toolkit, and accessibility bridge evidence",
			Stability: "current only for the proven linux-x64 real-window release path; no GTK, Qt, platform widget, metadata sidecar playback, macOS, or Windows production claim is implied",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/release/surface_v1_release_contract.md"},
		},
		{
			ID:        "ui.surface-web-wasm",
			Name:      "WASM web Tetra Surface",
			Status:    FeatureStatusCurrent,
			Since:     "surface-v1",
			Scope:     "current wasm32-web-browser-canvas-release-v1 Surface target with compiler-owned browser boot, browser canvas RGBA presentation/readback, browser input, clipboard, composition, accessibility snapshot, and accessibility mirror evidence",
			Stability: "current only for pure-Tetra apps running through the tiny Surface Host ABI; DOM UI, React, user JavaScript app logic, metadata-only UI sidecars, Node-only evidence, and arbitrary browser widget claims are rejected by validators",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/release/surface_v1_release_contract.md"},
		},
		{
			ID:        "ui.surface-component-model",
			Name:      "Tetra Surface component model",
			Status:    FeatureStatusCurrent,
			Since:     "surface-v1",
			Scope:     "component-tree-api release subset where ordinary Tetra structs use `lib.core.component`, helper-owned parent/child links, stable ids, layout helpers, hit testing, focus routing, root-to-leaf dispatch paths, and no manual app-side tree bookkeeping",
			Stability: "current for the static release subset only; dynamic trait-object child lists, witness-table component dispatch, arbitrary reactive frameworks, and platform-native component trees remain future work",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/release/surface_v1_release_contract.md"},
		},
		{
			ID:        "ui.surface-toolkit-v1",
			Name:      "Tetra Surface toolkit v1",
			Status:    FeatureStatusCurrent,
			Since:     "surface-v1",
			Scope:     "production-widgets-v1 release subset in `lib.core.widgets`: Text, Label, StatusText, Button, TextBox, Checkbox, Row, Column, Panel, Stack, Scroll, and Spacer over the ComponentTree helper API",
			Stability: "current for the release widget subset with owned/copy-safe state and no magical widgets, platform widgets, DOM UI, user JS, or demo-local widget structs; broader widget libraries remain post-release work",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/user/examples_index.md", "docs/release/surface_v1_release_notes.md"},
		},
		{
			ID:        "ui.surface-text-input-v1",
			Name:      "Tetra Surface text/input v1",
			Status:    FeatureStatusCurrent,
			Since:     "surface-v1",
			Scope:     "production-text-input-v1 baseline covering UTF-8 byte storage, caret, selection, copy/paste clipboard transfer, clipboard read/write, IME/composition trace, focused TextBox routing, and host-boundary copy semantics",
			Stability: "current for the bounded Surface v1 text/input baseline; rich text, IDE-grade editing, arbitrary native text controls, and full Unicode editor semantics remain unsupported in this release",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/user/examples_index.md", "docs/release/surface_v1_release_notes.md"},
		},
		{
			ID:        "ui.surface-accessibility-v1",
			Name:      "Tetra Surface accessibility v1",
			Status:    FeatureStatusCurrent,
			Since:     "surface-v1",
			Scope:     "platform-bridge-v1 accessibility for supported targets: metadata tree exported through the Linux accessibility bridge/probe path and wasm32-web browser accessibility snapshot/mirror",
			Stability: "current for supported targets only; metadata-only reports, DOM/ARIA claims without compiler-owned mirror evidence, screen-reader claims, macOS/Windows accessibility, and full AT-SPI claims remain unsupported without separate proof",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/user/examples_index.md", "docs/release/surface_v1_release_notes.md"},
		},
		{
			ID:        "ui.surface-minimal-toolkit",
			Name:      "Tetra Surface minimal widget toolkit",
			Status:    FeatureStatusExperimental,
			Scope:     "historical minimal-widgets-v1 evidence absorbed by ui.surface-toolkit-v1; retained for backward report references and regression evidence",
			Stability: "absorbed by ui.surface-toolkit-v1 and not a public current release API; reports remain experimental historical evidence and must not claim production toolkit support",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/user/examples_index.md"},
		},
		{
			ID:        "ui.surface-toolkit-reuse-v1",
			Name:      "Tetra Surface toolkit reuse v1",
			Status:    FeatureStatusExperimental,
			Scope:     "historical toolkit-reuse-v1 multi-form evidence absorbed by ui.surface-toolkit-v1; retained for backward report references and regression evidence",
			Stability: "absorbed by ui.surface-toolkit-v1 and not a public current release API; reports remain experimental historical evidence and must not claim production toolkit support",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/user/examples_index.md"},
		},
		{
			ID:        "ui.surface-accessibility-metadata-tree-v1",
			Name:      "Tetra Surface accessibility metadata tree v1",
			Status:    FeatureStatusExperimental,
			Scope:     "internal layer under ui.surface-accessibility-v1; retained as historical metadata-tree evidence for roles, labels, values, states, bounds, relationships, focus order, reading order, actions, snapshots, and status updates",
			Stability: "internal layer under ui.surface-accessibility-v1 and not a public production accessibility claim by itself; metadata-only evidence must not claim platform accessibility, DOM/ARIA, screen-reader, or full AT-SPI support",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/user/surface_guide.md", "docs/user/examples_index.md"},
		},
		{
			ID:        "ui.surface-macos-x64",
			Name:      "macOS Surface host",
			Status:    FeatureStatusUnsupported,
			Scope:     "unsupported for Surface v1; no production target evidence exists for macOS real-window Surface",
			Stability: "no production target evidence in surface-v1-linux-web and no current macOS Surface support claim",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/release/surface_v1_release_contract.md"},
		},
		{
			ID:        "ui.surface-windows-x64",
			Name:      "Windows Surface host",
			Status:    FeatureStatusUnsupported,
			Scope:     "unsupported for Surface v1; no production target evidence exists for Windows real-window Surface",
			Stability: "no production target evidence in surface-v1-linux-web and no current Windows Surface support claim",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/release/surface_v1_release_contract.md"},
		},
		{
			ID:        "ui.surface-wasm32-wasi",
			Name:      "WASI Surface UI runtime",
			Status:    FeatureStatusUnsupported,
			Scope:     "unsupported for Surface v1; wasm32-wasi has no Surface UI runtime production target evidence",
			Stability: "no production target evidence in surface-v1-linux-web and no current wasm32-wasi Surface UI support claim",
			Docs:      []string{"docs/spec/surface_v1.md", "docs/release/surface_v1_release_contract.md"},
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
			Scope:     "production Linux-x64 distributed actor runtime path with actornet loopback TCP broker, distributed node identity, remote actor handles, network mailbox send/receive for i32, tagged, and typed frames, missing-node failure/status propagation, compatibility with existing task cancel/join handles, and scoped actor runtime foundation gate evidence through tetra.actor.production_foundation.v1",
			Stability: "current Linux-x64 runtime/lowering slice with executable tetra.actors.distributed-runtime.v1 smoke evidence, tetra.actor.production_foundation.v1 gate evidence from actor-runtime-foundation-linux-x64-gate.sh, and strict validator rejection for transport-only or fake reports; non-Linux-x64 targets, non-Linux distributed runtime, distributed zero-copy transfer, cluster membership, reconnect/retry production, formal race proof, multi-threaded scheduling, and broader structured-concurrency guarantees remain outside this claim",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/actors.md", "docs/user/async_actors_guide.md", "docs/design/actor_region_transfer.md", "docs/audits/actor-runtime-production-boundary-v1.md", "docs/checklists/actors_linux_smoke.md", "docs/checklists/actors_platform_smoke.md"},
		},
		{
			ID:        "ui.native-runtime",
			Name:      "Linux-x64 native UI runtime",
			Status:    FeatureStatusCurrent,
			Since:     "v0.4.0",
			Scope:     "production Linux-x64 native UI runtime path that loads the checked tetra.ui.v0.4.0/native-shell widget tree, creates native runtime widget instances with IDs, hierarchy, bounds, text/value, enabled, and visible state, dispatches click/activate events to lowered command operations, propagates state and widget updates, records lifecycle close, and reports negative invalid widget, malformed metadata, unsupported event, and command failure cases",
			Stability: "current Linux-x64 deterministic native runtime slice with executable tetra.ui.native-runtime.v1 smoke evidence and strict validator rejection for metadata-only, web-only, native-shell sidecar-only, fake, mock, or placeholder evidence; macOS/Windows, GTK/Qt/OS widget backend claims, platform accessibility integration, and broad input/change/focus behavior remain outside this claim until host-native reports exist",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_v1.md", "docs/spec/ui_v0.4.0.md", "docs/user/wasm_ui_guide.md"},
		},
		{
			ID:        "ui.platform-runtime",
			Name:      "Cross-platform UI runtime promotion gate",
			Status:    FeatureStatusExperimental,
			Since:     "v0.4.0",
			Scope:     "tetra.ui.platform-runtime.v1 full-platform UI runtime promotion gate for Linux, Windows, macOS, and Web evidence; Windows/macOS require real Windows/macOS target-host reports before they can count as production UI runtime targets",
			Stability: "not production until the full-platform UI runtime promotion gate passes with real Windows/macOS target-host reports and rejects metadata-only, runtime-less, build-only, sidecar-only, fake/mock/placeholder, and startup_failure evidence as blockers rather than platform runtime proof",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_v1.md", "docs/user/wasm_ui_guide.md"},
		},
	}
	out := make([]FeatureInfo, len(features))
	copy(out, features)
	for i := range out {
		out[i].Docs = append([]string(nil), features[i].Docs...)
		if out[i].ID == "compiler.verified-track" {
			out[i].Scope += "; P17.3 simple map over []i32 executable evidence is limited to proof-tagged in-place add-constant-1 linux-x64 native SIMD through vector-i32x4-map-add-const-plan, single mutable slice in-place noalias-not-required evidence, safe unaligned i32x4 map load/store, scalar-i32-map fallback, and stack-fallback translation/differential validation; P17.3 memset/memcpy helper executable evidence is limited to proof-tagged zero-fill memset_zero_u8 through vector-u8x16-memset-zero-plan plus memcpy helper via copy []u8 evidence; no checked/no-proof map, broader map-shape vectorization, arbitrary non-zero memset, overlapping memcpy, checked/no-proof helper, libc/runtime helper lowering, or performance claim is made"
			out[i].Scope += "; typed actor ownership transfer v1 records tetra.actors.ownership_transfer.v1 rows for borrowed-view copy boundaries, owned-region move, sender use-after-move diagnostics, receiver ownership evidence, explicit copy fallback, unsafe-send contract model evidence, semantics transfer checker, PLIR moved facts with FactMoved and OpActorSend for direct core.send_typed ownership transfers, runtime mailbox representation, actor-transfer reports, stress diagnostics, fake distributed zero-copy rejection, and fake runtime-behavior-change rejection; no distributed pointer or region zero-copy, safe typed actor raw pointer payload, actor scheduler promotion, or production actor runtime claim is made"
			out[i].Scope += "; per-core scheduler v1 records tetra.parallel.per_core_scheduler.v1 rows for per-core queues, work stealing, bounded typed mailboxes, backpressure, timers sleep/wake, structured task groups, cancellation checkpoints, actor ping-pong, fanout/fanin, task group cancel, backpressure overflow, mailbox fairness with FIFO receive, stress evidence, race detector where applicable, fake full production actor-runtime rejection, fake runtime-behavior-change rejection, and fake all-target race-detector rejection; no non-Linux distributed actor runtime target, full production actor runtime, full race-safety proof, scheduler performance claim, public runtime mode, or safe-semantics flag change is claimed"
			out[i].Scope += "; stable generic collections v1 records tetra.stdlib.generic_collections.v1 rows for stable Tetra-source Vec<T> and HashMap<K,V> caller-owned slice views, generic value representation through genericTypeName and mangleGenericName, generic-struct parameter inference through bindGenericNamedTypeArgs, monomorphized vec_from_slice<T> and hash_map_from_slices<K,V> operations, common hash_map_get_i32_i32_or and hash_map_get_u8_i32_or specializations, allocation-plan report linkage through core.make_* caller allocations, and a checked truth-bench-harness dry-run artifact for scope p19.1_generic_collections with hash table Tetra/C++/Rust equivalents, report path reports/stable-generic-collections-v1/benchmarks/generic-collections-hash-table-report.json, matching algorithm_id/input metadata, and Tetra proof/allocation/bounds/performance report artifacts; no allocator-backed production Vec<T>/HashMap<K,V> runtime, generic hashing/equality protocol, C++/Rust parity, broad production stdlib, hidden runtime allocator, measured speed comparison, or official benchmark result is claimed"
			out[i].Scope += "; production HTTP/JSON stack v1 foundation records tetra.stdlib.http_json.production_stack.v1 rows for HTTP/1.1 request-head parsing, pipelined request heads, headers/body/keep-alive metadata, zero-heap request-view evidence, JSON parse/stringify, response building, internal per-server UTC-second Date cache helper evidence through HTTPDateCache and FormatWithReport, Linux writev/sendfile helper evidence through netrt.Writev and netrt.Sendfile, and a checked truth-bench-harness dry-run artifact for scope p19.2_http_json_source_first with Tetra-only HTTP plaintext and HTTP JSON rows, report path reports/production-http-json-v1/benchmarks/http-json-source-first-report.json, matching algorithm_id/input metadata, and Tetra proof/allocation/bounds/P19.2 coverage artifacts; webrt.flush scatter/gather integration, HTTP static-file sendfile path, and non-Linux writev/sendfile parity remain documented boundaries, and no full production web stack, official TechEmpower result, production PostgreSQL stack, P20 performance matrix, C++/Rust parity, measured speed comparison, source-level cached-date API, cross-worker Date cache, zero-copy production file-serving, or runtime behavior change is claimed"
			out[i].Scope += "; production PostgreSQL driver/pool v1 closure records tetra.stdlib.postgresql.production_driver.v1 rows for startup/SCRAM, prepared statements, binary int4 helpers, pooling/backpressure, borrowed DataRow decode, local DB single query, DB multiple queries, DB updates, DB fortunes endpoint workloads, a checked truth-bench-harness dry-run artifact for scope p19.3_postgres_source_first with Tetra-only DB rows, report path reports/production-postgres-v1/benchmarks/postgres-source-first-report.json, matching algorithm_id/input metadata, and Tetra proof/allocation/bounds/P19.3 coverage artifacts, plus live local SCRAM benchmark honesty evidence through validate-techempower-report on docs/benchmarks/techempower_scram_single_query_local_report.json, docs/benchmarks/techempower_scram_single_query_matrix_local_report.json, and docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json; no official TechEmpower result, production database benchmark, P20 performance matrix, C++/Rust parity, external production database deployment, full source-level PostgreSQL driver API, measured speed comparison, or runtime behavior change is claimed"
			out[i].Scope += "; benchmark matrix hardening v1 records the p20.0_benchmark_matrix truth-bench-harness contract with 68 checked dry-run rows for 17 master-plan categories across Tetra, C clang -O3, C++ clang++ -O3, and Rust rustc -C opt-level=3, including matching algorithm_id/input metadata, raw output artifacts on every row, Tetra proof/allocation/bounds/performance artifacts on every Tetra row, report path reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-hardening-report.json, and row target CPU consistency with host target CPU; no measured speed comparison, C++/Rust parity, official benchmark result, official TechEmpower result, production database benchmark, P20.1 blocker completeness, P20.2 claim-tier promotion, throughput advantage, latency advantage, startup-time advantage, binary-size advantage, or compile-time advantage is claimed"
			out[i].Scope += "; performance blocker reports v1 records compiler .perf.json schema_version 3 for P20.1 with matrix scope p20.0_benchmark_matrix, report path reports/benchmark-matrix-hardening-v1/benchmarks/artifacts/p20-matrix-hardening.perf.json, ValidatePerformanceBlockerReport, the exact blocker reasons left bounds check: missing dominance, heap allocation: escapes through return, heap allocation: unknown call, not vectorized: no noalias proof, not inlined: code-size budget, register spill: live range pressure, stack fallback: unsupported aggregate return, and actor copy: borrowed data crosses boundary, plus 17 P20.0 Tetra benchmark explanation rows from integer_loops_tetra through compile_time_tetra; no measured speed comparison, C++/Rust parity, official benchmark result, official TechEmpower result, P20.2 claim-tier promotion, optimizer behavior change, runtime behavior change, blocker removal, throughput advantage, or latency advantage is claimed"
			out[i].Scope += "; claim tiers v1 records tetra.performance.claim_tiers.v1 scope p20.2_claim_tiers with exact Tier 0 local smoke only, Tier 1 local benchmark evidence, Tier 2 reproducible cross-machine benchmark, Tier 3 independent reproduced benchmark, and Tier 4 official upstream benchmark submission policy rows, checked artifact reports/claim-tiers-v1/claim-tier-report.json, current P20.0/P20.1 public claim p20_current_local_smoke_only at tier0_local_smoke_only, required evidence classes local_smoke, local_benchmark, cross_machine_reproduction, independent_reproduction, and official_upstream_submission, and validator rejection for fake local benchmark evidence, cross-machine benchmark, independent reproduced benchmark, official upstream benchmark submission, official TechEmpower, measured speed, throughput advantage, latency advantage, and C++/Rust parity wording unless explicit non-claims or matching tier evidence exist; current P20.0/P20.1 evidence remains Tier 0 only"
			out[i].Scope += "; specialization machine-code evidence v1 records tetra.optimizer.specialization_machine_code.v1 scope p21.2_specialization_v1_v2 rows for generics, protocol/static conformance, extension methods, enum match known cases, optionals, and collections; BuildP21SpecializationMachineCodeWitness uses inline-small-pure plus machine.ScalarIntFunctionFromStackIR to prove a known direct helper call is present before optimization, absent from optimized Stack IR, and absent as OpCall from verified scalar Machine IR, with translation validation; rows connect P17.2 monomorphized generic identity/wrapper, statically checked protocol impl direct calls, statically resolved extension method direct calls, SCCP known enum discriminator branch folding, proven-some optional presence branch folding, and P19.1 caller-owned Vec<T>/HashMap<K,V> monomorphized helper evidence; validator rejects placeholder evidence, missing target rows, fake broad specialization, fake dynamic dispatch, fake runtime generic values, fake allocator-backed generic collections, fake layout/ABI freedom, fake performance, and fake safe-semantics changes"
			out[i].Scope += "; translation validation v2 records tetra.translation.validation.v2 scope p23.0_translation_validation_v2 rows for registered optimizer pass coverage, symbolic scalar equivalence, supported i32 slice memory equivalence, bounds proof preservation, allocation plan preservation, and machine-checkable sha256 before/after optimization metadata; witnesses run opt.NewManager over opt.RegisteredPasses, validation.ValidateTranslation scalar and proof cases, differential backend matrix loop/call/slice samples, validation.ValidateAllocationLowering, and BuildOptimizationValidationMetadata; validator rejects missing rows/witnesses, placeholders, incomplete registered-pass coverage, missing scalar/memory/loop/call/proof/allocation/hash evidence, fake full formal proof, fake exhaustive optimizer completeness, fake broad memory or loop proof claims, fake performance, fake runtime behavior change, and fake safe-semantics changes"
			out[i].Scope += "; fuzz/property/differential expansion v1 records tetra.fuzz.property.differential.v1 scope p23.1_fuzz_property_differential rows for parser/checker generated programs, PLIR/lowering verifier pipeline, backend differential matrix expansion, native backend boundary, runtime allocator properties, actor transfer stress boundary, fuzz nightly summary gate, and reducer failure artifacts; witnesses run compiler.Parse, compiler.Check, BuildPLIR, Lower, VerifyIRProgram, differential.CheckBackendMatrix with deterministic randomized samples, host-supported Linux x64 native backend lane or explicit unavailable boundary, runtimeabi.AlignRegionBytes valid/invalid allocator properties, actorsafety.TypedActorOwnershipTransferCoverage stress diagnostics and PLIR moved facts, fuzz-nightly/validate-fuzz-summary artifact contract, and reduced_to_single_sample mismatch reproducer; validator rejects missing rows/witnesses, placeholders, missing generated parser/checker cases, missing PLIR/lowering verifier cases, missing backend randomized samples, missing reducer evidence, missing native-host sample or explicit non-host boundary, missing runtime allocator property evidence, missing actor-transfer stress diagnostics, missing fuzz summary artifacts or nightly boundary, fake full program correctness, fake exhaustive fuzzing, fake full native differential, fake performance, fake runtime behavior change, and fake safe-semantics changes"
			out[i].Scope += "; formal core v1 records tetra.formal_core.v1 scope p23.2_formal_core_v1 rows for values, borrows and owned/copy, provenance and regions, bounds proof id semantics, allocation length contract, allocation intent lowering, raw pointer bounds metadata, and check-elimination validity; witnesses run formalcore.ValidateSpec, differential.CheckBackendMatrix, compiler.Parse, compiler.Check, BuildPLIR, plir.VerifyProgram, validation.CheckBoundsProofsWithPLIR, allocplan.FromPLIR, validation.ValidateAllocationLowering, runtimeabi.NewRawAllocationBounds, runtimeabi.DeriveRawPointerBounds, and runtimeabi.RawSliceBoundsFromParts; validator rejects missing rows/witnesses, placeholders, missing formal spec validation, missing value samples, missing borrow/copy or provenance/regions facts, missing bounds proof id or check-elimination evidence, missing allocation length contract evidence, missing allocation-intent lowering evidence, missing raw pointer bounds metadata evidence, fake full formal proof, fake broad language proof, fake unsafe policy change, fake runtime behavior change, fake safe-semantics changes, and fake performance"
			out[i].Scope += "; self-hosting gate v1 records tetra.self_hosting.gate.v1 scope p23.3_self_hosting_gate rows for self-host subset definition, small compiler component compile boundary, Go compiler output vs Tetra-compiled output comparison boundary, register backend stability, optimizer validation maturity, allocator/runtime stability, stdlib sufficiency, deterministic bootstrap chain, cross-platform bootstrap story, and no self-hosting claim; witnesses run selfhostgate.Evaluate, differential.CheckBackendMatrix, BuildP23TranslationValidationV2, runtimeabi.RuntimeAllocationContracts, runtimeabi.RuntimeRegionAllocatorConfig, runtimeabi.RuntimePerCoreSmallHeapABI, and stdlibrt.RegionAwareStdlibCoverage; current report requires SelfHostingClaimed=false and GateDecision.Allowed=false, records missing small compiler component, Go-vs-Tetra output comparison, deterministic bootstrap chain, and cross-platform bootstrap story blockers, and validator rejects missing rows/witnesses, placeholders, weak compiler subset/backend/optimizer/allocator/runtime/stdlib evidence, fake self-hosting claim, fake small compiler component, fake output comparison, fake deterministic bootstrap, fake cross-platform bootstrap, fake runtime behavior change, fake safe-semantics changes, and fake performance"
			out[i].Scope += "; security review gate v1 records tetra.security.review_gate.v1 scope p24.0_security_review_gate rows for unsafe API surface, capability surface, memory allocator, network runtime, actor runtime, DB protocol, package/Eco system, build scripts, supply chain, and required artifact set; artifacts are docs/audits/security-review.md, docs/audits/threat-model.md, docs/audits/unsafe-surface-map.md, and docs/audits/capability-surface-map.md; witnesses run runtimeabi.RuntimeAllocationContracts, runtimeabi.RuntimeRawPointerBoundsABI, netrt.IOReactorCoverage, actorsrt.ActorRuntimeProductionBoundaryAudit, pgrt.ProductionPostgresCoverage, Eco validator path checks, release security-review script checks, and artifact presence checks; validator rejects missing rows/witnesses, weak artifacts, fake security certification, fake external penetration test, fake CVE-free status, fake release security signoff, fake runtime behavior change, fake safe-semantics changes, and fake performance"
			out[i].Scope += "; runtime hardening v1 records tetra.runtime.hardening.v1 scope p24.1_runtime_hardening rows for deterministic traps, OOM policy, stack overflow guard boundary, integer overflow semantics audit, allocator corruption detection instrumentation, region double-free/use-after-free instrumentation, actor mailbox overflow policy, and network parser limits; witnesses run runtimeabi.RuntimeAllocationContracts, runtimeabi.RuntimeRegionAllocatorConfig, runtimeabi.RuntimePerCoreSmallHeapABI, runtimeabi.NewPerCoreSmallHeapAllocator, parallelrt.NewTypedMailbox, actorsrt.ActorRuntimeProductionBoundaryAudit, httprt.ParseRequest, httprt.ParseRequestView, pgrt.ReadFrame, backend trap/stack-depth file checks, and optimizer overflow-semantics file checks; validator rejects missing rows/witnesses, placeholders, missing runtime-hardening artifacts, fake full runtime-hardening proof, fake full stack-overflow protection, fake OOM recovery, fake full allocator-corruption detection, fake production actor-mailbox promotion, fake runtime behavior change, fake safe-semantics changes, and fake performance"
			out[i].Scope += "; compatibility/stability v1 records tetra.compatibility.stability.v1 scope p24.2_compatibility_stability rows for stable diagnostic codes, versioned report schemas, manifest compatibility checks, breaking-change migration guide, and deprecation policy; witnesses read DiagnosticCodeRegistry, validate-diagnostic, P21-P24 schema constants, validate-manifest, docs/generated/manifest.json, docs/spec/api_diff_policy.md, docs/release/breaking-change-migration-guide.md, docs/release/deprecation_policy.md, docs/release/v1_0_x_maintenance_policy.md, and docs/spec/stdlib_naming_versioning.md; validator rejects missing rows/witnesses, placeholders, missing compatibility-stability artifacts, fake full backward compatibility, fake frozen diagnostic messages, fake automatic migration, fake manifest/runtime ABI stability, fake breaking change without migration guide, fake removal without deprecation, fake runtime behavior change, fake safe-semantics changes, and fake performance"
			out[i].Docs = append(out[i].Docs, "docs/audits/typed-actor-ownership-transfer-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/per-core-scheduler-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/stable-generic-collections-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/production-http-json-stack-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/production-postgres-driver-pool-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/benchmark-matrix-hardening-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/performance-blocker-reports-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/claim-tiers-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/specialization-machine-code-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/translation-validation-v2.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/fuzz-property-differential-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/formal-core-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/self-hosting-gate-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/security-review.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/threat-model.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/unsafe-surface-map.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/capability-surface-map.md")
			out[i].Docs = append(out[i].Docs, "docs/plans/2026-06-03-p24.0-security-review-gate-design.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/runtime-hardening-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/plans/2026-06-03-p24.1-runtime-hardening-design.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/compatibility-stability-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/plans/2026-06-03-p24.2-compatibility-stability-design.md")
			out[i].Docs = append(out[i].Docs, "docs/release/breaking-change-migration-guide.md")
			out[i].Docs = append(out[i].Docs, "docs/release/deprecation_policy.md")
			out[i].Docs = append(out[i].Docs, "docs/benchmarks/truth_benchmark_harness.md")
		}
		if out[i].ID == "stdlib.core-current" {
			out[i].Scope += "; stable generic collection source views expose lib.core.collections.Vec<T> and HashMap<K,V> over caller-owned slices, generic vec_from_slice<T>/vec_len<T>/vec_get_or<T>/hash_map_from_slices<K,V>/hash_map_len<K,V> helpers, and common hash_map_get_i32_i32_or plus hash_map_get_u8_i32_or lookup specializations"
			out[i].Scope += "; P19.2 HTTP/JSON source-first evidence covers lib.core.http request-head framing, pipelined local buffers, plaintext/JSON response byte-buffer helpers, lib.core.json message-object writers, internal borrowed HTTP/JSON request-region coverage, internal per-server UTC-second Date cache evidence, and Linux netrt.Writev/netrt.Sendfile helper evidence through tetra.stdlib.http_json.production_stack.v1"
			out[i].Scope += "; P19.3 PostgreSQL source-first and local SCRAM evidence covers lib.core.postgres source rows for DB single query, DB multiple queries, DB updates, and DB fortunes plus internal runtime startup/SCRAM, prepared statements, binary int4 helpers, pooling/backpressure, borrowed DataRow decode, and checked local SCRAM benchmark reports through tetra.stdlib.postgresql.production_driver.v1, p19.3_postgres_source_first, and validate-techempower-report"
			out[i].Stability += "; generic collection views are source-level and caller-owned, with no hidden allocator, resizing, generic hashing/equality protocol, production runtime map/vector claim, C++/Rust parity, or official benchmark result"
			out[i].Stability += "; HTTP/JSON P19.2 evidence is source-first and local dry-run only, with no production HTTP server promotion, source-level cached-date API, cross-worker Date cache, webrt.flush scatter/gather integration, HTTP static-file sendfile path, zero-copy production file-serving, P20 performance matrix, C++/Rust parity, or official TechEmpower result"
			out[i].Stability += "; PostgreSQL P19.3 evidence is source-first plus checked local SCRAM evidence only, with no full source-level PostgreSQL driver API, external production database deployment, production database benchmark, P20 performance matrix, C++/Rust parity, official TechEmpower result, measured speed comparison, or runtime behavior change"
			out[i].Docs = append(out[i].Docs, "docs/audits/stable-generic-collections-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/production-http-json-stack-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/production-postgres-driver-pool-v1.md")
		}
	}
	return out
}
