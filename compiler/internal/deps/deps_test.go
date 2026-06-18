package deps

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func TestModuleDependencyCollectExternalCalleesByModule(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Funcs: []semantics.CheckedFunc{
			{
				Module: "app.main",
				Decl: &frontend.FuncDecl{
					Body: []frontend.Stmt{
						&frontend.ReturnStmt{
							Value: &frontend.CallExpr{
								Name: "engine.math.norm",
								Args: []frontend.Expr{
									&frontend.NumberExpr{Value: 1},
								},
							},
						},
					},
				},
			},
			{
				Module: "engine.math",
				Decl: &frontend.FuncDecl{
					Body: []frontend.Stmt{
						&frontend.ReturnStmt{
							Value: &frontend.CallExpr{
								Name: "engine.math.identity",
								Args: []frontend.Expr{
									&frontend.NumberExpr{Value: 2},
								},
							},
						},
					},
				},
			},
		},
	}

	got := CollectExternalCalleesByModule(checked)
	if _, ok := got["app.main"]["engine.math.norm"]; !ok {
		t.Fatalf("app.main deps = %#v, want engine.math.norm", got["app.main"])
	}
	if len(got["engine.math"]) != 0 {
		t.Fatalf("engine.math deps = %#v, want none for intra-module calls", got["engine.math"])
	}
}

func TestModuleDependencySkipsLocalFunctionTypedCallbackCallee(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Funcs: []semantics.CheckedFunc{
			{
				Module: "lib.callbacks",
				Locals: map[string]semantics.LocalInfo{
					"cb": {
						FunctionTypeValue: true,
						FunctionParamTypes: []string{
							"i32",
						},
						FunctionReturnType: "i32",
					},
				},
				Decl: &frontend.FuncDecl{
					Body: []frontend.Stmt{
						&frontend.ReturnStmt{
							Value: &frontend.CallExpr{
								Name: "cb",
								Args: []frontend.Expr{
									&frontend.NumberExpr{Value: 1},
								},
							},
						},
					},
				},
			},
		},
	}

	got := CollectExternalCalleesByModule(checked)
	if len(got["lib.callbacks"]) != 0 {
		t.Fatalf(
			"lib.callbacks deps = %#v, want none for local callback callee",
			got["lib.callbacks"],
		)
	}
}

func TestModuleDependencySkipsTargetSetFunctionTypedStructFieldCall(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Funcs: []semantics.CheckedFunc{
			{
				Module: "lib.callbacks",
				Locals: map[string]semantics.LocalInfo{
					"holder": {
						FunctionFields: map[string]semantics.FunctionFieldInfo{
							"cb": {
								FunctionParamTypes: []string{"i32"},
								FunctionReturnType: "i32",
							},
						},
					},
				},
				Decl: &frontend.FuncDecl{
					Body: []frontend.Stmt{
						&frontend.ReturnStmt{
							Value: &frontend.CallExpr{
								Name: "holder.cb",
								Args: []frontend.Expr{
									&frontend.NumberExpr{Value: 1},
								},
							},
						},
					},
				},
			},
		},
	}

	got := CollectExternalCalleesByModule(checked)
	if len(got["lib.callbacks"]) != 0 {
		t.Fatalf(
			"lib.callbacks deps = %#v, want none for target-set function field callee",
			got["lib.callbacks"],
		)
	}
}

func TestModuleDependencySkipsSameModuleEnumConstructor(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Types: map[string]*semantics.TypeInfo{
			"app.main.Maybe": {
				Kind: semantics.TypeEnum,
				CaseMap: map[string]semantics.EnumCaseInfo{
					"some": {Name: "some"},
				},
			},
		},
		Funcs: []semantics.CheckedFunc{
			{
				Module: "app.main",
				Decl: &frontend.FuncDecl{
					Body: []frontend.Stmt{
						&frontend.ReturnStmt{
							Value: &frontend.CallExpr{
								Name: "Maybe.some",
								Args: []frontend.Expr{&frontend.NumberExpr{Value: 1}},
							},
						},
					},
				},
			},
		},
	}

	got := CollectExternalCalleesByModule(checked)
	if len(got["app.main"]) != 0 {
		t.Fatalf("app.main deps = %#v, want none for same-module enum constructor", got["app.main"])
	}
}

func TestModuleDependencyCollectsImportedFunctionFieldAccess(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Funcs: []semantics.CheckedFunc{
			{
				Module:  "app.main",
				Imports: map[string]string{"math": "lib.math"},
				Decl: &frontend.FuncDecl{
					Body: []frontend.Stmt{
						&frontend.ReturnStmt{
							Value: &frontend.FieldAccessExpr{
								Base:  &frontend.IdentExpr{Name: "math"},
								Field: "add1",
							},
						},
					},
				},
			},
		},
	}

	got := CollectExternalCalleesByModule(checked)
	if _, ok := got["app.main"]["lib.math.add1"]; !ok {
		t.Fatalf("app.main deps = %#v, want lib.math.add1", got["app.main"])
	}
}

func TestModuleDependencyCollectsImportedFunctionTypedGlobalBackingSymbol(t *testing.T) {
	checked := &semantics.CheckedProgram{
		GlobalsByModule: map[string]map[string]semantics.GlobalInfo{
			"app.main": {
				"math.cb": {
					TypeName:          "fnptr",
					FunctionTypeValue: true,
					FunctionValue:     "lib.math.add2",
				},
			},
		},
		Funcs: []semantics.CheckedFunc{
			{
				Module:  "app.main",
				Imports: map[string]string{"math": "lib.math"},
				Decl: &frontend.FuncDecl{
					Body: []frontend.Stmt{
						&frontend.ReturnStmt{
							Value: &frontend.CallExpr{
								Name: "app.main.apply",
								Args: []frontend.Expr{
									&frontend.FieldAccessExpr{
										Base:  &frontend.IdentExpr{Name: "math"},
										Field: "cb",
									},
									&frontend.NumberExpr{Value: 40},
								},
							},
						},
					},
				},
			},
		},
	}

	got := CollectExternalCalleesByModule(checked)
	if _, ok := got["app.main"]["lib.math.add2"]; !ok {
		t.Fatalf("app.main deps = %#v, want lib.math.add2", got["app.main"])
	}
	if _, ok := got["app.main"]["lib.math.cb"]; ok {
		t.Fatalf(
			"app.main deps = %#v, did not want pseudo global symbol lib.math.cb",
			got["app.main"],
		)
	}
}

func TestModuleDependencySkipsImportedEnumTypePrefixAndNoPayloadCase(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Types: map[string]*semantics.TypeInfo{
			"lib.result.Result": {
				Kind: semantics.TypeEnum,
				CaseMap: map[string]semantics.EnumCaseInfo{
					"empty": {Name: "empty"},
				},
			},
		},
		Funcs: []semantics.CheckedFunc{
			{
				Module:  "app.main",
				Imports: map[string]string{"res": "lib.result"},
				Decl: &frontend.FuncDecl{
					Body: []frontend.Stmt{
						&frontend.LetStmt{
							Name: "result",
							Value: &frontend.FieldAccessExpr{
								Base: &frontend.FieldAccessExpr{
									Base:  &frontend.IdentExpr{Name: "res"},
									Field: "Result",
								},
								Field: "empty",
							},
						},
					},
				},
			},
		},
	}

	got := CollectExternalCalleesByModule(checked)
	if len(got["app.main"]) != 0 {
		t.Fatalf("app.main deps = %#v, want none for imported enum case", got["app.main"])
	}
}

func TestModuleDependencySkipsSelectiveImportedEnumConstructorCall(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Types: map[string]*semantics.TypeInfo{
			"lib.callbacks.MaybeCallback": {
				Kind: semantics.TypeEnum,
				CaseMap: map[string]semantics.EnumCaseInfo{
					"some": {Name: "some"},
				},
			},
		},
		Funcs: []semantics.CheckedFunc{
			{
				Module: "app.main",
				Imports: map[string]string{
					"MaybeCallback": "\x00symbol:lib.callbacks.MaybeCallback",
				},
				Decl: &frontend.FuncDecl{
					Body: []frontend.Stmt{
						&frontend.ReturnStmt{
							Value: &frontend.CallExpr{
								Name: "MaybeCallback.some",
								Args: []frontend.Expr{&frontend.NumberExpr{Value: 1}},
							},
						},
					},
				},
			},
		},
	}

	got := CollectExternalCalleesByModule(checked)
	if len(got["app.main"]) != 0 {
		t.Fatalf(
			"app.main deps = %#v, want none for selective imported enum constructor",
			got["app.main"],
		)
	}
}

func TestTypeDependencyCollectExternalTypesByModule(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Structs: []semantics.CheckedStruct{
			{
				Module: "app.main",
				Decl: &frontend.StructDecl{
					Name: "Frame",
					Fields: []frontend.FieldDecl{
						{
							Name: "vec",
							Type: frontend.TypeRef{
								Kind: frontend.TypeRefNamed,
								Name: "engine.math.Vec2",
							},
						},
					},
				},
			},
		},
		Funcs: []semantics.CheckedFunc{
			{
				Module: "app.main",
				Decl: &frontend.FuncDecl{
					ReturnType: frontend.TypeRef{
						Kind: frontend.TypeRefNamed,
						Name: "engine.math.Vec2",
					},
					Params: []frontend.ParamDecl{
						{
							Name: "input",
							Type: frontend.TypeRef{
								Kind: frontend.TypeRefNamed,
								Name: "engine.math.Vec2",
							},
						},
					},
				},
			},
			{
				Module: "engine.math",
				Decl: &frontend.FuncDecl{
					ReturnType: frontend.TypeRef{
						Kind: frontend.TypeRefNamed,
						Name: "engine.math.Vec2",
					},
				},
			},
		},
	}

	got := CollectExternalTypesByModule(checked)
	if _, ok := got["app.main"]["engine.math.Vec2"]; !ok {
		t.Fatalf("app.main type deps = %#v, want engine.math.Vec2", got["app.main"])
	}
	if _, ok := got["engine.math"]; ok {
		t.Fatalf("engine.math type deps = %#v, want none", got["engine.math"])
	}
}

func TestModuleDependencyBoundaryNilCheckedProgram(t *testing.T) {
	if got := CollectExternalCalleesByModule(nil); len(got) != 0 {
		t.Fatalf("CollectExternalCalleesByModule(nil) = %#v, want empty map", got)
	}
	if got := CollectExternalTypesByModule(nil); len(got) != 0 {
		t.Fatalf("CollectExternalTypesByModule(nil) = %#v, want empty map", got)
	}
}
