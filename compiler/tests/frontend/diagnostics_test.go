package compiler_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestDiagnosticFromFlowIndentationErrorJSONReady(t *testing.T) {
	_, err := compiler.ParseFile([]byte("func main() -> i32:\nreturn 0\n"), "app/main.tetra")
	if err == nil {
		t.Fatalf("expected parse error")
	}

	diag := compiler.DiagnosticFromError(err)
	if diag.Code != "TETRA0001" {
		t.Fatalf("code = %q, want TETRA0001", diag.Code)
	}
	if diag.Severity != "error" {
		t.Fatalf("severity = %q, want error", diag.Severity)
	}
	if diag.File != "app/main.tetra" || diag.Line != 2 || diag.Column != 1 {
		t.Fatalf("position = %q:%d:%d, want app/main.tetra:2:1", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "expected indented block after ':'" {
		t.Fatalf("message = %q", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"file":"app/main.tetra"`) {
		t.Fatalf("json = %s", raw)
	}
}

func TestDiagnosticFromCapsuleParserError(t *testing.T) {
	_, err := compiler.ParseFile([]byte("capsule Counter {}"), "ui/view.tetra")
	if err == nil {
		t.Fatalf("expected parse error")
	}

	diag := compiler.DiagnosticFromError(err)
	if diag.Code != "TETRA0001" || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	if diag.File != "ui/view.tetra" || diag.Line != 1 || diag.Column != 1 {
		t.Fatalf("position = %q:%d:%d, want ui/view.tetra:1:1", diag.File, diag.Line, diag.Column)
	}
	if !strings.Contains(diag.Message, "capsule requires at least one metadata entry") {
		t.Fatalf("message = %q", diag.Message)
	}
	if !strings.Contains(err.Error(), "ui/view.tetra:1:1: capsule requires at least one metadata entry") {
		t.Fatalf("text diagnostic changed unexpectedly: %q", err.Error())
	}
}

func TestDiagnosticFromParserErrorLineColumnConsistency(t *testing.T) {
	src := []byte("fun main() -> i32 {\n  return 1 == 2 == 3\n}\n")
	_, err := compiler.ParseFile(src, "app/math.tetra")
	if err == nil {
		t.Fatalf("expected parse error")
	}

	diag := compiler.DiagnosticFromError(err)
	if diag.File != "app/math.tetra" || diag.Line != 2 || diag.Column != 17 {
		t.Fatalf("position = %q:%d:%d, want app/math.tetra:2:17", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "chaining equality operators is not supported" {
		t.Fatalf("message = %q", diag.Message)
	}
	if got := err.Error(); !strings.HasPrefix(got, "app/math.tetra:2:17: ") {
		t.Fatalf("text position = %q", got)
	}
}

func TestDiagnosticFromSafetyEffectErrorUsesEffectCode(t *testing.T) {
	err := checkDiagnosticProgram(t, `
func main() -> Int:
    print("missing uses\n")
    return 0
`)
	if err == nil {
		t.Fatalf("expected semantic diagnostic")
	}

	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyEffect {
		t.Fatalf("code = %q, want %s", diag.Code, compiler.DiagnosticCodeSafetyEffect)
	}
	if diag.Severity != "error" || diag.Line != 3 || diag.Column != 5 {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if !strings.Contains(diag.Message, "uses effect 'io'") {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestDiagnosticFromRecursiveSecretSignatureUsesPrivacyCode(t *testing.T) {
	err := checkDiagnosticProgram(t, `
func seal(payload: secret.i32?) -> Int:
    return 0
`)
	if err == nil {
		t.Fatalf("expected semantic diagnostic")
	}

	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSafetyPrivacy {
		t.Fatalf("code = %q, want %s", diag.Code, compiler.DiagnosticCodeSafetyPrivacy)
	}
	if diag.Severity != "error" || diag.Line == 0 || diag.Column == 0 {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if diag.Message != "secret types in function signature require semantic clause 'privacy'" {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestDiagnosticFromPositionedSemanticErrorStripsTextPrefix(t *testing.T) {
	err := checkDiagnosticProgram(t, `
func main() -> Int:
    let x: Int = true
    return x
`)
	if err == nil {
		t.Fatalf("expected semantic diagnostic")
	}

	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSemantic || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	if diag.File != "" || diag.Line != 3 || diag.Column != 5 {
		t.Fatalf("position = %q:%d:%d, want :3:5", diag.File, diag.Line, diag.Column)
	}
	if strings.Contains(diag.Message, ":3:5:") {
		t.Fatalf("semantic diagnostic message retained text prefix: %q", diag.Message)
	}
	if !strings.Contains(diag.Message, "type mismatch: expected 'i32', got 'bool'") {
		t.Fatalf("message = %q", diag.Message)
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, want := range []string{`"code":"TETRA2001"`, `"line":3`, `"column":5`, `"severity":"error"`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("json = %s, missing %s", raw, want)
		}
	}
}

func TestDiagnosticFromPositionedSemanticErrorWithFile(t *testing.T) {
	tmp := t.TempDir()
	mainPath := filepath.Join(tmp, "main.t4")
	writeDiagnosticTestFiles(t, tmp, map[string]string{
		"main.t4": "func main() -> Int:\n    let x: Int = true\n    return x\n",
	})
	world, err := compiler.LoadWorld(mainPath)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected semantic diagnostic")
	}

	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSemantic || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	if !strings.HasSuffix(diag.File, "main.t4") || diag.Line != 2 || diag.Column != 5 {
		t.Fatalf("position = %q:%d:%d, want main.t4:2:5", diag.File, diag.Line, diag.Column)
	}
	if strings.Contains(diag.Message, "main.t4:2:5:") {
		t.Fatalf("semantic diagnostic message retained text prefix: %q", diag.Message)
	}
	if !strings.Contains(diag.Message, "type mismatch: expected 'i32', got 'bool'") {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestDiagnosticFromInvalidUTF8ParserError(t *testing.T) {
	_, err := compiler.ParseFile([]byte{'f', 'n', ' ', 0xff, '\n'}, "bad/utf8.tetra")
	if err == nil {
		t.Fatalf("expected invalid UTF-8 diagnostic")
	}

	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeParse || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	if diag.File != "bad/utf8.tetra" || diag.Line != 1 || diag.Column != 4 {
		t.Fatalf("position = %q:%d:%d, want bad/utf8.tetra:1:4", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "invalid UTF-8 encoding" {
		t.Fatalf("message = %q", diag.Message)
	}
	if diag.Hint == "" {
		t.Fatalf("expected invalid UTF-8 hint")
	}
	raw, err := json.Marshal(diag)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, want := range []string{
		`"code":"TETRA0001"`,
		`"severity":"error"`,
		`"line":1`,
		`"column":4`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("json = %s, missing %s", raw, want)
		}
	}
}

func TestDiagnosticFromFlowTabIndentationError(t *testing.T) {
	_, err := compiler.ParseFile([]byte("func main() -> Int:\n\treturn 0\n"), "app/tabbed.tetra")
	if err == nil {
		t.Fatalf("expected parse error")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeParse || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	if diag.File != "app/tabbed.tetra" || diag.Line != 2 || diag.Column != 1 {
		t.Fatalf("position = %q:%d:%d, want app/tabbed.tetra:2:1", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "tabs are not supported in Flow indentation" {
		t.Fatalf("message = %q", diag.Message)
	}
	if diag.Hint != "Replace tabs with spaces in Flow-indented blocks." {
		t.Fatalf("hint = %q", diag.Hint)
	}
}

func TestDiagnosticFromMalformedFlowTestDeclaration(t *testing.T) {
	_, err := compiler.ParseFile([]byte("test math:\n    expect 1 == 1\n"), "qa/bad_test_decl.tetra")
	if err == nil {
		t.Fatalf("expected parse error")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeParse || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	if diag.File != "qa/bad_test_decl.tetra" || diag.Line != 1 || diag.Column != 6 {
		t.Fatalf("position = %q:%d:%d, want qa/bad_test_decl.tetra:1:6", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "expected string, got identifier" {
		t.Fatalf("message = %q", diag.Message)
	}
	if !strings.Contains(diag.Hint, "nearby syntax") {
		t.Fatalf("hint = %q, want nearby syntax guidance", diag.Hint)
	}
}

func TestDiagnosticFromFlowTestSpanCRLFUnicode(t *testing.T) {
	src := []byte("test \"Привіт\":\r\n    expect @\r\n")
	_, err := compiler.ParseFile(src, "qa/span_unicode.tetra")
	if err == nil {
		t.Fatalf("expected parse error")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.File != "qa/span_unicode.tetra" || diag.Line != 2 || diag.Column != 8 {
		t.Fatalf("position = %q:%d:%d, want qa/span_unicode.tetra:2:8", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "expected expression, got ?" {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestDiagnosticHintsForCommonParserFailures(t *testing.T) {
	tests := []struct {
		name string
		src  []byte
		want string
	}{
		{"capsule empty block", []byte("capsule Counter {}"), "Add at least one metadata entry"},
		{"indentation", []byte("func main() -> Int:\nreturn 0\n"), "Indent the block"},
		{"tabs", []byte("func main() -> Int:\n\treturn 0\n"), "Replace tabs with spaces"},
		{"unexpected token", []byte("func main() -> Int:\n    return @\n"), "nearby syntax"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compiler.ParseFile(tt.src, tt.name+".tetra")
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if !strings.Contains(diag.Hint, tt.want) {
				t.Fatalf("hint = %q, want contains %q", diag.Hint, tt.want)
			}
		})
	}
}

func TestParserDiagnosticFixtureMatrixJSONReady(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		line    int
		column  int
		message string
	}{
		{
			name: "invalid indentation in flow block",
			src: `func main() -> Int:
return 0
`,
			line:    2,
			column:  1,
			message: "expected indented block after ':'",
		},
		{
			name: "malformed function type missing parameter type",
			src: `func apply(cb: fn(, Int) -> Int) -> Int:
    return 0
`,
			line:    1,
			column:  19,
			message: "expected identifier, got ,",
		},
		{
			name: "malformed function type missing return arrow",
			src: `func apply(cb: fn(Int) Int) -> Int:
    return 0
`,
			line:    1,
			column:  24,
			message: "expected ->, got identifier",
		},
		{
			name: "malformed capsule empty block",
			src: `capsule App {}
`,
			line:    1,
			column:  1,
			message: "capsule requires at least one metadata entry",
		},
		{
			name: "malformed property missing name",
			src: `property : String = "title"

func main() -> Int:
    return 0
`,
			line:    1,
			column:  10,
			message: "expected identifier, got :",
		},
		{
			name: "malformed actor member",
			src: `actor Worker:
    let value: Int = 1
`,
			line:    2,
			column:  5,
			message: "actor declarations currently support state fields and func methods only",
		},
		{
			name: "malformed await operand",
			src: `async func work() -> Int:
    return 1

func main() -> Int:
    return await @
`,
			line:    5,
			column:  18,
			message: "expected expression, got ?",
		},
		{
			name: "unsupported character in expression",
			src: `func main() -> Int:
    return @
`,
			line:    2,
			column:  12,
			message: "expected expression, got ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compiler.ParseFile([]byte(tt.src), "fixtures/"+strings.ReplaceAll(tt.name, " ", "_")+".tetra")
			if err == nil {
				t.Fatalf("expected parser diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeParse || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v", diag)
			}
			if diag.Line != tt.line || diag.Column != tt.column {
				t.Fatalf("position = %d:%d, want %d:%d; err=%v", diag.Line, diag.Column, tt.line, tt.column, err)
			}
			if diag.Message != tt.message {
				t.Fatalf("message = %q, want %q", diag.Message, tt.message)
			}
			if diag.File == "" {
				t.Fatalf("expected diagnostic file")
			}
			raw, err := json.Marshal(diag)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			for _, want := range []string{`"code":"TETRA0001"`, `"severity":"error"`, `"message":`} {
				if !strings.Contains(string(raw), want) {
					t.Fatalf("json = %s, missing %s", raw, want)
				}
			}
		})
	}
}

func TestDiagnosticCodeRegistryListsPublicCodes(t *testing.T) {
	registry := compiler.DiagnosticCodeRegistry()
	for _, want := range []string{
		compiler.DiagnosticCodeParse,
		compiler.DiagnosticCodeSemantic,
		compiler.DiagnosticCodeSafetyOwnership,
		compiler.DiagnosticCodeSafetyLifetime,
		compiler.DiagnosticCodeSafetyEffect,
		compiler.DiagnosticCodeSafetyPrivacy,
		compiler.DiagnosticCodeSafetyBudget,
		compiler.DiagnosticCodeIRVerifier,
		compiler.DiagnosticCodeLowerUnsupported,
		compiler.DiagnosticCodeFormatter,
		compiler.DiagnosticCodeFormatterCheck,
	} {
		if _, ok := registry[want]; !ok {
			t.Fatalf("diagnostic registry missing %s: %#v", want, registry)
		}
	}
	if got := registry[compiler.DiagnosticCodeFormatterCheck].Severity; got != "error" {
		t.Fatalf("formatter check severity = %q, want error", got)
	}
}

func TestDiagnosticFromCrossModuleSemanticError(t *testing.T) {
	tmp := t.TempDir()
	writeDiagnosticTestFiles(t, tmp, map[string]string{
		"engine/math.tetra": "module engine.math\nfun inc(x: i32): i32 {\n  return x + 1\n}\n",
		"app/main.tetra":    "module app.main\nimport engine.math as math\nfun math(): i32 {\n  return 1\n}\nfun main(): i32 {\n  return math()\n}\n",
	})

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected semantic alias conflict error")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSemantic || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	if !strings.HasSuffix(diag.File, filepath.FromSlash("app/main.tetra")) {
		t.Fatalf("diagnostic file = %q, want app/main.tetra suffix", diag.File)
	}
	if !strings.Contains(diag.Message, "import alias 'math' conflicts with declaration 'math'") {
		t.Fatalf("diagnostic message = %q", diag.Message)
	}
}

func TestDiagnosticFromUnstructuredErrorBoundary(t *testing.T) {
	diag := compiler.DiagnosticFromError(errors.New("plain failure"))
	if diag.Code != compiler.DiagnosticCodeParse {
		t.Fatalf("code = %q, want %q", diag.Code, compiler.DiagnosticCodeParse)
	}
	if diag.Message != "plain failure" {
		t.Fatalf("message = %q", diag.Message)
	}
	if diag.File != "" || diag.Line != 0 || diag.Column != 0 {
		t.Fatalf("position should be empty for unstructured error: %#v", diag)
	}
}

func TestDiagnosticFromNilErrorBoundary(t *testing.T) {
	diag := compiler.DiagnosticFromError(nil)
	if diag != (compiler.Diagnostic{}) {
		t.Fatalf("compiler.DiagnosticFromError(nil) = %#v, want zero-value diagnostic", diag)
	}
}

func checkDiagnosticProgram(t *testing.T, src string) error {
	t.Helper()

	prog, err := compiler.Parse([]byte(src))
	if err != nil {
		return err
	}
	_, err = compiler.Check(prog)
	return err
}

func writeDiagnosticTestFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()

	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}
}
