package compiler_test

import (
	"runtime"
	"testing"
)

func TestBuildFunctionTypedEnumPayloadDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadDirectCallAllowsArgumentLabelsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return cb(value: 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedParameterEnumPayloadDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func callPayload(f: fn(Int) -> Int, x: Int) -> Int:
    let choice: MaybeCallback = MaybeCallback.some(f)
    match choice:
    case MaybeCallback.some(cb):
        return cb(x)
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    return callPayload(add1, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedParameterEnumPayloadCallbackArgumentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func callPayload(f: fn(Int) -> Int, x: Int) -> Int:
    let choice: MaybeCallback = MaybeCallback.some(f)
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, x)
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    return callPayload(add1, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedParameterEnumPayloadReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func callPayload(f: fn(Int) -> Int, x: Int) -> Int:
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(f)
    match choice:
    case MaybeCallback.some(cb):
        return cb(x)
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    return callPayload(add1, 41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedParameterReturnedEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func makeChoice(f: fn(Int) -> Int) -> MaybeCallback:
    return MaybeCallback.some(f)

func main() -> Int:
    let choice: MaybeCallback = makeChoice(add1)
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureEnumPayloadCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    let extra: Int = 0
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base + extra
    let choice: MaybeCallback = MaybeCallback.some(f)
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedCapturedClosureEnumPayloadDirectCallSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let base: Int = 1
    let extra: Int = 0
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base + extra
    let choice: MaybeCallback = MaybeCallback.some(f)
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadDirectClosureLiteralSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let base: Int = 1
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int = x + base)
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedModuleEnumPayloadDirectClosureLiteralSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"app/main.t4": `module app.main

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let base: Int = 1
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int = x + base)
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

func TestBuildFunctionTypedEnumPayloadMutableCaptureSnapshotsAtBindingSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    var base: Int = 1
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int = x + base)
    base = 100
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadAliasSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        let f: fn(Int) -> Int = cb
        return apply(f, 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadWholeEnumAliasSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    let next: MaybeCallback = choice
    match next:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableEnumPayloadReassignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableEnumPayloadReassignmentFromReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func pick() -> fn(Int) -> Int:
    return add1

func main() -> Int:
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(pick())
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableEnumPayloadReassignmentFromMultiTargetReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

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
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(pick(0))
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableEnumPayloadReassignmentFromClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let base: Int = 1
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(fn(x: Int) -> Int = x + base)
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableEnumPayloadReassignmentFromWholeEnumAliasSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    var next: MaybeCallback = MaybeCallback.empty
    next = choice
    match next:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableEnumPayloadReassignmentMultiTargetSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func main() -> Int:
    var use_second: Int = 1
    var choice: MaybeCallback = MaybeCallback.empty
    if use_second:
        choice = MaybeCallback.some(add2)
    else:
        choice = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return cb(40)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedMutableEnumPayloadReassignmentMultiTargetCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    var use_second: Int = 1
    var choice: MaybeCallback = MaybeCallback.empty
    if use_second:
        choice = MaybeCallback.some(add2)
    else:
        choice = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 40)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func pick() -> fn(Int) -> Int:
    return add1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(pick())
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromMultiTargetReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

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
    let choice: MaybeCallback = MaybeCallback.some(pick(0))
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromMultiTargetCapturedClosureReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

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
    let choice: MaybeCallback = MaybeCallback.some(pick(0))
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromFourSlotCapturedClosureReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func makeChoice() -> MaybeCallback:
    let base: Int = 1
    let extra: Int = 0
    let third: Int = 0
    let fourth: Int = 0
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + base + extra + third + fourth
    )

func main() -> Int:
    let choice: MaybeCallback = makeChoice()
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromCrossModuleCapturedReturnedEnumSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(cb):
        return cb(41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
	}

	_, code := buildAndRunFiles(t, files, "app/main.t4")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromReturnedEnumSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func makeChoice() -> MaybeCallback:
    return MaybeCallback.some(add1)

func main() -> Int:
    let choice: MaybeCallback = makeChoice()
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromReturnedEnumWholeEnumAliasSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func makeChoice() -> MaybeCallback:
    return MaybeCallback.some(add1)

func main() -> Int:
    let choice: MaybeCallback = makeChoice()
    let next: MaybeCallback = choice
    match next:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadReturnedEnumMultiTargetCallbackSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func add2(x: Int) -> Int:
    return x + 2

func makeChoice(use_second: Int) -> MaybeCallback:
    if use_second:
        return MaybeCallback.some(add2)
    else:
        return MaybeCallback.some(add1)

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let choice: MaybeCallback = makeChoice(1)
    match choice:
    case MaybeCallback.some(cb):
        return apply(cb, 40)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromDirectReturnedEnumMatchSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func makeChoice() -> MaybeCallback:
    return MaybeCallback.some(add1)

func main() -> Int:
    match makeChoice():
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadCapturedClosureFromReturnedEnumSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int = x + base)

func main() -> Int:
    match makeChoice():
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadCapturedPtrClosureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let choice: MaybeCallback = MaybeCallback.some(captured)
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromStructFieldSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let holder: Holder = Holder(cb: add1)
    let choice: MaybeCallback = MaybeCallback.some(holder.cb)
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedStructFieldFromEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `struct Holder:
    cb: fn(Int) -> Int

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        let holder: Holder = Holder(cb: cb)
        return holder.cb(41)
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadFromEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    match choice:
    case MaybeCallback.some(cb):
        let next: MaybeCallback = MaybeCallback.some(cb)
        match next:
        case MaybeCallback.some(next_cb):
            return next_cb(41)
        case MaybeCallback.empty:
            return 0
    case MaybeCallback.empty:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildFunctionTypedEnumPayloadReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func add1(x: Int) -> Int:
    return x + 1

func pick() -> fn(Int) -> Int:
    let choice: MaybeCallback = MaybeCallback.some(add1)
    let fallback: fn(Int) -> Int = add1
    match choice:
    case MaybeCallback.some(cb):
        return cb
    case _:
        return fallback

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

func TestBuildCapturedFunctionTypedEnumPayloadReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func pick() -> fn(Int) -> Int:
    let base: Int = 1
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
    match choice:
    case MaybeCallback.some(cb):
        return cb
    case MaybeCallback.empty:
        return fn(x: Int) -> Int = x

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return cb(41)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
