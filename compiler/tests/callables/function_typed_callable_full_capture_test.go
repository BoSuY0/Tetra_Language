package compiler_test

import (
	"runtime"
	"testing"
)

func TestBuildFunctionTypedCapturedClosureEightSlotReturnCrossModuleCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/maker.t4": `module lib.maker

pub func make() -> fn(Int) -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    return fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight
`,
		"lib/callbacks.t4": `module lib.callbacks

pub func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks
import lib.maker as maker

func main() -> Int:
    return callbacks.apply(maker.make(), 6)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureEightSlotEnumReturnCrossModuleCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/maker.t4": `module lib.maker

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func make() -> MaybeCallback:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight
    )
`,
		"lib/callbacks.t4": `module lib.callbacks

pub func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks
import lib.maker as maker

func main() -> Int:
    let choice: maker.MaybeCallback = maker.make()
    match choice:
    case maker.MaybeCallback.some(cb):
        return callbacks.apply(cb, 6)
    case maker.MaybeCallback.empty:
        return 0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableEscapedNineCaptureReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func make() -> fn(Int) -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    return fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine

func main() -> Int:
    let cb: fn(Int) -> Int = make()
    return cb(-3)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableLocalNineCaptureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let cb: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    return cb(-3)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableMutableLocalReassignNineCaptureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func add0(x: Int) -> Int:
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
    let nine: Int = 9
    var cb: fn(Int) -> Int = add0
    cb = fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    return cb(-3)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableStructFieldNineCaptureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let holder: Holder = Holder(cb: fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    )
    return holder.cb(-3)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableStructFieldReassignNineCaptureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

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
    let nine: Int = 9
    var holder: Holder = Holder(cb: add0)
    holder.cb = fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    return holder.cb(-3)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableEnumPayloadNineCaptureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    )
    match choice:
    case MaybeCallback.some(cb):
        return cb(-3)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableEnumPayloadReassignNineCaptureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    )
    match choice:
    case MaybeCallback.some(cb):
        return cb(-3)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableEscapedGlobalNineCaptureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func install() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    cb = fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    return 0

func main() -> Int:
    let _: Int = install()
    return cb(-3)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableCallbackArgumentNineCaptureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    return apply(fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    , -3)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableLocalCallbackArgumentNineCaptureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let cb: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    return apply(cb, -3)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableReturnAliasTwelveCaptureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func make() -> fn(Int) -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let ten: Int = 10
    let eleven: Int = 11
    let twelve: Int = 12
    let captured: ptr = fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine + ten + eleven + twelve
    return captured

func main() -> Int:
    let cb: fn(Int) -> Int = make()
    return cb(-36)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableGlobalAliasTwelveCaptureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func install() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let ten: Int = 10
    let eleven: Int = 11
    let twelve: Int = 12
    let captured: ptr = fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine + ten + eleven + twelve
    let alias: fn(Int) -> Int = captured
    cb = alias
    return 0

func main() -> Int:
    let _: Int = install()
    return cb(-36)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableCallbackAliasTwelveCaptureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let ten: Int = 10
    let eleven: Int = 11
    let twelve: Int = 12
    let captured: ptr = fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine + ten + eleven + twelve
    return apply(captured, -36)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFullCallableCrossModuleReturnedNineCaptureMatrixSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/maker.t4": `module lib.maker

pub func make() -> fn(Int) -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    return fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
`,
		"lib/callbacks.t4": `module lib.callbacks

pub func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks
import lib.maker as maker

struct Holder:
    cb: fn(Int) -> Int

enum Choice:
    case some(fn(Int) -> Int)

func alias(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return cb

func throughLocal() -> Int:
    let cb: fn(Int) -> Int = maker.make()
    return cb(-3)

func throughStruct() -> Int:
    let holder: Holder = Holder(cb: maker.make())
    return holder.cb(-3)

func throughEnum() -> Int:
    let choice: Choice = Choice.some(maker.make())
    match choice:
    case Choice.some(cb):
        return cb(-3)

func throughCallback() -> Int:
    return callbacks.apply(maker.make(), -3)

func throughAlias() -> Int:
    let cb: fn(Int) -> Int = alias(maker.make())
    return cb(-3)

func main() -> Int:
    return throughLocal() + throughStruct() + throughEnum() + throughCallback() + throughAlias()
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 210 {
		t.Fatalf("exit code mismatch: got %d, want 210", code)
	}
}
