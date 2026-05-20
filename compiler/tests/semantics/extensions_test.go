package compiler_test

import (
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Extensions) != 1 {
		t.Fatalf("extensions = %d", len(prog.Extensions))
	}
	if len(prog.Funcs) != 2 || prog.Funcs[0].Name != "Vec2.sum" {
		t.Fatalf("funcs = %#v", prog.Funcs)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.FuncSigs["Vec2.sum"]; !ok {
		t.Fatalf("missing extension method signature")
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestExtensionMethodCanReturnOptionalPayload(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

extension Vec2:
    func nonzero(self: Vec2) -> Int?:
        if self.x == 0:
            return none
        return self.x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let maybe: Int? = Vec2.nonzero(v)
    if let x = maybe:
        return x
    else:
        return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["Vec2.nonzero"].ReturnType; got != "i32?" {
		t.Fatalf("Vec2.nonzero return type = %q, want i32?", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestExtensionNoLongerPlannedDiagnostic(t *testing.T) {
	_, err := compiler.Parse([]byte("extension Vec2:\n"))
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
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

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, ok := checked.FuncSigs["engine.vec.Vec2.sum"]; !ok {
		t.Fatalf("missing imported extension method signature: %#v", checked.FuncSigs)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}

	docs, err := compiler.GenerateAPIDocs([]string{filepath.Join(tmp, filepath.FromSlash("engine/vec.tetra"))})
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
