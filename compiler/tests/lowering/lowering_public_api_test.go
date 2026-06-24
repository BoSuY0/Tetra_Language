package compiler_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/frontend"
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	if irProg.MainName != "main" || irProg.MainIndex < 0 || irProg.MainIndex >= len(irProg.Funcs) {
		t.Fatalf(
			"invalid main metadata: name=%q index=%d funcs=%d",
			irProg.MainName,
			irProg.MainIndex,
			len(irProg.Funcs),
		)
	}
	moduleName := checked.Funcs[0].Module
	modFuncs, err := compiler.LowerModule(checked, moduleName)
	if err != nil {
		t.Fatalf("LowerModule: %v", err)
	}
	if len(modFuncs) != len(irProg.Funcs) {
		t.Fatalf("LowerModule funcs = %d, want %d", len(modFuncs), len(irProg.Funcs))
	}
	modules, err := compiler.LowerModules(checked)
	if err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
	if len(modules[moduleName]) != len(irProg.Funcs) {
		t.Fatalf(
			"LowerModules[%s] funcs = %d, want %d",
			moduleName,
			len(modules[moduleName]),
			len(irProg.Funcs),
		)
	}
	for _, fn := range irProg.Funcs {
		if fn.Name == "" || len(fn.Instrs) == 0 {
			t.Fatalf("invalid lowered function: %#v", fn)
		}
		if fn.ParamSlots < 0 || fn.LocalSlots < fn.ParamSlots || fn.ReturnSlots < 0 {
			t.Fatalf(
				"invalid slot metadata for %s: params=%d locals=%d returns=%d",
				fn.Name,
				fn.ParamSlots,
				fn.LocalSlots,
				fn.ReturnSlots,
			)
		}
	}
}

func TestLowerModulesMatchesCanonicalMemoryPipeline(t *testing.T) {
	src := []byte(`
func copied_len(xs: []u8) -> Int
uses alloc, mem:
    let copied: []u8 = xs.copy()
    return copied.len

func main() -> Int
uses alloc, mem:
    var xs: []u8 = core.make_u8(4)
    xs[0] = 7
    return copied_len(xs) + xs[0]
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	canonical, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	modules, err := compiler.LowerModules(checked)
	if err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
	moduleName := checked.Funcs[0].Module
	got := modules[moduleName]
	if !reflect.DeepEqual(got, canonical.Funcs) {
		t.Fatalf("LowerModules[%q] diverged from canonical Lower output\nmodules=%#v\ncanonical=%#v", moduleName, got, canonical.Funcs)
	}
}

func TestPublicCodegenRejectsInvalidIRBeforeBackend(t *testing.T) {
	backends := []struct {
		name    string
		codegen func([]compiler.IRFunc) (*compiler.Object, error)
	}{
		{name: "linux-x64", codegen: compiler.CodegenObjectLinuxX64},
		{name: "windows-x64", codegen: compiler.CodegenObjectWindowsX64},
		{name: "macos-x64", codegen: compiler.CodegenObjectMacOSX64},
	}
	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			_, err := backend.codegen([]compiler.IRFunc{
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
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != compiler.DiagnosticCodeIRVerifier {
				t.Fatalf("diagnostic = %#v", diag)
			}
			if diag.Severity != "error" || diag.Hint == "" || diag.Message == "" {
				t.Fatalf("incomplete verifier diagnostic = %#v", diag)
			}
		})
	}
}

func TestPublicIRVerifierRejectsProgramAndFunctionDriftWithStableDiagnostic(t *testing.T) {
	programErr := compiler.VerifyIRProgram(&compiler.IRProgram{
		MainIndex: 0,
		MainName:  "main",
		Funcs: []compiler.IRFunc{
			{Name: "not_main", Instrs: []ir.IRInstr{{Kind: ir.IRReturn}}},
		},
	})
	if programErr == nil {
		t.Fatalf("expected program verifier error")
	}
	programDiag := compiler.DiagnosticFromError(programErr)
	if programDiag.Code != compiler.DiagnosticCodeIRVerifier || programDiag.Severity != "error" ||
		programDiag.Hint == "" {
		t.Fatalf("program diagnostic = %#v", programDiag)
	}

	pos := frontend.Position{File: "api_bad_ir.t4", Line: 12, Col: 7}
	funcErr := compiler.VerifyIRFunc(compiler.IRFunc{
		Name: "bad_branch",
		Instrs: []ir.IRInstr{
			{Kind: ir.IRJmpIfZero, Label: 1, Pos: pos},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRReturn},
		},
	})
	if funcErr == nil {
		t.Fatalf("expected function verifier error")
	}
	funcDiag := compiler.DiagnosticFromError(funcErr)
	if funcDiag.Code != compiler.DiagnosticCodeIRVerifier || funcDiag.File != "api_bad_ir.t4" ||
		funcDiag.Line != 12 ||
		funcDiag.Column != 7 {
		t.Fatalf("function diagnostic = %#v", funcDiag)
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

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	funcs, err := compiler.LowerModule(checked, "lib.callbacks")
	if err != nil {
		t.Fatalf("LowerModule(lib.callbacks): %v", err)
	}
	if len(funcs) == 0 {
		t.Fatalf("LowerModule(lib.callbacks) returned no functions")
	}
}

func writeTestFiles(t *testing.T, base string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}
}
