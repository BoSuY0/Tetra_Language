package compiler

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagnosticFromFlowIndentationErrorJSONReady(t *testing.T) {
	_, err := ParseFile([]byte("func main() -> i32:\nreturn 0\n"), "app/main.tetra")
	if err == nil {
		t.Fatalf("expected parse error")
	}

	diag := DiagnosticFromError(err)
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
	_, err := ParseFile([]byte("capsule Counter {}"), "ui/view.tetra")
	if err == nil {
		t.Fatalf("expected parse error")
	}

	diag := DiagnosticFromError(err)
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
	_, err := ParseFile(src, "app/math.tetra")
	if err == nil {
		t.Fatalf("expected parse error")
	}

	diag := DiagnosticFromError(err)
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

func TestDiagnosticFromSemanticErrorUsesSemanticCode(t *testing.T) {
	err := checkProgram(`
func main() -> Int:
    print("missing uses\n")
    return 0
`)
	if err == nil {
		t.Fatalf("expected semantic diagnostic")
	}

	diag := DiagnosticFromError(err)
	if diag.Code != "TETRA2001" {
		t.Fatalf("code = %q, want TETRA2001", diag.Code)
	}
	if diag.Severity != "error" || diag.Line != 3 || diag.Column != 5 {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if !strings.Contains(diag.Message, "uses effect 'io'") {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestDiagnosticFromInvalidUTF8ParserError(t *testing.T) {
	_, err := ParseFile([]byte{'f', 'n', ' ', 0xff, '\n'}, "bad/utf8.tetra")
	if err == nil {
		t.Fatalf("expected invalid UTF-8 diagnostic")
	}

	diag := DiagnosticFromError(err)
	if diag.Code != DiagnosticCodeParse || diag.Severity != "error" {
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
	_, err := ParseFile([]byte("func main() -> Int:\n\treturn 0\n"), "app/tabbed.tetra")
	if err == nil {
		t.Fatalf("expected parse error")
	}
	diag := DiagnosticFromError(err)
	if diag.Code != DiagnosticCodeParse || diag.Severity != "error" {
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
	_, err := ParseFile([]byte("test math:\n    expect 1 == 1\n"), "qa/bad_test_decl.tetra")
	if err == nil {
		t.Fatalf("expected parse error")
	}
	diag := DiagnosticFromError(err)
	if diag.Code != DiagnosticCodeParse || diag.Severity != "error" {
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
	_, err := ParseFile(src, "qa/span_unicode.tetra")
	if err == nil {
		t.Fatalf("expected parse error")
	}
	diag := DiagnosticFromError(err)
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
			_, err := ParseFile(tt.src, tt.name+".tetra")
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			diag := DiagnosticFromError(err)
			if !strings.Contains(diag.Hint, tt.want) {
				t.Fatalf("hint = %q, want contains %q", diag.Hint, tt.want)
			}
		})
	}
}

func TestDiagnosticCodeRegistryListsPublicCodes(t *testing.T) {
	registry := DiagnosticCodeRegistry()
	for _, want := range []string{
		DiagnosticCodeParse,
		DiagnosticCodeSemantic,
		DiagnosticCodeIRVerifier,
		DiagnosticCodeLowerUnsupported,
		DiagnosticCodeFormatter,
		DiagnosticCodeFormatterCheck,
	} {
		if _, ok := registry[want]; !ok {
			t.Fatalf("diagnostic registry missing %s: %#v", want, registry)
		}
	}
	if got := registry[DiagnosticCodeFormatterCheck].Severity; got != "error" {
		t.Fatalf("formatter check severity = %q, want error", got)
	}
}

func TestDiagnosticFromCrossModuleSemanticError(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/math.tetra": "module engine.math\nfun inc(x: i32): i32 {\n  return x + 1\n}\n",
		"app/main.tetra":    "module app.main\nimport engine.math as math\nfun math(): i32 {\n  return 1\n}\nfun main(): i32 {\n  return math()\n}\n",
	})

	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = CheckWorld(world)
	if err == nil {
		t.Fatalf("expected semantic alias conflict error")
	}
	diag := DiagnosticFromError(err)
	if diag.Code != DiagnosticCodeSemantic || diag.Severity != "error" {
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
	diag := DiagnosticFromError(errors.New("plain failure"))
	if diag.Code != DiagnosticCodeParse {
		t.Fatalf("code = %q, want %q", diag.Code, DiagnosticCodeParse)
	}
	if diag.Message != "plain failure" {
		t.Fatalf("message = %q", diag.Message)
	}
	if diag.File != "" || diag.Line != 0 || diag.Column != 0 {
		t.Fatalf("position should be empty for unstructured error: %#v", diag)
	}
}

func TestDiagnosticFromNilErrorBoundary(t *testing.T) {
	diag := DiagnosticFromError(nil)
	if diag != (Diagnostic{}) {
		t.Fatalf("DiagnosticFromError(nil) = %#v, want zero-value diagnostic", diag)
	}
}
