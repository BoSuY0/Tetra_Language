package compiler_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"

	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/backend/x64obj"
	"tetra_language/compiler/internal/ir"
)

func TestX64ABICallsZeroThroughTenArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
fun f0(): i32 { return 1 }
fun f1(a: i32): i32 { return a }
fun f2(a: i32, b: i32): i32 { return a + b }
fun f3(a: i32, b: i32, c: i32): i32 { return a + b + c }
fun f4(a: i32, b: i32, c: i32, d: i32): i32 { return a + b + c + d }
fun f5(a: i32, b: i32, c: i32, d: i32, e: i32): i32 { return a + b + c + d + e }
fun f6(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32): i32 { return a + b + c + d + e + f }
fun f7(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32): i32 { return a + b + c + d + e + f + g }
fun f8(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32): i32 { return a + b + c + d + e + f + g + h }
fun f9(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32, i: i32): i32 { return a + b + c + d + e + f + g + h + i }
fun f10(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32, i: i32, j: i32): i32 { return a + b + c + d + e + f + g + h + i + j }

fun main(): i32 {
  return f0() + f1(1) + f2(1, 2) + f3(1, 2, 3) + f4(1, 2, 3, 4) + f5(1, 2, 3, 4, 5) + f6(1, 2, 3, 4, 5, 6) + f7(1, 2, 3, 4, 5, 6, 7) + f8(1, 2, 3, 4, 5, 6, 7, 8) + f9(1, 2, 3, 4, 5, 6, 7, 8, 9) + f10(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
}
`

	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 221 {
		t.Fatalf("exit code mismatch: got %d want 221", exitCode)
	}
}

func buildAndRun(t *testing.T, src string) (string, int) {
	t.Helper()
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	return testkit.RunBinary(t, outPath)
}

func TestX64ABIReturnSlotsThreeAndFourRegisterMapping(t *testing.T) {
	cases := []struct {
		name string
		abi  x64abi.ABI
	}{
		{name: "sysv", abi: x64abi.LinuxSysV()},
		{name: "win64", abi: x64abi.NewWin64()},
	}
	returnLayouts := []struct {
		slots int
		regs  []string
	}{
		{slots: 3, regs: []string{"rax", "rdx", "r8"}},
		{slots: 4, regs: []string{"rax", "rdx", "r8", "r9"}},
	}

	for _, tc := range cases {
		for _, layout := range returnLayouts {
			t.Run(tc.name+"/"+x64ReturnSlotName(layout.slots), func(t *testing.T) {
				e := &x64.Emitter{}
				stackDepth := 0
				var callPatches []x64obj.CallPatch
				err := tc.abi.EmitCall(e, ir.IRInstr{
					Kind:     ir.IRCall,
					Name:     "callee",
					ArgSlots: 0,
					RetSlots: layout.slots,
				}, &stackDepth, &callPatches)
				if err != nil {
					t.Fatalf("EmitCall: %v", err)
				}
				if stackDepth != layout.slots {
					t.Fatalf("stack depth = %d, want %d", stackDepth, layout.slots)
				}

				wantSuffix := &x64.Emitter{}
				emitX64ReturnSlotPushes(wantSuffix, layout.regs)
				if !bytes.HasSuffix(e.Buf, wantSuffix.Buf) {
					t.Fatalf("return-slot mapping mismatch for registers %v\n got=% x\nwant suffix=% x", layout.regs, e.Buf, wantSuffix.Buf)
				}
			})
		}
	}
}

func TestX64CodegenObjectsCarryTargetMetadata(t *testing.T) {
	funcs := []compiler.IRFunc{{
		Name:        "main",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}}

	cases := []struct {
		name   string
		target string
		build  func([]compiler.IRFunc) (*compiler.Object, error)
	}{
		{name: "linux", target: "linux-x64", build: compiler.CodegenObjectLinuxX64},
		{name: "macos", target: "macos-x64", build: compiler.CodegenObjectMacOSX64},
		{name: "windows", target: "windows-x64", build: compiler.CodegenObjectWindowsX64},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obj, err := tc.build(funcs)
			if err != nil {
				t.Fatalf("codegen: %v", err)
			}
			if obj.Target != tc.target {
				t.Fatalf("object target = %q, want %q", obj.Target, tc.target)
			}
		})
	}
}

func x64ReturnSlotName(n int) string {
	return "return_slots_" + string(rune('0'+n))
}

func emitX64ReturnSlotPushes(e *x64.Emitter, regs []string) {
	for _, reg := range regs {
		switch reg {
		case "rax":
			e.PushRax()
		case "rdx":
			e.PushRdx()
		case "r8":
			e.PushR8()
		case "r9":
			e.PushR9()
		default:
			panic("unknown return register: " + reg)
		}
	}
}

func TestX64CodegenObjectRelocKindsByPlatformABI(t *testing.T) {
	funcs := []compiler.IRFunc{{
		Name:        "main",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRStrLit, Str: []byte("hi")},
			{Kind: ir.IRWrite},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}}

	linuxObj, err := compiler.CodegenObjectLinuxX64(funcs)
	if err != nil {
		t.Fatalf("linux codegen: %v", err)
	}
	macosObj, err := compiler.CodegenObjectMacOSX64(funcs)
	if err != nil {
		t.Fatalf("macos codegen: %v", err)
	}
	windowsObj, err := compiler.CodegenObjectWindowsX64(funcs)
	if err != nil {
		t.Fatalf("windows codegen: %v", err)
	}

	hasKind := func(obj *compiler.Object, kind compiler.RelocKind) bool {
		for _, reloc := range obj.Relocs {
			if reloc.Kind == kind {
				return true
			}
		}
		return false
	}

	if hasKind(linuxObj, compiler.RelocIATDisp32) || hasKind(macosObj, compiler.RelocIATDisp32) {
		t.Fatalf("SysV objects must not carry IAT relocations")
	}
	if !hasKind(windowsObj, compiler.RelocIATDisp32) {
		t.Fatalf("Win64 object should carry IAT relocations")
	}
	if !hasKind(linuxObj, compiler.RelocDataDisp32) || !hasKind(macosObj, compiler.RelocDataDisp32) || !hasKind(windowsObj, compiler.RelocDataDisp32) {
		t.Fatalf("all native x64 objects should carry data relocations for string literal")
	}
}

func TestX64BuildOnlySmokeAcrossNativeTargets(t *testing.T) {
	tmp := t.TempDir()
	srcPath := testkit.RepoPath(t, "examples", "hello.tetra")
	targets := []struct {
		target string
		suffix string
	}{
		{target: "linux-x64", suffix: ""},
		{target: "macos-x64", suffix: ""},
		{target: "windows-x64", suffix: ".exe"},
	}

	for _, tc := range targets {
		t.Run(tc.target, func(t *testing.T) {
			outPath := filepath.Join(tmp, tc.target+tc.suffix)
			if err := compiler.BuildFile(srcPath, outPath, tc.target); err != nil {
				t.Fatalf("build %s: %v", tc.target, err)
			}
			if _, err := os.Stat(outPath); err != nil {
				t.Fatalf("missing output for %s: %v", tc.target, err)
			}
		})
	}
}
