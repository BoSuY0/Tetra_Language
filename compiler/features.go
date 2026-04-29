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
// reflect the v0.2.0 supported surface, while future work stays planned or
// post-v1 until promoted with release-gate evidence.
func FeatureRegistry() []FeatureInfo {
	features := []FeatureInfo{
		{
			ID:        "cli.core",
			Name:      "Core CLI workflows",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "check/build/run/fmt/test/doc/doctor/targets/smoke/eco/clean/version local workflows",
			Stability: "supported in the current v0.2.0 local profile",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/cli_contracts.md"},
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
			ID:        "targets.wasm-build-only",
			Name:      "WASM build-only targets",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "wasm32-wasi and wasm32-web build-only smoke/report validation; no runtime execution guarantee",
			Stability: "build-only in the current v0.2.0 local profile",
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
			Scope:     "Level 0 callable surface: function type references and narrow symbol-backed non-capturing callable paths",
			Stability: "current constrained MVP; full first-class function values remain out of scope",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_feature_status.md"},
		},
		{
			ID:        "language.callable-level1",
			Name:      "Callable Level 1 non-capturing expansion",
			Status:    FeatureStatusExperimental,
			Scope:     "experimental non-capturing callable expansion beyond the Level 0 MVP, limited to symbol-backed immutable function-typed values with stable diagnostics",
			Stability: "experimental; not part of the v0.2.0 stable baseline and not a full first-class function-value claim",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"},
		},
		{
			ID:        "language.callable-level2",
			Name:      "Callable Level 2 captured closure and escape model",
			Status:    FeatureStatusPlanned,
			Scope:     "planned/experimental design space for captured closures, broader callback movement, lifetime validation, and ABI evidence before promotion",
			Stability: "planned; no current v0.2.0 support guarantee and no full first-class callable semantics until gated",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"},
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
			Scope:     "local verify, lock generation/validation, pack/unpack, vault, and publish metadata fixtures",
			Stability: "local tooling support only; network ecosystem is not implied",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/user/eco_package_guide.md"},
		},
		{
			ID:        "stdlib.experimental-mirrors",
			Name:      "Experimental standard-library mirrors",
			Status:    FeatureStatusExperimental,
			Since:     "v0.2.0",
			Scope:     "lib.experimental mirrors for selected lib.core modules",
			Stability: "experimental mirrors with no stability guarantees",
			Docs:      []string{"docs/user/standard_library_guide.md"},
		},
		{
			ID:        "language.enum-payload-match",
			Name:      "Enum payload constructors and exhaustive match/catch",
			Status:    FeatureStatusExperimental,
			Scope:     "next-cycle experimental promotion of positional enum payload constructors/bindings plus exhaustive enum match/catch coverage",
			Stability: "experimental next-cycle slice; not part of the current v0.2.0 stable baseline until gated",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"},
		},
		{
			ID:        "ui.metadata-v1",
			Name:      "UI metadata v1 surface",
			Status:    FeatureStatusPlanned,
			Scope:     "checked view/state metadata, deterministic UI JSON, preview artifacts, and shell sidecars",
			Stability: "planned release behavior pending gate evidence",
			Docs:      []string{"docs/spec/v1_feature_status.md", "docs/user/wasm_ui_guide.md"},
		},
		{
			ID:        "wasm.runtime-execution",
			Name:      "WASM runtime execution",
			Status:    FeatureStatusPlanned,
			Scope:     "WASI runner and browser smoke execution beyond build-only validation",
			Stability: "planned until runner/browser automation evidence is present",
			Docs:      []string{"docs/backend/wasm_backend_plan.md", "docs/user/wasm_ui_guide.md"},
		},
		{
			ID:        "language.full-v1-guarantees",
			Name:      "Full v1.0 language guarantees",
			Status:    FeatureStatusPlanned,
			Scope:     "complete v1.0 release contract after mandatory release-gate evidence",
			Stability: "future label while repository remains on the v0.2.0 profile",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/v1_scope.md"},
		},
		{
			ID:        "language.full-first-class-callables",
			Name:      "Full first-class callable/function-value semantics",
			Status:    FeatureStatusPostV1,
			Scope:     "arbitrary escape, passing, storing, full capture matrix, and ABI redesign",
			Stability: "deferred post-v1",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/spec/v1_feature_status.md"},
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
			Name:      "Distributed actors and full structured concurrency",
			Status:    FeatureStatusPostV1,
			Scope:     "distributed actors plus full async cancellation and structured concurrency guarantees",
			Stability: "outside the current v0.2.0 support claim",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/user/async_actors_guide.md"},
		},
		{
			ID:        "ui.native-runtime",
			Name:      "Native UI runtime widgets and event dispatch",
			Status:    FeatureStatusPostV1,
			Scope:     "native widget rendering and runtime UI event dispatch",
			Stability: "outside the current v0.2.0 support claim",
			Docs:      []string{"docs/spec/current_supported_surface.md", "docs/user/wasm_ui_guide.md"},
		},
	}
	out := make([]FeatureInfo, len(features))
	copy(out, features)
	for i := range out {
		out[i].Docs = append([]string(nil), features[i].Docs...)
	}
	return out
}
