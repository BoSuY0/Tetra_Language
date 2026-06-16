package frontend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParserFixtureCorpus(t *testing.T) {
	root := filepath.Join("testdata", "parser")
	families := []string{
		"module",
		"function",
		"control_flow",
		"match",
		"optionals",
		"enums",
		"generics",
		"protocols",
		"extensions",
		"async",
		"declarations",
		"tests",
		"ui",
	}

	for _, family := range families {
		t.Run(family+"/positive", func(t *testing.T) {
			path := filepath.Join(root, "positive", family+".tetra")
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			if _, err := ParseFile(src, filepath.ToSlash(path)); err != nil {
				t.Fatalf("ParseFile(%s): %v", path, err)
			}
		})
		t.Run(family+"/negative", func(t *testing.T) {
			path := filepath.Join(root, "negative", family+".tetra")
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			wantPath := filepath.Join(root, "negative", family+".diag")
			wantRaw, err := os.ReadFile(wantPath)
			if err != nil {
				t.Fatalf("read diagnostic fixture: %v", err)
			}
			_, err = ParseFile(src, filepath.ToSlash(path))
			if err == nil {
				t.Fatalf("expected diagnostic from %s", path)
			}
			if got, want := strings.TrimSpace(err.Error()), strings.TrimSpace(string(wantRaw)); got != want {
				t.Fatalf("diagnostic mismatch:\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}

func TestParseFileDiagnosticsParsesCapsuleAndMainWithoutRecovery(t *testing.T) {
	src := []byte(`capsule App:
    id: "tetra://app"
    version: "0.1.0"

func main() -> Int:
    return 0
`)
	file, diagnostics := ParseFileDiagnostics(src, "recovery.tetra")
	if file == nil {
		t.Fatalf("expected file")
	}
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want 0", diagnostics)
	}
	if len(file.Capsules) != 1 || file.Capsules[0].Name != "App" {
		t.Fatalf("capsules = %#v, want one capsule App", file.Capsules)
	}
	if len(file.Funcs) != 1 || file.Funcs[0].Name != "main" {
		t.Fatalf("funcs = %#v, want main", file.Funcs)
	}
}

func TestParseActorDeclarationDesugarsMethods(t *testing.T) {
	src := []byte(`actor Worker:
    func run() -> Int:
        return 7
    func stop() -> Int:
        return 0

func main() -> Int:
    return Worker.run()
`)
	file, err := ParseFile(src, "actor_decl.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Funcs) != 3 {
		t.Fatalf("func count = %d, want 3: %#v", len(file.Funcs), file.Funcs)
	}
	if file.Funcs[0].Name != "Worker.run" || file.Funcs[1].Name != "Worker.stop" || file.Funcs[2].Name != "main" {
		t.Fatalf("func names = %q, %q, %q", file.Funcs[0].Name, file.Funcs[1].Name, file.Funcs[2].Name)
	}
}

func TestParseActorDeclarationPreservesMethodUsesActors(t *testing.T) {
	src := []byte(`actor Worker:
    func run() -> Int
    uses actors:
        let me: actor = core.self()
        return 7
`)
	file, err := ParseFile(src, "actor_uses.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Funcs) != 1 {
		t.Fatalf("func count = %d, want 1: %#v", len(file.Funcs), file.Funcs)
	}
	fn := file.Funcs[0]
	if fn.Name != "Worker.run" {
		t.Fatalf("func name = %q, want Worker.run", fn.Name)
	}
	if got := strings.Join(fn.Uses, ","); got != "actors" {
		t.Fatalf("uses = %q, want actors", got)
	}
}

func TestParseActorDeclarationSupportsImmutableStateFields(t *testing.T) {
	src := []byte(`actor Counter:
    val step: Int
    const ceiling: Int
    func run() -> Int:
        return 0
`)
	file, err := ParseFile(src, "actor_state.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Actors) != 1 {
		t.Fatalf("actor count = %d, want 1: %#v", len(file.Actors), file.Actors)
	}
	actor := file.Actors[0]
	if actor.Name != "Counter" {
		t.Fatalf("actor name = %q, want Counter", actor.Name)
	}
	if len(actor.Fields) != 2 {
		t.Fatalf("field count = %d, want 2: %#v", len(actor.Fields), actor.Fields)
	}
	if actor.Fields[0].Name != "step" || actor.Fields[0].Mutable || actor.Fields[0].Const {
		t.Fatalf("field[0] = %#v", actor.Fields[0])
	}
	if actor.Fields[1].Name != "ceiling" || actor.Fields[1].Mutable || !actor.Fields[1].Const {
		t.Fatalf("field[1] = %#v", actor.Fields[1])
	}
	if len(file.Funcs) != 1 || file.Funcs[0].Name != "Counter.run" {
		t.Fatalf("funcs = %#v", file.Funcs)
	}
}

func TestParseActorDeclarationSupportsStateFieldInitializers(t *testing.T) {
	src := []byte(`actor Counter:
    val step: Int = 2
    const enabled: Bool = true
    func run() -> Int:
        return 0
`)
	file, err := ParseFile(src, "actor_state_init.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Actors) != 1 {
		t.Fatalf("actor count = %d, want 1: %#v", len(file.Actors), file.Actors)
	}
	actor := file.Actors[0]
	if len(actor.Fields) != 2 {
		t.Fatalf("field count = %d, want 2: %#v", len(actor.Fields), actor.Fields)
	}
	if actor.Fields[0].Init == nil {
		t.Fatalf("field[0] initializer = nil, want non-nil")
	}
	if actor.Fields[1].Init == nil {
		t.Fatalf("field[1] initializer = nil, want non-nil")
	}
	if _, ok := actor.Fields[0].Init.(*NumberExpr); !ok {
		t.Fatalf("field[0] init = %T, want *NumberExpr", actor.Fields[0].Init)
	}
	if _, ok := actor.Fields[1].Init.(*BoolLitExpr); !ok {
		t.Fatalf("field[1] init = %T, want *BoolLitExpr", actor.Fields[1].Init)
	}
}

func TestParseActorDeclarationRejectsUnsupportedMembers(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "bare field",
			src: `actor Counter:
    count: Int
`,
			want: "actor state fields must use 'val' or 'const'",
		},
		{
			name: "nested state",
			src: `actor Counter:
    state CounterState:
        var count: Int = 0
`,
			want: "actor declarations do not support nested state blocks yet",
		},
		{
			name: "self member",
			src: `actor Counter:
    self: actor
`,
			want: "actor declarations do not support self members yet",
		},
		{
			name: "self parameter",
			src: `actor Counter:
    func run(self: actor) -> Int:
        return 0
`,
			want: "actor methods do not support explicit self parameters yet",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFile([]byte(tt.src), "actor_bad.tetra")
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestParseActorDeclarationSupportsMutableStateField(t *testing.T) {
	src := []byte(`actor Counter:
    var count: Int = 0
    func run() -> Int:
        return count
`)
	file, err := ParseFile(src, "actor_mutable_state.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Actors) != 1 {
		t.Fatalf("actor count = %d, want 1", len(file.Actors))
	}
	actor := file.Actors[0]
	if len(actor.Fields) != 1 {
		t.Fatalf("field count = %d, want 1", len(actor.Fields))
	}
	if !actor.Fields[0].Mutable || actor.Fields[0].Const {
		t.Fatalf("field = %#v, want mutable non-const field", actor.Fields[0])
	}
}

func TestParseFileDiagnosticsReturnsLexerError(t *testing.T) {
	file, diagnostics := ParseFileDiagnostics([]byte{'f', 'n', ' ', 0xff, '\n'}, "bad.tetra")
	if file != nil {
		t.Fatalf("file = %#v, want nil", file)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want 1", diagnostics)
	}
	if diagnostics[0].Message != "invalid UTF-8 encoding" || diagnostics[0].Line != 1 || diagnostics[0].Column != 4 {
		t.Fatalf("diagnostic = %#v", diagnostics[0])
	}
}

func TestParseGrammarSurfaceExamplesPositive(t *testing.T) {
	for _, name := range []string{"flow_grammar_surface_smoke.tetra", "flow_hello.tetra"} {
		path := filepath.Join("..", "..", "..", "examples", name)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if _, err := ParseFile(raw, filepath.ToSlash(path)); err != nil {
			t.Fatalf("ParseFile(%s): %v", path, err)
		}
	}
}

func TestParseDeferStatementFlowSyntax(t *testing.T) {
	src := []byte(`func main() -> Int
uses io:
    defer:
        print("done\n")
    return 0
`)
	file, err := ParseFile(src, "defer.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(file.Funcs) != 1 || len(file.Funcs[0].Body) != 2 {
		t.Fatalf("func body = %#v", file.Funcs)
	}
	deferStmt, ok := file.Funcs[0].Body[0].(*DeferStmt)
	if !ok {
		t.Fatalf("body[0] = %T, want *DeferStmt", file.Funcs[0].Body[0])
	}
	if len(deferStmt.Body) != 1 {
		t.Fatalf("defer body len = %d, want 1", len(deferStmt.Body))
	}
	if _, ok := deferStmt.Body[0].(*PrintStmt); !ok {
		t.Fatalf("defer body[0] = %T, want *PrintStmt", deferStmt.Body[0])
	}
}

func TestParseGrammarSurfaceExampleNegativeDiagnostic(t *testing.T) {
	path := filepath.Join("..", "..", "..", "examples", "flow_grammar_surface_smoke.tetra")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	bad := "test grammar surface:\n    expect 1 == 1\n\n" + string(raw)
	file := "examples/flow_grammar_surface_smoke.bad.tetra"
	_, err = ParseFile([]byte(bad), file)
	if err == nil {
		t.Fatalf("expected diagnostic")
	}
	diag, ok := DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected structured diagnostic: %T %v", err, err)
	}
	if diag.File != file || diag.Line != 1 || diag.Column != 6 {
		t.Fatalf("position = %q:%d:%d, want %s:1:6", diag.File, diag.Line, diag.Column, file)
	}
	if diag.Message != "expected string, got identifier" {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestParseGrammarSurfaceExampleSpanCRLFUnicode(t *testing.T) {
	path := filepath.Join("..", "..", "..", "examples", "flow_grammar_surface_smoke.tetra")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	base := string(raw)
	src := base + "test \"Привіт\":\r\n    expect @\r\n"
	baseLines := strings.Count(base, "\n") + 1
	file := "examples/flow_grammar_surface_smoke.span.tetra"
	_, err = ParseFile([]byte(src), file)
	if err == nil {
		t.Fatalf("expected diagnostic")
	}
	diag, ok := DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected structured diagnostic: %T %v", err, err)
	}
	if diag.File != file || diag.Line != baseLines+1 || diag.Column != 8 {
		t.Fatalf("position = %q:%d:%d, want %s:%d:8", diag.File, diag.Line, diag.Column, file, baseLines+1)
	}
	if diag.Message != "expected expression, got ?" {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestParsePlannedFeatureMatrixFromFlowSyntaxV1(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"capsule declaration", "capsule App:\n    id: \"tetra://app\"\n    version: \"0.1.0\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseFile([]byte(tt.src), tt.name+".tetra")
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			if len(file.Capsules) != 1 || file.Capsules[0].Name != "App" {
				t.Fatalf("capsules = %#v, want one capsule App", file.Capsules)
			}
		})
	}
}

func TestParseTopLevelPlannedFeatureDiagnostics(t *testing.T) {
	src := []byte(`class Box:
    value: Int

trait Renderable:
    func draw() -> Int

interface Renderable:
    func draw() -> Int

typealias UserID = Int
macro trace:
    return 0

capsule App:
    id: "tetra://planned"

func main() -> Int:
    return 0
`)
	file, diagnostics := ParseFileDiagnostics(src, "planned_forms.tetra")
	if file == nil {
		t.Fatalf("expected recovered file")
	}
	if len(file.Capsules) != 1 || file.Capsules[0].Name != "App" {
		t.Fatalf("capsules = %#v, want one capsule App", file.Capsules)
	}
	if len(file.Funcs) != 1 || file.Funcs[0].Name != "main" {
		t.Fatalf("funcs = %#v, want main", file.Funcs)
	}

	want := []struct {
		line    int
		message string
	}{
		{1, "planned feature 'class declarations' is not implemented in the Tetra v1.0 profile"},
		{4, "planned feature 'trait declarations' is not implemented in the Tetra v1.0 profile"},
		{7, "planned feature 'interface declarations' is not implemented in the Tetra v1.0 profile"},
		{10, "planned feature 'type alias declarations' is not implemented in the Tetra v1.0 profile"},
		{11, "planned feature 'macro declarations' is not implemented in the Tetra v1.0 profile"},
	}
	if len(diagnostics) != len(want) {
		t.Fatalf("diagnostic count = %d, want %d: %#v", len(diagnostics), len(want), diagnostics)
	}
	for i, wantDiag := range want {
		diag := diagnostics[i]
		if diag.File != "planned_forms.tetra" || diag.Line != wantDiag.line || diag.Column != 1 || diag.Message != wantDiag.message {
			t.Fatalf("diagnostic[%d] = %#v, want planned_forms.tetra:%d:1 %q", i, diag, wantDiag.line, wantDiag.message)
		}
	}
}

func TestParseOwnershipParamSyntaxMatrix(t *testing.T) {
	src := `
protocol BufferOps:
    func update(src: borrow []u8, dst: inout []u8, tmp: consume []u8) -> Int

closure local(read: borrow Int, write: inout Int, taken: consume Int) -> Int:
    write = write + read + taken
    return write

func mix(a: borrow Int, b: inout Int, c: consume Int, cb: borrow fn(Int) -> Int) -> Int:
    return cb(a) + b + c
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Protocols) != 1 || len(prog.Protocols[0].Requirements) != 1 {
		t.Fatalf("protocol requirements = %#v", prog.Protocols)
	}
	reqParams := prog.Protocols[0].Requirements[0].Params
	if got := reqParams[0].Ownership + "," + reqParams[1].Ownership + "," + reqParams[2].Ownership; got != "borrow,inout,consume" {
		t.Fatalf("protocol ownership = %q", got)
	}
	if len(prog.Funcs) != 2 {
		t.Fatalf("func count = %d, want closure plus func", len(prog.Funcs))
	}
	closureParams := prog.Funcs[0].Params
	if prog.Funcs[0].Name != "local" || closureParams[0].Ownership != "borrow" || closureParams[1].Ownership != "inout" || closureParams[2].Ownership != "consume" {
		t.Fatalf("closure ownership params = %#v", prog.Funcs[0])
	}
	fnParams := prog.Funcs[1].Params
	if got := fnParams[0].Ownership + "," + fnParams[1].Ownership + "," + fnParams[2].Ownership + "," + fnParams[3].Ownership; got != "borrow,inout,consume,borrow" {
		t.Fatalf("func ownership = %q", got)
	}
	if fnParams[3].Type.Kind != TypeRefFunction {
		t.Fatalf("borrowed callback type = %#v, want function type", fnParams[3].Type)
	}
}

func TestParseOwnershipMarkerSyntaxDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "missing type",
			src:  "func bad(x: borrow) -> Int:\n    return 0\n",
			want: "ownership marker 'borrow' must be followed by a parameter type",
		},
		{
			name: "stacked markers",
			src:  "func bad(x: borrow inout Int) -> Int:\n    return 0\n",
			want: "ownership marker 'inout' cannot follow ownership marker 'borrow'",
		},
		{
			name: "trailing comma after marker",
			src:  "func bad(x: consume, y: Int) -> Int:\n    return 0\n",
			want: "ownership marker 'consume' must be followed by a parameter type",
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

func TestParseBorrowReturnSyntaxMatrix(t *testing.T) {
	src := `
protocol Views:
    func middle(xs: borrow []u8) -> borrow []u8

func view_bytes(xs: borrow []u8) -> borrow []u8:
    return xs.borrow()

func view_text(s: borrow String) -> borrow String:
    return s.borrow()

func accepts_callback(cb: fn(borrow []u8) -> borrow []u8, xs: borrow []u8) -> Int:
    let v: []u8 = cb(xs)
    return v.len
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := prog.Protocols[0].Requirements[0].ReturnOwnership; got != "borrow" {
		t.Fatalf("protocol borrow return ownership = %q, want borrow", got)
	}
	viewBytes := prog.Funcs[0]
	if got := viewBytes.ReturnOwnership; got != "borrow" {
		t.Fatalf("view_bytes return ownership = %q, want borrow", got)
	}
	if viewBytes.ReturnType.Kind != TypeRefSlice || viewBytes.ReturnType.Elem == nil || viewBytes.ReturnType.Elem.Name != "u8" {
		t.Fatalf("view_bytes return type = %#v, want []u8", viewBytes.ReturnType)
	}
	viewText := prog.Funcs[1]
	if got := viewText.ReturnOwnership; got != "borrow" {
		t.Fatalf("view_text return ownership = %q, want borrow", got)
	}
	if viewText.ReturnType.Name != "String" {
		t.Fatalf("view_text return type = %#v, want String", viewText.ReturnType)
	}
	cb := prog.Funcs[2].Params[0].Type
	if cb.Kind != TypeRefFunction || cb.Return == nil {
		t.Fatalf("callback type = %#v, want function return", cb)
	}
	if got := cb.ReturnOwnership; got != "borrow" {
		t.Fatalf("callback return ownership = %q, want borrow", got)
	}
}

func TestParseBorrowReturnRejectsMissingType(t *testing.T) {
	_, err := Parse([]byte("func bad(xs: []u8) -> borrow:\n    return xs\n"))
	if err == nil {
		t.Fatalf("expected diagnostic")
	}
	if !strings.Contains(err.Error(), "expected return type after `borrow`") {
		t.Fatalf("error = %v, want missing borrow return type diagnostic", err)
	}
}

func TestParseFunctionTypeRef(t *testing.T) {
	src := `
func apply(cb: fn(Int, Bool) -> UInt8, value: Int, flag: Bool) -> UInt8:
    return cb(value, flag)
`
	prog, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("func count = %d, want 1", len(prog.Funcs))
	}
	cb := prog.Funcs[0].Params[0].Type
	if cb.Kind != TypeRefFunction {
		t.Fatalf("cb type kind = %d, want TypeRefFunction", cb.Kind)
	}
	if len(cb.Params) != 2 || cb.Return == nil {
		t.Fatalf("cb function type = %#v", cb)
	}
	if cb.Params[0].Kind != TypeRefNamed || cb.Params[0].Name != "Int" {
		t.Fatalf("cb param0 = %#v, want Int", cb.Params[0])
	}
	if cb.Params[1].Kind != TypeRefNamed || cb.Params[1].Name != "Bool" {
		t.Fatalf("cb param1 = %#v, want Bool", cb.Params[1])
	}
	if cb.Return.Kind != TypeRefNamed || cb.Return.Name != "UInt8" {
		t.Fatalf("cb return = %#v, want UInt8", cb.Return)
	}
}

func TestParseFunctionTypeSyntaxMatrix(t *testing.T) {
	tests := []struct {
		name       string
		typeSrc    string
		wantParms  []string
		wantRet    string
		wantThrows string
		wantUses   string
		wantKind   TypeRefKind
	}{
		{
			name:      "zero params",
			typeSrc:   "fn() -> Int",
			wantParms: nil,
			wantRet:   "Int",
		},
		{
			name:      "single param",
			typeSrc:   "fn(Int) -> Bool",
			wantParms: []string{"Int"},
			wantRet:   "Bool",
		},
		{
			name:      "multi param qualified return",
			typeSrc:   "fn(Int, core.String) -> app.Result",
			wantParms: []string{"Int", "core.String"},
			wantRet:   "app.Result",
		},
		{
			name:      "uses effects",
			typeSrc:   "fn(Int) -> Int uses io, privacy",
			wantParms: []string{"Int"},
			wantRet:   "Int",
			wantUses:  "io,privacy",
		},
		{
			name:       "throws error",
			typeSrc:    "fn(Int) -> Int throws Boom",
			wantParms:  []string{"Int"},
			wantRet:    "Int",
			wantThrows: "Boom",
		},
		{
			name:       "throws error with uses effects",
			typeSrc:    "fn(Int) -> Int throws Boom uses io, privacy",
			wantParms:  []string{"Int"},
			wantRet:    "Int",
			wantThrows: "Boom",
			wantUses:   "io,privacy",
		},
		{
			name:      "optional return",
			typeSrc:   "fn(Int) -> Int?",
			wantParms: []string{"Int"},
			wantKind:  TypeRefOptional,
		},
		{
			name:      "nested function param",
			typeSrc:   "fn(fn(Int) -> Int) -> Int",
			wantParms: []string{"fn"},
			wantRet:   "Int",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "func apply(cb: " + tt.typeSrc + ") -> Int:\n    return 0\n"
			prog, err := Parse([]byte(src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			cb := prog.Funcs[0].Params[0].Type
			if cb.Kind != TypeRefFunction {
				t.Fatalf("cb kind = %d, want TypeRefFunction", cb.Kind)
			}
			if got := len(cb.Params); got != len(tt.wantParms) {
				t.Fatalf("param count = %d, want %d: %#v", got, len(tt.wantParms), cb.Params)
			}
			for i, want := range tt.wantParms {
				param := cb.Params[i]
				if want == "fn" {
					if param.Kind != TypeRefFunction || param.Return == nil || param.Return.Name != "Int" {
						t.Fatalf("param[%d] = %#v, want nested fn(Int)->Int", i, param)
					}
					continue
				}
				if param.Kind != TypeRefNamed || param.Name != want {
					t.Fatalf("param[%d] = %#v, want named %s", i, param, want)
				}
			}
			if cb.Return == nil {
				t.Fatalf("return = nil")
			}
			if tt.wantKind != 0 {
				if cb.Return.Kind != tt.wantKind {
					t.Fatalf("return kind = %d, want %d", cb.Return.Kind, tt.wantKind)
				}
			} else if cb.Return.Kind != TypeRefNamed || cb.Return.Name != tt.wantRet {
				t.Fatalf("return = %#v, want named %s", cb.Return, tt.wantRet)
			}
			if got := strings.Join(cb.Uses, ","); got != tt.wantUses {
				t.Fatalf("uses = %q, want %q", got, tt.wantUses)
			}
			if tt.wantThrows == "" {
				if cb.Throws != nil {
					t.Fatalf("throws = %#v, want nil", cb.Throws)
				}
			} else if cb.Throws == nil || cb.Throws.Kind != TypeRefNamed || cb.Throws.Name != tt.wantThrows {
				t.Fatalf("throws = %#v, want named %s", cb.Throws, tt.wantThrows)
			}
		})
	}
}

func TestParseFunctionTypeSyntaxDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "trailing comma",
			src:  "func apply(cb: fn(Int,) -> Int) -> Int:\n    return 0\n",
			want: "function type parameter list does not allow a trailing comma",
		},
		{
			name: "missing arrow",
			src:  "func apply(cb: fn(Int) Int) -> Int:\n    return 0\n",
			want: "expected ->, got identifier",
		},
		{
			name: "semantic clauses unsupported in function type",
			src:  "func apply(cb: fn(Int) -> Int nothrow) -> Int:\n    return 0\n",
			want: "semantic clauses are not allowed in function types",
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

func TestPlan250ParserReleaseDeclarationFormsPositive(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "capsule and globals",
			src: `module plan.frontend
pub import core.math as math

capsule App:
    id: "tetra://plan/frontend"
    version: "1.0.0"

pub const answer: Int = 42
property title: String = "Plan"

func main() -> Int:
    return answer
`,
		},
		{
			name: "types protocols and impls",
			src: `module plan.protocols

pub struct Vec2:
    x: Int

pub enum Mode:
    case fast
    case slow

pub protocol Drawable:
    func draw(self: Vec2) -> Int

pub extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Drawable
`,
		},
		{
			name: "actor and test declarations",
			src: `module plan.actors

actor Worker:
    var count: Int = 0
    func run() -> Int:
        return count

test "worker":
    expect 1 == 1
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseFile([]byte(tt.src), tt.name+".tetra"); err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
		})
	}
}

func TestPlan250ParserInvalidDeclarationDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "module after declaration",
			src: `func main() -> Int:
    return 0
module late.name
`,
			want: "module must appear before declarations",
		},
		{
			name: "import after capsule",
			src: `capsule App:
    id: "tetra://app"
import core.math as math
`,
			want: "import must appear before declarations",
		},
		{
			name: "pub impl rejected",
			src: `struct Vec2:
    x: Int
pub impl Vec2: Drawable
`,
			want: "pub cannot apply to impl declarations",
		},
		{
			name: "duplicate capsule metadata",
			src: `capsule App:
    id: "tetra://app"
    id: "tetra://other"
`,
			want: "duplicate capsule metadata key 'id'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFile([]byte(tt.src), tt.name+".tetra")
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestPlan250ParserCapsuleMetadataShapeDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		file string
		line int
		col  int
		want string
	}{
		{
			name: "duplicate dotted key",
			src: `capsule App:
    flags.enabled: true
    flags.enabled: false
`,
			file: "capsule_duplicate.tetra",
			line: 3,
			col:  5,
			want: "duplicate capsule metadata key 'flags.enabled'",
		},
		{
			name: "missing metadata value",
			src: `capsule App:
    target:
`,
			file: "capsule_shape.tetra",
			line: 3,
			col:  1,
			want: "expected indented block after ':'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFile([]byte(tt.src), tt.file)
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			diag, ok := DiagnosticForError(err)
			if !ok {
				t.Fatalf("expected structured diagnostic: %T %v", err, err)
			}
			if diag.File != tt.file || diag.Line != tt.line || diag.Column != tt.col || diag.Message != tt.want {
				t.Fatalf("diagnostic = %#v, want %s:%d:%d %q", diag, tt.file, tt.line, tt.col, tt.want)
			}
		})
	}
}

func TestPlan250ParserIndentationAndElseIfDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		line   int
		column int
		want   string
	}{
		{
			name: "comment then eof after block header",
			src: `func main() -> Int:
    if true:
        // placeholder
`,
			line:   4,
			column: 1,
			want:   "expected indented block after ':'",
		},
		{
			name: "dedented else-if body",
			src: `func main() -> Int:
    if false:
        return 0
    else if true:
    return 1
`,
			line:   5,
			column: 1,
			want:   "expected indented block after ':'",
		},
		{
			name: "malformed else-if condition",
			src: `func main() -> Int:
    if false:
        return 0
    else if:
        return 1
`,
			line:   4,
			column: 15,
			want:   "expected expression, got {",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFile([]byte(tt.src), tt.name+".tetra")
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			diag, ok := DiagnosticForError(err)
			if !ok {
				t.Fatalf("expected structured diagnostic: %T %v", err, err)
			}
			if diag.Line != tt.line || diag.Column != tt.column || diag.Message != tt.want {
				t.Fatalf("diagnostic = %#v, want %d:%d %q", diag, tt.line, tt.column, tt.want)
			}
		})
	}
}

func TestPlan250ParserCallableUnsupportedAndPlannedDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "named closure literal rejected",
			src: `func main() -> Int:
    let f: fn(Int) -> Int = fn add1(x: Int) -> Int:
        return x + 1
    return f(0)
`,
			want: "closure literals cannot have names; use top-level closure declarations for named callables",
		},
		{
			name: "class planned",
			src:  "class Box:\n    value: Int\n",
			want: "planned feature 'class declarations' is not implemented in the Tetra v1.0 profile",
		},
		{
			name: "trait planned",
			src:  "trait Renderable:\n    func draw() -> Int\n",
			want: "planned feature 'trait declarations' is not implemented in the Tetra v1.0 profile",
		},
		{
			name: "interface planned",
			src:  "interface Renderable:\n    func draw() -> Int\n",
			want: "planned feature 'interface declarations' is not implemented in the Tetra v1.0 profile",
		},
		{
			name: "typealias planned",
			src:  "typealias UserID = Int\n",
			want: "planned feature 'type alias declarations' is not implemented in the Tetra v1.0 profile",
		},
		{
			name: "macro planned",
			src:  "macro trace:\n    return 0\n",
			want: "planned feature 'macro declarations' is not implemented in the Tetra v1.0 profile",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFile([]byte(tt.src), tt.name+".tetra")
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestParserSourceSpanPrecision(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		line   int
		column int
		msg    string
	}{
		{
			name:   "nested block expression",
			src:    "func main() -> Int:\n    if true:\n        return @\n    return 0\n",
			line:   3,
			column: 16,
			msg:    "expected expression, got ?",
		},
		{
			name:   "crlf eof block",
			src:    "func main() -> Int:\r\n    if true:\r\n",
			line:   3,
			column: 1,
			msg:    "expected indented block after ':'",
		},
		{
			name:   "unicode string keeps following column",
			src:    "func main() -> Int:\n    print(\"Привіт\")\n    return @\n",
			line:   3,
			column: 12,
			msg:    "expected expression, got ?",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFile([]byte(tt.src), tt.name+".tetra")
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			diag, ok := DiagnosticForError(err)
			if !ok {
				t.Fatalf("expected structured diagnostic: %T %v", err, err)
			}
			if diag.Line != tt.line || diag.Column != tt.column || diag.Message != tt.msg {
				t.Fatalf("diagnostic = %#v, want %d:%d %q", diag, tt.line, tt.column, tt.msg)
			}
		})
	}
}

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
