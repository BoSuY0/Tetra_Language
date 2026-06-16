package frontend

import (
	"strings"
	"testing"
)

func TestParseGenericProtocolRequirement(t *testing.T) {
	src := `
protocol Mapper:
    func map<T>(self: Vec2, value: T) -> T

struct Vec2:
    x: Int
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Protocols) != 1 {
		t.Fatalf("protocol count = %d, want 1", len(prog.Protocols))
	}
	reqs := prog.Protocols[0].Requirements
	if len(reqs) != 1 {
		t.Fatalf("requirement count = %d, want 1", len(reqs))
	}
	req := reqs[0]
	if req.Name != "map" {
		t.Fatalf("requirement name = %q, want map", req.Name)
	}
	if len(req.TypeParams) != 1 || req.TypeParams[0] != "T" {
		t.Fatalf("type params = %#v, want [T]", req.TypeParams)
	}
	if len(req.Params) != 2 || req.Params[1].Type.Name != "T" {
		t.Fatalf("params = %#v, want second param type T", req.Params)
	}
	if req.ReturnType.Name != "T" {
		t.Fatalf("return type = %q, want T", req.ReturnType.Name)
	}
}

func TestParseGenericProtocolRequirementRejectsDuplicateTypeParam(t *testing.T) {
	_, err := Parse([]byte("protocol P:\n  func id<T, T>(x: T) -> T\n"))
	if err == nil {
		t.Fatalf("expected duplicate type parameter diagnostic")
	}
	if !strings.Contains(err.Error(), "duplicate type parameter 'T'") {
		t.Fatalf("error = %v", err)
	}
}

func TestParseStateAndViewDecls(t *testing.T) {
	src := `
state CounterState:
    var count: Int = 0
    val title: String = "Counter"

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    bind titleText: String = state.title
    event click -> increment
    command increment:
        state.count = state.count + 1
    style width: Int = 320
    accessibility label: String = "increment"

func main() -> Int:
    return 0
`
	file, err := ParseFile([]byte(src), "ui/counter.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.States) != 1 {
		t.Fatalf("states = %d, want 1", len(file.States))
	}
	if len(file.Views) != 1 {
		t.Fatalf("views = %d, want 1", len(file.Views))
	}
	state := file.States[0]
	if state.Name != "CounterState" || len(state.Fields) != 2 {
		t.Fatalf("state = %#v", state)
	}
	view := file.Views[0]
	if view.Name != "CounterView" {
		t.Fatalf("view name = %q, want CounterView", view.Name)
	}
	if view.StateName.Name != "CounterState" {
		t.Fatalf("view state = %q, want CounterState", view.StateName.Name)
	}
	if len(view.Bindings) != 2 || len(view.Events) != 1 || len(view.Commands) != 1 || len(view.Styles) != 1 || len(view.Accessibility) != 1 {
		t.Fatalf("view sections = %#v", view)
	}
}

func TestParseViewRequiresCommand(t *testing.T) {
	src := `
state S:
    var count: Int = 0

view Broken(state: S):
    bind value: Int = state.count
`
	_, err := Parse([]byte(src))
	if err == nil {
		t.Fatalf("expected parse error")
	}
	if !strings.Contains(err.Error(), "view requires at least one command") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestParseClosureLiteralExpression(t *testing.T) {
	src := "fn main() -> i32 { let f: ptr = fn(x: i32) -> i32 { return x }; return 0 }"
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) < 2 {
		t.Fatalf("func count = %d, want at least 2 (main + synthetic closure)", len(prog.Funcs))
	}
	mainFn := prog.Funcs[0]
	letStmt, ok := mainFn.Body[0].(*LetStmt)
	if !ok {
		t.Fatalf("stmt = %T, want LetStmt", mainFn.Body[0])
	}
	closure, ok := letStmt.Value.(*ClosureExpr)
	if !ok {
		t.Fatalf("let value = %T, want ClosureExpr", letStmt.Value)
	}
	if closure.Name == "" {
		t.Fatalf("closure name = empty")
	}
	if !prog.Funcs[1].Synthetic || prog.Funcs[1].Name != closure.Name {
		t.Fatalf("synthetic closure func mismatch: %#v", prog.Funcs[1])
	}
}

func TestParseTopLevelClosureDeclaration(t *testing.T) {
	src := `
closure add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return add1(41)
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 2 {
		t.Fatalf("func count = %d, want 2", len(prog.Funcs))
	}
	closure := prog.Funcs[0]
	if closure.Name != "add1" {
		t.Fatalf("closure name = %q, want add1", closure.Name)
	}
	if closure.Synthetic {
		t.Fatalf("top-level closure should not be synthetic")
	}
	if len(closure.Params) != 1 || closure.Params[0].Name != "x" || closure.Params[0].Type.Name != "Int" {
		t.Fatalf("closure params = %#v", closure.Params)
	}
	if closure.ReturnType.Name != "Int" {
		t.Fatalf("closure return type = %q, want Int", closure.ReturnType.Name)
	}
}

func TestParseTopLevelClosureDeclarationMatrix(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		wantName    string
		wantParams  []string
		wantRetKind TypeRefKind
		wantRetName string
		wantUses    string
		wantClauses string
		wantBodyLen int
	}{
		{
			name: "expression bodied",
			src: `closure add1(x: Int) -> Int = x + 1
`,
			wantName:    "add1",
			wantParams:  []string{"x:Int"},
			wantRetName: "Int",
			wantBodyLen: 1,
		},
		{
			name:        "generic bounded uses and clause",
			src:         `closure id<T: Eq>(x: T) -> T uses io nothrow { return x }`,
			wantName:    "id",
			wantParams:  []string{"x:T"},
			wantRetName: "T",
			wantUses:    "io",
			wantClauses: "nothrow",
			wantBodyLen: 1,
		},
		{
			name:        "throws",
			src:         `closure fail() -> Int throws Boom { throw Boom.bad }`,
			wantName:    "fail",
			wantRetName: "Int",
			wantBodyLen: 1,
		},
		{
			name:        "optional return",
			src:         `closure maybe(x: Int) -> Int? { return none }`,
			wantName:    "maybe",
			wantParams:  []string{"x:Int"},
			wantRetKind: TypeRefOptional,
			wantBodyLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := Parse([]byte(tt.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(prog.Funcs) != 1 {
				t.Fatalf("func count = %d, want 1: %#v", len(prog.Funcs), prog.Funcs)
			}
			fn := prog.Funcs[0]
			if fn.Name != tt.wantName || fn.Synthetic {
				t.Fatalf("closure identity = name %q synthetic %v", fn.Name, fn.Synthetic)
			}
			if got := len(fn.Params); got != len(tt.wantParams) {
				t.Fatalf("param count = %d, want %d", got, len(tt.wantParams))
			}
			for i, want := range tt.wantParams {
				got := fn.Params[i].Name + ":" + fn.Params[i].Type.Name
				if got != want {
					t.Fatalf("param[%d] = %q, want %q", i, got, want)
				}
			}
			if tt.wantRetKind != 0 {
				if fn.ReturnType.Kind != tt.wantRetKind {
					t.Fatalf("return kind = %d, want %d", fn.ReturnType.Kind, tt.wantRetKind)
				}
			} else if fn.ReturnType.Name != tt.wantRetName {
				t.Fatalf("return type = %#v, want %s", fn.ReturnType, tt.wantRetName)
			}
			if got := strings.Join(fn.Uses, ","); got != tt.wantUses {
				t.Fatalf("uses = %q, want %q", got, tt.wantUses)
			}
			var clauses []string
			for _, clause := range fn.SemanticClauses {
				clauses = append(clauses, clause.Name)
			}
			if got := strings.Join(clauses, ","); got != tt.wantClauses {
				t.Fatalf("clauses = %q, want %q", got, tt.wantClauses)
			}
			if tt.name == "generic bounded uses and clause" {
				if len(fn.TypeParams) != 1 || fn.TypeParams[0] != "T" || len(fn.TypeParamBounds) != 1 || fn.TypeParamBounds[0].Bound.Name != "Eq" {
					t.Fatalf("generic metadata = params %#v bounds %#v", fn.TypeParams, fn.TypeParamBounds)
				}
			}
			if fn.HasThrows != (tt.name == "throws") {
				t.Fatalf("HasThrows = %v", fn.HasThrows)
			}
			if got := len(fn.Body); got != tt.wantBodyLen {
				t.Fatalf("body len = %d, want %d", got, tt.wantBodyLen)
			}
		})
	}
}

func TestParseClosureASTInvariants(t *testing.T) {
	src := `
func main() -> Int:
    let f: fn(Int) -> Int = fn(x: Int) -> Int = x + 1
    return 0
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 2 {
		t.Fatalf("func count = %d, want main plus synthetic closure", len(prog.Funcs))
	}
	mainFn, synthetic := prog.Funcs[0], prog.Funcs[1]
	letStmt, ok := mainFn.Body[0].(*LetStmt)
	if !ok {
		t.Fatalf("main stmt[0] = %T, want LetStmt", mainFn.Body[0])
	}
	if letStmt.Type.Kind != TypeRefFunction || letStmt.Type.Return == nil || letStmt.Type.Return.Name != "Int" {
		t.Fatalf("let type = %#v, want function type returning Int", letStmt.Type)
	}
	closure, ok := letStmt.Value.(*ClosureExpr)
	if !ok {
		t.Fatalf("let value = %T, want ClosureExpr", letStmt.Value)
	}
	if closure.Name == "" || closure.Decl == nil {
		t.Fatalf("closure expr = %#v, want name and decl", closure)
	}
	if closure.Decl != synthetic {
		t.Fatalf("closure.Decl does not point at appended synthetic function")
	}
	if closure.Pos() != closure.At {
		t.Fatalf("closure Pos() = %#v, want At %#v", closure.Pos(), closure.At)
	}
	if synthetic.Name != closure.Name || !synthetic.Synthetic {
		t.Fatalf("synthetic identity = %#v, closure name %q", synthetic, closure.Name)
	}
	if synthetic.Pos.Line != closure.At.Line || synthetic.Pos.Col != closure.At.Col {
		t.Fatalf("synthetic pos = %d:%d, closure pos = %d:%d", synthetic.Pos.Line, synthetic.Pos.Col, closure.At.Line, closure.At.Col)
	}
	if len(synthetic.Body) != 1 {
		t.Fatalf("synthetic body len = %d, want 1", len(synthetic.Body))
	}
	ret, ok := synthetic.Body[0].(*ReturnStmt)
	if !ok {
		t.Fatalf("synthetic body[0] = %T, want ReturnStmt", synthetic.Body[0])
	}
	if got := exprString(ret.Value); got != "+(x, 1)" {
		t.Fatalf("return expr = %s, want +(x, 1)", got)
	}
}

func TestParseClosureLiteralSyntaxDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "missing params",
			src:  "fn main() -> i32 { let f: ptr = fn -> i32 { return 0 }; return 0 }",
			want: "expected (, got ->",
		},
		{
			name: "named fn literal",
			src:  "fn main() -> i32 { let f: ptr = fn add1(x: i32) -> i32 { return x }; return 0 }",
			want: "closure literals cannot have names",
		},
		{
			name: "closure keyword literal",
			src:  "fn main() -> i32 { let f: ptr = closure(x: i32) -> i32 { return x }; return 0 }",
			want: "closure literal expressions use 'fn(...) -> Type'",
		},
		{
			name: "missing return arrow",
			src:  "fn main() -> i32 { let f: ptr = fn(x: i32) i32 { return x }; return 0 }",
			want: "expected -> or :, got identifier",
		},
		{
			name: "missing body",
			src:  "fn main() -> i32 { let f: ptr = fn(x: i32) -> i32; return 0 }",
			want: "expected {, got ;",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.src))
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestParseGenericClosureLiteralExpression(t *testing.T) {
	src := "fn main() -> i32 { let f: ptr = fn<T: Eq>(x: T) -> T { return x }; return 0 }"
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) < 2 {
		t.Fatalf("func count = %d, want at least 2 (main + synthetic closure)", len(prog.Funcs))
	}
	closure := prog.Funcs[1]
	if len(closure.TypeParams) != 1 || closure.TypeParams[0] != "T" {
		t.Fatalf("closure type params = %#v, want [T]", closure.TypeParams)
	}
	if len(closure.TypeParamBounds) != 1 {
		t.Fatalf("closure bounds = %#v, want one bound", closure.TypeParamBounds)
	}
	if closure.TypeParamBounds[0].Name != "T" || closure.TypeParamBounds[0].Bound.Name != "Eq" {
		t.Fatalf("closure bound = %#v, want T: Eq", closure.TypeParamBounds[0])
	}
}

func TestParseClosureLiteralMayReferenceOuterIdentifier(t *testing.T) {
	src := "fn main() -> i32 { let y: i32 = 1; let f: ptr = fn(x: i32) -> i32 { return x + y }; return 0 }"
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) < 2 {
		t.Fatalf("func count = %d, want at least 2 (main + synthetic closure)", len(prog.Funcs))
	}
	closure := prog.Funcs[1]
	ret, ok := closure.Body[0].(*ReturnStmt)
	if !ok {
		t.Fatalf("closure stmt = %T, want ReturnStmt", closure.Body[0])
	}
	bin, ok := ret.Value.(*BinaryExpr)
	if !ok {
		t.Fatalf("return value = %T, want BinaryExpr", ret.Value)
	}
	right, ok := bin.Right.(*IdentExpr)
	if !ok || right.Name != "y" {
		t.Fatalf("binary right = %#v, want IdentExpr y", bin.Right)
	}
}

func TestParsePropertyDeclaration(t *testing.T) {
	src := `
property title: Int
property enabled: Bool = true

func main() -> Int:
    if enabled:
        return title
    return 0
`
	file, err := ParseFile([]byte(src), "property.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Funcs) != 1 || len(file.Funcs[0].Body) == 0 {
		t.Fatalf("unexpected funcs: %#v", file.Funcs)
	}
	if len(file.Globals) != 2 {
		t.Fatalf("global count = %d, want 2", len(file.Globals))
	}
	if file.Globals[0].Name != "title" || file.Globals[0].Mutable || file.Globals[0].Const || file.Globals[0].Type.Name != "Int" || file.Globals[0].Init != nil {
		t.Fatalf("property title = %#v", file.Globals[0])
	}
	if file.Globals[1].Name != "enabled" || file.Globals[1].Mutable || file.Globals[1].Const || file.Globals[1].Type.Name != "Bool" || file.Globals[1].Init == nil {
		t.Fatalf("property enabled = %#v", file.Globals[1])
	}
}

func TestParseFunctionSemanticClauses(t *testing.T) {
	src := "fn main() -> i32 noalloc noblock realtime nothrow budget(10) { return 0 }"
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("func count = %d, want 1", len(prog.Funcs))
	}
	clauses := prog.Funcs[0].SemanticClauses
	if len(clauses) != 5 {
		t.Fatalf("semantic clauses = %d, want 5", len(clauses))
	}
	budget := clauses[len(clauses)-1]
	if budget.Name != "budget" {
		t.Fatalf("last clause = %q, want budget", budget.Name)
	}
	value, ok := budget.Value.(*NumberExpr)
	if !ok || value.Value != 10 {
		t.Fatalf("budget value = %#v, want NumberExpr(10)", budget.Value)
	}
}

func TestParsePrivacyConsentSemanticClauses(t *testing.T) {
	src := "fn audit(token: consent.token) -> i32 uses privacy privacy consent(token) { return 0 }"
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	clauses := prog.Funcs[0].SemanticClauses
	if len(clauses) != 2 {
		t.Fatalf("semantic clauses = %d, want 2", len(clauses))
	}
	if clauses[0].Name != "privacy" {
		t.Fatalf("clause[0] = %q, want privacy", clauses[0].Name)
	}
	consent := clauses[1]
	if consent.Name != "consent" {
		t.Fatalf("clause[1] = %q, want consent", consent.Name)
	}
	ident, ok := consent.Value.(*IdentExpr)
	if !ok || ident.Name != "token" {
		t.Fatalf("consent value = %#v, want IdentExpr(token)", consent.Value)
	}
}
