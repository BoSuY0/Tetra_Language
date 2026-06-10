package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestFeatureRegistryCoversReleaseStatusesAndKeyBoundaries(t *testing.T) {
	features := compiler.FeatureRegistry()
	if len(features) == 0 {
		t.Fatal("FeatureRegistry returned no entries")
	}
	seenStatus := map[compiler.FeatureStatus]bool{}
	seenID := map[string]compiler.FeatureStatus{}
	seenFeature := map[string]compiler.FeatureInfo{}
	for _, feature := range features {
		if feature.ID == "" || feature.Name == "" || feature.Scope == "" || feature.Stability == "" {
			t.Fatalf("feature has missing required metadata: %#v", feature)
		}
		if _, exists := seenID[feature.ID]; exists {
			t.Fatalf("duplicate feature ID %s", feature.ID)
		}
		seenID[feature.ID] = feature.Status
		seenFeature[feature.ID] = feature
		seenStatus[feature.Status] = true
		if feature.Status == compiler.FeatureStatusCurrent && feature.Since == "" {
			t.Fatalf("current feature %s missing since", feature.ID)
		}
		if len(feature.Docs) == 0 {
			t.Fatalf("feature %s missing docs", feature.ID)
		}
	}
	for _, status := range []compiler.FeatureStatus{compiler.FeatureStatusCurrent, compiler.FeatureStatusPlanned, compiler.FeatureStatusPostV1} {
		if !seenStatus[status] {
			t.Fatalf("feature registry missing status %s", status)
		}
	}
	for id, wantStatus := range map[string]compiler.FeatureStatus{
		"cli.core":                                compiler.FeatureStatusCurrent,
		"targets.wasm-artifact-preflight":         compiler.FeatureStatusCurrent,
		"language.generics-mvp":                   compiler.FeatureStatusCurrent,
		"language.layout-abi-policy":              compiler.FeatureStatusCurrent,
		"compiler.abi-verification":               compiler.FeatureStatusCurrent,
		"compiler.feature-surface-audit":          compiler.FeatureStatusCurrent,
		"compiler.first-class-callables-v1":       compiler.FeatureStatusCurrent,
		"compiler.protocol-trait-object-decision": compiler.FeatureStatusCurrent,
		"compiler.verified-track":                 compiler.FeatureStatusCurrent,
		"language.protocol-conformance-mvp":       compiler.FeatureStatusCurrent,
		"language.callable-mvp":                   compiler.FeatureStatusCurrent,
		"language.callable-level1":                compiler.FeatureStatusCurrent,
		"stdlib.core-current":                     compiler.FeatureStatusCurrent,
		"stdlib.experimental-mirrors":             compiler.FeatureStatusCurrent,
		"language.enum-payload-match":             compiler.FeatureStatusCurrent,
		"language.protocol-bound-generics-static": compiler.FeatureStatusCurrent,
		"safety.effects-mvp":                      compiler.FeatureStatusCurrent,
		"safety.capabilities-mvp":                 compiler.FeatureStatusCurrent,
		"safety.privacy-consent-mvp":              compiler.FeatureStatusCurrent,
		"safety.budget-mvp":                       compiler.FeatureStatusCurrent,
		"safety.production-core":                  compiler.FeatureStatusCurrent,
		"language.ownership-markers-mvp":          compiler.FeatureStatusCurrent,
		"language.resource-lifetime-mvp":          compiler.FeatureStatusCurrent,
		"actors.task-transfer-safety":             compiler.FeatureStatusCurrent,
		"language.lifetime-ssa":                   compiler.FeatureStatusCurrent,
		"language.callable-level2":                compiler.FeatureStatusCurrent,
		"wasm.runtime-execution":                  compiler.FeatureStatusCurrent,
		"ui.toolkit-core":                         compiler.FeatureStatusCurrent,
		"actors.distributed-runtime":              compiler.FeatureStatusCurrent,
		"eco.distributed-network":                 compiler.FeatureStatusPostV1,
		"ui.native-runtime":                       compiler.FeatureStatusCurrent,
		"ui.platform-runtime":                     compiler.FeatureStatusExperimental,
		"language.full-first-class-callables":     compiler.FeatureStatusCurrent,
	} {
		if gotStatus := seenID[id]; gotStatus != wantStatus {
			t.Fatalf("feature %s status = %q, want %q", id, gotStatus, wantStatus)
		}
	}
	genericsMVP := seenFeature["language.generics-mvp"]
	for _, want := range []string{"statically monomorphized", "tiny generic identity/wrapper", "no runtime generic values or dynamic dispatch", "generic structs", "future/post-v1"} {
		if !strings.Contains(genericsMVP.Scope+" "+genericsMVP.Stability, want) {
			t.Fatalf("generics MVP feature missing %q boundary: %#v", want, genericsMVP)
		}
	}
	safetyProductionCore := seenFeature["safety.production-core"]
	if !strings.Contains(safetyProductionCore.Scope+" "+safetyProductionCore.Stability, "memory cost model") {
		t.Fatalf("safety production core missing memory cost model boundary: %#v", safetyProductionCore)
	}
	if !strings.Contains(safetyProductionCore.Scope+" "+safetyProductionCore.Stability, "memory fuzz oracle") {
		t.Fatalf("safety production core missing memory fuzz oracle boundary: %#v", safetyProductionCore)
	}
	if !strings.Contains(safetyProductionCore.Scope+" "+safetyProductionCore.Stability, "memory production final audit") {
		t.Fatalf("safety production core missing memory production final audit boundary: %#v", safetyProductionCore)
	}
	hasMemoryCostModelDoc := false
	hasMemoryFuzzOracleDoc := false
	hasMemoryProductionFinalDoc := false
	hasMemoryProductionArtifactMapDoc := false
	hasMemoryProductionNonclaimsDoc := false
	for _, doc := range safetyProductionCore.Docs {
		hasMemoryCostModelDoc = hasMemoryCostModelDoc || doc == "docs/design/memory_cost_model.md"
		hasMemoryFuzzOracleDoc = hasMemoryFuzzOracleDoc || doc == "docs/audits/memory-fuzz-oracle-v1.md"
		hasMemoryProductionFinalDoc = hasMemoryProductionFinalDoc || doc == "docs/audits/memory-production-core-v1-final.md"
		hasMemoryProductionArtifactMapDoc = hasMemoryProductionArtifactMapDoc || doc == "docs/audits/memory-production-core-v1-artifact-map.md"
		hasMemoryProductionNonclaimsDoc = hasMemoryProductionNonclaimsDoc || doc == "docs/audits/memory-production-core-v1-nonclaims.md"
	}
	if !hasMemoryCostModelDoc {
		t.Fatalf("safety production core missing memory cost model doc: %#v", safetyProductionCore)
	}
	if !hasMemoryFuzzOracleDoc {
		t.Fatalf("safety production core missing memory fuzz oracle doc: %#v", safetyProductionCore)
	}
	if !hasMemoryProductionFinalDoc || !hasMemoryProductionArtifactMapDoc || !hasMemoryProductionNonclaimsDoc {
		t.Fatalf("safety production core missing MPC-16 final audit docs: %#v", safetyProductionCore)
	}
	layoutABI := seenFeature["language.layout-abi-policy"]
	for _, want := range []string{"default structs", "do not promise C layout", "repr(C)", "ABI-locked", "unavailable for repr(C)", "default layout freedom v1", "p21.0_default_layout_freedom_v1", ".layout.json schema_version 2", "compiler_owned_default", "abi_locked_repr_c", "exported_ffi_explicit_repr_c", "public ABI/exported FFI aggregate boundaries require explicit repr(C)", "field_reordering", "padding_removal", "hot_cold_splitting", "scalar_replacement", "aos_to_soa", "no field reordering", "performance change", "runtime behavior change"} {
		if !strings.Contains(layoutABI.Scope+" "+layoutABI.Stability, want) {
			t.Fatalf("layout/ABI policy feature missing %q boundary: %#v", want, layoutABI)
		}
	}
	hasLayoutAuditDoc := false
	for _, doc := range layoutABI.Docs {
		hasLayoutAuditDoc = hasLayoutAuditDoc || doc == "docs/audits/default-layout-freedom-v1.md"
	}
	if !hasLayoutAuditDoc {
		t.Fatalf("layout/ABI policy feature missing P21.0 audit doc: %#v", layoutABI)
	}
	abiVerification := seenFeature["compiler.abi-verification"]
	for _, want := range []string{"ABI verification v1", "tetra.abi.verification.v1", "p21.1_abi_verification", "linux-x64 SysV", "linux-x86 i386 SysV", "linux-x32 x32 SysV", "macos-x64 SysV", "windows-x64 Win64", "wasm32-wasi", "wasm32-web", "abi_test_corpus", "struct_enum_slice_string_return_validation", "call_boundary_validation", "ffi_repr_c_tests", "compiler-owned i32 slot ABI metadata", "IRCall arg/return slot matching", "no runtime execution claim", "no C ABI claim for default structs", "no native C aggregate ABI claim for wasm targets", "no performance claim", "no safe-program semantics change"} {
		if !strings.Contains(abiVerification.Scope+" "+abiVerification.Stability, want) {
			t.Fatalf("ABI verification feature missing %q boundary: %#v", want, abiVerification)
		}
	}
	hasABIAuditDoc := false
	for _, doc := range abiVerification.Docs {
		hasABIAuditDoc = hasABIAuditDoc || doc == "docs/audits/abi-verification-v1.md"
	}
	if !hasABIAuditDoc {
		t.Fatalf("ABI verification feature missing P21.1 audit doc: %#v", abiVerification)
	}
	featureSurfaceAudit := seenFeature["compiler.feature-surface-audit"]
	for _, want := range []string{"full feature surface audit", "tetra.language.feature_surface_audit.v1", "p22.0_full_feature_surface_audit", "first-class callables", "closures", "protocols/trait objects", "runtime generics", "advanced enums/pattern matching", "async typed errors", "structured concurrency", "modules/packages", "macros/metaprogramming", "UI/surface", "Eco/capsules", "FeatureRegistry statuses", "same-branch evidence", "no full v1 language guarantee", "runtime generic values", "trait objects", "runtime protocol values", "macro/metaprogramming system", "full structured concurrency", "cross-platform production UI runtime", "distributed EcoNet", "proof-carrying capsules", "performance claim", "runtime behavior change", "safe-program semantics change"} {
		if !strings.Contains(featureSurfaceAudit.Scope+" "+featureSurfaceAudit.Stability, want) {
			t.Fatalf("feature surface audit missing %q boundary: %#v", want, featureSurfaceAudit)
		}
	}
	hasFeatureSurfaceAuditDoc := false
	for _, doc := range featureSurfaceAudit.Docs {
		hasFeatureSurfaceAuditDoc = hasFeatureSurfaceAuditDoc || doc == "docs/audits/full-feature-surface-audit-v1.md"
	}
	if !hasFeatureSurfaceAuditDoc {
		t.Fatalf("feature surface audit missing P22.0 audit doc: %#v", featureSurfaceAudit)
	}
	firstClassCallableEvidence := seenFeature["compiler.first-class-callables-v1"]
	for _, want := range []string{"first-class callables v1", "tetra.language.first_class_callables.v1", "p22.1_first_class_callables_v1", "bounded fnptr fast path", "fat callable handle", "capture safety classifier", "mutable capture escape diagnostics", "resource/thread escape diagnostics", "fixed ABI width", "cross-module interface metadata", "storage/callback paths", "one-capture 9-slot fnptr", "without heap environment allocation", "nine-capture fixed 4-slot handle", "IRAllocBytes", "IRMemWritePtrOffset", "IRMemReadPtrOffset", "ArgSlots 10 RetSlots 1", "generated .t4i metadata", "ReturnFunctionHandleValue", "heap escape kind", "ReturnSlots = 4", "no variable-width callable ABI", "exploding return slots", "mutable by-reference capture support", "pointer/resource capture support", "thread-boundary callable transfer", "runtime generic callable polymorphism", "dynamic callable dispatch", "unsafe lifetime relaxation", "performance claim", "runtime behavior change", "safe-program semantics change"} {
		if !strings.Contains(firstClassCallableEvidence.Scope+" "+firstClassCallableEvidence.Stability, want) {
			t.Fatalf("first-class callable evidence feature missing %q boundary: %#v", want, firstClassCallableEvidence)
		}
	}
	hasFirstClassCallableAuditDoc := false
	for _, doc := range firstClassCallableEvidence.Docs {
		hasFirstClassCallableAuditDoc = hasFirstClassCallableAuditDoc || doc == "docs/audits/first-class-callables-v1.md"
	}
	if !hasFirstClassCallableAuditDoc {
		t.Fatalf("first-class callable evidence missing P22.1 audit doc: %#v", firstClassCallableEvidence)
	}
	protocolTraitDecision := seenFeature["compiler.protocol-trait-object-decision"]
	for _, want := range []string{"protocol / trait object decision", "tetra.language.protocol_trait_object_decision.v1", "p22.2_protocol_trait_object_decision", "keep_static_conformance_only", "static conformance fast path", "static protocol-bound generics", "runtime existential decision", "explicit dynamic-dispatch gate", "specialization static abstraction", "witness-table boundary", "trait-object boundary", "registry/docs alignment", "Vec2.draw IRCall", "id__T_Vec2 direct call", "unknown type 'Drawable'", "generic-bound requirement-call rejection", "P17/P21 known-direct specialization evidence", "runtime protocol values", "trait objects", "witness tables", "dynamic dispatch", "conformance-table lookup", "runtime existential ABI", "broad protocol specialization", "performance", "runtime behavior change", "safe-program semantics change"} {
		if !strings.Contains(protocolTraitDecision.Scope+" "+protocolTraitDecision.Stability, want) {
			t.Fatalf("protocol/trait decision feature missing %q boundary: %#v", want, protocolTraitDecision)
		}
	}
	hasProtocolTraitDecisionDoc := false
	for _, doc := range protocolTraitDecision.Docs {
		hasProtocolTraitDecisionDoc = hasProtocolTraitDecisionDoc || doc == "docs/audits/protocol-trait-object-decision-v1.md"
	}
	if !hasProtocolTraitDecisionDoc {
		t.Fatalf("protocol/trait decision missing P22.2 audit doc: %#v", protocolTraitDecision)
	}
	verifiedTrack := seenFeature["compiler.verified-track"]
	for _, want := range []string{"differential scalar-i32", "source interpreter", "stack backend", "register backend", "optimized backend", "optimizer pass contract v1", "input/output verifier evidence", "proof preservation or invalidation rules", "translation validation hooks", "stable report rows", "negative-test markers", "optimizer core coverage v1", "bounded evidence-backed P17.1 closure", "narrow safe const-denominator div_i32/mod_i32 constant folding plus same-local comparison algebraic simplification", "narrow SCCP constant-condition", "known-local and stored safe unary neg_i32 plus safe constant-expression facts including safe const-denominator div_i32/mod_i32", "constant unary neg_i32 and binary-expression branch folding including safe const-denominator div_i32/mod_i32 with unary min-int and denominator 0 and -1 rejected", "immediate and forward-terminated single-predecessor label propagation plus folded zero-branch target propagation for labels with one incoming edge and no fallthrough predecessor", "folded nonzero-branch fallthrough propagation through immediate labels with no explicit incoming branch/jump edges", "dynamic load-local zero-target and nonzero-fallthrough path facts", "fallthrough-predecessor rejection", "explicit-incoming fallthrough-label rejection", "fallthrough pruning", "narrow Stack IR adjacent and stack-neutral separated single-assignment mem2reg temp promotion", "bounded comparison-expression, safe const unary neg_i32, safe known-local unary neg_i32, safe const add_i32/sub_i32/mul_i32 arithmetic, safe known-local add_i32/sub_i32/mul_i32 arithmetic, safe const-denominator div_i32/mod_i32 producer temps, and safe known-local div_i32/mod_i32 producer temps", "unary min-int, arithmetic overflow, source-local mutation, and denominator 0 and -1 rejected", "bounded DCE for simple dead local stores, non-trapping comparison-expression stores, safe known-local unary neg_i32 stores, safe known-local add_i32/sub_i32/mul_i32 stores, safe const-denominator div_i32/mod_i32 stores, and safe known-local div_i32/mod_i32 stores", "narrow exact/commutative/mirrored-comparison local-load, local-load/constant, unary local neg_i32, safe known-local unary neg_i32 value, safe known-local add_i32/sub_i32/mul_i32 value, safe known-local cmp_*_i32 value, safe known-local div_i32/mod_i32 value, and safe const-denominator div_i32/mod_i32 CSE/GVN", "commutative add/mul/eq/ne and mirrored lt/gt/le/ge operand canonicalization", "narrow proof-tagged LICM pure invariant comparison, add/sub/mul arithmetic, known-local add_i32/sub_i32/mul_i32 left-or-right operand hoisting, known-local cmp_*_i32 left-or-right operand hoisting, safe const-denominator div_i32/mod_i32 hoisting, and safe known-local div_i32/mod_i32 denominator hoisting", "bounded hot-loop shape evidence", "scalar sum", "scalar constant-stride sum", "scalar sum-of-squares", "scalar product reduction bounded to product *= index + 1", "scalar branchy max reduction", "scalar affine sum with compile-time scale and bias 1..127", "scalar countdown", "proof-tagged slice sum", "proof-tagged slice constant-stride sum", "call-loop machine IR", "inlining specialization coverage v1", "P17.2 target rows", "monomorphized generic identity/wrapper", "small-pure inline-small-pure", "payload enum known-case match", "proven-some optional match", "sccp-constant-branch evidence", "statically checked protocol/conformance direct-call inline-small-pure evidence", "statically resolved extension-call inline-small-pure evidence", "inlined/not_inlined report reasons", "8-instruction body cap", "constant_stack_store tag tracking", "known direct Stack IR function symbol boundaries", "protocol-bound requirement calls", "witness tables", "trait objects", "runtime protocol values", "conformance-table lookup", "vectorization coverage v1", "P17.3 initial target rows", "proof-tagged sum []i32 candidate recognition", "range-proof evidence", "noalias-not-required read-only reduction evidence", "safe unaligned i32x4 vector backend lowering", "vector-i32x4-slice-sum-plan", "linux-x64 native SIMD lowering", "scalar tail handling", "scalar-i32-slice-sum fallback", "translation/differential validation", "proof-tagged copy []u8 vector backend lowering", "vector-u8x16-copy-plan", "noalias required source/dest disjoint", "safe unaligned u8x16 load/store", "scalar-u8-copy fallback", "linux-x64 native SIMD lowering for proof-tagged copy []u8", "copy []u8 translation/differential validation against stack fallback", "proof-tagged simple map over []i32 guarded vector backend lowering", "vector-i32x4-map-add-const-plan", "single mutable slice in-place noalias-not-required evidence", "safe unaligned i32x4 map load/store", "scalar-i32-map fallback", "linux-x64 native SIMD lowering for proof-tagged in-place add-constant-1 map []i32", "map []i32 translation/differential validation against stack fallback", "proof-tagged in-place add-constant-1 linux-x64 native SIMD", "proof-tagged memset/memcpy helper evidence", "vector-u8x16-memset-zero-plan", "single mutable slice zero-fill noalias-not-required evidence", "safe unaligned u8x16 zero-store", "scalar-u8-memset-zero fallback", "linux-x64 native SIMD zero-fill lowering for proof-tagged memset_zero_u8", "memset_zero_u8 translation/differential validation against stack fallback", "memcpy helper via copy []u8 evidence", "broader map-shape vectorization", "checked/no-proof copy", "overlapping copy", "checked/no-proof map", "arbitrary non-zero memset", "overlapping memcpy", "checked/no-proof helper", "libc/runtime helper lowering", "no broad SIMD auto-vectorization", "performance claim", "validation metadata", "sha256", "actor runtime production-boundary audit v1", "tetra.runtime.actor.production_boundary.v1", "current actor runtime limits", "scheduler prototype features", "production runtime acceptance", "full claim blockers", "fake full production actor runtime claim rejection", "production multi-threaded actor scheduling", "non-Linux-x64 distributed actor runtime targets", "message-pool exhaustion/reclamation", "full cancellation and structured concurrency", "full race-safety proof", "production broker deployment evidence", "self-hosting gate", "formal core spec", "not a public backend selector", "full formal proof"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing %q boundary: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"PGO/LTO/target-cpu evidence v1", "tetra.optimizer.profile.v1", "canonical JSON profile collection format", "duplicate and negative counter rejection", "Options.ProfileInput optimizer profile input API", "profile_input_policy pass-contract metadata", "profile digest validation metadata", "translation validation for profile-input foundation runs", "profile-guided rewrite policy rejection", "profile parsing is evidence-only", "target-cpu feature detection foundation", "portable baseline target-feature model", "guarded codegen contract", "no target-specific rewrite", "LTO/incremental module summary foundation", "tetra.incremental.module_summary.v1", "dependency hash contract", "non-consumer boundary", "no LTO optimizer or incremental speedup claim", "final safe-semantics closure validator rejects fake semantic-changing coverage", "target-specific optimization evidence", "LTO/codegen/linker consumers", "no PGO, LTO, target-cpu, or profile flag changes safe-program semantics"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P17.4 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"typed actor ownership transfer v1", "tetra.actors.ownership_transfer.v1", "borrowed-view copy boundaries", "owned-region move", "sender use-after-move diagnostics", "receiver ownership evidence", "explicit copy fallback", "unsafe-send contract model evidence", "semantics transfer checker", "PLIR moved facts", "FactMoved", "OpActorSend", "direct core.send_typed ownership transfers", "runtime mailbox representation", "actor-transfer reports", "stress diagnostics", "fake distributed zero-copy rejection", "fake runtime-behavior-change rejection", "no distributed pointer or region zero-copy", "safe typed actor raw pointer payload", "actor scheduler promotion", "production actor runtime claim"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P18.1 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"per-core scheduler v1", "tetra.parallel.per_core_scheduler.v1", "per-core queues", "work stealing", "bounded typed mailboxes", "backpressure", "timers sleep/wake", "structured task groups", "cancellation checkpoints", "actor ping-pong", "fanout/fanin", "task group cancel", "backpressure overflow", "mailbox fairness", "FIFO receive", "stress evidence", "race detector where applicable", "fake full production actor-runtime rejection", "fake runtime-behavior-change rejection", "fake all-target race-detector rejection", "no non-Linux distributed actor runtime target", "full production actor runtime", "full race-safety proof", "scheduler performance claim", "public runtime mode", "safe-semantics flag change"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P18.2 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"async I/O reactor v1", "tetra.runtime.io_reactor.v1", "Linux epoll v1", "io_uring future boundary", "kqueue macOS boundary", "IOCP Windows boundary", "WASI/web adapter boundary", "nonblocking accept/read/write", "readiness polling", "task wakeups from I/O readiness", "timer integration", "cancellation", "backpressure", "reactor report rows", "HTTP smoke", "DB smoke", "stress evidence", "fake full production web-stack rejection", "fake cross-platform reactor parity rejection", "fake io_uring rejection", "fake runtime-behavior-change rejection", "clear production boundary per platform", "no full production web stack", "cross-platform reactor parity", "io_uring support", "runtime behavior change", "official TechEmpower result", "production HTTP/PostgreSQL stack promotion"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P18.3 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"stable generic collections v1", "tetra.stdlib.generic_collections.v1", "Vec<T>", "HashMap<K,V>", "caller-owned slice views", "genericTypeName", "mangleGenericName", "bindGenericNamedTypeArgs", "vec_from_slice<T>", "hash_map_from_slices<K,V>", "hash_map_get_i32_i32_or", "hash_map_get_u8_i32_or", "allocation-plan report linkage", "core.make_*", "checked truth-bench-harness dry-run artifact", "p19.1_generic_collections", "hash table Tetra/C++/Rust equivalents", "reports/stable-generic-collections-v1/benchmarks/generic-collections-hash-table-report.json", "algorithm_id/input metadata", "Tetra proof/allocation/bounds/performance report artifacts", "no allocator-backed production Vec<T>/HashMap<K,V> runtime", "generic hashing/equality protocol", "C++/Rust parity", "broad production stdlib", "hidden runtime allocator", "measured speed comparison", "official benchmark result"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P19.1 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"production HTTP/JSON stack v1 foundation", "tetra.stdlib.http_json.production_stack.v1", "HTTP/1.1 request-head parsing", "pipelined request heads", "headers/body/keep-alive metadata", "zero-heap request-view evidence", "JSON parse/stringify", "response building", "internal per-server UTC-second Date cache helper evidence", "HTTPDateCache", "FormatWithReport", "Linux writev/sendfile helper evidence", "netrt.Writev", "netrt.Sendfile", "p19.2_http_json_source_first", "Tetra-only HTTP plaintext and HTTP JSON rows", "reports/production-http-json-v1/benchmarks/http-json-source-first-report.json", "algorithm_id/input metadata", "Tetra proof/allocation/bounds/P19.2 coverage artifacts", "webrt.flush scatter/gather integration", "HTTP static-file sendfile path", "non-Linux writev/sendfile parity", "no full production web stack", "official TechEmpower result", "production PostgreSQL stack", "P20 performance matrix", "C++/Rust parity", "measured speed comparison", "source-level cached-date API", "cross-worker Date cache", "zero-copy production file-serving", "runtime behavior change"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P19.2 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"production PostgreSQL driver/pool v1 closure", "tetra.stdlib.postgresql.production_driver.v1", "startup/SCRAM", "prepared statements", "binary int4 helpers", "pooling/backpressure", "borrowed DataRow decode", "DB single query", "DB multiple queries", "DB updates", "DB fortunes", "p19.3_postgres_source_first", "Tetra-only DB rows", "reports/production-postgres-v1/benchmarks/postgres-source-first-report.json", "algorithm_id/input metadata", "Tetra proof/allocation/bounds/P19.3 coverage artifacts", "live local SCRAM benchmark honesty evidence", "validate-techempower-report", "techempower_scram_single_query_local_report.json", "techempower_scram_single_query_matrix_local_report.json", "techempower_scram_endpoint_matrix_local_report.json", "production database benchmark", "external production database deployment", "full source-level PostgreSQL driver API", "official TechEmpower result", "P20 performance matrix", "C++/Rust parity", "measured speed comparison", "runtime behavior change"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P19.3 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"benchmark matrix hardening v1", "p20.0_benchmark_matrix", "68 checked dry-run rows", "17 master-plan categories", "Tetra, C clang -O3, C++ clang++ -O3, and Rust rustc -C opt-level=3", "algorithm_id/input metadata", "raw output artifacts on every row", "Tetra proof/allocation/bounds/performance artifacts", "reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-hardening-report.json", "row target CPU consistency", "host target CPU", "measured speed comparison", "C++/Rust parity", "official benchmark result", "official TechEmpower result", "production database benchmark", "P20.1 blocker completeness", "P20.2 claim-tier promotion", "throughput advantage", "latency advantage", "startup-time advantage", "binary-size advantage", "compile-time advantage"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P20.0 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"performance blocker reports v1", ".perf.json schema_version 3", "P20.1", "p20.0_benchmark_matrix", "reports/benchmark-matrix-hardening-v1/benchmarks/artifacts/p20-matrix-hardening.perf.json", "ValidatePerformanceBlockerReport", "left bounds check: missing dominance", "heap allocation: escapes through return", "heap allocation: unknown call", "not vectorized: no noalias proof", "not inlined: code-size budget", "register spill: live range pressure", "stack fallback: unsupported aggregate return", "actor copy: borrowed data crosses boundary", "17 P20.0 Tetra benchmark explanation rows", "integer_loops_tetra", "compile_time_tetra", "measured speed comparison", "C++/Rust parity", "official benchmark result", "official TechEmpower result", "P20.2 claim-tier promotion", "optimizer behavior change", "runtime behavior change", "blocker removal", "throughput advantage", "latency advantage"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P20.1 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"claim tiers v1", "tetra.performance.claim_tiers.v1", "p20.2_claim_tiers", "Tier 0 local smoke only", "Tier 1 local benchmark evidence", "Tier 2 reproducible cross-machine benchmark", "Tier 3 independent reproduced benchmark", "Tier 4 official upstream benchmark submission", "reports/claim-tiers-v1/claim-tier-report.json", "p20_current_local_smoke_only", "tier0_local_smoke_only", "local_smoke", "local_benchmark", "cross_machine_reproduction", "independent_reproduction", "official_upstream_submission", "fake local benchmark evidence", "cross-machine benchmark", "independent reproduced benchmark", "official upstream benchmark submission", "official TechEmpower", "measured speed", "throughput advantage", "latency advantage", "C++/Rust parity", "explicit non-claims", "current P20.0/P20.1 evidence remains Tier 0 only"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P20.2 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"specialization machine-code evidence v1", "tetra.optimizer.specialization_machine_code.v1", "p21.2_specialization_v1_v2", "generics", "protocol/static conformance", "extension methods", "enum match known cases", "optionals", "collections", "BuildP21SpecializationMachineCodeWitness", "inline-small-pure", "machine.ScalarIntFunctionFromStackIR", "absent from optimized Stack IR", "absent as OpCall", "verified scalar Machine IR", "translation validation", "monomorphized generic identity/wrapper", "statically checked protocol impl direct calls", "statically resolved extension method direct calls", "SCCP known enum discriminator branch folding", "proven-some optional presence branch folding", "P19.1 caller-owned Vec<T>/HashMap<K,V>", "validator rejects placeholder evidence", "fake broad specialization", "fake dynamic dispatch", "fake runtime generic values", "fake allocator-backed generic collections", "fake layout/ABI freedom", "fake performance", "fake safe-semantics changes"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P21.2 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"translation validation v2", "tetra.translation.validation.v2", "p23.0_translation_validation_v2", "registered optimizer pass coverage", "symbolic scalar equivalence", "supported i32 slice memory equivalence", "bounds proof preservation", "allocation plan preservation", "machine-checkable sha256 before/after optimization metadata", "opt.NewManager", "opt.RegisteredPasses", "validation.ValidateTranslation", "differential backend matrix loop/call/slice samples", "validation.ValidateAllocationLowering", "BuildOptimizationValidationMetadata", "fake full formal proof", "fake exhaustive optimizer completeness", "fake broad memory or loop proof claims", "fake performance", "fake runtime behavior change", "fake safe-semantics changes"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P23.0 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"fuzz/property/differential expansion v1", "tetra.fuzz.property.differential.v1", "p23.1_fuzz_property_differential", "parser/checker generated programs", "PLIR/lowering verifier pipeline", "backend differential matrix expansion", "native backend boundary", "runtime allocator properties", "actor transfer stress boundary", "fuzz nightly summary gate", "reducer failure artifacts", "compiler.Parse", "compiler.Check", "BuildPLIR", "Lower", "VerifyIRProgram", "differential.CheckBackendMatrix", "deterministic randomized samples", "Linux x64 native backend lane", "explicit unavailable boundary", "runtimeabi.AlignRegionBytes", "actorsafety.TypedActorOwnershipTransferCoverage", "stress diagnostics", "PLIR moved facts", "validate-fuzz-summary", "reduced_to_single_sample", "fake full program correctness", "fake exhaustive fuzzing", "fake full native differential", "fake performance", "fake runtime behavior change", "fake safe-semantics changes"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P23.1 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"formal core v1", "tetra.formal_core.v1", "p23.2_formal_core_v1", "values", "borrows and owned/copy", "provenance and regions", "bounds proof id semantics", "allocation length contract", "allocation intent lowering", "raw pointer bounds metadata", "check-elimination validity", "formalcore.ValidateSpec", "differential.CheckBackendMatrix", "compiler.Parse", "compiler.Check", "BuildPLIR", "plir.VerifyProgram", "validation.CheckBoundsProofsWithPLIR", "allocplan.FromPLIR", "validation.ValidateAllocationLowering", "runtimeabi.NewRawAllocationBounds", "runtimeabi.DeriveRawPointerBounds", "runtimeabi.RawSliceBoundsFromParts", "fake full formal proof", "fake broad language proof", "fake unsafe policy change", "fake runtime behavior change", "fake safe-semantics changes", "fake performance"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P23.2 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"self-hosting gate v1", "tetra.self_hosting.gate.v1", "p23.3_self_hosting_gate", "self-host subset definition", "small compiler component compile boundary", "Go compiler output vs Tetra-compiled output comparison boundary", "register backend stability", "optimizer validation maturity", "allocator/runtime stability", "stdlib sufficiency", "deterministic bootstrap chain", "cross-platform bootstrap story", "SelfHostingClaimed=false", "GateDecision.Allowed=false", "selfhostgate.Evaluate", "differential.CheckBackendMatrix", "BuildP23TranslationValidationV2", "runtimeabi.RuntimeAllocationContracts", "runtimeabi.RuntimeRegionAllocatorConfig", "runtimeabi.RuntimePerCoreSmallHeapABI", "stdlibrt.RegionAwareStdlibCoverage", "missing small compiler component", "Go-vs-Tetra output comparison", "fake self-hosting claim", "fake small compiler component", "fake output comparison", "fake deterministic bootstrap", "fake cross-platform bootstrap", "fake runtime behavior change", "fake safe-semantics changes", "fake performance"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P23.3 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"security review gate v1", "tetra.security.review_gate.v1", "p24.0_security_review_gate", "unsafe API surface", "capability surface", "memory allocator", "network runtime", "actor runtime", "DB protocol", "package/Eco system", "build scripts", "supply chain", "required artifact set", "security-review.md", "threat-model.md", "unsafe-surface-map.md", "capability-surface-map.md", "runtimeabi.RuntimeAllocationContracts", "runtimeabi.RuntimeRawPointerBoundsABI", "netrt.IOReactorCoverage", "actorsrt.ActorRuntimeProductionBoundaryAudit", "pgrt.ProductionPostgresCoverage", "Eco validator path checks", "release security-review script checks", "artifact presence checks", "fake security certification", "fake external penetration test", "fake CVE-free status", "fake release security signoff", "fake runtime behavior change", "fake safe-semantics changes", "fake performance"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P24.0 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"runtime hardening v1", "tetra.runtime.hardening.v1", "p24.1_runtime_hardening", "deterministic traps", "OOM policy", "stack overflow guard boundary", "integer overflow semantics audit", "allocator corruption detection instrumentation", "region double-free/use-after-free instrumentation", "actor mailbox overflow policy", "network parser limits", "runtimeabi.RuntimeAllocationContracts", "runtimeabi.RuntimeRegionAllocatorConfig", "runtimeabi.RuntimePerCoreSmallHeapABI", "runtimeabi.NewPerCoreSmallHeapAllocator", "parallelrt.NewTypedMailbox", "actorsrt.ActorRuntimeProductionBoundaryAudit", "httprt.ParseRequest", "httprt.ParseRequestView", "pgrt.ReadFrame", "backend trap/stack-depth file checks", "optimizer overflow-semantics file checks", "missing runtime-hardening artifacts", "fake full runtime-hardening proof", "fake full stack-overflow protection", "fake OOM recovery", "fake full allocator-corruption detection", "fake production actor-mailbox promotion", "fake runtime behavior change", "fake safe-semantics changes", "fake performance"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P24.1 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{"compatibility/stability v1", "tetra.compatibility.stability.v1", "p24.2_compatibility_stability", "stable diagnostic codes", "versioned report schemas", "manifest compatibility checks", "breaking-change migration guide", "deprecation policy", "DiagnosticCodeRegistry", "validate-diagnostic", "P21-P24 schema constants", "validate-manifest", "docs/generated/manifest.json", "docs/spec/api_diff_policy.md", "docs/release/breaking-change-migration-guide.md", "docs/release/deprecation_policy.md", "docs/release/v1_0_x_maintenance_policy.md", "docs/spec/stdlib_naming_versioning.md", "fake full backward compatibility", "fake frozen diagnostic messages", "fake automatic migration", "fake manifest/runtime ABI stability", "fake breaking change without migration guide", "fake removal without deprecation", "fake runtime behavior change", "fake safe-semantics changes", "fake performance"} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P24.2 boundary %q: %#v", want, verifiedTrack)
		}
	}
	hasP17Doc := false
	hasP18OwnershipDoc := false
	hasP18SchedulerDoc := false
	hasP18ReactorDoc := false
	hasP19GenericCollectionsDoc := false
	hasP19HTTPJSONDoc := false
	hasP19PostgresDoc := false
	hasP20BenchmarkMatrixDoc := false
	hasP20PerformanceBlockerDoc := false
	hasP20ClaimTiersDoc := false
	hasP21SpecializationMachineDoc := false
	hasP23TranslationValidationDoc := false
	hasP23FuzzPropertyDifferentialDoc := false
	hasP23FormalCoreDoc := false
	hasP23SelfHostingGateDoc := false
	hasP24SecurityReviewDoc := false
	hasP24ThreatModelDoc := false
	hasP24UnsafeSurfaceMapDoc := false
	hasP24CapabilitySurfaceMapDoc := false
	hasP24SecurityReviewDesignDoc := false
	hasP24RuntimeHardeningDoc := false
	hasP24RuntimeHardeningDesignDoc := false
	hasP24CompatibilityStabilityDoc := false
	hasP24CompatibilityStabilityDesignDoc := false
	hasP24BreakingChangeMigrationGuideDoc := false
	hasP24DeprecationPolicyDoc := false
	hasTruthBenchmarkHarnessDoc := false
	for _, doc := range verifiedTrack.Docs {
		hasP17Doc = hasP17Doc || doc == "docs/audits/pgo-lto-target-cpu-v1.md"
		hasP18OwnershipDoc = hasP18OwnershipDoc || doc == "docs/audits/typed-actor-ownership-transfer-v1.md"
		hasP18SchedulerDoc = hasP18SchedulerDoc || doc == "docs/audits/per-core-scheduler-v1.md"
		hasP18ReactorDoc = hasP18ReactorDoc || doc == "docs/audits/async-io-reactor-v1.md"
		hasP19GenericCollectionsDoc = hasP19GenericCollectionsDoc || doc == "docs/audits/stable-generic-collections-v1.md"
		hasP19HTTPJSONDoc = hasP19HTTPJSONDoc || doc == "docs/audits/production-http-json-stack-v1.md"
		hasP19PostgresDoc = hasP19PostgresDoc || doc == "docs/audits/production-postgres-driver-pool-v1.md"
		hasP20BenchmarkMatrixDoc = hasP20BenchmarkMatrixDoc || doc == "docs/audits/benchmark-matrix-hardening-v1.md"
		hasP20PerformanceBlockerDoc = hasP20PerformanceBlockerDoc || doc == "docs/audits/performance-blocker-reports-v1.md"
		hasP20ClaimTiersDoc = hasP20ClaimTiersDoc || doc == "docs/audits/claim-tiers-v1.md"
		hasP21SpecializationMachineDoc = hasP21SpecializationMachineDoc || doc == "docs/audits/specialization-machine-code-v1.md"
		hasP23TranslationValidationDoc = hasP23TranslationValidationDoc || doc == "docs/audits/translation-validation-v2.md"
		hasP23FuzzPropertyDifferentialDoc = hasP23FuzzPropertyDifferentialDoc || doc == "docs/audits/fuzz-property-differential-v1.md"
		hasP23FormalCoreDoc = hasP23FormalCoreDoc || doc == "docs/audits/formal-core-v1.md"
		hasP23SelfHostingGateDoc = hasP23SelfHostingGateDoc || doc == "docs/audits/self-hosting-gate-v1.md"
		hasP24SecurityReviewDoc = hasP24SecurityReviewDoc || doc == "docs/audits/security-review.md"
		hasP24ThreatModelDoc = hasP24ThreatModelDoc || doc == "docs/audits/threat-model.md"
		hasP24UnsafeSurfaceMapDoc = hasP24UnsafeSurfaceMapDoc || doc == "docs/audits/unsafe-surface-map.md"
		hasP24CapabilitySurfaceMapDoc = hasP24CapabilitySurfaceMapDoc || doc == "docs/audits/capability-surface-map.md"
		hasP24SecurityReviewDesignDoc = hasP24SecurityReviewDesignDoc || doc == "docs/plans/2026-06-03-p24.0-security-review-gate-design.md"
		hasP24RuntimeHardeningDoc = hasP24RuntimeHardeningDoc || doc == "docs/audits/runtime-hardening-v1.md"
		hasP24RuntimeHardeningDesignDoc = hasP24RuntimeHardeningDesignDoc || doc == "docs/plans/2026-06-03-p24.1-runtime-hardening-design.md"
		hasP24CompatibilityStabilityDoc = hasP24CompatibilityStabilityDoc || doc == "docs/audits/compatibility-stability-v1.md"
		hasP24CompatibilityStabilityDesignDoc = hasP24CompatibilityStabilityDesignDoc || doc == "docs/plans/2026-06-03-p24.2-compatibility-stability-design.md"
		hasP24BreakingChangeMigrationGuideDoc = hasP24BreakingChangeMigrationGuideDoc || doc == "docs/release/breaking-change-migration-guide.md"
		hasP24DeprecationPolicyDoc = hasP24DeprecationPolicyDoc || doc == "docs/release/deprecation_policy.md"
		hasTruthBenchmarkHarnessDoc = hasTruthBenchmarkHarnessDoc || doc == "docs/benchmarks/truth_benchmark_harness.md"
	}
	if !hasP17Doc {
		t.Fatalf("verified track feature missing P17.4 audit doc: %#v", verifiedTrack)
	}
	if !hasP18OwnershipDoc {
		t.Fatalf("verified track feature missing P18.1 audit doc: %#v", verifiedTrack)
	}
	if !hasP18SchedulerDoc {
		t.Fatalf("verified track feature missing P18.2 audit doc: %#v", verifiedTrack)
	}
	if !hasP18ReactorDoc {
		t.Fatalf("verified track feature missing P18.3 audit doc: %#v", verifiedTrack)
	}
	if !hasP19GenericCollectionsDoc {
		t.Fatalf("verified track feature missing P19.1 audit doc: %#v", verifiedTrack)
	}
	if !hasP19HTTPJSONDoc {
		t.Fatalf("verified track feature missing P19.2 audit doc: %#v", verifiedTrack)
	}
	if !hasP19PostgresDoc {
		t.Fatalf("verified track feature missing P19.3 audit doc: %#v", verifiedTrack)
	}
	if !hasP20BenchmarkMatrixDoc {
		t.Fatalf("verified track feature missing P20.0 audit doc: %#v", verifiedTrack)
	}
	if !hasP20PerformanceBlockerDoc {
		t.Fatalf("verified track feature missing P20.1 audit doc: %#v", verifiedTrack)
	}
	if !hasP20ClaimTiersDoc {
		t.Fatalf("verified track feature missing P20.2 audit doc: %#v", verifiedTrack)
	}
	if !hasP21SpecializationMachineDoc {
		t.Fatalf("verified track feature missing P21.2 audit doc: %#v", verifiedTrack)
	}
	if !hasP23TranslationValidationDoc {
		t.Fatalf("verified track feature missing P23.0 audit doc: %#v", verifiedTrack)
	}
	if !hasP23FuzzPropertyDifferentialDoc {
		t.Fatalf("verified track feature missing P23.1 audit doc: %#v", verifiedTrack)
	}
	if !hasP23FormalCoreDoc {
		t.Fatalf("verified track feature missing P23.2 audit doc: %#v", verifiedTrack)
	}
	if !hasP23SelfHostingGateDoc {
		t.Fatalf("verified track feature missing P23.3 audit doc: %#v", verifiedTrack)
	}
	if !hasP24SecurityReviewDoc || !hasP24ThreatModelDoc || !hasP24UnsafeSurfaceMapDoc || !hasP24CapabilitySurfaceMapDoc || !hasP24SecurityReviewDesignDoc {
		t.Fatalf("verified track feature missing P24.0 security review docs: %#v", verifiedTrack)
	}
	if !hasP24RuntimeHardeningDoc || !hasP24RuntimeHardeningDesignDoc {
		t.Fatalf("verified track feature missing P24.1 runtime hardening docs: %#v", verifiedTrack)
	}
	if !hasP24CompatibilityStabilityDoc || !hasP24CompatibilityStabilityDesignDoc || !hasP24BreakingChangeMigrationGuideDoc || !hasP24DeprecationPolicyDoc {
		t.Fatalf("verified track feature missing P24.2 compatibility/stability docs: %#v", verifiedTrack)
	}
	if !hasTruthBenchmarkHarnessDoc {
		t.Fatalf("verified track feature missing truth benchmark harness doc: %#v", verifiedTrack)
	}
	protocolMVP := seenFeature["language.protocol-conformance-mvp"]
	for _, want := range []string{"checked statically", "generic requirement signature shape", "no witness tables", "dynamic dispatch remain post-v1"} {
		if !strings.Contains(protocolMVP.Scope+" "+protocolMVP.Stability, want) {
			t.Fatalf("protocol conformance MVP feature missing %q boundary: %#v", want, protocolMVP)
		}
	}
	callableMVP := seenFeature["language.callable-mvp"]
	for _, want := range []string{"Level 0 callable surface", "legacy ptr closure local direct calls", "captured closure escape", "full first-class function values remain out of scope"} {
		if !strings.Contains(callableMVP.Scope+" "+callableMVP.Stability, want) {
			t.Fatalf("callable MVP feature missing %q boundary: %#v", want, callableMVP)
		}
	}
	stdlibCore := seenFeature["stdlib.core-current"]
	for _, want := range []string{"executable HTTP/1.1 String and byte-buffer request-line routing, request-head framing, and response byte-buffer helpers", "classify TechEmpower request lines from String text or caller-owned byte buffers", "locate CRLFCRLF request-head boundaries for pipelined buffers", "executable JSON byte-buffer response helpers", "caller-owned buffers", "networking exposes deterministic endpoint policy helpers", "executable Linux TCP socket client/server I/O helpers with recv/send, SO_REUSEPORT, TCP_NODELAY, nonblocking accept convenience, and epoll add/mod/delete plus wait-one readiness flag capture and predicates", "net socket open/bind/connect/listen/accept/read/recv/write/send/nonblocking/close plus SO_REUSEPORT, TCP_NODELAY, SOCK_NONBLOCK/SOCK_CLOEXEC accept helpers, and epoll create/add-read/add-read-write/mod-read/mod-read-write/delete/wait-one/wait-one-into helpers with EPOLLIN/EPOLLOUT/EPOLLERR/EPOLLHUP predicates are host-backed on linux-x64", "stable generic collection source views", "lib.core.collections.Vec<T>", "HashMap<K,V>", "hash_map_get_i32_i32_or", "hash_map_get_u8_i32_or", "no hidden allocator", "generic hashing/equality protocol", "production runtime map/vector claim", "C++/Rust parity"} {
		if !strings.Contains(stdlibCore.Scope+" "+stdlibCore.Stability, want) {
			t.Fatalf("stdlib core feature missing %q boundary: %#v", want, stdlibCore)
		}
	}
	for _, want := range []string{"P19.2 HTTP/JSON source-first evidence", "lib.core.http request-head framing", "pipelined local buffers", "lib.core.json message-object writers", "internal per-server UTC-second Date cache evidence", "Linux netrt.Writev/netrt.Sendfile helper evidence", "tetra.stdlib.http_json.production_stack.v1", "production HTTP server promotion", "source-level cached-date API", "cross-worker Date cache", "webrt.flush scatter/gather integration", "HTTP static-file sendfile path", "zero-copy production file-serving", "P20 performance matrix", "official TechEmpower result"} {
		if !strings.Contains(stdlibCore.Scope+" "+stdlibCore.Stability, want) {
			t.Fatalf("stdlib core feature missing P19.2 boundary %q: %#v", want, stdlibCore)
		}
	}
	for _, want := range []string{"P19.3 PostgreSQL source-first and local SCRAM evidence", "lib.core.postgres source rows", "DB single query", "DB multiple queries", "DB updates", "DB fortunes", "startup/SCRAM", "prepared statements", "binary int4 helpers", "pooling/backpressure", "borrowed DataRow decode", "checked local SCRAM benchmark reports", "tetra.stdlib.postgresql.production_driver.v1", "p19.3_postgres_source_first", "validate-techempower-report", "full source-level PostgreSQL driver API", "external production database deployment", "production database benchmark", "P20 performance matrix", "C++/Rust parity", "official TechEmpower result", "measured speed comparison", "runtime behavior change"} {
		if !strings.Contains(stdlibCore.Scope+" "+stdlibCore.Stability, want) {
			t.Fatalf("stdlib core feature missing P19.3 boundary %q: %#v", want, stdlibCore)
		}
	}
	callableLevel1 := seenFeature["language.callable-level1"]
	if callableLevel1.Since != "v0.4.0" {
		t.Fatalf("callable Level 1 since = %q, want v0.4.0", callableLevel1.Since)
	}
	for _, want := range []string{"production non-capturing symbol-backed callable Level 1", "function-typed locals", "target-set-backed function-typed parameter aliases", "function-typed parameter storage into struct fields with direct field calls or synchronous callback arguments", "function-typed parameter storage into enum payloads with direct payload calls, reassignment, returned enum propagation, or synchronous callback arguments", "callbacks", "optional argument labels on function-typed value calls including captured fnptr locals with mixed labeled/unlabeled lists rejected", "symbol-backed function-typed globals for same-module or namespace/selective imported public direct calls plus local initialization/reassignment", "non-capturing closure-literal function-typed globals", "same-module mutable global reassignment with direct calls, synchronous callback arguments, function-typed returns, generated .t4i function-typed parameter local-alias return metadata, and local or nested local struct-field/enum-payload storage/reassignment/returned-aggregate propagation", "imported mutable function-typed global boundary diagnostics", "actor/task boundary diagnostics across core.spawn, core.task_spawn_i32, core.task_spawn_i32_typed, core.task_spawn_group_i32, and core.task_spawn_group_i32_typed", "pass same-module or imported direct function-typed return-call callback arguments whose returned targets or multi-return target sets touch mutable globals, preserve that classification through local/field alias returns and returned struct/enum aggregate fields or payloads across module boundaries", "imported immutable function-typed globals whose targets touch mutable globals", "symbol-backed function-typed global initializers", "non-capturing generic closure literal binding/direct callback/return/mutable local or nested struct field reassignment/nested struct field initializer/enum payload initializer or reassignment", "inferable same-module or imported generic symbols", "function-typed returns", "target-set-backed function-typed parameter returns", "mutable local and nested struct field reassignment", "nested struct field initializers", "enum payload initializers", "signature-compatible mutable local reassignment", "captured closure escape beyond the fnptr Level 2 slice", "full first-class function values remain out of scope"} {
		if !strings.Contains(callableLevel1.Scope+" "+callableLevel1.Stability, want) {
			t.Fatalf("callable Level 1 feature missing %q boundary: %#v", want, callableLevel1)
		}
	}
	stdlibMirrors := seenFeature["stdlib.experimental-mirrors"]
	if stdlibMirrors.Since != "v0.4.0" {
		t.Fatalf("stdlib experimental mirrors since = %q, want v0.4.0", stdlibMirrors.Since)
	}
	for _, want := range []string{"production compatibility mirrors", "forward to lib.core", "stable callers should import lib.core"} {
		if !strings.Contains(stdlibMirrors.Scope+" "+stdlibMirrors.Stability, want) {
			t.Fatalf("stdlib mirrors feature missing %q boundary: %#v", want, stdlibMirrors)
		}
	}
	ownershipMVP := seenFeature["language.ownership-markers-mvp"]
	for _, want := range []string{"conservative borrow/inout/consume marker checks", "same-module/cross-module struct-field and enum-payload partial consume with whole-value call/let/return and enum wrapper-constructor rejection", "use-after-consume", "borrow escape diagnostics for scalar ptr including same-module/cross-module scalar ptr consume and inout assignment plus match/catch-expression return escapes and typed-error throw ptr/region payload escapes", "same-module/cross-module borrowed scalar ptr escapes through ptr-containing struct inout assignment", "same-module/cross-module fixed-array alias return plus direct global assignment, optional global assignment, and inout assignment escapes with stable TETRA2102 diagnostic evidence", "borrowed string alias return/global assignment escapes", "ptr/slice optional assignment return/owned/consume/inout escape", "slice optional payload binding owned/consume/inout call, inout-assignment, and global assignment escapes", "same-module/cross-module direct slice global assignment with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module optional ptr global assignment with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module optional aggregate global assignment with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module ptr optional assignment if-let/match global escape with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module ptr enum alias return escape with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module ptr-containing aggregate whole/field/alias/nested-field return escapes with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module whole-aggregate global assignment with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module ptr-containing enum whole-value global assignment with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module global field target assignment with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module aggregate and nested-aggregate global field escapes with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module ptr-containing and nested ptr-containing aggregates plus ptr-containing enum aggregates including whole-aggregate, whole-enum, global field target, and global field escapes", "optional ptr payloads including same-module/cross-module whole-optional use-after-payload-consume diagnostics", "same-module/cross-module optional-payload whole-value rejection after payload consume/free with stable TETRA2101 JSON diagnostic evidence", "same-module/cross-module ptr enum-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module ptr optional-payload return/global/inout assignment escapes with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module slice optional-payload inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module nested slice enum-payload return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module nested slice struct return/inout/global assignment escapes with stable TETRA2102 JSON diagnostic evidence", "same-module/cross-module pattern-bound enum payload and if-let/match optional payload return, owned/consume/inout call, inout-assignment, and global escapes", "same-module/cross-module ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence", "same-module/cross-module ptr enum-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence", "same-module/cross-module ptr optional-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence", "same-module/cross-module slice optional-payload owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence", "same-module/cross-module generic aggregate and optional-ptr owned/consume/inout instantiations including slice-containing struct/enum aggregate instantiations with stable TETRA2101 CLI JSON evidence", "same-module/cross-module generic borrow-aggregate/optional-ptr return diagnostics with stable TETRA2102 CLI JSON evidence", "same-module/cross-module protocol parameter ownership matching plus same-module/cross-module protocol impl parameter ownership mismatch diagnostics with stable TETRA2001 CLI JSON evidence", "same-module/cross-module generic protocol requirement parameter ownership mismatch diagnostics with stable TETRA2001 JSON diagnostic evidence", "same-module/cross-module function-typed value/struct-field/enum-payload optional-ptr owned/consume/inout callback diagnostics with stable TETRA2101 CLI JSON evidence", "function-typed value/struct-field/enum-payload callback slice-containing struct/enum owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence", "imported direct ptr-containing/nested aggregate owned/consume/inout call rejections with stable TETRA2101 JSON diagnostic evidence", "not a full SSA lifetime solver"} {
		if !strings.Contains(ownershipMVP.Scope+" "+ownershipMVP.Stability, want) {
			t.Fatalf("ownership markers MVP feature missing %q boundary: %#v", want, ownershipMVP)
		}
	}
	resourceMVP := seenFeature["language.resource-lifetime-mvp"]
	for _, want := range []string{"conservative resource finalization checks", "task handles", "branch/match/loop task-handle maybe-joined, task-group maybe-closed, and island maybe-freed merge diagnostics; branch/match/loop resource finalization merge diagnostics with stable TETRA2101 JSON evidence", "stable ownership safety JSON diagnostics for resource use-after-free, double-join, and ambiguous-provenance cases", "same-module/cross-module struct-field and enum-payload alias use-after-free with stable TETRA2101 JSON diagnostic evidence", "island handles", "same-module/cross-module task-handle/task-group struct-field/enum-payload join/close aliases", "same-module/cross-module task-handle struct-field/enum-payload alias join diagnostics with stable TETRA2101 JSON diagnostic evidence", "same-module/cross-module task-group struct-field/enum-payload alias close diagnostics with stable TETRA2101 JSON diagnostic evidence", "same-module/cross-module enum-constructor return resource aliases with stable TETRA2101 CLI JSON evidence", "same-module typed-error throw/catch and rethrow-through-try enum-payload resource aliases with stable TETRA2101 JSON diagnostic evidence", "generated .t4i direct/local/aggregate-local-alias/aggregate-field-access/aggregate-field-local-alias resource return, assignment/let/direct-if-let/direct-match/field-local/if-let/match optional and nested/field-local nested optional resource return, typed-error direct/field-local-alias throw, and rethrow-through-try direct/field-local-alias provenance stubs", "same-module/cross-module monomorphized generic struct task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence", "enum-payload", "if-let/match optional-payload return aliases including nested struct-field and enum-payload wrappers", "same-module/cross-module task-handle/task-group if-let/match optional-payload join/close aliases with stable TETRA2101 CLI JSON evidence", "same-module/cross-module island whole-optional use-after-payload-free diagnostics", "same-module/cross-module transitive interprocedural task-handle/task-group/island resource aliases with stable TETRA2101 CLI JSON evidence", "double-use", "ambiguous provenance", "not a full SSA lifetime solver"} {
		if !strings.Contains(resourceMVP.Scope+" "+resourceMVP.Stability, want) {
			t.Fatalf("resource lifetime MVP feature missing %q boundary: %#v", want, resourceMVP)
		}
	}
	transferMVP := seenFeature["actors.task-transfer-safety"]
	for _, want := range []string{"conservative actor/task ownership transfer checks", "worker entrypoints", "branch/match/loop actor consume reuse diagnostics with stable TETRA2101 CLI JSON evidence", "actor/task use-after-transfer diagnostics with stable TETRA2101 CLI JSON evidence", "island transfer non-local-payload rejection with stable TETRA2101 CLI JSON evidence", "same-module/cross-module transitive actor consume alias diagnostics with stable TETRA2101 CLI JSON evidence", "same-module/cross-module monomorphized generic struct actor consume alias diagnostics with stable TETRA2101 CLI JSON evidence", "same-module/cross-module task_group_cancel return provenance diagnostics with stable TETRA2101 CLI JSON evidence", "same-module/cross-module actor if-let/match optional-payload, struct-field, and enum-payload consume alias diagnostics", "same-module/cross-module actor struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence", "same-module/cross-module actor/task if-let/match optional-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence", "same-module/cross-module task-handle struct-field/enum-payload alias transfer diagnostics with stable TETRA2101 JSON diagnostic evidence", "same-module/cross-module task-handle struct-field/enum-payload alias join diagnostics with stable TETRA2101 JSON diagnostic evidence", "cooperative task_group_cancel", "conservative local MVP", "distributed actors"} {
		if !strings.Contains(transferMVP.Scope+" "+transferMVP.Stability, want) {
			t.Fatalf("actor/task transfer feature missing %q boundary: %#v", want, transferMVP)
		}
	}
	lifetimeSSA := seenFeature["language.lifetime-ssa"]
	if lifetimeSSA.Since != "v0.4.0" {
		t.Fatalf("lifetime SSA since = %q, want v0.4.0", lifetimeSSA.Since)
	}
	for _, want := range []string{"production SSA-like local lifetime join analysis", "ownership consume state", "resource finalization state", "optional region-wrapper escapes", "same-module and interface-only cross-module per-field interprocedural region summaries", "optional aggregate wrappers", "enum payload wrappers", "branch aggregate wrappers", "match aggregate wrappers", "if-let aggregate wrappers", "mixed safe/provenance aggregate branch and match returns", "optional mixed safe/provenance aggregate branch merges", "maybe-consumed diagnostics", "richer interprocedural lifetime proofs"} {
		if !strings.Contains(lifetimeSSA.Scope+" "+lifetimeSSA.Stability, want) {
			t.Fatalf("lifetime SSA feature missing %q boundary: %#v", want, lifetimeSSA)
		}
	}
	callableLevel2 := seenFeature["language.callable-level2"]
	if callableLevel2.Since != "v0.4.0" {
		t.Fatalf("callable Level 2 since = %q, want v0.4.0", callableLevel2.Since)
	}
	for _, want := range []string{"production captured closure Level 2 slice", "fnptr-backed function-typed locals", "captured ptr closure aliases into function-typed locals, mutable function-typed local reassignment", "same-module mutable function-typed global snapshot reassignment", "direct synchronous callback arguments including direct closure literals passed to imported callbacks", "function-typed returns including direct return of let-bound captured ptr closure values", "direct closure-literal container initializers in module-aware lowering", "direct calls including labeled direct calls on captured ptr closures", "imported parameter-return callbacks", "up to eight by-value snapshot environment slots", "cross-module returned captured closures used through locals or direct callback arguments", "cross-module struct-parameter function-field dispatch including namespace/selective imported direct struct constructors carrying closure literals or captured ptr closure locals", "cross-module enum-parameter function-payload dispatch including direct namespace/selective imported enum constructor arguments", "immutable local struct fields or enum payloads", "larger immutable environments are promoted under language.full-first-class-callables"} {
		if !strings.Contains(callableLevel2.Scope+" "+callableLevel2.Stability, want) {
			t.Fatalf("callable Level 2 feature missing %q boundary: %#v", want, callableLevel2)
		}
	}
	fullCallables := seenFeature["language.full-first-class-callables"]
	if fullCallables.Since != "v0.4.0" {
		t.Fatalf("full first-class callables since = %q, want v0.4.0", fullCallables.Since)
	}
	for _, want := range []string{"production first-class callable/function-value semantics", "bounded fnptr fast path", "fixed 4-slot callable handle", "larger immutable Int/Bool/String/simple-aggregate captures", "local storage", "mutable local reassignment", "returns", "same-module global snapshots", "struct fields", "enum payloads", "synchronous callback arguments", "cross-module returned values", "aliases", "generated .t4i function-typed parameter local-alias return metadata", "generated .t4i metadata", "stable JSON diagnostics for mutable by-reference captures including callable mutable-capture global-escape", "callable mutable-capture heap-escape", "callable pointer/resource capture escape", "function-typed storage/return unsupported capture rejection", "captured callable/function-typed parameter global-storage escape", "unsupported function-value escape outside the fnptr ABI", "unsupported function-value call", "capturing closure raw-ptr escape", "captured closure explicit type-arg rejection", "function-typed explicit type-arg rejection", "generic closure capture and generic callback-closure capture rejection", "generic closure pointer/direct-call rejection", "imported mutable function-typed global boundary", "thread-boundary callable escape"} {
		if !strings.Contains(fullCallables.Scope+" "+fullCallables.Stability, want) {
			t.Fatalf("full first-class callable feature missing %q boundary: %#v", want, fullCallables)
		}
	}
	enumFeature := seenFeature["language.enum-payload-match"]
	if enumFeature.Since != "v0.3.0" {
		t.Fatalf("enum payload feature since = %q, want v0.3.0", enumFeature.Since)
	}
	for _, want := range []string{"positional enum payload constructors", "match/catch/if-let", "exhaustive unguarded enum match/catch", "advanced ADT constructors", "nested destructuring patterns", "guard expansion remain future/post-v1"} {
		if !strings.Contains(enumFeature.Scope+" "+enumFeature.Stability, want) {
			t.Fatalf("enum payload feature missing %q boundary: %#v", want, enumFeature)
		}
	}
	protocolBoundGenerics := seenFeature["language.protocol-bound-generics-static"]
	if protocolBoundGenerics.Since != "v0.3.0" {
		t.Fatalf("protocol-bound generics since = %q, want v0.3.0", protocolBoundGenerics.Since)
	}
	for _, want := range []string{"validated statically during monomorphization", "same-module and cross-module impl conformance with parameter ownership markers", "visibility diagnostics", "calling protocol requirements through generic bounds", "witness tables", "dynamic dispatch remain unsupported"} {
		if !strings.Contains(protocolBoundGenerics.Scope+" "+protocolBoundGenerics.Stability, want) {
			t.Fatalf("protocol-bound generics feature missing %q boundary: %#v", want, protocolBoundGenerics)
		}
	}
	effectsMVP := seenFeature["safety.effects-mvp"]
	if effectsMVP.Since != "v0.3.0" {
		t.Fatalf("effects MVP since = %q, want v0.3.0", effectsMVP.Since)
	}
	for _, want := range []string{"stable uses effect names and groups", "transitive call propagation", "missing uses declarations are diagnostics", "checker-enforced optimizer facts", "pure/no-alloc/no-mem-write/no-actor-send/no-unknown-escape", "no effect inference"} {
		if !strings.Contains(effectsMVP.Scope+" "+effectsMVP.Stability, want) {
			t.Fatalf("effects MVP feature missing %q boundary: %#v", want, effectsMVP)
		}
	}
	capabilitiesMVP := seenFeature["safety.capabilities-mvp"]
	if capabilitiesMVP.Since != "v0.3.0" {
		t.Fatalf("capabilities MVP since = %q, want v0.3.0", capabilitiesMVP.Since)
	}
	for _, want := range []string{"cap.io and cap.mem opaque tokens", "unsafe blocks", "raw memory/MMIO", "capsule permissions", "not a broad safe-code capability construction model"} {
		if !strings.Contains(capabilitiesMVP.Scope+" "+capabilitiesMVP.Stability, want) {
			t.Fatalf("capabilities MVP feature missing %q boundary: %#v", want, capabilitiesMVP)
		}
	}
	privacyMVP := seenFeature["safety.privacy-consent-mvp"]
	if privacyMVP.Since != "v0.3.0" {
		t.Fatalf("privacy/consent MVP since = %q, want v0.3.0", privacyMVP.Since)
	}
	for _, want := range []string{"uses privacy requires privacy", "secret.i32/SecretInt", "consent token", "not cryptographic isolation", "distributed consent enforcement remains post-v1"} {
		if !strings.Contains(privacyMVP.Scope+" "+privacyMVP.Stability, want) {
			t.Fatalf("privacy/consent MVP feature missing %q boundary: %#v", want, privacyMVP)
		}
	}
	budgetMVP := seenFeature["safety.budget-mvp"]
	if budgetMVP.Since != "v0.3.0" {
		t.Fatalf("budget MVP since = %q, want v0.3.0", budgetMVP.Since)
	}
	for _, want := range []string{"budget(<non-negative integer constant>)", "uses budget", "deterministic budget guard instructions", "not cross-function runtime-wide", "distributed budget enforcement remains post-v1"} {
		if !strings.Contains(budgetMVP.Scope+" "+budgetMVP.Stability, want) {
			t.Fatalf("budget MVP feature missing %q boundary: %#v", want, budgetMVP)
		}
	}
	safetyCore := seenFeature["safety.production-core"]
	if safetyCore.Since != "v0.4.0" {
		t.Fatalf("safety production core since = %q, want v0.4.0", safetyCore.Since)
	}
	for _, want := range []string{"production local safety model", "ownership/lifetime/borrow/consume/inout", "resource finalization", "callable escape diagnostics", "effects/capabilities/privacy/consent/budget", "unsafe boundaries", "actor/task transfer safety", "pointer/MMIO/memory capability gates", "memory production final audit", "explicit diagnostics"} {
		if !strings.Contains(safetyCore.Scope+" "+safetyCore.Stability, want) {
			t.Fatalf("safety production core missing %q boundary: %#v", want, safetyCore)
		}
	}
	uiMetadata := seenFeature["ui.metadata-v1"]
	if uiMetadata.Status != compiler.FeatureStatusCurrent || uiMetadata.Since != "v0.4.0" {
		t.Fatalf("ui.metadata-v1 lifecycle = status %q since %q, want current since v0.4.0", uiMetadata.Status, uiMetadata.Since)
	}
	for _, want := range []string{"production UI metadata contract", "deterministic tetra.ui.v0.4.0 JSON", "browser-backed web command-dispatch runtime", "wasm32-web command dispatch", "post-v0.4 Web UI runtime smoke", "native shell command dispatch", "widget-tree traces", "JSON trace sidecars", "style metadata preview attributes", "accessibility metadata preview attributes"} {
		if !strings.Contains(uiMetadata.Scope+" "+uiMetadata.Stability, want) {
			t.Fatalf("UI metadata feature missing %q boundary: %#v", want, uiMetadata)
		}
	}
	uiToolkit := seenFeature["ui.toolkit-core"]
	if uiToolkit.Status != compiler.FeatureStatusCurrent || uiToolkit.Since != "v0.4.0" {
		t.Fatalf("ui.toolkit-core lifecycle = status %q since %q, want current since v0.4.0", uiToolkit.Status, uiToolkit.Since)
	}
	for _, want := range []string{"production platform-independent UI Toolkit Core contract", "tetra.ui.toolkit.v1", "widget model", "layout model", "accessibility model", "event dispatch", "state binding/update", "runtime trace artifacts", "metadata-only", "runtime-less", "native-shell sidecar-only", "web-only", "GTK/Qt/OS platform backend production", "full cross-platform UI"} {
		if !strings.Contains(uiToolkit.Scope+" "+uiToolkit.Stability, want) {
			t.Fatalf("UI toolkit core feature missing %q boundary: %#v", want, uiToolkit)
		}
	}
	for _, wantDoc := range []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_toolkit_core.md", "docs/spec/ui_v0.4.0.md"} {
		found := false
		for _, doc := range uiToolkit.Docs {
			if doc == wantDoc {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("ui.toolkit-core missing doc %s: %#v", wantDoc, uiToolkit.Docs)
		}
	}
	distributedActors := seenFeature["actors.distributed-runtime"]
	if distributedActors.Status != compiler.FeatureStatusCurrent || distributedActors.Since != "v0.4.0" {
		t.Fatalf("distributed actors lifecycle = status %q since %q, want current since v0.4.0", distributedActors.Status, distributedActors.Since)
	}
	for _, want := range []string{"production Linux-x64 distributed actor runtime path", "actornet loopback TCP broker", "distributed node identity", "remote actor handles", "network mailbox send/receive", "i32, tagged, and typed frames", "missing-node failure/status propagation", "task cancel/join handles", "tetra.actors.distributed-runtime.v1 smoke evidence", "tetra.actor.production_foundation.v1", "actor-runtime-foundation-linux-x64-gate.sh", "transport-only or fake reports", "non-Linux-x64 targets", "non-Linux distributed runtime", "distributed zero-copy", "cluster membership", "reconnect/retry production", "formal race proof", "broader structured-concurrency guarantees"} {
		if !strings.Contains(distributedActors.Scope+" "+distributedActors.Stability, want) {
			t.Fatalf("distributed actors feature missing %q boundary: %#v", want, distributedActors)
		}
	}
	uiRuntime := seenFeature["ui.native-runtime"]
	if uiRuntime.Status != compiler.FeatureStatusCurrent || uiRuntime.Since != "v0.4.0" {
		t.Fatalf("UI native runtime lifecycle = status %q since %q, want current since v0.4.0", uiRuntime.Status, uiRuntime.Since)
	}
	for _, want := range []string{"production Linux-x64 native UI runtime path", "native runtime widget instances", "click/activate events", "lowered command operations", "state and widget updates", "tetra.ui.native-runtime.v1 smoke evidence", "metadata-only", "web-only", "native-shell sidecar-only", "macOS/Windows", "platform accessibility integration"} {
		if !strings.Contains(uiRuntime.Scope+" "+uiRuntime.Stability, want) {
			t.Fatalf("UI runtime feature missing %q boundary: %#v", want, uiRuntime)
		}
	}
	platformUI := seenFeature["ui.platform-runtime"]
	if platformUI.Status != compiler.FeatureStatusExperimental || platformUI.Since != "v0.4.0" {
		t.Fatalf("UI platform runtime lifecycle = status %q since %q, want experimental since v0.4.0", platformUI.Status, platformUI.Since)
	}
	for _, want := range []string{"tetra.ui.platform-runtime.v1", "full-platform UI runtime promotion gate", "real Windows/macOS target-host reports", "not production until", "metadata-only", "runtime-less", "startup_failure"} {
		if !strings.Contains(platformUI.Scope+" "+platformUI.Stability, want) {
			t.Fatalf("UI platform runtime feature missing %q boundary: %#v", want, platformUI)
		}
	}
}

func TestFeatureRegistryCLICoreCoversDocumentedPublicCommands(t *testing.T) {
	var cliCore compiler.FeatureInfo
	for _, feature := range compiler.FeatureRegistry() {
		if feature.ID == "cli.core" {
			cliCore = feature
			break
		}
	}
	if cliCore.ID == "" {
		t.Fatal("feature registry missing cli.core")
	}

	scopeTokens := map[string]bool{}
	for _, token := range strings.FieldsFunc(cliCore.Scope, func(r rune) bool {
		return r == '/' || r == ',' || r == ' ' || r == ';'
	}) {
		scopeTokens[token] = true
	}
	for _, command := range []string{
		"check",
		"build",
		"run",
		"fmt",
		"test",
		"doc",
		"doctor",
		"targets",
		"features",
		"formats",
		"new",
		"interface",
		"project",
		"workspace",
		"smoke",
		"eco",
		"clean",
		"version",
		"lsp",
	} {
		if !scopeTokens[command] {
			t.Fatalf("cli.core scope missing documented public command %q: %q", command, cliCore.Scope)
		}
	}
}

func TestFeatureRegistryDeclaresSurfaceDirectionAndLegacyMetadataBoundary(t *testing.T) {
	byID := map[string]compiler.FeatureInfo{}
	for _, feature := range compiler.FeatureRegistry() {
		byID[feature.ID] = feature
	}

	wantCurrentSurface := map[string][]string{
		"ui.surface-core":             {"surface-v1-linux-web", "pure-Tetra UI", "Host ABI"},
		"ui.surface-headless":         {"release-test target", "deterministic"},
		"ui.surface-linux-x64":        {"linux-x64-release-window-v1", "Wayland shm"},
		"ui.surface-web-wasm":         {"wasm32-web-browser-canvas-release-v1", "browser canvas"},
		"ui.surface-component-model":  {"component-tree-api", "release subset"},
		"ui.surface-toolkit-v1":       {"production-widgets-v1", "TextBox", "Checkbox"},
		"ui.surface-text-input-v1":    {"production-text-input-v1", "clipboard", "composition"},
		"ui.surface-accessibility-v1": {"platform-bridge-v1", "supported targets"},
	}
	for id, wantPhrases := range wantCurrentSurface {
		feature, ok := byID[id]
		if !ok {
			t.Fatalf("feature registry missing %s", id)
		}
		if feature.Status != compiler.FeatureStatusCurrent {
			t.Fatalf("feature %s status = %q, want current Surface v1 release scope", id, feature.Status)
		}
		for _, want := range wantPhrases {
			if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
				t.Fatalf("feature %s missing %q in scope/stability: %#v", id, want, feature)
			}
		}
		if len(feature.Docs) == 0 || !hasFeatureDoc(feature.Docs, "docs/spec/surface_v1.md") {
			t.Fatalf("feature %s docs = %#v, want docs/spec/surface_v1.md", id, feature.Docs)
		}
	}

	blockSystem, ok := byID["ui.surface-block-system"]
	if !ok {
		t.Fatal("feature registry missing ui.surface-block-system")
	}
	if blockSystem.Status != compiler.FeatureStatusExperimental {
		t.Fatalf("ui.surface-block-system status = %q, want experimental", blockSystem.Status)
	}
	for _, want := range []string{
		"tetra.surface.block-system.gate.v1",
		"block_system.memory_budget",
		"reports/surface-block/p18-budget",
		"same-commit target evidence",
		"no production Block claim",
	} {
		if !strings.Contains(blockSystem.Scope+" "+blockSystem.Stability, want) {
			t.Fatalf("ui.surface-block-system missing P19 truth-boundary phrase %q: %#v", want, blockSystem)
		}
	}

	for _, id := range []string{
		"ui.surface-minimal-toolkit",
		"ui.surface-toolkit-reuse-v1",
		"ui.surface-accessibility-metadata-tree-v1",
	} {
		feature, ok := byID[id]
		if !ok {
			t.Fatalf("feature registry missing historical Surface feature %s", id)
		}
		if feature.Status == compiler.FeatureStatusCurrent {
			t.Fatalf("historical Surface feature %s must not be current: %#v", id, feature)
		}
		if !strings.Contains(feature.Scope+" "+feature.Stability, "absorbed by") &&
			!strings.Contains(feature.Scope+" "+feature.Stability, "internal layer") {
			t.Fatalf("historical Surface feature %s missing absorbed/internal note: %#v", id, feature)
		}
	}

	for _, id := range []string{
		"ui.surface-macos-x64",
		"ui.surface-windows-x64",
		"ui.surface-wasm32-wasi",
	} {
		feature, ok := byID[id]
		if !ok {
			t.Fatalf("feature registry missing unsupported Surface target %s", id)
		}
		if feature.Status != compiler.FeatureStatusUnsupported {
			t.Fatalf("feature %s status = %q, want unsupported", id, feature.Status)
		}
		if !strings.Contains(feature.Scope+" "+feature.Stability, "no production target evidence") {
			t.Fatalf("unsupported Surface target %s missing evidence boundary: %#v", id, feature)
		}
	}

	metadata := byID["ui.metadata-v1"]
	for _, want := range []string{
		"legacy metadata compatibility",
		"not the new Tetra Surface runtime",
	} {
		if !strings.Contains(metadata.Scope+" "+metadata.Stability, want) {
			t.Fatalf("ui.metadata-v1 missing legacy boundary %q: %#v", want, metadata)
		}
	}
}

func TestVerifiedTrackCitesMasterPlanAuditDocs(t *testing.T) {
	var verified compiler.FeatureInfo
	for _, feature := range compiler.FeatureRegistry() {
		if feature.ID == "compiler.verified-track" {
			verified = feature
			break
		}
	}
	if verified.ID == "" {
		t.Fatal("feature registry missing compiler.verified-track")
	}
	for _, want := range []string{
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
		"docs/audits/vectorization-v1.md",
		"docs/audits/pgo-lto-target-cpu-v1.md",
		"docs/audits/actor-runtime-production-boundary-v1.md",
		"docs/audits/typed-actor-ownership-transfer-v1.md",
		"docs/audits/per-core-scheduler-v1.md",
		"docs/audits/async-io-reactor-v1.md",
		"docs/audits/region-aware-stdlib-v1.md",
		"docs/audits/stable-generic-collections-v1.md",
	} {
		if !hasFeatureDoc(verified.Docs, want) {
			t.Fatalf("compiler.verified-track docs = %#v, want %s", verified.Docs, want)
		}
	}
}

func hasFeatureDoc(docs []string, want string) bool {
	for _, doc := range docs {
		if doc == want {
			return true
		}
	}
	return false
}

func TestFeatureRegistryReturnsDefensiveCopy(t *testing.T) {
	features := compiler.FeatureRegistry()
	features[0].ID = "mutated"
	features[0].Docs[0] = "mutated.md"
	fresh := compiler.FeatureRegistry()
	if fresh[0].ID == "mutated" || fresh[0].Docs[0] == "mutated.md" {
		t.Fatalf("FeatureRegistry did not return a defensive copy: %#v", fresh[0])
	}
}

func TestManifestBuiltinsExposeCanonicalSafetyEffectsAndPolicies(t *testing.T) {
	manifest, err := compiler.GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}

	allowedEffects := map[string]bool{
		"actors": true, "alloc": true, "budget": true, "capability": true,
		"control": true, "io": true, "islands": true, "link": true,
		"mem": true, "mmio": true, "privacy": true, "runtime": true, "surface": true,
	}
	for _, builtin := range manifest.Builtins {
		for _, effect := range builtin.Effects {
			if !allowedEffects[effect] {
				t.Fatalf("builtin %s exposes non-canonical effect %q in manifest", builtin.Name, effect)
			}
		}
	}

	byName := map[string]compiler.BuiltinManifest{}
	for _, builtin := range manifest.Builtins {
		byName[builtin.Name] = builtin
	}
	for name, want := range map[string]struct {
		effects      string
		unsafePolicy string
	}{
		"core.cap_io":          {effects: "capability,io", unsafePolicy: "always"},
		"core.cap_mem":         {effects: "capability,mem", unsafePolicy: "always"},
		"core.consent_token":   {effects: "privacy", unsafePolicy: "never"},
		"core.secret_seal_i32": {effects: "privacy", unsafePolicy: "never"},
	} {
		got, ok := byName[name]
		if !ok {
			t.Fatalf("manifest missing builtin %s", name)
		}
		if strings.Join(got.Effects, ",") != want.effects || got.UnsafePolicy != want.unsafePolicy {
			t.Fatalf("manifest builtin %s = effects=%q unsafe_policy=%q, want effects=%q unsafe_policy=%q", name, strings.Join(got.Effects, ","), got.UnsafePolicy, want.effects, want.unsafePolicy)
		}
	}

	island, ok := byName["core.island_make_i32"]
	if !ok {
		t.Fatalf("manifest missing builtin core.island_make_i32")
	}
	if island.UnsafePolicy != "conditional" || !strings.Contains(island.UnsafeDetails, "requires unsafe") {
		t.Fatalf("manifest builtin core.island_make_i32 = %#v", island)
	}
}
