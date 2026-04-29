package compiler

import "testing"

func TestFeatureRegistryCoversReleaseStatusesAndKeyBoundaries(t *testing.T) {
	features := FeatureRegistry()
	if len(features) == 0 {
		t.Fatal("FeatureRegistry returned no entries")
	}
	seenStatus := map[FeatureStatus]bool{}
	seenID := map[string]FeatureStatus{}
	for _, feature := range features {
		if feature.ID == "" || feature.Name == "" || feature.Scope == "" || feature.Stability == "" {
			t.Fatalf("feature has missing required metadata: %#v", feature)
		}
		if _, exists := seenID[feature.ID]; exists {
			t.Fatalf("duplicate feature ID %s", feature.ID)
		}
		seenID[feature.ID] = feature.Status
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
		"stdlib.experimental-mirrors":         FeatureStatusExperimental,
		"wasm.runtime-execution":              FeatureStatusPlanned,
		"eco.distributed-network":             FeatureStatusPostV1,
		"language.full-first-class-callables": FeatureStatusPostV1,
	} {
		if gotStatus := seenID[id]; gotStatus != wantStatus {
			t.Fatalf("feature %s status = %q, want %q", id, gotStatus, wantStatus)
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
