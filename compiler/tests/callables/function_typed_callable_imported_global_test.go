package compiler_test

import (
	"runtime"
	"strings"
	"testing"
)

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
