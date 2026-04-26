package compiler

import (
	"runtime"
	"testing"
)

func TestBuildForCollectionSliceSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, islands, mem:
    var total: Int = 0
    island(128) as isl:
        var xs: []i32 = core.island_make_i32(isl, 3)
        xs[0] = 10
        xs[1] = 20
        xs[2] = 12
        for x in xs:
            total = total + x
    return total
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildForCollectionStringSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    var total: Int = 0
    let text: String = "*"
    for ch in text:
        total = total + ch
    return total
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildForCollectionU8SliceSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, islands, mem:
    var total: Int = 0
    island(128) as isl:
        var bytes: []u8 = core.island_make_u8(isl, 2)
        bytes[0] = 40
        bytes[1] = 2
        for b in bytes:
            total = total + b
    return total
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
