package compiler_test

import (
	"runtime"
	"testing"
)

func TestBuildFunctionTypedParameterReturnCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func identity(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return cb

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let cb: fn(Int) -> Int = identity(captured)
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedParameterReturnCapturedPtrClosureCrossModuleSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

func identity(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return cb
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let cb: fn(Int) -> Int = callbacks.identity(captured)
    return cb(41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedReturnCallStructFieldCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func identity(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return cb

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let holder: Holder = Holder(cb: identity(captured))
    return holder.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedReturnCallEnumPayloadCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func identity(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return cb

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let choice: MaybeCallback = MaybeCallback.some(identity(captured))
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedParameterReturnStructFieldCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return cb
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let holder: Holder = Holder(cb: callbacks.identity(captured))
    return holder.cb(41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedParameterReturnEnumPayloadCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return cb
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let choice: MaybeCallback = MaybeCallback.some(callbacks.identity(captured))
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
