package frontend

import (
	"strings"
	"testing"
)

func TestParseCapsuleStructuredDiagnostic(t *testing.T) {
	_, err := ParseFile([]byte("capsule Renderable {}"), "ui/view.tetra")
	if err == nil {
		t.Fatalf("expected error")
	}
	diag, ok := DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected structured diagnostic for %T", err)
	}
	if diag.Code != DiagnosticCodeParse || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	if diag.File != "ui/view.tetra" || diag.Line != 1 || diag.Column != 1 {
		t.Fatalf("position = %q:%d:%d, want ui/view.tetra:1:1", diag.File, diag.Line, diag.Column)
	}
	if !strings.Contains(diag.Message, "capsule requires at least one metadata entry") {
		t.Fatalf("message = %q", diag.Message)
	}
	if got := err.Error(); !strings.HasPrefix(got, "ui/view.tetra:1:1: capsule requires at least one metadata entry") {
		t.Fatalf("text diagnostic = %q", got)
	}
}

func TestParseCapsuleDeclaration(t *testing.T) {
	src := `
capsule App:
    id: "tetra://app"
    version: "0.1.0"
    target: "linux-x64"
    flags.enabled: true

func main() -> Int:
    return 0
`
	file, err := ParseFile([]byte(src), "capsule_decl.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Capsules) != 1 {
		t.Fatalf("capsule count = %d, want 1", len(file.Capsules))
	}
	capsule := file.Capsules[0]
	if capsule.Name != "App" {
		t.Fatalf("capsule name = %q, want App", capsule.Name)
	}
	if len(capsule.Entries) != 4 {
		t.Fatalf("entry count = %d, want 4", len(capsule.Entries))
	}
	if capsule.Entries[3].Key != "flags.enabled" {
		t.Fatalf("entry[3].key = %q, want flags.enabled", capsule.Entries[3].Key)
	}
}

func TestParseFlowIndentationStructuredDiagnostic(t *testing.T) {
	_, err := ParseFile([]byte("func main() -> i32:\nreturn 0\n"), "app/main.tetra")
	if err == nil {
		t.Fatalf("expected error")
	}
	diag, ok := DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected structured diagnostic for %T", err)
	}
	if diag.File != "app/main.tetra" || diag.Line != 2 || diag.Column != 1 {
		t.Fatalf("position = %q:%d:%d, want app/main.tetra:2:1", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "expected indented block after ':'" {
		t.Fatalf("message = %q", diag.Message)
	}
	if got := err.Error(); got != "app/main.tetra:2:1: expected indented block after ':'" {
		t.Fatalf("text diagnostic = %q", got)
	}
}

func TestParseTestBlockAndExpect(t *testing.T) {
	file, err := ParseFile([]byte(`
test "math":
    expect 40 + 2 == 42
`), "math_test.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Tests) != 1 {
		t.Fatalf("tests = %d, want 1", len(file.Tests))
	}
	if file.Tests[0].Name != "math" {
		t.Fatalf("test name = %q, want math", file.Tests[0].Name)
	}
	if len(file.Tests[0].Body) != 1 {
		t.Fatalf("test body len = %d, want 1", len(file.Tests[0].Body))
	}
	if _, ok := file.Tests[0].Body[0].(*ExpectStmt); !ok {
		t.Fatalf("test stmt = %T, want ExpectStmt", file.Tests[0].Body[0])
	}
}

func TestParseFlowTestDeclarationCoverage(t *testing.T) {
	file, err := ParseFile([]byte(`
module qa.surface

test "math":
    expect 40 + 2 == 42

func answer() -> Int = 42
`), "qa/surface.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if file.Module != "qa.surface" {
		t.Fatalf("module = %q, want qa.surface", file.Module)
	}
	if len(file.Tests) != 1 || file.Tests[0].Name != "math" {
		t.Fatalf("tests = %#v, want one named math", file.Tests)
	}
	if len(file.Funcs) != 1 || file.Funcs[0].Name != "answer" {
		t.Fatalf("funcs = %#v, want one named answer", file.Funcs)
	}
	if len(file.Funcs[0].Body) != 1 {
		t.Fatalf("expression-bodied function should lower to one statement, got %d", len(file.Funcs[0].Body))
	}
	if _, ok := file.Funcs[0].Body[0].(*ReturnStmt); !ok {
		t.Fatalf("func body stmt = %T, want ReturnStmt", file.Funcs[0].Body[0])
	}
}

func TestParseFlowTestDeclarationDiagnostic(t *testing.T) {
	_, err := ParseFile([]byte("test math:\n    expect 1 == 1\n"), "qa/bad_test_decl.tetra")
	if err == nil {
		t.Fatalf("expected diagnostic")
	}
	diag, ok := DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected structured diagnostic: %T %v", err, err)
	}
	if diag.Code != DiagnosticCodeParse || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	if diag.File != "qa/bad_test_decl.tetra" || diag.Line != 1 || diag.Column != 6 {
		t.Fatalf("position = %q:%d:%d, want qa/bad_test_decl.tetra:1:6", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "expected string, got identifier" {
		t.Fatalf("message = %q, want expected string/identifier diagnostic", diag.Message)
	}
}

func TestParseTestBlockASTShape(t *testing.T) {
	file, err := ParseFile([]byte("test \"math\":\n    expect 40 + 2 == 42\n"), "qa/ast_shape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Tests) != 1 {
		t.Fatalf("tests = %d, want 1", len(file.Tests))
	}
	testDecl := file.Tests[0]
	if got := testDecl.Pos(); got.Line != 1 || got.Col != 1 {
		t.Fatalf("test decl pos = %d:%d, want 1:1", got.Line, got.Col)
	}
	if len(testDecl.Body) != 1 {
		t.Fatalf("test body len = %d, want 1", len(testDecl.Body))
	}
	expectStmt, ok := testDecl.Body[0].(*ExpectStmt)
	if !ok {
		t.Fatalf("test stmt = %T, want ExpectStmt", testDecl.Body[0])
	}
	if pos := expectStmt.Pos(); pos.Line != 2 || pos.Col != 5 {
		t.Fatalf("expect stmt pos = %d:%d, want 2:5", pos.Line, pos.Col)
	}
	root, ok := expectStmt.Cond.(*BinaryExpr)
	if !ok {
		t.Fatalf("expect condition = %T, want BinaryExpr", expectStmt.Cond)
	}
	if root.Op != TokenEqEq {
		t.Fatalf("root op = %s, want ==", TokenName(root.Op))
	}
	if pos := root.Pos(); pos.Line != 2 || pos.Col != 19 {
		t.Fatalf("root expr pos = %d:%d, want 2:19", pos.Line, pos.Col)
	}
}

func TestParseTestBlockASTShapeDiagnostic(t *testing.T) {
	_, err := ParseFile([]byte("test \"math\":\n    expect @\n"), "qa/ast_shape_bad.tetra")
	if err == nil {
		t.Fatalf("expected diagnostic")
	}
	diag, ok := DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected structured diagnostic: %T %v", err, err)
	}
	if diag.File != "qa/ast_shape_bad.tetra" || diag.Line != 2 || diag.Column != 8 {
		t.Fatalf("position = %q:%d:%d, want qa/ast_shape_bad.tetra:2:8", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "expected expression, got ?" {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestParseFlowTestBlockSpanCRLFAndUnicode(t *testing.T) {
	src := "test \"Привіт\":\r\n    expect @\r\n"
	_, err := ParseFile([]byte(src), "qa/span_unicode.tetra")
	if err == nil {
		t.Fatalf("expected diagnostic")
	}
	diag, ok := DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected structured diagnostic: %T %v", err, err)
	}
	if diag.File != "qa/span_unicode.tetra" || diag.Line != 2 || diag.Column != 8 {
		t.Fatalf("position = %q:%d:%d, want qa/span_unicode.tetra:2:8", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "expected expression, got ?" {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestParseStructDecl(t *testing.T) {
	src := "struct Vec2 { x: i32, y: i32 }\nfn main() -> i32 { return 0 }"
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Structs) != 1 {
		t.Fatalf("expected 1 struct, got %d", len(prog.Structs))
	}
	st := prog.Structs[0]
	if st.Name != "Vec2" {
		t.Errorf("struct name = %q, want Vec2", st.Name)
	}
	if len(st.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(st.Fields))
	}
	if st.Fields[0].Name != "x" || st.Fields[1].Name != "y" {
		t.Errorf("field names = %q/%q, want x/y", st.Fields[0].Name, st.Fields[1].Name)
	}
	if st.Repr != StructReprDefault {
		t.Fatalf("default struct repr = %q, want %q", st.Repr, StructReprDefault)
	}
}

func TestParseReprCStructDecl(t *testing.T) {
	src := "repr(C) struct Header { tag: c_int, ptr: ptr }\nfn main() -> i32 { return 0 }"
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Structs) != 1 {
		t.Fatalf("expected 1 struct, got %d", len(prog.Structs))
	}
	st := prog.Structs[0]
	if st.Name != "Header" || st.Repr != StructReprC {
		t.Fatalf("struct = %#v, want Header repr(C)", st)
	}
}

func TestParseGenericStructDeclAndTypeArgs(t *testing.T) {
	src := "struct Box<T>:\n    value: T\nfunc main() -> Int:\n    let b: Box<Int> = Box<Int>{value: 42}\n    return b.value\n"
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Structs) != 1 {
		t.Fatalf("struct count = %d, want 1", len(prog.Structs))
	}
	st := prog.Structs[0]
	if st.Name != "Box" || len(st.TypeParams) != 1 || st.TypeParams[0] != "T" {
		t.Fatalf("struct = %#v, want Box<T>", st)
	}
	if got := st.Fields[0].Type.Name; got != "T" {
		t.Fatalf("field type = %q, want T", got)
	}
	letStmt, ok := prog.Funcs[0].Body[0].(*LetStmt)
	if !ok {
		t.Fatalf("body[0] = %T, want *LetStmt", prog.Funcs[0].Body[0])
	}
	if letStmt.Type.Name != "Box" || len(letStmt.Type.TypeArgs) != 1 || letStmt.Type.TypeArgs[0].Name != "Int" {
		t.Fatalf("let type = %#v, want Box<Int>", letStmt.Type)
	}
	lit, ok := letStmt.Value.(*StructLitExpr)
	if !ok {
		t.Fatalf("let value = %T, want *StructLitExpr", letStmt.Value)
	}
	if lit.Type.Name != "Box" || len(lit.Type.TypeArgs) != 1 || lit.Type.TypeArgs[0].Name != "Int" {
		t.Fatalf("literal type = %#v, want Box<Int>", lit.Type)
	}
}

func TestParseGenericStructRejectsDuplicateTypeParam(t *testing.T) {
	_, err := Parse([]byte("struct Box<T, T>:\n    value: T\n"))
	if err == nil {
		t.Fatalf("expected duplicate type parameter diagnostic")
	}
	if !strings.Contains(err.Error(), "duplicate type parameter 'T'") {
		t.Fatalf("error = %v", err)
	}
}

func TestParseGenericFunctionProtocolBound(t *testing.T) {
	src := "protocol P:\n    func echo(self: Vec) -> Vec\nstruct Vec:\n    x: Int\nfunc id<T: P>(x: T) -> T:\n    return x\n"
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("func count = %d, want 1", len(prog.Funcs))
	}
	fn := prog.Funcs[0]
	if got := fn.TypeParams; len(got) != 1 || got[0] != "T" {
		t.Fatalf("type params = %#v, want [T]", got)
	}
	if got := fn.TypeParamBounds; len(got) != 1 || got[0].Name != "T" || got[0].Bound.Name != "P" {
		t.Fatalf("type param bounds = %#v, want T: P", got)
	}
}

func TestParseCallExpr(t *testing.T) {
	expr, err := parseExpr("add(1, 2)")
	if err != nil {
		t.Fatalf("parseExpr: %v", err)
	}
	call, ok := expr.(*CallExpr)
	if !ok {
		t.Fatalf("expected CallExpr, got %T", expr)
	}
	if call.Name != "add" {
		t.Errorf("call name = %q, want add", call.Name)
	}
	if len(call.Args) != 2 {
		t.Errorf("call args = %d, want 2", len(call.Args))
	}
	if len(call.ArgLabels) != 0 {
		t.Errorf("call arg labels = %#v, want none", call.ArgLabels)
	}
}

func TestParseCallExprWithArgumentLabels(t *testing.T) {
	expr, err := parseExpr("add(a: 1, b: 2)")
	if err != nil {
		t.Fatalf("parseExpr: %v", err)
	}
	call, ok := expr.(*CallExpr)
	if !ok {
		t.Fatalf("expected CallExpr, got %T", expr)
	}
	if call.Name != "add" {
		t.Errorf("call name = %q, want add", call.Name)
	}
	if len(call.Args) != 2 {
		t.Fatalf("call args = %d, want 2", len(call.Args))
	}
	if len(call.ArgLabels) != 2 || call.ArgLabels[0] != "a" || call.ArgLabels[1] != "b" {
		t.Fatalf("call arg labels = %#v, want [\"a\", \"b\"]", call.ArgLabels)
	}
}

func TestParseStructCallLiteralStillUsesFieldLabels(t *testing.T) {
	src := "struct Vec2 { x: i32, y: i32 }\nfn main() -> i32 { var v: Vec2 = Vec2(x: 1, y: 2); return v.x + v.y }"
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 1 || len(prog.Funcs[0].Body) < 1 {
		t.Fatalf("expected function with body")
	}
	letStmt, ok := prog.Funcs[0].Body[0].(*LetStmt)
	if !ok {
		t.Fatalf("expected LetStmt, got %T", prog.Funcs[0].Body[0])
	}
	lit, ok := letStmt.Value.(*StructLitExpr)
	if !ok {
		t.Fatalf("expected StructLitExpr, got %T", letStmt.Value)
	}
	if len(lit.Fields) != 2 || lit.Fields[0].Name != "x" || lit.Fields[1].Name != "y" {
		t.Fatalf("struct fields = %#v, want x/y", lit.Fields)
	}
}

func TestParseQualifiedCall(t *testing.T) {
	expr, err := parseExpr("mod.foo(1)")
	if err != nil {
		t.Fatalf("parseExpr: %v", err)
	}
	call, ok := expr.(*CallExpr)
	if !ok {
		t.Fatalf("expected CallExpr, got %T", expr)
	}
	if call.Name != "mod.foo" {
		t.Errorf("call name = %q, want mod.foo", call.Name)
	}
}

func TestParseFieldAccess(t *testing.T) {
	expr, err := parseExpr("v.x")
	if err != nil {
		t.Fatalf("parseExpr: %v", err)
	}
	fa, ok := expr.(*FieldAccessExpr)
	if !ok {
		t.Fatalf("expected FieldAccessExpr, got %T", expr)
	}
	if fa.Field != "x" {
		t.Errorf("field = %q, want x", fa.Field)
	}
	base, ok := fa.Base.(*IdentExpr)
	if !ok {
		t.Fatalf("expected IdentExpr base, got %T", fa.Base)
	}
	if base.Name != "v" {
		t.Errorf("base name = %q, want v", base.Name)
	}
}

func TestParseStructLiteral(t *testing.T) {
	src := "struct Vec2 { x: i32, y: i32 }\nfn main() -> i32 { var v: Vec2 = Vec2{ x: 1, y: 2 }; return 0 }"
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 1 || len(prog.Funcs[0].Body) < 1 {
		t.Fatalf("expected function with body")
	}
	letStmt, ok := prog.Funcs[0].Body[0].(*LetStmt)
	if !ok {
		t.Fatalf("expected LetStmt, got %T", prog.Funcs[0].Body[0])
	}
	lit, ok := letStmt.Value.(*StructLitExpr)
	if !ok {
		t.Fatalf("expected StructLitExpr, got %T", letStmt.Value)
	}
	if len(lit.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(lit.Fields))
	}
}

func TestParseModuleAndImport(t *testing.T) {
	src := "module foo.bar\nimport baz.qux as q\nfn main() -> i32 { return 0 }"
	file, err := ParseFile([]byte(src), "test.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if file.Module != "foo.bar" {
		t.Errorf("module = %q, want foo.bar", file.Module)
	}
	if len(file.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(file.Imports))
	}
	if file.Imports[0].Path != "baz.qux" {
		t.Errorf("import path = %q, want baz.qux", file.Imports[0].Path)
	}
	if file.Imports[0].Alias != "q" {
		t.Errorf("import alias = %q, want q", file.Imports[0].Alias)
	}
}

func TestParsePublicAndSelectiveImports(t *testing.T) {
	src := `module app.main
pub import engine.math.{add, Vec}
import engine.types as types
pub struct Frame { v: Vec }
pub fn main() -> i32 { return add(40, 2) }`
	file, err := ParseFile([]byte(src), "test.t4")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Imports) != 2 {
		t.Fatalf("imports = %d, want 2", len(file.Imports))
	}
	if imp := file.Imports[0]; imp.Path != "engine.math" || imp.Alias != "" || !imp.Public {
		t.Fatalf("import[0] = %#v, want public selective engine.math", imp)
	}
	if got := file.Imports[0].Items; len(got) != 2 || got[0] != "add" || got[1] != "Vec" {
		t.Fatalf("import[0].Items = %#v, want [add Vec]", got)
	}
	if !file.Structs[0].Public {
		t.Fatalf("struct Frame should be public")
	}
	if !file.Funcs[0].Public {
		t.Fatalf("main should be public")
	}
}

func TestParseGlobalDecl(t *testing.T) {
	src := "var g: i32 = 1\nvar ok: bool = true\nval c = 42\nconst k: i32 = 7\nfn main() -> i32 { return 0 }"
	file, err := ParseFile([]byte(src), "test.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Globals) != 4 {
		t.Fatalf("expected 4 globals, got %d", len(file.Globals))
	}
	if file.Globals[0].Name != "g" || !file.Globals[0].Mutable {
		t.Errorf("global[0] = %q mutable=%v, want g/true", file.Globals[0].Name, file.Globals[0].Mutable)
	}
	if file.Globals[0].Init == nil {
		t.Fatalf("global[0] expected initializer")
	}
	if file.Globals[1].Name != "ok" || !file.Globals[1].Mutable {
		t.Errorf("global[1] = %q mutable=%v, want ok/true", file.Globals[1].Name, file.Globals[1].Mutable)
	}
	if file.Globals[1].Init == nil {
		t.Fatalf("global[1] expected initializer")
	}
	if file.Globals[2].Name != "c" || file.Globals[2].Mutable {
		t.Errorf("global[2] = %q mutable=%v, want c/false", file.Globals[2].Name, file.Globals[2].Mutable)
	}
	if file.Globals[3].Name != "k" || file.Globals[3].Mutable || !file.Globals[3].Const {
		t.Errorf("global[3] = %q mutable=%v const=%v, want k/false/true", file.Globals[3].Name, file.Globals[3].Mutable, file.Globals[3].Const)
	}
}

func TestParseGlobalVarDeclRequiresExplicitType(t *testing.T) {
	src := "var g = 1\nfn main() -> i32 { return 0 }"
	_, err := ParseFile([]byte(src), "test.tetra")
	if err == nil {
		t.Fatalf("expected parse error")
	}
	if !strings.Contains(err.Error(), "global var requires an explicit type annotation") {
		t.Fatalf("error = %v", err)
	}
}

func TestParseUnaryMinus(t *testing.T) {
	expr, err := parseExpr("-42")
	if err != nil {
		t.Fatalf("parseExpr: %v", err)
	}
	u, ok := expr.(*UnaryExpr)
	if !ok {
		t.Fatalf("expected UnaryExpr, got %T", expr)
	}
	if u.Op != TokenMinus {
		t.Errorf("op = %s, want -", TokenName(u.Op))
	}
	num, ok := u.X.(*NumberExpr)
	if !ok {
		t.Fatalf("expected NumberExpr, got %T", u.X)
	}
	if num.Value != 42 {
		t.Errorf("value = %d, want 42", num.Value)
	}
}

func TestParseIntegerLiteralRange(t *testing.T) {
	t.Run("i32 max", func(t *testing.T) {
		expr, err := parseExpr("2147483647")
		if err != nil {
			t.Fatalf("parseExpr: %v", err)
		}
		num, ok := expr.(*NumberExpr)
		if !ok {
			t.Fatalf("expr = %T, want NumberExpr", expr)
		}
		if num.Value != 2147483647 {
			t.Fatalf("value = %d, want 2147483647", num.Value)
		}
	})
	t.Run("i32 min", func(t *testing.T) {
		expr, err := parseExpr("-2147483648")
		if err != nil {
			t.Fatalf("parseExpr: %v", err)
		}
		num, ok := expr.(*NumberExpr)
		if !ok {
			t.Fatalf("expr = %T, want NumberExpr", expr)
		}
		if num.Value != -2147483648 {
			t.Fatalf("value = %d, want -2147483648", num.Value)
		}
	})

	tests := []struct {
		name string
		src  string
	}{
		{
			name: "local int overflow",
			src:  "func main() -> Int:\n    let x: Int = 2147483648\n    return x\n",
		},
		{
			name: "global const overflow",
			src:  "const wrapped: Int = 4294967295\nfunc main() -> Int:\n    return wrapped\n",
		},
		{
			name: "budget overflow",
			src:  "func main() -> Int budget(4294967296):\n    return 0\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFile([]byte(tt.src), tt.name+".tetra")
			if err == nil {
				t.Fatalf("expected integer literal range diagnostic")
			}
			if !strings.Contains(err.Error(), "integer literal") || !strings.Contains(err.Error(), "exceeds i32 range") {
				t.Fatalf("error = %v, want integer literal range diagnostic", err)
			}
		})
	}
}

func TestParseExprStmt(t *testing.T) {
	src := "fun side(): i32 { return 0 }\nfun main(): i32 { side(); return 0 }"
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 2 {
		t.Fatalf("expected 2 funcs, got %d", len(prog.Funcs))
	}
	mainFn := prog.Funcs[1]
	if len(mainFn.Body) < 1 {
		t.Fatalf("expected at least 1 stmt in main")
	}
	es, ok := mainFn.Body[0].(*ExprStmt)
	if !ok {
		t.Fatalf("expected ExprStmt, got %T", mainFn.Body[0])
	}
	call, ok := es.Expr.(*CallExpr)
	if !ok {
		t.Fatalf("expected CallExpr in ExprStmt, got %T", es.Expr)
	}
	if call.Name != "side" {
		t.Errorf("call name = %q, want side", call.Name)
	}
}

func TestParseExprStmtQualified(t *testing.T) {
	src := "module test\nimport foo.bar as fb\nfun main(): i32 { fb.noop(); return 0 }"
	file, err := ParseFile([]byte(src), "test.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Funcs) < 1 || len(file.Funcs[0].Body) < 1 {
		t.Fatalf("expected function with body")
	}
	es, ok := file.Funcs[0].Body[0].(*ExprStmt)
	if !ok {
		t.Fatalf("expected ExprStmt, got %T", file.Funcs[0].Body[0])
	}
	call, ok := es.Expr.(*CallExpr)
	if !ok {
		t.Fatalf("expected CallExpr in ExprStmt, got %T", es.Expr)
	}
	if call.Name != "fb.noop" {
		t.Errorf("call name = %q, want fb.noop", call.Name)
	}
}

func TestParseParenGrouping(t *testing.T) {
	expr, err := parseExpr("(1 + 2) * 3")
	if err != nil {
		t.Fatalf("parseExpr: %v", err)
	}
	got := exprString(expr)
	want := "*(+(1, 2), 3)"
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
