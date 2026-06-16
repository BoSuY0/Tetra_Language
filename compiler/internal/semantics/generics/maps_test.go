package generics

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestCloneStringMapCopiesEntries(t *testing.T) {
	in := map[string]string{"T": "i32"}
	got := CloneStringMap(in)
	got["T"] = "bool"
	if in["T"] != "i32" {
		t.Fatalf("CloneStringMap returned aliased map: %#v", in)
	}
}

func TestCloneFunctionTypeMapCopiesEntries(t *testing.T) {
	in := map[string]frontend.TypeRef{"f": {Name: "i32"}}
	got := CloneFunctionTypeMap(in)
	got["f"] = frontend.TypeRef{Name: "bool"}
	if in["f"].Name != "i32" {
		t.Fatalf("CloneFunctionTypeMap returned aliased map: %#v", in)
	}
}
