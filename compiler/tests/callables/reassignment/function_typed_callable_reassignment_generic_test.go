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

func TestBuildFunctionTypedGenericSymbolCrossModuleNestedStructFieldReassignmentSmoke(
	t *testing.T,
) {
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
