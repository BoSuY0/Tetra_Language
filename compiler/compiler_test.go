package compiler

import (
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

func TestBuildMakeZeroLengthSlices(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "module test.zero_length_slices\n\nfun count_i32(values: []i32) -> Int uses mem {\n  var count: Int = 0\n  for value in values {\n    count = count + 1\n  }\n  return count\n}\nfun first_or_i32(values: []i32, fallback: Int) -> Int uses mem {\n  for value in values {\n    return value\n  }\n  return fallback\n}\nfun main(): i32 uses alloc, mem {\n  var bytes: []u8 = make_u8(0)\n  for byte in bytes {\n    return 1\n  }\n  var words: []u16 = make_u16(0)\n  for word in words {\n    return 2\n  }\n  var ints: []i32 = make_i32(0)\n  for value in ints {\n    return 3\n  }\n  if count_i32(ints) != 0 {\n    return 6\n  }\n  if first_or_i32(ints, 42) != 42 {\n    return 7\n  }\n  var flags: []bool = make_bool(0)\n  for flag in flags {\n    if flag {\n      return 4\n    }\n    return 5\n  }\n  return 42\n}\n"
	stdout, exitCode := buildAndRunFiles(t, map[string]string{
		"test/zero_length_slices.tetra": src,
	}, "test/zero_length_slices.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
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

func TestBuildCoreCollectionsGenericCacheKeyIncludesMonomorphizedFuncs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))
	writeTestFiles(t, tmp, map[string]string{
		"app/main.tetra": `module app.main
import lib.core.collections as collections

func main() -> Int
uses alloc, mem:
    var xs: []i32 = core.make_i32(1)
    xs[0] = 42
    return collections.len_i32(xs)
`,
	})
	opt := BuildOptions{
		ProjectRoot:     tmp,
		DependencyRoots: []ModuleRoot{{Root: projectRoot(t)}},
	}
	if _, err := BuildFileWithStatsOpt(entry, filepath.Join(tmp, "plain"), "linux-x64", opt); err != nil {
		t.Fatalf("plain collections build: %v", err)
	}

	writeTestFiles(t, tmp, map[string]string{
		"app/main.tetra": `module app.main
import lib.core.collections as collections

func main() -> Int
uses alloc, mem:
    var xs: []i32 = core.make_i32(1)
    xs[0] = 42
    let vec: collections.Vec<Int> = collections.vec_from_slice(xs)
    if collections.vec_len(vec) == 1:
        return 42
    return 0
`,
	})
	outPath := filepath.Join(tmp, "generic")
	if _, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", opt); err != nil {
		t.Fatalf("generic collections build after plain cache entry: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	_, exitCode := runBinary(t, outPath)
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}
