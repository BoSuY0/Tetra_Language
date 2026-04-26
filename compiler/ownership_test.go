package compiler

import (
	"strings"
	"testing"
)

func TestOwnershipMarkersParseAndFormat(t *testing.T) {
	src := []byte(`
func mix(a: borrow Int, b: inout Int, c: consume Int) -> Int:
    return a + b + c
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	params := prog.Funcs[0].Params
	if params[0].Ownership != "borrow" || params[1].Ownership != "inout" || params[2].Ownership != "consume" {
		t.Fatalf("ownership markers = %q/%q/%q", params[0].Ownership, params[1].Ownership, params[2].Ownership)
	}
	formatted, err := FormatSource(src, "ownership.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(string(formatted), "a: borrow Int, b: inout Int, c: consume Int") {
		t.Fatalf("formatted source missing markers:\n%s", string(formatted))
	}
}

func TestOwnershipInoutParamIsMutable(t *testing.T) {
	src := []byte(`
func bump(x: inout Int) -> Int:
    x = x + 1
    return x

func main() -> Int:
    var a: Int = 1
    return bump(a)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestOwnershipInoutRequiresMutableLocal(t *testing.T) {
	src := []byte(`
func bump(x: inout Int) -> Int:
    x = x + 1
    return x

func main() -> Int:
    let a: Int = 1
    return bump(a)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected inout argument mutability error")
	}
	if !strings.Contains(err.Error(), "inout argument") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipBorrowParamCannotMutate(t *testing.T) {
	src := []byte(`
func bump(x: borrow Int) -> Int:
    x = x + 1
    return x

func main() -> Int:
    return bump(1)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected borrow mutation error")
	}
	if !strings.Contains(err.Error(), "cannot assign to val 'x'") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsReturningBorrowedParam(t *testing.T) {
	src := []byte(`
func leak(x: borrow []u8) -> []u8:
    return x

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected escaping borrowed local error")
	}
	if !strings.Contains(err.Error(), "borrow") || !strings.Contains(err.Error(), "escape") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsBorrowEscapeViaAliasReturn(t *testing.T) {
	src := []byte(`
func leak(x: borrow []u8) -> []u8:
    let y: []u8 = x
    return y

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected escaping borrowed alias error")
	}
	if !strings.Contains(err.Error(), "borrow") || !strings.Contains(err.Error(), "escape") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsConsumeOfBorrowDerivedAlias(t *testing.T) {
	src := []byte(`
func take(x: consume []u8) -> Int:
    return 0

func leak(x: borrow []u8) -> Int:
    let y: []u8 = x
    return take(y)

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected borrowed consume rejection")
	}
	if !strings.Contains(err.Error(), "borrow") || !strings.Contains(err.Error(), "consume") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsIfBlockLocalEscape(t *testing.T) {
	src := []byte(`
func main() -> Int:
    if 1:
        let x: Int = 1
    return x
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected block-local scope error")
	}
	if !strings.Contains(err.Error(), "out of scope") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipConsumeArgumentCannotBeReused(t *testing.T) {
	src := []byte(`
func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let a: Int = 1
    let b: Int = take(a)
    return a + b
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected consumed reuse error")
	}
	if !strings.Contains(err.Error(), "cannot use consumed value 'a'") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipConsumedValueCannotBeReassigned(t *testing.T) {
	src := []byte(`
func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    var a: Int = 1
    let b: Int = take(a)
    a = b
    return b
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected consumed assignment error")
	}
	if !strings.Contains(err.Error(), "cannot use consumed value 'a'") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsBorrowInoutAlias(t *testing.T) {
	src := []byte(`
func mix(read: borrow Int, write: inout Int) -> Int:
    write = write + read
    return write

func main() -> Int:
    var a: Int = 1
    return mix(a, a)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected borrow/inout aliasing error")
	}
	if !strings.Contains(err.Error(), "alias") && !strings.Contains(err.Error(), "borrow") {
		t.Fatalf("error = %v", err)
	}
}

func TestOwnershipRejectsConsumeInoutAlias(t *testing.T) {
	src := []byte(`
func mix(taken: consume Int, write: inout Int) -> Int:
    write = write + taken
    return write

func main() -> Int:
    var a: Int = 1
    return mix(a, a)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected consume/inout aliasing error")
	}
	if !strings.Contains(err.Error(), "alias") && !strings.Contains(err.Error(), "consume") {
		t.Fatalf("error = %v", err)
	}
}
