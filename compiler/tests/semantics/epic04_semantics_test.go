package compiler_test

import (
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func TestEpic04SemanticCheckerCorePositiveCase(t *testing.T) {
	prog, err := compiler.Parse([]byte(`
fun main(): i32 {
  let x: i32 = 41
  return x + 1
}
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if checked.MainName != "main" {
		t.Fatalf("main name = %q, want main", checked.MainName)
	}
	if len(checked.Funcs) != 1 {
		t.Fatalf("func count = %d, want 1", len(checked.Funcs))
	}
}

func TestEpic04SemanticCheckerCoreNegativePositionedDiagnostic(t *testing.T) {
	err := testkit.CheckProgram(`
fun main(): i32 {
  let x: i32 = true
  return 0
}
`)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "line 3:3: type mismatch: expected 'i32', got 'bool'") {
		t.Fatalf("error = %v", err)
	}
}

func TestEpic04SemanticCheckerCoreCrossModuleParity(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/math.tetra": "module engine.math\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/main.tetra":    "module app.main\nimport engine.math as math\nfun main(): i32 {\n  return math.add_one(41)\n}\n",
	})

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checkedWorld, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if checkedWorld.MainName != "app.main.main" {
		t.Fatalf("main name = %q, want app.main.main", checkedWorld.MainName)
	}

	singleProg, err := compiler.Parse([]byte(`
fun add_one(x: i32): i32 {
  return x + 1
}
fun main(): i32 {
  return add_one(41)
}
`))
	if err != nil {
		t.Fatalf("single parse: %v", err)
	}
	checkedSingle, err := compiler.Check(singleProg)
	if err != nil {
		t.Fatalf("single check: %v", err)
	}
	if checkedSingle.FuncSigs["add_one"].ReturnType != checkedWorld.FuncSigs["engine.math.add_one"].ReturnType {
		t.Fatalf("return types diverged between single-file and module-world checks")
	}
}

func TestEpic04SemanticCheckerCoreDisplayTextStability(t *testing.T) {
	err := testkit.CheckProgram(`
fun main(): bool {
  return true
}
`)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "main must return i32") {
		t.Fatalf("error = %v", err)
	}
}

func TestEpic04SemanticCheckerCoreBoundaryNilProgram(t *testing.T) {
	_, err := compiler.Check(nil)
	if err == nil {
		t.Fatalf("expected nil program error")
	}
	if err.Error() != "no program provided" {
		t.Fatalf("error = %q, want no program provided", err.Error())
	}
}

func TestEpic04ExpressionTypingPositiveAndInferenceCrossModule(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/math.tetra": "module engine.math\nfun inc(x: i32): i32 {\n  return x + 1\n}\n",
		"app/main.tetra":    "module app.main\nimport engine.math as math\nfun main(): i32 {\n  let v = math.inc(1)\n  return v\n}\n",
	})

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	mainIdx := -1
	for i := range checked.Funcs {
		if checked.Funcs[i].Name == "app.main.main" {
			mainIdx = i
			break
		}
	}
	if mainIdx < 0 {
		t.Fatalf("missing app.main.main")
	}
	if got := checked.Funcs[mainIdx].Locals["v"].TypeName; got != "i32" {
		t.Fatalf("local v type = %q, want i32", got)
	}
}

func TestEpic04ExpressionTypingNegativeDiagnostic(t *testing.T) {
	err := testkit.CheckProgram(`
fun main(): i32 {
  let ok: bool = true
  let x: i32 = ok + 1
  return x
}
`)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "arithmetic operators require i32/u8") {
		t.Fatalf("error = %v", err)
	}
}

func TestEpic04ExpressionTypingDisplayTextAndBoundary(t *testing.T) {
	err := testkit.CheckProgram(`
fun main(): i32 {
  let value = none
  return 0
}
`)
	if err == nil {
		t.Fatalf("expected inference boundary error")
	}
	if !strings.Contains(err.Error(), "cannot infer type from 'none'; add an optional type annotation") {
		t.Fatalf("error = %v", err)
	}
}

func TestEpic04TypeModelPositiveOptionalSlots(t *testing.T) {
	prog, err := compiler.Parse([]byte(`
struct Box:
    value: Int?

func main() -> Int:
    let box: Box = Box(value: none)
    return 0
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	opt := checked.Types["i32?"]
	if opt == nil {
		t.Fatalf("missing optional i32 type")
	}
	if opt.SlotCount != 2 {
		t.Fatalf("optional slot count = %d, want 2", opt.SlotCount)
	}
}

func TestEpic04TypeModelNegativeArrayBoundary(t *testing.T) {
	err := testkit.CheckProgram(`
func main() -> Int:
    let xs: [0]Int = 0
    return 0
`)
	if err == nil {
		t.Fatalf("expected array boundary error")
	}
	if !strings.Contains(err.Error(), "array size must be positive constant") {
		t.Fatalf("error = %v", err)
	}
}

func TestEpic04TypeModelCrossModuleAndDisplayText(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/types.tetra": "module engine.types\nstruct Vec { x: i32, y: i32 }\n",
		"app/main.tetra":     "module app.main\nimport engine.types as t\nfun consume(v: t.Vec): i32 {\n  return v.x + v.y\n}\nfun main(): i32 {\n  return 0\n}\n",
	})
	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if got := checked.FuncSigs["app.main.consume"].ParamTypes[0]; got != "engine.types.Vec" {
		t.Fatalf("consume param = %q, want engine.types.Vec", got)
	}

	err = testkit.CheckProgram(`
func main() -> Int:
    let b: Byte = true
    return 0
`)
	if err == nil {
		t.Fatalf("expected type mismatch")
	}
	if !strings.Contains(err.Error(), "expected 'u8', got 'bool'") {
		t.Fatalf("error = %v", err)
	}
}

func TestEpic04LocalInferenceNegativeAndDisplayText(t *testing.T) {
	err := testkit.CheckProgram(`
fun main(): i32 {
  let x = missing(1)
  return x
}
`)
	if err == nil {
		t.Fatalf("expected unknown function inference error")
	}
	if !strings.Contains(err.Error(), "cannot infer type for 'x': unknown function 'missing'") {
		t.Fatalf("error = %v", err)
	}
}
