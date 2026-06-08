package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestAsyncBorrowLifetimeAllowsPreAwaitLocalUse(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
async func ready() -> Int:
    return 1

async func caller(xs: borrow []u8) -> Int:
    let view: []u8 = xs.borrow()
    let before: Int = view.len
    let after: Int = await ready()
    return before + after

func main() -> Int:
    return 0
`)
}

func TestAsyncBorrowLifetimeRejectsPostAwaitLocalUse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
async func ready() -> Int:
    return 1

async func caller(xs: borrow []u8) -> Int:
    let view: []u8 = xs.borrow()
    let _: Int = await ready()
    return view.len

func main() -> Int:
    return 0
`, "borrowed view 'view' cannot be used after await suspension")
}

func TestAsyncBorrowLifetimeRejectsPostTryAwaitLocalUse(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum AsyncErr:
    case failed

async func ready() -> Int throws AsyncErr:
    return 1

async func caller(xs: borrow []u8) -> Int throws AsyncErr:
    let view: []u8 = xs.borrow()
    let _: Int = try await ready()
    return view.len

func main() -> Int:
    return 0
`, "borrowed view 'view' cannot be used after await suspension")
}
