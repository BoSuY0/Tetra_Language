package compiler

import (
	"strings"
	"testing"
)

func TestTypedErrorsParseCheckAndLower(t *testing.T) {
	src := []byte(`
enum ReadError:
    case eof

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 42
    else:
        throw ReadError.eof

func caller() -> Int throws ReadError:
    let value: Int = try read(true)
    return value

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !prog.Funcs[0].HasThrows || prog.Funcs[0].Throws.Name != "ReadError" {
		t.Fatalf("throws = %#v", prog.Funcs[0].Throws)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["read"].ThrowsType; got != "ReadError" {
		t.Fatalf("read throws = %q", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsRejectBareThrowingCall(t *testing.T) {
	src := []byte(`
enum E:
    case bad

func f() -> Int throws E:
    throw E.bad

func main() -> Int:
    return f()
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected bare throwing call error")
	}
	if !strings.Contains(err.Error(), "requires try") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsRejectTryOutsideThrowingFunction(t *testing.T) {
	src := []byte(`
enum E:
    case bad

func f() -> Int throws E:
    throw E.bad

func main() -> Int:
    return try f()
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected try context error")
	}
	if !strings.Contains(err.Error(), "try is only allowed in throwing functions") {
		t.Fatalf("error = %v", err)
	}
}

func TestTypedErrorsAllowMultiSlotErrorPayload(t *testing.T) {
	src := []byte(`
func fail(flag: Bool) -> Int throws String:
    if flag:
        return 7
    else:
        throw "bad"

func caller(flag: Bool) -> Int throws String:
    return try fail(flag)

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["fail"].ReturnSlots; got != 4 {
		t.Fatalf("fail return slots = %d, want 4", got)
	}
	if got := checked.FuncSigs["fail"].ThrowsType; got != "str" {
		t.Fatalf("fail throws type = %q, want str", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsTryPropagatesIntoOptionalThrows(t *testing.T) {
	src := []byte(`
func fail(flag: Bool) -> Int throws Int:
    if flag:
        return 7
    else:
        throw 11

func caller(flag: Bool) -> Int throws Int?:
    return try fail(flag)

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["caller"].ThrowsType; got != "i32?" {
		t.Fatalf("caller throws type = %q, want i32?", got)
	}
	if got := checked.FuncSigs["caller"].ReturnSlots; got != 4 {
		t.Fatalf("caller return slots = %d, want 4", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestTypedErrorsTryPropagatesMultiSlotIntoOptionalThrows(t *testing.T) {
	src := []byte(`
func fail(flag: Bool) -> Int throws String:
    if flag:
        return 7
    else:
        throw "bad"

func caller(flag: Bool) -> Int throws String?:
    return try fail(flag)

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if got := checked.FuncSigs["caller"].ThrowsType; got != "str?" {
		t.Fatalf("caller throws type = %q, want str?", got)
	}
	if got := checked.FuncSigs["caller"].ReturnSlots; got != 5 {
		t.Fatalf("caller return slots = %d, want 5", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}
