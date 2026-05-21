package main

import (
	"strings"
	"testing"
)

func productionSafetyFeatures() []byte {
	return []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.4.0",
  "features": [
    {
      "id": "safety.production-core",
      "name": "Production safety core",
      "status": "current",
      "since": "v0.4.0",
      "scope": "production local safety model for ownership/lifetime/borrow/consume/inout checks, resource finalization, callable escape diagnostics, effects/capabilities/privacy/consent/budget policy, unsafe boundaries, actor/task transfer safety, and pointer/MMIO/memory capability gates",
      "stability": "release-gated current profile with explicit diagnostics for unsupported distributed, cryptographic, formal-proof, and runtime-wide guarantees",
      "docs": ["docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/effects_capabilities_privacy_v1.md"]
    },
    {"id":"safety.effects-mvp","name":"Effects","status":"current","since":"v0.3.0","scope":"stable uses effect names and groups with transitive call propagation","stability":"missing uses declarations are diagnostics","docs":["docs/spec/effects_capabilities_privacy_v1.md"]},
    {"id":"safety.capabilities-mvp","name":"Capabilities","status":"current","since":"v0.3.0","scope":"cap.io and cap.mem opaque tokens are obtained only inside unsafe blocks","stability":"raw memory/MMIO operations require matching uses effects, unsafe boundary, and capability argument","docs":["docs/spec/effects_capabilities_privacy_v1.md"]},
    {"id":"safety.privacy-consent-mvp","name":"Privacy","status":"current","since":"v0.3.0","scope":"uses privacy requires privacy semantic clauses; secret.i32/SecretInt signatures require consent token","stability":"static auditing and call-shape enforcement","docs":["docs/spec/effects_capabilities_privacy_v1.md"]},
    {"id":"safety.budget-mvp","name":"Budget","status":"current","since":"v0.3.0","scope":"budget requires uses budget and deterministic budget guard instructions","stability":"static cross-edge guardrail","docs":["docs/spec/effects_capabilities_privacy_v1.md"]},
    {"id":"language.ownership-markers-mvp","name":"Ownership","status":"current","since":"v0.2.0","scope":"conservative borrow/inout/consume marker checks for local calls, aliasing, use-after-consume, and borrow escape diagnostics","stability":"current bounded surface","docs":["docs/spec/ownership_v1.md"]},
    {"id":"language.resource-lifetime-mvp","name":"Resources","status":"current","since":"v0.2.0","scope":"conservative resource finalization checks for task handles, task groups, island handles, region-backed slices, and structs containing them","stability":"double-use and ambiguous provenance diagnostics","docs":["docs/spec/ownership_v1.md"]},
    {"id":"language.lifetime-ssa","name":"Lifetime SSA","status":"current","since":"v0.4.0","scope":"production SSA-like local lifetime join analysis for ownership consume state, resource finalization state, branch/match/loop flow snapshots, and maybe-consumed diagnostics","stability":"current local/control-flow solver","docs":["docs/spec/ownership_v1.md"]},
    {"id":"actors.task-transfer-safety","name":"Actor/task transfer","status":"current","since":"v0.2.0","scope":"conservative actor/task ownership transfer checks for worker entrypoints, handle transfer, and use-after-transfer diagnostics","stability":"current local transfer contract","docs":["docs/spec/ownership_v1.md"]},
    {"id":"language.full-first-class-callables","name":"Callables","status":"current","since":"v0.4.0","scope":"production first-class callable/function-value semantics with stable diagnostics for mutable by-reference captures, pointer/resource captures, and thread-boundary callable escape","stability":"current safe capture model","docs":["docs/spec/current_supported_surface.md"]}
  ]
}`)
}

func productionSafetyEvidence() safetyEvidence {
	return safetyEvidence{
		Features: productionSafetyFeatures(),
		CurrentSurface: []byte(`Safety production core is current.
Lifetime SSA local join solver is current since ` + "`v0.4.0`" + ` for branch, match, and loop flow snapshots.
Mutable by-reference captures, including callable mutable-capture global-escape, pointer/resource captures, and thread-boundary callable escape keep stable JSON diagnostics.
`),
		OwnershipSpec: []byte(`Ownership markers are part of the checked function-call contract in the current production surface.
The local lifetime solver is SSA-like for branch, match, and loop joins.
It rejects borrow escape diagnostics, use-after-transfer diagnostics, and worker effect boundary violations.
Resource lifetime checks reject double-use and ambiguous provenance.
`),
		EffectsSpec: []byte(`Canonical ` + "`uses`" + ` effect names are actors, alloc, budget, capability, io, mem, mmio, privacy, runtime.
Unsafe Policy Public API Boundary exports unsafe_policy for builtins.
Privacy And Consent requires consent.token for secret-bearing signatures.
Budget exhaustion uses the stable local policy-failure ABI.
Pointer/MMIO/memory operations require matching ` + "`uses`" + ` effects, unsafe boundaries, and capability tokens.
`),
	}
}

func TestValidateSafetyReadinessAcceptsProductionEvidence(t *testing.T) {
	if err := validateSafetyReadiness(productionSafetyEvidence()); err != nil {
		t.Fatalf("validateSafetyReadiness failed: %v", err)
	}
}

func TestValidateSafetyReadinessAllowsValidatorRejectionLanguage(t *testing.T) {
	evidence := productionSafetyEvidence()
	evidence.CurrentSurface = []byte(`Safety production core is current.
Lifetime SSA local join solver is current since ` + "`v0.4.0`" + ` for branch, match, and loop flow snapshots.
Mutable by-reference captures, including callable mutable-capture global-escape, pointer/resource captures, and thread-boundary callable escape keep stable JSON diagnostics.
The native runtime validator rejects metadata-only, web-only, native-shell sidecar-only,
fake/mock/placeholder, missing event execution, and missing state-transition reports.
`)

	if err := validateSafetyReadiness(evidence); err != nil {
		t.Fatalf("validateSafetyReadiness rejected negative validator wording: %v", err)
	}
}

func TestValidateSafetyReadinessRejectsMissingProductionCore(t *testing.T) {
	evidence := productionSafetyEvidence()
	evidence.Features = []byte(`{"schema":"tetra.features.v1","version":"v0.4.0","features":[
    {"id":"safety.effects-mvp","name":"Effects","status":"current","since":"v0.3.0","scope":"stable uses effect names and groups","stability":"current","docs":["docs/spec/effects_capabilities_privacy_v1.md"]},
    {"id":"language.full-v1-guarantees","name":"Full v1","status":"planned","scope":"future","stability":"planned","docs":["docs/spec/v1_scope.md"]},
    {"id":"eco.distributed-network","name":"Eco","status":"post-v1","scope":"future","stability":"post-v1","docs":["docs/spec/current_supported_surface.md"]}
  ]}`)

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
	evidence.CurrentSurface = []byte("Lifetime SSA solving is planned future work: the current ownership/resource safety implementation is a conservative MVP.")

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
