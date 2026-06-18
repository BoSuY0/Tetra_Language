package expressions

import (
	"reflect"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestCallbackArgumentNameFormatsIdentifiersFieldsAndCalls(t *testing.T) {
	field := &frontend.FieldAccessExpr{
		Field: "method",
		Base:  &frontend.IdentExpr{Name: "handler"},
	}
	if got := CallbackArgumentName(field); got != "handler.method" {
		t.Fatalf("CallbackArgumentName field = %q, want handler.method", got)
	}
	call := &frontend.CallExpr{Name: "factory"}
	if got := CallbackArgumentName(call); got != "factory()" {
		t.Fatalf("CallbackArgumentName call = %q, want factory()", got)
	}
}

func TestSplitFieldPathReturnsBaseFieldsAndLastPosition(t *testing.T) {
	expr := &frontend.FieldAccessExpr{
		At:    frontend.Position{Line: 3, Col: 4},
		Field: "leaf",
		Base: &frontend.FieldAccessExpr{
			At:    frontend.Position{Line: 2, Col: 4},
			Field: "child",
			Base:  &frontend.IdentExpr{At: frontend.Position{Line: 1, Col: 1}, Name: "root"},
		},
	}

	base, fields, pos, ok := SplitFieldPath(expr)
	if !ok {
		t.Fatalf("SplitFieldPath returned ok=false")
	}
	if base != "root" {
		t.Fatalf("base = %q, want root", base)
	}
	if !reflect.DeepEqual(fields, []string{"child", "leaf"}) {
		t.Fatalf("fields = %#v", fields)
	}
	if pos.Line != 3 || pos.Col != 4 {
		t.Fatalf("pos = %#v, want line 3 col 4", pos)
	}
}
