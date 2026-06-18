package compiler_test

import (
	"runtime"
	"testing"
)

func TestBuildFunctionTypedCapturedClosureImportedReturnedStructEnumPayloadMutableGlobalSnapshotSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/pack.t4": `module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let delta: Int = 2
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int:
        return x + delta
    ))
`,
		"app/main.t4": `module app.main
import lib.pack as pack

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let box: pack.Box = pack.makeBox()
    match box.choice:
    case pack.MaybeCallback.some(local):
        cb = local
        return 0
    case pack.MaybeCallback.empty:
        return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureImportedReturnedEnumPayloadMutableGlobalSnapshotSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/pack.t4": `module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice() -> MaybeCallback:
    let delta: Int = 2
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + delta
    )
`,
		"app/main.t4": `module app.main
import lib.pack as pack

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let choice: pack.MaybeCallback = pack.makeChoice()
    match choice:
    case pack.MaybeCallback.some(local):
        cb = local
        return 0
    case pack.MaybeCallback.empty:
        return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureMutableGlobalReassignmentCrossModuleReturnDirectCallSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/state.t4": `module lib.state

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

pub func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int:
        return x + delta
    return 0

pub func current() -> fn(Int) -> Int:
    return cb
`,
		"app/main.t4": `module app.main
import lib.state as state

func main() -> Int:
    let ignored: Int = state.configure()
    let f: fn(Int) -> Int = state.current()
    return f(40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureMutableGlobalCrossModuleReturnDirectCallbackArgumentSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/state.t4": `module lib.state

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

pub func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int:
        return x + delta
    return 0

pub func current() -> fn(Int) -> Int:
    return cb
`,
		"app/main.t4": `module app.main
import lib.state as state

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    let ignored: Int = state.configure()
    return apply(state.current(), 40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureMutableGlobalCrossModuleReturnMutableLocalReassignmentSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/state.t4": `module lib.state

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

pub func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int:
        return x + delta
    return 0

pub func current() -> fn(Int) -> Int:
    return cb
`,
		"app/main.t4": `module app.main
import lib.state as state

func fallback(x: Int) -> Int:
    return x

func main() -> Int:
    let ignored: Int = state.configure()
    var f: fn(Int) -> Int = fallback
    f = state.current()
    return f(40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureMutableGlobalCrossModuleReturnStructFieldDirectCallSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/state.t4": `module lib.state

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

pub func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int:
        return x + delta
    return 0

pub func current() -> fn(Int) -> Int:
    return cb
`,
		"app/main.t4": `module app.main
import lib.state as state

struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let ignored: Int = state.configure()
    let holder: Holder = Holder(cb: state.current())
    return holder.cb(40)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureMutableGlobalCrossModuleReturnEnumPayloadDirectCallSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/state.t4": `module lib.state

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

pub func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int:
        return x + delta
    return 0

pub func current() -> fn(Int) -> Int:
    return cb
`,
		"app/main.t4": `module app.main
import lib.state as state

enum Choice:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let ignored: Int = state.configure()
    let choice: Choice = Choice.some(state.current())
    match choice:
    case Choice.some(local):
        return local(40)
    case Choice.empty:
        return 0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingCapturedClosureMutableGlobalReassignmentDirectTrySmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum Boom:
    case bad

var cb: fn(Int) -> Int throws Boom = identity

func identity(x: Int) -> Int throws Boom:
    return x

func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int throws Boom:
        return x + delta
    return 0

func caller() -> Int throws Boom:
    return try cb(40)

func main() -> Int:
    let ignored: Int = configure()
    return catch caller():
    case Boom.bad:
        0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingCapturedClosureMutableGlobalCrossModuleReturnDirectTrySmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/state.t4": `module lib.state

pub enum Boom:
    case bad

var cb: fn(Int) -> Int throws Boom = identity

func identity(x: Int) -> Int throws Boom:
    return x

pub func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int throws Boom:
        return x + delta
    return 0

pub func current() -> fn(Int) -> Int throws Boom:
    return cb
`,
		"app/main.t4": `module app.main
import lib.state as state

func caller() -> Int throws state.Boom:
    let f: fn(Int) -> Int throws state.Boom = state.current()
    return try f(40)

func main() -> Int:
    let ignored: Int = state.configure()
    return catch caller():
    case state.Boom.bad:
        0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureMutableGlobalStructFieldInitializerDirectCallSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int:
        return x + delta
    return 0

func main() -> Int:
    let ignored: Int = configure()
    let holder: Holder = Holder(cb: cb)
    return holder.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureMutableGlobalStructFieldReassignmentDirectCallSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int:
        return x + delta
    return 0

func main() -> Int:
    let ignored: Int = configure()
    var holder: Holder = Holder(cb: identity)
    holder.cb = cb
    return holder.cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureMutableGlobalEnumPayloadReassignmentDirectCallSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum Choice:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int:
        return x + delta
    return 0

func main() -> Int:
    let ignored: Int = configure()
    var choice: Choice = Choice.empty
    choice = Choice.some(cb)
    match choice:
    case Choice.some(local):
        return local(40)
    case Choice.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureMutableGlobalEnumPayloadInitializerDirectCallSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum Choice:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int:
        return x + delta
    return 0

func main() -> Int:
    let ignored: Int = configure()
    let choice: Choice = Choice.some(cb)
    match choice:
    case Choice.some(local):
        return local(40)
    case Choice.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
