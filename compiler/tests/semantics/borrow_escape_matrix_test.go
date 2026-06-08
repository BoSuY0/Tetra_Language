package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestBorrowEscapeMatrixRejectsGenericBorrowAggregateGlobalFieldTarget(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct Box<T>:
    value: T

struct Slot:
    box: Box<[]u8>

var saved: Slot

func stash(xs: borrow []u8) -> Int:
    saved.box = Box<[]u8>{value: xs.window(1, 2).borrow()}
    return 0
`, "contains borrowed slice field 'value' that cannot be stored in global")
}

func TestBorrowEscapeMatrixRejectsCrossModuleGenericBorrowAggregateGlobalFieldTarget(t *testing.T) {
	files := map[string]string{
		"lib/model.t4": `module lib.model

pub struct Box<T>:
    value: T

pub struct Slot:
    box: Box<[]u8>
`,
		"app/main.t4": `module app.main
import lib.model as model

var saved: model.Slot

func stash(xs: borrow []u8) -> Int:
    saved.box = model.Box<[]u8>{value: xs.window(1, 2).borrow()}
    return 0

func main() -> Int:
    return 0
`,
	}
	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "contains borrowed slice field 'value' that cannot be stored in global")
}

func TestBorrowEscapeMatrixAllowsCopiedGenericAggregateGlobalFieldTarget(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
struct Box<T>:
    value: T

struct Slot:
    box: Box<[]u8>

var saved: Slot

func stash(xs: borrow []u8) -> Int
uses alloc, mem:
    saved.box = Box<[]u8>{value: xs.window(1, 2).copy()}
    return 0

func main() -> Int:
    return 0
`)
}
