package compiler

import (
	"strings"
	"testing"
)

func TestGenericFunctionParseCheckAndDocs(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := prog.Funcs[0].TypeParams; len(got) != 1 || got[0] != "T" {
		t.Fatalf("type params = %#v", got)
	}
	if _, err := Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
	docs, err := GenerateAPIDocsFromSource(src, "generics.tetra")
	if err != nil {
		t.Fatalf("GenerateAPIDocsFromSource: %v", err)
	}
	if !strings.Contains(string(docs), "`func id<T>(x: T) -> T`") {
		t.Fatalf("docs = %s", string(docs))
	}
}

func TestGenericFunctionMonomorphizedCall(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return id(42)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.FuncSigs["id__T_i32"]; !ok {
		t.Fatalf("missing monomorphized signature: %#v", checked.FuncSigs)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericFunctionUnsupportedArgDiagnostic(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return id(unknown)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected generic inference diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot infer generic argument") {
		t.Fatalf("error = %v", err)
	}
}
