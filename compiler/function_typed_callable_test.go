package compiler

import (
	"runtime"
	"strings"
	"testing"
)

func TestBuildFunctionTypedCallableParamSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return apply(f, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCallableParamCrossModuleSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return callbacks.apply(f, 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

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

func TestBuildFunctionTypedCallableParamDirectNamedSymbolCrossModuleSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return callbacks.apply(add1, 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCallableParamMultiTargetCrossModuleSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    let a: Int = callbacks.apply(add1, 10)
    let b: Int = callbacks.apply(add2, 20)
    return a + b
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 33 {
		t.Fatalf("exit code mismatch: got %d, want 33", code)
	}
}

func TestBuildFunctionTypedReturnSymbolBackedValueSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add1(x: Int) -> Int:
    return x + 1

func pick() -> fn(Int) -> Int:
    let f: fn(Int) -> Int = add1
    let g: fn(Int) -> Int = f
    return g

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return apply(cb, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCallableParamRejectsMutableReassignment(t *testing.T) {
	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    var f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + 1
    f = fn(x: Int) -> Int:
        return x + 2
    return apply(f, 40)
`
	err := buildOnly(t, src)
	if err == nil {
		t.Fatalf("expected mutable callback reassignment diagnostic")
	}
	if !strings.Contains(err.Error(), "reassignment of function-typed local") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildFunctionTypedCallableMVPRejectsUnsupportedForms(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "capturing literal binding",
			src: `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f(41)
`,
			want: "requires a non-capturing closure literal",
		},
		{
			name: "generic literal binding",
			src: `func main() -> Int:
    let f: fn(Int) -> Int = fn<T>(x: T) -> T:
        return x
    return f(41)
`,
			want: "generic closure literals are not supported for function-typed local",
		},
		{
			name: "throwing literal binding",
			src: `enum Boom:
    case bad

func main() -> Int:
    let f: fn(Int) -> Int = fn(x: Int) -> Int throws Boom:
        throw Boom.bad
    return f(41)
`,
			want: "throwing closure literals are not supported for function-typed local",
		},
		{
			name: "generic named symbol binding",
			src: `func id<T>(x: T) -> T:
    return x

func main() -> Int:
    let f: fn(Int) -> Int = id
    return f(41)
`,
			want: "generic function symbol 'id' is not supported for function-typed local",
		},
		{
			name: "throwing named symbol binding",
			src: `enum Boom:
    case bad

func risky(x: Int) -> Int throws Boom:
    throw Boom.bad

func main() -> Int:
    let f: fn(Int) -> Int = risky
    return f(41)
`,
			want: "throwing function symbol 'risky' is not supported for function-typed local",
		},
		{
			name: "symbol backed value escape",
			src: `func add1(x: Int) -> Int:
    return x + 1

func take_ptr(x: ptr) -> Int:
    return 0

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return take_ptr(f)
`,
			want: "function value 'f' cannot escape as a first-class value in this MVP",
		},
		{
			name: "callback wrong argument count",
			src: `func call(cb: fn(Int) -> Int) -> Int:
    return cb()

func main() -> Int:
    return 0
`,
			want: "wrong argument count for callback 'cb'",
		},
		{
			name: "callback argument type mismatch",
			src: `func call(cb: fn(Int) -> Int) -> Int:
    return cb(true)

func main() -> Int:
    return 0
`,
			want: "type mismatch for callback 'cb' arg 1",
		},
		{
			name: "callback argument labels",
			src: `func call(cb: fn(Int) -> Int) -> Int:
    return cb(x: 41)

func main() -> Int:
    return 0
`,
			want: "argument labels are not supported for callback 'cb' in this MVP",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := buildOnly(t, tc.src)
			if err == nil {
				t.Fatalf("expected diagnostic containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}
