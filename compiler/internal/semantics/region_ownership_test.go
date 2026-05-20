package semantics

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestCheckNoConsumedDescendantsCanonicalizesAliasPaths(t *testing.T) {
	t.Run("query_path_is_alias_to_consumed_parent", func(t *testing.T) {
		state := newRegionState(nil)
		state.bindOwnershipAlias("raw", "msg")
		state.markConsumed("msg", frontend.Position{})

		err := state.checkNoConsumedDescendants("raw", frontend.Position{})
		if err == nil {
			t.Fatalf("expected consumed value error, got nil")
		}
		if !strings.Contains(err.Error(), "'msg'") {
			t.Fatalf("error = %q, want canonical path 'msg'", err.Error())
		}
		if strings.Contains(err.Error(), "'raw'") {
			t.Fatalf("error = %q, should not use alias name 'raw'", err.Error())
		}
	})

	t.Run("query_nested_alias_resolves_to_canonical_descendant", func(t *testing.T) {
		state := newRegionState(nil)
		state.bindOwnershipAlias("raw", "msg")
		state.markConsumed("msg.$case0.payload0", frontend.Position{})

		err := state.checkNoConsumedDescendants("raw.$case0.payload0", frontend.Position{})
		if err == nil {
			t.Fatalf("expected consumed value error, got nil")
		}
		if !strings.Contains(err.Error(), "msg.$case0.payload0") {
			t.Fatalf("error = %q, want canonical path 'msg.$case0.payload0'", err.Error())
		}
		if strings.Contains(err.Error(), "raw.$case0.payload0") {
			t.Fatalf("error = %q, should not use alias path 'raw.$case0.payload0'", err.Error())
		}
	})
}

func TestCheckNotConsumedCanonicalizesAliasPaths(t *testing.T) {
	state := newRegionState(nil)
	state.bindOwnershipAlias("raw", "msg")
	state.markConsumed("msg", frontend.Position{
		File: "app/main.t4",
		Line: 12,
		Col:  3,
	})

	err := state.checkNotConsumed("raw", frontend.Position{})
	if err == nil {
		t.Fatalf("expected consumed value error, got nil")
	}
	if !strings.Contains(err.Error(), "'msg'") {
		t.Fatalf("error = %q, want canonical path 'msg'", err.Error())
	}
	if strings.Contains(err.Error(), "'raw'") {
		t.Fatalf("error = %q, should not use alias name 'raw'", err.Error())
	}
}

func TestCheckNotConsumedCanonicalizesAliasPathsWhenConsumedBeforeAlias(t *testing.T) {
	state := newRegionState(nil)
	state.markConsumed("raw", frontend.Position{
		File: "app/main.t4",
		Line: 7,
		Col:  2,
	})
	state.bindOwnershipAlias("raw", "msg")

	err := state.checkNotConsumed("raw", frontend.Position{})
	if err == nil {
		t.Fatalf("expected consumed value error, got nil")
	}
	if !strings.Contains(err.Error(), "msg") {
		t.Fatalf("error = %q, want canonical path 'msg'", err.Error())
	}
	if strings.Contains(err.Error(), "raw") {
		t.Fatalf("error = %q, should not use alias name 'raw'", err.Error())
	}
}

func TestCheckNotConsumedNestedAliasCanonicalizesPath(t *testing.T) {
	state := newRegionState(nil)
	state.bindOwnershipAlias("raw", "msg")
	state.markConsumed("msg.$case0.payload0", frontend.Position{})

	err := state.checkNotConsumed("raw.$case0.payload0", frontend.Position{})
	if err == nil {
		t.Fatalf("expected consumed value error, got nil")
	}
	if !strings.Contains(err.Error(), "msg.$case0.payload0") {
		t.Fatalf("error = %q, want canonical path 'msg.$case0.payload0'", err.Error())
	}
	if strings.Contains(err.Error(), "raw.$case0.payload0") {
		t.Fatalf("error = %q, should not use alias path 'raw.$case0.payload0'", err.Error())
	}
}

func TestClearConsumedTreeClearsAliasEquivalentPaths(t *testing.T) {
	state := newRegionState(nil)
	state.bindOwnershipAlias("raw", "msg")
	state.markConsumed("msg.$case0.payload0", frontend.Position{})
	state.markConsumed("msg", frontend.Position{})

	state.clearConsumedTree("raw.$case0.payload0")
	if _, ok := state.consumedVars["msg.$case0.payload0"]; ok {
		t.Fatalf("expected 'msg.$case0.payload0' to be cleared")
	}
	if _, ok := state.consumedVars["raw.$case0.payload0"]; ok {
		t.Fatalf("expected 'raw.$case0.payload0' to be cleared")
	}
	if _, ok := state.consumedVars["msg"]; !ok {
		t.Fatalf("expected 'msg' to remain when clearing descendant only? got %v", state.consumedVars["msg"])
	}
}

func TestClearConsumedTreeClearsAliasForConsumedBasePath(t *testing.T) {
	state := newRegionState(nil)
	state.bindOwnershipAlias("raw", "msg")
	state.markConsumed("raw", frontend.Position{})

	state.clearConsumedTree("msg")
	if _, ok := state.consumedVars["raw"]; ok {
		t.Fatalf("expected alias key 'raw' to be cleared")
	}
	if _, ok := state.consumedVars["msg"]; ok {
		t.Fatalf("expected canonical key 'msg' to be cleared")
	}
}

func TestMergeOwnershipAliasesIntersectsOnlyMatchingMappings(t *testing.T) {
	left := map[string]string{
		"common":   "root",
		"leftOnly": "msg",
	}
	right := map[string]string{
		"common":    "root",
		"rightOnly": "other",
	}
	merged := mergeOwnershipAliases(left, right)
	if got := len(merged); got != 1 {
		t.Fatalf("expected exactly one merged alias, got %d (%v)", got, merged)
	}
	if _, ok := merged["common"]; !ok {
		t.Fatalf("expected 'common' to remain after merge, got %v", merged)
	}
	if merged["common"] != "root" {
		t.Fatalf("expected 'common' to map to 'root', got %q", merged["common"])
	}
	if _, ok := merged["leftOnly"]; ok {
		t.Fatalf("did not expect 'leftOnly' in merged aliases: %v", merged)
	}
	if _, ok := merged["rightOnly"]; ok {
		t.Fatalf("did not expect 'rightOnly' in merged aliases: %v", merged)
	}
}

func TestMergeFlowWithLabelsIntersectsOwnershipAliases(t *testing.T) {
	state := newRegionState(nil)
	left := flowSnapshot{
		reachable: true,
		consumedVars: map[string]frontend.Position{
			"raw": {},
		},
		maybeConsumedVars: map[string]ownershipJoinConflict{},
		ownershipAliases: map[string]string{
			"common":   "root",
			"leftOnly": "msg",
		},
		borrowedPtrAliases: map[string]string{},
		consumedResources:  map[int]frontend.Position{},
		resourceVars:       map[string]int{},
		unknownResources:   map[int]bool{},
		finalizedResources: map[int]resourceFinalization{},
	}
	right := flowSnapshot{
		reachable: true,
		consumedVars: map[string]frontend.Position{
			"raw": {},
		},
		maybeConsumedVars: map[string]ownershipJoinConflict{},
		ownershipAliases: map[string]string{
			"common":    "root",
			"rightOnly": "other",
		},
		borrowedPtrAliases: map[string]string{},
		consumedResources:  map[int]frontend.Position{},
		resourceVars:       map[string]int{},
		unknownResources:   map[int]bool{},
		finalizedResources: map[int]resourceFinalization{},
	}

	mergeFlowWithLabels(state, left, right, "left", "right")
	if len(state.ownershipAliases) != 1 {
		t.Fatalf("expected exactly one ownership alias after merge, got %d (%v)", len(state.ownershipAliases), state.ownershipAliases)
	}
	if value, ok := state.ownershipAliases["common"]; !ok || value != "root" {
		t.Fatalf("expected alias common->root, got %v (ok=%v)", state.ownershipAliases["common"], ok)
	}
	if _, ok := state.ownershipAliases["leftOnly"]; ok {
		t.Fatalf("did not expect leftOnly alias after merge: %v", state.ownershipAliases)
	}
	if _, ok := state.ownershipAliases["rightOnly"]; ok {
		t.Fatalf("did not expect rightOnly alias after merge: %v", state.ownershipAliases)
	}
}
