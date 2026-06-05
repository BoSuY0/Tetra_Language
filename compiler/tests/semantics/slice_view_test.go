package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func TestSliceViewConstructorsTypeCheckForSupportedSlices(t *testing.T) {
	testkit.RequireCheckOK(t, `
func main() -> Int
uses alloc, mem:
    var bytes: []u8 = make_u8(4)
    var words: []u16 = make_u16(4)
    var nums: []i32 = make_i32(4)
    var flags: []bool = make_bool(4)
    let b0: []u8 = bytes.window(0, bytes.len)
    let b1: []u8 = bytes.prefix(2)
    let b2: []u8 = bytes.suffix(1)
    let w0: []u16 = words.window(1, 2)
    let n0: []i32 = nums.prefix(3)
    let f0: []bool = flags.suffix(2)
    return b0.len + b1.len + b2.len + w0.len + n0.len + f0.len
`)
}

func TestSliceViewConstructorMetadataAssignmentStillRejects(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Box<T>:
    value: T

enum BufferMsg:
    case bytes([]u8)

func main() -> Int
uses alloc, mem:
    var box: Box<[]u8> = Box<[]u8>{value: make_u8(2)}
    let nested: []u8 = box.value.window(0, 1)
    nested.ptr = 0
    return 0
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")

	testkit.RequireCheckErrorContains(t, `
enum BufferMsg:
    case bytes([]u8)

func main() -> Int
uses alloc, mem:
    var msg: BufferMsg = BufferMsg.bytes(make_u8(2))
    match msg:
        case BufferMsg.bytes(xs):
            let view: []u8 = xs.prefix(1)
            view.len = 9
    return 0
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")

	testkit.RequireCheckErrorContains(t, `
func main() -> Int
uses alloc, mem:
    var maybe: []u8? = make_u8(2)
    if let some(xs) = maybe:
        let view: []u8 = xs.suffix(1)
        view.ptr = 0
    return 0
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")
}

func TestSliceMetadataAssignmentRejectsGenericNestedLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Box<T>:
    value: T

func main() -> Int
uses alloc, mem:
    var box: Box<[]u8> = Box<[]u8>{value: make_u8(2)}
    box.value.len = 9
    return 0
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")
}

func TestSliceMetadataAssignmentRejectsInoutParameterLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func poke(xs: inout []u8) -> Int:
    xs.len = 9
    return 0

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(2)
    return poke(xs)
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")
}

func TestResolveAssignTargetRejectsSliceMetadataPathBeforeIndexing(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(2)
    xs.ptr[0] = 1
    return 0
`, "cannot assign to slice internals ('ptr'/'len'); assign elements via index instead")
}

func TestBuildSliceWindowPrefixSuffixSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `fun main(): i32 uses alloc, mem {
  var xs: []u8 = make_u8(4)
  xs[0] = 10
  xs[1] = 20
  xs[2] = 22
  xs[3] = 99
  let all: []u8 = xs.window(0, xs.len)
  let mid: []u8 = xs.window(1, 2)
  let pre: []u8 = xs.prefix(3)
  let suf: []u8 = xs.suffix(2)
  return all.len + mid[0] + mid[1] + pre[2] + suf[0] - 48
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

func TestBuildSliceViewConstructorsAllElementKindsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `fun main(): i32 uses alloc, mem {
  var nums: []i32 = make_i32(3)
  nums[0] = 4
  nums[1] = 40
  nums[2] = 100
  var words: []u16 = make_u16(2)
  words[0] = 1
  words[1] = 2
  var flags: []bool = make_bool(2)
  flags[0] = false
  flags[1] = true
  let a: []i32 = nums.window(1, 1)
  let b: []u16 = words.suffix(1)
  let c: []bool = flags.prefix(2)
  if c[1] {
    return a[0] + b[0]
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

func TestSliceWindowRejectsInvalidRangesBeforeConstruction(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tests := []struct {
		name string
		expr string
	}{
		{name: "negative_start", expr: "xs.window(-1, 1)"},
		{name: "negative_count", expr: "xs.window(0, -1)"},
		{name: "start_past_len", expr: "xs.window(xs.len + 1, 0)"},
		{name: "count_past_tail", expr: "xs.window(1, xs.len)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := `fun main(): i32 uses alloc, mem {
  var xs: []u8 = make_u8(2)
  let bad: []u8 = ` + tc.expr + `
  return bad.len
}
`
			stdout, exitCode := buildAndRun(t, src)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode == 0 || exitCode == 42 {
				t.Fatalf("invalid window exited %d, want trap/non-success", exitCode)
			}
		})
	}
}

func TestSliceViewConstructorsWasmBuildOnly(t *testing.T) {
	src := `func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    let mid: []u8 = xs.window(1, 2)
    let pre: []u8 = xs.prefix(2)
    let suf: []u8 = xs.suffix(1)
    return mid.len + pre.len + suf.len
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
