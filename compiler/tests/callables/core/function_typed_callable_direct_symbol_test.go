package compiler_test

import (
	"runtime"
	"testing"
)

func TestBuildFunctionTypedCallableParamDirectNamedSymbolSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(add1, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCallableParamMultiTargetSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let a: Int = apply(add1, 10)
    let b: Int = apply(add2, 20)
    return a + b
`
	_, code := buildAndRun(t, src)
	if code != 33 {
		t.Fatalf("exit code mismatch: got %d, want 33", code)
	}
}

func TestBuildFunctionTypedCallableParamMultiTargetStringReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func word1(x: Int) -> String:
    return "cat"

func word2(x: Int) -> String:
    return "zebra"

func apply(cb: fn(Int) -> String, x: Int) -> String:
    return cb(x)

func main() -> Int:
    let a: String = apply(word1, 0)
    let b: String = apply(word2, 0)
    return a.len + b.len
`
	_, code := buildAndRun(t, src)
	if code != 8 {
		t.Fatalf("exit code mismatch: got %d, want 8", code)
	}
}

func TestBuildFunctionTypedCallableParamMultiTargetStructReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Pair:
    x: Int
    y: Int

func pair1(x: Int) -> Pair:
    return Pair(x: x, y: 1)

func pair2(x: Int) -> Pair:
    return Pair(x: x, y: 2)

func apply(cb: fn(Int) -> Pair, x: Int) -> Pair:
    return cb(x)

func main() -> Int:
    let a: Pair = apply(pair1, 10)
    let b: Pair = apply(pair2, 20)
    return a.x + a.y + b.x + b.y
`
	_, code := buildAndRun(t, src)
	if code != 33 {
		t.Fatalf("exit code mismatch: got %d, want 33", code)
	}
}

func TestBuildFunctionTypedCallbackCallAllowsArgumentLabelsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add1(x: Int) -> Int:
    return x + 1

func call(cb: fn(Int) -> Int) -> Int:
    return cb(value: 41)

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return call(f)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldCallAllowsArgumentLabelsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let holder: Holder = Holder(cb: add1)
    return holder.cb(value: 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGlobalCallAllowsArgumentLabelsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `val cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return cb(value: 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
