package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestCallableBorrowEscapeRejectsBorrowedStringCaptureGlobalStore(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func install(text: borrow String) -> Int:
    let view: String = text.window(0, 1).borrow()
    let captured: ptr = fn(x: Int) -> Int:
        return x + view.len
    cb = captured
    return 0

func main() -> Int:
    return 0
`, "borrowed local 'view' cannot escape via function capture")
}
