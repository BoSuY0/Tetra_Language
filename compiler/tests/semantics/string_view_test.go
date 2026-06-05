package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func TestStringViewConstructorsTypeCheck(t *testing.T) {
	testkit.RequireCheckOK(t, `
func main() -> Int:
    let text: String = "abcdef"
    let mid: String = text.window(1, 3)
    let pre: String = text.prefix(2)
    let suf: String = text.suffix(3)
    return mid.len + pre.len + suf.len
`)
}

func TestStringViewConstructorsFromLiteralTypeCheck(t *testing.T) {
	testkit.RequireCheckOK(t, `
func main() -> Int:
    let mid: String = "abcdef".window(1, 3)
    let pre: String = "abcdef".prefix(2)
    let suf: String = "abcdef".suffix(3)
    return mid.len + pre.len + suf.len
`)
}

func TestBuildStringWindowPrefixSuffixSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    let text: String = "abcdef"
    let mid: String = text.window(1, 3)
    let pre: String = text.prefix(2)
    let suf: String = text.suffix(3)
    if mid.len != 3:
        return 1
    if mid[0] != 98:
        return 2
    if mid[1] != 99:
        return 3
    if mid[2] != 100:
        return 4
    if pre.len != 2:
        return 5
    if pre[0] != 97:
        return 6
    if pre[1] != 98:
        return 7
    if suf.len != 3:
        return 8
    if suf[0] != 100:
        return 9
    if suf[1] != 101:
        return 10
    if suf[2] != 102:
        return 11
    return 42
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestStringViewConstructorsRejectInvalidRangesBeforeConstruction(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tests := []struct {
		name string
		expr string
	}{
		{name: "window_negative_start", expr: "text.window(-1, 1)"},
		{name: "window_negative_count", expr: "text.window(0, -1)"},
		{name: "window_start_past_len", expr: "text.window(text.len + 1, 0)"},
		{name: "window_count_past_tail", expr: "text.window(1, text.len)"},
		{name: "prefix_negative_count", expr: "text.prefix(-1)"},
		{name: "prefix_count_past_len", expr: "text.prefix(text.len + 1)"},
		{name: "suffix_negative_start", expr: "text.suffix(-1)"},
		{name: "suffix_start_past_len", expr: "text.suffix(text.len + 1)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := `func main() -> Int:
    let text: String = "abcdef"
    let bad: String = ` + tc.expr + `
    return bad.len
`
			stdout, exitCode := buildAndRun(t, src)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode == 0 || exitCode == 42 {
				t.Fatalf("invalid String view exited %d, want trap/non-success", exitCode)
			}
		})
	}
}

func TestStringViewConstructorsBuildOnlyTargets(t *testing.T) {
	src := `func main() -> Int:
    let text: String = "abcdef"
    let mid: String = text.window(1, 3)
    let pre: String = text.prefix(2)
    let suf: String = text.suffix(3)
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
	for _, target := range []string{"linux-x86"} {
		outPath := filepath.Join(tmp, "app-"+target+".tobj")
		if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, target, compiler.BuildOptions{Jobs: 1, Emit: compiler.EmitLibrary}); err != nil {
			t.Fatalf("build %s library: %v", target, err)
		}
	}
}
