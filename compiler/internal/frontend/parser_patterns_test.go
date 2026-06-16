package frontend

import (
	"strings"
	"testing"
)

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

func TestParseEnumPayloadDeclarations(t *testing.T) {
	src := `
enum Result:
    case ok(Int)
    case err(Int, String)
    case empty
    case nested(core.Error)
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Enums) != 1 {
		t.Fatalf("enum count = %d, want 1", len(prog.Enums))
	}
	cases := prog.Enums[0].Cases
	if len(cases) != 4 {
		t.Fatalf("case count = %d, want 4", len(cases))
	}
	if !cases[0].HasPayload || len(cases[0].Payload) != 1 || cases[0].Payload[0].Name != "Int" {
		t.Fatalf("ok case = %#v", cases[0])
	}
	if !cases[1].HasPayload || len(cases[1].Payload) != 2 || cases[1].Payload[0].Name != "Int" || cases[1].Payload[1].Name != "String" {
		t.Fatalf("err case = %#v", cases[1])
	}
	if cases[2].HasPayload || len(cases[2].Payload) != 0 {
		t.Fatalf("empty case = %#v", cases[2])
	}
	if !cases[3].HasPayload || len(cases[3].Payload) != 1 || cases[3].Payload[0].Name != "core.Error" {
		t.Fatalf("nested case = %#v", cases[3])
	}
}

func TestParseEnumPayloadDeclarationDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "empty payload declaration",
			src: `enum Result:
    case ok()
`,
			want: "enum payload list must contain at least one type",
		},
		{
			name: "trailing comma",
			src: `enum Result:
    case err(Int,)
`,
			want: "enum payload declaration does not allow a trailing comma",
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

func TestParseMatchPayloadPattern(t *testing.T) {
	src := `
enum Result:
    case ok(Int)
    case err(Int, Int)

func main() -> Int:
    match result:
    case Result.ok(value):
        return value
    case Result.err(code, detail):
        return code + detail
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Enums) != 1 || len(prog.Enums[0].Cases) != 2 {
		t.Fatalf("enums = %#v", prog.Enums)
	}
	if got := len(prog.Enums[0].Cases[0].Payload); got != 1 {
		t.Fatalf("ok payload count = %d, want 1", got)
	}
	if got := len(prog.Enums[0].Cases[1].Payload); got != 2 {
		t.Fatalf("err payload count = %d, want 2", got)
	}
	match, ok := prog.Funcs[0].Body[0].(*MatchStmt)
	if !ok {
		t.Fatalf("stmt = %T, want MatchStmt", prog.Funcs[0].Body[0])
	}
	pat, ok := match.Cases[0].Pattern.(*EnumCasePatternExpr)
	if !ok {
		t.Fatalf("pattern = %T, want EnumCasePatternExpr", match.Cases[0].Pattern)
	}
	if pat.TypeName != "Result" || pat.CaseName != "ok" || strings.Join(pat.Bindings, ",") != "value" || !pat.HasPayload {
		t.Fatalf("pattern = %#v", pat)
	}
}

func TestParseCatchPayloadPattern(t *testing.T) {
	src := `
enum ReadError:
    case eof
    case denied(Int)

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 42
    throw ReadError.denied(7)

func main() -> Int:
    let value: Int = catch read(false):
    case ReadError.eof:
        0
    case ReadError.denied(code):
        code
    return value
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	letStmt, ok := prog.Funcs[1].Body[0].(*LetStmt)
	if !ok {
		t.Fatalf("stmt = %T, want LetStmt", prog.Funcs[1].Body[0])
	}
	catchExpr, ok := letStmt.Value.(*CatchExpr)
	if !ok {
		t.Fatalf("let value = %T, want CatchExpr", letStmt.Value)
	}
	if len(catchExpr.Cases) != 2 {
		t.Fatalf("catch case count = %d, want 2", len(catchExpr.Cases))
	}
	pat, ok := catchExpr.Cases[1].Pattern.(*EnumCasePatternExpr)
	if !ok {
		t.Fatalf("pattern = %T, want EnumCasePatternExpr", catchExpr.Cases[1].Pattern)
	}
	if pat.TypeName != "ReadError" || pat.CaseName != "denied" || strings.Join(pat.Bindings, ",") != "code" || !pat.HasPayload {
		t.Fatalf("pattern = %#v", pat)
	}
}

func TestParseModuleQualifiedPayloadPattern(t *testing.T) {
	src := `
func main() -> Int:
    match result:
    case api.Result.ok(value):
        return value
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	match, ok := prog.Funcs[0].Body[0].(*MatchStmt)
	if !ok {
		t.Fatalf("stmt = %T, want MatchStmt", prog.Funcs[0].Body[0])
	}
	pat, ok := match.Cases[0].Pattern.(*EnumCasePatternExpr)
	if !ok {
		t.Fatalf("pattern = %T, want EnumCasePatternExpr", match.Cases[0].Pattern)
	}
	if pat.TypeName != "api.Result" || pat.CaseName != "ok" || strings.Join(pat.Bindings, ",") != "value" || !pat.HasPayload {
		t.Fatalf("pattern = %#v", pat)
	}
}

func TestParseEnumPayloadPatternDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "empty payload pattern",
			src: `func main() -> Int:
    match result:
    case Result.ok():
        return 0
`,
			want: "enum payload pattern requires at least one binding",
		},
		{
			name: "duplicate payload binding",
			src: `func main() -> Int:
    match result:
    case Result.err(code, code):
        return code
`,
			want: "duplicate enum payload binding 'code'",
		},
		{
			name: "underscore payload binding",
			src: `func main() -> Int:
    match result:
    case Result.ok(_):
        return 0
`,
			want: "enum payload pattern binding must be a named identifier",
		},
		{
			name: "unqualified payload pattern",
			src: `func main() -> Int:
    match result:
    case ok(value):
        return value
`,
			want: "payload match patterns require qualified enum case syntax",
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

func TestParseMatchNoPayloadEnumCasePattern(t *testing.T) {
	src := `
enum Result:
    case ok(Int)
    case empty

func main() -> Int:
    match result:
    case Result.empty:
        return 0
    case Result.ok(value):
        return value
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	match, ok := prog.Funcs[0].Body[0].(*MatchStmt)
	if !ok {
		t.Fatalf("stmt = %T, want MatchStmt", prog.Funcs[0].Body[0])
	}
	pat, ok := match.Cases[0].Pattern.(*FieldAccessExpr)
	if !ok {
		t.Fatalf("pattern = %T, want FieldAccessExpr", match.Cases[0].Pattern)
	}
	base, ok := pat.Base.(*IdentExpr)
	if !ok || base.Name != "Result" || pat.Field != "empty" {
		t.Fatalf("pattern = %#v", pat)
	}
}

func TestParseMatchExpression(t *testing.T) {
	src := `
enum Result:
    case ok(Int)
    case err(Int)

func main() -> Int:
    let score: Int = match result:
    case Result.ok(value):
        value
    case Result.err(code):
        code
    return score
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	letStmt, ok := prog.Funcs[0].Body[0].(*LetStmt)
	if !ok {
		t.Fatalf("stmt = %T, want LetStmt", prog.Funcs[0].Body[0])
	}
	match, ok := letStmt.Value.(*MatchExpr)
	if !ok {
		t.Fatalf("let value = %T, want MatchExpr", letStmt.Value)
	}
	if len(match.Cases) != 2 {
		t.Fatalf("case count = %d, want 2", len(match.Cases))
	}
	pat, ok := match.Cases[0].Pattern.(*EnumCasePatternExpr)
	if !ok {
		t.Fatalf("pattern = %T, want EnumCasePatternExpr", match.Cases[0].Pattern)
	}
	if pat.CaseName != "ok" || strings.Join(pat.Bindings, ",") != "value" {
		t.Fatalf("pattern = %#v", pat)
	}
	if _, ok := match.Cases[0].Value.(*IdentExpr); !ok {
		t.Fatalf("case value = %T, want IdentExpr", match.Cases[0].Value)
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
