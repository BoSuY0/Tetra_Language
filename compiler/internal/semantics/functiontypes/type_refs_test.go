package functiontypes

import (
	"reflect"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestFunctionTypeRefOwnershipCopiesParamOwnership(t *testing.T) {
	ref := frontend.TypeRef{
		Kind:            frontend.TypeRefFunction,
		Params:          []frontend.TypeRef{{Name: "i32"}, {Name: "ptr"}},
		ParamOwnership:  []string{"borrow", "consume"},
		ReturnOwnership: "borrow",
	}

	got := ParamOwnership(ref)
	if !reflect.DeepEqual(got, []string{"borrow", "consume"}) {
		t.Fatalf("ParamOwnership = %#v", got)
	}
	got[0] = "mutated"
	if ref.ParamOwnership[0] != "borrow" {
		t.Fatalf("ParamOwnership returned aliased slice")
	}
	if got := ReturnOwnership(ref); got != "borrow" {
		t.Fatalf("ReturnOwnership = %q, want borrow", got)
	}
}

func TestFunctionTypeRefEffectsNormalizeUses(t *testing.T) {
	ref := frontend.TypeRef{
		Kind: frontend.TypeRefFunction,
		Uses: []string{"effects.cap.mem", "privacy"},
	}

	got, err := Effects(ref, frontend.Position{}, nil)
	if err != nil {
		t.Fatalf("Effects returned error: %v", err)
	}
	want := []string{"capability", "mem", "privacy"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Effects = %#v, want %#v", got, want)
	}
}

func TestNonFunctionTypeRefReturnsEmptyMetadata(t *testing.T) {
	ref := frontend.TypeRef{Kind: frontend.TypeRefNamed, Name: "i32"}

	if got := ParamOwnership(ref); got != nil {
		t.Fatalf("ParamOwnership = %#v, want nil", got)
	}
	if got := ReturnOwnership(ref); got != "" {
		t.Fatalf("ReturnOwnership = %q, want empty", got)
	}
	if got, err := Effects(ref, frontend.Position{}, nil); err != nil || got != nil {
		t.Fatalf("Effects = %#v, %v; want nil, nil", got, err)
	}
}
