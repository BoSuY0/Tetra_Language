package compiler

import (
	"strings"
	"testing"
)

func TestNormalizeFlowForMigrationAPI(t *testing.T) {
	src := []byte("func main() -> Int:\n    let x: Int = 1\n    if x > 0:\n        return x\n")
	got, err := NormalizeFlowForMigration(src, "main.tetra")
	if err != nil {
		t.Fatalf("NormalizeFlowForMigration: %v", err)
	}
	out := string(got)
	for _, want := range []string{"fun main() -> Int {", "val x: Int = 1", "if (x > 0) {"} {
		if !strings.Contains(out, want) {
			t.Fatalf("normalized output missing %q:\n%s", want, out)
		}
	}
}

func TestNormalizeFlowForMigrationAPICoversFlowTestAndMatchSurface(t *testing.T) {
	src := []byte(`func main() -> Int:
    let maybeValue: Int? = none
    if let value = maybeValue:
        return value
    else:
        match maybeValue:
        case none:
            return 0
        case some(x):
            return x
`)
	got, err := NormalizeFlowForMigration(src, "qa/main.tetra")
	if err != nil {
		t.Fatalf("NormalizeFlowForMigration: %v", err)
	}
	out := string(got)
	for _, want := range []string{
		"fun main() -> Int {",
		"val maybeValue: Int? = none",
		"if let value = maybeValue {",
		"case some(x) {",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("normalized output missing %q:\n%s", want, out)
		}
	}
}

func TestNormalizeFlowForMigrationAPITabDiagnostic(t *testing.T) {
	_, err := NormalizeFlowForMigration([]byte("func main() -> Int:\n\treturn 0\n"), "qa/tabbed.tetra")
	if err == nil {
		t.Fatalf("expected diagnostic")
	}
	diag := DiagnosticFromError(err)
	if diag.Code != DiagnosticCodeParse || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	if diag.File != "qa/tabbed.tetra" || diag.Line != 2 || diag.Column != 1 {
		t.Fatalf("position = %q:%d:%d, want qa/tabbed.tetra:2:1", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "tabs are not supported in Flow indentation" {
		t.Fatalf("message = %q", diag.Message)
	}
}
