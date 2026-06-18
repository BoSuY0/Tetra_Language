package compiler_test

import (
	"runtime"
	"testing"
)

func TestBuildFunctionTypedStructFieldDirectNamedSymbolCrossModuleSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub func add1(x: Int) -> Int:
    return x + 1
`,
		"app/main.t4": `module app.main
import lib.math as math

struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let holder: Holder = Holder(cb: math.add1)
    return holder.cb(41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadDirectNamedSymbolCrossModuleSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/math.t4": `module lib.math

pub func add1(x: Int) -> Int:
    return x + 1
`,
		"app/main.t4": `module app.main
import lib.math as math

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(math.add1)
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
