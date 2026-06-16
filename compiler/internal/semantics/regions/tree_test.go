package regions

import "testing"

func TestCopyRegionVarsCopiesMap(t *testing.T) {
	in := map[string]int{"x": 1}
	got := CopyVars(in)
	got["x"] = 2
	if in["x"] != 1 {
		t.Fatalf("CopyVars returned aliased map: %#v", in)
	}
}

func TestMergeRegionVarsMarksConflictsUnknown(t *testing.T) {
	got := MergeVars(map[string]int{"x": 1, "same": 2}, map[string]int{"x": 2, "same": 2, "right": 3})
	if got["x"] != Unknown {
		t.Fatalf("conflicting x = %d, want Unknown", got["x"])
	}
	if got["same"] != 2 {
		t.Fatalf("same = %d, want 2", got["same"])
	}
	if got["right"] != Unknown {
		t.Fatalf("right = %d, want Unknown", got["right"])
	}
}

func TestJoinAndCommonRegionFromTree(t *testing.T) {
	if got := Join(None, 7); got != 7 {
		t.Fatalf("Join(None, 7) = %d, want 7", got)
	}
	if got := Join(7, 8); got != Unknown {
		t.Fatalf("Join(7, 8) = %d, want Unknown", got)
	}
	if got := CommonFromTree(map[string]int{"a": 7, "b": 7}); got != 7 {
		t.Fatalf("CommonFromTree same = %d, want 7", got)
	}
	if got := ConstructorFromTree(map[string]int{"a": 7, "b": 8}); got != None {
		t.Fatalf("ConstructorFromTree conflict = %d, want None", got)
	}
}
