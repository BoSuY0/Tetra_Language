package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/internal/toon"
)

func TestValidateManifestAcceptsGeneratedShape(t *testing.T) {
	raw, err := os.ReadFile(filepath.FromSlash("../../../docs/generated/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := runManifestValidator(t, string(raw))
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateManifestAcceptsTOONGeneratedShape(t *testing.T) {
	raw, err := os.ReadFile(filepath.FromSlash("../../../docs/generated/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	toonRaw, err := toon.ConvertJSONToTOON(raw, toon.Options{Strict: true, Deterministic: true})
	if err != nil {
		t.Fatalf("json->toon: %v", err)
	}
	if err := validateManifestFormat(toonRaw, "toon"); err != nil {
		t.Fatalf("validateManifestFormat TOON: %v\n%s", err, toonRaw)
	}
}

func TestValidateFeaturesAcceptsMachineReadableCurrentFutureClaims(t *testing.T) {
	features := []featureManifest{
		{
			ID:        "cli.core",
			Name:      "CLI",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "core CLI",
			Stability: "supported",
			Docs:      []string{"docs/spec/core/current_supported_surface.md"},
		},
		{
			ID:        "language.flow",
			Name:      "Flow",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "flow syntax",
			Stability: "supported",
			Docs:      []string{"docs/spec/flow/flow_syntax_v1.md"},
		},
		{
			ID:     "language.generics-mvp",
			Name:   "Generics MVP",
			Status: "current",
			Since:  "v0.2.0",
			Scope: ("statically monomorphized generic functions with no runtime " +
				"generic values or dynamic dispatch"),
			Stability: "supported static MVP; generic structs remain future/post-v1",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:        "language.protocol-conformance-mvp",
			Name:      "Protocol conformance MVP",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "checked statically with generic requirement signature shape and no witness tables",
			Stability: "dynamic dispatch remain post-v1",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:        "language.callable-mvp",
			Name:      "Callable MVP",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "Level 0 callable surface",
			Stability: "current constrained MVP",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
			},
		},
		{
			ID:        "targets.wasm-artifact-preflight",
			Name:      "WASM artifact/import preflight",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "artifact/import smoke",
			Stability: "supported",
			Docs:      []string{"docs/backend/wasm_backend_plan.md"},
		},
		{
			ID:        "stdlib.experimental-mirrors",
			Name:      "Standard-library compatibility mirrors",
			Status:    "current",
			Since:     "v0.4.0",
			Scope:     "production compatibility mirrors forward to lib.core modules",
			Stability: "stable callers should import lib.core directly",
			Docs: []string{
				"docs/spec/standard_library/stdlib.md",
				"docs/spec/standard_library/stdlib_naming_versioning.md",
				"docs/user/platform/standard_library_guide.md",
			},
		},
		{
			ID:     "language.callable-level1",
			Name:   "Callable Level 1",
			Status: "current",
			Since:  "v0.4.0",
			Scope: ("production non-capturing symbol-backed callable Level 1 " +
				"with function-typed locals, aliases, callbacks, and " +
				"symbol-backed returns"),
			Stability: "captured closure escape and full first-class function values remain out of scope",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/policy/v1_feature_status.md",
			},
		},
		{
			ID:     "language.enum-payload-match",
			Name:   "Enum payload",
			Status: "current",
			Since:  "v0.3.0",
			Scope: ("positional enum payload constructors and payload bindings " +
				"for match/catch/if-let, with exhaustive unguarded enum " +
				"match/catch"),
			Stability: "nested destructuring patterns and guard expansion remain future/post-v1",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/flow/v0_3_scope.md",
			},
		},
		{
			ID:     "language.protocol-bound-generics-static",
			Name:   "Static protocol-bound generics",
			Status: "current",
			Since:  "v0.3.0",
			Scope: ("validated statically during monomorphization with " +
				"same-module and cross-module impl conformance plus " +
				"visibility diagnostics"),
			Stability: ("calling protocol requirements through generic bounds and " +
				"dynamic dispatch remain unsupported"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/v0_3_scope.md",
				"docs/spec/flow/flow_syntax_v1.md",
			},
		},
		{
			ID:     "language.ownership-markers-mvp",
			Name:   "Ownership markers MVP",
			Status: "current",
			Since:  "v0.2.0",
			Scope: ("conservative borrow/inout/consume marker checks with " +
				"use-after-consume and borrow escape diagnostics"),
			Stability: "supported conservative MVP; not a full SSA lifetime solver",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "language.resource-lifetime-mvp",
			Name:   "Resource lifetime MVP",
			Status: "current",
			Since:  "v0.2.0",
			Scope: ("conservative resource finalization checks for task handles, " +
				"task groups, island handles, region-backed slices, and " +
				"structs containing them, including double-use and ambiguous " +
				"provenance diagnostics"),
			Stability: ("supported conservative MVP; tracks common local scope and " +
				"control-flow merge cases, but is not a full SSA lifetime " +
				"solver"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "actors.task-transfer-safety",
			Name:   "Actor/task transfer safety MVP",
			Status: "current",
			Since:  "v0.2.0",
			Scope: ("conservative actor/task ownership transfer checks for " +
				"worker entrypoints and use-after-transfer diagnostics"),
			Stability: "supported conservative local MVP; distributed actors remain outside current support",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "language.lifetime-ssa",
			Name:   "Lifetime SSA local join solver",
			Status: "current",
			Since:  "v0.4.0",
			Scope: ("production SSA-like local lifetime join analysis for " +
				"ownership consume state, resource finalization state, " +
				"branch/match/loop flow snapshots, and maybe-consumed " +
				"diagnostics"),
			Stability: ("current local/control-flow solver; richer interprocedural " +
				"lifetime proofs, broad alias modeling, race proofs, and " +
				"full formal lifetime guarantees remain under full-v1 scope"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		validSafetyProductionCoreFeature(),
		validRAMContractFeature(),
		{
			ID:     "language.callable-level2",
			Name:   "Callable Level 2",
			Status: "current",
			Since:  "v0.4.0",
			Scope: ("production captured closure Level 2 slice with " +
				"function-typed locals called directly"),
			Stability: ("captured callback passing and full first-class callable " +
				"semantics remain out of scope"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/policy/v1_feature_status.md",
			},
		},
		{
			ID:        "ui.metadata-v1",
			Name:      "UI metadata v0.4.0",
			Status:    "current",
			Since:     "v0.4.0",
			Scope:     "production UI metadata contract with deterministic tetra.ui.v0.4.0 JSON",
			Stability: "web command dispatch; native widgets remain post-v1",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/ui/ui_v0.4.0.md",
				"docs/user/surface/wasm_ui_guide.md",
			},
		},
		{
			ID:     "ui.toolkit-core",
			Name:   "UI Toolkit Core",
			Status: "current",
			Since:  "v0.4.0",
			Scope: ("production platform-independent UI Toolkit Core contract " +
				"for tetra.ui.toolkit.v1 with widget model, layout model, " +
				"accessibility model, event dispatch, state binding/update, " +
				"and runtime trace artifacts"),
			Stability: ("rejects metadata-only, runtime-less, native-shell " +
				"sidecar-only, web-only evidence; no GTK/Qt/OS platform " +
				"backend production or full cross-platform UI claim"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/ui/ui_toolkit_core.md",
				"docs/spec/ui/ui_v0.4.0.md",
			},
		},
		{
			ID:        "wasm.runtime-execution",
			Name:      "WASM runtime execution",
			Status:    "current",
			Since:     "v0.4.0",
			Scope:     "production WASI runner and browser-backed wasm32-web execution",
			Stability: "supported with runner/browser availability diagnostics",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/backend/wasm_backend_plan.md",
				"docs/user/surface/wasm_ui_guide.md",
			},
		},
		{
			ID:        "language.full-v1-guarantees",
			Name:      "v1",
			Status:    "planned",
			Scope:     "v1",
			Stability: "planned",
			Docs:      []string{"docs/spec/flow/v1_scope.md"},
		},
		{
			ID:        "eco.distributed-network",
			Name:      "EcoNet",
			Status:    "post-v1",
			Scope:     "network",
			Stability: "deferred",
			Docs:      []string{"docs/release/policy/post_v1_promotion_checklist.md"},
		},
		{
			ID:        "language.full-first-class-callables",
			Name:      "Callables",
			Status:    "current",
			Since:     "v0.4.0",
			Scope:     "safe by-value first-class callable semantics",
			Stability: "current safe-capture model",
			Docs:      []string{"docs/spec/policy/v1_feature_status.md"},
		},
		{
			ID:        "ui.surface-macos-x64",
			Name:      "macOS Surface",
			Status:    "unsupported",
			Scope:     "unsupported for Surface v1",
			Stability: "not promoted without target evidence",
			Docs:      []string{"docs/spec/core/current_supported_surface.md"},
		},
		{
			ID:        "ui.surface-legacy-metadata",
			Name:      "Legacy UI metadata",
			Status:    "legacy_compatibility",
			Scope:     "legacy metadata compatibility",
			Stability: "not a new current runtime",
			Docs:      []string{"docs/spec/core/current_supported_surface.md"},
		},
		{
			ID:        "ui.surface-next",
			Name:      "Surface next",
			Status:    "release_candidate",
			Scope:     "candidate evidence under review",
			Stability: "not current until release gate passes",
			Docs:      []string{"docs/spec/core/current_supported_surface.md"},
		},
	}
	if err := validateFeatures(features); err != nil {
		t.Fatalf("validateFeatures: %v", err)
	}
}

func validSafetyProductionCoreFeature() featureManifest {
	return featureManifest{
		ID:     "safety.production-core",
		Name:   "Production safety core",
		Status: "current",
		Since:  "v0.4.0",
		Scope: ("production local safety model for " +
			"ownership/lifetime/borrow/consume/inout checks, resource " +
			"finalization, callable escape diagnostics, " +
			"effects/capabilities/privacy/consent/budget policy, unsafe " +
			"boundaries, actor/task transfer safety, and " +
			"pointer/MMIO/memory capability gates, Memory Production " +
			"Core v1 report evidence through compiler-owned facts rather " +
			"than report-reconstructed truth, a memory cost model with " +
			"zero_cost_proven, dynamic_check_required, " +
			"instrumentation_only, unsupported_rejected, and " +
			"conservative_fallback report classes, a memory fuzz oracle " +
			"with Tier 1 short CI smoke, Tier 2 nightly fuzz, Tier 3 " +
			"release-blocking focused memory fuzz, explicit oracle " +
			"categories, memory production final audit with artifact map " +
			"and explicit nonclaims, validate-island-proof " +
			"independent-ish verifier evidence, --islands-debug " +
			"sanitizer smoke, island-proof-fuzz-summary deterministic " +
			"mutation evidence, leak/resource finalization evidence, and " +
			"an integrated Memory/Islands/Surface release gate with " +
			"memory-islands-surface-production-manifest.json and " +
			"artifact-hashes.json, and no Memory 100% claim"),
		Stability: ("release-gated current profile with explicit diagnostics for " +
			"unsupported distributed, cryptographic, formal-proof, " +
			"runtime-wide guarantees, arbitrary unsafe external pointer " +
			"safety, full target parity, all-target Surface support, " +
			"clean release-candidate checkout claims, and no production " +
			"object memory or production persistent memory claim"),
		Docs: []string{
			"docs/spec/core/current_supported_surface.md",
			"docs/spec/runtime/ownership_v1.md",
			"docs/spec/runtime/effects_capabilities_privacy_v1.md",
			"docs/spec/runtime/unsafe.md",
			"docs/spec/memory/memory_report_schema_v1.md",
			"docs/spec/memory/islands.md",
			"docs/design/memory/memory_production_core_v1.md",
			"docs/design/memory/memory_cost_model.md",
			"docs/audits/memory/islands/memory-fuzz-oracle-v1.md",
			"docs/audits/memory/production/memory-production-core-v1-final.md",
			"docs/audits/memory/production/memory-production-core-v1-artifact-map.md",
			"docs/audits/memory/production/memory-production-core-v1-nonclaims.md",
			"docs/release/surface/memory_islands_surface_scope.md",
			"docs/audits/memory/ideal-v0-v1/memory-ideal-vslice-v0-baseline.md",
			"docs/audits/memory/ideal-v0-v1/memory-ideal-vslice-v0-correlation.md",
			"docs/audits/memory/ideal-v0-v1/memory-ideal-vslice-v0-final.md",
			"docs/audits/memory/ideal-v0-v1/memory-ideal-vslice-v1-correlation.md",
			"docs/audits/memory/ideal-v0-v1/memory-ideal-vslice-v1-final.md",
			"docs/audits/memory/ideal-v2-v4/memory-ideal-vslice-v2-correlation.md",
			"docs/audits/memory/ideal-v2-v4/memory-ideal-vslice-v2-final.md",
			"docs/audits/memory/ideal-v2-v4/memory-ideal-vslice-v3-correlation.md",
			"docs/audits/memory/ideal-v2-v4/memory-ideal-vslice-v3-final.md",
		},
	}
}

func validRAMContractFeature() featureManifest {
	return featureManifest{
		ID:     "compiler.ram-contracts",
		Name:   "RAM Contract Compiler reports",
		Status: "current",
		Since:  "v0.4.0",
		Scope: ("RAM Contract Compiler report evidence with " +
			"tetra.ram-contract-report.v1, tetra.memory-grade-report.v1, " +
			"tetra.proof-store-summary.v1, " +
			"tetra.validation-pipeline-coverage.v1, heap-blockers.json, " +
			"copy-blockers.json, ram-contract-fuzz-oracle.json, " +
			"--emit-ram-contract-report, --fail-if-heap, --fail-if-copy, " +
			"--fail-if-unbounded, --memory-budget, --ram-contract, " +
			"TETRA4100, validate-ram-contract-release, and " +
			"ram-contract-linux-x64-smoke.sh"),
		Stability: ("current report/gate contract only; no zero heap for all " +
			"programs claim, no zero-copy for all programs claim, no " +
			"full formal proof claim, no all-target RAM parity claim, no " +
			"production object memory claim, no production persistent " +
			"memory claim, and no performance claim"),
		Docs: []string{
			"docs/design/ram_contract_compiler.md",
			"docs/spec/memory/ram_contract_report_schema.md",
			"docs/user/platform/ram_contracts.md",
			"docs/audits/memory/ram-raw/ram-contract-compiler-readiness.md",
			"docs/audits/memory/ram-raw/ram-contract-compiler-handoff.md",
		},
	}
}

func TestValidateFeaturesRequiresMemoryProductionFinalAuditDocs(t *testing.T) {
	raw, err := os.ReadFile(filepath.FromSlash("../../../docs/generated/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Features []featureManifest `json:"features"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	for i := range manifest.Features {
		if manifest.Features[i].ID != "safety.production-core" {
			continue
		}
		filtered := manifest.Features[i].Docs[:0]
		for _, doc := range manifest.Features[i].Docs {
			if strings.Contains(doc, "memory-production-core-v1-final.md") ||
				strings.Contains(doc, "memory-production-core-v1-artifact-map.md") ||
				strings.Contains(doc, "memory-production-core-v1-nonclaims.md") {
				continue
			}
			filtered = append(filtered, doc)
		}
		manifest.Features[i].Docs = filtered
		break
	}

	err = validateFeatures(manifest.Features)
	if err == nil {
		t.Fatalf("expected safety.production-core final audit doc requirement failure")
	}
	if !strings.Contains(err.Error(), "memory production final audit") &&
		!strings.Contains(err.Error(), "memory-production-core-v1-final.md") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFeaturesRejectsMemory100Overclaims(t *testing.T) {
	feature := validSafetyProductionCoreFeature()
	feature.Scope += ". Memory 100% is guaranteed with no leaks."
	feature.Stability += ". This is a full formal proof with all-target memory parity."

	err := validateFeatures([]featureManifest{
		{
			ID:        "cli.core",
			Name:      "CLI",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "core CLI",
			Stability: "supported",
			Docs:      []string{"docs/spec/core/current_supported_surface.md"},
		},
		{
			ID:        "language.flow",
			Name:      "Flow",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "flow syntax",
			Stability: "supported",
			Docs:      []string{"docs/spec/flow/flow_syntax_v1.md"},
		},
		{
			ID:     "language.generics-mvp",
			Name:   "Generics MVP",
			Status: "current",
			Since:  "v0.2.0",
			Scope: ("statically monomorphized generic functions with no runtime " +
				"generic values or dynamic dispatch"),
			Stability: "supported static MVP; generic structs remain future/post-v1",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:        "language.protocol-conformance-mvp",
			Name:      "Protocol conformance MVP",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "checked statically with generic requirement signature shape and no witness tables",
			Stability: "dynamic dispatch remain post-v1",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:        "language.callable-mvp",
			Name:      "Callable MVP",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "Level 0 callable surface",
			Stability: "current constrained MVP",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
			},
		},
		{
			ID:        "targets.wasm-artifact-preflight",
			Name:      "WASM artifact/import preflight",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "artifact/import smoke",
			Stability: "supported",
			Docs:      []string{"docs/backend/wasm_backend_plan.md"},
		},
		{
			ID:        "stdlib.experimental-mirrors",
			Name:      "Standard-library compatibility mirrors",
			Status:    "current",
			Since:     "v0.4.0",
			Scope:     "production compatibility mirrors forward to lib.core modules",
			Stability: "stable callers should import lib.core directly",
			Docs: []string{
				"docs/spec/standard_library/stdlib.md",
				"docs/spec/standard_library/stdlib_naming_versioning.md",
				"docs/user/platform/standard_library_guide.md",
			},
		},
		{
			ID:     "language.callable-level1",
			Name:   "Callable Level 1",
			Status: "current",
			Since:  "v0.4.0",
			Scope: ("production non-capturing symbol-backed callable Level 1 " +
				"with function-typed locals, aliases, callbacks, and " +
				"symbol-backed returns"),
			Stability: "captured closure escape and full first-class function values remain out of scope",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/policy/v1_feature_status.md",
			},
		},
		{
			ID:     "language.enum-payload-match",
			Name:   "Enum payload",
			Status: "current",
			Since:  "v0.3.0",
			Scope: ("positional enum payload constructors and payload bindings " +
				"for match/catch/if-let, with exhaustive unguarded enum " +
				"match/catch"),
			Stability: "nested destructuring patterns and guard expansion remain future/post-v1",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/flow/v0_3_scope.md",
			},
		},
		{
			ID:     "language.protocol-bound-generics-static",
			Name:   "Static protocol-bound generics",
			Status: "current",
			Since:  "v0.3.0",
			Scope: ("validated statically during monomorphization with " +
				"same-module and cross-module impl conformance plus " +
				"visibility diagnostics"),
			Stability: ("calling protocol requirements through generic bounds and " +
				"dynamic dispatch remain unsupported"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/v0_3_scope.md",
				"docs/spec/flow/flow_syntax_v1.md",
			},
		},
		{
			ID:     "language.ownership-markers-mvp",
			Name:   "Ownership markers MVP",
			Status: "current",
			Since:  "v0.2.0",
			Scope: ("conservative borrow/inout/consume marker checks with " +
				"use-after-consume and borrow escape diagnostics"),
			Stability: "supported conservative MVP; not a full SSA lifetime solver",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "language.resource-lifetime-mvp",
			Name:   "Resource lifetime MVP",
			Status: "current",
			Since:  "v0.2.0",
			Scope: ("conservative resource finalization checks for task handles, " +
				"task groups, island handles, region-backed slices, and " +
				"structs containing them, including double-use and ambiguous " +
				"provenance diagnostics"),
			Stability: ("supported conservative MVP; tracks common local scope and " +
				"control-flow merge cases, but is not a full SSA lifetime " +
				"solver"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "actors.task-transfer-safety",
			Name:   "Actor/task transfer safety MVP",
			Status: "current",
			Since:  "v0.2.0",
			Scope: ("conservative actor/task ownership transfer checks for " +
				"worker entrypoints and use-after-transfer diagnostics"),
			Stability: "supported conservative local MVP; distributed actors remain outside current support",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "language.lifetime-ssa",
			Name:   "Lifetime SSA local join solver",
			Status: "current",
			Since:  "v0.4.0",
			Scope: ("production SSA-like local lifetime join analysis for " +
				"ownership consume state, resource finalization state, " +
				"branch/match/loop flow snapshots, and maybe-consumed " +
				"diagnostics"),
			Stability: ("current local/control-flow solver; richer interprocedural " +
				"lifetime proofs, broad alias modeling, race proofs, and " +
				"full formal lifetime guarantees remain under full-v1 scope"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		feature,
		validRAMContractFeature(),
		{
			ID:     "language.callable-level2",
			Name:   "Callable Level 2",
			Status: "current",
			Since:  "v0.4.0",
			Scope: ("production captured closure Level 2 slice with " +
				"function-typed locals called directly"),
			Stability: ("captured callback passing and full first-class callable " +
				"semantics remain out of scope"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/policy/v1_feature_status.md",
			},
		},
		{
			ID:        "ui.metadata-v1",
			Name:      "UI metadata v0.4.0",
			Status:    "current",
			Since:     "v0.4.0",
			Scope:     "production UI metadata contract with deterministic tetra.ui.v0.4.0 JSON",
			Stability: "web command dispatch; native widgets remain post-v1",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/ui/ui_v0.4.0.md",
				"docs/user/surface/wasm_ui_guide.md",
			},
		},
		{
			ID:     "ui.toolkit-core",
			Name:   "UI Toolkit Core",
			Status: "current",
			Since:  "v0.4.0",
			Scope: ("production platform-independent UI Toolkit Core contract " +
				"for tetra.ui.toolkit.v1 with widget model, layout model, " +
				"accessibility model, event dispatch, state binding/update, " +
				"and runtime trace artifacts"),
			Stability: ("rejects metadata-only, runtime-less, native-shell " +
				"sidecar-only, web-only evidence; no GTK/Qt/OS platform " +
				"backend production or full cross-platform UI claim"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/ui/ui_toolkit_core.md",
				"docs/spec/ui/ui_v0.4.0.md",
			},
		},
		{
			ID:        "wasm.runtime-execution",
			Name:      "WASM runtime execution",
			Status:    "current",
			Since:     "v0.4.0",
			Scope:     "production WASI runner and browser-backed wasm32-web execution",
			Stability: "supported with runner/browser availability diagnostics",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/backend/wasm_backend_plan.md",
				"docs/user/surface/wasm_ui_guide.md",
			},
		},
		{
			ID:        "language.full-v1-guarantees",
			Name:      "v1",
			Status:    "planned",
			Scope:     "v1",
			Stability: "planned",
			Docs:      []string{"docs/spec/flow/v1_scope.md"},
		},
		{
			ID:        "eco.distributed-network",
			Name:      "EcoNet",
			Status:    "post-v1",
			Scope:     "network",
			Stability: "deferred",
			Docs:      []string{"docs/release/policy/post_v1_promotion_checklist.md"},
		},
		{
			ID:        "language.full-first-class-callables",
			Name:      "Callables",
			Status:    "current",
			Since:     "v0.4.0",
			Scope:     "safe by-value first-class callable semantics",
			Stability: "current safe-capture model",
			Docs:      []string{"docs/spec/policy/v1_feature_status.md"},
		},
	})
	if err == nil {
		t.Fatalf("expected Memory100 manifest overclaim failure")
	}
	for _, want := range []string{
		"memory 100%",
		"no leaks",
		"full formal proof",
		"all-target memory parity",
	} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestValidateFeaturesRequiresIntegratedMemoryIslandsSurfaceEvidence(t *testing.T) {
	raw, err := os.ReadFile(filepath.FromSlash("../../../docs/generated/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Features []featureManifest `json:"features"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	for i := range manifest.Features {
		if manifest.Features[i].ID != "safety.production-core" {
			continue
		}
		manifest.Features[i].Scope = strings.ReplaceAll(
			manifest.Features[i].Scope,
			"validate-island-proof",
			"producer-only proof",
		)
		manifest.Features[i].Scope = strings.ReplaceAll(
			manifest.Features[i].Scope,
			"memory-islands-surface-production-manifest.json",
			"combined-report.json",
		)
		manifest.Features[i].Scope = strings.ReplaceAll(
			manifest.Features[i].Scope,
			"island-proof-fuzz-summary",
			"proof fuzz",
		)
		filtered := manifest.Features[i].Docs[:0]
		for _, doc := range manifest.Features[i].Docs {
			if doc == "docs/spec/memory/islands.md" ||
				doc == "docs/release/surface/memory_islands_surface_scope.md" {
				continue
			}
			filtered = append(filtered, doc)
		}
		manifest.Features[i].Docs = filtered
		break
	}

	err = validateFeatures(manifest.Features)
	if err == nil {
		t.Fatalf("expected integrated Memory/Islands/Surface evidence requirement failure")
	}
	for _, want := range []string{"safety.production-core", "validate-island-proof"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("unexpected error: %v, missing %q", err, want)
		}
	}
}

func TestValidateFeaturesRejectsProductionPersistentObjectMemoryClaim(t *testing.T) {
	raw, err := os.ReadFile(filepath.FromSlash("../../../docs/generated/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Features []featureManifest `json:"features"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	for i := range manifest.Features {
		if manifest.Features[i].ID != "safety.production-core" {
			continue
		}
		manifest.Features[i].Scope += (" Production object memory is backed by persistent memory, " +
			"Todium, memoryfield, WAL, FTS, vacuum, retention, stale " +
			"memory, and false memory gates.")
		break
	}

	err = validateFeatures(manifest.Features)
	if err == nil {
		t.Fatalf("expected production persistent/object memory claim failure")
	}
	for _, want := range []string{
		"production object memory",
		"persistent memory",
		"todium",
		"memoryfield",
	} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestValidateFeaturesAllowsPersistentObjectMemoryNonGoal(t *testing.T) {
	raw, err := os.ReadFile(filepath.FromSlash("../../../docs/generated/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Features []featureManifest `json:"features"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	for i := range manifest.Features {
		if manifest.Features[i].ID != "safety.production-core" {
			continue
		}
		manifest.Features[i].Stability += (" Persistent/object memory is an explicit non-goal: no " +
			"production object memory, no production persistent memory, " +
			"and no Todium or memoryfield production claim exists until " +
			"retention/WAL/FTS/vacuum/stale/false-memory gates exist.")
		break
	}

	if err := validateFeatures(manifest.Features); err != nil {
		t.Fatalf("validateFeatures: %v", err)
	}
}

func TestManifestSurfaceRequiresCurrentAndUnsupportedSurfaceRows(t *testing.T) {
	raw, err := os.ReadFile(filepath.FromSlash("../../../docs/generated/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Features []featureManifest `json:"features"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	filtered := manifest.Features[:0]
	for _, feature := range manifest.Features {
		if feature.ID == "ui.surface-windows-x64" {
			continue
		}
		filtered = append(filtered, feature)
	}
	manifest.Features = filtered

	err = validateFeatures(manifest.Features)
	if err == nil {
		t.Fatalf("expected missing Surface unsupported target row failure")
	}
	if !strings.Contains(err.Error(), "ui.surface-windows-x64") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestSurfaceRequiresBlockSystemRow(t *testing.T) {
	raw, err := os.ReadFile(filepath.FromSlash("../../../docs/generated/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Features []featureManifest `json:"features"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	filtered := manifest.Features[:0]
	for _, feature := range manifest.Features {
		if feature.ID == "ui.surface-block-system" {
			continue
		}
		filtered = append(filtered, feature)
	}
	manifest.Features = filtered

	err = validateFeatures(manifest.Features)
	if err == nil {
		t.Fatalf("expected missing Surface Block System row failure")
	}
	if !strings.Contains(err.Error(), "ui.surface-block-system") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestSurfaceRequiresMorphCapsuleRow(t *testing.T) {
	raw, err := os.ReadFile(filepath.FromSlash("../../../docs/generated/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Features []featureManifest `json:"features"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	filtered := manifest.Features[:0]
	for _, feature := range manifest.Features {
		if feature.ID == "ui.surface-morph-capsule" {
			continue
		}
		filtered = append(filtered, feature)
	}
	manifest.Features = filtered

	err = validateFeatures(manifest.Features)
	if err == nil {
		t.Fatalf("expected missing Surface Morph Capsule row failure")
	}
	if !strings.Contains(err.Error(), "ui.surface-morph-capsule") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFeaturesRejectsFutureStatusPromotionWithoutRegistryUpdate(t *testing.T) {
	features := []featureManifest{
		{
			ID:        "cli.core",
			Name:      "CLI",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "core CLI",
			Stability: "supported",
			Docs:      []string{"docs/spec/core/current_supported_surface.md"},
		},
		{
			ID:        "language.flow",
			Name:      "Flow",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "flow syntax",
			Stability: "supported",
			Docs:      []string{"docs/spec/flow/flow_syntax_v1.md"},
		},
		{
			ID:     "language.generics-mvp",
			Name:   "Generics MVP",
			Status: "current",
			Since:  "v0.2.0",
			Scope: ("statically monomorphized generic functions with no runtime " +
				"generic values or dynamic dispatch"),
			Stability: "supported static MVP; generic structs remain future/post-v1",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:        "language.protocol-conformance-mvp",
			Name:      "Protocol conformance MVP",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "checked statically with generic requirement signature shape and no witness tables",
			Stability: "dynamic dispatch remain post-v1",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:        "language.callable-mvp",
			Name:      "Callable MVP",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "Level 0 callable surface",
			Stability: "current constrained MVP",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
			},
		},
		{
			ID:        "targets.wasm-artifact-preflight",
			Name:      "WASM artifact/import preflight",
			Status:    "current",
			Since:     "v0.2.0",
			Scope:     "artifact/import smoke",
			Stability: "supported",
			Docs:      []string{"docs/backend/wasm_backend_plan.md"},
		},
		{
			ID:        "stdlib.experimental-mirrors",
			Name:      "Standard-library compatibility mirrors",
			Status:    "current",
			Since:     "v0.4.0",
			Scope:     "production compatibility mirrors forward to lib.core modules",
			Stability: "stable callers should import lib.core directly",
			Docs: []string{
				"docs/spec/standard_library/stdlib.md",
				"docs/spec/standard_library/stdlib_naming_versioning.md",
				"docs/user/platform/standard_library_guide.md",
			},
		},
		{
			ID:     "language.callable-level1",
			Name:   "Callable Level 1",
			Status: "current",
			Since:  "v0.4.0",
			Scope: ("production non-capturing symbol-backed callable Level 1 " +
				"with function-typed locals, aliases, callbacks, and " +
				"symbol-backed returns"),
			Stability: "captured closure escape and full first-class function values remain out of scope",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/policy/v1_feature_status.md",
			},
		},
		{
			ID:     "language.enum-payload-match",
			Name:   "Enum payload",
			Status: "current",
			Since:  "v0.3.0",
			Scope: ("positional enum payload constructors and payload bindings " +
				"for match/catch/if-let, with exhaustive unguarded enum " +
				"match/catch"),
			Stability: "nested destructuring patterns and guard expansion remain future/post-v1",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/flow/v0_3_scope.md",
			},
		},
		{
			ID:     "language.protocol-bound-generics-static",
			Name:   "Static protocol-bound generics",
			Status: "current",
			Since:  "v0.3.0",
			Scope: ("validated statically during monomorphization with " +
				"same-module and cross-module impl conformance plus " +
				"visibility diagnostics"),
			Stability: ("calling protocol requirements through generic bounds and " +
				"dynamic dispatch remain unsupported"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/v0_3_scope.md",
				"docs/spec/flow/flow_syntax_v1.md",
			},
		},
		{
			ID:     "language.ownership-markers-mvp",
			Name:   "Ownership markers MVP",
			Status: "current",
			Since:  "v0.2.0",
			Scope: ("conservative borrow/inout/consume marker checks with " +
				"use-after-consume and borrow escape diagnostics"),
			Stability: "supported conservative MVP; not a full SSA lifetime solver",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "language.resource-lifetime-mvp",
			Name:   "Resource lifetime MVP",
			Status: "current",
			Since:  "v0.2.0",
			Scope: ("conservative resource finalization checks for task handles, " +
				"task groups, island handles, region-backed slices, and " +
				"structs containing them, including double-use and ambiguous " +
				"provenance diagnostics"),
			Stability: ("supported conservative MVP; tracks common local scope and " +
				"control-flow merge cases, but is not a full SSA lifetime " +
				"solver"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "actors.task-transfer-safety",
			Name:   "Actor/task transfer safety MVP",
			Status: "current",
			Since:  "v0.2.0",
			Scope: ("conservative actor/task ownership transfer checks for " +
				"worker entrypoints and use-after-transfer diagnostics"),
			Stability: "supported conservative local MVP; distributed actors remain outside current support",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		validSafetyProductionCoreFeature(),
		validRAMContractFeature(),
		{
			ID:        "language.lifetime-ssa",
			Name:      "Lifetime SSA solver",
			Status:    "planned",
			Scope:     "stale planned lifetime solver fixture",
			Stability: "unsupported stale fixture",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "language.callable-level2",
			Name:   "Callable Level 2",
			Status: "current",
			Since:  "v0.4.0",
			Scope: ("production captured closure Level 2 slice with " +
				"function-typed locals called directly"),
			Stability: ("captured callback passing and full first-class callable " +
				"semantics remain out of scope"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/policy/v1_feature_status.md",
			},
		},
		{
			ID:        "ui.metadata-v1",
			Name:      "UI metadata v0.4.0",
			Status:    "current",
			Since:     "v0.4.0",
			Scope:     "production UI metadata contract with deterministic tetra.ui.v0.4.0 JSON",
			Stability: "web command dispatch; native widgets remain post-v1",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/ui/ui_v0.4.0.md",
				"docs/user/surface/wasm_ui_guide.md",
			},
		},
		{
			ID:     "ui.toolkit-core",
			Name:   "UI Toolkit Core",
			Status: "current",
			Since:  "v0.4.0",
			Scope: ("production platform-independent UI Toolkit Core contract " +
				"for tetra.ui.toolkit.v1 with widget model, layout model, " +
				"accessibility model, event dispatch, state binding/update, " +
				"and runtime trace artifacts"),
			Stability: ("rejects metadata-only, runtime-less, native-shell " +
				"sidecar-only, web-only evidence; no GTK/Qt/OS platform " +
				"backend production or full cross-platform UI claim"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/ui/ui_toolkit_core.md",
				"docs/spec/ui/ui_v0.4.0.md",
			},
		},
		{
			ID:        "wasm.runtime-execution",
			Name:      "WASM runtime execution",
			Status:    "current",
			Since:     "v0.4.0",
			Scope:     "production WASI runner and browser-backed wasm32-web execution",
			Stability: "supported with runner/browser availability diagnostics",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/backend/wasm_backend_plan.md",
				"docs/user/surface/wasm_ui_guide.md",
			},
		},
		{
			ID:        "language.full-v1-guarantees",
			Name:      "v1",
			Status:    "planned",
			Scope:     "v1",
			Stability: "planned",
			Docs:      []string{"docs/spec/flow/v1_scope.md"},
		},
		{
			ID:        "eco.distributed-network",
			Name:      "EcoNet",
			Status:    "post-v1",
			Scope:     "network",
			Stability: "deferred",
			Docs:      []string{"docs/release/policy/post_v1_promotion_checklist.md"},
		},
		{
			ID:        "language.full-first-class-callables",
			Name:      "Callables",
			Status:    "current",
			Since:     "v0.4.0",
			Scope:     "safe by-value first-class callable semantics",
			Stability: "current safe-capture model",
			Docs:      []string{"docs/spec/policy/v1_feature_status.md"},
		},
	}
	err := validateFeatures(features)
	if err == nil {
		t.Fatalf("expected future status promotion failure")
	}
	if !strings.Contains(err.Error(), "language.lifetime-ssa") ||
		!strings.Contains(err.Error(), "want current") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateManifestRejectsNullTargets(t *testing.T) {
	manifest := `{"compiler_version":"v0.6.0","targets":null,"builtins":[],"runtime_abi":{}}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "targets must be an array") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsUnknownFields(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    ` + manifestTargetWithImportsJSON("linux-x64", "linux", "x64", "sysv", "elf", "", false) + `,
    ` + manifestTargetWithImportsJSON(
		"windows-x64",
		"windows",
		"x64",
		"win64",
		"pe",
		".exe",
		true,
	) + `,
    ` + manifestTargetWithImportsJSON("macos-x64", "macos", "x64", "sysv", "macho", "", false) + `
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never","extra":true}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","macos-x64","windows-x64"],
    "actors_required_symbols": ` + manifestActorsRequiredSymbolsJSON() + `,
    "time_required_symbols": ` + manifestTimeRequiredSymbolsJSON() + `,
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsDuplicateBuiltin(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [
    {"name":"core.print","return_type":"i32","unsafe_policy":"never"},
    {"name":"core.print","return_type":"i32","unsafe_policy":"never"}
  ],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ` + manifestActorsRequiredSymbolsJSON() + `,
    "time_required_symbols": ` + manifestTimeRequiredSymbolsJSON() + `,
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate builtin core.print") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsMissingRuntimeSymbols(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": [],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "actors_required_symbols must not be empty") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsMissingTimeRuntimeSymbols(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ` + manifestActorsRequiredSymbolsJSON() + `,
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "time_required_symbols must not be empty") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateRuntimeABIRejectsMissingSurfaceRuntimeSymbols(t *testing.T) {
	abi := validRuntimeABIManifestForTest()
	abi.SurfaceRequiredSymbols = nil

	err := validateRuntimeABI(abi, validRuntimeTargetsForTest())
	if err == nil {
		t.Fatalf("expected surface_required_symbols validation error")
	}
	if !strings.Contains(err.Error(), "surface_required_symbols must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRuntimeABIRejectsWrongSurfaceRuntimeSymbols(t *testing.T) {
	abi := validRuntimeABIManifestForTest()
	abi.SurfaceRequiredSymbols = []string{"__tetra_surface_open"}

	err := validateRuntimeABI(abi, validRuntimeTargetsForTest())
	if err == nil {
		t.Fatalf("expected surface_required_symbols set validation error")
	}
	if !strings.Contains(err.Error(), "surface_required_symbols got") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateManifestRejectsInvalidUnsafePolicy(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"sometimes"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ` + manifestActorsRequiredSymbolsJSON() + `,
    "time_required_symbols": ` + manifestTimeRequiredSymbolsJSON() + `,
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "invalid unsafe_policy") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsPartialTargetSurface(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64"],
    "actors_required_symbols": ` + manifestActorsRequiredSymbolsJSON() + `,
    "time_required_symbols": ` + manifestTimeRequiredSymbolsJSON() + `,
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "targets got") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsPartialRuntimeABI(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn"],
    "time_required_symbols": ` + manifestTimeRequiredSymbolsJSON() + `,
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "actors_required_symbols got") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsUnsortedTargets(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ` + manifestActorsRequiredSymbolsJSON() + `,
    "time_required_symbols": ` + manifestTimeRequiredSymbolsJSON() + `,
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "targets must follow buildable target order") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsInflatedMemoryCapabilityClaims(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    ` + manifestMemoryTargetsJSON() + `
  ],
  "builtins": [
    ` + memoryBuiltinJSON() + `
  ],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","macos-x64","windows-x64"],
    "actors_required_symbols": ` + manifestActorsRequiredSymbolsJSON() + `,
    "actor_state_required_symbols": ["__tetra_actor_state_load","__tetra_actor_state_store"],
    "task_required_symbols": ` + manifestTaskRequiredSymbolsJSON() + `,
    "task_group_required_symbols": ` + manifestTaskGroupRequiredSymbolsJSON() + `,
    "typed_task_required_symbols": ` + manifestTypedTaskRequiredSymbolsJSON() + `,
    "time_required_symbols": ` + manifestTimeRequiredSymbolsJSON() + `,
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "surface_required_symbols": ` + manifestSurfaceRequiredSymbolsJSON() + `,
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  },
  "features": []
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected inflated memory capability claim rejection")
	}
	if !strings.Contains(string(out), "runtime memory claim") {
		t.Fatalf("expected runtime memory claim rejection, got err=%v out=%s", err, out)
	}
}

func TestValidateManifestRejectsUnsortedBuiltins(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [
    {"name":"core.z","return_type":"i32","unsafe_policy":"never"},
    {"name":"core.a","return_type":"i32","unsafe_policy":"never"}
  ],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ` + manifestActorsRequiredSymbolsJSON() + `,
    "time_required_symbols": ` + manifestTimeRequiredSymbolsJSON() + `,
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "builtins must be sorted") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func manifestJSONStringArray(values ...string) string {
	raw, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func manifestTargetJSON(
	triple string,
	osName string,
	arch string,
	abi string,
	format string,
) string {
	return strings.Join([]string{
		`{"triple":"`,
		triple,
		`","os":"`,
		osName,
		`","arch":"`,
		arch,
		`","abi":"`,
		abi,
		`","format":"`,
		format,
		`"}`,
	}, "")
}

func manifestTargetWithImportsJSON(
	triple string,
	osName string,
	arch string,
	abi string,
	format string,
	exeExt string,
	collectImports bool,
) string {
	return strings.Join([]string{
		`{"triple":"`,
		triple,
		`","os":"`,
		osName,
		`","arch":"`,
		arch,
		`","abi":"`,
		abi,
		`","format":"`,
		format,
		`","exe_ext":"`,
		exeExt,
		`","collect_imports":`,
		boolString(collectImports),
		`}`,
	}, "")
}

func manifestActorsRequiredSymbolsJSON() string {
	return manifestJSONStringArray(
		"__tetra_entry",
		"__tetra_actor_spawn",
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
		"__tetra_actor_send_slot",
		"__tetra_actor_send_commit",
		"__tetra_actor_recv",
		"__tetra_actor_recv_msg",
		"__tetra_actor_recv_poll",
		"__tetra_actor_recv_until",
		"__tetra_actor_recv_msg_until",
		"__tetra_actor_recv_begin",
		"__tetra_actor_recv_slot",
		"__tetra_actor_recv_count",
		"__tetra_actor_self",
		"__tetra_actor_sender",
		"__tetra_actor_yield_now",
	)
}

func manifestTimeRequiredSymbolsJSON() string {
	return manifestJSONStringArray(
		"__tetra_time_now_ms",
		"__tetra_sleep_ms",
		"__tetra_sleep_until_ms",
		"__tetra_deadline_ms",
		"__tetra_timer_ready_ms",
	)
}

func manifestTaskRequiredSymbolsJSON() string {
	return manifestJSONStringArray(
		"__tetra_task_spawn_i32",
		"__tetra_task_join_i32",
		"__tetra_task_join_result_i32",
		"__tetra_task_join_until_i32",
		"__tetra_task_poll_i32",
		"__tetra_task_is_canceled",
		"__tetra_task_checkpoint",
	)
}

func manifestTaskGroupRequiredSymbolsJSON() string {
	return manifestJSONStringArray(
		"__tetra_task_group_open",
		"__tetra_task_group_close",
		"__tetra_task_group_cancel",
		"__tetra_task_group_current",
		"__tetra_task_group_status",
		"__tetra_task_spawn_group_i32",
	)
}

func manifestTypedTaskRequiredSymbolsJSON() string {
	return manifestJSONStringArray(
		"__tetra_task_result_begin",
		"__tetra_task_result_slot",
		"__tetra_task_result_get",
		"__tetra_task_join_typed_2",
		"__tetra_task_join_typed_3",
		"__tetra_task_join_typed_4",
		"__tetra_task_join_typed_5",
		"__tetra_task_join_typed_6",
		"__tetra_task_join_typed_7",
		"__tetra_task_join_typed_8",
	)
}

func manifestSurfaceRequiredSymbolsJSON() string {
	return manifestJSONStringArray(
		"__tetra_surface_open",
		"__tetra_surface_close",
		"__tetra_surface_poll_event_kind",
		"__tetra_surface_poll_event_x",
		"__tetra_surface_poll_event_y",
		"__tetra_surface_poll_event_button",
		"__tetra_surface_poll_event_into",
		"__tetra_surface_poll_event_text_len",
		"__tetra_surface_poll_event_text_into",
		"__tetra_surface_clipboard_write_text",
		"__tetra_surface_clipboard_read_text_into",
		"__tetra_surface_poll_composition_into",
		"__tetra_surface_begin_frame",
		"__tetra_surface_present_rgba",
		"__tetra_surface_now_ms",
		"__tetra_surface_request_redraw",
	)
}

func manifestMemoryTargetsJSON() string {
	return strings.Join([]string{
		manifestMemoryTargetJSON(
			"linux-x64",
			"supported",
			"linux",
			"x64",
			"sysv",
			"lp64",
			"elf",
			"",
			false,
			"host_native",
			"yes",
			"yes",
			"yes",
			"yes",
			"yes/partial",
			"yes",
			"production/host_runtime",
			true,
		),
		manifestMemoryTargetJSON(
			"windows-x64",
			"supported",
			"windows",
			"x64",
			"win64",
			"llp64",
			"pe",
			".exe",
			true,
			"host_native",
			"yes",
			"yes",
			"host-required",
			"host-required",
			"host-required",
			"host-required",
			"build_lower_only unless run",
			false,
		),
		manifestMemoryTargetJSON(
			"macos-x64",
			"supported",
			"macos",
			"x64",
			"sysv",
			"lp64",
			"macho",
			"",
			false,
			"host_native",
			"yes",
			"yes",
			"host-required",
			"host-required",
			"host-required",
			"host-required",
			"build_lower_only unless run",
			false,
		),
		manifestMemoryTargetJSON(
			"wasm32-wasi",
			"supported",
			"wasi",
			"wasm32",
			"wasi",
			"ilp32",
			"wasm",
			".wasm",
			false,
			"wasi_runner",
			"yes",
			"yes",
			"runner-smoke if available",
			"safe-only",
			"limited",
			"wasm rules",
			"artifact/runtime tiered",
			false,
		),
		manifestMemoryTargetJSON(
			"wasm32-web",
			"supported",
			"web",
			"wasm32",
			"web",
			"ilp32",
			"wasm",
			".wasm",
			false,
			"web_runner",
			"yes",
			"yes",
			"browser-smoke if available",
			"safe-only",
			"limited",
			"wasm rules",
			"artifact/runtime tiered",
			false,
		),
		manifestInflatedBuildOnlyMemoryTargetJSON(),
		manifestMemoryTargetJSON(
			"linux-x32",
			"build_only",
			"linux",
			"x64",
			"x32-sysv",
			"x32",
			"elf",
			"",
			false,
			"host_probed",
			"yes",
			"yes",
			"no/host-dependent",
			"partial",
			"partial",
			"special",
			"build_lower_only",
			false,
		),
	}, ",\n    ")
}

func manifestMemoryTargetJSON(
	triple string,
	status string,
	osName string,
	arch string,
	abi string,
	dataModel string,
	format string,
	exeExt string,
	collectImports bool,
	runMode string,
	memoryBuild string,
	memoryLower string,
	memoryRun string,
	memoryRawDiagnostics string,
	memoryRegionLowering string,
	memoryAlignmentSemantics string,
	memoryClaimLevel string,
	includeEvidence bool,
) string {
	parts := []string{
		`{"triple":"` + triple + `"`,
		`"status":"` + status + `"`,
		`"os":"` + osName + `"`,
		`"arch":"` + arch + `"`,
		`"abi":"` + abi + `"`,
		`"data_model":"` + dataModel + `"`,
		`"format":"` + format + `"`,
		`"exe_ext":"` + exeExt + `"`,
		`"collect_imports":` + boolString(collectImports),
		`"run_mode":"` + runMode + `"`,
		`"memory_build":"` + memoryBuild + `"`,
		`"memory_lower":"` + memoryLower + `"`,
		`"memory_run":"` + memoryRun + `"`,
		`"memory_raw_diagnostics":"` + memoryRawDiagnostics + `"`,
		`"memory_region_lowering":"` + memoryRegionLowering + `"`,
		`"memory_alignment_semantics":"` + memoryAlignmentSemantics + `"`,
		`"memory_claim_level":"` + memoryClaimLevel + `"`,
	}
	if includeEvidence {
		parts = append(parts, `"evidence_artifacts":`+linuxX64EvidenceArtifactsJSON())
	}
	return strings.Join(parts, ",") + `}`
}

func manifestInflatedBuildOnlyMemoryTargetJSON() string {
	parts := []string{
		`{"triple":"linux-x86"`,
		`"status":"build_only"`,
		`"os":"linux"`,
		`"arch":"x86"`,
		`"abi":"i386-sysv"`,
		`"data_model":"ilp32"`,
		`"format":"elf"`,
		`"exe_ext":""`,
		`"collect_imports":false`,
		`"run_mode":"host_probed"`,
		`"runtime_status":"partial_build_only"`,
		`"memory_build":"yes"`,
		`"memory_lower":"yes"`,
		`"memory_run":"yes"`,
		`"memory_raw_diagnostics":"partial"`,
		`"memory_region_lowering":"partial"`,
		`"memory_alignment_semantics":"partial"`,
		`"memory_claim_level":"production/host_runtime"`,
	}
	return strings.Join(parts, ",") + `}`
}

func linuxX64EvidenceArtifactsJSON() string {
	return manifestJSONStringArray(
		"targets.json",
		"linux-x64-abi.json",
		"linux-x64-atomic-stress.json",
		"linux-x64-fuzz.json",
		"linux-x64-runner.json",
		"linux-native-targets-brutal.json",
		"artifact-hashes.json",
	)
}

func memoryBuiltinJSON() string {
	parts := []string{
		`{"name":"core.load_i32"`,
		`"param_types":["ptr","cap.mem"]`,
		`"return_type":"i32"`,
		`"effects":["mem"]`,
		`"unsafe_policy":"always"`,
	}
	return strings.Join(parts, ",") + `}`
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func validRuntimeABIManifestForTest() runtimeABIManifest {
	return runtimeABIManifest{
		ReservedPrefix:         "__tetra_",
		ActorsSupportedTargets: []string{"linux-x64", "macos-x64", "windows-x64"},
		ActorsRequiredSymbols: []string{
			"__tetra_entry",
			"__tetra_actor_spawn",
			"__tetra_actor_send",
			"__tetra_actor_send_msg",
			"__tetra_actor_send_begin",
			"__tetra_actor_send_slot",
			"__tetra_actor_send_commit",
			"__tetra_actor_recv",
			"__tetra_actor_recv_msg",
			"__tetra_actor_recv_poll",
			"__tetra_actor_recv_until",
			"__tetra_actor_recv_msg_until",
			"__tetra_actor_recv_begin",
			"__tetra_actor_recv_slot",
			"__tetra_actor_recv_count",
			"__tetra_actor_self",
			"__tetra_actor_sender",
			"__tetra_actor_yield_now",
		},
		ActorStateRequiredSymbols: []string{
			"__tetra_actor_state_load",
			"__tetra_actor_state_store",
		},
		TaskRequiredSymbols: []string{
			"__tetra_task_spawn_i32",
			"__tetra_task_join_i32",
			"__tetra_task_join_result_i32",
			"__tetra_task_join_until_i32",
			"__tetra_task_poll_i32",
			"__tetra_task_is_canceled",
			"__tetra_task_checkpoint",
		},
		TaskGroupRequiredSymbols: []string{
			"__tetra_task_group_open",
			"__tetra_task_group_close",
			"__tetra_task_group_cancel",
			"__tetra_task_group_current",
			"__tetra_task_group_status",
			"__tetra_task_spawn_group_i32",
		},
		TypedTaskRequiredSymbols: []string{
			"__tetra_task_result_begin",
			"__tetra_task_result_slot",
			"__tetra_task_result_get",
			"__tetra_task_join_typed_2",
			"__tetra_task_join_typed_3",
			"__tetra_task_join_typed_4",
			"__tetra_task_join_typed_5",
			"__tetra_task_join_typed_6",
			"__tetra_task_join_typed_7",
			"__tetra_task_join_typed_8",
		},
		TimeRequiredSymbols: []string{
			"__tetra_time_now_ms",
			"__tetra_sleep_ms",
			"__tetra_sleep_until_ms",
			"__tetra_deadline_ms",
			"__tetra_timer_ready_ms",
		},
		FilesystemRequiredSymbols: []string{"__tetra_fs_exists"},
		SurfaceRequiredSymbols: []string{
			"__tetra_surface_open",
			"__tetra_surface_close",
			"__tetra_surface_poll_event_kind",
			"__tetra_surface_poll_event_x",
			"__tetra_surface_poll_event_y",
			"__tetra_surface_poll_event_button",
			"__tetra_surface_poll_event_into",
			"__tetra_surface_poll_event_text_len",
			"__tetra_surface_poll_event_text_into",
			"__tetra_surface_begin_frame",
			"__tetra_surface_present_rgba",
			"__tetra_surface_now_ms",
			"__tetra_surface_request_redraw",
		},
		ActorsProgramGlueSymbols: []string{
			"__tetra_actor_dispatch",
			"__tetra_actor_main_entry_id",
		},
	}
}

func validRuntimeTargetsForTest() map[string]bool {
	return map[string]bool{
		"linux-x64":   true,
		"macos-x64":   true,
		"windows-x64": true,
	}
}

func runManifestValidator(t *testing.T, manifest string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--manifest", path)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
