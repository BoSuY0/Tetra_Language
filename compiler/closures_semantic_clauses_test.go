package compiler

import (
	"strings"
	"testing"
)

func TestClosureLiteralParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let f: ptr = fn(x: Int) -> Int:
        return x
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
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestSemanticClausesParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int
noalloc
noblock
realtime
nothrow
budget(10):
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
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestSemanticClauseNothrowRejectsThrows(t *testing.T) {
	src := []byte(`
enum E:
    case bad

func main() -> Int throws E nothrow:
    throw E.bad
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected nothrow/throws conflict")
	}
	if !strings.Contains(err.Error(), "nothrow") {
		t.Fatalf("error = %v", err)
	}
}
