package compiler_test

import (
	"runtime"
	"testing"
)

func TestBuildFunctionTypedParameterReturnEnumPayloadReassignmentCapturedPtrClosureSmoke(
	t *testing.T,
) {
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
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(identity(captured))
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

func TestBuildFunctionTypedImportedParameterReturnEnumPayloadReassignmentCapturedPtrClosureSmoke(
	t *testing.T,
) {
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
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(callbacks.identity(captured))
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

func TestBuildFunctionTypedImportedParameterReturnStructFieldEnumPayloadReassignmentCapturedPtrClosureSmoke(
	t *testing.T,
) {
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

struct Box:
    choice: MaybeCallback

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var box: Box = Box(choice: MaybeCallback.empty)
    box.choice = MaybeCallback.some(callbacks.identity(captured))
    let choice: MaybeCallback = box.choice
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
