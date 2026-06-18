package compiler_test

import (
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/tests/callables/testkit"
)

var (
	buildAndRun      = testkit.BuildAndRun
	buildAndRunFiles = testkit.BuildAndRunFiles
	buildOnly        = testkit.BuildOnly
	buildOnlyFiles   = testkit.BuildOnlyFiles
)

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
			want: ("function-typed global 'cb' initializer must be a same-module " +
				"named function symbol for the supported fnptr ABI"),
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
			want: ("function-typed global 'cb' initializer must be an imported " +
				"public function symbol for the supported fnptr ABI"),
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
			want: ("function-typed global 'cb' must be initialized with a direct " +
				"named function symbol or closure literal for the supported fnptr ABI"),
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
