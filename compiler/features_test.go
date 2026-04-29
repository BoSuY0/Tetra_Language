package compiler

import (
	"strings"
	"testing"
)

func TestFeatureRegistryCoversReleaseStatusesAndKeyBoundaries(t *testing.T) {
	features := FeatureRegistry()
	if len(features) == 0 {
		t.Fatal("FeatureRegistry returned no entries")
	}
	seenStatus := map[FeatureStatus]bool{}
	seenID := map[string]FeatureStatus{}
	seenFeature := map[string]FeatureInfo{}
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
		if feature.Status == FeatureStatusCurrent && feature.Since == "" {
			t.Fatalf("current feature %s missing since", feature.ID)
		}
		if len(feature.Docs) == 0 {
			t.Fatalf("feature %s missing docs", feature.ID)
		}
	}
	for _, status := range []FeatureStatus{FeatureStatusCurrent, FeatureStatusExperimental, FeatureStatusPlanned, FeatureStatusPostV1} {
		if !seenStatus[status] {
			t.Fatalf("feature registry missing status %s", status)
		}
	}
	for id, wantStatus := range map[string]FeatureStatus{
		"cli.core":                            FeatureStatusCurrent,
		"targets.wasm-build-only":             FeatureStatusCurrent,
		"language.generics-mvp":               FeatureStatusCurrent,
		"language.protocol-conformance-mvp":   FeatureStatusCurrent,
		"language.callable-mvp":               FeatureStatusCurrent,
		"language.callable-level1":            FeatureStatusExperimental,
		"stdlib.experimental-mirrors":         FeatureStatusExperimental,
		"language.enum-payload-match":         FeatureStatusExperimental,
		"language.ownership-markers-mvp":      FeatureStatusCurrent,
		"language.resource-lifetime-mvp":      FeatureStatusCurrent,
		"actors.task-transfer-safety":         FeatureStatusCurrent,
		"language.lifetime-ssa":               FeatureStatusPlanned,
		"language.callable-level2":            FeatureStatusPlanned,
		"wasm.runtime-execution":              FeatureStatusPlanned,
		"eco.distributed-network":             FeatureStatusPostV1,
		"language.full-first-class-callables": FeatureStatusPostV1,
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
	for _, want := range []string{"Level 0 callable surface", "full first-class function values remain out of scope"} {
		if !strings.Contains(callableMVP.Scope+" "+callableMVP.Stability, want) {
			t.Fatalf("callable MVP feature missing %q boundary: %#v", want, callableMVP)
		}
	}
	callableLevel1 := seenFeature["language.callable-level1"]
	if callableLevel1.Since != "" {
		t.Fatalf("callable Level 1 should not claim v0.2.0 since marker: %#v", callableLevel1)
	}
	for _, want := range []string{"experimental", "not part of the v0.2.0 stable baseline", "not a full first-class function-value claim"} {
		if !strings.Contains(callableLevel1.Scope+" "+callableLevel1.Stability, want) {
			t.Fatalf("callable Level 1 feature missing %q boundary: %#v", want, callableLevel1)
		}
	}
	ownershipMVP := seenFeature["language.ownership-markers-mvp"]
	for _, want := range []string{"conservative borrow/inout/consume marker checks", "use-after-consume", "borrow escape diagnostics", "not a full SSA lifetime solver"} {
		if !strings.Contains(ownershipMVP.Scope+" "+ownershipMVP.Stability, want) {
			t.Fatalf("ownership markers MVP feature missing %q boundary: %#v", want, ownershipMVP)
		}
	}
	resourceMVP := seenFeature["language.resource-lifetime-mvp"]
	for _, want := range []string{"conservative resource finalization checks", "task handles", "island handles", "double-use", "ambiguous provenance", "not a full SSA lifetime solver"} {
		if !strings.Contains(resourceMVP.Scope+" "+resourceMVP.Stability, want) {
			t.Fatalf("resource lifetime MVP feature missing %q boundary: %#v", want, resourceMVP)
		}
	}
	transferMVP := seenFeature["actors.task-transfer-safety"]
	for _, want := range []string{"conservative actor/task ownership transfer checks", "worker entrypoints", "use-after-transfer diagnostics", "conservative local MVP", "distributed actors"} {
		if !strings.Contains(transferMVP.Scope+" "+transferMVP.Stability, want) {
			t.Fatalf("actor/task transfer feature missing %q boundary: %#v", want, transferMVP)
		}
	}
	lifetimeSSA := seenFeature["language.lifetime-ssa"]
	if lifetimeSSA.Since != "" {
		t.Fatalf("lifetime SSA should not claim v0.2.0 since marker: %#v", lifetimeSSA)
	}
	for _, want := range []string{"planned full SSA lifetime analysis", "precise merge reasoning", "no current v0.2.0 support guarantee", "conservative ownership/resource MVP"} {
		if !strings.Contains(lifetimeSSA.Scope+" "+lifetimeSSA.Stability, want) {
			t.Fatalf("lifetime SSA feature missing %q boundary: %#v", want, lifetimeSSA)
		}
	}
	callableLevel2 := seenFeature["language.callable-level2"]
	if callableLevel2.Since != "" {
		t.Fatalf("callable Level 2 should not claim v0.2.0 since marker: %#v", callableLevel2)
	}
	for _, want := range []string{"planned/experimental", "captured closures", "no current v0.2.0 support guarantee", "no full first-class callable semantics"} {
		if !strings.Contains(callableLevel2.Scope+" "+callableLevel2.Stability, want) {
			t.Fatalf("callable Level 2 feature missing %q boundary: %#v", want, callableLevel2)
		}
	}
	enumFeature := seenFeature["language.enum-payload-match"]
	if enumFeature.Since != "" {
		t.Fatalf("enum payload feature should not claim v0.2.0 since marker: %#v", enumFeature)
	}
	for _, want := range []string{"positional enum payload constructors", "exhaustive enum match/catch", "not part of the current v0.2.0 stable baseline"} {
		if !strings.Contains(enumFeature.Scope+" "+enumFeature.Stability, want) {
			t.Fatalf("enum payload feature missing %q boundary: %#v", want, enumFeature)
		}
	}
}

func TestFeatureRegistryReturnsDefensiveCopy(t *testing.T) {
	features := FeatureRegistry()
	features[0].ID = "mutated"
	features[0].Docs[0] = "mutated.md"
	fresh := FeatureRegistry()
	if fresh[0].ID == "mutated" || fresh[0].Docs[0] == "mutated.md" {
		t.Fatalf("FeatureRegistry did not return a defensive copy: %#v", fresh[0])
	}
}
