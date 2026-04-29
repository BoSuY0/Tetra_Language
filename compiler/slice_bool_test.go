package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSliceBoolSemanticsAcceptance(t *testing.T) {
	requireCheckOK(t, `
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
	requireCheckErrorContains(t, `
func main() -> Int
uses alloc, mem:
    var xs: []bool = make_bool(1)
    xs[0] = 1
    return 0
`, "type mismatch: expected 'bool', got 'i32'")
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
		if _, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1}); err != nil {
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
		if _, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1}); err != nil {
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
		if _, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1}); err != nil {
			t.Fatalf("build %s: %v", target, err)
		}
	}
}
