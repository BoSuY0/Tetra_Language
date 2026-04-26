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
