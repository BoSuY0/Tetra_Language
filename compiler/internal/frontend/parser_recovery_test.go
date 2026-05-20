package frontend

import "testing"

func TestParseFileDiagnosticsRecoversIndependentTopLevelFlowDeclarations(t *testing.T) {
	src := []byte(`func badLet() -> Int:
    let value: = 1
    return value

func keepFirst() -> Int:
    return 42

func badReturn() -> Int:
    return

func keepSecond() -> Int:
    return 7
`)

	file, diagnostics := ParseFileDiagnostics(src, "multi_recovery.tetra")
	if file == nil {
		t.Fatalf("file = nil, want partial AST")
	}
	if len(diagnostics) != 2 {
		t.Fatalf("diagnostics = %#v, want 2", diagnostics)
	}

	want := []struct {
		line    int
		column  int
		message string
	}{
		{line: 2, column: 16, message: "expected identifier, got ="},
		{line: 9, column: 5, message: "expected expression, got }"},
	}
	for i, wantDiag := range want {
		diag := diagnostics[i]
		if diag.Code != DiagnosticCodeParse || diag.Severity != "error" || diag.File != "multi_recovery.tetra" {
			t.Fatalf("diagnostic[%d] identity = %#v", i, diag)
		}
		if diag.Line != wantDiag.line || diag.Column != wantDiag.column || diag.Message != wantDiag.message {
			t.Fatalf("diagnostic[%d] = %#v, want %d:%d %q", i, diag, wantDiag.line, wantDiag.column, wantDiag.message)
		}
	}

	if len(file.Funcs) != 2 {
		t.Fatalf("funcs = %#v, want two recovered valid functions", file.Funcs)
	}
	if file.Funcs[0].Name != "keepFirst" || file.Funcs[1].Name != "keepSecond" {
		t.Fatalf("func names = %q, %q; want keepFirst, keepSecond", file.Funcs[0].Name, file.Funcs[1].Name)
	}
}

func TestParseFileDiagnosticsRecoversIndependentTopLevelNonFunctionDeclarations(t *testing.T) {
	src := []byte(`enum Broken:
    case

struct Keep:
    value: Int

property broken: = "bad"

func keep() -> Int:
    return 9
`)

	file, diagnostics := ParseFileDiagnostics(src, "non_function_recovery.tetra")
	if file == nil {
		t.Fatalf("file = nil, want partial AST")
	}
	if len(diagnostics) != 2 {
		t.Fatalf("diagnostics = %#v, want 2", diagnostics)
	}
	for i, diag := range diagnostics {
		if diag.Code != DiagnosticCodeParse || diag.Severity != "error" || diag.File != "non_function_recovery.tetra" {
			t.Fatalf("diagnostic[%d] identity = %#v", i, diag)
		}
	}

	if len(file.Structs) != 1 || file.Structs[0].Name != "Keep" {
		t.Fatalf("structs = %#v, want one recovered Keep struct", file.Structs)
	}
	if len(file.Funcs) != 1 || file.Funcs[0].Name != "keep" {
		t.Fatalf("funcs = %#v, want one recovered keep func", file.Funcs)
	}
	if len(file.Enums) != 0 || len(file.Globals) != 0 {
		t.Fatalf("recovered invalid declarations: enums=%#v globals=%#v", file.Enums, file.Globals)
	}
}
