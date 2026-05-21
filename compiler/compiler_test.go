package compiler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func requireCheckFileErrorContains(t *testing.T, src string, want string) {
	t.Helper()
	testkit.RequireFileSemanticCheckErrorContains(t, src, want)
}

func requireCheckFileOK(t *testing.T, src string) {
	t.Helper()
	testkit.RequireFileSemanticCheckOK(t, src)
}

func TestBuildHello(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 uses io {\n  print(\"Hello from Tetra!\\n\");\n  return 0;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Hello from Tetra!\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildTwoPrints(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 uses io {\n  print(\"A\");\n  print(\"B\\n\");\n  return 0;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "AB\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStrLiteralValue(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 uses io {\n  val s: str = \"A\\n\"\n  print(s)\n  return 0\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "A\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStrParam(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun echo(x: str): i32 uses io {\n  print(x)\n  return 0\n}\nfun main(): i32 uses io {\n  return echo(\"Hi\\n\")\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Hi\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStrReturn(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun greet(): str {\n  return \"Hey\\n\"\n}\nfun main(): i32 uses io {\n  print(greet())\n  return 0\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Hey\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildExpressionBodiedFunction(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "func add(a: Int, b: Int) -> Int = a + b\nfunc main() -> Int = add(40, 2)\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMakeI32Slice(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 uses alloc, mem {\n  var xs: []i32 = make_i32(3)\n  xs[0] = 10\n  xs[1] = 20\n  xs[2] = xs[0] + xs[1]\n  return xs[2]\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 30 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMakeU8Print(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 uses alloc, io, mem {\n  var xs: []u8 = make_u8(2)\n  xs[0] = 65\n  xs[1] = 66\n  print(xs)\n  return 0\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "AB" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMmioSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 uses alloc, capability, io, mem, mmio {\n  var out: i32 = 0\n  unsafe {\n    let io: cap.io = core.cap_io()\n    let p: ptr = core.alloc_bytes(4)\n    let _w: i32 = core.mmio_write_i32(p, 123, io)\n    out = core.mmio_read_i32(p, io)\n  }\n  return out\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 123 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildIslandsDebugDoubleFreeRejectedBySemantics(t *testing.T) {
	requireCheckFileErrorContains(t, `
func alias(isl: island) -> island:
    return isl

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(64)
        let other: island = alias(isl)
        free(isl)
        free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestBuildScopedIslandAutoFreeRunsInDebugAndNonDebug(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 uses alloc, islands, mem {\n  var out: i32 = 0\n  island(64) as isl {\n    var xs: []u8 = core.island_make_u8(isl, 1)\n    xs[0] = 7\n    out = xs[0]\n  }\n  return out\n}\n"
	for _, tc := range []struct {
		name string
		opt  BuildOptions
	}{
		{name: "non_debug", opt: BuildOptions{Jobs: 1}},
		{name: "debug", opt: BuildOptions{Jobs: 1, IslandsDebug: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, exitCode := buildAndRunWithOptions(t, src, tc.opt)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != 7 {
				t.Fatalf("exit code mismatch: %d", exitCode)
			}
		})
	}
}

func TestBuildIslandsDebugOverflowFails(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 uses alloc, islands, mem {\n  island(64) as isl {\n    var xs: []u8 = core.island_make_u8(isl, 65)\n    xs[0] = 1\n  }\n  return 0\n}\n"
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Jobs: 1, IslandsDebug: true})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildSliceBoundsCheck(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 uses alloc, mem {\n  var xs: []i32 = make_i32(2)\n  xs[2] = 1\n  return 0\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildRawPtrAddNegativeOffsetBoundsDiagnostic(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let q: ptr = core.ptr_add(p, 0 - 1, mem)
        let _: UInt8 = core.store_u8(q, 7, mem)
        return 0
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", exitCode)
	}
}

func TestBuildRawPtrAddAllocationUpperBoundDiagnostic(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let q: ptr = core.ptr_add(p, 4, mem)
        let _: UInt8 = core.store_u8(q, 7, mem)
        return 0
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", exitCode)
	}
}

func TestBuildRawPtrAddDirectI32OffsetAccess(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let _: Int = core.store_i32(core.ptr_add(p, 4, mem), 42, mem)
        return core.load_i32(core.ptr_add(p, 4, mem), mem)
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildRawPtrAddDirectPtrOffsetAccess(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let stored: ptr = core.store_ptr(core.ptr_add(p, 8, mem), p, mem)
        let loaded: ptr = core.load_ptr(core.ptr_add(p, 8, mem), mem)
        return 42
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildRawAllocZeroSizeDiagnostic(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, mem:
    unsafe:
        let _: ptr = core.alloc_bytes(0)
        return 0
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", exitCode)
	}
}

func TestBuildRawStoreI32AllocationBaseWidthDiagnostic(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(3)
        let _: Int = core.store_i32(p, 123, mem)
        return 0
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", exitCode)
	}
}

func TestBuildRawStorePtrAllocationBaseWidthDiagnostic(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(7)
        let _: ptr = core.store_ptr(p, p, mem)
        return 0
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", exitCode)
	}
}

func TestBuildMemoryHelpersRejectNegativeLength(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_memory_negative_length_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", exitCode)
	}
}

func TestBuildNonZeroReturn(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 uses io {\n  print(\"Done\\n\");\n  return 7;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Done\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildLetExpr(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  let x: i32 = 2 + 3;\n  return x;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildIfElseReturn(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  if (0) { return 1; } else { return 2; }\n  return 3;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildWhileCounter(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  var n: i32 = 3;\n  var acc: i32 = 0;\n  while (n) {\n    acc = acc + 1;\n    n = n - 1;\n  }\n  return acc;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 3 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildLessThan(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  if (2 < 3) { return 1; }\n  return 0;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildEqEqFalse(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  if (2 == 3) { return 1; }\n  return 0;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildWhileLess(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  var i: i32 = 0;\n  while (i < 3) {\n    i = i + 1;\n  }\n  return i;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 3 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildNewStyleNoSemicolons(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 {\n  var x: i32 = 2 + 3\n  return x\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFlowSyntaxHelloWithAliases(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "func main() -> Int\nuses io:\n  let msg: String = \"Flow\\n\"\n  let ok: Bool = true\n  print(msg)\n  if ok:\n    return 0\n  else:\n    return 1\n  return 1\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Flow\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildBoolBranchSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "func main() -> Int:\n  let ok: Bool = true\n  if ok && (3 > 2):\n    return 42\n  return 1\n"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildForRangeSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "func main() -> Int:\n  var total: Int = 0\n  for i in 0..<11:\n    total = total + i\n  return total\n"
	_, code := buildAndRun(t, src)
	if code != 55 {
		t.Fatalf("exit code mismatch: got %d, want 55", code)
	}
}

func TestBuildEnumMatchSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "enum Color:\n  case red\n  case green\n  case blue\n\nfunc main() -> Int:\n  let color: Color = Color.green\n  match color:\n  case Color.red:\n    return 1\n  case Color.green:\n    return 42\n  case _:\n    return 0\n"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildEnumMatchExhaustiveNoDefaultSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "enum Color:\n  case red\n  case green\n\nfunc main() -> Int:\n  let color: Color = Color.green\n  match color:\n  case Color.red:\n    return 1\n  case Color.green:\n    return 42\n"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildEnumPayloadMatchSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "enum Result:\n  case ok(Int)\n  case err(Int, Int)\n  case empty\n\nfunc main() -> Int:\n  let result: Result = Result.ok(42)\n  match result:\n  case Result.ok(value):\n    return value\n  case Result.err(code, detail):\n    return code + detail\n  case Result.empty:\n    return 0\n"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildEnumPayloadMultiValueCaseSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "enum Result:\n  case ok(Int)\n  case err(Int, Int)\n\nfunc main() -> Int:\n  let result: Result = Result.err(40, 2)\n  match result:\n  case Result.ok(value):\n    return value\n  case Result.err(code, detail):\n    return code + detail\n"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildEnumPayloadNoPayloadCaseInWideEnumSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "enum Result:\n  case ok(Int)\n  case empty\n\nfunc main() -> Int:\n  let result: Result = Result.empty\n  match result:\n  case Result.ok(value):\n    return value\n  case Result.empty:\n    return 42\n"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildEnumPayloadActorMessageDataSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "enum CounterMsg:\n  case inc(Int)\n  case reset\n\nfunc handle(msg: CounterMsg) -> Int:\n  match msg:\n  case CounterMsg.inc(delta):\n    return delta\n  case CounterMsg.reset:\n    return 0\n\nfunc main() -> Int:\n  let msg: CounterMsg = CounterMsg.inc(42)\n  return handle(msg)\n"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildMatchExpressionEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "enum Result:\n  case ok(Int)\n  case err(Int)\n\nfunc main() -> Int:\n  let result: Result = Result.ok(42)\n  let score: Int = match result:\n  case Result.ok(value):\n    value\n  case Result.err(code):\n    code\n  return score\n"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestMatchExpressionRequiresExhaustiveCases(t *testing.T) {
	src := "enum Result:\n  case ok(Int)\n  case err(Int)\n\nfunc main() -> Int:\n  let result: Result = Result.ok(42)\n  let score: Int = match result:\n  case Result.ok(value):\n    value\n  return score\n"
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected non-exhaustive match expression diagnostic")
	} else if !strings.Contains(err.Error(), "match expression must be exhaustive") {
		t.Fatalf("error = %v", err)
	}
}

func TestMatchExpressionRejectsMismatchedCaseTypes(t *testing.T) {
	src := "enum Result:\n  case ok(Int)\n  case err(Int)\n\nfunc main() -> Int:\n  let result: Result = Result.ok(42)\n  let score: Int = match result:\n  case Result.ok(value):\n    value\n  case Result.err(code):\n    \"bad\"\n  return score\n"
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected match expression case type diagnostic")
	} else if !strings.Contains(err.Error(), "match expression case type mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestMatchExpressionBindingScopeDiagnostic(t *testing.T) {
	src := "enum Result:\n  case ok(Int)\n  case err(Int)\n\nfunc main() -> Int:\n  let result: Result = Result.ok(42)\n  let score: Int = match result:\n  case Result.ok(value):\n    value\n  case Result.err(code):\n    code\n  return value\n"
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected match expression binding scope diagnostic")
	} else if !strings.Contains(err.Error(), "out of scope") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildIfLetEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "enum Result:\n  case ok(Int)\n  case err(Int)\n\nfunc main() -> Int:\n  let result: Result = Result.ok(42)\n  if let Result.ok(value) = result:\n    return value\n  else:\n    return 0\n"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildIfLetEnumNoPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "enum Result:\n  case ok(Int)\n  case empty\n\nfunc main() -> Int:\n  let result: Result = Result.empty\n  if let Result.empty = result:\n    return 42\n  else:\n    return 0\n"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildMatchGuardEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "enum Result:\n  case ok(Int)\n  case err(Int)\n\nfunc main() -> Int:\n  let result: Result = Result.ok(42)\n  match result:\n  case Result.ok(value) if value > 40:\n    return value\n  case Result.ok(other):\n    return 1\n  case Result.err(code):\n    return code\n"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestEnumMatchExhaustiveThreeCasesNoDefaultCheck(t *testing.T) {
	src := "enum Color:\n  case red\n  case green\n  case blue\n\nfunc main() -> Int:\n  let color: Color = Color.blue\n  match color:\n  case Color.red:\n    return 1\n  case Color.green:\n    return 2\n  case Color.blue:\n    return 3\n"
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("unexpected non-exhaustive enum diagnostic: %v", err)
	}
}

func TestEnumMatchMissingCaseStillNeedsReturn(t *testing.T) {
	src := "enum Color:\n  case red\n  case green\n\nfunc main() -> Int:\n  let color: Color = Color.green\n  match color:\n  case Color.red:\n    return 1\n"
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected missing return for non-exhaustive enum match")
	} else if !strings.Contains(err.Error(), "must end with return") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadConstructorArityDiagnostic(t *testing.T) {
	src := "enum Result:\n  case ok(Int)\n\nfunc main() -> Int:\n  let result: Result = Result.ok()\n  return 0\n"
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected enum payload arity diagnostic")
	} else if !strings.Contains(err.Error(), "expects 1 payload argument") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadConstructorTypeDiagnostic(t *testing.T) {
	src := "enum Result:\n  case ok(Int)\n\nfunc main() -> Int:\n  let result: Result = Result.ok(\"nope\")\n  return 0\n"
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected enum payload type diagnostic")
	} else if !strings.Contains(err.Error(), "payload 1 expects 'i32', got 'str'") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadBindingScopeDiagnostic(t *testing.T) {
	src := "enum Result:\n  case ok(Int)\n  case empty\n\nfunc main() -> Int:\n  let result: Result = Result.ok(1)\n  match result:\n  case Result.ok(value):\n    let inside: Int = value\n  case Result.empty:\n    let other: Int = 0\n  return value\n"
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected enum payload binding scope diagnostic")
	} else if !strings.Contains(err.Error(), "out of scope") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumNoPayloadConstructorCallDiagnostic(t *testing.T) {
	src := "enum Color:\n  case red\n\nfunc main() -> Int:\n  let color: Color = Color.red()\n  return 0\n"
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected no-payload enum constructor diagnostic")
	} else if !strings.Contains(err.Error(), "has no payload; use 'Color.red'") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadPatternArityDiagnostic(t *testing.T) {
	src := "enum Result:\n  case ok(Int, Int)\n\nfunc main() -> Int:\n  let result: Result = Result.ok(1, 2)\n  match result:\n  case Result.ok(value):\n    return value\n"
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected enum payload pattern arity diagnostic")
	} else if !strings.Contains(err.Error(), "pattern expects 2 binding(s), got 1") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadPatternRequiresPayloadSyntaxDiagnostic(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty

func main() -> Int:
  let result: Result = Result.ok(1)
  let score: Int = match result:
  case Result.ok:
    1
  case Result.empty:
    0
  return score
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected enum payload syntax diagnostic")
	} else if !strings.Contains(err.Error(), "carries 1 payload value(s); use 'Result.ok(value1)'") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadGuardedBarePatternStillRequiresDestructuring(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty

func main() -> Int:
  let result: Result = Result.ok(1)
  match result:
  case Result.ok if true:
    return 1
  case Result.ok(value):
    return value
  case Result.empty:
    return 0
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected guarded enum payload syntax diagnostic")
	} else if !strings.Contains(err.Error(), "requires payload arguments") && !strings.Contains(err.Error(), "carries 1 payload value(s); use 'Result.ok(value1)'") {
		t.Fatalf("error = %v", err)
	}
}

func TestMatchExpressionGuardedEnumPayloadCaseIsNotExhaustive(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty

func main() -> Int:
  let result: Result = Result.ok(1)
  let score: Int = match result:
  case Result.ok(value) if value > 0:
    value
  case Result.empty:
    0
  return score
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected non-exhaustive guarded enum payload match expression diagnostic")
	} else if !strings.Contains(err.Error(), "match expression must be exhaustive") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumMatchGuardedCasesDoNotCountAsExhaustive(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty

func main() -> Int:
  let result: Result = Result.ok(1)
  match result:
  case Result.ok(value) if value > 0:
    return value
  case Result.empty:
    return 0
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected missing return for guarded non-exhaustive enum match")
	} else if !strings.Contains(err.Error(), "must end with return") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumMatchDuplicateUnguardedPayloadCaseDiagnostic(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty

func main() -> Int:
  let result: Result = Result.ok(1)
  match result:
  case Result.ok(value):
    return value
  case Result.ok(other):
    return other
  case Result.empty:
    return 0
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected duplicate enum payload case diagnostic")
	} else if !strings.Contains(err.Error(), "duplicate match pattern") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumMatchDefaultMustBeLastDiagnostic(t *testing.T) {
	src := `
enum Color:
  case red
  case green

func main() -> Int:
  let color: Color = Color.red
  match color:
  case _:
    return 0
  case Color.red:
    return 1
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected default ordering diagnostic")
	} else if !strings.Contains(err.Error(), "match default must be last") {
		t.Fatalf("error = %v", err)
	}
}

func TestMatchExpressionDefaultMustBeLastDiagnostic(t *testing.T) {
	src := `
enum Color:
  case red
  case green

func main() -> Int:
  let color: Color = Color.red
  let score: Int = match color:
  case _:
    0
  case Color.red:
    1
  return score
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected match expression default ordering diagnostic")
	} else if !strings.Contains(err.Error(), "match default must be last") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumMatchRejectsWrongEnumCaseDiagnostic(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty
enum Other:
  case ok(Int)

func main() -> Int:
  let result: Result = Result.ok(1)
  match result:
  case Other.ok(value):
    return value
  case Result.empty:
    return 0
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected wrong enum case diagnostic")
	} else if !strings.Contains(err.Error(), "enum pattern type mismatch") && !strings.Contains(err.Error(), "match pattern type mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumNoPayloadPatternRejectsPayloadSyntaxDiagnostic(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty

func main() -> Int:
  let result: Result = Result.empty
  match result:
  case Result.ok(value):
    return value
  case Result.empty(value):
    return value
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected no-payload enum pattern diagnostic")
	} else if !strings.Contains(err.Error(), "has no payload; use 'Result.empty'") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadPatternDuplicateBindingParseDiagnostic(t *testing.T) {
	src := "enum Result:\n  case ok(Int, Int)\n\nfunc main() -> Int:\n  let result: Result = Result.ok(1, 2)\n  match result:\n  case Result.ok(value, value):\n    return value\n"
	if _, err := Parse([]byte(src)); err == nil {
		t.Fatalf("expected duplicate payload binding parse diagnostic")
	} else if !strings.Contains(err.Error(), "duplicate enum payload binding 'value'") {
		t.Fatalf("error = %v", err)
	}
}

func TestCrossModuleEnumPayloadConstructorAndMatchCheckLower(t *testing.T) {
	files := map[string]string{
		"lib/result.tetra": "module lib.result\n\npub enum Result:\n  case ok(Int)\n  case err(Int)\n",
		"app/main.tetra":   "module app.main\nimport lib.result as res\n\nfunc main() -> Int:\n  let result: res.Result = res.Result.ok(42)\n  let score: Int = match result:\n  case res.Result.ok(value):\n    value\n  case res.Result.err(code):\n    code\n  return score\n",
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, err := LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestBuildCrossModuleNoPayloadEnumMatchSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/result.tetra": "module lib.result\n\npub enum Result:\n  case ok(Int)\n  case empty\n",
		"app/main.tetra":   "module app.main\nimport lib.result as res\n\nfunc main() -> Int:\n  let result: res.Result = res.Result.empty\n  match result:\n  case res.Result.ok(value):\n    return value\n  case res.Result.empty:\n    return 42\n",
	}

	_, code := buildAndRunFiles(t, files, "app/main.tetra")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildIntMatchSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "func main() -> Int:\n  let value: Int = 7\n  match value:\n  case 1:\n    return 1\n  case 7:\n    return 42\n  case _:\n    return 0\n"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildTypedErrorsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "typed_errors_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildEnumPayloadSmokeFile(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "enum_payload_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildAsyncSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "async_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildTaskSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "task_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreMathSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_math_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreMemorySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_memory_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreStringsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_strings_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreSlicesSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_slices_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreIOSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_io_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreTestingSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_testing_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreCollectionsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_collections_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreSerializationSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_serialization_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreFilesystemSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_filesystem_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreNetworkingSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_networking_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreNetSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_net_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreJSONSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_json_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreHTTPSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_http_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCorePostgresSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_postgres_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCorePostgresPreparedSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_postgres_prepared_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCorePostgresResultSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_postgres_result_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreAsyncSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_async_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreSyncSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_sync_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreTimeSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_time_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreCryptoSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "core_crypto_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildExtensionSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "extension_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildGenericSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "generic_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestCoreV015SemanticDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "bool from int",
			src:  "func main() -> Int:\n  let x: Bool = 1\n  return 0\n",
			want: "type mismatch: expected 'bool', got 'i32'",
		},
		{
			name: "int from bool",
			src:  "func main() -> Int:\n  let x: Int = true\n  return x\n",
			want: "type mismatch: expected 'i32', got 'bool'",
		},
		{
			name: "duplicate enum case",
			src:  "enum Color:\n  case red\n  case red\nfunc main() -> Int:\n  return 0\n",
			want: "duplicate enum case 'red'",
		},
		{
			name: "unknown enum case",
			src:  "enum Color:\n  case red\nfunc main() -> Int:\n  let c: Color = Color.blue\n  return 0\n",
			want: "unknown enum case 'blue'",
		},
		{
			name: "compare different enums",
			src:  "enum A:\n  case one\nenum B:\n  case one\nfunc main() -> Int:\n  let a: A = A.one\n  let b: B = B.one\n  if a == b:\n    return 1\n  return 0\n",
			want: "cannot compare 'A' and 'B'",
		},
		{
			name: "invalid match pattern",
			src:  "enum Color:\n  case red\nfunc main() -> Int:\n  let c: Color = Color.red\n  match c:\n  case 1:\n    return 1\n  return 0\n",
			want: "match pattern type mismatch",
		},
		{
			name: "multiple defaults",
			src:  "func main() -> Int:\n  match 1:\n  case _:\n    return 1\n  case _:\n    return 2\n  return 0\n",
			want: "match default must be last",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := buildOnly(t, tt.src)
			if err == nil {
				t.Fatalf("expected build error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestBuildFlowStructSyntax(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "struct Vec2:\n  x: Int\n  y: Int\n\nfunc sum(v: Vec2) -> Int:\n  return v.x + v.y\n\nfunc main() -> Int:\n  let v: Vec2 = Vec2(x: 40, y: 2)\n  return sum(v)\n"
	_, exitCode := buildAndRun(t, src)
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFlowIslandSyntax(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "func main() -> Int\nuses alloc, islands, io, mem:\n  island(64) as isl:\n    var msg: []UInt8 = core.island_make_u8(isl, 2)\n    msg[0] = 79\n    msg[1] = 10\n    print(msg)\n  return 0\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "O\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFlowUnsafeCapMemSyntax(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "func main() -> Int\nuses alloc, capability, mem:\n  var out: Int = 1\n  unsafe:\n    let mem: cap.mem = core.cap_mem()\n    let p: ptr = core.alloc_bytes(4)\n    let _: Int = core.store_i32(p, 42, mem)\n    out = core.load_i32(p, mem)\n  return out\n"
	_, exitCode := buildAndRun(t, src)
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildBudgetedUnsafeCallsPreserveIRStack(t *testing.T) {
	src := `func main() -> Int
uses alloc, budget, capability, mem
budget(16):
    var out: Int = 1
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let _: Int = core.store_i32(p, 42, mem)
        out = core.load_i32(p, mem)
    return out
`
	if err := buildOnly(t, src); err != nil {
		t.Fatalf("BuildFile: %v", err)
	}
}

func TestBuildBudgetRuntimeGuardAllowsAndFailsDeterministically(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	okSrc := `func tick() -> Int
uses budget
budget(1):
    return 9

func main() -> Int
uses budget
budget(4):
    return tick()
`
	stdout, exitCode := buildAndRun(t, okSrc)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 9 {
		t.Fatalf("exit code = %d, want 9", exitCode)
	}

	failSrc := `func tick() -> Int
uses budget
budget(1):
    return 9

func main() -> Int
uses budget
budget(0):
    return tick()
`
	err := buildOnly(t, failSrc)
	if err == nil {
		t.Fatalf("expected compile-time budget context rejection")
	}
	if !strings.Contains(err.Error(), "budget context for call to 'tick' requires caller budget at least 1, got 0") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildBudgetFailureABIReturnAndThrowShapes(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tests := []struct {
		name     string
		src      string
		wantExit int
	}{
		{
			name: "non throwing multi slot return defaults to zero slots",
			src: `struct Pair:
    x: Int
    y: Int

func one() -> Int:
    return 7

func pair() -> Pair
uses budget
budget(0):
    return Pair(x: one(), y: 8)

func main() -> Int
uses budget
budget(16):
    let p: Pair = pair()
    return p.x + p.y
`,
			wantExit: 0,
		},
		{
			name: "throwing compact result returns thrown default payload",
			src: `enum BudgetTrap:
    case exhausted
    case other

func one() -> Int:
    return 99

func guarded() -> Int throws BudgetTrap
uses budget
budget(0):
    return one()

func main() -> Int
uses budget
budget(16):
    return catch guarded():
    case BudgetTrap.exhausted:
        21
    case BudgetTrap.other:
        22
`,
			wantExit: 21,
		},
		{
			name: "throwing non compact result returns thrown zero payload",
			src: `enum BudgetTrap:
    case exhausted(Int)
    case other(Int)

func one() -> Int:
    return 99

func guarded() -> Int throws BudgetTrap
uses budget
budget(0):
    return one()

func main() -> Int
uses budget
budget(16):
    return catch guarded():
    case BudgetTrap.exhausted(code):
        30 + code
    case BudgetTrap.other(otherCode):
        40 + otherCode
`,
			wantExit: 30,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, exitCode := buildAndRun(t, tt.src)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != tt.wantExit {
				t.Fatalf("exit code = %d, want %d", exitCode, tt.wantExit)
			}
		})
	}
}

func TestBuildPrivacyConsentRuntimeSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func seal(token: consent.token) -> secret.i32
uses privacy
privacy
consent(token):
    return core.secret_seal_i32(33, token)

func reveal(token: consent.token, value: secret.i32) -> Int
uses privacy
privacy
consent(token):
    return core.secret_unseal_i32(value, token)

func main() -> Int
uses privacy
privacy:
    let token: consent.token = core.consent_token()
    let secret: secret.i32 = seal(token)
    return reveal(token, secret)
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 33 {
		t.Fatalf("exit code = %d, want 33", exitCode)
	}
}

func TestBuildPrivacySealUnsealStaticOnlyDeterministicIdentity(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func roundtrip(token: consent.token, value: Int) -> Int
uses privacy
privacy
consent(token):
    let sealed: secret.i32 = core.secret_seal_i32(value, token)
    return core.secret_unseal_i32(sealed, token)

func main() -> Int
uses privacy
privacy:
    let token: consent.token = core.consent_token()
    let first: Int = roundtrip(token, 17)
    let second: Int = roundtrip(token, 17)
    let third: Int = roundtrip(token, 9)
    return (first - second) + third
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 9 {
		t.Fatalf("exit code = %d, want 9", exitCode)
	}
}

func TestBuildValAssignmentFails(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 {\n  val x: i32 = 1\n  x = 2\n  return x\n}\n"
	if err := buildOnly(t, src); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestBuildStructLiteralAndFieldAssign(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "struct Vec2 { x: i32, y: i32 }\nfun main(): i32 {\n  var v: Vec2 = Vec2{ x: 1, y: 2 }\n  v.x = 10\n  return v.x + v.y\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 12 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStructParam(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "struct Vec2 { x: i32, y: i32 }\nfun sum(v: Vec2): i32 {\n  return v.x + v.y\n}\nfun main(): i32 {\n  return sum(Vec2{ x: 5, y: 7 })\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 12 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStructCrossModule(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/math.tetra": "module engine.math\nstruct Vec2 { x: i32, y: i32 }\nfun sum(v: Vec2): i32 {\n  return v.x + v.y\n}\n",
		"app/game.tetra":    "module app.game\nimport engine.math as m\nfun main(): i32 {\n  var v: m.Vec2 = m.Vec2{ x: 2, y: 3 }\n  return m.sum(v)\n}\n",
	}
	stdout, exitCode := buildAndRunFiles(t, files, "app/game.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildImportedGlobalOnlyModuleSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/constants.t4": `module lib.constants

pub val answer: Int = 42
`,
		"app/main.t4": `module app.main
import lib.constants as constants

func main() -> Int:
    return 42
`,
	}
	stdout, exitCode := buildAndRunFiles(t, files, "app/main.t4")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStructValFieldAssignFails(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "struct Vec2 { x: i32, y: i32 }\nfun main(): i32 {\n  val v: Vec2 = Vec2{ x: 1, y: 2 }\n  v.x = 3\n  return v.x\n}\n"
	if err := buildOnly(t, src); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestBuildFunctionCall(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun add(a: i32, b: i32): i32 {\n  return a + b\n}\nfun main(): i32 {\n  return add(2, 3)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallSevenArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum7(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32): i32 {\n  return a + b + c + d + e + f + g\n}\nfun main(): i32 {\n  return sum7(1, 2, 3, 4, 5, 6, 7)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 28 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallEightArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum8(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32): i32 {\n  return a + b + c + d + e + f + g + h\n}\nfun main(): i32 {\n  return sum8(1, 2, 3, 4, 5, 6, 7, 8)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 36 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallNineArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum9(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32, i: i32): i32 {\n  return a + b + c + d + e + f + g + h + i\n}\nfun main(): i32 {\n  return sum9(1, 2, 3, 4, 5, 6, 7, 8, 9)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 45 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallPackNineArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun pack9(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32, i: i32): i32 {\n  var ok: i32 = 1\n  if (a == 1) { ; } else { ok = 0 }\n  if (b == 2) { ; } else { ok = 0 }\n  if (c == 3) { ; } else { ok = 0 }\n  if (d == 4) { ; } else { ok = 0 }\n  if (e == 5) { ; } else { ok = 0 }\n  if (f == 6) { ; } else { ok = 0 }\n  if (g == 7) { ; } else { ok = 0 }\n  if (h == 8) { ; } else { ok = 0 }\n  if (i == 9) { ; } else { ok = 0 }\n  return ok\n}\nfun main(): i32 {\n  return pack9(1, 2, 3, 4, 5, 6, 7, 8, 9)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallEightArgsNonEmptyStack(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum8(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32): i32 {\n  return a + b + c + d + e + f + g + h\n}\nfun main(): i32 {\n  return 1 + sum8(1, 2, 3, 4, 5, 6, 7, 8)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 37 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallNineArgsNonEmptyStack(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum9(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32, i: i32): i32 {\n  return a + b + c + d + e + f + g + h + i\n}\nfun main(): i32 {\n  return 1 + sum9(1, 2, 3, 4, 5, 6, 7, 8, 9)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 46 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallSevenArgsNonEmptyStack(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum7(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32): i32 {\n  return a + b + c + d + e + f + g\n}\nfun main(): i32 {\n  return 1 + sum7(1, 2, 3, 4, 5, 6, 7)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 29 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallNestedArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun sum2(a: i32, b: i32): i32 {\n  return a + b\n}\nfun pack9(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32, h: i32, i: i32): i32 {\n  var ok: i32 = 1\n  if (a == 21) { ; } else { ok = 0 }\n  if (b == 2) { ; } else { ok = 0 }\n  if (c == 3) { ; } else { ok = 0 }\n  if (d == 4) { ; } else { ok = 0 }\n  if (e == 5) { ; } else { ok = 0 }\n  if (f == 6) { ; } else { ok = 0 }\n  if (g == 7) { ; } else { ok = 0 }\n  if (h == 8) { ; } else { ok = 0 }\n  if (i == 9) { ; } else { ok = 0 }\n  return ok\n}\nfun main(): i32 {\n  return pack9(sum2(10, 11), 2, 3, 4, 5, 6, 7, 8, 9)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMultiFileCrossModuleCall(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as render\nfun main(): i32 {\n  val v: i32 = render.add_one(41)\n  if (v == 42) { return 1 }\n  return 0\n}\n",
	}
	stdout, exitCode := buildAndRunFiles(t, files, "app/game.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMultiFileAliasCall(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as r\nfun main(): i32 {\n  return r.add_one(41)\n}\n",
	}
	stdout, exitCode := buildAndRunFiles(t, files, "app/game.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMultiFileMissingModule(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"app/game.tetra": "module app.game\nimport engine.render as r\nfun main(): i32 {\n  return r.add_one(1)\n}\n",
	}
	if err := buildOnlyFiles(t, files, "app/game.tetra"); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestBuildMultiFileImportCycle(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"app/game.tetra": "module app.game\nimport mod.a as a\nfun main(): i32 {\n  return a.ping()\n}\n",
		"mod/a.tetra":    "module mod.a\nimport mod.b as b\nfun ping(): i32 {\n  return b.pong()\n}\n",
		"mod/b.tetra":    "module mod.b\nimport mod.a as a\nfun pong(): i32 {\n  return 1\n}\n",
	}
	if err := buildOnlyFiles(t, files, "app/game.tetra"); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestBuildMultiFileDuplicateModule(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra":       "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"engine/render_alias.tetra": "module engine.render\nfun add_two(x: i32): i32 {\n  return x + 2\n}\n",
		"app/game.tetra":            "module app.game\nimport engine.render as r\nimport engine.render_alias as r2\nfun main(): i32 {\n  return r.add_one(1) + r2.add_two(1)\n}\n",
	}
	if err := buildOnlyFiles(t, files, "app/game.tetra"); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestLoadWorldRejectsDuplicateImportPath(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as render\nimport engine.render as r\nfun main(): i32 {\n  return render.add_one(41)\n}\n",
	}
	writeTestFiles(t, tmp, files)

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err == nil {
		t.Fatalf("expected duplicate import path error")
	}
	if !strings.Contains(err.Error(), "duplicate import 'engine.render'") {
		t.Fatalf("error = %v, want duplicate import path diagnostic", err)
	}
}

func TestLoadWorldReportsMissingImportPath(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"app/game.tetra": "module app.game\nimport engine.missing as missing\nfun main(): i32 {\n  return missing.add_one(41)\n}\n",
	}
	writeTestFiles(t, tmp, files)

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err == nil {
		t.Fatalf("expected missing import error")
	}
	if !strings.Contains(err.Error(), "load module 'engine.missing'") || !strings.Contains(err.Error(), "read source") {
		t.Fatalf("error = %v, want missing import path diagnostic", err)
	}
}

func TestLoadWorldReportsImportCycle(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"app/game.tetra": "module app.game\nimport mod.a as a\nfun main(): i32 {\n  return a.ping()\n}\n",
		"mod/a.tetra":    "module mod.a\nimport mod.b as b\nfun ping(): i32 {\n  return b.pong()\n}\n",
		"mod/b.tetra":    "module mod.b\nimport mod.a as a\nfun pong(): i32 {\n  return 1\n}\n",
	}
	writeTestFiles(t, tmp, files)

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	if !strings.Contains(err.Error(), "import cycle detected at 'mod.a'") {
		t.Fatalf("error = %v, want import cycle diagnostic", err)
	}
}

func TestLoadWorldReportsDuplicateModuleDeclaration(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/render.tetra":       "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"engine/render_alias.tetra": "module engine.render\nfun add_two(x: i32): i32 {\n  return x + 2\n}\n",
		"app/game.tetra":            "module app.game\nimport engine.render as r\nimport engine.render_alias as r2\nfun main(): i32 {\n  return r.add_one(1) + r2.add_two(1)\n}\n",
	}
	writeTestFiles(t, tmp, files)

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err == nil {
		t.Fatalf("expected duplicate module error")
	}
	if !strings.Contains(err.Error(), "duplicate module 'engine.render'") {
		t.Fatalf("error = %v, want duplicate module diagnostic", err)
	}
}

func TestCheckWorldRejectsImportAliasShadowingTopLevelName(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/math.tetra": "module engine.math\nfun inc(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra":    "module app.game\nimport engine.math as math\nfun math(): i32 {\n  return 1\n}\nfun main(): i32 {\n  return math()\n}\n",
	}
	writeTestFiles(t, tmp, files)

	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	_, err = CheckWorld(world)
	if err == nil {
		t.Fatalf("expected alias shadowing error")
	}
	if !strings.Contains(err.Error(), "import alias 'math' conflicts with declaration 'math'") {
		t.Fatalf("error = %v, want alias shadowing diagnostic", err)
	}
}

func TestBuildValArgument(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun add(a: i32, b: i32): i32 {\n  return a + b\n}\nfun main(): i32 {\n  val x: i32 = 4\n  return add(x, 1)\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildEmptyStatements(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  ;;;\n  let x: i32 = 2 + 3;\n  ;;;\n  return x;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func buildAndRunFile(t *testing.T, srcPath string) (string, int) {
	t.Helper()

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "app")
	if err := BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func buildAndRunFileWithOptions(t *testing.T, srcPath string, opt BuildOptions) (string, int) {
	t.Helper()

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "app")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", opt); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func projectRoot(t *testing.T) string {
	t.Helper()
	// Walk up from the test binary's working directory to find the project root.
	// The go test framework runs in the package dir, so we go up from compiler/.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// wd is .../compiler, project root is parent
	return filepath.Dir(wd)
}

func TestExampleHello(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "hello.tetra"))
	if stdout != "Hello from Tetra!\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestExampleGlobalsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "globals_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	// g_x = g_y + 2 = 40 + 2 = 42, store 7 at g_p, out = 7 + 42 = 49
	if exitCode != 49 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestExampleStructCtorSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "struct_ctor_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	// v.x + v.y + 52 = 40 + 2 + 52 = 94
	if exitCode != 94 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestExampleExperimentalMath(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "experimental_math_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	// math.add_i32(40, 2) = 42
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestExampleExperimentalMemcpy(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "experimental_memcpy_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 93 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestExampleCapMemPtr(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "cap_mem_ptr_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 77 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestExampleCapMemPtrAddLocal(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "cap_mem_smoke.tetra"))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 77 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestMicroserviceExamplesAndBugLedger(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	for _, name := range []string{
		"inventory_service.tetra",
		"payments_service.tetra",
		"orders_gateway.tetra",
		"memory_cache_service.tetra",
		"parallel_fanout_service.tetra",
		"compiler_pipeline_service.tetra",
		"island_cache_pool_service.tetra",
		"parallel_task_pool_service.tetra",
		"compiler_artifact_router_service.tetra",
		"memory_journal_service.tetra",
		"task_group_service.tetra",
		"typed_task_error_service.tetra",
		"task_group_cancel_service.tetra",
		"wait_select_service.tetra",
		"memory_bounds_probe_service.tetra",
		"callable_router_service.tetra",
		filepath.FromSlash("compiler_modular_gateway/app/main.tetra"),
		"island_slice_matrix_service.tetra",
		"generic_optional_router_service.tetra",
		"actor_deadline_router_service.tetra",
		"typed_task_success_service.tetra",
		"memory_byte_window_service.tetra",
		"callable_return_router_service.tetra",
		filepath.FromSlash("compiler_callable_pack/app/main.tetra"),
		"actor_tagged_loop_service.tetra",
		"task_group_lifecycle_service.tetra",
		"memory_negative_guard_service.tetra",
		"callable_identity_router_service.tetra",
		filepath.FromSlash("compiler_throwing_callable_pack/app/main.tetra"),
		"actor_poll_timeout_service.tetra",
		"task_timeout_recovery_service.tetra",
		"memory_u16_lane_service.tetra",
		"generic_struct_router_service.tetra",
		filepath.FromSlash("compiler_generic_box_pack/app/main.tetra"),
		"task_group_payload_service.tetra",
		"actor_sender_snapshot_service.tetra",
		"memory_copy_window_service.tetra",
		"protocol_bound_generic_service.tetra",
		filepath.FromSlash("compiler_protocol_pack/app/main.tetra"),
		"actor_state_counter_service.tetra",
		"task_group_self_cancel_service.tetra",
		"generic_typed_error_service.tetra",
		filepath.FromSlash("compiler_generic_error_pack/app/main.tetra"),
		"task_group_current_status_service.tetra",
		"actor_dual_mailbox_service.tetra",
		"memory_memset_stride_service.tetra",
		"island_bool_flags_service.tetra",
		filepath.FromSlash("compiler_generic_pair_pack/app/main.tetra"),
		"actor_dual_value_mailbox_service.tetra",
		"task_dual_deadline_service.tetra",
		"memory_zero_copy_service.tetra",
		filepath.FromSlash("compiler_optional_box_pack/app/main.tetra"),
		"actor_timeout_retry_service.tetra",
		"task_poll_deadline_matrix_service.tetra",
		"memory_ptr_table_service.tetra",
		"optional_enum_router_service.tetra",
		"optional_field_update_service.tetra",
		"actor_chain_reply_service.tetra",
		"task_group_poll_service.tetra",
		"memory_i32_stride_service.tetra",
		"actor_value_chain_service.tetra",
		"task_group_typed_success_service.tetra",
		"memory_chained_ptr_stride_service.tetra",
		filepath.FromSlash("compiler_optional_enum_pack/app/main.tetra"),
		"actor_typed_payload_service.tetra",
		"task_select_timeout_service.tetra",
		"memory_mixed_width_service.tetra",
		filepath.FromSlash("compiler_extension_pack/app/main.tetra"),
		"actor_self_mailbox_service.tetra",
		"task_group_cancel_after_spawn_service.tetra",
		"memory_derived_copy_service.tetra",
		filepath.FromSlash("compiler_protocol_extension_pack/app/main.tetra"),
		"actor_typed_chain_service.tetra",
		"task_group_multi_cancel_service.tetra",
		"memory_derived_ptr_table_service.tetra",
		filepath.FromSlash("compiler_generic_function_pack/app/main.tetra"),
		"actor_self_typed_mailbox_service.tetra",
		"actor_task_bridge_service.tetra",
		"memory_aggregate_ptr_service.tetra",
		"compiler_generic_extension_local_service.tetra",
		"actor_typed_task_bridge_service.tetra",
		"task_group_actor_fanout_service.tetra",
		"memory_optional_ptr_service.tetra",
		"compiler_callable_generic_route_service.tetra",
		"task_actor_roundtrip_service.tetra",
		"actor_typed_task_group_service.tetra",
		"memory_function_ptr_service.tetra",
		"task_typed_actor_roundtrip_service.tetra",
		"actor_task_select_service.tetra",
		"compiler_generic_optional_route_service.tetra",
		"memory_global_state_service.tetra",
		"actor_typed_task_error_bridge_service.tetra",
		"actor_task_cancel_select_service.tetra",
		filepath.FromSlash("compiler_generic_optional_import_pack/app/main.tetra"),
		"memory_mutable_ptr_service.tetra",
		"memory_struct_offset_service.tetra",
		"actor_task_recovery_service.tetra",
		filepath.FromSlash("compiler_generic_nested_optional_pack/app/main.tetra"),
		"memory_function_offset_service.tetra",
		"memory_expression_offset_service.tetra",
		"actor_timer_task_matrix_service.tetra",
		filepath.FromSlash("compiler_generic_enum_import_pack/app/main.tetra"),
		"memory_task_result_offset_service.tetra",
		"memory_actor_message_offset_service.tetra",
		"memory_actor_recv_value_offset_service.tetra",
		"memory_actor_poll_value_offset_service.tetra",
		"memory_actor_tag_offset_service.tetra",
		filepath.FromSlash("compiler_actor_wait_memory_pack/app/main.tetra"),
		"memory_actor_recv_error_offset_service.tetra",
		"memory_actor_poll_error_offset_service.tetra",
		"memory_actor_recv_msg_error_offset_service.tetra",
		filepath.FromSlash("compiler_actor_error_memory_pack/app/main.tetra"),
		"actor_task_group_error_recovery_service.tetra",
		filepath.FromSlash("compiler_generic_struct_field_pack/app/main.tetra"),
		"memory_indexed_metadata_offset_service.tetra",
		"parallel_typed_task_payload_handle_service.tetra",
		"actor_typed_dual_mailbox_service.tetra",
		"task_group_nested_service.tetra",
		filepath.FromSlash("compiler_generic_optional_struct_pack/app/main.tetra"),
		"memory_direct_base_offset_service.tetra",
		"parallel_typed_task_wide_payload_service.tetra",
		"actor_typed_wide_payload_service.tetra",
		filepath.FromSlash("compiler_cross_module_runtime_pack/app/main.tetra"),
		"actor_typed_envelope_service.tetra",
		"parallel_time_window_service.tetra",
		"actor_state_status_service.tetra",
		"memory_inline_ptradd_window_service.tetra",
		"parallel_typed_task_struct_payload_service.tetra",
		"actor_typed_struct_payload_service.tetra",
		"memory_callable_ptr_base_service.tetra",
		"memory_callable_optional_ptr_service.tetra",
		"compiler_match_ptr_base_service.tetra",
		"memory_typed_error_ptr_base_service.tetra",
		"parallel_join_until_rejoin_service.tetra",
		"actor_task_result_window_service.tetra",
		"compiler_inout_return_service.tetra",
		"memory_dynamic_base_offset_service.tetra",
		"parallel_group_close_before_join_service.tetra",
		"parallel_group_cancel_after_join_service.tetra",
		filepath.FromSlash("parallel_cross_module_typed_task_pack/app/main.tetra"),
		"memory_struct_base_dynamic_service.tetra",
		"memory_enum_base_dynamic_service.tetra",
		"memory_typed_error_base_dynamic_service.tetra",
		"parallel_select_recovery_service.tetra",
		"compiler_pattern_binding_unique_service.tetra",
		"memory_base_dynamic_copy_service.tetra",
		"parallel_select_rejoin_service.tetra",
		"parallel_group_cancel_select_service.tetra",
		filepath.FromSlash("compiler_interface_jobs_pack/app/main.tetra"),
		"memory_zero_length_derived_helper_service.tetra",
		"parallel_group_spawn_after_cancel_service.tetra",
		"parallel_join_until_poll_service.tetra",
		filepath.FromSlash("compiler_interface_control_pack/app/main.tetra"),
		"memory_zero_length_base_helper_service.tetra",
		"parallel_yield_join_window_service.tetra",
		"parallel_group_status_roundtrip_service.tetra",
		"memory_group_status_direct_offset_service.tetra",
		"memory_group_current_status_offset_service.tetra",
		"parallel_group_cancel_close_direct_service.tetra",
		filepath.FromSlash("compiler_group_status_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_import_alias_pack/app/main.tetra"),
		"memory_heap_u16_slice_service.tetra",
		"memory_heap_bool_flags_service.tetra",
		"parallel_actor_yield_mailbox_service.tetra",
		"parallel_group_current_cancel_status_service.tetra",
		filepath.FromSlash("compiler_cross_module_actor_pack/app/main.tetra"),
		"memory_heap_i32_bool_slice_service.tetra",
		"parallel_task_actor_deadline_service.tetra",
		filepath.FromSlash("compiler_actor_resource_pack/app/main.tetra"),
		"memory_heap_u8_slice_service.tetra",
		"parallel_typed_group_cancel_status_service.tetra",
		filepath.FromSlash("compiler_callable_return_pack/app/main.tetra"),
		filepath.FromSlash("compiler_callable_optional_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_interface_pack/app/main.tetra"),
		"memory_slice_optional_service.tetra",
		"memory_slice_enum_service.tetra",
		"parallel_task_result_box_service.tetra",
		"parallel_task_result_enum_service.tetra",
		"compiler_test_command_service.tetra",
		"parallel_task_test_command_service.tetra",
		"generic_typed_result_payload_service.tetra",
		"memory_slice_struct_loop_service.tetra",
		"memory_slice_generic_box_service.tetra",
		"parallel_task_result_optional_service.tetra",
		"parallel_nested_task_spawn_service.tetra",
		filepath.FromSlash("compiler_generic_slice_pack/app/main.tetra"),
		"memory_slice_for_loop_service.tetra",
		"memory_slice_inout_mutation_service.tetra",
		"parallel_task_handle_optional_join_service.tetra",
		filepath.FromSlash("compiler_optional_task_pack/app/main.tetra"),
		"memory_bool_for_loop_service.tetra",
		"memory_i32_for_loop_service.tetra",
		"parallel_actor_handle_optional_send_service.tetra",
		filepath.FromSlash("compiler_optional_actor_pack/app/main.tetra"),
		"memory_u16_for_loop_service.tetra",
		"parallel_group_optional_close_service.tetra",
		"parallel_group_optional_cancel_service.tetra",
		filepath.FromSlash("compiler_optional_group_pack/app/main.tetra"),
		"memory_bool_inout_toggle_service.tetra",
		"memory_i32_inout_fill_service.tetra",
		"parallel_group_optional_match_close_service.tetra",
		filepath.FromSlash("compiler_optional_group_match_pack/app/main.tetra"),
		"parallel_group_struct_spawn_service.tetra",
		"parallel_group_enum_spawn_service.tetra",
		"parallel_group_typed_struct_spawn_service.tetra",
		"parallel_group_typed_enum_spawn_service.tetra",
		"memory_u16_inout_stride_service.tetra",
		filepath.FromSlash("compiler_group_aggregate_pack/app/main.tetra"),
		"parallel_group_alias_spawn_service.tetra",
		"parallel_group_generic_box_spawn_service.tetra",
		"memory_optional_generic_u16_box_service.tetra",
		filepath.FromSlash("compiler_group_generic_pack/app/main.tetra"),
		"parallel_task_alias_join_service.tetra",
		"parallel_task_generic_box_join_service.tetra",
		"parallel_task_optional_struct_box_join_service.tetra",
		"parallel_task_optional_generic_box_join_service.tetra",
		"memory_optional_generic_bool_box_service.tetra",
		filepath.FromSlash("compiler_task_generic_pack/app/main.tetra"),
		"parallel_actor_alias_send_service.tetra",
		"parallel_actor_generic_box_send_service.tetra",
		"parallel_actor_optional_struct_box_send_service.tetra",
		"parallel_actor_optional_generic_box_send_service.tetra",
		"memory_optional_generic_i32_box_service.tetra",
		filepath.FromSlash("compiler_actor_generic_pack/app/main.tetra"),
		"memory_island_alias_region_service.tetra",
		"memory_island_generic_box_region_service.tetra",
		"memory_island_optional_struct_box_service.tetra",
		"memory_island_optional_generic_box_service.tetra",
		filepath.FromSlash("compiler_island_generic_pack/app/main.tetra"),
		"memory_ptr_alias_base_service.tetra",
		"memory_ptr_generic_identity_base_service.tetra",
		filepath.FromSlash("compiler_ptr_generic_pack/app/main.tetra"),
		"memory_task_result_optional_offset_service.tetra",
		"parallel_task_result_generic_box_service.tetra",
		filepath.FromSlash("compiler_task_result_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_optional_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_resource_wrapper_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_resource_wrapper_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_generic_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_generic_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_optional_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_generic_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_enum_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_slice_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_slice_lane_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_slice_shape_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_string_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_optional_generic_string_pack/app/main.tetra"),
		"memory_ptr_generic_optional_field_service.tetra",
		"parallel_task_result_generic_optional_field_service.tetra",
		filepath.FromSlash("compiler_generic_optional_field_pack/app/main.tetra"),
		"memory_ptr_generic_optional_call_service.tetra",
		"memory_optional_ptr_inout_return_service.tetra",
		filepath.FromSlash("compiler_ptr_optional_generic_call_pack/app/main.tetra"),
		"parallel_actor_optional_alias_send_service.tetra",
		"parallel_task_optional_alias_join_service.tetra",
		"parallel_group_optional_alias_close_service.tetra",
		filepath.FromSlash("compiler_optional_alias_resource_pack/app/main.tetra"),
		"parallel_actor_optional_enum_send_service.tetra",
		"parallel_task_optional_enum_join_service.tetra",
		"parallel_group_optional_enum_close_service.tetra",
		"memory_task_result_optional_enum_offset_service.tetra",
		filepath.FromSlash("compiler_optional_enum_resource_pack/app/main.tetra"),
		"parallel_actor_typed_optional_alias_send_service.tetra",
		"parallel_typed_group_optional_alias_spawn_service.tetra",
		filepath.FromSlash("compiler_typed_optional_alias_resource_pack/app/main.tetra"),
		"parallel_typed_task_match_catch_service.tetra",
		"memory_typed_task_error_offset_service.tetra",
		filepath.FromSlash("compiler_typed_task_match_pack/app/main.tetra"),
		"memory_typed_task_error_struct_offset_service.tetra",
		"memory_typed_task_error_nested_enum_offset_service.tetra",
		"memory_typed_task_error_optional_offset_service.tetra",
		"memory_typed_task_error_guarded_offset_service.tetra",
		filepath.FromSlash("compiler_typed_error_payload_memory_pack/app/main.tetra"),
		"parallel_defer_group_close_service.tetra",
		"memory_defer_store_service.tetra",
		"parallel_defer_group_cancel_checkpoint_service.tetra",
		"memory_defer_task_result_offset_service.tetra",
		filepath.FromSlash("compiler_defer_cleanup_pack/app/main.tetra"),
		"memory_defer_throw_base_store_service.tetra",
		"memory_defer_return_base_store_service.tetra",
		"parallel_typed_task_defer_actor_reply_service.tetra",
		filepath.FromSlash("compiler_defer_unwind_pack/app/main.tetra"),
		"memory_join_until_result_offset_service.tetra",
		"memory_poll_result_offset_service.tetra",
		"memory_select_result_offset_service.tetra",
		filepath.FromSlash("compiler_task_wait_memory_pack/app/main.tetra"),
		"memory_join_until_error_offset_service.tetra",
		"memory_poll_error_offset_service.tetra",
		"memory_select_error_offset_service.tetra",
		filepath.FromSlash("compiler_task_wait_error_memory_pack/app/main.tetra"),
		"memory_typed_error_optional_ptr_base_service.tetra",
		"memory_typed_error_optional_ptr_dynamic_service.tetra",
		filepath.FromSlash("compiler_typed_error_optional_ptr_pack/app/main.tetra"),
		"memory_actor_typed_payload_offset_service.tetra",
		"memory_actor_typed_struct_payload_offset_service.tetra",
		filepath.FromSlash("compiler_typed_actor_payload_memory_pack/app/main.tetra"),
		"memory_actor_typed_enum_payload_offset_service.tetra",
		"memory_actor_typed_enum_struct_payload_offset_service.tetra",
		filepath.FromSlash("compiler_typed_actor_enum_payload_memory_pack/app/main.tetra"),
	} {
		stdout, exitCode := buildAndRunFile(t, filepath.Join(root, "examples", "microservices", name))
		if stdout != "" {
			t.Fatalf("%s stdout mismatch: %q", name, stdout)
		}
		if exitCode != 0 {
			t.Fatalf("%s exit code mismatch: %d", name, exitCode)
		}
	}
	for _, name := range []string{
		filepath.FromSlash("compiler_parallel_jobs_pack/app/main.tetra"),
	} {
		stdout, exitCode := buildAndRunFileWithOptions(t, filepath.Join(root, "examples", "microservices", name), BuildOptions{Jobs: 4})
		if stdout != "" {
			t.Fatalf("%s stdout mismatch: %q", name, stdout)
		}
		if exitCode != 0 {
			t.Fatalf("%s exit code mismatch: %d", name, exitCode)
		}
	}
	for _, name := range []string{
		filepath.FromSlash("compiler_interface_jobs_pack/app/main.tetra"),
		filepath.FromSlash("compiler_interface_control_pack/app/main.tetra"),
		filepath.FromSlash("compiler_import_alias_pack/app/main.tetra"),
		filepath.FromSlash("compiler_cross_module_actor_pack/app/main.tetra"),
		filepath.FromSlash("compiler_actor_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_callable_return_pack/app/main.tetra"),
		filepath.FromSlash("compiler_callable_optional_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_interface_pack/app/main.tetra"),
		filepath.FromSlash("compiler_generic_slice_pack/app/main.tetra"),
		filepath.FromSlash("compiler_optional_task_pack/app/main.tetra"),
		filepath.FromSlash("compiler_optional_actor_pack/app/main.tetra"),
		filepath.FromSlash("compiler_optional_group_pack/app/main.tetra"),
		filepath.FromSlash("compiler_optional_group_match_pack/app/main.tetra"),
		filepath.FromSlash("compiler_group_aggregate_pack/app/main.tetra"),
		filepath.FromSlash("compiler_group_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_task_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_actor_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_island_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_ptr_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_task_result_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_optional_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_resource_wrapper_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_resource_wrapper_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_generic_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_generic_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_optional_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_generic_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_enum_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_slice_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_slice_lane_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_slice_shape_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_string_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_optional_generic_string_pack/app/main.tetra"),
		filepath.FromSlash("compiler_generic_optional_field_pack/app/main.tetra"),
		filepath.FromSlash("compiler_ptr_optional_generic_call_pack/app/main.tetra"),
		filepath.FromSlash("compiler_optional_alias_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_optional_enum_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_typed_optional_alias_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_typed_task_match_pack/app/main.tetra"),
		filepath.FromSlash("compiler_typed_error_payload_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_defer_cleanup_pack/app/main.tetra"),
		filepath.FromSlash("compiler_defer_unwind_pack/app/main.tetra"),
		filepath.FromSlash("compiler_task_wait_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_task_wait_error_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_actor_wait_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_actor_error_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_group_status_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_typed_error_optional_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_typed_actor_payload_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_typed_actor_enum_payload_memory_pack/app/main.tetra"),
	} {
		outPath := filepath.Join(t.TempDir(), "interface-only")
		if _, err := BuildFileWithStatsOpt(filepath.Join(root, "examples", "microservices", name), outPath, "linux-x64", BuildOptions{Jobs: 4, InterfaceOnly: true}); err != nil {
			t.Fatalf("%s interface-only build: %v", name, err)
		}
	}
	for _, name := range []string{
		"parallel_selfhost_deadline_service.tetra",
	} {
		stdout, exitCode := buildAndRunFileWithOptions(t, filepath.Join(root, "examples", "microservices", name), BuildOptions{Runtime: RuntimeSelfHost})
		if stdout != "" {
			t.Fatalf("%s stdout mismatch: %q", name, stdout)
		}
		if exitCode != 0 {
			t.Fatalf("%s exit code mismatch: %d", name, exitCode)
		}
	}

	raw, err := os.ReadFile(filepath.Join(root, "Tetra_BUGS.md"))
	if err != nil {
		t.Fatalf("read Tetra_BUGS.md: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"# Tetra Bugs",
		"Confirmed Language Bugs",
		"TETRA-BUG-0001",
		"cannot infer generic argument",
		"TETRA-BUG-0002",
		"unknown function 'Unit.score'",
		"TETRA-BUG-0003",
		"count mismatch: expected 1, got 2",
		"TETRA-BUG-0004",
		"Formatter drops function-typed local annotations",
		"TETRA-BUG-0005",
		"unknown function 'Router.run'",
		"TETRA-BUG-0006",
		"global var requires an explicit type annotation",
		"TETRA-BUG-0007",
		"Derived pointer arithmetic loses allocation provenance",
		"TETRA-BUG-0008",
		"Formatter rewrites mutable actor state fields as immutable",
		"TETRA-BUG-0009",
		"Blocking tagged receive fails in dual actor fan-in",
		"TETRA-BUG-0010",
		"Blocking value receive fails in dual actor fan-in",
		"TETRA-BUG-0011",
		"Struct constructors do not wrap scalar values into optional fields",
		"TETRA-BUG-0012",
		"Enum constructors do not wrap scalar values into optional payloads",
		"TETRA-BUG-0013",
		"Derived pointer loop arithmetic fails after pointer parameters",
		"TETRA-BUG-0014",
		"Formatter drops generic protocol requirement type parameters",
		"TETRA-BUG-0015",
		"Imported generic extension static calls do not monomorphize",
		"TETRA-BUG-0016",
		"Match case payload bindings leak across sibling cases",
		"TETRA-BUG-0017",
		"Stored derived pointers lose loadable memory provenance",
		"TETRA-BUG-0018",
		"Struct pointer fields lose memory provenance",
		"TETRA-BUG-0019",
		"Enum derived pointer payloads lose memory provenance",
		"TETRA-BUG-0020",
		"Generic identity over function-typed locals lowers to an unknown fn type",
		"TETRA-BUG-0021",
		"Optional derived pointer payloads lose memory provenance",
		"TETRA-BUG-0022",
		"Generic callback parameters do not accept compatible function symbols",
		"TETRA-BUG-0023",
		"Function returns of derived pointers lose memory provenance",
		"Function-typed callable returns of derived pointers hit the same guard",
		"TETRA-BUG-0024",
		"Global pointer variables lose memory provenance",
		"TETRA-BUG-0025",
		"Global integer offsets break raw pointer arithmetic provenance",
		"TETRA-BUG-0026",
		"Mutable local derived pointer variables lose memory provenance",
		"TETRA-BUG-0027",
		"Struct field offsets break raw pointer arithmetic provenance",
		"TETRA-BUG-0028",
		"Function-call offset operands break raw pointer arithmetic provenance",
		"TETRA-BUG-0029",
		"Expression offset operands break raw pointer arithmetic provenance",
		"TETRA-BUG-0030",
		"Runtime result and message fields break raw pointer arithmetic provenance",
		"TETRA-BUG-0031",
		"Enum payloads reject generic struct instantiations",
		"TETRA-BUG-0032",
		"Indexed and metadata offsets break raw pointer arithmetic provenance",
		"TETRA-BUG-0033",
		"Payload-typed task handles reject explicit task.i32 annotations",
		"TETRA-BUG-0034",
		"Direct pointer base expressions break raw pointer arithmetic provenance",
		"TETRA-BUG-0035",
		"Typed actor receives silently reinterpret mismatched enum message types",
		"TETRA-BUG-0036",
		"Typed error payloads of derived pointers lose memory provenance",
		"TETRA-BUG-0037",
		"Global fixed-array element writes do not round-trip at runtime",
		"TETRA-BUG-0038",
		"Scalar inout writes do not propagate back to caller locals",
		"TETRA-BUG-0039",
		"Dynamic ptr_add offsets from derived pointer locals lose memory provenance",
		"TETRA-BUG-0040",
		"Explicit selfhost task-group builds fail with raw missing ABI symbol",
		"TETRA-BUG-0041",
		"Scoped if-let and catch payload bindings remain reserved after scope exit",
		"TETRA-BUG-0042",
		"Stdlib byte helpers fail on valid derived memory windows",
		"TETRA-BUG-0043",
		"Formatter drops public visibility modifiers from declarations",
		"TETRA-BUG-0044",
		"Formatter corrupts selective import declarations",
		"TETRA-BUG-0045",
		"Spawning through an optional task-group payload returns the wrong worker value",
		"TETRA-BUG-0046",
		"Generic identity over actor/task/island resources loses usable provenance",
		"TETRA-BUG-0047",
		"Island parameters cannot be returned inside aggregate constructors",
		"TETRA-BUG-0048",
		"Function-typed local, field, and payload calls returning optionals fail as unknown functions",
		"TETRA-BUG-0049",
		"Generic inference fails on generic struct field selections",
		"TETRA-BUG-0050",
		"Task spawns inside match expressions miss required runtime symbols",
		"TETRA-BUG-0051",
		"Formatter corrupts nested catch cases inside match expression arms",
		"TETRA-BUG-0052",
		"Formatter corrupts nested match cases inside catch expression arms",
		"TETRA-BUG-0053",
		"Awaited optional resource locals lose provenance",
		"TETRA-BUG-0054",
		"Awaited resource aggregate locals lose provenance",
		"TETRA-BUG-0055",
		"Direct awaited pointer returns ignore await and try",
		"Microservice Bug-Hunt Runs",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Tetra_BUGS.md missing %q", want)
		}
	}
}

func TestNewOperatorMul(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { return 6 * 7 }"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestNewOperatorDiv(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { return 84 / 2 }"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestNewOperatorMod(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { return 47 % 5 }"
	_, code := buildAndRun(t, src)
	if code != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", code)
	}
}

func TestNewOperatorGreater(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (5 > 3) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorGreaterEq(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (3 >= 3) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorLessEq(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (3 <= 3) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorBangEq(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (2 != 3) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorAmpAmp(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (true && true) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorAmpAmpFalse(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (true && false) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 0 {
		t.Fatalf("exit code mismatch: got %d, want 0", code)
	}
}

func TestNewOperatorPipePipe(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (false || true) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorPipePipeFalse(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (false || false) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 0 {
		t.Fatalf("exit code mismatch: got %d, want 0", code)
	}
}

func TestNewOperatorPrecedenceMixed(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	// 2 + 3 * 4 = 2 + 12 = 14
	src := "fn main() -> i32 { return 2 + 3 * 4 }"
	_, code := buildAndRun(t, src)
	if code != 14 {
		t.Fatalf("exit code mismatch: got %d, want 14", code)
	}
}

func TestExprStmt(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `fun side(): i32 { return 0 }
fun main(): i32 { side(); return 42 }`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestExprStmtQualified(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun noop(): i32 {\n  return 0\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as r\nfun main(): i32 {\n  r.noop()\n  return 42\n}\n",
	}
	_, code := buildAndRunFiles(t, files, "app/game.tetra")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildWASMHelloWritesModule(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join("..", "examples", "hello.tetra")

	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		outPath := filepath.Join(tmp, target+".wasm")
		if _, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1}); err != nil {
			t.Fatalf("build %s: %v", target, err)
		}
		data, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("read wasm: %v", err)
		}
		if len(data) < 8 {
			t.Fatalf("wasm too short: %d bytes", len(data))
		}
		if !bytes.Equal(data[:4], []byte{0x00, 0x61, 0x73, 0x6d}) {
			t.Fatalf("missing wasm magic: % x", data[:4])
		}
		if !bytes.Equal(data[4:8], []byte{0x01, 0x00, 0x00, 0x00}) {
			t.Fatalf("unexpected wasm version header: % x", data[4:8])
		}
		if target == "wasm32-web" {
			loaderPath := strings.TrimSuffix(outPath, ".wasm") + ".mjs"
			loaderRaw, err := os.ReadFile(loaderPath)
			if err != nil {
				t.Fatalf("read web loader: %v", err)
			}
			loader := string(loaderRaw)
			if !strings.Contains(loader, "tetra_web_v1") || !strings.Contains(loader, "tetra_main") {
				t.Fatalf("unexpected web loader content:\n%s", loader)
			}
		}
	}
}

func TestBuildWASMWebUIWritesSidecars(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "ui_web.tetra")
	src := `state CounterState:
    var count: Int = 0
    val title: String = "Wave 9 Web"

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> increment
    command increment:
        state.count = state.count + 1
    style width: Int = 320
    accessibility label: String = "Increment"

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "ui.wasm")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "wasm32-web", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build wasm32-web: %v", err)
	}
	uiJSON := strings.TrimSuffix(outPath, ".wasm") + ".ui.json"
	uiModule := strings.TrimSuffix(outPath, ".wasm") + ".ui.web.mjs"
	uiHTML := strings.TrimSuffix(outPath, ".wasm") + ".ui.html"

	jsonRaw, err := os.ReadFile(uiJSON)
	if err != nil {
		t.Fatalf("read ui json: %v", err)
	}
	if !strings.Contains(string(jsonRaw), `"schema": "tetra.ui.v1"`) || !strings.Contains(string(jsonRaw), "CounterView") {
		t.Fatalf("unexpected ui json:\n%s", string(jsonRaw))
	}
	moduleRaw, err := os.ReadFile(uiModule)
	if err != nil {
		t.Fatalf("read ui module: %v", err)
	}
	if !strings.Contains(string(moduleRaw), "mountTetraUI") {
		t.Fatalf("unexpected ui module:\n%s", string(moduleRaw))
	}
	htmlRaw, err := os.ReadFile(uiHTML)
	if err != nil {
		t.Fatalf("read ui html: %v", err)
	}
	if !strings.Contains(string(htmlRaw), ".ui.web.mjs") || !strings.Contains(string(htmlRaw), "runTetra") {
		t.Fatalf("unexpected ui html:\n%s", string(htmlRaw))
	}
}

func TestBuildNativeUIWritesShellSidecar(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "ui_native.tetra")
	src := `state ShellState:
    var toggles: Int = 0
    val title: String = "Wave 9 Native"

view ShellView(state: ShellState):
    bind toggles: Int = state.toggles
    event submit -> toggle
    command toggle:
        state.toggles = state.toggles + 1
    style width: Int = 80
    accessibility label: String = "Toggle"

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "ui-app")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build linux-x64: %v", err)
	}
	sidecarPath := outPath + ".ui.shell.txt"
	sidecar, err := os.ReadFile(sidecarPath)
	if err != nil {
		t.Fatalf("read native ui shell sidecar: %v", err)
	}
	if !strings.Contains(string(sidecar), "ShellView") || !strings.Contains(string(sidecar), "event submit -> toggle") || !strings.Contains(string(sidecar), "state.toggles = 1") {
		t.Fatalf("unexpected native ui sidecar:\n%s", string(sidecar))
	}
	tracePath := outPath + ".ui.shell.json"
	trace, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("read native ui shell json trace: %v", err)
	}
	for _, want := range []string{
		`"schema": "tetra.ui.native-shell.v1"`,
		`"runtime": "native shell command dispatch"`,
		`"widgets":`,
		`"kind": "value"`,
		`"binding": "toggles"`,
		`"kind": "action"`,
		`"event": "submit"`,
		`"state_field": "toggles"`,
		`"state_value": "1"`,
	} {
		if !strings.Contains(string(trace), want) {
			t.Fatalf("native ui shell json trace missing %q:\n%s", want, trace)
		}
	}
}

func TestBuildCacheSeparatesNativeDebugAndReleaseModes(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as r\nfun main(): i32 {\n  return r.add_one(41)\n}\n",
	}
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	baseOpt := BuildOptions{Jobs: 1}
	stats1, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", baseOpt)
	if err != nil {
		t.Fatalf("build1: %v", err)
	}
	testkit.AssertModules(t, stats1.CompiledModules, []string{"app.game", "engine.render"})
	if len(stats1.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first base build")
	}

	stats2, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", baseOpt)
	if err != nil {
		t.Fatalf("build2: %v", err)
	}
	if len(stats2.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on base cache hit")
	}
	testkit.AssertModules(t, stats2.CacheHits, []string{"app.game", "engine.render"})

	debugOpt := BuildOptions{Jobs: 1, DebugInfo: true}
	stats3, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", debugOpt)
	if err != nil {
		t.Fatalf("build3 debug: %v", err)
	}
	testkit.AssertModules(t, stats3.CompiledModules, []string{"app.game", "engine.render"})
	if len(stats3.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first debug build")
	}

	stats4, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", debugOpt)
	if err != nil {
		t.Fatalf("build4 debug: %v", err)
	}
	if len(stats4.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on debug cache hit")
	}
	testkit.AssertModules(t, stats4.CacheHits, []string{"app.game", "engine.render"})

	releaseOpt := BuildOptions{Jobs: 1, ReleaseOptimize: true}
	stats5, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", releaseOpt)
	if err != nil {
		t.Fatalf("build5 release: %v", err)
	}
	testkit.AssertModules(t, stats5.CompiledModules, []string{"app.game", "engine.render"})
	if len(stats5.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first release build")
	}

	stats6, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", releaseOpt)
	if err != nil {
		t.Fatalf("build6 release: %v", err)
	}
	if len(stats6.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on release cache hit")
	}
	testkit.AssertModules(t, stats6.CacheHits, []string{"app.game", "engine.render"})

	stats7, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", baseOpt)
	if err != nil {
		t.Fatalf("build7 base: %v", err)
	}
	if len(stats7.CompiledModules) != 0 {
		t.Fatalf("expected base mode cache to remain warm")
	}
	testkit.AssertModules(t, stats7.CacheHits, []string{"app.game", "engine.render"})
}

func TestBuildWASMCacheStatsRemainColdAcrossBuilds(t *testing.T) {
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		tmp := t.TempDir()
		files := map[string]string{
			"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
			"app/game.tetra":      "module app.game\nimport engine.render as r\nfun main(): i32 {\n  return r.add_one(41)\n}\n",
		}
		writeTestFiles(t, tmp, files)
		entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
		outPath := filepath.Join(tmp, "out", target+".wasm")
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		stats1, err := BuildFileWithStatsOpt(entry, outPath, target, BuildOptions{Jobs: 1})
		if err != nil {
			t.Fatalf("build1 %s: %v", target, err)
		}
		testkit.AssertModules(t, stats1.CompiledModules, []string{"app.game", "engine.render"})
		if len(stats1.CacheHits) != 0 {
			t.Fatalf("%s unexpected cache hits on first build: %#v", target, stats1.CacheHits)
		}

		stats2, err := BuildFileWithStatsOpt(entry, outPath, target, BuildOptions{Jobs: 1})
		if err != nil {
			t.Fatalf("build2 %s: %v", target, err)
		}
		testkit.AssertModules(t, stats2.CompiledModules, []string{"app.game", "engine.render"})
		if len(stats2.CacheHits) != 0 {
			t.Fatalf("%s expected cache to stay cold: %#v", target, stats2.CacheHits)
		}
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
	if err := BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func buildAndRunWithOptions(t *testing.T, src string, opt BuildOptions) (string, int) {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")

	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", opt); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func buildOnly(t *testing.T, src string) error {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")

	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	return BuildFile(srcPath, outPath, "linux-x64")
}

func buildAndRunFiles(t *testing.T, files map[string]string, entry string) (string, int) {
	t.Helper()

	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)

	entryPath := filepath.Join(tmp, filepath.FromSlash(entry))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := BuildFile(entryPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func buildOnlyFiles(t *testing.T, files map[string]string, entry string) error {
	t.Helper()

	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)

	entryPath := filepath.Join(tmp, filepath.FromSlash(entry))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	return BuildFile(entryPath, outPath, "linux-x64")
}

func writeTestFiles(t *testing.T, base string, files map[string]string) {
	t.Helper()

	for path, src := range files {
		full := filepath.Join(base, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatalf("write source: %v", err)
		}
	}
}

func TestBuildRejectsInterfaceOnlyDependencyWithoutInterfaceOnlyMode(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4":   "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n",
		"math/core.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "app"),
		"linux-x64",
		BuildOptions{Jobs: 1},
	)
	if err == nil {
		t.Fatalf("expected interface-only dependency build rejection")
	}
	if !strings.Contains(err.Error(), "missing implementation object for interface module 'math.core'") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildInterfaceOnlyModeAllowsT4IDependencyWithoutOutput(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4":   "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n",
		"math/core.t4i": string(iface),
	})

	outPath := filepath.Join(tmp, "out", "app")
	stats, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		outPath,
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only: %v", err)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("interface-only build should not emit %s, stat err=%v", outPath, err)
	}
	if len(stats.InterfaceModules) != 1 || stats.InterfaceModules[0] != "math.core" {
		t.Fatalf("InterfaceModules = %#v, want [math.core]", stats.InterfaceModules)
	}
}

func TestBuildInterfaceOnlyModePreservesTypedErrorResourceThrowProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub enum TaskErr:
    case wrap(task.i32)

pub func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    throw TaskErr.wrap(task)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return catch resources.fail(task):
    case resources.TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only typed-error resource provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesTypedErrorFieldLocalAliasResourceThrowProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub enum TaskErr:
    case wrap(task.i32)

pub func fail(box: TaskBox) -> Int throws TaskErr
uses runtime:
    let other: task.i32 = box.handle
    throw TaskErr.wrap(other)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    return catch resources.fail(box):
    case resources.TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only typed-error field local alias resource provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesTypedErrorResourceRethrowProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub enum TaskErr:
    case wrap(task.i32)

pub func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    throw TaskErr.wrap(task)

pub func wrapper(task: task.i32) -> Int throws TaskErr
uses runtime:
    return try fail(task)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return catch resources.wrapper(task):
    case resources.TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only typed-error rethrow resource provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesTypedErrorFieldLocalAliasResourceRethrowProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub enum TaskErr:
    case wrap(task.i32)

pub func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    throw TaskErr.wrap(task)

pub func wrapper(box: TaskBox) -> Int throws TaskErr
uses runtime:
    let other: task.i32 = box.handle
    return try fail(other)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    return catch resources.wrapper(box):
    case resources.TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only typed-error field local alias rethrow resource provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub enum TaskMsg:
    case wrap(task.i32)

pub func wrap(task: task.i32) -> TaskMsg:
    return TaskMsg.wrap(task)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: resources.TaskMsg = resources.wrap(task)
    match returned:
    case resources.TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub func maybe(task: task.i32) -> task.i32?:
    var out: task.i32? = none
    out = task
    return out
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: task.i32? = resources.maybe(task)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesLocalAliasResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub func alias(task: task.i32) -> task.i32:
    let other: task.i32 = task
    return other
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let other: task.i32 = resources.alias(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only local alias resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesAggregateLocalAliasResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func box(task: task.i32) -> TaskBox:
    let other: task.i32 = task
    return TaskBox(handle: other)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: resources.TaskBox = resources.box(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only aggregate local alias resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'returned.handle'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesAggregateFieldResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func pass(box: TaskBox) -> TaskBox:
    return TaskBox(handle: box.handle)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: resources.TaskBox = resources.pass(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only aggregate field resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'returned.handle'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesAggregateFieldLocalAliasResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func pass(box: TaskBox) -> TaskBox:
    let other: task.i32 = box.handle
    return TaskBox(handle: other)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: resources.TaskBox = resources.pass(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only aggregate field local alias resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'returned.handle'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesLetOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub func maybe(task: task.i32) -> task.i32?:
    let out: task.i32? = task
    return out
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: task.i32? = resources.maybe(task)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only let optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesOptionalFieldLocalAliasResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func maybe(box: TaskBox) -> task.i32?:
    let out: task.i32? = box.handle
    return out
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: task.i32? = resources.maybe(box)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only optional field local alias resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesDirectIfLetOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    maybe: task.i32?

pub func maybe(input: TaskBox) -> task.i32?:
    if let other = input.maybe:
        return other
    else:
        return none
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.TaskBox = resources.TaskBox(maybe: task)
    let returned: task.i32? = resources.maybe(input)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only direct if-let optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesDirectMatchOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    maybe: task.i32?

pub func maybe(input: TaskBox) -> task.i32?:
    match input.maybe:
    case some(other):
        return other
    case none:
        return none
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.TaskBox = resources.TaskBox(maybe: task)
    let returned: task.i32? = resources.maybe(input)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only direct match optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesStructOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    maybe: task.i32?

pub func box(task: task.i32) -> TaskBox:
    var out: task.i32? = none
    out = task
    return TaskBox(maybe: out)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: resources.TaskBox = resources.box(task)
    if let other = returned.maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only struct optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesStructOptionalFieldLocalAliasResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    maybe: task.i32?

pub struct InputBox:
    handle: task.i32

pub func box(input: InputBox) -> TaskBox:
    let out: task.i32? = input.handle
    return TaskBox(maybe: out)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.InputBox = resources.InputBox(handle: task)
    let returned: resources.TaskBox = resources.box(input)
    if let other = returned.maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only struct optional field local alias resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesIfLetOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct InputBox:
    maybe: task.i32?

pub struct TaskBox:
    maybe: task.i32?

pub func box(input: InputBox) -> TaskBox:
    if let other = input.maybe:
        return TaskBox(maybe: other)
    else:
        return TaskBox(maybe: none)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.InputBox = resources.InputBox(maybe: task)
    let returned: resources.TaskBox = resources.box(input)
    if let other = returned.maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only if-let optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesMatchOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct InputBox:
    maybe: task.i32?

pub struct TaskBox:
    maybe: task.i32?

pub func box(input: InputBox) -> TaskBox:
    match input.maybe:
    case some(other):
        return TaskBox(maybe: other)
    case none:
        return TaskBox(maybe: none)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.InputBox = resources.InputBox(maybe: task)
    let returned: resources.TaskBox = resources.box(input)
    if let other = returned.maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only match optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModeDoesNotRequireMain(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"math/core.t4": "module math.core\npub func add(a: Int, b: Int) -> Int:\n    return a + b\n",
	})

	outPath := filepath.Join(tmp, "out", "app")
	stats, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("math/core.t4")),
		outPath,
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only no main: %v", err)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("interface-only build should not emit %s, stat err=%v", outPath, err)
	}
	if len(stats.InterfaceModules) != 0 {
		t.Fatalf("InterfaceModules = %#v, want none for source-only graph", stats.InterfaceModules)
	}
}

func TestBuildInterfaceOnlyModeAcceptsGeneratedT4IWithImportedSignatureType(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module math.core

import math.types as mt

pub func norm(v: mt.Vec) -> Int:
    return v.x
`), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4":   "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return 0\n",
		"math/core.t4i": string(iface),
		"math/types.t4": "module math.types\npub struct Vec:\n    x: Int\n",
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only imported signature type: %v", err)
	}
}

func TestBuildInterfaceOnlyModeAcceptsGeneratedT4IWithStructReturnStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module math.core

pub struct Point:
    x: Int

pub func origin() -> Point:
    return Point(x: 0)
`), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4":   "module app.main\nimport math.core as math\nfunc main() -> Int:\n    math.origin()\n    return 0\n",
		"math/core.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only struct return stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeRejectsAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func make_pair(a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.make_pair(a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only aggregate region return escape diagnostic")
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsOptionalAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func maybe_pair(a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.maybe_pair(a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only optional aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsEnumPayloadRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum BufMsg:
    case both([]u8, []u8)
    case empty

pub func make_msg(a: island, b: island) -> BufMsg
uses alloc, islands, mem:
    return BufMsg.both(core.island_make_u8(a, 1), core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var msg: buffers.BufMsg = buffers.BufMsg.empty
    island(64) as a:
        island(64) as b:
            msg = buffers.make_msg(a, b)
    match msg:
    case buffers.BufMsg.both(left, right):
        return left[0]
    case buffers.BufMsg.empty:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only enum payload region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsOptionalEnumPayloadRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum BufMsg:
    case both([]u8, []u8)
    case empty

pub func maybe_msg(a: island, b: island) -> BufMsg?
uses alloc, islands, mem:
    var out: BufMsg? = none
    out = BufMsg.both(core.island_make_u8(a, 1), core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.BufMsg? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.maybe_msg(a, b)
    match maybe:
    case some(msg):
        match msg:
        case buffers.BufMsg.both(left, right):
            return left[0]
        case buffers.BufMsg.empty:
            return 0
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only optional enum payload region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsBranchAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool, a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    if flag:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    else:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.choose_pair(true, a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only branch aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsBranchOptionalAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool, a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    if flag:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    else:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.choose_pair(true, a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only branch optional aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsBranchOptionalMixedAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool, a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    if flag:
        out = PairBuf(left: make_u8(1), right: make_u8(1))
    else:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.choose_pair(false, a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only branch optional mixed aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsMatchAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum Mode:
    case fast
    case slow

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(mode: Mode, a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    match mode:
    case Mode.fast:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    case Mode.slow:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.choose_pair(buffers.Mode.fast, a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only match aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsMatchOptionalAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum Mode:
    case fast
    case slow

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(mode: Mode, a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    match mode:
    case Mode.fast:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    case Mode.slow:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.choose_pair(buffers.Mode.fast, a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only match optional aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsIfLetOptionalAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool?, a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    if let enabled = flag:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    else:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.choose_pair(true, a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only if-let optional aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsIfLetMixedAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool?, a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    if let enabled = flag:
        return PairBuf(left: make_u8(1), right: make_u8(1))
    else:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.choose_pair(none, a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only if-let mixed aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsMatchMixedAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum Mode:
    case fast
    case slow

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(mode: Mode, a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    match mode:
    case Mode.fast:
        return PairBuf(left: make_u8(1), right: make_u8(1))
    case Mode.slow:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.choose_pair(buffers.Mode.slow, a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only match mixed aggregate region return escape diagnostic\ninterface:\n%s", iface)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedParameterReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.identity

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`), "lib/identity.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.identity as id

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = id.identity(f)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/identity.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed parameter-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedParameterLocalAliasReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.identity

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    let alias: fn(Int) -> Int = f
    return alias
`), "lib/identity.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.identity as id

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = id.identity(f)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/identity.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed parameter local-alias return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedStructFieldReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub func pick(holder: Holder) -> fn(Int) -> Int:
    return holder.cb
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	libIface, err := ParseFile(iface, "lib/callbacks.t4i")
	if err != nil {
		t.Fatalf("ParseFile interface: %v\ninterface:\n%s", err, iface)
	}
	checkedIface, err := CheckWorldOpt(&World{
		EntryModule:      "lib.callbacks",
		Files:            []*FileAST{libIface},
		InterfaceModules: map[string]bool{"lib.callbacks": true},
		ByModule: map[string]*FileAST{
			"lib.callbacks": libIface,
		},
	}, CheckOptions{RequireMain: false})
	if err != nil {
		t.Fatalf("CheckWorld interface: %v\ninterface:\n%s", err, iface)
	}
	pickSig := checkedIface.FuncSigs["lib.callbacks.pick"]
	if got := pickSig.ReturnFunctionParamName; got != "holder.cb" {
		t.Fatalf("pick ReturnFunctionParamName = %q, want holder.cb; interface:\n%s", got, iface)
	}
	if len(pickSig.ParamTypes) != 1 || pickSig.ParamTypes[0] != "lib.callbacks.Holder" {
		t.Fatalf("pick ParamTypes = %#v, want lib.callbacks.Holder; interface:\n%s", pickSig.ParamTypes, iface)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let holder: callbacks.Holder = callbacks.Holder(cb: f)
    cb = callbacks.pick(holder)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed struct-field-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedNestedStructFieldReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func pick(box: Box) -> fn(Int) -> Int:
    return box.holder.cb
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let box: callbacks.Box = callbacks.Box(holder: callbacks.Holder(cb: f))
    cb = callbacks.pick(box)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed nested-struct-field-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedStructParameterWholeReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func echo(box: Box) -> Box:
    return box
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let box: callbacks.Box = callbacks.Box(holder: callbacks.Holder(cb: f))
    let returned: callbacks.Box = callbacks.echo(box)
    cb = returned.holder.cb
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed struct-parameter whole-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedEnumParameterWholeReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func echo(choice: MaybeCallback) -> MaybeCallback:
    return choice
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let choice: callbacks.MaybeCallback = callbacks.echo(callbacks.MaybeCallback.some(f))
    match choice:
    case callbacks.MaybeCallback.some(local):
        cb = local
        return 0
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed enum-parameter whole-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedEnumPayloadMatchReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func fallback(x: Int) -> Int:
    return x

pub func pick(choice: MaybeCallback) -> fn(Int) -> Int:
    match choice:
    case some(local):
        return local
    case empty:
        return fallback
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = callbacks.pick(callbacks.MaybeCallback.some(f))
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed enum-payload match return global escape diagnostic\ninterface:\n%s", iface)
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedAggregateClosurePayloadStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        cb = local
        return 0
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned aggregate closure stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedEnumClosurePayloadStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        cb = local
        return 0
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned enum closure stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller() -> Int throws callbacks.Boom:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        return try local(41)
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned throwing aggregate closure stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller() -> Int throws callbacks.Boom:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        return try local(41)
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned throwing enum closure stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadRequiresTryDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        return local(41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only returned throwing aggregate closure payload requires-try diagnostic")
	}
	want := "call to throwing function 'local' requires try"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadRequiresTryDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        return local(41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only returned throwing enum closure payload requires-try diagnostic")
	}
	want := "call to throwing function 'local' requires try"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller() -> Int throws callbacks.Boom:
    let holder: callbacks.Holder = callbacks.makeHolder()
    return try holder.cb(41)

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned throwing struct-field closure stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureRequiresTryDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let holder: callbacks.Holder = callbacks.makeHolder()
    return holder.cb(41)
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only returned throwing struct-field closure requires-try diagnostic")
	}
	want := "call to throwing function 'holder.cb' requires try"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureCallbackStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int throws callbacks.Boom, x: Int) -> Int throws callbacks.Boom:
    return try f(x)

func caller() -> Int throws callbacks.Boom:
    let holder: callbacks.Holder = callbacks.makeHolder()
    return try apply(holder.cb, 41)

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned throwing struct-field closure callback stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureCallbackThrowsMismatchDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    let holder: callbacks.Holder = callbacks.makeHolder()
    return apply(holder.cb, 41)
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only returned throwing struct-field closure callback throws mismatch diagnostic")
	}
	want := "callback function symbol 'holder.cb' throws type mismatch: expected '', got 'lib.callbacks.Boom'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadCallbackStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int throws callbacks.Boom, x: Int) -> Int throws callbacks.Boom:
    return try f(x)

func caller() -> Int throws callbacks.Boom:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        return try apply(local, 41)
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned throwing aggregate closure callback stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadCallbackStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int throws callbacks.Boom, x: Int) -> Int throws callbacks.Boom:
    return try f(x)

func caller() -> Int throws callbacks.Boom:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        return try apply(local, 41)
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned throwing enum closure callback stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadCallbackThrowsMismatchDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        return apply(local, 41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only returned throwing enum closure callback throws mismatch diagnostic")
	}
	want := "callback function symbol 'local' throws type mismatch: expected '', got 'lib.callbacks.Boom'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadCallbackThrowsMismatchDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        return apply(local, 41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only returned throwing aggregate closure callback throws mismatch diagnostic")
	}
	want := "callback function symbol 'local' throws type mismatch: expected '', got 'lib.callbacks.Boom'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func runBinary(t *testing.T, path string) (string, int) {
	t.Helper()

	cmd := exec.Command(path)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return out.String(), exitErr.ProcessState.ExitCode()
		}
		t.Fatalf("run binary: %v", err)
	}
	return out.String(), cmd.ProcessState.ExitCode()
}

func verifyELF(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	hdr := make([]byte, 64)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return err
	}
	if !bytes.Equal(hdr[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		return fmt.Errorf("missing ELF magic")
	}
	if hdr[4] != 2 {
		return fmt.Errorf("expected ELF64")
	}
	if hdr[5] != 1 {
		return fmt.Errorf("expected little-endian")
	}
	eType := binary.LittleEndian.Uint16(hdr[16:18])
	eMachine := binary.LittleEndian.Uint16(hdr[18:20])
	entry := binary.LittleEndian.Uint64(hdr[24:32])
	if eType != 2 {
		return fmt.Errorf("expected ET_EXEC")
	}
	if eMachine != 0x3e {
		return fmt.Errorf("expected x86_64 machine")
	}
	if entry == 0 {
		return fmt.Errorf("entrypoint is zero")
	}
	return nil
}
