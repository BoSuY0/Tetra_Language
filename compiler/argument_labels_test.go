package compiler

import (
	"strings"
	"testing"
)

func TestArgumentLabelsAcceptedByChecker(t *testing.T) {
	src := `
func add(a: Int, b: Int) -> Int:
    return a + b

func main() -> Int:
    return add(a: 40, b: 2)
`
	if err := checkProgram(src); err != nil {
		t.Fatalf("checkProgram: %v", err)
	}
}

func TestArgumentLabelsRejectMismatchedOrder(t *testing.T) {
	src := `
func add(a: Int, b: Int) -> Int:
    return a + b

func main() -> Int:
    return add(b: 40, a: 2)
`
	err := checkProgram(src)
	if err == nil {
		t.Fatalf("expected label mismatch error")
	}
	if !strings.Contains(err.Error(), "argument label mismatch") {
		t.Fatalf("error = %v", err)
	}
}
