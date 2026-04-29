package compiler

import (
	"path/filepath"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestLowerPublicAPIVerifiesRepresentativeIR(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    var total: Int = 0
    for i in 0..<2:
        total = total + i
    return core.task_join_i32(task) + total
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	if irProg.MainName != "main" || irProg.MainIndex < 0 || irProg.MainIndex >= len(irProg.Funcs) {
		t.Fatalf("invalid main metadata: name=%q index=%d funcs=%d", irProg.MainName, irProg.MainIndex, len(irProg.Funcs))
	}
	moduleName := checked.Funcs[0].Module
	modFuncs, err := LowerModule(checked, moduleName)
	if err != nil {
		t.Fatalf("LowerModule: %v", err)
	}
	if len(modFuncs) != len(irProg.Funcs) {
		t.Fatalf("LowerModule funcs = %d, want %d", len(modFuncs), len(irProg.Funcs))
	}
	modules, err := LowerModules(checked)
	if err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
	if len(modules[moduleName]) != len(irProg.Funcs) {
		t.Fatalf("LowerModules[%s] funcs = %d, want %d", moduleName, len(modules[moduleName]), len(irProg.Funcs))
	}
	for _, fn := range irProg.Funcs {
		if fn.Name == "" || len(fn.Instrs) == 0 {
			t.Fatalf("invalid lowered function: %#v", fn)
		}
		if fn.ParamSlots < 0 || fn.LocalSlots < fn.ParamSlots || fn.ReturnSlots < 0 {
			t.Fatalf("invalid slot metadata for %s: params=%d locals=%d returns=%d", fn.Name, fn.ParamSlots, fn.LocalSlots, fn.ReturnSlots)
		}
	}
}

func TestPublicCodegenRejectsInvalidIRBeforeBackend(t *testing.T) {
	_, err := CodegenObjectLinuxX64([]IRFunc{
		{
			Name:        "main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRReturn},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected verifier error")
	}
	diag := DiagnosticFromError(err)
	if diag.Code != DiagnosticCodeIRVerifier {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestLowerModuleCallableFunctionTypedCrossModulePath(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return callbacks.apply(f, 41)
`,
	})

	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	funcs, err := LowerModule(checked, "lib.callbacks")
	if err != nil {
		t.Fatalf("LowerModule(lib.callbacks): %v", err)
	}
	if len(funcs) == 0 {
		t.Fatalf("LowerModule(lib.callbacks) returned no functions")
	}
}
