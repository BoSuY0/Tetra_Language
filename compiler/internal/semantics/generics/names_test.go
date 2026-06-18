package generics

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestMangleNameSanitizesGenericTypes(t *testing.T) {
	got := MangleName("id", []string{"T", "U"}, map[string]string{
		"T": "[]i32",
		"U": "core.Pair<i32,bool>",
	})

	want := "id__T__5b__5d_i32__U_core_2e_Pair_3c_i32_2c_bool_3e_"
	if got != want {
		t.Fatalf("MangleName = %q, want %q", got, want)
	}
}

func TestSanitizeAndUnsanitizeGenericTypeRoundTrip(t *testing.T) {
	want := "core.Pair<i32,bool>_x"
	sanitized := SanitizeType(want)
	got, err := UnsanitizeType(sanitized)
	if err != nil {
		t.Fatalf("UnsanitizeType returned error: %v", err)
	}
	if got != want {
		t.Fatalf("UnsanitizeType(%q) = %q, want %q", sanitized, got, want)
	}
}

func TestTypeNameFormatsFunctionTypeRefs(t *testing.T) {
	ret := frontend.TypeRef{Name: "bool"}
	throws := frontend.TypeRef{Name: "err"}
	ref := frontend.TypeRef{
		Kind: frontend.TypeRefFunction,
		Params: []frontend.TypeRef{
			{Name: "i32"},
			{Kind: frontend.TypeRefOptional, Elem: &frontend.TypeRef{Name: "str"}},
		},
		ParamOwnership: []string{"borrow", ""},
		Return:         &ret,
		Throws:         &throws,
		Uses:           []string{"io", "mem"},
	}

	got := TypeName(ref, nil)
	want := "fn(borrow i32,str?)->bool throws err uses io,mem"
	if got != want {
		t.Fatalf("TypeName = %q, want %q", got, want)
	}
}

func TestClosureBindingKeyUsesPrivatePrefix(t *testing.T) {
	if got := ClosureBindingKey("fn"); got != ClosureBindingPrefix+"fn" {
		t.Fatalf("ClosureBindingKey = %q", got)
	}
}
