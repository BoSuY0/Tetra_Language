package semantics

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestResolutionModuleImportAliasResolvesCallAndType(t *testing.T) {
	pos := frontend.Position{File: "app/main.tetra", Line: 2, Col: 5}
	imports := map[string]string{"math": "engine.math"}

	callName, err := resolveCallName("math.add_one", "app.main", imports, pos)
	if err != nil {
		t.Fatalf("resolveCallName: %v", err)
	}
	if callName != "engine.math.add_one" {
		t.Fatalf("resolved call = %q, want engine.math.add_one", callName)
	}

	ref := frontend.TypeRef{At: pos, Kind: frontend.TypeRefNamed, Name: "math.Vec2"}
	typeName, err := resolveTypeName(&ref, "app.main", imports)
	if err != nil {
		t.Fatalf("resolveTypeName: %v", err)
	}
	if typeName != "engine.math.Vec2" {
		t.Fatalf("resolved type = %q, want engine.math.Vec2", typeName)
	}
}

func TestResolutionDiagnosticForInvalidAliasCallShape(t *testing.T) {
	pos := frontend.Position{File: "app/main.tetra", Line: 4, Col: 7}
	_, err := resolveCallName("math.", "app.main", map[string]string{"math": "engine.math"}, pos)
	if err == nil {
		t.Fatalf("expected alias call shape error")
	}
	if !strings.Contains(err.Error(), "app/main.tetra:4:7: expected 'math.<func>'") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolutionDisplayTextForEnumCaseUsesLocalTypeName(t *testing.T) {
	pos := frontend.Position{File: "app/main.tetra", Line: 6, Col: 10}
	types := map[string]*TypeInfo{
		"app.main.Color": {
			Name: "app.main.Color",
			Kind: TypeEnum,
			CaseMap: map[string]EnumCaseInfo{
				"Red": {Name: "Red", Ordinal: 0},
			},
		},
	}
	expr := &frontend.FieldAccessExpr{
		At:    pos,
		Base:  &frontend.IdentExpr{Name: "Color", At: pos},
		Field: "Blue",
	}

	_, _, ok, err := resolveEnumCaseExpr(expr, nil, nil, types, "app.main", nil)
	if !ok {
		t.Fatalf("expected enum resolution path")
	}
	if err == nil {
		t.Fatalf("expected unknown enum case error")
	}
	if !strings.Contains(err.Error(), "unknown enum case 'Blue' for 'Color'") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolutionImportAliasConflictWithTopLevelDeclaration(t *testing.T) {
	file := &frontend.FileAST{
		Imports: []frontend.ImportDecl{
			{
				Path:  "engine.math",
				Alias: "math",
				At:    frontend.Position{File: "app/main.tetra", Line: 2, Col: 1},
			},
		},
		Funcs: []*frontend.FuncDecl{
			{Name: "math", Pos: frontend.Position{File: "app/main.tetra", Line: 3, Col: 1}},
		},
	}

	_, err := collectImportAliases(file)
	if err == nil {
		t.Fatalf("expected alias conflict error")
	}
	if !strings.Contains(err.Error(), "import alias 'math' conflicts with declaration 'math'") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolutionImportAliasRequiredBoundary(t *testing.T) {
	file := &frontend.FileAST{
		Imports: []frontend.ImportDecl{
			{
				Path: "engine.math",
				At:   frontend.Position{File: "app/main.tetra", Line: 2, Col: 1},
			},
		},
	}

	_, err := collectImportAliases(file)
	if err == nil {
		t.Fatalf("expected alias required error")
	}
	if !strings.Contains(err.Error(), "app/main.tetra:2:1: import alias required") {
		t.Fatalf("error = %v", err)
	}
}
