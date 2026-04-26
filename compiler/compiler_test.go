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
)

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

func TestBuildIslandsDebugDoubleFree(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 uses alloc, islands, mem {\n  unsafe {\n    let isl: island = core.island_new(64)\n    free(isl)\n    free(isl)\n  }\n  return 0\n}\n"
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Jobs: 1, IslandsDebug: true})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
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

func TestEnumMatchMissingCaseStillNeedsReturn(t *testing.T) {
	src := "enum Color:\n  case red\n  case green\n\nfunc main() -> Int:\n  let color: Color = Color.green\n  match color:\n  case Color.red:\n    return 1\n"
	if err := checkProgram(src); err == nil {
		t.Fatalf("expected missing return for non-exhaustive enum match")
	} else if !strings.Contains(err.Error(), "must end with return") {
		t.Fatalf("error = %v", err)
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
	assertModules(t, stats1.CompiledModules, []string{"app.game", "engine.render"})
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
	assertModules(t, stats2.CacheHits, []string{"app.game", "engine.render"})

	debugOpt := BuildOptions{Jobs: 1, DebugInfo: true}
	stats3, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", debugOpt)
	if err != nil {
		t.Fatalf("build3 debug: %v", err)
	}
	assertModules(t, stats3.CompiledModules, []string{"app.game", "engine.render"})
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
	assertModules(t, stats4.CacheHits, []string{"app.game", "engine.render"})

	releaseOpt := BuildOptions{Jobs: 1, ReleaseOptimize: true}
	stats5, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", releaseOpt)
	if err != nil {
		t.Fatalf("build5 release: %v", err)
	}
	assertModules(t, stats5.CompiledModules, []string{"app.game", "engine.render"})
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
	assertModules(t, stats6.CacheHits, []string{"app.game", "engine.render"})

	stats7, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", baseOpt)
	if err != nil {
		t.Fatalf("build7 base: %v", err)
	}
	if len(stats7.CompiledModules) != 0 {
		t.Fatalf("expected base mode cache to remain warm")
	}
	assertModules(t, stats7.CacheHits, []string{"app.game", "engine.render"})
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
		assertModules(t, stats1.CompiledModules, []string{"app.game", "engine.render"})
		if len(stats1.CacheHits) != 0 {
			t.Fatalf("%s unexpected cache hits on first build: %#v", target, stats1.CacheHits)
		}

		stats2, err := BuildFileWithStatsOpt(entry, outPath, target, BuildOptions{Jobs: 1})
		if err != nil {
			t.Fatalf("build2 %s: %v", target, err)
		}
		assertModules(t, stats2.CompiledModules, []string{"app.game", "engine.render"})
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
