package compiler_test

import (
	"runtime"
	"testing"
)

func TestBuildFunctionTypedParameterReturnMutableLocalReassignmentCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add0(x: Int) -> Int:
    return x

func identity(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return cb

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var cb: fn(Int) -> Int = add0
    cb = identity(captured)
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedParameterReturnStructFieldReassignmentCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add0(x: Int) -> Int:
    return x

func identity(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return cb

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var holder: Holder = Holder(cb: add0)
    holder.cb = identity(captured)
    return holder.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedParameterReturnMutableLocalReassignmentCapturedPtrClosureSmoke(t *testing.T) {
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

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var cb: fn(Int) -> Int = add0
    cb = callbacks.identity(captured)
    return cb(41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedParameterReturnStructFieldReassignmentCapturedPtrClosureSmoke(t *testing.T) {
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

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var holder: Holder = Holder(cb: add0)
    holder.cb = callbacks.identity(captured)
    return holder.cb(41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedParameterReturnNestedStructFieldReassignmentCapturedPtrClosureSmoke(t *testing.T) {
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

struct Box:
    holder: Holder

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var box: Box = Box(holder: Holder(cb: add0))
    box.holder.cb = callbacks.identity(captured)
    return box.holder.cb(41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedParameterReturnWholeStructReassignmentCapturedPtrClosureSmoke(t *testing.T) {
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

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var holder: Holder = Holder(cb: add0)
    holder = Holder(cb: callbacks.identity(captured))
    return holder.cb(41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedParameterReturnStructValuedFieldReassignmentCapturedPtrClosureSmoke(t *testing.T) {
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

struct Box:
    holder: Holder

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var box: Box = Box(holder: Holder(cb: add0))
    box.holder = Holder(cb: callbacks.identity(captured))
    return box.holder.cb(41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedParameterReturnWholeNestedStructReassignmentCapturedPtrClosureSmoke(t *testing.T) {
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

struct Box:
    holder: Holder

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var box: Box = Box(holder: Holder(cb: add0))
    box = Box(holder: Holder(cb: callbacks.identity(captured)))
    return box.holder.cb(41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
