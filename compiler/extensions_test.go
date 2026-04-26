package compiler

import (
	"strings"
	"testing"
)

func TestExtensionParseCheckAndLower(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int
    y: Int

extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x + self.y

func main() -> Int:
    let v: Vec2 = Vec2(x: 40, y: 2)
    return Vec2.sum(v)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Extensions) != 1 {
		t.Fatalf("extensions = %d", len(prog.Extensions))
	}
	if len(prog.Funcs) != 2 || prog.Funcs[0].Name != "Vec2.sum" {
		t.Fatalf("funcs = %#v", prog.Funcs)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.FuncSigs["Vec2.sum"]; !ok {
		t.Fatalf("missing extension method signature")
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestExtensionNoLongerPlannedDiagnostic(t *testing.T) {
	_, err := Parse([]byte("extension Vec2:\n"))
	if err == nil {
		t.Fatalf("expected block error, not silent success")
	}
	if strings.Contains(err.Error(), "planned feature 'extension'") {
		t.Fatalf("extension still reports planned diagnostic: %v", err)
	}
}
