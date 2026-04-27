package compiler

import (
	"path/filepath"
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

func TestExtensionRejectsDuplicateMethodName(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x
    func sum(self: Vec2) -> Int:
        return self.x

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected duplicate extension method error")
	}
	if !strings.Contains(err.Error(), "duplicate function 'Vec2.sum'") {
		t.Fatalf("error = %v", err)
	}
}

func TestImportedExtensionStaticCallAndDocsSurface(t *testing.T) {
	files := map[string]string{
		"engine/vec.tetra": `module engine.vec
struct Vec2:
    x: Int
    y: Int

extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x + self.y
`,
		"app/main.tetra": `module app.main
import engine.vec as vec

func main() -> Int:
    let v: vec.Vec2 = vec.Vec2(x: 40, y: 2)
    return vec.Vec2.sum(v)
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, ok := checked.FuncSigs["engine.vec.Vec2.sum"]; !ok {
		t.Fatalf("missing imported extension method signature: %#v", checked.FuncSigs)
	}
	if _, err := LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}

	docs, err := GenerateAPIDocs([]string{filepath.Join(tmp, filepath.FromSlash("engine/vec.tetra"))})
	if err != nil {
		t.Fatalf("GenerateAPIDocs: %v", err)
	}
	out := string(docs)
	for _, want := range []string{
		"### Extensions",
		"- `Vec2`",
		"`func Vec2.sum(self: Vec2) -> i32`",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("docs missing %q:\n%s", want, out)
		}
	}
}
