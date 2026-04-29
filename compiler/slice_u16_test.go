package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSliceU16SemanticsAcceptance(t *testing.T) {
	requireCheckOK(t, `
func main() -> Int
uses alloc, islands, mem:
    var xs: []u16 = make_u16(2)
    xs[0] = 7
    xs[1] = 35
    island(128) as isl:
        var ys: []u16 = core.island_make_u16(isl, 2)
        ys[0] = xs[0]
        ys[1] = xs[1]
    return 0
`)
}

func TestBuildMakeU16SliceSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `fun main(): i32 uses alloc, mem {
  var xs: []u16 = make_u16(3)
  xs[0] = 10
  xs[1] = 20
  xs[2] = xs[0] + xs[1]
  return xs[2]
}
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 30 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildIslandMakeU16SliceSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `fun main(): i32 uses alloc, islands, mem {
  var out: i32 = 0
  island(128) as isl {
    var xs: []u16 = core.island_make_u16(isl, 2)
    xs[0] = 40
    xs[1] = 2
    out = xs[0] + xs[1]
  }
  return out
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

func TestSliceU16WasmBuildOnlyIslandMakeU16Smoke(t *testing.T) {
	src := `func main() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var xs: []u16 = core.island_make_u16(isl, 2)
        xs[0] = 40
        xs[1] = 2
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
