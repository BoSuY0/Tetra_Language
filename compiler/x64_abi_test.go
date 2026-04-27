package compiler

import (
	"runtime"
	"testing"

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

func TestX64CodegenObjectsCarryTargetMetadata(t *testing.T) {
	funcs := []IRFunc{{
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
		build  func([]IRFunc) (*Object, error)
	}{
		{name: "linux", target: "linux-x64", build: CodegenObjectLinuxX64},
		{name: "macos", target: "macos-x64", build: CodegenObjectMacOSX64},
		{name: "windows", target: "windows-x64", build: CodegenObjectWindowsX64},
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
