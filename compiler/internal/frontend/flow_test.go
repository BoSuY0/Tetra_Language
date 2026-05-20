package frontend

import (
	"strings"
	"testing"
)

func TestNormalizeFlowForMigrationRewritesCompatibilitySurface(t *testing.T) {
	src := []byte(`func main() -> Int:
    let x: Int = 1
    if x > 0:
        return x
    return 0
`)
	got, err := NormalizeFlowForMigration(src, "main.tetra")
	if err != nil {
		t.Fatalf("NormalizeFlowForMigration: %v", err)
	}
	out := string(got)
	for _, want := range []string{
		"fun main() -> Int {",
		"val x: Int = 1",
		"if (x > 0) {",
		"return x",
		"return 0",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("normalized output missing %q:\n%s", want, out)
		}
	}
}

func TestNormalizeFlowForMigrationHandlesIfLetAndMatchCases(t *testing.T) {
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
	got, err := NormalizeFlowForMigration(src, "main.tetra")
	if err != nil {
		t.Fatalf("NormalizeFlowForMigration: %v", err)
	}
	out := string(got)
	for _, want := range []string{
		"fun main() -> Int {",
		"val maybeValue: Int? = none",
		"if let value = maybeValue {",
		"match maybeValue {",
		"case none {",
		"case some(x) {",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("normalized output missing %q:\n%s", want, out)
		}
	}
}

func TestNormalizeFlowForMigrationTabDiagnostic(t *testing.T) {
	_, err := NormalizeFlowForMigration([]byte("func main() -> Int:\n\treturn 0\n"), "tabbed/main.tetra")
	if err == nil {
		t.Fatalf("expected diagnostic")
	}
	diag, ok := DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected structured diagnostic: %T %v", err, err)
	}
	if diag.File != "tabbed/main.tetra" || diag.Line != 2 || diag.Column != 1 {
		t.Fatalf("position = %q:%d:%d, want tabbed/main.tetra:2:1", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "tabs are not supported in Flow indentation" {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestNormalizeFlowForMigrationSpanCRLFUnicode(t *testing.T) {
	src := []byte("func main() -> Int:\r\n    let msg: String = \"Привіт\"\r\n    if msg == \"ok\":")
	_, err := NormalizeFlowForMigration(src, "span/main.tetra")
	if err == nil {
		t.Fatalf("expected diagnostic")
	}
	diag, ok := DiagnosticForError(err)
	if !ok {
		t.Fatalf("expected structured diagnostic: %T %v", err, err)
	}
	if diag.File != "span/main.tetra" || diag.Line != 3 || diag.Column != 1 {
		t.Fatalf("position = %q:%d:%d, want span/main.tetra:3:1", diag.File, diag.Line, diag.Column)
	}
	if diag.Message != "expected indented block after ':'" {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestParseFileWithMigrationNormalizationParsesAndPreservesRawSource(t *testing.T) {
	src := []byte(`func main() -> Int:
    let x: Int = 1
    if x > 0:
        return x
    return 0
`)

	file, err := ParseFileWithMigrationNormalization(src, "migration/main.tetra")
	if err != nil {
		t.Fatalf("ParseFileWithMigrationNormalization: %v", err)
	}
	if file.Path != "migration/main.tetra" || string(file.Src) != string(src) {
		t.Fatalf("file metadata = path %q src %q, want raw migration input", file.Path, string(file.Src))
	}
	if len(file.Funcs) != 1 || file.Funcs[0].Name != "main" {
		t.Fatalf("funcs = %#v, want one main func", file.Funcs)
	}
	if len(file.Funcs[0].Body) != 3 {
		t.Fatalf("main body length = %d, want 3", len(file.Funcs[0].Body))
	}
	if _, ok := file.Funcs[0].Body[1].(*IfStmt); !ok {
		t.Fatalf("body[1] = %T, want *IfStmt after migration normalization", file.Funcs[0].Body[1])
	}
}
