package rangeproof

import (
	"reflect"
	"testing"

	"tetra_language/compiler/internal/frontend"
	corerangeproof "tetra_language/compiler/internal/rangeproof"
)

func TestStaticRangeFromConditionLessThanLen(t *testing.T) {
	cond := &frontend.BinaryExpr{
		Left: &frontend.IdentExpr{Name: "i"},
		Op:   frontend.TokenLess,
		Right: &frontend.FieldAccessExpr{
			Base:  &frontend.IdentExpr{Name: "items"},
			Field: "len",
		},
	}

	index, base, proofRange, ok := StaticRangeFromCondition(cond)
	if !ok {
		t.Fatalf("expected static range condition")
	}
	if index != "i" || base != "items" {
		t.Fatalf("condition = (%q, %q), want (i, items)", index, base)
	}
	want := corerangeproof.LessThanLen("i", "items")
	if !reflect.DeepEqual(proofRange, want) {
		t.Fatalf("range = %#v, want %#v", proofRange, want)
	}
}

func TestStaticRangeFromConditionLessEqualLenMinusOne(t *testing.T) {
	cond := &frontend.BinaryExpr{
		Left: &frontend.IdentExpr{Name: "i"},
		Op:   frontend.TokenLessEq,
		Right: &frontend.BinaryExpr{
			Left: &frontend.FieldAccessExpr{
				Base:  &frontend.IdentExpr{Name: "items"},
				Field: "len",
			},
			Op:    frontend.TokenMinus,
			Right: &frontend.NumberExpr{Value: 1},
		},
	}

	index, base, proofRange, ok := StaticRangeFromCondition(cond)
	if !ok {
		t.Fatalf("expected static range condition")
	}
	if index != "i" || base != "items" {
		t.Fatalf("condition = (%q, %q), want (i, items)", index, base)
	}
	want := corerangeproof.LessEqualLenMinusOne("i", "items")
	if !reflect.DeepEqual(proofRange, want) {
		t.Fatalf("range = %#v, want %#v", proofRange, want)
	}
}

func TestBranchRangeConditionAcceptsNonNegativeGuardInEitherOrder(t *testing.T) {
	lowerGuard := &frontend.BinaryExpr{
		Left:  &frontend.IdentExpr{Name: "i"},
		Op:    frontend.TokenGreaterEq,
		Right: &frontend.NumberExpr{Value: 0},
	}
	upperGuard := &frontend.BinaryExpr{
		Left: &frontend.IdentExpr{Name: "i"},
		Op:   frontend.TokenLess,
		Right: &frontend.FieldAccessExpr{
			Base:  &frontend.IdentExpr{Name: "items"},
			Field: "len",
		},
	}
	for _, cond := range []frontend.Expr{
		&frontend.BinaryExpr{Left: lowerGuard, Op: frontend.TokenAmpAmp, Right: upperGuard},
		&frontend.BinaryExpr{Left: upperGuard, Op: frontend.TokenAmpAmp, Right: lowerGuard},
	} {
		index, base, ok := BranchRangeCondition(cond)
		if !ok {
			t.Fatalf("expected branch range condition for %#v", cond)
		}
		if index != "i" || base != "items" {
			t.Fatalf("condition = (%q, %q), want (i, items)", index, base)
		}
	}
}

func TestPathsAndProofIDs(t *testing.T) {
	path := SimpleExprPath(&frontend.FieldAccessExpr{
		Base: &frontend.FieldAccessExpr{
			Base:  &frontend.IdentExpr{Name: "state"},
			Field: "items",
		},
		Field: "len",
	})
	if path != "state.items.len" {
		t.Fatalf("path = %q", path)
	}

	if !PathMatchesMutation("state.items.len", "state.items") {
		t.Fatalf("expected mutation to invalidate nested proof path")
	}
	if PathMatchesMutation("state.items", "state.items.len") {
		t.Fatalf("child mutation should not invalidate parent proof path")
	}

	id := WhileBoundsProofID("i", "state.items", frontend.Position{Line: 7, Col: 3})
	if id != "proof:while:i:state_items:7:3" {
		t.Fatalf("while proof id = %q", id)
	}
	copyID := CopyLoopBoundsProofID("core.slice copy", frontend.Position{Line: 8, Col: 4})
	if copyID != "proof:copy-loop:core_slice_copy:8:4" {
		t.Fatalf("copy proof id = %q", copyID)
	}
}

func TestBuiltinClassifiers(t *testing.T) {
	if !IsRawSliceConstructor(&frontend.CallExpr{Name: "core.raw_slice_i32_from_parts"}) {
		t.Fatalf("expected raw slice constructor")
	}
	if RawSliceElementShift("core.raw_slice_i32_from_parts") != 2 {
		t.Fatalf("unexpected i32 raw slice shift")
	}
	if !IsSliceCopyBuiltinName("core.slice_copy_i32") || IsSliceCopyBuiltinName("core.slice_window_i32") {
		t.Fatalf("slice copy classifier mismatch")
	}
	if !IsBorrowOrViewBuiltinName("core.slice_window_i32") || !IsBorrowOrViewBuiltinName("core.string_prefix") {
		t.Fatalf("borrow/view classifier mismatch")
	}
	forID := ForCollectionBoundsProofID(&frontend.ForRangeStmt{
		Name: "item",
		At:   frontend.Position{Line: 9, Col: 5},
		Iterable: &frontend.CallExpr{
			Name: "core.string_suffix",
		},
	})
	if forID != "proof:for-collection-view:item:9:5" {
		t.Fatalf("for collection proof id = %q", forID)
	}
}
