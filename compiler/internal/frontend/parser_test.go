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

func TestParseFieldIndexAssignmentTargetShape(t *testing.T) {
	prog, err := Parse([]byte(`
func main() -> Int:
    xs.ptr[0] = 1
    return 0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 1 || len(prog.Funcs[0].Body) < 1 {
		t.Fatalf("missing parsed function body: %#v", prog)
	}
	assign, ok := prog.Funcs[0].Body[0].(*AssignStmt)
	if !ok {
		t.Fatalf("first stmt = %T, want *AssignStmt", prog.Funcs[0].Body[0])
	}
	index, ok := assign.Target.(*IndexExpr)
	if !ok {
		t.Fatalf("assign target = %T, want *IndexExpr", assign.Target)
	}
	field, ok := index.Base.(*FieldAccessExpr)
	if !ok {
		t.Fatalf("index base = %T, want *FieldAccessExpr", index.Base)
	}
	if field.Field != "ptr" {
		t.Fatalf("field = %q, want ptr", field.Field)
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

func TestParseCapsuleDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"capsule empty block", "capsule App {}", "capsule requires at least one metadata entry"},
		{"capsule malformed key-value", "capsule App {\n    id:\n}\n", "expected expression, got }"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFile([]byte(tt.src), tt.name+".tetra")
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

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
