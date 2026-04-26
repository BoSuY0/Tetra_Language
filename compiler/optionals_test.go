package compiler

import (
	"testing"
)

func TestOptionalNoneEqualityLowers(t *testing.T) {
	src := []byte(`
func maybe() -> Int?:
    return none

func main() -> Int:
    let value: Int? = maybe()
    if value == none:
        return 0
    else:
        return 1
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if checked.FuncSigs["maybe"].ReturnSlots != 2 {
		t.Fatalf("maybe return slots = %d, want 2", checked.FuncSigs["maybe"].ReturnSlots)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalIfLetLowers(t *testing.T) {
	src := []byte(`
func unwrap(value: Int?) -> Int:
    if let x = value:
        return x
    else:
        return 0

func main() -> Int:
    return unwrap(none)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.Funcs[0].Locals["x"].TypeName; got != "i32" {
		t.Fatalf("if-let local type = %q, want i32", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalImplicitSomeReturnAndLetLower(t *testing.T) {
	src := []byte(`
func maybe() -> Int?:
    return 42

func main() -> Int:
    let value: Int? = 7
    if value != none:
        return 0
    else:
        return 1
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalAllowsMultiSlotPayload(t *testing.T) {
	src := []byte(`
func maybe(flag: Bool) -> String?:
    if flag:
        return "ok"
    else:
        return none

func length(value: String?) -> Int:
    if let s = value:
        return s.len
    else:
        return 0

func main() -> Int:
    return length(maybe(true))
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.FuncSigs["maybe"].ReturnSlots; got != 3 {
		t.Fatalf("maybe return slots = %d, want 3", got)
	}
	if got := checked.FuncSigs["length"].ParamSlots; got != 3 {
		t.Fatalf("length param slots = %d, want 3", got)
	}
	if got := checked.Funcs[1].Locals["s"].TypeName; got != "str" {
		t.Fatalf("if-let local type = %q, want str", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}
