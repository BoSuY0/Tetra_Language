package compiler_test

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

func TestBuildFunctionTypedCapturedClosureCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    let extra: Int = 0
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base + extra
    return apply(f, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureFourSlotCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    let extra: Int = 0
    let third: Int = 0
    let fourth: Int = 0
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base + extra + third + fourth
    return apply(f, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedDirectClosureLiteralCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    return apply(fn(x: Int) -> Int = x + base, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedDirectClosureLiteralCallbackArgumentCrossModuleSmoke(t *testing.T) {
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

func main() -> Int:
    let base: Int = 1
    return callbacks.apply(fn(x: Int) -> Int = x + base, 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureAliasCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    let extra: Int = 0
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base + extra
    let g: fn(Int) -> Int = f
    return apply(g, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func pick() -> fn(Int) -> Int:
    let base: Int = 1
    let extra: Int = 0
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base + extra
    return f

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return apply(cb, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func pick() -> fn(Int) -> Int:
    let base: Int = 1
    let extra: Int = 0
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base + extra
    return f

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedDirectClosureLiteralReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func pick() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int = x + base

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

func TestBuildFunctionTypedStructFieldDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let holder: Holder = Holder(cb: add1)
    return holder.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let holder: Holder = Holder(cb: add1)
    return apply(holder.cb, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureStructFieldCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    let extra: Int = 0
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base + extra
    let holder: Holder = Holder(cb: f)
    return apply(holder.cb, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedParameterStructFieldDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func callField(f: fn(Int) -> Int, x: Int) -> Int:
    let holder: Holder = Holder(cb: f)
    return holder.cb(x)

func main() -> Int:
    return callField(add1, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedParameterStructFieldCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func callField(f: fn(Int) -> Int, x: Int) -> Int:
    let holder: Holder = Holder(cb: f)
    return apply(holder.cb, x)

func main() -> Int:
    return callField(add1, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureStructFieldDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let base: Int = 1
    let extra: Int = 0
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base + extra
    let holder: Holder = Holder(cb: f)
    return holder.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let holder: Holder = Holder(cb: captured)
    return holder.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldDirectClosureLiteralSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let base: Int = 1
    let holder: Holder = Holder(cb: fn(x: Int) -> Int = x + base)
    return holder.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedStructFieldDirectClosureLiteralSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/types.t4": `module lib.types

pub struct Holder:
    cb: fn(Int) -> Int
`,
		"app/main.t4": `module app.main
import lib.types as types

func main() -> Int:
    let base: Int = 1
    let holder: types.Holder = types.Holder(cb: fn(x: Int) -> Int = x + base)
    return holder.cb(41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedStructFieldParamCapturedClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/types.t4": `module lib.types

pub struct Holder:
    cb: fn(Int) -> Int

pub func callHolder(holder: Holder, x: Int) -> Int:
    return holder.cb(x)
`,
		"app/main.t4": `module app.main
import lib.types as types

func main() -> Int:
    let base: Int = 1
    let holder: types.Holder = types.Holder(cb: fn(x: Int) -> Int = x + base)
    return types.callHolder(holder, 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedStructFieldParamDirectConstructorCapturedClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/types.t4": `module lib.types

pub struct Holder:
    cb: fn(Int) -> Int

pub func callHolder(holder: Holder, x: Int) -> Int:
    return holder.cb(x)
`,
		"app/main.t4": `module app.main
import lib.types as types

func main() -> Int:
    let base: Int = 1
    return types.callHolder(types.Holder(cb: fn(x: Int) -> Int = x + base), 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedStructFieldParamDirectConstructorCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/types.t4": `module lib.types

pub struct Holder:
    cb: fn(Int) -> Int

pub func callHolder(holder: Holder, x: Int) -> Int:
    return holder.cb(x)
`,
		"app/main.t4": `module app.main
import lib.types as types

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    return types.callHolder(types.Holder(cb: captured), 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedStructFieldParamCapturedReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/types.t4": `module lib.types

pub struct Holder:
    cb: fn(Int) -> Int

pub func callHolder(holder: Holder, x: Int) -> Int:
    return holder.cb(x)
`,
		"lib/maker.t4": `module lib.maker

pub func make() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int:
        return x + base
`,
		"app/main.t4": `module app.main
import lib.types as types
import lib.maker as maker

func main() -> Int:
    return types.callHolder(types.Holder(cb: maker.make()), 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedSelectiveImportedStructFieldParamDirectConstructorCapturedClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/types.t4": `module lib.types

pub struct Holder:
    cb: fn(Int) -> Int

pub func callHolder(holder: Holder, x: Int) -> Int:
    return holder.cb(x)
`,
		"app/main.t4": `module app.main
import lib.types.{Holder, callHolder}

func main() -> Int:
    let base: Int = 1
    return callHolder(Holder(cb: fn(x: Int) -> Int = x + base), 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldMutableCaptureSnapshotsAtBindingSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    var base: Int = 1
    let holder: Holder = Holder(cb: fn(x: Int) -> Int = x + base)
    base = 100
    return holder.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedNestedStructFieldDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let box: Box = Box(holder: Holder(cb: add1))
    return box.holder.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedNestedStructFieldCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let box: Box = Box(holder: Holder(cb: add1))
    return apply(box.holder.cb, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedNestedStructFieldReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    var box: Box = Box(holder: Holder(cb: add1))
    box.holder.cb = add2
    return box.holder.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedNestedStructFieldReassignmentCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    var box: Box = Box(holder: Holder(cb: add1))
    box.holder.cb = add2
    return apply(box.holder.cb, 40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldReassignmentMultiTargetCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    var use_second: Int = 0
    var holder: Holder = Holder(cb: add2)
    if use_second:
        holder.cb = add2
    else:
        holder.cb = add1
    return apply(holder.cb, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldReassignmentFromMultiTargetReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func pick(use_second: Int) -> fn(Int) -> Int:
    if use_second:
        return add2
    else:
        return add1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    var holder: Holder = Holder(cb: add2)
    holder.cb = pick(0)
    return apply(holder.cb, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructAliasPreservesFunctionFieldSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let box: Box = Box(holder: Holder(cb: add1))
    let holder: Holder = box.holder
    return holder.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructAliasAfterNestedReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    var box: Box = Box(holder: Holder(cb: add1))
    box.holder.cb = add2
    let holder: Holder = box.holder
    return holder.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldAliasSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let holder: Holder = Holder(cb: add1)
    let f: fn(Int) -> Int = holder.cb
    return apply(f, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldFromReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func pick() -> fn(Int) -> Int:
    return add1

func main() -> Int:
    let holder: Holder = Holder(cb: pick())
    return holder.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldFromMultiTargetReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func pick(use_second: Int) -> fn(Int) -> Int:
    if use_second:
        return add2
    else:
        return add1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let holder: Holder = Holder(cb: pick(0))
    return apply(holder.cb, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldFromMultiTargetCapturedClosureReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func pick(use_second: Int) -> fn(Int) -> Int:
    let base1: Int = 1
    let base2: Int = 2
    if use_second:
        return fn(x: Int) -> Int = x + base2
    else:
        return fn(x: Int) -> Int = x + base1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let holder: Holder = Holder(cb: pick(0))
    return apply(holder.cb, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldFromReturnedStructSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func makeHolder() -> Holder:
    return Holder(cb: add1)

func main() -> Int:
    let holder: Holder = makeHolder()
    return holder.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldCapturedClosureFromReturnedStructSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int = x + base)

func main() -> Int:
    let holder: Holder = makeHolder()
    return holder.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldCapturedClosureFromCrossModuleReturnedStructSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/types.t4": `module lib.types

pub struct Holder:
    cb: fn(Int) -> Int

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int:
        return x + base
    )
`,
		"app/main.t4": `module app.main
import lib.types as types

func main() -> Int:
    let holder: types.Holder = types.makeHolder()
    return holder.cb(41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldFromReassignedReturnedStructSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func makeHolder() -> Holder:
    var holder: Holder = Holder(cb: add1)
    holder.cb = add2
    return holder

func main() -> Int:
    let holder: Holder = makeHolder()
    return holder.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldFromReturnedStructCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func makeHolder() -> Holder:
    var holder: Holder = Holder(cb: add1)
    holder.cb = add2
    return holder

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let holder: Holder = makeHolder()
    return apply(holder.cb, 40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldReturnedStructMultiTargetCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func makeHolder(use_second: Int) -> Holder:
    if use_second:
        return Holder(cb: add2)
    else:
        return Holder(cb: add1)

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let holder: Holder = makeHolder(0)
    return apply(holder.cb, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldReturnedStructMultiTargetDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func makeHolder(use_second: Int) -> Holder:
    if use_second:
        return Holder(cb: add2)
    else:
        return Holder(cb: add1)

func main() -> Int:
    let holder: Holder = makeHolder(0)
    return holder.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedNestedStructFieldFromReturnedStructInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func makeHolder() -> Holder:
    var holder: Holder = Holder(cb: add1)
    holder.cb = add2
    return holder

func makeBox() -> Box:
    return Box(holder: makeHolder())

func main() -> Int:
    let box: Box = makeBox()
    return box.holder.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedNestedStructFieldReturnedStructMultiTargetInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func makeHolder(use_second: Int) -> Holder:
    if use_second:
        return Holder(cb: add2)
    else:
        return Holder(cb: add1)

func makeBox(use_second: Int) -> Box:
    return Box(holder: makeHolder(use_second))

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let box: Box = makeBox(0)
    return apply(box.holder.cb, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldFromStructFieldSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let source: Holder = Holder(cb: add1)
    let target: Holder = Holder(cb: source.cb)
    return target.cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func pick() -> fn(Int) -> Int:
    let holder: Holder = Holder(cb: add1)
    return holder.cb

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

func TestBuildCapturedFunctionTypedStructFieldReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func pick() -> fn(Int) -> Int:
    let base: Int = 1
    let holder: Holder = Holder(cb: fn(x: Int) -> Int:
        return x + base
    )
    return holder.cb

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
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

func TestBuildFunctionTypedEnumPayloadDirectCallAllowsArgumentLabelsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return cb(value: 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedParameterEnumPayloadDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func callPayload(f: fn(Int) -> Int, x: Int) -> Int:
    let choice: MaybeCallback = MaybeCallback.some(f)
    match choice:
    case MaybeCallback.some(cb):
        return cb(x)
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    return callPayload(add1, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedParameterEnumPayloadCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func callPayload(f: fn(Int) -> Int, x: Int) -> Int:
    let choice: MaybeCallback = MaybeCallback.some(f)
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, x)
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    return callPayload(add1, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedParameterEnumPayloadReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func callPayload(f: fn(Int) -> Int, x: Int) -> Int:
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(f)
    match choice:
    case MaybeCallback.some(cb):
        return cb(x)
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    return callPayload(add1, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedParameterReturnedEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func makeChoice(f: fn(Int) -> Int) -> MaybeCallback:
    return MaybeCallback.some(f)

func main() -> Int:
    let choice: MaybeCallback = makeChoice(add1)
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

func TestBuildFunctionTypedCapturedClosureEnumPayloadCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    let extra: Int = 0
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base + extra
    let choice: MaybeCallback = MaybeCallback.some(f)
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureEnumPayloadDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let base: Int = 1
    let extra: Int = 0
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base + extra
    let choice: MaybeCallback = MaybeCallback.some(f)
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

func TestBuildFunctionTypedEnumPayloadDirectClosureLiteralSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let base: Int = 1
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int = x + base)
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

func TestBuildFunctionTypedModuleEnumPayloadDirectClosureLiteralSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"app/main.t4": `module app.main

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let base: Int = 1
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int = x + base)
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

func TestBuildFunctionTypedEnumPayloadMutableCaptureSnapshotsAtBindingSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    var base: Int = 1
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int = x + base)
    base = 100
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

func TestBuildFunctionTypedEnumPayloadAliasSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        let f: fn(Int) -> Int = cb
        return apply(f, 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadWholeEnumAliasSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    let next: MaybeCallback = choice
    match next:
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

func TestBuildFunctionTypedMutableEnumPayloadReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(add1)
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

func TestBuildFunctionTypedMutableEnumPayloadReassignmentFromReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func pick() -> fn(Int) -> Int:
    return add1

func main() -> Int:
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(pick())
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

func TestBuildFunctionTypedMutableEnumPayloadReassignmentFromMultiTargetReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func pick(use_second: Int) -> fn(Int) -> Int:
    if use_second:
        return add2
    else:
        return add1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(pick(0))
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableEnumPayloadReassignmentFromClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let base: Int = 1
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(fn(x: Int) -> Int = x + base)
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

func TestBuildFunctionTypedMutableEnumPayloadReassignmentFromWholeEnumAliasSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    var next: MaybeCallback = MaybeCallback.empty
    next = choice
    match next:
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

func TestBuildFunctionTypedMutableEnumPayloadReassignmentMultiTargetSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    var use_second: Int = 1
    var choice: MaybeCallback = MaybeCallback.empty
    if use_second:
        choice = MaybeCallback.some(add2)
    else:
        choice = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return cb(40)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableEnumPayloadReassignmentMultiTargetCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    var use_second: Int = 1
    var choice: MaybeCallback = MaybeCallback.empty
    if use_second:
        choice = MaybeCallback.some(add2)
    else:
        choice = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 40)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func pick() -> fn(Int) -> Int:
    return add1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(pick())
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

func TestBuildFunctionTypedEnumPayloadFromMultiTargetReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func pick(use_second: Int) -> fn(Int) -> Int:
    if use_second:
        return add2
    else:
        return add1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(pick(0))
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromMultiTargetCapturedClosureReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func pick(use_second: Int) -> fn(Int) -> Int:
    let base1: Int = 1
    let base2: Int = 2
    if use_second:
        return fn(x: Int) -> Int = x + base2
    else:
        return fn(x: Int) -> Int = x + base1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(pick(0))
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromFourSlotCapturedClosureReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func makeChoice() -> MaybeCallback:
    let base: Int = 1
    let extra: Int = 0
    let third: Int = 0
    let fourth: Int = 0
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + base + extra + third + fourth
    )

func main() -> Int:
    let choice: MaybeCallback = makeChoice()
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

func TestBuildFunctionTypedEnumPayloadFromCrossModuleCapturedReturnedEnumSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
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

func TestBuildFunctionTypedEnumPayloadFromReturnedEnumSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func makeChoice() -> MaybeCallback:
    return MaybeCallback.some(add1)

func main() -> Int:
    let choice: MaybeCallback = makeChoice()
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

func TestBuildFunctionTypedEnumPayloadFromReturnedEnumWholeEnumAliasSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func makeChoice() -> MaybeCallback:
    return MaybeCallback.some(add1)

func main() -> Int:
    let choice: MaybeCallback = makeChoice()
    let next: MaybeCallback = choice
    match next:
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

func TestBuildFunctionTypedEnumPayloadReturnedEnumMultiTargetCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func makeChoice(use_second: Int) -> MaybeCallback:
    if use_second:
        return MaybeCallback.some(add2)
    else:
        return MaybeCallback.some(add1)

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let choice: MaybeCallback = makeChoice(1)
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 40)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromDirectReturnedEnumMatchSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func makeChoice() -> MaybeCallback:
    return MaybeCallback.some(add1)

func main() -> Int:
    match makeChoice():
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

func TestBuildFunctionTypedEnumPayloadCapturedClosureFromReturnedEnumSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int = x + base)

func main() -> Int:
    match makeChoice():
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

func TestBuildFunctionTypedEnumPayloadCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let choice: MaybeCallback = MaybeCallback.some(captured)
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

func TestBuildFunctionTypedEnumPayloadFromStructFieldSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let holder: Holder = Holder(cb: add1)
    let choice: MaybeCallback = MaybeCallback.some(holder.cb)
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

func TestBuildFunctionTypedStructFieldFromEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        let holder: Holder = Holder(cb: cb)
        return holder.cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        let next: MaybeCallback = MaybeCallback.some(cb)
        match next:
        case MaybeCallback.some(next_cb):
            return next_cb(41)
        case MaybeCallback.empty:
            return 0
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func pick() -> fn(Int) -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    let fallback: fn(Int) -> Int = add1
    match choice:
    case MaybeCallback.some(cb):
        return cb
    case _:
        return fallback

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

func TestBuildCapturedFunctionTypedEnumPayloadReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func pick() -> fn(Int) -> Int:
    let base: Int = 1
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
    match choice:
    case MaybeCallback.some(cb):
        return cb
    case MaybeCallback.empty:
        return fn(x: Int) -> Int = x

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCallableMutableReassignmentDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    var f: fn(Int) -> Int = add1
    f = add2
    return f(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCallableMutableReassignmentCallbackSmoke(t *testing.T) {
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
    var f: fn(Int) -> Int = add1
    f = add2
    return apply(f, 40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCallableMutableReassignmentFromReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add1(x: Int) -> Int:
    return x + 1

func pick() -> fn(Int) -> Int:
    let base: Int = 2
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f

func main() -> Int:
    var f: fn(Int) -> Int = add1
    f = pick()
    return f(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCallableMutableReassignmentFromClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let base: Int = 2
    var f: fn(Int) -> Int = add1
    f = fn(x: Int) -> Int:
        return x + base
    return f(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCallableMutableReassignmentFromStructFieldSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let base: Int = 2
    let captured: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    let holder: Holder = Holder(cb: captured)
    var f: fn(Int) -> Int = add1
    f = holder.cb
    return f(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableStructFieldReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    var holder: Holder = Holder(cb: add1)
    holder.cb = add2
    return holder.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableStructFieldReassignmentFromClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let base: Int = 2
    var holder: Holder = Holder(cb: add1)
    holder.cb = fn(x: Int) -> Int = x + base
    return holder.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableStructFieldReassignmentCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    var holder: Holder = Holder(cb: add1)
    holder.cb = add2
    return apply(holder.cb, 40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableStructFieldReassignmentAliasSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    var holder: Holder = Holder(cb: add1)
    holder.cb = add2
    let f: fn(Int) -> Int = holder.cb
    return f(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableStructFieldToStructFieldSnapshotSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    var source: Holder = Holder(cb: add1)
    var dest: Holder = Holder(cb: add1)
    source.cb = add2
    dest.cb = source.cb
    return dest.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImmutableAliasFromMutableLocalSnapshotSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    var f: fn(Int) -> Int = add1
    f = add2
    let g: fn(Int) -> Int = f
    return g(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolLocalSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func id<T>(x: T) -> T:
    return x

func main() -> Int:
    let f: fn(Int) -> Int = id
    return f(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolCrossModuleLocalSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/generic.t4": `module lib.generic

pub func id<T>(x: T) -> T:
    return x
`,
		"app/main.t4": `module app.main
import lib.generic as generic

func main() -> Int:
    let f: fn(Int) -> Int = generic.id
    return f(42)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericClosureLiteralLocalSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    let f: fn(Int) -> Int = fn<T>(x: T) -> T:
        return x
    return f(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericClosureLiteralDirectCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(fn<T>(x: T) -> T = x, 42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericClosureLiteralStructFieldSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let holder: Holder = Holder(cb: fn<T>(x: T) -> T = x)
    return holder.cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericClosureLiteralNestedStructInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func main() -> Int:
    let box: Box = Box(holder: Holder(cb: fn<T>(x: T) -> T = x))
    return box.holder.cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericClosureLiteralEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Choice:
    case some(fn(Int) -> Int)

func main() -> Int:
    let choice: Choice = Choice.some(fn<T>(x: T) -> T = x)
    match choice:
        case Choice.some(cb):
            return cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericClosureLiteralReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func pick() -> fn(Int) -> Int:
    return fn<T>(x: T) -> T = x

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericClosureLiteralMutableLocalReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var f: fn(Int) -> Int = add1
    f = fn<T>(x: T) -> T = x
    return f(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericClosureLiteralStructFieldReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var holder: Holder = Holder(cb: add1)
    holder.cb = fn<T>(x: T) -> T = x
    return holder.cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericClosureLiteralNestedStructFieldReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var box: Box = Box(holder: Holder(cb: add1))
    box.holder.cb = fn<T>(x: T) -> T = x
    return box.holder.cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericClosureLiteralEnumPayloadReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Choice:
    case some(fn(Int) -> Int)

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var choice: Choice = Choice.some(add1)
    choice = Choice.some(fn<T>(x: T) -> T = x)
    match choice:
        case Choice.some(cb):
            return cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolDirectCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func id<T>(x: T) -> T:
    return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(id, 42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolCrossModuleDirectCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/generic.t4": `module lib.generic

pub func id<T>(x: T) -> T:
    return x
`,
		"app/main.t4": `module app.main
import lib.generic as generic

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(generic.id, 42)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolStructFieldSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func id<T>(x: T) -> T:
    return x

func main() -> Int:
    let holder: Holder = Holder(cb: id)
    return holder.cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolNestedStructInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func id<T>(x: T) -> T:
    return x

func main() -> Int:
    let box: Box = Box(holder: Holder(cb: id))
    return box.holder.cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolCrossModuleStructFieldSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/generic.t4": `module lib.generic

pub func id<T>(x: T) -> T:
    return x
`,
		"app/main.t4": `module app.main
import lib.generic as generic

struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let holder: Holder = Holder(cb: generic.id)
    return holder.cb(42)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolCrossModuleNestedStructInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/generic.t4": `module lib.generic

pub func id<T>(x: T) -> T:
    return x
`,
		"app/main.t4": `module app.main
import lib.generic as generic

struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func main() -> Int:
    let box: Box = Box(holder: Holder(cb: generic.id))
    return box.holder.cb(42)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Choice:
    case some(fn(Int) -> Int)

func id<T>(x: T) -> T:
    return x

func main() -> Int:
    let choice: Choice = Choice.some(id)
    match choice:
        case Choice.some(cb):
            return cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolCrossModuleEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/generic.t4": `module lib.generic

pub func id<T>(x: T) -> T:
    return x
`,
		"app/main.t4": `module app.main
import lib.generic as generic

enum Choice:
    case some(fn(Int) -> Int)

func main() -> Int:
    let choice: Choice = Choice.some(generic.id)
    match choice:
        case Choice.some(cb):
            return cb(42)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolEnumPayloadReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Choice:
    case some(fn(Int) -> Int)

func add1(x: Int) -> Int:
    return x + 1

func id<T>(x: T) -> T:
    return x

func main() -> Int:
    var choice: Choice = Choice.some(add1)
    choice = Choice.some(id)
    match choice:
        case Choice.some(cb):
            return cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolCrossModuleEnumPayloadReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/generic.t4": `module lib.generic

pub func id<T>(x: T) -> T:
    return x
`,
		"app/main.t4": `module app.main
import lib.generic as generic

enum Choice:
    case some(fn(Int) -> Int)

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var choice: Choice = Choice.some(add1)
    choice = Choice.some(generic.id)
    match choice:
        case Choice.some(cb):
            return cb(42)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func id<T>(x: T) -> T:
    return x

func pick() -> fn(Int) -> Int:
    return id

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolCrossModuleReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/generic.t4": `module lib.generic

pub func id<T>(x: T) -> T:
    return x
`,
		"app/main.t4": `module app.main
import lib.generic as generic

func pick() -> fn(Int) -> Int:
    return generic.id

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return cb(42)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolMutableLocalReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add1(x: Int) -> Int:
    return x + 1

func id<T>(x: T) -> T:
    return x

func main() -> Int:
    var f: fn(Int) -> Int = add1
    f = id
    return f(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolCrossModuleMutableLocalReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/generic.t4": `module lib.generic

pub func id<T>(x: T) -> T:
    return x
`,
		"app/main.t4": `module app.main
import lib.generic as generic

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var f: fn(Int) -> Int = add1
    f = generic.id
    return f(42)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolStructFieldReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func id<T>(x: T) -> T:
    return x

func main() -> Int:
    var holder: Holder = Holder(cb: add1)
    holder.cb = id
    return holder.cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolCrossModuleStructFieldReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/generic.t4": `module lib.generic

pub func id<T>(x: T) -> T:
    return x
`,
		"app/main.t4": `module app.main
import lib.generic as generic

struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var holder: Holder = Holder(cb: add1)
    holder.cb = generic.id
    return holder.cb(42)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolNestedStructFieldReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func id<T>(x: T) -> T:
    return x

func main() -> Int:
    var box: Box = Box(holder: Holder(cb: add1))
    box.holder.cb = id
    return box.holder.cb(42)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolCrossModuleNestedStructFieldReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/generic.t4": `module lib.generic

pub func id<T>(x: T) -> T:
    return x
`,
		"app/main.t4": `module app.main
import lib.generic as generic

struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var box: Box = Box(holder: Holder(cb: add1))
    box.holder.cb = generic.id
    return box.holder.cb(42)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGenericSymbolImportedNestedStructFieldReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/types.t4": `module lib.types

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder
`,
		"lib/generic.t4": `module lib.generic

pub func id<T>(x: T) -> T:
    return x
`,
		"app/main.t4": `module app.main
import lib.types as types
import lib.generic as generic

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var box: types.Box = types.Box(holder: types.Holder(cb: add1))
    box.holder.cb = generic.id
    return box.holder.cb(42)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGlobalSymbolBackedSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `val cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingGlobalDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

val cb: fn(Int) -> Int throws Boom = risky

func risky(x: Int) -> Int throws Boom:
    return x + 1

func caller() -> Int throws Boom:
    return try cb(41)

func main() -> Int:
    return catch caller():
    case Boom.bad:
        0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingImportedGlobalDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub enum Boom:
    case bad

pub func risky(x: Int) -> Int throws Boom:
    return x + 1
`,
		"app/main.t4": `module app.main
import lib.math as math

val cb: fn(Int) -> Int throws math.Boom = math.risky

func caller() -> Int throws math.Boom:
    return try cb(41)

func main() -> Int:
    return catch caller():
    case math.Boom.bad:
        0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGlobalClosureLiteralSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `val cb: fn(Int) -> Int = fn(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGlobalInitializerDiagnostics(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		entry string
		want  string
	}{
		{
			name: "same module non function identifier",
			files: map[string]string{
				"main.t4": `val n: Int = 1
val cb: fn(Int) -> Int = n

func main() -> Int:
    return 0
`,
			},
			entry: "main.t4",
			want:  "function-typed global 'cb' initializer must be a same-module named function symbol for the supported fnptr ABI",
		},
		{
			name: "imported non function identifier",
			files: map[string]string{
				"lib/math.t4": `module lib.math

pub val n: Int = 1
`,
				"app/main.t4": `module app.main
import lib.math as math

val cb: fn(Int) -> Int = math.n

func main() -> Int:
    return 0
`,
			},
			entry: "app/main.t4",
			want:  "function-typed global 'cb' initializer must be an imported public function symbol for the supported fnptr ABI",
		},
		{
			name: "expression initializer",
			files: map[string]string{
				"main.t4": `val cb: fn(Int) -> Int = add1(1)

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return 0
`,
			},
			entry: "main.t4",
			want:  "function-typed global 'cb' must be initialized with a direct named function symbol or closure literal for the supported fnptr ABI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := buildOnlyFiles(t, tt.files, tt.entry)
			if err == nil {
				t.Fatalf("expected function-typed global initializer diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestBuildFunctionTypedMutableGlobalClosureLiteralInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `var cb: fn(Int) -> Int = fn(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingMutableGlobalDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

var cb: fn(Int) -> Int throws Boom = risky

func risky(x: Int) -> Int throws Boom:
    return x + 1

func caller() -> Int throws Boom:
    return try cb(41)

func main() -> Int:
    return catch caller():
    case Boom.bad:
        0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingMutableGlobalReassignmentDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

var cb: fn(Int) -> Int throws Boom = add1

func add1(x: Int) -> Int throws Boom:
    return x + 1

func add2(x: Int) -> Int throws Boom:
    return x + 2

func caller() -> Int throws Boom:
    cb = add2
    return try cb(40)

func main() -> Int:
    return catch caller():
    case Boom.bad:
        0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalReassignmentDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    cb = add2
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalReassignmentCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    cb = add2
    return apply(cb, 40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingMutableGlobalReassignmentCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

var cb: fn(Int) -> Int throws Boom = add1

func add1(x: Int) -> Int throws Boom:
    return x + 1

func add2(x: Int) -> Int throws Boom:
    return x + 2

func apply(f: fn(Int) -> Int throws Boom, x: Int) -> Int throws Boom:
    return try f(x)

func caller() -> Int throws Boom:
    cb = add2
    return try apply(cb, 40)

func main() -> Int:
    return catch caller():
    case Boom.bad:
        0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalLocalReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    cb = add2
    var f: fn(Int) -> Int = add1
    f = cb
    return f(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalLocalReassignmentCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    cb = add2
    var f: fn(Int) -> Int = add1
    f = cb
    return apply(f, 40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalStoredInStructFieldSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    cb = add2
    let holder: Holder = Holder(cb: cb)
    return holder.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingMutableGlobalStoredInStructFieldDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

struct Holder:
    cb: fn(Int) -> Int throws Boom

var cb: fn(Int) -> Int throws Boom = add1

func add1(x: Int) -> Int throws Boom:
    return x + 1

func add2(x: Int) -> Int throws Boom:
    return x + 2

func caller() -> Int throws Boom:
    cb = add2
    let holder: Holder = Holder(cb: cb)
    return try holder.cb(40)

func main() -> Int:
    return catch caller():
    case Boom.bad:
        0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalStoredInStructFieldCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    cb = add2
    let holder: Holder = Holder(cb: cb)
    return apply(holder.cb, 40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalNestedStructFieldSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    cb = add2
    let box: Box = Box(holder: Holder(cb: cb))
    return box.holder.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalNestedStructFieldCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    cb = add2
    let box: Box = Box(holder: Holder(cb: cb))
    return apply(box.holder.cb, 40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalNestedStructFieldReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    cb = add2
    var box: Box = Box(holder: Holder(cb: add1))
    box.holder.cb = cb
    return box.holder.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalStoredInEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Choice:
    case some(fn(Int) -> Int)

var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    cb = add2
    let choice: Choice = Choice.some(cb)
    match choice:
    case Choice.some(f):
        return f(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingMutableGlobalStoredInEnumPayloadDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

enum Choice:
    case some(fn(Int) -> Int throws Boom)

var cb: fn(Int) -> Int throws Boom = add1

func add1(x: Int) -> Int throws Boom:
    return x + 1

func add2(x: Int) -> Int throws Boom:
    return x + 2

func caller() -> Int throws Boom:
    cb = add2
    let choice: Choice = Choice.some(cb)
    match choice:
        case Choice.some(f):
            return try f(40)

func main() -> Int:
    return catch caller():
    case Boom.bad:
        0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalStoredInEnumPayloadCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Choice:
    case some(fn(Int) -> Int)

var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    cb = add2
    let choice: Choice = Choice.some(cb)
    match choice:
    case Choice.some(f):
        return apply(f, 40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalStructFieldReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    cb = add2
    var holder: Holder = Holder(cb: add1)
    holder.cb = cb
    return holder.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalReturnedStructFieldSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func makeHolder() -> Holder:
    return Holder(cb: cb)

func main() -> Int:
    cb = add2
    let holder: Holder = makeHolder()
    return holder.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalReturnedStructFieldCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func makeHolder() -> Holder:
    return Holder(cb: cb)

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    cb = add2
    let holder: Holder = makeHolder()
    return apply(holder.cb, 40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalEnumPayloadReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Choice:
    case some(fn(Int) -> Int)

var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    cb = add2
    var choice: Choice = Choice.some(add1)
    choice = Choice.some(cb)
    match choice:
    case Choice.some(f):
        return f(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalReturnedEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Choice:
    case some(fn(Int) -> Int)

var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func makeChoice() -> Choice:
    return Choice.some(cb)

func main() -> Int:
    cb = add2
    let choice: Choice = makeChoice()
    match choice:
    case Choice.some(f):
        return f(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalReturnedEnumPayloadCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Choice:
    case some(fn(Int) -> Int)

var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func makeChoice() -> Choice:
    return Choice.some(cb)

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    cb = add2
    let choice: Choice = makeChoice()
    match choice:
    case Choice.some(f):
        return apply(f, 40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func pick() -> fn(Int) -> Int:
    return cb

func main() -> Int:
    cb = add2
    let f: fn(Int) -> Int = pick()
    return f(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableGlobalReturnCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `var cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func pick() -> fn(Int) -> Int:
    return cb

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    cb = add2
    return apply(pick(), 40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGlobalCanInitializeAndAssignLocal(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `val plusOne: fn(Int) -> Int = add1
val plusTwo: fn(Int) -> Int = add2

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    let first: fn(Int) -> Int = plusOne
    var current: fn(Int) -> Int = first
    current = plusTwo
    return current(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGlobalCanBeCallbackArgument(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `val plusOne: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(plusOne, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGlobalCanUseImportedFunctionSymbol(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub func add2(x: Int) -> Int:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

val cb: fn(Int) -> Int = math.add2

func main() -> Int:
    return cb(40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGlobalFromImportedSymbolCanBeCallbackArgument(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub func add2(x: Int) -> Int:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

val cb: fn(Int) -> Int = math.add2

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    return apply(cb, 40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedGlobalDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub val cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

func main() -> Int:
    return math.cb(40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingImportedPublicGlobalDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub enum Boom:
    case bad

pub val cb: fn(Int) -> Int throws Boom = risky

pub func risky(x: Int) -> Int throws Boom:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

func caller() -> Int throws math.Boom:
    return try math.cb(40)

func main() -> Int:
    return catch caller():
    case math.Boom.bad:
        0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingImportedPublicGlobalLocalAliasDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub enum Boom:
    case bad

pub val cb: fn(Int) -> Int throws Boom = risky

pub func risky(x: Int) -> Int throws Boom:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

func caller() -> Int throws math.Boom:
    let local: fn(Int) -> Int throws math.Boom = math.cb
    return try local(40)

func main() -> Int:
    return catch caller():
    case math.Boom.bad:
        0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingImportedPublicGlobalMutableLocalReassignmentDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub enum Boom:
    case bad

pub val cb: fn(Int) -> Int throws Boom = risky

pub func risky(x: Int) -> Int throws Boom:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

func fallback(x: Int) -> Int throws math.Boom:
    return x

func caller() -> Int throws math.Boom:
    var local: fn(Int) -> Int throws math.Boom = fallback
    local = math.cb
    return try local(40)

func main() -> Int:
    return catch caller():
    case math.Boom.bad:
        0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingImportedPublicGlobalDirectCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub enum Boom:
    case bad

pub val cb: fn(Int) -> Int throws Boom = risky

pub func risky(x: Int) -> Int throws Boom:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

func apply(f: fn(Int) -> Int throws math.Boom, x: Int) -> Int throws math.Boom:
    return try f(x)

func caller() -> Int throws math.Boom:
    return try apply(math.cb, 40)

func main() -> Int:
    return catch caller():
    case math.Boom.bad:
        0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingImportedPublicGlobalStructFieldReassignmentDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub enum Boom:
    case bad

pub val cb: fn(Int) -> Int throws Boom = risky

pub func risky(x: Int) -> Int throws Boom:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

struct Holder:
    cb: fn(Int) -> Int throws math.Boom

func fallback(x: Int) -> Int throws math.Boom:
    return x

func caller() -> Int throws math.Boom:
    var holder: Holder = Holder(cb: fallback)
    holder.cb = math.cb
    return try holder.cb(40)

func main() -> Int:
    return catch caller():
    case math.Boom.bad:
        0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingImportedPublicGlobalStructFieldInitializerDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub enum Boom:
    case bad

pub val cb: fn(Int) -> Int throws Boom = risky

pub func risky(x: Int) -> Int throws Boom:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

struct Holder:
    cb: fn(Int) -> Int throws math.Boom

func caller() -> Int throws math.Boom:
    let holder: Holder = Holder(cb: math.cb)
    return try holder.cb(40)

func main() -> Int:
    return catch caller():
    case math.Boom.bad:
        0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingImportedPublicGlobalEnumPayloadReassignmentDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub enum Boom:
    case bad

pub val cb: fn(Int) -> Int throws Boom = risky

pub func risky(x: Int) -> Int throws Boom:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

enum Choice:
    case some(fn(Int) -> Int throws math.Boom)

func fallback(x: Int) -> Int throws math.Boom:
    return x

func caller() -> Int throws math.Boom:
    var choice: Choice = Choice.some(fallback)
    choice = Choice.some(math.cb)
    match choice:
        case Choice.some(cb):
            return try cb(40)

func main() -> Int:
    return catch caller():
    case math.Boom.bad:
        0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedGlobalDirectCallAllowsArgumentLabelsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub val cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

func main() -> Int:
    return math.cb(value: 40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedGlobalLocalAndCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub val cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    let local: fn(Int) -> Int = math.cb
    return apply(local, 40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedGlobalDirectCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub val cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    return apply(math.cb, 40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedMutableGlobalDirectCallRejected(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tests := []struct {
		name string
		app  string
		want string
	}{
		{
			name: "direct call",
			app: `module app.main
import lib.math as math

func main() -> Int:
    let _: Int = math.select_add2()
    return math.cb(40)
`,
			want: "imported mutable function-typed global 'math.cb' cannot be called directly across module boundary; cross-module mutable global-data ABI is not available, expose a module-local function wrapper or immutable public function-typed global",
		},
		{
			name: "local initializer",
			app: `module app.main
import lib.math as math

func main() -> Int:
    let local: fn(Int) -> Int = math.cb
    return local(40)
`,
			want: "imported mutable function-typed global 'math.cb' cannot be used across module boundary; cross-module mutable global-data ABI is not available, expose a module-local function wrapper or immutable public function-typed global",
		},
		{
			name: "callback argument",
			app: `module app.main
import lib.math as math

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(math.cb, 40)
`,
			want: "imported mutable function-typed global 'math.cb' cannot be used across module boundary; cross-module mutable global-data ABI is not available, expose a module-local function wrapper or immutable public function-typed global",
		},
		{
			name: "function typed return",
			app: `module app.main
import lib.math as math

func pick() -> fn(Int) -> Int:
    return math.cb

func main() -> Int:
    let local: fn(Int) -> Int = pick()
    return local(40)
`,
			want: "imported mutable function-typed global 'math.cb' cannot be used across module boundary; cross-module mutable global-data ABI is not available, expose a module-local function wrapper or immutable public function-typed global",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{
				"lib/math.t4": `module lib.math

pub var cb: fn(Int) -> Int = add1

pub func add1(x: Int) -> Int:
    return x + 1

pub func add2(x: Int) -> Int:
    return x + 2

pub func select_add2() -> Int:
    cb = add2
    return 0
`,
				"app/main.t4": tt.app,
			}

			err := buildOnlyFiles(t, files, "app/main.t4")
			if err == nil {
				t.Fatalf("expected imported mutable function-typed global diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestBuildFunctionTypedImportedGlobalMutableLocalReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub val cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var current: fn(Int) -> Int = add1
    current = math.cb
    return current(40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedGlobalStructFieldReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub val cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math as math

struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var holder: Holder = Holder(cb: add1)
    holder.cb = math.cb
    return holder.cb(40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedSelectiveImportedGlobalDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub val cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math.{cb}

func main() -> Int:
    return cb(40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedSelectiveImportedGlobalLocalAndCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub val cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math.{cb}

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    let local: fn(Int) -> Int = cb
    return apply(local, 40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedSelectiveImportedGlobalMutableLocalReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub val cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math.{cb}

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var current: fn(Int) -> Int = add1
    current = cb
    return current(40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedSelectiveImportedGlobalStructFieldReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub val cb: fn(Int) -> Int = add2

pub func add2(x: Int) -> Int:
    return x + 2
`,
		"app/main.t4": `module app.main
import lib.math.{cb}

struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var holder: Holder = Holder(cb: add1)
    holder.cb = cb
    return holder.cb(40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGlobalCanUseGenericFunctionSymbol(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"app/main.t4": `module app.main

val cb: fn(Int) -> Int = keep

func keep<T>(x: T) -> T:
    return x

func main() -> Int:
    return cb(42)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedGlobalCanUseImportedGenericFunctionSymbol(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub func keep<T>(x: T) -> T:
    return x
`,
		"app/main.t4": `module app.main
import lib.math as math

val cb: fn(Int) -> Int = math.keep

func main() -> Int:
    return cb(42)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCallableMVPRejectsUnsupportedForms(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "generic literal binding with capture",
			src: `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn<T>(x: T) -> T:
        let _: Int = base
        return x
    return f(41)
`,
			want: "generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly",
		},
		{
			name: "generic direct callback literal with capture",
			src: `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    return apply(fn<T>(x: T) -> T:
        let _: Int = base
        return x
    , 41)
`,
			want: "callback argument 'closure literal' captures local 'base'; generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly",
		},
		{
			name: "generic struct field literal with capture",
			src: `struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let base: Int = 1
    let holder: Holder = Holder(cb: fn<T>(x: T) -> T:
        let _: Int = base
        return x
    )
    return holder.cb(41)
`,
			want: "generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly",
		},
		{
			name: "generic nested struct initializer literal with capture",
			src: `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func main() -> Int:
    let base: Int = 1
    let box: Box = Box(holder: Holder(cb: fn<T>(x: T) -> T:
        let _: Int = base
        return x
    ))
    return box.holder.cb(41)
`,
			want: "generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly",
		},
		{
			name: "generic enum payload literal with capture",
			src: `enum Choice:
    case some(fn(Int) -> Int)

func main() -> Int:
    let base: Int = 1
    let choice: Choice = Choice.some(fn<T>(x: T) -> T:
        let _: Int = base
        return x
    )
    match choice:
        case Choice.some(cb):
            return cb(41)
`,
			want: "generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly",
		},
		{
			name: "generic return literal with capture",
			src: `func pick() -> fn(Int) -> Int:
    let base: Int = 1
    return fn<T>(x: T) -> T:
        let _: Int = base
        return x

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return cb(41)
`,
			want: "generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly",
		},
		{
			name: "generic mutable local reassignment literal with capture",
			src: `func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let base: Int = 1
    var f: fn(Int) -> Int = add1
    f = fn<T>(x: T) -> T:
        let _: Int = base
        return x
    return f(41)
`,
			want: "generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly",
		},
		{
			name: "generic struct field reassignment literal with capture",
			src: `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let base: Int = 1
    var holder: Holder = Holder(cb: add1)
    holder.cb = fn<T>(x: T) -> T:
        let _: Int = base
        return x
    return holder.cb(41)
`,
			want: "generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly",
		},
		{
			name: "generic nested struct field reassignment literal with capture",
			src: `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let base: Int = 1
    var box: Box = Box(holder: Holder(cb: add1))
    box.holder.cb = fn<T>(x: T) -> T:
        let _: Int = base
        return x
    return box.holder.cb(41)
`,
			want: "generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly",
		},
		{
			name: "generic enum payload reassignment literal with capture",
			src: `enum Choice:
    case some(fn(Int) -> Int)

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let base: Int = 1
    var choice: Choice = Choice.some(add1)
    choice = Choice.some(fn<T>(x: T) -> T:
        let _: Int = base
        return x
    )
    match choice:
        case Choice.some(cb):
            return cb(41)
`,
			want: "generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly",
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
			want: "function-typed local 'f' throws type mismatch: expected '', got 'Boom'",
		},
		{
			name: "function typed return closure literal ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

func pick() -> fn(Int) -> Int:
    let box: PtrBox = PtrBox(p: 0)
    return fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`,
			want: "function-typed return 'closure literal' captures unsupported local 'box' of type 'PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI",
		},
		{
			name: "function typed return oversized ptr field capture reports heap escape diagnostic",
			src: `struct PtrBox:
    p: ptr

func pick() -> fn(Int) -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let box: PtrBox = PtrBox(p: 0)
    return fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x + one + two + three + four + five + six + seven + eight

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`,
			want: "escaped function value captures local 'box' of type 'PtrBox'; pointer or resource captures require an explicit ownership transfer model",
		},
		{
			name: "function typed return oversized mutable capture reports heap escape diagnostic",
			src: `func pick() -> fn(Int) -> Int:
    var total: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    return fn(x: Int) -> Int:
        return x + total + two + three + four + five + six + seven + eight + nine

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`,
			want: "heap-escaped function value captures mutable local 'total'; mutable by-reference captures require a proven lifetime and synchronization model",
		},
		{
			name: "function typed global oversized ptr field capture reports resource escape diagnostic",
			src: `struct PtrBox:
    p: ptr

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let box: PtrBox = PtrBox(p: 0)
    cb = fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x + one + two + three + four + five + six + seven + eight
    return 0
`,
			want: "escaped function value captures local 'box' of type 'PtrBox'; pointer or resource captures require an explicit ownership transfer model",
		},
		{
			name: "function typed local closure literal rejects extra parameter",
			src: `func main() -> Int:
    let f: fn(Int) -> Int = fn(x: Int, y: Int) -> Int:
        return x + y
    return f(41)
`,
			want: "function-typed local 'f' parameter count mismatch: expected 1, got 2",
		},
		{
			name: "throwing direct closure literal callback argument",
			src: `enum Boom:
    case bad

func apply(x: Int, cb: fn(Int) -> Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(41, fn(x: Int) -> Int throws Boom:
        throw Boom.bad
    )
`,
			want: "callback argument 'closure literal' throws type mismatch: expected '', got 'Boom'",
		},
		{
			name: "direct closure literal callback argument parameter count mismatch",
			src: `func apply(x: Int, cb: fn(Int) -> Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(41, fn() -> Int:
        return 1
    )
`,
			want: "callback argument 'closure literal' parameter count mismatch: expected 1, got 0",
		},
		{
			name: "direct closure literal callback argument rejects extra parameter",
			src: `func apply(x: Int, cb: fn(Int) -> Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(41, fn(x: Int, y: Int) -> Int:
        return x + y
    )
`,
			want: "callback argument 'closure literal' parameter count mismatch: expected 1, got 2",
		},
		{
			name: "generic named symbol with uninferable type parameter",
			src: `func keep<T>(x: Int) -> Int:
    return x

func main() -> Int:
    let f: fn(Int) -> Int = keep
    return f(41)
`,
			want: "cannot infer generic argument 'T' for function-typed local 'f'",
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
			want: "throwing function symbol 'risky' cannot initialize function-typed local 'f'; local fnptr ABI requires the declared throws type to match",
		},
		{
			name: "throwing returned symbol without declared throws",
			src: `enum Boom:
    case bad

func risky(x: Int) -> Int throws Boom:
    throw Boom.bad

func pick() -> fn(Int) -> Int:
    return risky

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`,
			want: "returned function symbol 'risky' throws type mismatch: expected '', got 'Boom'",
		},
		{
			name: "throwing struct field call without try",
			src: `enum Boom:
    case bad

struct Holder:
    cb: fn(Int) -> Int throws Boom

func risky(x: Int) -> Int throws Boom:
    throw Boom.bad

func main() -> Int:
    let holder: Holder = Holder(cb: risky)
    return holder.cb(41)
`,
			want: "call to throwing function 'holder.cb' requires try",
		},
		{
			name: "throwing enum payload call without try",
			src: `enum Boom:
    case bad

enum Choice:
    case some(fn(Int) -> Int throws Boom)

func risky(x: Int) -> Int throws Boom:
    throw Boom.bad

func main() -> Int:
    let choice: Choice = Choice.some(risky)
    match choice:
        case Choice.some(cb):
            return cb(41)
`,
			want: "call to throwing function 'cb' requires try",
		},
		{
			name: "throwing global call without try",
			src: `enum Boom:
    case bad

val cb: fn(Int) -> Int throws Boom = risky

func risky(x: Int) -> Int throws Boom:
    throw Boom.bad

func main() -> Int:
    return cb(41)
`,
			want: "call to throwing function 'risky' requires try",
		},
		{
			name: "throwing mutable global reassignment throws mismatch",
			src: `enum Boom:
    case bad

enum Crash:
    case bad

var cb: fn(Int) -> Int throws Boom = risky

func risky(x: Int) -> Int throws Boom:
    return x + 1

func crash(x: Int) -> Int throws Crash:
    throw Crash.bad

func main() -> Int:
    cb = crash
    return 0
`,
			want: "function-typed assignment to 'cb' throws type mismatch: expected 'Boom', got 'Crash'",
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
			want: "function value 'f' cannot escape outside the supported fnptr ABI",
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
			name: "callback argument literal source rejected",
			src: `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(1, 41)
`,
			want: "callback argument for 'apply' must be a supported fnptr source: closure literal, function-typed local/global/struct field, direct named function/closure symbol, or function-typed return call",
		},
		{
			name: "callback argument local non function source rejected",
			src: `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let f: Int = 1
    return apply(f, 41)
`,
			want: "callback argument for 'apply' must be a supported fnptr source: closure literal, function-typed local/global/struct field, direct named function/closure symbol, or function-typed return call",
		},
		{
			name: "function typed parameter call without target set rejected by lowering",
			src: `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return 0
`,
			want: "function-typed parameter 'cb' cannot be lowered as a direct fnptr call without a known target set; pass a direct named function/closure symbol at each call site or use supported function-typed storage before dispatch",
		},
		{
			name: "callback labeled argument type mismatch",
			src: `func call(cb: fn(Int) -> Int) -> Int:
    return cb(x: true)

func main() -> Int:
    return 0
`,
			want: "type mismatch for callback 'cb' arg 1",
		},
		{
			name: "callback mixed labeled and unlabeled arguments",
			src: `func call(cb: fn(Int, Int) -> Int) -> Int:
    return cb(left: 1, 2)

func main() -> Int:
    return 0
`,
			want: "cannot mix labeled and unlabeled arguments in callback 'cb'",
		},
		{
			name: "function typed struct field mixed labeled and unlabeled arguments",
			src: `struct Holder:
    cb: fn(Int, Int) -> Int

func add(x: Int, y: Int) -> Int:
    return x + y

func main() -> Int:
    let holder: Holder = Holder(cb: add)
    return holder.cb(left: 1, 2)
`,
			want: "cannot mix labeled and unlabeled arguments in function-typed struct field call 'holder.cb'",
		},
		{
			name: "function typed struct field wrong argument count",
			src: `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let holder: Holder = Holder(cb: add1)
    return holder.cb()
`,
			want: "wrong argument count for function-typed struct field call 'holder.cb'",
		},
		{
			name: "function typed struct field literal initializer source rejected",
			src: `struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let holder: Holder = Holder(cb: 1)
    return 0
`,
			want: "function-typed struct field 'holder.cb' initializer must be a supported fnptr source: closure literal, function-typed local/global/struct field, direct named function/closure symbol, or function-typed return call",
		},
		{
			name: "function typed struct field generic ptr closure initializer rejected",
			src: `struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let id: ptr = fn<T>(x: T) -> T:
        return x
    let holder: Holder = Holder(cb: id)
    return 0
`,
			want: "generic function symbol 'id' cannot initialize function-typed struct field 'holder.cb'; struct-field fnptr ABI requires a monomorphic target",
		},
		{
			name: "function typed enum payload literal initializer source rejected",
			src: `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(1)
    return 0
`,
			want: "function-typed enum payload 'MaybeCallback.some[1]' initializer must be a supported fnptr source: closure literal, function-typed local/global/struct field, direct named function/closure symbol, or function-typed return call",
		},
		{
			name: "function typed enum payload generic ptr closure initializer rejected",
			src: `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let id: ptr = fn<T>(x: T) -> T:
        return x
    let choice: MaybeCallback = MaybeCallback.some(id)
    return 0
`,
			want: "generic function symbol 'id' cannot initialize function-typed enum payload 'MaybeCallback.some[1]'; enum-payload fnptr ABI requires a monomorphic target",
		},
		{
			name: "function typed local literal initializer source rejected",
			src: `func main() -> Int:
    let f: fn(Int) -> Int = 1
    return 0
`,
			want: "function-typed local 'f' initializer must be a symbol-backed function value, target-set-backed function value, direct named function symbol, or closure literal for the supported fnptr ABI",
		},
		{
			name: "function typed local unknown return call initializer source rejected",
			src: `func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return 0
`,
			want: "function-typed local 'f' initializer call 'pick' must resolve to a function-typed return for the supported fnptr ABI",
		},
		{
			name: "function typed local global non function initializer source rejected",
			src: `val value: Int = 1

func main() -> Int:
    let f: fn(Int) -> Int = value
    return 0
`,
			want: "function-typed local 'f' initializer must be a symbol-backed function value, target-set-backed function value, direct named function symbol, or closure literal for the supported fnptr ABI",
		},
		{
			name: "function typed local field non function initializer source rejected",
			src: `struct Box:
    value: Int

func main() -> Int:
    let box: Box = Box(value: 1)
    let f: fn(Int) -> Int = box.value
    return 0
`,
			want: "function-typed local 'f' initializer must be a symbol-backed function value, target-set-backed function value, direct named function symbol, or closure literal for the supported fnptr ABI",
		},
		{
			name: "function typed struct field explicit type arguments rejected",
			src: `struct Holder:
    cb: fn(Int) -> Int

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let holder: Holder = Holder(cb: add1)
    return holder.cb<Int>(41)
`,
			want: "explicit type arguments are not supported for function-typed struct field call 'holder.cb'; function-typed dispatch uses a monomorphic fnptr ABI, so remove explicit type arguments",
		},
		{
			name: "function typed struct field borrow inout alias rejected",
			src: `struct Holder:
    cb: fn(borrow Int, inout Int) -> Int

func mix(read: borrow Int, write: inout Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var a: Int = 1
    let holder: Holder = Holder(cb: mix)
    return holder.cb(a, a)
`,
			want: "inout argument 'a' aliases borrowed argument in function-typed struct field call 'holder.cb'",
		},
		{
			name: "function typed enum payload borrow inout alias rejected",
			src: `enum MaybeCallback:
    case some(fn(borrow Int, inout Int) -> Int)
    case empty

func mix(read: borrow Int, write: inout Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var a: Int = 1
    let choice: MaybeCallback = MaybeCallback.some(mix)
    match choice:
    case MaybeCallback.some(cb):
        return cb(a, a)
    case MaybeCallback.empty:
        return 0
`,
			want: "inout argument 'a' aliases borrowed argument in function-typed enum payload call 'cb'",
		},
		{
			name: "function typed enum payload explicit type arguments rejected",
			src: `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return cb<Int>(41)
    case MaybeCallback.empty:
        return 0
`,
			want: "explicit type arguments are not supported for function-typed enum payload call 'cb'; function-typed dispatch uses a monomorphic fnptr ABI, so remove explicit type arguments",
		},
		{
			name: "function typed enum payload wrong argument count",
			src: `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return cb()
    case MaybeCallback.empty:
        return 0
`,
			want: "wrong argument count for function-typed enum payload call 'cb'",
		},
		{
			name: "function typed global borrow inout alias rejected",
			src: `val cb: fn(borrow Int, inout Int) -> Int = mix

func mix(read: borrow Int, write: inout Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var a: Int = 1
    return cb(a, a)
`,
			want: "inout argument 'a' aliases borrowed argument in function-typed global call 'cb'",
		},
		{
			name: "function typed global consume non local rejected",
			src: `val cb: fn(consume Int) -> Int = take

func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    return cb(1)
`,
			want: "consume argument for function-typed global call 'cb' must be a local value",
		},
		{
			name: "function typed global wrong argument count rejected",
			src: `val cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return cb()
`,
			want: "wrong argument count for function-typed global call 'cb'",
		},
		{
			name: "function typed global type mismatch rejected",
			src: `val cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return cb(true)
`,
			want: "type mismatch for function-typed global call 'cb' arg 1",
		},
		{
			name: "function typed global explicit type arguments rejected",
			src: `val cb: fn(Int) -> Int = add1

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return cb<Int>(41)
`,
			want: "explicit type arguments are not supported for function-typed global call 'cb'; function-typed dispatch uses a monomorphic fnptr ABI, so remove explicit type arguments",
		},
		{
			name: "captured function typed local explicit type arguments rejected",
			src: `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f<Int>(41)
`,
			want: "explicit type arguments are not supported for function-typed callback 'f'; function-typed dispatch uses a monomorphic fnptr ABI, so remove explicit type arguments",
		},
		{
			name: "captured function typed local wrong argument count rejected",
			src: `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f()
`,
			want: "wrong argument count for function-typed callback 'f'",
		},
		{
			name: "captured function typed local type mismatch rejected",
			src: `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f(true)
`,
			want: "type mismatch for function-typed callback 'f' arg 1",
		},
		{
			name: "captured function typed local mixed labeled arguments rejected",
			src: `func main() -> Int:
    let base: Int = 1
    let f: fn(Int, Int) -> Int = fn(x: Int, y: Int) -> Int:
        return x + y + base
    return f(left: 1, 2)
`,
			want: "cannot mix labeled and unlabeled arguments in function-typed callback 'f'",
		},
		{
			name: "direct closure literal callback ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    return apply(fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    , 41)
`,
			want: "callback argument 'closure literal' captures unsupported local 'box' of type 'PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI",
		},
		{
			name: "function typed local initializer ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    return 0
`,
			want: "function-typed storage 'f' captures unsupported local 'box' of type 'PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI",
		},
		{
			name: "function typed struct field initializer ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let holder: Holder = Holder(cb: fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    )
    return 0
`,
			want: "function-typed storage 'holder.cb' captures unsupported local 'box' of type 'PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI",
		},
		{
			name: "function typed enum payload initializer ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    )
    return 0
`,
			want: "function-typed storage 'MaybeCallback.some[1]' captures unsupported local 'box' of type 'PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI",
		},
		{
			name: "function typed mutable local reassignment ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    var f: fn(Int) -> Int = add0
    f = fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    return 0
`,
			want: "function-typed storage 'f' captures unsupported local 'box' of type 'PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI",
		},
		{
			name: "function typed mutable local reassignment literal source rejected",
			src: `func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var f: fn(Int) -> Int = add0
    f = 1
    return 0
`,
			want: "function-typed assignment to 'f' must use a supported fnptr source: closure literal, function-typed local/global/struct field, direct named function/closure symbol, or function-typed return call",
		},
		{
			name: "function typed mutable local reassignment local non function source rejected",
			src: `func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let value: Int = 1
    var f: fn(Int) -> Int = add0
    f = value
    return 0
`,
			want: "function-typed assignment to 'f' must use a supported fnptr source: closure literal, function-typed local/global/struct field, direct named function/closure symbol, or function-typed return call",
		},
		{
			name: "function typed mutable local reassignment generic ptr closure alias rejected",
			src: `func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var f: fn(Int) -> Int = add0
    let id: ptr = fn<T>(x: T) -> T:
        return x
    f = id
    return 0
`,
			want: "generic function symbol 'id' cannot be assigned to function-typed target 'f'; assignment fnptr ABI requires a monomorphic target",
		},
		{
			name: "function typed mutable local reassignment throwing ptr closure alias rejected",
			src: `enum Boom:
    case bad

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var f: fn(Int) -> Int = add0
    let risky: ptr = fn(x: Int) -> Int throws Boom:
        throw Boom.bad
    f = risky
    return 0
`,
			want: "throwing function symbol 'risky' cannot be assigned to function-typed target 'f'; assignment fnptr ABI requires the target's declared throws type to match",
		},
		{
			name: "function typed mutable local reassignment unknown return call source rejected",
			src: `func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var f: fn(Int) -> Int = add0
    f = pick()
    return 0
`,
			want: "function-typed assignment to 'f' initializer call 'pick' must resolve to a function-typed return for the supported fnptr ABI",
		},
		{
			name: "function typed struct field reassignment ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

struct Holder:
    cb: fn(Int) -> Int

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    var holder: Holder = Holder(cb: add0)
    holder.cb = fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    return 0
`,
			want: "function-typed storage 'holder.cb' captures unsupported local 'box' of type 'PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI",
		},
		{
			name: "function typed enum payload reassignment ptr field capture rejected",
			src: `struct PtrBox:
    p: ptr

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    )
    return 0
`,
			want: "function-typed storage 'MaybeCallback.some[1]' captures unsupported local 'box' of type 'PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI",
		},
		{
			name: "direct callback signature parameter mismatch",
			src: `func as_bool(flag: Bool) -> Int:
    if flag:
        return 1
    return 0

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(as_bool, 1)
`,
			want: "callback function symbol 'as_bool' parameter 1 type mismatch: expected 'i32', got 'bool'",
		},
		{
			name: "direct callback signature return mismatch",
			src: `func truthy(x: Int) -> Bool:
    return true

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(truthy, 1)
`,
			want: "callback function symbol 'truthy' return type mismatch: expected 'i32', got 'bool'",
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

func TestBuildFunctionTypedCrossModuleUnsupportedCaptureDiagnostics(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			name: "imported direct closure literal callback ptr field capture rejected",
			files: map[string]string{
				"lib/callbacks.t4": `module lib.callbacks

pub func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)
`,
				"app/main.t4": `module app.main
import lib.callbacks as callbacks

struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    return callbacks.apply(fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    , 41)
`,
			},
			want: "callback argument 'closure literal' captures unsupported local 'box' of type 'app.main.PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI",
		},
		{
			name: "imported enum payload constructor ptr field capture rejected",
			files: map[string]string{
				"lib/types.t4": `module lib.types

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty
`,
				"app/main.t4": `module app.main
import lib.types as types

struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let choice: types.MaybeCallback = types.MaybeCallback.some(fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    )
    return 0
`,
			},
			want: "function-typed storage 'lib.types.MaybeCallback.some[1]' captures unsupported local 'box' of type 'app.main.PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI",
		},
		{
			name: "imported struct field constructor ptr field capture rejected",
			files: map[string]string{
				"lib/types.t4": `module lib.types

pub struct Holder:
    cb: fn(Int) -> Int
`,
				"app/main.t4": `module app.main
import lib.types as types

struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let holder: types.Holder = types.Holder(cb: fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    )
    return holder.cb(41)
`,
			},
			want: "function-typed storage 'holder.cb' captures unsupported local 'box' of type 'app.main.PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI",
		},
		{
			name: "imported struct field constructor argument ptr field capture rejected",
			files: map[string]string{
				"lib/types.t4": `module lib.types

pub struct Holder:
    cb: fn(Int) -> Int

pub func call(holder: Holder, x: Int) -> Int:
    return holder.cb(x)
`,
				"app/main.t4": `module app.main
import lib.types as types

struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    return types.call(types.Holder(cb: fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    ), 41)
`,
			},
			want: "function-typed storage 'lib.types.Holder.cb' captures unsupported local 'box' of type 'app.main.PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI",
		},
		{
			name: "imported function typed return ptr field capture rejected",
			files: map[string]string{
				"lib/maker.t4": `module lib.maker

struct PtrBox:
    p: ptr

pub func make() -> fn(Int) -> Int:
    let box: PtrBox = PtrBox(p: 0)
    return fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
`,
				"app/main.t4": `module app.main
import lib.maker as maker

func main() -> Int:
    let cb: fn(Int) -> Int = maker.make()
    return cb(41)
`,
			},
			want: "function-typed return 'closure literal' captures unsupported local 'box' of type 'lib.maker.PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := buildOnlyFiles(t, tc.files, "app/main.t4")
			if err == nil {
				t.Fatalf("expected diagnostic containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}
