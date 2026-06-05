package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestCompilerBuildPLIRFormatsSliceLoopFacts(t *testing.T) {
	prog, err := compiler.Parse([]byte(`
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    for x in xs:
        total = total + x
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 1
    return sum(xs)
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	plirProg, err := compiler.BuildPLIR(checked)
	if err != nil {
		t.Fatalf("BuildPLIR: %v", err)
	}
	dump := compiler.FormatPLIR(plirProg)
	for _, want := range []string{"func sum", "fact index_in_range", "range: 0..xs.len"} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
}
