package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestProtocolParseCheckAndDocs(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Protocols) != 1 {
		t.Fatalf("protocols = %d", len(prog.Protocols))
	}
	if got := prog.Protocols[0].Requirements[0].Name; got != "draw" {
		t.Fatalf("requirement name = %q", got)
	}
	if _, err := compiler.Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
	docs, err := compiler.GenerateAPIDocsFromSource(src, "protocols.tetra")
	if err != nil {
		t.Fatalf("GenerateAPIDocsFromSource: %v", err)
	}
	if !strings.Contains(string(docs), "`protocol Renderable`") || !strings.Contains(string(docs), "`func draw(self: Vec2) -> i32`") {
		t.Fatalf("docs = %s", string(docs))
	}
}

func TestProtocolNoLongerPlannedDiagnostic(t *testing.T) {
	_, err := compiler.Parse([]byte("protocol P:\n"))
	if err == nil {
		t.Fatalf("expected block error, not silent success")
	}
	if strings.Contains(err.Error(), "planned feature 'protocol'") {
		t.Fatalf("protocol still reports planned diagnostic: %v", err)
	}
}
