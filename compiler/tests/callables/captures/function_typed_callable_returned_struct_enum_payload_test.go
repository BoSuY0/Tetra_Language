package compiler_test

import (
	"runtime"
	"testing"
)

func TestBuildFunctionTypedReturnedStructEnumPayloadCapturedPtrClosureSmoke(t *testing.T) {
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

func makeBox(cb: fn(Int) -> Int) -> Box:
    return Box(choice: MaybeCallback.some(cb))

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let box: Box = makeBox(callbacks.identity(captured))
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

func TestBuildFunctionTypedImportedReturnedStructEnumPayloadCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return cb
`,
		"lib/pack.t4": `module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox(cb: fn(Int) -> Int) -> Box:
    return Box(choice: MaybeCallback.some(cb))
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks
import lib.pack as pack

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let box: pack.Box = pack.makeBox(callbacks.identity(captured))
    let choice: pack.MaybeCallback = box.choice
    match choice:
    case pack.MaybeCallback.some(cb):
        return cb(41)
    case pack.MaybeCallback.empty:
        return 0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedReturnedStructEnumPayloadDirectFieldMatchCapturedPtrClosureSmoke(
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

func makeBox(cb: fn(Int) -> Int) -> Box:
    return Box(choice: MaybeCallback.some(cb))

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let box: Box = makeBox(callbacks.identity(captured))
    match box.choice:
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

func TestBuildFunctionTypedReturnedStructEnumPayloadWholeStructReassignmentCapturedPtrClosureSmoke(
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

func makeBox(cb: fn(Int) -> Int) -> Box:
    return Box(choice: MaybeCallback.some(cb))

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var box: Box = Box(choice: MaybeCallback.empty)
    box = makeBox(callbacks.identity(captured))
    match box.choice:
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

func TestBuildFunctionTypedNestedReturnedStructEnumPayloadCapturedPtrClosureSmoke(t *testing.T) {
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

struct Outer:
    box: Box

func makeBox(cb: fn(Int) -> Int) -> Box:
    return Box(choice: MaybeCallback.some(cb))

func makeOuter(cb: fn(Int) -> Int) -> Outer:
    return Outer(box: makeBox(cb))

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let outer: Outer = makeOuter(callbacks.identity(captured))
    match outer.box.choice:
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

func TestBuildFunctionTypedReturnedStructEnumPayloadMultiTargetDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

struct Box:
    choice: MaybeCallback

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func makeBox(use_second: Int) -> Box:
    if use_second:
        return Box(choice: MaybeCallback.some(add2))
    else:
        return Box(choice: MaybeCallback.some(add1))

func main() -> Int:
    let box: Box = makeBox(0)
    match box.choice:
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
