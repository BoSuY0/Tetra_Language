package compiler_test

import (
	"runtime"
	"testing"
)

func TestBuildFunctionTypedCapturedClosureMutableGlobalReassignmentDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int:
        return x + delta
    return 0

func read() -> Int:
    return cb(40)

func main() -> Int:
    let ignored: Int = configure()
    return read()
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedPtrClosureMutableGlobalReassignmentDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    let captured: ptr = fn(x: Int) -> Int:
        return x + delta
    cb = captured
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureMutableGlobalReassignmentCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int:
        return x + delta
    return 0

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    let ignored: Int = configure()
    return apply(cb, 40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureMutableGlobalReassignmentReturnDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    cb = fn(x: Int) -> Int:
        return x + delta
    return 0

func current() -> fn(Int) -> Int:
    return cb

func main() -> Int:
    let ignored: Int = configure()
    let f: fn(Int) -> Int = current()
    return f(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnCallMutableGlobalReassignmentDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func configure() -> Int:
    cb = make()
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnLocalMutableGlobalReassignmentDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func configure() -> Int:
    let local: fn(Int) -> Int = make()
    cb = local
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnMutableLocalReassignmentDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func main() -> Int:
    var local: fn(Int) -> Int = identity
    local = make()
    return local(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnMutableLocalMutableGlobalReassignmentDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func configure() -> Int:
    var local: fn(Int) -> Int = identity
    local = make()
    cb = local
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnStructFieldMutableGlobalReassignmentDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func configure() -> Int:
    var holder: Holder = Holder(cb: identity)
    holder.cb = make()
    cb = holder.cb
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureStructFieldMutableGlobalSnapshotSmoke(t *testing.T) {
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
    let holder: Holder = Holder(cb: fn(x: Int) -> Int:
        return x + delta
    )
    cb = holder.cb
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnNestedStructFieldMutableGlobalReassignmentDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func configure() -> Int:
    var box: Box = Box(holder: Holder(cb: identity))
    box.holder.cb = make()
    cb = box.holder.cb
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnWholeStructMutableGlobalReassignmentDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func configure() -> Int:
    var holder: Holder = Holder(cb: identity)
    holder = Holder(cb: make())
    cb = holder.cb
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureWholeStructMutableGlobalSnapshotSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    var holder: Holder = Holder(cb: identity)
    holder = Holder(cb: fn(x: Int) -> Int:
        return x + delta
    )
    cb = holder.cb
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnWholeNestedStructMutableGlobalReassignmentDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func configure() -> Int:
    var box: Box = Box(holder: Holder(cb: identity))
    box = Box(holder: Holder(cb: make()))
    cb = box.holder.cb
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureWholeNestedStructMutableGlobalSnapshotSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    var box: Box = Box(holder: Holder(cb: identity))
    box = Box(holder: Holder(cb: fn(x: Int) -> Int:
        return x + delta
    ))
    cb = box.holder.cb
    return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnEnumPayloadMutableGlobalReassignmentDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let delta: Int = 2
    return fn(x: Int) -> Int:
        return x + delta

func configure() -> Int:
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(make())
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureEnumPayloadMutableGlobalSnapshotSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int:
        return x + delta
    )
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureWholeEnumMutableGlobalSnapshotSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func configure() -> Int:
    let delta: Int = 2
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(fn(x: Int) -> Int:
        return x + delta
    )
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnedStructEnumPayloadMutableGlobalSnapshotSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

struct Box:
    choice: MaybeCallback

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func makeBox() -> Box:
    let delta: Int = 2
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int:
        return x + delta
    ))

func configure() -> Int:
    let box: Box = makeBox()
    match box.choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureReturnedEnumPayloadMutableGlobalSnapshotSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = identity

func identity(x: Int) -> Int:
    return x

func makeChoice() -> MaybeCallback:
    let delta: Int = 2
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + delta
    )

func configure() -> Int:
    let choice: MaybeCallback = makeChoice()
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    let ignored: Int = configure()
    return cb(40)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
