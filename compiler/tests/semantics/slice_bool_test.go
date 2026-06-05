package compiler_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func TestSliceBoolSemanticsAcceptance(t *testing.T) {
	testkit.RequireCheckOK(t, `
func main() -> Int
uses alloc, islands, mem:
    var xs: []bool = make_bool(2)
    xs[0] = true
    xs[1] = false
    island(128) as isl:
        var ys: []bool = core.island_make_bool(isl, 2)
        ys[0] = xs[0]
        ys[1] = xs[1]
    return 0
`)
}

func TestSliceBoolSemanticsRejectWrongElementType(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int
uses alloc, mem:
    var xs: []bool = make_bool(1)
    xs[0] = 1
    return 0
`, "type mismatch: expected 'bool', got 'i32'")
}

func TestSliceMetadataAssignmentRejectsLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int
uses alloc, mem:
    var bytes: []u8 = make_u8(1)
    bytes.len = 64
    return 0
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")
}

func TestSliceMetadataAssignmentRejectsPtr(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int
uses alloc, mem:
    var tiny: []u8 = make_u8(1)
    var wide: []u8 = make_u8(64)
    wide.ptr = tiny.ptr
    return 0
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")
}

func TestSliceRepresentationMetadataRegistryRejectsReservedNames(t *testing.T) {
	for _, field := range []string{
		"owner_id",
		"region_id",
		"provenance_id",
		"borrow_source",
		"storage_class",
		"unsafe_class",
	} {
		t.Run(field, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, fmt.Sprintf(`
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(1)
    xs.%s = 1
    return 0
`, field), "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")
		})
	}
}

func TestSliceMetadataAssignmentRejectsNestedLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Box:
    bytes: []u8

func main() -> Int
uses alloc, mem:
    var box: Box = Box(bytes: make_u8(1))
    box.bytes.len = 64
    return 0
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")
}

func TestSliceMetadataAssignmentRejectsNestedPtr(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Box:
    bytes: []u8

func main() -> Int
uses alloc, mem:
    var box: Box = Box(bytes: make_u8(1))
    box.bytes.ptr = 0
    return 0
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")
}

func TestSliceMetadataAssignmentRejectsGenericNestedPtr(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Box<T>:
    value: T

func main() -> Int
uses alloc, mem:
    var box: Box<[]u8> = Box<[]u8>{value: make_u8(1)}
    box.value.ptr = 0
    return 0
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")
}

func TestSliceMetadataAssignmentRejectsInoutLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func mutate(xs: inout []u8) -> Int:
    xs.len = 64
    return 0
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")
}

func TestSliceMetadataAssignmentRejectsOptionalPayloadLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int
uses alloc, mem:
    var maybe: []u8? = make_u8(1)
    if let some(xs) = maybe:
        xs.len = 64
    return 0
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")
}

func TestSliceMetadataAssignmentRejectsEnumPayloadPtr(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum BufferMsg:
    case bytes([]u8)

func main() -> Int
uses alloc, mem:
    var msg: BufferMsg = BufferMsg.bytes(make_u8(1))
    match msg:
        case BufferMsg.bytes(xs):
            xs.ptr = 0
    return 0
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")
}

func TestRawSliceFromPartsRequiresUnsafe(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func forge(p: ptr, n: Int, mem: cap.mem) -> []u8
uses capability, mem:
    return core.raw_slice_u8_from_parts(p, n, mem)
`, "'core.raw_slice_u8_from_parts' is only allowed in unsafe blocks")
}

func TestRawSliceFromPartsUnsafeGatewayTypeChecks(t *testing.T) {
	testkit.RequireCheckOK(t, `
func main() -> Int
uses alloc, capability, mem:
    var xs: []u8 = make_u8(1)
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let ys: []u8 = core.raw_slice_u8_from_parts(xs.ptr, xs.len, mem)
        return ys.len
    return 0
`)
}

func TestBuildRawSliceFromPartsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `fun main(): i32 uses alloc, capability, mem {
  var xs: []u8 = make_u8(2)
  xs[0] = 41
  xs[1] = 1
  unsafe {
    let mem: cap.mem = core.cap_mem()
    let ys: []u8 = core.raw_slice_u8_from_parts(xs.ptr, xs.len, mem)
    return ys[0] + ys[1]
  }
  return 0
}
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMakeBoolSliceSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `fun main(): i32 uses alloc, mem {
  var xs: []bool = make_bool(2)
  xs[0] = true
  xs[1] = false
  if (xs[0] && (!xs[1])) {
    return 42
  }
  return 0
}
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildIslandMakeBoolSliceSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `fun main(): i32 uses alloc, islands, mem {
  island(128) as isl {
    var xs: []bool = core.island_make_bool(isl, 1)
    xs[0] = true
    if (xs[0]) {
      return 42
    }
  }
  return 0
}
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestSliceBoolWasmBuildOnlyMakeBoolSmoke(t *testing.T) {
	src := `func main() -> Int
uses alloc, mem:
    var xs: []bool = make_bool(2)
    xs[0] = true
    xs[1] = false
    if xs[0]:
        return 42
    return 0
`
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		outPath := filepath.Join(tmp, "app-"+target+".wasm")
		if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, target, compiler.BuildOptions{Jobs: 1}); err != nil {
			t.Fatalf("build %s: %v", target, err)
		}
	}
}

func TestSliceBoolWasmBuildOnlyIslandMakeBoolSmoke(t *testing.T) {
	src := `func main() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var xs: []bool = core.island_make_bool(isl, 1)
        xs[0] = true
    return 0
`
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		outPath := filepath.Join(tmp, "app-"+target+".wasm")
		if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, target, compiler.BuildOptions{Jobs: 1}); err != nil {
			t.Fatalf("build %s: %v", target, err)
		}
	}
}

func TestSliceWasmBuildOnlyIslandMakeU8I32Smoke(t *testing.T) {
	src := `func main() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var bytes: []u8 = core.island_make_u8(isl, 2)
        bytes[0] = 1
        bytes[1] = 2
        var nums: []i32 = core.island_make_i32(isl, 1)
        nums[0] = 42
    return 0
`
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		outPath := filepath.Join(tmp, "app-"+target+".wasm")
		if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, target, compiler.BuildOptions{Jobs: 1}); err != nil {
			t.Fatalf("build %s: %v", target, err)
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
	if err := compiler.BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
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
