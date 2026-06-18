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

func TestBuildFunctionTypedStructFieldCrossModuleCallbackSmoke(t *testing.T) {
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

struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let holder: Holder = Holder(cb: add1)
    return callbacks.apply(holder.cb, 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadCrossModuleCallbackSmoke(t *testing.T) {
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

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return callbacks.apply(cb, 41)
    case MaybeCallback.empty:
        return 0
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

func TestBuildFunctionTypedReturnDirectCallbackArgumentCrossModuleSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub func add1(x: Int) -> Int:
    return x + 1

pub func add2(x: Int) -> Int:
    return x + 2

pub func pick(use_second: Int) -> fn(Int) -> Int:
    if use_second:
        return add2
    else:
        return add1
`,
		"lib/callbacks.t4": `module lib.callbacks

pub func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks
import lib.math as math

func main() -> Int:
    return callbacks.apply(math.pick(0), 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedParameterReturnCapturedPtrClosureDirectCallbackArgumentSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return cb

pub func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    return callbacks.apply(callbacks.identity(captured), 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedReturnIgnoresCapturedCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func add0(x: Int) -> Int:
    return x

pub func ignore(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return add0
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let cb: fn(Int) -> Int = callbacks.ignore(captured)
    return cb(41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 41 {
		t.Fatalf("exit code mismatch: got %d, want 41", code)
	}
}
