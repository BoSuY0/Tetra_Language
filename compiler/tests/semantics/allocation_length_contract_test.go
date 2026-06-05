package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	compiler "tetra_language/compiler"
)

func TestBuildMakeConstructorsZeroLengthAndZeroIterationLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `fun main(): i32 uses alloc, islands, mem {
  var loops = 0
  var bytes: []u8 = make_u8(0)
  var words: []u16 = make_u16(0)
  var nums: []i32 = make_i32(0)
  var flags: []bool = make_bool(0)
  for b in bytes { loops = loops + 1 }
  for w in words { loops = loops + 1 }
  for n in nums { loops = loops + 1 }
  for f in flags { loops = loops + 1 }
  island(64) as isl {
    var ibytes: []u8 = core.island_make_u8(isl, 0)
    var iwords: []u16 = core.island_make_u16(isl, 0)
    var inums: []i32 = core.island_make_i32(isl, 0)
    var iflags: []bool = core.island_make_bool(isl, 0)
    for ib in ibytes { loops = loops + 1 }
    for iw in iwords { loops = loops + 1 }
    for inn in inums { loops = loops + 1 }
    for iff in iflags { loops = loops + 1 }
    return 42 + loops + bytes.len + words.len + nums.len + flags.len + ibytes.len + iwords.len + inums.len + iflags.len
  }
  return 1
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

func TestBuildMakeConstructorsRejectNegativeLengthLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tests := []struct {
		name string
		body string
	}{
		{name: "make_u8", body: "var xs: []u8 = make_u8(0 - 1)\n  return xs.len"},
		{name: "make_u16", body: "var xs: []u16 = make_u16(0 - 1)\n  return xs.len"},
		{name: "make_i32", body: "var xs: []i32 = make_i32(0 - 1)\n  return xs.len"},
		{name: "make_bool", body: "var xs: []bool = make_bool(0 - 1)\n  return xs.len"},
		{name: "island_make_u8", body: "island(64) as isl {\n    var xs: []u8 = core.island_make_u8(isl, 0 - 1)\n    return xs.len\n  }\n  return 0"},
		{name: "island_make_u16", body: "island(64) as isl {\n    var xs: []u16 = core.island_make_u16(isl, 0 - 1)\n    return xs.len\n  }\n  return 0"},
		{name: "island_make_i32", body: "island(64) as isl {\n    var xs: []i32 = core.island_make_i32(isl, 0 - 1)\n    return xs.len\n  }\n  return 0"},
		{name: "island_make_bool", body: "island(64) as isl {\n    var xs: []bool = core.island_make_bool(isl, 0 - 1)\n    return xs.len\n  }\n  return 0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := "fun main(): i32 uses alloc, islands, mem {\n  " + tc.body + "\n}\n"
			stdout, exitCode := buildAndRun(t, src)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode == 0 || exitCode == 42 {
				t.Fatalf("negative allocation length exited %d, want trap/non-success", exitCode)
			}
		})
	}
}

func TestBuildMakeConstructorsRejectByteSizeOverflowLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tests := []struct {
		name string
		body string
	}{
		{name: "make_u16", body: "var xs: []u16 = make_u16(1073741824)\n  return xs.len"},
		{name: "make_i32", body: "var xs: []i32 = make_i32(536870912)\n  return xs.len"},
		{name: "make_bool", body: "var xs: []bool = make_bool(536870912)\n  return xs.len"},
		{name: "island_make_u16", body: "island(64) as isl {\n    var xs: []u16 = core.island_make_u16(isl, 1073741824)\n    return xs.len\n  }\n  return 0"},
		{name: "island_make_i32", body: "island(64) as isl {\n    var xs: []i32 = core.island_make_i32(isl, 536870912)\n    return xs.len\n  }\n  return 0"},
		{name: "island_make_bool", body: "island(64) as isl {\n    var xs: []bool = core.island_make_bool(isl, 536870912)\n    return xs.len\n  }\n  return 0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := "fun main(): i32 uses alloc, islands, mem {\n  " + tc.body + "\n}\n"
			stdout, exitCode := buildAndRun(t, src)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode == 0 || exitCode == 42 {
				t.Fatalf("overflow allocation length exited %d, want trap/non-success", exitCode)
			}
		})
	}
}

func TestBuildMakeConstructorsWasmBuildOnly(t *testing.T) {
	src := `func main() -> Int
uses alloc, islands, mem:
    var total = 0
    var bytes: []u8 = make_u8(0)
    var words: []u16 = make_u16(0)
    var nums: []i32 = make_i32(0)
    var flags: []bool = make_bool(0)
    island(64) as isl:
        let ibytes: []u8 = core.island_make_u8(isl, 0)
        let iwords: []u16 = core.island_make_u16(isl, 0)
        let inums: []i32 = core.island_make_i32(isl, 0)
        let iflags: []bool = core.island_make_bool(isl, 0)
        total = bytes.len + words.len + nums.len + flags.len + ibytes.len + iwords.len + inums.len + iflags.len
    return total
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
