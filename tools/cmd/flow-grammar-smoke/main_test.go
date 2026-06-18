package main

import (
	"strings"
	"testing"

	"tetra_language/compiler"
)

func TestGenerateSmokeSourceCoversSupportedFamilies(t *testing.T) {
	src := generateSmokeSource()
	for _, want := range []string{
		"module generated.flow_grammar_smoke",
		"enum SmokeColor",
		"struct Pair",
		"func id<T>",
		"protocol Drawable",
		"extension Pair",
		"func answer() -> Int = 42",
		"async func worker",
		"state CounterState",
		"view CounterView",
		"test \"generated grammar smoke\"",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("generated source missing %q:\n%s", want, src)
		}
	}
}

func TestGenerateSmokeSourceParsesWithCompilerFrontend(t *testing.T) {
	src := generateSmokeSource()
	if _, err := compiler.ParseFile([]byte(src), "generated_smoke.tetra"); err != nil {
		t.Fatalf("ParseFile(generated_smoke.tetra): %v", err)
	}
}

func TestGenerateSmokeSourceMalformedTestDeclDiagnostic(t *testing.T) {
	src := generateSmokeSource()
	bad := "test generated grammar smoke:\n    expect 1 == 1\n\n" + src
	_, err := compiler.ParseFile([]byte(bad), "generated_bad.tetra")
	if err == nil {
		t.Fatalf("expected diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeParse || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	if diag.File != "generated_bad.tetra" || diag.Line != 1 || diag.Column != 6 {
		t.Fatalf(
			"position = %q:%d:%d, want generated_bad.tetra:1:6",
			diag.File,
			diag.Line,
			diag.Column,
		)
	}
	if diag.Message != "expected string, got identifier" {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestGenerateSmokeSourceSpanCRLFTabAndUnicode(t *testing.T) {
	base := generateSmokeSource()
	src := base + "test \"Привіт\":\r\n\texpect 1 == 1\r\n"
	expectedLine := strings.Count(base, "\n") + 2
	_, err := compiler.ParseFile([]byte(src), "generated_span.tetra")
	if err == nil {
		t.Fatalf("expected diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.File != "generated_span.tetra" || diag.Line != expectedLine || diag.Column != 1 {
		t.Fatalf(
			"position = %q:%d:%d, want generated_span.tetra:%d:1",
			diag.File,
			diag.Line,
			diag.Column,
			expectedLine,
		)
	}
	if diag.Message != "tabs are not supported in Flow indentation" {
		t.Fatalf("message = %q", diag.Message)
	}
}
