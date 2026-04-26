package compiler

import (
	"runtime"
	"testing"
)

func TestBuildWhileBreakContinueSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 10:
        i = i + 1
        if i == 3:
            continue
        if i == 6:
            break
        total = total + i
    return total
`
	_, code := buildAndRun(t, src)
	if code != 12 {
		t.Fatalf("exit code mismatch: got %d, want 12", code)
	}
}

func TestBuildRangeForBreakContinueSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    var total: Int = 0
    for i in 0..<10:
        if i == 2:
            continue
        if i == 5:
            break
        total = total + i
    return total
`
	_, code := buildAndRun(t, src)
	if code != 8 {
		t.Fatalf("exit code mismatch: got %d, want 8", code)
	}
}

func TestBuildCollectionForBreakContinueSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    var total: Int = 0
    let text: String = "*!+"
    for ch in text:
        if ch == 33:
            continue
        if ch == 43:
            break
        total = total + ch
    return total
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildUnaryBangSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    let off: Bool = false
    if !off && !0:
        return 42
    return 1
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildLogicalShortCircuitSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func mark() -> Bool:
    return true

func main() -> Int:
    let left: Bool = false
    if left && mark():
        return 1
    if left || mark():
        return 42
    return 2
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
