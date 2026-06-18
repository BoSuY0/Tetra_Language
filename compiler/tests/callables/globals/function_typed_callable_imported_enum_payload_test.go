package compiler_test

import (
	"runtime"
	"testing"
)

func TestBuildFunctionTypedImportedEnumPayloadParamCapturedClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice(f: fn(Int) -> Int) -> MaybeCallback:
    return MaybeCallback.some(f)

pub func callChoice(choice: MaybeCallback, x: Int) -> Int:
    match choice:
    case MaybeCallback.some(cb):
        return cb(x)
    case MaybeCallback.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let choice: callbacks.MaybeCallback = callbacks.makeChoice(captured)
    return callbacks.callChoice(choice, 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedEnumPayloadParamDirectReturnCapturedClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice(f: fn(Int) -> Int) -> MaybeCallback:
    return MaybeCallback.some(f)

pub func callChoice(choice: MaybeCallback, x: Int) -> Int:
    match choice:
    case MaybeCallback.some(cb):
        return cb(x)
    case MaybeCallback.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    return callbacks.callChoice(callbacks.makeChoice(captured), 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedEnumPayloadParamDirectConstructorCapturedClosureSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func callChoice(choice: MaybeCallback, x: Int) -> Int:
    match choice:
    case MaybeCallback.some(cb):
        return cb(x)
    case MaybeCallback.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    return callbacks.callChoice(callbacks.MaybeCallback.some(captured), 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedSelectiveImportedEnumPayloadParamDirectConstructorCapturedClosureSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func callChoice(choice: MaybeCallback, x: Int) -> Int:
    match choice:
    case MaybeCallback.some(cb):
        return cb(x)
    case MaybeCallback.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.callbacks.{callChoice, MaybeCallback}

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    return callChoice(MaybeCallback.some(captured), 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedImportedEnumPayloadParamDirectConstructorClosureLiteralSmoke(
	t *testing.T,
) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func callChoice(choice: MaybeCallback, x: Int) -> Int:
    match choice:
    case MaybeCallback.some(cb):
        return cb(x)
    case MaybeCallback.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let base: Int = 1
    return callbacks.callChoice(callbacks.MaybeCallback.some(fn(x: Int) -> Int = x + base), 41)
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
