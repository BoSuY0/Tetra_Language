package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func mustJSON(v interface{}) []byte {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(raw, '\n')
}

func joined(parts ...string) string {
	return strings.Join(parts, " ")
}

func textLines(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n") + "\n")
}

func productionSafetyFeatures() []byte {
	coreDocs := []string{"docs/spec/core/current_supported_surface.md"}
	ownershipDocs := []string{"docs/spec/runtime/ownership_v1.md"}
	effectsDocs := []string{"docs/spec/runtime/effects_capabilities_privacy_v1.md"}
	return mustJSON(featuresReport{
		Schema:  "tetra.features.v1",
		Version: "v0.4.0",
		Features: []featureEntry{
			{
				ID:     "safety.production-core",
				Name:   "Production safety core",
				Status: "current",
				Since:  "v0.4.0",
				Scope: joined(
					"production local safety model for ownership/lifetime/borrow/consume/inout",
					"checks, resource finalization, callable escape diagnostics,",
					"effects/capabilities/privacy/consent/budget policy, unsafe boundaries,",
					"actor/task transfer safety, and pointer/MMIO/memory capability gates",
				),
				Stability: joined(
					"release-gated current profile with explicit diagnostics for unsupported",
					"distributed, cryptographic, formal-proof, and runtime-wide guarantees",
				),
				Docs: append(append(coreDocs, ownershipDocs...), effectsDocs...),
			},
			{
				ID:     "safety.effects-mvp",
				Name:   "Effects",
				Status: "current",
				Since:  "v0.3.0",
				Scope:  "stable uses effect names and groups with transitive call propagation",
				Stability: joined(
					"missing uses declarations are diagnostics",
				),
				Docs: effectsDocs,
			},
			{
				ID:     "safety.capabilities-mvp",
				Name:   "Capabilities",
				Status: "current",
				Since:  "v0.3.0",
				Scope:  "cap.io and cap.mem opaque tokens are obtained only inside unsafe blocks",
				Stability: joined(
					"raw memory/MMIO operations require matching uses effects, unsafe boundary,",
					"and capability argument",
				),
				Docs: effectsDocs,
			},
			{
				ID:     "safety.privacy-consent-mvp",
				Name:   "Privacy",
				Status: "current",
				Since:  "v0.3.0",
				Scope: joined(
					"uses privacy requires privacy semantic clauses;",
					"secret.i32/SecretInt signatures require consent token",
				),
				Stability: "static auditing and call-shape enforcement",
				Docs:      effectsDocs,
			},
			{
				ID:        "safety.budget-mvp",
				Name:      "Budget",
				Status:    "current",
				Since:     "v0.3.0",
				Scope:     "budget requires uses budget and deterministic budget guard instructions",
				Stability: "static cross-edge guardrail",
				Docs:      effectsDocs,
			},
			{
				ID:     "language.ownership-markers-mvp",
				Name:   "Ownership",
				Status: "current",
				Since:  "v0.2.0",
				Scope: joined(
					"conservative borrow/inout/consume marker checks for local calls, aliasing,",
					"use-after-consume, and borrow escape diagnostics",
				),
				Stability: "current bounded surface",
				Docs:      ownershipDocs,
			},
			{
				ID:     "language.resource-lifetime-mvp",
				Name:   "Resources",
				Status: "current",
				Since:  "v0.2.0",
				Scope: joined(
					"conservative resource finalization checks for task handles, task groups,",
					"island handles, region-backed slices, and structs containing them",
				),
				Stability: "double-use and ambiguous provenance diagnostics",
				Docs:      ownershipDocs,
			},
			{
				ID:     "language.lifetime-ssa",
				Name:   "Lifetime SSA",
				Status: "current",
				Since:  "v0.4.0",
				Scope: joined(
					"production SSA-like local lifetime join analysis for ownership consume",
					"state, resource finalization state, branch/match/loop flow snapshots,",
					"and maybe-consumed diagnostics",
				),
				Stability: "current local/control-flow solver",
				Docs:      ownershipDocs,
			},
			{
				ID:     "actors.task-transfer-safety",
				Name:   "Actor/task transfer",
				Status: "current",
				Since:  "v0.2.0",
				Scope: joined(
					"conservative actor/task ownership transfer checks for worker entrypoints,",
					"handle transfer, and use-after-transfer diagnostics",
				),
				Stability: "current local transfer contract",
				Docs:      ownershipDocs,
			},
			{
				ID:     "language.full-first-class-callables",
				Name:   "Callables",
				Status: "current",
				Since:  "v0.4.0",
				Scope: joined(
					"production first-class callable/function-value semantics with stable",
					"diagnostics for mutable by-reference captures, pointer/resource",
					"captures, and thread-boundary callable escape",
				),
				Stability: "current safe capture model",
				Docs:      coreDocs,
			},
		},
	})
}

func productionSafetyEvidence() safetyEvidence {
	return safetyEvidence{
		Features: productionSafetyFeatures(),
		CurrentSurface: textLines(
			"Safety production core is current.",
			"Lifetime SSA local join solver is current since `v0.4.0` for branch,",
			"match, and loop flow snapshots.",
			"Mutable by-reference captures, including callable mutable-capture global-escape,",
			"pointer/resource captures, and thread-boundary callable escape keep",
			"stable JSON diagnostics.",
		),
		OwnershipSpec: textLines(
			"Ownership markers are part of the checked function-call contract in the",
			"current production surface.",
			"The local lifetime solver is SSA-like for branch, match, and loop joins.",
			"It rejects borrow escape diagnostics, use-after-transfer diagnostics,",
			"and worker effect boundary violations.",
			"Resource lifetime checks reject double-use and ambiguous provenance.",
		),
		EffectsSpec: textLines(
			"Canonical `uses` effect names are actors, alloc, budget, capability, io,",
			"mem, mmio, privacy, runtime.",
			"Unsafe Policy Public API Boundary exports unsafe_policy for builtins.",
			"Privacy And Consent requires consent.token for secret-bearing signatures.",
			"Budget exhaustion uses the stable local policy-failure ABI.",
			"Pointer/MMIO/memory operations require matching `uses` effects,",
			"unsafe boundaries, and capability tokens.",
		),
	}
}

func TestValidateSafetyReadinessAcceptsProductionEvidence(t *testing.T) {
	if err := validateSafetyReadiness(productionSafetyEvidence()); err != nil {
		t.Fatalf("validateSafetyReadiness failed: %v", err)
	}
}

func TestValidateSafetyReadinessAllowsValidatorRejectionLanguage(t *testing.T) {
	evidence := productionSafetyEvidence()
	evidence.CurrentSurface = textLines(
		"Safety production core is current.",
		"Lifetime SSA local join solver is current since `v0.4.0` for branch,",
		"match, and loop flow snapshots.",
		"Mutable by-reference captures, including callable mutable-capture global-escape,",
		"pointer/resource captures, and thread-boundary callable escape keep",
		"stable JSON diagnostics.",
		"The native runtime validator rejects metadata-only, web-only,",
		"native-shell sidecar-only, fake/mock/placeholder, missing event execution,",
		"and missing state-transition reports.",
	)

	if err := validateSafetyReadiness(evidence); err != nil {
		t.Fatalf("validateSafetyReadiness rejected negative validator wording: %v", err)
	}
}

func TestValidateSafetyReadinessRejectsMissingProductionCore(t *testing.T) {
	evidence := productionSafetyEvidence()
	evidence.Features = mustJSON(featuresReport{
		Schema:  "tetra.features.v1",
		Version: "v0.4.0",
		Features: []featureEntry{
			{
				ID:        "safety.effects-mvp",
				Name:      "Effects",
				Status:    "current",
				Since:     "v0.3.0",
				Scope:     "stable uses effect names and groups",
				Stability: "current",
				Docs:      []string{"docs/spec/runtime/effects_capabilities_privacy_v1.md"},
			},
			{
				ID:        "language.full-v1-guarantees",
				Name:      "Full v1",
				Status:    "planned",
				Scope:     "future",
				Stability: "planned",
				Docs:      []string{"docs/spec/flow/v1_scope.md"},
			},
			{
				ID:        "eco.distributed-network",
				Name:      "Eco",
				Status:    "post-v1",
				Scope:     "future",
				Stability: "post-v1",
				Docs:      []string{"docs/spec/core/current_supported_surface.md"},
			},
		},
	})

	err := validateSafetyReadiness(evidence)
	if err == nil {
		t.Fatalf("expected missing production core failure")
	}
	if !strings.Contains(err.Error(), "safety.production-core") {
		t.Fatalf("error = %v, want safety.production-core", err)
	}
}

func TestValidateSafetyReadinessRejectsStaleLifetimeClaim(t *testing.T) {
	evidence := productionSafetyEvidence()
	evidence.CurrentSurface = []byte(
		("Lifetime SSA solving is planned future work: the current " +
			"ownership/resource safety implementation is a conservative MVP."),
	)

	err := validateSafetyReadiness(evidence)
	if err == nil {
		t.Fatalf("expected stale lifetime claim failure")
	}
	if !strings.Contains(err.Error(), "Lifetime SSA solving is planned future work") {
		t.Fatalf("error = %v, want stale lifetime phrase", err)
	}
}

func TestValidateSafetyReadinessRejectsMockClaims(t *testing.T) {
	evidence := productionSafetyEvidence()
	evidence.OwnershipSpec = []byte("mock ownership safety claim")

	err := validateSafetyReadiness(evidence)
	if err == nil {
		t.Fatalf("expected mock claim failure")
	}
	if !strings.Contains(err.Error(), "mock") {
		t.Fatalf("error = %v, want mock", err)
	}
}
