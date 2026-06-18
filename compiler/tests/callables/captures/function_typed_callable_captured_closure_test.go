package compiler_test

import (
	"runtime"
	"testing"

	"tetra_language/compiler/tests/callables/testkit"
)

var (
	buildAndRun      = testkit.BuildAndRun
	buildAndRunFiles = testkit.BuildAndRunFiles
	buildOnly        = testkit.BuildOnly
	buildOnlyFiles   = testkit.BuildOnlyFiles
)

func TestBuildFunctionTypedCapturedClosureLocalDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureLocalDirectCallAllowsArgumentLabelsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f(value: 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedPtrClosureFiveSlotDirectAndCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let f: ptr = fn(x: Int) -> Int:
        return x + one + two + three + four + five
    let direct: Int = f(0)
    return direct + apply(f, 12)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureCompositeCaptureMatrixCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Pair:
    left: Int
    right: Bool

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: true)
    let text: String = "!"
    let enabled: Bool = true
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        if enabled && pair.right:
            return x + pair.left + text[0]
        return 0
    return apply(f, 8)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureEnumCaptureMatrixCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Mode:
    case fast
    case slow

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let mode: Mode = Mode.fast
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        match mode:
        case Mode.fast:
            return x + 1
        case Mode.slow:
            return 0
    return apply(f, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureOptionalCaptureMatrixCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let maybe: Int? = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        match maybe:
        case none:
            return 0
        case _:
            return x + 1
    return apply(f, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableCaptureSnapshotsAtBindingSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    var base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    base = 100
    return f(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedLocalAliasCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let cb: fn(Int) -> Int = captured
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildCapturedPtrClosureLabeledDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    return captured(x: 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildCapturedPtrClosureDirectCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    return apply(captured, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildCapturedPtrClosureReturnedFunctionValueSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func make() -> fn(Int) -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    return captured

func main() -> Int:
    let cb: fn(Int) -> Int = make()
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableLocalReassignCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var cb: fn(Int) -> Int = add0
    cb = captured
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedReturnMultiTargetCapturedClosureDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func pick(use_second: Int) -> fn(Int) -> Int:
    let base1: Int = 1
    let base2: Int = 2
    if use_second:
        return fn(x: Int) -> Int = x + base2
    else:
        return fn(x: Int) -> Int = x + base1

func main() -> Int:
    let cb: fn(Int) -> Int = pick(0)
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedReturnMultiTargetCapturedPtrClosureDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add2(x: Int) -> Int:
    return x + 2

func pick(use_second: Int) -> fn(Int) -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    if use_second:
        return add2
    else:
        return captured

func main() -> Int:
    let cb: fn(Int) -> Int = pick(0)
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedReturnCapturedPtrClosureDirectCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func make() -> fn(Int) -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    return captured

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(make(), 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedReturnMultiTargetCapturedClosureCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func pick(use_second: Int) -> fn(Int) -> Int:
    let base1: Int = 1
    let base2: Int = 2
    if use_second:
        return fn(x: Int) -> Int = x + base2
    else:
        return fn(x: Int) -> Int = x + base1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let cb: fn(Int) -> Int = pick(0)
    return apply(cb, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnCrossModuleCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/maker.t4": `module lib.maker

pub func make() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int:
        return x + base
`,
		"lib/callbacks.t4": `module lib.callbacks

pub func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks
import lib.maker as maker

func main() -> Int:
    let cb: fn(Int) -> Int = maker.make()
    return callbacks.apply(cb, 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnCrossModuleDirectCallbackArgumentSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/maker.t4": `module lib.maker

pub func make() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int:
        return x + base
`,
		"lib/callbacks.t4": `module lib.callbacks

pub func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks
import lib.maker as maker

func main() -> Int:
    return callbacks.apply(maker.make(), 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
