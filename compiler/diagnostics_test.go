package compiler

import (
	"encoding/json"
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

func TestDiagnosticFromPlannedFeatureParserError(t *testing.T) {
	_, err := ParseFile([]byte("actor Counter:\n"), "ui/view.tetra")
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
	if !strings.Contains(diag.Message, "planned feature 'actor'") {
		t.Fatalf("message = %q", diag.Message)
	}
	if !strings.Contains(err.Error(), "ui/view.tetra:1:1: planned feature 'actor'") {
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
	if diag.Severity != "error" || diag.Line != 3 || diag.Column != 1 {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if !strings.Contains(diag.Message, "uses effect 'io'") {
		t.Fatalf("message = %q", diag.Message)
	}
}
