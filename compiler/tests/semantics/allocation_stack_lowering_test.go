package compiler_test

import (
	"runtime"
	"testing"
)

func TestBuildStackLoweredFixedLocalSlicesLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "u8", body: "var xs: []u8 = make_u8(2)\n  xs[0] = 20\n  xs[1] = 22\n  return xs[0] + xs[1]", want: 42},
		{name: "u16", body: "var xs: []u16 = make_u16(2)\n  xs[0] = 20\n  xs[1] = 22\n  return xs[0] + xs[1]", want: 42},
		{name: "i32", body: "var xs: []i32 = make_i32(4)\n  xs[0] = 10\n  xs[1] = 11\n  xs[2] = 12\n  xs[3] = 9\n  return xs[0] + xs[1] + xs[2] + xs[3]", want: 42},
		{name: "bool", body: "var xs: []bool = make_bool(2)\n  xs[0] = true\n  xs[1] = false\n  if xs[0]:\n      return 42\n  return 1", want: 42},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := "func main() -> Int\nuses alloc, mem:\n    " + tc.body + "\n"
			stdout, exitCode := buildAndRun(t, src)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != tc.want {
				t.Fatalf("exit code mismatch: %d, want %d", exitCode, tc.want)
			}
		})
	}
}

func TestBuildStackBorrowedLocalViewsLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 1
    xs[1] = 20
    xs[2] = 22
    xs[3] = 4
    let mid: []i32 = xs.window(1, 2).borrow()
    return mid[0] + mid[1]
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d, want 42", exitCode)
	}
}

func TestBuildStackLoweredCopyOfLocalViewLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    xs[0] = 20
    xs[1] = 22
    let copied: []u8 = xs.window(0, 2).copy()
    return copied[0] + copied[1]
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d, want 42", exitCode)
	}
}

func TestBuildScalarReplacedTinySliceLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 20
    xs[1] = 22
    return xs[0] + xs[1]
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d, want 42", exitCode)
	}
}
