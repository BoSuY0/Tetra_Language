package semantics

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
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

func TestResolveTypeNameFunctionTypeRefMVP(t *testing.T) {
	ref := frontend.TypeRef{
		At:   frontend.Position{File: "fn_types.tetra", Line: 2, Col: 9},
		Kind: frontend.TypeRefFunction,
		Params: []frontend.TypeRef{
			{At: frontend.Position{File: "fn_types.tetra", Line: 2, Col: 12}, Kind: frontend.TypeRefNamed, Name: "Int"},
			{At: frontend.Position{File: "fn_types.tetra", Line: 2, Col: 17}, Kind: frontend.TypeRefNamed, Name: "Bool"},
		},
		Return: &frontend.TypeRef{At: frontend.Position{File: "fn_types.tetra", Line: 2, Col: 26}, Kind: frontend.TypeRefNamed, Name: "UInt8"},
	}
	got, err := resolveTypeName(&ref, "main", nil)
	if err != nil {
		t.Fatalf("resolveTypeName: %v", err)
	}
	if got != "fnptr" {
		t.Fatalf("resolved = %q, want fnptr", got)
	}
}

func TestResolveTypeNameFunctionTypeRefRequiresReturn(t *testing.T) {
	_, err := resolveTypeName(&frontend.TypeRef{
		At:     frontend.Position{File: "fn_types.tetra", Line: 4, Col: 5},
		Kind:   frontend.TypeRefFunction,
		Params: []frontend.TypeRef{{At: frontend.Position{File: "fn_types.tetra", Line: 4, Col: 8}, Kind: frontend.TypeRefNamed, Name: "Int"}},
	}, "main", nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "fn_types.tetra:4:5: missing function return type") {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckWorldAliasesImportedPublicFunctionTypedGlobals(t *testing.T) {
	lib, err := frontend.ParseFile([]byte(`module lib.math

pub val cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`), "lib/math.t4")
	if err != nil {
		t.Fatalf("parse lib: %v", err)
	}
	app, err := frontend.ParseFile([]byte(`module app.main
import lib.math as math

func main() -> Int:
    return 0
`), "app/main.t4")
	if err != nil {
		t.Fatalf("parse app: %v", err)
	}
	selective, err := frontend.ParseFile([]byte(`module app.selective
import lib.math.{cb}

func probe() -> Int:
    return 0
`), "app/selective.t4")
	if err != nil {
		t.Fatalf("parse selective app: %v", err)
	}
	world := &module.World{
		EntryModule: app.Module,
		Files:       []*frontend.FileAST{lib, app, selective},
		ByModule: map[string]*frontend.FileAST{
			lib.Module:       lib,
			app.Module:       app,
			selective.Module: selective,
		},
	}

	checked, err := CheckWorldOpt(world, CheckOptions{RequireMain: false})
	if err != nil {
		t.Fatalf("CheckWorldOpt: %v", err)
	}
	globals := checked.GlobalsByModule[app.Module]
	for _, name := range []string{"math.cb", "lib.math.cb"} {
		global, ok := globals[name]
		if !ok {
			t.Fatalf("missing imported function-typed global alias %q in %#v", name, globals)
		}
		if !global.FunctionTypeValue || global.FunctionValue != "lib.math.add2" || global.Mutable {
			t.Fatalf("alias %q = %#v, want immutable function-typed global backed by lib.math.add2", name, global)
		}
	}
	selectiveGlobal, ok := checked.GlobalsByModule[selective.Module]["cb"]
	if !ok {
		t.Fatalf("missing selective imported function-typed global alias cb in %#v", checked.GlobalsByModule[selective.Module])
	}
	if !selectiveGlobal.FunctionTypeValue || selectiveGlobal.FunctionValue != "lib.math.add2" || selectiveGlobal.Mutable {
		t.Fatalf("selective alias cb = %#v, want immutable function-typed global backed by lib.math.add2", selectiveGlobal)
	}
}

func TestCheckWorldAliasesImportedMutableFunctionTypedGlobalsAsBoundary(t *testing.T) {
	lib, err := frontend.ParseFile([]byte(`module lib.math

pub var cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`), "lib/math.t4")
	if err != nil {
		t.Fatalf("parse lib: %v", err)
	}
	app, err := frontend.ParseFile([]byte(`module app.main
import lib.math as math

func main() -> Int:
    return 0
`), "app/main.t4")
	if err != nil {
		t.Fatalf("parse app: %v", err)
	}
	selective, err := frontend.ParseFile([]byte(`module app.selective
import lib.math.{cb}

func probe() -> Int:
    return 0
`), "app/selective.t4")
	if err != nil {
		t.Fatalf("parse selective app: %v", err)
	}
	world := &module.World{
		EntryModule: app.Module,
		Files:       []*frontend.FileAST{lib, app, selective},
		ByModule: map[string]*frontend.FileAST{
			lib.Module:       lib,
			app.Module:       app,
			selective.Module: selective,
		},
	}

	checked, err := CheckWorldOpt(world, CheckOptions{RequireMain: false})
	if err != nil {
		t.Fatalf("CheckWorldOpt: %v", err)
	}
	globals := checked.GlobalsByModule[app.Module]
	for _, name := range []string{"math.cb", "lib.math.cb"} {
		global, ok := globals[name]
		if !ok {
			t.Fatalf("missing imported mutable function-typed global alias %q in %#v", name, globals)
		}
		if !global.FunctionTypeValue || !global.Mutable || global.FunctionValue != "" {
			t.Fatalf("alias %q = %#v, want mutable function-typed boundary alias without static function value", name, global)
		}
	}
	selectiveGlobal, ok := checked.GlobalsByModule[selective.Module]["cb"]
	if !ok {
		t.Fatalf("missing selective imported mutable function-typed global alias cb in %#v", checked.GlobalsByModule[selective.Module])
	}
	if !selectiveGlobal.FunctionTypeValue || !selectiveGlobal.Mutable || selectiveGlobal.FunctionValue != "" {
		t.Fatalf("selective alias cb = %#v, want mutable function-typed boundary alias without static function value", selectiveGlobal)
	}
}
