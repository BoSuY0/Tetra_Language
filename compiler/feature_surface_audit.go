package compiler

import (
	"fmt"
	"strings"
)

const (
	featureSurfaceAuditSchemaV1  = "tetra.language.feature_surface_audit.v1"
	featureSurfaceAuditScopeP220 = "p22.0_full_feature_surface_audit"
)

type FeatureSurfaceAuditCategory string

const (
	FeatureSurfaceFirstClassCallables          FeatureSurfaceAuditCategory = "first_class_callables"
	FeatureSurfaceClosures                     FeatureSurfaceAuditCategory = "closures"
	FeatureSurfaceProtocolsTraitObjects        FeatureSurfaceAuditCategory = "protocols_trait_objects"
	FeatureSurfaceRuntimeGenerics              FeatureSurfaceAuditCategory = "runtime_generics"
	FeatureSurfaceAdvancedEnumsPatternMatching FeatureSurfaceAuditCategory = "advanced_enums_pattern_matching"
	FeatureSurfaceAsyncTypedErrors             FeatureSurfaceAuditCategory = "async_typed_errors"
	FeatureSurfaceStructuredConcurrency        FeatureSurfaceAuditCategory = "structured_concurrency"
	FeatureSurfaceModulesPackages              FeatureSurfaceAuditCategory = "modules_packages"
	FeatureSurfaceMacrosMetaprogramming        FeatureSurfaceAuditCategory = "macros_metaprogramming"
	FeatureSurfaceUISurface                    FeatureSurfaceAuditCategory = "ui_surface"
	FeatureSurfaceEcoCapsules                  FeatureSurfaceAuditCategory = "eco_capsules"
)

type FeatureSurfaceAuditReport struct {
	SchemaVersion                     string                   `json:"schema_version"`
	Scope                             string                   `json:"scope"`
	Rows                              []FeatureSurfaceAuditRow `json:"rows"`
	NonClaims                         []string                 `json:"non_claims"`
	PromotedWithoutSameBranchEvidence bool                     `json:"promoted_without_same_branch_evidence"`
	FullV1GuaranteesClaimed           bool                     `json:"full_v1_guarantees_claimed"`
	RuntimeGenericValuesClaimed       bool                     `json:"runtime_generic_values_claimed"`
	TraitObjectsClaimed               bool                     `json:"trait_objects_claimed"`
	MacroSystemClaimed                bool                     `json:"macro_system_claimed"`
	StructuredConcurrencyClaimed      bool                     `json:"structured_concurrency_claimed"`
	CrossPlatformUIRuntimeClaimed     bool                     `json:"cross_platform_ui_runtime_claimed"`
	DistributedEcoClaimed             bool                     `json:"distributed_eco_claimed"`
	ProofCarryingCapsulesClaimed      bool                     `json:"proof_carrying_capsules_claimed"`
	PerformanceClaimed                bool                     `json:"performance_claimed"`
	SafeSemanticsChanged              bool                     `json:"safe_semantics_changed"`
}

type FeatureSurfaceAuditRow struct {
	Category                  FeatureSurfaceAuditCategory `json:"category"`
	Name                      string                      `json:"name"`
	Decision                  string                      `json:"decision"`
	FeatureIDs                []string                    `json:"feature_ids"`
	RegistryStatuses          map[string]FeatureStatus    `json:"registry_statuses"`
	Evidence                  []string                    `json:"evidence"`
	Boundaries                []string                    `json:"boundaries"`
	RequiredPromotionEvidence []string                    `json:"required_promotion_evidence"`
	SameBranchEvidence        bool                        `json:"same_branch_evidence"`
	PromotedInThisAudit       bool                        `json:"promoted_in_this_audit"`
}

func BuildP22FeatureSurfaceAudit() FeatureSurfaceAuditReport {
	registry := featureSurfaceRegistryByID()
	return FeatureSurfaceAuditReport{
		SchemaVersion: featureSurfaceAuditSchemaV1,
		Scope:         featureSurfaceAuditScopeP220,
		Rows: []FeatureSurfaceAuditRow{
			p22FeatureSurfaceRow(registry, FeatureSurfaceFirstClassCallables, "First-class callables", "keep_current_bounded_and_route_full_expansion_to_P22.1",
				[]string{"language.callable-mvp", "language.callable-level1", "language.callable-level2", "language.full-first-class-callables"},
				[]string{
					"FeatureRegistry records Level 0/1/2 callable support plus language.full-first-class-callables as current within the v0.4.0 bounded safe by-value model.",
					"language.full-first-class-callables evidence includes the bounded fnptr fast path and the fixed 4-slot callable handle for larger immutable Int/Bool/String/simple-aggregate captures.",
					"docs/spec/v1_feature_status.md keeps mutable by-reference capture, pointer/resource capture, thread-boundary callable escape, and dynamic/generic callable polymorphism outside the current promotion.",
				},
				[]string{
					"mutable by-reference capture remains diagnostic or future work",
					"pointer/resource capture and thread-boundary callable escape remain unpromoted",
					"P22.1 owns any future first-class callable expansion beyond the current bounded model",
				},
				[]string{
					"P22.1 lifetime/ABI evidence in the same branch",
					"stable diagnostics for unsupported callable movement",
					"registry, docs, manifest, and tests updated in the same branch before promotion",
				}),
			p22FeatureSurfaceRow(registry, FeatureSurfaceClosures, "Closures", "keep_safe_by_value_capture_slice_only",
				[]string{"language.callable-level2", "language.full-first-class-callables"},
				[]string{
					"FeatureRegistry records captured closure Level 2 plus full first-class callables as current only for safe by-value captures and fixed-handle movement.",
					"Current evidence covers local storage, aliases, returns, struct fields, enum payloads, synchronous callback arguments, and generated interface metadata.",
					"same-branch evidence is required before promoting pointer/resource capture, mutable by-reference capture, generic closure capture, or thread movement.",
				},
				[]string{
					"pointer/resource capture stays outside the promoted closure surface",
					"generic closure and generic callback-closure capture remain rejected",
					"mutable capture escape and thread-boundary movement stay gated by diagnostics",
				},
				[]string{
					"same-branch evidence for ownership, synchronization, lifetime, ABI, docs, and diagnostics",
					"new closure tests proving each movement path",
				}),
			p22FeatureSurfaceRow(registry, FeatureSurfaceProtocolsTraitObjects, "Protocols and trait objects", "keep_static_conformance_only_and_route_runtime_existentials_to_P22.2",
				[]string{"language.protocol-conformance-mvp", "language.protocol-bound-generics-static"},
				[]string{
					"FeatureRegistry records static conformance and static protocol-bound generic validation during monomorphization.",
					"Current scope explicitly says no witness tables, trait objects, runtime protocol values, or dynamic dispatch model.",
					"P22.2 owns any decision to design runtime existential values while keeping the static fast path.",
				},
				[]string{
					"no witness tables are promoted",
					"trait objects and runtime protocol values remain post-v1 unless P22.2 gates them",
					"dynamic dispatch and conformance-table lookup remain unsupported",
				},
				[]string{
					"P22.2 design and implementation evidence in the same branch",
					"ABI/report-visible dynamic dispatch evidence if runtime existentials are promoted",
				}),
			p22FeatureSurfaceRow(registry, FeatureSurfaceRuntimeGenerics, "Runtime generics", "keep_static_monomorphized_generic_functions_only",
				[]string{"language.generics-mvp", "language.protocol-bound-generics-static"},
				[]string{
					"FeatureRegistry records statically monomorphized generic functions with inferred value arguments and static protocol-bound validation.",
					"docs/spec/v1_scope.md keeps runtime generic values, explicit type arguments, generic structs, higher-ranked generics, and full protocol-bound generic dispatch post-v1 unless promoted.",
				},
				[]string{
					"runtime generic values are not current",
					"explicit type arguments, generic structs, and higher-ranked generics remain outside current support",
					"full protocol-bound generic dispatch and broad specialization guarantees are not promoted here",
				},
				[]string{
					"parser, semantics, ABI, optimizer, docs, manifest, and validator evidence in the same branch",
				}),
			p22FeatureSurfaceRow(registry, FeatureSurfaceAdvancedEnumsPatternMatching, "Advanced enums and pattern matching", "keep_positional_enum_payload_slice_only",
				[]string{"language.enum-payload-match"},
				[]string{
					"FeatureRegistry records positional enum payload constructors and payload bindings for match/catch/if-let.",
					"Current scope includes exhaustive unguarded enum match/catch coverage and stable diagnostics for payload arity/type/syntax errors.",
				},
				[]string{
					"advanced ADT constructors remain future/post-v1",
					"nested destructuring patterns remain future/post-v1",
					"guard expansion and richer payload algebra remain future/post-v1",
				},
				[]string{
					"same-branch parser, semantics, lowering, diagnostics, docs, and manifest evidence for each promoted pattern form",
				}),
			p22FeatureSurfaceRow(registry, FeatureSurfaceAsyncTypedErrors, "Async typed errors", "keep_try_await_boundary_only",
				[]string{"language.task-handles-mvp", "language.resource-lifetime-mvp"},
				[]string{
					"docs/spec/v1_scope.md defines async typed-error support as the checked try await <call>() synchronous-lowering boundary.",
					"FeatureRegistry records typed task handles and resource lifetime checks for task handles, task groups, islands, and typed-error resource aliases.",
					"await try <call>() remains rejected by stable diagnostics rather than promoted by this audit.",
				},
				[]string{
					"async typed-error behavior beyond try await stays post-v1",
					"cancellation and structured concurrency are not promoted by the async typed-error row",
					"await try remains a rejected boundary form",
				},
				[]string{
					"same-branch async parser/checker/lowering/runtime/docs evidence for any extension beyond try await",
				}),
			p22FeatureSurfaceRow(registry, FeatureSurfaceStructuredConcurrency, "Structured concurrency", "keep_local_task_actor_bounded_and_full_structured_concurrency_future",
				[]string{"actors.task-transfer-safety", "language.task-handles-mvp", "actors.distributed-runtime"},
				[]string{
					"FeatureRegistry records actor/task transfer safety as a conservative local MVP and typed task handle wrappers for slot counts 2..8.",
					"actors.distributed-runtime is current only for the Linux-x64 distributed actor runtime path and explicitly excludes broader structured-concurrency guarantees.",
					"Existing scheduler/reactor reports mention cancellation checkpoints and task groups as evidence rows, not as a full structured concurrency claim.",
				},
				[]string{
					"full cancellation remains outside the current support claim",
					"full race-safety proof remains outside the current support claim",
					"broader structured-concurrency guarantees remain outside the current actor/task MVP",
				},
				[]string{
					"same-branch scheduler, cancellation, task-group, actor, race, docs, and manifest evidence before any full structured concurrency promotion",
				}),
			p22FeatureSurfaceRow(registry, FeatureSurfaceModulesPackages, "Modules and packages", "keep_local_module_package_capsule_surface_only",
				[]string{"language.globals-properties-capsule-mvp", "eco.local-package-lifecycle"},
				[]string{
					"FeatureRegistry records compile-time capsule metadata plus local Eco package lifecycle support.",
					"Current local package lifecycle covers verify, lock generation/validation, pack/unpack, vault, stable/beta metadata, target-aware download, fixtures, local mirror reports, and single-origin HTTP(S) fetch into a verified local store.",
				},
				[]string{
					"capsule metadata is compile-time metadata, not a runtime proof-carrying capsule system",
					"distributed EcoNet and production TetraHub publishing remain post-v1",
				},
				[]string{
					"same-branch package, module, trust, capsule, docs, manifest, and security evidence for any distributed promotion",
				}),
			p22FeatureSurfaceRow(registry, FeatureSurfaceMacrosMetaprogramming, "Macros and metaprogramming", "keep_absent_post_v1",
				nil,
				[]string{
					"FeatureRegistry has no current macro/metaprogramming feature ID.",
					"no current macro/metaprogramming feature is promoted by P22.0; absence is the same-branch evidence for keeping this category post-v1.",
				},
				[]string{
					"macro and metaprogramming systems remain post-v1 until a concrete design, implementation, tests, docs, and registry entry exist",
				},
				[]string{
					"same-branch evidence must include a new registry ID, parser/semantics/tooling tests, docs, manifest updates, and non-claim review",
				}),
			p22FeatureSurfaceRow(registry, FeatureSurfaceUISurface, "UI and Surface", "keep_linux_web_surface_bounded_and_platform_gate_experimental",
				[]string{"ui.metadata-v1", "ui.surface-core", "ui.surface-linux-x64", "ui.surface-web-wasm", "ui.native-runtime", "ui.platform-runtime", "ui.surface-macos-x64", "ui.surface-windows-x64", "ui.surface-wasm32-wasi"},
				[]string{
					"FeatureRegistry records current UI metadata, bounded Surface core, Linux-x64 Surface host, wasm32-web Surface, and Linux-x64 native UI runtime evidence.",
					"ui.platform-runtime remains experimental and requires Linux, Windows, macOS, and Web runtime-backed reports before production promotion.",
					"macOS, Windows, and wasm32-wasi Surface hosts are unsupported in the registry.",
				},
				[]string{
					"cross-platform production UI runtime is not claimed",
					"macOS and Windows native runtime claims require real target-host reports",
					"platform accessibility integration and broad native widget behavior remain gated",
				},
				[]string{
					"same-branch Linux, Windows, macOS, Web runtime-backed reports with artifact hashes, docs, manifest, and validators before cross-platform promotion",
				}),
			p22FeatureSurfaceRow(registry, FeatureSurfaceEcoCapsules, "Eco and capsules", "keep_local_eco_and_metadata_capsules_current_distributed_post_v1",
				[]string{"language.globals-properties-capsule-mvp", "eco.local-package-lifecycle", "eco.distributed-network"},
				[]string{
					"FeatureRegistry records local Eco lifecycle as current and eco.distributed-network as post-v1.",
					"language.globals-properties-capsule-mvp covers compile-time capsule metadata; this is not proof-carrying capsules.",
					"Current support is local Eco evidence, not distributed EcoNet, production TetraHub publishing, global trust scoring, or proof-carrying capsules.",
				},
				[]string{
					"distributed EcoNet remains post-v1",
					"proof-carrying capsules remain post-v1",
					"global trust scoring and production publishing remain post-v1",
				},
				[]string{
					"same-branch distributed network, trust, capsule proof, package publishing, docs, manifest, and security evidence before promotion",
				}),
		},
		NonClaims: p22FeatureSurfaceAuditNonClaims(),
	}
}

func ValidateP22FeatureSurfaceAudit(report FeatureSurfaceAuditReport) error {
	if report.SchemaVersion != featureSurfaceAuditSchemaV1 {
		return fmt.Errorf("feature surface audit schema = %q, want %q", report.SchemaVersion, featureSurfaceAuditSchemaV1)
	}
	if report.Scope != featureSurfaceAuditScopeP220 {
		return fmt.Errorf("feature surface audit scope = %q, want %q", report.Scope, featureSurfaceAuditScopeP220)
	}
	if report.PromotedWithoutSameBranchEvidence {
		return fmt.Errorf("feature surface audit: same-branch evidence is required before promotion")
	}
	if report.FullV1GuaranteesClaimed {
		return fmt.Errorf("feature surface audit: full v1 guarantee claim is forbidden")
	}
	if report.RuntimeGenericValuesClaimed {
		return fmt.Errorf("feature surface audit: runtime generic value claim is forbidden")
	}
	if report.TraitObjectsClaimed {
		return fmt.Errorf("feature surface audit: trait object claim is forbidden")
	}
	if report.MacroSystemClaimed {
		return fmt.Errorf("feature surface audit: macro system claim is forbidden")
	}
	if report.StructuredConcurrencyClaimed {
		return fmt.Errorf("feature surface audit: structured concurrency claim is forbidden")
	}
	if report.CrossPlatformUIRuntimeClaimed {
		return fmt.Errorf("feature surface audit: cross-platform production UI runtime claim is forbidden")
	}
	if report.DistributedEcoClaimed {
		return fmt.Errorf("feature surface audit: distributed Eco claim is forbidden")
	}
	if report.ProofCarryingCapsulesClaimed {
		return fmt.Errorf("feature surface audit: proof-carrying capsule claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("feature surface audit: performance claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("feature surface audit: safe-program semantics change is forbidden")
	}
	for _, nonClaim := range p22FeatureSurfaceAuditNonClaims() {
		if !p22FeatureSurfaceHasString(report.NonClaims, nonClaim) {
			return fmt.Errorf("feature surface audit: missing non-claim %q", nonClaim)
		}
	}
	if err := validateP22FeatureSurfaceStrings("non-claim", report.NonClaims); err != nil {
		return err
	}

	registry := featureSurfaceRegistryByID()
	expected := map[FeatureSurfaceAuditCategory]bool{}
	for _, category := range p22FeatureSurfaceAuditCategories() {
		expected[category] = true
	}
	seen := map[FeatureSurfaceAuditCategory]bool{}
	for _, row := range report.Rows {
		if row.Category == "" || strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Decision) == "" {
			return fmt.Errorf("feature surface audit: row missing required metadata: %#v", row)
		}
		if !expected[row.Category] {
			return fmt.Errorf("feature surface audit: unexpected category %s", row.Category)
		}
		if seen[row.Category] {
			return fmt.Errorf("feature surface audit: duplicate category %s", row.Category)
		}
		seen[row.Category] = true
		if row.PromotedInThisAudit && !row.SameBranchEvidence {
			return fmt.Errorf("feature surface audit: row %s promotion lacks same-branch evidence", row.Category)
		}
		if !row.SameBranchEvidence {
			return fmt.Errorf("feature surface audit: row %s missing same-branch evidence", row.Category)
		}
		if err := validateP22FeatureSurfaceStrings("row "+string(row.Category)+" evidence", row.Evidence); err != nil {
			return err
		}
		if err := validateP22FeatureSurfaceStrings("row "+string(row.Category)+" boundary", row.Boundaries); err != nil {
			return err
		}
		if err := validateP22FeatureSurfaceStrings("row "+string(row.Category)+" promotion evidence", row.RequiredPromotionEvidence); err != nil {
			return err
		}
		if row.Category != FeatureSurfaceMacrosMetaprogramming && len(row.FeatureIDs) == 0 {
			return fmt.Errorf("feature surface audit: row %s missing feature IDs", row.Category)
		}
		if row.Category == FeatureSurfaceMacrosMetaprogramming && len(row.FeatureIDs) == 0 {
			combined := p22FeatureSurfaceCombined(row)
			if !strings.Contains(combined, "no current macro/metaprogramming feature") || !strings.Contains(combined, "post-v1") {
				return fmt.Errorf("feature surface audit: macro/metaprogramming row must record no current feature and post-v1 boundary")
			}
		}
		if row.RegistryStatuses == nil {
			return fmt.Errorf("feature surface audit: row %s missing registry statuses", row.Category)
		}
		featureSeen := map[string]bool{}
		for _, id := range row.FeatureIDs {
			if strings.TrimSpace(id) == "" {
				return fmt.Errorf("feature surface audit: row %s has empty feature ID", row.Category)
			}
			feature, ok := registry[id]
			if !ok {
				return fmt.Errorf("feature surface audit: unknown feature %s in row %s", id, row.Category)
			}
			if featureSeen[id] {
				return fmt.Errorf("feature surface audit: row %s duplicates feature ID %s", row.Category, id)
			}
			featureSeen[id] = true
			if row.RegistryStatuses[id] != feature.Status {
				return fmt.Errorf("feature surface audit: registry status drift for %s in row %s: got %q want %q", id, row.Category, row.RegistryStatuses[id], feature.Status)
			}
		}
	}
	for _, category := range p22FeatureSurfaceAuditCategories() {
		if !seen[category] {
			return fmt.Errorf("feature surface audit: missing category %s", category)
		}
	}
	return nil
}

func p22FeatureSurfaceAuditCategories() []FeatureSurfaceAuditCategory {
	return []FeatureSurfaceAuditCategory{
		FeatureSurfaceFirstClassCallables,
		FeatureSurfaceClosures,
		FeatureSurfaceProtocolsTraitObjects,
		FeatureSurfaceRuntimeGenerics,
		FeatureSurfaceAdvancedEnumsPatternMatching,
		FeatureSurfaceAsyncTypedErrors,
		FeatureSurfaceStructuredConcurrency,
		FeatureSurfaceModulesPackages,
		FeatureSurfaceMacrosMetaprogramming,
		FeatureSurfaceUISurface,
		FeatureSurfaceEcoCapsules,
	}
}

func p22FeatureSurfaceAuditNonClaims() []string {
	return []string{
		"no full v1 language guarantee is claimed",
		"no runtime generic values are claimed",
		"no trait objects or runtime protocol values are claimed",
		"no macro/metaprogramming system is claimed",
		"no full structured concurrency guarantee is claimed",
		"no cross-platform production UI runtime is claimed",
		"no distributed EcoNet or proof-carrying capsule promotion is claimed",
		"no performance claim is made",
		"safe-program semantics do not change",
	}
}

func p22FeatureSurfaceRow(registry map[string]FeatureInfo, category FeatureSurfaceAuditCategory, name, decision string, featureIDs, evidence, boundaries, promotionEvidence []string) FeatureSurfaceAuditRow {
	statuses := map[string]FeatureStatus{}
	for _, id := range featureIDs {
		statuses[id] = registry[id].Status
	}
	return FeatureSurfaceAuditRow{
		Category:                  category,
		Name:                      name,
		Decision:                  decision,
		FeatureIDs:                append([]string{}, featureIDs...),
		RegistryStatuses:          statuses,
		Evidence:                  append([]string{}, evidence...),
		Boundaries:                append([]string{}, boundaries...),
		RequiredPromotionEvidence: append([]string{}, promotionEvidence...),
		SameBranchEvidence:        true,
	}
}

func featureSurfaceRegistryByID() map[string]FeatureInfo {
	registry := map[string]FeatureInfo{}
	for _, feature := range FeatureRegistry() {
		registry[feature.ID] = feature
	}
	return registry
}

func validateP22FeatureSurfaceStrings(label string, items []string) error {
	if len(items) == 0 {
		return fmt.Errorf("feature surface audit: %s missing", label)
	}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return fmt.Errorf("feature surface audit: %s contains empty item", label)
		}
		if p22FeatureSurfaceContainsPlaceholder(trimmed) {
			return fmt.Errorf("feature surface audit: %s contains placeholder evidence: %q", label, item)
		}
	}
	return nil
}

func p22FeatureSurfaceContainsPlaceholder(text string) bool {
	lower := strings.ToLower(text)
	for _, token := range []string{"todo", "tbd", "placeholder", "fixme", "???"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func p22FeatureSurfaceCombined(row FeatureSurfaceAuditRow) string {
	return row.Name + " " + row.Decision + " " + strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ") + " " + strings.Join(row.RequiredPromotionEvidence, " ")
}

func p22FeatureSurfaceHasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
