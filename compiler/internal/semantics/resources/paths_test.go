package resources

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestPathHelpersComposeResourcePaths(t *testing.T) {
	if got := FieldPath("", "owner"); got != "owner" {
		t.Fatalf("FieldPath empty prefix = %q, want owner", got)
	}
	if got := FieldPath("owner", "field"); got != "owner.field" {
		t.Fatalf("FieldPath = %q, want owner.field", got)
	}
	if got := EnumPayloadPath("owner", 3, 2); got != "owner.$case3.payload2" {
		t.Fatalf("EnumPayloadPath = %q, want owner.$case3.payload2", got)
	}
	if got := JoinPath("owner", "field.leaf"); got != "owner.field.leaf" {
		t.Fatalf("JoinPath = %q, want owner.field.leaf", got)
	}
}

func TestLeafTailMatchesWholeHeadOrChildPath(t *testing.T) {
	tail, ok := LeafTail("$elem.child", "$elem")
	if !ok || tail != "child" {
		t.Fatalf("LeafTail child = (%q, %v), want (child, true)", tail, ok)
	}
	tail, ok = LeafTail("$elem", "$elem")
	if !ok || tail != "" {
		t.Fatalf("LeafTail exact = (%q, %v), want empty true", tail, ok)
	}
	if _, ok := LeafTail("other.child", "$elem"); ok {
		t.Fatalf("LeafTail matched unrelated path")
	}
}

func TestPathForExprBuildsFieldAccessPath(t *testing.T) {
	expr := &frontend.FieldAccessExpr{
		Field: "leaf",
		Base: &frontend.FieldAccessExpr{
			Field: "child",
			Base:  &frontend.IdentExpr{Name: "root"},
		},
	}

	got, ok := PathForExpr(expr)
	if !ok || got != "root.child.leaf" {
		t.Fatalf("PathForExpr = (%q, %v), want (root.child.leaf, true)", got, ok)
	}
}
