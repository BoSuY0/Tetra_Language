package frontend

import (
	"strings"
	"testing"
)

func parseExpr(src string) (Expr, error) {
	full := "fn main() -> i32 { return " + src + " }"
	prog, err := Parse([]byte(full))
	if err != nil {
		return nil, err
	}
	if len(prog.Funcs) == 0 || len(prog.Funcs[0].Body) == 0 {
		return nil, nil
	}
	ret, ok := prog.Funcs[0].Body[0].(*ReturnStmt)
	if !ok {
		return nil, nil
	}
	return ret.Value, nil
}

func exprString(e Expr) string {
	switch v := e.(type) {
	case *NumberExpr:
		return itoa(int(v.Value))
	case *BoolLitExpr:
		if v.Value {
			return "true"
		}
		return "false"
	case *NoneLitExpr:
		return "none"
	case *SomePatternExpr:
		return "some(" + v.Name + ")"
	case *IdentExpr:
		return v.Name
	case *BinaryExpr:
		return TokenName(v.Op) + "(" + exprString(v.Left) + ", " + exprString(v.Right) + ")"
	case *UnaryExpr:
		return TokenName(v.Op) + "(" + exprString(v.X) + ")"
	case *CallExpr:
		s := v.Name + "("
		for i, arg := range v.Args {
			if i > 0 {
				s += ", "
			}
			s += exprString(arg)
		}
		return s + ")"
	case *FieldAccessExpr:
		return exprString(v.Base) + "." + v.Field
	default:
		return "?"
	}
}

func TestParseOptionalTypeAndNone(t *testing.T) {
	src := `
func maybe() -> Int?:
    return none
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("func count = %d, want 1", len(prog.Funcs))
	}
	ret := prog.Funcs[0].ReturnType
	if ret.Kind != TypeRefOptional || ret.Elem == nil || ret.Elem.Name != "Int" {
		t.Fatalf("return type = %#v, want optional Int", ret)
	}
	stmt, ok := prog.Funcs[0].Body[0].(*ReturnStmt)
	if !ok {
		t.Fatalf("stmt = %T, want ReturnStmt", prog.Funcs[0].Body[0])
	}
	if got := exprString(stmt.Value); got != "none" {
		t.Fatalf("return expr = %s, want none", got)
	}
}

func TestParseFlowIfLet(t *testing.T) {
	src := `
func unwrap(value: Int?) -> Int:
    if let x = value:
        return x
    else:
        return 0
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 1 || len(prog.Funcs[0].Body) != 1 {
		t.Fatalf("unexpected program: %#v", prog)
	}
	stmt, ok := prog.Funcs[0].Body[0].(*IfLetStmt)
	if !ok {
		t.Fatalf("stmt = %T, want IfLetStmt", prog.Funcs[0].Body[0])
	}
	if stmt.Name != "x" {
		t.Fatalf("binding name = %q, want x", stmt.Name)
	}
	if _, ok := stmt.Value.(*IdentExpr); !ok {
		t.Fatalf("binding value = %T, want IdentExpr", stmt.Value)
	}
	if len(stmt.Then) != 1 || len(stmt.Else) != 1 {
		t.Fatalf("branches = %d/%d, want 1/1", len(stmt.Then), len(stmt.Else))
	}
}

func TestParseMatchPayloadPatternDiagnostic(t *testing.T) {
	src := `
func main() -> Int:
    match value:
    case Option.some(x):
        return x
`
	_, err := Parse([]byte(src))
	if err == nil {
		t.Fatalf("expected payload pattern diagnostic")
	}
	if !strings.Contains(err.Error(), "payload match patterns are planned") {
		t.Fatalf("error = %v", err)
	}
}

func TestParseMatchNonePattern(t *testing.T) {
	src := `
func main() -> Int:
    match value:
    case none:
        return 0
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	match, ok := prog.Funcs[0].Body[0].(*MatchStmt)
	if !ok {
		t.Fatalf("stmt = %T, want MatchStmt", prog.Funcs[0].Body[0])
	}
	if got := exprString(match.Cases[0].Pattern); got != "none" {
		t.Fatalf("pattern = %s, want none", got)
	}
}

func TestParseMatchSomePattern(t *testing.T) {
	src := `
func main() -> Int:
    match value:
    case some(x):
        return x
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	match, ok := prog.Funcs[0].Body[0].(*MatchStmt)
	if !ok {
		t.Fatalf("stmt = %T, want MatchStmt", prog.Funcs[0].Body[0])
	}
	if got := exprString(match.Cases[0].Pattern); got != "some(x)" {
		t.Fatalf("pattern = %s, want some(x)", got)
	}
}

func TestParseBoolLiterals(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{"true", "true"},
		{"false", "false"},
		{"true && false", "&&(true, false)"},
	}

	for _, tt := range tests {
		expr, err := parseExpr(tt.src)
		if err != nil {
			t.Errorf("parseExpr(%q): %v", tt.src, err)
			continue
		}
		if got := exprString(expr); got != tt.want {
			t.Errorf("parseExpr(%q) = %s, want %s", tt.src, got, tt.want)
		}
	}
}

func TestParseExprPrecedence(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		// multiplicative binds tighter than additive
		{"1 + 2 * 3", "+(1, *(2, 3))"},
		{"1 * 2 + 3", "+(*(1, 2), 3)"},
		// division and modulo at same level as multiply
		{"10 / 2 * 3", "*(/(10, 2), 3)"},
		{"10 % 3 + 1", "+(%(10, 3), 1)"},
		// additive is left-associative
		{"1 + 2 + 3", "+(+(1, 2), 3)"},
		{"1 - 2 - 3", "-(-(1, 2), 3)"},
		// multiplicative is left-associative
		{"2 * 3 * 4", "*(*(2, 3), 4)"},
		// relational binds tighter than equality
		{"a < b == c", "==(<(a, b), c)"},
		// unary minus
		{"-1", "-(1)"},
		{"-1 + 2", "+(-(1), 2)"},
	}

	for _, tt := range tests {
		expr, err := parseExpr(tt.src)
		if err != nil {
			t.Errorf("parseExpr(%q): %v", tt.src, err)
			continue
		}
		got := exprString(expr)
		if got != tt.want {
			t.Errorf("parseExpr(%q) = %s, want %s", tt.src, got, tt.want)
		}
	}
}

func TestParseLogicalPrecedence(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		// && binds tighter than ||
		{"a || b && c", "||(a, &&(b, c))"},
		{"a && b || c", "||(&&(a, b), c)"},
		// && is left-associative
		{"a && b && c", "&&(&&(a, b), c)"},
		// || is left-associative
		{"a || b || c", "||(||(a, b), c)"},
	}

	for _, tt := range tests {
		expr, err := parseExpr(tt.src)
		if err != nil {
			t.Errorf("parseExpr(%q): %v", tt.src, err)
			continue
		}
		got := exprString(expr)
		if got != tt.want {
			t.Errorf("parseExpr(%q) = %s, want %s", tt.src, got, tt.want)
		}
	}
}

func TestParseNonAssociativity(t *testing.T) {
	tests := []struct {
		src string
	}{
		// Chaining equality is not allowed
		{"a == b == c"},
		{"a != b != c"},
		{"a == b != c"},
		// Chaining relational is not allowed
		{"a < b < c"},
		{"a > b > c"},
		{"a <= b <= c"},
		{"a >= b >= c"},
	}

	for _, tt := range tests {
		full := "fn main() -> i32 { return " + tt.src + " }"
		_, err := Parse([]byte(full))
		if err == nil {
			t.Errorf("Parse(%q): expected error for chained operators", tt.src)
		}
	}
}

func TestParseAllStatements(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			"let",
			"fn main() -> i32 { let x: i32 = 1; return x }",
		},
		{
			"var",
			"fn main() -> i32 { var x: i32 = 1; return x }",
		},
		{
			"val",
			"fn main() -> i32 { val x: i32 = 1; return x }",
		},
		{
			"assign",
			"fn main() -> i32 { var x: i32 = 1; x = 2; return x }",
		},
		{
			"if-else",
			"fn main() -> i32 { if (1) { return 1 } else { return 0 } return 0 }",
		},
		{
			"while",
			"fn main() -> i32 { while (0) { return 1 } return 0 }",
		},
		{
			"return",
			"fn main() -> i32 { return 42 }",
		},
		{
			"print",
			`fn main() -> i32 { print("hi"); return 0 }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.src))
			if err != nil {
				t.Errorf("Parse: %v", err)
			}
		})
	}
}

func TestParseFlowSyntaxFunctionAndUses(t *testing.T) {
	src := `
func main() -> i32
uses app.start:
    let x: i32 = 40
    var y: i32 = 2
    if x > y:
        return x + y
    else:
        return 0
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("expected 1 function, got %d", len(prog.Funcs))
	}
	fn := prog.Funcs[0]
	if fn.Name != "main" || fn.ReturnType.Name != "i32" {
		t.Fatalf("unexpected function: %#v", fn)
	}
	if got := strings.Join(fn.Uses, ","); got != "app.start" {
		t.Fatalf("uses = %q, want app.start", got)
	}
	if len(fn.Body) != 3 {
		t.Fatalf("body len = %d, want 3", len(fn.Body))
	}
	letStmt, ok := fn.Body[0].(*LetStmt)
	if !ok {
		t.Fatalf("first stmt = %T, want LetStmt", fn.Body[0])
	}
	if letStmt.Mutable {
		t.Fatalf("Flow let should parse as immutable")
	}
}

func TestParseFlowStructAndNestedBlocks(t *testing.T) {
	src := `
// Comments before Flow syntax should not confuse normalization.
struct Vec2:
    x: Int
    y: Int

func main() -> Int:
    let start: Int = 0

    // Blank lines and comments stay inside the function block.
    var out: Int = start
    while out < 2:
        out = out + 1

    if out == 2:
        unsafe:
            let mem: cap.mem = core.cap_mem()
    else:
        out = 9
    return out
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Structs) != 1 {
		t.Fatalf("struct count = %d, want 1", len(prog.Structs))
	}
	if prog.Structs[0].Name != "Vec2" || len(prog.Structs[0].Fields) != 2 {
		t.Fatalf("unexpected struct: %#v", prog.Structs[0])
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("func count = %d, want 1", len(prog.Funcs))
	}
	if len(prog.Funcs[0].Body) != 5 {
		t.Fatalf("body len = %d, want 5", len(prog.Funcs[0].Body))
	}
	if _, ok := prog.Funcs[0].Body[3].(*IfStmt); !ok {
		t.Fatalf("stmt[3] = %T, want IfStmt", prog.Funcs[0].Body[3])
	}
}

func TestParseFlowIslandBlock(t *testing.T) {
	src := `
func main() -> Int:
    island(64) as isl:
        var msg: []UInt8 = core.island_make_u8(isl, 1)
        msg[0] = 10
    return 0
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs[0].Body) != 2 {
		t.Fatalf("body len = %d, want 2", len(prog.Funcs[0].Body))
	}
	if _, ok := prog.Funcs[0].Body[0].(*IslandStmt); !ok {
		t.Fatalf("stmt[0] = %T, want IslandStmt", prog.Funcs[0].Body[0])
	}
}

func TestParseFlowCoreV015Blocks(t *testing.T) {
	src := `
enum Color:
    case red
    case green

func main() -> Int:
    var total: Int = 0
    for i in 0..<10:
        total = total + i

    let color: Color = Color.green
    match color:
    case Color.red:
        return 1
    case Color.green:
        return total
    case _:
        return 0
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Enums) != 1 {
		t.Fatalf("enum count = %d, want 1", len(prog.Enums))
	}
	if got := prog.Enums[0].Name; got != "Color" {
		t.Fatalf("enum name = %q, want Color", got)
	}
	if len(prog.Enums[0].Cases) != 2 {
		t.Fatalf("enum cases = %d, want 2", len(prog.Enums[0].Cases))
	}
	fn := prog.Funcs[0]
	if _, ok := fn.Body[1].(*ForRangeStmt); !ok {
		t.Fatalf("stmt[1] = %T, want ForRangeStmt", fn.Body[1])
	}
	if _, ok := fn.Body[3].(*MatchStmt); !ok {
		t.Fatalf("stmt[3] = %T, want MatchStmt", fn.Body[3])
	}
}

func TestParseForCollectionStmt(t *testing.T) {
	src := `
func main(xs: []i32) -> Int:
    var total: Int = 0
    for x in xs:
        total = total + x
    return total
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	fn := prog.Funcs[0]
	loop, ok := fn.Body[1].(*ForRangeStmt)
	if !ok {
		t.Fatalf("stmt[1] = %T, want ForRangeStmt", fn.Body[1])
	}
	if loop.Start != nil || loop.End != nil {
		t.Fatalf("collection for has range bounds: start=%T end=%T", loop.Start, loop.End)
	}
	if loop.Iterable == nil {
		t.Fatalf("collection for missing iterable")
	}
	if got := exprString(loop.Iterable); got != "xs" {
		t.Fatalf("iterable = %s, want xs", got)
	}
}

func TestParseBreakContinueStmts(t *testing.T) {
	src := `
func main() -> Int:
    var i: Int = 0
    while i < 10:
        if i == 3:
            continue
        if i == 6:
            break
        i = i + 1
    return i
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	loop, ok := prog.Funcs[0].Body[1].(*WhileStmt)
	if !ok {
		t.Fatalf("stmt[1] = %T, want WhileStmt", prog.Funcs[0].Body[1])
	}
	firstIf, ok := loop.Body[0].(*IfStmt)
	if !ok {
		t.Fatalf("loop stmt[0] = %T, want IfStmt", loop.Body[0])
	}
	if _, ok := firstIf.Then[0].(*ContinueStmt); !ok {
		t.Fatalf("if then stmt = %T, want ContinueStmt", firstIf.Then[0])
	}
	secondIf, ok := loop.Body[1].(*IfStmt)
	if !ok {
		t.Fatalf("loop stmt[1] = %T, want IfStmt", loop.Body[1])
	}
	if _, ok := secondIf.Then[0].(*BreakStmt); !ok {
		t.Fatalf("if then stmt = %T, want BreakStmt", secondIf.Then[0])
	}
}

func TestParseFlowElseIf(t *testing.T) {
	src := `
func main() -> Int:
    let x: Int = 2
    if x == 1:
        return 1
    else if x == 2:
        return 42
    else:
        return 0
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	first, ok := prog.Funcs[0].Body[1].(*IfStmt)
	if !ok {
		t.Fatalf("stmt[1] = %T, want IfStmt", prog.Funcs[0].Body[1])
	}
	if len(first.Else) != 1 {
		t.Fatalf("else len = %d, want 1 nested if", len(first.Else))
	}
	second, ok := first.Else[0].(*IfStmt)
	if !ok {
		t.Fatalf("else stmt = %T, want IfStmt", first.Else[0])
	}
	if len(second.Else) != 1 {
		t.Fatalf("nested else len = %d, want 1", len(second.Else))
	}
}

func TestParseExpressionBodiedFunction(t *testing.T) {
	src := `
func add(a: Int, b: Int) -> Int = a + b
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("func count = %d, want 1", len(prog.Funcs))
	}
	if got := len(prog.Funcs[0].Body); got != 1 {
		t.Fatalf("body len = %d, want 1", got)
	}
	ret, ok := prog.Funcs[0].Body[0].(*ReturnStmt)
	if !ok {
		t.Fatalf("stmt = %T, want ReturnStmt", prog.Funcs[0].Body[0])
	}
	if got := exprString(ret.Value); got != "+(a, b)" {
		t.Fatalf("return expr = %s, want +(a, b)", got)
	}
}

func TestParseLocalConst(t *testing.T) {
	src := `
func main() -> Int:
    const answer: Int = 42
    return answer
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	stmt, ok := prog.Funcs[0].Body[0].(*LetStmt)
	if !ok {
		t.Fatalf("stmt[0] = %T, want LetStmt", prog.Funcs[0].Body[0])
	}
	if stmt.Name != "answer" || stmt.Mutable || !stmt.Const {
		t.Fatalf("local const = name %q mutable %v const %v, want answer/false/true", stmt.Name, stmt.Mutable, stmt.Const)
	}
}

func TestParseCompoundAssignment(t *testing.T) {
	src := `
func main() -> Int:
    var x: Int = 40
    x += 2
    return x
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	stmt, ok := prog.Funcs[0].Body[1].(*AssignStmt)
	if !ok {
		t.Fatalf("stmt[1] = %T, want AssignStmt", prog.Funcs[0].Body[1])
	}
	if stmt.Op != TokenPlus {
		t.Fatalf("assignment op = %s, want +", TokenName(stmt.Op))
	}
	if got := exprString(stmt.CompoundValue); got != "2" {
		t.Fatalf("compound rhs = %s, want 2", got)
	}
	if got := exprString(stmt.Value); got != "+(x, 2)" {
		t.Fatalf("lowered value = %s, want +(x, 2)", got)
	}
}

func TestParseUnaryBangExpr(t *testing.T) {
	expr, err := parseExpr("!ok")
	if err != nil {
		t.Fatalf("parseExpr: %v", err)
	}
	if got := exprString(expr); got != "!(ok)" {
		t.Fatalf("expr = %s, want !(ok)", got)
	}
}

func TestParseFlowIndentationErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "tab",
			src:  "func main() -> Int:\n\treturn 0\n",
			want: "tabs are not supported",
		},
		{
			name: "missing indent",
			src:  "func main() -> Int:\nreturn 0\n",
			want: "expected indented block",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.src))
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestParseLegacySyntaxUnaffectedByFlowMarkersInComments(t *testing.T) {
	src := "// func fake() -> Int:\nfun main(): i32 {\n  return 0\n}\n"
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 1 || prog.Funcs[0].Name != "main" {
		t.Fatalf("unexpected program: %#v", prog)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"missing rparen", "fn main( -> i32 { return 0 }"},
		{"missing return type", "fn main() { return 0 }"},
		{"missing body", "fn main() -> i32"},
		{"bad expr token", "fn main() -> i32 { return @ }"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.src))
			if err == nil {
				t.Errorf("Parse: expected error")
			}
		})
	}
}

func TestParsePlannedFeatureDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"capsule", "capsule app:\n  name: \"app\"\n", "planned feature 'capsule'"},
		{"closure literal", "fn main() -> i32 { val f = fn(x: i32) -> i32 { return x }; return 0 }", "planned feature 'closures'"},
		{"semantic clause", "fn main() -> i32 budget(10) { return 0 }", "planned feature 'semantic clauses'"},
		{"function call argument label", "func add(a: Int, b: Int) -> Int:\n  return a + b\n\nfunc main() -> Int:\n  return add(a: 40, b: 2)\n", "function-call argument labels are planned"},
		{"generic protocol requirement", "protocol P:\n  func id<T>(x: T) -> T\n", "generic protocol requirements are planned"},
		{"generic struct", "struct Box<T>:\n  value: T\n", "generic structs are planned"},
		{"enum payload case", "enum Option:\n  case some(Int)\n", "enum payload cases are planned"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.src))
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestParsePlannedFeatureStructuredDiagnostic(t *testing.T) {
	_, err := ParseFile([]byte("actor Renderable:\n"), "ui/view.tetra")
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
	if !strings.Contains(diag.Message, "planned feature 'actor'") {
		t.Fatalf("message = %q", diag.Message)
	}
	if got := err.Error(); !strings.HasPrefix(got, "ui/view.tetra:1:1: planned feature 'actor'") {
		t.Fatalf("text diagnostic = %q", got)
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

func TestParseGlobalDecl(t *testing.T) {
	src := "var g: i32\nval c = 42\nconst k: i32 = 7\nfn main() -> i32 { return 0 }"
	file, err := ParseFile([]byte(src), "test.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Globals) != 3 {
		t.Fatalf("expected 3 globals, got %d", len(file.Globals))
	}
	if file.Globals[0].Name != "g" || !file.Globals[0].Mutable {
		t.Errorf("global[0] = %q mutable=%v, want g/true", file.Globals[0].Name, file.Globals[0].Mutable)
	}
	if file.Globals[1].Name != "c" || file.Globals[1].Mutable {
		t.Errorf("global[1] = %q mutable=%v, want c/false", file.Globals[1].Name, file.Globals[1].Mutable)
	}
	if file.Globals[2].Name != "k" || file.Globals[2].Mutable || !file.Globals[2].Const {
		t.Errorf("global[2] = %q mutable=%v const=%v, want k/false/true", file.Globals[2].Name, file.Globals[2].Mutable, file.Globals[2].Const)
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
