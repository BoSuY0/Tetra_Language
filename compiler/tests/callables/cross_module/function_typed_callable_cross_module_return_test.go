package compiler_test

import (
	"runtime"
	"testing"
)

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

func TestBuildFunctionTypedReturnDirectNamedSymbolCrossModuleSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub func add1(x: Int) -> Int:
    return x + 1
`,
		"app/main.t4": `module app.main
import lib.math as math

func pick() -> fn(Int) -> Int:
    return math.add1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return apply(cb, 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedReturnMultiTargetCrossModuleCallbackSmoke(t *testing.T) {
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
		"app/main.t4": `module app.main
import lib.math as math

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let cb: fn(Int) -> Int = math.pick(0)
    return apply(cb, 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldFromMultiTargetCrossModuleReturnSmoke(t *testing.T) {
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
		"app/main.t4": `module app.main
import lib.math as math

struct Holder:
    cb: fn(Int) -> Int

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let holder: Holder = Holder(cb: math.pick(0))
    return apply(holder.cb, 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldFromCapturedCrossModuleReturnSmoke(t *testing.T) {
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
		"app/main.t4": `module app.main
import lib.maker as maker

struct Holder:
    cb: fn(Int) -> Int

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let holder: Holder = Holder(cb: maker.make())
    return apply(holder.cb, 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromMultiTargetCrossModuleReturnSmoke(t *testing.T) {
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
		"app/main.t4": `module app.main
import lib.math as math

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(math.pick(0))
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 41)
    case MaybeCallback.empty:
        return 0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedParameterReturnedEnumPayloadCrossModuleSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice(f: fn(Int) -> Int) -> MaybeCallback:
    return MaybeCallback.some(f)
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice(add1)
    match choice:
    case callbacks.MaybeCallback.some(cb):
        return cb(41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
