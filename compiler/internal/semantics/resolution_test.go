package semantics

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestResolveTypeNameUnsupportedPathsArePositioned(t *testing.T) {
	pos := frontend.Position{File: "bad_types.tetra", Line: 3, Col: 7}
	tests := []struct {
		name string
		ref  frontend.TypeRef
		want string
	}{
		{
			name: "unsupported kind",
			ref:  frontend.TypeRef{At: pos, Kind: frontend.TypeRefKind(99)},
			want: "bad_types.tetra:3:7: unsupported type reference kind 99",
		},
		{
			name: "missing slice element",
			ref:  frontend.TypeRef{At: pos, Kind: frontend.TypeRefSlice},
			want: "bad_types.tetra:3:7: missing slice element type",
		},
		{
			name: "missing array element",
			ref:  frontend.TypeRef{At: pos, Kind: frontend.TypeRefArray},
			want: "bad_types.tetra:3:7: missing array element type",
		},
		{
			name: "missing optional payload",
			ref:  frontend.TypeRef{At: pos, Kind: frontend.TypeRefOptional},
			want: "bad_types.tetra:3:7: missing optional payload type",
		},
		{
			name: "missing named type",
			ref:  frontend.TypeRef{At: pos, Kind: frontend.TypeRefNamed},
			want: "bad_types.tetra:3:7: missing type name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolveTypeName(&tt.ref, "main", nil)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestValidateGenericTypeRefUnsupportedKindIsActionable(t *testing.T) {
	err := validateGenericTypeRef(frontend.TypeRef{
		At:   frontend.Position{File: "generic.tetra", Line: 9, Col: 11},
		Kind: frontend.TypeRefKind(77),
	}, map[string]struct{}{"T": {}})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "generic.tetra:9:11: unsupported generic type reference kind 77") {
		t.Fatalf("error = %v", err)
	}
}
