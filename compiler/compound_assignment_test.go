package compiler

import (
	"runtime"
	"testing"
)

func TestBuildCompoundAssignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int:
    var x: Int = 4
    x += 3
    x *= 6
    x -= 0
    x /= 1
    x %= 100
    return x
`
	_, exitCode := buildAndRun(t, src)
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCompoundAssignmentFieldAndIndexSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
struct Box:
    x: Int

func main() -> Int
uses alloc, mem:
    var b: Box = Box(x: 20)
    b.x += 1
    var xs: []i32 = make_i32(1)
    xs[0] = 20
    xs[0] += 1
    return b.x + xs[0]
`
	_, exitCode := buildAndRun(t, src)
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}
