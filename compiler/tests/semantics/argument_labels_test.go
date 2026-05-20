package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestArgumentLabelsAcceptedByChecker(t *testing.T) {
	src := `
func add(a: Int, b: Int) -> Int:
    return a + b

func main() -> Int:
    return add(a: 40, b: 2)
`
	testkit.RequireCheckOK(t, src)
}

func TestArgumentLabelsRejectMismatchedOrder(t *testing.T) {
	src := `
func add(a: Int, b: Int) -> Int:
    return a + b

func main() -> Int:
    return add(b: 40, a: 2)
`
	testkit.RequireCheckErrorContains(t, src, "argument label mismatch")
}
