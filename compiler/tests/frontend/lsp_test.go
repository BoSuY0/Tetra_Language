package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestAnalyzeLSPSourceSymbolsAndHovers(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int
    y: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Renderable

const answer: Int = 42

func add(v: borrow Vec2) -> Int
uses mem, io:
    return v.x + v.y
`)
	got := compiler.AnalyzeLSPSource(src, "vec.tetra")
	if len(got.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", got.Diagnostics)
	}
	if len(got.Symbols) != 6 {
		t.Fatalf("symbols = %#v", got.Symbols)
	}
	wantNames := []string{"Vec2", "Renderable", "answer", "Vec2: Renderable", "add", "Vec2.draw"}
	for i, want := range wantNames {
		if got.Symbols[i].Name != want {
			t.Fatalf("symbols = %#v, want %q at %d", got.Symbols, want, i)
		}
	}
	if got.Symbols[2].Kind != "const" || got.Symbols[3].Kind != "impl" {
		t.Fatalf("symbols = %#v", got.Symbols)
	}
	if got.Symbols[4].Detail != "func add(v: borrow Vec2) -> i32 uses io, mem" {
		t.Fatalf("function detail = %q", got.Symbols[4].Detail)
	}
	if got.Symbols[5].Kind != "extension-method" || got.Symbols[5].Detail != "func Vec2.draw(self: Vec2) -> i32" {
		t.Fatalf("extension method symbol = %#v", got.Symbols[5])
	}
	if len(got.Hovers) < 6 {
		t.Fatalf("hovers = %#v", got.Hovers)
	}
	for _, hover := range got.Hovers {
		if hover.Name == "answer" && hover.Contents == "const answer: i32" {
			return
		}
	}
	t.Fatalf("missing const hover: %#v", got.Hovers)
}

func TestAnalyzeLSPSourceActorDeclarationDiagnostic(t *testing.T) {
	got := compiler.AnalyzeLSPSource([]byte("actor P:\n    count: Int\n"), "bad.tetra")
	if len(got.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v", got.Diagnostics)
	}
	if got.Diagnostics[0].File != "bad.tetra" || got.Diagnostics[0].Line != 2 {
		t.Fatalf("diagnostic = %#v", got.Diagnostics[0])
	}
	if !strings.Contains(got.Diagnostics[0].Message, "actor state fields must use 'val' or 'const'") {
		t.Fatalf("diagnostic = %#v", got.Diagnostics[0])
	}
}

func TestAnalyzeLSPSourceSemanticDiagnostics(t *testing.T) {
	got := compiler.AnalyzeLSPSource([]byte(`func main() -> Int:
    print("missing uses\n")
    return 0
`), "bad_semantic.tetra")
	if len(got.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v", got.Diagnostics)
	}
	diag := got.Diagnostics[0]
	if diag.Code != compiler.DiagnosticCodeSafetyEffect || diag.File != "bad_semantic.tetra" || diag.Line != 2 {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if diag.Message != "function 'main' uses effect 'io' but does not declare it" {
		t.Fatalf("message = %q", diag.Message)
	}
	if len(got.Symbols) != 1 || got.Symbols[0].Name != "main" {
		t.Fatalf("symbols should still be available after semantic diagnostic: %#v", got.Symbols)
	}
}

func TestAnalyzeLSPSourcePrivacyConsentDiagnosticCode(t *testing.T) {
	got := compiler.AnalyzeLSPSource([]byte(`func seal(token: consent.token) -> secret.i32
uses privacy:
    return core.secret_seal_i32(1, token)
`), "bad_privacy_semantic.tetra")
	if len(got.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v", got.Diagnostics)
	}
	diag := got.Diagnostics[0]
	if diag.Code != compiler.DiagnosticCodeSafetyPrivacy || diag.File != "bad_privacy_semantic.tetra" || diag.Line != 1 {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if diag.Message != "uses effect 'privacy' requires semantic clause 'privacy'" {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestAnalyzeLSPSourceRecursiveSecretSignatureDiagnosticCode(t *testing.T) {
	got := compiler.AnalyzeLSPSource([]byte(`func seal(payload: secret.i32?) -> Int:
    return 0
`), "bad_secret_signature.tetra")
	if len(got.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v", got.Diagnostics)
	}
	diag := got.Diagnostics[0]
	if diag.Code != compiler.DiagnosticCodeSafetyPrivacy || diag.File != "bad_secret_signature.tetra" || diag.Line != 1 {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if diag.Message != "secret types in function signature require semantic clause 'privacy'" {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestAnalyzeLSPSourceImportedFileDoesNotReportMissingWorkspace(t *testing.T) {
	got := compiler.AnalyzeLSPSource([]byte(`import lib.core.math as math

func answer() -> Int:
    return math.add_i32(40, 2)
`), "module_with_import.tetra")
	if len(got.Diagnostics) != 0 {
		t.Fatalf("single-file LSP should not report unresolved imports without a workspace graph: %#v", got.Diagnostics)
	}
	if len(got.Symbols) != 1 || got.Symbols[0].Name != "answer" {
		t.Fatalf("symbols = %#v", got.Symbols)
	}
}

func TestAnalyzeLSPFileChecksImportedModuleGraph(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(appDir, "main.tetra")
	helperPath := filepath.Join(appDir, "helper.tetra")
	if err := os.WriteFile(helperPath, []byte(`module app.helper

func value() -> Int:
    return 42
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainPath, []byte(`module app.main
import app.helper as helper

func answer() -> Int:
    return helper.missing()
`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := compiler.AnalyzeLSPFile(mainPath)
	if err != nil {
		t.Fatalf("AnalyzeLSPFile: %v", err)
	}
	if len(got.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v", got.Diagnostics)
	}
	diag := got.Diagnostics[0]
	if diag.Code != "TETRA2001" || !strings.Contains(diag.Message, "unknown function 'app.helper.missing'") {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if len(got.Symbols) != 1 || got.Symbols[0].Name != "answer" {
		t.Fatalf("symbols should remain from entry file: %#v", got.Symbols)
	}
}
