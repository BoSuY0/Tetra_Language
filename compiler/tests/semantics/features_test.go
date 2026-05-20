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
		"language.protocol-conformance-mvp":       compiler.FeatureStatusCurrent,
		"language.callable-mvp":                   compiler.FeatureStatusCurrent,
		"language.callable-level1":                compiler.FeatureStatusCurrent,
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
		"actors.distributed-runtime":              compiler.FeatureStatusCurrent,
		"eco.distributed-network":                 compiler.FeatureStatusPostV1,
		"ui.native-runtime":                       compiler.FeatureStatusCurrent,
		"language.full-first-class-callables":     compiler.FeatureStatusCurrent,
	} {
		if gotStatus := seenID[id]; gotStatus != wantStatus {
			t.Fatalf("feature %s status = %q, want %q", id, gotStatus, wantStatus)
		}
	}
	genericsMVP := seenFeature["language.generics-mvp"]
	for _, want := range []string{"statically monomorphized", "no runtime generic values or dynamic dispatch", "generic structs", "future/post-v1"} {
		if !strings.Contains(genericsMVP.Scope+" "+genericsMVP.Stability, want) {
			t.Fatalf("generics MVP feature missing %q boundary: %#v", want, genericsMVP)
		}
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
	for _, want := range []string{"stable uses effect names and groups", "transitive call propagation", "missing uses declarations are diagnostics", "no effect inference"} {
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
	for _, want := range []string{"production local safety model", "ownership/lifetime/borrow/consume/inout", "resource finalization", "callable escape diagnostics", "effects/capabilities/privacy/consent/budget", "unsafe boundaries", "actor/task transfer safety", "pointer/MMIO/memory capability gates", "explicit diagnostics"} {
		if !strings.Contains(safetyCore.Scope+" "+safetyCore.Stability, want) {
			t.Fatalf("safety production core missing %q boundary: %#v", want, safetyCore)
		}
	}
	uiMetadata := seenFeature["ui.metadata-v1"]
	if uiMetadata.Status != compiler.FeatureStatusCurrent || uiMetadata.Since != "v0.4.0" {
		t.Fatalf("ui.metadata-v1 lifecycle = status %q since %q, want current since v0.4.0", uiMetadata.Status, uiMetadata.Since)
	}
	for _, want := range []string{"production UI metadata contract", "deterministic tetra.ui.v1 JSON", "web command-dispatch preview", "wasm32-web command dispatch", "native shell command dispatch", "widget-tree traces", "JSON trace sidecars", "style metadata preview attributes", "accessibility metadata preview attributes"} {
		if !strings.Contains(uiMetadata.Scope+" "+uiMetadata.Stability, want) {
			t.Fatalf("UI metadata feature missing %q boundary: %#v", want, uiMetadata)
		}
	}
	distributedActors := seenFeature["actors.distributed-runtime"]
	if distributedActors.Status != compiler.FeatureStatusCurrent || distributedActors.Since != "v0.4.0" {
		t.Fatalf("distributed actors lifecycle = status %q since %q, want current since v0.4.0", distributedActors.Status, distributedActors.Since)
	}
	for _, want := range []string{"production Linux-x64 distributed actor runtime path", "actornet loopback TCP broker", "distributed node identity", "remote actor handles", "network mailbox send/receive", "i32, tagged, and typed frames", "missing-node failure/status propagation", "task cancel/join handles", "tetra.actors.distributed-runtime.v1 smoke evidence", "transport-only or fake reports", "non-Linux-x64 targets", "broader structured-concurrency guarantees"} {
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
		"mem": true, "mmio": true, "privacy": true, "runtime": true,
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
