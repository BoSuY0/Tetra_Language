package compiler_test

import (
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/opt"
	"tetra_language/compiler/internal/testkit"
)

// ---- features_test.go ----

func TestFeatureRegistryCoversReleaseStatusesAndKeyBoundaries(t *testing.T) {
	features := compiler.FeatureRegistry()
	if len(features) == 0 {
		t.Fatal("FeatureRegistry returned no entries")
	}
	seenStatus := map[compiler.FeatureStatus]bool{}
	seenID := map[string]compiler.FeatureStatus{}
	seenFeature := map[string]compiler.FeatureInfo{}
	for _, feature := range features {
		if feature.ID == "" || feature.Name == "" || feature.Scope == "" ||
			feature.Stability == "" {
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
	for _, status := range []compiler.FeatureStatus{
		compiler.FeatureStatusCurrent,
		compiler.FeatureStatusPlanned,
		compiler.FeatureStatusPostV1,
	} {
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
	for _, want := range []string{
		"statically monomorphized",
		"tiny generic identity/wrapper",
		"no runtime generic values or dynamic dispatch",
		"generic structs",
		"future/post-v1",
	} {
		if !strings.Contains(genericsMVP.Scope+" "+genericsMVP.Stability, want) {
			t.Fatalf("generics MVP feature missing %q boundary: %#v", want, genericsMVP)
		}
	}
	safetyProductionCore := seenFeature["safety.production-core"]
	if !strings.Contains(
		safetyProductionCore.Scope+" "+safetyProductionCore.Stability,
		"memory cost model",
	) {
		t.Fatalf(
			"safety production core missing memory cost model boundary: %#v",
			safetyProductionCore,
		)
	}
	if !strings.Contains(
		safetyProductionCore.Scope+" "+safetyProductionCore.Stability,
		"memory fuzz oracle",
	) {
		t.Fatalf(
			"safety production core missing memory fuzz oracle boundary: %#v",
			safetyProductionCore,
		)
	}
	if !strings.Contains(
		safetyProductionCore.Scope+" "+safetyProductionCore.Stability,
		"memory production final audit",
	) {
		t.Fatalf(
			"safety production core missing memory production final audit boundary: %#v",
			safetyProductionCore,
		)
	}
	hasMemoryCostModelDoc := false
	hasMemoryFuzzOracleDoc := false
	hasMemoryProductionFinalDoc := false
	hasMemoryProductionArtifactMapDoc := false
	hasMemoryProductionNonclaimsDoc := false
	for _, doc := range safetyProductionCore.Docs {
		hasMemoryCostModelDoc = hasMemoryCostModelDoc ||
			doc == "docs/design/memory/memory_cost_model.md"
		hasMemoryFuzzOracleDoc = hasMemoryFuzzOracleDoc ||
			doc == "docs/audits/memory/islands/memory-fuzz-oracle-v1.md"
		hasMemoryProductionFinalDoc = hasMemoryProductionFinalDoc ||
			doc == "docs/audits/memory/production/memory-production-core-v1-final.md"
		hasMemoryProductionArtifactMapDoc = hasMemoryProductionArtifactMapDoc ||
			doc == "docs/audits/memory/production/memory-production-core-v1-artifact-map.md"
		hasMemoryProductionNonclaimsDoc = hasMemoryProductionNonclaimsDoc ||
			doc == "docs/audits/memory/production/memory-production-core-v1-nonclaims.md"
	}
	if !hasMemoryCostModelDoc {
		t.Fatalf("safety production core missing memory cost model doc: %#v", safetyProductionCore)
	}
	if !hasMemoryFuzzOracleDoc {
		t.Fatalf("safety production core missing memory fuzz oracle doc: %#v", safetyProductionCore)
	}
	if !hasMemoryProductionFinalDoc || !hasMemoryProductionArtifactMapDoc ||
		!hasMemoryProductionNonclaimsDoc {
		t.Fatalf(
			"safety production core missing MPC-16 final audit docs: %#v",
			safetyProductionCore,
		)
	}
	layoutABI := seenFeature["language.layout-abi-policy"]
	for _, want := range []string{
		"default structs",
		"do not promise C layout",
		"repr(C)",
		"ABI-locked",
		"unavailable for repr(C)",
		"default layout freedom v1",
		"p21.0_default_layout_freedom_v1",
		".layout.json schema_version 2",
		"compiler_owned_default",
		"abi_locked_repr_c",
		"exported_ffi_explicit_repr_c",
		(("public ABI/exported FFI " +
			"aggregate boundaries require ") +
			"explicit repr(C)"),
		"field_reordering",
		"padding_removal",
		"hot_cold_splitting",
		"scalar_replacement",
		"aos_to_soa",
		"no field reordering",
		"performance change",
		"runtime behavior change",
	} {
		if !strings.Contains(layoutABI.Scope+" "+layoutABI.Stability, want) {
			t.Fatalf("layout/ABI policy feature missing %q boundary: %#v", want, layoutABI)
		}
	}
	hasLayoutAuditDoc := false
	for _, doc := range layoutABI.Docs {
		hasLayoutAuditDoc = hasLayoutAuditDoc ||
			doc == "docs/audits/compiler/language/default-layout-freedom-v1.md"
	}
	if !hasLayoutAuditDoc {
		t.Fatalf("layout/ABI policy feature missing P21.0 audit doc: %#v", layoutABI)
	}
	abiVerification := seenFeature["compiler.abi-verification"]
	for _, want := range []string{
		"ABI verification v1",
		"tetra.abi.verification.v1",
		"p21.1_abi_verification",
		"linux-x64 SysV",
		"linux-x86 i386 SysV",
		"linux-x32 x32 SysV",
		"macos-x64 SysV",
		"windows-x64 Win64",
		"wasm32-wasi",
		"wasm32-web",
		"abi_test_corpus",
		"struct_enum_slice_string_return_validation",
		"call_boundary_validation",
		"ffi_repr_c_tests",
		"compiler-owned i32 slot ABI metadata",
		"IRCall arg/return slot matching",
		"no runtime execution claim",
		"no C ABI claim for default structs",
		"no native C aggregate ABI claim for wasm targets",
		"no performance claim",
		"no safe-program semantics change",
	} {
		if !strings.Contains(abiVerification.Scope+" "+abiVerification.Stability, want) {
			t.Fatalf("ABI verification feature missing %q boundary: %#v", want, abiVerification)
		}
	}
	hasABIAuditDoc := false
	for _, doc := range abiVerification.Docs {
		hasABIAuditDoc = hasABIAuditDoc ||
			doc == "docs/audits/compiler/backend/abi-verification-v1.md"
	}
	if !hasABIAuditDoc {
		t.Fatalf("ABI verification feature missing P21.1 audit doc: %#v", abiVerification)
	}
	featureSurfaceAudit := seenFeature["compiler.feature-surface-audit"]
	for _, want := range []string{
		"full feature surface audit",
		"tetra.language.feature_surface_audit.v1",
		"p22.0_full_feature_surface_audit",
		"first-class callables",
		"closures",
		"protocols/trait objects",
		"runtime generics",
		"advanced enums/pattern matching",
		"async typed errors",
		"structured concurrency",
		"modules/packages",
		"macros/metaprogramming",
		"UI/surface",
		"Eco/capsules",
		"FeatureRegistry statuses",
		"same-branch evidence",
		"no full v1 language guarantee",
		"runtime generic values",
		"trait objects",
		"runtime protocol values",
		"macro/metaprogramming system",
		"full structured concurrency",
		"cross-platform production UI runtime",
		"distributed EcoNet",
		"proof-carrying capsules",
		"performance claim",
		"runtime behavior change",
		"safe-program semantics change",
	} {
		if !strings.Contains(featureSurfaceAudit.Scope+" "+featureSurfaceAudit.Stability, want) {
			t.Fatalf("feature surface audit missing %q boundary: %#v", want, featureSurfaceAudit)
		}
	}
	hasFeatureSurfaceAuditDoc := false
	for _, doc := range featureSurfaceAudit.Docs {
		hasFeatureSurfaceAuditDoc = hasFeatureSurfaceAuditDoc ||
			doc == "docs/audits/compiler/language/full-feature-surface-audit-v1.md"
	}
	if !hasFeatureSurfaceAuditDoc {
		t.Fatalf("feature surface audit missing P22.0 audit doc: %#v", featureSurfaceAudit)
	}
	firstClassCallableEvidence := seenFeature["compiler.first-class-callables-v1"]
	for _, want := range []string{
		"first-class callables v1",
		"tetra.language.first_class_callables.v1",
		"p22.1_first_class_callables_v1",
		"bounded fnptr fast path",
		"fat callable handle",
		"capture safety classifier",
		"mutable capture escape diagnostics",
		"resource/thread escape diagnostics",
		"fixed ABI width",
		"cross-module interface metadata",
		"storage/callback paths",
		"one-capture 9-slot fnptr",
		"without heap environment allocation",
		"nine-capture fixed 4-slot handle",
		"IRAllocBytes",
		"IRMemWritePtrOffset",
		"IRMemReadPtrOffset",
		"ArgSlots 10 RetSlots 1",
		"generated .t4i metadata",
		"ReturnFunctionHandleValue",
		"heap escape kind",
		"ReturnSlots = 4",
		"no variable-width callable ABI",
		"exploding return slots",
		"mutable by-reference capture support",
		"pointer/resource capture support",
		"thread-boundary callable transfer",
		"runtime generic callable polymorphism",
		"dynamic callable dispatch",
		"unsafe lifetime relaxation",
		"performance claim",
		"runtime behavior change",
		"safe-program semantics change",
	} {
		if !strings.Contains(
			firstClassCallableEvidence.Scope+" "+firstClassCallableEvidence.Stability,
			want,
		) {
			t.Fatalf(
				"first-class callable evidence feature missing %q boundary: %#v",
				want,
				firstClassCallableEvidence,
			)
		}
	}
	hasFirstClassCallableAuditDoc := false
	for _, doc := range firstClassCallableEvidence.Docs {
		hasFirstClassCallableAuditDoc = hasFirstClassCallableAuditDoc ||
			doc == "docs/audits/compiler/language/first-class-callables-v1.md"
	}
	if !hasFirstClassCallableAuditDoc {
		t.Fatalf(
			"first-class callable evidence missing P22.1 audit doc: %#v",
			firstClassCallableEvidence,
		)
	}
	protocolTraitDecision := seenFeature["compiler.protocol-trait-object-decision"]
	for _, want := range []string{
		"protocol / trait object decision",
		"tetra.language.protocol_trait_object_decision.v1",
		"p22.2_protocol_trait_object_decision",
		"keep_static_conformance_only",
		"static conformance fast path",
		"static protocol-bound generics",
		"runtime existential decision",
		"explicit dynamic-dispatch gate",
		"specialization static abstraction",
		"witness-table boundary",
		"trait-object boundary",
		"registry/docs alignment",
		"Vec2.draw IRCall",
		"id__T_Vec2 direct call",
		"unknown type 'Drawable'",
		"generic-bound requirement-call rejection",
		"P17/P21 known-direct specialization evidence",
		"runtime protocol values",
		"trait objects",
		"witness tables",
		"dynamic dispatch",
		"conformance-table lookup",
		"runtime existential ABI",
		"broad protocol specialization",
		"performance",
		"runtime behavior change",
		"safe-program semantics change",
	} {
		if !strings.Contains(
			protocolTraitDecision.Scope+" "+protocolTraitDecision.Stability,
			want,
		) {
			t.Fatalf(
				"protocol/trait decision feature missing %q boundary: %#v",
				want,
				protocolTraitDecision,
			)
		}
	}
	hasProtocolTraitDecisionDoc := false
	for _, doc := range protocolTraitDecision.Docs {
		hasProtocolTraitDecisionDoc = hasProtocolTraitDecisionDoc ||
			doc == "docs/audits/compiler/language/protocol-trait-object-decision-v1.md"
	}
	if !hasProtocolTraitDecisionDoc {
		t.Fatalf("protocol/trait decision missing P22.2 audit doc: %#v", protocolTraitDecision)
	}
	verifiedTrack := seenFeature["compiler.verified-track"]
	for _, want := range []string{
		"differential scalar-i32",
		"source interpreter",
		"stack backend",
		"register backend",
		"optimized backend",
		"optimizer pass contract v1",
		"input/output verifier evidence",
		("proof preservation or " +
			"invalidation rules"),
		"translation validation hooks",
		"stable report rows",
		"negative-test markers",
		"optimizer core coverage v1",
		"bounded evidence-backed P17.1 closure",
		(("narrow safe const-" +
			"denominator div_i32/mod_i32 constant ") +
			"folding plus same-local comparison algebraic simplification"),
		"narrow SCCP constant-condition",
		(("known-local and stored " +
			"safe unary neg_i32 plus safe ") +
			"constant-expression facts including safe const-denominator " +
			"div_i32/mod_i32"),
		("constant unary neg_i32 and binary-expression branch folding " +
			"including safe const-denominator div_i32/mod_i32 with unary " +
			"min-int and denominator 0 and -1 rejected"),
		(("immediate and forward-terminated " +
			"single-predecessor label ") +
			"propagation plus folded zero-branch target propagation for " +
			"labels with one incoming edge and no fallthrough predecessor"),
		(("folded nonzero-branch " +
			"fallthrough propagation through ") +
			"immediate labels with no explicit incoming branch/jump edges"),
		(("dynamic load-local zero-" +
			"target and nonzero-fallthrough path ") +
			"facts"),
		"fallthrough-predecessor rejection",
		("explicit-incoming fallthrough-label " +
			"rejection"),
		"fallthrough pruning",
		(("narrow Stack IR " +
			"adjacent and stack-neutral separated ") +
			"single-assignment mem2reg temp promotion"),
		(("bounded comparison-expression, safe " +
			"const unary neg_i32, ") +
			"safe known-local unary neg_i32, safe const " +
			"add_i32/sub_i32/mul_i32 arithmetic, safe known-local " +
			"add_i32/sub_i32/mul_i32 arithmetic, safe const-denominator " +
			"div_i32/mod_i32 producer temps, and safe known-local " +
			"div_i32/mod_i32 producer temps"),
		(("unary min-int, arithmetic overflow, source-local " +
			"mutation, ") +
			"and denominator 0 and -1 rejected"),
		("bounded DCE for simple dead local stores, non-trapping " +
			"comparison-expression stores, safe known-local unary " +
			"neg_i32 stores, safe known-local add_i32/sub_i32/mul_i32 " +
			"stores, safe const-denominator div_i32/mod_i32 stores, and " +
			"safe known-local div_i32/mod_i32 stores"),
		(("narrow exact/commutative/mirrored-" +
			"comparison local-load, ") +
			"local-load/constant, unary local neg_i32, safe known-local " +
			"unary neg_i32 value, safe known-local " +
			"add_i32/sub_i32/mul_i32 value, safe known-local cmp_*_i32 " +
			"value, safe known-local div_i32/mod_i32 value, and safe " +
			"const-denominator div_i32/mod_i32 CSE/GVN"),
		(("commutative add/mul/eq/ne and mirrored " +
			"lt/gt/le/ge operand ") +
			"canonicalization"),
		("narrow proof-tagged LICM pure invariant comparison, " +
			"add/sub/mul arithmetic, known-local add_i32/sub_i32/mul_i32 " +
			"left-or-right operand hoisting, known-local cmp_*_i32 " +
			"left-or-right operand hoisting, safe const-denominator " +
			"div_i32/mod_i32 hoisting, and safe known-local " +
			"div_i32/mod_i32 denominator hoisting"),
		"bounded hot-loop shape evidence",
		"scalar sum",
		"scalar constant-stride sum",
		"scalar sum-of-squares",
		("scalar product " +
			"reduction bounded to product *= index + 1"),
		"scalar branchy max reduction",
		("scalar affine sum with " +
			"compile-time scale and bias 1..127"),
		"scalar countdown",
		"proof-tagged slice sum",
		("proof-tagged slice " +
			"constant-stride sum"),
		"call-loop machine IR",
		"inlining specialization coverage v1",
		"P17.2 target rows",
		("monomorphized generic " +
			"identity/wrapper"),
		"small-pure inline-small-pure",
		"payload enum known-case match",
		"proven-some optional match",
		"sccp-constant-branch evidence",
		(("statically checked " +
			"protocol/conformance direct-call ") +
			"inline-small-pure evidence"),
		("statically resolved extension-call inline-small-pure " +
			"evidence"),
		"inlined/not_inlined report reasons",
		"8-instruction body cap",
		"constant_stack_store tag tracking",
		("known direct Stack IR " +
			"function symbol boundaries"),
		"protocol-bound requirement calls",
		"witness tables",
		"trait objects",
		"runtime protocol values",
		"conformance-table lookup",
		"vectorization coverage v1",
		"P17.3 initial target rows",
		("proof-tagged sum []i32 " +
			"candidate recognition"),
		"range-proof evidence",
		("noalias-not-required " +
			"read-only reduction evidence"),
		("safe unaligned i32x4 " +
			"vector backend lowering"),
		"vector-i32x4-slice-sum-plan",
		"linux-x64 native SIMD lowering",
		"scalar tail handling",
		"scalar-i32-slice-sum fallback",
		"translation/differential validation",
		("proof-tagged copy []u8 " +
			"vector backend lowering"),
		"vector-u8x16-copy-plan",
		"noalias required source/dest disjoint",
		"safe unaligned u8x16 load/store",
		"scalar-u8-copy fallback",
		("linux-x64 native SIMD " +
			"lowering for proof-tagged copy []u8"),
		(("copy []u8 translation/" +
			"differential validation against stack ") +
			"fallback"),
		("proof-tagged simple map over []i32 guarded vector backend " +
			"lowering"),
		"vector-i32x4-map-add-const-plan",
		("single mutable slice in-place noalias-" +
			"not-required evidence"),
		"safe unaligned i32x4 map load/store",
		"scalar-i32-map fallback",
		(("linux-x64 native SIMD " +
			"lowering for proof-tagged in-place ") +
			"add-constant-1 map []i32"),
		("map []i32 translation/differential validation against stack " +
			"fallback"),
		"proof-tagged in-place add-constant-1 linux-x64 native SIMD",
		("proof-tagged memset/" +
			"memcpy helper evidence"),
		"vector-u8x16-memset-zero-plan",
		("single mutable slice " +
			"zero-fill noalias-not-required evidence"),
		"safe unaligned u8x16 zero-store",
		"scalar-u8-memset-zero fallback",
		(("linux-x64 native SIMD " +
			"zero-fill lowering for proof-tagged ") +
			"memset_zero_u8"),
		("memset_zero_u8 translation/differential validation against " +
			"stack fallback"),
		"memcpy helper via copy []u8 evidence",
		"broader map-shape vectorization",
		"checked/no-proof copy",
		"overlapping copy",
		"checked/no-proof map",
		"arbitrary non-zero memset",
		"overlapping memcpy",
		"checked/no-proof helper",
		"libc/runtime helper lowering",
		"no broad SIMD auto-vectorization",
		"performance claim",
		"validation metadata",
		"sha256",
		("actor runtime " +
			"production-boundary audit v1"),
		("tetra.runtime.actor.prod" +
			"uction_boundary.v1"),
		"current actor runtime limits",
		"scheduler prototype features",
		"production runtime acceptance",
		"full claim blockers",
		("fake full production " +
			"actor runtime claim rejection"),
		("production multi-" +
			"threaded actor scheduling"),
		("non-Linux-x64 " +
			"distributed actor runtime targets"),
		"message-pool exhaustion/reclamation",
		("full cancellation and " +
			"structured concurrency"),
		"full race-safety proof",
		"production broker deployment evidence",
		"self-hosting gate",
		"formal core spec",
		"not a public backend selector",
		"full formal proof",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing %q boundary: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"PGO/LTO/target-cpu evidence v1",
		"tetra.optimizer.profile.v1",
		("canonical JSON profile " +
			"collection format"),
		("duplicate and negative " +
			"counter rejection"),
		("Options.ProfileInput " +
			"optimizer profile input API"),
		("profile_input_policy " +
			"pass-contract metadata"),
		"profile digest validation metadata",
		("translation validation " +
			"for profile-input foundation runs"),
		("profile-guided rewrite " +
			"policy rejection"),
		"profile parsing is evidence-only",
		("target-cpu feature " +
			"detection foundation"),
		("portable baseline " +
			"target-feature model"),
		"guarded codegen contract",
		"no target-specific rewrite",
		("LTO/incremental module " +
			"summary foundation"),
		"tetra.incremental.module_summary.v1",
		"dependency hash contract",
		"non-consumer boundary",
		("no LTO optimizer or " +
			"incremental speedup claim"),
		(("final safe-semantics " +
			"closure validator rejects fake ") +
			"semantic-changing coverage"),
		"target-specific optimization evidence",
		"LTO/codegen/linker consumers",
		(("no PGO, LTO, target-cpu," +
			" or profile flag changes ") +
			"safe-program semantics"),
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P17.4 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"typed actor ownership transfer v1",
		"tetra.actors.ownership_transfer.v1",
		"borrowed-view copy boundaries",
		"owned-region move",
		"sender use-after-move diagnostics",
		"receiver ownership evidence",
		"explicit copy fallback",
		"unsafe-send contract model evidence",
		"semantics transfer checker",
		"PLIR moved facts",
		"FactMoved",
		"OpActorSend",
		"direct core.send_typed ownership transfers",
		"runtime mailbox representation",
		"actor-transfer reports",
		"stress diagnostics",
		"fake distributed zero-copy rejection",
		"fake runtime-behavior-change rejection",
		"no distributed pointer or region zero-copy",
		"safe typed actor raw pointer payload",
		"actor scheduler promotion",
		"production actor runtime claim",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P18.1 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"per-core scheduler v1",
		"tetra.parallel.per_core_scheduler.v1",
		"per-core queues",
		"work stealing",
		"bounded typed mailboxes",
		"backpressure",
		"timers sleep/wake",
		"structured task groups",
		"cancellation checkpoints",
		"actor ping-pong",
		"fanout/fanin",
		"task group cancel",
		"backpressure overflow",
		"mailbox fairness",
		"FIFO receive",
		"stress evidence",
		"race detector where applicable",
		"fake full production actor-runtime rejection",
		"fake runtime-behavior-change rejection",
		"fake all-target race-detector rejection",
		"no non-Linux distributed actor runtime target",
		"full production actor runtime",
		"full race-safety proof",
		"scheduler performance claim",
		"public runtime mode",
		"safe-semantics flag change",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P18.2 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"async I/O reactor v1",
		"tetra.runtime.io_reactor.v1",
		"Linux epoll v1",
		"io_uring future boundary",
		"kqueue macOS boundary",
		"IOCP Windows boundary",
		"WASI/web adapter boundary",
		"nonblocking accept/read/write",
		"readiness polling",
		"task wakeups from I/O readiness",
		"timer integration",
		"cancellation",
		"backpressure",
		"reactor report rows",
		"HTTP smoke",
		"DB smoke",
		"stress evidence",
		"fake full production web-stack rejection",
		"fake cross-platform reactor parity rejection",
		"fake io_uring rejection",
		"fake runtime-behavior-change rejection",
		"clear production boundary per platform",
		"no full production web stack",
		"cross-platform reactor parity",
		"io_uring support",
		"runtime behavior change",
		"official TechEmpower result",
		"production HTTP/PostgreSQL stack promotion",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P18.3 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"stable generic collections v1",
		"tetra.stdlib.generic_collections.v1",
		"Vec<T>",
		"HashMap<K,V>",
		"caller-owned slice views",
		"genericTypeName",
		"mangleGenericName",
		"bindGenericNamedTypeArgs",
		"vec_from_slice<T>",
		"hash_map_from_slices<K,V>",
		"hash_map_get_i32_i32_or",
		"hash_map_get_u8_i32_or",
		"allocation-plan report linkage",
		"core.make_*",
		("checked truth-bench-" +
			"harness dry-run artifact"),
		"p19.1_generic_collections",
		"hash table Tetra/C++/Rust equivalents",
		(("reports/stable-generic-" +
			"collections-v1/benchmarks/generic-col") +
			"lections-hash-table-report.json"),
		"algorithm_id/input metadata",
		("Tetra proof/allocation/" +
			"bounds/performance report artifacts"),
		("no allocator-backed " +
			"production Vec<T>/HashMap<K,V> runtime"),
		"generic hashing/equality protocol",
		"C++/Rust parity",
		"broad production stdlib",
		"hidden runtime allocator",
		"measured speed comparison",
		"official benchmark result",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P19.1 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"production HTTP/JSON stack v1 foundation",
		("tetra.stdlib.http_" +
			"json.production_stack.v1"),
		"HTTP/1.1 request-head parsing",
		"pipelined request heads",
		"headers/body/keep-alive metadata",
		"zero-heap request-view evidence",
		"JSON parse/stringify",
		"response building",
		("internal per-server UTC-" +
			"second Date cache helper evidence"),
		"HTTPDateCache",
		"FormatWithReport",
		"Linux writev/sendfile helper evidence",
		"netrt.Writev",
		"netrt.Sendfile",
		"p19.2_http_json_source_first",
		("Tetra-only HTTP " +
			"plaintext and HTTP JSON rows"),
		(("reports/production-http-" +
			"json-v1/benchmarks/http-json-source-") +
			"first-report.json"),
		"algorithm_id/input metadata",
		("Tetra proof/allocation/bounds/" +
			"P19.2 coverage artifacts"),
		("webrt.flush scatter/" +
			"gather integration"),
		"HTTP static-file sendfile path",
		"non-Linux writev/sendfile parity",
		"no full production web stack",
		"official TechEmpower result",
		"production PostgreSQL stack",
		"P20 performance matrix",
		"C++/Rust parity",
		"measured speed comparison",
		"source-level cached-date API",
		"cross-worker Date cache",
		"zero-copy production file-serving",
		"runtime behavior change",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P19.2 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"production PostgreSQL driver/pool v1 closure",
		("tetra.stdlib.postgresql." +
			"production_driver.v1"),
		"startup/SCRAM",
		"prepared statements",
		"binary int4 helpers",
		"pooling/backpressure",
		"borrowed DataRow decode",
		"DB single query",
		"DB multiple queries",
		"DB updates",
		"DB fortunes",
		"p19.3_postgres_source_first",
		"Tetra-only DB rows",
		(("reports/production-" +
			"postgres-v1/benchmarks/postgres-source-fi") +
			"rst-report.json"),
		"algorithm_id/input metadata",
		("Tetra proof/allocation/bounds/" +
			"P19.3 coverage artifacts"),
		("live local SCRAM " +
			"benchmark honesty evidence"),
		"validate-techempower-report",
		("techempower_scram_" +
			"single_query_local_report.json"),
		("techempower_scram_" +
			"single_query_matrix_local_report.json"),
		("techempower_scram_" +
			"endpoint_matrix_local_report.json"),
		"production database benchmark",
		("external production " +
			"database deployment"),
		("full source-level " +
			"PostgreSQL driver API"),
		"official TechEmpower result",
		"P20 performance matrix",
		"C++/Rust parity",
		"measured speed comparison",
		"runtime behavior change",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P19.3 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"benchmark matrix hardening v1",
		"p20.0_benchmark_matrix",
		"68 checked dry-run rows",
		"17 master-plan categories",
		(("Tetra, C clang -O3, C++ " +
			"clang++ -O3, and Rust rustc -C ") +
			"opt-level=3"),
		"algorithm_id/input metadata",
		"raw output artifacts on every row",
		("Tetra proof/allocation/" +
			"bounds/performance artifacts"),
		(("reports/benchmark-" +
			"matrix-hardening-v1/benchmarks/p20-matrix-") +
			"hardening-report.json"),
		"row target CPU consistency",
		"host target CPU",
		"measured speed comparison",
		"C++/Rust parity",
		"official benchmark result",
		"official TechEmpower result",
		"production database benchmark",
		"P20.1 blocker completeness",
		"P20.2 claim-tier promotion",
		"throughput advantage",
		"latency advantage",
		"startup-time advantage",
		"binary-size advantage",
		"compile-time advantage",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P20.0 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"performance blocker reports v1",
		".perf.json schema_version 3",
		"P20.1",
		"p20.0_benchmark_matrix",
		(("reports/benchmark-" +
			"matrix-hardening-v1/benchmarks/artifacts/p") +
			"20-matrix-hardening.perf.json"),
		"ValidatePerformanceBlockerReport",
		"left bounds check: missing dominance",
		("heap allocation: " +
			"escapes through return"),
		"heap allocation: unknown call",
		("heap allocation: local " +
			"call boundary heap fallback"),
		"not vectorized: no noalias proof",
		"not inlined: code-size budget",
		"register spill: live range pressure",
		("stack fallback: " +
			"unsupported aggregate return"),
		("actor copy: borrowed " +
			"data crosses boundary"),
		("17 P20.0 Tetra " +
			"benchmark explanation rows"),
		"integer_loops_tetra",
		"compile_time_tetra",
		"measured speed comparison",
		"C++/Rust parity",
		"official benchmark result",
		"official TechEmpower result",
		"P20.2 claim-tier promotion",
		"optimizer behavior change",
		"runtime behavior change",
		"blocker removal",
		"throughput advantage",
		"latency advantage",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P20.1 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"claim tiers v1",
		"tetra.performance.claim_tiers.v1",
		"p20.2_claim_tiers",
		"Tier 0 local smoke only",
		"Tier 1 local benchmark evidence",
		"Tier 2 reproducible cross-machine benchmark",
		"Tier 3 independent reproduced benchmark",
		"Tier 4 official upstream benchmark submission",
		"reports/claim-tiers-v1/claim-tier-report.json",
		"p20_current_local_smoke_only",
		"tier0_local_smoke_only",
		"local_smoke",
		"local_benchmark",
		"cross_machine_reproduction",
		"independent_reproduction",
		"official_upstream_submission",
		"fake local benchmark evidence",
		"cross-machine benchmark",
		"independent reproduced benchmark",
		"official upstream benchmark submission",
		"official TechEmpower",
		"measured speed",
		"throughput advantage",
		"latency advantage",
		"C++/Rust parity",
		"explicit non-claims",
		"current P20.0/P20.1 evidence remains Tier 0 only",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P20.2 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"specialization machine-code evidence v1",
		"tetra.optimizer.specialization_machine_code.v1",
		"p21.2_specialization_v1_v2",
		"generics",
		"protocol/static conformance",
		"extension methods",
		"enum match known cases",
		"optionals",
		"collections",
		"BuildP21SpecializationMachineCodeWitness",
		"inline-small-pure",
		"machine.ScalarIntFunctionFromStackIR",
		"absent from optimized Stack IR",
		"absent as OpCall",
		"verified scalar Machine IR",
		"translation validation",
		"monomorphized generic identity/wrapper",
		"statically checked protocol impl direct calls",
		"statically resolved extension method direct calls",
		"SCCP known enum discriminator branch folding",
		"proven-some optional presence branch folding",
		"P19.1 caller-owned Vec<T>/HashMap<K,V>",
		"validator rejects placeholder evidence",
		"fake broad specialization",
		"fake dynamic dispatch",
		"fake runtime generic values",
		"fake allocator-backed generic collections",
		"fake layout/ABI freedom",
		"fake performance",
		"fake safe-semantics changes",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P21.2 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"translation validation v2",
		"tetra.translation.validation.v2",
		"p23.0_translation_validation_v2",
		"registered optimizer pass coverage",
		"symbolic scalar equivalence",
		"supported i32 slice memory equivalence",
		"bounds proof preservation",
		"allocation plan preservation",
		"machine-checkable sha256 before/after optimization metadata",
		"opt.NewManager",
		"opt.RegisteredPasses",
		"validation.ValidateTranslation",
		"differential backend matrix loop/call/slice samples",
		"validation.ValidateAllocationLowering",
		"BuildOptimizationValidationMetadata",
		"fake full formal proof",
		"fake exhaustive optimizer completeness",
		"fake broad memory or loop proof claims",
		"fake performance",
		"fake runtime behavior change",
		"fake safe-semantics changes",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P23.0 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"fuzz/property/differential expansion v1",
		"tetra.fuzz.property.differential.v1",
		"p23.1_fuzz_property_differential",
		"parser/checker generated programs",
		"PLIR/lowering verifier pipeline",
		"backend differential matrix expansion",
		"native backend boundary",
		"runtime allocator properties",
		"actor transfer stress boundary",
		"fuzz nightly summary gate",
		"reducer failure artifacts",
		"compiler.Parse",
		"compiler.Check",
		"BuildPLIR",
		"Lower",
		"VerifyIRProgram",
		"differential.CheckBackendMatrix",
		"deterministic randomized samples",
		"Linux x64 native backend lane",
		"explicit unavailable boundary",
		"runtimeabi.AlignRegionBytes",
		"actorsafety.TypedActorOwnershipTransferCoverage",
		"stress diagnostics",
		"PLIR moved facts",
		"validate-fuzz-summary",
		"reduced_to_single_sample",
		"fake full program correctness",
		"fake exhaustive fuzzing",
		"fake full native differential",
		"fake performance",
		"fake runtime behavior change",
		"fake safe-semantics changes",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P23.1 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"formal core v1",
		"tetra.formal_core.v1",
		"p23.2_formal_core_v1",
		"values",
		"borrows and owned/copy",
		"provenance and regions",
		"bounds proof id semantics",
		"allocation length contract",
		"allocation intent lowering",
		"raw pointer bounds metadata",
		"check-elimination validity",
		"formalcore.ValidateSpec",
		"differential.CheckBackendMatrix",
		"compiler.Parse",
		"compiler.Check",
		"BuildPLIR",
		"plir.VerifyProgram",
		"validation.CheckBoundsProofsWithPLIR",
		"allocplan.FromPLIR",
		"validation.ValidateAllocationLowering",
		"runtimeabi.NewRawAllocationBounds",
		"runtimeabi.DeriveRawPointerBounds",
		"runtimeabi.RawSliceBoundsFromParts",
		"fake full formal proof",
		"fake broad language proof",
		"fake unsafe policy change",
		"fake runtime behavior change",
		"fake safe-semantics changes",
		"fake performance",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P23.2 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"self-hosting gate v1",
		"tetra.self_hosting.gate.v1",
		"p23.3_self_hosting_gate",
		"self-host subset definition",
		("small compiler " +
			"component compile boundary"),
		(("Go compiler output vs " +
			"Tetra-compiled output comparison ") +
			"boundary"),
		"register backend stability",
		"optimizer validation maturity",
		"allocator/runtime stability",
		"stdlib sufficiency",
		"deterministic bootstrap chain",
		"cross-platform bootstrap story",
		"SelfHostingClaimed=false",
		"GateDecision.Allowed=false",
		"selfhostgate.Evaluate",
		"differential.CheckBackendMatrix",
		"BuildP23TranslationValidationV2",
		"runtimeabi.RuntimeAllocationContracts",
		("runtimeabi.RuntimeRegion" +
			"AllocatorConfig"),
		"runtimeabi.RuntimePerCoreSmallHeapABI",
		"stdlibrt.RegionAwareStdlibCoverage",
		"missing small compiler component",
		"Go-vs-Tetra output comparison",
		"fake self-hosting claim",
		"fake small compiler component",
		"fake output comparison",
		"fake deterministic bootstrap",
		"fake cross-platform bootstrap",
		"fake runtime behavior change",
		"fake safe-semantics changes",
		"fake performance",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P23.3 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"security review gate v1",
		"tetra.security.review_gate.v1",
		"p24.0_security_review_gate",
		"unsafe API surface",
		"capability surface",
		"memory allocator",
		"network runtime",
		"actor runtime",
		"DB protocol",
		"package/Eco system",
		"build scripts",
		"supply chain",
		"required artifact set",
		"security-review.md",
		"threat-model.md",
		"unsafe-surface-map.md",
		"capability-surface-map.md",
		"runtimeabi.RuntimeAllocationContracts",
		"runtimeabi.RuntimeRawPointerBoundsABI",
		"netrt.IOReactorCoverage",
		"actorsrt.ActorRuntimeProductionBoundaryAudit",
		"pgrt.ProductionPostgresCoverage",
		"Eco validator path checks",
		"release security-review script checks",
		"artifact presence checks",
		"fake security certification",
		"fake external penetration test",
		"fake CVE-free status",
		"fake release security signoff",
		"fake runtime behavior change",
		"fake safe-semantics changes",
		"fake performance",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P24.0 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"runtime hardening v1",
		"tetra.runtime.hardening.v1",
		"p24.1_runtime_hardening",
		"deterministic traps",
		"OOM policy",
		"stack overflow guard boundary",
		"integer overflow semantics audit",
		"allocator corruption detection instrumentation",
		"region double-free/use-after-free instrumentation",
		"actor mailbox overflow policy",
		"network parser limits",
		"runtimeabi.RuntimeAllocationContracts",
		"runtimeabi.RuntimeRegionAllocatorConfig",
		"runtimeabi.RuntimePerCoreSmallHeapABI",
		"runtimeabi.NewPerCoreSmallHeapAllocator",
		"parallelrt.NewTypedMailbox",
		"actorsrt.ActorRuntimeProductionBoundaryAudit",
		"httprt.ParseRequest",
		"httprt.ParseRequestView",
		"pgrt.ReadFrame",
		"backend trap/stack-depth file checks",
		"optimizer overflow-semantics file checks",
		"missing runtime-hardening artifacts",
		"fake full runtime-hardening proof",
		"fake full stack-overflow protection",
		"fake OOM recovery",
		"fake full allocator-corruption detection",
		"fake production actor-mailbox promotion",
		"fake runtime behavior change",
		"fake safe-semantics changes",
		"fake performance",
	} {
		if !strings.Contains(verifiedTrack.Scope+" "+verifiedTrack.Stability, want) {
			t.Fatalf("verified track feature missing P24.1 boundary %q: %#v", want, verifiedTrack)
		}
	}
	for _, want := range []string{
		"compatibility/stability v1",
		"tetra.compatibility.stability.v1",
		"p24.2_compatibility_stability",
		"stable diagnostic codes",
		"versioned report schemas",
		"manifest compatibility checks",
		"breaking-change migration guide",
		"deprecation policy",
		"DiagnosticCodeRegistry",
		"validate-diagnostic",
		"P21-P24 schema constants",
		"validate-manifest",
		"docs/generated/manifest.json",
		"docs/spec/policy/api_diff_policy.md",
		"docs/release/policy/breaking-change-migration-guide.md",
		"docs/release/policy/deprecation_policy.md",
		"docs/release/v1_0/v1_0_x_maintenance_policy.md",
		"docs/spec/standard_library/stdlib_naming_versioning.md",
		"fake full backward compatibility",
		"fake frozen diagnostic messages",
		"fake automatic migration",
		"fake manifest/runtime ABI stability",
		"fake breaking change without migration guide",
		"fake removal without deprecation",
		"fake runtime behavior change",
		"fake safe-semantics changes",
		"fake performance",
	} {
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
		hasP17Doc = hasP17Doc || doc == "docs/audits/compiler/optimizer/pgo-lto-target-cpu-v1.md"
		hasP18OwnershipDoc = hasP18OwnershipDoc ||
			doc == "docs/audits/runtime/actors/typed-actor-ownership-transfer-v1.md"
		hasP18SchedulerDoc = hasP18SchedulerDoc ||
			doc == "docs/audits/runtime/actors/per-core-scheduler-v1.md"
		hasP18ReactorDoc = hasP18ReactorDoc ||
			doc == "docs/audits/runtime/actors/async-io-reactor-v1.md"
		hasP19GenericCollectionsDoc = hasP19GenericCollectionsDoc ||
			doc == "docs/audits/compiler/language/stable-generic-collections-v1.md"
		hasP19HTTPJSONDoc = hasP19HTTPJSONDoc ||
			doc == "docs/audits/runtime/services/production-http-json-stack-v1.md"
		hasP19PostgresDoc = hasP19PostgresDoc ||
			doc == "docs/audits/runtime/services/production-postgres-driver-pool-v1.md"
		hasP20BenchmarkMatrixDoc = hasP20BenchmarkMatrixDoc ||
			doc == "docs/audits/performance/benchmark-matrix-hardening-v1.md"
		hasP20PerformanceBlockerDoc = hasP20PerformanceBlockerDoc ||
			doc == "docs/audits/performance/performance-blocker-reports-v1.md"
		hasP20ClaimTiersDoc = hasP20ClaimTiersDoc ||
			doc == "docs/audits/performance/claim-tiers-v1.md"
		hasP21SpecializationMachineDoc = hasP21SpecializationMachineDoc ||
			doc == "docs/audits/compiler/optimizer/specialization-machine-code-v1.md"
		hasP23TranslationValidationDoc = hasP23TranslationValidationDoc ||
			doc == "docs/audits/compiler/backend/translation-validation-v2.md"
		hasP23FuzzPropertyDifferentialDoc = hasP23FuzzPropertyDifferentialDoc ||
			doc == "docs/audits/compiler/safety/fuzz-property-differential-v1.md"
		hasP23FormalCoreDoc = hasP23FormalCoreDoc ||
			doc == "docs/audits/compiler/language/formal-core-v1.md"
		hasP23SelfHostingGateDoc = hasP23SelfHostingGateDoc ||
			doc == "docs/audits/compiler/safety/self-hosting-gate-v1.md"
		hasP24SecurityReviewDoc = hasP24SecurityReviewDoc ||
			doc == "docs/audits/security/security-review.md"
		hasP24ThreatModelDoc = hasP24ThreatModelDoc || doc == "docs/audits/security/threat-model.md"
		hasP24UnsafeSurfaceMapDoc = hasP24UnsafeSurfaceMapDoc ||
			doc == "docs/audits/security/unsafe-surface-map.md"
		hasP24CapabilitySurfaceMapDoc = hasP24CapabilitySurfaceMapDoc ||
			doc == "docs/audits/security/capability-surface-map.md"
		hasP24SecurityReviewDesignDoc = hasP24SecurityReviewDesignDoc ||
			doc == "docs/plans/2026-06-03/governance-p23-p24/2026-06-03-p24.0-security-review-gate-design.md"
		hasP24RuntimeHardeningDoc = hasP24RuntimeHardeningDoc ||
			doc == "docs/audits/runtime/services/runtime-hardening-v1.md"
		hasP24RuntimeHardeningDesignDoc = hasP24RuntimeHardeningDesignDoc ||
			doc == "docs/plans/2026-06-03/governance-p23-p24/2026-06-03-p24.1-runtime-hardening-design.md"
		hasP24CompatibilityStabilityDoc = hasP24CompatibilityStabilityDoc ||
			doc == "docs/audits/security/compatibility-stability-v1.md"
		hasP24CompatibilityStabilityDesignDoc = hasP24CompatibilityStabilityDesignDoc ||
			doc == ("docs/plans/2026-06-03/governance-p23-p24/2026-06-03-p24.2-co"+
				"mpatibility-stability-design.md")
		hasP24BreakingChangeMigrationGuideDoc = hasP24BreakingChangeMigrationGuideDoc ||
			doc == "docs/release/policy/breaking-change-migration-guide.md"
		hasP24DeprecationPolicyDoc = hasP24DeprecationPolicyDoc ||
			doc == "docs/release/policy/deprecation_policy.md"
		hasTruthBenchmarkHarnessDoc = hasTruthBenchmarkHarnessDoc ||
			doc == "docs/benchmarks/truth_benchmark_harness.md"
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
	if !hasP24SecurityReviewDoc || !hasP24ThreatModelDoc || !hasP24UnsafeSurfaceMapDoc ||
		!hasP24CapabilitySurfaceMapDoc ||
		!hasP24SecurityReviewDesignDoc {
		t.Fatalf("verified track feature missing P24.0 security review docs: %#v", verifiedTrack)
	}
	if !hasP24RuntimeHardeningDoc || !hasP24RuntimeHardeningDesignDoc {
		t.Fatalf("verified track feature missing P24.1 runtime hardening docs: %#v", verifiedTrack)
	}
	if !hasP24CompatibilityStabilityDoc || !hasP24CompatibilityStabilityDesignDoc ||
		!hasP24BreakingChangeMigrationGuideDoc ||
		!hasP24DeprecationPolicyDoc {
		t.Fatalf(
			"verified track feature missing P24.2 compatibility/stability docs: %#v",
			verifiedTrack,
		)
	}
	if !hasTruthBenchmarkHarnessDoc {
		t.Fatalf("verified track feature missing truth benchmark harness doc: %#v", verifiedTrack)
	}
	protocolMVP := seenFeature["language.protocol-conformance-mvp"]
	for _, want := range []string{
		"checked statically",
		"generic requirement signature shape",
		"no witness tables",
		"dynamic dispatch remain post-v1",
	} {
		if !strings.Contains(protocolMVP.Scope+" "+protocolMVP.Stability, want) {
			t.Fatalf("protocol conformance MVP feature missing %q boundary: %#v", want, protocolMVP)
		}
	}
	callableMVP := seenFeature["language.callable-mvp"]
	for _, want := range []string{
		"Level 0 callable surface",
		"legacy ptr closure local direct calls",
		"captured closure escape",
		"full first-class function values remain out of scope",
	} {
		if !strings.Contains(callableMVP.Scope+" "+callableMVP.Stability, want) {
			t.Fatalf("callable MVP feature missing %q boundary: %#v", want, callableMVP)
		}
	}
	stdlibCore := seenFeature["stdlib.core-current"]
	for _, want := range []string{
		("executable HTTP/1.1 String and byte-buffer request-line " +
			"routing, request-head framing, and response byte-buffer " +
			"helpers"),
		("classify TechEmpower request lines from String text or " +
			"caller-owned byte buffers"),
		("locate CRLFCRLF request-head boundaries for pipelined " +
			"buffers"),
		"executable JSON byte-buffer response helpers",
		"caller-owned buffers",
		("networking exposes " +
			"deterministic endpoint policy helpers"),
		(("executable Linux TCP " +
			"socket client/server I/O helpers with ") +
			"recv/send, SO_REUSEPORT, TCP_NODELAY, nonblocking accept " +
			"convenience, and epoll add/mod/delete plus wait-one " +
			"readiness flag capture and predicates"),
		("net socket " +
			"open/bind/connect/listen/accept/read/recv/write/send/nonbloc" +
			"king/close plus SO_REUSEPORT, TCP_NODELAY, " +
			"SOCK_NONBLOCK/SOCK_CLOEXEC accept helpers, and epoll " +
			"create/add-read/add-read-write/mod-read/mod-read-write/delet" +
			"e/wait-one/wait-one-into helpers with " +
			"EPOLLIN/EPOLLOUT/EPOLLERR/EPOLLHUP predicates are " +
			"host-backed on linux-x64"),
		"stable generic collection source views",
		"lib.core.collections.Vec<T>",
		"HashMap<K,V>",
		"hash_map_get_i32_i32_or",
		"hash_map_get_u8_i32_or",
		"no hidden allocator",
		"generic hashing/equality protocol",
		"production runtime map/vector claim",
		"C++/Rust parity",
	} {
		if !strings.Contains(stdlibCore.Scope+" "+stdlibCore.Stability, want) {
			t.Fatalf("stdlib core feature missing %q boundary: %#v", want, stdlibCore)
		}
	}
	for _, want := range []string{
		"P19.2 HTTP/JSON source-first evidence",
		"lib.core.http request-head framing",
		"pipelined local buffers",
		"lib.core.json message-object writers",
		"internal per-server UTC-second Date cache evidence",
		"Linux netrt.Writev/netrt.Sendfile helper evidence",
		"tetra.stdlib.http_json.production_stack.v1",
		"production HTTP server promotion",
		"source-level cached-date API",
		"cross-worker Date cache",
		"webrt.flush scatter/gather integration",
		"HTTP static-file sendfile path",
		"zero-copy production file-serving",
		"P20 performance matrix",
		"official TechEmpower result",
	} {
		if !strings.Contains(stdlibCore.Scope+" "+stdlibCore.Stability, want) {
			t.Fatalf("stdlib core feature missing P19.2 boundary %q: %#v", want, stdlibCore)
		}
	}
	for _, want := range []string{
		"P19.3 PostgreSQL source-first and local SCRAM evidence",
		"lib.core.postgres source rows",
		"DB single query",
		"DB multiple queries",
		"DB updates",
		"DB fortunes",
		"startup/SCRAM",
		"prepared statements",
		"binary int4 helpers",
		"pooling/backpressure",
		"borrowed DataRow decode",
		"checked local SCRAM benchmark reports",
		"tetra.stdlib.postgresql.production_driver.v1",
		"p19.3_postgres_source_first",
		"validate-techempower-report",
		"full source-level PostgreSQL driver API",
		"external production database deployment",
		"production database benchmark",
		"P20 performance matrix",
		"C++/Rust parity",
		"official TechEmpower result",
		"measured speed comparison",
		"runtime behavior change",
	} {
		if !strings.Contains(stdlibCore.Scope+" "+stdlibCore.Stability, want) {
			t.Fatalf("stdlib core feature missing P19.3 boundary %q: %#v", want, stdlibCore)
		}
	}
	callableLevel1 := seenFeature["language.callable-level1"]
	if callableLevel1.Since != "v0.4.0" {
		t.Fatalf("callable Level 1 since = %q, want v0.4.0", callableLevel1.Since)
	}
	for _, want := range []string{
		"production non-capturing symbol-backed callable Level 1",
		"function-typed locals",
		("target-set-backed " +
			"function-typed parameter aliases"),
		(("function-typed " +
			"parameter storage into struct fields with ") +
			"direct field calls or synchronous callback arguments"),
		(("function-typed parameter " +
			"storage into enum payloads with ") +
			"direct payload calls, reassignment, returned enum " +
			"propagation, or synchronous callback arguments"),
		"callbacks",
		(("optional argument " +
			"labels on function-typed value calls ") +
			"including captured fnptr locals with mixed " +
			"labeled/unlabeled lists rejected"),
		("symbol-backed function-typed globals for same-module or " +
			"namespace/selective imported public direct calls plus local " +
			"initialization/reassignment"),
		"non-capturing closure-literal function-typed globals",
		(("same-module mutable " +
			"global reassignment with direct calls, ") +
			"synchronous callback arguments, function-typed returns, " +
			"generated .t4i function-typed parameter local-alias return " +
			"metadata, and local or nested local " +
			"struct-field/enum-payload " +
			"storage/reassignment/returned-aggregate propagation"),
		("imported mutable function-" +
			"typed global boundary diagnostics"),
		(("actor/task boundary " +
			"diagnostics across core.spawn, ") +
			"core.task_spawn_i32, core.task_spawn_i32_typed, " +
			"core.task_spawn_group_i32, and " +
			"core.task_spawn_group_i32_typed"),
		("pass same-module or imported direct function-typed " +
			"return-call callback arguments whose returned targets or " +
			"multi-return target sets touch mutable globals, preserve " +
			"that classification through local/field alias returns and " +
			"returned struct/enum aggregate fields or payloads across " +
			"module boundaries"),
		("imported immutable function-typed globals whose targets " +
			"touch mutable globals"),
		"symbol-backed function-typed global initializers",
		(("non-capturing generic " +
			"closure literal binding/direct ") +
			"callback/return/mutable local or nested struct field " +
			"reassignment/nested struct field initializer/enum payload " +
			"initializer or reassignment"),
		"inferable same-module or imported generic symbols",
		"function-typed returns",
		("target-set-backed " +
			"function-typed parameter returns"),
		("mutable local and " +
			"nested struct field reassignment"),
		"nested struct field initializers",
		"enum payload initializers",
		("signature-compatible " +
			"mutable local reassignment"),
		("captured closure escape " +
			"beyond the fnptr Level 2 slice"),
		("full first-class " +
			"function values remain out of scope"),
	} {
		if !strings.Contains(callableLevel1.Scope+" "+callableLevel1.Stability, want) {
			t.Fatalf("callable Level 1 feature missing %q boundary: %#v", want, callableLevel1)
		}
	}
	stdlibMirrors := seenFeature["stdlib.experimental-mirrors"]
	if stdlibMirrors.Since != "v0.4.0" {
		t.Fatalf("stdlib experimental mirrors since = %q, want v0.4.0", stdlibMirrors.Since)
	}
	for _, want := range []string{
		"production compatibility mirrors",
		"forward to lib.core",
		"stable callers should import lib.core",
	} {
		if !strings.Contains(stdlibMirrors.Scope+" "+stdlibMirrors.Stability, want) {
			t.Fatalf("stdlib mirrors feature missing %q boundary: %#v", want, stdlibMirrors)
		}
	}
	ownershipMVP := seenFeature["language.ownership-markers-mvp"]
	for _, want := range []string{
		"conservative borrow/inout/consume marker checks",
		(("same-module/cross-" +
			"module struct-field and enum-payload ") +
			"partial consume with whole-value call/let/return and enum " +
			"wrapper-constructor rejection"), "use-after-consume", (("borrow escape diagnostics for " +
			"scalar ptr including ") +
			"same-module/cross-module scalar ptr consume and inout " +
			"assignment plus match/catch-expression return escapes and " +
			"typed-error throw ptr/region payload escapes"), (("same-module/cross-module borrowed " +
			"scalar ptr escapes ") +
			"through ptr-containing struct inout assignment"), (("same-module/cross-module fixed-" +
			"array alias return plus ") +
			"direct global assignment, optional global assignment, and " +
			"inout assignment escapes with stable TETRA2102 diagnostic " +
			"evidence"), "borrowed string alias return/global assignment escapes", (("ptr/slice optional " +
			"assignment return/owned/consume/inout ") +
			"escape"), ("slice optional payload binding owned/consume/inout call, " +
			"inout-assignment, and global assignment escapes"), (("same-module/cross-module direct " +
			"slice global assignment ") +
			"with stable TETRA2102 JSON diagnostic evidence"), (("same-module/cross-module optional " +
			"ptr global assignment ") +
			"with stable TETRA2102 JSON diagnostic evidence"), (("same-module/cross-module optional " +
			"aggregate global ") +
			"assignment with stable TETRA2102 JSON diagnostic evidence"), (("same-module/cross-" +
			"module ptr optional assignment ") +
			"if-let/match global escape with stable TETRA2102 JSON " +
			"diagnostic evidence"), ("same-module/cross-module ptr enum alias return escape with " +
			"stable TETRA2102 JSON diagnostic evidence"), (("same-module/cross-module ptr-" +
			"containing aggregate ") +
			"whole/field/alias/nested-field return escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"), (("same-module/cross-module whole-aggregate " +
			"global assignment ") +
			"with stable TETRA2102 JSON diagnostic evidence"), (("same-module/cross-module ptr-" +
			"containing enum whole-value ") +
			"global assignment with stable TETRA2102 JSON diagnostic " +
			"evidence"), ("same-module/cross-module global field target assignment " +
			"with stable TETRA2102 JSON diagnostic evidence"), (("same-module/cross-module " +
			"aggregate and nested-aggregate ") +
			"global field escapes with stable TETRA2102 JSON diagnostic " +
			"evidence"), ("same-module/cross-module ptr-containing and nested " +
			"ptr-containing aggregates plus ptr-containing enum " +
			"aggregates including whole-aggregate, whole-enum, global " +
			"field target, and global field escapes"), (("optional ptr payloads including same-" +
			"module/cross-module ") +
			"whole-optional use-after-payload-consume diagnostics"), (("same-module/cross-module " +
			"optional-payload whole-value ") +
			"rejection after payload consume/free with stable TETRA2101 " +
			"JSON diagnostic evidence"), ("same-module/cross-module ptr enum-payload " +
			"return/global/inout assignment escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"), ("same-module/cross-module ptr optional-payload " +
			"return/global/inout assignment escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"), ("same-module/cross-module slice optional-payload " +
			"inout/global assignment escapes with stable TETRA2102 JSON " +
			"diagnostic evidence"), ("same-module/cross-module nested slice enum-payload " +
			"return/inout/global assignment escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"), ("same-module/cross-module nested slice struct " +
			"return/inout/global assignment escapes with stable " +
			"TETRA2102 JSON diagnostic evidence"), (("same-module/cross-module pattern-bound enum " +
			"payload and ") +
			"if-let/match optional payload return, owned/consume/inout " +
			"call, inout-assignment, and global escapes"), (("same-module/cross-module ptr-" +
			"containing/nested aggregate ") +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"), ("same-module/cross-module ptr enum-payload " +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"), ("same-module/cross-module ptr optional-payload " +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"), ("same-module/cross-module slice optional-payload " +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"), ("same-module/cross-module generic aggregate and optional-ptr " +
			"owned/consume/inout instantiations including " +
			"slice-containing struct/enum aggregate instantiations with " +
			"stable TETRA2101 CLI JSON evidence"), ("same-module/cross-module generic " +
			"borrow-aggregate/optional-ptr return diagnostics with " +
			"stable TETRA2102 CLI JSON evidence"),
		("same-module/cross-module protocol parameter ownership " +
			"matching plus same-module/cross-module protocol impl " +
			"parameter ownership mismatch diagnostics with stable " +
			"TETRA2001 CLI JSON evidence"), ("same-module/cross-module generic protocol requirement " +
			"parameter ownership mismatch diagnostics with stable " +
			"TETRA2001 JSON diagnostic evidence"), ("same-module/cross-module function-typed " +
			"value/struct-field/enum-payload optional-ptr " +
			"owned/consume/inout callback diagnostics with stable " +
			"TETRA2101 CLI JSON evidence"), ("function-typed value/struct-field/enum-payload callback " +
			"slice-containing struct/enum owned/consume/inout call " +
			"rejections with stable TETRA2101 JSON diagnostic evidence"), (("imported direct ptr-" +
			"containing/nested aggregate ") +
			"owned/consume/inout call rejections with stable TETRA2101 " +
			"JSON diagnostic evidence"), "not a full SSA lifetime solver"} {
		if !strings.Contains(ownershipMVP.Scope+" "+ownershipMVP.Stability, want) {
			t.Fatalf("ownership markers MVP feature missing %q boundary: %#v", want, ownershipMVP)
		}
	}
	resourceMVP := seenFeature["language.resource-lifetime-mvp"]
	for _, want := range []string{
		"conservative resource finalization checks",
		"task handles",
		(("branch/match/loop task-" +
			"handle maybe-joined, task-group ") +
			"maybe-closed, and island maybe-freed merge diagnostics; " +
			"branch/match/loop resource finalization merge diagnostics " +
			"with stable TETRA2101 JSON evidence"),
		(("stable ownership safety JSON diagnostics for " +
			"resource ") +
			"use-after-free, double-join, and ambiguous-provenance cases"),
		(("same-module/cross-" +
			"module struct-field and enum-payload ") +
			"alias use-after-free with stable TETRA2101 JSON diagnostic " +
			"evidence"),
		"island handles",
		("same-module/cross-module task-handle/task-group " +
			"struct-field/enum-payload join/close aliases"),
		("same-module/cross-module task-handle " +
			"struct-field/enum-payload alias join diagnostics with " +
			"stable TETRA2101 JSON diagnostic evidence"),
		("same-module/cross-module task-group " +
			"struct-field/enum-payload alias close diagnostics with " +
			"stable TETRA2101 JSON diagnostic evidence"),
		(("same-module/cross-module enum-" +
			"constructor return resource ") +
			"aliases with stable TETRA2101 CLI JSON evidence"),
		(("same-module typed-error throw/" +
			"catch and rethrow-through-try ") +
			"enum-payload resource aliases with stable TETRA2101 JSON " +
			"diagnostic evidence"),
		("generated .t4i " +
			"direct/local/aggregate-local-alias/aggregate-field-access/ag" +
			"gregate-field-local-alias resource return, " +
			"assignment/let/direct-if-let/direct-match/field-local/if-let" +
			"/match optional and nested/field-local nested optional " +
			"resource return, typed-error direct/field-local-alias throw," +
			" and rethrow-through-try direct/field-local-alias " +
			"provenance stubs"),
		("same-module/cross-module monomorphized generic struct " +
			"task-handle/task-group/island resource aliases with stable " +
			"TETRA2101 CLI JSON evidence"),
		"enum-payload",
		(("if-let/match optional-payload return " +
			"aliases including ") +
			"nested struct-field and enum-payload wrappers"),
		(("same-module/cross-module task-" +
			"handle/task-group ") +
			"if-let/match optional-payload join/close aliases with " +
			"stable TETRA2101 CLI JSON evidence"),
		("same-module/cross-module island whole-optional " +
			"use-after-payload-free diagnostics"),
		("same-module/cross-module transitive interprocedural " +
			"task-handle/task-group/island resource aliases with stable " +
			"TETRA2101 CLI JSON evidence"),
		"double-use",
		"ambiguous provenance",
		"not a full SSA lifetime solver",
	} {
		if !strings.Contains(resourceMVP.Scope+" "+resourceMVP.Stability, want) {
			t.Fatalf("resource lifetime MVP feature missing %q boundary: %#v", want, resourceMVP)
		}
	}
	transferMVP := seenFeature["actors.task-transfer-safety"]
	for _, want := range []string{
		"conservative actor/task ownership transfer checks",
		"worker entrypoints",
		(("branch/match/loop actor " +
			"consume reuse diagnostics with ") +
			"stable TETRA2101 CLI JSON evidence"),
		("actor/task use-after-transfer diagnostics with stable " +
			"TETRA2101 CLI JSON evidence"),
		("island transfer non-local-payload rejection with stable " +
			"TETRA2101 CLI JSON evidence"),
		("same-module/cross-module transitive actor consume alias " +
			"diagnostics with stable TETRA2101 CLI JSON evidence"),
		(("same-module/cross-module " +
			"monomorphized generic struct actor ") +
			"consume alias diagnostics with stable TETRA2101 CLI JSON " +
			"evidence"),
		("same-module/cross-module task_group_cancel return " +
			"provenance diagnostics with stable TETRA2101 CLI JSON " +
			"evidence"),
		("same-module/cross-module actor if-let/match " +
			"optional-payload, struct-field, and enum-payload consume " +
			"alias diagnostics"),
		("same-module/cross-module actor struct-field/enum-payload " +
			"alias transfer diagnostics with stable TETRA2101 JSON " +
			"diagnostic evidence"),
		("same-module/cross-module actor/task if-let/match " +
			"optional-payload alias transfer diagnostics with stable " +
			"TETRA2101 JSON diagnostic evidence"),
		("same-module/cross-module task-handle " +
			"struct-field/enum-payload alias transfer diagnostics with " +
			"stable TETRA2101 JSON diagnostic evidence"),
		("same-module/cross-module task-handle " +
			"struct-field/enum-payload alias join diagnostics with " +
			"stable TETRA2101 JSON diagnostic evidence"),
		"cooperative task_group_cancel",
		"conservative local MVP",
		"distributed actors",
	} {
		if !strings.Contains(transferMVP.Scope+" "+transferMVP.Stability, want) {
			t.Fatalf("actor/task transfer feature missing %q boundary: %#v", want, transferMVP)
		}
	}
	lifetimeSSA := seenFeature["language.lifetime-ssa"]
	if lifetimeSSA.Since != "v0.4.0" {
		t.Fatalf("lifetime SSA since = %q, want v0.4.0", lifetimeSSA.Since)
	}
	for _, want := range []string{
		"production SSA-like local lifetime join analysis",
		"ownership consume state",
		"resource finalization state",
		"optional region-wrapper escapes",
		(("same-module and " +
			"interface-only cross-module per-field ") +
			"interprocedural region summaries"),
		"optional aggregate wrappers",
		"enum payload wrappers",
		"branch aggregate wrappers",
		"match aggregate wrappers",
		"if-let aggregate wrappers",
		("mixed safe/provenance " +
			"aggregate branch and match returns"),
		("optional mixed safe/" +
			"provenance aggregate branch merges"),
		"maybe-consumed diagnostics",
		("richer interprocedural " +
			"lifetime proofs"),
	} {
		if !strings.Contains(lifetimeSSA.Scope+" "+lifetimeSSA.Stability, want) {
			t.Fatalf("lifetime SSA feature missing %q boundary: %#v", want, lifetimeSSA)
		}
	}
	callableLevel2 := seenFeature["language.callable-level2"]
	if callableLevel2.Since != "v0.4.0" {
		t.Fatalf("callable Level 2 since = %q, want v0.4.0", callableLevel2.Since)
	}
	for _, want := range []string{
		"production captured closure Level 2 slice",
		"fnptr-backed function-typed locals",
		(("captured ptr closure " +
			"aliases into function-typed locals, ") +
			"mutable function-typed local reassignment"),
		(("same-module mutable function-typed " +
			"global snapshot ") +
			"reassignment"),
		("direct synchronous callback arguments including direct " +
			"closure literals passed to imported callbacks"),
		(("function-typed returns including " +
			"direct return of let-bound ") +
			"captured ptr closure values"),
		("direct closure-literal container initializers in " +
			"module-aware lowering"),
		("direct calls including labeled direct calls on captured ptr " +
			"closures"),
		"imported parameter-return callbacks",
		("up to eight by-value snapshot " +
			"environment slots"),
		(("cross-module returned " +
			"captured closures used through locals ") +
			"or direct callback arguments"),
		("cross-module struct-parameter function-field dispatch " +
			"including namespace/selective imported direct struct " +
			"constructors carrying closure literals or captured ptr " +
			"closure locals"),
		("cross-module enum-parameter function-payload dispatch " +
			"including direct namespace/selective imported enum " +
			"constructor arguments"),
		"immutable local struct fields or enum payloads",
		(("larger immutable " +
			"environments are promoted under ") +
			"language.full-first-class-callables"),
	} {
		if !strings.Contains(callableLevel2.Scope+" "+callableLevel2.Stability, want) {
			t.Fatalf("callable Level 2 feature missing %q boundary: %#v", want, callableLevel2)
		}
	}
	fullCallables := seenFeature["language.full-first-class-callables"]
	if fullCallables.Since != "v0.4.0" {
		t.Fatalf("full first-class callables since = %q, want v0.4.0", fullCallables.Since)
	}
	for _, want := range []string{
		"production first-class callable/function-value semantics",
		"bounded fnptr fast path",
		"fixed 4-slot callable handle",
		("larger immutable Int/" +
			"Bool/String/simple-aggregate captures"),
		"local storage",
		"mutable local reassignment",
		"returns",
		"same-module global snapshots",
		"struct fields",
		"enum payloads",
		"synchronous callback arguments",
		"cross-module returned values",
		"aliases",
		(("generated .t4i function-" +
			"typed parameter local-alias return ") +
			"metadata"),
		"generated .t4i metadata",
		(("stable JSON diagnostics for mutable by-" +
			"reference captures ") +
			"including callable mutable-capture global-escape"),
		"callable mutable-capture heap-escape",
		("callable pointer/" +
			"resource capture escape"),
		("function-typed storage/" +
			"return unsupported capture rejection"),
		(("captured callable/" +
			"function-typed parameter global-storage ") +
			"escape"),
		"unsupported function-value escape outside the fnptr ABI",
		"unsupported function-value call",
		"capturing closure raw-ptr escape",
		("captured closure " +
			"explicit type-arg rejection"),
		("function-typed explicit " +
			"type-arg rejection"),
		(("generic closure capture " +
			"and generic callback-closure ") +
			"capture rejection"),
		"generic closure pointer/direct-call rejection",
		("imported mutable " +
			"function-typed global boundary"),
		"thread-boundary callable escape",
	} {
		if !strings.Contains(fullCallables.Scope+" "+fullCallables.Stability, want) {
			t.Fatalf(
				"full first-class callable feature missing %q boundary: %#v",
				want,
				fullCallables,
			)
		}
	}
	enumFeature := seenFeature["language.enum-payload-match"]
	if enumFeature.Since != "v0.3.0" {
		t.Fatalf("enum payload feature since = %q, want v0.3.0", enumFeature.Since)
	}
	for _, want := range []string{
		"positional enum payload constructors",
		"match/catch/if-let",
		"exhaustive unguarded enum match/catch",
		"advanced ADT constructors",
		"nested destructuring patterns",
		"guard expansion remain future/post-v1",
	} {
		if !strings.Contains(enumFeature.Scope+" "+enumFeature.Stability, want) {
			t.Fatalf("enum payload feature missing %q boundary: %#v", want, enumFeature)
		}
	}
	protocolBoundGenerics := seenFeature["language.protocol-bound-generics-static"]
	if protocolBoundGenerics.Since != "v0.3.0" {
		t.Fatalf("protocol-bound generics since = %q, want v0.3.0", protocolBoundGenerics.Since)
	}
	for _, want := range []string{
		"validated statically during monomorphization",
		(("same-module and cross-" +
			"module impl conformance with ") +
			"parameter ownership markers"),
		"visibility diagnostics",
		"calling protocol requirements through generic bounds",
		"witness tables",
		"dynamic dispatch remain unsupported",
	} {
		if !strings.Contains(
			protocolBoundGenerics.Scope+" "+protocolBoundGenerics.Stability,
			want,
		) {
			t.Fatalf(
				"protocol-bound generics feature missing %q boundary: %#v",
				want,
				protocolBoundGenerics,
			)
		}
	}
	effectsMVP := seenFeature["safety.effects-mvp"]
	if effectsMVP.Since != "v0.3.0" {
		t.Fatalf("effects MVP since = %q, want v0.3.0", effectsMVP.Since)
	}
	for _, want := range []string{
		"stable uses effect names and groups",
		"transitive call propagation",
		"missing uses declarations are diagnostics",
		"checker-enforced optimizer facts",
		"pure/no-alloc/no-mem-write/no-actor-send/no-unknown-escape",
		"no effect inference",
	} {
		if !strings.Contains(effectsMVP.Scope+" "+effectsMVP.Stability, want) {
			t.Fatalf("effects MVP feature missing %q boundary: %#v", want, effectsMVP)
		}
	}
	capabilitiesMVP := seenFeature["safety.capabilities-mvp"]
	if capabilitiesMVP.Since != "v0.3.0" {
		t.Fatalf("capabilities MVP since = %q, want v0.3.0", capabilitiesMVP.Since)
	}
	for _, want := range []string{
		"cap.io and cap.mem opaque tokens",
		"unsafe blocks",
		"raw memory/MMIO",
		"capsule permissions",
		"not a broad safe-code capability construction model",
	} {
		if !strings.Contains(capabilitiesMVP.Scope+" "+capabilitiesMVP.Stability, want) {
			t.Fatalf("capabilities MVP feature missing %q boundary: %#v", want, capabilitiesMVP)
		}
	}
	privacyMVP := seenFeature["safety.privacy-consent-mvp"]
	if privacyMVP.Since != "v0.3.0" {
		t.Fatalf("privacy/consent MVP since = %q, want v0.3.0", privacyMVP.Since)
	}
	for _, want := range []string{
		"uses privacy requires privacy",
		"secret.i32/SecretInt",
		"consent token",
		"not cryptographic isolation",
		"distributed consent enforcement remains post-v1",
	} {
		if !strings.Contains(privacyMVP.Scope+" "+privacyMVP.Stability, want) {
			t.Fatalf("privacy/consent MVP feature missing %q boundary: %#v", want, privacyMVP)
		}
	}
	budgetMVP := seenFeature["safety.budget-mvp"]
	if budgetMVP.Since != "v0.3.0" {
		t.Fatalf("budget MVP since = %q, want v0.3.0", budgetMVP.Since)
	}
	for _, want := range []string{
		"budget(<non-negative integer constant>)",
		"uses budget",
		"deterministic budget guard instructions",
		"not cross-function runtime-wide",
		"distributed budget enforcement remains post-v1",
	} {
		if !strings.Contains(budgetMVP.Scope+" "+budgetMVP.Stability, want) {
			t.Fatalf("budget MVP feature missing %q boundary: %#v", want, budgetMVP)
		}
	}
	safetyCore := seenFeature["safety.production-core"]
	if safetyCore.Since != "v0.4.0" {
		t.Fatalf("safety production core since = %q, want v0.4.0", safetyCore.Since)
	}
	for _, want := range []string{
		"production local safety model",
		"ownership/lifetime/borrow/consume/inout",
		"resource finalization",
		"callable escape diagnostics",
		"effects/capabilities/privacy/consent/budget",
		"unsafe boundaries",
		"actor/task transfer safety",
		"pointer/MMIO/memory capability gates",
		"memory production final audit",
		"explicit diagnostics",
	} {
		if !strings.Contains(safetyCore.Scope+" "+safetyCore.Stability, want) {
			t.Fatalf("safety production core missing %q boundary: %#v", want, safetyCore)
		}
	}
	uiMetadata := seenFeature["ui.metadata-v1"]
	if uiMetadata.Status != compiler.FeatureStatusCurrent || uiMetadata.Since != "v0.4.0" {
		t.Fatalf(
			"ui.metadata-v1 lifecycle = status %q since %q, want current since v0.4.0",
			uiMetadata.Status,
			uiMetadata.Since,
		)
	}
	for _, want := range []string{
		"production UI metadata contract",
		"deterministic tetra.ui.v0.4.0 JSON",
		"browser-backed web command-dispatch runtime",
		"wasm32-web command dispatch",
		"post-v0.4 Web UI runtime smoke",
		"native shell command dispatch",
		"widget-tree traces",
		"JSON trace sidecars",
		"style metadata preview attributes",
		"accessibility metadata preview attributes",
	} {
		if !strings.Contains(uiMetadata.Scope+" "+uiMetadata.Stability, want) {
			t.Fatalf("UI metadata feature missing %q boundary: %#v", want, uiMetadata)
		}
	}
	uiToolkit := seenFeature["ui.toolkit-core"]
	if uiToolkit.Status != compiler.FeatureStatusCurrent || uiToolkit.Since != "v0.4.0" {
		t.Fatalf(
			"ui.toolkit-core lifecycle = status %q since %q, want current since v0.4.0",
			uiToolkit.Status,
			uiToolkit.Since,
		)
	}
	for _, want := range []string{
		"production platform-independent UI Toolkit Core contract",
		"tetra.ui.toolkit.v1",
		"widget model",
		"layout model",
		"accessibility model",
		"event dispatch",
		"state binding/update",
		"runtime trace artifacts",
		"metadata-only",
		"runtime-less",
		"native-shell sidecar-only",
		"web-only",
		"GTK/Qt/OS platform backend production",
		"full cross-platform UI",
	} {
		if !strings.Contains(uiToolkit.Scope+" "+uiToolkit.Stability, want) {
			t.Fatalf("UI toolkit core feature missing %q boundary: %#v", want, uiToolkit)
		}
	}
	for _, wantDoc := range []string{
		"docs/spec/core/current_supported_surface.md",
		"docs/spec/ui/ui_toolkit_core.md",
		"docs/spec/ui/ui_v0.4.0.md",
	} {
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
	if distributedActors.Status != compiler.FeatureStatusCurrent ||
		distributedActors.Since != "v0.4.0" {
		t.Fatalf(
			"distributed actors lifecycle = status %q since %q, want current since v0.4.0",
			distributedActors.Status,
			distributedActors.Since,
		)
	}
	for _, want := range []string{
		"production Linux-x64 distributed actor runtime path",
		"actornet loopback TCP broker",
		"distributed node identity",
		"remote actor handles",
		"network mailbox send/receive",
		"i32, tagged, and typed frames",
		"missing-node failure/status propagation",
		"task cancel/join handles",
		"tetra.actors.distributed-runtime.v1 smoke evidence",
		"tetra.actor.production_foundation.v1",
		"actor-runtime-foundation-linux-x64-gate.sh",
		"transport-only or fake reports",
		"non-Linux-x64 targets",
		"non-Linux distributed runtime",
		"distributed zero-copy",
		"cluster membership",
		"reconnect/retry production",
		"formal race proof",
		"broader structured-concurrency guarantees",
	} {
		if !strings.Contains(distributedActors.Scope+" "+distributedActors.Stability, want) {
			t.Fatalf("distributed actors feature missing %q boundary: %#v", want, distributedActors)
		}
	}
	uiRuntime := seenFeature["ui.native-runtime"]
	if uiRuntime.Status != compiler.FeatureStatusCurrent || uiRuntime.Since != "v0.4.0" {
		t.Fatalf(
			"UI native runtime lifecycle = status %q since %q, want current since v0.4.0",
			uiRuntime.Status,
			uiRuntime.Since,
		)
	}
	for _, want := range []string{
		"production Linux-x64 native UI runtime path",
		"native runtime widget instances",
		"click/activate events",
		"lowered command operations",
		"state and widget updates",
		"tetra.ui.native-runtime.v1 smoke evidence",
		"metadata-only",
		"web-only",
		"native-shell sidecar-only",
		"macOS/Windows",
		"platform accessibility integration",
	} {
		if !strings.Contains(uiRuntime.Scope+" "+uiRuntime.Stability, want) {
			t.Fatalf("UI runtime feature missing %q boundary: %#v", want, uiRuntime)
		}
	}
	platformUI := seenFeature["ui.platform-runtime"]
	if platformUI.Status != compiler.FeatureStatusExperimental || platformUI.Since != "v0.4.0" {
		t.Fatalf(
			"UI platform runtime lifecycle = status %q since %q, want experimental since v0.4.0",
			platformUI.Status,
			platformUI.Since,
		)
	}
	for _, want := range []string{
		"tetra.ui.platform-runtime.v1",
		"full-platform UI runtime promotion gate",
		"real Windows/macOS target-host reports",
		"not production until",
		"metadata-only",
		"runtime-less",
		"startup_failure",
	} {
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
			t.Fatalf(
				"cli.core scope missing documented public command %q: %q",
				command,
				cliCore.Scope,
			)
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
			t.Fatalf(
				"feature %s status = %q, want current Surface v1 release scope",
				id,
				feature.Status,
			)
		}
		for _, want := range wantPhrases {
			if !strings.Contains(feature.Scope+" "+feature.Stability, want) {
				t.Fatalf("feature %s missing %q in scope/stability: %#v", id, want, feature)
			}
		}
		if len(feature.Docs) == 0 ||
			!hasFeatureDoc(feature.Docs, "docs/spec/surface/surface_v1.md") {
			t.Fatalf(
				"feature %s docs = %#v, want docs/spec/surface/surface_v1.md",
				id,
				feature.Docs,
			)
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
			t.Fatalf(
				"ui.surface-block-system missing P19 truth-boundary phrase %q: %#v",
				want,
				blockSystem,
			)
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
			t.Fatalf(
				"historical Surface feature %s missing absorbed/internal note: %#v",
				id,
				feature,
			)
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
		"docs/audits/master-plan/master-plan-final-20260602.md",
		"docs/audits/master-plan/master-plan-final-20260602-artifact-map.md",
		"docs/audits/performance/truthful-performance-core-baseline.md",
		"docs/audits/compiler/safety/safe-borrow-returns-v1.md",
		"docs/audits/compiler/safety/noalias-mutable-borrow-v1.md",
		"docs/audits/compiler/safety/lifetime-module-boundaries-v1.md",
		"docs/audits/memory/ram-raw/implicit-region-lowering-readiness-v1.md",
		"docs/audits/memory/ram-raw/request-task-region-v1.md",
		"docs/audits/runtime/actors/thread-per-core-allocator-v1.md",
		"docs/audits/memory/ram-raw/raw-pointer-bounds-metadata-v1.md",
		"docs/audits/compiler/backend/backend-coverage-audit-v1.md",
		"docs/audits/compiler/backend/value-ssa-ir-v1.md",
		"docs/audits/compiler/backend/register-backend-coverage-expansion-v1.md",
		"docs/audits/compiler/backend/backend-differential-validation-v1.md",
		"docs/audits/compiler/optimizer/optimizer-pass-contract-v1.md",
		"docs/audits/compiler/optimizer/optimizer-core-coverage-v1.md",
		"docs/audits/compiler/optimizer/vectorization-v1.md",
		"docs/audits/compiler/optimizer/pgo-lto-target-cpu-v1.md",
		"docs/audits/runtime/actors/actor-runtime-production-boundary-v1.md",
		"docs/audits/runtime/actors/typed-actor-ownership-transfer-v1.md",
		"docs/audits/runtime/actors/per-core-scheduler-v1.md",
		"docs/audits/runtime/actors/async-io-reactor-v1.md",
		"docs/audits/runtime/services/region-aware-stdlib-v1.md",
		"docs/audits/compiler/language/stable-generic-collections-v1.md",
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
				t.Fatalf(
					"builtin %s exposes non-canonical effect %q in manifest",
					builtin.Name,
					effect,
				)
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
			t.Fatalf(
				"manifest builtin %s = effects=%q unsafe_policy=%q, want effects=%q unsafe_policy=%q",
				name,
				strings.Join(got.Effects, ","),
				got.UnsafePolicy,
				want.effects,
				want.unsafePolicy,
			)
		}
	}

	island, ok := byName["core.island_make_i32"]
	if !ok {
		t.Fatalf("manifest missing builtin core.island_make_i32")
	}
	if island.UnsafePolicy != "conditional" ||
		!strings.Contains(island.UnsafeDetails, "requires unsafe") {
		t.Fatalf("manifest builtin core.island_make_i32 = %#v", island)
	}
}

// ---- generics_test.go ----

func TestGenericFunctionParseCheckAndDocs(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := prog.Funcs[0].TypeParams; len(got) != 1 || got[0] != "T" {
		t.Fatalf("type params = %#v", got)
	}
	if _, err := compiler.Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
	docs, err := compiler.GenerateAPIDocsFromSource(src, "generics.tetra")
	if err != nil {
		t.Fatalf("GenerateAPIDocsFromSource: %v", err)
	}
	if !strings.Contains(string(docs), "`func id<T>(x: T) -> T`") {
		t.Fatalf("docs = %s", string(docs))
	}
}

func TestGenericFunctionMonomorphizedCall(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return id(42)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	sig, ok := checked.FuncSigs["id__T_i32"]
	if !ok {
		t.Fatalf("missing monomorphized signature: %#v", checked.FuncSigs)
	}
	if sig.Generic {
		t.Fatalf("id__T_i32 should be concrete after monomorphization: %#v", sig)
	}
	if sig.ParamSlots != 1 || sig.ReturnSlots != 1 || sig.ReturnType != "i32" {
		t.Fatalf(
			"id__T_i32 ABI = params %d returns %d type %q, want params 1 returns 1 type i32",
			sig.ParamSlots,
			sig.ReturnSlots,
			sig.ReturnType,
		)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	idFn := findIRFunc(t, irProg.Funcs, "id__T_i32")
	if idFn.ParamSlots != 1 || idFn.ReturnSlots != 1 {
		t.Fatalf(
			"lowered id__T_i32 ABI = params %d returns %d, want params 1 returns 1",
			idFn.ParamSlots,
			idFn.ReturnSlots,
		)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "id__T_i32") {
		t.Fatalf("main did not call monomorphized id__T_i32: %#v", mainFn.Instrs)
	}
}

func TestP9GenericIdentityDisappearsAfterSmallPureInlining(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return id(42)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	before := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(before, "id__T_i32") {
		t.Fatalf("pre-optimization main did not call id__T_i32: %#v", before.Instrs)
	}
	if _, err := opt.NewManager().Run(irProg, opt.InlineSmallPurePass()); err != nil {
		t.Fatalf("InlineSmallPurePass: %v", err)
	}
	after := findIRFunc(t, irProg.Funcs, "main")
	if hasIRCall(after, "id__T_i32") {
		t.Fatalf("generic identity call survived specialization/inlining: %#v", after.Instrs)
	}
}

func TestP17GenericWrapperDisappearsAfterSmallPureInlining(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func wrap<T>(x: T) -> T:
    return id(x)

func main() -> Int:
    return wrap(42)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	before := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(before, "wrap__T_i32") {
		t.Fatalf("pre-optimization main did not call wrap__T_i32: %#v", before.Instrs)
	}
	report, err := opt.NewManager().Run(irProg, opt.InlineSmallPurePass())
	if err != nil {
		t.Fatalf("InlineSmallPurePass: %v", err)
	}
	row := report.Passes[0]
	if !hasOptDecision(row.Decisions, "inlined", "main", "wrap__T_i32", "small_pure_wrapper") {
		t.Fatalf("missing wrapper inline decision in %#v", row.Decisions)
	}
	if !hasOptDecision(row.Decisions, "inlined", "main", "id__T_i32", "small_pure") {
		t.Fatalf("missing nested identity inline decision in %#v", row.Decisions)
	}
	after := findIRFunc(t, irProg.Funcs, "main")
	if hasIRCall(after, "wrap__T_i32") || hasIRCall(after, "id__T_i32") {
		t.Fatalf("generic wrapper call survived specialization/inlining: %#v", after.Instrs)
	}
}

func TestGenericFunctionProtocolBoundConformancePasses(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: Echoable

func id<T: Echoable>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	sig, ok := checked.FuncSigs["id__T_Vec2"]
	if !ok {
		t.Fatalf("missing protocol-bound monomorphized signature: %#v", checked.FuncSigs)
	}
	if sig.Generic {
		t.Fatalf("id__T_Vec2 should be concrete after monomorphization: %#v", sig)
	}
	if sig.ParamSlots != 1 || sig.ReturnSlots != 1 || sig.ReturnType != "Vec2" {
		t.Fatalf(
			"id__T_Vec2 ABI = params %d returns %d type %q, want params 1 returns 1 type Vec2",
			sig.ParamSlots,
			sig.ReturnSlots,
			sig.ReturnType,
		)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	idFn := findIRFunc(t, irProg.Funcs, "id__T_Vec2")
	if idFn.ParamSlots != 1 || idFn.ReturnSlots != 1 {
		t.Fatalf(
			"lowered id__T_Vec2 ABI = params %d returns %d, want params 1 returns 1",
			idFn.ParamSlots,
			idFn.ReturnSlots,
		)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "id__T_Vec2") {
		t.Fatalf("main did not call protocol-bound id__T_Vec2: %#v", mainFn.Instrs)
	}
}

func hasOptDecision(
	decisions []opt.PassDecision,
	action string,
	caller string,
	callee string,
	reason string,
) bool {
	for _, decision := range decisions {
		if decision.Action == action && decision.Caller == caller && decision.Callee == callee &&
			decision.Reason == reason {
			return true
		}
	}
	return false
}

func TestGenericFunctionProtocolBoundRejectsMissingImpl(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

func id<T: Echoable>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected protocol-bound conformance diagnostic")
	}
	if !strings.Contains(
		err.Error(),
		"generic argument 'Vec2' does not satisfy bound 'Echoable' for 'T'",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionProtocolBoundRejectsMismatchedImplSignature(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Int:
        return self.x

impl Vec2: Echoable

func id<T: Echoable>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected protocol-bound conformance diagnostic")
	}
	if !strings.Contains(err.Error(), "return type differs") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionProtocolBoundCrossModuleConformancePasses(t *testing.T) {
	files := map[string]string{
		"engine/core.tetra": `module engine.core
pub struct Vec2:
    x: Int

pub protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: Echoable

pub func id<T: Echoable>(x: T) -> T:
    return x
`,
		"app/main.tetra": `module app.main
import engine.core as core

func main() -> Int:
    let v: core.Vec2 = core.Vec2(x: 42)
    let out: core.Vec2 = core.id(v)
    return out.x
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, ok := checked.FuncSigs["engine.core.id__T_engine_2e_core_2e_Vec2"]; !ok {
		t.Fatalf(
			"missing cross-module protocol-bound monomorphized signature: %#v",
			checked.FuncSigs,
		)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestGenericFunctionProtocolBoundCrossModuleRejectsMissingImpl(t *testing.T) {
	files := map[string]string{
		"engine/core.tetra": `module engine.core
pub struct Vec2:
    x: Int

pub protocol Echoable:
    func echo(self: Vec2) -> Vec2

pub func id<T: Echoable>(x: T) -> T:
    return x
`,
		"app/main.tetra": `module app.main
import engine.core as core

func main() -> Int:
    let v: core.Vec2 = core.Vec2(x: 42)
    let out: core.Vec2 = core.id(v)
    return out.x
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected cross-module protocol-bound conformance diagnostic")
	}
	if !strings.Contains(
		err.Error(),
		"generic argument 'Vec2' does not satisfy bound 'Echoable' for 'T'",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionProtocolBoundRejectsUnknownProtocolBound(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

func id<T: MissingProtocol>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected unknown protocol bound diagnostic")
	}
	if !strings.Contains(
		err.Error(),
		"unknown protocol bound 'MissingProtocol' for generic parameter 'T'",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionProtocolBoundRejectsNonProtocolBound(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

func id<T: Vec2>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected non-protocol bound diagnostic")
	}
	if !strings.Contains(err.Error(), "generic bound 'Vec2' for 'T' must name a protocol") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionProtocolBoundRejectsPrivateCrossModuleProtocolBound(t *testing.T) {
	files := map[string]string{
		"engine/core.tetra": `module engine.core
pub struct Vec2:
    x: Int

protocol HiddenEchoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: HiddenEchoable

pub func id<T: HiddenEchoable>(x: T) -> T:
    return x
`,
		"app/main.tetra": `module app.main
import engine.core as core

func main() -> Int:
    let v: core.Vec2 = core.Vec2(x: 42)
    let out: core.Vec2 = core.id(v)
    return out.x
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected private protocol bound visibility diagnostic")
	}
	if !strings.Contains(
		err.Error(),
		"private protocol 'engine.core.HiddenEchoable' is not visible from module 'app.main'",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionProtocolBoundRequirementCallUnsupported(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: Echoable

func echoThroughBound<T: Echoable>(x: T) -> T:
    return T.echo(x)

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = echoThroughBound(v)
    return out.x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected unsupported generic-bound requirement call diagnostic")
	}
	if !strings.Contains(
		err.Error(),
		"calling protocol requirement 'echo' through generic bound 'T' is not supported in this MVP",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericStructSameModuleMonomorphizedHappyPath(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func main() -> Int:
    let b: Box<Int> = Box<Int>{value: 42}
    return b.value
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	box, ok := checked.Types["Box__T_i32"]
	if !ok {
		t.Fatalf("missing monomorphized struct type: %#v", checked.Types)
	}
	if box.SlotCount != 1 || len(box.Fields) != 1 || box.Fields[0].TypeName != "i32" ||
		box.Fields[0].SlotCount != 1 {
		t.Fatalf("Box__T_i32 layout = %#v, want one i32 field", box)
	}
	if _, exists := checked.Types["Box"]; exists {
		t.Fatalf(
			"generic struct template should not remain in checked types: %#v",
			checked.Types["Box"],
		)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if mainFn.ReturnSlots != 1 {
		t.Fatalf("main ReturnSlots = %d, want 1", mainFn.ReturnSlots)
	}
}

func TestGenericFunctionReturningGenericStructMonomorphizesStruct(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func make<T>(x: T) -> Box<T>:
    return Box<T>{value: x}

func main() -> Int:
    let b: Box<Int> = make(42)
    return b.value
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	box, ok := checked.Types["Box__T_i32"]
	if !ok {
		t.Fatalf("missing monomorphized struct type: %#v", checked.Types)
	}
	if box.SlotCount != 1 || len(box.Fields) != 1 || box.Fields[0].TypeName != "i32" {
		t.Fatalf("Box__T_i32 layout = %#v, want one i32 field", box)
	}
	sig, ok := checked.FuncSigs["make__T_i32"]
	if !ok {
		t.Fatalf("missing monomorphized function signature: %#v", checked.FuncSigs)
	}
	if sig.ReturnType != "Box__T_i32" {
		t.Fatalf("make__T_i32 return type = %q, want Box__T_i32", sig.ReturnType)
	}
	if sig.ParamSlots != 1 || sig.ReturnSlots != 1 {
		t.Fatalf(
			"make__T_i32 ABI = params %d returns %d, want params 1 returns 1",
			sig.ParamSlots,
			sig.ReturnSlots,
		)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	makeFn := findIRFunc(t, irProg.Funcs, "make__T_i32")
	if makeFn.ParamSlots != 1 || makeFn.ReturnSlots != 1 {
		t.Fatalf(
			"lowered make__T_i32 ABI = params %d returns %d, want params 1 returns 1",
			makeFn.ParamSlots,
			makeFn.ReturnSlots,
		)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "make__T_i32") {
		t.Fatalf("main did not call monomorphized make__T_i32: %#v", mainFn.Instrs)
	}
}

func TestGenericFunctionInfersThroughGenericStructParameter(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func get_or<T>(box: Box<T>, fallback: T) -> T:
    if fallback == 0:
        return box.value
    return fallback

func main() -> Int:
    let b: Box<Int> = Box<Int>{value: 42}
    return get_or(b, 0)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.Types["Box__T_i32"]; !ok {
		t.Fatalf("missing monomorphized Box type: %#v", checked.Types)
	}
	sig, ok := checked.FuncSigs["get_or__T_i32"]
	if !ok {
		t.Fatalf("missing generic-struct-parameter monomorphized function: %#v", checked.FuncSigs)
	}
	if sig.ReturnType != "i32" || sig.ParamSlots != 2 || sig.ReturnSlots != 1 {
		t.Fatalf("get_or__T_i32 signature = %#v, want concrete i32 return with two params", sig)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "get_or__T_i32") {
		t.Fatalf("main did not call get_or__T_i32: %#v", mainFn.Instrs)
	}
}

func TestStableGenericCollectionSourceAPIMonomorphizesVecAndHashMap(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.collections as collections

func main() -> Int
uses alloc, mem:
    var nums: []i32 = core.make_i32(3)
    nums[0] = 7
    nums[1] = 42
    nums[2] = 5
    let vec: collections.Vec<Int> = collections.vec_from_slice(nums)
    let second: Int = collections.vec_get_or(vec, 1, 0)
    let first: Int = collections.vec_first_or(vec, 0)

    var keys: []i32 = core.make_i32(2)
    var values: []i32 = core.make_i32(2)
    keys[0] = 7
    keys[1] = 9
    values[0] = 99
    values[1] = 11
    let map: collections.HashMap<Int, Int> = collections.hash_map_from_slices(keys, values)
    let found: Int = collections.hash_map_get_i32_i32_or(map, 7, 0)

    var byte_keys: []u8 = core.make_u8(1)
    var byte_values: []i32 = core.make_i32(1)
    byte_keys[0] = 2
    byte_values[0] = 5
    let byte_map: collections.HashMap<UInt8, Int> = ` +
			`collections.hash_map_from_slices(byte_keys, byte_values)
    let byte_key: UInt8 = 2
    let byte_found: Int = collections.hash_map_get_u8_i32_or(byte_map, byte_key, 0)

    if collections.vec_len(vec) == 3 && ` +
			`collections.hash_map_len(map) == 2 && second == 42 && ` +
			`first == 7 && found == 99 && byte_found == 5:
        return 42
    return 1
`,
	})
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.t4"))
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{Root: testkit.RepoRoot(t)}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	for _, want := range []string{
		"lib.core.collections.Vec__T_i32",
		"lib.core.collections.HashMap__K_i32__V_i32",
		"lib.core.collections.HashMap__K_u8__V_i32",
	} {
		if _, ok := checked.Types[want]; !ok {
			t.Fatalf("missing monomorphized collection type %q in %#v", want, checked.Types)
		}
	}
	for _, want := range []string{
		"lib.core.collections.vec_from_slice__T_i32",
		"lib.core.collections.vec_get_or__T_i32",
		"lib.core.collections.hash_map_from_slices__K_i32__V_i32",
		"lib.core.collections.hash_map_from_slices__K_u8__V_i32",
		"lib.core.collections.hash_map_get_i32_i32_or",
		"lib.core.collections.hash_map_get_u8_i32_or",
	} {
		if _, ok := checked.FuncSigs[want]; !ok {
			t.Fatalf("missing stable generic collection function %q in %#v", want, checked.FuncSigs)
		}
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestGenericStructRejectsMissingTypeArgs(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func main() -> Int:
    let b: Box = Box<Int>{value: 42}
    return b.value
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected missing type argument diagnostic")
	}
	if !strings.Contains(err.Error(), "generic struct 'Box' requires 1 type argument") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericStructRejectsInvalidArity(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func main() -> Int:
    let b: Box<Int, Bool> = Box<Int>{value: 42}
    return b.value
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected invalid arity diagnostic")
	}
	if !strings.Contains(err.Error(), "generic struct 'Box' expects 1 type argument, got 2") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionInfersOptionalParameterElement(t *testing.T) {
	src := []byte(`
func unwrap<T>(value: T?) -> T:
    if let x = value:
        return x
    else:
        return 0

func main() -> Int:
    let value: Int? = 42
    return unwrap(value)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.FuncSigs["unwrap__T_i32"]; !ok {
		t.Fatalf("missing optional monomorphized signature: %#v", checked.FuncSigs)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericFunctionUnsupportedArgDiagnostic(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return id(unknown)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected generic inference diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot infer generic argument") {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), "v0.5") {
		t.Fatalf("generic diagnostic should be versionless: %v", err)
	}
}

func TestGenericFunctionRejectsAmbiguousReturnOnlyInference(t *testing.T) {
	src := []byte(`
func zero<T>() -> T:
    return 0

func main() -> Int:
    return zero()
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected generic ambiguity diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot infer generic argument 'T'") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionCrossModuleMonomorphizedCall(t *testing.T) {
	files := map[string]string{
		"engine/util.tetra": `module engine.util
func id<T>(x: T) -> T:
    return x
`,
		"app/main.tetra": `module app.main
import engine.util as util

func main() -> Int:
    return util.id(42)
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, ok := checked.FuncSigs["engine.util.id__T_i32"]; !ok {
		t.Fatalf("missing cross-module monomorphized signature: %#v", checked.FuncSigs)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestGenericStructCrossModuleMonomorphizedHappyPath(t *testing.T) {
	files := map[string]string{
		"engine/box.tetra": `module engine.box
pub struct Box<T>:
    value: T
`,
		"app/main.tetra": `module app.main
import engine.box as box

func main() -> Int:
    let b: box.Box<Int> = box.Box<Int>{value: 42}
    return b.value
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, ok := checked.Types["engine.box.Box__T_i32"]; !ok {
		t.Fatalf("missing cross-module monomorphized struct type: %#v", checked.Types)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestGenericFunctionMonomorphizedNamesAvoidTypeCollisions(t *testing.T) {
	files := map[string]string{
		"a.tetra": `module a
struct b_c:
    x: Int
`,
		"a_b.tetra": `module a_b
struct c:
    y: Int
`,
		"util/gen.tetra": `module util.gen
func id<T>(x: T) -> T:
    return x
`,
		"app/main.tetra": `module app.main
import util.gen as util
import a as a
import a_b as ab

func main() -> Int:
    let first: a.b_c = a.b_c{x: 1}
    let second: ab.c = ab.c{y: 2}
    let firstOut: a.b_c = util.id(first)
    let secondOut: ab.c = util.id(second)
    let x: Int = firstOut.x
    let y: Int = secondOut.y
    return x + y
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	var names []string
	for name := range checked.FuncSigs {
		if strings.HasPrefix(name, "util.gen.id__") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) != 2 {
		t.Fatalf("monomorphized util.id variants = %v, want 2 distinct variants", names)
	}
	if names[0] == names[1] {
		t.Fatalf("colliding monomorphized names: %v", names)
	}
	if names[0] != "util.gen.id__T_a_2e_b__c" || names[1] != "util.gen.id__T_a__b_2e_c" {
		t.Fatalf(
			"monomorphized util.id variants = %v, want deterministic non-colliding names",
			names,
		)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestGenericFunctionMultiTypeParametersMonomorphized(t *testing.T) {
	src := []byte(`
func choose<T, U>(left: T, right: U) -> T:
    return left

func main() -> Int:
    return choose(42, false)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	sig, ok := checked.FuncSigs["choose__T_i32__U_bool"]
	if !ok {
		t.Fatalf("missing multi-type monomorphized signature: %#v", checked.FuncSigs)
	}
	if sig.ReturnType != "i32" {
		t.Fatalf("choose__T_i32__U_bool return type = %q, want i32", sig.ReturnType)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericStructMultiTypeParametersMonomorphized(t *testing.T) {
	src := []byte(`
struct Pair<T, U>:
    left: T
    right: U

func main() -> Int:
    let p: Pair<Int, Bool> = Pair<Int, Bool>{left: 42, right: true}
    if p.right:
        return p.left
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.Types["Pair__T_i32__U_bool"]; !ok {
		t.Fatalf("missing multi-type monomorphized struct type: %#v", checked.Types)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericStructMonomorphizedNamesAvoidTypeCollisions(t *testing.T) {
	files := map[string]string{
		"a.tetra": `module a
pub struct b_c:
    x: Int
`,
		"a_b.tetra": `module a_b
pub struct c:
    y: Int
`,
		"util/box.tetra": `module util.box
pub struct Box<T>:
    value: T
`,
		"app/main.tetra": `module app.main
import a as a
import a_b as ab
import util.box as box

func main() -> Int:
    let first: box.Box<a.b_c> = box.Box<a.b_c>{value: a.b_c{x: 1}}
    let second: box.Box<ab.c> = box.Box<ab.c>{value: ab.c{y: 2}}
    return first.value.x + second.value.y
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	var names []string
	for name := range checked.Types {
		if strings.HasPrefix(name, "util.box.Box__") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	want := []string{"util.box.Box__T_a_2e_b__c", "util.box.Box__T_a__b_2e_c"}
	if len(names) != len(want) || names[0] != want[0] || names[1] != want[1] {
		t.Fatalf("monomorphized Box variants = %v, want %v", names, want)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestGenericStructOptionalFieldMonomorphized(t *testing.T) {
	src := []byte(`
struct MaybeBox<T>:
    value: T?

func main() -> Int:
    let b: MaybeBox<Int> = MaybeBox<Int>{value: none}
    if let x = b.value:
        return x
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	info, ok := checked.Types["MaybeBox__T_i32"]
	if !ok {
		t.Fatalf("missing optional generic struct type: %#v", checked.Types)
	}
	field := info.FieldMap["value"]
	if field.TypeName != "i32?" {
		t.Fatalf("MaybeBox__T_i32.value type = %q, want i32?", field.TypeName)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericStructNestedGenericFieldExplicitDiagnostic(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

struct Outer<T>:
    inner: Box<T>

func main() -> Int:
    let outer: Outer<Int> = Outer<Int>{inner: Box<Int>{value: 42}}
    return outer.inner.value
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected nested generic struct diagnostic")
	}
	if !strings.Contains(err.Error(), "nested generic struct instantiation") ||
		!strings.Contains(err.Error(), "Outer__T_i32.inner") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionReturnOnlyInferenceDiagnosticStable(t *testing.T) {
	src := []byte(`
func make<T>() -> T:
    return 0

func main() -> Int:
    return make()
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected return-only inference diagnostic")
	}
	for _, want := range []string{"line 6:12", "cannot infer generic argument 'T' for 'make'"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
	if strings.Contains(err.Error(), "v0.") || strings.Contains(err.Error(), "MVP") {
		t.Fatalf("return-only inference diagnostic should remain stable and versionless: %v", err)
	}
}

func TestGenericFunctionDuplicateRecursiveWorkMonomorphizesOnce(t *testing.T) {
	src := []byte(`
func down<T>(x: T, n: Int) -> T:
    if n == 0:
        return x
    return down(x, n - 1)

func main() -> Int:
    return down(42, 2)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	count := 0
	for name := range checked.FuncSigs {
		if name == "down__T_i32" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("down__T_i32 signatures = %d, want 1", count)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericStructFunctionTypeArgumentRejected(t *testing.T) {
	src := []byte(`
struct Holder<T>:
    cb: T

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let holder: Holder<fn(Int) -> Int> = Holder<fn(Int) -> Int>{cb: add1}
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected function type argument diagnostic")
	}
	want := ("generic struct 'Holder' type argument 'T' uses function " +
		"type; generic struct instantiation cannot carry " +
		"function-typed values under the supported fnptr ABI")
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v", err)
	}
}

// ---- interface_test.go ----

func TestGenerateInterfaceFromSourceWritesT4IStubs(t *testing.T) {
	src := []byte(`module math.core

struct Point:
    x: Int
    y: Int

func add(a: Int, b: Int) -> Int:
    return a + b

func enabled() -> Bool:
    return true
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"module math.core",
		"struct Point:",
		"func add(a: i32, b: i32) -> i32:",
		"    return 0",
		"func enabled() -> bool:",
		"    return false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
}

func TestGenerateInterfaceFromSourcePreservesFunctionTypedParameterReturnStub(t *testing.T) {
	src := []byte(`module lib.identity

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/identity.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"func identity(f: fn(i32) -> i32) -> fn(i32) -> i32:",
		"    return f",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "return 0") {
		t.Fatalf("function-typed parameter-return interface stub fell back to return 0:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourcePreservesBorrowedReturnContract(t *testing.T) {
	src := []byte(`module lib.views

pub func view(xs: borrow []u8) -> borrow []u8:
    return xs.window(0, 1).borrow()
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/views.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"func view(xs: borrow []u8) -> borrow []u8:",
		"// tetra-interface-lifetime: return=borrow source=xs provenance=param lifetime=call",
		"    return xs",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}

	iface, err := compiler.ParseFile(out, "lib/views.t4i")
	if err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, out)
	}
	app, err := compiler.ParseFile([]byte(`module app.main
import lib.views as views

func relay(xs: borrow []u8) -> borrow []u8:
    return views.view(xs)

func main() -> Int:
    return 0
`), "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile app: %v", err)
	}
	checked, err := compiler.CheckWorld(&compiler.World{
		EntryModule:      "app.main",
		Files:            []*compiler.FileAST{iface, app},
		InterfaceModules: map[string]bool{"lib.views": true},
		ByModule: map[string]*compiler.FileAST{
			"lib.views": iface,
			"app.main":  app,
		},
	})
	if err != nil {
		t.Fatalf("CheckWorld: %v\ninterface:\n%s", err, out)
	}
	if got := checked.FuncSigs["lib.views.view"].ReturnOwnership; got != "borrow" {
		t.Fatalf("imported view ReturnOwnership = %q, want borrow; interface:\n%s", got, out)
	}
}

func TestInterfaceFingerprintTracksBorrowedReturnLifetimeSource(t *testing.T) {
	srcA := []byte(`module lib.views

pub func choose(a: borrow []u8, b: borrow []u8) -> borrow []u8:
    return a.borrow()
`)
	srcB := []byte(strings.Replace(string(srcA), "return a.borrow()", "return b.borrow()", 1))

	hashA, err := compiler.InterfaceFingerprintFromSource(srcA, "lib/views.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource A: %v", err)
	}
	hashB, err := compiler.InterfaceFingerprintFromSource(srcB, "lib/views.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource B: %v", err)
	}
	if hashA == hashB {
		t.Fatalf("borrowed return lifetime source did not affect interface hash: %s", hashA)
	}
}

func TestInterfaceFingerprintRejectsTamperedBorrowedReturnLifetimeMetadata(t *testing.T) {
	src := []byte(`module lib.views

pub func view(xs: borrow []u8) -> borrow []u8:
    return xs.borrow()
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "lib/views.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	tampered := strings.Replace(string(iface), "source=xs", "source=ys", 1)
	if tampered == string(iface) {
		t.Fatalf("test fixture did not find borrowed return lifetime metadata:\n%s", iface)
	}
	_, err = compiler.InterfaceFingerprintFromT4I([]byte(tampered))
	if err == nil || !strings.Contains(err.Error(), "invalid .t4i hash") {
		t.Fatalf("InterfaceFingerprintFromT4I tampered error = %v, want invalid .t4i hash", err)
	}
}

func TestGenerateInterfaceFromSourcePreservesReturnedFunctionHandleMetadata(t *testing.T) {
	src := []byte(`module lib.maker

pub func make() -> fn(Int) -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    return fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/maker.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	maker, err := compiler.ParseFile(out, "lib/maker.t4i")
	if err != nil {
		t.Fatalf("ParseFile maker interface: %v\n%s", err, out)
	}
	app, err := compiler.ParseFile([]byte(`module app.main
import lib.maker as maker

func main() -> Int:
    let cb: fn(Int) -> Int = maker.make()
    return cb(-3)
`), "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile app: %v", err)
	}
	checked, err := compiler.CheckWorld(&compiler.World{
		EntryModule:      "app.main",
		Files:            []*compiler.FileAST{maker, app},
		InterfaceModules: map[string]bool{"lib.maker": true},
		ByModule: map[string]*compiler.FileAST{
			"lib.maker": maker,
			"app.main":  app,
		},
	})
	if err != nil {
		t.Fatalf("CheckWorld: %v\ninterface:\n%s", err, out)
	}
	makeSig := checked.FuncSigs["lib.maker.make"]
	if makeSig.ReturnFunctionSymbol == "" {
		t.Fatalf("make ReturnFunctionSymbol empty; interface:\n%s", out)
	}
	if got := len(makeSig.ReturnFunctionCaptures); got != 9 {
		t.Fatalf(
			"make ReturnFunctionCaptures = %d, want 9; sig=%#v\ninterface:\n%s",
			got,
			makeSig,
			out,
		)
	}
	if string(makeSig.ReturnFunctionEscapeKind) != "heap" || !makeSig.ReturnFunctionHandleValue ||
		makeSig.ReturnSlots != 4 {
		t.Fatalf(
			("make returned handle metadata = (%q, %v, slots=%d), want " +
				"(heap, true, 4); sig=%#v\ninterface:\n%s"),
			makeSig.ReturnFunctionEscapeKind,
			makeSig.ReturnFunctionHandleValue,
			makeSig.ReturnSlots,
			makeSig,
			out,
		)
	}
	foundMain := false
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.main" {
			foundMain = true
			cb := fn.Locals["cb"]
			captureCount := len(cb.FunctionCaptures) + len(cb.FunctionEscapeCaptures)
			if captureCount != 9 || string(cb.FunctionEscapeKind) != "heap" ||
				!cb.FunctionHandleValue ||
				cb.SlotCount != 4 {
				t.Fatalf(
					("local cb metadata = captures:%d direct:%d escape-captures:" +
						"%d escape:%q handle:%v slots:%d; want " +
						"9/heap/true/4\ninterface:\n%s"),
					captureCount,
					len(cb.FunctionCaptures),
					len(cb.FunctionEscapeCaptures),
					cb.FunctionEscapeKind,
					cb.FunctionHandleValue,
					cb.SlotCount,
					out,
				)
			}
			break
		}
	}
	if !foundMain {
		t.Fatalf("checked funcs missing app.main.main")
	}
}

func TestGenerateInterfaceFromSourcePreservesFunctionTypedStructFieldReturnStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub func pick(holder: Holder) -> fn(Int) -> Int:
    return holder.cb
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"struct Holder:",
		"    cb: fn(i32) -> i32",
		"func pick(holder: Holder) -> fn(i32) -> i32:",
		"    return holder.cb",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "return fn(") || strings.Contains(text, "return 0") {
		t.Fatalf(
			"function-typed struct-field-return interface stub lost field return metadata:\n%s",
			text,
		)
	}
}

func TestGenerateInterfaceFromSourcePreservesFunctionTypedNestedStructFieldReturnStub(
	t *testing.T,
) {
	src := []byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func pick(box: Box) -> fn(Int) -> Int:
    return box.holder.cb
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"struct Holder:",
		"    cb: fn(i32) -> i32",
		"struct Box:",
		"    holder: Holder",
		"func pick(box: Box) -> fn(i32) -> i32:",
		"    return box.holder.cb",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "return fn(") || strings.Contains(text, "return 0") {
		t.Fatalf(
			"function-typed nested-struct-field-return interface stub lost field return metadata:\n%s",
			text,
		)
	}
}

func TestGenerateInterfaceFromSourcePreservesFunctionTypedStructParameterWholeReturnStub(
	t *testing.T,
) {
	src := []byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func echo(box: Box) -> Box:
    return box
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"struct Holder:",
		"    cb: fn(i32) -> i32",
		"struct Box:",
		"    holder: Holder",
		"func echo(box: Box) -> Box:",
		"    return box",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "return 0") {
		t.Fatalf(
			("function-typed struct-parameter whole-return interface stub " +
				"lost parameter return metadata:\n%s"),
			text,
		)
	}
}

func TestGenerateInterfaceFromSourcePreservesFunctionTypedEnumParameterWholeReturnStub(
	t *testing.T,
) {
	src := []byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func echo(choice: MaybeCallback) -> MaybeCallback:
    return choice
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"enum MaybeCallback:",
		"    case some(fn(i32) -> i32)",
		"    case empty",
		"func echo(choice: MaybeCallback) -> MaybeCallback:",
		"    return choice",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "return 0") {
		t.Fatalf(
			"function-typed enum-parameter whole-return interface stub lost parameter return metadata:\n%s",
			text,
		)
	}
}

func TestGenerateInterfaceFromSourcePreservesReturnedAggregateClosureStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    ))
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"enum MaybeCallback:",
		"    case some(fn(i32) -> i32)",
		"struct Box:",
		"    choice: MaybeCallback",
		"func makeBox() -> Box:",
		"    return Box(choice: MaybeCallback.some(fn(p0: i32) -> i32 = 0))",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "    return 0") {
		t.Fatalf("returned aggregate interface stub lost closure payload metadata:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourcePreservesReturnedEnumClosureStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"enum MaybeCallback:",
		"    case some(fn(i32) -> i32)",
		"func makeChoice() -> MaybeCallback:",
		"    return MaybeCallback.some(fn(p0: i32) -> i32 = 0)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "    return 0") {
		t.Fatalf("returned enum interface stub lost closure payload metadata:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourcePreservesReturnedThrowingAggregateClosureStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"enum Boom:",
		"enum MaybeCallback:",
		"    case some(fn(i32) -> i32 throws Boom)",
		"struct Box:",
		"    choice: MaybeCallback",
		"func makeBox() -> Box:",
		"    return Box(choice: MaybeCallback.some(fn(p0: i32) -> i32 throws Boom = 0))",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "    return 0") {
		t.Fatalf(
			"returned throwing aggregate interface stub lost closure payload metadata:\n%s",
			text,
		)
	}
}

func TestGenerateInterfaceFromSourcePreservesReturnedThrowingStructFieldClosureStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"enum Boom:",
		"struct Holder:",
		"    cb: fn(i32) -> i32 throws Boom",
		"func makeHolder() -> Holder:",
		"    return Holder(cb: fn(p0: i32) -> i32 throws Boom = 0)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "    return 0") {
		t.Fatalf(
			"returned throwing struct-field interface stub lost closure field metadata:\n%s",
			text,
		)
	}
}

func TestGenerateInterfaceFromSourcePreservesReturnedThrowingEnumClosureStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"enum Boom:",
		"enum MaybeCallback:",
		"    case some(fn(i32) -> i32 throws Boom)",
		"func makeChoice() -> MaybeCallback:",
		"    return MaybeCallback.some(fn(p0: i32) -> i32 throws Boom = 0)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "    return 0") {
		t.Fatalf("returned throwing enum interface stub lost closure payload metadata:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourceFiltersPrivateSurfaceAndHashesPublicAPI(t *testing.T) {
	src := []byte(`module math.core

import hidden.impl as impl
pub import public.types.{Vec}

pub struct Point:
    x: Int
    y: Int

struct Secret:
    value: Int

pub func add(a: Int, b: Int) -> Int:
    return a + b

func hidden() -> Int:
    return 99
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"// t4i-hash: sha256:",
		"pub import public.types.{Vec}",
		"pub struct Point:",
		"pub func add(a: i32, b: i32) -> i32:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	for _, leak := range []string{"hidden.impl", "struct Secret", "func hidden"} {
		if strings.Contains(text, leak) {
			t.Fatalf("interface leaked %q:\n%s", leak, text)
		}
	}

	out2, err := compiler.GenerateInterfaceFromSource(
		[]byte(strings.Replace(string(src), "return 99", "return 100", 1)),
		"math/core.t4",
	)
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource second: %v", err)
	}
	if string(out2) != text {
		t.Fatalf(
			"private body-only change should not change interface hash\nbefore:\n%s\nafter:\n%s",
			text,
			out2,
		)
	}
}

func TestInterfaceFingerprintFromSourceIsPublicAPIStable(t *testing.T) {
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b

func hidden() -> Int:
    return 1
`)
	hash1, err := compiler.InterfaceFingerprintFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	privateBodyChanged := []byte(strings.Replace(string(src), "return 1", "return 2", 1))
	hash2, err := compiler.InterfaceFingerprintFromSource(privateBodyChanged, "math/core.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource private change: %v", err)
	}
	if hash1 != hash2 {
		t.Fatalf("private implementation change changed public API hash: %s vs %s", hash1, hash2)
	}
	publicSigChanged := []byte(strings.Replace(string(src), "b: Int", "b: Bool", 1))
	hash3, err := compiler.InterfaceFingerprintFromSource(publicSigChanged, "math/core.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource public change: %v", err)
	}
	if hash1 == hash3 {
		t.Fatalf("public signature change did not change API hash: %s", hash1)
	}
}

func TestValidateInterfaceAgainstSourceReportsPublicAPIMismatch(t *testing.T) {
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	changedSource := []byte(strings.Replace(string(src), "b: Int", "b: Bool", 1))
	err = compiler.ValidateInterfaceAgainstSource(changedSource, iface, "math/core.t4")
	if err == nil {
		t.Fatalf("expected public API mismatch")
	}
	if !strings.Contains(err.Error(), "public API mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateInterfaceFromSourceKeepsImportsRequiredByPublicAPI(t *testing.T) {
	src := []byte(`module math.core

import math.types as mt
import hidden.impl as hidden

pub func norm(v: mt.Vec) -> Int:
    return v.x

func private_helper(v: hidden.Secret) -> Int:
    return 0
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	if !strings.Contains(text, "import math.types as mt") {
		t.Fatalf("interface omitted public-signature import:\n%s", text)
	}
	if strings.Contains(text, "hidden.impl") {
		t.Fatalf("interface leaked private-only import:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourceEmitsTypecheckableExtensionDeclarations(t *testing.T) {
	src := []byte(`module engine.vec

pub struct Vec2:
    x: Int
    y: Int

pub extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x + self.y
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "engine/vec.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	for _, want := range []string{
		"pub extension Vec2:",
		"func sum(self: Vec2) -> i32:",
		"    return 0",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}

	ifaceFile, err := compiler.ParseFile(iface, "engine/vec.t4i")
	if err != nil {
		t.Fatalf("ParseFile interface: %v\n%s", err, text)
	}
	ifaceFile.InterfaceHash, err = compiler.InterfaceFingerprintFromT4I(iface)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromT4I: %v", err)
	}
	app, err := compiler.ParseFile([]byte(`module app.main
import engine.vec as vec

func main() -> Int:
    let v: vec.Vec2 = vec.Vec2(x: 40, y: 2)
    return vec.Vec2.sum(v)
`), "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile app: %v", err)
	}
	world := &compiler.World{
		EntryModule: app.Module,
		Files:       []*compiler.FileAST{ifaceFile, app},
		ByModule: map[string]*compiler.FileAST{
			ifaceFile.Module: ifaceFile,
			app.Module:       app,
		},
		InterfaceModules: map[string]bool{ifaceFile.Module: true},
		InterfaceHashes:  map[string]string{ifaceFile.Module: ifaceFile.InterfaceHash},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld generated interface extension: %v\ninterface:\n%s", err, text)
	}
	if _, ok := checked.FuncSigs["engine.vec.Vec2.sum"]; !ok {
		t.Fatalf(
			"missing extension method signature from generated interface: %#v\ninterface:\n%s",
			checked.FuncSigs,
			text,
		)
	}
}

func TestGenerateInterfaceFromSourceEmitsProtocolImplDeclarationsBeforeFunctions(t *testing.T) {
	src := []byte(`module engine.core

pub struct Vec2:
    x: Int

pub protocol Echoable:
    func echo(self: Vec2) -> Vec2

pub extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: Echoable

pub func id<T: Echoable>(x: T) -> T:
    return x
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "engine/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	for _, want := range []string{
		"pub extension Vec2:",
		"impl Vec2: Echoable",
		"pub func id<T: Echoable>(x: T) -> T:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Index(
		text,
		"impl Vec2: Echoable",
	) > strings.Index(
		text,
		"pub func id<T: Echoable>",
	) {
		t.Fatalf("interface emitted impl after functions:\n%s", text)
	}

	ifaceFile, err := compiler.ParseFile(iface, "engine/core.t4i")
	if err != nil {
		t.Fatalf("ParseFile interface: %v\n%s", err, text)
	}
	ifaceFile.InterfaceHash, err = compiler.InterfaceFingerprintFromT4I(iface)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromT4I: %v", err)
	}
	app, err := compiler.ParseFile([]byte(`module app.main
import engine.core as core

func main() -> Int:
    let v: core.Vec2 = core.Vec2(x: 42)
    let out: core.Vec2 = core.id(v)
    return out.x
`), "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile app: %v", err)
	}
	world := &compiler.World{
		EntryModule: app.Module,
		Files:       []*compiler.FileAST{ifaceFile, app},
		ByModule: map[string]*compiler.FileAST{
			ifaceFile.Module: ifaceFile,
			app.Module:       app,
		},
		InterfaceModules: map[string]bool{ifaceFile.Module: true},
		InterfaceHashes:  map[string]string{ifaceFile.Module: ifaceFile.InterfaceHash},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld generated interface impl: %v\ninterface:\n%s", err, text)
	}
	if _, ok := checked.FuncSigs["engine.core.id__T_engine_2e_core_2e_Vec2"]; !ok {
		t.Fatalf(
			"missing protocol-bound monomorphized signature from generated interface: %#v\ninterface:\n%s",
			checked.FuncSigs,
			text,
		)
	}
}

func TestGenerateInterfaceFromSourceKeepsImportsRequiredOnlyByImpls(t *testing.T) {
	src := []byte(`module app.impls

import engine.core as core

impl core.Vec2: core.Renderable

pub func marker() -> Int:
    return 1
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/impls.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	for _, want := range []string{
		"import engine.core as core",
		"impl core.Vec2: core.Renderable",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing impl-only import surface %q:\n%s", want, text)
		}
	}
	if _, err := compiler.ParseFile(iface, "app/impls.t4i"); err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
}

func TestGenerateInterfaceFromSourceKeepsImportsRequiredOnlyByGenericBounds(t *testing.T) {
	src := []byte(`module app.generics

import engine.core as core

pub func id<T: core.Echoable>(x: T) -> T:
    return x
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/generics.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	for _, want := range []string{
		"import engine.core as core",
		"pub func id<T: core.Echoable>(x: T) -> T:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing generic-bound import surface %q:\n%s", want, text)
		}
	}
	if _, err := compiler.ParseFile(iface, "app/generics.t4i"); err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
}

func TestGenerateInterfaceFromSourcePreservesGenericStructTypeArgsAndImports(t *testing.T) {
	src := []byte(`module app.boxes

import engine.core as core

pub struct Box<T>:
    value: T

pub func wrap(box: Box<core.Vec2>) -> Box<core.Vec2>:
    return box
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/boxes.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	for _, want := range []string{
		"import engine.core as core",
		"pub struct Box<T>:",
		"pub func wrap(box: Box<core.Vec2>) -> Box<core.Vec2>:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing generic type-arg surface %q:\n%s", want, text)
		}
	}
	if _, err := compiler.ParseFile(iface, "app/boxes.t4i"); err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
}

func TestGeneratedInterfaceGenericStructTypeArgsCheckAndLowerAcrossModules(t *testing.T) {
	core, err := compiler.ParseFile([]byte(`module engine.core

pub struct Vec2:
    x: Int
`), "engine/core.t4")
	if err != nil {
		t.Fatalf("ParseFile core: %v", err)
	}
	src := []byte(`module app.boxes

import engine.core as core

pub struct Box<T>:
    value: T

pub func wrap(box: Box<core.Vec2>) -> Box<core.Vec2>:
    return box
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/boxes.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	boxes, err := compiler.ParseFile(iface, "app/boxes.t4i")
	if err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
	boxes.InterfaceHash, err = compiler.InterfaceFingerprintFromT4I(iface)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromT4I: %v", err)
	}
	app, err := compiler.ParseFile([]byte(`module app.main

import app.boxes as boxes
import engine.core as core

func main() -> Int:
    let v: boxes.Box<core.Vec2> = boxes.wrap(boxes.Box<core.Vec2>{value: core.Vec2{x: 42}})
    return v.value.x
`), "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile app: %v", err)
	}
	world := &compiler.World{
		EntryModule: app.Module,
		Files:       []*compiler.FileAST{core, boxes, app},
		ByModule: map[string]*compiler.FileAST{
			core.Module:  core,
			boxes.Module: boxes,
			app.Module:   app,
		},
		InterfaceModules: map[string]bool{boxes.Module: true},
		InterfaceHashes:  map[string]string{boxes.Module: boxes.InterfaceHash},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld generated interface generic struct: %v\ninterface:\n%s", err, text)
	}
	if _, ok := checked.FuncSigs["app.boxes.wrap"]; !ok {
		t.Fatalf(
			"missing generated interface function signature: %#v\ninterface:\n%s",
			checked.FuncSigs,
			text,
		)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules generated interface generic struct: %v\ninterface:\n%s", err, text)
	}
}

func TestGenerateInterfaceFromSourceKeepsImportsRequiredByFunctionTypeRefs(t *testing.T) {
	src := []byte(`module app.callbacks

import engine.core as core

` + `pub func install(cb: fn(core.Vec2) -> core.Vec2 throws core.Boom) -> ` +
		`fn(core.Vec2) -> core.Vec2 throws core.Boom:
    return cb
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	for _, want := range []string{
		"import engine.core as core",
		("pub func install(cb: fn(core.Vec2) -> core.Vec2 throws " +
			"core.Boom) -> fn(core.Vec2) -> core.Vec2 throws core.Boom:"),
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing function-type import surface %q:\n%s", want, text)
		}
	}
	if _, err := compiler.ParseFile(iface, "app/callbacks.t4i"); err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
}

func TestGeneratedInterfaceFunctionTypeRefsCheckAndLowerAcrossModules(t *testing.T) {
	core, err := compiler.ParseFile([]byte(`module engine.core

pub struct Vec2:
    x: Int
`), "engine/core.t4")
	if err != nil {
		t.Fatalf("ParseFile core: %v", err)
	}
	src := []byte(`module app.callbacks

import engine.core as core

pub func install(cb: fn(core.Vec2) -> core.Vec2) -> fn(core.Vec2) -> core.Vec2:
    return cb
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	callbacks, err := compiler.ParseFile(iface, "app/callbacks.t4i")
	if err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
	callbacks.InterfaceHash, err = compiler.InterfaceFingerprintFromT4I(iface)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromT4I: %v", err)
	}
	app, err := compiler.ParseFile([]byte(`module app.main

import app.callbacks as callbacks
import engine.core as core

func echo(v: core.Vec2) -> core.Vec2:
    return v

func main() -> Int:
    let cb: fn(core.Vec2) -> core.Vec2 = callbacks.install(echo)
    let out: core.Vec2 = cb(core.Vec2{x: 42})
    return out.x
`), "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile app: %v", err)
	}
	world := &compiler.World{
		EntryModule: app.Module,
		Files:       []*compiler.FileAST{core, callbacks, app},
		ByModule: map[string]*compiler.FileAST{
			core.Module:      core,
			callbacks.Module: callbacks,
			app.Module:       app,
		},
		InterfaceModules: map[string]bool{callbacks.Module: true},
		InterfaceHashes:  map[string]string{callbacks.Module: callbacks.InterfaceHash},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld generated interface function type refs: %v\ninterface:\n%s", err, text)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf(
			"LowerModules generated interface function type refs: %v\ninterface:\n%s",
			err,
			text,
		)
	}
}

func TestGenerateInterfaceFromSourcePreservesProtocolRequirementTypeParams(t *testing.T) {
	src := []byte(`module app.protocols

pub protocol Mapper:
    func map<T>(self: Int, value: T) -> T
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/protocols.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	want := "func map<T>(self: i32, value: T) -> T"
	if !strings.Contains(text, want) {
		t.Fatalf("interface missing protocol requirement type params %q:\n%s", want, text)
	}
	if _, err := compiler.ParseFile(iface, "app/protocols.t4i"); err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
}

func TestInterfaceFingerprintFromSourceTracksHashOnlyPublicSurface(t *testing.T) {
	src := []byte(`module app.config

pub const build: Int = 1
`)
	hash1, err := compiler.InterfaceFingerprintFromSource(src, "app/config.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	hash2, err := compiler.InterfaceFingerprintFromSource(
		[]byte(strings.Replace(string(src), "build: Int", "build: Bool", 1)),
		"app/config.t4",
	)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource changed: %v", err)
	}
	if hash1 == hash2 {
		t.Fatalf("public hash-only global surface change did not change API hash: %s", hash1)
	}
}

// ---- optional_match_test.go ----

func TestOptionalMatchNoneCheckAndLower(t *testing.T) {
	src := []byte(`
func maybe(flag: Bool) -> Int?:
    if flag:
        return 42
    else:
        return none

func main() -> Int:
    let value: Int? = maybe(false)
    match value:
    case none:
        return 42
    case _:
        return 1
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestOptionalMatchRejectsNonNonePattern(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let value: Int? = 1
    match value:
    case 1:
        return 1
    case _:
        return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected optional match pattern error")
	}
	if !strings.Contains(err.Error(), "optional match supports only 'none'") {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), "v0.") {
		t.Fatalf("optional match diagnostic should be versionless: %v", err)
	}
}

func TestOptionalMatchSomeBindingCheckAndLower(t *testing.T) {
	src := []byte(`
func maybe(flag: Bool) -> Int?:
    if flag:
        return 42
    else:
        return none

func main() -> Int:
    let value: Int? = maybe(true)
    match value:
    case some(x):
        return x
    case none:
        return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.Funcs[1].Locals["x"].TypeName; got != "i32" {
		t.Fatalf("some binding type = %q, want i32", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestEnumExhaustiveMatchNoDefaultCheckAndLower(t *testing.T) {
	src := []byte(`
enum Color:
    case red
    case green

func main() -> Int:
    let color: Color = Color.green
    match color:
    case Color.red:
        return 1
    case Color.green:
        return 42
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestBuildOptionalMatchNoneSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func maybe(flag: Bool) -> Int?:
    if flag:
        return 7
    else:
        return none

func main() -> Int:
    let value: Int? = maybe(false)
    match value:
    case none:
        return 42
    case _:
        return 1
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildOptionalMatchSomeSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func maybe(flag: Bool) -> Int?:
    if flag:
        return 42
    else:
        return none

func main() -> Int:
    let value: Int? = maybe(true)
    match value:
    case some(x):
        return x
    case none:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

// ---- optionals_test.go ----

func TestOptionalNoneEqualityLowers(t *testing.T) {
	src := []byte(`
func maybe() -> Int?:
    return none

func main() -> Int:
    let value: Int? = maybe()
    if value == none:
        return 0
    else:
        return 1
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if checked.FuncSigs["maybe"].ReturnSlots != 2 {
		t.Fatalf("maybe return slots = %d, want 2", checked.FuncSigs["maybe"].ReturnSlots)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalIfLetLowers(t *testing.T) {
	src := []byte(`
func unwrap(value: Int?) -> Int:
    if let x = value:
        return x
    else:
        return 0

func main() -> Int:
    return unwrap(none)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.Funcs[0].Locals["x"].TypeName; got != "i32" {
		t.Fatalf("if-let local type = %q, want i32", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalIfLetSomePatternCheckAndLower(t *testing.T) {
	src := []byte(`
func unwrap(value: Int?) -> Int:
    if let some(x) = value:
        return x
    else:
        return 0

func main() -> Int:
    return unwrap(41)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.Funcs[0].Locals["x"].TypeName; got != "i32" {
		t.Fatalf("some binding type = %q, want i32", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalIfLetNonePatternCheckAndLower(t *testing.T) {
	src := []byte(`
func score(value: Int?) -> Int:
    if let none = value:
        return 7
    else:
        return 1

func main() -> Int:
    return score(none)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalIfLetPatternRejectsNonOptionalValue(t *testing.T) {
	src := []byte(`
func main() -> Int:
    if let some(x) = 1:
        return x
    else:
        return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected if-let pattern type error")
	}
	if !strings.Contains(err.Error(), "if let pattern requires optional or enum value") {
		t.Fatalf("error = %v", err)
	}
}

func TestOptionalImplicitSomeReturnAndLetLower(t *testing.T) {
	src := []byte(`
func maybe() -> Int?:
    return 42

func main() -> Int:
    let value: Int? = 7
    if value != none:
        return 0
    else:
        return 1
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalSmallIntLiteralPayloadsCheckAndLower(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
func main() -> Int:
    let byte: UInt8? = 255
    let word: UInt16? = 65535
    var assigned_byte: UInt8? = none
    assigned_byte = 255
    var assigned_word: UInt16? = none
    assigned_word = 65535
    return 0
`)
}

func TestOptionalSmallIntLiteralPayloadsRejectOutOfRange(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "u8 local initializer",
			src: `
func main() -> Int:
    let maybe: UInt8? = 300
    return 0
`,
			want: "type mismatch: expected 'u8?', got 'i32'",
		},
		{
			name: "u16 local initializer",
			src: `
func main() -> Int:
    let maybe: UInt16? = 70000
    return 0
`,
			want: "type mismatch: expected 'u16?', got 'i32'",
		},
		{
			name: "u8 assignment",
			src: `
func main() -> Int:
    var maybe: UInt8? = none
    maybe = 300
    return 0
`,
			want: "type mismatch: expected 'u8?', got 'i32'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.want)
		})
	}
}

func TestNestedOptionalLiteralPayloadsCheckAndLower(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
func main() -> Int:
    let nested: Int?? = 42
    var assigned: Int?? = none
    assigned = 42
    let inner: Int? = 42
    let from_inner: Int?? = inner
    return 0
`)
}

func TestNestedOptionalReturnPayloadCheckAndLower(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
func make_nested() -> Int??:
    return 42

func main() -> Int:
    let nested: Int?? = make_nested()
    return 0
`)
}

func TestNestedOptionalSmallIntLiteralPayloadsRejectOutOfRange(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "nested u8 initializer",
			src: `
func main() -> Int:
    let maybe: UInt8?? = 300
    return 0
`,
			want: "type mismatch: expected 'u8??', got 'i32'",
		},
		{
			name: "nested u16 assignment",
			src: `
func main() -> Int:
    var maybe: UInt16?? = none
    maybe = 70000
    return 0
`,
			want: "type mismatch: expected 'u16??', got 'i32'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.want)
		})
	}
}

func TestOptionalAllowsMultiSlotPayload(t *testing.T) {
	src := []byte(`
func maybe(flag: Bool) -> String?:
    if flag:
        return "ok"
    else:
        return none

func length(value: String?) -> Int:
    if let s = value:
        return s.len
    else:
        return 0

func main() -> Int:
    return length(maybe(true))
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.FuncSigs["maybe"].ReturnSlots; got != 3 {
		t.Fatalf("maybe return slots = %d, want 3", got)
	}
	if got := checked.FuncSigs["length"].ParamSlots; got != 3 {
		t.Fatalf("length param slots = %d, want 3", got)
	}
	if got := checked.Funcs[1].Locals["s"].TypeName; got != "str" {
		t.Fatalf("if-let local type = %q, want str", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalMatchExhaustiveNoDefaultWithMultiSlotPayload(t *testing.T) {
	src := []byte(`
func maybe(flag: Bool) -> String?:
    if flag:
        return "ok"
    else:
        return none

func main() -> Int:
    let value: String? = maybe(true)
    match value:
    case some(s):
        return s.len
    case none:
        return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.Funcs[1].Locals["s"].TypeName; got != "str" {
		t.Fatalf("some binding type = %q, want str", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalMatchMissingSomeCaseNeedsReturn(t *testing.T) {
	src := []byte(`
func maybe(flag: Bool) -> String?:
    if flag:
        return "ok"
    else:
        return none

func main() -> Int:
    let value: String? = maybe(true)
    match value:
    case none:
        return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected non-exhaustive optional match error")
	}
	if !strings.Contains(err.Error(), "must end with return") {
		t.Fatalf("error = %v", err)
	}
}

func TestOptionalStructPayloadIfLetAndMatchLower(t *testing.T) {
	src := []byte(`
struct Pair:
    x: Int
    y: Int

func maybe(flag: Bool) -> Pair?:
    if flag:
        return Pair(x: 20, y: 22)
    else:
        return none

func unwrap_if(value: Pair?) -> Int:
    if let p = value:
        return p.x + p.y
    else:
        return 0

func unwrap_match(value: Pair?) -> Int:
    match value:
    case some(p):
        return p.x + p.y
    case none:
        return 0

func main() -> Int:
    return unwrap_if(maybe(true)) + unwrap_match(maybe(false))
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.FuncSigs["maybe"].ReturnSlots; got != 3 {
		t.Fatalf("maybe return slots = %d, want 3", got)
	}
	if got := checked.Funcs[1].Locals["p"].TypeName; got != "Pair" {
		t.Fatalf("if-let payload type = %q, want Pair", got)
	}
	if got := checked.Funcs[2].Locals["p"].TypeName; got != "Pair" {
		t.Fatalf("match payload type = %q, want Pair", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalNarrowingBindingsDoNotEscapeCaseScope(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if let",
			src: `
func main() -> Int:
    let value: Int? = 1
    if let x = value:
        let y: Int = x
    return x
`,
		},
		{
			name: "match some",
			src: `
func main() -> Int:
    let value: Int? = 1
    match value:
    case some(x):
        let y: Int = x
    case none:
        let z: Int = 0
    return x
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected narrowing binding scope error")
			}
			if !strings.Contains(err.Error(), "out of scope") &&
				!strings.Contains(err.Error(), "unknown identifier") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

// ---- protocol_conformance_test.go ----

func TestProtocolConformanceChecksExtensionMethod(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Renderable

func main() -> Int:
    return Vec2.draw(Vec2(x: 42))
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Impls) != 1 {
		t.Fatalf("impls = %d", len(prog.Impls))
	}
	if _, err := compiler.Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestProtocolConformanceChecksThrowingExtensionMethod(t *testing.T) {
	src := []byte(`
enum DrawError:
    case failed

struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int throws DrawError

extension Vec2:
    func draw(self: Vec2) -> Int throws DrawError:
        if self.x == 0:
            throw DrawError.failed
        return self.x

impl Vec2: Renderable

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["Vec2.draw"].ThrowsType; got != "DrawError" {
		t.Fatalf("Vec2.draw throws = %q, want DrawError", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestProtocolConformanceRejectsThrowingRequirementMismatch(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

enum DrawError:
    case failed

protocol Renderable:
    func draw(self: Vec2) -> Int throws DrawError

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Renderable

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected throws conformance error")
	}
	if !strings.Contains(err.Error(), "throws type differs") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceReportsMissingMethod(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

impl Vec2: Renderable

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected conformance error")
	}
	if !strings.Contains(err.Error(), "missing protocol requirement 'draw'") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceSupportsGenericRequirementMVP(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Mapper:
    func map<T>(self: Vec2, value: T) -> T

extension Vec2:
    func map<T>(self: Vec2, value: T) -> T:
        return value

impl Vec2: Mapper

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := compiler.Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestProtocolConformanceRejectsGenericRequirementCountMismatch(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Mapper:
    func map<T>(self: Vec2, value: T) -> T

extension Vec2:
    func map(self: Vec2, value: Int) -> Int:
        return value

impl Vec2: Mapper

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected conformance error")
	}
	if !strings.Contains(err.Error(), "generic parameter count differs") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceRejectsUndeclaredGenericTypeInRequirement(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Mapper:
    func map<T>(self: Vec2, value: U) -> U

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected requirement signature error")
	}
	if !strings.Contains(err.Error(), "unknown type 'U'") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceViaImportedExtensionClause(t *testing.T) {
	files := map[string]string{
		"engine/core.tetra": `module engine.core
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int
`,
		"app/ext.tetra": `module app.ext
import engine.core as core

extension core.Vec2:
    func draw(self: core.Vec2) -> Int:
        return self.x

impl core.Vec2: core.Renderable
`,
		"app/main.tetra": `module app.main
import app.ext as ext
import engine.core as core

func main() -> Int:
    let v: core.Vec2 = core.Vec2(x: 7)
    return core.Vec2.draw(v)
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, ok := checked.FuncSigs["engine.core.Vec2.draw"]; !ok {
		t.Fatalf("missing imported extension method signature: %#v", checked.FuncSigs)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestProtocolConformanceViaImportedExtensionGenericRequirement(t *testing.T) {
	files := map[string]string{
		"engine/core.tetra": `module engine.core
struct Vec2:
    x: Int

protocol Mapper:
    func map<T>(self: Vec2, value: T) -> T
`,
		"app/ext.tetra": `module app.ext
import engine.core as core

extension core.Vec2:
    func map<T>(self: core.Vec2, value: T) -> T:
        return value

impl core.Vec2: core.Mapper
`,
		"app/main.tetra": `module app.main
import app.ext as ext
import engine.core as core

func main() -> Int:
    let v: core.Vec2 = core.Vec2(x: 7)
    return v.x
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))
	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
}

func TestProtocolConformanceRejectsDuplicateImplClause(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Renderable
impl Vec2: Renderable

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected duplicate impl clause error")
	}
	if !strings.Contains(err.Error(), "duplicate impl conformance") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceReportsReturnTypeMismatch(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Bool:
        return true

impl Vec2: Renderable

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected wrong signature conformance error")
	}
	if !strings.Contains(err.Error(), "return type differs") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceRejectsDuplicateRequirement(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int
    func draw(self: Vec2) -> Int

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate protocol requirement 'draw'") {
			t.Fatalf("Parse error = %v", err)
		}
		return
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected duplicate requirement error")
	}
	if !strings.Contains(err.Error(), "duplicate protocol requirement 'draw'") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceReportsParameterCountMismatch(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Scalable:
    func scale(self: Vec2, factor: Int) -> Int

extension Vec2:
    func scale(self: Vec2) -> Int:
        return self.x

impl Vec2: Scalable

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected parameter count conformance error")
	}
	if !strings.Contains(err.Error(), "parameter count differs") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceReportsParameterTypeMismatch(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Scalable:
    func scale(self: Vec2, factor: Int) -> Int

extension Vec2:
    func scale(self: Vec2, factor: Bool) -> Int:
        return self.x

impl Vec2: Scalable

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected parameter type conformance error")
	}
	if !strings.Contains(err.Error(), "parameter 2 type differs") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceRejectsThrowingMethodForNonThrowingRequirement(t *testing.T) {
	src := []byte(`
enum DrawError:
    case failed

struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int throws DrawError:
        if self.x == 0:
            throw DrawError.failed
        return self.x

impl Vec2: Renderable

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected throws conformance error")
	}
	if !strings.Contains(err.Error(), "throws type differs") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceRejectsThrowTypeMismatch(t *testing.T) {
	src := []byte(`
enum DrawError:
    case failed

enum OtherError:
    case failed

struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int throws DrawError

extension Vec2:
    func draw(self: Vec2) -> Int throws OtherError:
        if self.x == 0:
            throw OtherError.failed
        return self.x

impl Vec2: Renderable

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected throws type conformance error")
	}
	if !strings.Contains(err.Error(), "throws type differs") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceRejectsMissingRequiredEffect(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int uses io

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Renderable

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected effects conformance error")
	}
	if !strings.Contains(err.Error(), "missing required effects io") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceSupportsGenericRequirementAlphaEquivalence(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Mapper:
    func map<T>(self: Vec2, value: T) -> T

extension Vec2:
    func map<U>(self: Vec2, value: U) -> U:
        return value

impl Vec2: Mapper

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := compiler.Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestProtocolConformanceRejectsInvalidSelfParameterName(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(this: Vec2) -> Int

extension Vec2:
    func draw(this: Vec2) -> Int:
        return this.x

impl Vec2: Renderable

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected self parameter conformance error")
	}
	if !strings.Contains(err.Error(), "first parameter must be 'self'") {
		t.Fatalf("error = %v", err)
	}
}

func TestProtocolConformanceRejectsSelfParameterTypeMismatch(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

struct Point:
    x: Int

protocol Renderable:
    func draw(self: Point) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Renderable

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected self parameter type conformance error")
	}
	if !strings.Contains(err.Error(), "self parameter type must be 'Vec2'") {
		t.Fatalf("error = %v", err)
	}
}

// ---- protocols_test.go ----

func TestProtocolParseCheckAndDocs(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Protocols) != 1 {
		t.Fatalf("protocols = %d", len(prog.Protocols))
	}
	if got := prog.Protocols[0].Requirements[0].Name; got != "draw" {
		t.Fatalf("requirement name = %q", got)
	}
	if _, err := compiler.Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
	docs, err := compiler.GenerateAPIDocsFromSource(src, "protocols.tetra")
	if err != nil {
		t.Fatalf("GenerateAPIDocsFromSource: %v", err)
	}
	if !strings.Contains(string(docs), "`protocol Renderable`") ||
		!strings.Contains(string(docs), "`func draw(self: Vec2) -> i32`") {
		t.Fatalf("docs = %s", string(docs))
	}
}

func TestProtocolNoLongerPlannedDiagnostic(t *testing.T) {
	_, err := compiler.Parse([]byte("protocol P:\n"))
	if err == nil {
		t.Fatalf("expected block error, not silent success")
	}
	if strings.Contains(err.Error(), "planned feature 'protocol'") {
		t.Fatalf("protocol still reports planned diagnostic: %v", err)
	}
}

// ---- small_int_range_test.go ----

func TestSmallIntLiteralRangeBoundaries(t *testing.T) {
	testkit.RequireCheckOK(t, `
func take_byte(value: UInt8) -> Int:
    return value

func take_word(value: UInt16) -> Int:
    return value

func byte_value() -> UInt8:
    return 255

func word_value() -> UInt16:
    return 65535

func main() -> Int:
    let b: UInt8 = 0
    let max_b: UInt8 = 255
    let expr_b: UInt8 = 128 + 127
    let w: UInt16 = 65530 + 5
    return take_byte(b) + take_byte(max_b) + take_word(w) + byte_value() + word_value()
`)
}

func TestSmallIntLiteralRangeRejectsOutOfRangeContextualValues(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "u8 local above max",
			src: `
func main() -> Int:
    let b: UInt8 = 256
    return b
`,
			want: "type mismatch: expected 'u8', got 'i32'",
		},
		{
			name: "u8 local below zero",
			src: `
func main() -> Int:
    let b: UInt8 = -1
    return b
`,
			want: "type mismatch: expected 'u8', got 'i32'",
		},
		{
			name: "u16 local above max",
			src: `
func main() -> Int:
    let w: UInt16 = 70000
    return w
`,
			want: "type mismatch: expected 'u16', got 'i32'",
		},
		{
			name: "u8 binary expression below zero",
			src: `
func main() -> Int:
    let b: UInt8 = 0 - 1
    return b
`,
			want: "type mismatch: expected 'u8', got 'i32'",
		},
		{
			name: "u8 binary expression above max",
			src: `
func main() -> Int:
    let b: UInt8 = 250 + 6
    return b
`,
			want: "type mismatch: expected 'u8', got 'i32'",
		},
		{
			name: "u16 binary expression above max",
			src: `
func main() -> Int:
    let w: UInt16 = 60000 + 6000
    return w
`,
			want: "type mismatch: expected 'u16', got 'i32'",
		},
		{
			name: "u16 overflow expression",
			src: `
func main() -> Int:
    let w: UInt16 = 65536 * 65536
    return w
`,
			want: "type mismatch: expected 'u16', got 'i32'",
		},
		{
			name: "function argument",
			src: `
func take_byte(value: UInt8) -> Int:
    return value

func main() -> Int:
    return take_byte(256)
`,
			want: "type mismatch for 'take_byte' arg 1",
		},
		{
			name: "function return",
			src: `
func byte_value() -> UInt8:
    return 300

func main() -> Int:
    return 0
`,
			want: "return type mismatch: expected 'u8', got 'i32'",
		},
		{
			name: "throw value",
			src: `
func fail() -> Int throws UInt8:
    throw 300

func main() -> Int:
    return 0
`,
			want: "throw type mismatch: expected 'u8', got 'i32'",
		},
		{
			name: "struct field",
			src: `
struct Header:
    byte: UInt8

func main() -> Int:
    let h: Header = Header(byte: 300)
    return h.byte
`,
			want: "type mismatch for field 'byte'",
		},
		{
			name: "enum payload",
			src: `
enum Packet:
    case byte(UInt8)

func main() -> Int:
    let p: Packet = Packet.byte(300)
    return 0
`,
			want: "enum case 'Packet.byte' payload 1 expects 'u8', got 'i32'",
		},
		{
			name: "local assignment",
			src: `
func main() -> Int:
    var b: UInt8 = 0
    b = 300
    return b
`,
			want: "type mismatch: expected 'u8', got 'i32'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.want)
		})
	}
}

// ---- type_inference_test.go ----

func TestLocalTypeInference(t *testing.T) {
	src := []byte(`
fun main(): i32 {
  let x = 40
  let y: i32 = 2
  return x + y
}
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := compiler.Check(prog); err != nil {
		t.Fatalf("check: %v", err)
	}
}

func TestFlowLetIsImmutable(t *testing.T) {
	src := []byte(`
func main() -> i32:
  let x = 1
  x = 2
  return x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := compiler.Check(prog); err == nil {
		t.Fatalf("expected immutable Flow let assignment to fail")
	}
}

func TestV1CanonicalTypeNamesAndStructuralSlots(t *testing.T) {
	src := []byte(`
struct Packet:
    id: Int
    payload: String
    owned: island

func main() -> Int:
    let ok: Bool = true
    let byte: Byte = 7
    let text: String = "ok"
    return byte
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	main := checked.Funcs[0]
	if got := main.Locals["ok"].TypeName; got != "bool" {
		t.Fatalf("Bool alias resolved to %q, want bool", got)
	}
	if got := main.Locals["byte"].TypeName; got != "u8" {
		t.Fatalf("Byte alias resolved to %q, want u8", got)
	}
	if got := main.Locals["text"].TypeName; got != "str" {
		t.Fatalf("String alias resolved to %q, want str", got)
	}
	packet := checked.Types["Packet"]
	if packet == nil {
		t.Fatalf("missing Packet type")
	}
	if got := packet.SlotCount; got != 4 {
		t.Fatalf("Packet slots = %d, want 4", got)
	}
	if got := packet.FieldMap["payload"].TypeName; got != "str" {
		t.Fatalf("payload type = %q, want str", got)
	}
}

func TestV1StructConstructorsRejectInvalidFields(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "missing field",
			src: `
struct Pair:
    x: Int
    y: Int

func main() -> Int:
    let p: Pair = Pair(x: 1)
    return 0
`,
			want: "missing field 'y'",
		},
		{
			name: "unknown field",
			src: `
struct Pair:
    x: Int

func main() -> Int:
    let p: Pair = Pair(y: 1)
    return 0
`,
			want: "unknown field 'y'",
		},
		{
			name: "duplicate field",
			src: `
struct Pair:
    x: Int
    y: Int

func main() -> Int:
    let p: Pair = Pair(x: 1, x: 2)
    return 0
`,
			want: "duplicate field 'x'",
		},
		{
			name: "type mismatch",
			src: `
struct Pair:
    x: Int

func main() -> Int:
    let p: Pair = Pair(x: true)
    return 0
`,
			want: "type mismatch for field 'x'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testkit.CheckProgram(tt.src); err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.want)
			} else if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestV1InferenceRequiresAnnotationForNoneAndUsesExpectedOptionals(t *testing.T) {
	err := testkit.CheckProgram(`
func main() -> Int:
    let value = none
    return 0
`)
	if err == nil {
		t.Fatalf("expected none inference error")
	}
	if !strings.Contains(err.Error(), "cannot infer type from 'none'") {
		t.Fatalf("error = %v", err)
	}

	if err := testkit.CheckProgram(`
func consume(value: Int?) -> Int:
    if value == none:
        return 0
    return 1

func main() -> Int:
    let value: Int? = none
    return consume(value)
`); err != nil {
		t.Fatalf("expected annotated optional none to check: %v", err)
	}
}

func TestV1APIDocsUseCanonicalBuiltinTypeNames(t *testing.T) {
	src := []byte(`
const answer: Int = 42

func audit(token: ConsentToken, secret: SecretInt, text: String, byte: Byte) -> Bool:
    return true
`)
	docs, err := compiler.GenerateAPIDocsFromSource(src, "types.tetra")
	if err != nil {
		t.Fatalf("GenerateAPIDocsFromSource: %v", err)
	}
	out := string(docs)
	for _, want := range []string{
		"`const answer: i32`",
		"`func audit(token: consent.token, secret: secret.i32, text: str, byte: u8) -> bool`",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("docs missing %q:\n%s", want, out)
		}
	}
}

func TestV1OpaqueHandleTypesAreNotInterchangeable(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "island to ptr",
			src: `
func main() -> Int
uses alloc, capability, islands, mem:
    island(64) as isl:
        let p: ptr = isl
    return 0
`,
			want: "type mismatch",
		},
		{
			name: "ptr to island",
			src: `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let isl: island = p
        return 0
    return 0
`,
			want: "type mismatch",
		},
		{
			name: "capability families",
			src: `
func main() -> Int
uses capability, io:
    unsafe:
        let io: cap.io = core.cap_io()
        let mem: cap.mem = io
        return 0
    return 0
`,
			want: "type mismatch",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testkit.CheckProgram(tt.src); err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.want)
			} else if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

// ---- typed_errors_test.go ----

func TestTypedErrorsParseCheckAndLower(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 42
    else:
        throw ReadError.eof

func caller() -> Int throws ReadError:
    let value: Int = try read(true)
    return value

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !prog.Funcs[0].HasThrows || prog.Funcs[0].Throws.Name != "ReadError" {
		t.Fatalf("throws = %#v", prog.Funcs[0].Throws)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["read"].ThrowsType; got != "ReadError" {
		t.Fatalf("read throws = %q", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsRejectBareThrowingCall(t *testing.T) {
	src := []byte(`
enum E:
    case bad

func f() -> Int throws E:
    throw E.bad

func main() -> Int:
    return f()
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected bare throwing call error")
	}
	if !strings.Contains(err.Error(), "requires try") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsRejectTryOutsideThrowingFunction(t *testing.T) {
	src := []byte(`
enum E:
    case bad

func f() -> Int throws E:
    throw E.bad

func main() -> Int:
    return try f()
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected try context error")
	}
	if !strings.Contains(err.Error(), "try is only allowed in throwing functions") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsAllowMultiSlotErrorPayload(t *testing.T) {
	src := []byte(`
func fail(flag: Bool) -> Int throws String:
    if flag:
        return 7
    else:
        throw "bad"

func caller(flag: Bool) -> Int throws String:
    return try fail(flag)

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["fail"].ReturnSlots; got != 4 {
		t.Fatalf("fail return slots = %d, want 4", got)
	}
	if got := checked.FuncSigs["fail"].ThrowsType; got != "str" {
		t.Fatalf("fail throws type = %q, want str", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsAllowEnumPayloadError(t *testing.T) {
	src := []byte(`
enum ParseError:
    case unexpected(Int)
    case eof

func fail(flag: Bool) -> Int throws ParseError:
    if flag:
        return 7
    else:
        throw ParseError.unexpected(9)

func caller(flag: Bool) -> Int throws ParseError:
    return try fail(flag)

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["fail"].ThrowsType; got != "ParseError" {
		t.Fatalf("fail throws = %q, want ParseError", got)
	}
	if got := checked.FuncSigs["fail"].ReturnSlots; got != 4 {
		t.Fatalf("fail return slots = %d, want 4", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsTryPropagatesIntoOptionalThrows(t *testing.T) {
	src := []byte(`
func fail(flag: Bool) -> Int throws Int:
    if flag:
        return 7
    else:
        throw 11

func caller(flag: Bool) -> Int throws Int?:
    return try fail(flag)

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["caller"].ThrowsType; got != "i32?" {
		t.Fatalf("caller throws type = %q, want i32?", got)
	}
	if got := checked.FuncSigs["caller"].ReturnSlots; got != 4 {
		t.Fatalf("caller return slots = %d, want 4", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsTryPropagatesMultiSlotIntoOptionalThrows(t *testing.T) {
	src := []byte(`
func fail(flag: Bool) -> Int throws String:
    if flag:
        return 7
    else:
        throw "bad"

func caller(flag: Bool) -> Int throws String?:
    return try fail(flag)

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["caller"].ThrowsType; got != "str?" {
		t.Fatalf("caller throws type = %q, want str?", got)
	}
	if got := checked.FuncSigs["caller"].ReturnSlots; got != 5 {
		t.Fatalf("caller return slots = %d, want 5", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsGenericEnumThrowMonomorphizes(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof

func fail<T>(err: T) -> Int throws T:
    throw err

func caller() -> Int throws ReadError:
    let err: ReadError = ReadError.eof
    return try fail(err)

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["fail__T_ReadError"].ThrowsType; got != "ReadError" {
		t.Fatalf("monomorphized fail throws = %q, want ReadError", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsRejectWrongThrowType(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 1
    throw 7

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected throw type mismatch")
	}
	if !strings.Contains(err.Error(), "throw type mismatch: expected 'ReadError', got 'i32'") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsImportedThrowingFunctionCheckAndLower(t *testing.T) {
	files := map[string]string{
		"engine/errors.tetra": `module engine.errors
enum ReadError:
    case eof

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 42
    throw ReadError.eof
`,
		"app/main.tetra": `module app.main
import engine.errors as errors

func caller(flag: Bool) -> Int throws errors.ReadError:
    return try errors.read(flag)

func main() -> Int:
    return 0
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if got := checked.FuncSigs["engine.errors.read"].ThrowsType; got != "engine.errors.ReadError" {
		t.Fatalf("imported read throws type = %q, want engine.errors.ReadError", got)
	}
	if got := checked.FuncSigs["app.main.caller"].ThrowsType; got != "engine.errors.ReadError" {
		t.Fatalf("caller throws type = %q, want engine.errors.ReadError", got)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestTypedErrorsCatchExpressionEnumPayloadSmoke(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof
    case denied(Int)

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 42
    throw ReadError.denied(7)

func main() -> Int:
    let value: Int = catch read(false):
    case ReadError.eof:
        0
    case ReadError.denied(code):
        code
    return value
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsCatchPayloadCaseRequiresDestructuringDiagnostic(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.denied(7)

func main() -> Int:
    return catch read():
    case ReadError.eof:
        0
    case ReadError.denied:
        1
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected catch payload destructuring diagnostic")
	}
	if !strings.Contains(
		err.Error(),
		"carries 1 payload value(s); use 'ReadError.denied(value1)'",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchNoPayloadCaseRejectsPayloadSyntaxDiagnostic(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.eof

func main() -> Int:
    return catch read():
    case ReadError.eof(code):
        code
    case ReadError.denied(code):
        code
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected catch no-payload pattern diagnostic")
	}
	if !strings.Contains(err.Error(), "has no payload; use 'ReadError.eof'") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchRejectsNonThrowingCall(t *testing.T) {
	src := []byte(`
func read() -> Int:
    return 42

func main() -> Int:
    return catch read():
    case _:
        0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected catch non-throwing call error")
	}
	if !strings.Contains(err.Error(), "catch expects a throwing function call") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchBindingScopeDiagnostic(t *testing.T) {
	src := []byte(`
enum ReadError:
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.denied(7)

func main() -> Int:
    let value: Int = catch read():
    case ReadError.denied(code):
        code
    return code
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected catch binding scope error")
	}
	if !strings.Contains(err.Error(), "identifier 'code' is out of scope") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchRequiresExhaustiveCases(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.eof

func main() -> Int:
    return catch read():
    case ReadError.eof:
        0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected catch exhaustiveness error")
	}
	if !strings.Contains(err.Error(), "catch expression must be exhaustive") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchRejectsHandlerTypeMismatch(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof

func read() -> Int throws ReadError:
    throw ReadError.eof

func main() -> Int:
    return catch read():
    case ReadError.eof:
        "bad"
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected catch handler type mismatch")
	}
	if !strings.Contains(err.Error(), "catch expression case type mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchGuardEnumPayloadSmoke(t *testing.T) {
	src := []byte(`
enum ReadError:
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.denied(7)

func main() -> Int:
    return catch read():
    case ReadError.denied(code) if code > 0:
        code
    case ReadError.denied(other):
        other
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsCatchGuardedEnumPayloadCaseIsNotExhaustive(t *testing.T) {
	src := []byte(`
enum ReadError:
    case denied(Int)
    case eof

func read() -> Int throws ReadError:
    throw ReadError.denied(7)

func main() -> Int:
    return catch read():
    case ReadError.denied(code) if code > 0:
        code
    case ReadError.eof:
        0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected guarded catch exhaustiveness error")
	}
	if !strings.Contains(err.Error(), "catch expression must be exhaustive") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchDuplicateUnguardedEnumPayloadCaseDiagnostic(t *testing.T) {
	src := []byte(`
enum ReadError:
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.denied(7)

func main() -> Int:
    return catch read():
    case ReadError.denied(code):
        code
    case ReadError.denied(other):
        other
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected duplicate catch enum payload case diagnostic")
	}
	if !strings.Contains(err.Error(), "duplicate catch pattern") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchDefaultMustBeLastDiagnostic(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.eof

func main() -> Int:
    return catch read():
    case _:
        0
    case ReadError.eof:
        1
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected catch default ordering diagnostic")
	}
	if !strings.Contains(err.Error(), "catch default must be last") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchRejectsWrongEnumCaseDiagnostic(t *testing.T) {
	src := []byte(`
enum ReadError:
    case denied(Int)
enum WriteError:
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.denied(7)

func main() -> Int:
    return catch read():
    case WriteError.denied(code):
        code
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected wrong catch enum case diagnostic")
	}
	if !strings.Contains(err.Error(), "enum pattern type mismatch") &&
		!strings.Contains(err.Error(), "catch pattern type mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsCatchRuntimeErrorAndSuccessPaths(t *testing.T) {
	src := `
enum ReadError:
    case eof

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 35
    throw ReadError.eof

func recover(flag: Bool) -> Int:
    return catch read(flag):
    case ReadError.eof:
        7

func main() -> Int:
    return recover(false) + recover(true)
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want 42", exitCode)
	}
}
