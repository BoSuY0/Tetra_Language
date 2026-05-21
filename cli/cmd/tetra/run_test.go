package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

func TestRunCommandJSONDiagnosticsForHostTargetMismatch(t *testing.T) {
	target := nonHostTarget(t)
	diag := runCLIJSONDiagnostic(t, []string{"run", "--diagnostics=json", "--target", target}, 2)
	if diag.Code != "TETRA0001" || diag.Severity != "error" || !strings.Contains(diag.Message, "cannot run target "+target) {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestRunCommandJSONDiagnosticsForWASMWebRuntimeUnsupported(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	restore := stubLookPath(func(name string) (string, error) {
		return "", exec.ErrNotFound
	})
	defer restore()

	diag := runCLIJSONDiagnostic(t, []string{"run", "--diagnostics=json", "--target", "wasm32-web", srcPath}, 1)
	for _, want := range []string{"cannot run target wasm32-web", "browser runner unavailable"} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
}

func TestRunCommandUsesBrowserRunnerForWASMWeb(t *testing.T) {
	requireLocalTCPBind(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 7\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	browser := filepath.Join(dir, "fake-chromium")
	if err := os.WriteFile(browser, []byte(`#!/bin/sh
printf '<html><body><pre id="result">exit:7</pre></body></html>\n'
`), 0o755); err != nil {
		t.Fatalf("write fake browser: %v", err)
	}
	restore := stubLookPath(func(name string) (string, error) {
		if name == "chromium" {
			return browser, nil
		}
		return "", exec.ErrNotFound
	})
	defer restore()

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "wasm32-web", srcPath}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandJSONDiagnosticsForLinuxX32HostUnsupported(t *testing.T) {
	restore := stubLinuxX32HostSupport(false)
	defer restore()

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"run", "--diagnostics=json", "--target", "x32", srcPath}, 2)
	for _, want := range []string{"cannot run target linux-x32", "host does not support Linux x32 ABI execution", "no host fallback"} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("diagnostic missing %q: %#v", want, diag)
		}
	}
}

func TestRunCommandUsesLinuxX32HostRunnerWhenProbePasses(t *testing.T) {
	restoreHost := stubLinuxX32HostSupport(true)
	defer restoreHost()
	restoreExec := stubNativeExec(func(path string, stdout io.Writer, stderr io.Writer) int {
		if err := requireX32ExecutableFile(path); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 42
	})
	defer restoreExec()

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x32", srcPath}, &stdout, &stderr)
	if code != 42 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestExecWebProgramWithBrowserRunnerParsesBrowserExitResult(t *testing.T) {
	requireLocalTCPBind(t)

	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "app.wasm")
	if err := os.WriteFile(wasmPath, []byte("\x00asm\x01\x00\x00\x00"), 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.mjs"), []byte("export async function runTetra() { return 7; }\n"), 0o644); err != nil {
		t.Fatalf("write loader: %v", err)
	}
	browser := filepath.Join(dir, "fake-chromium")
	if err := os.WriteFile(browser, []byte(`#!/bin/sh
printf '<html><body><pre id="result">exit:7</pre></body></html>\n'
`), 0o755); err != nil {
		t.Fatalf("write fake browser: %v", err)
	}

	exit, err := execWebProgramWithBrowserRunner(wasmPath, browser, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("execWebProgramWithBrowserRunner: %v", err)
	}
	if exit != 7 {
		t.Fatalf("exit = %d, want 7", exit)
	}
}

func requireX32ExecutableFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 20 {
		return fmt.Errorf("x32 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x32 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		return fmt.Errorf("x32 executable class = %d, want ELFCLASS32", data[4])
	}
	if machine := binary.LittleEndian.Uint16(data[18:20]); machine != 0x3e {
		return fmt.Errorf("x32 executable machine = %#x, want EM_X86_64", machine)
	}
	return nil
}

func requireLocalTCPBind(t *testing.T) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local TCP bind unavailable in this environment: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close local TCP probe: %v", err)
	}
}

func TestRunCommandPropagatesProgramExitCode(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", mustHostTarget(t), srcPath}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86NoRuntimeExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86FunctionArgumentExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "func add(a: Int, b: Int) -> Int:\n    return a + b\n\nfunc main() -> Int:\n    return add(40, 2)\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 42 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86GlobalExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "var answer: Int = 1\n\nfunc main() -> Int:\n    answer = 42\n    return answer\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 42 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86DirectCallbackExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "func add1(x: Int) -> Int:\n    return x + 1\n\nfunc apply(cb: fn(Int) -> Int, x: Int) -> Int:\n    return cb(x)\n\nfunc main() -> Int:\n    return apply(add1, 41)\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 42 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86MakeI32SliceExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "fun main(): i32 uses alloc, mem {\n  var xs: []i32 = make_i32(3)\n  xs[0] = 10\n  xs[1] = 20\n  xs[2] = xs[0] + xs[1]\n  return xs[2]\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 30 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86AllocBytesZeroExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "fun main(): i32 uses alloc, mem {\n  unsafe {\n    let _p: ptr = core.alloc_bytes(0)\n    return 0\n  }\n  return 0\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86RawStoreLoadExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "func main() -> Int\nuses alloc, capability, mem:\n  unsafe:\n    let mem: cap.mem = core.cap_mem()\n    let p: ptr = core.alloc_bytes(4)\n    let _: Int = core.store_i32(p, 42, mem)\n    return core.load_i32(p, mem)\n  return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 42 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86RawPtrAddU8ExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "func main() -> Int\nuses alloc, capability, mem:\n  unsafe:\n    let mem: cap.mem = core.cap_mem()\n    let p: ptr = core.alloc_bytes(4)\n    let _: UInt8 = core.store_u8(core.ptr_add(p, 1, mem), 7, mem)\n    return core.load_u8(core.ptr_add(p, 1, mem), mem)\n  return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86RawPtrAddUpperBoundExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "func main() -> Int\nuses alloc, capability, mem:\n  unsafe:\n    let mem: cap.mem = core.cap_mem()\n    let p: ptr = core.alloc_bytes(4)\n    let q: ptr = core.ptr_add(p, 4, mem)\n    let _: UInt8 = core.store_u8(q, 7, mem)\n    return 0\n  return 0\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86PrintStringStdout(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "fun main(): i32 uses io {\n  print(\"x86 says hi\\n\")\n  return 0\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "x86 says hi\n" {
		t.Fatalf("run stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86PrintSliceStdout(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "fun main(): i32 uses alloc, io, mem {\n  var xs: []u8 = make_u8(2)\n  xs[0] = 65\n  xs[1] = 66\n  print(xs)\n  return 0\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "AB" {
		t.Fatalf("run stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86ScopedIslandExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "fun main(): i32 uses alloc, islands, mem {\n  var out: i32 = 0\n  island(64) as isl {\n    var xs: []u8 = core.island_make_u8(isl, 1)\n    xs[0] = 7\n    out = xs[0]\n  }\n  return out\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86ScopedIslandDebugExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "fun main(): i32 uses alloc, islands, mem {\n  var out: i32 = 0\n  island(64) as isl {\n    var xs: []u8 = core.island_make_u8(isl, 1)\n    xs[0] = 7\n    out = xs[0]\n  }\n  return out\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", "--islands-debug", srcPath}, &stdout, &stderr)
	if code != 7 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86ScopedIslandOverflowExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "fun main(): i32 uses alloc, islands, mem {\n  island(16) as isl {\n    var xs: []u8 = core.island_make_u8(isl, 17)\n    xs[0] = 1\n  }\n  return 0\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86MMIOExitCode(t *testing.T) {
	requireLinuxX86Execution(t)

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	src := "fun main(): i32 uses alloc, capability, io, mem, mmio {\n  var out: i32 = 0\n  unsafe {\n    let io: cap.io = core.cap_io()\n    let p: ptr = core.alloc_bytes(4)\n    let _w: i32 = core.mmio_write_i32(p, 123, io)\n    out = core.mmio_read_i32(p, io)\n  }\n  return out\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
	if code != 123 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run stderr = %q", stderr.String())
	}
}

func TestRunCommandPropagatesLinuxX86RuntimeMatrixExitCodes(t *testing.T) {
	requireLinuxX86Execution(t)

	tests := []struct {
		name string
		src  string
		want int
	}{
		{
			name: "recursion",
			src:  "func fact(n: Int) -> Int:\n    if n <= 1:\n        return 1\n    return n * fact(n - 1)\n\nfunc main() -> Int:\n    return fact(5)\n",
			want: 120,
		},
		{
			name: "while_loop",
			src:  "func main() -> Int:\n    var i: Int = 0\n    var acc: Int = 0\n    while i < 6:\n        acc = acc + i\n        i = i + 1\n    return acc\n",
			want: 15,
		},
		{
			name: "struct_fields",
			src:  "struct Pair:\n    left: Int\n    right: Int\n\nfunc main() -> Int:\n    let p: Pair = Pair(left: 19, right: 23)\n    return p.left + p.right\n",
			want: 42,
		},
		{
			name: "enum_payload_match",
			src:  "enum Msg:\n    case left(Int)\n    case right(Int)\n\nfunc choose(flag: Int) -> Msg:\n    if flag:\n        return Msg.left(40)\n    return Msg.right(2)\n\nfunc main() -> Int:\n    let msg: Msg = choose(0)\n    match msg:\n    case Msg.left(value):\n        return value\n    case Msg.right(value):\n        return value + 40\n",
			want: 42,
		},
		{
			name: "u16_slice",
			src:  "fun main(): i32 uses alloc, mem {\n  var xs: []u16 = make_u16(2)\n  xs[0] = 40\n  xs[1] = 2\n  return xs[0] + xs[1]\n}\n",
			want: 42,
		},
		{
			name: "bool_slice",
			src:  "func main() -> Int\nuses alloc, mem:\n    var flags: []bool = make_bool(2)\n    flags[0] = true\n    flags[1] = false\n    if flags[0]:\n        return 42\n    return 1\n",
			want: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "main.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{"run", "--target", "x86", srcPath}, &stdout, &stderr)
			if code != tt.want {
				t.Fatalf("run exit code = %d, want %d, stdout=%q stderr=%q", code, tt.want, stdout.String(), stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("run stderr = %q", stderr.String())
			}
		})
	}
}

func TestRunCommandWithoutOutputDoesNotLeaveDefaultBinary(t *testing.T) {
	if _, ok := hostTarget(); !ok {
		t.Skip("host target unsupported")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"run", "--target", mustHostTarget(t), srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	tgt, err := ctarget.Parse(mustHostTarget(t))
	if err != nil {
		t.Fatal(err)
	}
	defaultPath := filepath.Join(dir, defaultOutput(tgt, "exe"))
	if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
		t.Fatalf("run without -o should not leave %s, stat err=%v", defaultPath, err)
	}
}

func requireLinuxX86Execution(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "386") {
		t.Skipf("linux-x86 execution requires a Linux i386-compatible host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "probe.tetra")
	outPath := filepath.Join(dir, "probe")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "x86"); err != nil {
		t.Fatalf("build linux-x86 execution probe: %v", err)
	}
	var stderr bytes.Buffer
	code := execProgram(outPath, &bytes.Buffer{}, &stderr)
	if code == 7 {
		return
	}
	if strings.Contains(stderr.String(), "exec format error") || strings.Contains(stderr.String(), "no such file or directory") {
		t.Skipf("Linux kernel cannot execute generated i386 ELF on this host: exit=%d stderr=%q", code, stderr.String())
	}
	t.Fatalf("linux-x86 execution probe exit=%d stderr=%q", code, stderr.String())
}
