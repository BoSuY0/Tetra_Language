package compiler

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestGenericFunctionParseCheckAndDocs(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := prog.Funcs[0].TypeParams; len(got) != 1 || got[0] != "T" {
		t.Fatalf("type params = %#v", got)
	}
	if _, err := Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
	docs, err := GenerateAPIDocsFromSource(src, "generics.tetra")
	if err != nil {
		t.Fatalf("GenerateAPIDocsFromSource: %v", err)
	}
	if !strings.Contains(string(docs), "`func id<T>(x: T) -> T`") {
		t.Fatalf("docs = %s", string(docs))
	}
}

func TestGenericFunctionMonomorphizedCall(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return id(42)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.FuncSigs["id__T_i32"]; !ok {
		t.Fatalf("missing monomorphized signature: %#v", checked.FuncSigs)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericFunctionUnsupportedArgDiagnostic(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return id(unknown)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected generic inference diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot infer generic argument") {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), "v0.5") {
		t.Fatalf("generic diagnostic should be versionless: %v", err)
	}
}

func TestGenericFunctionCrossModuleMonomorphizedCall(t *testing.T) {
	files := map[string]string{
		"engine/util.tetra": `module engine.util
func id<T>(x: T) -> T:
    return x
`,
		"app/main.tetra": `module app.main
import engine.util as util

func main() -> Int:
    return util.id(42)
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
	if _, ok := checked.FuncSigs["engine.util.id__T_i32"]; !ok {
		t.Fatalf("missing cross-module monomorphized signature: %#v", checked.FuncSigs)
	}
	if _, err := LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestGenericFunctionMonomorphizedNamesAvoidTypeCollisions(t *testing.T) {
	files := map[string]string{
		"a.tetra": `module a
struct b_c:
    x: Int
`,
		"a_b.tetra": `module a_b
struct c:
    y: Int
`,
		"util/gen.tetra": `module util.gen
func id<T>(x: T) -> T:
    return x
`,
		"app/main.tetra": `module app.main
import util.gen as util
import a as a
import a_b as ab

func main() -> Int:
    let first: a.b_c = a.b_c{x: 1}
    let second: ab.c = ab.c{y: 2}
    let firstOut: a.b_c = util.id(first)
    let secondOut: ab.c = util.id(second)
    let x: Int = firstOut.x
    let y: Int = secondOut.y
    return x + y
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
	var names []string
	for name := range checked.FuncSigs {
		if strings.HasPrefix(name, "util.gen.id__") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) != 2 {
		t.Fatalf("monomorphized util.id variants = %v, want 2 distinct variants", names)
	}
	if names[0] == names[1] {
		t.Fatalf("colliding monomorphized names: %v", names)
	}
	if _, err := LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}
