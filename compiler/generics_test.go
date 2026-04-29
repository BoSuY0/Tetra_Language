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

func TestGenericFunctionProtocolBoundConformancePasses(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: Echoable

func id<T: Echoable>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.FuncSigs["id__T_Vec2"]; !ok {
		t.Fatalf("missing protocol-bound monomorphized signature: %#v", checked.FuncSigs)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericFunctionProtocolBoundRejectsMissingImpl(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

func id<T: Echoable>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected protocol-bound conformance diagnostic")
	}
	if !strings.Contains(err.Error(), "generic argument 'Vec2' does not satisfy bound 'Echoable' for 'T'") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionProtocolBoundRejectsMismatchedImplSignature(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Int:
        return self.x

impl Vec2: Echoable

func id<T: Echoable>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected protocol-bound conformance diagnostic")
	}
	if !strings.Contains(err.Error(), "return type differs") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericStructSameModuleMonomorphizedHappyPath(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func main() -> Int:
    let b: Box<Int> = Box<Int>{value: 42}
    return b.value
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.Types["Box__T_i32"]; !ok {
		t.Fatalf("missing monomorphized struct type: %#v", checked.Types)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericFunctionReturningGenericStructMonomorphizesStruct(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func make<T>(x: T) -> Box<T>:
    return Box<T>{value: x}

func main() -> Int:
    let b: Box<Int> = make(42)
    return b.value
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.Types["Box__T_i32"]; !ok {
		t.Fatalf("missing monomorphized struct type: %#v", checked.Types)
	}
	sig, ok := checked.FuncSigs["make__T_i32"]
	if !ok {
		t.Fatalf("missing monomorphized function signature: %#v", checked.FuncSigs)
	}
	if sig.ReturnType != "Box__T_i32" {
		t.Fatalf("make__T_i32 return type = %q, want Box__T_i32", sig.ReturnType)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericStructRejectsMissingTypeArgs(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func main() -> Int:
    let b: Box = Box<Int>{value: 42}
    return b.value
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected missing type argument diagnostic")
	}
	if !strings.Contains(err.Error(), "generic struct 'Box' requires 1 type argument") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericStructRejectsInvalidArity(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func main() -> Int:
    let b: Box<Int, Bool> = Box<Int>{value: 42}
    return b.value
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected invalid arity diagnostic")
	}
	if !strings.Contains(err.Error(), "generic struct 'Box' expects 1 type argument, got 2") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionInfersOptionalParameterElement(t *testing.T) {
	src := []byte(`
func unwrap<T>(value: T?) -> T:
    if let x = value:
        return x
    else:
        return 0

func main() -> Int:
    let value: Int? = 42
    return unwrap(value)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.FuncSigs["unwrap__T_i32"]; !ok {
		t.Fatalf("missing optional monomorphized signature: %#v", checked.FuncSigs)
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

func TestGenericFunctionRejectsAmbiguousReturnOnlyInference(t *testing.T) {
	src := []byte(`
func zero<T>() -> T:
    return 0

func main() -> Int:
    return zero()
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected generic ambiguity diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot infer generic argument 'T'") {
		t.Fatalf("error = %v", err)
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

func TestGenericStructCrossModuleMonomorphizedHappyPath(t *testing.T) {
	files := map[string]string{
		"engine/box.tetra": `module engine.box
pub struct Box<T>:
    value: T
`,
		"app/main.tetra": `module app.main
import engine.box as box

func main() -> Int:
    let b: box.Box<Int> = box.Box<Int>{value: 42}
    return b.value
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
	if _, ok := checked.Types["engine.box.Box__T_i32"]; !ok {
		t.Fatalf("missing cross-module monomorphized struct type: %#v", checked.Types)
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
