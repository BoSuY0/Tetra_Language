package compiler_test

import (
	"runtime"
	"testing"
)

func TestBuildFunctionTypedThrowingLocalDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

func risky(x: Int) -> Int throws Boom:
    return x + 1

func caller() -> Int throws Boom:
    let cb: fn(Int) -> Int throws Boom = risky
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

func TestBuildFunctionTypedThrowingCapturedClosureLocalDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

func caller() -> Int throws Boom:
    let base: Int = 1
    let cb: fn(Int) -> Int throws Boom = fn(x: Int) -> Int throws Boom:
        return x + base
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

func TestBuildFunctionTypedThrowingCapturedClosureMutableLocalReassignmentDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

func risky(x: Int) -> Int throws Boom:
    return x

func caller() -> Int throws Boom:
    let base: Int = 2
    var cb: fn(Int) -> Int throws Boom = risky
    cb = fn(x: Int) -> Int throws Boom:
        return x + base
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

func TestBuildFunctionTypedThrowingCallbackParamDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

func risky(x: Int) -> Int throws Boom:
    return x + 1

func apply(cb: fn(Int) -> Int throws Boom, x: Int) -> Int throws Boom:
    return try cb(x)

func main() -> Int:
    return catch apply(risky, 41):
    case Boom.bad:
        0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingCapturedClosureCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

func apply(cb: fn(Int) -> Int throws Boom, x: Int) -> Int throws Boom:
    return try cb(x)

func caller() -> Int throws Boom:
    let base: Int = 1
    return try apply(fn(x: Int) -> Int throws Boom:
        return x + base
    , 41)

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

func TestBuildFunctionTypedThrowingReturnDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

func risky(x: Int) -> Int throws Boom:
    return x + 1

func pick() -> fn(Int) -> Int throws Boom:
    return risky

func caller() -> Int throws Boom:
    let cb: fn(Int) -> Int throws Boom = pick()
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

func TestBuildFunctionTypedThrowingCapturedClosureReturnDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

func pick() -> fn(Int) -> Int throws Boom:
    let base: Int = 1
    return fn(x: Int) -> Int throws Boom:
        return x + base

func caller() -> Int throws Boom:
    let cb: fn(Int) -> Int throws Boom = pick()
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

func TestBuildFunctionTypedThrowingStructFieldDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

struct Holder:
    cb: fn(Int) -> Int throws Boom

func risky(x: Int) -> Int throws Boom:
    return x + 1

func caller() -> Int throws Boom:
    let holder: Holder = Holder(cb: risky)
    return try holder.cb(41)

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

func TestBuildFunctionTypedThrowingReturnedStructFieldDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

struct Holder:
    cb: fn(Int) -> Int throws Boom

func risky(x: Int) -> Int throws Boom:
    return x + 1

func makeHolder() -> Holder:
    return Holder(cb: risky)

func caller() -> Int throws Boom:
    let holder: Holder = makeHolder()
    return try holder.cb(41)

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

func TestBuildFunctionTypedThrowingCapturedClosureReturnedStructFieldDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

struct Holder:
    cb: fn(Int) -> Int throws Boom

func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )

func caller() -> Int throws Boom:
    let holder: Holder = makeHolder()
    return try holder.cb(41)

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

func TestBuildFunctionTypedThrowingCapturedClosureStructFieldDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

struct Holder:
    cb: fn(Int) -> Int throws Boom

func caller() -> Int throws Boom:
    let base: Int = 1
    let holder: Holder = Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
    return try holder.cb(41)

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

func TestBuildFunctionTypedThrowingCapturedClosureStructFieldAliasDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

struct Holder:
    cb: fn(Int) -> Int throws Boom

func caller() -> Int throws Boom:
    let base: Int = 1
    let holder: Holder = Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
    let cb: fn(Int) -> Int throws Boom = holder.cb
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

func TestBuildFunctionTypedThrowingCapturedClosureStructFieldReassignmentDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

struct Holder:
    cb: fn(Int) -> Int throws Boom

func risky(x: Int) -> Int throws Boom:
    return x

func caller() -> Int throws Boom:
    let base: Int = 2
    var holder: Holder = Holder(cb: risky)
    holder.cb = fn(x: Int) -> Int throws Boom:
        return x + base
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

func TestBuildFunctionTypedThrowingEnumPayloadDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

enum Choice:
    case some(fn(Int) -> Int throws Boom)

func risky(x: Int) -> Int throws Boom:
    return x + 1

func caller() -> Int throws Boom:
    let choice: Choice = Choice.some(risky)
    match choice:
        case Choice.some(cb):
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

func TestBuildFunctionTypedThrowingCapturedClosureEnumPayloadDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

enum Choice:
    case some(fn(Int) -> Int throws Boom)

func caller() -> Int throws Boom:
    let base: Int = 1
    let choice: Choice = Choice.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
    match choice:
        case Choice.some(cb):
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

func TestBuildFunctionTypedThrowingCapturedClosureReturnedEnumPayloadDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

enum Choice:
    case some(fn(Int) -> Int throws Boom)

func makeChoice() -> Choice:
    let base: Int = 1
    return Choice.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )

func caller() -> Int throws Boom:
    let choice: Choice = makeChoice()
    match choice:
        case Choice.some(cb):
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

func TestBuildFunctionTypedThrowingCapturedClosureEnumPayloadAliasDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

enum Choice:
    case some(fn(Int) -> Int throws Boom)

func caller() -> Int throws Boom:
    let base: Int = 1
    let choice: Choice = Choice.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
    match choice:
        case Choice.some(f):
            let cb: fn(Int) -> Int throws Boom = f
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

func TestBuildFunctionTypedThrowingCapturedClosureEnumPayloadReassignmentDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Boom:
    case bad

enum Choice:
    case some(fn(Int) -> Int throws Boom)

func risky(x: Int) -> Int throws Boom:
    return x

func caller() -> Int throws Boom:
    let base: Int = 2
    var choice: Choice = Choice.some(risky)
    choice = Choice.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
    match choice:
        case Choice.some(cb):
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

func TestBuildFunctionTypedThrowingCapturedClosureReturnCrossModuleDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/maker.t4": `module lib.maker

pub enum Boom:
    case bad

pub func make() -> fn(Int) -> Int throws Boom:
    let base: Int = 1
    return fn(x: Int) -> Int throws Boom:
        return x + base
`,
		"app/main.t4": `module app.main
import lib.maker as maker

func caller() -> Int throws maker.Boom:
    let cb: fn(Int) -> Int throws maker.Boom = maker.make()
    return try cb(41)

func main() -> Int:
    return catch caller():
    case maker.Boom.bad:
        0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingCapturedClosureReturnCrossModuleDirectCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/maker.t4": `module lib.maker

pub enum Boom:
    case bad

pub func make() -> fn(Int) -> Int throws Boom:
    let base: Int = 1
    return fn(x: Int) -> Int throws Boom:
        return x + base
`,
		"lib/callbacks.t4": `module lib.callbacks
import lib.maker as maker

pub func apply(cb: fn(Int) -> Int throws maker.Boom, x: Int) -> Int throws maker.Boom:
    return try cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks
import lib.maker as maker

func caller() -> Int throws maker.Boom:
    return try callbacks.apply(maker.make(), 41)

func main() -> Int:
    return catch caller():
    case maker.Boom.bad:
        0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingCapturedClosureReturnedStructFieldCrossModuleDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/maker.t4": `module lib.maker

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`,
		"app/main.t4": `module app.main
import lib.maker as maker

func caller() -> Int throws maker.Boom:
    let holder: maker.Holder = maker.makeHolder()
    return try holder.cb(41)

func main() -> Int:
    return catch caller():
    case maker.Boom.bad:
        0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedThrowingCapturedClosureReturnedEnumPayloadCrossModuleDirectTrySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/maker.t4": `module lib.maker

pub enum Boom:
    case bad

pub enum Choice:
    case some(fn(Int) -> Int throws Boom)

pub func makeChoice() -> Choice:
    let base: Int = 1
    return Choice.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`,
		"app/main.t4": `module app.main
import lib.maker as maker

func caller() -> Int throws maker.Boom:
    let choice: maker.Choice = maker.makeChoice()
    match choice:
        case maker.Choice.some(cb):
            return try cb(41)

func main() -> Int:
    return catch caller():
    case maker.Boom.bad:
        0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
