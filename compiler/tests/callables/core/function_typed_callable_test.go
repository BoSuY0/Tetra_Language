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

func TestBuildFunctionTypedImportedStructFieldParamDirectConstructorCapturedClosureSmoke(
	t *testing.T,
) {
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

func TestBuildFunctionTypedImportedStructFieldParamDirectConstructorCapturedPtrClosureSmoke(
	t *testing.T,
) {
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

func TestBuildFunctionTypedSelectiveImportedStructFieldParamDirectConstructorCapturedClosureSmoke(
	t *testing.T,
) {
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

func TestBuildFunctionTypedStructFieldCapturedClosureFromCrossModuleReturnedStructSmoke(
	t *testing.T,
) {
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

func TestBuildFunctionTypedNestedStructFieldReturnedStructMultiTargetInitializerSmoke(
	t *testing.T,
) {
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
